package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// Helper function to create float64 pointers
func ptrFloat(f float64) *float64 {
	return &f
}

// TestNewService tests service creation
func TestNewService(t *testing.T) {
	t.Run("creates service with valid inputs", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := &Repository{}

		service := NewService(repo, nil, nil, logger)

		assert.NotNil(t, service)
		assert.NotNil(t, service.logger)
		assert.NotNil(t, service.repo)
	})

	t.Run("service initializes with workers not running", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := &Repository{}

		service := NewService(repo, nil, nil, logger)

		assert.False(t, service.aggregationWorkerRunning)
		assert.False(t, service.alertEvaluatorRunning)
		assert.False(t, service.reportSchedulerRunning)
	})
}

// TestService_SessionAnalyticsStructure tests session analytics structure
func TestService_SessionAnalyticsStructure(t *testing.T) {
	t.Run("creates valid session analytics", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now().Truncate(time.Hour)
		avgDuration := 1800

		analytics := &SessionAnalytics{
			ID:                uuid.New(),
			TenantID:          tenantID,
			PeriodType:        PeriodHour,
			PeriodStart:       now,
			PeriodEnd:         now.Add(time.Hour),
			TotalSessions:     10,
			ActiveSessions:    2,
			CompletedSessions: 8,
			FailedSessions:    0,
			TerminatedSessions: 0,
			AvgDurationSeconds: &avgDuration,
			UniqueUsers:       5,
		}

		assert.Equal(t, tenantID, analytics.TenantID)
		assert.Equal(t, PeriodHour, analytics.PeriodType)
		assert.Equal(t, 10, analytics.TotalSessions)
		assert.Equal(t, 5, analytics.UniqueUsers)
	})
}

// TestService_UserActivityStructure tests user activity structure
func TestService_UserActivityStructure(t *testing.T) {
	t.Run("creates valid user activity", func(t *testing.T) {
		userID := uuid.New()
		tenantID := uuid.New()
		periodStart := time.Now().Truncate(24 * time.Hour)

		activity := &UserActivity{
			ID:                  uuid.New(),
			UserID:              userID,
			TenantID:            tenantID,
			PeriodStart:         periodStart,
			SessionsCreated:     10,
			CredentialsAccessed: 5,
			OffHoursAccess:      false,
			PolicyViolations:    2,
			FailedAuthAttempts:  1,
		}

		assert.Equal(t, userID, activity.UserID)
		assert.Equal(t, tenantID, activity.TenantID)
		assert.Equal(t, 10, activity.SessionsCreated)
		assert.Equal(t, 5, activity.CredentialsAccessed)
		assert.False(t, activity.OffHoursAccess)
		assert.Equal(t, 2, activity.PolicyViolations)
	})
}

// TestService_RiskScoreStructure tests risk score structure
func TestService_RiskScoreStructure(t *testing.T) {
	t.Run("creates valid risk score", func(t *testing.T) {
		entityID := uuid.New()
		tenantID := uuid.New()
		score := 75.5
		accessFreqScore := 80.0

		riskScore := &RiskScore{
			ID:                  uuid.New(),
			EntityID:            entityID,
			EntityType:          EntityUser,
			TenantID:            tenantID,
			CalculatedAt:        time.Now(),
			AccessFrequencyScore: &accessFreqScore,
			OverallRiskScore:    score,
			RiskLevel:           RiskLevelMedium,
		}

		assert.Equal(t, entityID, riskScore.EntityID)
		assert.Equal(t, EntityUser, riskScore.EntityType)
		assert.Equal(t, tenantID, riskScore.TenantID)
		assert.Equal(t, score, riskScore.OverallRiskScore)
		assert.Equal(t, RiskLevelMedium, riskScore.RiskLevel)
	})
}

// TestService_SessionStatsStructure tests session stats structure
func TestService_SessionStatsStructure(t *testing.T) {
	t.Run("creates valid session stats", func(t *testing.T) {
		stats := &SessionStats{
			Total:       100,
			Active:      15,
			Completed:   80,
			Failed:      5,
			Terminated:  0,
			AvgDuration: 1800,
			MaxDuration: 7200,
			ByProtocol: map[string]int64{
				"ssh": 80,
				"rdp": 20,
			},
		}

		assert.Equal(t, int64(100), stats.Total)
		assert.Equal(t, int64(15), stats.Active)
		assert.Equal(t, 1800, stats.AvgDuration)
		assert.NotEmpty(t, stats.ByProtocol)
	})
}

// TestService_EventStatsStructure tests event stats structure
func TestService_EventStatsStructure(t *testing.T) {
	t.Run("creates valid event stats", func(t *testing.T) {
		stats := &EventStats{
			Total:      500,
			Successful: 450,
			Failed:     50,
			Denied:     25,
		}

		assert.Equal(t, int64(500), stats.Total)
		assert.Equal(t, int64(450), stats.Successful)
		assert.Equal(t, int64(50), stats.Failed)
	})
}

