export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: 'super_admin' | 'admin' | 'operator' | 'auditor' | 'user';
  status: 'active' | 'suspended' | 'locked';
  mfa_enabled: boolean;
  tenant_id: string;
  last_login_at: string | null;
  created_at: string;
}

export interface Credential {
  id: string;
  name: string;
  type: 'ssh_key' | 'password' | 'api_key' | 'certificate' | 'database';
  host: string;
  port: number;
  username: string;
  rotation_policy: 'manual' | 'daily' | 'weekly' | 'on_checkin';
  last_rotated_at: string | null;
  folder_id: string | null;
  tenant_id: string;
  created_at: string;
}

export interface Session {
  id: string;
  user_id: string;
  credential_id: string;
  type: 'ssh' | 'rdp' | 'database' | 'kubernetes' | 'web';
  status: 'active' | 'ended' | 'terminated' | 'failed';
  target_host: string;
  target_port: number;
  recording_url: string;
  started_at: string;
  ended_at: string | null;
  terminated_by: string | null;
}

export interface CheckoutRequest {
  id: string;
  user_id: string;
  credential_id: string;
  justification: string;
  duration_minutes: number;
  status: 'pending' | 'approved' | 'denied' | 'checked_out' | 'checked_in' | 'expired';
  approved_by: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface AuditEvent {
  id: string;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  outcome: 'success' | 'failure' | 'denied';
  ip: string;
  details: Record<string, unknown>;
  created_at: string;
}

export interface DiscoveredAsset {
  id: string;
  hostname: string;
  ip: string;
  os: string;
  open_ports: number[];
  services: string[];
  status: 'discovered' | 'managed' | 'ignored';
  risk_score: number;
  last_scanned_at: string;
}
