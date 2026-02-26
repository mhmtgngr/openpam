package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/identity/repository"
)

var (
	ErrRequestNotFound      = errors.New("access request not found")
	ErrRequestAlreadyApproved = errors.New("request already approved")
	ErrRequestAlreadyDenied = errors.New("request already denied")
	ErrRequestNotPending    = errors.New("request is not pending approval")
	ErrUnauthorizedApprover = errors.New("user not authorized to approve this request")
	ErrApprovalRequired     = errors.New("request requires approval")
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrInvalidDuration      = errors.New("invalid duration requested")
	ErrTicketRequired       = errors.New("ITSM ticket reference required")
	ErrMFAMissing           = errors.New("MFA verification required")
)

// ITSMClient defines the interface for ITSM system integrations
type ITSMClient interface {
	Type() string
	CreateTicket(ctx context.Context, req *ITSMTicketCreate) (*ITSMTicket, error)
	GetTicketStatus(ctx context.Context, ticketID string) (string, error)
	UpdateTicket(ctx context.Context, ticketID string, status string, notes string) error
	AddComment(ctx context.Context, ticketID string, comment string) error
}

// ITSMTicketCreate represents a request to create an ITSM ticket
type ITSMTicketCreate struct {
	Title       string
	Description string
	Requester   string
	Priority    string
	Type        string
	Metadata    map[string]string
}

// ITSMTicket represents an ITSM ticket
type ITSMTicket struct {
	ID       string
	Status   string
	URL      string
	Metadata map[string]string
}

// AccessRequestService handles business logic for access requests
type AccessRequestService struct {
	repo           *repository.AccessRequestRepository
	workflowRepo   *repository.WorkflowRepository
	approvalRepo   *repository.ApprovalRepository
	userRepo       *repository.UserRepository
	cache          *cache.Cache
	itsmClients    []ITSMClient
	logger         *zerolog.Logger
}

// NewAccessRequestService creates a new access request service
func NewAccessRequestService(
	repo *repository.AccessRequestRepository,
	workflowRepo *repository.WorkflowRepository,
	approvalRepo *repository.ApprovalRepository,
	userRepo *repository.UserRepository,
	cache *cache.Cache,
	itsmClients []ITSMClient,
	logger *zerolog.Logger,
) *AccessRequestService {
	return &AccessRequestService{
		repo:         repo,
		workflowRepo: workflowRepo,
		approvalRepo: approvalRepo,
		userRepo:     userRepo,
		cache:        cache,
		itsmClients:  itsmClients,
		logger:       logger,
	}
}

// CreateRequest creates a new access request
func (s *AccessRequestService) CreateRequest(ctx context.Context, req *repository.AccessRequest) (*repository.AccessRequest, error) {
	// Validate duration
	if req.RequestedDuration <= 0 || req.RequestedDuration > 10080 { // Max 1 week
		return nil, ErrInvalidDuration
	}

	// Find applicable workflow
	workflow, err := s.workflowRepo.FindWorkflowForRequest(
		ctx,
		req.TenantID,
		req.TargetResourceType,
		req.Priority,
	)
	if err != nil {
		return nil, fmt.Errorf("find workflow: %w", err)
	}

	// Set workflow and initial status
	if workflow != nil {
		req.WorkflowID = &workflow.ID

		// Check if ticket is required
		if workflow.RequireTicket && req.TicketRef == "" {
			// Create ITSM ticket if configured
			if len(s.itsmClients) > 0 {
				ticket, err := s.createITSMTicket(ctx, req)
				if err != nil {
					s.logger.Error().Err(err).Str("request_id", req.ID.String()).Msg("Failed to create ITSM ticket")
					return nil, fmt.Errorf("create ITSM ticket: %w", err)
				}
				req.TicketRef = ticket.ID
			} else {
				return nil, ErrTicketRequired
			}
		}

		req.Status = repository.RequestStatusPending
	} else {
		// Auto-approve if no workflow
		req.Status = repository.RequestStatusApproved
		now := time.Now()
		req.StartTime = &now
		endTime := now.Add(time.Duration(req.RequestedDuration) * time.Minute)
		req.EndTime = &endTime
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Send notifications
	if req.Status == repository.RequestStatusPending {
		s.notifyApprovers(ctx, req)
	}

	return req, nil
}

// GetRequest retrieves an access request
func (s *AccessRequestService) GetRequest(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*repository.AccessRequest, error) {
	req, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrRequestNotFound
	}
	return req, nil
}

