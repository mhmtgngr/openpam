package policy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/events"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define test errors
var (
	errTestCacheNotFound = errors.New("cache: key not found")
	errTestCacheSetFailed = errors.New("cache: set failed")
	errTestCacheDeleteFailed = errors.New("cache: delete failed")
)

// mockPolicyCache is a mock cache implementation for testing
type mockPolicyCache struct {
	policies      map[uuid.UUID]*Policy
	policyLists   map[string][]Policy
	applicable    map[string][]Policy
	evalResults   map[string]*EvaluationResponse
	getErrors     map[uuid.UUID]bool
	setErrors     map[uuid.UUID]bool
	deleteErrors  map[uuid.UUID]bool
	invalidateCalled bool
	warmupCalled     bool
}

func newMockPolicyCache() *mockPolicyCache {
	return &mockPolicyCache{
		policies:    make(map[uuid.UUID]*Policy),
		policyLists: make(map[string][]Policy),
		applicable:  make(map[string][]Policy),
		evalResults: make(map[string]*EvaluationResponse),
		getErrors:   make(map[uuid.UUID]bool),
		setErrors:   make(map[uuid.UUID]bool),
		deleteErrors: make(map[uuid.UUID]bool),
	}
}

func (m *mockPolicyCache) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	if m.getErrors[id] {
		return nil, errTestCacheNotFound
	}
	policy, ok := m.policies[id]
	if !ok {
		return nil, errTestCacheNotFound
	}
	return policy, nil
}

func (m *mockPolicyCache) SetPolicy(ctx context.Context, policy *Policy, ttl time.Duration) error {
	if m.setErrors[policy.ID] {
		return errTestCacheSetFailed
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyCache) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if m.deleteErrors[id] {
		return errTestCacheDeleteFailed
	}
	delete(m.policies, id)
	return nil
}

func (m *mockPolicyCache) GetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string) ([]Policy, int, error) {
	key := tenantID.String() + ":" + filterKey
	policies, ok := m.policyLists[key]
	if !ok {
		return nil, 0, errTestCacheNotFound
	}
	return policies, len(policies), nil
}

func (m *mockPolicyCache) SetPolicyList(ctx context.Context, tenantID uuid.UUID, filterKey string, policies []Policy, total int, ttl time.Duration) error {
	key := tenantID.String() + ":" + filterKey
	m.policyLists[key] = policies
	return nil
}

func (m *mockPolicyCache) InvalidPolicyList(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) GetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID) ([]Policy, error) {
	key := tenantID.String() + ":" + userID.String() + ":" + resourceID.String()
	policies, ok := m.applicable[key]
	if !ok {
		return nil, errTestCacheNotFound
	}
	return policies, nil
}

func (m *mockPolicyCache) SetApplicablePolicies(ctx context.Context, tenantID, userID, resourceID uuid.UUID, policies []Policy, ttl time.Duration) error {
	key := tenantID.String() + ":" + userID.String() + ":" + resourceID.String()
	m.applicable[key] = policies
	return nil
}

