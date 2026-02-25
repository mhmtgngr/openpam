-- Kubernetes Access Management Migration
-- Migration 002: Add tables for Kubernetes JIT access

-- Kubernetes access requests (JIT role bindings)
CREATE TABLE IF NOT EXISTS kubernetes_access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    request_id VARCHAR(255) NOT NULL UNIQUE, -- Generated request ID

    -- User info
    username VARCHAR(255) NOT NULL,
    groups JSONB DEFAULT '[]',

    -- Kubernetes resource
    namespace VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,        -- Role or ClusterRole name
    kind VARCHAR(50) NOT NULL,         -- Role, ClusterRole
    resource VARCHAR(50) NOT NULL,     -- RoleBinding, ClusterRoleBinding

    -- Request details
    duration_seconds INTEGER NOT NULL,
    reason TEXT NOT NULL,
    approved_by UUID,
    approval_method VARCHAR(50),       -- mfa, approval, auto

    -- Binding info
    binding_name VARCHAR(255) NOT NULL,
    kubeconfig TEXT,                   -- Temporary kubeconfig (encrypted)

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, expired, revoked
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,

    -- Metadata
    cluster_name VARCHAR(255),
    cluster_endpoint VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT k8s_access_status CHECK (
        status IN ('pending', 'active', 'expired', 'revoked')
    ),
    CONSTRAINT k8s_access_kind CHECK (
        kind IN ('Role', 'ClusterRole')
    ),
    CONSTRAINT k8s_access_resource CHECK (
        resource IN ('RoleBinding', 'ClusterRoleBinding')
    )
);

CREATE INDEX idx_k8s_access_tenant ON kubernetes_access_requests(tenant_id);
CREATE INDEX idx_k8s_access_status ON kubernetes_access_requests(status);
CREATE INDEX idx_k8s_access_user ON kubernetes_access_requests(username);
CREATE INDEX idx_k8s_access_namespace ON kubernetes_access_requests(namespace);
CREATE INDEX idx_k8s_access_expires ON kubernetes_access_requests(expires_at) WHERE status = 'active';

-- Kubernetes cluster configurations
CREATE TABLE IF NOT EXISTS kubernetes_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,

    -- Connection info
    api_endpoint VARCHAR(500) NOT NULL,
    certificate_authority_data TEXT,   -- Base64 encoded CA cert

    -- Authentication
    auth_type VARCHAR(50) NOT NULL DEFAULT 'certificate', -- certificate, token, oidc
    auth_config JSONB NOT NULL DEFAULT '{}',

    -- Service account for JIT access
    service_account_namespace VARCHAR(255),
    service_account_name VARCHAR(255),

    -- Configuration
    default_namespace VARCHAR(255) DEFAULT 'default',
    allowed_namespaces JSONB DEFAULT '[]',
    max_duration_seconds INTEGER NOT NULL DEFAULT 3600,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, error
    last_health_check TIMESTAMPTZ,
    health_check_status VARCHAR(50),
    error_message TEXT,

    -- Metadata
    description TEXT,
    labels JSONB DEFAULT '{}',
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT k8s_clusters_status CHECK (
        status IN ('active', 'inactive', 'error')
    )
);

CREATE UNIQUE INDEX idx_k8s_clusters_tenant_name ON kubernetes_clusters(tenant_id, name);
CREATE INDEX idx_k8s_clusters_tenant ON kubernetes_clusters(tenant_id);
CREATE INDEX idx_k8s_clusters_status ON kubernetes_clusters(status);

-- Kubernetes access templates (pre-configured access patterns)
CREATE TABLE IF NOT EXISTS kubernetes_access_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Template configuration
    cluster_id UUID NOT NULL REFERENCES kubernetes_clusters(id),
    namespace VARCHAR(255) NOT NULL,
    role VARCHAR(255) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    resource VARCHAR(50) NOT NULL,

    -- Access control
    allowed_groups JSONB DEFAULT '[]',
    approval_required BOOLEAN NOT NULL DEFAULT true,
    max_duration_seconds INTEGER NOT NULL DEFAULT 3600,

    -- Status
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_k8s_templates_tenant ON kubernetes_access_templates(tenant_id);
CREATE INDEX idx_k8s_templates_cluster ON kubernetes_access_templates(cluster_id);
CREATE INDEX idx_k8s_templates_enabled ON kubernetes_access_templates(tenant_id, enabled) WHERE enabled = true;

-- Kubernetes audit log
CREATE TABLE IF NOT EXISTS kubernetes_audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    cluster_id UUID REFERENCES kubernetes_clusters(id),
    access_request_id UUID REFERENCES kubernetes_access_requests(id),

    -- Event details
    event_type VARCHAR(50) NOT NULL, -- request, grant, revoke, command, error
    username VARCHAR(255) NOT NULL,
    namespace VARCHAR(255),

    -- Action details
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100),
    resource_name VARCHAR(255),
    command TEXT,

    -- Result
    status VARCHAR(50) NOT NULL, -- success, denied, error
    error_message TEXT,

    -- Context
    source_ip VARCHAR(45),
    user_agent TEXT,
    metadata JSONB DEFAULT '{}',

    -- Timestamp
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_k8s_audit_tenant ON kubernetes_audit_events(tenant_id);
CREATE INDEX idx_k8s_audit_cluster ON kubernetes_audit_events(cluster_id);
CREATE INDEX idx_k8s_audit_user ON kubernetes_audit_events(username);
CREATE INDEX idx_k8s_audit_type ON kubernetes_audit_events(event_type);
CREATE INDEX idx_k8s_audit_occurred ON kubernetes_audit_events(occurred_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_k8s_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER update_k8s_access_updated_at
    BEFORE UPDATE ON kubernetes_access_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_k8s_updated_at();

CREATE TRIGGER update_k8s_clusters_updated_at
    BEFORE UPDATE ON kubernetes_clusters
    FOR EACH ROW
    EXECUTE FUNCTION update_k8s_updated_at();

CREATE TRIGGER update_k8s_templates_updated_at
    BEFORE UPDATE ON kubernetes_access_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_k8s_updated_at();

-- Function to expire old Kubernetes access
CREATE OR REPLACE FUNCTION expire_kubernetes_access()
RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    UPDATE kubernetes_access_requests
    SET status = 'expired', updated_at = NOW()
    WHERE status = 'active'
        AND expires_at <= NOW();

    GET DIAGNOSTICS expired_count = ROW_COUNT;
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get active Kubernetes access count
CREATE OR REPLACE FUNCTION get_active_k8s_access_count(p_tenant_id UUID)
RETURNS BIGINT AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)
        FROM kubernetes_access_requests
        WHERE tenant_id = p_tenant_id
            AND status = 'active'
            AND expires_at > NOW()
    );
END;
$$ LANGUAGE plpgsql;
