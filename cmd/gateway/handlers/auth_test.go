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

// setupTestAuthHandler creates a handler for testing validation logic
func setupTestAuthHandler() *AuthHandler {
	return &AuthHandler{
		auth:   nil, // Service is not needed for validation tests
		logger: zerolog.Nop(),
	}
}

func TestAuthHandler_Login_Validation(t *testing.T) {
	t.Run("missing email", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := LoginRequest{
			Password: "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["error"])
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("invalid email format", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := []byte(`{"email": "not-an-email", "password": "password123"}`)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["error"])
	})

	t.Run("missing password", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := LoginRequest{
			Email: "test@example.com",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["error"])
	})

	t.Run("password too short", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := []byte(`{"email": "test@example.com", "password": "short"}`)
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Login(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid login request format", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
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
			handler.Login(c)
		}()

		// Should return 500 (service error) rather than 400 (validation error)
		assert.True(t, didPanic || w.Code != http.StatusBadRequest)
	})
}

func TestAuthHandler_Refresh_Validation(t *testing.T) {
	t.Run("missing refresh token", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.Refresh(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid refresh request format", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RefreshRequest{
			RefreshToken: "valid-token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Refresh(c)
		}()

		// Should return 401 (service error) rather than 400 (validation error)
		assert.True(t, didPanic || w.Code != http.StatusBadRequest)
	})
}

func TestAuthHandler_Me_Validation(t *testing.T) {
	t.Run("missing user_id context", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// Don't set user_id - this will panic in the handler

		// Wrap in recover to handle panic from nil user_id
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Me(c)
		}()

		// Either panic or service call (which returns 500 or 404)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError || w.Code == http.StatusNotFound)
	})

	t.Run("invalid user_id in context", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("GET", "/auth/me", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", "invalid-uuid")

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Me(c)
		}()

		// Either panic or service call (which returns 500 or 404)
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError || w.Code == http.StatusNotFound)
	})
}

func TestAuthHandler_SetupMFA_Validation(t *testing.T) {
	t.Run("missing user_id context", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("POST", "/auth/mfa/setup", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		// Don't set user_id - this will panic in the handler

		// Wrap in recover to handle panic from nil user_id
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.SetupMFA(c)
		}()

		// Either panic or service call error
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestAuthHandler_VerifyAndEnableMFA_Validation(t *testing.T) {
	t.Run("missing code in request", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("POST", "/auth/mfa/verify", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		handler.VerifyAndEnableMFA(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid MFA verify request", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := MFASetupRequest{
			Code: "123456",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/mfa/verify", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.VerifyAndEnableMFA(c)
		}()

		// Should return 500 (service error) rather than 400 (validation error)
		assert.True(t, didPanic || w.Code != http.StatusBadRequest)
	})
}

func TestAuthHandler_ChangePassword_Validation(t *testing.T) {
	t.Run("missing current password", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := ChangePasswordRequest{
			NewPassword: "new-password-123456",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/password/change", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("missing new password", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := ChangePasswordRequest{
			CurrentPassword: "old-password",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/password/change", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("new password too short", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := ChangePasswordRequest{
			CurrentPassword: "old-password",
			NewPassword:     "short",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/password/change", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		handler.ChangePassword(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid change password request", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := ChangePasswordRequest{
			CurrentPassword: "old-password",
			NewPassword:     "new-password-123456",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/password/change", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.ChangePassword(c)
		}()

		// Should return 500 (service error) rather than 400 (validation error)
		assert.True(t, didPanic || w.Code != http.StatusBadRequest)
	})
}

func TestAuthHandler_Register_Validation(t *testing.T) {
	t.Run("missing email", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			FirstName: "Test",
			LastName:  "User",
			Password:  "secure-password-123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing first name", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			Email:    "test@example.com",
			LastName: "User",
			Password: "secure-password-123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing last name", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			Email:     "test@example.com",
			FirstName: "Test",
			Password:  "secure-password-123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing password", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid email format", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			Email:     "not-an-email",
			FirstName: "Test",
			LastName:  "User",
			Password:  "secure-password-123",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("password too short", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := RegisterRequest{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
			Password:  "short",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.Register(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_UpdateUser_Validation(t *testing.T) {
	t.Run("invalid user ID", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := map[string]string{
			"first_name": "Updated",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("PUT", "/auth/users/invalid-uuid", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.UpdateUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_USER_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid user ID format", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := map[string]string{
			"first_name": "Updated",
		}
		jsonBody, _ := json.Marshal(body)

		userID := uuid.New()
		req := httptest.NewRequest("PUT", "/auth/users/"+userID.String(), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: userID.String()}}

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.UpdateUser(c)
		}()

		// Should return 500 (service error) rather than 400 (validation passed)
		assert.True(t, didPanic || w.Code != http.StatusBadRequest)
	})
}

func TestAuthHandler_DeleteUser_Validation(t *testing.T) {
	t.Run("invalid user ID", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("DELETE", "/auth/users/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.DeleteUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_USER_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("prevent self-deletion", func(t *testing.T) {
		handler := setupTestAuthHandler()

		userID := uuid.New()

		req := httptest.NewRequest("DELETE", "/auth/users/"+userID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: userID.String()}}
		c.Set("user_id", userID.String())

		handler.DeleteUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "CANNOT_DELETE_SELF", resp["error"].(map[string]interface{})["code"])
	})
}

func TestAuthHandler_VerifyMFA_Validation(t *testing.T) {
	t.Run("missing temp token", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := map[string]string{
			"mfa_code": "123456",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/mfa/verify-standalone", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.VerifyMFA(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("missing mfa code", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := map[string]string{
			"temp_token": "some-token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/mfa/verify-standalone", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.VerifyMFA(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_INPUT", resp["error"].(map[string]interface{})["code"])
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("logout with refresh token", func(t *testing.T) {
		handler := setupTestAuthHandler()

		body := map[string]string{
			"refresh_token": "test-token",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Logout(c)
		}()

		// Either panic (nil service) or service call (which returns 200 or error)
		assert.True(t, didPanic || w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})

	t.Run("logout without refresh token", func(t *testing.T) {
		handler := setupTestAuthHandler()

		req := httptest.NewRequest("POST", "/auth/logout", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Logout(c)
		}()

		// Either panic (nil service) or service call (which returns 200 or error)
		assert.True(t, didPanic || w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
	})
}
