package handlers

import (
	"encoding/json"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/HW618/mdict-server/internal/dict"
	"github.com/gin-gonic/gin"
)

func TestSearchValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewSearchHandler(dict.NewEngine("", nil))
	router.GET("/search", handler.Search)
	router.GET("/fuzzy", handler.FuzzySearch)
	for _, tt := range []struct {
		url    string
		status int
	}{
		{"/search?word=%20%20", http.StatusBadRequest},
		{"/fuzzy?keyword=%20%20", http.StatusBadRequest},
		{"/fuzzy?keyword=中", http.StatusBadRequest},
		{"/fuzzy?keyword=中文", http.StatusOK},
		{"/fuzzy?keyword=hello&page=9223372036854775807", http.StatusOK},
		{"/fuzzy?keyword=hello&page=0&page_size=0", http.StatusOK},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.url, nil))
		if response.Code != tt.status {
			t.Errorf("%s: got %d, want %d", tt.url, response.Code, tt.status)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
	}
}

func TestFuzzySearchDictionaryFilter(t *testing.T) {
	database, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	dir := t.TempDir()
	for _, name := range []string{"primary.mdx", "secondary.mdx"} {
		data, err := os.ReadFile(filepath.Join("..", "dict", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	engine := dict.NewEngine(dir, store.NewDictStore(database))
	if err := engine.LoadAll(); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/fuzzy", NewSearchHandler(engine).FuzzySearch)
	second := models.NewDictionary("secondary.mdx", 0).ID
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/fuzzy?keyword=ap&dict_id="+second, nil))
	var result struct {
		Data models.FuzzySearchResult `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || result.Data.Total != 2 || result.Data.Items[1].Word != "application" {
		t.Fatalf("filter ignored: %s", response.Body.String())
	}
}
