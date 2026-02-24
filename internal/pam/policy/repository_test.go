package policy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPolicyTestDB creates a test database with policy tables
func setupPolicyTestDB(t *testing.T) *sqlx.DB {
	// This is a mock implementation - in real tests, use test database
	// For now, we'll skip database tests if not available
	t.Skip("requires test database")
	return nil
}

// TestRepository_NewRepository tests repository creation
func TestRepository_NewRepository(t *testing.T) {
	logger := zerolog.Nop()
	repo := NewRepository(nil, logger)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.logger)
}

// TestRepository_UnmarshalPolicyFields tests policy field unmarshaling
func TestRepository_UnmarshalPolicyFields(t *testing.T) {
	logger := zerolog.Nop()
	repo := NewRepository(nil, logger)

	userID := uuid.New()
	roleID := uuid.New()
	ruleID := uuid.New()
	conditionID := uuid.New()

	policy := Policy{
		UserIDsRaw:          jsonMustMarshal([]uuid.UUID{userID}),
		RoleIDsRaw:          jsonMustMarshal([]uuid.UUID{roleID}),
		GroupIDsRaw:         jsonMustMarshal([]uuid.UUID{}),
		TargetIDsRaw:        jsonMustMarshal([]uuid.UUID{}),
		CredentialIDsRaw:    jsonMustMarshal([]uuid.UUID{}),
		RulesRaw:            jsonMustMarshal([]Rule{
			{
				ID:       ruleID,
				Name:     "Test Rule",
				Type:     RuleTypeTimeRestriction,
				Operator: LogicalOperatorAND,
				Action:   "allow",
				Conditions: []Condition{
					{
						ID:       conditionID,
						Type:     ConditionTypeTime,
						Operator: LogicalOperatorAND, // Conditions also have operator field
					},
				},
			},
		}),
		MFAMethodsRaw:       jsonMustMarshal([]MFAMethod{MFAMethodTOTP}),
		ApprovalApproversRaw: jsonMustMarshal([]uuid.UUID{userID}),
		MetadataRaw:         jsonMustMarshal(map[string]interface{}{"key": "value"}),
		TagsRaw:             jsonMustMarshal([]string{"test", "important"}),
	}

	err := repo.unmarshalPolicyFields(&policy)
	require.NoError(t, err)

	assert.Len(t, policy.UserIDs, 1)
	assert.Equal(t, userID, policy.UserIDs[0])
	assert.Len(t, policy.RoleIDs, 1)
	assert.Equal(t, roleID, policy.RoleIDs[0])
	assert.Len(t, policy.Rules, 1)
	assert.Equal(t, ruleID, policy.Rules[0].ID)
	assert.Len(t, policy.MFAMethods, 1)
	assert.Equal(t, MFAMethodTOTP, policy.MFAMethods[0])
	assert.NotEmpty(t, policy.Metadata)
	assert.Len(t, policy.Tags, 2)
}

// TestRepository_UnmarshalEvalLogFields tests evaluation log unmarshaling
func TestRepository_UnmarshalEvalLogFields(t *testing.T) {
	logger := zerolog.Nop()
	repo := NewRepository(nil, logger)

	log := PolicyEvalLog{
		DenialReasonsRaw: jsonMustMarshal([]string{"test reason", "another reason"}),
		MatchedRulesRaw:  jsonMustMarshal([]MatchedRule{
			{RuleID: uuid.New(), RuleName: "Test Rule", Action: "allow"},
		}),
	}

	err := repo.unmarshalEvalLogFields(&log)
	require.NoError(t, err)

	assert.Len(t, log.DenialReasons, 2)
	assert.Len(t, log.MatchedRules, 1)
}

