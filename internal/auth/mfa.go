package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/openpam/openpam/internal/cache"
	"github.com/rs/zerolog"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// TOTPConfig holds TOTP configuration
type TOTPConfig struct {
	Issuer      string
	Algorithm   string // SHA256
	Digits      int    // 6
	Period      int    // 30
	SecretSize  int    // 32
	Leeway      int    // 1 (window in periods)
}

// TOTPManager handles TOTP-based MFA
type TOTPManager struct {
	config TOTPConfig
	cache  *cache.Cache
	logger zerolog.Logger
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager(cfg TOTPConfig, c *cache.Cache, logger zerolog.Logger) *TOTPManager {
	return &TOTPManager{
		config: cfg,
		cache:  c,
		logger: logger,
	}
}

// GenerateSecret generates a new TOTP secret
func (t *TOTPManager) GenerateSecret() (string, error) {
	secret := make([]byte, t.config.SecretSize)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("totp.GenerateSecret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateQRCode generates a QR code for TOTP enrollment
func (t *TOTPManager) GenerateQRCode(email, secret string) ([]byte, error) {
	// otpauth://totp/ISSUER:EMAIL?secret=SECRET&issuer=ISSUER&algorithm=SHA256&digits=6&period=30
	u := url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   fmt.Sprintf("%s:%s", url.QueryEscape(t.config.Issuer), url.QueryEscape(email)),
	}

	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", t.config.Issuer)
	params.Set("algorithm", t.config.Algorithm)
	params.Set("digits", fmt.Sprintf("%d", t.config.Digits))
	params.Set("period", fmt.Sprintf("%d", t.config.Period))
	u.RawQuery = params.Encode()

	return qrcode.Encode(u.String(), qrcode.Medium, 256)
}

// VerifyCode verifies a TOTP code
func (t *TOTPManager) VerifyCode(secret, code string) bool {
	// Generate valid codes for current time and leeway windows
	now := time.Now().Unix()
	period := int64(t.config.Period)
	leeway := t.config.Leeway

	// Check current and adjacent time windows
	for i := -leeway; i <= leeway; i++ {
		counter := (now / period) + int64(i)
		expectedCode := t.generateCode(secret, counter)
		if expectedCode == code {
			return true
		}
	}

	return false
}

// generateCode generates a TOTP code for a given counter
func (t *TOTPManager) generateCode(secret string, counter int64) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}

	// HMAC-SHA256
	buf := make([]byte, 8)
	buf[0] = byte(counter >> 56)
	buf[1] = byte(counter >> 48)
	buf[2] = byte(counter >> 40)
	buf[3] = byte(counter >> 32)
	buf[4] = byte(counter >> 24)
	buf[5] = byte(counter >> 16)
	buf[6] = byte(counter >> 8)
	buf[7] = byte(counter)

	h := hmac.New(sha256.New, key)
	h.Write(buf)
	hash := h.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	truncated := hash[offset : offset+4]
	truncated[0] &= 0x7f

	code := int(truncated[0])<<24 | int(truncated[1])<<16 | int(truncated[2])<<8 | int(truncated[3])
	code = code % 1000000

	return fmt.Sprintf("%0*d", t.config.Digits, code)
}

// VerifyBackupCode verifies a backup code
func (t *TOTPManager) VerifyBackupCode(hashedCodes, code string) bool {
	codes := strings.Split(hashedCodes, ",")
	for _, hashedCode := range codes {
		if bcrypt.CompareHashAndPassword([]byte(hashedCode), []byte(code)) == nil {
			return true
		}
	}
	return false
}

// GenerateBackupCodes generates backup recovery codes
func (t *TOTPManager) GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate 16 character alphanumeric code
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return nil, fmt.Errorf("totp.GenerateBackupCodes: %w", err)
		}
		codes[i] = fmt.Sprintf("%016x", b)
	}
	return codes, nil
}

// HashBackupCodes hashes backup codes for storage
func (t *TOTPManager) HashBackupCodes(codes []string) (string, error) {
	hashed := make([]string, len(codes))
	for i, code := range codes {
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return "", fmt.Errorf("totp.HashBackupCode: %w", err)
		}
		hashed[i] = string(hash)
	}
	return strings.Join(hashed, ","), nil
}

// WebAuthnManager handles WebAuthn/FIDO2 MFA
type WebAuthnManager struct {
	webauthn *webauthn.WebAuthn
	cache    *cache.Cache
	logger   zerolog.Logger
}

// WebAuthnConfig holds WebAuthn configuration
type WebAuthnConfig struct {
	RPDisplayName string
	RPID          string
	RPOrigins     []string
}

// NewWebAuthnManager creates a new WebAuthn manager
func NewWebAuthnManager(cfg WebAuthnConfig, c *cache.Cache, logger zerolog.Logger) (*WebAuthnManager, error) {
	wconfig := webauthn.Config{
		RPDisplayName: cfg.RPDisplayName,
		RPID:          cfg.RPID,
		RPOrigins:     cfg.RPOrigins,
	}

	webauthn, err := webauthn.New(&wconfig)
	if err != nil {
		return nil, fmt.Errorf("webauthn.New: %w", err)
	}

	return &WebAuthnManager{
		webauthn: webauthn,
		cache:    c,
		logger:   logger,
	}, nil
}

