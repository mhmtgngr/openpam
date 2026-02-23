package checkout

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/rs/zerolog"
)

// CheckoutStatus represents the status of a checkout
type CheckoutStatus string

const (
	CheckoutStatusPending   CheckoutStatus = "pending"
	CheckoutStatusApproved  CheckoutStatus = "approved"
	CheckoutStatusDenied    CheckoutStatus = "denied"
	CheckoutStatusCheckedOut CheckoutStatus = "checked_out"
	CheckoutStatusCheckedIn CheckoutStatus = "checked_in"
	CheckoutStatusExpired   CheckoutStatus = "expired"
	CheckoutStatusRevoked   CheckoutStatus = "revoked"
)

// Checkout represents a credential checkout session
type Checkout struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	UserID         uuid.UUID       `db:"user_id" json:"user_id"`
	CredentialID   uuid.UUID       `db:"credential_id" json:"credential_id"`
	RequestID      *uuid.UUID      `db:"request_id" json:"request_id,omitempty"`
	Justification  string          `db:"justification" json:"justification"`
	Duration       int             `db:"duration_minutes" json:"duration_minutes"`
	Status         CheckoutStatus  `db:"status" json:"status"`

	// Approval info
	ApprovedBy     *uuid.UUID      `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt     *time.Time      `db:"approved_at" json:"approved_at,omitempty"`

	// Checkout info
	CheckedOutAt   *time.Time      `db:"checked_out_at" json:"checked_out_at,omitempty"`
	ExpiresAt      *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	CheckedInAt    *time.Time      `db:"checked_in_at" json:"checked_in_at,omitempty"`
	RevokedAt      *time.Time      `db:"revoked_at" json:"revoked_at,omitempty"`
	RevokedBy      *uuid.UUID      `db:"revoked_by" json:"revoked_by,omitempty"`
	RevokedReason  string          `db:"revoked_reason" json:"revoked_reason,omitempty"`

	// Session tracking
	SessionID      *uuid.UUID      `db:"session_id" json:"session_id,omitempty"`

	// Break-glass
	IsBreakGlass   bool            `db:"is_break_glass" json:"is_break_glass"`
	BreakGlassReason string         `db:"break_glass_reason" json:"break_glass_reason,omitempty"`

	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updated_at"`
}

// CheckoutRecord tracks credential access for audit
type CheckoutRecord struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	CheckoutID   uuid.UUID    `db:"checkout_id" json:"checkout_id"`
	UserID       uuid.UUID    `db:"user_id" json:"user_id"`
	CredentialID uuid.UUID    `db:"credential_id" json:"credential_id"`
	Action       string       `db:"action" json:"action"` // checkout, checkin, view
	ClientIP     string       `db:"client_ip" json:"client_ip"`
	UserAgent    string       `db:"user_agent" json:"user_agent"`
	Timestamp    time.Time    `db:"timestamp" json:"timestamp"`
}

// Repository handles checkout data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new checkout repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	return &Repository{db: db, cache: c, logger: logger}
}

// Create creates a new checkout request
func (r *Repository) Create(ctx context.Context, checkout *Checkout) error {
	checkout.ID = uuid.New()
	checkout.CreatedAt = time.Now()
	checkout.UpdatedAt = time.Now()
	checkout.Status = CheckoutStatusPending

	query := `
		INSERT INTO checkouts (id, user_id, credential_id, request_id, justification,
			duration_minutes, status, approved_by, approved_at, checked_out_at, expires_at,
			checked_in_at, is_break_glass, break_glass_reason, tenant_id, created_at, updated_at)
		VALUES (:id, :user_id, :credential_id, :request_id, :justification,
			:duration_minutes, :status, :approved_by, :approved_at, :checked_out_at, :expires_at,
			:checked_in_at, :is_break_glass, :break_glass_reason, :tenant_id, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, checkout)
	if err != nil {
		return fmt.Errorf("checkout.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a checkout by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Checkout, error) {
	var checkout Checkout
	query := `SELECT * FROM checkouts WHERE id = $1`
	err := r.db.GetContext(ctx, &checkout, query, id)
	if err != nil {
		return nil, fmt.Errorf("checkout.GetByID: %w", err)
	}
	return &checkout, nil
}

// List retrieves checkouts with filtering
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, filter CheckoutFilter, limit, offset int) ([]Checkout, int, error) {
	baseQuery := `
		SELECT * FROM checkouts
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM checkouts WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}
	if filter.CredentialID != nil {
		baseQuery += fmt.Sprintf(" AND credential_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND credential_id = $%d", argCount)
		args = append(args, *filter.CredentialID)
		argCount++
	}
	if filter.Active != nil && *filter.Active {
		baseQuery += " AND status = 'checked_out' AND (expires_at IS NULL OR expires_at > NOW())"
		countQuery += " AND status = 'checked_out' AND (expires_at IS NULL OR expires_at > NOW())"
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("checkout.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var checkouts []Checkout
	if err := r.db.SelectContext(ctx, &checkouts, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("checkout.List: %w", err)
	}

	return checkouts, total, nil
}

// Update updates a checkout
func (r *Repository) Update(ctx context.Context, checkout *Checkout) error {
	checkout.UpdatedAt = time.Now()

	query := `
		UPDATE checkouts SET
			status = :status,
			approved_by = :approved_by,
			approved_at = :approved_at,
			checked_out_at = :checked_out_at,
			expires_at = :expires_at,
			checked_in_at = :checked_in_at,
			revoked_at = :revoked_at,
			revoked_by = :revoked_by,
			revoked_reason = :revoked_reason,
			session_id = :session_id,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, checkout)
	if err != nil {
		return fmt.Errorf("checkout.Update: %w", err)
	}

	return nil
}

// GetActiveCheckoutForCredential retrieves active checkout for a credential
func (r *Repository) GetActiveCheckoutForCredential(ctx context.Context, credentialID uuid.UUID) (*Checkout, error) {
	var checkout Checkout
	query := `
		SELECT * FROM checkouts
		WHERE credential_id = $1
			AND status = 'checked_out'
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY checked_out_at DESC
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &checkout, query, credentialID)
	if err != nil {
		return nil, fmt.Errorf("checkout.GetActiveCheckout: %w", err)
	}
	return &checkout, nil
}

