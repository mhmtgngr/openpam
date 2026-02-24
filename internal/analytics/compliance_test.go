package analytics

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockComplianceRepository is a mock repository for compliance testing
type mockComplianceRepository struct {
	reports      []ComplianceReport
	evaluations  []ComplianceControlEvaluation
	exceptions   []ComplianceException
	controlErr   error
	evalErr      error
	exceptionErr error
}

func (m *mockComplianceRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	m.reports = append(m.reports, *report)
	return nil
}

func (m *mockComplianceRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	return nil, nil
}

func (m *mockComplianceRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	return m.reports, m.controlErr
}

func (m *mockComplianceRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error {
	m.evaluations = append(m.evaluations, *evaluation)
	return nil
}

func (m *mockComplianceRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	return m.evaluations, m.evalErr
}

func (m *mockComplianceRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	m.exceptions = append(m.exceptions, *exception)
	return nil
}

func (m *mockComplianceRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return m.exceptions, m.exceptionErr
}

// Implement other required Repository methods with no-ops
func (m *mockComplianceRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error { return nil }
func (m *mockComplianceRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error { return nil }
func (m *mockComplianceRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}
func (m *mockComplianceRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return []SessionAnalytics{}, nil
}
func (m *mockComplianceRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *mockComplianceRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *mockComplianceRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	return nil, nil
}
func (m *mockComplianceRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	return []UserActivity{}, nil
}
func (m *mockComplianceRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error { return nil }
func (m *mockComplianceRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	return []CommandFrequency{}, nil
}
func (m *mockComplianceRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return []CommandRank{}, nil
}
func (m *mockComplianceRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error { return nil }
func (m *mockComplianceRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) { return nil, nil }
func (m *mockComplianceRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	return []AnomalyDetection{}, nil
}
func (m *mockComplianceRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error { return nil }
func (m *mockComplianceRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error { return nil }
func (m *mockComplianceRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) { return nil, nil }
func (m *mockComplianceRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	return []RansomwareEvent{}, nil
}
func (m *mockComplianceRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error { return nil }
func (m *mockComplianceRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *mockComplianceRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) { return nil, nil }
func (m *mockComplianceRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockComplianceRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *mockComplianceRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockComplianceRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockComplianceRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *mockComplianceRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *mockComplianceRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	return nil, nil
}
func (m *mockComplianceRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return []SSHKeyAnalytics{}, nil
}
func (m *mockComplianceRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) { return nil, nil }
func (m *mockComplianceRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}

func TestNewComplianceEngine(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.repo)
	assert.NotNil(t, engine.frameworks)
	assert.NotEmpty(t, engine.frameworks)
}

func TestComplianceEngine_Frameworks(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	expectedFrameworks := []ComplianceFramework{
		FrameworkSOC2, FrameworkISO27001, FrameworkPCIDSS,
		FrameworkHIPAA, FrameworkNIST,
	}

	for _, fw := range expectedFrameworks {
		t.Run(string(fw), func(t *testing.T) {
			def, ok := engine.frameworks[fw]
			assert.True(t, ok, "framework %s should exist", fw)
			assert.NotNil(t, def)
			assert.NotEmpty(t, def.Name)
			assert.NotEmpty(t, def.Version)
			assert.NotEmpty(t, def.Controls)
		})
	}
}

func TestComplianceEngine_ControlDefinitions(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	t.Run("SOC2 controls", func(t *testing.T) {
		def, ok := engine.frameworks[FrameworkSOC2]
		require.True(t, ok)
		assert.NotEmpty(t, def.Controls)

		// Check for known controls
		controlIDs := make(map[string]bool)
		for _, ctrl := range def.Controls {
			controlIDs[ctrl.ID] = true
		}

		assert.True(t, controlIDs["CC1.1"], "CC1.1 should exist")
		assert.True(t, controlIDs["CC3.1"], "CC3.1 should exist")
		assert.True(t, controlIDs["CC4.1"], "CC4.1 should exist")
	})

	t.Run("ISO27001 controls", func(t *testing.T) {
		def, ok := engine.frameworks[FrameworkISO27001]
		require.True(t, ok)
		assert.NotEmpty(t, def.Controls)

		controlIDs := make(map[string]bool)
		for _, ctrl := range def.Controls {
			controlIDs[ctrl.ID] = true
		}

		assert.True(t, controlIDs["A.9.1"], "A.9.1 should exist")
		assert.True(t, controlIDs["A.12.4"], "A.12.4 should exist")
	})
}

func TestComplianceEngine_GenerateReport(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	ctx := context.Background()
	tenantID := uuid.New()
	generatedBy := uuid.New()
	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	t.Run("SOC2 report", func(t *testing.T) {
		report, err := engine.GenerateReport(ctx, tenantID, generatedBy, FrameworkSOC2, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, report)

		assert.Equal(t, tenantID, report.TenantID)
		assert.Equal(t, generatedBy, report.GeneratedBy)
		assert.Equal(t, string(FrameworkSOC2), report.Framework)
		assert.Equal(t, "2022", report.Version)
		assert.NotEmpty(t, report.ReportName)
		assert.NotEmpty(t, report.Status)
		assert.Greater(t, report.TotalControls, 0)
		assert.GreaterOrEqual(t, report.PassedControls, 0)
		assert.GreaterOrEqual(t, report.FailedControls, 0)
		assert.NotNil(t, report.Summary)
		assert.NotEmpty(t, report.Findings)
		assert.NotEmpty(t, report.Recommendations)
	})

	t.Run("ISO27001 report", func(t *testing.T) {
		report, err := engine.GenerateReport(ctx, tenantID, generatedBy, FrameworkISO27001, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, report)
		assert.Equal(t, string(FrameworkISO27001), report.Framework)
		assert.Equal(t, "2022", report.Version)
	})

	t.Run("PCI-DSS report", func(t *testing.T) {
		report, err := engine.GenerateReport(ctx, tenantID, generatedBy, FrameworkPCIDSS, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, report)
		assert.Equal(t, string(FrameworkPCIDSS), report.Framework)
		assert.Equal(t, "4.0", report.Version)
	})

	t.Run("Unknown framework", func(t *testing.T) {
		_, err := engine.GenerateReport(ctx, tenantID, generatedBy, ComplianceFramework("UNKNOWN"), periodStart, periodEnd)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown framework")
	})
}

func TestComplianceEngine_EvaluateControl(t *testing.T) {
	repo := &mockComplianceRepository{exceptions: []ComplianceException{}}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)
	ctx := context.Background()
	tenantID := uuid.New()
	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	t.Run("data_query evaluation", func(t *testing.T) {
		control := ControlDefinition{
			ID:               "CC1.1",
			Name:             "Access Control Policy",
			Category:         "Access",
			Description:      "Formal access control policy",
			EvaluationMethod: "data_query",
			Required:         true,
		}

		evaluation, err := engine.evaluateControl(ctx, tenantID, control, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, evaluation)
		assert.Equal(t, "CC1.1", evaluation.ControlID)
		assert.Equal(t, "Access Control Policy", evaluation.ControlName)
		assert.NotNil(t, evaluation.ControlCategory)
		assert.NotEmpty(t, evaluation.Status)
	})

	t.Run("configuration_check evaluation", func(t *testing.T) {
		control := ControlDefinition{
			ID:               "CC5.1",
			Name:             "Encryption",
			Category:         "Data Protection",
			EvaluationMethod: "configuration_check",
			Required:         true,
		}

		evaluation, err := engine.evaluateControl(ctx, tenantID, control, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, evaluation)
		assert.NotEmpty(t, evaluation.Status)
	})

	t.Run("manual_check evaluation", func(t *testing.T) {
		control := ControlDefinition{
			ID:               "CC7.1",
			Name:             "Privilege Review",
			Category:         "Access",
			EvaluationMethod: "manual_check",
			Required:         true,
		}

		evaluation, err := engine.evaluateControl(ctx, tenantID, control, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, evaluation)
		assert.Equal(t, "pending", evaluation.Status)
		assert.Nil(t, evaluation.Score)
	})

	t.Run("unknown evaluation method defaults to passed", func(t *testing.T) {
		control := ControlDefinition{
			ID:               "UNKNOWN.1",
			Name:             "Unknown Control",
			EvaluationMethod: "unknown_method",
			Required:         false,
		}

		evaluation, err := engine.evaluateControl(ctx, tenantID, control, periodStart, periodEnd)

		require.NoError(t, err)
		require.NotNil(t, evaluation)
		assert.Equal(t, "passed", evaluation.Status)
		assert.NotNil(t, evaluation.Score)
		assert.Equal(t, 100.0, *evaluation.Score)
	})
}

func TestComplianceEngine_EvaluateControlWithException(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	periodStart := time.Now().AddDate(0, 0, -30)
	periodEnd := time.Now()

	// Add an approved exception
	exception := ComplianceException{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ControlID:   "CC1.1",
		ControlName: "Access Control Policy",
		Framework:   string(FrameworkSOC2),
		Status:      "approved",
		RiskLevel:   "low",
		RequestedBy: uuid.New(),
		RequestedAt: time.Now(),
	}

	repo := &mockComplianceRepository{exceptions: []ComplianceException{exception}}
	engine := NewComplianceEngine(repo, nil, logger)

	control := ControlDefinition{
		ID:               "CC1.1",
		Name:             "Access Control Policy",
		EvaluationMethod: "data_query",
		Required:         true,
	}

	evaluation, err := engine.evaluateControl(ctx, tenantID, control, periodStart, periodEnd)

	require.NoError(t, err)
	require.NotNil(t, evaluation)
	assert.Equal(t, "passed", evaluation.Status)
	assert.NotNil(t, evaluation.Score)
	assert.Equal(t, 100.0, *evaluation.Score)
	assert.Contains(t, *evaluation.Findings, "Approved exception")
}

func TestComplianceEngine_GenerateSummary(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	report := &ComplianceReport{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ReportName:     "Test Report",
		Framework:      "SOC2",
		Version:        "2022",
		GeneratedAt:    time.Now(),
		GeneratedBy:    uuid.New(),
		Status:         "passed",
		TotalControls:  10,
		PassedControls: 8,
		FailedControls: 1,
		SkippedControls: 1,
		PeriodStart:    time.Now().AddDate(0, 0, -30),
		PeriodEnd:      time.Now(),
	}

	def := &FrameworkDefinition{
		Name:    "SOC 2",
		Version: "2022",
	}

	summary := engine.generateSummary(report, def)

	require.NotNil(t, summary)
	assert.Contains(t, *summary, "SOC 2")
	assert.Contains(t, *summary, "2022")
	assert.Contains(t, *summary, "8/10")
}

func TestComplianceEngine_GenerateFindings(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	t.Run("failed controls generate findings", func(t *testing.T) {
		lowScore := 60.0
		report := &ComplianceReport{
			FailedControls: 3,
			OverallScore:   &lowScore,
		}
		def := &FrameworkDefinition{Name: "SOC 2"}

		findings := engine.generateFindings(report, def)

		assert.NotEmpty(t, findings)
		assert.GreaterOrEqual(t, len(findings), 2) // At least failed controls and low score findings
	})

	t.Run("perfect compliance no findings", func(t *testing.T) {
		highScore := 100.0
		report := &ComplianceReport{
			FailedControls: 0,
			OverallScore:   &highScore,
		}
		def := &FrameworkDefinition{Name: "SOC 2"}

		findings := engine.generateFindings(report, def)

		assert.Empty(t, findings)
	})
}

func TestComplianceEngine_GenerateRecommendations(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	t.Run("failed controls generate recommendations", func(t *testing.T) {
		report := &ComplianceReport{
			FailedControls: 2,
		}
		def := &FrameworkDefinition{Name: "SOC 2"}

		recs := engine.generateRecommendations(report, def)

		assert.NotEmpty(t, recs)
		// Should always have schedule recommendation
		assert.True(t, len(recs) >= 2)
	})

	t.Run("check recommendation structure", func(t *testing.T) {
		score := 90.0
		report := &ComplianceReport{
			FailedControls: 1,
			OverallScore:   &score,
		}
		def := &FrameworkDefinition{Name: "SOC 2"}

		recs := engine.generateRecommendations(report, def)

		for _, rec := range recs {
			assert.NotEmpty(t, rec["priority"])
			assert.NotEmpty(t, rec["action"])
			assert.NotEmpty(t, rec["description"])
			assert.NotEmpty(t, rec["effort"])
		}
	})
}

func TestComplianceEngine_GetControlRequirements(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()

	engine := NewComplianceEngine(repo, nil, logger)

	t.Run("existing control", func(t *testing.T) {
		control, err := engine.GetControlRequirements(FrameworkSOC2, "CC1.1")

		require.NoError(t, err)
		require.NotNil(t, control)
		assert.Equal(t, "CC1.1", control.ID)
		assert.NotEmpty(t, control.Name)
		assert.NotEmpty(t, control.Category)
		assert.NotEmpty(t, control.Description)
	})

	t.Run("unknown framework", func(t *testing.T) {
		_, err := engine.GetControlRequirements(ComplianceFramework("UNKNOWN"), "CC1.1")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown framework")
	})

	t.Run("unknown control", func(t *testing.T) {
		_, err := engine.GetControlRequirements(FrameworkSOC2, "UNKNOWN.999")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestComplianceEngine_ValidateComplianceException(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()
	ctx := context.Background()

	engine := NewComplianceEngine(repo, nil, logger)
	tenantID := uuid.New()

	t.Run("valid exception for non-required control", func(t *testing.T) {
		exception := &ComplianceException{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ControlID:     "CC7.1",
			ControlName:   "Privilege Review",
			Framework:     string(FrameworkSOC2),
			Status:        "pending",
			RiskLevel:     "medium",
			RequestedBy:   uuid.New(),
			Justification: "Technical limitation prevents implementation",
		}

		err := engine.ValidateComplianceException(ctx, exception)
		assert.NoError(t, err)
	})

	t.Run("missing justification", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:  tenantID,
			ControlID: "CC7.1",
			Framework: string(FrameworkSOC2),
			RiskLevel: "low",
		}

		err := engine.ValidateComplianceException(ctx, exception)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "justification required")
	})

	t.Run("invalid risk level", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC7.1",
			Framework:     string(FrameworkSOC2),
			RiskLevel:     "invalid",
			Justification: "Test",
		}

		err := engine.ValidateComplianceException(ctx, exception)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid risk level")
	})

	t.Run("required control cannot have exception", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC1.1",
			Framework:     string(FrameworkSOC2),
			RiskLevel:     "medium",
			Justification: "Test justification",
		}

		err := engine.ValidateComplianceException(ctx, exception)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot have exceptions")
	})
}

func TestComplianceEngine_EvaluateControlForException(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()
	ctx := context.Background()

	engine := NewComplianceEngine(repo, nil, logger)
	tenantID := uuid.New()

	t.Run("required control", func(t *testing.T) {
		canExcept, reason, err := engine.EvaluateControlForException(ctx, tenantID, "CC1.1", FrameworkSOC2)

		assert.False(t, canExcept)
		assert.NotEmpty(t, reason)
		assert.NoError(t, err)
	})

	t.Run("non-required control", func(t *testing.T) {
		canExcept, reason, err := engine.EvaluateControlForException(ctx, tenantID, "CC7.1", FrameworkSOC2)

		assert.True(t, canExcept)
		assert.Empty(t, reason)
		assert.NoError(t, err)
	})

	t.Run("unknown framework", func(t *testing.T) {
		_, _, err := engine.EvaluateControlForException(ctx, tenantID, "CC1.1", ComplianceFramework("UNKNOWN"))

		assert.Error(t, err)
	})

	t.Run("unknown control", func(t *testing.T) {
		_, _, err := engine.EvaluateControlForException(ctx, tenantID, "UNKNOWN.999", FrameworkSOC2)

		assert.Error(t, err)
	})
}

func TestComplianceEngine_RequestComplianceException(t *testing.T) {
	repo := &mockComplianceRepository{}
	logger := zerolog.Nop()
	ctx := context.Background()

	engine := NewComplianceEngine(repo, nil, logger)
	tenantID := uuid.New()

	t.Run("valid request", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:      tenantID,
			ControlID:     "CC7.1",
			ControlName:   "Privilege Review",
			Framework:     string(FrameworkSOC2),
			RiskLevel:     "low",
			Justification: "Technical limitation",
		}

		err := engine.RequestComplianceException(ctx, exception)
		assert.NoError(t, err)
		assert.Equal(t, "pending", exception.Status)
		assert.False(t, exception.RequestedAt.IsZero())
	})

	t.Run("invalid request fails validation", func(t *testing.T) {
		exception := &ComplianceException{
			TenantID:  tenantID,
			ControlID: "CC1.1", // Required control
			Framework: string(FrameworkSOC2),
			RiskLevel: "low",
		}

		err := engine.RequestComplianceException(ctx, exception)
		assert.Error(t, err)
	})
}

