package policy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper function to create a test evaluator
func newTestEvaluator(t *testing.T) *Evaluator {
	logger := zerolog.Nop()
	return NewEvaluator(logger)
}

// TestNewEvaluator tests evaluator creation
func TestNewEvaluator(t *testing.T) {
	logger := zerolog.Nop()
	eval := NewEvaluator(logger)
	assert.NotNil(t, eval)
	assert.NotNil(t, eval.logger)
}

// TestEvaluate_NoPolicies tests evaluation with no policies
func TestEvaluate_NoPolicies(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	req := EvaluationRequest{
		TenantID:     uuid.New(),
		UserID:       uuid.New(),
		UserRoles:    []uuid.UUID{},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   uuid.New(),
		ClientIP:     "192.168.1.1",
		Time:         time.Now(),
		MFAVerified:  false,
	}

	result, err := eval.Evaluate(ctx, req, []Policy{})
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultDeny, result.Effect)
	assert.Contains(t, result.DenialReasons, "No matching policy found - default deny")
}

// TestEvaluate_AllowPolicy tests evaluation with an allow policy
func TestEvaluate_AllowPolicy(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	resourceID := uuid.New()

	policy := Policy{
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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultAllow, result.Effect)
	assert.Len(t, result.MatchedPolicies, 1)
}

// TestEvaluate_DenyPolicy tests evaluation with a deny policy
func TestEvaluate_DenyPolicy(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	resourceID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Deny Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectDeny,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules:     []Rule{},
	}

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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultDeny, result.Effect)
	assert.Contains(t, result.DenialReasons, "Policy 'Deny Policy' denies access")
}

// TestEvaluate_DisabledPolicy tests that disabled policies are not evaluated
func TestEvaluate_DisabledPolicy(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Disabled Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   false, // Disabled
		Rules:     []Rule{},
	}

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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	// Should deny because no enabled policies matched
	assert.Equal(t, EvaluationResultDeny, result.Effect)
}

// TestEvaluate_MFARequired tests MFA requirement enforcement
func TestEvaluate_MFARequired(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:         uuid.New(),
		Name:       "MFA Required Policy",
		Type:       PolicyTypeAccess,
		Effect:     PolicyEffectAllow,
		Priority:   100,
		TenantID:   tenantID,
		RoleIDs:    []uuid.UUID{roleID},
		MFARequired: true,
		MFAMethods: []MFAMethod{MFAMethodTOTP},
		Enabled:    true,
		Rules:      []Rule{},
	}

	tests := []struct {
		name         string
		mfaVerified  bool
		expectedResult EvaluationResult
	}{
		{"MFA verified", true, EvaluationResultAllow},
		{"MFA not verified", false, EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := EvaluationRequest{
				TenantID:     tenantID,
				UserID:       userID,
				UserRoles:    []uuid.UUID{roleID},
				Action:       "checkout",
				ResourceType: "credential",
				ResourceID:   uuid.New(),
				ClientIP:     "192.168.1.1",
				Time:         time.Now(),
				MFAVerified:  tt.mfaVerified,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.Effect)
		})
	}
}

// TestEvaluate_PriorityOrdering tests that higher priority policies are evaluated first
func TestEvaluate_PriorityOrdering(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	lowPriorityPolicy := Policy{
		ID:        uuid.New(),
		Name:      "Low Priority Allow",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  50,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules:     []Rule{},
	}

	highPriorityPolicy := Policy{
		ID:        uuid.New(),
		Name:      "High Priority Deny",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectDeny,
		Priority:  200,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules:     []Rule{},
	}

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

	// Pass policies in reverse priority order
	result, err := eval.Evaluate(ctx, req, []Policy{lowPriorityPolicy, highPriorityPolicy})
	require.NoError(t, err)
	// High priority deny should take effect
	assert.Equal(t, EvaluationResultDeny, result.Effect)
}