// User represents a WebAuthn user
type WebAuthnUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

// WebAuthnID returns the user's WebAuthn ID
func (u *WebAuthnUser) WebAuthnID() []byte {
	return u.ID
}

// WebAuthnName returns the user's username
func (u *WebAuthnUser) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the user's display name
func (u *WebAuthnUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnIcon returns the user's icon URL
func (u *WebAuthnUser) WebAuthnIcon() string {
	return ""
}

// WebAuthnCredentials returns the user's credentials
func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// BeginRegistration starts WebAuthn registration
func (w *WebAuthnManager) BeginRegistration(user *WebAuthnUser) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	options, sessionData, err := w.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, nil, fmt.Errorf("webauthn.BeginRegistration: %w", err)
	}

	// Store session data
	key := fmt.Sprintf("webauthn:registration:%s", string(user.WebAuthnID()))
	if err := w.cache.Set(context.Background(), key, sessionData, 5*time.Minute); err != nil {
		w.logger.Error().Err(err).Msg("Failed to cache WebAuthn session")
	}

	return options, sessionData, nil
}

// FinishRegistration completes WebAuthn registration
func (w *WebAuthnManager) FinishRegistration(user *WebAuthnUser, req *http.Request) (*webauthn.Credential, error) {
	// Get session data
	key := fmt.Sprintf("webauthn:registration:%s", string(user.WebAuthnID()))
	var sessionData webauthn.SessionData
	if err := w.cache.Get(context.Background(), key, &sessionData); err != nil {
		return nil, fmt.Errorf("webauthn: session not found")
	}

	credential, err := w.webauthn.FinishRegistration(user, sessionData, req)
	if err != nil {
		return nil, fmt.Errorf("webauthn.FinishRegistration: %w", err)
	}

	// Clean up session
	_ = w.cache.Delete(context.Background(), key)

	return credential, nil
}

// BeginLogin starts WebAuthn login
func (w *WebAuthnManager) BeginLogin(user *WebAuthnUser) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	options, sessionData, err := w.webauthn.BeginLogin(user)
	if err != nil {
		return nil, nil, fmt.Errorf("webauthn.BeginLogin: %w", err)
	}

	// Store session data
	key := fmt.Sprintf("webauthn:login:%s", string(user.WebAuthnID()))
	if err := w.cache.Set(context.Background(), key, sessionData, 5*time.Minute); err != nil {
		w.logger.Error().Err(err).Msg("Failed to cache WebAuthn session")
	}

	return options, sessionData, nil
}

// FinishLogin completes WebAuthn login
func (w *WebAuthnManager) FinishLogin(user *WebAuthnUser, req *http.Request) error {
	// Get session data
	key := fmt.Sprintf("webauthn:login:%s", string(user.WebAuthnID()))
	var sessionData webauthn.SessionData
	if err := w.cache.Get(context.Background(), key, &sessionData); err != nil {
		return fmt.Errorf("webauthn: session not found")
	}

	_, err := w.webauthn.FinishLogin(user, sessionData, req)
	if err != nil {
		return fmt.Errorf("webauthn.FinishLogin: %w", err)
	}

	// Clean up session
	_ = w.cache.Delete(context.Background(), key)

	return nil
}

// MFAManager manages multiple MFA methods
type MFAManager struct {
	totp     *TOTPManager
	webauthn *WebAuthnManager
	cache    *cache.Cache
	logger   zerolog.Logger
}

// NewMFAManager creates a new MFA manager
func NewMFAManager(totp *TOTPManager, webauthn *WebAuthnManager, c *cache.Cache, logger zerolog.Logger) *MFAManager {
	return &MFAManager{
		totp:     totp,
		webauthn: webauthn,
		cache:    c,
		logger:   logger,
	}
}

// MFAMethod represents an MFA method
type MFAMethod string

const (
	MFAMethodTOTP    MFAMethod = "totp"
	MFAMethodWebAuthn MFAMethod = "webauthn"
	MFAMethodBackup  MFAMethod = "backup"
)

// Verify verifies an MFA challenge
func (m *MFAManager) Verify(ctx context.Context, userID string, method MFAMethod, secret, response string) bool {
	switch method {
	case MFAMethodTOTP:
		return m.totp.VerifyCode(secret, response)
	case MFAMethodBackup:
		return m.totp.VerifyBackupCode(secret, response)
	default:
		return false
	}
}

// CheckRequired checks if MFA is required for a user
func (m *MFAManager) CheckRequired(ctx context.Context, userID string, roles []string) bool {
	// MFA is required for privileged roles
	for _, role := range roles {
		if role == "admin" || role == "super_admin" || role == "operator" {
			return true
		}
	}
	return false
}

// VerifySession checks if MFA was verified for a session
func (m *MFAManager) VerifySession(ctx context.Context, sessionID string) bool {
	key := fmt.Sprintf("mfa:verified:%s", sessionID)
	return m.cache.Exists(ctx, key)
}

// MarkSessionVerified marks a session as MFA verified
func (m *MFAManager) MarkSessionVerified(ctx context.Context, sessionID string, expiration time.Duration) error {
	key := fmt.Sprintf("mfa:verified:%s", sessionID)
	return m.cache.Set(ctx, key, true, expiration)
}
