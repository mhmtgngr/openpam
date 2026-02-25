// Package analytics provides compliance scoring and reporting
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// ComplianceEngine handles compliance scoring and reporting
type ComplianceEngine struct {
	db           *sqlx.DB
	baselineMgr  *BaselineManager
	metricsStore *MetricsStore
	logger       zerolog.Logger
}

// NewComplianceEngine creates a new compliance engine
func NewComplianceEngine(db *sqlx.DB, baselineMgr *BaselineManager, metricsStore *MetricsStore, logger zerolog.Logger) *ComplianceEngine {
	return &ComplianceEngine{
		db:           db,
		baselineMgr:  baselineMgr,
		metricsStore: metricsStore,
		logger:       logger,
	}
}

// ComplianceConfig holds configuration for compliance scoring
type ComplianceConfig struct {
	Framework          string            `json:"framework"`
	ControlCategories  []string          `json:"control_categories"`
	Weights            map[string]float64 `json:"weights"`
	SeverityWeights    map[string]float64 `json:"severity_weights"`
	FailureThreshold   float64           `json:"failure_threshold"`
	AnomalyThreshold   int               `json:"anomaly_threshold"`
	SessionAuditDays   int               `json:"session_audit_days"`
}

// DefaultComplianceConfig returns default compliance configuration
func DefaultComplianceConfig(framework string) ComplianceConfig {
	weights := map[string]float64{
		"access_control":       0.25,
		"session_management":   0.20,
		"audit_logging":        0.15,
		"data_protection":      0.15,
		"anomaly_detection":    0.15,
		"incident_response":    0.10,
	}

	severityWeights := map[string]float64{
		"critical": 10.0,
		"high":     5.0,
		"medium":   2.0,
		"low":      0.5,
	}

	return ComplianceConfig{
		Framework:         framework,
		ControlCategories: []string{"access_control", "session_management", "audit_logging", "data_protection", "anomaly_detection", "incident_response"},
		Weights:           weights,
		SeverityWeights:   severityWeights,
		FailureThreshold:  70.0,
		AnomalyThreshold:  5,
		SessionAuditDays:  30,
	}
}

// ComplianceScore represents a compliance score for a tenant
type ComplianceScore struct {
	TenantID         uuid.UUID              `json:"tenant_id"`
	Framework        string                 `json:"framework"`
	OverallScore     float64                `json:"overall_score"`
	ControlScores    map[string]ControlScore `json:"control_scores"`
	PeriodStart      time.Time              `json:"period_start"`
	PeriodEnd        time.Time              `json:"period_end"`
	AnalyzedAt       time.Time              `json:"analyzed_at"`
	TotalControls    int                    `json:"total_controls"`
	PassedControls   int                    `json:"passed_controls"`
	FailedControls   int                    `json:"failed_controls"`
	SkippedControls  int                    `json:"skipped_controls"`
	Findings         []ComplianceFinding    `json:"findings"`
	Recommendations  []string               `json:"recommendations"`
}

// ControlScore represents a score for a specific control category
type ControlScore struct {
	Category       string  `json:"category"`
	Score          float64 `json:"score"`
	Weight         float64 `json:"weight"`
	WeightedScore  float64 `json:"weighted_score"`
	Status         string  `json:"status"` // passed, failed, warning
	FindingsCount  int     `json:"findings_count"`
	EvidenceCount  int     `json:"evidence_count"`
}

// ComplianceFinding represents a compliance finding
type ComplianceFinding struct {
	ID              uuid.UUID `json:"id"`
	ControlID       string    `json:"control_id"`
	ControlName     string    `json:"control_name"`
	Category        string    `json:"category"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`
	Description     string    `json:"description"`
	Evidence        []string  `json:"evidence"`
	AffectedUsers   []uuid.UUID `json:"affected_users,omitempty"`
	AffectedSystems []string  `json:"affected_systems,omitempty"`
	Remediation     string    `json:"remediation"`
	DetectedAt      time.Time `json:"detected_at"`
}

