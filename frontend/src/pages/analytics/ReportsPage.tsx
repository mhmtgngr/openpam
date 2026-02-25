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
import { Card, CardHeader } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import { EmptyState } from '@/components/common';
import { ReportCard } from '@/components/analytics/ReportCard';
import { ReportViewer } from '@/components/common/ReportViewer';
import { useReports } from '@/contexts/ReportContext';
import type { ReportSnapshot, ReportType, ReportStatus, ReportFormat } from '@/types/reports';
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

export const ReportsPage: React.FC = () => {
  const navigate = useNavigate();
  const {
    snapshots,
    selectedSnapshot,
    isLoading,
    filters,
    setFilters,
    totalSnapshots,
    refreshSnapshots,
    deleteSnapshot,
    downloadSnapshot,
  } = useReports();

  const [searchQuery, setSearchQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Query for detailed snapshot when viewing
  const { data: detailedSnapshot, isLoading: isLoadingDetail } = useQuery({
    queryKey: ['reportSnapshot', selectedSnapshot?.id],
    queryFn: () => reportsApi.getSnapshot(selectedSnapshot!.id),
    enabled: !!selectedSnapshot?.id,
    staleTime: 30 * 1000,
  });

  const currentSnapshot = detailedSnapshot || selectedSnapshot;

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
    setSelectedIds(new Set());
  };

  const handleSelectReport = (report: ReportSnapshot) => {
    setSelectedIds((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(report.id)) {
        newSet.delete(report.id);
      } else {
        newSet.add(report.id);
      }
      return newSet;
    });
  };

  const handleSelectAll = () => {
    if (selectedIds.size === snapshots.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(snapshots.map((r) => r.id)));
    }
  };

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return;

    if (!confirm(`Are you sure you want to delete ${selectedIds.size} report(s)?`)) {
      return;
    }

    for (const id of selectedIds) {
      await deleteSnapshot(id);
    }

    setSelectedIds(new Set());
  };

  const handleViewReport = (report: ReportSnapshot) => {
    navigate(`/analytics/reports/${report.id}`);
  };

  const statusCounts = snapshots.reduce(
    (acc, report) => {
      acc[report.status] = (acc[report.status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const activeFilterCount =
    (filters.type ? 1 : 0) +
    (filters.status ? 1 : 0) +
    (filters.format ? 1 : 0) +
    (filters.search ? 1 : 0);

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
            leftIcon={<RefreshCw className="h-4 w-4" />}
            onClick={refreshSnapshots}
          >
            Refresh
          </Button>
          <Button
            variant="primary"
            leftIcon={<Plus className="h-4 w-4" />}
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
            <p className="text-2xl font-bold text-white">{totalSnapshots}</p>
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
                leftIcon={<Filter className="h-4 w-4" />}
              >
                Filters
                {activeFilterCount > 0 && (
                  <Badge variant="primary" className="ml-2">
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
                  <Select options={reportTypeOptions} value={filters.type || ''} onChange={handleTypeChange} />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Status</label>
                  <Select options={statusOptions} value={filters.status || ''} onChange={handleStatusChange} />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Format</label>
                  <Select options={formatOptions} value={filters.format || ''} onChange={handleFormatChange} />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Sort By</label>
                  <Select options={sortOptions} value={`${filters.sort_by}:${filters.sort_order}`} onChange={handleSortChange} />
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

      {/* Bulk Actions */}
      {selectedIds.size > 0 && (
        <Card className="border-l-4 border-l-primary-500">
          <div className="card-body flex items-center justify-between py-3">
            <p className="text-sm text-white">
              {selectedIds.size} report(s) selected
            </p>
            <div className="flex items-center gap-2">
              <Button
                variant="danger"
                size="sm"
                leftIcon={<Trash2 className="h-4 w-4" />}
                onClick={handleBulkDelete}
              >
                Delete Selected
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelectedIds(new Set())}
              >
                Cancel
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Reports List */}
      {isLoading ? (
        <div className="flex min-h-[400px] items-center justify-center">
          <LoadingState message="Loading reports..." />
        </div>
      ) : snapshots.length === 0 ? (
        <Card>
          <EmptyState
            title="No reports found"
            description={
              activeFilterCount > 0
                ? 'Try adjusting your filters or search query'
                : 'Get started by generating your first report'
            }
            action={
              activeFilterCount === 0 ? (
                <Button
                  variant="primary"
                  leftIcon={<Plus className="h-4 w-4" />}
                  onClick={() => navigate('/analytics/reports/generate')}
                >
                  Generate Report
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {selectedIds.size > 0 && selectedIds.size === snapshots.length ? (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedIds(new Set())}
                >
                  Deselect All
                </Button>
              ) : (
                <Button variant="ghost" size="sm" onClick={handleSelectAll}>
                  Select All
                </Button>
              )}
              <p className="text-sm text-gray-400">
                Showing {snapshots.length} of {totalSnapshots} reports
              </p>
            </div>
          </div>

          <div className="space-y-3">
            {snapshots.map((report) => (
              <div key={report.id} className="relative">
                <input
                  type="checkbox"
                  checked={selectedIds.has(report.id)}
                  onChange={() => handleSelectReport(report)}
                  className="absolute left-4 top-4 z-10 h-4 w-4 rounded border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                  onClick={(e) => e.stopPropagation()}
                />
                <ReportCard
                  report={report}
                  onView={handleViewReport}
                  onDownload={(r) => downloadSnapshot(r.id)}
                  onDelete={(r) => deleteSnapshot(r.id)}
                  showActions
                />
              </div>
            ))}
          </div>

          {/* Pagination */}
          {totalSnapshots > (filters.limit || 20) && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={!filters.offset}
                onClick={() => setFilters({ ...filters, offset: Math.max(0, (filters.offset || 0) - (filters.limit || 20)) })}
              >
                Previous
              </Button>
              <span className="text-sm text-gray-400">
                Page {Math.floor((filters.offset || 0) / (filters.limit || 20)) + 1} of{' '}
                {Math.ceil(totalSnapshots / (filters.limit || 20))}
              </span>
              <Button
                variant="secondary"
                size="sm"
                disabled={(filters.offset || 0) + (filters.limit || 20) >= totalSnapshots}
                onClick={() => setFilters({ ...filters, offset: (filters.offset || 0) + (filters.limit || 20) })}
              >
                Next
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Report Viewer Modal */}
      {selectedSnapshot && (
        <ReportViewer
          report={currentSnapshot}
          isLoading={isLoadingDetail}
          onClose={() => setSelectedIds(new Set())}
          onDownload={(r) => downloadSnapshot(r.id)}
        />
      )}
    </div>
  );
};

export default ReportsPage;
