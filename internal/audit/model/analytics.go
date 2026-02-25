// Package model provides domain models for audit analytics
package model

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Compliance Report Models
// =============================================================================

// ComplianceReport represents a framework compliance evaluation
type ComplianceReport struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	TenantID      uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ReportName    string     `db:"report_name" json:"report_name"`
	Framework     string     `db:"framework" json:"framework"`
	Version       string     `db:"version" json:"version,omitempty"`
	GeneratedAt   time.Time  `db:"generated_at" json:"generated_at"`
	GeneratedBy   uuid.UUID  `db:"generated_by" json:"generated_by"`
	Status        string     `db:"status" json:"status"` // pending, passed, failed, partial
	OverallScore  *float64   `db:"overall_score" json:"overall_score,omitempty"`
	TotalControls int        `db:"total_controls" json:"total_controls"`
	PassedControls int       `db:"passed_controls" json:"passed_controls"`
	FailedControls int       `db:"failed_controls" json:"failed_controls"`
	SkippedControls int      `db:"skipped_controls" json:"skipped_controls"`
	PeriodStart    time.Time `db:"period_start" json:"period_start"`
	PeriodEnd      time.Time `db:"period_end" json:"period_end"`
	Summary        *string   `db:"summary" json:"summary,omitempty"`
	Findings       []byte    `db:"findings" json:"findings,omitempty"`
	Recommendations []byte   `db:"recommendations" json:"recommendations,omitempty"`
	Metadata       []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

// ComplianceControlEvaluation represents individual control evaluation results
type ComplianceControlEvaluation struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	ReportID         uuid.UUID  `db:"report_id" json:"report_id"`
	TenantID         uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID        string     `db:"control_id" json:"control_id"`
	ControlName      string     `db:"control_name" json:"control_name"`
	ControlCategory  *string    `db:"control_category" json:"control_category,omitempty"`
	Status           string     `db:"status" json:"status"` // passed, failed, skipped, not_applicable
	Score            *float64   `db:"score" json:"score,omitempty"`
	EvidenceCount    int        `db:"evidence_count" json:"evidence_count"`
	EvidenceURLs     []string   `db:"evidence_urls" json:"evidence_urls,omitempty"`
	Findings         *string    `db:"findings" json:"findings,omitempty"`
	RemediationSteps []string   `db:"remediation_steps" json:"remediation_steps,omitempty"`
	Metadata         []byte     `db:"metadata" json:"metadata,omitempty"`
	EvaluatedAt      time.Time  `db:"evaluated_at" json:"evaluated_at"`
}

// ComplianceException represents approved exceptions to controls
type ComplianceException struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	TenantID             uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID            string     `db:"control_id" json:"control_id"`
	ControlName          string     `db:"control_name" json:"control_name"`
	Framework            string     `db:"framework" json:"framework"`
	Status               string     `db:"status" json:"status"` // pending, approved, denied, expired, revoked
	RiskLevel            string     `db:"risk_level" json:"risk_level"` // low, medium, high, critical
	RequestedBy          uuid.UUID  `db:"requested_by" json:"requested_by"`
	RequestedAt          time.Time  `db:"requested_at" json:"requested_at"`
	ApprovedBy           *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt           *time.Time `db:"approved_at" json:"approved_at,omitempty"`
	ExpiresAt            *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	Justification        string     `db:"justification" json:"justification"`
	BusinessReason       *string    `db:"business_reason" json:"business_reason,omitempty"`
	CompensatingControls []string   `db:"compensating_controls" json:"compensating_controls,omitempty"`
	RiskAcceptedBy       *uuid.UUID `db:"risk_accepted_by" json:"risk_accepted_by,omitempty"`
	RiskAcceptedAt       *time.Time `db:"risk_accepted_at" json:"risk_accepted_at,omitempty"`
	ReviewDate           *time.Time `db:"review_date" json:"review_date,omitempty"`
	ReviewNotes          *string    `db:"review_notes" json:"review_notes,omitempty"`
	Metadata             []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}

// =============================================================================
// Anomaly Detection Models
// =============================================================================

