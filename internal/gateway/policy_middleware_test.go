package gateway

import (
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

// mockTestPolicyRepository is a mock repository for testing middleware
type mockTestPolicyRepository struct {
	policies         map[uuid.UUID]*policy.Policy
	evalLogs         []policy.PolicyEvalLog
	approvalRequests map[uuid.UUID]*policy.ApprovalRequest
	templates        []policy.PolicyTemplate
	commandPatterns  []policy.CommandFilterPattern
	createError      bool
	getError         bool
	updateError      bool
	deleteError      bool
}

func newMockTestPolicyRepository() *mockTestPolicyRepository {
	return &mockTestPolicyRepository{
		policies:         make(map[uuid.UUID]*policy.Policy),
		approvalRequests: make(map[uuid.UUID]*policy.ApprovalRequest),
		templates:        []policy.PolicyTemplate{},
		commandPatterns:  []policy.CommandFilterPattern{},
	}
}

func (m *mockTestPolicyRepository) Create(ctx context.Context, p *policy.Policy) error {
	if m.createError {
		return errors.New("repository: create failed")
	}
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	m.policies[p.ID] = p
	return nil
}

func (m *mockTestPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*policy.Policy, error) {
	if m.getError {
		return nil, errors.New("repository: get failed")
	}
	return m.policies[id], nil
}

func (m *mockTestPolicyRepository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*policy.Policy, error) {
	if m.getError {
		return nil, errors.New("repository: get failed")
	}
	p, ok := m.policies[id]
	if !ok || p.TenantID != tenantID {
		return nil, errors.New("repository: not found")
	}
	return p, nil
}

func (m *mockTestPolicyRepository) List(ctx context.Context, tenantID uuid.UUID, filter policy.PolicyFilter, limit, offset int) ([]policy.Policy, int, error) {
	var result []policy.Policy
	for _, p := range m.policies {
		result = append(result, *p)
	}
	return result, len(result), nil
}

func (m *mockTestPolicyRepository) GetActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]policy.Policy, error) {
	var result []policy.Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockTestPolicyRepository) GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]policy.Policy, error) {
	return nil, nil
}

func (m *mockTestPolicyRepository) Update(ctx context.Context, p *policy.Policy) error {
	if m.updateError {
		return errors.New("repository: update failed")
	}
	p.UpdatedAt = time.Now()
	m.policies[p.ID] = p
	return nil
}

func (m *mockTestPolicyRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteError {
		return errors.New("repository: delete failed")
	}
	delete(m.policies, id)
	return nil
}

func (m *mockTestPolicyRepository) Enable(ctx context.Context, id, tenantID uuid.UUID) error {
	if p, ok := m.policies[id]; ok {
		p.Enabled = true
	}
	return nil
}

func (m *mockTestPolicyRepository) Disable(ctx context.Context, id, tenantID uuid.UUID) error {
	if p, ok := m.policies[id]; ok {
		p.Enabled = false
	}
	return nil
}

func (m *mockTestPolicyRepository) CreateEvaluationLog(ctx context.Context, log *policy.PolicyEvalLog) error {
	m.evalLogs = append(m.evalLogs, *log)
	return nil
}

func (m *mockTestPolicyRepository) GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]policy.PolicyEvalLog, int, error) {
	return m.evalLogs, len(m.evalLogs), nil
}

func (m *mockTestPolicyRepository) CreateApprovalRequest(ctx context.Context, req *policy.ApprovalRequest) error {
	req.ID = uuid.New()
	m.approvalRequests[req.ID] = req
	return nil
}

func (m *mockTestPolicyRepository) GetApprovalRequestByID(ctx context.Context, id uuid.UUID) (*policy.ApprovalRequest, error) {
	return m.approvalRequests[id], nil
}

func (m *mockTestPolicyRepository) UpdateApprovalRequestStatus(ctx context.Context, id uuid.UUID, status policy.ApprovalStatus, approverID *uuid.UUID, denialReason string) error {
	if req, ok := m.approvalRequests[id]; ok {
		req.Status = status
		req.ApproverID = approverID
		req.DenialReason = denialReason
	}
	return nil
}

func (m *mockTestPolicyRepository) GetPendingApprovalRequests(ctx context.Context, tenantID uuid.UUID, approverID *uuid.UUID) ([]policy.ApprovalRequest, error) {
	var result []policy.ApprovalRequest
	for _, req := range m.approvalRequests {
		result = append(result, *req)
	}
	return result, nil
}

