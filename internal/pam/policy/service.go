package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// Service handles policy business logic
type Service struct {
	repo          RepositoryInterface
	cache         CacheInterface
	evaluator     *Evaluator
	eventBus      *events.EventBus
	logger        zerolog.Logger
	denyByDefault bool
}

// NewService creates a new policy service with secure default-deny behavior
func NewService(repo RepositoryInterface, c *cache.Cache, eventBus *events.EventBus, logger zerolog.Logger) *Service {
	return NewServiceWithConfig(repo, c, eventBus, logger, true)
}

// NewServiceWithConfig creates a new policy service with specified default behavior
func NewServiceWithConfig(repo RepositoryInterface, c *cache.Cache, eventBus *events.EventBus, logger zerolog.Logger, denyByDefault bool) *Service {
	policyCache := NewPolicyCache(c, logger)

	return &Service{
		repo:          repo,
		cache:         policyCache,
		evaluator:     NewEvaluatorWithConfig(logger, denyByDefault),
		eventBus:      eventBus,
		logger:        logger,
		denyByDefault: denyByDefault,
	}
}

// CreatePolicy creates a new policy
func (s *Service) CreatePolicy(ctx context.Context, policy *Policy, createdBy uuid.UUID) error {
	// Validate policy
	if err := s.evaluator.ValidatePolicy(policy); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Set audit fields
	policy.CreatedBy = createdBy
	policy.UpdatedBy = createdBy

	// Create in repository
	if err := s.repo.Create(ctx, policy); err != nil {
		return fmt.Errorf("service.CreatePolicy: %w", err)
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnPolicyChange(ctx, policy.TenantID, policy.ID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate cache after policy creation")
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "policy.created",
		TenantID: policy.TenantID.String(),
		ActorID:  createdBy.String(),
		Action:   "create",
		Resource: "policy",
		Data: map[string]interface{}{
			"policy_id": policy.ID.String(),
			"name":      policy.Name,
			"type":      string(policy.Type),
			"effect":    string(policy.Effect),
		},
	})

	s.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("name", policy.Name).
		Str("tenant_id", policy.TenantID.String()).
		Msg("Policy created")

	return nil
}

// GetPolicy retrieves a policy by ID
func (s *Service) GetPolicy(ctx context.Context, id, tenantID uuid.UUID) (*Policy, error) {
	// Try cache first
	policy, err := s.cache.GetPolicy(ctx, id)
	if err == nil {
		return policy, nil
	}

	// Fall back to repository
	policy, err = s.repo.GetByIDForTenant(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("service.GetPolicy: %w", err)
	}

	// Update cache
	_ = s.cache.SetPolicy(ctx, policy, 0)

	return policy, nil
}

// ListPolicies retrieves policies with filtering
func (s *Service) ListPolicies(ctx context.Context, tenantID uuid.UUID, filter PolicyFilter, limit, offset int) ([]Policy, int, error) {
	// Generate filter key for cache
	filterKey := s.generateFilterKey(filter)

	// Try cache first
	policies, total, err := s.cache.GetPolicyList(ctx, tenantID, filterKey)
	if err == nil {
		return policies, total, nil
	}

	// Fall back to repository
	policies, total, err = s.repo.List(ctx, tenantID, filter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("service.ListPolicies: %w", err)
	}

	// Update cache
	_ = s.cache.SetPolicyList(ctx, tenantID, filterKey, policies, total, 0)

	return policies, total, nil
}

// UpdatePolicy updates an existing policy
func (s *Service) UpdatePolicy(ctx context.Context, policy *Policy, updatedBy uuid.UUID) error {
	// Validate policy
	if err := s.evaluator.ValidatePolicy(policy); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Set audit field
	policy.UpdatedBy = updatedBy

	// Get original for comparison
	original, err := s.repo.GetByIDForTenant(ctx, policy.ID, policy.TenantID)
	if err != nil {
		return fmt.Errorf("service.UpdatePolicy.GetOriginal: %w", err)
	}

	// Check if system policy
	if original.SystemPolicy {
		return fmt.Errorf("cannot modify system policy")
	}

	// Update in repository
	if err := s.repo.Update(ctx, policy); err != nil {
		return fmt.Errorf("service.UpdatePolicy: %w", err)
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnPolicyChange(ctx, policy.TenantID, policy.ID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate cache after policy update")
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "policy.updated",
		TenantID: policy.TenantID.String(),
		ActorID:  updatedBy.String(),
		Action:   "update",
		Resource: "policy",
		Data: map[string]interface{}{
			"policy_id": policy.ID.String(),
			"name":      policy.Name,
			"version":   policy.Version,
		},
	})

	s.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("name", policy.Name).
		Int("version", policy.Version).
		Msg("Policy updated")

	return nil
}

