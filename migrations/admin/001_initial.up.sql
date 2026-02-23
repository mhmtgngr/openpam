-- System configuration
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    tenant_id UUID NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Discovered assets
CREATE TABLE IF NOT EXISTS discovered_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hostname VARCHAR(255),
    ip VARCHAR(45) NOT NULL,
    os VARCHAR(100),
    open_ports JSONB DEFAULT '[]',
    services JSONB DEFAULT '[]',
    status VARCHAR(50) NOT NULL DEFAULT 'discovered',
    risk_score INTEGER DEFAULT 0,
    last_scanned_at TIMESTAMPTZ,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assets_tenant ON discovered_assets(tenant_id);
CREATE INDEX idx_assets_status ON discovered_assets(status);
CREATE INDEX idx_assets_ip ON discovered_assets(ip);
