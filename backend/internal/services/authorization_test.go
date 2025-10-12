package services_test

import (
	"context"
	"testing"
	"time"

	"task-manager/backend/internal/models"
	"task-manager/backend/internal/services"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type AuthorizationTestSuite struct {
	suite.Suite
	db      *gorm.DB
	service services.AuthorizationService

	userID    uuid.UUID
	adminID   uuid.UUID
	managerID uuid.UUID
	userRole  models.Role
	adminRole models.Role
	taskPerm  models.Permission
	userPerm  models.Permission
}

func (suite *AuthorizationTestSuite) SetupSuite() {
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

	err = db.Exec(`
		CREATE TABLE roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			modified_by TEXT
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE permissions (
			id TEXT PRIMARY KEY,
			name TEXT,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			description TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			modified_by TEXT
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE user_roles (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			assigned_by TEXT,
			assigned_at DATETIME,
			expires_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE role_permissions (
			id TEXT PRIMARY KEY,
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			assigned_by TEXT,
			assigned_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (permission_id) REFERENCES permissions(id)
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE user_attributes (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			source TEXT,
			expires_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE resource_attributes (
			id TEXT PRIMARY KEY,
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			name TEXT NOT NULL,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			source TEXT,
			expires_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			resource_id TEXT,
			decision TEXT NOT NULL,
			reason TEXT,
			ip_address TEXT,
			user_agent TEXT,
			request_method TEXT,
			request_path TEXT,
			context TEXT,
			timestamp DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`).Error
	suite.Require().NoError(err)

	err = db.Exec(`
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT,
			priority TEXT,
			due_date DATETIME,
			user_id TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`).Error
	suite.Require().NoError(err)

	suite.db = db

	suite.service = services.NewAuthorizationService(db)
}

func (suite *AuthorizationTestSuite) SetupTest() {
	suite.db.Exec("DELETE FROM audit_logs")
	suite.db.Exec("DELETE FROM user_attributes")
	suite.db.Exec("DELETE FROM resource_attributes")
	suite.db.Exec("DELETE FROM role_permissions")
	suite.db.Exec("DELETE FROM user_roles")
	suite.db.Exec("DELETE FROM permissions")
	suite.db.Exec("DELETE FROM roles")
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM tasks")

	suite.userID = uuid.Must(uuid.NewV4())
	suite.adminID = uuid.Must(uuid.NewV4())
	suite.managerID = uuid.Must(uuid.NewV4())

	users := []models.User{
		{
			ID:         suite.userID,
			Username:   "testuser",
			Email:      "user@test.com",
			Password:   "hashedpassword",
			FirstName:  "Test",
			LastName:   "User",
			Department: "Engineering",
			Position:   "Developer",
			IsActive:   true,
		},
		{
			ID:         suite.adminID,
			Username:   "admin",
			Email:      "admin@test.com",
			Password:   "hashedpassword",
			FirstName:  "Admin",
			LastName:   "User",
			Department: "IT",
			Position:   "Administrator",
			IsActive:   true,
		},
		{
			ID:         suite.managerID,
			Username:   "manager",
			Email:      "manager@test.com",
			Password:   "hashedpassword",
			FirstName:  "Manager",
			LastName:   "User",
			Department: "Engineering",
			Position:   "Manager",
			IsActive:   true,
		},
	}

	for _, user := range users {
		err := suite.db.Create(&user).Error
		suite.Require().NoError(err)
	}

	suite.userRole = models.Role{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "user",
		Description: "Regular user role",
	}
	suite.adminRole = models.Role{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "admin",
		Description: "Administrator role",
	}

	roles := []models.Role{suite.userRole, suite.adminRole}
	for _, role := range roles {
		err := suite.db.Create(&role).Error
		suite.Require().NoError(err)
	}

	suite.taskPerm = models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "task:read",
		Resource:    "task",
		Action:      "read",
		Description: "Read task permission",
	}
	suite.userPerm = models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "user:read",
		Resource:    "user",
		Action:      "read",
		Description: "Read user permission",
	}

	taskCreatePerm := models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "task:create",
		Resource:    "task",
		Action:      "create",
		Description: "Create task permission",
	}
	taskUpdatePerm := models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "task:update",
		Resource:    "task",
		Action:      "update",
		Description: "Update task permission",
	}
	taskDeletePerm := models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "task:delete",
		Resource:    "task",
		Action:      "delete",
		Description: "Delete task permission",
	}
	userUpdatePerm := models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        "user:update",
		Resource:    "user",
		Action:      "update",
		Description: "Update user permission",
	}

	permissions := []models.Permission{
		suite.taskPerm, suite.userPerm, taskCreatePerm, taskUpdatePerm, taskDeletePerm, userUpdatePerm,
	}
	for _, perm := range permissions {
		err := suite.db.Create(&perm).Error
		suite.Require().NoError(err)
	}

	userRoles := []models.UserRole{
		{
			ID:         uuid.Must(uuid.NewV4()),
			UserID:     suite.userID,
			RoleID:     suite.userRole.ID,
			AssignedBy: suite.adminID,
			AssignedAt: time.Now(),
		},
		{
			ID:         uuid.Must(uuid.NewV4()),
			UserID:     suite.adminID,
			RoleID:     suite.adminRole.ID,
			AssignedBy: suite.adminID,
			AssignedAt: time.Now(),
		},
		{
			ID:         uuid.Must(uuid.NewV4()),
			UserID:     suite.managerID,
			RoleID:     suite.userRole.ID,
			AssignedBy: suite.adminID,
			AssignedAt: time.Now(),
		},
	}

	for _, userRole := range userRoles {
		err := suite.db.Create(&userRole).Error
		suite.Require().NoError(err)
	}

	rolePermissions := []models.RolePermission{
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.userRole.ID,
			PermissionID: suite.taskPerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.userRole.ID,
			PermissionID: taskCreatePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.userRole.ID,
			PermissionID: taskUpdatePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.userRole.ID,
			PermissionID: taskDeletePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: suite.taskPerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: suite.userPerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: taskCreatePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: taskUpdatePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: taskDeletePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
		{
			ID:           uuid.Must(uuid.NewV4()),
			RoleID:       suite.adminRole.ID,
			PermissionID: userUpdatePerm.ID,
			AssignedBy:   suite.adminID,
			AssignedAt:   time.Now(),
		},
	}

	for _, rolePerm := range rolePermissions {
		err := suite.db.Create(&rolePerm).Error
		suite.Require().NoError(err)
	}

	userAttributes := []models.UserAttribute{
		{
			ID:     uuid.Must(uuid.NewV4()),
			UserID: suite.userID,
			Name:   "department",
			Value:  "Engineering",
			Type:   "string",
			Source: "system",
		},
		{
			ID:     uuid.Must(uuid.NewV4()),
			UserID: suite.managerID,
			Name:   "department",
			Value:  "Engineering",
			Type:   "string",
			Source: "system",
		},
		{
			ID:     uuid.Must(uuid.NewV4()),
			UserID: suite.adminID,
			Name:   "department",
			Value:  "IT",
			Type:   "string",
			Source: "system",
		},
	}

	for _, attr := range userAttributes {
		err := suite.db.Create(&attr).Error
		suite.Require().NoError(err)
	}
}

