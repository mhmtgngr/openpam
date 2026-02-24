package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnalyticsService is a mock service for testing
type mockAnalyticsService struct {
	sessionMetrics     *analytics.SessionSummary
	userActivities     []analytics.UserActivity
	riskScore          float64
	commands           []analytics.CommandRank
	dashboard          *analytics.DashboardMetrics
	blacklist          []analytics.CommandBlacklist
	sshKeyAnalytics    []analytics.SSHKeyAnalytics
	timeSeries         []analytics.DataPoint
	cacheStats         map[string]interface{}
	complianceSummary  *analytics.ComplianceSummary
	recordSessionStart error
	recordSessionEnd   error
	recordCommand      error
	createBlacklist    error
	updateBlacklist    error
	deleteBlacklist    error
	invalidateCache    error
	warmCache          error
}

func (m *mockAnalyticsService) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*analytics.SessionSummary, error) {
	return m.sessionMetrics, nil
}
func (m *mockAnalyticsService) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, dateFrom, dateTo time.Time) ([]analytics.UserActivity, error) {
	return m.userActivities, nil
}
func (m *mockAnalyticsService) GetUserRiskScore(ctx context.Context, tenantID, userID uuid.UUID, days int) (float64, error) {
	return m.riskScore, nil
}
func (m *mockAnalyticsService) GetCommandFrequency(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]analytics.CommandRank, error) {
	return m.commands, nil
}
func (m *mockAnalyticsService) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*analytics.DashboardMetrics, error) {
	return m.dashboard, nil
}
func (m *mockAnalyticsService) GetTimeSeriesData(ctx context.Context, tenantID uuid.UUID, metric string, dateFrom, dateTo time.Time) ([]analytics.DataPoint, error) {
	return m.timeSeries, nil
}
func (m *mockAnalyticsService) GetComplianceReport(ctx context.Context, id uuid.UUID) (*analytics.ComplianceReport, error) {
	return nil, nil
}
func (m *mockAnalyticsService) ListComplianceReports(ctx context.Context, tenantID uuid.UUID, framework *string) ([]analytics.ComplianceReport, error) {
	return nil, nil
}
func (m *mockAnalyticsService) GenerateComplianceReport(ctx context.Context, tenantID uuid.UUID, generatedBy uuid.UUID, framework analytics.ComplianceFramework, periodStart, periodEnd time.Time) (*analytics.ComplianceReport, error) {
	return nil, nil
}
func (m *mockAnalyticsService) CreateComplianceException(ctx context.Context, exception *analytics.ComplianceException) error {
	return nil
}
func (m *mockAnalyticsService) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]analytics.ComplianceException, error) {
	return nil, nil
}
func (m *mockAnalyticsService) GetComplianceSummary(ctx context.Context, tenantID uuid.UUID) (*analytics.ComplianceSummary, error) {
	return m.complianceSummary, nil
}
func (m *mockAnalyticsService) DetectAnomalies(ctx context.Context, tenantID uuid.UUID) ([]analytics.AnomalyDetection, error) {
	return nil, nil
}
func (m *mockAnalyticsService) GetAnomaly(ctx context.Context, id uuid.UUID) (*analytics.AnomalyDetection, error) {
	return nil, nil
}
func (m *mockAnalyticsService) ListAnomalies(ctx context.Context, tenantID uuid.UUID, status *string) ([]analytics.AnomalyDetection, error) {
	return nil, nil
}
func (m *mockAnalyticsService) UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status analytics.AnomalyStatus, assignedTo *uuid.UUID, notes *string, resolvedBy *uuid.UUID) error {
	return nil
}
func (m *mockAnalyticsService) RunAnomalyDetection(ctx context.Context, tenantID uuid.UUID) ([]analytics.AnomalyDetection, error) {
	return nil, nil
}
func (m *mockAnalyticsService) EvaluateUserForAnomalies(ctx context.Context, tenantID, userID uuid.UUID) ([]analytics.AnomalyDetection, error) {
	return nil, nil
}
func (m *mockAnalyticsService) CreateRansomwareEvent(ctx context.Context, id uuid.UUID, event *analytics.RansomwareEvent) error {
	return nil
}
func (m *mockAnalyticsService) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*analytics.RansomwareEvent, error) {
	return nil, nil
}
func (m *mockAnalyticsService) TriggerEmergencyResponse(ctx context.Context, tenantID, eventID uuid.UUID) error {
	return nil
}
func (m *mockAnalyticsService) CreateCommandBlacklist(ctx context.Context, blacklist *analytics.CommandBlacklist) error {
	return m.createBlacklist
}
func (m *mockAnalyticsService) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*analytics.CommandBlacklist, error) {
	return nil, nil
}
func (m *mockAnalyticsService) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]analytics.CommandBlacklist, error) {
	return m.blacklist, nil
}
func (m *mockAnalyticsService) UpdateCommandBlacklist(ctx context.Context, blacklist *analytics.CommandBlacklist) error {
	return m.updateBlacklist
}
func (m *mockAnalyticsService) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	return m.deleteBlacklist
}
func (m *mockAnalyticsService) EvaluateCommandAgainstBlacklist(ctx context.Context, tenantID, userID uuid.UUID, command string, groups []uuid.UUID) (bool, string, *uuid.UUID) {
	return true, "", nil
}
func (m *mockAnalyticsService) RecordSSHKeyUsage(ctx context.Context, tenantID, sshKeyID, userID uuid.UUID, target string, duration time.Duration, failed bool) error {
	return nil
}
func (m *mockAnalyticsService) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]analytics.SSHKeyAnalytics, error) {
	return m.sshKeyAnalytics, nil
}
func (m *mockAnalyticsService) InvalidateCache(ctx context.Context, tenantID uuid.UUID, types ...string) error {
	return m.invalidateCache
}
func (m *mockAnalyticsService) WarmCache(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) error {
	return m.warmCache
}
func (m *mockAnalyticsService) GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	return m.cacheStats, nil
}
func (m *mockAnalyticsService) HandleSessionStarted(ctx context.Context, event events.Event) error {
	return m.recordSessionStart
}
func (m *mockAnalyticsService) HandleSessionEnded(ctx context.Context, event events.Event) error {
	return m.recordSessionEnd
}
func (m *mockAnalyticsService) HandleCommandExecuted(ctx context.Context, event events.Event) error {
	return m.recordCommand
}

