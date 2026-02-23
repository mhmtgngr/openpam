-- OpenPAM Database Schema Rollback
-- Drops all tables in reverse order of creation

DROP TRIGGER IF EXISTS update_recordings_updated_at ON recordings;
DROP TRIGGER IF EXISTS update_discovered_assets_updated_at ON discovered_assets;
DROP TRIGGER IF EXISTS update_discovery_scans_updated_at ON discovery_scans;
DROP TRIGGER IF EXISTS update_approval_groups_updated_at ON approval_groups;
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
DROP TRIGGER IF EXISTS update_checkouts_updated_at ON checkouts;
DROP TRIGGER IF EXISTS update_approval_requests_updated_at ON approval_requests;
DROP TRIGGER IF EXISTS update_secrets_updated_at ON secrets;
DROP TRIGGER IF EXISTS update_targets_updated_at ON targets;
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS credential_history;
DROP TABLE IF EXISTS rotation_tasks;
DROP TABLE IF EXISTS discovered_assets;
DROP TABLE IF EXISTS discovery_scans;
DROP TABLE IF EXISTS recording_events;
DROP TABLE IF EXISTS recordings;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS checkout_records;
DROP TABLE IF EXISTS checkouts;
DROP TABLE IF EXISTS approval_groups;
DROP TABLE IF EXISTS approvals;
DROP TABLE IF EXISTS approval_requests;
DROP TABLE IF EXISTS encrypted_deks;
DROP TABLE IF EXISTS key_metadata;
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS targets;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
