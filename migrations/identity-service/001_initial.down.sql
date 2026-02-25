-- Identity Service Database Schema Rollback
-- Migration 001: Rollback initial schema

DROP TABLE IF EXISTS access_requests CASCADE;
DROP TABLE IF EXISTS workflows CASCADE;
DROP TABLE IF EXISTS break_glass_requests CASCADE;

DROP FUNCTION IF EXISTS update_identity_updated_at_column() CASCADE;
DROP FUNCTION IF EXISTS get_expiring_access_requests(UUID, INTEGER) CASCADE;
