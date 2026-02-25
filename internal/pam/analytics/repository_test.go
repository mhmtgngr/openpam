package analytics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewRepository tests repository creation
func TestNewRepository(t *testing.T) {
	t.Run("creates repository with valid inputs", func(t *testing.T) {
		logger := zerolog.Nop()
		repo := NewRepository(nil, logger)

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.logger)
		assert.Nil(t, repo.Db)
	})

	t.Run("creates repository with database", func(t *testing.T) {
		logger := zerolog.Nop()
		db, err := sqlx.Connect("sqlite3", ":memory:")
		if err != nil {
			t.Skip("requires sqlite")
		}
		defer db.Close()

		repo := NewRepository(db, logger)

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.Db)
		assert.NotNil(t, repo.logger)
	})
}

// TestRepository_SessionAnalyticsStructure tests session analytics structure
func TestRepository_SessionAnalyticsStructure(t *testing.T) {
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
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		assert.Equal(t, tenantID, analytics.TenantID)
		assert.Equal(t, PeriodHour, analytics.PeriodType)
		assert.Equal(t, 10, analytics.TotalSessions)
		assert.Equal(t, 5, analytics.UniqueUsers)
	})
}

// TestRepository_EventAnalyticsStructure tests event analytics structure
func TestRepository_EventAnalyticsStructure(t *testing.T) {
	t.Run("creates valid event analytics", func(t *testing.T) {
		tenantID := uuid.New()
		now := time.Now().Truncate(time.Hour)

		analytics := &EventAnalytics{
			ID:          uuid.New(),
			TenantID:    tenantID,
			PeriodType:  PeriodDay,
			PeriodStart: now,
			PeriodEnd:   now.Add(24 * time.Hour),
			TotalEvents: 100,
			SuccessfulEvents: 90,
			FailedEvents:    10,
			DeniedEvents:    5,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		assert.Equal(t, tenantID, analytics.TenantID)
		assert.Equal(t, PeriodDay, analytics.PeriodType)
		assert.Equal(t, 100, analytics.TotalEvents)
		assert.Equal(t, 90, analytics.SuccessfulEvents)
	})
}

// TestRepository_ReportStructure tests report structure
func TestRepository_ReportStructure(t *testing.T) {
	t.Run("creates valid report", func(t *testing.T) {
		tenantID := uuid.New()
		scheduleWeekly := ScheduleWeekly

		report := &Report{
			ID:              uuid.New(),
			TenantID:        tenantID,
			Name:            "Daily Summary",
			Description:     "Daily security summary",
			ReportType:      ReportTypeSessionSummary,
			ScheduleEnabled: true,
			ScheduleType:    &scheduleWeekly,
			ScheduleTimezone: "UTC",
		}

		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, "Daily Summary", report.Name)
		assert.Equal(t, ReportTypeSessionSummary, report.ReportType)
		assert.True(t, report.ScheduleEnabled)
	})
}

// TestRepository_DashboardStructure tests dashboard structure
func TestRepository_DashboardStructure(t *testing.T) {
	t.Run("creates valid dashboard", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		layout := json.RawMessage(`{"widgets": []}`)

		dashboard := &Dashboard{
			ID:        uuid.New(),
			TenantID:  tenantID,
			Name:      "Security Dashboard",
			IsDefault: false,
			IsPublic:  true,
			Layout:    layout,
			CreatedBy: userID,
		}

		assert.Equal(t, tenantID, dashboard.TenantID)
		assert.Equal(t, "Security Dashboard", dashboard.Name)
		assert.False(t, dashboard.IsDefault)
		assert.True(t, dashboard.IsPublic)
		assert.Equal(t, userID, dashboard.CreatedBy)
	})
}

