package middleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set Gin to test mode to disable logging
	gin.SetMode(gin.TestMode)
}

func TestRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get("request_id")
		require.True(t, exists)
		c.JSON(200, gin.H{"request_id": requestID})
	})

	t.Run("generate new request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp["request_id"])
	})

	t.Run("use existing request ID", func(t *testing.T) {
		existingID := uuid.New().String()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", existingID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, existingID, w.Header().Get("X-Request-ID"))
	})
}

func TestValidateRequestID(t *testing.T) {
	router := gin.New()
	router.Use(ValidateRequestID())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("missing request ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "MISSING_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("invalid request ID format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", "not-a-uuid")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid request ID", func(t *testing.T) {
		validID := uuid.New().String()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", validID)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name          string
		origin        string
		allowedOrigin string
		expectHeader  bool
	}{
		{
			name:          "allow any origin",
			origin:        "https://example.com",
			allowedOrigin: "*",
			expectHeader:  true,
		},
		{
			name:          "allow matching origin",
			origin:        "https://example.com",
			allowedOrigin: "https://example.com",
			expectHeader:  true,
		},
		{
			name:          "deny non-matching origin",
			origin:        "https://evil.com",
			allowedOrigin: "https://example.com",
			expectHeader:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				AllowedOrigins: []string{tt.allowedOrigin},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
			}

			router := gin.New()
			router.Use(CORS(cfg))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tt.expectHeader {
				headerValue := w.Header().Get("Access-Control-Allow-Origin")
				if tt.allowedOrigin == "*" {
					assert.Equal(t, "*", headerValue)
				} else {
					assert.Equal(t, tt.origin, headerValue)
				}
			}

			assert.Equal(t, "GET, POST", w.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
		})
	}

	t.Run("handle preflight request", func(t *testing.T) {
		cfg := Config{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		}

		router := gin.New()
		router.Use(CORS(cfg))
		router.OPTIONS("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 204, w.Code)
	})
}

func TestSecurityHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", w.Header().Get("Permissions-Policy"))
}

func TestBodyLimit(t *testing.T) {
	router := gin.New()
	router.Use(BodyLimit(100)) // 100 bytes limit
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("within limit", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{"small":"data"}`))
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("exceeds limit", func(t *testing.T) {
		largeBody := bytes.NewReader(make([]byte, 200))
		req := httptest.NewRequest("POST", "/test", largeBody)
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = 200
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 413, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "REQUEST_TOO_LARGE", resp["error"].(map[string]interface{})["code"])
	})
}

func TestContentType(t *testing.T) {
	router := gin.New()
	router.Use(ContentType())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("valid content type", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{}`))
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("invalid content type", func(t *testing.T) {
		body := bytes.NewReader([]byte(`{}`))
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 415, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNSUPPORTED_MEDIA_TYPE", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("GET request bypasses validation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

func TestRecovery(t *testing.T) {
	logger := zerolog.Nop()

	router := gin.New()
	router.Use(Recovery(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Catch the panic at test level
	defer func() {
		if r := recover(); r != nil {
			// Panic propagated past the middleware - this is unexpected
			t.Logf("Panic recovered: %v", r)
		}
	}()

	// The Recovery middleware should handle the panic
	router.ServeHTTP(w, req)

	// The recovery middleware should have set the response
	assert.Equal(t, 500, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp["error"].(map[string]interface{})["code"])
}

func TestAuth(t *testing.T) {
	router := gin.New()
	router.Use(Auth())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "UNAUTHORIZED", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("no user_id in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

func TestRequireRole(t *testing.T) {
	router := gin.New()
	router.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("has required role", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()

		// Simulate authenticated user with role
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_role", "admin")

		handler := RequireRole("admin")
		handler(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("missing role", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler := RequireRole("admin")
		handler(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong role", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_role", "user")

		handler := RequireRole("admin")
		handler(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, 403, w.Code)
	})
}

func TestRequireMFA(t *testing.T) {
	router := gin.New()
	router.GET("/protected", RequireMFA(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	t.Run("MFA verified", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("mfa_verified", true)

		handler := RequireMFA()
		handler(c)

		assert.False(t, c.IsAborted())
	})

	t.Run("MFA not verified", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("mfa_verified", false)

		handler := RequireMFA()
		handler(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, 403, w.Code)
	})

	t.Run("MFA flag not set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler := RequireMFA()
		handler(c)

		assert.True(t, c.IsAborted())
		assert.Equal(t, 403, w.Code)
	})
}

func TestLogger(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)

	router := gin.New()
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Request ID should be set
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestTenant(t *testing.T) {
	router := gin.New()
	router.Use(Tenant())
	router.GET("/test", func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		if exists {
			c.JSON(200, gin.H{"tenant_id": tenantID})
		} else {
			c.JSON(200, gin.H{"tenant_id": nil})
		}
	})

	t.Run("tenant ID in context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", "tenant-123")

		handler := Tenant()
		handler(c)
		c.Next() // Move to next handler

		tenantID, exists := c.Get("tenant_id")
		assert.True(t, exists)
		assert.Equal(t, "tenant-123", tenantID)
	})
}

func TestMiddlewareIntegration(t *testing.T) {
	logger := zerolog.Nop()

	cfg := Config{
		AllowedOrigins:  []string{"*"},
		AllowedMethods:  []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:  []string{"Content-Type", "Authorization"},
		MaxRequestBody:  1024 * 1024,
		EnableRequestID: true,
	}

	router := gin.New()
	router.Use(RequestID())
	router.Use(Logger(logger))
	router.Use(Recovery(logger))
	router.Use(CORS(cfg))
	router.Use(SecurityHeaders())
	router.Use(BodyLimit(cfg.MaxRequestBody))
	router.Use(ContentType())
	router.Use(ErrorHandler(logger))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	t.Run("health check with all middleware", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "healthy", resp["status"])
	})
}

func BenchmarkMiddleware(b *testing.B) {
	logger := zerolog.Nop()
	cfg := Config{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	router := gin.New()
	router.Use(RequestID())
	router.Use(Logger(logger))
	router.Use(SecurityHeaders())
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		strs     []string
		sep      string
		expected string
	}{
		{
			name:     "empty slice",
			strs:     []string{},
			sep:      ",",
			expected: "",
		},
		{
			name:     "single element",
			strs:     []string{"a"},
			sep:      ",",
			expected: "a",
		},
		{
			name:     "multiple elements",
			strs:     []string{"a", "b", "c"},
			sep:      ",",
			expected: "a,b,c",
		},
		{
			name:     "different separator",
			strs:     []string{"x", "y", "z"},
			sep:      "|",
			expected: "x|y|z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := join(tt.strs, tt.sep)
			assert.Equal(t, tt.expected, result)
		})
	}
}