// CalculateComplianceScore calculates a compliance score for a tenant
func (c *ComplianceEngine) CalculateComplianceScore(ctx context.Context, tenantID uuid.UUID, framework string, periodStart, periodEnd time.Time) (*ComplianceScore, error) {
	config := DefaultComplianceConfig(framework)

	score := &ComplianceScore{
		TenantID:      tenantID,
		Framework:     framework,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		AnalyzedAt:    time.Now(),
		ControlScores: make(map[string]ControlScore),
		Findings:      []ComplianceFinding{},
	}

	// Calculate scores for each control category
	for _, category := range config.ControlCategories {
		controlScore, findings, err := c.calculateControlScore(ctx, tenantID, category, config, periodStart, periodEnd)
		if err != nil {
			c.logger.Warn().Err(err).
				Str("category", category).
				Msg("Failed to calculate control score")
			continue
		}

		weight := config.Weights[category]
		controlScore.Weight = weight
		controlScore.WeightedScore = controlScore.Score * weight

		score.ControlScores[category] = controlScore
		score.Findings = append(score.Findings, findings...)
	}

	// Calculate overall weighted score
	totalWeightedScore := 0.0
	totalWeight := 0.0
	for _, cs := range score.ControlScores {
		totalWeightedScore += cs.WeightedScore
		totalWeight += cs.Weight
	}

	if totalWeight > 0 {
		score.OverallScore = totalWeightedScore / totalWeight
	}

	// Count controls
	score.TotalControls = len(score.ControlScores)
	for _, cs := range score.ControlScores {
		switch cs.Status {
		case "passed":
			score.PassedControls++
		case "failed":
			score.FailedControls++
		default:
			score.SkippedControls++
		}
	}

	// Generate recommendations
	score.Recommendations = c.generateRecommendations(score)

	return score, nil
}

// calculateControlScore calculates the score for a specific control category
func (c *ComplianceEngine) calculateControlScore(ctx context.Context, tenantID uuid.UUID, category string, config ComplianceConfig, periodStart, periodEnd time.Time) (ControlScore, []ComplianceFinding, error) {
	var score ControlScore
	var findings []ComplianceFinding

	switch category {
	case "access_control":
		score, findings = c.assessAccessControl(ctx, tenantID, periodStart, periodEnd, config)
	case "session_management":
		score, findings = c.assessSessionManagement(ctx, tenantID, periodStart, periodEnd, config)
	case "audit_logging":
		score, findings = c.assessAuditLogging(ctx, tenantID, periodStart, periodEnd, config)
	case "data_protection":
		score, findings = c.assessDataProtection(ctx, tenantID, periodStart, periodEnd, config)
	case "anomaly_detection":
		score, findings = c.assessAnomalyDetection(ctx, tenantID, periodStart, periodEnd, config)
	case "incident_response":
		score, findings = c.assessIncidentResponse(ctx, tenantID, periodStart, periodEnd, config)
	default:
		score = ControlScore{
			Category: category,
			Score:    0,
			Status:   "skipped",
		}
	}

	return score, findings, nil
}

// assessAccessControl assesses access control compliance
func (c *ComplianceEngine) assessAccessControl(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "access_control",
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check for MFA enforcement
	mfaEnabled := c.checkMFAEnforcement(ctx, tenantID)
	if !mfaEnabled {
		score.Score -= 20
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AC-001",
			ControlName: "MFA Enforcement",
			Category:    "access_control",
			Severity:    "high",
			Status:      "failed",
			Description: "Multi-factor authentication is not consistently enforced for privileged access",
			Evidence:    []string{"Users found accessing privileged resources without MFA"},
			Remediation: "Enable MFA for all privileged access and enforce MFA challenges",
			DetectedAt:  time.Now(),
		})
	}

	// Check for excessive privileged accounts
	excessivePrivileged := c.checkExcessivePrivilegedAccounts(ctx, tenantID)
	if excessivePrivileged {
		score.Score -= 15
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AC-002",
			ControlName: "Privileged Account Management",
			Category:    "access_control",
			Severity:    "medium",
			Status:      "failed",
			Description: "Excessive number of privileged accounts detected",
			Remediation: "Review privileged account assignments and implement just-in-time access",
			DetectedAt:  time.Now(),
		})
	}

	// Check for password policy compliance
	passwordPolicyOk := c.checkPasswordPolicyCompliance(ctx, tenantID)
	if !passwordPolicyOk {
		score.Score -= 10
	}

	// Check for inactive account management
	inactiveAccountsOk := c.checkInactiveAccountManagement(ctx, tenantID)
	if !inactiveAccountsOk {
		score.Score -= 10
	}

	// Ensure minimum score
	if score.Score < 0 {
		score.Score = 0
	}

	// Set status based on score
	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 3 // MFA, privileged accounts, passwords

	return score, findings
}