// DeletePolicy soft deletes a policy
func (s *Service) DeletePolicy(ctx context.Context, id, tenantID uuid.UUID, deletedBy uuid.UUID) error {
	// Check if policy exists and is not a system policy
	policy, err := s.repo.GetByIDForTenant(ctx, id, tenantID)
	if err != nil {
		return fmt.Errorf("service.DeletePolicy.Get: %w", err)
	}

	if policy.SystemPolicy {
		return fmt.Errorf("cannot delete system policy")
	}

	// Delete in repository
	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return fmt.Errorf("service.DeletePolicy: %w", err)
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnPolicyChange(ctx, tenantID, id); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate cache after policy deletion")
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "policy.deleted",
		TenantID: tenantID.String(),
		ActorID:  deletedBy.String(),
		Action:   "delete",
		Resource: "policy",
		Data: map[string]interface{}{
			"policy_id": id.String(),
			"name":      policy.Name,
		},
	})

	s.logger.Info().
		Str("policy_id", id.String()).
		Str("name", policy.Name).
		Msg("Policy deleted")

	return nil
}

// EnablePolicy enables a policy
func (s *Service) EnablePolicy(ctx context.Context, id, tenantID uuid.UUID, enabledBy uuid.UUID) error {
	if err := s.repo.Enable(ctx, id, tenantID); err != nil {
		return fmt.Errorf("service.EnablePolicy: %w", err)
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnPolicyChange(ctx, tenantID, id); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate cache after policy enable")
	}

	s.logger.Info().
		Str("policy_id", id.String()).
		Str("enabled_by", enabledBy.String()).
		Msg("Policy enabled")

	return nil
}

// DisablePolicy disables a policy
func (s *Service) DisablePolicy(ctx context.Context, id, tenantID uuid.UUID, disabledBy uuid.UUID) error {
	if err := s.repo.Disable(ctx, id, tenantID); err != nil {
		return fmt.Errorf("service.DisablePolicy: %w", err)
	}

	// Invalidate cache
	if err := s.cache.InvalidateOnPolicyChange(ctx, tenantID, id); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to invalidate cache after policy disable")
	}

	s.logger.Info().
		Str("policy_id", id.String()).
		Str("disabled_by", disabledBy.String()).
		Msg("Policy disabled")

	return nil
}

// Evaluate evaluates policies for a request
func (s *Service) Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResponse, error) {
	startTime := time.Now()

	// Generate cache key
	cacheKey := GenerateEvalCacheKey(req)

	// Try cache first for short-lived evaluations
	cachedResult, err := s.cache.GetEvalResult(ctx, cacheKey)
	if err == nil {
		return cachedResult, nil
	}

	// Get applicable policies
	policies, err := s.getApplicablePolicies(ctx, req.TenantID, req.UserID, req.UserRoles, req.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("service.Evaluate.GetApplicable: %w", err)
	}

	// Evaluate policies
	result, err := s.evaluator.Evaluate(ctx, req, policies)
	if err != nil {
		return nil, fmt.Errorf("service.Evaluate: %w", err)
	}

	// Cache the result for a short time
	_ = s.cache.SetEvalResult(ctx, cacheKey, result, 0)

	// Log evaluation
	s.logEvaluation(ctx, req, result, policies, startTime)

	return result, nil
}

// EvaluateCommand checks if a command is allowed
func (s *Service) EvaluateCommand(ctx context.Context, tenantID, userID uuid.UUID, command string, sessionID *uuid.UUID) (bool, string, error) {
	startTime := time.Now()

	// Get command filter policies for the tenant
	policies, err := s.repo.GetActiveForTenant(ctx, tenantID)
	if err != nil {
		return false, "", fmt.Errorf("service.EvaluateCommand.GetPolicies: %w", err)
	}

	// Collect all command filter patterns
	var allPatterns []CommandFilterPattern
	for _, policy := range policies {
		if policy.Type == PolicyTypeCommandFilter {
			patterns, err := s.repo.GetCommandFilterPatterns(ctx, policy.ID)
			if err != nil {
				s.logger.Error().Err(err).Str("policy_id", policy.ID.String()).Msg("Failed to get command patterns")
				continue
			}
			allPatterns = append(allPatterns, patterns...)
		}
	}

	// Evaluate command
	allowed, reason, err := s.evaluator.IsCommandAllowed(ctx, command, allPatterns)
	if err != nil {
		return false, "", fmt.Errorf("service.EvaluateCommand: %w", err)
	}

	// Log evaluation
	logEntry := &PolicyEvalLog{
		TenantID:             tenantID,
		UserID:               userID,
		TargetType:           "command",
		Action:               "execute",
		Result:               EvaluationResultAllow,
		ClientIP:             "", // Would be passed in context
		SessionID:            sessionID,
		EvaluationDurationMs: intPtr(int(time.Since(startTime).Milliseconds())),
	}

	if !allowed {
		logEntry.Result = EvaluationResultDeny
		logEntry.DenialReasons = []string{reason}
	}

	_ = s.repo.CreateEvaluationLog(ctx, logEntry)

	return allowed, reason, nil
}

