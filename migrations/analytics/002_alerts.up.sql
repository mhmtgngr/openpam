-- Analytics Alerts Migration
-- Migration 002: Add alert management tables

-- Alert rules configuration
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- anomaly_detected, session_anomaly, user_anomaly, etc.
    description TEXT,

    -- Rule definition
    conditions JSONB NOT NULL DEFAULT '[]',
    actions JSONB NOT NULL DEFAULT '[]',

    -- Configuration
    enabled BOOLEAN NOT NULL DEFAULT true,
    severity VARCHAR(50) NOT NULL DEFAULT 'medium', -- critical, high, medium, low, info
    throttle_minutes INTEGER NOT NULL DEFAULT 5,

    -- Metadata
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT alert_rules_type CHECK (
        type IN ('anomaly_detected', 'session_anomaly', 'user_anomaly', 'command_anomaly',
                 'access_violation', 'policy_violation', 'threshold_exceeded',
                 'security_event', 'compliance_violation')
    ),
    CONSTRAINT alert_rules_severity CHECK (
        severity IN ('critical', 'high', 'medium', 'low', 'info')
    )
);

CREATE INDEX idx_alert_rules_tenant ON alert_rules(tenant_id);
CREATE INDEX idx_alert_rules_type ON alert_rules(type);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(tenant_id, enabled) WHERE enabled = true;

-- Alerts table
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'open', -- open, acknowledged, resolved, false_positive
    title VARCHAR(500) NOT NULL,
    description TEXT,
    source VARCHAR(255), -- anomaly_detection, policy_engine, user_report, etc.

    -- Context
    user_id UUID,
    session_id UUID,
    target_host VARCHAR(255),
    metadata JSONB DEFAULT '{}',

    -- Scoring
    risk_score FLOAT DEFAULT 0,
    confidence FLOAT DEFAULT 0,

    -- Escalation
    escalation_level INTEGER NOT NULL DEFAULT 1,
    escalated_at TIMESTAMPTZ,

    -- Acknowledgment
    acked_at TIMESTAMPTZ,
    acked_by UUID,

    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,

    -- Rule reference
    alert_rule_id UUID REFERENCES alert_rules(id),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT alerts_status CHECK (
        status IN ('open', 'acknowledged', 'resolved', 'false_positive')
    ),
    CONSTRAINT alerts_severity CHECK (
        severity IN ('critical', 'high', 'medium', 'low', 'info')
    )
);

CREATE INDEX idx_alerts_tenant ON alerts(tenant_id);
CREATE INDEX idx_alerts_type ON alerts(type);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_user ON alerts(user_id);
CREATE INDEX idx_alerts_session ON alerts(session_id);
CREATE INDEX idx_alerts_created ON alerts(created_at DESC);
CREATE INDEX idx_alerts_rule ON alerts(alert_rule_id);
CREATE INDEX idx_alerts_tenant_status ON alerts(tenant_id, status) WHERE status IN ('open', 'acknowledged');

-- Alert notifications log
CREATE TABLE IF NOT EXISTS alert_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES alerts(id),
    channel_id UUID NOT NULL,

    -- Notification details
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, sent, failed
    sent_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,

    -- Response
    delivery_status VARCHAR(255),
    external_id VARCHAR(255),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_notifications_alert ON alert_notifications(alert_id);
CREATE INDEX idx_alert_notifications_status ON alert_notifications(status) WHERE status = 'pending';

-- Notification channels
CREATE TABLE IF NOT EXISTS notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- email, slack, webhook, pagerduty, teams

    -- Channel configuration
    config JSONB NOT NULL DEFAULT '{}',

    -- Filtering
    severity_filter JSONB DEFAULT '[]', -- Which severities to send
    type_filter JSONB DEFAULT '[]',     -- Which alert types to send

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    verified BOOLEAN NOT NULL DEFAULT false,
    last_verified_at TIMESTAMPTZ,

    -- Metadata
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_channels_type CHECK (
        type IN ('email', 'slack', 'webhook', 'pagerduty', 'teams', 'sms')
    )
);

CREATE INDEX idx_notification_channels_tenant ON notification_channels(tenant_id);
CREATE INDEX idx_notification_channels_enabled ON notification_channels(tenant_id, enabled) WHERE enabled = true;

-- Alert acknowledgments log
CREATE TABLE IF NOT EXISTS alert_acknowledgments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_id UUID NOT NULL REFERENCES alerts(id),
    user_id UUID NOT NULL,
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT,

    CONSTRAINT alert_acks_unique UNIQUE (alert_id, user_id)
);

CREATE INDEX idx_alert_acks_alert ON alert_acknowledgments(alert_id);
CREATE INDEX idx_alert_acks_user ON alert_acknowledgments(user_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_alerts_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER update_alert_rules_updated_at_trigger
    BEFORE UPDATE ON alert_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_alerts_updated_at();

CREATE TRIGGER update_alerts_updated_at_trigger
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_alerts_updated_at();

CREATE TRIGGER update_notification_channels_updated_at_trigger
    BEFORE UPDATE ON notification_channels
    FOR EACH ROW
    EXECUTE FUNCTION update_alerts_updated_at();

-- Function to get active alerts for escalation
CREATE OR REPLACE FUNCTION get_active_alerts_for_escalation()
RETURNS TABLE (
    alert_id UUID,
    tenant_id UUID,
    type VARCHAR,
    severity VARCHAR,
    escalation_level INTEGER,
    created_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        a.id,
        a.tenant_id,
        a.type,
        a.severity,
        a.escalation_level,
        a.created_at
    FROM alerts a
    WHERE a.status = 'open'
        AND a.created_at < NOW() - INTERVAL '1 hour'
        AND (a.escalated_at IS NULL OR a.escalated_at < NOW() - INTERVAL '4 hours')
    ORDER BY a.severity DESC, a.created_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to get alert statistics
CREATE OR REPLACE FUNCTION get_alert_statistics(p_tenant_id UUID)
RETURNS TABLE (
    total BIGINT,
    open BIGINT,
    critical BIGINT,
    high BIGINT,
    medium BIGINT,
    low BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT,
        COUNT(*) FILTER (WHERE status = 'open')::BIGINT,
        COUNT(*) FILTER (WHERE severity = 'critical')::BIGINT,
        COUNT(*) FILTER (WHERE severity = 'high')::BIGINT,
        COUNT(*) FILTER (WHERE severity = 'medium')::BIGINT,
        COUNT(*) FILTER (WHERE severity = 'low')::BIGINT
    FROM alerts
    WHERE tenant_id = p_tenant_id
        AND created_at > NOW() - INTERVAL '24 hours';
END;
$$ LANGUAGE plpgsql;
