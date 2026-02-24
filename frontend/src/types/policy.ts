// Access Policy Types for Policy Engine

export type PolicyEffect = 'allow' | 'deny';
export type PolicyStatus = 'active' | 'inactive' | 'draft';
export type ConditionType = 'time_range' | 'ip_range' | 'role' | 'resource' | 'user_attribute' | 'custom';
export type ConditionOperator =
  | 'equals'
  | 'not_equals'
  | 'contains'
  | 'not_contains'
  | 'starts_with'
  | 'ends_with'
  | 'in'
  | 'not_in'
  | 'regex'
  | 'ip_in_range'
  | 'time_between'
  | 'day_of_week'
  | 'greater_than'
  | 'less_than';
export type LogicalOperator = 'AND' | 'OR';

// Condition for policy rules
export interface PolicyCondition {
  id?: string;
  type: ConditionType;
  field: string;
  operator: ConditionOperator;
  value: string | string[] | number | boolean;
  label?: string; // Human-readable description
  negated?: boolean; // If true, condition is inverted
}

// Rule within a policy
export interface PolicyRule {
  id?: string;
  name: string;
  description?: string;
  effect: PolicyEffect; // allow or deny
  conditions: PolicyCondition[];
  logical_operator: LogicalOperator; // How to combine conditions
  priority: number; // Higher priority evaluated first
  resources: string[]; // Target resources (e.g., ['target:prod-db', 'credential:ssh-key'])
  actions: string[]; // Actions (e.g., ['checkout', 'connect', 'approve'])
  roles: string[]; // Applicable roles
  users: string[]; // Specific users (empty means all users in roles)
}

// Main Access Policy
export interface AccessPolicy {
  id: string;
  name: string;
  description?: string;
  status: PolicyStatus;
  priority: number;
  rules: PolicyRule[];
  conflict_resolution: 'deny_overrides' | 'allow_overrides' | 'first_applicable';
  is_default: boolean;
  is_system: boolean;
  tenant_id: string;
  created_at: string;
  updated_at: string;
  created_by?: string;
  updated_by?: string;
  effective_from?: string;
  effective_until?: string;
  tags: string[];
}

// Policy evaluation request
export interface PolicyEvaluationRequest {
  user_id: string;
  user_roles: string[];
  user_attributes?: Record<string, string | number | boolean>;
  resource: string;
  action: string;
  context: PolicyEvaluationContext;
}

export interface PolicyEvaluationContext {
  ip_address?: string;
  timestamp?: string;
  request_id?: string;
  session_id?: string;
  metadata?: Record<string, unknown>;
}

// Policy evaluation result
export interface PolicyEvaluationResult {
  allowed: boolean;
  effect: PolicyEffect;
  matched_policy_id?: string;
  matched_rule_id?: string;
  reason: string;
  details: PolicyEvaluationDetails[];
  evaluated_at: string;
}

export interface PolicyEvaluationDetails {
  policy_id: string;
  policy_name: string;
  rule_id?: string;
  rule_name?: string;
  effect: PolicyEffect;
  matched: boolean;
  conditions_met: string[];
  conditions_failed: string[];
}

// Policy evaluation log (for audit/compliance)
export interface PolicyEvaluationLog {
  id: string;
  policy_id?: string;
  policy_name?: string;
  rule_id?: string;
  rule_name?: string;
  user_id: string;
  user_roles: string[];
  resource: string;
  action: string;
  allowed: boolean;
  effect: PolicyEffect;
  reason: string;
  context: PolicyEvaluationContext;
  evaluated_at: string;
  tenant_id: string;
}

// Policy validation result
export interface PolicyValidationResult {
  valid: boolean;
  errors: PolicyValidationError[];
  warnings: PolicyValidationError[];
}

export interface PolicyValidationError {
  field?: string;
  message: string;
  severity: 'error' | 'warning';
  code?: string;
}

// Filter types for list views
export interface AccessPolicyListFilter {
  search?: string;
  status?: PolicyStatus;
  effect?: PolicyEffect;
  tags?: string[];
  limit?: number;
  offset?: number;
}

// Create/Update data types
export interface CreateAccessPolicyData {
  name: string;
  description?: string;
  status: PolicyStatus;
  priority: number;
  rules: Omit<PolicyRule, 'id'>[];
  conflict_resolution: 'deny_overrides' | 'allow_overrides' | 'first_applicable';
  effective_from?: string;
  effective_until?: string;
  tags: string[];
}

export interface UpdateAccessPolicyData extends Partial<CreateAccessPolicyData> {}

// Test Policy Request
export interface TestPolicyRequest {
  policy: AccessPolicy;
  test_scenarios: TestScenario[];
}

export interface TestScenario {
  name: string;
  user_id: string;
  user_roles: string[];
  resource: string;
  action: string;
  context: PolicyEvaluationContext;
  expected_allowed?: boolean;
}

export interface TestPolicyResult {
  scenario_name: string;
  result: PolicyEvaluationResult;
  passed: boolean;
  expected_allowed?: boolean;
}

// Helper type for creating new conditions in the UI
export interface ConditionTemplate {
  type: ConditionType;
  label: string;
  fields: ConditionFieldTemplate[];
  description: string;
}

export interface ConditionFieldTemplate {
  name: string;
  label: string;
  type: 'text' | 'number' | 'select' | 'multiselect' | 'boolean' | 'ip' | 'time' | 'date' | 'regex';
  options?: { value: string; label: string }[];
  placeholder?: string;
  default?: unknown;
  validation?: string; // regex or validation rule
}

// Rule template for quick creation
export interface RuleTemplate {
  name: string;
  description: string;
  effect: PolicyEffect;
  conditions: Omit<PolicyCondition, 'id'>[];
  resources: string[];
  actions: string[];
  category: string;
}
