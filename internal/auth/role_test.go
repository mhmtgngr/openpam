package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// setupRoleTest creates a test database and role repository
func setupRoleTest(t *testing.T) (*sqlx.DB, *RoleRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)
	logger := opamtesting.Logger(t)
	repo := NewRoleRepository(db, logger)

	return db, repo
}

func TestRoleRepository_Create(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		Description: "A test role",
		TenantID:    tenantID,
		IsSystem:    false,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, role.ID)
	assert.False(t, role.CreatedAt.IsZero())
	assert.False(t, role.UpdatedAt.IsZero())
}

func TestRoleRepository_GetByID(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "getbyid-role",
		DisplayName: "Get By ID Role",
		Description: "Test get by ID",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Get role by ID
	found, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, role.ID, found.ID)
	assert.Equal(t, role.Name, found.Name)
	assert.Equal(t, role.DisplayName, found.DisplayName)
}

func TestRoleRepository_GetByID_NotFound(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	randomID := uuid.New()
	_, err := repo.GetByID(ctx, randomID)
	assert.Error(t, err)
}

func TestRoleRepository_GetByName(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "getbyname-role",
		DisplayName: "Get By Name Role",
		Description: "Test get by name",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Get role by name
	found, err := repo.GetByName(ctx, tenantID, "getbyname-role")
	require.NoError(t, err)
	assert.Equal(t, role.ID, found.ID)
	assert.Equal(t, role.Name, found.Name)
}

func TestRoleRepository_GetByName_WrongTenant(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID1, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	tenantID2 := uuid.New()

	role := &Role{
		Name:        "tenant-role",
		DisplayName: "Tenant Role",
		Description: "Test tenant isolation",
		TenantID:    tenantID1,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Try to get with different tenant ID
	_, err = repo.GetByName(ctx, tenantID2, "tenant-role")
	assert.Error(t, err)
}

func TestRoleRepository_List(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple roles
	for i := 0; i < 5; i++ {
		role := &Role{
			Name:        uuid.New().String() + "-role",
			DisplayName: string(rune('A' + i)) + " Role",
			Description: "Test role",
			TenantID:    tenantID,
		}
		require.NoError(t, repo.Create(ctx, role))
	}

	// List roles
	roles, err := repo.List(ctx, tenantID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(roles), 5)
}

func TestRoleRepository_List_Pagination(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple roles
	for i := 0; i < 15; i++ {
		role := &Role{
			Name:        uuid.New().String() + "-role",
			DisplayName: "Paginated Role",
			Description: "Test pagination",
			TenantID:    tenantID,
		}
		require.NoError(t, repo.Create(ctx, role))
	}

	// First page
	page1, err := repo.List(ctx, tenantID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 10)

	// Second page
	page2, err := repo.List(ctx, tenantID, 10, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page2), 5)
}

func TestRoleRepository_Update(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "update-role",
		DisplayName: "Original Name",
		Description: "Original description",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Update role
	role.DisplayName = "Updated Name"
	role.Description = "Updated description"
	err = repo.Update(ctx, role)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.DisplayName)
	assert.Equal(t, "Updated description", updated.Description)
}

func TestRoleRepository_Delete(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "delete-role",
		DisplayName: "Delete Role",
		Description: "To be deleted",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Delete role
	err = repo.Delete(ctx, role.ID)
	require.NoError(t, err)

	// Verify soft delete (role should not be found)
	_, err = repo.GetByID(ctx, role.ID)
	assert.Error(t, err)
}

func TestRoleRepository_GetPermissions_Empty(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "noperm-role",
		DisplayName: "No Permissions",
		Description: "Role with no permissions",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Get permissions (should be empty)
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Empty(t, permissions)
}

func TestRoleRepository_GrantPermission(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "grant-role",
		DisplayName: "Grant Permission",
		Description: "Test granting permissions",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create a permission first
	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "credential", "create", "all", "Create credential")
	require.NoError(t, err)

	// Grant permission to role
	err = repo.GrantPermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Verify permission was granted
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "credential", permissions[0].Resource)
	assert.Equal(t, "create", permissions[0].Action)
}

func TestRoleRepository_RevokePermission(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "revoke-role",
		DisplayName: "Revoke Permission",
		Description: "Test revoking permissions",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create and grant permission
	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "credential", "delete", "all", "Delete credential")
	require.NoError(t, err)

	err = repo.GrantPermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Verify permission exists
	permissions, _ := repo.GetPermissions(ctx, role.ID)
	assert.Len(t, permissions, 1)

	// Revoke permission
	err = repo.RevokePermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Verify permission was revoked
	permissions, err = repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 0)
}