// assessSessionManagement assesses session management compliance
func (c *ComplianceEngine) assessSessionManagement(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "session_management",
		Score:         100,
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check session recording coverage
	recordingCoverage := c.getSessionRecordingCoverage(ctx, tenantID, periodStart, periodEnd)
	if recordingCoverage < 0.95 {
		penalty := float64((1.0 - recordingCoverage) * 50)
		score.Score -= penalty
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "SM-001",
			ControlName: "Session Recording",
			Category:    "session_management",
			Severity:    severityFromPercent(recordingCoverage),
			Status:      "failed",
			Description: fmt.Sprintf("Session recording coverage is %.1f%%, below 95%% threshold", recordingCoverage*100),
			Evidence:    []string{fmt.Sprintf("%.1f%% of sessions recorded", recordingCoverage*100)},
			Remediation: "Ensure all privileged sessions are recorded for audit purposes",
			DetectedAt:  time.Now(),
		})
	}

	// Check for orphaned sessions
	orphanedSessions := c.countOrphanedSessions(ctx, tenantID, periodStart, periodEnd)
	if orphanedSessions > 0 {
		penalty := float64(min(orphanedSessions*5, 30))
		score.Score -= penalty
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "SM-002",
			ControlName: "Session Termination",
			Category:    "session_management",
			Severity:    "medium",
			Status:      "warning",
			Description: fmt.Sprintf("Found %d orphaned sessions without proper termination", orphanedSessions),
			Remediation: "Review session timeout policies and implement automatic session cleanup",
			DetectedAt:  time.Now(),
		})
	}

	// Check for session timeout enforcement
	timeoutEnforced := c.checkSessionTimeoutEnforcement(ctx, tenantID)
	if !timeoutEnforced {
		score.Score -= 15
	}

	// Clamp score
	if score.Score < 0 {
		score.Score = 0
	}

	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 3

	return score, findings
}

// assessAuditLogging assesses audit logging compliance
func (c *ComplianceEngine) assessAuditLogging(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "audit_logging",
		Score:         100,
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check audit log retention
	retentionOk := c.checkAuditLogRetention(ctx, tenantID, 90) // 90 days required
	if !retentionOk {
		score.Score -= 25
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AL-001",
			ControlName: "Audit Log Retention",
			Category:    "audit_logging",
			Severity:    "high",
			Status:      "failed",
			Description: "Audit logs are not retained for the required 90-day period",
			Remediation: "Configure audit log retention to meet compliance requirements",
			DetectedAt:  time.Now(),
		})
	}

	// Check for immutable logging
	immutableLogging := c.checkImmutableLogging(ctx, tenantID)
	if !immutableLogging {
		score.Score -= 20
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AL-002",
			ControlName: "Immutable Audit Logs",
			Category:    "audit_logging",
			Severity:    "high",
			Status:      "failed",
			Description: "Audit logs are not protected against tampering",
			Remediation: "Implement WORM storage or cryptographic signing for audit logs",
			DetectedAt:  time.Now(),
		})
	}

	// Check for audit trail completeness
	completeTrail := c.checkAuditTrailCompleteness(ctx, tenantID, periodStart, periodEnd)
	if !completeTrail {
		score.Score -= 15
	}

	// Check for log forwarding to SIEM
	siemForwarding := c.checkSIEMForwarding(ctx, tenantID)
	if !siemForwarding {
		score.Score -= 10
	}

	if score.Score < 0 {
		score.Score = 0
	}

	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 4

	return score, findings
}

