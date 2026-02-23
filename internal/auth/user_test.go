package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/crypto"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/lib/pq"
)

// setupUserTest creates a test database and user repository
func setupUserTest(t *testing.T) (*sqlx.DB, *UserRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)
	logger := opamtesting.Logger(t)
	repo := NewUserRepository(db, logger)

	return db, repo
}

// setupJWTManager creates a JWT manager for testing
func setupJWTManager(t *testing.T) (*JWTManager, *rsa.PrivateKey, *rsa.PublicKey, interface{}) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil, // No cache in tests
		logger:     zerolog.Logger{},
	}

	return jm, privateKey, publicKey, nil
}

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	password := "MySecurePassword123!"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	assert.Contains(t, hash, "$") // bcrypt hash format
}

func TestPasswordHasher_Hash_Uniqueness(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "SamePassword"

	hash1, err := hasher.Hash(password)
	require.NoError(t, err)

	hash2, err := hasher.Hash(password)
	require.NoError(t, err)

	// Different hashes for same password (due to salt)
	assert.NotEqual(t, hash1, hash2)
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "MySecurePassword123!"

	hash, err := hasher.Hash(password)
	require.NoError(t, err)

	// Correct password should verify
	assert.True(t, hasher.Verify(password, hash))

	// Wrong password should not verify
	assert.False(t, hasher.Verify("WrongPassword", hash))

	// Empty password should not verify
	assert.False(t, hasher.Verify("", hash))
}

func TestPasswordHasher_Verify_EmptyHash(t *testing.T) {
	hasher := NewPasswordHasher()

	assert.False(t, hasher.Verify("password", ""))
	assert.False(t, hasher.Verify("password", "invalid"))
	assert.False(t, hasher.Verify("password", "$2a$invalid"))
}

func TestUserRepository_Create(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "test@example.com",
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
	assert.Equal(t, "active", user.Status)
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user1 := &User{
		Email:        "duplicate@example.com",
		FirstName:    "First",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	// Try to create another user with same email
	user2 := &User{
		Email:        "duplicate@example.com",
		FirstName:    "Second",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err = repo.Create(ctx, user2)
	assert.Error(t, err)
}

func TestUserRepository_GetByID(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "getbyid@example.com",
		FirstName:    "Get",
		LastName:     "ByID",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Get user by ID
	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.FirstName, found.FirstName)
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	randomID := uuid.New()
	_, err := repo.GetByID(ctx, randomID)
	assert.Error(t, err)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "getbyemail@example.com",
		FirstName:    "Get",
		LastName:     "ByEmail",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Get user by email
	found, err := repo.GetByEmail(ctx, tenantID, "getbyemail@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Email, found.Email)
}

func TestUserRepository_GetByEmail_WrongTenant(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID1, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	tenantID2 := uuid.New()

	user := &User{
		Email:        "tenantuser@example.com",
		FirstName:    "Tenant",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID1,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Try to get with different tenant ID
	_, err = repo.GetByEmail(ctx, tenantID2, "tenantuser@example.com")
	assert.Error(t, err)
}

func TestUserRepository_List(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple users
	for i := 0; i < 5; i++ {
		user := &User{
			Email:        uuid.New().String() + "@example.com",
			FirstName:    "User",
			LastName:     string(rune('A' + i)),
			PasswordHash: "hash",
			TenantID:     tenantID,
		}
		require.NoError(t, repo.Create(ctx, user))
	}

	// List users
	users, err := repo.List(ctx, tenantID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 5)
}

func TestUserRepository_List_Pagination(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Create multiple users
	for i := 0; i < 15; i++ {
		user := &User{
			Email:        uuid.New().String() + "@example.com",
			FirstName:    "Paginated",
			LastName:     string(rune('A' + i)),
			PasswordHash: "hash",
			TenantID:     tenantID,
		}
		require.NoError(t, repo.Create(ctx, user))
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

func TestUserRepository_Update(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "update@example.com",
		FirstName:    "Original",
		LastName:     "Name",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update user
	user.FirstName = "Updated"
	user.LastName = "NameToo"
	err = repo.Update(ctx, user)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.FirstName)
	assert.Equal(t, "NameToo", updated.LastName)
}

func TestUserRepository_Delete(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "delete@example.com",
		FirstName:    "Delete",
		LastName:     "Me",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Delete user
	err = repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	// Verify soft delete (user should still exist but with deleted_at set)
	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err) // Should not find active user
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "password@example.com",
		FirstName:    "Password",
		LastName:     "User",
		PasswordHash: "oldhash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update password
	newHash := "newhash"
	err = repo.UpdatePassword(ctx, user.ID, newHash)
	require.NoError(t, err)

	// Verify password was updated
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, newHash, updated.PasswordHash)
}

func TestUserRepository_EnableMFA(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "mfa@example.com",
		FirstName:    "MFA",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)
	assert.False(t, user.MFAEnabled)

	// Enable MFA
	secret := "JBSWY3DPEHPK3PXP"
	backupCodes := "code1,code2,code3"
	err = repo.EnableMFA(ctx, user.ID, secret, backupCodes)
	require.NoError(t, err)

	// Verify MFA enabled
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.True(t, updated.MFAEnabled)
	assert.Equal(t, secret, updated.MFASecret)
}

func TestUserRepository_DisableMFA(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "nofa@example.com",
		FirstName:    "NoMFA",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
		MFAEnabled:   true,
		MFASecret:    "secret",
		BackupCodes:  "codes",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Disable MFA
	err = repo.DisableMFA(ctx, user.ID)
	require.NoError(t, err)

	// Verify MFA disabled
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, updated.MFAEnabled)
	assert.Empty(t, updated.MFASecret)
	assert.Empty(t, updated.BackupCodes)
}

