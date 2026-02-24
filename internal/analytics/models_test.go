package analytics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionAnalytics_Struct(t *testing.T) {
	now := time.Now()
	avgDuration := 120.5
	minDuration := 30
	maxDuration := 300
	peakTime := now
	metadata := []byte(`{"key":"value"}`)

	analytics := SessionAnalytics{
		ID:                       uuid.New(),
		TenantID:                 uuid.New(),
		Date:                     now.Truncate(24 * time.Hour),
		Hour:                     14,
		TotalSessions:            10,
		ActiveSessions:           5,
		CompletedSessions:        4,
		TerminatedSessions:       1,
		FailedSessions:           0,
		SSHSessions:              6,
		RDPSessions:              2,
		DatabaseSessions:         1,
		KubernetesSessions:       1,
		WebSessions:              0,
		AvgDurationSeconds:       &avgDuration,
		MinDurationSeconds:       &minDuration,
		MaxDurationSeconds:       &maxDuration,
		TotalDurationSeconds:     1200,
		PeakConcurrentSessions:   8,
		PeakConcurrentTime:       &peakTime,
		UniqueUsers:              5,
		UniqueTargets:            7,
		TotalRecordings:          10,
		RecordingSizeBytes:       1024000,
		RecordingDurationSeconds: 6000,
		Metadata:                 metadata,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	assert.NotEqual(t, uuid.Nil, analytics.ID)
	assert.Equal(t, 14, analytics.Hour)
	assert.Equal(t, 10, analytics.TotalSessions)
	assert.Equal(t, 5, analytics.ActiveSessions)
	assert.NotNil(t, analytics.AvgDurationSeconds)
	assert.Equal(t, 120.5, *analytics.AvgDurationSeconds)
	assert.Equal(t, 6, analytics.SSHSessions)
	assert.Equal(t, 2, analytics.RDPSessions)
	assert.NotNil(t, analytics.PeakConcurrentTime)
	assert.NotEmpty(t, analytics.Metadata)
}

func TestUserActivity_Struct(t *testing.T) {
	now := time.Now()
	firstAccess := now.Add(-2 * time.Hour)
	lastAccess := now
	peakHour := 14
	countryCode := "US"
	city := "San Francisco"

	activity := UserActivity{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		UserID:            uuid.New(),
		Date:              now.Truncate(24 * time.Hour),
		Hour:              14,
		SessionsInitiated: 3,
		SessionsCompleted: 2,
		CommandsExecuted:  45,
		TargetsAccessed:   5,
		TotalSessionSeconds: 7200,
		ActiveSeconds:     6000,
		IdleSeconds:       1200,
		FirstAccessTime:   &firstAccess,
		LastAccessTime:    &lastAccess,
		PeakHour:          &peakHour,
		CountryCode:       &countryCode,
		City:              &city,
		OffHoursAccess:    false,
		UnusualAccess:     false,
		RiskScore:         25.5,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	assert.NotEqual(t, uuid.Nil, activity.ID)
	assert.Equal(t, 3, activity.SessionsInitiated)
	assert.Equal(t, 45, activity.CommandsExecuted)
	assert.NotNil(t, activity.FirstAccessTime)
	assert.NotNil(t, activity.CountryCode)
	assert.Equal(t, "US", *activity.CountryCode)
	assert.Equal(t, "San Francisco", *activity.City)
	assert.False(t, activity.OffHoursAccess)
	assert.Equal(t, 25.5, activity.RiskScore)
}

func TestCommandFrequency_Struct(t *testing.T) {
	now := time.Now()
	exitCode := 0
	execDuration := 150

	cmd := CommandFrequency{
		ID:              12345,
		TenantID:        uuid.New(),
		Date:            now.Truncate(24 * time.Hour),
		Hour:            14,
		CommandHash:     "abc123def456",
		CommandPattern:  "ls <ARG>",
		BaseCommand:     "ls",
		SessionID:       uuid.New(),
		UserID:          uuid.New(),
		TargetHost:      "server1.example.com",
		RiskLevel:       "low",
		IsDangerous:     false,
		IsBlocked:       false,
		ExecutedAt:      now,
		ExitCode:        &exitCode,
		ExecutionDurationMs: &execDuration,
	}

	assert.Equal(t, int64(12345), cmd.ID)
	assert.Equal(t, "ls", cmd.BaseCommand)
	assert.Equal(t, "server1.example.com", cmd.TargetHost)
	assert.Equal(t, "low", cmd.RiskLevel)
	assert.False(t, cmd.IsDangerous)
	assert.NotNil(t, cmd.ExitCode)
	assert.Equal(t, 0, *cmd.ExitCode)
}

func TestComplianceReport_Struct(t *testing.T) {
	now := time.Now()
	periodStart := now.AddDate(0, 0, -30)
	periodEnd := now
	overallScore := 87.5
	summary := "Compliance evaluation completed"
	findings := []byte(`[{"severity":"high"}]`)
	recommendations := []byte(`[{"action":"review"}`)

	report := ComplianceReport{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		ReportName:      "SOC2 Compliance Report",
		Framework:       "SOC2",
		Version:         "2022",
		GeneratedAt:     now,
		GeneratedBy:     uuid.New(),
		Status:          "passed",
		OverallScore:    &overallScore,
		TotalControls:   50,
		PassedControls:  45,
		FailedControls:  3,
		SkippedControls: 2,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Summary:         &summary,
		Findings:        findings,
		Recommendations: recommendations,
		CreatedAt:       now,
	}

	assert.NotEqual(t, uuid.Nil, report.ID)
	assert.Equal(t, "SOC2", report.Framework)
	assert.Equal(t, "2022", report.Version)
	assert.Equal(t, "passed", report.Status)
	assert.NotNil(t, report.OverallScore)
	assert.Equal(t, 87.5, *report.OverallScore)
	assert.Equal(t, 50, report.TotalControls)
	assert.Equal(t, 45, report.PassedControls)
	assert.NotNil(t, report.Summary)
	assert.NotEmpty(t, report.Findings)
}

func TestComplianceControlEvaluation_Struct(t *testing.T) {
	now := time.Now()
	score := 95.0
	category := "Access Control"

	evaluation := ComplianceControlEvaluation{
		ID:              uuid.New(),
		ReportID:        uuid.New(),
		TenantID:        uuid.New(),
		ControlID:       "CC1.1",
		ControlName:     "Access Control Policy",
		ControlCategory: &category,
		Status:          "passed",
		Score:           &score,
		EvidenceCount:   5,
		EvidenceURLs:    []string{"http://example.com/evidence1"},
		Findings:        stringPtr("Control implemented correctly"),
		RemediationSteps: []string{"No remediation needed"},
		EvaluatedAt:     now,
	}

	assert.NotEqual(t, uuid.Nil, evaluation.ID)
	assert.Equal(t, "CC1.1", evaluation.ControlID)
	assert.NotNil(t, evaluation.ControlCategory)
	assert.Equal(t, "Access Control", *evaluation.ControlCategory)
	assert.NotNil(t, evaluation.Score)
	assert.Equal(t, 95.0, *evaluation.Score)
	assert.NotEmpty(t, evaluation.EvidenceURLs)
	assert.Len(t, evaluation.RemediationSteps, 1)
}

func TestComplianceException_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)
	approvedBy := uuid.New()
	approvedAt := now
	riskAcceptedBy := uuid.New()
	reviewDate := now.AddDate(0, 6, 0)

	exception := ComplianceException{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		ControlID:            "CC3.1",
		ControlName:          "Authentication",
		Framework:            "SOC2",
		Status:               "approved",
		RiskLevel:            "medium",
		RequestedBy:          uuid.New(),
		RequestedAt:          now,
		ApprovedBy:           &approvedBy,
		ApprovedAt:           &approvedAt,
		ExpiresAt:            &expiresAt,
		Justification:        "Technical limitation prevents implementation",
		BusinessReason:       stringPtr("Legacy system compatibility"),
		CompensatingControls: []string{"Manual review", "Enhanced monitoring"},
		RiskAcceptedBy:       &riskAcceptedBy,
		RiskAcceptedAt:       &approvedAt,
		ReviewDate:           &reviewDate,
		ReviewNotes:          stringPtr("Review in 6 months"),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.NotEqual(t, uuid.Nil, exception.ID)
	assert.Equal(t, "CC3.1", exception.ControlID)
	assert.Equal(t, "approved", exception.Status)
	assert.Equal(t, "medium", exception.RiskLevel)
	assert.NotNil(t, exception.ApprovedBy)
	assert.NotNil(t, exception.ExpiresAt)
	assert.NotEmpty(t, exception.Justification)
	assert.NotEmpty(t, exception.CompensatingControls)
}

// TestAnomalyDetection_Struct is in anomaly_test.go to avoid duplication

func TestCommandBlacklist_Struct(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	baseCommand := "rm"
	appliesToUsers := []uuid.UUID{uuid.New()}
	appliesToGroups := []uuid.UUID{uuid.New()}
	appliesToTargets := []string{"/var/*", "/etc/*"}
	overrideRoles := []string{"admin"}
	riskCategory := "data_destruction"

	blacklist := CommandBlacklist{
		ID:               uuid.New(),
		TenantID:         &tenantID,
		CommandPattern:   "rm -rf *",
		PatternType:      "glob",
		BaseCommand:      &baseCommand,
		Action:           "block",
		Severity:         "critical",
		AppliesToUsers:   appliesToUsers,
		AppliesToGroups:  appliesToGroups,
		AppliesToTargets: appliesToTargets,
		AllowOverride:    true,
		OverrideRoles:    overrideRoles,
		Reason:           "Dangerous command that can delete system files",
		RiskCategory:     &riskCategory,
		Enabled:          true,
		CreatedBy:        uuid.New(),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.NotEqual(t, uuid.Nil, blacklist.ID)
	assert.NotNil(t, blacklist.TenantID)
	assert.Equal(t, "rm -rf *", blacklist.CommandPattern)
	assert.Equal(t, "glob", blacklist.PatternType)
	assert.Equal(t, "block", blacklist.Action)
	assert.Equal(t, "critical", blacklist.Severity)
	assert.NotEmpty(t, blacklist.AppliesToUsers)
	assert.NotEmpty(t, blacklist.AppliesToTargets)
	assert.True(t, blacklist.AllowOverride)
	assert.True(t, blacklist.Enabled)
}

func TestSSHKeyAnalytics_Struct(t *testing.T) {
	now := time.Now()
	firstUse := now.Add(-30 * 24 * time.Hour)
	lastUse := now
	avgDuration := 300.0

	analytics := SSHKeyAnalytics{
		ID:                        uuid.New(),
		TenantID:                  uuid.New(),
		SSHKeyID:                  uuid.New(),
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

	assert.NotEqual(t, uuid.Nil, analytics.ID)
	assert.Equal(t, 25, analytics.UsageCount)
	assert.Equal(t, 3, analytics.UniqueUsers)
	assert.NotNil(t, analytics.FirstUseTime)
	assert.NotNil(t, analytics.AvgSessionDurationSeconds)
	assert.Equal(t, 300.0, *analytics.AvgSessionDurationSeconds)
	assert.Equal(t, 5, analytics.OffHoursUsage)
	assert.Equal(t, 1, analytics.FailedAttempts)
}

func TestEnums_SessionStatus(t *testing.T) {
	tests := []struct {
		name   string
		status SessionStatus
		value  string
	}{
		{"active", SessionStatusActive, "active"},
		{"ended", SessionStatusEnded, "ended"},
		{"terminated", SessionStatusTerminated, "terminated"},
		{"failed", SessionStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.status))
		})
	}
}

