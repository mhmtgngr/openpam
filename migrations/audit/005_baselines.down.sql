-- Rollback script for baselines migration

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_session_metrics_summary CASCADE;
DROP MATERIALIZED VIEW IF EXISTS mv_user_baseline_summary CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS refresh_analytics_views CASCADE;
DROP FUNCTION IF EXISTS calculate_session_failure_rate CASCADE;
DROP FUNCTION IF EXISTS is_off_hours CASCADE;

-- Drop tables
DROP TABLE IF EXISTS daily_user_metrics CASCADE;
DROP TABLE IF EXISTS session_analytics CASCADE;
DROP TABLE IF EXISTS user_baselines CASCADE;
