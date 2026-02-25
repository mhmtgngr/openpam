package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerateRSAKeys(t *testing.T) {
	t.Run("generates new keys when files don't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "test_private.pem")
		publicKeyPath := filepath.Join(tmpDir, "test_public.pem")
		logger := zerolog.Nop()

		privateKey, publicKey, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		require.NoError(t, err)
		assert.NotNil(t, privateKey)
		assert.NotNil(t, publicKey)
		assert.IsType(t, &rsa.PrivateKey{}, privateKey)
		assert.IsType(t, &rsa.PublicKey{}, publicKey)

		// Verify key size
		assert.Equal(t, 2048, privateKey.N.BitLen())

		// Verify files were created
		_, err = os.Stat(privateKeyPath)
		assert.NoError(t, err)
		_, err = os.Stat(publicKeyPath)
		assert.NoError(t, err)
	})

	t.Run("loads existing keys when files exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "test_private.pem")
		publicKeyPath := filepath.Join(tmpDir, "test_public.pem")
		logger := zerolog.Nop()

		// Generate keys first
		originalPrivate, originalPublic, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)
		require.NoError(t, err)

		// Load them again
		loadedPrivate, loadedPublic, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		require.NoError(t, err)
		assert.Equal(t, originalPrivate.D, loadedPrivate.D)
		assert.Equal(t, originalPublic.N, loadedPublic.N)
	})

	t.Run("regenerates keys when private key exists but public key is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "test_private.pem")
		publicKeyPath := filepath.Join(tmpDir, "test_public.pem")
		logger := zerolog.Nop()

		// Generate keys first
		originalPrivate, _, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)
		require.NoError(t, err)

		// Delete public key
		err = os.Remove(publicKeyPath)
		require.NoError(t, err)

		// This should regenerate the keys
		newPrivate, newPublic, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		require.NoError(t, err)
		assert.NotNil(t, newPrivate)
		assert.NotNil(t, newPublic)

		// Keys should be different since they were regenerated
		assert.NotEqual(t, originalPrivate.D, newPrivate.D)
	})

	t.Run("regenerates keys when public key exists but private key is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "test_private.pem")
		publicKeyPath := filepath.Join(tmpDir, "test_public.pem")
		logger := zerolog.Nop()

		// Generate keys first
		originalPublic, _, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)
		require.NoError(t, err)

		// Delete private key
		err = os.Remove(privateKeyPath)
		require.NoError(t, err)

		// This should regenerate the keys
		newPrivate, newPublic, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		require.NoError(t, err)
		assert.NotNil(t, newPrivate)
		assert.NotNil(t, newPublic)

		// Keys should be different since they were regenerated
		assert.NotEqual(t, originalPublic.N, newPublic.N)
	})

	t.Run("creates parent directories for key files", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir", "nested")
		privateKeyPath := filepath.Join(subDir, "test_private.pem")
		publicKeyPath := filepath.Join(subDir, "test_public.pem")
		logger := zerolog.Nop()

		privateKey, publicKey, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		require.NoError(t, err)
		assert.NotNil(t, privateKey)
		assert.NotNil(t, publicKey)

		// Verify files were created in the nested directory
		_, err = os.Stat(privateKeyPath)
		assert.NoError(t, err)
		_, err = os.Stat(publicKeyPath)
		assert.NoError(t, err)
	})

	t.Run("handles file creation errors gracefully", func(t *testing.T) {
		// Use an invalid path (directory instead of file)
		tmpDir := t.TempDir()
		privateKeyPath := tmpDir // This is a directory, not a file
		publicKeyPath := filepath.Join(tmpDir, "test_public.pem")
		logger := zerolog.Nop()

		privateKey, publicKey, err := LoadOrGenerateRSAKeys(privateKeyPath, publicKeyPath, logger)

		// Should still generate keys in memory even if saving fails
		assert.NoError(t, err)
		assert.NotNil(t, privateKey)
		assert.NotNil(t, publicKey)
	})
}

