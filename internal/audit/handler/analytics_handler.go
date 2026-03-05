// Package handler provides HTTP handlers for audit analytics endpoints
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// AnalyticsHandler handles analytics HTTP requests
type AnalyticsHandler struct {
	analyticsService *audit.AnalyticsService
	logger           zerolog.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(analyticsService *audit.AnalyticsService, logger zerolog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// =============================================================================
// Dashboard Handlers
// =============================================================================

// GetDashboardSummary handles GET /api/v1/analytics/dashboard
func (h *AnalyticsHandler) GetDashboardSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	summary, err := h.analyticsService.GetDashboardSummary(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get dashboard summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get dashboard summary"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// =============================================================================
// Compliance Report Handlers
// =============================================================================

// CreateComplianceReport handles POST /api/v1/analytics/compliance/reports
func (h *AnalyticsHandler) CreateComplianceReport(c *gin.Context) {
	var req model.CreateComplianceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	report := &model.ComplianceReport{
		TenantID:     tenantID,
		ReportName:   req.ReportName,
		Framework:    req.Framework,
		Version:      req.Version,
		GeneratedAt:  time.Now(),
		GeneratedBy:  userID,
		PeriodStart:  req.PeriodStart,
		PeriodEnd:    req.PeriodEnd,
		Status:       string(model.ComplianceStatusPending),
		TotalControls: 0,
	}

	if err := h.analyticsService.CreateComplianceReport(c.Request.Context(), report); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create compliance report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create report"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"report": report})
}

// GetComplianceReport handles GET /api/v1/analytics/compliance/reports/:id
func (h *AnalyticsHandler) GetComplianceReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	report, err := h.analyticsService.GetComplianceReport(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get compliance report")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"report": report})
}

// ListComplianceReports handles GET /api/v1/analytics/compliance/reports
func (h *AnalyticsHandler) ListComplianceReports(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.ComplianceReportFilter{TenantID: &tenantID}

	if framework := c.Query("framework"); framework != "" {
		filter.Framework = &framework
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	reports, total, err := h.analyticsService.ListComplianceReports(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list compliance reports")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list reports"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// UpdateComplianceReportStatus handles PATCH /api/v1/analytics/compliance/reports/:id/status
func (h *AnalyticsHandler) UpdateComplianceReportStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	var req struct {
		Status          *string  `json:"status"`
		OverallScore    *float64 `json:"overall_score"`
		PassedControls  *int     `json:"passed_controls"`
		FailedControls  *int     `json:"failed_controls"`
		SkippedControls *int     `json:"skipped_controls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	if req.Status == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "MISSING_STATUS", "message": "Status is required"}})
		return
	}

	passedControls := 0
	if req.PassedControls != nil {
		passedControls = *req.PassedControls
	}
	failedControls := 0
	if req.FailedControls != nil {
		failedControls = *req.FailedControls
	}
	skippedControls := 0
	if req.SkippedControls != nil {
		skippedControls = *req.SkippedControls
	}

	if err := h.analyticsService.UpdateComplianceReportStatus(c.Request.Context(), id, *req.Status, req.OverallScore, passedControls, failedControls, skippedControls); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update compliance report status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update status"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// DeleteComplianceReport handles DELETE /api/v1/analytics/compliance/reports/:id
func (h *AnalyticsHandler) DeleteComplianceReport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	if err := h.analyticsService.DeleteComplianceReport(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete compliance report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete report"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Report deleted"})
}

// =============================================================================
// Compliance Exception Handlers
// =============================================================================

// CreateComplianceException handles POST /api/v1/analytics/compliance/exceptions
func (h *AnalyticsHandler) CreateComplianceException(c *gin.Context) {
	var req model.CreateComplianceExceptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	exception := &model.ComplianceException{
		TenantID:             tenantID,
		ControlID:            req.ControlID,
		ControlName:          req.ControlName,
		Framework:            req.Framework,
		Status:               string(model.ExceptionStatusPending),
		RiskLevel:            req.RiskLevel,
		RequestedBy:          userID,
		Justification:        req.Justification,
		BusinessReason:       &req.BusinessReason,
		CompensatingControls: req.CompensatingControls,
		ExpiresAt:            req.ExpiresAt,
	}

	if err := h.analyticsService.CreateComplianceException(c.Request.Context(), exception); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create compliance exception")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create exception"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"exception": exception})
}

// GetComplianceException handles GET /api/v1/analytics/compliance/exceptions/:id
func (h *AnalyticsHandler) GetComplianceException(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid exception ID"}})
		return
	}

	exception, err := h.analyticsService.GetComplianceException(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get compliance exception")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Exception not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exception": exception})
}

// ListComplianceExceptions handles GET /api/v1/analytics/compliance/exceptions
func (h *AnalyticsHandler) ListComplianceExceptions(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.ComplianceExceptionFilter{TenantID: &tenantID}

	if controlID := c.Query("control_id"); controlID != "" {
		filter.ControlID = &controlID
	}
	if framework := c.Query("framework"); framework != "" {
		filter.Framework = &framework
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if riskLevel := c.Query("risk_level"); riskLevel != "" {
		filter.RiskLevel = &riskLevel
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	exceptions, total, err := h.analyticsService.ListComplianceExceptions(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list compliance exceptions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list exceptions"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exceptions": exceptions,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// UpdateComplianceExceptionStatus handles PATCH /api/v1/analytics/compliance/exceptions/:id/status
func (h *AnalyticsHandler) UpdateComplianceExceptionStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid exception ID"}})
		return
	}

	var req struct {
		Status         *string   `json:"status"`
		ApprovedBy     *uuid.UUID `json:"approved_by,omitempty"`
		RiskAcceptedBy *uuid.UUID `json:"risk_accepted_by,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	if req.Status == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "MISSING_STATUS", "message": "Status is required"}})
		return
	}

	if err := h.analyticsService.UpdateComplianceExceptionStatus(c.Request.Context(), id, *req.Status, req.ApprovedBy, req.RiskAcceptedBy); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update compliance exception status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update status"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// DeleteComplianceException handles DELETE /api/v1/analytics/compliance/exceptions/:id
func (h *AnalyticsHandler) DeleteComplianceException(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid exception ID"}})
		return
	}

	if err := h.analyticsService.DeleteComplianceException(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete compliance exception")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete exception"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Exception deleted"})
}

// GetExpiringExceptions handles GET /api/v1/analytics/compliance/exceptions/expiring
func (h *AnalyticsHandler) GetExpiringExceptions(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Default to 30 days
	within := 30 * 24 * time.Hour
	if days := c.Query("days"); days != "" {
		if d, err := strconv.Atoi(days); err == nil && d > 0 {
			within = time.Duration(d) * 24 * time.Hour
		}
	}

	exceptions, err := h.analyticsService.GetExpiringExceptions(c.Request.Context(), tenantID, within)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get expiring exceptions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get expiring exceptions"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exceptions": exceptions})
}

// =============================================================================
// Anomaly Handlers
// =============================================================================

// CreateAnomaly handles POST /api/v1/analytics/anomalies
func (h *AnalyticsHandler) CreateAnomaly(c *gin.Context) {
	var anomaly model.AnomalyDetection
	if err := c.ShouldBindJSON(&anomaly); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	if err := h.analyticsService.CreateAnomaly(c.Request.Context(), &anomaly); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create anomaly")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create anomaly"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"anomaly": anomaly})
}

// GetAnomaly handles GET /api/v1/analytics/anomalies/:id
func (h *AnalyticsHandler) GetAnomaly(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
		return
	}

	anomaly, err := h.analyticsService.GetAnomaly(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get anomaly")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Anomaly not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomaly": anomaly})
}

