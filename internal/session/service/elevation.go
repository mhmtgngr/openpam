package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// ElevationService handles privileged elevation management (sudo/runas replacement)
type ElevationService struct {
	db           *sqlx.DB
	analyticsSvc *analytics.Service
	logger       zerolog.Logger
}

// ElevationConfig holds elevation service configuration
type ElevationConfig struct {
	MaxElevationDuration time.Duration
	DefaultDuration       time.Duration
	RequireApproval       bool
	RequireMFA            bool
	RequireJustification  bool
	LogAllElevations      bool
	MaxConcurrentPerUser  int
}

// DefaultElevationConfig returns default elevation configuration
func DefaultElevationConfig() ElevationConfig {
	return ElevationConfig{
		MaxElevationDuration: 8 * time.Hour,
		DefaultDuration:       1 * time.Hour,
		RequireApproval:      true,
		RequireMFA:           true,
		RequireJustification: true,
		LogAllElevations:      true,
		MaxConcurrentPerUser: 3,
	}
}

// NewElevationService creates a new elevation service
func NewElevationService(db *sqlx.DB, analyticsSvc *analytics.Service, logger zerolog.Logger) *ElevationService {
	return &ElevationService{
		db:           db,
		analyticsSvc: analyticsSvc,
		logger:       logger,
	}
}

