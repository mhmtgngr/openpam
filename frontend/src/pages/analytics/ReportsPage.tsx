/**
 * ReportsPage - Main reports listing page with filters and generation trigger
 * Displays all generated reports with filtering, search, and pagination
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  FileText,
  Plus,
  Filter,
  Search,
  RefreshCw,
  Trash2,
  Calendar,
  Download,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import type { ReportSnapshot, ReportType, ReportStatus, ReportFormat } from '@/types/reports';
import { format } from 'date-fns';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const reportTypeOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Types' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'session_activity', label: 'Session Activity' },
  { value: 'command_analysis', label: 'Command Analysis' },
  { value: 'user_access', label: 'User Access' },
  { value: 'anomaly_summary', label: 'Anomaly Summary' },
  { value: 'audit_trail', label: 'Audit Trail' },
  { value: 'credential_usage', label: 'Credential Usage' },
];

const statusOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Statuses' },
  { value: 'completed', label: 'Completed' },
  { value: 'generating', label: 'Generating' },
  { value: 'pending', label: 'Pending' },
  { value: 'failed', label: 'Failed' },
  { value: 'expired', label: 'Expired' },
];

const formatOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Formats' },
  { value: 'pdf', label: 'PDF' },
  { value: 'csv', label: 'CSV' },
  { value: 'json', label: 'JSON' },
  { value: 'xlsx', label: 'Excel' },
  { value: 'html', label: 'HTML' },
];

const sortOptions: Array<{ value: string; label: string }> = [
  { value: 'created_at:desc', label: 'Newest First' },
  { value: 'created_at:asc', label: 'Oldest First' },
  { value: 'period_start:desc', label: 'Period Start (Newest)' },
  { value: 'period_start:asc', label: 'Period Start (Oldest)' },
];

const statusIcons: Record<ReportStatus, React.ElementType> = {
  completed: () => <span className="h-2 w-2 rounded-full bg-success-400" />,
  generating: () => <span className="h-2 w-2 rounded-full bg-primary-400 animate-pulse" />,
  failed: () => <span className="h-2 w-2 rounded-full bg-danger-400" />,
  scheduled: () => <span className="h-2 w-2 rounded-full bg-warning-400" />,
  pending: () => <span className="h-2 w-2 rounded-full bg-gray-400" />,
  expired: () => <span className="h-2 w-2 rounded-full bg-gray-400" />,
  cancelled: () => <span className="h-2 w-2 rounded-full bg-gray-400" />,
};

const statusColors: Record<ReportStatus, string> = {
  completed: 'text-success-400 bg-success-400/10',
  generating: 'text-primary-400 bg-primary-400/10',
  failed: 'text-danger-400 bg-danger-400/10',
  scheduled: 'text-warning-400 bg-warning-400/10',
  pending: 'text-gray-400 bg-gray-400/10',
  expired: 'text-gray-400 bg-gray-400/10',
  cancelled: 'text-gray-400 bg-gray-400/10',
};

export const ReportsPage: React.FC = () => {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<{
    limit?: number;
    offset?: number;
    type?: ReportType;
    status?: ReportStatus;
    format?: ReportFormat;
    search?: string;
    sort_by?: 'created_at' | 'period_start' | 'period_end' | 'type';
    sort_order?: 'asc' | 'desc';
  }>({
    limit: 20,
    offset: 0,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);

  const { data: reportsData, isLoading, refetch } = useQuery({
    queryKey: ['reportSnapshots', filters],
    queryFn: () => reportsApi.listSnapshots(filters),
  });

  const snapshots = reportsData?.data || [];
  const pagination = reportsData?.pagination;

  const handleTypeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilters({ ...filters, type: e.target.value as ReportType | undefined, offset: 0 });
  };

  const handleStatusChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilters({ ...filters, status: e.target.value as ReportStatus | undefined, offset: 0 });
  };

  const handleFormatChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setFilters({ ...filters, format: e.target.value as ReportFormat | undefined, offset: 0 });
  };

  const handleSortChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const [sortBy, sortOrder] = e.target.value.split(':');
    setFilters({
      ...filters,
      sort_by: sortBy as 'created_at' | 'period_start' | 'period_end' | 'type',
      sort_order: sortOrder as 'asc' | 'desc',
      offset: 0,
    });
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setFilters({ ...filters, search: searchQuery || undefined, offset: 0 });
  };

  const handleClearFilters = () => {
    setSearchQuery('');
    setFilters({
      limit: 20,
      offset: 0,
      sort_by: 'created_at',
      sort_order: 'desc',
    });
  };

  const handleViewReport = (report: ReportSnapshot) => {
    navigate(`/reports/${report.id}`);
  };

  const handleDownload = async (report: ReportSnapshot) => {
    try {
      const response = await reportsApi.downloadSnapshot(report.id);
      const downloadUrl = (response as any).download_url || (response as any).data?.download_url;
      if (downloadUrl) {
        window.open(downloadUrl, '_blank');
        toast.success('Download started');
      }
    } catch (error) {
      toast.error('Failed to download report');
    }
  };

  const handleDelete = async (report: ReportSnapshot) => {
    if (!confirm(`Are you sure you want to delete "${report.report_name || 'this report'}"?`)) {
      return;
    }
    try {
      await reportsApi.deleteSnapshot(report.id);
      toast.success('Report deleted successfully');
      refetch();
    } catch (error) {
      toast.error('Failed to delete report');
    }
  };

  const activeFilterCount =
    (filters.type ? 1 : 0) +
    (filters.status ? 1 : 0) +
    (filters.format ? 1 : 0) +
    (filters.search ? 1 : 0);

  const statusCounts = snapshots.reduce(
    (acc, report) => {
      acc[report.status] = (acc[report.status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const handleNextPage = () => {
    if (pagination?.has_more) {
      setFilters({ ...filters, offset: (filters.offset || 0) + (filters.limit || 20) });
    }
  };

  const handlePrevPage = () => {
    if ((filters.offset || 0) > 0) {
      setFilters({ ...filters, offset: Math.max(0, (filters.offset || 0) - (filters.limit || 20)) });
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Reports</h1>
          <p className="mt-1 text-sm text-gray-400">
            Generate, view, and manage compliance and analytics reports
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="secondary"
            onClick={() => refetch()}
            className="flex items-center gap-2"
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button
            variant="primary"
            onClick={() => navigate('/analytics/reports/generate')}
          >
            Generate Report
          </Button>
        </div>
      </div>

      {/* Status Summary */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
        <Card className="cursor-pointer transition-colors hover:bg-gray-800/50">
          <div className="card-body py-3 text-center">
            <p className="text-2xl font-bold text-white">{pagination?.total || snapshots.length}</p>
            <p className="text-xs text-gray-400">Total Reports</p>
          </div>
        </Card>
        <Card
          className={clsx(
            'cursor-pointer transition-colors hover:bg-gray-800/50',
            !filters.status && 'border-l-2 border-l-success-500'
          )}
          onClick={() => setFilters({ ...filters, status: 'completed' as ReportStatus, offset: 0 })}
        >
          <div className="card-body py-3 text-center">
            <p className="text-2xl font-bold text-success-400">{statusCounts.completed || 0}</p>
            <p className="text-xs text-gray-400">Completed</p>
          </div>
        </Card>
        <Card
          className={clsx(
            'cursor-pointer transition-colors hover:bg-gray-800/50',
            filters.status === 'generating' && 'border-l-2 border-l-warning-500'
          )}
          onClick={() => setFilters({ ...filters, status: 'generating' as ReportStatus, offset: 0 })}
        >
          <div className="card-body py-3 text-center">
            <p className="text-2xl font-bold text-warning-400">
              {(statusCounts.generating || 0) + (statusCounts.pending || 0)}
            </p>
            <p className="text-xs text-gray-400">In Progress</p>
          </div>
        </Card>
        <Card
          className={clsx(
            'cursor-pointer transition-colors hover:bg-gray-800/50',
            filters.status === 'failed' && 'border-l-2 border-l-danger-500'
          )}
          onClick={() => setFilters({ ...filters, status: 'failed' as ReportStatus, offset: 0 })}
        >
          <div className="card-body py-3 text-center">
            <p className="text-2xl font-bold text-danger-400">{statusCounts.failed || 0}</p>
            <p className="text-xs text-gray-400">Failed</p>
          </div>
        </Card>
        <Card
          className={clsx(
            'cursor-pointer transition-colors hover:bg-gray-800/50',
            filters.status === 'expired' && 'border-l-2 border-l-gray-500'
          )}
          onClick={() => setFilters({ ...filters, status: 'expired' as ReportStatus, offset: 0 })}
        >
          <div className="card-body py-3 text-center">
            <p className="text-2xl font-bold text-gray-400">{statusCounts.expired || 0}</p>
            <p className="text-xs text-gray-400">Expired</p>
          </div>
        </Card>
      </div>

      {/* Filters */}
      <Card>
        <div className="card-body">
          <form onSubmit={handleSearch} className="space-y-4">
            {/* Search and Toggle */}
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
                <Input
                  type="text"
                  placeholder="Search reports..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={() => setShowFilters(!showFilters)}
                className="flex items-center gap-2"
              >
                <Filter className="h-4 w-4" />
                Filters
                {activeFilterCount > 0 && (
                  <Badge variant="info" className="ml-2">
                    {activeFilterCount}
                  </Badge>
                )}
              </Button>
            </div>

            {/* Expanded Filters */}
            {showFilters && (
              <div className="grid gap-4 border-t border-gray-800 pt-4 sm:grid-cols-2 lg:grid-cols-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Report Type</label>
                  <select
                    value={filters.type || ''}
                    onChange={handleTypeChange}
                    className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none"
                  >
                    {reportTypeOptions.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Status</label>
                  <select
                    value={filters.status || ''}
                    onChange={handleStatusChange}
                    className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none"
                  >
                    {statusOptions.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Format</label>
                  <select
                    value={filters.format || ''}
                    onChange={handleFormatChange}
                    className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none"
                  >
                    {formatOptions.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Sort By</label>
                  <select
                    value={`${filters.sort_by}:${filters.sort_order}`}
                    onChange={handleSortChange}
                    className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white text-sm focus:border-primary-500 focus:outline-none"
                  >
                    {sortOptions.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}

            {/* Filter Actions */}
            {activeFilterCount > 0 && (
              <div className="flex items-center justify-between border-t border-gray-800 pt-4">
                <p className="text-sm text-gray-400">
                  {activeFilterCount} filter(s) applied
                </p>
                <Button type="button" variant="ghost" size="sm" onClick={handleClearFilters}>
                  Clear All
                </Button>
              </div>
            )}
          </form>
        </div>
      </Card>

      {/* Reports List */}
      {isLoading ? (
        <div className="flex min-h-[400px] items-center justify-center">
          <LoadingState message="Loading reports..." />
        </div>
      ) : snapshots.length === 0 ? (
        <Card>
          <div className="card-body py-12 text-center">
            <FileText className="mx-auto h-12 w-12 text-gray-600" />
            <h3 className="mt-4 text-sm font-medium text-white">No reports found</h3>
            <p className="mt-2 text-sm text-gray-400">
              {activeFilterCount > 0
                ? 'Try adjusting your filters or search query'
                : 'Get started by generating your first report'}
            </p>
            {activeFilterCount === 0 && (
              <div className="mt-6">
                <Button
                  variant="primary"
                  onClick={() => navigate('/reports/generate')}
                  className="flex items-center gap-2"
                >
                  <Plus className="h-4 w-4" />
                  Generate Report
                </Button>
              </div>
            )}
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-gray-400">
              Showing {snapshots.length} of {pagination?.total || snapshots.length} reports
            </p>
          </div>

          <div className="space-y-3">
            {snapshots.map((report) => {
              const StatusIcon = statusIcons[report.status];
              const isGenerating = report.status === 'generating' || report.status === 'pending';

              return (
                <div
                  key={report.id}
                  className={clsx(
                    'card cursor-pointer transition-colors hover:bg-gray-800/50',
                    isGenerating && 'animate-pulse'
                  )}
                  onClick={() => handleViewReport(report)}
                >
                  <div className="card-body">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-4 flex-1 min-w-0">
                        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-800">
                          <FileText className="h-5 w-5 text-gray-400" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="text-sm font-medium text-white truncate">
                              {report.report_name || `Report ${report.id.slice(0, 8)}`}
                            </h3>
                            <Badge
                              variant="neutral"
                              className={clsx('border capitalize text-xs', statusColors[report.status])}
                            >
                              <span className="flex items-center gap-1">
                                <StatusIcon />
                                {report.status.replace('_', ' ')}
                              </span>
                            </Badge>
                            <Badge variant="neutral" className="border text-xs">
                              {report.format.toUpperCase()}
                            </Badge>
                            {report.framework && (
                              <Badge variant="info" className="text-xs">
                                {report.framework.toUpperCase()}
                              </Badge>
                            )}
                          </div>
                          <p className="mt-1 text-xs text-gray-400">
                            {format(new Date(report.config.period_start), 'MMM dd, yyyy')} -{' '}
                            {format(new Date(report.config.period_end), 'MMM dd, yyyy')}
                          </p>
                          <p className="mt-1 text-xs text-gray-500">
                            Created {format(new Date(report.created_at), 'MMM dd, yyyy HH:mm')}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {report.status === 'completed' && (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDownload(report);
                              }}
                              title="Download"
                            >
                              <Download className="h-4 w-4" />
                            </Button>
                          </>
                        )}
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDelete(report);
                          }}
                          className="text-danger-400 hover:text-danger-300 hover:bg-danger-400/10"
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Pagination */}
          {pagination && pagination.total > (filters.limit || 20) && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={handlePrevPage}
                disabled={(filters.offset || 0) === 0}
              >
                Previous
              </Button>
              <span className="text-sm text-gray-400">
                Page {Math.floor((filters.offset || 0) / (filters.limit || 20)) + 1} of{' '}
                {Math.ceil(pagination.total / (filters.limit || 20))}
              </span>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleNextPage}
                disabled={!pagination.has_more}
              >
                Next
              </Button>
            </div>
          )}
        </div>
      )}

    </div>
  );
};

export default ReportsPage;