func (m *mockPolicyCache) InvalidateApplicablePolicies(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func (m *mockPolicyCache) GetEvalResult(ctx context.Context, cacheKey string) (*EvaluationResponse, error) {
	result, ok := m.evalResults[cacheKey]
	if !ok {
		return nil, errTestCacheNotFound
	}
	return result, nil
}

func (m *mockPolicyCache) SetEvalResult(ctx context.Context, cacheKey string, result *EvaluationResponse, ttl time.Duration) error {
	m.evalResults[cacheKey] = result
	return nil
}

func (m *mockPolicyCache) InvalidateTenant(ctx context.Context, tenantID uuid.UUID) error {
	m.invalidateCalled = true
	return nil
}

func (m *mockPolicyCache) InvalidateOnPolicyChange(ctx context.Context, tenantID uuid.UUID, policyID uuid.UUID) error {
	m.invalidateCalled = true
	return nil
}

func (m *mockPolicyCache) Warmup(ctx context.Context, tenantID uuid.UUID, policies []Policy) error {
	m.warmupCalled = true
	return nil
}

// mockPolicyRepository is a mock repository for testing
type mockPolicyRepository struct {
	policies         map[uuid.UUID]*Policy
	evalLogs         []PolicyEvalLog
	approvalRequests map[uuid.UUID]*ApprovalRequest
	templates        []PolicyTemplate
	commands         map[uuid.UUID][]CommandFilterPattern
	createError      bool
	getError         bool
	updateError      bool
	deleteError      bool
	enableError      bool
	disableError     bool
}

func newMockPolicyRepository() *mockPolicyRepository {
	return &mockPolicyRepository{
		policies:         make(map[uuid.UUID]*Policy),
		approvalRequests: make(map[uuid.UUID]*ApprovalRequest),
		commands:         make(map[uuid.UUID][]CommandFilterPattern),
	}
}

func (m *mockPolicyRepository) Create(ctx context.Context, policy *Policy) error {
	if m.createError {
		return errTestCacheSetFailed
	}
	policy.ID = uuid.New()
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = policy.CreatedAt
	policy.Version = 1
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*Policy, error) {
	if m.getError {
		return nil, errTestCacheNotFound
	}
	return m.policies[id], nil
}

func (m *mockPolicyRepository) GetByIDForTenant(ctx context.Context, id, tenantID uuid.UUID) (*Policy, error) {
	if m.getError {
		return nil, errTestCacheNotFound
	}
	return m.policies[id], nil
}

func (m *mockPolicyRepository) List(ctx context.Context, tenantID uuid.UUID, filter PolicyFilter, limit, offset int) ([]Policy, int, error) {
	var policies []Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID {
			policies = append(policies, *p)
		}
	}
	return policies, len(policies), nil
}

func (m *mockPolicyRepository) GetActiveForTenant(ctx context.Context, tenantID uuid.UUID) ([]Policy, error) {
	var policies []Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			policies = append(policies, *p)
		}
	}
	return policies, nil
}

func (m *mockPolicyRepository) GetApplicablePolicies(ctx context.Context, tenantID, userID uuid.UUID, userRoles []uuid.UUID, resourceID uuid.UUID) ([]Policy, error) {
	var policies []Policy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			policies = append(policies, *p)
		}
	}
	return policies, nil
}

func (m *mockPolicyRepository) Update(ctx context.Context, policy *Policy) error {
	if m.updateError {
		return errTestCacheSetFailed
	}
	m.policies[policy.ID] = policy
	return nil
}

func (m *mockPolicyRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.deleteError {
		return errTestCacheDeleteFailed
	}
	delete(m.policies, id)
	return nil
}

func (m *mockPolicyRepository) Enable(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.enableError {
		return errTestCacheSetFailed
	}
	if p, ok := m.policies[id]; ok {
		p.Enabled = true
	}
	return nil
}

func (m *mockPolicyRepository) Disable(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.disableError {
		return errTestCacheSetFailed
	}
	if p, ok := m.policies[id]; ok {
		p.Enabled = false
	}
	return nil
}

func (m *mockPolicyRepository) CreateEvaluationLog(ctx context.Context, log *PolicyEvalLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	m.evalLogs = append(m.evalLogs, *log)
	return nil
}

func (m *mockPolicyRepository) GetEvaluationLogs(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, limit, offset int) ([]PolicyEvalLog, int, error) {
	return m.evalLogs, len(m.evalLogs), nil
}

func (m *mockPolicyRepository) CreateApprovalRequest(ctx context.Context, req *ApprovalRequest) error {
	req.ID = uuid.New()
	req.CreatedAt = time.Now()
	req.UpdatedAt = req.CreatedAt
	req.Status = ApprovalStatusPending
	m.approvalRequests[req.ID] = req
	return nil
}

func (m *mockPolicyRepository) GetApprovalRequestByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error) {
	return m.approvalRequests[id], nil
}

func (m *mockPolicyRepository) UpdateApprovalRequestStatus(ctx context.Context, id uuid.UUID, status ApprovalStatus, approverID *uuid.UUID, denialReason string) error {
	if req, ok := m.approvalRequests[id]; ok {
		req.Status = status
		req.ApproverID = approverID
		if status == ApprovalStatusApproved {
			now := time.Now()
			req.ApprovedAt = &now
		}
		req.DenialReason = denialReason
	}
	return nil
}

func (m *mockPolicyRepository) GetPendingApprovalRequests(ctx context.Context, tenantID uuid.UUID, approverID *uuid.UUID) ([]ApprovalRequest, error) {
	var requests []ApprovalRequest
	for _, req := range m.approvalRequests {
		if req.TenantID == tenantID && req.Status == ApprovalStatusPending {
			requests = append(requests, *req)
		}
	}
	return requests, nil
}

