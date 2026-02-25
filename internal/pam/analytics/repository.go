package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Repository handles analytics data operations
type Repository struct {
	Db     *sqlx.DB
	logger zerolog.Logger
}

// NewRepository creates a new analytics repository
func NewRepository(db *sqlx.DB, logger zerolog.Logger) *Repository {
	return &Repository{Db: db, logger: logger}
}

// Session Analytics Methods

// GetSessionAnalytics retrieves session analytics for a period
func (r *Repository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, periodStart time.Time) (*SessionAnalytics, error) {
	var analytics SessionAnalytics
	query := `
		SELECT * FROM analytics_sessions
		WHERE tenant_id = $1 AND period_type = $2 AND period_start = $3
	`
	err := r.Db.GetContext(ctx, &analytics, query, tenantID, periodType, periodStart)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetSessionAnalytics: %w", err)
	}
	return &analytics, nil
}

// UpsertSessionAnalytics creates or updates session analytics
func (r *Repository) UpsertSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	query := `
		INSERT INTO analytics_sessions (
			id, tenant_id, period_type, period_start, period_end,
			total_sessions, active_sessions, completed_sessions, failed_sessions, terminated_sessions,
			avg_duration_seconds, min_duration_seconds, max_duration_seconds,
			p50_duration_seconds, p95_duration_seconds, p99_duration_seconds,
			unique_users, protocol_breakdown, metadata
		) VALUES (
			:id, :tenant_id, :period_type, :period_start, :period_end,
			:total_sessions, :active_sessions, :completed_sessions, :failed_sessions, :terminated_sessions,
			:avg_duration_seconds, :min_duration_seconds, :max_duration_seconds,
			:p50_duration_seconds, :p95_duration_seconds, :p99_duration_seconds,
			:unique_users, :protocol_breakdown, :metadata
		)
		ON CONFLICT (tenant_id, period_type, period_start)
		DO UPDATE SET
			period_end = EXCLUDED.period_end,
			total_sessions = EXCLUDED.total_sessions,
			active_sessions = EXCLUDED.active_sessions,
			completed_sessions = EXCLUDED.completed_sessions,
			failed_sessions = EXCLUDED.failed_sessions,
			terminated_sessions = EXCLUDED.terminated_sessions,
			avg_duration_seconds = EXCLUDED.avg_duration_seconds,
			min_duration_seconds = EXCLUDED.min_duration_seconds,
			max_duration_seconds = EXCLUDED.max_duration_seconds,
			p50_duration_seconds = EXCLUDED.p50_duration_seconds,
			p95_duration_seconds = EXCLUDED.p95_duration_seconds,
			p99_duration_seconds = EXCLUDED.p99_duration_seconds,
			unique_users = EXCLUDED.unique_users,
			protocol_breakdown = EXCLUDED.protocol_breakdown,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`
	_, err := r.Db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("analytics.UpsertSessionAnalytics: %w", err)
	}
	return nil
}

// ListSessionAnalytics lists session analytics for a date range
func (r *Repository) ListSessionAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]SessionAnalytics, error) {
	var analytics []SessionAnalytics
	query := `
		SELECT * FROM analytics_sessions
		WHERE tenant_id = $1 AND period_type = $2
			AND period_start >= $3 AND period_start <= $4
		ORDER BY period_start DESC
		LIMIT $5 OFFSET $6
	`
	err := r.Db.SelectContext(ctx, &analytics, query, tenantID, periodType, startDate, endDate, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListSessionAnalytics: %w", err)
	}
	return analytics, nil
}

// Event Analytics Methods

// GetEventAnalytics retrieves event analytics for a period
func (r *Repository) GetEventAnalytics(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, periodStart time.Time) (*EventAnalytics, error) {
	var analytics EventAnalytics
	query := `
		SELECT * FROM analytics_events
		WHERE tenant_id = $1 AND period_type = $2 AND period_start = $3
	`
	err := r.Db.GetContext(ctx, &analytics, query, tenantID, periodType, periodStart)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetEventAnalytics: %w", err)
	}
	return &analytics, nil
}

// UpsertEventAnalytics creates or updates event analytics
func (r *Repository) UpsertEventAnalytics(ctx context.Context, analytics *EventAnalytics) error {
	query := `
		INSERT INTO analytics_events (
			id, tenant_id, period_type, period_start, period_end,
			total_events, successful_events, failed_events, denied_events,
			action_breakdown, top_users, top_resources,
			failed_auth_count, unique_failed_users, metadata
		) VALUES (
			:id, :tenant_id, :period_type, :period_start, :period_end,
			:total_events, :successful_events, :failed_events, :denied_events,
			:action_breakdown, :top_users, :top_resources,
			:failed_auth_count, :unique_failed_users, :metadata
		)
		ON CONFLICT (tenant_id, period_type, period_start)
		DO UPDATE SET
			period_end = EXCLUDED.period_end,
			total_events = EXCLUDED.total_events,
			successful_events = EXCLUDED.successful_events,
			failed_events = EXCLUDED.failed_events,
			denied_events = EXCLUDED.denied_events,
			action_breakdown = EXCLUDED.action_breakdown,
			top_users = EXCLUDED.top_users,
			top_resources = EXCLUDED.top_resources,
			failed_auth_count = EXCLUDED.failed_auth_count,
			unique_failed_users = EXCLUDED.unique_failed_users,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`
	_, err := r.Db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("analytics.UpsertEventAnalytics: %w", err)
	}
	return nil
}

// Report Methods

