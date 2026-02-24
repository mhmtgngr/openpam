package policy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Evaluator evaluates policies against requests
type Evaluator struct {
	logger zerolog.Logger
}

// NewEvaluator creates a new policy evaluator
func NewEvaluator(logger zerolog.Logger) *Evaluator {
	return &Evaluator{
		logger: logger,
	}
}

// Evaluate evaluates a request against applicable policies and returns the decision
func (e *Evaluator) Evaluate(ctx context.Context, req EvaluationRequest, policies []Policy) (*EvaluationResponse, error) {
	startTime := time.Now()

	resp := &EvaluationResponse{
		Effect:        EvaluationResultAllow, // Default to allow
		MatchedPolicies: []MatchedPolicy{},
		DenialReasons:  []string{},
		RequiredActions: []RequiredAction{},
		EvaluatedAt:    time.Now(),
	}

	// Sort policies by priority (higher priority first)
	sortedPolicies := make([]Policy, len(policies))
	copy(sortedPolicies, policies)
	e.sortPoliciesByPriority(sortedPolicies)

	// Evaluate each policy
	for _, policy := range sortedPolicies {
		if !policy.Enabled {
			continue
		}

		matched, matchedRules, err := e.evaluatePolicy(ctx, req, policy)
		if err != nil {
			e.logger.Error().Err(err).
				Str("policy_id", policy.ID.String()).
				Msg("Failed to evaluate policy")
			continue
		}

		if matched {
			matchedPolicy := MatchedPolicy{
				PolicyID:   policy.ID,
				PolicyName: policy.Name,
				Effect:     policy.Effect,
				Rules:      matchedRules,
				Priority:   policy.Priority,
			}
			resp.MatchedPolicies = append(resp.MatchedPolicies, matchedPolicy)

			// Process the effect
			if policy.Effect == PolicyEffectDeny {
				resp.Effect = EvaluationResultDeny
				resp.DenialReasons = append(resp.DenialReasons,
					fmt.Sprintf("Policy '%s' denies access", policy.Name))

				// Check for approval requirement
				if policy.ApprovalRequired {
					resp.Effect = EvaluationResultApprovalRequired
					resp.RequiredActions = append(resp.RequiredActions, RequiredAction{
						Type:        "approval",
						Description: "Manager approval required",
						RequiredBy:  policy.ID,
						Timeout:     intPtr(policy.ApprovalTimeoutMinutes * 60),
					})
				}
				break // Deny takes precedence
			} else if policy.Effect == PolicyEffectAllow && resp.Effect != EvaluationResultDeny {
				resp.Effect = EvaluationResultAllow

				// Check for additional requirements
				if policy.MFARequired && !req.MFAVerified {
					resp.Effect = EvaluationResultDeny
					resp.DenialReasons = append(resp.DenialReasons,
						fmt.Sprintf("Policy '%s' requires MFA verification", policy.Name))
				}
			}
		}
	}

	// If no policies matched, apply default behavior (deny)
	if len(resp.MatchedPolicies) == 0 {
		resp.Effect = EvaluationResultDeny
		resp.DenialReasons = append(resp.DenialReasons, "No matching policy found - default deny")
	}

	resp.DurationMs = time.Since(startTime).Milliseconds()

	return resp, nil
}

// evaluatePolicy evaluates if a policy matches the request
func (e *Evaluator) evaluatePolicy(ctx context.Context, req EvaluationRequest, policy Policy) (bool, []MatchedRule, error) {
	// Check if policy applies to the user
	if !e.policyAppliesToUser(policy, req.UserID, req.UserRoles) {
		return false, nil, nil
	}

	// Check if policy applies to the resource
	if !e.policyAppliesToResource(policy, req.ResourceID) {
		return false, nil, nil
	}

	// Evaluate rules
	var allMatchedRules []MatchedRule
	for _, rule := range policy.Rules {
		if !rule.Enabled {
			continue
		}

		matched, matchedConditions, err := e.evaluateRule(ctx, req, rule)
		if err != nil {
			return false, nil, fmt.Errorf("evaluateRule: %w", err)
		}

		if matched {
			matchedRule := MatchedRule{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				Action:     rule.Action,
				Conditions: matchedConditions,
			}
			allMatchedRules = append(allMatchedRules, matchedRule)
		}
	}

	// If there are rules, at least one must match
	if len(policy.Rules) > 0 {
		return len(allMatchedRules) > 0, allMatchedRules, nil
	}

	// No rules means policy applies based on scope only
	return true, allMatchedRules, nil
}

