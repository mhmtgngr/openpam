package session

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
)

// RecordingEncryption handles encryption of session recordings before storage
type RecordingEncryption struct {
	masterKey     []byte
	minioClient   *minio.Client
	bucketName    string
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
}

// NewRecordingEncryption creates a new recording encryption handler
func NewRecordingEncryption(config RecordingConfig, logger zerolog.Logger) (*RecordingEncryption, error) {
	if len(config.MasterKey) != 32 {
		return nil, fmt.Errorf("session: master key must be 32 bytes")
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

	return &RecordingEncryption{
		masterKey:   config.MasterKey,
		minioClient: minioClient,
		bucketName:  config.MinioBucket,
		logger:      logger,
	}, nil
}

// EncryptAndStore encrypts recording data and stores it in MinIO
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

	// Store encrypted DEK as metadata
	objectID := uuid.New().String()
	metadata := map[string]string{
		"session-id":    sessionID.String(),
		"encrypted-dek": string(encryptedDEK), // In production, store this separately or use KMS
		"content-type":  "application/octet-stream",
		"encryption":    "aes-256-gcm",
		"created-at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Upload encrypted data to MinIO
	objectName := fmt.Sprintf("recordings/%s/%s.enc", sessionID.String(), objectID)
	_, err = re.minioClient.PutObject(ctx, re.bucketName, objectName, bytes.NewReader(encryptedData), int64(len(encryptedData)), minio.PutObjectOptions{
		ContentType:  "application/octet-stream",
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("session.UploadRecording: %w", err)
	}

	re.logger.Info().
		Str("session_id", sessionID.String()).
		Str("object_name", objectName).
		Int("size_bytes", len(encryptedData)).
		Msg("Encrypted and stored session recording")

	return objectName, nil
}

// RetrieveAndDecrypt retrieves and decrypts a recording
func (re *RecordingEncryption) RetrieveAndDecrypt(ctx context.Context, sessionID uuid.UUID, objectName string) ([]byte, error) {
	// Get the object to retrieve metadata
	obj, err := re.minioClient.GetObject(ctx, re.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("session.GetObject: %w", err)
	}
	defer obj.Close()

	// Read encrypted DEK from metadata
	stat, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("session.StatObject: %w", err)
	}

	encryptedDEK := []byte(stat.Metadata.Get("encrypted-dek"))
	if len(encryptedDEK) == 0 {
		return nil, fmt.Errorf("session: missing encrypted DEK")
	}

	// Decrypt the DEK
	dek, err := re.decryptDEK(encryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("session.DecryptDEK: %w", err)
	}

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
	return re.encryptWithKey(dek, re.masterKey)
}

// decryptDEK decrypts a DEK with the master key
func (re *RecordingEncryption) decryptDEK(encryptedDEK []byte) ([]byte, error) {
	return re.decryptWithKey(encryptedDEK, re.masterKey)
}

// DeleteRecording deletes a recording from storage
func (re *RecordingEncryption) DeleteRecording(ctx context.Context, objectName string) error {
	err := re.minioClient.RemoveObject(ctx, re.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("session.RemoveObject: %w", err)
	}

	re.logger.Info().
		Str("object_name", objectName).
		Msg("Deleted session recording")

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
