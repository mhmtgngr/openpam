/**
 * Report-related type definitions for the PAM platform
 * Supports compliance reports, analytics reports, and custom report generation
 */

import type { ComplianceFramework, User } from './index';

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
 * Base compliance report definition
 */
export interface ComplianceReport {
  id: string;
  name: string;
  type: ReportType;
  framework?: ComplianceFramework;
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
 * Report snapshot - represents a generated report instance
 */
export interface ReportSnapshot {
  id: string;
  report_id: string;
  report_name?: string;
  type: ReportType;
  framework?: ComplianceFramework;
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
  day_of_week?: number; // 0-6 (Sunday-Saturday)
  day_of_month?: number; // 1-31
  time?: string; // HH:MM format
  timezone?: string;
  enabled: boolean;
  recipients?: string[]; // Email addresses
  next_run_at?: string;
}

/**
 * Report generation request
 */
export interface GenerateReportRequest {
  report_id?: string; // Optional if using ad-hoc generation
  type: ReportType;
  framework?: ComplianceFramework;
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
  framework?: ComplianceFramework;
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
  framework?: ComplianceFramework;
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
 * Report section data for rendering
 */
export interface ReportSection {
  id: string;
  title: string;
  type: 'table' | 'chart' | 'summary' | 'text' | 'heatmap';
  content: unknown;
  metadata?: Record<string, unknown>;
}

/**
 * Report generation progress
 */
export interface ReportGenerationProgress {
  report_snapshot_id: string;
  status: ReportStatus;
  progress: number; // 0-100
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
}
