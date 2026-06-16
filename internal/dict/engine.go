package dict

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	mdx "github.com/lib-x/mdx"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog/log"

	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
)

// isPathSafe checks that resolvedPath stays within baseDir, preventing path traversal.
func isPathSafe(baseDir, resolvedPath string) bool {
	cleanBase := filepath.Clean(baseDir)
	cleanResolved := filepath.Clean(resolvedPath)
	return strings.HasPrefix(cleanResolved, cleanBase+string(os.PathSeparator)) || cleanResolved == cleanBase
}

// Engine manages dictionary loading and querying
type Engine struct {
	mu         sync.RWMutex
	dicts      map[string]*LoadedDict
	dictDir    string
	dictStore  *store.DictStore
	sanitizer  *bluemonday.Policy
	wordIndex  []string // global sorted deduplicated word list for fast prefix search
}

// LoadedDict represents a loaded dictionary with its parsed data
type LoadedDict struct {
	Info       *models.Dictionary
	mdxDict    *mdx.Mdict                 // MDX dictionary for word lookup
	mddDict    *mdx.Mdict                 // MDD resource file (nil if no .mdd)
	fuzzyStore *mdx.MemoryFuzzyIndexStore // fuzzy search index
	dictName   string                     // dictionary name used in fuzzy store (DictionaryInfo.Name)
	resDir     string                     // directory for standalone resource files (same as .mdx filename without ext)
}

// NewEngine creates a new dictionary engine
func NewEngine(dictDir string, dictStore *store.DictStore) *Engine {
	// Create a sanitization policy for dictionary HTML content.
	// Allow common HTML tags used in dictionary definitions.
	p := bluemonday.UGCPolicy()
	p.AllowElements("div", "span", "p", "b", "i", "u", "a", "img", "audio", "source",
		"br", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li",
		"table", "tr", "td", "th", "thead", "tbody", "font", "sup", "sub",
		"blockquote", "pre", "code", "dl", "dt", "dd", "em", "strong", "small",
		"mark", "abbr", "ruby", "rt", "rp", "bdo",
		"style", "link")
	p.AllowAttrs("href", "src", "alt", "title", "class", "style", "width", "height",
		"controls", "autoplay", "loop", "type", "color", "size", "face",
		"rel", "media", "charset", "cellpadding", "cellspacing", "border",
		"align", "valign", "bgcolor", "nowrap", "colspan", "rowspan",
		"scope", "id", "name").Globally()
	p.AllowDataAttributes()

	return &Engine{
		dicts:     make(map[string]*LoadedDict),
		dictDir:   dictDir,
		dictStore: dictStore,
		sanitizer: p,
	}
}

// LoadAll loads all dictionaries from the dictionary directory (including subdirectories)
func (e *Engine) LoadAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Recursively scan for .mdx files
	var mdxFiles []string
	_ = filepath.WalkDir(e.dictDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".mdx") {
			relPath, _ := filepath.Rel(e.dictDir, path)
			mdxFiles = append(mdxFiles, relPath)
		}
		return nil
	})

	loadedCount := 0
	for _, filename := range mdxFiles {
		// Check if already loaded
		dictID := models.NewDictionary(filename, 0).ID
		if _, exists := e.dicts[dictID]; exists {
			continue
		}

		// Load dictionary
		if err := e.loadDict(filename); err != nil {
			log.Error().Err(err).Str("filename", filename).Msg("Failed to load dictionary")
			continue
		}

		loadedCount++
	}

	// Disable orphaned records: dicts in DB whose files no longer exist on disk
	if err := e.disableOrphans(); err != nil {
		log.Error().Err(err).Msg("Failed to check orphaned dictionaries")
	}

	log.Info().Int("count", loadedCount).Msg("Loaded dictionaries")

	// Rebuild global word index after loading
	e.rebuildWordIndex()

	return nil
}