func TestRoleRepository_AssignRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "assign-role",
		DisplayName: "Assign Role",
		Description: "Test role assignment",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create a user
	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "assign@example.com", "Assign", "User", "hash", tenantID)
	require.NoError(t, err)

	// Assign role to user
	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Verify role was assigned
	userRoles, err := repo.GetUserRoles(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 1)
	assert.Equal(t, role.ID, userRoles[0].ID)
}

func TestRoleRepository_RevokeRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "revokeuser-role",
		DisplayName: "Revoke User Role",
		Description: "Test role revocation",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create a user and assign role
	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "revokeuser@example.com", "Revoke", "User", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Verify role was assigned
	userRoles, _ := repo.GetUserRoles(ctx, userID)
	assert.Len(t, userRoles, 1)

	// Revoke role from user
	err = repo.RevokeRole(ctx, userID, role.ID)
	require.NoError(t, err)

	// Verify role was revoked
	userRoles, err = repo.GetUserRoles(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 0)
}

func TestRoleRepository_GetUserRoles(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple roles
	roles := make([]*Role, 3)
	for i := 0; i < 3; i++ {
		role := &Role{
			Name:        uuid.New().String() + "-role",
			DisplayName: "User Role",
			Description: "Test",
			TenantID:    tenantID,
		}
		err := repo.Create(ctx, role)
		require.NoError(t, err)
		roles[i] = role
	}

	// Create a user
	userID := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "multiuser@example.com", "Multi", "Role", "hash", tenantID)
	require.NoError(t, err)

	// Assign multiple roles to user
	assignedBy := uuid.New()
	for _, role := range roles {
		err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
		require.NoError(t, err)
	}

	// Get user roles
	userRoles, err := repo.GetUserRoles(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 3)
}

func TestRoleRepository_GetRoleUsers(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "roleusers-role",
		DisplayName: "Role Users",
		Description: "Test getting role users",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create multiple users and assign them to the role
	assignedBy := uuid.New()
	for i := 0; i < 3; i++ {
		userID := uuid.New()
		_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
			userID, uuid.New().String()+"@example.com", "Role", string(rune('A'+i)), "hash", tenantID)
		require.NoError(t, err)

		err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
		require.NoError(t, err)
	}

	// Get role users
	users, err := repo.GetRoleUsers(ctx, role.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestRoleRepository_GetPermissionByID(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	// Create a permission
	permID := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "session", "approve", "all", "Approve session")
	require.NoError(t, err)

	// Get permission by ID
	permission, err := repo.GetPermissionByID(ctx, permID)
	require.NoError(t, err)
	assert.Equal(t, permID, permission.ID)
	assert.Equal(t, "session", permission.Resource)
	assert.Equal(t, "approve", permission.Action)
}

func TestRoleRepository_GetPermissionByResourceAction(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	// Create a permission
	permID := uuid.New()
	_, err := db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "user", "update", "own", "Update own user")
	require.NoError(t, err)

	// Get permission by resource and action
	permission, err := repo.GetPermissionByResourceAction(ctx, "user", "update")
	require.NoError(t, err)
	assert.Equal(t, permID, permission.ID)
	assert.Equal(t, "user", permission.Resource)
	assert.Equal(t, "update", permission.Action)
	assert.Equal(t, "own", permission.Scope)
}

func TestRoleRepository_ListAllPermissions(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	// Create multiple permissions
	resources := []string{"credential", "session", "user"}
	actions := []string{"create", "read", "update", "delete"}

	for _, res := range resources {
		for _, act := range actions {
			permID := uuid.New()
			_, err := db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
				permID, res, act, "all", res+" "+act)
			require.NoError(t, err)
		}
	}

	// List all permissions
	permissions, err := repo.ListAllPermissions(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(permissions), len(resources)*len(actions))
}

func TestRoleService_CreateRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	role := &Role{
		Name:        "service-create",
		DisplayName: "Service Create Role",
		Description: "Created by service",
		TenantID:    tenantID,
	}

	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}
	err := service.CreateRole(ctx, role, authCtx)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, role.ID)
}

