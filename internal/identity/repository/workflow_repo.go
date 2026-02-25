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

// WorkflowStatus represents the status of a workflow
type WorkflowStatus string

const (
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusInactive WorkflowStatus = "inactive"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// StepType represents the type of workflow step
type StepType string

const (
	StepTypeApproval StepType = "approval"
	StepTypeNotification StepType = "notification"
	StepTypeCondition StepType = "condition"
	StepTypeAutomation StepType = "automation"
)

// ApprovalType represents how approval is handled
type ApprovalType string

const (
	ApprovalTypeAny      ApprovalType = "any"      // Any one approver
	ApprovalTypeAll      ApprovalType = "all"      // All approvers required
	ApprovalTypeMajority ApprovalType = "majority" // Majority of approvers
	ApprovalTypeManager  ApprovalType = "manager"  // Direct manager approval
)

// Workflow represents an approval workflow configuration
type Workflow struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	TenantID    uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Status      WorkflowStatus   `json:"status" db:"status"`
	Version     int              `json:"version" db:"version"`
	// Conditions for when this workflow applies
	ResourceTypes []string        `json:"resource_types" db:"resource_types"` // credential, session, system
	AccessLevels  []string        `json:"access_levels" db:"access_levels"`   // read, write, admin
	Priority      string          `json:"priority" db:"priority"`             // low, medium, high
	TimeoutMinutes int           `json:"timeout_minutes" db:"timeout_minutes"`
	RequireMFA    bool            `json:"require_mfa" db:"require_mfa"`
	RequireTicket bool            `json:"require_ticket" db:"require_ticket"`
	Metadata      map[string]string `json:"metadata" db:"metadata"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// WorkflowStep represents a single step in a workflow
type WorkflowStep struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	WorkflowID  uuid.UUID  `json:"workflow_id" db:"workflow_id"`
	Order       int        `json:"order" db:"order"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Type        StepType   `json:"type" db:"type"`
	ApprovalType ApprovalType `json:"approval_type" db:"approval_type"`
	TimeoutMinutes int      `json:"timeout_minutes" db:"timeout_minutes"`
	AutoApprove  bool       `json:"auto_approve" db:"auto_approve"`
	EscalationStepID *uuid.UUID `json:"escalation_step_id" db:"escalation_step_id"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkflowApprover represents an approver assigned to a workflow step
type WorkflowApprover struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	StepID     uuid.UUID  `json:"step_id" db:"step_id"`
	ApproverID uuid.UUID  `json:"approver_id" db:"approver_id"`
	ApproverType string   `json:"approver_type" db:"approver_type"` // user, group, role
	Order      int        `json:"order" db:"order"`
	ApprovedAt *time.Time `json:"approved_at" db:"approved_at"`
	Response   string     `json:"response" db:"response"` // approve, deny, comment
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// WorkflowFilter represents filters for listing workflows
type WorkflowFilter struct {
	TenantID     *uuid.UUID
	Status       *WorkflowStatus
	ResourceType *string
	Priority     *string
	AfterID      *uuid.UUID
	Limit        int
}

// WorkflowRepository handles database operations for workflows
type WorkflowRepository struct {
	db     *sqlx.DB
	logger *zerolog.Logger
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *sqlx.DB, logger *zerolog.Logger) *WorkflowRepository {
	return &WorkflowRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new workflow
func (r *WorkflowRepository) Create(ctx context.Context, wf *Workflow) error {
	wf.ID = uuid.New()
	wf.CreatedAt = time.Now()
	wf.UpdatedAt = time.Now()
	wf.Version = 1

	query := `
		INSERT INTO workflows (
			id, tenant_id, name, description, status, version,
			resource_types, access_levels, priority, timeout_minutes,
			require_mfa, require_ticket, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :description, :status, :version,
			:resource_types, :access_levels, :priority, :timeout_minutes,
			:require_mfa, :require_ticket, :metadata, :created_at, :updated_at
		) RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, wf)
	if err != nil {
		return fmt.Errorf("workflow.Create: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&wf.ID); err != nil {
			return fmt.Errorf("workflow.Create scan: %w", err)
		}
	}

	return nil
}

