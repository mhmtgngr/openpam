import { api } from './client';
import type {
  AnomalyDetection,
  AnomalyListParams,
  AnomalySeverity,
  AnomalyType,
  PaginatedResponse,
} from '@/types';

export interface AnomalyUpdateData {
  status?: 'open' | 'investigating' | 'resolved' | 'false_positive';
  assigned_to?: string;
  resolution_notes?: string;
}

export interface AnomalySummary {
  total: number;
  by_severity: Record<AnomalySeverity, number> & Record<string, number>;
  by_type: Record<AnomalyType | string, number>;
  by_status: Record<string, number>;
  resolved_this_period: number;
  avg_resolution_time_hours: number;
  critical_open: number;
  high_open: number;
}

export interface AnomalyListFilters extends AnomalyListParams {
  start_date?: string;
  end_date?: string;
}

export const anomalyApi = {
  // List anomalies with filters
  list: (params?: AnomalyListFilters) =>
    api.get<PaginatedResponse<AnomalyDetection>>('/compliance/anomalies', params),

  // Get single anomaly by ID
  get: (id: string) =>
    api.get<AnomalyDetection>(`/compliance/anomalies/${id}`),

  // Update anomaly status and notes
  update: (id: string, data: AnomalyUpdateData) =>
    api.patch<AnomalyDetection>(`/compliance/anomalies/${id}`, data),

  // Acknowledge anomaly (start investigation)
  acknowledge: (id: string) =>
    api.post<AnomalyDetection>(`/compliance/anomalies/${id}/acknowledge`, {}),

  // Get anomaly summary statistics
  getSummary: (params?: Pick<AnomalyListFilters, 'start_date' | 'end_date'>) =>
    api.get<AnomalySummary>('/compliance/anomalies/summary', params),

  // Bulk update anomalies
  bulkUpdate: (ids: string[], data: AnomalyUpdateData) =>
    api.post<{ updated: number; failed: string[] }>('/compliance/anomalies/bulk-update', { ids, ...data }),

  // Get related anomalies for a user or target
  getRelated: (params: { user_id?: string; target_id?: string; session_id?: string; limit?: number }) =>
    api.get<PaginatedResponse<AnomalyDetection>>('/compliance/anomalies/related', params),

  // Export anomalies
  export: (params: AnomalyListFilters & { format: 'csv' | 'json' | 'pdf' }) =>
    api.post<{ download_url: string; expires_at: string }>('/compliance/anomalies/export', params),
};

export type {
  AnomalyDetection,
  AnomalyListParams,
  AnomalySeverity,
  AnomalyType,
  AnomalyUpdateData,
  AnomalySummary,
  AnomalyListFilters,
};
