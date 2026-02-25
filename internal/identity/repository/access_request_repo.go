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

// AccessRequestStatus represents the status of an access request
type AccessRequestStatus string

const (
	RequestStatusDraft      AccessRequestStatus = "draft"
	RequestStatusPending    AccessRequestStatus = "pending"
	RequestStatusApproved   AccessRequestStatus = "approved"
	RequestStatusDenied     AccessRequestStatus = "denied"
	RequestStatusCancelled  AccessRequestStatus = "cancelled"
	RequestStatusExpired    AccessRequestStatus = "expired"
	RequestStatusActive     AccessRequestStatus = "active"
	RequestStatusCompleted  AccessRequestStatus = "completed"
	RequestStatusRevoked    AccessRequestStatus = "revoked"
)

// AccessRequestType represents the type of access being requested
type AccessRequestType string

const (
	RequestTypeCredential    AccessRequestType = "credential"
	RequestTypeSession       AccessRequestType = "session"
	RequestTypeBreakGlass    AccessRequestType = "break_glass"
	RequestTypePrivilege     AccessRequestType = "privilege"
)

// AccessRequest represents an access request in the system
type AccessRequest struct {
	ID                uuid.UUID          `json:"id" db:"id"`
	TenantID          uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	RequesterID       uuid.UUID          `json:"requester_id" db:"requester_id"`
	Type              AccessRequestType  `json:"type" db:"type"`
	Status            AccessRequestStatus `json:"status" db:"status"`
	Title             string             `json:"title" db:"title"`
	Description       string             `json:"description" db:"description"`
	TargetResourceID  uuid.UUID          `json:"target_resource_id" db:"target_resource_id"`
	TargetResourceType string            `json:"target_resource_type" db:"target_resource_type"`
	RequestedDuration int                `json:"requested_duration" db:"requested_duration"` // minutes
	StartTime         *time.Time         `json:"start_time" db:"start_time"`
	EndTime           *time.Time         `json:"end_time" db:"end_time"`
	Reason            string             `json:"reason" db:"reason"`
	BusinessJustification string         `json:"business_justification" db:"business_justification"`
	TicketRef         string             `json:"ticket_ref" db:"ticket_ref"` // ITSM ticket reference
	WorkflowID        *uuid.UUID         `json:"workflow_id" db:"workflow_id"`
	Priority          string             `json:"priority" db:"priority"` // low, medium, high, critical
	Metadata          map[string]string  `json:"metadata" db:"metadata"`
	CreatedAt         time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" db:"updated_at"`
	DeletedAt         *time.Time         `json:"deleted_at,omitempty" db:"deleted_at"`
}

// AccessRequestFilter represents filters for listing access requests
type AccessRequestFilter struct {
	TenantID        *uuid.UUID
	RequesterID     *uuid.UUID
	Status          *AccessRequestStatus
	Type            *AccessRequestType
	WorkflowID      *uuid.UUID
	Priority        *string
	StartDate       *time.Time
	EndDate         *time.Time
	TicketRef       *string
	AfterID         *uuid.UUID
	Limit           int
}

// AccessRequestRepository handles database operations for access requests
type AccessRequestRepository struct {
	db     *sqlx.DB
	logger *zerolog.Logger
}

