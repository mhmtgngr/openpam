-- Rollback Analytics Service Schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_session_analytics_updated_at ON session_analytics;
DROP TRIGGER IF EXISTS update_user_activity_updated_at ON user_activity;
DROP TRIGGER IF EXISTS update_compliance_exceptions_updated_at ON compliance_exceptions;
DROP TRIGGER IF EXISTS update_anomaly_detections_updated_at ON anomaly_detections;
DROP TRIGGER IF EXISTS update_ransomware_events_updated_at ON ransomware_events;
DROP TRIGGER IF EXISTS update_ssh_key_analytics_updated_at ON ssh_key_analytics;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS create_monthly_partition();
DROP FUNCTION IF EXISTS refresh_analytics_views();

-- Drop materialized views
DROP MATERIALIZED VIEW IF EXISTS mv_user_risk_summary;
DROP MATERIALIZED VIEW IF EXISTS mv_session_summary;

-- Drop tables (in order of dependencies)
DROP TABLE IF EXISTS ransomware_events;
DROP TABLE IF EXISTS anomaly_detections;
DROP TABLE IF EXISTS ssh_key_analytics;
DROP TABLE IF EXISTS command_blacklist;
DROP TABLE IF EXISTS compliance_exceptions;
DROP TABLE IF EXISTS compliance_control_evaluations;
DROP TABLE IF EXISTS compliance_reports;
DROP TABLE IF EXISTS command_frequency_2025_12;
DROP TABLE IF EXISTS command_frequency_2025_11;
DROP TABLE IF EXISTS command_frequency_2025_10;
DROP TABLE IF EXISTS command_frequency_2025_09;
DROP TABLE IF EXISTS command_frequency_2025_08;
DROP TABLE IF EXISTS command_frequency_2025_07;
DROP TABLE IF EXISTS command_frequency_2025_06;
DROP TABLE IF EXISTS command_frequency_2025_05;
DROP TABLE IF EXISTS command_frequency_2025_04;
DROP TABLE IF EXISTS command_frequency_2025_03;
DROP TABLE IF EXISTS command_frequency_2025_02;
DROP TABLE IF EXISTS command_frequency_2025_01;
DROP TABLE IF EXISTS command_frequency;
DROP TABLE IF EXISTS user_activity;
DROP TABLE IF EXISTS session_analytics;