// evaluateRule evaluates if a rule matches the request
func (e *Evaluator) evaluateRule(ctx context.Context, req EvaluationRequest, rule Rule) (bool, []uuid.UUID, error) {
	var matchedConditions []uuid.UUID

	for _, condition := range rule.Conditions {
		matched, err := e.evaluateCondition(ctx, req, condition)
		if err != nil {
			return false, nil, fmt.Errorf("evaluateCondition: %w", err)
		}

		if matched {
			matchedConditions = append(matchedConditions, condition.ID)
		} else {
			// Apply negation
			if condition.Negate {
				matchedConditions = append(matchedConditions, condition.ID)
				matched = true
			}
		}

		// Apply logical operator
		if rule.Operator == LogicalOperatorAND && !matched && !condition.Negate {
			return false, nil, nil
		}
		if rule.Operator == LogicalOperatorOR && matched && !condition.Negate {
			return true, matchedConditions, nil
		}
	}

	return len(matchedConditions) > 0, matchedConditions, nil
}

// evaluateCondition evaluates a single condition
func (e *Evaluator) evaluateCondition(ctx context.Context, req EvaluationRequest, condition Condition) (bool, error) {
	switch condition.Type {
	case ConditionTypeTime:
		return e.evaluateTimeCondition(req, condition)
	case ConditionTypeIP:
		return e.evaluateIPCondition(req, condition)
	case ConditionTypeRole:
		return e.evaluateRoleCondition(req, condition)
	case ConditionTypeResource:
		return e.evaluateResourceCondition(req, condition)
	case ConditionTypeUser:
		return e.evaluateUserCondition(req, condition)
	case ConditionTypeGroup:
		return e.evaluateGroupCondition(req, condition)
	case ConditionTypeMFAMethod:
		return e.evaluateMFAMethodCondition(req, condition)
	case ConditionTypeMFATrust:
		return e.evaluateMFATrustCondition(req, condition)
	case ConditionTypeGeo:
		return e.evaluateGeoCondition(req, condition)
	case ConditionTypeDeviceTrust:
		return e.evaluateDeviceCondition(req, condition)
	default:
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}
}

// evaluateTimeCondition evaluates time-based conditions
func (e *Evaluator) evaluateTimeCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.TimeData == nil {
		return false, nil
	}

	data := condition.TimeData
	now := req.Time

	// Get the timezone for evaluation
	evalTime := now
	if data.Timezone != "" {
		loc, err := time.LoadLocation(data.Timezone)
		if err != nil {
			e.logger.Warn().Str("timezone", data.Timezone).Err(err).Msg("Invalid timezone, using UTC")
			loc = time.UTC
		}
		evalTime = now.In(loc)
	}

	// Check day of week
	if len(data.DaysOfWeek) > 0 {
		dayMatch := false
		currentDay := evalTime.Weekday().String()
		for _, allowedDay := range data.DaysOfWeek {
			if strings.EqualFold(currentDay, allowedDay.String()) {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			return false, nil
		}
	}

	// Check time range
	if data.StartTime != "" || data.EndTime != "" {
		currentTime := evalTime.Format("15:04")

		if data.StartTime != "" && currentTime < data.StartTime {
			return false, nil
		}
		if data.EndTime != "" && currentTime > data.EndTime {
			return false, nil
		}
	}

	// Check date range
	if data.DateRange != nil {
		if now.Before(data.DateRange.StartDate) || now.After(data.DateRange.EndDate) {
			return false, nil
		}
	}

	return true, nil
}