// disableOrphans marks DB records as disabled when the corresponding .mdx file no longer exists on disk.
func (e *Engine) disableOrphans() error {
	allDicts, err := e.dictStore.List()
	if err != nil {
		return err
	}

	for _, d := range allDicts {
		if _, loaded := e.dicts[d.ID]; loaded {
			continue
		}
		// Not loaded — check if file exists
		mdxPath := filepath.Join(e.dictDir, d.Filename)
		if _, err := os.Stat(mdxPath); os.IsNotExist(err) {
			if d.IsEnabled {
				if err := e.dictStore.UpdateStatus(d.ID, false); err != nil {
					log.Error().Err(err).Str("id", d.ID).Str("filename", d.Filename).Msg("Failed to disable orphaned dictionary")
				} else {
					log.Warn().Str("id", d.ID).Str("filename", d.Filename).Msg("Disabled orphaned dictionary (file missing)")
				}
			}
		}
	}
	return nil
}

// rebuildWordIndex rebuilds the global sorted deduplicated word list from all loaded dictionaries.
// Must be called with e.mu held.
func (e *Engine) rebuildWordIndex() {
	wordSet := make(map[string]struct{})
	for _, dict := range e.dicts {
		if !dict.Info.IsEnabled {
			continue
		}
		entries, err := dict.mdxDict.ExportEntries()
		if err != nil {
			continue
		}
		for _, entry := range entries {
			w := strings.TrimSpace(entry.Keyword)
			if w != "" {
				wordSet[w] = struct{}{}
			}
		}
	}

	words := make([]string, 0, len(wordSet))
	for w := range wordSet {
		words = append(words, w)
	}
	sort.Strings(words)
	e.wordIndex = words

	log.Info().Int("unique_words", len(words)).Msg("Global word index rebuilt")
}

// loadDict loads a single dictionary file
func (e *Engine) loadDict(filename string) error {
	mdxPath := filepath.Join(e.dictDir, filename)

	// Get file info
	fileInfo, err := os.Stat(mdxPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create dictionary info
	dictInfo := models.NewDictionary(filename, fileInfo.Size())

	// Parse MDX dictionary
	mdxDict, err := InitMDX(mdxPath)
	if err != nil {
		return fmt.Errorf("failed to parse dictionary: %w", err)
	}

	// Get metadata from the parsed dictionary
	info := mdxDict.DictionaryInfo()
	// Always use filename (without .mdx extension) as the initial title.
	// Users can change it later in the admin UI.
	dictInfo.Title = strings.TrimSuffix(filename, filepath.Ext(filename))
	dictInfo.Description = info.Description
	dictInfo.EntryCount = info.EntryCount

	// Check for .mdd file
	var mddDict *mdx.Mdict
	mddFilename := strings.TrimSuffix(filename, ".mdx") + ".mdd"
	mddPath := filepath.Join(e.dictDir, mddFilename)
	if _, err := os.Stat(mddPath); err == nil {
		dictInfo.HasMdd = true
		mddDict, err = InitMDD(mddPath)
		if err != nil {
			log.Warn().Err(err).Str("path", mddPath).Msg("Failed to load MDD file, continuing without assets")
			mddDict = nil
			dictInfo.HasMdd = false
		}
	}

	// Resource subdirectory: same name as .mdx without extension
	resDir := strings.TrimSuffix(filename, filepath.Ext(filename))
	resDirPath := filepath.Join(e.dictDir, resDir)
	if info, err := os.Stat(resDirPath); err == nil && info.IsDir() {
		log.Info().Str("resDir", resDirPath).Msg("Found resource directory for dictionary")
	}

	// Build fuzzy search index
	fuzzyStore := mdx.NewMemoryFuzzyIndexStore()
	exportedEntries, err := mdxDict.ExportEntries()
	if err != nil {
		log.Warn().Err(err).Str("filename", filename).Msg("Failed to export entries for fuzzy index")
	} else {
		if err := fuzzyStore.Put(info, exportedEntries); err != nil {
			log.Warn().Err(err).Str("filename", filename).Msg("Failed to build fuzzy index")
		} else {
			log.Info().
				Str("filename", filename).
				Int("fuzzyEntries", len(exportedEntries)).
				Msg("Fuzzy index built")
		}
	}

	// Store in database
	exists, err := e.dictStore.ExistsByFilename(filename)
	if err != nil {
		return fmt.Errorf("failed to check dictionary existence: %w", err)
	}

	if exists {
		// Update existing
		existing, err := e.dictStore.GetByFilename(filename)
		if err != nil {
			return fmt.Errorf("failed to get existing dictionary: %w", err)
		}
		dictInfo.ID = existing.ID
		dictInfo.IsEnabled = existing.IsEnabled
		if err := e.dictStore.Update(dictInfo); err != nil {
			return fmt.Errorf("failed to update dictionary: %w", err)
		}
	} else {
		// Create new
		if err := e.dictStore.Create(dictInfo); err != nil {
			return fmt.Errorf("failed to create dictionary: %w", err)
		}
	}

	// Store in memory
	e.dicts[dictInfo.ID] = &LoadedDict{
		Info:       dictInfo,
		mdxDict:    mdxDict,
		mddDict:    mddDict,
		fuzzyStore: fuzzyStore,
		dictName:   info.Name,
		resDir:     resDir,
	}

	log.Info().
		Str("id", dictInfo.ID).
		Str("filename", filename).
		Int64("entries", dictInfo.EntryCount).
		Bool("hasMdd", dictInfo.HasMdd).
		Msg("Loaded dictionary")

	return nil
}

// Reload reloads a specific dictionary
func (e *Engine) Reload(dictID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	loaded, exists := e.dicts[dictID]
	if !exists {
		return fmt.Errorf("dictionary not loaded: %s", dictID)
	}

	// Remove from memory
	delete(e.dicts, dictID)

	// Reload
	if err := e.loadDict(loaded.Info.Filename); err != nil {
		return err
	}

	// Rebuild global word index after reload
	e.rebuildWordIndex()
	return nil
}

// Unload removes a dictionary from memory
func (e *Engine) Unload(dictID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.dicts, dictID)
}

