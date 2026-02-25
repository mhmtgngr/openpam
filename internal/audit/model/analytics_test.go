// Package model provides tests for audit analytics domain models
package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Compliance Report Tests
// =============================================================================

func TestComplianceReport_Struct(t *testing.T) {
	now := time.Now()
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now
	generatedAt := now
	overallScore := 87.5
	summary := "Compliance evaluation completed"
	findings := []byte(`[{"severity":"high","control":"CC1.1"}]`)
	recommendations := []byte(`[{"action":"review","priority":"medium"}]`)
	metadata := []byte(`{"assessor":"john.doe","version":"1.0"}`)

	report := ComplianceReport{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		ReportName:      "SOC2 Compliance Report Q4 2024",
		Framework:       FrameworkSOC2,
		Version:         "2022",
		GeneratedAt:     generatedAt,
		GeneratedBy:     uuid.New(),
		Status:          string(ComplianceStatusPassed),
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
		Metadata:        metadata,
		CreatedAt:       now,
	}

	assert.NotEqual(t, uuid.Nil, report.ID)
	assert.NotEqual(t, uuid.Nil, report.TenantID)
	assert.Equal(t, "SOC2 Compliance Report Q4 2024", report.ReportName)
	assert.Equal(t, FrameworkSOC2, report.Framework)
	assert.Equal(t, "2022", report.Version)
	assert.Equal(t, "passed", report.Status)
	assert.NotNil(t, report.OverallScore)
	assert.Equal(t, 87.5, *report.OverallScore)
	assert.Equal(t, 50, report.TotalControls)
	assert.Equal(t, 45, report.PassedControls)
	assert.Equal(t, 3, report.FailedControls)
	assert.Equal(t, 2, report.SkippedControls)
	assert.NotNil(t, report.Summary)
	assert.Equal(t, "Compliance evaluation completed", *report.Summary)
	assert.NotEmpty(t, report.Findings)
	assert.NotEmpty(t, report.Recommendations)
	assert.NotEmpty(t, report.Metadata)
}

func TestComplianceControlEvaluation_Struct(t *testing.T) {
	now := time.Now()
	score := 95.0
	category := "Access Control"
	findings := "Control implemented correctly with proper documentation"

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
		EvidenceURLs:    []string{"http://example.com/evidence1", "http://example.com/evidence2"},
		Findings:        &findings,
		RemediationSteps: []string{},
		EvaluatedAt:     now,
	}

	assert.NotEqual(t, uuid.Nil, evaluation.ID)
	assert.Equal(t, "CC1.1", evaluation.ControlID)
	assert.Equal(t, "Access Control Policy", evaluation.ControlName)
	assert.NotNil(t, evaluation.ControlCategory)
	assert.Equal(t, "Access Control", *evaluation.ControlCategory)
	assert.Equal(t, "passed", evaluation.Status)
	assert.NotNil(t, evaluation.Score)
	assert.Equal(t, 95.0, *evaluation.Score)
	assert.Equal(t, 5, evaluation.EvidenceCount)
	assert.NotEmpty(t, evaluation.EvidenceURLs)
	assert.NotNil(t, evaluation.Findings)
}