func TestComplianceEngine_GetPendingExceptions(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	approved := &ComplianceException{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ControlID: "CC7.1",
		Status:    "approved",
	}

	pending := &ComplianceException{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ControlID: "CC7.2",
		Status:    "pending",
	}

	repo := &mockComplianceRepository{exceptions: []ComplianceException{*approved, *pending}}
	engine := NewComplianceEngine(repo, nil, logger)

	pendingExceptions, err := engine.GetPendingExceptions(ctx, tenantID)

	require.NoError(t, err)
	assert.Len(t, pendingExceptions, 1)
	assert.Equal(t, "pending", pendingExceptions[0].Status)
}

func TestComplianceEngine_CheckExpiringExceptions(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	// Create exceptions with different expiration dates
	expiringSoon := time.Now().Add(5 * 24 * time.Hour)
	notExpiring := time.Now().Add(60 * 24 * time.Hour)
	expired := time.Now().Add(-24 * time.Hour)

	exceptions := []ComplianceException{
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ControlID: "CC7.1",
			Status:    "approved",
			ExpiresAt: &expiringSoon,
		},
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ControlID: "CC7.2",
			Status:    "approved",
			ExpiresAt: &notExpiring,
		},
		{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ControlID: "CC7.3",
			Status:    "approved",
			ExpiresAt: &expired,
		},
	}

	repo := &mockComplianceRepository{exceptions: exceptions}
	engine := NewComplianceEngine(repo, nil, logger)

	// Check for exceptions expiring in 30 days
	expiring, err := engine.CheckExpiringExceptions(ctx, tenantID, 30)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(expiring), 2) // At least expiringSoon and expired
}