func TestEnums_ComplianceFramework(t *testing.T) {
	tests := []struct {
		name      string
		framework ComplianceFramework
		value     string
	}{
		{"SOC2", FrameworkSOC2, "SOC2"},
		{"ISO27001", FrameworkISO27001, "ISO27001"},
		{"PCI-DSS", FrameworkPCIDSS, "PCI-DSS"},
		{"HIPAA", FrameworkHIPAA, "HIPAA"},
		{"NIST", FrameworkNIST, "NIST-800-53"},
		{"GDPR", FrameworkGDPR, "GDPR"},
		{"Custom", FrameworkCustom, "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.framework))
		})
	}
}

func TestEnums_ComplianceStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ComplianceStatus
		value  string
	}{
		{"pending", ComplianceStatusPending, "pending"},
		{"passed", ComplianceStatusPassed, "passed"},
		{"failed", ComplianceStatusFailed, "failed"},
		{"partial", ComplianceStatusPartial, "partial"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.status))
		})
	}
}

func TestEnums_AnomalyType(t *testing.T) {
	tests := []struct {
		name   string
		atype  AnomalyType
		value  string
	}{
		{"behavioral", AnomalyTypeBehavioral, "behavioral"},
		{"temporal", AnomalyTypeTemporal, "temporal"},
		{"spatial", AnomalyTypeSpatial, "spatial"},
		{"pattern", AnomalyTypePattern, "pattern"},
		{"volumetric", AnomalyTypeVolumetric, "volumetric"},
		{"ransomware", AnomalyTypeRansomware, "ransomware"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.atype))
		})
	}
}