func (suite *AuthorizationTestSuite) TestHasRole() {
	ctx := context.Background()

	hasRole, err := suite.service.HasRole(ctx, suite.userID, "user")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	hasRole, err = suite.service.HasRole(ctx, suite.userID, "admin")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)

	hasRole, err = suite.service.HasRole(ctx, suite.adminID, "admin")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)
}

func (suite *AuthorizationTestSuite) TestHasPermission() {
	ctx := context.Background()

	hasPerm, err := suite.service.HasPermission(ctx, suite.userID, "task", "read")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasPerm)

	hasPerm, err = suite.service.HasPermission(ctx, suite.userID, "user", "read")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasPerm)

	hasPerm, err = suite.service.HasPermission(ctx, suite.adminID, "task", "read")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasPerm)

	hasPerm, err = suite.service.HasPermission(ctx, suite.adminID, "user", "read")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasPerm)
}

func (suite *AuthorizationTestSuite) TestAssignRole() {
	ctx := context.Background()

	newRole, err := suite.service.CreateRole(ctx, "manager", "Manager role")
	assert.NoError(suite.T(), err)

	err = suite.service.AssignRole(ctx, suite.userID, newRole.ID, suite.adminID)
	assert.NoError(suite.T(), err)

	hasRole, err := suite.service.HasRole(ctx, suite.userID, "manager")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	err = suite.service.AssignRole(ctx, suite.userID, newRole.ID, suite.adminID)
	assert.Error(suite.T(), err)
}

