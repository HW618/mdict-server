package mcp

import (
	"fmt"
	"strings"
	"unicode/utf8"

	markdown "github.com/JohannesKaufmann/html-to-markdown/v2"

	"github.com/HW618/mdict-server/internal/models"
)

const (
	formatMarkdown = "markdown"
	formatJSON     = "json"

	maxWordRunes       = 200
	maxKeywordRunes    = 100
	maxDefinitionRunes = 12000
	defaultLimit       = 20
	maxLimit           = 100
	maxOffset          = 10000
)

func normalizeFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", formatMarkdown:
		return formatMarkdown, nil
	case formatJSON:
		return formatJSON, nil
	default:
		return "", fmt.Errorf("response_format must be \"markdown\" or \"json\", not %q", format)
	}
}

func cleanWord(word string) (string, error) {
	word = strings.TrimSpace(word)
	if word == "" {
		return "", fmt.Errorf("word is required. Pass the headword to look up, for example {\"word\":\"hello\"}")
	}
	if strings.ContainsRune(word, 0) {
		return "", fmt.Errorf("word contains a null character. Pass the plain headword")
	}
	if count := utf8.RuneCountInString(word); count > maxWordRunes {
		return "", fmt.Errorf("word is %d characters. Pass one headword of at most %d characters", count, maxWordRunes)
	}
	return word, nil
}

func cleanKeyword(keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if utf8.RuneCountInString(keyword) < 2 {
		return "", fmt.Errorf("keyword must be at least 2 characters. Pass a longer prefix, for example {\"keyword\":\"app\"}")
	}
	if count := utf8.RuneCountInString(keyword); count > maxKeywordRunes {
		return "", fmt.Errorf("keyword is %d characters. Pass a prefix of at most %d characters", count, maxKeywordRunes)
	}
	return keyword, nil
}

func cleanID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil
	}
	if strings.ContainsAny(id, "/\\?# \t\r\n") || strings.ContainsRune(id, 0) {
		return "", fmt.Errorf("dict_id %q is not a dictionary id. Call mdict_list_dictionaries and copy an id from that list", id)
	}
	return id, nil
}

func normalizePage(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return 0, 0, fmt.Errorf("limit must be from 1 to %d, not %d. Use offset to read the next page", maxLimit, limit)
	}
	if offset < 0 || offset > maxOffset {
		return 0, 0, fmt.Errorf("offset must be from 0 to %d, not %d. Narrow keyword or dict_id instead of skipping further", maxOffset, offset)
	}
	return limit, offset, nil
}

func htmlToMarkdown(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	converted, err := markdown.ConvertString(html)
	if err != nil {
		return strings.TrimSpace(html)
	}
	return strings.TrimSpace(converted)
}

func truncateDefinition(text string) (string, bool) {
	if utf8.RuneCountInString(text) <= maxDefinitionRunes {
		return text, false
	}
	cut := string([]rune(text)[:maxDefinitionRunes])
	return cut + "\n\n[truncated: this entry is longer than 12000 characters. Pass dict_id to read one dictionary, or look up a more specific headword.]", true
}

func displayTitle(dict models.DictResponse) string {
	if strings.TrimSpace(dict.Title) != "" {
		return dict.Title
	}
	return dict.Filename
}

func renderEntries(word string, entries []entryOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", word)
	if len(entries) == 0 {
		b.WriteString("No entries.\n")
		return b.String()
	}
	for i, entry := range entries {
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		fmt.Fprintf(&b, "## %s (`%s`)\n\n", entry.DictName, entry.DictID)
		if entry.Definition != "" {
			b.WriteString(entry.Definition)
			b.WriteString("\n")
		} else {
			b.WriteString("_This dictionary returned an empty entry._\n")
		}
		if entry.Truncated {
			b.WriteString("\n")
		}
		if entry.AudioURL != "" {
			fmt.Fprintf(&b, "\nAudio: `%s`\n", entry.AudioURL)
		}
	}
	return b.String()
}

func renderDictionaries(dicts []dictionaryInfo) string {
	var b strings.Builder
	b.WriteString("# Dictionaries\n\n")
	if len(dicts) == 0 {
		b.WriteString("No dictionaries are installed. Add an .mdx file on the server or upload one from the admin panel.\n")
		return b.String()
	}
	for _, dict := range dicts {
		state := "disabled"
		if dict.IsEnabled {
			state = "enabled"
		}
		fmt.Fprintf(&b, "- **%s** (`%s`) — %s, %d entries, `%s`\n", dict.Title, dict.ID, state, dict.EntryCount, dict.Filename)
	}
	return b.String()
}
