package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"net/http"
)

// Config holds middleware configuration
type Config struct {
	TrustProxy      bool
	AllowedOrigins  []string
	AllowedMethods  []string
	AllowedHeaders  []string
	ExposeHeaders   []string
	MaxRequestBody  int64
	EnableRequestID bool
}

// Logger returns a gin middleware for structured logging
func Logger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Extract request ID or generate new
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		// Process request
		c.Next()

		// Build log entry
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

		// Add tenant ID if present
		if tenantID, exists := c.Get("tenant_id"); exists {
			event.Str("tenant_id", tenantID.(string))
		}

		// Add user ID if present
		if userID, exists := c.Get("user_id"); exists {
			event.Str("user_id", userID.(string))
		}

		// Log errors at error level
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

// CORS returns a gin middleware for CORS handling
func CORS(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range cfg.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			if cfg.AllowedOrigins[0] == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			} else {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}

		c.Header("Access-Control-Allow-Methods", join(cfg.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", join(cfg.AllowedHeaders, ", "))
		c.Header("Access-Control-Expose-Headers", join(cfg.ExposeHeaders, ", "))
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeaders adds security-related headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

// BodyLimit limits the size of request body
func BodyLimit(maxBody int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBody {
			c.JSON(413, gin.H{
				"error": gin.H{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request body exceeds maximum allowed size",
				},
			})
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
		c.Next()
	}
}

// ContentType validates JSON content type for write operations
func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == "POST" || method == "PUT" || method == "PATCH" {
			ct := c.Request.Header.Get("Content-Type")
			if ct != "application/json" {
				c.JSON(415, gin.H{
					"error": gin.H{
						"code":    "UNSUPPORTED_MEDIA_TYPE",
						"message": "Content-Type must be application/json",
					},
				})
				c.Abort()
				return
			}
		}
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

// Tenant extracts tenant from context and adds it to gin context
func Tenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if exists {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	}
}

// Auth validates JWT token and extracts user info
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing authorization header",
				},
			})
			c.Abort()
			return
		}

		// Token validation is handled by auth package
		// This middleware checks if user is set by auth handler
		if _, exists := c.Get("user_id"); !exists {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User not authenticated",
				},
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		allowed := false
		for _, r := range roles {
			if r == role {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireMFA checks if user has completed MFA verification
func RequireMFA() gin.HandlerFunc {
	return func(c *gin.Context) {
		mfaVerified, exists := c.Get("mfa_verified")
		if !exists || !mfaVerified.(bool) {
			c.JSON(403, gin.H{
				"error": gin.H{
					"code":    "MFA_REQUIRED",
					"message": "Multi-factor authentication required",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuditLog creates an audit log entry
func AuditLog(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// Only log mutating operations
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return
		}

		requestID, _ := c.Get("request_id")
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")
		duration := time.Since(start)

		event := logger.Info().
			Str("request_id", requestID.(string)).
			Str("action", method+" "+c.Request.URL.Path).
			Str("user_id", userID.(string)).
			Str("tenant_id", tenantID.(string)).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("ip", c.ClientIP())

		// Add request body for audit (sanitize sensitive data)
		if c.Request.Body != nil && c.Request.Method != "GET" {
			body, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			// Don't log passwords
			var sanitized map[string]interface{}
			_ = json.Unmarshal(body, &sanitized)
			delete(sanitized, "password")
			delete(sanitized, "current_password")
			delete(sanitized, "new_password")
			if len(sanitized) > 0 {
				event.Interface("request_body", sanitized)
			}
		}

		event.Msg("Audit log entry")
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

		// Validate UUID format
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

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
