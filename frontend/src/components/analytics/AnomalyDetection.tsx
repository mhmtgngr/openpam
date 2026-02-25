import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format, subDays } from 'date-fns';
import {
  AlertTriangle,
  Search,
  Filter,
  Calendar,
  Eye,
  CheckCircle,
  XCircle,
  AlertCircle,
  User,
  Server,
  Clock,
  Activity,
  RefreshCw,
  Ban,
  AlertOctagon,
} from 'lucide-react';
import { complianceApi, type AnomalyListParams, type AnomalySeverity, type AnomalyType } from '@/api/compliance';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge, StatusBadge } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Button } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import { Pagination } from '@/components/common';
import { Modal } from '@/components/common';
import { Textarea } from '@/components/common';
import type { AnomalyDetection as AnomalyDetectionType, AnomalyIndicator } from '@/types';
import clsx from 'clsx';

const periodOptions = [
  { value: '7', label: 'Last 7 days' },
  { value: '14', label: 'Last 14 days' },
  { value: '30', label: 'Last 30 days' },
  { value: '90', label: 'Last 90 days' },
];

const severityOptions = [
  { value: '', label: 'All Severities' },
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
];

const statusOptions = [
  { value: '', label: 'All Statuses' },
  { value: 'open', label: 'Open' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'false_positive', label: 'False Positive' },
];

const typeOptions: Array<{ value: AnomalyType; label: string }> = [
  { value: 'unusual_access_time', label: 'Unusual Access Time' },
  { value: 'unusual_location', label: 'Unusual Location' },
  { value: 'privileged_escalation', label: 'Privileged Escalation' },
  { value: 'bulk_data_access', label: 'Bulk Data Access' },
  { value: 'command_injection', label: 'Command Injection' },
  { value: 'ransomware_indicators', label: 'Ransomware Indicators' },
  { value: 'impossible_travel', label: 'Impossible Travel' },
  { value: 'account_takeover', label: 'Account Takeover' },
  { value: 'credential_theft', label: 'Credential Theft' },
  { value: 'excessive_failed_logins', label: 'Excessive Failed Logins' },
];

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

interface SummaryCardProps {
  title: string;
  count: number;
  variant: 'danger' | 'warning' | 'success' | 'neutral';
}

const SummaryCard: React.FC<SummaryCardProps> = ({ title, count, variant }) => {
  const variantClasses: Record<typeof variant, string> = {
    danger: 'bg-danger-500/10 text-danger-400 border-danger-500/30',
    warning: 'bg-warning-500/10 text-warning-400 border-warning-500/30',
    success: 'bg-success-500/10 text-success-400 border-success-500/30',
    neutral: 'bg-gray-500/10 text-gray-400 border-gray-500/30',
  };

  return (
    <div className={clsx('card border', variantClasses[variant])}>
      <div className="card-body">
        <p className="text-sm font-medium uppercase opacity-80">{title}</p>
        <p className="mt-2 text-3xl font-bold">{count}</p>
      </div>
    </div>
  );
};

interface AnomalyDetailModalProps {
  anomaly: AnomalyDetectionType;
  isOpen: boolean;
  onClose: () => void;
  onUpdate: (id: string, data: { status?: 'open' | 'investigating' | 'resolved' | 'false_positive'; assigned_to?: string; resolution_notes?: string }) => void;
}

