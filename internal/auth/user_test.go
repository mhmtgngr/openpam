package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/testing"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserTestDB(t *testing.T) *sqlx.DB {
	db := testing.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil
	}
	testing.SetupTestDatabase(t, db)
	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.Equal(t, "active", user.Status)
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	t.Run("get existing user", func(t *testing.T) {
		found, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("get non-existent user", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	t.Run("get existing user by email", func(t *testing.T) {
		found, err := repo.GetByEmail(ctx, tenantID, "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, user.ID, found.ID)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("get user from different tenant", func(t *testing.T) {
		otherTenantID := uuid.New()
		_, err := repo.GetByEmail(ctx, otherTenantID, "test@example.com")
		assert.Error(t, err)
	})

	t.Run("get non-existent email", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, tenantID, "nonexistent@example.com")
		assert.Error(t, err)
	})
}

func TestUserRepository_List(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	// Create multiple users
	for i := 0; i < 5; i++ {
		user := &User{
			Email:     fmt.Sprintf("user%d@example.com", i),
			FirstName: "User",
			LastName:  fmt.Sprintf("%d", i),
			TenantID:  tenantID,
		}
		require.NoError(t, repo.Create(ctx, user))
	}

	t.Run("list users", func(t *testing.T) {
		users, err := repo.List(ctx, tenantID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, users, 5)
	})

	t.Run("list with limit", func(t *testing.T) {
		users, err := repo.List(ctx, tenantID, 3, 0)
		require.NoError(t, err)
		assert.Len(t, users, 3)
	})

	t.Run("list with offset", func(t *testing.T) {
		users, err := repo.List(ctx, tenantID, 10, 2)
		require.NoError(t, err)
		assert.Len(t, users, 3)
	})
}

func TestUserRepository_Update(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	// Update user
	user.FirstName = "Updated"
	user.LastName = "UserUpdated"
	err := repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.FirstName)
	assert.Equal(t, "UserUpdated", updated.LastName)
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	// Verify soft delete - user should not be found
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:        "test@example.com",
		FirstName:    "Test",
		LastName:     "User",
		TenantID:     tenantID,
		PasswordHash: "oldhash",
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.UpdatePassword(ctx, user.ID, "newhash")
	require.NoError(t, err)

	// Verify password changed
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "newhash", updated.PasswordHash)
}

func TestUserRepository_EnableMFA(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	secret := "JBSWY3DPEHPK3PXP"
	backupCodes := "hash1,hash2,hash3"

	err := repo.EnableMFA(ctx, user.ID, secret, backupCodes)
	require.NoError(t, err)

	// Verify MFA enabled
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, updated.MFAEnabled)
	assert.Equal(t, secret, updated.MFASecret)
	assert.Equal(t, backupCodes, updated.BackupCodes)
}

func TestUserRepository_DisableMFA(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
		MFAEnabled: true,
		MFASecret: "secret",
		BackupCodes: "codes",
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.DisableMFA(ctx, user.ID)
	require.NoError(t, err)

	// Verify MFA disabled
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, updated.MFAEnabled)
	assert.Empty(t, updated.MFASecret)
	assert.Empty(t, updated.BackupCodes)
}

func TestUserRepository_RecordLogin(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:        "test@example.com",
		FirstName:    "Test",
		LastName:     "User",
		TenantID:     tenantID,
		FailedLogins: 5,
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.RecordLogin(ctx, user.ID)
	require.NoError(t, err)

	// Verify login recorded
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.FailedLogins)
	assert.NotNil(t, updated.LastLoginAt)
}

func TestUserRepository_RecordFailedLogin(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, repo.Create(ctx, user))

	t.Run("record failed login", func(t *testing.T) {
		err := repo.RecordFailedLogin(ctx, user.ID, 5)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, updated.FailedLogins)
	})

	t.Run("lock after max attempts", func(t *testing.T) {
		// Record 5 failed logins
		for i := 0; i < 5; i++ {
			_ = repo.RecordFailedLogin(ctx, user.ID, 5)
		}

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, updated.FailedLogins)
		assert.Equal(t, "locked", updated.Status)
		assert.NotNil(t, updated.LockedUntil)
	})
}

