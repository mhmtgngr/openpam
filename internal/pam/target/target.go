package target

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// TargetType represents the type of target system
type TargetType string

const (
	TargetTypeSSH       TargetType = "ssh"
	TargetTypeRDP       TargetType = "rdp"
	TargetTypeDatabase  TargetType = "database"
	TargetTypeWeb       TargetType = "web"
	TargetTypeKubernetes TargetType = "kubernetes"
	TargetTypeAPI       TargetType = "api"
)

// TargetEnvironment represents the environment tier
type TargetEnvironment string

const (
	EnvironmentProduction  TargetEnvironment = "production"
	EnvironmentStaging     TargetEnvironment = "staging"
	EnvironmentDevelopment TargetEnvironment = "development"
	EnvironmentTest       TargetEnvironment = "test"
)

// TargetSensitivity represents the sensitivity level
type TargetSensitivity string

const (
	SensitivityCritical TargetSensitivity = "critical"
	SensitivityHigh     TargetSensitivity = "high"
	SensitivityMedium   TargetSensitivity = "medium"
	SensitivityLow      TargetSensitivity = "low"
)

// Target represents a target system that can be accessed
type Target struct {
	ID              uuid.UUID           `db:"id" json:"id"`
	Name            string              `db:"name" json:"name"`
	Description     string              `db:"description" json:"description"`
	Type            TargetType          `db:"type" json:"type"`
	Environment     TargetEnvironment   `db:"environment" json:"environment"`
	Sensitivity     TargetSensitivity   `db:"sensitivity" json:"sensitivity"`
	TenantID        uuid.UUID           `db:"tenant_id" json:"tenant_id"`

	// Connection details
	Host            string              `db:"host" json:"host"`
	Port            int                 `db:"port" json:"port"`
	ConnectionString string             `db:"connection_string" json:"connection_string,omitempty"`

	// Access control
	RequireApproval bool                `db:"require_approval" json:"require_approval"`
	RequireMFA      bool                `db:"require_mfa" json:"require_mfa"`
	MaxDuration     int                 `db:"max_duration_minutes" json:"max_duration_minutes"`
	ApprovalGroupID *uuid.UUID          `db:"approval_group_id" json:"approval_group_id,omitempty"`

	// Configuration
	Platform        string              `db:"platform" json:"platform"` // e.g., "linux", "windows", "postgresql"
	Tags            []string            `db:"tags" json:"tags"`
	Metadata        map[string]interface{} `db:"metadata" json:"metadata"`

	// Status
	Status          string              `db:"status" json:"status"` // active, inactive, maintenance
	LastVerifiedAt  *time.Time          `db:"last_verified_at" json:"last_verified_at"`

	CreatedAt       time.Time           `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at" json:"updated_at"`
	DeletedAt       *time.Time          `db:"deleted_at" json:"deleted_at,omitempty"`
}

// TargetFilter filters target queries
type TargetFilter struct {
	Type         *TargetType
	Environment  *TargetEnvironment
	Sensitivity  *TargetSensitivity
	Status       *string
	RequireApproval *bool
	Tags         []string
	SearchTerm   string
}

// TargetRepository handles target data operations
type TargetRepository struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewTargetRepository creates a new target repository
func NewTargetRepository(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *TargetRepository {
	return &TargetRepository{db: db, cache: c, logger: logger}
}

// Create creates a new target
func (r *TargetRepository) Create(ctx context.Context, target *Target) error {
	target.ID = uuid.New()
	target.CreatedAt = time.Now()
	target.UpdatedAt = time.Now()
	target.Status = "active"

	query := `
		INSERT INTO targets (id, name, description, type, environment, sensitivity,
			tenant_id, host, port, connection_string, require_approval, require_mfa,
			max_duration_minutes, approval_group_id, platform, tags, metadata, status,
			created_at, updated_at)
		VALUES (:id, :name, :description, :type, :environment, :sensitivity,
			:tenant_id, :host, :port, :connection_string, :require_approval, :require_mfa,
			:max_duration_minutes, :approval_group_id, :platform, :tags, :metadata, :status,
			:created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, target)
	if err != nil {
		return fmt.Errorf("target.Create: %w", err)
	}

	// Invalidate cache
	r.invalidateCache(ctx, target.TenantID)

	return nil
}

// GetByID retrieves a target by ID
func (r *TargetRepository) GetByID(ctx context.Context, id uuid.UUID) (*Target, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("target:%s", id)
	var target Target
	if err := r.cache.Get(ctx, cacheKey, &target); err == nil {
		return &target, nil
	}

	query := `SELECT * FROM targets WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &target, query, id)
	if err != nil {
		return nil, fmt.Errorf("target.GetByID: %w", err)
	}

	// Cache result
	_ = r.cache.Set(ctx, cacheKey, target, 5*time.Minute)

	return &target, nil
}

