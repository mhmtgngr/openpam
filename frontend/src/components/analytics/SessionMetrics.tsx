import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  Activity,
  Clock,
  TrendingUp,
  Monitor,
  Calendar,
  BarChart3,
  RefreshCw,
} from 'lucide-react';
import { analyticsApi, type SessionMetricsParams } from '@/api/analytics';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import { Select } from '@/components/common';

const granularityOptions = [
  { value: 'hour', label: 'Hourly' },
  { value: 'day', label: 'Daily' },
  { value: 'week', label: 'Weekly' },
  { value: 'month', label: 'Monthly' },
];

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: React.ElementType;
  trend?: number;
  variant?: 'default' | 'success' | 'warning' | 'danger';
}

const StatCard: React.FC<StatCardProps> = ({
  title,
  value,
  subtitle,
  icon: Icon,
  trend,
  variant = 'default',
}) => {
  const variantClasses: Record<typeof variant, string> = {
    default: 'text-primary-400',
    success: 'text-success-400',
    warning: 'text-warning-400',
    danger: 'text-danger-400',
  };

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-400">{title}</p>
            <p className="mt-2 text-3xl font-bold text-white">{value}</p>
            {subtitle && (
              <p className="mt-1 text-sm text-gray-500">{subtitle}</p>
            )}
            {trend !== undefined && (
              <div className="mt-2 flex items-center gap-1">
                <TrendingUp
                  className={`h-4 w-4 ${trend >= 0 ? 'text-success-400' : 'text-danger-400'}`}
                />
                <span className={`text-sm ${trend >= 0 ? 'text-success-400' : 'text-danger-400'}`}>
                  {trend >= 0 ? '+' : ''}{trend}%
                </span>
                <span className="text-xs text-gray-500">vs last period</span>
              </div>
            )}
          </div>
          <div className={`rounded-lg bg-gray-800 p-3 ${variantClasses[variant]}`}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};

interface MiniChartProps {
  data: Array<{ timestamp: string; value: number }>;
  color?: string;
}

const MiniChart: React.FC<MiniChartProps> = ({ data, color = '#6366f1' }) => {
  if (!data || data.length === 0) return null;

  const max = Math.max(...data.map((d) => d.value), 1);
  const min = Math.min(...data.map((d) => d.value), 0);
  const range = max - min || 1;

  const points = data
    .map((d, i) => {
      const x = (i / (data.length - 1)) * 100;
      const y = 100 - ((d.value - min) / range) * 80 - 10;
      return `${x},${y}`;
    })
    .join(' ');

  return (
    <svg
      viewBox="0 0 100 100"
      className="h-16 w-full"
      preserveAspectRatio="none"
    >
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth="2"
        vectorEffect="non-scaling-stroke"
      />
      <polyline
        points={`${points} 100,100 0,100`}
        fill={`${color}20`}
        stroke="none"
      />
    </svg>
  );
};

