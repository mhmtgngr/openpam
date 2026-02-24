package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestTargetHandler creates a handler for testing validation logic
func setupTestTargetHandler() *TargetHandler {
	return &TargetHandler{
		service: nil, // Service is not needed for validation tests
		logger:  zerolog.Nop(),
	}
}

func TestTargetHandler_List(t *testing.T) {
	t.Run("parse pagination parameters", func(t *testing.T) {
		handler := setupTestTargetHandler()

		req := httptest.NewRequest("GET", "/targets?limit=20&offset=40", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.List(c)
		}()

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})

	t.Run("parse filter parameters", func(t *testing.T) {
		handler := setupTestTargetHandler()

		req := httptest.NewRequest("GET", "/targets?type=ssh&environment=production", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.List(c)
		}()

		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestTargetHandler_Get(t *testing.T) {
	t.Run("invalid target ID", func(t *testing.T) {
		handler := setupTestTargetHandler()

		req := httptest.NewRequest("GET", "/targets/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.Get(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_TARGET_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid target ID format", func(t *testing.T) {
		handler := setupTestTargetHandler()
		targetID := uuid.New()

		req := httptest.NewRequest("GET", "/targets/"+targetID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: targetID.String()}}

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Get(c)
		}()

		// Either panic (nil service) or 404/500 (validation passed, service call failed)
		assert.True(t, didPanic || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

func TestTargetHandler_Create(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		handler := setupTestTargetHandler()

		body := CreateTargetRequest{
			Name: "test-target",
			// Missing Type, Environment, Host, Port, Sensitivity
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())
		c.Set("user_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid type", func(t *testing.T) {
		handler := setupTestTargetHandler()

		body := CreateTargetRequest{
			Name:        "test-target",
			Type:        "invalid-type",
			Environment: "production",
			Sensitivity: "medium",
			Host:        "192.168.1.1",
			Port:        22,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())
		c.Set("user_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid port - too low", func(t *testing.T) {
		handler := setupTestTargetHandler()

		body := CreateTargetRequest{
			Name:        "test-target",
			Type:        "ssh",
			Environment: "production",
			Sensitivity: "medium",
			Host:        "192.168.1.1",
			Port:        0,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())
		c.Set("user_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid port - too high", func(t *testing.T) {
		handler := setupTestTargetHandler()

		body := CreateTargetRequest{
			Name:        "test-target",
			Type:        "ssh",
			Environment: "production",
			Sensitivity: "medium",
			Host:        "192.168.1.1",
			Port:        70000,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())
		c.Set("user_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid types", func(t *testing.T) {
		validTypes := []string{"ssh", "rdp", "database", "web", "kubernetes", "api"}

		for _, targetType := range validTypes {
			handler := setupTestTargetHandler()

			body := CreateTargetRequest{
				Name:        "test-target",
				Type:        targetType,
				Environment: "production",
				Sensitivity: "medium",
				Host:        "192.168.1.1",
				Port:        22,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", uuid.New().String())
			c.Set("user_id", uuid.New().String())

			// Wrap in recover to handle nil service panic
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				handler.Create(c)
			}()

			// Should not return 400 (validation passed)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Type: %s", targetType)
		}
	})

	t.Run("valid environments", func(t *testing.T) {
		validEnvs := []string{"production", "staging", "development", "test"}

		for _, env := range validEnvs {
			handler := setupTestTargetHandler()

			body := CreateTargetRequest{
				Name:        "test-target",
				Type:        "ssh",
				Environment: env,
				Sensitivity: "medium",
				Host:        "192.168.1.1",
				Port:        22,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", uuid.New().String())
			c.Set("user_id", uuid.New().String())

			// Wrap in recover to handle nil service panic
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				handler.Create(c)
			}()

			// Should not return 400 (validation passed)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Environment: %s", env)
		}
	})

	t.Run("valid sensitivity levels", func(t *testing.T) {
		validLevels := []string{"low", "medium", "high", "critical"}

		for _, level := range validLevels {
			handler := setupTestTargetHandler()

			body := CreateTargetRequest{
				Name:        "test-target",
				Type:        "ssh",
				Environment: "production",
				Sensitivity: level,
				Host:        "192.168.1.1",
				Port:        22,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/targets", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("tenant_id", uuid.New().String())
			c.Set("user_id", uuid.New().String())

			// Wrap in recover to handle nil service panic
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				handler.Create(c)
			}()

			// Should not return 400 (validation passed)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Sensitivity: %s", level)
		}
	})
}

func TestTargetHandler_Update(t *testing.T) {
	t.Run("invalid target ID", func(t *testing.T) {
		handler := setupTestTargetHandler()

		body := UpdateTargetRequest{
			Name: targetStrPtr("updated-name"),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/targets/invalid-uuid", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.Update(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_TARGET_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid target ID format", func(t *testing.T) {
		handler := setupTestTargetHandler()
		targetID := uuid.New()

		body := UpdateTargetRequest{
			Name: targetStrPtr("updated-name"),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/targets/"+targetID.String(), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: targetID.String()}}

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Update(c)
		}()

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestTargetHandler_Delete(t *testing.T) {
	t.Run("invalid target ID", func(t *testing.T) {
		handler := setupTestTargetHandler()

		req := httptest.NewRequest("DELETE", "/targets/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Delete(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_TARGET_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid target ID format", func(t *testing.T) {
		handler := setupTestTargetHandler()
		targetID := uuid.New()

		req := httptest.NewRequest("DELETE", "/targets/"+targetID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: targetID.String()}}
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Delete(c)
		}()

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

// Helper functions
func targetStrPtr(s string) *string {
	return &s
}
