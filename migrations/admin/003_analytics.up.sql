-- 003_analytics.up.sql
-- Analytics and Reporting Migration
-- Creates tables for privileged access analytics, dashboards, reports, and alerts

-- Analytics session aggregations (pre-computed session statistics)
CREATE TABLE IF NOT EXISTS analytics_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- hour, day, week, month
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Session counts
    total_sessions INTEGER NOT NULL DEFAULT 0,
    active_sessions INTEGER NOT NULL DEFAULT 0,
    completed_sessions INTEGER NOT NULL DEFAULT 0,
    failed_sessions INTEGER NOT NULL DEFAULT 0,
    terminated_sessions INTEGER NOT NULL DEFAULT 0,

    -- Session duration stats (seconds)
    avg_duration_seconds INTEGER,
    min_duration_seconds INTEGER,
    max_duration_seconds INTEGER,
    p50_duration_seconds INTEGER,
    p95_duration_seconds INTEGER,
    p99_duration_seconds INTEGER,

    -- User breakdown
    unique_users INTEGER NOT NULL DEFAULT 0,

    -- Protocol breakdown
    protocol_breakdown JSONB DEFAULT '{}', -- {"ssh": 100, "rdp": 50, "https": 25}

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT analytics_sessions_unique_period UNIQUE (tenant_id, period_type, period_start)
);

CREATE INDEX idx_analytics_sessions_tenant ON analytics_sessions(tenant_id);
CREATE INDEX idx_analytics_sessions_period ON analytics_sessions(period_start, period_end);
CREATE INDEX idx_analytics_sessions_period_type ON analytics_sessions(period_type);

-- Analytics events aggregations (pre-computed event statistics)
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- hour, day, week, month
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Event counts by outcome
    total_events INTEGER NOT NULL DEFAULT 0,
    successful_events INTEGER NOT NULL DEFAULT 0,
    failed_events INTEGER NOT NULL DEFAULT 0,
    denied_events INTEGER NOT NULL DEFAULT 0,

    -- Event counts by action type
    action_breakdown JSONB DEFAULT '{}', -- {"credential_checkout": 100, "session_start": 50}

    -- Top users
    top_users JSONB DEFAULT '[]', -- [{"user_id": "...", "username": "...", "count": 100}]

    -- Top resources
    top_resources JSONB DEFAULT '[]', -- [{"resource_type": "...", "resource_id": "...", "count": 50}]

    -- Failed authentication attempts
    failed_auth_count INTEGER NOT NULL DEFAULT 0,
    unique_failed_users INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT analytics_events_unique_period UNIQUE (tenant_id, period_type, period_start)
);

CREATE INDEX idx_analytics_events_tenant ON analytics_events(tenant_id);
CREATE INDEX idx_analytics_events_period ON analytics_events(period_start, period_end);
CREATE INDEX idx_analytics_events_period_type ON analytics_events(period_type);

-- Analytics reports (saved report configurations)
CREATE TABLE IF NOT EXISTS analytics_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    report_type VARCHAR(50) NOT NULL, -- session_summary, access_patterns, compliance, user_activity, risk_analysis

    -- Report configuration
    config JSONB NOT NULL DEFAULT '{}', -- Report-specific filters and parameters

    -- Schedule
    schedule_enabled BOOLEAN NOT NULL DEFAULT false,
    schedule_type VARCHAR(20), -- daily, weekly, monthly
    schedule_day_of_week INTEGER, -- 0-6 (Sunday-Saturday)
    schedule_day_of_month INTEGER, -- 1-31
    schedule_hour INTEGER, -- 0-23
    schedule_timezone VARCHAR(50) DEFAULT 'UTC',

    -- Delivery
    delivery_methods JSONB DEFAULT '[]', -- [{"type": "email", "config": {...}}]

    -- Status
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    last_status VARCHAR(50), -- success, error, pending
    last_error TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags JSONB DEFAULT '[]',

    -- Audit
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT valid_schedule_day CHECK (schedule_day_of_week IS NULL OR (schedule_day_of_week >= 0 AND schedule_day_of_week <= 6)),
    CONSTRAINT valid_schedule_month CHECK (schedule_day_of_month IS NULL OR (schedule_day_of_month >= 1 AND schedule_day_of_month <= 31)),
    CONSTRAINT valid_schedule_hour CHECK (schedule_hour IS NULL OR (schedule_hour >= 0 AND schedule_hour <= 23))
);

