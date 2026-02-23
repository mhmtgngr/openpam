import { api } from './client';
import type { User, PaginatedResponse } from '@/types';

export interface UserListParams {
  limit?: number;
  offset?: number;
  search?: string;
  role?: string;
  status?: string;
}

export interface CreateUserData {
  email: string;
  first_name: string;
  last_name: string;
  role: string;
  password?: string;
  send_invite?: boolean;
}

export interface UpdateUserData {
  first_name?: string;
  last_name?: string;
  role?: string;
  status?: string;
}

export const usersApi = {
  // List users
  list: (params?: UserListParams) =>
    api.get<PaginatedResponse<User>>('/users', params),

  // Get user by ID
  get: (id: string) => api.get<User>(`/users/${id}`),

  // Create user
  create: (data: CreateUserData) => api.post<User>('/users', data),

  // Update user
  update: (id: string, data: UpdateUserData) => api.patch<User>(`/users/${id}`, data),

  // Delete user
  delete: (id: string) => api.delete<void>(`/users/${id}`),

  // Assign role
  assignRole: (id: string, roleId: string) =>
    api.post<User>(`/users/${id}/roles/${roleId}`, {}),

  // Revoke role
  revokeRole: (id: string, roleId: string) =>
    api.delete<User>(`/users/${id}/roles/${roleId}`),

  // Reset password
  resetPassword: (id: string) => api.post<{ temp_password: string }>(`/users/${id}/password/reset`, {}),

  // Force password change
  forcePasswordChange: (id: string) => api.post<void>(`/users/${id}/password/force-change`, {}),

  // Unlock user
  unlock: (id: string) => api.post<void>(`/users/${id}/unlock`, {}),

  // Disable MFA for user (admin only)
  disableMFA: (id: string) => api.post<void>(`/users/${id}/mfa/disable`, {}),

  // Get user activity
  getActivity: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<unknown>>(`/users/${id}/activity`, params),
};
