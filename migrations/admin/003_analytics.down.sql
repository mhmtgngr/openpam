-- 003_analytics.down.sql
-- Rollback analytics and reporting migration

-- Drop default records
DELETE FROM analytics_reports WHERE id LIKE '00000000-0000-0000-0003-%';
DELETE FROM analytics_alerts WHERE id LIKE '00000000-0000-0000-0002-%';
DELETE FROM analytics_widgets WHERE dashboard_id = '00000000-0000-0000-0001-000000000001';
DELETE FROM analytics_dashboards WHERE id = '00000000-0000-0000-0001-000000000001';

-- Drop tables
DROP TABLE IF EXISTS analytics_refresh_log;
DROP TABLE IF EXISTS analytics_risk_scores;
DROP TABLE IF EXISTS analytics_user_activity;
DROP TABLE IF EXISTS analytics_metrics;
DROP TABLE IF EXISTS analytics_alert_triggers;
DROP TABLE IF EXISTS analytics_alerts;
DROP TABLE IF EXISTS analytics_widgets;
DROP TABLE IF EXISTS analytics_dashboards;
DROP TABLE IF EXISTS analytics_report_snapshots;
DROP TABLE IF EXISTS analytics_reports;
DROP TABLE IF EXISTS analytics_events;
DROP TABLE IF EXISTS analytics_sessions;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_analytics_updated_at;
