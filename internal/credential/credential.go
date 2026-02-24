package credential

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/openpam/openpam/internal/pam/rotation"
	"github.com/rs/zerolog"
)

// Service handles credential business operations
type Service struct {
	db      *sqlx.DB
	vault   *vault.VaultService
	rotation *rotation.Service
	cache   *cache.Cache
	logger  zerolog.Logger
}

// NewService creates a new credential service
func NewService(db *sqlx.DB, vaultSvc *vault.VaultService, rotationSvc *rotation.Service, c *cache.Cache, logger zerolog.Logger) *Service {
	return &Service{
		db:       db,
		vault:    vaultSvc,
		rotation: rotationSvc,
		cache:    c,
		logger:   logger,
	}
}

// CreateCredential creates a new credential in the vault
func (s *Service) CreateCredential(ctx context.Context, secret *vault.Secret, plaintext vault.SecretData) error {
	// Validate input
	if err := s.validateSecret(secret); err != nil {
		return err
	}

	// Store in vault with encryption
	if err := s.vault.StoreSecret(ctx, secret, plaintext); err != nil {
		return fmt.Errorf("credential.StoreSecret: %w", err)
	}

	// Schedule rotation if needed
	// SECURITY: Fail credential creation if rotation scheduling fails for non-manual policies.
	// Credentials without rotation scheduling could persist indefinitely without being rotated,
	// violating the principle of zero standing privileges and compliance requirements.
	if secret.RotationPolicy != "" && secret.RotationPolicy != vault.RotationManual {
		if err := s.rotation.ScheduleRotation(ctx, secret.ID, secret.RotationPolicy); err != nil {
			// Rollback: delete the credential that was already stored since rotation scheduling failed
			if deleteErr := s.vault.DeleteSecret(ctx, secret.ID); deleteErr != nil {
				s.logger.Error().
					Err(deleteErr).
					Str("credential_id", secret.ID.String()).
					Msg("Failed to rollback credential after rotation scheduling error")
			}
			return fmt.Errorf("credential.ScheduleRotation: %w (credential rolled back)", err)
		}
	}

	s.logger.Info().
		Str("credential_id", secret.ID.String()).
		Str("name", secret.Name).
		Str("type", string(secret.Type)).
		Str("rotation_policy", string(secret.RotationPolicy)).
		Msg("Credential created")

	return nil
}

// GetCredential retrieves a credential (without secret data)
func (s *Service) GetCredential(ctx context.Context, id uuid.UUID) (*vault.Secret, error) {
	repo := vault.NewSecretRepository(s.db, s.cache, s.logger)
	return repo.GetByID(ctx, id)
}

// ListCredentials lists credentials for a tenant
func (s *Service) ListCredentials(ctx context.Context, tenantID uuid.UUID, filter vault.SecretFilter, limit, offset int) ([]vault.Secret, int, error) {
	repo := vault.NewSecretRepository(s.db, s.cache, s.logger)
	return repo.List(ctx, tenantID, filter, limit, offset)
}

// UpdateCredential updates a credential
func (s *Service) UpdateCredential(ctx context.Context, id uuid.UUID, plaintext vault.SecretData) error {
	return s.vault.UpdateSecret(ctx, id, plaintext)
}

// DeleteCredential deletes a credential
func (s *Service) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	return s.vault.DeleteSecret(ctx, id)
}

// RotateCredential rotates a credential immediately
func (s *Service) RotateCredential(ctx context.Context, id uuid.UUID, newPlaintext vault.SecretData) error {
	return s.vault.RotateSecret(ctx, id, newPlaintext)
}

// TestCredential tests if a credential is valid
func (s *Service) TestCredential(ctx context.Context, id uuid.UUID) error {
	repo := vault.NewSecretRepository(s.db, s.cache, s.logger)
	secret, err := repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Based on credential type, test connection
	return s.testCredentialConnection(ctx, secret)
}

// GetCredentialHistory retrieves rotation history for a credential
func (s *Service) GetCredentialHistory(ctx context.Context, credentialID uuid.UUID, limit int) ([]CredentialHistoryEntry, error) {
	query := `
		SELECT * FROM credential_history
		WHERE credential_id = $1
		ORDER BY rotated_at DESC
		LIMIT $2
	`

	var entries []CredentialHistoryEntry
	err := s.db.SelectContext(ctx, &entries, query, credentialID, limit)
	if err != nil {
		return nil, fmt.Errorf("credential.GetHistory: %w", err)
	}

	return entries, nil
}

// CredentialHistoryEntry represents a credential rotation history entry
type CredentialHistoryEntry struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	CredentialID uuid.UUID    `db:"credential_id" json:"credential_id"`
	RotatedAt    time.Time    `db:"rotated_at" json:"rotated_at"`
	RotatedBy    uuid.UUID    `db:"rotated_by" json:"rotated_by"`
	Method       string       `db:"method" json:"method"` // manual, scheduled, on_checkin
	Success      bool         `db:"success" json:"success"`
	ErrorMsg     string       `db:"error_msg" json:"error_msg,omitempty"`
}

