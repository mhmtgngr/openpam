// Package repository provides tests for command blacklist repository
package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	opamtesting "github.com/openpam/openpam/internal/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCommandBlacklistTest creates a test database with command blacklist table
func setupCommandBlacklistTest(t *testing.T) (*sqlx.DB, *CommandBlacklistRepository) {
	db := opamtesting.NewTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
		return nil, nil
	}

	opamtesting.SetupTestDatabase(t, db)

	// Create command_blacklist table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS command_blacklist (
			id UUID PRIMARY KEY,
			tenant_id UUID,
			command_pattern TEXT NOT NULL,
			pattern_type TEXT NOT NULL,
			base_command TEXT,
			action TEXT NOT NULL,
			severity TEXT NOT NULL,
			applies_to_users UUID[],
			applies_to_groups UUID[],
			applies_to_targets TEXT[],
			allow_override BOOLEAN DEFAULT false,
			override_roles TEXT[],
			reason TEXT NOT NULL,
			risk_category TEXT,
			enabled BOOLEAN DEFAULT true,
			created_by UUID NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_command_blacklist_tenant ON command_blacklist(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_command_blacklist_pattern ON command_blacklist(command_pattern);
		CREATE INDEX IF NOT EXISTS idx_command_blacklist_base ON command_blacklist(base_command);
		CREATE INDEX IF NOT EXISTS idx_command_blacklist_action ON command_blacklist(action);
		CREATE INDEX IF NOT EXISTS idx_command_blacklist_severity ON command_blacklist(severity);
		CREATE INDEX IF NOT EXISTS idx_command_blacklist_enabled ON command_blacklist(enabled);
	`)
	require.NoError(t, err)

	logger := opamtesting.Logger(t)
	repo := NewCommandBlacklistRepository(db, logger)

	return db, repo
}

func TestNewCommandBlacklistRepository(t *testing.T) {
	db, _ := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}

	logger := opamtesting.Logger(t)
	repo := NewCommandBlacklistRepository(db, logger)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	assert.NotNil(t, repo.logger)
}

// =============================================================================
// Create Tests
// =============================================================================

func TestCommandBlacklistRepository_Create(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	baseCommand := "rm"
	riskCategory := "data_destruction"
	appliesToUsers := []uuid.UUID{uuid.New(), uuid.New()}
	appliesToTargets := []string{"/var/*", "/etc/*"}

	t.Run("create tenant-specific blacklist with all fields", func(t *testing.T) {
		blacklist := &model.CommandBlacklist{
			TenantID:         &tenantID,
			CommandPattern:   "rm -rf /*",
			PatternType:      string(model.PatternTypeGlob),
			BaseCommand:      &baseCommand,
			Action:           string(model.CommandActionBlock),
			Severity:         string(model.SeverityCritical),
			AppliesToUsers:   appliesToUsers,
			AppliesToTargets: appliesToTargets,
			AllowOverride:    true,
			OverrideRoles:    []string{"admin", "super_admin"},
			Reason:           "Dangerous command that can delete system files",
			RiskCategory:     &riskCategory,
			Enabled:          true,
			CreatedBy:        createdBy,
		}

		err := repo.Create(ctx, blacklist)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, blacklist.ID)
		assert.False(t, blacklist.CreatedAt.IsZero())
		assert.False(t, blacklist.UpdatedAt.IsZero())
	})

	t.Run("create global blacklist (nil tenant)", func(t *testing.T) {
		baseCommand := "dd"
		blacklist := &model.CommandBlacklist{
			TenantID:        nil, // Global rule
			CommandPattern:  "dd if=/dev/zero of=/dev/sda",
			PatternType:     string(model.PatternTypeExact),
			BaseCommand:     &baseCommand,
			Action:          string(model.CommandActionBlock),
			Severity:        string(model.SeverityCritical),
			AppliesToUsers:  nil,
			AppliesToGroups: nil,
			AppliesToTargets: nil,
			AllowOverride:   false,
			OverrideRoles:   nil,
			Reason:          "Global rule: destructive disk operation",
			RiskCategory:    &riskCategory,
			Enabled:         true,
			CreatedBy:       createdBy,
		}

		err := repo.Create(ctx, blacklist)
		require.NoError(t, err)
		assert.Nil(t, blacklist.TenantID)
	})

	t.Run("create with minimal fields", func(t *testing.T) {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: "chmod 000 /etc/*",
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityHigh),
			Reason:         "Dangerous permission modification",
			Enabled:        true,
			CreatedBy:      createdBy,
		}

		err := repo.Create(ctx, blacklist)
		require.NoError(t, err)
	})
}

// =============================================================================
// GetByID Tests
// =============================================================================

func TestCommandBlacklistRepository_GetByID(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	baseCommand := "rm"
	riskCategory := "data_destruction"

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "rm -rf /*",
		PatternType:    string(model.PatternTypeGlob),
		BaseCommand:    &baseCommand,
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityCritical),
		Reason:         "Dangerous command",
		RiskCategory:   &riskCategory,
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("get existing blacklist", func(t *testing.T) {
		found, err := repo.GetByID(ctx, blacklist.ID)
		require.NoError(t, err)

		assert.Equal(t, blacklist.ID, found.ID)
		assert.Equal(t, tenantID, *found.TenantID)
		assert.Equal(t, "rm -rf /*", found.CommandPattern)
		assert.Equal(t, "glob", found.PatternType)
		assert.Equal(t, "rm", *found.BaseCommand)
		assert.Equal(t, "block", found.Action)
		assert.Equal(t, "critical", found.Severity)
		assert.True(t, found.Enabled)
	})

	t.Run("get non-existent blacklist", func(t *testing.T) {
		_, err := repo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
	})
}

// =============================================================================
// List Tests
// =============================================================================

func TestCommandBlacklistRepository_List(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	baseCommand := "rm"
	riskCategory := "data_destruction"

	// Create tenant-specific rules
	for i := 0; i < 3; i++ {
		enabled := i%2 == 0
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: fmt.Sprintf("test command %d", i),
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    &baseCommand,
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityHigh),
			Reason:         fmt.Sprintf("Test reason %d", i),
			RiskCategory:   &riskCategory,
			Enabled:        enabled,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	// Create global rules
	for i := 0; i < 2; i++ {
		blacklist := &model.CommandBlacklist{
			TenantID:       nil,
			CommandPattern: fmt.Sprintf("global command %d", i),
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         fmt.Sprintf("Global reason %d", i),
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	t.Run("list all for tenant (includes global)", func(t *testing.T) {
		filter := model.CommandBlacklistFilter{
			TenantID: &tenantID,
		}

		blacklists, err := repo.List(ctx, filter)

		require.NoError(t, err)
		// Should include both tenant-specific and global rules
		assert.GreaterOrEqual(t, len(blacklists), 5)
	})

	t.Run("list only enabled", func(t *testing.T) {
		enabled := true
		filter := model.CommandBlacklistFilter{
			TenantID: &tenantID,
			Enabled:  &enabled,
		}

		blacklists, err := repo.List(ctx, filter)

		require.NoError(t, err)
		for _, b := range blacklists {
			assert.True(t, b.Enabled)
		}
	})

	t.Run("list by action type", func(t *testing.T) {
		action := string(model.CommandActionBlock)
		filter := model.CommandBlacklistFilter{
			TenantID: &tenantID,
			Action:   &action,
		}

		blacklists, err := repo.List(ctx, filter)

		require.NoError(t, err)
		for _, b := range blacklists {
			assert.Equal(t, "block", b.Action)
		}
	})

	t.Run("list by severity", func(t *testing.T) {
		severity := string(model.SeverityCritical)
		filter := model.CommandBlacklistFilter{
			TenantID: &tenantID,
			Severity: &severity,
		}

		blacklists, err := repo.List(ctx, filter)

		require.NoError(t, err)
		for _, b := range blacklists {
			assert.Equal(t, "critical", b.Severity)
		}
	})

	t.Run("list by pattern type", func(t *testing.T) {
		patternType := string(model.PatternTypeExact)
		filter := model.CommandBlacklistFilter{
			TenantID:    &tenantID,
			PatternType: &patternType,
		}

		blacklists, err := repo.List(ctx, filter)

		require.NoError(t, err)
		for _, b := range blacklists {
			assert.Equal(t, "exact", b.PatternType)
		}
	})
}

// =============================================================================
// Update Tests
// =============================================================================

func TestCommandBlacklistRepository_Update(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "original pattern",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Original reason",
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("update blacklist fields", func(t *testing.T) {
		newPattern := "updated pattern"
		newSeverity := string(model.SeverityCritical)
		newReason := "Updated reason"
		enabled := false

		blacklist.CommandPattern = newPattern
		blacklist.Severity = newSeverity
		blacklist.Reason = newReason
		blacklist.Enabled = enabled

		err := repo.Update(ctx, blacklist)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, blacklist.ID)
		require.NoError(t, err)

		assert.Equal(t, "updated pattern", updated.CommandPattern)
		assert.Equal(t, "critical", updated.Severity)
		assert.Equal(t, "Updated reason", updated.Reason)
		assert.False(t, updated.Enabled)
	})
}

// =============================================================================
// Delete Tests
// =============================================================================

func TestCommandBlacklistRepository_Delete(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "to be deleted",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Will be deleted",
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("soft delete (disable)", func(t *testing.T) {
		err := repo.Delete(ctx, blacklist.ID)
		require.NoError(t, err)

		// Verify it's disabled but not removed
		found, err := repo.GetByID(ctx, blacklist.ID)
		require.NoError(t, err)
		assert.False(t, found.Enabled)
	})

	t.Run("delete non-existent", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no rows affected")
	})
}

func TestCommandBlacklistRepository_DeleteHard(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "to be hard deleted",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Will be hard deleted",
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("hard delete", func(t *testing.T) {
		err := repo.DeleteHard(ctx, blacklist.ID)
		require.NoError(t, err)

		// Verify it's removed
		_, err = repo.GetByID(ctx, blacklist.ID)
		assert.Error(t, err)
	})
}

// =============================================================================
// FindMatching Tests
// =============================================================================

func TestCommandBlacklistRepository_FindMatching(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	userID := uuid.New()
	groupID := uuid.New()
	baseCommand := "rm"
	riskCategory := "data_destruction"

	// Create blacklist entries
	entries := []*model.CommandBlacklist{
		{
			TenantID:       &tenantID,
			CommandPattern: "rm -rf",
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    &baseCommand,
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         "Dangerous",
			RiskCategory:   &riskCategory,
			Enabled:        true,
			CreatedBy:      createdBy,
		},
		{
			TenantID:       &tenantID,
			CommandPattern: "chmod",
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    stringPtr("chmod"),
			Action:         string(model.CommandActionWarn),
			Severity:       string(model.SeverityMedium),
			AppliesToUsers:  []uuid.UUID{userID},
			Reason:         "Warn on chmod",
			Enabled:        true,
			CreatedBy:      createdBy,
		},
		{
			TenantID:       &tenantID,
			CommandPattern: "sudo",
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    stringPtr("sudo"),
			Action:         string(model.CommandActionAudit),
			Severity:       string(model.SeverityLow),
			AppliesToGroups: []uuid.UUID{groupID},
			Reason:         "Audit sudo",
			Enabled:        true,
			CreatedBy:      createdBy,
		},
		{
			TenantID:       &tenantID,
			CommandPattern: "dd",
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    stringPtr("dd"),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         "Block dd",
			Enabled:        false, // Disabled
			CreatedBy:      createdBy,
		},
	}

	for _, e := range entries {
		_ = repo.Create(ctx, e)
	}

	t.Run("find matching without user restrictions", func(t *testing.T) {
		matches, err := repo.FindMatching(ctx, tenantID, "rm -rf", nil, nil)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(matches), 1)
	})

	t.Run("find matching with user restrictions", func(t *testing.T) {
		matches, err := repo.FindMatching(ctx, tenantID, "chmod", []uuid.UUID{userID}, nil)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(matches), 1)
	})

	t.Run("find matching with group restrictions", func(t *testing.T) {
		matches, err := repo.FindMatching(ctx, tenantID, "sudo", nil, []uuid.UUID{groupID})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(matches), 1)
	})

	t.Run("disabled rules should not match", func(t *testing.T) {
		matches, err := repo.FindMatching(ctx, tenantID, "dd", nil, nil)

		require.NoError(t, err)
		// dd rule is disabled, so no matches
		assert.Empty(t, matches)
	})

	t.Run("no matching rules", func(t *testing.T) {
		matches, err := repo.FindMatching(ctx, tenantID, "innocent command", nil, nil)

		require.NoError(t, err)
		assert.Empty(t, matches)
	})
}

// =============================================================================
// Pattern Matching Tests
// =============================================================================

func TestCommandBlacklistRepository_PatternMatching(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	baseCommand := "rm"

	t.Run("exact pattern matching", func(t *testing.T) {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: "rm -rf /",
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    &baseCommand,
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         "Exact match test",
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)

		// Should match exactly
		matches, _ := repo.FindMatching(ctx, tenantID, "rm -rf /", nil, nil)
		assert.GreaterOrEqual(t, len(matches), 1)

		// Should not match similar
		matches, _ = repo.FindMatching(ctx, tenantID, "rm -rf /home", nil, nil)
		assert.Empty(t, matches)
	})

	t.Run("glob pattern matching", func(t *testing.T) {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: "rm -rf *",
			PatternType:    string(model.PatternTypeGlob),
			BaseCommand:    &baseCommand,
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         "Glob match test",
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)

		// Should match with wildcard
		matches, _ := repo.FindMatching(ctx, tenantID, "rm -rf /var", nil, nil)
		assert.GreaterOrEqual(t, len(matches), 1)
	})
}

// =============================================================================
// GetByAction Tests
// =============================================================================

func TestCommandBlacklistRepository_GetByAction(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	// Create rules with different actions
	actions := []string{
		string(model.CommandActionBlock),
		string(model.CommandActionWarn),
		string(model.CommandActionAudit),
	}

	for _, action := range actions {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: fmt.Sprintf("test-%s", action),
			PatternType:    string(model.PatternTypeExact),
			Action:         action,
			Severity:       string(model.SeverityMedium),
			Reason:         fmt.Sprintf("Test %s", action),
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	t.Run("get by action for tenant", func(t *testing.T) {
		blacklists, err := repo.GetByAction(ctx, &tenantID, string(model.CommandActionBlock))

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(blacklists), 1)

		for _, b := range blacklists {
			assert.Equal(t, "block", b.Action)
		}
	})

	t.Run("get global rules by action", func(t *testing.T) {
		globalBlacklist := &model.CommandBlacklist{
			TenantID:       nil,
			CommandPattern: "global block",
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         "Global block",
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, globalBlacklist)

		blacklists, err := repo.GetByAction(ctx, nil, string(model.CommandActionBlock))

		require.NoError(t, err)
		assert.Greater(t, len(blacklists), 0)

		for _, b := range blacklists {
			assert.Nil(t, b.TenantID)
		}
	})
}

// =============================================================================
// GetByBaseCommand Tests
// =============================================================================

func TestCommandBlacklistRepository_GetByBaseCommand(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	baseCommand := "sudo"

	// Create multiple rules for same base command
	for i := 0; i < 3; i++ {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: fmt.Sprintf("sudo %d", i),
			PatternType:    string(model.PatternTypeExact),
			BaseCommand:    &baseCommand,
			Action:         string(model.CommandActionWarn),
			Severity:       string(model.SeverityMedium),
			Reason:         fmt.Sprintf("Sudo rule %d", i),
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	t.Run("get by base command", func(t *testing.T) {
		blacklists, err := repo.GetByBaseCommand(ctx, tenantID, "sudo")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(blacklists), 3)

		for _, b := range blacklists {
			assert.Equal(t, "sudo", *b.BaseCommand)
		}
	})

	t.Run("get by non-existent base command", func(t *testing.T) {
		blacklists, err := repo.GetByBaseCommand(ctx, tenantID, "nonexistent")

		require.NoError(t, err)
		assert.Empty(t, blacklists)
	})
}

// =============================================================================
// GetStats Tests
// =============================================================================

func TestCommandBlacklistRepository_GetStats(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	// Create rules with various properties
	actions := []string{
		string(model.CommandActionBlock),
		string(model.CommandActionBlock),
		string(model.CommandActionWarn),
		string(model.CommandActionAudit),
	}
	severities := []string{
		string(model.SeverityCritical),
		string(model.SeverityHigh),
		string(model.SeverityMedium),
		string(model.SeverityLow),
	}

	for i := 0; i < 4; i++ {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: fmt.Sprintf("stats test %d", i),
			PatternType:    string(model.PatternTypeExact),
			Action:         actions[i],
			Severity:       severities[i],
			Reason:         fmt.Sprintf("Stats test %d", i),
			Enabled:        i%2 == 0,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	// Add global rule
	globalBlacklist := &model.CommandBlacklist{
		TenantID:       nil,
		CommandPattern: "global rule",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityCritical),
		Reason:         "Global stats",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, globalBlacklist)

	t.Run("get stats", func(t *testing.T) {
		stats, err := repo.GetStats(ctx, tenantID)

		require.NoError(t, err)
		assert.NotNil(t, stats)

		assert.GreaterOrEqual(t, stats.TotalEnabled, 2)
		assert.GreaterOrEqual(t, stats.TotalDisabled, 2)
		assert.GreaterOrEqual(t, stats.BlockCount, 2)
		assert.GreaterOrEqual(t, stats.WarnCount, 1)
		assert.GreaterOrEqual(t, stats.AuditCount, 1)
		assert.GreaterOrEqual(t, stats.CriticalCount, 1)
		assert.GreaterOrEqual(t, stats.GlobalCount, 1)
	})
}

// =============================================================================
// Enable/Disable Tests
// =============================================================================

func TestCommandBlacklistRepository_EnableDisable(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "toggle test",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Toggle test",
		Enabled:        false,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("enable blacklist", func(t *testing.T) {
		err := repo.Enable(ctx, blacklist.ID)
		require.NoError(t, err)

		found, _ := repo.GetByID(ctx, blacklist.ID)
		assert.True(t, found.Enabled)
	})

	t.Run("disable blacklist", func(t *testing.T) {
		err := repo.Disable(ctx, blacklist.ID)
		require.NoError(t, err)

		found, _ := repo.GetByID(ctx, blacklist.ID)
		assert.False(t, found.Enabled)
	})
}

// =============================================================================
// GetGlobal Tests
// =============================================================================

func TestCommandBlacklistRepository_GetGlobal(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	createdBy := uuid.New()

	// Create global rules
	for i := 0; i < 3; i++ {
		blacklist := &model.CommandBlacklist{
			TenantID:       nil,
			CommandPattern: fmt.Sprintf("global %d", i),
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         fmt.Sprintf("Global rule %d", i),
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	// Create tenant-specific rule
	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	tenantBlacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "tenant specific",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Tenant rule",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, tenantBlacklist)

	t.Run("get global rules only", func(t *testing.T) {
		globalRules, err := repo.GetGlobal(ctx)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(globalRules), 3)

		for _, r := range globalRules {
			assert.Nil(t, r.TenantID)
			assert.True(t, r.Enabled)
		}
	})
}

// =============================================================================
// CheckCommandAgainstBlacklist Tests
// =============================================================================

func TestCommandBlacklistRepository_CheckCommandAgainstBlacklist(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	userID := uuid.New()

	// Create block rule
	blockRule := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "rm -rf",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityCritical),
		Reason:         "Block rm -rf",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, blockRule)

	// Create warn rule
	warnRule := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "chmod",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionWarn),
		Severity:       string(model.SeverityMedium),
		Reason:         "Warn chmod",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, warnRule)

	t.Run("blocked command", func(t *testing.T) {
		allowed, action, rule := repo.CheckCommandAgainstBlacklist(ctx, tenantID, "rm -rf", []uuid.UUID{userID}, nil)

		assert.False(t, allowed)
		assert.Equal(t, "blocked", action)
		assert.NotNil(t, rule)
		assert.Equal(t, "block", rule.Action)
	})

	t.Run("warned command", func(t *testing.T) {
		allowed, action, rule := repo.CheckCommandAgainstBlacklist(ctx, tenantID, "chmod", []uuid.UUID{userID}, nil)

		assert.True(t, allowed)
		assert.Equal(t, "warned", action)
		assert.NotNil(t, rule)
		assert.Equal(t, "warn", rule.Action)
	})

	t.Run("allowed command (no match)", func(t *testing.T) {
		allowed, action, rule := repo.CheckCommandAgainstBlacklist(ctx, tenantID, "ls -la", []uuid.UUID{userID}, nil)

		assert.True(t, allowed)
		assert.Equal(t, "", action)
		assert.Nil(t, rule)
	})
}

// =============================================================================
// BatchCreate Tests
// =============================================================================

func TestCommandBlacklistRepository_BatchCreate(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	t.Run("batch create multiple entries", func(t *testing.T) {
		blacklists := make([]model.CommandBlacklist, 5)
		for i := 0; i < 5; i++ {
			blacklists[i] = model.CommandBlacklist{
				TenantID:       &tenantID,
				CommandPattern: fmt.Sprintf("batch test %d", i),
				PatternType:    string(model.PatternTypeExact),
				Action:         string(model.CommandActionBlock),
				Severity:       string(model.SeverityHigh),
				Reason:         fmt.Sprintf("Batch %d", i),
				Enabled:        true,
				CreatedBy:      createdBy,
			}
		}

		err := repo.BatchCreate(ctx, blacklists)
		require.NoError(t, err)

		// Verify all were created
		for _, b := range blacklists {
			found, err := repo.GetByID(ctx, b.ID)
			require.NoError(t, err)
			assert.Equal(t, b.CommandPattern, found.CommandPattern)
		}
	})

	t.Run("batch create empty slice", func(t *testing.T) {
		err := repo.BatchCreate(ctx, []model.CommandBlacklist{})
		require.NoError(t, err)
	})
}

// =============================================================================
// GetForUser Tests
// =============================================================================

func TestCommandBlacklistRepository_GetForUser(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()
	userID := uuid.New()
	groupID := uuid.New()

	// Create rules for specific user
	userRule := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "user-specific",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		AppliesToUsers:  []uuid.UUID{userID},
		Reason:         "User rule",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, userRule)

	// Create rules for specific group
	groupRule := &model.CommandBlacklist{
		TenantID:        &tenantID,
		CommandPattern: "group-specific",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		AppliesToGroups: []uuid.UUID{groupID},
		Reason:         "Group rule",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, groupRule)

	// Create global rule (applies to all)
	globalRule := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "all-users",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionWarn),
		Severity:       string(model.SeverityMedium),
		AppliesToUsers: nil,
		AppliesToGroups: nil,
		Reason:         "All users rule",
		Enabled:        true,
		CreatedBy:      createdBy,
	}
	_ = repo.Create(ctx, globalRule)

	t.Run("get rules for user", func(t *testing.T) {
		userGroupIDs := []uuid.UUID{groupID}
		rules, err := repo.GetForUser(ctx, tenantID, userID, userGroupIDs)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(rules), 3)
	})
}

// =============================================================================
// ExistsByPattern Tests
// =============================================================================

func TestCommandBlacklistRepository_ExistsByPattern(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	blacklist := &model.CommandBlacklist{
		TenantID:       &tenantID,
		CommandPattern: "unique pattern",
		PatternType:    string(model.PatternTypeExact),
		Action:         string(model.CommandActionBlock),
		Severity:       string(model.SeverityHigh),
		Reason:         "Unique pattern test",
		Enabled:        true,
		CreatedBy:      createdBy,
	}

	err := repo.Create(ctx, blacklist)
	require.NoError(t, err)

	t.Run("pattern exists", func(t *testing.T) {
		exists, err := repo.ExistsByPattern(ctx, &tenantID, "unique pattern", string(model.PatternTypeExact))

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("pattern does not exist", func(t *testing.T) {
		exists, err := repo.ExistsByPattern(ctx, &tenantID, "nonexistent pattern", string(model.PatternTypeExact))

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// =============================================================================
// GetByRiskCategory Tests
// =============================================================================

func TestCommandBlacklistRepository_GetByRiskCategory(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	// Create rules with different risk categories
	riskCategories := []string{
		"data_destruction",
		"privilege_escalation",
		"information_disclosure",
	}

	for _, category := range riskCategories {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: fmt.Sprintf("%s command", category),
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityCritical),
			Reason:         fmt.Sprintf("Risk: %s", category),
			Enabled:        true,
			CreatedBy:      createdBy,
		}
		_ = repo.Create(ctx, blacklist)
	}

	t.Run("get by risk category", func(t *testing.T) {
		blacklists, err := repo.GetByRiskCategory(ctx, tenantID, "data_destruction")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(blacklists), 1)

		for _, b := range blacklists {
			assert.NotNil(t, b.RiskCategory)
			assert.Equal(t, "data_destruction", *b.RiskCategory)
		}
	})
}

// =============================================================================
// Edge Cases Tests
// =============================================================================

func TestCommandBlacklistRepository_EdgeCases(t *testing.T) {
	db, repo := setupCommandBlacklistTest(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	tenantID := uuid.MustParse(opamtesting.GenerateTestTenantID())
	createdBy := uuid.New()

	t.Run("create with empty pattern", func(t *testing.T) {
		blacklist := &model.CommandBlacklist{
			TenantID:       &tenantID,
			CommandPattern: "",
			PatternType:    string(model.PatternTypeExact),
			Action:         string(model.CommandActionBlock),
			Severity:       string(model.SeverityHigh),
			Reason:         "Empty pattern",
			Enabled:        true,
			CreatedBy:      createdBy,
		}

		err := repo.Create(ctx, blacklist)
		// May fail validation or succeed
		if err != nil {
			t.Logf("Empty pattern handled: %v", err)
		}
	})

	t.Run("create with all action types", func(t *testing.T) {
		actions := []string{
			string(model.CommandActionBlock),
			string(model.CommandActionWarn),
			string(model.CommandActionAllow),
			string(model.CommandActionAudit),
		}

		for _, action := range actions {
			blacklist := &model.CommandBlacklist{
				TenantID:       &tenantID,
				CommandPattern: fmt.Sprintf("test action %s", action),
				PatternType:    string(model.PatternTypeExact),
				Action:         action,
				Severity:       string(model.SeverityMedium),
				Reason:         fmt.Sprintf("Test %s", action),
				Enabled:        true,
				CreatedBy:      createdBy,
			}

			err := repo.Create(ctx, blacklist)
			require.NoError(t, err)
		}
	})

	t.Run("create with all severities", func(t *testing.T) {
		severities := []string{
			string(model.SeverityCritical),
			string(model.SeverityHigh),
			string(model.SeverityMedium),
			string(model.SeverityLow),
		}

		for _, severity := range severities {
			blacklist := &model.CommandBlacklist{
				TenantID:       &tenantID,
				CommandPattern: fmt.Sprintf("test severity %s", severity),
				PatternType:    string(model.PatternTypeExact),
				Action:         string(model.CommandActionBlock),
				Severity:       severity,
				Reason:         fmt.Sprintf("Test %s", severity),
				Enabled:        true,
				CreatedBy:      createdBy,
			}

			err := repo.Create(ctx, blacklist)
			require.NoError(t, err)
		}
	})

	t.Run("create with all pattern types", func(t *testing.T) {
		patternTypes := []string{
			string(model.PatternTypeExact),
			string(model.PatternTypeGlob),
			string(model.PatternTypeRegex),
		}

		for _, ptype := range patternTypes {
			blacklist := &model.CommandBlacklist{
				TenantID:       &tenantID,
				CommandPattern: fmt.Sprintf("test pattern %s", ptype),
				PatternType:    ptype,
				Action:         string(model.CommandActionBlock),
				Severity:       string(model.SeverityHigh),
				Reason:         fmt.Sprintf("Test %s", ptype),
				Enabled:        true,
				CreatedBy:      createdBy,
			}

			err := repo.Create(ctx, blacklist)
			require.NoError(t, err)
		}
	})
}
