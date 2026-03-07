package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// RoleRepositoryInterface defines the contract for role data operations.
// Implement this interface to swap storage backends (e.g., for testing or
// migrating from PostgreSQL to another datastore).
type RoleRepositoryInterface interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*Role, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error

	GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	GrantPermission(ctx context.Context, roleID, permissionID uuid.UUID) error
	RevokePermission(ctx context.Context, roleID, permissionID uuid.UUID) error

	AssignRole(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error
	RevokeRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)
	GetRoleUsers(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]User, error)

	GetPermissionByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	GetPermissionByResourceAction(ctx context.Context, resource, action string) (*Permission, error)
	ListAllPermissions(ctx context.Context) ([]Permission, error)

	CheckTenantAccess(ctx context.Context, roleID, tenantID uuid.UUID) error
	CheckAdminAccess(ctx context.Context, userID, tenantID uuid.UUID) (bool, error)
}

// RoleRepository handles role data operations using PostgreSQL.
type RoleRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *sqlx.DB, logger zerolog.Logger) *RoleRepository {
	return &RoleRepository{db: db, logger: logger}
}

// CheckTenantAccess verifies that the role belongs to the same tenant
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
	query := `SELECT * FROM roles WHERE name = $1 AND tenant_id = $2 AND deleted_at IS NULL`
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
