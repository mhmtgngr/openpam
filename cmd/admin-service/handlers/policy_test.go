package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/events"
	"github.com/openpam/openpam/internal/pam/policy"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTestHandlerRepository is a mock repository for testing handlers
type mockTestHandlerRepository struct {
	policies         map[uuid.UUID]*policy.Policy
	evalLogs         []policy.PolicyEvalLog
	approvalRequests map[uuid.UUID]*policy.ApprovalRequest
	templates        []policy.PolicyTemplate
	commandPatterns  []policy.CommandFilterPattern
	createError      bool
	getError         bool
	updateError      bool
	deleteError      bool
	enableError      bool
	disableError     bool
}

func newMockTestHandlerRepository() *mockTestHandlerRepository {
	return &mockTestHandlerRepository{
		policies:         make(map[uuid.UUID]*policy.Policy),
		approvalRequests: make(map[uuid.UUID]*policy.ApprovalRequest),
		templates:        []policy.PolicyTemplate{},
		commandPatterns:  []policy.CommandFilterPattern{},
	}
}

func (m *mockTestHandlerRepository) Create(ctx context.Context, p *policy.Policy) error {
	if m.createError {
		return errors.New("repository: create failed")
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.policies[p.ID] = p
	return nil
}

func (m *mockTestHandlerRepository) GetByID(ctx context.Context, id uuid.UUID) (*policy.Policy, error) {
	if m.getError {
		return nil, errors.New("repository: get failed")
	}
	return m.policies[id], nil
}

func (m *mockTestHandlerRepository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*policy.Policy, error) {
	if m.getError {
		return nil, errors.New("repository: get failed")
	}
	p, ok := m.policies[id]
	if !ok || p.TenantID != tenantID {
		return nil, errors.New("repository: not found")
	}
	return p, nil
}

func (m *mockTestHandlerRepository) List(ctx context.Context, tenantID uuid.UUID, filter policy.PolicyFilter, limit, offset int) ([]policy.Policy, int, error) {
	var result []policy.Policy
	for _, p := range m.policies {
		if p.TenantID != tenantID {
			continue
		}
		if filter.Type != nil && p.Type != *filter.Type {
			continue
		}
		if filter.Effect != nil && p.Effect != *filter.Effect {
			continue
		}
		if filter.Enabled != nil && p.Enabled != *filter.Enabled {
			continue
		}
		if filter.Search != "" && !contains(p.Name, filter.Search) {
			continue
		}
		result = append(result, *p)
	}
	return result, len(result), nil
}

func (m *mockTestHandlerRepository) GetActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]policy.Policy, error) {
	var result []policy.Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockTestHandlerRepository) GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]policy.Policy, error) {
	var result []policy.Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockTestHandlerRepository) Update(ctx context.Context, p *policy.Policy) error {
	if m.updateError {
		return errors.New("repository: update failed")
	}
	p.UpdatedAt = time.Now()
	m.policies[p.ID] = p
	return nil
}

func (m *mockTestHandlerRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteError {
		return errors.New("repository: delete failed")
	}
	delete(m.policies, id)
	return nil
}

func (m *mockTestHandlerRepository) Enable(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.enableError {
		return errors.New("repository: enable failed")
	}
	if p, ok := m.policies[id]; ok {
		p.Enabled = true
	}
	return nil
}

func (m *mockTestHandlerRepository) Disable(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.disableError {
		return errors.New("repository: disable failed")
	}
	if p, ok := m.policies[id]; ok {
		p.Enabled = false
	}
	return nil
}

func (m *mockTestHandlerRepository) CreateEvaluationLog(ctx context.Context, log *policy.PolicyEvalLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	m.evalLogs = append(m.evalLogs, *log)
	return nil
}

func (m *mockTestHandlerRepository) GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]policy.PolicyEvalLog, int, error) {
	return m.evalLogs, len(m.evalLogs), nil
}

func (m *mockTestHandlerRepository) CreateApprovalRequest(ctx context.Context, req *policy.ApprovalRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	req.UpdatedAt = time.Now()
	if req.Status == "" {
		req.Status = policy.ApprovalStatusPending
	}
	m.approvalRequests[req.ID] = req
	return nil
}

func (m *mockTestHandlerRepository) GetApprovalRequestByID(ctx context.Context, id uuid.UUID) (*policy.ApprovalRequest, error) {
	req, ok := m.approvalRequests[id]
	if !ok {
		return nil, errors.New("repository: approval request not found")
	}
	return req, nil
}

