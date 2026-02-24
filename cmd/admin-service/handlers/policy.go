package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/policy"
	"github.com/rs/zerolog"
)

// PolicyHandler handles policy-related HTTP requests
type PolicyHandler struct {
	service *policy.Service
	logger  zerolog.Logger
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(service *policy.Service, logger zerolog.Logger) *PolicyHandler {
	return &PolicyHandler{
		service: service,
		logger:  logger,
	}
}

// CreatePolicyRequest represents a request to create a policy
type CreatePolicyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        policy.PolicyType      `json:"type" binding:"required"`
	Effect      policy.PolicyEffect    `json:"effect" binding:"required,oneof=allow deny"`
	Priority    int                    `json:"priority"`
	UserIDs     []uuid.UUID            `json:"user_ids"`
	RoleIDs     []uuid.UUID            `json:"role_ids"`
	GroupIDs    []uuid.UUID            `json:"group_ids"`
	TargetIDs   []uuid.UUID            `json:"target_ids"`
	CredentialIDs []uuid.UUID          `json:"credential_ids"`
	Rules       []policy.Rule          `json:"rules"`
	MaxSessionDurationSeconds *int     `json:"max_session_duration_seconds"`
	SessionExtensionAllowed   bool     `json:"session_extension_allowed"`
	MaxExtensions             *int     `json:"max_extensions"`
	MFARequired               bool     `json:"mfa_required"`
	MFAMethods                []policy.MFAMethod `json:"mfa_methods"`
	ApprovalRequired          bool     `json:"approval_required"`
	ApprovalApprovers         []uuid.UUID `json:"approval_approvers"`
	ApprovalTimeoutMinutes    int      `json:"approval_timeout_minutes"`
	RecordingRequired         bool                `json:"recording_required"`
	RecordingMode             policy.RecordingMode `json:"recording_mode"`
	Enabled                   bool     `json:"enabled"`
	Metadata                  map[string]interface{} `json:"metadata"`
	Tags                      []string `json:"tags"`
}

// UpdatePolicyRequest represents a request to update a policy
type UpdatePolicyRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        *policy.PolicyType     `json:"type"`
	Effect      *policy.PolicyEffect   `json:"type"`
	Priority    *int                   `json:"priority"`
	UserIDs     []uuid.UUID            `json:"user_ids"`
	RoleIDs     []uuid.UUID            `json:"role_ids"`
	GroupIDs    []uuid.UUID            `json:"group_ids"`
	TargetIDs   []uuid.UUID            `json:"target_ids"`
	CredentialIDs []uuid.UUID          `json:"credential_ids"`
	Rules       []policy.Rule          `json:"rules"`
	MaxSessionDurationSeconds *int     `json:"max_session_duration_seconds"`
	SessionExtensionAllowed   *bool    `json:"session_extension_allowed"`
	MaxExtensions             *int     `json:"max_extensions"`
	MFARequired               *bool    `json:"mfa_required"`
	MFAMethods                []policy.MFAMethod `json:"mfa_methods"`
	ApprovalRequired          *bool   `json:"approval_required"`
	ApprovalApprovers         []uuid.UUID `json:"approval_approvers"`
	ApprovalTimeoutMinutes    *int    `json:"approval_timeout_minutes"`
	RecordingRequired         *bool               `json:"recording_required"`
	RecordingMode             *policy.RecordingMode `json:"recording_mode"`
	Enabled                   *bool   `json:"enabled"`
	Metadata                  map[string]interface{} `json:"metadata"`
	Tags                      []string `json:"tags"`
}

// EvaluatePolicyRequest represents a request to evaluate policies
type EvaluatePolicyRequest struct {
	UserID        uuid.UUID                 `json:"user_id" binding:"required"`
	UserRoles     []uuid.UUID               `json:"user_roles"`
	UserGroups    []uuid.UUID               `json:"user_groups"`
	Action        string                    `json:"action" binding:"required"`
	ResourceType  string                    `json:"resource_type" binding:"required"`
	ResourceID    uuid.UUID                 `json:"resource_id" binding:"required"`
	ClientIP      string                    `json:"client_ip"`
	UserAgent     string                    `json:"user_agent"`
	SessionID     *uuid.UUID                `json:"session_id"`
	Time          time.Time                 `json:"time"`
	MFAVerified   bool                      `json:"mfa_verified"`
	MFAMethod     policy.MFAMethod          `json:"mfa_method"`
	DeviceTrust   int                       `json:"device_trust"`
	Context       map[string]interface{}    `json:"context"`
}

