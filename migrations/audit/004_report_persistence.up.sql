-- Report Persistence Migration for Audit Service
-- This migration creates the report_snapshots table for storing generated
-- compliance and analytics report instances with file URLs and status tracking.

-- ============================================================================
-- Report Snapshots Table
-- ============================================================================
-- This table stores generated report instances (PDF, Excel, etc.) that are
-- created from compliance_reports templates. It provides persistent storage
-- for generated reports with download URLs and generation status tracking.

CREATE TABLE IF NOT EXISTS report_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Reference to the source compliance report template
    report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,

    -- Report identification
    snapshot_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Generation metadata
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by UUID NOT NULL,

    -- Generation status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending: generation queued/in progress
    -- completed: report file generated successfully
    -- failed: generation failed
    -- expired: report file has expired and been removed

    -- File storage information
    file_url TEXT,
    file_size_bytes BIGINT,
    file_format VARCHAR(50), -- pdf, xlsx, csv, html, json
    storage_path TEXT, -- internal storage path for the file

    -- Report period (inherited from source report)
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,

    -- Report content summary
    summary TEXT,
    metadata JSONB DEFAULT '{}',

    -- Expiration and retention
    expires_at TIMESTAMPTZ,

    -- Error tracking for failed generations
    error_message TEXT,
    error_details JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT report_snapshots_status_check CHECK (status IN ('pending', 'completed', 'failed', 'expired')),
    CONSTRAINT report_snapshots_format_check CHECK (file_format IS NULL OR file_format IN ('pdf', 'xlsx', 'csv', 'html', 'json'))
);

-- Indexes for report_snapshots
CREATE INDEX idx_report_snapshots_tenant ON report_snapshots(tenant_id);
CREATE INDEX idx_report_snapshots_report_id ON report_snapshots(report_id);
CREATE INDEX idx_report_snapshots_framework ON report_snapshots(framework);
CREATE INDEX idx_report_snapshots_status ON report_snapshots(status);
CREATE INDEX idx_report_snapshots_generated_at ON report_snapshots(generated_at DESC);
CREATE INDEX idx_report_snapshots_period ON report_snapshots(period_start, period_end);
CREATE INDEX idx_report_snapshots_expires_at ON report_snapshots(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_report_snapshots_metadata ON report_snapshots USING GIN (metadata);

-- Partial index for completed reports (common query for listing available reports)
CREATE INDEX idx_report_snapshots_completed_tenant
    ON report_snapshots(tenant_id, generated_at DESC)
    WHERE status = 'completed';

-- ============================================================================
-- Report Generation Queue Table
-- ============================================================================
-- This table tracks asynchronous report generation jobs with their status
-- and allows for retry logic and progress tracking.

CREATE TABLE IF NOT EXISTS report_generation_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Job identification
    job_type VARCHAR(50) NOT NULL, -- compliance, analytics, custom
    snapshot_id UUID REFERENCES report_snapshots(id) ON DELETE SET NULL,

    -- Source report reference
    report_id UUID REFERENCES compliance_reports(id) ON DELETE CASCADE,

    -- Job status and progress
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    -- queued: job is queued for processing
    -- processing: job is currently being processed
    -- completed: job completed successfully
    -- failed: job failed
    -- cancelled: job was cancelled

    progress INTEGER NOT NULL DEFAULT 0, -- 0-100

    -- Job configuration
    format VARCHAR(50) NOT NULL, -- pdf, xlsx, csv, html, json
    options JSONB DEFAULT '{}', -- additional generation options

    -- Timing information
    queued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Error tracking
    error_message TEXT,
    error_details JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,

    -- Processing metadata
    worker_id VARCHAR(255),
    correlation_id UUID,

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT report_jobs_status_check CHECK (status IN ('queued', 'processing', 'completed', 'failed', 'cancelled')),
    CONSTRAINT report_jobs_format_check CHECK (format IN ('pdf', 'xlsx', 'csv', 'html', 'json')),
    CONSTRAINT report_jobs_progress_check CHECK (progress >= 0 AND progress <= 100)
);

