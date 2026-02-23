-- Credential Vault
CREATE TABLE IF NOT EXISTS credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- ssh_key, password, api_key, certificate, database
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

CREATE INDEX idx_creds_tenant ON credentials(tenant_id);
CREATE INDEX idx_creds_type ON credentials(type);
CREATE INDEX idx_creds_folder ON credentials(folder_id);

-- Encryption keys (envelope encryption)
CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    encrypted_dek BYTEA NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at TIMESTAMPTZ
);

-- Credential folders
CREATE TABLE IF NOT EXISTS credential_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES credential_folders(id),
    tenant_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