func TestUserRepository_RecordLogin(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "login@example.com",
		FirstName:    "Login",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
		FailedLogins: 3,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Record successful login
	err = repo.RecordLogin(ctx, user.ID)
	require.NoError(t, err)

	// Verify failed logins reset and last_login_at set
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.FailedLogins)
	assert.NotNil(t, updated.LastLoginAt)
}

func TestUserRepository_RecordFailedLogin(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "failed@example.com",
		FirstName:    "Failed",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Record failed logins
	for i := 0; i < 3; i++ {
		err = repo.RecordFailedLogin(ctx, user.ID, 5)
		require.NoError(t, err)
	}

	// Verify failed login count
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, updated.FailedLogins)
}

func TestUserRepository_RecordFailedLogin_LocksAccount(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "lock@example.com",
		FirstName:    "Lock",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Record 5 failed logins (should lock account)
	for i := 0; i < 5; i++ {
		err = repo.RecordFailedLogin(ctx, user.ID, 5)
		require.NoError(t, err)
	}

	// Verify account is locked
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "locked", updated.Status)
	assert.NotNil(t, updated.LockedUntil)
}

func TestUserRepository_UnlockAccount(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	now := time.Now()
	user := &User{
		Email:        "locked@example.com",
		FirstName:    "Locked",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
		Status:       "locked",
		FailedLogins: 5,
		LockedUntil:  &now,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Unlock account
	err = repo.UnlockAccount(ctx, user.ID)
	require.NoError(t, err)

	// Verify account unlocked
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
	assert.Equal(t, 0, updated.FailedLogins)
	assert.Nil(t, updated.LockedUntil)
}

func TestUserRepository_Suspend(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "suspend@example.com",
		FirstName:    "Suspend",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Suspend user
	err = repo.Suspend(ctx, user.ID)
	require.NoError(t, err)

	// Verify status
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "suspended", updated.Status)
}

func TestUserRepository_Activate(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:        "activate@example.com",
		FirstName:    "Activate",
		LastName:     "User",
		PasswordHash: "hash",
		TenantID:     tenantID,
		Status:       "suspended",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Activate user
	err = repo.Activate(ctx, user.ID)
	require.NoError(t, err)

	// Verify status
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "active", updated.Status)
}

func TestUserService_CreateUser(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	// Create mock JWT and MFA managers (not needed for CreateUser)
	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	user := &User{
		Email:     "service@example.com",
		FirstName: "Service",
		LastName:  "User",
		TenantID:  tenantID,
	}

	err := service.CreateUser(ctx, user, "SecurePassword123!")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotEmpty(t, user.PasswordHash)
	assert.NotEqual(t, "SecurePassword123!", user.PasswordHash)
}

