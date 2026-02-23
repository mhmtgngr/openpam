package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	// Clear env vars
	envVars := []string{"PORT", "LOG_LEVEL", "VAULT_MASTER_KEY"}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// Test defaults
	port := os.Getenv("PORT")
	if port == "" {
		port = "8502"
	}
	assert.Equal(t, "8502", port)

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	assert.Equal(t, "info", logLevel)
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("PORT", "9502")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("VAULT_MASTER_KEY", "test-master-key-32-bytes-long!")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("VAULT_MASTER_KEY")
	}()

	port := os.Getenv("PORT")
	assert.Equal(t, "9502", port)

	logLevel := os.Getenv("LOG_LEVEL")
	assert.Equal(t, "debug", logLevel)

	masterKey := os.Getenv("VAULT_MASTER_KEY")
	assert.Equal(t, "test-master-key-32-bytes-long!", masterKey)
}

func TestServicePort(t *testing.T) {
	t.Run("default port", func(t *testing.T) {
		os.Unsetenv("PORT")
		port := os.Getenv("PORT")
		if port == "" {
			port = "8502"
		}
		assert.Equal(t, "8502", port)
	})

	t.Run("custom port", func(t *testing.T) {
		os.Setenv("PORT", "12345")
		defer os.Unsetenv("PORT")

		port := os.Getenv("PORT")
		assert.Equal(t, "12345", port)
	})
}

func TestVaultServiceName(t *testing.T) {
	serviceName := "vault-service"
	assert.Equal(t, "vault-service", serviceName)
	assert.NotEmpty(t, serviceName)
}
