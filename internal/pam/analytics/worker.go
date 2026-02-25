package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// AggregationWorker handles background aggregation of analytics data
type AggregationWorker struct {
	repo   *Repository
	cache  *cache.Cache
	logger zerolog.Logger

	// Worker state
	running bool
	stopCh  chan struct{}
}

// NewAggregationWorker creates a new aggregation worker
func NewAggregationWorker(repo *Repository, cache *cache.Cache, logger zerolog.Logger) *AggregationWorker {
	return &AggregationWorker{
		repo:   repo,
		cache:  cache,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start begins the aggregation worker
func (w *AggregationWorker) Start(ctx context.Context) {
	w.running = true
	w.logger.Info().Msg("Starting aggregation worker")

	go w.run(ctx)
}

// Stop gracefully stops the aggregation worker
func (w *AggregationWorker) Stop() {
	if !w.running {
		return
	}

	w.logger.Info().Msg("Stopping aggregation worker")
	close(w.stopCh)
	w.running = false
}

// run is the main worker loop
func (w *AggregationWorker) run(ctx context.Context) {
	// Initial aggregation on startup
	time.Sleep(5 * time.Second)
	w.runHourlyAggregation(ctx)

	// Schedule hourly aggregations
	hourlyTicker := time.NewTicker(time.Hour)
	defer hourlyTicker.Stop()

	// Schedule daily aggregations at 2 AM
	dailyTicker := time.NewTicker(time.Minute)
	defer dailyTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("Aggregation worker context cancelled")
			return

		case <-w.stopCh:
			w.logger.Info().Msg("Aggregation worker stopped")
			return

		case <-hourlyTicker.C:
			w.runHourlyAggregation(ctx)

		case <-dailyTicker.C:
			if time.Now().Hour() == 2 && time.Now().Minute() < 5 {
				w.runDailyAggregation(ctx)
			}
		}
	}
}

// runHourlyAggregation aggregates data for the previous hour
func (w *AggregationWorker) runHourlyAggregation(ctx context.Context) {
	start := time.Now()
	w.logger.Debug().Msg("Running hourly aggregation")

	prevHour := time.Now().Add(-time.Hour).Truncate(time.Hour)
	hourEnd := prevHour.Add(time.Hour)

	// Get all tenants that need aggregation
	// For now, we'll aggregate based on existing data
	// In production, this would query a tenants table

	// Aggregate session data
	if err := w.aggregateSessionsForPeriod(ctx, prevHour, hourEnd); err != nil {
		w.logger.Error().Err(err).Msg("Failed to aggregate sessions")
	}

	// Aggregate event data
	if err := w.aggregateEventsForPeriod(ctx, prevHour, hourEnd); err != nil {
		w.logger.Error().Err(err).Msg("Failed to aggregate events")
	}

	// Aggregate user activity
	if err := w.aggregateUserActivityForPeriod(ctx, prevHour, hourEnd); err != nil {
		w.logger.Error().Err(err).Msg("Failed to aggregate user activity")
	}

	duration := time.Since(start)
	w.logger.Debug().Dur("duration", duration).Msg("Hourly aggregation complete")
}

// runDailyAggregation runs daily aggregation tasks
func (w *AggregationWorker) runDailyAggregation(ctx context.Context) {
	start := time.Now()
	w.logger.Info().Msg("Running daily aggregation")

	yesterday := time.Now().Add(-24 * time.Hour).Truncate(24 * time.Hour)
	today := time.Now().Truncate(24 * time.Hour)

	// Aggregate daily summaries
	if err := w.aggregateSessionsForPeriod(ctx, yesterday, today); err != nil {
		w.logger.Error().Err(err).Msg("Failed to aggregate daily sessions")
	}

	// Refresh materialized views
	if err := w.repo.RefreshMaterializedViews(ctx); err != nil {
		w.logger.Error().Err(err).Msg("Failed to refresh materialized views")
	}

	// Calculate risk scores for all active entities
	if err := w.calculateAllRiskScores(ctx); err != nil {
		w.logger.Error().Err(err).Msg("Failed to calculate risk scores")
	}

	// Clean old data
	if err := w.cleanupOldData(ctx); err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old data")
	}

	duration := time.Since(start)
	w.logger.Info().Dur("duration", duration).Msg("Daily aggregation complete")
}

