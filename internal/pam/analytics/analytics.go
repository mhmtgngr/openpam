package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// Package analytics provides comprehensive privileged access analytics including:
// - Session analytics and aggregation
// - User activity tracking and anomaly detection
// - Risk scoring for users, targets, and credentials
// - Compliance reporting and dashboards
// - Alert evaluation and notification
// - Report generation and scheduling
//
// The analytics system aggregates raw data from sessions, events, and other sources
// into pre-computed statistics for efficient querying and dashboard visualization.

// InitService initializes the analytics service with all its dependencies
func InitService(db *sqlx.DB, cache *cache.Cache, logger zerolog.Logger) (*Service, *AggregationWorker, *AlertEvaluator, *ReportScheduler) {
	repo := NewRepository(db, logger)
	anomalyRepo := NewAnomalyRepository(db, logger)
	reportRepo := NewReportRepository(db, logger)
	service := NewService(repo, anomalyRepo, reportRepo, cache, logger)

	// Initialize background workers
	aggregationWorker := NewAggregationWorker(repo, cache, logger)
	alertEvaluator := NewAlertEvaluator(repo, cache, logger)
	reportScheduler := NewReportScheduler(repo, cache, logger)

	return service, aggregationWorker, alertEvaluator, reportScheduler
}

// StartWorkers starts all background workers for the analytics service
func StartWorkers(ctx context.Context, aggregationWorker *AggregationWorker, alertEvaluator *AlertEvaluator, reportScheduler *ReportScheduler) {
	aggregationWorker.Start(ctx)
	alertEvaluator.Start(ctx)
	reportScheduler.Start(ctx)
}

// StopWorkers stops all background workers gracefully
func StopWorkers(aggregationWorker *AggregationWorker, alertEvaluator *AlertEvaluator, reportScheduler *ReportScheduler) {
	if aggregationWorker != nil {
		aggregationWorker.Stop()
	}
	if alertEvaluator != nil {
		alertEvaluator.Stop()
	}
	if reportScheduler != nil {
		reportScheduler.Stop()
	}
}

// RecordSessionEvent is a convenience function to record session-related events
func RecordSessionEvent(ctx context.Context, service *Service, eventType string, tenantID, userID, sessionID uuid.UUID, eventData map[string]interface{}) error {
	switch eventType {
	case "session_start":
		sessionType, _ := eventData["type"].(string)
		targetHost, _ := eventData["target_host"].(string)
		targetPort := int(eventData["target_port"].(float64))
		return service.RecordSessionStart(ctx, tenantID, userID, sessionID, sessionType, targetHost, targetPort)

	case "session_end":
		durationSecs := int64(eventData["duration_seconds"].(float64))
		sessionType, _ := eventData["type"].(string)
		return service.RecordSessionEnd(ctx, tenantID, userID, sessionID, time.Duration(durationSecs)*time.Second, sessionType)

	default:
		return fmt.Errorf("unknown session event type: %s", eventType)
	}
}

// GetOverallCompliance retrieves overall compliance status across all frameworks
func GetOverallCompliance(ctx context.Context, service *Service, tenantID uuid.UUID) (*ComplianceStatus, error) {
	return service.GetComplianceSummary(ctx, tenantID)
}

// CalculateUserRisk calculates the current risk score for a user based on recent activity
func CalculateUserRisk(ctx context.Context, service *Service, tenantID, userID uuid.UUID, days int) (float64, RiskLevel, error) {
	score, err := service.GetUserRiskScore(ctx, tenantID, userID, days)
	if err != nil {
		return 0, RiskLevelLow, err
	}

	var level RiskLevel
	switch {
	case score >= 90:
		level = RiskLevelCritical
	case score >= 70:
		level = RiskLevelHigh
	case score >= 40:
		level = RiskLevelMedium
	default:
		level = RiskLevelLow
	}

	return score, level, nil
}

// DetectAnomaliesForUser runs anomaly detection for a specific user
func DetectAnomaliesForUser(ctx context.Context, service *Service, tenantID, userID uuid.UUID) ([]AnomalyResponse, error) {
	return service.EvaluateUserForAnomalies(ctx, tenantID, userID)
}

