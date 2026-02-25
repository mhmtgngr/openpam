// Package repository provides data access layer for audit analytics
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/audit/model"
	"github.com/rs/zerolog"
)

// CommandBlacklistRepository handles command blacklist management
type CommandBlacklistRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewCommandBlacklistRepository creates a new command blacklist repository
func NewCommandBlacklistRepository(db *sqlx.DB, logger zerolog.Logger) *CommandBlacklistRepository {
	return &CommandBlacklistRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new command blacklist entry
func (r *CommandBlacklistRepository) Create(ctx context.Context, blacklist *model.CommandBlacklist) error {
	blacklist.ID = uuid.New()
	blacklist.CreatedAt = time.Now()
	blacklist.UpdatedAt = time.Now()

	query := `
		INSERT INTO command_blacklist (
			id, tenant_id, command_pattern, pattern_type, base_command, action, severity,
			applies_to_users, applies_to_groups, applies_to_targets, allow_override,
			override_roles, reason, risk_category, enabled, created_by, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :command_pattern, :pattern_type, :base_command, :action, :severity,
			:applies_to_users, :applies_to_groups, :applies_to_targets, :allow_override,
			:override_roles, :reason, :risk_category, :enabled, :created_by, :created_at, :updated_at
		)
	`

	_, err := r.db.NamedExecContext(ctx, query, blacklist)
	if err != nil {
		return fmt.Errorf("command_blacklist.Create: %w", err)
	}

	r.logger.Info().
		Str("blacklist_id", blacklist.ID.String()).
		Str("pattern", blacklist.CommandPattern).
		Str("action", blacklist.Action).
		Msg("Command blacklist created")

	return nil
}

// GetByID retrieves a command blacklist entry by ID
func (r *CommandBlacklistRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.CommandBlacklist, error) {
	var blacklist model.CommandBlacklist
	query := `SELECT * FROM command_blacklist WHERE id = $1`

	err := r.db.GetContext(ctx, &blacklist, query, id)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetByID: %w", err)
	}

	return &blacklist, nil
}

// List retrieves command blacklist entries with filtering
func (r *CommandBlacklistRepository) List(ctx context.Context, filter model.CommandBlacklistFilter) ([]model.CommandBlacklist, error) {
	baseQuery := `SELECT * FROM command_blacklist WHERE 1=1`
	args := []interface{}{}
	argCount := 1

	if filter.TenantID != nil {
		baseQuery += fmt.Sprintf(" AND (tenant_id IS NULL OR tenant_id = $%d)", argCount)
		args = append(args, *filter.TenantID)
		argCount++
	}

	if filter.Enabled != nil {
		baseQuery += fmt.Sprintf(" AND enabled = $%d", argCount)
		args = append(args, *filter.Enabled)
		argCount++
	}

	if filter.PatternType != nil {
		baseQuery += fmt.Sprintf(" AND pattern_type = $%d", argCount)
		args = append(args, *filter.PatternType)
		argCount++
	}

	if filter.Action != nil {
		baseQuery += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, *filter.Action)
		argCount++
	}

	if filter.Severity != nil {
		baseQuery += fmt.Sprintf(" AND severity = $%d", argCount)
		args = append(args, *filter.Severity)
		argCount++
	}

	// Order: tenant-specific first, then global, then by created date
	baseQuery += " ORDER BY tenant_id NULLS LAST, created_at DESC"

	var blacklists []model.CommandBlacklist
	err := r.db.SelectContext(ctx, &blacklists, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.List: %w", err)
	}

	return blacklists, nil
}