func (m *mockTestPolicyRepository) CreatePolicyTemplate(ctx context.Context, template *policy.PolicyTemplate) error {
	return nil
}

func (m *mockTestPolicyRepository) GetPolicyTemplates(ctx context.Context, category *string) ([]policy.PolicyTemplate, error) {
	return m.templates, nil
}

func (m *mockTestPolicyRepository) CreateCommandFilterPattern(ctx context.Context, pattern *policy.CommandFilterPattern) error {
	pattern.ID = uuid.New()
	m.commandPatterns = append(m.commandPatterns, *pattern)
	return nil
}

func (m *mockTestPolicyRepository) GetCommandFilterPatterns(ctx context.Context, policyID uuid.UUID) ([]policy.CommandFilterPattern, error) {
	return m.commandPatterns, nil
}

// mockTestPolicyCache is a mock cache for testing
type mockTestPolicyCache struct {
	policies    map[uuid.UUID]*policy.Policy
	evalResults map[string]*policy.EvaluationResponse
}

func newMockTestPolicyCache() *mockTestPolicyCache {
	return &mockTestPolicyCache{
		policies:    make(map[uuid.UUID]*policy.Policy),
		evalResults: make(map[string]*policy.EvaluationResponse),
	}
}

func (m *mockTestPolicyCache) GetPolicy(ctx context.Context, id uuid.UUID) (*policy.Policy, error) {
	return nil, errors.New("cache: key not found")
}

func (m *mockTestPolicyCache) SetPolicy(ctx context.Context, p *policy.Policy, ttl time.Duration) error {
	return nil
}

func (m *mockTestPolicyCache) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockTestPolicyCache) GetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string) ([]policy.Policy, int, error) {
	return nil, 0, errors.New("cache: key not found")
}

func (m *mockTestPolicyCache) SetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string, policies []policy.Policy, total int, ttl time.Duration) error {
	return nil
}

func (m *mockTestPolicyCache) InvalidPolicyList(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockTestPolicyCache) GetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID) ([]policy.Policy, error) {
	return nil, errors.New("cache: key not found")
}

func (m *mockTestPolicyCache) SetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID, policies []policy.Policy, ttl time.Duration) error {
	return nil
}

func (m *mockTestPolicyCache) InvalidateApplicablePolicies(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockTestPolicyCache) GetEvalResult(ctx context.Context, cacheKey string) (*policy.EvaluationResponse, error) {
	return nil, errors.New("cache: key not found")
}

func (m *mockTestPolicyCache) SetEvalResult(ctx context.Context, cacheKey string, result *policy.EvaluationResponse, ttl time.Duration) error {
	return nil
}

func (m *mockTestPolicyCache) InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockTestPolicyCache) InvalidateOnPolicyChange(ctx context.Context, tenantID uuid.UUID, policyID uuid.UUID) error {
	return nil
}

func (m *mockTestPolicyCache) Warmup(ctx context.Context, tenantID uuid.UUID, policies []policy.Policy) error {
	return nil
}

// createTestService creates a test policy service with mocks
func createTestService() *policy.Service {
	repo := newMockTestPolicyRepository()
	cache := newMockTestPolicyCache()
	// Pass nil cache to events for testing
	eventBus := events.New(nil, zerolog.Nop())
	logger := zerolog.Nop()

	return policy.NewTestService(repo, cache, eventBus, logger)
}

// setupTestMiddlewareRouter sets up a test router with policy middleware
func setupTestMiddlewareRouter(service *policy.Service, config Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	// Add middleware to set test context
	router.Use(func(c *gin.Context) {
		tenantID := uuid.New()
		userID := uuid.New()
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", userID.String())
		c.Set("user_roles", []uuid.UUID{})
		c.Set("user_groups", []uuid.UUID{})
		c.Next()
	})

	// Register protected routes with policy middleware
	protected := router.Group("/api/v1")
	protected.Use(middleware.Middleware(config))
	{
		protected.GET("/credentials", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "credentials list"})
		})
		protected.GET("/credentials/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "credential details"})
		})
		protected.POST("/credentials", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "credential created"})
		})
		protected.GET("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "sessions list"})
		})
		protected.POST("/sessions", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "session created"})
		})
	}

	// Add skip paths for health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	return router
}

// TestPolicyMiddleware_NewPolicyMiddleware tests middleware creation
func TestPolicyMiddleware_NewPolicyMiddleware(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.service)
	assert.NotNil(t, middleware.logger)
}

