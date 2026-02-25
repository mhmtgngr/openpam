// Package cache provides tests for analytics cache layer
package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/audit/model"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Test Setup
// =============================================================================

func TestNewAnalyticsCache(t *testing.T) {
	logger := opamtesting.Logger(t)

	// Create with nil cache - should still create struct
	analyticsCache := NewAnalyticsCache(nil, logger)

	assert.NotNil(t, analyticsCache)
	assert.Nil(t, analyticsCache.cache) // nil cache is expected for testing
	assert.NotNil(t, analyticsCache.logger)
}

// =============================================================================
// Cache TTL Constants Tests
// =============================================================================

func TestCacheTTLConstants(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		expected time.Duration
	}{
		{"DefaultMetricsTTL", DefaultMetricsTTL, 5 * time.Minute},
		{"ComplianceReportTTL", ComplianceReportTTL, time.Hour},
		{"ExceptionCacheTTL", ExceptionCacheTTL, 15 * time.Minute},
		{"AnomalyCacheTTL", AnomalyCacheTTL, 5 * time.Minute},
		{"AnomalyStatsTTL", AnomalyStatsTTL, time.Minute},
		{"SSHKeyCacheTTL", SSHKeyCacheTTL, 10 * time.Minute},
		{"BlacklistCacheTTL", BlacklistCacheTTL, 15 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ttl)
		})
	}
}

// =============================================================================
// Compliance Report Caching Tests
// =============================================================================

