// Package kms provides Key Management Service abstractions for secure key handling.
// SECURITY: This package implements the master key hierarchy using KMS-wrapped keys.
//
// In production, use one of:
// 1. AWS KMS (https://docs.aws.amazon.com/kms/)
// 2. Google Cloud KMS (https://cloud.google.com/kms)
// 3. Azure Key Vault (https://azure.microsoft.com/en-us/services/key-vault/)
// 4. HashiCorp Vault (https://www.vaultproject.io/)
// 5. Hardware Security Module (HSM)
//
// NEVER pass raw master keys via environment variables or configuration files.
package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

// KeyManager defines the interface for key management operations
// Implementations must provide secure key storage and retrieval
type KeyManager interface {
	// GetKey retrieves a key by ID. Returns the key material and key version.
	GetKey(ctx context.Context, keyID string) ([]byte, error)

	// EncryptData encrypts plaintext using the specified key
	EncryptData(ctx context.Context, keyID string, plaintext []byte) ([]byte, error)

	// DecryptData decrypts ciphertext using the specified key
	DecryptData(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error)

	// RotateKey creates a new version of a key
	RotateKey(ctx context.Context, keyID string) (newKeyID string, err error)

	// ScheduleKeyDeletion schedules a key for deletion
	ScheduleKeyDeletion(ctx context.Context, keyID string, pendingWindowInDays int) error
}

// KeyVersion represents a specific version of a key
type KeyVersion struct {
	KeyID      string
	Version    int
	CreatedAt  int64
	IsActive   bool
}

// WrappedKey represents a key wrapped by KMS
type WrappedKey struct {
	KeyID         string
	WrappedKey    []byte // The encrypted key material
	KeyVersion    int
	EncryptionAlg string // e.g., "AES256_GCM"
}

// MasterKeyManager manages master keys using KMS
// SECURITY: Master keys are NEVER stored in plaintext or logged
type MasterKeyManager struct {
	provider       KeyManager
	masterKeyID    string // The KMS key ID for the master key
	currentVersion int
	logger         zerolog.Logger
	cache          map[string][]byte // In-memory cache for unwrapped keys (secured at runtime)
}

// NewMasterKeyManager creates a new master key manager
// SECURITY: The masterKeyID must reference a KMS key, not raw key material
func NewMasterKeyManager(provider KeyManager, masterKeyID string, logger zerolog.Logger) (*MasterKeyManager, error) {
	if provider == nil {
		return nil, fmt.Errorf("kms: provider cannot be nil")
	}
	if masterKeyID == "" {
		return nil, fmt.Errorf("kms: masterKeyID cannot be empty")
	}

	return &MasterKeyManager{
		provider:     provider,
		masterKeyID:  masterKeyID,
		currentVersion: 1,
		logger:       logger,
		cache:        make(map[string][]byte),
	}, nil
}

// GetMasterKey retrieves the current master key from KMS
// SECURITY: Key material is cached in memory only and never logged
func (mkm *MasterKeyManager) GetMasterKey(ctx context.Context) ([]byte, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:v%d", mkm.masterKeyID, mkm.currentVersion)
	if key, ok := mkm.cache[cacheKey]; ok {
		return key, nil
	}

	// Fetch from KMS
	key, err := mkm.provider.GetKey(ctx, mkm.masterKeyID)
	if err != nil {
		return nil, fmt.Errorf("kms.GetMasterKey: %w", err)
	}

	// Validate key size
	if len(key) != 32 {
		return nil, fmt.Errorf("kms: invalid master key size %d, expected 32 bytes for AES-256", len(key))
	}

	// Cache in memory (runtime only)
	mkm.cache[cacheKey] = key

	return key, nil
}

// WrapKey wraps a data encryption key (DEK) using the master key from KMS
// SECURITY: The wrapped key can be safely stored in databases or files
func (mkm *MasterKeyManager) WrapKey(ctx context.Context, dek []byte) (*WrappedKey, error) {
	// Encrypt the DEK using the master key from KMS
	wrappedKey, err := mkm.provider.EncryptData(ctx, mkm.masterKeyID, dek)
	if err != nil {
		return nil, fmt.Errorf("kms.WrapKey: %w", err)
	}

	return &WrappedKey{
		KeyID:         mkm.masterKeyID,
		WrappedKey:    wrappedKey,
		KeyVersion:    mkm.currentVersion,
		EncryptionAlg: "AES256_GCM",
	}, nil
}

// UnwrapKey unwraps a data encryption key (DEK) using the master key from KMS
func (mkm *MasterKeyManager) UnwrapKey(ctx context.Context, wrapped *WrappedKey) ([]byte, error) {
	if wrapped == nil {
		return nil, fmt.Errorf("kms: wrapped key cannot be nil")
	}

	// Decrypt the DEK using the master key from KMS
	dek, err := mkm.provider.DecryptData(ctx, wrapped.KeyID, wrapped.WrappedKey)
	if err != nil {
		return nil, fmt.Errorf("kms.UnwrapKey: %w", err)
	}

	return dek, nil
}

