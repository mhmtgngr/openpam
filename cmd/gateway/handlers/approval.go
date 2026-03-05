package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/approval"
	"github.com/rs/zerolog"
)

// ApprovalHandler handles approval endpoints
type ApprovalHandler struct {
	service *approval.WorkflowService
	logger  zerolog.Logger
}

// NewApprovalHandler creates a new approval handler
func NewApprovalHandler(service *approval.WorkflowService, logger zerolog.Logger) *ApprovalHandler {
	return &ApprovalHandler{
		service: service,
		logger:  logger,
	}
}

// CreateRequestRequest represents an approval request creation
type CreateRequestRequest struct {
	Type          string `json:"type" binding:"required,oneof=credential_access session_access privilege_escalation"`
	TargetID      string `json:"target_id"`
	CredentialID  string `json:"credential_id"`
	Justification string `json:"justification" binding:"required"`
	Duration      int    `json:"duration_minutes" binding:"required,min=1"`
	Priority      string `json:"priority" binding:"omitempty,oneof=low normal high emergency"`
	TicketRef     string `json:"ticket_ref"`
}

// ApproveRequestRequest represents an approval decision
type ApproveRequestRequest struct {
	Decision string `json:"decision" binding:"required,oneof=approve deny"`
	Comments string `json:"comments"`
}

// List returns a paginated list of approval requests
func (h *ApprovalHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	_ = tenantID // Will be used when implementing list query

	// Parse query parameters
	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Use the service's repository to list requests
	// For now, return empty list since we need direct repo access
	c.JSON(http.StatusOK, gin.H{
		"requests": []interface{}{},
		"total":    0,
		"limit":    limit,
		"offset":   offset,
	})
}

// Get retrieves a single approval request by ID
func (h *ApprovalHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST_ID",
				"message": "Invalid request ID",
			},
		})
		return
	}

	// Get request - need to access repo directly
	// For now, return placeholder
	c.JSON(http.StatusOK, gin.H{
		"request": gin.H{
			"id":     id,
			"status": "pending",
		},
	})
}

// Create creates a new approval request
func (h *ApprovalHandler) Create(c *gin.Context) {
	var req CreateRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	approvalReq := &approval.Request{
		Type:          approval.RequestType(req.Type),
		UserID:        userIDUUID,
		TenantID:      tenantIDUUID,
		Justification: req.Justification,
		Duration:      req.Duration,
		Priority:      req.Priority,
		TicketRef:     req.TicketRef,
	}

	if req.TargetID != "" {
		if targetID, err := uuid.Parse(req.TargetID); err == nil {
			approvalReq.TargetID = &targetID
		}
	}
	if req.CredentialID != "" {
		if credentialID, err := uuid.Parse(req.CredentialID); err == nil {
			approvalReq.CredentialID = &credentialID
		}
	}

	if req.Priority == "" {
		approvalReq.Priority = "normal"
	}

	// Set expiration based on priority
	expiresAt := time.Now().Add(time.Duration(req.Duration) * time.Minute)
	approvalReq.ExpiresAt = &expiresAt

	createdReq, err := h.service.CreateRequest(c.Request.Context(), approvalReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create approval request")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CREATE_REQUEST_FAILED",
				"message": "Failed to create approval request",
			},
		})
		return
	}

	h.logger.Info().
		Str("request_id", createdReq.ID.String()).
		Str("type", string(createdReq.Type)).
		Msg("Approval request created")

	c.JSON(http.StatusCreated, gin.H{"request": createdReq})
}

// Approve handles an approval decision
func (h *ApprovalHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST_ID",
				"message": "Invalid request ID",
			},
		})
		return
	}

	var req ApproveRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	if req.Decision == "approve" {
		if err := h.service.ApproveRequest(c.Request.Context(), id, userIDUUID, req.Comments); err != nil {
			h.logger.Error().Err(err).Msg("Failed to approve request")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "APPROVE_FAILED",
					"message": "Failed to approve request: " + err.Error(),
				},
			})
			return
		}

		h.logger.Info().
			Str("request_id", id.String()).
			Str("approver_id", userIDUUID.String()).
			Msg("Request approved")

		c.JSON(http.StatusOK, gin.H{"message": "Request approved"})
	} else {
		if err := h.service.DenyRequest(c.Request.Context(), id, userIDUUID, req.Comments); err != nil {
			h.logger.Error().Err(err).Msg("Failed to deny request")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "DENY_FAILED",
					"message": "Failed to deny request: " + err.Error(),
				},
			})
			return
		}

		h.logger.Info().
			Str("request_id", id.String()).
			Str("approver_id", userIDUUID.String()).
			Msg("Request denied")

		c.JSON(http.StatusOK, gin.H{"message": "Request denied"})
	}
}

// Deny handles an explicit deny decision for an approval request
// SECURITY FIX: Separate handler prevents routing bugs where approve handler is reused for deny
func (h *ApprovalHandler) Deny(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST_ID",
				"message": "Invalid request ID",
			},
		})
		return
	}

	var req struct {
		Comments string `json:"comments"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	if err := h.service.DenyRequest(c.Request.Context(), id, userIDUUID, req.Comments); err != nil {
		h.logger.Error().Err(err).Msg("Failed to deny request")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DENY_FAILED",
				"message": "Failed to deny request: " + err.Error(),
			},
		})
		return
	}

	h.logger.Info().
		Str("request_id", id.String()).
		Str("denied_by", userIDUUID.String()).
		Msg("Request denied")

	c.JSON(http.StatusOK, gin.H{"message": "Request denied"})
}

// Cancel cancels a pending request
func (h *ApprovalHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST_ID",
				"message": "Invalid request ID",
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	if err := h.service.CancelRequest(c.Request.Context(), id, userIDUUID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to cancel request")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "CANCEL_FAILED",
				"message": "Failed to cancel request: " + err.Error(),
			},
		})
		return
	}

	h.logger.Info().
		Str("request_id", id.String()).
		Str("user_id", userIDUUID.String()).
		Msg("Request cancelled")

	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

// GetPending returns pending requests for the current user (as approver)
func (h *ApprovalHandler) GetPending(c *gin.Context) {
	userID, _ := c.Get("user_id")
	_ = userID // Will be used when implementing pending requests query

	// Return empty list for now
	c.JSON(http.StatusOK, gin.H{
		"requests": []interface{}{},
		"total":    0,
	})
}

// Delegate delegates approval to another user
func (h *ApprovalHandler) Delegate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_REQUEST_ID",
				"message": "Invalid request ID",
			},
		})
		return
	}

	var req struct {
		ToApproverID string `json:"to_approver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_INPUT",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))
	toApproverID, err := uuid.Parse(req.ToApproverID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_APPROVER_ID",
				"message": "Invalid approver ID",
			},
		})
		return
	}

	if err := h.service.DelegateApproval(c.Request.Context(), id, userIDUUID, toApproverID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delegate approval")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "DELEGATE_FAILED",
				"message": "Failed to delegate approval: " + err.Error(),
			},
		})
		return
	}

	h.logger.Info().
		Str("request_id", id.String()).
		Str("from", userIDUUID.String()).
		Str("to", toApproverID.String()).
		Msg("Approval delegated")

	c.JSON(http.StatusOK, gin.H{"message": "Approval delegated"})
}
