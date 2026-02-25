-- Analytics Tables Migration for Audit Service
-- This migration creates tables for compliance reports, exceptions,
-- anomaly detection, SSH key analytics, and command blacklist.

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For pattern matching

-- ============================================================================
-- Compliance Reports Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Report identification
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,  -- SOC2, ISO27001, PCI-DSS, HIPAA, NIST-800-53, GDPR
    version VARCHAR(50),

    -- Generation metadata
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by UUID NOT NULL,

    -- Status and scoring
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, passed, failed, partial
    overall_score NUMERIC(5,2),

    -- Control counts
    total_controls INTEGER NOT NULL DEFAULT 0,
    passed_controls INTEGER NOT NULL DEFAULT 0,
    failed_controls INTEGER NOT NULL DEFAULT 0,
    skipped_controls INTEGER NOT NULL DEFAULT 0,

    -- Report period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Report content (stored as JSONB for flexibility)
    summary TEXT,
    findings JSONB,
    recommendations JSONB,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_reports_framework_check CHECK (framework IN ('SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'NIST-800-53', 'GDPR', 'CUSTOM')),
    CONSTRAINT compliance_reports_status_check CHECK (status IN ('pending', 'passed', 'failed', 'partial')),
    CONSTRAINT compliance_reports_score_check CHECK (overall_score IS NULL OR (overall_score >= 0 AND overall_score <= 100))
);

-- Indexes for compliance_reports
CREATE INDEX idx_compliance_reports_tenant ON compliance_reports(tenant_id);
CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_status ON compliance_reports(status);
CREATE INDEX idx_compliance_reports_generated_at ON compliance_reports(generated_at DESC);
CREATE INDEX idx_compliance_reports_period ON compliance_reports(period_start, period_end);
CREATE INDEX idx_compliance_reports_metadata ON compliance_reports USING GIN (metadata);

-- ============================================================================
-- Compliance Control Evaluations Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_control_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,

    -- Control identification
    control_id VARCHAR(100) NOT NULL,
    control_name VARCHAR(255) NOT NULL,
    control_category VARCHAR(100),

    -- Evaluation results
    status VARCHAR(50) NOT NULL, -- passed, failed, skipped, not_applicable
    score NUMERIC(5,2),

    -- Evidence and findings
    evidence_count INTEGER NOT NULL DEFAULT 0,
    evidence_urls TEXT[],
    findings TEXT,
    remediation_steps TEXT[],

    metadata JSONB DEFAULT '{}',
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_eval_status_check CHECK (status IN ('passed', 'failed', 'skipped', 'not_applicable')),
    CONSTRAINT compliance_eval_score_check CHECK (score IS NULL OR (score >= 0 AND score <= 100))
);

-- Indexes for compliance_control_evaluations
CREATE INDEX idx_compliance_eval_report ON compliance_control_evaluations(report_id);
CREATE INDEX idx_compliance_eval_tenant ON compliance_control_evaluations(tenant_id);
CREATE INDEX idx_compliance_eval_control_id ON compliance_control_evaluations(control_id);
CREATE INDEX idx_compliance_eval_status ON compliance_control_evaluations(status);

-- ============================================================================
-- Compliance Exceptions Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Control reference
    control_id VARCHAR(100) NOT NULL,
    control_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Exception status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, denied, expired, revoked
    risk_level VARCHAR(50) NOT NULL, -- low, medium, high, critical

    -- Request information
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Approval information
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,

    -- Justification and compensating controls
    justification TEXT NOT NULL,
    business_reason TEXT,
    compensating_controls TEXT[],

    -- Risk acceptance
    risk_accepted_by UUID,
    risk_accepted_at TIMESTAMPTZ,

    -- Review tracking
    review_date TIMESTAMPTZ,
    review_notes TEXT,

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_exc_status_check CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'revoked')),
    CONSTRAINT compliance_exc_risk_check CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

-- Indexes for compliance_exceptions
CREATE INDEX idx_compliance_exc_tenant ON compliance_exceptions(tenant_id);
CREATE INDEX idx_compliance_exc_control ON compliance_exceptions(control_id);
CREATE INDEX idx_compliance_exc_framework ON compliance_exceptions(framework);
CREATE INDEX idx_compliance_exc_status ON compliance_exceptions(status);
CREATE INDEX idx_compliance_exc_expires_at ON compliance_exceptions(expires_at);
CREATE INDEX idx_compliance_exc_metadata ON compliance_exceptions USING GIN (metadata);

