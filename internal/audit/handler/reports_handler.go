package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/pam/analytics"
	"github.com/rs/zerolog"
)

// ReportsHandler handles report-related HTTP requests
type ReportsHandler struct {
	service *analytics.Service
	logger  zerolog.Logger
}

// NewReportsHandler creates a new reports handler
func NewReportsHandler(service *analytics.Service, logger zerolog.Logger) *ReportsHandler {
	return &ReportsHandler{
		service: service,
		logger:  logger,
	}
}

// =============================================================================
// Report Snapshot Handlers
// =============================================================================

// GenerateReport handles POST /api/v1/reports/generate
func (h *ReportsHandler) GenerateReport(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	var req analytics.GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	snapshot, err := h.service.GenerateReportSnapshot(c.Request.Context(), tenantID, userID.(uuid.UUID), req.ReportID, req.PeriodStart, req.PeriodEnd, string(req.Format))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate report"}})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"snapshot_id": snapshot.ID,
		"status":      snapshot.Status,
		"message":     "Report generation started",
	})
}

// ListReportSnapshots handles GET /api/v1/reports/snapshots
func (h *ReportsHandler) ListReportSnapshots(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Parse query parameters
	filter := analytics.ReportSnapshotFilter{
		TenantID: tenantID,
		Limit:    parseIntQuery(c, "limit", 50),
		Offset:   parseIntQuery(c, "offset", 0),
	}

	if reportID := c.Query("report_id"); reportID != "" {
		if id, err := uuid.Parse(reportID); err == nil {
			filter.ReportID = &id
		}
	}

	filter.Framework = c.Query("framework")
	filter.Status = c.Query("status")

	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = t
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = t
		}
	}

	snapshots, total, err := h.service.ListReportSnapshots(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report snapshots")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list reports"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshots": snapshots,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})
}

// GetReportSnapshot handles GET /api/v1/reports/snapshots/:id
func (h *ReportsHandler) GetReportSnapshot(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	snapshotID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid snapshot ID"}})
		return
	}

	snapshot, err := h.service.GetReportSnapshot(c.Request.Context(), snapshotID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report snapshot")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report snapshot not found"}})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// DownloadReportSnapshot handles GET /api/v1/reports/snapshots/:id/download
func (h *ReportsHandler) DownloadReportSnapshot(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	snapshotID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid snapshot ID"}})
		return
	}

	snapshot, err := h.service.GetReportSnapshot(c.Request.Context(), snapshotID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report snapshot")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report snapshot not found"}})
		return
	}

	if snapshot.Status != analytics.ReportSnapshotStatusCompleted {
		c.JSON(http.StatusAccepted, gin.H{
			"error": gin.H{"code": "REPORT_PENDING", "message": "Report is still being generated"},
			"status": snapshot.Status,
		})
		return
	}

	if snapshot.FileURL == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "FILE_NOT_AVAILABLE", "message": "Report file not available"}})
		return
	}

	// Generate signed URL for download (15 minute expiry)
	expiry := time.Now().Add(15 * time.Minute)
	signedURL, err := generateSignedDownloadURL(*snapshot.FileURL, snapshotID, tenantID, expiry)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate signed URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to generate download URL"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"download_url": signedURL,
		"expires_at":   expiry.Format(time.RFC3339),
		"filename":     fmt.Sprintf("%s.%s", snapshot.SnapshotName, *snapshot.FileFormat),
	})
}

// GetReportSnapshotStats handles GET /api/v1/reports/snapshots/stats
func (h *ReportsHandler) GetReportSnapshotStats(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.service.GetReportSnapshotStats(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report snapshot stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get stats"}})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// =============================================================================
// Report Generation Job Handlers
// =============================================================================

// ListReportJobs handles GET /api/v1/reports/jobs
func (h *ReportsHandler) ListReportJobs(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Parse query parameters
	filter := analytics.ReportJobFilter{
		TenantID: tenantID,
		Limit:    parseIntQuery(c, "limit", 50),
		Offset:   parseIntQuery(c, "offset", 0),
	}

	filter.Status = c.Query("status")
	filter.JobType = c.Query("job_type")

	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse(time.RFC3339, dateFrom); err == nil {
			filter.DateFrom = t
		}
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse(time.RFC3339, dateTo); err == nil {
			filter.DateTo = t
		}
	}

	jobs, total, err := h.service.ListReportJobs(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list jobs"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetReportJob handles GET /api/v1/reports/jobs/:id
func (h *ReportsHandler) GetReportJob(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid job ID"}})
		return
	}

	job, err := h.service.GetReportJob(c.Request.Context(), jobID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report job")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report job not found"}})
		return
	}

	c.JSON(http.StatusOK, job)
}

// CancelReportJob handles POST /api/v1/reports/jobs/:id/cancel
func (h *ReportsHandler) CancelReportJob(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid job ID"}})
		return
	}

	err = h.service.CancelReportJob(c.Request.Context(), jobID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to cancel report job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to cancel job"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled"})
}

