-- Certificate Management Database Schema
-- Migration 001: Certificate lifecycle management

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Certificates: Managed certificates inventory
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- internal, external, acme
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, active, expired, revoked

    -- Certificate data (encrypted at rest)
    certificate_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    chain_pem TEXT,

    -- Certificate metadata
    serial_number VARCHAR(255),
    issuer VARCHAR(255),
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,

    -- Renewal configuration
    auto_renew BOOLEAN NOT NULL DEFAULT false,
    renew_before_days INTEGER NOT NULL DEFAULT 30,

    -- Key management
    key_size INTEGER NOT NULL DEFAULT 2048,
    key_type VARCHAR(20) NOT NULL DEFAULT 'RSA', -- RSA, ECDSA

    -- Tracking
    last_renewed_at TIMESTAMPTZ,
    renewal_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    last_failure_at TIMESTAMPTZ,
    last_error TEXT,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT certificates_status CHECK (
        status IN ('pending', 'active', 'expired', 'revoked')
    ),
    CONSTRAINT certificates_type CHECK (
        type IN ('internal', 'external', 'acme')
    )
);

-- Indexes for certificates
CREATE INDEX idx_certificates_tenant ON certificates(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_certificates_domain ON certificates(domain, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_certificates_status ON certificates(status, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_certificates_type ON certificates(type, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_certificates_expires_at ON certificates(expires_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_certificates_auto_renew ON certificates(tenant_id, auto_renew) WHERE deleted_at IS NULL AND auto_renew = true;

-- Certificate Signing Requests (CSRs)
CREATE TABLE IF NOT EXISTS certificate_signing_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    certificate_id UUID REFERENCES certificates(id) ON DELETE SET NULL,
    common_name VARCHAR(255) NOT NULL,
    sans TEXT[], -- Subject Alternative Names
    organization VARCHAR(255),
    organizational_unit VARCHAR(255),
    country VARCHAR(2),
    state VARCHAR(255),
    locality VARCHAR(255),

    -- CSR data
    csr_pem TEXT NOT NULL,
    private_key_pem TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, expired
    approved_by UUID,
    approved_at TIMESTAMPTZ,

    -- ACME challenge data
    challenge_token VARCHAR(255),
    challenge_url TEXT,
    challenge_valid_until TIMESTAMPTZ,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes for CSRs
CREATE INDEX idx_csrs_tenant ON certificate_signing_requests(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_csrs_certificate ON certificate_signing_requests(certificate_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_csrs_status ON certificate_signing_requests(status, deleted_at) WHERE deleted_at IS NULL;

-- Certificate Revocation List (CRL)
CREATE TABLE IF NOT EXISTS certificate_revocation_list (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    certificate_id UUID REFERENCES certificates(id) ON DELETE CASCADE,
    serial_number VARCHAR(255) NOT NULL,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_by UUID NOT NULL,
    reason TEXT,

    -- CRL entry data
    crl_number BIGINT,
    invalidity_date TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for CRL
CREATE INDEX idx_crl_tenant ON certificate_revocation_list(tenant_id);
CREATE INDEX idx_crl_certificate ON certificate_revocation_list(certificate_id);
CREATE INDEX idx_crl_serial ON certificate_revocation_list(serial_number);

-- Function to check for certificates needing renewal
CREATE OR REPLACE FUNCTION get_certificates_for_renewal()
RETURNS TABLE (
    id UUID,
    tenant_id UUID,
    name VARCHAR(255),
    domain VARCHAR(255),
    expires_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.tenant_id,
        c.name,
        c.domain,
        c.expires_at
    FROM certificates c
    WHERE c.status = 'active'
        AND c.auto_renew = true
        AND c.deleted_at IS NULL
        AND c.expires_at <= NOW() + (
            (SELECT renew_before_days FROM certificates c2 WHERE c2.id = c.id) * INTERVAL '1 day'
        )
        AND c.expires_at > NOW()
    ORDER BY c.expires_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to update certificate status based on expiration
CREATE OR REPLACE FUNCTION update_expired_certificates()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    UPDATE certificates
    SET status = 'expired',
        updated_at = NOW()
    WHERE status = 'active'
        AND expires_at <= NOW()
        AND deleted_at IS NULL;

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;
