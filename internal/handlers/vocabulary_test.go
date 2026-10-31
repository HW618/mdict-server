package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/HW618/mdict-server/internal/auth"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/gin-gonic/gin"
)

type vocabularyEntryResponse struct {
	Code int `json:"code"`
	Data struct {
		Word  string `json:"word"`
		Level int    `json:"level"`
	} `json:"data"`
}

type vocabularyListResponse struct {
	Code int `json:"code"`
	Data struct {
		Items []struct {
			Word  string `json:"word"`
			Level int    `json:"level"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"data"`
}

type vocabularyTestEnv struct {
	t      *testing.T
	dir    string
	db     *store.SQLiteStore
	router *gin.Engine
	tokens []string
	users  []*models.User
}

func newVocabularyTestEnv(t *testing.T) *vocabularyTestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	env := &vocabularyTestEnv{t: t, dir: t.TempDir()}
	var err error
	env.db, err = store.NewSQLiteStore(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = env.db.Close() })
	users := store.NewUserStore(env.db)
	jwt := auth.NewJWTManager("test-only-jwt-secret", time.Hour, time.Hour)
	for _, name := range []string{"vocab-user-a", "vocab-user-b"} {
		user := models.NewUser(name, "test-password-hash")
		user.CanUseAPI = true
		if err := users.Create(user); err != nil {
			t.Fatal(err)
		}
		token, err := jwt.GenerateAccessToken(user)
		if err != nil {
			t.Fatal(err)
		}
		env.users = append(env.users, user)
		env.tokens = append(env.tokens, token)
	}
	env.bindRouter(jwt)
	return env
}

func (env *vocabularyTestEnv) bindRouter(jwt *auth.JWTManager) {
	env.t.Helper()
	env.router = gin.New()
	middleware := auth.NewAuthMiddleware(jwt, store.NewUserStore(env.db))
	handler := NewVocabularyHandler(store.NewVocabularyStore(env.db))
	group := env.router.Group("/vocabulary", middleware.RequireAuth())
	group.GET("", handler.List)
	group.GET("/entry", handler.Get)
	group.PUT("/entry", handler.Set)
	group.DELETE("/entry", handler.Delete)
}

func (env *vocabularyTestEnv) request(method, path, token string, body any) *httptest.ResponseRecorder {
	env.t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			env.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &b)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	return rec
}

func (env *vocabularyTestEnv) entry(method, word, token string) (*httptest.ResponseRecorder, vocabularyEntryResponse) {
	env.t.Helper()
	rec := env.request(method, "/vocabulary/entry?word="+url.QueryEscape(word), token, nil)
	var response vocabularyEntryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		env.t.Fatalf("decode entry response: %v: %s", err, rec.Body.String())
	}
	return rec, response
}

func (env *vocabularyTestEnv) set(word string, level int, token string) *httptest.ResponseRecorder {
	env.t.Helper()
	return env.request(http.MethodPut, "/vocabulary/entry", token, map[string]any{"word": word, "level": level})
}

func (env *vocabularyTestEnv) list(query, token string) (*httptest.ResponseRecorder, vocabularyListResponse) {
	env.t.Helper()
	rec := env.request(http.MethodGet, "/vocabulary"+query, token, nil)
	var response vocabularyListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		env.t.Fatalf("decode list response: %v: %s", err, rec.Body.String())
	}
	return rec, response
}