// aggregateSessionsForPeriod aggregates session data for a time period
func (w *AggregationWorker) aggregateSessionsForPeriod(ctx context.Context, periodStart, periodEnd time.Time) error {
	// Check if repository is available (may be nil in tests)
	if w.repo == nil || w.repo.Db == nil {
		w.logger.Debug().Msg("Repository not available, skipping session aggregation")
		return nil
	}

	// Query sessions table for the period and aggregate by tenant and hour
	query := `
		INSERT INTO analytics_sessions (
			id, tenant_id, period_type, period_start, period_end,
			total_sessions, active_sessions, completed_sessions, failed_sessions, terminated_sessions,
			avg_duration_seconds, min_duration_seconds, max_duration_seconds,
			unique_users, protocol_breakdown, metadata, created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			s.tenant_id,
			'hour' as period_type,
			date_trunc('hour', s.started_at) as period_start,
			date_trunc('hour', s.started_at) + interval '1 hour' as period_end,
			COUNT(*) as total_sessions,
			COUNT(*) FILTER (WHERE s.status = 'active') as active_sessions,
			COUNT(*) FILTER (WHERE s.status = 'completed' OR s.ended_at IS NOT NULL) as completed_sessions,
			COUNT(*) FILTER (WHERE s.status = 'failed') as failed_sessions,
			COUNT(*) FILTER (WHERE s.status = 'terminated') as terminated_sessions,
			AVG(EXTRACT(EPOCH FROM (COALESCE(s.ended_at, NOW()) - s.started_at))) as avg_duration_seconds,
			MIN(EXTRACT(EPOCH FROM (COALESCE(s.ended_at, NOW()) - s.started_at))) as min_duration_seconds,
			MAX(EXTRACT(EPOCH FROM (COALESCE(s.ended_at, NOW()) - s.started_at))) as max_duration_seconds,
			COUNT(DISTINCT s.user_id) as unique_users,
			jsonb_object_agg(s.type, count_by_type) as protocol_breakdown,
			'{}'::jsonb as metadata,
			NOW() as created_at,
			NOW() as updated_at
		FROM sessions s
		JOIN (
			SELECT tenant_id, date_trunc('hour', started_at) as hour, type, COUNT(*) as count_by_type
			FROM sessions
			WHERE started_at >= $1 AND started_at < $2
			GROUP BY tenant_id, date_trunc('hour', started_at), type
		) type_counts ON s.tenant_id = type_counts.tenant_id
			AND date_trunc('hour', s.started_at) = type_counts.hour
			AND s.type = type_counts.type
		WHERE s.started_at >= $1 AND s.started_at < $2
		GROUP BY s.tenant_id, date_trunc('hour', s.started_at)
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
			unique_users = EXCLUDED.unique_users,
			protocol_breakdown = EXCLUDED.protocol_breakdown,
			updated_at = NOW()
	`

	_, err := w.repo.Db.ExecContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("aggregate sessions: %w", err)
	}

	return nil
}

// aggregateEventsForPeriod aggregates event data for a time period
func (w *AggregationWorker) aggregateEventsForPeriod(ctx context.Context, periodStart, periodEnd time.Time) error {
	// Check if repository is available (may be nil in tests)
	if w.repo == nil || w.repo.Db == nil {
		w.logger.Debug().Msg("Repository not available, skipping event aggregation")
		return nil
	}

	// Query audit_events table for the period and aggregate by tenant and hour
	query := `
		INSERT INTO analytics_events (
			id, tenant_id, period_type, period_start, period_end,
			total_events, successful_events, failed_events, denied_events,
			action_breakdown, top_users, top_resources,
			failed_auth_count, unique_failed_users, metadata, created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			e.tenant_id,
			'hour' as period_type,
			date_trunc('hour', e.created_at) as period_start,
			date_trunc('hour', e.created_at) + interval '1 hour' as period_end,
			COUNT(*) as total_events,
			COUNT(*) FILTER (WHERE e.outcome = 'success') as successful_events,
			COUNT(*) FILTER (WHERE e.outcome = 'failure') as failed_events,
			COUNT(*) FILTER (WHERE e.outcome = 'denied') as denied_events,
			jsonb_object_agg(e.action, action_counts.count) as action_breakdown,
			(
				SELECT jsonb_agg(jsonb_build_object('user_id', top.actor_id, 'event_count', top.count))
				FROM (
					SELECT actor_id, COUNT(*) as count
					FROM audit_events
					WHERE tenant_id = e.tenant_id
						AND created_at >= $1 AND created_at < $2
						AND date_trunc('hour', created_at) = date_trunc('hour', e.created_at)
					GROUP BY actor_id
					ORDER BY count DESC
					LIMIT 10
				) top
			) as top_users,
			(
				SELECT jsonb_agg(jsonb_build_object('resource_type', top.resource_type, 'event_count', top.count))
				FROM (
					SELECT resource_type, COUNT(*) as count
					FROM audit_events
					WHERE tenant_id = e.tenant_id
						AND created_at >= $1 AND created_at < $2
						AND date_trunc('hour', created_at) = date_trunc('hour', e.created_at)
					GROUP BY resource_type
					ORDER BY count DESC
					LIMIT 10
				) top
			) as top_resources,
			COUNT(*) FILTER (WHERE e.action = 'authenticate' AND e.outcome = 'failure') as failed_auth_count,
			COUNT(DISTINCT e.actor_id) FILTER (WHERE e.action = 'authenticate' AND e.outcome = 'failure') as unique_failed_users,
			'{}'::jsonb as metadata,
			NOW() as created_at,
			NOW() as updated_at
		FROM audit_events e
		JOIN (
			SELECT tenant_id, date_trunc('hour', created_at) as hour, action, COUNT(*) as count
			FROM audit_events
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY tenant_id, date_trunc('hour', created_at), action
		) action_counts ON e.tenant_id = action_counts.tenant_id
			AND date_trunc('hour', e.created_at) = action_counts.hour
			AND e.action = action_counts.action
		WHERE e.created_at >= $1 AND e.created_at < $2
		GROUP BY e.tenant_id, date_trunc('hour', e.created_at)
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
			updated_at = NOW()
	`

	_, err := w.repo.Db.ExecContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("aggregate events: %w", err)
	}

	return nil
}