func TestComplianceException_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)
	approvedBy := uuid.New()
	approvedAt := now
	riskAcceptedBy := uuid.New()
	reviewDate := now.AddDate(0, 6, 0)
	businessReason := "Legacy system compatibility"
	compensatingControls := []string{"Manual review", "Enhanced monitoring", "Weekly audits"}

	exception := ComplianceException{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		ControlID:            "CC3.1",
		ControlName:          "Authentication",
		Framework:            FrameworkSOC2,
		Status:               string(ExceptionStatusApproved),
		RiskLevel:            string(SeverityMedium),
		RequestedBy:          uuid.New(),
		RequestedAt:          now,
		ApprovedBy:           &approvedBy,
		ApprovedAt:           &approvedAt,
		ExpiresAt:            &expiresAt,
		Justification:        "Technical limitation prevents implementation until Q2 2025",
		BusinessReason:       &businessReason,
		CompensatingControls: compensatingControls,
		RiskAcceptedBy:       &riskAcceptedBy,
		RiskAcceptedAt:       &approvedAt,
		ReviewDate:           &reviewDate,
		ReviewNotes:          stringPtr("Review in 6 months for migration progress"),
		Metadata:             []byte(`{"approved_by_committee":true}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.NotEqual(t, uuid.Nil, exception.ID)
	assert.Equal(t, "CC3.1", exception.ControlID)
	assert.Equal(t, "Authentication", exception.ControlName)
	assert.Equal(t, FrameworkSOC2, exception.Framework)
	assert.Equal(t, "approved", exception.Status)
	assert.Equal(t, "medium", exception.RiskLevel)
	assert.NotNil(t, exception.ApprovedBy)
	assert.NotNil(t, exception.ExpiresAt)
	assert.NotEmpty(t, exception.Justification)
	assert.NotNil(t, exception.BusinessReason)
	assert.NotEmpty(t, exception.CompensatingControls)
	assert.NotNil(t, exception.RiskAcceptedBy)
	assert.NotNil(t, exception.ReviewDate)
}

func TestComplianceException_NoOptionalFields(t *testing.T) {
	now := time.Now()
	exception := ComplianceException{
		ID:          uuid.New(),
		TenantID:    uuid.New(),
		ControlID:   "CC5.2",
		ControlName: "Encryption at Rest",
		Framework:   FrameworkISO27001,
		Status:      string(ExceptionStatusPending),
		RiskLevel:   string(SeverityLow),
		RequestedBy: uuid.New(),
		RequestedAt: now,
		Justification: "Waiting for vendor update",
		// All optional fields left nil
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Nil(t, exception.ApprovedBy)
	assert.Nil(t, exception.ApprovedAt)
	assert.Nil(t, exception.ExpiresAt)
	assert.Nil(t, exception.BusinessReason)
	assert.Nil(t, exception.RiskAcceptedBy)
	assert.Nil(t, exception.RiskAcceptedAt)
	assert.Nil(t, exception.ReviewDate)
}

// =============================================================================
// Anomaly Detection Tests
// =============================================================================

func TestAnomalyDetection_Struct(t *testing.T) {
	now := time.Now()
	userID := uuid.New()
	sessionID := uuid.New()
	targetHost := "prod-server-01.example.com"
	assignedTo := uuid.New()
	indicators := []byte(`{"type":"behavioral","baseline":5,"observed":50}`)
	metadata := []byte(`{"model_version":"2.1","confidence":0.95}`)

	anomaly := AnomalyDetection{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		AnomalyType:     string(AnomalyTypeBehavioral),
		UserID:          &userID,
		SessionID:       &sessionID,
		TargetHost:      &targetHost,
		Severity:        string(SeverityHigh),
		ConfidenceScore: 85.5,
		RiskScore:       75.0,
		Title:           "Unusual login pattern detected",
		Description:     stringPtr("User logged in from 3 different countries within 1 hour"),
		Indicators:      indicators,
		DetectionMethod: "ml_behavioral_analysis",
		DetectedAt:      now,
		ModelVersion:    stringPtr("v2.1.0"),
		Status:          string(AnomalyStatusOpen),
		AssignedTo:      &assignedTo,
		ResolutionNotes: nil,
		ResolvedAt:      nil,
		ResolvedBy:      nil,
		AutoTriggered:   true,
		AutoActionTaken: stringPtr("session_terminated"),
		Metadata:        metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.NotEqual(t, uuid.Nil, anomaly.ID)
	assert.Equal(t, "behavioral", anomaly.AnomalyType)
	assert.NotNil(t, anomaly.UserID)
	assert.NotNil(t, anomaly.SessionID)
	assert.NotNil(t, anomaly.TargetHost)
	assert.Equal(t, "high", anomaly.Severity)
	assert.Equal(t, 85.5, anomaly.ConfidenceScore)
	assert.Equal(t, 75.0, anomaly.RiskScore)
	assert.Equal(t, "Unusual login pattern detected", anomaly.Title)
	assert.NotNil(t, anomaly.Description)
	assert.NotEmpty(t, anomaly.Indicators)
	assert.Equal(t, "ml_behavioral_analysis", anomaly.DetectionMethod)
	assert.True(t, anomaly.AutoTriggered)
}

func TestAnomalyDetection_Ransomware(t *testing.T) {
	now := time.Now()
	detectionID := uuid.New()

	event := RansomwareEvent{
		ID:                   uuid.New(),
		TenantID:             uuid.New(),
		DetectionID:          detectionID,
		EncryptionActivity:   true,
		MassFileModification: true,
		SuspiciousProcesses:  []string{"cryptolocker.exe", "vssadmin.exe"},
		AffectedPaths:        []string{"/data", "/backup", "/shared"},
		FilesAffected:        15420,
		SystemsAffected:      5,
		DataExfiltrated:      true,
		EmergencyTriggered:   true,
		SessionsTerminated:   3,
		CredentialsRevoked:   2,
		ContainmentStatus:    stringPtr("partial"),
		RecoveryStatus:       stringPtr("in_progress"),
		RawIndicators:        []byte(`{"entropy":0.98,"file_types":[".encrypted"]}`),
		Metadata:             []byte(`{"incident_id":"INC-2024-001"}`),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.True(t, event.EncryptionActivity)
	assert.True(t, event.MassFileModification)
	assert.NotEmpty(t, event.SuspiciousProcesses)
	assert.NotEmpty(t, event.AffectedPaths)
	assert.Equal(t, 15420, event.FilesAffected)
	assert.Equal(t, 5, event.SystemsAffected)
	assert.True(t, event.DataExfiltrated)
	assert.True(t, event.EmergencyTriggered)
	assert.Equal(t, 3, event.SessionsTerminated)
	assert.Equal(t, 2, event.CredentialsRevoked)
}

// =============================================================================
// SSH Key Analytics Tests
// =============================================================================

func TestSSHKeyAnalytics_Struct(t *testing.T) {
	now := time.Now()
	date := now.Truncate(24 * time.Hour)
	firstUse := now.Add(-30 * 24 * time.Hour)
	lastUse := now
	avgDuration := 300.0

	analytics := SSHKeyAnalytics{
		ID:                        uuid.New(),
		TenantID:                  uuid.New(),
		SSHKeyID:                  uuid.New(),
		Date:                      date,
		UsageCount:                25,
		UniqueUsers:               3,
		UniqueTargets:             8,
		FirstUseTime:              &firstUse,
		LastUseTime:               &lastUse,
		AvgSessionDurationSeconds: &avgDuration,
		OffHoursUsage:             5,
		UnusualSourceUsage:        2,
		FailedAttempts:            1,
		Metadata:                  []byte(`{"key_type":"rsa","bits":4096}`),
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	assert.NotEqual(t, uuid.Nil, analytics.ID)
	assert.Equal(t, 25, analytics.UsageCount)
	assert.Equal(t, 3, analytics.UniqueUsers)
	assert.Equal(t, 8, analytics.UniqueTargets)
	assert.NotNil(t, analytics.FirstUseTime)
	assert.NotNil(t, analytics.LastUseTime)
	assert.NotNil(t, analytics.AvgSessionDurationSeconds)
	assert.Equal(t, 300.0, *analytics.AvgSessionDurationSeconds)
	assert.Equal(t, 5, analytics.OffHoursUsage)
	assert.Equal(t, 2, analytics.UnusualSourceUsage)
	assert.Equal(t, 1, analytics.FailedAttempts)
}

func TestSSHKeyAnalytics_NoUsage(t *testing.T) {
	now := time.Now()
	date := now.Truncate(24 * time.Hour)

	analytics := SSHKeyAnalytics{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		SSHKeyID:          uuid.New(),
		Date:              date,
		UsageCount:        0,
		UniqueUsers:       0,
		UniqueTargets:     0,
		FirstUseTime:      nil,
		LastUseTime:       nil,
		OffHoursUsage:     0,
		UnusualSourceUsage: 0,
		FailedAttempts:    0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	assert.Equal(t, 0, analytics.UsageCount)
	assert.Nil(t, analytics.FirstUseTime)
	assert.Nil(t, analytics.LastUseTime)
}

// =============================================================================
// Command Blacklist Tests
// =============================================================================

func TestCommandBlacklist_Struct(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	baseCommand := "rm"
	appliesToUsers := []uuid.UUID{uuid.New(), uuid.New()}
	appliesToGroups := []uuid.UUID{uuid.New()}
	appliesToTargets := []string{"/var/*", "/etc/*", "/home/*"}
	overrideRoles := []string{"admin", "super_admin"}
	riskCategory := "data_destruction"

	blacklist := CommandBlacklist{
		ID:               uuid.New(),
		TenantID:         &tenantID,
		CommandPattern:   "rm -rf /*",
		PatternType:      string(PatternTypeGlob),
		BaseCommand:      &baseCommand,
		Action:           string(CommandActionBlock),
		Severity:         string(SeverityCritical),
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
	assert.Equal(t, "rm -rf /*", blacklist.CommandPattern)
	assert.Equal(t, "glob", blacklist.PatternType)
	assert.NotNil(t, blacklist.BaseCommand)
	assert.Equal(t, "rm", *blacklist.BaseCommand)
	assert.Equal(t, "block", blacklist.Action)
	assert.Equal(t, "critical", blacklist.Severity)
	assert.NotEmpty(t, blacklist.AppliesToUsers)
	assert.NotEmpty(t, blacklist.AppliesToGroups)
	assert.NotEmpty(t, blacklist.AppliesToTargets)
	assert.True(t, blacklist.AllowOverride)
	assert.NotEmpty(t, blacklist.OverrideRoles)
	assert.True(t, blacklist.Enabled)
}

func TestCommandBlacklist_GlobalRule(t *testing.T) {
	now := time.Now()
	baseCommand := "dd"
	riskCategory := "data_destruction"

	blacklist := CommandBlacklist{
		ID:              uuid.New(),
		TenantID:        nil, // NULL for global rules
		CommandPattern:  "dd if=/dev/zero of=/dev/sda",
		PatternType:     string(PatternTypeExact),
		BaseCommand:     &baseCommand,
		Action:          string(CommandActionBlock),
		Severity:        string(SeverityCritical),
		AppliesToUsers:  nil,
		AppliesToGroups: nil,
		AppliesToTargets: nil,
		AllowOverride:   false,
		OverrideRoles:   nil,
		Reason:          "Global rule: destructive disk operation",
		RiskCategory:    &riskCategory,
		Enabled:         true,
		CreatedBy:       uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Nil(t, blacklist.TenantID, "Global rules should have NULL tenant_id")
	assert.False(t, blacklist.AllowOverride, "Global rules should not allow override")
	assert.Nil(t, blacklist.AppliesToUsers, "Global rules apply to all users")
}

// =============================================================================
// Enum Tests
// =============================================================================

func TestEnums_Framework(t *testing.T) {
	tests := []struct {
		name      string
		framework string
		value     string
	}{
		{"SOC2", FrameworkSOC2, "SOC2"},
		{"ISO27001", FrameworkISO27001, "ISO27001"},
		{"PCI-DSS", FrameworkPCIDSS, "PCI-DSS"},
		{"HIPAA", FrameworkHIPAA, "HIPAA"},
		{"NIST-800-53", FrameworkNIST, "NIST-800-53"},
		{"GDPR", FrameworkGDPR, "GDPR"},
		{"CUSTOM", FrameworkCustom, "CUSTOM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, tt.framework)
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

func TestEnums_ExceptionStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ExceptionStatus
		value  string
	}{
		{"pending", ExceptionStatusPending, "pending"},
		{"approved", ExceptionStatusApproved, "approved"},
		{"denied", ExceptionStatusDenied, "denied"},
		{"expired", ExceptionStatusExpired, "expired"},
		{"revoked", ExceptionStatusRevoked, "revoked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, string(tt.status))
		})
	}
}

func TestEnums_AnomalyType(t *testing.T) {
	tests := []struct {
		name  string
		atype AnomalyType
		value string
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

func TestEnums_Severity(t *testing.T) {
	tests := []struct {
		name  string
		level Severity
		value string
	}{
		{"low", SeverityLow, "low"},
		{"medium", SeverityMedium, "medium"},
		{"high", SeverityHigh, "high"},
		{"critical", SeverityCritical, "critical"},
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

func TestEnums_PatternType(t *testing.T) {
	tests := []struct {
		name string
		ptype PatternType
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

// =============================================================================
// Filter Tests
// =============================================================================

func TestFilter_ComplianceReportFilter(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	framework := FrameworkSOC2
	status := string(ComplianceStatusPassed)
	dateFrom := now.Add(-30 * 24 * time.Hour)
	dateTo := now

	filter := ComplianceReportFilter{
		TenantID:  &tenantID,
		Framework: &framework,
		Status:    &status,
		DateFrom:  &dateFrom,
		DateTo:    &dateTo,
	}

	assert.NotNil(t, filter.TenantID)
	assert.NotNil(t, filter.Framework)
	assert.Equal(t, "SOC2", *filter.Framework)
	assert.NotNil(t, filter.Status)
	assert.Equal(t, "passed", *filter.Status)
	assert.NotNil(t, filter.DateFrom)
	assert.NotNil(t, filter.DateTo)
}

func TestFilter_ComplianceExceptionFilter(t *testing.T) {
	tenantID := uuid.New()
	controlID := "CC1.1"
	framework := FrameworkSOC2
	status := string(ExceptionStatusApproved)
	riskLevel := string(SeverityMedium)

	filter := ComplianceExceptionFilter{
		TenantID:      &tenantID,
		ControlID:     &controlID,
		Framework:     &framework,
		Status:        &status,
		RiskLevel:     &riskLevel,
		IncludeExpired: true,
	}

	assert.NotNil(t, filter.ControlID)
	assert.Equal(t, "CC1.1", *filter.ControlID)
	assert.True(t, filter.IncludeExpired)
}

func TestFilter_AnomalyFilter(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	userID := uuid.New()
	anomalyType := string(AnomalyTypeBehavioral)
	severity := string(SeverityHigh)
	status := string(AnomalyStatusOpen)
	dateFrom := now.Add(-7 * 24 * time.Hour)
	dateTo := now

	filter := AnomalyFilter{
		TenantID:    &tenantID,
		UserID:      &userID,
		AnomalyType: &anomalyType,
		Severity:    &severity,
		Status:      &status,
		DateFrom:    &dateFrom,
		DateTo:      &dateTo,
	}

	assert.NotNil(t, filter.UserID)
	assert.Equal(t, "behavioral", *filter.AnomalyType)
	assert.Equal(t, "high", *filter.Severity)
	assert.Equal(t, "open", *filter.Status)
}

func TestFilter_CommandBlacklistFilter(t *testing.T) {
	tenantID := uuid.New()
	enabled := true
	patternType := string(PatternTypeRegex)
	action := string(CommandActionBlock)
	severity := string(SeverityCritical)

	filter := CommandBlacklistFilter{
		TenantID:    &tenantID,
		Enabled:     &enabled,
		PatternType: &patternType,
		Action:      &action,
		Severity:    &severity,
	}

	assert.NotNil(t, filter.Enabled)
	assert.True(t, *filter.Enabled)
	assert.Equal(t, "regex", *filter.PatternType)
}

func TestFilter_SSHKeyAnalyticsFilter(t *testing.T) {
	now := time.Now()
	tenantID := uuid.New()
	sshKeyID := uuid.New()
	dateFrom := now.Add(-30 * 24 * time.Hour)
	dateTo := now

	filter := SSHKeyAnalyticsFilter{
		TenantID: &tenantID,
		SSHKeyID: &sshKeyID,
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	}

	assert.NotNil(t, filter.SSHKeyID)
}

// =============================================================================
// Request/Response DTO Tests
// =============================================================================

func TestCreateComplianceReportRequest_Valid(t *testing.T) {
	now := time.Now()
	periodStart := now.Add(-30 * 24 * time.Hour)
	periodEnd := now

	req := CreateComplianceReportRequest{
		ReportName:  "Q4 2024 SOC2 Report",
		Framework:   FrameworkSOC2,
		Version:     "2022",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
	}

	assert.Equal(t, "Q4 2024 SOC2 Report", req.ReportName)
	assert.Equal(t, FrameworkSOC2, req.Framework)
	assert.Equal(t, "2022", req.Version)
	assert.False(t, req.PeriodStart.IsZero())
	assert.False(t, req.PeriodEnd.IsZero())
}

func TestCreateComplianceExceptionRequest_Valid(t *testing.T) {
	now := time.Now()
	expiresAt := now.AddDate(1, 0, 0)

	req := CreateComplianceExceptionRequest{
		ControlID:            "CC3.1",
		ControlName:          "Multi-Factor Authentication",
		Framework:            FrameworkSOC2,
		RiskLevel:            string(SeverityMedium),
		Justification:        "Third-party provider does not support MFA",
		BusinessReason:       "Critical vendor integration",
		CompensatingControls: []string{"IP whitelist", "Certificate-based auth"},
		ExpiresAt:            &expiresAt,
	}

	assert.Equal(t, "CC3.1", req.ControlID)
	assert.NotEmpty(t, req.Justification)
	assert.NotEmpty(t, req.CompensatingControls)
	assert.NotNil(t, req.ExpiresAt)
}

func TestUpdateAnomalyRequest_Valid(t *testing.T) {
	assignedTo := uuid.New()
	resolutionNotes := "Investigated - legitimate user travel"

	req := UpdateAnomalyRequest{
		Status:          stringPtr("resolved"),
		AssignedTo:      &assignedTo,
		ResolutionNotes: &resolutionNotes,
	}

	assert.NotNil(t, req.Status)
	assert.Equal(t, "resolved", *req.Status)
	assert.NotNil(t, req.AssignedTo)
	assert.NotNil(t, req.ResolutionNotes)
}

func TestCreateCommandBlacklistRequest_Valid(t *testing.T) {
	tenantID := uuid.New()
	req := CreateCommandBlacklistRequest{
		CommandPattern:   "chmod 000 /etc/*",
		PatternType:      string(PatternTypeExact),
		BaseCommand:      "chmod",
		Action:           string(CommandActionBlock),
		Severity:         string(SeverityCritical),
		AppliesToUsers:   []uuid.UUID{tenantID},
		AppliesToGroups:  nil,
		AppliesToTargets: []string{"/etc/*"},
		AllowOverride:    false,
		OverrideRoles:    nil,
		Reason:           "Dangerous permission modification",
		RiskCategory:     "privilege_escalation",
	}

	assert.Equal(t, "chmod 000 /etc/*", req.CommandPattern)
	assert.Equal(t, "exact", req.PatternType)
	assert.False(t, req.AllowOverride)
	assert.NotEmpty(t, req.Reason)
}

func TestUpdateCommandBlacklistRequest_Valid(t *testing.T) {
	enabled := false
	newReason := "Updated after policy review"

	req := UpdateCommandBlacklistRequest{
		CommandPattern: stringPtr("rm -rf *"),
		Enabled:        &enabled,
		Reason:         &newReason,
	}

	assert.NotNil(t, req.CommandPattern)
	assert.NotNil(t, req.Enabled)
	assert.False(t, *req.Enabled)
	assert.NotNil(t, req.Reason)
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestComplianceReport_JSON(t *testing.T) {
	now := time.Now()
	score := 85.0
	summary := "Test summary"

	report := ComplianceReport{
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		TenantID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		ReportName:     "Test Report",
		Framework:      FrameworkSOC2,
		GeneratedAt:    now,
		GeneratedBy:    uuid.New(),
		Status:         "passed",
		OverallScore:   &score,
		TotalControls:  10,
		PassedControls: 8,
		Summary:        &summary,
		CreatedAt:      now,
	}

	// Marshal to JSON
	data, err := json.Marshal(report)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled ComplianceReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, report.ID, unmarshaled.ID)
	assert.Equal(t, report.ReportName, unmarshaled.ReportName)
	assert.Equal(t, report.Framework, unmarshaled.Framework)
	assert.Equal(t, report.OverallScore, unmarshaled.OverallScore)
}

func TestComplianceException_EvidenceURLs_JSON(t *testing.T) {
	now := time.Now()
	urls := []string{"https://example.com/evidence1", "https://example.com/evidence2"}

	evaluation := ComplianceControlEvaluation{
		ID:           uuid.New(),
		ReportID:     uuid.New(),
		TenantID:     uuid.New(),
		ControlID:    "CC1.1",
		ControlName:  "Test Control",
		Status:       "passed",
		EvidenceURLs: urls,
		EvaluatedAt:  now,
	}

	data, err := json.Marshal(evaluation)
	require.NoError(t, err)

	var unmarshaled ComplianceControlEvaluation
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, urls, unmarshaled.EvidenceURLs)
}

// =============================================================================
// Helper Functions
// =============================================================================

func stringPtr(s string) *string {
	return &s
}

// =============================================================================
// Edge Cases and Validation Tests
// =============================================================================

func TestComplianceReport_ScoreRange(t *testing.T) {
	tests := []struct {
		name  string
		score *float64
		valid bool
	}{
		{"valid score", float64Ptr(75.5), true},
		{"perfect score", float64Ptr(100.0), true},
		{"zero score", float64Ptr(0.0), true},
		{"nil score", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := ComplianceReport{
				OverallScore: tt.score,
			}

			if tt.score != nil {
				assert.GreaterOrEqual(t, *tt.score, 0.0)
				assert.LessOrEqual(t, *tt.score, 100.0)
			}
			assert.NotNil(t, report)
		})
	}
}

func TestAnomalyDetection_ConfidenceScoreRange(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		valid bool
	}{
		{"minimum confidence", 0.0, true},
		{"maximum confidence", 100.0, true},
		{"valid confidence", 75.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anomaly := AnomalyDetection{
				ConfidenceScore: tt.score,
			}

			assert.GreaterOrEqual(t, anomaly.ConfidenceScore, 0.0)
			assert.LessOrEqual(t, anomaly.ConfidenceScore, 100.0)
		})
	}
}

func TestCommandBlacklist_AllActions(t *testing.T) {
	actions := []CommandAction{
		CommandActionBlock,
		CommandActionWarn,
		CommandActionAllow,
		CommandActionAudit,
	}

	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			blacklist := CommandBlacklist{
				Action: string(action),
			}
			assert.NotEmpty(t, blacklist.Action)
		})
	}
}

func TestComplianceReport_AllStatuses(t *testing.T) {
	statuses := []ComplianceStatus{
		ComplianceStatusPending,
		ComplianceStatusPassed,
		ComplianceStatusFailed,
		ComplianceStatusPartial,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			report := ComplianceReport{
				Status: string(status),
			}
			assert.NotEmpty(t, report.Status)
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

// =============================================================================
// Metadata Handling Tests
// =============================================================================

func TestComplianceReport_Metadata(t *testing.T) {
	t.Run("marshal metadata to JSON", func(t *testing.T) {
		metadata := map[string]interface{}{
			"assessor":    "john.doe",
			"version":     "1.0",
			"reviewed_by": "jane.smith",
		}

		metadataBytes, err := json.Marshal(metadata)
		require.NoError(t, err)

		report := ComplianceReport{
			Metadata: metadataBytes,
		}

		assert.NotEmpty(t, report.Metadata)
	})

	t.Run("unmarshal metadata from JSON", func(t *testing.T) {
		metadata := []byte(`{"assessor":"john.doe","version":"1.0"}`)

		report := ComplianceReport{
			Metadata: metadata,
		}

		var result map[string]interface{}
		err := json.Unmarshal(report.Metadata, &result)
		require.NoError(t, err)

		assert.Equal(t, "john.doe", result["assessor"])
		assert.Equal(t, "1.0", result["version"])
	})
}

func TestAnomalyDetection_Indicators(t *testing.T) {
	t.Run("complex indicators structure", func(t *testing.T) {
		indicators := map[string]interface{}{
			"type":          "behavioral",
			"baseline":      5.0,
			"observed":      50.0,
			"deviation":     10.0,
			"threshold":     3.0,
			"violations":    []int{1, 2, 3},
			"confidences":   []float64{0.85, 0.90, 0.95},
			"flagged_users": []string{"user1", "user2"},
		}

		indicatorsBytes, err := json.Marshal(indicators)
		require.NoError(t, err)

		anomaly := AnomalyDetection{
			Indicators: indicatorsBytes,
		}

		assert.NotEmpty(t, anomaly.Indicators)

		var result map[string]interface{}
		err = json.Unmarshal(anomaly.Indicators, &result)
		require.NoError(t, err)

		assert.Equal(t, "behavioral", result["type"])
		assert.Equal(t, 50.0, result["observed"])
	})
}