// ListRequests retrieves access requests based on filters
func (s *AccessRequestService) ListRequests(ctx context.Context, filter repository.AccessRequestFilter) ([]repository.AccessRequest, error) {
	requests, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	return requests, nil
}

// ApproveRequest approves an access request
func (s *AccessRequestService) ApproveRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, approverID uuid.UUID, comments string, mfaVerified bool) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != repository.RequestStatusPending {
		return ErrRequestNotPending
	}

	// Get current workflow step
	step, err := s.approvalRepo.GetCurrentStepForRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("get current step: %w", err)
	}
	if step == nil {
		return ErrWorkflowNotFound
	}

	// Check if user is authorized to approve
	authorized, err := s.isAuthorizedApprover(ctx, approverID, step.ID)
	if err != nil {
		return fmt.Errorf("check authorization: %w", err)
	}
	if !authorized {
		return ErrUnauthorizedApprover
	}

	// Check if already approved
	alreadyApproved, err := s.approvalRepo.HasUserApproved(ctx, requestID, approverID, step.ID)
	if err != nil {
		return fmt.Errorf("check existing approval: %w", err)
	}
	if alreadyApproved {
		return ErrRequestAlreadyApproved
	}

	// Verify MFA if required
	workflow, err := s.workflowRepo.GetByID(ctx, *req.WorkflowID, tenantID)
	if err != nil {
		return fmt.Errorf("get workflow: %w", err)
	}
	if workflow.RequireMFA && !mfaVerified {
		return ErrMFAMissing
	}

	// Create approval record
	approval := &repository.Approval{
		TenantID:    tenantID,
		RequestID:   requestID,
		WorkflowID:  *req.WorkflowID,
		StepID:      step.ID,
		ApproverID:  approverID,
		Decision:    "approve",
		Comments:    comments,
		MFAVerified: mfaVerified,
	}
	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		return fmt.Errorf("create approval: %w", err)
	}

	// Update workflow approver
	if err := s.workflowRepo.UpdateApproverResponse(ctx, approval.ID, "approve"); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to update workflow approver response")
	}

	// Check if step is complete
	stepComplete, err := s.isStepComplete(ctx, step)
	if err != nil {
		return fmt.Errorf("check step completion: %w", err)
	}

	if stepComplete {
		// Move to next step or complete request
		if err := s.advanceWorkflow(ctx, req); err != nil {
			return fmt.Errorf("advance workflow: %w", err)
		}
	}

	// Update ITSM ticket
	if req.TicketRef != "" {
		s.updateITSMTicket(ctx, req, "approved")
	}

	return nil
}

// DenyRequest denies an access request
func (s *AccessRequestService) DenyRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, approverID uuid.UUID, comments string, mfaVerified bool) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != repository.RequestStatusPending {
		return ErrRequestNotPending
	}

	// Get current workflow step
	step, err := s.approvalRepo.GetCurrentStepForRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("get current step: %w", err)
	}
	if step == nil {
		return ErrWorkflowNotFound
	}

	// Check authorization
	authorized, err := s.isAuthorizedApprover(ctx, approverID, step.ID)
	if err != nil {
		return fmt.Errorf("check authorization: %w", err)
	}
	if !authorized {
		return ErrUnauthorizedApprover
	}

	// Create denial record
	approval := &repository.Approval{
		TenantID:    tenantID,
		RequestID:   requestID,
		WorkflowID:  *req.WorkflowID,
		StepID:      step.ID,
		ApproverID:  approverID,
		Decision:    "deny",
		Comments:    comments,
		MFAVerified: mfaVerified,
	}
	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		return fmt.Errorf("create approval: %w", err)
	}

	// Deny the entire request
	if err := s.repo.UpdateStatus(ctx, requestID, tenantID, repository.RequestStatusDenied); err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	// Update ITSM ticket
	if req.TicketRef != "" {
		s.updateITSMTicket(ctx, req, "denied")
	}

	// Notify requester
	s.notifyRequester(ctx, req, "denied")

	return nil
}