func (m *mockTestHandlerRepository) UpdateApprovalRequestStatus(ctx context.Context, id uuid.UUID, status policy.ApprovalStatus, approverID *uuid.UUID, denialReason string) error {
	if req, ok := m.approvalRequests[id]; ok {
		req.Status = status
		req.ApproverID = approverID
		if status == policy.ApprovalStatusApproved {
			now := time.Now()
			req.ApprovedAt = &now
		}
		req.DenialReason = denialReason
	}
	return nil
}

func (m *mockTestHandlerRepository) GetPendingApprovalRequests(ctx context.Context, tenantID uuid.UUID, approverID *uuid.UUID) ([]policy.ApprovalRequest, error) {
	var result []policy.ApprovalRequest
	for _, req := range m.approvalRequests {
		if req.TenantID == tenantID && req.Status == policy.ApprovalStatusPending {
			result = append(result, *req)
		}
	}
	return result, nil
}

func (m *mockTestHandlerRepository) CreatePolicyTemplate(ctx context.Context, template *policy.PolicyTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = template.CreatedAt
	m.templates = append(m.templates, *template)
	return nil
}

func (m *mockTestHandlerRepository) GetPolicyTemplates(ctx context.Context, category *string) ([]policy.PolicyTemplate, error) {
	return m.templates, nil
}

func (m *mockTestHandlerRepository) CreateCommandFilterPattern(ctx context.Context, pattern *policy.CommandFilterPattern) error {
	pattern.ID = uuid.New()
	pattern.CreatedAt = time.Now()
	m.commandPatterns = append(m.commandPatterns, *pattern)
	return nil
}

func (m *mockTestHandlerRepository) GetCommandFilterPatterns(ctx context.Context, policyID uuid.UUID) ([]policy.CommandFilterPattern, error) {
	return m.commandPatterns, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

// mockPolicyCache is a mock cache implementation for testing
type mockPolicyCache struct {
	policies    map[uuid.UUID]*policy.Policy
	evalResults map[string]*policy.EvaluationResponse
}

func newMockPolicyCache() *mockPolicyCache {
	return &mockPolicyCache{
		policies:    make(map[uuid.UUID]*policy.Policy),
		evalResults: make(map[string]*policy.EvaluationResponse),
	}
}

func (m *mockPolicyCache) GetPolicy(ctx context.Context, id uuid.UUID) (*policy.Policy, error) {
	policy, ok := m.policies[id]
	if !ok {
		return nil, errors.New("cache: not found")
	}
	return policy, nil
}

func (m *mockPolicyCache) SetPolicy(ctx context.Context, policy *policy.Policy, ttl time.Duration) error {
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyCache) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	delete(m.policies, id)
	return nil
}

func (m *mockPolicyCache) GetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string) ([]policy.Policy, int, error) {
	return nil, 0, errors.New("cache: not found")
}

func (m *mockPolicyCache) SetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string, policies []policy.Policy, total int, ttl time.Duration) error {
	return nil
}

func (m *mockPolicyCache) InvalidPolicyList(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) GetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID) ([]policy.Policy, error) {
	return nil, errors.New("cache: not found")
}

func (m *mockPolicyCache) SetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID, policies []policy.Policy, ttl time.Duration) error {
	return nil
}

func (m *mockPolicyCache) InvalidateApplicablePolicies(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) GetEvalResult(ctx context.Context, cacheKey string) (*policy.EvaluationResponse, error) {
	result, ok := m.evalResults[cacheKey]
	if !ok {
		return nil, errors.New("cache: not found")
	}
	return result, nil
}

func (m *mockPolicyCache) SetEvalResult(ctx context.Context, cacheKey string, result *policy.EvaluationResponse, ttl time.Duration) error {
	m.evalResults[cacheKey] = result
	return nil
}

func (m *mockPolicyCache) InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) InvalidateOnPolicyChange(ctx context.Context, tenantID uuid.UUID, policyID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) Warmup(ctx context.Context, tenantID uuid.UUID, policies []policy.Policy) error {
	return nil
}

// testContext holds test context values
type testContext struct {
	tenantID string
	userID   string
}

// createTestService creates a test policy service with mock repository
func createTestService() *policy.Service {
	repo := newMockTestHandlerRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewForTest(nil, zerolog.Nop())
	logger := zerolog.Nop()

	return policy.NewTestService(repo, mockCache, eventBus, logger)
}

