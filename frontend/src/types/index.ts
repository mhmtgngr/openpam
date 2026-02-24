// User & Authentication
export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  display_name?: string;
  role: UserRole;
  status: UserStatus;
  mfa_enabled: boolean;
  mfa_method?: 'totp' | 'webauthn' | 'both';
  tenant_id: string;
  last_login_at: string | null;
  failed_login_attempts: number;
  locked_until: string | null;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export type UserRole = 'super_admin' | 'admin' | 'operator' | 'auditor' | 'requester' | 'user';
export type UserStatus = 'active' | 'suspended' | 'locked' | 'pending';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
  mfa_required?: boolean;
  mfa_setup_token?: string;
}

export interface MFADevice {
  id: string;
  type: 'totp' | 'webauthn';
  name: string;
  last_used_at: string | null;
  created_at: string;
}

export interface TOTPSetup {
  secret: string;
  qr_code_url: string;
  backup_codes: string[];
}

// Roles & Permissions
export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  permissions: Permission[];
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  resource: string;
  action: string;
  condition?: string;
  description?: string;
}

export const PERMISSION_RESOURCES = [
  'users',
  'roles',
  'targets',
  'credentials',
  'requests',
  'sessions',
  'audits',
  'vaults',
  'policies',
  'tenants',
] as const;

export const PERMISSION_ACTIONS = [
  'create',
  'read',
  'update',
  'delete',
  'approve',
  'checkout',
  'checkin',
  'terminate',
  'export',
] as const;

