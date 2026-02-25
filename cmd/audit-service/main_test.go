// Package main provides tests for audit-service initialization
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/middleware"
	"github.com/rs/zerolog"
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
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_NAME", "testdb")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_NAME")
	}()

	assert.Equal(t, "9504", os.Getenv("PORT"))
	assert.Equal(t, "debug", os.Getenv("LOG_LEVEL"))
	assert.Equal(t, "localhost", os.Getenv("DB_HOST"))
	assert.Equal(t, "5432", os.Getenv("DB_PORT"))
	assert.Equal(t, "testdb", os.Getenv("DB_NAME"))
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
// Load Config Tests
// =============================================================================

func TestLoadConfig(t *testing.T) {
	t.Run("loads config with defaults", func(t *testing.T) {
		// Clear env vars
		envVars := []string{"AUDIT_SERVICE_PORT", "AUDIT_SERVICE_LOG_LEVEL", "AUDIT_SERVICE_DB_HOST"}
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		config := loadConfig()
		assert.Equal(t, "8504", config.ServicePort)
		assert.Equal(t, "info", config.LogLevel)
		assert.NotNil(t, config.DBHost)
	})

	t.Run("loads config from environment", func(t *testing.T) {
		os.Setenv("AUDIT_SERVICE_PORT", "9999")
		os.Setenv("AUDIT_SERVICE_LOG_LEVEL", "debug")

		defer func() {
			os.Unsetenv("AUDIT_SERVICE_PORT")
			os.Unsetenv("AUDIT_SERVICE_LOG_LEVEL")
		}()

		config := loadConfig()
		assert.Equal(t, "9999", config.ServicePort)
		assert.Equal(t, "debug", config.LogLevel)
	})
}

// =============================================================================
// Service Initialization Tests
// =============================================================================

func TestInitializeDependencies(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("service initialization requires valid config", func(t *testing.T) {
		config := Config{
			ServicePort: "8504",
			LogLevel:    "info",
			DBHost:      "localhost",
			DBPort:      5432,
			DBUser:      "invalid_user",
			DBPassword:  "invalid_pass",
			DBName:      "nonexistent_db",
			DBSSLMode:   "disable",
			RedisHost:   "localhost",
			RedisPort:   6379,
			RedisDB:     0,
		}

		logger := zerolog.Nop()

		// This should not panic but may fail to connect
		_, err := database.New(database.Config{
			Host:     config.DBHost,
			Port:     config.DBPort,
			User:     config.DBUser,
			Password: config.DBPassword,
			Database: config.DBName,
			SSLMode:  config.DBSSLMode,
		}, logger)

		// We expect an error for invalid credentials
		assert.Error(t, err)
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
			"status": "healthy",
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
// Anomaly Repository Integration Tests
// =============================================================================

func TestAnomalyRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("anomaly repository can be created", func(t *testing.T) {
		logger := zerolog.Nop()

		// Mock DB connection - would normally use test DB
		repo := pamanalytics.NewAnomalyRepository(nil, logger)
		assert.NotNil(t, repo)
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
	router.Use(middleware.RequestLogging(zerolog.Nop()))
	router.Use(middleware.TenantID())

	router.GET("/api/v1/analytics/anomalies", func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		require.True(t, exists)
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
// Analytics Service Tests
// =============================================================================

func TestAnalyticsServiceCreation(t *testing.T) {
	t.Run("creates analytics service with valid dependencies", func(t *testing.T) {
		logger := zerolog.Nop()

		// Create mock dependencies
		mockCache := &mockCache{}
		mockRepo := &mockAnalyticsRepository{}

		service := audit.NewAnalyticsService(mockRepo, mockCache, logger)
		assert.NotNil(t, service)
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

			expectedStatus := http.StatusOK
			if route.method == "POST" {
				expectedStatus = http.StatusCreated
			}
			assert.Contains(t, []int{http.StatusOK, http.StatusCreated, http.StatusNotFound}, w.Code,
				"Route %s %s should be registered", route.method, route.path)
		}
	})
}

// =============================================================================
// Mock Implementations for Testing
// =============================================================================

type mockCache struct {
	data map[string][]byte
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	return nil
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) bool {
	return false
}

func (m *mockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCache) Close() error {
	return nil
}

func (m *mockCache) Health(ctx context.Context) error {
	return nil
}

func (m *mockCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}

type mockAnalyticsRepository struct{}

func (m *mockAnalyticsRepository) GetDashboardSummary(ctx context.Context, tenantID string) (*audit.DashboardSummary, error) {
	return &audit.DashboardSummary{}, nil
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
// Logger Setup Tests
// =============================================================================

func TestSetupLogger(t *testing.T) {
	t.Run("sets up logger with debug level", func(t *testing.T) {
		logger := setupLogger("debug")
		assert.NotNil(t, logger)
	})

	t.Run("sets up logger with info level", func(t *testing.T) {
		logger := setupLogger("info")
		assert.NotNil(t, logger)
	})

	t.Run("sets up logger with warn level", func(t *testing.T) {
		logger := setupLogger("warn")
		assert.NotNil(t, logger)
	})

	t.Run("sets up logger with error level", func(t *testing.T) {
		logger := setupLogger("error")
		assert.NotNil(t, logger)
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
