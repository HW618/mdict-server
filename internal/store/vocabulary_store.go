package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/HW618/mdict-server/internal/models"
	"golang.org/x/text/cases"
)

type VocabularyStore struct{ store *SQLiteStore }

func NewVocabularyStore(store *SQLiteStore) *VocabularyStore { return &VocabularyStore{store: store} }

func vocabularyKey(word string) string { return cases.Fold().String(strings.TrimSpace(word)) }

func (s *VocabularyStore) Get(userID, word string) (*models.VocabularyEntry, error) {
	entry := &models.VocabularyEntry{Word: strings.TrimSpace(word)}
	err := s.store.db.QueryRow(`SELECT word, level, created_at, updated_at FROM vocabulary
		WHERE user_id = ? AND word_key = ?`, userID, vocabularyKey(entry.Word)).Scan(
		&entry.Word, &entry.Level, &entry.CreatedAt, &entry.UpdatedAt)
	if err == sql.ErrNoRows {
		return entry, nil
	}
	return entry, err
}

func (s *VocabularyStore) Set(userID, word string, level int) (*models.VocabularyEntry, error) {
	word = strings.TrimSpace(word)
	now := time.Now().UTC()
	_, err := s.store.db.Exec(`INSERT INTO vocabulary (user_id, word_key, word, level, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(user_id, word_key) DO UPDATE SET level=excluded.level, updated_at=excluded.updated_at`,
		userID, vocabularyKey(word), word, level, now, now)
	if err != nil {
		return nil, err
	}
	return s.Get(userID, word)
}

func (s *VocabularyStore) Delete(userID, word string) error {
	_, err := s.store.db.Exec(`DELETE FROM vocabulary WHERE user_id = ? AND word_key = ?`, userID, vocabularyKey(word))
	return err
}

func (s *VocabularyStore) List(userID string, level int, keyword string, page, pageSize int) ([]models.VocabularyEntry, int, error) {
	where := " WHERE user_id = ?"
	args := []any{userID}
	if level > 0 {
		where += " AND level = ?"
		args = append(args, level)
	}
	if keyword != "" {
		where += " AND instr(word_key, ?) > 0"
		args = append(args, vocabularyKey(keyword))
	}
	var total int
	if err := s.store.db.QueryRow("SELECT count(*) FROM vocabulary"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.store.db.Query("SELECT word, level, created_at, updated_at FROM vocabulary"+where+" ORDER BY updated_at DESC, word_key ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]models.VocabularyEntry, 0)
	for rows.Next() {
		var item models.VocabularyEntry
		if err := rows.Scan(&item.Word, &item.Level, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
