/**
 * Report-related type definitions for the PAM platform
 * Supports compliance reports, analytics reports, and custom report generation
 */

import type { User, PaginatedResponse } from './index';

// Re-export from main types
export type { ComplianceFramework } from './index';

/**
 * Report type enumeration
 */
export type ReportType =
  | 'compliance'
  | 'session_activity'
  | 'command_analysis'
  | 'user_access'
  | 'anomaly_summary'
  | 'audit_trail'
  | 'credential_usage'
  | 'custom';

/**
 * Report format options
 */
export type ReportFormat = 'pdf' | 'csv' | 'json' | 'xlsx' | 'html';

/**
 * Report generation status
 */
export type ReportStatus =
  | 'pending'
  | 'generating'
  | 'completed'
  | 'failed'
  | 'expired'
  | 'scheduled'
  | 'cancelled';

/**
 * Report schedule frequency
 */
export type ReportScheduleFrequency =
  | 'once'
  | 'hourly'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'quarterly'
  | 'yearly';

/**
 * Exception Status Types
 */
export type ExceptionStatus = 'pending' | 'approved' | 'denied' | 'expired' | 'revoked';

/**
 * Distribution Types
 */
export type DistributionType = 'email' | 'webhook' | 's3' | 'sharepoint';

/**
 * Report snapshot - represents a generated report instance
 */
export interface ReportSnapshot {
  id: string;
  report_id: string;
  report_name?: string;
  type: ReportType;
  framework?: string;
  status: ReportStatus;
  format: ReportFormat;
  file_url?: string;
  file_size_bytes?: number;
  expires_at?: string;
  generation_started_at: string;
  generation_completed_at?: string;
  error_message?: string;
  generated_by: string;
  generated_by_user?: User;
  tenant_id: string;
  config: ReportConfig;
  metadata?: ReportSnapshotMetadata;
  created_at: string;
  snapshot_data?: Record<string, unknown>;
  file_path?: string;
  checksum?: string;
}

/**
 * Extended metadata for report snapshots
 */
export interface ReportSnapshotMetadata {
  row_count?: number;
  duration_seconds?: number;
  sections?: string[];
  summary?: {
    total_records?: number;
    filtered_records?: number;
    date_range?: {
      start: string;
      end: string;
    };
  };
}

/**
 * Report configuration
 */
export interface ReportConfig {
  period_start: string;
  period_end: string;
  include_sections: string[];
  filters: ReportFilter;
  format_options?: FormatOptions;
}

/**
 * Report filters
 */
export interface ReportFilter {
  user_ids?: string[];
  target_ids?: string[];
  credential_ids?: string[];
  session_ids?: string[];
  risk_levels?: string[];
  statuses?: string[];
  environments?: string[];
  session_types?: string[];
  tags?: string[];
  custom_filters?: Record<string, unknown>;
}

/**
 * Format-specific options
 */
export interface FormatOptions {
  include_charts?: boolean;
  include_raw_data?: boolean;
  page_size?: 'letter' | 'a4' | 'legal';
  orientation?: 'portrait' | 'landscape';
  timezone?: string;
  locale?: string;
}

/**
 * Report schedule configuration
 */
export interface ReportSchedule {
  frequency: ReportScheduleFrequency;
  cron_expression?: string;
  day_of_week?: number;
  day_of_month?: number;
  time?: string;
  timezone?: string;
  enabled: boolean;
  recipients?: string[];
  next_run_at?: string;
}

/**
 * Report generation request
 */
export interface GenerateReportRequest {
  report_id?: string;
  type: ReportType;
  framework?: string;
  format: ReportFormat;
  period_start: string;
  period_end: string;
  include_sections?: string[];
  filters?: ReportFilter;
  format_options?: FormatOptions;
  schedule?: ReportSchedule;
}

/**
 * Report list query parameters
 */
export interface ReportListParams {
  type?: ReportType;
  framework?: string;
  status?: ReportStatus;
  format?: ReportFormat;
  generated_by?: string;
  start_date?: string;
  end_date?: string;
  search?: string;
  limit?: number;
  offset?: number;
  sort_by?: 'created_at' | 'period_start' | 'period_end' | 'type';
  sort_order?: 'asc' | 'desc';
}

/**
 * Report template definition
 */
