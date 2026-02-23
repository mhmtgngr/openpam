package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
)

// Service handles admin operations
type Service struct {
	db     *sqlx.DB
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewService creates a new admin service
func NewService(db *sqlx.DB, c *cache.Cache, logger zerolog.Logger) *Service {
	return &Service{db: db, cache: c, logger: logger}
}

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        uuid.UUID          `db:"id" json:"id"`
	Name      string             `db:"name" json:"name"`
	Domain    string             `db:"domain" json:"domain"`
	Plan      string             `db:"plan" json:"plan"` // free, pro, enterprise
	Config    json.RawMessage    `db:"config" json:"config"`
	Status    string             `db:"status" json:"status"` // active, suspended, cancelled
	UserLimit int                `db:"user_limit" json:"user_limit"`
	StorageGB int                `db:"storage_gb" json:"storage_gb"`
	CreatedAt time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt time.Time          `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time         `db:"deleted_at" json:"deleted_at,omitempty"`
}

// TenantConfig represents tenant configuration
type TenantConfig struct {
	SessionTimeoutMinutes int      `json:"session_timeout_minutes"`
	MFARequired          bool     `json:"mfa_required"`
	AllowedIPs           []string `json:"allowed_ips,omitempty"`
	SSOConfig            *SSOConfig `json:"sso_config,omitempty"`
	Branding             *BrandingConfig `json:"branding,omitempty"`
}

// SSOConfig represents SSO configuration
type SSOConfig struct {
	Provider   string `json:"provider"` // saml, oidc
	EntityID   string `json:"entity_id"`
	MetadataURL string `json:"metadata_url"`
}

// BrandingConfig represents branding customization
type BrandingConfig struct {
	LogoURL      string `json:"logo_url"`
	PrimaryColor string `json:"primary_color"`
	CompanyName  string `json:"company_name"`
}

// CreateTenant creates a new tenant
func (s *Service) CreateTenant(ctx context.Context, tenant *Tenant) error {
	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	tenant.Status = "active"

	// Set default limits based on plan
	s.setDefaultsForPlan(tenant)

	query := `
		INSERT INTO tenants (id, name, domain, plan, config, status, user_limit, storage_gb, created_at, updated_at)
		VALUES (:id, :name, :domain, :plan, :config, :status, :user_limit, :storage_gb, :created_at, :updated_at)
	`

	_, err := s.db.NamedExecContext(ctx, query, tenant)
	if err != nil {
		return fmt.Errorf("admin.CreateTenant: %w", err)
	}

	// Create tenant schema
	if err := s.createTenantSchema(ctx, tenant.ID); err != nil {
		s.logger.Error().Err(err).Str("tenant_id", tenant.ID.String()).Msg("Failed to create tenant schema")
	}

	s.logger.Info().
		Str("tenant_id", tenant.ID.String()).
		Str("name", tenant.Name).
		Str("plan", tenant.Plan).
		Msg("Tenant created")

	return nil
}

// GetTenant retrieves a tenant by ID
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var tenant Tenant
	query := `SELECT * FROM tenants WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.GetContext(ctx, &tenant, query, id)
	if err != nil {
		return nil, fmt.Errorf("admin.GetTenant: %w", err)
	}
	return &tenant, nil
}

// GetTenantByDomain retrieves a tenant by domain
func (s *Service) GetTenantByDomain(ctx context.Context, domain string) (*Tenant, error) {
	var tenant Tenant
	query := `SELECT * FROM tenants WHERE domain = $1 AND deleted_at IS NULL`
	err := s.db.GetContext(ctx, &tenant, query, domain)
	if err != nil {
		return nil, fmt.Errorf("admin.GetTenantByDomain: %w", err)
	}
	return &tenant, nil
}

// ListTenants lists all tenants with pagination
func (s *Service) ListTenants(ctx context.Context, limit, offset int) ([]Tenant, int, error) {
	var tenants []Tenant

	// Get count
	var total int
	countQuery := `SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL`
	if err := s.db.GetContext(ctx, &total, countQuery); err != nil {
		return nil, 0, fmt.Errorf("admin.ListTenants.Count: %w", err)
	}

	// Get tenants
	query := `
		SELECT * FROM tenants
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	err := s.db.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("admin.ListTenants: %w", err)
	}

	return tenants, total, nil
}

// UpdateTenant updates a tenant
func (s *Service) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	tenant.UpdatedAt = time.Now()

	query := `
		UPDATE tenants SET
			name = :name,
			domain = :domain,
			plan = :plan,
			config = :config,
			user_limit = :user_limit,
			storage_gb = :storage_gb,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	_, err := s.db.NamedExecContext(ctx, query, tenant)
	if err != nil {
		return fmt.Errorf("admin.UpdateTenant: %w", err)
	}

	return nil
}

// SuspendTenant suspends a tenant
func (s *Service) SuspendTenant(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE tenants
		SET status = 'suspended', updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("admin.SuspendTenant: %w", err)
	}

	s.logger.Warn().
		Str("tenant_id", id.String()).
		Str("reason", reason).
		Msg("Tenant suspended")

	return nil
}

// DeleteTenant soft deletes a tenant
func (s *Service) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tenants SET deleted_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("admin.DeleteTenant: %w", err)
	}

	return nil
}

