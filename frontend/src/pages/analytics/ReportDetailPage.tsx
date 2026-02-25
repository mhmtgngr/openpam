<<<<<<< HEAD
import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft,
  FileText,
  Download,
  Share,
  Trash2,
  Calendar,
  CheckCircle,
  XCircle,
  AlertCircle,
  RefreshCw,
  History,
  Shield,
  TrendingUp,
  AlertTriangle,
  Eye,
  EyeOff,
  ChevronDown,
  ChevronRight,
  Copy,
} from 'lucide-react';
import { reportsApi, reportSnapshotsApi } from '@/api/reports';
import { ReportDistributionDialog } from '@/components/analytics/ReportDistributionDialog';
import { Card, Badge, Button } from '@/components/common';
import { LoadingState } from '@/components/common';
import type { Report, ReportSnapshot } from '@/types/reports';
import { format } from 'date-fns';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const statusColors: Record<string, string> = {
  completed: 'text-success-400 bg-success-400/10',
  generating: 'text-primary-400 bg-primary-400/10',
  failed: 'text-danger-400 bg-danger-400/10',
  scheduled: 'text-warning-400 bg-warning-400/10',
  pending: 'text-gray-400 bg-gray-400/10',
};

const severityColors: Record<string, string> = {
  critical: 'text-danger-400 bg-danger-400/10 border-danger-400/30',
  high: 'text-orange-400 bg-orange-400/10 border-orange-400/30',
  medium: 'text-warning-400 bg-warning-400/10 border-warning-400/30',
  low: 'text-blue-400 bg-blue-400/10 border-blue-400/30',
  info: 'text-gray-400 bg-gray-400/10 border-gray-400/30',
};

const controlStatusColors: Record<string, string> = {
  compliant: 'text-success-400 bg-success-400/10',
  non_compliant: 'text-danger-400 bg-danger-400/10',
  partial: 'text-warning-400 bg-warning-400/10',
  not_applicable: 'text-gray-400 bg-gray-400/10',
};

=======
/**
 * ReportDetailPage - Report detail page showing snapshot status and download options
 * Displays comprehensive information about a generated report
 */

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  ArrowLeft,
  Download,
  Trash2,
  Share2,
  RefreshCw,
  FileText,
  Calendar,
  User,
  Clock,
  HardDrive,
  Settings,
  AlertCircle,
  CheckCircle,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Button } from '@/components/common';
import { Badge } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { LoadingState } from '@/components/common';
import { EmptyState } from '@/components/common';
import { ReportViewer } from '@/components/common/ReportViewer';
import { toast } from 'react-hot-toast';
import type { ReportSnapshot } from '@/types/reports';
import clsx from 'clsx';