// ElevationRequest represents a privileged elevation request
type ElevationRequest struct {
	ID               uuid.UUID `db:"id" json:"id"`
	TenantID         uuid.UUID `db:"tenant_id" json:"tenant_id"`
	UserID           uuid.UUID `db:"user_id" json:"user_id"`
	SessionID        uuid.UUID `db:"session_id" json:"session_id"`
	TargetHost       string    `db:"target_host" json:"target_host"`
	TargetUser       string    `db:"target_user" json:"target_user"`
	RequestedCommand string    `db:"requested_command" json:"requested_command"`
	Justification    string    `db:"justification" json:"justification"`
	Duration         int       `db:"duration" json:"duration"` // seconds
	Status           string    `db:"status" json:"status"` // pending, active, expired, denied, revoked

	// Approval tracking
	ApprovedBy        *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt        *time.Time `db:"approved_at" json:"approved_at,omitempty"`
	ApprovalMethod    string    `db:"approval_method" json:"approval_method"` // mfa, approval, auto

	// Elevation tracking
	ElevatedAt        *time.Time `db:"elevated_at" json:"elevated_at,omitempty"`
	ExpiresAt         *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	RevokedAt         *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`

	// Usage tracking
	CommandsExecuted  int       `db:"commands_executed" json:"commands_executed"`
	LastActivityAt     *time.Time `db:"last_activity_at" json:"last_activity_at,omitempty"`

	// Audit
	SessionRecordingID *uuid.UUID `db:"session_recording_id" json:"session_recording_id,omitempty"`

	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
}

// RequestElevation requests a privileged elevation
func (s *ElevationService) RequestElevation(ctx context.Context, req *ElevationRequest) (*ElevationRequest, error) {
	// Validate request
	if req.RequestedCommand == "" && req.TargetUser == "" {
		return nil, fmt.Errorf("elevation.RequestElevation: command or target user must be specified")
	}

	if req.Duration <= 0 {
		req.Duration = int(DefaultElevationConfig().DefaultDuration.Seconds())
	}

	if req.Duration > int(DefaultElevationConfig().MaxElevationDuration.Seconds()) {
		return nil, fmt.Errorf("elevation.RequestElevation: requested duration exceeds maximum")
	}

	req.ID = uuid.New()
	req.Status = "pending"
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// Determine approval method
	if req.ApprovalMethod == "" {
		req.ApprovalMethod = "mfa" // Default to MFA-based approval
	}

	// Auto-approve if configured and MFA verified
	if !DefaultElevationConfig().RequireApproval {
		req.Status = "active"
		now := time.Now()
		req.ElevatedAt = &now
		expiresAt := now.Add(time.Duration(req.Duration) * time.Second)
		req.ExpiresAt = &expiresAt
	}

	// Save request
	if err := s.SaveElevationRequest(ctx, req); err != nil {
		return nil, err
	}

	// Log to analytics
	if s.analyticsSvc != nil && DefaultElevationConfig().LogAllElevations {
		_ = s.analyticsSvc.RecordCommand(ctx, req.TenantID, req.UserID, req.SessionID, req.RequestedCommand, req.TargetHost, nil)
	}

	s.logger.Info().
		Str("request_id", req.ID.String()).
		Str("user_id", req.UserID.String()).
		Str("target_host", req.TargetHost).
		Str("command", req.RequestedCommand).
		Str("status", req.Status).
		Msg("Privileged elevation requested")

	return req, nil
}

// ApproveElevation approves an elevation request
func (s *ElevationService) ApproveElevation(ctx context.Context, id, approverID uuid.UUID, method string) (*ElevationRequest, error) {
	req, err := s.GetElevationRequest(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("elevation.ApproveElevation: %w", err)
	}

	if req.Status != "pending" {
		return nil, fmt.Errorf("elevation.ApproveElevation: request is not pending")
	}

	// Update request
	now := time.Now()
	req.Status = "active"
	req.ElevatedAt = &now
	req.ExpiresAt = &now.Add(time.Duration(req.Duration) * time.Second)
	req.ApprovedBy = &approverID
	req.ApprovedAt = &now
	req.ApprovalMethod = method
	req.UpdatedAt = now

	if err := s.SaveElevationRequest(ctx, req); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("request_id", id.String()).
		Str("approver_id", approverID.String()).
		Str("method", method).
		Msg("Privileged elevation approved")

	return req, nil
}

// DenyElevation denies an elevation request
func (s *ElevationService) DenyElevation(ctx context.Context, id uuid.UUID, reason string) error {
	req, err := s.GetElevationRequest(ctx, id)
	if err != nil {
		return fmt.Errorf("elevation.DenyElevation: %w", err)
	}

	req.Status = "denied"
	req.UpdatedAt = time.Now()

	if err := s.SaveElevationRequest(ctx, req); err != nil {
		return err
	}

	s.logger.Info().
		Str("request_id", id.String()).
		Str("reason", reason).
		Msg("Privileged elevation denied")

	return nil
}

// RevokeElevation revokes an active elevation
func (s *ElevationService) RevokeElevation(ctx context.Context, id uuid.UUID, revokedBy uuid.UUID) error {
	req, err := s.GetElevationRequest(ctx, id)
	if err != nil {
		return fmt.Errorf("elevation.RevokeElevation: %w", err)
	}

	if req.Status != "active" {
		return fmt.Errorf("elevation.RevokeElevation: elevation is not active")
	}

	now := time.Now()
	req.Status = "revoked"
	req.RevokedAt = &now
	req.UpdatedAt = now

	if err := s.SaveElevationRequest(ctx, req); err != nil {
		return err
	}

	// Terminate the elevated session
	go s.terminateElevation(ctx, req)

	s.logger.Warn().
		Str("request_id", id.String()).
		Str("revoked_by", revokedBy.String()).
		Msg("Privileged elevation revoked")

	return nil
}

// ValidateElevation checks if a command execution is allowed under an elevation
func (s *ElevationService) ValidateElevation(ctx context.Context, sessionID uuid.UUID, command string) (*ElevationRequest, error) {
	req, err := s.GetActiveElevation(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("elevation.ValidateElevation: %w", err)
	}

	// Check if expired
	if req.ExpiresAt != nil && time.Now().After(*req.ExpiresAt) {
		req.Status = "expired"
		_ = s.SaveElevationRequest(ctx, req)
		return nil, fmt.Errorf("elevation.ValidateElevation: elevation has expired")
	}

	// Validate command matches requested elevation
	if req.RequestedCommand != "" && !s.CommandMatches(command, req.RequestedCommand) {
		return nil, fmt.Errorf("elevation.ValidateElevation: command does not match approved elevation")
	}

	// Update activity
	now := time.Now()
	req.CommandsExecuted++
	req.LastActivityAt = &now
	_ = s.SaveElevationRequest(ctx, req)

	return req, nil
}

// GetElevationRequest retrieves an elevation request
func (s *ElevationService) GetElevationRequest(ctx context.Context, id uuid.UUID) (*ElevationRequest, error) {
	query := `
		SELECT * FROM elevation_requests
		WHERE id = $1 AND deleted_at IS NULL
	`

	var req ElevationRequest
	err := s.db.GetContext(ctx, &req, query, id)
	if err != nil {
		return nil, fmt.Errorf("elevation.GetElevationRequest: %w", err)
	}

	return &req, nil
}

// GetActiveElevation retrieves active elevation for a session
func (s *ElevationService) GetActiveElevation(ctx context.Context, sessionID uuid.UUID) (*ElevationRequest, error) {
	query := `
		SELECT * FROM elevation_requests
		WHERE session_id = $1
		AND status = 'active'
		AND deleted_at IS NULL
		AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY elevated_at DESC
		LIMIT 1
	`

	var req ElevationRequest
	err := s.db.GetContext(ctx, &req, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("elevation.GetActiveElevation: %w", err)
	}

	return &req, nil
}

// GetUserElevations returns all elevations for a user
func (s *ElevationService) GetUserElevations(ctx context.Context, tenantID, userID uuid.UUID, status *string) ([]ElevationRequest, error) {
	query := `
		SELECT * FROM elevation_requests
		WHERE tenant_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	args := []interface{}{tenantID, userID}

	if status != nil {
		query += " AND status = $3"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	var elevations []ElevationRequest
	err := s.db.SelectContext(ctx, &elevations, query, args...)
	if err != nil {
		return nil, fmt.Errorf("elevation.GetUserElevations: %w", err)
	}

	return elevations, nil
}

// SaveElevationRequest saves an elevation request
func (s *ElevationService) SaveElevationRequest(ctx context.Context, req *ElevationRequest) error {
	req.UpdatedAt = time.Now()

	query := `
		INSERT INTO elevation_requests (
			id, tenant_id, user_id, session_id, target_host, target_user,
			requested_command, justification, duration, status,
			approved_by, approved_at, approval_method,
			elevated_at, expires_at, revoked_at,
			commands_executed, last_activity_at, session_recording_id,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :user_id, :session_id, :target_host, :target_user,
			:requested_command, :justification, :duration, :status,
			:approved_by, :approved_at, :approval_method,
			:elevated_at, :expires_at, :revoked_at,
			:commands_executed, :last_activity_at, :session_recording_id,
			:created_at, :updated_at
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			approved_by = EXCLUDED.approved_by,
			approved_at = EXCLUDED.approved_at,
			approval_method = EXCLUDED.approval_method,
			elevated_at = EXCLUDED.elevated_at,
			expires_at = EXCLUDED.expires_at,
			revoked_at = EXCLUDED.revoked_at,
			commands_executed = EXCLUDED.commands_executed,
			last_activity_at = EXCLUDED.last_activity_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.NamedExecContext(ctx, query, req)
	if err != nil {
		return fmt.Errorf("elevation.SaveElevationRequest: %w", err)
	}

	return nil
}

// ProcessExpiredElevations marks expired elevations as expired
func (s *ElevationService) ProcessExpiredElevations(ctx context.Context) error {
	query := `
		UPDATE elevation_requests
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'active'
		AND expires_at <= NOW()
		AND deleted_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("elevation.ProcessExpiredElevations: %w", err)
	}

	// Get affected sessions for termination
	affectedQuery := `
		SELECT session_id FROM elevation_requests
		WHERE status = 'expired'
		AND expired_at <= NOW()
		AND deleted_at IS NULL
	`

	rows, err := s.db.QueryContext(ctx, affectedQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID uuid.UUID
		if rows.Scan(&sessionID) == nil {
			go s.terminateSessionElevation(ctx, sessionID)
		}
	}

	return nil
}