// Update updates a command blacklist entry
func (r *CommandBlacklistRepository) Update(ctx context.Context, blacklist *model.CommandBlacklist) error {
	blacklist.UpdatedAt = time.Now()

	query := `
		UPDATE command_blacklist SET
			command_pattern = :command_pattern,
			pattern_type = :pattern_type,
			base_command = :base_command,
			action = :action,
			severity = :severity,
			applies_to_users = :applies_to_users,
			applies_to_groups = :applies_to_groups,
			applies_to_targets = :applies_to_targets,
			allow_override = :allow_override,
			override_roles = :override_roles,
			reason = :reason,
			risk_category = :risk_category,
			enabled = :enabled,
			updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExecContext(ctx, query, blacklist)
	if err != nil {
		return fmt.Errorf("command_blacklist.Update: %w", err)
	}

	return nil
}

// Delete deletes a command blacklist entry (soft delete by disabling)
func (r *CommandBlacklistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE command_blacklist SET enabled = false, updated_at = NOW() WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("command_blacklist.Delete: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("command_blacklist.Delete: no rows affected")
	}

	return nil
}

// DeleteHard permanently deletes a command blacklist entry
func (r *CommandBlacklistRepository) DeleteHard(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM command_blacklist WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("command_blacklist.DeleteHard: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("command_blacklist.DeleteHard: no rows affected")
	}

	return nil
}

// FindMatching finds all blacklist entries that match a command for a user
func (r *CommandBlacklistRepository) FindMatching(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]model.CommandBlacklist, error) {
	// Get all applicable blacklist rules for this tenant
	blacklists, err := r.List(ctx, model.CommandBlacklistFilter{
		TenantID: &tenantID,
		Enabled:  boolPtr(true),
	})
	if err != nil {
		return nil, err
	}

	var matches []model.CommandBlacklist

	for _, bl := range blacklists {
		// Check if rule applies to this user/group
		if !r.appliesToEntity(&bl, userIDs, groupIDs) {
			continue
		}

		// Check if command matches pattern
		if r.commandMatchesPattern(command, &bl) {
			matches = append(matches, bl)
		}
	}

	return matches, nil
}

// appliesToEntity checks if a blacklist rule applies to the given user/groups
func (r *CommandBlacklistRepository) appliesToEntity(bl *model.CommandBlacklist, userIDs, groupIDs []uuid.UUID) bool {
	// If no user restrictions, rule applies to everyone
	if len(bl.AppliesToUsers) == 0 && len(bl.AppliesToGroups) == 0 {
		return true
	}

	// Check user match
	if len(bl.AppliesToUsers) > 0 {
		for _, uid := range userIDs {
			for _, allowedUID := range bl.AppliesToUsers {
				if uid == allowedUID {
					return true
				}
			}
		}
	}

	// Check group match
	if len(bl.AppliesToGroups) > 0 {
		for _, gid := range groupIDs {
			for _, allowedGID := range bl.AppliesToGroups {
				if gid == allowedGID {
					return true
				}
			}
		}
	}

	return false
}

// commandMatchesPattern checks if a command matches the blacklist pattern
func (r *CommandBlacklistRepository) commandMatchesPattern(command string, bl *model.CommandBlacklist) bool {
	switch bl.PatternType {
	case string(model.PatternTypeExact):
		return command == bl.CommandPattern

	case string(model.PatternTypeGlob):
		// Use proper glob matching with filepath.Match
		matched, _ := r.globMatch(command, bl.CommandPattern)
		return matched

	case string(model.PatternTypeRegex):
		// Use proper regex matching
		matched, _ := r.regexMatch(command, bl.CommandPattern)
		return matched
	}

	return false
}

// auditRegexCache caches compiled regex patterns for performance
var (
	auditRegexCache = make(map[string]*regexp.Regexp)
	auditRegexMu    sync.RWMutex
)

// globMatch performs glob pattern matching using filepath.Match
func (r *CommandBlacklistRepository) globMatch(command, pattern string) (bool, error) {
	// filepath.Match requires pattern to be a valid glob pattern
	// It supports * (matches any sequence) and ? (matches single character)
	matched, err := filepath.Match(pattern, command)
	if err != nil {
		// Invalid pattern - treat as no match
		return false, nil
	}
	return matched, nil
}

// regexMatch performs regex pattern matching with caching
func (r *CommandBlacklistRepository) regexMatch(command, pattern string) (bool, error) {
	auditRegexMu.RLock()
	re, exists := auditRegexCache[pattern]
	auditRegexMu.RUnlock()

	if !exists {
		var err error
		re, err = regexp.Compile(pattern)
		if err != nil {
			r.logger.Warn().Err(err).
				Str("pattern", pattern).
				Msg("Invalid regex pattern in blacklist")
			return false, nil
		}

		auditRegexMu.Lock()
		auditRegexCache[pattern] = re
		auditRegexMu.Unlock()
	}

	return re.MatchString(command), nil
}

// GetByAction retrieves blacklist entries by action type
func (r *CommandBlacklistRepository) GetByAction(ctx context.Context, tenantID *uuid.UUID, action string) ([]model.CommandBlacklist, error) {
	var blacklists []model.CommandBlacklist
	var query string
	var args []interface{}

	if tenantID == nil {
		query = `
			SELECT * FROM command_blacklist
			WHERE action = $1 AND enabled = true AND tenant_id IS NULL
			ORDER BY created_at DESC
		`
		args = []interface{}{action}
	} else {
		query = `
			SELECT * FROM command_blacklist
			WHERE action = $1 AND enabled = true AND (tenant_id IS NULL OR tenant_id = $2)
			ORDER BY tenant_id NULLS LAST, created_at DESC
		`
		args = []interface{}{action, *tenantID}
	}

	err := r.db.SelectContext(ctx, &blacklists, query, args...)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetByAction: %w", err)
	}

	return blacklists, nil
}

// GetByBaseCommand retrieves blacklist entries by base command
func (r *CommandBlacklistRepository) GetByBaseCommand(ctx context.Context, tenantID uuid.UUID, baseCommand string) ([]model.CommandBlacklist, error) {
	var blacklists []model.CommandBlacklist
	query := `
		SELECT * FROM command_blacklist
		WHERE base_command = $1 AND enabled = true AND (tenant_id IS NULL OR tenant_id = $2)
		ORDER BY tenant_id NULLS LAST, severity DESC
	`

	err := r.db.SelectContext(ctx, &blacklists, query, baseCommand, tenantID)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetByBaseCommand: %w", err)
	}

	return blacklists, nil
}

// GetStats returns statistics about command blacklist for a tenant
func (r *CommandBlacklistRepository) GetStats(ctx context.Context, tenantID uuid.UUID) (*BlacklistStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE enabled) as total_enabled,
			COUNT(*) FILTER (WHERE NOT enabled) as total_disabled,
			COUNT(*) FILTER (WHERE action = 'block') as block_count,
			COUNT(*) FILTER (WHERE action = 'warn') as warn_count,
			COUNT(*) FILTER (WHERE action = 'audit') as audit_count,
			COUNT(*) FILTER (WHERE severity = 'critical') as critical_count,
			COUNT(*) FILTER (WHERE severity = 'high') as high_count,
			COUNT(*) FILTER (WHERE tenant_id IS NULL) as global_count
		FROM command_blacklist
		WHERE tenant_id = $1 OR tenant_id IS NULL
	`

	var stats BlacklistStats
	err := r.db.GetContext(ctx, &stats, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetStats: %w", err)
	}

	return &stats, nil
}

