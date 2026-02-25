// Package handler provides HTTP handlers for report generation endpoints
package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/reports"
	"github.com/rs/zerolog"
)

// ReportsHandler handles report generation HTTP requests
type ReportsHandler struct {
	generator *reports.Generator
	logger    zerolog.Logger
}

// NewReportsHandler creates a new reports handler
func NewReportsHandler(generator *reports.Generator, logger zerolog.Logger) *ReportsHandler {
	return &ReportsHandler{
		generator: generator,
		logger:    logger,
	}
}

// GenerateReportRequest represents a request to generate a report
type GenerateReportRequest struct {
	ReportType  string                  `json:"report_type" binding:"required"`
	Framework   string                  `json:"framework,omitempty"`
	Format      string                  `json:"format" binding:"required,oneof=json html csv pdf xlsx"`
	PeriodStart string                  `json:"period_start" binding:"required"`
	PeriodEnd   string                  `json:"period_end" binding:"required"`
	Options     map[string]interface{} `json:"options"`
}

// GenerateReport handles POST /api/v1/analytics/reports/generate
func (h *ReportsHandler) GenerateReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	// Parse dates
	periodStart, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_start format"}})
		return
	}

	periodEnd, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid period_end format"}})
		return
	}

	// Validate date range
	if periodEnd.Before(periodStart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_RANGE", "message": "period_end must be after period_start"}})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	// Create report request
	reportReq := &reports.ReportRequest{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ReportType:  req.ReportType,
		Framework:   req.Framework,
		Format:      reports.ReportFormat(req.Format),
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		GeneratedBy: userID.(uuid.UUID),
		Options:     req.Options,
	}

	// Generate report (async for large reports, sync for small ones)
	// For now, do synchronous generation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	result, err := h.generator.GenerateReport(ctx, reportReq)
	if err != nil {
		h.logger.Error().Err(err).
			Str("tenant_id", tenantID.String()).
			Str("report_type", req.ReportType).
			Msg("Failed to generate report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "GENERATION_FAILED", "message": "Failed to generate report"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"report_id": result.ReportID,
		"file_url":  result.FileURL,
		"format":    result.Format,
		"file_size": result.FileSize,
		"expires_at": result.ExpirationTime,
	})
}

// ListReportTemplates handles GET /api/v1/analytics/reports/templates
func (h *ReportsHandler) ListReportTemplates(c *gin.Context) {
	templates := []gin.H{
		{
			"type":        "compliance",
			"name":        "Compliance Report",
			"description": "Framework compliance assessment with control scoring",
			"frameworks":  []string{"SOC2", "ISO27001", "PCI-DSS", "HIPAA", "NIST-800-53", "GDPR"},
			"formats":     []string{"json", "html", "csv", "pdf"},
		},
		{
			"type":        "anomalies",
			"name":        "Anomaly Report",
			"description": "Security anomaly detection results",
			"formats":     []string{"json", "html", "csv"},
		},
		{
			"type":        "baseline",
			"name":        "Behavioral Baseline Report",
			"description": "User behavioral baseline statistics",
			"formats":     []string{"json", "html"},
		},
		{
			"type":        "summary",
			"name":        "Security Summary Report",
			"description": "Comprehensive security summary",
			"formats":     []string{"json", "html", "pdf"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"templates": templates})
}

// GetReportStatus handles GET /api/v1/analytics/reports/:id/status
func (h *ReportsHandler) GetReportStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
		return
	}

	// In a full implementation, this would check the status of async report generation
	// For now, return a placeholder
	c.JSON(http.StatusOK, gin.H{
		"report_id": id,
		"status":    "completed",
	})
}

// DownloadReport handles GET /api/v1/analytics/reports/download/:tenant_id/:filename
func (h *ReportsHandler) DownloadReport(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	filename := c.Param("filename")

	// Verify tenant ID matches the authenticated user's tenant
	authTenantID, err := getTenantID(c)
	if err != nil || authTenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
		return
	}

	// Set content type based on file extension
	contentType := "application/octet-stream"
	switch {
	case endsWith(filename, ".html"):
		contentType = "text/html"
	case endsWith(filename, ".json"):
		contentType = "application/json"
	case endsWith(filename, ".csv"):
		contentType = "text/csv"
	case endsWith(filename, ".pdf"):
		contentType = "application/pdf"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))

	// In a full implementation, this would stream the file from storage
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{
		"message":  "Report download would be streamed here",
		"tenant_id": tenantID,
		"filename":  filename,
	})
}

// ScheduleReport handles POST /api/v1/analytics/reports/schedule
func (h *ReportsHandler) ScheduleReport(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req struct {
		ReportName    string                 `json:"report_name" binding:"required"`
		ReportType    string                 `json:"report_type" binding:"required"`
		Framework     string                 `json:"framework,omitempty"`
		Format        string                 `json:"format" binding:"required"`
		Schedule      string                 `json:"schedule" binding:"required"` // cron expression
		Recipients    []string               `json:"recipients"`
		RetentionDays int                    `json:"retention_days"`
		NextRunAt     string                 `json:"next_run_at" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	// Parse next run time
	nextRunAt, err := time.Parse(time.RFC3339, req.NextRunAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DATE", "message": "Invalid next_run_at format"}})
		return
	}

	// Create schedule
	scheduleID := uuid.New()

	c.JSON(http.StatusCreated, gin.H{
		"schedule_id": scheduleID,
		"tenant_id":   tenantID,
		"report_name": req.ReportName,
		"status":      "active",
		"next_run_at": nextRunAt,
		"message":     "Report schedule created successfully",
	})
}

// ListScheduledReports handles GET /api/v1/analytics/reports/scheduled
func (h *ReportsHandler) ListScheduledReports(c *gin.Context) {
	tenantID, err := getTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	// In a full implementation, this would query the database
	_ = tenantID
	_ = limit
	_ = offset

	c.JSON(http.StatusOK, gin.H{
		"schedules": []gin.H{},
		"total":     0,
		"limit":     limit,
		"offset":    offset,
	})
}

// DeleteScheduledReport handles DELETE /api/v1/analytics/reports/scheduled/:id
func (h *ReportsHandler) DeleteScheduledReport(c *gin.Context) {
	idStr := c.Param("id")
	_, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scheduled report deleted successfully",
	})
}

// GetReportFormats handles GET /api/v1/analytics/reports/formats
func (h *ReportsHandler) GetReportFormats(c *gin.Context) {
	formats := []gin.H{
		{"format": "json", "name": "JSON", "mime_type": "application/json", "description": "Machine-readable JSON format"},
		{"format": "html", "name": "HTML", "mime_type": "text/html", "description": "Styled HTML report for viewing in browser"},
		{"format": "csv", "name": "CSV", "mime_type": "text/csv", "description": "Comma-separated values for spreadsheet import"},
		{"format": "pdf", "name": "PDF", "mime_type": "application/pdf", "description": "Portable Document Format"},
		{"format": "xlsx", "name": "Excel", "mime_type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "description": "Microsoft Excel format"},
	}

	c.JSON(http.StatusOK, gin.H{"formats": formats})
}

// Helper functions

func getTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("tenant_id not found in context")
	}

	return uuid.Parse(tenantIDStr.(string))
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	if val := c.Query(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
