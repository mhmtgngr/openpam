/**
 * Tests for analytics API client
 */

import { vi } from 'vitest';
import { analyticsApi } from './analytics';
import { api } from './client';
import type {
  SessionMetrics,
  UserActivity,
  CommandFrequency,
  DashboardTrends,
} from '@/types';

// Mock the API client module
vi.mock('./client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

describe('analyticsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Session Metrics', () => {
    it('should get session metrics with default params', async () => {
      const mockMetrics: SessionMetrics = {
        active_sessions: 5,
        peak_concurrent_sessions: 12,
        total_sessions_today: 23,
        total_sessions_week: 161,
        avg_session_duration_seconds: 1800,
        total_session_duration_today_seconds: 41400,
        sessions_by_type: { ssh: 15, rdp: 8 },
        sessions_by_environment: { production: 15, staging: 8 },
        sessions_over_time: [],
      };

      (api.get as any).mockResolvedValue({ data: mockMetrics });

      const response = await analyticsApi.getSessionMetrics();
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/sessions/metrics', undefined);
      expect(result).toEqual(mockMetrics);
    });

    it('should get session metrics with custom params', async () => {
      const mockMetrics: SessionMetrics = {
        active_sessions: 10,
        peak_concurrent_sessions: 20,
        total_sessions_today: 50,
        total_sessions_week: 350,
        avg_session_duration_seconds: 2400,
        total_session_duration_today_seconds: 120000,
        sessions_by_type: { ssh: 30, rdp: 20 },
        sessions_by_environment: { production: 30, staging: 15, development: 5 },
        sessions_over_time: [],
      };

      (api.get as any).mockResolvedValue({ data: mockMetrics });

      const params = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        granularity: 'day' as const,
      };

      const response = await analyticsApi.getSessionMetrics(params);
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/sessions/metrics', params);
      expect(result).toEqual(mockMetrics);
    });

    it('should handle API errors gracefully', async () => {
      (api.get as any).mockRejectedValue(new Error('Network error'));

      await expect(analyticsApi.getSessionMetrics()).rejects.toThrow('Network error');
    });
  });

  describe('Dashboard Trends', () => {
    it('should get dashboard trends with default period', async () => {
      const mockTrends: DashboardTrends = {
        sessions: { current: 145, previous: 130, change_percent: 11.5, trend: 'up', data_points: [] },
        users: { current: 45, previous: 42, change_percent: 7.1, trend: 'up', data_points: [] },
        credentials: { current: 230, previous: 225, change_percent: 2.2, trend: 'up', data_points: [] },
        requests: { current: 18, previous: 22, change_percent: -18.2, trend: 'down', data_points: [] },
        period: 'week',
      };

      (api.get as any).mockResolvedValue({ data: mockTrends });

      const response = await analyticsApi.getDashboardTrends();
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/dashboard/trends', { period: undefined });
      expect(result).toEqual(mockTrends);
    });

    it('should get dashboard trends with custom period', async () => {
      const mockTrends: DashboardTrends = {
        sessions: { current: 500, previous: 450, change_percent: 11.1, trend: 'up', data_points: [] },
        users: { current: 120, previous: 100, change_percent: 20, trend: 'up', data_points: [] },
        credentials: { current: 600, previous: 580, change_percent: 3.4, trend: 'up', data_points: [] },
        requests: { current: 50, previous: 55, change_percent: -9.1, trend: 'down', data_points: [] },
        period: 'month',
      };

      (api.get as any).mockResolvedValue({ data: mockTrends });

      const response = await analyticsApi.getDashboardTrends('month');
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/dashboard/trends', { period: 'month' });
      expect(result).toEqual(mockTrends);
    });

    it('should support all period types', async () => {
      const periods: Array<'day' | 'week' | 'month' | 'quarter'> = ['day', 'week', 'month', 'quarter'];

      for (const period of periods) {
        (api.get as any).mockResolvedValue({
          data: {
            sessions: { current: 1, previous: 1, change_percent: 0, trend: 'up' as const, data_points: [] },
            users: { current: 1, previous: 1, change_percent: 0, trend: 'up' as const, data_points: [] },
            credentials: { current: 1, previous: 1, change_percent: 0, trend: 'up' as const, data_points: [] },
            requests: { current: 1, previous: 1, change_percent: 0, trend: 'up' as const, data_points: [] },
            period,
          },
        });

        await analyticsApi.getDashboardTrends(period);

        expect(api.get).toHaveBeenCalledWith('/analytics/dashboard/trends', { period });
      }
    });
  });

  describe('Dashboard Data', () => {
    it('should get combined dashboard data', async () => {
      const mockMetrics: SessionMetrics = {
        active_sessions: 8,
        peak_concurrent_sessions: 15,
        total_sessions_today: 30,
        total_sessions_week: 210,
        avg_session_duration_seconds: 2100,
        total_session_duration_today_seconds: 63000,
        sessions_by_type: { ssh: 20, rdp: 10 },
        sessions_by_environment: { production: 20, staging: 10 },
        sessions_over_time: [],
      };

      const mockTrends: DashboardTrends = {
        sessions: { current: 100, previous: 90, change_percent: 11.1, trend: 'up', data_points: [] },
        users: { current: 30, previous: 25, change_percent: 20, trend: 'up', data_points: [] },
        credentials: { current: 150, previous: 140, change_percent: 7.1, trend: 'up', data_points: [] },
        requests: { current: 15, previous: 18, change_percent: -16.7, trend: 'down', data_points: [] },
        period: 'week',
      };

      const mockRealtime = {
        active_sessions: 5,
        active_users: 3,
        sessions_last_hour: 8,
        avg_active_duration: 1200,
      };

      (api.get as any).mockImplementation((endpoint: string) => {
        if (endpoint === '/analytics/dashboard') {
          return Promise.resolve({
            data: {
              metrics: mockMetrics,
              trends: mockTrends,
              realtime: mockRealtime,
            },
          });
        }
        return Promise.reject(new Error('Unknown endpoint'));
      });

      const result = await analyticsApi.getDashboard();
      const data = result.data;

      expect(data).toEqual({
        metrics: mockMetrics,
        trends: mockTrends,
        realtime: mockRealtime,
      });
    });

    it('should pass date params to dashboard endpoint', async () => {
      (api.get as any).mockResolvedValue({
        data: {
          metrics: {
            active_sessions: 1,
            peak_concurrent_sessions: 1,
            total_sessions_today: 1,
            total_sessions_week: 7,
            avg_session_duration_seconds: 1,
            total_session_duration_today_seconds: 1,
            sessions_by_type: {},
            sessions_by_environment: {},
            sessions_over_time: [],
          },
          trends: {
            sessions: { current: 1, previous: 1, change_percent: 0, trend: 'up', data_points: [] },
            users: { current: 1, previous: 1, change_percent: 0, trend: 'up', data_points: [] },
            credentials: { current: 1, previous: 1, change_percent: 0, trend: 'up', data_points: [] },
            requests: { current: 1, previous: 1, change_percent: 0, trend: 'up', data_points: [] },
            period: 'week',
          },
          realtime: { active_sessions: 1, active_users: 1, sessions_last_hour: 1, avg_active_duration: 1 },
        },
      });

      await analyticsApi.getDashboard({ start_date: '2024-01-01', end_date: '2024-01-31' });

      expect(api.get).toHaveBeenCalledWith('/analytics/dashboard', {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
    });
  });

  describe('Time Series Data', () => {
    it('should get time series data', async () => {
      const mockTimeSeries = [
        { timestamp: '2024-01-01T00:00:00Z', value: 100 },
        { timestamp: '2024-01-02T00:00:00Z', value: 120 },
        { timestamp: '2024-01-03T00:00:00Z', value: 115 },
      ];

      (api.get as any).mockResolvedValue({ data: mockTimeSeries });

      const response = await analyticsApi.getTimeSeries({
        metric: 'sessions',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/timeseries', {
        metric: 'sessions',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      expect(result).toEqual(mockTimeSeries);
    });
  });

  describe('User Activity', () => {
    it('should get user activity list', async () => {
      const mockActivity: UserActivity[] = [
        {
          user_id: '1',
          user_email: 'admin@example.com',
          user_name: 'Admin User',
          total_sessions: 15,
          total_duration_seconds: 27000,
          avg_session_duration_seconds: 1800,
          last_activity_at: '2024-01-15T10:30:00Z',
          most_used_targets: ['prod-server-01', 'staging-db-01'],
          activity_heatmap: [],
        },
        {
          user_id: '2',
          user_email: 'user@example.com',
          user_name: 'Regular User',
          total_sessions: 8,
          total_duration_seconds: 14400,
          avg_session_duration_seconds: 1800,
          last_activity_at: '2024-01-15T09:45:00Z',
          most_used_targets: ['dev-server-01'],
          activity_heatmap: [],
        },
      ];

      (api.get as any).mockResolvedValue({
        data: mockActivity,
        pagination: { total: 2, limit: 20, offset: 0, has_more: false },
      });

      const result = await analyticsApi.getUserActivity();

      expect(api.get).toHaveBeenCalledWith('/analytics/users/activity', undefined);
      expect(result.data).toEqual(mockActivity);
      expect(result.pagination.total).toBe(2);
    });

    it('should get user activity with filters', async () => {
      const mockActivity: UserActivity[] = [];

      (api.get as any).mockResolvedValue({
        data: mockActivity,
        pagination: { total: 0, limit: 10, offset: 0, has_more: false },
      });

      const params = {
        user_id: 'user-123',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        limit: 10,
        offset: 0,
        search: 'admin',
        include_heatmap: true,
      };

      const result = await analyticsApi.getUserActivity(params);

      expect(api.get).toHaveBeenCalledWith('/analytics/users/activity', params);
      expect(result.data).toEqual(mockActivity);
    });

    it('should get user activity detail', async () => {
      const mockDetail: UserActivity = {
        user_id: 'user-123',
        user_email: 'admin@example.com',
        user_name: 'Admin User',
        total_sessions: 50,
        total_duration_seconds: 90000,
        avg_session_duration_seconds: 1800,
        last_activity_at: '2024-01-15T10:30:00Z',
        most_used_targets: ['server-1', 'server-2', 'server-3'],
        activity_heatmap: [
          { date: '2024-01-15', hour: 9, session_count: 5 },
          { date: '2024-01-15', hour: 14, session_count: 3 },
        ],
      };

      (api.get as any).mockResolvedValue({ data: mockDetail });

      const response = await analyticsApi.getUserActivityDetail('user-123');
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/users/user-123/activity', undefined);
      expect(result).toEqual(mockDetail);
    });
  });

  describe('Command Analysis', () => {
    it('should get command frequency', async () => {
      const mockCommands: CommandFrequency[] = [
        {
          command: 'ls -la',
          risk_level: 'low',
          count: 1500,
          first_seen_at: '2024-01-01T00:00:00Z',
          last_seen_at: '2024-01-15T23:59:59Z',
          users: [],
          targets: [],
        },
        {
          command: 'sudo su',
          risk_level: 'high',
          count: 45,
          first_seen_at: '2024-01-01T00:00:00Z',
          last_seen_at: '2024-01-15T18:30:00Z',
          users: [],
          targets: [],
        },
      ];

      (api.get as any).mockResolvedValue({
        data: mockCommands,
        pagination: { total: 2, limit: 20, offset: 0, has_more: false },
      });

      const result = await analyticsApi.getCommandFrequency();

      expect(api.get).toHaveBeenCalledWith('/analytics/commands/frequency', undefined);
      expect(result.data).toHaveLength(2);
    });

    it('should get command risk summary', async () => {
      const mockSummary = {
        total_commands: 3500,
        high_risk_commands: 150,
        medium_risk_commands: 450,
        low_risk_commands: 2900,
        unique_users: 25,
        unique_targets: 12,
      };

      (api.get as any).mockResolvedValue({ data: mockSummary });

      const response = await analyticsApi.getCommandRiskSummary();
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/commands/risk-summary', undefined);
      expect(result.total_commands).toBe(3500);
      expect(result.high_risk_commands).toBe(150);
    });

    it('should export command analysis', async () => {
      const mockExport = {
        download_url: 'https://example.com/exports/commands-2024-01-15.csv',
        expires_at: '2024-01-16T00:00:00Z',
      };

      (api.post as any).mockResolvedValue({ data: mockExport });

      const response = await analyticsApi.exportCommandAnalysis({
        format: 'csv',
        start_date: '2024-01-01',
        end_date: '2024-01-15',
      });
      const result = response.data;

      expect(api.post).toHaveBeenCalledWith('/analytics/commands/export', {
        format: 'csv',
        start_date: '2024-01-01',
        end_date: '2024-01-15',
      });
      expect(result.download_url).toBe(mockExport.download_url);
    });
  });

  describe('Top N Analytics', () => {
    it('should get top users', async () => {
      const mockTopUsers = [
        {
          user_id: 'user-1',
          user_name: 'Admin User',
          user_email: 'admin@example.com',
          session_count: 50,
          total_duration_seconds: 90000,
        },
        {
          user_id: 'user-2',
          user_name: 'Regular User',
          user_email: 'user@example.com',
          session_count: 30,
          total_duration_seconds: 54000,
        },
      ];

      (api.get as any).mockResolvedValue({ data: mockTopUsers });

      const response = await analyticsApi.getTopUsers(10);
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/users/top', {
        limit: 10,
      });
      expect(result).toHaveLength(2);
    });

    it('should get top targets', async () => {
      const mockTopTargets = [
        {
          target_id: 'target-1',
          target_name: 'prod-server-01',
          target_type: 'ssh',
          session_count: 100,
          unique_users: 15,
        },
        {
          target_id: 'target-2',
          target_name: 'staging-db-01',
          target_type: 'database',
          session_count: 75,
          unique_users: 12,
        },
      ];

      (api.get as any).mockResolvedValue({ data: mockTopTargets });

      const response = await analyticsApi.getTopTargets(20, {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/targets/top', {
        limit: 20,
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      expect(result).toHaveLength(2);
    });
  });

  describe('Real-time Stats', () => {
    it('should get real-time statistics', async () => {
      const mockStats = {
        active_sessions: 8,
        active_users: 5,
        sessions_last_hour: 12,
        avg_active_duration: 1500,
      };

      (api.get as any).mockResolvedValue({ data: mockStats });

      const response = await analyticsApi.getRealtimeStats();
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/realtime');
      expect(result.active_sessions).toBe(8);
      expect(result.active_users).toBe(5);
    });
  });

  describe('Anomalies', () => {
    it('should list anomalies', async () => {
      const mockAnomalies = [
        {
          id: 'anomaly-1',
          type: 'unusual_access_time',
          severity: 'critical',
          title: 'After-Hours Access',
          description: 'User accessed production at 3 AM',
          detected_at: '2024-01-15T03:00:00Z',
          user_id: 'user-1',
          user_name: 'Admin User',
          target_id: 'target-1',
          target_name: 'prod-server-01',
          session_id: 'session-1',
          confidence_score: 0.92,
          indicators: [],
          status: 'open',
        },
      ];

      (api.get as any).mockResolvedValue({
        data: mockAnomalies,
        pagination: { total: 1, limit: 20, offset: 0, has_more: false },
      });

      const result = await analyticsApi.listAnomalies({
        severity: 'critical',
        status: 'open',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });

      expect(api.get).toHaveBeenCalledWith('/analytics/anomalies', {
        severity: 'critical',
        status: 'open',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      expect(result.data).toHaveLength(1);
    });

    it('should update anomaly status', async () => {
      const mockResponse = {
        id: 'anomaly-1',
        status: 'resolved',
      };

      (api.patch as any).mockResolvedValue({ data: mockResponse });

      const response = await analyticsApi.updateAnomalyStatus('anomaly-1', {
        status: 'resolved',
        notes: 'Investigated and confirmed as legitimate',
      });
      const result = response.data;

      expect((api.patch as any).mock.calls[0][0]).toContain('/analytics/anomalies/anomaly-1');
      expect(result.status).toBe('resolved');
    });
  });

  describe('Compliance Summary', () => {
    it('should get compliance summary', async () => {
      const mockSummary = {
        overall_score: 85,
        control_count: 50,
        compliant_count: 42,
        non_compliant_count: 8,
        last_assessed: '2024-01-15T10:00:00Z',
      };

      (api.get as any).mockResolvedValue({ data: mockSummary });

      const response = await analyticsApi.getComplianceSummary('SOC2');
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/compliance/summary', {
        framework: 'SOC2',
      });
      expect(result.overall_score).toBe(85);
    });

    it('should get compliance summary without framework', async () => {
      const mockSummary = {
        overall_score: 75,
        control_count: 100,
        compliant_count: 75,
        non_compliant_count: 25,
        last_assessed: '2024-01-10T10:00:00Z',
      };

      (api.get as any).mockResolvedValue({ data: mockSummary });

      const response = await analyticsApi.getComplianceSummary();
      const result = response.data;

      expect(api.get).toHaveBeenCalledWith('/analytics/compliance/summary', {
        framework: undefined,
      });
      expect(result.overall_score).toBe(75);
    });
  });

  describe('Error Handling', () => {
    it('should handle 401 unauthorized responses', async () => {
      const error = new Error('Unauthorized');
      error.name = '401';
      (api.get as any).mockRejectedValue(error);

      await expect(analyticsApi.getSessionMetrics()).rejects.toThrow();
    });

    it('should handle 404 not found responses', async () => {
      const error = new Error('Not Found');
      error.name = '404';
      (api.get as any).mockRejectedValue(error);

      await expect(analyticsApi.getUserActivityDetail('non-existent')).rejects.toThrow();
    });

    it('should handle 500 server errors', async () => {
      const error = new Error('Internal Server Error');
      error.name = '500';
      (api.get as any).mockRejectedValue(error);

      await expect(analyticsApi.getDashboardTrends()).rejects.toThrow();
    });

    it('should handle network timeouts', async () => {
      const error = new Error('Request timeout');
      error.name = 'TIMEOUT';
      (api.get as any).mockRejectedValue(error);

      await expect(analyticsApi.getRealtimeStats()).rejects.toThrow();
    });
  });

  describe('Request Parameters', () => {
    it('should properly encode date parameters', async () => {
      (api.get as any).mockResolvedValue({
        data: {
          active_sessions: 1,
          peak_concurrent_sessions: 1,
          total_sessions_today: 1,
          total_sessions_week: 7,
          avg_session_duration_seconds: 1,
          total_session_duration_today_seconds: 1,
          sessions_by_type: {},
          sessions_by_environment: {},
          sessions_over_time: [],
        },
      });

      await analyticsApi.getSessionMetrics({
        start_date: '2024-01-01T00:00:00Z',
        end_date: '2024-01-31T23:59:59Z',
        granularity: 'hour',
      });

      expect(api.get).toHaveBeenCalledWith('/analytics/sessions/metrics', {
        start_date: '2024-01-01T00:00:00Z',
        end_date: '2024-01-31T23:59:59Z',
        granularity: 'hour',
      });
    });

    it('should handle pagination parameters correctly', async () => {
      (api.get as any).mockResolvedValue({
        data: [],
        pagination: { total: 0, limit: 50, offset: 100, has_more: false },
      });

      await analyticsApi.getUserActivity({
        limit: 50,
        offset: 100,
      });

      expect(api.get).toHaveBeenCalledWith('/analytics/users/activity', {
        limit: 50,
        offset: 100,
      });
    });

    it('should handle boolean parameters', async () => {
      (api.get as any).mockResolvedValue({
        data: [],
        pagination: { total: 0, limit: 20, offset: 0, has_more: false },
      });

      await analyticsApi.getUserActivity({
        include_heatmap: true,
      });

      expect(api.get).toHaveBeenCalledWith('/analytics/users/activity', {
        include_heatmap: true,
      });
    });
  });

  describe('Type Safety', () => {
    it('should return typed SessionMetrics', async () => {
      const mockMetrics: SessionMetrics = {
        active_sessions: 5,
        peak_concurrent_sessions: 10,
        total_sessions_today: 20,
        total_sessions_week: 140,
        avg_session_duration_seconds: 1800,
        total_session_duration_today_seconds: 36000,
        sessions_by_type: { ssh: 12, rdp: 8 },
        sessions_by_environment: { production: 15, staging: 5 },
        sessions_over_time: [],
      };

      (api.get as any).mockResolvedValue({ data: mockMetrics });

      const response = await analyticsApi.getSessionMetrics();
      const result = response.data;

      // Type check - should have SessionMetrics properties
      expect(result).toHaveProperty('active_sessions');
      expect(result).toHaveProperty('sessions_by_type');
      expect(typeof result.active_sessions).toBe('number');
    });

    it('should return typed UserActivity array', async () => {
      const mockActivity: UserActivity[] = [
        {
          user_id: '1',
          user_email: 'user@example.com',
          user_name: 'Test User',
          total_sessions: 10,
          total_duration_seconds: 18000,
          avg_session_duration_seconds: 1800,
          last_activity_at: '2024-01-15T10:00:00Z',
          most_used_targets: [],
          activity_heatmap: [],
        },
      ];

      (api.get as any).mockResolvedValue({
        data: mockActivity,
        pagination: { total: 1, limit: 20, offset: 0, has_more: false },
      });

      const result = await analyticsApi.getUserActivity();

      // Type check
      expect(Array.isArray(result.data)).toBe(true);
      if (result.data.length > 0) {
        expect(result.data[0]).toHaveProperty('user_id');
        expect(result.data[0]).toHaveProperty('user_email');
      }
    });
  });

  describe('Export Functions', () => {
    it('should export user activity data', async () => {
      const mockExport = {
        download_url: 'https://example.com/exports/user-activity-2024.csv',
        expires_at: '2024-01-16T00:00:00Z',
      };

      (api.post as any).mockResolvedValue({ data: mockExport });

      const response = await analyticsApi.exportUserActivity({
        format: 'csv',
        start_date: '2024-01-01',
        end_date: '2024-01-15',
        limit: 1000,
      });
      const result = response.data;

      expect(api.post).toHaveBeenCalledWith('/analytics/users/export', {
        format: 'csv',
        start_date: '2024-01-01',
        end_date: '2024-01-15',
        limit: 1000,
      });
      expect(result.download_url).toContain('user-activity');
    });

    it('should support JSON export format', async () => {
      const mockExport = {
        download_url: 'https://example.com/exports/commands.json',
        expires_at: '2024-01-16T12:00:00Z',
      };

      (api.post as any).mockResolvedValue({ data: mockExport });

      const response = await analyticsApi.exportCommandAnalysis({
        format: 'json',
      });
      const result = response.data;

      expect((api.post as any).mock.calls[0][1]).toHaveProperty('format', 'json');
      expect(result.download_url).toContain('.json');
    });
  });
});
