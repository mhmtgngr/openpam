package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/events"
)

// AnalyticsService defines the interface for analytics operations.
//
// This is the PRIMARY analytics service interface that should be used for
// all analytics operations in OpenPAM. The canonical implementation is
// internal/analytics.Service.
//
// NOTE: There is a separate PAM-specific analytics service at
// internal/pam/analytics.Service which provides domain-specific analytics
// for privileged access management. That service has a different interface
// tailored to PAM use cases (session tracking, command monitoring, etc.).
// The two services serve different purposes:
//
//   - internal/analytics.Service (this interface): Comprehensive analytics
//     including compliance, anomaly detection, ransomware detection, SSH key
//     analytics, and command blacklisting. Use this for general analytics,
//     compliance reporting, and security monitoring.
//
//   - internal/pam/analytics.Service: PAM-specific analytics focused on
//     session metrics, user activity, risk scoring, and alerting for
//     privileged access scenarios. Use this for PAM workflow integration.
//
// When choosing which service to use:
//   - For compliance reports, use internal/analytics.Service
//   - for PAM session workflows, use internal/pam/analytics.Service
//   - For command blacklist enforcement, use internal/analytics.Service
//   - For PAM-specific dashboards, use internal/pam/analytics.Service
type AnalyticsService interface {
	// Session Analytics
	GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*SessionSummary, error)
	GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, dateFrom, dateTo time.Time) ([]UserActivity, error)
	GetUserRiskScore(ctx context.Context, tenantID, userID uuid.UUID, days int) (float64, error)
	GetCommandFrequency(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error)
	GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error)
	GetTimeSeriesData(ctx context.Context, tenantID uuid.UUID, metric string, dateFrom, dateTo time.Time) ([]DataPoint, error)

	// Compliance
	GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error)
	ListComplianceReports(ctx context.Context, tenantID uuid.UUID, framework *string) ([]ComplianceReport, error)
	GenerateComplianceReport(ctx context.Context, tenantID uuid.UUID, generatedBy uuid.UUID, framework ComplianceFramework, periodStart, periodEnd time.Time) (*ComplianceReport, error)
	CreateComplianceException(ctx context.Context, exception *ComplianceException) error
	ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error)
	GetComplianceSummary(ctx context.Context, tenantID uuid.UUID) (*ComplianceSummary, error)

	// Anomaly Detection
	DetectAnomalies(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error)
	GetAnomaly(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error)
	ListAnomalies(ctx context.Context, tenantID uuid.UUID, status *string) ([]AnomalyDetection, error)
	UpdateAnomalyStatus(ctx context.Context, id uuid.UUID, status AnomalyStatus, assignedTo *uuid.UUID, notes *string, resolvedBy *uuid.UUID) error
	RunAnomalyDetection(ctx context.Context, tenantID uuid.UUID) ([]AnomalyDetection, error)
	EvaluateUserForAnomalies(ctx context.Context, tenantID, userID uuid.UUID) ([]AnomalyDetection, error)

	// Ransomware Detection
	CreateRansomwareEvent(ctx context.Context, detectionID uuid.UUID, event *RansomwareEvent) error
	GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error)
	TriggerEmergencyResponse(ctx context.Context, tenantID, eventID uuid.UUID) error

	// Command Blacklist
	CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error
	GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error)
	ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error)
	UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error
	DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error
	EvaluateCommandAgainstBlacklist(ctx context.Context, tenantID, userID uuid.UUID, command string, groups []uuid.UUID) (allowed bool, action string, blacklistID *uuid.UUID)

	// SSH Key Analytics
	RecordSSHKeyUsage(ctx context.Context, tenantID, sshKeyID, userID uuid.UUID, target string, duration time.Duration, failed bool) error
	GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error)

	// Cache Management
	InvalidateCache(ctx context.Context, tenantID uuid.UUID, types ...string) error
	WarmCache(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) error
	GetCacheStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error)

	// Event Handlers
	HandleSessionStarted(ctx context.Context, event events.Event) error
	HandleSessionEnded(ctx context.Context, event events.Event) error
	HandleCommandExecuted(ctx context.Context, event events.Event) error
}

// Ensure Service implements the interface
var _ AnalyticsService = (*Service)(nil)