// NewAccessRequestRepository creates a new access request repository
func NewAccessRequestRepository(db *sqlx.DB, logger *zerolog.Logger) *AccessRequestRepository {
	return &AccessRequestRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new access request
func (r *AccessRequestRepository) Create(ctx context.Context, req *AccessRequest) error {
	req.ID = uuid.New()
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	query := `
		INSERT INTO access_requests (
			id, tenant_id, requester_id, type, status, title, description,
			target_resource_id, target_resource_type, requested_duration,
			start_time, end_time, reason, business_justification, ticket_ref,
			workflow_id, priority, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :requester_id, :type, :status, :title, :description,
			:target_resource_id, :target_resource_type, :requested_duration,
			:start_time, :end_time, :reason, :business_justification, :ticket_ref,
			:workflow_id, :priority, :metadata, :created_at, :updated_at
		) RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, req)
	if err != nil {
		return fmt.Errorf("access_request.Create: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&req.ID); err != nil {
			return fmt.Errorf("access_request.Create scan: %w", err)
		}
	}

	return nil
}

// GetByID retrieves an access request by ID
func (r *AccessRequestRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*AccessRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, type, status, title, description,
			target_resource_id, target_resource_type, requested_duration,
			start_time, end_time, reason, business_justification, ticket_ref,
			workflow_id, priority, metadata, created_at, updated_at, deleted_at
		FROM access_requests
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	var req AccessRequest
	err := r.db.GetContext(ctx, &req, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("access_request not found")
		}
		return nil, fmt.Errorf("access_request.GetByID: %w", err)
	}

	return &req, nil
}

// List retrieves access requests based on filters
func (r *AccessRequestRepository) List(ctx context.Context, filter AccessRequestFilter) ([]AccessRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, type, status, title, description,
			target_resource_id, target_resource_type, requested_duration,
			start_time, end_time, reason, business_justification, ticket_ref,
			workflow_id, priority, metadata, created_at, updated_at, deleted_at
		FROM access_requests
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}
	if filter.RequesterID != nil {
		query += fmt.Sprintf(" AND requester_id = $%d", argCount)
		args = append(args, *filter.RequesterID)
		argCount++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.WorkflowID != nil {
		query += fmt.Sprintf(" AND workflow_id = $%d", argCount)
		args = append(args, *filter.WorkflowID)
		argCount++
	}
	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, *filter.Priority)
		argCount++
	}
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartDate)
		argCount++
	}
	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndDate)
		argCount++
	}
	if filter.TicketRef != nil {
		query += fmt.Sprintf(" AND ticket_ref = $%d", argCount)
		args = append(args, *filter.TicketRef)
		argCount++
	}
	if filter.AfterID != nil {
		query += fmt.Sprintf(" AND id > $%d", argCount)
		args = append(args, *filter.AfterID)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 50"
	}

	var requests []AccessRequest
	err := r.db.SelectContext(ctx, &requests, query, args...)
	if err != nil {
		return nil, fmt.Errorf("access_request.List: %w", err)
	}

	return requests, nil
}

// Update updates an access request
func (r *AccessRequestRepository) Update(ctx context.Context, req *AccessRequest) error {
	req.UpdatedAt = time.Now()

	query := `
		UPDATE access_requests
		SET status = :status,
			title = :title,
			description = :description,
			start_time = :start_time,
			end_time = :end_time,
			reason = :reason,
			business_justification = :business_justification,
			ticket_ref = :ticket_ref,
			priority = :priority,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`

	result, err := r.db.NamedExecContext(ctx, query, req)
	if err != nil {
		return fmt.Errorf("access_request.Update: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("access_request.Update rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("access_request not found")
	}

	return nil
}

// UpdateStatus updates the status of an access request
func (r *AccessRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, status AccessRequestStatus) error {
	query := `
		UPDATE access_requests
		SET status = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("access_request.UpdateStatus: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("access_request.UpdateStatus rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("access_request not found")
	}

	return nil
}

// SoftDelete soft deletes an access request
func (r *AccessRequestRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		UPDATE access_requests
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("access_request.SoftDelete: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("access_request.SoftDelete rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("access_request not found")
	}

	return nil
}

// GetPendingApprovals retrieves requests pending approval for a specific approver
func (r *AccessRequestRepository) GetPendingApprovals(ctx context.Context, approverID uuid.UUID, tenantID uuid.UUID) ([]AccessRequest, error) {
	query := `
		SELECT DISTINCT ar.id, ar.tenant_id, ar.requester_id, ar.type, ar.status, ar.title, ar.description,
			ar.target_resource_id, ar.target_resource_type, ar.requested_duration,
			ar.start_time, ar.end_time, ar.reason, ar.business_justification, ar.ticket_ref,
			ar.workflow_id, ar.priority, ar.metadata, ar.created_at, ar.updated_at, ar.deleted_at
		FROM access_requests ar
		INNER JOIN workflow_steps ws ON ar.workflow_id = ws.workflow_id
		INNER JOIN workflow_approvers wa ON ws.id = wa.step_id
		WHERE ar.status = 'pending'
			AND ar.tenant_id = $1
			AND wa.approver_id = $2
			AND ar.deleted_at IS NULL
			AND wa.approved_at IS NULL
		ORDER BY ar.created_at ASC
	`

	var requests []AccessRequest
	err := r.db.SelectContext(ctx, &requests, query, tenantID, approverID)
	if err != nil {
		return nil, fmt.Errorf("access_request.GetPendingApprovals: %w", err)
	}

	return requests, nil
}

// GetByTicketRef retrieves an access request by ITSM ticket reference
func (r *AccessRequestRepository) GetByTicketRef(ctx context.Context, ticketRef string, tenantID uuid.UUID) (*AccessRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, type, status, title, description,
			target_resource_id, target_resource_type, requested_duration,
			start_time, end_time, reason, business_justification, ticket_ref,
			workflow_id, priority, metadata, created_at, updated_at, deleted_at
		FROM access_requests
		WHERE ticket_ref = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	var req AccessRequest
	err := r.db.GetContext(ctx, &req, query, ticketRef, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("access_request not found")
		}
		return nil, fmt.Errorf("access_request.GetByTicketRef: %w", err)
	}

	return &req, nil
}

// GetActiveRequests retrieves all currently active access requests for a tenant
func (r *AccessRequestRepository) GetActiveRequests(ctx context.Context, tenantID uuid.UUID) ([]AccessRequest, error) {
	query := `
		SELECT id, tenant_id, requester_id, type, status, title, description,
			target_resource_id, target_resource_type, requested_duration,
			start_time, end_time, reason, business_justification, ticket_ref,
			workflow_id, priority, metadata, created_at, updated_at, deleted_at
		FROM access_requests
		WHERE status = 'active'
			AND tenant_id = $1
			AND deleted_at IS NULL
			AND end_time > NOW()
		ORDER BY end_time ASC
	`

	var requests []AccessRequest
	err := r.db.SelectContext(ctx, &requests, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("access_request.GetActiveRequests: %w", err)
	}

	return requests, nil
}
