package kms

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMasterKeyManager(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name       string
		provider   KeyManager
		masterKeyID string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid inputs",
			provider:   NewInMemoryKeyManager(logger),
			masterKeyID: "test-key-id",
			wantErr:    false,
		},
		{
			name:       "nil provider",
			provider:   nil,
			masterKeyID: "test-key-id",
			wantErr:    true,
			errMsg:     "provider cannot be nil",
		},
		{
			name:       "empty master key ID",
			provider:   NewInMemoryKeyManager(logger),
			masterKeyID: "",
			wantErr:    true,
			errMsg:     "masterKeyID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mkm, err := NewMasterKeyManager(tt.provider, tt.masterKeyID, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, mkm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mkm)
				assert.Equal(t, tt.masterKeyID, mkm.masterKeyID)
				assert.Equal(t, 1, mkm.currentVersion)
				assert.NotNil(t, mkm.cache)
			}
		})
	}
}

func TestMasterKeyManager_GetMasterKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("gets key from provider and caches it", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "test-master-key"
		expectedKey := make([]byte, 32)
		expectedKey[0] = 0xAA
		provider.SetKey(masterKeyID, expectedKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		// First call should fetch from provider
		key, err := mkm.GetMasterKey(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedKey, key)

		// Second call should use cache
		key2, err := mkm.GetMasterKey(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedKey, key2)
		assert.Equal(t, key, key2)
	})

	t.Run("returns error when key not found", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		mkm, err := NewMasterKeyManager(provider, "nonexistent-key", logger)
		require.NoError(t, err)

		_, err = mkm.GetMasterKey(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found")
	})

	t.Run("returns error for invalid key size", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "bad-key"
		invalidKey := make([]byte, 16) // Wrong size
		provider.SetKey(masterKeyID, invalidKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		_, err = mkm.GetMasterKey(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid master key size")
	})
}

func TestMasterKeyManager_WrapKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("wraps a data encryption key", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "test-master-key"
		masterKey := make([]byte, 32)
		provider.SetKey(masterKeyID, masterKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		dek := []byte("test-data-encryption-key")
		wrapped, err := mkm.WrapKey(ctx, dek)

		require.NoError(t, err)
		assert.NotNil(t, wrapped)
		assert.Equal(t, masterKeyID, wrapped.KeyID)
		assert.Equal(t, 1, wrapped.KeyVersion)
		assert.Equal(t, "AES256_GCM", wrapped.EncryptionAlg)
		assert.NotEmpty(t, wrapped.WrappedKey)
		assert.NotEqual(t, dek, wrapped.WrappedKey)
	})

	t.Run("wrapping different keys produces different results", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "test-master-key"
		masterKey := make([]byte, 32)
		provider.SetKey(masterKeyID, masterKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		dek1 := []byte("key-1")
		dek2 := []byte("key-2")

		wrapped1, err := mkm.WrapKey(ctx, dek1)
		require.NoError(t, err)

		wrapped2, err := mkm.WrapKey(ctx, dek2)
		require.NoError(t, err)

		assert.NotEqual(t, wrapped1.WrappedKey, wrapped2.WrappedKey)
	})
}

func TestMasterKeyManager_UnwrapKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("unwraps a wrapped data encryption key", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "test-master-key"
		masterKey := make([]byte, 32)
		provider.SetKey(masterKeyID, masterKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		originalDEK := []byte("test-data-encryption-key")
		wrapped, err := mkm.WrapKey(ctx, originalDEK)
		require.NoError(t, err)

		unwrapped, err := mkm.UnwrapKey(ctx, wrapped)
		require.NoError(t, err)
		assert.Equal(t, originalDEK, unwrapped)
	})

	t.Run("returns error for nil wrapped key", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		mkm, err := NewMasterKeyManager(provider, "test-key", logger)
		require.NoError(t, err)

		_, err = mkm.UnwrapKey(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrapped key cannot be nil")
	})

	t.Run("returns error for invalid wrapped key", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		mkm, err := NewMasterKeyManager(provider, "test-key", logger)
		require.NoError(t, err)

		invalidWrapped := &WrappedKey{
			KeyID:         "nonexistent-key",
			WrappedKey:    []byte("invalid-ciphertext"),
			KeyVersion:    1,
			EncryptionAlg: "AES256_GCM",
		}

		_, err = mkm.UnwrapKey(ctx, invalidWrapped)
		assert.Error(t, err)
	})
}