// CheckoutFilter filters checkout queries
type CheckoutFilter struct {
	Status       *CheckoutStatus
	UserID       *uuid.UUID
	CredentialID *uuid.UUID
	Active       *bool
}

// Service handles checkout business logic
type Service struct {
	repo     *Repository
	vault    *vault.VaultService
	publisher *events.Publisher
	cache    *cache.Cache
	logger   zerolog.Logger
}

// NewService creates a new checkout service
func NewService(repo *Repository, vault *vault.VaultService, publisher *events.Publisher, c *cache.Cache, logger zerolog.Logger) *Service {
	return &Service{
		repo:     repo,
		vault:    vault,
		publisher: publisher,
		cache:    c,
		logger:   logger,
	}
}

// RequestCheckout creates a checkout request
func (s *Service) RequestCheckout(ctx context.Context, userID, credentialID uuid.UUID, justification string, duration int, isBreakGlass bool, breakGlassReason string) (*Checkout, error) {
	// Create checkout request
	checkout := &Checkout{
		UserID:          userID,
		CredentialID:    credentialID,
		Justification:   justification,
		Duration:        duration,
		IsBreakGlass:    isBreakGlass,
		BreakGlassReason: breakGlassReason,
	}

	// Get tenant ID from user context (would be passed in)
	// For now, use a default
	checkout.TenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

	// Handle break-glass access
	if isBreakGlass {
		// Break-glass requires additional justification and logging
		checkout.Status = CheckoutStatusApproved
		now := time.Now()
		checkout.ApprovedAt = &now
		checkout.CheckedOutAt = &now
		expiresAt := now.Add(time.Duration(duration) * time.Minute)
		checkout.ExpiresAt = &expiresAt
		checkout.Status = CheckoutStatusCheckedOut

		s.logger.Warn().
			Str("user_id", userID.String()).
			Str("credential_id", credentialID.String()).
			Str("reason", breakGlassReason).
			Msg("Break-glass access used")

		// Publish alert event
		_ = s.publisher.PublishAlertTriggered(ctx, checkout.TenantID.String(),
			uuid.New().String(), "break_glass_access", "high",
			map[string]interface{}{
				"user_id":   userID.String(),
				"credential_id": credentialID.String(),
				"reason":    breakGlassReason,
			})
	}

	if err := s.repo.Create(ctx, checkout); err != nil {
		return nil, err
	}

	return checkout, nil
}

// ApproveCheckout approves a pending checkout request
func (s *Service) ApproveCheckout(ctx context.Context, checkoutID, approverID uuid.UUID) error {
	checkout, err := s.repo.GetByID(ctx, checkoutID)
	if err != nil {
		return err
	}

	if checkout.Status != CheckoutStatusPending {
		return fmt.Errorf("checkout: not in pending status")
	}

	now := time.Now()
	checkout.Status = CheckoutStatusApproved
	checkout.ApprovedBy = &approverID
	checkout.ApprovedAt = &now

	return s.repo.Update(ctx, checkout)
}

// DenyCheckout denies a checkout request
func (s *Service) DenyCheckout(ctx context.Context, checkoutID, approverID uuid.UUID, reason string) error {
	checkout, err := s.repo.GetByID(ctx, checkoutID)
	if err != nil {
		return err
	}

	if checkout.Status != CheckoutStatusPending {
		return fmt.Errorf("checkout: not in pending status")
	}

	checkout.Status = CheckoutStatusDenied

	return s.repo.Update(ctx, checkout)
}

