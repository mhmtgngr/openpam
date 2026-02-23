package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Role represents a role with permissions
type Role struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	DisplayName string     `db:"display_name" json:"display_name"`
	Description string     `db:"description" json:"description"`
	TenantID    uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	IsSystem    bool       `db:"is_system" json:"is_system"` // Predefined roles
	InheritsFrom *uuid.UUID `db:"inherits_from_id" json:"inherits_from,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Permission represents a specific permission
type Permission struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Resource    string    `db:"resource" json:"resource"`     // e.g., "credential", "session", "user"
	Action      string    `db:"action" json:"action"`         // e.g., "create", "read", "update", "delete", "approve"
	Scope       string    `db:"scope" json:"scope"`           // e.g., "all", "own", "team"
	Description string    `db:"description" json:"description"`
}

// RolePermission links a role to a permission
type RolePermission struct {
	RoleID       uuid.UUID `db:"role_id" json:"role_id"`
	PermissionID uuid.UUID `db:"permission_id" json:"permission_id"`
}

// UserRole links a user to a role
type UserRole struct {
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	RoleID    uuid.UUID `db:"role_id" json:"role_id"`
	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
	AssignedBy uuid.UUID `db:"assigned_by" json:"assigned_by"`
}

// RoleRepository handles role data operations
type RoleRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sqlx.DB, logger zerolog.Logger) *RoleRepository {
	return &RoleRepository{db: db, logger: logger}
}

// CheckTenantAccess verifies that the requesting user has access to the tenant
// and that the role belongs to the same tenant
func (r *RoleRepository) CheckTenantAccess(ctx context.Context, roleID, tenantID uuid.UUID) error {
	var roleTenantID uuid.UUID
	query := `SELECT tenant_id FROM roles WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &roleTenantID, query, roleID)
	if err != nil {
		return fmt.Errorf("role.CheckTenantAccess: role not found")
	}

	if roleTenantID != tenantID {
		return fmt.Errorf("role.CheckTenantAccess: unauthorized access to role from different tenant")
	}

	return nil
}

// CheckAdminAccess verifies the user has admin or super_admin role within the tenant
func (r *RoleRepository) CheckAdminAccess(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.tenant_id = $2
		AND r.name IN ('admin', 'super_admin')
		AND r.deleted_at IS NULL
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, userID, tenantID)
	if err != nil {
		return false, fmt.Errorf("role.CheckAdminAccess: %w", err)
	}
	return count > 0, nil
}

// Create creates a new role
func (r *RoleRepository) Create(ctx context.Context, role *Role) error {
	role.ID = uuid.New()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	query := `
		INSERT INTO roles (id, name, display_name, description, tenant_id, is_system,
			inherits_from_id, created_at, updated_at)
		VALUES (:id, :name, :display_name, :description, :tenant_id, :is_system,
			:inherits_from_id, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, role)
	if err != nil {
		return fmt.Errorf("role.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a role by ID
func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	var role Role
	query := `SELECT * FROM roles WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &role, query, id)
	if err != nil {
		return nil, fmt.Errorf("role.GetByID: %w", err)
	}
	return &role, nil
}

// GetByName retrieves a role by name
func (r *RoleRepository) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error) {
	var role Role
	query := `
		SELECT * FROM roles
		WHERE name = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &role, query, name, tenantID)
	if err != nil {
		return nil, fmt.Errorf("role.GetByName: %w", err)
	}
	return &role, nil
}

// List retrieves roles with pagination
func (r *RoleRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Role, error) {
	var roles []Role
	query := `
		SELECT * FROM roles
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &roles, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("role.List: %w", err)
	}
	return roles, nil
}

