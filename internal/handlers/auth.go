package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/HW618/mdict-server/internal/auth"
	"github.com/HW618/mdict-server/internal/store"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userStore  *store.UserStore
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userStore *store.UserStore, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		userStore:  userStore,
		jwtManager: jwtManager,
	}
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Get user by username
	user, err := h.userStore.GetByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40104,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40104,
			"message": "Account is disabled",
			"data":    nil,
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40104,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Generate access token
	accessToken, err := h.jwtManager.GenerateAccessToken(user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to generate token",
			"data":    nil,
		})
		return
	}

	// Generate refresh token
	refreshToken, err := h.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to generate refresh token",
			"data":    nil,
		})
		return
	}

	// Save refresh token
	if err := h.userStore.SaveRefreshToken(user.ID, refreshToken, time.Now().Add(h.jwtManager.GetRefreshTTL())); err != nil {
		log.Error().Err(err).Msg("Failed to save refresh token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to complete login",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"expires_in":    int(h.jwtManager.GetAccessTTL().Seconds()),
			"user":          user.ToResponse(),
		},
	})
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid request body",
			"data":    nil,
		})
		return
	}

	// Validate refresh token
	claims, err := h.jwtManager.ValidateToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40102,
			"message": "Invalid or expired refresh token",
			"data":    nil,
		})
		return
	}

	// Check if refresh token exists in database
	userID, err := h.userStore.GetRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40103,
			"message": "Refresh token revoked",
			"data":    nil,
		})
		return
	}

	// Get user
	user, err := h.userStore.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40103,
			"message": "User not found",
			"data":    nil,
		})
		return
	}

	// Verify token belongs to user
	if claims.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40103,
			"message": "Token user mismatch",
			"data":    nil,
		})
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateAccessToken(user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate access token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to generate token",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"access_token": accessToken,
			"expires_in":   int(h.jwtManager.GetAccessTTL().Seconds()),
		},
	})
}

// LogoutRequest represents logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, that's okay - just log out
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "Logged out successfully",
		})
		return
	}

	// If refresh token provided, revoke it
	if req.RefreshToken != "" {
		if err := h.userStore.DeleteRefreshToken(req.RefreshToken); err != nil {
			log.Error().Err(err).Msg("Failed to delete refresh token")
		}
	} else {
		// Revoke all refresh tokens for user
		userID, exists := auth.GetUserIDFromContext(c)
		if exists {
			if err := h.userStore.DeleteUserRefreshTokens(userID); err != nil {
				log.Error().Err(err).Msg("Failed to delete user refresh tokens")
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Logged out successfully",
	})
}
