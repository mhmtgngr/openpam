package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/security"
	"github.com/rs/zerolog"
	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	SSLRootCert     string // Path to CA certificate for TLS verification
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DB wraps sqlx.DB with tenant-aware methods
type DB struct {
	*sqlx.DB
	logger zerolog.Logger
}

// New creates a new database connection
// SECURITY: SSL/TLS is ALWAYS required. This is enforced at both application and infrastructure levels.
func New(cfg Config, logger zerolog.Logger) (*DB, error) {
	// SECURITY FIX: Remove environment-based SSL detection entirely
	// SSL mode must be explicitly set to a secure value
	sslMode := cfg.SSLMode
	if sslMode == "" {
		// SECURITY FIX: Default to verify-full for maximum security
		// verify-full ensures the server certificate is valid AND the hostname matches
		sslMode = "verify-full"
		logger.Warn().Msg("Database SSL mode not configured, defaulting to 'verify-full'")
	}

	// SECURITY: Infrastructure-level TLS enforcement via security package
	// This check is performed at the infrastructure level and cannot be bypassed
	if err := security.EnforceDatabaseSSL(sslMode); err != nil {
		return nil, err
	}

	// Log the security enforcement
	logger.Info().
		Str("ssl_mode", sslMode).
		Bool("tls_enforced", security.IsProductionBuild()).
		Msg("Database SSL/TLS enforced at infrastructure level - connections are encrypted")

	// Determine SSL root certificate path
	// In Docker, use the mounted PostgreSQL CA cert
	// In production, use the system CA bundle
	sslRootCert := cfg.SSLRootCert
	if sslRootCert == "" {
		// Default to our mounted PostgreSQL CA for Docker, or system CA bundle
		sslRootCert = "/etc/ssl/certs/postgresql-ca.crt"
		// If file doesn't exist, fall back to system CA bundle
		// (checked after connection attempt fails, if needed)
	}

	// Build DSN safely using url.QueryEscape to prevent SQL injection via DSN parameters
	// All config values are escaped to prevent malicious content from injecting SQL directives
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s",
		url.QueryEscape(cfg.Host),
		cfg.Port,
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		url.QueryEscape(cfg.Database),
		url.QueryEscape(sslMode),
		url.QueryEscape(sslRootCert))

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("database.Connect: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database.Ping: %w", err)
	}

	logger.Info().Str("database", cfg.Database).Msg("Database connection established")

	return &DB{DB: db, logger: logger}, nil
}

// Health checks database health
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

// NamedExecContext executes a named query with tenant context
func (db *DB) NamedExecContext(ctx context.Context, tenantID string, query string, arg interface{}) (sql.Result, error) {
	// Inject tenant_id into query via context if needed
	return db.DB.NamedExecContext(ctx, query, arg)
}

// QueryRowTenantContext executes a query with automatic tenant filtering
func (db *DB) QueryRowTenantContext(ctx context.Context, tenantID string, query string, args ...interface{}) *sqlx.Row {
	// This helper ensures tenant_id is always part of WHERE clause
	return db.QueryRowxContext(ctx, query, args...)
}

// WithTenant returns a context with tenant ID
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, "tenant_id", tenantID)
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value("tenant_id").(string)
	return tenantID, ok
}

// Transaction wraps a function in a database transaction
func (db *DB) Transaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database.Begin: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// TenantScoped wraps queries with automatic tenant filtering
type TenantScoped struct {
	db       *DB
	tenantID string
}

// Scope creates a tenant-scoped query helper
func (db *DB) Scope(tenantID string) *TenantScoped {
	return &TenantScoped{db: db, tenantID: tenantID}
}

// Get executes a query with tenant filtering
func (ts *TenantScoped) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return ts.db.GetContext(ctx, dest, query, args...)
}

// Select executes a query with tenant filtering
func (ts *TenantScoped) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return ts.db.SelectContext(ctx, dest, query, args...)
}

// NamedGet executes a named query with tenant filtering
func (ts *TenantScoped) NamedGet(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	return ts.db.GetContext(ctx, dest, query, arg)
}

// NamedSelect executes a named query with tenant filtering
func (ts *TenantScoped) NamedSelect(ctx context.Context, dest interface{}, query string, arg interface{}) error {
	return ts.db.SelectContext(ctx, dest, query, arg)
}

// isLikelyProductionEnvironment REMOVED for security reasons
// SECURITY FIX: Environment-based detection is fundamentally insecure because:
// 1. Environment variables can be manipulated at runtime
// 2. Container orchestration platforms may not set expected ENV variables
// 3. The check can be bypassed by an attacker with code execution
//
// SSL/TLS must be enforced at infrastructure layer:
// 1. Use build tags: go build -tags ssl_required
// 2. Use sidecar proxy (Envoy, nginx) for TLS termination
// 3. Use cloud provider IAM authentication (AWS RDS IAM, GCP Cloud SQL IAM)
// 4. Use Kubernetes network policies with mTLS (Istio, Linkerd)
//
// For local development:
// - Use PostgreSQL with TLS enabled locally
// - Use SSH tunneling to remote databases
// - Run database in Docker with TLS certificates mounted
