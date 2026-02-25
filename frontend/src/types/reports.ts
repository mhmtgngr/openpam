<<<<<<< HEAD
// Report Types for OpenPAM Analytics Service

// Compliance Framework Types
export type ComplianceFramework =
  | 'soc2'
  | 'iso27001'
  | 'pci_dss'
  | 'hipaa'
  | 'gdpr'
  | 'nerc_cip'
  | 'custom';

// Report Status Types
=======
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
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
export type ReportStatus =
  | 'pending'
  | 'generating'
  | 'completed'
  | 'failed'
<<<<<<< HEAD
  | 'scheduled'
  | 'cancelled';

// Report Format Types
export type ReportFormat = 'pdf' | 'html' | 'csv' | 'json' | 'xlsx';

// Report Frequency Types
export type ScheduleFrequency =
  | 'once'
  | 'hourly'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'quarterly'
  | 'yearly';

// Exception Status Types
export type ExceptionStatus = 'pending' | 'approved' | 'denied' | 'expired' | 'revoked';

// Distribution Types
export type DistributionType = 'email' | 'webhook' | 's3' | 'sharepoint';

// Main Report Entity
export interface Report {
  id: string;
  name: string;
  description?: string;
  framework: ComplianceFramework;
  status: ReportStatus;
  period_start: string;
  period_end: string;
  generated_at?: string;
  expires_at?: string;
  created_by: string;
  created_by_user?: UserSummary;
  tenant_id: string;
  report_data: ReportData;
  report_metadata: ReportMetadata;
  snapshot_id?: string;
  file_url?: string;
  file_size_bytes?: number;
  download_count: number;
  created_at: string;
  updated_at: string;
  deleted_at: string | null;
}

// Report Snapshot (immutable version of generated report)
export interface ReportSnapshot {
  id: string;
  report_id: string;
  report_name: string;
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  snapshot_data: Record<string, unknown>;
  generated_at: string;
  generated_by: string;
  tenant_id: string;
  file_path?: string;
  file_format: ReportFormat;
  file_size_bytes?: number;
  checksum?: string;
  expires_at?: string;
  created_at: string;
}