// AnomalyDetection represents detected anomalies and alerts
type AnomalyDetection struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	AnomalyType     string     `db:"anomaly_type" json:"anomaly_type"` // behavioral, temporal, spatial, pattern, volumetric, ransomware
	UserID          *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	SessionID       *uuid.UUID `db:"session_id" json:"session_id,omitempty"`
	TargetHost      *string    `db:"target_host" json:"target_host,omitempty"`
	Severity        string     `db:"severity" json:"severity"` // low, medium, high, critical
	ConfidenceScore float64    `db:"confidence_score" json:"confidence_score"` // 0-100
	RiskScore       float64    `db:"risk_score" json:"risk_score"` // 0-100
	Title           string     `db:"title" json:"title"`
	Description     *string    `db:"description" json:"description,omitempty"`
	Indicators      []byte     `db:"indicators" json:"indicators,omitempty"`
	DetectionMethod string     `db:"detection_method" json:"detection_method"`
	DetectedAt      time.Time  `db:"detected_at" json:"detected_at"`
	ModelVersion    *string    `db:"model_version" json:"model_version,omitempty"`
	Status          string     `db:"status" json:"status"` // open, investigating, resolved, false_positive, ignored
	AssignedTo      *uuid.UUID `db:"assigned_to" json:"assigned_to,omitempty"`
	ResolutionNotes *string    `db:"resolution_notes" json:"resolution_notes,omitempty"`
	ResolvedAt      *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy      *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
	AutoTriggered   bool       `db:"auto_triggered" json:"auto_triggered"`
	AutoActionTaken *string    `db:"auto_action_taken" json:"auto_action_taken,omitempty"`
	// Deduplication fields
	CorrelationID     *uuid.UUID `db:"correlation_id" json:"correlation_id,omitempty"`
	CorrelationKey    *string    `db:"correlation_key" json:"correlation_key,omitempty"`
	DuplicateCount    int        `db:"duplicate_count" json:"duplicate_count"`
	IsDuplicate       bool       `db:"is_duplicate" json:"is_duplicate"`
	FirstDetectionID  *uuid.UUID `db:"first_detection_id" json:"first_detection_id,omitempty"`
	MergedIntoID      *uuid.UUID `db:"merged_into_id" json:"merged_into_id,omitempty"`
	Metadata        []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// RansomwareEvent represents high-priority ransomware detection events
type RansomwareEvent struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	TenantID             uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	DetectionID          uuid.UUID  `db:"detection_id" json:"detection_id"`
	EncryptionActivity   bool       `db:"encryption_activity" json:"encryption_activity"`
	MassFileModification bool       `db:"mass_file_modification" json:"mass_file_modification"`
	SuspiciousProcesses  []string   `db:"suspicious_processes" json:"suspicious_processes"`
	AffectedPaths        []string   `db:"affected_paths" json:"affected_paths"`
	FilesAffected        int        `db:"files_affected" json:"files_affected"`
	SystemsAffected      int        `db:"systems_affected" json:"systems_affected"`
	DataExfiltrated      bool       `db:"data_exfiltrated" json:"data_exfiltrated"`
	EmergencyTriggered   bool       `db:"emergency_triggered" json:"emergency_triggered"`
	SessionsTerminated   int        `db:"sessions_terminated" json:"sessions_terminated"`
	CredentialsRevoked   int        `db:"credentials_revoked" json:"credentials_revoked"`
	ContainmentStatus    *string    `db:"containment_status" json:"containment_status,omitempty"`
	RecoveryStatus       *string    `db:"recovery_status" json:"recovery_status,omitempty"`
	RawIndicators        []byte     `db:"raw_indicators" json:"raw_indicators,omitempty"`
	Metadata             []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updated_at"`
}

// =============================================================================
// SSH Key Analytics Models
// =============================================================================

// SSHKeyAnalytics represents SSH key usage patterns
type SSHKeyAnalytics struct {
	ID                         uuid.UUID  `db:"id" json:"id"`
	TenantID                   uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	SSHKeyID                   uuid.UUID  `db:"ssh_key_id" json:"ssh_key_id"`
	Date                       time.Time `db:"date" json:"date"`
	UsageCount                 int       `db:"usage_count" json:"usage_count"`
	UniqueUsers                int       `db:"unique_users" json:"unique_users"`
	UniqueTargets              int       `db:"unique_targets" json:"unique_targets"`
	FirstUseTime               *time.Time `db:"first_use_time" json:"first_use_time,omitempty"`
	LastUseTime                *time.Time `db:"last_use_time" json:"last_use_time,omitempty"`
	AvgSessionDurationSeconds  *float64   `db:"avg_session_duration_seconds" json:"avg_session_duration_seconds,omitempty"`
	OffHoursUsage              int       `db:"off_hours_usage" json:"off_hours_usage"`
	UnusualSourceUsage         int       `db:"unusual_source_usage" json:"unusual_source_usage"`
	FailedAttempts             int       `db:"failed_attempts" json:"failed_attempts"`
	Metadata                   []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt                  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at" json:"updated_at"`
}