// GetDailyStats retrieves daily statistics for a tenant
func GetDailyStats(ctx context.Context, service *Service, tenantID uuid.UUID, date time.Time) (*DailyStats, error) {
	startDate := date.Truncate(24 * time.Hour)
	endDate := startDate.Add(24 * time.Hour)

	// Get session stats
	sessionStats, err := service.GetSessionMetricsSummary(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	// Get event stats
	eventStats, err := service.repo.GetRawEventStats(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get event stats: %w", err)
	}

	// Get compliance status
	complianceStatus, err := service.GetComplianceSummary(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance status: %w", err)
	}

	return &DailyStats{
		TenantID:        tenantID,
		Date:            startDate,
		SessionStats:    sessionStats,
		EventStats:      eventStats,
		ComplianceScore: complianceStatus.OverallPercentage,
		GeneratedAt:     time.Now(),
	}, nil
}

// DailyStats represents aggregated statistics for a single day
type DailyStats struct {
	TenantID        uuid.UUID     `json:"tenant_id"`
	Date            time.Time     `json:"date"`
	SessionStats    *SessionStats `json:"session_stats,omitempty"`
	EventStats      *EventStats   `json:"event_stats,omitempty"`
	ComplianceScore float64       `json:"compliance_score"`
	GeneratedAt     time.Time     `json:"generated_at"`
}

// GenerateWeeklyReport generates a comprehensive weekly report for a tenant
func GenerateWeeklyReport(ctx context.Context, service *Service, tenantID, generatedBy uuid.UUID, reportType ReportType, format string) (*ReportSnapshot, error) {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)

	report, err := service.CreateReport(ctx, tenantID, generatedBy, &CreateReportRequest{
		Name:        fmt.Sprintf("Weekly %s Report", reportType),
		Description: "Automatically generated weekly analytics report",
		ReportType:  reportType,
		Config:      mustMarshalJSON(map[string]interface{}{"period": "week"}),
	})
	if err != nil {
		return nil, err
	}

	snapshot, err := service.GenerateReportSnapshot(ctx, report.ID, generatedBy, weekAgo, now, format)
	if err != nil {
		return nil, err
	}

	return snapshot, nil
}

// UpdateAlert updates an existing alert with the given parameters
func UpdateAlert(ctx context.Context, service *Service, alertID uuid.UUID, userID uuid.UUID, enabled *bool, severity *Severity, conditions map[string]interface{}) error {
	_, err := service.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	req := &UpdateAlertRequest{
		Enabled: enabled,
		Severity: severity,
	}

	if conditions != nil {
		req.Conditions = mustMarshalJSON(conditions)
	}

	_, err = service.UpdateAlert(ctx, alertID, userID, req)
	return err
}

// CreateDashboardWithWidgets creates a new dashboard with the specified widgets
func CreateDashboardWithWidgets(ctx context.Context, service *Service, tenantID, createdBy uuid.UUID, name, description string, widgets []CreateWidgetRequest) (*Dashboard, error) {
	req := &CreateDashboardRequest{
		Name:        name,
		Description: description,
		IsPublic:    true,
		Layout:      mustMarshalJSON(map[string]interface{}{"columns": 12}),
		Widgets:     widgets,
	}

	return service.CreateDashboard(ctx, tenantID, createdBy, req)
}

// GetWidgetDataForPeriod retrieves widget data for a specific time period
func GetWidgetDataForPeriod(ctx context.Context, service *Service, widgetID uuid.UUID, daysAgo int) (interface{}, error) {
	now := time.Now()
	startDate := now.Add(-time.Duration(daysAgo) * 24 * time.Hour)
	return service.GetWidgetData(ctx, widgetID, &startDate, &now)
}

