import React from 'react';
import { format } from 'date-fns';
import {
  AlertTriangle,
  AlertOctagon,
  AlertCircle,
  Activity,
  Eye,
  User,
  Server,
  Clock,
  CheckCircle,
  Search,
  XCircle,
  Ban,
} from 'lucide-react';
import { Badge, Button } from '@/components/common';
import type { AnomalyDetection, AnomalySeverity } from '@/types';
import clsx from 'clsx';

const severityConfig: Record<AnomalySeverity, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType; color: string }> = {
  critical: { variant: 'danger', icon: AlertOctagon, color: 'text-danger-400' },
  high: { variant: 'danger', icon: AlertTriangle, color: 'text-danger-400' },
  medium: { variant: 'warning', icon: AlertCircle, color: 'text-warning-400' },
  low: { variant: 'neutral', icon: Activity, color: 'text-gray-400' },
};

const statusConfig: Record<string, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType; label: string }> = {
  open: { variant: 'danger', icon: AlertCircle, label: 'Open' },
  investigating: { variant: 'warning', icon: Eye, label: 'Investigating' },
  resolved: { variant: 'success', icon: CheckCircle, label: 'Resolved' },
  false_positive: { variant: 'neutral', icon: XCircle, label: 'False Positive' },
};

const typeLabels: Record<string, string> = {
  unusual_access_time: 'Unusual Access Time',
  unusual_location: 'Unusual Location',
  privileged_escalation: 'Privileged Escalation',
  bulk_data_access: 'Bulk Data Access',
  command_injection: 'Command Injection',
  ransomware_indicators: 'Ransomware Indicators',
  impossible_travel: 'Impossible Travel',
  account_takeover: 'Account Takeover',
  credential_theft: 'Credential Theft',
  excessive_failed_logins: 'Excessive Failed Logins',
};

export interface AnomalyTableColumn {
  id: string;
  label: string;
  sortable?: boolean;
  width?: string;
}

export const defaultAnomalyColumns: AnomalyTableColumn[] = [
  { id: 'severity', label: 'Severity', sortable: true, width: '100px' },
  { id: 'type', label: 'Type', sortable: true },
  { id: 'title', label: 'Title', sortable: true },
  { id: 'user', label: 'User', width: '150px' },
  { id: 'target', label: 'Target', width: '150px' },
  { id: 'detected_at', label: 'Detected', sortable: true, width: '160px' },
  { id: 'confidence', label: 'Confidence', width: '100px' },
  { id: 'status', label: 'Status', sortable: true, width: '120px' },
  { id: 'actions', label: 'Actions', width: '100px' },
];

interface AnomalyTableProps {
  anomalies: AnomalyDetection[];
  isLoading?: boolean;
  onViewDetail: (anomaly: AnomalyDetection) => void;
  onQuickStatusUpdate?: (anomaly: AnomalyDetection, status: string) => void;
  selectedIds?: Set<string>;
  onSelect?: (id: string, selected: boolean) => void;
  onSelectAll?: (selected: boolean) => void;
  columns?: AnomalyTableColumn[];
  emptyMessage?: string;
}

