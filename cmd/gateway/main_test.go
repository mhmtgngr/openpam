package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns existing env var", func(t *testing.T) {
		key := "TEST_GATEWAY_ENV"
		value := "test-value"
		os.Setenv(key, value)
		defer os.Unsetenv(key)

		result := getEnv(key, "default")
		assert.Equal(t, value, result)
	})

	t.Run("returns default when env var not set", func(t *testing.T) {
		key := "TEST_GATEWAY_NONEXISTENT"
		defaultVal := "default-value"

		result := getEnv(key, defaultVal)
		assert.Equal(t, defaultVal, result)
	})

	t.Run("returns default when env var is empty string", func(t *testing.T) {
		key := "TEST_GATEWAY_EMPTY"
		os.Setenv(key, "")
		defer os.Unsetenv(key)

		result := getEnv(key, "default")
		// The implementation treats empty string as not set
		assert.Equal(t, "default", result)
	})
}

func TestGetEnvInt(t *testing.T) {
	t.Run("returns valid integer", func(t *testing.T) {
		key := "TEST_GATEWAY_INT"
		os.Setenv(key, "5432")
		defer os.Unsetenv(key)

		result := getEnvInt(key, 3306)
		assert.Equal(t, 5432, result)
	})

	t.Run("returns default for invalid integer", func(t *testing.T) {
		key := "TEST_GATEWAY_INVALID_INT"
		os.Setenv(key, "not-a-number")
		defer os.Unsetenv(key)

		result := getEnvInt(key, 3306)
		assert.Equal(t, 3306, result)
	})

	t.Run("returns default when env var not set", func(t *testing.T) {
		key := "TEST_GATEWAY_INT_NOT_SET"
		defaultVal := 8500

		result := getEnvInt(key, defaultVal)
		assert.Equal(t, defaultVal, result)
	})

	t.Run("handles zero value", func(t *testing.T) {
		key := "TEST_GATEWAY_INT_ZERO"
		os.Setenv(key, "0")
		defer os.Unsetenv(key)

		result := getEnvInt(key, 8500)
		assert.Equal(t, 0, result)
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads default config", func(t *testing.T) {
		// Clear relevant env vars
		envVars := []string{
			"PORT", "LOG_LEVEL",
			"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
			"REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD", "REDIS_DB",
			"JWT_PRIVATE_KEY_PATH", "JWT_PUBLIC_KEY_PATH",
		}
		for _, env := range envVars {
			os.Unsetenv(env)
		}

		config := loadConfig()

		assert.Equal(t, "8500", config.Port)
		assert.Equal(t, "info", config.LogLevel)
		assert.Equal(t, "localhost", config.DBHost)
		assert.Equal(t, 5432, config.DBPort)
		assert.Equal(t, "openpam", config.DBUser)
		assert.Equal(t, "openpam", config.DBName)
		assert.Equal(t, "require", config.DBSSLMode)
		assert.Equal(t, "localhost", config.RedisHost)
		assert.Equal(t, 6379, config.RedisPort)
		assert.Equal(t, 0, config.RedisDB)
		assert.Equal(t, "/etc/openpam/jwt/private.pem", config.JWTPrivateKeyPath)
		assert.Equal(t, "/etc/openpam/jwt/public.pem", config.JWTPublicKeyPath)
	})

	t.Run("loads config from env vars", func(t *testing.T) {
		envVars := map[string]string{
			"PORT":                "9000",
			"LOG_LEVEL":           "debug",
			"DB_HOST":             "db.example.com",
			"DB_PORT":             "5433",
			"DB_USER":             "testuser",
			"DB_PASSWORD":         "testpass",
			"DB_NAME":             "testdb",
			"DB_SSLMODE":          "require",
			"REDIS_HOST":          "redis.example.com",
			"REDIS_PORT":          "6380",
			"REDIS_PASSWORD":      "redispass",
			"REDIS_DB":            "1",
			"JWT_PRIVATE_KEY_PATH": "/tmp/private.pem",
			"JWT_PUBLIC_KEY_PATH":  "/tmp/public.pem",
		}

		// Set env vars
		for k, v := range envVars {
			os.Setenv(k, v)
		}
		defer func() {
			for k := range envVars {
				os.Unsetenv(k)
			}
		}()

		config := loadConfig()

		assert.Equal(t, "9000", config.Port)
		assert.Equal(t, "debug", config.LogLevel)
		assert.Equal(t, "db.example.com", config.DBHost)
		assert.Equal(t, 5433, config.DBPort)
		assert.Equal(t, "testuser", config.DBUser)
		assert.Equal(t, "testpass", config.DBPassword)
		assert.Equal(t, "testdb", config.DBName)
		assert.Equal(t, "require", config.DBSSLMode)
		assert.Equal(t, "redis.example.com", config.RedisHost)
		assert.Equal(t, 6380, config.RedisPort)
		assert.Equal(t, "redispass", config.RedisPassword)
		assert.Equal(t, 1, config.RedisDB)
		assert.Equal(t, "/tmp/private.pem", config.JWTPrivateKeyPath)
		assert.Equal(t, "/tmp/public.pem", config.JWTPublicKeyPath)
	})
}