func (suite *AuthorizationTestSuite) TestRevokeRole() {
	ctx := context.Background()

	hasRole, err := suite.service.HasRole(ctx, suite.userID, "user")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasRole)

	err = suite.service.RevokeRole(ctx, suite.userID, suite.userRole.ID)
	assert.NoError(suite.T(), err)

	hasRole, err = suite.service.HasRole(ctx, suite.userID, "user")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), hasRole)
}

func (suite *AuthorizationTestSuite) TestIsAuthorized_TaskOwnership() {
	ctx := context.Background()

	taskID := uuid.Must(uuid.NewV4())
	task := models.Task{
		ID:          taskID,
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test task description",
		Status:      "pending",
	}
	err := suite.db.Create(&task).Error
	suite.Require().NoError(err)

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &taskID,
		Context:    map[string]interface{}{"target_task_id": taskID.String()},
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
		RequestID:  "test-123",
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "Access granted")

	request.UserID = suite.managerID
	decision, err = suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
}

func (suite *AuthorizationTestSuite) TestIsAuthorized_AdminOverride() {
	ctx := context.Background()

	taskID := uuid.Must(uuid.NewV4())
	task := models.Task{
		ID:          taskID,
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test task description",
		Status:      "pending",
	}
	err := suite.db.Create(&task).Error
	suite.Require().NoError(err)

	request := services.AuthorizationRequest{
		UserID:     suite.adminID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &taskID,
		Context:    map[string]interface{}{"target_task_id": taskID.String()},
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
		RequestID:  "test-123",
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "Access granted")
}

func (suite *AuthorizationTestSuite) TestIsAuthorized_DepartmentAccess() {
	ctx := context.Background()

	taskID := uuid.Must(uuid.NewV4())
	task := models.Task{
		ID:          taskID,
		UserID:      suite.userID,
		Title:       "Test Task",
		Description: "Test task description",
		Status:      "pending",
	}
	err := suite.db.Create(&task).Error
	suite.Require().NoError(err)

	request := services.AuthorizationRequest{
		UserID:     suite.managerID,
		Resource:   "task",
		Action:     "read",
		ResourceID: &taskID,
		Context:    map[string]interface{}{"target_task_id": taskID.String()},
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
		RequestID:  "test-123",
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), decision.Reason)
}

func (suite *AuthorizationTestSuite) TestIsAuthorized_UserProfile() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:     suite.userID,
		Resource:   "user",
		Action:     "read",
		ResourceID: &suite.userID,
		Context:    map[string]interface{}{"target_user_id": suite.userID.String()},
		IPAddress:  "127.0.0.1",
		UserAgent:  "test",
		RequestID:  "test-123",
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "denied", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "User lacks required RBAC permission")

	request.ResourceID = &suite.managerID
	request.Action = "update"
	request.Context["target_user_id"] = suite.managerID.String()

	decision, err = suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "denied", decision.Decision)
}

func (suite *AuthorizationTestSuite) TestIsAuthorized_MissingRBACPermission() {
	ctx := context.Background()

	request := services.AuthorizationRequest{
		UserID:    suite.userID,
		Resource:  "admin",
		Action:    "read",
		Context:   map[string]interface{}{},
		IPAddress: "127.0.0.1",
		UserAgent: "test",
		RequestID: "test-123",
	}

	decision, err := suite.service.IsAuthorized(ctx, request)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "denied", decision.Decision)
	assert.Contains(suite.T(), decision.Reason, "RBAC")
}

func (suite *AuthorizationTestSuite) TestSetUserAttribute() {
	ctx := context.Background()

	err := suite.service.SetUserAttribute(ctx, suite.userID, "clearance_level", "secret", "string")
	assert.NoError(suite.T(), err)

	var attr models.UserAttribute
	err = suite.db.Where("user_id = ? AND name = ?", suite.userID, "clearance_level").First(&attr).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "secret", attr.Value)
	assert.Equal(suite.T(), "string", attr.Type)
}