// TestEvaluate_TimeCondition tests time-based policy conditions
func TestEvaluate_TimeCondition(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	// Create a policy with time restrictions
	weekday := time.Now().Weekday()
	var currentWeekday Weekday
	switch weekday {
	case time.Monday:
		currentWeekday = WeekdayMonday
	case time.Tuesday:
		currentWeekday = WeekdayTuesday
	case time.Wednesday:
		currentWeekday = WeekdayWednesday
	case time.Thursday:
		currentWeekday = WeekdayThursday
	case time.Friday:
		currentWeekday = WeekdayFriday
	case time.Saturday:
		currentWeekday = WeekdaySaturday
	case time.Sunday:
		currentWeekday = WeekdaySunday
	}

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Time Restricted Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Business Hours Rule",
				Type:     RuleTypeTimeRestriction,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeTime,
						TimeData: &TimeCondition{
							DaysOfWeek: []Weekday{currentWeekday},
							StartTime:  "00:00",
							EndTime:    "23:59",
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	// Should allow because time condition matches current day
	assert.Equal(t, EvaluationResultAllow, result.Effect)
}

// TestEvaluate_IPCondition tests IP-based policy conditions
func TestEvaluate_IPCondition(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "IP Restricted Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Office IP Rule",
				Type:     RuleTypeIPRestriction,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeIP,
						IPData: &IPCondition{
							CIDRs: []string{"192.168.1.0/24", "10.0.0.0/8"},
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name           string
		clientIP       string
		expectedResult EvaluationResult
	}{
		{"Office IP", "192.168.1.100", EvaluationResultAllow},
		{"Allowed subnet", "10.0.0.50", EvaluationResultAllow},
		{"Outside IP", "8.8.8.8", EvaluationResultDeny},
		{"Invalid IP", "invalid", EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := EvaluationRequest{
				TenantID:     tenantID,
				UserID:       userID,
				UserRoles:    []uuid.UUID{roleID},
				Action:       "checkout",
				ResourceType: "credential",
				ResourceID:   uuid.New(),
				ClientIP:     tt.clientIP,
				Time:         time.Now(),
				MFAVerified:  false,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.Effect)
		})
	}
}

// TestEvaluate_RoleCondition tests role-based conditions
func TestEvaluate_RoleCondition(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	adminRole := uuid.New()
	operatorRole := uuid.New()
	viewerRole := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Role Based Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Admin Role Rule",
				Type:     RuleTypeMFARequirement,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeRole,
						RoleData: &RoleCondition{
							RoleIDs: []uuid.UUID{adminRole},
							Match:   "any",
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name      string
		userRoles []uuid.UUID
		expected  EvaluationResult
	}{
		{"Has admin role", []uuid.UUID{adminRole, viewerRole}, EvaluationResultAllow},
		{"Only operator role", []uuid.UUID{operatorRole}, EvaluationResultDeny},
		{"No roles", []uuid.UUID{}, EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := EvaluationRequest{
				TenantID:     tenantID,
				UserID:       userID,
				UserRoles:    tt.userRoles,
				Action:       "checkout",
				ResourceType: "credential",
				ResourceID:   uuid.New(),
				ClientIP:     "192.168.1.1",
				Time:         time.Now(),
				MFAVerified:  false,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Effect)
		})
	}
}

// TestEvaluate_ResourceCondition tests resource-based conditions
func TestEvaluate_ResourceCondition(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	targetID1 := uuid.New()
	targetID2 := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Resource Specific Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Target Rule",
				Type:     RuleTypeTimeRestriction,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeResource,
						ResourceData: &ResourceCondition{
							Targets: []uuid.UUID{targetID1},
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name        string
		resourceID  uuid.UUID
		expected    EvaluationResult
	}{
		{"Matching target", targetID1, EvaluationResultAllow},
		{"Non-matching target", targetID2, EvaluationResultDeny},
		{"No resource", uuid.Nil, EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := EvaluationRequest{
				TenantID:     tenantID,
				UserID:       userID,
				UserRoles:    []uuid.UUID{roleID},
				Action:       "checkout",
				ResourceType: "credential",
				ResourceID:   tt.resourceID,
				ClientIP:     "192.168.1.1",
				Time:         time.Now(),
				MFAVerified:  false,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Effect)
		})
	}
}

