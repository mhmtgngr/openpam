-- Credential Checkout / Check-in
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

CREATE INDEX idx_checkout_user ON checkout_requests(user_id);
CREATE INDEX idx_checkout_cred ON checkout_requests(credential_id);
CREATE INDEX idx_checkout_status ON checkout_requests(status);
CREATE INDEX idx_checkout_expires ON checkout_requests(expires_at);

-- Credential rotation history
CREATE TABLE IF NOT EXISTS rotation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id UUID NOT NULL,
    trigger VARCHAR(50) NOT NULL, -- scheduled, on_checkin, manual, emergency
    status VARCHAR(50) NOT NULL, -- success, failed, rolled_back
    old_version_hash TEXT,
    new_version_hash TEXT,
    error_message TEXT,
    tenant_id UUID NOT NULL,
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