func TestEnums_RiskLevel(t *testing.T) {
	tests := []struct {
		name  string
		level RiskLevel
		value string
	}{
		{"low", RiskLevelLow, "low"},
		{"medium", RiskLevelMedium, "medium"},
		{"high", RiskLevelHigh, "high"},
		{"critical", RiskLevelCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.level))
		})
	}
}

func TestEnums_AnomalyStatus(t *testing.T) {
	tests := []struct {
		name   string
		status AnomalyStatus
		value  string
	}{
		{"open", AnomalyStatusOpen, "open"},
		{"investigating", AnomalyStatusInvestigating, "investigating"},
		{"resolved", AnomalyStatusResolved, "resolved"},
		{"false_positive", AnomalyStatusFalsePositive, "false_positive"},
		{"ignored", AnomalyStatusIgnored, "ignored"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.status))
		})
	}
}

func TestEnums_CommandPatternType(t *testing.T) {
	tests := []struct {
		name string
		ptype CommandPatternType
		value string
	}{
		{"exact", PatternTypeExact, "exact"},
		{"regex", PatternTypeRegex, "regex"},
		{"glob", PatternTypeGlob, "glob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.ptype))
		})
	}
}

func TestEnums_CommandAction(t *testing.T) {
	tests := []struct {
		name   string
		action CommandAction
		value  string
	}{
		{"block", CommandActionBlock, "block"},
		{"warn", CommandActionWarn, "warn"},
		{"allow", CommandActionAllow, "allow"},
		{"audit", CommandActionAudit, "audit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.action))
		})
	}
}