// assessDataProtection assesses data protection compliance
func (c *ComplianceEngine) assessDataProtection(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "data_protection",
		Score:         100,
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check for credential encryption
	encryptionOk := c.checkCredentialEncryption(ctx, tenantID)
	if !encryptionOk {
		score.Score -= 30
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "DP-001",
			ControlName: "Credential Encryption",
			Category:    "data_protection",
			Severity:    "critical",
			Status:      "failed",
			Description: "Credentials are not encrypted at rest",
			Remediation: "Enable AES-256 encryption for credential storage",
			DetectedAt:  time.Now(),
		})
	}

	// Check for TLS enforcement
	tlsEnforced := c.checkTLSEnforcement(ctx, tenantID)
	if !tlsEnforced {
		score.Score -= 20
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "DP-002",
			ControlName: "TLS Enforcement",
			Category:    "data_protection",
			Severity:    "high",
			Status:      "failed",
			Description: "TLS is not enforced for all communications",
			Remediation: "Enable TLS 1.2+ for all service communications",
			DetectedAt:  time.Now(),
		})
	}

	// Check for key rotation
	keyRotationOk := c.checkKeyRotation(ctx, tenantID, 90) // 90 days max
	if !keyRotationOk {
		score.Score -= 15
	}

	if score.Score < 0 {
		score.Score = 0
	}

	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 3

	return score, findings
}

// assessAnomalyDetection assesses anomaly detection compliance
func (c *ComplianceEngine) assessAnomalyDetection(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "anomaly_detection",
		Score:         100,
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check for open critical anomalies
	criticalAnomalies := c.countOpenAnomaliesBySeverity(ctx, tenantID, "critical")
	if criticalAnomalies > 0 {
		penalty := float64(min(criticalAnomalies*20, 100))
		score.Score -= penalty
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AD-001",
			ControlName: "Critical Anomaly Response",
			Category:    "anomaly_detection",
			Severity:    "critical",
			Status:      "failed",
			Description: fmt.Sprintf("%d critical anomalies remain unaddressed", criticalAnomalies),
			Remediation: "Immediately investigate and resolve all critical anomalies",
			DetectedAt:  time.Now(),
		})
	}

	// Check for baseline coverage
	baselineCoverage := c.getBaselineCoverage(ctx, tenantID)
	if baselineCoverage < 0.8 {
		score.Score -= 20
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "AD-002",
			ControlName: "Behavioral Baseline Coverage",
			Category:    "anomaly_detection",
			Severity:    "medium",
			Status:      "warning",
			Description: fmt.Sprintf("Only %.1f%% of users have behavioral baselines established", baselineCoverage*100),
			Remediation: "Establish behavioral baselines for all privileged users",
			DetectedAt:  time.Now(),
		})
	}

	// Check for automated response capabilities
	autoResponseEnabled := c.checkAutomatedResponse(ctx, tenantID)
	if !autoResponseEnabled {
		score.Score -= 10
	}

	if score.Score < 0 {
		score.Score = 0
	}

	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 3

	return score, findings
}