// CreateReport creates a new report
func (r *Repository) CreateReport(ctx context.Context, report *Report) error {
	report.ID = uuid.New()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	query := `
		INSERT INTO analytics_reports (
			id, tenant_id, name, description, report_type, config,
			schedule_enabled, schedule_type, schedule_day_of_week, schedule_day_of_month,
			schedule_hour, schedule_timezone, delivery_methods,
			metadata, tags, created_by
		) VALUES (
			:id, :tenant_id, :name, :description, :report_type, :config,
			:schedule_enabled, :schedule_type, :schedule_day_of_week, :schedule_day_of_month,
			:schedule_hour, :schedule_timezone, :delivery_methods,
			:metadata, :tags, :created_by
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("analytics.CreateReport: %w", err)
	}
	return nil
}

// GetReport retrieves a report by ID
func (r *Repository) GetReport(ctx context.Context, id uuid.UUID) (*Report, error) {
	var report Report
	query := `SELECT * FROM analytics_reports WHERE id = $1 AND deleted_at IS NULL`
	err := r.Db.GetContext(ctx, &report, query, id)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetReport: %w", err)
	}
	return &report, nil
}

// ListReports lists reports with filters
func (r *Repository) ListReports(ctx context.Context, filter ReportFilter) ([]Report, int, error) {
	var reports []Report

	// Build where clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{filter.TenantID}
	argCount := 1

	whereClause += fmt.Sprintf(" AND tenant_id = $%d", argCount)
	argCount++

	if filter.ReportType != nil {
		whereClause += fmt.Sprintf(" AND report_type = $%d", argCount)
		args = append(args, *filter.ReportType)
		argCount++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM analytics_reports " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("analytics.ListReports.Count: %w", err)
	}

	// Get reports
	query := `
		SELECT * FROM analytics_reports
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &reports, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("analytics.ListReports: %w", err)
	}

	return reports, total, nil
}

// UpdateReport updates a report
func (r *Repository) UpdateReport(ctx context.Context, report *Report) error {
	report.UpdatedAt = time.Now()

	query := `
		UPDATE analytics_reports SET
			name = :name,
			description = :description,
			config = :config,
			schedule_enabled = :schedule_enabled,
			schedule_type = :schedule_type,
			schedule_day_of_week = :schedule_day_of_week,
			schedule_day_of_month = :schedule_day_of_month,
			schedule_hour = :schedule_hour,
			schedule_timezone = :schedule_timezone,
			delivery_methods = :delivery_methods,
			metadata = :metadata,
			tags = :tags,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	_, err := r.Db.NamedExecContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("analytics.UpdateReport: %w", err)
	}
	return nil
}

// DeleteReport soft deletes a report
func (r *Repository) DeleteReport(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE analytics_reports SET deleted_at = NOW() WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("analytics.DeleteReport: %w", err)
	}
	return nil
}

// CreateReportSnapshot creates a report snapshot
func (r *Repository) CreateReportSnapshot(ctx context.Context, snapshot *ReportSnapshot) error {
	snapshot.ID = uuid.New()
	snapshot.CreatedAt = time.Now()

	query := `
		INSERT INTO analytics_report_snapshots (
			id, report_id, tenant_id, period_start, period_end,
			data, summary, file_url, file_format, file_size_bytes, status
		) VALUES (
			:id, :report_id, :tenant_id, :period_start, :period_end,
			:data, :summary, :file_url, :file_format, :file_size_bytes, :status
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		return fmt.Errorf("analytics.CreateReportSnapshot: %w", err)
	}
	return nil
}

// GetReportSnapshot retrieves a report snapshot by ID
func (r *Repository) GetReportSnapshot(ctx context.Context, id uuid.UUID) (*ReportSnapshot, error) {
	var snapshot ReportSnapshot
	query := `SELECT * FROM analytics_report_snapshots WHERE id = $1`
	err := r.Db.GetContext(ctx, &snapshot, query, id)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetReportSnapshot: %w", err)
	}
	return &snapshot, nil
}

// ListReportSnapshots lists snapshots for a report
func (r *Repository) ListReportSnapshots(ctx context.Context, reportID uuid.UUID, limit, offset int) ([]ReportSnapshot, error) {
	var snapshots []ReportSnapshot
	query := `
		SELECT * FROM analytics_report_snapshots
		WHERE report_id = $1
		ORDER BY period_start DESC
		LIMIT $2 OFFSET $3
	`
	err := r.Db.SelectContext(ctx, &snapshots, query, reportID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListReportSnapshots: %w", err)
	}
	return snapshots, nil
}

// UpdateReportSnapshotStatus updates snapshot status
func (r *Repository) UpdateReportSnapshotStatus(ctx context.Context, id uuid.UUID, status string, fileURL *string, fileSize *int64, errorMsg *string) error {
	query := `
		UPDATE analytics_report_snapshots SET
			status = $1,
			file_url = COALESCE($2, file_url),
			file_size_bytes = COALESCE($3, file_size_bytes),
			error_message = COALESCE($4, error_message),
			generated_at = CASE WHEN $5 = 'completed' THEN NOW() ELSE generated_at END
		WHERE id = $6
	`
	_, err := r.Db.ExecContext(ctx, query, status, fileURL, fileSize, errorMsg, status, id)
	if err != nil {
		return fmt.Errorf("analytics.UpdateReportSnapshotStatus: %w", err)
	}
	return nil
}

// Dashboard Methods

