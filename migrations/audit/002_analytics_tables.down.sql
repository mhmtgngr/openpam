-- Down migration for analytics tables
-- This removes all analytics-related objects in reverse order of creation

-- Drop triggers first
DROP TRIGGER IF EXISTS update_command_blacklist_updated_at ON command_blacklist;
DROP TRIGGER IF EXISTS update_ssh_key_analytics_updated_at ON ssh_key_analytics;
DROP TRIGGER IF EXISTS update_ransomware_events_updated_at ON ransomware_events;
DROP TRIGGER IF EXISTS update_anomaly_detections_updated_at ON anomaly_detections;
DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_user_risk_summary CASCADE;
DROP MATERIALIZED VIEW IF EXISTS mv_session_summary CASCADE;

-- Drop RLS policies
DROP POLICY IF EXISTS blacklist_tenant_policy ON command_blacklist;
DROP POLICY IF EXISTS ssh_key_tenant_policy ON ssh_key_analytics;
DROP POLICY IF EXISTS ransomware_tenant_policy ON ransomware_events;
DROP POLICY IF EXISTS anomaly_tenant_policy ON anomaly_detections;
DROP POLICY IF EXISTS compliance_exc_tenant_policy ON compliance_exceptions;
DROP POLICY IF EXISTS compliance_eval_tenant_policy ON compliance_control_evaluations;
DROP POLICY IF EXISTS compliance_reports_tenant_policy ON compliance_reports;

-- Disable RLS
ALTER TABLE command_blacklist DISABLE ROW LEVEL SECURITY;
ALTER TABLE ssh_key_analytics DISABLE ROW LEVEL SECURITY;
ALTER TABLE ransomware_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_detections DISABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_exceptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_control_evaluations DISABLE ROW LEVEL SECURITY;
ALTER TABLE compliance_reports DISABLE ROW LEVEL SECURITY;

-- Drop tables (in order of dependencies)
DROP TABLE IF EXISTS command_blacklist CASCADE;
DROP TABLE IF EXISTS ssh_key_analytics CASCADE;
DROP TABLE IF EXISTS ransomware_events CASCADE;
DROP TABLE IF EXISTS anomaly_detections CASCADE;
DROP TABLE IF EXISTS compliance_exceptions CASCADE;
DROP TABLE IF EXISTS compliance_control_evaluations CASCADE;
DROP TABLE IF EXISTS compliance_reports CASCADE;

-- Drop helper function
DROP FUNCTION IF EXISTS update_updated_at_column();