export const AnomalyTable: React.FC<AnomalyTableProps> = ({
  anomalies,
  isLoading = false,
  onViewDetail,
  onQuickStatusUpdate,
  selectedIds,
  onSelect,
  onSelectAll,
  columns = defaultAnomalyColumns,
  emptyMessage = 'No anomalies found',
}) => {
  const [hoveredRow, setHoveredRow] = React.useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="flex min-h-[300px] items-center justify-center">
        <div className="text-center">
          <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent" />
          <p className="mt-3 text-sm text-gray-400">Loading anomalies...</p>
        </div>
      </div>
    );
  }

  if (anomalies.length === 0) {
    return (
      <div className="flex min-h-[300px] flex-col items-center justify-center text-center">
        <Search className="h-12 w-12 text-gray-600" />
        <p className="mt-3 text-lg font-medium text-gray-300">{emptyMessage}</p>
        <p className="mt-1 text-sm text-gray-500">Try adjusting your filters or date range</p>
      </div>
    );
  }

  const allSelected = selectedIds && anomalies.length > 0 && anomalies.every(a => selectedIds.has(a.id)) || false;
  const someSelected = selectedIds ? anomalies.some(a => selectedIds.has(a.id)) : false;

  const renderColumn = (anomaly: AnomalyDetection, column: AnomalyTableColumn) => {
    switch (column.id) {
      case 'severity':
        return renderSeverityCell(anomaly);
      case 'type':
        return renderTypeCell(anomaly);
      case 'title':
        return renderTitleCell(anomaly);
      case 'user':
        return renderUserCell(anomaly);
      case 'target':
        return renderTargetCell(anomaly);
      case 'detected_at':
        return renderDetectedAtCell(anomaly);
      case 'confidence':
        return renderConfidenceCell(anomaly);
      case 'status':
        return renderStatusCell(anomaly);
      case 'actions':
        return renderActionsCell(anomaly);
      default:
        return null;
    }
  };

  const renderSeverityCell = (anomaly: AnomalyDetection) => {
    const config = severityConfig[anomaly.severity];
    const Icon = config.icon;

    return (
      <div className="flex items-center gap-2">
        <div className={clsx('rounded-lg p-1.5', {
          'bg-danger-500/20 text-danger-400': anomaly.severity === 'critical' || anomaly.severity === 'high',
          'bg-warning-500/20 text-warning-400': anomaly.severity === 'medium',
          'bg-gray-500/20 text-gray-400': anomaly.severity === 'low',
        })}>
          <Icon className="h-4 w-4" />
        </div>
        <span className="text-sm font-medium capitalize text-white">{anomaly.severity}</span>
      </div>
    );
  };

  const renderTypeCell = (anomaly: AnomalyDetection) => {
    return (
      <span className="text-sm text-gray-300">
        {typeLabels[anomaly.type] || anomaly.type}
      </span>
    );
  };

  const renderTitleCell = (anomaly: AnomalyDetection) => {
    return (
      <div className="max-w-md">
        <p className="text-sm font-medium text-white">{anomaly.title}</p>
        <p className="mt-0.5 line-clamp-1 text-xs text-gray-500">{anomaly.description}</p>
      </div>
    );
  };

  const renderUserCell = (anomaly: AnomalyDetection) => {
    return (
      <div className="flex items-center gap-2">
        <User className="h-3.5 w-3.5 text-gray-500" />
        {anomaly.user_name ? (
          <span className="text-sm text-gray-300">{anomaly.user_name}</span>
        ) : (
          <span className="text-sm text-gray-600">N/A</span>
        )}
      </div>
    );
  };

  const renderTargetCell = (anomaly: AnomalyDetection) => {
    return (
      <div className="flex items-center gap-2">
        <Server className="h-3.5 w-3.5 text-gray-500" />
        {anomaly.target_name ? (
          <span className="text-sm text-gray-300">{anomaly.target_name}</span>
        ) : (
          <span className="text-sm text-gray-600">N/A</span>
        )}
      </div>
    );
  };

  const renderDetectedAtCell = (anomaly: AnomalyDetection) => {
    return (
      <div className="flex items-center gap-2">
        <Clock className="h-3.5 w-3.5 text-gray-500" />
        <span className="text-sm text-gray-400">
          {format(new Date(anomaly.detected_at), 'MMM d, HH:mm')}
        </span>
      </div>
    );
  };

  const renderConfidenceCell = (anomaly: AnomalyDetection) => {
    const confidence = Math.round(anomaly.confidence_score * 100);
    const barColor =
      confidence >= 80 ? 'bg-danger-500' :
      confidence >= 60 ? 'bg-warning-500' :
      'bg-success-500';

    return (
      <div className="w-full">
        <div className="flex items-center justify-between">
          <span className="text-sm text-gray-400">{confidence}%</span>
        </div>
        <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-800">
          <div
            className={clsx('h-full rounded-full transition-all', barColor)}
            style={{ width: `${confidence}%` }}
          />
        </div>
      </div>
    );
  };

  const renderStatusCell = (anomaly: AnomalyDetection) => {
    const config = statusConfig[anomaly.status] || statusConfig.open;
    const Icon = config.icon;

    return (
      <div className="flex items-center gap-1.5">
        <Icon className="h-3.5 w-3.5 text-gray-500" />
        <Badge variant={config.variant} className="text-xs">{config.label}</Badge>
      </div>
    );
  };

  const renderActionsCell = (anomaly: AnomalyDetection) => {
    return (
      <div className="flex items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          leftIcon={<Eye className="h-4 w-4" />}
          onClick={() => onViewDetail(anomaly)}
        >
          View
        </Button>
      </div>
    );
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="border-b border-gray-800">
            {onSelect && (
              <th className="w-12 px-4 py-3 text-left">
                <input
                  type="checkbox"
                  checked={allSelected}
                  ref={(input) => {
                    if (input) {
                      input.indeterminate = someSelected && !allSelected;
                    }
                  }}
                  onChange={(e) => onSelectAll?.(e.target.checked)}
                  className="rounded border-gray-700 bg-gray-800 text-primary-500 focus:ring-2 focus:ring-primary-500"
                />
              </th>
            )}
            {columns.map((column) => (
              <th
                key={column.id}
                style={{ width: column.width }}
                className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-800">
          {anomalies.map((anomaly) => (
            <tr
              key={anomaly.id}
              className={clsx(
                'transition-colors hover:bg-gray-800/50',
                hoveredRow === anomaly.id && 'bg-gray-800/50'
              )}
              onMouseEnter={() => setHoveredRow(anomaly.id)}
              onMouseLeave={() => setHoveredRow(null)}
            >
              {onSelect && (
                <td className="px-4 py-3">
                  <input
                    type="checkbox"
                    checked={selectedIds?.has(anomaly.id) || false}
                    onChange={(e) => onSelect(anomaly.id, e.target.checked)}
                    className="rounded border-gray-700 bg-gray-800 text-primary-500 focus:ring-2 focus:ring-primary-500"
                  />
                </td>
              )}
              {columns.map((column) => (
                <td
                  key={column.id}
                  className={clsx(
                    'px-4 py-3',
                    column.id === 'actions' && 'text-right'
                  )}
                >
                  {renderColumn(anomaly, column)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default AnomalyTable;
