package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/HW618/mdict-server/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Claims represents JWT claims
type Claims struct {
	PasswordVersion string                 `json:"pv"`
	UserID          string                 `json:"sub"`
	Username        string                 `json:"username"`
	Permissions     models.UserPermissions `json:"permissions"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a new access token
func (m *JWTManager) GenerateAccessToken(user *models.User) (string, error) {
	claims := Claims{
		PasswordVersion: m.passwordVersion(user),
		UserID:          user.ID,
		Username:        user.Username,
		Permissions:     user.Permissions(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "mdict-server",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token
func (m *JWTManager) GenerateRefreshToken(user *models.User) (string, error) {
	claims := Claims{
		PasswordVersion: m.passwordVersion(user),
		UserID:          user.ID,
		Username:        user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "mdict-server",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GetAccessTTL returns the access token TTL
func (m *JWTManager) GetAccessTTL() time.Duration {
	return m.accessTTL
}

// GetRefreshTTL returns the refresh token TTL
func (m *JWTManager) GetRefreshTTL() time.Duration {
	return m.refreshTTL
}

// MatchesUser rejects login tokens issued before the user's password changed.
func (m *JWTManager) MatchesUser(claims *Claims, user *models.User) bool {
	return hmac.Equal([]byte(claims.PasswordVersion), []byte(m.passwordVersion(user)))
}
func (m *JWTManager) passwordVersion(user *models.User) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(user.Password))
	return hex.EncodeToString(mac.Sum(nil))
}