// getTestRepo returns the mock repository from the service for test setup
func getTestRepo(service *policy.Service) *mockTestHandlerRepository {
	// Since we can't access the private repo field, we'll use service methods
	// and create a separate mock for test data setup
	return newMockTestHandlerRepository()
}

// setupTestRouter sets up a test router with policy handlers
func setupTestRouter(service *policy.Service, testCtx testContext) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set test context
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", testCtx.tenantID)
		c.Set("user_id", testCtx.userID)
		c.Set("user_roles", []uuid.UUID{})
		c.Next()
	})

	handler := NewPolicyHandler(service, zerolog.Nop())

	// Register routes
	router.GET("/api/v1/policies", handler.ListPolicies())
	router.POST("/api/v1/policies", handler.CreatePolicy())
	router.GET("/api/v1/policies/:id", handler.GetPolicy())
	router.PUT("/api/v1/policies/:id", handler.UpdatePolicy())
	router.DELETE("/api/v1/policies/:id", handler.DeletePolicy())
	router.POST("/api/v1/policies/:id/enable", handler.EnablePolicy())
	router.POST("/api/v1/policies/:id/disable", handler.DisablePolicy())
	router.POST("/api/v1/policies/evaluate", handler.EvaluatePolicy())
	router.GET("/api/v1/policies/evaluation-logs", handler.GetEvaluationLogs())
	router.GET("/api/v1/policies/templates", handler.GetPolicyTemplates())
	router.POST("/api/v1/policies/approvals", handler.CreateApprovalRequest())
	router.GET("/api/v1/policies/approvals/pending", handler.GetPendingApprovals())
	router.POST("/api/v1/policies/approvals/:id/action", handler.ProcessApprovalRequest())
	router.POST("/api/v1/policies/command-filters", handler.AddCommandFilter())

	return router
}

// newTestContext creates a new test context with random IDs
func newTestContext() testContext {
	return testContext{
		tenantID: uuid.New().String(),
		userID:   uuid.New().String(),
	}
}

// TestPolicyHandler_ListPolicies tests listing policies
func TestPolicyHandler_ListPolicies(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	// Create test policies
	for i := 0; i < 3; i++ {
		p := &policy.Policy{
			Name:     "Test Policy",
			Type:     policy.PolicyTypeAccess,
			Effect:   policy.PolicyEffectAllow,
			Priority: 100 + i,
			TenantID: tenantUUID,
			Enabled:  true,
			Rules:    []policy.Rule{},
		}
		require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))
	}

	req, _ := http.NewRequest("GET", "/api/v1/policies?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["policies"])
	assert.NotEmpty(t, response["total"])
}

// TestPolicyHandler_CreatePolicy tests creating a policy
func TestPolicyHandler_CreatePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	createReq := CreatePolicyRequest{
		Name:        "New Policy",
		Description: "A new test policy",
		Type:        policy.PolicyTypeAccess,
		Effect:      policy.PolicyEffectAllow,
		Priority:    100,
		Enabled:     true,
		Rules:       []policy.Rule{},
	}

	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/policies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["policy"])
}

// TestPolicyHandler_CreatePolicy_ValidationError tests validation on policy creation
func TestPolicyHandler_CreatePolicy_ValidationError(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	createReq := CreatePolicyRequest{
		// Missing required name field
		Type:   policy.PolicyTypeAccess,
		Effect: policy.PolicyEffectAllow,
	}

	body, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/policies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["error"])
}

// TestPolicyHandler_GetPolicy tests getting a single policy
func TestPolicyHandler_GetPolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	p := &policy.Policy{
		Name:     "Test Policy",
		Type:     policy.PolicyTypeAccess,
		Effect:   policy.PolicyEffectAllow,
		Priority: 100,
		TenantID: tenantUUID,
		Enabled:  true,
		Rules:    []policy.Rule{},
	}
	require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))

	req, _ := http.NewRequest("GET", "/api/v1/policies/"+p.ID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["policy"])
}