// getApplicablePolicies retrieves applicable policies with cache-aside
func (s *Service) getApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]Policy, error) {
	// Try cache first
	policies, err := s.cache.GetApplicablePolicies(ctx, tenantID, userID, resourceID)
	if err == nil {
		return policies, nil
	}

	// Fall back to repository
	policies, err = s.repo.GetApplicablePolicies(ctx, tenantID, userID, userRoles, resourceID)
	if err != nil {
		return nil, err
	}

	// Update cache
	_ = s.cache.SetApplicablePolicies(ctx, tenantID, userID, resourceID, policies, 0)

	return policies, nil
}

// logEvaluation logs a policy evaluation to the repository
func (s *Service) logEvaluation(ctx context.Context, req EvaluationRequest, result *EvaluationResponse, policies []Policy, startTime time.Time) {
	// Find the policy that caused the decision
	var policyID *uuid.UUID
	for _, matched := range result.MatchedPolicies {
		if matched.Effect == PolicyEffectDeny || result.Effect == EvaluationResultAllow {
			policyID = &matched.PolicyID
			break
		}
	}

	logEntry := &PolicyEvalLog{
		TenantID:             req.TenantID,
		PolicyID:             policyID,
		UserID:               req.UserID,
		TargetType:           req.ResourceType,
		TargetID:             &req.ResourceID,
		Action:               req.Action,
		Result:               result.Effect,
		DenialReasons:        result.DenialReasons,
		MatchedRules:         []MatchedRule{},
		ClientIP:             req.ClientIP,
		UserAgent:            req.UserAgent,
		SessionID:            req.SessionID,
		EvaluationDurationMs: intPtr(int(time.Since(startTime).Milliseconds())),
	}

	// Extract matched rules
	for _, mp := range result.MatchedPolicies {
		logEntry.MatchedRules = append(logEntry.MatchedRules, mp.Rules...)
	}

	_ = s.repo.CreateEvaluationLog(ctx, logEntry)
}

// RequestApproval creates an approval request
func (s *Service) RequestApproval(ctx context.Context, req *ApprovalRequest) error {
	// Set expiry if not set
	if req.ExpiresAt.IsZero() {
		// Default 1 hour timeout
		req.ExpiresAt = time.Now().Add(60 * time.Minute)
	}

	// Create in repository
	if err := s.repo.CreateApprovalRequest(ctx, req); err != nil {
		return fmt.Errorf("service.RequestApproval: %w", err)
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "approval.requested",
		TenantID: req.TenantID.String(),
		ActorID:  req.RequesterID.String(),
		Action:   "request",
		Resource: "approval",
		Data: map[string]interface{}{
			"request_id":  req.ID.String(),
			"target_type": req.TargetType,
			"target_id":   req.TargetID.String(),
		},
	})

	s.logger.Info().
		Str("request_id", req.ID.String()).
		Str("requester_id", req.RequesterID.String()).
		Str("target_type", req.TargetType).
		Msg("Approval request created")

	return nil
}

// ApproveRequest approves an approval request
func (s *Service) ApproveRequest(ctx context.Context, requestID, approverID uuid.UUID) error {
	req, err := s.repo.GetApprovalRequestByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("service.ApproveRequest.Get: %w", err)
	}

	if err := s.repo.UpdateApprovalRequestStatus(ctx, requestID, ApprovalStatusApproved, &approverID, ""); err != nil {
		return fmt.Errorf("service.ApproveRequest.Update: %w", err)
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "approval.approved",
		TenantID: req.TenantID.String(),
		ActorID:  approverID.String(),
		Action:   "approve",
		Resource: "approval",
		Data: map[string]interface{}{
			"request_id": requestID.String(),
		},
	})

	s.logger.Info().
		Str("request_id", requestID.String()).
		Str("approver_id", approverID.String()).
		Msg("Approval request approved")

	return nil
}