func (m *mockPolicyRepository) CreatePolicyTemplate(ctx context.Context, template *PolicyTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = template.CreatedAt
	m.templates = append(m.templates, *template)
	return nil
}

func (m *mockPolicyRepository) GetPolicyTemplates(ctx context.Context, category *string) ([]PolicyTemplate, error) {
	return m.templates, nil
}

func (m *mockPolicyRepository) CreateCommandFilterPattern(ctx context.Context, pattern *CommandFilterPattern) error {
	pattern.ID = uuid.New()
	pattern.CreatedAt = time.Now()
	if m.commands[pattern.PolicyID] == nil {
		m.commands[pattern.PolicyID] = []CommandFilterPattern{*pattern}
	} else {
		m.commands[pattern.PolicyID] = append(m.commands[pattern.PolicyID], *pattern)
	}
	return nil
}

func (m *mockPolicyRepository) GetCommandFilterPatterns(ctx context.Context, policyID uuid.UUID) ([]CommandFilterPattern, error) {
	return m.commands[policyID], nil
}

// TestNewService tests service creation
func TestNewService(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := NewService(repo, &cache.Cache{}, eventBus, logger)
	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.cache)
	assert.NotNil(t, service.evaluator)
	assert.NotNil(t, service.eventBus)
}

// TestService_CreatePolicy tests policy creation
func TestService_CreatePolicy(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	policy := &Policy{
		Name:      "Test Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   true,
		Rules:     []Rule{},
	}

	err := service.CreatePolicy(ctx, policy, userID)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, policy.ID)
	assert.Equal(t, userID, policy.CreatedBy)
	assert.Equal(t, userID, policy.UpdatedBy)
	assert.True(t, mockCache.invalidateCalled)
}

// TestService_GetPolicy tests policy retrieval with cache
func TestService_GetPolicy(t *testing.T) {
	logger := zerolog.Nop()
	mockRepo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     mockRepo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	policyID := uuid.New()

	// Create a policy in the mock repo
	policy := &Policy{
		ID:        policyID,
		Name:      "Test Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   true,
		Rules:     []Rule{},
	}
	mockRepo.policies[policyID] = policy

	// First call should hit cache miss, then store in cache
	result, err := service.GetPolicy(ctx, policyID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, policyID, result.ID)
	assert.Equal(t, "Test Policy", result.Name)

	// Verify policy was cached
	cachedPolicy := mockCache.policies[policyID]
	assert.NotNil(t, cachedPolicy)
	assert.Equal(t, policyID, cachedPolicy.ID)
}

// TestService_ListPolicies tests policy listing
func TestService_ListPolicies(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()

	// Add some policies to the mock repo
	for i := 0; i < 3; i++ {
		policy := &Policy{
			ID:        uuid.New(),
			Name:      "Test Policy",
			Type:      PolicyTypeAccess,
			Effect:    PolicyEffectAllow,
			Priority:  100 + i,
			TenantID:  tenantID,
			Enabled:   true,
			Rules:     []Rule{},
		}
		repo.policies[policy.ID] = policy
	}

	filter := PolicyFilter{}
	policies, total, err := service.ListPolicies(ctx, tenantID, filter, 10, 0)
	require.NoError(t, err)
	assert.Len(t, policies, 3)
	assert.Equal(t, 3, total)
}

// TestService_Evaluate tests policy evaluation
func TestService_Evaluate(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	// Create an allow policy
	policy := &Policy{
		ID:        uuid.New(),
		Name:      "Allow Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules:     []Rule{},
	}
	repo.policies[policy.ID] = policy

	req := EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{roleID},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   uuid.New(),
		ClientIP:     "192.168.1.1",
		Time:         time.Now(),
		MFAVerified:  false,
	}

	result, err := service.Evaluate(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultAllow, result.Effect)
	assert.Len(t, repo.evalLogs, 1)
}

