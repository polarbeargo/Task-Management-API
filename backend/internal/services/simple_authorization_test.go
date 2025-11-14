package services_test

import (
	"context"
	"testing"

	"task-manager/backend/internal/services"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SimpleAuthorizationTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service services.AuthorizationService

	userID  uuid.UUID
	adminID uuid.UUID
	taskID  uuid.UUID
}

func (suite *SimpleAuthorizationTestSuite) SetupSuite() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			username TEXT,
			email TEXT,
			department TEXT,
			is_active BOOLEAN DEFAULT true
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE user_roles (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			role_id TEXT,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE permissions (
			id TEXT PRIMARY KEY,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			description TEXT
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE role_permissions (
			id TEXT PRIMARY KEY,
			role_id TEXT,
			permission_id TEXT,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE user_attributes (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			key TEXT,
			value TEXT,
			data_type TEXT,
			source TEXT,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			title TEXT,
			user_id TEXT,
			status TEXT,
			deleted_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	suite.db = db
	suite.service = services.NewAuthorizationService(db)
}

func (suite *SimpleAuthorizationTestSuite) SetupTest() {
	suite.db.Exec("DELETE FROM user_attributes")
	suite.db.Exec("DELETE FROM role_permissions")
	suite.db.Exec("DELETE FROM permissions")
	suite.db.Exec("DELETE FROM user_roles")
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM roles")
	suite.db.Exec("DELETE FROM tasks")

	suite.userID = uuid.Must(uuid.NewV4())
	suite.adminID = uuid.Must(uuid.NewV4())
	suite.taskID = uuid.Must(uuid.NewV4())

	suite.db.Exec("INSERT INTO users (id, username, email, department) VALUES (?, ?, ?, ?)",
		suite.userID.String(), "testuser", "test@example.com", "Engineering")
	suite.db.Exec("INSERT INTO users (id, username, email, department) VALUES (?, ?, ?, ?)",
		suite.adminID.String(), "admin", "admin@example.com", "IT")

	adminRoleID := uuid.Must(uuid.NewV4())
	userRoleID := uuid.Must(uuid.NewV4())

	suite.db.Exec("INSERT INTO roles (id, name) VALUES (?, ?)", adminRoleID.String(), "admin")
	suite.db.Exec("INSERT INTO roles (id, name) VALUES (?, ?)", userRoleID.String(), "user")

	suite.db.Exec("INSERT INTO user_roles (id, user_id, role_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), suite.adminID.String(), adminRoleID.String())
	suite.db.Exec("INSERT INTO user_roles (id, user_id, role_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), suite.userID.String(), userRoleID.String())

	taskReadPermID := uuid.Must(uuid.NewV4())
	profileReadPermID := uuid.Must(uuid.NewV4())

	suite.db.Exec("INSERT INTO permissions (id, resource, action, description) VALUES (?, ?, ?, ?)",
		taskReadPermID.String(), "task", "read", "Read task permission")
	suite.db.Exec("INSERT INTO permissions (id, resource, action, description) VALUES (?, ?, ?, ?)",
		profileReadPermID.String(), "profile", "read", "Read profile permission")

	suite.db.Exec("INSERT INTO role_permissions (id, role_id, permission_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), adminRoleID.String(), taskReadPermID.String())
	suite.db.Exec("INSERT INTO role_permissions (id, role_id, permission_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), adminRoleID.String(), profileReadPermID.String())
	suite.db.Exec("INSERT INTO role_permissions (id, role_id, permission_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), userRoleID.String(), profileReadPermID.String())

	suite.db.Exec("INSERT INTO tasks (id, title, user_id, status) VALUES (?, ?, ?, ?)",
		suite.taskID.String(), "Test Task", suite.userID.String(), "open")
}

func (suite *SimpleAuthorizationTestSuite) TestHasRole_Success() {
	ctx := context.Background()

	hasRole, err := suite.service.HasRole(ctx, suite.adminID, "admin")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	hasRole, err = suite.service.HasRole(ctx, suite.userID, "user")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)
}

func (suite *SimpleAuthorizationTestSuite) TestHasRole_NotFound() {
	ctx := context.Background()

	hasRole, err := suite.service.HasRole(ctx, suite.userID, "admin")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)

	hasRole, err = suite.service.HasRole(ctx, suite.adminID, "nonexistent")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)
}

func (suite *SimpleAuthorizationTestSuite) TestIsAuthorized_TaskOwnership() {
	ctx := context.Background()

	taskReadPermID := uuid.Must(uuid.NewV4())

	suite.db.Exec("INSERT INTO permissions (id, resource, action, description) VALUES (?, ?, ?, ?)",
		taskReadPermID.String(), "task", "read", "Read task permission")

	var existingUserRoleID string
	suite.db.Raw("SELECT role_id FROM user_roles WHERE user_id = ?", suite.userID.String()).Scan(&existingUserRoleID)

	suite.db.Exec("INSERT INTO role_permissions (id, role_id, permission_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), existingUserRoleID, taskReadPermID.String())

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &suite.taskID,
		Context:    map[string]interface{}{"task_owner": suite.userID.String()},
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "granted by RBAC")
}

func (suite *SimpleAuthorizationTestSuite) TestIsAuthorized_AdminOverride() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:     suite.adminID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &suite.taskID,
		Context:    map[string]interface{}{"task_owner": suite.userID.String()},
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "granted by RBAC")
}

func (suite *SimpleAuthorizationTestSuite) TestIsAuthorized_ProfileAccess() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "profile",
		Action:     "read",
		ResourceID: &suite.userID,
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "granted by RBAC")
}

func (suite *SimpleAuthorizationTestSuite) TestIsAuthorized_ProfileAccessDenied() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "profile",
		Action:     "update",
		ResourceID: &suite.adminID,
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "denied", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "lacks required RBAC permission")
}

