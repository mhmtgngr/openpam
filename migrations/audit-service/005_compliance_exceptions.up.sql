-- Compliance Exceptions Migration for Audit Service
-- This migration creates the compliance_exceptions table for tracking
-- exceptions to compliance controls with approval workflow.

-- ============================================================================
-- Compliance Exceptions Table
-- ============================================================================
-- This table stores exceptions to compliance controls, allowing for
-- temporary waivers of controls with proper approval workflow and
-- risk acceptance documentation.

CREATE TABLE IF NOT EXISTS compliance_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Control identification
    control_id VARCHAR(100) NOT NULL,
    control_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Exception status
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending: awaiting approval
    -- approved: exception approved and active
    -- denied: exception request denied
    -- expired: exception period expired
    -- revoked: exception revoked before expiration

    risk_level VARCHAR(50) NOT NULL, -- low, medium, high, critical

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
    review_date VARCHAR(50), -- ISO date string for recurring review
    review_notes TEXT,

    -- Notifications
    last_notification_at TIMESTAMPTZ,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT compliance_exceptions_status_check CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'revoked')),
    CONSTRAINT compliance_exceptions_risk_level_check CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

-- Indexes for compliance_exceptions
CREATE INDEX idx_compliance_exceptions_tenant ON compliance_exceptions(tenant_id);
CREATE INDEX idx_compliance_exceptions_control ON compliance_exceptions(control_id, framework);
CREATE INDEX idx_compliance_exceptions_framework ON compliance_exceptions(framework);
CREATE INDEX idx_compliance_exceptions_status ON compliance_exceptions(status);
CREATE INDEX idx_compliance_exceptions_risk_level ON compliance_exceptions(risk_level);
CREATE INDEX idx_compliance_exceptions_expires_at ON compliance_exceptions(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_compliance_exceptions_requested_at ON compliance_exceptions(requested_at DESC);
CREATE INDEX idx_compliance_exceptions_requested_by ON compliance_exceptions(requested_by);
CREATE INDEX idx_compliance_exceptions_approved_by ON compliance_exceptions(approved_by) WHERE approved_by IS NOT NULL;
CREATE INDEX idx_compliance_exceptions_metadata ON compliance_exceptions USING GIN (metadata);

-- Partial index for active exceptions (common query for active exceptions)
CREATE INDEX idx_compliance_exceptions_active_tenant
    ON compliance_exceptions(tenant_id, framework)
    WHERE status = 'approved' AND (expires_at IS NULL OR expires_at > NOW());

-- Partial index for pending approvals
CREATE INDEX idx_compliance_exceptions_pending
    ON compliance_exceptions(tenant_id, requested_at ASC)
    WHERE status = 'pending';

-- Partial index for expiring soon (within 7 days)
CREATE INDEX idx_compliance_exceptions_expiring_soon
    ON compliance_exceptions(tenant_id, expires_at)
    WHERE status = 'approved'
      AND expires_at <= NOW() + INTERVAL '7 days'
      AND expires_at > NOW();

-- ============================================================================
-- Row Level Security (RLS) for multi-tenancy
-- ============================================================================

ALTER TABLE compliance_exceptions ENABLE ROW LEVEL SECURITY;

-- Policy for compliance_exceptions
CREATE POLICY compliance_exceptions_tenant_policy ON compliance_exceptions
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- ============================================================================
-- Functions for Compliance Exception Management
-- ============================================================================

-- Function to update exception status
CREATE OR REPLACE FUNCTION update_exception_status(
    p_exception_id UUID,
    p_status VARCHAR,
    p_approved_by UUID DEFAULT NULL,
    p_expires_at TIMESTAMPTZ DEFAULT NULL
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE compliance_exceptions
    SET
        status = p_status,
        approved_by = COALESCE(p_approved_by, approved_by),
        approved_at = CASE WHEN p_approved_by IS NOT NULL THEN NOW() ELSE approved_at END,
        expires_at = COALESCE(p_expires_at, expires_at),
        updated_at = NOW()
    WHERE id = p_exception_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to get exception statistics
CREATE OR REPLACE FUNCTION get_exception_stats(p_tenant_id UUID)
RETURNS TABLE (
    pending_count INTEGER,
    approved_count INTEGER,
    denied_count INTEGER,
    expired_count INTEGER,
    revoked_count INTEGER,
    total_count INTEGER,
    critical_count INTEGER,
    high_count INTEGER,
    expiring_soon_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
        COUNT(*) FILTER (WHERE status = 'approved') as approved_count,
        COUNT(*) FILTER (WHERE status = 'denied') as denied_count,
        COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
        COUNT(*) FILTER (WHERE status = 'revoked') as revoked_count,
        COUNT(*) as total_count,
        COUNT(*) FILTER (WHERE risk_level = 'critical') as critical_count,
        COUNT(*) FILTER (WHERE risk_level = 'high') as high_count,
        COUNT(*) FILTER (WHERE status = 'approved'
                          AND expires_at <= NOW() + INTERVAL '7 days'
                          AND expires_at > NOW()) as expiring_soon_count
    FROM compliance_exceptions
    WHERE tenant_id = p_tenant_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get exceptions for a specific control
CREATE OR REPLACE FUNCTION get_control_exceptions(
    p_tenant_id UUID,
    p_control_id VARCHAR,
    p_framework VARCHAR
) RETURNS TABLE (
    id UUID,
    status VARCHAR,
    risk_level VARCHAR,
    expires_at TIMESTAMPTZ,
    justification TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ce.id,
        ce.status,
        ce.risk_level,
        ce.expires_at,
        ce.justification
    FROM compliance_exceptions ce
    WHERE ce.tenant_id = p_tenant_id
      AND ce.control_id = p_control_id
      AND ce.framework = p_framework
      AND (ce.expires_at IS NULL OR ce.expires_at > NOW())
    ORDER BY ce.requested_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to get expiring exceptions for notification
CREATE OR REPLACE FUNCTION get_expiring_exceptions(p_days INTEGER DEFAULT 7)
RETURNS TABLE (
    exception_id UUID,
    tenant_id UUID,
    control_id VARCHAR,
    control_name VARCHAR,
    framework VARCHAR,
    expires_at TIMESTAMPTZ,
    risk_level VARCHAR,
    last_notification_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ce.id as exception_id,
        ce.tenant_id,
        ce.control_id,
        ce.control_name,
        ce.framework,
        ce.expires_at,
        ce.risk_level,
        ce.last_notification_at
    FROM compliance_exceptions ce
    WHERE ce.status = 'approved'
      AND ce.expires_at <= NOW() + INTERVAL '1 day' * p_days
      AND ce.expires_at > NOW()
      AND (ce.last_notification_at IS NULL
           OR ce.last_notification_at < NOW() - INTERVAL '1 day')
    ORDER BY ce.expires_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to update notification timestamp
CREATE OR REPLACE FUNCTION update_exception_notification(p_exception_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE compliance_exceptions
    SET last_notification_at = NOW(),
        updated_at = NOW()
    WHERE id = p_exception_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to expire old exceptions (run periodically)
CREATE OR REPLACE FUNCTION expire_old_exceptions() RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    UPDATE compliance_exceptions
    SET status = 'expired',
        updated_at = NOW()
    WHERE status = 'approved'
      AND expires_at IS NOT NULL
      AND expires_at < NOW();

    GET DIAGNOSTICS expired_count = ROW_COUNT;
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;
CREATE TRIGGER update_compliance_exceptions_updated_at
    BEFORE UPDATE ON compliance_exceptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE compliance_exceptions IS 'Stores exceptions to compliance controls with approval workflow and risk acceptance';
COMMENT ON COLUMN compliance_exceptions.status IS 'Exception status: pending, approved, denied, expired, revoked';
COMMENT ON COLUMN compliance_exceptions.risk_level IS 'Risk level: low, medium, high, critical';
COMMENT ON COLUMN compliance_exceptions.compensating_controls IS 'List of compensating controls in place while exception is active';
COMMENT ON COLUMN compliance_exceptions.expires_at IS 'When the exception expires (can be NULL for permanent exceptions)';
COMMENT ON COLUMN compliance_exceptions.review_date IS 'ISO date string for recurring review (e.g., 2024-01-15 for annual review)';
COMMENT ON COLUMN compliance_exceptions.last_notification_at IS 'Last time notification was sent about this exception';
