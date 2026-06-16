package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/HW618/mdict-server/internal/models"
)

// DictStore handles dictionary database operations
type DictStore struct {
	store *SQLiteStore
}

// NewDictStore creates a new dictionary store
func NewDictStore(store *SQLiteStore) *DictStore {
	return &DictStore{store: store}
}

// Create creates a new dictionary entry
func (s *DictStore) Create(dict *models.Dictionary) error {
	query := `
		INSERT INTO dicts (id, filename, title, description, file_size, entry_count, is_enabled, has_mdd, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.store.db.Exec(query,
		dict.ID,
		dict.Filename,
		dict.Title,
		dict.Description,
		dict.FileSize,
		dict.EntryCount,
		dict.IsEnabled,
		dict.HasMdd,
		dict.CreatedAt,
		dict.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create dictionary: %w", err)
	}

	return nil
}

// GetByID retrieves a dictionary by ID
func (s *DictStore) GetByID(id string) (*models.Dictionary, error) {
	query := `
		SELECT id, filename, title, description, file_size, entry_count, is_enabled, has_mdd, created_at, updated_at
		FROM dicts
		WHERE id = ?
	`

	dict := &models.Dictionary{}
	err := s.store.db.QueryRow(query, id).Scan(
		&dict.ID,
		&dict.Filename,
		&dict.Title,
		&dict.Description,
		&dict.FileSize,
		&dict.EntryCount,
		&dict.IsEnabled,
		&dict.HasMdd,
		&dict.CreatedAt,
		&dict.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dictionary not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get dictionary: %w", err)
	}

	return dict, nil
}

// GetByFilename retrieves a dictionary by filename
func (s *DictStore) GetByFilename(filename string) (*models.Dictionary, error) {
	query := `
		SELECT id, filename, title, description, file_size, entry_count, is_enabled, has_mdd, created_at, updated_at
		FROM dicts
		WHERE filename = ?
	`

	dict := &models.Dictionary{}
	err := s.store.db.QueryRow(query, filename).Scan(
		&dict.ID,
		&dict.Filename,
		&dict.Title,
		&dict.Description,
		&dict.FileSize,
		&dict.EntryCount,
		&dict.IsEnabled,
		&dict.HasMdd,
		&dict.CreatedAt,
		&dict.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("dictionary not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get dictionary: %w", err)
	}

	return dict, nil
}

// List retrieves all dictionaries
func (s *DictStore) List() ([]*models.Dictionary, error) {
	query := `
		SELECT id, filename, title, description, file_size, entry_count, is_enabled, has_mdd, created_at, updated_at
		FROM dicts
		ORDER BY filename ASC
	`

	rows, err := s.store.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list dictionaries: %w", err)
	}
	defer rows.Close()

	var dicts []*models.Dictionary
	for rows.Next() {
		dict := &models.Dictionary{}
		err := rows.Scan(
			&dict.ID,
			&dict.Filename,
			&dict.Title,
			&dict.Description,
			&dict.FileSize,
			&dict.EntryCount,
			&dict.IsEnabled,
			&dict.HasMdd,
			&dict.CreatedAt,
			&dict.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dictionary: %w", err)
		}
		dicts = append(dicts, dict)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dictionaries: %w", err)
	}

	return dicts, nil
}

// ListEnabled retrieves all enabled dictionaries
func (s *DictStore) ListEnabled() ([]*models.Dictionary, error) {
	query := `
		SELECT id, filename, title, description, file_size, entry_count, is_enabled, has_mdd, created_at, updated_at
		FROM dicts
		WHERE is_enabled = TRUE
		ORDER BY filename ASC
	`

	rows, err := s.store.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled dictionaries: %w", err)
	}
	defer rows.Close()

	var dicts []*models.Dictionary
	for rows.Next() {
		dict := &models.Dictionary{}
		err := rows.Scan(
			&dict.ID,
			&dict.Filename,
			&dict.Title,
			&dict.Description,
			&dict.FileSize,
			&dict.EntryCount,
			&dict.IsEnabled,
			&dict.HasMdd,
			&dict.CreatedAt,
			&dict.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dictionary: %w", err)
		}
		dicts = append(dicts, dict)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating dictionaries: %w", err)
	}

	return dicts, nil
}

// Update updates a dictionary
func (s *DictStore) Update(dict *models.Dictionary) error {
	query := `
		UPDATE dicts
		SET title = ?, description = ?, file_size = ?, entry_count = ?, is_enabled = ?, has_mdd = ?, updated_at = ?
		WHERE id = ?
	`

	dict.UpdatedAt = time.Now()

	_, err := s.store.db.Exec(query,
		dict.Title,
		dict.Description,
		dict.FileSize,
		dict.EntryCount,
		dict.IsEnabled,
		dict.HasMdd,
		dict.UpdatedAt,
		dict.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update dictionary: %w", err)
	}

	return nil
}

// UpdateStatus updates dictionary enabled status
func (s *DictStore) UpdateStatus(id string, isEnabled bool) error {
	query := `
		UPDATE dicts
		SET is_enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.store.db.Exec(query, isEnabled, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update dictionary status: %w", err)
	}

	return nil
}

// UpdateTitle updates dictionary title
func (s *DictStore) UpdateTitle(id, title string) error {
	query := `
		UPDATE dicts
		SET title = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.store.db.Exec(query, title, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update dictionary title: %w", err)
	}

	return nil
}

// Delete deletes a dictionary
func (s *DictStore) Delete(id string) error {
	query := `DELETE FROM dicts WHERE id = ?`

	result, err := s.store.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete dictionary: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dictionary not found")
	}

	return nil
}

// Count returns the total number of dictionaries
func (s *DictStore) Count() (int, error) {
	query := `SELECT COUNT(*) FROM dicts`

	var count int
	err := s.store.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count dictionaries: %w", err)
	}

	return count, nil
}

// CountEnabled returns the number of enabled dictionaries
func (s *DictStore) CountEnabled() (int, error) {
	query := `SELECT COUNT(*) FROM dicts WHERE is_enabled = TRUE`

	var count int
	err := s.store.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count enabled dictionaries: %w", err)
	}

	return count, nil
}

// ExistsByFilename checks if a dictionary exists with the given filename
func (s *DictStore) ExistsByFilename(filename string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM dicts WHERE filename = ?)`

	var exists bool
	err := s.store.db.QueryRow(query, filename).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check dictionary existence: %w", err)
	}

	return exists, nil
}

// GetStats returns dictionary statistics
func (s *DictStore) GetStats() (int, int, int64, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_enabled THEN 1 ELSE 0 END), 0) as enabled,
			COALESCE(SUM(entry_count), 0) as total_entries
		FROM dicts
	`

	var total, enabled int
	var totalEntries int64
	err := s.store.db.QueryRow(query).Scan(&total, &enabled, &totalEntries)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get dictionary stats: %w", err)
	}

	return total, enabled, totalEntries, nil
}
