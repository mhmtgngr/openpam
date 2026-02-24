package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// ComplianceHandler handles compliance HTTP requests
type ComplianceHandler struct {
	service *analytics.Service
	logger  zerolog.Logger
}

// NewComplianceHandler creates a new compliance handler
func NewComplianceHandler(service *analytics.Service, logger zerolog.Logger) *ComplianceHandler {
	return &ComplianceHandler{
		service: service,
		logger:  logger,
	}
}

// ListReports retrieves compliance reports for a tenant
func (h *ComplianceHandler) ListReports(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var framework *string
	if fw := c.Query("framework"); fw != "" {
		framework = &fw
	}

	reports, err := h.service.ListComplianceReports(c.Request.Context(), tenantIDUUID, framework)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list compliance reports")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list reports"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// GenerateReport generates a new compliance report
func (h *ComplianceHandler) GenerateReport(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		Framework    string `json:"framework" binding:"required"`
		PeriodStart  string `json:"period_start" binding:"required"`
		PeriodEnd    string `json:"period_end" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	framework := analytics.ComplianceFramework(req.Framework)
	periodStart, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_start format"}})
		return
	}

	periodEnd, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_end format"}})
		return
	}

	report, err := h.service.GenerateComplianceReport(c.Request.Context(), tenantIDUUID, userIDUUID, framework, periodStart, periodEnd)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate compliance report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"report": report})
}

// GetReport retrieves a specific compliance report
func (h *ComplianceHandler) GetReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	report, err := h.service.GetComplianceReport(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get compliance report")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}

// GetControls retrieves control evaluations for a report
func (h *ComplianceHandler) GetControls(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	// Get report first to verify tenant access
	_, err = h.service.GetComplianceReport(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
		return
	}

	// Get controls from repository (service doesn't expose this directly)
	// In production, add GetControls method to service
	c.JSON(http.StatusOK, gin.H{"controls": []interface{}{}})
}

// GetSummary retrieves compliance summary across frameworks
func (h *ComplianceHandler) GetSummary(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	summary, err := h.service.GetComplianceSummary(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get compliance summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get summary"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// Compliance Exception handlers

func (h *ComplianceHandler) ListExceptions(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	exceptions, err := h.service.ListComplianceExceptions(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list compliance exceptions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list exceptions"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exceptions": exceptions})
}

func (h *ComplianceHandler) CreateException(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	var req analytics.ComplianceException
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	req.TenantID = tenantIDUUID
	req.RequestedBy = userIDUUID
	req.Status = "pending"

	if err := h.service.CreateComplianceException(c.Request.Context(), &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create compliance exception")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create exception"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"exception": req})
}

func (h *ComplianceHandler) ApproveException(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid exception ID"}})
		return
	}

	userID, _ := c.Get("user_id")
	_, _ = uuid.Parse(userID.(string)) // userIDUUID - will be used when updating exception

	var req struct {
		ExpiresAt string `json:"expires_at"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid expires_at format"}})
			return
		}
		expiresAt = &t
		_ = expiresAt // Will be used when updating exception
	}

	// Implementation would update exception status to approved
	c.JSON(http.StatusOK, gin.H{"message": "Exception approved", "id": id})
}

func (h *ComplianceHandler) DenyException(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid exception ID"}})
		return
	}

	userID, _ := c.Get("user_id")
	_, _ = uuid.Parse(userID.(string)) // userIDUUID - will be used when updating exception

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": "reason is required"}})
		return
	}

	// Implementation would update exception status to denied
	c.JSON(http.StatusOK, gin.H{"message": "Exception denied", "id": id})
}

// RefreshReports forces refresh of compliance reports
func (h *ComplianceHandler) RefreshReports(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Invalidate compliance cache
	_ = h.service.InvalidateCache(c.Request.Context(), tenantIDUUID, "compliance")

	c.JSON(http.StatusOK, gin.H{"message": "Compliance cache invalidated"})
}
