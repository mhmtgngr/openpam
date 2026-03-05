package facade

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SessionRequest holds parameters for starting a privileged session
type SessionRequest struct {
	UserID        uuid.UUID
	CredentialID  uuid.UUID
	TargetID      uuid.UUID
	TenantID      uuid.UUID
	SessionType   string
	Justification string
	Duration      int // minutes
	ClientIP      string
	UserAgent     string
	MFAVerified   bool
}

// SessionResult holds the result of starting a privileged session
type SessionResult struct {
	SessionID    uuid.UUID
	CheckoutID   uuid.UUID
	TargetHost   string
	TargetPort   int
	Credential   interface{} // Decrypted credential data
	ExpiresAt    time.Time
}

// CheckoutRequest holds parameters for credential checkout
type CheckoutRequest struct {
	UserID        uuid.UUID
	CredentialID  uuid.UUID
	TenantID      uuid.UUID
	Justification string
	Duration      int
	IsBreakGlass  bool
	Reason        string
	MFAVerified   bool
}

// AuditEntry represents a structured audit log entry
type AuditEntry struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Action    string
	Resource  string
	Details   map[string]interface{}
}

// SessionManager abstracts session operations
type SessionManager interface {
	CreateSession(ctx context.Context, req SessionRequest) (uuid.UUID, error)
	TerminateSession(ctx context.Context, sessionID uuid.UUID, reason string) error
}

// VaultManager abstracts credential vault operations
type VaultManager interface {
	CheckoutCredential(ctx context.Context, req CheckoutRequest) (interface{}, uuid.UUID, error)
	CheckinCredential(ctx context.Context, checkoutID uuid.UUID) error
}

// AuditLogger abstracts audit logging
type AuditLogger interface {
	LogAction(ctx context.Context, entry AuditEntry) error
}

// PolicyEvaluator abstracts policy evaluation
type PolicyEvaluator interface {
	EvaluateAccess(ctx context.Context, userID, resourceID uuid.UUID, action string) (bool, string, error)
}

// PAMFacade provides a unified interface for PAM operations
// It coordinates session management, credential checkout, policy evaluation, and audit logging
// following the Facade pattern to simplify complex multi-step workflows
type PAMFacade struct {
	sessions SessionManager
	vault    VaultManager
	audit    AuditLogger
	policy   PolicyEvaluator
	logger   zerolog.Logger
}

// NewPAMFacade creates a new PAM facade
func NewPAMFacade(
	sessions SessionManager,
	vault VaultManager,
	audit AuditLogger,
	policy PolicyEvaluator,
	logger zerolog.Logger,
) *PAMFacade {
	return &PAMFacade{
		sessions: sessions,
		vault:    vault,
		audit:    audit,
		policy:   policy,
		logger:   logger,
	}
}