-- Indexes for report_generation_jobs
CREATE INDEX idx_report_jobs_tenant ON report_generation_jobs(tenant_id);
CREATE INDEX idx_report_jobs_snapshot_id ON report_generation_jobs(snapshot_id);
CREATE INDEX idx_report_jobs_report_id ON report_generation_jobs(report_id);
CREATE INDEX idx_report_jobs_status ON report_generation_jobs(status);
CREATE INDEX idx_report_jobs_queued_at ON report_generation_jobs(queued_at) WHERE status = 'queued';
CREATE INDEX idx_report_jobs_worker_id ON report_generation_jobs(worker_id) WHERE worker_id IS NOT NULL;
CREATE INDEX idx_report_jobs_metadata ON report_generation_jobs USING GIN (metadata);

-- ============================================================================
-- Report Schedule Table
-- ============================================================================
-- This table stores scheduled report generation jobs for recurring reports.

CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    -- Schedule identification
    schedule_name VARCHAR(255) NOT NULL,
    framework VARCHAR(50) NOT NULL,

    -- Source report configuration
    report_id UUID NOT NULL REFERENCES compliance_reports(id) ON DELETE CASCADE,

    -- Schedule configuration
    schedule_type VARCHAR(50) NOT NULL, -- daily, weekly, monthly, quarterly, yearly
    cron_expression TEXT, -- custom cron expression for complex schedules

    -- Output configuration
    format VARCHAR(50) NOT NULL DEFAULT 'pdf',
    options JSONB DEFAULT '{}',

    -- Recipient configuration
    recipients TEXT[], -- email addresses or user IDs
    notify_on_completion BOOLEAN NOT NULL DEFAULT true,
    notify_on_failure BOOLEAN NOT NULL DEFAULT true,

    -- Schedule status
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    -- active: schedule is active
    -- paused: schedule is paused
    -- disabled: schedule is disabled

    -- Timing
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    last_successful_run_at TIMESTAMPTZ,

    -- Run tracking
    total_runs INTEGER NOT NULL DEFAULT 0,
    successful_runs INTEGER NOT NULL DEFAULT 0,
    failed_runs INTEGER NOT NULL DEFAULT 0,

    -- Owner information
    created_by UUID NOT NULL,
    owned_by UUID NOT NULL,

    -- Retention policy for generated reports
    retention_days INTEGER NOT NULL DEFAULT 90,

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT report_schedules_type_check CHECK (schedule_type IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly', 'custom')),
    CONSTRAINT report_schedules_format_check CHECK (format IN ('pdf', 'xlsx', 'csv', 'html', 'json')),
    CONSTRAINT report_schedules_status_check CHECK (status IN ('active', 'paused', 'disabled'))
);

-- Indexes for report_schedules
CREATE INDEX idx_report_schedules_tenant ON report_schedules(tenant_id);
CREATE INDEX idx_report_schedules_report_id ON report_schedules(report_id);
CREATE INDEX idx_report_schedules_framework ON report_schedules(framework);
CREATE INDEX idx_report_schedules_status ON report_schedules(status);
CREATE INDEX idx_report_schedules_next_run ON report_schedules(next_run_at) WHERE status = 'active';
CREATE INDEX idx_report_schedules_owned_by ON report_schedules(owned_by);
CREATE INDEX idx_report_schedules_metadata ON report_schedules USING GIN (metadata);

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

-- Apply updated_at trigger to report_snapshots
DROP TRIGGER IF EXISTS update_report_snapshots_updated_at ON report_snapshots;
CREATE TRIGGER update_report_snapshots_updated_at
    BEFORE UPDATE ON report_snapshots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply updated_at trigger to report_generation_jobs
DROP TRIGGER IF EXISTS update_report_jobs_updated_at ON report_generation_jobs;
CREATE TRIGGER update_report_jobs_updated_at
    BEFORE UPDATE ON report_generation_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Apply updated_at trigger to report_schedules
DROP TRIGGER IF EXISTS update_report_schedules_updated_at ON report_schedules;
CREATE TRIGGER update_report_schedules_updated_at
    BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Row Level Security (RLS) for multi-tenancy
-- ============================================================================

-- Enable RLS on all new tables
ALTER TABLE report_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_generation_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;