// BlacklistStats represents statistics about command blacklist
type BlacklistStats struct {
	TotalEnabled  int `db:"total_enabled"`
	TotalDisabled int `db:"total_disabled"`
	BlockCount    int `db:"block_count"`
	WarnCount     int `db:"warn_count"`
	AuditCount    int `db:"audit_count"`
	CriticalCount int `db:"critical_count"`
	HighCount     int `db:"high_count"`
	GlobalCount   int `db:"global_count"`
}

// GetByRiskCategory retrieves blacklist entries by risk category
func (r *CommandBlacklistRepository) GetByRiskCategory(ctx context.Context, tenantID uuid.UUID, riskCategory string) ([]model.CommandBlacklist, error) {
	var blacklists []model.CommandBlacklist
	query := `
		SELECT * FROM command_blacklist
		WHERE risk_category = $1 AND enabled = true AND (tenant_id IS NULL OR tenant_id = $2)
		ORDER BY severity DESC, created_at DESC
	`

	err := r.db.SelectContext(ctx, &blacklists, query, riskCategory, tenantID)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetByRiskCategory: %w", err)
	}

	return blacklists, nil
}

// Enable enables a command blacklist entry
func (r *CommandBlacklistRepository) Enable(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE command_blacklist SET enabled = true, updated_at = NOW() WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("command_blacklist.Enable: %w", err)
	}

	return nil
}

// Disable disables a command blacklist entry
func (r *CommandBlacklistRepository) Disable(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE command_blacklist SET enabled = false, updated_at = NOW() WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("command_blacklist.Disable: %w", err)
	}

	return nil
}