// evaluateIPCondition evaluates IP-based conditions
func (e *Evaluator) evaluateIPCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.IPData == nil {
		return false, nil
	}

	data := condition.IPData
	clientIP := net.ParseIP(req.ClientIP)
	if clientIP == nil {
		return false, nil
	}

	// Check CIDR ranges
	if len(data.CIDRs) > 0 {
		for _, cidr := range data.CIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				e.logger.Warn().Str("cidr", cidr).Err(err).Msg("Invalid CIDR")
				continue
			}
			if ipNet.Contains(clientIP) {
				return true, nil
			}
		}
		return false, nil
	}

	// Check IP ranges
	if len(data.IPRanges) > 0 {
		for _, ipRange := range data.IPRanges {
			start := net.ParseIP(ipRange.Start)
			end := net.ParseIP(ipRange.End)
			if start != nil && end != nil {
				if e.isIPInRange(clientIP, start, end) {
					return true, nil
				}
			}
		}
		return false, nil
	}

	return true, nil
}

// isIPInRange checks if an IP is within a range
func (e *Evaluator) isIPInRange(ip, start, end net.IP) bool {
	ip = ip.To16()
	start = start.To16()
	end = end.To16()

	for i := 0; i < len(ip); i++ {
		if ip[i] < start[i] || ip[i] > end[i] {
			return false
		}
	}
	return true
}

// evaluateRoleCondition evaluates role-based conditions
func (e *Evaluator) evaluateRoleCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.RoleData == nil {
		return false, nil
	}

	data := condition.RoleData
	if len(data.RoleIDs) == 0 {
		return true, nil
	}

	roleMap := make(map[uuid.UUID]bool)
	for _, roleID := range req.UserRoles {
		roleMap[roleID] = true
	}

	switch data.Match {
	case "exact":
		return len(roleMap) == len(data.RoleIDs), nil
	case "all":
		for _, requiredRole := range data.RoleIDs {
			if !roleMap[requiredRole] {
				return false, nil
			}
		}
		return true, nil
	default: // "any"
		for _, requiredRole := range data.RoleIDs {
			if roleMap[requiredRole] {
				return true, nil
			}
		}
		return false, nil
	}
}

// evaluateResourceCondition evaluates resource-based conditions
func (e *Evaluator) evaluateResourceCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.ResourceData == nil {
		return true, nil
	}

	data := condition.ResourceData

	// If no specific resources, matches all
	if len(data.ResourceIDs) == 0 && len(data.Targets) == 0 && len(data.Credentials) == 0 {
		return true, nil
	}

	// Check if the resource ID matches
	resourceID := req.ResourceID
	if resourceID == uuid.Nil {
		return false, nil
	}

	for _, rid := range data.ResourceIDs {
		if rid == resourceID {
			return true, nil
		}
	}

	for _, tid := range data.Targets {
		if tid == resourceID {
			return true, nil
		}
	}

	for _, cid := range data.Credentials {
		if cid == resourceID {
			return true, nil
		}
	}

	return false, nil
}

// evaluateUserCondition evaluates user-based conditions
func (e *Evaluator) evaluateUserCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.UserData == nil {
		return true, nil
	}

	data := condition.UserData

	if len(data.UserIDs) > 0 {
		for _, uid := range data.UserIDs {
			if uid == req.UserID {
				return true, nil
			}
		}
		return false, nil
	}

	// Email matching would require additional user data from request
	// For now, check if there's a custom attribute match
	if data.Attribute != "" {
		if val, ok := req.Context[data.Attribute]; ok {
			return fmt.Sprintf("%v", val) == data.Value, nil
		}
	}

	return true, nil
}

// evaluateGroupCondition evaluates group-based conditions
func (e *Evaluator) evaluateGroupCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.GroupData == nil {
		return true, nil
	}

	data := condition.GroupData
	if len(data.GroupIDs) == 0 {
		return true, nil
	}

	groupMap := make(map[uuid.UUID]bool)
	for _, groupID := range req.UserGroups {
		groupMap[groupID] = true
	}

	for _, requiredGroup := range data.GroupIDs {
		if groupMap[requiredGroup] {
			return true, nil
		}
	}

	return false, nil
}