-- ============================================================================
-- Anomaly Detections Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS anomaly_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Anomaly classification
    anomaly_type VARCHAR(50) NOT NULL, -- behavioral, temporal, spatial, pattern, volumetric, ransomware

    -- Related entities
    user_id UUID,
    session_id UUID,
    target_host VARCHAR(255),

    -- Severity and risk
    severity VARCHAR(50) NOT NULL, -- low, medium, high, critical
    confidence_score NUMERIC(5,2) NOT NULL, -- 0-100
    risk_score NUMERIC(5,2) NOT NULL, -- 0-100

    -- Anomaly details
    title VARCHAR(500) NOT NULL,
    description TEXT,
    indicators JSONB, -- Detailed indicators of the anomaly

    -- Detection metadata
    detection_method VARCHAR(100) NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model_version VARCHAR(50),

    -- Status and resolution
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, investigating, resolved, false_positive, ignored
    assigned_to UUID,
    resolution_notes TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,

    -- Automated response
    auto_triggered BOOLEAN NOT NULL DEFAULT false,
    auto_action_taken VARCHAR(255),

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT anomaly_type_check CHECK (anomaly_type IN ('behavioral', 'temporal', 'spatial', 'pattern', 'volumetric', 'ransomware')),
    CONSTRAINT anomaly_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    CONSTRAINT anomaly_confidence_check CHECK (confidence_score >= 0 AND confidence_score <= 100),
    CONSTRAINT anomaly_risk_check CHECK (risk_score >= 0 AND risk_score <= 100),
    CONSTRAINT anomaly_status_check CHECK (status IN ('open', 'investigating', 'resolved', 'false_positive', 'ignored'))
);

-- Indexes for anomaly_detections
CREATE INDEX idx_anomaly_tenant ON anomaly_detections(tenant_id);
CREATE INDEX idx_anomaly_user ON anomaly_detections(user_id);
CREATE INDEX idx_anomaly_session ON anomaly_detections(session_id);
CREATE INDEX idx_anomaly_type ON anomaly_detections(anomaly_type);
CREATE INDEX idx_anomaly_severity ON anomaly_detections(severity);
CREATE INDEX idx_anomaly_status ON anomaly_detections(status);
CREATE INDEX idx_anomaly_detected_at ON anomaly_detections(detected_at DESC);
CREATE INDEX idx_anomaly_indicators ON anomaly_detections USING GIN (indicators);

-- Partial index for open anomalies (most common query)
CREATE INDEX idx_anomaly_open_tenant ON anomaly_detections(tenant_id, detected_at DESC)
    WHERE status = 'open';

-- ============================================================================
-- Ransomware Events Table (High-priority security events)
-- ============================================================================
CREATE TABLE IF NOT EXISTS ransomware_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    detection_id UUID NOT NULL REFERENCES anomaly_detections(id) ON DELETE CASCADE,

    -- Detection indicators
    encryption_activity BOOLEAN NOT NULL DEFAULT false,
    mass_file_modification BOOLEAN NOT NULL DEFAULT false,
    suspicious_processes TEXT[],
    affected_paths TEXT[],

    -- Impact assessment
    files_affected INTEGER NOT NULL DEFAULT 0,
    systems_affected INTEGER NOT NULL DEFAULT 0,
    data_exfiltrated BOOLEAN NOT NULL DEFAULT false,

    -- Emergency response
    emergency_triggered BOOLEAN NOT NULL DEFAULT false,
    sessions_terminated INTEGER NOT NULL DEFAULT 0,
    credentials_revoked INTEGER NOT NULL DEFAULT 0,

    -- Status tracking
    containment_status VARCHAR(50), -- contained, containing, not_contained
    recovery_status VARCHAR(50), -- recovered, recovering, not_recovered

    -- Raw detection data
    raw_indicators JSONB,
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for ransomware_events
CREATE INDEX idx_ransomware_tenant ON ransomware_events(tenant_id);
CREATE INDEX idx_ransomware_detection ON ransomware_events(detection_id);
CREATE INDEX idx_ransomware_created_at ON ransomware_events(created_at DESC);

