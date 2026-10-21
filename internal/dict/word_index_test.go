package dict

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
)

func wordsFromResult(result *models.FuzzySearchResult) []string {
	words := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		words = append(words, item.Word)
	}
	return words
}

func TestFuzzySearchCaseUnicodeAndPagination(t *testing.T) {
	engine := NewEngine("", nil)
	engine.wordIndex = newWordIndex([]string{"Zebra", "apple", "Apple", "APPLE", "apricot", "Banana", "application", " apple ", "", "中文", "中文词典", "École", "éclair"})
	tests := []struct {
		name, keyword            string
		page, size, total, pages int
		want                     []string
	}{
		{"mixed case", " AP ", 1, 20, 5, 1, []string{"APPLE", "Apple", "apple", "application", "apricot"}},
		{"page one", "ap", 1, 2, 5, 3, []string{"APPLE", "Apple"}},
		{"page two", "ap", 2, 2, 5, 3, []string{"apple", "application"}},
		{"last page", "ap", 3, 2, 5, 3, []string{"apricot"}},
		{"past end", "ap", 4, 2, 5, 3, []string{}},
		{"overflow", "ap", math.MaxInt, 100, 5, 1, []string{}},
		{"normalized pagination", "ap", -1, 0, 5, 1, []string{"APPLE", "Apple", "apple", "application", "apricot"}},
		{"Chinese", " 中文 ", 1, 20, 2, 1, []string{"中文", "中文词典"}},
		{"Unicode case", "ÉC", 1, 20, 2, 1, []string{"éclair", "École"}},
		{"no matches", "az", 1, 20, 0, 1, []string{}},
		{"after index", "zz", 1, 20, 0, 1, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.FuzzySearch(tt.keyword, "", tt.page, tt.size)
			if err != nil {
				t.Fatal(err)
			}
			if got := wordsFromResult(result); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if result.Total != tt.total || result.TotalPages != tt.pages {
				t.Fatalf("unexpected counts: %+v", result)
			}
		})
	}
	for _, keyword := range []string{"", " ", "a", "中", "😀"} {
		if _, err := engine.FuzzySearch(keyword, "", 1, 20); err == nil {
			t.Errorf("expected error for %q", keyword)
		}
	}
}

func TestPrefixMatchesAgainstLinearScan(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	alphabet := []rune("aAbBéÉ中文")
	words := make([]string, 2000)
	for i := range words {
		runes := make([]rune, 2+rng.Intn(8))
		for j := range runes {
			runes[j] = alphabet[rng.Intn(len(alphabet))]
		}
		words[i] = string(runes)
	}
	index := newWordIndex(words)
	for _, a := range alphabet {
		for _, b := range alphabet {
			prefix := strings.ToLower(string([]rune{a, b}))
			var want, got []string
			for _, word := range index {
				if strings.HasPrefix(word.normalized, prefix) {
					want = append(want, word.word)
				}
			}
			for _, word := range prefixMatches(index, prefix) {
				got = append(got, word.word)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("prefix %q: got %v, want %v", prefix, got, want)
			}
		}
	}
}

func TestDictionaryIndexLifecycle(t *testing.T) {
	database, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	dictStore := store.NewDictStore(database)
	engine := NewEngine(t.TempDir(), dictStore)
	first := models.NewDictionary("first.mdx", 1)
	second := models.NewDictionary("second.mdx", 1)
	for _, info := range []*models.Dictionary{first, second} {
		if err := dictStore.Create(info); err != nil {
			t.Fatal(err)
		}
	}
	engine.dicts[first.ID] = &LoadedDict{Info: first, wordIndex: newWordIndex([]string{"apple", "apricot"})}
	engine.dicts[second.ID] = &LoadedDict{Info: second, wordIndex: newWordIndex([]string{"apple", "application"})}
	engine.rebuildWordIndex()
	assertWords := func(id string, want []string) {
		t.Helper()
		result, err := engine.FuzzySearch("ap", id, 1, 20)
		if err != nil {
			t.Fatal(err)
		}
		if got := wordsFromResult(result); !reflect.DeepEqual(got, want) {
			t.Fatalf("dict %q: got %v, want %v", id, got, want)
		}
	}
	assertWords("", []string{"apple", "application", "apricot"})
	assertWords(first.ID, []string{"apple", "apricot"})
	assertWords("missing", []string{})
	if err := engine.SetEnabled(first.ID, false); err != nil {
		t.Fatal(err)
	}
	assertWords("", []string{"apple", "application"})
	assertWords(first.ID, []string{})
	if err := engine.SetEnabled(first.ID, true); err != nil {
		t.Fatal(err)
	}
	assertWords(first.ID, []string{"apple", "apricot"})
	persisted, err := dictStore.GetByID(first.ID)
	if err != nil || !persisted.IsEnabled {
		t.Fatalf("state was not persisted: %v", err)
	}
	// A failed reload must preserve the working index.
	if err := engine.Reload(first.ID); err == nil {
		t.Fatal("expected missing-file error")
	}
	assertWords(first.ID, []string{"apple", "apricot"})
	engine.Unload(first.ID)
	assertWords("", []string{"apple", "application"})
	engine.Unload(second.ID)
	assertWords("", []string{})
}

