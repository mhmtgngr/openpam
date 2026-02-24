-- 002_policies.up.sql
-- Policy Engine Migration
-- Creates tables for access policies, rules, and approval workflows

-- Policy types enumeration
CREATE TYPE policy_type AS ENUM (
    'access',           -- General access policy
    'session',          -- Session-specific policy
    'credential',       -- Credential checkout policy
    'approval',         -- Approval workflow policy
    'compliance',       -- Compliance/audit policy
    'command_filter'    -- Command filtering policy
);

-- Policy effects enumeration
CREATE TYPE policy_effect AS ENUM ('allow', 'deny');

-- Rule types enumeration
CREATE TYPE rule_type AS ENUM (
    'time_restriction',     -- Time-based access rules
    'ip_restriction',       -- IP/network-based rules
    'mfa_requirement',      -- MFA enforcement rules
    'command_filter',       -- Command filtering rules
    'session_recording',    -- Session recording rules
    'approval_requirement', -- Approval workflow rules
    'duration_limit',       -- Maximum session duration
    'concurrent_limit'      -- Concurrent session limits
);

-- Approval status enumeration
CREATE TYPE approval_status AS ENUM (
    'pending',
    'approved',
    'denied',
    'cancelled',
    'expired'
);

-- Main policies table
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type policy_type NOT NULL,
    effect policy_effect NOT NULL DEFAULT 'allow',
    priority INTEGER NOT NULL DEFAULT 100,

    -- Policy scope
    tenant_id UUID NOT NULL,
    user_ids JSONB DEFAULT '[]',           -- Specific users this applies to
    role_ids JSONB DEFAULT '[]',           -- Specific roles this applies to
    group_ids JSONB DEFAULT '[]',          -- Specific groups this applies to
    target_ids JSONB DEFAULT '[]',         -- Specific targets this applies to
    credential_ids JSONB DEFAULT '[]',     -- Specific credentials this applies to

    -- Policy rules (flexible JSONB for extensibility)
    rules JSONB NOT NULL DEFAULT '{}',

    -- Duration settings
    max_session_duration_seconds INTEGER,
    session_extension_allowed BOOLEAN DEFAULT false,
    max_extensions INTEGER DEFAULT 0,

    -- MFA settings
    mfa_required BOOLEAN DEFAULT false,
    mfa_methods JSONB DEFAULT '[]',        -- totp, push, sms, hardware_key

    -- Approval settings
    approval_required BOOLEAN DEFAULT false,
    approval_approvers JSONB DEFAULT '[]', -- User or role IDs that can approve
    approval_timeout_minutes INTEGER DEFAULT 60,

    -- Session recording
    recording_required BOOLEAN DEFAULT false,
    recording_mode VARCHAR(50) DEFAULT 'all', -- all, on_command, none

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    system_policy BOOLEAN DEFAULT false,     -- System policies cannot be deleted

    -- Metadata
    metadata JSONB DEFAULT '{}',
    tags JSONB DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,

    -- Audit
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT valid_duration CHECK (
        max_session_duration_seconds IS NULL OR max_session_duration_seconds > 0
    ),
    CONSTRAINT positive_priority CHECK (priority >= 0),
    CONSTRAINT positive_timeout CHECK (
        approval_timeout_minutes IS NULL OR approval_timeout_minutes > 0
    )
);

