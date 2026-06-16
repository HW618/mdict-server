package config

import (
	"os"
	"testing"
	"time"
)

func TestGetServerAddress(t *testing.T) {
	cfg := &Config{
		ServerAddr: "127.0.0.1",
		ServerPort: 9090,
	}
	if cfg.GetServerAddress() != "127.0.0.1:9090" {
		t.Errorf("expected '127.0.0.1:9090', got '%s'", cfg.GetServerAddress())
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := &Config{
		ServerPort:    8080,
		JWTSecret:     "this-is-a-32-char-secret-key!!!!!!",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		RateLimit:     100,
		LogLevel:      "info",
		LogFormat:     "json",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := &Config{
		ServerPort:    0,
		JWTSecret:     "this-is-a-32-char-secret-key!!!!",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		LogLevel:      "info",
		LogFormat:     "json",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid port")
	}

	cfg.ServerPort = 70000
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port > 65535")
	}
}

func TestValidateShortJWTSecret(t *testing.T) {
	cfg := &Config{
		ServerPort:    8080,
		JWTSecret:     "short",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		LogLevel:      "info",
		LogFormat:     "json",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for short JWT secret")
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	cfg := &Config{
		ServerPort:    8080,
		JWTSecret:     "this-is-a-32-char-secret-key!!!!",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		LogLevel:      "verbose",
		LogFormat:     "json",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid log level")
	}
}

func TestValidateInvalidLogFormat(t *testing.T) {
	cfg := &Config{
		ServerPort:    8080,
		JWTSecret:     "this-is-a-32-char-secret-key!!!!",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		LogLevel:      "info",
		LogFormat:     "xml",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid log format")
	}
}

func TestValidateNegativeRateLimit(t *testing.T) {
	cfg := &Config{
		ServerPort:    8080,
		JWTSecret:     "this-is-a-32-char-secret-key!!!!",
		JWTAccessTTL:  2 * time.Hour,
		JWTRefreshTTL: 168 * time.Hour,
		RateLimit:     -1,
		LogLevel:      "info",
		LogFormat:     "json",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative rate limit")
	}
}

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"500MB", 500 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1024KB", 1024 * 1024, false},
		{"1024", 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"100TB", 0, true},
	}

	for _, tt := range tests {
		result, err := parseSizeString(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("expected error for input '%s'", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input '%s': %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("expected %d for '%s', got %d", tt.expected, tt.input, result)
			}
		}
	}
}

func TestGenerateRandomUsername(t *testing.T) {
	username := generateRandomUsername()

	if len(username) != 10 {
		t.Errorf("expected username length 10, got %d: '%s'", len(username), username)
	}
	if username[:6] != "admin_" {
		t.Errorf("expected username to start with 'admin_', got '%s'", username)
	}

	// Two usernames should be different
	u2 := generateRandomUsername()
	if username == u2 {
		t.Error("expected two generated usernames to be different")
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	password := generateRandomPassword(16)

	if len(password) != 16 {
		t.Errorf("expected password length 16, got %d", len(password))
	}

	// Check character diversity
	hasLower, hasUpper, hasDigit, hasSpecial := false, false, false, false
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if !hasLower {
		t.Error("expected password to contain lowercase letters")
	}
	if !hasUpper {
		t.Error("expected password to contain uppercase letters")
	}
	if !hasDigit {
		t.Error("expected password to contain digits")
	}
	if !hasSpecial {
		t.Error("expected password to contain special characters")
	}
}

func TestGenerateRandomPasswordMinLength(t *testing.T) {
	password := generateRandomPassword(2)
	if len(password) != 4 {
		t.Errorf("expected minimum length 4, got %d", len(password))
	}
}

func TestGenerateBase64Secret(t *testing.T) {
	secret := generateBase64Secret(32)
	if len(secret) < 32 {
		t.Errorf("expected secret length >= 32, got %d", len(secret))
	}

	// Two secrets should be different
	secret2 := generateBase64Secret(32)
	if secret == secret2 {
		t.Error("expected two generated secrets to be different")
	}
}

func TestGetEnv(t *testing.T) {
	// Test default value
	val := getEnv("NONEXISTENT_VAR_XYZ", "default")
	if val != "default" {
		t.Errorf("expected 'default', got '%s'", val)
	}

	// Test with set env var
	_ = os.Setenv("TEST_GETENV_VAR", "custom")
	defer func() { _ = os.Unsetenv("TEST_GETENV_VAR") }()
	val = getEnv("TEST_GETENV_VAR", "default")
	if val != "custom" {
		t.Errorf("expected 'custom', got '%s'", val)
	}
}

func TestGetEnvInt(t *testing.T) {
	// Test default value
	val := getEnvInt("NONEXISTENT_INT_XYZ", 42)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}

	// Test with set env var
	_ = os.Setenv("TEST_GETENVINT_VAR", "123")
	defer func() { _ = os.Unsetenv("TEST_GETENVINT_VAR") }()
	val = getEnvInt("TEST_GETENVINT_VAR", 42)
	if val != 123 {
		t.Errorf("expected 123, got %d", val)
	}

	// Test invalid int
	_ = os.Setenv("TEST_GETENVINT_VAR", "notanint")
	val = getEnvInt("TEST_GETENVINT_VAR", 42)
	if val != 42 {
		t.Errorf("expected default 42 for invalid int, got %d", val)
	}
}

func TestGetEnvDuration(t *testing.T) {
	val := getEnvDuration("NONEXISTENT_DUR_XYZ", 2*time.Hour)
	if val != 2*time.Hour {
		t.Errorf("expected 2h, got %v", val)
	}

	_ = os.Setenv("TEST_GETENVDUR_VAR", "30m")
	defer func() { _ = os.Unsetenv("TEST_GETENVDUR_VAR") }()
	val = getEnvDuration("TEST_GETENVDUR_VAR", 2*time.Hour)
	if val != 30*time.Minute {
		t.Errorf("expected 30m, got %v", val)
	}
}

func TestGetEnvSlice(t *testing.T) {
	val := getEnvSlice("NONEXISTENT_SLICE_XYZ", []string{"a", "b"})
	if len(val) != 2 || val[0] != "a" {
		t.Errorf("expected [a b], got %v", val)
	}

	_ = os.Setenv("TEST_GETENVSLICE_VAR", "x,y,z")
	defer func() { _ = os.Unsetenv("TEST_GETENVSLICE_VAR") }()
	val = getEnvSlice("TEST_GETENVSLICE_VAR", []string{"a"})
	if len(val) != 3 || val[0] != "x" || val[1] != "y" || val[2] != "z" {
		t.Errorf("expected [x y z], got %v", val)
	}
}