var benchmarkResult *models.FuzzySearchResult
var benchmarkWords []string

// Both paths use the same broad prefix and 10-item page. The baseline reproduces
// the previous scan-and-copy algorithm; all lowercase data avoids its sorting bug.
func BenchmarkFuzzySearch(b *testing.B) {
	for _, count := range []int{100_000, 1_000_000} {
		words := make([]string, count)
		for i := range words {
			words[i] = fmt.Sprintf("prefix%07d", i)
		}
		engine := NewEngine("", nil)
		engine.wordIndex = newWordIndex(words)
		b.Run(fmt.Sprintf("Indexed/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkResult, _ = engine.FuzzySearch("pre", "", 1, 10)
			}
		})
		b.Run(fmt.Sprintf("Baseline/%d", count), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				start := sort.Search(len(words), func(j int) bool { return strings.ToLower(words[j]) >= "pre" })
				var matches []string
				for _, word := range words[start:] {
					if !strings.HasPrefix(strings.ToLower(word), "pre") {
						break
					}
					matches = append(matches, word)
				}
				benchmarkWords = matches[:min(10, len(matches))]
			}
		})
	}
}

func TestMDXLoadReloadAndConcurrentSearch(t *testing.T) {
	database, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	dictStore := store.NewDictStore(database)
	dir := t.TempDir()
	for _, name := range []string{"primary.mdx", "secondary.mdx"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	engine := NewEngine(dir, dictStore)
	if err := engine.LoadAll(); err != nil {
		t.Fatal(err)
	}
	result, err := engine.FuzzySearch("ap", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got := wordsFromResult(result); !reflect.DeepEqual(got, []string{"Apple", "apple", "application", "apricot"}) {
		t.Fatalf("unexpected MDX index: %v", got)
	}
	primary := models.NewDictionary("primary.mdx", 0).ID
	if err := dictStore.UpdateTitle(primary, "Custom title"); err != nil {
		t.Fatal(err)
	}
	engine.SetTitle(primary, "Custom title")
	if err := engine.Reload(primary); err != nil {
		t.Fatal(err)
	}
	exact, err := engine.Search("apple", primary)
	if err != nil || len(exact.Results) != 1 || exact.Results[0].DictName != "Custom title" {
		t.Fatalf("title or exact lookup lost after reload: %+v, %v", exact, err)
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if _, err := engine.FuzzySearch("ap", "", 1, 10); err != nil {
					t.Error(err)
				}
				if _, err := engine.Search("apple", primary); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	for range 10 {
		if err := engine.SetEnabled(primary, false); err != nil {
			t.Fatal(err)
		}
		if err := engine.SetEnabled(primary, true); err != nil {
			t.Fatal(err)
		}
		engine.SetTitle(primary, "Custom title")
	}
	wg.Wait()
	engine.Unload(primary)
	if err := engine.SetEnabled(primary, true); err != nil {
		t.Fatal(err)
	}
	result, err = engine.FuzzySearch("ap", primary, 1, 20)
	if err != nil || result.Total != 3 {
		t.Fatalf("failed to reload unloaded dictionary: %+v, %v", result, err)
	}
}