const AnomalyDetailModal: React.FC<AnomalyDetailModalProps> = ({ anomaly, isOpen, onClose, onUpdate }) => {
  const [status, setStatus] = useState<'open' | 'investigating' | 'resolved' | 'false_positive'>(anomaly.status);
  const [notes, setNotes] = useState(anomaly.resolution_notes || '');
  const [assignedTo, setAssignedTo] = useState(anomaly.assigned_to || '');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onUpdate(anomaly.id, {
      status,
      resolution_notes: notes,
      assigned_to: assignedTo || undefined,
    });
    onClose();
  };

  const severityInfo = severityConfig[anomaly.severity];
  const statusInfo = statusConfig[anomaly.status];
  const SeverityIcon = severityInfo.icon;
  const StatusIcon = statusInfo.icon;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Anomaly: ${anomaly.title}`} size="lg">
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className={clsx('rounded-lg p-2', {
              'bg-danger-500/20 text-danger-400': anomaly.severity === 'critical' || anomaly.severity === 'high',
              'bg-warning-500/20 text-warning-400': anomaly.severity === 'medium',
              'bg-gray-500/20 text-gray-400': anomaly.severity === 'low',
            })}>
              <SeverityIcon className="h-6 w-6" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-white">{anomaly.title}</h3>
              <p className="text-sm text-gray-400">{anomaly.description}</p>
            </div>
          </div>
          <div className="flex flex-col items-end gap-2">
            <Badge variant={severityInfo.variant}>{anomaly.severity}</Badge>
            <Badge variant={statusInfo.variant}>{statusInfo.label}</Badge>
          </div>
        </div>

        {/* Details */}
        <div className="grid grid-cols-2 gap-4 rounded-lg bg-gray-900/50 p-4">
          <div>
            <p className="text-xs text-gray-500">Detected At</p>
            <p className="text-sm text-white">
              {format(new Date(anomaly.detected_at), 'MMM d, yyyy HH:mm')}
            </p>
          </div>
          <div>
            <p className="text-xs text-gray-500">Confidence Score</p>
            <p className="text-sm text-white">{Math.round(anomaly.confidence_score * 100)}%</p>
          </div>
          {anomaly.user_name && (
            <div>
              <p className="text-xs text-gray-500">User</p>
              <p className="text-sm text-white">{anomaly.user_name}</p>
            </div>
          )}
          {anomaly.target_name && (
            <div>
              <p className="text-xs text-gray-500">Target</p>
              <p className="text-sm text-white">{anomaly.target_name}</p>
            </div>
          )}
        </div>

        {/* Indicators */}
        {anomaly.indicators && anomaly.indicators.length > 0 && (
          <div>
            <h4 className="mb-3 text-sm font-medium text-gray-300">Indicators</h4>
            <div className="space-y-2">
              {anomaly.indicators.map((indicator, idx) => (
                <div key={idx} className="rounded-lg bg-gray-900/50 p-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-white">{indicator.type}</p>
                      <p className="text-xs text-gray-400">{indicator.description}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-lg font-semibold text-white">
                        {typeof indicator.value === 'number' ? indicator.value.toFixed(2) : indicator.value}
                      </p>
                      {indicator.threshold && (
                        <p className="text-xs text-gray-500">Threshold: {indicator.threshold}</p>
                      )}
                    </div>
                  </div>
                  <div className="mt-2">
                    <div className="h-1 w-full overflow-hidden rounded-full bg-gray-800">
                      <div
                        className={clsx('h-full rounded-full', {
                          'bg-danger-500': indicator.confidence > 0.7,
                          'bg-warning-500': indicator.confidence > 0.4 && indicator.confidence <= 0.7,
                          'bg-success-500': indicator.confidence <= 0.4,
                        })}
                        style={{ width: `${indicator.confidence * 100}%` }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-gray-500">
                      Confidence: {Math.round(indicator.confidence * 100)}%
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Update Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          <h4 className="text-sm font-medium text-gray-300">Update Anomaly</h4>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Status</label>
            <Select
              options={statusOptions.filter(o => o.value !== '')}
              value={status}
              onChange={(e) => setStatus(e.target.value as 'open' | 'investigating' | 'resolved' | 'false_positive')}
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Assigned To</label>
            <Input
              placeholder="Email or user ID"
              value={assignedTo}
              onChange={(e) => setAssignedTo(e.target.value)}
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Resolution Notes</label>
            <Textarea
              placeholder="Add investigation notes or resolution details..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>

          <div className="flex justify-end gap-3">
            <Button type="button" variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" variant="primary">
              Update
            </Button>
          </div>
        </form>

        {anomaly.resolved_at && (
          <div className="rounded-lg bg-success-500/10 p-3">
            <p className="text-sm text-success-400">
              Resolved on {format(new Date(anomaly.resolved_at), 'MMM d, yyyy HH:mm')}
            </p>
            {anomaly.resolution_notes && (
              <p className="mt-1 text-xs text-gray-400">{anomaly.resolution_notes}</p>
            )}
          </div>
        )}
      </div>
    </Modal>
  );
};

export const AnomalyDetection: React.FC = () => {
  const [days, setDays] = useState(30);
  const [severity, setSeverity] = useState('');
  const [status, setStatus] = useState('');
  const [type, setType] = useState('');
  const [search, setSearch] = useState('');
  const [offset, setOffset] = useState(0);
  const [selectedAnomaly, setSelectedAnomaly] = useState<AnomalyDetectionType | null>(null);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [detailStatus, setDetailStatus] = useState<'open' | 'investigating' | 'resolved' | 'false_positive'>('open');
  const limit = 20;

  const { data: anomalies, isLoading } = useQuery({
    queryKey: ['anomalies', days, severity, status, type, search, offset, limit],
    queryFn: () =>
      complianceApi.getAnomalies({
        start_date: subDays(new Date(), days).toISOString(),
        end_date: new Date().toISOString(),
        severity: severity as AnomalySeverity,
        status,
        type: type as AnomalyType,
        search,
        limit,
        offset,
      }),
    refetchInterval: 2 * 60 * 1000, // Refresh every 2 minutes
  });

  const { data: summary } = useQuery({
    queryKey: ['anomalySummary', days],
    queryFn: () =>
      complianceApi.getAnomalySummary({
        start_date: subDays(new Date(), days).toISOString(),
        end_date: new Date().toISOString(),
      }),
  });

  const handleUpdateAnomaly = async (id: string, data: { status?: 'open' | 'investigating' | 'resolved' | 'false_positive'; assigned_to?: string; resolution_notes?: string }) => {
    try {
      await complianceApi.updateAnomaly(id, data);
      setShowDetailModal(false);
      // Refetch would happen automatically via query invalidation
      window.location.reload();
    } catch {
      // Error handled by interceptor
    }
  };

  const totalAnomalies = anomalies?.pagination?.total || 0;

  return (
    <div className="space-y-6">
      <CardHeader
        title="Anomaly Detection"
        subtitle="Detect and investigate security anomalies"
        action={
          <div className="flex items-center gap-3">
            <Select
              options={periodOptions}
              value={String(days)}
              onChange={(e) => {
                setDays(Number(e.target.value));
                setOffset(0);
              }}
              className="w-32"
            />
          </div>
        }
      />

      {/* Summary Cards */}
      {summary && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <SummaryCard title="Total" count={summary.total} variant="neutral" />
          <SummaryCard title="Critical" count={summary.by_severity.critical || 0} variant="danger" />
          <SummaryCard title="High" count={summary.by_severity.high || 0} variant="danger" />
          <SummaryCard title="Medium" count={summary.by_severity.medium || 0} variant="warning" />
          <SummaryCard title="Resolved This Period" count={summary.resolved_this_period} variant="success" />
        </div>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search anomalies..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={severityOptions}
              value={severity}
              onChange={(e) => {
                setSeverity(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={statusOptions}
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={[{ value: '', label: 'All Types' }, ...typeOptions]}
              value={type}
              onChange={(e) => {
                setType(e.target.value);
                setOffset(0);
              }}
            />
          </div>
        </div>
      </Card>

      {/* Anomalies List */}
      <Card>
        {isLoading ? (
          <div className="card-body">
            <LoadingState message="Loading anomalies..." />
          </div>
        ) : !anomalies || anomalies.data.length === 0 ? (
          <div className="card-body">
            <EmptyState title="No anomalies found" />
          </div>
        ) : (
          <>
            <div className="divide-y divide-gray-800">
              {anomalies.data.map((anomaly) => {
                const severityInfo = severityConfig[anomaly.severity];
                const statusInfo = statusConfig[anomaly.status];
                const SeverityIcon = severityInfo.icon;
                const StatusIcon = statusInfo.icon;

                return (
                  <div
                    key={anomaly.id}
                    className="flex items-start gap-4 p-4 hover:bg-gray-800/30 cursor-pointer"
                    onClick={() => {
                      setSelectedAnomaly(anomaly);
                      setShowDetailModal(true);
                    }}
                  >
                    <div className={clsx('rounded-lg p-2', {
                      'bg-danger-500/20 text-danger-400': anomaly.severity === 'critical' || anomaly.severity === 'high',
                      'bg-warning-500/20 text-warning-400': anomaly.severity === 'medium',
                      'bg-gray-500/20 text-gray-400': anomaly.severity === 'low',
                    })}>
                      <SeverityIcon className="h-5 w-5" />
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <h4 className="font-medium text-white">{anomaly.title}</h4>
                          <p className="mt-1 text-sm text-gray-400 line-clamp-2">{anomaly.description}</p>
                        </div>
                        <div className="flex flex-col items-end gap-2">
                          <Badge variant={severityInfo.variant}>{anomaly.severity}</Badge>
                          <div className="flex items-center gap-1">
                            <StatusIcon className="h-3 w-3 text-gray-500" />
                            <Badge variant={statusInfo.variant} className="text-xs">{statusInfo.label}</Badge>
                          </div>
                        </div>
                      </div>

                      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-gray-500">
                        {anomaly.user_name && (
                          <span className="flex items-center gap-1">
                            <User className="h-3 w-3" />
                            {anomaly.user_name}
                          </span>
                        )}
                        {anomaly.target_name && (
                          <span className="flex items-center gap-1">
                            <Server className="h-3 w-3" />
                            {anomaly.target_name}
                          </span>
                        )}
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {format(new Date(anomaly.detected_at), 'MMM d, HH:mm')}
                        </span>
                        <span>
                          Confidence: {Math.round(anomaly.confidence_score * 100)}%
                        </span>
                      </div>
                    </div>

                    <Button
                      variant="secondary"
                      size="sm"
                      leftIcon={<Eye className="h-4 w-4" />}
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedAnomaly(anomaly);
                        setShowDetailModal(true);
                      }}
                    >
                      View
                    </Button>
                  </div>
                );
              })}
            </div>

            {totalAnomalies > limit && (
              <div className="card-footer">
                <Pagination
                  total={totalAnomalies}
                  limit={limit}
                  offset={offset}
                  onPageChange={setOffset}
                />
              </div>
            )}
          </>
        )}
      </Card>

      {/* Detail Modal */}
      {selectedAnomaly && (
        <AnomalyDetailModal
          anomaly={selectedAnomaly}
          isOpen={showDetailModal}
          onClose={() => {
            setShowDetailModal(false);
            setSelectedAnomaly(null);
          }}
          onUpdate={handleUpdateAnomaly}
        />
      )}
    </div>
  );
};
