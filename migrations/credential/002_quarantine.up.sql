-- Credential Quarantine Table
-- SECURITY: Stores credentials that failed rotation scheduling rollback
-- These credentials exist in the vault but have no rotation scheduled,
-- requiring immediate admin intervention to either fix rotation or delete
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
    severity VARCHAR(20) NOT NULL DEFAULT 'critical' -- critical, high, medium
);

-- Indexes for quarantine lookups
CREATE INDEX idx_quarantine_credential ON credential_quarantine(credential_id);
CREATE INDEX idx_quarantine_tenant ON credential_quarantine(tenant_id);
CREATE INDEX idx_quarantine_resolved ON credential_quarantine(resolved_at);
CREATE INDEX idx_quarantine_severity ON credential_quarantine(severity);

-- Comment for documentation
COMMENT ON TABLE credential_quarantine IS 'Stores credentials that entered an inconsistent state during creation (rotation scheduling failed and rollback also failed)';
COMMENT ON COLUMN credential_quarantine.severity IS 'Severity level: critical=both rotation and rollback failed, high=rotation failed with uncertain state';
