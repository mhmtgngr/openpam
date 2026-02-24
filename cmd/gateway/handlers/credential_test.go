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

// setupTestCredentialHandler creates a handler for testing validation logic
func setupTestCredentialHandler() *CredentialHandler {
	return &CredentialHandler{
		service: nil, // Service is not needed for validation tests
		logger:  zerolog.Nop(),
	}
}

func TestCredentialHandler_List(t *testing.T) {
	t.Run("parse pagination parameters", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("GET", "/credentials?limit=20&offset=40", nil)
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
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("GET", "/credentials?status=active&type=password", nil)
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

func TestCredentialHandler_Get(t *testing.T) {
	t.Run("invalid credential ID", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("GET", "/credentials/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.Get(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_CREDENTIAL_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid credential ID format", func(t *testing.T) {
		handler := setupTestCredentialHandler()
		credID := uuid.New()

		req := httptest.NewRequest("GET", "/credentials/"+credID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: credID.String()}}

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

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestCredentialHandler_Create(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		body := CreateCredentialRequest{
			Name: "test-credential",
			// Missing Type, Host, Port, Username
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(jsonBody))
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
		handler := setupTestCredentialHandler()

		body := CreateCredentialRequest{
			Name:     "test-credential",
			Type:     "invalid-type",
			Host:     "192.168.1.1",
			Port:     22,
			Username: "admin",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(jsonBody))
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
		validTypes := []string{"password", "ssh_key", "api_token", "certificate", "database", "aws_key", "azure_key"}

		for _, credType := range validTypes {
			handler := setupTestCredentialHandler()

			body := CreateCredentialRequest{
				Name:     "test-credential",
				Type:     credType,
				Host:     "192.168.1.1",
				Port:     22,
				Username: "admin",
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(jsonBody))
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
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Type: %s", credType)
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		body := CreateCredentialRequest{
			Name:     "test-credential",
			Type:     "password",
			Host:     "192.168.1.1",
			Port:     0, // Invalid port
			Username: "admin",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/credentials", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())
		c.Set("user_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCredentialHandler_Update(t *testing.T) {
	t.Run("invalid credential ID", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		body := UpdateCredentialRequest{
			Name: credStrPtr("updated-name"),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/credentials/invalid-uuid", bytes.NewBuffer(jsonBody))
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
		assert.Equal(t, "INVALID_CREDENTIAL_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid credential ID format", func(t *testing.T) {
		handler := setupTestCredentialHandler()
		credID := uuid.New()

		body := UpdateCredentialRequest{
			Name: credStrPtr("updated-name"),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/credentials/"+credID.String(), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: credID.String()}}

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

func TestCredentialHandler_Delete(t *testing.T) {
	t.Run("invalid credential ID", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("DELETE", "/credentials/invalid-uuid", nil)
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
		assert.Equal(t, "INVALID_CREDENTIAL_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid credential ID format", func(t *testing.T) {
		handler := setupTestCredentialHandler()
		credID := uuid.New()

		req := httptest.NewRequest("DELETE", "/credentials/"+credID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: credID.String()}}
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

func TestCredentialHandler_Reveal(t *testing.T) {
	t.Run("invalid credential ID", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("POST", "/credentials/invalid-uuid/reveal", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Reveal(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_CREDENTIAL_ID", resp["error"].(map[string]interface{})["code"])
	})
}

func TestCredentialHandler_Rotate(t *testing.T) {
	t.Run("invalid credential ID", func(t *testing.T) {
		handler := setupTestCredentialHandler()

		req := httptest.NewRequest("POST", "/credentials/invalid-uuid/rotate", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Rotate(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid credential ID format", func(t *testing.T) {
		handler := setupTestCredentialHandler()
		credID := uuid.New()

		body := map[string]string{
			"password": "new-password-123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/credentials/"+credID.String()+"/rotate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: credID.String()}}
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Rotate(c)
		}()

		// Either panic (nil service) or 404/500 (validation passed, service call failed)
		assert.True(t, didPanic || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

// Helper function
func credStrPtr(s string) *string {
	return &s
}
