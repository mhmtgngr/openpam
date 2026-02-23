import { api } from './client';
import type { DashboardStats, ActivityFeed, PaginatedResponse } from '@/types';

export const dashboardApi = {
  // Get dashboard statistics
  getStats: () => api.get<DashboardStats>('/dashboard/stats'),

  // Get activity feed
  getActivity: (params?: { limit?: number }) =>
    api.get<PaginatedResponse<ActivityFeed>>('/dashboard/activity', params),

  // Get pending requests for dashboard
  getPendingRequests: () => api.get<PaginatedResponse<unknown>>('/dashboard/pending-requests'),

  // Get active sessions for dashboard
  getActiveSessions: () => api.get<PaginatedResponse<unknown>>('/dashboard/active-sessions'),

  // Get expiring credentials
  getExpiringCredentials: (days?: number) =>
    api.get<PaginatedResponse<unknown>>('/dashboard/expiring-credentials', { days: days || 7 }),
};