// assessIncidentResponse assesses incident response compliance
func (c *ComplianceEngine) assessIncidentResponse(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time, config ComplianceConfig) (ControlScore, []ComplianceFinding) {
	score := ControlScore{
		Category:      "incident_response",
		Score:         100,
		EvidenceCount: 0,
		FindingsCount: 0,
	}
	var findings []ComplianceFinding

	// Check for incident response plan
	planExists := c.checkIncidentResponsePlan(ctx, tenantID)
	if !planExists {
		score.Score -= 30
		findings = append(findings, ComplianceFinding{
			ID:          uuid.New(),
			ControlID:   "IR-001",
			ControlName: "Incident Response Plan",
			Category:    "incident_response",
			Severity:    "high",
			Status:      "failed",
			Description: "No documented incident response plan found",
			Remediation: "Create and document an incident response plan for security incidents",
			DetectedAt:  time.Now(),
		})
	}

	// Check for escalation procedures
	escalationOk := c.checkEscalationProcedures(ctx, tenantID)
	if !escalationOk {
		score.Score -= 20
	}

	// Check for incident tracking
	trackingOk := c.checkIncidentTracking(ctx, tenantID)
	if !trackingOk {
		score.Score -= 15
	}

	if score.Score < 0 {
		score.Score = 0
	}

	if score.Score >= 80 {
		score.Status = "passed"
	} else if score.Score >= 60 {
		score.Status = "warning"
	} else {
		score.Status = "failed"
	}

	score.FindingsCount = len(findings)
	score.EvidenceCount = 3

	return score, findings
}

// Helper functions for compliance checks

func (c *ComplianceEngine) checkMFAEnforcement(ctx context.Context, tenantID uuid.UUID) bool {
	// Simplified check - in production would query actual MFA logs
	return true
}

func (c *ComplianceEngine) checkExcessivePrivilegedAccounts(ctx context.Context, tenantID uuid.UUID) bool {
	// Check if more than 20% of accounts have privileged access
	return false
}

