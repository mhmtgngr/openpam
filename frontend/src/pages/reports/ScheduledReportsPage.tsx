/**
 * ScheduledReportsPage Component
 *
 * Manages scheduled report generation and distribution.
 * Users can create, edit, pause, resume, and delete scheduled reports.
 */

import React, { useState, useCallback, useEffect } from 'react';
import {
  Plus,
  Edit,
  Trash2,
  Play,
  Pause,
  Clock,
  Calendar,
  Users,
  Search,
  CheckCircle,
  XCircle,
  AlertCircle,
  MoreVertical,
} from 'lucide-react';
import { reportSchedulesApi } from '@/api/reports';
import type {
  ReportScheduleEntity,
  ReportScheduleFrequency,
  ComplianceFramework,
  ScheduledReportExecution,
} from '@/types/reports';
import { ReportScheduleDialog } from '@/components/analytics/ReportScheduleDialog';

const FREQUENCY_LABELS: Record<ReportScheduleFrequency, string> = {
  once: 'One-time',
  daily: 'Daily',
  weekly: 'Weekly',
  monthly: 'Monthly',
  quarterly: 'Quarterly',
};

const FREQUENCY_ICONS: Record<ReportScheduleFrequency, React.ReactNode> = {
  once: <Clock size={16} />,
  daily: <Calendar size={16} />,
  weekly: <Calendar size={16} />,
  monthly: <Calendar size={16} />,
  quarterly: <Calendar size={16} />,
};

const FRAMEWORK_LABELS: Record<string, string> = {
  soc2: 'SOC 2',
  iso27001: 'ISO 27001',
  pci_dss: 'PCI DSS',
  hipaa: 'HIPAA',
  gdpr: 'GDPR',
  nerc_cip: 'NERC CIP',
  custom: 'Custom',
};

type ViewMode = 'list' | 'create' | 'edit';

interface Filters {
  framework?: ComplianceFramework;
  frequency?: ReportScheduleFrequency;
  isActive?: boolean;
  search: string;
}

