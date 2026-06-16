package models

import (
	"crypto/md5"
	"fmt"
	"time"
)

// Dictionary represents a loaded dictionary
type Dictionary struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	FileSize    int64     `json:"file_size"`
	EntryCount  int64     `json:"entry_count"`
	IsEnabled   bool      `json:"is_enabled"`
	HasMdd      bool      `json:"has_mdd"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewDictionary creates a new dictionary entry
func NewDictionary(filename string, fileSize int64) *Dictionary {
	now := time.Now()
	return &Dictionary{
		ID:        generateDictID(filename),
		Filename:  filename,
		FileSize:  fileSize,
		IsEnabled: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateDictID generates a unique ID for a dictionary based on filename
func generateDictID(filename string) string {
	hash := md5.Sum([]byte(filename))
	return fmt.Sprintf("%x", hash[:4])[:8]
}

// DictResponse represents the dictionary data in API responses
type DictResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	FileSize    int64     `json:"file_size"`
	EntryCount  int64     `json:"entry_count"`
	IsEnabled   bool      `json:"is_enabled"`
	HasMdd      bool      `json:"has_mdd"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts Dictionary to DictResponse
func (d *Dictionary) ToResponse() DictResponse {
	return DictResponse{
		ID:          d.ID,
		Filename:    d.Filename,
		Title:       d.Title,
		Description: d.Description,
		FileSize:    d.FileSize,
		EntryCount:  d.EntryCount,
		IsEnabled:   d.IsEnabled,
		HasMdd:      d.HasMdd,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// DictStatusUpdateRequest represents the request to update dictionary status
type DictStatusUpdateRequest struct {
	IsEnabled bool `json:"is_enabled"`
}

// DictTitleUpdateRequest represents the request to update dictionary title
type DictTitleUpdateRequest struct {
	Title string `json:"title" binding:"required"`
}

// SearchResult represents a search result
type SearchResult struct {
	Word    string           `json:"word"`
	Results []DictResult     `json:"results"`
}

// DictResult represents a result from a single dictionary
type DictResult struct {
	DictID   string `json:"dict_id"`
	DictName string `json:"dict_name"`
	HTML     string `json:"html"`
	HasAudio bool   `json:"has_audio"`
	AudioURL string `json:"audio_url,omitempty"`
}

// FuzzySearchResult represents fuzzy search results
type FuzzySearchResult struct {
	Items      []FuzzyItem `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// FuzzyItem represents a single fuzzy search result
type FuzzyItem struct {
	Word     string `json:"word"`
	DictID   string `json:"dict_id"`
	DictName string `json:"dict_name"`
}