export const SessionMetrics: React.FC = () => {
  const [granularity, setGranularity] = useState<SessionMetricsParams['granularity']>('day');
  const [endDate] = useState(new Date());
  const [startDate] = useState(
    new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)
  );

  const { data, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ['sessionMetrics', granularity, startDate.toISOString(), endDate.toISOString()],
    queryFn: () =>
      analyticsApi.getSessionMetrics({
        start_date: startDate.toISOString(),
        end_date: endDate.toISOString(),
        granularity,
      }),
    refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
  });

  const formatDuration = (seconds: number): string => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  };

  if (isLoading) {
    return (
      <div className="card">
        <div className="card-body">
          <LoadingState message="Loading session metrics..." />
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="card">
        <div className="card-body">
          <EmptyState title="No session metrics available" />
        </div>
      </div>
    );
  }

  const timeSeriesData = data.sessions_over_time.map((d: { timestamp: string; count: number }) => ({
    timestamp: d.timestamp,
    value: d.count,
  }));

  return (
    <div className="space-y-6">
      <CardHeader
        title="Session Metrics"
        subtitle="Track session activity patterns and trends"
        action={
          <div className="flex items-center gap-3">
            <Select
              options={granularityOptions}
              value={granularity || 'day'}
              onChange={(e) => setGranularity(e.target.value as SessionMetricsParams['granularity'])}
              className="w-32"
            />
            <button
              onClick={() => refetch()}
              disabled={isRefetching}
              className="rounded-lg bg-gray-800 p-2 text-gray-400 hover:bg-gray-700 hover:text-white disabled:opacity-50"
            >
              <RefreshCw className={`h-4 w-4 ${isRefetching ? 'animate-spin' : ''}`} />
            </button>
            <div className="flex items-center gap-2 rounded-lg bg-gray-800 px-3 py-2">
              <Calendar className="h-4 w-4 text-gray-400" />
              <span className="text-sm text-gray-300">
                {format(startDate, 'MMM d')} - {format(endDate, 'MMM d, yyyy')}
              </span>
            </div>
          </div>
        }
      />

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Active Sessions"
          value={data.active_sessions}
          icon={Activity}
          variant="success"
        />
        <StatCard
          title="Peak Concurrent"
          value={data.peak_concurrent_sessions}
          subtitle="Highest concurrent sessions"
          icon={TrendingUp}
          variant="default"
        />
        <StatCard
          title="Today's Sessions"
          value={data.total_sessions_today}
          subtitle="Total sessions started today"
          icon={Monitor}
          variant="default"
        />
        <StatCard
          title="Avg Duration"
          value={formatDuration(data.avg_session_duration_seconds)}
          subtitle="Average session length"
          icon={Clock}
          variant="warning"
        />
      </div>

      {/* Weekly Stats */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white">This Week</h3>
            <div className="mt-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-400">Total Sessions</span>
                <span className="text-lg font-semibold text-white">
                  {data.total_sessions_week.toLocaleString()}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-400">Total Duration</span>
                <span className="text-lg font-semibold text-white">
                  {formatDuration(data.total_session_duration_today_seconds)}
                </span>
              </div>
            </div>
          </div>
        </Card>

        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white">Sessions by Type</h3>
            <div className="mt-4 space-y-2">
              {Object.entries(data.sessions_by_type).map(([type, count]) => (
                <div key={type} className="flex items-center justify-between">
                  <span className="flex items-center gap-2 text-sm text-gray-400">
                    <Badge variant="neutral" className="text-xs">
                      {type}
                    </Badge>
                  </span>
                  <span className="text-sm font-medium text-white">{count as number}</span>
                </div>
              ))}
              {Object.keys(data.sessions_by_type).length === 0 && (
                <p className="text-sm text-gray-500">No data available</p>
              )}
            </div>
          </div>
        </Card>

        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white">Sessions by Environment</h3>
            <div className="mt-4 space-y-2">
              {Object.entries(data.sessions_by_environment).map(([env, count]) => {
                const envColors: Record<string, string> = {
                  production: 'danger',
                  staging: 'warning',
                  development: 'success',
                  test: 'neutral',
                };
                return (
                  <div key={env} className="flex items-center justify-between">
                    <span className="flex items-center gap-2 text-sm text-gray-400">
                      <Badge variant={(envColors[env] as 'danger' | 'warning' | 'success' | 'neutral') || 'neutral'} className="text-xs">
                        {env}
                      </Badge>
                    </span>
                    <span className="text-sm font-medium text-white">{count as number}</span>
                  </div>
                );
              })}
              {Object.keys(data.sessions_by_environment).length === 0 && (
                <p className="text-sm text-gray-500">No data available</p>
              )}
            </div>
          </div>
        </Card>
      </div>

      {/* Sessions Over Time Chart */}
      <Card>
        <CardHeader title="Sessions Over Time" subtitle="Session count trends" />
        <div className="card-body">
          {timeSeriesData.length > 0 ? (
            <div className="space-y-4">
              <MiniChart data={timeSeriesData} />
              <div className="flex justify-between text-xs text-gray-500">
                <span>{format(timeSeriesData[0]?.timestamp || '', 'MMM d')}</span>
                <span>{format(timeSeriesData[timeSeriesData.length - 1]?.timestamp || '', 'MMM d')}</span>
              </div>
            </div>
          ) : (
            <EmptyState title="No time series data available" />
          )}
        </div>
      </Card>

      {/* Detailed Time Series Table */}
      <Card>
        <CardHeader
          title="Detailed Timeline"
          subtitle="Session metrics over time"
          action={
            <Badge variant="neutral">{granularityOptions.find(o => o.value === granularity)?.label} granularity</Badge>
          }
        />
        <div className="card-body">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-700">
                  <th className="pb-3 text-left font-medium text-gray-400">Timestamp</th>
                  <th className="pb-3 text-right font-medium text-gray-400">Sessions</th>
                  <th className="pb-3 text-right font-medium text-gray-400">Duration</th>
                  <th className="pb-3 text-right font-medium text-gray-400">Avg Duration</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {data.sessions_over_time.slice(0, 10).map((entry: { timestamp: string; count: number; duration_seconds: number }, idx: number) => (
                  <tr key={idx}>
                    <td className="py-3 text-white">
                      {format(new Date(entry.timestamp), 'MMM d, HH:mm')}
                    </td>
                    <td className="py-3 text-right text-white">{entry.count}</td>
                    <td className="py-3 text-right text-gray-300">
                      {formatDuration(entry.duration_seconds)}
                    </td>
                    <td className="py-3 text-right text-gray-300">
                      {entry.count > 0 ? formatDuration(entry.duration_seconds / entry.count) : '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        {data.sessions_over_time.length > 10 && (
          <CardFooter className="text-sm text-gray-400">
            Showing first 10 of {data.sessions_over_time.length} entries
          </CardFooter>
        )}
      </Card>
    </div>
  );
};
