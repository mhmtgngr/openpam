package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// EnvelopeEncryption implements envelope encryption for the vault
type EnvelopeEncryption struct {
	masterKey     []byte
	db            *sqlx.DB
	cache         *cache.Cache
	logger        zerolog.Logger
	currentKeyID  string
}

// EncryptedDEK represents an encrypted Data Encryption Key
type EncryptedDEK struct {
	KeyID       string    `db:"key_id" json:"key_id"`
	EncryptedKey []byte   `db:"encrypted_key" json:"encrypted_key"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// NewEnvelopeEncryption creates a new envelope encryption handler
func NewEnvelopeEncryption(masterKey []byte, keyID string, db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) (*EnvelopeEncryption, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("vault: master key must be 32 bytes for AES-256")
	}

	return &EnvelopeEncryption{
		masterKey:    masterKey,
		db:          db,
		cache:       c,
		logger:      logger,
		currentKeyID: keyID,
	}, nil
}

// GenerateDEK generates a new Data Encryption Key
func (ee *EnvelopeEncryption) GenerateDEK() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("vault.GenerateDEK: %w", err)
	}
	return key, nil
}

// EncryptDEK encrypts a DEK with the master key using AES-GCM
func (ee *EnvelopeEncryption) EncryptDEK(dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(ee.masterKey)
	if err != nil {
		return nil, fmt.Errorf("vault.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("vault.GenerateNonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, dek, nil)
	return ciphertext, nil
}

// DecryptDEK decrypts a DEK with the master key
func (ee *EnvelopeEncryption) DecryptDEK(encryptedDEK []byte) ([]byte, error) {
	block, err := aes.NewCipher(ee.masterKey)
	if err != nil {
		return nil, fmt.Errorf("vault.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedDEK) < nonceSize {
		return nil, fmt.Errorf("vault: ciphertext too short")
	}

	nonce, ciphertext := encryptedDEK[:nonceSize], encryptedDEK[nonceSize:]
	dek, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("vault.GCMOpen: %w", err)
	}

	return dek, nil
}

// StoreDEK stores an encrypted DEK in the database
func (ee *EnvelopeEncryption) StoreDEK(ctx context.Context, credentialID string, encryptedDEK []byte) error {
	keyID := ee.currentKeyID

	query := `
		INSERT INTO encrypted_deks (id, credential_id, key_id, encrypted_key, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (credential_id) DO UPDATE SET
			key_id = $3,
			encrypted_key = $4
	`

	_, err := ee.db.ExecContext(ctx, query, uuid.New().String(), credentialID, keyID, encryptedDEK)
	if err != nil {
		return fmt.Errorf("vault.StoreDEK: %w", err)
	}

	return nil
}

// RetrieveDEK retrieves and decrypts a DEK
func (ee *EnvelopeEncryption) RetrieveDEK(ctx context.Context, credentialID string) ([]byte, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("dek:%s", credentialID)
	if encryptedDEK, err := ee.cache.Get(ctx, cacheKey); err == nil {
		return ee.DecryptDEK([]byte(encryptedDEK))
	}

	// Fetch from database
	var encryptedDEK []byte
	var keyID string
	query := `
		SELECT encrypted_key, key_id FROM encrypted_deks
		WHERE credential_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := ee.db.QueryRowContext(ctx, query, credentialID).Scan(&encryptedDEK, &keyID)
	if err != nil {
		return nil, fmt.Errorf("vault.RetrieveDEK: %w", err)
	}

	// Cache for 5 minutes
	_ = ee.cache.Set(ctx, cacheKey, string(encryptedDEK), 5*time.Minute)

	// If key is rotated, handle old key decryption
	if keyID != ee.currentKeyID {
		return ee.decryptDEKWithKey(encryptedDEK, keyID)
	}

	return ee.DecryptDEK(encryptedDEK)
}

// decryptDEKWithKey decrypts a DEK with a specific key (for rotation)
func (ee *EnvelopeEncryption) decryptDEKWithKey(encryptedDEK []byte, keyID string) ([]byte, error) {
	// For now, use current master key
	// In production, you'd maintain key versions and old master keys
	return ee.DecryptDEK(encryptedDEK)
}

// RotateMasterKey rotates the master key and re-encrypts all DEKs
func (ee *EnvelopeEncryption) RotateMasterKey(ctx context.Context, newMasterKey []byte, newKeyID string) error {
	oldMasterKey := ee.masterKey

	// Get all encrypted DEKs
	query := `SELECT credential_id, encrypted_key FROM encrypted_deks WHERE key_id = $1`
	rows, err := ee.db.QueryContext(ctx, query, ee.currentKeyID)
	if err != nil {
		return fmt.Errorf("vault.RotateMasterKey.Query: %w", err)
	}
	defer rows.Close()

	var updates []struct {
		credentialID string
		oldEncrypted []byte
		newEncrypted []byte
	}

	for rows.Next() {
		var credentialID string
		var oldEncrypted []byte
		if err := rows.Scan(&credentialID, &oldEncrypted); err != nil {
			return fmt.Errorf("vault.RotateMasterKey.Scan: %w", err)
		}

		// Decrypt with old key
		dek, err := ee.decryptWithKey(oldEncrypted, oldMasterKey)
		if err != nil {
			return fmt.Errorf("vault.RotateMasterKey.Decrypt: %w", err)
		}

		// Encrypt with new key
		newEncrypted, err := ee.encryptWithKey(dek, newMasterKey)
		if err != nil {
			return fmt.Errorf("vault.RotateMasterKey.Encrypt: %w", err)
		}

		updates = append(updates, struct {
			credentialID string
			oldEncrypted []byte
			newEncrypted []byte
		}{credentialID, oldEncrypted, newEncrypted})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("vault.RotateMasterKey.Rows: %w", err)
	}

	// Update all DEKs in transaction
	tx, err := ee.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vault.RotateMasterKey.Begin: %w", err)
	}
	defer tx.Rollback()

	updateQuery := `
		UPDATE encrypted_deks
		SET key_id = $2, encrypted_key = $3
		WHERE credential_id = $1
	`

	for _, update := range updates {
		if _, err := tx.Exec(updateQuery, update.credentialID, newKeyID, update.newEncrypted); err != nil {
			return fmt.Errorf("vault.RotateMasterKey.Update: %w", err)
		}

		// Invalidate cache
		cacheKey := fmt.Sprintf("dek:%s", update.credentialID)
		_ = ee.cache.Delete(ctx, cacheKey)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vault.RotateMasterKey.Commit: %w", err)
	}

	// Update current master key
	ee.masterKey = newMasterKey
	ee.currentKeyID = newKeyID

	ee.logger.Info().Str("key_id", newKeyID).Msg("Master key rotated")

	return nil
}

