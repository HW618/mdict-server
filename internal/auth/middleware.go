package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
)

// AuthMiddleware handles authentication
type AuthMiddleware struct {
	jwtManager *JWTManager
	userStore  *store.UserStore
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *JWTManager, userStore *store.UserStore) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		userStore:  userStore,
	}
}

// RequireAuth middleware requires authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := m.authenticate(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "Authentication required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

// RequireAPIAccess middleware requires API access permission
func (m *AuthMiddleware) RequireAPIAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := m.authenticate(c)
		if err != nil {
			// Try to authenticate with API token
			user, err = m.authenticateWithAPIToken(c)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    40101,
					"message": "Authentication required",
					"data":    nil,
				})
				c.Abort()
				return
			}
		}

		// Check if user has API access
		if !user.CanUseAPI && !user.IsAdmin() {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "API access not allowed",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

// authenticateAny tries JWT first, then falls back to API token.
func (m *AuthMiddleware) authenticateAny(c *gin.Context) (*models.User, error) {
	user, err := m.authenticate(c)
	if err == nil {
		return user, nil
	}
	return m.authenticateWithAPIToken(c)
}

// RequireDictAdmin middleware requires dictionary admin permission
func (m *AuthMiddleware) RequireDictAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := m.authenticateAny(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "Authentication required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Check if user is dict admin
		if !user.CanManageDicts() {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "Dictionary admin access required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

// RequireUserAdmin middleware requires user admin permission
func (m *AuthMiddleware) RequireUserAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := m.authenticateAny(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "Authentication required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Check if user is user admin
		if !user.CanManageUsers() {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "User admin access required",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

// authenticate authenticates the request using JWT token
func (m *AuthMiddleware) authenticate(c *gin.Context) (*models.User, error) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	tokenString := parts[1]

	// Validate token
	claims, err := m.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get user from database
	user, err := m.userStore.GetByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	return user, nil
}

// authenticateWithAPIToken authenticates using API token
func (m *AuthMiddleware) authenticateWithAPIToken(c *gin.Context) (*models.User, error) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	// Check for Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := parts[1]

	// Check if it's an API token (starts with mdtk_)
	if !strings.HasPrefix(token, "mdtk_") {
		return nil, fmt.Errorf("not an API token")
	}

	// Get user by API token
	user, err := m.userStore.GetByAPIToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid API token: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	return user, nil
}

// GetUserFromContext gets user from gin context
func GetUserFromContext(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}

	u, ok := user.(*models.User)
	return u, ok
}

// GetUserIDFromContext gets user ID from gin context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return "", false
	}

	id, ok := userID.(string)
	return id, ok
}
