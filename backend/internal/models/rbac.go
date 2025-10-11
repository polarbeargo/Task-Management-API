package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name        string         `json:"name" gorm:"unique;not null"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	CreatedBy  uuid.UUID `json:"created_by" gorm:"type:uuid"`
	ModifiedBy uuid.UUID `json:"modified_by" gorm:"type:uuid"`

	Users       []UserRole       `json:"users,omitempty" gorm:"foreignKey:RoleID"`
	Permissions []RolePermission `json:"permissions,omitempty" gorm:"foreignKey:RoleID"`
}

type Permission struct {
	ID          uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Name        string         `json:"name" gorm:"unique;not null"`
	Resource    string         `json:"resource" gorm:"not null"`
	Action      string         `json:"action" gorm:"not null"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	CreatedBy  uuid.UUID `json:"created_by" gorm:"type:uuid"`
	ModifiedBy uuid.UUID `json:"modified_by" gorm:"type:uuid"`

	Roles []RolePermission `json:"roles,omitempty" gorm:"foreignKey:PermissionID"`
}

type UserRole struct {
	ID         uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID     uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	RoleID     uuid.UUID      `json:"role_id" gorm:"type:uuid;not null"`
	AssignedBy uuid.UUID      `json:"assigned_by" gorm:"type:uuid"`
	AssignedAt time.Time      `json:"assigned_at"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	User           User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role           Role `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	AssignedByUser User `json:"assigned_by_user,omitempty" gorm:"foreignKey:AssignedBy"`
}

type RolePermission struct {
	ID           uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	RoleID       uuid.UUID      `json:"role_id" gorm:"type:uuid;not null"`
	PermissionID uuid.UUID      `json:"permission_id" gorm:"type:uuid;not null"`
	AssignedBy   uuid.UUID      `json:"assigned_by" gorm:"type:uuid"`
	AssignedAt   time.Time      `json:"assigned_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	Role           Role       `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Permission     Permission `json:"permission,omitempty" gorm:"foreignKey:PermissionID"`
	AssignedByUser User       `json:"assigned_by_user,omitempty" gorm:"foreignKey:AssignedBy"`
}

type UserAttribute struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	Name      string         `json:"name" gorm:"not null"`
	Value     string         `json:"value" gorm:"not null"`
	Type      string         `json:"type" gorm:"not null"`
	Source    string         `json:"source"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type ResourceAttribute struct {
	ID           uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ResourceType string         `json:"resource_type" gorm:"not null"`
	ResourceID   uuid.UUID      `json:"resource_id" gorm:"type:uuid;not null"`
	Name         string         `json:"name" gorm:"not null"`
	Value        string         `json:"value" gorm:"not null"`
	Type         string         `json:"type" gorm:"not null"`
	Source       string         `json:"source"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

type AuditLog struct {
	ID            uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID        uuid.UUID `json:"user_id" gorm:"type:uuid"`
	Action        string    `json:"action" gorm:"not null"`
	Resource      string    `json:"resource" gorm:"not null"`
	ResourceID    uuid.UUID `json:"resource_id" gorm:"type:uuid"`
	Decision      string    `json:"decision" gorm:"not null"`
	Reason        string    `json:"reason"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	RequestMethod string    `json:"request_method"`
	RequestPath   string    `json:"request_path"`
	Context       string    `json:"context" gorm:"type:text"`
	Timestamp     time.Time `json:"timestamp"`

	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

type AuthorizationRequest struct {
	UserID             uuid.UUID              `json:"user_id"`
	Action             string                 `json:"action"`
	Resource           string                 `json:"resource"`
	ResourceID         uuid.UUID              `json:"resource_id,omitempty"`
	UserAttributes     map[string]string      `json:"user_attributes,omitempty"`
	ResourceAttributes map[string]string      `json:"resource_attributes,omitempty"`
	Context            map[string]interface{} `json:"context,omitempty"`
}

type AuthorizationDecision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func (ur *UserRole) IsExpired() bool {
	return ur.ExpiresAt != nil && time.Now().After(*ur.ExpiresAt)
}

func (ua *UserAttribute) IsExpired() bool {
	return ua.ExpiresAt != nil && time.Now().After(*ua.ExpiresAt)
}

func (ua *UserAttribute) GetTypedValue() interface{} {
	switch ua.Type {
	case "boolean":
		return ua.Value == "true"
	case "number":
		if val := ua.Value; val != "" {
			return val
		}
		return 0
	case "date":
		if t, err := time.Parse(time.RFC3339, ua.Value); err == nil {
			return t
		}
		return nil
	default:
		return ua.Value
	}
}

func (ra *ResourceAttribute) IsExpired() bool {
	return ra.ExpiresAt != nil && time.Now().After(*ra.ExpiresAt)
}

func (ra *ResourceAttribute) GetTypedValue() interface{} {
	switch ra.Type {
	case "boolean":
		return ra.Value == "true"
	case "number":
		if val := ra.Value; val != "" {
			return val
		}
		return 0
	case "date":
		if t, err := time.Parse(time.RFC3339, ra.Value); err == nil {
			return t
		}
		return nil
	default:
		return ra.Value
	}
}