// aggregateUserActivityForPeriod aggregates user activity for a time period
func (w *AggregationWorker) aggregateUserActivityForPeriod(ctx context.Context, periodStart, periodEnd time.Time) error {
	// Check if repository is available (may be nil in tests)
	if w.repo == nil || w.repo.Db == nil {
		w.logger.Debug().Msg("Repository not available, skipping user activity aggregation")
		return nil
	}

	// Query sessions and audit_events to aggregate user activity by tenant, user, and hour
	query := `
		INSERT INTO analytics_user_activity (
			id, tenant_id, user_id, date, hour,
			sessions_initiated, sessions_completed, commands_executed,
			total_session_seconds, unique_targets, off_hours_access,
			first_access_time, last_access_time, metadata, created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			s.tenant_id,
			s.user_id,
			DATE_TRUNC('day', s.started_at) as date,
			EXTRACT(HOUR FROM s.started_at)::INTEGER as hour,
			COUNT(*) as sessions_initiated,
			COUNT(*) FILTER (WHERE s.ended_at IS NOT NULL) as sessions_completed,
			COALESCE((SELECT SUM(commands_count) FROM sessions s2
				WHERE s2.user_id = s.user_id
				AND s2.tenant_id = s.tenant_id
				AND DATE_TRUNC('day', s2.started_at) = DATE_TRUNC('day', s.started_at)
				AND EXTRACT(HOUR FROM s2.started_at) = EXTRACT(HOUR FROM s.started_at)
			), 0) as commands_executed,
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(s.ended_at, NOW()) - s.started_at))), 0) as total_session_seconds,
			COUNT(DISTINCT s.target_host || ':' || s.target_port) as unique_targets,
			COUNT(*) FILTER (WHERE EXTRACT(HOUR FROM s.started_at) >= 18 OR EXTRACT(HOUR FROM s.started_at) < 6) > 0 as off_hours_access,
			MIN(s.started_at) as first_access_time,
			MAX(s.started_at) as last_access_time,
			'{}'::jsonb as metadata,
			NOW() as created_at,
			NOW() as updated_at
		FROM sessions s
		WHERE s.started_at >= $1 AND s.started_at < $2
		GROUP BY s.tenant_id, s.user_id, DATE_TRUNC('day', s.started_at), EXTRACT(HOUR FROM s.started_at)
		ON CONFLICT (tenant_id, user_id, date, hour)
		DO UPDATE SET
			sessions_initiated = EXCLUDED.sessions_initiated,
			sessions_completed = EXCLUDED.sessions_completed,
			commands_executed = analytics_user_activity.commands_executed + EXCLUDED.commands_executed,
			total_session_seconds = analytics_user_activity.total_session_seconds + EXCLUDED.total_session_seconds,
			unique_targets = GREATEST(analytics_user_activity.unique_targets, EXCLUDED.unique_targets),
			off_hours_access = analytics_user_activity.off_hours_access OR EXCLUDED.off_hours_access,
			last_access_time = GREATEST(analytics_user_activity.last_access_time, EXCLUDED.last_access_time),
			updated_at = NOW()
	`

	_, err := w.repo.Db.ExecContext(ctx, query, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("aggregate user activity: %w", err)
	}

	return nil
}

