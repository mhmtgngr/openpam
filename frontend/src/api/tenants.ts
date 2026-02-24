import { api } from './client';
import type { Tenant, PaginatedResponse } from '@/types';

export interface TenantListParams {
  limit?: number;
  offset?: number;
  search?: string;
  status?: string;
}

export interface CreateTenantData {
  name: string;
  slug: string;
  logo_url?: string;
  primary_color?: string;
  settings?: {
    enforce_mfa?: boolean;
    mfa_methods?: ('totp' | 'webauthn')[];
    session_timeout_minutes?: number;
    audit_retention_days?: number;
    recording_retention_days?: number;
    ip_whitelist?: string[];
  };
  max_users?: number;
  max_targets?: number;
}

export interface UpdateTenantData extends Partial<CreateTenantData> {}

export const tenantsApi = {
  // List tenants
  list: (params?: TenantListParams) =>
    api.get<PaginatedResponse<Tenant>>('/tenants', params),

  // Get tenant by ID
  get: (id: string) => api.get<Tenant>(`/tenants/${id}`),

  // Create tenant
  create: (data: CreateTenantData) => api.post<Tenant>('/tenants', data),

  // Update tenant
  update: (id: string, data: UpdateTenantData) => api.patch<Tenant>(`/tenants/${id}`, data),

  // Delete tenant
  delete: (id: string) => api.delete<void>(`/tenants/${id}`),

  // Get current tenant
  current: () => api.get<Tenant>('/tenants/current'),

  // Update current tenant settings
  updateSettings: (settings: CreateTenantData['settings']) =>
    api.patch<Tenant>('/tenants/current/settings', settings),

  // Suspend tenant
  suspend: (id: string, reason: string) =>
    api.post<Tenant>(`/tenants/${id}/suspend`, { reason }),

  // Activate tenant
  activate: (id: string) => api.post<Tenant>(`/tenants/${id}/activate`, {}),

  // Get tenant usage stats
  getStats: (id: string) =>
    api.get<{ users: number; targets: number; credentials: number; sessions: number }>(
      `/tenants/${id}/stats`
    ),
};