func TestLoadPrivateKey(t *testing.T) {
	t.Run("loads valid RSA private key from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "private.pem")

		// Generate and save a private key
		generateTestRSAPrivateKeyToFile(t, privateKeyPath)

		loadedKey, err := loadPrivateKey(privateKeyPath)

		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.NotNil(t, loadedKey.D)
	})

	t.Run("returns error when file doesn't exist", func(t *testing.T) {
		_, err := loadPrivateKey("/nonexistent/path/key.pem")

		assert.Error(t, err)
	})

	t.Run("returns error for invalid key data", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "invalid.pem")

		err := os.WriteFile(keyPath, []byte("not a valid PEM key"), 0600)
		require.NoError(t, err)

		_, err = loadPrivateKey(keyPath)

		assert.Error(t, err)
	})
}

func TestLoadPublicKey(t *testing.T) {
	t.Run("loads valid RSA public key from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "private.pem")
		publicKeyPath := filepath.Join(tmpDir, "public.pem")

		// Generate and save a key pair
		privateKey := generateTestRSAPrivateKeyToFile(t, privateKeyPath)

		// Save public key
		pemData, err := crypto.MarshalRSAPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)
		err = os.WriteFile(publicKeyPath, pemData, 0644)
		require.NoError(t, err)

		loadedKey, err := loadPublicKey(publicKeyPath)

		require.NoError(t, err)
		assert.NotNil(t, loadedKey)
		assert.Equal(t, privateKey.PublicKey.N, loadedKey.N)
		assert.Equal(t, privateKey.PublicKey.E, loadedKey.E)
	})

	t.Run("returns error when file doesn't exist", func(t *testing.T) {
		_, err := loadPublicKey("/nonexistent/path/key.pem")

		assert.Error(t, err)
	})

	t.Run("returns error for invalid key data", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "invalid.pem")

		err := os.WriteFile(keyPath, []byte("not a valid PEM key"), 0600)
		require.NoError(t, err)

		_, err = loadPublicKey(keyPath)

		assert.Error(t, err)
	})
}

func TestSavePrivateKey(t *testing.T) {
	t.Run("saves RSA private key to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "private.pem")

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		err = savePrivateKey(keyPath, key)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(keyPath)
		assert.NoError(t, err)

		// Verify we can load it back
		loadedKey, err := loadPrivateKey(keyPath)
		require.NoError(t, err)
		assert.Equal(t, key.D, loadedKey.D)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedPath := filepath.Join(tmpDir, "nested", "dir", "private.pem")

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		err = savePrivateKey(nestedPath, key)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(nestedPath)
		assert.NoError(t, err)
	})
}

func TestSavePublicKey(t *testing.T) {
	t.Run("saves RSA public key to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "public.pem")

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		err = savePublicKey(keyPath, &key.PublicKey)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(keyPath)
		assert.NoError(t, err)

		// Verify we can load it back
		loadedKey, err := loadPublicKey(keyPath)
		require.NoError(t, err)
		assert.Equal(t, key.PublicKey.N, loadedKey.N)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		nestedPath := filepath.Join(tmpDir, "nested", "dir", "public.pem")

		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		err = savePublicKey(nestedPath, &key.PublicKey)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(nestedPath)
		assert.NoError(t, err)
	})
}

func TestFilepathDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "extracts directory from simple path",
			path:     "/path/to/file.txt",
			expected: "/path/to",
		},
		{
			name:     "extracts directory from nested path",
			path:     "/a/b/c/d/e/file.txt",
			expected: "/a/b/c/d/e",
		},
		{
			name:     "handles path with trailing slash",
			path:     "/path/to/dir/",
			expected: "/path/to/dir",
		},
		{
			name:     "handles relative path",
			path:     "relative/path/file.txt",
			expected: "relative/path",
		},
		{
			name:     "handles single filename",
			path:     "file.txt",
			expected: "",
		},
		{
			name:     "handles root path",
			path:     "/file.txt",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepathDir(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// generateTestRSAPrivateKeyToFile generates a test RSA private key and saves it to file
func generateTestRSAPrivateKeyToFile(t *testing.T, privateKeyPath string) *rsa.PrivateKey {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pemData, err := crypto.MarshalRSAPrivateKey(privateKey)
	require.NoError(t, err)
	err = os.WriteFile(privateKeyPath, pemData, 0600)
	require.NoError(t, err)

	return privateKey
}
