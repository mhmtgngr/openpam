package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Sensitive field names that should never be logged.
// SECURITY: This list prevents credential leakage in structured logs.
var sensitiveFields = map[string]bool{
	"password":               true,
	"current_password":       true,
	"new_password":           true,
	"old_password":           true,
	"confirm_password":       true,
	"token":                  true,
	"api_token":              true,
	"access_token":           true,
	"refresh_token":          true,
	"auth_token":             true,
	"session_token":          true,
	"secret":                 true,
	"api_secret":             true,
	"client_secret":          true,
	"shared_secret":          true,
	"webhook_secret":         true,
	"private_key":            true,
	"private_key_pem":        true,
	"ssh_private_key":        true,
	"credential_value":       true,
	"credential_password":    true,
	"credential":             true,
	"credit_card":            true,
	"ssn":                    true,
	"social_security_number": true,
	"pin":                    true,
	"mfa_code":               true,
	"mfa_token":              true,
	"totp_secret":            true,
	"recovery_code":          true,
	"backup_code":            true,
	"answer":                 true,
	"credential_id":          true,
	"checkout_token":         true,
	"recording_key":          true,
	"vault_key":              true,
	"encryption_key":         true,
}

// Sensitive endpoints that should never have their bodies logged.
// SECURITY: Authentication endpoints are excluded entirely from body logging.
var sensitiveEndpoints = map[string]bool{
	"/api/v1/auth/login":    true,
	"/api/v1/auth/register": true,
	"/api/v1/auth/logout":   true,
	"/api/v1/auth/refresh":  true,
	"/api/v1/auth/verify":   true,
	"/api/v1/auth/forgot":   true,
	"/api/v1/auth/reset":    true,
	"/api/v1/auth/password": true,
	"/api/v1/credentials":   true,
	"/api/v1/vault/secrets": true,
	"/api/v1/sessions/start": true,
}

// sanitizeRequestBody removes sensitive fields from request body before logging.
// SECURITY: This prevents credentials, tokens, and PII from leaking in logs.
func sanitizeRequestBody(body []byte) map[string]interface{} {
	if len(body) == 0 {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	for key, value := range data {
		if sensitiveFields[key] {
			sanitized[key] = "[REDACTED]"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			sanitized[key] = sanitizeNestedMap(nestedMap)
		} else {
			sanitized[key] = value
		}
	}

	return sanitized
}

func sanitizeNestedMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range m {
		if sensitiveFields[key] {
			result[key] = "[REDACTED]"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			result[key] = sanitizeNestedMap(nestedMap)
		} else if arr, ok := value.([]interface{}); ok {
			var sanitizedArray []interface{}
			for _, item := range arr {
				if nestedItem, ok := item.(map[string]interface{}); ok {
					sanitizedArray = append(sanitizedArray, sanitizeNestedMap(nestedItem))
				} else {
					sanitizedArray = append(sanitizedArray, item)
				}
			}
			result[key] = sanitizedArray
		} else {
			result[key] = value
		}
	}
	return result
}

// AuditLog creates an audit log entry.
// SECURITY FIX: Request body logging is sanitized to prevent credential leakage.
func AuditLog(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

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

		// SECURITY FIX: Do NOT log request bodies for sensitive endpoints
		if sensitiveEndpoints[c.Request.URL.Path] {
			event.Str("request_body", "[REDACTED_SENSITIVE_ENDPOINT]")
		} else if c.Request.Body != nil && c.Request.Method != "GET" {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil && len(bodyBytes) > 0 {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				sanitized := sanitizeRequestBody(bodyBytes)
				if sanitized != nil && len(sanitized) > 0 {
					event.Interface("request_body", sanitized)
				}
			}
		}

		event.Msg("Audit log entry")
	}
}

// SanitizedRequestBodyKey is the context key for storing sanitized request bodies
const SanitizedRequestBodyKey = "sanitized_body"

// SanitizeRequestBody reads and sanitizes the request body before processing.
// SECURITY: This middleware ensures sensitive credential data is never logged.
func SanitizeRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" && c.Request.Method != "PUT" && c.Request.Method != "PATCH" {
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		sanitized := sanitizeRequestBody(bodyBytes)
		if sanitized != nil {
			c.Set(SanitizedRequestBodyKey, sanitized)
		}

		c.Next()
	}
}

// GetSanitizedBody retrieves the sanitized request body from context.
// SECURITY: Always use this for logging instead of reading the raw request body.
func GetSanitizedBody(c *gin.Context) map[string]interface{} {
	if sanitized, exists := c.Get(SanitizedRequestBodyKey); exists {
		if body, ok := sanitized.(map[string]interface{}); ok {
			return body
		}
	}
	return nil
}
