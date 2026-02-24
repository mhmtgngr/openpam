import { api } from './client';
import type { ComplianceReport, PaginatedResponse } from '@/types';

export interface ReportListParams {
  limit?: number;
  offset?: number;
  type?: string;
}

export interface CreateReportData {
  name: string;
  type: 'soc2' | 'iso27001' | 'pci_dss' | 'hipaa' | 'custom';
  description?: string;
  schedule: string;
  config: {
    period_start: string;
    period_end: string;
    include_sections: string[];
    filters?: Record<string, unknown>;
  };
}

export const reportsApi = {
  // List reports
  list: (params?: ReportListParams) =>
    api.get<PaginatedResponse<ComplianceReport>>('/reports', params),

  // Get report by ID
  get: (id: string) => api.get<ComplianceReport>(`/reports/${id}`),

  // Create report
  create: (data: CreateReportData) => api.post<ComplianceReport>('/reports', data),

  // Update report
  update: (id: string, data: Partial<CreateReportData>) =>
    api.patch<ComplianceReport>(`/reports/${id}`, data),

  // Delete report
  delete: (id: string) => api.delete<void>(`/reports/${id}`),

  // Generate report
  generate: (id: string, format: 'pdf' | 'csv' | 'json') =>
    api.post<{ download_url: string; expires_at: string }>(`/reports/${id}/generate`, { format }),

  // Run report now
  run: (id: string) => api.post<ComplianceReport>(`/reports/${id}/run`, {}),

  // Get report types
  getTypes: () => api.get<{ type: string; name: string; description: string }[]>('/reports/types'),

  // Get compliance status
  getComplianceStatus: (standard?: string) =>
    api.get<{
      overall_score: number;
      controls: { name: string; status: 'compliant' | 'non_compliant' | 'partial'; score: number }[];
      last_updated: string;
    }>('/reports/compliance-status', { standard }),
};