-- Indexes for policies
CREATE INDEX idx_policies_tenant ON policies(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_type ON policies(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_enabled ON policies(enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_priority ON policies(priority) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_user_ids ON policies USING GIN(user_ids) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_role_ids ON policies USING GIN(role_ids) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_target_ids ON policies USING GIN(target_ids) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_credential_ids ON policies USING GIN(credential_ids) WHERE deleted_at IS NULL;
CREATE INDEX idx_policies_tags ON policies USING GIN(tags) WHERE deleted_at IS NULL;

-- Policy assignments (explicit user-policy mappings for override cases)
CREATE TABLE IF NOT EXISTS policy_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,

    -- Assignment-specific overrides
    overrides JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,

    CONSTRAINT unique_policy_user UNIQUE (policy_id, user_id)
);

CREATE INDEX idx_policy_assignments_policy ON policy_assignments(policy_id);
CREATE INDEX idx_policy_assignments_user ON policy_assignments(user_id);
CREATE INDEX idx_policy_assignments_tenant ON policy_assignments(tenant_id);

-- Approval requests table
CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID REFERENCES policies(id) ON DELETE SET NULL,
    requester_id UUID NOT NULL,
    tenant_id UUID NOT NULL,

    -- Request details
    target_type VARCHAR(50) NOT NULL,      -- session, credential, target
    target_id UUID NOT NULL,
    reason TEXT NOT NULL,

    -- Timing
    requested_duration_seconds INTEGER,
    requested_start_time TIMESTAMPTZ,
    requested_end_time TIMESTAMPTZ,

    -- Status
    status approval_status NOT NULL DEFAULT 'pending',

    -- Approval
    approver_id UUID,
    approved_at TIMESTAMPTZ,
    denial_reason TEXT,

    -- Expiry
    expires_at TIMESTAMPTZ NOT NULL,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_approval_window CHECK (
        requested_end_time IS NULL OR requested_start_time IS NULL OR
        requested_end_time > requested_start_time
    )
);

CREATE INDEX idx_approval_requests_policy ON approval_requests(policy_id);
CREATE INDEX idx_approval_requests_requester ON approval_requests(requester_id);
CREATE INDEX idx_approval_requests_tenant ON approval_requests(tenant_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_approval_requests_target ON approval_requests(target_type, target_id);
CREATE INDEX idx_approval_requests_expires_at ON approval_requests(expires_at) WHERE status = 'pending';

-- Policy evaluation log (for audit and compliance)
CREATE TABLE IF NOT EXISTS policy_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    policy_id UUID REFERENCES policies(id),

    -- Request context
    user_id UUID NOT NULL,
    target_type VARCHAR(50),               -- session, credential, target
    target_id UUID,
    action VARCHAR(100) NOT NULL,          -- checkout, extend, terminate, etc.

    -- Evaluation result
    result VARCHAR(50) NOT NULL,           -- allow, deny, approval_required
    denial_reasons JSONB DEFAULT '[]',
    matched_rules JSONB DEFAULT '[]',

    -- Context at evaluation time
    client_ip VARCHAR(45),
    user_agent TEXT,
    session_id UUID,

    -- Performance tracking
    evaluation_duration_ms INTEGER,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_evaluations_tenant ON policy_evaluations(tenant_id);
CREATE INDEX idx_policy_evaluations_policy ON policy_evaluations(policy_id);
CREATE INDEX idx_policy_evaluations_user ON policy_evaluations(user_id);
CREATE INDEX idx_policy_evaluations_result ON policy_evaluations(result);
CREATE INDEX idx_policy_evaluations_created_at ON policy_evaluations(created_at);

-- Partition policy_evaluations by month for performance
-- (PostgreSQL 16+ declarative partitioning)
-- CREATE TABLE policy_evaluations_y2024m01 PARTITION OF policy_evaluations
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Policy templates for common use cases
CREATE TABLE IF NOT EXISTS policy_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(100) NOT NULL,

    -- Template definition
    template_policy JSONB NOT NULL,        -- Policy structure with placeholders

    -- Template metadata
    is_system BOOLEAN DEFAULT false,
    tags JSONB DEFAULT '[]',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policy_templates_category ON policy_templates(category);
CREATE INDEX idx_policy_templates_tags ON policy_templates USING GIN(tags);

-- Command filter patterns (for RE2 regex caching)
CREATE TABLE IF NOT EXISTS command_filter_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,

    pattern VARCHAR(1000) NOT NULL,
    is_whitelist BOOLEAN NOT NULL DEFAULT false,  -- true=allow, false=deny
    description TEXT,

    -- Caching
    compiled_hash VARCHAR(64) NOT NULL,          -- SHA-256 of pattern for cache key

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_policy_pattern UNIQUE (policy_id, pattern)
);

CREATE INDEX idx_cmd_filter_policy ON command_filter_patterns(policy_id);
CREATE INDEX idx_cmd_filter_hash ON command_filter_patterns(compiled_hash);

-- Insert system policies for default behavior
INSERT INTO policies (
    name,
    description,
    type,
    effect,
    priority,
    tenant_id,
    enabled,
    system_policy,
    rules
) VALUES
(
    'Default Allow Policy',
    'Default policy allowing access with appropriate permissions',
    'access',
    'allow',
    0,
    '00000000-0000-0000-0000-000000000000'::uuid,  -- System tenant
    true,
    true,
    '{}'::jsonb
),
(
    'System Compliance Policy',
    'System-wide compliance and audit requirements',
    'compliance',
    'deny',
    1000,
    '00000000-0000-0000-0000-000000000000'::uuid,
    true,
    true,
    '{"require_mfa": true, "require_recording": true}'::jsonb
)
ON CONFLICT DO NOTHING;

-- Insert common policy templates
INSERT INTO policy_templates (name, description, category, template_policy, is_system, tags) VALUES
(
    'Business Hours Only',
    'Restricts access to business hours (9 AM - 5 PM, Monday-Friday)',
    'time',
    '{
        "name": "Business Hours - {{name}}",
        "description": "{{description}}",
        "type": "access",
        "effect": "deny",
        "rules": {
            "time_restrictions": [
                {
                    "days": ["saturday", "sunday"],
                    "action": "deny"
                },
                {
                    "start_time": "00:00",
                    "end_time": "09:00",
                    "days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
                    "action": "deny"
                },
                {
                    "start_time": "17:00",
                    "end_time": "23:59",
                    "days": ["monday", "tuesday", "wednesday", "thursday", "friday"],
                    "action": "deny"
                }
            ]
        }
    }'::jsonb,
    true,
    '["time", "business-hours"]'::jsonb
),
(
    'MFA Required',
    'Requires multi-factor authentication for access',
    'security',
    '{
        "name": "MFA Required - {{name}}",
        "description": "{{description}}",
        "type": "access",
        "effect": "deny",
        "mfa_required": true,
        "mfa_methods": ["totp", "push", "hardware_key"],
        "rules": {
            "mfa_requirement": {
                "required": true,
                "methods": ["totp", "push", "hardware_key"]
            }
        }
    }'::jsonb,
    true,
    '["mfa", "security"]'::jsonb
),
(
    'Command Whitelist',
    'Allows only specific commands during SSH sessions',
    'security',
    '{
        "name": "Command Whitelist - {{name}}",
        "description": "{{description}}",
        "type": "command_filter",
        "effect": "deny",
        "rules": {
            "command_filters": [
                {
                    "pattern": "{{pattern}}",
                    "is_whitelist": true,
                    "description": "{{command_description}}"
                }
            ]
        }
    }'::jsonb,
    true,
    '["command", "whitelist", "ssh"]'::jsonb
),
(
    'Approval Required for Production',
    'Requires manager approval for production access',
    'approval',
    '{
        "name": "Approval Required - {{name}}",
        "description": "{{description}}",
        "type": "approval",
        "effect": "deny",
        "approval_required": true,
        "approval_approvers": "{{approver_roles}}",
        "approval_timeout_minutes": 60,
        "rules": {
            "approval_requirement": {
                "required": true,
                "approvers": "{{approver_roles}}",
                "timeout_minutes": 60
            }
        }
    }'::jsonb,
    true,
    '["approval", "production"]'::jsonb
),
(
    'Session Recording Required',
    'Mandates session recording for compliance',
    'compliance',
    '{
        "name": "Session Recording - {{name}}",
        "description": "{{description}}",
        "type": "session",
        "effect": "allow",
        "recording_required": true,
        "recording_mode": "all",
        "rules": {
            "session_recording": {
                "required": true,
                "mode": "all"
            }
        }
    }'::jsonb,
    true,
    '["recording", "compliance"]'::jsonb
)
ON CONFLICT (name) DO NOTHING;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_policy_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    IF OLD.version IS NOT NULL AND OLD IS DISTINCT FROM NEW THEN
        NEW.version = OLD.version + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER policies_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_updated_at();

CREATE TRIGGER approval_requests_updated_at
    BEFORE UPDATE ON approval_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_updated_at();

CREATE TRIGGER policy_templates_updated_at
    BEFORE UPDATE ON policy_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_policy_updated_at();

-- Grant permissions (adjust schema/user as needed)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON policies, policy_assignments, approval_requests, policy_evaluations, policy_templates, command_filter_patterns TO openpam_app;
-- GRANT USAGE, SELECT ON SEQUENCES TO openpam_app;