func TestFilter_Types(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	userID := uuid.New()
	dateFrom := now.Add(-7 * 24 * time.Hour)
	dateTo := now
	hour := 14
	minRiskScore := 50.0
	offHoursOnly := true
	unusualOnly := true
	sessionType := "ssh"
	framework := "SOC2"
	status := "passed"
	severity := "high"
	baseCommand := "rm"
	isDangerous := true

	t.Run("SessionAnalyticsFilter", func(t *testing.T) {
		filter := SessionAnalyticsFilter{
			TenantID:    &tenantID,
			DateFrom:    &dateFrom,
			DateTo:      &dateTo,
			Hour:        &hour,
			SessionType: &sessionType,
			Granularity: "day",
		}

		assert.NotNil(t, filter.TenantID)
		assert.NotNil(t, filter.DateFrom)
		assert.NotNil(t, filter.Hour)
		assert.Equal(t, "ssh", *filter.SessionType)
		assert.Equal(t, "day", filter.Granularity)
	})

	t.Run("UserActivityFilter", func(t *testing.T) {
		filter := UserActivityFilter{
			TenantID:     &tenantID,
			UserID:       &userID,
			DateFrom:     &dateFrom,
			DateTo:       &dateTo,
			MinRiskScore: &minRiskScore,
			OffHoursOnly: &offHoursOnly,
			UnusualOnly:  &unusualOnly,
		}

		assert.NotNil(t, filter.UserID)
		assert.NotNil(t, filter.MinRiskScore)
		assert.Equal(t, 50.0, *filter.MinRiskScore)
		assert.True(t, *filter.OffHoursOnly)
		assert.True(t, *filter.UnusualOnly)
	})

	t.Run("CommandFrequencyFilter", func(t *testing.T) {
		filter := CommandFrequencyFilter{
			TenantID:    &tenantID,
			UserID:      &userID,
			DateFrom:    &dateFrom,
			DateTo:      &dateTo,
			BaseCommand: &baseCommand,
			RiskLevel:   &severity,
			IsDangerous: &isDangerous,
		}

		assert.Equal(t, "rm", *filter.BaseCommand)
		assert.Equal(t, "high", *filter.RiskLevel)
		assert.True(t, *filter.IsDangerous)
	})

	t.Run("ComplianceFilter", func(t *testing.T) {
		filter := ComplianceFilter{
			TenantID:  &tenantID,
			Framework: &framework,
			Status:    &status,
			DateFrom:  &dateFrom,
			DateTo:    &dateTo,
		}

		assert.Equal(t, "SOC2", *filter.Framework)
		assert.Equal(t, "passed", *filter.Status)
	})

	t.Run("AnomalyFilter", func(t *testing.T) {
		atype := "behavioral"
		filter := AnomalyFilter{
			TenantID:    &tenantID,
			UserID:      &userID,
			AnomalyType: &atype,
			Severity:    &severity,
			Status:      &status,
			DateFrom:    &dateFrom,
			DateTo:      &dateTo,
		}

		assert.Equal(t, "behavioral", *filter.AnomalyType)
		assert.Equal(t, "high", *filter.Severity)
	})
}