// evaluateMFAMethodCondition evaluates MFA method conditions
func (e *Evaluator) evaluateMFAMethodCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.MFAMethodData == nil {
		return true, nil
	}

	data := condition.MFAMethodData
	if len(data.Methods) == 0 {
		return true, nil
	}

	// If MFA not verified, fail
	if !req.MFAVerified {
		return false, nil
	}

	// Check if the used MFA method matches requirements
	for _, method := range data.Methods {
		if method == req.MFAMethod {
			return true, nil
		}
	}

	return false, nil
}

// evaluateMFATrustCondition evaluates MFA trust level conditions
func (e *Evaluator) evaluateMFATrustCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.MFATrustData == nil {
		return true, nil
	}

	// If MFA not verified, fail
	if !req.MFAVerified {
		return false, nil
	}

	// Trust level evaluation would require additional trust context
	// For now, check if MFA was verified
	if req.MFAVerified {
		return true, nil
	}

	return false, nil
}

// evaluateGeoCondition evaluates geographical conditions
func (e *Evaluator) evaluateGeoCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.GeoData == nil {
		return true, nil
	}

	// Geo-based evaluation requires additional context or integration
	// For now, return true as this is a placeholder for future geo-IP lookup
	return true, nil
}

// evaluateDeviceCondition evaluates device trust conditions
func (e *Evaluator) evaluateDeviceCondition(req EvaluationRequest, condition Condition) (bool, error) {
	if condition.DeviceData == nil {
		return true, nil
	}

	data := condition.DeviceData

	// Check minimum trust score
	if data.MinimumTrustScore > 0 && req.DeviceTrust < data.MinimumTrustScore {
		return false, nil
	}

	// Check managed device requirement
	if data.Managed {
		// Would require device management integration
		// For now, check trust score as proxy
		return req.DeviceTrust >= 50, nil
	}

	return true, nil
}

// policyAppliesToUser checks if a policy applies to a user
func (e *Evaluator) policyAppliesToUser(policy Policy, userID uuid.UUID, userRoles []uuid.UUID) bool {
	// If no user/role restrictions, applies to all
	if len(policy.UserIDs) == 0 && len(policy.RoleIDs) == 0 {
		return true
	}

	// Check user IDs
	for _, uid := range policy.UserIDs {
		if uid == userID {
			return true
		}
	}

	// Check role IDs
	roleMap := make(map[uuid.UUID]bool)
	for _, roleID := range userRoles {
		roleMap[roleID] = true
	}

	for _, policyRoleID := range policy.RoleIDs {
		if roleMap[policyRoleID] {
			return true
		}
	}

	return false
}

// policyAppliesToResource checks if a policy applies to a resource
func (e *Evaluator) policyAppliesToResource(policy Policy, resourceID uuid.UUID) bool {
	// If no resource restrictions, applies to all
	if len(policy.TargetIDs) == 0 && len(policy.CredentialIDs) == 0 {
		return true
	}

	// Check target IDs
	for _, tid := range policy.TargetIDs {
		if tid == resourceID {
			return true
		}
	}

	// Check credential IDs
	for _, cid := range policy.CredentialIDs {
		if cid == resourceID {
			return true
		}
	}

	return false
}

// sortPoliciesByPriority sorts policies by priority (higher first)
func (e *Evaluator) sortPoliciesByPriority(policies []Policy) {
	// Simple insertion sort (small lists typically)
	for i := 1; i < len(policies); i++ {
		key := policies[i]
		j := i - 1
		for j >= 0 && policies[j].Priority < key.Priority {
			policies[j+1] = policies[j]
			j--
		}
		policies[j+1] = key
	}
}

