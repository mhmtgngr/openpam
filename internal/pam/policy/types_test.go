package policy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPolicyTypeFromString tests policy type string conversion
func TestPolicyTypeFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PolicyType
		wantErr bool
	}{
		{"access lower", "access", PolicyTypeAccess, false},
		{"access upper", "ACCESS", PolicyTypeAccess, false},
		{"access mixed", "AcCeSs", PolicyTypeAccess, false},
		{"session", "session", PolicyTypeSession, false},
		{"credential", "credential", PolicyTypeCredential, false},
		{"approval", "approval", PolicyTypeApproval, false},
		{"compliance", "compliance", PolicyTypeCompliance, false},
		{"command_filter dash", "command_filter", PolicyTypeCommandFilter, false},
		{"command_filter hyphen", "command-filter", PolicyTypeCommandFilter, false},
		{"command_filter combined", "commandfilter", PolicyTypeCommandFilter, false},
		{"invalid type", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PolicyTypeFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestPolicyTypeIsValid tests policy type validation
func TestPolicyTypeIsValid(t *testing.T) {
	tests := []struct {
		name string
		pt   PolicyType
		want bool
	}{
		{"access valid", PolicyTypeAccess, true},
		{"session valid", PolicyTypeSession, true},
		{"credential valid", PolicyTypeCredential, true},
		{"approval valid", PolicyTypeApproval, true},
		{"compliance valid", PolicyTypeCompliance, true},
		{"command_filter valid", PolicyTypeCommandFilter, true},
		{"invalid type", PolicyType("invalid"), false},
		{"empty type", PolicyType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.pt.IsValid())
		})
	}
}

// TestPolicyTypeMarshalJSON tests JSON marshaling for PolicyType
func TestPolicyTypeMarshalJSON(t *testing.T) {
	pt := PolicyTypeAccess
	data, err := json.Marshal(pt)
	require.NoError(t, err)
	assert.Equal(t, `"access"`, string(data))
}

