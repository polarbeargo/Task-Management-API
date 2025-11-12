package handlers

import (
	"context"
	"net/http"
	"task-manager/backend/internal/services"
	"task-manager/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	db           *gorm.DB
	userService  services.UserService
	authzService services.AuthorizationService
}

func NewUserHandler(db *gorm.DB, userService services.UserService, authzService services.AuthorizationService) *UserHandler {
	return &UserHandler{db: db, userService: userService, authzService: authzService}
}

func (h *UserHandler) GetUserProfile(c *gin.Context) {

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentUserUUID, ok := currentUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	authRequest := services.AuthorizationRequest{
		UserID:     currentUserUUID,
		Resource:   "profile",
		Action:     "read",
		ResourceID: &currentUserUUID,
	}

	decision, err := h.authzService.IsAuthorized(context.Background(), authRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
		return
	}

	if decision.Decision != "allowed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "reason": decision.Reason})
		return
	}

	user, err := h.userService.GetUserProfile(h.db, currentUserUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		return
	}

	response := gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetUserProfileByUserId(c *gin.Context) {

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentUserUUID, ok := currentUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userIDParam := c.Param("user_id")
	if !utils.IsValidUUID(userIDParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	targetUserID, err := uuid.FromString(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	authRequest := services.AuthorizationRequest{
		UserID:     currentUserUUID,
		Resource:   "profile",
		Action:     "read",
		ResourceID: &targetUserID,
	}

	decision, err := h.authzService.IsAuthorized(context.Background(), authRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
		return
	}

	if decision.Decision != "allowed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "reason": decision.Reason})
		return
	}

	user, err := h.userService.GetUserProfile(h.db, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		return
	}

	response := gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentUserUUID, ok := currentUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	authRequest := services.AuthorizationRequest{
		UserID:   currentUserUUID,
		Resource: "users",
		Action:   "list",
	}

	decision, err := h.authzService.IsAuthorized(context.Background(), authRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
		return
	}

	if decision.Decision != "allowed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "reason": decision.Reason})
		return
	}

	users, err := h.userService.GetUsers(h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	var response []gin.H
	for _, user := range users {
		response = append(response, gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentUserUUID, ok := currentUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userIDParam := c.Param("user_id")
	if !utils.IsValidUUID(userIDParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	targetUserID, err := uuid.FromString(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	authRequest := services.AuthorizationRequest{
		UserID:     currentUserUUID,
		Resource:   "user",
		Action:     "delete",
		ResourceID: &targetUserID,
	}

	decision, err := h.authzService.IsAuthorized(context.Background(), authRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
		return
	}

	if decision.Decision != "allowed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "reason": decision.Reason})
		return
	}

	err = h.userService.DeleteUser(h.db, targetUserID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *UserHandler) UpdateUserProfile(c *gin.Context) {

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentUserUUID, ok := currentUserID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	userIDParam := c.Param("user_id")
	if !utils.IsValidUUID(userIDParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	targetUserID, err := uuid.FromString(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	authRequest := services.AuthorizationRequest{
		UserID:     currentUserUUID,
		Resource:   "profile",
		Action:     "update",
		ResourceID: &targetUserID,
	}

	decision, err := h.authzService.IsAuthorized(context.Background(), authRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
		return
	}

	if decision.Decision != "allowed" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied", "reason": decision.Reason})
		return
	}

	err = h.userService.UpdateUser(h.db, targetUserID, updateData)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}
