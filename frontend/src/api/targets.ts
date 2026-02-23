import { api } from './client';
import type { Target, TargetGroup, PaginatedResponse } from '@/types';

export interface TargetListParams {
  limit?: number;
  offset?: number;
  search?: string;
  type?: string;
  environment?: string;
  status?: string;
  folder_id?: string;
}

export interface CreateTargetData {
  name: string;
  type: string;
  host: string;
  port: number;
  description?: string;
  environment?: string;
  sensitivity?: string;
  tags?: string[];
  approver_ids?: string[];
  connection_timeout?: number;
  max_session_duration?: number;
  require_approval?: boolean;
  require_reason?: boolean;
  require_mfa?: boolean;
  allow_recording?: boolean;
  folder_id?: string;
}

export interface UpdateTargetData extends Partial<CreateTargetData> {}

export const targetsApi = {
  // List targets
  list: (params?: TargetListParams) =>
    api.get<PaginatedResponse<Target>>('/targets', params),

  // Get target by ID
  get: (id: string) => api.get<Target>(`/targets/${id}`),

  // Create target
  create: (data: CreateTargetData) => api.post<Target>('/targets', data),

  // Update target
  update: (id: string, data: UpdateTargetData) => api.patch<Target>(`/targets/${id}`, data),

  // Delete target
  delete: (id: string) => api.delete<void>(`/targets/${id}`),

  // Test connection
  testConnection: (id: string) => api.post<{ status: 'online' | 'offline'; latency_ms?: number }>(`/targets/${id}/test`, {}),

  // Get target groups
  listGroups: () => api.get<PaginatedResponse<TargetGroup>>('/target-groups'),

  // Create target group
  createGroup: (name: string, description?: string, targetIds?: string[]) =>
    api.post<TargetGroup>('/target-groups', { name, description, target_ids: targetIds }),

  // Update target group
  updateGroup: (id: string, data: { name?: string; description?: string; target_ids?: string[] }) =>
    api.patch<TargetGroup>(`/target-groups/${id}`, data),

  // Delete target group
  deleteGroup: (id: string) => api.delete<void>(`/target-groups/${id}`),

  // Get target sessions
  getSessions: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<unknown>>(`/targets/${id}/sessions`, params),
};
