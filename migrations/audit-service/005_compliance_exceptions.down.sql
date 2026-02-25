-- Rollback compliance exceptions migration

-- Drop functions
DROP FUNCTION IF EXISTS expire_old_exceptions();
DROP FUNCTION IF EXISTS update_exception_notification(UUID);
DROP FUNCTION IF EXISTS get_expiring_exceptions(INTEGER);
DROP FUNCTION IF EXISTS get_control_exceptions(UUID, VARCHAR, VARCHAR);
DROP FUNCTION IF EXISTS get_exception_stats(UUID);
DROP FUNCTION IF EXISTS update_exception_status(UUID, VARCHAR, UUID, TIMESTAMPTZ);

-- Drop trigger
DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;

-- Drop policy
DROP POLICY IF EXISTS compliance_exceptions_tenant_policy ON compliance_exceptions;

-- Disable RLS
ALTER TABLE compliance_exceptions DISABLE ROW LEVEL SECURITY;

-- Drop indexes
DROP INDEX IF EXISTS idx_compliance_exceptions_expiring_soon;
DROP INDEX IF EXISTS idx_compliance_exceptions_pending;
DROP INDEX IF EXISTS idx_compliance_exceptions_active_tenant;
DROP INDEX IF EXISTS idx_compliance_exceptions_metadata;
DROP INDEX IF EXISTS idx_compliance_exceptions_approved_by;
DROP INDEX IF EXISTS idx_compliance_exceptions_requested_by;
DROP INDEX IF EXISTS idx_compliance_exceptions_requested_at;
DROP INDEX IF EXISTS idx_compliance_exceptions_expires_at;
DROP INDEX IF EXISTS idx_compliance_exceptions_risk_level;
DROP INDEX IF EXISTS idx_compliance_exceptions_status;
DROP INDEX IF EXISTS idx_compliance_exceptions_framework;
DROP INDEX IF EXISTS idx_compliance_exceptions_control;
DROP INDEX IF EXISTS idx_compliance_exceptions_tenant;

-- Drop table
DROP TABLE IF EXISTS compliance_exceptions;