func (ee *EnvelopeEncryption) decryptWithKey(encryptedDEK []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := encryptedDEK[:nonceSize], encryptedDEK[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (ee *EnvelopeEncryption) encryptWithKey(dek []byte, key []byte) ([]byte, error) {
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

	return gcm.Seal(nonce, nonce, dek, nil), nil
}

// KeyMetadata stores metadata about encryption keys
type KeyMetadata struct {
	KeyID      string    `db:"key_id" json:"key_id"`
	KeyVersion int       `db:"key_version" json:"key_version"`
	Algorithm  string    `db:"algorithm" json:"algorithm"`
	KeySize    int       `db:"key_size" json:"key_size"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	RotatedAt  *time.Time `db:"rotated_at" json:"rotated_at,omitempty"`
}

// StoreKeyMetadata stores key metadata
func (ee *EnvelopeEncryption) StoreKeyMetadata(ctx context.Context, metadata KeyMetadata) error {
	query := `
		INSERT INTO key_metadata (key_id, key_version, algorithm, key_size, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err := ee.db.ExecContext(ctx, query, metadata.KeyID, metadata.KeyVersion, metadata.Algorithm, metadata.KeySize, metadata.IsActive)
	if err != nil {
		return fmt.Errorf("vault.StoreKeyMetadata: %w", err)
	}
	return nil
}

// GetCurrentKeyMetadata retrieves current key metadata
func (ee *EnvelopeEncryption) GetCurrentKeyMetadata(ctx context.Context) (*KeyMetadata, error) {
	var metadata KeyMetadata
	query := `
		SELECT key_id, key_version, algorithm, key_size, is_active, created_at, rotated_at
		FROM key_metadata
		WHERE is_active = true
		ORDER BY key_version DESC
		LIMIT 1
	`
	err := ee.db.GetContext(ctx, &metadata, query)
	if err != nil {
		return nil, fmt.Errorf("vault.GetCurrentKeyMetadata: %w", err)
	}
	return &metadata, nil
}