// RotateMasterKey rotates the master key in KMS
// SECURITY: This creates a new key version. Old data encrypted with previous
// versions can still be decrypted using key versioning.
func (mkm *MasterKeyManager) RotateMasterKey(ctx context.Context) error {
	newKeyID, err := mkm.provider.RotateKey(ctx, mkm.masterKeyID)
	if err != nil {
		return fmt.Errorf("kms.RotateMasterKey: %w", err)
	}

	mkm.masterKeyID = newKeyID
	mkm.currentVersion++
	mkm.cache = make(map[string][]byte) // Clear cache on rotation

	mkm.logger.Info().
		Str("key_id", newKeyID).
		Int("version", mkm.currentVersion).
		Msg("Master key rotated successfully")

	return nil
}

// GenerateDEK generates a new data encryption key
// SECURITY: DEKs are generated locally, wrapped with KMS, and never stored in plaintext
func GenerateDEK() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("kms.GenerateDEK: %w", err)
	}
	return key, nil
}

// InMemoryKeyManager is an INSECURE implementation for testing only
// WARNING: Do NOT use in production. Keys are stored in plaintext memory.
type InMemoryKeyManager struct {
	keys map[string][]byte
	logger zerolog.Logger
}

// NewInMemoryKeyManager creates an insecure in-memory key manager for testing
func NewInMemoryKeyManager(logger zerolog.Logger) *InMemoryKeyManager {
	return &InMemoryKeyManager{
		keys:   make(map[string][]byte),
		logger: logger,
	}
}

// GenerateKey creates a new key in the in-memory store
func (im *InMemoryKeyManager) GenerateKey(keyID string) ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	im.keys[keyID] = key
	im.logger.Warn().Str("key_id", keyID).Msg("SECURITY: Using insecure in-memory key manager - DO NOT USE IN PRODUCTION")
	return key, nil
}

// SetKey sets a key in the in-memory store
func (im *InMemoryKeyManager) SetKey(keyID string, key []byte) {
	im.keys[keyID] = key
	im.logger.Warn().Str("key_id", keyID).Msg("SECURITY: Using insecure in-memory key manager - DO NOT USE IN PRODUCTION")
}

func (im *InMemoryKeyManager) GetKey(ctx context.Context, keyID string) ([]byte, error) {
	key, ok := im.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("kms: key not found: %s", keyID)
	}
	return key, nil
}

func (im *InMemoryKeyManager) EncryptData(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	key, err := im.GetKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (im *InMemoryKeyManager) DecryptData(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	key, err := im.GetKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("kms: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (im *InMemoryKeyManager) RotateKey(ctx context.Context, keyID string) (string, error) {
	newKeyID := keyID + "-v2"
	newKey, err := im.GenerateKey(newKeyID)
	if err != nil {
		return "", err
	}
	im.keys[newKeyID] = newKey
	return newKeyID, nil
}

func (im *InMemoryKeyManager) ScheduleKeyDeletion(ctx context.Context, keyID string, pendingWindowInDays int) error {
	delete(im.keys, keyID)
	return nil
}

// WrappedKeySerializer handles serialization of wrapped keys
type WrappedKeySerializer struct{}

// Serialize converts a wrapped key to a string for storage
func (ws *WrappedKeySerializer) Serialize(wk *WrappedKey) (string, error) {
	if wk == nil {
		return "", fmt.Errorf("kms: wrapped key is nil")
	}

	// Format: base64(keyID) . base64(wrappedKey) . version . alg
	encodedKeyID := base64.RawURLEncoding.EncodeToString([]byte(wk.KeyID))
	encodedKey := base64.RawURLEncoding.EncodeToString(wk.WrappedKey)

	return fmt.Sprintf("%s.%s.%d.%s", encodedKeyID, encodedKey, wk.KeyVersion, wk.EncryptionAlg), nil
}

// Deserialize parses a wrapped key from storage
func (ws *WrappedKeySerializer) Deserialize(data string) (*WrappedKey, error) {
	parts := strings.Split(data, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("kms: invalid wrapped key format: expected 4 parts, got %d", len(parts))
	}

	encodedKeyID := parts[0]
	encodedKey := parts[1]
	versionStr := parts[2]
	alg := parts[3]

	var version int
	_, err := fmt.Sscanf(versionStr, "%d", &version)
	if err != nil {
		return nil, fmt.Errorf("kms: invalid version format: %w", err)
	}

	decodedKeyID, err := base64.RawURLEncoding.DecodeString(encodedKeyID)
	if err != nil {
		return nil, fmt.Errorf("kms: invalid key ID encoding: %w", err)
	}

	decodedKey, err := base64.RawURLEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("kms: invalid wrapped key encoding: %w", err)
	}

	return &WrappedKey{
		KeyID:         string(decodedKeyID),
		WrappedKey:    decodedKey,
		KeyVersion:    version,
		EncryptionAlg: alg,
	}, nil
}