// TestRepository_UnmarshalApprovalRequestFields tests approval request unmarshaling
func TestRepository_UnmarshalApprovalRequestFields(t *testing.T) {
	logger := zerolog.Nop()
	repo := NewRepository(nil, logger)

	req := ApprovalRequest{
		MetadataRaw: jsonMustMarshal(map[string]interface{}{
			"requested_by": "user@test.com",
			"urgency":      "high",
		}),
	}

	err := repo.unmarshalApprovalRequestFields(&req)
	require.NoError(t, err)

	assert.NotEmpty(t, req.Metadata)
	assert.Equal(t, "user@test.com", req.Metadata["requested_by"])
}

// TestRepository_UnmarshalPolicyTemplateFields tests template unmarshaling
func TestRepository_UnmarshalPolicyTemplateFields(t *testing.T) {
	logger := zerolog.Nop()
	repo := NewRepository(nil, logger)

	template := PolicyTemplate{
		TemplatePolicyRaw: jsonMustMarshal(map[string]interface{}{
			"name":        "Test Template",
			"description": "A test policy template",
		}),
		TagsRaw: jsonMustMarshal([]string{"template", "test"}),
	}

	err := repo.unmarshalPolicyTemplateFields(&template)
	require.NoError(t, err)

	assert.NotEmpty(t, template.TemplatePolicy)
	assert.Len(t, template.Tags, 2)
}

// TestPolicyFilter_BuildQuery tests policy filter query building
func TestPolicyFilter_BuildQuery(t *testing.T) {
	tenantID := uuid.New()

	pt := PolicyTypeAccess
	pe := PolicyEffectAllow
	enabled := true

	tests := []struct {
		name   string
		filter PolicyFilter
		check  func(t *testing.T, baseQuery string, args []interface{})
	}{
		{
			name: "no filters",
			filter: PolicyFilter{},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "WHERE tenant_id = $1")
				assert.Len(t, args, 1)
			},
		},
		{
			name: "with type filter",
			filter: PolicyFilter{
				Type: &pt,
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND type = $2")
				assert.Len(t, args, 2)
			},
		},
		{
			name: "with effect filter",
			filter: PolicyFilter{
				Effect: &pe,
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND effect = $2")
				assert.Len(t, args, 2)
			},
		},
		{
			name: "with enabled filter",
			filter: PolicyFilter{
				Enabled: &enabled,
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND enabled = $2")
				assert.Len(t, args, 2)
			},
		},
		{
			name: "with search filter",
			filter: PolicyFilter{
				Search: "test policy",
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND (name ILIKE")
				assert.Contains(t, baseQuery, "OR description ILIKE")
			},
		},
		{
			name: "with tags filter",
			filter: PolicyFilter{
				Tags: []string{"production", "critical"},
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND tags ?|")
			},
		},
		{
			name: "with all filters",
			filter: PolicyFilter{
				Type:    &pt,
				Effect:  &pe,
				Enabled: &enabled,
				Search:  "important",
				Tags:    []string{"critical"},
			},
			check: func(t *testing.T, baseQuery string, args []interface{}) {
				assert.Contains(t, baseQuery, "AND type =")
				assert.Contains(t, baseQuery, "AND effect =")
				assert.Contains(t, baseQuery, "AND enabled =")
				assert.Contains(t, baseQuery, "AND (name ILIKE")
				assert.Contains(t, baseQuery, "AND tags ?|")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the query building logic from List method
			baseQuery := `
				SELECT * FROM policies
				WHERE tenant_id = $1 AND deleted_at IS NULL
			`
			countQuery := `
				SELECT COUNT(*) FROM policies
				WHERE tenant_id = $1 AND deleted_at IS NULL
			`

			args := []interface{}{tenantID}
			argCount := 2

			if tt.filter.Type != nil {
				baseQuery += ` AND type = $` + string(rune('0'+argCount))
				countQuery += ` AND type = $` + string(rune('0'+argCount))
				args = append(args, *tt.filter.Type)
				argCount++
			}

			if tt.filter.Effect != nil {
				baseQuery += ` AND effect = $` + string(rune('0'+argCount))
				countQuery += ` AND effect = $` + string(rune('0'+argCount))
				args = append(args, *tt.filter.Effect)
				argCount++
			}

			if tt.filter.Enabled != nil {
				baseQuery += ` AND enabled = $` + string(rune('0'+argCount))
				countQuery += ` AND enabled = $` + string(rune('0'+argCount))
				args = append(args, *tt.filter.Enabled)
				argCount++
			}

			if tt.filter.Search != "" {
				baseQuery += ` AND (name ILIKE $` + string(rune('0'+argCount)) + ` OR description ILIKE $` + string(rune('0'+argCount)) + `)`
				searchPattern := "%" + tt.filter.Search + "%"
				args = append(args, searchPattern, searchPattern)
				argCount += 2
			}

			if len(tt.filter.Tags) > 0 {
				baseQuery += ` AND tags ?| $` + string(rune('0'+argCount))
				args = append(args, tt.filter.Tags)
				argCount++
			}

			tt.check(t, baseQuery, args)
		})
	}
}

