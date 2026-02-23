package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// SecretType represents the type of secret
type SecretType string

const (
	SecretTypePassword    SecretType = "password"
	SecretTypeSSHKey      SecretType = "ssh_key"
	SecretTypeAPIToken    SecretType = "api_token"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeDatabase    SecretType = "database"
	SecretTypeAWSKey      SecretType = "aws_key"
	SecretTypeAzureKey    SecretType = "azure_key"
)

// RotationPolicy represents credential rotation policy
type RotationPolicy string

const (
	RotationManual     RotationPolicy = "manual"
	RotationDaily      RotationPolicy = "daily"
	RotationWeekly     RotationPolicy = "weekly"
	RotationMonthly    RotationPolicy = "monthly"
	RotationOnCheckin  RotationPolicy = "on_checkin"
)

// Secret represents a stored credential
type Secret struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	Name            string          `db:"name" json:"name"`
	Description     string          `db:"description" json:"description"`
	Type            SecretType      `db:"type" json:"type"`

	// Target info
	TargetID        *uuid.UUID      `db:"target_id" json:"target_id,omitempty"`
	Host            string          `db:"host" json:"host"`
	Port            int             `db:"port" json:"port"`
	Username        string          `db:"username" json:"username"`

	// Encrypted data (envelope encrypted)
	EncryptedSecret  []byte         `db:"encrypted_secret" json:"-"`
	EncryptedSecretKey string        `db:"encrypted_secret_key" json:"-"`

	// SSH Key specific
	SSHKeyPassphrase []byte         `db:"ssh_key_passphrase" json:"-"`

	// Rotation
	RotationPolicy  RotationPolicy  `db:"rotation_policy" json:"rotation_policy"`
	LastRotatedAt   *time.Time      `db:"last_rotated_at" json:"last_rotated_at"`
	NextRotationAt  *time.Time      `db:"next_rotation_at" json:"next_rotation_at"`

	// Organization
	FolderID        *uuid.UUID      `db:"folder_id" json:"folder_id,omitempty"`
	Tags            []string        `db:"tags" json:"tags"`
	TenantID        uuid.UUID       `db:"tenant_id" json:"tenant_id"`

	// Access control
	CreatedBy       uuid.UUID       `db:"created_by" json:"created_by"`
	OwnerID        *uuid.UUID      `db:"owner_id" json:"owner_id,omitempty"`

	// Status
	Status          string          `db:"status" json:"status"` // active, inactive, compromised, expired

	// Metadata
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
}

// SecretData represents decrypted secret data
type SecretData struct {
	Type        SecretType           `json:"type"`
	Username    string               `json:"username"`
	Password    string               `json:"password,omitempty"`
	PrivateKey  string               `json:"private_key,omitempty"`
	PublicKey   string               `json:"public_key,omitempty"`
	Passphrase  string               `json:"passphrase,omitempty"`
	Token       string               `json:"token,omitempty"`
	AccessToken string               `json:"access_token,omitempty"`
	SecretKey   string               `json:"secret_key,omitempty"`
	Certificate string               `json:"certificate,omitempty"`
	Extra       map[string]string    `json:"extra,omitempty"`
}

// SecretRepository handles secret data operations
type SecretRepository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewSecretRepository creates a new secret repository
func NewSecretRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *SecretRepository {
	return &SecretRepository{db: db, cache: c, logger: logger}
}

// Create creates a new secret
func (r *SecretRepository) Create(ctx context.Context, secret *Secret) error {
	secret.ID = uuid.New()
	secret.CreatedAt = time.Now()
	secret.UpdatedAt = time.Now()
	secret.Status = "active"

	query := `
		INSERT INTO secrets (id, name, description, type, target_id, host, port, username,
			encrypted_secret, ssh_key_passphrase, rotation_policy, last_rotated_at, next_rotation_at,
			folder_id, tags, tenant_id, created_by, owner_id, status, metadata, created_at, updated_at)
		VALUES (:id, :name, :description, :type, :target_id, :host, :port, :username,
			:encrypted_secret, :ssh_key_passphrase, :rotation_policy, :last_rotated_at, :next_rotation_at,
			:folder_id, :tags, :tenant_id, :created_by, :owner_id, :status, :metadata, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, secret)
	if err != nil {
		return fmt.Errorf("secret.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a secret by ID
func (r *SecretRepository) GetByID(ctx context.Context, id uuid.UUID) (*Secret, error) {
	var secret Secret
	query := `SELECT * FROM secrets WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &secret, query, id)
	if err != nil {
		return nil, fmt.Errorf("secret.GetByID: %w", err)
	}
	return &secret, nil
}

