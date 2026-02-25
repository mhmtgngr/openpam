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

export const ReportDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

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
  };

  if (isLoading) {
    return (
      <div className="flex min-h-[600px] items-center justify-center">
        <LoadingState message="Loading report details..." />
      </div>
    );
  }

  if (!snapshot) {
    return (
      <div className="flex min-h-[600px] items-center justify-center">
        <EmptyState title="Report not found" description="The requested report could not be found." />
      </div>
    );
  }

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
          </div>
        </Card>
      )}

      {/* Report Metadata */}
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
              </div>
            </div>
          </div>
        </Card>
      )}

      {/* Report Viewer Modal */}
      {showViewer && (
        <ReportViewer
          report={snapshot}
          onClose={() => setShowViewer(false)}
          onDownload={handleDownload}
        />
      )}
    </div>
  );
};

export default ReportDetailPage;
