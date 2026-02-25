package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// PostgresRepository implements Repository interface with PostgreSQL
type PostgresRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB, logger zerolog.Logger) *PostgresRepository {
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}
}

// Session Analytics Methods

func (r *PostgresRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	query := `
		INSERT INTO session_analytics (
			id, tenant_id, date, hour, total_sessions, active_sessions,
			completed_sessions, terminated_sessions, failed_sessions,
			ssh_sessions, rdp_sessions, database_sessions, kubernetes_sessions, web_sessions,
			avg_duration_seconds, min_duration_seconds, max_duration_seconds, total_duration_seconds,
			peak_concurrent_sessions, peak_concurrent_time, unique_users, unique_targets,
			total_recordings, recording_size_bytes, recording_duration_seconds,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :date, :hour, :total_sessions, :active_sessions,
			:completed_sessions, :terminated_sessions, :failed_sessions,
			:ssh_sessions, :rdp_sessions, :database_sessions, :kubernetes_sessions, :web_sessions,
			:avg_duration_seconds, :min_duration_seconds, :max_duration_seconds, :total_duration_seconds,
			:peak_concurrent_sessions, :peak_concurrent_time, :unique_users, :unique_targets,
			:total_recordings, :recording_size_bytes, :recording_duration_seconds,
			:metadata, :created_at, :updated_at
		)
		ON CONFLICT (tenant_id, date, hour) DO UPDATE SET
			total_sessions = EXCLUDED.total_sessions + session_analytics.total_sessions,
			active_sessions = EXCLUDED.active_sessions,
			completed_sessions = EXCLUDED.completed_sessions + session_analytics.completed_sessions,
			terminated_sessions = EXCLUDED.terminated_sessions + session_analytics.terminated_sessions,
			failed_sessions = EXCLUDED.failed_sessions + session_analytics.failed_sessions,
			ssh_sessions = EXCLUDED.ssh_sessions + session_analytics.ssh_sessions,
			rdp_sessions = EXCLUDED.rdp_sessions + session_analytics.rdp_sessions,
			database_sessions = EXCLUDED.database_sessions + session_analytics.database_sessions,
			kubernetes_sessions = EXCLUDED.kubernetes_sessions + session_analytics.kubernetes_sessions,
			web_sessions = EXCLUDED.web_sessions + session_analytics.web_sessions,
			total_duration_seconds = EXCLUDED.total_duration_seconds + session_analytics.total_duration_seconds,
			peak_concurrent_sessions = GREATEST(session_analytics.peak_concurrent_sessions, EXCLUDED.peak_concurrent_sessions),
			unique_users = GREATEST(session_analytics.unique_users, EXCLUDED.unique_users),
			unique_targets = GREATEST(session_analytics.unique_targets, EXCLUDED.unique_targets),
			total_recordings = EXCLUDED.total_recordings + session_analytics.total_recordings,
			recording_size_bytes = EXCLUDED.recording_size_bytes + session_analytics.recording_size_bytes,
			recording_duration_seconds = EXCLUDED.recording_duration_seconds + session_analytics.recording_duration_seconds,
			updated_at = NOW()
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("repository.CreateSessionAnalytics: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(analytics); err != nil {
			return fmt.Errorf("repository.CreateSessionAnalytics scan: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	query := `
		UPDATE session_analytics SET
			total_sessions = :total_sessions,
			active_sessions = :active_sessions,
			completed_sessions = :completed_sessions,
			terminated_sessions = :terminated_sessions,
			failed_sessions = :failed_sessions,
			ssh_sessions = :ssh_sessions,
			rdp_sessions = :rdp_sessions,
			database_sessions = :database_sessions,
			kubernetes_sessions = :kubernetes_sessions,
			web_sessions = :web_sessions,
			avg_duration_seconds = :avg_duration_seconds,
			min_duration_seconds = :min_duration_seconds,
			max_duration_seconds = :max_duration_seconds,
			total_duration_seconds = :total_duration_seconds,
			peak_concurrent_sessions = :peak_concurrent_sessions,
			peak_concurrent_time = :peak_concurrent_time,
			unique_users = :unique_users,
			unique_targets = :unique_targets,
			total_recordings = :total_recordings,
			recording_size_bytes = :recording_size_bytes,
			recording_duration_seconds = :recording_duration_seconds,
			metadata = :metadata,
			updated_at = NOW()
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("repository.UpdateSessionAnalytics: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	var analytics SessionAnalytics
	query := `SELECT * FROM session_analytics WHERE tenant_id = $1 AND date = $2 AND hour = $3`

	err := r.db.GetContext(ctx, &analytics, query, tenantID, date, hour)
	if err != nil {
		return nil, fmt.Errorf("repository.GetSessionAnalytics: %w", err)
	}

	return &analytics, nil
}

func (r *PostgresRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	baseQuery := `SELECT * FROM session_analytics WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND date >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND date <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	if filter.Hour != nil {
		baseQuery += fmt.Sprintf(" AND hour = $%d", argCount)
		args = append(args, *filter.Hour)
		argCount++
	}

	baseQuery += " ORDER BY date DESC, hour DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var analytics []SessionAnalytics
	err := r.db.SelectContext(ctx, &analytics, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListSessionAnalytics: %w", err)
	}

	return analytics, nil
}

// User Activity Methods

func (r *PostgresRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error {
	query := `
		INSERT INTO user_activity (
			id, tenant_id, user_id, date, hour, sessions_initiated, sessions_completed,
			commands_executed, targets_accessed, total_session_seconds, active_seconds,
			idle_seconds, first_access_time, last_access_time, peak_hour,
			country_code, city, off_hours_access, unusual_access, risk_score,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :user_id, :date, :hour, :sessions_initiated, :sessions_completed,
			:commands_executed, :targets_accessed, :total_session_seconds, :active_seconds,
			:idle_seconds, :first_access_time, :last_access_time, :peak_hour,
			:country_code, :city, :off_hours_access, :unusual_access, :risk_score,
			:metadata, :created_at, :updated_at
		)
		ON CONFLICT (tenant_id, user_id, date, hour) DO UPDATE SET
			sessions_initiated = user_activity.sessions_initiated + EXCLUDED.sessions_initiated,
			sessions_completed = user_activity.sessions_completed + EXCLUDED.sessions_completed,
			commands_executed = user_activity.commands_executed + EXCLUDED.commands_executed,
			targets_accessed = GREATEST(user_activity.targets_accessed, EXCLUDED.targets_accessed),
			total_session_seconds = user_activity.total_session_seconds + EXCLUDED.total_session_seconds,
			active_seconds = user_activity.active_seconds + EXCLUDED.active_seconds,
			idle_seconds = user_activity.idle_seconds + EXCLUDED.idle_seconds,
			first_access_time = LEAST(COALESCE(user_activity.first_access_time, EXCLUDED.first_access_time), EXCLUDED.first_access_time),
			last_access_time = GREATEST(COALESCE(user_activity.last_access_time, EXCLUDED.last_access_time), EXCLUDED.last_access_time),
			off_hours_access = user_activity.off_hours_access OR EXCLUDED.off_hours_access,
			unusual_access = user_activity.unusual_access OR EXCLUDED.unusual_access,
			risk_score = GREATEST(user_activity.risk_score, EXCLUDED.risk_score),
			updated_at = NOW()
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, activity)
	if err != nil {
		return fmt.Errorf("repository.CreateUserActivity: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(activity); err != nil {
			return fmt.Errorf("repository.CreateUserActivity scan: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error {
	query := `
		UPDATE user_activity SET
			sessions_initiated = :sessions_initiated,
			sessions_completed = :sessions_completed,
			commands_executed = :commands_executed,
			targets_accessed = :targets_accessed,
			total_session_seconds = :total_session_seconds,
			active_seconds = :active_seconds,
			idle_seconds = :idle_seconds,
			first_access_time = :first_access_time,
			last_access_time = :last_access_time,
			peak_hour = :peak_hour,
			off_hours_access = :off_hours_access,
			unusual_access = :unusual_access,
			risk_score = :risk_score,
			metadata = :metadata,
			updated_at = NOW()
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, activity)
	if err != nil {
		return fmt.Errorf("repository.UpdateUserActivity: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	var activity UserActivity
	query := `SELECT * FROM user_activity WHERE tenant_id = $1 AND user_id = $2 AND date = $3 AND hour = $4`

	err := r.db.GetContext(ctx, &activity, query, tenantID, userID, date, hour)
	if err != nil {
		return nil, fmt.Errorf("repository.GetUserActivity: %w", err)
	}

	return &activity, nil
}

func (r *PostgresRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	baseQuery := `SELECT * FROM user_activity WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND date >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND date <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	if filter.MinRiskScore != nil {
		baseQuery += fmt.Sprintf(" AND risk_score >= $%d", argCount)
		args = append(args, *filter.MinRiskScore)
		argCount++
	}

	if filter.OffHoursOnly != nil && *filter.OffHoursOnly {
		baseQuery += " AND off_hours_access = true"
	}

	if filter.UnusualOnly != nil && *filter.UnusualOnly {
		baseQuery += " AND unusual_access = true"
	}

	baseQuery += " ORDER BY date DESC, hour DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var activities []UserActivity
	err := r.db.SelectContext(ctx, &activities, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListUserActivity: %w", err)
	}

	return activities, nil
}

// Command Frequency Methods

func (r *PostgresRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error {
	query := `
		INSERT INTO command_frequency (
			tenant_id, date, hour, command_hash, command_pattern, base_command,
			session_id, user_id, target_host, risk_level, is_dangerous, is_blocked,
			executed_at, exit_code, execution_duration_ms, metadata
		) VALUES (
			:tenant_id, :date, :hour, :command_hash, :command_pattern, :base_command,
			:session_id, :user_id, :target_host, :risk_level, :is_dangerous, :is_blocked,
			:executed_at, :exit_code, :execution_duration_ms, :metadata
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, cmd)
	if err != nil {
		return fmt.Errorf("repository.RecordCommand: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	baseQuery := `SELECT * FROM command_frequency WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND date >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND date <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	if filter.BaseCommand != nil {
		baseQuery += fmt.Sprintf(" AND base_command = $%d", argCount)
		args = append(args, *filter.BaseCommand)
		argCount++
	}

	if filter.RiskLevel != nil {
		baseQuery += fmt.Sprintf(" AND risk_level = $%d", argCount)
		args = append(args, *filter.RiskLevel)
		argCount++
	}

	if filter.IsDangerous != nil {
		baseQuery += fmt.Sprintf(" AND is_dangerous = $%d", argCount)
		args = append(args, *filter.IsDangerous)
		argCount++
	}

	baseQuery += " ORDER BY executed_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var commands []CommandFrequency
	err := r.db.SelectContext(ctx, &commands, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListCommandFrequency: %w", err)
	}

	return commands, nil
}

func (r *PostgresRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	query := `
		SELECT
			base_command as command,
			COUNT(*) as count,
			MAX(risk_level) as risk_level
		FROM command_frequency
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		GROUP BY base_command
		ORDER BY count DESC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, dateFrom, dateTo, limit)
	if err != nil {
		return nil, fmt.Errorf("repository.GetTopCommands: %w", err)
	}
	defer rows.Close()

	var ranks []CommandRank
	for rows.Next() {
		var rank CommandRank
		if err := rows.Scan(&rank.Command, &rank.Count, &rank.RiskLevel); err != nil {
			return nil, fmt.Errorf("repository.GetTopCommands scan: %w", err)
		}
		ranks = append(ranks, rank)
	}

	return ranks, nil
}

// Compliance Methods

func (r *PostgresRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	report.ID = uuid.New()
	report.CreatedAt = time.Now()

	query := `
		INSERT INTO compliance_report (
			id, tenant_id, report_name, framework, version, generated_at, generated_by,
			status, overall_score, total_controls, passed_controls, failed_controls,
			skipped_controls, period_start, period_end, summary, findings, recommendations, metadata
		) VALUES (
			:id, :tenant_id, :report_name, :framework, :version, :generated_at, :generated_by,
			:status, :overall_score, :total_controls, :passed_controls, :failed_controls,
			:skipped_controls, :period_start, :period_end, :summary, :findings, :recommendations, :metadata
		)
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, report)
	if err != nil {
		return fmt.Errorf("repository.CreateComplianceReport: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(report); err != nil {
			return fmt.Errorf("repository.CreateComplianceReport scan: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	var report ComplianceReport
	query := `SELECT * FROM compliance_reports WHERE id = $1`

	err := r.db.GetContext(ctx, &report, query, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetComplianceReport: %w", err)
	}

	return &report, nil
}

func (r *PostgresRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	baseQuery := `SELECT * FROM compliance_reports WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.Framework != nil {
		baseQuery += fmt.Sprintf(" AND framework = $%d", argCount)
		args = append(args, *filter.Framework)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND generated_at >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND generated_at <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	baseQuery += " ORDER BY generated_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var reports []ComplianceReport
	err := r.db.SelectContext(ctx, &reports, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListComplianceReports: %w", err)
	}

	return reports, nil
}

func (r *PostgresRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error {
	evaluation.ID = uuid.New()
	evaluation.EvaluatedAt = time.Now()

	query := `
		INSERT INTO compliance_control_evaluations (
			id, report_id, tenant_id, control_id, control_name, control_category,
			status, score, evidence_count, evidence_urls, findings, remediation_steps,
			metadata, evaluated_at
		) VALUES (
			:id, :report_id, :tenant_id, :control_id, :control_name, :control_category,
			:status, :score, :evidence_count, :evidence_urls, :findings, :remediation_steps,
			:metadata, :evaluated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, evaluation)
	if err != nil {
		return fmt.Errorf("repository.CreateControlEvaluation: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	var evaluations []ComplianceControlEvaluation
	query := `SELECT * FROM compliance_control_evaluations WHERE report_id = $1 ORDER BY control_id`

	err := r.db.SelectContext(ctx, &evaluations, query, reportID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListControlEvaluations: %w", err)
	}

	return evaluations, nil
}

func (r *PostgresRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	exception.ID = uuid.New()
	exception.CreatedAt = time.Now()
	exception.UpdatedAt = time.Now()
	exception.RequestedAt = time.Now()

	query := `
		INSERT INTO compliance_exceptions (
			id, tenant_id, control_id, control_name, framework, status, risk_level,
			requested_by, requested_at, approved_by, approved_at, expires_at,
			justification, business_reason, compensating_controls,
			risk_accepted_by, risk_accepted_at, review_date, review_notes, metadata,
			created_at, updated_at
		) VALUES (
			:id, :tenant_id, :control_id, :control_name, :framework, :status, :risk_level,
			:requested_by, :requested_at, :approved_by, :approved_at, :expires_at,
			:justification, :business_reason, :compensating_controls,
			:risk_accepted_by, :risk_accepted_at, :review_date, :review_notes, :metadata,
			:created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, exception)
	if err != nil {
		return fmt.Errorf("repository.CreateComplianceException: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	var exceptions []ComplianceException
	query := `
		SELECT * FROM compliance_exceptions
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &exceptions, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListComplianceExceptions: %w", err)
	}

	return exceptions, nil
}

// Anomaly Detection Methods

func (r *PostgresRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	anomaly.ID = uuid.New()
	anomaly.CreatedAt = time.Now()
	anomaly.UpdatedAt = time.Now()
	anomaly.DetectedAt = time.Now()

	query := `
		INSERT INTO anomaly_detections (
			id, tenant_id, anomaly_type, user_id, session_id, target_host,
			severity, confidence_score, risk_score, title, description,
			indicators, detection_method, detected_at, model_version, status,
			assigned_to, resolution_notes, resolved_at, resolved_by,
			auto_triggered, auto_action_taken, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :anomaly_type, :user_id, :session_id, :target_host,
			:severity, :confidence_score, :risk_score, :title, :description,
			:indicators, :detection_method, :detected_at, :model_version, :status,
			:assigned_to, :resolution_notes, :resolved_at, :resolved_by,
			:auto_triggered, :auto_action_taken, :metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("repository.CreateAnomalyDetection: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	var anomaly AnomalyDetection
	query := `SELECT * FROM anomaly_detections WHERE id = $1`

	err := r.db.GetContext(ctx, &anomaly, query, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetAnomalyDetection: %w", err)
	}

	return &anomaly, nil
}

func (r *PostgresRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	baseQuery := `SELECT * FROM anomaly_detections WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND tenant_id = $%d", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.AnomalyType != nil {
		baseQuery += fmt.Sprintf(" AND anomaly_type = $%d", argCount)
		args = append(args, *filter.AnomalyType)
		argCount++
	}

	if filter.Severity != nil {
		baseQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
		argCount++
	}

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.DateFrom != nil {
		baseQuery += fmt.Sprintf(" AND detected_at >= $%d", argCount)
		args = append(args, *filter.DateFrom)
		argCount++
	}

	if filter.DateTo != nil {
		baseQuery += fmt.Sprintf(" AND detected_at <= $%d", argCount)
		args = append(args, *filter.DateTo)
		argCount++
	}

	baseQuery += " ORDER BY detected_at DESC"

	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	var anomalies []AnomalyDetection
	err := r.db.SelectContext(ctx, &anomalies, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListAnomalyDetections: %w", err)
	}

	return anomalies, nil
}

func (r *PostgresRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	query := `
		UPDATE anomaly_detections SET
			status = :status,
			assigned_to = :assigned_to,
			resolution_notes = :resolution_notes,
			resolved_at = :resolved_at,
			resolved_by = :resolved_by,
			updated_at = NOW()
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, anomaly)
	if err != nil {
		return fmt.Errorf("repository.UpdateAnomalyDetection: %w", err)
	}

	return nil
}

// Ransomware Event Methods

func (r *PostgresRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	query := `
		INSERT INTO ransomware_events (
			id, tenant_id, detection_id, encryption_activity, mass_file_modification,
			suspicious_processes, affected_paths, files_affected, systems_affected,
			data_exfiltrated, emergency_triggered, sessions_terminated, credentials_revoked,
			containment_status, recovery_status, raw_indicators, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :detection_id, :encryption_activity, :mass_file_modification,
			:suspicious_processes, :affected_paths, :files_affected, :systems_affected,
			:data_exfiltrated, :emergency_triggered, :sessions_terminated, :credentials_revoked,
			:containment_status, :recovery_status, :raw_indicators, :metadata, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("repository.CreateRansomwareEvent: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	var event RansomwareEvent
	query := `SELECT * FROM ransomware_events WHERE id = $1`

	err := r.db.GetContext(ctx, &event, query, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetRansomwareEvent: %w", err)
	}

	return &event, nil
}

func (r *PostgresRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	var events []RansomwareEvent
	query := `
		SELECT * FROM ransomware_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.db.SelectContext(ctx, &events, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("repository.ListRansomwareEvents: %w", err)
	}

	return events, nil
}

func (r *PostgresRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	event.UpdatedAt = time.Now()

	query := `
		UPDATE ransomware_events SET
			encryption_activity = :encryption_activity,
			mass_file_modification = :mass_file_modification,
			suspicious_processes = :suspicious_processes,
			affected_paths = :affected_paths,
			files_affected = :files_affected,
			systems_affected = :systems_affected,
			data_exfiltrated = :data_exfiltrated,
			emergency_triggered = :emergency_triggered,
			sessions_terminated = :sessions_terminated,
			credentials_revoked = :credentials_revoked,
			containment_status = :containment_status,
			recovery_status = :recovery_status,
			raw_indicators = :raw_indicators,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("repository.UpdateRansomwareEvent: %w", err)
	}

	return nil
}

// Command Blacklist Methods

func (r *PostgresRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	blacklist.ID = uuid.New()
	blacklist.CreatedAt = time.Now()
	blacklist.UpdatedAt = time.Now()

	query := `
		INSERT INTO command_blacklist (
			id, tenant_id, command_pattern, pattern_type, base_command, action, severity,
			applies_to_users, applies_to_groups, applies_to_targets, allow_override,
			override_roles, reason, risk_category, enabled, created_by, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :command_pattern, :pattern_type, :base_command, :action, :severity,
			:applies_to_users, :applies_to_groups, :applies_to_targets, :allow_override,
			:override_roles, :reason, :risk_category, :enabled, :created_by, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, blacklist)
	if err != nil {
		return fmt.Errorf("repository.CreateCommandBlacklist: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	var blacklist CommandBlacklist
	query := `SELECT * FROM command_blacklist WHERE id = $1`

	err := r.db.GetContext(ctx, &blacklist, query, id)
	if err != nil {
		return nil, fmt.Errorf("repository.GetCommandBlacklist: %w", err)
	}

	return &blacklist, nil
}

func (r *PostgresRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	var blacklists []CommandBlacklist
	var query string
	var args []interface{}

	if tenantID == nil {
		query = `SELECT * FROM command_blacklist WHERE tenant_id IS NULL AND enabled = true ORDER BY created_at DESC`
	} else {
		query = `SELECT * FROM command_blacklist WHERE (tenant_id IS NULL OR tenant_id = $1) AND enabled = true ORDER BY tenant_id NULLS LAST, created_at DESC`
		args = []interface{}{*tenantID}
	}

	err := r.db.SelectContext(ctx, &blacklists, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ListCommandBlacklist: %w", err)
	}

	return blacklists, nil
}

func (r *PostgresRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	blacklist.UpdatedAt = time.Now()

	query := `
		UPDATE command_blacklist SET
			command_pattern = :command_pattern,
			pattern_type = :pattern_type,
			base_command = :base_command,
			action = :action,
			severity = :severity,
			applies_to_users = :applies_to_users,
			applies_to_groups = :applies_to_groups,
			applies_to_targets = :applies_to_targets,
			allow_override = :allow_override,
			override_roles = :override_roles,
			reason = :reason,
			risk_category = :risk_category,
			enabled = :enabled,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, blacklist)
	if err != nil {
		return fmt.Errorf("repository.UpdateCommandBlacklist: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM command_blacklist WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository.DeleteCommandBlacklist: %w", err)
	}

	return nil
}

func (r *PostgresRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	// First, get all applicable blacklist rules for this tenant
	blacklists, err := r.ListCommandBlacklist(ctx, &tenantID)
	if err != nil {
		return nil, err
	}

	var matches []CommandBlacklist

	for _, bl := range blacklists {
		// Check if rule applies to this user/group
		if len(bl.AppliesToUsers) > 0 {
			userMatch := false
			for _, uid := range userIDs {
				for _, allowedUID := range bl.AppliesToUsers {
					if uid == allowedUID {
						userMatch = true
						break
					}
				}
				if userMatch {
					break
				}
			}
			if !userMatch {
				continue
			}
		}

		if len(bl.AppliesToGroups) > 0 {
			groupMatch := false
			for _, gid := range groupIDs {
				for _, allowedGID := range bl.AppliesToGroups {
					if gid == allowedGID {
						groupMatch = true
						break
					}
				}
				if groupMatch {
					break
				}
			}
			if !groupMatch {
				continue
			}
		}

		// Check if command matches pattern
		matches, err = r.commandMatchesPattern(command, &bl, matches)
		if err != nil {
			continue
		}
	}

	return matches, nil
}

func (r *PostgresRepository) commandMatchesPattern(command string, bl *CommandBlacklist, matches []CommandBlacklist) ([]CommandBlacklist, error) {
	// Pattern matching logic based on pattern_type
	switch bl.PatternType {
	case string(PatternTypeExact):
		if command == bl.CommandPattern {
			matches = append(matches, *bl)
		}
	case string(PatternTypeGlob):
		// Use proper glob matching with filepath.Match
		matched, err := globMatch(command, bl.CommandPattern)
		if err == nil && matched {
			matches = append(matches, *bl)
		}
	case string(PatternTypeRegex):
		// Use proper regex matching
		matched, err := regexMatch(command, bl.CommandPattern)
		if err == nil && matched {
			matches = append(matches, *bl)
		}
	}
	return matches, nil
}

// regexCache caches compiled regex patterns for performance
var (
	regexCache = make(map[string]*regexp.Regexp)
	regexMu    sync.RWMutex
)

// regexMatch performs regex pattern matching with caching
func regexMatch(command, pattern string) (bool, error) {
	regexMu.RLock()
	re, exists := regexCache[pattern]
	regexMu.RUnlock()

	if !exists {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}

		regexMu.Lock()
		regexCache[pattern] = re
		regexMu.Unlock()
	}

	return re.MatchString(command), nil
}

// globMatch performs glob pattern matching using filepath.Match
func globMatch(command, pattern string) (bool, error) {
	// filepath.Match requires pattern to be a valid glob pattern
	// It supports * (matches any sequence) and ? (matches single character)
	matched, err := filepath.Match(pattern, command)
	if err != nil {
		// Invalid pattern - treat as no match
		return false, nil
	}
	return matched, nil
}

// SSH Key Analytics Methods

func (r *PostgresRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	analytics.ID = uuid.New()
	analytics.CreatedAt = time.Now()
	analytics.UpdatedAt = time.Now()

	query := `
		INSERT INTO ssh_key_analytics (
			id, tenant_id, ssh_key_id, date, usage_count, unique_users, unique_targets,
			first_use_time, last_use_time, avg_session_duration_seconds,
			off_hours_usage, unusual_source_usage, failed_attempts, metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :ssh_key_id, :date, :usage_count, :unique_users, :unique_targets,
			:first_use_time, :last_use_time, :avg_session_duration_seconds,
			:off_hours_usage, :unusual_source_usage, :failed_attempts, :metadata, :created_at, :updated_at
		)
		ON CONFLICT (tenant_id, ssh_key_id, date) DO UPDATE SET
			usage_count = ssh_key_analytics.usage_count + EXCLUDED.usage_count,
			unique_users = GREATEST(ssh_key_analytics.unique_users, EXCLUDED.unique_users),
			unique_targets = GREATEST(ssh_key_analytics.unique_targets, EXCLUDED.unique_targets),
			first_use_time = LEAST(COALESCE(ssh_key_analytics.first_use_time, EXCLUDED.first_use_time), EXCLUDED.first_use_time),
			last_use_time = GREATEST(COALESCE(ssh_key_analytics.last_use_time, EXCLUDED.last_use_time), EXCLUDED.last_use_time),
			off_hours_usage = ssh_key_analytics.off_hours_usage + EXCLUDED.off_hours_usage,
			unusual_source_usage = ssh_key_analytics.unusual_source_usage + EXCLUDED.unusual_source_usage,
			failed_attempts = ssh_key_analytics.failed_attempts + EXCLUDED.failed_attempts,
			updated_at = NOW()
		RETURNING *
	`

	rows, err := r.db.NamedQueryContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("repository.CreateSSHKeyAnalytics: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.StructScan(analytics); err != nil {
			return fmt.Errorf("repository.CreateSSHKeyAnalytics scan: %w", err)
		}
	}

	return nil
}

func (r *PostgresRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	analytics.UpdatedAt = time.Now()

	query := `
		UPDATE ssh_key_analytics SET
			usage_count = :usage_count,
			unique_users = :unique_users,
			unique_targets = :unique_targets,
			first_use_time = :first_use_time,
			last_use_time = :last_use_time,
			avg_session_duration_seconds = :avg_session_duration_seconds,
			off_hours_usage = :off_hours_usage,
			unusual_source_usage = :unusual_source_usage,
			failed_attempts = :failed_attempts,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("repository.UpdateSSHKeyAnalytics: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	var analytics SSHKeyAnalytics
	query := `SELECT * FROM ssh_key_analytics WHERE tenant_id = $1 AND ssh_key_id = $2 AND date = $3`

	err := r.db.GetContext(ctx, &analytics, query, tenantID, sshKeyID, date)
	if err != nil {
		return nil, fmt.Errorf("repository.GetSSHKeyAnalytics: %w", err)
	}

	return &analytics, nil
}

func (r *PostgresRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	var analytics []SSHKeyAnalytics
	query := `
		SELECT * FROM ssh_key_analytics
		WHERE tenant_id = $1 AND ssh_key_id = $2 AND date >= $3 AND date <= $4
		ORDER BY date DESC
	`

	err := r.db.SelectContext(ctx, &analytics, query, tenantID, sshKeyID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("repository.ListSSHKeyAnalytics: %w", err)
	}

	return analytics, nil
}

// Dashboard Metrics

func (r *PostgresRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	metrics := &DashboardMetrics{
		Timestamp: time.Now(),
	}

	// Get session summary from materialized view
	sessionQuery := `
		SELECT * FROM mv_session_summary
		WHERE tenant_id = $1
		ORDER BY date DESC
		LIMIT 1
	`
	var sessionSummary struct {
		TotalSessions  int     `db:"total_sessions"`
		ActiveSessions int     `db:"active_sessions"`
		AvgDuration    float64 `db:"avg_duration_seconds"`
		UniqueUsers    int     `db:"unique_users"`
		UniqueTargets  int     `db:"unique_targets"`
	}

	if err := r.db.GetContext(ctx, &sessionSummary, sessionQuery, tenantID); err == nil {
		metrics.SessionMetrics = &SessionSummary{
			TotalSessions:  sessionSummary.TotalSessions,
			ActiveSessions: sessionSummary.ActiveSessions,
			AvgDuration:    sessionSummary.AvgDuration,
			SessionsByType: make(map[string]int),
		}
	}

	// Get user activity summary
	activityQuery := `
		SELECT
			COUNT(DISTINCT user_id) as active_users,
			SUM(commands_executed) as total_commands,
			COUNT(*) FILTER (WHERE risk_score > 50) as high_risk_users,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_access
		FROM user_activity
		WHERE tenant_id = $1 AND date >= CURRENT_DATE
	`
	var userSummary struct {
		ActiveUsers   int `db:"active_users"`
		TotalCommands int `db:"total_commands"`
		HighRiskUsers int `db:"high_risk_users"`
		OffHoursAccess int `db:"off_hours_access"`
	}

	if err := r.db.GetContext(ctx, &userSummary, activityQuery, tenantID); err == nil {
		metrics.UserActivity = &UserActivitySummary{
			ActiveUsers:    userSummary.ActiveUsers,
			TotalCommands:  userSummary.TotalCommands,
			HighRiskUsers:  userSummary.HighRiskUsers,
			OffHoursAccess: userSummary.OffHoursAccess,
		}
	}

	// Get command metrics
	commandQuery := `
		SELECT
			COUNT(*) as total_commands,
			COUNT(*) FILTER (WHERE risk_level IN ('high', 'critical')) as high_risk_commands,
			COUNT(*) FILTER (WHERE is_blocked = true) as blocked_commands
		FROM command_frequency
		WHERE tenant_id = $1 AND date >= CURRENT_DATE
	`
	var commandSummary struct {
		TotalCommands     int `db:"total_commands"`
		HighRiskCommands  int `db:"high_risk_commands"`
		BlockedCommands   int `db:"blocked_commands"`
	}

	if err := r.db.GetContext(ctx, &commandSummary, commandQuery, tenantID); err == nil {
		metrics.CommandMetrics = &CommandSummary{
			TotalCommands:    commandSummary.TotalCommands,
			HighRiskCommands: commandSummary.HighRiskCommands,
			BlockedCommands:  commandSummary.BlockedCommands,
		}
	}

	// Get recent anomalies
	anomalies, _ := r.ListAnomalyDetections(ctx, AnomalyFilter{
		TenantID: &tenantID,
		Status:   stringPtr(string(AnomalyStatusOpen)),
	}, 10, 0)
	metrics.RecentAnomalies = anomalies

	// Get compliance status
	complianceQuery := `
		SELECT
			COALESCE(AVG(overall_score), 0) as overall_score,
			SUM(passed_controls) as passed_controls,
			SUM(failed_controls) as failed_controls
		FROM compliance_reports
		WHERE tenant_id = $1 AND generated_at >= CURRENT_DATE - INTERVAL '30 days'
	`
	var complianceSummary struct {
		OverallScore   float64 `db:"overall_score"`
		PassedControls int     `db:"passed_controls"`
		FailedControls int     `db:"failed_controls"`
	}

	if err := r.db.GetContext(ctx, &complianceSummary, complianceQuery, tenantID); err == nil {
		metrics.ComplianceStatus = &ComplianceSummary{
			OverallScore:   complianceSummary.OverallScore,
			PassedControls: complianceSummary.PassedControls,
			FailedControls: complianceSummary.FailedControls,
			Frameworks:     make(map[string]FrameworkStatus),
		}
	}

	return metrics, nil
}

// GetSessionMetrics is an alias for GetSessionAnalytics for cache warming compatibility
func (r *PostgresRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return r.GetSessionAnalytics(ctx, tenantID, date, hour)
}

// Materialized view refresh
func (r *PostgresRepository) RefreshMaterializedViews(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_session_summary")
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to refresh mv_session_summary")
	}

	_, err = r.db.ExecContext(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_risk_summary")
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to refresh mv_user_risk_summary")
	}

	return nil
}

// GetTenantSessionSummary aggregates session data for a tenant
func (r *PostgresRepository) GetTenantSessionSummary(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time) (*SessionSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(total_sessions), 0) as total_sessions,
			COALESCE(SUM(active_sessions), 0) as active_sessions,
			COALESCE(AVG(avg_duration_seconds), 0) as avg_duration,
			COALESCE(MAX(peak_concurrent_sessions), 0) as peak_concurrent
		FROM session_analytics
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
	`

	var summary struct {
		TotalSessions  int     `db:"total_sessions"`
		ActiveSessions int     `db:"active_sessions"`
		AvgDuration    float64 `db:"avg_duration"`
		PeakConcurrent int     `db:"peak_concurrent"`
	}

	err := r.db.GetContext(ctx, &summary, query, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("repository.GetTenantSessionSummary: %w", err)
	}

	return &SessionSummary{
		TotalSessions:  summary.TotalSessions,
		ActiveSessions: summary.ActiveSessions,
		AvgDuration:    summary.AvgDuration,
		PeakConcurrent: summary.PeakConcurrent,
		SessionsByType: make(map[string]int),
	}, nil
}

// BatchUpsertSessionAnalytics performs batch upsert for session analytics
func (r *PostgresRepository) BatchUpsertSessionAnalytics(ctx context.Context, analytics []SessionAnalytics) error {
	if len(analytics) == 0 {
		return nil
	}

	query := `
		INSERT INTO session_analytics (
			id, tenant_id, date, hour, total_sessions, active_sessions,
			completed_sessions, terminated_sessions, failed_sessions,
			ssh_sessions, rdp_sessions, database_sessions, kubernetes_sessions, web_sessions,
			avg_duration_seconds, min_duration_seconds, max_duration_seconds, total_duration_seconds,
			peak_concurrent_sessions, peak_concurrent_time, unique_users, unique_targets,
			total_recordings, recording_size_bytes, recording_duration_seconds,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :date, :hour, :total_sessions, :active_sessions,
			:completed_sessions, :terminated_sessions, :failed_sessions,
			:ssh_sessions, :rdp_sessions, :database_sessions, :kubernetes_sessions, :web_sessions,
			:avg_duration_seconds, :min_duration_seconds, :max_duration_seconds, :total_duration_seconds,
			:peak_concurrent_sessions, :peak_concurrent_time, :unique_users, :unique_targets,
			:total_recordings, :recording_size_bytes, :recording_duration_seconds,
			:metadata, :created_at, :updated_at
		)
		ON CONFLICT (tenant_id, date, hour) DO UPDATE SET
			total_sessions = session_analytics.total_sessions + EXCLUDED.total_sessions,
			active_sessions = EXCLUDED.active_sessions,
			completed_sessions = session_analytics.completed_sessions + EXCLUDED.completed_sessions,
			terminated_sessions = session_analytics.terminated_sessions + EXCLUDED.terminated_sessions,
			failed_sessions = session_analytics.failed_sessions + EXCLUDED.failed_sessions,
			ssh_sessions = session_analytics.ssh_sessions + EXCLUDED.ssh_sessions,
			rdp_sessions = session_analytics.rdp_sessions + EXCLUDED.rdp_sessions,
			database_sessions = session_analytics.database_sessions + EXCLUDED.database_sessions,
			kubernetes_sessions = session_analytics.kubernetes_sessions + EXCLUDED.kubernetes_sessions,
			web_sessions = session_analytics.web_sessions + EXCLUDED.web_sessions,
			total_duration_seconds = session_analytics.total_duration_seconds + EXCLUDED.total_duration_seconds,
			peak_concurrent_sessions = GREATEST(session_analytics.peak_concurrent_sessions, EXCLUDED.peak_concurrent_sessions),
			unique_users = GREATEST(session_analytics.unique_users, EXCLUDED.unique_users),
			unique_targets = GREATEST(session_analytics.unique_targets, EXCLUDED.unique_targets),
			total_recordings = session_analytics.total_recordings + EXCLUDED.total_recordings,
			recording_size_bytes = session_analytics.recording_size_bytes + EXCLUDED.recording_size_bytes,
			recording_duration_seconds = session_analytics.recording_duration_seconds + EXCLUDED.recording_duration_seconds,
			updated_at = NOW()
	`

	_, err := r.db.NamedExecContext(ctx, query, analytics)
	if err != nil {
		return fmt.Errorf("repository.BatchUpsertSessionAnalytics: %w", err)
	}

	return nil
}

// GetUserRiskScore calculates a composite risk score for a user
func (r *PostgresRepository) GetUserRiskScore(ctx context.Context, tenantID, userID uuid.UUID, days int) (float64, error) {
	query := `
		SELECT
			COALESCE(AVG(risk_score), 0) as avg_risk_score,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count,
			COUNT(*) FILTER (WHERE unusual_access) as unusual_count
		FROM user_activity
		WHERE tenant_id = $1 AND user_id = $2 AND date >= CURRENT_DATE - INTERVAL '1 day' * $3
	`

	var result struct {
		AvgRiskScore  float64 `db:"avg_risk_score"`
		OffHoursCount int     `db:"off_hours_count"`
		UnusualCount  int     `db:"unusual_count"`
	}

	err := r.db.GetContext(ctx, &result, query, tenantID, userID, days)
	if err != nil {
		return 0, fmt.Errorf("repository.GetUserRiskScore: %w", err)
	}

	// Calculate composite score
	riskScore := result.AvgRiskScore
	riskScore += float64(result.OffHoursCount) * 5
	riskScore += float64(result.UnusualCount) * 10

	// Cap at 100
	if riskScore > 100 {
		riskScore = 100
	}

	return riskScore, nil
}

// GetAnomalyStats returns statistics about anomalies for a tenant
func (r *PostgresRepository) GetAnomalyStats(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT
			status,
			COUNT(*) as count
		FROM anomaly_detections
		WHERE tenant_id = $1
		GROUP BY status
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetAnomalyStats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}

// ExecuteAnalyticsQuery executes a custom analytics query
func (r *PostgresRepository) ExecuteAnalyticsQuery(ctx context.Context, tenantID uuid.UUID, query string, args ...interface{}) ([]map[string]interface{}, error) {
	// Security: Validate that query is read-only and only accesses analytics tables
	// In production, add query validation here

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository.ExecuteAnalyticsQuery: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, nil
}

// GetTimeSeriesData retrieves time-series data for charts
func (r *PostgresRepository) GetTimeSeriesData(ctx context.Context, tenantID uuid.UUID, metric string, dateFrom, dateTo time.Time) ([]DataPoint, error) {
	// Validate metric to prevent SQL injection
	validMetrics := map[string]string{
		"sessions":          "total_sessions",
		"active_sessions":   "active_sessions",
		"commands":          "commands_executed",
		"unique_users":      "unique_users",
		"avg_duration":      "avg_duration_seconds",
	}

	col, ok := validMetrics[metric]
	if !ok {
		return nil, fmt.Errorf("invalid metric: %s", metric)
	}

	query := fmt.Sprintf(`
		SELECT date, SUM(%s) as value
		FROM session_analytics
		WHERE tenant_id = $1 AND date >= $2 AND date <= $3
		GROUP BY date
		ORDER BY date ASC
	`, col)

	rows, err := r.db.QueryContext(ctx, query, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("repository.GetTimeSeriesData: %w", err)
	}
	defer rows.Close()

	var dataPoints []DataPoint
	for rows.Next() {
		var dp DataPoint
		var value float64
		if err := rows.Scan(&dp.Timestamp, &value); err != nil {
			return nil, err
		}
		dp.Value = value
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}

// MarshalJSON helpers for complex types
func (a *SessionAnalytics) GetMetadataMap() (map[string]interface{}, error) {
	if len(a.Metadata) == 0 {
		return nil, nil
	}
	var m map[string]interface{}
	err := json.Unmarshal(a.Metadata, &m)
	return m, err
}

func (a *SessionAnalytics) SetMetadataMap(m map[string]interface{}) error {
	if m == nil {
		a.Metadata = nil
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	a.Metadata = data
	return nil
}
