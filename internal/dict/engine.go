package dict

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	mdx "github.com/lib-x/mdx"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog/log"

	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
)

// Engine manages dictionary loading and querying
type Engine struct {
	mu         sync.RWMutex
	dicts      map[string]*LoadedDict
	dictDir    string
	dictStore  *store.DictStore
	sanitizer  *bluemonday.Policy
}

// LoadedDict represents a loaded dictionary with its parsed data
type LoadedDict struct {
	Info       *models.Dictionary
	mdxDict    *mdx.Mdict                 // MDX dictionary for word lookup
	mddDict    *mdx.Mdict                 // MDD resource file (nil if no .mdd)
	fuzzyStore *mdx.MemoryFuzzyIndexStore // fuzzy search index
	dictName   string                     // dictionary name used in fuzzy store (DictionaryInfo.Name)
}

// NewEngine creates a new dictionary engine
func NewEngine(dictDir string, dictStore *store.DictStore) *Engine {
	// Create a sanitization policy for dictionary HTML content.
	// Allow common HTML tags used in dictionary definitions.
	p := bluemonday.UGCPolicy()
	p.AllowElements("div", "span", "p", "b", "i", "u", "a", "img", "audio", "source",
		"br", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li",
		"table", "tr", "td", "th", "thead", "tbody", "font", "sup", "sub",
		"blockquote", "pre", "code", "dl", "dt", "dd")
	p.AllowAttrs("href", "src", "alt", "title", "class", "style", "width", "height",
		"controls", "autoplay", "loop", "type", "color", "size", "face").Globally()

	return &Engine{
		dicts:     make(map[string]*LoadedDict),
		dictDir:   dictDir,
		dictStore: dictStore,
		sanitizer: p,
	}
}

// LoadAll loads all dictionaries from the dictionary directory
func (e *Engine) LoadAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Scan directory for .mdx files
	entries, err := os.ReadDir(e.dictDir)
	if err != nil {
		return fmt.Errorf("failed to read dictionary directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(strings.ToLower(filename), ".mdx") {
			continue
		}

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
	dictInfo.Title = info.Title
	if dictInfo.Title == "" {
		dictInfo.Title = strings.TrimSuffix(filename, ".mdx")
	}
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
	return e.loadDict(loaded.Info.Filename)
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
		cleanHTML := e.sanitizer.Sanitize(string(htmlBytes))

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

// FuzzySearch performs a fuzzy search across all enabled dictionaries
func (e *Engine) FuzzySearch(keyword string, dictID string, page, pageSize int) (*models.FuzzySearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if len(keyword) < 2 {
		return nil, fmt.Errorf("keyword must be at least 2 characters")
	}

	// Collect matches from all dictionaries
	type wordMatch struct {
		word     string
		dictID   string
		dictName string
	}

	matches := []wordMatch{}

	for id, dict := range e.dicts {
		// Filter by dictID if specified
		if dictID != "" && id != dictID {
			continue
		}

		// Skip disabled dictionaries
		if !dict.Info.IsEnabled {
			continue
		}

		// Use fuzzy store for search (limit to get enough results for pagination)
		hits, err := dict.fuzzyStore.Search(dict.dictName, keyword, 1000)
		if err != nil || len(hits) == 0 {
			// Fallback: prefix-match scan from exported entries
			if err != nil {
				log.Warn().Err(err).Str("dict", id).Str("keyword", keyword).Msg("Fuzzy store miss, using prefix fallback")
			}
			entries, exportErr := dict.mdxDict.ExportEntries()
			if exportErr != nil {
				log.Warn().Err(exportErr).Str("dict", id).Msg("Failed to export entries for fallback")
				continue
			}
			for _, entry := range entries {
				if strings.HasPrefix(strings.ToLower(entry.Keyword), keyword) {
					matches = append(matches, wordMatch{
						word:     entry.Keyword,
						dictID:   id,
						dictName: dict.Info.Title,
					})
				}
			}
		} else {
			for _, hit := range hits {
				matches = append(matches, wordMatch{
					word:     hit.Entry.Keyword,
					dictID:   id,
					dictName: dict.Info.Title,
				})
			}
		}
	}

	// Sort matches alphabetically for consistent results
	sort.Slice(matches, func(i, j int) bool {
		return strings.ToLower(matches[i].word) < strings.ToLower(matches[j].word)
	})

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

	// Build result
	items := []models.FuzzyItem{}
	for _, match := range matches[start:end] {
		items = append(items, models.FuzzyItem{
			Word:     match.word,
			DictID:   match.dictID,
			DictName: match.dictName,
		})
	}

	return &models.FuzzySearchResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
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

	dict, exists := e.dicts[dictID]
	if !exists {
		return nil, "", fmt.Errorf("dictionary not found: %s", dictID)
	}

	if dict.mddDict == nil {
		return nil, "", fmt.Errorf("dictionary has no media files")
	}

	// Read asset from MDD using AssetResolver
	// Normalize path: MDD keys use backslash prefix like \image\foo.png
	normalizedPath := assetPath
	if !strings.HasPrefix(normalizedPath, "\\") {
		normalizedPath = "\\" + strings.ReplaceAll(normalizedPath, "/", "\\")
	}

	data, err := dict.mddDict.AssetResolver().Read(normalizedPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read asset: %w", err)
	}

	mimeType := GetMimeType(assetPath)
	return data, mimeType, nil
}
