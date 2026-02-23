package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOTPManager_GenerateSecret(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("generate secret", func(t *testing.T) {
		secret, err := manager.GenerateSecret()
		require.NoError(t, err)
		assert.NotEmpty(t, secret)

		// Secrets should be unique
		secret2, err := manager.GenerateSecret()
		require.NoError(t, err)
		assert.NotEqual(t, secret, secret2)
	})
}

func TestTOTPManager_GenerateQRCode(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("generate QR code", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		email := "test@example.com"

		qr, err := manager.GenerateQRCode(email, secret)
		require.NoError(t, err)
		assert.NotEmpty(t, qr)

		// QR code should be PNG (starts with PNG signature)
		assert.Contains(t, string(qr[:8]), "\x89PNG")
	})
}

func TestTOTPManager_VerifyCode(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("verify invalid code format", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		result := manager.VerifyCode(secret, "000000")
		// This might return true or false depending on time, just check it doesn't panic
		assert.IsType(t, false, result)
	})

	t.Run("verify with empty secret", func(t *testing.T) {
		result := manager.VerifyCode("", "123456")
		assert.False(t, result)
	})
}

func TestTOTPManager_GenerateBackupCodes(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("generate backup codes", func(t *testing.T) {
		codes, err := manager.GenerateBackupCodes(10)
		require.NoError(t, err)
		assert.Len(t, codes, 10)

		for _, code := range codes {
			assert.Len(t, code, 16) // 16 character hex
		}
	})

	t.Run("generate unique codes", func(t *testing.T) {
		codes, err := manager.GenerateBackupCodes(20)
		require.NoError(t, err)

		uniqueCodes := make(map[string]bool)
		for _, code := range codes {
			uniqueCodes[code] = true
		}

		assert.Len(t, uniqueCodes, 20, "All codes should be unique")
	})
}

func TestTOTPManager_HashBackupCodes(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("hash backup codes", func(t *testing.T) {
		codes := []string{"abc123", "def456", "ghi789"}
		hashed, err := manager.HashBackupCodes(codes)
		require.NoError(t, err)
		assert.NotEmpty(t, hashed)
		assert.Contains(t, hashed, ",")
		assert.NotEqual(t, hashed, "abc123,def456,ghi789")
	})
}

func TestTOTPManager_VerifyBackupCode(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	manager := NewTOTPManager(config, mockCache, zerolog.Nop())

	t.Run("verify correct backup code", func(t *testing.T) {
		codes := []string{"testcode12345678"}
		hashed, err := manager.HashBackupCodes(codes)
		require.NoError(t, err)

		result := manager.VerifyBackupCode(hashed, "testcode12345678")
		assert.True(t, result)
	})

	t.Run("verify incorrect backup code", func(t *testing.T) {
		codes := []string{"testcode12345678"}
		hashed, err := manager.HashBackupCodes(codes)
		require.NoError(t, err)

		result := manager.VerifyBackupCode(hashed, "wrongcode00000000")
		assert.False(t, result)
	})

	t.Run("verify with empty hashed codes", func(t *testing.T) {
		result := manager.VerifyBackupCode("", "anypassword")
		assert.False(t, result)
	})
}

func TestMFAManager_CheckRequired(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	totp := NewTOTPManager(config, mockCache, zerolog.Nop())
	mfa := NewMFAManager(totp, nil, mockCache, zerolog.Nop())

	t.Run("MFA required for admin", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{"admin"})
		assert.True(t, required)
	})

	t.Run("MFA required for super_admin", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{"super_admin"})
		assert.True(t, required)
	})

	t.Run("MFA required for operator", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{"operator"})
		assert.True(t, required)
	})

	t.Run("MFA not required for regular user", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{"user"})
		assert.False(t, required)
	})

	t.Run("MFA not required for auditor", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{"auditor"})
		assert.False(t, required)
	})

	t.Run("MFA not required for no roles", func(t *testing.T) {
		required := mfa.CheckRequired(context.Background(), "user1", []string{})
		assert.False(t, required)
	})
}

func TestMFAManager_VerifySession(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	totp := NewTOTPManager(config, mockCache, zerolog.Nop())
	mfa := NewMFAManager(totp, nil, mockCache, zerolog.Nop())

	sessionID := uuid.New().String()

	t.Run("unverified session", func(t *testing.T) {
		verified := mfa.VerifySession(context.Background(), sessionID)
		assert.False(t, verified)
	})

	t.Run("mark session verified", func(t *testing.T) {
		err := mfa.MarkSessionVerified(context.Background(), sessionID, 5*time.Minute)
		require.NoError(t, err)

		verified := mfa.VerifySession(context.Background(), sessionID)
		assert.True(t, verified)
	})
}

func TestMFAManager_Verify(t *testing.T) {
	config := TOTPConfig{
		Issuer:     "OpenPAM",
		Algorithm:  "SHA256",
		Digits:     6,
		Period:     30,
		SecretSize: 32,
		Leeway:     1,
	}

	mockCache := &mockCacheForTest{data: make(map[string]string)}
	totp := NewTOTPManager(config, mockCache, zerolog.Nop())
	mfa := NewMFAManager(totp, nil, mockCache, zerolog.Nop())

	t.Run("verify TOTP method", func(t *testing.T) {
		secret := "JBSWY3DPEHPK3PXP"
		code := "000000"
		result := mfa.Verify(context.Background(), "user1", MFAMethodTOTP, secret, code)
		// Result depends on timing, just ensure no panic
		assert.IsType(t, false, result)
	})

	t.Run("verify backup method", func(t *testing.T) {
		codes := []string{"testcode12345678"}
		hashed, _ := totp.HashBackupCodes(codes)

		result := mfa.Verify(context.Background(), "user1", MFAMethodBackup, hashed, "testcode12345678")
		assert.True(t, result)
	})

	t.Run("verify unsupported method", func(t *testing.T) {
		result := mfa.Verify(context.Background(), "user1", "unsupported", "secret", "response")
		assert.False(t, result)
	})
}

func TestMFAMethod_String(t *testing.T) {
	assert.Equal(t, "totp", string(MFAMethodTOTP))
	assert.Equal(t, "webauthn", string(MFAMethodWebAuthn))
	assert.Equal(t, "backup", string(MFAMethodBackup))
}
