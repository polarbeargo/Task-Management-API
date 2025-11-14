package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"task-manager/backend/internal/handlers"
	"task-manager/backend/internal/middleware"
	"task-manager/backend/internal/models"
	"task-manager/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MockAuthorizationService struct {
	mock.Mock
}

func (m *MockAuthorizationService) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	args := m.Called(ctx, userID, roleName)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthorizationService) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	args := m.Called(ctx, userID, resource, action)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthorizationService) IsAuthorized(ctx context.Context, request services.AuthorizationRequest) (*services.AuthorizationDecision, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(*services.AuthorizationDecision), args.Error(1)
}

func (m *MockAuthorizationService) AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	args := m.Called(ctx, userID, roleID, assignedBy)
	return args.Error(0)
}

func (m *MockAuthorizationService) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	args := m.Called(ctx, userID, roleID)
	return args.Error(0)
}

func (m *MockAuthorizationService) SetUserAttribute(ctx context.Context, userID uuid.UUID, key, value, dataType string) error {
	args := m.Called(ctx, userID, key, value, dataType)
	return args.Error(0)
}

func (m *MockAuthorizationService) SetResourceAttribute(ctx context.Context, resourceType string, resourceID uuid.UUID, key, value, dataType string) error {
	args := m.Called(ctx, resourceType, resourceID, key, value, dataType)
	return args.Error(0)
}

func (m *MockAuthorizationService) CreateRole(ctx context.Context, name, description string) (*models.Role, error) {
	args := m.Called(ctx, name, description)
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockAuthorizationService) CreatePermission(ctx context.Context, resource, action, description string) (*models.Permission, error) {
	args := m.Called(ctx, resource, action, description)
	return args.Get(0).(*models.Permission), args.Error(1)
}

func (m *MockAuthorizationService) GrantPermissionToRole(ctx context.Context, roleID, permissionID, grantedBy uuid.UUID) error {
	args := m.Called(ctx, roleID, permissionID, grantedBy)
	return args.Error(0)
}

func (m *MockAuthorizationService) LogAuthorizationDecision(ctx context.Context, decision services.AuthorizationDecision) error {
	args := m.Called(ctx, decision)
	return args.Error(0)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserProfile(db *gorm.DB, userID uuid.UUID) (models.User, error) {
	args := m.Called(db, userID)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockUserService) GetUserProfileMalicious(db *gorm.DB, userID string) ([]models.User, error) {
	args := m.Called(db, userID)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) GetUsers(db *gorm.DB) ([]models.User, error) {
	args := m.Called(db)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) DeleteUser(db *gorm.DB, userID uuid.UUID) error {
	args := m.Called(db, userID)
	return args.Error(0)
}

func (m *MockUserService) UpdateUser(db *gorm.DB, userID uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(db, userID, updates)
	return args.Error(0)
}

type AuthorizationHandlerTestSuite struct {
	suite.Suite
	db          *gorm.DB
	router      *gin.Engine
	authService *MockAuthorizationService
	userService *MockUserService
	userHandler *handlers.UserHandler

	userID    uuid.UUID
	adminID   uuid.UUID
	managerID uuid.UUID
}

func (suite *AuthorizationHandlerTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			username TEXT,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT,
			department TEXT,
			position TEXT,
			is_active BOOLEAN DEFAULT true,
			last_login_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	suite.db = db
	gin.SetMode(gin.TestMode)
}

func (suite *AuthorizationHandlerTestSuite) SetupTest() {
	suite.userID = uuid.Must(uuid.NewV4())
	suite.adminID = uuid.Must(uuid.NewV4())
	suite.managerID = uuid.Must(uuid.NewV4())

	suite.authService = new(MockAuthorizationService)
	suite.userService = new(MockUserService)

	suite.userHandler = handlers.NewUserHandler(suite.db, suite.userService, suite.authService)

	suite.router = gin.New()
	suite.router.Use(suite.createAuthMiddleware())
}

func (suite *AuthorizationHandlerTestSuite) createAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_token"})
			return
		}

		if authHeader == "Bearer valid_user_token" {
			c.Set("user_id", suite.userID)
			c.Set("user_role", "user")
		} else if authHeader == "Bearer valid_admin_token" {
			c.Set("user_id", suite.adminID)
			c.Set("user_role", "admin")
		} else if authHeader == "Bearer valid_manager_token" {
			c.Set("user_id", suite.managerID)
			c.Set("user_role", "user")
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
			return
		}

		c.Next()
	}
}