func TestMasterKeyManager_RotateMasterKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("rotates master key and clears cache", func(t *testing.T) {
		provider := NewInMemoryKeyManager(logger)
		masterKeyID := "test-master-key"
		masterKey := make([]byte, 32)
		provider.SetKey(masterKeyID, masterKey)

		mkm, err := NewMasterKeyManager(provider, masterKeyID, logger)
		require.NoError(t, err)

		// Get key to populate cache
		_, err = mkm.GetMasterKey(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, mkm.cache)

		// Rotate key
		err = mkm.RotateMasterKey(ctx)
		require.NoError(t, err)

		// Verify cache is cleared
		assert.Empty(t, mkm.cache)

		// Verify version incremented
		assert.Equal(t, 2, mkm.currentVersion)

		// Verify key ID changed
		assert.NotEqual(t, masterKeyID, mkm.masterKeyID)
		assert.Contains(t, mkm.masterKeyID, "-v2")
	})
}

func TestGenerateDEK(t *testing.T) {
	t.Run("generates a valid data encryption key", func(t *testing.T) {
		dek, err := GenerateDEK()

		require.NoError(t, err)
		assert.Len(t, dek, 32, "DEK should be 32 bytes for AES-256")
	})

	t.Run("generates unique keys", func(t *testing.T) {
		dek1, err := GenerateDEK()
		require.NoError(t, err)

		dek2, err := GenerateDEK()
		require.NoError(t, err)

		assert.NotEqual(t, dek1, dek2, "Each DEK should be unique")
	})
}

func TestInMemoryKeyManager_GenerateKey(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("generates a new key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"

		key, err := im.GenerateKey(keyID)

		require.NoError(t, err)
		assert.Len(t, key, 32)
		assert.NotNil(t, im.keys[keyID])
	})

	t.Run("generates unique keys", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)

		key1, err := im.GenerateKey("key1")
		require.NoError(t, err)

		key2, err := im.GenerateKey("key2")
		require.NoError(t, err)

		assert.NotEqual(t, key1, key2)
	})
}

func TestInMemoryKeyManager_SetKey(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("sets a key in memory", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := []byte{0x01, 0x02, 0x03}

		im.SetKey(keyID, key)

		assert.Equal(t, key, im.keys[keyID])
	})
}

func TestInMemoryKeyManager_GetKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("gets existing key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := []byte{0x01, 0x02, 0x03}
		im.keys[keyID] = key

		retrieved, err := im.GetKey(ctx, keyID)

		require.NoError(t, err)
		assert.Equal(t, key, retrieved)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)

		_, err := im.GetKey(ctx, "nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found")
	})
}

func TestInMemoryKeyManager_EncryptData(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("encrypts data with existing key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := make([]byte, 32)
		im.keys[keyID] = key

		plaintext := []byte("sensitive data")
		ciphertext, err := im.EncryptData(ctx, keyID, plaintext)

		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext)
		assert.Greater(t, len(ciphertext), len(plaintext), "Ciphertext should include nonce and tag")
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)

		_, err := im.EncryptData(ctx, "nonexistent", []byte("data"))

		assert.Error(t, err)
	})
}

func TestInMemoryKeyManager_DecryptData(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("decrypts previously encrypted data", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := make([]byte, 32)
		im.keys[keyID] = key

		plaintext := []byte("sensitive data")
		ciphertext, err := im.EncryptData(ctx, keyID, plaintext)
		require.NoError(t, err)

		decrypted, err := im.DecryptData(ctx, keyID, ciphertext)

		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)

		_, err := im.DecryptData(ctx, "nonexistent", []byte("ciphertext"))

		assert.Error(t, err)
	})

	t.Run("returns error for invalid ciphertext", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := make([]byte, 32)
		im.keys[keyID] = key

		invalidCiphertext := []byte{0x01, 0x02} // Too short

		_, err := im.DecryptData(ctx, keyID, invalidCiphertext)

		assert.Error(t, err)
	})
}