// TestService_EvaluationRequestCaching tests that evaluation results are cached
func TestService_EvaluationRequestCaching(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	resourceID := uuid.New()

	// Create a cached evaluation result
	cachedResult := &EvaluationResponse{
		Effect:          EvaluationResultAllow,
		MatchedPolicies: []MatchedPolicy{},
		DenialReasons:   []string{},
		RequiredActions: []RequiredAction{},
		EvaluatedAt:     time.Now(),
		DurationMs:      10,
	}
	cacheKey := GenerateEvalCacheKey(EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{roleID},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   resourceID,
	})
	mockCache.evalResults[cacheKey] = cachedResult

	req := EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{roleID},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   resourceID,
		ClientIP:     "192.168.1.1",
		Time:         time.Now(),
		MFAVerified:  false,
	}

	result, err := service.Evaluate(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultAllow, result.Effect)
	// Should not create eval log when using cached result
	assert.Len(t, repo.evalLogs, 0)
}

// TestService_ApprovalWorkflow tests approval request workflow
func TestService_ApprovalWorkflow(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	requesterID := uuid.New()
	approverID := uuid.New()
	targetID := uuid.New()

	t.Run("RequestApproval", func(t *testing.T) {
		req := &ApprovalRequest{
			RequesterID: requesterID,
			TenantID:    tenantID,
			TargetType:  "credential",
			TargetID:    targetID,
			Reason:      "Need access for maintenance",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}

		err := service.RequestApproval(ctx, req)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, req.ID)
		assert.Equal(t, ApprovalStatusPending, req.Status)
	})

	t.Run("ApproveRequest", func(t *testing.T) {
		// Find the created request
		var requestID uuid.UUID
		for id, req := range repo.approvalRequests {
			if req.Status == ApprovalStatusPending {
				requestID = id
				break
			}
		}

		err := service.ApproveRequest(ctx, requestID, approverID)
		require.NoError(t, err)

		approvedReq := repo.approvalRequests[requestID]
		assert.Equal(t, ApprovalStatusApproved, approvedReq.Status)
		assert.Equal(t, &approverID, approvedReq.ApproverID)
		assert.NotNil(t, approvedReq.ApprovedAt)
	})

	t.Run("DenyRequest", func(t *testing.T) {
		req := &ApprovalRequest{
			RequesterID: requesterID,
			TenantID:    tenantID,
			TargetType:  "credential",
			TargetID:    targetID,
			Reason:      "Another request",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}

		err := service.RequestApproval(ctx, req)
		require.NoError(t, err)

		err = service.DenyRequest(ctx, req.ID, approverID, "Not approved")
		require.NoError(t, err)

		deniedReq := repo.approvalRequests[req.ID]
		assert.Equal(t, ApprovalStatusDenied, deniedReq.Status)
		assert.Equal(t, "Not approved", deniedReq.DenialReason)
	})
}

// TestService_EnableDisablePolicy tests enabling and disabling policies
func TestService_EnableDisablePolicy(t *testing.T) {
	logger := zerolog.Nop()
	repo := newMockPolicyRepository()
	mockCache := newMockPolicyCache()
	eventBus := events.NewWithoutConfig(&cache.Cache{}, logger)

	service := &Service{
		repo:     repo,
		cache:    mockCache,
		evaluator: NewEvaluator(logger),
		eventBus: eventBus,
		logger:   logger,
	}

	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()
	policyID := uuid.New()

	// Create a disabled policy
	policy := &Policy{
		ID:        policyID,
		Name:      "Test Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   false,
		Rules:     []Rule{},
	}
	repo.policies[policyID] = policy

	t.Run("EnablePolicy", func(t *testing.T) {
		err := service.EnablePolicy(ctx, policyID, tenantID, userID)
		require.NoError(t, err)
		assert.True(t, repo.policies[policyID].Enabled)
		assert.True(t, mockCache.invalidateCalled)
	})

	// Reset for next test
	mockCache.invalidateCalled = false

	t.Run("DisablePolicy", func(t *testing.T) {
		err := service.DisablePolicy(ctx, policyID, tenantID, userID)
		require.NoError(t, err)
		assert.False(t, repo.policies[policyID].Enabled)
		assert.True(t, mockCache.invalidateCalled)
	})
}

// TestGenerateEvalCacheKey tests cache key generation
func TestGenerateEvalCacheKey(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	resourceID := uuid.New()

	req := EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   resourceID,
	}

	cacheKey := GenerateEvalCacheKey(req)
	assert.Contains(t, cacheKey, tenantID.String())
	assert.Contains(t, cacheKey, userID.String())
	assert.Contains(t, cacheKey, "checkout")
	assert.Contains(t, cacheKey, "credential")
}
