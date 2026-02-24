package session

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
)

// RecordingEncryption handles encryption of session recordings before storage
// SECURITY FIX: Encrypted DEKs are now stored in a secure database table, not in MinIO metadata
type RecordingEncryption struct {
	masterKey     []byte
	minioClient   *minio.Client
	bucketName    string
	db            *sqlx.DB
	logger        zerolog.Logger
}

// RecordingConfig holds configuration for recording encryption
type RecordingConfig struct {
	MinioEndpoint        string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioBucket          string
	MinioUseSSL          bool
	MasterKey            []byte
	DB                   *sqlx.DB
}

// NewRecordingEncryption creates a new recording encryption handler
func NewRecordingEncryption(config RecordingConfig, logger zerolog.Logger) (*RecordingEncryption, error) {
	if len(config.MasterKey) != 32 {
		return nil, fmt.Errorf("session: master key must be 32 bytes")
	}

	if config.DB == nil {
		return nil, fmt.Errorf("session: database connection required for secure DEK storage")
	}

	// Initialize MinIO client
	minioClient, err := minio.New(config.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, ""),
		Secure: config.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("session.MinioInit: %w", err)
	}

	// Ensure bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, config.MinioBucket)
	if err != nil {
		return nil, fmt.Errorf("session.BucketCheck: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, config.MinioBucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("session.CreateBucket: %w", err)
		}
		logger.Info().Str("bucket", config.MinioBucket).Msg("Created MinIO bucket for recordings")
	}

	// Ensure recording_keys table exists
	if err := createRecordingKeysTable(ctx, config.DB); err != nil {
		return nil, fmt.Errorf("session.CreateKeysTable: %w", err)
	}

	return &RecordingEncryption{
		masterKey:   config.MasterKey,
		minioClient: minioClient,
		bucketName:  config.MinioBucket,
		db:          config.DB,
		logger:      logger,
	}, nil
}

// createRecordingKeysTable creates the secure table for storing encrypted DEKs
func createRecordingKeysTable(ctx context.Context, db *sqlx.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS recording_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			session_id UUID NOT NULL,
			object_name TEXT NOT NULL UNIQUE,
			encrypted_dek BYTEA NOT NULL,
			key_version INTEGER NOT NULL DEFAULT 1,
			algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_recording_keys_session ON recording_keys(session_id);
		CREATE INDEX IF NOT EXISTS idx_recording_keys_object ON recording_keys(object_name);
		CREATE INDEX IF NOT EXISTS idx_recording_keys_deleted ON recording_keys(deleted_at);
	`
	_, err := db.ExecContext(ctx, query)
	return err
}

// EncryptAndStore encrypts recording data and stores it in MinIO
// SECURITY FIX: Encrypted DEK is stored in the database, not in MinIO metadata
func (re *RecordingEncryption) EncryptAndStore(ctx context.Context, sessionID uuid.UUID, recordingData []byte) (string, error) {
	// Generate a unique data encryption key (DEK) for this recording
	dek, err := crypto.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("session.GenerateDEK: %w", err)
	}

	// Encrypt the recording data with the DEK using AES-256-GCM
	encryptedData, err := re.encryptWithKey(recordingData, dek)
	if err != nil {
		return "", fmt.Errorf("session.EncryptRecording: %w", err)
	}

	// Encrypt the DEK with the master key
	encryptedDEK, err := re.encryptDEK(dek)
	if err != nil {
		return "", fmt.Errorf("session.EncryptDEK: %w", err)
	}

	// Generate object name
	objectID := uuid.New()
	objectName := fmt.Sprintf("recordings/%s/%s.enc", sessionID.String(), objectID.String())

	// SECURITY FIX: Store encrypted DEK in database, NOT in MinIO metadata
	// Object metadata should only contain non-sensitive information
	metadata := map[string]string{
		"session-id":   sessionID.String(),
		"content-type": "application/octet-stream",
		"encryption":   "aes-256-gcm",
		"created-at":   time.Now().UTC().Format(time.RFC3339),
	}

	// Start a transaction for atomic storage
	tx, err := re.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("session.BeginTx: %w", err)
	}
	defer tx.Rollback()

	// Store encrypted DEK in database
	keyID := uuid.New()
	insertKeyQuery := `
		INSERT INTO recording_keys (id, session_id, object_name, encrypted_dek)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, insertKeyQuery, keyID, sessionID, objectName, encryptedDEK)
	if err != nil {
		return "", fmt.Errorf("session.StoreDEK: %w", err)
	}

	// Upload encrypted data to MinIO (without DEK in metadata)
	_, err = re.minioClient.PutObject(ctx, re.bucketName, objectName, bytes.NewReader(encryptedData), int64(len(encryptedData)), minio.PutObjectOptions{
		ContentType:  "application/octet-stream",
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("session.UploadRecording: %w", err)
	}

	if err := tx.Commit(); err != nil {
		// Rollback: delete the uploaded object since we couldn't store the key
		_ = re.minioClient.RemoveObject(ctx, re.bucketName, objectName, minio.RemoveObjectOptions{})
		return "", fmt.Errorf("session.CommitTx: %w", err)
	}

	re.logger.Info().
		Str("session_id", sessionID.String()).
		Str("object_name", objectName).
		Str("key_id", keyID.String()).
		Int("size_bytes", len(encryptedData)).
		Msg("Encrypted and stored session recording with secure DEK storage")

	return objectName, nil
}

