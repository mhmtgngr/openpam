package strategy

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// RotationResult holds the outcome of a credential rotation
type RotationResult struct {
	NewCredential string
	RotatedAt     time.Time
	NextRotation  time.Time
	Metadata      map[string]string
}

// RotationStrategy defines the interface for credential rotation algorithms
type RotationStrategy interface {
	// Name returns the strategy name
	Name() string
	// Rotate generates a new credential value
	Rotate(ctx context.Context, currentValue string, opts RotationOptions) (*RotationResult, error)
	// Validate checks if a credential meets the strategy's requirements
	Validate(credential string) error
	// NextRotationTime calculates when the next rotation should occur
	NextRotationTime(from time.Time) time.Time
}

// RotationOptions configures rotation behavior
type RotationOptions struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigits  bool
	RequireSpecial bool
	ExcludeChars   string
	CustomPolicy   map[string]string
}

// DefaultRotationOptions returns secure defaults for PAM
func DefaultRotationOptions() RotationOptions {
	return RotationOptions{
		MinLength:      24,
		MaxLength:      64,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigits:  true,
		RequireSpecial: true,
	}
}

// PasswordRotationStrategy generates strong random passwords
type PasswordRotationStrategy struct {
	interval time.Duration
	logger   zerolog.Logger
}

// NewPasswordRotation creates a new password rotation strategy
func NewPasswordRotation(interval time.Duration, logger zerolog.Logger) *PasswordRotationStrategy {
	return &PasswordRotationStrategy{interval: interval, logger: logger}
}

func (s *PasswordRotationStrategy) Name() string { return "password" }

func (s *PasswordRotationStrategy) Rotate(ctx context.Context, currentValue string, opts RotationOptions) (*RotationResult, error) {
	length := opts.MinLength
	if length < 16 {
		length = 24
	}

	password, err := generateSecurePassword(length, opts)
	if err != nil {
		return nil, fmt.Errorf("rotation.password: %w", err)
	}

	now := time.Now()
	return &RotationResult{
		NewCredential: password,
		RotatedAt:     now,
		NextRotation:  s.NextRotationTime(now),
		Metadata:      map[string]string{"strategy": "password", "length": fmt.Sprintf("%d", length)},
	}, nil
}

func (s *PasswordRotationStrategy) Validate(credential string) error {
	if len(credential) < 16 {
		return fmt.Errorf("password must be at least 16 characters")
	}
	return nil
}

func (s *PasswordRotationStrategy) NextRotationTime(from time.Time) time.Time {
	return from.Add(s.interval)
}

// SSHKeyRotationStrategy generates new SSH key pairs
type SSHKeyRotationStrategy struct {
	interval time.Duration
	keySize  int
	logger   zerolog.Logger
}

// NewSSHKeyRotation creates a new SSH key rotation strategy
func NewSSHKeyRotation(interval time.Duration, keySize int, logger zerolog.Logger) *SSHKeyRotationStrategy {
	if keySize < 4096 {
		keySize = 4096
	}
	return &SSHKeyRotationStrategy{interval: interval, keySize: keySize, logger: logger}
}

func (s *SSHKeyRotationStrategy) Name() string { return "ssh_key" }

func (s *SSHKeyRotationStrategy) Rotate(ctx context.Context, currentValue string, opts RotationOptions) (*RotationResult, error) {
	// SSH key generation would be done here using crypto/rsa or ed25519
	// For now, return a placeholder indicating the key type and size
	now := time.Now()
	return &RotationResult{
		RotatedAt:    now,
		NextRotation: s.NextRotationTime(now),
		Metadata: map[string]string{
			"strategy": "ssh_key",
			"key_size": fmt.Sprintf("%d", s.keySize),
			"key_type": "rsa",
		},
	}, nil
}

func (s *SSHKeyRotationStrategy) Validate(credential string) error {
	if !strings.Contains(credential, "BEGIN") {
		return fmt.Errorf("invalid SSH key format")
	}
	return nil
}

func (s *SSHKeyRotationStrategy) NextRotationTime(from time.Time) time.Time {
	return from.Add(s.interval)
}

// APITokenRotationStrategy handles API token rotation
type APITokenRotationStrategy struct {
	interval time.Duration
	logger   zerolog.Logger
}

// NewAPITokenRotation creates a new API token rotation strategy
func NewAPITokenRotation(interval time.Duration, logger zerolog.Logger) *APITokenRotationStrategy {
	return &APITokenRotationStrategy{interval: interval, logger: logger}
}

func (s *APITokenRotationStrategy) Name() string { return "api_token" }

func (s *APITokenRotationStrategy) Rotate(ctx context.Context, currentValue string, opts RotationOptions) (*RotationResult, error) {
	token, err := generateSecureToken(48)
	if err != nil {
		return nil, fmt.Errorf("rotation.api_token: %w", err)
	}

	now := time.Now()
	return &RotationResult{
		NewCredential: token,
		RotatedAt:     now,
		NextRotation:  s.NextRotationTime(now),
		Metadata:      map[string]string{"strategy": "api_token"},
	}, nil
}

func (s *APITokenRotationStrategy) Validate(credential string) error {
	if len(credential) < 32 {
		return fmt.Errorf("API token must be at least 32 characters")
	}
	return nil
}

func (s *APITokenRotationStrategy) NextRotationTime(from time.Time) time.Time {
	return from.Add(s.interval)
}

// RotationStrategyRegistry holds available rotation strategies
type RotationStrategyRegistry struct {
	strategies map[string]RotationStrategy
}

// NewRotationRegistry creates a new rotation strategy registry
func NewRotationRegistry() *RotationStrategyRegistry {
	return &RotationStrategyRegistry{
		strategies: make(map[string]RotationStrategy),
	}
}

// Register adds a strategy to the registry
func (r *RotationStrategyRegistry) Register(strategy RotationStrategy) {
	r.strategies[strategy.Name()] = strategy
}

// Get returns a strategy by name
func (r *RotationStrategyRegistry) Get(name string) (RotationStrategy, error) {
	s, ok := r.strategies[name]
	if !ok {
		return nil, fmt.Errorf("rotation strategy %q not registered", name)
	}
	return s, nil
}

// List returns all registered strategy names
func (r *RotationStrategyRegistry) List() []string {
	names := make([]string, 0, len(r.strategies))
	for name := range r.strategies {
		names = append(names, name)
	}
	return names
}

func generateSecurePassword(length int, opts RotationOptions) (string, error) {
	const (
		upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower   = "abcdefghijklmnopqrstuvwxyz"
		digits  = "0123456789"
		special = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	)

	charset := ""
	if opts.RequireUpper {
		charset += upper
	}
	if opts.RequireLower {
		charset += lower
	}
	if opts.RequireDigits {
		charset += digits
	}
	if opts.RequireSpecial {
		charset += special
	}
	if charset == "" {
		charset = upper + lower + digits + special
	}

	// Remove excluded characters
	for _, c := range opts.ExcludeChars {
		charset = strings.ReplaceAll(charset, string(c), "")
	}

	password := make([]byte, length)
	for i := range password {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[idx.Int64()]
	}

	return string(password), nil
}

func generateSecureToken(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	token := make([]byte, length)
	for i := range token {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		token[i] = charset[idx.Int64()]
	}
	return string(token), nil
}
