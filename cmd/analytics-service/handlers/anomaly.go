package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// AnomalyHandler handles anomaly detection HTTP requests
type AnomalyHandler struct {
	service *analytics.Service
	logger  zerolog.Logger
}

// NewAnomalyHandler creates a new anomaly handler
func NewAnomalyHandler(service *analytics.Service, logger zerolog.Logger) *AnomalyHandler {
	return &AnomalyHandler{
		service: service,
		logger:  logger,
	}
}

// ListAnomalies retrieves anomaly detections with filtering
func (h *AnomalyHandler) ListAnomalies(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	anomalies, err := h.service.ListAnomalies(c.Request.Context(), tenantIDUUID, status)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list anomalies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to list anomalies"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomalies": anomalies})
}

// GetAnomaly retrieves a specific anomaly detection
func (h *AnomalyHandler) GetAnomaly(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
		return
	}

	anomaly, err := h.service.GetAnomaly(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get anomaly")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Anomaly not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"anomaly": anomaly})
}

// RunDetection triggers anomaly detection for a tenant
func (h *AnomalyHandler) RunDetection(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, err := uuid.Parse(tenantID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
		return
	}

	var req struct {
		UserID *uuid.UUID `json:"user_id"`
	}
	c.ShouldBindJSON(&req)

	var detections []analytics.AnomalyDetection

	if req.UserID != nil {
		// Run detection for specific user
		detections, err = h.service.EvaluateUserForAnomalies(c.Request.Context(), tenantIDUUID, *req.UserID)
	} else {
		// Run detection for entire tenant
		detections, err = h.service.RunAnomalyDetection(c.Request.Context(), tenantIDUUID)
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to run anomaly detection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to run detection"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Anomaly detection completed",
		"detections": detections,
		"count":      len(detections),
	})
}

// UpdateStatus updates the status of an anomaly
func (h *AnomalyHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid anomaly ID"}})
		return
	}

	userID, _ := c.Get("user_id")
	userIDUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		Status           string     `json:"status" binding:"required"`
		AssignedTo       *uuid.UUID `json:"assigned_to"`
		ResolutionNotes  *string    `json:"resolution_notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_INPUT", "message": err.Error()}})
		return
	}

	status := analytics.AnomalyStatus(req.Status)

	// Validate status transition
	validStatuses := map[string]bool{
		"open":          true,
		"investigating": true,
		"resolved":      true,
		"false_positive": true,
		"ignored":       true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_STATUS", "message": "Invalid status"}})
		return
	}

	if err := h.service.UpdateAnomalyStatus(c.Request.Context(), id, status, req.AssignedTo, req.ResolutionNotes, &userIDUUID); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update anomaly status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to update status"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Anomaly status updated", "id": id})
}

// GetTrends retrieves anomaly detection trends
func (h *AnomalyHandler) GetTrends(c *gin.Context) {
	_, _ = c.Get("tenant_id")

	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := fmt.Sscanf(d, "%d", &days); err == nil && parsed == 1 && days > 0 && days <= 365 {
			// Use parsed days
		}
	}

	// This would use the anomaly detector's trend analysis
	// For now, return empty trends
	trends := make(map[string]int)

	c.JSON(http.StatusOK, gin.H{
		"trends": trends,
		"period": fmt.Sprintf("%d days", days),
	})
}

// GetUserAnomalies retrieves anomalies for a specific user
func (h *AnomalyHandler) GetUserAnomalies(c *gin.Context) {
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

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err == nil && parsed == 1 {
			// Use parsed limit
		}
	}

	// Get anomalies for user
	anomalies, err := h.service.ListAnomalies(c.Request.Context(), tenantIDUUID, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get user anomalies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get anomalies"}})
		return
	}

	// Filter by user
	var userAnomalies []analytics.AnomalyDetection
	for _, anomaly := range anomalies {
		if anomaly.UserID != nil && *anomaly.UserID == userID {
			userAnomalies = append(userAnomalies, anomaly)
		}
	}

	c.JSON(http.StatusOK, gin.H{"anomalies": userAnomalies})
}

// Ransomware Event handlers

func (h *AnomalyHandler) ListRansomwareEvents(c *gin.Context) {
	_, _ = c.Get("tenant_id")
	_, _ = getIntQuery(c, "limit", 50), getIntQuery(c, "offset", 0)

	// This would query the repository for ransomware events
	// For now, return empty list
	c.JSON(http.StatusOK, gin.H{"events": []interface{}{}})
}

func (h *AnomalyHandler) GetRansomwareEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event ID"}})
		return
	}

	event, err := h.service.GetRansomwareEvent(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get ransomware event")
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Event not found"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"event": event})
}

func (h *AnomalyHandler) TriggerEmergency(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid event ID"}})
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	if err := h.service.TriggerEmergencyResponse(c.Request.Context(), tenantIDUUID, id); err != nil {
		h.logger.Error().Err(err).Msg("Failed to trigger emergency response")
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to trigger emergency response"}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Emergency response triggered"})
}