func TestRoleService_CreateRole_Validation(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))
	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	tests := []struct {
		name    string
		role    *Role
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing name",
			role: &Role{
				DisplayName: "No Name",
				TenantID:    tenantID,
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "missing display name",
			role: &Role{
				Name:     "no-display",
				TenantID: tenantID,
			},
			wantErr: true,
			errMsg:  "display name",
		},
		{
			name: "missing tenant ID",
			role: &Role{
				Name:        "no-tenant",
				DisplayName: "No Tenant",
			},
			wantErr: true,
			errMsg:  "tenant",
		},
		{
			name: "duplicate name",
			role: &Role{
				Name:        "duplicate-name",
				DisplayName: "Duplicate Name Role",
				TenantID:    tenantID,
			},
			wantErr: true,
			errMsg:  "already exists",
		},
	}

	// Create the duplicate role first
	duplicateRole := &Role{
		Name:        "duplicate-name",
		DisplayName: "Original Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, duplicateRole))

	// Create auth context for tests
	userID := uuid.New()
	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateRole(ctx, tt.role, authCtx)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRoleService_UpdateRole_SystemRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	role := &Role{
		Name:        "system-role",
		DisplayName: "System Role",
		Description: "System role that cannot be modified",
		TenantID:    tenantID,
		IsSystem:    true,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Try to update system role
	role.DisplayName = "Modified"
	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}
	err = service.UpdateRole(ctx, role, authCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot modify system roles")
}

func TestRoleService_DeleteRole_SystemRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	role := &Role{
		Name:        "delete-system",
		DisplayName: "Delete System Role",
		Description: "System role that cannot be deleted",
		TenantID:    tenantID,
		IsSystem:    true,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Try to delete system role
	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}
	err = service.DeleteRole(ctx, role.ID, authCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete system roles")
}

func TestRoleService_DeleteRole_WithUsers(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	role := &Role{
		Name:        "users-role",
		DisplayName: "Users Role",
		Description: "Role with users",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create a user and assign role
	testUserID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		testUserID, "withrole@example.com", "With", "Role", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, testUserID, role.ID, assignedBy)
	require.NoError(t, err)

	// Try to delete role with users
	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}
	err = service.DeleteRole(ctx, role.ID, authCtx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete role with assigned users")
}

func TestRoleService_AssignPermission(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	userID := uuid.New()
	role := &Role{
		Name:        "assignperm-role",
		DisplayName: "Assign Perm Role",
		Description: "Test permission assignment",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Assign permission (will be created if it doesn't exist)
	authCtx := AuthorizationContext{
		UserID:   userID,
		TenantID: tenantID,
		IsAdmin:  true,
	}
	err = service.AssignPermission(ctx, role.ID, "credential", "create", authCtx)
	require.NoError(t, err)

	// Verify permission was assigned
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Greater(t, len(permissions), 0)

	// Find the credential:create permission
	found := false
	for _, perm := range permissions {
		if perm.Resource == "credential" && (perm.Action == "create" || perm.Action == "*") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find credential:create permission")
}

func TestRoleService_RevokePermission(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "revokeperm-role",
		DisplayName: "Revoke Perm Role",
		Description: "Test permission revocation",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	// Create and grant permission
	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "session", "terminate", "all", "Terminate session")
	require.NoError(t, err)

	err = repo.GrantPermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Revoke permission
	err = service.RevokePermission(ctx, role.ID, "session", "terminate")
	require.NoError(t, err)

	// Verify permission was revoked
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 0)
}

func TestRoleService_CheckPermission(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create role and assign permission
	role := &Role{
		Name:        "checkperm-role",
		DisplayName: "Check Perm Role",
		Description: "Test permission check",
		TenantID:    tenantID,
	}
	err := repo.Create(ctx, role)
	require.NoError(t, err)

	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "audit", "read", "all", "Read audit logs")
	require.NoError(t, err)

	err = repo.GrantPermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Create user and assign role
	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "checkperm@example.com", "Check", "Perm", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Check permission
	hasPermission, err := service.CheckPermission(ctx, userID, "audit", "read")
	require.NoError(t, err)
	assert.True(t, hasPermission)

	// Check non-existent permission
	hasPermission, err = service.CheckPermission(ctx, userID, "audit", "delete")
	require.NoError(t, err)
	assert.False(t, hasPermission)
}

func TestRoleService_AssignUserToRole(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "assignuser-role",
		DisplayName: "Assign User Role",
		Description: "Test user assignment",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "assignuser@example.com", "Assign", "User", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = service.AssignUserToRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Verify assignment
	userRoles, err := repo.GetUserRoles(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 1)
	assert.Equal(t, role.ID, userRoles[0].ID)
}

func TestRoleService_RevokeRoleFromUser(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	role := &Role{
		Name:        "revokefrom-role",
		DisplayName: "Revoke From Role",
		Description: "Test role revocation from user",
		TenantID:    tenantID,
	}

	err := repo.Create(ctx, role)
	require.NoError(t, err)

	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "revokefrom@example.com", "Revoke", "From", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Verify assignment
	userRoles, _ := repo.GetUserRoles(ctx, userID)
	assert.Len(t, userRoles, 1)

	// Revoke role from user
	err = service.RevokeRoleFromUser(ctx, userID, role.ID)
	require.NoError(t, err)

	// Verify revocation
	userRoles, err = repo.GetUserRoles(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 0)
}

func TestRoleService_InitializeSystemRoles(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))
	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Initialize system roles
	err := service.InitializeSystemRoles(ctx, tenantID)
	require.NoError(t, err)

	// Verify all system roles were created
	systemRoles := []string{RoleSuperAdmin, RoleAdmin, RoleOperator, RoleAuditor, RoleRequester}
	for _, roleName := range systemRoles {
		role, err := repo.GetByName(ctx, tenantID, roleName)
		require.NoError(t, err, "System role "+roleName+" should exist")
		assert.True(t, role.IsSystem, "System role "+roleName+" should be marked as system")
	}

	// Running again should not create duplicates
	err = service.InitializeSystemRoles(ctx, tenantID)
	require.NoError(t, err)

	// Verify no duplicates
	roles, _ := repo.List(ctx, tenantID, 100, 0)
	superAdminCount := 0
	for _, r := range roles {
		if r.Name == RoleSuperAdmin {
			superAdminCount++
		}
	}
	assert.Equal(t, 1, superAdminCount, "Should only have one super_admin role")
}