export interface ReportTemplate {
  id: string;
  name: string;
  type: ReportType;
  framework?: string;
  description?: string;
  thumbnail_url?: string;
  config: ReportConfig;
  sections: ReportTemplateSection[];
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Report template section
 */
export interface ReportTemplateSection {
  id: string;
  name: string;
  title: string;
  type: 'table' | 'chart' | 'summary' | 'text' | 'heatmap';
  required: boolean;
  config: Record<string, unknown>;
  order: number;
}

/**
 * Report generation progress
 */
export interface ReportGenerationProgress {
  report_snapshot_id: string;
  status: ReportStatus;
  progress: number;
  current_step: string;
  started_at: string;
  estimated_completion_at?: string;
}

/**
 * Scheduled report execution
 */
export interface ScheduledReportExecution {
  id: string;
  report_id: string;
  report_name: string;
  scheduled_for: string;
  executed_at?: string;
  status: ReportStatus;
  snapshot_id?: string;
  error_message?: string;
  recipients?: string[];
  run_count?: number;
  last_job_id?: string;
}

/**
 * Report Generation Job
 */
export interface ReportJob {
  id: string;
  report_id?: string;
  report_name?: string;
  framework?: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  started_at?: string;
  completed_at?: string;
  error_message?: string;
  created_by: string;
  tenant_id: string;
  schedule_id?: string;
  created_at: string;
  updated_at: string;
}

/**
 * Report Schedule Entity
 */
export interface ReportScheduleEntity {
  id: string;
  name: string;
  report_id?: string;
  framework?: string;
  description?: string;
  frequency: ReportScheduleFrequency;
  cron_expression?: string;
  timezone: string;
  next_run_at: string;
  last_run_at?: string;
  is_active: boolean;
  recipients: string[];
  distribution_config?: DistributionConfig;
  report_config: ReportConfig;
  created_by: string;
  created_by_user?: { id: string; email: string; display_name?: string };
  tenant_id: string;
  created_at: string;
  updated_at: string;
  run_count: number;
  last_job_id?: string;
}

/**
 * Compliance Exception
 */
export interface ComplianceException {
  id: string;
  control_id: string;
  control_name: string;
  control_description?: string;
  framework: string;
  reason: string;
  business_justification: string;
  mitigation_plan?: string;
  requested_by: string;
  requested_by_user?: { id: string; email: string; display_name?: string };
  approved_by?: string;
  approved_by_user?: { id: string; email: string; display_name?: string };
  requested_at: string;
  reviewed_at?: string;
  expires_at?: string;
  status: ExceptionStatus;
  denial_reason?: string;
  review_notes?: string;
  documents?: ExceptionDocument[];
  tenant_id: string;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

/**
 * Exception Document attachment
 */
export interface ExceptionDocument {
  id: string;
  name: string;
  file_url: string;
  file_size_bytes: number;
  uploaded_at: string;
  uploaded_by: string;
}

/**
 * Distribution Configuration
 */
export interface DistributionConfig {
  enabled: boolean;
  methods: DistributionMethod[];
}

/**
 * Distribution Method
 */
export interface DistributionMethod {
  type: DistributionType;
  enabled: boolean;
  config: Record<string, unknown>;
}

/**
 * Exception List Params
 */
export interface ExceptionListParams {
  limit?: number;
  offset?: number;
  status?: ExceptionStatus;
  framework?: string;
  control_id?: string;
  search?: string;
  expires_after?: string;
  expires_before?: string;
}

/**
 * Base compliance report definition
 */
export interface ComplianceReport {
  id: string;
  name: string;
  type: ReportType;
  framework?: string;
  description?: string;
  schedule?: ReportSchedule;
  last_run_at?: string;
  next_run_at?: string;
  status: ReportStatus;
  config: ReportConfig;
  created_by: string;
  created_by_user?: User;
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

/**
 * Report type for distribution (common properties)
 */
export interface Report {
  id: string;
  name: string;
  framework?: string;
  created_at: string;
}

/**
 * Export report request
 */
export interface ExportReportRequest {
  format: ReportFormat;
  include_metadata?: boolean;
  redact_pii?: boolean;
}

/**
 * Create exception request data
 */
export interface CreateExceptionData {
  control_id: string;
  control_name: string;
  framework: string;
  reason: string;
  business_justification: string;
  mitigation_plan?: string;
  expires_at?: string;
  documents?: File[];
}

/**
 * Framework Metadata
 */
export interface FrameworkMetadata {
  type: string;
  name: string;
  description: string;
  version?: string;
  controls: FrameworkControl[];
  categories: FrameworkCategory[];
}

/**
 * Framework Control
 */
export interface FrameworkControl {
  id: string;
  name: string;
  description: string;
  category: string;
  default_severity: 'critical' | 'high' | 'medium' | 'low';
}

/**
 * Framework Category
 */
export interface FrameworkCategory {
  id: string;
  name: string;
  description?: string;
  control_count: number;
}

/**
 * Report Dashboard Data
 */
export interface ReportDashboardData {
  total_reports: number;
  scheduled_reports: number;
  pending_exceptions: number;
  generation_queue: Array<{ id: string; name: string; status: string }>;
}
