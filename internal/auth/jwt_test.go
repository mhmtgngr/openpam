package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	tests := []struct {
		name      string
		userID    string
		email     string
		tenantID  string
		roles     []string
		mfaEnabled bool
		wantErr   bool
	}{
		{
			name:      "valid token pair",
			userID:    uuid.New().String(),
			email:     "test@example.com",
			tenantID:  uuid.New().String(),
			roles:     []string{"admin", "operator"},
			mfaEnabled: true,
			wantErr:   false,
		},
		{
			name:      "user without roles",
			userID:    uuid.New().String(),
			email:     "user@example.com",
			tenantID:  uuid.New().String(),
			roles:     []string{},
			mfaEnabled: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessToken, refreshToken, err := jwtManager.GenerateTokenPair(
				context.Background(),
				tt.userID,
				tt.email,
				tt.tenantID,
				tt.roles,
				tt.mfaEnabled,
			)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, accessToken)
			assert.NotEmpty(t, refreshToken)
			assert.NotEqual(t, accessToken, refreshToken)
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()
	roles := []string{"admin"}

	accessToken, refreshToken, err := jwtManager.GenerateTokenPair(
		context.Background(),
		userID,
		email,
		tenantID,
		roles,
		true,
	)
	require.NoError(t, err)

	t.Run("valid access token", func(t *testing.T) {
		claims, err := jwtManager.ValidateToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, tenantID, claims.TenantID)
		assert.Equal(t, roles, claims.Roles)
		assert.True(t, claims.MFAEnabled)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("valid refresh token", func(t *testing.T) {
		claims, err := jwtManager.ValidateToken(refreshToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "refresh", claims.TokenType)
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := jwtManager.ValidateToken("invalid.token.here")
		assert.Error(t, err)
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := jwtManager.ValidateToken("")
		assert.Error(t, err)
	})

	t.Run("token signed with wrong key", func(t *testing.T) {
		wrongKeyPair, _ := crypto.GenerateKeyPair()
		wrongJWT := NewJWTManager(wrongKeyPair.PrivateKey, wrongKeyPair.PublicKey, mockCache, logger)
		wrongToken, _, _ := wrongJWT.GenerateTokenPair(context.Background(), userID, email, tenantID, roles, false)

		_, err := jwtManager.ValidateToken(wrongToken)
		assert.Error(t, err)
	})
}

func TestJWTManager_RefreshToken(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()
	roles := []string{"admin"}

	accessToken, refreshToken, err := jwtManager.GenerateTokenPair(
		context.Background(),
		userID,
		email,
		tenantID,
		roles,
		true,
	)
	require.NoError(t, err)

	t.Run("valid refresh token", func(t *testing.T) {
		newAccessToken, err := jwtManager.RefreshToken(context.Background(), refreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEqual(t, accessToken, newAccessToken)

		// Verify new token is valid
		claims, err := jwtManager.ValidateToken(newAccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := jwtManager.RefreshToken(context.Background(), "invalid")
		assert.Error(t, err)
	})

	t.Run("access token cannot be used for refresh", func(t *testing.T) {
		_, err := jwtManager.RefreshToken(context.Background(), accessToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token type")
	})

	t.Run("refresh token not in cache", func(t *testing.T) {
		_, err := jwtManager.RefreshToken(context.Background(), refreshToken)
		// First call should work (token is in cache from generation)
		require.NoError(t, err)

		// After removing from cache, it should fail
		mockCache.mu.Lock()
		mockCache.data = make(map[string]string)
		mockCache.mu.Unlock()

		_, err = jwtManager.RefreshToken(context.Background(), refreshToken)
		assert.Error(t, err)
	})
}

func TestJWTManager_RevokeToken(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	t.Run("revoke token", func(t *testing.T) {
		tokenID := uuid.New().String()
		err := jwtManager.RevokeToken(context.Background(), tokenID)
		assert.NoError(t, err)
	})
}

func TestJWTManager_RevokeUserTokens(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()

	// Generate multiple token pairs for the same user
	for i := 0; i < 3; i++ {
		_, _, err := jwtManager.GenerateTokenPair(
			context.Background(),
			userID,
			email,
			tenantID,
			[]string{"admin"},
			false,
		)
		require.NoError(t, err)
	}

	t.Run("revoke all user tokens", func(t *testing.T) {
		err := jwtManager.RevokeUserTokens(context.Background(), userID)
		assert.NoError(t, err)
	})
}

func TestJWTManager_ExtractClaims(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()

	accessToken, _, err := jwtManager.GenerateTokenPair(
		context.Background(),
		userID,
		email,
		tenantID,
		[]string{"admin"},
		true,
	)
	require.NoError(t, err)

	t.Run("extract claims from valid token", func(t *testing.T) {
		claims, err := jwtManager.ExtractClaims(accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, tenantID, claims.TenantID)
	})

	t.Run("extract claims from invalid token", func(t *testing.T) {
		_, err := jwtManager.ExtractClaims("invalid.token")
		assert.Error(t, err)
	})
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	keyPair, err := crypto.GenerateKeyPair()
	require.NoError(t, err)

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	logger := zerolog.Nop()

	jwtManager := NewJWTManager(keyPair.PrivateKey, keyPair.PublicKey, mockCache, logger)

	t.Run("access token expiration", func(t *testing.T) {
		userID := uuid.New().String()
		accessToken, _, err := jwtManager.GenerateTokenPair(
			context.Background(),
			userID,
			"test@example.com",
			uuid.New().String(),
			[]string{},
			false,
		)
		require.NoError(t, err)

		// Parse without validation to check expiration
		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(accessToken, &Claims{})
		require.NoError(t, err)

		claims := token.Claims.(*Claims)
		expectedExpiry := time.Now().Add(AccessTokenDuration)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, time.Second)
	})

	t.Run("refresh token expiration", func(t *testing.T) {
		userID := uuid.New().String()
		_, refreshToken, err := jwtManager.GenerateTokenPair(
			context.Background(),
			userID,
			"test@example.com",
			uuid.New().String(),
			[]string{},
			false,
		)
		require.NoError(t, err)

		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(refreshToken, &Claims{})
		require.NoError(t, err)

		claims := token.Claims.(*Claims)
		expectedExpiry := time.Now().Add(RefreshTokenDuration)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, time.Second)
	})
}

// mockCacheForTest is a minimal mock for testing JWT manager
type mockCacheForTest struct {
	mu   sync.Mutex
	data map[string]string
}

func (m *mockCacheForTest) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = "cached"
	return nil
}

func (m *mockCacheForTest) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.data[key]; exists {
		return nil
	}
	return fmt.Errorf("key not found")
}

func (m *mockCacheForTest) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockCacheForTest) DeleteByPattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *mockCacheForTest) Exists(ctx context.Context, key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.data[key]
	return exists
}

func (m *mockCacheForTest) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCacheForTest) Client() interface{} {
	return m
}

func (m *mockCacheForTest) Close() error {
	return nil
}

func (m *mockCacheForTest) Health(ctx context.Context) error {
	return nil
}

func (m *mockCacheForTest) Sessions() *cache.SessionStore {
	return nil
}

func (m *mockCacheForTest) RateLimiter() *cache.RateLimiter {
	return nil
}

func (m *mockCacheForTest) PubSub() *cache.PubSub {
	return nil
}

func (m *mockCacheForTest) Tokens() *cache.TokenStore {
	return &mockTokenStore{cache: m}
}

type mockTokenStore struct {
	cache *mockCacheForTest
}

func (m *mockTokenStore) RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error {
	m.cache.Set(ctx, "revoked:"+tokenID, true, expiration)
	return nil
}

func (m *mockTokenStore) IsRevoked(ctx context.Context, tokenID string) bool {
	return m.cache.Exists(ctx, "revoked:"+tokenID)
}