// RetrieveAndDecrypt retrieves and decrypts a recording
// SECURITY FIX: Retrieves encrypted DEK from database instead of MinIO metadata
func (re *RecordingEncryption) RetrieveAndDecrypt(ctx context.Context, sessionID uuid.UUID, objectName string) ([]byte, error) {
	// SECURITY FIX: Get encrypted DEK from database, not from MinIO metadata
	var encryptedDEK []byte
	getKeyQuery := `
		SELECT encrypted_dek FROM recording_keys
		WHERE object_name = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`
	err := re.db.GetContext(ctx, &encryptedDEK, getKeyQuery, objectName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session: DEK not found in database - recording may be corrupted")
		}
		return nil, fmt.Errorf("session.GetDEK: %w", err)
	}

	// Decrypt the DEK
	dek, err := re.decryptDEK(encryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("session.DecryptDEK: %w", err)
	}

	// Get the object from MinIO
	obj, err := re.minioClient.GetObject(ctx, re.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("session.GetObject: %w", err)
	}
	defer obj.Close()

	// Read encrypted data
	encryptedData, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("session.ReadObject: %w", err)
	}

	// Decrypt the data
	plaintext, err := re.decryptWithKey(encryptedData, dek)
	if err != nil {
		return nil, fmt.Errorf("session.DecryptData: %w", err)
	}

	re.logger.Info().
		Str("session_id", sessionID.String()).
		Str("object_name", objectName).
		Int("size_bytes", len(plaintext)).
		Msg("Retrieved and decrypted session recording")

	return plaintext, nil
}

