package auth

import (
	"testing"
	"time"

	"github.com/HW618/mdict-server/internal/models"
)

func newTestJWTManager() *JWTManager {
	return NewJWTManager("test-secret-key-32-chars-long-for-testing", 2*time.Hour, 168*time.Hour)
}

func newTestUser() *models.User {
	return &models.User{
		ID:          "user-001",
		Username:    "testuser",
		CanUseAPI:   true,
		IsDictAdmin: false,
		IsUserAdmin: false,
		IsActive:    true,
	}
}

func TestGenerateAccessToken(t *testing.T) {
	mgr := newTestJWTManager()
	user := newTestUser()

	token, err := mgr.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	mgr := newTestJWTManager()
	user := newTestUser()

	token, err := mgr.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

func TestValidateAccessToken(t *testing.T) {
	mgr := newTestJWTManager()
	user := newTestUser()

	token, err := mgr.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != "user-001" {
		t.Errorf("expected user ID 'user-001', got '%s'", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", claims.Username)
	}
	if !claims.Permissions.CanUseAPI {
		t.Error("expected CanUseAPI to be true")
	}
	if claims.Permissions.IsDictAdmin {
		t.Error("expected IsDictAdmin to be false")
	}
}

func TestValidateRefreshToken(t *testing.T) {
	mgr := newTestJWTManager()
	user := newTestUser()

	token, err := mgr.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if claims.UserID != "user-001" {
		t.Errorf("expected user ID 'user-001', got '%s'", claims.UserID)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	mgr := newTestJWTManager()

	_, err := mgr.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	mgr1 := NewJWTManager("secret-key-1-32-chars-long!!!!!", 2*time.Hour, 168*time.Hour)
	mgr2 := NewJWTManager("secret-key-2-32-chars-long!!!!!", 2*time.Hour, 168*time.Hour)

	user := newTestUser()
	token, err := mgr1.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = mgr2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}

func TestExpiredToken(t *testing.T) {
	// Create a manager with very short TTL
	mgr := NewJWTManager("test-secret-key-32-chars-long-for-testing", 1*time.Millisecond, 168*time.Hour)
	user := newTestUser()

	token, err := mgr.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = mgr.ValidateToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestGetAccessTTL(t *testing.T) {
	mgr := newTestJWTManager()
	if mgr.GetAccessTTL() != 2*time.Hour {
		t.Errorf("expected 2h, got %v", mgr.GetAccessTTL())
	}
}

func TestGetRefreshTTL(t *testing.T) {
	mgr := newTestJWTManager()
	if mgr.GetRefreshTTL() != 168*time.Hour {
		t.Errorf("expected 168h, got %v", mgr.GetRefreshTTL())
	}
}

func TestTokenPermissionsInClaims(t *testing.T) {
	mgr := newTestJWTManager()
	user := &models.User{
		ID:          "user-002",
		Username:    "admin",
		CanUseAPI:   true,
		IsDictAdmin: true,
		IsUserAdmin: true,
	}

	token, err := mgr.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if !claims.Permissions.CanUseAPI {
		t.Error("expected CanUseAPI to be true")
	}
	if !claims.Permissions.IsDictAdmin {
		t.Error("expected IsDictAdmin to be true")
	}
	if !claims.Permissions.IsUserAdmin {
		t.Error("expected IsUserAdmin to be true")
	}
}
