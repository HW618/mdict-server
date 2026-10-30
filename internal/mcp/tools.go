package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HW618/mdict-server/internal/models"
)

type searchWordInput struct {
	Word           string `json:"word" jsonschema:"Headword to look up, for example hello or 苹果."`
	DictID         string `json:"dict_id,omitempty" jsonschema:"Dictionary id from mdict_list_dictionaries. Omit it to search every enabled dictionary."`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"markdown (default) or json. markdown is the text shown to the model; structured fields are always returned."`
}

type entryOutput struct {
	DictID     string `json:"dict_id"`
	DictName   string `json:"dict_name"`
	Definition string `json:"definition"`
	Truncated  bool   `json:"truncated,omitempty"`
	HasAudio   bool   `json:"has_audio"`
	AudioURL   string `json:"audio_url,omitempty"`
}

type searchWordOutput struct {
	Word    string        `json:"word"`
	Count   int           `json:"count"`
	Results []entryOutput `json:"results"`
}

type fuzzySearchInput struct {
	Keyword        string `json:"keyword" jsonschema:"Case-insensitive prefix of at least 2 characters, for example app."`
	DictID         string `json:"dict_id,omitempty" jsonschema:"Limit matches to one dictionary id. Omit it to search every enabled dictionary."`
	Limit          int    `json:"limit,omitempty" jsonschema:"Maximum headwords to return. Default 20, maximum 100."`
	Offset         int    `json:"offset,omitempty" jsonschema:"Number of matches to skip. Default 0, maximum 10000."`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"markdown (default) or json."`
}

type fuzzyItem struct {
	Word string `json:"word"`
}

type fuzzySearchOutput struct {
	Keyword    string      `json:"keyword"`
	Items      []fuzzyItem `json:"items"`
	Total      int         `json:"total"`
	Count      int         `json:"count"`
	Offset     int         `json:"offset"`
	Limit      int         `json:"limit"`
	HasMore    bool        `json:"has_more"`
	NextOffset int         `json:"next_offset,omitempty"`
	Note       string      `json:"note,omitempty"`
}