-- ============================================================================
-- SSH Key Analytics Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS ssh_key_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    ssh_key_id UUID NOT NULL,

    -- Time period (daily aggregation)
    date DATE NOT NULL,

    -- Usage metrics
    usage_count INTEGER NOT NULL DEFAULT 0,
    unique_users INTEGER NOT NULL DEFAULT 0,
    unique_targets INTEGER NOT NULL DEFAULT 0,

    -- Time tracking
    first_use_time TIMESTAMPTZ,
    last_use_time TIMESTAMPTZ,
    avg_session_duration_seconds NUMERIC(10,2),

    -- Security metrics
    off_hours_usage INTEGER NOT NULL DEFAULT 0,
    unusual_source_usage INTEGER NOT NULL DEFAULT 0,
    failed_attempts INTEGER NOT NULL DEFAULT 0,

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, ssh_key_id, date)
);

-- Indexes for ssh_key_analytics
CREATE INDEX idx_ssh_key_tenant ON ssh_key_analytics(tenant_id);
CREATE INDEX idx_ssh_key_key_id ON ssh_key_analytics(ssh_key_id);
CREATE INDEX idx_ssh_key_date ON ssh_key_analytics(date DESC);
CREATE INDEX idx_ssh_key_metadata ON ssh_key_analytics USING GIN (metadata);

-- ============================================================================
-- Command Blacklist Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS command_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- NULL for global rules (apply to all tenants)

    -- Pattern matching
    command_pattern TEXT NOT NULL,
    pattern_type VARCHAR(20) NOT NULL, -- exact, regex, glob
    base_command VARCHAR(100),

    -- Action to take
    action VARCHAR(20) NOT NULL, -- block, warn, allow, audit
    severity VARCHAR(50) NOT NULL, -- low, medium, high, critical

    -- Scope (who/what this applies to)
    applies_to_users UUID[],
    applies_to_groups UUID[],
    applies_to_targets TEXT[],

    -- Override permissions
    allow_override BOOLEAN NOT NULL DEFAULT false,
    override_roles TEXT[],

    -- Metadata
    reason TEXT NOT NULL,
    risk_category VARCHAR(50), -- privilege_escalation, data_destruction, data_exfiltration, reconnaissance, etc.

    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT blacklist_pattern_type_check CHECK (pattern_type IN ('exact', 'regex', 'glob')),
    CONSTRAINT blacklist_action_check CHECK (action IN ('block', 'warn', 'allow', 'audit')),
    CONSTRAINT blacklist_severity_check CHECK (severity IN ('low', 'medium', 'high', 'critical'))
);

