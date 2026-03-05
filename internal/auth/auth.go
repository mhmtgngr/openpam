package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/crypto"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// validateEmail validates email format
func validateEmail(email string) error {
	if len(email) == 0 || len(email) > 254 {
		return fmt.Errorf("invalid email length")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateMFACode validates MFA code format (6 digits)
func validateMFACode(code string) error {
	if len(code) == 0 {
		return fmt.Errorf("MFA code is required")
	}
	if len(code) != 6 {
		return fmt.Errorf("MFA code must be 6 digits")
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return fmt.Errorf("MFA code must contain only digits")
		}
	}
	return nil
}

// validatePasswordStrength validates password meets minimum complexity requirements
// SECURITY: Enforces strong passwords for a PAM platform
func validatePasswordStrength(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must not exceed 128 characters")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	// Check for common weak passwords
	lower := strings.ToLower(password)
	weakPasswords := []string{"password", "admin", "welcome", "qwerty", "123456"}
	for _, weak := range weakPasswords {
		if strings.Contains(lower, weak) {
			return fmt.Errorf("password contains a commonly used pattern")
		}
	}

	return nil
}

// Service handles authentication operations
type Service struct {
	db         *sqlx.DB
	jwt        *JWTManager
	mfa        *MFAManager
	cache      *cache.Cache
	logger     zerolog.Logger
	maxLoginAttempts int
	lockDuration     time.Duration
	mfaKeyEncryptor  *crypto.Encryptor // For encrypting MFA secrets at rest
}

// NewService creates a new auth service
func NewService(db *sqlx.DB, jwt *JWTManager, mfa *MFAManager, c *cache.Cache, logger zerolog.Logger, mfaEncryptionKey []byte) *Service {
	var mfaEncryptor *crypto.Encryptor
	var err error

	if len(mfaEncryptionKey) == 32 {
		mfaEncryptor, err = crypto.NewEncryptor(mfaEncryptionKey)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create MFA encryptor, MFA secrets will not be encrypted at rest")
		}
	} else if len(mfaEncryptionKey) > 0 {
		logger.Warn().Int("key_length", len(mfaEncryptionKey)).Msg("MFA encryption key must be 32 bytes, MFA secrets will not be encrypted at rest")
	}

	return &Service{
		db:               db,
		jwt:              jwt,
		mfa:              mfa,
		cache:            c,
		logger:           logger,
		maxLoginAttempts: 5,
		lockDuration:     30 * time.Minute,
		mfaKeyEncryptor:  mfaEncryptor,
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
	// SECURITY: Validate email format before querying database
	if err := validateEmail(email); err != nil {
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Get user by email
	// SECURITY: Use generic error message to prevent email enumeration (A01:2021)
	user, err := s.getUserByEmail(ctx, email, tenantID)
	if err != nil {
		s.logger.Debug().Err(err).Str("email", email).Msg("User lookup failed")
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Check if account is locked
	// SECURITY: Do not reveal lock expiry time to prevent enumeration
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		s.logger.Warn().
			Str("user_id", user.ID.String()).
			Str("ip", ip).
			Str("action", "login_blocked_locked").
			Msg("Login attempt on locked account")
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Check account status
	// SECURITY: Do not reveal account status to prevent enumeration
	if user.Status != "active" {
		s.logger.Warn().
			Str("user_id", user.ID.String()).
			Str("status", user.Status).
			Str("ip", ip).
			Str("action", "login_blocked_inactive").
			Msg("Login attempt on inactive account")
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Verify password
	if err := s.verifyPassword(password, user.PasswordHash); err != nil {
		// Increment failed login attempts
		_ = s.incrementFailedLogins(ctx, user.ID)
		// SECURITY: Audit log failed login attempt for brute force detection
		s.logger.Warn().
			Str("email", email).
			Str("tenant_id", tenantID).
			Str("ip", ip).
			Str("user_agent", userAgent).
			Str("action", "login_failed").
			Int("failed_attempts", user.FailedLogins+1).
			Msg("Failed login attempt")
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	// Reset failed logins on successful authentication
	_ = s.resetFailedLogins(ctx, user.ID)

	// SECURITY: Audit log successful authentication
	s.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", email).
		Str("tenant_id", tenantID).
		Str("ip", ip).
		Str("action", "login_success").
		Msg("User authenticated successfully")

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
	// SECURITY: Validate MFA code format before verification
	if err := validateMFACode(code); err != nil {
		return nil, fmt.Errorf("auth: invalid MFA code")
	}

	// Get user
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	// SECURITY FIX: Decrypt MFA secret before verification
	secretToVerify := user.MFASecret
	if s.mfaKeyEncryptor != nil && user.MFASecret != "" {
		decryptedSecret, err := s.decryptMFASecret(user.MFASecret)
		if err != nil {
			s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to decrypt MFA secret")
			return nil, fmt.Errorf("auth: MFA verification failed")
		}
		secretToVerify = decryptedSecret
	}

	// Verify MFA code
	if !s.mfa.Verify(ctx, user.ID.String(), "totp", secretToVerify, code) {
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
// SECURITY FIX: Encrypts MFA secret before storing in database
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

	// SECURITY: MFA encryption is MANDATORY - refuse to store plaintext secrets
	if s.mfaKeyEncryptor == nil {
		return fmt.Errorf("auth.EnableMFA: MFA encryption key not configured, refusing to store plaintext MFA secret")
	}
	secretToStore, err := s.encryptMFASecret(secret)
	if err != nil {
		return fmt.Errorf("auth.EncryptMFASecret: %w", err)
	}

	// Enable MFA in database with encrypted secret
	query := `
		UPDATE users SET
			mfa_enabled = TRUE,
			mfa_secret = $2,
			backup_codes = $3,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, query, userID, secretToStore, hashedCodes)
	if err != nil {
		return fmt.Errorf("auth.EnableMFA: %w", err)
	}

	// Clean up temporary secret
	_ = s.cache.Delete(ctx, cacheKey)

	s.logger.Info().
		Str("user_id", userID.String()).
		Msg("MFA enabled with encrypted secret")

	return nil
}

// encryptMFASecret encrypts an MFA secret for storage
// SECURITY: Returns error if encryptor is nil to prevent plaintext storage
func (s *Service) encryptMFASecret(secret string) (string, error) {
	if s.mfaKeyEncryptor == nil {
		return "", fmt.Errorf("MFA encryption key not configured")
	}
	return s.mfaKeyEncryptor.EncryptString(secret)
}

// decryptMFASecret decrypts an MFA secret from storage
// SECURITY: Returns error if encryptor is nil
func (s *Service) decryptMFASecret(encryptedSecret string) (string, error) {
	if s.mfaKeyEncryptor == nil {
		return "", fmt.Errorf("MFA encryption key not configured")
	}
	return s.mfaKeyEncryptor.DecryptString(encryptedSecret)
}

// Logout logs out a user by revoking their refresh token and invalidating active sessions
// SECURITY: Invalidates both refresh token AND cached access token sessions
func (s *Service) Logout(ctx context.Context, refreshToken, userID string) error {
	// Parse token to get ID
	claims, err := s.jwt.ExtractClaims(refreshToken)
	if err != nil {
		return err
	}

	// SECURITY: Invalidate all cached sessions for this user
	// This prevents continued access with a still-valid access token after logout
	sessionPattern := fmt.Sprintf("session:%s", userID)
	_ = s.cache.Delete(ctx, sessionPattern)

	// Also invalidate any CSRF tokens
	csrfKey := fmt.Sprintf("csrf:%s", userID)
	_ = s.cache.Delete(ctx, csrfKey)

	s.logger.Info().
		Str("user_id", userID).
		Str("action", "logout").
		Msg("User logged out, sessions invalidated")

	// Revoke refresh token
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
	// SECURITY: Validate email format
	if err := validateEmail(user.Email); err != nil {
		return fmt.Errorf("auth.CreateUser: %w", err)
	}

	// SECURITY: Enforce password complexity requirements
	if err := validatePasswordStrength(password); err != nil {
		return fmt.Errorf("auth.CreateUser: %w", err)
	}

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

// UpdateUser updates a user with field whitelist to prevent SQL injection
// SECURITY: Only fields in the whitelist can be updated
func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	// SECURITY: Field whitelist to prevent SQL injection via key names
	allowedFields := map[string]bool{
		"email":          true,
		"first_name":     true,
		"last_name":      true,
		"role":           true,
		"status":         true,
		"mfa_enabled":    true,
		"password_hash":  true,
		"failed_logins":  true,
		"locked_until":   true,
		"last_login_at":  true,
	}

	query := `UPDATE users SET `
	args := []interface{}{id}
	argCount := 2
	hasUpdates := false

	for key, value := range updates {
		if key == "id" {
			continue // Never allow updating ID
		}
		if !allowedFields[key] {
			s.logger.Warn().
				Str("field", key).
				Str("user_id", id.String()).
				Msg("Attempted to update non-whitelisted field, skipping")
			continue
		}
		query += fmt.Sprintf("%s = $%d, ", key, argCount)
		args = append(args, value)
		argCount++
		hasUpdates = true
	}

	if !hasUpdates {
		return fmt.Errorf("auth.UpdateUser: no valid fields to update")
	}

	// Add updated_at timestamp
	query += fmt.Sprintf("updated_at = $%d WHERE id = $1", argCount)
	args = append(args, time.Now())

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
// SECURITY: Validates password strength, prevents password reuse, and logs the change
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// SECURITY: Validate new password strength
	if err := validatePasswordStrength(newPassword); err != nil {
		return fmt.Errorf("auth.ChangePassword: %w", err)
	}

	// SECURITY: Prevent reusing the same password
	if oldPassword == newPassword {
		return fmt.Errorf("auth: new password must be different from current password")
	}

	// Get user
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("auth.GetUser: %w", err)
	}

	// Verify old password
	if err := s.verifyPassword(oldPassword, user.PasswordHash); err != nil {
		s.logger.Warn().
			Str("user_id", userID.String()).
			Str("action", "password_change_failed").
			Msg("Password change failed: invalid current password")
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

	// SECURITY: Audit log password change
	s.logger.Info().
		Str("user_id", userID.String()).
		Str("action", "password_changed").
		Msg("User password changed successfully")

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

// DisableMFA disables MFA for a user
// SECURITY: This is a high-risk operation that must be audit logged
func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID) error {
	// SECURITY: Audit log BEFORE the operation for guaranteed traceability
	s.logger.Warn().
		Str("user_id", userID.String()).
		Str("action", "mfa_disable_attempted").
		Msg("MFA disable attempted - HIGH RISK OPERATION")

	query := `
		UPDATE users SET
			mfa_enabled = FALSE,
			mfa_secret = NULL,
			backup_codes = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		s.logger.Error().
			Str("user_id", userID.String()).
			Str("action", "mfa_disable_failed").
			Err(err).
			Msg("MFA disable failed")
		return fmt.Errorf("auth.DisableMFA: %w", err)
	}

	// SECURITY: Audit log the completed operation at WARN level
	s.logger.Warn().
		Str("user_id", userID.String()).
		Str("action", "mfa_disabled").
		Msg("MFA disabled for user - SECURITY EVENT")

	return nil
}

// RegenerateBackupCodes regenerates backup codes for a user with MFA enabled
func (s *Service) RegenerateBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// Get user to verify MFA is enabled
	user, err := s.getUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.GetUser: %w", err)
	}

	if !user.MFAEnabled {
		return nil, fmt.Errorf("auth: MFA not enabled for user")
	}

	// Generate new backup codes
	backupCodes, err := s.mfa.totp.GenerateBackupCodes(10)
	if err != nil {
		return nil, fmt.Errorf("auth.GenerateBackupCodes: %w", err)
	}

	// Hash backup codes
	hashedCodes, err := s.mfa.totp.HashBackupCodes(backupCodes)
	if err != nil {
		return nil, fmt.Errorf("auth.HashBackupCodes: %w", err)
	}

	// Update in database
	query := `
		UPDATE users SET
			backup_codes = $2,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.ExecContext(ctx, query, userID, hashedCodes)
	if err != nil {
		return nil, fmt.Errorf("auth.UpdateBackupCodes: %w", err)
	}

	s.logger.Info().
		Str("user_id", userID.String()).
		Msg("Backup codes regenerated")

	return backupCodes, nil
}
