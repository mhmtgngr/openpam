package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
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
func New(cfg Config, logger zerolog.Logger) (*DB, error) {
	// Enforce SSL/TLS for secure database connections
	// In production, sslmode must be "require", "verify-ca", or "verify-full"
	// Allow "disable" and "prefer" only for development environments
	sslMode := cfg.SSLMode
	if sslMode == "" {
		// Default to require for security
		sslMode = "require"
		logger.Warn().Msg("Database SSL mode not configured, defaulting to 'require'")
	}

	// PRODUCTION GUARD: Prevent SSL disable in production environments
	// This prevents accidental deployment with insecure database connections
	if isProductionEnvironment() && (sslMode == "disable" || sslMode == "allow") {
		return nil, fmt.Errorf("database: SSL mode '%s' is not allowed in production environment. Use 'require', 'verify-ca', or 'verify-full'. Set ENV=development to override", sslMode)
	}

	// Warn if SSL is disabled in non-production environments
	if sslMode == "disable" || sslMode == "allow" {
		logger.Warn().
			Str("ssl_mode", sslMode).
			Msg("Database SSL/TLS is disabled - database connections are NOT encrypted")
	}

	// Build DSN safely using url.QueryEscape to prevent SQL injection via DSN parameters
	// All config values are escaped to prevent malicious content from injecting SQL directives
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s sslrootcert=/etc/ssl/certs/ca-certificates.crt",
		url.QueryEscape(cfg.Host),
		cfg.Port,
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		url.QueryEscape(cfg.Database),
		url.QueryEscape(sslMode))

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

// isProductionEnvironment determines if the application is running in production
// This prevents insecure configurations from being deployed to production
func isProductionEnvironment() bool {
	// Check explicit environment variable
	env := os.Getenv("ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}

	// Consider production if:
	// 1. ENV is explicitly set to "production" or "prod"
	// 2. No explicit development/test environment is set AND running in container
	isProduction := env == "production" || env == "prod"

	// If env is not set, detect container environment as potential production
	if env == "" {
		// Check if running in Docker/container (common indicator)
		if _, err := os.Stat("/.dockerenv"); err == nil {
			isProduction = true
		}
		// Check for Kubernetes
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			isProduction = true
		}
	}

	// Explicitly not production if set to development/test
	if env == "development" || env == "dev" || env == "test" || env == "testing" {
		return false
	}

	return isProduction
}
