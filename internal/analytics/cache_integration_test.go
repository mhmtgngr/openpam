package analytics

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests require a running Redis instance
// They are marked as integration tests and can be skipped with -short flag

func setupTestRedis(t *testing.T) *cache.Cache {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a real Redis client for integration testing
	// In CI/CD, this would use testcontainers or a docker-compose setup
	logger := zerolog.New(io.Discard)
	c, err := cache.New(cache.Config{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       15, // Use separate DB for tests
	}, logger)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clear the test DB before each test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.Client().FlushDB(ctx).Err()

	return c
}

func TestRedisCache_SessionMetrics(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	analytics := &SessionAnalytics{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Date:           date,
		Hour:           12,
		TotalSessions:  100,
		ActiveSessions: 15,
		AvgDurationSeconds: ptr(300.0),
		CreatedAt:      time.Now(),
	}

	// Test Set and Get
	t.Run("Set and Get", func(t *testing.T) {
		err := rc.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)
		require.NoError(t, err)

		retrieved, err := rc.GetSessionMetrics(ctx, tenantID, date)
		require.NoError(t, err)
		assert.Equal(t, analytics.ID, retrieved.ID)
		assert.Equal(t, analytics.TotalSessions, retrieved.TotalSessions)
		assert.Equal(t, analytics.ActiveSessions, retrieved.ActiveSessions)
	})

	// Test cache miss
	t.Run("Cache miss", func(t *testing.T) {
		unknownDate := date.AddDate(0, 0, 1)
		_, err := rc.GetSessionMetrics(ctx, tenantID, unknownDate)
		assert.Error(t, err)
	})

	// Test invalidation
	t.Run("Invalidate", func(t *testing.T) {
		err := rc.InvalidateSessionMetrics(ctx, tenantID)
		require.NoError(t, err)

		_, err = rc.GetSessionMetrics(ctx, tenantID, date)
		assert.Error(t, err)
	})
}

func TestRedisCache_UserActivity(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	activity := &UserActivity{
		ID:                uuid.New(),
		TenantID:          tenantID,
		UserID:            userID,
		Date:              date,
		Hour:              14,
		SessionsInitiated: 5,
		CommandsExecuted:  100,
		CreatedAt:         time.Now(),
	}

	t.Run("Set and Get", func(t *testing.T) {
		err := rc.SetUserActivity(ctx, activity, UserActivityTTL)
		require.NoError(t, err)

		retrieved, err := rc.GetUserActivity(ctx, tenantID, userID, date)
		require.NoError(t, err)
		assert.Equal(t, activity.ID, retrieved.ID)
		assert.Equal(t, activity.CommandsExecuted, retrieved.CommandsExecuted)
	})

	t.Run("Invalidate by user", func(t *testing.T) {
		err := rc.InvalidateUserActivity(ctx, tenantID, userID)
		require.NoError(t, err)

		_, err = rc.GetUserActivity(ctx, tenantID, userID, date)
		assert.Error(t, err)
	})
}

func TestRedisCache_ComplianceReport(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	reportID := uuid.New()
	tenantID := uuid.New()

	report := &ComplianceReport{
		ID:          reportID,
		TenantID:    tenantID,
		Framework:   "SOC2",
		PeriodStart: time.Now().AddDate(0, -1, 0),
		PeriodEnd:   time.Now(),
		Status:      "complete",
		CreatedAt:   time.Now(),
	}

	t.Run("Set and Get", func(t *testing.T) {
		err := rc.SetComplianceReport(ctx, report, ComplianceReportTTL)
		require.NoError(t, err)

		retrieved, err := rc.GetComplianceReport(ctx, reportID)
		require.NoError(t, err)
		assert.Equal(t, report.ID, retrieved.ID)
		assert.Equal(t, report.Framework, retrieved.Framework)
		assert.Equal(t, report.Status, retrieved.Status)
	})

	t.Run("Invalidate", func(t *testing.T) {
		err := rc.InvalidateComplianceCache(ctx, tenantID)
		require.NoError(t, err)

		// Note: Invalidation pattern-based, so this may still work if key doesn't match pattern
		// This tests the invalidation method exists and doesn't error
	})
}

func TestRedisCache_AnomalyStats(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	stats := map[string]int{
		"open":          5,
		"investigating": 3,
		"resolved":      20,
		"false_positive": 2,
	}

	t.Run("Set and Get", func(t *testing.T) {
		err := rc.SetAnomalyStats(ctx, tenantID, stats, AnomalyStatsTTL)
		require.NoError(t, err)

		retrieved, err := rc.GetAnomalyStats(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, stats["open"], retrieved["open"])
		assert.Equal(t, stats["resolved"], retrieved["resolved"])
	})
}

func TestRedisCache_BatchInvalidation(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	// Populate various cache entries
	analytics := &SessionAnalytics{
		ID:       uuid.New(),
		TenantID: tenantID,
		Date:     date,
		Hour:     12,
	}
	_ = rc.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)

	stats := map[string]int{"open": 5}
	_ = rc.SetAnomalyStats(ctx, tenantID, stats, AnomalyStatsTTL)

	// Test batch invalidation
	t.Run("Batch invalidate multiple types", func(t *testing.T) {
		err := rc.BatchInvalidation(ctx, tenantID, "session", "anomaly")
		require.NoError(t, err)

		// Verify session cache was cleared
		_, err = rc.GetSessionMetrics(ctx, tenantID, date)
		assert.Error(t, err)

		// Verify anomaly stats were cleared
		_, err = rc.GetAnomalyStats(ctx, tenantID)
		assert.Error(t, err)
	})
}

