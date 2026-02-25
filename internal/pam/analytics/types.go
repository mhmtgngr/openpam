package analytics

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PeriodType represents the aggregation period type
type PeriodType string

const (
	PeriodHour  PeriodType = "hour"
	PeriodDay   PeriodType = "day"
	PeriodWeek  PeriodType = "week"
	PeriodMonth PeriodType = "month"
)

// Severity represents alert severity
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeThreshold AlertType = "threshold"
	AlertTypeAnomaly   AlertType = "anomaly"
	AlertTypePattern   AlertType = "pattern"
	AlertTypeCompliance AlertType = "compliance"
)

// ReportType represents the type of report
type ReportType string

const (
	ReportTypeSessionSummary ReportType = "session_summary"
	ReportTypeAccessPatterns ReportType = "access_patterns"
	ReportTypeCompliance     ReportType = "compliance"
	ReportTypeUserActivity   ReportType = "user_activity"
	ReportTypeRiskAnalysis   ReportType = "risk_analysis"
)

// ScheduleType represents report schedule type
type ScheduleType string

const (
	ScheduleDaily   ScheduleType = "daily"
	ScheduleWeekly  ScheduleType = "weekly"
	ScheduleMonthly ScheduleType = "monthly"
)

// RiskLevel represents risk level
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// EntityType represents entity type for risk scoring
type EntityType string

const (
	EntityUser       EntityType = "user"
	EntityTarget     EntityType = "target"
	EntityCredential EntityType = "credential"
)

// MetricType represents metric type
type MetricType string

const (
	MetricTypeGauge    MetricType = "gauge"
	MetricTypeCounter  MetricType = "counter"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary  MetricType = "summary"
)

// WidgetType represents widget type
type WidgetType string

const (
	WidgetTypeLineChart  WidgetType = "line_chart"
	WidgetTypeBarChart   WidgetType = "bar_chart"
	WidgetTypePieChart   WidgetType = "pie_chart"
	WidgetTypeStatCard   WidgetType = "stat_card"
	WidgetTypeTable      WidgetType = "table"
	WidgetTypeHeatmap    WidgetType = "heatmap"
	WidgetTypeGauge      WidgetType = "gauge"
)

// DataSource represents data source for widgets
type DataSource string

const (
	DataSourceSessions   DataSource = "sessions"
	DataSourceEvents     DataSource = "events"
	DataSourceCredentials DataSource = "credentials"
	DataSourceUsers      DataSource = "users"
	DataSourceRisks      DataSource = "risks"
)