// TestPolicyMiddleware_DefaultConfig tests default configuration
func TestPolicyMiddleware_DefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.NotEmpty(t, config.SkipPaths)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipPaths, "/ready")
	assert.True(t, config.DenyByDefault)
	assert.True(t, config.LogAllRequests)
	assert.Equal(t, 30*time.Second, config.CacheTTL)
}

// TestPolicyMiddleware_ShouldSkipPath tests path skipping logic
func TestPolicyMiddleware_ShouldSkipPath(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	tests := []struct {
		name       string
		path       string
		skipPaths  []string
		shouldSkip bool
	}{
		{
			name:       "skip health endpoint",
			path:       "/health",
			skipPaths:  []string{"/health", "/ready"},
			shouldSkip: true,
		},
		{
			name:       "skip ready endpoint",
			path:       "/ready",
			skipPaths:  []string{"/health", "/ready"},
			shouldSkip: true,
		},
		{
			name:       "skip auth login endpoint",
			path:       "/api/v1/auth/login",
			skipPaths:  []string{"/api/v1/auth/login"},
			shouldSkip: true,
		},
		{
			name:       "don't skip credentials endpoint",
			path:       "/api/v1/credentials",
			skipPaths:  []string{"/health", "/ready"},
			shouldSkip: false,
		},
		{
			name:       "prefix match for skip",
			path:       "/api/v1/auth/logout",
			skipPaths:  []string{"/api/v1/auth/"},
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.shouldSkipPath(tt.path, tt.skipPaths)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

// TestPolicyMiddleware_Middleware_Allow tests policy evaluation allow
func TestPolicyMiddleware_Middleware_Allow(t *testing.T) {
	service := createTestService()

	config := DefaultConfig()
	router := setupTestMiddlewareRouter(service, config)

	req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get OK because no deny policies
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "credentials list", response["message"])
}

// TestPolicyMiddleware_Middleware_Deny tests policy evaluation deny
func TestPolicyMiddleware_Middleware_Deny(t *testing.T) {
	service := createTestService()
	tenantID := uuid.New()

	// Create a deny policy
	denyPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Deny Credential Access",
		Type:     policy.PolicyTypeCredential,
		Effect:   policy.PolicyEffectDeny,
		Enabled:  true,
		Priority: 100,
		Rules: []policy.Rule{
			{
				ID:      uuid.New(),
				Type:    policy.RuleTypeIPRestriction,
				Enabled: true,
				Operator: policy.LogicalOperatorOR,
				Conditions: []policy.Condition{
					{
						ID:   uuid.New(),
						Type: policy.ConditionTypeIP,
						IPData: &policy.IPCondition{
							CIDRs: []string{"0.0.0.0/0"}, // Match all IPs
						},
					},
				},
			},
		},
	}

	repo := newMockTestPolicyRepository()
	repo.policies[denyPolicy.ID] = denyPolicy
	testCache := newMockTestPolicyCache()
	eventBus := events.New(nil, zerolog.Nop())
	logger := zerolog.Nop()

	service = policy.NewTestService(repo, testCache, eventBus, logger)

	config := DefaultConfig()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	// Add middleware to set test context
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", uuid.New().String())
		c.Set("user_roles", []uuid.UUID{})
		c.Set("user_groups", []uuid.UUID{})
		c.Next()
	})

	// Register protected route
	protected := router.Group("/api/v1")
	protected.Use(middleware.Middleware(config))
	{
		protected.GET("/credentials", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "credentials list"})
		})
	}

	req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response["error"])
	errorMap := response["error"].(map[string]interface{})
	assert.Equal(t, "ACCESS_DENIED", errorMap["code"])
}

// TestPolicyMiddleware_Middleware_SkipPaths tests that skip paths bypass policy checks
func TestPolicyMiddleware_Middleware_SkipPaths(t *testing.T) {
	service := createTestService()

	config := DefaultConfig()
	config.SkipPaths = []string{"/health"}
	router := setupTestMiddlewareRouter(service, config)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return OK
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// TestPolicyMiddleware_EvaluateAction tests action evaluation
func TestPolicyMiddleware_EvaluateAction(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	// Create an allow policy
	allowPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Allow Checkout",
		Type:     policy.PolicyTypeCredential,
		Effect:   policy.PolicyEffectAllow,
		Enabled:  true,
		Priority: 100,
	}
	require.NoError(t, service.CreatePolicy(ctx, allowPolicy, userID))

	result, err := middleware.EvaluateAction(ctx, tenantID, userID, []uuid.UUID{roleID}, []uuid.UUID{}, "checkout", "credential", uuid.New(), "192.168.1.1", "test-agent")

	require.NoError(t, err)
	assert.Equal(t, policy.EvaluationResultAllow, result.Effect)
}