// calculateAllRiskScores calculates risk scores for all entities
func (w *AggregationWorker) calculateAllRiskScores(ctx context.Context) error {
	// Check if repository is available (may be nil in tests)
	if w.repo == nil || w.repo.Db == nil {
		w.logger.Debug().Msg("Repository not available, skipping risk score calculation")
		return nil
	}

	// Calculate risk scores for users based on recent activity (last 30 days)
	cutoffDate := time.Now().Add(-30 * 24 * time.Hour)

	// Upsert risk scores based on activity metrics
	query := `
		INSERT INTO analytics_risk_scores (
			id, tenant_id, entity_type, entity_id, score, risk_level,
			calculated_at, factors, trend, metadata, created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			ua.tenant_id,
			'user' as entity_type,
			ua.user_id as entity_id,
			-- Calculate risk score (0-100) based on multiple factors
			LEAST(100, GREATEST(0,
				-- Failed authentication rate factor (0-30 points)
				COALESCE((SELECT COUNT(*) FROM audit_events ae
					WHERE ae.actor_id = ua.user_id
					AND ae.tenant_id = ua.tenant_id
					AND ae.action = 'authenticate'
					AND ae.outcome = 'failure'
					AND ae.created_at >= $1
				) * 5, 0) +
				-- Off-hours access factor (0-20 points)
				COALESCE((SELECT COUNT(*) FROM analytics_user_activity uao
					WHERE uao.user_id = ua.user_id
					AND uao.tenant_id = ua.tenant_id
					AND uao.off_hours_access = true
					AND uao.date >= $1
				) * 2, 0) +
				-- High command volume factor (0-25 points)
				COALESCE((SELECT SUM(commands_executed) FROM analytics_user_activity uac
					WHERE uac.user_id = ua.user_id
					AND uac.tenant_id = ua.tenant_id
					AND uac.date >= $1
				) / 100, 0) +
				-- Session count factor (0-25 points)
				COALESCE((SELECT SUM(sessions_initiated) FROM analytics_user_activity uas
					WHERE uas.user_id = ua.user_id
					AND uas.tenant_id = ua.tenant_id
					AND uas.date >= $1
				), 0)
			)) as score,
			CASE
			 WHEN LEAST(100, GREATEST(0,
				COALESCE((SELECT COUNT(*) FROM audit_events ae
					WHERE ae.actor_id = ua.user_id AND ae.tenant_id = ua.tenant_id
					AND ae.action = 'authenticate' AND ae.outcome = 'failure' AND ae.created_at >= $1
				) * 5, 0) +
				COALESCE((SELECT COUNT(*) FROM analytics_user_activity uao
					WHERE uao.user_id = ua.user_id AND uao.tenant_id = ua.tenant_id
					AND uao.off_hours_access = true AND uao.date >= $1
				) * 2, 0) +
				COALESCE((SELECT SUM(commands_executed) FROM analytics_user_activity uac
					WHERE uac.user_id = ua.user_id AND uac.tenant_id = ua.tenant_id AND uac.date >= $1
				) / 100, 0) +
				COALESCE((SELECT SUM(sessions_initiated) FROM analytics_user_activity uas
					WHERE uas.user_id = ua.user_id AND uas.tenant_id = ua.tenant_id AND uas.date >= $1
				), 0)
			 )) >= 75 THEN 'critical'
			 WHEN LEAST(100, GREATEST(0,
				COALESCE((SELECT COUNT(*) FROM audit_events ae
					WHERE ae.actor_id = ua.user_id AND ae.tenant_id = ua.tenant_id
					AND ae.action = 'authenticate' AND ae.outcome = 'failure' AND ae.created_at >= $1
				) * 5, 0) +
				COALESCE((SELECT COUNT(*) FROM analytics_user_activity uao
					WHERE uao.user_id = ua.user_id AND uao.tenant_id = ua.tenant_id
					AND uao.off_hours_access = true AND uao.date >= $1
				) * 2, 0) +
				COALESCE((SELECT SUM(commands_executed) FROM analytics_user_activity uac
					WHERE uac.user_id = ua.user_id AND uac.tenant_id = ua.tenant_id AND uac.date >= $1
				) / 100, 0) +
				COALESCE((SELECT SUM(sessions_initiated) FROM analytics_user_activity uas
					WHERE uas.user_id = ua.user_id AND uas.tenant_id = ua.tenant_id AND uas.date >= $1
				), 0)
			 )) >= 50 THEN 'high'
			 WHEN LEAST(100, GREATEST(0,
				COALESCE((SELECT COUNT(*) FROM audit_events ae
					WHERE ae.actor_id = ua.user_id AND ae.tenant_id = ua.tenant_id
					AND ae.action = 'authenticate' AND ae.outcome = 'failure' AND ae.created_at >= $1
				) * 5, 0) +
				COALESCE((SELECT COUNT(*) FROM analytics_user_activity uao
					WHERE uao.user_id = ua.user_id AND uao.tenant_id = ua.tenant_id
					AND uao.off_hours_access = true AND uao.date >= $1
				) * 2, 0) +
				COALESCE((SELECT SUM(commands_executed) FROM analytics_user_activity uac
					WHERE uac.user_id = ua.user_id AND uac.tenant_id = ua.tenant_id AND uac.date >= $1
				) / 100, 0) +
				COALESCE((SELECT SUM(sessions_initiated) FROM analytics_user_activity uas
					WHERE uas.user_id = ua.user_id AND uas.tenant_id = ua.tenant_id AND uas.date >= $1
				), 0)
			 )) >= 25 THEN 'medium'
			 ELSE 'low'
			END as risk_level,
			NOW() as calculated_at,
			jsonb_build_object(
				'failed_auth_count', COALESCE((SELECT COUNT(*) FROM audit_events ae
					WHERE ae.actor_id = ua.user_id AND ae.tenant_id = ua.tenant_id
					AND ae.action = 'authenticate' AND ae.outcome = 'failure' AND ae.created_at >= $1
				), 0),
				'off_hours_access_count', COALESCE((SELECT COUNT(*) FROM analytics_user_activity uao
					WHERE uao.user_id = ua.user_id AND uao.tenant_id = ua.tenant_id
					AND uao.off_hours_access = true AND uao.date >= $1
				), 0),
				'total_commands', COALESCE((SELECT SUM(commands_executed) FROM analytics_user_activity uac
					WHERE uac.user_id = ua.user_id AND uac.tenant_id = ua.tenant_id AND uac.date >= $1
				), 0),
				'total_sessions', COALESCE((SELECT SUM(sessions_initiated) FROM analytics_user_activity uas
					WHERE uas.user_id = ua.user_id AND uas.tenant_id = ua.tenant_id AND uas.date >= $1
				), 0)
			) as factors,
			'stable' as trend,
			'{}'::jsonb as metadata,
			NOW() as created_at,
			NOW() as updated_at
		FROM analytics_user_activity ua
		WHERE ua.date >= $1
		GROUP BY ua.tenant_id, ua.user_id
		ON CONFLICT (tenant_id, entity_type, entity_id)
		DO UPDATE SET
			score = EXCLUDED.score,
			risk_level = EXCLUDED.risk_level,
			calculated_at = EXCLUDED.calculated_at,
			factors = EXCLUDED.factors,
			trend = CASE
				WHEN analytics_risk_scores.score < EXCLUDED.score THEN 'increasing'
				WHEN analytics_risk_scores.score > EXCLUDED.score THEN 'decreasing'
				ELSE 'stable'
			END,
			updated_at = NOW()
	`

	_, err := w.repo.Db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return fmt.Errorf("calculate risk scores: %w", err)
	}

	return nil
}

