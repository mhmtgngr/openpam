import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  FileText,
  Plus,
  Calendar,
  Filter,
  Download,
  Share,
  Trash2,
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  Search,
  FileSpreadsheet,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { ReportScheduleDialog } from '@/components/analytics/ReportScheduleDialog';
import { ReportDistributionDialog } from '@/components/analytics/ReportDistributionDialog';
import { Card } from '@/components/common';
import { Button, Input, Select, Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import type { Report, ReportListParams, ComplianceFramework, ReportStatus } from '@/types/reports';
import { format } from 'date-fns';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const frameworks: { value: ComplianceFramework; label: string }[] = [
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'pci_dss', label: 'PCI DSS' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'gdpr', label: 'GDPR' },
  { value: 'nerc_cip', label: 'NERC CIP' },
  { value: 'custom', label: 'Custom' },
];

const statusOptions: { value: ReportStatus; label: string }[] = [
  { value: 'completed', label: 'Completed' },
  { value: 'generating', label: 'Generating' },
  { value: 'failed', label: 'Failed' },
  { value: 'scheduled', label: 'Scheduled' },
  { value: 'pending', label: 'Pending' },
];

const statusIcons: Record<ReportStatus, React.ElementType> = {
  completed: CheckCircle,
  generating: RefreshCw,
  failed: XCircle,
  scheduled: Clock,
  pending: AlertCircle,
  cancelled: XCircle,
};

const statusColors: Record<ReportStatus, string> = {
  completed: 'text-success-400 bg-success-400/10',
  generating: 'text-primary-400 bg-primary-400/10',
  failed: 'text-danger-400 bg-danger-400/10',
  scheduled: 'text-warning-400 bg-warning-400/10',
  pending: 'text-gray-400 bg-gray-400/10',
  cancelled: 'text-gray-400 bg-gray-400/10',
};

const frameworkColors: Record<string, string> = {
  soc2: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  iso27001: 'bg-green-500/20 text-green-400 border-green-500/30',
  pci_dss: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  hipaa: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
  gdpr: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
  nerc_cip: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
  custom: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
};

