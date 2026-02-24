-- Analytics Service Database Schema
-- This migration creates tables for session analytics, user activity tracking,
-- command frequency analysis, compliance reporting, and anomaly detection.

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Session Analytics: Aggregated session metrics
CREATE TABLE IF NOT EXISTS session_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    date DATE NOT NULL,
    hour SMALLINT NOT NULL CHECK (hour >= 0 AND hour <= 23),

    -- Session counts
    total_sessions INTEGER NOT NULL DEFAULT 0,
    active_sessions INTEGER NOT NULL DEFAULT 0,
    completed_sessions INTEGER NOT NULL DEFAULT 0,
    terminated_sessions INTEGER NOT NULL DEFAULT 0,
    failed_sessions INTEGER NOT NULL DEFAULT 0,

    -- Session type breakdown
    ssh_sessions INTEGER NOT NULL DEFAULT 0,
    rdp_sessions INTEGER NOT NULL DEFAULT 0,
    database_sessions INTEGER NOT NULL DEFAULT 0,
    kubernetes_sessions INTEGER NOT NULL DEFAULT 0,
    web_sessions INTEGER NOT NULL DEFAULT 0,

    -- Duration metrics (in seconds)
    avg_duration_seconds NUMERIC(12, 2),
    min_duration_seconds INTEGER,
    max_duration_seconds INTEGER,
    total_duration_seconds BIGINT NOT NULL DEFAULT 0,

    -- Concurrent sessions
    peak_concurrent_sessions INTEGER NOT NULL DEFAULT 0,
    peak_concurrent_time TIMESTAMPTZ,

    -- User activity
    unique_users INTEGER NOT NULL DEFAULT 0,
    unique_targets INTEGER NOT NULL DEFAULT 0,

    -- Recording metrics
    total_recordings BIGINT NOT NULL DEFAULT 0,
    recording_size_bytes BIGINT NOT NULL DEFAULT 0,
    recording_duration_seconds BIGINT NOT NULL DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT session_analytics_unique UNIQUE (tenant_id, date, hour)
);

-- Indexes for session_analytics
CREATE INDEX idx_session_analytics_tenant_date ON session_analytics(tenant_id, date);
CREATE INDEX idx_session_analytics_date_range ON session_analytics(date DESC);
CREATE INDEX idx_session_analytics_tenant_hour ON session_analytics(tenant_id, date, hour);

-- User Activity: Individual user activity patterns
CREATE TABLE IF NOT EXISTS user_activity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    date DATE NOT NULL,
    hour SMALLINT NOT NULL CHECK (hour >= 0 AND hour <= 23),

    -- Activity metrics
    sessions_initiated INTEGER NOT NULL DEFAULT 0,
    sessions_completed INTEGER NOT NULL DEFAULT 0,
    commands_executed INTEGER NOT NULL DEFAULT 0,
    targets_accessed INTEGER NOT NULL DEFAULT 0,

    -- Time-based metrics
    total_session_seconds BIGINT NOT NULL DEFAULT 0,
    active_seconds BIGINT NOT NULL DEFAULT 0,
    idle_seconds BIGINT NOT NULL DEFAULT 0,

    -- Access patterns
    first_access_time TIMESTAMPTZ,
    last_access_time TIMESTAMPTZ,
    peak_hour SMALLINT CHECK (peak_hour >= 0 AND peak_hour <= 23),

    -- Geographic (if available)
    country_code VARCHAR(2),
    city VARCHAR(100),

    -- Risk indicators
    off_hours_access BOOLEAN NOT NULL DEFAULT FALSE,
    unusual_access BOOLEAN NOT NULL DEFAULT FALSE,
    risk_score NUMERIC(5, 2) DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_activity_unique UNIQUE (tenant_id, user_id, date, hour)
);

-- Indexes for user_activity
CREATE INDEX idx_user_activity_tenant_user ON user_activity(tenant_id, user_id);
CREATE INDEX idx_user_activity_date ON user_activity(date DESC);
CREATE INDEX idx_user_activity_user_date ON user_activity(user_id, date DESC);
CREATE INDEX idx_user_activity_risk ON user_activity(tenant_id, risk_score DESC) WHERE risk_score > 50;

