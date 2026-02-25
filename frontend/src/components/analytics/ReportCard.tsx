/**
 * ReportCard - Reusable card component for displaying report information
 * Shows report status, download options, and key metadata
 */

import React from 'react';
import { format } from 'date-fns';
import {
  FileText,
  Download,
  Trash2,
  Eye,
  RefreshCw,
  Calendar,
  User,
  HardDrive,
  Clock,
  AlertCircle,
} from 'lucide-react';
import { Badge, StatusBadge } from '@/components/common';
import { Button } from '@/components/common';
import type { ReportSnapshot, ReportStatus, ReportFormat } from '@/types/reports';
import clsx from 'clsx';

interface ReportCardProps {
  report: ReportSnapshot;
  onView?: (report: ReportSnapshot) => void;
  onDownload?: (report: ReportSnapshot) => void;
  onDelete?: (report: ReportSnapshot) => void;
  onRefresh?: (report: ReportSnapshot) => void;
  showActions?: boolean;
  compact?: boolean;
}

const statusConfig: Record<ReportStatus, { variant: 'success' | 'warning' | 'danger' | 'neutral'; icon: React.ElementType; label: string }> = {
  pending: { variant: 'neutral', icon: Clock, label: 'Pending' },
  generating: { variant: 'warning', icon: RefreshCw, label: 'Generating' },
  completed: { variant: 'success', icon: FileText, label: 'Completed' },
  failed: { variant: 'danger', icon: AlertCircle, label: 'Failed' },
  expired: { variant: 'neutral', icon: Clock, label: 'Expired' },
  scheduled: { variant: 'neutral', icon: Calendar, label: 'Scheduled' },
  cancelled: { variant: 'neutral', icon: Clock, label: 'Cancelled' },
};

const formatLabels: Record<ReportFormat, string> = {
  pdf: 'PDF',
  csv: 'CSV',
  json: 'JSON',
  xlsx: 'Excel',
  html: 'HTML',
};

const typeLabels: Record<string, string> = {
  compliance: 'Compliance Report',
  session_activity: 'Session Activity',
  command_analysis: 'Command Analysis',
  user_access: 'User Access',
  anomaly_summary: 'Anomaly Summary',
  audit_trail: 'Audit Trail',
  credential_usage: 'Credential Usage',
  custom: 'Custom Report',
};

const frameworkLabels: Record<string, string> = {
  soc2: 'SOC 2',
  iso27001: 'ISO 27001',
  pci_dss: 'PCI DSS',
  hipaa: 'HIPAA',
  gdpr: 'GDPR',
  custom: 'Custom',
};

