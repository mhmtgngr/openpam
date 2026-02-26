package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/openpam/openpam/internal/identity/repository"
	"github.com/openpam/openpam/internal/identity/service"
)

// AccessRequestHandler handles HTTP requests for access requests
type AccessRequestHandler struct {
	service *service.AccessRequestService
	logger  *zerolog.Logger
}

// NewAccessRequestHandler creates a new access request handler
func NewAccessRequestHandler(svc *service.AccessRequestService, logger *zerolog.Logger) *AccessRequestHandler {
	return &AccessRequestHandler{
		service: svc,
		logger:  logger,
	}
}

// CreateRequest represents a request to create an access request
type CreateRequest struct {
	Title               string             `json:"title" binding:"required,min=3,max=200"`
	Description         string             `json:"description" binding:"max=2000"`
	Type                repository.AccessRequestType `json:"type" binding:"required"`
	TargetResourceID    uuid.UUID          `json:"target_resource_id" binding:"required"`
	TargetResourceType  string             `json:"target_resource_type" binding:"required"`
	RequestedDuration   int                `json:"requested_duration" binding:"required,min=1,max=10080"`
	Reason              string             `json:"reason" binding:"required,min=10,max=1000"`
	BusinessJustification string           `json:"business_justification" binding:"max=2000"`
	TicketRef           string             `json:"ticket_ref"`
	Priority            string             `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	StartTime           *time.Time         `json:"start_time"`
	Metadata            map[string]string  `json:"metadata"`
}

// UpdateRequest represents a request to update an access request
type UpdateRequest struct {
	Title               string             `json:"title" binding:"omitempty,min=3,max=200"`
	Description         string             `json:"description" binding:"omitempty,max=2000"`
	Reason              string             `json:"reason" binding:"omitempty,min=10,max=1000"`
	BusinessJustification string           `json:"business_justification" binding:"omitempty,max=2000"`
	Priority            string             `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	StartTime           *time.Time         `json:"start_time"`
	Metadata            map[string]string  `json:"metadata"`
}

// ApprovalRequest represents a request to approve/deny
type ApprovalRequest struct {
	Comments     string `json:"comments" binding:"max=2000"`
	MFAVerified  bool   `json:"mfa_verified"`
}

// Create creates a new access request
func (h *AccessRequestHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	tenantID := c.GetHeader("X-Tenant-ID")
	requesterID := c.GetString("user_id")

	accessReq := &repository.AccessRequest{
		TenantID:             uuid.MustParse(tenantID),
		RequesterID:          uuid.MustParse(requesterID),
		Type:                 req.Type,
		Title:                req.Title,
		Description:          req.Description,
		TargetResourceID:     req.TargetResourceID,
		TargetResourceType:   req.TargetResourceType,
		RequestedDuration:    req.RequestedDuration,
		Reason:               req.Reason,
		BusinessJustification: req.BusinessJustification,
		TicketRef:            req.TicketRef,
		Priority:             req.Priority,
		StartTime:            req.StartTime,
		Metadata:             req.Metadata,
	}

	result, err := h.service.CreateRequest(c.Request.Context(), accessReq)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

// List lists access requests
func (h *AccessRequestHandler) List(c *gin.Context) {
	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")

	filter := repository.AccessRequestFilter{
		TenantID: &tenantID,
		Limit:    50,
	}

	// Non-admin users can only see their own requests
	role := c.GetString("role")
	if role != "admin" {
		uid, _ := uuid.Parse(userID)
		filter.RequesterID = &uid
	}

	// Parse query parameters
	if status := c.Query("status"); status != "" {
		s := repository.AccessRequestStatus(status)
		filter.Status = &s
	}
	if reqType := c.Query("type"); reqType != "" {
		t := repository.AccessRequestType(reqType)
		filter.Type = &t
	}
	if priority := c.Query("priority"); priority != "" {
		filter.Priority = &priority
	}

	requests, err := h.service.ListRequests(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to list requests",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": requests, "count": len(requests)})
}

// Get retrieves a single access request
func (h *AccessRequestHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	req, err := h.service.GetRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": req})
}

// Update updates an access request
func (h *AccessRequestHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")

	// Get existing request
	existing, err := h.service.GetRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Only requester can update
	if existing.RequesterID.String() != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "FORBIDDEN",
			"message": "You can only update your own requests",
		}})
		return
	}

	// Only allow updating pending requests
	if existing.Status != repository.RequestStatusDraft && existing.Status != repository.RequestStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_STATUS",
			"message": "Can only update draft or pending requests",
		}})
		return
	}

	// Apply updates
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Reason != "" {
		existing.Reason = req.Reason
	}
	if req.BusinessJustification != "" {
		existing.BusinessJustification = req.BusinessJustification
	}
	if req.Priority != "" {
		existing.Priority = req.Priority
	}
	if req.StartTime != nil {
		existing.StartTime = req.StartTime
	}
	if req.Metadata != nil {
		existing.Metadata = req.Metadata
	}

	updated, err := h.service.GetRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// Delete deletes an access request
func (h *AccessRequestHandler) Delete(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	_, _ = uuid.Parse(c.GetHeader("X-Tenant-ID"))

	c.JSON(http.StatusNoContent, nil)
}

// Approve approves an access request
func (h *AccessRequestHandler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	approverID := c.GetString("user_id")
	approverUUID, _ := uuid.Parse(approverID)

	if err := h.service.ApproveRequest(c.Request.Context(), id, tenantID, approverUUID, req.Comments, req.MFAVerified); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request approved"})
}

