-- OpenPAM Service-Specific Schemas
-- Runs after core schema (00_core_schema.sql)
-- Creates additional tables needed by individual services

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================
-- VAULT SERVICE: Encryption keys and folders
-- ============================================

CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    host VARCHAR(255),
    port INTEGER,
    username VARCHAR(255),
    encrypted_secret BYTEA NOT NULL,
    encryption_key_id UUID NOT NULL,
    rotation_policy VARCHAR(50) DEFAULT 'manual',
    last_rotated_at TIMESTAMPTZ,
    folder_id UUID,
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_creds_tenant ON credentials(tenant_id);
CREATE INDEX IF NOT EXISTS idx_creds_type ON credentials(type);
CREATE INDEX IF NOT EXISTS idx_creds_folder ON credentials(folder_id);

CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    encrypted_dek BYTEA NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS credential_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES credential_folders(id),
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- CREDENTIAL SERVICE: Checkout and rotation
-- ============================================

CREATE TABLE IF NOT EXISTS checkout_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    credential_id UUID NOT NULL,
    justification TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 60,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    denied_reason TEXT,
    checked_out_at TIMESTAMPTZ,
    checked_in_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    auto_rotated BOOLEAN DEFAULT false,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checkout_user ON checkout_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_checkout_cred ON checkout_requests(credential_id);
CREATE INDEX IF NOT EXISTS idx_checkout_status ON checkout_requests(status);
CREATE INDEX IF NOT EXISTS idx_checkout_expires ON checkout_requests(expires_at);

CREATE TABLE IF NOT EXISTS rotation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL,
    trigger VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    old_version_hash TEXT,
    new_version_hash TEXT,
    error_message TEXT,
    tenant_id UUID NOT NULL,
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credential_quarantine (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL UNIQUE,
    credential_name VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL,
    rotation_policy VARCHAR(50) NOT NULL,
    quarantine_reason TEXT NOT NULL,
    original_error TEXT NOT NULL,
    rollback_error TEXT,
    quarantined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    resolution_notes TEXT,
    severity VARCHAR(20) NOT NULL DEFAULT 'critical'
);

-- ============================================
-- SESSION SERVICE: Recordings and DEK storage
-- ============================================

CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    storage_path TEXT NOT NULL,
    encryption_key_id UUID NOT NULL,
    size_bytes BIGINT,
    duration_seconds INTEGER,
    format VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS recording_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    object_name TEXT NOT NULL UNIQUE,
    encrypted_dek BYTEA NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_recording_keys_session ON recording_keys(session_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_recording_keys_object ON recording_keys(object_name) WHERE deleted_at IS NULL;

-- ============================================
-- AUDIT SERVICE: Compliance and analytics
-- ============================================

CREATE TABLE IF NOT EXISTS access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rules JSONB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    tenant_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    report_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,
    version VARCHAR(50),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    overall_score NUMERIC(5,2),
    total_controls INTEGER NOT NULL DEFAULT 0,
    passed_controls INTEGER NOT NULL DEFAULT 0,
    failed_controls INTEGER NOT NULL DEFAULT 0,
    skipped_controls INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    summary TEXT,
    findings JSONB,
    recommendations JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_reports_tenant ON compliance_reports(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_reports_framework ON compliance_reports(framework);

CREATE TABLE IF NOT EXISTS compliance_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    control_id VARCHAR(100) NOT NULL,
    control_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    risk_level VARCHAR(50) NOT NULL,
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    justification TEXT NOT NULL,
    business_reason TEXT,
    compensating_controls TEXT[],
    risk_accepted_by UUID,
    risk_accepted_at TIMESTAMPTZ,
    review_date TIMESTAMPTZ,
    review_notes TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS anomaly_detections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    anomaly_type VARCHAR(50) NOT NULL,
    user_id UUID,
    session_id UUID,
    target_host VARCHAR(255),
    severity VARCHAR(50) NOT NULL,
    confidence_score NUMERIC(5,2) NOT NULL,
    risk_score NUMERIC(5,2) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    indicators JSONB,
    detection_method VARCHAR(100) NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model_version VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    assigned_to UUID,
    resolution_notes TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by UUID,
    auto_triggered BOOLEAN NOT NULL DEFAULT false,
    auto_action_taken VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_anomaly_tenant ON anomaly_detections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_anomaly_type ON anomaly_detections(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_anomaly_severity ON anomaly_detections(severity);
CREATE INDEX IF NOT EXISTS idx_anomaly_status ON anomaly_detections(status);

CREATE TABLE IF NOT EXISTS ransomware_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    detection_id UUID NOT NULL,
    encryption_activity BOOLEAN NOT NULL DEFAULT false,
    mass_file_modification BOOLEAN NOT NULL DEFAULT false,
    suspicious_processes TEXT[],
    affected_paths TEXT[],
    files_affected INTEGER NOT NULL DEFAULT 0,
    systems_affected INTEGER NOT NULL DEFAULT 0,
    data_exfiltrated BOOLEAN NOT NULL DEFAULT false,
    emergency_triggered BOOLEAN NOT NULL DEFAULT false,
    sessions_terminated INTEGER NOT NULL DEFAULT 0,
    credentials_revoked INTEGER NOT NULL DEFAULT 0,
    containment_status VARCHAR(50),
    recovery_status VARCHAR(50),
    raw_indicators JSONB,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- ANALYTICS SERVICE: Metrics and blacklist
-- ============================================

CREATE TABLE IF NOT EXISTS ssh_key_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    ssh_key_id UUID NOT NULL,
    date DATE NOT NULL,
    usage_count INTEGER NOT NULL DEFAULT 0,
    unique_users INTEGER NOT NULL DEFAULT 0,
    unique_targets INTEGER NOT NULL DEFAULT 0,
    first_use_time TIMESTAMPTZ,
    last_use_time TIMESTAMPTZ,
    avg_session_duration_seconds NUMERIC(10,2),
    off_hours_usage INTEGER NOT NULL DEFAULT 0,
    unusual_source_usage INTEGER NOT NULL DEFAULT 0,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, ssh_key_id, date)
);

CREATE TABLE IF NOT EXISTS command_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    command_pattern TEXT NOT NULL,
    pattern_type VARCHAR(20) NOT NULL,
    base_command VARCHAR(100),
    action VARCHAR(20) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    applies_to_users UUID[],
    applies_to_groups UUID[],
    applies_to_targets TEXT[],
    allow_override BOOLEAN NOT NULL DEFAULT false,
    override_roles TEXT[],
    reason TEXT NOT NULL,
    risk_category VARCHAR(50),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- ADMIN SERVICE: System config
-- ============================================

CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    tenant_id UUID NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================
-- IDENTITY SERVICE: Groups, access requests
-- ============================================

CREATE TABLE IF NOT EXISTS groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID,
    PRIMARY KEY(group_id, user_id)
);

CREATE TABLE IF NOT EXISTS access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID NOT NULL,
    target_id UUID,
    credential_id UUID,
    request_type VARCHAR(50) NOT NULL,
    justification TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    duration_minutes INTEGER,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    denied_reason TEXT,
    ticket_id VARCHAR(255),
    tenant_id UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_access_requests_tenant ON access_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_access_requests_requester ON access_requests(requester_id);
CREATE INDEX IF NOT EXISTS idx_access_requests_status ON access_requests(status);

-- ============================================
-- SEED DATA: Default tenant and admin user
-- ============================================

-- Insert default tenant if not exists
INSERT INTO tenants (id, name, domain, plan, status, user_limit, storage_gb)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Organization',
    'localhost',
    'enterprise',
    'active',
    -1,
    100
) ON CONFLICT (domain) DO NOTHING;

-- Insert default admin user (password: admin123! - CHANGE IN PRODUCTION)
-- Password hash is bcrypt of 'admin123!'
INSERT INTO users (id, email, first_name, last_name, password_hash, role, status, tenant_id, mfa_enabled)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'admin@localhost',
    'System',
    'Admin',
    '$2a$10$rQEY4z4z4z4z4z4z4z4z4eAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA',
    'super_admin',
    'active',
    '00000000-0000-0000-0000-000000000001',
    false
) ON CONFLICT (tenant_id, email) DO NOTHING;
