package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SessionAnalytics represents aggregated session metrics for a time period
type SessionAnalytics struct {
	ID                    uuid.UUID  `db:"id" json:"id"`
	TenantID              uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Date                  time.Time `db:"date" json:"date"`
	Hour                  int       `db:"hour" json:"hour"`
	TotalSessions         int       `db:"total_sessions" json:"total_sessions"`
	ActiveSessions        int       `db:"active_sessions" json:"active_sessions"`
	CompletedSessions     int       `db:"completed_sessions" json:"completed_sessions"`
	TerminatedSessions    int       `db:"terminated_sessions" json:"terminated_sessions"`
	FailedSessions        int       `db:"failed_sessions" json:"failed_sessions"`
	SSHSessions           int       `db:"ssh_sessions" json:"ssh_sessions"`
	RDPSessions           int       `db:"rdp_sessions" json:"rdp_sessions"`
	DatabaseSessions      int       `db:"database_sessions" json:"database_sessions"`
	KubernetesSessions    int       `db:"kubernetes_sessions" json:"kubernetes_sessions"`
	WebSessions           int       `db:"web_sessions" json:"web_sessions"`
	AvgDurationSeconds    *float64  `db:"avg_duration_seconds" json:"avg_duration_seconds"`
	MinDurationSeconds    *int      `db:"min_duration_seconds" json:"min_duration_seconds"`
	MaxDurationSeconds    *int      `db:"max_duration_seconds" json:"max_duration_seconds"`
	TotalDurationSeconds  int64     `db:"total_duration_seconds" json:"total_duration_seconds"`
	PeakConcurrentSessions int      `db:"peak_concurrent_sessions" json:"peak_concurrent_sessions"`
	PeakConcurrentTime    *time.Time `db:"peak_concurrent_time" json:"peak_concurrent_time"`
	UniqueUsers           int       `db:"unique_users" json:"unique_users"`
	UniqueTargets         int       `db:"unique_targets" json:"unique_targets"`
	TotalRecordings       int64     `db:"total_recordings" json:"total_recordings"`
	RecordingSizeBytes    int64     `db:"recording_size_bytes" json:"recording_size_bytes"`
	RecordingDurationSeconds int64  `db:"recording_duration_seconds" json:"recording_duration_seconds"`
	Metadata              []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

// UserActivity represents individual user activity patterns
type UserActivity struct {
	ID                 uuid.UUID  `db:"id" json:"id"`
	TenantID           uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	UserID             uuid.UUID  `db:"user_id" json:"user_id"`
	Date               time.Time `db:"date" json:"date"`
	Hour               int       `db:"hour" json:"hour"`
	SessionsInitiated  int       `db:"sessions_initiated" json:"sessions_initiated"`
	SessionsCompleted  int       `db:"sessions_completed" json:"sessions_completed"`
	CommandsExecuted   int       `db:"commands_executed" json:"commands_executed"`
	TargetsAccessed    int       `db:"targets_accessed" json:"targets_accessed"`
	TotalSessionSeconds int64    `db:"total_session_seconds" json:"total_session_seconds"`
	ActiveSeconds      int64     `db:"active_seconds" json:"active_seconds"`
	IdleSeconds        int64     `db:"idle_seconds" json:"idle_seconds"`
	FirstAccessTime    *time.Time `db:"first_access_time" json:"first_access_time"`
	LastAccessTime     *time.Time `db:"last_access_time" json:"last_access_time"`
	PeakHour           *int      `db:"peak_hour" json:"peak_hour"`
	CountryCode        *string   `db:"country_code" json:"country_code,omitempty"`
	City               *string   `db:"city" json:"city,omitempty"`
	OffHoursAccess     bool      `db:"off_hours_access" json:"off_hours_access"`
	UnusualAccess      bool      `db:"unusual_access" json:"unusual_access"`
	RiskScore          float64   `db:"risk_score" json:"risk_score"`
	Metadata           []byte    `db:"metadata" json:"metadata,omitempty"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

// CommandFrequency represents command execution tracking
type CommandFrequency struct {
	ID                  int64      `db:"id" json:"id"`
	TenantID            uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Date                time.Time  `db:"date" json:"date"`
	Hour                int        `db:"hour" json:"hour"`
	CommandHash         string     `db:"command_hash" json:"command_hash"`
	CommandPattern      string     `db:"command_pattern" json:"command_pattern"`
	BaseCommand         string     `db:"base_command" json:"base_command"`
	SessionID           uuid.UUID  `db:"session_id" json:"session_id"`
	UserID              uuid.UUID  `db:"user_id" json:"user_id"`
	TargetHost          string     `db:"target_host" json:"target_host"`
	RiskLevel           string     `db:"risk_level" json:"risk_level"`
	IsDangerous         bool       `db:"is_dangerous" json:"is_dangerous"`
	IsBlocked           bool       `db:"is_blocked" json:"is_blocked"`
	ExecutedAt          time.Time  `db:"executed_at" json:"executed_at"`
	ExitCode            *int       `db:"exit_code" json:"exit_code,omitempty"`
	ExecutionDurationMs *int       `db:"execution_duration_ms" json:"execution_duration_ms,omitempty"`
	Metadata            []byte     `db:"metadata" json:"metadata,omitempty"`
}

// ComplianceReport represents a framework compliance evaluation
type ComplianceReport struct {
	ID                  uuid.UUID  `db:"id" json:"id"`
	TenantID            uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ReportName          string     `db:"report_name" json:"report_name"`
	Framework           string     `db:"framework" json:"framework"`
	Version             string     `db:"version" json:"version"`
	GeneratedAt         time.Time  `db:"generated_at" json:"generated_at"`
	GeneratedBy         uuid.UUID  `db:"generated_by" json:"generated_by"`
	Status              string     `db:"status" json:"status"`
	OverallScore        *float64   `db:"overall_score" json:"overall_score"`
	TotalControls       int        `db:"total_controls" json:"total_controls"`
	PassedControls      int        `db:"passed_controls" json:"passed_controls"`
	FailedControls      int        `db:"failed_controls" json:"failed_controls"`
	SkippedControls     int        `db:"skipped_controls" json:"skipped_controls"`
	PeriodStart         time.Time  `db:"period_start" json:"period_start"`
	PeriodEnd           time.Time  `db:"period_end" json:"period_end"`
	Summary             *string    `db:"summary" json:"summary,omitempty"`
	Findings            []byte     `db:"findings" json:"findings,omitempty"`
	Recommendations     []byte     `db:"recommendations" json:"recommendations,omitempty"`
	Metadata            []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
}

// ComplianceControlEvaluation represents individual control evaluation results
type ComplianceControlEvaluation struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	ReportID          uuid.UUID  `db:"report_id" json:"report_id"`
	TenantID          uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID         string     `db:"control_id" json:"control_id"`
	ControlName       string     `db:"control_name" json:"control_name"`
	ControlCategory   *string    `db:"control_category" json:"control_category,omitempty"`
	Status            string     `db:"status" json:"status"`
	Score             *float64   `db:"score" json:"score,omitempty"`
	EvidenceCount     int        `db:"evidence_count" json:"evidence_count"`
	EvidenceURLs      []string   `db:"evidence_urls" json:"evidence_urls,omitempty"`
	Findings          *string    `db:"findings" json:"findings,omitempty"`
	RemediationSteps  []string   `db:"remediation_steps" json:"remediation_steps,omitempty"`
	Metadata          []byte     `db:"metadata" json:"metadata,omitempty"`
	EvaluatedAt       time.Time  `db:"evaluated_at" json:"evaluated_at"`
}

// ComplianceException represents approved exceptions to controls
type ComplianceException struct {
	ID                   uuid.UUID  `db:"id" json:"id"`
	TenantID             uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ControlID            string     `db:"control_id" json:"control_id"`
	ControlName          string     `db:"control_name" json:"control_name"`
	Framework            string     `db:"framework" json:"framework"`
	Status               string     `db:"status" json:"status"`
	RiskLevel            string     `db:"risk_level" json:"risk_level"`
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

// AnomalyDetection represents detected anomalies and alerts
type AnomalyDetection struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	TenantID          uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	AnomalyType       string     `db:"anomaly_type" json:"anomaly_type"`
	UserID            *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	SessionID         *uuid.UUID `db:"session_id" json:"session_id,omitempty"`
	TargetHost        *string    `db:"target_host" json:"target_host,omitempty"`
	Severity          string     `db:"severity" json:"severity"`
	ConfidenceScore   float64    `db:"confidence_score" json:"confidence_score"`
	RiskScore         float64    `db:"risk_score" json:"risk_score"`
	Title             string     `db:"title" json:"title"`
	Description       *string    `db:"description" json:"description,omitempty"`
	Indicators        []byte     `db:"indicators" json:"indicators,omitempty"`
	DetectionMethod   string     `db:"detection_method" json:"detection_method"`
	DetectedAt        time.Time  `db:"detected_at" json:"detected_at"`
	ModelVersion      *string    `db:"model_version" json:"model_version,omitempty"`
	Status            string     `db:"status" json:"status"`
	AssignedTo        *uuid.UUID `db:"assigned_to" json:"assigned_to,omitempty"`
	ResolutionNotes   *string    `db:"resolution_notes" json:"resolution_notes,omitempty"`
	ResolvedAt        *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy        *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
	AutoTriggered     bool       `db:"auto_triggered" json:"auto_triggered"`
	AutoActionTaken   *string    `db:"auto_action_taken" json:"auto_action_taken,omitempty"`
	Metadata          []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
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

// CommandBlacklist represents dangerous commands that can be blocked
type CommandBlacklist struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	TenantID          *uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty"`
	CommandPattern    string     `db:"command_pattern" json:"command_pattern"`
	PatternType       string     `db:"pattern_type" json:"pattern_type"`
	BaseCommand       *string    `db:"base_command" json:"base_command,omitempty"`
	Action            string     `db:"action" json:"action"`
	Severity          string     `db:"severity" json:"severity"`
	AppliesToUsers    []uuid.UUID `db:"applies_to_users" json:"applies_to_users,omitempty"`
	AppliesToGroups   []uuid.UUID `db:"applies_to_groups" json:"applies_to_groups,omitempty"`
	AppliesToTargets  []string   `db:"applies_to_targets" json:"applies_to_targets,omitempty"`
	AllowOverride     bool       `db:"allow_override" json:"allow_override"`
	OverrideRoles     []string   `db:"override_roles" json:"override_roles,omitempty"`
	Reason            string     `db:"reason" json:"reason"`
	RiskCategory      *string    `db:"risk_category" json:"risk_category,omitempty"`
	Enabled           bool       `db:"enabled" json:"enabled"`
	CreatedBy         uuid.UUID  `db:"created_by" json:"created_by"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

// SSHKeyAnalytics represents SSH key usage patterns
type SSHKeyAnalytics struct {
	ID                      uuid.UUID  `db:"id" json:"id"`
	TenantID                uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	SSHKeyID                uuid.UUID  `db:"ssh_key_id" json:"ssh_key_id"`
	Date                    time.Time `db:"date" json:"date"`
	UsageCount              int        `db:"usage_count" json:"usage_count"`
	UniqueUsers             int        `db:"unique_users" json:"unique_users"`
	UniqueTargets           int        `db:"unique_targets" json:"unique_targets"`
	FirstUseTime            *time.Time `db:"first_use_time" json:"first_use_time"`
	LastUseTime             *time.Time `db:"last_use_time" json:"last_use_time"`
	AvgSessionDurationSeconds *float64 `db:"avg_session_duration_seconds" json:"avg_session_duration_seconds"`
	OffHoursUsage           int        `db:"off_hours_usage" json:"off_hours_usage"`
	UnusualSourceUsage      int        `db:"unusual_source_usage" json:"unusual_source_usage"`
	FailedAttempts          int        `db:"failed_attempts" json:"failed_attempts"`
	Metadata                []byte     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt               time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time  `db:"updated_at" json:"updated_at"`
}

// Enums

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusEnded      SessionStatus = "ended"
	SessionStatusTerminated SessionStatus = "terminated"
	SessionStatusFailed     SessionStatus = "failed"
)

// ComplianceFramework represents supported compliance frameworks
type ComplianceFramework string

const (
	FrameworkSOC2      ComplianceFramework = "SOC2"
	FrameworkISO27001  ComplianceFramework = "ISO27001"
	FrameworkPCIDSS    ComplianceFramework = "PCI-DSS"
	FrameworkHIPAA     ComplianceFramework = "HIPAA"
	FrameworkNIST      ComplianceFramework = "NIST-800-53"
	FrameworkGDPR      ComplianceFramework = "GDPR"
	FrameworkCustom    ComplianceFramework = "CUSTOM"
)

// ComplianceStatus represents the status of a compliance evaluation
type ComplianceStatus string

const (
	ComplianceStatusPending ComplianceStatus = "pending"
	ComplianceStatusPassed  ComplianceStatus = "passed"
	ComplianceStatusFailed  ComplianceStatus = "failed"
	ComplianceStatusPartial ComplianceStatus = "partial"
)

// AnomalyType represents types of anomalies
type AnomalyType string

const (
	AnomalyTypeBehavioral  AnomalyType = "behavioral"
	AnomalyTypeTemporal    AnomalyType = "temporal"
	AnomalyTypeSpatial     AnomalyType = "spatial"
	AnomalyTypePattern     AnomalyType = "pattern"
	AnomalyTypeVolumetric  AnomalyType = "volumetric"
	AnomalyTypeRansomware  AnomalyType = "ransomware"
)

// RiskLevel represents risk levels
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// AnomalyStatus represents the status of an anomaly
type AnomalyStatus string

const (
	AnomalyStatusOpen          AnomalyStatus = "open"
	AnomalyStatusInvestigating AnomalyStatus = "investigating"
	AnomalyStatusResolved      AnomalyStatus = "resolved"
	AnomalyStatusFalsePositive AnomalyStatus = "false_positive"
	AnomalyStatusIgnored       AnomalyStatus = "ignored"
)

// CommandPatternType represents pattern matching types
type CommandPatternType string

const (
	PatternTypeExact CommandPatternType = "exact"
	PatternTypeRegex CommandPatternType = "regex"
	PatternTypeGlob  CommandPatternType = "glob"
)

// CommandAction represents actions to take for commands
type CommandAction string

const (
	CommandActionBlock CommandAction = "block"
	CommandActionWarn  CommandAction = "warn"
	CommandActionAllow CommandAction = "allow"
	CommandActionAudit CommandAction = "audit"
)

// Filter and Query Types

// SessionAnalyticsFilter filters session analytics queries
type SessionAnalyticsFilter struct {
	TenantID      *uuid.UUID
	DateFrom      *time.Time
	DateTo        *time.Time
	Hour          *int
	SessionType   *string
	Granularity   string // hour, day, week, month
}

// UserActivityFilter filters user activity queries
type UserActivityFilter struct {
	TenantID      *uuid.UUID
	UserID        *uuid.UUID
	DateFrom      *time.Time
	DateTo        *time.Time
	MinRiskScore  *float64
	OffHoursOnly  *bool
	UnusualOnly   *bool
}

// CommandFrequencyFilter filters command frequency queries
type CommandFrequencyFilter struct {
	TenantID      *uuid.UUID
	UserID        *uuid.UUID
	DateFrom      *time.Time
	DateTo        *time.Time
	BaseCommand   *string
	RiskLevel     *string
	IsDangerous   *bool
}

// ComplianceFilter filters compliance report queries
type ComplianceFilter struct {
	TenantID   *uuid.UUID
	Framework  *string
	Status     *string
	DateFrom   *time.Time
	DateTo     *time.Time
}

// AnomalyFilter filters anomaly queries
type AnomalyFilter struct {
	TenantID    *uuid.UUID
	UserID      *uuid.UUID
	AnomalyType *string
	Severity    *string
	Status      *string
	DateFrom    *time.Time
	DateTo      *time.Time
}

// Aggregated Metrics Types

// DashboardMetrics represents metrics for the analytics dashboard
type DashboardMetrics struct {
	SessionMetrics    *SessionSummary    `json:"session_metrics"`
	UserActivity      *UserActivitySummary `json:"user_activity"`
	CommandMetrics    *CommandSummary    `json:"command_metrics"`
	ComplianceStatus  *ComplianceSummary `json:"compliance_status"`
	RecentAnomalies   []AnomalyDetection `json:"recent_anomalies"`
	Timestamp         time.Time          `json:"timestamp"`
}

// SessionSummary summarizes session metrics
type SessionSummary struct {
	TotalSessions      int     `json:"total_sessions"`
	ActiveSessions     int     `json:"active_sessions"`
	AvgDuration        float64 `json:"avg_duration"`
	PeakConcurrent     int     `json:"peak_concurrent"`
	SessionsByType     map[string]int `json:"sessions_by_type"`
	Trend              []DataPoint `json:"trend"`
}

// UserActivitySummary summarizes user activity
type UserActivitySummary struct {
	ActiveUsers        int     `json:"active_users"`
	TotalCommands      int     `json:"total_commands"`
	HighRiskUsers      int     `json:"high_risk_users"`
	OffHoursAccess     int     `json:"off_hours_access"`
}

// CommandSummary summarizes command metrics
type CommandSummary struct {
	TotalCommands      int               `json:"total_commands"`
	HighRiskCommands   int               `json:"high_risk_commands"`
	BlockedCommands    int               `json:"blocked_commands"`
	TopCommands        []CommandRank     `json:"top_commands"`
}

// CommandRank represents a command ranking
type CommandRank struct {
	Command    string `json:"command"`
	Count      int    `json:"count"`
	RiskLevel  string `json:"risk_level"`
}

// ComplianceSummary summarizes compliance status
type ComplianceSummary struct {
	OverallScore       float64                      `json:"overall_score"`
	PassedControls     int                          `json:"passed_controls"`
	FailedControls     int                          `json:"failed_controls"`
	OpenExceptions     int                          `json:"open_exceptions"`
	Frameworks         map[string]FrameworkStatus   `json:"frameworks"`
}

// FrameworkStatus represents status of a framework
type FrameworkStatus struct {
	Score          float64 `json:"score"`
	Status         string  `json:"status"`
	LastEvaluated  time.Time `json:"last_evaluated"`
}

// DataPoint represents a time-series data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}

// Repository interface defines database operations
type Repository interface {
	// Session Analytics
	CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error
	UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error
	GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error)
	ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error)

	// User Activity
	CreateUserActivity(ctx context.Context, activity *UserActivity) error
	UpdateUserActivity(ctx context.Context, activity *UserActivity) error
	GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error)
	ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error)

	// Command Frequency
	RecordCommand(ctx context.Context, cmd *CommandFrequency) error
	ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error)
	GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error)

	// Compliance
	CreateComplianceReport(ctx context.Context, report *ComplianceReport) error
	GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)
	ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error)
	CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error
	ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error)
	CreateComplianceException(ctx context.Context, exception *ComplianceException) error
	ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error)

	// Anomaly Detection
	CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error
	GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error)
	ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error)
	UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error

	// Ransomware Events
	CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error
	GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error)
	ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error)
	UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error

	// Command Blacklist
	CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error
	GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error)
	ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error)
	UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error
	DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error
	FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error)

	// SSH Key Analytics
	CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error
	UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error
	GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error)
	ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error)

	// Dashboard
	GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error)
	GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error)
}

// Cache interface defines caching operations
type Cache interface {
	GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time) (*SessionAnalytics, error)
	SetSessionMetrics(ctx context.Context, analytics *SessionAnalytics, ttl time.Duration) error
	InvalidateSessionMetrics(ctx context.Context, tenantID uuid.UUID) error

	GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time) (*UserActivity, error)
	SetUserActivity(ctx context.Context, activity *UserActivity, ttl time.Duration) error
	InvalidateUserActivity(ctx context.Context, tenantID, userID uuid.UUID) error

	GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)
	SetComplianceReport(ctx context.Context, report *ComplianceReport, ttl time.Duration) error
	InvalidateComplianceCache(ctx context.Context, tenantID uuid.UUID) error

	GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (map[string]int, error)
	SetAnomalyStats(ctx context.Context, tenantID uuid.UUID, stats map[string]int, ttl time.Duration) error

	InvalidateAll(ctx context.Context, tenantID uuid.UUID) error
}