// TestPolicyHandler_GetPolicy_NotFound tests getting a non-existent policy
func TestPolicyHandler_GetPolicy_NotFound(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	policyID := uuid.New()

	req, _ := http.NewRequest("GET", "/api/v1/policies/"+policyID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestPolicyHandler_GetPolicy_InvalidID tests getting a policy with invalid ID
func TestPolicyHandler_GetPolicy_InvalidID(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	req, _ := http.NewRequest("GET", "/api/v1/policies/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["error"])
}

// TestPolicyHandler_UpdatePolicy tests updating a policy
func TestPolicyHandler_UpdatePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	p := &policy.Policy{
		Name:     "Original Name",
		Type:     policy.PolicyTypeAccess,
		Effect:   policy.PolicyEffectAllow,
		Priority: 100,
		TenantID: tenantUUID,
		Enabled:  true,
		Rules:    []policy.Rule{},
	}
	require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))

	updatedName := "Updated Name"
	updateReq := UpdatePolicyRequest{
		Name: updatedName,
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/v1/policies/"+p.ID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["policy"])

	// Verify the policy was updated
	updatedPolicy, err := service.GetPolicy(context.Background(), p.ID, tenantUUID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedPolicy.Name)
}

// TestPolicyHandler_DeletePolicy tests deleting a policy
func TestPolicyHandler_DeletePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	p := &policy.Policy{
		Name:         "Test Policy",
		Type:         policy.PolicyTypeAccess,
		Effect:       policy.PolicyEffectAllow,
		Priority:     100,
		TenantID:     tenantUUID,
		SystemPolicy: false,
		Enabled:      true,
		Rules:        []policy.Rule{},
	}
	require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))

	req, _ := http.NewRequest("DELETE", "/api/v1/policies/"+p.ID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the policy was deleted
	_, err := service.GetPolicy(context.Background(), p.ID, tenantUUID)
	assert.Error(t, err)
}

// TestPolicyHandler_EnablePolicy tests enabling a policy
func TestPolicyHandler_EnablePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	p := &policy.Policy{
		Name:     "Test Policy",
		Type:     policy.PolicyTypeAccess,
		Effect:   policy.PolicyEffectAllow,
		Priority: 100,
		TenantID: tenantUUID,
		Enabled:  false,
		Rules:    []policy.Rule{},
	}
	require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))

	req, _ := http.NewRequest("POST", "/api/v1/policies/"+p.ID.String()+"/enable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	enabledPolicy, err := service.GetPolicy(context.Background(), p.ID, tenantUUID)
	require.NoError(t, err)
	assert.True(t, enabledPolicy.Enabled)
}

// TestPolicyHandler_DisablePolicy tests disabling a policy
func TestPolicyHandler_DisablePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)

	p := &policy.Policy{
		Name:     "Test Policy",
		Type:     policy.PolicyTypeAccess,
		Effect:   policy.PolicyEffectAllow,
		Priority: 100,
		TenantID: tenantUUID,
		Enabled:  true,
		Rules:    []policy.Rule{},
	}
	require.NoError(t, service.CreatePolicy(context.Background(), p, uuid.New()))

	req, _ := http.NewRequest("POST", "/api/v1/policies/"+p.ID.String()+"/disable", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	disabledPolicy, err := service.GetPolicy(context.Background(), p.ID, tenantUUID)
	require.NoError(t, err)
	assert.False(t, disabledPolicy.Enabled)
}

