package auth

import (
	"context"
	"encoding/base32"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTOTPManager creates a TOTP manager for testing
func setupTOTPManager(t *testing.T) *TOTPManager {
	cfg := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}
	return NewTOTPManager(cfg, nil, zerolog.Logger{})
}

func TestTOTPManager_GenerateSecret(t *testing.T) {
	totp := setupTOTPManager(t)

	secret, err := totp.GenerateSecret()
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Len(t, secret, 52) // 32 bytes = 52 base32 characters

	// Should be valid base32
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	require.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func TestTOTPManager_GenerateSecret_Uniqueness(t *testing.T) {
	totp := setupTOTPManager(t)

	secrets := make(map[string]bool)
	for i := 0; i < 100; i++ {
		secret, err := totp.GenerateSecret()
		require.NoError(t, err)
		assert.False(t, secrets[secret], "Secret should be unique")
		secrets[secret] = true
	}
	assert.Len(t, secrets, 100)
}

func TestTOTPManager_GenerateQRCode(t *testing.T) {
	totp := setupTOTPManager(t)

	tests := []struct {
		name  string
		email string
	}{
		{"standard email", "user@example.com"},
		{"email with subdomain", "user@mail.example.com"},
		{"email with plus", "user+tag@example.com"},
		{"uppercase email", "USER@EXAMPLE.COM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := "JBSWY3DPEHPK3PXP"
			qrCode, err := totp.GenerateQRCode(tt.email, secret)
			require.NoError(t, err)
			assert.NotEmpty(t, qrCode)
			assert.Greater(t, len(qrCode), 100) // QR code should be substantial
		})
	}
}

func TestTOTPManager_VerifyCode_Valid(t *testing.T) {
	totp := setupTOTPManager(t)

	// Use a known secret and generate a valid code for current time
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now().Unix()
	counter := now / 30
	validCode := totp.generateCode(secret, counter)

	assert.True(t, totp.VerifyCode(secret, validCode))
}

func TestTOTPManager_VerifyCode_WithLeeway(t *testing.T) {
	cfg := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     2, // 2 periods of leeway
	}
	totp := NewTOTPManager(cfg, nil, zerolog.Logger{})

	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now().Unix()

	// Generate code for previous period
	counterPrev := (now / 30) - 1
	codePrev := totp.generateCode(secret, counterPrev)
	assert.True(t, totp.VerifyCode(secret, codePrev), "Should verify code from previous period")

	// Generate code for next period
	counterNext := (now / 30) + 1
	codeNext := totp.generateCode(secret, counterNext)
	assert.True(t, totp.VerifyCode(secret, codeNext), "Should verify code from next period")
}

func TestTOTPManager_VerifyCode_Invalid(t *testing.T) {
	totp := setupTOTPManager(t)

	secret := "JBSWY3DPEHPK3PXP"

	invalidCodes := []string{
		"000000",
		"123456",
		"999999",
		"111111",
		"abcdef",
		"12345",
		"1234567",
	}

	for _, code := range invalidCodes {
		t.Run("code_"+code, func(t *testing.T) {
			assert.False(t, totp.VerifyCode(secret, code))
		})
	}
}

func TestTOTPManager_VerifyCode_InvalidSecret(t *testing.T) {
	totp := setupTOTPManager(t)

	invalidSecrets := []string{
		"",
		"!!!",
		"invalid",
		"TOOLONGTOOLONGTOOLONGTOOLONGTOOLONGTOOLONG",
	}

	for _, secret := range invalidSecrets {
		t.Run("secret_"+secret, func(t *testing.T) {
			assert.False(t, totp.VerifyCode(secret, "123456"))
		})
	}
}

func TestTOTPManager_GenerateCode_Consistency(t *testing.T) {
	totp := setupTOTPManager(t)

	secret := "JBSWY3DPEHPK3PXP"
	counter := int64(12345)

	// Same secret and counter should generate same code
	code1 := totp.generateCode(secret, counter)
	code2 := totp.generateCode(secret, counter)

	assert.Equal(t, code1, code2)
	assert.Len(t, code1, 6)
}

func TestTOTPManager_GenerateCode_DifferentCounters(t *testing.T) {
	totp := setupTOTPManager(t)

	secret := "JBSWY3DPEHPK3PXP"

	code1 := totp.generateCode(secret, 100)
	code2 := totp.generateCode(secret, 101)
	code3 := totp.generateCode(secret, 102)

	// Different counters should produce different codes
	assert.NotEqual(t, code1, code2)
	assert.NotEqual(t, code2, code3)
	assert.NotEqual(t, code1, code3)
}