// TestRepository_WidgetStructure tests widget structure
func TestRepository_WidgetStructure(t *testing.T) {
	t.Run("creates valid widget", func(t *testing.T) {
		dashboardID := uuid.New()
		tenantID := uuid.New()
		userID := uuid.New()
		queryConfig := json.RawMessage(`{"period": "7d"}`)

		widget := &Widget{
			ID:           uuid.New(),
			DashboardID:  dashboardID,
			TenantID:     tenantID,
			Name:         "Active Sessions",
			WidgetType:   WidgetTypeStatCard,
			PositionX:    0,
			PositionY:    0,
			Width:        3,
			Height:       2,
			DataSource:   DataSourceSessions,
			QueryConfig:  queryConfig,
			CreatedBy:    userID,
		}

		assert.Equal(t, dashboardID, widget.DashboardID)
		assert.Equal(t, tenantID, widget.TenantID)
		assert.Equal(t, "Active Sessions", widget.Name)
		assert.Equal(t, WidgetTypeStatCard, widget.WidgetType)
		assert.Equal(t, DataSourceSessions, widget.DataSource)
	})
}

// TestRepository_AlertStructure tests alert structure
func TestRepository_AlertStructure(t *testing.T) {
	t.Run("creates valid alert", func(t *testing.T) {
		tenantID := uuid.New()
		conditions := json.RawMessage(`{"metric": "failed_auth_rate", "operator": "greater_than", "threshold": 10}`)

		alert := &Alert{
			ID:                         uuid.New(),
			TenantID:                   tenantID,
			Name:                       "High Failed Auth Rate",
			Description:                "Alert when failed authentication rate exceeds threshold",
			AlertType:                  AlertTypeThreshold,
			Severity:                   SeverityHigh,
			Conditions:                 conditions,
			EvaluationIntervalMinutes:  15,
			Enabled:                    true,
		}

		assert.Equal(t, tenantID, alert.TenantID)
		assert.Equal(t, "High Failed Auth Rate", alert.Name)
		assert.Equal(t, AlertTypeThreshold, alert.AlertType)
		assert.Equal(t, SeverityHigh, alert.Severity)
		assert.True(t, alert.Enabled)
	})
}

// TestRepository_AlertTriggerStructure tests alert trigger structure
func TestRepository_AlertTriggerStructure(t *testing.T) {
	t.Run("creates valid alert trigger", func(t *testing.T) {
		alertID := uuid.New()
		tenantID := uuid.New()
		data := json.RawMessage(`{"value": 15.5, "threshold": 10}`)

		trigger := &AlertTrigger{
			ID:          uuid.New(),
			AlertID:     alertID,
			TenantID:    tenantID,
			Severity:    SeverityHigh,
			TriggerData: data,
			TriggeredAt: time.Now(),
		}

		assert.Equal(t, alertID, trigger.AlertID)
		assert.Equal(t, tenantID, trigger.TenantID)
		assert.Equal(t, SeverityHigh, trigger.Severity)
	})
}

// TestRepository_MetricStructure tests metric structure
func TestRepository_MetricStructure(t *testing.T) {
	t.Run("creates valid metric", func(t *testing.T) {
		tenantID := uuid.New()

		metric := &Metric{
			ID:         uuid.New(),
			TenantID:   tenantID,
			MetricName: "active_sessions",
			MetricType: MetricTypeGauge,
			Value:      15.5,
			RecordedAt: time.Now(),
		}

		assert.Equal(t, tenantID, metric.TenantID)
		assert.Equal(t, "active_sessions", metric.MetricName)
		assert.Equal(t, MetricTypeGauge, metric.MetricType)
		assert.Equal(t, 15.5, metric.Value)
	})
}

// TestRepository_UserActivityStructure tests user activity structure
func TestRepository_UserActivityStructure(t *testing.T) {
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

// TestRepository_RiskScoreStructure tests risk score structure
func TestRepository_RiskScoreStructure(t *testing.T) {
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
		assert.Equal(t, &accessFreqScore, riskScore.AccessFrequencyScore)
	})
}

