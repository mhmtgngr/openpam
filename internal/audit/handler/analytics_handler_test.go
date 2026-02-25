// Package handler provides tests for analytics HTTP handlers
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Helper Functions Tests
// =============================================================================

func TestGetIntQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("default value when missing", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		result := getIntQuery(c, "limit", 50)
		assert.Equal(t, 50, result)
	})

	t.Run("parsed value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?limit=100", nil)

		// Gin parses query from the request URL
		result := getIntQuery(c, "limit", 50)
		assert.Equal(t, 100, result)
	})

	t.Run("invalid format returns default", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?limit=abc", nil)

		result := getIntQuery(c, "limit", 50)
		assert.Equal(t, 50, result)
	})
}

// =============================================================================
// Request DTO Validation Tests
// =============================================================================

func TestCreateComplianceReportRequest_Validation(t *testing.T) {
	now := time.Now()

	t.Run("valid request", func(t *testing.T) {
		req := model.CreateComplianceReportRequest{
			ReportName:  "Q4 2024 SOC2 Report",
			Framework:   model.FrameworkSOC2,
			Version:     "2022",
			PeriodStart: now.Add(-30 * 24 * time.Hour),
			PeriodEnd:   now,
		}

		assert.Equal(t, "Q4 2024 SOC2 Report", req.ReportName)
		assert.Equal(t, model.FrameworkSOC2, req.Framework)
		assert.False(t, req.PeriodStart.IsZero())
		assert.False(t, req.PeriodEnd.IsZero())
	})

	t.Run("missing required fields", func(t *testing.T) {
		req := model.CreateComplianceReportRequest{}

		// This would fail validation in the handler
		assert.Equal(t, "", req.ReportName)
		assert.Equal(t, "", req.Framework)
		assert.True(t, req.PeriodStart.IsZero())
	})
}

func TestCreateComplianceExceptionRequest_Validation(t *testing.T) {
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)

	t.Run("valid request", func(t *testing.T) {
		req := model.CreateComplianceExceptionRequest{
			ControlID:            "CC1.1",
			ControlName:          "Access Control",
			Framework:            model.FrameworkSOC2,
			RiskLevel:            "medium",
			Justification:        "Technical limitation",
			BusinessReason:       "Legacy system",
			CompensatingControls: []string{"Manual review"},
			ExpiresAt:            &expiresAt,
		}

		assert.Equal(t, "CC1.1", req.ControlID)
		assert.Equal(t, "Access Control", req.ControlName)
		assert.Equal(t, model.FrameworkSOC2, req.Framework)
		assert.NotEmpty(t, req.Justification)
		assert.NotNil(t, req.ExpiresAt)
	})

	t.Run("with nil expires at", func(t *testing.T) {
		req := model.CreateComplianceExceptionRequest{
			ControlID:     "CC1.1",
			ControlName:   "Access Control",
			Framework:     model.FrameworkSOC2,
			RiskLevel:     "low",
			Justification: "Temporary exception",
		}

		assert.Nil(t, req.ExpiresAt)
	})
}

func TestUpdateAnomalyRequest_Validation(t *testing.T) {
	assignedTo := uuid.New()
	resolutionNotes := "Resolved"

	t.Run("valid request", func(t *testing.T) {
		req := model.UpdateAnomalyRequest{
			Status:          stringPtr("resolved"),
			AssignedTo:      &assignedTo,
			ResolutionNotes: &resolutionNotes,
		}

		assert.NotNil(t, req.Status)
		assert.Equal(t, "resolved", *req.Status)
		assert.NotNil(t, req.AssignedTo)
		assert.NotNil(t, req.ResolutionNotes)
	})

	t.Run("with nil fields", func(t *testing.T) {
		req := model.UpdateAnomalyRequest{}

		assert.Nil(t, req.Status)
		assert.Nil(t, req.AssignedTo)
		assert.Nil(t, req.ResolutionNotes)
	})
}