// ListAnomalies handles GET /api/v1/analytics/anomalies
func (h *AnalyticsHandler) ListAnomalies(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.AnomalyFilter{TenantID: &tenantID}

	if userID := c.Query("user_id"); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			filter.UserID = &uid
		}
	}
	if anomalyType := c.Query("type"); anomalyType != "" {
		filter.AnomalyType = &anomalyType
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = &severity
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	anomalies, total, err := h.analyticsService.ListAnomalies(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list anomalies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list anomalies"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"anomalies": anomalies,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// UpdateAnomaly handles PATCH /api/v1/analytics/anomalies/:id
func (h *AnalyticsHandler) UpdateAnomaly(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
		return
	}

	var req model.UpdateAnomalyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Get existing anomaly
	anomaly, err := h.analyticsService.GetAnomaly(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Anomaly not found"}})
		return
	}

	// Update fields
	if req.Status != nil {
		anomaly.Status = *req.Status
	}
	if req.AssignedTo != nil {
		anomaly.AssignedTo = req.AssignedTo
	}
	if req.ResolutionNotes != nil {
		anomaly.ResolutionNotes = req.ResolutionNotes
	}

	if err := h.analyticsService.UpdateAnomaly(c.Request.Context(), anomaly); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update anomaly")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update anomaly"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomaly": anomaly})
}

// DeleteAnomaly handles DELETE /api/v1/analytics/anomalies/:id
func (h *AnalyticsHandler) DeleteAnomaly(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
		return
	}

	if err := h.analyticsService.DeleteAnomaly(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete anomaly")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete anomaly"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Anomaly deleted"})
}

// GetAnomalyStats handles GET /api/v1/analytics/anomalies/stats
func (h *AnalyticsHandler) GetAnomalyStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.analyticsService.GetAnomalyStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get anomaly stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// =============================================================================
// SSH Key Analytics Handlers
// =============================================================================

// GetSSHKeyAnalytics handles GET /api/v1/analytics/ssh-keys/:id/analytics
func (h *AnalyticsHandler) GetSSHKeyAnalytics(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	sshKeyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid SSH key ID"}})
		return
	}

	var date time.Time
	if dateStr := c.Query("date"); dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid date format, use YYYY-MM-DD"}})
			return
		}
	} else {
		date = time.Now().Truncate(24 * time.Hour)
	}

	analytics, err := h.analyticsService.GetSSHKeyAnalytics(c.Request.Context(), tenantID, sshKeyID, date)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get SSH key analytics")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Analytics not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// ListSSHKeyAnalytics handles GET /api/v1/analytics/ssh-keys/analytics