-- Policies for report_snapshots
CREATE POLICY report_snapshots_tenant_policy ON report_snapshots
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for report_generation_jobs
CREATE POLICY report_jobs_tenant_policy ON report_generation_jobs
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- Policies for report_schedules
CREATE POLICY report_schedules_tenant_policy ON report_schedules
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::UUID);

-- ============================================================================
-- Functions for Report Management
-- ============================================================================

-- Function to update snapshot status and set file URL
CREATE OR REPLACE FUNCTION update_report_snapshot_status(
    p_snapshot_id UUID,
    p_status VARCHAR,
    p_file_url TEXT DEFAULT NULL,
    p_file_size BIGINT DEFAULT NULL,
    p_error_message TEXT DEFAULT NULL
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE report_snapshots
    SET
        status = p_status,
        file_url = COALESCE(p_file_url, file_url),
        file_size_bytes = COALESCE(p_file_size, file_size_bytes),
        error_message = COALESCE(p_error_message, error_message),
        updated_at = NOW()
    WHERE id = p_snapshot_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- Function to expire old report snapshots
CREATE OR REPLACE FUNCTION expire_old_report_snapshots() RETURNS INTEGER AS $$
DECLARE
    expired_count INTEGER;
BEGIN
    UPDATE report_snapshots
    SET
        status = 'expired',
        file_url = NULL,
        updated_at = NOW()
    WHERE status = 'completed'
      AND expires_at IS NOT NULL
      AND expires_at < NOW();

    GET DIAGNOSTICS expired_count = ROW_COUNT;
    RETURN expired_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get next scheduled reports for processing
CREATE OR REPLACE FUNCTION get_scheduled_reports(limit_count INTEGER DEFAULT 10)
RETURNS TABLE (
    schedule_id UUID,
    report_id UUID,
    tenant_id UUID,
    framework VARCHAR,
    format VARCHAR,
    options JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        rs.id,
        rs.report_id,
        rs.tenant_id,
        rs.framework,
        rs.format,
        rs.options
    FROM report_schedules rs
    WHERE rs.status = 'active'
      AND rs.next_run_at <= NOW()
    ORDER BY rs.next_run_at ASC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to update schedule after run
CREATE OR REPLACE FUNCTION update_schedule_after_run(
    p_schedule_id UUID,
    p_success BOOLEAN,
    p_next_run_at TIMESTAMPTZ
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE report_schedules
    SET
        last_run_at = NOW(),
        next_run_at = COALESCE(p_next_run_at, next_run_at),
        total_runs = total_runs + 1,
        successful_runs = successful_runs + CASE WHEN p_success THEN 1 ELSE 0 END,
        failed_runs = failed_runs + CASE WHEN NOT p_success THEN 1 ELSE 0 END,
        last_successful_run_at = CASE WHEN p_success THEN NOW() ELSE last_successful_run_at END,
        updated_at = NOW()
    WHERE id = p_schedule_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON TABLE report_snapshots IS 'Stores generated compliance and analytics report instances with file storage references';
COMMENT ON TABLE report_generation_jobs IS 'Tracks asynchronous report generation jobs with status and retry logic';
COMMENT ON TABLE report_schedules IS 'Configures recurring report generation schedules';

COMMENT ON COLUMN report_snapshots.status IS 'Generation status: pending, completed, failed, expired';
COMMENT ON COLUMN report_snapshots.file_url IS 'URL to download the generated report file';
COMMENT ON COLUMN report_snapshots.storage_path IS 'Internal storage path for the report file';
COMMENT ON COLUMN report_snapshots.expires_at IS 'When the report file should be expired and removed';

COMMENT ON COLUMN report_generation_jobs.status IS 'Job status: queued, processing, completed, failed, cancelled';
COMMENT ON COLUMN report_generation_jobs.progress IS 'Job completion progress from 0 to 100';
COMMENT ON COLUMN report_generation_jobs.retry_count IS 'Number of retry attempts made';

COMMENT ON COLUMN report_schedules.schedule_type IS 'Schedule frequency: daily, weekly, monthly, quarterly, yearly, custom';
COMMENT ON COLUMN report_schedules.cron_expression IS 'Custom cron expression for complex schedules';
COMMENT ON COLUMN report_schedules.next_run_at IS 'When the next scheduled run should occur';
COMMENT ON COLUMN report_schedules.retention_days IS 'Number of days to keep generated reports';
