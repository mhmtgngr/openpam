import { analyticsApi, type SessionMetricsParams, type UserActivityParams } from './analytics';
import { api } from './client';

// Mock the api client
jest.mock('./client');

const mockGet = api.get as jest.MockedFunction<typeof api.get>;
const mockPost = api.post as jest.MockedFunction<typeof api.post>;

describe('analyticsApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Session Metrics', () => {
    it('getSessionMetrics should call the correct endpoint with default params', async () => {
      const mockResponse = {
        total_sessions: 100,
        active_sessions: 15,
        avg_duration: 300.5,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getSessionMetrics();

      expect(mockGet).toHaveBeenCalledWith('/analytics/sessions/metrics', undefined);
      expect(result).toEqual(mockResponse);
    });

    it('getSessionMetrics should pass params to the endpoint', async () => {
      const mockResponse = { total_sessions: 50 };
      const params: SessionMetricsParams = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        granularity: 'day',
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getSessionMetrics(params);

      expect(mockGet).toHaveBeenCalledWith('/analytics/sessions/metrics', params);
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Dashboard Trends', () => {
    it('getDashboardTrends should call the correct endpoint with default period', async () => {
      const mockResponse = {
        total_sessions: 1000,
        active_users: 50,
        commands_executed: 50000,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getDashboardTrends();

      expect(mockGet).toHaveBeenCalledWith('/analytics/dashboard/trends', { period: undefined });
      expect(result).toEqual(mockResponse);
    });

    it('getDashboardTrends should accept different periods', async () => {
      const mockResponse = { total_sessions: 100 };
      mockGet.mockResolvedValue(mockResponse);

      await analyticsApi.getDashboardTrends('week');
      expect(mockGet).toHaveBeenCalledWith('/analytics/dashboard/trends', { period: 'week' });

      await analyticsApi.getDashboardTrends('month');
      expect(mockGet).toHaveBeenCalledWith('/analytics/dashboard/trends', { period: 'month' });
    });
  });

  describe('User Activity', () => {
    it('getUserActivity should return paginated user activity', async () => {
      const mockResponse = {
        data: [
          {
            id: 'user-1',
            user_id: 'user-1',
            date: '2024-01-15',
            commands_executed: 150,
            sessions_initiated: 3,
          },
        ],
        total: 1,
        page: 1,
        per_page: 10,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getUserActivity();

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/activity', undefined);
      expect(result).toEqual(mockResponse);
    });

    it('getUserActivity should accept query params', async () => {
      const mockResponse = { data: [], total: 0, page: 1, per_page: 10 };
      const params: UserActivityParams = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        user_id: 'user-123',
        limit: 20,
        offset: 0,
        search: 'john',
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getUserActivity(params);

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/activity', params);
      expect(result).toEqual(mockResponse);
    });

    it('getUserActivityDetail should get activity for a specific user', async () => {
      const mockResponse = {
        id: 'activity-1',
        user_id: 'user-123',
        commands_executed: 500,
        sessions_initiated: 10,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getUserActivityDetail('user-123');

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/user-123/activity', undefined);
      expect(result).toEqual(mockResponse);
    });

    it('getUserActivityDetail should accept date params', async () => {
      const mockResponse = {
        id: 'activity-1',
        user_id: 'user-123',
        commands_executed: 200,
      };
      const params = { start_date: '2024-01-01', end_date: '2024-01-31' };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getUserActivityDetail('user-123', params);

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/user-123/activity', params);
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Command Analysis', () => {
    it('getCommandFrequency should return command frequency data', async () => {
      const mockResponse = {
        data: [
          {
            id: 1,
            base_command: 'ls',
            count: 1500,
            risk_level: 'low',
          },
          {
            id: 2,
            base_command: 'rm',
            count: 50,
            risk_level: 'high',
          },
        ],
        total: 2,
        page: 1,
        per_page: 100,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getCommandFrequency();

      expect(mockGet).toHaveBeenCalledWith('/analytics/commands/frequency', undefined);
      expect(result).toEqual(mockResponse);
    });

    it('getCommandRiskSummary should return risk breakdown', async () => {
      const mockResponse = {
        total_commands: 5000,
        high_risk_commands: 100,
        medium_risk_commands: 500,
        low_risk_commands: 4400,
        unique_users: 25,
        unique_targets: 50,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getCommandRiskSummary({
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });

      expect(mockGet).toHaveBeenCalledWith('/analytics/commands/risk-summary', {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Top N Analytics', () => {
    it('getTopUsers should return top users by activity', async () => {
      const mockResponse = [
        {
          user_id: 'user-1',
          user_name: 'John Doe',
          user_email: 'john@example.com',
          session_count: 50,
          total_duration_seconds: 15000,
        },
        {
          user_id: 'user-2',
          user_name: 'Jane Smith',
          user_email: 'jane@example.com',
          session_count: 35,
          total_duration_seconds: 10500,
        },
      ];
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getTopUsers();

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/top', { limit: 10 });
      expect(result).toEqual(mockResponse);
    });

    it('getTopUsers should accept custom limit', async () => {
      const mockResponse = [];
      mockGet.mockResolvedValue(mockResponse);

      await analyticsApi.getTopUsers(25);

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/top', { limit: 25 });
    });

    it('getTopUsers should accept date params', async () => {
      const mockResponse = [];
      mockGet.mockResolvedValue(mockResponse);

      await analyticsApi.getTopUsers(10, {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/top', {
        limit: 10,
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
    });

    it('getTopTargets should return top targets', async () => {
      const mockResponse = [
        {
          target_id: 'target-1',
          target_name: 'production-server',
          target_type: 'ssh',
          session_count: 100,
          unique_users: 15,
        },
      ];
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getTopTargets();

      expect(mockGet).toHaveBeenCalledWith('/analytics/targets/top', { limit: 10 });
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Real-time Stats', () => {
    it('getRealtimeStats should return current statistics', async () => {
      const mockResponse = {
        active_sessions: 15,
        active_users: 12,
        sessions_last_hour: 25,
        avg_active_duration: 450.5,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getRealtimeStats();

      expect(mockGet).toHaveBeenCalledWith('/analytics/realtime', undefined);
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Export Functions', () => {
    it('exportCommandAnalysis should request CSV export', async () => {
      const mockResponse = {
        download_url: 'https://storage.example.com/exports/commands-123.csv',
        expires_at: '2024-01-16T00:00:00Z',
      };
      mockPost.mockResolvedValue(mockResponse);

      const result = await analyticsApi.exportCommandAnalysis({
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        format: 'csv',
      });

      expect(mockPost).toHaveBeenCalledWith('/analytics/commands/export', {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        format: 'csv',
      });
      expect(result).toEqual(mockResponse);
    });

    it('exportUserActivity should request JSON export', async () => {
      const mockResponse = {
        download_url: 'https://storage.example.com/exports/activity-456.json',
        expires_at: '2024-01-16T00:00:00Z',
      };
      mockPost.mockResolvedValue(mockResponse);

      const result = await analyticsApi.exportUserActivity({
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        format: 'json',
      });

      expect(mockPost).toHaveBeenCalledWith('/analytics/users/export', {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        format: 'json',
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe('Type Exports', () => {
    it('should export SessionMetrics type', () => {
      const params: SessionMetricsParams = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        granularity: 'day',
      };
      expect(params.start_date).toBe('2024-01-01');
      expect(params.granularity).toBe('day');
    });

    it('should export UserActivityParams type', () => {
      const params: UserActivityParams = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
        user_id: 'user-123',
        limit: 20,
        offset: 0,
        include_heatmap: true,
        search: 'john',
      };
      expect(params.user_id).toBe('user-123');
      expect(params.limit).toBe(20);
      expect(params.include_heatmap).toBe(true);
    });
  });

  describe('Error Handling', () => {
    it('getSessionMetrics should handle API errors', async () => {
      const mockError = new Error('API Error');
      mockGet.mockRejectedValue(mockError);

      await expect(analyticsApi.getSessionMetrics()).rejects.toThrow('API Error');
    });

    it('getUserActivity should handle API errors', async () => {
      const mockError = new Error('Network error');
      mockGet.mockRejectedValue(mockError);

      await expect(analyticsApi.getUserActivity()).rejects.toThrow('Network error');
    });

    it('exportCommandAnalysis should handle export errors', async () => {
      const mockError = new Error('Export failed');
      mockPost.mockRejectedValue(mockError);

      await expect(
        analyticsApi.exportCommandAnalysis({
          start_date: '2024-01-01',
          end_date: '2024-01-31',
          format: 'csv',
        })
      ).rejects.toThrow('Export failed');
    });
  });

  describe('Request Validation', () => {
    it('should handle empty response from getDashboardTrends', async () => {
      const mockResponse = {
        total_sessions: 0,
        active_users: 0,
        commands_executed: 0,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getDashboardTrends();

      expect(result).toBeDefined();
      expect(result.total_sessions).toBe(0);
    });

    it('should handle empty array from getTopUsers', async () => {
      mockGet.mockResolvedValue([]);

      const result = await analyticsApi.getTopUsers();

      expect(result).toEqual([]);
      expect(mockGet).toHaveBeenCalledWith('/analytics/users/top', { limit: 10 });
    });

    it('should handle pagination params correctly', async () => {
      const mockResponse = {
        data: [],
        total: 0,
        page: 2,
        per_page: 20,
      };
      const params: UserActivityParams = {
        limit: 20,
        offset: 20,
      };
      mockGet.mockResolvedValue(mockResponse);

      const result = await analyticsApi.getUserActivity(params);

      expect(mockGet).toHaveBeenCalledWith('/analytics/users/activity', params);
      expect(result).toBeDefined();
    });
  });
});