type listDictionariesInput struct {
	EnabledOnly    bool   `json:"enabled_only,omitempty" jsonschema:"When true, omit dictionaries that are turned off."`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"markdown (default) or json."`
}

type dictionaryInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Filename    string `json:"filename"`
	EntryCount  int64  `json:"entry_count"`
	FileSize    int64  `json:"file_size"`
	IsEnabled   bool   `json:"is_enabled"`
	HasMDD      bool   `json:"has_mdd"`
}

type listDictionariesOutput struct {
	Dictionaries []dictionaryInfo `json:"dictionaries"`
	Total        int              `json:"total"`
	EnabledCount int              `json:"enabled_count"`
}

func (s *Service) registerTools() {
	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "mdict_search_word",
		Title:       "Search word",
		Description: "Look up the exact dictionary entry for one headword. Returns the definition from every enabled dictionary, or from dict_id when that is set. Call mdict_fuzzy_search first when the spelling may be incomplete.",
		Annotations: readOnlyAnnotations("Search word"),
	}, s.searchWord)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "mdict_fuzzy_search",
		Title:       "Fuzzy search",
		Description: "List headwords that start with a keyword. Matching ignores case and needs at least 2 characters. Page with limit and offset. Then call mdict_search_word with a returned headword.",
		Annotations: readOnlyAnnotations("Fuzzy search"),
	}, s.fuzzySearch)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "mdict_list_dictionaries",
		Title:       "List dictionaries",
		Description: "List the dictionaries installed on this server, including id, title, entry count, and whether each one is enabled. Use an id as dict_id in the search tools.",
		Annotations: readOnlyAnnotations("List dictionaries"),
	}, s.listDictionaries)
}

func (s *Service) searchWord(_ context.Context, _ *sdkmcp.CallToolRequest, input searchWordInput) (*sdkmcp.CallToolResult, searchWordOutput, error) {
	var zero searchWordOutput
	format, err := normalizeFormat(input.ResponseFormat)
	if err != nil {
		return nil, zero, err
	}
	word, err := cleanWord(input.Word)
	if err != nil {
		return nil, zero, err
	}
	dictID, err := s.readyDictionary(input.DictID)
	if err != nil {
		return nil, zero, err
	}

	result, err := s.engine.Search(word, dictID)
	if err != nil || result == nil {
		return nil, zero, fmt.Errorf("search failed for %q. Retry once. If it fails again, call mdict_list_dictionaries to confirm a dictionary is enabled", word)
	}
	if len(result.Results) == 0 {
		if dictID != "" {
			return nil, zero, fmt.Errorf("no entry for %q in dictionary %q. Call mdict_fuzzy_search with a shorter prefix and the same dict_id, or omit dict_id to search every enabled dictionary", word, dictID)
		}
		enabled, listErr := s.enabledCount()
		if listErr == nil && enabled == 0 {
			return nil, zero, fmt.Errorf("no enabled dictionaries are available, so %q cannot be looked up. Call mdict_list_dictionaries, then enable a dictionary in the admin panel", word)
		}
		return nil, zero, fmt.Errorf("no entry for %q. Call mdict_fuzzy_search with the first few characters, then call mdict_search_word with a headword from that list", word)
	}

	output := searchWordOutput{Word: result.Word, Results: make([]entryOutput, 0, len(result.Results))}
	if output.Word == "" {
		output.Word = word
	}
	for _, item := range result.Results {
		definition, truncated := truncateDefinition(htmlToMarkdown(item.HTML))
		output.Results = append(output.Results, entryOutput{
			DictID:     item.DictID,
			DictName:   item.DictName,
			Definition: definition,
			Truncated:  truncated,
			HasAudio:   item.HasAudio,
			AudioURL:   item.AudioURL,
		})
	}
	output.Count = len(output.Results)
	return toolResult(format, renderEntries(output.Word, output.Results), output)
}

func (s *Service) fuzzySearch(_ context.Context, _ *sdkmcp.CallToolRequest, input fuzzySearchInput) (*sdkmcp.CallToolResult, fuzzySearchOutput, error) {
	var zero fuzzySearchOutput
	format, err := normalizeFormat(input.ResponseFormat)
	if err != nil {
		return nil, zero, err
	}
	keyword, err := cleanKeyword(input.Keyword)
	if err != nil {
		return nil, zero, err
	}
	limit, offset, err := normalizePage(input.Limit, input.Offset)
	if err != nil {
		return nil, zero, err
	}
	dictID, err := s.readyDictionary(input.DictID)
	if err != nil {
		return nil, zero, err
	}

	total, matches, err := s.collectFuzzy(keyword, dictID, offset, limit)
	if err != nil {
		if strings.Contains(err.Error(), "at least 2") {
			return nil, zero, fmt.Errorf("keyword must be at least 2 characters. Pass a longer prefix, for example {\"keyword\":\"app\"}")
		}
		return nil, zero, fmt.Errorf("fuzzy search failed for %q. Retry once, or call mdict_list_dictionaries to confirm a dictionary is enabled", keyword)
	}

	output := fuzzySearchOutput{
		Keyword: keyword,
		Items:   make([]fuzzyItem, 0, len(matches)),
		Total:   total,
		Offset:  offset,
		Limit:   limit,
	}
	for _, match := range matches {
		output.Items = append(output.Items, fuzzyItem{Word: match.Word})
	}
	output.Count = len(output.Items)
	output.HasMore = offset+output.Count < total
	if output.HasMore {
		output.NextOffset = offset + output.Count
	}
	if total == 0 {
		output.Note = fmt.Sprintf("No headwords start with %q. Try a shorter prefix of at least 2 characters, or call mdict_list_dictionaries if no dictionary is enabled.", keyword)
	}

	text := renderFuzzy(output)
	return toolResult(format, text, output)
}

func (s *Service) listDictionaries(_ context.Context, _ *sdkmcp.CallToolRequest, input listDictionariesInput) (*sdkmcp.CallToolResult, listDictionariesOutput, error) {
	var zero listDictionariesOutput
	format, err := normalizeFormat(input.ResponseFormat)
	if err != nil {
		return nil, zero, err
	}
	listed, err := s.dictionaryCatalog(input.EnabledOnly)
	if err != nil {
		return nil, zero, err
	}
	output := listDictionariesOutput{Dictionaries: listed, Total: len(listed)}
	for _, dict := range listed {
		if dict.IsEnabled {
			output.EnabledCount++
		}
	}
	return toolResult(format, renderDictionaries(listed), output)
}

func (s *Service) readyDictionary(rawID string) (string, error) {
	dictID, err := cleanID(rawID)
	if err != nil || dictID == "" {
		return dictID, err
	}
	if s.dicts == nil {
		return dictID, nil
	}
	record, err := s.dicts.GetByID(dictID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("dictionary %q was not found. Call mdict_list_dictionaries and copy an id from that list", dictID)
		}
		return "", fmt.Errorf("could not read dictionary %q. Retry, or call mdict_list_dictionaries", dictID)
	}
	if !record.IsEnabled {
		title := record.Title
		if title == "" {
			title = record.Filename
		}
		return "", fmt.Errorf("dictionary %q (%s) is disabled. Enable it in the admin panel, or omit dict_id to search every enabled dictionary", dictID, title)
	}
	return dictID, nil
}

func (s *Service) enabledCount() (int, error) {
	catalog, err := s.dictionaryCatalog(true)
	if err != nil {
		return 0, err
	}
	return len(catalog), nil
}

func (s *Service) dictionaryCatalog(enabledOnly bool) ([]dictionaryInfo, error) {
	if s.dicts == nil {
		return nil, fmt.Errorf("dictionary catalog is unavailable. Retry after the server finishes starting")
	}
	records, err := s.dicts.List()
	if err != nil {
		return nil, fmt.Errorf("could not list dictionaries. Retry the call")
	}
	listed := make([]dictionaryInfo, 0, len(records))
	for _, record := range records {
		if enabledOnly && !record.IsEnabled {
			continue
		}
		listed = append(listed, dictionaryFromModel(record))
	}
	sortDictionaries(listed)
	return listed, nil
}

func sortDictionaries(listed []dictionaryInfo) {
	sort.Slice(listed, func(i, j int) bool {
		if listed[i].Title == listed[j].Title {
			return listed[i].ID < listed[j].ID
		}
		return listed[i].Title < listed[j].Title
	})
}

func dictionaryFromModel(record *models.Dictionary) dictionaryInfo {
	response := record.ToResponse()
	return dictionaryInfo{
		ID:          response.ID,
		Title:       displayTitle(response),
		Description: response.Description,
		Filename:    response.Filename,
		EntryCount:  response.EntryCount,
		FileSize:    response.FileSize,
		IsEnabled:   response.IsEnabled,
		HasMDD:      response.HasMdd,
	}
}

func (s *Service) collectFuzzy(keyword, dictID string, offset, limit int) (int, []models.FuzzyItem, error) {
	const window = 100
	skipped := 0
	items := make([]models.FuzzyItem, 0, limit)
	total := 0
	for page := 1; page <= (maxOffset/window)+1; page++ {
		result, err := s.engine.FuzzySearch(keyword, dictID, page, window)
		if err != nil {
			return 0, nil, err
		}
		total = result.Total
		if offset >= total || len(result.Items) == 0 {
			return total, items, nil
		}
		for _, item := range result.Items {
			if skipped < offset {
				skipped++
				continue
			}
			items = append(items, item)
			if len(items) == limit {
				return total, items, nil
			}
		}
		if page >= result.TotalPages {
			return total, items, nil
		}
	}
	return total, items, nil
}

func renderFuzzy(output fuzzySearchOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Matches for `%s`\n\n", output.Keyword)
	fmt.Fprintf(&b, "Showing %d of %d, starting at offset %d.\n\n", output.Count, output.Total, output.Offset)
	if output.Note != "" {
		b.WriteString(output.Note)
		b.WriteString("\n")
		return b.String()
	}
	for _, item := range output.Items {
		fmt.Fprintf(&b, "- %s\n", item.Word)
	}
	if output.HasMore {
		fmt.Fprintf(&b, "\nMore matches are available. Call mdict_fuzzy_search again with offset %d.\n", output.NextOffset)
	}
	return b.String()
}

func toolResult[T any](format, markdown string, output T) (*sdkmcp.CallToolResult, T, error) {
	text := markdown
	if format == formatJSON {
		encoded, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			var zero T
			return nil, zero, fmt.Errorf("could not encode the result. Retry the call")
		}
		text = string(encoded)
	}
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: text}},
	}, output, nil
}