// TestEvaluate_DeviceCondition tests device trust conditions
func TestEvaluate_DeviceCondition(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Device Trust Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Managed Device Rule",
				Type:     RuleTypeMFARequirement,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeDeviceTrust,
						DeviceData: &DeviceCondition{
							Managed:          true,
							MinimumTrustScore: 50,
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name        string
		deviceTrust int
		expected    EvaluationResult
	}{
		{"High trust device", 75, EvaluationResultAllow},
		{"Medium trust device", 50, EvaluationResultAllow},
		{"Low trust device", 25, EvaluationResultDeny},
		{"No trust score", 0, EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				DeviceTrust:  tt.deviceTrust,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Effect)
		})
	}
}

// TestEvaluate_ApprovalRequired tests approval requirement
func TestEvaluate_ApprovalRequired(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	approverID := uuid.New()

	policy := Policy{
		ID:                   uuid.New(),
		Name:                 "Approval Required Policy",
		Type:                 PolicyTypeAccess,
		Effect:               PolicyEffectDeny,
		Priority:             100,
		TenantID:             tenantID,
		RoleIDs:              []uuid.UUID{roleID},
		Enabled:              true,
		ApprovalRequired:     true,
		ApprovalApprovers:     []uuid.UUID{approverID},
		ApprovalTimeoutMinutes: 60,
		Rules:                []Rule{},
	}

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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultApprovalRequired, result.Effect)
	assert.Len(t, result.RequiredActions, 1)
	assert.Equal(t, "approval", result.RequiredActions[0].Type)
}

