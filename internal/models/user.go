package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // bcrypt hash, not exposed in JSON
	APIToken  string    `json:"api_token,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Permissions
	CanUseAPI   bool `json:"can_use_api"`
	IsDictAdmin bool `json:"is_dict_admin"`
	IsUserAdmin bool `json:"is_user_admin"`
}

// NewUser creates a new user with default values
func NewUser(username, hashedPassword string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New().String(),
		Username:  username,
		Password:  hashedPassword,
		APIToken:  GenerateAPIToken(),
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GenerateAPIToken generates a new API token
func GenerateAPIToken() string {
	return "mdtk_" + uuid.New().String()
}

// Permissions returns user permissions as a struct
func (u *User) Permissions() UserPermissions {
	return UserPermissions{
		CanUseAPI:   u.CanUseAPI,
		IsDictAdmin: u.IsDictAdmin,
		IsUserAdmin: u.IsUserAdmin,
	}
}

// UserPermissions represents user permission flags
type UserPermissions struct {
	CanUseAPI   bool `json:"can_use_api"`
	IsDictAdmin bool `json:"is_dict_admin"`
	IsUserAdmin bool `json:"is_user_admin"`
}

// IsAdmin checks if user has admin privileges
func (u *User) IsAdmin() bool {
	return u.IsUserAdmin
}

// CanManageDicts checks if user can manage dictionaries
func (u *User) CanManageDicts() bool {
	return u.IsDictAdmin || u.IsUserAdmin
}

// CanManageUsers checks if user can manage users
func (u *User) CanManageUsers() bool {
	return u.IsUserAdmin
}

// UserCreateRequest represents the request to create a new user
type UserCreateRequest struct {
	Username    string           `json:"username" binding:"required"`
	Password    string           `json:"password"`
	Permissions UserPermissions  `json:"permissions"`
}

// UserUpdatePermissionsRequest represents the request to update user permissions
type UserUpdatePermissionsRequest struct {
	CanUseAPI   bool `json:"can_use_api"`
	IsDictAdmin bool `json:"is_dict_admin"`
	IsUserAdmin bool `json:"is_user_admin"`
}

// UserResponse represents the user data in API responses
type UserResponse struct {
	ID          string          `json:"id"`
	Username    string          `json:"username"`
	APIToken    string          `json:"api_token,omitempty"`
	IsActive    bool            `json:"is_active"`
	Permissions UserPermissions `json:"permissions"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		APIToken:  u.APIToken,
		IsActive:  u.IsActive,
		Permissions: u.Permissions(),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
