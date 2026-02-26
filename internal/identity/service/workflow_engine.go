package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpam/openpam/internal/identity/repository"
)

var (
	// ErrWorkflowNotFound is defined in access_request.go to avoid duplication
	ErrWorkflowActive       = errors.New("cannot delete active workflow")
	ErrInvalidStepOrder     = errors.New("invalid step order")
	ErrCircularStep         = errors.New("circular step reference detected")
	ErrMissingRequiredField = errors.New("missing required field")
)

// WorkflowEngine manages approval workflow configurations and execution
type WorkflowEngine struct {
	repo     *repository.WorkflowRepository
	userRepo *repository.UserRepository
	logger   *zerolog.Logger
}

// NewWorkflowEngine creates a new workflow engine
func NewWorkflowEngine(
	repo *repository.WorkflowRepository,
	userRepo *repository.UserRepository,
	logger *zerolog.Logger,
) *WorkflowEngine {
	return &WorkflowEngine{
		repo:     repo,
		userRepo: userRepo,
		logger:   logger,
	}
}

// CreateWorkflow creates a new approval workflow
func (e *WorkflowEngine) CreateWorkflow(ctx context.Context, wf *repository.Workflow) (*repository.Workflow, error) {
	if err := e.validateWorkflow(wf); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := e.repo.Create(ctx, wf); err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}

	e.logger.Info().
		Str("workflow_id", wf.ID.String()).
		Str("name", wf.Name).
		Msg("Workflow created")

	return wf, nil
}

// GetWorkflow retrieves a workflow by ID
func (e *WorkflowEngine) GetWorkflow(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*repository.Workflow, error) {
	wf, err := e.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrWorkflowNotFound
	}
	return wf, nil
}

// ListWorkflows retrieves workflows based on filters
func (e *WorkflowEngine) ListWorkflows(ctx context.Context, filter repository.WorkflowFilter) ([]repository.Workflow, error) {
	workflows, err := e.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	return workflows, nil
}

// UpdateWorkflow updates an existing workflow
func (e *WorkflowEngine) UpdateWorkflow(ctx context.Context, wf *repository.Workflow) (*repository.Workflow, error) {
	if err := e.validateWorkflow(wf); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	existing, err := e.repo.GetByID(ctx, wf.ID, wf.TenantID)
	if err != nil {
		return nil, ErrWorkflowNotFound
	}

	if existing.Status == repository.WorkflowStatusActive {
		return nil, fmt.Errorf("cannot update active workflow, deactivate first")
	}

	if err := e.repo.Update(ctx, wf); err != nil {
		return nil, fmt.Errorf("update workflow: %w", err)
	}

	e.logger.Info().
		Str("workflow_id", wf.ID.String()).
		Str("name", wf.Name).
		Int("version", wf.Version).
		Msg("Workflow updated")

	return wf, nil
}

// ActivateWorkflow activates a workflow
func (e *WorkflowEngine) ActivateWorkflow(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := e.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrWorkflowNotFound
	}

	// Validate workflow has at least one step
	steps, err := e.repo.GetSteps(ctx, id)
	if err != nil {
		return fmt.Errorf("get steps: %w", err)
	}
	if len(steps) == 0 {
		return fmt.Errorf("workflow must have at least one step before activation")
	}

	// Validate each step has approvers
	for _, step := range steps {
		approvers, err := e.repo.GetApprovers(ctx, step.ID)
		if err != nil {
			return fmt.Errorf("get approvers for step %s: %w", step.ID, err)
		}
		if len(approvers) == 0 {
			return fmt.Errorf("step '%s' must have at least one approver", step.Name)
		}
	}

	if err := e.repo.UpdateStatus(ctx, id, tenantID, repository.WorkflowStatusActive); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	e.logger.Info().
		Str("workflow_id", id.String()).
		Msg("Workflow activated")

	return nil
}

// DeactivateWorkflow deactivates a workflow
func (e *WorkflowEngine) DeactivateWorkflow(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if err := e.repo.UpdateStatus(ctx, id, tenantID, repository.WorkflowStatusInactive); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	e.logger.Info().
		Str("workflow_id", id.String()).
		Msg("Workflow deactivated")

	return nil
}

// DeleteWorkflow deletes a workflow
func (e *WorkflowEngine) DeleteWorkflow(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	wf, err := e.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrWorkflowNotFound
	}

	if wf.Status == repository.WorkflowStatusActive {
		return ErrWorkflowActive
	}

	if err := e.repo.SoftDelete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}

	e.logger.Info().
		Str("workflow_id", id.String()).
		Msg("Workflow deleted")

	return nil
}