export const ScheduledReportsPage: React.FC = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [schedules, setSchedules] = useState<ReportScheduleEntity[]>([]);
  const [selectedSchedule, setSelectedSchedule] = useState<ReportScheduleEntity | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Filters>({ search: '' });
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [expandedSchedule, setExpandedSchedule] = useState<string | null>(null);
  const [executions, setExecutions] = useState<Record<string, ScheduledReportExecution[]>>({});

  // Fetch schedules
  const fetchSchedules = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await reportSchedulesApi.list({
        framework: filters.framework,
        frequency: filters.frequency,
        is_active: filters.isActive,
      });
      setSchedules((data.data || []) as unknown as ReportScheduleEntity[]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load schedules');
    } finally {
      setLoading(false);
    }
  }, [filters.framework, filters.frequency, filters.isActive]);

  // Fetch executions for a schedule
  const fetchExecutions = useCallback(async (scheduleId: string) => {
    try {
      const data = await reportSchedulesApi.getHistory(scheduleId, { limit: 5 });
      setExecutions((prev) => ({ ...prev, [scheduleId]: (data.data || []) as unknown as ScheduledReportExecution[] }));
    } catch (err) {
      console.warn('Failed to fetch executions:', err);
    }
  }, []);

  useEffect(() => {
    fetchSchedules();
  }, [fetchSchedules]);

  // Filter schedules
  const filteredSchedules = schedules.filter((s) => {
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      return (
        s.name.toLowerCase().includes(searchLower) ||
        s.description?.toLowerCase().includes(searchLower)
      );
    }
    return true;
  });

  // Handle create new
  const handleCreate = useCallback(() => {
    setSelectedSchedule(undefined);
    setViewMode('create');
  }, []);

  // Handle edit
  const handleEdit = useCallback((schedule: ReportScheduleEntity) => {
    setSelectedSchedule(schedule);
    setViewMode('edit');
  }, []);

  // Handle pause/resume
  const handleToggleActive = useCallback(async (schedule: ReportScheduleEntity) => {
    try {
      const apiMethod = schedule.is_active ? reportSchedulesApi.pause : reportSchedulesApi.resume;
      await apiMethod(schedule.id);
      setSchedules((prev) =>
        prev.map((s) =>
          s.id === schedule.id ? { ...s, is_active: !s.is_active } : s
        )
      );
      setSuccessMessage(schedule.is_active ? 'Schedule paused' : 'Schedule resumed');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update schedule');
    }
  }, []);

  // Handle run now
  const handleRunNow = useCallback(async (scheduleId: string) => {
    try {
      await reportSchedulesApi.runNow(scheduleId);
      setSuccessMessage('Report generation started');
      setTimeout(() => setSuccessMessage(null), 3000);
      // Refresh executions
      fetchExecutions(scheduleId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run schedule');
    }
  }, [fetchExecutions]);

  // Handle delete
  const handleDelete = useCallback(async (id: string) => {
    try {
      await reportSchedulesApi.delete(id);
      setSchedules((prev) => prev.filter((s) => s.id !== id));
      setDeleteConfirm(null);
      setSuccessMessage('Schedule deleted successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete schedule');
    }
  }, []);

  // Handle save
  const handleSave = useCallback(async (data: any) => {
    try {
      if (selectedSchedule) {
        const updated = await reportSchedulesApi.update(selectedSchedule.id, data);
        setSchedules((prev) => prev.map((s) => (s.id === selectedSchedule.id ? updated as unknown as ReportScheduleEntity : s)));
      } else {
        const created = await reportSchedulesApi.create(data);
        setSchedules((prev) => [...prev, created as unknown as ReportScheduleEntity]);
      }
      setViewMode('list');
      setSuccessMessage(selectedSchedule ? 'Schedule updated successfully' : 'Schedule created successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save schedule');
    }
  }, [selectedSchedule]);

  // Toggle expanded schedule
  const toggleExpanded = useCallback(async (scheduleId: string) => {
    if (expandedSchedule === scheduleId) {
      setExpandedSchedule(null);
    } else {
      setExpandedSchedule(scheduleId);
      if (!executions[scheduleId]) {
        await fetchExecutions(scheduleId);
      }
    }
  }, [expandedSchedule, executions, fetchExecutions]);

  // Render list view
  if (viewMode === 'list') {
    return (
      <div className="scheduled-reports-page">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              Scheduled Reports
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              Manage automated report generation and delivery
            </p>
          </div>
          <button
            onClick={handleCreate}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <Plus size={20} />
            New Schedule
          </button>
        </div>

        {/* Success Message */}
        {successMessage && (
          <div className="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg flex items-center gap-2 text-green-800 dark:text-green-200">
            <CheckCircle size={20} />
            {successMessage}
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-800 dark:text-red-200">
            {error}
            <button
              onClick={() => setError(null)}
              className="ml-4 underline hover:no-underline"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Filters */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Search */}
            <div className="md:col-span-2">
              <div className="relative">
                <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search schedules..."
                  value={filters.search}
                  onChange={(e) => setFilters({ ...filters, search: e.target.value })}
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            {/* Framework Filter */}
            <div>
              <select
                value={filters.framework || ''}
                onChange={(e) => setFilters({ ...filters, framework: e.target.value as ComplianceFramework | undefined })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="">All Frameworks</option>
                {Object.entries(FRAMEWORK_LABELS).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </div>

            {/* Status Filter */}
            <div>
              <select
                value={filters.isActive === undefined ? '' : filters.isActive ? 'active' : 'paused'}
                onChange={(e) => {
                  const val = e.target.value;
                  setFilters({
                    ...filters,
                    isActive: val === '' ? undefined : val === 'active',
                  });
                }}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="">All Status</option>
                <option value="active">Active</option>
                <option value="paused">Paused</option>
              </select>
            </div>
          </div>
        </div>

        {/* Schedules List */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : filteredSchedules.length === 0 ? (
          <div className="text-center py-12">
            <Clock size={48} className="mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              {filters.search || filters.framework || filters.isActive !== undefined
                ? 'No schedules found'
                : 'No scheduled reports yet'}
            </h3>
            <p className="text-gray-500 mb-6">
              {filters.search || filters.framework || filters.isActive !== undefined
                ? 'Try adjusting your filters'
                : 'Create a schedule to automatically generate and deliver reports'}
            </p>
            {!filters.search && !filters.framework && filters.isActive === undefined && (
              <button
                onClick={handleCreate}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 inline-flex items-center gap-2"
              >
                <Plus size={18} />
                Create Schedule
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            {filteredSchedules.map((schedule) => (
              <ScheduleRow
                key={schedule.id}
                schedule={schedule}
                isExpanded={expandedSchedule === schedule.id}
                executions={executions[schedule.id] || []}
                onToggleExpanded={() => toggleExpanded(schedule.id)}
                onEdit={() => handleEdit(schedule)}
                onToggleActive={() => handleToggleActive(schedule)}
                onRunNow={() => handleRunNow(schedule.id)}
                onDelete={() => setDeleteConfirm(schedule.id)}
                isDeleting={deleteConfirm === schedule.id}
                onConfirmDelete={() => handleDelete(schedule.id)}
                onCancelDelete={() => setDeleteConfirm(null)}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  // Render editor view
  return (
    <div>
      <ReportScheduleDialog
        isOpen={true}
        existingSchedule={selectedSchedule ? {
          frequency: selectedSchedule.frequency,
          cron_expression: selectedSchedule.cron_expression,
          day_of_week: undefined,
          day_of_month: undefined,
          time: selectedSchedule.next_run_at ? new Date(selectedSchedule.next_run_at).toTimeString().slice(0, 5) : undefined,
          timezone: selectedSchedule.timezone,
          enabled: selectedSchedule.is_active,
          recipients: selectedSchedule.recipients,
          next_run_at: selectedSchedule.next_run_at,
        } : undefined}
        onSave={handleSave}
        onClose={() => setViewMode('list')}
      />
    </div>
  );
};

interface ScheduleRowProps {
  schedule: ReportScheduleEntity;
  isExpanded: boolean;
  executions: ScheduledReportExecution[];
  onToggleExpanded: () => void;
  onEdit: () => void;
  onToggleActive: () => void;
  onRunNow: () => void;
  onDelete: () => void;
  isDeleting: boolean;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;
}

const ScheduleRow: React.FC<ScheduleRowProps> = ({
  schedule,
  isExpanded,
  executions,
  onToggleExpanded,
  onEdit,
  onToggleActive,
  onRunNow,
  onDelete,
  isDeleting,
  onConfirmDelete,
  onCancelDelete,
}) => {
  const statusIcon = schedule.is_active ? (
    <CheckCircle size={16} className="text-green-500" />
  ) : (
    <Pause size={16} className="text-yellow-500" />
  );

  const nextRunDate = schedule.next_run_at
    ? new Date(schedule.next_run_at)
    : null;

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
      {/* Main Row */}
      <div className="p-4 flex items-center gap-4">
        {/* Drag Handle / Icon */}
        <div className="flex-shrink-0">
          {FREQUENCY_ICONS[schedule.frequency] || <Clock size={20} />}
        </div>

        {/* Status */}
        <div className="flex-shrink-0">{statusIcon}</div>

        {/* Schedule Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-gray-900 dark:text-white truncate">
              {schedule.name}
            </h3>
            <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
              {FREQUENCY_LABELS[schedule.frequency]}
            </span>
            <span className="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded">
              {FRAMEWORK_LABELS[schedule.framework]}
            </span>
          </div>
          {schedule.description && (
            <p className="text-sm text-gray-500 truncate">{schedule.description}</p>
          )}
          <div className="flex items-center gap-4 mt-2 text-sm text-gray-500">
            <span className="flex items-center gap-1">
              <Calendar size={14} />
              Next run: {nextRunDate ? nextRunDate.toLocaleString() : 'Not scheduled'}
            </span>
            <span className="flex items-center gap-1">
              <Users size={14} />
              {schedule.recipients.length} recipients
            </span>
            <span>Run count: {schedule.run_count}</span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {schedule.is_active && (
            <button
              onClick={onRunNow}
              className="p-2 text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/20 rounded"
              title="Run now"
            >
              <Play size={18} />
            </button>
          )}
          <button
            onClick={onToggleActive}
            className={`p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded ${
              schedule.is_active ? 'text-yellow-600 dark:text-yellow-400' : 'text-green-600 dark:text-green-400'
            }`}
            title={schedule.is_active ? 'Pause schedule' : 'Resume schedule'}
          >
            {schedule.is_active ? <Pause size={18} /> : <Play size={18} />}
          </button>
          <button
            onClick={onEdit}
            className="p-2 text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 rounded"
            title="Edit"
          >
            <Edit size={18} />
          </button>
          {isDeleting ? (
            <>
              <button
                onClick={onConfirmDelete}
                className="px-3 py-1 bg-red-600 text-white text-sm rounded hover:bg-red-700"
              >
                Confirm
              </button>
              <button
                onClick={onCancelDelete}
                className="px-3 py-1 bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 text-sm rounded hover:bg-gray-300 dark:hover:bg-gray-500"
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              <button
                onClick={onToggleExpanded}
                className="p-2 text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 rounded"
                title={isExpanded ? 'Hide history' : 'Show history'}
              >
                <MoreVertical size={18} />
              </button>
              <button
                onClick={onDelete}
                className="p-2 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 rounded"
                title="Delete"
              >
                <Trash2 size={18} />
              </button>
            </>
          )}
        </div>
      </div>

      {/* Expanded History */}
      {isExpanded && (
        <div className="border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 p-4">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            Recent Runs
          </h4>
          {executions.length === 0 ? (
            <p className="text-sm text-gray-500">No executions yet</p>
          ) : (
            <div className="space-y-2">
              {executions.map((execution) => (
                <ExecutionRow key={execution.id} execution={execution} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

interface ExecutionRowProps {
  execution: ScheduledReportExecution;
}

const ExecutionRow: React.FC<ExecutionRowProps> = ({ execution }) => {
  const statusConfig = {
    completed: { icon: CheckCircle, color: 'text-green-500', bg: 'bg-green-50 dark:bg-green-900/20' },
    failed: { icon: XCircle, color: 'text-red-500', bg: 'bg-red-50 dark:bg-red-900/20' },
    running: { icon: Clock, color: 'text-blue-500', bg: 'bg-blue-50 dark:bg-blue-900/20' },
    pending: { icon: Clock, color: 'text-gray-500', bg: 'bg-gray-50 dark:bg-gray-900/20' },
    cancelled: { icon: XCircle, color: 'text-gray-500', bg: 'bg-gray-50 dark:bg-gray-900/20' },
  };

  const config = statusConfig[execution.status as keyof typeof statusConfig] || statusConfig.pending;
  const StatusIcon = config.icon;

  return (
    <div className={`flex items-center gap-3 p-3 rounded-lg ${config.bg}`}>
      <StatusIcon size={16} className={config.color} />
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium text-gray-900 dark:text-white">
            {execution.report_name}
          </span>
          <span className={`text-xs capitalize ${config.color}`}>
            {execution.status}
          </span>
        </div>
        <div className="text-sm text-gray-500">
          Scheduled: {new Date(execution.scheduled_for).toLocaleString()}
          {execution.executed_at && (
            <span> • Executed: {new Date(execution.executed_at).toLocaleString()}</span>
          )}
        </div>
        {execution.error_message && (
          <div className="text-sm text-red-600 dark:text-red-400 mt-1">
            {execution.error_message}
          </div>
        )}
      </div>
      {execution.snapshot_id && (
        <a
          href={`/reports/snapshots/${execution.snapshot_id}`}
          className="text-blue-600 hover:text-blue-700 text-sm"
        >
          View
        </a>
      )}
    </div>
  );
};

export default ScheduledReportsPage;