CREATE INDEX idx_reports_tenant ON analytics_reports(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_type ON analytics_reports(report_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_created_by ON analytics_reports(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_reports_tags ON analytics_reports USING GIN(tags) WHERE deleted_at IS NULL;

-- Analytics report snapshots (generated reports)
CREATE TABLE IF NOT EXISTS analytics_report_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES analytics_reports(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,

    -- Report period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Report data
    data JSONB NOT NULL DEFAULT '{}',
    summary JSONB DEFAULT '{}',

    -- File storage (for exported reports)
    file_url TEXT,
    file_format VARCHAR(20), -- pdf, csv, xlsx, json
    file_size_bytes BIGINT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, generating, completed, failed
    generated_at TIMESTAMPTZ,
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_snapshots_report ON analytics_report_snapshots(report_id);
CREATE INDEX idx_report_snapshots_tenant ON analytics_report_snapshots(tenant_id);
CREATE INDEX idx_report_snapshots_period ON analytics_report_snapshots(period_start, period_end);
CREATE INDEX idx_report_snapshots_status ON analytics_report_snapshots(status);

-- Analytics dashboards
CREATE TABLE IF NOT EXISTS analytics_dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_public BOOLEAN NOT NULL DEFAULT true,

    -- Layout configuration
    layout JSONB NOT NULL DEFAULT '{}', -- Grid layout for widgets

    -- Filters (applied to all widgets)
    global_filters JSONB DEFAULT '{}',

    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags JSONB DEFAULT '[]',

    -- Audit
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT one_default_per_tenant EXCLUDE (tenant_id WITH =) WHERE (is_default = true AND deleted_at IS NULL)
);

CREATE INDEX idx_dashboards_tenant ON analytics_dashboards(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_dashboards_created_by ON analytics_dashboards(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_dashboards_tags ON analytics_dashboards USING GIN(tags) WHERE deleted_at IS NULL;

-- Analytics dashboard widgets
CREATE TABLE IF NOT EXISTS analytics_widgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dashboard_id UUID NOT NULL REFERENCES analytics_dashboards(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    widget_type VARCHAR(50) NOT NULL, -- line_chart, bar_chart, pie_chart, stat_card, table, heatmap, gauge

    -- Position and size
    position_x INTEGER NOT NULL DEFAULT 0,
    position_y INTEGER NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 4, -- Grid units (typically 12-column grid)
    height INTEGER NOT NULL DEFAULT 3,

    -- Data source
    data_source VARCHAR(100) NOT NULL, -- sessions, events, credentials, users, risks
    query_config JSONB NOT NULL DEFAULT '{}', -- Query parameters

    -- Display configuration
    display_config JSONB NOT NULL DEFAULT '{}', -- Colors, labels, axes, etc.

    -- Refresh settings
    refresh_interval_seconds INTEGER DEFAULT 300, -- 5 minutes default

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Audit
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_position CHECK (position_x >= 0 AND position_y >= 0),
    CONSTRAINT valid_size CHECK (width > 0 AND height > 0)
);

CREATE INDEX idx_widgets_dashboard ON analytics_widgets(dashboard_id);
CREATE INDEX idx_widgets_tenant ON analytics_widgets(tenant_id);
CREATE INDEX idx_widgets_type ON analytics_widgets(widget_type);

-- Analytics alerts
CREATE TABLE IF NOT EXISTS analytics_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    alert_type VARCHAR(50) NOT NULL, -- threshold, anomaly, pattern, compliance
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- low, medium, high, critical

    -- Alert conditions
    conditions JSONB NOT NULL DEFAULT '{}', -- Alert-specific conditions

    -- Evaluation schedule
    evaluation_interval_minutes INTEGER NOT NULL DEFAULT 15,

    -- Notification
    notification_methods JSONB NOT NULL DEFAULT '[]', -- [{"type": "email", "config": {...}}]

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    last_evaluated_at TIMESTAMPTZ,
    trigger_count INTEGER NOT NULL DEFAULT 0,

    -- Cooldown
    cooldown_minutes INTEGER NOT NULL DEFAULT 60,
    next_trigger_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags JSONB DEFAULT '[]',

    -- Audit
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT valid_interval CHECK (evaluation_interval_minutes > 0),
    CONSTRAINT valid_cooldown CHECK (cooldown_minutes >= 0)
);

CREATE INDEX idx_alerts_tenant ON analytics_alerts(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_alerts_enabled ON analytics_alerts(enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_alerts_type ON analytics_alerts(alert_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_alerts_severity ON analytics_alerts(severity) WHERE deleted_at IS NULL;
CREATE INDEX idx_alerts_tags ON analytics_alerts USING GIN(tags) WHERE deleted_at IS NULL;

-- Analytics alert triggers (alert history)
CREATE TABLE IF NOT EXISTS analytics_alert_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES analytics_alerts(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,

    -- Trigger details
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    severity VARCHAR(20) NOT NULL,
    trigger_data JSONB NOT NULL DEFAULT '{}', -- What caused the alert

    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,

    -- Notifications sent
    notifications_sent JSONB DEFAULT '[]', -- [{"type": "email", "status": "sent", "at": "..."}]

    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_alert_triggers_alert ON analytics_alert_triggers(alert_id);
CREATE INDEX idx_alert_triggers_tenant ON analytics_alert_triggers(tenant_id);
CREATE INDEX idx_alert_triggers_triggered_at ON analytics_alert_triggers(triggered_at DESC);
CREATE INDEX idx_alert_triggers_resolved ON analytics_alert_triggers(resolved_at) WHERE resolved_at IS NOT NULL;

-- Analytics metrics (real-time and historical metrics)
CREATE TABLE IF NOT EXISTS analytics_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- gauge, counter, histogram, summary

    -- Metric value
    value DOUBLE PRECISION NOT NULL,
    labels JSONB DEFAULT '{}', -- Additional dimensions (e.g., {"protocol": "ssh"})

    -- Timestamp
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Metadata
    metadata JSONB DEFAULT '{}'
) PARTITION BY RANGE (recorded_at);

CREATE INDEX idx_metrics_tenant ON analytics_metrics(tenant_id);
CREATE INDEX idx_metrics_name ON analytics_metrics(metric_name);
CREATE INDEX idx_metrics_recorded_at ON analytics_metrics(recorded_at);
CREATE INDEX idx_metrics_labels ON analytics_metrics USING GIN(labels);

-- Create daily partitions for metrics (last 30 days + default)
CREATE TABLE analytics_metrics_default PARTITION OF analytics_metrics DEFAULT;

-- User activity analytics
CREATE TABLE IF NOT EXISTS analytics_user_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    period_type VARCHAR(20) NOT NULL, -- day, week, month
    period_start TIMESTAMPTZ NOT NULL,

    -- Activity counts
    sessions_created INTEGER NOT NULL DEFAULT 0,
    credentials_accessed INTEGER NOT NULL DEFAULT 0,
    approvals_requested INTEGER NOT NULL DEFAULT 0,
    approvals_granted INTEGER NOT NULL DEFAULT 0,

    -- Time-based stats
    total_active_seconds BIGINT NOT NULL DEFAULT 0,
    avg_daily_seconds INTEGER,

    -- Risk indicators
    high_risk_sessions INTEGER NOT NULL DEFAULT 0,
    policy_violations INTEGER NOT NULL DEFAULT 0,
    failed_auth_attempts INTEGER NOT NULL DEFAULT 0,

    -- Unusual activity flags
    is_anomaly BOOLEAN NOT NULL DEFAULT false,
    anomaly_score NUMERIC(5,2), -- 0-100

    -- Access patterns
    off_hours_access BOOLEAN NOT NULL DEFAULT false,
    first_access_time TIMESTAMPTZ,
    last_access_time TIMESTAMPTZ,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_activity_unique_period UNIQUE (tenant_id, user_id, period_type, period_start)
);

CREATE INDEX idx_user_activity_tenant ON analytics_user_activity(tenant_id);
CREATE INDEX idx_user_activity_user ON analytics_user_activity(user_id);
CREATE INDEX idx_user_activity_period ON analytics_user_activity(period_start);
CREATE INDEX idx_user_activity_anomaly ON analytics_user_activity(is_anomaly) WHERE is_anomaly = true;

-- Risk scoring table
CREATE TABLE IF NOT EXISTS analytics_risk_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL, -- user, target, credential
    entity_id UUID NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Risk components (0-100 each)
    access_frequency_score NUMERIC(5,2) DEFAULT 0,
    time_pattern_score NUMERIC(5,2) DEFAULT 0,
    geography_score NUMERIC(5,2) DEFAULT 0,
    behavior_drift_score NUMERIC(5,2) DEFAULT 0,
    compliance_score NUMERIC(5,2) DEFAULT 0,

    -- Overall risk
    overall_risk_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    risk_level VARCHAR(20) NOT NULL DEFAULT 'low', -- low, medium, high, critical

    -- Contributing factors
    factors JSONB DEFAULT '{}',

    -- Trend
    previous_score NUMERIC(5,2),
    score_change NUMERIC(5,2),

    metadata JSONB DEFAULT '{}',

    CONSTRAINT risk_entity_unique UNIQUE (entity_type, entity_id, calculated_at)
);

CREATE INDEX idx_risk_scores_tenant ON analytics_risk_scores(tenant_id);
CREATE INDEX idx_risk_scores_entity ON analytics_risk_scores(entity_type, entity_id);
CREATE INDEX idx_risk_scores_calculated ON analytics_risk_scores(calculated_at DESC);
CREATE INDEX idx_risk_scores_level ON analytics_risk_scores(risk_level);
CREATE INDEX idx_risk_scores_overall ON analytics_risk_scores(overall_risk_score);

-- Analytics materialized view refresh helper
CREATE TABLE IF NOT EXISTS analytics_refresh_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    materialized_view VARCHAR(255) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL, -- running, completed, failed
    duration_ms INTEGER,
    error_message TEXT,
    rows_affected INTEGER
);

CREATE INDEX idx_refresh_log_view ON analytics_refresh_log(materialized_view);
CREATE INDEX idx_refresh_log_started ON analytics_refresh_log(started_at DESC);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_analytics_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply triggers
CREATE TRIGGER analytics_sessions_updated_at
    BEFORE UPDATE ON analytics_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_events_updated_at
    BEFORE UPDATE ON analytics_events
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_reports_updated_at
    BEFORE UPDATE ON analytics_reports
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_dashboards_updated_at
    BEFORE UPDATE ON analytics_dashboards
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_widgets_updated_at
    BEFORE UPDATE ON analytics_widgets
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_alerts_updated_at
    BEFORE UPDATE ON analytics_alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

CREATE TRIGGER analytics_user_activity_updated_at
    BEFORE UPDATE ON analytics_user_activity
    FOR EACH ROW
    EXECUTE FUNCTION update_analytics_updated_at();

-- Insert default dashboard
INSERT INTO analytics_dashboards (id, tenant_id, name, description, is_default, is_public, layout, global_filters, created_by) VALUES
('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000000'::uuid,
 'Overview Dashboard',
 'Default analytics dashboard with key metrics and visualizations',
 true, true,
 '{
   "widgets": [
     {"id": "total-sessions", "x": 0, "y": 0, "w": 3, "h": 2},
     {"id": "active-users", "x": 3, "y": 0, "w": 3, "h": 2},
     {"id": "risk-score", "x": 6, "y": 0, "w": 3, "h": 2},
     {"id": "compliance-rate", "x": 9, "y": 0, "w": 3, "h": 2},
     {"id": "session-trends", "x": 0, "y": 2, "w": 8, "h": 4},
     {"id": "top-users", "x": 8, "y": 2, "w": 4, "h": 4}
   ]
 }'::jsonb,
 '{"timeRange": "7d"}'::jsonb,
 '00000000-0000-0000-0000-000000000000'::uuid
)
ON CONFLICT DO NOTHING;

-- Insert default widgets for overview dashboard
INSERT INTO analytics_widgets (dashboard_id, tenant_id, name, widget_type, position_x, position_y, width, height, data_source, query_config, display_config, refresh_interval_seconds, created_by)
SELECT
  '00000000-0000-0000-0001-000000000001'::uuid,
  '00000000-0000-0000-0000-000000000000'::uuid,
  widget.name,
  widget.widget_type,
  widget.position_x,
  widget.position_y,
  widget.width,
  widget.height,
  widget.data_source,
  widget.query_config,
  widget.display_config,
  widget.refresh_interval_seconds,
  '00000000-0000-0000-0000-000000000000'::uuid
FROM (
  VALUES
    ('Total Sessions', 'stat_card', 0, 0, 3, 2, 'sessions',
     '{"period": "7d", "metric": "total_sessions"}'::jsonb,
     '{"label": "Sessions", "showTrend": true, "color": "#3b82f6"}'::jsonb, 300),
    ('Active Users', 'stat_card', 3, 0, 3, 2, 'users',
     '{"period": "7d", "metric": "unique_users"}'::jsonb,
     '{"label": "Active Users", "showTrend": true, "color": "#10b981"}'::jsonb, 300),
    ('Risk Score', 'gauge', 6, 0, 3, 2, 'risks',
     '{"period": "7d", "metric": "avg_risk_score"}'::jsonb,
     '{"label": "Avg Risk", "min": 0, "max": 100, "thresholds": [50, 70, 90]}'::jsonb, 300),
    ('Compliance Rate', 'stat_card', 9, 0, 3, 2, 'events',
     '{"period": "7d", "metric": "compliance_rate"}'::jsonb,
     '{"label": "Compliance", "suffix": "%", "showTrend": true, "color": "#8b5cf6"}'::jsonb, 300),
    ('Session Trends', 'line_chart', 0, 2, 8, 4, 'sessions',
     '{"period": "30d", "granularity": "day"}'::jsonb,
     '{"xAxis": "date", "yAxis": "count", "multipleSeries": ["total", "active", "completed"]}'::jsonb, 300),
    ('Top Users', 'table', 8, 2, 4, 4, 'users',
     '{"period": "7d", "limit": 10, "sortBy": "session_count"}'::jsonb,
     '{"columns": ["username", "session_count", "total_duration", "last_seen"]}'::jsonb, 300)
) AS widget(name, widget_type, position_x, position_y, width, height, data_source, query_config, display_config, refresh_interval_seconds)
ON CONFLICT DO NOTHING;

-- Insert default alerts
INSERT INTO analytics_alerts (id, tenant_id, name, description, alert_type, severity, conditions, evaluation_interval_minutes, notification_methods, created_by) VALUES
('00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000000'::uuid,
 'High Failed Authentication Rate',
 'Alerts when the failed authentication rate exceeds threshold',
 'threshold', 'high',
 '{"metric": "failed_auth_rate", "operator": "greater_than", "threshold": 10, "window_minutes": 15}'::jsonb,
 15,
 '[{"type": "email", "config": {"recipients": ["security@example.com"]}}]'::jsonb,
 '00000000-0000-0000-0000-000000000000'::uuid
),
('00000000-0000-0000-0002-000000000002', '00000000-0000-0000-0000-000000000000'::uuid,
 'Unusual Access Pattern',
 'Detects unusual access patterns outside business hours',
 'pattern', 'medium',
 '{"pattern": "after_hours_access", "days": ["saturday", "sunday"], "hours": {"start": 18, "end": 6}}'::jsonb,
 15,
 '[{"type": "email", "config": {"recipients": ["security@example.com"]}}]'::jsonb,
 '00000000-0000-0000-0000-000000000000'::uuid
),
('00000000-0000-0000-0002-000000000003', '00000000-0000-0000-0000-000000000000'::uuid,
 'Critical Risk Score',
 'Alerts when any entity reaches critical risk level',
 'threshold', 'critical',
 '{"metric": "risk_score", "operator": "greater_or_equal", "threshold": 90}'::jsonb,
 5,
 '[{"type": "email", "config": {"recipients": ["security@example.com"]}}, {"type": "webhook", "config": {"url": "https://hooks.example.com/alerts"}}]'::jsonb,
 '00000000-0000-0000-0000-000000000000'::uuid
)
ON CONFLICT DO NOTHING;

-- Insert default report templates
INSERT INTO analytics_reports (id, tenant_id, name, description, report_type, config, schedule_enabled, created_by) VALUES
('00000000-0000-0000-0003-000000000001', '00000000-0000-0000-0000-000000000000'::uuid,
 'Weekly Access Summary',
 'Weekly summary of privileged access activities',
 'session_summary',
 '{"period": "week", "include_charts": true, "include_details": true}'::jsonb,
 false,
 '00000000-0000-0000-0000-000000000000'::uuid
),
('00000000-0000-0000-0003-000000000002', '00000000-0000-0000-0000-000000000000'::uuid,
 'Compliance Report',
 'Monthly compliance and audit report',
 'compliance',
 '{"period": "month", "include_policy_violations": true, "include_unapproved_access": true}'::jsonb,
 false,
 '00000000-0000-0000-0000-000000000000'::uuid
),
('00000000-0000-0000-0003-000000000003', '00000000-0000-0000-0000-000000000000'::uuid,
 'User Activity Report',
 'Detailed user activity and access patterns',
 'user_activity',
 '{"period": "month", "include_risk_scores": true, "include_anomalies": true}'::jsonb,
 false,
 '00000000-0000-0000-0000-000000000000'::uuid
)
ON CONFLICT DO NOTHING;

-- Grant permissions (adjust schema/user as needed)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_sessions, analytics_events, analytics_reports, analytics_report_snapshots, analytics_dashboards, analytics_widgets, analytics_alerts, analytics_alert_triggers, analytics_metrics, analytics_user_activity, analytics_risk_scores, analytics_refresh_log TO openpam_app;