func TestRoleService_CheckPermission_Inherited(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create parent role with permission
	parentRole := &Role{
		Name:        "parent-role",
		DisplayName: "Parent Role",
		Description: "Parent role with permissions",
		TenantID:    tenantID,
	}
	err := repo.Create(ctx, parentRole)
	require.NoError(t, err)

	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "credential", "read", "all", "Read credential")
	require.NoError(t, err)

	err = repo.GrantPermission(ctx, parentRole.ID, permID)
	require.NoError(t, err)

	// Create child role that inherits from parent
	childRole := &Role{
		Name:        "child-role",
		DisplayName: "Child Role",
		Description: "Child role inheriting from parent",
		TenantID:    tenantID,
	}
	err = repo.Create(ctx, childRole)
	require.NoError(t, err)

	childRole.InheritsFrom = &parentRole.ID
	err = repo.Update(ctx, childRole)
	require.NoError(t, err)

	// Create user and assign child role
	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "inherit@example.com", "Inherit", "User", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, childRole.ID, assignedBy)
	require.NoError(t, err)

	// Check inherited permission
	hasPermission, err := service.CheckPermission(ctx, userID, "credential", "read")
	require.NoError(t, err)
	assert.True(t, hasPermission, "User should have inherited permission from parent role")
}

func TestRoleService_CheckPermission_Wildcard(t *testing.T) {
	db, repo := setupRoleTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	service := NewRoleService(repo, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create role with wildcard permission
	role := &Role{
		Name:        "wildcard-role",
		DisplayName: "Wildcard Role",
		Description: "Role with wildcard permission",
		TenantID:    tenantID,
	}
	err := repo.Create(ctx, role)
	require.NoError(t, err)

	permID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO permissions (id, resource, action, scope, description) VALUES ($1, $2, $3, $4, $5)`,
		permID, "session", "*", "all", "All session actions")
	require.NoError(t, err)

	err = repo.GrantPermission(ctx, role.ID, permID)
	require.NoError(t, err)

	// Create user and assign role
	userID := uuid.New()
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, first_name, last_name, password_hash, tenant_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, "wildcard@example.com", "Wildcard", "User", "hash", tenantID)
	require.NoError(t, err)

	assignedBy := uuid.New()
	err = repo.AssignRole(ctx, userID, role.ID, assignedBy)
	require.NoError(t, err)

	// Check various actions - all should return true due to wildcard
	actions := []string{"create", "read", "update", "delete", "approve", "terminate"}
	for _, action := range actions {
		hasPermission, err := service.CheckPermission(ctx, userID, "session", action)
		require.NoError(t, err)
		assert.True(t, hasPermission, "Wildcard should grant "+action+" permission")
	}
}

func TestRoleConstants(t *testing.T) {
	// Verify role constants are set correctly
	assert.Equal(t, "super_admin", RoleSuperAdmin)
	assert.Equal(t, "admin", RoleAdmin)
	assert.Equal(t, "operator", RoleOperator)
	assert.Equal(t, "auditor", RoleAuditor)
	assert.Equal(t, "requester", RoleRequester)
}