func (h *AnalyticsHandler) ListSSHKeyAnalytics(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.SSHKeyAnalyticsFilter{TenantID: &tenantID}

	if sshKeyID := c.Query("ssh_key_id"); sshKeyID != "" {
		if id, err := uuid.Parse(sshKeyID); err == nil {
			filter.SSHKeyID = &id
		}
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	analytics, err := h.analyticsService.ListSSHKeyAnalytics(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list SSH key analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list analytics"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// GetSSHKeyUsageSummary handles GET /api/v1/analytics/ssh-keys/:id/summary
func (h *AnalyticsHandler) GetSSHKeyUsageSummary(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Get date range (default to last 30 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30)

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = t
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = t
		}
	}

	summary, err := h.analyticsService.GetSSHKeyUsageSummary(c.Request.Context(), tenantID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get SSH key usage summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get summary"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// GetMostUsedSSHKeys handles GET /api/v1/analytics/ssh-keys/most-used
func (h *AnalyticsHandler) GetMostUsedSSHKeys(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Get date range (default to last 30 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30)

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = t
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = t
		}
	}

	limit := getIntQuery(c, "limit", 10)

	ranks, err := h.analyticsService.GetMostUsedSSHKeys(c.Request.Context(), tenantID, dateFrom, dateTo, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get most used SSH keys")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get most used keys"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": ranks})
}

// GetAnomalousSSHKeys handles GET /api/v1/analytics/ssh-keys/anomalous
func (h *AnalyticsHandler) GetAnomalousSSHKeys(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Get date range (default to last 30 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30)

	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if t, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			dateFrom = t
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if t, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = t
		}
	}

	threshold := getIntQuery(c, "threshold", 5)

	analytics, err := h.analyticsService.GetAnomalousSSHKeys(c.Request.Context(), tenantID, dateFrom, dateTo, threshold)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get anomalous SSH keys")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get anomalous keys"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// =============================================================================
// Command Blacklist Handlers
// =============================================================================

// CreateCommandBlacklist handles POST /api/v1/analytics/blacklist
func (h *AnalyticsHandler) CreateCommandBlacklist(c *gin.Context) {
	var req model.CreateCommandBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	blacklist := &model.CommandBlacklist{
		TenantID:        &tenantID,
		CommandPattern:  req.CommandPattern,
		PatternType:     req.PatternType,
		Action:          req.Action,
		Severity:        req.Severity,
		AppliesToUsers:  req.AppliesToUsers,
		AppliesToGroups: req.AppliesToGroups,
		AppliesToTargets: req.AppliesToTargets,
		AllowOverride:   req.AllowOverride,
		OverrideRoles:   req.OverrideRoles,
		Reason:          req.Reason,
		CreatedBy:       userID,
		Enabled:         true,
	}

	if req.BaseCommand != "" {
		blacklist.BaseCommand = &req.BaseCommand
	}
	if req.RiskCategory != "" {
		blacklist.RiskCategory = &req.RiskCategory
	}

	if err := h.analyticsService.CreateCommandBlacklist(c.Request.Context(), blacklist); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create blacklist entry"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"blacklist": blacklist})
}

// GetCommandBlacklist handles GET /api/v1/analytics/blacklist/:id
func (h *AnalyticsHandler) GetCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	blacklist, err := h.analyticsService.GetCommandBlacklist(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get command blacklist")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Blacklist entry not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": blacklist})
}

