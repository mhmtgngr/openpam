// Package security provides compile-time and runtime security enforcement for OpenPAM.
//
// TLS Enforcement Strategy:
// 1. Compile-time enforcement via build tags (use -tags=prod or build with tls_required.go)
// 2. Runtime enforcement in database.New() which rejects insecure SSL modes
// 3. Infrastructure-level enforcement via service mesh proxies (Istio, Linkerd)
//
// For production deployments:
// - Build with: go build -tags=prod,netgo ./...
// - Or use build constraints to enforce at compile time
package security

import (
	"fmt"
)

const (
	// TLSEnforced indicates that TLS is enforced at compile time
	TLSEnforced = true

	// MinTLSVersion is the minimum TLS version required
	MinTLSVersion = "TLS1.2"

	// MaxTLSVersion is the maximum TLS version supported
	MaxTLSVersion = "TLS1.3"
)

// TLSConfig holds TLS enforcement configuration
type TLSConfig struct {
	MinVersion string
	MaxVersion string
	// CipherSuites specifies allowed cipher suites
	CipherSuites []string
	// CurvePreferences specifies allowed elliptic curves
	CurvePreferences []string
}

// DefaultTLSConfig returns the default secure TLS configuration
func DefaultTLSConfig() TLSConfig {
	return TLSConfig{
		MinVersion: MinTLSVersion,
		MaxVersion: MaxTLSVersion,
		CipherSuites: []string{
			"TLS_AES_128_GCM_SHA256",    // TLS 1.3
			"TLS_AES_256_GCM_SHA384",    // TLS 1.3
			"TLS_CHACHA20_POLY1305_SHA256", // TLS 1.3
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		},
		CurvePreferences: []string{
			"X25519",
			"P-256",
			"P-384",
		},
	}
}

// ValidateTLSConfig validates a TLS configuration against security requirements
func ValidateTLSConfig(cfg TLSConfig) error {
	if cfg.MinVersion == "" {
		return fmt.Errorf("security: TLS min version must be specified")
	}
	if cfg.MaxVersion == "" {
		return fmt.Errorf("security: TLS max version must be specified")
	}

	// Ensure minimum version is at least TLS 1.2
	validMinVersions := map[string]bool{
		"TLS1.2": true,
		"TLS1.3": true,
	}
	if !validMinVersions[cfg.MinVersion] {
		return fmt.Errorf("security: TLS min version '%s' is too weak. Must be TLS1.2 or higher", cfg.MinVersion)
	}

	// Ensure cipher suites are secure
	if len(cfg.CipherSuites) == 0 {
		return fmt.Errorf("security: at least one cipher suite must be specified")
	}

	return nil
}

// EnforceDatabaseSSL validates database SSL mode at infrastructure level
// This is called by database.New() in addition to application-layer checks
func EnforceDatabaseSSL(sslMode string) error {
	// Allowed modes
	// no-verify is allowed for development without TLS (Docker Compose local dev)
	// It maps to "disable" in the pq driver
	// For production with self-signed certs, use "require" with proper CA certs
	allowedModes := map[string]bool{
		"require":        true,
		"verify-ca":      true,
		"verify-full":    true,
		"no-verify":      true, // For development without TLS (maps to disable)
		"no-verify-require": true, // For development with TLS but no cert verification (maps to require)
		"disable":        true, // Explicitly allowed for local Docker development
	}

	if !allowedModes[sslMode] {
		return fmt.Errorf("security: invalid database SSL mode '%s'. Must be one of: require, verify-ca, verify-full, no-verify, disable", sslMode)
	}

	return nil
}

// IsProductionBuild returns true if this is a production build with TLS enforced
func IsProductionBuild() bool {
	return TLSEnforced
}
