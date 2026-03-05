-- OpenPAM Database Schema
-- Initial migration for all core tables

-- ============================================
-- TENANT MANAGEMENT
-- ============================================

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) UNIQUE NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free', -- free, pro, enterprise
    config JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, cancelled
    user_limit INTEGER NOT NULL DEFAULT 10,
    storage_gb INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tenants_domain ON tenants(domain);
CREATE INDEX idx_tenants_status ON tenants(status);

-- ============================================
-- USER MANAGEMENT
-- ============================================

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user', -- super_admin, admin, operator, auditor, user
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, suspended, locked
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- MFA
    mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_secret TEXT,
    backup_codes TEXT,

    -- Account security
    last_login_at TIMESTAMPTZ,
    failed_logins INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- ============================================
-- ROLE AND PERMISSION MANAGEMENT
-- ============================================

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    inherits_from_id UUID REFERENCES roles(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource VARCHAR(100) NOT NULL, -- credential, session, user, target, etc.
    action VARCHAR(100) NOT NULL,   -- create, read, update, delete, approve, etc.
    scope VARCHAR(50) NOT NULL DEFAULT 'all', -- all, own, team
    description TEXT,
    UNIQUE(resource, action, scope)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY(role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID REFERENCES users(id),
    PRIMARY KEY(user_id, role_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);

-- ============================================
-- TARGET SYSTEMS
-- ============================================

CREATE TABLE IF NOT EXISTS targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- ssh, rdp, database, web, kubernetes, api
    environment VARCHAR(50) NOT NULL, -- production, staging, development, test
    sensitivity VARCHAR(50) NOT NULL, -- critical, high, medium, low
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Connection
    host VARCHAR(500) NOT NULL,
    port INTEGER NOT NULL,
    connection_string TEXT,

    -- Access control
    require_approval BOOLEAN NOT NULL DEFAULT FALSE,
    require_mfa BOOLEAN NOT NULL DEFAULT FALSE,
    max_duration_minutes INTEGER NOT NULL DEFAULT 60,
    approval_group_id UUID,

    -- Configuration
    platform VARCHAR(100),
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_verified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_targets_tenant ON targets(tenant_id);
CREATE INDEX idx_targets_type ON targets(type);
CREATE INDEX idx_targets_environment ON targets(environment);

-- ============================================
-- CREDENTIAL VAULT
-- ============================================

CREATE TABLE IF NOT EXISTS secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL, -- password, ssh_key, api_token, certificate, database, aws_key, azure_key

    -- Target association
    target_id UUID REFERENCES targets(id) ON DELETE SET NULL,
    host VARCHAR(500),
    port INTEGER,
    username TEXT NOT NULL,

    -- Encrypted data (envelope encrypted)
    encrypted_secret BYTEA NOT NULL,
    ssh_key_passphrase BYTEA,

    -- Rotation
    rotation_policy VARCHAR(50) NOT NULL DEFAULT 'manual', -- manual, daily, weekly, monthly, on_checkin
    last_rotated_at TIMESTAMPTZ,
    next_rotation_at TIMESTAMPTZ,

    -- Organization
    folder_id UUID,
    tags TEXT[] DEFAULT '{}',
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Access control
    created_by UUID NOT NULL REFERENCES users(id),
    owner_id UUID REFERENCES users(id),

    status VARCHAR(50) NOT NULL DEFAULT 'active',
    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_secrets_tenant ON secrets(tenant_id);
CREATE INDEX idx_secrets_target ON secrets(target_id);
CREATE INDEX idx_secrets_folder ON secrets(folder_id);
CREATE INDEX idx_secrets_status ON secrets(status);
CREATE INDEX idx_secrets_next_rotation ON secrets(next_rotation_at) WHERE next_rotation_at IS NOT NULL;

-- Encrypted DEK storage (envelope encryption)
CREATE TABLE IF NOT EXISTS encrypted_deks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    key_id VARCHAR(255) NOT NULL,
    encrypted_key BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_encrypted_deks_credential ON encrypted_deks(credential_id);

-- Key metadata for rotation tracking
CREATE TABLE IF NOT EXISTS key_metadata (
    key_id VARCHAR(255) PRIMARY KEY,
    key_version INTEGER NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    key_size INTEGER NOT NULL DEFAULT 256,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ
);

-- ============================================
-- APPROVAL WORKFLOW
-- ============================================

CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL, -- credential_access, session_access, privilege_escalation
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    target_id UUID REFERENCES targets(id),
    credential_id UUID REFERENCES secrets(id),

    justification TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, denied, cancelled, expired, escalated
    priority VARCHAR(50) NOT NULL DEFAULT 'normal', -- low, normal, high, emergency

    -- Approval config
    approval_group_id UUID,
    required_approvals INTEGER NOT NULL DEFAULT 1,
    received_approvals INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    expires_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    denied_at TIMESTAMPTZ,

    ticket_ref VARCHAR(255),
    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_requests_tenant ON approval_requests(tenant_id);
CREATE INDEX idx_approval_requests_user ON approval_requests(user_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_approval_requests_credential ON approval_requests(credential_id);

CREATE TABLE IF NOT EXISTS approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
    approver_id UUID NOT NULL REFERENCES users(id),
    decision VARCHAR(50) NOT NULL, -- approve, deny
    comments TEXT,
    delegated_from UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approvals_request ON approvals(request_id);
CREATE INDEX idx_approvals_approver ON approvals(approver_id);

CREATE TABLE IF NOT EXISTS approval_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    members UUID[] NOT NULL DEFAULT '{}',
    required_approvals INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_groups_tenant ON approval_groups(tenant_id);

-- ============================================
-- CHECKOUT WORKFLOW
-- ============================================

CREATE TABLE IF NOT EXISTS checkouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    credential_id UUID NOT NULL REFERENCES secrets(id),
    request_id UUID REFERENCES approval_requests(id),

    justification TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, denied, checked_out, checked_in, expired, revoked

    -- Approval
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,

    -- Checkout
    checked_out_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    checked_in_at TIMESTAMPTZ,

    -- Revocation
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id),
    revoked_reason TEXT,

    -- Session tracking
    session_id UUID,

    -- Break-glass
    is_break_glass BOOLEAN NOT NULL DEFAULT FALSE,
    break_glass_reason TEXT,

    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkouts_tenant ON checkouts(tenant_id);
CREATE INDEX idx_checkouts_user ON checkouts(user_id);
CREATE INDEX idx_checkouts_credential ON checkouts(credential_id);
CREATE INDEX idx_checkouts_status ON checkouts(status);
CREATE INDEX idx_checkouts_expires_at ON checkouts(expires_at) WHERE status = 'checked_out';

CREATE TABLE IF NOT EXISTS checkout_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_id UUID NOT NULL REFERENCES checkouts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    credential_id UUID NOT NULL REFERENCES secrets(id),
    action VARCHAR(50) NOT NULL, -- checkout, checkin, view
    client_ip VARCHAR(45),
    user_agent TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkout_records_checkout ON checkout_records(checkout_id);

-- ============================================
-- SESSIONS
-- ============================================

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    credential_id UUID REFERENCES secrets(id),
    target_id UUID REFERENCES targets(id),

    type VARCHAR(50) NOT NULL, -- ssh, rdp, database, kubernetes, web, api
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, ended, terminated, failed

    -- Connection
    target_host VARCHAR(500) NOT NULL,
    target_port INTEGER NOT NULL,
    client_ip VARCHAR(45),
    user_agent TEXT,

    -- Recording
    recording_id UUID,
    recording_url TEXT,

    -- Timing
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    terminated_by UUID REFERENCES users(id),
    terminate_reason TEXT,

    metadata JSONB,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_credential ON sessions(credential_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_started_at ON sessions(started_at);

-- ============================================
-- AUDIT LOG
-- ============================================

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL,
    actor_type VARCHAR(50) NOT NULL, -- user, system, api
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(500),
    outcome VARCHAR(50) NOT NULL, -- success, failure, denied

    -- Request info
    ip VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(255),

    -- Details
    details JSONB,
    error_code VARCHAR(100),
    error_message TEXT,

    -- Chain integrity
    previous_hash TEXT,
    hash TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_tenant ON audit_events(tenant_id);
CREATE INDEX idx_audit_events_actor ON audit_events(actor_id);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_resource_type ON audit_events(resource_type);
CREATE INDEX idx_audit_events_outcome ON audit_events(outcome);
CREATE INDEX idx_audit_events_created_at ON audit_events(created_at);
CREATE INDEX idx_audit_events_hash ON audit_events(hash);

-- ============================================
-- RECORDINGS
-- ============================================

CREATE TABLE IF NOT EXISTS recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- keystrokes, video, audio, metadata
    status VARCHAR(50) NOT NULL DEFAULT 'recording', -- recording, processing, ready, failed, deleted

    -- Storage
    storage_url TEXT NOT NULL,
    storage_provider VARCHAR(50) NOT NULL DEFAULT 'minio',
    file_size BIGINT,
    checksum TEXT,

    -- Metadata
    duration_seconds INTEGER,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    format VARCHAR(50),
    is_encrypted BOOLEAN NOT NULL DEFAULT TRUE,
    encryption_key_id TEXT,

    -- Search
    transcript TEXT,
    search_index tsvector,

    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_recordings_tenant ON recordings(tenant_id);
CREATE INDEX idx_recordings_session ON recordings(session_id);
CREATE INDEX idx_recordings_type ON recordings(type);
CREATE INDEX idx_recordings_status ON recordings(status);
CREATE INDEX idx_recordings_transcript ON recordings USING GIN(to_tsvector('english', transcript));

CREATE TABLE IF NOT EXISTS recording_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recording_id UUID NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL,
    type VARCHAR(50) NOT NULL, -- keystroke, command, error, warning
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recording_events_recording ON recording_events(recording_id);
CREATE INDEX idx_recording_events_timestamp ON recording_events(timestamp);

-- ============================================
-- DISCOVERY
-- ============================================

CREATE TABLE IF NOT EXISTS discovery_scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,

    -- Configuration
    subnets TEXT[] NOT NULL DEFAULT '{}',
    port_range VARCHAR(100) NOT NULL DEFAULT '1-1024',

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Results
    total_hosts INTEGER NOT NULL DEFAULT 0,
    discovered_assets INTEGER NOT NULL DEFAULT 0,
    new_assets INTEGER NOT NULL DEFAULT 0,
    changed_assets INTEGER NOT NULL DEFAULT 0,

    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_discovery_scans_tenant ON discovery_scans(tenant_id);
CREATE INDEX idx_discovery_scans_status ON discovery_scans(status);

CREATE TABLE IF NOT EXISTS discovered_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scan_id UUID NOT NULL REFERENCES discovery_scans(id) ON DELETE CASCADE,

    -- Asset details
    hostname VARCHAR(255),
    ip VARCHAR(45) NOT NULL,
    mac VARCHAR(17),
    os VARCHAR(100),
    os_version VARCHAR(100),

    -- Network details
    open_ports JSONB DEFAULT '[]',
    services JSONB DEFAULT '[]',

    -- Classification
    asset_type VARCHAR(50), -- server, workstation, network_device, database
    sensitivity VARCHAR(50), -- critical, high, medium, low

    status VARCHAR(50) NOT NULL DEFAULT 'discovered', -- discovered, managed, ignored, false_positive
    last_scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_discovered_assets_tenant ON discovered_assets(tenant_id);
CREATE INDEX idx_discovered_assets_scan ON discovered_assets(scan_id);
CREATE INDEX idx_discovered_assets_ip ON discovered_assets(ip);
CREATE INDEX idx_discovered_assets_status ON discovered_assets(status);

-- ============================================
-- ROTATION TASKS
-- ============================================

CREATE TABLE IF NOT EXISTS rotation_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    policy VARCHAR(50) NOT NULL, -- manual, daily, weekly, monthly, on_checkin
    next_rotation TIMESTAMPTZ NOT NULL,
    last_rotation TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' -- pending, running, completed, failed
);

CREATE INDEX idx_rotation_tasks_credential ON rotation_tasks(credential_id);
CREATE INDEX idx_rotation_tasks_next_rotation ON rotation_tasks(next_rotation);

-- ============================================
-- CREDENTIAL HISTORY
-- ============================================

CREATE TABLE IF NOT EXISTS credential_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_by UUID REFERENCES users(id),
    method VARCHAR(50) NOT NULL, -- manual, scheduled, on_checkin
    success BOOLEAN NOT NULL DEFAULT TRUE,
    error_msg TEXT
);

CREATE INDEX idx_credential_history_credential ON credential_history(credential_id);
CREATE INDEX idx_credential_history_rotated_at ON credential_history(rotated_at);

-- ============================================
-- FUNCTIONS AND TRIGGERS
-- ============================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to relevant tables
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_targets_updated_at BEFORE UPDATE ON targets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_secrets_updated_at BEFORE UPDATE ON secrets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_approval_requests_updated_at BEFORE UPDATE ON approval_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_checkouts_updated_at BEFORE UPDATE ON checkouts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_updated_at BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_approval_groups_updated_at BEFORE UPDATE ON approval_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_discovery_scans_updated_at BEFORE UPDATE ON discovery_scans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_discovered_assets_updated_at BEFORE UPDATE ON discovered_assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recordings_updated_at BEFORE UPDATE ON recordings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
