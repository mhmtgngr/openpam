// Package analytics provides metrics collection for security monitoring
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/analytics"
	"github.com/rs/zerolog"
)

// MetricsStore handles collection and storage of session metrics
type MetricsStore struct {
	db        *sqlx.DB
	redis     *analytics.RedisMetricsCache
	logger    zerolog.Logger
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore(db *sqlx.DB, redis *analytics.RedisMetricsCache, logger zerolog.Logger) *MetricsStore {
	return &MetricsStore{
		db:     db,
		redis:  redis,
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
	UniqueTargets    int       `json:"unique_targets" db:"unique_targets"`
	IPAddress        string    `json:"ip_address" db:"ip_address"`
	UserAgent        string    `json:"user_agent" db:"user_agent"`
	OffHoursAccess   bool      `json:"off_hours_access" db:"off_hours_access"`
	WeekendAccess    bool      `json:"weekend_access" db:"weekend_access"`
	FailureRate      float64   `json:"failure_rate" db:"failure_rate"`
	Commands         []string  `json:"commands" db:"commands"`
	Metadata         json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CollectedAt      time.Time `json:"collected_at" db:"collected_at"`
}

// RecordSessionMetrics records metrics for a completed session
func (s *MetricsStore) RecordSessionMetrics(ctx context.Context, metrics *SessionMetrics) error {
	metrics.CollectedAt = time.Now()

	// Store in Redis for fast access (with TTL)
	if err := s.redis.StoreSessionMetrics(ctx, metrics); err != nil {
		s.logger.Warn().Err(err).
			Str("session_id", metrics.SessionID.String()).
			Msg("Failed to cache metrics in Redis")
	}

	// Store in database for persistence
	query := `
		INSERT INTO session_analytics (
			session_id, tenant_id, user_id, target_host,
			start_time, end_time, duration_seconds,
			command_count, failed_commands, unique_targets,
			ip_address, user_agent, off_hours_access, weekend_access,
			failure_rate, commands, metadata, collected_at
		) VALUES (
			:session_id, :tenant_id, :user_id, :target_host,
			:start_time, :end_time, :duration_seconds,
			:command_count, :failed_commands, :unique_targets,
			:ip_address, :user_agent, :off_hours_access, :weekend_access,
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
	// Try Redis cache first
	cached, err := s.redis.GetSessionMetrics(ctx, sessionID)
	if err == nil && cached != nil {
		return cached, nil
	}

	var metrics SessionMetrics
	query := `
		SELECT * FROM session_analytics
		WHERE session_id = $1
	`

	err = s.db.GetContext(ctx, &metrics, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetSessionMetrics: %w", err)
	}

	return &metrics, nil
}

// GetUserBehaviorMetrics retrieves behavior metrics for analysis
func (s *MetricsStore) GetUserBehaviorMetrics(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]SessionMetric, error) {
	query := `
		SELECT
			session_id, user_id, start_time as timestamp, target_host,
			duration_seconds, ip_address,
			off_hours_access, commands,
			CASE WHEN failure_rate > 0.5 THEN true ELSE false END as failed
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= $2
			AND start_time <= $3
		ORDER BY start_time DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetUserBehaviorMetrics: %w", err)
	}
	defer rows.Close()

	var metrics []SessionMetric
	for rows.Next() {
		var m SessionMetric
		if err := rows.Scan(
			&m.SessionID, &m.UserID, &m.Timestamp, &m.TargetHost,
			&m.DurationSeconds, &m.IPAddress,
			&m.OffHoursAccess, &m.Commands, &m.Failed,
		); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetTemporalMetrics retrieves metrics for temporal analysis
func (s *MetricsStore) GetTemporalMetrics(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]SessionMetric, error) {
	return s.GetUserBehaviorMetrics(ctx, tenantID, from, to)
}

// GetSpatialMetrics retrieves metrics for spatial analysis
func (s *MetricsStore) GetSpatialMetrics(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]SessionMetric, error) {
	query := `
		SELECT
			session_id, user_id, start_time as timestamp, target_host,
			duration_seconds, ip_address,
			off_hours_access, commands,
			CASE WHEN failure_rate > 0.5 THEN true ELSE false END as failed
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= $2
			AND start_time <= $3
		ORDER BY target_host, start_time DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetSpatialMetrics: %w", err)
	}
	defer rows.Close()

	var metrics []SessionMetric
	for rows.Next() {
		var m SessionMetric
		if err := rows.Scan(
			&m.SessionID, &m.UserID, &m.Timestamp, &m.TargetHost,
			&m.DurationSeconds, &m.IPAddress,
			&m.OffHoursAccess, &m.Commands, &m.Failed,
		); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetVolumetricMetrics retrieves aggregated volumetric metrics
func (s *MetricsStore) GetVolumetricMetrics(ctx context.Context, tenantID uuid.UUID, from, to time.Time) ([]VolumetricMetric, error) {
	query := `
		SELECT
			user_id,
			COUNT(*) as total_sessions,
			COALESCE(SUM(command_count), 0) as total_commands,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_sessions,
			COALESCE(SUM(duration_seconds), 0) as total_duration
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= $2
			AND start_time <= $3
		GROUP BY user_id
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetVolumetricMetrics: %w", err)
	}
	defer rows.Close()

	var metrics []VolumetricMetric
	for rows.Next() {
		var m VolumetricMetric
		if err := rows.Scan(
			&m.UserID, &m.TotalSessions, &m.TotalCommands,
			&m.FailedSessions, &m.TotalDuration,
		); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// AggregateMetricsByTimeBucket aggregates metrics into time buckets
func (s *MetricsStore) AggregateMetricsByTimeBucket(ctx context.Context, tenantID uuid.UUID, from, to time.Time, bucket time.Duration) ([]TimeBucketMetrics, error) {
	// Determine bucket size in seconds
	bucketSeconds := int(bucket.Seconds())

	query := `
		SELECT
			date_trunc('bucket', start_time) as bucket_start,
			COUNT(*) as session_count,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT target_host) as unique_targets,
			COALESCE(SUM(command_count), 0) as total_commands,
			COALESCE(AVG(duration_seconds), 0) as avg_duration,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_session_count
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= $2
			AND start_time <= $3
		GROUP BY date_trunc('bucket', start_time)
		ORDER BY bucket_start
	`

	// Use time_bucket with PostgreSQL's date_trunc
	bucketSize := ""
	switch bucketSeconds {
	case 60:
		bucketSize = "minute"
	case 3600:
		bucketSize = "hour"
	case 86400:
		bucketSize = "day"
	default:
		bucketSize = "hour"
	}

	query = fmt.Sprintf(query, bucketSize)

	rows, err := s.db.QueryContext(ctx, query, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.AggregateMetricsByTimeBucket: %w", err)
	}
	defer rows.Close()

	var metrics []TimeBucketMetrics
	for rows.Next() {
		var m TimeBucketMetrics
		if err := rows.Scan(
			&m.BucketStart, &m.SessionCount, &m.UniqueUsers,
			&m.UniqueTargets, &m.TotalCommands, &m.AvgDuration,
			&m.OffHoursCount, &m.FailedSessionCount,
		); err != nil {
			continue
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// GetCommandFrequency returns command execution frequency for a user
func (s *MetricsStore) GetCommandFrequency(ctx context.Context, tenantID, userID uuid.UUID, from, to time.Time, limit int) (map[string]int, error) {
	query := `
		SELECT jsonb_object_keys(commands) as command, COUNT(*) as frequency
		FROM session_analytics
		WHERE tenant_id = $1
			AND user_id = $2
			AND start_time >= $3
			AND start_time <= $4
			AND commands IS NOT NULL
		GROUP BY command
		ORDER BY frequency DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, tenantID, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetCommandFrequency: %w", err)
	}
	defer rows.Close()

	frequency := make(map[string]int)
	for rows.Next() {
		var cmd string
		var count int
		if err := rows.Scan(&cmd, &count); err != nil {
			continue
		}
		frequency[cmd] = count
	}

	return frequency, nil
}

// GetTargetAccessFrequency returns target access frequency for a user
func (s *MetricsStore) GetTargetAccessFrequency(ctx context.Context, tenantID, userID uuid.UUID, from, to time.Time, limit int) (map[string]int, error) {
	query := `
		SELECT target_host, COUNT(*) as access_count
		FROM session_analytics
		WHERE tenant_id = $1
			AND user_id = $2
			AND start_time >= $3
			AND start_time <= $3
		GROUP BY target_host
		ORDER BY access_count DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, tenantID, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetTargetAccessFrequency: %w", err)
	}
	defer rows.Close()

	frequency := make(map[string]int)
	for rows.Next() {
		var host string
		var count int
		if err := rows.Scan(&host, &count); err != nil {
			continue
		}
		frequency[host] = count
	}

	return frequency, nil
}

// GetRecentActivityForUser gets recent session activity for a specific user
func (s *MetricsStore) GetRecentActivityForUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]SessionMetrics, error) {
	query := `
		SELECT * FROM session_analytics
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY start_time DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var metrics []SessionMetrics
	err := s.db.SelectContext(ctx, &metrics, query, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetRecentActivityForUser: %w", err)
	}

	return metrics, nil
}

// GetMetricsSummary returns a summary of metrics for a tenant
func (s *MetricsStore) GetMetricsSummary(ctx context.Context, tenantID uuid.UUID, from, to time.Time) (*MetricsSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_sessions,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT target_host) as unique_targets,
			COALESCE(SUM(command_count), 0) as total_commands,
			COALESCE(AVG(duration_seconds), 0) as avg_session_duration,
			COUNT(*) FILTER (WHERE off_hours_access) as off_hours_sessions,
			COUNT(*) FILTER (WHERE weekend_access) as weekend_sessions,
			COUNT(*) FILTER (WHERE failure_rate > 0.5) as failed_sessions,
			COUNT(*) FILTER (WHERE failure_rate > 0.5)::float / COUNT(*) as overall_failure_rate
		FROM session_analytics
		WHERE tenant_id = $1
			AND start_time >= $2
			AND start_time <= $3
	`

	var summary MetricsSummary
	err := s.db.GetContext(ctx, &summary, query, tenantID, from, to)
	if err != nil {
		return nil, fmt.Errorf("metrics_store.GetMetricsSummary: %w", err)
	}

	return &summary, nil
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

// RebuildUserMetrics triggers a rebuild of user metrics cache
func (s *MetricsStore) RebuildUserMetrics(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) error {
	// This would trigger a recalculation of baselines
	// For now, just log the request
	s.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("user_id", userID.String()).
		Msg("User metrics rebuild requested")

	return nil
}

// Data structures

// TimeBucketMetrics represents aggregated metrics for a time bucket
type TimeBucketMetrics struct {
	BucketStart       time.Time `json:"bucket_start" db:"bucket_start"`
	SessionCount      int       `json:"session_count" db:"session_count"`
	UniqueUsers       int       `json:"unique_users" db:"unique_users"`
	UniqueTargets     int       `json:"unique_targets" db:"unique_targets"`
	TotalCommands     int       `json:"total_commands" db:"total_commands"`
	AvgDuration       float64   `json:"avg_duration" db:"avg_duration"`
	OffHoursCount     int       `json:"off_hours_count" db:"off_hours_count"`
	FailedSessionCount int      `json:"failed_session_count" db:"failed_session_count"`
}

// MetricsSummary represents a summary of metrics
type MetricsSummary struct {
	TotalSessions        int     `json:"total_sessions" db:"total_sessions"`
	UniqueUsers          int     `json:"unique_users" db:"unique_users"`
	UniqueTargets        int     `json:"unique_targets" db:"unique_targets"`
	TotalCommands        int     `json:"total_commands" db:"total_commands"`
	AvgSessionDuration   float64 `json:"avg_session_duration" db:"avg_session_duration"`
	OffHoursSessions     int     `json:"off_hours_sessions" db:"off_hours_sessions"`
	WeekendSessions      int     `json:"weekend_sessions" db:"weekend_sessions"`
	FailedSessions       int     `json:"failed_sessions" db:"failed_sessions"`
	OverallFailureRate   float64 `json:"overall_failure_rate" db:"overall_failure_rate"`
}
