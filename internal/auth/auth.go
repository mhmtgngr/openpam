package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication operations
type Service struct {
	db         *sqlx.DB
	jwt        *JWTManager
	mfa        *MFAManager
	cache      *cache.Cache
	logger     zerolog.Logger
	maxLoginAttempts int
	lockDuration     time.Duration
}

// NewService creates a new auth service
func NewService(db *sqlx.DB, jwt *JWTManager, mfa *MFAManager, c *cache.Cache, logger zerolog.Logger) *Service {
	return &Service{
		db:               db,
		jwt:              jwt,
		mfa:              mfa,
		cache:            c,
		logger:           logger,
		maxLoginAttempts: 5,
		lockDuration:     30 * time.Minute,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken      string   `json:"access_token"`
	RefreshToken     string   `json:"refresh_token"`
	TokenType        string   `json:"token_type"`
	ExpiresIn        int64    `json:"expires_in"`
	User             *User    `json:"user"`
	MFARequired      bool     `json:"mfa_required"`
	MFASetupRequired bool     `json:"mfa_setup_required"`
}

// Authenticate authenticates a user with email and password
func (s *Service) Authenticate(ctx context.Context, email, password, tenantID string, ip, userAgent string) (*LoginResponse, error) {
	// Get user by email
	user, err := s.getUserByEmail(ctx, email, tenantID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("auth: account is locked until %s", user.LockedUntil.Format(time.RFC3339))
	}

	// Check account status
	if user.Status != "active" {
		return nil, fmt.Errorf("auth: account is %s", user.Status)
	}

	// Verify password
	if err := s.verifyPassword(password, user.PasswordHash); err != nil {
		// Increment failed login attempts
		_ = s.incrementFailedLogins(ctx, user.ID)
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Reset failed logins on successful authentication
	_ = s.resetFailedLogins(ctx, user.ID)

	// Generate tokens
	roles := []string{user.Role}
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID.String(), user.Email, user.TenantID.String(), roles, user.MFAEnabled)
	if err != nil {
		return nil, fmt.Errorf("auth.GenerateTokens: %w", err)
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.updateLastLogin(ctx, user.ID)

	// Check if MFA is required
	mfaRequired := user.MFAEnabled
	mfaSetupRequired := false

	// For privileged roles, require MFA setup if not enabled
	if !user.MFAEnabled && (user.Role == "admin" || user.Role == "super_admin" || user.Role == "operator") {
		mfaRequired = true
		mfaSetupRequired = true
	}

	// Cache session
	sessionKey := fmt.Sprintf("session:%s", accessToken)
	_ = s.cache.Set(ctx, sessionKey, user.ID.String(), 15*time.Minute)

	return &LoginResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        900, // 15 minutes
		User:             user,
		MFARequired:      mfaRequired,
		MFASetupRequired: mfaSetupRequired,
	}, nil
}

// AuthenticateMFA completes MFA authentication
func (s *Service) AuthenticateMFA(ctx context.Context, userID uuid.UUID, code, tenantID string) (*LoginResponse, error) {
	// Get user
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	// Verify MFA code
	if !s.mfa.Verify(ctx, user.ID.String(), "totp", user.MFASecret, code) {
		return nil, fmt.Errorf("auth: invalid MFA code")
	}

	// Generate tokens
	roles := []string{user.Role}
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(ctx, user.ID.String(), user.Email, user.TenantID.String(), roles, true)
	if err != nil {
		return nil, fmt.Errorf("auth.GenerateTokens: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User:         user,
		MFARequired:  false,
	}, nil
}

// RefreshToken refreshes an access token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// Validate refresh token
	claims, err := s.jwt.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("auth.ValidateToken: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("auth: invalid token type")
	}

	// Get user
	user, err := s.getUserByID(ctx, uuid.MustParse(claims.UserID))
	if err != nil {
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	// Generate new access token
	accessToken, err := s.jwt.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("auth.RefreshToken: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		User:         user,
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.jwt.ValidateToken(tokenString)
}

// SetupMFA initializes MFA for a user
func (s *Service) SetupMFA(ctx context.Context, userID uuid.UUID) (string, []byte, error) {
	// Generate TOTP secret
	secret, err := s.mfa.totp.GenerateSecret()
	if err != nil {
		return "", nil, fmt.Errorf("auth.GenerateSecret: %w", err)
	}

	// Get user email
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return "", nil, err
	}

	// Generate QR code
	qrCode, err := s.mfa.totp.GenerateQRCode(user.Email, secret)
	if err != nil {
		return "", nil, fmt.Errorf("auth.GenerateQRCode: %w", err)
	}

	// Store secret temporarily (not active yet)
	cacheKey := fmt.Sprintf("mfa_setup:%s", userID)
	_ = s.cache.Set(ctx, cacheKey, secret, 5*time.Minute)

	return secret, qrCode, nil
}

// VerifyAndEnableMFA verifies MFA setup and enables it
func (s *Service) VerifyAndEnableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	// Get temporary secret
	cacheKey := fmt.Sprintf("mfa_setup:%s", userID)
	var secret string
	if err := s.cache.Get(ctx, cacheKey, &secret); err != nil {
		return fmt.Errorf("auth: MFA setup not initiated or expired")
	}

	// Verify code
	if !s.mfa.totp.VerifyCode(secret, code) {
		return fmt.Errorf("auth: invalid MFA code")
	}

	// Generate backup codes
	backupCodes, err := s.mfa.totp.GenerateBackupCodes(10)
	if err != nil {
		return fmt.Errorf("auth.GenerateBackupCodes: %w", err)
	}

	// Hash backup codes
	hashedCodes, err := s.mfa.totp.HashBackupCodes(backupCodes)
	if err != nil {
		return fmt.Errorf("auth.HashBackupCodes: %w", err)
	}

	// Enable MFA in database
	query := `
		UPDATE users SET
			mfa_enabled = TRUE,
			mfa_secret = $2,
			backup_codes = $3,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, query, userID, secret, hashedCodes)
	if err != nil {
		return fmt.Errorf("auth.EnableMFA: %w", err)
	}

	// Clean up temporary secret
	_ = s.cache.Delete(ctx, cacheKey)

	return nil
}

// Logout logs out a user by revoking their refresh token
func (s *Service) Logout(ctx context.Context, refreshToken, userID string) error {
	// Parse token to get ID
	claims, err := s.jwt.ExtractClaims(refreshToken)
	if err != nil {
		return err
	}

	// Revoke token
	return s.jwt.RevokeToken(ctx, claims.ID)
}

// getUserByEmail retrieves a user by email
func (s *Service) getUserByEmail(ctx context.Context, email, tenantID string) (*User, error) {
	var user User
	query := `
		SELECT * FROM users
		WHERE email = $1 AND tenant_id = (SELECT id FROM tenants WHERE id = $2 OR domain = $2 LIMIT 1)
			AND deleted_at IS NULL
	`
	err := s.db.GetContext(ctx, &user, query, email, tenantID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// getUserByID retrieves a user by ID
func (s *Service) getUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := s.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// verifyPassword verifies a password against a hash
func (s *Service) verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// hashPassword hashes a password using bcrypt
func (s *Service) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// incrementFailedLogins increments failed login counter
func (s *Service) incrementFailedLogins(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			failed_logins = failed_logins + 1,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	// Check if we need to lock the account
	var user User
	query = `SELECT failed_logins FROM users WHERE id = $1`
	err = s.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		return err
	}

	if user.FailedLogins >= s.maxLoginAttempts {
		lockUntil := time.Now().Add(s.lockDuration)
		query = `
			UPDATE users SET
				status = 'locked',
				locked_until = $2,
				updated_at = NOW()
			WHERE id = $1
		`
		_, err = s.db.ExecContext(ctx, query, userID, lockUntil)
	}

	return err
}

// resetFailedLogins resets failed login counter
func (s *Service) resetFailedLogins(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			failed_logins = 0,
			locked_until = NULL,
			status = 'active',
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// updateLastLogin updates the last login timestamp
func (s *Service) updateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			last_login_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.getUserByID(ctx, id)
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, user *User, password string) error {
	// Hash password
	hash, err := s.hashPassword(password)
	if err != nil {
		return fmt.Errorf("auth.HashPassword: %w", err)
	}

	user.PasswordHash = hash
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, email, first_name, last_name, password_hash, role, status,
			mfa_enabled, tenant_id, created_at, updated_at)
		VALUES (:id, :email, :first_name, :last_name, :password_hash, :role, :status,
			:mfa_enabled, :tenant_id, :created_at, :updated_at)
	`

	_, err = s.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("auth.CreateUser: %w", err)
	}

	return nil
}

