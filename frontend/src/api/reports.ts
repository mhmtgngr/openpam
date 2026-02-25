/**
 * Reports API client for the PAM platform
 * Handles report generation, retrieval, and management
 */

import { api } from './client';
import type {
  PaginatedResponse,
  ComplianceFramework,
} from '@/types';
import type {
  ReportSnapshot,
  ReportType,
  ReportFormat,
  ReportStatus,
  ReportListParams,
  GenerateReportRequest,
  ReportTemplate,
  ReportSchedule,
  ReportScheduleFrequency,
  ReportGenerationProgress,
  ScheduledReportExecution,
  Report,
  ExportReportRequest,
  ComplianceException,
  ExceptionListParams,
  ExceptionStatus,
  CreateExceptionData,
  ComplianceReport,
} from '@/types/reports';

// Reports API
export const reportsApi = {
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

  // Generate a new report (ad-hoc or from definition)
  generate: (data: GenerateReportRequest) =>
    api.post<{
      snapshot_id: string;
      status: ReportStatus;
      estimated_completion_at?: string;
      job_id: string;
    }>('/reports/generate', data),

  // ========== Compliance Report Definitions ==========

  // List compliance report definitions
  list: (params?: { framework?: ComplianceFramework; status?: ReportStatus }) =>
    api.get<PaginatedResponse<ComplianceReport>>('/reports', params),

  // Get report by ID
  get: (id: string) => api.get<ComplianceReport>(`/reports/${id}`),

  // Create report definition
  create: (data: {
    name: string;
    type: ReportType;
    framework?: ComplianceFramework;
    description?: string;
    schedule?: ReportSchedule;
    config: {
      period_start: string;
      period_end: string;
      include_sections: string[];
      filters?: Record<string, unknown>;
    };
  }) => api.post<ComplianceReport>('/reports', data),

  // Update report definition
  update: (id: string, data: Partial<{
    name: string;
    type: ReportType;
    framework?: ComplianceFramework;
    description?: string;
    schedule?: ReportSchedule;
    config: {
      period_start: string;
      period_end: string;
      include_sections: string[];
      filters?: Record<string, unknown>;
    };
  }>) =>
    api.patch<ComplianceReport>(`/reports/${id}`, data),

  // Delete report definition
  delete: (id: string) => api.delete<void>(`/reports/${id}`),

  // Run report now
  run: (id: string) => api.post<{ snapshot_id: string; status: string }>(`/reports/${id}/run`, {}),

  // Generate report with format
  generateReport: (id: string, format: ReportFormat) =>
    api.post<{ download_url: string; expires_at: string }>(`/reports/${id}/generate`, { format }),

  // Export a report (returns download URL)
  export: (id: string, request: ExportReportRequest) =>
    api.post<{ download_url: string; expires_at: string }>(`/reports/${id}/export`, request),

  // Get direct download URL for a report
  download: (id: string) => `/reports/${id}/download`,

  // ========== Scheduling ==========

  // Schedule a report
  schedule: (reportId: string, schedule: ReportSchedule) =>
    api.post<{ schedule_id: string; next_run_at: string }>(`/reports/${reportId}/schedule`, schedule),

  // Update report schedule
  updateSchedule: (reportId: string, schedule: ReportSchedule) =>
    api.patch<{ schedule_id: string; next_run_at: string }>(`/reports/${reportId}/schedule`, schedule),

  // Get scheduled executions
  getScheduledExecutions: (reportId: string, params?: { status?: ReportStatus; limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<ScheduledReportExecution>>(`/reports/${reportId}/executions`, params),

  // Cancel scheduled execution
  cancelExecution: (executionId: string) =>
    api.post<{ message: string }>(`/reports/executions/${executionId}/cancel`, {}),

  // ========== Templates ==========

  // Get report templates
  listTemplates: (params?: { type?: ReportType; framework?: ComplianceFramework }) =>
    api.get<ReportTemplate[]>('/reports/templates', params),

  // Get specific template
  getTemplate: (id: string) => api.get<ReportTemplate>(`/reports/templates/${id}`),

  // Generate from template
  generateFromTemplate: (templateId: string, data: {
    name: string;
    period_start: string;
    period_end: string;
    filters?: Record<string, unknown>;
    format?: ReportFormat;
  }) =>
    api.post<{ snapshot_id: string; status: string }>(`/reports/templates/${templateId}/generate`, data),

  // ========== Metadata & Utilities ==========

  // Get available report types
  getTypes: () => api.get<Array<{
    type: ReportType;
    name: string;
    description: string;
    formats: ReportFormat[];
  }>>('/reports/types'),

  // Get available frameworks
  getFrameworks: () => api.get<Array<{
    framework: ComplianceFramework;
    name: string;
    description: string;
  }>>('/reports/frameworks'),

  // Export raw data without full report generation
  exportData: (data: {
    type: ReportType;
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

  // Get dashboard data
  getDashboard: () =>
    api.get<{
      total_reports: number;
      scheduled_reports: number;
      pending_exceptions: number;
      generation_queue: Array<{ id: string; name: string; status: string }>;
    }>('/reports/dashboard'),
};

// Export types
export type {
  ReportSnapshot,
  ReportType,
  ReportFormat,
  ReportStatus,
  ReportListParams,
  GenerateReportRequest,
  ReportTemplate,
  ReportSchedule,
  ReportScheduleFrequency,
  ReportGenerationProgress,
  ScheduledReportExecution,
  Report,
  ExportReportRequest,
  ComplianceException,
  ExceptionListParams,
  ExceptionStatus,
  CreateExceptionData,
};

// Compliance Exceptions API
export const complianceExceptionsApi = {
  // List exceptions with filters
  list: (params?: ExceptionListParams) =>
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions', params),

  // Get a specific exception by ID
  get: (id: string) =>
    api.get<ComplianceException>(`/compliance/exceptions/${id}`),

  // Create a new exception request
  create: (data: {
    control_id: string;
    control_name: string;
    framework: string;
    reason: string;
    business_justification: string;
    mitigation_plan?: string;
    expires_at?: string;
    documents?: File[];
  }) => api.post<ComplianceException>('/compliance/exceptions', data),

  // Update an existing exception
  update: (id: string, data: {
    control_id?: string;
    control_name?: string;
    framework?: string;
    reason?: string;
    business_justification?: string;
    mitigation_plan?: string;
    expires_at?: string;
  }) => api.patch<ComplianceException>(`/compliance/exceptions/${id}`, data),

  // Approve an exception
  approve: (id: string, notes?: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/approve`, { notes }),

  // Deny an exception
  deny: (id: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/deny`, { reason }),

  // Revoke an exception
  revoke: (id: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/revoke`, { reason }),

  // Delete an exception
  delete: (id: string) =>
    api.delete<void>(`/compliance/exceptions/${id}`),

  // Get pending exceptions
  getPending: () =>
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions/pending'),

  // Get expiring exceptions
  getExpiring: (days: number) =>
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions/expiring', { days }),

  // Get exceptions by framework
  getByFramework: (framework: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<ComplianceException>>(`/compliance/exceptions/framework/${framework}`, params),
};
