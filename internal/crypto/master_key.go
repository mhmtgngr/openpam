package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/scrypt"
)

const (
	// MasterKeySize is the size of the master key in bytes
	MasterKeySize = 32
	// KeyVersionCurrent is the current key version identifier
	KeyVersionCurrent = 1
)

var (
	// ErrMasterKeyNotFound is returned when master key is not found
	ErrMasterKeyNotFound = errors.New("crypto: master key not found")
	// ErrMasterKeyLocked is returned when master key is locked
	ErrMasterKeyLocked = errors.New("crypto: master key is locked")
	// ErrInvalidKeyFormat is returned when key format is invalid
	ErrInvalidKeyFormat = errors.New("crypto: invalid key format")
)

// MasterKeyManager manages secure storage and rotation of master keys
type MasterKeyManager struct {
	mu              sync.RWMutex
	currentKey      []byte
	currentKeyID    string
	keyVersions     map[string]*KeyVersion
	keyPath         string
	locked          bool
	lastAccess      time.Time
	autoLockTimeout time.Duration
	logger          zerolog.Logger
}

// KeyVersion represents a version of the master key
type KeyVersion struct {
	KeyID      string    `json:"key_id"`
	Version    int       `json:"version"`
	Algorithm  string    `json:"algorithm"`
	CreatedAt  time.Time `json:"created_at"`
	RotatedAt  *time.Time `json:"rotated_at,omitempty"`
	IsActive   bool      `json:"is_active"`
	Salt       []byte    `json:"salt,omitempty"` // For encrypted storage
	EncryptedKey []byte   `json:"encrypted_key,omitempty"`
}

// MasterKeyConfig holds configuration for master key management
type MasterKeyConfig struct {
	KeyPath         string
	AutoLockTimeout time.Duration
	EnableFileBackup bool
	KMSEndpoint     string // For future KMS integration
}

// NewMasterKeyManager creates a new master key manager
func NewMasterKeyManager(config MasterKeyConfig, logger zerolog.Logger) (*MasterKeyManager, error) {
	m := &MasterKeyManager{
		keyVersions:     make(map[string]*KeyVersion),
		keyPath:         config.KeyPath,
		autoLockTimeout: config.AutoLockTimeout,
		logger:          logger,
		locked:          false,
	}

	if m.autoLockTimeout == 0 {
		m.autoLockTimeout = 15 * time.Minute // Default timeout
	}

	// Try to load existing keys
	if err := m.loadKeysFromFile(); err != nil {
		if !errors.Is(err, ErrMasterKeyNotFound) {
			logger.Warn().Err(err).Msg("Failed to load existing keys, will create new on first use")
		}
	}

	return m, nil
}

// InitializeFromEnvironment initializes master key from environment variable
// For production, this should be replaced with KMS integration
func (m *MasterKeyManager) InitializeFromEnvironment(envKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keyStr := os.Getenv(envKey)
	if keyStr == "" {
		return ErrMasterKeyNotFound
	}

	// Derive key from environment string using SHA-256 (basic implementation)
	hash := sha256.Sum256([]byte(keyStr))
	m.currentKey = hash[:]

	keyID := fmt.Sprintf("mk-%d", time.Now().Unix())
	m.currentKeyID = keyID

	m.keyVersions[keyID] = &KeyVersion{
		KeyID:     keyID,
		Version:   KeyVersionCurrent,
		Algorithm: "AES-256-GCM",
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	m.logger.Info().Str("key_id", keyID).Msg("Master key initialized from environment")

	return nil
}

// InitializeFromPassphrase initializes master key from a passphrase
func (m *MasterKeyManager) InitializeFromPassphrase(passphrase string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a random salt
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("crypto.GenerateSalt: %w", err)
	}

	// Derive key using scrypt
	key, err := scrypt.Key([]byte(passphrase), salt, 32768, 8, 1, MasterKeySize)
	if err != nil {
		return fmt.Errorf("crypto.Scrypt: %w", err)
	}

	m.currentKey = key

	keyID := fmt.Sprintf("mk-%d", time.Now().Unix())
	m.currentKeyID = keyID

	m.keyVersions[keyID] = &KeyVersion{
		KeyID:     keyID,
		Version:   KeyVersionCurrent,
		Algorithm: "AES-256-GCM",
		CreatedAt: time.Now(),
		IsActive:  true,
		Salt:      salt,
	}

	m.logger.Info().Str("key_id", keyID).Msg("Master key derived from passphrase")

	return m.saveKeysToFile()
}

