package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
)

// AnalyticsHandler handles analytics HTTP requests
type AnalyticsHandler struct {
	service analytics.AnalyticsService
	logger  zerolog.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service analytics.AnalyticsService, logger zerolog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		logger:  logger,
	}
}

// GetSessionMetrics retrieves session metrics for a date range
func (h *AnalyticsHandler) GetSessionMetrics(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Parse date range
	dateFrom, dateTo := h.parseDateRange(c)

	metrics, err := h.service.GetSessionMetrics(c.Request.Context(), tenantIDUUID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get session metrics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get session metrics"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

// GetSessionTrends retrieves session trends for time-series visualization
func (h *AnalyticsHandler) GetSessionTrends(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	metric := c.Query("metric")
	if metric == "" {
		metric = "sessions"
	}

	dateFrom, dateTo := h.parseDateRange(c)

	data, err := h.service.GetTimeSeriesData(c.Request.Context(), tenantIDUUID, metric, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get session trends")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get session trends"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetSessionSummary retrieves session summary statistics
func (h *AnalyticsHandler) GetSessionSummary(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)

	summary, err := h.service.GetSessionMetrics(c.Request.Context(), tenantIDUUID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get session summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get session summary"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// GetUserActivity retrieves activity for a specific user
func (h *AnalyticsHandler) GetUserActivity(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)

	activities, err := h.service.GetUserActivity(c.Request.Context(), tenantIDUUID, userID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get user activity"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// GetUserRiskScore retrieves risk score for a user
func (h *AnalyticsHandler) GetUserRiskScore(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}})
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	riskScore, err := h.service.GetUserRiskScore(c.Request.Context(), tenantIDUUID, userID, days)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user risk score")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get risk score"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"risk_score": riskScore,
		"days":       days,
	})
}

// GetTopUsers retrieves users with highest activity/risk
func (h *AnalyticsHandler) GetTopUsers(c *gin.Context) {
	_, _ = c.Get("tenant_id")
	c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"code": "NOT_IMPLEMENTED", "message": "Top users query not yet implemented"}})
}

// GetCommandFrequency retrieves command frequency data
func (h *AnalyticsHandler) GetCommandFrequency(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)
	limit := getIntQuery(c, "limit", 100)

	commands, err := h.service.GetCommandFrequency(c.Request.Context(), tenantIDUUID, dateFrom, dateTo, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get command frequency")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get command frequency"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"commands": commands})
}

// GetTopCommands retrieves top commands by frequency
func (h *AnalyticsHandler) GetTopCommands(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)
	limit := getIntQuery(c, "limit", 20)

	commands, err := h.service.GetCommandFrequency(c.Request.Context(), tenantIDUUID, dateFrom, dateTo, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get top commands")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get top commands"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"top_commands": commands})
}

// Command blacklist handlers

func (h *AnalyticsHandler) GetCommandBlacklist(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	blacklists, err := h.service.ListCommandBlacklist(c.Request.Context(), &tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get command blacklist"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": blacklists})
}

func (h *AnalyticsHandler) CreateCommandBlacklist(c *gin.Context) {
	var req analytics.CommandBlacklist
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	req.TenantID = &tenantIDUUID
	req.CreatedBy = userIDUUID

	if err := h.service.CreateCommandBlacklist(c.Request.Context(), &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to create command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to create blacklist entry"}})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"blacklist": req})
}

func (h *AnalyticsHandler) UpdateCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	var req analytics.CommandBlacklist
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	req.ID = id

	if err := h.service.UpdateCommandBlacklist(c.Request.Context(), &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update blacklist entry"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"blacklist": req})
}

func (h *AnalyticsHandler) DeleteCommandBlacklist(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid blacklist ID"}})
		return
	}

	if err := h.service.DeleteCommandBlacklist(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to delete command blacklist")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to delete blacklist entry"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blacklist entry deleted"})
}

// Dashboard handlers

func (h *AnalyticsHandler) GetDashboard(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	metrics, err := h.service.GetDashboardMetrics(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get dashboard metrics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get dashboard metrics"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": metrics})
}