-- Command Frequency: Track command execution patterns (partitioned by month)
CREATE TABLE IF NOT EXISTS command_frequency (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    date DATE NOT NULL,
    hour SMALLINT NOT NULL CHECK (hour >= 0 AND hour <= 23),

    -- Command details
    command_hash VARCHAR(64) NOT NULL,  -- SHA-256 hash of normalized command
    command_pattern TEXT NOT NULL,       -- Pattern for grouping (e.g., "rm -rf *")
    base_command VARCHAR(100) NOT NULL,  -- First word (executable)

    -- Execution context
    session_id UUID NOT NULL,
    user_id UUID NOT NULL,
    target_host VARCHAR(255) NOT NULL,

    -- Risk assessment
    risk_level VARCHAR(20) NOT NULL DEFAULT 'low',  -- low, medium, high, critical
    is_dangerous BOOLEAN NOT NULL DEFAULT FALSE,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,

    -- Timing
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Metadata
    exit_code SMALLINT,
    execution_duration_ms INTEGER,
    metadata JSONB DEFAULT '{}',

    CONSTRAINT command_frequency_check CHECK (
        risk_level IN ('low', 'medium', 'high', 'critical')
    )
) PARTITION BY RANGE (date);

-- Create monthly partitions for command_frequency
-- Current and future partitions will be created automatically
CREATE TABLE IF NOT EXISTS command_frequency_2025_01 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_02 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_03 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_04 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_05 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_06 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_07 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_08 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_09 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_10 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_11 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE IF NOT EXISTS command_frequency_2025_12 PARTITION OF command_frequency
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Indexes for command_frequency (local to partitions)
CREATE INDEX idx_command_frequency_tenant_date ON command_frequency(tenant_id, date);
CREATE INDEX idx_command_frequency_user ON command_frequency(user_id, executed_at DESC);
CREATE INDEX idx_command_frequency_command_hash ON command_frequency(command_hash);
CREATE INDEX idx_command_frequency_risk ON command_frequency(risk_level, is_dangerous);
CREATE INDEX idx_command_frequency_target ON command_frequency(target_host);

-- Compliance Reports: Framework evaluation results
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,  -- SOC2, ISO27001, PCI-DSS, HIPAA, NIST-800-53

    -- Report metadata
    version VARCHAR(20) NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by UUID NOT NULL,

    -- Overall status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, passed, failed, partial
    overall_score NUMERIC(5, 2),  -- 0-100
    total_controls INTEGER NOT NULL DEFAULT 0,
    passed_controls INTEGER NOT NULL DEFAULT 0,
    failed_controls INTEGER NOT NULL DEFAULT 0,
    skipped_controls INTEGER NOT NULL DEFAULT 0,

    -- Evaluation period
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Report content
    summary TEXT,
    findings JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_reports_status_check CHECK (
        status IN ('pending', 'passed', 'failed', 'partial')
    ),
    CONSTRAINT compliance_reports_framework_check CHECK (
        framework IN ('SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'NIST-800-53', 'GDPR', 'CUSTOM')
    )
);

-- Indexes for compliance_reports
CREATE INDEX idx_compliance_reports_tenant ON compliance_reports(tenant_id, generated_at DESC);
CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework, generated_at DESC);
CREATE INDEX idx_compliance_reports_status ON compliance_reports(status);

-- Compliance Control Evaluations: Detailed control results
CREATE TABLE IF NOT EXISTS compliance_control_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,

    -- Control identification
    control_id VARCHAR(100) NOT NULL,
    control_name TEXT NOT NULL,
    control_category VARCHAR(100),

    -- Evaluation result
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- passed, failed, skipped, partial
    score NUMERIC(5, 2),

    -- Evidence
    evidence_count INTEGER NOT NULL DEFAULT 0,
    evidence_urls TEXT[],
    findings TEXT,
    remediation_steps TEXT[],

    -- Metadata
    metadata JSONB DEFAULT '{}',
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_control_status CHECK (
        status IN ('passed', 'failed', 'skipped', 'partial')
    )
);

-- Indexes for compliance_control_evaluations
CREATE INDEX idx_compliance_control_report ON compliance_control_evaluations(report_id);
CREATE INDEX idx_compliance_control_tenant ON compliance_control_evaluations(tenant_id);
CREATE INDEX idx_compliance_control_id ON compliance_control_evaluations(control_id);

-- Compliance Exceptions: Approved exceptions to controls
CREATE TABLE IF NOT EXISTS compliance_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Exception details
    control_id VARCHAR(100) NOT NULL,
    control_name TEXT NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Exception status
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, approved, denied, expired, revoked
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',  -- low, medium, high, critical

    -- Approval workflow
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    -- Justification
    justification TEXT NOT NULL,
    business_reason TEXT,
    compensating_controls TEXT[],

    -- Risk acceptance
    risk_accepted_by UUID,
    risk_accepted_at TIMESTAMPTZ,

    -- Review
    review_date DATE,
    review_notes TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_exceptions_status CHECK (
        status IN ('pending', 'approved', 'denied', 'expired', 'revoked')
    ),
    CONSTRAINT compliance_exceptions_risk CHECK (
        risk_level IN ('low', 'medium', 'high', 'critical')
    )
);