// RetryReportJob handles POST /api/v1/reports/jobs/:id/retry
func (h *ReportsHandler) RetryReportJob(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid job ID"}})
		return
	}

	err = h.service.RetryReportJob(c.Request.Context(), jobID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to retry report job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to retry job"}})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Job queued for retry"})
}

// =============================================================================
// Report Schedule Handlers
// =============================================================================

// CreateReportSchedule handles POST /api/v1/reports/schedules
func (h *ReportsHandler) CreateReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
		return
	}

	var req analytics.ReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	schedule, err := h.service.CreateReportSchedule(c.Request.Context(), tenantID, userID.(uuid.UUID), userID.(uuid.UUID), &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create schedule"}})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// ListReportSchedules handles GET /api/v1/reports/schedules
func (h *ReportsHandler) ListReportSchedules(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Parse query parameters
	filter := analytics.ReportScheduleFilter{
		TenantID: tenantID,
		Limit:    parseIntQuery(c, "limit", 50),
		Offset:   parseIntQuery(c, "offset", 0),
	}

	filter.Status = c.Query("status")
	filter.Framework = c.Query("framework")

	if ownedBy := c.Query("owned_by"); ownedBy != "" {
		if id, err := uuid.Parse(ownedBy); err == nil {
			filter.OwnedBy = &id
		}
	}

	schedules, total, err := h.service.ListReportSchedules(c.Request.Context(), tenantID, filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list report schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list schedules"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"total":     total,
		"limit":     filter.Limit,
		"offset":    filter.Offset,
	})
}

// GetReportSchedule handles GET /api/v1/reports/schedules/:id
func (h *ReportsHandler) GetReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	schedule, err := h.service.GetReportSchedule(c.Request.Context(), scheduleID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get report schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report schedule not found"}})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// UpdateReportSchedule handles PUT /api/v1/reports/schedules/:id
func (h *ReportsHandler) UpdateReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	var req analytics.ReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_REQUEST", "message": err.Error()}})
		return
	}

	err = h.service.UpdateReportSchedule(c.Request.Context(), scheduleID, tenantID, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to update report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update schedule"}})
		return
	}

	// Fetch the updated schedule to return
	schedule, err := h.service.GetReportSchedule(c.Request.Context(), scheduleID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get updated schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to retrieve schedule"}})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// DeleteReportSchedule handles DELETE /api/v1/reports/schedules/:id
func (h *ReportsHandler) DeleteReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	err = h.service.DeleteReportSchedule(c.Request.Context(), scheduleID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete schedule"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted"})
}

// PauseReportSchedule handles POST /api/v1/reports/schedules/:id/pause
func (h *ReportsHandler) PauseReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	err = h.service.PauseReportSchedule(c.Request.Context(), scheduleID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to pause report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to pause schedule"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule paused"})
}

// ResumeReportSchedule handles POST /api/v1/reports/schedules/:id/resume
func (h *ReportsHandler) ResumeReportSchedule(c *gin.Context) {
	tenantID, err := reportsGetTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	scheduleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid schedule ID"}})
		return
	}

	err = h.service.ResumeReportSchedule(c.Request.Context(), scheduleID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to resume report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to resume schedule"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule resumed"})
}

// =============================================================================
// Helper Functions
// =============================================================================

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// generateSignedDownloadURL generates an HMAC-signed URL for secure report downloads.
// The URL includes the original file URL, snapshot ID, tenant ID, and expiry timestamp,
// all protected by an HMAC-SHA256 signature using the REPORT_SIGNING_KEY.
func generateSignedDownloadURL(fileURL string, snapshotID, tenantID uuid.UUID, expiry time.Time) (string, error) {
	signingKey := os.Getenv("REPORT_SIGNING_KEY")
	if signingKey == "" {
		signingKey = os.Getenv("EVENT_SIGNING_KEY")
	}
	if signingKey == "" {
		// In development, fall back to the raw URL
		return fileURL, nil
	}

	expiryUnix := fmt.Sprintf("%d", expiry.Unix())

	// Build the message to sign: fileURL|snapshotID|tenantID|expiry
	message := fmt.Sprintf("%s|%s|%s|%s", fileURL, snapshotID, tenantID, expiryUnix)

	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Append signature parameters to the URL
	separator := "?"
	if len(fileURL) > 0 {
		for _, c := range fileURL {
			if c == '?' {
				separator = "&"
				break
			}
		}
	}

	signedURL := fmt.Sprintf("%s%ssnapshot_id=%s&tenant_id=%s&expires=%s&signature=%s",
		fileURL, separator, snapshotID, tenantID, expiryUnix, signature)

	return signedURL, nil
}

func reportsGetTenantID(c *gin.Context) (uuid.UUID, error) {
	tenantIDStr := c.GetHeader("X-Tenant-ID")
	if tenantIDStr == "" {
		// Try from context
		if tenantID, exists := c.Get("tenant_id"); exists {
			if id, ok := tenantID.(uuid.UUID); ok {
				return id, nil
			}
			if idStr, ok := tenantID.(string); ok {
				return uuid.Parse(idStr)
			}
		}
		return uuid.Nil, fmt.Errorf("tenant ID not found")
	}
	return uuid.Parse(tenantIDStr)
}