// SessionAnalytics represents session analytics aggregation
type SessionAnalytics struct {
	ID              uuid.UUID      `db:"id" json:"id"`
	TenantID        uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	PeriodType      PeriodType     `db:"period_type" json:"period_type"`
	PeriodStart     time.Time      `db:"period_start" json:"period_start"`
	PeriodEnd       time.Time      `db:"period_end" json:"period_end"`

	// Session counts
	TotalSessions      int    `db:"total_sessions" json:"total_sessions"`
	ActiveSessions     int    `db:"active_sessions" json:"active_sessions"`
	CompletedSessions  int    `db:"completed_sessions" json:"completed_sessions"`
	FailedSessions     int    `db:"failed_sessions" json:"failed_sessions"`
	TerminatedSessions int    `db:"terminated_sessions" json:"terminated_sessions"`

	// Duration stats (seconds)
	AvgDurationSeconds *int    `db:"avg_duration_seconds" json:"avg_duration_seconds,omitempty"`
	MinDurationSeconds *int    `db:"min_duration_seconds" json:"min_duration_seconds,omitempty"`
	MaxDurationSeconds *int    `db:"max_duration_seconds" json:"max_duration_seconds,omitempty"`
	P50DurationSeconds *int    `db:"p50_duration_seconds" json:"p50_duration_seconds,omitempty"`
	P95DurationSeconds *int    `db:"p95_duration_seconds" json:"p95_duration_seconds,omitempty"`
	P99DurationSeconds *int    `db:"p99_duration_seconds" json:"p99_duration_seconds,omitempty"`

	// User breakdown
	UniqueUsers int `db:"unique_users" json:"unique_users"`

	// Protocol breakdown
	ProtocolBreakdown json.RawMessage `db:"protocol_breakdown" json:"protocol_breakdown,omitempty"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// EventAnalytics represents event analytics aggregation
type EventAnalytics struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	TenantID    uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	PeriodType  PeriodType     `db:"period_type" json:"period_type"`
	PeriodStart time.Time      `db:"period_start" json:"period_start"`
	PeriodEnd   time.Time      `db:"period_end" json:"period_end"`

	// Event counts by outcome
	TotalEvents     int `db:"total_events" json:"total_events"`
	SuccessfulEvents int `db:"successful_events" json:"successful_events"`
	FailedEvents    int `db:"failed_events" json:"failed_events"`
	DeniedEvents    int `db:"denied_events" json:"denied_events"`

	// Event counts by action type
	ActionBreakdown json.RawMessage `db:"action_breakdown" json:"action_breakdown,omitempty"`

	// Top users and resources
	TopUsers     json.RawMessage `db:"top_users" json:"top_users,omitempty"`
	TopResources json.RawMessage `db:"top_resources" json:"top_resources,omitempty"`

	// Failed authentication
	FailedAuthCount  int `db:"failed_auth_count" json:"failed_auth_count"`
	UniqueFailedUsers int `db:"unique_failed_users" json:"unique_failed_users"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Report represents an analytics report
type Report struct {
	ID              uuid.UUID   `db:"id" json:"id"`
	TenantID        uuid.UUID   `db:"tenant_id" json:"tenant_id"`
	Name            string      `db:"name" json:"name"`
	Description     string      `db:"description" json:"description,omitempty"`
	ReportType      ReportType  `db:"report_type" json:"report_type"`

	// Report configuration
	Config json.RawMessage `db:"config" json:"config"`

	// Schedule
	ScheduleEnabled    bool             `db:"schedule_enabled" json:"schedule_enabled"`
	ScheduleType       *ScheduleType    `db:"schedule_type" json:"schedule_type,omitempty"`
	ScheduleDayOfWeek  *int             `db:"schedule_day_of_week" json:"schedule_day_of_week,omitempty"`
	ScheduleDayOfMonth *int             `db:"schedule_day_of_month" json:"schedule_day_of_month,omitempty"`
	ScheduleHour       *int             `db:"schedule_hour" json:"schedule_hour,omitempty"`
	ScheduleTimezone   string           `db:"schedule_timezone" json:"schedule_timezone"`

	// Delivery
	DeliveryMethods json.RawMessage `db:"delivery_methods" json:"delivery_methods,omitempty"`

	// Status
	LastRunAt   *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	NextRunAt   *time.Time `db:"next_run_at" json:"next_run_at,omitempty"`
	LastStatus  *string    `db:"last_status" json:"last_status,omitempty"`
	LastError   *string    `db:"last_error" json:"last_error,omitempty"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	Tags     json.RawMessage `db:"tags" json:"tags,omitempty"`

	// Audit
	CreatedBy uuid.UUID  `db:"created_by" json:"created_by"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Dashboard represents an analytics dashboard
type Dashboard struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	TenantID    uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description,omitempty"`
	IsDefault   bool       `db:"is_default" json:"is_default"`
	IsPublic    bool       `db:"is_public" json:"is_public"`

	// Layout configuration
	Layout         json.RawMessage `db:"layout" json:"layout"`
	GlobalFilters  json.RawMessage `db:"global_filters" json:"global_filters,omitempty"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	Tags     json.RawMessage `db:"tags" json:"tags,omitempty"`

	// Audit
	CreatedBy uuid.UUID  `db:"created_by" json:"created_by"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`

	// Widgets (loaded separately)
	Widgets []Widget `json:"widgets,omitempty"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	DashboardID uuid.UUID  `db:"dashboard_id" json:"dashboard_id"`
	TenantID    uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Name        string     `db:"name" json:"name"`
	WidgetType  WidgetType `db:"widget_type" json:"widget_type"`

	// Position and size
	PositionX int `db:"position_x" json:"position_x"`
	PositionY int `db:"position_y" json:"position_y"`
	Width     int `db:"width" json:"width"`
	Height    int `db:"height" json:"height"`

	// Data source
	DataSource    DataSource      `db:"data_source" json:"data_source"`
	QueryConfig   json.RawMessage `db:"query_config" json:"query_config"`

	// Display configuration
	DisplayConfig json.RawMessage `db:"display_config" json:"display_config"`

	// Refresh settings
	RefreshIntervalSeconds int `db:"refresh_interval_seconds" json:"refresh_interval_seconds"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	// Audit
	CreatedBy uuid.UUID `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Alert represents an analytics alert
type Alert struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	TenantID    uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description,omitempty"`
	AlertType   AlertType  `db:"alert_type" json:"alert_type"`
	Severity    Severity   `db:"severity" json:"severity"`

	// Alert conditions
	Conditions json.RawMessage `db:"conditions" json:"conditions"`

	// Evaluation schedule
	EvaluationIntervalMinutes int `db:"evaluation_interval_minutes" json:"evaluation_interval_minutes"`

	// Notification
	NotificationMethods json.RawMessage `db:"notification_methods" json:"notification_methods"`

	// Status
	Enabled          bool       `db:"enabled" json:"enabled"`
	LastTriggeredAt  *time.Time `db:"last_triggered_at" json:"last_triggered_at,omitempty"`
	LastEvaluatedAt  *time.Time `db:"last_evaluated_at" json:"last_evaluated_at,omitempty"`
	TriggerCount     int        `db:"trigger_count" json:"trigger_count"`

	// Cooldown
	CooldownMinutes int        `db:"cooldown_minutes" json:"cooldown_minutes"`
	NextTriggerAt   *time.Time `db:"next_trigger_at" json:"next_trigger_at,omitempty"`

	// Metadata
	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	Tags     json.RawMessage `db:"tags" json:"tags,omitempty"`

	// Audit
	CreatedBy uuid.UUID  `db:"created_by" json:"created_by"`
	UpdatedBy *uuid.UUID `db:"updated_by" json:"updated_by,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// AlertTrigger represents an alert trigger history
type AlertTrigger struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	AlertID         uuid.UUID  `db:"alert_id" json:"alert_id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	TriggeredAt     time.Time  `db:"triggered_at" json:"triggered_at"`
	Severity        Severity   `db:"severity" json:"severity"`
	TriggerData     json.RawMessage `db:"trigger_data" json:"trigger_data"`

	// Resolution
	ResolvedAt      *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy      *uuid.UUID `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolutionNotes *string    `db:"resolution_notes" json:"resolution_notes,omitempty"`

	// Notifications
	NotificationsSent json.RawMessage `db:"notifications_sent" json:"notifications_sent,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

// Metric represents a metric data point
type Metric struct {
	ID         uuid.UUID      `db:"id" json:"id"`
	TenantID   uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	MetricName string         `db:"metric_name" json:"metric_name"`
	MetricType MetricType     `db:"metric_type" json:"metric_type"`
	Value      float64        `db:"value" json:"value"`
	Labels     json.RawMessage `db:"labels" json:"labels,omitempty"`
	RecordedAt time.Time      `db:"recorded_at" json:"recorded_at"`
	Metadata   json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

// UserActivity represents user activity analytics
type UserActivity struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	UserID          uuid.UUID  `db:"user_id" json:"user_id"`
	PeriodType      PeriodType `db:"period_type" json:"period_type"`
	PeriodStart     time.Time  `db:"period_start" json:"period_start"`

	// Activity counts
	SessionsCreated    int `db:"sessions_created" json:"sessions_created"`
	CredentialsAccessed int `db:"credentials_accessed" json:"credentials_accessed"`
	ApprovalsRequested int `db:"approvals_requested" json:"approvals_requested"`
	ApprovalsGranted   int `db:"approvals_granted" json:"approvals_granted"`

	// Time-based stats
	TotalActiveSeconds int64  `db:"total_active_seconds" json:"total_active_seconds"`
	AvgDailySeconds    *int   `db:"avg_daily_seconds" json:"avg_daily_seconds,omitempty"`

	// Risk indicators
	HighRiskSessions  int     `db:"high_risk_sessions" json:"high_risk_sessions"`
	PolicyViolations  int     `db:"policy_violations" json:"policy_violations"`
	FailedAuthAttempts int    `db:"failed_auth_attempts" json:"failed_auth_attempts"`

	// Unusual activity flags
	IsAnomaly    bool    `db:"is_anomaly" json:"is_anomaly"`
	AnomalyScore *float64 `db:"anomaly_score" json:"anomaly_score,omitempty"`

	// Access patterns
	OffHoursAccess bool      `db:"off_hours_access" json:"off_hours_access"`
	FirstAccessTime *time.Time `db:"first_access_time" json:"first_access_time,omitempty"`
	LastAccessTime  *time.Time `db:"last_access_time" json:"last_access_time,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// RiskScore represents entity risk score
type RiskScore struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	TenantID          uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	EntityType        EntityType `db:"entity_type" json:"entity_type"`
	EntityID          uuid.UUID  `db:"entity_id" json:"entity_id"`
	CalculatedAt      time.Time  `db:"calculated_at" json:"calculated_at"`

	// Risk components (0-100 each)
	AccessFrequencyScore *float64 `db:"access_frequency_score" json:"access_frequency_score,omitempty"`
	TimePatternScore     *float64 `db:"time_pattern_score" json:"time_pattern_score,omitempty"`
	GeographyScore       *float64 `db:"geography_score" json:"geography_score,omitempty"`
	BehaviorDriftScore   *float64 `db:"behavior_drift_score" json:"behavior_drift_score,omitempty"`
	ComplianceScore      *float64 `db:"compliance_score" json:"compliance_score,omitempty"`

	// Overall risk
	OverallRiskScore float64  `db:"overall_risk_score" json:"overall_risk_score"`
	RiskLevel        RiskLevel `db:"risk_level" json:"risk_level"`

	// Contributing factors
	Factors json.RawMessage `db:"factors" json:"factors,omitempty"`

	// Trend
	PreviousScore *float64 `db:"previous_score" json:"previous_score,omitempty"`
	ScoreChange   *float64 `db:"score_change" json:"score_change,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

// FilterOptions represents common filter options for analytics queries
type FilterOptions struct {
	TenantID     uuid.UUID     `json:"tenant_id"`
	StartDate    *time.Time    `json:"start_date,omitempty"`
	EndDate      *time.Time    `json:"end_date,omitempty"`
	PeriodType   *PeriodType   `json:"period_type,omitempty"`
	UserID       *uuid.UUID    `json:"user_id,omitempty"`
	TargetID     *uuid.UUID    `json:"target_id,omitempty"`
	CredentialID *uuid.UUID    `json:"credential_id,omitempty"`
	Protocol     *string       `json:"protocol,omitempty"`
	Outcome      *string       `json:"outcome,omitempty"`
	Action       *string       `json:"action,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Offset       int           `json:"offset,omitempty"`
}

// DashboardFilter represents filter options for dashboard queries
type DashboardFilter struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	IsDefault *bool     `json:"is_default,omitempty"`
	IsPublic  *bool     `json:"is_public,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	Search    string    `json:"search,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// ReportFilter represents filter options for report queries
type ReportFilter struct {
	TenantID   uuid.UUID    `json:"tenant_id"`
	ReportType *ReportType `json:"report_type,omitempty"`
	Search     string      `json:"search,omitempty"`
	Tags       []string    `json:"tags,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
}

// AlertFilter represents filter options for alert queries
type AlertFilter struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	AlertType *AlertType `json:"alert_type,omitempty"`
	Severity  *Severity  `json:"severity,omitempty"`
	Enabled   *bool      `json:"enabled,omitempty"`
	Search    string     `json:"search,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// SessionStats represents session statistics
type SessionStats struct {
	Total      int64     `json:"total"`
	Active     int64     `json:"active"`
	Completed  int64     `json:"completed"`
	Failed     int64     `json:"failed"`
	Terminated int64     `json:"terminated"`
	AvgDuration int      `json:"avg_duration"`
	MaxDuration int      `json:"max_duration"`
	ByProtocol  map[string]int64 `json:"by_protocol"`
	ByUser      map[string]int64 `json:"by_user"`
}

// EventStats represents event statistics
type EventStats struct {
	Total       int64            `json:"total"`
	Successful  int64            `json:"successful"`
	Failed      int64            `json:"failed"`
	Denied      int64            `json:"denied"`
	ByAction    map[string]int64 `json:"by_action"`
	ByUser      map[string]int64 `json:"by_user"`
	ByResource  map[string]int64 `json:"by_resource"`
}

// TopUser represents top user by activity
type TopUser struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	SessionCount int       `json:"session_count"`
	TotalSeconds int64     `json:"total_seconds"`
	LastSeen     time.Time `json:"last_seen"`
}

// AccessPattern represents access pattern data
type AccessPattern struct {
	HourOfDay         int     `json:"hour_of_day"`
	DayOfWeek         int     `json:"day_of_week"`
	SessionCount      int     `json:"session_count"`
	UniqueUsers       int     `json:"unique_users"`
	AvgDuration       int     `json:"avg_duration"`
	IsOutsideBusiness bool    `json:"is_outside_business"`
}

// TimeSeriesDataPoint represents a time series data point
type TimeSeriesDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ComplianceStatus represents compliance status
type ComplianceStatus struct {
	OverallPercentage float64            `json:"overall_percentage"`
	PassedChecks      int                `json:"passed_checks"`
	TotalChecks       int                `json:"total_checks"`
	Violations        []ComplianceViolation `json:"violations,omitempty"`
	ByPolicy          map[string]CompliancePolicyStatus `json:"by_policy,omitempty"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	PolicyID      uuid.UUID `json:"policy_id"`
	PolicyName    string    `json:"policy_name"`
	ViolationType string    `json:"violation_type"`
	Severity      string    `json:"severity"`
	Description   string    `json:"description"`
	Count         int       `json:"count"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

// CompliancePolicyStatus represents status for a specific policy
type CompliancePolicyStatus struct {
	PolicyID          uuid.UUID `json:"policy_id"`
	PolicyName        string    `json:"policy_name"`
	ComplianceRate    float64   `json:"compliance_rate"`
	TotalEvaluations  int       `json:"total_evaluations"`
	PassedEvaluations int       `json:"passed_evaluations"`
}

// CreateReportRequest represents a request to create a report
type CreateReportRequest struct {
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	ReportType      ReportType      `json:"report_type" binding:"required"`
	Config          json.RawMessage `json:"config"`
	ScheduleEnabled bool            `json:"schedule_enabled"`
	ScheduleType    *ScheduleType   `json:"schedule_type"`
	ScheduleDayOfWeek *int          `json:"schedule_day_of_week"`
	ScheduleDayOfMonth *int         `json:"schedule_day_of_month"`
	ScheduleHour    *int            `json:"schedule_hour"`
	DeliveryMethods json.RawMessage `json:"delivery_methods"`
	Metadata        json.RawMessage `json:"metadata"`
	Tags            json.RawMessage `json:"tags"`
}

// UpdateReportRequest represents a request to update a report
type UpdateReportRequest struct {
	Name            *string          `json:"name"`
	Description     *string          `json:"description"`
	Config          json.RawMessage `json:"config"`
	ScheduleEnabled *bool            `json:"schedule_enabled"`
	ScheduleType    *ScheduleType    `json:"schedule_type"`
	ScheduleDayOfWeek *int           `json:"schedule_day_of_week"`
	ScheduleDayOfMonth *int          `json:"schedule_day_of_month"`
	ScheduleHour    *int             `json:"schedule_hour"`
	DeliveryMethods json.RawMessage  `json:"delivery_methods"`
	Metadata        json.RawMessage  `json:"metadata"`
	Tags            json.RawMessage  `json:"tags"`
}

// CreateDashboardRequest represents a request to create a dashboard
type CreateDashboardRequest struct {
	Name           string          `json:"name" binding:"required"`
	Description    string          `json:"description"`
	IsDefault      bool            `json:"is_default"`
	IsPublic       bool            `json:"is_public"`
	Layout         json.RawMessage `json:"layout" binding:"required"`
	GlobalFilters  json.RawMessage `json:"global_filters"`
	Metadata       json.RawMessage `json:"metadata"`
	Tags           json.RawMessage `json:"tags"`
	Widgets        []CreateWidgetRequest `json:"widgets"`
}

// UpdateDashboardRequest represents a request to update a dashboard
type UpdateDashboardRequest struct {
	Name          *string         `json:"name"`
	Description   *string         `json:"description"`
	IsDefault     *bool           `json:"is_default"`
	IsPublic      *bool           `json:"is_public"`
	Layout        json.RawMessage `json:"layout"`
	GlobalFilters json.RawMessage `json:"global_filters"`
	Metadata      json.RawMessage `json:"metadata"`
	Tags          json.RawMessage `json:"tags"`
}

// CreateWidgetRequest represents a request to create a widget
type CreateWidgetRequest struct {
	Name                  string          `json:"name" binding:"required"`
	WidgetType            WidgetType      `json:"widget_type" binding:"required"`
	PositionX             int             `json:"position_x" binding:"required"`
	PositionY             int             `json:"position_y" binding:"required"`
	Width                 int             `json:"width" binding:"required,min=1"`
	Height                int             `json:"height" binding:"required,min=1"`
	DataSource            DataSource      `json:"data_source" binding:"required"`
	QueryConfig           json.RawMessage `json:"query_config" binding:"required"`
	DisplayConfig         json.RawMessage `json:"display_config"`
	RefreshIntervalSeconds *int            `json:"refresh_interval_seconds"`
	Metadata              json.RawMessage `json:"metadata"`
}

// UpdateWidgetRequest represents a request to update a widget
type UpdateWidgetRequest struct {
	Name                  *string         `json:"name"`
	PositionX             *int            `json:"position_x"`
	PositionY             *int            `json:"position_y"`
	Width                 *int            `json:"width"`
	Height                *int            `json:"height"`
	QueryConfig           json.RawMessage `json:"query_config"`
	DisplayConfig         json.RawMessage `json:"display_config"`
	RefreshIntervalSeconds *int            `json:"refresh_interval_seconds"`
	Metadata              json.RawMessage `json:"metadata"`
}

// CreateAlertRequest represents a request to create an alert
type CreateAlertRequest struct {
	Name                       string          `json:"name" binding:"required"`
	Description                string          `json:"description"`
	AlertType                  AlertType       `json:"alert_type" binding:"required"`
	Severity                   Severity        `json:"severity" binding:"required,oneof=low medium high critical"`
	Conditions                 json.RawMessage `json:"conditions" binding:"required"`
	EvaluationIntervalMinutes  int             `json:"evaluation_interval_minutes" binding:"required,min=1"`
	NotificationMethods        json.RawMessage `json:"notification_methods" binding:"required"`
	CooldownMinutes            int             `json:"cooldown_minutes"`
	Metadata                   json.RawMessage `json:"metadata"`
	Tags                       json.RawMessage `json:"tags"`
}

// UpdateAlertRequest represents a request to update an alert
type UpdateAlertRequest struct {
	Name                       *string         `json:"name"`
	Description                *string         `json:"description"`
	Severity                   *Severity       `json:"severity"`
	Conditions                 json.RawMessage `json:"conditions"`
	EvaluationIntervalMinutes  *int            `json:"evaluation_interval_minutes"`
	NotificationMethods        json.RawMessage `json:"notification_methods"`
	Enabled                    *bool           `json:"enabled"`
	CooldownMinutes            *int            `json:"cooldown_minutes"`
	Metadata                   json.RawMessage `json:"metadata"`
	Tags                       json.RawMessage `json:"tags"`
}

// WidgetDataResponse represents widget data response
type WidgetDataResponse struct {
	WidgetID   uuid.UUID          `json:"widget_id"`
	DataSource DataSource         `json:"data_source"`
	Data       interface{}        `json:"data"`
	GeneratedAt time.Time         `json:"generated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// Anomaly Detection Types
// =============================================================================

// AnomalyStatus represents the status of an anomaly
type AnomalyStatus string

const (
	AnomalyStatusOpen          AnomalyStatus = "open"
	AnomalyStatusInvestigating AnomalyStatus = "investigating"
	AnomalyStatusResolved      AnomalyStatus = "resolved"
	AnomalyStatusFalsePositive AnomalyStatus = "false_positive"
	AnomalyStatusIgnored       AnomalyStatus = "ignored"
)

// AnomalyType represents the type of anomaly detected
type AnomalyType string

const (
	AnomalyTypeBehavioral AnomalyType = "behavioral"
	AnomalyTypeTemporal   AnomalyType = "temporal"
	AnomalyTypeSpatial    AnomalyType = "spatial"
	AnomalyTypePattern    AnomalyType = "pattern"
	AnomalyTypeVolumetric AnomalyType = "volumetric"
	AnomalyTypeRansomware AnomalyType = "ransomware"
)

// Anomaly represents a detected security anomaly with full persistence support
type Anomaly struct {
	ID              uuid.UUID      `db:"id" json:"id"`
	TenantID        uuid.UUID      `db:"tenant_id" json:"tenant_id"`
	AnomalyType     AnomalyType    `db:"anomaly_type" json:"anomaly_type"`
	UserID          *uuid.UUID     `db:"user_id" json:"user_id,omitempty"`
	SessionID       *uuid.UUID     `db:"session_id" json:"session_id,omitempty"`
	TargetHost      *string        `db:"target_host" json:"target_host,omitempty"`
	Severity        Severity       `db:"severity" json:"severity"`
	ConfidenceScore float64        `db:"confidence_score" json:"confidence_score"`
	RiskScore       float64        `db:"risk_score" json:"risk_score"`
	Title           string         `db:"title" json:"title"`
	Description     *string        `db:"description" json:"description,omitempty"`
	Indicators      json.RawMessage `db:"indicators" json:"indicators,omitempty"`

	// Detection metadata
	DetectionMethod string     `db:"detection_method" json:"detection_method"`
	DetectedAt      time.Time  `db:"detected_at" json:"detected_at"`
	ModelVersion    *string    `db:"model_version" json:"model_version,omitempty"`

	// Status and resolution
	Status          AnomalyStatus `db:"status" json:"status"`
	AssignedTo      *uuid.UUID    `db:"assigned_to" json:"assigned_to,omitempty"`
	ResolutionNotes *string       `db:"resolution_notes" json:"resolution_notes,omitempty"`
	ResolvedAt      *time.Time    `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy      *uuid.UUID    `db:"resolved_by" json:"resolved_by,omitempty"`

	// Automated response
	AutoTriggered   bool    `db:"auto_triggered" json:"auto_triggered"`
	AutoActionTaken *string `db:"auto_action_taken" json:"auto_action_taken,omitempty"`

	// Deduplication fields
	CorrelationID     *uuid.UUID `db:"correlation_id" json:"correlation_id,omitempty"`
	CorrelationKey    *string    `db:"correlation_key" json:"correlation_key,omitempty"`
	DuplicateCount    int        `db:"duplicate_count" json:"duplicate_count"`
	IsDuplicate       bool       `db:"is_duplicate" json:"is_duplicate"`
	FirstDetectionID  *uuid.UUID `db:"first_detection_id" json:"first_detection_id,omitempty"`
	MergedIntoID      *uuid.UUID `db:"merged_into_id" json:"merged_into_id,omitempty"`

	Metadata    json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// AnomalyFilter represents filter options for anomaly queries
type AnomalyFilter struct {
	TenantID     *uuid.UUID    `json:"tenant_id"`
	UserID       *uuid.UUID    `json:"user_id,omitempty"`
	AnomalyType  *AnomalyType  `json:"anomaly_type,omitempty"`
	Severity     *Severity     `json:"severity,omitempty"`
	Status       *AnomalyStatus `json:"status,omitempty"`
	IsDuplicate  *bool         `json:"is_duplicate,omitempty"`
	DateFrom     *time.Time    `json:"date_from,omitempty"`
	DateTo       *time.Time    `json:"date_to,omitempty"`
	AssignedTo   *uuid.UUID    `json:"assigned_to,omitempty"`
	CorrelationID *uuid.UUID   `json:"correlation_id,omitempty"`
	Search       string        `json:"search,omitempty"`
}

// AnomalyStats represents statistics about anomalies
type AnomalyStats struct {
	Total              int     `json:"total"`
	OpenCount          int     `json:"open_count"`
	InvestigatingCount int     `json:"investigating_count"`
	ResolvedCount      int     `json:"resolved_count"`
	CriticalCount      int     `json:"critical_count"`
	HighCount          int     `json:"high_count"`
	MediumCount        int     `json:"medium_count"`
	LowCount           int     `json:"low_count"`
	TodayCount         int     `json:"today_count"`
	WeekCount          int     `json:"week_count"`
	UniqueCorrelations int     `json:"unique_correlations"`
	TotalDuplicates    int     `json:"total_duplicates"`
	AvgRiskScore       float64 `json:"avg_risk_score"`
}

// CreateAnomalyRequest represents a request to create an anomaly
type CreateAnomalyRequest struct {
	TenantID        uuid.UUID              `json:"tenant_id" binding:"required"`
	AnomalyType     AnomalyType            `json:"anomaly_type" binding:"required"`
	UserID          *uuid.UUID             `json:"user_id"`
	SessionID       *uuid.UUID             `json:"session_id"`
	TargetHost      *string                `json:"target_host"`
	Title           string                 `json:"title" binding:"required"`
	Description     *string                `json:"description"`
	Indicators      map[string]interface{} `json:"indicators"`
	Severity        Severity               `json:"severity" binding:"required,oneof=low medium high critical"`
	ConfidenceScore float64                `json:"confidence_score" binding:"required,min=0,max=100"`
	RiskScore       float64                `json:"risk_score" binding:"required,min=0,max=100"`
	DetectionMethod string                 `json:"detection_method" binding:"required"`
	ModelVersion    *string                `json:"model_version"`
	AutoTriggered   bool                   `json:"auto_triggered"`
	AutoActionTaken *string                `json:"auto_action_taken"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// UpdateAnomalyStatusRequest represents a request to update anomaly status
type UpdateAnomalyStatusRequest struct {
	Status           *AnomalyStatus `json:"status" binding:"omitempty,oneof=open investigating resolved false_positive ignored"`
	AssignedTo       *uuid.UUID     `json:"assigned_to"`
	ResolutionNotes  *string        `json:"resolution_notes"`
}

// AnomalyBulkUpdateRequest represents a request to bulk update anomalies
type AnomalyBulkUpdateRequest struct {
	AnomalyIDs      []uuid.UUID    `json:"anomaly_ids" binding:"required"`
	Status          *AnomalyStatus `json:"status"`
	AssignedTo      *uuid.UUID     `json:"assigned_to"`
	ResolutionNotes *string        `json:"resolution_notes"`
}

// =============================================================================
// Report Persistence Types
// =============================================================================

// ReportSnapshotStatus represents the status of a report snapshot
type ReportSnapshotStatus string

const (
	ReportSnapshotStatusPending   ReportSnapshotStatus = "pending"
	ReportSnapshotStatusCompleted ReportSnapshotStatus = "completed"
	ReportSnapshotStatusFailed    ReportSnapshotStatus = "failed"
	ReportSnapshotStatusExpired   ReportSnapshotStatus = "expired"
)

// ReportJobStatus represents the status of a report generation job
type ReportJobStatus string

const (
	ReportJobStatusQueued     ReportJobStatus = "queued"
	ReportJobStatusProcessing ReportJobStatus = "processing"
	ReportJobStatusCompleted  ReportJobStatus = "completed"
	ReportJobStatusFailed     ReportJobStatus = "failed"
	ReportJobStatusCancelled  ReportJobStatus = "cancelled"
)

// ReportScheduleStatus represents the status of a report schedule
type ReportScheduleStatus string

const (
	ReportScheduleStatusActive   ReportScheduleStatus = "active"
	ReportScheduleStatusPaused   ReportScheduleStatus = "paused"
	ReportScheduleStatusDisabled ReportScheduleStatus = "disabled"
)

// ReportFormat represents the output format for reports
type ReportFormat string

const (
	ReportFormatPDF  ReportFormat = "pdf"
	ReportFormatXLSX ReportFormat = "xlsx"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatJSON ReportFormat = "json"
)

// ReportScheduleType represents the type of schedule
type ReportScheduleType string

const (
	ReportScheduleDaily    ReportScheduleType = "daily"
	ReportScheduleWeekly   ReportScheduleType = "weekly"
	ReportScheduleMonthly  ReportScheduleType = "monthly"
	ReportScheduleQuarterly ReportScheduleType = "quarterly"
	ReportScheduleYearly   ReportScheduleType = "yearly"
	ReportScheduleCustom   ReportScheduleType = "custom"
)

// ReportSnapshot represents a generated compliance report instance
type ReportSnapshot struct {
	ID           uuid.UUID          `db:"id" json:"id"`
	TenantID     uuid.UUID          `db:"tenant_id" json:"tenant_id"`
	ReportID     uuid.UUID          `db:"report_id" json:"report_id"`
	SnapshotName string             `db:"snapshot_name" json:"snapshot_name"`
	Framework    string             `db:"framework" json:"framework"`

	// Generation metadata
	GeneratedAt time.Time `db:"generated_at" json:"generated_at"`
	GeneratedBy uuid.UUID `db:"generated_by" json:"generated_by"`

	// Status
	Status ReportSnapshotStatus `db:"status" json:"status"`

	// File storage
	FileURL       *string `db:"file_url" json:"file_url,omitempty"`
	FileSizeBytes *int64  `db:"file_size_bytes" json:"file_size_bytes,omitempty"`
	FileFormat    *string `db:"file_format" json:"file_format,omitempty"`
	StoragePath   *string `db:"storage_path" json:"storage_path,omitempty"`

	// Report period
	PeriodStart time.Time `db:"period_start" json:"period_start"`
	PeriodEnd   time.Time `db:"period_end" json:"period_end"`

	// Content
	Summary       *string         `db:"summary" json:"summary,omitempty"`
	Metadata      json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	ExpiresAt     *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	ErrorMessage  *string         `db:"error_message" json:"error_message,omitempty"`
	ErrorDetails  json.RawMessage `db:"error_details" json:"error_details,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ReportGenerationJob represents an asynchronous report generation job
type ReportGenerationJob struct {
	ID        uuid.UUID `db:"id" json:"id"`
	TenantID  uuid.UUID `db:"tenant_id" json:"tenant_id"`
	JobType   string    `db:"job_type" json:"job_type"` // compliance, analytics, custom
	SnapshotID *uuid.UUID `db:"snapshot_id" json:"snapshot_id,omitempty"`
	ReportID   *uuid.UUID `db:"report_id" json:"report_id,omitempty"`

	// Status and progress
	Status  ReportJobStatus `db:"status" json:"status"`
	Progress int            `db:"progress" json:"progress"` // 0-100

	// Configuration
	Format  string          `db:"format" json:"format"`
	Options json.RawMessage `db:"options" json:"options,omitempty"`

	// Timing
	QueuedAt    time.Time  `db:"queued_at" json:"queued_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Error tracking
	ErrorMessage *string         `db:"error_message" json:"error_message,omitempty"`
	ErrorDetails  json.RawMessage `db:"error_details" json:"error_details,omitempty"`
	RetryCount    int             `db:"retry_count" json:"retry_count"`
	MaxRetries    int             `db:"max_retries" json:"max_retries"`

	// Processing metadata
	WorkerID     *string  `db:"worker_id" json:"worker_id,omitempty"`
	CorrelationID *uuid.UUID `db:"correlation_id" json:"correlation_id,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ReportSchedule represents a recurring report generation schedule
type ReportSchedule struct {
	ID           uuid.UUID             `db:"id" json:"id"`
	TenantID     uuid.UUID             `db:"tenant_id" json:"tenant_id"`
	ScheduleName string                `db:"schedule_name" json:"schedule_name"`
	Framework    string                `db:"framework" json:"framework"`
	ReportID     uuid.UUID             `db:"report_id" json:"report_id"`

	// Schedule configuration
	ScheduleType   ReportScheduleType `db:"schedule_type" json:"schedule_type"`
	CronExpression *string            `db:"cron_expression" json:"cron_expression,omitempty"`

	// Output configuration
	Format  string          `db:"format" json:"format"`
	Options json.RawMessage `db:"options" json:"options,omitempty"`

	// Recipient configuration
	Recipients           []string `db:"recipients" json:"recipients"`
	NotifyOnCompletion   bool     `db:"notify_on_completion" json:"notify_on_completion"`
	NotifyOnFailure      bool     `db:"notify_on_failure" json:"notify_on_failure"`

	// Schedule status
	Status ReportScheduleStatus `db:"status" json:"status"`

	// Timing
	NextRunAt          time.Time  `db:"next_run_at" json:"next_run_at"`
	LastRunAt          *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	LastSuccessfulRunAt *time.Time `db:"last_successful_run_at" json:"last_successful_run_at,omitempty"`

	// Run tracking
	TotalRuns     int `db:"total_runs" json:"total_runs"`
	SuccessfulRuns int `db:"successful_runs" json:"successful_runs"`
	FailedRuns    int `db:"failed_runs" json:"failed_runs"`

	// Owner information
	CreatedBy uuid.UUID `db:"created_by" json:"created_by"`
	OwnedBy   uuid.UUID `db:"owned_by" json:"owned_by"`

	// Retention
	RetentionDays int `db:"retention_days" json:"retention_days"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ReportSnapshotStats represents statistics about report snapshots
type ReportSnapshotStats struct {
	PendingCount     int   `json:"pending_count"`
	CompletedCount   int   `json:"completed_count"`
	FailedCount      int   `json:"failed_count"`
	ExpiredCount     int   `json:"expired_count"`
	TotalCount       int   `json:"total_count"`
	TotalStorageBytes int64 `json:"total_storage_bytes"`
}

// =============================================================================
// Compliance Exception Types
// =============================================================================

// ExceptionStatus represents the status of a compliance exception
type ExceptionStatus string

const (
	ExceptionStatusPending ExceptionStatus = "pending"
	ExceptionStatusApproved ExceptionStatus = "approved"
	ExceptionStatusDenied ExceptionStatus = "denied"
	ExceptionStatusExpired ExceptionStatus = "expired"
	ExceptionStatusRevoked ExceptionStatus = "revoked"
)

// ComplianceException represents an exception to a compliance control
type ComplianceException struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	TenantID      uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	ControlID     string          `db:"control_id" json:"control_id"`
	ControlName   string          `db:"control_name" json:"control_name"`
	Framework     string          `db:"framework" json:"framework"`

	// Exception status
	Status    ExceptionStatus `db:"status" json:"status"`
	RiskLevel string          `db:"risk_level" json:"risk_level"` // low, medium, high, critical

	// Approval workflow
	RequestedBy uuid.UUID  `db:"requested_by" json:"requested_by"`
	RequestedAt time.Time `db:"requested_at" json:"requested_at"`
	ApprovedBy   *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt   *time.Time `db:"approved_at" json:"approved_at,omitempty"`
	ExpiresAt    *time.Time `db:"expires_at" json:"expires_at,omitempty"`

	// Justification
	Justification        string   `db:"justification" json:"justification"`
	BusinessReason       string   `db:"business_reason" json:"business_reason,omitempty"`
	CompensatingControls []string `db:"compensating_controls" json:"compensating_controls,omitempty"`

	// Risk acceptance
	RiskAcceptedBy *uuid.UUID `db:"risk_accepted_by" json:"risk_accepted_by,omitempty"`
	RiskAcceptedAt *time.Time `db:"risk_accepted_at" json:"risk_accepted_at,omitempty"`

	// Review
	ReviewDate  *string  `db:"review_date" json:"review_date,omitempty"`
	ReviewNotes *string  `db:"review_notes" json:"review_notes,omitempty"`

	Metadata json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ExceptionStats represents statistics about compliance exceptions
type ExceptionStats struct {
	PendingCount    int `json:"pending_count"`
	ApprovedCount   int `json:"approved_count"`
	DeniedCount     int `json:"denied_count"`
	ExpiredCount    int `json:"expired_count"`
	RevokedCount    int `json:"revoked_count"`
	TotalCount      int `json:"total_count"`
	CriticalCount   int `json:"critical_count"`
	HighCount       int `json:"high_count"`
	ExpiringSoonCount int `json:"expiring_soon_count"`
}

// CreateComplianceExceptionRequest represents a request to create a compliance exception
type CreateComplianceExceptionRequest struct {
	ControlID            string   `json:"control_id" binding:"required"`
	ControlName          string   `json:"control_name" binding:"required"`
	Framework            string   `json:"framework" binding:"required"`
	RiskLevel            string   `json:"risk_level" binding:"required,oneof=low medium high critical"`
	Justification        string   `json:"justification" binding:"required"`
	BusinessReason       string   `json:"business_reason"`
	CompensatingControls []string `json:"compensating_controls"`
	ExpiresAt            *time.Time `json:"expires_at"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// UpdateComplianceExceptionRequest represents a request to update a compliance exception
type UpdateComplianceExceptionRequest struct {
	Justification        *string  `json:"justification"`
	BusinessReason       *string  `json:"business_reason"`
	CompensatingControls *[]string `json:"compensating_controls"`
	ReviewDate           *string  `json:"review_date"`
	ReviewNotes          *string  `json:"review_notes"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// ExceptionApprovalRequest represents a request to approve/deny an exception
type ExceptionApprovalRequest struct {
	Status    ExceptionStatus `json:"status" binding:"required,oneof=approved denied"`
	ExpiresAt *time.Time      `json:"expires_at"`
	Notes     string          `json:"notes"`
}

// ComplianceExceptionResponse represents a compliance exception response
type ComplianceExceptionResponse struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	ControlID        string     `json:"control_id"`
	ControlName      string     `json:"control_name"`
	Framework        string     `json:"framework"`
	Status           string     `json:"status"`
	RiskLevel        string     `json:"risk_level"`
	RequestedBy      uuid.UUID  `json:"requested_by"`
	RequestedAt      time.Time  `json:"requested_at"`
	ApprovedBy       *uuid.UUID `json:"approved_by,omitempty"`
	ApprovedAt       *time.Time `json:"approved_at,omitempty"`
	Justification    string     `json:"justification"`
	BusinessReason   string     `json:"business_reason,omitempty"`
	ReviewDate       *time.Time `json:"review_date,omitempty"`
}

// ReportScheduleRequest represents a request to create/update a report schedule
type ReportScheduleRequest struct {
	ScheduleName       string                `json:"schedule_name" binding:"required"`
	Framework          string                `json:"framework" binding:"required"`
	ReportID           uuid.UUID             `json:"report_id" binding:"required"`
	ScheduleType       ReportScheduleType    `json:"schedule_type" binding:"required"`
	CronExpression     *string               `json:"cron_expression"`
	Format             ReportFormat          `json:"format" binding:"required"`
	Options            map[string]interface{} `json:"options"`
	Recipients         []string              `json:"recipients"`
	NotifyOnCompletion bool                  `json:"notify_on_completion"`
	NotifyOnFailure    bool                  `json:"notify_on_failure"`
	RetentionDays      int                   `json:"retention_days"`
}

// GenerateReportRequest represents a request to generate a report
type GenerateReportRequest struct {
	ReportID   uuid.UUID    `json:"report_id" binding:"required"`
	PeriodStart time.Time   `json:"period_start" binding:"required"`
	PeriodEnd   time.Time   `json:"period_end" binding:"required"`
	Format      ReportFormat `json:"format" binding:"required"`
	Options     map[string]interface{} `json:"options"`
}

// =============================================================================
// Report Persistence Types (compliance_reports table)
// =============================================================================

// ComplianceReport represents a compliance report in the database
type ComplianceReport struct {
	ID              uuid.UUID       `db:"id" json:"id"`
	TenantID        uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	Framework       string          `db:"framework" json:"framework"`
	Status          string          `db:"status" json:"status"` // pending, completed, failed
	GeneratedAt     time.Time       `db:"generated_at" json:"generated_at"`

	// Report period
	PeriodStart     time.Time       `db:"period_start" json:"period_start"`
	PeriodEnd       time.Time       `db:"period_end" json:"period_end"`
	ExpiresAt       *time.Time      `db:"expires_at" json:"expires_at,omitempty"`

	// Compliance metrics
	OverallScore    float64         `db:"overall_score" json:"overall_score"`
	PassedControls  int             `db:"passed_controls" json:"passed_controls"`
	FailedControls  int             `db:"failed_controls" json:"failed_controls"`

	// Report data
	Data            json.RawMessage `db:"data" json:"data,omitempty"`
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`

	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}

// =============================================================================
// Filter Types for Report Queries
// =============================================================================

// ComplianceReportFilter represents filter options for compliance reports
type ComplianceReportFilter struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	Framework string     `json:"framework,omitempty"`
	Status    string     `json:"status,omitempty"`
	DateFrom  time.Time  `json:"date_from,omitempty"`
	DateTo    time.Time  `json:"date_to,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// ReportSnapshotFilter represents filter options for report snapshots
type ReportSnapshotFilter struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	ReportID  *uuid.UUID `json:"report_id,omitempty"`
	Framework string     `json:"framework,omitempty"`
	Status    string     `json:"status,omitempty"`
	DateFrom  time.Time  `json:"date_from,omitempty"`
	DateTo    time.Time  `json:"date_to,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// ReportJobFilter represents filter options for report generation jobs
type ReportJobFilter struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	Status    string     `json:"status,omitempty"`
	JobType   string     `json:"job_type,omitempty"`
	DateFrom  time.Time  `json:"date_from,omitempty"`
	DateTo    time.Time  `json:"date_to,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}

// ReportScheduleFilter represents filter options for report schedules
type ReportScheduleFilter struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	Status    string     `json:"status,omitempty"`
	Framework string     `json:"framework,omitempty"`
	OwnedBy   *uuid.UUID `json:"owned_by,omitempty"`
	Limit     int        `json:"limit"`
	Offset    int        `json:"offset"`
}