-- Indexes for compliance_exceptions
CREATE INDEX idx_compliance_exceptions_tenant ON compliance_exceptions(tenant_id, status);
CREATE INDEX idx_compliance_exceptions_control ON compliance_exceptions(control_id, status);
CREATE INDEX idx_compliance_exceptions_framework ON compliance_exceptions(framework, status);
CREATE INDEX idx_compliance_exceptions_expires ON compliance_exceptions(expires_at) WHERE status = 'approved';

-- Anomaly Detection: Detected anomalies and alerts
CREATE TABLE IF NOT EXISTS anomaly_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    anomaly_type VARCHAR(50) NOT NULL,  -- behavioral, temporal, spatial, pattern, volumetric

    -- Context
    user_id UUID,
    session_id UUID,
    target_host VARCHAR(255),

    -- Anomaly details
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',  -- low, medium, high, critical
    confidence_score NUMERIC(5, 2) NOT NULL,  -- 0-100
    risk_score NUMERIC(5, 2) NOT NULL DEFAULT 0,  -- Calculated risk score

    -- Description
    title TEXT NOT NULL,
    description TEXT,
    indicators JSONB DEFAULT '{}',

    -- Detection metadata
    detection_method VARCHAR(100) NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model_version VARCHAR(50),

    -- Status and response
    status VARCHAR(20) NOT NULL DEFAULT 'open',  -- open, investigating, resolved, false_positive, ignored
    assigned_to UUID,
    resolution_notes TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,

    -- Automated response
    auto_triggered BOOLEAN NOT NULL DEFAULT FALSE,
    auto_action_taken VARCHAR(100),

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT anomaly_detections_severity CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    ),
    CONSTRAINT anomaly_detections_status CHECK (
        status IN ('open', 'investigating', 'resolved', 'false_positive', 'ignored')
    ),
    CONSTRAINT anomaly_detections_type CHECK (
        anomaly_type IN ('behavioral', 'temporal', 'spatial', 'pattern', 'volumetric', 'ransomware')
    )
);

-- Indexes for anomaly_detections
CREATE INDEX idx_anomaly_detections_tenant ON anomaly_detections(tenant_id, detected_at DESC);
CREATE INDEX idx_anomaly_detections_user ON anomaly_detections(user_id, detected_at DESC);
CREATE INDEX idx_anomaly_detections_status ON anomaly_detections(status, detected_at DESC);
CREATE INDEX idx_anomaly_detections_severity ON anomaly_detections(severity, status);
CREATE INDEX idx_anomaly_detections_type ON anomaly_detections(anomaly_type, detected_at DESC);

-- Ransomware Detection Events: Specific high-priority security events
CREATE TABLE IF NOT EXISTS ransomware_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    detection_id UUID NOT NULL REFERENCES anomaly_detections(id) ON DELETE CASCADE,

    -- Ransomware-specific indicators
    encryption_activity BOOLEAN NOT NULL DEFAULT FALSE,
    mass_file_modification BOOLEAN NOT NULL DEFAULT FALSE,
    suspicious_processes TEXT[],
    affected_paths TEXT[],

    -- Impact assessment
    files_affected INTEGER DEFAULT 0,
    systems_affected INTEGER DEFAULT 0,
    data_exfiltrated BOOLEAN NOT NULL DEFAULT FALSE,

    -- Emergency response
    emergency_triggered BOOLEAN NOT NULL DEFAULT FALSE,
    sessions_terminated INTEGER DEFAULT 0,
    credentials_revoked INTEGER DEFAULT 0,

    -- Post-incident
    containment_status VARCHAR(50),
    recovery_status VARCHAR(50),

    -- Metadata
    raw_indicators JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for ransomware_events
CREATE INDEX idx_ransomware_events_tenant ON ransomware_events(tenant_id, created_at DESC);
CREATE INDEX idx_ransomware_events_detection ON ransomware_events(detection_id);
CREATE INDEX idx_ransomware_events_status ON ransomware_events(containment_status);

-- Command Blacklist: Dangerous commands that can be blocked
CREATE TABLE IF NOT EXISTS command_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,  -- NULL means global blacklist

    -- Command pattern
    command_pattern TEXT NOT NULL,
    pattern_type VARCHAR(20) NOT NULL DEFAULT 'exact',  -- exact, regex, glob
    base_command VARCHAR(100),

    -- Action
    action VARCHAR(20) NOT NULL DEFAULT 'block',  -- block, warn, allow, audit
    severity VARCHAR(20) NOT NULL DEFAULT 'high',

    -- Context restrictions
    applies_to_users UUID[],
    applies_to_groups UUID[],
    applies_to_targets TEXT[],

    -- Exception handling
    allow_override BOOLEAN NOT NULL DEFAULT FALSE,
    override_roles TEXT[],

    -- Reasoning
    reason TEXT NOT NULL,
    risk_category VARCHAR(50),  -- data_deletion, privilege_escalation, exfiltration, system_modification

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT command_blacklist_action CHECK (
        action IN ('block', 'warn', 'allow', 'audit')
    ),
    CONSTRAINT command_blacklist_pattern_type CHECK (
        pattern_type IN ('exact', 'regex', 'glob')
    )
);

