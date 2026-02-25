package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Approval represents an approval action on an access request
type Approval struct {
	ID               uuid.UUID          `json:"id" db:"id"`
	TenantID         uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	RequestID        uuid.UUID          `json:"request_id" db:"request_id"`
	WorkflowID       uuid.UUID          `json:"workflow_id" db:"workflow_id"`
	StepID           uuid.UUID          `json:"step_id" db:"step_id"`
	ApproverID       uuid.UUID          `json:"approver_id" db:"approver_id"`
	Decision         string             `json:"decision" db:"decision"` // approve, deny, comment
	Comments         string             `json:"comments" db:"comments"`
	IPAddress        string             `json:"ip_address" db:"ip_address"`
	UserAgent        string             `json:"user_agent" db:"user_agent"`
	MFAVerified      bool               `json:"mfa_verified" db:"mfa_verified"`
	BreakGlassOverride bool            `json:"break_glass_override" db:"break_glass_override"`
	Justification    string             `json:"justification" db:"justification"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
}

// ApprovalStatus represents the current status of approvals for a request
type ApprovalStatus struct {
	RequestID          uuid.UUID `json:"request_id" db:"request_id"`
	CurrentStepOrder   int       `json:"current_step_order" db:"current_step_order"`
	TotalSteps         int       `json:"total_steps" db:"total_steps"`
	PendingApprovals   int       `json:"pending_approvals" db:"pending_approvals"`
	CompletedApprovals int       `json:"completed_approvals" db:"completed_approvals"`
	ApprovedCount      int       `json:"approved_count" db:"approved_count"`
	DeniedCount        int       `json:"denied_count" db:"denied_count"`
	LastApprovalAt     *time.Time `json:"last_approval_at" db:"last_approval_at"`
}

// ApprovalRepository handles database operations for approvals
type ApprovalRepository struct {
	db     *sqlx.DB
	logger *zerolog.Logger
}

// NewApprovalRepository creates a new approval repository
func NewApprovalRepository(db *sqlx.DB, logger *zerolog.Logger) *ApprovalRepository {
	return &ApprovalRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new approval record
func (r *ApprovalRepository) Create(ctx context.Context, approval *Approval) error {
	approval.ID = uuid.New()
	approval.CreatedAt = time.Now()

	query := `
		INSERT INTO approvals (
			id, tenant_id, request_id, workflow_id, step_id,
			approver_id, decision, comments, ip_address, user_agent,
			mfa_verified, break_glass_override, justification, created_at
		) VALUES (
			:id, :tenant_id, :request_id, :workflow_id, :step_id,
			:approver_id, :decision, :comments, :ip_address, :user_agent,
			:mfa_verified, :break_glass_override, :justification, :created_at
		) RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, approval)
	if err != nil {
		return fmt.Errorf("approval.Create: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&approval.ID); err != nil {
			return fmt.Errorf("approval.Create scan: %w", err)
		}
	}

	return nil
}

// GetByRequestID retrieves all approvals for a request
func (r *ApprovalRepository) GetByRequestID(ctx context.Context, requestID uuid.UUID) ([]Approval, error) {
	query := `
		SELECT id, tenant_id, request_id, workflow_id, step_id,
			approver_id, decision, comments, ip_address, user_agent,
			mfa_verified, break_glass_override, justification, created_at
		FROM approvals
		WHERE request_id = $1
		ORDER BY created_at ASC
	`

	var approvals []Approval
	err := r.db.SelectContext(ctx, &approvals, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("approval.GetByRequestID: %w", err)
	}

	return approvals, nil
}

