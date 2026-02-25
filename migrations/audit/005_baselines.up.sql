-- Baselines Migration for Audit Analytics
-- Creates tables for user behavioral baselines and session metrics collection

-- ============================================================================
-- User Baselines Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,

    -- Baseline metadata
    baseline_type VARCHAR(20) NOT NULL DEFAULT 'daily', -- daily, weekly, monthly
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    sample_size INTEGER NOT NULL DEFAULT 0,

    -- Session metrics baseline
    mean_daily_sessions NUMERIC(10,2) NOT NULL DEFAULT 0,
    std_dev_daily_sessions NUMERIC(10,2) NOT NULL DEFAULT 0,
    mean_session_duration NUMERIC(12,2) NOT NULL DEFAULT 0, -- in seconds
    std_dev_session_duration NUMERIC(12,2) NOT NULL DEFAULT 0,

    -- Command metrics baseline
    mean_daily_commands NUMERIC(10,2) NOT NULL DEFAULT 0,
    std_dev_daily_commands NUMERIC(10,2) NOT NULL DEFAULT 0,

    -- Failure metrics baseline
    mean_failure_rate NUMERIC(5,4) NOT NULL DEFAULT 0,
    std_dev_failure_rate NUMERIC(5,4) NOT NULL DEFAULT 0,

    -- Temporal metrics baseline
    off_hours_access_mean NUMERIC(5,4) NOT NULL DEFAULT 0,
    off_hours_access_std_dev NUMERIC(5,4) NOT NULL DEFAULT 0,
    hourly_access_pattern JSONB, -- hour (0-23) -> mean access count
    weekday_access_pattern JSONB, -- weekday (0-6) -> mean access count

    -- Spatial metrics baseline
    target_access_stats JSONB, -- target -> {mean_access, std_dev, first_seen, last_seen}
    target_access_std_dev NUMERIC(10,2) NOT NULL DEFAULT 0,

    -- Command patterns (top commands)
    top_commands JSONB, -- command -> count

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_calculated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    confidence NUMERIC(5,2) NOT NULL DEFAULT 0, -- 0-100

    CONSTRAINT user_baselines_type_check CHECK (baseline_type IN ('daily', 'weekly', 'monthly', 'custom')),
    CONSTRAINT user_baselines_confidence_check CHECK (confidence >= 0 AND confidence <= 100)
);

-- Indexes for user_baselines
CREATE INDEX idx_user_baselines_tenant ON user_baselines(tenant_id);
CREATE INDEX idx_user_baselines_user ON user_baselines(user_id);
CREATE INDEX idx_user_baselines_active ON user_baselines(tenant_id, user_id) WHERE is_active = true;
CREATE INDEX idx_user_baselines_calculated ON user_baselines(last_calculated DESC);
CREATE INDEX idx_user_baselines_period ON user_baselines(period_start, period_end);
CREATE INDEX idx_user_baselines_metadata ON user_baselines USING GIN (target_access_stats);

-- Unique constraint to prevent overlapping active baselines
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_baselines_unique_active
    ON user_baselines(tenant_id, user_id, baseline_type)
    WHERE is_active = true;

-- ============================================================================
-- Session Analytics Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS session_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,

    -- Session identification
    session_id UUID NOT NULL,
    target_host VARCHAR(255) NOT NULL,

    -- Timing
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    duration_seconds NUMERIC(12,2) NOT NULL DEFAULT 0,

    -- Command metrics
    command_count INTEGER NOT NULL DEFAULT 0,
    failed_commands INTEGER NOT NULL DEFAULT 0,
    unique_targets INTEGER NOT NULL DEFAULT 0,

    -- Network/connection info
    ip_address INET,
    user_agent TEXT,

    -- Temporal flags
    off_hours_access BOOLEAN NOT NULL DEFAULT false,
    weekend_access BOOLEAN NOT NULL DEFAULT false,

    -- Failure metrics
    failure_rate NUMERIC(5,4) NOT NULL DEFAULT 0,

    -- Commands executed (as JSON array or JSONB object with counts)
    commands JSONB,

    -- Additional metadata
    metadata JSONB DEFAULT '{}',
    collected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT session_analytics_duration_check CHECK (duration_seconds >= 0),
    CONSTRAINT session_analytics_failure_check CHECK (failure_rate >= 0 AND failure_rate <= 1)
);

-- Indexes for session_analytics
CREATE INDEX idx_session_analytics_tenant ON session_analytics(tenant_id);
CREATE INDEX idx_session_analytics_user ON session_analytics(user_id);
CREATE INDEX idx_session_analytics_session ON session_analytics(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_session_analytics_target ON session_analytics(target_host);
CREATE INDEX idx_session_analytics_time ON session_analytics(start_time DESC);
CREATE INDEX idx_session_analytics_collected ON session_analytics(collected_at DESC);
CREATE INDEX idx_session_analytics_off_hours ON session_analytics(tenant_id, off_hours_access);
CREATE INDEX idx_session_analytics_commands ON session_analytics USING GIN (commands);

-- Unique constraint on session_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_session_analytics_session_unique
    ON session_analytics(session_id) WHERE session_id IS NOT NULL;

-- Partial index for finding orphaned sessions
CREATE INDEX idx_session_analytics_orphaned
    ON session_analytics(tenant_id, user_id, start_time)
    WHERE end_time IS NULL AND start_time < NOW() - INTERVAL '24 hours';

-- ============================================================================
-- Metrics Aggregation Tables (Time-series data)
-- ============================================================================

-- Daily user metrics (aggregated)
CREATE TABLE IF NOT EXISTS daily_user_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    metric_date DATE NOT NULL,

    -- Session counts
    total_sessions INTEGER NOT NULL DEFAULT 0,
    completed_sessions INTEGER NOT NULL DEFAULT 0,
    failed_sessions INTEGER NOT NULL DEFAULT 0,
    orphaned_sessions INTEGER NOT NULL DEFAULT 0,

    -- Command metrics
    total_commands INTEGER NOT NULL DEFAULT 0,
    unique_commands INTEGER NOT NULL DEFAULT 0,
    avg_commands_per_session NUMERIC(10,2),

    -- Duration metrics
    total_duration_seconds NUMERIC(15,2) NOT NULL DEFAULT 0,
    avg_session_duration NUMERIC(12,2),
    max_session_duration NUMERIC(12,2),

    -- Target metrics
    unique_targets INTEGER NOT NULL DEFAULT 0,
    most_frequent_target VARCHAR(255),

    -- Temporal metrics
    off_hours_sessions INTEGER NOT NULL DEFAULT 0,
    weekend_sessions INTEGER NOT NULL DEFAULT 0,

    -- Security metrics
    failed_command_rate NUMERIC(5,4),
    anomalous_sessions INTEGER NOT NULL DEFAULT 0,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, user_id, metric_date)
);

