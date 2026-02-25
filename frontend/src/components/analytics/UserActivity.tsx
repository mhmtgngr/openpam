import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format, subDays, startOfDay } from 'date-fns';
import {
  User,
  Search,
  Filter,
  Calendar,
  Activity,
  Clock,
  MapPin,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { analyticsApi, type UserActivityParams } from '@/api/analytics';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge, StatusBadge } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import { Pagination } from '@/components/common';
import type { UserActivity as UserActivityType } from '@/types';
import clsx from 'clsx';

const periodOptions = [
  { value: '7', label: 'Last 7 days' },
  { value: '14', label: 'Last 14 days' },
  { value: '30', label: 'Last 30 days' },
  { value: '90', label: 'Last 90 days' },
];

interface HeatmapProps {
  data: Array<{ date: string; hour: number; session_count: number }>;
  startDate: Date;
  endDate: Date;
}

const ActivityHeatmap: React.FC<HeatmapProps> = ({ data, startDate, endDate }) => {
  const hours = Array.from({ length: 24 }, (_, i) => i);
  const days = [];
  const currentDate = new Date(startDate);

  while (currentDate <= endDate) {
    days.push(new Date(currentDate));
    currentDate.setDate(currentDate.getDate() + 1);
  }

  // Limit to last 14 days for display
  const displayDays = days.slice(-14);

  const getActivityCount = (date: Date, hour: number): number => {
    const dateStr = format(date, 'yyyy-MM-dd');
    const entry = data.find((d) => d.date === dateStr && d.hour === hour);
    return entry?.session_count || 0;
  };

  const getIntensityColor = (count: number): string => {
    if (count === 0) return 'bg-gray-800';
    if (count < 3) return 'bg-primary-900';
    if (count < 6) return 'bg-primary-700';
    if (count < 10) return 'bg-primary-500';
    return 'bg-primary-400';
  };

  return (
    <div className="overflow-x-auto">
      <div className="min-w-[600px]">
        {/* Time labels */}
        <div className="flex border-l border-gray-700">
          <div className="w-12 shrink-0" />
          {displayDays.map((day) => (
            <div
              key={day.toISOString()}
              className="flex-1 text-center text-xs text-gray-500"
            >
              {format(day, 'EEE')}
            </div>
          ))}
        </div>

        {/* Heatmap grid */}
        <div className="space-y-1">
          {hours.map((hour) => (
            <div key={hour} className="flex items-center">
              <div className="w-12 shrink-0 text-xs text-gray-500">
                {String(hour).padStart(2, '0')}:00
              </div>
              <div className="flex flex-1 gap-0.5">
                {displayDays.map((day) => {
                  const count = getActivityCount(day, hour);
                  return (
                    <div
                      key={day.toISOString()}
                      className={clsx(
                        'flex-1 h-6 rounded-sm',
                        getIntensityColor(count)
                      )}
                      title={`${format(day, 'MMM d')} ${String(hour).padStart(2, '0')}:00 - ${count} sessions`}
                    />
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        {/* Legend */}
        <div className="mt-3 flex items-center justify-end gap-2 text-xs text-gray-500">
          <span>Less</span>
          <div className="flex gap-0.5">
            <div className="h-3 w-3 rounded-sm bg-gray-800" />
            <div className="h-3 w-3 rounded-sm bg-primary-900" />
            <div className="h-3 w-3 rounded-sm bg-primary-700" />
            <div className="h-3 w-3 rounded-sm bg-primary-500" />
            <div className="h-3 w-3 rounded-sm bg-primary-400" />
          </div>
          <span>More</span>
        </div>
      </div>
    </div>
  );
};

interface ActivityRowProps {
  activity: UserActivityType;
  expanded: boolean;
  onToggle: () => void;
}

const ActivityRow: React.FC<ActivityRowProps> = ({ activity, expanded, onToggle }) => {
  const formatDuration = (seconds: number): string => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  };

  return (
    <>
      <tr className="border-b border-gray-800 hover:bg-gray-800/50">
        <td className="py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-white">
              {activity.user_name.charAt(0).toUpperCase()}
            </div>
            <div>
              <p className="text-sm font-medium text-white">{activity.user_name}</p>
              <p className="text-xs text-gray-500">{activity.user_email}</p>
            </div>
          </div>
        </td>
        <td className="py-4 text-center">
          <span className="text-sm text-white">{activity.total_sessions}</span>
        </td>
        <td className="py-4 text-center">
          <span className="text-sm text-white">{formatDuration(activity.total_duration_seconds)}</span>
        </td>
        <td className="py-4 text-center">
          <span className="text-sm text-white">{formatDuration(activity.avg_session_duration_seconds)}</span>
        </td>
        <td className="py-4">
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <Clock className="h-4 w-4" />
            {activity.last_activity_at
              ? format(new Date(activity.last_activity_at), 'MMM d, HH:mm')
              : 'Never'}
          </div>
        </td>
        <td className="py-4 text-right">
          <button
            onClick={onToggle}
            className="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white"
          >
            {expanded ? (
              <ChevronUp className="h-5 w-5" />
            ) : (
              <ChevronDown className="h-5 w-5" />
            )}
          </button>
        </td>
      </tr>

      {expanded && (
        <tr className="border-b border-gray-800 bg-gray-800/30">
          <td colSpan={6} className="py-4 px-4">
            <div className="space-y-4">
              <div>
                <h4 className="mb-2 text-sm font-medium text-gray-300">Most Used Targets</h4>
                <div className="flex flex-wrap gap-2">
                  {activity.most_used_targets.length > 0 ? (
                    activity.most_used_targets.map((target, idx) => (
                      <Badge key={idx} variant="neutral">
                        {target}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-sm text-gray-500">No targets accessed</span>
                  )}
                </div>
              </div>

              {activity.activity_heatmap && activity.activity_heatmap.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium text-gray-300">Activity Heatmap (Last 14 Days)</h4>
                  <ActivityHeatmap
                    data={activity.activity_heatmap.map((h) => ({
                      date: h.day,
                      hour: h.hour,
                      session_count: h.count,
                    }))}
                    startDate={subDays(new Date(), 14)}
                    endDate={new Date()}
                  />
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
};

export const UserActivity: React.FC = () => {
  const [days, setDays] = useState(30);
  const [search, setSearch] = useState('');
  const [offset, setOffset] = useState(0);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const limit = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['userActivity', days, search, offset, limit],
    queryFn: () =>
      analyticsApi.getUserActivity({
        start_date: subDays(startOfDay(new Date()), days).toISOString(),
        end_date: new Date().toISOString(),
        search,
        limit,
        offset,
        include_heatmap: true,
      }),
  });

  const totalUsers = data?.pagination?.total || 0;

  return (
    <div className="space-y-6">
      <CardHeader
        title="User Activity"
        subtitle="Track user access patterns and behavior"
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
            <div className="flex items-center gap-2 rounded-lg bg-gray-800 px-3 py-2">
              <Calendar className="h-4 w-4 text-gray-400" />
              <span className="text-sm text-gray-300">Last {days} days</span>
            </div>
          </div>
        }
      />

      <Card>
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search users..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
          </div>
        </div>
      </Card>

      <Card>
        {isLoading ? (
          <div className="card-body">
            <LoadingState message="Loading user activity..." />
          </div>
        ) : !data || data.data.length === 0 ? (
          <div className="card-body">
            <EmptyState title="No user activity found" />
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-700">
                    <th className="pb-3 text-left text-sm font-medium text-gray-400">
                      <div className="flex items-center gap-2">
                        <User className="h-4 w-4" />
                        User
                      </div>
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Sessions
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Total Time
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Avg Duration
                    </th>
                    <th className="pb-3 text-left text-sm font-medium text-gray-400">
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4" />
                        Last Activity
                      </div>
                    </th>
                    <th className="pb-3 text-right" />
                  </tr>
                </thead>
                <tbody>
                  {data.data.map((activity) => (
                    <ActivityRow
                      key={activity.user_id}
                      activity={activity}
                      expanded={expandedRow === activity.user_id}
                      onToggle={() =>
                        setExpandedRow(expandedRow === activity.user_id ? null : activity.user_id)
                      }
                    />
                  ))}
                </tbody>
              </table>
            </div>

            {totalUsers > limit && (
              <div className="card-footer">
                <Pagination
                  total={totalUsers}
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
