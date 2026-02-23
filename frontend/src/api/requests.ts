import { api } from './client';
import type { AccessRequest, RequestComment, PaginatedResponse } from '@/types';

export interface RequestListParams {
  limit?: number;
  offset?: number;
  status?: string;
  type?: string;
  user_id?: string;
}

export interface CreateRequestData {
  target_id: string;
  credential_id?: string;
  type: string;
  reason: string;
  duration_minutes: number;
  scheduled_start_at?: string;
}

export const requestsApi = {
  // List requests
  list: (params?: RequestListParams) =>
    api.get<PaginatedResponse<AccessRequest>>('/requests', params),

  // Get request by ID
  get: (id: string) => api.get<AccessRequest>(`/requests/${id}`),

  // Create request
  create: (data: CreateRequestData) => api.post<AccessRequest>('/requests', data),

  // Cancel request
  cancel: (id: string, reason: string) =>
    api.post<AccessRequest>(`/requests/${id}/cancel`, { reason }),

  // Approve request
  approve: (id: string, comment?: string) =>
    api.post<AccessRequest>(`/requests/${id}/approve`, { comment }),

  // Deny request
  deny: (id: string, reason: string) =>
    api.post<AccessRequest>(`/requests/${id}/deny`, { reason }),

  // Delegate approval
  delegate: (id: string, toUserId: string, reason: string) =>
    api.post<AccessRequest>(`/requests/${id}/delegate`, { to_user_id: toUserId, reason }),

  // Expedite request
  expedite: (id: string, reason: string) =>
    api.post<AccessRequest>(`/requests/${id}/expedite`, { reason }),

  // Add comment
  addComment: (id: string, content: string) =>
    api.post<RequestComment>(`/requests/${id}/comments`, { content }),

  // List pending approvals for current user
  pendingApprovals: () => api.get<PaginatedResponse<AccessRequest>>('/requests/pending-approvals'),

  // Get my requests
  myRequests: (params?: { limit?: number; offset?: number; status?: string }) =>
    api.get<PaginatedResponse<AccessRequest>>('/requests/my', params),
};
