import { api } from './client';
import type {
  Report,
  ReportSnapshot,
  ReportJob,
  ReportSchedule,
  ComplianceException,
  FrameworkMetadata,
  ReportDashboardData,
  ReportListParams,
  ReportScheduleListParams,
  ExceptionListParams,
  CreateReportData,
  UpdateReportData,
  CreateReportScheduleData,
  UpdateReportScheduleData,
  CreateExceptionData,
  UpdateExceptionData,
  GenerateReportRequest,
  GenerateReportResponse,
  ExportReportRequest,
  ExportReportResponse,
  PaginatedResponse,
} from '@/types/reports';
import type { PaginatedResponse as BasePaginatedResponse } from '@/types';

// Reports API
export const reportsApi = {
  // List reports with filtering
  list: (params?: ReportListParams) =>
    api.get<BasePaginatedResponse<Report>>('/reports', params),

  // Get report by ID
  get: (id: string) =>
    api.get<Report>(`/reports/${id}`),

  // Create a new report
  create: (data: CreateReportData) =>
    api.post<Report>('/reports', data),

  // Update report metadata
  update: (id: string, data: UpdateReportData) =>
    api.patch<Report>(`/reports/${id}`, data),

  // Delete report (soft delete)
  delete: (id: string) =>
    api.delete<void>(`/reports/${id}`),

  // Generate report on-demand
  generate: (data: GenerateReportRequest) =>
    api.post<GenerateReportResponse>('/reports/generate', data),

  // Get generation job status
  getJobStatus: (jobId: string) =>
    api.get<ReportJob>(`/reports/jobs/${jobId}`),

  // List jobs for a report
  listJobs: (reportId: string, params?: { limit?: number; offset?: number }) =>
    api.get<BasePaginatedResponse<ReportJob>>(`/reports/${reportId}/jobs`, params),

  // Cancel running job
  cancelJob: (jobId: string) =>
    api.post<{ cancelled: boolean }>(`/reports/jobs/${jobId}/cancel`, {}),

  // Export report
  export: (id: string, request: ExportReportRequest) =>
    api.post<ExportReportResponse>(`/reports/${id}/export`, request),

  // Download report file
  download: (id: string) => {
    // Return the URL for direct download
    const API_BASE = (import.meta as unknown as { env: { VITE_API_URL?: string } }).env.VITE_API_URL || 'http://localhost:8500/api/v1';
    return `${API_BASE}/reports/${id}/download`;
  },

  // Get dashboard data
  getDashboard: () =>
    api.get<ReportDashboardData>('/reports/dashboard'),

  // Get framework metadata
  getFrameworkMetadata: (framework: string) =>
    api.get<FrameworkMetadata>(`/reports/frameworks/${framework}`),

  // List all frameworks
  listFrameworks: () =>
    api.get<FrameworkMetadata[]>('/reports/frameworks'),
};

// Report Schedules API
export const reportSchedulesApi = {
  // List schedules
  list: (params?: ReportScheduleListParams) =>
    api.get<BasePaginatedResponse<ReportSchedule>>('/reports/schedules', params),

  // Get schedule by ID
  get: (id: string) =>
    api.get<ReportSchedule>(`/reports/schedules/${id}`),

  // Create schedule
  create: (data: CreateReportScheduleData) =>
    api.post<ReportSchedule>('/reports/schedules', data),

  // Update schedule
  update: (id: string, data: UpdateReportScheduleData) =>
    api.patch<ReportSchedule>(`/reports/schedules/${id}`, data),

  // Delete schedule
  delete: (id: string) =>
    api.delete<void>(`/reports/schedules/${id}`),

  // Pause schedule
  pause: (id: string) =>
    api.post<ReportSchedule>(`/reports/schedules/${id}/pause`, {}),

  // Resume schedule
  resume: (id: string) =>
    api.post<ReportSchedule>(`/reports/schedules/${id}/resume`, {}),

  // Trigger immediate run
  runNow: (id: string) =>
    api.post<GenerateReportResponse>(`/reports/schedules/${id}/run`, {}),

  // Get schedule run history
  getHistory: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get<BasePaginatedResponse<ReportJob>>(`/reports/schedules/${id}/history`, params),

  // Preview next run times
  previewNextRuns: (id: string, count: number = 5) =>
    api.get<{ next_runs: string[] }>(`/reports/schedules/${id}/preview`, { count }),
};