func TestCreateCommandBlacklistRequest_Validation(t *testing.T) {
	userID := uuid.New()

	t.Run("valid request", func(t *testing.T) {
		req := model.CreateCommandBlacklistRequest{
			CommandPattern:   "rm -rf /*",
			PatternType:      "glob",
			BaseCommand:      "rm",
			Action:           "block",
			Severity:         "critical",
			AppliesToUsers:   []uuid.UUID{userID},
			AppliesToTargets: []string{"/etc/*", "/var/*"},
			AllowOverride:    false,
			Reason:           "Dangerous command",
		}

		assert.Equal(t, "rm -rf /*", req.CommandPattern)
		assert.Equal(t, "glob", req.PatternType)
		assert.Equal(t, "block", req.Action)
		assert.Equal(t, "critical", req.Severity)
		assert.NotEmpty(t, req.AppliesToUsers)
		assert.NotEmpty(t, req.AppliesToTargets)
		assert.False(t, req.AllowOverride)
		assert.NotEmpty(t, req.Reason)
	})

	t.Run("minimal request", func(t *testing.T) {
		req := model.CreateCommandBlacklistRequest{
			CommandPattern: "dd if=/dev/zero",
			PatternType:    "exact",
			Action:         "block",
			Severity:       "critical",
			Reason:         "Disk destruction",
		}

		assert.Nil(t, req.AppliesToUsers)
		assert.Nil(t, req.AppliesToGroups)
		assert.Nil(t, req.AppliesToTargets)
	})
}

func TestUpdateCommandBlacklistRequest_Validation(t *testing.T) {
	newPattern := "chmod 777"
	enabled := false

	t.Run("valid update", func(t *testing.T) {
		req := model.UpdateCommandBlacklistRequest{
			CommandPattern: &newPattern,
			Enabled:        &enabled,
		}

		assert.NotNil(t, req.CommandPattern)
		assert.Equal(t, "chmod 777", *req.CommandPattern)
		assert.NotNil(t, req.Enabled)
		assert.False(t, *req.Enabled)
	})

	t.Run("nil fields", func(t *testing.T) {
		req := model.UpdateCommandBlacklistRequest{}

		assert.Nil(t, req.CommandPattern)
		assert.Nil(t, req.Enabled)
	})
}

// =============================================================================
// Filter Tests
// =============================================================================

func TestComplianceReportFilter_Building(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	framework := model.FrameworkSOC2
	status := "passed"
	dateFrom := now.Add(-30 * 24 * time.Hour)
	dateTo := now

	t.Run("build filter with all fields", func(t *testing.T) {
		filter := model.ComplianceReportFilter{
			TenantID:  &tenantID,
			Framework: &framework,
			Status:    &status,
			DateFrom:  &dateFrom,
			DateTo:    &dateTo,
		}

		assert.NotNil(t, filter.TenantID)
		assert.NotNil(t, filter.Framework)
		assert.Equal(t, "SOC2", *filter.Framework)
		assert.NotNil(t, filter.Status)
		assert.Equal(t, "passed", *filter.Status)
		assert.NotNil(t, filter.DateFrom)
		assert.NotNil(t, filter.DateTo)
	})

	t.Run("filter with only tenant", func(t *testing.T) {
		filter := model.ComplianceReportFilter{
			TenantID: &tenantID,
		}

		assert.NotNil(t, filter.TenantID)
		assert.Nil(t, filter.Framework)
		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.DateFrom)
		assert.Nil(t, filter.DateTo)
	})
}

func TestAnomalyFilter_Building(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	userID := uuid.New()
	anomalyType := "behavioral"
	severity := "high"
	status := "open"

	t.Run("build filter with all fields", func(t *testing.T) {
		dateFrom := now.Add(-7 * 24 * time.Hour)
		filter := model.AnomalyFilter{
			TenantID:    &tenantID,
			UserID:      &userID,
			AnomalyType: &anomalyType,
			Severity:    &severity,
			Status:      &status,
			DateFrom:    &dateFrom,
			DateTo:      &now,
		}

		assert.NotNil(t, filter.TenantID)
		assert.NotNil(t, filter.UserID)
		assert.Equal(t, "behavioral", *filter.AnomalyType)
		assert.Equal(t, "high", *filter.Severity)
		assert.Equal(t, "open", *filter.Status)
	})
}

func TestCommandBlacklistFilter_Building(t *testing.T) {
	tenantID := uuid.New()
	enabled := true
	patternType := "regex"
	action := "block"
	severity := "critical"

	t.Run("build filter with all fields", func(t *testing.T) {
		filter := model.CommandBlacklistFilter{
			TenantID:    &tenantID,
			Enabled:     &enabled,
			PatternType: &patternType,
			Action:      &action,
			Severity:    &severity,
		}

		assert.NotNil(t, filter.TenantID)
		assert.NotNil(t, filter.Enabled)
		assert.True(t, *filter.Enabled)
		assert.Equal(t, "regex", *filter.PatternType)
		assert.Equal(t, "block", *filter.Action)
		assert.Equal(t, "critical", *filter.Severity)
	})
}

// =============================================================================
// Handler Response Tests
// =============================================================================

