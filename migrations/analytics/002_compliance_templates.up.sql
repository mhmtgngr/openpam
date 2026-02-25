-- Compliance Report Templates Database Schema
-- Migration 001: Compliance report templates and evidence collection

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Report Templates: Pre-built compliance report templates
CREATE TABLE IF NOT EXISTS report_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- NULL for global templates
    name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL, -- SOC2, ISO27001, PCI-DSS, HIPAA, NIST-800-53, GDPR
    version VARCHAR(50) NOT NULL,
    description TEXT,

    -- Template definition
    template_data JSONB NOT NULL DEFAULT '{}',

    -- Sections configuration
    sections JSONB NOT NULL DEFAULT '[]',

    -- Metadata
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_by UUID NOT NULL,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT report_templates_framework CHECK (
        framework IN ('SOC2', 'ISO27001', 'PCI-DSS', 'HIPAA', 'NIST-800-53', 'GDPR', 'CUSTOM')
    ),
    CONSTRAINT report_templates_unique UNIQUE (tenant_id, name, framework, version, deleted_at)
);

-- Indexes for report_templates
CREATE INDEX idx_report_templates_tenant ON report_templates(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_report_templates_framework ON report_templates(framework, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_report_templates_public ON report_templates(is_public) WHERE deleted_at IS NULL AND is_public = true;

-- Evidence: Collected evidence for compliance reports
CREATE TABLE IF NOT EXISTS evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    report_id UUID, -- Optional: associate with a specific report

    -- Evidence metadata
    type VARCHAR(100) NOT NULL, -- session_log, access_request, policy_review, risk_assessment, training_record, etc.
    category VARCHAR(100),
    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Evidence data
    evidence_data JSONB NOT NULL DEFAULT '{}',
    file_path TEXT, -- Path to stored evidence file
    file_hash VARCHAR(255), -- SHA-256 hash of the file

    -- Period validity
    applicable_from DATE NOT NULL,
    applicable_until DATE,

    -- Source
    source_type VARCHAR(50), -- manual, automated, imported
    source_id UUID,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, archived, deleted
    verified_by UUID,
    verified_at TIMESTAMPTZ,

    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT evidence_type CHECK (
        type IN ('session_log', 'access_request', 'policy_review', 'risk_assessment', 'training_record', 'incident_report', 'change_log', 'audit_trail', 'configuration', 'other')
    )
);

-- Indexes for evidence
CREATE INDEX idx_evidence_tenant ON evidence(tenant_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_report ON evidence(report_id, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_type ON evidence(type, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_category ON evidence(category, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_period ON evidence(applicable_from, applicable_until, deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_tags ON evidence USING GIN(tags) WHERE deleted_at IS NULL;

-- Report Runs: Generated compliance reports
CREATE TABLE IF NOT EXISTS report_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    template_id UUID NOT NULL REFERENCES report_templates(id) ON DELETE RESTRICT,
    report_name VARCHAR(255) NOT NULL,

    -- Run parameters
    framework VARCHAR(50) NOT NULL,
    version VARCHAR(50) NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,

    -- Results
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, generating, completed, failed
    overall_score NUMERIC(5, 2),
    total_controls INTEGER NOT NULL DEFAULT 0,
    passed_controls INTEGER NOT NULL DEFAULT 0,
    failed_controls INTEGER NOT NULL DEFAULT 0,
    skipped_controls INTEGER NOT NULL DEFAULT 0,

    -- Output
    report_path TEXT, -- Path to generated report file
    report_format VARCHAR(20) NOT NULL DEFAULT 'pdf', -- pdf, html, json

    -- Execution
    generated_by UUID NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_seconds INTEGER,

    -- Error handling
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for report_runs
CREATE INDEX idx_report_runs_tenant ON report_runs(tenant_id, started_at DESC);
CREATE INDEX idx_report_runs_template ON report_runs(template_id);
CREATE INDEX idx_report_runs_framework ON report_runs(framework, period_start DESC);
CREATE INDEX idx_report_runs_status ON report_runs(status);

-- Control Evaluations: Detailed control results for each report
CREATE TABLE IF NOT EXISTS control_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_run_id UUID NOT NULL REFERENCES report_runs(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,

    -- Control identification
    control_id VARCHAR(100) NOT NULL,
    control_name TEXT NOT NULL,
    control_category VARCHAR(100),

    -- Evaluation result
    status VARCHAR(50) NOT NULL, -- passed, failed, partial, not_applicable, not_tested
    score NUMERIC(5, 2),

    -- Evidence
    evidence_count INTEGER NOT NULL DEFAULT 0,
    evidence_urls TEXT[],
    findings TEXT,

    -- Risk assessment
    risk_level VARCHAR(20), -- low, medium, high, critical

    -- Remediation
    remediation_required BOOLEAN NOT NULL DEFAULT false,
    remediation_plan TEXT,
    remediation_due_date DATE,

    -- Metadata
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT control_evaluations_status CHECK (
        status IN ('passed', 'failed', 'partial', 'not_applicable', 'not_tested')
    )
);

-- Indexes for control_evaluations
CREATE INDEX idx_control_evaluations_report ON control_evaluations(report_run_id);
CREATE INDEX idx_control_evaluations_tenant ON control_evaluations(tenant_id);
CREATE INDEX idx_control_evaluations_control ON control_evaluations(control_id);
CREATE INDEX idx_control_evaluations_status ON control_evaluations(status);

-- Compliance Exceptions: Approved exceptions to controls
CREATE TABLE IF NOT EXISTS compliance_exceptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Exception details
    control_id VARCHAR(100) NOT NULL,
    control_name TEXT NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Exception status
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, denied, expired, revoked
    risk_level VARCHAR(20) NOT NULL DEFAULT 'medium',

    -- Approval workflow
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    expires_at DATE,

    -- Justification
    justification TEXT NOT NULL,
    business_reason TEXT,
    compensating_controls TEXT[],

    -- Risk acceptance
    risk_accepted_by UUID,
    risk_accepted_at TIMESTAMPTZ,

    -- Review
    review_date DATE,
    review_notes TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT compliance_exceptions_status CHECK (
        status IN ('pending', 'approved', 'denied', 'expired', 'revoked')
    )
);

-- Indexes for compliance_exceptions
CREATE INDEX idx_compliance_exceptions_tenant ON compliance_exceptions(tenant_id, status);
CREATE INDEX idx_compliance_exceptions_control ON compliance_exceptions(control_id, status);
CREATE INDEX idx_compliance_exceptions_framework ON compliance_exceptions(framework, status);
CREATE INDEX idx_compliance_exceptions_expires ON compliance_exceptions(expires_at) WHERE status = 'approved';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_compliance_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_report_templates_updated_at ON report_templates;
CREATE TRIGGER update_report_templates_updated_at
    BEFORE UPDATE ON report_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_compliance_updated_at_column();

DROP TRIGGER IF EXISTS update_evidence_updated_at ON evidence;
CREATE TRIGGER update_evidence_updated_at
    BEFORE UPDATE ON evidence
    FOR EACH ROW
    EXECUTE FUNCTION update_compliance_updated_at_column();

DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;
CREATE TRIGGER update_compliance_exceptions_updated_at
    BEFORE UPDATE ON compliance_exceptions
    FOR EACH ROW
    EXECUTE FUNCTION update_compliance_updated_at_column();

-- Function to get compliance summary for a tenant
CREATE OR REPLACE FUNCTION get_compliance_summary(p_tenant_id UUID)
RETURNS TABLE (
    framework VARCHAR(50),
    total_controls INTEGER,
    passed_controls INTEGER,
    failed_controls INTEGER,
    overall_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        framework,
        SUM(total_controls) as total_controls,
        SUM(passed_controls) as passed_controls,
        SUM(failed_controls) as failed_controls,
        AVG(overall_score) as overall_score
    FROM report_runs
    WHERE tenant_id = p_tenant_id
        AND status = 'completed'
    GROUP BY framework
    ORDER BY framework;
END;
$$ LANGUAGE plpgsql;