func setupTestRouter(handler *AnalyticsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up auth middleware mock
	router.Use(func(c *gin.Context) {
		tenantID := uuid.New().String()
		userID := uuid.New().String()
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})

	router.GET("/analytics/sessions/metrics", handler.GetSessionMetrics)
	router.GET("/analytics/sessions/trends", handler.GetSessionTrends)
	router.GET("/analytics/sessions/summary", handler.GetSessionSummary)
	router.GET("/analytics/users/:id/activity", handler.GetUserActivity)
	router.GET("/analytics/users/:id/risk", handler.GetUserRiskScore)
	router.GET("/analytics/users/top", handler.GetTopUsers)
	router.GET("/analytics/commands/frequency", handler.GetCommandFrequency)
	router.GET("/analytics/commands/top", handler.GetTopCommands)
	router.GET("/analytics/blacklist", handler.GetCommandBlacklist)
	router.POST("/analytics/blacklist", handler.CreateCommandBlacklist)
	router.PUT("/analytics/blacklist/:id", handler.UpdateCommandBlacklist)
	router.DELETE("/analytics/blacklist/:id", handler.DeleteCommandBlacklist)
	router.GET("/analytics/dashboard", handler.GetDashboard)
	router.GET("/analytics/timeseries", handler.GetTimeSeries)
	router.GET("/analytics/ssh-keys/:id/analytics", handler.GetSSHKeyAnalytics)
	router.POST("/analytics/cache/invalidate", handler.InvalidateCache)
	router.POST("/analytics/cache/warm", handler.WarmCache)
	router.GET("/analytics/cache/stats", handler.GetCacheStats)
	router.GET("/analytics/sessions", handler.GetSessions)
	router.GET("/analytics/commands", handler.GetCommands)
	router.POST("/analytics/events/ingest", handler.IngestEvent)

	return router
}

func TestNewAnalyticsHandler(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}

	handler := NewAnalyticsHandler(service, logger)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
	assert.NotNil(t, handler.logger)
}