// CancelRequest cancels an access request
func (s *AccessRequestService) CancelRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrRequestNotFound
	}

	// Only requester can cancel
	if req.RequesterID != userID {
		return ErrUnauthorizedApprover
	}

	if req.Status != repository.RequestStatusPending {
		return ErrRequestNotPending
	}

	if err := s.repo.UpdateStatus(ctx, requestID, tenantID, repository.RequestStatusCancelled); err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	// Update ITSM ticket
	if req.TicketRef != "" {
		s.updateITSMTicket(ctx, req, "cancelled")
	}

	return nil
}

// ActivateRequest activates an approved request
func (s *AccessRequestService) ActivateRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != repository.RequestStatusApproved {
		return ErrRequestNotPending
	}

	now := time.Now()
	req.Status = repository.RequestStatusActive
	req.StartTime = &now
	endTime := now.Add(time.Duration(req.RequestedDuration) * time.Minute)
	req.EndTime = &endTime

	if err := s.repo.Update(ctx, req); err != nil {
		return fmt.Errorf("update request: %w", err)
	}

	// Update ITSM ticket
	if req.TicketRef != "" {
		s.updateITSMTicket(ctx, req, "active")
	}

	return nil
}

// RevokeRequest revokes an active access request
func (s *AccessRequestService) RevokeRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, reason string) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrRequestNotFound
	}

	if req.Status != repository.RequestStatusActive {
		return ErrRequestNotPending
	}

	if err := s.repo.UpdateStatus(ctx, requestID, tenantID, repository.RequestStatusRevoked); err != nil {
		return fmt.Errorf("update request status: %w", err)
	}

	// Update ITSM ticket
	if req.TicketRef != "" {
		s.updateITSMTicketWithComment(ctx, req, "revoked", reason)
	}

	return nil
}

// SyncITSMStatus syncs the status of requests with ITSM systems
func (s *AccessRequestService) SyncITSMStatus(ctx context.Context, client ITSMClient) error {
	status := repository.RequestStatusPending
	requests, err := s.repo.List(ctx, repository.AccessRequestFilter{
		Status:   &status,
		Limit:    100,
	})
	if err != nil {
		return fmt.Errorf("list requests: %w", err)
	}

	for _, req := range requests {
		if req.TicketRef == "" {
			continue
		}

		status, err := client.GetTicketStatus(ctx, req.TicketRef)
		if err != nil {
			s.logger.Error().Err(err).Str("ticket_ref", req.TicketRef).Msg("Failed to get ticket status")
			continue
		}

		// Map ITSM status to request status
		switch status {
		case "approved", "closed":
			if req.Status == repository.RequestStatusPending {
				if err := s.repo.UpdateStatus(ctx, req.ID, req.TenantID, repository.RequestStatusApproved); err != nil {
					s.logger.Error().Err(err).Str("request_id", req.ID.String()).Msg("Failed to update status")
				}
			}
		case "rejected", "cancelled":
			if err := s.repo.UpdateStatus(ctx, req.ID, req.TenantID, repository.RequestStatusDenied); err != nil {
				s.logger.Error().Err(err).Str("request_id", req.ID.String()).Msg("Failed to update status")
			}
		}
	}

	return nil
}

// Helper methods

func (s *AccessRequestService) isAuthorizedApprover(ctx context.Context, approverID uuid.UUID, stepID uuid.UUID) (bool, error) {
	approvers, err := s.workflowRepo.GetApprovers(ctx, stepID)
	if err != nil {
		return false, err
	}

	for _, a := range approvers {
		if a.ApproverID == approverID {
			return true, nil
		}
	}
	return false, nil
}

func (s *AccessRequestService) isStepComplete(ctx context.Context, step *repository.WorkflowStep) (bool, error) {
	approvers, err := s.workflowRepo.GetApprovers(ctx, step.ID)
	if err != nil {
		return false, err
	}

	approvals, err := s.approvalRepo.GetApprovalsForStep(ctx, step.ID)
	if err != nil {
		return false, err
	}

	switch step.ApprovalType {
	case repository.ApprovalTypeAny:
		return len(approvals) > 0, nil
	case repository.ApprovalTypeAll:
		return len(approvals) >= len(approvers), nil
	case repository.ApprovalTypeMajority:
		return len(approvals) > len(approvers)/2, nil
	default:
		return false, nil
	}
}