// DenyRequest denies an approval request
func (s *Service) DenyRequest(ctx context.Context, requestID, approverID uuid.UUID, reason string) error {
	req, err := s.repo.GetApprovalRequestByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("service.DenyRequest.Get: %w", err)
	}

	if err := s.repo.UpdateApprovalRequestStatus(ctx, requestID, ApprovalStatusDenied, &approverID, reason); err != nil {
		return fmt.Errorf("service.DenyRequest.Update: %w", err)
	}

	// Publish event
	_ = s.eventBus.Publish(ctx, events.Event{
		Type:     "approval.denied",
		TenantID: req.TenantID.String(),
		ActorID:  approverID.String(),
		Action:   "deny",
		Resource: "approval",
		Data: map[string]interface{}{
			"request_id": requestID.String(),
			"reason":     reason,
		},
	})

	s.logger.Info().
		Str("request_id", requestID.String()).
		Str("approver_id", approverID.String()).
		Str("reason", reason).
		Msg("Approval request denied")

	return nil
}

// GetPendingApprovals retrieves pending approval requests
func (s *Service) GetPendingApprovals(ctx context.Context, tenantID, approverID uuid.UUID) ([]ApprovalRequest, error) {
	return s.repo.GetPendingApprovalRequests(ctx, tenantID, &approverID)
}

// GetEvaluationLogs retrieves evaluation logs
func (s *Service) GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]PolicyEvalLog, int, error) {
	return s.repo.GetEvaluationLogs(ctx, tenantID, userID, limit, offset)
}

// CreatePolicyTemplate creates a policy template
func (s *Service) CreatePolicyTemplate(ctx context.Context, template *PolicyTemplate) error {
	if err := s.repo.CreatePolicyTemplate(ctx, template); err != nil {
		return fmt.Errorf("service.CreatePolicyTemplate: %w", err)
	}

	s.logger.Info().
		Str("template_id", template.ID.String()).
		Str("name", template.Name).
		Msg("Policy template created")

	return nil
}

// GetPolicyTemplates retrieves policy templates
func (s *Service) GetPolicyTemplates(ctx context.Context, category *string) ([]PolicyTemplate, error) {
	return s.repo.GetPolicyTemplates(ctx, category)
}

// AddCommandFilterPattern adds a command filter pattern to a policy
func (s *Service) AddCommandFilterPattern(ctx context.Context, pattern *CommandFilterPattern) error {
	// Generate hash for pattern
	pattern.CompiledHash = s.hashPattern(pattern.Pattern)

	if err := s.repo.CreateCommandFilterPattern(ctx, pattern); err != nil {
		return fmt.Errorf("service.AddCommandFilterPattern: %w", err)
	}

	// Invalidate cache for the policy
	_ = s.cache.DeletePolicy(ctx, pattern.PolicyID)

	s.logger.Info().
		Str("policy_id", pattern.PolicyID.String()).
		Str("pattern", pattern.Pattern).
		Msg("Command filter pattern added")

	return nil
}

// WarmupCache warms up the cache with active policies
func (s *Service) WarmupCache(ctx context.Context, tenantID uuid.UUID) error {
	policies, err := s.repo.GetActiveForTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("service.WarmupCache: %w", err)
	}

	return s.cache.Warmup(ctx, tenantID, policies)
}

// hashPattern creates a hash for a command filter pattern
func (s *Service) hashPattern(pattern string) string {
	hash := sha256.Sum256([]byte(pattern))
	return hex.EncodeToString(hash[:])
}

// generateFilterKey generates a cache key from filter parameters
func (s *Service) generateFilterKey(filter PolicyFilter) string {
	key := ""
	if filter.Type != nil {
		key += string(*filter.Type) + ":"
	}
	if filter.Effect != nil {
		key += string(*filter.Effect) + ":"
	}
	if filter.Enabled != nil {
		key += fmt.Sprintf("enabled_%v:", *filter.Enabled)
	}
	if filter.Search != "" {
		key += "search:" + filter.Search + ":"
	}
	for _, tag := range filter.Tags {
		key += "tag:" + tag + ":"
	}
	if key == "" {
		return "all"
	}
	return key
}

// GetRepository returns the underlying repository (for handlers that need direct access)
func (s *Service) GetRepository() RepositoryInterface {
	return s.repo
}

// GetEvaluator returns the underlying evaluator (for testing)
func (s *Service) GetEvaluator() *Evaluator {
	return s.evaluator
}

func intPtr(i int) *int {
	return &i
}
