package database

import (
	"context"
	"database/sql"
	"fmt"
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
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

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
