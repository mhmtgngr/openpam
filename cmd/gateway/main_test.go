package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set Gin to test mode to disable logging
	gin.SetMode(gin.TestMode)
}

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
		assert.Equal(t, "disable", config.DBSSLMode)
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

func TestHandleListUsers(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListUsers(nil, logger)

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"users":[]`)
}

func TestHandleGetUser(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleGetUser(nil, logger)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user":null`)
}

func TestHandleCreateUser(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleCreateUser(nil, logger)

	req := httptest.NewRequest("POST", "/users", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"user":null`)
}

func TestHandleUpdateUser(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleUpdateUser(nil, logger)

	req := httptest.NewRequest("PUT", "/users/123", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user":null`)
}

func TestHandleDeleteUser(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleDeleteUser(nil, logger)

	req := httptest.NewRequest("DELETE", "/users/123", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"User deleted"`)
}

func TestHandleListTargets(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListTargets(nil, logger)

	req := httptest.NewRequest("GET", "/targets", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"targets":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleGetTarget(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleGetTarget(nil, logger)

	req := httptest.NewRequest("GET", "/targets/123", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"target":null`)
}

func TestHandleListCredentials(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListCredentials(nil, logger)

	req := httptest.NewRequest("GET", "/credentials", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"credentials":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleListSessions(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListSessions(nil, logger)

	req := httptest.NewRequest("GET", "/sessions", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sessions":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleTerminateSession(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleTerminateSession(nil, logger)

	req := httptest.NewRequest("POST", "/sessions/123/terminate", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"Session terminated"`)
}

func TestHandleListApprovalRequests(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListApprovalRequests(nil, logger)

	req := httptest.NewRequest("GET", "/approvals/requests", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"requests":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleApproveRequest(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleApproveRequest(nil, logger)

	req := httptest.NewRequest("POST", "/approvals/requests/123/approve", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"Request approved"`)
}

func TestHandleDenyRequest(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleDenyRequest(nil, logger)

	req := httptest.NewRequest("POST", "/approvals/requests/123/deny", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "123"}}
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"Request denied"`)
}

func TestHandleListAuditEvents(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListAuditEvents(nil, logger)

	req := httptest.NewRequest("GET", "/audit/events", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"events":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleExportAuditEvents(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleExportAuditEvents(nil, logger)

	req := httptest.NewRequest("GET", "/audit/export", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "attachment; filename=audit_export.json", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "[]", w.Body.String())
}

func TestHandleListDiscoveryScans(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListDiscoveryScans(nil, logger)

	req := httptest.NewRequest("GET", "/discovery/scans", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"scans":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleListDiscoveredAssets(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListDiscoveredAssets(nil, logger)

	req := httptest.NewRequest("GET", "/discovery/assets", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"assets":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleListTenants(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleListTenants(nil, logger)

	req := httptest.NewRequest("GET", "/admin/tenants", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"tenants":[]`)
	assert.Contains(t, w.Body.String(), `"total":0`)
}

func TestHandleSystemStats(t *testing.T) {
	logger := zerolog.Nop()
	handler := handleSystemStats(nil, logger)

	req := httptest.NewRequest("GET", "/admin/stats", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"stats":null`)
}