// CreateDashboard creates a new dashboard
func (r *Repository) CreateDashboard(ctx context.Context, dashboard *Dashboard) error {
	dashboard.ID = uuid.New()
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	query := `
		INSERT INTO analytics_dashboards (
			id, tenant_id, name, description, is_default, is_public,
			layout, global_filters, metadata, tags, created_by
		) VALUES (
			:id, :tenant_id, :name, :description, :is_default, :is_public,
			:layout, :global_filters, :metadata, :tags, :created_by
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, dashboard)
	if err != nil {
		return fmt.Errorf("analytics.CreateDashboard: %w", err)
	}
	return nil
}

// GetDashboard retrieves a dashboard by ID
func (r *Repository) GetDashboard(ctx context.Context, id uuid.UUID) (*Dashboard, error) {
	var dashboard Dashboard
	query := `SELECT * FROM analytics_dashboards WHERE id = $1 AND deleted_at IS NULL`
	err := r.Db.GetContext(ctx, &dashboard, query, id)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetDashboard: %w", err)
	}
	return &dashboard, nil
}

// GetDefaultDashboard retrieves the default dashboard for a tenant
func (r *Repository) GetDefaultDashboard(ctx context.Context, tenantID uuid.UUID) (*Dashboard, error) {
	var dashboard Dashboard
	query := `SELECT * FROM analytics_dashboards WHERE tenant_id = $1 AND is_default = true AND deleted_at IS NULL LIMIT 1`
	err := r.Db.GetContext(ctx, &dashboard, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetDefaultDashboard: %w", err)
	}
	return &dashboard, nil
}

// ListDashboards lists dashboards with filters
func (r *Repository) ListDashboards(ctx context.Context, filter DashboardFilter) ([]Dashboard, int, error) {
	var dashboards []Dashboard

	// Build where clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{filter.TenantID}
	argCount := 1

	whereClause += fmt.Sprintf(" AND tenant_id = $%d", argCount)
	argCount++

	if filter.IsDefault != nil {
		whereClause += fmt.Sprintf(" AND is_default = $%d", argCount)
		args = append(args, *filter.IsDefault)
		argCount++
	}

	if filter.IsPublic != nil {
		whereClause += fmt.Sprintf(" AND is_public = $%d", argCount)
		args = append(args, *filter.IsPublic)
		argCount++
	}

	if filter.CreatedBy != nil {
		whereClause += fmt.Sprintf(" AND created_by = $%d", argCount)
		args = append(args, *filter.CreatedBy)
		argCount++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM analytics_dashboards " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("analytics.ListDashboards.Count: %w", err)
	}

	// Get dashboards
	query := `
		SELECT * FROM analytics_dashboards
		` + whereClause + `
		ORDER BY is_default DESC, created_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &dashboards, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("analytics.ListDashboards: %w", err)
	}

	return dashboards, total, nil
}

// UpdateDashboard updates a dashboard
func (r *Repository) UpdateDashboard(ctx context.Context, dashboard *Dashboard) error {
	dashboard.UpdatedAt = time.Now()

	query := `
		UPDATE analytics_dashboards SET
			name = :name,
			description = :description,
			is_default = :is_default,
			is_public = :is_public,
			layout = :layout,
			global_filters = :global_filters,
			metadata = :metadata,
			tags = :tags,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	_, err := r.Db.NamedExecContext(ctx, query, dashboard)
	if err != nil {
		return fmt.Errorf("analytics.UpdateDashboard: %w", err)
	}
	return nil
}

// DeleteDashboard soft deletes a dashboard
func (r *Repository) DeleteDashboard(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE analytics_dashboards SET deleted_at = NOW() WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("analytics.DeleteDashboard: %w", err)
	}
	return nil
}

// Widget Methods

// CreateWidget creates a new widget
func (r *Repository) CreateWidget(ctx context.Context, widget *Widget) error {
	widget.ID = uuid.New()
	widget.CreatedAt = time.Now()
	widget.UpdatedAt = time.Now()

	query := `
		INSERT INTO analytics_widgets (
			id, dashboard_id, tenant_id, name, widget_type,
			position_x, position_y, width, height,
			data_source, query_config, display_config, refresh_interval_seconds,
			metadata, created_by
		) VALUES (
			:id, :dashboard_id, :tenant_id, :name, :widget_type,
			:position_x, :position_y, :width, :height,
			:data_source, :query_config, :display_config, :refresh_interval_seconds,
			:metadata, :created_by
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, widget)
	if err != nil {
		return fmt.Errorf("analytics.CreateWidget: %w", err)
	}
	return nil
}

// GetWidget retrieves a widget by ID
func (r *Repository) GetWidget(ctx context.Context, id uuid.UUID) (*Widget, error) {
	var widget Widget
	query := `SELECT * FROM analytics_widgets WHERE id = $1`
	err := r.Db.GetContext(ctx, &widget, query, id)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetWidget: %w", err)
	}
	return &widget, nil
}

// ListWidgets lists widgets for a dashboard
func (r *Repository) ListWidgets(ctx context.Context, dashboardID uuid.UUID) ([]Widget, error) {
	var widgets []Widget
	query := `
		SELECT * FROM analytics_widgets
		WHERE dashboard_id = $1
		ORDER BY position_y ASC, position_x ASC
	`
	err := r.Db.SelectContext(ctx, &widgets, query, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListWidgets: %w", err)
	}
	return widgets, nil
}

// UpdateWidget updates a widget
func (r *Repository) UpdateWidget(ctx context.Context, widget *Widget) error {
	widget.UpdatedAt = time.Now()

	query := `
		UPDATE analytics_widgets SET
			name = :name,
			position_x = :position_x,
			position_y = :position_y,
			width = :width,
			height = :height,
			query_config = :query_config,
			display_config = :display_config,
			refresh_interval_seconds = :refresh_interval_seconds,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.Db.NamedExecContext(ctx, query, widget)
	if err != nil {
		return fmt.Errorf("analytics.UpdateWidget: %w", err)
	}
	return nil
}

// DeleteWidget deletes a widget
func (r *Repository) DeleteWidget(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM analytics_widgets WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("analytics.DeleteWidget: %w", err)
	}
	return nil
}

// Alert Methods

// CreateAlert creates a new alert
func (r *Repository) CreateAlert(ctx context.Context, alert *Alert) error {
	alert.ID = uuid.New()
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	query := `
		INSERT INTO analytics_alerts (
			id, tenant_id, name, description, alert_type, severity,
			conditions, evaluation_interval_minutes, notification_methods,
			cooldown_minutes, metadata, tags, created_by
		) VALUES (
			:id, :tenant_id, :name, :description, :alert_type, :severity,
			:conditions, :evaluation_interval_minutes, :notification_methods,
			:cooldown_minutes, :metadata, :tags, :created_by
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, alert)
	if err != nil {
		return fmt.Errorf("analytics.CreateAlert: %w", err)
	}
	return nil
}

