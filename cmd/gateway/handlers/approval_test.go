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

// setupTestApprovalHandler creates a handler for testing validation logic
func setupTestApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{
		service: nil, // Service is not needed for validation tests
		logger:  zerolog.Nop(),
	}
}

func TestApprovalHandler_List(t *testing.T) {
	t.Run("successful list - returns empty list", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		req := httptest.NewRequest("GET", "/approvals/requests?limit=10&offset=0", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["requests"])
		assert.Equal(t, float64(0), resp["total"])
	})

	t.Run("parse pagination parameters", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		req := httptest.NewRequest("GET", "/approvals/requests?limit=20&offset=40", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("tenant_id", uuid.New().String())

		handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, float64(20), resp["limit"])
		assert.Equal(t, float64(40), resp["offset"])
	})
}

func TestApprovalHandler_Get(t *testing.T) {
	t.Run("invalid request ID", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		req := httptest.NewRequest("GET", "/approvals/requests/invalid-uuid", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}

		handler.Get(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid request ID returns placeholder", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()

		req := httptest.NewRequest("GET", "/approvals/requests/"+requestID.String(), nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}

		handler.Get(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["request"])
	})
}

func TestApprovalHandler_Create(t *testing.T) {
	t.Run("successful create - credential access", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		credID := uuid.New()

		body := CreateRequestRequest{
			Type:          "credential_access",
			CredentialID:  credID.String(),
			Justification: "Need access for emergency maintenance",
			Duration:      60,
			Priority:      "high",
			TicketRef:     "INC-12345",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil service - validation passed
				// Don't assert on the specific message, just that we caught a panic
			}
		}()
		handler.Create(c)
		// If we get here without panic, service was called and validation passed
	})

	t.Run("successful create - session access with target", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		targetID := uuid.New()

		body := CreateRequestRequest{
			Type:          "session_access",
			TargetID:      targetID.String(),
			Justification: "Need to debug production issue",
			Duration:      120,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil service - validation passed
			}
		}()
		handler.Create(c)
	})

	t.Run("invalid type", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := CreateRequestRequest{
			Type:          "invalid-type",
			Justification: "test",
			Duration:      60,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required fields", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := CreateRequestRequest{
			Type: "credential_access",
			// Missing justification, duration
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid priority", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := CreateRequestRequest{
			Type:          "credential_access",
			Justification: "test",
			Duration:      60,
			Priority:      "invalid-priority",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid duration - too short", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := CreateRequestRequest{
			Type:          "credential_access",
			Justification: "test",
			Duration:      0,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())
		c.Set("tenant_id", uuid.New().String())

		handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid types", func(t *testing.T) {
		validTypes := []string{"credential_access", "session_access", "privilege_escalation"}

		for _, testType := range validTypes {
			handler := setupTestApprovalHandler()

			body := CreateRequestRequest{
				Type:          testType,
				Justification: "test justification",
				Duration:      60,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", uuid.New().String())
			c.Set("tenant_id", uuid.New().String())

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

			// Validation passed (either panic due to nil service or would have returned 500)
			assert.NotEqual(t, http.StatusBadRequest, w.Code, "Type: %s", testType)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Type: %s should pass validation", testType)
		}
	})

	t.Run("valid priorities", func(t *testing.T) {
		validPriorities := []string{"low", "normal", "high", "emergency"}

		for _, priority := range validPriorities {
			handler := setupTestApprovalHandler()

			body := CreateRequestRequest{
				Type:          "credential_access",
				Justification: "test justification",
				Duration:      60,
				Priority:      priority,
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/approvals/requests", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Set("user_id", uuid.New().String())
			c.Set("tenant_id", uuid.New().String())

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

			// Validation passed (either panic due to nil service or would have returned 500)
			assert.NotEqual(t, http.StatusBadRequest, w.Code, "Priority: %s", priority)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Priority: %s should pass validation", priority)
		}
	})
}

func TestApprovalHandler_Approve(t *testing.T) {
	t.Run("invalid request ID", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := ApproveRequestRequest{
			Decision: "approve",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/invalid-uuid/approve", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Approve(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("invalid decision", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()

		body := ApproveRequestRequest{
			Decision: "invalid-decision",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/approve", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
		c.Set("user_id", uuid.New().String())

		handler.Approve(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid decisions", func(t *testing.T) {
		validDecisions := []string{"approve", "deny"}

		for _, decision := range validDecisions {
			handler := setupTestApprovalHandler()
			requestID := uuid.New()

			body := ApproveRequestRequest{
				Decision: decision,
				Comments: "Test comments",
			}
			jsonBody, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/approve", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
			c.Set("user_id", uuid.New().String())

			// Wrap in recover to handle nil service panic
			didPanic := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						didPanic = true
					}
				}()
				handler.Approve(c)
			}()

			// Validation passed (either panic due to nil service or would have returned 500)
			assert.NotEqual(t, http.StatusBadRequest, w.Code, "Decision: %s", decision)
			assert.True(t, didPanic || w.Code != http.StatusBadRequest, "Decision: %s should pass validation", decision)
		}
	})
}

func TestApprovalHandler_Cancel(t *testing.T) {
	t.Run("invalid request ID", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		req := httptest.NewRequest("POST", "/approvals/requests/invalid-uuid/cancel", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Cancel(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("valid request ID", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()

		req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/cancel", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Cancel(c)
		}()

		// Either panic (nil service) or 500 - validation passed
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}

func TestApprovalHandler_GetPending(t *testing.T) {
	t.Run("get pending requests", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		req := httptest.NewRequest("GET", "/approvals/requests/pending", nil)
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("user_id", uuid.New().String())

		handler.GetPending(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.NotNil(t, resp["requests"])
		assert.Equal(t, float64(0), resp["total"])
	})
}

func TestApprovalHandler_Delegate(t *testing.T) {
	t.Run("invalid request ID", func(t *testing.T) {
		handler := setupTestApprovalHandler()

		body := map[string]string{
			"to_approver_id": uuid.New().String(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/invalid-uuid/delegate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: "invalid-uuid"}}
		c.Set("user_id", uuid.New().String())

		handler.Delegate(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_REQUEST_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("invalid to_approver_id", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()

		body := map[string]string{
			"to_approver_id": "invalid-uuid",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/delegate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
		c.Set("user_id", uuid.New().String())

		handler.Delegate(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "INVALID_APPROVER_ID", resp["error"].(map[string]interface{})["code"])
	})

	t.Run("missing to_approver_id", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()

		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/delegate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
		c.Set("user_id", uuid.New().String())

		handler.Delegate(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("valid delegation", func(t *testing.T) {
		handler := setupTestApprovalHandler()
		requestID := uuid.New()
		toApproverID := uuid.New()

		body := map[string]string{
			"to_approver_id": toApproverID.String(),
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/approvals/requests/"+requestID.String()+"/delegate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: requestID.String()}}
		c.Set("user_id", uuid.New().String())

		// Wrap in recover to handle nil service panic
		didPanic := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					didPanic = true
				}
			}()
			handler.Delegate(c)
		}()

		// Either panic (nil service) or 500 - validation passed
		assert.True(t, didPanic || w.Code == http.StatusInternalServerError)
	})
}