func TestComplianceEngine_GetSummary(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	// Add some reports
	score1 := 85.0
	score2 := 92.0

	reports := []ComplianceReport{
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			Framework:      "SOC2",
			OverallScore:   &score1,
			PassedControls: 8,
			FailedControls: 2,
			GeneratedAt:    time.Now(),
			Status:         "passed",
		},
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			Framework:      "ISO27001",
			OverallScore:   &score2,
			PassedControls: 10,
			FailedControls: 0,
			GeneratedAt:    time.Now(),
			Status:         "passed",
		},
	}

	repo := &mockComplianceRepository{reports: reports}
	engine := NewComplianceEngine(repo, nil, logger)

	summary, err := engine.GetSummary(ctx, tenantID)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.NotEmpty(t, summary.Frameworks)
	assert.Greater(t, summary.OverallScore, 0.0)
	assert.Greater(t, summary.PassedControls, 0)
}

func TestComplianceReport_MarshalUnmarshal(t *testing.T) {
	now := time.Now()
	score := 87.5
	summary := "Test summary"
	findings := []byte(`[{"type":"high"}]`)
	recommendations := []byte(`[{"action":"fix"}]`)

	report := ComplianceReport{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		ReportName:      "Test Report",
		Framework:       "SOC2",
		Version:         "2022",
		GeneratedAt:     now,
		GeneratedBy:     uuid.New(),
		Status:          "passed",
		OverallScore:    &score,
		TotalControls:   10,
		PassedControls:  8,
		FailedControls:  1,
		SkippedControls: 1,
		PeriodStart:     now.Add(-30 * 24 * time.Hour),
		PeriodEnd:       now,
		Summary:         &summary,
		Findings:        findings,
		Recommendations: recommendations,
		CreatedAt:       now,
	}

	// Marshal to JSON
	data, err := json.Marshal(report)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal back
	var unmarshaled ComplianceReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, report.ID, unmarshaled.ID)
	assert.Equal(t, report.TenantID, unmarshaled.TenantID)
	assert.Equal(t, report.Framework, unmarshaled.Framework)
	assert.Equal(t, report.Status, unmarshaled.Status)
	assert.NotNil(t, unmarshaled.OverallScore)
	assert.Equal(t, score, *unmarshaled.OverallScore)
}

func TestComplianceControlEvaluation_StatusValues(t *testing.T) {
	validStatuses := []string{"passed", "failed", "skipped", "pending", "error"}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			evaluation := ComplianceControlEvaluation{
				ID:       uuid.New(),
				ReportID: uuid.New(),
				Status:   status,
			}

			assert.Equal(t, status, evaluation.Status)
		})
	}
}

func TestComplianceException_StatusValues(t *testing.T) {
	validStatuses := []string{"pending", "approved", "denied", "expired"}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			exception := ComplianceException{
				ID:     uuid.New(),
				Status: status,
			}

			assert.Equal(t, status, exception.Status)
		})
	}
}