// GetCurrentKey returns the current master key
// Returns error if key is locked or not initialized
func (m *MasterKeyManager) GetCurrentKey(ctx context.Context) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.locked {
		return nil, ErrMasterKeyLocked
	}

	if m.currentKey == nil {
		return nil, ErrMasterKeyNotFound
	}

	// Update last access time
	m.lastAccess = time.Now()

	// Check for auto-lock
	if time.Since(m.lastAccess) > m.autoLockTimeout {
		m.locked = true
		return nil, ErrMasterKeyLocked
	}

	return m.currentKey, nil
}

// GetCurrentKeyID returns the current key ID
func (m *MasterKeyManager) GetCurrentKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKeyID
}

// RotateKey rotates the master key to a new value
func (m *MasterKeyManager) RotateKey(ctx context.Context, newKey []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(newKey) != MasterKeySize {
		return fmt.Errorf("crypto: invalid key size %d, expected %d", len(newKey), MasterKeySize)
	}

	oldKeyID := m.currentKeyID
	oldKey := m.currentKey

	// Create new key version
	newKeyID := fmt.Sprintf("mk-%d", time.Now().Unix())
	now := time.Now()

	// Deactivate old key
	if oldVersion, ok := m.keyVersions[oldKeyID]; ok {
		oldVersion.IsActive = false
		oldVersion.RotatedAt = &now
	}

	// Set new key as current
	m.currentKey = newKey
	m.currentKeyID = newKeyID

	m.keyVersions[newKeyID] = &KeyVersion{
		KeyID:     newKeyID,
		Version:   KeyVersionCurrent + len(m.keyVersions),
		Algorithm: "AES-256-GCM",
		CreatedAt: now,
		IsActive:  true,
	}

	m.logger.Info().
		Str("old_key_id", oldKeyID).
		Str("new_key_id", newKeyID).
		Msg("Master key rotated")

	if err := m.saveKeysToFile(); err != nil {
		// Rollback on save failure
		m.currentKey = oldKey
		m.currentKeyID = oldKeyID
		return fmt.Errorf("crypto.SaveKeys: %w", err)
	}

	return nil
}

// GenerateNewKey generates a new random master key
func (m *MasterKeyManager) GenerateNewKey() ([]byte, error) {
	key := make([]byte, MasterKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("crypto.GenerateKey: %w", err)
	}
	return key, nil
}

// Lock locks the master key manager
func (m *MasterKeyManager) Lock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.locked = true
	m.logger.Info().Msg("Master key manager locked")
}

// Unlock unlocks the master key manager
func (m *MasterKeyManager) Unlock(passphrase string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.locked && m.currentKey != nil {
		return nil // Already unlocked
	}

	// For passphrase-based keys, re-derive
	// In production, this would query KMS or secure enclave
	if len(m.keyVersions) > 0 {
		var keyVersion *KeyVersion
		for _, kv := range m.keyVersions {
			if kv.IsActive {
				keyVersion = kv
				break
			}
		}

		if keyVersion != nil && len(keyVersion.Salt) > 0 {
			key, err := scrypt.Key([]byte(passphrase), keyVersion.Salt, 32768, 8, 1, MasterKeySize)
			if err != nil {
				return fmt.Errorf("crypto.Scrypt: %w", err)
			}
			m.currentKey = key
			m.currentKeyID = keyVersion.KeyID
			m.locked = false
			m.lastAccess = time.Now()
			m.logger.Info().Msg("Master key manager unlocked")
			return nil
		}
	}

	return ErrMasterKeyNotFound
}

// GetKeyVersions returns all key versions
func (m *MasterKeyManager) GetKeyVersions() []*KeyVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions := make([]*KeyVersion, 0, len(m.keyVersions))
	for _, v := range m.keyVersions {
		versions = append(versions, v)
	}
	return versions
}

