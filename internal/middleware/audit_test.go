package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDefaultAuditConfig(t *testing.T) {
	cfg := DefaultAuditConfig()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.ExcludePaths)
	assert.Contains(t, cfg.ExcludePaths, "/health")
	assert.Contains(t, cfg.ExcludePaths, "/ready")
	assert.Contains(t, cfg.ExcludePaths, "/api/v1/audit/events")

	assert.True(t, cfg.LogRequestBody)
	assert.False(t, cfg.LogResponseBody)

	assert.NotEmpty(t, cfg.SanitizeHeaders)
	assert.Contains(t, cfg.SanitizeHeaders, "authorization")
	assert.Contains(t, cfg.SanitizeHeaders, "cookie")

	assert.NotEmpty(t, cfg.SensitiveFields)
	assert.Contains(t, cfg.SensitiveFields, "password")
	assert.Contains(t, cfg.SensitiveFields, "token")
}

func TestSanitizeData(t *testing.T) {
	sensitiveFields := []string{"password", "secret", "token"}

	tests := []struct {
		name           string
		input          map[string]interface{}
		expectedOutput map[string]interface{}
	}{
		{
			name: "redacts sensitive fields at top level",
			input: map[string]interface{}{
				"username": "john",
				"password": "secret123",
			},
			expectedOutput: map[string]interface{}{
				"username": "john",
				"password": "***REDACTED***",
			},
		},
		{
			name: "redacts fields case-insensitively",
			input: map[string]interface{}{
				"Password": "secret123",
				"SECRET":    "value",
				"Api_Key":   "key123",
			},
			expectedOutput: map[string]interface{}{
				"Password": "***REDACTED***",
				"SECRET":    "***REDACTED***",
				"Api_Key":   "key123", // Not redacted because "api_key" is not in sensitiveFields
			},
		},
		{
			name: "redacts partial matches",
			input: map[string]interface{}{
				"current_password": "old123",
				"new_password":     "new123",
				"access_token":     "token123",
			},
			expectedOutput: map[string]interface{}{
				"current_password": "***REDACTED***",
				"new_password":     "***REDACTED***",
				"access_token":     "***REDACTED***",
			},
		},
		{
			name: "preserves non-sensitive fields",
			input: map[string]interface{}{
				"username": "john",
				"email":    "john@example.com",
			},
			expectedOutput: map[string]interface{}{
				"username": "john",
				"email":    "john@example.com",
			},
		},
		{
			name: "handles nested objects",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "john",
					"password": "secret123",
				},
			},
			expectedOutput: map[string]interface{}{
				"user": map[string]interface{}{
					"username": "john",
					"password": "***REDACTED***",
				},
			},
		},
		{
			name: "handles arrays",
			input: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"username": "john", "password": "secret1"},
					map[string]interface{}{"username": "jane", "password": "secret2"},
				},
			},
			expectedOutput: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{"username": "john", "password": "***REDACTED***"},
					map[string]interface{}{"username": "jane", "password": "***REDACTED***"},
				},
			},
		},
		{
			name: "handles empty input",
			input: map[string]interface{}{},
			expectedOutput: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := sanitizeData(tt.input, sensitiveFields)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "extracts resource from API v1 path",
			path:     "/api/v1/users",
			expected: "users",
		},
		{
			name:     "extracts resource from API v1 path with ID",
			path:     "/api/v1/credentials/123",
			expected: "credentials",
		},
		{
			name:     "extracts resource from nested path",
			path:     "/api/v1/sessions/123/commands",
			expected: "sessions",
		},
		{
			name:     "handles path without trailing slash",
			path:     "/api/v1/tenants",
			expected: "tenants",
		},
		{
			name:     "handles path with trailing slash",
			path:     "/api/v1/roles/",
			expected: "roles",
		},
		{
			name:     "returns unknown for malformed path",
			path:     "/api",
			expected: "unknown",
		},
		{
			name:     "returns unknown for root path",
			path:     "/",
			expected: "unknown",
		},
		{
			name:     "returns unknown for empty path",
			path:     "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getResourceType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetResourceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		setupCtx func(*gin.Context)
		expected string
	}{
		{
			name: "extracts id from path params",
			setupCtx: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "id", Value: "12345"}}
			},
			expected: "12345",
		},
		{
			name: "extracts user_id from path params",
			setupCtx: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "user_id", Value: "user-123"}}
			},
			expected: "user-123",
		},
		{
			name: "extracts role_id from path params",
			setupCtx: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "role_id", Value: "role-456"}}
			},
			expected: "role-456",
		},
		{
			name: "extracts tenant_id from path params",
			setupCtx: func(c *gin.Context) {
				c.Params = gin.Params{{Key: "tenant_id", Value: "tenant-789"}}
			},
			expected: "tenant-789",
		},
		{
			name:     "returns empty string when no params",
			setupCtx: func(c *gin.Context) {},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &gin.Context{}
			tt.setupCtx(c)
			result := getResourceID(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUUID(t *testing.T) {
	validUUID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		input    interface{}
		expected uuid.UUID
	}{
		{
			name:     "returns uuid.UUID input",
			input:    validUUID,
			expected: validUUID,
		},
		{
			name:     "parses valid UUID string",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: validUUID,
		},
		{
			name:     "returns Nil for invalid UUID string",
			input:    "not-a-uuid",
			expected: uuid.Nil,
		},
		{
			name:     "returns Nil for empty string",
			input:    "",
			expected: uuid.Nil,
		},
		{
			name:     "returns Nil for unsupported type",
			input:    12345,
			expected: uuid.Nil,
		},
		{
			name:     "returns Nil for nil",
			input:    nil,
			expected: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestZeroizeBytes(t *testing.T) {
	t.Run("zeroizes byte slice", func(t *testing.T) {
		data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		zeroizeBytes(data)

		for _, b := range data {
			assert.Equal(t, byte(0), b)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		data := []byte{}
		zeroizeBytes(data) // Should not panic
		assert.Len(t, data, 0)
	})

	t.Run("handles single byte", func(t *testing.T) {
		data := []byte{0xAA}
		zeroizeBytes(data)

		assert.Equal(t, byte(0), data[0])
	})
}

func TestVolatileBytes(t *testing.T) {
	t.Run("does not panic on non-zero first byte", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02}
		volatileBytes(data) // First byte is 0, should not execute
	})

	t.Run("does not panic on zero byte", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03}
		volatileBytes(data) // First byte is not 42, should not panic
	})
}

func TestAuditMiddleware_WithNilService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zerolog.Nop()
	cfg := DefaultAuditConfig()

	router := gin.New()
	// Pass nil for audit service - should only log to zerolog
	router.Use(AuditMiddleware(nil, logger, cfg))
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("handles nil audit service gracefully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestAuditOnlyLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := zerolog.Nop()
	cfg := DefaultAuditConfig()

	router := gin.New()
	router.Use(AuditOnlyLogs(logger, cfg))
	router.GET("/api/v1/users", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.DELETE("/api/v1/users/123", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("tenant_id", "tenant-456")
		c.Status(http.StatusNoContent)
	})

	t.Run("logs non-excluded paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("identifies privileged actions", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/users/123", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