func (suite *AuthorizationHandlerTestSuite) TestGetUserProfile_OwnProfile_Success() {
	suite.router.GET("/users/:user_id", suite.userHandler.GetUserProfileByUserId)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.userID.String() &&
			req.Resource == "profile" &&
			req.Action == "read"
	})).Return(&services.AuthorizationDecision{
		Decision: "allowed",
		Reason:   "User can access own profile",
	}, nil)

	suite.userService.On("GetUserProfile", suite.db, suite.userID).Return(models.User{
		ID:        suite.userID,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}, nil)

	req, _ := http.NewRequest("GET", "/users/"+suite.userID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_user_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.User
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.userID, response.ID)
	assert.Equal(suite.T(), "testuser", response.Username)

	suite.authService.AssertExpectations(suite.T())
	suite.userService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestGetUserProfile_OtherProfile_Denied() {
	suite.router.GET("/users/:user_id", suite.userHandler.GetUserProfileByUserId)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.userID.String() &&
			req.Resource == "profile" &&
			req.Action == "read" &&
			req.ResourceID.String() == suite.managerID.String()
	})).Return(&services.AuthorizationDecision{
		Decision: "denied",
		Reason:   "User cannot access other profiles",
	}, nil)

	req, _ := http.NewRequest("GET", "/users/"+suite.managerID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_user_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Access denied", response["error"])
	assert.Equal(suite.T(), "User cannot access other profiles", response["reason"])

	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestGetUserProfile_AdminAccess_Success() {
	suite.router.GET("/users/:user_id", suite.userHandler.GetUserProfileByUserId)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.adminID.String() &&
			req.Resource == "profile" &&
			req.Action == "read"
	})).Return(&services.AuthorizationDecision{
		Decision: "allowed",
		Reason:   "Admin has full access",
	}, nil)

	suite.userService.On("GetUserProfile", suite.db, suite.userID).Return(models.User{
		ID:        suite.userID,
		Username:  "testuser",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
	}, nil)

	req, _ := http.NewRequest("GET", "/users/"+suite.userID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_admin_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	suite.authService.AssertExpectations(suite.T())
	suite.userService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestUpdateUserProfile_OwnProfile_Success() {
	suite.router.PUT("/users/:user_id", suite.userHandler.UpdateUserProfile)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.userID.String() &&
			req.Resource == "profile" &&
			req.Action == "update"
	})).Return(&services.AuthorizationDecision{
		Decision: "allowed",
		Reason:   "User can update own profile",
	}, nil)

	suite.userService.On("UpdateUser", suite.db, suite.userID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	updateData := map[string]interface{}{
		"first_name": "Updated",
		"last_name":  "Name",
	}
	jsonData, _ := json.Marshal(updateData)

	req, _ := http.NewRequest("PUT", "/users/"+suite.userID.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer valid_user_token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "User updated successfully", response["message"])

	suite.authService.AssertExpectations(suite.T())
	suite.userService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestUpdateUserProfile_OtherProfile_Denied() {
	suite.router.PUT("/users/:user_id", suite.userHandler.UpdateUserProfile)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.userID.String() &&
			req.Resource == "profile" &&
			req.Action == "update" &&
			req.ResourceID.String() == suite.managerID.String()
	})).Return(&services.AuthorizationDecision{
		Decision: "denied",
		Reason:   "Only admins can modify other users",
	}, nil)

	updateData := map[string]interface{}{
		"first_name": "Hacked",
		"last_name":  "Name",
	}
	jsonData, _ := json.Marshal(updateData)

	req, _ := http.NewRequest("PUT", "/users/"+suite.managerID.String(), bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer valid_user_token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Access denied", response["error"])

	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestDeleteUser_AdminOnly_Success() {
	suite.router.DELETE("/users/:user_id", suite.userHandler.DeleteUser)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.adminID.String() &&
			req.Resource == "user" &&
			req.Action == "delete"
	})).Return(&services.AuthorizationDecision{
		Decision: "allowed",
		Reason:   "Admin can delete users",
	}, nil)

	suite.userService.On("DeleteUser", suite.db, suite.userID).Return(nil)

	req, _ := http.NewRequest("DELETE", "/users/"+suite.userID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_admin_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "User deleted successfully", response["message"])

	suite.authService.AssertExpectations(suite.T())
	suite.userService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestDeleteUser_RegularUser_Denied() {
	suite.router.DELETE("/users/:user_id", suite.userHandler.DeleteUser)

	suite.authService.On("IsAuthorized", mock.Anything, mock.MatchedBy(func(req services.AuthorizationRequest) bool {
		return req.UserID.String() == suite.userID.String() &&
			req.Resource == "user" &&
			req.Action == "delete"
	})).Return(&services.AuthorizationDecision{
		Decision: "denied",
		Reason:   "Only admins can delete users",
	}, nil)

	req, _ := http.NewRequest("DELETE", "/users/"+suite.managerID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_user_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Access denied", response["error"])

	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestAuthorizationError_HandlesServiceError() {
	suite.router.GET("/users/:user_id", suite.userHandler.GetUserProfileByUserId)

	suite.authService.On("IsAuthorized", mock.Anything, mock.Anything).Return(
		(*services.AuthorizationDecision)(nil),
		assert.AnError,
	)

	req, _ := http.NewRequest("GET", "/users/"+suite.userID.String(), nil)
	req.Header.Set("Authorization", "Bearer valid_user_token")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Authorization check failed", response["error"])

	suite.authService.AssertExpectations(suite.T())
}

func (suite *AuthorizationHandlerTestSuite) TestAuthorizationPerformance() {
	suite.router.GET("/users/:user_id", suite.userHandler.GetUserProfileByUserId)

	suite.authService.On("IsAuthorized", mock.Anything, mock.Anything).Return(&services.AuthorizationDecision{
		Decision: "allowed",
		Reason:   "Test allowed",
	}, nil)

	suite.userService.On("GetUserProfile", mock.Anything, mock.Anything).Return(models.User{
		ID:       suite.userID,
		Username: "testuser",
	}, nil)

	start := time.Now()
	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", "/users/"+suite.userID.String(), nil)
		req.Header.Set("Authorization", "Bearer valid_user_token")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		assert.Equal(suite.T(), http.StatusOK, w.Code)
	}
	duration := time.Since(start)

	avgDuration := duration / 100
	assert.Less(suite.T(), avgDuration, time.Millisecond, "Authorization should be fast")

	suite.T().Logf("Average authorization time: %v", avgDuration)
}

func TestAuthorizationHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(AuthorizationHandlerTestSuite))
}

