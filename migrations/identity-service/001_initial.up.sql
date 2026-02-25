-- Identity Service Database Schema
-- Migration 001: Initial schema for access requests, workflows, and break-glass access

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Access Requests: Self-service access request management
CREATE TABLE IF NOT EXISTS access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    requester_id UUID NOT NULL,
    target_id UUID NOT NULL,
    target_type VARCHAR(50) NOT NULL, -- server, database, kubernetes, application
    access_type VARCHAR(50) NOT NULL, -- ssh, rdp, database, kubernetes, web
    justification TEXT NOT NULL,
    duration INTEGER NOT NULL, -- minutes

    -- Timing
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending_approval', -- pending_approval, approved, active, completed, denied, cancelled, expired
    workflow_id UUID,
    current_step INTEGER NOT NULL DEFAULT 1,
    approval_chain JSONB DEFAULT '[]',

    -- ITSM Integration
    ticket_id VARCHAR(255),
    ticket_url TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT access_requests_status CHECK (
        status IN ('pending_approval', 'approved', 'active', 'completed', 'denied', 'cancelled', 'expired')
    )
);

-- Indexes for access_requests
CREATE INDEX idx_access_requests_tenant ON access_requests(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_requester ON access_requests(requester_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_target ON access_requests(target_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_status ON access_requests(status, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_type ON access_requests(access_type, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_requested_at ON access_requests(requested_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_access_requests_expires_at ON access_requests(end_at) WHERE deleted_at IS NULL AND end_at IS NOT NULL;

-- Workflows: Approval workflow configurations
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    target_type VARCHAR(50) NOT NULL,
    access_type VARCHAR(50) NOT NULL,

    -- Workflow definition
    steps JSONB NOT NULL DEFAULT '[]',

    -- Configuration
    is_active BOOLEAN NOT NULL DEFAULT true,
    require_mfa BOOLEAN NOT NULL DEFAULT false,
    max_duration INTEGER NOT NULL DEFAULT 480, -- minutes (8 hours default)
    auto_approve_days INTEGER NOT NULL DEFAULT 30,
    escalation_policy JSONB DEFAULT '{}',

    -- Metadata
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT workflows_unique UNIQUE (tenant_id, name, target_type, access_type, deleted_at)
);

-- Indexes for workflows
CREATE INDEX idx_workflows_tenant ON workflows(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_workflows_active ON workflows(tenant_id, is_active, deleted_at) WHERE deleted_at IS NULL AND is_active = true;

-- Break Glass Requests: Emergency access requests
CREATE TABLE IF NOT EXISTS break_glass_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    requester_id UUID NOT NULL,
    target_id UUID NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    reason TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL, -- low, medium, high, critical

    -- ITSM integration
    ticket_id VARCHAR(255),
    incident_id VARCHAR(255),

    -- Timing
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    extended_until TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, expired, revoked
    approval_method VARCHAR(50) NOT NULL, -- mfa, dual_auth, ticket
    approver_id UUID,

    -- Session tracking
    session_id UUID,
    commands_executed INTEGER NOT NULL DEFAULT 0,
    commands_blocked INTEGER NOT NULL DEFAULT 0,

    -- Audit
    audit_log_id UUID,
    justification_doc TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT break_glass_status CHECK (
        status IN ('pending', 'active', 'expired', 'revoked')
    ),
    CONSTRAINT break_glass_severity CHECK (
        severity IN ('low', 'medium', 'high', 'critical')
    )
);

-- Indexes for break_glass_requests
CREATE INDEX idx_break_glass_tenant ON break_glass_requests(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_break_glass_requester ON break_glass_requests(requester_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_break_glass_status ON break_glass_requests(status, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_break_glass_severity ON break_glass_requests(severity, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_break_glass_active ON break_glass_requests(tenant_id, status) WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX idx_break_glass_expires_at ON break_glass_requests(expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_identity_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER update_access_requests_updated_at
    BEFORE UPDATE ON access_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_identity_updated_at_column();

CREATE TRIGGER update_workflows_updated_at
    BEFORE UPDATE ON workflows
    FOR EACH ROW
    EXECUTE FUNCTION update_identity_updated_at_column();

CREATE TRIGGER update_break_glass_requests_updated_at
    BEFORE UPDATE ON break_glass_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_identity_updated_at_column();

-- Function to check for expiring access requests
CREATE OR REPLACE FUNCTION get_expiring_access_requests(p_tenant_id UUID, p_within_minutes INTEGER)
RETURNS TABLE (
    id UUID,
    tenant_id UUID,
    requester_id UUID,
    target_id UUID,
    access_type VARCHAR(50),
    expires_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ar.id,
        ar.tenant_id,
        ar.requester_id,
        ar.target_id,
        ar.access_type,
        ar.end_time AS expires_at
    FROM access_requests ar
    WHERE ar.tenant_id = p_tenant_id
        AND ar.status IN ('approved', 'active')
        AND ar.end_time <= NOW() + (p_within_minutes || ' minutes')::INTERVAL
        AND ar.end_time > NOW()
        AND ar.deleted_at IS NULL
    ORDER BY ar.end_time ASC;
END;
$$ LANGUAGE plpgsql;