// DeleteRecording deletes a recording from storage and securely removes the DEK
func (re *RecordingEncryption) DeleteRecording(ctx context.Context, objectName string) error {
	tx, err := re.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("session.BeginTx: %w", err)
	}
	defer tx.Rollback()

	// Soft delete the key (mark as deleted)
	deleteKeyQuery := `
		UPDATE recording_keys SET deleted_at = NOW()
		WHERE object_name = $1
	`
	result, err := tx.ExecContext(ctx, deleteKeyQuery, objectName)
	if err != nil {
		return fmt.Errorf("session.DeleteDEK: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		re.logger.Warn().Str("object_name", objectName).Msg("No DEK found to delete")
	}

	// Delete the object from MinIO
	err = re.minioClient.RemoveObject(ctx, re.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("session.RemoveObject: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("session.CommitTx: %w", err)
	}

	re.logger.Info().
		Str("object_name", objectName).
		Msg("Deleted session recording and secure DEK")

	return nil
}

// ListRecordings lists all recordings for a session
func (re *RecordingEncryption) ListRecordings(ctx context.Context, sessionID uuid.UUID) ([]string, error) {
	prefix := fmt.Sprintf("recordings/%s/", sessionID.String())

	objectsCh := re.minioClient.ListObjects(ctx, re.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var recordings []string
	for object := range objectsCh {
		if object.Err != nil {
			return nil, fmt.Errorf("session.ListObjects: %w", object.Err)
		}
		recordings = append(recordings, object.Key)
	}

	return recordings, nil
}

// RotateDEK rotates a DEK for a recording (for master key rotation)
func (re *RecordingEncryption) RotateDEK(ctx context.Context, objectName string, newMasterKey []byte) error {
	// Get the current encrypted DEK
	type keyResult struct {
		ID           uuid.UUID `db:"id"`
		EncryptedDEK []byte    `db:"encrypted_dek"`
	}
	var result keyResult
	getKeyQuery := `
		SELECT id, encrypted_dek FROM recording_keys
		WHERE object_name = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`
	err := re.db.GetContext(ctx, &result, getKeyQuery, objectName)
	if err != nil {
		return fmt.Errorf("session.GetDEK: %w", err)
	}
	keyID := result.ID
	encryptedDEK := result.EncryptedDEK

	// Decrypt DEK with old master key
	oldEncryptor, err := crypto.NewEncryptor(re.masterKey)
	if err != nil {
		return err
	}
	dek, err := oldEncryptor.Decrypt(encryptedDEK)
	if err != nil {
		return fmt.Errorf("session.DecryptDEK: %w", err)
	}

	// Re-encrypt DEK with new master key
	newEncryptor, err := crypto.NewEncryptor(newMasterKey)
	if err != nil {
		return err
	}
	newEncryptedDEK, err := newEncryptor.Encrypt(dek)
	if err != nil {
		return fmt.Errorf("session.EncryptDEK: %w", err)
	}

	// Update the database
	updateKeyQuery := `
		UPDATE recording_keys SET encrypted_dek = $2, key_version = key_version + 1
		WHERE id = $1
	`
	_, err = re.db.ExecContext(ctx, updateKeyQuery, keyID, newEncryptedDEK)
	if err != nil {
		return fmt.Errorf("session.UpdateDEK: %w", err)
	}

	re.logger.Info().
		Str("object_name", objectName).
		Msg("Rotated DEK for recording")

	return nil
}

// encryptWithKey encrypts data using AES-256-GCM with the given key
func (re *RecordingEncryption) encryptWithKey(plaintext, key []byte) ([]byte, error) {
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

	// Seal appends the nonce to the ciphertext
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decryptWithKey decrypts data using AES-256-GCM with the given key
func (re *RecordingEncryption) decryptWithKey(ciphertext, key []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("session: ciphertext too short")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// encryptDEK encrypts a DEK with the master key
func (re *RecordingEncryption) encryptDEK(dek []byte) ([]byte, error) {
	encryptor, err := crypto.NewEncryptor(re.masterKey)
	if err != nil {
		return nil, err
	}
	return encryptor.Encrypt(dek)
}

// decryptDEK decrypts a DEK with the master key
func (re *RecordingEncryption) decryptDEK(encryptedDEK []byte) ([]byte, error) {
	encryptor, err := crypto.NewEncryptor(re.masterKey)
	if err != nil {
		return nil, err
	}
	return encryptor.Decrypt(encryptedDEK)
}

// ExportDEK exports an encrypted DEK for backup purposes (returns base64 encoded)
func (re *RecordingEncryption) ExportDEK(ctx context.Context, objectName string) (string, error) {
	var encryptedDEK []byte
	getKeyQuery := `
		SELECT encrypted_dek FROM recording_keys
		WHERE object_name = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`
	err := re.db.GetContext(ctx, &encryptedDEK, getKeyQuery, objectName)
	if err != nil {
		return "", fmt.Errorf("session.GetDEK: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encryptedDEK), nil
}

// ImportDEK imports an encrypted DEK for restore purposes
func (re *RecordingEncryption) ImportDEK(ctx context.Context, sessionID uuid.UUID, objectName string, encryptedDEKB64 string) error {
	encryptedDEK, err := base64.StdEncoding.DecodeString(encryptedDEKB64)
	if err != nil {
		return fmt.Errorf("session.DecodeDEK: %w", err)
	}

	keyID := uuid.New()
	insertKeyQuery := `
		INSERT INTO recording_keys (id, session_id, object_name, encrypted_dek)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (object_name) DO UPDATE SET
			encrypted_dek = EXCLUDED.encrypted_dek,
			updated_at = NOW()
	`
	_, err = re.db.ExecContext(ctx, insertKeyQuery, keyID, sessionID, objectName, encryptedDEK)
	if err != nil {
		return fmt.Errorf("session.ImportDEK: %w", err)
	}

	return nil
}
