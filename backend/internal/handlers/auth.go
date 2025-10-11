package handlers

import (
	"net/http"
	"strings"
	"task-manager/backend/internal/services"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db          *gorm.DB
	authService services.AuthService
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	AccessToken  string               `json:"access_token"`
	RefreshToken string               `json:"refresh_token"`
	TokenType    string               `json:"token_type"`
	ExpiresIn    int64                `json:"expires_in"`
	User         *UserProfileResponse `json:"user"`
	Permissions  []string             `json:"permissions"`
}

type UserProfileResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Department  string     `json:"department"`
	Position    string     `json:"position"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	Roles       []string   `json:"roles"`
}

func NewAuthHandler(db *gorm.DB, authService services.AuthService) *AuthHandler {
	return &AuthHandler{db: db, authService: authService}
}

func (h *AuthHandler) Token(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := h.authService.LoginUser(h.db, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_credentials",
			"message": "Invalid email or password",
		})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "account_disabled",
			"message": "Your account has been disabled. Please contact support.",
		})
		return
	}

	accessToken, refreshToken, err := h.authService.GenerateToken(h.db, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_generation_failed",
			"message": "Failed to generate authentication tokens",
		})
		return
	}

	now := time.Now()
	user.LastLoginAt = &now
	if err := h.db.Save(user).Error; err != nil {
	}

	permissions, err := h.authService.GetUserPermissions(h.db, user.ID)
	if err != nil {
		permissions = []string{} 
	}

	userProfile := &UserProfileResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Department:  user.Department,
		Position:    user.Position,
		IsActive:    user.IsActive,
		LastLoginAt: user.LastLoginAt,
		Roles:       user.GetRoleNames(),
	}

	response := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, 
		User:         userProfile,
		Permissions:  permissions,
	}

	c.JSON(http.StatusOK, response)
}
