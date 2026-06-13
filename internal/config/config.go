package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`

	// Directory paths
	DictDir string `json:"dict_dir"`
	DataDir string `json:"data_dir"`

	// Admin credentials
	AdminUser string `json:"admin_user"`
	AdminPass string `json:"admin_pass"`

	// JWT configuration
	JWTSecret     string        `json:"jwt_secret"`
	JWTAccessTTL  time.Duration `json:"jwt_access_ttl"`
	JWTRefreshTTL time.Duration `json:"jwt_refresh_ttl"`

	// Agent Skill configuration
	SkillServerURL string `json:"skill_server_url"`

	// Upload limits
	MaxUploadSize      string `json:"max_upload_size"`
	MaxUploadSizeBytes int64  `json:"-"`

	// Rate limiting
	RateLimit int `json:"rate_limit"`

	// Logging
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// CORS
	CORSOrigins []string `json:"cors_origins"`
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	// Load .env file if it exists (non-fatal)
	loadDotEnv(".env")

	cfg := &Config{
		ServerAddr:    getEnv("SERVER_ADDR", "0.0.0.0"),
		ServerPort:    getEnvInt("SERVER_PORT", 8080),
		DictDir:       getEnv("DICT_DIR", "./dicts"),
		DataDir:       getEnv("DATA_DIR", "./data"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTAccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 2*time.Hour),
		JWTRefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 168*time.Hour),
		SkillServerURL: getEnv("SKILL_SERVER_URL", "http://localhost:8080"),
		MaxUploadSize: getEnv("MAX_UPLOAD_SIZE", "500MB"),
		RateLimit:     getEnvInt("RATE_LIMIT", 100),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		LogFormat:     getEnv("LOG_FORMAT", "json"),
		CORSOrigins:   getEnvSlice("CORS_ORIGINS", []string{"*"}),
	}

	// Generate admin credentials if not provided
	cfg.AdminUser = getEnv("ADMIN_USER", "")
	cfg.AdminPass = getEnv("ADMIN_PASS", "")

	if cfg.AdminUser == "" {
		cfg.AdminUser = generateRandomUsername()
		log.Info().Str("username", cfg.AdminUser).Msg("Generated admin username")
	}

	if cfg.AdminPass == "" {
		cfg.AdminPass = generateRandomPassword(16)
		log.Info().Str("password", cfg.AdminPass).Msg("Generated admin password")
	}

	// Generate JWT secret if not provided
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = generateBase64Secret(32)
		log.Info().Msg("Generated JWT secret key")
	}

	// Parse max upload size
	maxUploadBytes, err := parseSizeString(cfg.MaxUploadSize)
	if err != nil {
		log.Warn().Str("value", cfg.MaxUploadSize).Msg("Invalid MAX_UPLOAD_SIZE, using default 500MB")
		maxUploadBytes = 500 * 1024 * 1024
	}
	cfg.MaxUploadSizeBytes = maxUploadBytes

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure directories exist
	if err := ensureDir(cfg.DictDir); err != nil {
		return nil, fmt.Errorf("failed to create dict directory: %w", err)
	}

	if err := ensureDir(cfg.DataDir); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}

	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}

	if c.JWTAccessTTL <= 0 {
		return fmt.Errorf("JWT access TTL must be positive")
	}

	if c.JWTRefreshTTL <= 0 {
		return fmt.Errorf("JWT refresh TTL must be positive")
	}

	if c.RateLimit < 0 {
		return fmt.Errorf("rate limit cannot be negative")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("invalid log format: %s", c.LogFormat)
	}

	return nil
}

// GetServerAddress returns the full server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.ServerAddr, c.ServerPort)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid integer value, using default")
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid duration value, using default")
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		if value == "" {
			return []string{}
		}
		return strings.Split(value, ",")
	}
	return defaultValue
}

// generateRandomUsername generates a random admin username: "admin_" + 4 lowercase alphanumeric chars.
func generateRandomUsername() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 4)
	for i := range suffix {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate random username")
		}
		suffix[i] = chars[n.Int64()]
	}
	return "admin_" + string(suffix)
}

// generateRandomPassword generates a random password with guaranteed character diversity.
// Ensures at least one uppercase, one lowercase, one digit, and one special character.
func generateRandomPassword(length int) string {
	if length < 4 {
		length = 4
	}

	const (
		lower   = "abcdefghijklmnopqrstuvwxyz"
		upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits  = "0123456789"
		special = "!@#$%^&*"
	)
	all := lower + upper + digits + special

	// Pick one from each required category
	categories := []string{lower, upper, digits, special}
	result := make([]byte, length)
	for i, cat := range categories {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cat))))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate random password")
		}
		result[i] = cat[n.Int64()]
	}

	// Fill the rest from the full charset
	for i := len(categories); i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate random password")
		}
		result[i] = all[n.Int64()]
	}

	// Shuffle to avoid predictable positions
	for i := len(result) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to shuffle password")
		}
		j := int(n.Int64())
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// generateBase64Secret generates a random byte sequence and returns it as Base64.
func generateBase64Secret(byteLength int) string {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal().Err(err).Msg("Failed to generate random secret")
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// loadDotEnv loads environment variables from a .env file.
// Lines starting with # are comments. Empty lines are skipped.
// Only sets variables that are not already set in the environment.
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // .env file doesn't exist or can't be read — that's fine
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// Only set if not already in environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// parseSizeString parses a human-readable size string like "500MB", "1GB", "1024KB" into bytes.
func parseSizeString(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	multipliers := map[string]int64{
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(strings.ToUpper(s), suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(suffix)])
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid size number: %s", numStr)
			}
			return int64(num * float64(mult)), nil
		}
	}

	// Plain number = bytes
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size string: %s", s)
	}
	return num, nil
}
