/**
 * ScheduledReports Page - Manage scheduled report generation
 * Allows users to create, edit, and manage report schedules
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Calendar,
  Plus,
  Search,
  Edit,
  Trash2,
  Play,
  Pause,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle,
  RefreshCw,
} from 'lucide-react';
import { reportSchedulesApi, reportsApi } from '@/api/reports';
import { Card, CardHeader } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import { EmptyState } from '@/components/common';
import { Modal } from '@/components/common';
import { ReportScheduleDialog } from '@/components/analytics/ReportScheduleDialog';
import { toast } from 'react-hot-toast';
import { format, formatDistanceToNow } from 'date-fns';
import type { ReportSchedule, ReportScheduleFrequency } from '@/types/reports';
import clsx from 'clsx';

const frequencyOptions: Array<{ value: ReportScheduleFrequency; label: string }> = [
  { value: 'once', label: 'One-time' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
  { value: 'quarterly', label: 'Quarterly' },
];

const statusOptions = [
  { value: '', label: 'All Status' },
  { value: 'active', label: 'Active' },
  { value: 'paused', label: 'Paused' },
];

export const ScheduledReportsPage: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<ReportSchedule | undefined>();
  const [viewingHistory, setViewingHistory] = useState<string | undefined>();

  // Query: Schedules
  const { data: schedulesData, isLoading } = useQuery({
    queryKey: ['reportSchedules', statusFilter === 'active' ? true : undefined],
    queryFn: () => reportSchedulesApi.list({
      is_active: statusFilter === 'active' ? true : undefined,
    }),
  });

  // Mutation: Toggle Active/Paused
  const toggleMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      isActive ? reportSchedulesApi.resume(id) : reportSchedulesApi.pause(id),
    onSuccess: () => {
      toast.success('Schedule updated successfully');
      queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to update schedule: ${error.message}`);
    },
  });

  // Mutation: Delete
  const deleteMutation = useMutation({
    mutationFn: (id: string) => reportSchedulesApi.delete(id),
    onSuccess: () => {
      toast.success('Schedule deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete schedule: ${error.message}`);
    },
  });

  // Mutation: Run Now
  const runNowMutation = useMutation({
    mutationFn: (id: string) => reportSchedulesApi.runNow(id),
    onSuccess: () => {
      toast.success('Report generation started');
      queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to run report: ${error.message}`);
    },
  });

  const schedules = schedulesData?.data?.data || [];

  const handleCreate = () => {
    setEditingSchedule(undefined);
    setShowScheduleDialog(true);
  };

  const handleEdit = (schedule: ReportSchedule) => {
    setEditingSchedule(schedule);
    setShowScheduleDialog(true);
  };

  const handleToggle = (id: string, isActive: boolean) => {
    toggleMutation.mutate({ id, isActive: !isActive });
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this schedule?')) {
      deleteMutation.mutate(id);
    }
  };

  const handleRunNow = (id: string) => {
    runNowMutation.mutate(id);
  };

  const handleViewHistory = (id: string) => {
    setViewingHistory(id);
  };

  const filteredSchedules = schedules.filter((schedule) => {
    if (searchQuery && !schedule.name.toLowerCase().includes(searchQuery.toLowerCase())) {
      return false;
    }
    if (statusFilter === 'active' && !schedule.is_active) return false;
    if (statusFilter === 'paused' && schedule.is_active) return false;
    return true;
  });

  const getFrequencyLabel = (frequency: ReportScheduleFrequency) => {
    return frequencyOptions.find((f) => f.value === frequency)?.label || frequency;
  };

  const getNextRunDisplay = (schedule: ReportSchedule) => {
    if (!schedule.is_active) return 'Paused';
    if (schedule.next_run_at) {
      return formatDistanceToNow(new Date(schedule.next_run_at), { addSuffix: true });
    }
    return 'Not scheduled';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Scheduled Reports</h1>
          <p className="mt-1 text-sm text-gray-400">
            Automate report generation on a recurring schedule
          </p>
        </div>
        <Button
          variant="primary"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={handleCreate}
        >
          Schedule Report
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap gap-3">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search schedules..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select
              options={statusOptions}
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="w-40"
            />
          </div>
        </div>
      </Card>

      {/* Schedules List */}
      {isLoading ? (
        <div className="flex min-h-[400px] items-center justify-center">
          <LoadingState message="Loading schedules..." />
        </div>
      ) : filteredSchedules.length === 0 ? (
        <Card>
          <EmptyState
            title="No scheduled reports found"
            description={
              searchQuery || statusFilter
                ? 'Try adjusting your filters'
                : 'Get started by creating your first scheduled report'
            }
            action={
              !searchQuery && !statusFilter ? (
                <Button
                  variant="primary"
                  leftIcon={<Plus className="h-4 w-4" />}
                  onClick={handleCreate}
                >
                  Schedule Report
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <div className="space-y-4">
          {filteredSchedules.map((schedule) => (
            <Card
              key={schedule.id}
              className={clsx(
                'transition-all',
                !schedule.is_active && 'opacity-60'
              )}
            >
              <div className="card-body">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-4">
                    {/* Icon */}
                    <div className={clsx('rounded-lg p-3', {
                      'bg-success-500/20 text-success-400': schedule.is_active,
                      'bg-gray-700 text-gray-500': !schedule.is_active,
                    })}>
                      <Calendar className="h-6 w-6" />
                    </div>

                    {/* Content */}
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="text-lg font-semibold text-white">{schedule.name}</h3>
                        <Badge variant={schedule.is_active ? 'success' : 'neutral'}>
                          {schedule.is_active ? 'Active' : 'Paused'}
                        </Badge>
                      </div>
                      {schedule.description && (
                        <p className="mt-1 text-sm text-gray-400">{schedule.description}</p>
                      )}

                      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-2 text-sm text-gray-400">
                        <span className="flex items-center gap-1">
                          <Clock className="h-4 w-4" />
                          {getFrequencyLabel(schedule.frequency)}
                        </span>
                        <span>•</span>
                        <span>Next: {getNextRunDisplay(schedule)}</span>
                        {schedule.last_run_at && (
                          <>
                            <span>•</span>
                            <span>Last: {format(new Date(schedule.last_run_at), 'MMM d, yyyy')}</span>
                          </>
                        )}
                        <span>•</span>
                        <span>Run count: {schedule.run_count || 0}</span>
                      </div>

                      {/* Recipients */}
                      {schedule.recipients && schedule.recipients.length > 0 && (
                        <div className="mt-2 text-xs text-gray-500">
                          Sends to: {schedule.recipients.join(', ')}
                        </div>
                      )}
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => handleRunNow(schedule.id)}
                      disabled={!schedule.is_active}
                      leftIcon={<Play className="h-4 w-4" />}
                    >
                      Run Now
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleToggle(schedule.id, schedule.is_active)}
                    >
                      {schedule.is_active ? (
                        <Pause className="h-4 w-4" />
                      ) : (
                        <RefreshCw className="h-4 w-4" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEdit(schedule)}
                    >
                      <Edit className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(schedule.id)}
                      className="text-danger-400 hover:text-danger-300"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Schedule Dialog */}
      <ReportScheduleDialog
        isOpen={showScheduleDialog}
        onClose={() => {
          setShowScheduleDialog(false);
          setEditingSchedule(undefined);
        }}
        schedule={editingSchedule}
        onSuccess={() => {
          queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
        }}
      />
    </div>
  );
};

export default ScheduledReportsPage;