// saveKeysToFile saves encrypted key metadata to file
func (m *MasterKeyManager) saveKeysToFile() error {
	if m.keyPath == "" {
		return nil // No file storage configured
	}

	// Ensure directory exists
	dir := filepath.Dir(m.keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("crypto.MkdirAll: %w", err)
	}

	data, err := json.MarshalIndent(m.keyVersions, "", "  ")
	if err != nil {
		return fmt.Errorf("crypto.Marshal: %w", err)
	}

	// Write with restricted permissions
	return os.WriteFile(m.keyPath, data, 0600)
}

// loadKeysFromFile loads key metadata from file
func (m *MasterKeyManager) loadKeysFromFile() error {
	if m.keyPath == "" {
		return ErrMasterKeyNotFound
	}

	data, err := os.ReadFile(m.keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrMasterKeyNotFound
		}
		return fmt.Errorf("crypto.ReadFile: %w", err)
	}

	var versions map[string]*KeyVersion
	if err := json.Unmarshal(data, &versions); err != nil {
		return ErrInvalidKeyFormat
	}

	m.keyVersions = versions

	// Find active key
	for _, v := range versions {
		if v.IsActive {
			m.currentKeyID = v.KeyID
			break
		}
	}

	m.logger.Info().
		Str("current_key_id", m.currentKeyID).
		Int("total_versions", len(versions)).
		Msg("Loaded key metadata from file")

	return nil
}

// EncryptForStorage encrypts a key for secure storage
func EncryptForStorage(key, encryptionKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
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

	return gcm.Seal(nonce, nonce, key, nil), nil
}

// DecryptFromStorage decrypts a key from secure storage
func DecryptFromStorage(encryptedKey, encryptionKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedKey) < nonceSize {
		return nil, errors.New("crypto: ciphertext too short")
	}

	nonce, ciphertext := encryptedKey[:nonceSize], encryptedKey[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GenerateMasterKeyFromString creates a master key from a string using SHA-256
// This is a simplified method for development/testing
func GenerateMasterKeyFromString(keyStr string) []byte {
	hash := sha256.Sum256([]byte(keyStr))
	return hash[:]
}

// EncodeMasterKey encodes a master key to base64 for storage/transmission
func EncodeMasterKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodeMasterKey decodes a base64-encoded master key
func DecodeMasterKey(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// KMSEncryptor is an interface for KMS-based encryption
// This allows for future integration with AWS KMS, GCP KMS, Azure Key Vault, etc.
type KMSEncryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
	GenerateDataKey(ctx context.Context) ([]byte, error)
}

// MockKMS is a mock KMS implementation for development/testing
type MockKMS struct {
	mu   sync.Mutex
	keys map[string][]byte
}

// NewMockKMS creates a new mock KMS
func NewMockKMS() *MockKMS {
	return &MockKMS{
		keys: make(map[string][]byte),
	}
}

// Encrypt encrypts data using mock KMS
func (k *MockKMS) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Generate a random key ID
	keyID := fmt.Sprintf("kms-key-%d", time.Now().UnixNano())

	// Generate a data key
	dataKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, err
	}

	// Store the data key
	k.keys[keyID] = dataKey

	// Encrypt the plaintext with the data key
	block, err := aes.NewCipher(dataKey)
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

	// Return: keyID + encrypted data
	result := append([]byte(keyID), ciphertext...)
	return result, nil
}

// Decrypt decrypts data using mock KMS
func (k *MockKMS) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Extract key ID (first 32 bytes or until null)
	keyIDEnd := 0
	for i, b := range ciphertext {
		if b == 0 {
			keyIDEnd = i
			break
		}
		if i >= 100 { // Safety limit
			keyIDEnd = 32
			break
		}
	}

	keyID := string(ciphertext[:keyIDEnd])
	encryptedData := ciphertext[keyIDEnd:]

	// Get the data key
	dataKey, ok := k.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("kms: key not found")
	}

	// Decrypt the data
	block, err := aes.NewCipher(dataKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, errors.New("kms: ciphertext too short")
	}

	nonce, ct := encryptedData[:nonceSize], encryptedData[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// GenerateDataKey generates a new data key using mock KMS
func (k *MockKMS) GenerateDataKey(ctx context.Context) ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}
