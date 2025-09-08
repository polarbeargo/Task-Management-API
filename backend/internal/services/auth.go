package services

import (
	"os"
	"task-manager/backend/internal/models"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	LoginUser(db *gorm.DB, username, password string) (*models.User, error)
	GenerateToken(db *gorm.DB, userID uuid.UUID) (string, string, error)
	RefreshToken(db *gorm.DB, refreshToken string) (string, string, int64, error)
}

type AuthServiceImpl struct{}

func (s *AuthServiceImpl) RefreshToken(db *gorm.DB, refreshToken string) (string, string, int64, error) {

	var token models.Token
	err := db.Where("refresh_token = ? AND expires_at > ?", refreshToken, time.Now()).First(&token).Error
	if err != nil {
		return "", "", 0, err
	}

	accessToken, newRefreshToken, err := s.GenerateToken(db, token.UserId)
	if err != nil {
		return "", "", 0, err
	}
	expiresIn := int64(3600)

	db.Delete(&token)

	return accessToken, newRefreshToken, expiresIn, nil
}

func NewAuthService() *AuthServiceImpl {
	return &AuthServiceImpl{}
}

func VerifyPassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func (s *AuthServiceImpl) LoginUser(db *gorm.DB, username, password string) (*models.User, error) {
	var user models.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	if !VerifyPassword(user.Password, password) {
		return nil, gorm.ErrInvalidData
	}
	return &user, nil
}

func (s *AuthServiceImpl) GenerateToken(db *gorm.DB, userID uuid.UUID) (string, string, error) {

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}

	var roleName string
	err := db.Raw("SELECT r.name FROM roles r JOIN user_roles ur ON r.id = ur.role_id WHERE ur.user_id = ? LIMIT 1", userID).Scan(&roleName).Error
	if err != nil || roleName == "" {
		roleName = "user"
	}

	var permissions []string
	err = db.Raw(`SELECT p.resource || ':' || p.action FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?`, userID).Scan(&permissions).Error
	if err != nil {
		permissions = []string{}
	}

	accessTokenClaims := jwt.MapClaims{
		"user_id":     userID.String(),
		"role":        roleName,
		"permissions": permissions,
		"exp":         time.Now().Add(time.Hour).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	refreshTokenUUID, err := uuid.NewV4()
	if err != nil {
		return "", "", err
	}
	refreshTokenString := refreshTokenUUID.String()

	expiresAt := time.Now().Add(time.Hour)
	token := models.Token{
		ID:           uuid.Must(uuid.NewV4()),
		UserId:       userID,
		RefreshToken: refreshTokenUUID,
		ExpiresAt:    expiresAt,
	}
	if err := db.Create(&token).Error; err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}