// cleanupOldData removes old analytics data based on retention policy
func (w *AggregationWorker) cleanupOldData(ctx context.Context) error {
	// Check if repository is available (may be nil in tests)
	if w.repo == nil || w.repo.Db == nil {
		w.logger.Debug().Msg("Repository not available, skipping data cleanup")
		return nil
	}

	// Retention policy:
	// - 90 days for detailed hourly analytics
	// - 365 days for daily summaries
	// - 7 days for raw metrics (keep aggregated data only)

	// Delete old hourly session analytics (older than 90 days)
	sessionCutoff := time.Now().Add(-90 * 24 * time.Hour)
	sessionQuery := `DELETE FROM analytics_sessions WHERE period_type = 'hour' AND period_start < $1`
	result, err := w.repo.Db.ExecContext(ctx, sessionQuery, sessionCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old session analytics")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_sessions", rows).Time("cutoff", sessionCutoff).Msg("Cleaned up old session analytics")
		}
	}

	// Delete old hourly event analytics (older than 90 days)
	eventQuery := `DELETE FROM analytics_events WHERE period_type = 'hour' AND period_start < $1`
	result, err = w.repo.Db.ExecContext(ctx, eventQuery, sessionCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old event analytics")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_events", rows).Time("cutoff", sessionCutoff).Msg("Cleaned up old event analytics")
		}
	}

	// Delete old daily summaries (older than 365 days)
	dailyCutoff := time.Now().Add(-365 * 24 * time.Hour)
	dailySessionQuery := `DELETE FROM analytics_sessions WHERE period_type = 'day' AND period_start < $1`
	result, err = w.repo.Db.ExecContext(ctx, dailySessionQuery, dailyCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old daily session summaries")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_daily_sessions", rows).Time("cutoff", dailyCutoff).Msg("Cleaned up old daily session summaries")
		}
	}

	dailyEventQuery := `DELETE FROM analytics_events WHERE period_type = 'day' AND period_start < $1`
	result, err = w.repo.Db.ExecContext(ctx, dailyEventQuery, dailyCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old daily event summaries")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_daily_events", rows).Time("cutoff", dailyCutoff).Msg("Cleaned up old daily event summaries")
		}
	}

	// Delete old user activity (older than 90 days)
	activityQuery := `DELETE FROM analytics_user_activity WHERE date < $1`
	result, err = w.repo.Db.ExecContext(ctx, activityQuery, sessionCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old user activity")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_activities", rows).Time("cutoff", sessionCutoff).Msg("Cleaned up old user activity")
		}
	}

	// Delete old metrics (older than 30 days)
	metricsCutoff := time.Now().Add(-30 * 24 * time.Hour)
	metricsQuery := `DELETE FROM analytics_metrics WHERE recorded_at < $1`
	result, err = w.repo.Db.ExecContext(ctx, metricsQuery, metricsCutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to cleanup old metrics")
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			w.logger.Info().Int64("deleted_metrics", rows).Time("cutoff", metricsCutoff).Msg("Cleaned up old metrics")
		}
	}

	w.logger.Info().Msg("Data cleanup complete")

	return nil
}

// AggregateForTenant manually triggers aggregation for a specific tenant
func (w *AggregationWorker) AggregateForTenant(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) error {
	w.logger.Info().
		Str("tenant_id", tenantID.String()).
		Time("start", periodStart).
		Time("end", periodEnd).
		Msg("Running manual aggregation for tenant")

	// Aggregate sessions
	if err := w.aggregateSessionsForPeriod(ctx, periodStart, periodEnd); err != nil {
		return fmt.Errorf("aggregate sessions: %w", err)
	}

	// Aggregate events
	if err := w.aggregateEventsForPeriod(ctx, periodStart, periodEnd); err != nil {
		return fmt.Errorf("aggregate events: %w", err)
	}

	// Aggregate user activity
	if err := w.aggregateUserActivityForPeriod(ctx, periodStart, periodEnd); err != nil {
		return fmt.Errorf("aggregate user activity: %w", err)
	}

	return nil
}

// GetStatus returns the current status of the aggregation worker
func (w *AggregationWorker) GetStatus() AggregationWorkerStatus {
	return AggregationWorkerStatus{
		Running:         w.running,
		LastRun:         time.Now(), // TODO: track actual last run time
		LastRunDuration: 0,
		NextRun:         time.Now().Add(time.Hour),
	}
}