func TestDashboardMetrics_Struct(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	userID := uuid.New()

	metrics := DashboardMetrics{
		SessionMetrics: &SessionSummary{
			TotalSessions:  100,
			ActiveSessions: 15,
			AvgDuration:    300.5,
			PeakConcurrent: 25,
			SessionsByType: map[string]int{
				"ssh":      60,
				"rdp":      25,
				"database": 15,
			},
			Trend: []DataPoint{
				{Timestamp: now.Add(-24 * time.Hour), Value: 80},
				{Timestamp: now, Value: 100},
			},
		},
		UserActivity: &UserActivitySummary{
			ActiveUsers:    20,
			TotalCommands:  5000,
			HighRiskUsers:  3,
			OffHoursAccess: 5,
		},
		CommandMetrics: &CommandSummary{
			TotalCommands:     5000,
			HighRiskCommands:  50,
			BlockedCommands:   5,
			TopCommands: []CommandRank{
				{Command: "ls", Count: 1500, RiskLevel: "low"},
				{Command: "cd", Count: 1200, RiskLevel: "low"},
				{Command: "rm", Count: 50, RiskLevel: "high"},
			},
		},
		ComplianceStatus: &ComplianceSummary{
			OverallScore:   87.5,
			PassedControls: 45,
			FailedControls: 5,
			OpenExceptions: 2,
			Frameworks: map[string]FrameworkStatus{
				"SOC 2": {
					Score:         90.0,
					Status:        "passed",
					LastEvaluated: now,
				},
			},
		},
		RecentAnomalies: []AnomalyDetection{
			{
				ID:              uuid.New(),
				TenantID:        tenantID,
				AnomalyType:     "behavioral",
				UserID:          &userID,
				Severity:        "high",
				ConfidenceScore: 85.0,
				Title:           "Test Anomaly",
			},
		},
		Timestamp: now,
	}

	assert.NotNil(t, metrics.SessionMetrics)
	assert.Equal(t, 100, metrics.SessionMetrics.TotalSessions)
	assert.NotNil(t, metrics.UserActivity)
	assert.Equal(t, 20, metrics.UserActivity.ActiveUsers)
	assert.NotNil(t, metrics.CommandMetrics)
	assert.Equal(t, 5000, metrics.CommandMetrics.TotalCommands)
	assert.NotNil(t, metrics.ComplianceStatus)
	assert.Equal(t, 87.5, metrics.ComplianceStatus.OverallScore)
	assert.NotEmpty(t, metrics.RecentAnomalies)
	assert.NotEmpty(t, metrics.SessionMetrics.SessionsByType)
	assert.NotEmpty(t, metrics.CommandMetrics.TopCommands)
}

