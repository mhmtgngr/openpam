import { api } from './client';
import type {
  PasswordPolicy,
  SessionPolicy,
  PaginatedResponse,
} from '@/types';
import type {
  AccessPolicy,
  AccessPolicyListFilter,
  CreateAccessPolicyData,
  UpdateAccessPolicyData,
  PolicyEvaluationRequest,
  PolicyEvaluationResult,
  PolicyEvaluationLog,
  PolicyValidationResult,
  TestPolicyRequest,
  TestPolicyResult,
} from '@/types/policy';

export interface PasswordPolicyListParams {
  limit?: number;
  offset?: number;
  search?: string;
}

export interface CreatePasswordPolicyData {
  name: string;
  min_length: number;
  max_length?: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_numbers: boolean;
  require_special: boolean;
  forbidden_passwords?: string[];
  expiration_days?: number;
  history_count: number;
}

export interface UpdatePasswordPolicyData extends Partial<CreatePasswordPolicyData> {}

export const passwordPoliciesApi = {
  // List password policies
  list: (params?: PasswordPolicyListParams) =>
    api.get<PaginatedResponse<PasswordPolicy>>('/policies/password', params),

  // Get password policy by ID
  get: (id: string) => api.get<PasswordPolicy>(`/policies/password/${id}`),

  // Create password policy
  create: (data: CreatePasswordPolicyData) =>
    api.post<PasswordPolicy>('/policies/password', data),

  // Update password policy
  update: (id: string, data: UpdatePasswordPolicyData) =>
    api.patch<PasswordPolicy>(`/policies/password/${id}`, data),

  // Delete password policy
  delete: (id: string) => api.delete<void>(`/policies/password/${id}`),

  // Set as default
  setDefault: (id: string) => api.post<PasswordPolicy>(`/policies/password/${id}/default`, {}),
};

export interface CreateSessionPolicyData {
  name: string;
  max_duration_minutes: number;
  require_approval: boolean;
  require_reason: boolean;
  require_mfa: boolean;
  allow_recording: boolean;
  monitor_keywords: string[];
  blocked_commands: string[];
  idle_timeout_minutes: number;
  warning_minutes_before_end: number;
}

export interface UpdateSessionPolicyData extends Partial<CreateSessionPolicyData> {}

export const sessionPoliciesApi = {
  // List session policies
  list: (params?: PasswordPolicyListParams) =>
    api.get<PaginatedResponse<SessionPolicy>>('/policies/session', params),

  // Get session policy by ID
  get: (id: string) => api.get<SessionPolicy>(`/policies/session/${id}`),

  // Create session policy
  create: (data: CreateSessionPolicyData) =>
    api.post<SessionPolicy>('/policies/session', data),

  // Update session policy
  update: (id: string, data: UpdateSessionPolicyData) =>
    api.patch<SessionPolicy>(`/policies/session/${id}`, data),

  // Delete session policy
  delete: (id: string) => api.delete<void>(`/policies/session/${id}`),

  // Set as default
  setDefault: (id: string) => api.post<SessionPolicy>(`/policies/session/${id}/default`, {}),
};

// Access Policies (Policy Engine)
export interface AccessPolicyListParams extends AccessPolicyListFilter {}

export const accessPoliciesApi = {
  // List access policies
  list: (params?: AccessPolicyListParams) =>
    api.get<PaginatedResponse<AccessPolicy>>('/policies/access', params),

  // Get access policy by ID
  get: (id: string) => api.get<AccessPolicy>(`/policies/access/${id}`),

  // Create access policy
  create: (data: CreateAccessPolicyData) =>
    api.post<AccessPolicy>('/policies/access', data),

  // Update access policy
  update: (id: string, data: UpdateAccessPolicyData) =>
    api.patch<AccessPolicy>(`/policies/access/${id}`, data),

  // Delete access policy
  delete: (id: string) => api.delete<void>(`/policies/access/${id}`),

  // Validate policy before saving
  validate: (data: CreateAccessPolicyData | UpdateAccessPolicyData) =>
    api.post<PolicyValidationResult>('/policies/access/validate', data),

  // Set as default
  setDefault: (id: string) => api.post<AccessPolicy>(`/policies/access/${id}/default`, {}),

  // Evaluate policy (test access)
  evaluate: (request: PolicyEvaluationRequest) =>
    api.post<PolicyEvaluationResult>('/policies/access/evaluate', request),

  // Get evaluation logs (audit trail)
  evaluationLogs: (params?: {
    user_id?: string;
    policy_id?: string;
    resource?: string;
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
  }) =>
    api.get<PaginatedResponse<PolicyEvaluationLog>>('/policies/access/evaluation-logs', params),

  // Test policy with scenarios
  test: (data: TestPolicyRequest) =>
    api.post<TestPolicyResult[]>('/policies/access/test', data),

  // Clone policy
  clone: (id: string, name: string) =>
    api.post<AccessPolicy>(`/policies/access/${id}/clone`, { name }),

  // Bulk operations
  bulkActivate: (ids: string[]) =>
    api.post<{ success: string[]; failed: string[] }>('/policies/access/bulk/activate', { ids }),
  bulkDeactivate: (ids: string[]) =>
    api.post<{ success: string[]; failed: string[] }>('/policies/access/bulk/deactivate', { ids }),
  bulkDelete: (ids: string[]) =>
    api.post<{ success: string[]; failed: string[] }>('/policies/access/bulk/delete', { ids }),
};

export const policiesApi = {
  password: passwordPoliciesApi,
  session: sessionPoliciesApi,
  access: accessPoliciesApi,
};
