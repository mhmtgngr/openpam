import { api } from './client';
import type { Role, Permission, PaginatedResponse } from '@/types';

export interface RoleListParams {
  limit?: number;
  offset?: number;
  search?: string;
}

export interface CreateRoleData {
  name: string;
  description?: string;
  permission_ids?: string[];
}

export interface UpdateRoleData {
  name?: string;
  description?: string;
  permission_ids?: string[];
}

export const rolesApi = {
  // List roles
  list: (params?: RoleListParams) =>
    api.get<PaginatedResponse<Role>>('/roles', params),

  // Get role by ID
  get: (id: string) => api.get<Role>(`/roles/${id}`),

  // Create role
  create: (data: CreateRoleData) => api.post<Role>('/roles', data),

  // Update role
  update: (id: string, data: UpdateRoleData) => api.patch<Role>(`/roles/${id}`, data),

  // Delete role
  delete: (id: string) => api.delete<void>(`/roles/${id}`),

  // Grant permission
  grantPermission: (id: string, permissionId: string) =>
    api.post<Role>(`/roles/${id}/permissions/${permissionId}`, {}),

  // Revoke permission
  revokePermission: (id: string, permissionId: string) =>
    api.delete<Role>(`/roles/${id}/permissions/${permissionId}`),

  // List all permissions
  listPermissions: () => api.get<Permission[]>('/permissions'),

  // Check permission
  checkPermission: (resource: string, action: string) =>
    api.get<{ allowed: boolean }>(`/permissions/check?resource=${resource}&action=${action}`),
};