func (h *AnalyticsHandler) GetTimeSeries(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	metric := c.Query("metric")
	if metric == "" {
		metric = "sessions"
	}

	dateFrom, dateTo := h.parseDateRange(c)

	data, err := h.service.GetTimeSeriesData(c.Request.Context(), tenantIDUUID, metric, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get time series data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get time series data"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// SSH Key Analytics

func (h *AnalyticsHandler) GetSSHKeyAnalytics(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	sshKeyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_KEY", "message": "Invalid SSH key ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)

	analyticsData, err := h.service.GetSSHKeyAnalytics(c.Request.Context(), tenantIDUUID, sshKeyID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get SSH key analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get SSH key analytics"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analyticsData})
}

// Cache management handlers (admin only)

func (h *AnalyticsHandler) InvalidateCache(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	// Get cache types from query
	var cacheTypes []string
	if types := c.Query("types"); types != "" {
		// Parse comma-separated types
		// In production, use proper parsing
		cacheTypes = []string{types}
	}

	if err := h.service.InvalidateCache(c.Request.Context(), tenantIDUUID, cacheTypes...); err != nil {
		h.logger.Error().Err(err).Msg("Failed to invalidate cache")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to invalidate cache"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache invalidated"})
}

func (h *AnalyticsHandler) WarmCache(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)

	if err := h.service.WarmCache(c.Request.Context(), tenantIDUUID, dateFrom, dateTo); err != nil {
		h.logger.Error().Err(err).Msg("Failed to warm cache")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to warm cache"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cache warming initiated"})
}

func (h *AnalyticsHandler) GetCacheStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	stats, err := h.service.GetCacheStats(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get cache stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get cache stats"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Helper methods

func (h *AnalyticsHandler) parseDateRange(c *gin.Context) (time.Time, time.Time) {
	// Default to last 7 days
	dateTo := time.Now().Truncate(24 * time.Hour)
	dateFrom := dateTo.AddDate(0, 0, -7)

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			dateFrom = t
		}
	}

	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			dateTo = t
		}
	}

	return dateFrom, dateTo
}

// GetSessions retrieves sessions with filtering
func (h *AnalyticsHandler) GetSessions(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)
	limit := getIntQuery(c, "limit", 50)
	offset := getIntQuery(c, "offset", 0)

	metrics, err := h.service.GetSessionMetrics(c.Request.Context(), tenantIDUUID, dateFrom, dateTo)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get sessions"}})
		return
	}

	// Return metrics with pagination info
	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetCommands retrieves command frequency data
func (h *AnalyticsHandler) GetCommands(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	dateFrom, dateTo := h.parseDateRange(c)
	limit := getIntQuery(c, "limit", 100)

	commands, err := h.service.GetCommandFrequency(c.Request.Context(), tenantIDUUID, dateFrom, dateTo, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get command frequency")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get command frequency"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"commands": commands})
}

// IngestEvent ingests events from other services
func (h *AnalyticsHandler) IngestEvent(c *gin.Context) {
	var req struct {
		Type      string                 `json:"type" binding:"required"`
		TenantID  string                 `json:"tenant_id" binding:"required"`
		UserID    string                 `json:"user_id" binding:"required"`
		Data      map[string]interface{} `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	// Convert to event
	event := events.Event{
		Type:     req.Type,
		TenantID: req.TenantID,
		ActorID:  req.UserID,
		Data:     req.Data,
	}

	ctx := c.Request.Context()

	// Handle based on event type
	switch req.Type {
	case "analytics.session.started":
		_ = h.service.HandleSessionStarted(ctx, event)
	case "analytics.session.ended":
		_ = h.service.HandleSessionEnded(ctx, event)
	case "analytics.command.executed":
		_ = h.service.HandleCommandExecuted(ctx, event)
	default:
		h.logger.Warn().Str("type", req.Type).Msg("Unknown event type")
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Event ingested"})
}