// Report Generation Job
export interface ReportJob {
  id: string;
  report_id?: string;
  report_name?: string;
  framework: ComplianceFramework;
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

// Report Schedule
export interface ReportSchedule {
  id: string;
  name: string;
  report_id?: string;
  framework: ComplianceFramework;
  description?: string;
  frequency: ScheduleFrequency;
  cron_expression?: string;
  timezone: string;
  next_run_at: string;
  last_run_at?: string;
  is_active: boolean;
  recipients: string[];
  distribution_config: DistributionConfig;
  report_config: ReportConfig;
  created_by: string;
  created_by_user?: UserSummary;
  tenant_id: string;
  created_at: string;
  updated_at: string;
  run_count: number;
  last_job_id?: string;
}

// Compliance Exception
export interface ComplianceException {
  id: string;
  control_id: string;
  control_name: string;
  control_description?: string;
  framework: ComplianceFramework;
  reason: string;
  business_justification: string;
  mitigation_plan?: string;
  requested_by: string;
  requested_by_user?: UserSummary;
  approved_by?: string;
  approved_by_user?: UserSummary;
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

// Exception Document attachment
export interface ExceptionDocument {
  id: string;
  name: string;
  file_url: string;
  file_size_bytes: number;
  uploaded_at: string;
  uploaded_by: string;
}

// Distribution Configuration
export interface DistributionConfig {
  enabled: boolean;
  methods: DistributionMethod[];
}

export interface DistributionMethod {
  type: DistributionType;
  enabled: boolean;
  config: Record<string, unknown>;
}

// Email Distribution Config
export interface EmailDistributionConfig {
  to: string[];
  cc?: string[];
  bcc?: string[];
  subject?: string;
  include_attachments: boolean;
  formats: ReportFormat[];
}

// Webhook Distribution Config
export interface WebhookDistributionConfig {
  url: string;
  headers?: Record<string, string>;
  retry_count: number;
  timeout_seconds: number;
  verify_ssl: boolean;
}

// S3 Distribution Config
export interface S3DistributionConfig {
  bucket: string;
  prefix: string;
  region: string;
  formats: ReportFormat[];
}

// Report Configuration
export interface ReportConfig {
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  include_sections: ReportSection[];
  filters?: ReportFilter;
  format?: ReportFormat;
  options?: ReportOptions;
}

export interface ReportSection {
  id: string;
  name: string;
  enabled: boolean;
  config?: Record<string, unknown>;
}

=======
  | 'expired'
  | 'scheduled';

/**
 * Report schedule frequency
 */
export type ReportScheduleFrequency =
  | 'once'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'quarterly';

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
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
export interface ReportFilter {
  user_ids?: string[];
  target_ids?: string[];
  credential_ids?: string[];
<<<<<<< HEAD
  session_types?: string[];
  environments?: string[];
  risk_levels?: string[];
=======
  session_ids?: string[];
  risk_levels?: string[];
  statuses?: string[];
  environments?: string[];
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
  tags?: string[];
  custom_filters?: Record<string, unknown>;
}

<<<<<<< HEAD
export interface ReportOptions {
  include_pii: boolean;
  include_session_logs: boolean;
  include_command_history: boolean;
  redact_sensitive_data: boolean;
  aggregate_results: boolean;
  comparison_period?: {
    start: string;
    end: string;
  };
}

// Report Data (stored in JSONB)
export interface ReportData {
  framework: ComplianceFramework;
  period: {
    start: string;
    end: string;
  };
  summary: ReportSummary;
  controls: ControlResult[];
  findings: ReportFinding[];
  metrics: ReportMetrics;
  recommendations?: string[];
}

export interface ReportSummary {
  overall_score: number;
  compliant_controls: number;
  non_compliant_controls: number;
  partial_controls: number;
  not_applicable_controls: number;
  total_findings: number;
  critical_findings: number;
  high_findings: number;
  medium_findings: number;
  low_findings: number;
}

export interface ControlResult {
  id: string;
  name: string;
  description?: string;
  category?: string;
  status: 'compliant' | 'non_compliant' | 'partial' | 'not_applicable';
  score: number;
  evidence_count: number;
  findings: string[];
  last_assessed: string;
}

export interface ReportFinding {
  id: string;
  control_id: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  title: string;
  description: string;
  affected_resources?: string[];
  recommendation?: string;
  status: 'open' | 'in_progress' | 'resolved' | 'mitigated';
  discovered_at: string;
}

export interface ReportMetrics {
  total_sessions: number;
  total_users: number;
  privileged_sessions: number;
  failed_authentications: number;
  high_risk_commands: number;
  anomalies_detected: number;
  exceptions_active: number;
}

// Report Metadata
export interface ReportMetadata {
  version: string;
  generated_by: string;
  generated_at: string;
  tenant_id: string;
  data_range: {
    start: string;
    end: string;
  };
  filters_used?: ReportFilter;
  processing_time_ms: number;
  row_count?: number;
}

// User Summary (lightweight user reference)
export interface UserSummary {
  id: string;
  email: string;
  display_name?: string;
  first_name?: string;
  last_name?: string;
}

// List and Filter Types
export interface ReportListParams {
  limit?: number;
  offset?: number;
  framework?: ComplianceFramework;
  status?: ReportStatus;
  period_start?: string;
  period_end?: string;
  search?: string;
  sort_by?: 'created_at' | 'generated_at' | 'name' | 'framework';
  sort_order?: 'asc' | 'desc';
}

export interface ReportScheduleListParams {
  limit?: number;
  offset?: number;
  is_active?: boolean;
  framework?: ComplianceFramework;
  frequency?: ScheduleFrequency;
  search?: string;
}

export interface ExceptionListParams {
  limit?: number;
  offset?: number;
  status?: ExceptionStatus;
  framework?: ComplianceFramework;
  control_id?: string;
  search?: string;
  expires_after?: string;
  expires_before?: string;
}

// Create/Update Types
export interface CreateReportData {
  name: string;
  description?: string;
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  config: ReportConfig;
  schedule_id?: string;
}

export interface UpdateReportData {
  name?: string;
  description?: string;
  expires_at?: string;
  report_data?: Partial<ReportData>;
}

export interface CreateReportScheduleData {
  name: string;
  description?: string;
  report_id?: string;
  framework: ComplianceFramework;
  frequency: ScheduleFrequency;
  cron_expression?: string;
  timezone?: string;
  is_active?: boolean;
  recipients: string[];
  distribution_config: DistributionConfig;
  report_config: ReportConfig;
}

export interface UpdateReportScheduleData {
  name?: string;
  description?: string;
  frequency?: ScheduleFrequency;
  cron_expression?: string;
  timezone?: string;
  is_active?: boolean;
  recipients?: string[];
  distribution_config?: DistributionConfig;
  report_config?: ReportConfig;
}

export interface CreateExceptionData {
  control_id: string;
  control_name: string;
  framework: ComplianceFramework;
  reason: string;
  business_justification: string;
  mitigation_plan?: string;
  expires_at?: string;
  documents?: File[];
}

export interface UpdateExceptionData {
  status?: ExceptionStatus;
  denial_reason?: string;
  review_notes?: string;
  expires_at?: string;
  mitigation_plan?: string;
}

// Report Generation Request
export interface GenerateReportRequest {
  framework: ComplianceFramework;
=======
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
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
  period_start: string;
  period_end: string;
  include_sections?: string[];
  filters?: ReportFilter;
<<<<<<< HEAD
  options?: ReportOptions;
  format?: ReportFormat;
}

export interface GenerateReportResponse {
  job_id: string;
  report_id?: string;
  status: ReportStatus;
  estimated_completion?: string;
}

// Report Export
export interface ExportReportRequest {
  format: ReportFormat;
  include_metadata?: boolean;
  redact_pii?: boolean;
}

export interface ExportReportResponse {
  download_url: string;
  expires_at: string;
  file_size_bytes?: number;
}

// Framework Metadata
export interface FrameworkMetadata {
  type: ComplianceFramework;
  name: string;
  description: string;
  version?: string;
  controls: FrameworkControl[];
  categories: FrameworkCategory[];
}

export interface FrameworkControl {
  id: string;
  name: string;
  description: string;
  category: string;
  default_severity: 'critical' | 'high' | 'medium' | 'low';
}

export interface FrameworkCategory {
  id: string;
  name: string;
  description?: string;
  control_count: number;
}

// Report Summary Card Data
export interface ReportSummaryCard {
  id: string;
  name: string;
  framework: ComplianceFramework;
  status: ReportStatus;
  score: number;
  period: string;
  created_at: string;
}

// Dashboard Widget Data
export interface ReportDashboardData {
  total_reports: number;
  recent_reports: Report[];
  scheduled_reports: number;
  active_schedules: ReportSchedule[];
  pending_exceptions: number;
  expiring_exceptions: ComplianceException[];
  framework_breakdown: FrameworkBreakdown;
  generation_queue: ReportJob[];
}

export interface FrameworkBreakdown {
  [framework: string]: {
    total: number;
    compliant: number;
    non_compliant: number;
    avg_score: number;
  };
=======
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
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
}
