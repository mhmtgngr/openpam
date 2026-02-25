-- Rollback Report Persistence Migration
-- This migration removes the report persistence tables and related objects

-- Drop functions
DROP FUNCTION IF EXISTS update_schedule_after_run(UUID, BOOLEAN, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS get_scheduled_reports(INTEGER);
DROP FUNCTION IF EXISTS expire_old_report_snapshots();
DROP FUNCTION IF EXISTS update_report_snapshot_status(UUID, VARCHAR, TEXT, BIGINT, TEXT);

-- Drop triggers
DROP TRIGGER IF EXISTS update_report_schedules_updated_at ON report_schedules;
DROP TRIGGER IF EXISTS update_report_jobs_updated_at ON report_generation_jobs;
DROP TRIGGER IF EXISTS update_report_snapshots_updated_at ON report_snapshots;

-- Drop policies
DROP POLICY IF EXISTS report_schedules_tenant_policy ON report_schedules;
DROP POLICY IF EXISTS report_jobs_tenant_policy ON report_generation_jobs;
DROP POLICY IF EXISTS report_snapshots_tenant_policy ON report_snapshots;

-- Disable RLS
ALTER TABLE report_schedules DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_generation_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE report_snapshots DISABLE ROW LEVEL SECURITY;

-- Drop indexes (they will be dropped with tables, but being explicit)
DROP INDEX IF EXISTS idx_report_schedules_metadata;
DROP INDEX IF EXISTS idx_report_schedules_owned_by;
DROP INDEX IF EXISTS idx_report_schedules_next_run;
DROP INDEX IF EXISTS idx_report_schedules_status;
DROP INDEX IF EXISTS idx_report_schedules_framework;
DROP INDEX IF EXISTS idx_report_schedules_report_id;
DROP INDEX IF EXISTS idx_report_schedules_tenant;

DROP INDEX IF EXISTS idx_report_jobs_metadata;
DROP INDEX IF EXISTS idx_report_jobs_worker_id;
DROP INDEX IF EXISTS idx_report_jobs_queued_at;
DROP INDEX IF EXISTS idx_report_jobs_status;
DROP INDEX IF EXISTS idx_report_jobs_report_id;
DROP INDEX IF EXISTS idx_report_jobs_snapshot_id;
DROP INDEX IF EXISTS idx_report_jobs_tenant;

DROP INDEX IF EXISTS idx_report_snapshots_completed_tenant;
DROP INDEX IF EXISTS idx_report_snapshots_metadata;
DROP INDEX IF EXISTS idx_report_snapshots_expires_at;
DROP INDEX IF EXISTS idx_report_snapshots_period;
DROP INDEX IF EXISTS idx_report_snapshots_generated_at;
DROP INDEX IF EXISTS idx_report_snapshots_status;
DROP INDEX IF EXISTS idx_report_snapshots_framework;
DROP INDEX IF EXISTS idx_report_snapshots_report_id;
DROP INDEX IF EXISTS idx_report_snapshots_tenant;

-- Drop tables
DROP TABLE IF EXISTS report_schedules CASCADE;
DROP TABLE IF EXISTS report_generation_jobs CASCADE;
DROP TABLE IF EXISTS report_snapshots CASCADE;
