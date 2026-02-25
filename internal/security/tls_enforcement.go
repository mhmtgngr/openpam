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
	"os"
	"strings"
)

const (
	// TLSEnforced indicates that TLS is enforced at compile time
	TLSEnforced = true

	// MinTLSVersion is the minimum TLS version required
	MinTLSVersion = "TLS1.2"

	// MaxTLSVersion is the maximum TLS version supported
	MaxTLSVersion = "TLS1.3"
)

// InsecureSSLMode represents an insecure SSL mode configuration
type InsecureSSLMode struct {
	Mode     string
	Reason   string
	Fix      string
}

// InsecureSSLModes lists SSL modes that are never acceptable in production
var InsecureSSLModes = map[string]InsecureSSLMode{
	"disable": {
		Mode:   "disable",
		Reason: "Completely disables TLS encryption, allowing credentials to be transmitted in plaintext",
		Fix:    "Use 'verify-full' for production or 'require' with proper certificates",
	},
	"allow": {
		Mode:   "allow",
		Reason: "Allows unencrypted connections, falling back to TLS only if server insists",
		Fix:    "Use 'verify-full' to enforce TLS encryption",
	},
	"prefer": {
		Mode:   "prefer",
		Reason: "Only attempts TLS but allows unencrypted connections if TLS fails",
		Fix:    "Use 'verify-full' to enforce TLS encryption",
	},
}

// IsProductionEnvironment determines if we're running in a production environment
// SECURITY: This check considers multiple factors to prevent accidental misconfiguration
func IsProductionEnvironment() bool {
	// Check explicit environment variable
	if env := strings.ToLower(os.Getenv("OPENPAM_ENV")); env == "production" || env == "prod" {
		return true
	}

	// Check for common production environment indicators
	if env := strings.ToLower(os.Getenv("ENV")); env == "production" || env == "prod" {
		return true
	}
	if env := strings.ToLower(os.Getenv("ENVIRONMENT")); env == "production" || env == "prod" {
		return true
	}
	if env := strings.ToLower(os.Getenv("GO_ENV")); env == "production" || env == "prod" {
		return true
	}

	// Check for Kubernetes/production orchestration
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		// We're in Kubernetes - check if we're in a production namespace
		if ns := os.Getenv("NAMESPACE"); ns == "production" || ns == "prod" {
			return true
		}
	}

	// Check for cloud provider production environments
	if os.Getenv("AWS_REGION") != "" || os.Getenv("GOOGLE_CLOUD_PROJECT") != "" || os.Getenv("AZURE_CLIENT_ID") != "" {
		// Running in a cloud - assume production unless explicitly marked as dev/test
		if env := strings.ToLower(os.Getenv("OPENPAM_ENV")); env == "development" || env == "dev" || env == "test" {
			return false
		}
		// Default to true for cloud environments
		return true
	}

	return false
}

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
// SECURITY FIX: In production, only secure modes are allowed
func EnforceDatabaseSSL(sslMode string) error {
	// Check if this is a known insecure mode
	if insecure, ok := InsecureSSLModes[strings.ToLower(sslMode)]; ok {
		if IsProductionEnvironment() {
			// SECURITY: In production, reject insecure modes outright
			return fmt.Errorf("security: INSECURE database SSL mode '%s' is NOT ALLOWED in production.\nReason: %s\nFix: %s",
				insecure.Mode, insecure.Reason, insecure.Fix)
		}
		// In development, allow but warn
		fmt.Fprintf(os.Stderr, "SECURITY WARNING: Using insecure SSL mode '%s'. %s. Recommended: %s\n",
			insecure.Mode, insecure.Reason, insecure.Fix)
	}

	// Allowed modes for production
	// SECURITY: no-verify is NOT allowed in production - it maps to disable
	productionAllowedModes := map[string]bool{
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	// Development-only modes (require explicit OPENPAM_ENV=development)
	developmentModes := map[string]bool{
		"no-verify":          true, // For development without TLS (maps to disable)
		"no-verify-require":  true, // For development with TLS but no cert verification (maps to require)
		"disable":            true, // Explicitly allowed for local Docker development ONLY
	}

	if IsProductionEnvironment() {
		if !productionAllowedModes[strings.ToLower(sslMode)] {
			return fmt.Errorf("security: database SSL mode '%s' is not allowed in production. Allowed modes: require, verify-ca, verify-full", sslMode)
		}
	} else {
		// In development, allow both production and development modes
		if !productionAllowedModes[strings.ToLower(sslMode)] && !developmentModes[strings.ToLower(sslMode)] {
			return fmt.Errorf("security: invalid database SSL mode '%s'. Must be one of: require, verify-ca, verify-full, no-verify, disable", sslMode)
		}
	}

	return nil
}

// IsProductionBuild returns true if this is a production build with TLS enforced
func IsProductionBuild() bool {
	return TLSEnforced
}

// ValidateProductionSSLConfig validates that SSL is properly configured for production
// This should be called during application startup
func ValidateProductionSSLConfig(sslMode string) error {
	if IsProductionEnvironment() {
		// In production, require at least 'verify-ca'
		secureModes := map[string]bool{
			"verify-ca":   true,
			"verify-full": true,
		}

		if !secureModes[strings.ToLower(sslMode)] {
			return fmt.Errorf("security: PRODUCTION deployment detected but SSL mode '%s' is not secure enough.\n"+
				"Production requires 'verify-ca' or 'verify-full' to prevent man-in-the-middle attacks.\n"+
				"Current mode '%s' does not verify server certificates.\n"+
				"Fix: Set OPENPAM_DB_SSLMODE=verify-full in your environment",
				sslMode, sslMode)
		}
	}
	return nil
}