// GetTenantUsage retrieves usage statistics for a tenant
func (s *Service) GetTenantUsage(ctx context.Context, tenantID uuid.UUID) (*TenantUsage, error) {
	usage := &TenantUsage{TenantID: tenantID}

	// Get user count
	userQuery := `
		SELECT COUNT(*) FROM users
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &usage.UserCount, userQuery, tenantID); err != nil {
		return nil, err
	}

	// Get credential count
	credentialQuery := `
		SELECT COUNT(*) FROM secrets
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &usage.CredentialCount, credentialQuery, tenantID); err != nil {
		return nil, err
	}

	// Get session count (last 30 days)
	sessionQuery := `
		SELECT COUNT(*) FROM sessions
		WHERE tenant_id = $1 AND started_at >= NOW() - INTERVAL '30 days'
	`
	if err := s.db.GetContext(ctx, &usage.SessionCount30d, sessionQuery, tenantID); err != nil {
		return nil, err
	}

	// Get storage usage (approximate)
	storageQuery := `
		SELECT COALESCE(SUM(file_size), 0) FROM recordings
		WHERE tenant_id = $1 AND deleted_at IS NULL
	`
	if err := s.db.GetContext(ctx, &usage.StorageUsedBytes, storageQuery, tenantID); err != nil {
		return nil, err
	}

	return usage, nil
}

// TenantUsage represents tenant usage statistics
type TenantUsage struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	UserCount        int       `json:"user_count"`
	CredentialCount  int       `json:"credential_count"`
	SessionCount30d  int       `json:"session_count_30d"`
	StorageUsedBytes int64     `json:"storage_used_bytes"`
}

// GetSystemStats retrieves system-wide statistics
func (s *Service) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	stats := &SystemStats{GeneratedAt: time.Now()}

	// Get tenant count
	if err := s.db.GetContext(ctx, &stats.TenantCount, `SELECT COUNT(*) FROM tenants WHERE deleted_at IS NULL`); err != nil {
		return nil, err
	}

	// Get user count
	if err := s.db.GetContext(ctx, &stats.UserCount, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`); err != nil {
		return nil, err
	}

	// Get credential count
	if err := s.db.GetContext(ctx, &stats.CredentialCount, `SELECT COUNT(*) FROM secrets WHERE deleted_at IS NULL`); err != nil {
		return nil, err
	}

	// Get active session count
	if err := s.db.GetContext(ctx, &stats.ActiveSessionCount, `SELECT COUNT(*) FROM sessions WHERE status = 'active'`); err != nil {
		return nil, err
	}

	// Get today's activity
	todayQuery := `
		SELECT
			COUNT(*) FILTER (WHERE outcome = 'success') as successful,
			COUNT(*) FILTER (WHERE outcome = 'failure') as failed,
			COUNT(*) FILTER (WHERE outcome = 'denied') as denied
		FROM audit_events
		WHERE created_at >= CURRENT_DATE
	`

	var activity struct {
		Successful int `db:"successful"`
		Failed     int `db:"failed"`
		Denied     int `db:"denied"`
	}
	if err := s.db.GetContext(ctx, &activity, todayQuery); err == nil {
		stats.TodaySuccessfulEvents = activity.Successful
		stats.TodayFailedEvents = activity.Failed
		stats.TodayDeniedEvents = activity.Denied
	}

	return stats, nil
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	GeneratedAt         time.Time `json:"generated_at"`
	TenantCount         int       `json:"tenant_count"`
	UserCount           int       `json:"user_count"`
	CredentialCount     int       `json:"credential_count"`
	ActiveSessionCount  int       `json:"active_session_count"`
	TodaySuccessfulEvents int     `json:"today_successful_events"`
	TodayFailedEvents   int       `json:"today_failed_events"`
	TodayDeniedEvents   int       `json:"today_denied_events"`
}

// setDefaultsForPlan sets default values based on plan
func (s *Service) setDefaultsForPlan(tenant *Tenant) {
	switch tenant.Plan {
	case "free":
		tenant.UserLimit = 5
		tenant.StorageGB = 1
	case "pro":
		tenant.UserLimit = 50
		tenant.StorageGB = 10
	case "enterprise":
		tenant.UserLimit = -1 // unlimited
		tenant.StorageGB = 100
	default:
		tenant.UserLimit = 10
		tenant.StorageGB = 5
	}
}

// createTenantSchema creates database schema for a tenant
func (s *Service) createTenantSchema(ctx context.Context, tenantID uuid.UUID) error {
	// Create tenant-specific schema if needed
	// For multi-tenancy with schema isolation
	schemaName := fmt.Sprintf("tenant_%s", tenantID.String())

	query := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schemaName)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("admin.CreateSchema: %w", err)
	}

	return nil
}

// GetTenantConfig retrieves tenant configuration
func (s *Service) GetTenantConfig(ctx context.Context, tenantID uuid.UUID) (*TenantConfig, error) {
	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var config TenantConfig
	if tenant.Config != nil {
		if err := json.Unmarshal(tenant.Config, &config); err != nil {
			return nil, err
		}
	}

	return &config, nil
}

// UpdateTenantConfig updates tenant configuration
func (s *Service) UpdateTenantConfig(ctx context.Context, tenantID uuid.UUID, config *TenantConfig) error {
	tenant, err := s.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	tenant.Config = data
	return s.UpdateTenant(ctx, tenant)
}