// Targets
export interface Target {
  id: string;
  name: string;
  type: TargetType;
  host: string;
  port: number;
  description?: string;
  environment: 'production' | 'staging' | 'development' | 'test';
  sensitivity: 'high' | 'medium' | 'low';
  status: 'online' | 'offline' | 'unknown';
  platform?: string;
  os_version?: string;
  tags: string[];
  approver_ids: string[];
  connection_timeout: number;
  max_session_duration: number;
  require_approval: boolean;
  require_reason: boolean;
  require_mfa: boolean;
  allow_recording: boolean;
  auto_rotate_credentials: boolean;
  folder_id?: string;
  tenant_id: string;
  last_connected_at?: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export type TargetType = 'ssh' | 'rdp' | 'database' | 'kubernetes' | 'web' | 'api';

export interface TargetGroup {
  id: string;
  name: string;
  description?: string;
  target_ids: string[];
  tenant_id: string;
  created_at: string;
}

// Credentials
export interface Credential {
  id: string;
  name: string;
  type: CredentialType;
  target_id?: string;
  target?: Target;
  username: string;
  password?: string; // Only returned on checkout
  ssh_key?: string; // Only returned on checkout
  ssh_key_passphrase?: string;
  api_key?: string;
  database_name?: string;
  database_type?: 'mysql' | 'postgresql' | 'mssql' | 'oracle' | 'mongodb' | 'redis';
  rotation_policy: RotationPolicy;
  rotation_schedule?: string; // Cron expression
  last_rotated_at?: string;
  next_rotation_at?: string;
  rotation_status: 'success' | 'failed' | 'pending' | 'never';
  checkout_enabled: boolean;
  max_checkout_duration: number;
  auto_checkin: boolean;
  require_approval: boolean;
  approver_ids: string[];
  folder_id?: string;
  description?: string;
  tags: string[];
  status: 'active' | 'expiring' | 'expired' | 'rotating';
  tenant_id: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

export type CredentialType = 'password' | 'ssh_key' | 'api_key' | 'certificate' | 'database' | 'service_account';
export type RotationPolicy = 'manual' | 'daily' | 'weekly' | 'monthly' | 'on_checkin' | 'on_expiry';

export interface CredentialCheckout {
  id: string;
  credential_id: string;
  credential_name?: string;
  user_id: string;
  user?: User;
  request_id?: string;
  reason: string;
  duration_minutes: number;
  auto_checkin: boolean;
  checked_out_at: string;
  expires_at: string;
  checked_in_at?: string;
  terminated_by?: string;
  terminated_reason?: string;
  status: CheckoutStatus;
}

export type CheckoutStatus = 'active' | 'checked_in' | 'terminated' | 'expired';

// Access Requests
export interface AccessRequest {
  id: string;
  user_id: string;
  user?: User;
  target_id: string;
  target?: Target;
  credential_id?: string;
  credential?: Credential;
  type: RequestType;
  reason: string;
  duration_minutes: number;
  scheduled_start_at?: string;
  scheduled_end_at?: string;
  status: RequestStatus;
  approval_workflow_id?: string;
  current_step?: number;
  approvers: RequestApprover[];
  comments?: RequestComment[];
  expedited: boolean;
  expedited_by?: string;
  expedited_at?: string;
  expedited_reason?: string;
  denied_by?: string;
  denied_reason?: string;
  denied_at?: string;
  approved_at?: string;
  expires_at?: string;
  created_at: string;
  updated_at: string;
}

export type RequestType = 'credential_checkout' | 'session_access' | 'elevated_privileges';
export type RequestStatus = 'pending' | 'approved' | 'denied' | 'cancelled' | 'expired' | 'active' | 'completed';

export interface RequestApprover {
  id: string;
  user_id: string;
  user?: User;
  role_id?: string;
  step: number;
  status: 'pending' | 'approved' | 'denied' | 'delegated';
  delegated_to?: string;
  decided_at?: string;
  comment?: string;
}

export interface RequestComment {
  id: string;
  user_id: string;
  user?: User;
  content: string;
  created_at: string;
}

export interface ApprovalWorkflow {
  id: string;
  name: string;
  description?: string;
  type: RequestType;
  target_ids?: string[];
  credential_ids?: string[];
  conditions?: WorkflowCondition[];
  steps: WorkflowStep[];
  is_default: boolean;
  status: 'active' | 'inactive';
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

export interface WorkflowCondition {
  field: string;
  operator: 'equals' | 'contains' | 'starts_with' | 'ends_with' | 'regex' | 'in';
  value: string | string[];
}

export interface WorkflowStep {
  step: number;
  type: 'any' | 'all' | 'role' | 'specific';
  user_ids?: string[];
  role_ids?: string[];
  timeout_minutes?: number;
  escalation_step?: number;
}

// Sessions
export interface Session {
  id: string;
  user_id: string;
  user?: User;
  target_id: string;
  target?: Target;
  credential_id?: string;
  request_id?: string;
  type: SessionType;
  status: SessionStatus;
  client_ip: string;
  client_user_agent?: string;
  started_at: string;
  ended_at?: string;
  duration_seconds?: number;
  recording_url?: string;
  recording_size?: number;
  monitoring_enabled: boolean;
  can_terminate: boolean;
  terminated_by?: string;
  terminated_reason?: string;
  metadata?: Record<string, unknown>;
}

export type SessionType = 'ssh' | 'rdp' | 'database' | 'kubernetes' | 'web';
export type SessionStatus = 'starting' | 'active' | 'ended' | 'terminated' | 'failed';

export interface SessionEvent {
  id: string;
  session_id: string;
  timestamp: string;
  type: 'keystroke' | 'command' | 'file_upload' | 'file_download' | 'screenshot' | 'warning' | 'error';
  data: Record<string, unknown>;
}

// Audit & Compliance
export interface AuditEvent {
  id: string;
  tenant_id: string;
  actor_id: string;
  actor?: User;
  actor_ip: string;
  action: string;
  resource_type: string;
  resource_id: string;
  resource_name?: string;
  outcome: AuditOutcome;
  details?: Record<string, unknown>;
  correlation_id?: string;
  session_id?: string;
  request_id?: string;
  created_at: string;
}

export type AuditOutcome = 'success' | 'failure' | 'denied' | 'partial';

export interface ComplianceReport {
  id: string;
  name: string;
  type: 'soc2' | 'iso27001' | 'pci_dss' | 'hipaa' | 'custom';
  description?: string;
  schedule: string;
  last_run_at?: string;
  next_run_at?: string;
  status: 'scheduled' | 'running' | 'completed' | 'failed';
  config: ReportConfig;
  created_by: string;
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

export interface ReportConfig {
  period_start: string;
  period_end: string;
  include_sections: string[];
  filters: Record<string, unknown>;
}

// Dashboard & Metrics
export interface DashboardStats {
  users: {
    total: number;
    active: number;
    online: number;
  };
  targets: {
    total: number;
    online: number;
    offline: number;
  };
  credentials: {
    total: number;
    expiring_soon: number;
    expired: number;
  };
  sessions: {
    active: number;
    today: number;
    avg_duration: number;
  };
  requests: {
    pending: number;
    approved_today: number;
    denied_today: number;
  };
  audit: {
    total_events: number;
    failures: number;
    denied_access: number;
  };
}

export interface ActivityFeed {
  id: string;
  type: 'user_login' | 'user_logout' | 'request_created' | 'request_approved' | 'request_denied' | 'session_started' | 'session_ended' | 'credential_rotated' | 'target_added';
  message: string;
  actor?: string;
  resource_type?: string;
  resource_name?: string;
  timestamp: string;
}

// Folders
export interface Folder {
  id: string;
  name: string;
  parent_id?: string;
  path: string;
  type: 'targets' | 'credentials' | 'policies';
  description?: string;
  children?: Folder[];
  item_count: number;
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

// Notifications
export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  data?: Record<string, unknown>;
  read: boolean;
  created_at: string;
}

export type NotificationType =
  | 'request_pending_approval'
  | 'request_approved'
  | 'request_denied'
  | 'request_expiring'
  | 'session_warning'
  | 'credential_expiring'
  | 'mfa_enabled_required'
  | 'account_locked';

// Policies - exported from policy.ts
export type {
  PolicyEffect,
  PolicyStatus,
  ConditionType,
  ConditionOperator,
  LogicalOperator,
  PolicyCondition,
  PolicyRule,
  AccessPolicy,
  PolicyEvaluationRequest,
  PolicyEvaluationResult,
  PolicyEvaluationDetails,
  PolicyEvaluationLog,
  PolicyValidationResult,
  PolicyValidationError,
  AccessPolicyListFilter,
  CreateAccessPolicyData,
  UpdateAccessPolicyData,
  TestPolicyRequest,
  TestScenario,
  TestPolicyResult,
  ConditionTemplate,
  ConditionFieldTemplate,
  RuleTemplate,
} from './policy';

export interface PasswordPolicy {
  id: string;
  name: string;
  min_length: number;
  max_length: number;
  require_uppercase: boolean;
  require_lowercase: boolean;
  require_numbers: boolean;
  require_special: boolean;
  forbidden_passwords?: string[];
  expiration_days?: number;
  history_count: number;
  tenant_id: string;
}

export interface SessionPolicy {
  id: string;
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

// Tenants
export interface Tenant {
  id: string;
  name: string;
  slug: string;
  logo_url?: string;
  primary_color?: string;
  settings: TenantSettings;
  status: 'active' | 'suspended' | 'trial';
  max_users: number;
  max_targets: number;
  created_at: string;
  updated_at: string;
}

export interface TenantSettings {
  enforce_mfa: boolean;
  mfa_methods: ('totp' | 'webauthn')[];
  session_timeout_minutes: number;
  password_policy_id?: string;
  default_approval_workflow_id?: string;
  audit_retention_days: number;
  recording_retention_days: number;
  ip_whitelist?: string[];
  allowed_email_domains?: string[];
}

// API Responses
export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
    has_more: boolean;
    next_cursor?: string;
    prev_cursor?: string;
  };
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    request_id?: string;
  };
}

export interface SuccessResponse<T = unknown> {
  data: T;
  message?: string;
}