// Deny denies an access request
func (h *AccessRequestHandler) Deny(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	approverID := c.GetString("user_id")
	approverUUID, _ := uuid.Parse(approverID)

	if err := h.service.DenyRequest(c.Request.Context(), id, tenantID, approverUUID, req.Comments, req.MFAVerified); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request denied"})
}

// Cancel cancels an access request
func (h *AccessRequestHandler) Cancel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")
	userUUID, _ := uuid.Parse(userID)

	if err := h.service.CancelRequest(c.Request.Context(), id, tenantID, userUUID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

// GetHistory retrieves the approval history for a request
func (h *AccessRequestHandler) GetHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	// Get the request first to verify access
	_, err = h.service.GetRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	// Return approval history
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "request_id": id})
}

func (h *AccessRequestHandler) handleError(c *gin.Context, err error) {
	switch {
	case err == service.ErrRequestNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Access request not found",
		}})
	case err == service.ErrUnauthorizedApprover:
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":    "FORBIDDEN",
			"message": "You are not authorized to perform this action",
		}})
	case err == service.ErrRequestNotPending:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_STATUS",
			"message": "Request is not in the correct status for this action",
		}})
	case err == service.ErrRequestAlreadyApproved:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "ALREADY_APPROVED",
			"message": "Request has already been approved",
		}})
	case err == service.ErrRequestAlreadyDenied:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "ALREADY_DENIED",
			"message": "Request has already been denied",
		}})
	case err == service.ErrMFAMissing:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "MFA_REQUIRED",
			"message": "MFA verification is required for this action",
		}})
	case err == service.ErrInvalidDuration:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_DURATION",
			"message": "Requested duration is invalid",
		}})
	case err == service.ErrTicketRequired:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "TICKET_REQUIRED",
			"message": "ITSM ticket reference is required for this request",
		}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "An internal error occurred",
		}})
	}
}

// WorkflowHandler handles HTTP requests for workflows
type WorkflowHandler struct {
	service *service.WorkflowEngine
	logger  *zerolog.Logger
}

// NewWorkflowHandler creates a new workflow handler
func NewWorkflowHandler(svc *service.WorkflowEngine, logger *zerolog.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *WorkflowHandler) Create(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{
		"code":    "NOT_IMPLEMENTED",
		"message": "Workflow creation via API not yet implemented",
	}})
}

func (h *WorkflowHandler) List(c *gin.Context) {
	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	filter := repository.WorkflowFilter{
		TenantID: &tenantID,
		Limit:    50,
	}

	workflows, err := h.service.ListWorkflows(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to list workflows",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workflows})
}

func (h *WorkflowHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid workflow ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	workflow, err := h.service.GetWorkflow(c.Request.Context(), id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Workflow not found",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workflow})
}

func (h *WorkflowHandler) Update(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{
		"code":    "NOT_IMPLEMENTED",
		"message": "Workflow update via API not yet implemented",
	}})
}

func (h *WorkflowHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid workflow ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	if err := h.service.DeleteWorkflow(c.Request.Context(), id, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		}})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *WorkflowHandler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid workflow ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	if err := h.service.ActivateWorkflow(c.Request.Context(), id, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow activated"})
}

func (h *WorkflowHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid workflow ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	if err := h.service.DeactivateWorkflow(c.Request.Context(), id, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow deactivated"})
}

// BreakGlassHandler handles HTTP requests for break glass access
type BreakGlassHandler struct {
	service *service.BreakGlassService
	logger  *zerolog.Logger
}

// NewBreakGlassHandler creates a new break glass handler
func NewBreakGlassHandler(svc *service.BreakGlassService, logger *zerolog.Logger) *BreakGlassHandler {
	return &BreakGlassHandler{
		service: svc,
		logger:  logger,
	}
}

func (h *BreakGlassHandler) Request(c *gin.Context) {
	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var req service.BreakGlassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	req.TenantID = tenantID
	req.RequesterID = uuid.MustParse(userID)

	result, err := h.service.RequestBreakGlass(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *BreakGlassHandler) List(c *gin.Context) {
	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	requests, err := h.service.ListBreakGlassRequests(c.Request.Context(), tenantID, status, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to list break glass requests",
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": requests})
}

func (h *BreakGlassHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	// Get as access request
	_, err = h.service.GetBreakGlassRequest(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{}})
}

func (h *BreakGlassHandler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.service.ActivateBreakGlass(c.Request.Context(), id, tenantID, uuid.MustParse(userID), ipAddress, userAgent)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *BreakGlassHandler) Revoke(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_INPUT",
			"message": err.Error(),
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))
	userID := c.GetString("user_id")
	ipAddress := c.ClientIP()

	if err := h.service.RevokeBreakGlass(c.Request.Context(), id, tenantID, uuid.MustParse(userID), req.Reason, ipAddress); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Break glass access revoked"})
}

func (h *BreakGlassHandler) AuditLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_ID",
			"message": "Invalid request ID",
		}})
		return
	}

	tenantID, _ := uuid.Parse(c.GetHeader("X-Tenant-ID"))

	log, err := h.service.GetAuditLog(c.Request.Context(), id, tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": log})
}

func (h *BreakGlassHandler) handleError(c *gin.Context, err error) {
	switch {
	case err == service.ErrBreakGlassNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"code":    "NOT_FOUND",
			"message": "Break glass request not found",
		}})
	case err == service.ErrBreakGlassActive:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "ACTIVE_EXISTS",
			"message": "User already has active break glass access",
		}})
	case err == service.ErrInvalidJustification:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "INVALID_JUSTIFICATION",
			"message": "Business justification is required and must be sufficient",
		}})
	case err == service.ErrMaxDurationExceeded:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "MAX_DURATION_EXCEEDED",
			"message": "Requested duration exceeds maximum allowed",
		}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "An internal error occurred",
		}})
	}
}
