package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// User represents a user in the system
type User struct {
	ID            uuid.UUID `json:"id" db:"id"`
	TenantID      uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Email         string    `json:"email" db:"email"`
	FirstName     string    `json:"first_name" db:"first_name"`
	LastName      string    `json:"last_name" db:"last_name"`
	DisplayName   string    `json:"display_name" db:"display_name"`
	Department    string    `json:"department" db:"department"`
	Title         string    `json:"title" db:"title"`
	ManagerID     *uuid.UUID `json:"manager_id" db:"manager_id"`
	EmployeeID    string    `json:"employee_id" db:"employee_id"`
	Status        string    `json:"status" db:"status"`
	MFAPhone      string    `json:"mfa_phone" db:"mfa_phone"`
	MFAEmail      string    `json:"mfa_email" db:"mfa_email"`
	MFAEnabled    bool      `json:"mfa_enabled" db:"mfa_enabled"`
	Roles         []string  `json:"roles" db:"roles"`
	Groups        []string  `json:"groups" db:"groups"`
}

// UserRepository handles database operations for users
type UserRepository struct {
	db     *sqlx.DB
	logger *zerolog.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB, logger *zerolog.Logger) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
			department, title, manager_id, employee_id, status,
			mfa_phone, mfa_email, mfa_enabled, roles, groups
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("user.GetByID: %w", err)
	}

	return &user, nil
}

// GetByTenantID retrieves users by tenant ID
func (r *UserRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) ([]User, error) {
	query := `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
			department, title, manager_id, employee_id, status,
			mfa_phone, mfa_email, mfa_enabled, roles, groups
		FROM users
		WHERE tenant_id = $1 AND deleted_at IS NULL AND status = 'active'
		ORDER BY display_name ASC
	`

	var users []User
	err := r.db.SelectContext(ctx, &users, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("user.GetByTenantID: %w", err)
	}

	return users, nil
}

// GetManager retrieves the manager of a given user
func (r *UserRepository) GetManager(ctx context.Context, userID uuid.UUID) (*User, error) {
	query := `
		SELECT u.id, u.tenant_id, u.email, u.first_name, u.last_name, u.display_name,
			u.department, u.title, u.manager_id, u.employee_id, u.status,
			u.mfa_phone, u.mfa_email, u.mfa_enabled, u.roles, u.groups
		FROM users u
		INNER JOIN users sub ON sub.manager_id = u.id
		WHERE sub.id = $1 AND u.deleted_at IS NULL
		LIMIT 1
	`

	var user User
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No manager found
		}
		return nil, fmt.Errorf("user.GetManager: %w", err)
	}

	return &user, nil
}

// GetByRole retrieves users by role within a tenant
func (r *UserRepository) GetByRole(ctx context.Context, tenantID uuid.UUID, role string) ([]User, error) {
	query := `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
			department, title, manager_id, employee_id, status,
			mfa_phone, mfa_email, mfa_enabled, roles, groups
		FROM users
		WHERE tenant_id = $1
			AND $2 = ANY(roles)
			AND deleted_at IS NULL
			AND status = 'active'
		ORDER BY display_name ASC
	`

	var users []User
	err := r.db.SelectContext(ctx, &users, query, tenantID, role)
	if err != nil {
		return nil, fmt.Errorf("user.GetByRole: %w", err)
	}

	return users, nil
}

// GetByGroup retrieves users by group within a tenant
func (r *UserRepository) GetByGroup(ctx context.Context, tenantID uuid.UUID, group string) ([]User, error) {
	query := `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
			department, title, manager_id, employee_id, status,
			mfa_phone, mfa_email, mfa_enabled, roles, groups
		FROM users
		WHERE tenant_id = $1
			AND $2 = ANY(groups)
			AND deleted_at IS NULL
			AND status = 'active'
		ORDER BY display_name ASC
	`

	var users []User
	err := r.db.SelectContext(ctx, &users, query, tenantID, group)
	if err != nil {
		return nil, fmt.Errorf("user.GetByGroup: %w", err)
	}

	return users, nil
}

// HasRole checks if a user has a specific role
func (r *UserRepository) HasRole(ctx context.Context, userID uuid.UUID, role string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM users
			WHERE id = $1 AND $2 = ANY(roles) AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, userID, role)
	if err != nil {
		return false, fmt.Errorf("user.HasRole: %w", err)
	}

	return exists, nil
}

// GetDirectReports retrieves users that report to a manager
func (r *UserRepository) GetDirectReports(ctx context.Context, managerID uuid.UUID) ([]User, error) {
	query := `
		SELECT id, tenant_id, email, first_name, last_name, display_name,
			department, title, manager_id, employee_id, status,
			mfa_phone, mfa_email, mfa_enabled, roles, groups
		FROM users
		WHERE manager_id = $1 AND deleted_at IS NULL AND status = 'active'
		ORDER BY display_name ASC
	`

	var users []User
	err := r.db.SelectContext(ctx, &users, query, managerID)
	if err != nil {
		return nil, fmt.Errorf("user.GetDirectReports: %w", err)
	}

	return users, nil
}