// IsCommandAllowed checks if a command is allowed based on command filter patterns
func (e *Evaluator) IsCommandAllowed(ctx context.Context, command string, patterns []CommandFilterPattern) (bool, string, error) {
	// If no patterns, allow all
	if len(patterns) == 0 {
		return true, "", nil
	}

	// Check whitelist patterns first
	for _, pattern := range patterns {
		if pattern.IsWhitelist {
			matched, err := e.matchesPattern(command, pattern.Pattern)
			if err != nil {
				return false, "", fmt.Errorf("whitelist pattern match error: %w", err)
			}
			if matched {
				return true, "", nil
			}
		}
	}

	// Check blacklist patterns
	for _, pattern := range patterns {
		if !pattern.IsWhitelist {
			matched, err := e.matchesPattern(command, pattern.Pattern)
			if err != nil {
				return false, "", fmt.Errorf("blacklist pattern match error: %w", err)
			}
			if matched {
				return false, pattern.Description, nil
			}
		}
	}

	// If whitelist exists but no match, deny
	for _, pattern := range patterns {
		if pattern.IsWhitelist {
			return false, "Command not in whitelist", nil
		}
	}

	// Default allow if no whitelist
	return true, "", nil
}

// matchesPattern checks if a command matches a pattern (simplified regex)
func (e *Evaluator) matchesPattern(command, pattern string) (bool, error) {
	// For now, use simple string matching
	// In production, use RE2 for proper regex matching
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		patternRegex := strings.ReplaceAll(pattern, "*", ".*")
		return strings.Contains(command, strings.Trim(patternRegex, ".*")), nil
	}
	return strings.Contains(command, pattern), nil
}

// ValidatePolicy validates a policy before saving
func (e *Evaluator) ValidatePolicy(policy *Policy) error {
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if !policy.Type.IsValid() {
		return fmt.Errorf("invalid policy type: %s", policy.Type)
	}

	if policy.Effect != PolicyEffectAllow && policy.Effect != PolicyEffectDeny {
		return fmt.Errorf("invalid policy effect: %s", policy.Effect)
	}

	if policy.Priority < 0 {
		return fmt.Errorf("priority must be non-negative")
	}

	if policy.TenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}

	// Validate rules
	for i, rule := range policy.Rules {
		if err := e.validateRule(&rule); err != nil {
			return fmt.Errorf("rule %d: %w", i, err)
		}
	}

	// Validate MFA settings
	if policy.MFARequired && len(policy.MFAMethods) == 0 {
		return fmt.Errorf("MFA required but no methods specified")
	}

	// Validate approval settings
	if policy.ApprovalRequired && len(policy.ApprovalApprovers) == 0 {
		return fmt.Errorf("approval required but no approvers specified")
	}

	return nil
}

// validateRule validates a rule
func (e *Evaluator) validateRule(rule *Rule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	// Validate rule type if specified
	if rule.Type != "" {
		validTypes := map[RuleType]bool{
			RuleTypeTimeRestriction:     true,
			RuleTypeIPRestriction:       true,
			RuleTypeMFARequirement:      true,
			RuleTypeCommandFilter:       true,
			RuleTypeSessionRecording:    true,
			RuleTypeApprovalRequirement: true,
			RuleTypeDurationLimit:       true,
			RuleTypeConcurrentLimit:     true,
		}
		if !validTypes[rule.Type] {
			return fmt.Errorf("invalid rule type: %s", rule.Type)
		}
	}

	// Validate conditions
	for i, cond := range rule.Conditions {
		if err := e.validateCondition(&cond); err != nil {
			return fmt.Errorf("condition %d: %w", i, err)
		}
	}

	return nil
}

// validateCondition validates a condition
func (e *Evaluator) validateCondition(condition *Condition) error {
	if condition.Type == "" {
		return fmt.Errorf("condition type is required")
	}

	// Validate condition-specific data
	switch condition.Type {
	case ConditionTypeTime:
		if condition.TimeData == nil {
			return fmt.Errorf("time data required for time condition")
		}
	case ConditionTypeIP:
		if condition.IPData == nil {
			return fmt.Errorf("IP data required for IP condition")
		}
	case ConditionTypeRole:
		if condition.RoleData == nil {
			return fmt.Errorf("role data required for role condition")
		}
	}

	return nil
}