// Report Snapshots API
export const reportSnapshotsApi = {
  // List snapshots for a report
  list: (reportId: string, params?: { limit?: number; offset?: number }) =>
    api.get<BasePaginatedResponse<ReportSnapshot>>(`/reports/${reportId}/snapshots`, params),

  // Get specific snapshot
  get: (id: string) =>
    api.get<ReportSnapshot>(`/reports/snapshots/${id}`),

  // Compare two snapshots
  compare: (snapshotId1: string, snapshotId2: string) =>
    api.get<{
      snapshot1: ReportSnapshot;
      snapshot2: ReportSnapshot;
      differences: Array<{
        field: string;
        value1: unknown;
        value2: unknown;
      }>;
    }>(`/reports/snapshots/compare`, { snapshot_id_1: snapshotId1, snapshot_id_2: snapshotId2 }),

  // Restore snapshot to create new report version
  restore: (id: string) =>
    api.post<Report>(`/reports/snapshots/${id}/restore`, {}),

  // Download snapshot
  download: (id: string) => {
    const API_BASE = (import.meta as unknown as { env: { VITE_API_URL?: string } }).env.VITE_API_URL || 'http://localhost:8500/api/v1';
    return `${API_BASE}/reports/snapshots/${id}/download`;
  },
};

// Compliance Exceptions API
export const complianceExceptionsApi = {
  // List exceptions
  list: (params?: ExceptionListParams) =>
    api.get<BasePaginatedResponse<ComplianceException>>('/compliance/exceptions', params),

  // Get exception by ID
  get: (id: string) =>
    api.get<ComplianceException>(`/compliance/exceptions/${id}`),

  // Create exception request
  create: (data: CreateExceptionData) =>
    api.post<ComplianceException>('/compliance/exceptions', data),

  // Update exception (approve/deny/update notes)
  update: (id: string, data: UpdateExceptionData) =>
    api.patch<ComplianceException>(`/compliance/exceptions/${id}`, data),

  // Delete exception
  delete: (id: string) =>
    api.delete<void>(`/compliance/exceptions/${id}`),

  // Approve exception
  approve: (id: string, notes?: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/approve`, { notes }),

  // Deny exception
  deny: (id: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/deny`, { reason }),

  // Revoke exception
  revoke: (id: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/revoke`, { reason }),

  // Extend exception expiration
  extend: (id: string, expiresAt: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/extend`, {
      expires_at: expiresAt,
      reason,
    }),

  // Get pending exceptions
  getPending: (params?: Pick<ExceptionListParams, 'limit' | 'offset'>) =>
    api.get<BasePaginatedResponse<ComplianceException>>('/compliance/exceptions/pending', params),

  // Get expiring exceptions
  getExpiring: (days: number = 30, params?: Pick<ExceptionListParams, 'limit' | 'offset'>) =>
    api.get<BasePaginatedResponse<ComplianceException>>(`/compliance/exceptions/expiring`, { days, ...params }),

  // Upload exception document
  uploadDocument: (exceptionId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const API_BASE = (import.meta as unknown as { env: { VITE_API_URL?: string } }).env.VITE_API_URL || 'http://localhost:8500/api/v1';
    const token = localStorage.getItem('access_token');

    return fetch(`${API_BASE}/compliance/exceptions/${exceptionId}/documents`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: formData,
    }).then((res) => res.json());
  },

  // Delete exception document
  deleteDocument: (exceptionId: string, documentId: string) =>
    api.delete<void>(`/compliance/exceptions/${exceptionId}/documents/${documentId}`),
};

// Type exports for convenience
export type {
  Report,
  ReportSnapshot,
  ReportJob,
  ReportSchedule,
  ComplianceException,
  FrameworkMetadata,
  ReportDashboardData,
  ReportListParams,
  ReportScheduleListParams,
  ExceptionListParams,
  CreateReportData,
  UpdateReportData,
  CreateReportScheduleData,
  UpdateReportScheduleData,
  CreateExceptionData,
  UpdateExceptionData,
  GenerateReportRequest,
  GenerateReportResponse,
  ExportReportRequest,
  ExportReportResponse,
};
