package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"task-manager/backend/internal/models"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type AuthorizationService interface {
	HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error)
	HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
	RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error

	IsAuthorized(ctx context.Context, request AuthorizationRequest) (*AuthorizationDecision, error)
	SetUserAttribute(ctx context.Context, userID uuid.UUID, key, value, dataType string) error
	SetResourceAttribute(ctx context.Context, resourceType string, resourceID uuid.UUID, key, value, dataType string) error

	CreateRole(ctx context.Context, name, description string) (*models.Role, error)
	CreatePermission(ctx context.Context, resource, action, description string) (*models.Permission, error)
	GrantPermissionToRole(ctx context.Context, roleID, permissionID, grantedBy uuid.UUID) error

	LogAuthorizationDecision(ctx context.Context, decision AuthorizationDecision) error
}

type AuthorizationServiceImpl struct {
	db *gorm.DB
}

func NewAuthorizationService(db *gorm.DB) AuthorizationService {
	return &AuthorizationServiceImpl{db: db}
}

type AuthorizationRequest struct {
	UserID     uuid.UUID              `json:"user_id"`
	Resource   string                 `json:"resource"`    
	Action     string                 `json:"action"`      
	ResourceID *uuid.UUID             `json:"resource_id"` 
	Context    map[string]interface{} `json:"context"`     
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	RequestID  string                 `json:"request_id"`
}

