-- Rollback secure DEK storage
DROP INDEX IF EXISTS idx_recording_keys_deleted;
DROP INDEX IF EXISTS idx_recording_keys_object;
DROP INDEX IF EXISTS idx_recording_keys_session;
DROP TABLE IF EXISTS recording_keys;
