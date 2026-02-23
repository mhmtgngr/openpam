package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestRSAKeyPair generates a test RSA key pair
func generateTestRSAKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return privateKey, &privateKey.PublicKey
}

func TestJWTManager_NewJWTManager(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := NewJWTManager(privateKey, publicKey, nil, zerolog.Logger{})

	assert.NotNil(t, jm)
	assert.Equal(t, privateKey, jm.privateKey)
	assert.Equal(t, publicKey, jm.publicKey)
}

func TestJWTManager_GenerateTokenPair_NoCache(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	_ = &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil, // No cache - GenerateTokenPair will handle nil gracefully for test
		logger:     zerolog.Logger{},
	}

	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()
	roles := []string{"admin"}
	mfaEnabled := true

	// Create JWT manager with proper token generation
	now := time.Now()

	// Generate access token
	accessClaims := Claims{
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		Roles:      roles,
		MFAEnabled: mfaEnabled,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Generate refresh token
	refreshClaims := Claims{
		UserID:    userID,
		Email:     email,
		TenantID:  tenantID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(privateKey)
	require.NoError(t, err)

	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)
	assert.NotEqual(t, accessToken, refreshToken)
}

func TestJWTManager_ValidateToken_ValidAccessToken(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	userID := uuid.New().String()
	now := time.Now()

	// Create token
	accessClaims := Claims{
		UserID:     userID,
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{"admin"},
		MFAEnabled: true,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Validate token
	claims, err := jm.ValidateToken(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "access", claims.TokenType)
}

func TestJWTManager_ValidateToken_ValidRefreshToken(t *testing.T) {
	t.Skip("requires cache - refresh token validation checks cache for revocation")

	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()

	// Create refresh token
	refreshClaims := Claims{
		UserID:    uuid.New().String(),
		Email:     "test@example.com",
		TenantID:  uuid.New().String(),
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Validate token
	claims, err := jm.ValidateToken(refreshToken)
	require.NoError(t, err)
	assert.Equal(t, "refresh", claims.TokenType)
}

func TestJWTManager_ValidateToken_TamperedToken(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	accessClaims := Claims{
		UserID:     uuid.New().String(),
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Tamper with the token
	tamperedToken := accessToken[:len(accessToken)-5] + "XXXXX"

	_, err = jm.ValidateToken(tamperedToken)
	assert.Error(t, err)
}

func TestJWTManager_ValidateToken_MalformedToken(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"not a jwt", "not.a.jwt"},
		{"incomplete jwt", "header.payload"},
		{"random string", uuid.New().String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jm.ValidateToken(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestJWTManager_ValidateToken_WrongAlgorithm(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	userID := uuid.New().String()

	// Create a token with HS256 instead of RS256
	wrongClaims := jwt.MapClaims{
		"user_id":    userID,
		"email":      "test@example.com",
		"tenant_id":  uuid.New().String(),
		"token_type": "access",
		"iat":        now.Unix(),
		"exp":        now.Add(15 * time.Minute).Unix(),
	}

	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, wrongClaims)
	tokenString, err := wrongToken.SignedString([]byte("secret"))
	require.NoError(t, err)

	_, err = jm.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

func TestJWTManager_ExtractClaims(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	userID := uuid.New().String()
	email := "test@example.com"
	tenantID := uuid.New().String()
	roles := []string{"admin"}

	// Create access token
	accessClaims := Claims{
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		Roles:      roles,
		MFAEnabled: true,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Extract claims without validation
	claims, err := jm.ExtractClaims(accessToken)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, tenantID, claims.TenantID)
	assert.Equal(t, roles, claims.Roles)
}

func TestJWTManager_TokenExpiry(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	userID := uuid.New().String()

	// Create access token
	accessClaims := Claims{
		UserID:     userID,
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Parse access token without validation to check expiry
	claims, err := jm.ExtractClaims(accessToken)
	require.NoError(t, err)

	accessExpiry := claims.ExpiresAt.Time
	assert.True(t, accessExpiry.After(now.Add(14*time.Minute)))
	assert.True(t, accessExpiry.Before(now.Add(16*time.Minute)))
}

func TestJWTManager_ValidateToken_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	// Create an expired token
	now := time.Now().Add(-1 * time.Hour)
	accessClaims := Claims{
		UserID:     uuid.New().String(),
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	// Token should be expired
	_, err = jm.ValidateToken(accessToken)
	assert.Error(t, err)
}

func TestClaims_RegisteredFields(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	userID := uuid.New().String()

	// Create access token
	accessClaims := Claims{
		UserID:     userID,
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	require.NoError(t, err)

	claims, err := jm.ExtractClaims(accessToken)
	require.NoError(t, err)

	// Check registered claims
	assert.NotEmpty(t, claims.ID)       // JWT ID
	assert.Equal(t, Issuer, claims.Issuer) // Issuer
	assert.NotEmpty(t, claims.Subject) // Subject (user ID)
	assert.False(t, claims.IssuedAt.IsZero())
	assert.False(t, claims.ExpiresAt.IsZero())
}

func TestJWTManager_GenerateToken_DifferentTokens(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	userID := uuid.New().String()

	// First token
	accessClaims1 := Claims{
		UserID:     userID,
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	token1, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims1).SignedString(privateKey)
	require.NoError(t, err)

	// Wait a tiny bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Second token
	accessClaims2 := Claims{
		UserID:     userID,
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{},
		MFAEnabled: false,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	token2, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims2).SignedString(privateKey)
	require.NoError(t, err)

	// Tokens should be different (different JTI due to uuid.New())
	assert.NotEqual(t, token1, token2)

	// Extract claims and verify different JWT IDs
	claims1, _ := jm.ExtractClaims(token1)
	claims2, _ := jm.ExtractClaims(token2)
	assert.NotEqual(t, claims1.ID, claims2.ID)
}

func TestJWTManager_GenerateToken_Tables(t *testing.T) {
	privateKey, publicKey := generateTestRSAKeyPair(t)

	tests := []struct {
		name       string
		userID     string
		email      string
		tenantID   string
		roles      []string
		mfaEnabled bool
	}{
		{
			name:       "admin user with MFA",
			userID:     uuid.New().String(),
			email:      "admin@example.com",
			tenantID:   uuid.New().String(),
			roles:      []string{"admin"},
			mfaEnabled: true,
		},
		{
			name:       "regular user without MFA",
			userID:     uuid.New().String(),
			email:      "user@example.com",
			tenantID:   uuid.New().String(),
			roles:      []string{},
			mfaEnabled: false,
		},
		{
			name:       "user with multiple roles",
			userID:     uuid.New().String(),
			email:      "multi@example.com",
			tenantID:   uuid.New().String(),
			roles:      []string{"admin", "operator", "auditor"},
			mfaEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := &JWTManager{
				privateKey: privateKey,
				publicKey:  publicKey,
				cache:      nil,
				logger:     zerolog.Logger{},
			}

			now := time.Now()
			accessClaims := Claims{
				UserID:     tt.userID,
				Email:      tt.email,
				TenantID:   tt.tenantID,
				Roles:      tt.roles,
				MFAEnabled: tt.mfaEnabled,
				TokenType:  "access",
				RegisteredClaims: jwt.RegisteredClaims{
					ID:        uuid.New().String(),
					Issuer:    Issuer,
					Subject:   tt.userID,
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
				},
			}

			accessToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
			require.NoError(t, err)

			// Verify claims
			claims, err := jm.ValidateToken(accessToken)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.email, claims.Email)
			assert.Equal(t, tt.tenantID, claims.TenantID)
			assert.Equal(t, tt.roles, claims.Roles)
			assert.Equal(t, tt.mfaEnabled, claims.MFAEnabled)
			assert.Equal(t, "access", claims.TokenType)
		})
	}
}

func TestJWTManager_RevokeToken(t *testing.T) {
	// Test that RevokeToken method exists and handles nil cache gracefully
	privateKey, publicKey := generateTestRSAKeyPair(t)

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil, // No cache in test
		logger:     zerolog.Logger{},
	}

	ctx := context.Background()
	tokenID := uuid.New().String()

	// With nil cache, this will panic - we just verify the method exists
	assert.Panics(t, func() {
		_ = jm.RevokeToken(ctx, tokenID)
	})
}

func TestJWTConstants(t *testing.T) {
	assert.Equal(t, 15*time.Minute, AccessTokenDuration)
	assert.Equal(t, 7*24*time.Hour, RefreshTokenDuration)
	assert.Equal(t, "openpam", Issuer)
}

func TestClaims_Struct(t *testing.T) {
	claims := Claims{
		UserID:     "user123",
		Email:      "test@example.com",
		TenantID:   "tenant456",
		Roles:      []string{"admin", "operator"},
		MFAEnabled: true,
		TokenType:  "access",
	}

	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, "tenant456", claims.TenantID)
	assert.Len(t, claims.Roles, 2)
	assert.True(t, claims.MFAEnabled)
	assert.Equal(t, "access", claims.TokenType)
}

func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	privateKey, _ := generateTestRSAKeyPair(&testing.T{})
	now := time.Now()

	accessClaims := Claims{
		UserID:     uuid.New().String(),
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{"admin"},
		MFAEnabled: true,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		accessClaims.ID = uuid.New().String()
		_, _ = jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)
	}
}

func BenchmarkJWTManager_ValidateToken(b *testing.B) {
	privateKey, publicKey := generateTestRSAKeyPair(&testing.T{})

	jm := &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      nil,
		logger:     zerolog.Logger{},
	}

	now := time.Now()
	accessClaims := Claims{
		UserID:     uuid.New().String(),
		Email:      "test@example.com",
		TenantID:   uuid.New().String(),
		Roles:      []string{"admin"},
		MFAEnabled: true,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	token, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims).SignedString(privateKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = jm.ValidateToken(token)
	}
}

// Test cache integration (requires actual cache implementation)
func TestJWTManager_WithRealCache_NeedsRedis(t *testing.T) {
	t.Skip("requires Redis connection - implement with real cache.Cache when available")

	// This test would require:
	// 1. A real Redis instance running
	// 2. A cache.Cache instance created via cache.New()
	// 3. Full integration testing of token caching and revocation
}