func TestVocabularyLifecyclePersistenceAndCase(t *testing.T) {
	env := newVocabularyTestEnv(t)
	token := env.tokens[0]
	rec, entry := env.entry(http.MethodGet, "  apple  ", token)
	if rec.Code != http.StatusOK || entry.Data.Word != "apple" || entry.Data.Level != 0 {
		t.Fatalf("missing: %d %s", rec.Code, rec.Body.String())
	}
	rec = env.set("  Apple  ", 2, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("set: %d %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &entry); err != nil || strings.TrimSpace(entry.Data.Word) != entry.Data.Word || !strings.EqualFold(entry.Data.Word, "apple") || entry.Data.Level != 2 {
		t.Fatalf("set response: %s (%v)", rec.Body.String(), err)
	}
	rec = env.set("APPLE", 5, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	rec, list := env.list("", token)
	if rec.Code != http.StatusOK || list.Data.Total != 1 || len(list.Data.Items) != 1 || list.Data.Items[0].Level != 5 {
		t.Fatalf("expected one updated entry: %d %s", rec.Code, rec.Body.String())
	}
	if err := env.db.Close(); err != nil {
		t.Fatal(err)
	}
	var err error
	env.db, err = store.NewSQLiteStore(env.dir)
	if err != nil {
		t.Fatal(err)
	}
	env.bindRouter(auth.NewJWTManager("test-only-jwt-secret", time.Hour, time.Hour))
	rec, entry = env.entry(http.MethodGet, "aPpLe", token)
	if rec.Code != http.StatusOK || entry.Data.Level != 5 {
		t.Fatalf("after reopening database: %d %s", rec.Code, rec.Body.String())
	}
	rec, _ = env.entry(http.MethodDelete, "  ApPlE  ", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	rec, entry = env.entry(http.MethodGet, "apple", token)
	if rec.Code != http.StatusOK || entry.Data.Level != 0 {
		t.Fatalf("after delete: %d %s", rec.Code, rec.Body.String())
	}
}

func TestVocabularyRequiresLoginAndIsolatesUsers(t *testing.T) {
	env := newVocabularyTestEnv(t)
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/vocabulary", nil}, {http.MethodGet, "/vocabulary/entry?word=apple", nil},
		{http.MethodPut, "/vocabulary/entry", map[string]any{"word": "apple", "level": 1}}, {http.MethodDelete, "/vocabulary/entry?word=apple", nil},
	} {
		for _, token := range []string{"", env.users[0].APIToken} {
			rec := env.request(tc.method, tc.path, token, tc.body)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s accepted non-session credential: %d %s", tc.method, tc.path, rec.Code, rec.Body.String())
			}
		}
	}
	if rec := env.set("private", 3, env.tokens[0]); rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	rec, entry := env.entry(http.MethodGet, "private", env.tokens[1])
	if rec.Code != http.StatusOK || entry.Data.Level != 0 {
		t.Fatalf("cross-user read: %d %s", rec.Code, rec.Body.String())
	}
	target := url.QueryEscape(env.users[0].ID)
	rec = env.request(http.MethodPut, "/vocabulary/entry?user_id="+target, env.tokens[1], map[string]any{"word": "private", "level": 5, "user_id": env.users[0].ID})
	if rec.Code != http.StatusOK {
		t.Fatalf("own set with forged user id: %d %s", rec.Code, rec.Body.String())
	}
	rec, entry = env.entry(http.MethodGet, "private", env.tokens[0])
	if rec.Code != http.StatusOK || entry.Data.Level != 3 {
		t.Fatalf("cross-user modification: %d %s", rec.Code, rec.Body.String())
	}
	rec = env.request(http.MethodDelete, "/vocabulary/entry?word=private&user_id="+target, env.tokens[1], nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("own delete with forged user id: %d %s", rec.Code, rec.Body.String())
	}
	rec, entry = env.entry(http.MethodGet, "private", env.tokens[0])
	if rec.Code != http.StatusOK || entry.Data.Level != 3 {
		t.Fatalf("cross-user deletion: %d %s", rec.Code, rec.Body.String())
	}
	rec, list := env.list("?user_id="+target, env.tokens[1])
	if rec.Code != http.StatusOK || list.Data.Total != 0 || len(list.Data.Items) != 0 {
		t.Fatalf("cross-user list: %d %s", rec.Code, rec.Body.String())
	}
}

func TestVocabularyValidatesWordAndLevel(t *testing.T) {
	env := newVocabularyTestEnv(t)
	for _, level := range []int{-1, 0, 6, 100} {
		if rec := env.set("valid", level, env.tokens[0]); rec.Code != http.StatusBadRequest {
			t.Errorf("level %d: %d %s", level, rec.Code, rec.Body.String())
		}
	}
	for _, word := range []string{"", "  ", "bad\nword", "bad\x00word", strings.Repeat("中", 201)} {
		if rec := env.set(word, 1, env.tokens[0]); rec.Code != http.StatusBadRequest {
			t.Errorf("invalid word %q: %d %s", word, rec.Code, rec.Body.String())
		}
	}
	if rec := env.set(strings.Repeat("中", 200), 1, env.tokens[0]); rec.Code != http.StatusOK {
		t.Fatalf("200 Unicode characters: %d %s", rec.Code, rec.Body.String())
	}
}

func TestVocabularyListFiltersAndPagination(t *testing.T) {
	env := newVocabularyTestEnv(t)
	for _, item := range []struct {
		word  string
		level int
	}{{"Apple", 1}, {"Application", 1}, {"Apricot", 2}, {"Pineapple", 2}, {"Banana", 3}} {
		if rec := env.set(item.word, item.level, env.tokens[0]); rec.Code != http.StatusOK {
			t.Fatalf("seed %s: %s", item.word, rec.Body.String())
		}
	}
	if rec := env.set("secret", 1, env.tokens[1]); rec.Code != http.StatusOK {
		t.Fatal(rec.Body.String())
	}
	rec, list := env.list("?level=1&keyword=app", env.tokens[0])
	if rec.Code != http.StatusOK || list.Data.Total != 2 || len(list.Data.Items) != 2 {
		t.Fatalf("combined filters: %d %s", rec.Code, rec.Body.String())
	}
	for _, item := range list.Data.Items {
		if item.Level != 1 || !strings.Contains(strings.ToLower(item.Word), "app") {
			t.Errorf("bad filtered item: %+v", item)
		}
	}
	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		rec, list = env.list(fmt.Sprintf("?page=%d&page_size=2", page), env.tokens[0])
		want := 2
		if page == 3 {
			want = 1
		}
		if rec.Code != http.StatusOK || list.Data.Total != 5 || len(list.Data.Items) != want {
			t.Fatalf("page %d: %d %s", page, rec.Code, rec.Body.String())
		}
		for _, item := range list.Data.Items {
			if seen[item.Word] {
				t.Fatalf("duplicate across pages: %s", item.Word)
			}
			seen[item.Word] = true
		}
	}
	if len(seen) != 5 {
		t.Fatalf("pagination lost records: %v", seen)
	}
}

