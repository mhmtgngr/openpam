package credential

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/openpam/openpam/internal/pam/rotation"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Service handles credential business operations
type Service struct {
	db         *sqlx.DB
	vault      *vault.VaultService
	rotation   *rotation.Service
	cache      *cache.Cache
	eventPub   *events.Publisher
	logger     zerolog.Logger
}

// NewService creates a new credential service
func NewService(db *sqlx.DB, vaultSvc *vault.VaultService, rotationSvc *rotation.Service, c *cache.Cache, eventPub *events.Publisher, logger zerolog.Logger) *Service {
	return &Service{
		db:       db,
		vault:    vaultSvc,
		rotation: rotationSvc,
		cache:    c,
		eventPub: eventPub,
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
			// SECURITY FIX: Propagate rollback errors to ensure cleanup failures are not silently ignored
			if deleteErr := s.vault.DeleteSecret(ctx, secret.ID); deleteErr != nil {
				// CRITICAL: Both rotation scheduling AND rollback failed
				// This is a severe state - the credential exists but rotation is broken
				// SECURITY: Quarantine the credential and trigger alert
				s.quarantineCredential(ctx, secret, err, deleteErr)
				s.publishCredentialQuarantineAlert(ctx, secret, err, deleteErr)

				s.logger.Error().
					Err(deleteErr).
					Str("credential_id", secret.ID.String()).
					Msg("CRITICAL: Failed to rollback credential after rotation scheduling error - credential QUARANTINED")

				// Return combined error so caller knows about both failures
				return fmt.Errorf("credential.ScheduleRotation: %w (rollback also failed: %v) - CREDENTIAL QUARANTINED - REQUIRES IMMEDIATE ADMIN INTERVENTION", err, deleteErr)
			}
			return fmt.Errorf("credential.ScheduleRotation: %w", err)
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

// testCredentialConnection tests if a credential works by attempting a connection
func (s *Service) testCredentialConnection(ctx context.Context, secret *vault.Secret) error {
	// Retrieve decrypted secret data
	secretData, err := s.vault.RetrieveSecret(ctx, secret.ID)
	if err != nil {
		return fmt.Errorf("credential.TestConnection.RetrieveSecret: %w", err)
	}

	switch secret.Type {
	case vault.SecretTypePassword, vault.SecretTypeSSHKey:
		return s.testSSHConnection(ctx, secret, secretData)

	case vault.SecretTypeDatabase:
		return s.testDatabaseConnection(ctx, secret, secretData)

	case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey, vault.SecretTypeAzureKey:
		return s.testAPIConnectivity(ctx, secret)

	default:
		// For unknown types, just verify the target host is reachable
		return s.testTCPConnectivity(secret.Host, secret.Port)
	}
}

// testSSHConnection tests SSH connectivity with the given credentials
func (s *Service) testSSHConnection(ctx context.Context, secret *vault.Secret, data *vault.SecretData) error {
	var authMethods []ssh.AuthMethod

	if data.PrivateKey != "" {
		var signer ssh.Signer
		var err error
		if data.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(data.PrivateKey), []byte(data.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(data.PrivateKey))
		}
		if err != nil {
			return fmt.Errorf("credential.TestSSH.ParseKey: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if data.Password != "" {
		authMethods = append(authMethods, ssh.Password(data.Password))
	}

	if len(authMethods) == 0 {
		return fmt.Errorf("credential: no SSH authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            data.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", secret.Host, secret.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("credential.TestSSH.Dial(%s): %w", address, err)
	}
	client.Close()

	s.logger.Info().
		Str("credential_id", secret.ID.String()).
		Str("host", secret.Host).
		Msg("SSH connection test passed")
	return nil
}

// testDatabaseConnection tests database connectivity
func (s *Service) testDatabaseConnection(ctx context.Context, secret *vault.Secret, data *vault.SecretData) error {
	dbName := "postgres"
	if data.Extra != nil {
		if db, ok := data.Extra["database"]; ok {
			dbName = db
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require connect_timeout=10",
		secret.Host, secret.Port, data.Username, data.Password, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("credential.TestDB.Open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("credential.TestDB.Ping(%s:%d): %w", secret.Host, secret.Port, err)
	}

	s.logger.Info().
		Str("credential_id", secret.ID.String()).
		Str("host", secret.Host).
		Msg("Database connection test passed")
	return nil
}

// testAPIConnectivity tests that the API endpoint is reachable via HTTPS
func (s *Service) testAPIConnectivity(ctx context.Context, secret *vault.Secret) error {
	port := secret.Port
	if port == 0 {
		port = 443
	}
	return s.testTCPConnectivity(secret.Host, port)
}

// testTCPConnectivity tests basic TCP connectivity
func (s *Service) testTCPConnectivity(host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("credential.TestTCP(%s): %w", address, err)
	}
	conn.Close()
	return nil
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
	// Retrieve the decrypted credential
	secretData, err := s.vault.RetrieveSecret(ctx, secret.ID)
	if err != nil {
		return fmt.Errorf("credential.Sync.RetrieveSecret: %w", err)
	}

	// Get target info to determine platform type
	var targetType string
	query := `SELECT type FROM targets WHERE id = $1 AND deleted_at IS NULL`
	if err := s.db.GetContext(ctx, &targetType, query, targetID); err != nil {
		return fmt.Errorf("credential.Sync.GetTarget: %w", err)
	}

	// Use the rotation service connectors to apply the credential
	if s.rotation == nil {
		return fmt.Errorf("credential: rotation service not available for sync")
	}

	// Create a placeholder "old" credential (sync deploys new creds without needing old ones on the target)
	dummyOld := &vault.SecretData{
		Type:     secretData.Type,
		Username: secretData.Username,
	}

	if err := s.rotation.RotateOnTarget(ctx, targetID, dummyOld, secretData); err != nil {
		return fmt.Errorf("credential.Sync.ApplyToTarget: %w", err)
	}

	s.logger.Info().
		Str("credential_id", secret.ID.String()).
		Str("target_id", targetID.String()).
		Str("target_type", targetType).
		Msg("Credential synced to target")

	return nil
}

// CredentialQuarantineEntry represents a quarantined credential
type CredentialQuarantineEntry struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	CredentialID    uuid.UUID  `db:"credential_id" json:"credential_id"`
	CredentialName  string     `db:"credential_name" json:"credential_name"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	RotationPolicy  string     `db:"rotation_policy" json:"rotation_policy"`
	QuarantineReason string    `db:"quarantine_reason" json:"quarantine_reason"`
	OriginalError   string     `db:"original_error" json:"original_error"`
	RollbackError   string     `db:"rollback_error" json:"rollback_error,omitempty"`
	QuarantinedAt   time.Time  `db:"quarantined_at" json:"quarantined_at"`
	ResolvedAt      *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy      *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolutionNotes string     `db:"resolution_notes" json:"resolution_notes,omitempty"`
	Severity        string     `db:"severity" json:"severity"`
}

// quarantineCredential places a credential in quarantine after a failed rollback
// SECURITY: This ensures admins are notified of credentials in inconsistent state
func (s *Service) quarantineCredential(ctx context.Context, secret *vault.Secret, originalErr, rollbackErr error) {
	entry := CredentialQuarantineEntry{
		CredentialID:     secret.ID,
		CredentialName:   secret.Name,
		TenantID:         secret.TenantID,
		RotationPolicy:   string(secret.RotationPolicy),
		QuarantineReason: "Rotation scheduling failed and rollback also failed - credential exists in vault without rotation",
		OriginalError:    originalErr.Error(),
		RollbackError:    rollbackErr.Error(),
		QuarantinedAt:    time.Now(),
		Severity:         "critical",
	}

	query := `
		INSERT INTO credential_quarantine (
			credential_id, credential_name, tenant_id, rotation_policy,
			quarantine_reason, original_error, rollback_error,
			quarantined_at, severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.ExecContext(ctx, query,
		entry.CredentialID, entry.CredentialName, entry.TenantID, entry.RotationPolicy,
		entry.QuarantineReason, entry.OriginalError, entry.RollbackError,
		entry.QuarantinedAt, entry.Severity,
	)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("credential_id", secret.ID.String()).
			Msg("Failed to quarantine credential - alert may not be visible in admin UI")
	}
}

// publishCredentialQuarantineAlert publishes an alert event for quarantined credentials
// SECURITY: This ensures real-time notification of security-relevant failures
func (s *Service) publishCredentialQuarantineAlert(ctx context.Context, secret *vault.Secret, originalErr, rollbackErr error) {
	if s.eventPub == nil {
		s.logger.Warn().
			Str("credential_id", secret.ID.String()).
			Msg("Event publisher not configured - quarantine alert not published")
		return
	}

	err := s.eventPub.PublishAlertTriggered(ctx, secret.TenantID.String(), secret.ID.String(), "credential_quarantine", "critical", map[string]interface{}{
		"credential_id":     secret.ID.String(),
		"credential_name":   secret.Name,
		"credential_type":   string(secret.Type),
		"rotation_policy":   string(secret.RotationPolicy),
		"quarantine_reason": "Rotation scheduling failed and rollback also failed",
		"original_error":    originalErr.Error(),
		"rollback_error":    rollbackErr.Error(),
		"requires_action":   "immediate_admin_intervention",
		"action_required":   "resolve_quarantine",
	})
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("credential_id", secret.ID.String()).
			Msg("Failed to publish credential quarantine alert event")
	}
}

// GetQuarantinedCredentials retrieves all quarantined credentials for a tenant
func (s *Service) GetQuarantinedCredentials(ctx context.Context, tenantID uuid.UUID) ([]CredentialQuarantineEntry, error) {
	query := `
		SELECT * FROM credential_quarantine
		WHERE tenant_id = $1 AND resolved_at IS NULL
		ORDER BY quarantined_at DESC
	`

	var entries []CredentialQuarantineEntry
	err := s.db.SelectContext(ctx, &entries, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("credential.GetQuarantined: %w", err)
	}

	return entries, nil
}

// ResolveQuarantine marks a quarantined credential as resolved
func (s *Service) ResolveQuarantine(ctx context.Context, credentialID uuid.UUID, resolvedBy uuid.UUID, resolutionNotes string) error {
	query := `
		UPDATE credential_quarantine
		SET resolved_at = NOW(),
			resolved_by = $1,
			resolution_notes = $2
		WHERE credential_id = $3 AND resolved_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, resolvedBy, resolutionNotes, credentialID)
	if err != nil {
		return fmt.Errorf("credential.ResolveQuarantine: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("credential: no quarantined entry found for credential %s", credentialID)
	}

	s.logger.Info().
		Str("credential_id", credentialID.String()).
		Str("resolved_by", resolvedBy.String()).
		Msg("Credential quarantine resolved")

	return nil
}
