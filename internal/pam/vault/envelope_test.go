package vault

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/testing"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVaultTestDB(t *testing.T) *sqlx.DB {
	db := testing.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil
	}

	// Create vault-specific tables
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS encrypted_deks (
			id UUID PRIMARY KEY,
			credential_id TEXT NOT NULL UNIQUE,
			key_id TEXT NOT NULL,
			encrypted_key BYTEA NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS key_metadata (
			key_id TEXT PRIMARY KEY,
			key_version INTEGER NOT NULL,
			algorithm TEXT NOT NULL,
			key_size INTEGER NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			rotated_at TIMESTAMP
		);
	`)
	require.NoError(t, err)

	return db
}

func TestNewEnvelopeEncryption(t *testing.T) {
	t.Run("valid key size", func(t *testing.T) {
		key := make([]byte, 32)
		db := setupVaultTestDB(t)
		if db == nil {
			return
		}

		mockCache := testing.NewMockCache()
		ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
		require.NoError(t, err)
		assert.NotNil(t, ee)
	})

	t.Run("invalid key size", func(t *testing.T) {
		key := make([]byte, 16)
		db := setupVaultTestDB(t)
		if db == nil {
			return
		}

		mockCache := testing.NewMockCache()
		_, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "master key must be 32 bytes")
	})
}

func TestEnvelopeEncryption_GenerateDEK(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	dek1, err := ee.GenerateDEK()
	require.NoError(t, err)
	assert.Len(t, dek1, 32)

	// Verify uniqueness
	dek2, err := ee.GenerateDEK()
	require.NoError(t, err)
	assert.NotEqual(t, dek1, dek2)
}

func TestEnvelopeEncryption_EncryptDecryptDEK(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	dek, err := ee.GenerateDEK()
	require.NoError(t, err)

	t.Run("encrypt DEK", func(t *testing.T) {
		encrypted, err := ee.EncryptDEK(dek)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, dek, encrypted)
		assert.GreaterOrEqual(t, len(encrypted), 12+16) // nonce + tag + data
	})

	t.Run("decrypt DEK", func(t *testing.T) {
		encrypted, err := ee.EncryptDEK(dek)
		require.NoError(t, err)

		decrypted, err := ee.DecryptDEK(encrypted)
		require.NoError(t, err)
		assert.Equal(t, dek, decrypted)
	})

	t.Run("decrypt invalid ciphertext", func(t *testing.T) {
		invalid := []byte("too short")
		_, err := ee.DecryptDEK(invalid)
		assert.Error(t, err)
	})

	t.Run("decrypt modified ciphertext", func(t *testing.T) {
		encrypted, err := ee.EncryptDEK(dek)
		require.NoError(t, err)

		// Modify ciphertext
		encrypted[0] ^= 0xFF

		_, err = ee.DecryptDEK(encrypted)
		assert.Error(t, err)
	})
}

func TestEnvelopeEncryption_StoreDEK(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)
	credentialID := uuid.New().String()
	dek, _ := ee.GenerateDEK()
	encrypted, _ := ee.EncryptDEK(dek)

	t.Run("store DEK", func(t *testing.T) {
		err := ee.StoreDEK(ctx, credentialID, encrypted)
		require.NoError(t, err)

		// Verify in database
		var storedKey string
		err = db.QueryRowContext(ctx, "SELECT key_id FROM encrypted_deks WHERE credential_id = $1", credentialID).Scan(&storedKey)
		require.NoError(t, err)
		assert.Equal(t, "key-1", storedKey)
	})

	t.Run("update existing DEK", func(t *testing.T) {
		newEncrypted, _ := ee.EncryptDEK(dek)

		err := ee.StoreDEK(ctx, credentialID, newEncrypted)
		require.NoError(t, err)
	})
}

func TestEnvelopeEncryption_RetrieveDEK(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)
	credentialID := uuid.New().String()
	dek, _ := ee.GenerateDEK()
	encrypted, _ := ee.EncryptDEK(dek)

	// Store in database
	_, err = db.ExecContext(ctx, `
		INSERT INTO encrypted_deks (id, credential_id, key_id, encrypted_key, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, uuid.New(), credentialID, "key-1", encrypted)
	require.NoError(t, err)

	t.Run("retrieve existing DEK", func(t *testing.T) {
		retrieved, err := ee.RetrieveDEK(ctx, credentialID)
		require.NoError(t, err)
		assert.Equal(t, dek, retrieved)
	})

	t.Run("retrieve non-existent DEK", func(t *testing.T) {
		_, err := ee.RetrieveDEK(ctx, "non-existent")
		assert.Error(t, err)
	})

	t.Run("cache hit", func(t *testing.T) {
		// First call retrieves from DB and caches
		_, err := ee.RetrieveDEK(ctx, credentialID)
		require.NoError(t, err)

		// Second call should use cache
		_, err = ee.RetrieveDEK(ctx, credentialID)
		require.NoError(t, err)
	})
}