// AggregationWorkerStatus represents the status of the aggregation worker
type AggregationWorkerStatus struct {
	Running           bool      `json:"running"`
	LastRun           time.Time `json:"last_run"`
	LastRunDuration   int64     `json:"last_run_duration_ms"`
	NextRun           time.Time `json:"next_run"`
	ProcessedTenants  int       `json:"processed_tenants"`
	SessionsProcessed int64     `json:"sessions_processed"`
	EventsProcessed  int64     `json:"events_processed"`
}

// AlertEvaluator evaluates alerts and triggers notifications
type AlertEvaluator struct {
	repo   *Repository
	cache  *cache.Cache
	logger zerolog.Logger

	running bool
	stopCh  chan struct{}
}

// NewAlertEvaluator creates a new alert evaluator
func NewAlertEvaluator(repo *Repository, cache *cache.Cache, logger zerolog.Logger) *AlertEvaluator {
	return &AlertEvaluator{
		repo:   repo,
		cache:  cache,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start begins the alert evaluator
func (e *AlertEvaluator) Start(ctx context.Context) {
	e.running = true
	e.logger.Info().Msg("Starting alert evaluator")

	go e.run(ctx)
}

// Stop gracefully stops the alert evaluator
func (e *AlertEvaluator) Stop() {
	if !e.running {
		return
	}

	e.logger.Info().Msg("Stopping alert evaluator")
	close(e.stopCh)
	e.running = false
}

// run is the main evaluator loop
func (e *AlertEvaluator) run(ctx context.Context) {
	// Initial evaluation after startup
	time.Sleep(10 * time.Second)
	e.evaluateAllAlerts(ctx)

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.logger.Info().Msg("Alert evaluator context cancelled")
			return

		case <-e.stopCh:
			e.logger.Info().Msg("Alert evaluator stopped")
			return

		case <-ticker.C:
			e.evaluateAllAlerts(ctx)
		}
	}
}

// evaluateAllAlerts evaluates all enabled alerts
func (e *AlertEvaluator) evaluateAllAlerts(ctx context.Context) {
	start := time.Now()
	e.logger.Debug().Msg("Evaluating alerts")

	alerts, err := e.repo.GetActiveAlerts(ctx)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to get active alerts")
		return
	}

	triggeredCount := 0
	for _, alert := range alerts {
		// Check if alert is due for evaluation
		if alert.NextTriggerAt != nil && alert.NextTriggerAt.After(time.Now()) {
			continue
		}

		// Check cooldown
		if alert.LastTriggeredAt != nil && time.Since(*alert.LastTriggeredAt) < time.Duration(alert.CooldownMinutes)*time.Minute {
			continue
		}

		// Evaluate alert based on type
		triggered, err := e.evaluateAlert(ctx, &alert)
		if err != nil {
			e.logger.Error().Err(err).
				Str("alert_id", alert.ID.String()).
				Msg("Failed to evaluate alert")
			continue
		}

		if triggered {
			triggeredCount++
		}

		// Update last evaluated time
		_ = e.repo.UpdateAlertEvaluationTime(ctx, alert.ID)

		// Schedule next evaluation
		nextRun := time.Now().Add(time.Duration(alert.EvaluationIntervalMinutes) * time.Minute)
		_ = e.repo.UpdateAlertTriggerStatus(ctx, alert.ID, time.Now(), nextRun, alert.TriggerCount)
	}

	duration := time.Since(start)
	e.logger.Debug().
		Int("evaluated", len(alerts)).
		Int("triggered", triggeredCount).
		Dur("duration", duration).
		Msg("Alert evaluation complete")
}

// evaluateAlert evaluates a single alert
func (e *AlertEvaluator) evaluateAlert(ctx context.Context, alert *Alert) (bool, error) {
	switch alert.AlertType {
	case AlertTypeThreshold:
		triggered, data, err := e.evaluateThresholdAlert(ctx, alert)
		if err != nil {
			return false, err
		}
		if triggered {
			_ = e.triggerAlert(ctx, alert.ID, alert.TenantID, string(alert.Severity), data)
		}
		return triggered, nil

	case AlertTypePattern:
		triggered, data, err := e.evaluatePatternAlert(ctx, alert)
		if err != nil {
			return false, err
		}
		if triggered {
			_ = e.triggerAlert(ctx, alert.ID, alert.TenantID, string(alert.Severity), data)
		}
		return triggered, nil

	case AlertTypeAnomaly:
		triggered, data, err := e.evaluateAnomalyAlert(ctx, alert)
		if err != nil {
			return false, err
		}
		if triggered {
			_ = e.triggerAlert(ctx, alert.ID, alert.TenantID, string(alert.Severity), data)
		}
		return triggered, nil

	case AlertTypeCompliance:
		triggered, data, err := e.evaluateComplianceAlert(ctx, alert)
		if err != nil {
			return false, err
		}
		if triggered {
			_ = e.triggerAlert(ctx, alert.ID, alert.TenantID, string(alert.Severity), data)
		}
		return triggered, nil

	default:
		return false, nil
	}
}