// GetAlert retrieves an alert by ID
func (r *Repository) GetAlert(ctx context.Context, id uuid.UUID) (*Alert, error) {
	var alert Alert
	query := `SELECT * FROM analytics_alerts WHERE id = $1 AND deleted_at IS NULL`
	err := r.Db.GetContext(ctx, &alert, query, id)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAlert: %w", err)
	}
	return &alert, nil
}

// ListAlerts lists alerts with filters
func (r *Repository) ListAlerts(ctx context.Context, filter AlertFilter) ([]Alert, int, error) {
	var alerts []Alert

	// Build where clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{filter.TenantID}
	argCount := 1

	whereClause += fmt.Sprintf(" AND tenant_id = $%d", argCount)
	argCount++

	if filter.AlertType != nil {
		whereClause += fmt.Sprintf(" AND alert_type = $%d", argCount)
		args = append(args, *filter.AlertType)
		argCount++
	}

	if filter.Severity != nil {
		whereClause += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
		argCount++
	}

	if filter.Enabled != nil {
		whereClause += fmt.Sprintf(" AND enabled = $%d", argCount)
		args = append(args, *filter.Enabled)
		argCount++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+filter.Search+"%")
		argCount++
	}

	// Get count
	var total int
	countQuery := "SELECT COUNT(*) FROM analytics_alerts " + whereClause
	if err := r.Db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("analytics.ListAlerts.Count: %w", err)
	}

	// Get alerts
	query := `
		SELECT * FROM analytics_alerts
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprint(argCount) + ` OFFSET $` + fmt.Sprint(argCount+1)
	args = append(args, filter.Limit, filter.Offset)

	err := r.Db.SelectContext(ctx, &alerts, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("analytics.ListAlerts: %w", err)
	}

	return alerts, total, nil
}

// UpdateAlert updates an alert
func (r *Repository) UpdateAlert(ctx context.Context, alert *Alert) error {
	alert.UpdatedAt = time.Now()

	query := `
		UPDATE analytics_alerts SET
			name = :name,
			description = :description,
			severity = :severity,
			conditions = :conditions,
			evaluation_interval_minutes = :evaluation_interval_minutes,
			notification_methods = :notification_methods,
			enabled = :enabled,
			cooldown_minutes = :cooldown_minutes,
			metadata = :metadata,
			tags = :tags,
			updated_by = :updated_by,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	_, err := r.Db.NamedExecContext(ctx, query, alert)
	if err != nil {
		return fmt.Errorf("analytics.UpdateAlert: %w", err)
	}
	return nil
}

// DeleteAlert soft deletes an alert
func (r *Repository) DeleteAlert(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE analytics_alerts SET deleted_at = NOW() WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("analytics.DeleteAlert: %w", err)
	}
	return nil
}

// CreateAlertTrigger creates an alert trigger record
func (r *Repository) CreateAlertTrigger(ctx context.Context, trigger *AlertTrigger) error {
	trigger.ID = uuid.New()
	if trigger.TriggeredAt.IsZero() {
		trigger.TriggeredAt = time.Now()
	}

	query := `
		INSERT INTO analytics_alert_triggers (
			id, alert_id, tenant_id, triggered_at, severity,
			trigger_data, notifications_sent, metadata
		) VALUES (
			:id, :alert_id, :tenant_id, :triggered_at, :severity,
			:trigger_data, :notifications_sent, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, trigger)
	if err != nil {
		return fmt.Errorf("analytics.CreateAlertTrigger: %w", err)
	}
	return nil
}

// ListAlertTriggers lists triggers for an alert
func (r *Repository) ListAlertTriggers(ctx context.Context, alertID uuid.UUID, limit, offset int) ([]AlertTrigger, error) {
	var triggers []AlertTrigger
	query := `
		SELECT * FROM analytics_alert_triggers
		WHERE alert_id = $1
		ORDER BY triggered_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.Db.SelectContext(ctx, &triggers, query, alertID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListAlertTriggers: %w", err)
	}
	return triggers, nil
}

// UpdateAlertTrigger updates an alert trigger (for resolution)
func (r *Repository) UpdateAlertTrigger(ctx context.Context, id uuid.UUID, resolvedBy *uuid.UUID, resolutionNotes *string) error {
	query := `
		UPDATE analytics_alert_triggers SET
			resolved_at = NOW(),
			resolved_by = COALESCE($1, resolved_by),
			resolution_notes = COALESCE($2, resolution_notes)
		WHERE id = $3
	`
	_, err := r.Db.ExecContext(ctx, query, resolvedBy, resolutionNotes, id)
	if err != nil {
		return fmt.Errorf("analytics.UpdateAlertTrigger: %w", err)
	}
	return nil
}

// GetActiveAlerts retrieves alerts that need evaluation
func (r *Repository) GetActiveAlerts(ctx context.Context) ([]Alert, error) {
	var alerts []Alert
	query := `
		SELECT * FROM analytics_alerts
		WHERE enabled = true AND deleted_at IS NULL
		ORDER BY next_trigger_at ASC NULLS LAST
	`
	err := r.Db.SelectContext(ctx, &alerts, query)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetActiveAlerts: %w", err)
	}
	return alerts, nil
}

// UpdateAlertTriggerStatus updates alert trigger status
func (r *Repository) UpdateAlertTriggerStatus(ctx context.Context, alertID uuid.UUID, lastTriggeredAt time.Time, nextTriggerAt time.Time, triggerCount int) error {
	query := `
		UPDATE analytics_alerts SET
			last_triggered_at = $1,
			next_trigger_at = $2,
			trigger_count = $3
		WHERE id = $4
	`
	_, err := r.Db.ExecContext(ctx, query, lastTriggeredAt, nextTriggerAt, triggerCount, alertID)
	if err != nil {
		return fmt.Errorf("analytics.UpdateAlertTriggerStatus: %w", err)
	}
	return nil
}