// Update updates a role
func (r *RoleRepository) Update(ctx context.Context, role *Role) error {
	role.UpdatedAt = time.Now()

	query := `
		UPDATE roles SET
			name = :name,
			display_name = :display_name,
			description = :description,
			inherits_from_id = :inherits_from_id,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	_, err := r.db.NamedExecContext(ctx, query, role)
	if err != nil {
		return fmt.Errorf("role.Update: %w", err)
	}

	return nil
}

// Delete soft deletes a role
func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE roles SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("role.Delete: %w", err)
	}
	return nil
}

// GetPermissions retrieves all permissions for a role
func (r *RoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	var permissions []Permission
	query := `
		SELECT p.* FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
	`
	err := r.db.SelectContext(ctx, &permissions, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("role.GetPermissions: %w", err)
	}
	return permissions, nil
}

// GrantPermission grants a permission to a role
func (r *RoleRepository) GrantPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `
		INSERT INTO role_permissions (role_id, permission_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("role.GrantPermission: %w", err)
	}
	return nil
}

// RevokePermission revokes a permission from a role
func (r *RoleRepository) RevokePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	query := `DELETE FROM role_permissions WHERE role_id = $1 AND permission_id = $2`
	_, err := r.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("role.RevokePermission: %w", err)
	}
	return nil
}

// AssignRole assigns a role to a user
func (r *RoleRepository) AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_at, assigned_by)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (user_id, role_id) DO UPDATE SET
			assigned_at = NOW(),
			assigned_by = $3
	`
	_, err := r.db.ExecContext(ctx, query, userID, roleID, assignedBy)
	if err != nil {
		return fmt.Errorf("role.AssignRole: %w", err)
	}
	return nil
}

// RevokeRole revokes a role from a user
func (r *RoleRepository) RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("role.RevokeRole: %w", err)
	}
	return nil
}

// GetUserRoles retrieves all roles for a user
func (r *RoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	var roles []Role
	query := `
		SELECT r.* FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
	`
	err := r.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("role.GetUserRoles: %w", err)
	}
	return roles, nil
}

// GetRoleUsers retrieves all users with a specific role
func (r *RoleRepository) GetRoleUsers(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]User, error) {
	var users []User
	query := `
		SELECT u.* FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_id = $1 AND u.deleted_at IS NULL
		ORDER BY u.created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &users, query, roleID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("role.GetRoleUsers: %w", err)
	}
	return users, nil
}

// GetPermissionByID retrieves a permission by ID
func (r *RoleRepository) GetPermissionByID(ctx context.Context, id uuid.UUID) (*Permission, error) {
	var permission Permission
	query := `SELECT * FROM permissions WHERE id = $1`
	err := r.db.GetContext(ctx, &permission, query, id)
	if err != nil {
		return nil, fmt.Errorf("role.GetPermissionByID: %w", err)
	}
	return &permission, nil
}

// GetPermissionByResourceAction retrieves a permission by resource and action
func (r *RoleRepository) GetPermissionByResourceAction(ctx context.Context, resource, action string) (*Permission, error) {
	var permission Permission
	query := `SELECT * FROM permissions WHERE resource = $1 AND action = $2`
	err := r.db.GetContext(ctx, &permission, query, resource, action)
	if err != nil {
		return nil, fmt.Errorf("role.GetPermissionByResourceAction: %w", err)
	}
	return &permission, nil
}

// ListAllPermissions retrieves all available permissions
func (r *RoleRepository) ListAllPermissions(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	query := `SELECT * FROM permissions ORDER BY resource, action`
	err := r.db.SelectContext(ctx, &permissions, query)
	if err != nil {
		return nil, fmt.Errorf("role.ListAllPermissions: %w", err)
	}
	return permissions, nil
}

// RoleService handles role business logic
type RoleService struct {
	repo   *RoleRepository
	logger zerolog.Logger
}

// NewRoleService creates a new role service
func NewRoleService(repo *RoleRepository, logger zerolog.Logger) *RoleService {
	return &RoleService{repo: repo, logger: logger}
}

// AuthorizationContext holds context for authorization checks
type AuthorizationContext struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	IsAdmin  bool
}

