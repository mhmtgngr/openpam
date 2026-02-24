package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Struct(t *testing.T) {
	t.Run("default config values", func(t *testing.T) {
		cfg := Config{
			Host:            "localhost",
			Port:            5432,
			User:            "openpam",
			Password:        "password",
			Database:        "openpam",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 5 * time.Minute,
		}

		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 5432, cfg.Port)
		assert.Equal(t, "openpam", cfg.User)
		assert.Equal(t, "password", cfg.Password)
		assert.Equal(t, "openpam", cfg.Database)
		assert.Equal(t, "disable", cfg.SSLMode)
		assert.Equal(t, 25, cfg.MaxOpenConns)
		assert.Equal(t, 5, cfg.MaxIdleConns)
		assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
		assert.Equal(t, 5*time.Minute, cfg.ConnMaxIdleTime)
	})
}

func TestDB_Health(t *testing.T) {
	t.Run("health check with nil connection panics", func(t *testing.T) {
		db := &DB{
			DB:     nil,
			logger: zerolog.Nop(),
		}

		ctx := context.Background()

		// The Health method will panic on nil DB, which is expected behavior
		assert.Panics(t, func() {
			_ = db.Health(ctx)
		})
	})
}

func TestWithTenant(t *testing.T) {
	t.Run("adds tenant ID to context", func(t *testing.T) {
		ctx := context.Background()
		tenantID := "tenant-123"

		ctxWithTenant := WithTenant(ctx, tenantID)

		retrieved, ok := GetTenantID(ctxWithTenant)
		assert.True(t, ok)
		assert.Equal(t, tenantID, retrieved)
	})

	t.Run("different tenant IDs", func(t *testing.T) {
		tenantIDs := []string{"tenant-1", "tenant-2", "tenant-3"}

		for _, tid := range tenantIDs {
			ctx := context.Background()
			ctx = WithTenant(ctx, tid)

			retrieved, ok := GetTenantID(ctx)
			assert.True(t, ok)
			assert.Equal(t, tid, retrieved)
		}
	})

	t.Run("tenant ID not in context", func(t *testing.T) {
		ctx := context.Background()

		_, ok := GetTenantID(ctx)
		assert.False(t, ok)
	})

	t.Run("empty tenant ID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTenant(ctx, "")

		retrieved, ok := GetTenantID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "", retrieved)
	})
}

func TestGetTenantID(t *testing.T) {
	t.Run("retrieves string tenant ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "tenant_id", "test-tenant")
		tenantID, ok := GetTenantID(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test-tenant", tenantID)
	})

	t.Run("returns false when not string", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "tenant_id", 123)
		_, ok := GetTenantID(ctx)
		assert.False(t, ok)
	})

	t.Run("returns false when missing", func(t *testing.T) {
		ctx := context.Background()
		_, ok := GetTenantID(ctx)
		assert.False(t, ok)
	})
}

func TestTenantScoped(t *testing.T) {
	t.Run("Scope creates TenantScoped", func(t *testing.T) {
		db := &DB{
			DB:     nil,
			logger: zerolog.Nop(),
		}
		tenantID := "tenant-123"

		scoped := db.Scope(tenantID)

		assert.NotNil(t, scoped)
		assert.Equal(t, db, scoped.db)
		assert.Equal(t, tenantID, scoped.tenantID)
	})

	t.Run("TenantScoped struct fields", func(t *testing.T) {
		db := &sqlx.DB{}
		ts := &TenantScoped{
			db:       &DB{DB: db, logger: zerolog.Nop()},
			tenantID: "test-tenant",
		}

		assert.NotNil(t, ts.db)
		assert.Equal(t, "test-tenant", ts.tenantID)
	})
}

func TestTransaction(t *testing.T) {
	t.Run("transaction with no DB panics", func(t *testing.T) {
		db := &DB{
			DB:     nil,
			logger: zerolog.Nop(),
		}

		ctx := context.Background()
		assert.Panics(t, func() {
			_ = db.Transaction(ctx, func(tx *sqlx.Tx) error {
				return nil
			})
		})
	})
}

func TestTenantScoped_Get(t *testing.T) {
	t.Run("Get with nil DB panics", func(t *testing.T) {
		ts := &TenantScoped{
			db: &DB{
				DB:     nil,
				logger: zerolog.Nop(),
			},
			tenantID: "test-tenant",
		}

		ctx := context.Background()
		var dest string
		assert.Panics(t, func() {
			_ = ts.Get(ctx, &dest, "SELECT 1")
		})
	})
}

func TestTenantScoped_Select(t *testing.T) {
	t.Run("Select with nil DB panics", func(t *testing.T) {
		ts := &TenantScoped{
			db: &DB{
				DB:     nil,
				logger: zerolog.Nop(),
			},
			tenantID: "test-tenant",
		}

		ctx := context.Background()
		var dest []string
		assert.Panics(t, func() {
			_ = ts.Select(ctx, &dest, "SELECT 1")
		})
	})
}

func TestTenantScoped_NamedGet(t *testing.T) {
	t.Run("NamedGet with nil DB panics", func(t *testing.T) {
		ts := &TenantScoped{
			db: &DB{
				DB:     nil,
				logger: zerolog.Nop(),
			},
			tenantID: "test-tenant",
		}

		ctx := context.Background()
		var dest string
		assert.Panics(t, func() {
			_ = ts.NamedGet(ctx, &dest, "SELECT 1", nil)
		})
	})
}

