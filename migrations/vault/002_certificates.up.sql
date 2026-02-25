-- Certificate Management Migration
-- Migration 002: Add certificate tables for PKI and ACME

-- Certificates table for storing X.509 certificates
CREATE TABLE IF NOT EXISTS certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- root_ca, intermediate_ca, leaf, external, acme
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, expired, revoked, pending, renewing

    -- Certificate data (PEM encoded)
    pem_certificate TEXT NOT NULL,
    pem_private_key TEXT NOT NULL,
    serial_number VARCHAR(255) NOT NULL UNIQUE,
    subject VARCHAR(500) NOT NULL,
    issuer_id UUID REFERENCES certificates(id),

    -- Validity period
    not_before TIMESTAMPTZ NOT NULL,
    not_after TIMESTAMPTZ NOT NULL,

    -- Certificate extensions
    key_usage JSONB DEFAULT '[]',      -- digital_signature, key_encipherment, etc.
    ext_key_usage JSONB DEFAULT '[]',  -- server_auth, client_auth, etc.
    dns_names JSONB DEFAULT '[]',
    ip_addresses JSONB DEFAULT '[]',

    -- ACME-specific fields
    acme_account_id VARCHAR(255),
    acme_order_url TEXT,

    -- Revocation
    revoked_at TIMESTAMPTZ,
    revoked_by UUID,
    revocation_reason TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT certificates_type CHECK (
        type IN ('root_ca', 'intermediate_ca', 'leaf', 'external', 'acme')
    ),
    CONSTRAINT certificates_status CHECK (
        status IN ('active', 'expired', 'revoked', 'pending', 'renewing')
    )
);

-- Indexes for certificates
CREATE INDEX idx_certificates_tenant ON certificates(tenant_id);
CREATE INDEX idx_certificates_type ON certificates(type);
CREATE INDEX idx_certificates_status ON certificates(status);
CREATE INDEX idx_certificates_issuer ON certificates(issuer_id);
CREATE INDEX idx_certificates_serial ON certificates(serial_number);
CREATE INDEX idx_certificates_expires ON certificates(not_after) WHERE status = 'active';
CREATE INDEX idx_certificates_dns_names ON certificates USING GIN(dns_names);

-- Certificate Signing Requests (for pending certificates)
CREATE TABLE IF NOT EXISTS certificate_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    certificate_id UUID REFERENCES certificates(id),

    -- Request details
    common_name VARCHAR(255) NOT NULL,
    organization VARCHAR(255),
    organizational_unit VARCHAR(255),
    country VARCHAR(2),
    locality VARCHAR(255),
    province VARCHAR(255),

    -- Certificate specifications
    dns_names JSONB DEFAULT '[]',
    ip_addresses JSONB DEFAULT '[]',
    key_usage JSONB DEFAULT '[]',
    ext_key_usage JSONB DEFAULT '[]',
    duration_hours INTEGER NOT NULL,

    -- CSR data
    csr_pem TEXT NOT NULL,
    private_key_pem TEXT,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, issued
    requested_by UUID NOT NULL,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    issued_certificate_id UUID REFERENCES certificates(id),

    -- Metadata
    justification TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cert_requests_tenant ON certificate_requests(tenant_id);
CREATE INDEX idx_cert_requests_status ON certificate_requests(status);

-- Certificate Revocation Lists
CREATE TABLE IF NOT EXISTS certificate_crls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    certificate_id UUID REFERENCES certificates(id) NOT NULL,
    issuer_id UUID REFERENCES certificates(id) NOT NULL,

    -- CRL data
    crl_number BIGINT NOT NULL,
    crl_pem TEXT NOT NULL,
    revoked_serials JSONB DEFAULT '[]',

    -- Validity
    this_update TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_update TIMESTAMPTZ NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crls_issuer ON certificate_crls(issuer_id);
CREATE INDEX idx_crls_certificate ON certificate_crls(certificate_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_certificates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
CREATE TRIGGER update_certificates_updated_at_trigger
    BEFORE UPDATE ON certificates
    FOR EACH ROW
    EXECUTE FUNCTION update_certificates_updated_at();

CREATE TRIGGER update_cert_requests_updated_at_trigger
    BEFORE UPDATE ON certificate_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_certificates_updated_at();

-- Function to check for expiring certificates
CREATE OR REPLACE FUNCTION get_expiring_certificates(p_days INTEGER)
RETURNS TABLE (
    id UUID,
    tenant_id UUID,
    name VARCHAR,
    type VARCHAR,
    not_after TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.tenant_id,
        c.name,
        c.type,
        c.not_after
    FROM certificates c
    WHERE c.status = 'active'
        AND c.not_after <= NOW() + (p_days || ' days')::INTERVAL
        AND c.not_after > NOW()
    ORDER BY c.not_after ASC;
END;
$$ LANGUAGE plpgsql;
