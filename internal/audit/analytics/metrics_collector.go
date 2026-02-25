// Package analytics provides metrics collection for audit service
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

// MetricsStore handles collection and storage of session metrics
type MetricsStore struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore(db *sqlx.DB, logger zerolog.Logger) *MetricsStore {
	return &MetricsStore{
		db:     db,
		logger: logger,
	}
}

// SessionMetrics represents metrics for a single session
type SessionMetrics struct {
	SessionID        uuid.UUID `json:"session_id" db:"session_id"`
	TenantID         uuid.UUID `json:"tenant_id" db:"tenant_id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	TargetHost       string    `json:"target_host" db:"target_host"`
	StartTime        time.Time `json:"start_time" db:"start_time"`
	EndTime          *time.Time `json:"end_time,omitempty" db:"end_time"`
	DurationSeconds  float64   `json:"duration_seconds" db:"duration_seconds"`
	CommandCount     int       `json:"command_count" db:"command_count"`
	FailedCommands   int       `json:"failed_commands" db:"failed_commands"`
	OffHoursAccess   bool      `json:"off_hours_access" db:"off_hours_access"`
	WeekendAccess    bool      `json:"weekend_access" db:"weekend_access"`
	FailureRate      float64   `json:"failure_rate" db:"failure_rate"`
	Commands         json.RawMessage `json:"commands,omitempty" db:"commands"`
	Metadata         json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CollectedAt      time.Time `json:"collected_at" db:"collected_at"`
}

// RecordSessionMetrics records metrics for a completed session
func (s *MetricsStore) RecordSessionMetrics(ctx context.Context, metrics *SessionMetrics) error {
	metrics.CollectedAt = time.Now()

	query := `
		INSERT INTO session_analytics (
			session_id, tenant_id, user_id, target_host,
			start_time, end_time, duration_seconds,
			command_count, failed_commands,
			off_hours_access, weekend_access,
			failure_rate, commands, metadata, collected_at
		) VALUES (
			:session_id, :tenant_id, :user_id, :target_host,
			:start_time, :end_time, :duration_seconds,
			:command_count, :failed_commands,
			:off_hours_access, :weekend_access,
			:failure_rate, :commands, :metadata, :collected_at
		)
		ON CONFLICT (session_id) DO UPDATE SET
			end_time = EXCLUDED.end_time,
			duration_seconds = EXCLUDED.duration_seconds,
			command_count = EXCLUDED.command_count,
			failed_commands = EXCLUDED.failed_commands,
			failure_rate = EXCLUDED.failure_rate,
			commands = EXCLUDED.commands,
			collected_at = EXCLUDED.collected_at
	`

	_, err := s.db.NamedExecContext(ctx, query, metrics)
	if err != nil {
		return fmt.Errorf("metrics_store.RecordSessionMetrics: %w", err)
	}

	s.logger.Debug().
		Str("session_id", metrics.SessionID.String()).
		Str("user_id", metrics.UserID.String()).
		Int("command_count", metrics.CommandCount).
		Msg("Session metrics recorded")

	return nil
}

// GetSessionMetrics retrieves metrics for a specific session
func (s *MetricsStore) GetSessionMetrics(ctx context.Context, sessionID uuid.UUID) (*SessionMetrics, error) {
	var metrics SessionMetrics
	query := `SELECT * FROM session_analytics WHERE session_id = $1`

	err := s.db.GetContext(ctx, &metrics, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetSessionMetrics: %w", err)
	}

	return &metrics, nil
}

// GetUserMetrics retrieves aggregated metrics for a user
func (s *MetricsStore) GetUserMetrics(ctx context.Context, tenantID, userID uuid.UUID, days int) (*UserMetricsSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_sessions,
			SUM(command_count) as total_commands,
			AVG(duration_seconds) as avg_duration,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_sessions,
			COUNT(DISTINCT target_host) as unique_targets
		FROM session_analytics
		WHERE tenant_id = $1 AND user_id = $2
			AND start_time >= NOW() - INTERVAL '1 day' * $3
	`

	var summary UserMetricsSummary
	err := s.db.GetContext(ctx, &summary, query, tenantID, userID, days)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetUserMetrics: %w", err)
	}

	return &summary, nil
}

// AggregateDailyMetrics aggregates daily metrics for all users
func (s *MetricsStore) AggregateDailyMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time) error {
	query := `
		INSERT INTO daily_user_metrics (
			tenant_id, user_id, metric_date,
			total_sessions, completed_sessions, failed_sessions,
			total_commands, avg_commands_per_session,
			total_duration_seconds, avg_session_duration,
			unique_targets, off_hours_sessions, weekend_sessions,
			failed_command_rate
		)
		SELECT
			$1::uuid as tenant_id,
			user_id,
			$2::date as metric_date,
			COUNT(*) as total_sessions,
			COUNT(*) FILTER (WHERE end_time IS NOT NULL) as completed_sessions,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_sessions,
			SUM(command_count) as total_commands,
			AVG(command_count) as avg_commands_per_session,
			SUM(duration_seconds) as total_duration_seconds,
			AVG(duration_seconds) as avg_session_duration,
			COUNT(DISTINCT target_host) as unique_targets,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_sessions,
			COUNT(*) FILTER (WHERE weekend_access) as weekend_sessions,
			AVG(failure_rate) as failed_command_rate
		FROM session_analytics
		WHERE tenant_id = $1
			AND DATE(start_time) = $2::date
		GROUP BY user_id
		ON CONFLICT (tenant_id, user_id, metric_date) DO UPDATE SET
			total_sessions = EXCLUDED.total_sessions,
			completed_sessions = EXCLUDED.completed_sessions,
			failed_sessions = EXCLUDED.failed_sessions,
			total_commands = EXCLUDED.total_commands,
			avg_commands_per_session = EXCLUDED.avg_commands_per_session,
			total_duration_seconds = EXCLUDED.total_duration_seconds,
			avg_session_duration = EXCLUDED.avg_session_duration,
			unique_targets = EXCLUDED.unique_targets,
			off_hours_sessions = EXCLUDED.off_hours_sessions,
			weekend_sessions = EXCLUDED.weekend_sessions,
			failed_command_rate = EXCLUDED.failed_command_rate,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query, tenantID, date)
	if err != nil {
		return fmt.Errorf("metrics_store.AggregateDailyMetrics: %w", err)
	}

	return nil
}

// PurgeOldMetrics removes metrics older than the retention period
func (s *MetricsStore) PurgeOldMetrics(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM session_analytics
		WHERE collected_at < NOW() - INTERVAL '1 day' * $1
	`

	result, err := s.db.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("metrics_store.PurgeOldMetrics: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	s.logger.Info().
		Int64("rows_deleted", rowsAffected).
		Int("retention_days", retentionDays).
		Msg("Old session metrics purged")

	return rowsAffected, nil
}

// UserMetricsSummary represents aggregated metrics for a user
type UserMetricsSummary struct {
	TotalSessions     int     `db:"total_sessions"`
	TotalCommands     int     `db:"total_commands"`
	AvgDuration       float64 `db:"avg_duration"`
	OffHoursCount     int     `db:"off_hours_count"`
	FailedSessions    int     `db:"failed_sessions"`
	UniqueTargets     int     `db:"unique_targets"`
}
