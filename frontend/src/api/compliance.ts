import { api } from './client';
import type {
  ComplianceDashboard,
  ComplianceControl,
  ComplianceException,
  ComplianceFramework,
  ComplianceReportParams,
  PaginatedResponse,
  AnomalyDetection,
  AnomalyListParams,
  AnomalySeverity,
  AnomalyType,
} from '@/types';

export interface ComplianceControlListParams {
  framework?: ComplianceFramework;
  status?: string;
  category?: string;
  limit?: number;
  offset?: number;
}

export interface ComplianceExceptionCreateData {
  control_id: string;
  reason: string;
  expires_at?: string;
}

export interface AnomalyUpdateData {
  status?: 'open' | 'investigating' | 'resolved' | 'false_positive';
  assigned_to?: string;
  resolution_notes?: string;
}

export const complianceApi = {
  // Dashboard
  getDashboard: (framework?: ComplianceFramework) =>
    api.get<ComplianceDashboard>('/compliance/dashboard', { framework }),

  // Controls
  getControls: (params?: ComplianceControlListParams) =>
    api.get<PaginatedResponse<ComplianceControl>>('/compliance/controls', params),

  getControl: (id: string) =>
    api.get<ComplianceControl>(`/compliance/controls/${id}`),

  // Assessments
  runAssessment: (framework?: ComplianceFramework) =>
    api.post<{ assessment_id: string; status: string; started_at: string }>('/compliance/assessments/run', { framework }),

  getAssessmentStatus: (assessmentId: string) =>
    api.get<{
      id: string;
      status: 'running' | 'completed' | 'failed';
      progress: number;
      started_at: string;
      completed_at?: string;
    }>(`/compliance/assessments/${assessmentId}/status`),

  // Exceptions
  getExceptions: (params?: { status?: string; limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<ComplianceException>>('/compliance/exceptions', params),

  createException: (data: ComplianceExceptionCreateData) =>
    api.post<ComplianceException>('/compliance/exceptions', data),

  updateException: (id: string, data: Partial<ComplianceExceptionCreateData>) =>
    api.patch<ComplianceException>(`/compliance/exceptions/${id}`, data),

  revokeException: (id: string, reason: string) =>
    api.post<ComplianceException>(`/compliance/exceptions/${id}/revoke`, { reason }),

  // Reports
  generateReport: (params: ComplianceReportParams & { format: 'pdf' | 'csv' | 'json' }) =>
    api.post<{ report_id: string; download_url?: string; status: string }>('/compliance/reports/generate', params),

  getReport: (reportId: string) =>
    api.get<{
      id: string;
      framework: ComplianceFramework;
      status: string;
      download_url?: string;
      created_at: string;
    }>(`/compliance/reports/${reportId}`),

  // Frameworks
  getFrameworks: () =>
    api.get<Array<{
      framework: ComplianceFramework;
      name: string;
      description: string;
      control_count: number;
    }>>('/compliance/frameworks'),

  // Anomaly Detection
  getAnomalies: (params?: AnomalyListParams) =>
    api.get<PaginatedResponse<AnomalyDetection>>('/compliance/anomalies', params),

  getAnomaly: (id: string) =>
    api.get<AnomalyDetection>(`/compliance/anomalies/${id}`),

  updateAnomaly: (id: string, data: AnomalyUpdateData) =>
    api.patch<AnomalyDetection>(`/compliance/anomalies/${id}`, data),

  acknowledgeAnomaly: (id: string) =>
    api.post<AnomalyDetection>(`/compliance/anomalies/${id}/acknowledge`, {}),

  // Anomaly summary
  getAnomalySummary: (params?: Pick<AnomalyListParams, 'start_date' | 'end_date'>) =>
    api.get<{
      total: number;
      by_severity: Record<string, number>;
      by_type: Record<string, number>;
      by_status: Record<string, number>;
      resolved_this_period: number;
      avg_resolution_time_hours: number;
    }>('/compliance/anomalies/summary', params),

  // Ransomware detection
  getRansomwareIndicators: () =>
    api.get<Array<{
      indicator_type: string;
      description: string;
      severity: 'critical' | 'high' | 'medium' | 'low';
      detected_count: number;
    }>>('/compliance/ransomware/indicators'),

  // Emergency workflows
  triggerEmergencyLockdown: (data: { reason: string; scope?: 'all' | 'target' | 'user'; scope_id?: string }) =>
    api.post<{ lockdown_id: string; status: string; initiated_at: string }>('/compliance/emergency/lockdown', data),

  getLockdownStatus: (lockdownId: string) =>
    api.get<{
      id: string;
      status: string;
      reason: string;
      initiated_at: string;
      initiated_by: string;
      released_at?: string;
    }>(`/compliance/emergency/lockdown/${lockdownId}`),

  releaseLockdown: (lockdownId: string, reason: string) =>
    api.post<{ status: string; released_at: string }>(`/compliance/emergency/lockdown/${lockdownId}/release`, { reason }),
};

export type {
  ComplianceDashboard,
  ComplianceControl,
  ComplianceException,
  ComplianceFramework,
  AnomalyDetection,
  AnomalyListParams,
  AnomalySeverity,
  AnomalyType,
};
