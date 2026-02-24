import { api } from './client';
import type {
  SessionMetrics,
  UserActivity,
  CommandFrequency,
  CommandAnalysisParams,
  DashboardTrends,
  PaginatedResponse,
} from '@/types';

export interface SessionMetricsParams {
  start_date?: string;
  end_date?: string;
  granularity?: 'hour' | 'day' | 'week' | 'month';
}

export interface UserActivityParams {
  start_date?: string;
  end_date?: string;
  user_id?: string;
  limit?: number;
  offset?: number;
  include_heatmap?: boolean;
  search?: string;
}

export type { SessionMetrics, UserActivity, CommandFrequency, CommandAnalysisParams, DashboardTrends };

export const analyticsApi = {
  // Session Metrics
  getSessionMetrics: (params?: SessionMetricsParams) =>
    api.get<SessionMetrics>('/analytics/sessions/metrics', params),

  // Dashboard trends
  getDashboardTrends: (period?: 'day' | 'week' | 'month' | 'quarter') =>
    api.get<DashboardTrends>('/analytics/dashboard/trends', { period }),

  // User Activity
  getUserActivity: (params?: UserActivityParams) =>
    api.get<PaginatedResponse<UserActivity>>('/analytics/users/activity', params),

  getUserActivityDetail: (userId: string, params?: Omit<UserActivityParams, 'user_id'>) =>
    api.get<UserActivity>(`/analytics/users/${userId}/activity`, params),

  // Command Analysis
  getCommandFrequency: (params?: CommandAnalysisParams) =>
    api.get<PaginatedResponse<CommandFrequency>>('/analytics/commands/frequency', params),

  getCommandRiskSummary: (params?: Pick<CommandAnalysisParams, 'start_date' | 'end_date'>) =>
    api.get<{
      total_commands: number;
      high_risk_commands: number;
      medium_risk_commands: number;
      low_risk_commands: number;
      unique_users: number;
      unique_targets: number;
    }>('/analytics/commands/risk-summary', params),

  // Top N analytics
  getTopUsers: (limit?: number, params?: Pick<SessionMetricsParams, 'start_date' | 'end_date'>) =>
    api.get<Array<{
      user_id: string;
      user_name: string;
      user_email: string;
      session_count: number;
      total_duration_seconds: number;
    }>>('/analytics/users/top', { limit: limit || 10, ...params }),

  getTopTargets: (limit?: number, params?: Pick<SessionMetricsParams, 'start_date' | 'end_date'>) =>
    api.get<Array<{
      target_id: string;
      target_name: string;
      target_type: string;
      session_count: number;
      unique_users: number;
    }>>('/analytics/targets/top', { limit: limit || 10, ...params }),

  // Real-time stats
  getRealtimeStats: () =>
    api.get<{
      active_sessions: number;
      active_users: number;
      sessions_last_hour: number;
      avg_active_duration: number;
    }>('/analytics/realtime'),

  // Export analytics
  exportCommandAnalysis: (params: CommandAnalysisParams & { format: 'csv' | 'json' }) =>
    api.post<{ download_url: string; expires_at: string }>('/analytics/commands/export', params),

  exportUserActivity: (params: UserActivityParams & { format: 'csv' | 'json' }) =>
    api.post<{ download_url: string; expires_at: string }>('/analytics/users/export', params),
};