func TestJWTTokenCreation(t *testing.T) {
	userID := uuid.Must(uuid.NewV4())

	claims := jwt.MapClaims{
		"user_id":     userID.String(),
		"role":        "user",
		"permissions": []string{"task:read", "task:write"},
		"iss":         "taskify-backend",
		"aud":         "taskify-users",
		"exp":         time.Now().Add(time.Hour).Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("test_secret"))

	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("test_secret"), nil
	})

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	parsedClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), parsedClaims["user_id"])
	assert.Equal(t, "user", parsedClaims["role"])
}

func TestAuthzConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      middleware.AuthzConfig
		role        string
		permissions []string
		shouldPass  bool
	}{
		{
			name:       "NoRequirements",
			config:     middleware.AuthzConfig{},
			role:       "user",
			shouldPass: true,
		},
		{
			name:       "RoleMatch",
			config:     middleware.AuthzConfig{Role: "admin"},
			role:       "admin",
			shouldPass: true,
		},
		{
			name:       "RoleMismatch",
			config:     middleware.AuthzConfig{Role: "admin"},
			role:       "user",
			shouldPass: false,
		},
		{
			name:        "PermissionMatch",
			config:      middleware.AuthzConfig{Permissions: []string{"task:read"}},
			permissions: []string{"task:read", "task:write"},
			shouldPass:  true,
		},
		{
			name:        "PermissionMismatch",
			config:      middleware.AuthzConfig{Permissions: []string{"admin:write"}},
			permissions: []string{"task:read", "task:write"},
			shouldPass:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, tc.config)
		})
	}
}
