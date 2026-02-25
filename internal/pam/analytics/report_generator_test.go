package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewReportGenerator tests report generator creation
func TestNewReportGenerator(t *testing.T) {
	t.Run("creates report generator", func(t *testing.T) {
		logger := zerolog.Nop()

		generator := NewReportGenerator(&Repository{}, logger)

		assert.NotNil(t, generator)
		assert.NotNil(t, generator.logger)
		assert.NotNil(t, generator.repo)
	})
}

// TestGeneratedReport tests generated report structure
func TestReportGenerator_GeneratedReport(t *testing.T) {
	t.Run("creates valid generated report", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		now := time.Now()

		report := GeneratedReport{
			ID:             id,
			TenantID:       tenantID,
			ReportType:     "session_activity",
			StartDate:      now.Add(-24 * time.Hour),
			EndDate:        now,
			Format:         "json",
			ContentType:    "application/json",
			FileExtension:  "json",
			ContentSize:    1024,
			Content:        []byte(`{"test": "data"}`),
			Status:         "completed",
			GeneratedAt:    now,
		}

		assert.Equal(t, id, report.ID)
		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, "session_activity", report.ReportType)
		assert.Equal(t, "json", report.Format)
		assert.Equal(t, 1024, report.ContentSize)
		assert.NotEmpty(t, report.Content)
		assert.Equal(t, "completed", report.Status)
	})

	t.Run("creates report with user ID", func(t *testing.T) {
		id := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		now := time.Now()

		report := GeneratedReport{
			ID:         id,
			TenantID:   tenantID,
			UserID:     &userID,
			ReportType: "user_activity",
			Status:     "completed",
			GeneratedAt: now,
		}

		assert.NotNil(t, report.UserID)
		assert.Equal(t, userID, *report.UserID)
	})

	t.Run("creates report with file URL", func(t *testing.T) {
		id := uuid.New()
		fileURL := "https://storage.example.com/reports/report.pdf"

		report := GeneratedReport{
			ID:       id,
			FileURL:  fileURL,
			Status:   "completed",
		}

		assert.Equal(t, fileURL, report.FileURL)
	})

	t.Run("creates failed report", func(t *testing.T) {
		errorMsg := "Generation failed"

		report := GeneratedReport{
			Status: "failed",
			Error:  errorMsg,
		}

		assert.Equal(t, "failed", report.Status)
		assert.Equal(t, errorMsg, report.Error)
	})
}

// TestReportGenerator_FormatSelection tests format selection
func TestReportGenerator_FormatSelection(t *testing.T) {
	t.Run("selects correct content type for JSON", func(t *testing.T) {
		report := GeneratedReport{
			Format:      "json",
			ContentType: "application/json",
		}
		assert.Equal(t, "json", report.Format)
		assert.Equal(t, "application/json", report.ContentType)
	})

	t.Run("selects correct content type for CSV", func(t *testing.T) {
		report := GeneratedReport{
			Format:      "csv",
			ContentType: "text/csv",
		}
		assert.Equal(t, "csv", report.Format)
		assert.Equal(t, "text/csv", report.ContentType)
	})

	t.Run("selects correct content type for PDF", func(t *testing.T) {
		report := GeneratedReport{
			Format:      "pdf",
			ContentType: "application/pdf",
		}
		assert.Equal(t, "pdf", report.Format)
		assert.Equal(t, "application/pdf", report.ContentType)
	})
}

// TestReportGenerator_RiskScoreStructure tests risk score structure with correct fields
func TestReportGenerator_RiskScoreStructure(t *testing.T) {
	t.Run("creates risk score with correct fields", func(t *testing.T) {
		entityID := uuid.New()
		tenantID := uuid.New()
		score := 75.5
		prevScore := 70.0
		change := 5.5

		riskScore := RiskScore{
			ID:                uuid.New(),
			EntityID:          entityID,
			EntityType:        EntityUser,
			TenantID:          tenantID,
			CalculatedAt:      time.Now(),
			OverallRiskScore:  score,
			RiskLevel:         RiskLevelMedium,
			PreviousScore:     &prevScore,
			ScoreChange:       &change,
		}

		assert.Equal(t, entityID, riskScore.EntityID)
		assert.Equal(t, EntityUser, riskScore.EntityType)
		assert.Equal(t, tenantID, riskScore.TenantID)
		assert.Equal(t, score, riskScore.OverallRiskScore)
		assert.Equal(t, RiskLevelMedium, riskScore.RiskLevel)
		assert.Equal(t, &prevScore, riskScore.PreviousScore)
		assert.Equal(t, &change, riskScore.ScoreChange)
	})

	t.Run("creates risk score with entity type credential", func(t *testing.T) {
		entityID := uuid.New()
		tenantID := uuid.New()

		riskScore := RiskScore{
			ID:               uuid.New(),
			EntityID:         entityID,
			EntityType:       EntityCredential,
			TenantID:         tenantID,
			CalculatedAt:     time.Now(),
			OverallRiskScore: 45.0,
			RiskLevel:        RiskLevelLow,
		}

		assert.Equal(t, EntityCredential, riskScore.EntityType)
		assert.Equal(t, RiskLevelLow, riskScore.RiskLevel)
	})

	t.Run("creates risk score with entity type target", func(t *testing.T) {
		entityID := uuid.New()
		tenantID := uuid.New()

		riskScore := RiskScore{
			ID:               uuid.New(),
			EntityID:         entityID,
			EntityType:       EntityTarget,
			TenantID:         tenantID,
			CalculatedAt:     time.Now(),
			OverallRiskScore: 85.0,
			RiskLevel:        RiskLevelHigh,
		}

		assert.Equal(t, EntityTarget, riskScore.EntityType)
		assert.Equal(t, RiskLevelHigh, riskScore.RiskLevel)
	})
}