// Search performs an exact search across all enabled dictionaries
func (e *Engine) Search(word string, dictID string) (*models.SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := &models.SearchResult{
		Word:    word,
		Results: []models.DictResult{},
	}

	// Normalize word
	word = strings.TrimSpace(word)
	if word == "" {
		return result, nil
	}

	for id, dict := range e.dicts {
		// Filter by dictID if specified
		if dictID != "" && id != dictID {
			continue
		}

		// Skip disabled dictionaries
		if !dict.Info.IsEnabled {
			continue
		}

		// Look up word in MDX dictionary
		htmlBytes, err := dict.mdxDict.Lookup(word)
		if err != nil {
			// Word not found in this dictionary, skip
			continue
		}

		// Sanitize HTML output
		rawHTML := string(htmlBytes)
		// Extract <style> blocks before sanitization (bluemonday strips them)
		styleBlocks := extractStyleBlocks(rawHTML)
		log.Debug().
			Str("word", word).
			Str("dictID", id).
			Int("styleBlocks", len(styleBlocks)).
			Int("rawHTMLLen", len(rawHTML)).
			Msg("Extracted style blocks from dictionary HTML")
		cleanHTML := e.sanitizer.Sanitize(rawHTML)
		// Re-inject style blocks after sanitization, with rewritten URLs
		if len(styleBlocks) > 0 {
			var styleHTML string
			for _, css := range styleBlocks {
				css = rewriteAssetURLs(css, id)
				styleHTML += "<style type=\"text/css\">" + css + "</style>"
			}
			cleanHTML = styleHTML + cleanHTML
		}
		// Rewrite resource URLs to point to the asset endpoint
		cleanHTML = rewriteAssetURLs(cleanHTML, id)

		// Check for audio in MDD
		hasAudio := false
		audioURL := ""
		if dict.mddDict != nil {
			audioPath := fmt.Sprintf("\\%s.mp3", strings.ToLower(word))
			if _, err := dict.mddDict.AssetResolver().Read(audioPath); err == nil {
				hasAudio = true
				audioURL = fmt.Sprintf("/api/v1/assets/%s/%s", id, audioPath)
			}
		}

		result.Results = append(result.Results, models.DictResult{
			DictID:   id,
			DictName: dict.Info.Title,
			HTML:     cleanHTML,
			HasAudio: hasAudio,
			AudioURL: audioURL,
		})
	}

	return result, nil
}

// styleBlockPattern matches <style>...</style> blocks in dictionary HTML.
var styleBlockPattern = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)

// linkStylePattern matches <link rel="stylesheet" href="..."> tags.
var linkStylePattern = regexp.MustCompile(`(?i)<link[^>]+rel\s*=\s*["']?stylesheet["']?[^>]*>`)