// ListUsers retrieves users for a tenant
func (s *Service) ListUsers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]User, int, error) {
	var users []User

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND deleted_at IS NULL`
	err := s.db.GetContext(ctx, &total, countQuery, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// Get users
	query := `
		SELECT id, email, first_name, last_name, role, status, mfa_enabled, tenant_id,
			last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err = s.db.SelectContext(ctx, &users, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	query := `
		UPDATE users SET
	`
	args := []interface{}{id}
	argCount := 2

	for key, value := range updates {
		if key != "id" {
			query += fmt.Sprintf("%s = $%d, ", key, argCount)
			args = append(args, value)
			argCount++
		}
	}

	query = query[:len(query)-2] + " WHERE id = $1"

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("auth.UpdateUser: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user
func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users SET
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("auth.DeleteUser: %w", err)
	}
	return nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("auth.GetUser: %w", err)
	}

	// Verify old password
	if err := s.verifyPassword(oldPassword, user.PasswordHash); err != nil {
		return fmt.Errorf("auth: invalid current password")
	}

	// Hash new password
	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth.HashPassword: %w", err)
	}

	// Update password
	query := `
		UPDATE users SET
			password_hash = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, query, userID, hash)
	if err != nil {
		return fmt.Errorf("auth.UpdatePassword: %w", err)
	}

	return nil
}

// GetTenantIDFromDomain retrieves tenant ID from domain
func (s *Service) GetTenantIDFromDomain(ctx context.Context, domain string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	query := `SELECT id FROM tenants WHERE domain = $1 AND deleted_at IS NULL LIMIT 1`
	err := s.db.GetContext(ctx, &tenantID, query, domain)
	if err != nil {
		return uuid.Nil, err
	}
	return tenantID, nil
}

// HasPermission checks if a user has a specific permission
func (s *Service) HasPermission(ctx context.Context, userID uuid.UUID, resource, action string) bool {
	// Check user role
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return false
	}

	// Super admins have all permissions
	if user.Role == "super_admin" {
		return true
	}

	// Check role permissions
	query := `
		SELECT EXISTS(
			SELECT 1 FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			JOIN user_roles ur ON ur.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ur.user_id = $1
				AND p.resource = $2
				AND p.action = $3
		)
	`

	var exists bool
	err = s.db.GetContext(ctx, &exists, query, userID, resource, action)
	return err == nil && exists
}

// GetRoles retrieves all roles for a user
func (s *Service) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1 AND r.deleted_at IS NULL
	`

	var roles []string
	err := s.db.SelectContext(ctx, &roles, query, userID)
	if err != nil {
		return nil, err
	}

	// Always include the base role
	user, err := s.getUserByID(ctx, userID)
	if err == nil {
		roles = append(roles, user.Role)
	}

	return roles, nil
}
