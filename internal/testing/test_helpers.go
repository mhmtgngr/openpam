package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// Context returns a context with timeout for tests
func Context(t testing.TB) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// Logger returns a test logger that writes to testing.T
func Logger(t testing.TB) zerolog.Logger {
	return zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger().Level(zerolog.DebugLevel)
}

// NewTestDB creates a test database connection
// The caller is responsible for setting up the test database
func NewTestDB(t testing.TB) *sqlx.DB {
	// This should connect to a test database
	// For CI/CD, use docker-compose to spin up test dependencies
	dsn := "host=localhost port=5432 user=openpam_test password=openpam_test dbname=openpam_test sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
		return nil
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

// GenerateTestMasterKey creates a test master key for encryption
func GenerateTestMasterKey() []byte {
	// Use a fixed key for testing - DO NOT use in production
	// Must be exactly 32 bytes for AES-256
	return []byte("test-master-key-32bytes-long-ok!")
}

// GenerateTestTenantID returns a consistent UUID for testing
func GenerateTestTenantID() string {
	return "00000000-0000-0000-0000-000000000001"
}

// GenerateTestUserID returns a consistent UUID for testing
func GenerateTestUserID() string {
	return "00000000-0000-0000-0000-000000000002"
}

// GenerateTestCredentialID returns a consistent UUID for testing
func GenerateTestCredentialID() string {
	return "00000000-0000-0000-0000-000000000003"
}

// SetupTestDatabase creates and migrates a test database schema
func SetupTestDatabase(t testing.TB, db *sqlx.DB) {
	// Drop existing tables
	schemas := []string{
		"audit_events",
		"users",
		"roles",
		"permissions",
		"role_permissions",
		"user_roles",
		"credentials",
		"sessions",
		"checkout_requests",
		"tenants",
	}

	for _, table := range schemas {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}

	// Create test schema
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			domain TEXT UNIQUE,
			plan TEXT NOT NULL DEFAULT 'free',
			config BYTEA,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email TEXT NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			mfa_secret TEXT,
			backup_codes TEXT,
			last_login_at TIMESTAMP,
			failed_logins INTEGER NOT NULL DEFAULT 0,
			locked_until TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS roles (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			is_system BOOLEAN NOT NULL DEFAULT FALSE,
			inherits_from_id UUID REFERENCES roles(id),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS permissions (
			id UUID PRIMARY KEY,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			scope TEXT NOT NULL DEFAULT 'all',
			description TEXT
		);

		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id UUID NOT NULL REFERENCES roles(id),
			permission_id UUID NOT NULL REFERENCES permissions(id),
			PRIMARY KEY (role_id, permission_id)
		);

		CREATE TABLE IF NOT EXISTS user_roles (
			user_id UUID NOT NULL REFERENCES users(id),
			role_id UUID NOT NULL REFERENCES roles(id),
			assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
			assigned_by UUID NOT NULL,
			PRIMARY KEY (user_id, role_id)
		);

		CREATE TABLE IF NOT EXISTS credentials (
			id UUID PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			username TEXT NOT NULL,
			encrypted_secret BYTEA NOT NULL,
			rotation_policy TEXT NOT NULL DEFAULT 'manual',
			last_rotated_at TIMESTAMP,
			folder_id UUID,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			created_by UUID NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			credential_id UUID REFERENCES credentials(id),
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			target_host TEXT NOT NULL,
			target_port INTEGER NOT NULL,
			recording_url TEXT,
			started_at TIMESTAMP NOT NULL DEFAULT NOW(),
			ended_at TIMESTAMP,
			terminated_by UUID REFERENCES users(id),
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			metadata BYTEA
		);

		CREATE TABLE IF NOT EXISTS checkout_requests (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id),
			credential_id UUID NOT NULL REFERENCES credentials(id),
			justification TEXT,
			duration_minutes INTEGER NOT NULL,
			status TEXT NOT NULL,
			approved_by UUID REFERENCES users(id),
			checked_out_at TIMESTAMP,
			expires_at TIMESTAMP,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS audit_events (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL REFERENCES tenants(id),
			actor_id UUID NOT NULL,
			actor_type TEXT NOT NULL,
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT,
			outcome TEXT NOT NULL,
			ip TEXT,
			user_agent TEXT,
			correlation_id TEXT,
			details BYTEA,
			previous_hash TEXT,
			hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
	`)

	require.NoError(t, err)

	// Insert test tenant
	_, err = db.Exec(`INSERT INTO tenants (id, name, domain, plan) VALUES ($1, $2, $3, $4)`,
		GenerateTestTenantID(), "Test Tenant", "test.example.com", "enterprise")
	require.NoError(t, err)
}

// CleanupTestDatabase drops all test tables
func CleanupTestDatabase(t testing.TB, db *sqlx.DB) {
	tables := []string{
		"audit_events",
		"checkout_requests",
		"sessions",
		"credentials",
		"user_roles",
		"role_permissions",
		"permissions",
		"roles",
		"users",
		"tenants",
	}

	for _, table := range tables {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
	}
}

// NewMockCache creates a mock cache for testing
func NewMockCache(t testing.TB) *MockCache {
	return &MockCache{
		data: make(map[string]string),
	}
}

// MockCache is a simple in-memory cache for testing
type MockCache struct {
	data map[string]string
}

func (m *MockCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, ok := m.data[key]
	if !ok {
		return fmt.Errorf("key not found: %s", key)
	}
	// Simple string assignment - in real usage would need proper unmarshaling
	if s, ok := dest.(*string); ok {
		*s = val
	}
	return nil
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.data[key] = fmt.Sprintf("%v", value)
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockCache) DeleteByPattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCache) Exists(ctx context.Context, key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *MockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *MockCache) Close() error {
	return nil
}

func (m *MockCache) Health(ctx context.Context) error {
	return nil
}

// Increment increments a counter and returns the new value
func (m *MockCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	currentVal, ok := m.data[key]
	if !ok {
		m.data[key] = fmt.Sprintf("%d", delta)
		return delta, nil
	}
	var current int64
	_, err := fmt.Sscanf(currentVal, "%d", &current)
	if err != nil {
		return 0, fmt.Errorf("invalid counter value: %w", err)
	}
	newVal := current + delta
	m.data[key] = fmt.Sprintf("%d", newVal)
	return newVal, nil
}
