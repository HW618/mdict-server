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

// GetCurrentUser returns the current authenticated user's info
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []models.UserResponse{user.ToResponse()},
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

	log.Info().
		Str("audit", "true").
		Str("action", "user_created").
		Str("target_user_id", user.ID).
		Str("target_username", user.Username).
		Str("operator_id", c.GetString("userID")).
		Msg("User created")

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

	log.Info().
		Str("audit", "true").
		Str("action", "user_deleted").
		Str("target_user_id", userID).
		Str("target_username", user.Username).
		Str("operator_id", c.GetString("userID")).
		Msg("User deleted")

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

	log.Info().
		Str("audit", "true").
		Str("action", "permissions_updated").
		Str("target_user_id", userID).
		Bool("can_use_api", req.CanUseAPI).
		Bool("is_dict_admin", req.IsDictAdmin).
		Bool("is_user_admin", req.IsUserAdmin).
		Str("operator_id", c.GetString("userID")).
		Msg("User permissions updated")

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

	log.Info().
		Str("audit", "true").
		Str("action", "api_token_reset").
		Str("target_user_id", userID).
		Str("operator_id", c.GetString("userID")).
		Msg("API token reset")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "API token reset successfully",
		"data": gin.H{
			"api_token": newToken,
		},
	})
}

// ChangePassword handles a user changing their own password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	userID := c.GetString("userID")

	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    40301,
			"message": "Old password is incorrect",
			"data":    nil,
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to update password",
			"data":    nil,
		})
		return
	}

	if err := h.userStore.UpdatePassword(userID, string(hashedPassword)); err != nil {
		log.Error().Err(err).Msg("Failed to update password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to update password",
			"data":    nil,
		})
		return
	}

	log.Info().
		Str("audit", "true").
		Str("action", "password_changed").
		Str("user_id", userID).
		Msg("User changed their password")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Password changed successfully",
	})
}

// AdminResetPassword handles an admin resetting a user's password
func (h *UserHandler) AdminResetPassword(c *gin.Context) {
	operatorID := c.GetString("userID")

	// Validate operator's current password
	operator, err := h.userStore.GetByID(operatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to verify operator",
			"data":    nil,
		})
		return
	}

	userID := c.Param("id")

	var req models.AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Verify operator's old password
	if err := bcrypt.CompareHashAndPassword([]byte(operator.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    40301,
			"message": "Current password is incorrect",
			"data":    nil,
		})
		return
	}

	// Check if target user exists
	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to reset password",
			"data":    nil,
		})
		return
	}

	if err := h.userStore.UpdatePassword(userID, string(hashedPassword)); err != nil {
		log.Error().Err(err).Msg("Failed to reset password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50002,
			"message": "Failed to reset password",
			"data":    nil,
		})
		return
	}

	log.Info().
		Str("audit", "true").
		Str("action", "admin_password_reset").
		Str("target_user_id", userID).
		Str("target_username", user.Username).
		Str("operator_id", c.GetString("userID")).
		Msg("Admin reset user password")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Password reset successfully",
	})
}

// generateRandomPassword generates a cryptographically secure random password
// with guaranteed character diversity (at least one lowercase, uppercase, digit, special).
func generateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
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
	buf := make([]byte, 1)
	for i, cat := range categories {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		result[i] = cat[int(buf[0])%len(cat)]
	}

	// Fill the rest from the full charset
	for i := len(categories); i < length; i++ {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		result[i] = all[int(buf[0])%len(all)]
	}

	// Shuffle to avoid predictable positions
	for i := len(result) - 1; i > 0; i-- {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		j := int(buf[0]) % (i + 1)
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}
