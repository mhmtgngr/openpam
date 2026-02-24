package analytics

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnomalyRepository is a mock repository for anomaly detection testing
type mockAnomalyRepository struct {
	anomalies    []AnomalyDetection
	ransomware   []RansomwareEvent
	activities   []UserActivity
	commands     []CommandFrequency
	createCalled bool
	createErr    error
}

func (m *mockAnomalyRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	m.createCalled = true
	m.anomalies = append(m.anomalies, *anomaly)
	return m.createErr
}

func (m *mockAnomalyRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	if m.anomalies == nil {
		return nil, nil
	}
	for _, a := range m.anomalies {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *mockAnomalyRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	if m.anomalies == nil {
		return []AnomalyDetection{}, nil
	}
	return m.anomalies, nil
}

func (m *mockAnomalyRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	for i, a := range m.anomalies {
		if a.ID == anomaly.ID {
			m.anomalies[i] = *anomaly
			break
		}
	}
	return nil
}

func (m *mockAnomalyRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	m.ransomware = append(m.ransomware, *event)
	return nil
}

func (m *mockAnomalyRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	for _, r := range m.ransomware {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, nil
}

func (m *mockAnomalyRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	if m.ransomware == nil {
		return []RansomwareEvent{}, nil
	}
	return m.ransomware, nil
}

func (m *mockAnomalyRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	for i, r := range m.ransomware {
		if r.ID == event.ID {
			m.ransomware[i] = *event
			break
		}
	}
	return nil
}

func (m *mockAnomalyRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	if m.activities == nil {
		return []UserActivity{}, nil
	}
	return m.activities, nil
}

func (m *mockAnomalyRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error {
	m.commands = append(m.commands, *cmd)
	return nil
}

// Implement remaining Repository methods as no-ops
func (m *mockAnomalyRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error { return nil }
func (m *mockAnomalyRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error { return nil }
func (m *mockAnomalyRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}
func (m *mockAnomalyRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return []SessionAnalytics{}, nil
}
func (m *mockAnomalyRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *mockAnomalyRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error { return nil }
func (m *mockAnomalyRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	return nil, nil
}
func (m *mockAnomalyRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	if m.commands == nil {
		return []CommandFrequency{}, nil
	}
	return m.commands, nil
}
func (m *mockAnomalyRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return nil, nil
}
func (m *mockAnomalyRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error { return nil }
func (m *mockAnomalyRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) { return nil, nil }
func (m *mockAnomalyRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	return []ComplianceReport{}, nil
}
func (m *mockAnomalyRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error { return nil }
func (m *mockAnomalyRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	return []ComplianceControlEvaluation{}, nil
}
func (m *mockAnomalyRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error { return nil }
func (m *mockAnomalyRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return []ComplianceException{}, nil
}
func (m *mockAnomalyRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *mockAnomalyRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) { return nil, nil }
func (m *mockAnomalyRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockAnomalyRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error { return nil }
func (m *mockAnomalyRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockAnomalyRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	return []CommandBlacklist{}, nil
}
func (m *mockAnomalyRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *mockAnomalyRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error { return nil }
func (m *mockAnomalyRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	return nil, nil
}
func (m *mockAnomalyRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return []SSHKeyAnalytics{}, nil
}
func (m *mockAnomalyRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) { return nil, nil }
func (m *mockAnomalyRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}

func TestNewAnomalyDetector(t *testing.T) {
	repo := &mockAnomalyRepository{}
	logger := zerolog.Nop()

	// Note: Passing nil for cache since tests don't use Redis
	detector := NewAnomalyDetector(repo, nil, 75.0, logger)

	assert.NotNil(t, detector)
	assert.NotNil(t, detector.repo)
	assert.Equal(t, 75.0, detector.threshold)
}

func TestAnomalyDetector_RunDetection(t *testing.T) {
	repo := &mockAnomalyRepository{}
	logger := zerolog.Nop()

	detector := NewAnomalyDetector(repo, nil, 75.0, logger)
	ctx := context.Background()
	tenantID := uuid.New()

	detections, err := detector.RunDetection(ctx, tenantID)

	// Should not error even with empty data
	assert.NoError(t, err)
	// detections returns empty slice when no anomalies found
	// The mock repo is not a PostgresRepository so detection methods return nil/empty
	assert.NotNil(t, detections)
	assert.IsType(t, []AnomalyDetection{}, detections)
}

func TestAnomalyDetector_EvaluateCommand(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("critical risk command creates anomaly", func(t *testing.T) {
		repo := &mockAnomalyRepository{}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		cmd := &ParsedCommand{
			Original:    "rm -rf /",
			BaseCommand: "rm",
			Pattern:     "rm -rf <ARG>",
			RiskLevel:   "critical",
			Normalized:  "rm -rf /",
		}

		err := detector.EvaluateCommand(ctx, tenantID, userID, cmd)

		assert.NoError(t, err)
		assert.True(t, repo.createCalled, "should create anomaly for critical command")
		assert.Len(t, repo.anomalies, 1)
		assert.Equal(t, "critical", repo.anomalies[0].Severity)
		assert.Equal(t, string(AnomalyTypePattern), repo.anomalies[0].AnomalyType)
	})

	t.Run("non-critical command no anomaly", func(t *testing.T) {
		repo := &mockAnomalyRepository{}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		cmd := &ParsedCommand{
			Original:    "ls -la",
			BaseCommand: "ls",
			Pattern:     "ls -la",
			RiskLevel:   "low",
			Normalized:  "ls -la",
		}

		err := detector.EvaluateCommand(ctx, tenantID, userID, cmd)

		assert.NoError(t, err)
		assert.False(t, repo.createCalled)
		assert.Len(t, repo.anomalies, 0)
	})

	t.Run("low risk command", func(t *testing.T) {
		repo := &mockAnomalyRepository{}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		cmd := &ParsedCommand{
			Original:    "pwd",
			BaseCommand: "pwd",
			Pattern:     "pwd",
			RiskLevel:   "low",
			Normalized:  "pwd",
		}

		err := detector.EvaluateCommand(ctx, tenantID, userID, cmd)

		assert.NoError(t, err)
		assert.False(t, repo.createCalled)
		assert.Len(t, repo.anomalies, 0)
	})
}

func TestAnomalyDetector_EvaluateUser(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	t.Run("high risk user creates anomaly", func(t *testing.T) {
		highRisk := 75.0
		activities := []UserActivity{
			{
				ID:            uuid.New(),
				TenantID:      tenantID,
				UserID:        userID,
				Date:          time.Now(),
				RiskScore:     highRisk,
				OffHoursAccess: true,
				UnusualAccess:  true,
			},
		}

		repo := &mockAnomalyRepository{activities: activities}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		detections, err := detector.EvaluateUser(ctx, tenantID, userID)

		assert.NoError(t, err)
		assert.NotEmpty(t, detections)

		// Should detect high risk score
		found := false
		for _, d := range detections {
			if d.Title == "High User Risk Score" {
				found = true
				assert.Equal(t, userID, *d.UserID)
			}
		}
		assert.True(t, found, "should find high risk score anomaly")
	})

	t.Run("excessive off-hours access", func(t *testing.T) {
		activities := []UserActivity{
			{UserID: userID, OffHoursAccess: true},
			{UserID: userID, OffHoursAccess: true},
			{UserID: userID, OffHoursAccess: true},
			{UserID: userID, OffHoursAccess: true},
			{UserID: userID, OffHoursAccess: true},
		}

		repo := &mockAnomalyRepository{activities: activities}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		detections, err := detector.EvaluateUser(ctx, tenantID, userID)

		assert.NoError(t, err)
		assert.NotEmpty(t, detections)

		// Should detect off-hours access
		found := false
		for _, d := range detections {
			if d.Title == "Excessive Off-Hours Access" {
				found = true
				assert.Equal(t, string(AnomalyTypeTemporal), d.AnomalyType)
			}
		}
		assert.True(t, found, "should find off-hours access anomaly")
	})

	t.Run("no activity returns no anomalies", func(t *testing.T) {
		repo := &mockAnomalyRepository{activities: []UserActivity{}}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		detections, err := detector.EvaluateUser(ctx, tenantID, userID)

		assert.NoError(t, err)
		assert.Empty(t, detections)
	})
}

func TestAnomalyDetector_CheckRansomwareIndicators(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	repo := &mockAnomalyRepository{}
	detector := NewAnomalyDetector(repo, nil, 75.0, logger)

	t.Run("no indicators", func(t *testing.T) {
		event, err := detector.CheckRansomwareIndicators(ctx, tenantID, userID)

		assert.NoError(t, err)
		assert.Nil(t, event)
		assert.Len(t, repo.ransomware, 0)
	})
}

func TestAnomalyDetector_TriggerEmergencyWorkflow(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	event := &RansomwareEvent{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		DetectionID:        uuid.New(),
		EmergencyTriggered: false,
		ContainmentStatus:  nil,
	}

	repo := &mockAnomalyRepository{ransomware: []RansomwareEvent{*event}}
	detector := NewAnomalyDetector(repo, nil, 75.0, logger)

	err := detector.TriggerEmergencyWorkflow(ctx, event.ID)

	assert.NoError(t, err)
	assert.True(t, repo.ransomware[0].EmergencyTriggered)
	assert.NotNil(t, repo.ransomware[0].ContainmentStatus)
	assert.Equal(t, "isolating", *repo.ransomware[0].ContainmentStatus)
}

func TestAnomalyDetector_GetAnomalyTrends(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("empty trends", func(t *testing.T) {
		repo := &mockAnomalyRepository{}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		trends, err := detector.GetAnomalyTrends(ctx, tenantID, 30)

		assert.NoError(t, err)
		assert.NotNil(t, trends)
		assert.IsType(t, map[string]int{}, trends)
	})

	t.Run("with anomalies", func(t *testing.T) {
		anomalies := []AnomalyDetection{
			{TenantID: tenantID, AnomalyType: "behavioral"},
			{TenantID: tenantID, AnomalyType: "behavioral"},
			{TenantID: tenantID, AnomalyType: "temporal"},
			{TenantID: tenantID, AnomalyType: "pattern"},
		}

		repo := &mockAnomalyRepository{anomalies: anomalies}
		detector := NewAnomalyDetector(repo, nil, 75.0, logger)

		trends, err := detector.GetAnomalyTrends(ctx, tenantID, 30)

		assert.NoError(t, err)
		assert.Equal(t, 2, trends["behavioral"])
		assert.Equal(t, 1, trends["temporal"])
		assert.Equal(t, 1, trends["pattern"])
	})
}

func TestAnomalyDetector_GetUserAnomalyHistory(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()
	tenantID := uuid.New()

	// Add some anomalies to the mock repo
	userID := uuid.New()
	anomalies := []AnomalyDetection{
		{
			ID:          uuid.New(),
			TenantID:    tenantID,
			UserID:      &userID,
			AnomalyType: "behavioral",
			Severity:    "high",
			Title:       "Test Anomaly",
			Status:      "open",
		},
	}

	repo := &mockAnomalyRepository{anomalies: anomalies}
	detector := NewAnomalyDetector(repo, nil, 75.0, logger)

	history, err := detector.GetUserAnomalyHistory(ctx, tenantID, userID, 10)

	assert.NoError(t, err)
	assert.NotNil(t, history)
	// The mock returns the anomalies directly
	assert.Len(t, history, 1)
}

func TestCalculateBehavioralConfidence(t *testing.T) {
	tests := []struct {
		name      string
		avgRisk   float64
		offHours  int
		unusual   int
		min       float64
		max       float64
	}{
		{"low risk", 10.0, 0, 0, 6.0, 6.0},
		// Formula: avgRisk * 0.6 + offHours * 5 + unusual * 10, capped at 100
		{"medium risk", 50.0, 2, 1, 50.0, 50.0},           // 50*0.6 + 2*5 + 1*10 = 30 + 10 + 10 = 50
		{"high risk", 80.0, 5, 3, 100.0, 100.0},            // 80*0.6 + 5*5 + 3*10 = 48 + 25 + 30 = 103 → 100
		{"maximum", 100.0, 10, 10, 100.0, 100.0},           // 100*0.6 + 10*5 + 10*10 = 60 + 50 + 100 = 210 → 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBehavioralConfidence(tt.avgRisk, tt.offHours, tt.unusual)
			assert.GreaterOrEqual(t, result, tt.min)
			assert.LessOrEqual(t, result, tt.max)
		})
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name      string
		confidence float64
		severity  string
	}{
		{"critical", 95.0, "critical"},
		{"high", 80.0, "high"},
		{"medium", 60.0, "medium"},
		{"low", 40.0, "low"},
		{"very low", 10.0, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSeverity(tt.confidence)
			assert.Equal(t, tt.severity, result)
		})
	}
}

func TestAnomalyDetection_Struct(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	sessionID := uuid.New()
	targetHost := "server1.example.com"
	description := "Test anomaly"
	indicators := map[string]interface{}{"type": "test"}
	indicatorsJSON, _ := json.Marshal(indicators)
	modelVersion := "1.0"
	assignedTo := uuid.New()
	resolutionNotes := "Resolved"
	resolvedAt := now.Add(time.Hour)
	resolvedBy := uuid.New()
	autoAction := "Blocked"

	anomaly := AnomalyDetection{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		AnomalyType:     "behavioral",
		UserID:          &userID,
		SessionID:       &sessionID,
		TargetHost:      &targetHost,
		Severity:        "high",
		ConfidenceScore: 85.5,
		RiskScore:       75.0,
		Title:           "Test Anomaly",
		Description:     &description,
		Indicators:      indicatorsJSON,
		DetectionMethod: "test",
		DetectedAt:      now,
		ModelVersion:    &modelVersion,
		Status:          "open",
		AssignedTo:      &assignedTo,
		ResolutionNotes: &resolutionNotes,
		ResolvedAt:      &resolvedAt,
		ResolvedBy:      &resolvedBy,
		AutoTriggered:   true,
		AutoActionTaken: &autoAction,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.NotEqual(t, uuid.Nil, anomaly.ID)
	assert.Equal(t, "behavioral", anomaly.AnomalyType)
	assert.NotNil(t, anomaly.UserID)
	assert.NotNil(t, anomaly.SessionID)
	assert.Equal(t, "high", anomaly.Severity)
	assert.Equal(t, 85.5, anomaly.ConfidenceScore)
	assert.NotEmpty(t, anomaly.Indicators)
	assert.True(t, anomaly.AutoTriggered)
	assert.NotNil(t, anomaly.ResolvedAt)
}

func TestAnomalyDetection_IndicatorsJSON(t *testing.T) {
	now := time.Now()

	t.Run("marshal indicators", func(t *testing.T) {
		indicators := map[string]interface{}{
			"risk_score": 75.0,
			"off_hours":  5,
			"command":    "rm",
			"count":      10,
		}

		anomaly := AnomalyDetection{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			AnomalyType:     "pattern",
			Severity:        "high",
			ConfidenceScore: 85.0,
			RiskScore:       75.0,
			Title:           "Test",
			DetectionMethod: "test",
			DetectedAt:      now,
			Status:          "open",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		indicatorsJSON, err := json.Marshal(indicators)
		require.NoError(t, err)
		anomaly.Indicators = indicatorsJSON

		assert.NotEmpty(t, anomaly.Indicators)

		var decoded map[string]interface{}
		err = json.Unmarshal(anomaly.Indicators, &decoded)
		require.NoError(t, err)
		assert.Equal(t, 75.0, decoded["risk_score"])
		assert.Equal(t, 5.0, decoded["off_hours"])
	})

	t.Run("nil indicators", func(t *testing.T) {
		anomaly := AnomalyDetection{
			ID:              uuid.New(),
			TenantID:        uuid.New(),
			AnomalyType:     "behavioral",
			Severity:        "medium",
			ConfidenceScore: 50.0,
			Title:           "Test",
			DetectionMethod: "test",
			DetectedAt:      now,
			Status:          "open",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		assert.Empty(t, anomaly.Indicators)
	})
}

func TestRansomwareEvent_Struct(t *testing.T) {
	now := time.Now()
	detectionID := uuid.New()
	containmentStatus := "contained"
	recoveryStatus := "restoring"

	event := RansomwareEvent{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		DetectionID:          detectionID,
		EncryptionActivity:   true,
		MassFileModification: true,
		SuspiciousProcesses:  []string{"chmod", "chown", "find"},
		AffectedPaths:        []string{"/data", "/home"},
		FilesAffected:        500,
		SystemsAffected:      3,
		DataExfiltrated:      false,
		EmergencyTriggered:   true,
		SessionsTerminated:   5,
		CredentialsRevoked:   2,
		ContainmentStatus:    &containmentStatus,
		RecoveryStatus:       &recoveryStatus,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, detectionID, event.DetectionID)
	assert.True(t, event.EncryptionActivity)
	assert.True(t, event.MassFileModification)
	assert.NotEmpty(t, event.SuspiciousProcesses)
	assert.Equal(t, 500, event.FilesAffected)
	assert.True(t, event.EmergencyTriggered)
	assert.NotNil(t, event.ContainmentStatus)
	assert.Equal(t, "contained", *event.ContainmentStatus)
}

func TestAnomalyStatus_Values(t *testing.T) {
	validStatuses := []AnomalyStatus{
		AnomalyStatusOpen,
		AnomalyStatusInvestigating,
		AnomalyStatusResolved,
		AnomalyStatusFalsePositive,
		AnomalyStatusIgnored,
	}

	expectedValues := []string{"open", "investigating", "resolved", "false_positive", "ignored"}

	for i, status := range validStatuses {
		t.Run(expectedValues[i], func(t *testing.T) {
			assert.Equal(t, expectedValues[i], string(status))
		})
	}
}

func TestAnomalyType_Values(t *testing.T) {
	validTypes := []AnomalyType{
		AnomalyTypeBehavioral,
		AnomalyTypeTemporal,
		AnomalyTypeSpatial,
		AnomalyTypePattern,
		AnomalyTypeVolumetric,
		AnomalyTypeRansomware,
	}

	expectedValues := []string{"behavioral", "temporal", "spatial", "pattern", "volumetric", "ransomware"}

	for i, atype := range validTypes {
		t.Run(expectedValues[i], func(t *testing.T) {
			assert.Equal(t, expectedValues[i], string(atype))
		})
	}
}

func TestAnomalyDetection_StatusTransition(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	anomaly := &AnomalyDetection{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		AnomalyType:     "behavioral",
		UserID:          &userID,
		Severity:        "high",
		ConfidenceScore: 85.0,
		Title:           "Test",
		DetectionMethod: "test",
		DetectedAt:      now,
		Status:          string(AnomalyStatusOpen),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	t.Run("open to investigating", func(t *testing.T) {
		anomaly.Status = string(AnomalyStatusInvestigating)
		assert.Equal(t, "investigating", anomaly.Status)
	})

	t.Run("investigating to resolved", func(t *testing.T) {
		anomaly.Status = string(AnomalyStatusResolved)
		resolvedAt := now.Add(time.Hour)
		anomaly.ResolvedAt = &resolvedAt
		assert.Equal(t, "resolved", anomaly.Status)
		assert.NotNil(t, anomaly.ResolvedAt)
	})

	t.Run("resolved to false positive", func(t *testing.T) {
		anomaly.Status = string(AnomalyStatusFalsePositive)
		assert.Equal(t, "false_positive", anomaly.Status)
	})
}

func TestAnomalyPattern_Struct(t *testing.T) {
	indicators := map[string]interface{}{
		"count": 10,
		"users": []string{"user1", "user2"},
	}

	pattern := AnomalyPattern{
		Type:        AnomalyTypeBehavioral,
		Severity:    RiskLevelHigh,
		Confidence:  85.0,
		Title:       "Test Pattern",
		Description: "Test description",
		Indicators:  indicators,
	}

	assert.Equal(t, AnomalyTypeBehavioral, pattern.Type)
	assert.Equal(t, RiskLevelHigh, pattern.Severity)
	assert.Equal(t, 85.0, pattern.Confidence)
	assert.NotNil(t, pattern.Indicators)
}

func TestConfidenceScoreCalculation(t *testing.T) {
	t.Run("behavioral confidence is capped at 100", func(t *testing.T) {
		result := calculateBehavioralConfidence(100.0, 20, 20)
		assert.LessOrEqual(t, result, 100.0)
	})

	t.Run("temporal confidence is capped", func(t *testing.T) {
		confidence := math.Min(95.0, float64(100)*5)
		assert.LessOrEqual(t, confidence, 95.0)
	})

	t.Run("volumetric z-score calculation", func(t *testing.T) {
		stdDev := 30.0
		totalCommands := 500
		avgCommands := 100

		zScore := math.Abs(float64(totalCommands-avgCommands) / stdDev)
		confidence := math.Min(98.0, zScore*20)

		assert.LessOrEqual(t, confidence, 98.0)
		assert.Greater(t, confidence, 0.0)
	})
}

func TestAnomalyDetection_CreateDefault(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	userID := uuid.New()

	anomaly := AnomalyDetection{
		ID:              uuid.New(),
		TenantID:        tenantID,
		AnomalyType:     "behavioral",
		UserID:          &userID,
		Severity:        "high",
		ConfidenceScore: 75.0,
		RiskScore:       60.0,
		Title:           "Test",
		Description:     stringPtr("Test description"),
		DetectionMethod: "test",
		DetectedAt:      now,
		Status:          "open",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.NotEqual(t, uuid.Nil, anomaly.ID)
	assert.Equal(t, tenantID, anomaly.TenantID)
	assert.NotNil(t, anomaly.Description)
	assert.NotNil(t, anomaly.UserID)
}

func TestAnomalyDetection_WithResolution(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	resolvedBy := uuid.New()
	resolutionNotes := "Investigated and resolved"
	resolvedAt := now.Add(2 * time.Hour)

	anomaly := &AnomalyDetection{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		AnomalyType:     "pattern",
		UserID:          &userID,
		Severity:        "medium",
		ConfidenceScore: 60.0,
		RiskScore:       50.0,
		Title:           "Resolved Anomaly",
		DetectionMethod: "test",
		DetectedAt:      now,
		Status:          string(AnomalyStatusResolved),
		AssignedTo:      &userID,
		ResolutionNotes: &resolutionNotes,
		ResolvedAt:      &resolvedAt,
		ResolvedBy:      &resolvedBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, "resolved", anomaly.Status)
	assert.NotNil(t, anomaly.ResolvedAt)
	assert.NotNil(t, anomaly.ResolvedBy)
	assert.NotNil(t, anomaly.ResolutionNotes)
	assert.Equal(t, "Investigated and resolved", *anomaly.ResolutionNotes)
}

func TestRansomwareDetection_WithIndicators(t *testing.T) {
	now := time.Now()

	rawIndicators := map[string]interface{}{
		"encryption_detected": true,
		"file_count":          1500,
		"processes":           []string{"openssl", "chmod"},
	}
	rawIndicatorsJSON, _ := json.Marshal(rawIndicators)

	event := &RansomwareEvent{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		DetectionID:          uuid.New(),
		EncryptionActivity:   true,
		MassFileModification: true,
		SuspiciousProcesses:  []string{"openssl", "chmod", "chown"},
		AffectedPaths:        []string{"/data", "/home/user/docs"},
		FilesAffected:        1500,
		SystemsAffected:      5,
		DataExfiltrated:      false,
		EmergencyTriggered:   true,
		RawIndicators:        rawIndicatorsJSON,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.True(t, event.EncryptionActivity)
	assert.True(t, event.MassFileModification)
	assert.Len(t, event.SuspiciousProcesses, 3)
	assert.Equal(t, 1500, event.FilesAffected)
	assert.NotEmpty(t, event.RawIndicators)
}