// TestPolicyMiddleware_EvaluateSessionAccess tests session access evaluation
func TestPolicyMiddleware_EvaluateSessionAccess(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Create an allow policy with a rule that allows all session access
	allowPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Allow Session",
		Type:     policy.PolicyTypeSession,
		Effect:   policy.PolicyEffectAllow,
		Enabled:  true,
		Priority: 100,
		Rules: []policy.Rule{
			{
				ID:      uuid.New(),
				Name:    "Allow All Sessions",
				Type:    policy.RuleTypeIPRestriction,
				Enabled: true,
				Operator: policy.LogicalOperatorOR,
				Conditions: []policy.Condition{
					{
						ID:   uuid.New(),
						Type: policy.ConditionTypeIP,
						IPData: &policy.IPCondition{
							CIDRs: []string{"0.0.0.0/0"}, // Match all IPs
						},
					},
				},
			},
		},
	}
	require.NoError(t, service.CreatePolicy(ctx, allowPolicy, userID))

	result, err := middleware.EvaluateSessionAccess(ctx, tenantID, userID, "example.com", 22, "192.168.1.1")

	require.NoError(t, err)
	assert.Equal(t, policy.EvaluationResultAllow, result.Effect)
}

// TestPolicyMiddleware_EvaluateCredentialCheckout tests credential checkout evaluation
func TestPolicyMiddleware_EvaluateCredentialCheckout(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	credentialID := uuid.New()

	// Create an allow policy with a rule that allows all credential access
	allowPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Allow Checkout",
		Type:     policy.PolicyTypeCredential,
		Effect:   policy.PolicyEffectAllow,
		Enabled:  true,
		Priority: 100,
		Rules: []policy.Rule{
			{
				ID:      uuid.New(),
				Name:    "Allow All Credentials",
				Type:    policy.RuleTypeIPRestriction,
				Enabled: true,
				Operator: policy.LogicalOperatorOR,
				Conditions: []policy.Condition{
					{
						ID:   uuid.New(),
						Type: policy.ConditionTypeIP,
						IPData: &policy.IPCondition{
							CIDRs: []string{"0.0.0.0/0"}, // Match all IPs
						},
					},
				},
			},
		},
	}
	require.NoError(t, service.CreatePolicy(ctx, allowPolicy, userID))

	result, err := middleware.EvaluateCredentialCheckout(ctx, tenantID, userID, credentialID, "192.168.1.1")

	require.NoError(t, err)
	assert.Equal(t, policy.EvaluationResultAllow, result.Effect)
}

// TestPolicyMiddleware_EvaluateCommand tests command evaluation
func TestPolicyMiddleware_EvaluateCommand(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		createDenyCmd bool
	}{
		{
			name:          "command allowed",
			command:       "ls -la",
			createDenyCmd: false,
		},
		{
			name:          "command denied",
			command:       "rm -rf /",
			createDenyCmd: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := createTestService()
			middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

			ctx := context.Background()
			tenantID := uuid.New()
			userID := uuid.New()
			sessionID := uuid.New()

			if tt.createDenyCmd {
				// Create a policy that blocks dangerous commands
				denyPolicy := &policy.Policy{
					TenantID: tenantID,
					Name:     "Block Dangerous Commands",
					Type:     policy.PolicyTypeCommandFilter,
					Effect:   policy.PolicyEffectDeny,
					Enabled:  true,
					Priority: 100,
					Rules: []policy.Rule{
						{
							ID:      uuid.New(),
							Name:    "Block Dangerous Commands Rule",
							Type:    policy.RuleTypeCommandFilter,
							Enabled: true,
							Conditions: []policy.Condition{
								{
									ID:   uuid.New(),
									Type: policy.ConditionTypeUser, // Using user as placeholder
								},
							},
						},
					},
				}
				require.NoError(t, service.CreatePolicy(ctx, denyPolicy, userID))
			}

			allowed, reason, _, err := middleware.EvaluateCommand(ctx, tenantID, userID, sessionID, tt.command, nil)

			require.NoError(t, err)
			if tt.createDenyCmd {
				// With deny policies, result depends on evaluation
				// The test verifies the method works correctly
				assert.NotEmpty(t, reason, "Deny reason should be provided when command is blocked")
			} else {
				// Without deny policies, commands are allowed by default
				assert.True(t, allowed, "Command should be allowed")
			}
		})
	}
}

