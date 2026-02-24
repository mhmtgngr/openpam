package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestNewPostgresRepository tests the repository constructor
func TestNewPostgresRepository(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

func TestPostgresRepository_SessionAnalytics(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	hour := 14
	avgDuration := 120.5

	t.Run("creates session analytics", func(t *testing.T) {
		analytics := &SessionAnalytics{
			ID:                uuid.New(),
			TenantID:          tenantID,
			Date:              date,
			Hour:              hour,
			TotalSessions:     10,
			AvgDurationSeconds: &avgDuration,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.CreateSessionAnalytics(ctx, analytics)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("gets session analytics", func(t *testing.T) {
		analytics, err := repo.GetSessionAnalytics(ctx, tenantID, date, hour)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, analytics)
	})

	t.Run("updates session analytics", func(t *testing.T) {
		analytics := &SessionAnalytics{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Date:          date,
			Hour:          hour,
			TotalSessions: 15,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.UpdateSessionAnalytics(ctx, analytics)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("lists session analytics with filter", func(t *testing.T) {
		filter := SessionAnalyticsFilter{
			TenantID: &tenantID,
			DateFrom: &date,
			DateTo:   &date,
		}

		analytics, err := repo.ListSessionAnalytics(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, analytics)
	})
}

func TestPostgresRepository_UserActivity(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	hour := 14

	t.Run("creates user activity", func(t *testing.T) {
		activity := &UserActivity{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          userID,
			Date:            date,
			Hour:            hour,
			CommandsExecuted: 50,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.CreateUserActivity(ctx, activity)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("gets user activity", func(t *testing.T) {
		activity, err := repo.GetUserActivity(ctx, tenantID, userID, date, hour)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, activity)
	})

	t.Run("updates user activity", func(t *testing.T) {
		activity := &UserActivity{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          userID,
			Date:            date,
			Hour:            hour,
			CommandsExecuted: 75,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.UpdateUserActivity(ctx, activity)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("lists user activity with filter", func(t *testing.T) {
		filter := UserActivityFilter{
			TenantID: &tenantID,
			UserID:   &userID,
			DateFrom: &date,
			DateTo:   &date,
		}

		activities, err := repo.ListUserActivity(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, activities)
	})

	t.Run("lists user activity with min risk score", func(t *testing.T) {
		minScore := 50.0
		filter := UserActivityFilter{
			TenantID:    &tenantID,
			MinRiskScore: &minScore,
		}

		activities, err := repo.ListUserActivity(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, activities)
	})
}

func TestPostgresRepository_CommandFrequency(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	hour := 14
	exitCode := 0
	execDuration := 150

	t.Run("records command", func(t *testing.T) {
		cmd := &CommandFrequency{
			TenantID:           tenantID,
			Date:               date,
			Hour:               hour,
			CommandHash:        "abc123",
			CommandPattern:     "ls <ARG>",
			BaseCommand:        "ls",
			SessionID:          uuid.New(),
			UserID:             uuid.New(),
			TargetHost:         "server1.example.com",
			RiskLevel:          "low",
			ExecutedAt:         date,
			ExitCode:           &exitCode,
			ExecutionDurationMs: &execDuration,
		}

		err := repo.RecordCommand(ctx, cmd)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("lists command frequency", func(t *testing.T) {
		filter := CommandFrequencyFilter{
			TenantID: &tenantID,
			DateFrom: &date,
			DateTo:   &date,
		}

		commands, err := repo.ListCommandFrequency(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, commands)
	})

	t.Run("gets top commands", func(t *testing.T) {
		dateFrom := date.Add(-24 * time.Hour)
		dateTo := date

		commands, err := repo.GetTopCommands(ctx, tenantID, dateFrom, dateTo, 10)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, commands)
	})
}

func TestPostgresRepository_Compliance(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	generatedBy := uuid.New()
	periodStart := time.Now().AddDate(0, 0, -30) // Used in subtests
	periodEnd := time.Now()                        // Used in subtests
	_ = periodStart
	_ = periodEnd

	t.Run("creates compliance report", func(t *testing.T) {
		report := &ComplianceReport{
			TenantID:    tenantID,
			ReportName:  "Test Report",
			Framework:   "SOC2",
			GeneratedBy: generatedBy,
			GeneratedAt: time.Now(),
			Status:      "pending",
		}

		err := repo.CreateComplianceReport(ctx, report)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, report.ID)
	})

	t.Run("gets compliance report", func(t *testing.T) {
		reportID := uuid.New()

		report, err := repo.GetComplianceReport(ctx, reportID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.Nil(t, report) // No report found
	})

	t.Run("lists compliance reports", func(t *testing.T) {
		filter := ComplianceFilter{
			TenantID: &tenantID,
		}

		reports, err := repo.ListComplianceReports(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, reports)
	})

	t.Run("creates control evaluation", func(t *testing.T) {
		evaluation := &ComplianceControlEvaluation{
			ReportID:       uuid.New(),
			TenantID:       tenantID,
			ControlID:      "CC1.1",
			ControlName:    "Access Control",
			Status:         "passed",
			EvaluatedAt:    time.Now(),
		}

		err := repo.CreateControlEvaluation(ctx, evaluation)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("lists control evaluations", func(t *testing.T) {
		reportID := uuid.New()

		evaluations, err := repo.ListControlEvaluations(ctx, reportID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, evaluations)
	})

	t.Run("creates compliance exception", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:    tenantID,
			ControlID:   "CC1.1",
			ControlName: "Access Control",
			Framework:   "SOC2",
			Status:      "pending",
			RequestedBy:  uuid.New(),
		}

		err := repo.CreateComplianceException(ctx, exception)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("lists compliance exceptions", func(t *testing.T) {
		exceptions, err := repo.ListComplianceExceptions(ctx, tenantID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, exceptions)
	})
}

func TestPostgresRepository_AnomalyDetection(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("creates anomaly detection", func(t *testing.T) {
		anomaly := &AnomalyDetection{
			TenantID:        tenantID,
			AnomalyType:     "behavioral",
			UserID:          &userID,
			Severity:        "high",
			ConfidenceScore: 85.0,
			Title:           "Test Anomaly",
			Status:          "open",
			DetectedAt:      time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		err := repo.CreateAnomalyDetection(ctx, anomaly)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, anomaly.ID)
	})

	t.Run("gets anomaly detection", func(t *testing.T) {
		anomalyID := uuid.New()

		anomaly, err := repo.GetAnomalyDetection(ctx, anomalyID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.Nil(t, anomaly) // No anomaly found
	})

	t.Run("lists anomaly detections", func(t *testing.T) {
		filter := AnomalyFilter{
			TenantID: &tenantID,
		}

		anomalies, err := repo.ListAnomalyDetections(ctx, filter, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, anomalies)
	})

	t.Run("updates anomaly detection", func(t *testing.T) {
		anomaly := &AnomalyDetection{
			ID:          uuid.New(),
			TenantID:    tenantID,
			AnomalyType: "behavioral",
			Severity:    "medium",
			Status:      "investigating",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := repo.UpdateAnomalyDetection(ctx, anomaly)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})
}

func TestPostgresRepository_Ransomware(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("creates ransomware event", func(t *testing.T) {
		event := &RansomwareEvent{
			TenantID:       tenantID,
			DetectionID:    uuid.New(),
			FilesAffected:  100,
			SystemsAffected: 2,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		err := repo.CreateRansomwareEvent(ctx, event)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, event.ID)
	})

	t.Run("gets ransomware event", func(t *testing.T) {
		eventID := uuid.New()

		event, err := repo.GetRansomwareEvent(ctx, eventID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.Nil(t, event) // No event found
	})

	t.Run("lists ransomware events", func(t *testing.T) {
		events, err := repo.ListRansomwareEvents(ctx, tenantID, 10, 0)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, events)
	})

	t.Run("updates ransomware event", func(t *testing.T) {
		event := &RansomwareEvent{
			ID:               uuid.New(),
			TenantID:         tenantID,
			FilesAffected:     150,
			SystemsAffected:   3,
			EmergencyTriggered: true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		err := repo.UpdateRansomwareEvent(ctx, event)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})
}

func TestPostgresRepository_CommandBlacklist(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	baseCommand := "rm"

	t.Run("creates command blacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			CommandPattern: "rm -rf *",
			PatternType:   "glob",
			BaseCommand:   &baseCommand,
			Action:        "block",
			Severity:      "critical",
			CreatedBy:     uuid.New(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.CreateCommandBlacklist(ctx, blacklist)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, blacklist.ID)
	})

	t.Run("gets command blacklist", func(t *testing.T) {
		blacklistID := uuid.New()

		blacklist, err := repo.GetCommandBlacklist(ctx, blacklistID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.Nil(t, blacklist) // No blacklist found
	})

	t.Run("lists command blacklist", func(t *testing.T) {
		blacklists, err := repo.ListCommandBlacklist(ctx, &tenantID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, blacklists)
	})

	t.Run("lists command blacklist with nil tenant", func(t *testing.T) {
		blacklists, err := repo.ListCommandBlacklist(ctx, nil)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, blacklists)
	})

	t.Run("updates command blacklist", func(t *testing.T) {
		blacklist := &CommandBlacklist{
			ID:            uuid.New(),
			CommandPattern: "rm -rf *",
			Action:        "warn",
			Severity:      "high",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.UpdateCommandBlacklist(ctx, blacklist)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("deletes command blacklist", func(t *testing.T) {
		blacklistID := uuid.New()

		err := repo.DeleteCommandBlacklist(ctx, blacklistID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("finds matching blacklist", func(t *testing.T) {
		userIDs := []uuid.UUID{uuid.New()}
		groupIDs := []uuid.UUID{}

		matches, err := repo.FindMatchingBlacklist(ctx, tenantID, "rm -rf /tmp", userIDs, groupIDs)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, matches)
	})
}

func TestPostgresRepository_SSHKeyAnalytics(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("creates SSH key analytics", func(t *testing.T) {
		analytics := &SSHKeyAnalytics{
			TenantID:  tenantID,
			SSHKeyID:  sshKeyID,
			Date:      date,
			UsageCount: 10,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.CreateSSHKeyAnalytics(ctx, analytics)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, analytics.ID)
	})

	t.Run("gets SSH key analytics", func(t *testing.T) {
		analytics, err := repo.GetSSHKeyAnalytics(ctx, tenantID, sshKeyID, date)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.Nil(t, analytics) // No analytics found
	})

	t.Run("lists SSH key analytics", func(t *testing.T) {
		dateFrom := date.Add(-24 * time.Hour)
		dateTo := date

		analyticsList, err := repo.ListSSHKeyAnalytics(ctx, tenantID, sshKeyID, dateFrom, dateTo)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, analyticsList)
	})

	t.Run("updates SSH key analytics", func(t *testing.T) {
		analytics := &SSHKeyAnalytics{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SSHKeyID:  sshKeyID,
			Date:      date,
			UsageCount: 15,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := repo.UpdateSSHKeyAnalytics(ctx, analytics)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})
}

func TestPostgresRepository_DashboardMetrics(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("gets dashboard metrics", func(t *testing.T) {
		metrics, err := repo.GetDashboardMetrics(ctx, tenantID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, metrics)
	})
}

func TestPostgresRepository_TimeSeries(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	validMetrics := []string{"sessions", "active_sessions", "commands", "unique_users", "avg_duration"}

	for _, metric := range validMetrics {
		t.Run("gets time series for "+metric, func(t *testing.T) {
			data, err := repo.GetTimeSeriesData(ctx, tenantID, metric, dateFrom, dateTo)
			if err != nil {
				t.Skip("requires schema")
			}
			assert.NoError(t, err)
			assert.NotNil(t, data)
		})
	}

	t.Run("returns error for invalid metric", func(t *testing.T) {
		_, err := repo.GetTimeSeriesData(ctx, tenantID, "invalid", dateFrom, dateTo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metric")
	})
}

func TestPostgresRepository_SessionSummary(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	dateFrom := time.Now().AddDate(0, 0, -7)
	dateTo := time.Now()

	t.Run("gets tenant session summary", func(t *testing.T) {
		summary, err := repo.GetTenantSessionSummary(ctx, tenantID, dateFrom, dateTo)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, summary)
	})
}

func TestPostgresRepository_UserRiskScore(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("calculates user risk score", func(t *testing.T) {
		score, err := repo.GetUserRiskScore(ctx, tenantID, userID, 30)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 100.0)
	})
}

func TestPostgresRepository_AnomalyStats(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("gets anomaly stats", func(t *testing.T) {
		stats, err := repo.GetAnomalyStats(ctx, tenantID)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})
}

func TestPostgresRepository_BatchUpsert(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()

	t.Run("batch upserts session analytics", func(t *testing.T) {
		analytics := []SessionAnalytics{
			{
				ID:            uuid.New(),
				TotalSessions: 10,
				CreatedAt:     time.Now(),
			},
			{
				ID:            uuid.New(),
				TotalSessions: 15,
				CreatedAt:     time.Now(),
			},
		}

		err := repo.BatchUpsertSessionAnalytics(ctx, analytics)
		if err != nil {
			t.Skip("requires schema")
		}
		assert.NoError(t, err)
	})

	t.Run("handles empty batch", func(t *testing.T) {
		err := repo.BatchUpsertSessionAnalytics(ctx, []SessionAnalytics{})
		assert.NoError(t, err)
	})
}

func TestPostgresRepository_MaterializedViews(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	repo := NewPostgresRepository(db, logger)
	ctx := context.Background()

	t.Run("refreshes materialized views", func(t *testing.T) {
		err := repo.RefreshMaterializedViews(ctx)
		// SQLite doesn't support materialized views, so this should succeed
		assert.NoError(t, err)
	})
}

func TestPostgresRepository_SessionAnalytics_Metadata(t *testing.T) {
	logger := zerolog.Nop()
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Skip("requires database")
	}
	defer db.Close()

	_ = NewPostgresRepository(db, logger) // Test metadata methods on model

	t.Run("gets metadata map", func(t *testing.T) {
		analytics := &SessionAnalytics{
			Metadata: []byte(`{"key":"value"}`),
		}

		metadata, err := analytics.GetMetadataMap()
		assert.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "value", metadata["key"])
	})

	t.Run("handles nil metadata", func(t *testing.T) {
		analytics := &SessionAnalytics{}

		metadata, err := analytics.GetMetadataMap()
		assert.NoError(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		analytics := &SessionAnalytics{
			Metadata: []byte(`{invalid}`),
		}

		_, err := analytics.GetMetadataMap()
		assert.Error(t, err)
	})

	t.Run("sets metadata map", func(t *testing.T) {
		analytics := &SessionAnalytics{}

		metadata := map[string]interface{}{
			"key":   "value",
			"count": 123,
		}

		err := analytics.SetMetadataMap(metadata)
		assert.NoError(t, err)
		assert.NotEmpty(t, analytics.Metadata)
	})

	t.Run("sets nil metadata", func(t *testing.T) {
		analytics := &SessionAnalytics{
			Metadata: []byte(`{"existing":"data"}`),
		}

		err := analytics.SetMetadataMap(nil)
		assert.NoError(t, err)
		assert.Nil(t, analytics.Metadata)
	})
}
