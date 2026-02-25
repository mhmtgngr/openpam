// Package middleware provides tests for analytics-specific middleware
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup
// =============================================================================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func performRequest(router *gin.Engine, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// =============================================================================
// TenantID Middleware Tests
// =============================================================================

func TestTenantID_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(TenantID())
	router.GET("/test", func(c *gin.Context) {
		tenantID, exists := c.Get("tenant_id")
		c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "exists": exists})
	})

	validUUID := uuid.New().String()

	t.Run("valid tenant ID from header", func(t *testing.T) {
		headers := map[string]string{"X-Tenant-ID": validUUID}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["exists"].(bool))
		assert.Equal(t, validUUID, response["tenant_id"].(string))
	})

	t.Run("valid tenant ID from query parameter", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?tenant_id="+validUUID, "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["exists"].(bool))
	})

	t.Run("header takes precedence over query parameter", func(t *testing.T) {
		headerUUID := uuid.New().String()
		queryUUID := uuid.New().String()

		headers := map[string]string{"X-Tenant-ID": headerUUID}
		w := performRequest(router, "GET", "/test?tenant_id="+queryUUID, "", headers)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, headerUUID, response["tenant_id"].(string))
	})

	t.Run("missing tenant ID", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])
	})

	t.Run("invalid tenant ID format", func(t *testing.T) {
		headers := map[string]string{"X-Tenant-ID": "not-a-uuid"}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])
	})

	t.Run("empty tenant ID", func(t *testing.T) {
		headers := map[string]string{"X-Tenant-ID": ""}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestRequireTenantID_Alias(t *testing.T) {
	router := setupTestRouter()
	router.Use(RequireTenantID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	validUUID := uuid.New().String()
	headers := map[string]string{"X-Tenant-ID": validUUID}
	w := performRequest(router, "GET", "/test", "", headers)

	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Pagination Middleware Tests
// =============================================================================

func TestPagination_Middleware(t *testing.T) {
	config := DefaultAnalyticsConfig()

	router := setupTestRouter()
	router.Use(Pagination(config))
	router.GET("/test", func(c *gin.Context) {
		pagination, _ := c.Get("pagination")
		c.JSON(http.StatusOK, pagination)
	})

	t.Run("default pagination values", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, config.MaxResultsPerPage, int(response["limit"].(float64)))
		assert.Equal(t, 0, int(response["offset"].(float64)))
		assert.Equal(t, "", response["cursor"])
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?limit=20&offset=40", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, 20, int(response["limit"].(float64)))
		assert.Equal(t, 40, int(response["offset"].(float64)))
	})

	t.Run("limit exceeds maximum", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?limit=5000", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should cap at max
		assert.Equal(t, config.MaxResultsPerPage, int(response["limit"].(float64)))
	})

	t.Run("negative limit", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?limit=-10", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should use default
		assert.Equal(t, config.MaxResultsPerPage, int(response["limit"].(float64)))
	})

	t.Run("negative offset", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?offset=-5", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should use default
		assert.Equal(t, 0, int(response["offset"].(float64)))
	})

	t.Run("with cursor", func(t *testing.T) {
		cursor := "eyJpZCI6IjEyMzQ1Njc4LTEyMzQtNTY3OC0xMjM0LTU2NzgxMjM0NTY3OCJ9"
		w := performRequest(router, "GET", "/test?cursor="+cursor, "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, cursor, response["cursor"])
	})

	t.Run("invalid limit format", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?limit=abc", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should use default
		assert.Equal(t, config.MaxResultsPerPage, int(response["limit"].(float64)))
	})
}

// =============================================================================
// DateRange Middleware Tests
// =============================================================================

func TestDateRange_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(DateRange())
	router.GET("/test", func(c *gin.Context) {
		dateRange, _ := c.Get("date_range")
		c.JSON(http.StatusOK, dateRange)
	})

	t.Run("no date parameters", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Nil(t, response["date_from"])
		assert.Nil(t, response["date_to"])
	})

	t.Run("valid RFC3339 dates", func(t *testing.T) {
		dateFrom := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
		dateTo := time.Now().Format(time.RFC3339)

		w := performRequest(router, "GET", "/test?date_from="+dateFrom+"&date_to="+dateTo, "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response["date_from"])
		assert.NotNil(t, response["date_to"])
	})

	t.Run("valid YYYY-MM-DD dates", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?date_from=2024-01-01&date_to=2024-01-31", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotNil(t, response["date_from"])
		assert.NotNil(t, response["date_to"])
	})

	t.Run("date_to before date_from", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?date_from=2024-01-31&date_to=2024-01-01", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_DATE_RANGE", errorResp["code"])
	})

	t.Run("date range exceeds 1 year", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?date_from=2023-01-01&date_to=2024-02-01", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "DATE_RANGE_TOO_LARGE", errorResp["code"])
	})

	t.Run("exactly 1 year range", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?date_from=2023-01-01&date_to=2024-01-01", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid date format", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?date_from=invalid-date&date_to=2024-01-31", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should not error, just ignore the invalid date
		assert.Nil(t, response["date_from"])
		assert.NotNil(t, response["date_to"])
	})
}