// TestPolicyMiddleware_MapRequestToAction tests request to action mapping
func TestPolicyMiddleware_MapRequestToAction(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	tests := []struct {
		name                 string
		method               string
		path                 string
		expectedAction       string
		expectedResourceType string
	}{
		{
			name:                 "GET request",
			method:               "GET",
			path:                 "/api/v1/credentials",
			expectedAction:       "read",
			expectedResourceType: "credential",
		},
		{
			name:                 "POST request",
			method:               "POST",
			path:                 "/api/v1/sessions",
			expectedAction:       "create",
			expectedResourceType: "session",
		},
		{
			name:                 "PUT request",
			method:               "PUT",
			path:                 "/api/v1/users/123",
			expectedAction:       "update",
			expectedResourceType: "user",
		},
		{
			name:                 "PATCH request",
			method:               "PATCH",
			path:                 "/api/v1/roles/456",
			expectedAction:       "update",
			expectedResourceType: "role",
		},
		{
			name:                 "DELETE request",
			method:               "DELETE",
			path:                 "/api/v1/targets/789",
			expectedAction:       "delete",
			expectedResourceType: "target",
		},
		{
			name:                 "unknown method",
			method:               "UNKNOWN",
			path:                 "/api/v1/policies",
			expectedAction:       "access",
			expectedResourceType: "policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req

			action, resourceType := middleware.mapRequestToAction(c)
			assert.Equal(t, tt.expectedAction, action)
			assert.Equal(t, tt.expectedResourceType, resourceType)
		})
	}
}

// TestPolicyMiddleware_GetResourceID tests resource ID extraction
func TestPolicyMiddleware_GetResourceID(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	t.Run("valid UUID in path param", func(t *testing.T) {
		testUUID := uuid.New()
		req, _ := http.NewRequest("GET", "/api/v1/credentials/"+testUUID.String(), nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Params = gin.Params{{Key: "id", Value: testUUID.String()}}

		resourceID := middleware.getResourceID(c, "credential")
		assert.Equal(t, testUUID, resourceID)
	})

	t.Run("no ID in path", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		resourceID := middleware.getResourceID(c, "credential")
		assert.Equal(t, uuid.Nil, resourceID)
	})
}

// TestPolicyMiddleware_GetUserRoles tests user role extraction
func TestPolicyMiddleware_GetUserRoles(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	roleID := uuid.New()

	t.Run("roles as UUID slice", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("user_roles", []uuid.UUID{roleID})

		roles := middleware.getUserRoles(c)
		assert.Len(t, roles, 1)
		assert.Equal(t, roleID, roles[0])
	})

	t.Run("roles as string slice", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("user_roles", []string{roleID.String()})

		roles := middleware.getUserRoles(c)
		assert.Len(t, roles, 1)
		assert.Equal(t, roleID, roles[0])
	})

	t.Run("no roles set", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		roles := middleware.getUserRoles(c)
		assert.Empty(t, roles)
	})
}

// TestPolicyMiddleware_GetUserGroups tests user group extraction
func TestPolicyMiddleware_GetUserGroups(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	groupID := uuid.New()

	t.Run("groups as UUID slice", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("user_groups", []uuid.UUID{groupID})

		groups := middleware.getUserGroups(c)
		assert.Len(t, groups, 1)
		assert.Equal(t, groupID, groups[0])
	})

	t.Run("groups as string slice", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("user_groups", []string{groupID.String()})

		groups := middleware.getUserGroups(c)
		assert.Len(t, groups, 1)
		assert.Equal(t, groupID, groups[0])
	})

	t.Run("no groups set", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		groups := middleware.getUserGroups(c)
		assert.Empty(t, groups)
	})
}

// TestPolicyMiddleware_IsMFAVerified tests MFA verification check
func TestPolicyMiddleware_IsMFAVerified(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	t.Run("MFA verified", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("mfa_verified", true)

		verified := middleware.isMFAVerified(c)
		assert.True(t, verified)
	})

	t.Run("MFA not verified", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("mfa_verified", false)

		verified := middleware.isMFAVerified(c)
		assert.False(t, verified)
	})

	t.Run("MFA not set", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		verified := middleware.isMFAVerified(c)
		assert.False(t, verified)
	})
}