func (suite *AuthorizationTestSuite) TestSetResourceAttribute() {
	ctx := context.Background()

	resourceID := uuid.Must(uuid.NewV4())
	err := suite.service.SetResourceAttribute(ctx, "task", resourceID, "classification", "confidential", "string")
	assert.NoError(suite.T(), err)

	var attr models.ResourceAttribute
	err = suite.db.Where("resource_type = ? AND resource_id = ? AND name = ?", "task", resourceID, "classification").First(&attr).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "confidential", attr.Value)
}

func (suite *AuthorizationTestSuite) TestCreatePermission() {
	ctx := context.Background()

	perm, err := suite.service.CreatePermission(ctx, "document", "write", "Write document permission")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "document:write", perm.Name)
	assert.Equal(suite.T(), "document", perm.Resource)
	assert.Equal(suite.T(), "write", perm.Action)
}

func (suite *AuthorizationTestSuite) TestGrantPermissionToRole() {
	ctx := context.Background()

	perm, err := suite.service.CreatePermission(ctx, "document", "write", "Write document permission")
	assert.NoError(suite.T(), err)

	err = suite.service.GrantPermissionToRole(ctx, suite.userRole.ID, perm.ID, suite.adminID)
	assert.NoError(suite.T(), err)

	hasPerm, err := suite.service.HasPermission(ctx, suite.userID, "document", "write")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), hasPerm)
}

func (suite *AuthorizationTestSuite) TestLogAuthorizationDecision() {
	ctx := context.Background()

	decision := services.AuthorizationDecision{
		UserID:     suite.userID,
		Resource:   "task",
		Action:     "read",
		Decision:   "allowed",
		Reason:     "User owns the task",
		PolicyType: "abac",
		Context:    map[string]interface{}{"test": "value"},
		Timestamp:  time.Now(),
		IPAddress:  "127.0.0.1",
		UserAgent:  "test-agent",
		RequestID:  "test-123",
	}

	err := suite.service.LogAuthorizationDecision(ctx, decision)
	assert.NoError(suite.T(), err)

	var auditLog models.AuditLog
	err = suite.db.Where("user_id = ? AND action = ?", suite.userID, "read_task").First(&auditLog).Error
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "allowed", auditLog.Decision)
	assert.Equal(suite.T(), "127.0.0.1", auditLog.IPAddress)
}

func (suite *AuthorizationTestSuite) TestAuthorizationScenarios() {
	ctx := context.Background()

	scenarios := []struct {
		name           string
		userID         uuid.UUID
		resource       string
		action         string
		resourceID     *uuid.UUID
		expectedResult string
		description    string
	}{
		{
			name:           "UserAccessOwnTask",
			userID:         suite.userID,
			resource:       "task",
			action:         "create",
			resourceID:     nil,
			expectedResult: "allowed",
			description:    "User should be able to create tasks",
		},
		{
			name:           "AdminAccessAnyResource",
			userID:         suite.adminID,
			resource:       "task",
			action:         "read",
			resourceID:     nil,
			expectedResult: "allowed",
			description:    "Admin should access any resource",
		},
		{
			name:           "UserAccessWithoutPermission",
			userID:         suite.userID,
			resource:       "admin",
			action:         "read",
			resourceID:     nil,
			expectedResult: "denied",
			description:    "User should not access admin resources",
		},
	}

	for _, scenario := range scenarios {
		suite.T().Run(scenario.name, func(t *testing.T) {
			request := services.AuthorizationRequest{
				UserID:     scenario.userID,
				Resource:   scenario.resource,
				Action:     scenario.action,
				ResourceID: scenario.resourceID,
				Context:    map[string]interface{}{},
				IPAddress:  "127.0.0.1",
				UserAgent:  "test",
				RequestID:  "test-" + scenario.name,
			}

			decision, err := suite.service.IsAuthorized(ctx, request)
			assert.NoError(t, err, scenario.description)
			assert.Equal(t, scenario.expectedResult, decision.Decision, scenario.description)
		})
	}
}

func TestAuthorizationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthorizationTestSuite))
}