// =============================================================================
// Validation Middleware Tests
// =============================================================================

func TestValidateFramework_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(ValidateFramework())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("valid frameworks", func(t *testing.T) {
		validFrameworks := []string{
			model.FrameworkSOC2,
			model.FrameworkISO27001,
			model.FrameworkPCIDSS,
			model.FrameworkHIPAA,
			model.FrameworkNIST,
			model.FrameworkGDPR,
			model.FrameworkCustom,
		}

		for _, fw := range validFrameworks {
			w := performRequest(router, "GET", "/test?framework="+fw, "", nil)
			assert.Equal(t, http.StatusOK, w.Code, "Framework "+fw+" should be valid")
		}
	})

	t.Run("invalid framework", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?framework=INVALID", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_FRAMEWORK", errorResp["code"])
	})

	t.Run("no framework parameter", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestValidateSeverity_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(ValidateSeverity())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("valid severities", func(t *testing.T) {
		validSeverities := []string{
			string(model.SeverityLow),
			string(model.SeverityMedium),
			string(model.SeverityHigh),
			string(model.SeverityCritical),
		}

		for _, sv := range validSeverities {
			w := performRequest(router, "GET", "/test?severity="+sv, "", nil)
			assert.Equal(t, http.StatusOK, w.Code, "Severity "+sv+" should be valid")
		}
	})

	t.Run("invalid severity", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?severity=INVALID", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_SEVERITY", errorResp["code"])
	})

	t.Run("no severity parameter", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestValidateAnomalyStatus_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(ValidateAnomalyStatus())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("valid statuses", func(t *testing.T) {
		validStatuses := []string{
			string(model.AnomalyStatusOpen),
			string(model.AnomalyStatusInvestigating),
			string(model.AnomalyStatusResolved),
			string(model.AnomalyStatusFalsePositive),
			string(model.AnomalyStatusIgnored),
		}

		for _, st := range validStatuses {
			w := performRequest(router, "GET", "/test?status="+st, "", nil)
			assert.Equal(t, http.StatusOK, w.Code, "Status "+st+" should be valid")
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?status=INVALID", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_STATUS", errorResp["code"])
	})
}

func TestValidatePatternType_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(ValidatePatternType())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("valid pattern types", func(t *testing.T) {
		validTypes := []string{
			string(model.PatternTypeExact),
			string(model.PatternTypeRegex),
			string(model.PatternTypeGlob),
		}

		for _, pt := range validTypes {
			w := performRequest(router, "GET", "/test?pattern_type="+pt, "", nil)
			assert.Equal(t, http.StatusOK, w.Code, "Pattern type "+pt+" should be valid")
		}
	})

	t.Run("invalid pattern type", func(t *testing.T) {
		w := performRequest(router, "GET", "/test?pattern_type=INVALID", "", nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "INVALID_PATTERN_TYPE", errorResp["code"])
	})
}

// =============================================================================
// QueryTimeout Middleware Tests
// =============================================================================

func TestQueryTimeout_Middleware(t *testing.T) {
	router := setupTestRouter()

	// Very short timeout for testing
	router.Use(QueryTimeout(10 * time.Millisecond))
	router.GET("/test", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	t.Run("request completes within timeout", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request exceeds timeout", func(t *testing.T) {
		w := performRequest(router, "GET", "/slow", "", nil)
		assert.Equal(t, http.StatusRequestTimeout, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "QUERY_TIMEOUT", errorResp["code"])
	})
}

// =============================================================================
// CorrelationID Middleware Tests
// =============================================================================

