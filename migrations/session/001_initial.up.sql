-- Privileged Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    credential_id UUID NOT NULL,
    checkout_id UUID,
    type VARCHAR(50) NOT NULL, -- ssh, rdp, database, kubernetes, web
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    target_host VARCHAR(255) NOT NULL,
    target_port INTEGER NOT NULL,
    recording_url TEXT,
    recording_size_bytes BIGINT,
    commands_count INTEGER DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    terminated_by UUID,
    termination_reason TEXT,
    tenant_id UUID NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_started ON sessions(started_at);

-- Session recordings metadata
CREATE TABLE IF NOT EXISTS session_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id),
    storage_path TEXT NOT NULL,
    encryption_key_id UUID NOT NULL,
    size_bytes BIGINT,
    duration_seconds INTEGER,
    format VARCHAR(50) NOT NULL, -- asciicast, mp4, json
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
