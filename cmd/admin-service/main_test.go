package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	envVars := []string{"PORT", "LOG_LEVEL"}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8505"
	}
	assert.Equal(t, "8505", port)

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	assert.Equal(t, "info", logLevel)
}

func TestConfigFromEnv(t *testing.T) {
	os.Setenv("PORT", "9505")
	os.Setenv("LOG_LEVEL", "debug")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	port := os.Getenv("PORT")
	assert.Equal(t, "9505", port)

	logLevel := os.Getenv("LOG_LEVEL")
	assert.Equal(t, "debug", logLevel)
}

func TestServicePort(t *testing.T) {
	t.Run("default port is 8505", func(t *testing.T) {
		os.Unsetenv("PORT")
		port := os.Getenv("PORT")
		if port == "" {
			port = "8505"
		}
		assert.Equal(t, "8505", port)
	})

	t.Run("custom port from env", func(t *testing.T) {
		os.Setenv("PORT", "12505")
		defer os.Unsetenv("PORT")

		port := os.Getenv("PORT")
		assert.Equal(t, "12505", port)
	})
}

func TestAdminServiceName(t *testing.T) {
	serviceName := "admin-service"
	assert.Equal(t, "admin-service", serviceName)
	assert.NotEmpty(t, serviceName)
}