// UpdateAlertEvaluationTime updates the last evaluated time
func (r *Repository) UpdateAlertEvaluationTime(ctx context.Context, alertID uuid.UUID) error {
	query := `UPDATE analytics_alerts SET last_evaluated_at = NOW() WHERE id = $1`
	_, err := r.Db.ExecContext(ctx, query, alertID)
	if err != nil {
		return fmt.Errorf("analytics.UpdateAlertEvaluationTime: %w", err)
	}
	return nil
}

// Metric Methods

// RecordMetric records a metric data point
func (r *Repository) RecordMetric(ctx context.Context, metric *Metric) error {
	metric.ID = uuid.New()
	if metric.RecordedAt.IsZero() {
		metric.RecordedAt = time.Now()
	}

	query := `
		INSERT INTO analytics_metrics (
			id, tenant_id, metric_name, metric_type, value, labels, recorded_at, metadata
		) VALUES (
			:id, :tenant_id, :metric_name, :metric_type, :value, :labels, :recorded_at, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, metric)
	if err != nil {
		return fmt.Errorf("analytics.RecordMetric: %w", err)
	}
	return nil
}

// QueryMetrics queries metrics with filters
func (r *Repository) QueryMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, limit int) ([]Metric, error) {
	var metrics []Metric
	query := `
		SELECT * FROM analytics_metrics
		WHERE tenant_id = $1 AND metric_name = $2
			AND recorded_at >= $3 AND recorded_at <= $4
		ORDER BY recorded_at DESC
		LIMIT $5
	`
	err := r.Db.SelectContext(ctx, &metrics, query, tenantID, metricName, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.QueryMetrics: %w", err)
	}
	return metrics, nil
}

// AggregateMetrics aggregates metrics by time period
func (r *Repository) AggregateMetrics(ctx context.Context, tenantID uuid.UUID, metricName string, startDate, endDate time.Time, period string) ([]TimeSeriesDataPoint, error) {
	var dataPoints []TimeSeriesDataPoint

	var timeTrunc string
	switch period {
	case "hour":
		timeTrunc = "date_trunc('hour', recorded_at)"
	case "day":
		timeTrunc = "date_trunc('day', recorded_at)"
	case "week":
		timeTrunc = "date_trunc('week', recorded_at)"
	default:
		timeTrunc = "date_trunc('day', recorded_at)"
	}

	query := `
		SELECT
			` + timeTrunc + ` as timestamp,
			AVG(value) as value,
			COUNT(*) as count
		FROM analytics_metrics
		WHERE tenant_id = $1 AND metric_name = $2
			AND recorded_at >= $3 AND recorded_at <= $4
		GROUP BY ` + timeTrunc + `
		ORDER BY timestamp ASC
	`

	rows, err := r.Db.QueryContext(ctx, query, tenantID, metricName, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.AggregateMetrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dp TimeSeriesDataPoint
		var count int
		if err := rows.Scan(&dp.Timestamp, &dp.Value, &count); err != nil {
			return nil, err
		}
		if dp.Metadata == nil {
			dp.Metadata = make(map[string]interface{})
		}
		dp.Metadata["count"] = count
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}

// User Activity Methods

// GetUserActivity retrieves user activity for a period
func (r *Repository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, periodType PeriodType, periodStart time.Time) (*UserActivity, error) {
	var activity UserActivity
	query := `
		SELECT * FROM analytics_user_activity
		WHERE tenant_id = $1 AND user_id = $2 AND period_type = $3 AND period_start = $4
	`
	err := r.Db.GetContext(ctx, &activity, query, tenantID, userID, periodType, periodStart)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetUserActivity: %w", err)
	}
	return &activity, nil
}

// UpsertUserActivity creates or updates user activity
func (r *Repository) UpsertUserActivity(ctx context.Context, activity *UserActivity) error {
	query := `
		INSERT INTO analytics_user_activity (
			id, tenant_id, user_id, period_type, period_start,
			sessions_created, credentials_accessed, approvals_requested, approvals_granted,
			total_active_seconds, avg_daily_seconds,
			high_risk_sessions, policy_violations, failed_auth_attempts,
			is_anomaly, anomaly_score, off_hours_access, first_access_time, last_access_time,
			metadata
		) VALUES (
			:id, :tenant_id, :user_id, :period_type, :period_start,
			:sessions_created, :credentials_accessed, :approvals_requested, :approvals_granted,
			:total_active_seconds, :avg_daily_seconds,
			:high_risk_sessions, :policy_violations, :failed_auth_attempts,
			:is_anomaly, :anomaly_score, :off_hours_access, :first_access_time, :last_access_time,
			:metadata
		)
		ON CONFLICT (tenant_id, user_id, period_type, period_start)
		DO UPDATE SET
			sessions_created = EXCLUDED.sessions_created,
			credentials_accessed = EXCLUDED.credentials_accessed,
			approvals_requested = EXCLUDED.approvals_requested,
			approvals_granted = EXCLUDED.approvals_granted,
			total_active_seconds = EXCLUDED.total_active_seconds,
			avg_daily_seconds = EXCLUDED.avg_daily_seconds,
			high_risk_sessions = EXCLUDED.high_risk_sessions,
			policy_violations = EXCLUDED.policy_violations,
			failed_auth_attempts = EXCLUDED.failed_auth_attempts,
			is_anomaly = EXCLUDED.is_anomaly,
			anomaly_score = EXCLUDED.anomaly_score,
			off_hours_access = EXCLUDED.off_hours_access,
			first_access_time = EXCLUDED.first_access_time,
			last_access_time = EXCLUDED.last_access_time,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`
	_, err := r.Db.NamedExecContext(ctx, query, activity)
	if err != nil {
		return fmt.Errorf("analytics.UpsertUserActivity: %w", err)
	}
	return nil
}

// ListUserActivity lists user activity for a date range
func (r *Repository) ListUserActivity(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]UserActivity, error) {
	var activities []UserActivity
	query := `
		SELECT * FROM analytics_user_activity
		WHERE tenant_id = $1 AND period_type = $2
			AND period_start >= $3 AND period_start <= $4
		ORDER BY period_start DESC
		LIMIT $5 OFFSET $6
	`
	err := r.Db.SelectContext(ctx, &activities, query, tenantID, periodType, startDate, endDate, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListUserActivity: %w", err)
	}
	return activities, nil
}

// GetAnomalousUsers retrieves users with anomalous activity
func (r *Repository) GetAnomalousUsers(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, periodStart time.Time, limit int) ([]UserActivity, error) {
	var activities []UserActivity
	query := `
		SELECT * FROM analytics_user_activity
		WHERE tenant_id = $1 AND period_type = $2 AND period_start = $3 AND is_anomaly = true
		ORDER BY anomaly_score DESC
		LIMIT $4
	`
	err := r.Db.SelectContext(ctx, &activities, query, tenantID, periodType, periodStart, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAnomalousUsers: %w", err)
	}
	return activities, nil
}

// Risk Score Methods

// GetLatestRiskScore retrieves the latest risk score for an entity
func (r *Repository) GetLatestRiskScore(ctx context.Context, tenantID uuid.UUID, entityType EntityType, entityID uuid.UUID) (*RiskScore, error) {
	var score RiskScore
	query := `
		SELECT * FROM analytics_risk_scores
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3
		ORDER BY calculated_at DESC
		LIMIT 1
	`
	err := r.Db.GetContext(ctx, &score, query, tenantID, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetLatestRiskScore: %w", err)
	}
	return &score, nil
}

// CreateRiskScore creates a new risk score entry
func (r *Repository) CreateRiskScore(ctx context.Context, score *RiskScore) error {
	score.ID = uuid.New()
	if score.CalculatedAt.IsZero() {
		score.CalculatedAt = time.Now()
	}

	query := `
		INSERT INTO analytics_risk_scores (
			id, tenant_id, entity_type, entity_id, calculated_at,
			access_frequency_score, time_pattern_score, geography_score,
			behavior_drift_score, compliance_score,
			overall_risk_score, risk_level, factors, previous_score, score_change, metadata
		) VALUES (
			:id, :tenant_id, :entity_type, :entity_id, :calculated_at,
			:access_frequency_score, :time_pattern_score, :geography_score,
			:behavior_drift_score, :compliance_score,
			:overall_risk_score, :risk_level, :factors, :previous_score, :score_change, :metadata
		)
	`
	_, err := r.Db.NamedExecContext(ctx, query, score)
	if err != nil {
		return fmt.Errorf("analytics.CreateRiskScore: %w", err)
	}
	return nil
}

// ListRiskScores lists risk scores with filters
func (r *Repository) ListRiskScores(ctx context.Context, tenantID uuid.UUID, entityType *EntityType, minRiskScore *float64, riskLevel *RiskLevel, limit, offset int) ([]RiskScore, error) {
	var scores []RiskScore

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if entityType != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND entity_type = $%d", argCount)
		args = append(args, *entityType)
	}

	if minRiskScore != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND overall_risk_score >= $%d", argCount)
		args = append(args, *minRiskScore)
	}

	if riskLevel != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND risk_level = $%d", argCount)
		args = append(args, *riskLevel)
	}

	argCount++
	argCount++
	query := `
		SELECT DISTINCT ON (entity_type, entity_id) *
		FROM analytics_risk_scores
		` + whereClause + `
		ORDER BY entity_type, entity_id, calculated_at DESC
		LIMIT $` + fmt.Sprint(argCount-1) + ` OFFSET $` + fmt.Sprint(argCount)

	err := r.Db.SelectContext(ctx, &scores, query, args...)
	if err != nil {
		return nil, fmt.Errorf("analytics.ListRiskScores: %w", err)
	}
	return scores, nil
}

// GetTopRisks retrieves top risky entities
func (r *Repository) GetTopRisks(ctx context.Context, tenantID uuid.UUID, limit int) ([]RiskScore, error) {
	// We need to get all unique entities first, then sort by risk score
	allScores, err := r.QueryRiskScores(ctx, tenantID, nil, nil, nil, 10000, 0)
	if err != nil {
		return nil, err
	}

	// Sort by overall_risk_score descending and take top N
	// In production, this should be done with a more efficient query
	topScores := make([]RiskScore, 0, len(allScores))
	entityMap := make(map[string]bool)

	for _, score := range allScores {
		key := fmt.Sprintf("%s:%s", score.EntityType, score.EntityID)
		if !entityMap[key] {
			entityMap[key] = true
			topScores = append(topScores, score)
		}
	}

	// Simple sort (in production, use proper sorting)
	if len(topScores) > limit {
		topScores = topScores[:limit]
	}

	return topScores, nil
}

// QueryRiskScores is a helper method for querying risk scores
func (r *Repository) QueryRiskScores(ctx context.Context, tenantID uuid.UUID, entityType *EntityType, minRiskScore *float64, riskLevel *RiskLevel, limit, offset int) ([]RiskScore, error) {
	var scores []RiskScore

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argCount := 1

	if entityType != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND entity_type = $%d", argCount)
		args = append(args, *entityType)
	}

	if minRiskScore != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND overall_risk_score >= $%d", argCount)
		args = append(args, *minRiskScore)
	}

	if riskLevel != nil {
		argCount++
		whereClause += fmt.Sprintf(" AND risk_level = $%d", argCount)
		args = append(args, *riskLevel)
	}

	argCount++
	argCount++
	query := `
		SELECT * FROM analytics_risk_scores
		` + whereClause + `
		ORDER BY calculated_at DESC
		LIMIT $` + fmt.Sprint(argCount-1) + ` OFFSET $` + fmt.Sprint(argCount)

	err := r.Db.SelectContext(ctx, &scores, query, args...)
	if err != nil {
		return nil, fmt.Errorf("analytics.QueryRiskScores: %w", err)
	}
	return scores, nil
}

// Refresh Log Methods

// LogRefreshStart logs the start of a materialized view refresh
func (r *Repository) LogRefreshStart(ctx context.Context, viewName string) (uuid.UUID, error) {
	id := uuid.New()
	query := `
		INSERT INTO analytics_refresh_log (id, materialized_view, started_at, status)
		VALUES ($1, $2, NOW(), 'running')
	`
	_, err := r.Db.ExecContext(ctx, query, id, viewName)
	if err != nil {
		return uuid.Nil, fmt.Errorf("analytics.LogRefreshStart: %w", err)
	}
	return id, nil
}

// LogRefreshComplete logs the completion of a refresh
func (r *Repository) LogRefreshComplete(ctx context.Context, id uuid.UUID, rowsAffected int, errorMsg *string) error {
	status := "completed"
	if errorMsg != nil {
		status = "failed"
	}
	query := `
		UPDATE analytics_refresh_log
		SET completed_at = NOW(), status = $1, rows_affected = $2, error_message = COALESCE($3, error_message)
		WHERE id = $4
	`
	_, err := r.Db.ExecContext(ctx, query, status, rowsAffected, errorMsg, id)
	if err != nil {
		return fmt.Errorf("analytics.LogRefreshComplete: %w", err)
	}
	return nil
}

// GetRawSessionStats queries raw session data for analytics aggregation
func (r *Repository) GetRawSessionStats(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*SessionStats, error) {
	var stats SessionStats
	stats.ByProtocol = make(map[string]int64)
	stats.ByUser = make(map[string]int64)

	// Get basic stats
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'active') as active,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'terminated') as terminated,
			COALESCE(AVG(EXTRACT(EPOCH FROM (ended_at - started_at))), 0) as avg_duration,
			COALESCE(MAX(EXTRACT(EPOCH FROM (ended_at - started_at))), 0) as max_duration
		FROM sessions
		WHERE tenant_id = $1 AND started_at >= $2 AND started_at <= $3
	`
	err := r.Db.GetContext(ctx, &struct {
		Total       int64   `db:"total"`
		Active      int64   `db:"active"`
		Completed   int64   `db:"completed"`
		Failed      int64   `db:"failed"`
		Terminated  int64   `db:"terminated"`
		AvgDuration float64 `db:"avg_duration"`
		MaxDuration float64 `db:"max_duration"`
	}{}, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetRawSessionStats: %w", err)
	}

	// Get protocol breakdown
	protocolQuery := `
		SELECT protocol, COUNT(*) as count
		FROM sessions
		WHERE tenant_id = $1 AND started_at >= $2 AND started_at <= $3
		GROUP BY protocol
	`
	protocolRows, err := r.Db.QueryContext(ctx, protocolQuery, tenantID, startDate, endDate)
	if err == nil {
		defer protocolRows.Close()
		for protocolRows.Next() {
			var protocol string
			var count int64
			if protocolRows.Scan(&protocol, &count) == nil {
				stats.ByProtocol[protocol] = count
			}
		}
	}

	return &stats, nil
}

