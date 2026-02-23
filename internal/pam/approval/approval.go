package approval

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// RequestStatus represents the status of an approval request
type RequestStatus string

const (
	StatusPending    RequestStatus = "pending"
	StatusApproved   RequestStatus = "approved"
	StatusDenied     RequestStatus = "denied"
	StatusCancelled  RequestStatus = "cancelled"
	StatusExpired    RequestStatus = "expired"
	StatusEscalated  RequestStatus = "escalated"
)

// RequestType represents the type of request
type RequestType string

const (
	RequestTypeCredentialAccess RequestType = "credential_access"
	RequestTypeSessionAccess    RequestType = "session_access"
	RequestTypePrivilegeEscalation RequestType = "privilege_escalation"
)

// Request represents an access request
type Request struct {
	ID              uuid.UUID      `db:"id" json:"id"`
	Type            RequestType    `db:"type" json:"type"`
	TenantID        uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	UserID          uuid.UUID      `db:"user_id" json:"user_id"`
	TargetID        *uuid.UUID     `db:"target_id" json:"target_id,omitempty"`
	CredentialID    *uuid.UUID     `db:"credential_id" json:"credential_id,omitempty"`
	Justification   string         `db:"justification" json:"justification"`
	Duration        int            `db:"duration_minutes" json:"duration_minutes"`
	Status          RequestStatus  `db:"status" json:"status"`
	Priority        string         `db:"priority" json:"priority"` // low, normal, high, emergency

	// Approval configuration
	ApprovalGroupID *uuid.UUID     `db:"approval_group_id" json:"approval_group_id,omitempty"`
	RequiredApprovals int          `db:"required_approvals" json:"required_approvals"`
	ReceivedApprovals int          `db:"received_approvals" json:"received_approvals"`

	// Timestamps
	ExpiresAt       *time.Time     `db:"expires_at" json:"expires_at,omitempty"`
	ApprovedAt      *time.Time     `db:"approved_at" json:"approved_at,omitempty"`
	DeniedAt        *time.Time     `db:"denied_at" json:"denied_at,omitempty"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`

	// Metadata
	TicketRef       string         `db:"ticket_ref" json:"ticket_ref,omitempty"` // External ticket reference
	Metadata        map[string]interface{} `db:"metadata" json:"metadata,omitempty"`
}

// Approval represents an approval decision
type Approval struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	RequestID    uuid.UUID    `db:"request_id" json:"request_id"`
	ApproverID   uuid.UUID    `db:"approver_id" json:"approver_id"`
	Decision     string       `db:"decision" json:"decision"` // approve, deny
	Comments     string       `db:"comments" json:"comments"`
	DelegatedFrom *uuid.UUID  `db:"delegated_from" json:"delegated_from,omitempty"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
}

// ApprovalChain represents a sequence of required approvals
type ApprovalChain struct {
	ID          uuid.UUID   `db:"id" json:"id"`
	TenantID    uuid.UUID   `db:"tenant_id" json:"tenant_id"`
	Name        string      `db:"name" json:"name"`
	Levels      []ChainLevel `db:"levels" json:"levels"`
	IsActive    bool        `db:"is_active" json:"is_active"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
}

// ChainLevel represents one level in an approval chain
type ChainLevel struct {
	Order          int      `json:"order"`
	ApprovalGroupID uuid.UUID `json:"approval_group_id"`
	RequiredCount  int      `json:"required_count"`
	TimeoutMinutes int      `json:"timeout_minutes"`
	EscalateTo     *uuid.UUID `json:"escalate_to,omitempty"`
}

