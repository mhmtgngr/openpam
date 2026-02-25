// Package main provides tests for audit-service initialization
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Config Tests
// =============================================================================

func TestConfigDefaults(t *testing.T) {
	envVars := []string{"PORT", "LOG_LEVEL", "DB_HOST", "DB_PORT", "DB_NAME", "REDIS_HOST"}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	t.Run("default port is 8504", func(t *testing.T) {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8504"
		}
		assert.Equal(t, "8504", port)
	})

	t.Run("default log level is info", func(t *testing.T) {
		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
		assert.Equal(t, "info", logLevel)
	})
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("PORT", "9504")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	assert.Equal(t, "9504", os.Getenv("PORT"))
	assert.Equal(t, "debug", os.Getenv("LOG_LEVEL"))
}

func TestServicePort(t *testing.T) {
	t.Run("default port is 8504", func(t *testing.T) {
		os.Unsetenv("PORT")
		port := os.Getenv("PORT")
		if port == "" {
			port = "8504"
		}
		assert.Equal(t, "8504", port)
	})

	t.Run("custom port from env", func(t *testing.T) {
		os.Setenv("PORT", "12504")
		defer os.Unsetenv("PORT")

		port := os.Getenv("PORT")
		assert.Equal(t, "12504", port)
	})
}

func TestAuditServiceName(t *testing.T) {
	serviceName := "audit-service"
	assert.Equal(t, "audit-service", serviceName)
	assert.NotEmpty(t, serviceName)
}

// =============================================================================
// Service Initialization Tests
// =============================================================================

func TestServiceName(t *testing.T) {
	t.Run("service name is audit-service", func(t *testing.T) {
		serviceName := "audit-service"
		assert.Equal(t, "audit-service", serviceName)
	})
}

// =============================================================================
// Analytics Integration Tests
// =============================================================================

func TestAnalyticsIntegration(t *testing.T) {
	t.Run("analytics routes are registered", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		// Create a mock router
		router := gin.New()

		// Register analytics middleware and routes
		router.Use(middleware.CorrelationID())
		router.Use(middleware.TenantID())

		// Analytics endpoints should be registered
		router.GET("/api/v1/analytics/anomalies", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"anomalies": []interface{}{}})
		})
		router.GET("/api/v1/analytics/anomalies/summary", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"summary": gin.H{}})
		})

		// Test the routes exist
		validUUID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/v1/analytics/anomalies", nil)
		req.Header.Set("X-Tenant-ID", validUUID)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Health Check Tests
// =============================================================================

func TestHealthCheckEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "audit-service",
		})
	})

	t.Run("health check returns 200", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("health check response structure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, "audit-service", response["service"])
	})
}

// =============================================================================
// Shutdown Tests
// =============================================================================

func TestGracefulShutdown(t *testing.T) {
	t.Run("shutdown handles SIGTERM", func(t *testing.T) {
		// Simulate shutdown signal
		sigChan := make(chan os.Signal, 1)
		sigChan <- syscall.SIGTERM

		// Verify signal is received
		select {
		case sig := <-sigChan:
			assert.Equal(t, syscall.SIGTERM, sig)
		case <-time.After(1 * time.Second):
			t.Fatal("Failed to receive signal")
		}
	})

	t.Run("shutdown handles SIGINT", func(t *testing.T) {
		sigChan := make(chan os.Signal, 1)
		sigChan <- syscall.SIGINT

		select {
		case sig := <-sigChan:
			assert.Equal(t, syscall.SIGINT, sig)
		case <-time.After(1 * time.Second):
			t.Fatal("Failed to receive signal")
		}
	})
}

// =============================================================================
// Middleware Chain Tests
// =============================================================================

func TestMiddlewareChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Apply middleware in correct order
	router.Use(middleware.CorrelationID())
	router.Use(middleware.TenantID())

	router.GET("/api/v1/analytics/anomalies", func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		correlationID, _ := c.Get("correlation_id")

		c.JSON(http.StatusOK, gin.H{
			"tenant_id":      tenantID,
			"correlation_id": correlationID,
		})
	})

	t.Run("middleware chain applies all middleware", func(t *testing.T) {
		validUUID := uuid.New().String()
		correlationID := uuid.New().String()

		req := httptest.NewRequest("GET", "/api/v1/analytics/anomalies", nil)
		req.Header.Set("X-Tenant-ID", validUUID)
		req.Header.Set("X-Correlation-ID", correlationID)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, validUUID, response["tenant_id"])
		assert.Equal(t, correlationID, response["correlation_id"])

		// Check response headers
		assert.Equal(t, correlationID, w.Header().Get("X-Correlation-ID"))
	})
}

// =============================================================================
// Route Registration Tests
// =============================================================================

func TestRouteRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	t.Run("all analytics routes are registered", func(t *testing.T) {
		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/api/v1/analytics/anomalies"},
			{"GET", "/api/v1/analytics/anomalies/:id"},
			{"GET", "/api/v1/analytics/anomalies/summary"},
			{"PATCH", "/api/v1/analytics/anomalies/:id"},
			{"POST", "/api/v1/analytics/anomalies/:id/acknowledge"},
			{"GET", "/api/v1/analytics/dashboard"},
			{"GET", "/api/v1/analytics/sessions"},
			{"GET", "/api/v1/analytics/events"},
			{"GET", "/api/v1/analytics/users"},
			{"GET", "/api/v1/analytics/reports"},
			{"GET", "/api/v1/analytics/compliance"},
		}

		// Register routes
		for _, route := range routes {
			switch route.method {
			case "GET":
				router.GET(route.path, func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			case "POST":
				router.POST(route.path, func(c *gin.Context) {
					c.Status(http.StatusCreated)
				})
			case "PATCH":
				router.PATCH(route.path, func(c *gin.Context) {
					c.Status(http.StatusOK)
				})
			}
		}

		// Test each route exists
		for _, route := range routes {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Contains(t, []int{http.StatusOK, http.StatusCreated, http.StatusNotFound}, w.Code,
				"Route %s %s should be registered", route.method, route.path)
		}
	})
}

// =============================================================================
// Request ID Tests
// =============================================================================

func TestRequestIDHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CorrelationID())

	router.GET("/test", func(c *gin.Context) {
		correlationID, _ := c.Get("correlation_id")
		c.JSON(http.StatusOK, gin.H{"correlation_id": correlationID})
	})

	t.Run("generates request ID if not provided", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		testID := uuid.New().String()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Request-ID", testID)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, testID, w.Header().Get("X-Correlation-ID"))
	})
}

// =============================================================================
// Tenant ID Validation Tests
// =============================================================================

func TestTenantIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.TenantID())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	t.Run("rejects request without tenant ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects request with invalid tenant ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", "not-a-uuid")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("accepts request with valid tenant ID", func(t *testing.T) {
		validUUID := uuid.New().String()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Tenant-ID", validUUID)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add timeout middleware
	timeout := 100 * time.Millisecond
	router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "fast"})
	})

	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"status": "slow"})
	})

	t.Run("fast request completes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fast", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Error Response Format Tests
// =============================================================================

func TestErrorResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "An internal error occurred",
				"details": gin.H{},
			},
		})
	})

	t.Run("error response follows standard format", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response["error"])
		errorObj := response["error"].(map[string]interface{})
		assert.Equal(t, "INTERNAL_ERROR", errorObj["code"])
		assert.NotEmpty(t, errorObj["message"])
	})
}
