package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit"
	"github.com/rs/zerolog"
)

// AuditHandler handles audit endpoints
type AuditHandler struct {
	service *audit.Service
	logger  zerolog.Logger
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(service *audit.Service, logger zerolog.Logger) *AuditHandler {
	return &AuditHandler{
		service: service,
		logger:  logger,
	}
}

// List returns a paginated list of audit events
func (h *AuditHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

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

	// Build filter
	filter := audit.EventFilter{}
	if actorID := c.Query("actor_id"); actorID != "" {
		if uid, err := uuid.Parse(actorID); err == nil {
			filter.ActorID = &uid
		}
	}
	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if outcome := c.Query("outcome"); outcome != "" {
		o := audit.EventOutcome(outcome)
		filter.Outcome = &o
	}
	if from := c.Query("from"); from != "" {
		if startTime, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartTimeFrom = &startTime
		}
	}
	if to := c.Query("to"); to != "" {
		if endTime, err := time.Parse(time.RFC3339, to); err == nil {
			filter.StartTimeTo = &endTime
		}
	}

	events, total, err := h.service.Query(c.Request.Context(), tenantIDUUID, filter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to list audit events")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_AUDIT_EVENTS_FAILED",
				"message": "Failed to list audit events",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Get retrieves a single audit event by ID
func (h *AuditHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_EVENT_ID",
				"message": "Invalid event ID",
			},
		})
		return
	}

	// Use the service to get event - need to add GetByID method to service
	// For now, query by tenant with ID filter
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	filter := audit.EventFilter{}
	limit, offset := 1, 0
	events, _, err := h.service.Query(c.Request.Context(), tenantIDUUID, filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GET_AUDIT_EVENT_FAILED",
				"message": "Failed to get audit event",
			},
		})
		return
	}

	// Find the matching event
	for _, event := range events {
		if event.ID == id {
			c.JSON(http.StatusOK, gin.H{"event": event})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"code":    "EVENT_NOT_FOUND",
			"message": "Audit event not found",
		},
	})
}

// Export exports audit events in JSON format
func (h *AuditHandler) Export(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Build filter from query parameters
	filter := audit.EventFilter{}
	if from := c.Query("from"); from != "" {
		if startTime, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartTimeFrom = &startTime
		}
	}
	if to := c.Query("to"); to != "" {
		if endTime, err := time.Parse(time.RFC3339, to); err == nil {
			filter.StartTimeTo = &endTime
		}
	}

	// Get all matching events (use high limit)
	events, _, err := h.service.Query(c.Request.Context(), tenantIDUUID, filter, 10000, 0)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to export audit events")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "EXPORT_AUDIT_EVENTS_FAILED",
				"message": "Failed to export audit events",
			},
		})
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=audit_export.json")

	c.JSON(http.StatusOK, gin.H{
		"events":   events,
		"total":    len(events),
		"exported": time.Now().UTC().Format(time.RFC3339),
	})
}

// VerifyIntegrity verifies the integrity of the audit log chain
func (h *AuditHandler) VerifyIntegrity(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	valid, errors, err := h.service.VerifyIntegrity(c.Request.Context(), tenantIDUUID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to verify audit integrity")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "VERIFY_INTEGRITY_FAILED",
				"message": "Failed to verify audit integrity",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":          valid,
		"errors":         errors,
		"total_errors":   len(errors),
		"verified_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

// GenerateComplianceReport generates a compliance report for a date range
func (h *AuditHandler) GenerateComplianceReport(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Parse date range
	var startTime, endTime time.Time
	var err error

	if from := c.Query("from"); from != "" {
		startTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_DATE_RANGE",
					"message": "Invalid start date format",
				},
			})
			return
		}
	} else {
		// Default to last 30 days
		startTime = time.Now().AddDate(0, 0, -30)
	}

	if to := c.Query("to"); to != "" {
		endTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "INVALID_DATE_RANGE",
					"message": "Invalid end date format",
				},
			})
			return
		}
	} else {
		endTime = time.Now()
	}

	report, err := h.service.GenerateComplianceReport(c.Request.Context(), tenantIDUUID, startTime, endTime)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to generate compliance report")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GENERATE_REPORT_FAILED",
				"message": "Failed to generate compliance report",
			},
		})
		return
	}

	// Verify integrity
	valid, integrityErrors, _ := h.service.VerifyIntegrity(c.Request.Context(), tenantIDUUID)
	report.IntegrityValid = valid
	report.IntegrityErrors = integrityErrors

	c.JSON(http.StatusOK, gin.H{"report": report})
}

// GetStats returns audit statistics
func (h *AuditHandler) GetStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDUUID, _ := uuid.Parse(tenantID.(string))

	// Generate report for last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	report, err := h.service.GenerateComplianceReport(c.Request.Context(), tenantIDUUID, startTime, endTime)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get audit stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "GET_STATS_FAILED",
				"message": "Failed to get audit statistics",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": gin.H{
			"total_events":      report.TotalEvents,
			"successful_events": report.SuccessfulEvents,
			"failed_events":     report.FailedEvents,
			"denied_events":     report.DeniedEvents,
			"period": gin.H{
				"start": startTime.Format(time.RFC3339),
				"end":   endTime.Format(time.RFC3339),
			},
		},
	})
}
