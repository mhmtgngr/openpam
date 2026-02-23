package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

const (
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
	Issuer              = "openpam"
)

// Claims represents JWT claims
type Claims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	TenantID  string   `json:"tenant_id"`
	Roles     []string `json:"roles"`
	MFAEnabled bool    `json:"mfa_enabled"`
	TokenType string   `json:"token_type"` // access or refresh
	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	cache      *cache.Cache
	logger     zerolog.Logger
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, c *cache.Cache, logger zerolog.Logger) *JWTManager {
	return &JWTManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		cache:      c,
		logger:     logger,
	}
}

// GenerateTokenPair generates an access and refresh token pair
func (j *JWTManager) GenerateTokenPair(ctx context.Context, userID, email, tenantID string, roles []string, mfaEnabled bool) (accessToken, refreshToken string, err error) {
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

	accessToken, err = j.generateToken(accessClaims)
	if err != nil {
		return "", "", fmt.Errorf("jwt.GenerateAccessToken: %w", err)
	}

	// Generate refresh token
	refreshClaims := Claims{
		UserID:     userID,
		Email:      email,
		TenantID:   tenantID,
		TokenType:  "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenDuration)),
		},
	}

	refreshToken, err = j.generateToken(refreshClaims)
	if err != nil {
		return "", "", fmt.Errorf("jwt.GenerateRefreshToken: %w", err)
	}

	// Store refresh token in cache for revocation
	tokenID := refreshClaims.ID
	if err := j.cache.Set(ctx, fmt.Sprintf("refresh_token:%s", tokenID), userID, RefreshTokenDuration); err != nil {
		j.logger.Error().Err(err).Msg("Failed to cache refresh token")
	}

	return accessToken, refreshToken, nil
}

// generateToken generates a JWT token
func (j *JWTManager) generateToken(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(j.privateKey)
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", token.Header["alg"])
		}
		return j.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("jwt.Parse: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}

	// Check if token is revoked
	if claims.TokenType == "refresh" {
		if j.cache.Tokens().IsRevoked(context.Background(), claims.ID) {
			return nil, fmt.Errorf("jwt: token revoked")
		}
	}

	return claims, nil
}

// RefreshToken generates a new access token using a refresh token
func (j *JWTManager) RefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Validate refresh token
	claims, err := j.ValidateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("jwt.ValidateToken: %w", err)
	}

	if claims.TokenType != "refresh" {
		return "", fmt.Errorf("jwt: invalid token type")
	}

	// Check if token is in cache
	key := fmt.Sprintf("refresh_token:%s", claims.ID)
	if !j.cache.Exists(ctx, key) {
		return "", fmt.Errorf("jwt: refresh token not found or expired")
	}

	// Generate new access token
	now := time.Now()
	accessClaims := Claims{
		UserID:     claims.UserID,
		Email:      claims.Email,
		TenantID:   claims.TenantID,
		Roles:      claims.Roles,
		MFAEnabled: claims.MFAEnabled,
		TokenType:  "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenDuration)),
		},
	}

	accessToken, err := j.generateToken(accessClaims)
	if err != nil {
		return "", fmt.Errorf("jwt.GenerateAccessToken: %w", err)
	}

	return accessToken, nil
}

// RevokeToken revokes a refresh token
func (j *JWTManager) RevokeToken(ctx context.Context, tokenID string) error {
	return j.cache.Tokens().RevokeToken(ctx, tokenID, RefreshTokenDuration)
}

// RevokeUserTokens revokes all refresh tokens for a user
func (j *JWTManager) RevokeUserTokens(ctx context.Context, userID string) error {
	// Delete all refresh tokens for user
	pattern := fmt.Sprintf("refresh_token:*")
	iter := j.cache.Client().Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		var storedUserID string
		if err := j.cache.Get(ctx, iter.Val(), &storedUserID); err == nil && storedUserID == userID {
			_ = j.cache.Delete(ctx, iter.Val())
		}
	}
	return iter.Err()
}

// ExtractClaims extracts claims without validation (for logging)
func (j *JWTManager) ExtractClaims(tokenString string) (*Claims, error) {
	parser := jwt.NewParser()
	claims := &Claims{}
	_, _, err := parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, fmt.Errorf("jwt.ParseUnverified: %w", err)
	}
	return claims, nil
}
