package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/identity/repository"
)

var (
	ErrBreakGlassNotFound    = errors.New("break glass request not found")
	ErrBreakGlassActive      = errors.New("user already has active break glass access")
	ErrBreakGlassExpired     = errors.New("break glass access has expired")
	ErrInvalidJustification  = errors.New("insufficient justification for break glass access")
	ErrMaxDurationExceeded   = errors.New("requested duration exceeds maximum allowed")
	ErrConcurrentBreakGlass  = errors.New("concurrent break glass sessions not allowed")
)

// BreakGlassService manages emergency/break-glass access requests
type BreakGlassService struct {
	repo     *repository.AccessRequestRepository
	approvalRepo *repository.ApprovalRepository
	userRepo  *repository.UserRepository
	cache     *cache.Cache
	logger    *zerolog.Logger

	// Configuration
	maxDurationMinutes int
	requireManagerApproval bool
	requireTicket     bool
	auditLogRetention int // days
}

// NewBreakGlassService creates a new break glass service
func NewBreakGlassService(
	repo *repository.AccessRequestRepository,
	approvalRepo *repository.ApprovalRepository,
	userRepo *repository.UserRepository,
	cache *cache.Cache,
	logger *zerolog.Logger,
) *BreakGlassService {
	return &BreakGlassService{
		repo:                    repo,
		approvalRepo:            approvalRepo,
		userRepo:                userRepo,
		cache:                   cache,
		logger:                  logger,
		maxDurationMinutes:      60, // Default 1 hour max
		requireManagerApproval:  true,
		requireTicket:           true,
		auditLogRetention:       2555, // 7 years
	}
}