func TestUserService_CreateUser_Validation(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	tests := []struct {
		name    string
		user    *User
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing email",
			user: &User{
				FirstName: "Test",
				LastName:  "User",
				TenantID:  tenantID,
			},
			wantErr: true,
			errMsg:  "email",
		},
		{
			name: "missing first name",
			user: &User{
				Email:    "test@example.com",
				LastName: "User",
				TenantID: tenantID,
			},
			wantErr: true,
			errMsg:  "first name",
		},
		{
			name: "missing last name",
			user: &User{
				Email:     "test@example.com",
				FirstName: "Test",
				TenantID:  tenantID,
			},
			wantErr: true,
			errMsg:  "last name",
		},
		{
			name: "missing tenant ID",
			user: &User{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
			errMsg:  "tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.CreateUser(ctx, tt.user, "password")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestUserService_Authenticate_Success(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	password := "SecurePassword123!"
	hash, _ := crypto.HashPassword(password)

	user := &User{
		Email:        "auth@example.com",
		FirstName:    "Auth",
		LastName:     "User",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Authenticate with correct credentials
	authenticated, err := service.Authenticate(ctx, tenantID, "auth@example.com", password)
	require.NoError(t, err)
	assert.Equal(t, user.Email, authenticated.Email)
	assert.Equal(t, "active", authenticated.Status)
}

func TestUserService_Authenticate_WrongPassword(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	password := "SecurePassword123!"
	hash, _ := crypto.HashPassword(password)

	user := &User{
		Email:        "wrongpass@example.com",
		FirstName:    "Wrong",
		LastName:     "Pass",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Authenticate with wrong password
	_, err = service.Authenticate(ctx, tenantID, "wrongpass@example.com", "WrongPassword")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid credentials")
}

func TestUserService_Authenticate_LockedAccount(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	password := "SecurePassword123!"
	hash, _ := crypto.HashPassword(password)
	future := time.Now().Add(time.Hour)

	user := &User{
		Email:        "locked@example.com",
		FirstName:    "Locked",
		LastName:     "User",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "locked",
		LockedUntil:  &future,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Try to authenticate locked account
	_, err = service.Authenticate(ctx, tenantID, "locked@example.com", password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "locked")
}

func TestUserService_Authenticate_SuspendedAccount(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	password := "SecurePassword123!"
	hash, _ := crypto.HashPassword(password)

	user := &User{
		Email:        "suspended@example.com",
		FirstName:    "Suspended",
		LastName:     "User",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "suspended",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Try to authenticate suspended account
	_, err = service.Authenticate(ctx, tenantID, "suspended@example.com", password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "suspended")
}

func TestUserService_Authenticate_NonExistentUser(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())

	// Try to authenticate non-existent user
	_, err := service.Authenticate(ctx, tenantID, "nonexistent@example.com", "password")
	assert.Error(t, err)
}

func TestUserService_Login_Success(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	password := "SecurePassword123!"
	hash, _ := crypto.HashPassword(password)

	user := &User{
		Email:        "loginuser@example.com",
		FirstName:    "Login",
		LastName:     "User",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Login
	accessToken, refreshToken, loggedInUser, err := service.Login(ctx, tenantID, "loginuser@example.com", password)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.Equal(t, user.Email, loggedInUser.Email)
}

func TestUserService_UpdatePassword(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	oldPassword := "OldPassword123!"
	hash, _ := crypto.HashPassword(oldPassword)

	user := &User{
		Email:        "changepass@example.com",
		FirstName:    "Change",
		LastName:     "Pass",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Update password
	newPassword := "NewPassword456!"
	err = service.UpdatePassword(ctx, user.ID, oldPassword, newPassword)
	require.NoError(t, err)

	// Verify password changed
	updated, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.NotEqual(t, hash, updated.PasswordHash)
	assert.True(t, crypto.VerifyPassword(newPassword, updated.PasswordHash))
}

func TestUserService_UpdatePassword_WrongCurrentPassword(t *testing.T) {
	db, repo := setupUserTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	jm, _, _, _ := setupJWTManager(t)
	mfa := NewMFAManager(nil, nil, nil, opamtesting.Logger(t))

	service := NewUserService(repo, jm, mfa, opamtesting.Logger(t))

	tenantID, _ := uuid.Parse(opamtesting.GenerateTestTenantID())
	hash, _ := crypto.HashPassword("OldPassword123!")

	user := &User{
		Email:        "wrongcurrent@example.com",
		FirstName:    "Wrong",
		LastName:     "Current",
		PasswordHash: hash,
		TenantID:     tenantID,
		Status:       "active",
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	// Try to update with wrong current password
	err = service.UpdatePassword(ctx, user.ID, "WrongPassword", "NewPassword456!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid current password")
}
