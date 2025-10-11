package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID       uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Username string    `json:"username" gorm:"unique"`
	Email    string    `json:"email" gorm:"unique;not null"`
	Password string    `json:"password" gorm:"not null"`

	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Department  string     `json:"department"`
	Position    string     `json:"position"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt *time.Time `json:"last_login_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Roles      []Role          `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	UserRoles  []UserRole      `json:"user_roles,omitempty" gorm:"foreignKey:UserID"`
	Attributes []UserAttribute `json:"attributes,omitempty" gorm:"foreignKey:UserID"`

	Tasks []Task `json:"tasks,omitempty" gorm:"foreignKey:UserID"`

	AuditLogs []AuditLog `json:"audit_logs,omitempty" gorm:"foreignKey:UserID"`
}

func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

func (u *User) HasPermission(resource, action string) bool {
	for _, role := range u.Roles {
		for _, rolePermission := range role.Permissions {
			if rolePermission.Permission.Resource == resource && rolePermission.Permission.Action == action {
				return true
			}
		}
	}
	return false
}

func (u *User) GetRoleNames() []string {
	var roleNames []string
	for _, role := range u.Roles {
		roleNames = append(roleNames, role.Name)
	}
	return roleNames
}

func (u *User) GetPermissions() []Permission {
	var permissions []Permission
	permissionMap := make(map[uuid.UUID]Permission)

	for _, role := range u.Roles {
		for _, rolePermission := range role.Permissions {
			permissionMap[rolePermission.Permission.ID] = rolePermission.Permission
		}
	}

	for _, permission := range permissionMap {
		permissions = append(permissions, permission)
	}

	return permissions
}

func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}

func (u *User) IsUser() bool {
	return u.HasRole("user")
}
