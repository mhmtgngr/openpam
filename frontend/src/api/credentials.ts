import { api } from './client';
import type { Credential, CredentialCheckout, PaginatedResponse } from '@/types';

export interface CredentialListParams {
  limit?: number;
  offset?: number;
  search?: string;
  type?: string;
  status?: string;
  folder_id?: string;
}

export interface CreateCredentialData {
  name: string;
  type: string;
  target_id?: string;
  username: string;
  password?: string;
  ssh_key?: string;
  ssh_key_passphrase?: string;
  api_key?: string;
  database_name?: string;
  database_type?: string;
  rotation_policy?: string;
  rotation_schedule?: string;
  checkout_enabled?: boolean;
  max_checkout_duration?: number;
  auto_checkin?: boolean;
  require_approval?: boolean;
  approver_ids?: string[];
  folder_id?: string;
  description?: string;
  tags?: string[];
}

export const credentialsApi = {
  // List credentials
  list: (params?: CredentialListParams) =>
    api.get<PaginatedResponse<Credential>>('/credentials', params),

  // Get credential by ID (without secret)
  get: (id: string) => api.get<Credential>(`/credentials/${id}`),

  // Create credential
  create: (data: CreateCredentialData) => api.post<Credential>('/credentials', data),

  // Update credential (metadata only)
  update: (id: string, data: Partial<CreateCredentialData>) =>
    api.patch<Credential>(`/credentials/${id}`, data),

  // Delete credential
  delete: (id: string) => api.delete<void>(`/credentials/${id}`),

  // Checkout credential (returns secret)
  checkout: (id: string, reason: string, durationMinutes: number, requestId?: string) =>
    api.post<{ credential: CredentialCheckout; secret: string }>(`/credentials/${id}/checkout`, {
      reason,
      duration_minutes: durationMinutes,
      request_id: requestId,
    }),

  // Checkin credential
  checkin: (id: string) => api.post<void>(`/credentials/${id}/checkin`, {}),

  // Rotate credential now
  rotate: (id: string) => api.post<Credential>(`/credentials/${id}/rotate`, {}),

  // Get active checkouts
  getActiveCheckouts: (params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<CredentialCheckout>>('/credentials/checkouts', params),

  // Force checkin
  forceCheckin: (checkoutId: string, reason: string) =>
    api.post<void>(`/credentials/checkouts/${checkoutId}/force-checkin`, { reason }),

  // Get credential history
  getHistory: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<unknown>>(`/credentials/${id}/history`, params),
};