// ListCommandBlacklist handles GET /api/v1/analytics/blacklist
func (h *AnalyticsHandler) ListCommandBlacklist(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.CommandBlacklistFilter{TenantID: &tenantID}

	if enabled := c.Query("enabled"); enabled != "" {
		if e, err := strconv.ParseBool(enabled); err == nil {
			filter.Enabled = &e
		}
	}
	if patternType := c.Query("pattern_type"); patternType != "" {
		filter.PatternType = &patternType
	}
	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = &severity
	}

	blacklists, err := h.analyticsService.ListCommandBlacklist(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": blacklists})
}

// UpdateCommandBlacklist handles PATCH /api/v1/analytics/blacklist/:id
func (h *AnalyticsHandler) UpdateCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	blacklist, err := h.analyticsService.GetCommandBlacklist(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Blacklist entry not found"}})
		return
	}

	var req model.UpdateCommandBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Update fields
	if req.CommandPattern != nil {
		blacklist.CommandPattern = *req.CommandPattern
	}
	if req.PatternType != nil {
		blacklist.PatternType = *req.PatternType
	}
	if req.BaseCommand != nil {
		blacklist.BaseCommand = req.BaseCommand
	}
	if req.Action != nil {
		blacklist.Action = *req.Action
	}
	if req.Severity != nil {
		blacklist.Severity = *req.Severity
	}
	if req.AppliesToUsers != nil {
		blacklist.AppliesToUsers = req.AppliesToUsers
	}
	if req.AppliesToGroups != nil {
		blacklist.AppliesToGroups = req.AppliesToGroups
	}
	if req.AppliesToTargets != nil {
		blacklist.AppliesToTargets = req.AppliesToTargets
	}
	if req.AllowOverride != nil {
		blacklist.AllowOverride = *req.AllowOverride
	}
	if req.OverrideRoles != nil {
		blacklist.OverrideRoles = req.OverrideRoles
	}
	if req.Reason != nil {
		blacklist.Reason = *req.Reason
	}
	if req.RiskCategory != nil {
		blacklist.RiskCategory = req.RiskCategory
	}
	if req.Enabled != nil {
		blacklist.Enabled = *req.Enabled
	}

	if err := h.analyticsService.UpdateCommandBlacklist(c.Request.Context(), blacklist); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": blacklist})
}