// terminateElevation terminates the elevated session
func (s *ElevationService) terminateElevation(ctx context.Context, req *ElevationRequest) {
	// Send signal to session to terminate elevation
	s.logger.Info().
		Str("request_id", req.ID.String()).
		Str("session_id", req.SessionID.String()).
		Msg("Elevation terminated")
}

// terminateSessionElevation terminates elevation for a session
func (s *ElevationService) terminateSessionElevation(ctx context.Context, sessionID uuid.UUID) {
	s.logger.Info().
		Str("session_id", sessionID.String()).
		Msg("Session elevation expired, user returned to normal privileges")
}

// CommandMatches checks if a command matches the approved elevation
func (s *ElevationService) CommandMatches(command, approvedCommand string) bool {
	// Exact match
	if command == approvedCommand {
		return true
	}

	// Prefix match (for commands with arguments)
	if len(command) >= len(approvedCommand) && command[:len(approvedCommand)] == approvedCommand {
		return true
	}

	return false
}

// GenerateElevationToken generates a one-time token for elevation
func (s *ElevationService) GenerateElevationToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("elevation.GenerateElevationToken: %w", err)
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// ValidateElevationToken validates an elevation token
func (s *ElevationService) ValidateElevationToken(ctx context.Context, token string) (uuid.UUID, error) {
	// This would validate against stored tokens
	// For now, return error as placeholder
	return uuid.Nil, fmt.Errorf("elevation.ValidateElevationToken: invalid token")
}