-- Indexes for command_blacklist
CREATE INDEX idx_blacklist_tenant ON command_blacklist(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_blacklist_global ON command_blacklist(id) WHERE tenant_id IS NULL;
CREATE INDEX idx_blacklist_base_command ON command_blacklist(base_command) WHERE enabled = true;
CREATE INDEX idx_blacklist_enabled ON command_blacklist(tenant_id, enabled) WHERE enabled = true;

-- Trigram index for fast pattern matching
CREATE INDEX idx_blacklist_pattern_trgm ON command_blacklist USING GIN (command_pattern gin_trgm_ops) WHERE enabled = true;

-- ============================================================================
-- Materialized Views for Dashboard Performance
-- ============================================================================

-- Session Summary Materialized View
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_session_summary AS
SELECT
    tenant_id,
    date,
    SUM(total_sessions) as total_sessions,
    SUM(active_sessions) as active_sessions,
    SUM(completed_sessions) as completed_sessions,
    SUM(terminated_sessions) as terminated_sessions,
    SUM(failed_sessions) as failed_sessions,
    SUM(ssh_sessions) as ssh_sessions,
    SUM(rdp_sessions) as rdp_sessions,
    SUM(database_sessions) as database_sessions,
    SUM(kubernetes_sessions) as kubernetes_sessions,
    SUM(web_sessions) as web_sessions,
    COALESCE(AVG(avg_duration_seconds), 0) as avg_duration_seconds,
    MAX(peak_concurrent_sessions) as peak_concurrent_sessions,
    SUM(unique_users) as unique_users,
    SUM(unique_targets) as unique_targets
FROM session_analytics
GROUP BY tenant_id, date;

-- Index for materialized view
CREATE UNIQUE INDEX idx_mv_session_summary_tenant_date ON mv_session_summary(tenant_id, date);
CREATE INDEX idx_mv_session_summary_date ON mv_session_summary(date DESC);

-- User Risk Summary Materialized View
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_user_risk_summary AS
SELECT
    tenant_id,
    user_id,
    date,
    SUM(sessions_initiated) as sessions_initiated,
    SUM(commands_executed) as commands_executed,
    COALESCE(AVG(risk_score), 0) as avg_risk_score,
    SUM(CASE WHEN off_hours_access THEN 1 ELSE 0 END) as off_hours_count,
    SUM(CASE WHEN unusual_access THEN 1 ELSE 0 END) as unusual_count,
    SUM(total_session_seconds) as total_session_seconds
FROM user_activity
GROUP BY tenant_id, user_id, date;

-- Index for user risk materialized view
CREATE UNIQUE INDEX idx_mv_user_risk_summary ON mv_user_risk_summary(tenant_id, user_id, date);
CREATE INDEX idx_mv_user_risk_score ON mv_user_risk_summary(avg_risk_score DESC);

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

-- Helper function for updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to tables with updated_at
DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;
CREATE TRIGGER update_compliance_exceptions_updated_at
    BEFORE UPDATE ON compliance_exceptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_anomaly_detections_updated_at ON anomaly_detections;
CREATE TRIGGER update_anomaly_detections_updated_at
    BEFORE UPDATE ON anomaly_detections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_ransomware_events_updated_at ON ransomware_events;
CREATE TRIGGER update_ransomware_events_updated_at
    BEFORE UPDATE ON ransomware_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_ssh_key_analytics_updated_at ON ssh_key_analytics;
CREATE TRIGGER update_ssh_key_analytics_updated_at
    BEFORE UPDATE ON ssh_key_analytics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_command_blacklist_updated_at ON command_blacklist;
CREATE TRIGGER update_command_blacklist_updated_at
    BEFORE UPDATE ON command_blacklist
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Row Level Security (RLS) for multi-tenancy
-- ============================================================================

-- Enable RLS on all tenant-scoped tables
ALTER TABLE compliance_reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_control_evaluations ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_exceptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_detections ENABLE ROW LEVEL SECURITY;
ALTER TABLE ransomware_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE ssh_key_analytics ENABLE ROW LEVEL SECURITY;
ALTER TABLE command_blacklist ENABLE ROW LEVEL SECURITY;

-- Policies for compliance_reports
CREATE POLICY compliance_reports_tenant_policy ON compliance_reports
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for compliance_control_evaluations
CREATE POLICY compliance_eval_tenant_policy ON compliance_control_evaluations
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for compliance_exceptions
CREATE POLICY compliance_exc_tenant_policy ON compliance_exceptions
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for anomaly_detections
CREATE POLICY anomaly_tenant_policy ON anomaly_detections
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for ransomware_events
CREATE POLICY ransomware_tenant_policy ON ransomware_events
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for ssh_key_analytics
CREATE POLICY ssh_key_tenant_policy ON ssh_key_analytics
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for command_blacklist (tenant-specific or global NULL)
CREATE POLICY blacklist_tenant_policy ON command_blacklist
    USING (tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)::UUID);

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE compliance_reports IS 'Stores compliance framework evaluation reports (SOC2, ISO27001, PCI-DSS, etc.)';
COMMENT ON TABLE compliance_control_evaluations IS 'Individual control evaluations within a compliance report';
COMMENT ON TABLE compliance_exceptions IS 'Approved exceptions to compliance controls with justification and expiration';
COMMENT ON TABLE anomaly_detections IS 'Detected security anomalies and behavioral alerts';
COMMENT ON TABLE ransomware_events IS 'High-priority ransomware detection events with emergency response tracking';
COMMENT ON TABLE ssh_key_analytics IS 'Daily aggregated SSH key usage analytics for security monitoring';
COMMENT ON TABLE command_blacklist IS 'Dangerous command patterns that can be blocked or audited';
COMMENT ON MATERIALIZED VIEW mv_session_summary IS 'Aggregated session metrics by tenant and date';
COMMENT ON MATERIALIZED VIEW mv_user_risk_summary IS 'Aggregated user risk metrics by tenant, user, and date';