func TestTOTPManager_GenerateBackupCodes(t *testing.T) {
	totp := setupTOTPManager(t)

	tests := []struct {
		name  string
		count int
	}{
		{"5 codes", 5},
		{"10 codes", 10},
		{"15 codes", 15},
		{"20 codes", 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, err := totp.GenerateBackupCodes(tt.count)
			require.NoError(t, err)
			assert.Len(t, codes, tt.count)

			// Each code should be 16 hex characters
			for _, code := range codes {
				assert.Len(t, code, 16)
			}

			// All codes should be unique
			unique := make(map[string]bool)
			for _, code := range codes {
				unique[code] = true
			}
			assert.Len(t, unique, tt.count)
		})
	}
}

func TestTOTPManager_HashBackupCodes(t *testing.T) {
	totp := setupTOTPManager(t)

	codes := []string{"abc123def4567890", "xyz987uvw6543210", "0123456789abcdef"}

	hashed, err := totp.HashBackupCodes(codes)
	require.NoError(t, err)
	assert.NotEmpty(t, hashed)

	// Hashed should be different from original
	assert.NotEqual(t, codes[0], hashed)

	// Should contain comma separators
	assert.Contains(t, hashed, ",")

	// Parts should be bcrypt hashes (start with $2)
	parts := splitString(hashed, ",")
	for _, part := range parts {
		assert.Contains(t, part, "$2")
	}
}

// Helper function to split string
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func TestTOTPManager_VerifyBackupCode_Valid(t *testing.T) {
	totp := setupTOTPManager(t)

	codes := []string{"abc123def4567890", "xyz987uvw6543210"}

	hashed, err := totp.HashBackupCodes(codes)
	require.NoError(t, err)

	// Should verify one of the original codes
	assert.True(t, totp.VerifyBackupCode(hashed, "abc123def4567890"))
	assert.True(t, totp.VerifyBackupCode(hashed, "xyz987uvw6543210"))
}

func TestTOTPManager_VerifyBackupCode_Invalid(t *testing.T) {
	totp := setupTOTPManager(t)

	codes := []string{"abc123def4567890", "xyz987uvw6543210"}

	hashed, err := totp.HashBackupCodes(codes)
	require.NoError(t, err)

	// Should not verify wrong codes
	assert.False(t, totp.VerifyBackupCode(hashed, "wrongcode1234567"))
	assert.False(t, totp.VerifyBackupCode(hashed, ""))
	assert.False(t, totp.VerifyBackupCode(hashed, "abc123def456789")) // Wrong length
}

func TestWebAuthnUser(t *testing.T) {
	userID := []byte("test-user-id")
	user := &WebAuthnUser{
		ID:          userID,
		Name:        "testuser",
		DisplayName: "Test User",
		Credentials: []webauthn.Credential{},
	}

	// Test interface methods
	assert.Equal(t, userID, user.WebAuthnID())
	assert.Equal(t, "testuser", user.WebAuthnName())
	assert.Equal(t, "Test User", user.WebAuthnDisplayName())
	assert.Equal(t, "", user.WebAuthnIcon())
	assert.NotNil(t, user.WebAuthnCredentials())
	assert.Empty(t, user.WebAuthnCredentials())
}

func TestWebAuthnUser_WithCredentials(t *testing.T) {
	userID := []byte("test-user-id")
	cred := webauthn.Credential{
		ID: []byte("credential-id"),
	}

	user := &WebAuthnUser{
		ID:          userID,
		Name:        "testuser",
		DisplayName: "Test User",
		Credentials: []webauthn.Credential{cred},
	}

	creds := user.WebAuthnCredentials()
	assert.Len(t, creds, 1)
	assert.Equal(t, []byte("credential-id"), creds[0].ID)
}