func TestSessionAnalytics_Metadata(t *testing.T) {
	t.Run("GetMetadataMap with valid JSON", func(t *testing.T) {
		metadata := []byte(`{"key":"value","nested":{"field":123}}`)
		analytics := &SessionAnalytics{Metadata: metadata}

		result, err := analytics.GetMetadataMap()
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "value", result["key"])
		assert.NotNil(t, result["nested"])
	})

	t.Run("GetMetadataMap with empty metadata", func(t *testing.T) {
		analytics := &SessionAnalytics{Metadata: nil}

		result, err := analytics.GetMetadataMap()
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("GetMetadataMap with invalid JSON", func(t *testing.T) {
		metadata := []byte(`{invalid json}`)
		analytics := &SessionAnalytics{Metadata: metadata}

		result, err := analytics.GetMetadataMap()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("SetMetadataMap with valid data", func(t *testing.T) {
		analytics := &SessionAnalytics{}
		metadata := map[string]interface{}{
			"key":   "value",
			"count": 123,
		}

		err := analytics.SetMetadataMap(metadata)
		require.NoError(t, err)
		assert.NotNil(t, analytics.Metadata)

		var decoded map[string]interface{}
		err = json.Unmarshal(analytics.Metadata, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "value", decoded["key"])
		assert.Equal(t, float64(123), decoded["count"])
	})

	t.Run("SetMetadataMap with nil", func(t *testing.T) {
		analytics := &SessionAnalytics{Metadata: []byte(`{"existing":"data"}`)}

		err := analytics.SetMetadataMap(nil)
		require.NoError(t, err)
		assert.Nil(t, analytics.Metadata)
	})
}

func TestDataPoint_Struct(t *testing.T) {
	now := time.Now()
	dp := DataPoint{
		Timestamp: now,
		Value:     123.45,
		Label:     "test-point",
	}

	assert.Equal(t, now, dp.Timestamp)
	assert.Equal(t, 123.45, dp.Value)
	assert.Equal(t, "test-point", dp.Label)
}

func TestCommandRank_Struct(t *testing.T) {
	rank := CommandRank{
		Command:   "ls",
		Count:     1500,
		RiskLevel: "low",
	}

	assert.Equal(t, "ls", rank.Command)
	assert.Equal(t, 1500, rank.Count)
	assert.Equal(t, "low", rank.RiskLevel)
}

func TestFrameworkStatus_Struct(t *testing.T) {
	now := time.Now()
	status := FrameworkStatus{
		Score:         95.5,
		Status:        "passed",
		LastEvaluated: now,
	}

	assert.Equal(t, 95.5, status.Score)
	assert.Equal(t, "passed", status.Status)
	assert.Equal(t, now, status.LastEvaluated)
}