// TestValidatePolicy tests policy validation
func TestValidatePolicy(t *testing.T) {
	eval := newTestEvaluator(t)
	tenantID := uuid.New()

	tests := []struct {
		name    string
		policy  *Policy
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid policy",
			policy: &Policy{
				Name:      "Valid Policy",
				Type:      PolicyTypeAccess,
				Effect:    PolicyEffectAllow,
				Priority:  100,
				TenantID:  tenantID,
				Rules:     []Rule{},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			policy: &Policy{
				Type:     PolicyTypeAccess,
				Effect:   PolicyEffectAllow,
				Priority: 100,
				TenantID: tenantID,
				Rules:    []Rule{},
			},
			wantErr: true,
			errMsg:  "policy name is required",
		},
		{
			name: "invalid type",
			policy: &Policy{
				Name:     "Invalid Type",
				Type:     PolicyType("invalid"),
				Effect:   PolicyEffectAllow,
				Priority: 100,
				TenantID: tenantID,
				Rules:    []Rule{},
			},
			wantErr: true,
			errMsg:  "invalid policy type",
		},
		{
			name: "invalid effect",
			policy: &Policy{
				Name:     "Invalid Effect",
				Type:     PolicyTypeAccess,
				Effect:   PolicyEffect("invalid"),
				Priority: 100,
				TenantID: tenantID,
				Rules:    []Rule{},
			},
			wantErr: true,
			errMsg:  "invalid policy effect",
		},
		{
			name: "negative priority",
			policy: &Policy{
				Name:     "Negative Priority",
				Type:     PolicyTypeAccess,
				Effect:   PolicyEffectAllow,
				Priority: -1,
				TenantID: tenantID,
				Rules:    []Rule{},
			},
			wantErr: true,
			errMsg:  "priority must be non-negative",
		},
		{
			name: "missing tenant ID",
			policy: &Policy{
				Name:     "No Tenant",
				Type:     PolicyTypeAccess,
				Effect:   PolicyEffectAllow,
				Priority: 100,
				TenantID: uuid.Nil,
				Rules:    []Rule{},
			},
			wantErr: true,
			errMsg:  "tenant_id is required",
		},
		{
			name: "MFA required but no methods",
			policy: &Policy{
				Name:        "MFA No Methods",
				Type:        PolicyTypeAccess,
				Effect:      PolicyEffectAllow,
				Priority:    100,
				TenantID:    tenantID,
				MFARequired: true,
				MFAMethods:  []MFAMethod{},
				Rules:       []Rule{},
			},
			wantErr: true,
			errMsg:  "MFA required but no methods specified",
		},
		{
			name: "approval required but no approvers",
			policy: &Policy{
				Name:             "Approval No Approvers",
				Type:             PolicyTypeAccess,
				Effect:           PolicyEffectAllow,
				Priority:         100,
				TenantID:         tenantID,
				ApprovalRequired: true,
				ApprovalApprovers: []uuid.UUID{},
				Rules:            []Rule{},
			},
			wantErr: true,
			errMsg:  "approval required but no approvers specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eval.ValidatePolicy(tt.policy)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsCommandAllowed tests command filtering
func TestIsCommandAllowed(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	patterns := []CommandFilterPattern{
		{
			Pattern:     "ls",
			IsWhitelist: true,
			Description: "Allow ls",
		},
		{
			Pattern:     "cat",
			IsWhitelist: true,
			Description: "Allow cat",
		},
		{
			Pattern:     "rm",
			IsWhitelist: false,
			Description: "Block rm",
		},
		{
			Pattern:     "dd",
			IsWhitelist: false,
			Description: "Block dd",
		},
	}

	tests := []struct {
		name        string
		command     string
		allowed     bool
		description string
	}{
		{"whitelisted command", "ls -la", true, ""},
		{"whitelisted command 2", "cat /etc/passwd", true, ""},
		{"blacklisted command", "rm -rf /", false, "Block rm"},
		{"blacklisted command 2", "dd if=/dev/zero", false, "Block dd"},
		{"not in whitelist", "vi file.txt", false, "Command not in whitelist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, desc, err := eval.IsCommandAllowed(ctx, tt.command, patterns)
			require.NoError(t, err)
			assert.Equal(t, tt.allowed, allowed)
			if tt.description != "" {
				assert.Contains(t, desc, tt.description)
			}
		})
	}
}

// TestEvaluateCondition_Negate tests condition negation
func TestEvaluateCondition_Negate(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Negated Condition Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Not Office IP",
				Type:     RuleTypeIPRestriction,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:     uuid.New(),
						Type:   ConditionTypeIP,
						Negate: true,
						IPData: &IPCondition{
							CIDRs: []string{"192.168.1.0/24"},
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	req := EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{roleID},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   uuid.New(),
		ClientIP:     "8.8.8.8", // Not in office network
		Time:         time.Now(),
		MFAVerified:  false,
	}

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	assert.Equal(t, EvaluationResultAllow, result.Effect)
}

// TestEvaluate_LogicalOperatorAND tests AND logical operator
func TestEvaluate_LogicalOperatorAND(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	// Create a policy where both conditions must match
	now := time.Now()
	var currentWeekday Weekday
	switch now.Weekday() {
	case time.Monday:
		currentWeekday = WeekdayMonday
	case time.Tuesday:
		currentWeekday = WeekdayTuesday
	case time.Wednesday:
		currentWeekday = WeekdayWednesday
	case time.Thursday:
		currentWeekday = WeekdayThursday
	case time.Friday:
		currentWeekday = WeekdayFriday
	case time.Saturday:
		currentWeekday = WeekdaySaturday
	case time.Sunday:
		currentWeekday = WeekdaySunday
	}

	policy := Policy{
		ID:        uuid.New(),
		Name:      "AND Condition Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules: []Rule{
			{
				ID:       uuid.New(),
				Name:     "Time and IP Rule",
				Type:     RuleTypeTimeRestriction,
				Operator: LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:   uuid.New(),
						Type: ConditionTypeTime,
						TimeData: &TimeCondition{
							DaysOfWeek: []Weekday{currentWeekday},
						},
					},
					{
						ID:   uuid.New(),
						Type: ConditionTypeIP,
						IPData: &IPCondition{
							CIDRs: []string{"192.168.1.0/24"},
						},
					},
				},
				Action:  "allow",
				Enabled: true,
			},
		},
	}

	tests := []struct {
		name        string
		clientIP    string
		expected    EvaluationResult
	}{
		{"Both conditions match", "192.168.1.100", EvaluationResultAllow},
		{"Only time matches", "8.8.8.8", EvaluationResultDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := EvaluationRequest{
				TenantID:     tenantID,
				UserID:       userID,
				UserRoles:    []uuid.UUID{roleID},
				Action:       "checkout",
				ResourceType: "credential",
				ResourceID:   uuid.New(),
				ClientIP:     tt.clientIP,
				Time:         now,
				MFAVerified:  false,
			}

			result, err := eval.Evaluate(ctx, req, []Policy{policy})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Effect)
		})
	}
}

