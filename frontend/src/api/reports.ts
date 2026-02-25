/**
 * Reports API client for the PAM platform
 * Handles report generation, retrieval, and management
 */

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
  ReportTemplate,
  ReportGenerationProgress,
  ScheduledReportExecution,
} from '@/types/reports';
import type { PaginatedResponse } from '@/types';

// Reports API
export const reportsApi = {
  // List reports with filtering
  list: (params?: ReportListParams) =>
    api.get<PaginatedResponse<Report>>('/reports', params),

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
    api.get<PaginatedResponse<ReportJob>>(`/reports/${reportId}/jobs`, params),

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

  // ========== Report Snapshots (Generated Reports) ==========

  // List report snapshots with filters
  listSnapshots: (params?: ReportListParams) =>
    api.get<PaginatedResponse<ReportSnapshot>>('/reports/snapshots', params),

  // Get a specific report snapshot by ID
  getSnapshot: (id: string) =>
    api.get<ReportSnapshot>(`/reports/snapshots/${id}`),

  // Download a report file
  downloadSnapshot: (id: string) =>
    api.get<{ download_url: string; filename: string }>(`/reports/snapshots/${id}/download`),

  // Delete a report snapshot
  deleteSnapshot: (id: string) =>
    api.delete<{ message: string }>(`/reports/snapshots/${id}`),

  // Get report generation progress
  getSnapshotProgress: (id: string) =>
    api.get<ReportGenerationProgress>(`/reports/snapshots/${id}/progress`),

  // Run report now
  run: (id: string) => api.post<{ snapshot_id: string; status: string }>(`/reports/${id}/run`, {}),

  // Generate report with format
  generateReport: (id: string, format: string) =>
    api.post<{ download_url: string; expires_at: string }>(`/reports/${id}/generate`, { format }),

  // ========== Templates ==========

  // Get report templates
  listTemplates: (params?: { type?: string; framework?: string }) =>
    api.get<ReportTemplate[]>('/reports/templates', params),

  // Get specific template
  getTemplate: (id: string) => api.get<ReportTemplate>(`/reports/templates/${id}`),

  // Generate from template
  generateFromTemplate: (templateId: string, data: {
    name: string;
    period_start: string;
    period_end: string;
    filters?: Record<string, unknown>;
    format?: string;
  }) =>
    api.post<{ snapshot_id: string; status: string }>(`/reports/templates/${templateId}/generate`, data),

  // ========== Metadata & Utilities ==========

  // Get available report types
  getTypes: () => api.get<Array<{
    type: string;
    name: string;
    description: string;
    formats: string[];
  }>>('/reports/types'),

  // Get available frameworks
  getFrameworks: () => api.get<Array<{
    framework: string;
    name: string;
    description: string;
  }>>('/reports/frameworks'),

  // Export raw data without full report generation
  exportData: (data: {
    type: string;
    format: 'csv' | 'json' | 'xlsx';
    period_start: string;
    period_end: string;
    filters?: Record<string, unknown>;
  }) =>
    api.post<{ download_url: string; expires_at: string }>('/reports/export', data),

  // Validate report configuration
  validateConfig: (data: Omit<GenerateReportRequest, 'report_id'>) =>
    api.post<{
      valid: boolean;
      errors?: Array<{ field: string; message: string }>;
      estimated_rows?: number;
      estimated_duration_seconds?: number;
    }>('/reports/validate', data),

  // Get compliance status summary
  getComplianceStatus: (standard?: string) =>
    api.get<{
      overall_score: number;
      controls: { name: string; status: 'compliant' | 'non_compliant' | 'partial'; score: number }[];
      last_updated: string;
    }>('/reports/compliance-status', { standard }),
};

// Report Schedules API
export const reportSchedulesApi = {
  // List schedules
  list: (params?: ReportScheduleListParams) =>
    api.get<PaginatedResponse<ReportSchedule>>('/reports/schedules', params),

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
    api.get<PaginatedResponse<ReportJob>>(`/reports/schedules/${id}/history`, params),

  // Preview next run times
  previewNextRuns: (id: string, count: number = 5) =>
    api.get<{ next_runs: string[] }>(`/reports/schedules/${id}/preview`, { count }),

  // Get scheduled executions
  getScheduledExecutions: (reportId: string, params?: { status?: string; limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<ScheduledReportExecution>>(`/reports/${reportId}/executions`, params),

  // Cancel scheduled execution
  cancelExecution: (executionId: string) =>
    api.post<{ message: string }>(`/reports/executions/${executionId}/cancel`, {}),

  // Schedule a report
  schedule: (reportId: string, schedule: ReportSchedule) =>
    api.post<{ schedule_id: string; next_run_at: string }>(`/reports/${reportId}/schedule`, schedule),

  // Update report schedule
  updateSchedule: (reportId: string, schedule: ReportSchedule) =>
    api.patch<{ schedule_id: string; next_run_at: string }>(`/reports/${reportId}/schedule`, schedule),
};

// Report Snapshots API (legacy methods for backward compatibility)
export const reportSnapshotsApi = {
  // List snapshots for a report
  list: (reportId: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<ReportSnapshot>>(`/reports/${reportId}/snapshots`, params),

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
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions', params),

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
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions/pending', params),

  // Get expiring exceptions
  getExpiring: (days: number = 30, params?: Pick<ExceptionListParams, 'limit' | 'offset'>) =>
    api.get<PaginatedResponse<ComplianceException>>(`/compliance/exceptions/expiring`, { days, ...params }),

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
  ReportTemplate,
  ReportGenerationProgress,
  ScheduledReportExecution,
};
