package dict

import (
	"sort"
	"strings"
)

type indexedWord struct {
	word       string
	normalized string
}

func newWordIndex(words []string) []indexedWord {
	seen := make(map[string]struct{}, len(words))
	index := make([]indexedWord, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		if _, exists := seen[word]; exists {
			continue
		}
		seen[word] = struct{}{}
		index = append(index, indexedWord{word: word, normalized: strings.ToLower(word)})
	}
	sortWordIndex(index)
	return index
}

func sortWordIndex(index []indexedWord) {
	sort.Slice(index, func(i, j int) bool {
		if index[i].normalized == index[j].normalized {
			return index[i].word < index[j].word
		}
		return index[i].normalized < index[j].normalized
	})
}

// prefixMatches locates both bounds without scanning or copying the matches.
// The caller copies only the requested page, making search O(log n + pageSize).
// The returned slice is immutable and valid while the engine read lock is held.
func prefixMatches(index []indexedWord, keyword string) []indexedWord {
	start := sort.Search(len(index), func(i int) bool {
		return index[i].normalized >= keyword
	})
	end := start + sort.Search(len(index)-start, func(i int) bool {
		return !strings.HasPrefix(index[start+i].normalized, keyword)
	})
	return index[start:end]
}