type AuthorizationDecision struct {
	UserID     uuid.UUID              `json:"user_id"`
	Resource   string                 `json:"resource"`
	Action     string                 `json:"action"`
	ResourceID *uuid.UUID             `json:"resource_id"`
	Decision   string                 `json:"decision"` 
	Reason     string                 `json:"reason"`
	PolicyType string                 `json:"policy_type"` 
	Context    map[string]interface{} `json:"context"`
	Timestamp  time.Time              `json:"timestamp"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	RequestID  string                 `json:"request_id"`
}

func (s *AuthorizationServiceImpl) HasRole(ctx context.Context, userID uuid.UUID, roleName string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.name = ? AND user_roles.deleted_at IS NULL", userID, roleName).
		Count(&count).Error

	return count > 0, err
}

func (s *AuthorizationServiceImpl) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where(`user_roles.user_id = ? 
			   AND permissions.resource = ? 
			   AND permissions.action = ? 
			   AND user_roles.deleted_at IS NULL 
			   AND role_permissions.deleted_at IS NULL`, userID, resource, action).
		Count(&count).Error

	return count > 0, err
}

func (s *AuthorizationServiceImpl) AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	var existing models.UserRole
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		First(&existing).Error

	if err == nil {
		return fmt.Errorf("user already has this role")
	}

	if err != gorm.ErrRecordNotFound {
		return err
	}

	userRole := models.UserRole{
		ID:         uuid.Must(uuid.NewV4()),
		UserID:     userID,
		RoleID:     roleID,
		AssignedBy: assignedBy,
		AssignedAt: time.Now(),
	}

	return s.db.WithContext(ctx).Create(&userRole).Error
}

func (s *AuthorizationServiceImpl) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&models.UserRole{}).Error
}

func (s *AuthorizationServiceImpl) IsAuthorized(ctx context.Context, request AuthorizationRequest) (*AuthorizationDecision, error) {
	decision := &AuthorizationDecision{
		UserID:     request.UserID,
		Resource:   request.Resource,
		Action:     request.Action,
		ResourceID: request.ResourceID,
		Decision:   "denied",
		Reason:     "",
		PolicyType: "combined",
		Context:    request.Context,
		Timestamp:  time.Now(),
		IPAddress:  request.IPAddress,
		UserAgent:  request.UserAgent,
		RequestID:  request.RequestID,
	}

	hasPermission, err := s.HasPermission(ctx, request.UserID, request.Resource, request.Action)
	if err != nil {
		decision.Reason = fmt.Sprintf("RBAC check failed: %v", err)
		return decision, err
	}

	if !hasPermission {
		decision.Reason = "User lacks required RBAC permission"
		decision.PolicyType = "rbac"
		return decision, nil
	}

	allowed, reason, err := s.evaluateABACPolicies(ctx, request)
	if err != nil {
		decision.Reason = fmt.Sprintf("ABAC evaluation failed: %v", err)
		return decision, err
	}

	if !allowed {
		decision.Reason = reason
		decision.PolicyType = "abac"
		return decision, nil
	}

	decision.Decision = "allowed"
	decision.Reason = "Access granted by RBAC and ABAC policies"

	return decision, nil
}

func (s *AuthorizationServiceImpl) evaluateABACPolicies(ctx context.Context, request AuthorizationRequest) (bool, string, error) {
	var userAttrs []models.UserAttribute
	err := s.db.WithContext(ctx).
		Where("user_id = ?", request.UserID).
		Find(&userAttrs).Error
	if err != nil {
		return false, "Failed to retrieve user attributes", err
	}

	userAttrMap := make(map[string]string)
	for _, attr := range userAttrs {
		userAttrMap[attr.Name] = attr.Value
	}

	isAdmin, err := s.HasRole(ctx, request.UserID, "admin")
	if err != nil {
		return false, "Failed to check admin role", err
	}
	if isAdmin {
		return true, "Admin has full access", nil
	}

	switch request.Resource {
	case "task":
		return s.evaluateTaskABACPolicy(ctx, request, userAttrMap)
	case "user", "profile":
		return s.evaluateUserABACPolicy(ctx, request, userAttrMap)
	default:
		return true, "No specific ABAC policy, allowing based on RBAC", nil
	}
}

func (s *AuthorizationServiceImpl) evaluateTaskABACPolicy(ctx context.Context, request AuthorizationRequest, userAttrs map[string]string) (bool, string, error) {
	if request.Action == "create" {
		return true, "Task creation allowed", nil
	}

	if request.ResourceID != nil {
		var task models.Task
		err := s.db.WithContext(ctx).
			Where("id = ?", *request.ResourceID).
			First(&task).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return false, "Task not found", nil
			}
			return false, "Failed to retrieve task", err
		}

		if task.UserID == request.UserID {
			return true, "Task owner has access", nil
		}

		if userDept, exists := userAttrs["department"]; exists {
			var taskOwner models.User
			err := s.db.WithContext(ctx).
				Select("department").
				Where("id = ?", task.UserID).
				First(&taskOwner).Error
			if err == nil && taskOwner.Department == userDept {
				if request.Action == "read" {
					return true, "Department member can view task", nil
				}
			}
		}

		return false, "User can only access own tasks", nil
	}

	return true, "Task listing allowed", nil
}

func (s *AuthorizationServiceImpl) evaluateUserABACPolicy(ctx context.Context, request AuthorizationRequest, userAttrs map[string]string) (bool, string, error) {
	if request.ResourceID != nil && *request.ResourceID == request.UserID {
		return true, "User can access own profile", nil
	}

	if request.ResourceID != nil && *request.ResourceID != request.UserID {
		if request.Action == "update" || request.Action == "delete" {
			return false, "Only admins can modify other users", nil
		}

		if request.Action == "read" {
			return true, "User can view other profiles", nil
		}
	}

	return true, "User listing allowed", nil
}

func (s *AuthorizationServiceImpl) SetUserAttribute(ctx context.Context, userID uuid.UUID, key, value, dataType string) error {
	attr := models.UserAttribute{
		ID:        uuid.Must(uuid.NewV4()),
		UserID:    userID,
		Name:      key,
		Value:     value,
		Type:      dataType,
		Source:    "manual",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.db.WithContext(ctx).
		Save(&attr).Error
}

func (s *AuthorizationServiceImpl) SetResourceAttribute(ctx context.Context, resourceType string, resourceID uuid.UUID, key, value, dataType string) error {
	attr := models.ResourceAttribute{
		ID:           uuid.Must(uuid.NewV4()),
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Name:         key,
		Value:        value,
		Type:         dataType,
		Source:       "manual",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.db.WithContext(ctx).
		Save(&attr).Error
}

func (s *AuthorizationServiceImpl) CreateRole(ctx context.Context, name, description string) (*models.Role, error) {
	role := &models.Role{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        name,
		Description: description,
	}

	err := s.db.WithContext(ctx).Create(role).Error
	return role, err
}

func (s *AuthorizationServiceImpl) CreatePermission(ctx context.Context, resource, action, description string) (*models.Permission, error) {
	permission := &models.Permission{
		ID:          uuid.Must(uuid.NewV4()),
		Name:        resource + ":" + action,
		Resource:    resource,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.db.WithContext(ctx).Create(permission).Error
	return permission, err
}

func (s *AuthorizationServiceImpl) GrantPermissionToRole(ctx context.Context, roleID, permissionID, grantedBy uuid.UUID) error {
	rolePermission := models.RolePermission{
		ID:           uuid.Must(uuid.NewV4()),
		RoleID:       roleID,
		PermissionID: permissionID,
		AssignedBy:   grantedBy,
		AssignedAt:   time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return s.db.WithContext(ctx).Create(&rolePermission).Error
}

func (s *AuthorizationServiceImpl) LogAuthorizationDecision(ctx context.Context, decision AuthorizationDecision) error {
	var resourceID uuid.UUID
	if decision.ResourceID != nil {
		resourceID = *decision.ResourceID
	}

	contextJSON := ""
	if decision.Context != nil {
		if jsonBytes, err := json.Marshal(decision.Context); err == nil {
			contextJSON = string(jsonBytes)
		}
	}

	auditLog := models.AuditLog{
		ID:         uuid.Must(uuid.NewV4()),
		UserID:     decision.UserID,
		Action:     fmt.Sprintf("%s_%s", decision.Action, decision.Resource),
		Resource:   decision.Resource,
		ResourceID: resourceID,
		Decision:   decision.Decision,
		Reason:     decision.Reason,
		IPAddress:  decision.IPAddress,
		UserAgent:  decision.UserAgent,
		Context:    contextJSON,
		Timestamp:  decision.Timestamp,
	}

	return s.db.WithContext(ctx).Create(&auditLog).Error
}
