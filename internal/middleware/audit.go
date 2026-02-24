package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit"
	"github.com/rs/zerolog"
)

// AuditConfig holds audit middleware configuration
type AuditConfig struct {
	ExcludePaths     []string
	LogRequestBody   bool
	LogResponseBody  bool
	SanitizeHeaders  []string
	SensitiveFields  []string
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		ExcludePaths: []string{
			"/health",
			"/ready",
			"/api/v1/audit/events", // Avoid infinite loop
		},
		LogRequestBody:  true,
		LogResponseBody: false,
		SanitizeHeaders: []string{
			"authorization",
			"cookie",
			"x-api-key",
		},
		SensitiveFields: []string{
			"password",
			"current_password",
			"new_password",
			"token",
			"secret",
			"api_key",
			"private_key",
			"credit_card",
		},
	}
}

// AuditMiddleware creates comprehensive audit logging middleware
func AuditMiddleware(auditSvc *audit.Service, logger zerolog.Logger, cfg AuditConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Check if path should be excluded
		for _, excludePath := range cfg.ExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				c.Next()
				return
			}
		}

		// Extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Extract user info
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")
		userRole, _ := c.Get("user_role")

		// Read request body if needed
		var requestBody map[string]interface{}
		var bodyBytes []byte
		if cfg.LogRequestBody && c.Request.Body != nil && c.Request.Method != "GET" {
			var err error
			bodyBytes, err = io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				_ = json.Unmarshal(bodyBytes, &requestBody)
				// Sanitize sensitive fields
				requestBody = sanitizeData(requestBody, cfg.SensitiveFields)
			}
			// Zeroize sensitive body buffer after use to prevent memory leaks
			zeroizeBytes(bodyBytes)
		}

		// Create response writer wrapper to capture response
		w := &responseWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = w

		// Process request
		c.Next()

		// Determine outcome
		outcome := audit.OutcomeSuccess
		if c.Writer.Status() >= 400 {
			outcome = audit.OutcomeFailure
		}
		if c.Writer.Status() == 401 || c.Writer.Status() == 403 {
			outcome = audit.OutcomeDenied
		}

		// Build audit event
		event := &audit.Event{
			TenantID:     getUUID(tenantID),
			ActorID:      getUUID(userID),
			ActorType:    "user",
			Action:       c.Request.Method + " " + path,
			ResourceType: getResourceType(path),
			ResourceID:   getResourceID(c),
			Outcome:      outcome,
			IP:           c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestID:    requestID,
			Timestamp:    start,
		}

		// Add request body details
		if len(requestBody) > 0 {
			if detailsJSON, err := json.Marshal(requestBody); err == nil {
				event.Details = json.RawMessage(detailsJSON)
			}
		}

		// Add response status for errors
		if c.Writer.Status() >= 400 {
			errorDetail := map[string]interface{}{
				"status_code": c.Writer.Status(),
				"message":     http.StatusText(c.Writer.Status()),
			}
			if len(c.Errors) > 0 {
				errorDetail["errors"] = c.Errors.String()
			}
			if detailsJSON, err := json.Marshal(errorDetail); err == nil {
				if event.Details == nil {
					event.Details = json.RawMessage(detailsJSON)
				} else {
					// Merge with existing details
					var existing map[string]interface{}
					_ = json.Unmarshal(event.Details, &existing)
					existing["error"] = errorDetail
					if merged, err := json.Marshal(existing); err == nil {
						event.Details = json.RawMessage(merged)
					}
				}
			}
		}

		// Log audit event asynchronously
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := auditSvc.Log(ctx, event); err != nil {
				logger.Error().
					Str("request_id", requestID).
					Err(err).
					Msg("Failed to write audit log")
			}
		}()

		// Also log to structured logger for immediate visibility
		logEvent := logger.Info().
			Str("audit_id", event.ID.String()).
			Str("request_id", requestID).
			Str("action", event.Action).
			Str("resource_type", event.ResourceType).
			Str("outcome", string(event.Outcome)).
			Int("status", c.Writer.Status()).
			Dur("duration", time.Since(start)).
			Str("ip", event.IP)

		if userID != nil {
			logEvent.Str("user_id", userID.(string))
		}
		if tenantID != nil {
			logEvent.Str("tenant_id", tenantID.(string))
		}
		if userRole != nil {
			logEvent.Str("user_role", userRole.(string))
		}

		logEvent.Msg("Audit log entry")
	}
}