func TestAnalyticsHandler_GetSessionMetrics(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{
		sessionMetrics: &analytics.SessionSummary{
			TotalSessions:  100,
			ActiveSessions: 15,
			AvgDuration:    300.5,
			PeakConcurrent: 25,
			SessionsByType: map[string]int{
				"ssh": 60,
				"rdp": 25,
				"database": 15,
			},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/sessions/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["metrics"])
}

func TestAnalyticsHandler_GetSessionTrends(t *testing.T) {
	logger := zerolog.Nop()
	now := time.Now()
	service := &mockAnalyticsService{
		timeSeries: []analytics.DataPoint{
			{Timestamp: now.Add(-48 * time.Hour), Value: 80},
			{Timestamp: now.Add(-24 * time.Hour), Value: 90},
			{Timestamp: now, Value: 100},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/sessions/trends?metric=sessions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["data"])
}

func TestAnalyticsHandler_GetUserActivity(t *testing.T) {
	logger := zerolog.Nop()
	userID := uuid.New()
	service := &mockAnalyticsService{
		userActivities: []analytics.UserActivity{
			{
				ID:       uuid.New(),
				UserID:   userID,
				Date:     time.Now().Truncate(24 * time.Hour),
				Hour:     14,
				SessionsInitiated: 3,
				CommandsExecuted:  50,
			},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/users/"+userID.String()+"/activity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["activities"])
}

func TestAnalyticsHandler_GetUserActivityInvalidUserID(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/users/invalid-uuid/activity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["error"])
}

func TestAnalyticsHandler_GetUserRiskScore(t *testing.T) {
	logger := zerolog.Nop()
	userID := uuid.New()
	service := &mockAnalyticsService{
		riskScore: 45.5,
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/users/"+userID.String()+"/risk", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 45.5, response["risk_score"])
	assert.Equal(t, float64(30), response["days"]) // default
}

func TestAnalyticsHandler_GetUserRiskScoreCustomDays(t *testing.T) {
	logger := zerolog.Nop()
	userID := uuid.New()
	service := &mockAnalyticsService{
		riskScore: 60.0,
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/users/"+userID.String()+"/risk?days=7", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(7), response["days"])
}

func TestAnalyticsHandler_GetTopUsers(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/users/top", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestAnalyticsHandler_GetCommandFrequency(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{
		commands: []analytics.CommandRank{
			{Command: "ls", Count: 1500, RiskLevel: "low"},
			{Command: "cd", Count: 1200, RiskLevel: "low"},
			{Command: "rm", Count: 50, RiskLevel: "high"},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/commands/frequency?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["commands"])
}

func TestAnalyticsHandler_GetTopCommands(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{
		commands: []analytics.CommandRank{
			{Command: "ls", Count: 1500, RiskLevel: "low"},
			{Command: "cat", Count: 800, RiskLevel: "low"},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/commands/top?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["top_commands"])
}

func TestAnalyticsHandler_GetCommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	tenantID := uuid.New()
	baseCommand := "rm"
	service := &mockAnalyticsService{
		blacklist: []analytics.CommandBlacklist{
			{
				ID:            uuid.New(),
				TenantID:      &tenantID,
				CommandPattern: "rm -rf *",
				PatternType:   "glob",
				BaseCommand:   &baseCommand,
				Action:        "block",
				Severity:      "critical",
				CreatedBy:     uuid.New(),
				CreatedAt:     time.Now(),
			},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/blacklist", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["blacklist"])
}

func TestAnalyticsHandler_CreateCommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	blacklist := analytics.CommandBlacklist{
		CommandPattern: "rm -rf *",
		PatternType:   "glob",
		Action:        "block",
		Severity:      "critical",
		Reason:        "Dangerous command",
		Enabled:       true,
	}

	body, _ := json.Marshal(blacklist)
	req, _ := http.NewRequest("POST", "/analytics/blacklist", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["blacklist"])
}

func TestAnalyticsHandler_UpdateCommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	blacklist := analytics.CommandBlacklist{
		CommandPattern: "rm -rf *",
		PatternType:   "glob",
		Action:        "warn",
		Severity:      "high",
		Enabled:       true,
	}

	body, _ := json.Marshal(blacklist)
	id := uuid.New()
	req, _ := http.NewRequest("PUT", "/analytics/blacklist/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["blacklist"])
}

func TestAnalyticsHandler_DeleteCommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	id := uuid.New()
	req, _ := http.NewRequest("DELETE", "/analytics/blacklist/"+id.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["message"])
}

func TestAnalyticsHandler_GetDashboard(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{
		dashboard: &analytics.DashboardMetrics{
			SessionMetrics: &analytics.SessionSummary{
				TotalSessions:  100,
				ActiveSessions: 15,
				AvgDuration:    300.5,
			},
			UserActivity: &analytics.UserActivitySummary{
				ActiveUsers:   20,
				TotalCommands: 5000,
			},
			Timestamp: time.Now(),
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/dashboard", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["dashboard"])
}

func TestAnalyticsHandler_GetTimeSeries(t *testing.T) {
	logger := zerolog.Nop()
	now := time.Now()
	service := &mockAnalyticsService{
		timeSeries: []analytics.DataPoint{
			{Timestamp: now.Add(-48 * time.Hour), Value: 100},
			{Timestamp: now.Add(-24 * time.Hour), Value: 150},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/timeseries?metric=active_sessions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["data"])
}

func TestAnalyticsHandler_GetSSHKeyAnalytics(t *testing.T) {
	logger := zerolog.Nop()
	sshKeyID := uuid.New()
	service := &mockAnalyticsService{
		sshKeyAnalytics: []analytics.SSHKeyAnalytics{
			{
				ID:       uuid.New(),
				SSHKeyID: sshKeyID,
				Date:     time.Now().Truncate(24 * time.Hour),
				UsageCount: 10,
			},
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/ssh-keys/"+sshKeyID.String()+"/analytics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["analytics"])
}

func TestAnalyticsHandler_InvalidateCache(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("POST", "/analytics/cache/invalidate?types=session,activity", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["message"])
}

func TestAnalyticsHandler_WarmCache(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("POST", "/analytics/cache/warm", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["message"])
}

func TestAnalyticsHandler_GetCacheStats(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{
		cacheStats: map[string]interface{}{
			"session": 150,
			"activity": 75,
			"compliance": 2,
		},
	}

	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	req, _ := http.NewRequest("GET", "/analytics/cache/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["stats"])
}

func TestAnalyticsHandler_IngestEvent(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	event := map[string]interface{}{
		"type":      "analytics.session.started",
		"tenant_id": uuid.New().String(),
		"user_id":   uuid.New().String(),
		"data": map[string]interface{}{
			"session_id":  uuid.New().String(),
			"target_host": "server1.example.com",
			"target_port": float64(22),
			"type":        "ssh",
		},
	}

	body, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", "/analytics/events/ingest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["message"])
}

func TestAnalyticsHandler_IngestEventInvalidInput(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)
	router := setupTestRouter(handler)

	// Missing required field
	event := map[string]interface{}{
		"type": "analytics.session.started",
		// missing tenant_id, user_id, data
	}

	body, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", "/analytics/events/ingest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAnalyticsHandler_ParseDateRange(t *testing.T) {
	logger := zerolog.Nop()
	service := &mockAnalyticsService{}
	handler := NewAnalyticsHandler(service, logger)

	t.Run("default date range", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/analytics/sessions/metrics", nil)

		dateFrom, dateTo := handler.parseDateRange(c)

		// Should default to 7 days ago to now
		assert.False(t, dateFrom.IsZero())
		assert.False(t, dateTo.IsZero())
		assert.True(t, dateTo.After(dateFrom) || dateTo.Equal(dateFrom))
	})

	t.Run("custom date range", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/analytics/sessions/metrics?from=2024-01-01&to=2024-01-31", nil)

		dateFrom, dateTo := handler.parseDateRange(c)

		expectedFrom, _ := time.Parse("2006-01-02", "2024-01-01")
		expectedTo, _ := time.Parse("2006-01-02", "2024-01-31")

		assert.Equal(t, expectedFrom, dateFrom)
		assert.Equal(t, expectedTo, dateTo)
	})
}

func TestGetIntQueryHelper(t *testing.T) {
	// This is a helper function used by handlers
	// Testing it indirectly through the handlers
	t.Run("default limit", func(t *testing.T) {
		logger := zerolog.Nop()
		service := &mockAnalyticsService{}
		handler := NewAnalyticsHandler(service, logger)
		router := setupTestRouter(handler)

		req, _ := http.NewRequest("GET", "/analytics/commands/frequency", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should succeed with default limit
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
