package models

import (
	"testing"
)

func TestNewDictionary(t *testing.T) {
	dict := NewDictionary("oxford.mdx", 1024000)

	if dict.ID == "" {
		t.Error("expected non-empty ID")
	}
	if len(dict.ID) != 8 {
		t.Errorf("expected ID length 8, got %d: '%s'", len(dict.ID), dict.ID)
	}
	if dict.Filename != "oxford.mdx" {
		t.Errorf("expected filename 'oxford.mdx', got '%s'", dict.Filename)
	}
	if dict.FileSize != 1024000 {
		t.Errorf("expected file size 1024000, got %d", dict.FileSize)
	}
	if !dict.IsEnabled {
		t.Error("expected dict to be enabled by default")
	}
	if dict.HasMdd {
		t.Error("expected HasMdd to be false by default")
	}
}

func TestGenerateDictID(t *testing.T) {
	// Same filename should produce same ID
	id1 := generateDictID("oxford.mdx")
	id2 := generateDictID("oxford.mdx")
	if id1 != id2 {
		t.Errorf("expected same ID for same filename, got '%s' and '%s'", id1, id2)
	}

	// Different filenames should produce different IDs
	id3 := generateDictID("cambridge.mdx")
	if id1 == id3 {
		t.Errorf("expected different IDs for different filenames, both got '%s'", id1)
	}

	// ID should be 8 hex chars
	if len(id1) != 8 {
		t.Errorf("expected ID length 8, got %d: '%s'", len(id1), id1)
	}
}

func TestDictToResponse(t *testing.T) {
	dict := &Dictionary{
		ID:          "abcd1234",
		Filename:    "test.mdx",
		Title:       "Test Dict",
		Description: "A test dictionary",
		FileSize:    1024,
		EntryCount:  100,
		IsEnabled:   true,
		HasMdd:      true,
	}

	resp := dict.ToResponse()

	if resp.ID != "abcd1234" {
		t.Errorf("expected ID 'abcd1234', got '%s'", resp.ID)
	}
	if resp.Filename != "test.mdx" {
		t.Errorf("expected filename 'test.mdx', got '%s'", resp.Filename)
	}
	if resp.Title != "Test Dict" {
		t.Errorf("expected title 'Test Dict', got '%s'", resp.Title)
	}
	if resp.Description != "A test dictionary" {
		t.Errorf("expected description 'A test dictionary', got '%s'", resp.Description)
	}
	if resp.FileSize != 1024 {
		t.Errorf("expected file size 1024, got %d", resp.FileSize)
	}
	if resp.EntryCount != 100 {
		t.Errorf("expected entry count 100, got %d", resp.EntryCount)
	}
	if !resp.IsEnabled {
		t.Error("expected IsEnabled to be true")
	}
	if !resp.HasMdd {
		t.Error("expected HasMdd to be true")
	}
}

func TestSearchResultStructure(t *testing.T) {
	result := SearchResult{
		Word: "hello",
		Results: []DictResult{
			{
				DictID:   "abc12345",
				DictName: "Oxford",
				HTML:     "<div>hello</div>",
				HasAudio: true,
				AudioURL: "/api/v1/assets/abc12345/hello.mp3",
			},
		},
	}

	if result.Word != "hello" {
		t.Errorf("expected word 'hello', got '%s'", result.Word)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].DictID != "abc12345" {
		t.Errorf("expected dict ID 'abc12345', got '%s'", result.Results[0].DictID)
	}
}

func TestFuzzySearchResultStructure(t *testing.T) {
	result := FuzzySearchResult{
		Items: []FuzzyItem{
			{Word: "hello"},
			{Word: "help"},
		},
		Total:      2,
		Page:       1,
		PageSize:   20,
		TotalPages: 1,
	}

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}