// ApprovalPolicy defines auto-approval rules
type ApprovalPolicy struct {
	ID              uuid.UUID   `db:"id" json:"id"`
	TenantID        uuid.UUID   `db:"tenant_id" json:"tenant_id"`
	Name            string      `db:"name" json:"name"`
	Description     string      `db:"description" json:"description"`
	Rules           []PolicyRule `json:"rules"`
	AutoApprove     bool        `db:"auto_approve" json:"auto_approve"`
	IsActive        bool        `db:"is_active" json:"is_active"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
}

// PolicyRule defines a single policy rule
type PolicyRule struct {
	Field    string      `json:"field"` // user_role, target_sensitivity, time_range, etc.
	Operator string      `json:"operator"` // equals, contains, in_range, etc.
	Value    interface{} `json:"value"`
}

// Repository handles approval data operations
type Repository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewRepository creates a new approval repository
func NewRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Repository {
	return &Repository{db: db, cache: c, logger: logger}
}

// CreateRequest creates a new approval request
func (r *Repository) CreateRequest(ctx context.Context, req *Request) error {
	req.ID = uuid.New()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.Status = StatusPending

	query := `
		INSERT INTO approval_requests (id, type, tenant_id, user_id, target_id, credential_id,
			justification, duration_minutes, status, priority, approval_group_id,
			required_approvals, received_approvals, expires_at, ticket_ref, metadata, created_at, updated_at)
		VALUES (:id, :type, :tenant_id, :user_id, :target_id, :credential_id,
			:justification, :duration_minutes, :status, :priority, :approval_group_id,
			:required_approvals, :received_approvals, :expires_at, :ticket_ref, :metadata, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, req)
	if err != nil {
		return fmt.Errorf("approval.CreateRequest: %w", err)
	}

	return nil
}

// GetRequest retrieves a request by ID
func (r *Repository) GetRequest(ctx context.Context, id uuid.UUID) (*Request, error) {
	var req Request
	query := `SELECT * FROM approval_requests WHERE id = $1`
	err := r.db.GetContext(ctx, &req, query, id)
	if err != nil {
		return nil, fmt.Errorf("approval.GetRequest: %w", err)
	}
	return &req, nil
}

// ListRequests retrieves requests with filtering
func (r *Repository) ListRequests(ctx context.Context, tenantID uuid.UUID, filter RequestFilter, limit, offset int) ([]Request, int, error) {
	baseQuery := `
		SELECT * FROM approval_requests
		WHERE tenant_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM approval_requests WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argCount := 2

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}
	if filter.ApproverID != nil {
		// Join with approvals table
		baseQuery += fmt.Sprintf(" AND id IN (SELECT request_id FROM approvals WHERE approver_id = $%d)", argCount)
		args = append(args, *filter.ApproverID)
		argCount++
	}

	// Get count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("approval.ListRequests.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var requests []Request
	if err := r.db.SelectContext(ctx, &requests, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("approval.ListRequests: %w", err)
	}

	return requests, total, nil
}

// UpdateRequestStatus updates a request's status
func (r *Repository) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status RequestStatus, approvedAt, deniedAt *time.Time) error {
	query := `
		UPDATE approval_requests
		SET status = $2, approved_at = $3, denied_at = $4, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, approvedAt, deniedAt)
	if err != nil {
		return fmt.Errorf("approval.UpdateRequestStatus: %w", err)
	}
	return nil
}

// RecordApproval records an approval decision
func (r *Repository) RecordApproval(ctx context.Context, approval *Approval) error {
	approval.ID = uuid.New()
	approval.CreatedAt = time.Now()

	query := `
		INSERT INTO approvals (id, request_id, approver_id, decision, comments, delegated_from, created_at)
		VALUES (:id, :request_id, :approver_id, :decision, :comments, :delegated_from, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, approval)
	if err != nil {
		return fmt.Errorf("approval.RecordApproval: %w", err)
	}

	// Update received approvals count
	updateQuery := `
		UPDATE approval_requests
		SET received_approvals = received_approvals + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err = r.db.ExecContext(ctx, updateQuery, approval.RequestID)
	if err != nil {
		return fmt.Errorf("approval.UpdateReceivedApprovals: %w", err)
	}

	return nil
}

// GetApprovalsForRequest retrieves all approvals for a request
func (r *Repository) GetApprovalsForRequest(ctx context.Context, requestID uuid.UUID) ([]Approval, error) {
	var approvals []Approval
	query := `SELECT * FROM approvals WHERE request_id = $1 ORDER BY created_at ASC`
	err := r.db.SelectContext(ctx, &approvals, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("approval.GetApprovalsForRequest: %w", err)
	}
	return approvals, nil
}

