package models

import "time"

// VocabularyEntry is a word saved in one user's notebook.
type VocabularyEntry struct {
	Word      string    `json:"word"`
	Level     int       `json:"level"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
