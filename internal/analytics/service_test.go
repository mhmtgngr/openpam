package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// mockServiceRepository is a minimal mock for service testing
type mockServiceRepository struct {
	createSessionErr error
	updateSessionErr error
	createActivityErr error
	updateActivityErr error
	recordCommandErr  error
}

func (m *mockServiceRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	return m.createSessionErr
}
func (m *mockServiceRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	return m.updateSessionErr
}
func (m *mockServiceRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return &SessionAnalytics{}, nil
}
func (m *mockServiceRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return []SessionAnalytics{}, nil
}
func (m *mockServiceRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error {
	return m.createActivityErr
}
func (m *mockServiceRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error {
	return m.updateActivityErr
}
func (m *mockServiceRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	return &UserActivity{}, nil
}
func (m *mockServiceRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	return []UserActivity{}, nil
}
func (m *mockServiceRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error {
	return m.recordCommandErr
}
func (m *mockServiceRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	return []CommandFrequency{}, nil
}
func (m *mockServiceRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return []CommandRank{}, nil
}
func (m *mockServiceRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	return nil
}
func (m *mockServiceRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	return nil, nil
}
func (m *mockServiceRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	return []ComplianceReport{}, nil
}
func (m *mockServiceRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error {
	return nil
}
func (m *mockServiceRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	return []ComplianceControlEvaluation{}, nil
}
func (m *mockServiceRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	return nil
}
func (m *mockServiceRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return []ComplianceException{}, nil
}
func (m *mockServiceRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	return nil
}
func (m *mockServiceRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	return nil, nil
}
func (m *mockServiceRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	return []AnomalyDetection{}, nil
}
func (m *mockServiceRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	return nil
}
func (m *mockServiceRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	return nil
}
func (m *mockServiceRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	return nil, nil
}
func (m *mockServiceRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	return []RansomwareEvent{}, nil
}
func (m *mockServiceRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	return nil
}
func (m *mockServiceRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return nil
}
func (m *mockServiceRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	return nil, nil
}
func (m *mockServiceRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockServiceRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return nil
}
func (m *mockServiceRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockServiceRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockServiceRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return nil
}
func (m *mockServiceRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return nil
}
func (m *mockServiceRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	return nil, nil
}
func (m *mockServiceRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return []SSHKeyAnalytics{}, nil
}
func (m *mockServiceRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	return nil, nil
}
func (m *mockServiceRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}

func TestNewService(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()

	service := NewService(repo, redisCache, publisher, logger, config)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.complianceEngine)
	assert.NotNil(t, service.anomalyDetector)
	assert.NotNil(t, service.commandExtractor)
}

func TestService_RecordSessionStart(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("records SSH session", func(t *testing.T) {
		err := service.RecordSessionStart(ctx, tenantID, userID, "ssh", "server1.example.com", 22)
		assert.NoError(t, err)
	})

	t.Run("records RDP session", func(t *testing.T) {
		err := service.RecordSessionStart(ctx, tenantID, userID, "rdp", "rdp-server.example.com", 3389)
		assert.NoError(t, err)
	})

	t.Run("records database session", func(t *testing.T) {
		err := service.RecordSessionStart(ctx, tenantID, userID, "database", "db.example.com", 5432)
		assert.NoError(t, err)
	})

	t.Run("handles create error", func(t *testing.T) {
		repo := &mockServiceRepository{createSessionErr: assert.AnError}
		redisCache := NewRedisCache(&cache.Cache{}, logger)
		service := NewService(repo, redisCache, publisher, logger, config)

		err := service.RecordSessionStart(ctx, tenantID, userID, "ssh", "server.example.com", 22)
		assert.Error(t, err)
	})
}

func TestService_RecordSessionEnd(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()
	duration := 30 * time.Minute

	t.Run("records session end", func(t *testing.T) {
		err := service.RecordSessionEnd(ctx, tenantID, userID, sessionID, duration, "ssh")
		assert.NoError(t, err)
	})

	t.Run("handles zero duration", func(t *testing.T) {
		err := service.RecordSessionEnd(ctx, tenantID, userID, sessionID, 0, "ssh")
		assert.NoError(t, err)
	})
}

func TestService_RecordCommand(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()

	t.Run("records safe command", func(t *testing.T) {
		err := service.RecordCommand(ctx, tenantID, userID, sessionID, "ls -la", "server1", nil)
		assert.NoError(t, err)
	})

	t.Run("records risky command", func(t *testing.T) {
		exitCode := 1
		err := service.RecordCommand(ctx, tenantID, userID, sessionID, "rm -rf /tmp/test", "server1", &exitCode)
		assert.NoError(t, err)
	})

	t.Run("handles empty command", func(t *testing.T) {
		err := service.RecordCommand(ctx, tenantID, userID, sessionID, "", "server1", nil)
		assert.NoError(t, err)
	})

	t.Run("handles whitespace only command", func(t *testing.T) {
		err := service.RecordCommand(ctx, tenantID, userID, sessionID, "   ", "server1", nil)
		assert.NoError(t, err)
	})
}

func TestService_GetSessionMetrics(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("returns session summary", func(t *testing.T) {
		summary, err := service.GetSessionMetrics(ctx, tenantID, dateFrom, dateTo)
		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.NotNil(t, summary.SessionsByType)
	})

	t.Run("handles empty date range", func(t *testing.T) {
		summary, err := service.GetSessionMetrics(ctx, tenantID, time.Time{}, time.Time{})
		assert.NoError(t, err)
		assert.NotNil(t, summary)
	})
}

func TestService_GetUserActivity(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("returns user activities", func(t *testing.T) {
		activities, err := service.GetUserActivity(ctx, tenantID, userID, dateFrom, dateTo)
		assert.NoError(t, err)
		assert.NotNil(t, activities)
	})

	t.Run("handles nil date range", func(t *testing.T) {
		activities, err := service.GetUserActivity(ctx, tenantID, userID, time.Time{}, time.Time{})
		assert.NoError(t, err)
		assert.NotNil(t, activities)
	})
}

func TestService_GetUserRiskScore(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("returns risk score for default days", func(t *testing.T) {
		score, err := service.GetUserRiskScore(ctx, tenantID, userID, 30)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
	})

	t.Run("returns risk score for custom days", func(t *testing.T) {
		score, err := service.GetUserRiskScore(ctx, tenantID, userID, 7)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
	})

	t.Run("handles zero days", func(t *testing.T) {
		score, err := service.GetUserRiskScore(ctx, tenantID, userID, 0)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
	})
}

func TestService_GetCommandFrequency(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("returns command ranks", func(t *testing.T) {
		commands, err := service.GetCommandFrequency(ctx, tenantID, dateFrom, dateTo, 10)
		assert.NoError(t, err)
		assert.NotNil(t, commands)
	})

	t.Run("handles limit parameter", func(t *testing.T) {
		commands, err := service.GetCommandFrequency(ctx, tenantID, dateFrom, dateTo, 100)
		assert.NoError(t, err)
		assert.NotNil(t, commands)
	})
}

func TestService_GetDashboardMetrics(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("returns dashboard metrics", func(t *testing.T) {
		metrics, err := service.GetDashboardMetrics(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
	})
}

func TestService_GetTimeSeriesData(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("returns time series data", func(t *testing.T) {
		data, err := service.GetTimeSeriesData(ctx, tenantID, "sessions", dateFrom, dateTo)
		assert.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("returns empty data for unknown metric", func(t *testing.T) {
		data, err := service.GetTimeSeriesData(ctx, tenantID, "unknown", dateFrom, dateTo)
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Empty(t, data)
	})
}

func TestService_Compliance(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	t.Run("generates compliance report", func(t *testing.T) {
		report, err := service.GenerateComplianceReport(ctx, tenantID, userID, FrameworkSOC2, periodStart, periodEnd)
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, string(FrameworkSOC2), report.Framework)
	})

	t.Run("gets compliance report", func(t *testing.T) {
		reportID := uuid.New()
		report, err := service.GetComplianceReport(ctx, reportID)
		assert.NoError(t, err)
		assert.Nil(t, report) // No report found
	})

	t.Run("lists compliance reports", func(t *testing.T) {
		reports, err := service.ListComplianceReports(ctx, tenantID, nil)
		assert.NoError(t, err)
		assert.NotNil(t, reports)
	})

	t.Run("lists compliance reports with framework filter", func(t *testing.T) {
		framework := "SOC2"
		reports, err := service.ListComplianceReports(ctx, tenantID, &framework)
		assert.NoError(t, err)
		assert.NotNil(t, reports)
	})

	t.Run("creates compliance exception", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:    tenantID,
			ControlID:   "CC1.1",
			ControlName: "Access Control",
			Framework:   "SOC2",
		}

		err := service.CreateComplianceException(ctx, exception)
		assert.NoError(t, err)
	})

	t.Run("lists compliance exceptions", func(t *testing.T) {
		exceptions, err := service.ListComplianceExceptions(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, exceptions)
	})

	t.Run("gets compliance summary", func(t *testing.T) {
		summary, err := service.GetComplianceSummary(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, summary)
		assert.NotNil(t, summary.Frameworks)
	})
}

func TestService_AnomalyDetection(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("runs anomaly detection", func(t *testing.T) {
		detections, err := service.RunAnomalyDetection(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, detections)
	})

	t.Run("evaluates user for anomalies", func(t *testing.T) {
		detections, err := service.EvaluateUserForAnomalies(ctx, tenantID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, detections)
	})

	t.Run("gets anomaly", func(t *testing.T) {
		anomalyID := uuid.New()
		anomaly, err := service.GetAnomaly(ctx, anomalyID)
		assert.NoError(t, err)
		assert.Nil(t, anomaly) // No anomaly found
	})

	t.Run("lists anomalies", func(t *testing.T) {
		status := "open"
		anomalies, err := service.ListAnomalies(ctx, tenantID, &status)
		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
	})

	t.Run("updates anomaly status", func(t *testing.T) {
		anomalyID := uuid.New()
		notes := "Investigated and resolved"
		resolvedBy := uuid.New()

		err := service.UpdateAnomalyStatus(ctx, anomalyID, AnomalyStatusResolved, nil, &notes, &resolvedBy)
		assert.NoError(t, err)
	})

	t.Run("updates anomaly with assigned user", func(t *testing.T) {
		anomalyID := uuid.New()
		assignedTo := uuid.New()

		err := service.UpdateAnomalyStatus(ctx, anomalyID, AnomalyStatusInvestigating, &assignedTo, nil, nil)
		assert.NoError(t, err)
	})
}

func TestService_Ransomware(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("creates ransomware event", func(t *testing.T) {
		detectionID := uuid.New()
		event := &RansomwareEvent{
			FilesAffected: 100,
		}

		err := service.CreateRansomwareEvent(ctx, detectionID, event)
		assert.NoError(t, err)
	})

	t.Run("gets ransomware event", func(t *testing.T) {
		eventID := uuid.New()
		event, err := service.GetRansomwareEvent(ctx, eventID)
		assert.NoError(t, err)
		assert.Nil(t, event) // No event found
	})

	t.Run("triggers emergency response", func(t *testing.T) {
		eventID := uuid.New()
		err := service.TriggerEmergencyResponse(ctx, tenantID, eventID)
		assert.NoError(t, err)
	})
}

func TestService_CommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates command blacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			CommandPattern: "rm -rf *",
			PatternType:   "glob",
			Action:        "block",
			Severity:      "critical",
		}

		err := service.CreateCommandBlacklist(ctx, blacklist)
		assert.NoError(t, err)
	})

	t.Run("gets command blacklist", func(t *testing.T) {
		blacklistID := uuid.New()
		blacklist, err := service.GetCommandBlacklist(ctx, blacklistID)
		assert.NoError(t, err)
		assert.Nil(t, blacklist) // No blacklist found
	})

	t.Run("lists command blacklist", func(t *testing.T) {
		blacklists, err := service.ListCommandBlacklist(ctx, &tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, blacklists)
	})

	t.Run("updates command blacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			CommandPattern: "rm -rf *",
			Action:        "warn",
			Severity:      "high",
		}

		err := service.UpdateCommandBlacklist(ctx, blacklist)
		assert.NoError(t, err)
	})

	t.Run("deletes command blacklist", func(t *testing.T) {
		blacklistID := uuid.New()
		err := service.DeleteCommandBlacklist(ctx, blacklistID)
		assert.NoError(t, err)
	})

	t.Run("evaluates command against blacklist", func(t *testing.T) {
		allowed, action, blacklistID := service.EvaluateCommandAgainstBlacklist(ctx, tenantID, userID, "ls -la", nil)
		assert.True(t, allowed)
		assert.Empty(t, action)
		assert.Nil(t, blacklistID)
	})
}

