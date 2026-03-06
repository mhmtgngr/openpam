package middleware

import (
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Logger returns a gin middleware for structured logging.
// SECURITY: Request bodies are NEVER logged to prevent credential leakage.
// For audit trails with sanitized request bodies, use AuditLog middleware.
func Logger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()
		ip := c.ClientIP()

		event := logger.Info().
			Str("request_id", requestID).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("duration", duration).
			Int("size", size).
			Str("ip", ip).
			Str("user_agent", c.Request.UserAgent())

		if tenantID, exists := c.Get("tenant_id"); exists {
			event.Str("tenant_id", tenantID.(string))
		}
		if userID, exists := c.Get("user_id"); exists {
			event.Str("user_id", userID.(string))
		}

		if status >= 400 {
			event = logger.Error()
			if len(c.Errors) > 0 {
				event.Str("errors", c.Errors.String())
			}
		}

		event.Msg("HTTP request")
	}
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Recovery returns a gin middleware for panic recovery
func Recovery(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				logger.Error().
					Str("request_id", requestID.(string)).
					Interface("panic", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Bytes("stack", debug.Stack()).
					Msg("Panic recovered")

				c.JSON(500, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An internal error occurred",
					},
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// ErrorHandler centralizes error handling
func ErrorHandler(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID, _ := c.Get("request_id")

			logger.Error().
				Str("request_id", requestID.(string)).
				Err(err.Err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Request error")

			if !c.Writer.Written() {
				c.JSON(500, gin.H{
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An internal error occurred",
					},
				})
			}
		}
	}
}

// ValidateRequestID ensures X-Request-ID header is present
func ValidateRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			c.JSON(400, gin.H{
				"error": gin.H{
					"code":    "MISSING_REQUEST_ID",
					"message": "X-Request-ID header is required",
				},
			})
			c.Abort()
			return
		}

		if _, err := uuid.Parse(requestID); err != nil {
			c.JSON(400, gin.H{
				"error": gin.H{
					"code":    "INVALID_REQUEST_ID",
					"message": "X-Request-ID must be a valid UUID",
				},
			})
			c.Abort()
			return
		}

		c.Set("request_id", requestID)
		c.Next()
	}
}