// linkHrefPattern extracts href from a link tag.
var linkHrefPattern = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)

// extractStyleBlocks extracts the CSS content from <style> blocks.
func extractStyleBlocks(html string) []string {
	matches := styleBlockPattern.FindAllStringSubmatch(html, -1)
	var blocks []string
	// First, collect <link rel="stylesheet"> as @import directives (must come first in CSS)
	linkMatches := linkStylePattern.FindAllString(html, -1)
	for _, linkTag := range linkMatches {
		hrefSub := linkHrefPattern.FindStringSubmatch(linkTag)
		if len(hrefSub) > 1 && strings.TrimSpace(hrefSub[1]) != "" {
			blocks = append(blocks, "@import \""+hrefSub[1]+"\";")
		}
	}
	// Then, collect inline <style> block contents
	for _, m := range matches {
		if len(m) > 1 && strings.TrimSpace(m[1]) != "" {
			blocks = append(blocks, m[1])
		}
	}

	return blocks
}

// assetURLPattern matches href/src attributes referencing resource files.
var assetURLPattern = regexp.MustCompile(`(?:href|src)\s*=\s*["']([^"']+?\.(css|js|png|jpg|jpeg|gif|svg|webp|mp3|wav|ogg|woff|woff2|ttf|eot|ico|pdf|m4a))["']`)

// cssURLPattern matches url() references in inline styles and <style> blocks.
var cssURLPattern = regexp.MustCompile(`url\(\s*["']?([^"')]+?\.(css|js|png|jpg|jpeg|gif|svg|webp|mp3|wav|ogg|woff|woff2|ttf|eot|ico|pdf|m4a))\s*["']?\)`)

// cssImportPattern matches @import 'file.css' or @import "file.css" (without url() wrapper).
var cssImportPattern = regexp.MustCompile(`@import\s+["']([^"']+\.(css))["']`)

