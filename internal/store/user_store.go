package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/HW618/mdict-server/internal/models"
)

// UserStore handles user database operations
type UserStore struct {
	store *SQLiteStore
}

// NewUserStore creates a new user store
func NewUserStore(store *SQLiteStore) *UserStore {
	return &UserStore{store: store}
}

// Create creates a new user
func (s *UserStore) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, username, password, api_token, can_use_api, is_dict_admin, is_user_admin, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.store.db.Exec(query,
		user.ID,
		user.Username,
		user.Password,
		user.APIToken,
		user.CanUseAPI,
		user.IsDictAdmin,
		user.IsUserAdmin,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (s *UserStore) GetByID(id string) (*models.User, error) {
	query := `
		SELECT id, username, password, api_token, can_use_api, is_dict_admin, is_user_admin, is_active, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	user := &models.User{}
	err := s.store.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.APIToken,
		&user.CanUseAPI,
		&user.IsDictAdmin,
		&user.IsUserAdmin,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByUsername retrieves a user by username
func (s *UserStore) GetByUsername(username string) (*models.User, error) {
	query := `
		SELECT id, username, password, api_token, can_use_api, is_dict_admin, is_user_admin, is_active, created_at, updated_at
		FROM users
		WHERE username = ?
	`

	user := &models.User{}
	err := s.store.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.APIToken,
		&user.CanUseAPI,
		&user.IsDictAdmin,
		&user.IsUserAdmin,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByAPIToken retrieves a user by API token
func (s *UserStore) GetByAPIToken(token string) (*models.User, error) {
	query := `
		SELECT id, username, password, api_token, can_use_api, is_dict_admin, is_user_admin, is_active, created_at, updated_at
		FROM users
		WHERE api_token = ? AND is_active = TRUE
	`

	user := &models.User{}
	err := s.store.db.QueryRow(query, token).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.APIToken,
		&user.CanUseAPI,
		&user.IsDictAdmin,
		&user.IsUserAdmin,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// List retrieves all users
func (s *UserStore) List() ([]*models.User, error) {
	query := `
		SELECT id, username, password, api_token, can_use_api, is_dict_admin, is_user_admin, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := s.store.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Password,
			&user.APIToken,
			&user.CanUseAPI,
			&user.IsDictAdmin,
			&user.IsUserAdmin,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// Update updates a user
func (s *UserStore) Update(user *models.User) error {
	query := `
		UPDATE users
		SET username = ?, password = ?, api_token = ?, can_use_api = ?, is_dict_admin = ?, is_user_admin = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`

	user.UpdatedAt = time.Now()

	_, err := s.store.db.Exec(query,
		user.Username,
		user.Password,
		user.APIToken,
		user.CanUseAPI,
		user.IsDictAdmin,
		user.IsUserAdmin,
		user.IsActive,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePermissions updates user permissions
func (s *UserStore) UpdatePermissions(id string, perms *models.UserPermissions) error {
	query := `
		UPDATE users
		SET can_use_api = ?, is_dict_admin = ?, is_user_admin = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.store.db.Exec(query,
		perms.CanUseAPI,
		perms.IsDictAdmin,
		perms.IsUserAdmin,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update permissions: %w", err)
	}

	return nil
}

// UpdateAPIToken updates user's API token
func (s *UserStore) UpdateAPIToken(id, token string) error {
	query := `
		UPDATE users
		SET api_token = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.store.db.Exec(query, token, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update API token: %w", err)
	}

	return nil
}

// Delete deletes a user
func (s *UserStore) Delete(id string) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.store.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Count returns the total number of users
func (s *UserStore) Count() (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	err := s.store.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// ExistsByUsername checks if a user exists with the given username
func (s *UserStore) ExistsByUsername(username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`

	var exists bool
	err := s.store.db.QueryRow(query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return exists, nil
}

// SaveRefreshToken saves a refresh token
func (s *UserStore) SaveRefreshToken(userID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := s.store.db.Exec(query,
		fmt.Sprintf("%s_%d", userID, time.Now().UnixNano()),
		userID,
		token,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token
func (s *UserStore) GetRefreshToken(token string) (string, error) {
	query := `
		SELECT user_id
		FROM refresh_tokens
		WHERE token = ? AND expires_at > ?
	`

	var userID string
	err := s.store.db.QueryRow(query, token, time.Now()).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("refresh token not found or expired")
	}

	if err != nil {
		return "", fmt.Errorf("failed to get refresh token: %w", err)
	}

	return userID, nil
}

// DeleteRefreshToken deletes a refresh token
func (s *UserStore) DeleteRefreshToken(token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = ?`

	_, err := s.store.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

// DeleteUserRefreshTokens deletes all refresh tokens for a user
func (s *UserStore) DeleteUserRefreshTokens(userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = ?`

	_, err := s.store.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user refresh tokens: %w", err)
	}

	return nil
}

// CleanExpiredTokens removes expired refresh tokens
func (s *UserStore) CleanExpiredTokens() error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < ?`

	_, err := s.store.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to clean expired tokens: %w", err)
	}

	return nil
}
