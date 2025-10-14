package services

import (
	"errors"
	"task-manager/backend/internal/models"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegistrationRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=50"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	FirstName  string `json:"first_name" binding:"required,min=1,max=50"`
	LastName   string `json:"last_name" binding:"required,min=1,max=50"`
	Department string `json:"department,omitempty" binding:"max=100"`
	Position   string `json:"position,omitempty" binding:"max=100"`
}

type RegisterService interface {
	RegisterUser(db *gorm.DB, req RegistrationRequest) (*models.User, error)
}

type RegisterServiceImpl struct{}

func NewRegisterService() *RegisterServiceImpl {
	return &RegisterServiceImpl{}
}

func (s *RegisterServiceImpl) RegisterUser(db *gorm.DB, req RegistrationRequest) (*models.User, error) {
	var existingEmail models.User
	if err := db.Where("email = ?", req.Email).First(&existingEmail).Error; err == nil {
		return nil, errors.New("email already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	var existingUsername models.User
	if err := db.Where("username = ?", req.Username).First(&existingUsername).Error; err == nil {
		return nil, errors.New("username already exists")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		ID:         uuid.Must(uuid.NewV4()),
		Username:   req.Username,
		Email:      req.Email,
		Password:   string(hashedPassword),
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Department: req.Department,
		Position:   req.Position,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	var userRole models.Role
	if err := tx.Where("name = ?", "user").First(&userRole).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("default user role not found - please contact administrator")
		}
		return nil, err
	}

	userRoleAssignment := models.UserRole{
		ID:         uuid.Must(uuid.NewV4()),
		UserID:     user.ID,
		RoleID:     userRole.ID,
		AssignedAt: time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := tx.Create(&userRoleAssignment).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	defaultAttributes := []models.UserAttribute{
		{
			ID:        uuid.Must(uuid.NewV4()),
			UserID:    user.ID,
			Name:      "department",
			Value:     req.Department,
			Type:      "string",
			Source:    "registration",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.Must(uuid.NewV4()),
			UserID:    user.ID,
			Name:      "position",
			Value:     req.Position,
			Type:      "string",
			Source:    "registration",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.Must(uuid.NewV4()),
			UserID:    user.ID,
			Name:      "account_type",
			Value:     "standard",
			Type:      "string",
			Source:    "system",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.Must(uuid.NewV4()),
			UserID:    user.ID,
			Name:      "clearance_level",
			Value:     "basic",
			Type:      "string",
			Source:    "system",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, attr := range defaultAttributes {
		if attr.Value != "" {
			if err := tx.Create(&attr).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if err := db.Preload("Roles").First(&user, user.ID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