// GetByApproverID retrieves approvals by approver ID
func (r *ApprovalRepository) GetByApproverID(ctx context.Context, approverID uuid.UUID, limit int) ([]Approval, error) {
	query := `
		SELECT id, tenant_id, request_id, workflow_id, step_id,
			approver_id, decision, comments, ip_address, user_agent,
			mfa_verified, break_glass_override, justification, created_at
		FROM approvals
		WHERE approver_id = $1
		ORDER BY created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	} else {
		query += " LIMIT 100"
	}

	var approvals []Approval
	err := r.db.SelectContext(ctx, &approvals, query, approverID)
	if err != nil {
		return nil, fmt.Errorf("approval.GetByApproverID: %w", err)
	}

	return approvals, nil
}

// GetStatus retrieves the current approval status for a request
func (r *ApprovalRepository) GetStatus(ctx context.Context, requestID uuid.UUID) (*ApprovalStatus, error) {
	query := `
		SELECT
			ar.id as request_id,
			COALESCE(ws.order, 0) as current_step_order,
			COALESCE((SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = ar.workflow_id), 0) as total_steps,
			COALESCE((SELECT COUNT(*) FROM workflow_approvers wa
				INNER JOIN workflow_steps ws2 ON wa.step_id = ws2.id
				WHERE ws2.workflow_id = ar.workflow_id AND wa.approved_at IS NULL), 0) as pending_approvals,
			COALESCE((SELECT COUNT(*) FROM workflow_approvers wa
				INNER JOIN workflow_steps ws2 ON wa.step_id = ws2.id
				WHERE ws2.workflow_id = ar.workflow_id AND wa.approved_at IS NOT NULL), 0) as completed_approvals,
			COALESCE((SELECT COUNT(*) FROM approvals a WHERE a.request_id = ar.id AND a.decision = 'approve'), 0) as approved_count,
			COALESCE((SELECT COUNT(*) FROM approvals a WHERE a.request_id = ar.id AND a.decision = 'deny'), 0) as denied_count,
			(SELECT MAX(created_at) FROM approvals a WHERE a.request_id = ar.id) as last_approval_at
		FROM access_requests ar
		LEFT JOIN workflow_steps ws ON ws.id = (
			SELECT wa2.step_id FROM workflow_approvers wa2
			INNER JOIN workflow_steps ws3 ON wa2.step_id = ws3.id
			WHERE ws3.workflow_id = ar.workflow_id AND wa2.approved_at IS NULL
			ORDER BY ws3.order ASC
			LIMIT 1
		)
		WHERE ar.id = $1
	`

	var status ApprovalStatus
	err := r.db.GetContext(ctx, &status, query, requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("approval status not found")
		}
		return nil, fmt.Errorf("approval.GetStatus: %w", err)
	}

	return &status, nil
}

// HasUserApproved checks if a user has already approved a specific request step
func (r *ApprovalRepository) HasUserApproved(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, stepID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM approvals
			WHERE request_id = $1 AND approver_id = $2 AND step_id = $3 AND decision = 'approve'
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, requestID, approverID, stepID)
	if err != nil {
		return false, fmt.Errorf("approval.HasUserApproved: %w", err)
	}

	return exists, nil
}

// GetPendingApprovalsForUser retrieves pending approvals for a specific user
func (r *ApprovalRepository) GetPendingApprovalsForUser(ctx context.Context, approverID uuid.UUID, tenantID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT ar.id
		FROM access_requests ar
		INNER JOIN workflow_steps ws ON ar.workflow_id = ws.workflow_id
		INNER JOIN workflow_approvers wa ON ws.id = wa.step_id
		WHERE ar.status = 'pending'
			AND ar.tenant_id = $1
			AND wa.approver_id = $2
			AND wa.approved_at IS NULL
			AND ar.deleted_at IS NULL
		ORDER BY ar.created_at ASC
	`

	var requestIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &requestIDs, query, tenantID, approverID)
	if err != nil {
		return nil, fmt.Errorf("approval.GetPendingApprovalsForUser: %w", err)
	}

	return requestIDs, nil
}

// GetCurrentStepForRequest retrieves the current pending step for a request
func (r *ApprovalRepository) GetCurrentStepForRequest(ctx context.Context, requestID uuid.UUID) (*WorkflowStep, error) {
	query := `
		SELECT ws.id, ws.workflow_id, ws.order, ws.name, ws.description, ws.type,
			ws.approval_type, ws.timeout_minutes, ws.auto_approve, ws.escalation_step_id,
			ws.created_at, ws.updated_at
		FROM workflow_steps ws
		INNER JOIN access_requests ar ON ar.workflow_id = ws.workflow_id
		WHERE ar.id = $1
		AND ws.id = (
			SELECT wa.step_id FROM workflow_approvers wa
			WHERE wa.step_id = ws.id AND wa.approved_at IS NULL
			ORDER BY ws.order ASC
			LIMIT 1
		)
		LIMIT 1
	`

	var step WorkflowStep
	err := r.db.GetContext(ctx, &step, query, requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No pending step
		}
		return nil, fmt.Errorf("approval.GetCurrentStepForRequest: %w", err)
	}

	return &step, nil
}

// GetApprovalsForStep retrieves approvals for a specific step
func (r *ApprovalRepository) GetApprovalsForStep(ctx context.Context, stepID uuid.UUID) ([]Approval, error) {
	query := `
		SELECT id, tenant_id, request_id, workflow_id, step_id,
			approver_id, decision, comments, ip_address, user_agent,
			mfa_verified, break_glass_override, justification, created_at
		FROM approvals
		WHERE step_id = $1
		ORDER BY created_at ASC
	`

	var approvals []Approval
	err := r.db.SelectContext(ctx, &approvals, query, stepID)
	if err != nil {
		return nil, fmt.Errorf("approval.GetApprovalsForStep: %w", err)
	}

	return approvals, nil
}
