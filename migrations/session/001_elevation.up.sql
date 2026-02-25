-- Session Service Elevation Database Schema
-- Migration 001: Privileged elevation management

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Elevation Requests: Privileged elevation tracking
CREATE TABLE IF NOT EXISTS elevation_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    session_id UUID NOT NULL,
    target_host VARCHAR(255) NOT NULL,
    target_user VARCHAR(255), -- Target user for elevation (e.g., root, postgres)
    requested_command TEXT, -- Specific command being requested
    justification TEXT NOT NULL,
    duration INTEGER NOT NULL, -- seconds
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, expired, denied, revoked

    -- Approval tracking
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    approval_method VARCHAR(50), -- mfa, approval, auto

    -- Elevation tracking
    elevated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    -- Usage tracking
    commands_executed INTEGER NOT NULL DEFAULT 0,
    last_activity_at TIMESTAMPTZ,

    -- Session recording
    session_recording_id UUID,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT elevation_requests_status CHECK (
        status IN ('pending', 'active', 'expired', 'denied', 'revoked')
    )
);

-- Indexes for elevation_requests
CREATE INDEX idx_elevation_requests_tenant ON elevation_requests(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_elevation_requests_user ON elevation_requests(user_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_elevation_requests_session ON elevation_requests(session_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_elevation_requests_status ON elevation_requests(status, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_elevation_requests_active ON elevation_requests(tenant_id, status) WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX idx_elevation_requests_expires_at ON elevation_requests(expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- Elevation Audit: Detailed audit log for elevation events
CREATE TABLE IF NOT EXISTS elevation_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    elevation_id UUID NOT NULL,
    user_id UUID NOT NULL,
    session_id UUID NOT NULL,

    -- Event details
    event_type VARCHAR(50) NOT NULL, -- requested, approved, denied, active, expired, revoked, command_executed
    event_data JSONB DEFAULT '{}',

    -- Context
    target_host VARCHAR(255),
    command TEXT,
    exit_code INTEGER,

    -- Timestamp
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for elevation_audit
CREATE INDEX idx_elevation_audit_tenant ON elevation_audit(tenant_id, occurred_at DESC);
CREATE INDEX idx_elevation_audit_elevation ON elevation_audit(elevation_id, occurred_at DESC);
CREATE INDEX idx_elevation_audit_user ON elevation_audit(user_id, occurred_at DESC);
CREATE INDEX idx_elevation_audit_event_type ON elevation_audit(event_type, occurred_at DESC);

-- Function to check for active elevations
CREATE OR REPLACE FUNCTION get_active_elevations(p_tenant_id UUID, p_user_id UUID)
RETURNS TABLE (
    id UUID,
    session_id UUID,
    target_host VARCHAR(255),
    expires_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        er.id,
        er.session_id,
        er.target_host,
        er.expires_at
    FROM elevation_requests er
    WHERE er.tenant_id = p_tenant_id
        AND er.user_id = p_user_id
        AND er.status = 'active'
        AND er.deleted_at IS NULL
        AND (er.expires_at IS NULL OR er.expires_at > NOW());
END;
$$ LANGUAGE plpgsql;

-- Function to expire outdated elevations
CREATE OR REPLACE FUNCTION expire_elevations()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    UPDATE elevation_requests
    SET status = 'expired',
        updated_at = NOW()
    WHERE status = 'active'
        AND expires_at <= NOW()
        AND deleted_at IS NULL;

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;