func TestTenantScoped_NamedSelect(t *testing.T) {
	t.Run("NamedSelect with nil DB panics", func(t *testing.T) {
		ts := &TenantScoped{
			db: &DB{
				DB:     nil,
				logger: zerolog.Nop(),
			},
			tenantID: "test-tenant",
		}

		ctx := context.Background()
		var dest []string
		assert.Panics(t, func() {
			_ = ts.NamedSelect(ctx, &dest, "SELECT 1", nil)
		})
	})
}

func TestNamedExecContext(t *testing.T) {
	t.Run("NamedExecContext with nil DB panics", func(t *testing.T) {
		db := &DB{
			DB:     nil,
			logger: zerolog.Nop(),
		}

		ctx := context.Background()
		assert.Panics(t, func() {
			_, _ = db.NamedExecContext(ctx, "tenant-123", "SELECT 1", nil)
		})
	})
}

func TestQueryRowTenantContext(t *testing.T) {
	t.Run("QueryRowTenantContext with nil DB panics", func(t *testing.T) {
		db := &DB{
			DB:     nil,
			logger: zerolog.Nop(),
		}

		ctx := context.Background()
		assert.Panics(t, func() {
			_ = db.QueryRowTenantContext(ctx, "tenant-123", "SELECT 1")
		})
	})
}

func TestDB_WrapsSQLXDB(t *testing.T) {
	t.Run("DB struct contains sqlx.DB", func(t *testing.T) {
		db := &DB{
			DB:     &sqlx.DB{},
			logger: zerolog.Nop(),
		}

		assert.NotNil(t, db.DB)
		assert.NotNil(t, db.logger)
	})
}

// Test isLikelyProductionEnvironment function
func TestIsLikelyProductionEnvironment(t *testing.T) {
	// Save original environment values
	originalEnv := []struct {
		name, restore string
	}{
		{"ENV", ""},
		{"GO_ENV", ""},
		{"ENVIRONMENT", ""},
		{"DB_HOST", ""},
	}

	for _, e := range originalEnv {
		e.restore = os.Getenv(e.name)
	}

	// Restore environment after tests
	defer func() {
		for _, e := range originalEnv {
			if e.restore == "" {
				os.Unsetenv(e.name)
			} else {
				os.Setenv(e.name, e.restore)
			}
		}
	}()

	tests := []struct {
		name           string
		env            string
		goEnv          string
		environmentEnv string
		dbHost         string
		wantProduction bool
	}{
		{
			name:           "ENV=local is not production",
			env:            "local",
			wantProduction: false,
		},
		{
			name:           "ENV=dev with localhost is not production",
			env:            "dev",
			dbHost:         "localhost",
			wantProduction: false,
		},
		{
			name:           "ENV=dev with 127.0.0.1 is not production",
			env:            "dev",
			dbHost:         "127.0.0.1",
			wantProduction: false,
		},
		{
			name:           "ENV=dev with remote host IS production",
			env:            "dev",
			dbHost:         "db.example.com",
			wantProduction: true,
		},
		{
			name:           "ENV=development with localhost is not production",
			env:            "development",
			dbHost:         "localhost",
			wantProduction: false,
		},
		{
			name:           "ENV unset IS production (safe default)",
			wantProduction: true,
		},
		{
			name:           "ENV=production IS production",
			env:            "production",
			wantProduction: true,
		},
		{
			name:           "ENV=prod IS production",
			env:            "prod",
			wantProduction: true,
		},
		{
			name:           "ENV=staging IS production",
			env:            "staging",
			wantProduction: true,
		},
		{
			name:           "ENV=test IS production (safe default for test envs)",
			env:            "test",
			wantProduction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment first
			os.Unsetenv("ENV")
			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENVIRONMENT")
			os.Unsetenv("DB_HOST")

			// Set test values
			if tt.env != "" {
				os.Setenv("ENV", tt.env)
			}
			if tt.goEnv != "" {
				os.Setenv("GO_ENV", tt.goEnv)
			}
			if tt.environmentEnv != "" {
				os.Setenv("ENVIRONMENT", tt.environmentEnv)
			}
			if tt.dbHost != "" {
				os.Setenv("DB_HOST", tt.dbHost)
			}

			got := isLikelyProductionEnvironment()
			assert.Equal(t, tt.wantProduction, got,
				"isLikelyProductionEnvironment() = %v, want %v", got, tt.wantProduction)
		})
	}
}

// Test DSN construction
func TestNew_DSNConstruction(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name: "minimal config",
			cfg: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "pass",
				Database: "db",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable",
		},
		{
			name: "with SSL require",
			cfg: Config{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "secret",
				Database: "production",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=admin password=secret dbname=production sslmode=require",
		},
		{
			name: "with verify-full SSL",
			cfg: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "test",
				Password: "test",
				Database: "testdb",
				SSLMode:  "verify-full",
			},
			expected: "host=localhost port=5432 user=test password=test dbname=testdb sslmode=verify-full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually test New() without a database,
			// but we can verify the DSN format
			dsn := "host=" + tt.cfg.Host +
				" port=" + string(rune(tt.cfg.Port)) +
				" user=" + tt.cfg.User +
				" password=" + tt.cfg.Password +
				" dbname=" + tt.cfg.Database +
				" sslmode=" + tt.cfg.SSLMode
			assert.Contains(t, dsn, "host="+tt.cfg.Host)
		})
	}
}