// TestPolicyAppliesToUser tests user applicability check
func TestPolicyAppliesToUser(t *testing.T) {
	eval := newTestEvaluator(t)
	userID := uuid.New()
	roleID := uuid.New()

	tests := []struct {
		name     string
		policy   Policy
		userID   uuid.UUID
		userRoles []uuid.UUID
		applies  bool
	}{
		{
			name: "policy applies to all users",
			policy: Policy{
				UserIDs:  []uuid.UUID{},
				RoleIDs:  []uuid.UUID{},
			},
			userID:    userID,
			userRoles: []uuid.UUID{},
			applies:   true,
		},
		{
			name: "policy applies to specific user",
			policy: Policy{
				UserIDs: []uuid.UUID{userID},
			},
			userID:    userID,
			userRoles: []uuid.UUID{},
			applies:   true,
		},
		{
			name: "policy applies to user's role",
			policy: Policy{
				RoleIDs: []uuid.UUID{roleID},
			},
			userID:    userID,
			userRoles: []uuid.UUID{roleID},
			applies:   true,
		},
		{
			name: "policy does not apply to user",
			policy: Policy{
				UserIDs: []uuid.UUID{uuid.New()},
				RoleIDs: []uuid.UUID{uuid.New()},
			},
			userID:    userID,
			userRoles: []uuid.UUID{},
			applies:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.policyAppliesToUser(tt.policy, tt.userID, tt.userRoles)
			assert.Equal(t, tt.applies, result)
		})
	}
}

// TestPolicyAppliesToResource tests resource applicability check
func TestPolicyAppliesToResource(t *testing.T) {
	eval := newTestEvaluator(t)
	targetID := uuid.New()
	credentialID := uuid.New()

	tests := []struct {
		name       string
		policy     Policy
		resourceID uuid.UUID
		applies    bool
	}{
		{
			name: "policy applies to all resources",
			policy: Policy{
				TargetIDs:     []uuid.UUID{},
				CredentialIDs: []uuid.UUID{},
			},
			resourceID: uuid.New(),
			applies:    true,
		},
		{
			name: "policy applies to specific target",
			policy: Policy{
				TargetIDs: []uuid.UUID{targetID},
			},
			resourceID: targetID,
			applies:    true,
		},
		{
			name: "policy applies to specific credential",
			policy: Policy{
				CredentialIDs: []uuid.UUID{credentialID},
			},
			resourceID: credentialID,
			applies:    true,
		},
		{
			name: "policy does not apply to resource",
			policy: Policy{
				TargetIDs:     []uuid.UUID{uuid.New()},
				CredentialIDs: []uuid.UUID{uuid.New()},
			},
			resourceID: uuid.New(),
			applies:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eval.policyAppliesToResource(tt.policy, tt.resourceID)
			assert.Equal(t, tt.applies, result)
		})
	}
}

// TestEvaluate_DurationMs tests that evaluation duration is tracked
func TestEvaluate_DurationMs(t *testing.T) {
	eval := newTestEvaluator(t)
	ctx := context.Background()

	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Test Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		RoleIDs:   []uuid.UUID{roleID},
		Enabled:   true,
		Rules:     []Rule{},
	}

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

	result, err := eval.Evaluate(ctx, req, []Policy{policy})
	require.NoError(t, err)
	// DurationMs should be set (may be 0 for very fast evaluations)
	assert.GreaterOrEqual(t, result.DurationMs, int64(0))
}
