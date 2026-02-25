import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format, subDays } from 'date-fns';
import {
  AlertTriangle,
  Download,
  RefreshCw,
  ChevronLeft,
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { anomalyApi, type AnomalySummary } from '@/api/anomaly';
import type { AnomalyFilterValues } from '@/components/analytics/AnomalyFilters';
import { AnomalyTable, defaultAnomalyColumns } from '@/components/analytics/AnomalyTable';
import { AnomalyFilters } from '@/components/analytics/AnomalyFilters';
import { Card, CardHeader } from '@/components/common';
import { Badge, Button, Pagination } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import type { AnomalyDetection } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const limit = 20;

interface SummaryCardProps {
  title: string;
  count: number;
  variant: 'danger' | 'warning' | 'success' | 'neutral';
  trend?: number;
}

const SummaryCard: React.FC<SummaryCardProps> = ({ title, count, variant, trend }) => {
  const variantClasses: Record<typeof variant, string> = {
    danger: 'bg-danger-500/10 border-danger-500/30 text-danger-400',
    warning: 'bg-warning-500/10 border-warning-500/30 text-warning-400',
    success: 'bg-success-500/10 border-success-500/30 text-success-400',
    neutral: 'bg-gray-500/10 border-gray-500/30 text-gray-400',
  };

  return (
    <div className={clsx('card border', variantClasses[variant])}>
      <div className="card-body">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium uppercase opacity-80">{title}</p>
          {trend !== undefined && (
            <span className={clsx(
              'text-xs font-medium',
              trend >= 0 ? 'text-success-400' : 'text-danger-400'
            )}>
              {trend >= 0 ? '+' : ''}{trend}%
            </span>
          )}
        </div>
        <p className="mt-2 text-3xl font-bold">{count}</p>
      </div>
    </div>
  );
};

export const AnomalyListPage: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Filter state
  const [filters, setFilters] = useState<AnomalyFilterValues>({
    period: 30,
    severity: '',
    status: '',
    type: '',
    search: '',
  });
  const [offset, setOffset] = useState(0);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Build API params from filters
  const apiParams = React.useMemo(() => {
    const params: Record<string, string | number> = {
      limit,
      offset,
    };

    if (filters.period) {
      params.start_date = subDays(new Date(), filters.period).toISOString();
      params.end_date = new Date().toISOString();
    }
    if (filters.severity) params.severity = filters.severity;
    if (filters.status) params.status = filters.status;
    if (filters.type) params.type = filters.type;
    if (filters.search) params.search = filters.search;

    return params;
  }, [filters, offset]);

  // Fetch anomalies
  const {
    data: anomaliesResponse,
    isLoading: isLoadingAnomalies,
    refetch: refetchAnomalies,
  } = useQuery({
    queryKey: ['anomalies', apiParams],
    queryFn: () => anomalyApi.list(apiParams),
    refetchInterval: 2 * 60 * 1000, // Refresh every 2 minutes
  });

  // Fetch summary
  const { data: summary, isLoading: isLoadingSummary } = useQuery({
    queryKey: ['anomalySummary', { start_date: apiParams.start_date, end_date: apiParams.end_date }],
    queryFn: () => anomalyApi.getSummary({
      start_date: apiParams.start_date as string,
      end_date: apiParams.end_date as string,
    }),
    refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
  });

  // Bulk update mutation
  const bulkUpdateMutation = useMutation({
    mutationFn: (data: { ids: string[]; status: string; notes?: string }) =>
      anomalyApi.bulkUpdate(data.ids, { status: data.status as any, resolution_notes: data.notes }),
    onSuccess: (response) => {
      toast.success(`Updated ${response.data.updated} anomalies`);
      setSelectedIds(new Set());
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
      queryClient.invalidateQueries({ queryKey: ['anomalySummary'] });
    },
    onError: () => {
      toast.error('Failed to update anomalies');
    },
  });

  // Export mutation
  const exportMutation = useMutation({
    mutationFn: (format: 'csv' | 'json') =>
      anomalyApi.export({ ...apiParams, format }),
    onSuccess: (response) => {
      toast.success(`Export ready: ${response.data.download_url}`);
      // Optionally trigger download
      window.open(response.data.download_url, '_blank');
    },
    onError: () => {
      toast.error('Failed to export anomalies');
    },
  });

  const handleFilterChange = (newFilters: AnomalyFilterValues) => {
    setFilters(newFilters);
    setOffset(0); // Reset to first page when filters change
  };

  const handleResetFilters = () => {
    setFilters({
      period: 30,
      severity: '',
      status: '',
      type: '',
      search: '',
    });
    setOffset(0);
  };

  const handleViewDetail = (anomaly: AnomalyDetection) => {
    navigate(`/analytics/anomalies/${anomaly.id}`);
  };

  const handleSelectAll = (selected: boolean) => {
    if (selected) {
      setSelectedIds(new Set(anomaliesResponse?.data?.data?.map((a: AnomalyDetection) => a.id) || []));
    } else {
      setSelectedIds(new Set());
    }
  };

  const handleSelect = (id: string, selected: boolean) => {
    const newSelected = new Set(selectedIds);
    if (selected) {
      newSelected.add(id);
    } else {
      newSelected.delete(id);
    }
    setSelectedIds(newSelected);
  };

  const handleBulkStatusUpdate = (status: string) => {
    if (selectedIds.size === 0) {
      toast.error('Please select at least one anomaly');
      return;
    }
    bulkUpdateMutation.mutate({
      ids: Array.from(selectedIds),
      status,
    });
  };

  const handleExport = (format: 'csv' | 'json') => {
    exportMutation.mutate(format);
  };

  const anomalies = anomaliesResponse?.data?.data || [];
  const total = anomaliesResponse?.data?.pagination?.total || 0;
  const hasMore = anomaliesResponse?.data?.pagination?.has_more || false;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            {navigate.length > 0 && (
              <button
                onClick={() => navigate(-1)}
                className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-800 hover:text-white"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
            )}
            <div>
              <h1 className="text-2xl font-bold text-white">Anomaly Detection</h1>
              <p className="mt-1 text-sm text-gray-400">
                Monitor and investigate security anomalies
              </p>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="secondary"
            size="sm"
            leftIcon={<Download className="h-4 w-4" />}
            onClick={() => handleExport('csv')}
            disabled={exportMutation.isPending || anomalies.length === 0}
          >
            Export
          </Button>
          <Button
            variant="secondary"
            size="sm"
            leftIcon={<RefreshCw className={clsx('h-4 w-4', bulkUpdateMutation.isPending && 'animate-spin')} />}
            onClick={() => refetchAnomalies()}
            disabled={isLoadingAnomalies}
          >
            Refresh
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      {!isLoadingSummary && summary && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-6">
          <SummaryCard title="Total" count={summary.data.total} variant="neutral" />
          <SummaryCard title="Critical" count={summary.data.by_severity.critical || 0} variant="danger" />
          <SummaryCard title="High" count={summary.data.by_severity.high || 0} variant="danger" />
          <SummaryCard title="Medium" count={summary.data.by_severity.medium || 0} variant="warning" />
          <SummaryCard title="Low" count={summary.data.by_severity.low || 0} variant="neutral" />
          <SummaryCard
            title="Resolved This Period"
            count={summary.data.resolved_this_period}
            variant="success"
          />
        </div>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <AnomalyFilters
            filters={filters}
            onChange={handleFilterChange}
            onReset={handleResetFilters}
            isLoading={isLoadingAnomalies}
          />
        </div>
      </Card>

      {/* Bulk Actions */}
      {selectedIds.size > 0 && (
        <div className="flex items-center justify-between rounded-lg bg-primary-500/10 border border-primary-500/30 px-4 py-3">
          <p className="text-sm text-gray-300">
            <span className="font-medium text-white">{selectedIds.size}</span> anomalies selected
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => handleBulkStatusUpdate('investigating')}
              disabled={bulkUpdateMutation.isPending}
            >
              Mark Investigating
            </Button>
            <Button
              variant="success"
              size="sm"
              onClick={() => handleBulkStatusUpdate('resolved')}
              disabled={bulkUpdateMutation.isPending}
            >
              Mark Resolved
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => handleBulkStatusUpdate('false_positive')}
              disabled={bulkUpdateMutation.isPending}
            >
              False Positive
            </Button>
          </div>
        </div>
      )}

      {/* Anomalies Table */}
      <Card>
        <CardHeader
          title={`Anomalies${total > 0 ? ` (${total.toLocaleString()})` : ''}`}
          subtitle="Detected security anomalies requiring investigation"
        />
        {isLoadingAnomalies ? (
          <div className="card-body">
            <LoadingState message="Loading anomalies..." />
          </div>
        ) : anomalies.length === 0 ? (
          <div className="card-body">
            <EmptyState
              title="No anomalies found"
              description={Object.values(filters).some(v => v) ? 'Try adjusting your filters' : 'No anomalies have been detected in the selected time period'}
            />
          </div>
        ) : (
          <>
            <AnomalyTable
              anomalies={anomalies}
              isLoading={isLoadingAnomalies}
              onViewDetail={handleViewDetail}
              selectedIds={selectedIds}
              onSelect={handleSelect}
              onSelectAll={handleSelectAll}
              columns={defaultAnomalyColumns}
            />
            {total > limit && (
              <div className="card-footer">
                <Pagination
                  total={total}
                  limit={limit}
                  offset={offset}
                  onPageChange={setOffset}
                />
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
};

export default AnomalyListPage;
