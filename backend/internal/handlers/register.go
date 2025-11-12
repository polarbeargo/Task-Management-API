package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"task-manager/backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var ErrDuplicateUsername = errors.New("username already exists")
var ErrDuplicateEmail = errors.New("email already exists")

type RegisterHandler struct {
	db              *gorm.DB
	registerService services.RegisterService
}

func NewRegisterHandler(db *gorm.DB, registerService services.RegisterService) *RegisterHandler {
	return &RegisterHandler{db: db, registerService: registerService}
}

type RegistrationResponse struct {
	Message string                 `json:"message"`
	User    RegistrationUserDetail `json:"user"`
}

type RegistrationUserDetail struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Department string `json:"department,omitempty"`
	Position   string `json:"position,omitempty"`
	IsActive   bool   `json:"is_active"`
	Role       string `json:"role"`
}

func (h *RegisterHandler) Registration(c *gin.Context) {
	var req services.RegistrationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	if err := h.validateRegistrationRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	user, err := h.registerService.RegisterUser(h.db, req)
	if err != nil {
		log.Printf("❌ Registration error: %v", err)

		if strings.Contains(err.Error(), "email already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Registration failed",
				"details": "An account with this email already exists",
			})
		} else if strings.Contains(err.Error(), "username already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Registration failed",
				"details": "This username is already taken",
			})
		} else if strings.Contains(err.Error(), "default user role not found") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Registration failed",
				"details": "System configuration error. Please contact administrator.",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Registration failed",
				"details": "An unexpected error occurred. Please try again later.",
			})
		}
		return
	}

	response := RegistrationResponse{
		Message: "Welcome to Taskify! Your account has been created successfully.",
		User: RegistrationUserDetail{
			ID:         user.ID.String(),
			Username:   user.Username,
			Email:      user.Email,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			Department: user.Department,
			Position:   user.Position,
			IsActive:   user.IsActive,
			Role:       "user",
		},
	}

	c.JSON(http.StatusCreated, response)
}

func (h *RegisterHandler) validateRegistrationRequest(req *services.RegistrationRequest) error {
	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	for _, char := range req.Username {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_') {
			return errors.New("username can only contain letters, numbers, and underscores")
		}
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if len(req.FirstName) == 0 {
		return errors.New("first name is required")
	}
	if len(req.LastName) == 0 {
		return errors.New("last name is required")
	}

	if err := h.validatePassword(req.Password); err != nil {
		return err
	}

	req.Department = strings.TrimSpace(req.Department)
	req.Position = strings.TrimSpace(req.Position)

	return nil
}

func (h *RegisterHandler) validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "number")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return errors.New("password must contain at least one " + strings.Join(missing, ", "))
	}

	return nil
}