// TestPolicyTypeUnmarshalJSON tests JSON unmarshaling for PolicyType
func TestPolicyTypeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PolicyType
		wantErr bool
	}{
		{"valid access", `"access"`, PolicyTypeAccess, false},
		{"valid session", `"session"`, PolicyTypeSession, false},
		{"invalid type", `"invalid"`, "", true},
		{"not a string", `123`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got PolicyType
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestEffectFromString tests effect string conversion
func TestEffectFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    PolicyEffect
		wantErr bool
	}{
		{"allow lower", "allow", PolicyEffectAllow, false},
		{"allow upper", "ALLOW", PolicyEffectAllow, false},
		{"deny lower", "deny", PolicyEffectDeny, false},
		{"deny upper", "DENY", PolicyEffectDeny, false},
		{"invalid effect", "invalid", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EffectFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestPolicyEffectIsAllowDeny tests effect helper methods
func TestPolicyEffectIsAllowDeny(t *testing.T) {
	assert.True(t, PolicyEffectAllow.IsAllow())
	assert.False(t, PolicyEffectAllow.IsDeny())
	assert.False(t, PolicyEffectDeny.IsAllow())
	assert.True(t, PolicyEffectDeny.IsDeny())
}

// TestRuleTypeFromString tests rule type string conversion
func TestRuleTypeFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    RuleType
		wantErr bool
	}{
		{"time_restriction", "time_restriction", RuleTypeTimeRestriction, false},
		{"time", "time", RuleTypeTimeRestriction, false},
		{"ip_restriction", "ip_restriction", RuleTypeIPRestriction, false},
		{"ip", "ip", RuleTypeIPRestriction, false},
		{"mfa_requirement", "mfa_requirement", RuleTypeMFARequirement, false},
		{"mfa", "mfa", RuleTypeMFARequirement, false},
		{"command_filter", "command_filter", RuleTypeCommandFilter, false},
		{"session_recording", "session_recording", RuleTypeSessionRecording, false},
		{"recording", "recording", RuleTypeSessionRecording, false},
		{"approval_requirement", "approval_requirement", RuleTypeApprovalRequirement, false},
		{"approval", "approval", RuleTypeApprovalRequirement, false},
		{"duration_limit", "duration_limit", RuleTypeDurationLimit, false},
		{"duration", "duration", RuleTypeDurationLimit, false},
		{"concurrent_limit", "concurrent_limit", RuleTypeConcurrentLimit, false},
		{"concurrent", "concurrent", RuleTypeConcurrentLimit, false},
		{"invalid type", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RuleTypeFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestApprovalStatusFromString tests approval status string conversion
func TestApprovalStatusFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ApprovalStatus
		wantErr bool
	}{
		{"pending", "pending", ApprovalStatusPending, false},
		{"approved lower", "approved", ApprovalStatusApproved, false},
		{"approved action", "approve", ApprovalStatusApproved, false},
		{"denied lower", "denied", ApprovalStatusDenied, false},
		{"deny action", "deny", ApprovalStatusDenied, false},
		{"rejected", "rejected", ApprovalStatusDenied, false},
		{"cancelled lower", "cancelled", ApprovalStatusCancelled, false},
		{"canceled", "canceled", ApprovalStatusCancelled, false},
		{"cancel action", "cancel", ApprovalStatusCancelled, false},
		{"expired", "expired", ApprovalStatusExpired, false},
		{"invalid status", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApprovalStatusFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestApprovalStatusIsFinal tests approval status finality check
func TestApprovalStatusIsFinal(t *testing.T) {
	assert.False(t, ApprovalStatusPending.IsFinal())
	assert.True(t, ApprovalStatusApproved.IsFinal())
	assert.True(t, ApprovalStatusDenied.IsFinal())
	assert.True(t, ApprovalStatusCancelled.IsFinal())
	assert.True(t, ApprovalStatusExpired.IsFinal())
}

// TestEvaluationResultStringAndHelpers tests evaluation result helpers
func TestEvaluationResultStringAndHelpers(t *testing.T) {
	tests := []struct {
		result    EvaluationResult
		allowed   bool
		denied    bool
		approval  bool
	}{
		{EvaluationResultAllow, true, false, false},
		{EvaluationResultDeny, false, true, false},
		{EvaluationResultApprovalRequired, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.result.String(), func(t *testing.T) {
			assert.Equal(t, tt.allowed, tt.result.IsAllowed())
			assert.Equal(t, tt.denied, tt.result.IsDenied())
			assert.Equal(t, tt.approval, tt.result.RequiresApproval())
		})
	}
}

// TestEvaluationResultMarshalJSON tests JSON marshaling for EvaluationResult
func TestEvaluationResultMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		result   EvaluationResult
		expected string
	}{
		{"allow", EvaluationResultAllow, `"allow"`},
		{"deny", EvaluationResultDeny, `"deny"`},
		{"approval required", EvaluationResultApprovalRequired, `"approval_required"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

// TestWeekdayFromString tests weekday string conversion
func TestWeekdayFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Weekday
		wantErr bool
	}{
		{"monday full", "monday", WeekdayMonday, false},
		{"monday short", "mon", WeekdayMonday, false},
		{"tuesday full", "tuesday", WeekdayTuesday, false},
		{"tuesday short", "tue", WeekdayTuesday, false},
		{"wednesday full", "wednesday", WeekdayWednesday, false},
		{"wednesday short", "wed", WeekdayWednesday, false},
		{"thursday full", "thursday", WeekdayThursday, false},
		{"thursday short", "thu", WeekdayThursday, false},
		{"friday full", "friday", WeekdayFriday, false},
		{"friday short", "fri", WeekdayFriday, false},
		{"saturday full", "saturday", WeekdaySaturday, false},
		{"saturday short", "sat", WeekdaySaturday, false},
		{"sunday full", "sunday", WeekdaySunday, false},
		{"sunday short", "sun", WeekdaySunday, false},
		{"invalid day", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WeekdayFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestRecordingModeFromString tests recording mode string conversion
func TestRecordingModeFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    RecordingMode
		wantErr bool
	}{
		{"all", "all", RecordingModeAll, false},
		{"on_command underscore", "on_command", RecordingModeOnCommand, false},
		{"on_command", "oncommand", RecordingModeOnCommand, false},
		{"none", "none", RecordingModeNone, false},
		{"invalid mode", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RecordingModeFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestRecordingModeShouldRecord tests recording mode check
func TestRecordingModeShouldRecord(t *testing.T) {
	assert.True(t, RecordingModeAll.ShouldRecord())
	assert.True(t, RecordingModeOnCommand.ShouldRecord())
	assert.False(t, RecordingModeNone.ShouldRecord())
}

// TestMFAMethodFromString tests MFA method string conversion
func TestMFAMethodFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    MFAMethod
		wantErr bool
	}{
		{"totp", "totp", MFAMethodTOTP, false},
		{"push", "push", MFAMethodPush, false},
		{"sms", "sms", MFAMethodSMS, false},
		{"hardware_key", "hardware_key", MFAMethodHardwareKey, false},
		{"yubikey", "yubikey", MFAMethodHardwareKey, false},
		{"u2f", "u2f", MFAMethodHardwareKey, false},
		{"email", "email", MFAMethodEmail, false},
		{"webauthn", "webauthn", MFAMethodWebAuthn, false},
		{"fido2", "fido2", MFAMethodWebAuthn, false},
		{"invalid method", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MFAMethodFromString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestLogicalOperatorUnmarshalJSON tests logical operator JSON unmarshaling
func TestLogicalOperatorUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    LogicalOperator
		wantErr bool
	}{
		{"and lower", `"and"`, LogicalOperatorAND, false},
		{"and upper", `"AND"`, LogicalOperatorAND, false},
		{"or lower", `"or"`, LogicalOperatorOR, false},
		{"or upper", `"OR"`, LogicalOperatorOR, false},
		{"invalid operator", `"invalid"`, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got LogicalOperator
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestConditionTypeString tests condition type string representation
func TestConditionTypeString(t *testing.T) {
	tests := []struct {
		ct   ConditionType
		want string
	}{
		{ConditionTypeTime, "time"},
		{ConditionTypeIP, "ip"},
		{ConditionTypeRole, "role"},
		{ConditionTypeResource, "resource"},
		{ConditionTypeUser, "user"},
		{ConditionTypeGroup, "group"},
		{ConditionTypeMFAMethod, "mfa_method"},
		{ConditionTypeMFATrust, "mfa_trust"},
		{ConditionTypeGeo, "geo"},
		{ConditionTypeDeviceTrust, "device_trust"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.ct.String())
		})
	}
}

// TestPolicyCreation tests creating a policy with all fields
func TestPolicyCreation(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	targetID := uuid.New()
	credentialID := uuid.New()

	policy := Policy{
		ID:        uuid.New(),
		Name:      "Test Policy",
		Description: "A test policy",
		Type:      PolicyTypeAccess,
		Effect:    PolicyEffectAllow,
		Priority:  100,
		TenantID:  tenantID,
		UserIDs:   []uuid.UUID{userID},
		RoleIDs:   []uuid.UUID{roleID},
		TargetIDs: []uuid.UUID{targetID},
		CredentialIDs: []uuid.UUID{credentialID},
		Rules: []Rule{
			{
				ID:          uuid.New(),
				Name:        "Test Rule",
				Description: "A test rule",
				Type:        RuleTypeTimeRestriction,
				Operator:    LogicalOperatorAND,
				Conditions: []Condition{
					{
						ID:     uuid.New(),
						Type:   ConditionTypeTime,
						Negate: false,
					},
				},
				Action:   "allow",
				Priority: 1,
				Enabled:  true,
			},
		},
		MFARequired:     true,
		MFAMethods:      []MFAMethod{MFAMethodTOTP, MFAMethodPush},
		ApprovalRequired: true,
		ApprovalApprovers: []uuid.UUID{userID},
		ApprovalTimeoutMinutes: 60,
		RecordingRequired: true,
		RecordingMode:    RecordingModeAll,
		Enabled:          true,
		SystemPolicy:     false,
		Tags:             []string{"test", "important"},
		Version:          1,
		CreatedBy:        userID,
		UpdatedBy:        userID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	assert.Equal(t, "Test Policy", policy.Name)
	assert.Equal(t, PolicyTypeAccess, policy.Type)
	assert.Equal(t, PolicyEffectAllow, policy.Effect)
	assert.Equal(t, 100, policy.Priority)
	assert.Len(t, policy.UserIDs, 1)
	assert.Len(t, policy.Rules, 1)
	assert.Len(t, policy.MFAMethods, 2)
	assert.True(t, policy.MFARequired)
	assert.True(t, policy.ApprovalRequired)
	assert.True(t, policy.RecordingRequired)
	assert.True(t, policy.Enabled)
}

// TestEvaluationRequestCreation tests creating an evaluation request
func TestEvaluationRequestCreation(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	resourceID := uuid.New()
	sessionID := uuid.New()

	req := EvaluationRequest{
		TenantID:     tenantID,
		UserID:       userID,
		UserRoles:    []uuid.UUID{roleID},
		UserGroups:   []uuid.UUID{},
		Action:       "checkout",
		ResourceType: "credential",
		ResourceID:   resourceID,
		ClientIP:     "192.168.1.1",
		UserAgent:    "test-agent",
		SessionID:    &sessionID,
		Time:         time.Now(),
		MFAVerified:  true,
		MFAMethod:    MFAMethodTOTP,
		DeviceTrust:  75,
		Context: map[string]interface{}{
			"department": "engineering",
			"location":   "US",
		},
	}

	assert.Equal(t, tenantID, req.TenantID)
	assert.Equal(t, userID, req.UserID)
	assert.Equal(t, "checkout", req.Action)
	assert.Equal(t, "credential", req.ResourceType)
	assert.True(t, req.MFAVerified)
	assert.Equal(t, 75, req.DeviceTrust)
	assert.NotEmpty(t, req.Context)
}

// TestEvaluationResponseCreation tests creating an evaluation response
func TestEvaluationResponseCreation(t *testing.T) {
	policyID := uuid.New()

	resp := EvaluationResponse{
		Effect:        EvaluationResultAllow,
		MatchedPolicies: []MatchedPolicy{
			{
				PolicyID:   policyID,
				PolicyName: "Test Policy",
				Effect:     PolicyEffectAllow,
				Rules:      []MatchedRule{},
				Priority:   100,
			},
		},
		DenialReasons:   []string{},
		RequiredActions: []RequiredAction{
			{
				Type:        "mfa",
				Description: "MFA verification required",
				RequiredBy:  policyID,
				Timeout:     intPtr(300),
			},
		},
		EvaluatedAt: time.Now(),
		DurationMs:  15,
	}

	assert.Equal(t, EvaluationResultAllow, resp.Effect)
	assert.Len(t, resp.MatchedPolicies, 1)
	assert.Len(t, resp.RequiredActions, 1)
	assert.Equal(t, int64(15), resp.DurationMs)
}

// TestConditionDataStructures tests condition-specific data structures
func TestConditionDataStructures(t *testing.T) {
	t.Run("TimeCondition", func(t *testing.T) {
		tc := TimeCondition{
			DaysOfWeek: []Weekday{WeekdayMonday, WeekdayFriday},
			StartTime:  "09:00",
			EndTime:    "17:00",
			Timezone:   "America/New_York",
			DateRange: &DateRange{
				StartDate: time.Now(),
				EndDate:   time.Now().Add(24 * time.Hour),
			},
		}
		assert.Len(t, tc.DaysOfWeek, 2)
		assert.Equal(t, "09:00", tc.StartTime)
		assert.NotNil(t, tc.DateRange)
	})

	t.Run("IPCondition", func(t *testing.T) {
		ic := IPCondition{
			CIDRs: []string{"192.168.1.0/24", "10.0.0.0/8"},
			IPRanges: []IPRange{
				{Start: "192.168.1.1", End: "192.168.1.255"},
			},
			Countries: []string{"US", "CA"},
			Source:    "client_ip",
		}
		assert.Len(t, ic.CIDRs, 2)
		assert.Len(t, ic.IPRanges, 1)
		assert.Len(t, ic.Countries, 2)
	})

	t.Run("RoleCondition", func(t *testing.T) {
		rc := RoleCondition{
			RoleIDs: []uuid.UUID{uuid.New(), uuid.New()},
			Roles:   []string{"admin", "operator"},
			Match:   "all",
		}
		assert.Len(t, rc.RoleIDs, 2)
		assert.Len(t, rc.Roles, 2)
		assert.Equal(t, "all", rc.Match)
	})

	t.Run("ResourceCondition", func(t *testing.T) {
		resID := uuid.New()
		rc := ResourceCondition{
			ResourceIDs: []uuid.UUID{resID},
			Targets:     []uuid.UUID{resID},
			Credentials: []uuid.UUID{},
			Tags:        []string{"production", "database"},
			Type:        "ssh",
		}
		assert.Len(t, rc.ResourceIDs, 1)
		assert.Len(t, rc.Tags, 2)
		assert.Equal(t, "ssh", rc.Type)
	})

	t.Run("DeviceCondition", func(t *testing.T) {
		dc := DeviceCondition{
			Managed:          true,
			MinimumTrustScore: 75,
			AllowedPlatforms: []string{"windows", "macos", "linux"},
		}
		assert.True(t, dc.Managed)
		assert.Equal(t, 75, dc.MinimumTrustScore)
		assert.Len(t, dc.AllowedPlatforms, 3)
	})
}

// TestPolicyFilterTests tests policy filtering
func TestPolicyFilterTests(t *testing.T) {
	pt := PolicyTypeAccess
	pe := PolicyEffectAllow
	enabled := true

	filter := PolicyFilter{
		Type:    &pt,
		Effect:  &pe,
		Enabled: &enabled,
		Tags:    []string{"test", "production"},
		Search:  "database",
	}

	assert.NotNil(t, filter.Type)
	assert.Equal(t, PolicyTypeAccess, *filter.Type)
	assert.NotNil(t, filter.Effect)
	assert.Equal(t, PolicyEffectAllow, *filter.Effect)
	assert.True(t, *filter.Enabled)
	assert.Len(t, filter.Tags, 2)
	assert.Equal(t, "database", filter.Search)
}

// TestCommandFilterPattern tests command filter patterns
func TestCommandFilterPattern(t *testing.T) {
	pattern := CommandFilterPattern{
		ID:          uuid.New(),
		PolicyID:    uuid.New(),
		Pattern:     "rm -rf.*",
		IsWhitelist: false,
		Description: "Block dangerous file deletion",
		CompiledHash: "abc123",
		CreatedAt:   time.Now(),
	}

	assert.Equal(t, "rm -rf.*", pattern.Pattern)
	assert.False(t, pattern.IsWhitelist)
	assert.NotEmpty(t, pattern.Description)
}