// DeleteCommandBlacklist handles DELETE /api/v1/analytics/blacklist/:id
func (h *AnalyticsHandler) DeleteCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	if err := h.analyticsService.DeleteCommandBlacklist(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry deleted"})
}

// GetBlacklistStats handles GET /api/v1/analytics/blacklist/stats
func (h *AnalyticsHandler) GetBlacklistStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.analyticsService.GetBlacklistStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get blacklist stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// EnableCommandBlacklist handles POST /api/v1/analytics/blacklist/:id/enable
func (h *AnalyticsHandler) EnableCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	if err := h.analyticsService.EnableCommandBlacklist(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to enable command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to enable blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry enabled"})
}

// DisableCommandBlacklist handles POST /api/v1/analytics/blacklist/:id/disable
func (h *AnalyticsHandler) DisableCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	if err := h.analyticsService.DisableCommandBlacklist(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to disable command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to disable blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry disabled"})
}

// CheckCommandAgainstBlacklist handles POST /api/v1/analytics/blacklist/check
func (h *AnalyticsHandler) CheckCommandAgainstBlacklist(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req struct {
		Command string     `json:"command" binding:"required"`
		UserIDs []uuid.UUID `json:"user_ids"`
		GroupIDs []uuid.UUID `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	allowed, action, entry := h.analyticsService.EvaluateCommandAgainstBlacklist(
		c.Request.Context(),
		tenantID,
		req.Command,
		req.UserIDs,
		req.GroupIDs,
	)

	result := gin.H{
		"allowed": allowed,
		"action":  action,
	}
	if entry != nil {
		result["blacklist"] = entry
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// =============================================================================
// Cache Handlers
// =============================================================================

// InvalidateCache handles POST /api/v1/analytics/cache/invalidate
func (h *AnalyticsHandler) InvalidateCache(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req struct {
		CacheTypes []string `json:"cache_types"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body, invalidate all
		req.CacheTypes = []string{}
	}

	if err := h.analyticsService.InvalidateCache(c.Request.Context(), tenantID, req.CacheTypes...); err != nil {
		h.logger.Error().Err(err).Msg("Failed to invalidate cache")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to invalidate cache"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated"})
}

// GetCacheStats handles GET /api/v1/analytics/cache/stats
func (h *AnalyticsHandler) GetCacheStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.analyticsService.GetCacheStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get cache stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get cache stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// =============================================================================
// Export Handlers
// =============================================================================

// ExportComplianceReport handles GET /api/v1/analytics/export/compliance
func (h *AnalyticsHandler) ExportComplianceReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	reportID, err := uuid.Parse(c.Query("report_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	report, err := h.analyticsService.GetComplianceReport(c.Request.Context(), reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
		return
	}

	// Verify tenant matches
	if report.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
		return
	}

	switch format {
	case "json":
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=compliance_report_%s.json", reportID))
		c.JSON(http.StatusOK, report)
	case "csv":
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=compliance_report_%s.csv", reportID))
		csv := complianceReportToCSV(report)
		c.Data(http.StatusOK, "text/csv", []byte(csv))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_FORMAT", "message": "Invalid export format"}})
	}
}

// ExportAnomalies handles GET /api/v1/analytics/export/anomalies
func (h *AnalyticsHandler) ExportAnomalies(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.AnomalyFilter{TenantID: &tenantID}

	// Parse date range
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	anomalies, _, err := h.analyticsService.ListAnomalies(c.Request.Context(), tenantID, filter, 10000, 0)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to export anomalies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to export anomalies"}})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		c.Header("Content-Disposition", "attachment; filename=anomalies_export.json")
		c.JSON(http.StatusOK, anomalies)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_FORMAT", "message": "Invalid export format"}})
	}
}