func TestEnvelopeEncryption_RotateMasterKey(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	oldKey := make([]byte, 32)
	newKey := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(oldKey, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)

	// Create some test DEKs
	credentialIDs := []string{"cred-1", "cred-2", "cred-3"}
	for _, credID := range credentialIDs {
		dek, _ := ee.GenerateDEK()
		encrypted, _ := ee.EncryptDEK(dek)

		_, err = db.ExecContext(ctx, `
			INSERT INTO encrypted_deks (id, credential_id, key_id, encrypted_key, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), credID, "key-1", encrypted)
		require.NoError(t, err)
	}

	t.Run("rotate master key", func(t *testing.T) {
		err := ee.RotateMasterKey(ctx, newKey, "key-2")
		require.NoError(t, err)

		// Verify current key updated
		assert.Equal(t, newKey, ee.masterKey)
		assert.Equal(t, "key-2", ee.currentKeyID)

		// Verify all DEKs re-encrypted with new key
		for _, credID := range credentialIDs {
			var keyID string
			err = db.QueryRowContext(ctx, "SELECT key_id FROM encrypted_deks WHERE credential_id = $1", credID).Scan(&keyID)
			require.NoError(t, err)
			assert.Equal(t, "key-2", keyID)
		}
	})
}

func TestEnvelopeEncryption_StoreKeyMetadata(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)

	metadata := KeyMetadata{
		KeyID:      "key-1",
		KeyVersion: 1,
		Algorithm:  "AES-256-GCM",
		KeySize:    256,
		IsActive:   true,
	}

	err = ee.StoreKeyMetadata(ctx, metadata)
	require.NoError(t, err)

	// Verify in database
	var algo string
	err = db.QueryRowContext(ctx, "SELECT algorithm FROM key_metadata WHERE key_id = $1", "key-1").Scan(&algo)
	require.NoError(t, err)
	assert.Equal(t, "AES-256-GCM", algo)
}

func TestEnvelopeEncryption_GetCurrentKeyMetadata(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)

	// Insert test metadata
	_, err = db.ExecContext(ctx, `
		INSERT INTO key_metadata (key_id, key_version, algorithm, key_size, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, "key-1", 1, "AES-256-GCM", 256, true)
	require.NoError(t, err)

	metadata, err := ee.GetCurrentKeyMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, "key-1", metadata.KeyID)
	assert.Equal(t, 1, metadata.KeyVersion)
	assert.Equal(t, "AES-256-GCM", metadata.Algorithm)
	assert.Equal(t, 256, metadata.KeySize)
	assert.True(t, metadata.IsActive)
}

func TestEnvelopeEncryption_DecryptWithKey(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	dek, _ := ee.GenerateDEK()
	encrypted, _ := ee.EncryptDEK(dek)

	t.Run("decrypt with same key", func(t *testing.T) {
		decrypted, err := ee.decryptWithKey(encrypted, key)
		require.NoError(t, err)
		assert.Equal(t, dek, decrypted)
	})

	t.Run("decrypt with different key", func(t *testing.T) {
		differentKey := make([]byte, 32)
		differentKey[0] = 0xFF // Make it different

		_, err := ee.decryptWithKey(encrypted, differentKey)
		assert.Error(t, err)
	})
}

func TestEnvelopeEncryption_EncryptWithKey(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	key := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(key, "key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	dek, _ := ee.GenerateDEK()

	t.Run("encrypt with key", func(t *testing.T) {
		encrypted, err := ee.encryptWithKey(dek, key)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, dek, encrypted)

		// Verify we can decrypt
		decrypted, err := ee.decryptWithKey(encrypted, key)
		require.NoError(t, err)
		assert.Equal(t, dek, decrypted)
	})
}

func TestEnvelopeEncryption_Integration(t *testing.T) {
	db := setupVaultTestDB(t)
	if db == nil {
		return
	}

	// This test simulates the full envelope encryption workflow
	masterKey := make([]byte, 32)
	mockCache := testing.NewMockCache()
	ee, err := NewEnvelopeEncryption(masterKey, "master-key-1", db, mockCache, zerolog.Nop())
	require.NoError(t, err)

	ctx := testing.Context(t)

	// 1. Store key metadata
	metadata := KeyMetadata{
		KeyID:      "master-key-1",
		KeyVersion: 1,
		Algorithm:  "AES-256-GCM",
		KeySize:    256,
		IsActive:   true,
	}
	err = ee.StoreKeyMetadata(ctx, metadata)
	require.NoError(t, err)

	// 2. Generate and encrypt a DEK
	dek, err := ee.GenerateDEK()
	require.NoError(t, err)

	encryptedDEK, err := ee.EncryptDEK(dek)
	require.NoError(t, err)

	// 3. Store encrypted DEK
	credentialID := uuid.New().String()
	err = ee.StoreDEK(ctx, credentialID, encryptedDEK)
	require.NoError(t, err)

	// 4. Retrieve and decrypt DEK
	retrievedDEK, err := ee.RetrieveDEK(ctx, credentialID)
	require.NoError(t, err)
	assert.Equal(t, dek, retrievedDEK)

	// 5. Use DEK to encrypt actual data (simulate)
	// In real usage, this would encrypt credential data
	assert.Len(t, retrievedDEK, 32)

	// 6. Verify key metadata
	storedMetadata, err := ee.GetCurrentKeyMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, "AES-256-GCM", storedMetadata.Algorithm)
}

func BenchmarkEnvelopeEncryption_GenerateDEK(b *testing.B) {
	key := make([]byte, 32)
	ee, _ := NewEnvelopeEncryption(key, "key-1", nil, nil, zerolog.Nop())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ee.GenerateDEK()
	}
}

func BenchmarkEnvelopeEncryption_EncryptDEK(b *testing.B) {
	key := make([]byte, 32)
	ee, _ := NewEnvelopeEncryption(key, "key-1", nil, nil, zerolog.Nop())
	dek, _ := ee.GenerateDEK()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ee.EncryptDEK(dek)
	}
}

func BenchmarkEnvelopeEncryption_DecryptDEK(b *testing.B) {
	key := make([]byte, 32)
	ee, _ := NewEnvelopeEncryption(key, "key-1", nil, nil, zerolog.Nop())
	dek, _ := ee.GenerateDEK()
	encrypted, _ := ee.EncryptDEK(dek)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ee.DecryptDEK(encrypted)
	}
}
