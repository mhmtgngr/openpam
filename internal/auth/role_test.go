package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/testing"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRoleTestDB(t *testing.T) *sqlx.DB {
	db := testing.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil
	}
	testing.SetupTestDatabase(t, db)
	return db
}

func TestRoleRepository_Create(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

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
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		Description: "A test role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	t.Run("get existing role", func(t *testing.T) {
		found, err := repo.GetByID(ctx, role.ID)
		require.NoError(t, err)
		assert.Equal(t, role.ID, found.ID)
		assert.Equal(t, role.Name, found.Name)
		assert.Equal(t, role.DisplayName, found.DisplayName)
	})

	t.Run("get non-existent role", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestRoleRepository_GetByName(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	t.Run("get existing role by name", func(t *testing.T) {
		found, err := repo.GetByName(ctx, tenantID, "test-role")
		require.NoError(t, err)
		assert.Equal(t, role.ID, found.ID)
		assert.Equal(t, role.Name, found.Name)
	})

	t.Run("get role from different tenant", func(t *testing.T) {
		otherTenantID := uuid.New()
		_, err := repo.GetByName(ctx, otherTenantID, "test-role")
		assert.Error(t, err)
	})

	t.Run("get non-existent name", func(t *testing.T) {
		_, err := repo.GetByName(ctx, tenantID, "nonexistent")
		assert.Error(t, err)
	})
}

func TestRoleRepository_List(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	// Create multiple roles
	for i := 0; i < 5; i++ {
		role := &Role{
			Name:        fmt.Sprintf("role-%d", i),
			DisplayName: fmt.Sprintf("Role %d", i),
			TenantID:    tenantID,
		}
		require.NoError(t, repo.Create(ctx, role))
	}

	t.Run("list roles", func(t *testing.T) {
		roles, err := repo.List(ctx, tenantID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, roles, 5)
	})

	t.Run("list with limit", func(t *testing.T) {
		roles, err := repo.List(ctx, tenantID, 3, 0)
		require.NoError(t, err)
		assert.Len(t, roles, 3)
	})

	t.Run("list with offset", func(t *testing.T) {
		roles, err := repo.List(ctx, tenantID, 10, 2)
		require.NoError(t, err)
		assert.Len(t, roles, 3)
	})
}

func TestRoleRepository_Update(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		Description: "Original description",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	// Update role
	role.DisplayName = "Updated Role"
	role.Description = "Updated description"
	err := repo.Update(ctx, role)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Role", updated.DisplayName)
	assert.Equal(t, "Updated description", updated.Description)
}

func TestRoleRepository_Delete(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	err := repo.Delete(ctx, role.ID)
	require.NoError(t, err)

	// Verify soft delete - role should not be found
	_, err = repo.GetByID(ctx, role.ID)
	assert.Error(t, err)
}

func TestRoleRepository_GrantPermission(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	// First, create a permission in the database
	permissionID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO permissions (id, resource, action, scope, description)
		VALUES ($1, $2, $3, $4, $5)
	`, permissionID, "credential", "read", "all", "Read credentials")
	require.NoError(t, err)

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	err = repo.GrantPermission(ctx, role.ID, permissionID)
	require.NoError(t, err)

	// Verify permission granted
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "credential", permissions[0].Resource)
	assert.Equal(t, "read", permissions[0].Action)
}

func TestRoleRepository_RevokePermission(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	// Create permission
	permissionID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO permissions (id, resource, action, scope, description)
		VALUES ($1, $2, $3, $4, $5)
	`, permissionID, "credential", "read", "all", "Read credentials")
	require.NoError(t, err)

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))
	require.NoError(t, repo.GrantPermission(ctx, role.ID, permissionID))

	// Revoke permission
	err = repo.RevokePermission(ctx, role.ID, permissionID)
	require.NoError(t, err)

	// Verify permission revoked
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 0)
}