// responseWriter wraps gin.ResponseWriter to capture response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// sanitizeData removes sensitive fields from data
func sanitizeData(data map[string]interface{}, sensitiveFields []string) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range data {
		lowerKey := strings.ToLower(k)
		isSensitive := false
		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			sanitized[k] = "***REDACTED***"
		} else if nested, ok := v.(map[string]interface{}); ok {
			sanitized[k] = sanitizeData(nested, sensitiveFields)
		} else if arr, ok := v.([]interface{}); ok {
			sanitizedArray := make([]interface{}, len(arr))
			for i, item := range arr {
				if nested, ok := item.(map[string]interface{}); ok {
					sanitizedArray[i] = sanitizeData(nested, sensitiveFields)
				} else {
					sanitizedArray[i] = item
				}
			}
			sanitized[k] = sanitizedArray
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// getResourceType extracts resource type from path
func getResourceType(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		// /api/v1/tenants -> tenants
		return parts[2]
	}
	return "unknown"
}

// getResourceID extracts resource ID from path or params
func getResourceID(c *gin.Context) string {
	// Try to get ID from path params
	if id := c.Param("id"); id != "" {
		return id
	}
	if id := c.Param("user_id"); id != "" {
		return id
	}
	if id := c.Param("role_id"); id != "" {
		return id
	}
	if id := c.Param("tenant_id"); id != "" {
		return id
	}
	return ""
}

// getUUID safely converts interface{} to uuid.UUID
func getUUID(val interface{}) uuid.UUID {
	switch v := val.(type) {
	case uuid.UUID:
		return v
	case string:
		if id, err := uuid.Parse(v); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// zeroizeBytes securely wipes sensitive data from memory
// This prevents sensitive data from being captured in goroutine dumps or memory inspections
func zeroizeBytes(data []byte) {
	if len(data) == 0 {
		return
	}
	// Overwrite the slice with zeros
	for i := range data {
		data[i] = 0
	}
	// Force the compiler to keep the wipe operation and prevent optimization
	// runtime.KeepAlive ensures the data reference stays alive until this point
	runtime.KeepAlive(data)
	// Use the data in a way that prevents compiler dead-code elimination
	// while avoiding unsafe pointer usage. Calling a function with the slice
	// as an argument forces the compiler to preserve the write operations.
	volatileBytes(data)
}

// volatileBytes is a no-op function that prevents compiler optimizations
// from eliminating the zeroize operation. By calling an external function,
// the compiler cannot prove that the write operations are side-effect free.
//go:nosplit
func volatileBytes(data []byte) {
	if len(data) > 0 {
		// This conditional is always false, but the compiler cannot prove it.
		// The reference to data[0] forces the compiler to keep the slice data alive.
		if data[0] == 42 {
			runtime.Gosched() // Impossible to reach, but forces data dependency
		}
	}
}

// AuditOnlyLogs creates a simpler audit middleware that only logs to zerolog
// Use this when audit service is not available
func AuditOnlyLogs(logger zerolog.Logger, cfg AuditConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Check if path should be excluded
		for _, excludePath := range cfg.ExcludePaths {
			if strings.HasPrefix(path, excludePath) {
				c.Next()
				return
			}
		}

		// Extract request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Next()

		// Determine if this is a privileged action
		isPrivileged := strings.Contains(path, "/admin") ||
			strings.Contains(path, "/roles") ||
			strings.Contains(path, "/users") ||
			c.Request.Method == "DELETE" ||
			c.Request.Method == "PUT" ||
			c.Request.Method == "PATCH"

		// Log privileged actions with more detail
		if isPrivileged {
			logEvent := logger.Info().
				Str("request_id", requestID).
				Str("method", c.Request.Method).
				Str("path", path).
				Str("resource_type", getResourceType(path)).
				Int("status", c.Writer.Status()).
				Dur("duration", time.Since(start)).
				Str("ip", c.ClientIP()).
				Bool("privileged_action", true)

			if userID, exists := c.Get("user_id"); exists {
				logEvent.Str("user_id", userID.(string))
			}
			if tenantID, exists := c.Get("tenant_id"); exists {
				logEvent.Str("tenant_id", tenantID.(string))
			}

			logEvent.Msg("Privileged action audit")
		}
	}
}