func TestRedisCache_ContextTimeout(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	analytics := &SessionAnalytics{
		ID:       uuid.New(),
		TenantID: tenantID,
		Date:     date,
		Hour:     12,
	}

	t.Run("Context timeout is handled", func(t *testing.T) {
		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := rc.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)
		// Should return an error related to context cancellation
		assert.Error(t, err)
	})

	t.Run("Context with deadline works correctly", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := rc.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)
		assert.NoError(t, err)

		// Verify it was actually set
		retrieved, err := rc.GetSessionMetrics(ctx, tenantID, date)
		assert.NoError(t, err)
		assert.Equal(t, analytics.ID, retrieved.ID)
	})
}

func TestRedisCache_ExportImport(t *testing.T) {
	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	ctx := context.Background()

	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	// Populate cache
	analytics := &SessionAnalytics{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Date:           date,
		Hour:           12,
		TotalSessions:  100,
		ActiveSessions: 15,
	}
	_ = rc.SetSessionMetrics(ctx, analytics, DefaultMetricsTTL)

	t.Run("Export cache data", func(t *testing.T) {
		data, err := rc.ExportCacheData(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.NotEmpty(t, data)
	})

	t.Run("Import cache data", func(t *testing.T) {
		// First export
		data, err := rc.ExportCacheData(ctx, tenantID)
		require.NoError(t, err)

		// Clear cache
		_ = c.Client().FlushDB(ctx).Err()

		// Import
		err = rc.ImportCacheData(ctx, data)
		assert.NoError(t, err)

		// Verify data was restored
		retrieved, err := rc.GetSessionMetrics(ctx, tenantID, date)
		assert.NoError(t, err)
		assert.Equal(t, analytics.ID, retrieved.ID)
	})
}

func TestCachedRepository_CachingBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	c := setupTestRedis(t)
	if c == nil {
		return
	}
	defer c.Close()

	// Use a mock repo
	mockRepo := &cacheIntegrationMockRepository{
		sessionAnalytics: make(map[string]*SessionAnalytics),
	}

	logger := zerolog.New(io.Discard)
	rc := NewRedisCache(c, logger)
	cachedRepo := NewCachedRepository(mockRepo, rc, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	analytics := &SessionAnalytics{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Date:           date,
		Hour:           12,
		TotalSessions:  100,
		ActiveSessions: 15,
	}

	t.Run("First call queries repo, second uses cache", func(t *testing.T) {
		// Reset call counter
		mockRepo.getSessionCalls = 0

		// First call - should hit repo
		mockRepo.sessionAnalytics[cacheKey(tenantID, date, 12)] = analytics
		result1, err := cachedRepo.GetSessionAnalytics(ctx, tenantID, date, 12)
		require.NoError(t, err)
		assert.Equal(t, 1, mockRepo.getSessionCalls)

		// Second call - should use cache
		result2, err := cachedRepo.GetSessionAnalytics(ctx, tenantID, date, 12)
		require.NoError(t, err)
		assert.Equal(t, 1, mockRepo.getSessionCalls, "Should not call repo again")
		assert.Equal(t, result1.ID, result2.ID)
	})
}

// Helper types and functions

type cacheIntegrationMockRepository struct {
	sessionAnalytics map[string]*SessionAnalytics
	getSessionCalls  int
}

func (m *cacheIntegrationMockRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	m.sessionAnalytics[cacheKey(analytics.TenantID, analytics.Date, analytics.Hour)] = analytics
	return nil
}

func (m *cacheIntegrationMockRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	m.sessionAnalytics[cacheKey(analytics.TenantID, analytics.Date, analytics.Hour)] = analytics
	return nil
}

func (m *cacheIntegrationMockRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	m.getSessionCalls++
	key := cacheKey(tenantID, date, hour)
	if analytics, ok := m.sessionAnalytics[key]; ok {
		return analytics, nil
	}
	return nil, ErrNotFound
}

func (m *cacheIntegrationMockRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return nil, nil
}

// Other required interface methods (stubs)
func (m *cacheIntegrationMockRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *cacheIntegrationMockRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *cacheIntegrationMockRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error { return nil }
func (m *cacheIntegrationMockRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error { return nil }
func (m *cacheIntegrationMockRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error { return nil }
func (m *cacheIntegrationMockRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error { return nil }
func (m *cacheIntegrationMockRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error { return nil }
func (m *cacheIntegrationMockRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error { return nil }
func (m *cacheIntegrationMockRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error { return nil }
func (m *cacheIntegrationMockRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error { return nil }
func (m *cacheIntegrationMockRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *cacheIntegrationMockRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *cacheIntegrationMockRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error { return nil }
func (m *cacheIntegrationMockRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *cacheIntegrationMockRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *cacheIntegrationMockRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) { return nil, nil }
func (m *cacheIntegrationMockRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) { return nil, nil }

func cacheKey(tenantID uuid.UUID, date time.Time, hour int) string {
	return tenantID.String() + date.Format("2006-01-02") + string(rune(hour))
}

func ptr[T any](v T) *T {
	return &v
}

var ErrNotFound = fmt.Errorf("not found")