func TestVocabularyPaginationDefaultsAndMaximum(t *testing.T) {
	env := newVocabularyTestEnv(t)
	for i := 0; i < 101; i++ {
		word := fmt.Sprintf("word-%03d", i)
		if rec := env.set(word, 4, env.tokens[0]); rec.Code != http.StatusOK {
			t.Fatalf("seed %s: %d %s", word, rec.Code, rec.Body.String())
		}
	}
	rec, list := env.list("", env.tokens[0])
	if rec.Code != http.StatusOK || list.Data.Total != 101 || len(list.Data.Items) != 30 {
		t.Fatalf("default page: %d %s", rec.Code, rec.Body.String())
	}
	rec, list = env.list("?page_size=100", env.tokens[0])
	if rec.Code != http.StatusOK || list.Data.Total != 101 || len(list.Data.Items) != 100 {
		t.Fatalf("maximum page size: %d %s", rec.Code, rec.Body.String())
	}
	rec, list = env.list("?page_size=1000", env.tokens[0])
	if rec.Code != http.StatusBadRequest && (rec.Code != http.StatusOK || len(list.Data.Items) > 100) {
		t.Fatalf("exceeded maximum page size: %d %s", rec.Code, rec.Body.String())
	}
}

func TestVocabularyUnicodeCaseFolding(t *testing.T) {
	env := newVocabularyTestEnv(t)
	if rec := env.set("ΟΣ", 1, env.tokens[0]); rec.Code != http.StatusOK {
		t.Fatalf("set uppercase: %d %s", rec.Code, rec.Body.String())
	}
	rec, entry := env.entry(http.MethodGet, "ος", env.tokens[0])
	if rec.Code != http.StatusOK || entry.Data.Level != 1 {
		t.Fatalf("lowercase final sigma lookup: %d %s", rec.Code, rec.Body.String())
	}
	if rec := env.set("ος", 4, env.tokens[0]); rec.Code != http.StatusOK {
		t.Fatalf("update lowercase: %d %s", rec.Code, rec.Body.String())
	}
	rec, list := env.list("?keyword="+url.QueryEscape("ος"), env.tokens[0])
	if rec.Code != http.StatusOK || list.Data.Total != 1 || len(list.Data.Items) != 1 || list.Data.Items[0].Level != 4 {
		t.Fatalf("case-folded filter: %d %s", rec.Code, rec.Body.String())
	}
}