// TestPolicyHandler_EvaluatePolicy tests policy evaluation
func TestPolicyHandler_EvaluatePolicy(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	userUUID, _ := uuid.Parse(testCtx.userID)

	evalReq := EvaluatePolicyRequest{
		UserID:       userUUID,
		UserRoles:    []uuid.UUID{},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   uuid.New(),
		ClientIP:     "192.168.1.1",
		MFAVerified:  true,
		MFAMethod:    policy.MFAMethodTOTP, // Must set valid MFA method when verified
	}

	body, _ := json.Marshal(evalReq)
	req, _ := http.NewRequest("POST", "/api/v1/policies/evaluate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["result"])
}

// TestPolicyHandler_GetEvaluationLogs tests getting evaluation logs
func TestPolicyHandler_GetEvaluationLogs(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	req, _ := http.NewRequest("GET", "/api/v1/policies/evaluation-logs?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that logs and total fields exist (they may be empty)
	_, hasLogs := response["logs"]
	_, hasTotal := response["total"]
	assert.True(t, hasLogs, "Response should have logs field")
	assert.True(t, hasTotal, "Response should have total field")
}

// TestPolicyHandler_GetPolicyTemplates tests getting policy templates
func TestPolicyHandler_GetPolicyTemplates(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	req, _ := http.NewRequest("GET", "/api/v1/policies/templates", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that templates field exists (it may be empty in mock)
	_, hasTemplates := response["templates"]
	assert.True(t, hasTemplates, "Response should have templates field")
}

// TestPolicyHandler_CreateApprovalRequest tests creating an approval request
func TestPolicyHandler_CreateApprovalRequest(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	targetID := uuid.New()

	// Create request body with status included
	approvalReqBody := map[string]interface{}{
		"target_type": "credential",
		"target_id":   targetID.String(),
		"reason":      "Need access for maintenance",
		"expires_at":  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		"status":      "pending",
	}

	body, _ := json.Marshal(approvalReqBody)
	req, _ := http.NewRequest("POST", "/api/v1/policies/approvals", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["request"])
}

// TestPolicyHandler_GetPendingApprovals tests getting pending approvals
func TestPolicyHandler_GetPendingApprovals(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	req, _ := http.NewRequest("GET", "/api/v1/policies/approvals/pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check that requests field exists (it may be empty in mock)
	_, hasRequests := response["requests"]
	assert.True(t, hasRequests, "Response should have requests field")
}

// TestPolicyHandler_ProcessApprovalRequest tests processing an approval request
func TestPolicyHandler_ProcessApprovalRequest(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	tenantUUID, _ := uuid.Parse(testCtx.tenantID)
	userUUID, _ := uuid.Parse(testCtx.userID)

	requestID := uuid.New()

	// Create an approval request
	approvalReq := &policy.ApprovalRequest{
		ID:          requestID,
		TenantID:    tenantUUID,
		RequesterID: userUUID,
		TargetType:  "credential",
		TargetID:    uuid.New(),
		Reason:      "Test approval",
		Status:      policy.ApprovalStatusPending,
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, service.RequestApproval(context.Background(), approvalReq))

	t.Run("approve", func(t *testing.T) {
		actionReq := ApprovalActionRequest{
			Action: "approve",
		}

		body, _ := json.Marshal(actionReq)
		req, _ := http.NewRequest("POST", "/api/v1/policies/approvals/"+requestID.String()+"/action", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Create another request for deny test
	requestID2 := uuid.New()
	approvalReq2 := &policy.ApprovalRequest{
		ID:          requestID2,
		TenantID:    tenantUUID,
		RequesterID: userUUID,
		TargetType:  "credential",
		TargetID:    uuid.New(),
		Reason:      "Test approval 2",
		Status:      policy.ApprovalStatusPending,
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, service.RequestApproval(context.Background(), approvalReq2))

	t.Run("deny", func(t *testing.T) {
		actionReq := ApprovalActionRequest{
			Action: "deny",
			Reason: "Not approved",
		}

		body, _ := json.Marshal(actionReq)
		req, _ := http.NewRequest("POST", "/api/v1/policies/approvals/"+requestID2.String()+"/action", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestPolicyHandler_AddCommandFilter tests adding a command filter
func TestPolicyHandler_AddCommandFilter(t *testing.T) {
	service := createTestService()
	testCtx := newTestContext()
	router := setupTestRouter(service, testCtx)

	filterReq := CommandFilterRequest{
		PolicyID:    uuid.New(),
		Pattern:     "rm -rf.*",
		IsWhitelist: false,
		Description: "Block dangerous file deletion",
	}

	body, _ := json.Marshal(filterReq)
	req, _ := http.NewRequest("POST", "/api/v1/policies/command-filters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["pattern"])
}

// TestPolicyHandler_ParsePolicyFilter tests policy filter parsing
func TestPolicyHandler_ParsePolicyFilter(t *testing.T) {
	service := createTestService()
	handler := NewPolicyHandler(service, zerolog.Nop())

	t.Run("with type filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/policies?type=access", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		filter := handler.parsePolicyFilter(c)
		assert.NotNil(t, filter.Type)
		assert.Equal(t, policy.PolicyTypeAccess, *filter.Type)
	})

	t.Run("with effect filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/policies?effect=allow", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		filter := handler.parsePolicyFilter(c)
		assert.NotNil(t, filter.Effect)
		assert.Equal(t, policy.PolicyEffectAllow, *filter.Effect)
	})

	t.Run("with enabled filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/policies?enabled=true", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		filter := handler.parsePolicyFilter(c)
		assert.NotNil(t, filter.Enabled)
		assert.True(t, *filter.Enabled)
	})

	t.Run("with search filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/policies?search=database", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		filter := handler.parsePolicyFilter(c)
		assert.Equal(t, "database", filter.Search)
	})

	t.Run("with tags filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/policies?tags=production&tags=critical", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		filter := handler.parsePolicyFilter(c)
		assert.Len(t, filter.Tags, 2)
		assert.Contains(t, filter.Tags, "production")
		assert.Contains(t, filter.Tags, "critical")
	})
}

// TestPolicyHandler_BuildPolicyFromRequest tests building a policy from request
func TestPolicyHandler_BuildPolicyFromRequest(t *testing.T) {
	service := createTestService()
	handler := NewPolicyHandler(service, zerolog.Nop())

	tenantID := uuid.New().String()
	userID := uuid.New().String()
	targetID := uuid.New()
	credentialID := uuid.New()
	roleID := uuid.New()

	createReq := CreatePolicyRequest{
		Name:                      "Test Policy",
		Description:               "A test policy",
		Type:                      policy.PolicyTypeAccess,
		Effect:                    policy.PolicyEffectAllow,
		Priority:                  100,
		UserIDs:                   []uuid.UUID{},
		RoleIDs:                   []uuid.UUID{roleID},
		GroupIDs:                  []uuid.UUID{},
		TargetIDs:                 []uuid.UUID{targetID},
		CredentialIDs:             []uuid.UUID{credentialID},
		Rules:                     []policy.Rule{},
		MFARequired:               true,
		MFAMethods:                []policy.MFAMethod{policy.MFAMethodTOTP},
		ApprovalRequired:          false,
		RecordingRequired:         true,
		RecordingMode:             policy.RecordingModeAll,
		Enabled:                   true,
		SessionExtensionAllowed:   true,
		ApprovalTimeoutMinutes:    60,
		Tags:                      []string{"test", "important"},
	}

	builtPolicy := handler.buildPolicyFromRequest(createReq, tenantID, userID)

	assert.Equal(t, "Test Policy", builtPolicy.Name)
	assert.Equal(t, policy.PolicyTypeAccess, builtPolicy.Type)
	assert.Equal(t, policy.PolicyEffectAllow, builtPolicy.Effect)
	assert.Equal(t, 100, builtPolicy.Priority)
	assert.Len(t, builtPolicy.RoleIDs, 1)
	assert.Len(t, builtPolicy.TargetIDs, 1)
	assert.Len(t, builtPolicy.CredentialIDs, 1)
	assert.True(t, builtPolicy.MFARequired)
	assert.True(t, builtPolicy.RecordingRequired)
	assert.True(t, builtPolicy.SessionExtensionAllowed)
	assert.Len(t, builtPolicy.Tags, 2)
}

// TestPolicyHandler_UpdatePolicyFromRequest tests updating a policy from request
func TestPolicyHandler_UpdatePolicyFromRequest(t *testing.T) {
	service := createTestService()
	handler := NewPolicyHandler(service, zerolog.Nop())

	tenantID := uuid.New()

	testPolicy := &policy.Policy{
		ID:        uuid.New(),
		Name:      "Original Name",
		Type:      policy.PolicyTypeAccess,
		Effect:    policy.PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   true,
		Rules:     []policy.Rule{},
	}

	pt := policy.PolicyTypeSession
	pe := policy.PolicyEffectDeny
	enabled := false
	maxExtensions := 5

	updateReq := UpdatePolicyRequest{
		Name:                    "Updated Name",
		Description:             "Updated description",
		Type:                    &pt,
		Effect:                  &pe,
		Priority:                intPtr(200),
		Enabled:                 &enabled,
		SessionExtensionAllowed: boolPtr(false),
		MaxExtensions:           &maxExtensions,
		Tags:                    []string{"updated"},
	}

	handler.updatePolicyFromRequest(testPolicy, updateReq)

	assert.Equal(t, "Updated Name", testPolicy.Name)
	assert.Equal(t, "Updated description", testPolicy.Description)
	assert.Equal(t, policy.PolicyTypeSession, testPolicy.Type)
	assert.Equal(t, policy.PolicyEffectDeny, testPolicy.Effect)
	assert.Equal(t, 200, testPolicy.Priority)
	assert.False(t, testPolicy.Enabled)
	assert.False(t, testPolicy.SessionExtensionAllowed)
	assert.Equal(t, 5, *testPolicy.MaxExtensions)
	assert.Len(t, testPolicy.Tags, 1)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
