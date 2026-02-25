-- Rollback Anomaly Deduplication Migration

-- Drop views
DROP VIEW IF EXISTS anomaly_statistics;
DROP VIEW IF EXISTS active_anomalies;

-- Drop functions
DROP FUNCTION IF EXISTS merge_duplicate_anomalies(UUID);
DROP FUNCTION IF EXISTS mark_anomaly_duplicates();
DROP FUNCTION IF EXISTS set_anomaly_correlation_key();
DROP FUNCTION IF EXISTS generate_anomaly_correlation_key(UUID, VARCHAR, UUID, VARCHAR, JSONB);

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_mark_anomaly_duplicates ON anomaly_detections;
DROP TRIGGER IF EXISTS trigger_set_anomaly_correlation_key ON anomaly_detections;

-- Drop indexes
DROP INDEX IF EXISTS idx_anomaly_merged_into;
DROP INDEX IF EXISTS idx_anomaly_first_detection;
DROP INDEX IF EXISTS idx_anomaly_duplicates;
DROP INDEX IF EXISTS idx_anomaly_correlation_id;
DROP INDEX IF EXISTS idx_anomaly_correlation_key;

-- Drop columns
ALTER TABLE anomaly_detections
DROP COLUMN IF EXISTS merged_into_id,
DROP COLUMN IF EXISTS first_detection_id,
DROP COLUMN IF EXISTS is_duplicate,
DROP COLUMN IF EXISTS duplicate_count,
DROP COLUMN IF EXISTS correlation_key,
DROP COLUMN IF EXISTS correlation_id;