// =============================================================================
// Command Blacklist Models
// =============================================================================

// CommandBlacklist represents dangerous commands that can be blocked
type CommandBlacklist struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	TenantID         *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"` // NULL for global rules
	CommandPattern   string     `db:"command_pattern" json:"command_pattern"`
	PatternType      string     `db:"pattern_type" json:"pattern_type"` // exact, regex, glob
	BaseCommand      *string    `db:"base_command" json:"base_command,omitempty"`
	Action           string     `db:"action" json:"action"` // block, warn, allow, audit
	Severity         string     `db:"severity" json:"severity"` // low, medium, high, critical
	AppliesToUsers   []uuid.UUID `db:"applies_to_users" json:"applies_to_users,omitempty"`
	AppliesToGroups  []uuid.UUID `db:"applies_to_groups" json:"applies_to_groups,omitempty"`
	AppliesToTargets []string   `db:"applies_to_targets" json:"applies_to_targets,omitempty"`
	AllowOverride    bool       `db:"allow_override" json:"allow_override"`
	OverrideRoles    []string   `db:"override_roles" json:"override_roles,omitempty"`
	Reason           string     `db:"reason" json:"reason"`
	RiskCategory     *string    `db:"risk_category" json:"risk_category,omitempty"`
	Enabled          bool       `db:"enabled" json:"enabled"`
	CreatedBy        uuid.UUID  `db:"created_by" json:"created_by"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updated_at"`
}

// =============================================================================
// Enums and Constants
// =============================================================================

// Compliance Framework values
const (
	FrameworkSOC2     = "SOC2"
	FrameworkISO27001 = "ISO27001"
	FrameworkPCIDSS   = "PCI-DSS"
	FrameworkHIPAA    = "HIPAA"
	FrameworkNIST     = "NIST-800-53"
	FrameworkGDPR     = "GDPR"
	FrameworkCustom   = "CUSTOM"
)

// Compliance Status values
const (
	ComplianceStatusPending ComplianceStatus = "pending"
	ComplianceStatusPassed  ComplianceStatus = "passed"
	ComplianceStatusFailed  ComplianceStatus = "failed"
	ComplianceStatusPartial ComplianceStatus = "partial"
)

type ComplianceStatus string

// Exception Status values
const (
	ExceptionStatusPending ExceptionStatus = "pending"
	ExceptionStatusApproved ExceptionStatus = "approved"
	ExceptionStatusDenied   ExceptionStatus = "denied"
	ExceptionStatusExpired  ExceptionStatus = "expired"
	ExceptionStatusRevoked  ExceptionStatus = "revoked"
)

type ExceptionStatus string

// Anomaly Type values
const (
	AnomalyTypeBehavioral AnomalyType = "behavioral"
	AnomalyTypeTemporal   AnomalyType = "temporal"
	AnomalyTypeSpatial    AnomalyType = "spatial"
	AnomalyTypePattern    AnomalyType = "pattern"
	AnomalyTypeVolumetric AnomalyType = "volumetric"
	AnomalyTypeRansomware AnomalyType = "ransomware"
)

type AnomalyType string

// Severity Level values
const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Severity string

// Anomaly Status values
const (
	AnomalyStatusOpen          AnomalyStatus = "open"
	AnomalyStatusInvestigating AnomalyStatus = "investigating"
	AnomalyStatusResolved      AnomalyStatus = "resolved"
	AnomalyStatusFalsePositive AnomalyStatus = "false_positive"
	AnomalyStatusIgnored       AnomalyStatus = "ignored"
)

type AnomalyStatus string

// Pattern Type values
const (
	PatternTypeExact PatternType = "exact"
	PatternTypeRegex PatternType = "regex"
	PatternTypeGlob  PatternType = "glob"
)

type PatternType string

// Command Action values
const (
	CommandActionBlock CommandAction = "block"
	CommandActionWarn  CommandAction = "warn"
	CommandActionAllow CommandAction = "allow"
	CommandActionAudit CommandAction = "audit"
)

type CommandAction string

// =============================================================================
// Filter Types
// =============================================================================

// ComplianceReportFilter filters compliance report queries
type ComplianceReportFilter struct {
	TenantID  *uuid.UUID
	Framework *string
	Status    *string
	DateFrom  *time.Time
	DateTo    *time.Time
}