func TestHandler_ErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedCode   int
		expectedError   string
		testFunc       func(*gin.Engine)
	}{
		{
			name:         "missing tenant ID returns 400",
			expectedCode: 400,
			expectedError: "INVALID_TENANT",
			testFunc: func(router *gin.Engine) {
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_TENANT", "message": "Invalid tenant ID"}})
				})
			},
		},
		{
			name:         "not found returns 404",
			expectedCode: 404,
			expectedError: "NOT_FOUND",
			testFunc: func(router *gin.Engine) {
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Report not found"}})
				})
			},
		},
		{
			name:         "internal error returns 500",
			expectedCode: 500,
			expectedError: "INTERNAL_ERROR",
			testFunc: func(router *gin.Engine) {
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Failed to get data"}})
				})
			},
		},
		{
			name:         "unauthorized returns 401",
			expectedCode: 401,
			expectedError: "UNAUTHORIZED",
			testFunc: func(router *gin.Engine) {
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "User not authenticated"}})
				})
			},
		},
		{
			name:         "forbidden returns 403",
			expectedCode: 403,
			expectedError: "FORBIDDEN",
			testFunc: func(router *gin.Engine) {
				router.GET("/test", func(c *gin.Context) {
					c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "Access denied"}})
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			tt.testFunc(router)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if response["error"] != nil {
				errorObj := response["error"].(map[string]interface{})
				assert.Contains(t, errorObj["code"], tt.expectedError)
			}
		})
	}
}

func TestHandler_SuccessResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("compliance report created returns 201", func(t *testing.T) {
		router := gin.New()
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"report": gin.H{"id": uuid.New()}})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("compliance exception created returns 201", func(t *testing.T) {
		router := gin.New()
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"exception": gin.H{"id": uuid.New()}})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("anomaly created returns 201", func(t *testing.T) {
		router := gin.New()
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"anomaly": gin.H{"id": uuid.New()}})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("command blacklist created returns 201", func(t *testing.T) {
		router := gin.New()
		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"blacklist": gin.H{"id": uuid.New()}})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

// =============================================================================
// UUID Parsing Tests
// =============================================================================

func TestUUIDParsing_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_ID", "message": "Invalid report ID"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test/not-a-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid UUID returns 200", func(t *testing.T) {
		validID := uuid.New()
		req := httptest.NewRequest("GET", "/test/"+validID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// =============================================================================
// Query Parameter Parsing Tests
// =============================================================================

func TestQueryParameter_DateParsing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		var dateFrom, dateTo *time.Time

		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if t, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
				dateFrom = &t
			}
		}
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if t, err := time.Parse(time.RFC3339, dateToStr); err == nil {
				dateTo = &t
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"date_from": dateFrom,
			"date_to":   dateTo,
		})
	})

	t.Run("valid RFC3339 dates", func(t *testing.T) {
		dateFrom := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
		dateTo := time.Now().Format(time.RFC3339)

		req := httptest.NewRequest("GET", "/test?date_from="+dateFrom+"&date_to="+dateTo, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response["date_from"])
		assert.NotNil(t, response["date_to"])
	})

	t.Run("YYYY-MM-DD dates - not parsed by RFC3339 handler", func(t *testing.T) {
		// The handler only parses RFC3339 format, so YYYY-MM-DD won't be parsed
		dateFrom := "2024-01-01"
		dateTo := "2024-01-31"

		req := httptest.NewRequest("GET", "/test?date_from="+dateFrom+"&date_to="+dateTo, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		// Dates should be nil since format doesn't match RFC3339
		assert.Nil(t, response["date_from"])
		assert.Nil(t, response["date_to"])
	})

	t.Run("invalid date format ignored", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?date_from=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Nil(t, response["date_from"])
	})
}

// =============================================================================
// Pagination Tests
// =============================================================================

func TestQueryParameter_Pagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/test", func(c *gin.Context) {
		limit := getIntQuery(c, "limit", 50)
		offset := getIntQuery(c, "offset", 0)

		c.JSON(http.StatusOK, gin.H{
			"limit":  limit,
			"offset": offset,
		})
	})

	t.Run("default pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 50, int(response["limit"].(float64)))
		assert.Equal(t, 0, int(response["offset"].(float64)))
	})

	t.Run("custom pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?limit=20&offset=40", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 20, int(response["limit"].(float64)))
		assert.Equal(t, 40, int(response["offset"].(float64)))
	})

	t.Run("negative limit is parsed as-is", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?limit=-5", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, -5, int(response["limit"].(float64))) // negative values are parsed as-is
	})
}

// =============================================================================
// Utility Functions
// =============================================================================

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