// =============================================================================
// Admin Handlers
// =============================================================================

// GetAdminStats handles GET /api/v1/analytics/admin/stats
func (h *AnalyticsHandler) GetAdminStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Gather all stats
	stats := make(map[string]interface{})

	// Anomaly stats
	if anomalyStats, err := h.analyticsService.GetAnomalyStats(c.Request.Context(), tenantID); err == nil {
		stats["anomalies"] = anomalyStats
	}

	// Blacklist stats
	if blacklistStats, err := h.analyticsService.GetBlacklistStats(c.Request.Context(), tenantID); err == nil {
		stats["blacklist"] = blacklistStats
	}

	// SSH key summary (last 30 days)
	dateTo := time.Now()
	dateFrom := dateTo.AddDate(0, 0, -30)
	if sshSummary, err := h.analyticsService.GetSSHKeyUsageSummary(c.Request.Context(), tenantID, dateFrom, dateTo); err == nil {
		stats["ssh_keys"] = sshSummary
	}

	// Compliance summary
	if complianceSummary, err := h.analyticsService.GetFrameworkSummary(c.Request.Context(), tenantID); err == nil {
		stats["compliance"] = complianceSummary
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// RunAnomalyDetection handles POST /api/v1/analytics/admin/anomaly-detection/run
func (h *AnalyticsHandler) RunAnomalyDetection(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req struct {
		AnomalyType string                 `json:"anomaly_type"`
		Options     map[string]interface{} `json:"options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults
		req.AnomalyType = "behavioral"
		req.Options = make(map[string]interface{})
	}

	// This is a placeholder - in production, this would trigger
	// an actual anomaly detection job
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Anomaly detection job started",
		"tenant_id": tenantID,
		"type": req.AnomalyType,
		"job_id": uuid.New().String(),
	})
}

// GenerateComplianceReport handles POST /api/v1/analytics/admin/compliance/generate
func (h *AnalyticsHandler) GenerateComplianceReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	var req model.CreateComplianceReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	report := &model.ComplianceReport{
		TenantID:      tenantID,
		ReportName:    req.ReportName,
		Framework:     req.Framework,
		Version:       req.Version,
		GeneratedAt:   time.Now(),
		GeneratedBy:   userID,
		PeriodStart:   req.PeriodStart,
		PeriodEnd:     req.PeriodEnd,
		Status:        string(model.ComplianceStatusPending),
		TotalControls: 0,
	}

	if err := h.analyticsService.CreateComplianceReport(c.Request.Context(), report); err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate compliance report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
		return
	}

	// In production, this would trigger an async job to evaluate controls
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Compliance report generation started",
		"report_id": report.ID,
		"status": "pending",
	})
}

// =============================================================================
// Report Snapshot Handlers
// =============================================================================

// GenerateReportSnapshot handles POST /api/v1/analytics/reports/generate
func (h *AnalyticsHandler) GenerateReportSnapshot(c *gin.Context) {
	var req model.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	snapshot, err := h.analyticsService.GenerateReport(
		c.Request.Context(),
		tenantID,
		req.ReportID,
		userID,
		req.SnapshotName,
		req.Format,
		req.Options,
		req.RetentionDays,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate report snapshot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"snapshot": snapshot})
}

// QueueReportGeneration handles POST /api/v1/analytics/reports/queue
func (h *AnalyticsHandler) QueueReportGeneration(c *gin.Context) {
	var req model.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	job, err := h.analyticsService.QueueReportGeneration(
		c.Request.Context(),
		tenantID,
		req.ReportID,
		req.SnapshotName,
		req.Format,
		req.Options,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to queue report generation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to queue report generation"}})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Report generation queued",
	})
}

// GetReportSnapshot handles GET /api/v1/analytics/reports/snapshots/:id
func (h *AnalyticsHandler) GetReportSnapshot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid snapshot ID"}})
		return
	}

	snapshot, err := h.analyticsService.GetReportSnapshot(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report snapshot")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Snapshot not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"snapshot": snapshot})
}

// ListReportSnapshots handles GET /api/v1/analytics/reports/snapshots
func (h *AnalyticsHandler) ListReportSnapshots(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.ReportSnapshotFilter{TenantID: &tenantID}

	if reportID := c.Query("report_id"); reportID != "" {
		if id, err := uuid.Parse(reportID); err == nil {
			filter.ReportID = &id
		}
	}
	if framework := c.Query("framework"); framework != "" {
		filter.Framework = &framework
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if format := c.Query("format"); format != "" {
		filter.Format = &format
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = &t
		}
	}
	filter.IncludeExpired = c.Query("include_expired") == "true"

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	snapshots, total, err := h.analyticsService.ListReportSnapshots(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report snapshots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list snapshots"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshots": snapshots,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// DeleteReportSnapshot handles DELETE /api/v1/analytics/reports/snapshots/:id
func (h *AnalyticsHandler) DeleteReportSnapshot(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid snapshot ID"}})
		return
	}

	if err := h.analyticsService.DeleteReportSnapshot(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete report snapshot")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete snapshot"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snapshot deleted"})
}

// GetReportSnapshotStats handles GET /api/v1/analytics/reports/stats
func (h *AnalyticsHandler) GetReportSnapshotStats(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.analyticsService.GetReportSnapshotStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report snapshot stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// =============================================================================
// Report Generation Job Handlers
// =============================================================================

// GetReportGenerationJob handles GET /api/v1/analytics/reports/jobs/:id
func (h *AnalyticsHandler) GetReportGenerationJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid job ID"}})
		return
	}

	job, err := h.analyticsService.GetReportGenerationJob(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report generation job")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Job not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

// ListReportGenerationJobs handles GET /api/v1/analytics/reports/jobs
func (h *AnalyticsHandler) ListReportGenerationJobs(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.ReportGenerationJobFilter{TenantID: &tenantID}

	if jobType := c.Query("job_type"); jobType != "" {
		filter.JobType = &jobType
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	jobs, total, err := h.analyticsService.ListReportGenerationJobs(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report generation jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list jobs"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteReportGenerationJob handles DELETE /api/v1/analytics/reports/jobs/:id
func (h *AnalyticsHandler) DeleteReportGenerationJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid job ID"}})
		return
	}

	if err := h.analyticsService.DeleteReportGenerationJob(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete report generation job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete job"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted"})
}

// =============================================================================
// Report Schedule Handlers
// =============================================================================

// CreateReportSchedule handles POST /api/v1/analytics/reports/schedules
func (h *AnalyticsHandler) CreateReportSchedule(c *gin.Context) {
	var req model.CreateReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	// Marshal options
	var optionsBytes []byte
	if req.Options != nil {
		optionsBytes, _ = json.Marshal(req.Options)
	}

	// Set default retention
	retentionDays := req.RetentionDays
	if retentionDays == 0 {
		retentionDays = 90
	}

	schedule := &model.ReportSchedule{
		TenantID:           tenantID,
		ScheduleName:       req.ScheduleName,
		ReportID:           req.ReportID,
		ScheduleType:       req.ScheduleType,
		CronExpression:     req.CronExpression,
		Format:             req.Format,
		Options:            optionsBytes,
		Recipients:         req.Recipients,
		NotifyOnCompletion: req.NotifyOnCompletion,
		NotifyOnFailure:    req.NotifyOnFailure,
		Status:             string(model.ScheduleStatusActive),
		NextRunAt:          req.NextRunAt,
		CreatedBy:          userID,
		OwnedBy:            userID,
		RetentionDays:      retentionDays,
	}

	if err := h.analyticsService.CreateReportSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create schedule"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

// GetReportSchedule handles GET /api/v1/analytics/reports/schedules/:id
func (h *AnalyticsHandler) GetReportSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	schedule, err := h.analyticsService.GetReportSchedule(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Schedule not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

// UpdateReportSchedule handles PATCH /api/v1/analytics/reports/schedules/:id
func (h *AnalyticsHandler) UpdateReportSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	// Get existing schedule
	schedule, err := h.analyticsService.GetReportSchedule(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Schedule not found"}})
		return
	}

	var req model.UpdateReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Update fields from request
	if req.ScheduleName != nil {
		schedule.ScheduleName = *req.ScheduleName
	}
	if req.ScheduleType != nil {
		schedule.ScheduleType = *req.ScheduleType
	}
	if req.CronExpression != nil {
		schedule.CronExpression = req.CronExpression
	}
	if req.Format != nil {
		schedule.Format = *req.Format
	}
	if req.Options != nil {
		optionsBytes, _ := json.Marshal(req.Options)
		schedule.Options = optionsBytes
	}
	if req.Recipients != nil {
		schedule.Recipients = req.Recipients
	}
	if req.NotifyOnCompletion != nil {
		schedule.NotifyOnCompletion = *req.NotifyOnCompletion
	}
	if req.NotifyOnFailure != nil {
		schedule.NotifyOnFailure = *req.NotifyOnFailure
	}
	if req.Status != nil {
		schedule.Status = *req.Status
	}
	if req.NextRunAt != nil {
		schedule.NextRunAt = *req.NextRunAt
	}
	if req.RetentionDays != nil {
		schedule.RetentionDays = *req.RetentionDays
	}

	if err := h.analyticsService.UpdateReportSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update schedule"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

// DeleteReportSchedule handles DELETE /api/v1/analytics/reports/schedules/:id
func (h *AnalyticsHandler) DeleteReportSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	if err := h.analyticsService.DeleteReportSchedule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete schedule"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted"})
}

// ListReportSchedules handles GET /api/v1/analytics/reports/schedules
func (h *AnalyticsHandler) ListReportSchedules(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filter := model.ReportScheduleFilter{TenantID: &tenantID}

	if framework := c.Query("framework"); framework != "" {
		filter.Framework = &framework
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}
	if scheduleType := c.Query("schedule_type"); scheduleType != "" {
		filter.ScheduleType = &scheduleType
	}
	if ownedBy := c.Query("owned_by"); ownedBy != "" {
		if id, err := uuid.Parse(ownedBy); err == nil {
			filter.OwnedBy = &id
		}
	}
	filter.IncludeActive = c.Query("active") == "true"

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	schedules, total, err := h.analyticsService.ListReportSchedules(c.Request.Context(), tenantID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list schedules"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func getTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}
	return uuid.Parse(tenantIDStr.(string))
}

func getUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user_id not found in context")
	}
	return uuid.Parse(userIDStr.(string))
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	if intVal, err := strconv.Atoi(val); err == nil {
		return intVal
	}
	return defaultVal
}

// complianceReportToCSV converts a compliance report to CSV format
func complianceReportToCSV(report *model.ComplianceReport) string {
	var b strings.Builder

	// Header
	b.WriteString("Report ID,Report Name,Framework,Version,Status,Overall Score,Total Controls,Passed,Failed,Skipped,Period Start,Period End,Generated At\n")

	// Report summary row
	score := ""
	if report.OverallScore != nil {
		score = fmt.Sprintf("%.2f", *report.OverallScore)
	}
	summary := ""
	if report.Summary != nil {
		summary = csvEscape(*report.Summary)
	}
	_ = summary

	b.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d,%d,%d,%d,%s,%s,%s\n",
		report.ID,
		csvEscape(report.ReportName),
		csvEscape(report.Framework),
		csvEscape(report.Version),
		csvEscape(report.Status),
		score,
		report.TotalControls,
		report.PassedControls,
		report.FailedControls,
		report.SkippedControls,
		report.PeriodStart.Format(time.RFC3339),
		report.PeriodEnd.Format(time.RFC3339),
		report.GeneratedAt.Format(time.RFC3339),
	))

	return b.String()
}

// csvEscape escapes a string for CSV output
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