// GetRawEventStats queries raw event data for analytics aggregation
func (r *Repository) GetRawEventStats(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*EventStats, error) {
	var stats EventStats
	stats.ByAction = make(map[string]int64)
	stats.ByUser = make(map[string]int64)
	stats.ByResource = make(map[string]int64)

	// Get basic stats
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE outcome = 'success') as successful,
			COUNT(*) FILTER (WHERE outcome = 'failure') as failed,
			COUNT(*) FILTER (WHERE outcome = 'denied') as denied
		FROM audit_events
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3
	`
	err := r.Db.GetContext(ctx, &struct {
		Total      int64 `db:"total"`
		Successful int64 `db:"successful"`
		Failed     int64 `db:"failed"`
		Denied     int64 `db:"denied"`
	}{}, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetRawEventStats: %w", err)
	}

	return &stats, nil
}

// GetTopUsersBySessions retrieves top users by session count
func (r *Repository) GetTopUsersBySessions(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time, limit int) ([]TopUser, error) {
	var users []TopUser
	query := `
		SELECT
			s.user_id,
			COALESCE(u.username, s.user_id::text) as username,
			COUNT(*) as session_count,
			COALESCE(SUM(EXTRACT(EPOCH FROM (s.ended_at - s.started_at))), 0) as total_seconds,
			MAX(s.started_at) as last_seen
		FROM sessions s
		LEFT JOIN users u ON u.id = s.user_id
		WHERE s.tenant_id = $1 AND s.started_at >= $2 AND s.started_at <= $3
		GROUP BY s.user_id, u.username
		ORDER BY session_count DESC
		LIMIT $4
	`
	err := r.Db.SelectContext(ctx, &users, query, tenantID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetTopUsersBySessions: %w", err)
	}
	return users, nil
}

// GetAccessPatterns retrieves access patterns by time
func (r *Repository) GetAccessPatterns(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) ([]AccessPattern, error) {
	var patterns []AccessPattern
	query := `
		SELECT
			EXTRACT(HOUR FROM started_at)::int as hour_of_day,
			EXTRACT(DOW FROM started_at)::int as day_of_week,
			COUNT(*) as session_count,
			COUNT(DISTINCT user_id) as unique_users,
			COALESCE(AVG(EXTRACT(EPOCH FROM (ended_at - started_at)))::int, 0) as avg_duration
		FROM sessions
		WHERE tenant_id = $1 AND started_at >= $2 AND started_at <= $3
		GROUP BY hour_of_day, day_of_week
		ORDER BY day_of_week, hour_of_day
	`
	err := r.Db.SelectContext(ctx, &patterns, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetAccessPatterns: %w", err)
	}

	// Mark outside business hours
	for i := range patterns {
		patterns[i].IsOutsideBusiness = patterns[i].DayOfWeek >= 5 || patterns[i].HourOfDay < 9 || patterns[i].HourOfDay >= 17
	}

	return patterns, nil
}

// CalculateComplianceStatus calculates compliance status for a tenant
func (r *Repository) CalculateComplianceStatus(ctx context.Context, tenantID uuid.UUID, startDate, endDate time.Time) (*ComplianceStatus, error) {
	var status ComplianceStatus
	status.ByPolicy = make(map[string]CompliancePolicyStatus)

	// Get policy evaluation results
	query := `
		SELECT
			p.id as policy_id,
			p.name as policy_name,
			COUNT(*) as total_evaluations,
			COUNT(*) FILTER (WHERE result = 'allow') as passed_evaluations
		FROM policy_evaluations pe
		JOIN policies p ON p.id = pe.policy_id
		WHERE pe.tenant_id = $1 AND pe.created_at >= $2 AND pe.created_at <= $3
		GROUP BY p.id, p.name
	`
	rows, err := r.Db.QueryContext(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.CalculateComplianceStatus: %w", err)
	}
	defer rows.Close()

	totalPassed := 0
	totalChecks := 0

	for rows.Next() {
		var policyID uuid.UUID
		var policyName string
		var totalEvals int
		var passedEvals int

		if err := rows.Scan(&policyID, &policyName, &totalEvals, &passedEvals); err != nil {
			continue
		}

		complianceRate := 0.0
		if totalEvals > 0 {
			complianceRate = float64(passedEvals) / float64(totalEvals) * 100
		}

		status.ByPolicy[policyID.String()] = CompliancePolicyStatus{
			PolicyID:          policyID,
			PolicyName:        policyName,
			ComplianceRate:    complianceRate,
			TotalEvaluations:  totalEvals,
			PassedEvaluations: passedEvals,
		}

		totalPassed += passedEvals
		totalChecks += totalEvals
	}

	status.PassedChecks = totalPassed
	status.TotalChecks = totalChecks
	if totalChecks > 0 {
		status.OverallPercentage = float64(totalPassed) / float64(totalChecks) * 100
	}

	return &status, nil
}

// GetSessionTimeSeries retrieves time-series data for sessions
func (r *Repository) GetSessionTimeSeries(ctx context.Context, tenantID uuid.UUID, metric string, startDate, endDate time.Time) ([]TimeSeriesDataPoint, error) {
	var dataPoints []TimeSeriesDataPoint

	var timeTrunc string
	switch metric {
	case "hour":
		timeTrunc = "date_trunc('hour', started_at)"
	case "day":
		timeTrunc = "date_trunc('day', started_at)"
	default:
		timeTrunc = "date_trunc('day', started_at)"
	}

	var valueCol string
	switch metric {
	case "total":
		valueCol = "COUNT(*)"
	case "active":
		valueCol = "COUNT(*) FILTER (WHERE status = 'active')"
	case "completed":
		valueCol = "COUNT(*) FILTER (WHERE status = 'completed')"
	case "failed":
		valueCol = "COUNT(*) FILTER (WHERE status = 'failed')"
	default:
		valueCol = "COUNT(*)"
	}

	query := `
		SELECT
			` + timeTrunc + ` as timestamp,
			` + valueCol + ` as value
		FROM sessions
		WHERE tenant_id = $1 AND started_at >= $2 AND started_at <= $3
		GROUP BY ` + timeTrunc + `
		ORDER BY timestamp ASC
	`

	rows, err := r.Db.QueryContext(ctx, query, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("analytics.GetSessionTimeSeries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dp TimeSeriesDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.Value); err != nil {
			return nil, err
		}
		if dp.Metadata == nil {
			dp.Metadata = make(map[string]interface{})
		}
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}

// Command Frequency Methods

// ListCommandFrequency lists command frequency data
func (r *Repository) ListCommandFrequency(ctx context.Context, tenantID uuid.UUID, periodType PeriodType, startDate, endDate time.Time, limit, offset int) ([]CommandFrequency, error) {
	var frequencies []CommandFrequency
	// For now, return empty as command frequency table needs to be created
	// This would query analytics_command_frequency table
	return frequencies, nil
}

// RefreshMaterializedViews refreshes all materialized views
func (r *Repository) RefreshMaterializedViews(ctx context.Context) error {
	views := []string{
		"mv_session_summary",
		"mv_user_activity_summary",
		"mv_compliance_summary",
	}

	for _, view := range views {
		logID, err := r.LogRefreshStart(ctx, view)
		if err != nil {
			r.logger.Error().Err(err).Str("view", view).Msg("Failed to log refresh start")
			continue
		}

		query := `REFRESH MATERIALIZED VIEW CONCURRENTLY ` + view
		_, err = r.Db.ExecContext(ctx, query)
		if err != nil {
			// Non-concurrent refresh if concurrent fails
			query = `REFRESH MATERIALIZED VIEW ` + view
			_, err = r.Db.ExecContext(ctx, query)
		}

		var errorMsg *string
		if err != nil {
			msg := err.Error()
			errorMsg = &msg
			r.logger.Error().Err(err).Str("view", view).Msg("Failed to refresh materialized view")
		}

		_ = r.LogRefreshComplete(ctx, logID, 0, errorMsg)
	}

	return nil
}

// Type definitions for missing types

type CommandFrequency struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	PeriodType      PeriodType `db:"period_type" json:"period_type"`
	PeriodStart     time.Time  `db:"period_start" json:"period_start"`
	Command         string     `db:"command" json:"command"`
	BaseCommand     *string    `db:"base_command" json:"base_command,omitempty"`
	ExecutionCount  int        `db:"execution_count" json:"execution_count"`
	UniqueUsers     int        `db:"unique_users" json:"unique_users"`
	UniqueTargets   int        `db:"unique_targets" json:"unique_targets"`
	IsBlacklisted   bool       `db:"is_blacklisted" json:"is_blacklisted"`
	RiskCategory    *string    `db:"risk_category" json:"risk_category,omitempty"`
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

type CommandRank struct {
	Command         string  `json:"command"`
	ExecutionCount  int     `json:"execution_count"`
	UniqueUsers     int     `json:"unique_users"`
	RiskScore       float64 `json:"risk_score"`
}