func TestService_SSHKeyAnalytics(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	sshKeyID := uuid.New()
	userID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("records SSH key usage", func(t *testing.T) {
		err := service.RecordSSHKeyUsage(ctx, tenantID, sshKeyID, userID, "server1", 5*time.Minute, false)
		assert.NoError(t, err)
	})

	t.Run("records failed SSH key usage", func(t *testing.T) {
		err := service.RecordSSHKeyUsage(ctx, tenantID, sshKeyID, userID, "server1", 0, true)
		assert.NoError(t, err)
	})

	t.Run("gets SSH key analytics", func(t *testing.T) {
		analytics, err := service.GetSSHKeyAnalytics(ctx, tenantID, sshKeyID, dateFrom, dateTo)
		assert.NoError(t, err)
		assert.NotNil(t, analytics)
	})
}

func TestService_CacheManagement(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("invalidates cache", func(t *testing.T) {
		err := service.InvalidateCache(ctx, tenantID)
		assert.NoError(t, err)
	})

	t.Run("invalidates cache with types", func(t *testing.T) {
		err := service.InvalidateCache(ctx, tenantID, "sessions", "activity")
		assert.NoError(t, err)
	})

	t.Run("warms cache", func(t *testing.T) {
		err := service.WarmCache(ctx, tenantID, dateFrom, dateTo)
		assert.NoError(t, err)
	})

	t.Run("gets cache stats", func(t *testing.T) {
		stats, err := service.GetCacheStats(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})

	t.Run("handles nil cache gracefully", func(t *testing.T) {
		serviceWithoutCache := NewService(repo, nil, publisher, logger, config)

		err := serviceWithoutCache.InvalidateCache(ctx, tenantID)
		assert.NoError(t, err)

		err = serviceWithoutCache.WarmCache(ctx, tenantID, dateFrom, dateTo)
		assert.NoError(t, err)

		stats, err := serviceWithoutCache.GetCacheStats(ctx, tenantID)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})
}

func TestService_EventHandlers(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()

	t.Run("handles session started event", func(t *testing.T) {
		tenantID := uuid.New().String()
		userID := uuid.New().String()
		sessionID := uuid.New().String()

		event := events.Event{
			TenantID: tenantID,
			ActorID:  userID,
			Data: map[string]interface{}{
				"session_id":  sessionID,
				"target_host":  "server1.example.com",
				"target_port": float64(22),
				"type":        "ssh",
			},
		}

		err := service.HandleSessionStarted(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("handles session ended event", func(t *testing.T) {
		tenantID := uuid.New().String()
		userID := uuid.New().String()
		sessionID := uuid.New().String()
		duration := "30m"

		event := events.Event{
			TenantID: tenantID,
			ActorID:  userID,
			Data: map[string]interface{}{
				"session_id": sessionID,
				"duration":   duration,
				"type":      "ssh",
			},
		}

		err := service.HandleSessionEnded(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("handles command executed event", func(t *testing.T) {
		tenantID := uuid.New().String()
		userID := uuid.New().String()
		sessionID := uuid.New().String()

		event := events.Event{
			TenantID: tenantID,
			ActorID:  userID,
			Data: map[string]interface{}{
				"session_id":  sessionID,
				"command":     "ls -la",
				"target_host": "server1",
			},
		}

		err := service.HandleCommandExecuted(ctx, event)
		assert.NoError(t, err)
	})
}

func TestService_DisabledFeatures(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := ServiceConfig{
		EnableCompliance:        false,
		EnableAnomaly:           false,
		EnableCommandTracking:   false,
	}

	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("compliance disabled returns error", func(t *testing.T) {
		_, err := service.GenerateComplianceReport(ctx, tenantID, userID, FrameworkSOC2, time.Now(), time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("anomaly detection disabled returns error", func(t *testing.T) {
		_, err := service.RunAnomalyDetection(ctx, tenantID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("command tracking with disabled extractor", func(t *testing.T) {
		sessionID := uuid.New()
		err := service.RecordCommand(ctx, tenantID, userID, sessionID, "rm -rf /", "server1", nil)
		assert.NoError(t, err) // Should not error, just skip
	})
}

func TestService_isOffHour(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}
	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	tests := []struct {
		name     string
		hour     int
		expected bool
	}{
		{"midnight is off hours", 0, true},
		{"early morning is off hours", 5, true},
		{"6 AM is boundary", 6, false},
		{"morning is normal hours", 9, false},
		{"noon is normal hours", 12, false},
		{"6 PM is off hours start", 18, true},
		{"evening is off hours", 20, true},
		{"late night is off hours", 23, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isOffHour(tt.hour)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_ExportAnalyticsData(t *testing.T) {
	logger := zerolog.Nop()
	repo := &mockServiceRepository{}
	redisCache := NewRedisCache(&cache.Cache{}, logger)
	publisher := &events.Publisher{}

	config := DefaultServiceConfig()
	service := NewService(repo, redisCache, publisher, logger, config)

	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("exports analytics data", func(t *testing.T) {
		data, err := service.ExportAnalyticsData(ctx, tenantID, dateFrom, dateTo)
		assert.NoError(t, err)
		assert.Nil(t, data) // Returns nil, placeholder implementation
	})
}