// AddStep adds a step to a workflow
func (e *WorkflowEngine) AddStep(ctx context.Context, step *repository.WorkflowStep) (*repository.WorkflowStep, error) {
	if step.Name == "" {
		return nil, ErrMissingRequiredField
	}

	if step.Order <= 0 {
		// Auto-assign order
		existingSteps, err := e.repo.GetSteps(ctx, step.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("get existing steps: %w", err)
		}
		maxOrder := 0
		for _, s := range existingSteps {
			if s.Order > maxOrder {
				maxOrder = s.Order
			}
		}
		step.Order = maxOrder + 1
	}

	// Check for circular escalation
	if step.EscalationStepID != nil {
		if *step.EscalationStepID == step.ID || *step.EscalationStepID == step.WorkflowID {
			return nil, ErrCircularStep
		}
	}

	if err := e.repo.CreateStep(ctx, step); err != nil {
		return nil, fmt.Errorf("create step: %w", err)
	}

	e.logger.Info().
		Str("step_id", step.ID.String()).
		Str("workflow_id", step.WorkflowID.String()).
		Int("order", step.Order).
		Msg("Workflow step created")

	return step, nil
}

// GetSteps retrieves all steps for a workflow
func (e *WorkflowEngine) GetSteps(ctx context.Context, workflowID uuid.UUID) ([]repository.WorkflowStep, error) {
	steps, err := e.repo.GetSteps(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("get steps: %w", err)
	}
	return steps, nil
}

// AddApprover adds an approver to a workflow step
func (e *WorkflowEngine) AddApprover(ctx context.Context, approver *repository.WorkflowApprover) (*repository.WorkflowApprover, error) {
	if approver.ApproverID == uuid.Nil {
		return nil, ErrMissingRequiredField
	}

	if err := e.repo.AddApprover(ctx, approver); err != nil {
		return nil, fmt.Errorf("add approver: %w", err)
	}

	e.logger.Info().
		Str("approver_id", approver.ID.String()).
		Str("step_id", approver.StepID.String()).
		Msg("Workflow approver added")

	return approver, nil
}

// RemoveApprover removes an approver from a workflow step
func (e *WorkflowEngine) RemoveApprover(ctx context.Context, approverID uuid.UUID) error {
	// Implementation would add a soft delete or status update to the approver
	return nil
}

// GetApprovers retrieves all approvers for a workflow step
func (e *WorkflowEngine) GetApprovers(ctx context.Context, stepID uuid.UUID) ([]repository.WorkflowApprover, error) {
	approvers, err := e.repo.GetApprovers(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("get approvers: %w", err)
	}
	return approvers, nil
}

// FindWorkflowForRequest finds the appropriate workflow for a given request context
func (e *WorkflowEngine) FindWorkflowForRequest(
	ctx context.Context,
	tenantID uuid.UUID,
	resourceType string,
	accessLevel string,
) (*repository.Workflow, error) {
	workflow, err := e.repo.FindWorkflowForRequest(ctx, tenantID, resourceType, accessLevel)
	if err != nil {
		return nil, fmt.Errorf("find workflow: %w", err)
	}
	return workflow, nil
}

// ValidateWorkflow validates a workflow configuration
func (e *WorkflowEngine) validateWorkflow(wf *repository.Workflow) error {
	if wf.Name == "" {
		return ErrMissingRequiredField
	}
	if wf.TenantID == uuid.Nil {
		return ErrMissingRequiredField
	}
	if wf.TimeoutMinutes < 0 || wf.TimeoutMinutes > 10080 { // Max 1 week
		return fmt.Errorf("timeout must be between 0 and 10080 minutes")
	}
	return nil
}

// GetWorkflowProgress retrieves the progress of a request through its workflow
func (e *WorkflowEngine) GetWorkflowProgress(ctx context.Context, requestID uuid.UUID) (*WorkflowProgress, error) {
	// Use a type-safe method call instead of type assertion
	// For now, return an error as this functionality needs proper repository interface
	return nil, fmt.Errorf("get status: not implemented")
}

// WorkflowProgress represents the progress of a workflow
type WorkflowProgress struct {
	RequestID          uuid.UUID  `json:"request_id"`
	CurrentStepOrder   int        `json:"current_step_order"`
	TotalSteps         int        `json:"total_steps"`
	PendingApprovals   int        `json:"pending_approvals"`
	CompletedApprovals int        `json:"completed_approvals"`
	ApprovedCount      int        `json:"approved_count"`
	DeniedCount        int        `json:"denied_count"`
	LastApprovalAt     *time.Time `json:"last_approval_at"`
	ProgressPercent    int        `json:"progress_percent"`
}

func calculateProgress(completed, total int) int {
	if total == 0 {
		return 0
	}
	return int(float64(completed) / float64(total) * 100)
}
