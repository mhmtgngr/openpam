package testing

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	t.Run("creates context with timeout", func(t *testing.T) {
		ctx := Context(t)

		assert.NotNil(t, ctx)
		// Context should have a deadline
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("context is cancelled on test completion", func(t *testing.T) {
		ctx := Context(t)
		done := ctx.Done()

		// Context should have a Done channel that is not nil
		assert.NotNil(t, done)

		// Context should still be valid during test (channel not closed yet)
		select {
		case <-done:
			t.Error("Context should not be cancelled during test")
		default:
			// Expected - context is still valid
		}

		// After test ends, context would be cancelled by Cleanup
		// We can't test this directly without the test ending
	})
}

func TestLogger(t *testing.T) {
	t.Run("creates test logger", func(t *testing.T) {
		logger := Logger(t)

		assert.NotNil(t, logger)
		// Logger should be at debug level
		assert.Equal(t, zerolog.DebugLevel, logger.GetLevel())
	})

	t.Run("logger writes to test", func(t *testing.T) {
		logger := Logger(t)

		// Should not panic when writing
		logger.Info().Msg("test message")
		logger.Debug().Msg("debug message")
		logger.Warn().Msg("warning message")
		logger.Error().Msg("error message")
	})
}

func TestGenerateTestMasterKey(t *testing.T) {
	t.Run("generates consistent master key", func(t *testing.T) {
		key1 := GenerateTestMasterKey()
		key2 := GenerateTestMasterKey()

		assert.NotNil(t, key1)
		assert.NotNil(t, key2)
		assert.Equal(t, key1, key2, "master key should be consistent for testing")
		assert.Len(t, key1, 32, "master key should be 32 bytes for AES-256")
	})
}

func TestGenerateTestTenantID(t *testing.T) {
	t.Run("generates consistent tenant ID", func(t *testing.T) {
		id1 := GenerateTestTenantID()
		id2 := GenerateTestTenantID()

		assert.Equal(t, id1, id2)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", id1)
	})
}

func TestGenerateTestUserID(t *testing.T) {
	t.Run("generates consistent user ID", func(t *testing.T) {
		id1 := GenerateTestUserID()
		id2 := GenerateTestUserID()

		assert.Equal(t, id1, id2)
		assert.Equal(t, "00000000-0000-0000-0000-000000000002", id1)
	})
}

func TestGenerateTestCredentialID(t *testing.T) {
	t.Run("generates consistent credential ID", func(t *testing.T) {
		id1 := GenerateTestCredentialID()
		id2 := GenerateTestCredentialID()

		assert.Equal(t, id1, id2)
		assert.Equal(t, "00000000-0000-0000-0000-000000000003", id1)
	})
}

func TestMockCache_Get(t *testing.T) {
	t.Run("get returns error for missing key", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()
		var dest string

		err := cache.Get(ctx, "nonexistent", &dest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key not found")
	})

	t.Run("get retrieves stored value", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		// Store a value using the mock
		cache.data["test-key"] = "test-value"

		var dest string
		err := cache.Get(ctx, "test-key", &dest)

		assert.NoError(t, err)
		assert.Equal(t, "test-value", dest)
	})
}

func TestMockCache_Set(t *testing.T) {
	t.Run("set stores value", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		err := cache.Set(ctx, "test-key", "test-value", 1*time.Hour)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", cache.data["test-key"])
	})
}

func TestMockCache_Delete(t *testing.T) {
	t.Run("delete removes value", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		cache.data["test-key"] = "test-value"
		err := cache.Delete(ctx, "test-key")

		assert.NoError(t, err)
		_, exists := cache.data["test-key"]
		assert.False(t, exists)
	})
}

func TestMockCache_Exists(t *testing.T) {
	t.Run("exists returns false for missing key", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		exists := cache.Exists(ctx, "nonexistent")
		assert.False(t, exists)
	})

	t.Run("exists returns true for existing key", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		cache.data["test-key"] = "test-value"
		exists := cache.Exists(ctx, "test-key")

		assert.True(t, exists)
	})
}

func TestMockCache_TTL(t *testing.T) {
	t.Run("TTL returns zero duration", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		ttl, err := cache.TTL(ctx, "test-key")

		assert.NoError(t, err)
		assert.Equal(t, time.Duration(0), ttl)
	})
}

func TestMockCache_Close(t *testing.T) {
	t.Run("close returns nil", func(t *testing.T) {
		cache := NewMockCache(t)

		err := cache.Close()
		assert.NoError(t, err)
	})
}

func TestMockCache_Health(t *testing.T) {
	t.Run("health returns nil", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		err := cache.Health(ctx)
		assert.NoError(t, err)
	})
}

func TestMockCache_DeleteByPattern(t *testing.T) {
	t.Run("delete by pattern returns nil", func(t *testing.T) {
		cache := NewMockCache(t)
		ctx := context.Background()

		err := cache.DeleteByPattern(ctx, "test-*")
		assert.NoError(t, err)
	})
}

func TestNewTestDB(t *testing.T) {
	t.Run("creates database connection or skips", func(t *testing.T) {
		db := NewTestDB(t)

		// Either we get a DB or the test was skipped
		// We can't assert much here without an actual database
		if db == nil {
			t.Log("Test database not available - test was skipped")
		} else {
			assert.NotNil(t, db)
			assert.NotNil(t, t) // t should not be nil
		}
	})
}

func TestSetupTestDatabase(t *testing.T) {
	t.Run("setup database requires DB", func(t *testing.T) {
		db := NewTestDB(t)
		if db == nil {
			t.Skip("Database not available")
			return
		}

		// This should not panic
		SetupTestDatabase(t, db)

		// Verify tables exist by querying them
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM tenants")
		require.NoError(t, err)
		assert.Equal(t, 1, count, "Should have one test tenant")
	})
}

func TestCleanupTestDatabase(t *testing.T) {
	t.Run("cleanup database requires DB", func(t *testing.T) {
		db := NewTestDB(t)
		if db == nil {
			t.Skip("Database not available")
			return
		}

		SetupTestDatabase(t, db)

		// Cleanup should not panic
		CleanupTestDatabase(t, db)

		// Verify tables are dropped
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM tenants")
		assert.Error(t, err, "Table should not exist after cleanup")
	})
}

func TestMockCache_Struct(t *testing.T) {
	t.Run("mock cache data map", func(t *testing.T) {
		cache := NewMockCache(t)

		assert.NotNil(t, cache.data)
		assert.IsType(t, make(map[string]string), cache.data)
	})
}

func TestTestHelper_IDGeneration(t *testing.T) {
	t.Run("IDs are unique across types", func(t *testing.T) {
		tenantID := GenerateTestTenantID()
		userID := GenerateTestUserID()
		credentialID := GenerateTestCredentialID()

		assert.NotEqual(t, tenantID, userID)
		assert.NotEqual(t, userID, credentialID)
		assert.NotEqual(t, tenantID, credentialID)
	})
}