// ApprovalActionRequest represents a request to approve/deny an approval
type ApprovalActionRequest struct {
	Action string `json:"action" binding:"required,oneof=approve deny"`
	Reason string `json:"reason"`
}

// CommandFilterRequest represents a request to add a command filter
type CommandFilterRequest struct {
	PolicyID    uuid.UUID `json:"policy_id" binding:"required"`
	Pattern     string    `json:"pattern" binding:"required"`
	IsWhitelist bool      `json:"is_whitelist"`
	Description string    `json:"description"`
}

// ListPolicies returns a handler for listing policies
func (h *PolicyHandler) ListPolicies() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		tenantIDUUID, err := uuid.Parse(tenantID.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
			return
		}

		// Parse query parameters
		filter := h.parsePolicyFilter(c)
		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		policies, total, err := h.service.ListPolicies(c.Request.Context(), tenantIDUUID, filter, limit, offset)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to list policies")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list policies"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"policies": policies,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

// CreatePolicy returns a handler for creating a policy
func (h *PolicyHandler) CreatePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		var req CreatePolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		policyObj := h.buildPolicyFromRequest(req, tenantID.(string), userID.(string))

		if err := h.service.CreatePolicy(c.Request.Context(), policyObj, uuid.MustParse(userID.(string))); err != nil {
			h.logger.Error().Err(err).Msg("Failed to create policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create policy"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"policy": policyObj})
	}
}

// GetPolicy returns a handler for getting a policy
func (h *PolicyHandler) GetPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid policy ID"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		policyObj, err := h.service.GetPolicy(c.Request.Context(), id, uuid.MustParse(tenantID.(string)))
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get policy")
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Policy not found"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"policy": policyObj})
	}
}

// UpdatePolicy returns a handler for updating a policy
func (h *PolicyHandler) UpdatePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid policy ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		var req UpdatePolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Get existing policy
		policyObj, err := h.service.GetPolicy(c.Request.Context(), id, uuid.MustParse(tenantID.(string)))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Policy not found"}})
			return
		}

		// Update fields from request
		h.updatePolicyFromRequest(policyObj, req)

		if err := h.service.UpdatePolicy(c.Request.Context(), policyObj, uuid.MustParse(userID.(string))); err != nil {
			h.logger.Error().Err(err).Msg("Failed to update policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update policy"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"policy": policyObj})
	}
}

// DeletePolicy returns a handler for deleting a policy
func (h *PolicyHandler) DeletePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid policy ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		if err := h.service.DeletePolicy(c.Request.Context(), id, uuid.MustParse(tenantID.(string)), uuid.MustParse(userID.(string))); err != nil {
			h.logger.Error().Err(err).Msg("Failed to delete policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete policy"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Policy deleted"})
	}
}

// EnablePolicy returns a handler for enabling a policy
func (h *PolicyHandler) EnablePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid policy ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		if err := h.service.EnablePolicy(c.Request.Context(), id, uuid.MustParse(tenantID.(string)), uuid.MustParse(userID.(string))); err != nil {
			h.logger.Error().Err(err).Msg("Failed to enable policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to enable policy"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Policy enabled"})
	}
}

// DisablePolicy returns a handler for disabling a policy
func (h *PolicyHandler) DisablePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid policy ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		if err := h.service.DisablePolicy(c.Request.Context(), id, uuid.MustParse(tenantID.(string)), uuid.MustParse(userID.(string))); err != nil {
			h.logger.Error().Err(err).Msg("Failed to disable policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to disable policy"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Policy disabled"})
	}
}