// CleanupOldData removes old analytics data based on retention policy
func CleanupOldData(ctx context.Context, repo *Repository, logger zerolog.Logger) error {
	// Default retention policies:
	// - Session analytics: 90 days for hourly, 1 year for daily
	// - Event analytics: 90 days for hourly, 1 year for daily
	// - User activity: 1 year
	// - Metrics: 30 days
	// - Alert triggers: 1 year

	logger.Info().Msg("Starting analytics data cleanup")

	// Delete old hourly session analytics (90 days)
	sessionCutoff := time.Now().Add(-90 * 24 * time.Hour)
	_, err := repo.Db.ExecContext(ctx, `
		DELETE FROM analytics_sessions
		WHERE period_type = 'hour' AND period_start < $1
	`, sessionCutoff)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clean up old session analytics")
	} else {
		logger.Info().Time("cutoff", sessionCutoff).Msg("Cleaned up old hourly session analytics")
	}

	// Delete old hourly event analytics (90 days)
	_, err = repo.Db.ExecContext(ctx, `
		DELETE FROM analytics_events
		WHERE period_type = 'hour' AND period_start < $1
	`, sessionCutoff)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clean up old event analytics")
	}

	// Delete old metrics (30 days)
	metricsCutoff := time.Now().Add(-30 * 24 * time.Hour)
	_, err = repo.Db.ExecContext(ctx, `
		DELETE FROM analytics_metrics
		WHERE recorded_at < $1
	`, metricsCutoff)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clean up old metrics")
	} else {
		logger.Info().Time("cutoff", metricsCutoff).Msg("Cleaned up old metrics")
	}

	logger.Info().Msg("Analytics data cleanup complete")
	return nil
}

// RefreshAllMaterializedViews refreshes all analytics materialized views
func RefreshAllMaterializedViews(ctx context.Context, repo *Repository, logger zerolog.Logger) error {
	logger.Info().Msg("Refreshing all analytics materialized views")

	if err := repo.RefreshMaterializedViews(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to refresh materialized views")
		return err
	}

	logger.Info().Msg("Materialized views refresh complete")
	return nil
}

// ExportMetrics exports metrics in Prometheus format
func ExportMetrics(ctx context.Context, service *Service, tenantID uuid.UUID) (string, error) {
	metrics, err := service.QueryMetrics(ctx, tenantID, "", time.Now().Add(-24*time.Hour), time.Now(), 1000)
	if err != nil {
		return "", err
	}

	// Format as Prometheus text format
	var output string
	for _, metric := range metrics {
		output += fmt.Sprintf("%s%s{tenant_id=\"%s\"} %f\n",
			metric.MetricName,
			formatMetricLabels(metric.Labels),
			metric.TenantID.String(),
			metric.Value,
		)
	}

	return output, nil
}

func formatMetricLabels(labels json.RawMessage) string {
	if len(labels) == 0 || string(labels) == "{}" {
		return ""
	}

	// Parse and format labels
	var labelMap map[string]interface{}
	if err := json.Unmarshal(labels, &labelMap); err != nil {
		return ""
	}

	var result string
	for k, v := range labelMap {
		if result != "" {
			result += ","
		}
		result += fmt.Sprintf("%s=\"%v\"", k, v)
	}

	if result != "" {
		return "{" + result + "}"
	}
	return ""
}

// GetTenantOverview returns a comprehensive overview of analytics for a tenant
func GetTenantOverview(ctx context.Context, service *Service, tenantID uuid.UUID) (*TenantOverview, error) {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	monthAgo := now.Add(-30 * 24 * time.Hour)

	// Get session metrics
	sessionMetrics, _ := service.GetSessionMetricsSummary(ctx, tenantID, weekAgo, now)

	// Get dashboard metrics
	dashboardMetrics, _ := service.GetDashboardMetrics(ctx, tenantID)

	// Get top users
	topUsers, _ := service.GetTopUsersByActivity(ctx, tenantID, weekAgo, now, 10)

	// Get compliance status
	complianceStatus, _ := service.GetComplianceSummary(ctx, tenantID)

	// Get risk scores
	riskScores, _ := service.ListRiskScores(ctx, tenantID, "", "", "", 20)

	// Get anomalies
	anomalies, _ := service.DetectAnomalies(ctx, tenantID)

	// Count active alerts
	alerts, total, _ := service.ListAlerts(ctx, AlertFilter{
		TenantID: tenantID,
		Enabled:  boolPtr(true),
		Limit:    1000,
		Offset:   0,
	})

	_ = total // Use total to avoid unused variable warning

	return &TenantOverview{
		TenantID:           tenantID,
		GeneratedAt:        now,
		PeriodStart:        weekAgo,
		PeriodEnd:          now,
		SessionMetrics:     sessionMetrics,
		DashboardMetrics:   dashboardMetrics,
		TopUsers:           topUsers,
		ComplianceStatus:   complianceStatus,
		RiskScores:         riskScores,
		RecentAnomalies:    anomalies,
		ActiveAlertsCount:  len(alerts),
		MonthlyTrend:       getMonthlyTrend(service, ctx, tenantID, monthAgo, now),
	}, nil
}