func (suite *SimpleAuthorizationTestSuite) TestIsAuthorized_NoTaskOwnership() {
	ctx := context.Background()

	otherUserID := uuid.Must(uuid.NewV4())
	otherUserRoleID := uuid.Must(uuid.NewV4())

	suite.db.Exec("INSERT INTO users (id, username, email) VALUES (?, ?, ?)",
		otherUserID.String(), "otheruser", "other@example.com")
	suite.db.Exec("INSERT INTO roles (id, name) VALUES (?, ?)", otherUserRoleID.String(), "user")
	suite.db.Exec("INSERT INTO user_roles (id, user_id, role_id) VALUES (?, ?, ?)",
		uuid.Must(uuid.NewV4()).String(), otherUserID.String(), otherUserRoleID.String())

	request := services.AuthorizationRequest{
		UserID:     otherUserID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &suite.taskID,
		Context:    map[string]interface{}{"task_owner": suite.userID.String()},
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "denied", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "lacks required RBAC permission")
}

func (suite *SimpleAuthorizationTestSuite) TestAuthorizationCaching() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "profile",
		Action:     "read",
		ResourceID: &suite.userID,
	}

	for i := 0; i < 10; i++ {
		decision, err := suite.service.IsAuthorized(ctx, request)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), "allowed", decision.Decision)
	}
}

func (suite *SimpleAuthorizationTestSuite) TestAuthorizationContextValidation() {
	ctx := context.Background()

	tests := []struct {
		name     string
		request  services.AuthorizationRequest
		expected string
		reason   string
	}{
		{
			name: "EmptyResource",
			request: services.AuthorizationRequest{
				UserID:   suite.userID,
				Resource: "",
				Action:   "read",
			},
			expected: "denied",
			reason:   "lacks required RBAC permission",
		},
		{
			name: "EmptyAction",
			request: services.AuthorizationRequest{
				UserID:   suite.userID,
				Resource: "task",
				Action:   "",
			},
			expected: "denied",
			reason:   "lacks required RBAC permission",
		},
		{
			name: "InvalidUserID",
			request: services.AuthorizationRequest{
				UserID:   uuid.Nil,
				Resource: "task",
				Action:   "read",
			},
			expected: "denied",
			reason:   "lacks required RBAC permission",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			decision, err := suite.service.IsAuthorized(ctx, tt.request)
			if err != nil {
				assert.Contains(t, err.Error(), "invalid")
				return
			}
			assert.Equal(t, tt.expected, decision.Decision)
			assert.Contains(t, decision.Reason, tt.reason)
		})
	}
}

func TestSimpleAuthorizationTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleAuthorizationTestSuite))
}

func BenchmarkAuthorizationCheck(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service := services.NewAuthorizationService(db)
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY)")
	db.Exec("CREATE TABLE roles (id TEXT PRIMARY KEY, name TEXT)")
	db.Exec("CREATE TABLE user_roles (user_id TEXT, role_id TEXT)")

	request := services.AuthorizationRequest{
		UserID:     userID,
		Resource:   "profile",
		Action:     "read",
		ResourceID: &userID,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.IsAuthorized(ctx, request)
	}
}

func BenchmarkRoleCheck(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	service := services.NewAuthorizationService(db)
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())

	db.Exec("CREATE TABLE roles (id TEXT PRIMARY KEY, name TEXT)")
	db.Exec("CREATE TABLE user_roles (user_id TEXT, role_id TEXT)")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.HasRole(ctx, userID, "admin")
	}
}