// rewriteAssetURLs rewrites resource URLs in MDX HTML output to point to the
// server's asset endpoint (/api/v1/assets/:id/...) so the browser can load
// CSS, images, fonts, etc. from either the MDD or the filesystem.
func rewriteAssetURLs(html, dictID string) string {
	origHTML := html
	rewritePath := func(originalPath string) string {
		if strings.HasPrefix(originalPath, "http") || strings.HasPrefix(originalPath, "/api/") {
			return originalPath
		}
		assetPath := strings.ReplaceAll(originalPath, `\`, "/")
		assetPath = strings.TrimPrefix(assetPath, "./")
		assetPath = strings.TrimPrefix(assetPath, "/")
		return "/api/v1/assets/" + dictID + "/" + assetPath
	}

	// Rewrite href/src attributes
	html = assetURLPattern.ReplaceAllStringFunc(html, func(match string) string {
		sub := assetURLPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		replacement := rewritePath(sub[1])
		return strings.Replace(match, sub[1], replacement, 1)
	})

	// Rewrite url() references in inline styles
	html = cssURLPattern.ReplaceAllStringFunc(html, func(match string) string {
		sub := cssURLPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		replacement := rewritePath(sub[1])
		return strings.Replace(match, sub[1], replacement, 1)
	})

	// Rewrite @import 'file.css' references (without url() wrapper)
	html = cssImportPattern.ReplaceAllStringFunc(html, func(match string) string {
		sub := cssImportPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		replacement := rewritePath(sub[1])
		return strings.Replace(match, sub[1], replacement, 1)
	})

	if html != origHTML {
		log.Debug().
			Str("dictID", dictID).
			Bool("rewritten", true).
			Msg("Rewrote asset URLs in HTML/CSS")
	}
	return html
}
func (e *Engine) FuzzySearch(keyword string, dictID string, page, pageSize int) (*models.FuzzySearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if len(keyword) < 2 {
		return nil, fmt.Errorf("keyword must be at least 2 characters")
	}

	// Binary search on the global sorted word index for prefix matches
	// Find the first word that starts with the keyword (case-insensitive)
	matches := e.prefixSearch(keyword)

	// Calculate pagination
	total := len(matches)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// Build result — just words, no per-dictionary info
	items := make([]models.FuzzyItem, 0, end-start)
	for _, word := range matches[start:end] {
		items = append(items, models.FuzzyItem{Word: word})
	}

	return &models.FuzzySearchResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// prefixSearch uses binary search on the global word index to find all words
// whose lowercase form starts with the given keyword prefix.
// Returns words in sorted order. O(log n + k) where k = number of matches.
func (e *Engine) prefixSearch(keyword string) []string {
	idx := e.wordIndex
	if len(idx) == 0 {
		return nil
	}

	// Find the leftmost position where keyword could be a prefix
	lo := sort.Search(len(idx), func(i int) bool {
		return strings.ToLower(idx[i]) >= keyword
	})

	// Collect all words starting with keyword
	var matches []string
	for i := lo; i < len(idx); i++ {
		if !strings.HasPrefix(strings.ToLower(idx[i]), keyword) {
			break
		}
		matches = append(matches, idx[i])
	}

	return matches
}

// GetDict returns a loaded dictionary
func (e *Engine) GetDict(dictID string) (*LoadedDict, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	dict, exists := e.dicts[dictID]
	return dict, exists
}

// GetDicts returns all loaded dictionaries
func (e *Engine) GetDicts() map[string]*LoadedDict {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]*LoadedDict)
	for k, v := range e.dicts {
		result[k] = v
	}
	return result
}

// GetAsset returns an asset from a dictionary's .mdd file
func (e *Engine) GetAsset(dictID, assetPath string) ([]byte, string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Safety: strip leading slash (may come from Gin wildcard param)
	assetPath = strings.TrimPrefix(assetPath, "/")

	dict, exists := e.dicts[dictID]
	if !exists {
		return nil, "", fmt.Errorf("dictionary not found: %s", dictID)
	}

	// 1. Try MDD first
	if dict.mddDict != nil {
		normalizedPath := assetPath
		// Normalize: convert / to \ and ensure leading \
		if !strings.HasPrefix(normalizedPath, "\\") {
			normalizedPath = "\\" + strings.ReplaceAll(normalizedPath, "/", "\\")
		} else {
			normalizedPath = strings.ReplaceAll(normalizedPath, "/", "\\")
		}
		data, err := dict.mddDict.AssetResolver().Read(normalizedPath)
		if err == nil {
			mimeType := GetMimeType(assetPath)
			return data, mimeType, nil
		}
		// Retry without leading backslash (some MDD files don't use it)
		withoutLeadingSlash := strings.TrimPrefix(normalizedPath, "\\")
		if withoutLeadingSlash != normalizedPath {
			data, err = dict.mddDict.AssetResolver().Read(withoutLeadingSlash)
			if err == nil {
				mimeType := GetMimeType(assetPath)
				return data, mimeType, nil
			}
		}
	}

	// 2. Try resource subdirectory (e.g. dictDir/DictName/style.css)
	if dict.resDir != "" {
		fsPath := filepath.Join(e.dictDir, dict.resDir, filepath.FromSlash(assetPath))
		if isPathSafe(e.dictDir, fsPath) {
			if data, err := os.ReadFile(fsPath); err == nil {
				mimeType := GetMimeType(assetPath)
				return data, mimeType, nil
			}
		}
	}

	// 3. Try dictDir root (e.g. dictDir/style.css)
	fsPath := filepath.Join(e.dictDir, filepath.FromSlash(assetPath))
	if !isPathSafe(e.dictDir, fsPath) {
		return nil, "", fmt.Errorf("asset not found: %s", assetPath)
	}
	if data, err := os.ReadFile(fsPath); err == nil {
		mimeType := GetMimeType(assetPath)
		return data, mimeType, nil
	}

	return nil, "", fmt.Errorf("asset not found: %s", assetPath)
}

// DebugLogAssetMiss logs when an asset lookup fails (for troubleshooting dictionary styling).
func (e *Engine) DebugLogAssetMiss(dictID, assetPath string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	dict, exists := e.dicts[dictID]
	if !exists {
		log.Debug().Str("dictID", dictID).Str("assetPath", assetPath).Msg("Asset miss: dict not found")
		return
	}
	log.Debug().
		Str("dictID", dictID).
		Str("assetPath", assetPath).
		Bool("hasMdd", dict.mddDict != nil).
		Str("resDir", dict.resDir).
		Msg("Asset miss: not found in MDD, resDir, or dictDir")
}