func TestRoleRepository_GetPermissions(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	// Create permissions
	perm1ID := uuid.New()
	perm2ID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO permissions (id, resource, action, scope, description)
		VALUES ($1, $2, $3, $4, $5), ($6, $7, $8, $9, $10)
	`, perm1ID, "credential", "read", "all", "Read credentials",
		perm2ID, "credential", "write", "all", "Write credentials")
	require.NoError(t, err)

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	// Grant both permissions
	require.NoError(t, repo.GrantPermission(ctx, role.ID, perm1ID))
	require.NoError(t, repo.GrantPermission(ctx, role.ID, perm2ID))

	// Get permissions
	permissions, err := repo.GetPermissions(ctx, role.ID)
	require.NoError(t, err)
	assert.Len(t, permissions, 2)
}

func TestRoleRepository_AssignRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	// Create user
	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	userRepo := NewUserRepository(db, zerolog.Nop())
	require.NoError(t, userRepo.Create(ctx, user))

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	err := repo.AssignRole(ctx, user.ID, role.ID, assignedBy)
	require.NoError(t, err)

	// Verify assignment
	userRoles, err := repo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 1)
	assert.Equal(t, role.ID, userRoles[0].ID)
}

func TestRoleRepository_RevokeRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	// Create user
	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	userRepo := NewUserRepository(db, zerolog.Nop())
	require.NoError(t, userRepo.Create(ctx, user))

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))
	require.NoError(t, repo.AssignRole(ctx, user.ID, role.ID, assignedBy))

	// Revoke role
	err := repo.RevokeRole(ctx, user.ID, role.ID)
	require.NoError(t, err)

	// Verify revocation
	userRoles, err := repo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 0)
}

func TestRoleRepository_GetUserRoles(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	// Create user
	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	userRepo := NewUserRepository(db, zerolog.Nop())
	require.NoError(t, userRepo.Create(ctx, user))

	// Create multiple roles
	for i := 0; i < 3; i++ {
		role := &Role{
			Name:        fmt.Sprintf("role-%d", i),
			DisplayName: fmt.Sprintf("Role %d", i),
			TenantID:    tenantID,
		}
		require.NoError(t, repo.Create(ctx, role))
		require.NoError(t, repo.AssignRole(ctx, user.ID, role.ID, assignedBy))
	}

	// Get user roles
	userRoles, err := repo.GetUserRoles(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 3)
}

func TestRoleRepository_GetRoleUsers(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	// Create multiple users and assign them to the role
	for i := 0; i < 3; i++ {
		user := &User{
			Email:     fmt.Sprintf("user%d@example.com", i),
			FirstName: "User",
			LastName:  fmt.Sprintf("%d", i),
			TenantID:  tenantID,
		}
		userRepo := NewUserRepository(db, zerolog.Nop())
		require.NoError(t, userRepo.Create(ctx, user))
		require.NoError(t, repo.AssignRole(ctx, user.ID, role.ID, assignedBy))
	}

	// Get role users
	users, err := repo.GetRoleUsers(ctx, role.ID, 10, 0)
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestRoleService_CreateRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		Description: "A test role",
		TenantID:    tenantID,
	}

	err := service.CreateRole(ctx, role)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, role.ID)
}

func TestRoleService_CreateRole_DuplicateName(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role1 := &Role{
		Name:        "test-role",
		DisplayName: "Test Role 1",
		TenantID:    tenantID,
	}
	require.NoError(t, service.CreateRole(ctx, role1))

	role2 := &Role{
		Name:        "test-role",
		DisplayName: "Test Role 2",
		TenantID:    tenantID,
	}
	err := service.CreateRole(ctx, role2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name already exists")
}

func TestRoleService_ValidateRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	t.Run("valid role", func(t *testing.T) {
		role := &Role{
			Name:        "test-role",
			DisplayName: "Test Role",
			TenantID:    tenantID,
		}
		err := service.validateRole(role)
		assert.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		role := &Role{
			DisplayName: "Test Role",
			TenantID:    tenantID,
		}
		err := service.validateRole(role)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing display name", func(t *testing.T) {
		role := &Role{
			Name:     "test-role",
			TenantID: tenantID,
		}
		err := service.validateRole(role)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "display name")
	})

	t.Run("missing tenant ID", func(t *testing.T) {
		role := &Role{
			Name:        "test-role",
			DisplayName: "Test Role",
		}
		err := service.validateRole(role)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant")
	})
}

func TestRoleService_UpdateRole_SystemRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "system-role",
		DisplayName: "System Role",
		TenantID:    tenantID,
		IsSystem:    true,
	}
	require.NoError(t, repo.Create(ctx, role))

	role.DisplayName = "Updated"
	err := service.UpdateRole(ctx, role)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot modify system roles")
}

func TestRoleService_DeleteRole_SystemRole(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	role := &Role{
		Name:        "system-role",
		DisplayName: "System Role",
		TenantID:    tenantID,
		IsSystem:    true,
	}
	require.NoError(t, repo.Create(ctx, role))

	err := service.DeleteRole(ctx, role.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete system roles")
}

func TestRoleService_DeleteRole_WithUsers(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))

	// Create and assign a user
	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	userRepo := NewUserRepository(db, zerolog.Nop())
	require.NoError(t, userRepo.Create(ctx, user))
	require.NoError(t, repo.AssignRole(ctx, user.ID, role.ID, assignedBy))

	err := service.DeleteRole(ctx, role.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete role with assigned users")
}

func TestRoleService_InitializeSystemRoles(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	err := service.InitializeSystemRoles(ctx, tenantID)
	require.NoError(t, err)

	// Verify all system roles were created
	systemRoles := []string{
		RoleSuperAdmin,
		RoleAdmin,
		RoleOperator,
		RoleAuditor,
		RoleRequester,
	}

	for _, roleName := range systemRoles {
		role, err := repo.GetByName(ctx, tenantID, roleName)
		require.NoError(t, err, "System role %s should exist", roleName)
		assert.True(t, role.IsSystem, "Role %s should be marked as system", roleName)
	}

	// Running again should not error (idempotent)
	err = service.InitializeSystemRoles(ctx, tenantID)
	assert.NoError(t, err)
}

func TestRoleService_CheckPermission(t *testing.T) {
	db := setupRoleTestDB(t)
	if db == nil {
		return
	}

	repo := NewRoleRepository(db, zerolog.Nop())
	service := NewRoleService(repo, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())
	assignedBy := uuid.New()

	// Create user
	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	userRepo := NewUserRepository(db, zerolog.Nop())
	require.NoError(t, userRepo.Create(ctx, user))

	// Create permission
	permissionID := uuid.New()
	_, err := db.ExecContext(ctx, `
		INSERT INTO permissions (id, resource, action, scope, description)
		VALUES ($1, $2, $3, $4, $5)
	`, permissionID, "credential", "read", "all", "Read credentials")
	require.NoError(t, err)

	// Create role and grant permission
	role := &Role{
		Name:        "credential-reader",
		DisplayName: "Credential Reader",
		TenantID:    tenantID,
	}
	require.NoError(t, repo.Create(ctx, role))
	require.NoError(t, repo.GrantPermission(ctx, role.ID, permissionID))
	require.NoError(t, repo.AssignRole(ctx, user.ID, role.ID, assignedBy))

	// Check permission
	hasPermission, err := service.CheckPermission(ctx, user.ID, "credential", "read")
	require.NoError(t, err)
	assert.True(t, hasPermission)

	// Check permission user doesn't have
	hasPermission, err = service.CheckPermission(ctx, user.ID, "credential", "write")
	require.NoError(t, err)
	assert.False(t, hasPermission)
}