// BreakGlassRequest represents a break glass access request
type BreakGlassRequest struct {
	ID                uuid.UUID          `json:"id"`
	TenantID          uuid.UUID          `json:"tenant_id"`
	RequesterID       uuid.UUID          `json:"requester_id"`
	Reason            string             `json:"reason"`
	BusinessJustification string         `json:"business_justification"`
	Urgency            string            `json:"urgency"` // critical, high, medium, low
	TargetSystems      []string          `json:"target_systems"`
	DurationMinutes    int               `json:"duration_minutes"`
	TicketRef          string            `json:"ticket_ref"`
	ManagerApproval    bool              `json:"manager_approval"`
	ApprovedBy         *uuid.UUID        `json:"approved_by"`
	ApprovedAt         *time.Time        `json:"approved_at"`
	ActivatedAt        *time.Time        `json:"activated_at"`
	ExpiresAt          *time.Time        `json:"expires_at"`
	RevokedAt          *time.Time        `json:"revoked_at"`
	RevokedBy          *uuid.UUID        `json:"revoked_by"`
	RevokeReason       string            `json:"revoke_reason"`
	Status             string            `json:"status"` // pending, approved, active, expired, revoked, denied
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// BreakGlassAuditLog represents an audit log entry for break glass access
type BreakGlassAuditLog struct {
	ID              uuid.UUID          `json:"id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	RequestID       uuid.UUID          `json:"request_id"`
	EventType       string             `json:"event_type"` // requested, approved, activated, revoked, expired, session_created, session_ended
	UserID          uuid.UUID          `json:"user_id"`
	IPAddress       string             `json:"ip_address"`
	UserAgent       string             `json:"user_agent"`
	Timestamp       time.Time          `json:"timestamp"`
	Details         map[string]string  `json:"details"`
}

// RequestBreakGlass creates a new break glass access request
func (s *BreakGlassService) RequestBreakGlass(ctx context.Context, req *BreakGlassRequest, ipAddress, userAgent string) (*BreakGlassRequest, error) {
	// Validate duration
	if req.DurationMinutes <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}
	if req.DurationMinutes > s.maxDurationMinutes {
		return nil, ErrMaxDurationExceeded
	}

	// Validate justification
	if req.BusinessJustification == "" {
		return nil, ErrInvalidJustification
	}

	// Check for existing active break glass access
	active, err := s.getActiveBreakGlass(ctx, req.TenantID, req.RequesterID)
	if err != nil {
		return nil, fmt.Errorf("check active break glass: %w", err)
	}
	if active != nil {
		return nil, ErrBreakGlassActive
	}

	// Check if manager approval is required and provided
	if s.requireManagerApproval && !req.ManagerApproval {
		return nil, fmt.Errorf("manager approval required for break glass access")
	}

	// Check if ticket is required
	if s.requireTicket && req.TicketRef == "" {
		return nil, ErrTicketRequired
	}

	// Create the request
	req.ID = uuid.New()
	req.Status = "pending"
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// Store as access request
	accessReq := &repository.AccessRequest{
		ID:                  req.ID,
		TenantID:            req.TenantID,
		RequesterID:         req.RequesterID,
		Type:                repository.RequestTypeBreakGlass,
		Status:              repository.RequestStatusPending,
		Title:               fmt.Sprintf("Break Glass Access - %s", req.Urgency),
		Description:         req.Reason,
		Reason:              req.Reason,
		BusinessJustification: req.BusinessJustification,
		TicketRef:           req.TicketRef,
		Priority:            req.Urgency,
		RequestedDuration:   req.DurationMinutes,
		Metadata: map[string]string{
			"break_glass":       "true",
			"target_systems":    fmt.Sprintf("%v", req.TargetSystems),
			"manager_approval":  fmt.Sprintf("%t", req.ManagerApproval),
		},
	}

	if err := s.repo.Create(ctx, accessReq); err != nil {
		return nil, fmt.Errorf("create access request: %w", err)
	}

	// Create audit log
	s.createAuditLog(ctx, &BreakGlassAuditLog{
		TenantID:  req.TenantID,
		RequestID: req.ID,
		EventType: "requested",
		UserID:    req.RequesterID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
		Details: map[string]string{
			"urgency":     req.Urgency,
			"duration":    fmt.Sprintf("%d minutes", req.DurationMinutes),
			"ticket_ref":  req.TicketRef,
		},
	})

	s.logger.Warn().
		Str("request_id", req.ID.String()).
		Str("requester_id", req.RequesterID.String()).
		Str("urgency", req.Urgency).
		Int("duration_minutes", req.DurationMinutes).
		Msg("Break glass access requested")

	return req, nil
}

// ApproveBreakGlass approves a break glass request
func (s *BreakGlassService) ApproveBreakGlass(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, approverID uuid.UUID, justification string, ipAddress string) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrBreakGlassNotFound
	}

	if req.Status != repository.RequestStatusPending {
		return errors.New("request is not pending")
	}

	req.Status = repository.RequestStatusApproved
	now := time.Now()
	req.UpdatedAt = now

	if err := s.repo.Update(ctx, req); err != nil {
		return fmt.Errorf("update request: %w", err)
	}

	// Create approval record
	approval := &repository.Approval{
		TenantID:    tenantID,
		RequestID:   requestID,
		WorkflowID:  uuid.Nil, // No workflow for break glass
		StepID:      uuid.Nil,
		ApproverID:  approverID,
		Decision:    "approve",
		Comments:    justification,
		IPAddress:   ipAddress,
	}
	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		s.logger.Error().Err(err).Msg("Failed to create approval record")
	}

	// Create audit log
	s.createAuditLog(ctx, &BreakGlassAuditLog{
		TenantID:  tenantID,
		RequestID: requestID,
		EventType: "approved",
		UserID:    approverID,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
		Details: map[string]string{
			"justification": justification,
		},
	})

	s.logger.Warn().
		Str("request_id", requestID.String()).
		Str("approver_id", approverID.String()).
		Msg("Break glass access approved")

	return nil
}

// ActivateBreakGlass activates an approved break glass request
func (s *BreakGlassService) ActivateBreakGlass(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, ipAddress, userAgent string) (*BreakGlassRequest, error) {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return nil, ErrBreakGlassNotFound
	}

	if req.Status != repository.RequestStatusApproved {
		return nil, errors.New("request is not approved")
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(req.RequestedDuration) * time.Minute)

	req.Status = repository.RequestStatusActive
	req.StartTime = &now
	req.EndTime = &expiresAt
	req.UpdatedAt = now

	if err := s.repo.Update(ctx, req); err != nil {
		return nil, fmt.Errorf("update request: %w", err)
	}

	// Store active session in cache with expiration
	cacheKey := fmt.Sprintf("break_glass:active:%s:%s", tenantID, userID)
	sessionData := map[string]interface{}{
		"request_id":    req.ID,
		"expires_at":    expiresAt,
		"target_systems": req.Metadata,
	}
	data, _ := json.Marshal(sessionData)
	s.cache.Set(ctx, cacheKey, string(data), time.Duration(req.RequestedDuration)*time.Minute)

	// Create audit log
	s.createAuditLog(ctx, &BreakGlassAuditLog{
		TenantID:  tenantID,
		RequestID: requestID,
		EventType: "activated",
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Timestamp: time.Now(),
		Details: map[string]string{
			"expires_at": expiresAt.Format(time.RFC3339),
		},
	})

	s.logger.Warn().
		Str("request_id", requestID.String()).
		Str("user_id", userID.String()).
		Time("expires_at", expiresAt).
		Msg("Break glass access ACTIVATED - enhanced auditing enabled")

	breakGlassReq := &BreakGlassRequest{
		ID:                 req.ID,
		TenantID:           req.TenantID,
		RequesterID:        req.RequesterID,
		Reason:             req.Reason,
		BusinessJustification: req.BusinessJustification,
		DurationMinutes:    req.RequestedDuration,
		TicketRef:          req.TicketRef,
		ActivatedAt:        &now,
		ExpiresAt:          &expiresAt,
		Status:             string(req.Status),
		CreatedAt:          req.CreatedAt,
		UpdatedAt:          req.UpdatedAt,
	}

	return breakGlassReq, nil
}

// RevokeBreakGlass revokes active break glass access
func (s *BreakGlassService) RevokeBreakGlass(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID, revokedBy uuid.UUID, reason, ipAddress string) error {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return ErrBreakGlassNotFound
	}

	if req.Status != repository.RequestStatusActive {
		return errors.New("request is not active")
	}

	now := time.Now()
	req.Status = repository.RequestStatusRevoked
	req.UpdatedAt = now

	if err := s.repo.Update(ctx, req); err != nil {
		return fmt.Errorf("update request: %w", err)
	}

	// Remove from cache
	cacheKey := fmt.Sprintf("break_glass:active:%s:%s", tenantID, req.RequesterID)
	s.cache.Delete(ctx, cacheKey)

	// Create audit log
	s.createAuditLog(ctx, &BreakGlassAuditLog{
		TenantID:  tenantID,
		RequestID: requestID,
		EventType: "revoked",
		UserID:    revokedBy,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
		Details: map[string]string{
			"reason": reason,
		},
	})

	s.logger.Warn().
		Str("request_id", requestID.String()).
		Str("revoked_by", revokedBy.String()).
		Str("reason", reason).
		Msg("Break glass access REVOKED")

	return nil
}

// ListBreakGlassRequests lists break glass requests for a tenant
func (s *BreakGlassService) ListBreakGlassRequests(ctx context.Context, tenantID uuid.UUID, status *string, limit int) ([]BreakGlassRequest, error) {
	requestType := repository.RequestTypeBreakGlass
	filter := repository.AccessRequestFilter{
		TenantID: &tenantID,
		Type:     &requestType,
		Limit:    limit,
	}

	accessRequests, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}

	var breakGlassRequests []BreakGlassRequest
	for _, ar := range accessRequests {
		if status != nil && string(ar.Status) != *status {
			continue
		}

		bgReq := BreakGlassRequest{
			ID:                  ar.ID,
			TenantID:            ar.TenantID,
			RequesterID:         ar.RequesterID,
			Reason:              ar.Reason,
			BusinessJustification: ar.BusinessJustification,
			DurationMinutes:     ar.RequestedDuration,
			TicketRef:           ar.TicketRef,
			ActivatedAt:         ar.StartTime,
			ExpiresAt:           ar.EndTime,
			Status:              string(ar.Status),
			CreatedAt:           ar.CreatedAt,
			UpdatedAt:           ar.UpdatedAt,
		}

		// Parse metadata for additional fields
		if ar.Metadata != nil {
			if urgency, ok := ar.Metadata["urgency"]; ok {
				bgReq.Urgency = urgency
			}
		}

		breakGlassRequests = append(breakGlassRequests, bgReq)
	}

	return breakGlassRequests, nil
}

// GetBreakGlassRequest retrieves a specific break glass request
func (s *BreakGlassService) GetBreakGlassRequest(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID) (*BreakGlassRequest, error) {
	req, err := s.repo.GetByID(ctx, requestID, tenantID)
	if err != nil {
		return nil, ErrBreakGlassNotFound
	}

	bgReq := &BreakGlassRequest{
		ID:                  req.ID,
		TenantID:            req.TenantID,
		RequesterID:         req.RequesterID,
		Reason:              req.Reason,
		BusinessJustification: req.BusinessJustification,
		DurationMinutes:     req.RequestedDuration,
		TicketRef:           req.TicketRef,
		ActivatedAt:         req.StartTime,
		ExpiresAt:           req.EndTime,
		Status:              string(req.Status),
		CreatedAt:           req.CreatedAt,
		UpdatedAt:           req.UpdatedAt,
	}

	return bgReq, nil
}

// GetAuditLog retrieves the audit log for a break glass request
func (s *BreakGlassService) GetAuditLog(ctx context.Context, requestID uuid.UUID, tenantID uuid.UUID) ([]BreakGlassAuditLog, error) {
	// This would query from a dedicated audit log table
	// For now, return approvals as audit entries
	approvals, err := s.approvalRepo.GetByRequestID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("get approvals: %w", err)
	}

	var auditLogs []BreakGlassAuditLog
	for _, a := range approvals {
		auditLogs = append(auditLogs, BreakGlassAuditLog{
			TenantID:  a.TenantID,
			RequestID: a.RequestID,
			EventType: a.Decision,
			UserID:    a.ApproverID,
			IPAddress: a.IPAddress,
			Timestamp: a.CreatedAt,
			Details: map[string]string{
				"comments": a.Comments,
			},
		})
	}

	return auditLogs, nil
}

// IsActiveSession checks if a user has an active break glass session
func (s *BreakGlassService) IsActiveSession(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (bool, error) {
	cacheKey := fmt.Sprintf("break_glass:active:%s:%s", tenantID, userID)
	exists := s.cache.Exists(ctx, cacheKey)
	return exists, nil
}

// getActiveBreakGlass retrieves active break glass access for a user
func (s *BreakGlassService) getActiveBreakGlass(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (*repository.AccessRequest, error) {
	requests, err := s.repo.GetActiveRequests(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	for _, req := range requests {
		if req.RequesterID == userID && req.Type == repository.RequestTypeBreakGlass {
			return &req, nil
		}
	}

	return nil, nil
}

// createAuditLog creates an audit log entry
func (s *BreakGlassService) createAuditLog(ctx context.Context, log *BreakGlassAuditLog) error {
	log.ID = uuid.New()
	// In production, this would write to a dedicated audit log table
	// that is append-only and tamper-evident
	s.logger.Info().
		Str("audit_id", log.ID.String()).
		Str("event_type", log.EventType).
		Str("request_id", log.RequestID.String()).
		Str("user_id", log.UserID.String()).
		Interface("details", log.Details).
		Msg("Break glass audit log entry created")
	return nil
}
