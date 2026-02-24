package handlers

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// Helper functions for handlers

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

func parseDateRange(c *gin.Context) (time.Time, time.Time) {
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

func parseTenantID(c *gin.Context) (uuid.UUID, *gin.H) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, ok := tenantID.(string)
	if !ok {
		return uuid.Nil, &gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}
	}
	tenantIDUUID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, &gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}
	}
	return tenantIDUUID, nil
}

func parseUserID(c *gin.Context) (uuid.UUID, *gin.H) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, &gin.H{"code": "INVALID_USER", "message": "Invalid user ID"}
	}
	return userID, nil
}

type HandlerBase struct {
	service *analytics.Service
	logger  zerolog.Logger
}

func NewHandlerBase(service *analytics.Service, logger zerolog.Logger) *HandlerBase {
	return &HandlerBase{
		service: service,
		logger:  logger,
	}
}