// GetGlobal retrieves global (tenant_id IS NULL) blacklist entries
func (r *CommandBlacklistRepository) GetGlobal(ctx context.Context) ([]model.CommandBlacklist, error) {
	var blacklists []model.CommandBlacklist
	query := `
		SELECT * FROM command_blacklist
		WHERE tenant_id IS NULL AND enabled = true
		ORDER BY severity DESC, created_at DESC
	`

	err := r.db.SelectContext(ctx, &blacklists, query)
	if err != nil {
		return nil, fmt.Errorf("command_blacklist.GetGlobal: %w", err)
	}

	return blacklists, nil
}

// Helper function

func boolPtr(b bool) *bool {
	return &b
}

// CheckCommandAgainstBlacklist checks if a command matches any blacklist rule and returns the action to take
func (r *CommandBlacklistRepository) CheckCommandAgainstBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) (bool, string, *model.CommandBlacklist) {
	matches, err := r.FindMatching(ctx, tenantID, command, userIDs, groupIDs)
	if err != nil || len(matches) == 0 {
		return true, "", nil
	}

	// Return the first (highest priority) match
	bl := matches[0]

	// Determine if action allows the command
	switch bl.Action {
	case string(model.CommandActionBlock):
		return false, "blocked", &bl
	case string(model.CommandActionWarn):
		return true, "warned", &bl
	case string(model.CommandActionAudit):
		return true, "audited", &bl
	default:
		return true, "", nil
	}
}

// BatchCreate creates multiple command blacklist entries in a single transaction
func (r *CommandBlacklistRepository) BatchCreate(ctx context.Context, blacklists []model.CommandBlacklist) error {
	if len(blacklists) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("command_blacklist.BatchCreate: begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	query := `
		INSERT INTO command_blacklist (
			id, tenant_id, command_pattern, pattern_type, base_command, action, severity,
			applies_to_users, applies_to_groups, applies_to_targets, allow_override,
			override_roles, reason, risk_category, enabled, created_by, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :command_pattern, :pattern_type, :base_command, :action, :severity,
			:applies_to_users, :applies_to_groups, :applies_to_targets, :allow_override,
			:override_roles, :reason, :risk_category, :enabled, :created_by, :created_at, :updated_at
		)
	`

	for i := range blacklists {
		blacklists[i].ID = uuid.New()
		blacklists[i].CreatedAt = now
		blacklists[i].UpdatedAt = now

		_, err := tx.NamedExec(query, &blacklists[i])
		if err != nil {
			return fmt.Errorf("command_blacklist.BatchCreate: insert entry %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("command_blacklist.BatchCreate: commit: %w", err)
	}

	r.logger.Info().
		Int("count", len(blacklists)).
		Msg("Batch command blacklist entries created")

	return nil
}

// GetForUser retrieves blacklist entries that apply to a specific user
func (r *CommandBlacklistRepository) GetForUser(ctx context.Context, tenantID, userID uuid.UUID, userGroupIDs []uuid.UUID) ([]model.CommandBlacklist, error) {
	// First get all enabled rules for the tenant
	allRules, err := r.List(ctx, model.CommandBlacklistFilter{
		TenantID: &tenantID,
		Enabled:  boolPtr(true),
	})
	if err != nil {
		return nil, err
	}

	var applicableRules []model.CommandBlacklist
	userIDs := []uuid.UUID{userID}

	for _, rule := range allRules {
		if r.appliesToEntity(&rule, userIDs, userGroupIDs) {
			applicableRules = append(applicableRules, rule)
		}
	}

	return applicableRules, nil
}

// ExistsByPattern checks if a blacklist entry with the same pattern already exists
func (r *CommandBlacklistRepository) ExistsByPattern(ctx context.Context, tenantID *uuid.UUID, pattern, patternType string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM command_blacklist
			WHERE command_pattern = $1 AND pattern_type = $2
			  AND (tenant_id = $3 OR (tenant_id IS NULL AND $3 IS NULL))
		)
	`

	err := r.db.GetContext(ctx, &exists, query, pattern, patternType, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("command_blacklist.ExistsByPattern: %w", err)
	}

	return exists, nil
}