// CheckoutCredential checks out a credential (returns the actual secret)
func (s *Service) CheckoutCredential(ctx context.Context, checkoutID uuid.UUID) (*vault.SecretData, error) {
	checkout, err := s.repo.GetByID(ctx, checkoutID)
	if err != nil {
		return nil, err
	}

	if checkout.Status != CheckoutStatusApproved {
		return nil, fmt.Errorf("checkout: not approved")
	}

	// Check for concurrent checkout
	active, err := s.repo.GetActiveCheckoutForCredential(ctx, checkout.CredentialID)
	if err == nil && active.ID != checkout.ID {
		return nil, fmt.Errorf("checkout: credential already checked out")
	}

	// Get secret from vault
	secret, err := s.vault.RetrieveSecret(ctx, checkout.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("checkout.RetrieveSecret: %w", err)
	}

	// Update checkout status
	now := time.Now()
	checkout.Status = CheckoutStatusCheckedOut
	checkout.CheckedOutAt = &now
	expiresAt := now.Add(time.Duration(checkout.Duration) * time.Minute)
	checkout.ExpiresAt = &expiresAt

	if err := s.repo.Update(ctx, checkout); err != nil {
		return nil, err
	}

	// Cache checkout for quick expiration checks
	cacheKey := fmt.Sprintf("checkout:%s", checkout.ID)
	_ = s.cache.Set(ctx, cacheKey, checkout, time.Duration(checkout.Duration)*time.Minute)

	// Record checkout
	s.recordAccess(ctx, checkout, "checkout")

	// Publish event
	_ = s.publisher.PublishCheckoutApproved(ctx, checkout.TenantID.String(),
		checkout.UserID.String(), checkout.ID.String())

	return secret, nil
}

// CheckinCredential checks in a credential
func (s *Service) CheckinCredential(ctx context.Context, checkoutID uuid.UUID) error {
	checkout, err := s.repo.GetByID(ctx, checkoutID)
	if err != nil {
		return err
	}

	if checkout.Status != CheckoutStatusCheckedOut {
		return fmt.Errorf("checkout: not checked out")
	}

	now := time.Now()
	checkout.Status = CheckoutStatusCheckedIn
	checkout.CheckedInAt = &now

	if err := s.repo.Update(ctx, checkout); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("checkout:%s", checkout.ID)
	_ = s.cache.Delete(ctx, cacheKey)

	// Record checkin
	s.recordAccess(ctx, checkout, "checkin")

	return nil
}

// ForceCheckin forces checkin of a credential (admin operation)
func (s *Service) ForceCheckin(ctx context.Context, credentialID, revokedBy uuid.UUID, reason string) error {
	checkout, err := s.repo.GetActiveCheckoutForCredential(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("checkout: no active checkout found")
	}

	now := time.Now()
	checkout.Status = CheckoutStatusCheckedIn
	checkout.CheckedInAt = &now
	checkout.RevokedAt = &now
	checkout.RevokedBy = &revokedBy
	checkout.RevokedReason = reason

	if err := s.repo.Update(ctx, checkout); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("checkout:%s", checkout.ID)
	_ = s.cache.Delete(ctx, cacheKey)

	s.logger.Warn().
		Str("checkout_id", checkout.ID.String()).
		Str("revoked_by", revokedBy.String()).
		Str("reason", reason).
		Msg("Credential force checked in")

	return nil
}

// ListActiveCheckouts lists all currently checked out credentials
func (s *Service) ListActiveCheckouts(ctx context.Context, tenantID uuid.UUID) ([]Checkout, error) {
	checkouts, _, err := s.repo.List(ctx, tenantID, CheckoutFilter{Active: boolPtr(true)}, 1000, 0)
	return checkouts, err
}

// ExpireCheckouts expires checkouts that have passed their expiration time
func (s *Service) ExpireCheckouts(ctx context.Context) error {
	query := `
		UPDATE checkouts
		SET status = 'expired', checked_in_at = NOW(), updated_at = NOW()
		WHERE status = 'checked_out'
			AND expires_at IS NOT NULL
			AND expires_at <= NOW()
	`
	_, err := s.repo.db.ExecContext(ctx, query)
	return err
}

// recordAccess records a checkout/access event
func (s *Service) recordAccess(ctx context.Context, checkout *Checkout, action string) {
	record := &CheckoutRecord{
		ID:           uuid.New(),
		CheckoutID:   checkout.ID,
		UserID:       checkout.UserID,
		CredentialID: checkout.CredentialID,
		Action:       action,
		Timestamp:    time.Now(),
	}

	query := `
		INSERT INTO checkout_records (id, checkout_id, user_id, credential_id, action, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.repo.db.ExecContext(ctx, query, record.ID, record.CheckoutID, record.UserID,
		record.CredentialID, record.Action, record.Timestamp)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to record checkout access")
	}
}

func boolPtr(b bool) *bool {
	return &b
}