func TestInMemoryKeyManager_RotateKey(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("rotates a key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := make([]byte, 32)
		im.keys[keyID] = key

		newKeyID, err := im.RotateKey(ctx, keyID)

		require.NoError(t, err)
		assert.Equal(t, keyID+"-v2", newKeyID)
		assert.Contains(t, im.keys, newKeyID)
		assert.Len(t, im.keys[newKeyID], 32)
	})
}

func TestInMemoryKeyManager_ScheduleKeyDeletion(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	t.Run("deletes a key", func(t *testing.T) {
		im := NewInMemoryKeyManager(logger)
		keyID := "test-key"
		key := make([]byte, 32)
		im.keys[keyID] = key

		err := im.ScheduleKeyDeletion(ctx, keyID, 30)

		require.NoError(t, err)
		assert.NotContains(t, im.keys, keyID)
	})
}

func TestWrappedKeySerializer_Serialize(t *testing.T) {
	ws := &WrappedKeySerializer{}

	t.Run("serializes a wrapped key", func(t *testing.T) {
		wk := &WrappedKey{
			KeyID:         "test-key-id",
			WrappedKey:    []byte{0x01, 0x02, 0x03, 0x04},
			KeyVersion:    5,
			EncryptionAlg: "AES256_GCM",
		}

		serialized, err := ws.Serialize(wk)

		require.NoError(t, err)
		assert.NotEmpty(t, serialized)
		assert.Contains(t, serialized, ".") // Should have separators
		assert.NotContains(t, serialized, "test-key-id") // KeyID should be base64 encoded
	})

	t.Run("returns error for nil wrapped key", func(t *testing.T) {
		_, err := ws.Serialize(nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "wrapped key is nil")
	})
}

func TestWrappedKeySerializer_Deserialize(t *testing.T) {
	ws := &WrappedKeySerializer{}

	t.Run("deserializes a valid wrapped key", func(t *testing.T) {
		original := &WrappedKey{
			KeyID:         "test-key-id",
			WrappedKey:    []byte{0x01, 0x02, 0x03, 0x04},
			KeyVersion:    5,
			EncryptionAlg: "AES256_GCM",
		}

		serialized, err := ws.Serialize(original)
		require.NoError(t, err)

		deserialized, err := ws.Deserialize(serialized)

		require.NoError(t, err)
		assert.Equal(t, original.KeyID, deserialized.KeyID)
		assert.Equal(t, original.WrappedKey, deserialized.WrappedKey)
		assert.Equal(t, original.KeyVersion, deserialized.KeyVersion)
		assert.Equal(t, original.EncryptionAlg, deserialized.EncryptionAlg)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		invalidFormats := []string{
			"",
			"only-one-part",
			"two.parts",
		}

		for _, invalid := range invalidFormats {
			_, err := ws.Deserialize(invalid)
			assert.Error(t, err)
		}
	})

	t.Run("returns error for invalid base64", func(t *testing.T) {
		invalid := "!!!invalid-base64!!!.!!!also-invalid!!!.1.AES256_GCM"

		_, err := ws.Deserialize(invalid)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

func TestWrappedKeySerializer_RoundTrip(t *testing.T) {
	ws := &WrappedKeySerializer{}

	tests := []struct {
		name string
		wk   *WrappedKey
	}{
		{
			name: "standard wrapped key",
			wk: &WrappedKey{
				KeyID:         "master-key-123",
				WrappedKey:    []byte("encrypted-key-material"),
				KeyVersion:    1,
				EncryptionAlg: "AES256_GCM",
			},
		},
		{
			name: "wrapped key with special characters in ID",
			wk: &WrappedKey{
				KeyID:         "key/with/special@chars#123",
				WrappedKey:    []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
				KeyVersion:    10,
				EncryptionAlg: "AES256_GCM",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := ws.Serialize(tt.wk)
			require.NoError(t, err)

			deserialized, err := ws.Deserialize(serialized)
			require.NoError(t, err)

			assert.Equal(t, tt.wk.KeyID, deserialized.KeyID)
			assert.Equal(t, tt.wk.WrappedKey, deserialized.WrappedKey)
			assert.Equal(t, tt.wk.KeyVersion, deserialized.KeyVersion)
			assert.Equal(t, tt.wk.EncryptionAlg, deserialized.EncryptionAlg)
		})
	}
}