func TestMFAManager_CheckRequired(t *testing.T) {
	mfa := NewMFAManager(nil, nil, nil, zerolog.Logger{})

	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "super admin",
			roles:    []string{"super_admin"},
			expected: true,
		},
		{
			name:     "admin",
			roles:    []string{"admin"},
			expected: true,
		},
		{
			name:     "operator",
			roles:    []string{"operator"},
			expected: true,
		},
		{
			name:     "regular user",
			roles:    []string{"user"},
			expected: false,
		},
		{
			name:     "auditor",
			roles:    []string{"auditor"},
			expected: false,
		},
		{
			name:     "mixed roles with admin",
			roles:    []string{"user", "admin"},
			expected: true,
		},
		{
			name:     "no roles",
			roles:    []string{},
			expected: false,
		},
		{
			name:     "case insensitive admin",
			roles:    []string{"Admin"},
			expected: false, // Case sensitive check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := mfa.CheckRequired(ctx, "user-id", tt.roles)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMFAManager_Verify_TOTP(t *testing.T) {
	totp := setupTOTPManager(t)
	mfa := NewMFAManager(totp, nil, nil, zerolog.Logger{})

	ctx := context.Background()
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now().Unix()
	counter := now / 30
	validCode := totp.generateCode(secret, counter)

	// Should verify valid TOTP code
	assert.True(t, mfa.Verify(ctx, "user-id", MFAMethodTOTP, secret, validCode))
	assert.False(t, mfa.Verify(ctx, "user-id", MFAMethodTOTP, secret, "000000"))
}

func TestMFAManager_Verify_Backup(t *testing.T) {
	totp := setupTOTPManager(t)
	mfa := NewMFAManager(totp, nil, nil, zerolog.Logger{})

	ctx := context.Background()
	codes := []string{"abc123def4567890"}
	hashed, _ := totp.HashBackupCodes(codes)

	// Should verify valid backup code
	assert.True(t, mfa.Verify(ctx, "user-id", MFAMethodBackup, hashed, "abc123def4567890"))
	assert.False(t, mfa.Verify(ctx, "user-id", MFAMethodBackup, hashed, "wrongcode"))
}

func TestMFAManager_Verify_UnknownMethod(t *testing.T) {
	mfa := NewMFAManager(nil, nil, nil, zerolog.Logger{})

	ctx := context.Background()

	// Unknown method should return false
	assert.False(t, mfa.Verify(ctx, "user-id", MFAMethod("unknown"), "secret", "response"))
}

func TestMFAMethod_String(t *testing.T) {
	tests := []struct {
		method   MFAMethod
		expected string
	}{
		{MFAMethodTOTP, "totp"},
		{MFAMethodWebAuthn, "webauthn"},
		{MFAMethodBackup, "backup"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.method))
		})
	}
}

// Test for edge cases
func TestTOTPManager_EdgeCases(t *testing.T) {
	totp := setupTOTPManager(t)

	t.Run("empty code", func(t *testing.T) {
		assert.False(t, totp.VerifyCode("JBSWY3DPEHPK3PXP", ""))
	})

	t.Run("code too short", func(t *testing.T) {
		assert.False(t, totp.VerifyCode("JBSWY3DPEHPK3PXP", "12345"))
	})

	t.Run("code too long", func(t *testing.T) {
		assert.False(t, totp.VerifyCode("JBSWY3DPEHPK3PXP", "1234567"))
	})

	t.Run("non-numeric code", func(t *testing.T) {
		assert.False(t, totp.VerifyCode("JBSWY3DPEHPK3PXP", "abcdef"))
	})
}

func TestTOTPConfig_DefaultValues(t *testing.T) {
	cfg := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	assert.Equal(t, "OpenPAM", cfg.Issuer)
	assert.Equal(t, "SHA256", cfg.Algorithm)
	assert.Equal(t, 6, cfg.Digits)
	assert.Equal(t, 30, cfg.Period)
	assert.Equal(t, 32, cfg.SecretSize)
	assert.Equal(t, 1, cfg.Leeway)
}

func TestTOTPCode_Format(t *testing.T) {
	totp := setupTOTPManager(t)
	secret := "JBSWY3DPEHPK3PXP"

	code := totp.generateCode(secret, 12345)

	// Code should be 6 digits
	assert.Len(t, code, 6)

	// Code should be numeric
	for _, c := range code {
		assert.GreaterOrEqual(t, c, '0')
		assert.LessOrEqual(t, c, '9')
	}
}

func TestWebAuthnConfig(t *testing.T) {
	cfg := WebAuthnConfig{
		RPDisplayName: "OpenPAM",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
	}

	assert.Equal(t, "OpenPAM", cfg.RPDisplayName)
	assert.Equal(t, "localhost", cfg.RPID)
	assert.Len(t, cfg.RPOrigins, 1)
	assert.Equal(t, "http://localhost:3000", cfg.RPOrigins[0])
}
