package handlers

import (
	"net/http"
	"task-manager/backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LogoutHandler struct {
	db          *gorm.DB
	authService services.AuthService
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewLogoutHandler(db *gorm.DB, authService services.AuthService) *LogoutHandler {
	return &LogoutHandler{db: db, authService: authService}
}

func (h *LogoutHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err := h.authService.RevokeToken(h.db, req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Successfully logged out",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}