export const ReportsPage: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<ReportListParams>({
    limit: 20,
    offset: 0,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedReport, setSelectedReport] = useState<Report | null>(null);
  const [scheduleDialogOpen, setScheduleDialogOpen] = useState(false);
  const [distributionDialogOpen, setDistributionDialogOpen] = useState(false);

  const { data: reportsData, isLoading } = useQuery({
    queryKey: ['reports', filters],
    queryFn: () => reportsApi.list(filters),
  });

  const { data: dashboardData } = useQuery({
    queryKey: ['reportDashboard'],
    queryFn: () => reportsApi.getDashboard(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => reportsApi.delete(id),
    onSuccess: () => {
      toast.success('Report deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      queryClient.invalidateQueries({ queryKey: ['reportDashboard'] });
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to delete report');
    },
  });

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setFilters((prev) => ({ ...prev, search: value || undefined, offset: 0 }));
  };

  const handleFilterChange = (key: keyof ReportListParams, value: string | undefined) => {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, offset: 0 }));
  };

  const handleSortChange = (sortBy: string) => {
    setFilters((prev) => ({
      ...prev,
      sort_by: sortBy as any,
      sort_order: prev.sort_by === sortBy && prev.sort_order === 'desc' ? 'asc' : 'desc',
    }));
  };

  const handleDelete = (id: string, name: string) => {
    if (confirm(`Are you sure you want to delete report "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  const handleRowClick = (report: Report) => {
    navigate(`/reports/${report.id}`);
  };

  const reports = reportsData?.data || [];
  const pagination = reportsData?.pagination;

  const handleNextPage = () => {
    if (pagination?.has_more) {
      setFilters((prev) => ({ ...prev, offset: (prev.offset || 0) + (prev.limit || 20) }));
    }
  };

  const handlePrevPage = () => {
    if ((filters.offset || 0) > 0) {
      setFilters((prev) => ({ ...prev, offset: Math.max(0, (prev.offset || 0) - (prev.limit || 20)) }));
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Compliance Reports</h1>
          <p className="mt-1 text-sm text-gray-400">
            Generate, manage, and distribute compliance reports
          </p>
        </div>
        <div className="flex gap-3">
          <Button
            variant="secondary"
            onClick={() => setScheduleDialogOpen(true)}
            className="flex items-center gap-2"
          >
            <Clock className="h-4 w-4" />
            Schedule Report
          </Button>
          <Button
            variant="primary"
            onClick={() => navigate('/reports/generate')}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Generate Report
          </Button>
        </div>
      </div>

      {/* Summary Stats */}
      {dashboardData && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            title="Total Reports"
            value={dashboardData.total_reports}
            icon={FileText}
            color="blue"
          />
          <StatCard
            title="Active Schedules"
            value={dashboardData.scheduled_reports}
            icon={Clock}
            color="purple"
          />
          <StatCard
            title="Pending Exceptions"
            value={dashboardData.pending_exceptions}
            icon={AlertCircle}
            color="orange"
          />
          <StatCard
            title="In Queue"
            value={dashboardData.generation_queue.length}
            icon={RefreshCw}
            color="green"
          />
        </div>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap items-center gap-4">
            {/* Search */}
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search reports..."
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>

            {/* Framework Filter */}
            <Select
              value={filters.framework || ''}
              onChange={(e) => handleFilterChange('framework', e.target.value || undefined)}
              className="w-40"
            >
              <option value="">All Frameworks</option>
              {frameworks.map((fw) => (
                <option key={fw.value} value={fw.value}>
                  {fw.label}
                </option>
              ))}
            </Select>

            {/* Status Filter */}
            <Select
              value={filters.status || ''}
              onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
              className="w-40"
            >
              <option value="">All Status</option>
              {statusOptions.map((st) => (
                <option key={st.value} value={st.value}>
                  {st.label}
                </option>
              ))}
            </Select>

            {/* Sort */}
            <Select
              value={`${filters.sort_by}-${filters.sort_order}`}
              onChange={(e) => {
                const [sort, order] = e.target.value.split('-');
                handleSortChange(sort);
              }}
              className="w-48"
            >
              <option value="created_at-desc">Newest First</option>
              <option value="created_at-asc">Oldest First</option>
              <option value="name-asc">Name (A-Z)</option>
              <option value="name-desc">Name (Z-A)</option>
              <option value="generated_at-desc">Recently Generated</option>
            </Select>

            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setFilters({ limit: 20, offset: 0, sort_by: 'created_at', sort_order: 'desc' });
                setSearchQuery('');
              }}
            >
              Clear Filters
            </Button>
          </div>
        </div>
      </Card>

      {/* Reports List */}
      <Card>
        <div className="card-body p-0">
          {isLoading ? (
            <div className="py-12">
              <LoadingState message="Loading reports..." />
            </div>
          ) : reports.length === 0 ? (
            <div className="py-12 text-center">
              <FileSpreadsheet className="mx-auto h-12 w-12 text-gray-600" />
              <h3 className="mt-4 text-sm font-medium text-white">No reports found</h3>
              <p className="mt-2 text-sm text-gray-400">
                {searchQuery || filters.framework || filters.status
                  ? 'Try adjusting your filters'
                  : 'Get started by generating your first compliance report'}
              </p>
              {!searchQuery && !filters.framework && !filters.status && (
                <div className="mt-6">
                  <Button variant="primary" onClick={() => navigate('/reports/generate')}>
                    <Plus className="mr-2 h-4 w-4" />
                    Generate Report
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-700">
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-400">
                      Report Name
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-400">
                      Framework
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-400">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-400">
                      Period
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-400">
                      Created
                    </th>
                    <th className="px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-400">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-700">
                  {reports.map((report) => {
                    const StatusIcon = statusIcons[report.status];
                    const isGenerating = report.status === 'generating';

                    return (
                      <tr
                        key={report.id}
                        className={clsx(
                          'cursor-pointer transition-colors hover:bg-gray-800/50',
                          isGenerating && 'animate-pulse'
                        )}
                        onClick={() => handleRowClick(report)}
                      >
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-800">
                              <FileText className="h-5 w-5 text-gray-400" />
                            </div>
                            <div>
                              <p className="text-sm font-medium text-white">{report.name}</p>
                              {report.description && (
                                <p className="text-xs text-gray-400 truncate max-w-xs">
                                  {report.description}
                                </p>
                              )}
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <Badge
                            variant="outline"
                            className={clsx('border', frameworkColors[report.framework] || frameworkColors.custom)}
                          >
                            {report.framework.toUpperCase()}
                          </Badge>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex items-center gap-2">
                            <StatusIcon className={clsx('h-4 w-4', statusColors[report.status]?.split(' ')[0])} />
                            <span className={clsx('text-sm capitalize', statusColors[report.status])}>
                              {report.status.replace('_', ' ')}
                            </span>
                          </div>
                          {isGenerating && (
                            <div className="mt-1 h-1 w-24 overflow-hidden rounded-full bg-gray-700">
                              <div className="h-full animate-progress w-full bg-primary-500" />
                            </div>
                          )}
                        </td>
                        <td className="px-6 py-4">
                          <p className="text-sm text-gray-300">
                            {format(new Date(report.period_start), 'MMM dd, yyyy')} -{' '}
                            {format(new Date(report.period_end), 'MMM dd, yyyy')}
                          </p>
                        </td>
                        <td className="px-6 py-4">
                          <p className="text-sm text-gray-300">
                            {format(new Date(report.created_at), 'MMM dd, yyyy')}
                          </p>
                        </td>
                        <td
                          className="px-6 py-4 text-right"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <div className="flex items-center justify-end gap-2">
                            {report.status === 'completed' && (
                              <>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => navigate(`/reports/${report.id}`)}
                                  title="View Details"
                                >
                                  <FileText className="h-4 w-4" />
                                </Button>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => {
                                    setSelectedReport(report);
                                    setDistributionDialogOpen(true);
                                  }}
                                  title="Distribute"
                                >
                                  <Share className="h-4 w-4" />
                                </Button>
                              </>
                            )}
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDelete(report.id, report.name)}
                              className="text-danger-400 hover:text-danger-300 hover:bg-danger-400/10"
                              title="Delete"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Pagination */}
        {pagination && pagination.total > (filters.limit || 20) && (
          <div className="card-body border-t border-gray-700">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-400">
                Showing {Math.max(0, filters.offset || 0) + 1} to{' '}
                {Math.min(pagination.total, (filters.offset || 0) + (filters.limit || 20))} of{' '}
                {pagination.total} reports
              </p>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={(filters.offset || 0) === 0}
                >
                  Previous
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!pagination.has_more}
                >
                  Next
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Dialogs */}
      <ReportScheduleDialog
        isOpen={scheduleDialogOpen}
        onClose={() => setScheduleDialogOpen(false)}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['reportDashboard'] })}
      />

      <ReportDistributionDialog
        isOpen={distributionDialogOpen}
        onClose={() => {
          setDistributionDialogOpen(false);
          setSelectedReport(null);
        }}
        report={selectedReport}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['reports'] })}
      />
    </div>
  );
};

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ElementType;
  color: 'blue' | 'green' | 'purple' | 'orange';
}

const StatCard: React.FC<StatCardProps> = ({ title, value, icon: Icon, color }) => {
  const colorClasses = {
    blue: 'text-blue-400 bg-blue-400/10',
    green: 'text-success-400 bg-success-400/10',
    purple: 'text-purple-400 bg-purple-400/10',
    orange: 'text-warning-400 bg-warning-400/10',
  };

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-gray-400">{title}</p>
            <p className="mt-2 text-2xl font-bold text-white">{value}</p>
          </div>
          <div className={clsx('rounded-lg p-3', colorClasses[color])}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};