-- Indexes for command_blacklist
CREATE INDEX idx_command_blacklist_tenant ON command_blacklist(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_command_blacklist_global ON command_blacklist(id) WHERE tenant_id IS NULL;
CREATE INDEX idx_command_blacklist_enabled ON command_blacklist(enabled);
CREATE INDEX idx_command_blacklist_base_command ON command_blacklist(base_command);

-- SSH Key Analytics: Track SSH key usage and access patterns
CREATE TABLE IF NOT EXISTS ssh_key_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    ssh_key_id UUID NOT NULL,

    -- Usage metrics
    date DATE NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,
    unique_users INTEGER NOT NULL DEFAULT 0,
    unique_targets INTEGER NOT NULL DEFAULT 0,

    -- Access patterns
    first_use_time TIMESTAMPTZ,
    last_use_time TIMESTAMPTZ,
    avg_session_duration_seconds NUMERIC(12, 2),

    -- Risk indicators
    off_hours_usage INTEGER NOT NULL DEFAULT 0,
    unusual_source_usage INTEGER NOT NULL DEFAULT 0,
    failed_attempts INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ssh_key_analytics_unique UNIQUE (tenant_id, ssh_key_id, date)
);

-- Indexes for ssh_key_analytics
CREATE INDEX idx_ssh_key_analytics_tenant ON ssh_key_analytics(tenant_id, date DESC);
CREATE INDEX idx_ssh_key_analytics_key ON ssh_key_analytics(ssh_key_id, date DESC);
CREATE INDEX idx_ssh_key_analytics_usage ON ssh_key_analytics(usage_count DESC);

-- Materialized Views for performance
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_session_summary AS
SELECT
    tenant_id,
    date,
    SUM(total_sessions) as total_sessions,
    SUM(active_sessions) as active_sessions,
    SUM(completed_sessions) as completed_sessions,
    SUM(terminated_sessions) as terminated_sessions,
    SUM(ssh_sessions) as ssh_sessions,
    SUM(rdp_sessions) as rdp_sessions,
    SUM(database_sessions) as database_sessions,
    SUM(kubernetes_sessions) as kubernetes_sessions,
    AVG(avg_duration_seconds) as avg_duration_seconds,
    SUM(unique_users) as unique_users,
    SUM(unique_targets) as unique_targets
FROM session_analytics
GROUP BY tenant_id, date
ORDER BY date DESC;

-- Create index on materialized view
CREATE UNIQUE INDEX idx_mv_session_summary_unique ON mv_session_summary(tenant_id, date);

-- Materialized view for user risk scoring
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_user_risk_summary AS
SELECT
    tenant_id,
    user_id,
    date,
    SUM(sessions_initiated) as total_sessions,
    SUM(commands_executed) as total_commands,
    AVG(risk_score) as avg_risk_score,
    SUM(CASE WHEN off_hours_access THEN 1 ELSE 0 END) as off_hours_count,
    SUM(CASE WHEN unusual_access THEN 1 ELSE 0 END) as unusual_access_count
FROM user_activity
GROUP BY tenant_id, user_id, date
ORDER BY date DESC;

-- Create index on user risk summary
CREATE INDEX idx_mv_user_risk_summary ON mv_user_risk_summary(tenant_id, user_id, date DESC);

-- Functions for automatic partition management
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    partition_date := date_trunc('month', CURRENT_DATE + interval '1 month');
    partition_name := 'command_frequency_' || to_char(partition_date, 'YYYY_MM');
    start_date := partition_date::TEXT;
    end_date := (partition_date + interval '1 month')::TEXT;

    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF command_frequency FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$ LANGUAGE plpgsql;

-- Trigger function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER update_session_analytics_updated_at
    BEFORE UPDATE ON session_analytics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_activity_updated_at
    BEFORE UPDATE ON user_activity
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_compliance_exceptions_updated_at
    BEFORE UPDATE ON compliance_exceptions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_anomaly_detections_updated_at
    BEFORE UPDATE ON anomaly_detections
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ransomware_events_updated_at
    BEFORE UPDATE ON ransomware_events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ssh_key_analytics_updated_at
    BEFORE UPDATE ON ssh_key_analytics
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_session_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_user_risk_summary;
END;
$$ LANGUAGE plpgsql;
