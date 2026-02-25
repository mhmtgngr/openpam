/**
 * Tests for anomaly API client
 */

import { vi, describe, beforeEach, it, expect } from 'vitest';
import { anomalyApi } from './anomaly';
import { api } from './client';
import type {
  AnomalyDetection,
  AnomalySeverity,
  AnomalyType,
  PaginatedResponse,
} from '@/types';

// Mock the API client module
vi.mock('./client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}));

// Mock data
const mockAnomalies: AnomalyDetection[] = [
  {
    id: 'anom-1',
    type: 'unusual_access_time' as AnomalyType,
    severity: 'critical' as AnomalySeverity,
    title: 'Unusual After-Hours Access',
    description: 'User accessed production system at 3 AM',
    detected_at: new Date().toISOString(),
    user_id: 'user-1',
    user_name: 'Admin User',
    target_id: 'target-1',
    target_name: 'prod-server-01',
    session_id: 'sess-1',
    confidence_score: 0.92,
    risk_score: 95,
    status: 'open',
    assigned_to: '',
    resolution_notes: '',
    indicators: [
      {
        type: 'time_anomaly',
        description: 'Access time outside normal hours',
        value: '03:00',
        threshold: '06:00-22:00' as string | number,
        confidence: 0.92,
      },
    ],
  },
  {
    id: 'anom-2',
    type: 'impossible_travel' as AnomalyType,
    severity: 'high' as AnomalySeverity,
    title: 'Impossible Travel Detected',
    description: 'User logged in from two locations 1000 miles apart within 5 minutes',
    detected_at: new Date(Date.now() - 3600000).toISOString(),
    user_id: 'user-2',
    user_name: 'Jane Smith',
    target_id: 'target-2',
    target_name: 'db-server-02',
    session_id: 'sess-2',
    confidence_score: 0.88,
    risk_score: 85,
    status: 'investigating',
    assigned_to: 'admin@example.com',
    resolution_notes: 'Under investigation',
    indicators: [
      {
        type: 'geo_anomaly',
        description: 'Login from New York, then London 5 minutes later',
        value: 1000,
        threshold: 500,
        confidence: 0.88,
      },
    ],
  },
  {
    id: 'anom-3',
    type: 'excessive_failed_logins' as AnomalyType,
    severity: 'medium' as AnomalySeverity,
    title: 'Excessive Failed Login Attempts',
    description: 'User had 15 failed login attempts within 5 minutes',
    detected_at: new Date(Date.now() - 7200000).toISOString(),
    user_id: 'user-3',
    user_name: 'Bob Johnson',
    target_id: 'target-3',
    target_name: 'auth-server',
    session_id: undefined,
    confidence_score: 0.75,
    risk_score: 60,
    status: 'resolved',
    assigned_to: 'security@example.com',
    resolution_notes: 'User reset password, account secured',
    indicators: [
      {
        type: 'brute_force_indicators',
        description: 'Multiple failed attempts from different IPs',
        value: 15,
        threshold: 10,
        confidence: 0.75,
      },
    ],
  },
];

const mockAnomalySummary = {
  total: 3,
  by_severity: {
    critical: 1,
    high: 1,
    medium: 1,
    low: 0,
  },
  by_type: {
    unusual_access_time: 1,
    impossible_travel: 1,
    excessive_failed_logins: 1,
  },
  by_status: {
    open: 1,
    investigating: 1,
    resolved: 1,
    false_positive: 0,
  },
  resolved_this_period: 1,
  avg_resolution_time_hours: 4.5,
  critical_open: 1,
  high_open: 0,
};

describe('anomalyApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('list', () => {
    it('should list anomalies with default params', async () => {
      const mockResponse: PaginatedResponse<AnomalyDetection> = {
        data: mockAnomalies,
        pagination: {
          total: 3,
          offset: 0,
          limit: 20,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(mockResponse);

      const result = await anomalyApi.list();

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies', undefined);
      expect(result).toEqual(mockResponse);
    });

    it('should list anomalies with filters', async () => {
      const mockResponse: PaginatedResponse<AnomalyDetection> = {
        data: [mockAnomalies[0]],
        pagination: {
          total: 1,
          offset: 0,
          limit: 10,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(mockResponse);

      const filters = {
        severity: 'critical' as AnomalySeverity,
        status: 'open',
        limit: 10,
        offset: 0,
      };

      const result = await anomalyApi.list(filters);

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies', filters);
      expect(result.data).toHaveLength(1);
      expect(result.data[0].severity).toBe('critical');
    });

    it('should handle empty results', async () => {
      const mockResponse: PaginatedResponse<AnomalyDetection> = {
        data: [],
        pagination: {
          total: 0,
          offset: 0,
          limit: 20,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(mockResponse);

      const result = await anomalyApi.list();

      expect(result.data).toEqual([]);
      expect(result.pagination.total).toBe(0);
    });

    it('should handle API errors', async () => {
      (api.get as any).mockRejectedValue(new Error('Network error'));

      await expect(anomalyApi.list()).rejects.toThrow('Network error');
    });
  });

  describe('get', () => {
    it('should get a single anomaly by ID', async () => {
      (api.get as any).mockResolvedValue(mockAnomalies[0]);

      const result = await anomalyApi.get('anom-1');

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/anom-1');
      expect(result).toEqual(mockAnomalies[0]);
    });

    it('should handle not found error', async () => {
      (api.get as any).mockRejectedValue({
        response: { status: 404 },
        message: 'Anomaly not found',
      });

      await expect(anomalyApi.get('non-existent')).rejects.toThrow();
    });
  });

  describe('update', () => {
    it('should update anomaly status', async () => {
      const updateData = {
        status: 'investigating' as const,
        assigned_to: 'admin@example.com',
        resolution_notes: 'Starting investigation',
      };

      const updatedAnomaly = {
        ...mockAnomalies[0],
        ...updateData,
      };

      (api.patch as any).mockResolvedValue(updatedAnomaly);

      const result = await anomalyApi.update('anom-1', updateData);

      expect(api.patch).toHaveBeenCalledWith('/compliance/anomalies/anom-1', updateData);
      expect(result.status).toBe('investigating');
      expect(result.assigned_to).toBe('admin@example.com');
    });

    it('should update only provided fields', async () => {
      const updateData = {
        resolution_notes: 'Additional notes',
      };

      const updatedAnomaly = {
        ...mockAnomalies[0],
        ...updateData,
      };

      (api.patch as any).mockResolvedValue(updatedAnomaly);

      const result = await anomalyApi.update('anom-1', updateData);

      expect(api.patch).toHaveBeenCalledWith('/compliance/anomalies/anom-1', updateData);
      expect(result.resolution_notes).toBe('Additional notes');
      expect(result.status).toBe('open'); // Unchanged
    });
  });

  describe('acknowledge', () => {
    it('should acknowledge an anomaly', async () => {
      const acknowledgedAnomaly = {
        ...mockAnomalies[0],
        status: 'investigating',
      };

      (api.post as any).mockResolvedValue(acknowledgedAnomaly);

      const result = await anomalyApi.acknowledge('anom-1');

      expect(api.post).toHaveBeenCalledWith('/compliance/anomalies/anom-1/acknowledge', {});
      expect(result.status).toBe('investigating');
    });

    it('should send acknowledgment with notes', async () => {
      const acknowledgedAnomaly = {
        ...mockAnomalies[0],
        status: 'investigating',
      };

      (api.post as any).mockResolvedValue(acknowledgedAnomaly);

      await anomalyApi.acknowledge('anom-1');

      expect(api.post).toHaveBeenCalled();
    });
  });

  describe('getSummary', () => {
    it('should get anomaly summary statistics', async () => {
      (api.get as any).mockResolvedValue(mockAnomalySummary);

      const result = await anomalyApi.getSummary();

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/summary', undefined);
      expect(result).toEqual(mockAnomalySummary);
      expect(result.total).toBe(3);
      expect(result.by_severity.critical).toBe(1);
    });

    it('should get summary with date filters', async () => {
      (api.get as any).mockResolvedValue(mockAnomalySummary);

      const params = {
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      };

      await anomalyApi.getSummary(params);

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/summary', params);
    });
  });

  describe('bulkUpdate', () => {
    it('should bulk update anomalies', async () => {
      const updateData = {
        status: 'resolved' as const,
        resolution_notes: 'Bulk resolved',
      };

      const bulkResult = {
        updated: 2,
        failed: [] as string[],
      };

      (api.post as any).mockResolvedValue(bulkResult);

      const result = await anomalyApi.bulkUpdate(['anom-1', 'anom-2'], updateData);

      expect(api.post).toHaveBeenCalledWith('/compliance/anomalies/bulk-update', {
        ids: ['anom-1', 'anom-2'],
        ...updateData,
      });
      expect(result.updated).toBe(2);
      expect(result.failed).toEqual([]);
    });

    it('should handle partial failures in bulk update', async () => {
      const updateData = {
        status: 'resolved' as const,
      };

      const bulkResult = {
        updated: 1,
        failed: ['anom-2'],
      };

      (api.post as any).mockResolvedValue(bulkResult);

      const result = await anomalyApi.bulkUpdate(['anom-1', 'anom-2'], updateData);

      expect(result.updated).toBe(1);
      expect(result.failed).toContain('anom-2');
    });
  });

  describe('getRelated', () => {
    it('should get related anomalies by user', async () => {
      const relatedAnomalies: PaginatedResponse<AnomalyDetection> = {
        data: [mockAnomalies[0], mockAnomalies[1]],
        pagination: {
          total: 2,
          offset: 0,
          limit: 10,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(relatedAnomalies);

      const result = await anomalyApi.getRelated({
        user_id: 'user-1',
        limit: 10,
      });

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/related', {
        user_id: 'user-1',
        limit: 10,
      });
      expect(result.data).toHaveLength(2);
    });

    it('should get related anomalies by session', async () => {
      const relatedAnomalies: PaginatedResponse<AnomalyDetection> = {
        data: [mockAnomalies[0]],
        pagination: {
          total: 1,
          offset: 0,
          limit: 10,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(relatedAnomalies);

      const result = await anomalyApi.getRelated({
        session_id: 'sess-1',
      });

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/related', {
        session_id: 'sess-1',
      });
      expect(result.data).toHaveLength(1);
    });

    it('should get related anomalies by target', async () => {
      const relatedAnomalies: PaginatedResponse<AnomalyDetection> = {
        data: [mockAnomalies[0], mockAnomalies[2]],
        pagination: {
          total: 2,
          offset: 0,
          limit: 10,
          has_more: false,
        },
      };

      (api.get as any).mockResolvedValue(relatedAnomalies);

      const result = await anomalyApi.getRelated({
        target_id: 'target-1',
      });

      expect(api.get).toHaveBeenCalledWith('/compliance/anomalies/related', {
        target_id: 'target-1',
      });
      expect(result.data).toHaveLength(2);
    });
  });

  describe('export', () => {
    it('should export anomalies as CSV', async () => {
      const exportResult = {
        download_url: 'https://storage.example.com/exports/anomalies-123.csv',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      };

      (api.post as any).mockResolvedValue(exportResult);

      const result = await anomalyApi.export({
        format: 'csv',
        severity: 'critical' as AnomalySeverity,
      });

      expect(api.post).toHaveBeenCalledWith('/compliance/anomalies/export', {
        format: 'csv',
        severity: 'critical',
      });
      expect(result.download_url).toBeDefined();
      expect(result.expires_at).toBeDefined();
    });

    it('should export anomalies as JSON', async () => {
      const exportResult = {
        download_url: 'https://storage.example.com/exports/anomalies-456.json',
        expires_at: new Date(Date.now() + 3600000).toISOString(),
      };

      (api.post as any).mockResolvedValue(exportResult);

      const result = await anomalyApi.export({
        format: 'json',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });

      expect(api.post).toHaveBeenCalledWith('/compliance/anomalies/export', {
        format: 'json',
        start_date: '2024-01-01',
        end_date: '2024-01-31',
      });
      expect(result.download_url).toContain('.json');
    });
  });
});

describe('anomalyApi error handling', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should handle 401 unauthorized', async () => {
    const error = {
      response: {
        status: 401,
        data: {
          error: {
            code: 'UNAUTHORIZED',
            message: 'Authentication required',
          },
        },
      },
    };

    (api.get as any).mockRejectedValue(error);

    await expect(anomalyApi.list()).rejects.toMatchObject({
      response: { status: 401 },
    });
  });

  it('should handle 403 forbidden', async () => {
    const error = {
      response: {
        status: 403,
        data: {
          error: {
            code: 'FORBIDDEN',
            message: 'Insufficient permissions',
          },
        },
      },
    };

    (api.get as any).mockRejectedValue(error);

    await expect(anomalyApi.list()).rejects.toMatchObject({
      response: { status: 403 },
    });
  });

  it('should handle 500 server error', async () => {
    const error = {
      response: {
        status: 500,
        data: {
          error: {
            code: 'INTERNAL_ERROR',
            message: 'Internal server error',
          },
        },
      },
    };

    (api.get as any).mockRejectedValue(error);

    await expect(anomalyApi.list()).rejects.toMatchObject({
      response: { status: 500 },
    });
  });
});

describe('anomalyApi type safety', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should preserve AnomalyDetection type structure', async () => {
    (api.get as any).mockResolvedValue(mockAnomalies[0]);

    const result = await anomalyApi.get('anom-1');

    expect(result).toHaveProperty('id');
    expect(result).toHaveProperty('type');
    expect(result).toHaveProperty('severity');
    expect(result).toHaveProperty('title');
    expect(result).toHaveProperty('description');
    expect(result).toHaveProperty('detected_at');
    expect(result).toHaveProperty('confidence_score');
    expect(result).toHaveProperty('risk_score');
    expect(result).toHaveProperty('status');
  });

  it('should preserve indicators array structure', async () => {
    (api.get as any).mockResolvedValue(mockAnomalies[0]);

    const result = await anomalyApi.get('anom-1');

    expect(Array.isArray(result.indicators)).toBe(true);
    if (result.indicators && result.indicators.length > 0) {
      expect(result.indicators[0]).toHaveProperty('type');
      expect(result.indicators[0]).toHaveProperty('description');
      expect(result.indicators[0]).toHaveProperty('confidence');
    }
  });
});

describe('anomalyApi params serialization', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should serialize date range params correctly', async () => {
    (api.get as any).mockResolvedValue([]);

    const filters = {
      start_date: '2024-01-01T00:00:00Z',
      end_date: '2024-01-31T23:59:59Z',
      period: 30,
    };

    await anomalyApi.list(filters);

    expect(api.get).toHaveBeenCalledWith('/compliance/anomalies', filters);
  });

  it('should handle complex filter combinations', async () => {
    (api.get as any).mockResolvedValue([]);

    const filters = {
      severity: 'high' as AnomalySeverity,
      status: 'open',
      type: 'behavioral' as AnomalyType,
      start_date: '2024-01-01',
      end_date: '2024-01-31',
      limit: 50,
    };

    await anomalyApi.list(filters);

    expect(api.get).toHaveBeenCalledWith('/compliance/anomalies', filters);
  });
});
