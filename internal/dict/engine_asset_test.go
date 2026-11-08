package dict

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HW618/mdict-server/internal/models"
)

func TestGetAssetStaysInsideDictionaryResources(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secret.mdx"), []byte("dictionary-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "primary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "primary", "ok.css"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".uploads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".uploads", "chunk.tmp"), []byte("chunk"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.css"), []byte("hidden"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.css"), filepath.Join(dir, "link.css")); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(dir, nil)
	engine.dicts["dict1"] = &LoadedDict{
		Info:   &models.Dictionary{ID: "dict1", Filename: "primary.mdx", IsEnabled: true},
		resDir: "primary",
	}

	data, _, err := engine.GetAsset("dict1", "ok.css")
	if err != nil || string(data) != "ok" {
		t.Fatalf("resource css: %v %q", err, data)
	}
	data, _, err = engine.GetAsset("dict1", "style.css")
	if err != nil || string(data) != "body{}" {
		t.Fatalf("sibling css: %v %q", err, data)
	}
	for _, path := range []string{"secret.mdx", ".uploads/chunk.tmp", "link.css", "../secret.css"} {
		if _, _, err := engine.GetAsset("dict1", path); err == nil {
			t.Errorf("served %s", path)
		}
	}
}