// EvaluatePolicy returns a handler for evaluating policies
func (h *PolicyHandler) EvaluatePolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		var req EvaluatePolicyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		// Set default time if not provided
		if req.Time.IsZero() {
			req.Time = time.Now()
		}

		// Get user roles from context if not provided
		if len(req.UserRoles) == 0 {
			if roles, exists := c.Get("user_roles"); exists {
				req.UserRoles = roles.([]uuid.UUID)
			}
		}

		// Get client IP from context if not provided
		if req.ClientIP == "" {
			if clientIP, exists := c.Get("client_ip"); exists {
				req.ClientIP = clientIP.(string)
			}
		}

		// Build evaluation request
		evalReq := policy.EvaluationRequest{
			TenantID:     uuid.MustParse(tenantID.(string)),
			UserID:       req.UserID,
			UserRoles:    req.UserRoles,
			UserGroups:   req.UserGroups,
			Action:       req.Action,
			ResourceType: req.ResourceType,
			ResourceID:   req.ResourceID,
			ClientIP:     req.ClientIP,
			UserAgent:    req.UserAgent,
			SessionID:    req.SessionID,
			Time:         req.Time,
			MFAVerified:  req.MFAVerified,
			MFAMethod:    req.MFAMethod,
			DeviceTrust:  req.DeviceTrust,
			Context:      req.Context,
		}

		result, err := h.service.Evaluate(c.Request.Context(), evalReq)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to evaluate policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to evaluate policy"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": result})
	}
}

// GetEvaluationLogs returns a handler for getting evaluation logs
func (h *PolicyHandler) GetEvaluationLogs() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		var userID *uuid.UUID
		if userIDStr := c.Query("user_id"); userIDStr != "" {
			if parsedID, err := uuid.Parse(userIDStr); err == nil {
				userID = &parsedID
			}
		}

		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		logs, total, err := h.service.GetEvaluationLogs(c.Request.Context(), uuid.MustParse(tenantID.(string)), userID, limit, offset)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get evaluation logs")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get evaluation logs"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"logs":   logs,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

// GetPolicyTemplates returns a handler for getting policy templates
func (h *PolicyHandler) GetPolicyTemplates() gin.HandlerFunc {
	return func(c *gin.Context) {
		var category *string
		if cat := c.Query("category"); cat != "" {
			category = &cat
		}

		templates, err := h.service.GetPolicyTemplates(c.Request.Context(), category)
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get policy templates")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get policy templates"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"templates": templates})
	}
}

// CreateApprovalRequest returns a handler for creating an approval request
func (h *PolicyHandler) CreateApprovalRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		var req policy.ApprovalRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		req.RequesterID = uuid.MustParse(userID.(string))
		req.TenantID = uuid.MustParse(tenantID.(string))

		if err := h.service.RequestApproval(c.Request.Context(), &req); err != nil {
			h.logger.Error().Err(err).Msg("Failed to create approval request")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create approval request"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"request": req})
	}
}

// GetPendingApprovals returns a handler for getting pending approvals
func (h *PolicyHandler) GetPendingApprovals() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		tenantID, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Tenant ID required"}})
			return
		}

		requests, err := h.service.GetPendingApprovals(c.Request.Context(), uuid.MustParse(tenantID.(string)), uuid.MustParse(userID.(string)))
		if err != nil {
			h.logger.Error().Err(err).Msg("Failed to get pending approvals")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get pending approvals"}})
			return
		}

		c.JSON(http.StatusOK, gin.H{"requests": requests})
	}
}

// ProcessApprovalRequest returns a handler for approving/denying an approval
func (h *PolicyHandler) ProcessApprovalRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid request ID"}})
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User ID required"}})
			return
		}

		var req ApprovalActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		approverID := uuid.MustParse(userID.(string))

		if req.Action == "approve" {
			if err := h.service.ApproveRequest(c.Request.Context(), id, approverID); err != nil {
				h.logger.Error().Err(err).Msg("Failed to approve request")
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to approve request"}})
				return
			}
		} else {
			if err := h.service.DenyRequest(c.Request.Context(), id, approverID, req.Reason); err != nil {
				h.logger.Error().Err(err).Msg("Failed to deny request")
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to deny request"}})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Request processed"})
	}
}

// AddCommandFilter returns a handler for adding a command filter
func (h *PolicyHandler) AddCommandFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CommandFilterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
			return
		}

		pattern := &policy.CommandFilterPattern{
			PolicyID:    req.PolicyID,
			Pattern:     req.Pattern,
			IsWhitelist: req.IsWhitelist,
			Description: req.Description,
		}

		if err := h.service.AddCommandFilterPattern(c.Request.Context(), pattern); err != nil {
			h.logger.Error().Err(err).Msg("Failed to add command filter")
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to add command filter"}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"pattern": pattern})
	}
}

// Helper methods

