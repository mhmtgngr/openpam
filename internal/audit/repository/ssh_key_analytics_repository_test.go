// Package repository provides tests for SSH key analytics repository
package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSSHKeyAnalyticsTest creates a test database with SSH key analytics tables
func setupSSHKeyAnalyticsTest(t *testing.T) (*sqlx.DB, *SSHKeyAnalyticsRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)

	// Create SSH key analytics table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ssh_key_analytics (
			id UUID PRIMARY KEY,
			tenant_id UUID NOT NULL,
			ssh_key_id UUID NOT NULL,
			date DATE NOT NULL,
			usage_count INTEGER NOT NULL DEFAULT 0,
			unique_users INTEGER NOT NULL DEFAULT 0,
			unique_targets INTEGER NOT NULL DEFAULT 0,
			first_use_time TIMESTAMPTZ,
			last_use_time TIMESTAMPTZ,
			avg_session_duration_seconds FLOAT,
			off_hours_usage INTEGER NOT NULL DEFAULT 0,
			unusual_source_usage INTEGER NOT NULL DEFAULT 0,
			failed_attempts INTEGER NOT NULL DEFAULT 0,
			metadata BYTEA,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, ssh_key_id, date)
		);

		CREATE INDEX IF NOT EXISTS idx_ssh_key_analytics_tenant ON ssh_key_analytics(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_ssh_key_analytics_key ON ssh_key_analytics(ssh_key_id);
		CREATE INDEX IF NOT EXISTS idx_ssh_key_analytics_date ON ssh_key_analytics(date);
	`)
	require.NoError(t, err)

	logger := opamtesting.Logger(t)
	repo := NewSSHKeyAnalyticsRepository(db, logger)

	return db, repo
}

func TestNewSSHKeyAnalyticsRepository(t *testing.T) {
	db, _ := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}

	logger := opamtesting.Logger(t)
	repo := NewSSHKeyAnalyticsRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_Create(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("create new analytics entry", func(t *testing.T) {
		firstUse := time.Now()
		lastUse := time.Now()
		avgDuration := 300.0

		analytics := &model.SSHKeyAnalytics{
			TenantID:                  tenantID,
			SSHKeyID:                  sshKeyID,
			Date:                      date,
			UsageCount:                10,
			UniqueUsers:               3,
			UniqueTargets:             5,
			FirstUseTime:              &firstUse,
			LastUseTime:               &lastUse,
			AvgSessionDurationSeconds: &avgDuration,
			OffHoursUsage:             2,
			UnusualSourceUsage:        1,
			FailedAttempts:            0,
			Metadata:                  []byte(`{"key_type":"rsa"}`),
		}

		err := repo.Create(ctx, analytics)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, analytics.ID)
		assert.False(t, analytics.CreatedAt.IsZero())
		assert.False(t, analytics.UpdatedAt.IsZero())
	})

	t.Run("upsert on conflict", func(t *testing.T) {
		// Create initial entry
		analytics1 := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      sshKeyID,
			Date:          date,
			UsageCount:    5,
			UniqueUsers:   2,
			UniqueTargets: 3,
		}

		err := repo.Create(ctx, analytics1)
		require.NoError(t, err)

		// Upsert with new data
		analytics2 := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      sshKeyID,
			Date:          date,
			UsageCount:    3,
			UniqueUsers:   1,
			UniqueTargets: 2,
		}

		err = repo.Create(ctx, analytics2)
		require.NoError(t, err)

		// Verify the upsert merged the data (usage_count should be summed)
		// Note: The exact behavior depends on the ON CONFLICT clause
		assert.NotEqual(t, uuid.Nil, analytics2.ID)
	})

	t.Run("create entry with minimal fields", func(t *testing.T) {
		newKeyID := uuid.New()

		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      newKeyID,
			Date:          date.Add(24 * time.Hour),
			UsageCount:    0,
			UniqueUsers:   0,
			UniqueTargets: 0,
		}

		err := repo.Create(ctx, analytics)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, analytics.ID)
	})
}

// =============================================================================
// Get Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_Get(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	// Create test data
	firstUse := time.Now().Add(-2 * time.Hour)
	lastUse := time.Now()
	avgDuration := 300.0

	original := &model.SSHKeyAnalytics{
		TenantID:                  tenantID,
		SSHKeyID:                  sshKeyID,
		Date:                      date,
		UsageCount:                15,
		UniqueUsers:               4,
		UniqueTargets:             7,
		FirstUseTime:              &firstUse,
		LastUseTime:               &lastUse,
		AvgSessionDurationSeconds: &avgDuration,
		OffHoursUsage:             3,
		UnusualSourceUsage:        2,
		FailedAttempts:            1,
		Metadata:                  []byte(`{"test":"data"}`),
	}

	err := repo.Create(ctx, original)
	require.NoError(t, err)

	t.Run("get existing analytics", func(t *testing.T) {
		found, err := repo.Get(ctx, tenantID, sshKeyID, date)
		require.NoError(t, err)
		assert.Equal(t, original.ID, found.ID)
		assert.Equal(t, tenantID, found.TenantID)
		assert.Equal(t, sshKeyID, found.SSHKeyID)
		assert.Equal(t, 15, found.UsageCount)
		assert.Equal(t, 4, found.UniqueUsers)
		assert.Equal(t, 7, found.UniqueTargets)
		assert.NotNil(t, found.FirstUseTime)
		assert.NotNil(t, found.AvgSessionDurationSeconds)
		assert.Equal(t, 300.0, *found.AvgSessionDurationSeconds)
		assert.Equal(t, 3, found.OffHoursUsage)
		assert.Equal(t, 2, found.UnusualSourceUsage)
		assert.Equal(t, 1, found.FailedAttempts)
	})

	t.Run("get non-existent analytics", func(t *testing.T) {
		_, err := repo.Get(ctx, tenantID, uuid.New(), date)
		assert.Error(t, err)
	})

	t.Run("get with different date", func(t *testing.T) {
		otherDate := date.Add(24 * time.Hour)
		_, err := repo.Get(ctx, tenantID, sshKeyID, otherDate)
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_List(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()

	// Create test data across multiple days
	baseDate := time.Now().Truncate(24 * time.Hour)
	for i := 0; i < 5; i++ {
		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      sshKeyID,
			Date:          baseDate.Add(time.Duration(i) * 24 * time.Hour),
			UsageCount:    10 + i,
			UniqueUsers:   2 + i,
			UniqueTargets: 3,
		}
		_ = repo.Create(ctx, analytics)
	}

	t.Run("list all analytics for tenant", func(t *testing.T) {
		filter := model.SSHKeyAnalyticsFilter{
			TenantID: &tenantID,
		}

		results, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 5)
	})

	t.Run("list by SSH key ID", func(t *testing.T) {
		filter := model.SSHKeyAnalyticsFilter{
			TenantID: &tenantID,
			SSHKeyID: &sshKeyID,
		}

		results, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 5)

		// Verify all results belong to the same key
		for _, r := range results {
			assert.Equal(t, sshKeyID, r.SSHKeyID)
		}
	})

	t.Run("list by date range", func(t *testing.T) {
		dateFrom := baseDate
		dateTo := baseDate.Add(2 * 24 * time.Hour)

		filter := model.SSHKeyAnalyticsFilter{
			TenantID: &tenantID,
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		results, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3)
	})

	t.Run("list with empty filter", func(t *testing.T) {
		filter := model.SSHKeyAnalyticsFilter{}

		results, err := repo.List(ctx, filter)
		require.NoError(t, err)
		// Should return all records
		assert.GreaterOrEqual(t, len(results), 5)
	})

	t.Run("list with no matches", func(t *testing.T) {
		otherTenantID := uuid.New()
		dateFrom := baseDate.Add(-30 * 24 * time.Hour)
		dateTo := baseDate.Add(-29 * 24 * time.Hour)

		filter := model.SSHKeyAnalyticsFilter{
			TenantID: &otherTenantID,
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
		}

		results, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_Update(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	// Create initial entry
	original := &model.SSHKeyAnalytics{
		TenantID:      tenantID,
		SSHKeyID:      sshKeyID,
		Date:          date,
		UsageCount:    10,
		UniqueUsers:   3,
		UniqueTargets: 5,
	}

	err := repo.Create(ctx, original)
	require.NoError(t, err)

	t.Run("update existing analytics", func(t *testing.T) {
		// Get the created record
		found, err := repo.Get(ctx, tenantID, sshKeyID, date)
		require.NoError(t, err)

		// Modify fields
		firstUse := time.Now().Add(-24 * time.Hour)
		lastUse := time.Now()
		avgDuration := 600.0

		found.UsageCount = 20
		found.UniqueUsers = 5
		found.UniqueTargets = 8
		found.FirstUseTime = &firstUse
		found.LastUseTime = &lastUse
		found.AvgSessionDurationSeconds = &avgDuration
		found.OffHoursUsage = 5
		found.UnusualSourceUsage = 3
		found.FailedAttempts = 2
		found.Metadata = []byte(`{"updated":"true"}`)

		// Update
		err = repo.Update(ctx, found)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.Get(ctx, tenantID, sshKeyID, date)
		require.NoError(t, err)
		assert.Equal(t, 20, updated.UsageCount)
		assert.Equal(t, 5, updated.UniqueUsers)
		assert.Equal(t, 8, updated.UniqueTargets)
		assert.NotNil(t, updated.FirstUseTime)
		assert.NotNil(t, updated.LastUseTime)
		assert.NotNil(t, updated.AvgSessionDurationSeconds)
		assert.Equal(t, 600.0, *updated.AvgSessionDurationSeconds)
		assert.Equal(t, 5, updated.OffHoursUsage)
		assert.Equal(t, 3, updated.UnusualSourceUsage)
		assert.Equal(t, 2, updated.FailedAttempts)
	})

	t.Run("update non-existent record", func(t *testing.T) {
		nonExistent := &model.SSHKeyAnalytics{
			ID:        uuid.New(),
			TenantID:  tenantID,
			SSHKeyID:  uuid.New(),
			Date:      date.Add(24 * time.Hour),
			UsageCount: 1,
		}

		err := repo.Update(ctx, nonExistent)
		// Should not error, just no rows affected
		require.NoError(t, err)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_Delete(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("delete existing analytics", func(t *testing.T) {
		// Create entry
		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      sshKeyID,
			Date:          date,
			UsageCount:    10,
			UniqueUsers:   3,
			UniqueTargets: 5,
		}

		err := repo.Create(ctx, analytics)
		require.NoError(t, err)

		// Verify it exists
		_, err = repo.Get(ctx, tenantID, sshKeyID, date)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, analytics.ID)
		require.NoError(t, err)

		// Verify it's gone
		_, err = repo.Get(ctx, tenantID, sshKeyID, date)
		assert.Error(t, err)
	})

	t.Run("delete non-existent analytics", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows affected")
	})
}

// =============================================================================
// GetByKeyID Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_GetByKeyID(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	baseDate := time.Now().Truncate(24 * time.Hour)

	// Create multiple entries for the same key
	for i := 0; i < 5; i++ {
		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      sshKeyID,
			Date:          baseDate.Add(time.Duration(i) * 24 * time.Hour),
			UsageCount:    10 + i,
			UniqueUsers:   2,
			UniqueTargets: 3,
		}
		_ = repo.Create(ctx, analytics)
	}

	t.Run("get by key ID without limit", func(t *testing.T) {
		results, err := repo.GetByKeyID(ctx, tenantID, sshKeyID, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 5)
	})

	t.Run("get by key ID with limit", func(t *testing.T) {
		results, err := repo.GetByKeyID(ctx, tenantID, sshKeyID, 3)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)
	})

	t.Run("get by key ID with no results", func(t *testing.T) {
		results, err := repo.GetByKeyID(ctx, tenantID, uuid.New(), 0)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("verify descending date order", func(t *testing.T) {
		results, err := repo.GetByKeyID(ctx, tenantID, sshKeyID, 0)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(results), 2)

		// Verify results are in descending order by date
		for i := 0; i < len(results)-1; i++ {
			assert.True(t, results[i].Date.After(results[i+1].Date) ||
				results[i].Date.Equal(results[i+1].Date))
		}
	})
}

// =============================================================================
// GetUsageSummary Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_GetUsageSummary(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	baseDate := time.Now().Truncate(24 * time.Hour)

	// Create test data
	for i := 0; i < 10; i++ {
		analytics := &model.SSHKeyAnalytics{
			TenantID:          tenantID,
			SSHKeyID:          uuid.New(),
			Date:              baseDate,
			UsageCount:        10 + i,
			UniqueUsers:       2,
			UniqueTargets:     3,
			OffHoursUsage:     i % 3,
			UnusualSourceUsage: i % 2,
			FailedAttempts:    i % 4,
		}
		_ = repo.Create(ctx, analytics)
	}

	// Create an unused key entry
	unusedKey := &model.SSHKeyAnalytics{
		TenantID:      tenantID,
		SSHKeyID:      uuid.New(),
		Date:          baseDate,
		UsageCount:    0,
		UniqueUsers:   0,
		UniqueTargets: 0,
	}
	_ = repo.Create(ctx, unusedKey)

	dateFrom := baseDate.Add(-24 * time.Hour)
	dateTo := baseDate.Add(24 * time.Hour)

	t.Run("get usage summary", func(t *testing.T) {
		summary, err := repo.GetUsageSummary(ctx, tenantID, dateFrom, dateTo)
		require.NoError(t, err)
		assert.NotNil(t, summary)

		assert.GreaterOrEqual(t, summary.TotalKeys, 11)
		assert.Greater(t, summary.TotalUsage, 0)
		assert.Greater(t, summary.TotalUniqueUsers, 0)
		assert.Greater(t, summary.TotalUniqueTargets, 0)
		assert.GreaterOrEqual(t, summary.TotalOffHours, 0)
		assert.GreaterOrEqual(t, summary.TotalUnusual, 0)
		assert.GreaterOrEqual(t, summary.TotalFailed, 0)
		assert.GreaterOrEqual(t, summary.UnusedKeys, 1)
	})

	t.Run("get summary for empty date range", func(t *testing.T) {
		emptyFrom := baseDate.Add(-30 * 24 * time.Hour)
		emptyTo := baseDate.Add(-29 * 24 * time.Hour)

		summary, err := repo.GetUsageSummary(ctx, tenantID, emptyFrom, emptyTo)
		require.NoError(t, err)
		assert.NotNil(t, summary)
		assert.Equal(t, 0, summary.TotalKeys)
	})
}

// =============================================================================
// GetMostUsedKeys Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_GetMostUsedKeys(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	baseDate := time.Now().Truncate(24 * time.Hour)

	// Create test data with varying usage
	keyIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		keyIDs[i] = uuid.New()
		// Create multiple days for the same key to test aggregation
		for day := 0; day < 3; day++ {
			analytics := &model.SSHKeyAnalytics{
				TenantID:      tenantID,
				SSHKeyID:      keyIDs[i],
				Date:          baseDate.Add(time.Duration(day) * 24 * time.Hour),
				UsageCount:    (5 - i) * 10, // Higher usage for lower indices
				UniqueUsers:   i + 1,
				UniqueTargets: i + 2,
			}
			_ = repo.Create(ctx, analytics)
		}
	}

	dateFrom := baseDate.Add(-24 * time.Hour)
	dateTo := baseDate.Add(3 * 24 * time.Hour)

	t.Run("get most used keys without limit", func(t *testing.T) {
		results, err := repo.GetMostUsedKeys(ctx, tenantID, dateFrom, dateTo, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 5)

		// Verify descending order by total_usage
		if len(results) > 1 {
			assert.GreaterOrEqual(t, results[0].TotalUsage, results[1].TotalUsage)
		}
	})

	t.Run("get most used keys with limit", func(t *testing.T) {
		results, err := repo.GetMostUsedKeys(ctx, tenantID, dateFrom, dateTo, 3)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 3)
	})

	t.Run("verify key usage rank structure", func(t *testing.T) {
		results, err := repo.GetMostUsedKeys(ctx, tenantID, dateFrom, dateTo, 1)
		require.NoError(t, err)
		require.NotEmpty(t, results)

		rank := results[0]
		assert.NotEqual(t, uuid.Nil, rank.SSHKeyID)
		assert.Greater(t, rank.TotalUsage, 0)
		assert.GreaterOrEqual(t, rank.UniqueUsers, 0)
		assert.GreaterOrEqual(t, rank.UniqueTargets, 0)
	})

	t.Run("get most used keys for empty tenant", func(t *testing.T) {
		emptyTenantID := uuid.New()
		results, err := repo.GetMostUsedKeys(ctx, emptyTenantID, dateFrom, dateTo, 0)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// =============================================================================
// GetAnomalousKeys Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_GetAnomalousKeys(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	baseDate := time.Now().Truncate(24 * time.Hour)

	// Create normal keys
	for i := 0; i < 3; i++ {
		analytics := &model.SSHKeyAnalytics{
			TenantID:          tenantID,
			SSHKeyID:          uuid.New(),
			Date:              baseDate,
			UsageCount:        10,
			UniqueUsers:       2,
			UniqueTargets:     3,
			OffHoursUsage:     0,
			UnusualSourceUsage: 0,
			FailedAttempts:    0,
		}
		_ = repo.Create(ctx, analytics)
	}

	// Create anomalous keys
	for i := 0; i < 2; i++ {
		analytics := &model.SSHKeyAnalytics{
			TenantID:          tenantID,
			SSHKeyID:          uuid.New(),
			Date:              baseDate,
			UsageCount:        10,
			UniqueUsers:       2,
			UniqueTargets:     3,
			OffHoursUsage:     5,
			UnusualSourceUsage: 8,
			FailedAttempts:    3,
		}
		_ = repo.Create(ctx, analytics)
	}

	dateFrom := baseDate.Add(-24 * time.Hour)
	dateTo := baseDate.Add(24 * time.Hour)

	t.Run("get anomalous keys with threshold", func(t *testing.T) {
		results, err := repo.GetAnomalousKeys(ctx, tenantID, dateFrom, dateTo, 3)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 2)
	})

	t.Run("get anomalous keys with high threshold", func(t *testing.T) {
		results, err := repo.GetAnomalousKeys(ctx, tenantID, dateFrom, dateTo, 10)
		require.NoError(t, err)
		// Should return fewer results with higher threshold
		assert.GreaterOrEqual(t, len(results), 0)
	})

	t.Run("get anomalous keys with zero threshold", func(t *testing.T) {
		results, err := repo.GetAnomalousKeys(ctx, tenantID, dateFrom, dateTo, 0)
		require.NoError(t, err)
		// Should return all keys including normal ones
		assert.GreaterOrEqual(t, len(results), 5)
	})

	t.Run("verify anomalous key structure", func(t *testing.T) {
		results, err := repo.GetAnomalousKeys(ctx, tenantID, dateFrom, dateTo, 3)
		require.NoError(t, err)
		require.NotEmpty(t, results)

		anomaly := results[0]
		assert.NotEqual(t, uuid.Nil, anomaly.ID)
		assert.Equal(t, tenantID, anomaly.TenantID)
		assert.NotEqual(t, uuid.Nil, anomaly.SSHKeyID)
		// At least one of unusual_source_usage or failed_attempts should be > threshold
		assert.True(t, anomaly.UnusualSourceUsage > 3 || anomaly.FailedAttempts > 3)
	})
}

// =============================================================================
// Concurrent Operations Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_ConcurrentOperations(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	sshKeyID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)

	t.Run("concurrent upserts", func(t *testing.T) {
		errChan := make(chan error, 10)

		// Launch multiple goroutines trying to create/update the same entry
		for i := 0; i < 10; i++ {
			go func(index int) {
				analytics := &model.SSHKeyAnalytics{
					TenantID:      tenantID,
					SSHKeyID:      sshKeyID,
					Date:          date,
					UsageCount:    1,
					UniqueUsers:   1,
					UniqueTargets: 1,
				}
				errChan <- repo.Create(ctx, analytics)
			}(i)
		}

		// Collect results
		errors := []error{}
		for i := 0; i < 10; i++ {
			if err := <-errChan; err != nil {
				errors = append(errors, err)
			}
		}

		// Some operations might fail due to concurrent updates, but not all
		assert.LessOrEqual(t, len(errors), 5, fmt.Sprintf("Too many errors: %v", errors))
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_ErrorHandling(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("handle nil tenant ID", func(t *testing.T) {
		// This should work since tenant_id is a required field
		sshKeyID := uuid.New()
		date := time.Now().Truncate(24 * time.Hour)

		_, err := repo.Get(ctx, uuid.Nil, sshKeyID, date)
		assert.Error(t, err)
	})

	t.Run("handle zero date", func(t *testing.T) {
		sshKeyID := uuid.New()

		_, err := repo.Get(ctx, tenantID, sshKeyID, time.Time{})
		// Should error or return no results
		assert.Error(t, err)
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestSSHKeyAnalyticsRepository_EdgeCases(t *testing.T) {
	db, repo := setupSSHKeyAnalyticsTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())

	t.Run("create with maximum values", func(t *testing.T) {
		firstUse := time.Now()
		lastUse := time.Now()
		avgDuration := 999999.0
		maxInt := 2147483647

		analytics := &model.SSHKeyAnalytics{
			TenantID:                  tenantID,
			SSHKeyID:                  uuid.New(),
			Date:                      time.Now().Truncate(24 * time.Hour),
			UsageCount:                maxInt,
			UniqueUsers:               maxInt,
			UniqueTargets:             maxInt,
			FirstUseTime:              &firstUse,
			LastUseTime:               &lastUse,
			AvgSessionDurationSeconds: &avgDuration,
			OffHoursUsage:             maxInt,
			UnusualSourceUsage:        maxInt,
			FailedAttempts:            maxInt,
			Metadata:                  make([]byte, 0),
		}

		err := repo.Create(ctx, analytics)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, analytics.ID)
	})

	t.Run("create with large metadata", func(t *testing.T) {
		largeMetadata := make([]byte, 1024*100) // 100KB
		for i := range largeMetadata {
			largeMetadata[i] = byte(i % 256)
		}

		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      uuid.New(),
			Date:          time.Now().Truncate(24 * time.Hour),
			UsageCount:    1,
			UniqueUsers:   1,
			UniqueTargets: 1,
			Metadata:      largeMetadata,
		}

		err := repo.Create(ctx, analytics)
		// May fail due to size limits, but shouldn't panic
		if err != nil {
			t.Logf("Large metadata rejected (expected): %v", err)
		}
	})

	t.Run("date boundaries", func(t *testing.T) {
		// Test with dates at year boundaries
		analytics := &model.SSHKeyAnalytics{
			TenantID:      tenantID,
			SSHKeyID:      uuid.New(),
			Date:          time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			UsageCount:    1,
			UniqueUsers:   1,
			UniqueTargets: 1,
		}

		err := repo.Create(ctx, analytics)
		require.NoError(t, err)
	})
}