// TenantOverview represents a comprehensive analytics overview for a tenant
type TenantOverview struct {
	TenantID           uuid.UUID           `json:"tenant_id"`
	GeneratedAt        time.Time           `json:"generated_at"`
	PeriodStart        time.Time           `json:"period_start"`
	PeriodEnd          time.Time           `json:"period_end"`
	SessionMetrics     *SessionStats       `json:"session_metrics,omitempty"`
	DashboardMetrics   *DashboardMetrics  `json:"dashboard_metrics,omitempty"`
	TopUsers           []TopUser           `json:"top_users,omitempty"`
	ComplianceStatus   *ComplianceStatus  `json:"compliance_status,omitempty"`
	RiskScores         []RiskScoreResponse `json:"risk_scores,omitempty"`
	RecentAnomalies    []AnomalyResponse  `json:"recent_anomalies,omitempty"`
	ActiveAlertsCount  int                 `json:"active_alerts_count"`
	MonthlyTrend       []TimeSeriesDataPoint `json:"monthly_trend,omitempty"`
}

func getMonthlyTrend(service *Service, ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) []TimeSeriesDataPoint {
	trend, err := service.GetSessionTimeSeries(ctx, tenantID, "total", startDate, endDate)
	if err != nil {
		return []TimeSeriesDataPoint{}
	}
	return trend
}

// GetSystemHealth returns health status of the analytics system
func GetSystemHealth(ctx context.Context, repo *Repository) (*SystemHealth, error) {
	health := &SystemHealth{
		Status:     "healthy",
		CheckedAt:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	// Check database connectivity
	if err := repo.Db.PingContext(ctx); err != nil {
		health.Components["database"] = ComponentHealth{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		health.Status = "degraded"
	} else {
		health.Components["database"] = ComponentHealth{
			Status: "healthy",
		}
	}

	// Check recent aggregations
	var aggregationCount int
	_ = repo.Db.GetContext(ctx, &aggregationCount, `
		SELECT COUNT(*) FROM analytics_sessions
		WHERE period_start > NOW() - INTERVAL '2 hours'
	`)
	health.Components["aggregation"] = ComponentHealth{
		Status: "healthy",
		Metadata: map[string]interface{}{
			"recent_aggregations": aggregationCount,
		},
	}

	// Check for recent errors
	var errorCount int
	_ = repo.Db.GetContext(ctx, &errorCount, `
		SELECT COUNT(*) FROM analytics_refresh_log
		WHERE status = 'failed' AND started_at > NOW() - INTERVAL '1 hour'
	`)
	if errorCount > 0 {
		health.Components["refresh_errors"] = ComponentHealth{
			Status: "warning",
			Metadata: map[string]interface{}{
				"recent_errors": errorCount,
			},
		}
		if health.Status == "healthy" {
			health.Status = "warning"
		}
	}

	return health, nil
}

// SystemHealth represents the health status of the analytics system
type SystemHealth struct {
	Status     string                    `json:"status"` // healthy, warning, degraded, unhealthy
	CheckedAt  time.Time                 `json:"checked_at"`
	Components map[string]ComponentHealth `json:"components"`
}

// ComponentHealth represents health of a single component
type ComponentHealth struct {
	Status   string                 `json:"status"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Helper function for boolean pointer
func boolPtr(b bool) *bool {
	return &b
}