// List retrieves secrets with pagination and filtering
func (r *SecretRepository) List(ctx context.Context, tenantID uuid.UUID, filter SecretFilter, limit, offset int) ([]Secret, int, error) {
	baseQuery := `
		SELECT * FROM secrets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	countQuery := `
		SELECT COUNT(*) FROM secrets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{tenantID}
	argCount := 2

	// Apply filters
	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.TargetID != nil {
		baseQuery += fmt.Sprintf(" AND target_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND target_id = $%d", argCount)
		args = append(args, *filter.TargetID)
		argCount++
	}
	if filter.FolderID != nil {
		baseQuery += fmt.Sprintf(" AND folder_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND folder_id = $%d", argCount)
		args = append(args, *filter.FolderID)
		argCount++
	}
	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.SearchTerm != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR host ILIKE $%d)", argCount, argCount, argCount)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR host ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+filter.SearchTerm+"%")
		argCount++
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("secret.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var secrets []Secret
	if err := r.db.SelectContext(ctx, &secrets, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("secret.List: %w", err)
	}

	return secrets, total, nil
}

// Update updates a secret
func (r *SecretRepository) Update(ctx context.Context, secret *Secret) error {
	secret.UpdatedAt = time.Now()

	query := `
		UPDATE secrets SET
			name = :name,
			description = :description,
			type = :type,
			target_id = :target_id,
			host = :host,
			port = :port,
			username = :username,
			encrypted_secret = :encrypted_secret,
			ssh_key_passphrase = :ssh_key_passphrase,
			rotation_policy = :rotation_policy,
			last_rotated_at = :last_rotated_at,
			next_rotation_at = :next_rotation_at,
			folder_id = :folder_id,
			tags = :tags,
			owner_id = :owner_id,
			status = :status,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	_, err := r.db.NamedExecContext(ctx, query, secret)
	if err != nil {
		return fmt.Errorf("secret.Update: %w", err)
	}

	return nil
}

// Delete soft deletes a secret
func (r *SecretRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE secrets SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("secret.Delete: %w", err)
	}
	return nil
}

// SecretFilter filters secret queries
type SecretFilter struct {
	Type       *SecretType
	TargetID   *uuid.UUID
	FolderID   *uuid.UUID
	Status     *string
	SearchTerm string
	Tags       []string
}

// VaultService handles secret business logic with encryption
type VaultService struct {
	repo    *SecretRepository
	envelope *EnvelopeEncryption
	logger  zerolog.Logger
}

// NewVaultService creates a new vault service
func NewVaultService(repo *SecretRepository, envelope *EnvelopeEncryption, logger zerolog.Logger) *VaultService {
	return &VaultService{
		repo:    repo,
		envelope: envelope,
		logger:  logger,
	}
}

// StoreSecret stores a new secret with envelope encryption
func (s *VaultService) StoreSecret(ctx context.Context, secret *Secret, plaintext SecretData) error {
	// Validate secret data
	if err := s.validateSecretData(plaintext); err != nil {
		return err
	}

	// Serialize secret data
	data, err := json.Marshal(plaintext)
	if err != nil {
		return fmt.Errorf("vault.Marshal: %w", err)
	}

	// Generate DEK
	dek, err := s.envelope.GenerateDEK()
	if err != nil {
		return fmt.Errorf("vault.GenerateDEK: %w", err)
	}

	// Encrypt secret with DEK
	encryptedData, err := s.encryptWithDEK(data, dek)
	if err != nil {
		return fmt.Errorf("vault.EncryptSecret: %w", err)
	}
	secret.EncryptedSecret = encryptedData

	// Encrypt DEK with master key
	encryptedDEK, err := s.envelope.EncryptDEK(dek)
	if err != nil {
		return fmt.Errorf("vault.EncryptDEK: %w", err)
	}
	secret.EncryptedSecretKey = string(encryptedDEK)

	// Set next rotation based on policy
	if secret.RotationPolicy != "" && secret.RotationPolicy != RotationManual {
		nextRotation := s.calculateNextRotation(time.Now(), secret.RotationPolicy)
		secret.NextRotationAt = &nextRotation
	}

	// Create secret record
	if err := s.repo.Create(ctx, secret); err != nil {
		return err
	}

	// Store encrypted DEK
	if err := s.envelope.StoreDEK(ctx, secret.ID.String(), encryptedDEK); err != nil {
		s.logger.Error().Err(err).Str("secret_id", secret.ID.String()).Msg("Failed to store DEK")
	}

	return nil
}

// RetrieveSecret retrieves and decrypts a secret
func (s *VaultService) RetrieveSecret(ctx context.Context, id uuid.UUID) (*SecretData, error) {
	// Get secret record
	secret, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check status
	if secret.Status != "active" {
		return nil, fmt.Errorf("vault: secret is %s", secret.Status)
	}

	// Retrieve and decrypt DEK
	dek, err := s.envelope.RetrieveDEK(ctx, secret.ID.String())
	if err != nil {
		return nil, fmt.Errorf("vault.RetrieveDEK: %w", err)
	}

	// Decrypt secret data with DEK
	data, err := s.decryptWithDEK(secret.EncryptedSecret, dek)
	if err != nil {
		return nil, fmt.Errorf("vault.DecryptSecret: %w", err)
	}

	// Deserialize secret data
	var plaintext SecretData
	if err := json.Unmarshal(data, &plaintext); err != nil {
		return nil, fmt.Errorf("vault.Unmarshal: %w", err)
	}

	// Log access
	s.logger.Info().
		Str("secret_id", id.String()).
		Str("secret_name", secret.Name).
		Msg("Secret retrieved")

	return &plaintext, nil
}

// UpdateSecret updates an existing secret
func (s *VaultService) UpdateSecret(ctx context.Context, id uuid.UUID, plaintext SecretData) error {
	// Get existing secret
	secret, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Serialize and encrypt new data
	data, err := json.Marshal(plaintext)
	if err != nil {
		return fmt.Errorf("vault.Marshal: %w", err)
	}

	// Generate new DEK (rotate on update)
	dek, err := s.envelope.GenerateDEK()
	if err != nil {
		return fmt.Errorf("vault.GenerateDEK: %w", err)
	}

	// Encrypt secret with new DEK
	encryptedData, err := s.encryptWithDEK(data, dek)
	if err != nil {
		return fmt.Errorf("vault.EncryptSecret: %w", err)
	}
	secret.EncryptedSecret = encryptedData

	// Encrypt new DEK
	encryptedDEK, err := s.envelope.EncryptDEK(dek)
	if err != nil {
		return fmt.Errorf("vault.EncryptDEK: %w", err)
	}
	secret.EncryptedSecretKey = string(encryptedDEK)

	// Update record
	if err := s.repo.Update(ctx, secret); err != nil {
		return err
	}

	// Store new encrypted DEK
	if err := s.envelope.StoreDEK(ctx, secret.ID.String(), encryptedDEK); err != nil {
		s.logger.Error().Err(err).Str("secret_id", secret.ID.String()).Msg("Failed to store DEK")
	}

	return nil
}

// DeleteSecret deletes a secret
func (s *VaultService) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// MarkCompromised marks a secret as compromised
func (s *VaultService) MarkCompromised(ctx context.Context, id uuid.UUID) error {
	secret, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	secret.Status = "compromised"
	return s.repo.Update(ctx, secret)
}

// RotateSecret rotates a secret (for automated rotation)
func (s *VaultService) RotateSecret(ctx context.Context, id uuid.UUID, newPlaintext SecretData) error {
	secret, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Update secret with new data
	if err := s.UpdateSecret(ctx, id, newPlaintext); err != nil {
		return err
	}

	// Update rotation timestamps
	now := time.Now()
	secret.LastRotatedAt = &now
	if secret.RotationPolicy != "" && secret.RotationPolicy != RotationManual {
		nextRotation := s.calculateNextRotation(now, secret.RotationPolicy)
		secret.NextRotationAt = &nextRotation
	}

	return s.repo.Update(ctx, secret)
}

// GetPendingRotation retrieves secrets pending rotation
func (s *VaultService) GetPendingRotation(ctx context.Context, tenantID uuid.UUID) ([]Secret, error) {
	query := `
		SELECT * FROM secrets
		WHERE tenant_id = $1
			AND status = 'active'
			AND next_rotation_at IS NOT NULL
			AND next_rotation_at <= NOW()
			AND deleted_at IS NULL
		ORDER BY next_rotation_at ASC
	`

	var secrets []Secret
	err := s.repo.db.SelectContext(ctx, &secrets, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("vault.GetPendingRotation: %w", err)
	}

	return secrets, nil
}

func (s *VaultService) encryptWithDEK(data []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
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

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *VaultService) decryptWithDEK(data []byte, dek []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *VaultService) validateSecretData(data SecretData) error {
	if data.Username == "" {
		return fmt.Errorf("vault: username is required")
	}

	switch data.Type {
	case SecretTypePassword:
		if data.Password == "" {
			return fmt.Errorf("vault: password is required")
		}
	case SecretTypeSSHKey:
		if data.PrivateKey == "" {
			return fmt.Errorf("vault: private key is required")
		}
	case SecretTypeAPIToken, SecretTypeAWSKey, SecretTypeAzureKey:
		if data.Token == "" && data.SecretKey == "" {
			return fmt.Errorf("vault: token or secret key is required")
		}
	}

	return nil
}

func (s *VaultService) calculateNextRotation(from time.Time, policy RotationPolicy) time.Time {
	switch policy {
	case RotationDaily:
		return from.AddDate(0, 0, 1)
	case RotationWeekly:
		return from.AddDate(0, 0, 7)
	case RotationMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 30)
	}
}
