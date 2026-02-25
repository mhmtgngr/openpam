// Package middleware provides analytics-specific middleware
package middleware

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

const (
	// Query timeout for analytics queries
	defaultQueryTimeout = 30 * time.Second

	// Max results per page
	maxResultsPerPage = 1000
)

// AnalyticsConfig holds configuration for analytics middleware
type AnalyticsConfig struct {
	QueryTimeout      time.Duration
	MaxResultsPerPage int
	EnableProfiling   bool
}

// DefaultAnalyticsConfig returns default analytics middleware configuration
func DefaultAnalyticsConfig() AnalyticsConfig {
	return AnalyticsConfig{
		QueryTimeout:      defaultQueryTimeout,
		MaxResultsPerPage: maxResultsPerPage,
		EnableProfiling:   false,
	}
}

// TenantID middleware ensures tenant_id is present in context
func TenantID() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			// Try from query parameter
			tenantID = c.Query("tenant_id")
		}

		if tenantID == "" {
			c.JSON(400, gin.H{"error": gin.H{"code": "MISSING_TENANT_ID", "message": "X-Tenant-ID header is required"}})
			c.Abort()
			return
		}

		if _, err := uuid.Parse(tenantID); err != nil {
			c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_TENANT_ID", "message": "Invalid tenant ID format"}})
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

// RequireTenantID is an alias for TenantID for backward compatibility
func RequireTenantID() gin.HandlerFunc {
	return TenantID()
}

// Pagination adds pagination parameters to context
func Pagination(config AnalyticsConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := config.MaxResultsPerPage
		offset := 0

		// Parse limit
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := parseLimit(limitStr, config.MaxResultsPerPage); err == nil {
				limit = parsedLimit
			}
		}

		// Parse offset
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if parsedOffset, err := parseOffset(offsetStr); err == nil {
				offset = parsedOffset
			}
		}

		// Parse cursor if provided (for cursor-based pagination)
		var cursor string
		if cursorStr := c.Query("cursor"); cursorStr != "" {
			cursor = cursorStr
		}

		c.Set("pagination", gin.H{
			"limit":  limit,
			"offset": offset,
			"cursor": cursor,
		})

		c.Next()
	}
}

// DateRange adds date range parameters to context with validation
func DateRange() gin.HandlerFunc {
	return func(c *gin.Context) {
		var dateFrom, dateTo *time.Time
		var err error

		// Parse date_from
		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			var t time.Time
			t, err = time.Parse(time.RFC3339, dateFromStr)
			if err != nil {
				t, err = time.Parse("2006-01-02", dateFromStr)
			}
			if err == nil {
				dateFrom = &t
			}
		}

		// Parse date_to
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			var t time.Time
			t, err = time.Parse(time.RFC3339, dateToStr)
			if err != nil {
				t, err = time.Parse("2006-01-02", dateToStr)
			}
			if err == nil {
				dateTo = &t
			}
		}

		// Validate date range
		if dateFrom != nil && dateTo != nil && dateTo.Before(*dateFrom) {
			c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_DATE_RANGE", "message": "date_to must be after date_from"}})
			c.Abort()
			return
		}

		// Limit range to 1 year
		if dateFrom != nil && dateTo != nil {
			if dateTo.Sub(*dateFrom) > 365*24*time.Hour {
				c.JSON(400, gin.H{"error": gin.H{"code": "DATE_RANGE_TOO_LARGE", "message": "Date range cannot exceed 1 year"}})
				c.Abort()
				return
			}
		}

		c.Set("date_range", gin.H{
			"date_from": dateFrom,
			"date_to":   dateTo,
		})

		c.Next()
	}
}