// TestRepository_FilterOptionsStructure tests filter options structure
func TestRepository_FilterOptionsStructure(t *testing.T) {
	t.Run("creates valid filter options", func(t *testing.T) {
		tenantID := uuid.New()
		startDate := time.Now().Add(-7 * 24 * time.Hour)
		endDate := time.Now()
		userID := uuid.New()

		filters := &FilterOptions{
			TenantID:  tenantID,
			StartDate: &startDate,
			EndDate:   &endDate,
			UserID:    &userID,
			Limit:     100,
			Offset:    0,
		}

		assert.Equal(t, tenantID, filters.TenantID)
		assert.NotNil(t, filters.StartDate)
		assert.NotNil(t, filters.EndDate)
		assert.Equal(t, &userID, filters.UserID)
		assert.Equal(t, 100, filters.Limit)
	})
}

// TestRepository_DashboardFilterStructure tests dashboard filter structure
func TestRepository_DashboardFilterStructure(t *testing.T) {
	t.Run("creates valid dashboard filter", func(t *testing.T) {
		tenantID := uuid.New()
		userID := uuid.New()
		isDefault := false

		filter := &DashboardFilter{
			TenantID:  tenantID,
			IsDefault: &isDefault,
			IsPublic:  nil,
			CreatedBy: &userID,
			Search:    "security",
			Limit:     10,
		}

		assert.Equal(t, tenantID, filter.TenantID)
		assert.Equal(t, &isDefault, filter.IsDefault)
		assert.Equal(t, &userID, filter.CreatedBy)
		assert.Equal(t, "security", filter.Search)
	})
}

// TestRepository_ReportFilterStructure tests report filter structure
func TestRepository_ReportFilterStructure(t *testing.T) {
	t.Run("creates valid report filter", func(t *testing.T) {
		tenantID := uuid.New()
		reportType := ReportTypeCompliance

		filter := &ReportFilter{
			TenantID:   tenantID,
			ReportType: &reportType,
			Search:     "compliance",
			Limit:      20,
		}

		assert.Equal(t, tenantID, filter.TenantID)
		assert.Equal(t, &reportType, filter.ReportType)
		assert.Equal(t, "compliance", filter.Search)
	})
}

// TestRepository_AlertFilterStructure tests alert filter structure
func TestRepository_AlertFilterStructure(t *testing.T) {
	t.Run("creates valid alert filter", func(t *testing.T) {
		tenantID := uuid.New()
		alertType := AlertTypeThreshold
		enabled := true

		filter := &AlertFilter{
			TenantID:  tenantID,
			AlertType: &alertType,
			Enabled:   &enabled,
			Limit:     50,
		}

		assert.Equal(t, tenantID, filter.TenantID)
		assert.Equal(t, &alertType, filter.AlertType)
		assert.Equal(t, &enabled, filter.Enabled)
	})
}