func TestCorrelationID_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(CorrelationID())
	router.GET("/test", func(c *gin.Context) {
		correlationID, _ := c.Get("correlation_id")
		c.JSON(http.StatusOK, gin.H{"correlation_id": correlationID})
	})

	t.Run("correlation ID from header", func(t *testing.T) {
		testID := uuid.New().String()
		headers := map[string]string{"X-Correlation-ID": testID}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, testID, response["correlation_id"].(string))

		// Check response header
		assert.Equal(t, testID, w.Header().Get("X-Correlation-ID"))
	})

	t.Run("correlation ID from request ID header", func(t *testing.T) {
		testID := uuid.New().String()
		headers := map[string]string{"X-Request-ID": testID}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, testID, response["correlation_id"].(string))
	})

	t.Run("generate new correlation ID", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["correlation_id"])

		// Validate it's a UUID
		_, err = uuid.Parse(response["correlation_id"].(string))
		assert.NoError(t, err)

		// Check response header
		assert.NotEmpty(t, w.Header().Get("X-Correlation-ID"))
	})

	t.Run("X-Correlation-ID takes precedence over X-Request-ID", func(t *testing.T) {
		corrID := uuid.New().String()
		reqID := uuid.New().String()
		headers := map[string]string{
			"X-Correlation-ID": corrID,
			"X-Request-ID":     reqID,
		}
		w := performRequest(router, "GET", "/test", "", headers)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, corrID, response["correlation_id"].(string))
	})
}

// =============================================================================
// CacheControl Middleware Tests
// =============================================================================

func TestCacheControl_Middleware(t *testing.T) {
	router := setupTestRouter()

	maxAge := 5 * time.Minute
	router.Use(CacheControl(maxAge))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("sets cache control header", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "max-age=300", w.Header().Get("Cache-Control"))
	})

	t.Run("max-age with seconds", func(t *testing.T) {
		router2 := setupTestRouter()
		router2.Use(CacheControl(30 * time.Second))
		router2.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := performRequest(router2, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "max-age=30", w.Header().Get("Cache-Control"))
	})
}

// =============================================================================
// RequireAdmin Middleware Tests
// =============================================================================

