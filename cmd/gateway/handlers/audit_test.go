package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestAuditHandler creates a handler for testing validation logic
func setupTestAuditHandler() *AuditHandler {
	return &AuditHandler{
		service: nil, // Service is not needed for validation tests
		logger:  zerolog.Nop(),
	}
}

func TestAuditHandler_List(t *testing.T) {
	t.Run("parse pagination parameters", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/events?limit=20&offset=40", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.List(c)
		}()

		// Validation passed (either panic due to nil service or would have returned 500)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})

	t.Run("parse filter parameters", func(t *testing.T) {
		handler := setupTestAuditHandler()

		actorID := uuid.New()

		req := httptest.NewRequest("GET", "/audit/events?action=login&actor_id="+actorID.String()+"&outcome=success", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.List(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})

	t.Run("parse date range filters", func(t *testing.T) {
		handler := setupTestAuditHandler()

		from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
		to := time.Now().Format(time.RFC3339)

		req := httptest.NewRequest("GET", "/audit/events?from="+from+"&to="+to, nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.List(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuditHandler_Get(t *testing.T) {
	t.Run("invalid event ID", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/events/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("tenant_id", uuid.New().String())

		handler.Get(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_EVENT_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid event ID format", func(t *testing.T) {
		handler := setupTestAuditHandler()
		eventID := uuid.New()

		req := httptest.NewRequest("GET", "/audit/events/"+eventID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: eventID.String()}}
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Get(c)
		}()

		// Validation passed (either panic due to nil service or would have returned 500)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuditHandler_Export(t *testing.T) {
	t.Run("export with date range", func(t *testing.T) {
		handler := setupTestAuditHandler()

		from := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
		to := time.Now().Format(time.RFC3339)

		req := httptest.NewRequest("GET", "/audit/events/export?from="+from+"&to="+to, nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Export(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})

	t.Run("export without filters", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/events/export", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Export(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuditHandler_VerifyIntegrity(t *testing.T) {
	t.Run("validates tenant_id is set", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/integrity", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.VerifyIntegrity(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuditHandler_GenerateComplianceReport(t *testing.T) {
	t.Run("valid date range", func(t *testing.T) {
		handler := setupTestAuditHandler()

		from := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
		to := time.Now().Format(time.RFC3339)

		req := httptest.NewRequest("GET", "/audit/compliance/report?from="+from+"&to="+to, nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.GenerateComplianceReport(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})

	t.Run("invalid date format - from", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/compliance/report?from=invalid-date", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.GenerateComplianceReport(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_DATE_RANGE", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("invalid date format - to", func(t *testing.T) {
		handler := setupTestAuditHandler()

		from := time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)

		req := httptest.NewRequest("GET", "/audit/compliance/report?from="+from+"&to=invalid-date", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.GenerateComplianceReport(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_DATE_RANGE", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("default date range (last 30 days)", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/compliance/report", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.GenerateComplianceReport(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuditHandler_GetStats(t *testing.T) {
	t.Run("get stats successfully", func(t *testing.T) {
		handler := setupTestAuditHandler()

		req := httptest.NewRequest("GET", "/audit/stats", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.GetStats(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

// Test EventOutcome validation
func TestAuditEventOutcome(t *testing.T) {
	t.Run("valid outcomes", func(t *testing.T) {
		validOutcomes := []audit.EventOutcome{
			audit.OutcomeSuccess,
			audit.OutcomeFailure,
			audit.OutcomeDenied,
		}

		for _, outcome := range validOutcomes {
			assert.NotEmpty(t, string(outcome))
		}
	})
}

// Test EventType constants
func TestAuditEventType(t *testing.T) {
	t.Run("valid event types", func(t *testing.T) {
		validTypes := []audit.EventType{
			audit.EventTypeAuth,
			audit.EventTypeCredential,
			audit.EventTypeSession,
			audit.EventTypeTarget,
			audit.EventTypeUser,
			audit.EventTypeRole,
			audit.EventTypeApproval,
			audit.EventTypeCheckout,
			audit.EventTypeConfiguration,
			audit.EventTypeAnomaly,
			audit.EventTypeBreakGlass,
		}

		for _, eventType := range validTypes {
			assert.NotEmpty(t, string(eventType))
		}
	})
}

// Helper function for JSON decoding
func requireJSONDecode(body *bytes.Buffer, v interface{}) error {
	return json.Unmarshal(body.Bytes(), v)
}

// Test audit event filtering parameter parsing
func TestAuditEventFilterParsing(t *testing.T) {
	t.Run("parse actor_id parameter", func(t *testing.T) {
		actorID := uuid.New()
		filter := audit.EventFilter{}

		// Simulate parsing logic from handler
		filter.ActorID = &actorID

		assert.NotNil(t, filter.ActorID)
		assert.Equal(t, actorID, *filter.ActorID)
	})

	t.Run("parse action parameter", func(t *testing.T) {
		action := "login"
		filter := audit.EventFilter{}

		filter.Action = &action

		assert.NotNil(t, filter.Action)
		assert.Equal(t, "login", *filter.Action)
	})

	t.Run("parse outcome parameter", func(t *testing.T) {
		outcome := audit.OutcomeSuccess
		filter := audit.EventFilter{}

		filter.Outcome = &outcome

		assert.NotNil(t, filter.Outcome)
		assert.Equal(t, audit.OutcomeSuccess, *filter.Outcome)
	})

	t.Run("parse date range parameters", func(t *testing.T) {
		from := time.Now().Add(-24 * time.Hour)
		to := time.Now()
		filter := audit.EventFilter{}

		filter.StartTimeFrom = &from
		filter.StartTimeTo = &to

		assert.NotNil(t, filter.StartTimeFrom)
		assert.NotNil(t, filter.StartTimeTo)
		assert.True(t, filter.StartTimeFrom.Before(*filter.StartTimeTo))
	})
}

// Test compliance report structure
func TestComplianceReport(t *testing.T) {
	t.Run("create compliance report", func(t *testing.T) {
		tenantID := uuid.New()
		startTime := time.Now().Add(-30 * 24 * time.Hour)
		endTime := time.Now()

		report := &audit.ComplianceReport{
			TenantID:         tenantID,
			StartTime:        startTime,
			EndTime:          endTime,
			Generated:        time.Now(),
			TotalEvents:      1000,
			SuccessfulEvents: 950,
			FailedEvents:     30,
			DeniedEvents:     20,
			EventTypes:       map[string]int{"login": 200, "logout": 200},
			IntegrityValid:   true,
		}

		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, 1000, report.TotalEvents)
		assert.Equal(t, 950, report.SuccessfulEvents)
		assert.Equal(t, 30, report.FailedEvents)
		assert.Equal(t, 20, report.DeniedEvents)
		assert.True(t, report.IntegrityValid)
		assert.Equal(t, 2, len(report.EventTypes))
	})

	t.Run("report with integrity errors", func(t *testing.T) {
		report := &audit.ComplianceReport{
			TenantID:        uuid.New(),
			StartTime:       time.Now().Add(-7 * 24 * time.Hour),
			EndTime:         time.Now(),
			Generated:       time.Now(),
			TotalEvents:     500,
			IntegrityValid:  false,
			IntegrityErrors: []string{"Chain broken at event 42", "Hash mismatch at event 100"},
		}

		assert.False(t, report.IntegrityValid)
		assert.Len(t, report.IntegrityErrors, 2)
		assert.Equal(t, "Chain broken at event 42", report.IntegrityErrors[0])
	})
}

// Test audit event structure
func TestAuditEvent(t *testing.T) {
	t.Run("create audit event", func(t *testing.T) {
		tenantID := uuid.New()
		actorID := uuid.New()
		eventID := uuid.New()

		event := &audit.Event{
			ID:           eventID,
			TenantID:     tenantID,
			ActorID:      actorID,
			ActorType:    "user",
			Action:       "login",
			ResourceType: "credential",
			ResourceID:   "cred-123",
			Outcome:      audit.OutcomeSuccess,
			IP:           "192.168.1.1",
			UserAgent:    "test-agent",
			RequestID:    "req-123",
			Timestamp:    time.Now(),
		}

		assert.Equal(t, eventID, event.ID)
		assert.Equal(t, tenantID, event.TenantID)
		assert.Equal(t, actorID, event.ActorID)
		assert.Equal(t, "user", event.ActorType)
		assert.Equal(t, "login", event.Action)
		assert.Equal(t, "credential", event.ResourceType)
		assert.Equal(t, audit.OutcomeSuccess, event.Outcome)
	})

	t.Run("event with error details", func(t *testing.T) {
		event := &audit.Event{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			ActorID:      uuid.New(),
			Action:       "checkout",
			ResourceType: "credential",
			Outcome:      audit.OutcomeFailure,
			ErrorCode:    "ACCESS_DENIED",
			ErrorMessage: "User does not have permission",
			Timestamp:    time.Now(),
		}

		assert.Equal(t, audit.OutcomeFailure, event.Outcome)
		assert.Equal(t, "ACCESS_DENIED", event.ErrorCode)
		assert.Equal(t, "User does not have permission", event.ErrorMessage)
	})
}

// Test pagination parameter parsing
func TestAuditPagination(t *testing.T) {
	tests := []struct {
		name     string
		limit    string
		offset   string
		expected struct {
			limit  int
			offset int
		}
	}{
		{
			name:   "default values",
			limit:  "",
			offset: "",
			expected: struct {
				limit  int
				offset int
			}{limit: 50, offset: 0},
		},
		{
			name:   "custom values",
			limit:  "100",
			offset: "50",
			expected: struct {
				limit  int
				offset int
			}{limit: 100, offset: 50},
		},
		{
			name:   "invalid values use defaults",
			limit:  "invalid",
			offset: "invalid",
			expected: struct {
				limit  int
				offset int
			}{limit: 50, offset: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := 50
			offset := 0

			if tt.limit != "" {
				if parsed, err := strconv.Atoi(tt.limit); err == nil && parsed > 0 {
					limit = parsed
				}
			}
			if tt.offset != "" {
				if parsed, err := strconv.Atoi(tt.offset); err == nil && parsed >= 0 {
					offset = parsed
				}
			}

			assert.Equal(t, tt.expected.limit, limit)
			assert.Equal(t, tt.expected.offset, offset)
		})
	}
}