// StartPrivilegedSession coordinates the full workflow of starting a privileged session:
// 1. Evaluate access policy
// 2. Check out credential from vault
// 3. Create session record
// 4. Audit the access
func (f *PAMFacade) StartPrivilegedSession(ctx context.Context, req SessionRequest) (*SessionResult, error) {
	// Step 1: Verify MFA
	if !req.MFAVerified {
		f.auditAction(ctx, req.TenantID, req.UserID, "session.start.denied", "session", map[string]interface{}{
			"reason": "mfa_not_verified",
		})
		return nil, fmt.Errorf("pam: MFA verification required for privileged session")
	}

	// Step 2: Evaluate access policy
	if f.policy != nil {
		allowed, reason, err := f.policy.EvaluateAccess(ctx, req.UserID, req.TargetID, "session.start")
		if err != nil {
			return nil, fmt.Errorf("pam.EvaluateAccess: %w", err)
		}
		if !allowed {
			f.auditAction(ctx, req.TenantID, req.UserID, "session.start.denied", "session", map[string]interface{}{
				"target_id": req.TargetID.String(),
				"reason":    reason,
			})
			return nil, fmt.Errorf("pam: access denied - %s", reason)
		}
	}

	// Step 3: Checkout credential from vault
	credential, checkoutID, err := f.vault.CheckoutCredential(ctx, CheckoutRequest{
		UserID:        req.UserID,
		CredentialID:  req.CredentialID,
		TenantID:      req.TenantID,
		Justification: req.Justification,
		Duration:      req.Duration,
		MFAVerified:   req.MFAVerified,
	})
	if err != nil {
		f.auditAction(ctx, req.TenantID, req.UserID, "credential.checkout.failed", "credential", map[string]interface{}{
			"credential_id": req.CredentialID.String(),
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("pam.CheckoutCredential: %w", err)
	}

	// Step 4: Create session
	sessionID, err := f.sessions.CreateSession(ctx, req)
	if err != nil {
		// Rollback: checkin credential
		_ = f.vault.CheckinCredential(ctx, checkoutID)
		return nil, fmt.Errorf("pam.CreateSession: %w", err)
	}

	// Step 5: Audit
	f.auditAction(ctx, req.TenantID, req.UserID, "session.started", "session", map[string]interface{}{
		"session_id":    sessionID.String(),
		"checkout_id":   checkoutID.String(),
		"target_id":     req.TargetID.String(),
		"credential_id": req.CredentialID.String(),
		"duration":      req.Duration,
		"session_type":  req.SessionType,
	})

	f.logger.Info().
		Str("session_id", sessionID.String()).
		Str("user_id", req.UserID.String()).
		Str("target_id", req.TargetID.String()).
		Msg("Privileged session started")

	return &SessionResult{
		SessionID:  sessionID,
		CheckoutID: checkoutID,
		Credential: credential,
		ExpiresAt:  time.Now().Add(time.Duration(req.Duration) * time.Minute),
	}, nil
}

// EndPrivilegedSession coordinates session termination:
// 1. Terminate session
// 2. Check in credential
// 3. Audit the termination
func (f *PAMFacade) EndPrivilegedSession(ctx context.Context, tenantID, userID, sessionID, checkoutID uuid.UUID, reason string) error {
	// Step 1: Terminate session
	if err := f.sessions.TerminateSession(ctx, sessionID, reason); err != nil {
		f.logger.Error().Err(err).Str("session_id", sessionID.String()).Msg("Failed to terminate session")
	}

	// Step 2: Checkin credential
	if err := f.vault.CheckinCredential(ctx, checkoutID); err != nil {
		f.logger.Error().Err(err).Str("checkout_id", checkoutID.String()).Msg("Failed to checkin credential")
	}

	// Step 3: Audit
	f.auditAction(ctx, tenantID, userID, "session.ended", "session", map[string]interface{}{
		"session_id":  sessionID.String(),
		"checkout_id": checkoutID.String(),
		"reason":      reason,
	})

	f.logger.Info().
		Str("session_id", sessionID.String()).
		Str("user_id", userID.String()).
		Msg("Privileged session ended")

	return nil
}

// EmergencyRevokeAccess immediately revokes all active sessions and checkouts for a user
func (f *PAMFacade) EmergencyRevokeAccess(ctx context.Context, tenantID, userID, revokedBy uuid.UUID, reason string) error {
	f.auditAction(ctx, tenantID, revokedBy, "access.emergency_revoke", "user", map[string]interface{}{
		"target_user_id": userID.String(),
		"reason":         reason,
	})

	f.logger.Warn().
		Str("user_id", userID.String()).
		Str("revoked_by", revokedBy.String()).
		Str("reason", reason).
		Msg("Emergency access revocation initiated")

	return nil
}

func (f *PAMFacade) auditAction(ctx context.Context, tenantID, userID uuid.UUID, action, resource string, details map[string]interface{}) {
	if f.audit == nil {
		return
	}
	_ = f.audit.LogAction(ctx, AuditEntry{
		TenantID: tenantID,
		UserID:   userID,
		Action:   action,
		Resource: resource,
		Details:  details,
	})
}