// CreateRole creates a new role with authorization check
func (s *RoleService) CreateRole(ctx context.Context, role *Role, authCtx AuthorizationContext) error {
	// Authorization: user must be from the same tenant
	if role.TenantID != authCtx.TenantID {
		return fmt.Errorf("role: unauthorized - cannot create role for different tenant")
	}

	// Only admins can create roles
	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	// Validate role
	if err := s.validateRole(role); err != nil {
		return err
	}

	// Check if role name already exists
	existing, err := s.repo.GetByName(ctx, role.TenantID, role.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("role: name already exists")
	}

	return s.repo.Create(ctx, role)
}

// CreateRoleLegacy creates a new role (legacy method for backward compatibility)
func (s *RoleService) CreateRoleLegacy(ctx context.Context, role *Role) error {
	// Validate role
	if err := s.validateRole(role); err != nil {
		return err
	}

	// Check if role name already exists
	existing, err := s.repo.GetByName(ctx, role.TenantID, role.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("role: name already exists")
	}

	return s.repo.Create(ctx, role)
}

// UpdateRole updates an existing role with authorization check
func (s *RoleService) UpdateRole(ctx context.Context, role *Role, authCtx AuthorizationContext) error {
	// Authorization: check tenant access
	if err := s.repo.CheckTenantAccess(ctx, role.ID, authCtx.TenantID); err != nil {
		return err
	}

	// Only admins can update roles
	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	// Validate role
	if err := s.validateRole(role); err != nil {
		return err
	}

	// Don't allow modifying system roles
	existing, err := s.repo.GetByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("role: cannot modify system roles")
	}

	return s.repo.Update(ctx, role)
}

// UpdateRoleLegacy updates an existing role (legacy method)
func (s *RoleService) UpdateRoleLegacy(ctx context.Context, role *Role) error {
	// Validate role
	if err := s.validateRole(role); err != nil {
		return err
	}

	// Don't allow modifying system roles
	existing, err := s.repo.GetByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("role: cannot modify system roles")
	}

	return s.repo.Update(ctx, role)
}

// DeleteRole deletes a role with authorization check
func (s *RoleService) DeleteRole(ctx context.Context, roleID uuid.UUID, authCtx AuthorizationContext) error {
	// Authorization: check tenant access
	if err := s.repo.CheckTenantAccess(ctx, roleID, authCtx.TenantID); err != nil {
		return err
	}

	// Only admins can delete roles
	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	// Don't allow deleting system roles
	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return fmt.Errorf("role: cannot delete system roles")
	}

	// Check if role has users
	users, err := s.repo.GetRoleUsers(ctx, roleID, 1, 0)
	if err == nil && len(users) > 0 {
		return fmt.Errorf("role: cannot delete role with assigned users")
	}

	return s.repo.Delete(ctx, roleID)
}

// DeleteRoleLegacy deletes a role (legacy method)
func (s *RoleService) DeleteRoleLegacy(ctx context.Context, roleID uuid.UUID) error {
	// Don't allow deleting system roles
	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return fmt.Errorf("role: cannot delete system roles")
	}

	// Check if role has users
	users, err := s.repo.GetRoleUsers(ctx, roleID, 1, 0)
	if err == nil && len(users) > 0 {
		return fmt.Errorf("role: cannot delete role with assigned users")
	}

	return s.repo.Delete(ctx, roleID)
}

// AssignPermission assigns a permission to a role with authorization check
func (s *RoleService) AssignPermission(ctx context.Context, roleID uuid.UUID, resource, action string, authCtx AuthorizationContext) error {
	// Authorization: check tenant access
	if err := s.repo.CheckTenantAccess(ctx, roleID, authCtx.TenantID); err != nil {
		return err
	}

	// Only admins can assign permissions
	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	// Get role
	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	// Get permission
	permission, err := s.repo.GetPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		// Create permission if it doesn't exist
		permission = &Permission{
			ID:       uuid.New(),
			Resource: resource,
			Action:   action,
			Scope:    "all",
		}
		// Note: You'd want to create it in DB here
	}

	return s.repo.GrantPermission(ctx, role.ID, permission.ID)
}