>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
export const ReportDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
<<<<<<< HEAD
  const [distributionDialogOpen, setDistributionDialogOpen] = useState(false);
  const [showSnapshots, setShowSnapshots] = useState(false);
  const [expandedControls, setExpandedControls] = useState<Set<string>>(new Set());
  const [redacted, setRedacted] = useState(true);

  const { data: report, isLoading } = useQuery({
    queryKey: ['report', id],
    queryFn: () => reportsApi.get(id!),
    enabled: !!id,
  });

  const { data: snapshotsData } = useQuery({
    queryKey: ['reportSnapshots', id],
    queryFn: () => reportSnapshotsApi.list(id!),
    enabled: !!id && showSnapshots,
  });

  const deleteMutation = useMutation({
    mutationFn: () => reportsApi.delete(id!),
    onSuccess: () => {
      toast.success('Report deleted successfully');
      navigate('/reports');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to delete report');
    },
  });

  const handleDownload = () => {
    if (!report) return;

    const downloadUrl = reportsApi.download(report.id);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = `${report.name.replace(/\s+/g, '_')}.pdf`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    toast.success('Report download started');
  };

  const handleShare = () => {
    setDistributionDialogOpen(true);
  };

  const handleDelete = () => {
    if (!report) return;

    if (confirm(`Are you sure you want to delete report "${report.name}"?`)) {
      deleteMutation.mutate();
    }
  };

  const toggleControl = (controlId: string) => {
    setExpandedControls((prev) => {
      const next = new Set(prev);
      if (next.has(controlId)) {
        next.delete(controlId);
      } else {
        next.add(controlId);
      }
      return next;
    });
  };

  const expandAllControls = () => {
    if (!report) return;
    setExpandedControls(new Set(report.report_data.controls.map((c) => c.id)));
  };

  const collapseAllControls = () => {
    setExpandedControls(new Set());
=======

  const [showViewer, setShowViewer] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  // Query: Report snapshot
  const {
    data: snapshotData,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ['reportSnapshot', id],
    queryFn: () => reportsApi.getSnapshot(id!),
    enabled: !!id,
    refetchInterval: (data) => {
      // Auto-refresh if report is still generating
      return data?.data.status === 'generating' || data?.data.status === 'pending' ? 5000 : false;
    },
  });

  const snapshot = snapshotData?.data as ReportSnapshot | undefined;

  useEffect(() => {
    if (snapshot?.status === 'completed' && showViewer === false) {
      // Auto-stop refresh when completed
      queryClient.invalidateQueries({ queryKey: ['reportSnapshot', id] });
    }
  }, [snapshot?.status, showViewer, id, queryClient]);

  const handleDownload = async () => {
    if (!snapshot) return;

    try {
      const response = await reportsApi.downloadSnapshot(snapshot.id);
      if (response.data.download_url) {
        window.open(response.data.download_url, '_blank');
        toast.success('Download started');
      }
    } catch (error) {
      toast.error('Failed to download report');
    }
  };

  const handleDelete = async () => {
    if (!snapshot) return;

    if (!confirm(`Are you sure you want to delete "${snapshot.report_name || 'this report'}"?`)) {
      return;
    }

    setIsDeleting(true);
    try {
      await reportsApi.deleteSnapshot(snapshot.id);
      toast.success('Report deleted successfully');
      navigate('/analytics/reports');
    } catch (error) {
      toast.error('Failed to delete report');
      setIsDeleting(false);
    }
  };

  const handleShare = async () => {
    if (!snapshot?.file_url) return;

    try {
      await navigator.clipboard.writeText(window.location.origin + snapshot.file_url);
      toast.success('Report link copied to clipboard');
    } catch {
      toast.error('Failed to copy link');
    }
  };

  const formatFileSize = (bytes?: number): string => {
    if (!bytes) return '-';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const getStatusBadge = () => {
    if (!snapshot) return null;

    const statusConfig: Record<string, { variant: 'success' | 'warning' | 'danger' | 'neutral'; icon: React.ElementType; label: string }> = {
      pending: { variant: 'neutral', icon: Clock, label: 'Pending' },
      generating: { variant: 'warning', icon: RefreshCw, label: 'Generating' },
      completed: { variant: 'success', icon: CheckCircle, label: 'Completed' },
      failed: { variant: 'danger', icon: AlertCircle, label: 'Failed' },
      expired: { variant: 'neutral', icon: Clock, label: 'Expired' },
      scheduled: { variant: 'neutral', icon: Calendar, label: 'Scheduled' },
    };

    const config = statusConfig[snapshot.status];
    const Icon = config.icon;

    return (
      <Badge variant={config.variant} className="flex items-center gap-1">
        <Icon className="h-3 w-3" />
        {config.label}
      </Badge>
    );
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
  };

  if (isLoading) {
    return (
<<<<<<< HEAD
      <div className="flex min-h-[400px] items-center justify-center">
        <LoadingState message="Loading report..." />
=======
      <div className="flex min-h-[600px] items-center justify-center">
        <LoadingState message="Loading report details..." />
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
      </div>
    );
  }

<<<<<<< HEAD
  if (!report) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <div className="text-center">
          <FileText className="mx-auto h-12 w-12 text-gray-600" />
          <h3 className="mt-4 text-lg font-medium text-white">Report not found</h3>
          <Button variant="primary" className="mt-4" onClick={() => navigate('/reports')}>
            Go to Reports
          </Button>
        </div>
=======
  if (!snapshot) {
    return (
      <div className="flex min-h-[600px] items-center justify-center">
        <EmptyState title="Report not found" description="The requested report could not be found." />
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
      </div>
    );
  }

<<<<<<< HEAD
  const { report_data, report_metadata } = report;
  const snapshots = snapshotsData?.data || [];

  const maskData = (value: string): string => {
    if (!redacted) return value;
    if (value.length <= 4) return '***';
    return value.substring(0, 2) + '***' + value.substring(value.length - 2);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/reports')}
            className="rounded-lg p-2 text-gray-400 hover:bg-gray-800 hover:text-white"
          >
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-white">{report.name}</h1>
              <Badge
                variant="neutral"
                className={clsx('border capitalize', statusColors[report.status] || statusColors.pending)}
              >
                {report.status.replace('_', ' ')}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-gray-400">
              {report.framework.toUpperCase()} •{' '}
              {format(new Date(report.created_at), 'MMM dd, yyyy HH:mm')}
            </p>
          </div>
        </div>

        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => setRedacted(!redacted)}
            title={redacted ? 'Show sensitive data' : 'Hide sensitive data'}
          >
            {redacted ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            {redacted ? 'Unmask' : 'Mask'}
          </Button>

          {report.status === 'completed' && (
            <>
              <Button variant="secondary" onClick={() => setShowSnapshots(!showSnapshots)}>
                <History className="mr-2 h-4 w-4" />
                Snapshots ({snapshots.length})
              </Button>
              <Button variant="secondary" onClick={handleDownload}>
                <Download className="mr-2 h-4 w-4" />
                Download
              </Button>
              <Button variant="secondary" onClick={handleShare}>
                <Share className="mr-2 h-4 w-4" />
                Share
              </Button>
            </>
          )}

          <Button
            variant="danger"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Report Period */}
      <Card>
        <div className="card-body">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-medium text-gray-400">Report Period</h3>
              <p className="mt-1 text-lg font-semibold text-white">
                {format(new Date(report.period_start), 'MMM dd, yyyy')} —{' '}
                {format(new Date(report.period_end), 'MMM dd, yyyy')}
              </p>
            </div>
            {report_metadata && (
              <div className="text-right">
                <h3 className="text-sm font-medium text-gray-400">Generated</h3>
                <p className="mt-1 text-lg font-semibold text-white">
                  {format(new Date(report_metadata.generated_at), 'MMM dd, yyyy HH:mm')}
                </p>
              </div>
            )}
          </div>
        </div>
      </Card>

      {/* Snapshots Panel */}
      {showSnapshots && (
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
              <History className="h-5 w-5" />
              Report Snapshots
            </h3>
            {snapshots.length === 0 ? (
              <p className="text-gray-400">No snapshots available for this report.</p>
            ) : (
              <div className="space-y-2">
                {snapshots.map((snapshot: ReportSnapshot) => (
                  <div
                    key={snapshot.id}
                    className="flex items-center justify-between rounded-lg border border-gray-700 bg-gray-800/50 p-3"
                  >
                    <div>
                      <p className="text-sm font-medium text-white">
                        {format(new Date(snapshot.generated_at), 'MMM dd, yyyy HH:mm')}
                      </p>
                      <p className="text-xs text-gray-400">
                        {snapshot.file_format.toUpperCase()} •{' '}
                        {snapshot.file_size_bytes
                          ? `${(snapshot.file_size_bytes / 1024).toFixed(1)} KB`
                          : 'N/A'}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const url = reportSnapshotsApi.download(snapshot.id);
                        window.open(url, '_blank');
                      }}
                    >
                      <Download className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Summary Score */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
        <ScoreCard
          title="Overall Score"
          value={report_data.summary.overall_score}
          icon={TrendingUp}
          color="blue"
        />
        <ScoreCard
          title="Compliant"
          value={report_data.summary.compliant_controls}
          total={report_data.summary.compliant_controls + report_data.summary.non_compliant_controls + report_data.summary.partial_controls}
          icon={CheckCircle}
          color="green"
        />
        <ScoreCard
          title="Non-Compliant"
          value={report_data.summary.non_compliant_controls}
          total={report_data.summary.compliant_controls + report_data.summary.non_compliant_controls + report_data.summary.partial_controls}
          icon={XCircle}
          color="red"
        />
        <ScoreCard
          title="Findings"
          value={report_data.summary.total_findings}
          critical={report_data.summary.critical_findings}
          high={report_data.summary.high_findings}
          icon={AlertTriangle}
          color="orange"
        />
      </div>

      {/* Findings */}
      {report_data.findings.length > 0 && (
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-warning-400" />
              Findings ({report_data.findings.length})
            </h3>
            <div className="mt-4 space-y-3">
              {report_data.findings.map((finding) => (
                <div
                  key={finding.id}
                  className="rounded-lg border border-gray-700 bg-gray-800/50 p-4"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Badge
                          variant="neutral"
                          className={clsx('border capitalize', severityColors[finding.severity] || severityColors.info)}
                        >
                          {finding.severity}
                        </Badge>
                        <h4 className="font-medium text-white">{finding.title}</h4>
                      </div>
                      <p className="mt-2 text-sm text-gray-300">{finding.description}</p>
                      {finding.recommendation && (
                        <p className="mt-2 text-sm text-gray-400">
                          <strong>Recommendation:</strong> {finding.recommendation}
                        </p>
                      )}
                      {finding.affected_resources && finding.affected_resources.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {finding.affected_resources.map((resource, idx) => (
                            <span
                              key={idx}
                              className="rounded bg-gray-700 px-2 py-0.5 text-xs text-gray-300"
                            >
                              {redacted ? maskData(resource) : resource}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}

      {/* Controls */}
      <Card>
        <div className="card-body">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-white flex items-center gap-2">
              <Shield className="h-5 w-5" />
              Controls ({report_data.controls.length})
            </h3>
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={expandAllControls}>
                Expand All
              </Button>
              <Button variant="ghost" size="sm" onClick={collapseAllControls}>
                Collapse All
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            {report_data.controls.map((control) => {
              const isExpanded = expandedControls.has(control.id);
              const ControlIcon = isExpanded ? ChevronDown : ChevronRight;

              return (
                <div
                  key={control.id}
                  className="rounded-lg border border-gray-700 bg-gray-800/50 overflow-hidden"
                >
                  <button
                    onClick={() => toggleControl(control.id)}
                    className="flex w-full items-center justify-between p-4 text-left transition-colors hover:bg-gray-800"
                  >
                    <div className="flex items-center gap-3 flex-1">
                      <ControlIcon className="h-4 w-4 text-gray-400" />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-white">{control.name}</span>
                          <Badge
                            variant="neutral"
                            className={clsx('border capitalize text-xs', controlStatusColors[control.status] || controlStatusColors.not_applicable)}
                          >
                            {control.status.replace('_', ' ')}
                          </Badge>
                          <span className="text-sm text-gray-400">
                            Score: {control.score}%
                          </span>
                        </div>
                        {control.description && (
                          <p className="mt-1 text-sm text-gray-400">{control.description}</p>
                        )}
                      </div>
                    </div>
                  </button>

                  {isExpanded && (
                    <div className="border-t border-gray-700 p-4 space-y-3">
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="text-gray-400">Control ID:</span>
                          <span className="ml-2 text-white">{control.id}</span>
                        </div>
                        <div>
                          <span className="text-gray-400">Last Assessed:</span>
                          <span className="ml-2 text-white">
                            {format(new Date(control.last_assessed), 'MMM dd, yyyy')}
                          </span>
                        </div>
                        <div>
                          <span className="text-gray-400">Evidence Count:</span>
                          <span className="ml-2 text-white">{control.evidence_count}</span>
                        </div>
                        {control.category && (
                          <div>
                            <span className="text-gray-400">Category:</span>
                            <span className="ml-2 text-white">{control.category}</span>
                          </div>
                        )}
                      </div>

                      {control.findings && control.findings.length > 0 && (
                        <div>
                          <h4 className="text-sm font-medium text-white mb-2">Findings</h4>
                          <ul className="space-y-1">
                            {control.findings.map((finding, idx) => (
                              <li key={idx} className="flex items-start gap-2 text-sm text-gray-300">
                                <span className="text-danger-400 mt-0.5">•</span>
                                <span>{finding}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </Card>

      {/* Metrics */}
      <Card>
        <div className="card-body">
          <h3 className="text-lg font-semibold text-white mb-4">Metrics</h3>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <MetricItem
              label="Total Sessions"
              value={report_data.metrics.total_sessions}
            />
            <MetricItem
              label="Total Users"
              value={report_data.metrics.total_users}
            />
            <MetricItem
              label="Privileged Sessions"
              value={report_data.metrics.privileged_sessions}
            />
            <MetricItem
              label="Failed Authentications"
              value={report_data.metrics.failed_authentications}
              highlight
            />
            <MetricItem
              label="High-Risk Commands"
              value={report_data.metrics.high_risk_commands}
              highlight
            />
            <MetricItem
              label="Anomalies Detected"
              value={report_data.metrics.anomalies_detected}
              highlight
            />
            <MetricItem
              label="Active Exceptions"
              value={report_data.metrics.exceptions_active}
            />
          </div>
        </div>
      </Card>

      {/* Recommendations */}
      {report_data.recommendations && report_data.recommendations.length > 0 && (
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Recommendations</h3>
            <ul className="space-y-2">
              {report_data.recommendations.map((recommendation, idx) => (
                <li key={idx} className="flex items-start gap-3 text-sm text-gray-300">
                  <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary-500/20 text-primary-400 text-xs">
                    {idx + 1}
                  </span>
                  <span>{recommendation}</span>
                </li>
              ))}
            </ul>
=======
  const isProcessing = snapshot.status === 'pending' || snapshot.status === 'generating';
  const isFailed = snapshot.status === 'failed';
  const isCompleted = snapshot.status === 'completed';
  const isExpired = snapshot.status === 'expired';

  return (
    <div className="space-y-6">
      {/* Back Button & Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/analytics/reports')}
          leftIcon={<ArrowLeft className="h-4 w-4" />}
        >
          Back to Reports
        </Button>
      </div>

      {/* Report Header */}
      <Card>
        <div className="card-body">
          <div className="flex items-start justify-between">
            <div className="flex items-start gap-4">
              <div className={clsx('rounded-lg p-3', {
                'bg-success-500/20 text-success-400': isCompleted,
                'bg-warning-500/20 text-warning-400': isProcessing,
                'bg-danger-500/20 text-danger-400': isFailed,
                'bg-gray-500/20 text-gray-400': isExpired,
              })}>
                <FileText className="h-8 w-8" />
              </div>
              <div>
                <h1 className="text-2xl font-bold text-white">
                  {snapshot.report_name || 'Report'}
                </h1>
                <div className="mt-2 flex flex-wrap items-center gap-3">
                  {getStatusBadge()}
                  <Badge variant="neutral">{snapshot.format.toUpperCase()}</Badge>
                  {snapshot.framework && (
                    <Badge variant="info">{snapshot.framework.toUpperCase()}</Badge>
                  )}
                  <span className="text-sm text-gray-400">
                    ID: {snapshot.id.slice(0, 8)}
                  </span>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => refetch()}
                leftIcon={<RefreshCw className="h-4 w-4" />}
              >
                Refresh
              </Button>
              {isCompleted && snapshot.file_url && (
                <>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleShare}
                    leftIcon={<Share2 className="h-4 w-4" />}
                  >
                    Share
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => setShowViewer(true)}
                    leftIcon={<FileText className="h-4 w-4" />}
                  >
                    View
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={handleDownload}
                    leftIcon={<Download className="h-4 w-4" />}
                  >
                    Download
                  </Button>
                </>
              )}
              <Button
                variant="ghost"
                size="sm"
                onClick={handleDelete}
                isLoading={isDeleting}
                leftIcon={<Trash2 className="h-4 w-4 text-danger-400" />}
              />
            </div>
          </div>
        </div>
      </Card>

      {/* Error Display */}
      {isFailed && snapshot.error_message && (
        <Card className="border-l-4 border-l-danger-500">
          <div className="card-body">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-0.5 h-5 w-5 text-danger-400" />
              <div>
                <h3 className="font-medium text-white">Generation Failed</h3>
                <p className="mt-1 text-sm text-gray-400">{snapshot.error_message}</p>
              </div>
            </div>
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
          </div>
        </Card>
      )}

      {/* Report Metadata */}
<<<<<<< HEAD
      {report_metadata && (
        <Card>
          <div className="card-body">
            <h3 className="text-sm font-semibold text-white mb-3">Report Metadata</h3>
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-400">Version:</span>
                <span className="ml-2 text-white">{report_metadata.version}</span>
              </div>
              <div>
                <span className="text-gray-400">Processing Time:</span>
                <span className="ml-2 text-white">
                  {report_metadata.processing_time_ms}ms
                </span>
              </div>
              <div>
                <span className="text-gray-400">Generated By:</span>
                <span className="ml-2 text-white">{redacted ? maskData(report_metadata.generated_by) : report_metadata.generated_by}</span>
              </div>
              <div>
                <span className="text-gray-400">Tenant ID:</span>
                <span className="ml-2 text-white">{report_metadata.tenant_id}</span>
=======
      <div className="grid gap-6 lg:grid-cols-2">
        {/* Period Information */}
        <Card>
          <CardHeader title="Report Period" />
          <div className="card-body space-y-4">
            <div className="flex items-center gap-3">
              <Calendar className="h-5 w-5 text-gray-500" />
              <div>
                <p className="text-sm text-gray-400">Start Date</p>
                <p className="text-white">
                  {format(new Date(snapshot.config.period_start), 'MMM d, yyyy')}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Calendar className="h-5 w-5 text-gray-500" />
              <div>
                <p className="text-sm text-gray-400">End Date</p>
                <p className="text-white">
                  {format(new Date(snapshot.config.period_end), 'MMM d, yyyy')}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Clock className="h-5 w-5 text-gray-500" />
              <div>
                <p className="text-sm text-gray-400">Duration</p>
                <p className="text-white">
                  {Math.ceil(
                    (new Date(snapshot.config.period_end).getTime() -
                      new Date(snapshot.config.period_start).getTime()) /
                      (1000 * 60 * 60 * 24)
                  )}{' '}
                  days
                </p>
              </div>
            </div>
          </div>
        </Card>

        {/* Generation Information */}
        <Card>
          <CardHeader title="Generation Details" />
          <div className="card-body space-y-4">
            <div className="flex items-center gap-3">
              <User className="h-5 w-5 text-gray-500" />
              <div>
                <p className="text-sm text-gray-400">Generated By</p>
                <p className="text-white">
                  {snapshot.generated_by_user?.display_name ||
                    snapshot.generated_by_user?.email ||
                    snapshot.generated_by}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Clock className="h-5 w-5 text-gray-500" />
              <div>
                <p className="text-sm text-gray-400">Created At</p>
                <p className="text-white">
                  {format(new Date(snapshot.created_at), 'MMM d, yyyy HH:mm:ss')}
                </p>
              </div>
            </div>
            {snapshot.generation_completed_at && (
              <div className="flex items-center gap-3">
                <CheckCircle className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm text-gray-400">Completed At</p>
                  <p className="text-white">
                    {format(new Date(snapshot.generation_completed_at), 'MMM d, yyyy HH:mm:ss')}
                  </p>
                </div>
              </div>
            )}
            {snapshot.metadata?.duration_seconds && (
              <div className="flex items-center gap-3">
                <Clock className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm text-gray-400">Generation Time</p>
                  <p className="text-white">{snapshot.metadata.duration_seconds}s</p>
                </div>
              </div>
            )}
          </div>
        </Card>
      </div>

      {/* File Information */}
      {isCompleted && (
        <Card>
          <CardHeader title="File Information" />
          <div className="card-body">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="flex items-center gap-3">
                <HardDrive className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm text-gray-400">File Size</p>
                  <p className="text-white">{formatFileSize(snapshot.file_size_bytes)}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <FileText className="h-5 w-5 text-gray-500" />
                <div>
                  <p className="text-sm text-gray-400">Format</p>
                  <p className="text-white uppercase">{snapshot.format}</p>
                </div>
              </div>
              {snapshot.expires_at && (
                <div className="flex items-center gap-3">
                  <Clock className="h-5 w-5 text-gray-500" />
                  <div>
                    <p className="text-sm text-gray-400">Expires</p>
                    <p className={clsx('text-white', isExpired && 'text-danger-400')}>
                      {format(new Date(snapshot.expires_at), 'MMM d, yyyy HH:mm')}
                    </p>
                  </div>
                </div>
              )}
            </div>
            {snapshot.metadata?.row_count && (
              <div className="mt-4 pt-4 border-t border-gray-800">
                <div className="flex items-center gap-3">
                  <FileText className="h-5 w-5 text-gray-500" />
                  <div>
                    <p className="text-sm text-gray-400">Records</p>
                    <p className="text-white">{snapshot.metadata.row_count.toLocaleString()}</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Report Configuration */}
      <Card>
        <CardHeader title="Report Configuration" />
        <div className="card-body">
          <div className="space-y-3">
            <div>
              <p className="text-sm text-gray-400">Report Type</p>
              <p className="text-white capitalize">{snapshot.type.replace('_', ' ')}</p>
            </div>
            {snapshot.framework && (
              <div>
                <p className="text-sm text-gray-400">Compliance Framework</p>
                <p className="text-white uppercase">{snapshot.framework}</p>
              </div>
            )}
            {snapshot.config.include_sections && snapshot.config.include_sections.length > 0 && (
              <div>
                <p className="text-sm text-gray-400">Included Sections</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {snapshot.config.include_sections.map((section) => (
                    <Badge key={section} variant="neutral">
                      {section.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            {snapshot.config.filters && Object.keys(snapshot.config.filters).length > 0 && (
              <div>
                <p className="text-sm text-gray-400">Applied Filters</p>
                <pre className="mt-2 overflow-auto rounded-md bg-gray-900 p-3 text-xs text-gray-300">
                  {JSON.stringify(snapshot.config.filters, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      </Card>

      {/* Processing Indicator */}
      {isProcessing && (
        <Card className="border-l-4 border-l-warning-500">
          <div className="card-body">
            <div className="flex items-center gap-4">
              <RefreshCw className="h-6 w-6 animate-spin text-warning-400" />
              <div>
                <h3 className="font-medium text-white">
                  {snapshot.status === 'pending' ? 'Report is queued' : 'Report is being generated'}
                </h3>
                <p className="mt-1 text-sm text-gray-400">
                  This may take a few minutes depending on the report size and complexity.
                  The page will automatically refresh when complete.
                </p>
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
              </div>
            </div>
          </div>
        </Card>
      )}

<<<<<<< HEAD
      {/* Distribution Dialog */}
      <ReportDistributionDialog
        isOpen={distributionDialogOpen}
        onClose={() => setDistributionDialogOpen(false)}
        report={report}
      />
=======
      {/* Report Viewer Modal */}
      {showViewer && (
        <ReportViewer
          report={snapshot}
          onClose={() => setShowViewer(false)}
          onDownload={handleDownload}
        />
      )}
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
    </div>
  );
};

<<<<<<< HEAD
interface ScoreCardProps {
  title: string;
  value: number;
  total?: number;
  critical?: number;
  high?: number;
  icon: React.ElementType;
  color: 'blue' | 'green' | 'red' | 'orange';
}

const ScoreCard: React.FC<ScoreCardProps> = ({
  title,
  value,
  total,
  critical,
  high,
  icon: Icon,
  color,
}) => {
  const colorClasses = {
    blue: 'text-blue-400 bg-blue-400/10',
    green: 'text-success-400 bg-success-400/10',
    red: 'text-danger-400 bg-danger-400/10',
    orange: 'text-warning-400 bg-warning-400/10',
  };

  const getScoreColor = (score: number) => {
    if (score >= 80) return 'text-success-400';
    if (score >= 60) return 'text-warning-400';
    return 'text-danger-400';
  };

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-gray-400">{title}</p>
            <p className={clsx('mt-2 text-2xl font-bold', title === 'Overall Score' ? getScoreColor(value) : 'text-white')}>
              {title === 'Overall Score' ? `${value}%` : total ? `${value}/${total}` : value}
            </p>
            {(critical !== undefined || high !== undefined) && (
              <p className="mt-1 text-xs text-gray-400">
                {critical !== undefined && critical > 0 && (
                  <span className="text-danger-400">{critical} critical</span>
                )}
                {(critical !== undefined && critical > 0 && high !== undefined && high > 0) && ' • '}
                {high !== undefined && high > 0 && (
                  <span className="text-orange-400">{high} high</span>
                )}
              </p>
            )}
          </div>
          <div className={clsx('rounded-lg p-3', colorClasses[color])}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};

interface MetricItemProps {
  label: string;
  value: number;
  highlight?: boolean;
}

const MetricItem: React.FC<MetricItemProps> = ({ label, value, highlight }) => (
  <div className={clsx(
    'rounded-lg border p-3',
    highlight
      ? 'border-warning-500/30 bg-warning-500/10'
      : 'border-gray-700 bg-gray-800/50'
  )}>
    <p className="text-xs text-gray-400">{label}</p>
    <p className={clsx('mt-1 text-lg font-semibold', highlight ? 'text-warning-400' : 'text-white')}>
      {value.toLocaleString()}
    </p>
  </div>
);
=======
export default ReportDetailPage;
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