func (s *AccessRequestService) advanceWorkflow(ctx context.Context, req *repository.AccessRequest) error {
	steps, err := s.workflowRepo.GetSteps(ctx, *req.WorkflowID)
	if err != nil {
		return err
	}

	// Find next pending step
	var hasNextStep bool
	for _, step := range steps {
		approvers, _ := s.workflowRepo.GetApprovers(ctx, step.ID)
		allApproved := true
		for _, a := range approvers {
			if a.ApprovedAt == nil {
				allApproved = false
				break
			}
		}
		if !allApproved {
			hasNextStep = true
			break
		}
	}

	if !hasNextStep {
		// All steps complete, approve request
		if err := s.repo.UpdateStatus(ctx, req.ID, req.TenantID, repository.RequestStatusApproved); err != nil {
			return err
		}

		// Notify requester
		s.notifyRequester(ctx, req, "approved")
	}

	return nil
}

func (s *AccessRequestService) createITSMTicket(ctx context.Context, req *repository.AccessRequest) (*ITSMTicket, error) {
	if len(s.itsmClients) == 0 {
		return nil, errors.New("no ITSM clients configured")
	}

	client := s.itsmClients[0] // Use first configured client

	ticketCreate := &ITSMTicketCreate{
		Title:       req.Title,
		Description: fmt.Sprintf("%s\n\nReason: %s\n\nResource: %s", req.Description, req.Reason, req.TargetResourceType),
		Requester:   req.RequesterID.String(),
		Priority:    req.Priority,
		Type:        string(req.Type),
		Metadata: map[string]string{
			"request_id":     req.ID.String(),
			"resource_id":    req.TargetResourceID.String(),
			"resource_type":  req.TargetResourceType,
			"duration":       fmt.Sprintf("%d minutes", req.RequestedDuration),
			"start_time":     req.StartTime.Format(time.RFC3339),
			"end_time":       req.EndTime.Format(time.RFC3339),
		},
	}

	return client.CreateTicket(ctx, ticketCreate)
}

func (s *AccessRequestService) updateITSMTicket(ctx context.Context, req *repository.AccessRequest, status string) {
	for _, client := range s.itsmClients {
		if err := client.UpdateTicket(ctx, req.TicketRef, status, ""); err != nil {
			s.logger.Error().Err(err).Str("ticket_ref", req.TicketRef).Msg("Failed to update ITSM ticket")
		}
	}
}

func (s *AccessRequestService) updateITSMTicketWithComment(ctx context.Context, req *repository.AccessRequest, status string, comment string) {
	for _, client := range s.itsmClients {
		if comment != "" {
			if err := client.AddComment(ctx, req.TicketRef, comment); err != nil {
				s.logger.Error().Err(err).Str("ticket_ref", req.TicketRef).Msg("Failed to add comment to ITSM ticket")
			}
		}
		if err := client.UpdateTicket(ctx, req.TicketRef, status, comment); err != nil {
			s.logger.Error().Err(err).Str("ticket_ref", req.TicketRef).Msg("Failed to update ITSM ticket")
		}
	}
}

func (s *AccessRequestService) notifyApprovers(ctx context.Context, req *repository.AccessRequest) {
	// Get current step approvers
	step, err := s.approvalRepo.GetCurrentStepForRequest(ctx, req.ID)
	if err != nil || step == nil {
		return
	}

	approvers, err := s.workflowRepo.GetApprovers(ctx, step.ID)
	if err != nil {
		return
	}

	for _, a := range approvers {
		// Send notification to each approver
		s.logger.Info().
			Str("approver_id", a.ApproverID.String()).
			Str("request_id", req.ID.String()).
			Msg("Notification sent to approver")
	}
}

func (s *AccessRequestService) notifyRequester(ctx context.Context, req *repository.AccessRequest, status string) {
	s.logger.Info().
		Str("requester_id", req.RequesterID.String()).
		Str("request_id", req.ID.String()).
		Str("status", status).
		Msg("Notification sent to requester")
}
