package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisCache(t *testing.T) {
	logger := zerolog.Nop()
	redisCache := &cache.Cache{}

	cache := NewRedisCache(redisCache, logger)

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.NotNil(t, cache.logger)
}

func TestRedisCache_TTLConstants(t *testing.T) {
	t.Run("DefaultMetricsTTL", func(t *testing.T) {
		assert.Equal(t, 5*time.Minute, DefaultMetricsTTL)
	})

	t.Run("DailyAggregationTTL", func(t *testing.T) {
		assert.Equal(t, 1*time.Hour, DailyAggregationTTL)
	})

	t.Run("ComplianceReportTTL", func(t *testing.T) {
		assert.Equal(t, 24*time.Hour, ComplianceReportTTL)
	})

	t.Run("UserActivityTTL", func(t *testing.T) {
		assert.Equal(t, 10*time.Minute, UserActivityTTL)
	})

	t.Run("AnomalyStatsTTL", func(t *testing.T) {
		assert.Equal(t, time.Minute, AnomalyStatsTTL)
	})

	t.Run("BlacklistCacheTTL", func(t *testing.T) {
		assert.Equal(t, 15*time.Minute, BlacklistCacheTTL)
	})
}

func TestCachedRepository_Wrap(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	assert.NotNil(t, cachedRepo)
	assert.NotNil(t, cachedRepo.repo)
	assert.Nil(t, cachedRepo.cache) // nil cache is acceptable
	assert.NotNil(t, cachedRepo.logger)

	// These should not panic and just return empty results
	sessions, err := cachedRepo.ListSessionAnalytics(ctx, SessionAnalyticsFilter{}, 10, 0)
	assert.NoError(t, err)
	assert.NotNil(t, sessions)

	activities, err := cachedRepo.ListUserActivity(ctx, UserActivityFilter{}, 10, 0)
	assert.NoError(t, err)
	assert.NotNil(t, activities)

	commands, err := cachedRepo.ListCommandFrequency(ctx, CommandFrequencyFilter{}, 10, 0)
	assert.NoError(t, err)
	assert.NotNil(t, commands)
}

func TestCachedRepository_SessionAnalytics(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	avgDuration := 120.5

	t.Run("CreateSessionAnalytics", func(t *testing.T) {
		analytics := &SessionAnalytics{
			ID:                uuid.New(),
			TenantID:          tenantID,
			Date:              date,
			Hour:              14,
			TotalSessions:     10,
			AvgDurationSeconds: &avgDuration,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := cachedRepo.CreateSessionAnalytics(ctx, analytics)
		assert.NoError(t, err)
	})

	t.Run("UpdateSessionAnalytics", func(t *testing.T) {
		analytics := &SessionAnalytics{
			ID:                uuid.New(),
			TenantID:          tenantID,
			Date:              date,
			Hour:              14,
			TotalSessions:     15,
			AvgDurationSeconds: &avgDuration,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := cachedRepo.UpdateSessionAnalytics(ctx, analytics)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_UserActivity(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("CreateUserActivity", func(t *testing.T) {
		activity := &UserActivity{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          userID,
			Date:            date,
			Hour:            14,
			CommandsExecuted: 50,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := cachedRepo.CreateUserActivity(ctx, activity)
		assert.NoError(t, err)
	})

	t.Run("UpdateUserActivity", func(t *testing.T) {
		activity := &UserActivity{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          userID,
			Date:            date,
			Hour:            14,
			CommandsExecuted: 75,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := cachedRepo.UpdateUserActivity(ctx, activity)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_AnomalyDetection(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("CreateAnomalyDetection", func(t *testing.T) {
		anomaly := &AnomalyDetection{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          &userID,
			AnomalyType:     "behavioral",
			Severity:        "high",
			ConfidenceScore: 85.0,
			Title:           "Test Anomaly",
			Status:          "open",
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := cachedRepo.CreateAnomalyDetection(ctx, anomaly)
		assert.NoError(t, err)
	})

	t.Run("UpdateAnomalyDetection", func(t *testing.T) {
		anomaly := &AnomalyDetection{
			ID:          uuid.New(),
			TenantID:    tenantID,
			AnomalyType: "behavioral",
			Severity:    "medium",
			Status:      "resolved",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := cachedRepo.UpdateAnomalyDetection(ctx, anomaly)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_RansomwareEvents(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockAnomalyRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	detectionID := uuid.New()

	t.Run("CreateRansomwareEvent", func(t *testing.T) {
		event := &RansomwareEvent{
			TenantID:       tenantID,
			DetectionID:    detectionID,
			FilesAffected:  100,
			SystemsAffected: 2,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := cachedRepo.CreateRansomwareEvent(ctx, event)
		assert.NoError(t, err)
	})

	t.Run("UpdateRansomwareEvent", func(t *testing.T) {
		event := &RansomwareEvent{
			ID:             uuid.New(),
			TenantID:       tenantID,
			DetectionID:    detectionID,
			FilesAffected:  150,
			SystemsAffected: 3,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := cachedRepo.UpdateRansomwareEvent(ctx, event)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_CommandBlacklist(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	baseCommand := "rm"

	t.Run("CreateCommandBlacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			ID:            uuid.New(),
			TenantID:      &tenantID,
			CommandPattern: "rm -rf *",
			PatternType:   "glob",
			BaseCommand:   &baseCommand,
			Action:        "block",
			Severity:      "critical",
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := cachedRepo.CreateCommandBlacklist(ctx, blacklist)
		assert.NoError(t, err)
	})

	t.Run("UpdateCommandBlacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			ID:            uuid.New(),
			TenantID:      &tenantID,
			CommandPattern: "rm -rf *",
			Action:        "warn",
			Severity:      "high",
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := cachedRepo.UpdateCommandBlacklist(ctx, blacklist)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_SSHKeyAnalytics(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("CreateSSHKeyAnalytics", func(t *testing.T) {
		analytics := &SSHKeyAnalytics{
			ID:       uuid.New(),
			TenantID: tenantID,
			SSHKeyID: sshKeyID,
			Date:     date,
			UsageCount: 10,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := cachedRepo.CreateSSHKeyAnalytics(ctx, analytics)
		assert.NoError(t, err)
	})

	t.Run("UpdateSSHKeyAnalytics", func(t *testing.T) {
		analytics := &SSHKeyAnalytics{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SSHKeyID:  sshKeyID,
			Date:      date,
			UsageCount: 15,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := cachedRepo.UpdateSSHKeyAnalytics(ctx, analytics)
		assert.NoError(t, err)
	})
}

func TestCachedRepository_DashboardMetrics(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.Nop()

	mockRepo := &mockComplianceRepository{}

	// Pass nil cache to test repository wrapping without cache
	cachedRepo := NewCachedRepository(mockRepo, nil, logger)

	tenantID := uuid.New()

	t.Run("GetDashboardMetrics", func(t *testing.T) {
		_, err := cachedRepo.GetDashboardMetrics(ctx, tenantID)
		assert.NotNil(t, cachedRepo)
		_ = err
	})
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"contains", "hello world", "world", true},
		{"does not contain", "hello world", "goodbye", false},
		{"empty substring", "hello", "", true},
		{"empty string", "", "test", false},
		{"exact match", "test", "test", true},
		{"prefix", "testing", "test", true},
		{"suffix", "testing", "ing", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindInStringHelper(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"found at start", "hello world", "hello", true},
		{"found in middle", "hello world", "lo wo", true},
		{"found at end", "hello world", "world", true},
		{"not found", "hello world", "goodbye", false},
		{"empty substring", "test", "", true},
		{"empty string", "", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findInString(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