// ComplianceExceptionFilter filters compliance exception queries
type ComplianceExceptionFilter struct {
	TenantID     *uuid.UUID
	ControlID    *string
	Framework    *string
	Status       *string
	RiskLevel    *string
	IncludeExpired bool
}

// AnomalyFilter filters anomaly queries
type AnomalyFilter struct {
	TenantID     *uuid.UUID
	UserID       *uuid.UUID
	AnomalyType  *string
	Severity     *string
	Status       *string
	IsDuplicate  *bool
	DateFrom     *time.Time
	DateTo       *time.Time
	AssignedTo   *uuid.UUID
	CorrelationID *uuid.UUID
	Search       string
}

// CommandBlacklistFilter filters command blacklist queries
type CommandBlacklistFilter struct {
	TenantID *uuid.UUID
	Enabled  *bool
	PatternType *string
	Action      *string
	Severity    *string
}

// SSHKeyAnalyticsFilter filters SSH key analytics queries
type SSHKeyAnalyticsFilter struct {
	TenantID  *uuid.UUID
	SSHKeyID  *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
}

// =============================================================================
// Request/Response DTOs
// =============================================================================

// CreateComplianceReportRequest represents a request to create a compliance report
type CreateComplianceReportRequest struct {
	ReportName string     `json:"report_name" binding:"required"`
	Framework string     `json:"framework" binding:"required"`
	Version   string     `json:"version"`
	PeriodStart time.Time `json:"period_start" binding:"required"`
	PeriodEnd   time.Time `json:"period_end" binding:"required"`
}

// CreateComplianceExceptionRequest represents a request to create a compliance exception
type CreateComplianceExceptionRequest struct {
	ControlID            string   `json:"control_id" binding:"required"`
	ControlName          string   `json:"control_name" binding:"required"`
	Framework            string   `json:"framework" binding:"required"`
	RiskLevel            string   `json:"risk_level" binding:"required"`
	Justification        string   `json:"justification" binding:"required"`
	BusinessReason       string   `json:"business_reason"`
	CompensatingControls []string `json:"compensating_controls"`
	ExpiresAt            *time.Time `json:"expires_at"`
}

// UpdateAnomalyRequest represents a request to update an anomaly
type UpdateAnomalyRequest struct {
	Status           *string   `json:"status"`
	AssignedTo       *uuid.UUID `json:"assigned_to"`
	ResolutionNotes  *string   `json:"resolution_notes"`
}

// AcknowledgeAnomalyRequest represents a request to acknowledge an anomaly
type AcknowledgeAnomalyRequest struct {
	Notes string `json:"notes"`
}

// ResolveAnomalyRequest represents a request to resolve an anomaly
type ResolveAnomalyRequest struct {
	Status string `json:"status" binding:"required,oneof=resolved false_positive ignored"`
	Notes  string `json:"notes"`
}

// CreateCommandBlacklistRequest represents a request to create a command blacklist entry
type CreateCommandBlacklistRequest struct {
	CommandPattern    string     `json:"command_pattern" binding:"required"`
	PatternType       string     `json:"pattern_type" binding:"required"`
	BaseCommand       string     `json:"base_command"`
	Action            string     `json:"action" binding:"required"`
	Severity          string     `json:"severity" binding:"required"`
	AppliesToUsers    []uuid.UUID `json:"applies_to_users"`
	AppliesToGroups   []uuid.UUID `json:"applies_to_groups"`
	AppliesToTargets  []string   `json:"applies_to_targets"`
	AllowOverride     bool       `json:"allow_override"`
	OverrideRoles     []string   `json:"override_roles"`
	Reason            string     `json:"reason" binding:"required"`
	RiskCategory      string     `json:"risk_category"`
}

// UpdateCommandBlacklistRequest represents a request to update a command blacklist entry
type UpdateCommandBlacklistRequest struct {
	CommandPattern    *string      `json:"command_pattern"`
	PatternType       *string      `json:"pattern_type"`
	BaseCommand       *string      `json:"base_command"`
	Action            *string      `json:"action"`
	Severity          *string      `json:"severity"`
	AppliesToUsers    []uuid.UUID  `json:"applies_to_users"`
	AppliesToGroups   []uuid.UUID  `json:"applies_to_groups"`
	AppliesToTargets  []string     `json:"applies_to_targets"`
	AllowOverride     *bool        `json:"allow_override"`
	OverrideRoles     []string     `json:"override_roles"`
	Reason            *string      `json:"reason"`
	RiskCategory      *string      `json:"risk_category"`
	Enabled           *bool        `json:"enabled"`
}
