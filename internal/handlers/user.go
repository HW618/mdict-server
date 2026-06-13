package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/auth"
	"github.com/HW618/mdict-server/internal/models"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles user endpoints
type UserHandler struct {
	userStore  *store.UserStore
	jwtManager *auth.JWTManager
}

// NewUserHandler creates a new user handler
func NewUserHandler(userStore *store.UserStore, jwtManager *auth.JWTManager) *UserHandler {
	return &UserHandler{
		userStore:  userStore,
		jwtManager: jwtManager,
	}
}

// List returns all users
func (h *UserHandler) List(c *gin.Context) {
	users, err := h.userStore.List()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list users")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to list users",
			"data":    nil,
		})
		return
	}

	// Convert to response
	responses := make([]models.UserResponse, len(users))
	for i, u := range users {
		responses[i] = u.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    responses,
	})
}

// Create creates a new user
func (h *UserHandler) Create(c *gin.Context) {
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Check if username already exists
	exists, err := h.userStore.ExistsByUsername(req.Username)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check username existence")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to check username",
			"data":    nil,
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"code":    40901,
			"message": "Username already exists",
			"data":    nil,
		})
		return
	}

	// Generate password if not provided
	password := req.Password
	if password == "" {
		var err error
		password, err = generateRandomPassword(16)
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate random password")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    50001,
				"message": "Failed to generate password",
				"data":    nil,
			})
			return
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to create user",
			"data":    nil,
		})
		return
	}

	// Create user
	user := models.NewUser(req.Username, string(hashedPassword))
	user.CanUseAPI = req.Permissions.CanUseAPI
	user.IsDictAdmin = req.Permissions.IsDictAdmin
	user.IsUserAdmin = req.Permissions.IsUserAdmin

	if err := h.userStore.Create(user); err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to create user",
			"data":    nil,
		})
		return
	}

	// Return response with password (only on creation)
	response := user.ToResponse()
	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "User created successfully",
		"data": gin.H{
			"id":          response.ID,
			"username":    response.Username,
			"api_token":   response.APIToken,
			"password":    password, // Only returned on creation
			"permissions": response.Permissions,
		},
	})
}

// Delete deletes a user
func (h *UserHandler) Delete(c *gin.Context) {
	userID := c.Param("id")

	// Get user to check if it's an admin
	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Prevent deleting admin users
	if user.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    40301,
			"message": "Cannot delete admin user",
			"data":    nil,
		})
		return
	}

	// Delete user's refresh tokens
	if err := h.userStore.DeleteUserRefreshTokens(userID); err != nil {
		log.Error().Err(err).Msg("Failed to delete user refresh tokens")
	}

	// Delete user
	if err := h.userStore.Delete(userID); err != nil {
		log.Error().Err(err).Msg("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to delete user",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "User deleted",
	})
}

// UpdatePermissions updates user permissions
func (h *UserHandler) UpdatePermissions(c *gin.Context) {
	userID := c.Param("id")

	var req models.UserUpdatePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Get user to check current permissions
	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Prevent removing last admin
	if user.IsAdmin() && !req.IsUserAdmin {
		// Count admins
		users, err := h.userStore.List()
		if err != nil {
			log.Error().Err(err).Msg("Failed to list users")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    50002,
				"message": "Failed to check admin count",
				"data":    nil,
			})
			return
		}

		adminCount := 0
		for _, u := range users {
			if u.IsAdmin() {
				adminCount++
			}
		}

		if adminCount <= 1 {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "Cannot remove last admin",
				"data":    nil,
			})
			return
		}
	}

	// Update permissions
	perms := &models.UserPermissions{
		CanUseAPI:   req.CanUseAPI,
		IsDictAdmin: req.IsDictAdmin,
		IsUserAdmin: req.IsUserAdmin,
	}

	if err := h.userStore.UpdatePermissions(userID, perms); err != nil {
		log.Error().Err(err).Msg("Failed to update permissions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to update permissions",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Permissions updated",
	})
}

// ResetToken resets user's API token
func (h *UserHandler) ResetToken(c *gin.Context) {
	userID := c.Param("id")

	// Check if user exists
	_, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Generate new token
	newToken := models.GenerateAPIToken()

	// Update token
	if err := h.userStore.UpdateAPIToken(userID, newToken); err != nil {
		log.Error().Err(err).Msg("Failed to reset API token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to reset API token",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "API token reset successfully",
		"data": gin.H{
			"api_token": newToken,
		},
	})
}

// generateRandomPassword generates a cryptographically secure random password.
// Returns an error if the crypto/rand source fails.
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random password: %w", err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}