export const ReportCard: React.FC<ReportCardProps> = ({
  report,
  onView,
  onDownload,
  onDelete,
  onRefresh,
  showActions = true,
  compact = false,
}) => {
  const status = statusConfig[report.status] || statusConfig.pending;
  const StatusIcon = status.icon;
  const isProcessing = report.status === 'pending' || report.status === 'generating';

  const formatFileSize = (bytes?: number): string => {
    if (!bytes) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const handleDownload = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDownload?.(report);
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete?.(report);
  };

  const handleRefresh = (e: React.MouseEvent) => {
    e.stopPropagation();
    onRefresh?.(report);
  };

  if (compact) {
    return (
      <div
        className={clsx(
          'flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-gray-800/50',
          {
            'border-gray-700': report.status === 'completed',
            'border-warning-500/30 bg-warning-500/5': report.status === 'generating',
            'border-danger-500/30 bg-danger-500/5': report.status === 'failed',
          }
        )}
      >
        <div className="flex items-center gap-3">
          <div className={clsx('rounded-lg p-2', {
            'bg-success-500/20 text-success-400': report.status === 'completed',
            'bg-warning-500/20 text-warning-400': report.status === 'generating' || report.status === 'pending',
            'bg-danger-500/20 text-danger-400': report.status === 'failed',
            'bg-gray-500/20 text-gray-400': report.status === 'expired',
          })}>
            <StatusIcon className="h-4 w-4" />
          </div>
          <div>
            <p className="text-sm font-medium text-white">{report.report_name || typeLabels[report.type] || report.type}</p>
            <div className="flex items-center gap-2 text-xs text-gray-400">
              <span>{formatLabels[report.format]}</span>
              {report.framework && <span>• {frameworkLabels[report.framework]}</span>}
              <span>• {format(new Date(report.created_at), 'MMM d, yyyy')}</span>
            </div>
          </div>
        </div>
        {showActions && (
          <div className="flex items-center gap-2">
            {isProcessing && onRefresh && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                leftIcon={<RefreshCw className="h-3 w-3" />}
              >
                Refresh
              </Button>
            )}
            {report.status === 'completed' && report.file_url && onDownload && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDownload}
                leftIcon={<Download className="h-3 w-3" />}
              >
                Download
              </Button>
            )}
            {onDelete && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDelete}
                leftIcon={<Trash2 className="h-3 w-3 text-danger-400" />}
              />
            )}
          </div>
        )}
      </div>
    );
  }

  return (
    <div
      className={clsx(
        'card group cursor-pointer transition-all hover:shadow-lg',
        {
          'border-l-4 border-l-success-500': report.status === 'completed',
          'border-l-4 border-l-warning-500': report.status === 'generating' || report.status === 'pending',
          'border-l-4 border-l-danger-500': report.status === 'failed',
        }
      )}
      onClick={() => onView?.(report)}
    >
      <div className="card-body">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-3">
            <div className={clsx('rounded-lg p-2', {
              'bg-success-500/20 text-success-400': report.status === 'completed',
              'bg-warning-500/20 text-warning-400': report.status === 'generating' || report.status === 'pending',
              'bg-danger-500/20 text-danger-400': report.status === 'failed',
              'bg-gray-500/20 text-gray-400': report.status === 'expired',
            })}>
              <StatusIcon className={clsx('h-5 w-5', isProcessing && 'animate-spin')} />
            </div>
            <div>
              <h3 className="font-semibold text-white">
                {report.report_name || typeLabels[report.type] || 'Report'}
              </h3>
              <p className="mt-1 text-sm text-gray-400">
                {report.framework ? `${frameworkLabels[report.framework]} - ` : ''}
                {formatLabels[report.format]} Report
              </p>
            </div>
          </div>
          <Badge variant={status.variant} className="flex items-center gap-1">
            <StatusIcon className="h-3 w-3" />
            {status.label}
          </Badge>
        </div>

        {/* Metadata Grid */}
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div className="flex items-center gap-2 text-sm">
            <Calendar className="h-4 w-4 text-gray-500" />
            <div>
              <p className="text-gray-500">Period</p>
              <p className="text-white">
                {format(new Date(report.config.period_start), 'MMM d')} - {format(new Date(report.config.period_end), 'MMM d, yyyy')}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <User className="h-4 w-4 text-gray-500" />
            <div>
              <p className="text-gray-500">Generated By</p>
              <p className="text-white">
                {report.generated_by_user?.display_name || report.generated_by_user?.email || report.generated_by}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Clock className="h-4 w-4 text-gray-500" />
            <div>
              <p className="text-gray-500">Created</p>
              <p className="text-white">{format(new Date(report.created_at), 'MMM d, yyyy HH:mm')}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <HardDrive className="h-4 w-4 text-gray-500" />
            <div>
              <p className="text-gray-500">Size</p>
              <p className="text-white">{formatFileSize(report.file_size_bytes)}</p>
            </div>
          </div>
        </div>

        {/* Error Message */}
        {report.status === 'failed' && report.error_message && (
          <div className="mt-3 rounded-md bg-danger-500/10 p-3 text-sm text-danger-400">
            <AlertCircle className="mr-1 inline h-4 w-4" />
            {report.error_message}
          </div>
        )}

        {/* Expiration Notice */}
        {report.expires_at && report.status === 'completed' && (
          <div className="mt-3 text-xs text-gray-500">
            <Clock className="mr-1 inline h-3 w-3" />
            Expires {format(new Date(report.expires_at), 'MMM d, yyyy HH:mm')}
          </div>
        )}

        {/* Actions */}
        {showActions && (
          <div className="mt-4 flex items-center justify-end gap-2 opacity-0 transition-opacity group-hover:opacity-100">
            {isProcessing && onRefresh && (
              <Button
                variant="secondary"
                size="sm"
                onClick={handleRefresh}
                leftIcon={<RefreshCw className="h-4 w-4" />}
              >
                Refresh Status
              </Button>
            )}
            {report.status === 'completed' && onView && (
              <Button
                variant="secondary"
                size="sm"
                onClick={(e) => {
                  e.stopPropagation();
                  onView(report);
                }}
                leftIcon={<Eye className="h-4 w-4" />}
              >
                View Details
              </Button>
            )}
            {report.status === 'completed' && report.file_url && onDownload && (
              <Button
                variant="primary"
                size="sm"
                onClick={handleDownload}
                leftIcon={<Download className="h-4 w-4" />}
              >
                Download
              </Button>
            )}
            {onDelete && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDelete}
                leftIcon={<Trash2 className="h-4 w-4 text-danger-400" />}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ReportCard;
