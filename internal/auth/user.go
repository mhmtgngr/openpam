package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
)

// User represents a user with additional auth fields
type User struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	FirstName    string     `db:"first_name" json:"first_name"`
	LastName     string     `db:"last_name" json:"last_name"`
	PasswordHash string     `db:"password_hash" json:"-"`
	Status       string     `db:"status" json:"status"` // active, suspended, locked
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	MFAEnabled   bool       `db:"mfa_enabled" json:"mfa_enabled"`
	MFASecret    string     `db:"mfa_secret" json:"-"`            // TOTP secret
	BackupCodes  string     `db:"backup_codes" json:"-"`          // Hashed backup codes
	LastLoginAt  *time.Time `db:"last_login_at" json:"last_login_at"`
	FailedLogins int        `db:"failed_logins" json:"failed_logins"`
	LockedUntil  *time.Time `db:"locked_until" json:"locked_until,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// UserRepository handles user data operations
type UserRepository struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB, logger zerolog.Logger) *UserRepository {
	return &UserRepository{db: db, logger: logger}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Status = "active"

	query := `
		INSERT INTO users (id, email, first_name, last_name, password_hash, status,
			tenant_id, mfa_enabled, mfa_secret, backup_codes, created_at, updated_at)
		VALUES (:id, :email, :first_name, :last_name, :password_hash, :status,
			:tenant_id, :mfa_enabled, :mfa_secret, :backup_codes, :created_at, :updated_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("user.Create: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, fmt.Errorf("user.GetByID: %w", err)
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error) {
	var user User
	query := `
		SELECT * FROM users
		WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &user, query, email, tenantID)
	if err != nil {
		return nil, fmt.Errorf("user.GetByEmail: %w", err)
	}
	return &user, nil
}

// List retrieves users with pagination
func (r *UserRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]User, error) {
	var users []User
	query := `
		SELECT * FROM users
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &users, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("user.List: %w", err)
	}
	return users, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = :email,
			first_name = :first_name,
			last_name = :last_name,
			password_hash = :password_hash,
			status = :status,
			mfa_enabled = :mfa_enabled,
			mfa_secret = :mfa_secret,
			backup_codes = :backup_codes,
			last_login_at = :last_login_at,
			failed_logins = :failed_logins,
			locked_until = :locked_until,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`

	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("user.Update: %w", err)
	}

	return nil
}

// Delete soft deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user.Delete: %w", err)
	}
	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users SET
			password_hash = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("user.UpdatePassword: %w", err)
	}
	return nil
}

// EnableMFA enables MFA for a user
func (r *UserRepository) EnableMFA(ctx context.Context, id uuid.UUID, secret string, backupCodes string) error {
	query := `
		UPDATE users SET
			mfa_enabled = true,
			mfa_secret = $2,
			backup_codes = $3,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id, secret, backupCodes)
	if err != nil {
		return fmt.Errorf("user.EnableMFA: %w", err)
	}
	return nil
}

// DisableMFA disables MFA for a user
func (r *UserRepository) DisableMFA(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET
			mfa_enabled = false,
			mfa_secret = NULL,
			backup_codes = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user.DisableMFA: %w", err)
	}
	return nil
}

// RecordLogin records a successful login
func (r *UserRepository) RecordLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE users SET
			last_login_at = $2,
			failed_logins = 0,
			locked_until = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("user.RecordLogin: %w", err)
	}
	return nil
}

// RecordFailedLogin records a failed login attempt
func (r *UserRepository) RecordFailedLogin(ctx context.Context, id uuid.UUID, maxAttempts int) error {
	query := `
		UPDATE users SET
			failed_logins = failed_logins + 1,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING failed_logins
	`

	var failedLogins int
	err := r.db.GetContext(ctx, &failedLogins, query, id)
	if err != nil {
		return fmt.Errorf("user.RecordFailedLogin: %w", err)
	}

	// Lock account if too many failed attempts
	if failedLogins >= maxAttempts {
		lockDuration := 30 * time.Minute
		lockUntil := time.Now().Add(lockDuration)
		lockQuery := `
			UPDATE users SET
				status = 'locked',
				locked_until = $2,
				updated_at = NOW()
			WHERE id = $1
		`
		_, err = r.db.ExecContext(ctx, lockQuery, id, lockUntil)
		if err != nil {
			return fmt.Errorf("user.LockAccount: %w", err)
		}
	}

	return nil
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET
			status = 'active',
			failed_logins = 0,
			locked_until = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user.UnlockAccount: %w", err)
	}
	return nil
}