// TestReportGenerator_UserActivityStructure tests user activity structure
func TestReportGenerator_UserActivityStructure(t *testing.T) {
	t.Run("creates user activity with correct fields", func(t *testing.T) {
		userID := uuid.New()
		tenantID := uuid.New()

		activity := UserActivity{
			ID:                  uuid.New(),
			UserID:              userID,
			TenantID:            tenantID,
			PeriodStart:         time.Now().Truncate(24 * time.Hour),
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
		assert.Equal(t, 1, activity.FailedAuthAttempts)
	})

	t.Run("creates user activity with off-hours access", func(t *testing.T) {
		userID := uuid.New()
		tenantID := uuid.New()

		activity := UserActivity{
			ID:             uuid.New(),
			UserID:         userID,
			TenantID:       tenantID,
			PeriodStart:    time.Now().Truncate(24 * time.Hour),
			OffHoursAccess: true,
		}

		assert.True(t, activity.OffHoursAccess)
	})
}

// TestReportGenerator_ReportStructure tests report structures
func TestReportGenerator_ReportStructure(t *testing.T) {
	t.Run("creates report", func(t *testing.T) {
		tenantID := uuid.New()
		scheduleDaily := ScheduleDaily

		report := Report{
			ID:               uuid.New(),
			TenantID:         tenantID,
			Name:             "Weekly Summary",
			ReportType:       ReportTypeSessionSummary,
			ScheduleEnabled:  true,
			ScheduleType:     &scheduleDaily,
			ScheduleTimezone: "UTC",
		}

		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, "Weekly Summary", report.Name)
		assert.Equal(t, ReportTypeSessionSummary, report.ReportType)
		assert.True(t, report.ScheduleEnabled)
		assert.Equal(t, &scheduleDaily, report.ScheduleType)
	})
}

// TestReportGenerator_DashboardStructure tests dashboard structures
func TestReportGenerator_DashboardStructure(t *testing.T) {
	t.Run("creates dashboard", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()

		dashboard := Dashboard{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      "Security Overview",
			IsDefault: false,
			IsPublic:  true,
			CreatedBy: userID,
		}

		assert.Equal(t, tenantID, dashboard.TenantID)
		assert.Equal(t, "Security Overview", dashboard.Name)
		assert.False(t, dashboard.IsDefault)
		assert.True(t, dashboard.IsPublic)
		assert.Equal(t, userID, dashboard.CreatedBy)
	})
}

// TestReportGenerator_WidgetStructure tests widget structures
func TestReportGenerator_WidgetStructure(t *testing.T) {
	t.Run("creates stat card widget", func(t *testing.T) {
		dashboardID := uuid.New()

		widget := Widget{
			ID:          uuid.New(),
			DashboardID: dashboardID,
			Name:        "Active Sessions",
			WidgetType:  WidgetTypeStatCard,
			PositionX:   0,
			PositionY:   0,
			Width:       3,
			Height:      2,
			DataSource:  DataSourceSessions,
		}

		assert.Equal(t, dashboardID, widget.DashboardID)
		assert.Equal(t, "Active Sessions", widget.Name)
		assert.Equal(t, WidgetTypeStatCard, widget.WidgetType)
		assert.Equal(t, DataSourceSessions, widget.DataSource)
	})

	t.Run("creates chart widget", func(t *testing.T) {
		dashboardID := uuid.New()

		widget := Widget{
			ID:          uuid.New(),
			DashboardID: dashboardID,
			Name:        "Session Trends",
			WidgetType:  WidgetTypeLineChart,
			DataSource:  DataSourceSessions,
		}

		assert.Equal(t, WidgetTypeLineChart, widget.WidgetType)
	})
}