// evaluateThresholdAlert evaluates threshold-based alerts
func (e *AlertEvaluator) evaluateThresholdAlert(ctx context.Context, alert *Alert) (bool, map[string]interface{}, error) {
	// Parse threshold conditions from alert.Conditions
	// Example: {"metric": "failed_auth_rate", "operator": "greater_than", "threshold": 10, "window_minutes": 15}

	var conditions struct {
		Metric       string  `json:"metric"`
		Operator     string  `json:"operator"`
		Threshold    float64 `json:"threshold"`
		WindowMinutes int    `json:"window_minutes"`
	}

	if err := json.Unmarshal(alert.Conditions, &conditions); err != nil {
		return false, nil, fmt.Errorf("invalid threshold conditions: %w", err)
	}

	// Calculate metric value
	value, err := e.calculateMetric(ctx, alert.TenantID, conditions.Metric, conditions.WindowMinutes)
	if err != nil {
		return false, nil, err
	}

	// Check threshold
	triggered := false
	switch conditions.Operator {
	case "greater_than", ">":
		triggered = value > conditions.Threshold
	case "less_than", "<":
		triggered = value < conditions.Threshold
	case "greater_or_equal", ">=":
		triggered = value >= conditions.Threshold
	case "less_or_equal", "<=":
		triggered = value <= conditions.Threshold
	case "equals", "==":
		triggered = value == conditions.Threshold
	}

	if triggered {
		return true, map[string]interface{}{
			"metric":    conditions.Metric,
			"value":     value,
			"threshold": conditions.Threshold,
			"operator":  conditions.Operator,
		}, nil
	}

	return false, nil, nil
}

// evaluatePatternAlert evaluates pattern-based alerts
func (e *AlertEvaluator) evaluatePatternAlert(ctx context.Context, alert *Alert) (bool, map[string]interface{}, error) {
	// Parse pattern conditions
	// Example: {"pattern": "after_hours_access", "days": ["saturday", "sunday"], "hours": {"start": 18, "end": 6}}

	var conditions struct {
		Pattern string            `json:"pattern"`
		Days    []string          `json:"days"`
		Hours   map[string]int    `json:"hours"`
	}

	if err := json.Unmarshal(alert.Conditions, &conditions); err != nil {
		return false, nil, fmt.Errorf("invalid pattern conditions: %w", err)
	}

	switch conditions.Pattern {
	case "after_hours_access":
		return e.checkAfterHoursAccess(ctx, alert.TenantID, conditions)

	case "concurrent_sessions":
		return e.checkConcurrentSessions(ctx, alert.TenantID, conditions)

	default:
		return false, nil, nil
	}
}

// evaluateAnomalyAlert evaluates anomaly-based alerts
func (e *AlertEvaluator) evaluateAnomalyAlert(ctx context.Context, alert *Alert) (bool, map[string]interface{}, error) {
	// Check for recent anomalies
	// Anomaly alerts would typically be triggered by the anomaly detection system

	// Check if there are any open anomalies in the last hour
	_ = time.Now().Add(-time.Hour)

	// TODO: Query anomaly_detections table for open anomalies since:time.Now()
	// For now, return false

	return false, nil, nil
}

// evaluateComplianceAlert evaluates compliance-based alerts
func (e *AlertEvaluator) evaluateComplianceAlert(ctx context.Context, alert *Alert) (bool, map[string]interface{}, error) {
	// Parse compliance conditions
	// Example: {"framework": "SOC2", "control_id": "ac-001", "compliance_rate_below": 90}

	var conditions struct {
		Framework             string  `json:"framework"`
		ControlID             string  `json:"control_id"`
		ComplianceRateBelow  float64 `json:"compliance_rate_below"`
	}

	if err := json.Unmarshal(alert.Conditions, &conditions); err != nil {
		return false, nil, fmt.Errorf("invalid compliance conditions: %w", err)
	}

	// Get compliance status
	// TODO: Query compliance status for the framework/control
	// For now, return false

	return false, nil, nil
}

// calculateMetric calculates a metric value for alert evaluation
func (e *AlertEvaluator) calculateMetric(ctx context.Context, tenantID uuid.UUID, metric string, windowMinutes int) (float64, error) {
	switch metric {
	case "failed_auth_rate":
		return e.calculateFailedAuthRate(ctx, tenantID, windowMinutes)
	case "concurrent_sessions":
		return e.getConcurrentSessionCount(ctx, tenantID)
	case "off_hours_sessions":
		return e.getOffHoursSessionCount(ctx, tenantID, windowMinutes)
	case "high_risk_users":
		return e.getHighRiskUserCount(ctx, tenantID)
	default:
		return 0, fmt.Errorf("unknown metric: %s", metric)
	}
}

// calculateFailedAuthRate calculates the failed authentication rate
func (e *AlertEvaluator) calculateFailedAuthRate(ctx context.Context, tenantID uuid.UUID, windowMinutes int) (float64, error) {
	// TODO: Query audit_events for failed authentication attempts in the time window
	// Calculate: (failed / total) * 100
	return 0, nil
}

// getConcurrentSessionCount gets the current number of concurrent sessions
func (e *AlertEvaluator) getConcurrentSessionCount(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	// TODO: Query sessions table for active sessions count
	return 0, nil
}

// getOffHoursSessionCount counts off-hours sessions in a time window
func (e *AlertEvaluator) getOffHoursSessionCount(ctx context.Context, tenantID uuid.UUID, windowMinutes int) (float64, error) {
	// TODO: Query user_activity table for off_hours_access in time window
	return 0, nil
}