// Suspend suspends a user account
func (r *UserRepository) Suspend(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET
			status = 'suspended',
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user.Suspend: %w", err)
	}
	return nil
}

// Activate activates a user account
func (r *UserRepository) Activate(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET
			status = 'active',
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("user.Activate: %w", err)
	}
	return nil
}

// UserService handles user business logic
type UserService struct {
	repo   *UserRepository
	jwt    *JWTManager
	mfa    *MFAManager
	hasher *PasswordHasher
	logger zerolog.Logger
}

// PasswordHasher handles password hashing
type PasswordHasher struct{}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash hashes a password
func (p *PasswordHasher) Hash(password string) (string, error) {
	return crypto.HashPassword(password)
}

// Verify verifies a password against a hash
func (p *PasswordHasher) Verify(password, hash string) bool {
	return crypto.VerifyPassword(password, hash)
}

// NewUserService creates a new user service
func NewUserService(repo *UserRepository, jwt *JWTManager, mfa *MFAManager, logger zerolog.Logger) *UserService {
	return &UserService{
		repo:   repo,
		jwt:    jwt,
		mfa:    mfa,
		hasher: NewPasswordHasher(),
		logger: logger,
	}
}

// CreateUser creates a new user with password hashing
func (s *UserService) CreateUser(ctx context.Context, user *User, password string) error {
	// Hash password
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("user.HashPassword: %w", err)
	}
	user.PasswordHash = hash

	// Validate user
	if err := s.validateUser(user); err != nil {
		return err
	}

	return s.repo.Create(ctx, user)
}

// Authenticate authenticates a user with email and password
func (s *UserService) Authenticate(ctx context.Context, tenantID uuid.UUID, email, password string) (*User, error) {
	// Get user
	user, err := s.repo.GetByEmail(ctx, tenantID, email)
	if err != nil {
		return nil, fmt.Errorf("user.GetByEmail: %w", err)
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("user: account locked until %s", user.LockedUntil.Format(time.RFC3339))
	}

	// Verify password
	if !s.hasher.Verify(password, user.PasswordHash) {
		_ = s.repo.RecordFailedLogin(ctx, user.ID, 5)
		return nil, fmt.Errorf("user: invalid credentials")
	}

	// Check status
	if user.Status != "active" {
		return nil, fmt.Errorf("user: account is %s", user.Status)
	}

	// Record successful login
	if err := s.repo.RecordLogin(ctx, user.ID); err != nil {
		s.logger.Error().Err(err).Msg("Failed to record login")
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *UserService) Login(ctx context.Context, tenantID uuid.UUID, email, password string) (accessToken, refreshToken string, user *User, err error) {
	user, err = s.Authenticate(ctx, tenantID, email, password)
	if err != nil {
		return "", "", nil, err
	}

	// Get user roles
	roles, err := s.GetUserRoles(ctx, user.ID)
	if err != nil {
		return "", "", nil, fmt.Errorf("user.GetUserRoles: %w", err)
	}

	// Generate tokens
	accessToken, refreshToken, err = s.jwt.GenerateTokenPair(
		ctx,
		user.ID.String(),
		user.Email,
		user.TenantID.String(),
		roles,
		user.MFAEnabled,
	)
	if err != nil {
		return "", "", nil, fmt.Errorf("jwt.GenerateTokenPair: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

// GetUserRoles retrieves a user's roles
func (s *UserService) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
	`

	var roles []string
	err := s.repo.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, fmt.Errorf("user.GetUserRoles: %w", err)
	}

	return roles, nil
}

// validateUser validates user data
func (s *UserService) validateUser(user *User) error {
	if user.Email == "" {
		return fmt.Errorf("user: email is required")
	}
	if user.FirstName == "" {
		return fmt.Errorf("user: first name is required")
	}
	if user.LastName == "" {
		return fmt.Errorf("user: last name is required")
	}
	if user.TenantID == uuid.Nil {
		return fmt.Errorf("user: tenant ID is required")
	}
	return nil
}

// UpdatePassword updates a user's password
func (s *UserService) UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user.GetByID: %w", err)
	}

	// Verify current password
	if !s.hasher.Verify(currentPassword, user.PasswordHash) {
		return fmt.Errorf("user: invalid current password")
	}

	// Hash new password
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("user.HashPassword: %w", err)
	}

	return s.repo.UpdatePassword(ctx, userID, hash)
}