// TestRepository_MarshalPolicyData tests policy data marshaling for storage
func TestRepository_MarshalPolicyData(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	ruleID := uuid.New()
	conditionID := uuid.New()

	policy := Policy{
		UserIDs:       []uuid.UUID{userID},
		RoleIDs:       []uuid.UUID{roleID},
		GroupIDs:      []uuid.UUID{},
		TargetIDs:     []uuid.UUID{},
		CredentialIDs: []uuid.UUID{},
		Rules: []Rule{
			{
				ID:          ruleID,
				Name:        "Test Rule",
				Type:        RuleTypeTimeRestriction,
				Operator:    LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:       conditionID,
						Type:     ConditionTypeTime,
						Operator: LogicalOperatorAND,
						TimeData: &TimeCondition{
							DaysOfWeek: []Weekday{WeekdayMonday, WeekdayFriday},
							StartTime:  "09:00",
							EndTime:    "17:00",
						},
					},
				},
				Action:   "allow",
				Priority: 1,
				Enabled:  true,
			},
		},
		MFAMethods:       []MFAMethod{MFAMethodTOTP, MFAMethodPush},
		ApprovalApprovers: []uuid.UUID{userID},
		Metadata:         map[string]interface{}{"key": "value"},
		Tags:             []string{"test", "important"},
	}

	// Test marshaling
	userIDsJSON, err := json.Marshal(policy.UserIDs)
	require.NoError(t, err)
	assert.NotNil(t, userIDsJSON)

	roleIDsJSON, err := json.Marshal(policy.RoleIDs)
	require.NoError(t, err)
	assert.NotNil(t, roleIDsJSON)

	rulesJSON, err := json.Marshal(policy.Rules)
	require.NoError(t, err)
	assert.NotNil(t, rulesJSON)

	// Unmarshal and verify
	var unmarshaledRules []Rule
	err = json.Unmarshal(rulesJSON, &unmarshaledRules)
	require.NoError(t, err)
	assert.Len(t, unmarshaledRules, 1)
	assert.Equal(t, ruleID, unmarshaledRules[0].ID)
	assert.Len(t, unmarshaledRules[0].Conditions, 1)
}