func (h *PolicyHandler) parsePolicyFilter(c *gin.Context) policy.PolicyFilter {
	filter := policy.PolicyFilter{}

	if typeStr := c.Query("type"); typeStr != "" {
		if pt, err := policy.PolicyTypeFromString(typeStr); err == nil {
			filter.Type = &pt
		}
	}

	if effectStr := c.Query("effect"); effectStr != "" {
		if pe, err := policy.EffectFromString(effectStr); err == nil {
			filter.Effect = &pe
		}
	}

	if enabledStr := c.Query("enabled"); enabledStr != "" {
		if enabled, err := strconv.ParseBool(enabledStr); err == nil {
			filter.Enabled = &enabled
		}
	}

	filter.Search = c.Query("search")

	if tags := c.QueryArray("tags"); len(tags) > 0 {
		filter.Tags = tags
	}

	return filter
}

func (h *PolicyHandler) buildPolicyFromRequest(req CreatePolicyRequest, tenantID, userID string) *policy.Policy {
	return &policy.Policy{
		Name:                      req.Name,
		Description:               req.Description,
		Type:                      req.Type,
		Effect:                    req.Effect,
		Priority:                  req.Priority,
		TenantID:                  uuid.MustParse(tenantID),
		UserIDs:                   req.UserIDs,
		RoleIDs:                   req.RoleIDs,
		GroupIDs:                  req.GroupIDs,
		TargetIDs:                 req.TargetIDs,
		CredentialIDs:             req.CredentialIDs,
		Rules:                     req.Rules,
		MaxSessionDurationSeconds: req.MaxSessionDurationSeconds,
		SessionExtensionAllowed:   req.SessionExtensionAllowed,
		MaxExtensions:             req.MaxExtensions,
		MFARequired:               req.MFARequired,
		MFAMethods:                req.MFAMethods,
		ApprovalRequired:          req.ApprovalRequired,
		ApprovalApprovers:         req.ApprovalApprovers,
		ApprovalTimeoutMinutes:    req.ApprovalTimeoutMinutes,
		RecordingRequired:         req.RecordingRequired,
		RecordingMode:             req.RecordingMode,
		Enabled:                   req.Enabled,
		Metadata:                  req.Metadata,
		Tags:                      req.Tags,
		CreatedBy:                 uuid.MustParse(userID),
		UpdatedBy:                 uuid.MustParse(userID),
	}
}

func (h *PolicyHandler) updatePolicyFromRequest(p *policy.Policy, req UpdatePolicyRequest) {
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Type != nil {
		p.Type = *req.Type
	}
	if req.Effect != nil {
		p.Effect = *req.Effect
	}
	if req.Priority != nil {
		p.Priority = *req.Priority
	}
	if req.UserIDs != nil {
		p.UserIDs = req.UserIDs
	}
	if req.RoleIDs != nil {
		p.RoleIDs = req.RoleIDs
	}
	if req.GroupIDs != nil {
		p.GroupIDs = req.GroupIDs
	}
	if req.TargetIDs != nil {
		p.TargetIDs = req.TargetIDs
	}
	if req.CredentialIDs != nil {
		p.CredentialIDs = req.CredentialIDs
	}
	if req.Rules != nil {
		p.Rules = req.Rules
	}
	if req.MaxSessionDurationSeconds != nil {
		p.MaxSessionDurationSeconds = req.MaxSessionDurationSeconds
	}
	if req.SessionExtensionAllowed != nil {
		p.SessionExtensionAllowed = *req.SessionExtensionAllowed
	}
	if req.MaxExtensions != nil {
		p.MaxExtensions = req.MaxExtensions
	}
	if req.MFARequired != nil {
		p.MFARequired = *req.MFARequired
	}
	if req.MFAMethods != nil {
		p.MFAMethods = req.MFAMethods
	}
	if req.ApprovalRequired != nil {
		p.ApprovalRequired = *req.ApprovalRequired
	}
	if req.ApprovalApprovers != nil {
		p.ApprovalApprovers = req.ApprovalApprovers
	}
	if req.ApprovalTimeoutMinutes != nil {
		p.ApprovalTimeoutMinutes = *req.ApprovalTimeoutMinutes
	}
	if req.RecordingRequired != nil {
		p.RecordingRequired = *req.RecordingRequired
	}
	if req.RecordingMode != nil {
		p.RecordingMode = *req.RecordingMode
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.Metadata != nil {
		p.Metadata = req.Metadata
	}
	if req.Tags != nil {
		p.Tags = req.Tags
	}
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	var intVal int
	if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
		return intVal
	}
	return defaultVal
}