// TestService_PeriodType tests period type constants
func TestService_PeriodType(t *testing.T) {
	t.Run("has correct period types", func(t *testing.T) {
		assert.Equal(t, PeriodType("hour"), PeriodHour)
		assert.Equal(t, PeriodType("day"), PeriodDay)
		assert.Equal(t, PeriodType("week"), PeriodWeek)
		assert.Equal(t, PeriodType("month"), PeriodMonth)
	})
}

// TestService_Severity tests severity constants
func TestService_Severity(t *testing.T) {
	t.Run("has correct severities", func(t *testing.T) {
		assert.Equal(t, Severity("low"), SeverityLow)
		assert.Equal(t, Severity("medium"), SeverityMedium)
		assert.Equal(t, Severity("high"), SeverityHigh)
		assert.Equal(t, Severity("critical"), SeverityCritical)
	})
}

// TestService_ReportType tests report type constants
func TestService_ReportType(t *testing.T) {
	t.Run("has correct report types", func(t *testing.T) {
		assert.Equal(t, ReportType("session_summary"), ReportTypeSessionSummary)
		assert.Equal(t, ReportType("access_patterns"), ReportTypeAccessPatterns)
		assert.Equal(t, ReportType("compliance"), ReportTypeCompliance)
		assert.Equal(t, ReportType("user_activity"), ReportTypeUserActivity)
		assert.Equal(t, ReportType("risk_analysis"), ReportTypeRiskAnalysis)
	})
}

// TestService_WidgetType tests widget type constants
func TestService_WidgetType(t *testing.T) {
	t.Run("has correct widget types", func(t *testing.T) {
		assert.Equal(t, WidgetType("line_chart"), WidgetTypeLineChart)
		assert.Equal(t, WidgetType("bar_chart"), WidgetTypeBarChart)
		assert.Equal(t, WidgetType("pie_chart"), WidgetTypePieChart)
		assert.Equal(t, WidgetType("stat_card"), WidgetTypeStatCard)
		assert.Equal(t, WidgetType("table"), WidgetTypeTable)
		assert.Equal(t, WidgetType("heatmap"), WidgetTypeHeatmap)
		assert.Equal(t, WidgetType("gauge"), WidgetTypeGauge)
	})
}

// TestService_DataSource tests data source constants
func TestService_DataSource(t *testing.T) {
	t.Run("has correct data sources", func(t *testing.T) {
		assert.Equal(t, DataSource("sessions"), DataSourceSessions)
		assert.Equal(t, DataSource("events"), DataSourceEvents)
		assert.Equal(t, DataSource("credentials"), DataSourceCredentials)
		assert.Equal(t, DataSource("users"), DataSourceUsers)
		assert.Equal(t, DataSource("risks"), DataSourceRisks)
	})
}

// TestService_EntityType tests entity type constants
func TestService_EntityType(t *testing.T) {
	t.Run("has correct entity types", func(t *testing.T) {
		assert.Equal(t, EntityType("user"), EntityUser)
		assert.Equal(t, EntityType("target"), EntityTarget)
		assert.Equal(t, EntityType("credential"), EntityCredential)
	})
}

// TestService_RiskLevel tests risk level constants
func TestService_RiskLevel(t *testing.T) {
	t.Run("has correct risk levels", func(t *testing.T) {
		assert.Equal(t, RiskLevel("low"), RiskLevelLow)
		assert.Equal(t, RiskLevel("medium"), RiskLevelMedium)
		assert.Equal(t, RiskLevel("high"), RiskLevelHigh)
		assert.Equal(t, RiskLevel("critical"), RiskLevelCritical)
	})
}

// TestService_MetricType tests metric type constants
func TestService_MetricType(t *testing.T) {
	t.Run("has correct metric types", func(t *testing.T) {
		assert.Equal(t, MetricType("gauge"), MetricTypeGauge)
		assert.Equal(t, MetricType("counter"), MetricTypeCounter)
		assert.Equal(t, MetricType("histogram"), MetricTypeHistogram)
		assert.Equal(t, MetricType("summary"), MetricTypeSummary)
	})
}

// TestService_ScheduleType tests schedule type constants
func TestService_ScheduleType(t *testing.T) {
	t.Run("has correct schedule types", func(t *testing.T) {
		assert.Equal(t, ScheduleType("daily"), ScheduleDaily)
		assert.Equal(t, ScheduleType("weekly"), ScheduleWeekly)
		assert.Equal(t, ScheduleType("monthly"), ScheduleMonthly)
	})
}
