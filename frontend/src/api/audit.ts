import { api } from './client';
import type { AuditEvent, ComplianceReport, PaginatedResponse } from '@/types';

export interface AuditListParams {
  limit?: number;
  offset?: number;
  search?: string;
  actor_id?: string;
  action?: string;
  resource_type?: string;
  outcome?: string;
  from?: string;
  to?: string;
}

export interface ComplianceReportListParams {
  limit?: number;
  offset?: number;
  type?: string;
}

export const auditApi = {
  // List audit events
  list: (params?: AuditListParams) =>
    api.get<PaginatedResponse<AuditEvent>>('/audit', params),

  // Get audit event by ID
  get: (id: string) => api.get<AuditEvent>(`/audit/${id}`),

  // Export audit logs
  export: (params?: AuditListParams & { format?: 'csv' | 'json' }) =>
    api.get<Blob>('/audit/export', params),

  // List compliance reports
  listReports: (params?: ComplianceReportListParams) =>
    api.get<PaginatedResponse<ComplianceReport>>('/compliance/reports', params),

  // Get compliance report
  getReport: (id: string) => api.get<ComplianceReport>(`/compliance/reports/${id}`),

  // Create compliance report
  createReport: (data: {
    name: string;
    type: string;
    description?: string;
    schedule?: string;
    config: unknown;
  }) => api.post<ComplianceReport>('/compliance/reports', data),

  // Run compliance report
  runReport: (id: string) => api.post<{ report_id: string; status: string }>(`/compliance/reports/${id}/run`, {}),

  // Download compliance report
  downloadReport: (id: string, format?: 'pdf' | 'csv' | 'json') =>
    api.get<Blob>(`/compliance/reports/${id}/download`, { format }),
};
