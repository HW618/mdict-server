package mcp

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/HW618/mdict-server/internal/dict"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
)

func TestDictionaryToolsResourcesAndPrompts(t *testing.T) {
	session := connect(t, newTestService(t))
	ctx := context.Background()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s is not marked read-only", tool.Name)
		}
	}
	for _, name := range []string{"mdict_search_word", "mdict_fuzzy_search", "mdict_list_dictionaries"} {
		if !names[name] {
			t.Errorf("missing tool %s", name)
		}
	}

	catalog := callTool(t, session, "mdict_list_dictionaries", map[string]any{})
	if catalog.IsError || !strings.Contains(textOf(t, catalog), "primary") {
		t.Fatalf("catalog: %#v", catalog)
	}

	exact := callTool(t, session, "mdict_search_word", map[string]any{"word": "apple"})
	body := textOf(t, exact)
	if exact.IsError || !strings.Contains(body, "apple") {
		t.Fatalf("search: %s", body)
	}
	structured, ok := exact.StructuredContent.(map[string]any)
	if !ok || structured["count"].(float64) < 1 {
		t.Fatalf("structured search: %#v", exact.StructuredContent)
	}

	missing := callTool(t, session, "mdict_search_word", map[string]any{"word": "not-a-word"})
	if !missing.IsError || !strings.Contains(textOf(t, missing), "mdict_fuzzy_search") {
		t.Fatalf("missing word: %s", textOf(t, missing))
	}

	unknown := callTool(t, session, "mdict_search_word", map[string]any{"word": "apple", "dict_id": "missing"})
	if !unknown.IsError || !strings.Contains(textOf(t, unknown), "mdict_list_dictionaries") {
		t.Fatalf("unknown dict: %s", textOf(t, unknown))
	}

	fuzzy := callTool(t, session, "mdict_fuzzy_search", map[string]any{
		"keyword": "ap",
		"limit":   1,
		"offset":  0,
	})
	fuzzyBody := textOf(t, fuzzy)
	if fuzzy.IsError || !strings.Contains(fuzzyBody, "offset 1") {
		t.Fatalf("fuzzy: %s", fuzzyBody)
	}
	short := callTool(t, session, "mdict_fuzzy_search", map[string]any{"keyword": "a"})
	if !short.IsError {
		t.Fatal("expected a short keyword to fail")
	}

	primary := dictID(t, "primary.mdx")
	resource, err := session.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "mdict://entries/" + primary + "/apple"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resource.Contents) != 1 || !strings.Contains(resource.Contents[0].Text, "apple") {
		t.Fatalf("entry resource: %#v", resource.Contents)
	}

	prompt, err := session.GetPrompt(ctx, &sdkmcp.GetPromptParams{
		Name:      "mdict_define_word",
		Arguments: map[string]string{"word": "apple", "dict_id": primary},
	})
	if err != nil {
		t.Fatal(err)
	}
	message := prompt.Messages[0].Content.(*sdkmcp.TextContent).Text
	if !strings.Contains(message, "using only the dictionary text") && !strings.Contains(message, "Explain") {
		t.Fatalf("prompt: %s", message)
	}
	if !strings.Contains(message, "apple") {
		t.Fatalf("prompt missing entry: %s", message)
	}

	// A missing entry is a protocol error. On protocol 2026-07-28 that is HTTP 400,
	// which closes this client session, so it uses its own connection.
	missingSession := connect(t, newTestService(t))
	_, err = missingSession.ReadResource(ctx, &sdkmcp.ReadResourceParams{URI: "mdict://entries/" + primary + "/missing"})
	if err == nil || !strings.Contains(err.Error(), "Resource not found") {
		t.Fatalf("missing entry: %v", err)
	}
}

func TestInputValidation(t *testing.T) {
	if _, err := normalizeFormat("html"); err == nil {
		t.Fatal("html format was accepted")
	}
	if _, _, err := normalizePage(0, 0); err != nil {
		t.Fatal(err)
	}
	if limit, offset, err := normalizePage(0, 0); err != nil || limit != 20 || offset != 0 {
		t.Fatalf("default page: %d %d %v", limit, offset, err)
	}
	text, truncated := truncateDefinition(strings.Repeat("字", maxDefinitionRunes+1))
	if !truncated || !strings.Contains(text, "truncated") {
		t.Fatalf("truncation failed: %t %d", truncated, len(text))
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
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
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	engine := dict.NewEngine(dir, store.NewDictStore(database))
	if err := engine.LoadAll(); err != nil {
		t.Fatal(err)
	}
	return New(engine, store.NewDictStore(database), "test")
}

func connect(t *testing.T, service *Service) *sdkmcp.ClientSession {
	t.Helper()
	httpServer := httptest.NewServer(service.Handler())
	t.Cleanup(httpServer.Close)
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "mdict-test", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), &sdkmcp.StreamableClientTransport{
		Endpoint:             httpServer.URL,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callTool(t *testing.T, session *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func textOf(t *testing.T, result *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("tool returned no content")
	}
	block, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("content type %T", result.Content[0])
	}
	return block.Text
}

func dictID(t *testing.T, filename string) string {
	t.Helper()
	return models.NewDictionary(filename, 0).ID
}