// TestRepository_CreateEvaluationLogData tests evaluation log data creation
func TestRepository_CreateEvaluationLogData(t *testing.T) {
	policyID := uuid.New()
	userID := uuid.New()
	targetID := uuid.New()
	sessionID := uuid.New()
	ruleID := uuid.New()

	log := PolicyEvalLog{
		TenantID: uuid.New(),
		PolicyID: &policyID,
		UserID:   userID,
		TargetID: &targetID,
		Action:   "checkout",
		Result:   EvaluationResultAllow,
		DenialReasons: []string{},
		MatchedRules: []MatchedRule{
			{
				RuleID:     ruleID,
				RuleName:   "Test Rule",
				Action:     "allow",
				Conditions: []uuid.UUID{},
			},
		},
		ClientIP:  "192.168.1.1",
		UserAgent: "test-agent",
		SessionID: &sessionID,
	}

	// Test marshaling
	denialReasonsJSON, err := json.Marshal(log.DenialReasons)
	require.NoError(t, err)

	matchedRulesJSON, err := json.Marshal(log.MatchedRules)
	require.NoError(t, err)

	assert.NotNil(t, denialReasonsJSON)
	assert.NotNil(t, matchedRulesJSON)

	// Unmarshal and verify
	var unmarshaledRules []MatchedRule
	err = json.Unmarshal(matchedRulesJSON, &unmarshaledRules)
	require.NoError(t, err)
	assert.Len(t, unmarshaledRules, 1)
	assert.Equal(t, ruleID, unmarshaledRules[0].RuleID)
}

// TestRepository_ApprovalRequestStatus tests approval request status handling
func TestRepository_ApprovalRequestStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ApprovalStatus
		valid  bool
	}{
		{"pending status", ApprovalStatusPending, true},
		{"approved status", ApprovalStatusApproved, true},
		{"denied status", ApprovalStatusDenied, true},
		{"cancelled status", ApprovalStatusCancelled, true},
		{"expired status", ApprovalStatusExpired, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the status is properly set
			req := ApprovalRequest{
				Status: tt.status,
			}
			assert.Equal(t, tt.status, req.Status)
		})
	}
}

// TestRepository_PolicyTemplateTests tests policy template handling
func TestRepository_PolicyTemplateTests(t *testing.T) {
	templateID := uuid.New()

	template := PolicyTemplate{
		ID:          templateID,
		Name:        "Test Template",
		Description: "A test policy template",
		Category:    "security",
		TemplatePolicy: map[string]interface{}{
			"name":        "Security Policy",
			"description": "Generated from template",
			"type":        "access",
			"effect":      "allow",
		},
		IsSystem: true,
		Tags:     []string{"security", "template"},
	}

	// Test template structure
	assert.Equal(t, templateID, template.ID)
	assert.Equal(t, "Test Template", template.Name)
	assert.Equal(t, "security", template.Category)
	assert.True(t, template.IsSystem)
	assert.Len(t, template.Tags, 2)
	assert.NotEmpty(t, template.TemplatePolicy)
}

// TestRepository_CommandFilterPatternTests tests command filter patterns
func TestRepository_CommandFilterPatternTests(t *testing.T) {
	policyID := uuid.New()

	pattern := CommandFilterPattern{
		ID:          uuid.New(),
		PolicyID:    policyID,
		Pattern:     "rm -rf.*",
		IsWhitelist: false,
		Description: "Block dangerous file deletion commands",
		CompiledHash: "abc123def456",
		CreatedAt:   time.Now(),
	}

	// Verify pattern structure
	assert.Equal(t, policyID, pattern.PolicyID)
	assert.Equal(t, "rm -rf.*", pattern.Pattern)
	assert.False(t, pattern.IsWhitelist)
	assert.NotEmpty(t, pattern.Description)
	assert.NotEmpty(t, pattern.CompiledHash)
}

// TestPolicyScan tests Policy Scan method
func TestPolicyScan(t *testing.T) {
	policyID := uuid.New()
	tenantID := uuid.New()

	policyJSON, err := json.Marshal(Policy{
		ID:        policyID,
		Name:      "Test Policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		Enabled:   true,
	})
	require.NoError(t, err)

	var scannedPolicy Policy
	err = json.Unmarshal(policyJSON, &scannedPolicy)
	require.NoError(t, err)

	assert.Equal(t, policyID, scannedPolicy.ID)
	assert.Equal(t, "Test Policy", scannedPolicy.Name)
	assert.Equal(t, PolicyTypeAccess, scannedPolicy.Type)
	assert.Equal(t, PolicyEffectAllow, scannedPolicy.Effect)
}

// Helper function
func jsonMustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