// ValidateFramework validates compliance framework parameter
func ValidateFramework() gin.HandlerFunc {
	return func(c *gin.Context) {
		if framework := c.Query("framework"); framework != "" {
			if !isValidFramework(framework) {
				c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_FRAMEWORK", "message": "Invalid compliance framework"}})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ValidateSeverity validates severity parameter
func ValidateSeverity() gin.HandlerFunc {
	return func(c *gin.Context) {
		if severity := c.Query("severity"); severity != "" {
			if !isValidSeverity(severity) {
				c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_SEVERITY", "message": "Invalid severity level"}})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ValidateAnomalyStatus validates anomaly status parameter
func ValidateAnomalyStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		if status := c.Query("status"); status != "" {
			if !isValidAnomalyStatus(status) {
				c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_STATUS", "message": "Invalid anomaly status"}})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ValidatePatternType validates command pattern type
func ValidatePatternType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if patternType := c.Query("pattern_type"); patternType != "" {
			if !isValidPatternType(patternType) {
				c.JSON(400, gin.H{"error": gin.H{"code": "INVALID_PATTERN_TYPE", "message": "Invalid pattern type"}})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// QueryTimeout sets a timeout for long-running queries
func QueryTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Next()
		}()

		select {
		case <-done:
			// Request completed normally
		case <-ctx.Done():
			// Timeout occurred
			c.JSON(408, gin.H{"error": gin.H{"code": "QUERY_TIMEOUT", "message": "Query exceeded time limit"}})
			c.Abort()
		}
	}
}

// RequestLogging logs analytics requests with additional context
func RequestLogging(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Log request
		logger.Info().
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("query", c.Request.URL.RawQuery).
			Str("tenant_id", c.GetHeader("X-Tenant-ID")).
			Str("user_agent", c.Request.UserAgent()).
			Str("client_ip", c.ClientIP()).
			Msg("Analytics request")

		// Process request
		c.Next()

		// Log response
		duration := time.Since(start)
		logger.Info().
			Str("request_id", requestID).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Msg("Analytics response")
	}
}

// CorrelationID adds/ensures correlation ID for request tracing
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = c.GetHeader("X-Request-ID")
		}
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		c.Header("X-Correlation-ID", correlationID)
		c.Set("correlation_id", correlationID)

		c.Next()
	}
}

// Profiling records query performance metrics
func Profiling(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record initial memory stats
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()

		c.Next()

		// Record final memory stats
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		duration := time.Since(start)

		// Log performance metrics for slow queries
		if duration > 500*time.Millisecond {
			logger.Warn().
				Str("path", c.Request.URL.Path).
				Dur("duration", duration).
				Uint64("alloc_bytes", m2.TotalAlloc-m1.TotalAlloc).
				Msg("Slow analytics query")
		}
	}
}

// CacheControl adds cache control headers for analytics endpoints
func CacheControl(maxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", fmt.Sprintf("max-age=%d", int(maxAge.Seconds())))
		c.Next()
	}
}

// RequireAdmin requires admin role for certain operations
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"}})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok || (role != "admin" && role != "super_admin") {
			c.JSON(403, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Admin role required"}})
			c.Abort()
			return
		}

		c.Next()
	}
}

// =============================================================================
// Validation Helper Functions
// =============================================================================

func isValidFramework(framework string) bool {
	validFrameworks := []string{
		model.FrameworkSOC2,
		model.FrameworkISO27001,
		model.FrameworkPCIDSS,
		model.FrameworkHIPAA,
		model.FrameworkNIST,
		model.FrameworkGDPR,
		model.FrameworkCustom,
	}
	for _, f := range validFrameworks {
		if framework == f {
			return true
		}
	}
	return false
}

func isValidSeverity(severity string) bool {
	validSeverities := []string{
		string(model.SeverityLow),
		string(model.SeverityMedium),
		string(model.SeverityHigh),
		string(model.SeverityCritical),
	}
	for _, s := range validSeverities {
		if severity == s {
			return true
		}
	}
	return false
}

func isValidAnomalyStatus(status string) bool {
	validStatuses := []string{
		string(model.AnomalyStatusOpen),
		string(model.AnomalyStatusInvestigating),
		string(model.AnomalyStatusResolved),
		string(model.AnomalyStatusFalsePositive),
		string(model.AnomalyStatusIgnored),
	}
	for _, s := range validStatuses {
		if status == s {
			return true
		}
	}
	return false
}

func isValidPatternType(patternType string) bool {
	validTypes := []string{
		string(model.PatternTypeExact),
		string(model.PatternTypeRegex),
		string(model.PatternTypeGlob),
	}
	for _, t := range validTypes {
		if patternType == t {
			return true
		}
	}
	return false
}

func parseLimit(limitStr string, maxLimit int) (int, error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, err
	}
	if limit < 0 {
		return 0, fmt.Errorf("limit must be positive")
	}
	if limit > maxLimit {
		return maxLimit, nil
	}
	return limit, nil
}

func parseOffset(offsetStr string) (int, error) {
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, err
	}
	if offset < 0 {
		return 0, fmt.Errorf("offset must be non-negative")
	}
	return offset, nil
}
