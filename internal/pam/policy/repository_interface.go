package policy

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the policy repository interface
// This allows for mocking in tests and alternative implementations
type RepositoryInterface interface {
	// Policy CRUD
	Create(ctx context.Context, policy *Policy) error
	GetByID(ctx context.Context, id uuid.UUID) (*Policy, error)
	GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Policy, error)
	List(ctx context.Context, tenantID uuid.UUID, filter PolicyFilter, limit, offset int) ([]Policy, int, error)
	GetActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]Policy, error)
	GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]Policy, error)
	Update(ctx context.Context, policy *Policy) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	Enable(ctx context.Context, id, tenantID uuid.UUID) error
	Disable(ctx context.Context, id, tenantID uuid.UUID) error

	// Evaluation Logs
	CreateEvaluationLog(ctx context.Context, log *PolicyEvalLog) error
	GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]PolicyEvalLog, int, error)

	// Approval Requests
	CreateApprovalRequest(ctx context.Context, req *ApprovalRequest) error
	GetApprovalRequestByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error)
	UpdateApprovalRequestStatus(ctx context.Context, id uuid.UUID, status ApprovalStatus, approverID *uuid.UUID, denialReason string) error
	GetPendingApprovalRequests(ctx context.Context, tenantID uuid.UUID, approverID *uuid.UUID) ([]ApprovalRequest, error)

	// Policy Templates
	CreatePolicyTemplate(ctx context.Context, template *PolicyTemplate) error
	GetPolicyTemplates(ctx context.Context, category *string) ([]PolicyTemplate, error)

	// Command Filter Patterns
	CreateCommandFilterPattern(ctx context.Context, pattern *CommandFilterPattern) error
	GetCommandFilterPatterns(ctx context.Context, policyID uuid.UUID) ([]CommandFilterPattern, error)
}

// Ensure Repository implements the interface
var _ RepositoryInterface = (*Repository)(nil)