// TestRepository_SessionStatsStructure tests session stats structure
func TestRepository_SessionStatsStructure(t *testing.T) {
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

// TestRepository_EventStatsStructure tests event stats structure
func TestRepository_EventStatsStructure(t *testing.T) {
	t.Run("creates valid event stats", func(t *testing.T) {
		stats := &EventStats{
			Total:       500,
			Successful:  450,
			Failed:      50,
			Denied:      25,
		}

		assert.Equal(t, int64(500), stats.Total)
		assert.Equal(t, int64(450), stats.Successful)
		assert.Equal(t, int64(50), stats.Failed)
	})
}

// TestRepository_TopUserStructure tests top user structure
func TestRepository_TopUserStructure(t *testing.T) {
	t.Run("creates valid top user", func(t *testing.T) {
		userID := uuid.New()

		topUser := &TopUser{
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

// TestRepository_AccessPatternStructure tests access pattern structure
func TestRepository_AccessPatternStructure(t *testing.T) {
	t.Run("creates valid access pattern", func(t *testing.T) {
		pattern := &AccessPattern{
			HourOfDay:         14,
			DayOfWeek:         2,
			SessionCount:      25,
			UniqueUsers:       10,
			AvgDuration:       1800,
			IsOutsideBusiness: false,
		}

		assert.Equal(t, 14, pattern.HourOfDay)
		assert.Equal(t, 2, pattern.DayOfWeek)
		assert.Equal(t, 25, pattern.SessionCount)
		assert.Equal(t, 10, pattern.UniqueUsers)
		assert.False(t, pattern.IsOutsideBusiness)
	})
}

// TestRepository_TimeSeriesDataPointStructure tests time series data point structure
func TestRepository_TimeSeriesDataPointStructure(t *testing.T) {
	t.Run("creates valid time series data point", func(t *testing.T) {
		point := &TimeSeriesDataPoint{
			Timestamp: time.Now(),
			Value:     15.5,
			Label:     "sessions",
			Metadata: map[string]interface{}{
				"protocol": "ssh",
			},
		}

		assert.False(t, point.Timestamp.IsZero())
		assert.Equal(t, 15.5, point.Value)
		assert.Equal(t, "sessions", point.Label)
		assert.NotEmpty(t, point.Metadata)
	})
}

// TestRepository_ComplianceStatusStructure tests compliance status structure
func TestRepository_ComplianceStatusStructure(t *testing.T) {
	t.Run("creates valid compliance status", func(t *testing.T) {
		violations := []ComplianceViolation{
			{
				PolicyID:      uuid.New(),
				PolicyName:    "Access Control",
				ViolationType: "no_mfa",
				Severity:      "high",
				Description:   "MFA not enabled",
				Count:         5,
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			},
		}
		byPolicy := map[string]CompliancePolicyStatus{
			"session_recording": {
				PolicyID:          uuid.New(),
				PolicyName:        "Session Recording",
				ComplianceRate:    90.0,
				TotalEvaluations:  10,
				PassedEvaluations: 9,
			},
		}

		status := &ComplianceStatus{
			OverallPercentage: 85.5,
			PassedChecks:      85,
			TotalChecks:       100,
			Violations:        violations,
			ByPolicy:          byPolicy,
		}

		assert.Equal(t, 85.5, status.OverallPercentage)
		assert.Equal(t, 100, status.TotalChecks)
		assert.Equal(t, 85, status.PassedChecks)
		assert.NotEmpty(t, status.Violations)
		assert.NotEmpty(t, status.ByPolicy)
	})
}

// TestRepository_ComplianceViolationStructure tests compliance violation structure
func TestRepository_ComplianceViolationStructure(t *testing.T) {
	t.Run("creates valid compliance violation", func(t *testing.T) {
		policyID := uuid.New()

		violation := &ComplianceViolation{
			PolicyID:      policyID,
			PolicyName:    "Access Control",
			ViolationType: "no_mfa",
			Severity:      "high",
			Description:   "MFA not enabled for privileged access",
			Count:         5,
			FirstSeen:     time.Now(),
			LastSeen:      time.Now(),
		}

		assert.Equal(t, policyID, violation.PolicyID)
		assert.Equal(t, "Access Control", violation.PolicyName)
		assert.Equal(t, "no_mfa", violation.ViolationType)
		assert.Equal(t, 5, violation.Count)
	})
}

// TestRepository_CompliancePolicyStatusStructure tests compliance policy status structure
func TestRepository_CompliancePolicyStatusStructure(t *testing.T) {
	t.Run("creates valid compliance policy status", func(t *testing.T) {
		policyID := uuid.New()

		status := &CompliancePolicyStatus{
			PolicyID:          policyID,
			PolicyName:        "Session Recording",
			ComplianceRate:    90.0,
			TotalEvaluations:  10,
			PassedEvaluations: 9,
		}

		assert.Equal(t, policyID, status.PolicyID)
		assert.Equal(t, "Session Recording", status.PolicyName)
		assert.Equal(t, 90.0, status.ComplianceRate)
		assert.Equal(t, 10, status.TotalEvaluations)
		assert.Equal(t, 9, status.PassedEvaluations)
	})
}

// TestRepository_CreateReportRequestStructure tests create report request structure
func TestRepository_CreateReportRequestStructure(t *testing.T) {
	t.Run("creates valid create report request", func(t *testing.T) {
		config := json.RawMessage(`{"period": "7d"}`)
		delivery := json.RawMessage(`{"email": ["admin@example.com"]}`)
		tags := json.RawMessage(`["weekly", "summary"]`)
		scheduleWeekly := ScheduleWeekly

		req := &CreateReportRequest{
			Name:              "Weekly Report",
			Description:       "Weekly security report",
			ReportType:        ReportTypeSessionSummary,
			Config:            config,
			ScheduleEnabled:   true,
			ScheduleType:      &scheduleWeekly,
			ScheduleDayOfWeek: nil,
			ScheduleHour:      nil,
			DeliveryMethods:   delivery,
			Metadata:          json.RawMessage(`{}`),
			Tags:              tags,
		}

		assert.Equal(t, "Weekly Report", req.Name)
		assert.Equal(t, ReportTypeSessionSummary, req.ReportType)
		assert.True(t, req.ScheduleEnabled)
		assert.Equal(t, &scheduleWeekly, req.ScheduleType)
	})
}

// TestRepository_UpdateReportRequestStructure tests update report request structure
func TestRepository_UpdateReportRequestStructure(t *testing.T) {
	t.Run("creates valid update report request", func(t *testing.T) {
		config := json.RawMessage(`{"period": "30d"}`)
		name := "Monthly Report"
		description := "Monthly security report"
		enabled := false
		tags := json.RawMessage(`["monthly"]`)

		req := &UpdateReportRequest{
			Name:            &name,
			Description:     &description,
			Config:          config,
			ScheduleEnabled: &enabled,
			Tags:            tags,
		}

		assert.Equal(t, &name, req.Name)
		assert.Equal(t, &enabled, req.ScheduleEnabled)
	})
}

// TestRepository_CreateDashboardRequestStructure tests create dashboard request structure
func TestRepository_CreateDashboardRequestStructure(t *testing.T) {
	t.Run("creates valid create dashboard request", func(t *testing.T) {
		layout := json.RawMessage(`{"widgets": []}`)
		tags := json.RawMessage(`["operations"]`)

		req := &CreateDashboardRequest{
			Name:      "Operations Dashboard",
			IsDefault: false,
			IsPublic:  true,
			Layout:    layout,
			Tags:      tags,
		}

		assert.Equal(t, "Operations Dashboard", req.Name)
		assert.False(t, req.IsDefault)
		assert.True(t, req.IsPublic)
	})
}

// TestRepository_UpdateDashboardRequestStructure tests update dashboard request structure
func TestRepository_UpdateDashboardRequestStructure(t *testing.T) {
	t.Run("creates valid update dashboard request", func(t *testing.T) {
		layout := json.RawMessage(`{"widgets": []}`)
		name := "Updated Dashboard"

		req := &UpdateDashboardRequest{
			Name:     &name,
			IsPublic: nil,
			Layout:   layout,
		}

		assert.Equal(t, &name, req.Name)
	})
}

// TestRepository_CreateWidgetRequestStructure tests create widget request structure
func TestRepository_CreateWidgetRequestStructure(t *testing.T) {
	t.Run("creates valid create widget request", func(t *testing.T) {
		queryConfig := json.RawMessage(`{"period": "24h"}`)
		displayConfig := json.RawMessage(`{"color": "blue"}`)
		refresh := 300

		req := &CreateWidgetRequest{
			Name:                   "CPU Usage",
			WidgetType:             WidgetTypeGauge,
			PositionX:              0,
			PositionY:              0,
			Width:                  2,
			Height:                 2,
			DataSource:             "metrics",
			QueryConfig:            queryConfig,
			DisplayConfig:          displayConfig,
			RefreshIntervalSeconds: &refresh,
		}

		assert.Equal(t, "CPU Usage", req.Name)
		assert.Equal(t, WidgetTypeGauge, req.WidgetType)
		assert.Equal(t, &refresh, req.RefreshIntervalSeconds)
	})
}

// TestRepository_UpdateWidgetRequestStructure tests update widget request structure
func TestRepository_UpdateWidgetRequestStructure(t *testing.T) {
	t.Run("creates valid update widget request", func(t *testing.T) {
		queryConfig := json.RawMessage(`{"period": "7d"}`)
		name := "Updated Widget"

		req := &UpdateWidgetRequest{
			Name:        &name,
			PositionX:   nil,
			PositionY:   nil,
			QueryConfig: queryConfig,
		}

		assert.Equal(t, &name, req.Name)
	})
}

// TestRepository_CreateAlertRequestStructure tests create alert request structure
func TestRepository_CreateAlertRequestStructure(t *testing.T) {
	t.Run("creates valid create alert request", func(t *testing.T) {
		conditions := json.RawMessage(`{"threshold": 10}`)
		notification := json.RawMessage(`{"email": ["admin@example.com"]}`)
		tags := json.RawMessage(`["cpu", "alert"]`)

		req := &CreateAlertRequest{
			Name:                      "High CPU Alert",
			Description:               "Alert when CPU exceeds threshold",
			AlertType:                 AlertTypeThreshold,
			Severity:                  SeverityCritical,
			Conditions:                conditions,
			EvaluationIntervalMinutes: 5,
			NotificationMethods:       notification,
			Tags:                      tags,
		}

		assert.Equal(t, "High CPU Alert", req.Name)
		assert.Equal(t, AlertTypeThreshold, req.AlertType)
		assert.Equal(t, SeverityCritical, req.Severity)
		assert.Equal(t, 5, req.EvaluationIntervalMinutes)
	})
}

// TestRepository_UpdateAlertRequestStructure tests update alert request structure
func TestRepository_UpdateAlertRequestStructure(t *testing.T) {
	t.Run("creates valid update alert request", func(t *testing.T) {
		conditions := json.RawMessage(`{"threshold": 20}`)
		name := "Updated Alert"

		req := &UpdateAlertRequest{
			Name:       &name,
			Severity:   nil,
			Conditions: conditions,
			Enabled:    nil,
		}

		assert.Equal(t, &name, req.Name)
	})
}

// TestRepository_GenerateReportRequestStructure tests generate report request structure
func TestRepository_GenerateReportRequestStructure(t *testing.T) {
	t.Run("creates valid generate report request", func(t *testing.T) {
		reportID := uuid.New()
		req := &GenerateReportRequest{
			ReportID:   reportID,
			PeriodStart: time.Now().Add(-7 * 24 * time.Hour),
			PeriodEnd:   time.Now(),
			Format:      ReportFormatPDF,
		}

		assert.Equal(t, ReportFormatPDF, req.Format)
		assert.False(t, req.PeriodStart.IsZero())
		assert.False(t, req.PeriodEnd.IsZero())
	})
}

// TestRepository_WidgetDataResponseStructure tests widget data response structure
func TestRepository_WidgetDataResponseStructure(t *testing.T) {
	t.Run("creates valid widget data response", func(t *testing.T) {
		data := map[string]interface{}{"value": 42}
		metadata := map[string]interface{}{"refreshed_at": "2024-01-01T00:00:00Z"}

		resp := &WidgetDataResponse{
			WidgetID:    uuid.New(),
			DataSource:  DataSourceSessions,
			Data:        data,
			GeneratedAt: time.Now(),
			Metadata:    metadata,
		}

		assert.NotEqual(t, uuid.Nil, resp.WidgetID)
		assert.Equal(t, DataSourceSessions, resp.DataSource)
		assert.False(t, resp.GeneratedAt.IsZero())
	})
}