// List retrieves targets with filtering and pagination
func (r *TargetRepository) List(ctx context.Context, tenantID uuid.UUID, filter TargetFilter, limit, offset int) ([]Target, int, error) {
	// Build query with filters
	baseQuery := `
		SELECT * FROM targets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	countQuery := `
		SELECT COUNT(*) FROM targets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`

	args := []interface{}{tenantID}
	argCount := 2

	// Apply filters
	if filter.Type != nil {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		countQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, *filter.Type)
		argCount++
	}
	if filter.Environment != nil {
		baseQuery += fmt.Sprintf(" AND environment = $%d", argCount)
		countQuery += fmt.Sprintf(" AND environment = $%d", argCount)
		args = append(args, *filter.Environment)
		argCount++
	}
	if filter.Sensitivity != nil {
		baseQuery += fmt.Sprintf(" AND sensitivity = $%d", argCount)
		countQuery += fmt.Sprintf(" AND sensitivity = $%d", argCount)
		args = append(args, *filter.Sensitivity)
		argCount++
	}
	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.RequireApproval != nil {
		baseQuery += fmt.Sprintf(" AND require_approval = $%d", argCount)
		countQuery += fmt.Sprintf(" AND require_approval = $%d", argCount)
		args = append(args, *filter.RequireApproval)
		argCount++
	}
	if filter.SearchTerm != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR host ILIKE $%d)", argCount, argCount, argCount)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR host ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+filter.SearchTerm+"%")
		argCount++
	}

	// Get total count
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("target.List.Count: %w", err)
	}

	// Add pagination
	baseQuery += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	var targets []Target
	if err := r.db.SelectContext(ctx, &targets, baseQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("target.List: %w", err)
	}

	return targets, total, nil
}

// Update updates a target
func (r *TargetRepository) Update(ctx context.Context, target *Target) error {
	target.UpdatedAt = time.Now()

	query := `
		UPDATE targets SET
			name = :name,
			description = :description,
			type = :type,
			environment = :environment,
			sensitivity = :sensitivity,
			host = :host,
			port = :port,
			connection_string = :connection_string,
			require_approval = :require_approval,
			require_mfa = :require_mfa,
			max_duration_minutes = :max_duration_minutes,
			approval_group_id = :approval_group_id,
			platform = :platform,
			tags = :tags,
			metadata = :metadata,
			status = :status,
			last_verified_at = :last_verified_at,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	_, err := r.db.NamedExecContext(ctx, query, target)
	if err != nil {
		return fmt.Errorf("target.Update: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("target:%s", target.ID)
	_ = r.cache.Delete(ctx, cacheKey)
	r.invalidateCache(ctx, target.TenantID)

	return nil
}

// Delete soft deletes a target
func (r *TargetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE targets SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("target.Delete: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("target:%s", id)
	_ = r.cache.Delete(ctx, cacheKey)

	return nil
}

// VerifyConnection verifies connectivity to a target
func (r *TargetRepository) VerifyConnection(ctx context.Context, target *Target) error {
	// Implementation depends on target type
	// For SSH targets, we'd attempt an SSH connection
	// For RDP, we'd check RDP port
	// For databases, we'd attempt a connection

	// Basic connectivity check
	if target.Type == TargetTypeSSH || target.Type == TargetTypeRDP {
		// Check if port is open using TCP connection
		timeout := 5 * time.Second
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", target.Host, target.Port), timeout)
		if err != nil {
			return fmt.Errorf("target.VerifyConnection: %w", err)
		}
		conn.Close()
	}

	// Update last verified time
	now := time.Now()
	target.LastVerifiedAt = &now
	return r.Update(ctx, target)
}

// GetByEnvironment retrieves targets by environment
func (r *TargetRepository) GetByEnvironment(ctx context.Context, tenantID uuid.UUID, env TargetEnvironment) ([]Target, error) {
	var targets []Target
	query := `
		SELECT * FROM targets
		WHERE tenant_id = $1 AND environment = $2 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &targets, query, tenantID, env)
	if err != nil {
		return nil, fmt.Errorf("target.GetByEnvironment: %w", err)
	}
	return targets, nil
}

// GetBySensitivity retrieves targets by sensitivity level
func (r *TargetRepository) GetBySensitivity(ctx context.Context, tenantID uuid.UUID, sensitivity TargetSensitivity) ([]Target, error) {
	var targets []Target
	query := `
		SELECT * FROM targets
		WHERE tenant_id = $1 AND sensitivity = $2 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &targets, query, tenantID, sensitivity)
	if err != nil {
		return nil, fmt.Errorf("target.GetBySensitivity: %w", err)
	}
	return targets, nil
}

// GetApprovalGroup retrieves targets that require approval from a specific group
func (r *TargetRepository) GetByApprovalGroup(ctx context.Context, approvalGroupID uuid.UUID) ([]Target, error) {
	var targets []Target
	query := `
		SELECT * FROM targets
		WHERE approval_group_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	err := r.db.SelectContext(ctx, &targets, query, approvalGroupID)
	if err != nil {
		return nil, fmt.Errorf("target.GetByApprovalGroup: %w", err)
	}
	return targets, nil
}

// invalidateCache invalidates tenant-specific cache
func (r *TargetRepository) invalidateCache(ctx context.Context, tenantID uuid.UUID) {
	cacheKey := fmt.Sprintf("targets:%s", tenantID)
	_ = r.cache.Delete(ctx, cacheKey)
}

// TargetService handles target business logic
type TargetService struct {
	repo   *TargetRepository
	logger zerolog.Logger
}

// NewTargetService creates a new target service
func NewTargetService(repo *TargetRepository, logger zerolog.Logger) *TargetService {
	return &TargetService{repo: repo, logger: logger}
}

// CreateTarget creates a new target with validation
func (s *TargetService) CreateTarget(ctx context.Context, target *Target) error {
	// Validate target
	if err := s.validateTarget(target); err != nil {
		return err
	}

	// Set default values
	if target.MaxDuration == 0 {
		switch target.Sensitivity {
		case SensitivityCritical:
			target.MaxDuration = 60 // 1 hour max for critical
		case SensitivityHigh:
			target.MaxDuration = 240 // 4 hours max for high
		case SensitivityMedium:
			target.MaxDuration = 480 // 8 hours max for medium
		default:
			target.MaxDuration = 1440 // 24 hours for low
		}
	}

	// Force MFA for critical targets
	if target.Sensitivity == SensitivityCritical {
		target.RequireMFA = true
		target.RequireApproval = true
	}

	return s.repo.Create(ctx, target)
}

// UpdateTarget updates an existing target
func (s *TargetService) UpdateTarget(ctx context.Context, target *Target) error {
	// Validate target
	if err := s.validateTarget(target); err != nil {
		return err
	}

	return s.repo.Update(ctx, target)
}

// DeleteTarget deletes a target
func (s *TargetService) DeleteTarget(ctx context.Context, id uuid.UUID) error {
	// Check if target has active credentials or sessions
	// (Would implement this check)

	return s.repo.Delete(ctx, id)
}

// VerifyTarget verifies connectivity to a target
func (s *TargetService) VerifyTarget(ctx context.Context, id uuid.UUID) error {
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.VerifyConnection(ctx, target)
}

// GetTargetsForUser retrieves targets a user can access
func (s *TargetService) GetTargetsForUser(ctx context.Context, userID, tenantID uuid.UUID, filter TargetFilter, limit, offset int) ([]Target, int, error) {
	// This would integrate with RBAC to filter targets based on user permissions
	// For now, return all tenant targets
	return s.repo.List(ctx, tenantID, filter, limit, offset)
}

// validateTarget validates target data
func (s *TargetService) validateTarget(target *Target) error {
	if target.Name == "" {
		return fmt.Errorf("target: name is required")
	}
	if target.Type == "" {
		return fmt.Errorf("target: type is required")
	}
	if target.Host == "" {
		return fmt.Errorf("target: host is required")
	}
	if target.Port <= 0 || target.Port > 65535 {
		return fmt.Errorf("target: invalid port")
	}
	if target.TenantID == uuid.Nil {
		return fmt.Errorf("target: tenant ID is required")
	}

	// Validate connection string for database targets
	if target.Type == TargetTypeDatabase && target.ConnectionString == "" {
		return fmt.Errorf("target: connection string required for database targets")
	}

	return nil
}

// ApprovalGroup represents a group of approvers
type ApprovalGroup struct {
	ID          uuid.UUID   `db:"id" json:"id"`
	Name        string      `db:"name" json:"name"`
	Description string      `db:"description" json:"description"`
	TenantID    uuid.UUID   `db:"tenant_id" json:"tenant_id"`
	Members     []uuid.UUID `db:"members" json:"members"`
	Required    int         `db:"required_approvals" json:"required_approvals"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
}

// CreateApprovalGroup creates a new approval group
func (s *TargetService) CreateApprovalGroup(ctx context.Context, group *ApprovalGroup) error {
	group.ID = uuid.New()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	query := `
		INSERT INTO approval_groups (id, name, description, tenant_id, members, required_approvals, created_at, updated_at)
		VALUES (:id, :name, :description, :tenant_id, :members, :required_approvals, :created_at, :updated_at)
	`

	_, err := s.repo.db.NamedExecContext(ctx, query, group)
	if err != nil {
		return fmt.Errorf("target.CreateApprovalGroup: %w", err)
	}

	return nil
}