CREATE INDEX idx_daily_user_metrics_tenant_date ON daily_user_metrics(tenant_id, metric_date DESC);
CREATE INDEX idx_daily_user_metrics_user_date ON daily_user_metrics(user_id, metric_date DESC);

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

CREATE TRIGGER update_user_baselines_updated_at
    BEFORE UPDATE ON user_baselines
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_session_analytics_updated_at
    BEFORE UPDATE ON session_analytics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_daily_user_metrics_updated_at
    BEFORE UPDATE ON daily_user_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Row Level Security
-- ============================================================================

ALTER TABLE user_baselines ENABLE ROW LEVEL SECURITY;
ALTER TABLE session_analytics ENABLE ROW LEVEL SECURITY;
ALTER TABLE daily_user_metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY user_baselines_tenant_policy ON user_baselines
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

CREATE POLICY session_analytics_tenant_policy ON session_analytics
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

CREATE POLICY daily_user_metrics_tenant_policy ON daily_user_metrics
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE user_baselines IS 'Stores behavioral baselines for anomaly detection';
COMMENT ON TABLE session_analytics IS 'Stores detailed metrics for each privileged session';
COMMENT ON TABLE daily_user_metrics IS 'Aggregated daily metrics per user for trend analysis';

COMMENT ON COLUMN user_baselines.hourly_access_pattern IS 'JSON mapping hour (0-23) to mean access count for that hour';
COMMENT ON COLUMN user_baselines.weekday_access_pattern IS 'JSON mapping weekday to mean access count';
COMMENT ON COLUMN user_baselines.target_access_stats IS 'JSON mapping target host to access statistics';
COMMENT ON COLUMN user_baselines.confidence IS 'Statistical confidence of the baseline (0-100), based on sample size and period';
COMMENT ON COLUMN session_analytics.off_hours_access IS 'True if session occurred outside normal business hours (6 AM - 6 PM)';
COMMENT ON COLUMN session_analytics.commands IS 'Array of commands executed or JSON object with command counts';

-- ============================================================================
-- Helper functions for analytics
-- ============================================================================

-- Function to determine if a timestamp is off-hours
CREATE OR REPLACE FUNCTION is_off_hours(ts TIMESTAMPTZ)
RETURNS BOOLEAN AS $$
BEGIN
    -- Off-hours defined as before 6 AM or after 6 PM
    -- Adjust for timezone if needed
    RETURN EXTRACT(HOUR FROM ts AT TIME ZONE 'UTC') < 6
        OR EXTRACT(HOUR FROM ts AT TIME ZONE 'UTC') >= 18;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to calculate session failure rate
CREATE OR REPLACE FUNCTION calculate_session_failure_rate(
    p_commands INT,
    p_failed_commands INT
) RETURNS NUMERIC AS $$
BEGIN
    IF p_commands = 0 THEN
        RETURN 0;
    END IF;
    RETURN (p_failed_commands::NUMERIC / p_commands::NUMERIC);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================================================
-- Materialized views for dashboard performance
-- ============================================================================

-- User baseline summary view
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_user_baseline_summary AS
SELECT
    tenant_id,
    COUNT(*) as total_users,
    COUNT(*) FILTER (WHERE is_active) as users_with_baseline,
    AVG(confidence) as avg_confidence,
    AVG(sample_size) as avg_sample_size
FROM user_baselines
GROUP BY tenant_id;

CREATE UNIQUE INDEX idx_mv_user_baseline_summary_tenant
    ON mv_user_baseline_summary(tenant_id);

-- Session metrics summary view (last 30 days)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_session_metrics_summary AS
SELECT
    tenant_id,
    COUNT(*) as total_sessions,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT target_host) as unique_targets,
    SUM(command_count) as total_commands,
    AVG(duration_seconds) as avg_duration,
    COUNT(*) FILTER (WHERE off_hours_access) as off_hours_count,
    COUNT(*) FILTER (WHERE weekend_access) as weekend_count,
    AVG(failure_rate) as avg_failure_rate
FROM session_analytics
WHERE start_time >= NOW() - INTERVAL '30 days'
GROUP BY tenant_id;

CREATE UNIQUE INDEX idx_mv_session_metrics_summary_tenant
    ON mv_session_metrics_summary(tenant_id);

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_baseline_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_session_metrics_summary;
END;
$$ LANGUAGE plpgsql;