// validateSecret validates credential data
func (s *Service) validateSecret(secret *vault.Secret) error {
	if secret.Name == "" {
		return fmt.Errorf("credential: name is required")
	}
	if secret.Type == "" {
		return fmt.Errorf("credential: type is required")
	}
	if secret.TenantID == uuid.Nil {
		return fmt.Errorf("credential: tenant ID is required")
	}
	if secret.Host == "" {
		return fmt.Errorf("credential: host is required")
	}
	if secret.Port <= 0 {
		return fmt.Errorf("credential: port is required")
	}
	return nil
}

// testCredentialConnection tests if a credential works
func (s *Service) testCredentialConnection(ctx context.Context, secret *vault.Secret) error {
	// Implement connection testing based on credential type
	// For SSH: try SSH connection
	// For database: try database connection
	// For API: try API call

	return fmt.Errorf("credential: connection testing not implemented")
}

// GetCredentialsRequiringRotation returns credentials that need rotation
func (s *Service) GetCredentialsRequiringRotation(ctx context.Context, tenantID uuid.UUID) ([]vault.Secret, error) {
	return s.vault.GetPendingRotation(ctx, tenantID)
}

// BatchRotateCredentials rotates multiple credentials
func (s *Service) BatchRotateCredentials(ctx context.Context, credentialIDs []uuid.UUID) (map[uuid.UUID]error, []vault.Secret) {
	results := make(map[uuid.UUID]error)
	var rotated []vault.Secret

	repo := vault.NewSecretRepository(s.db, s.cache, s.logger)

	for _, id := range credentialIDs {
		secret, err := repo.GetByID(ctx, id)
		if err != nil {
			results[id] = err
			continue
		}

		// Rotate - the rotation service will generate and apply new credentials
		if err := s.rotation.RotateCredential(ctx, id); err != nil {
			results[id] = err
			continue
		}

		results[id] = nil
		rotated = append(rotated, *secret)
	}

	return results, rotated
}

// GetCredentialUsageStats returns usage statistics for a credential
func (s *Service) GetCredentialUsageStats(ctx context.Context, credentialID uuid.UUID) (*CredentialStats, error) {
	// Get checkout count
	checkoutQuery := `
		SELECT COUNT(*) as count FROM checkouts
		WHERE credential_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
	`
	var checkoutCount int
	if err := s.db.GetContext(ctx, &checkoutCount, checkoutQuery, credentialID); err != nil {
		return nil, err
	}

	// Get session count
	sessionQuery := `
		SELECT COUNT(*) as count FROM sessions
		WHERE credential_id = $1 AND started_at >= NOW() - INTERVAL '30 days'
	`
	var sessionCount int
	if err := s.db.GetContext(ctx, &sessionCount, sessionQuery, credentialID); err != nil {
		return nil, err
	}

	// Get last used
	lastUsedQuery := `
		SELECT MAX(checked_out_at) as last_used FROM checkouts
		WHERE credential_id = $1
	`
	var lastUsed *time.Time
	if err := s.db.GetContext(ctx, &lastUsed, lastUsedQuery, credentialID); err != nil {
		return nil, err
	}

	// Get top users
	usersQuery := `
		SELECT user_id, COUNT(*) as count
		FROM checkouts
		WHERE credential_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY user_id
		ORDER BY count DESC
		LIMIT 5
	`
	type dbUserStat struct {
		UserID uuid.UUID `db:"user_id"`
		Count  int       `db:"count"`
	}
	var dbUsers []dbUserStat
	if err := s.db.SelectContext(ctx, &dbUsers, usersQuery, credentialID); err != nil {
		return nil, err
	}

	// Convert to json-annotated struct
	topUsers := make([]struct {
		UserID uuid.UUID `json:"user_id"`
		Count  int       `json:"count"`
	}, len(dbUsers))
	for i, u := range dbUsers {
		topUsers[i] = struct {
			UserID uuid.UUID `json:"user_id"`
			Count  int       `json:"count"`
		}{
			UserID: u.UserID,
			Count:  u.Count,
		}
	}

	return &CredentialStats{
		CredentialID:     credentialID,
		CheckoutCount30d: checkoutCount,
		SessionCount30d:  sessionCount,
		LastUsedAt:       lastUsed,
		TopUsers:         topUsers,
	}, nil
}

// CredentialStats represents credential usage statistics
type CredentialStats struct {
	CredentialID     uuid.UUID `json:"credential_id"`
	CheckoutCount30d int       `json:"checkout_count_30d"`
	SessionCount30d  int       `json:"session_count_30d"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	TopUsers         []struct {
		UserID uuid.UUID `json:"user_id"`
		Count  int       `json:"count"`
	} `json:"top_users"`
}

// SyncCredential syncs a credential with a target system
func (s *Service) SyncCredential(ctx context.Context, credentialID uuid.UUID, targetID uuid.UUID) error {
	// Retrieve credential metadata
	repo := vault.NewSecretRepository(s.db, s.cache, s.logger)
	secret, err := repo.GetByID(ctx, credentialID)
	if err != nil {
		return err
	}

	// Apply credential to target
	// This would use platform-specific connectors (Linux, Windows AD, databases, etc.)
	return s.applyCredentialToTarget(ctx, secret, targetID)
}

func (s *Service) applyCredentialToTarget(ctx context.Context, secret *vault.Secret, targetID uuid.UUID) error {
	// Implement platform-specific credential application
	return fmt.Errorf("credential: target sync not implemented")
}