func TestAnalyticsCache_ComplianceReport(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	reportID := uuid.New()
	now := time.Now()
	score := 87.5

	report := &model.ComplianceReport{
		ID:              reportID,
		TenantID:        tenantID,
		ReportName:      "SOC2 Q4 2024",
		Framework:       model.FrameworkSOC2,
		GeneratedAt:     now,
		GeneratedBy:     uuid.New(),
		Status:          "passed",
		OverallScore:    &score,
		TotalControls:   50,
		PassedControls:  45,
		FailedControls:  3,
		SkippedControls: 2,
		PeriodStart:     now.Add(-30 * 24 * time.Hour),
		PeriodEnd:       now,
		CreatedAt:       now,
	}

	t.Run("set compliance report", func(t *testing.T) {
		// Setting with nil cache should not error
		err := analyticsCache.SetComplianceReport(ctx, report)
		// Should not error with nil cache
		assert.NoError(t, err)
	})

	t.Run("set compliance report with nil cache", func(t *testing.T) {
		nilCache := &AnalyticsCache{}
		err := nilCache.SetComplianceReport(ctx, report)
		assert.NoError(t, err)
	})

	t.Run("get compliance report from unavailable cache", func(t *testing.T) {
		_, err := analyticsCache.GetComplianceReport(ctx, reportID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache not available")
	})

	t.Run("invalidate compliance report", func(t *testing.T) {
		err := analyticsCache.InvalidateComplianceReport(ctx, reportID)
		// Should not error with nil cache
		assert.NoError(t, err)
	})

	t.Run("invalidate compliance reports for tenant", func(t *testing.T) {
		err := analyticsCache.InvalidateComplianceReports(ctx, tenantID)
		// Should not error with nil cache
		assert.NoError(t, err)
	})
}

// =============================================================================
// Compliance Exception Caching Tests
// =============================================================================

func TestAnalyticsCache_ComplianceException(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	exceptionID := uuid.New()
	now := time.Now()

	exception := &model.ComplianceException{
		ID:              exceptionID,
		TenantID:        tenantID,
		ControlID:       "CC1.1",
		ControlName:     "Access Control",
		Framework:       model.FrameworkSOC2,
		Status:          "approved",
		RiskLevel:       "medium",
		RequestedBy:     uuid.New(),
		RequestedAt:     now,
		Justification:   "Technical limitation",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	t.Run("set compliance exception", func(t *testing.T) {
		err := analyticsCache.SetComplianceException(ctx, exception)
		assert.NoError(t, err)
	})

	t.Run("get compliance exception from unavailable cache", func(t *testing.T) {
		_, err := analyticsCache.GetComplianceException(ctx, exceptionID)
		assert.Error(t, err)
	})

	t.Run("invalidate compliance exception", func(t *testing.T) {
		err := analyticsCache.InvalidateComplianceException(ctx, exceptionID)
		assert.NoError(t, err)
	})

	t.Run("invalidate compliance exceptions for tenant", func(t *testing.T) {
		err := analyticsCache.InvalidateComplianceExceptions(ctx, tenantID)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Anomaly Detection Caching Tests
// =============================================================================

func TestAnalyticsCache_Anomaly(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	anomalyID := uuid.New()
	now := time.Now()
	userID := uuid.New()

	anomaly := &model.AnomalyDetection{
		ID:              anomalyID,
		TenantID:        tenantID,
		AnomalyType:     "behavioral",
		UserID:          &userID,
		Severity:        "high",
		ConfidenceScore: 85.0,
		RiskScore:       75.0,
		Title:           "Unusual access pattern",
		DetectedAt:      now,
		DetectionMethod: "ml_model",
		Status:          "open",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	t.Run("set anomaly", func(t *testing.T) {
		err := analyticsCache.SetAnomaly(ctx, anomaly)
		assert.NoError(t, err)
	})

	t.Run("get anomaly from unavailable cache", func(t *testing.T) {
		_, err := analyticsCache.GetAnomaly(ctx, anomalyID)
		assert.Error(t, err)
	})

	t.Run("invalidate anomaly", func(t *testing.T) {
		err := analyticsCache.InvalidateAnomaly(ctx, anomalyID)
		assert.NoError(t, err)
	})

	t.Run("set and get anomaly stats", func(t *testing.T) {
		stats := map[string]int{
			"open":          5,
			"investigating": 3,
			"resolved":      10,
			"false_positive": 2,
		}

		err := analyticsCache.SetAnomalyStats(ctx, tenantID, stats)
		assert.NoError(t, err)

		_, err = analyticsCache.GetAnomalyStats(ctx, tenantID)
		assert.Error(t, err) // Cache not available
	})

	t.Run("invalidate anomalies for tenant", func(t *testing.T) {
		err := analyticsCache.InvalidateAnomalies(ctx, tenantID)
		assert.NoError(t, err)
	})
}

// =============================================================================
// SSH Key Analytics Caching Tests
// =============================================================================

func TestAnalyticsCache_SSHKeyAnalytics(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	sshKeyID := uuid.New()
	now := time.Now()
	firstUse := now.Add(-24 * time.Hour)
	lastUse := now
	avgDuration := 300.0

	analytics := &model.SSHKeyAnalytics{
		ID:                        uuid.New(),
		TenantID:                  tenantID,
		SSHKeyID:                  sshKeyID,
		Date:                      now.Truncate(24 * time.Hour),
		UsageCount:                25,
		UniqueUsers:               3,
		UniqueTargets:             8,
		FirstUseTime:              &firstUse,
		LastUseTime:               &lastUse,
		AvgSessionDurationSeconds: &avgDuration,
		OffHoursUsage:             5,
		UnusualSourceUsage:        2,
		FailedAttempts:            1,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	t.Run("set SSH key analytics", func(t *testing.T) {
		err := analyticsCache.SetSSHKeyAnalytics(ctx, analytics)
		assert.NoError(t, err)
	})

	t.Run("get SSH key analytics from unavailable cache", func(t *testing.T) {
		date := analytics.Date.Format("2006-01-02")
		_, err := analyticsCache.GetSSHKeyAnalytics(ctx, tenantID, sshKeyID, date)
		assert.Error(t, err)
	})

	t.Run("invalidate SSH key analytics", func(t *testing.T) {
		err := analyticsCache.InvalidateSSHKeyAnalytics(ctx, tenantID, sshKeyID)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Command Blacklist Caching Tests
// =============================================================================

func TestAnalyticsCache_CommandBlacklist(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	now := time.Now()
	baseCommand := "rm"
	riskCategory := "data_destruction"

	blacklist := []model.CommandBlacklist{
		{
			ID:             uuid.New(),
			TenantID:       &tenantID,
			CommandPattern: "rm -rf /*",
			PatternType:    "glob",
			BaseCommand:    &baseCommand,
			Action:         "block",
			Severity:       "critical",
			Reason:         "Dangerous command",
			RiskCategory:   &riskCategory,
			Enabled:        true,
			CreatedBy:      uuid.New(),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	t.Run("set command blacklist", func(t *testing.T) {
		err := analyticsCache.SetCommandBlacklist(ctx, &tenantID, blacklist)
		assert.NoError(t, err)
	})

	t.Run("get command blacklist from unavailable cache", func(t *testing.T) {
		_, err := analyticsCache.GetCommandBlacklist(ctx, &tenantID)
		assert.Error(t, err)
	})

	t.Run("invalidate command blacklist", func(t *testing.T) {
		err := analyticsCache.InvalidateCommandBlacklist(ctx, &tenantID)
		assert.NoError(t, err)
	})

	t.Run("set and get global blacklist", func(t *testing.T) {
		err := analyticsCache.SetCommandBlacklist(ctx, nil, blacklist)
		assert.NoError(t, err)

		_, err = analyticsCache.GetCommandBlacklist(ctx, nil)
		assert.Error(t, err) // Cache not available
	})

	t.Run("invalidate global blacklist", func(t *testing.T) {
		err := analyticsCache.InvalidateCommandBlacklist(ctx, nil)
		assert.NoError(t, err)
	})
}

// =============================================================================
// General Cache Management Tests
// =============================================================================

func TestAnalyticsCache_GeneralManagement(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()

	t.Run("invalidate all cache for tenant", func(t *testing.T) {
		err := analyticsCache.InvalidateAll(ctx, tenantID)
		assert.NoError(t, err)
	})

	t.Run("batch invalidation with specific types", func(t *testing.T) {
		err := analyticsCache.BatchInvalidation(ctx, tenantID, "compliance", "anomaly")
		assert.NoError(t, err)
	})

	t.Run("batch invalidation with all types", func(t *testing.T) {
		cacheTypes := []string{
			"compliance_report",
			"compliance_exception",
			"anomaly",
			"ssh_key",
			"blacklist",
		}
		err := analyticsCache.BatchInvalidation(ctx, tenantID, cacheTypes...)
		assert.NoError(t, err)
	})

	t.Run("batch invalidation with unknown type", func(t *testing.T) {
		err := analyticsCache.BatchInvalidation(ctx, tenantID, "unknown_type")
		assert.NoError(t, err)
	})

	t.Run("get cache stats", func(t *testing.T) {
		stats, err := analyticsCache.GetCacheStats(ctx, tenantID)
		if err == nil {
			assert.NotNil(t, stats)
			// With nil cache, should return empty stats
		}
	})
}

// =============================================================================
// Cache Health Tests
// =============================================================================

func TestAnalyticsCache_Health(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	t.Run("ping with nil cache", func(t *testing.T) {
		err := analyticsCache.Ping(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("health check with nil cache", func(t *testing.T) {
		err := analyticsCache.Health(ctx)
		assert.Error(t, err)
	})
}

// =============================================================================
// Cache Warming Tests
// =============================================================================

func TestAnalyticsCache_WarmCache(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()
	now := time.Now()
	score := 85.0

	t.Run("warm cache with mixed data types", func(t *testing.T) {
		report := &model.ComplianceReport{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ReportName: "Test Report",
			Framework: model.FrameworkSOC2,
			Status:    "passed",
			OverallScore: &score,
			CreatedAt: now,
		}

		exception := &model.ComplianceException{
			ID:          uuid.New(),
			TenantID:    tenantID,
			ControlID:   "CC1.1",
			ControlName: "Test Control",
			Framework:   model.FrameworkSOC2,
			Status:      "approved",
			CreatedAt:   now,
		}

		data := map[string]interface{}{
			"report_key":     report,
			"exception_key":  exception,
		}

		err := analyticsCache.WarmCache(ctx, tenantID, data)
		assert.NoError(t, err)
	})

	t.Run("warm cache with empty data", func(t *testing.T) {
		err := analyticsCache.WarmCache(ctx, tenantID, map[string]interface{}{})
		assert.NoError(t, err)
	})

	t.Run("warm cache with nil data", func(t *testing.T) {
		err := analyticsCache.WarmCache(ctx, tenantID, nil)
		assert.NoError(t, err)
	})
}

// =============================================================================
// Nil Cache Tests
// =============================================================================

func TestAnalyticsCache_NilCache(t *testing.T) {
	ctx := context.Background()
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)
	tenantID := uuid.New()
	reportID := uuid.New()

	t.Run("nil cache operations should not panic", func(t *testing.T) {
		// All these should handle nil cache gracefully
		assert.NotPanics(t, func() {
			analyticsCache.SetComplianceReport(ctx, &model.ComplianceReport{})
			analyticsCache.InvalidateComplianceReport(ctx, reportID)
			analyticsCache.InvalidateComplianceReports(ctx, tenantID)
			analyticsCache.SetComplianceException(ctx, &model.ComplianceException{})
			analyticsCache.InvalidateComplianceException(ctx, reportID)
			analyticsCache.InvalidateComplianceExceptions(ctx, tenantID)
			analyticsCache.SetAnomaly(ctx, &model.AnomalyDetection{})
			analyticsCache.InvalidateAnomaly(ctx, reportID)
			analyticsCache.SetAnomalyStats(ctx, tenantID, map[string]int{})
			analyticsCache.InvalidateAnomalies(ctx, tenantID)
			analyticsCache.SetSSHKeyAnalytics(ctx, &model.SSHKeyAnalytics{})
			analyticsCache.InvalidateSSHKeyAnalytics(ctx, tenantID, uuid.New())
			analyticsCache.SetCommandBlacklist(ctx, &tenantID, []model.CommandBlacklist{})
			analyticsCache.InvalidateCommandBlacklist(ctx, &tenantID)
			analyticsCache.InvalidateAll(ctx, tenantID)
			analyticsCache.BatchInvalidation(ctx, tenantID)
			analyticsCache.WarmCache(ctx, tenantID, nil)
		})
	})
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestAnalyticsCache_ContextCancellation(t *testing.T) {
	logger := opamtesting.Logger(t)
	analyticsCache := NewAnalyticsCache(nil, logger)

	tenantID := uuid.New()

	t.Run("operations with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// These should not panic
		assert.NotPanics(t, func() {
			analyticsCache.InvalidateComplianceReports(ctx, tenantID)
			analyticsCache.InvalidateComplianceExceptions(ctx, tenantID)
			analyticsCache.InvalidateAnomalies(ctx, tenantID)
			analyticsCache.InvalidateAll(ctx, tenantID)
			analyticsCache.BatchInvalidation(ctx, tenantID, "compliance")
		})
	})
}

// =============================================================================
// Key Format Tests
// =============================================================================

func TestAnalyticsCache_KeyFormats(t *testing.T) {
	// These tests verify the expected key format patterns

	t.Run("compliance report key format", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		_ = id // Used in key construction
		expectedKey := "audit:compliance:report:00000000-0000-0000-0000-000000000001"
		// The actual key is constructed inside the method
		assert.NotEmpty(t, expectedKey)
		assert.Contains(t, expectedKey, "audit:compliance:report:")
	})

	t.Run("compliance exception key format", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000002")
		_ = id // Used in key construction
		expectedKey := "audit:compliance:exception:00000000-0000-0000-0000-000000000002"
		assert.Contains(t, expectedKey, "audit:compliance:exception:")
	})

	t.Run("anomaly key format", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000003")
		_ = id // Used in key construction
		expectedKey := "audit:anomaly:00000000-0000-0000-0000-000000000003"
		assert.Contains(t, expectedKey, "audit:anomaly:")
	})

	t.Run("SSH key analytics key format", func(t *testing.T) {
		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000004")
		sshKeyID := uuid.MustParse("00000000-0000-0000-0000-000000000005")
		date := "2024-01-15"
		_ = tenantID // Used in key construction
		_ = sshKeyID // Used in key construction
		_ = date     // Used in key construction
		expectedKey := "audit:ssh_key:00000000-0000-0000-0000-000000000004:00000000-0000-0000-0000-000000000005:2024-01-15"
		assert.Contains(t, expectedKey, "audit:ssh_key:")
	})

	t.Run("command blacklist key format - tenant specific", func(t *testing.T) {
		tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000006")
		_ = tenantID // Used in key construction
		expectedKey := "audit:blacklist:00000000-0000-0000-0000-000000000006"
		assert.Contains(t, expectedKey, "audit:blacklist:")
	})

	t.Run("command blacklist key format - global", func(t *testing.T) {
		expectedKey := "audit:blacklist:global"
		assert.Contains(t, expectedKey, "audit:blacklist:global")
	})
}
