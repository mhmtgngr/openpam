package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLSEnforced(t *testing.T) {
	t.Run("TLS is always enforced", func(t *testing.T) {
		assert.True(t, TLSEnforced, "TLS must be enforced at compile time")
	})

	t.Run("Production build returns true", func(t *testing.T) {
		assert.True(t, IsProductionBuild(), "IsProductionBuild should return true")
	})
}

func TestDefaultTLSConfig(t *testing.T) {
	cfg := DefaultTLSConfig()

	assert.Equal(t, MinTLSVersion, cfg.MinVersion)
	assert.Equal(t, MaxTLSVersion, cfg.MaxVersion)
	assert.NotEmpty(t, cfg.CipherSuites, "Cipher suites must be specified")
	assert.NotEmpty(t, cfg.CurvePreferences, "Curve preferences must be specified")
}

func TestValidateTLSConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultTLSConfig()
		err := ValidateTLSConfig(cfg)
		require.NoError(t, err)
	})

	t.Run("rejects missing min version", func(t *testing.T) {
		cfg := TLSConfig{
			MinVersion: "",
			MaxVersion: "TLS1.3",
			CipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}
		err := ValidateTLSConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "min version must be specified")
	})

	t.Run("rejects missing max version", func(t *testing.T) {
		cfg := TLSConfig{
			MinVersion:  "TLS1.2",
			MaxVersion:  "",
			CipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}
		err := ValidateTLSConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max version must be specified")
	})

	t.Run("rejects weak min version", func(t *testing.T) {
		cfg := TLSConfig{
			MinVersion:  "TLS1.0",
			MaxVersion:  "TLS1.3",
			CipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}
		err := ValidateTLSConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too weak")
	})

	t.Run("rejects TLS1.1 min version", func(t *testing.T) {
		cfg := TLSConfig{
			MinVersion:  "TLS1.1",
			MaxVersion:  "TLS1.3",
			CipherSuites: []string{"TLS_AES_128_GCM_SHA256"},
		}
		err := ValidateTLSConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too weak")
	})

	t.Run("rejects empty cipher suites", func(t *testing.T) {
		cfg := TLSConfig{
			MinVersion:    "TLS1.2",
			MaxVersion:    "TLS1.3",
			CipherSuites:  []string{},
		}
		err := ValidateTLSConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one cipher suite")
	})
}

func TestEnforceDatabaseSSL(t *testing.T) {
	t.Run("allows verify-full", func(t *testing.T) {
		err := EnforceDatabaseSSL("verify-full")
		assert.NoError(t, err)
	})

	t.Run("allows verify-ca", func(t *testing.T) {
		err := EnforceDatabaseSSL("verify-ca")
		assert.NoError(t, err)
	})

	t.Run("allows require", func(t *testing.T) {
		err := EnforceDatabaseSSL("require")
		assert.NoError(t, err)
	})

	t.Run("allows disable", func(t *testing.T) {
		err := EnforceDatabaseSSL("disable")
		assert.NoError(t, err)
	})

	t.Run("rejects allow", func(t *testing.T) {
		err := EnforceDatabaseSSL("allow")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid database SSL mode")
	})

	t.Run("rejects prefer", func(t *testing.T) {
		err := EnforceDatabaseSSL("prefer")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid database SSL mode")
	})

	t.Run("rejects invalid mode", func(t *testing.T) {
		err := EnforceDatabaseSSL("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid database SSL mode")
	})
}