func TestRequireAdmin_Middleware(t *testing.T) {
	router := setupTestRouter()
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("missing user role", func(t *testing.T) {
		w := performRequest(router, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "UNAUTHORIZED", errorResp["code"])
	})

	t.Run("non-admin role", func(t *testing.T) {
		router2 := setupTestRouter()
		router2.Use(func(c *gin.Context) {
			c.Set("user_role", "user")
			c.Next()
		})
		router2.Use(RequireAdmin())
		router2.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := performRequest(router2, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["error"])

		errorResp := response["error"].(map[string]interface{})
		assert.Equal(t, "FORBIDDEN", errorResp["code"])
	})

	t.Run("admin role allowed", func(t *testing.T) {
		router2 := setupTestRouter()
		router2.Use(func(c *gin.Context) {
			c.Set("user_role", "admin")
			c.Next()
		})
		router2.Use(RequireAdmin())
		router2.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := performRequest(router2, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("super_admin role allowed", func(t *testing.T) {
		router2 := setupTestRouter()
		router2.Use(func(c *gin.Context) {
			c.Set("user_role", "super_admin")
			c.Next()
		})
		router2.Use(RequireAdmin())
		router2.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := performRequest(router2, "GET", "/test", "", nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// RequestLogging Middleware Tests
// =============================================================================

func TestRequestLogging_Middleware(t *testing.T) {
	t.Skip("Request logging middleware test requires logger capture")

	// This test would need to capture logger output to verify
	// For now, we just verify it doesn't panic
	router := setupTestRouter()
	logger := zerolog.Nop()
	router.Use(RequestLogging(logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := performRequest(router, "GET", "/test", "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Profiling Middleware Tests
// =============================================================================

func TestProfiling_Middleware(t *testing.T) {
	t.Skip("Profiling middleware test requires memory stats capture")

	router := setupTestRouter()
	logger := zerolog.Nop()
	router.Use(Profiling(logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := performRequest(router, "GET", "/test", "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// =============================================================================
// Config Tests
// =============================================================================

func TestDefaultAnalyticsConfig(t *testing.T) {
	config := DefaultAnalyticsConfig()

	assert.Equal(t, defaultQueryTimeout, config.QueryTimeout)
	assert.Equal(t, maxResultsPerPage, config.MaxResultsPerPage)
	assert.False(t, config.EnableProfiling)
}

func TestAnalyticsConfig_Custom(t *testing.T) {
	config := AnalyticsConfig{
		QueryTimeout:      60 * time.Second,
		MaxResultsPerPage: 500,
		EnableProfiling:   true,
	}

	assert.Equal(t, 60*time.Second, config.QueryTimeout)
	assert.Equal(t, 500, config.MaxResultsPerPage)
	assert.True(t, config.EnableProfiling)
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestParseLimit(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLimit  int
		expected  int
		expectErr bool
	}{
		{"valid limit", "100", 1000, 100, false},
		{"limit exceeds max", "2000", 1000, 1000, false},
		{"negative limit", "-10", 100, 0, true},
		{"zero limit", "0", 100, 0, false},
		{"invalid format", "abc", 100, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLimit(tt.input, tt.maxLimit)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseOffset(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int
		expectErr bool
	}{
		{"valid offset", "100", 100, false},
		{"zero offset", "0", 0, false},
		{"negative offset", "-10", 0, true},
		{"invalid format", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseOffset(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsValidFramework(t *testing.T) {
	validFrameworks := []string{
		model.FrameworkSOC2,
		model.FrameworkISO27001,
		model.FrameworkPCIDSS,
		model.FrameworkHIPAA,
		model.FrameworkNIST,
		model.FrameworkGDPR,
		model.FrameworkCustom,
	}

	for _, fw := range validFrameworks {
		t.Run("valid_"+fw, func(t *testing.T) {
			assert.True(t, isValidFramework(fw))
		})
	}

	t.Run("invalid framework", func(t *testing.T) {
		assert.False(t, isValidFramework("INVALID"))
		assert.False(t, isValidFramework(""))
		assert.False(t, isValidFramework("soc2")) // case sensitive
	})
}

func TestIsValidSeverity(t *testing.T) {
	validSeverities := []string{
		string(model.SeverityLow),
		string(model.SeverityMedium),
		string(model.SeverityHigh),
		string(model.SeverityCritical),
	}

	for _, sv := range validSeverities {
		t.Run("valid_"+sv, func(t *testing.T) {
			assert.True(t, isValidSeverity(sv))
		})
	}

	t.Run("invalid severity", func(t *testing.T) {
		assert.False(t, isValidSeverity("INVALID"))
		assert.False(t, isValidSeverity(""))
		assert.False(t, isValidSeverity("HIGH")) // case sensitive
	})
}

func TestIsValidAnomalyStatus(t *testing.T) {
	validStatuses := []string{
		string(model.AnomalyStatusOpen),
		string(model.AnomalyStatusInvestigating),
		string(model.AnomalyStatusResolved),
		string(model.AnomalyStatusFalsePositive),
		string(model.AnomalyStatusIgnored),
	}

	for _, st := range validStatuses {
		t.Run("valid_"+st, func(t *testing.T) {
			assert.True(t, isValidAnomalyStatus(st))
		})
	}

	t.Run("invalid status", func(t *testing.T) {
		assert.False(t, isValidAnomalyStatus("INVALID"))
		assert.False(t, isValidAnomalyStatus(""))
		assert.False(t, isValidAnomalyStatus("OPEN")) // case sensitive
	})
}

func TestIsValidPatternType(t *testing.T) {
	validTypes := []string{
		string(model.PatternTypeExact),
		string(model.PatternTypeRegex),
		string(model.PatternTypeGlob),
	}

	for _, pt := range validTypes {
		t.Run("valid_"+pt, func(t *testing.T) {
			assert.True(t, isValidPatternType(pt))
		})
	}

	t.Run("invalid type", func(t *testing.T) {
		assert.False(t, isValidPatternType("INVALID"))
		assert.False(t, isValidPatternType(""))
		assert.False(t, isValidPatternType("EXACT")) // case sensitive
	})
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestMiddleware_Chain(t *testing.T) {
	router := setupTestRouter()

	validUUID := uuid.New().String()

	// Apply multiple middlewares
	router.Use(TenantID())
	router.Use(Pagination(DefaultAnalyticsConfig()))
	router.Use(DateRange())
	router.GET("/test", func(c *gin.Context) {
		tenantID, _ := c.Get("tenant_id")
		pagination, _ := c.Get("pagination")
		dateRange, _ := c.Get("date_range")

		c.JSON(http.StatusOK, gin.H{
			"tenant_id":  tenantID,
			"pagination": pagination,
			"date_range": dateRange,
		})
	})

	headers := map[string]string{"X-Tenant-ID": validUUID}
	w := performRequest(router, "GET", "/test?limit=25&offset=10&date_from=2024-01-01&date_to=2024-01-31", "", headers)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, validUUID, response["tenant_id"])
	assert.NotNil(t, response["pagination"])
	assert.NotNil(t, response["date_range"])

	pagination := response["pagination"].(map[string]interface{})
	assert.Equal(t, 25, int(pagination["limit"].(float64)))
	assert.Equal(t, 10, int(pagination["offset"].(float64)))

	dateRange := response["date_range"].(map[string]interface{})
	assert.NotNil(t, dateRange["date_from"])
	assert.NotNil(t, dateRange["date_to"])
}

func TestMiddleware_ErrorPropagation(t *testing.T) {
	router := setupTestRouter()

	// Chain middlewares where first one should error
	router.Use(TenantID())
	router.Use(DateRange()) // This won't run if TenantID errors
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Missing tenant ID should cause early termination
	w := performRequest(router, "GET", "/test?date_from=2024-01-01", "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
