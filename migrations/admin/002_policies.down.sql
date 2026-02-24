-- 002_policies.down.sql
-- Rollback Policy Engine Migration

-- Drop triggers
DROP TRIGGER IF EXISTS policies_updated_at ON policies;
DROP TRIGGER IF EXISTS approval_requests_updated_at ON approval_requests;
DROP TRIGGER IF EXISTS policy_templates_updated_at ON policy_templates;

-- Drop function
DROP FUNCTION IF EXISTS update_policy_updated_at();

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS command_filter_patterns CASCADE;
DROP TABLE IF EXISTS policy_templates CASCADE;
DROP TABLE IF EXISTS policy_evaluations CASCADE;
DROP TABLE IF EXISTS approval_requests CASCADE;
DROP TABLE IF EXISTS policy_assignments CASCADE;
DROP TABLE IF EXISTS policies CASCADE;

-- Drop custom types
DROP TYPE IF EXISTS approval_status CASCADE;
DROP TYPE IF EXISTS rule_type CASCADE;
DROP TYPE IF EXISTS policy_effect CASCADE;
DROP TYPE IF EXISTS policy_type CASCADE;