func TestUserRepository_UnlockAccount(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:        "test@example.com",
		FirstName:    "Test",
		LastName:     "User",
		TenantID:     tenantID,
		Status:       "locked",
		FailedLogins: 5,
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.UnlockAccount(ctx, user.ID)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
	assert.Equal(t, 0, updated.FailedLogins)
	assert.Nil(t, updated.LockedUntil)
}

func TestUserRepository_Suspend(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
		Status:    "active",
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.Suspend(ctx, user.ID)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", updated.Status)
}

func TestUserRepository_Activate(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
		Status:    "suspended",
	}
	require.NoError(t, repo.Create(ctx, user))

	err := repo.Activate(ctx, user.ID)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
}

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	t.Run("hash password", func(t *testing.T) {
		hash, err := hasher.Hash("password123")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, "password123", hash)
		assert.Contains(t, hash, "$")
	})

	t.Run("unique hashes", func(t *testing.T) {
		hash1, err := hasher.Hash("password123")
		require.NoError(t, err)

		hash2, err := hasher.Hash("password123")
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2) // Due to random salt
	})
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher()

	hash, _ := hasher.Hash("password123")

	t.Run("verify correct password", func(t *testing.T) {
		result := hasher.Verify("password123", hash)
		assert.True(t, result)
	})

	t.Run("verify incorrect password", func(t *testing.T) {
		result := hasher.Verify("wrongpassword", hash)
		assert.False(t, result)
	})

	t.Run("verify against empty hash", func(t *testing.T) {
		result := hasher.Verify("password123", "")
		assert.False(t, result)
	})
}

func TestUserService_CreateUser(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	jwtMgr := NewJWTManager(nil, nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	mfa := NewMFAManager(nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	service := NewUserService(repo, jwtMgr, mfa, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}

	err := service.CreateUser(ctx, user, "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "password123", user.PasswordHash)
}

func TestUserService_validateUser(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	jwtMgr := NewJWTManager(nil, nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	mfa := NewMFAManager(nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	service := NewUserService(repo, jwtMgr, mfa, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	t.Run("valid user", func(t *testing.T) {
		user := &User{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			TenantID:  tenantID,
		}
		err := service.validateUser(user)
		assert.NoError(t, err)
	})

	t.Run("missing email", func(t *testing.T) {
		user := &User{
			FirstName: "Test",
			LastName:  "User",
			TenantID:  tenantID,
		}
		err := service.validateUser(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("missing first name", func(t *testing.T) {
		user := &User{
			Email:    "test@example.com",
			LastName: "User",
			TenantID: tenantID,
		}
		err := service.validateUser(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "first name")
	})

	t.Run("missing last name", func(t *testing.T) {
		user := &User{
			Email:     "test@example.com",
			FirstName: "Test",
			TenantID:  tenantID,
		}
		err := service.validateUser(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "last name")
	})

	t.Run("missing tenant ID", func(t *testing.T) {
		user := &User{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		}
		err := service.validateUser(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant")
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	db := setupUserTestDB(t)
	if db == nil {
		return
	}

	repo := NewUserRepository(db, zerolog.Nop())
	jwtMgr := NewJWTManager(nil, nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	mfa := NewMFAManager(nil, &mockCacheForTest{data: make(map[string]string)}, zerolog.Nop())
	service := NewUserService(repo, jwtMgr, mfa, zerolog.Nop())
	ctx := testing.Context(t)

	tenantID := uuid.MustParse(testing.GenerateTestTenantID())

	user := &User{
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		TenantID:  tenantID,
	}
	require.NoError(t, service.CreateUser(ctx, user, "oldpassword"))

	t.Run("update with correct current password", func(t *testing.T) {
		err := service.UpdatePassword(ctx, user.ID, "oldpassword", "newpassword")
		require.NoError(t, err)
	})

	t.Run("update with incorrect current password", func(t *testing.T) {
		err := service.UpdatePassword(ctx, user.ID, "wrongpassword", "newpassword")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid current password")
	})
}
