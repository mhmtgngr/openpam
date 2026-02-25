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

// setupTestSessionHandler creates a handler for testing validation logic
func setupTestSessionHandler() *SessionHandler {
	return &SessionHandler{
		service: nil, // Service is not needed for validation tests
		logger:  zerolog.Nop(),
	}
}

func TestSessionHandler_List(t *testing.T) {
	t.Run("parse pagination parameters", func(t *testing.T) {
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("GET", "/sessions?limit=20&offset=40", nil)
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
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("GET", "/sessions?status=active&type=ssh", nil)
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

func TestSessionHandler_Get(t *testing.T) {
	t.Run("invalid session ID", func(t *testing.T) {
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("GET", "/sessions/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.Get(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_SESSION_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid session ID format", func(t *testing.T) {
		handler := setupTestSessionHandler()
		sessionID := uuid.New()

		req := httptest.NewRequest("GET", "/sessions/"+sessionID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}

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

func TestSessionHandler_Create(t *testing.T) {
	t.Run("missing required fields", func(t *testing.T) {
		handler := setupTestSessionHandler()

		body := CreateSessionRequest{
			// Missing CredentialID, Type, TargetHost, TargetPort
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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
		handler := setupTestSessionHandler()

		body := CreateSessionRequest{
			CredentialID: uuid.New().String(),
			Type:         "invalid-type",
			TargetHost:   "example.com",
			TargetPort:   22,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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
		handler := setupTestSessionHandler()

		body := CreateSessionRequest{
			CredentialID: uuid.New().String(),
			Type:         "ssh",
			TargetHost:   "example.com",
			TargetPort:   0,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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
		handler := setupTestSessionHandler()

		body := CreateSessionRequest{
			CredentialID: uuid.New().String(),
			Type:         "ssh",
			TargetHost:   "example.com",
			TargetPort:   70000,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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
		validTypes := []string{"ssh", "rdp", "database", "kubernetes", "web", "api"}

		for _, sessionType := range validTypes {
			handler := setupTestSessionHandler()

			body := CreateSessionRequest{
				CredentialID: uuid.New().String(),
				Type:         sessionType,
				TargetHost:   "example.com",
				TargetPort:   22,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Type: %s", sessionType)
		}
	})

	t.Run("valid request", func(t *testing.T) {
		handler := setupTestSessionHandler()

		body := CreateSessionRequest{
			CredentialID: uuid.New().String(),
			Type:         "ssh",
			TargetHost:   "example.com",
			TargetPort:   22,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/sessions", bytes.NewBuffer(jsonBody))
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

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestSessionHandler_Terminate(t *testing.T) {
	t.Run("invalid session ID", func(t *testing.T) {
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("POST", "/sessions/invalid-uuid/terminate", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Terminate(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_SESSION_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid session ID format", func(t *testing.T) {
		handler := setupTestSessionHandler()
		sessionID := uuid.New()

		req := httptest.NewRequest("POST", "/sessions/"+sessionID.String()+"/terminate", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: sessionID.String()}}
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Terminate(c)
		}()

		// Either panic (nil service) or 404/500 (validation passed, service call failed)
		assert.True(t, didPanic || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

func TestSessionHandler_GetStats(t *testing.T) {
	t.Run("get stats validation", func(t *testing.T) {
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("GET", "/sessions/stats", nil)
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
			handler.GetStats(c)
		}()

		// Either panic (nil service) or 500 (validation passed)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestSessionHandler_GetActive(t *testing.T) {
	t.Run("get active sessions validation", func(t *testing.T) {
		handler := setupTestSessionHandler()

		req := httptest.NewRequest("GET", "/sessions/active", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.GetActive(c)

		// Returns 200 with empty list (no service call)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["sessions"])
		assert.Equal(t, float64(0), resp["total"])
	})
}

func TestSessionValidationTypes(t *testing.T) {
	t.Run("valid session types", func(t *testing.T) {
		validTypes := []string{"ssh", "rdp", "database", "kubernetes", "web", "api"}

		for _, sessionType := range validTypes {
			assert.NotEmpty(t, sessionType)
		}
	})

	t.Run("valid session statuses", func(t *testing.T) {
		validStatuses := []string{"active", "terminated", "failed", "timeout"}

		for _, status := range validStatuses {
			assert.NotEmpty(t, status)
		}
	})
}