// AssignPermissionLegacy assigns a permission (legacy method)
func (s *RoleService) AssignPermissionLegacy(ctx context.Context, roleID uuid.UUID, resource, action string) error {
	// Get role
	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	// Get permission
	permission, err := s.repo.GetPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		// Create permission if it doesn't exist
		permission = &Permission{
			ID:       uuid.New(),
			Resource: resource,
			Action:   action,
			Scope:    "all",
		}
		// Note: You'd want to create it in DB here
	}

	return s.repo.GrantPermission(ctx, role.ID, permission.ID)
}

// RevokePermission revokes a permission from a role
func (s *RoleService) RevokePermission(ctx context.Context, roleID uuid.UUID, resource, action string) error {
	permission, err := s.repo.GetPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		return err
	}

	return s.repo.RevokePermission(ctx, roleID, permission.ID)
}

// CheckPermission checks if a user has a specific permission
func (s *RoleService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check each role (including inherited)
	for _, role := range roles {
		if s.checkRolePermission(ctx, role, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

// checkRolePermission checks if a role has a specific permission, including inherited roles
func (s *RoleService) checkRolePermission(ctx context.Context, role Role, resource, action string) bool {
	// Get direct permissions
	permissions, err := s.repo.GetPermissions(ctx, role.ID)
	if err != nil {
		return false
	}

	// Check direct permissions
	for _, perm := range permissions {
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
			return true
		}
	}

	// Check inherited permissions
	if role.InheritsFrom != nil {
		parentRole, err := s.repo.GetByID(ctx, *role.InheritsFrom)
		if err == nil {
			return s.checkRolePermission(ctx, *parentRole, resource, action)
		}
	}

	return false
}

// AssignUserToRole assigns a role to a user
func (s *RoleService) AssignUserToRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error {
	// Verify role exists
	_, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	return s.repo.AssignRole(ctx, userID, roleID, assignedBy)
}

// RevokeRoleFromUser revokes a role from a user
func (s *RoleService) RevokeRoleFromUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return s.repo.RevokeRole(ctx, userID, roleID)
}

// validateRole validates role data
func (s *RoleService) validateRole(role *Role) error {
	if role.Name == "" {
		return fmt.Errorf("role: name is required")
	}
	if role.DisplayName == "" {
		return fmt.Errorf("role: display name is required")
	}
	if role.TenantID == uuid.Nil {
		return fmt.Errorf("role: tenant ID is required")
	}
	return nil
}

// GetRepo returns the role repository
func (s *RoleService) GetRepo() *RoleRepository {
	return s.repo
}

// Predefined system roles
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleOperator   = "operator"
	RoleAuditor    = "auditor"
	RoleRequester  = "requester"
)

// InitializeSystemRoles creates predefined system roles for a tenant
func (s *RoleService) InitializeSystemRoles(ctx context.Context, tenantID uuid.UUID) error {
	roles := []Role{
		{
			Name:        RoleSuperAdmin,
			DisplayName: "Super Administrator",
			Description: "Full system access across all tenants",
			IsSystem:    true,
			TenantID:    tenantID,
		},
		{
			Name:        RoleAdmin,
			DisplayName: "Administrator",
			Description: "Full access within tenant",
			IsSystem:    true,
			TenantID:    tenantID,
		},
		{
			Name:        RoleOperator,
			DisplayName: "Operator",
			Description: "Can manage credentials and sessions",
			IsSystem:    true,
			TenantID:    tenantID,
		},
		{
			Name:        RoleAuditor,
			DisplayName: "Auditor",
			Description: "Read-only access for auditing",
			IsSystem:    true,
			TenantID:    tenantID,
		},
		{
			Name:        RoleRequester,
			DisplayName: "Requester",
			Description: "Can request access to credentials",
			IsSystem:    true,
			TenantID:    tenantID,
		},
	}

	for _, role := range roles {
		_, err := s.repo.GetByName(ctx, tenantID, role.Name)
		if err == nil {
			continue // Role already exists
		}
		if err := s.repo.Create(ctx, &role); err != nil {
			return fmt.Errorf("initialize role %s: %w", role.Name, err)
		}
	}

	return nil
}