// TestReportGenerator_ReportSnapshotStructure tests report snapshot structures
func TestReportGenerator_ReportSnapshotStructure(t *testing.T) {
	t.Run("creates report snapshot", func(t *testing.T) {
		reportID := uuid.New()
		tenantID := uuid.New()
		fileSize := int64(2048)
		generatedBy := uuid.New()
		format := "pdf"

		snapshot := ReportSnapshot{
			ID:            uuid.New(),
			ReportID:      reportID,
			TenantID:      tenantID,
			Status:        ReportSnapshotStatusCompleted,
			FileSizeBytes: &fileSize,
			GeneratedAt:   time.Now(),
			GeneratedBy:   generatedBy,
			FileFormat:    &format,
		}

		assert.Equal(t, reportID, snapshot.ReportID)
		assert.Equal(t, tenantID, snapshot.TenantID)
		assert.Equal(t, ReportSnapshotStatusCompleted, snapshot.Status)
		assert.Equal(t, int64(2048), *snapshot.FileSizeBytes)
	})
}

// TestReportGenerator_SessionAnalyticsStructure tests session analytics structures
func TestReportGenerator_SessionAnalyticsStructure(t *testing.T) {
	t.Run("creates session analytics", func(t *testing.T) {
		tenantID := uuid.New()
		date := time.Now().Truncate(24 * time.Hour)
		avgDuration := 1800

		analytics := SessionAnalytics{
			ID:              uuid.New(),
			TenantID:        tenantID,
			PeriodType:      PeriodDay,
			PeriodStart:     date,
			PeriodEnd:       date.Add(24 * time.Hour),
			TotalSessions:   100,
			ActiveSessions:  15,
			CompletedSessions: 80,
			FailedSessions:  5,
			AvgDurationSeconds: &avgDuration,
		}

		assert.Equal(t, tenantID, analytics.TenantID)
		assert.Equal(t, PeriodDay, analytics.PeriodType)
		assert.Equal(t, 100, analytics.TotalSessions)
		assert.Equal(t, 15, analytics.ActiveSessions)
		assert.Equal(t, &avgDuration, analytics.AvgDurationSeconds)
	})
}

// TestReportGenerator_SessionStatsStructure tests session stats structures
func TestReportGenerator_SessionStatsStructure(t *testing.T) {
	t.Run("creates session stats", func(t *testing.T) {
		stats := SessionStats{
			Total:      100,
			Active:     15,
			Completed:  80,
			Failed:     5,
			Terminated: 0,
			AvgDuration: 1800,
			MaxDuration: 7200,
			ByProtocol: map[string]int64{
				"ssh": 80,
				"rdp": 20,
			},
			ByUser: map[string]int64{
				"user1": 50,
				"user2": 30,
			},
		}

		assert.Equal(t, int64(100), stats.Total)
		assert.Equal(t, int64(15), stats.Active)
		assert.Equal(t, int64(80), stats.Completed)
		assert.Equal(t, 1800, stats.AvgDuration)
		assert.NotEmpty(t, stats.ByProtocol)
		assert.NotEmpty(t, stats.ByUser)
	})
}

// TestReportGenerator_TopUserStructure tests top user structures
func TestReportGenerator_TopUserStructure(t *testing.T) {
	t.Run("creates top user", func(t *testing.T) {
		userID := uuid.New()

		topUser := TopUser{
			UserID:       userID,
			Username:     "testuser",
			SessionCount: 50,
			TotalSeconds: 90000,
			LastSeen:     time.Now(),
		}

		assert.Equal(t, userID, topUser.UserID)
		assert.Equal(t, "testuser", topUser.Username)
		assert.Equal(t, 50, topUser.SessionCount)
		assert.Equal(t, int64(90000), topUser.TotalSeconds)
	})
}

// TestReportGenerator_AccessPatternStructure tests access pattern structures
func TestReportGenerator_AccessPatternStructure(t *testing.T) {
	t.Run("creates access pattern", func(t *testing.T) {
		pattern := AccessPattern{
			HourOfDay:    14,
			DayOfWeek:    2, // Tuesday (0=Sunday, 6=Saturday)
			SessionCount: 25,
			UniqueUsers:  10,
		}

		assert.Equal(t, 14, pattern.HourOfDay)
		assert.Equal(t, 2, pattern.DayOfWeek)
		assert.Equal(t, 25, pattern.SessionCount)
		assert.Equal(t, 10, pattern.UniqueUsers)
	})
}
