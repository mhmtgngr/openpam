-- Security Fix: Add secure storage for encrypted DEKs
-- This migration creates a dedicated table for storing encrypted Data Encryption Keys
-- instead of storing them in MinIO object metadata, which is not encrypted at rest
-- and may leak in logs/API responses.

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

-- Indexes for efficient lookups
CREATE INDEX idx_recording_keys_session ON recording_keys(session_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_recording_keys_object ON recording_keys(object_name) WHERE deleted_at IS NULL;
CREATE INDEX idx_recording_keys_deleted ON recording_keys(deleted_at);

-- Add comment for documentation
COMMENT ON TABLE recording_keys IS 'Secure storage for encrypted DEKs used in session recordings. DEKs are encrypted with the master key and stored here instead of MinIO metadata.';
COMMENT ON COLUMN recording_keys.encrypted_dek IS 'Data Encryption Key (DEK) encrypted with the master key using AES-256-GCM';
COMMENT ON COLUMN recording_keys.object_name IS 'MinIO object name (reference to the actual encrypted recording)';