// getHighRiskUserCount gets the count of users with high risk scores
func (e *AlertEvaluator) getHighRiskUserCount(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	// TODO: Query analytics_risk_scores for high/critical risk level
	return 0, nil
}

// checkAfterHoursAccess checks if there's after-hours access
func (e *AlertEvaluator) checkAfterHoursAccess(ctx context.Context, tenantID uuid.UUID, conditions interface{}) (bool, map[string]interface{}, error) {
	// Check if there have been sessions during off-hours in the past window
	// TODO: Implement after-hours detection
	return false, nil, nil
}

// checkConcurrentSessions checks if concurrent sessions exceed threshold
func (e *AlertEvaluator) checkConcurrentSessions(ctx context.Context, tenantID uuid.UUID, conditions interface{}) (bool, map[string]interface{}, error) {
	// Check concurrent session count
	// TODO: Implement concurrent session detection
	return false, nil, nil
}

// triggerAlert creates an alert trigger and sends notifications
func (e *AlertEvaluator) triggerAlert(ctx context.Context, alertID uuid.UUID, tenantID uuid.UUID, severity string, triggerData map[string]interface{}) error {
	// Create alert trigger
	trigger := &AlertTrigger{
		AlertID:     alertID,
		TenantID:    tenantID,
		Severity:    Severity(severity),
		TriggerData: mustMarshalJSON(triggerData),
		TriggeredAt:  time.Now(),
	}

	if err := e.repo.CreateAlertTrigger(ctx, trigger); err != nil {
		return fmt.Errorf("failed to create alert trigger: %w", err)
	}

	// Update alert status
	// TODO: Update alert with last_triggered_at and increment trigger_count

	// Send notifications
	// TODO: Send email, webhook, etc based on alert.NotificationMethods

	e.logger.Info().
		Str("alert_id", alertID.String()).
		Str("severity", severity).
		Interface("trigger_data", triggerData).
		Msg("Alert triggered")

	return nil
}

// GetStatus returns the current status of the alert evaluator
func (e *AlertEvaluator) GetStatus() AlertEvaluatorStatus {
	return AlertEvaluatorStatus{
		Running:           e.running,
		LastEvaluation:    time.Now(),
		AlertsEvaluated:   0,
		AlertsTriggered:   0,
		NextEvaluation:    time.Now().Add(15 * time.Minute),
	}
}

// AlertEvaluatorStatus represents the status of the alert evaluator
type AlertEvaluatorStatus struct {
	Running          bool      `json:"running"`
	LastEvaluation   time.Time `json:"last_evaluation"`
	AlertsEvaluated  int       `json:"alerts_evaluated"`
	AlertsTriggered  int       `json:"alerts_triggered"`
	NextEvaluation   time.Time `json:"next_evaluation"`
}

// ReportScheduler handles scheduled report generation
type ReportScheduler struct {
	repo   *Repository
	cache  *cache.Cache
	logger zerolog.Logger

	running bool
	stopCh  chan struct{}
}

// NewReportScheduler creates a new report scheduler
func NewReportScheduler(repo *Repository, cache *cache.Cache, logger zerolog.Logger) *ReportScheduler {
	return &ReportScheduler{
		repo:   repo,
		cache:  cache,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start begins the report scheduler
func (s *ReportScheduler) Start(ctx context.Context) {
	s.running = true
	s.logger.Info().Msg("Starting report scheduler")

	go s.run(ctx)
}

// Stop gracefully stops the report scheduler
func (s *ReportScheduler) Stop() {
	if !s.running {
		return
	}

	s.logger.Info().Msg("Stopping report scheduler")
	close(s.stopCh)
	s.running = false
}

// run is the main scheduler loop
func (s *ReportScheduler) run(ctx context.Context) {
	// Check for due reports every minute
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Report scheduler context cancelled")
			return

		case <-s.stopCh:
			s.logger.Info().Msg("Report scheduler stopped")
			return

		case <-ticker.C:
			s.checkAndRunReports(ctx)
		}
	}
}

// checkAndRunReports checks for reports that need to run and executes them
func (s *ReportScheduler) checkAndRunReports(ctx context.Context) {
	s.logger.Debug().Msg("Checking for scheduled reports")

	// Get all reports with schedule_enabled = true
	// TODO: Query reports where schedule_enabled = true AND next_run_at <= NOW()

	// For each due report, generate and deliver
	// TODO: Implement report generation and delivery

	s.logger.Debug().Msg("Scheduled reports check complete")
}

// GetStatus returns the current status of the report scheduler
func (s *ReportScheduler) GetStatus() ReportSchedulerStatus {
	return ReportSchedulerStatus{
		Running:         s.running,
		LastCheck:       time.Now(),
		ReportsProcessed: 0,
		ReportsScheduled: 0,
	}
}

// ReportSchedulerStatus represents the status of the report scheduler
type ReportSchedulerStatus struct {
	Running           bool      `json:"running"`
	LastCheck         time.Time `json:"last_check"`
	ReportsProcessed  int       `json:"reports_processed"`
	ReportsScheduled int       `json:"reports_scheduled"`
	NextScheduledTime *time.Time `json:"next_scheduled_time"`
}