// CheckIfApproved checks if a user has already approved a request
func (r *Repository) CheckIfApproved(ctx context.Context, requestID, approverID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM approvals WHERE request_id = $1 AND approver_id = $2 AND decision = 'approve'`
	err := r.db.GetContext(ctx, &count, query, requestID, approverID)
	if err != nil {
		return false, fmt.Errorf("approval.CheckIfApproved: %w", err)
	}
	return count > 0, nil
}

// RequestFilter filters request queries
type RequestFilter struct {
	Status     *RequestStatus
	Type       *RequestType
	UserID     *uuid.UUID
	ApproverID *uuid.UUID
}

// WorkflowService handles approval workflow logic
type WorkflowService struct {
	repo       *Repository
	publisher  *events.Publisher
	cache      *cache.Cache
	logger     zerolog.Logger
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(repo *Repository, publisher *events.Publisher, c *cache.Cache, logger zerolog.Logger) *WorkflowService {
	return &WorkflowService{
		repo:      repo,
		publisher: publisher,
		cache:     c,
		logger:    logger,
	}
}

// CreateRequest creates and routes a new approval request
func (s *WorkflowService) CreateRequest(ctx context.Context, req *Request) (*Request, error) {
	// Check auto-approve policies
	if s.shouldAutoApprove(ctx, req) {
		req.Status = StatusApproved
		now := time.Now()
		req.ApprovedAt = &now
		req.ReceivedApprovals = req.RequiredApprovals
	} else {
		// Route to appropriate approvers
		if err := s.RouteRequest(ctx, req); err != nil {
			return nil, fmt.Errorf("workflow.RouteRequest: %w", err)
		}
	}

	if err := s.repo.CreateRequest(ctx, req); err != nil {
		return nil, err
	}

	// Publish event
	if req.Type == RequestTypeCredentialAccess {
		_ = s.publisher.PublishCheckoutRequested(ctx, req.TenantID.String(), req.UserID.String(),
			req.CredentialID.String(), req.ID.String(), req.Duration)
	}

	return req, nil
}

// RouteRequest determines the approval routing for a request
func (s *WorkflowService) RouteRequest(ctx context.Context, req *Request) error {
	// If approval group is already set, use it
	if req.ApprovalGroupID != nil {
		// Get required approvals from group
		required, err := s.getRequiredApprovalsForGroup(ctx, *req.ApprovalGroupID)
		if err != nil {
			return err
		}
		req.RequiredApprovals = required
		return nil
	}

	// Route based on target/credential sensitivity
	if req.CredentialID != nil {
		groupID, required, err := s.getApprovalGroupForCredential(ctx, *req.CredentialID)
		if err != nil {
			return err
		}
		req.ApprovalGroupID = groupID
		req.RequiredApprovals = required
	} else if req.TargetID != nil {
		groupID, required, err := s.getApprovalGroupForTarget(ctx, *req.TargetID)
		if err != nil {
			return err
		}
		req.ApprovalGroupID = groupID
		req.RequiredApprovals = required
	}

	// Default: require 1 approval
	if req.RequiredApprovals == 0 {
		req.RequiredApprovals = 1
	}

	return nil
}

// ApproveRequest approves a request
func (s *WorkflowService) ApproveRequest(ctx context.Context, requestID, approverID uuid.UUID, comments string) error {
	// Get request
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if req.Status != StatusPending {
		return fmt.Errorf("workflow: request is not pending (status: %s)", req.Status)
	}

	// Check if already approved
	approved, err := s.repo.CheckIfApproved(ctx, requestID, approverID)
	if err != nil {
		return err
	}
	if approved {
		return fmt.Errorf("workflow: already approved by this user")
	}

	// Record approval
	approval := &Approval{
		RequestID:  requestID,
		ApproverID: approverID,
		Decision:   "approve",
		Comments:   comments,
	}
	if err := s.repo.RecordApproval(ctx, approval); err != nil {
		return err
	}

	// Check if request is fully approved
	if req.ReceivedApprovals+1 >= req.RequiredApprovals {
		now := time.Now()
		req.Status = StatusApproved
		req.ApprovedAt = &now
		if err := s.repo.UpdateRequestStatus(ctx, requestID, StatusApproved, &now, nil); err != nil {
			return err
		}

		// Publish approval event
		_ = s.publisher.PublishCheckoutApproved(ctx, req.TenantID.String(), approverID.String(), requestID.String())
	}

	return nil
}

// DenyRequest denies a request
func (s *WorkflowService) DenyRequest(ctx context.Context, requestID, approverID uuid.UUID, comments string) error {
	// Get request
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if req.Status != StatusPending {
		return fmt.Errorf("workflow: request is not pending (status: %s)", req.Status)
	}

	// Record denial
	approval := &Approval{
		RequestID:  requestID,
		ApproverID: approverID,
		Decision:   "deny",
		Comments:   comments,
	}
	if err := s.repo.RecordApproval(ctx, approval); err != nil {
		return err
	}

	// Update request status
	now := time.Now()
	req.Status = StatusDenied
	req.DeniedAt = &now
	if err := s.repo.UpdateRequestStatus(ctx, requestID, StatusDenied, nil, &now); err != nil {
		return err
	}

	return nil
}

// CancelRequest cancels a request (by requester)
func (s *WorkflowService) CancelRequest(ctx context.Context, requestID, userID uuid.UUID) error {
	req, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return err
	}

	if req.UserID != userID {
		return fmt.Errorf("workflow: only requester can cancel")
	}

	if req.Status != StatusPending {
		return fmt.Errorf("workflow: cannot cancel non-pending request")
	}

	return s.repo.UpdateRequestStatus(ctx, requestID, StatusCancelled, nil, nil)
}

// DelegateApproval delegates approval to another user
func (s *WorkflowService) DelegateApproval(ctx context.Context, requestID, fromApproverID, toApproverID uuid.UUID) error {
	// Check if from approver has already approved
	approved, err := s.repo.CheckIfApproved(ctx, requestID, fromApproverID)
	if err != nil {
		return err
	}
	if !approved {
		return fmt.Errorf("workflow: can only delegate after approving")
	}

	// Record delegation approval
	approval := &Approval{
		RequestID:    requestID,
		ApproverID:   toApproverID,
		Decision:     "approve",
		DelegatedFrom: &fromApproverID,
	}
	if err := s.repo.RecordApproval(ctx, approval); err != nil {
		return err
	}

	return nil
}

// shouldAutoApprove checks if a request should be auto-approved based on policies
func (s *WorkflowService) shouldAutoApprove(ctx context.Context, req *Request) bool {
	// Check for auto-approve policies for this tenant
	policies, err := s.getApprovalPolicies(ctx, req.TenantID)
	if err != nil {
		return false
	}

	for _, policy := range policies {
		if policy.AutoApprove && s.matchesPolicy(ctx, req, policy) {
			return true
		}
	}

	return false
}

// matchesPolicy checks if a request matches a policy
func (s *WorkflowService) matchesPolicy(ctx context.Context, req *Request, policy ApprovalPolicy) bool {
	// Implement policy matching logic
	// This would check user role, time of day, target sensitivity, etc.
	return false
}

// getApprovalGroupForCredential gets the approval group for a credential
func (s *WorkflowService) getApprovalGroupForCredential(ctx context.Context, credentialID uuid.UUID) (*uuid.UUID, int, error) {
	// Query credential to get target, then get approval group
	return nil, 1, nil // Default
}

// getApprovalGroupForTarget gets the approval group for a target
func (s *WorkflowService) getApprovalGroupForTarget(ctx context.Context, targetID uuid.UUID) (*uuid.UUID, int, error) {
	// Query target for approval_group_id
	return nil, 1, nil // Default
}

// getRequiredApprovalsForGroup gets the number of required approvals for a group
func (s *WorkflowService) getRequiredApprovalsForGroup(ctx context.Context, groupID uuid.UUID) (int, error) {
	// Query approval_group table
	return 1, nil // Default
}

// getApprovalPolicies retrieves active approval policies for a tenant
func (s *WorkflowService) getApprovalPolicies(ctx context.Context, tenantID uuid.UUID) ([]ApprovalPolicy, error) {
	// Query approval_policies table
	return nil, nil
}

// CheckExpiredRequests checks for expired pending requests
func (s *WorkflowService) CheckExpiredRequests(ctx context.Context) error {
	query := `
		UPDATE approval_requests
		SET status = 'expired', updated_at = NOW()
		WHERE status = 'pending'
			AND expires_at IS NOT NULL
			AND expires_at <= NOW()
	`
	_, err := s.repo.db.ExecContext(ctx, query)
	return err
}