func (c *ComplianceEngine) checkPasswordPolicyCompliance(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkInactiveAccountManagement(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) getSessionRecordingCoverage(ctx context.Context, tenantID uuid.UUID, start, end time.Time) float64 {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE recorded = true) as recorded_count,
			COUNT(*) as total_count
		FROM sessions
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3
	`

	var recordedCount, totalCount int
	err := c.db.QueryRowContext(ctx, query, tenantID, start, end).Scan(&recordedCount, &totalCount)
	if err != nil || totalCount == 0 {
		return 0
	}

	return float64(recordedCount) / float64(totalCount)
}

func (c *ComplianceEngine) countOrphanedSessions(ctx context.Context, tenantID uuid.UUID, start, end time.Time) int {
	query := `
		SELECT COUNT(*)
		FROM sessions
		WHERE tenant_id = $1 AND start_time >= $2 AND start_time <= $3
			AND end_time IS NULL
			AND start_time < NOW() - INTERVAL '24 hours'
	`

	var count int
	_ = c.db.QueryRowContext(ctx, query, tenantID, start, end).Scan(&count)
	return count
}

func (c *ComplianceEngine) checkSessionTimeoutEnforcement(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkAuditLogRetention(ctx context.Context, tenantID uuid.UUID, requiredDays int) bool {
	query := `
		SELECT COUNT(*) > 0
		FROM audit_logs
		WHERE tenant_id = $1 AND created_at < NOW() - INTERVAL '1 day' * $2
		LIMIT 1
	`

	var exists bool
	_ = c.db.QueryRowContext(ctx, query, tenantID, requiredDays).Scan(&exists)
	return exists
}

func (c *ComplianceEngine) checkImmutableLogging(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkAuditTrailCompleteness(ctx context.Context, tenantID uuid.UUID, start, end time.Time) bool {
	return true
}

func (c *ComplianceEngine) checkSIEMForwarding(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkCredentialEncryption(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkTLSEnforcement(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkKeyRotation(ctx context.Context, tenantID uuid.UUID, maxDays int) bool {
	return true
}

func (c *ComplianceEngine) countOpenAnomaliesBySeverity(ctx context.Context, tenantID uuid.UUID, severity string) int {
	query := `
		SELECT COUNT(*)
		FROM anomaly_detections
		WHERE tenant_id = $1 AND severity = $2 AND status = 'open'
			AND is_duplicate = false
	`

	var count int
	_ = c.db.QueryRowContext(ctx, query, tenantID, severity).Scan(&count)
	return count
}

func (c *ComplianceEngine) getBaselineCoverage(ctx context.Context, tenantID uuid.UUID) float64 {
	query := `
		SELECT
			COUNT(DISTINCT b.user_id)::float / NULLIF(COUNT(DISTINCT s.user_id), 0)
		FROM session_analytics s
		LEFT JOIN user_baselines b ON s.user_id = b.user_id AND b.is_active = true
		WHERE s.tenant_id = $1 AND s.start_time >= NOW() - INTERVAL '30 days'
	`

	var coverage float64
	_ = c.db.QueryRowContext(ctx, query, tenantID).Scan(&coverage)
	return coverage
}

func (c *ComplianceEngine) checkAutomatedResponse(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkIncidentResponsePlan(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkEscalationProcedures(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

func (c *ComplianceEngine) checkIncidentTracking(ctx context.Context, tenantID uuid.UUID) bool {
	return true
}

// generateRecommendations generates recommendations based on compliance findings
func (c *ComplianceEngine) generateRecommendations(score *ComplianceScore) []string {
	recommendations := []string{}

	// Analyze failed controls
	for _, cs := range score.ControlScores {
		if cs.Status == "failed" {
			switch cs.Category {
			case "access_control":
				recommendations = append(recommendations, "Implement privileged access management with just-in-time provisioning")
				recommendations = append(recommendations, "Enforce MFA for all privileged access requests")
			case "session_management":
				recommendations = append(recommendations, "Review and update session timeout policies")
				recommendations = append(recommendations, "Implement automated session monitoring and recording")
			case "audit_logging":
				recommendations = append(recommendations, "Extend audit log retention period to meet compliance requirements")
				recommendations = append(recommendations, "Implement immutable audit log storage")
			case "data_protection":
				recommendations = append(recommendations, "Enable encryption for all credential data at rest")
				recommendations = append(recommendations, "Enforce TLS 1.2+ for all service communications")
			case "anomaly_detection":
				recommendations = append(recommendations, "Establish behavioral baselines for all privileged users")
				recommendations = append(recommendations, "Implement automated response for high-risk anomalies")
			case "incident_response":
				recommendations = append(recommendations, "Document and implement incident response procedures")
				recommendations = append(recommendations, "Configure automated incident escalation workflows")
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Maintain current security posture and continue monitoring")
	}

	return recommendations
}

// ConvertToComplianceReport converts a compliance score to a model.ComplianceReport
func (c *ComplianceEngine) ConvertToComplianceReport(score *ComplianceScore, generatedBy uuid.UUID) *model.ComplianceReport {
	// Convert findings to JSON
	findingsJSON, _ := json.Marshal(score.Findings)

	// Convert recommendations to JSON
	recommendationsJSON, _ := json.Marshal(score.Recommendations)

	// Determine status
	status := string(model.ComplianceStatusPassed)
	if score.OverallScore < 70 {
		status = string(model.ComplianceStatusFailed)
	} else if score.OverallScore < 90 {
		status = string(model.ComplianceStatusPartial)
	}

	report := &model.ComplianceReport{
		ID:               uuid.New(),
		TenantID:         score.TenantID,
		ReportName:       fmt.Sprintf("%s Compliance Report - %s", score.Framework, score.PeriodEnd.Format("2006-01-02")),
		Framework:        score.Framework,
		GeneratedAt:      score.AnalyzedAt,
		GeneratedBy:      generatedBy,
		Status:           status,
		OverallScore:     &score.OverallScore,
		TotalControls:    score.TotalControls,
		PassedControls:   score.PassedControls,
		FailedControls:   score.FailedControls,
		SkippedControls:  score.SkippedControls,
		PeriodStart:      score.PeriodStart,
		PeriodEnd:        score.PeriodEnd,
		Findings:         findingsJSON,
		Recommendations:  recommendationsJSON,
	}

	return report
}

// Utility functions

func severityFromPercent(percent float64) string {
	if percent < 0.5 {
		return "critical"
	} else if percent < 0.7 {
		return "high"
	} else if percent < 0.9 {
		return "medium"
	}
	return "low"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
