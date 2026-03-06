package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RoleService handles role business logic. It depends on the RoleRepositoryInterface
// for data access, making it pluggable — swap the repository implementation for
// testing or to use a different storage backend.
type RoleService struct {
	repo   RoleRepositoryInterface
	logger zerolog.Logger
}

// NewRoleService creates a new role service
func NewRoleService(repo RoleRepositoryInterface, logger zerolog.Logger) *RoleService {
	return &RoleService{repo: repo, logger: logger}
}

// CreateRole creates a new role with authorization check
func (s *RoleService) CreateRole(ctx context.Context, role *Role, authCtx AuthorizationContext) error {
	if role.TenantID != authCtx.TenantID {
		return fmt.Errorf("role: unauthorized - cannot create role for different tenant")
	}

	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	if err := s.validateRole(role); err != nil {
		return err
	}

	existing, err := s.repo.GetByName(ctx, role.TenantID, role.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("role: name already exists")
	}

	return s.repo.Create(ctx, role)
}

// UpdateRole updates an existing role with authorization check
func (s *RoleService) UpdateRole(ctx context.Context, role *Role, authCtx AuthorizationContext) error {
	if err := s.repo.CheckTenantAccess(ctx, role.ID, authCtx.TenantID); err != nil {
		return err
	}

	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	if err := s.validateRole(role); err != nil {
		return err
	}

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
	if err := s.repo.CheckTenantAccess(ctx, roleID, authCtx.TenantID); err != nil {
		return err
	}

	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsSystem {
		return fmt.Errorf("role: cannot delete system roles")
	}

	users, err := s.repo.GetRoleUsers(ctx, roleID, 1, 0)
	if err == nil && len(users) > 0 {
		return fmt.Errorf("role: cannot delete role with assigned users")
	}

	return s.repo.Delete(ctx, roleID)
}

// AssignPermission assigns a permission to a role with authorization check
func (s *RoleService) AssignPermission(ctx context.Context, roleID uuid.UUID, resource, action string, authCtx AuthorizationContext) error {
	if err := s.repo.CheckTenantAccess(ctx, roleID, authCtx.TenantID); err != nil {
		return err
	}

	isAdmin, err := s.repo.CheckAdminAccess(ctx, authCtx.UserID, authCtx.TenantID)
	if err != nil || !isAdmin {
		return fmt.Errorf("role: unauthorized - admin access required")
	}

	role, err := s.repo.GetByID(ctx, roleID)
	if err != nil {
		return err
	}

	permission, err := s.repo.GetPermissionByResourceAction(ctx, resource, action)
	if err != nil {
		permission = &Permission{
			ID:       uuid.New(),
			Resource: resource,
			Action:   action,
			Scope:    "all",
		}
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

// CheckPermission checks if a user has a specific permission (including inherited)
func (s *RoleService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		if s.checkRolePermission(ctx, role, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

func (s *RoleService) checkRolePermission(ctx context.Context, role Role, resource, action string) bool {
	permissions, err := s.repo.GetPermissions(ctx, role.ID)
	if err != nil {
		return false
	}

	for _, perm := range permissions {
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
			return true
		}
	}

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

// GetRepo returns the role repository
func (s *RoleService) GetRepo() RoleRepositoryInterface {
	return s.repo
}

// InitializeSystemRoles creates predefined system roles for a tenant
func (s *RoleService) InitializeSystemRoles(ctx context.Context, tenantID uuid.UUID) error {
	roles := []Role{
		{Name: RoleSuperAdmin, DisplayName: "Super Administrator", Description: "Full system access across all tenants", IsSystem: true, TenantID: tenantID},
		{Name: RoleAdmin, DisplayName: "Administrator", Description: "Full access within tenant", IsSystem: true, TenantID: tenantID},
		{Name: RoleOperator, DisplayName: "Operator", Description: "Can manage credentials and sessions", IsSystem: true, TenantID: tenantID},
		{Name: RoleAuditor, DisplayName: "Auditor", Description: "Read-only access for auditing", IsSystem: true, TenantID: tenantID},
		{Name: RoleRequester, DisplayName: "Requester", Description: "Can request access to credentials", IsSystem: true, TenantID: tenantID},
	}

	for _, role := range roles {
		_, err := s.repo.GetByName(ctx, tenantID, role.Name)
		if err == nil {
			continue
		}
		if err := s.repo.Create(ctx, &role); err != nil {
			return fmt.Errorf("initialize role %s: %w", role.Name, err)
		}
	}

	return nil
}

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
