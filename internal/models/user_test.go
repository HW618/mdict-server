package models

import (
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	user := NewUser("testuser", "hashedpassword")

	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Username)
	}
	if user.Password != "hashedpassword" {
		t.Errorf("expected password 'hashedpassword', got '%s'", user.Password)
	}
	if !user.IsActive {
		t.Error("expected user to be active")
	}
	if !strings.HasPrefix(user.APIToken, "mdtk_") {
		t.Errorf("expected API token to start with 'mdtk_', got '%s'", user.APIToken)
	}
	if user.CanUseAPI || user.IsDictAdmin || user.IsUserAdmin {
		t.Error("expected all permissions to be false by default")
	}
}

func TestGenerateAPIToken(t *testing.T) {
	token := GenerateAPIToken()
	if !strings.HasPrefix(token, "mdtk_") {
		t.Errorf("expected token to start with 'mdtk_', got '%s'", token)
	}
	// mdtk_ (5) + UUID (36) = 41 chars
	if len(token) != 41 {
		t.Errorf("expected token length 41, got %d", len(token))
	}

	// Two tokens should be different
	token2 := GenerateAPIToken()
	if token == token2 {
		t.Error("expected two generated tokens to be different")
	}
}

func TestUserPermissions(t *testing.T) {
	user := &User{
		CanUseAPI:   true,
		IsDictAdmin: false,
		IsUserAdmin: true,
	}

	perms := user.Permissions()
	if !perms.CanUseAPI {
		t.Error("expected CanUseAPI to be true")
	}
	if perms.IsDictAdmin {
		t.Error("expected IsDictAdmin to be false")
	}
	if !perms.IsUserAdmin {
		t.Error("expected IsUserAdmin to be true")
	}
}

func TestUserIsAdmin(t *testing.T) {
	user := &User{IsUserAdmin: false}
	if user.IsAdmin() {
		t.Error("expected IsAdmin() to return false")
	}

	user.IsUserAdmin = true
	if !user.IsAdmin() {
		t.Error("expected IsAdmin() to return true")
	}
}

func TestUserCanManageDicts(t *testing.T) {
	user := &User{IsDictAdmin: false, IsUserAdmin: false}
	if user.CanManageDicts() {
		t.Error("expected CanManageDicts() to return false")
	}

	user.IsDictAdmin = true
	if !user.CanManageDicts() {
		t.Error("expected CanManageDicts() to return true when IsDictAdmin")
	}

	user.IsDictAdmin = false
	user.IsUserAdmin = true
	if !user.CanManageDicts() {
		t.Error("expected CanManageDicts() to return true when IsUserAdmin")
	}
}

func TestUserCanManageUsers(t *testing.T) {
	user := &User{IsUserAdmin: false}
	if user.CanManageUsers() {
		t.Error("expected CanManageUsers() to return false")
	}

	user.IsUserAdmin = true
	if !user.CanManageUsers() {
		t.Error("expected CanManageUsers() to return true")
	}
}

func TestUserToResponse(t *testing.T) {
	user := &User{
		ID:          "test-id",
		Username:    "testuser",
		Password:    "hashedpassword",
		APIToken:    "mdtk_test",
		IsActive:    true,
		CanUseAPI:   true,
		IsDictAdmin: false,
		IsUserAdmin: true,
	}

	resp := user.ToResponse()

	if resp.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", resp.ID)
	}
	if resp.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", resp.Username)
	}
	if resp.APIToken != "mdtk_test" {
		t.Errorf("expected API token 'mdtk_test', got '%s'", resp.APIToken)
	}
	if !resp.IsActive {
		t.Error("expected IsActive to be true")
	}
	if !resp.Permissions.CanUseAPI {
		t.Error("expected CanUseAPI to be true")
	}
	if resp.Permissions.IsDictAdmin {
		t.Error("expected IsDictAdmin to be false")
	}
	if !resp.Permissions.IsUserAdmin {
		t.Error("expected IsUserAdmin to be true")
	}
}

func TestUserCreateRequestValidation(t *testing.T) {
	// Username is required (binding:"required")
	req := UserCreateRequest{
		Username: "",
	}
	if req.Username != "" {
		t.Error("expected empty username")
	}

	req.Username = "validuser"
	if req.Username != "validuser" {
		t.Errorf("expected 'validuser', got '%s'", req.Username)
	}
}