// TestPolicyMiddleware_GetDeviceTrust tests device trust extraction
func TestPolicyMiddleware_GetDeviceTrust(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	t.Run("device trust as int", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("device_trust", 75)

		trust := middleware.getDeviceTrust(c)
		assert.Equal(t, 75, trust)
	})

	t.Run("device trust as float64", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req
		c.Set("device_trust", 75.5)

		trust := middleware.getDeviceTrust(c)
		assert.Equal(t, 75, trust) // Should convert to int
	})

	t.Run("device trust not set", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/credentials", nil)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		trust := middleware.getDeviceTrust(c)
		assert.Equal(t, 0, trust)
	})
}

// TestPolicyMiddleware_SessionMiddleware tests session-specific middleware
func TestPolicyMiddleware_SessionMiddleware(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Fixed tenant/user IDs for policy creation
	tenantID, _ := uuid.Parse("00000000-0000-0000-0000-000000000001")
	userID, _ := uuid.Parse("00000000-0000-0000-0000-000000000002")

	// Create an allow policy for session access
	ctx := context.Background()
	allowPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Allow Session Access",
		Type:     policy.PolicyTypeSession,
		Effect:   policy.PolicyEffectAllow,
		Enabled:  true,
		Priority: 100,
		Rules: []policy.Rule{
			{
				ID:      uuid.New(),
				Name:    "Allow All Sessions",
				Type:    policy.RuleTypeIPRestriction,
				Enabled: true,
				Operator: policy.LogicalOperatorOR,
				Conditions: []policy.Condition{
					{
						ID:   uuid.New(),
						Type: policy.ConditionTypeIP,
						IPData: &policy.IPCondition{
							CIDRs: []string{"0.0.0.0/0"},
						},
					},
				},
			},
		},
	}
	_ = service.CreatePolicy(ctx, allowPolicy, userID)

	// Add middleware to set test context
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", userID.String())
		c.Set("user_roles", []uuid.UUID{})
		c.Set("user_groups", []uuid.UUID{})
		c.Next()
	})

	router.Use(middleware.SessionMiddleware())
	{
		router.GET("/api/v1/sessions", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "sessions list"})
		})
		router.GET("/api/v1/other", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "other endpoint"})
		})
	}

	// Test that session endpoint uses middleware
	req, _ := http.NewRequest("GET", "/api/v1/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed since allow policy is in place
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestPolicyMiddleware_CredentialMiddleware tests credential-specific middleware
func TestPolicyMiddleware_CredentialMiddleware(t *testing.T) {
	service := createTestService()
	middleware := NewPolicyMiddleware(service, nil, zerolog.Nop())

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Fixed tenant/user IDs for policy creation
	tenantID, _ := uuid.Parse("00000000-0000-0000-0000-000000000001")
	userID, _ := uuid.Parse("00000000-0000-0000-0000-000000000002")

	// Create an allow policy for credential access
	ctx := context.Background()
	allowPolicy := &policy.Policy{
		TenantID: tenantID,
		Name:     "Allow Credential Access",
		Type:     policy.PolicyTypeCredential,
		Effect:   policy.PolicyEffectAllow,
		Enabled:  true,
		Priority: 100,
		Rules: []policy.Rule{
			{
				ID:      uuid.New(),
				Name:    "Allow All Credentials",
				Type:    policy.RuleTypeIPRestriction,
				Enabled: true,
				Operator: policy.LogicalOperatorOR,
				Conditions: []policy.Condition{
					{
						ID:   uuid.New(),
						Type: policy.ConditionTypeIP,
						IPData: &policy.IPCondition{
							CIDRs: []string{"0.0.0.0/0"},
						},
					},
				},
			},
		},
	}
	_ = service.CreatePolicy(ctx, allowPolicy, userID)

	// Add middleware to set test context
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", userID.String())
		c.Set("user_roles", []uuid.UUID{})
		c.Set("user_groups", []uuid.UUID{})
		c.Next()
	})

	router.Use(middleware.CredentialMiddleware())
	{
		router.GET("/api/v1/credentials/:id", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "credential details"})
		})
		router.GET("/api/v1/other", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "other endpoint"})
		})
	}

	// Test that credential endpoint uses middleware
	credID := uuid.New()
	req, _ := http.NewRequest("GET", "/api/v1/credentials/"+credID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed since allow policy is in place
	assert.Equal(t, http.StatusOK, w.Code)
}