// GetByID retrieves a workflow by ID
func (r *WorkflowRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version,
			resource_types, access_levels, priority, timeout_minutes,
			require_mfa, require_ticket, metadata, created_at, updated_at, deleted_at
		FROM workflows
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`

	var wf Workflow
	err := r.db.GetContext(ctx, &wf, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, fmt.Errorf("workflow.GetByID: %w", err)
	}

	return &wf, nil
}

// List retrieves workflows based on filters
func (r *WorkflowRepository) List(ctx context.Context, filter WorkflowFilter) ([]Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version,
			resource_types, access_levels, priority, timeout_minutes,
			require_mfa, require_ticket, metadata, created_at, updated_at, deleted_at
		FROM workflows
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, *filter.Priority)
		argCount++
	}
	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND $%d = ANY(resource_types)", argCount)
		args = append(args, *filter.ResourceType)
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

	var workflows []Workflow
	err := r.db.SelectContext(ctx, &workflows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("workflow.List: %w", err)
	}

	return workflows, nil
}

// Update updates a workflow
func (r *WorkflowRepository) Update(ctx context.Context, wf *Workflow) error {
	wf.UpdatedAt = time.Now()
	wf.Version++

	query := `
		UPDATE workflows
		SET name = :name,
			description = :description,
			status = :status,
			version = :version,
			resource_types = :resource_types,
			access_levels = :access_levels,
			priority = :priority,
			timeout_minutes = :timeout_minutes,
			require_mfa = :require_mfa,
			require_ticket = :require_ticket,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id AND deleted_at IS NULL
	`

	result, err := r.db.NamedExecContext(ctx, query, wf)
	if err != nil {
		return fmt.Errorf("workflow.Update: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("workflow.Update rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// UpdateStatus updates the status of a workflow
func (r *WorkflowRepository) UpdateStatus(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, status WorkflowStatus) error {
	query := `
		UPDATE workflows
		SET status = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("workflow.UpdateStatus: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("workflow.UpdateStatus rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// SoftDelete soft deletes a workflow
func (r *WorkflowRepository) SoftDelete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		UPDATE workflows
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id, tenantID)
	if err != nil {
		return fmt.Errorf("workflow.SoftDelete: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("workflow.SoftDelete rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workflow not found")
	}

	return nil
}

// CreateStep creates a new workflow step
func (r *WorkflowRepository) CreateStep(ctx context.Context, step *WorkflowStep) error {
	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()

	query := `
		INSERT INTO workflow_steps (
			id, workflow_id, order, name, description, type,
			approval_type, timeout_minutes, auto_approve, escalation_step_id,
			created_at, updated_at
		) VALUES (
			:id, :workflow_id, :order, :name, :description, :type,
			:approval_type, :timeout_minutes, :auto_approve, :escalation_step_id,
			:created_at, :updated_at
		) RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, step)
	if err != nil {
		return fmt.Errorf("workflow.CreateStep: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&step.ID); err != nil {
			return fmt.Errorf("workflow.CreateStep scan: %w", err)
		}
	}

	return nil
}

// GetSteps retrieves all steps for a workflow
func (r *WorkflowRepository) GetSteps(ctx context.Context, workflowID uuid.UUID) ([]WorkflowStep, error) {
	query := `
		SELECT id, workflow_id, order, name, description, type,
			approval_type, timeout_minutes, auto_approve, escalation_step_id,
			created_at, updated_at
		FROM workflow_steps
		WHERE workflow_id = $1
		ORDER BY order ASC
	`

	var steps []WorkflowStep
	err := r.db.SelectContext(ctx, &steps, query, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow.GetSteps: %w", err)
	}

	return steps, nil
}

// AddApprover adds an approver to a workflow step
func (r *WorkflowRepository) AddApprover(ctx context.Context, approver *WorkflowApprover) error {
	approver.ID = uuid.New()
	approver.CreatedAt = time.Now()
	approver.UpdatedAt = time.Now()

	query := `
		INSERT INTO workflow_approvers (
			id, step_id, approver_id, approver_type, order, approved_at, response,
			created_at, updated_at
		) VALUES (
			:id, :step_id, :approver_id, :approver_type, :order, :approved_at, :response,
			:created_at, :updated_at
		) RETURNING id
	`

	rows, err := r.db.NamedQueryContext(ctx, query, approver)
	if err != nil {
		return fmt.Errorf("workflow.AddApprover: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&approver.ID); err != nil {
			return fmt.Errorf("workflow.AddApprover scan: %w", err)
		}
	}

	return nil
}

// GetApprovers retrieves all approvers for a workflow step
func (r *WorkflowRepository) GetApprovers(ctx context.Context, stepID uuid.UUID) ([]WorkflowApprover, error) {
	query := `
		SELECT id, step_id, approver_id, approver_type, order, approved_at, response,
			created_at, updated_at
		FROM workflow_approvers
		WHERE step_id = $1
		ORDER BY order ASC
	`

	var approvers []WorkflowApprover
	err := r.db.SelectContext(ctx, &approvers, query, stepID)
	if err != nil {
		return nil, fmt.Errorf("workflow.GetApprovers: %w", err)
	}

	return approvers, nil
}

// UpdateApproverResponse updates an approver's response
func (r *WorkflowRepository) UpdateApproverResponse(ctx context.Context, id uuid.UUID, response string) error {
	now := time.Now()
	query := `
		UPDATE workflow_approvers
		SET response = $1, approved_at = $2, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, response, now, id)
	if err != nil {
		return fmt.Errorf("workflow.UpdateApproverResponse: %w", err)
	}

	return nil
}

// FindWorkflowForRequest finds the appropriate workflow for a given access request
func (r *WorkflowRepository) FindWorkflowForRequest(ctx context.Context, tenantID uuid.UUID, resourceType, accessLevel string) (*Workflow, error) {
	query := `
		SELECT id, tenant_id, name, description, status, version,
			resource_types, access_levels, priority, timeout_minutes,
			require_mfa, require_ticket, metadata, created_at, updated_at, deleted_at
		FROM workflows
		WHERE tenant_id = $1
			AND status = 'active'
			AND ($2 = ANY(resource_types) OR resource_types = '{}')
			AND ($3 = ANY(access_levels) OR access_levels = '{}')
			AND deleted_at IS NULL
		ORDER BY priority DESC
		LIMIT 1
	`

	var wf Workflow
	err := r.db.GetContext(ctx, &wf, query, tenantID, resourceType, accessLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No workflow found
		}
		return nil, fmt.Errorf("workflow.FindWorkflowForRequest: %w", err)
	}

	return &wf, nil
}
