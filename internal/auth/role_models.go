package auth

import (
	"time"

	"github.com/google/uuid"
)

// Role represents a role with permissions
type Role struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	DisplayName  string     `db:"display_name" json:"display_name"`
	Description  string     `db:"description" json:"description"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	IsSystem     bool       `db:"is_system" json:"is_system"`
	InheritsFrom *uuid.UUID `db:"inherits_from_id" json:"inherits_from,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Permission represents a specific permission
type Permission struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Resource    string    `db:"resource" json:"resource"`
	Action      string    `db:"action" json:"action"`
	Scope       string    `db:"scope" json:"scope"`
	Description string    `db:"description" json:"description"`
}

// RolePermission links a role to a permission
type RolePermission struct {
	RoleID       uuid.UUID `db:"role_id" json:"role_id"`
	PermissionID uuid.UUID `db:"permission_id" json:"permission_id"`
}

// UserRole links a user to a role
type UserRole struct {
	UserID     uuid.UUID `db:"user_id" json:"user_id"`
	RoleID     uuid.UUID `db:"role_id" json:"role_id"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
	AssignedBy uuid.UUID `db:"assigned_by" json:"assigned_by"`
}

// AuthorizationContext holds context for authorization checks
type AuthorizationContext struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	IsAdmin  bool
}

// Predefined system roles
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
	RoleAuditor    = "auditor"
	RoleRequester  = "requester"
)
