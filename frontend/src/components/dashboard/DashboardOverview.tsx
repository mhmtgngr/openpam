import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { analyticsApi } from '../../api/analytics';

interface SessionSummary {
  total_sessions: number;
  active_sessions: number;
  avg_duration: number;
  peak_concurrent: number;
  sessions_by_type: Record<string, number>;
}

interface UserActivitySummary {
  active_users: number;
  total_commands: number;
  high_risk_users: number;
}

interface DashboardMetrics {
  session_metrics: SessionSummary;
  user_activity: UserActivitySummary;
  timestamp: string;
}

interface DashboardOverviewProps {
  dateFrom?: string;
  dateTo?: string;
}

export const DashboardOverview: React.FC<DashboardOverviewProps> = ({
  dateFrom,
  dateTo,
}) => {
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMetrics = async () => {
      setLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = {};
        if (dateFrom) params.from = dateFrom;
        if (dateTo) params.to = dateTo;

        const response = await analyticsApi.getDashboard(params);
        setMetrics(response.dashboard);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard metrics');
      } finally {
        setLoading(false);
      }
    };

    fetchMetrics();
  }, [dateFrom, dateTo]);

  if (loading) {
    return <LoadingState message="Loading dashboard..." />;
  }

  if (error || !metrics) {
    return (
      <Card className="p-6">
        <div className="text-center text-red-600">
          <p className="font-semibold">Error Loading Dashboard</p>
          <p className="text-sm">{error || 'Unknown error'}</p>
        </div>
      </Card>
    );
  }

  const { session_metrics, user_activity } = metrics;

  return (
    <div className="space-y-6">
      {/* Session Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Total Sessions"
          value={session_metrics.total_sessions}
          icon="📊"
          trend={null}
        />
        <MetricCard
          title="Active Sessions"
          value={session_metrics.active_sessions}
          icon="🔴"
          trend={null}
        />
        <MetricCard
          title="Avg Duration"
          value={`${Math.round(session_metrics.avg_duration / 60)}m`}
          icon="⏱️"
          trend={null}
        />
        <MetricCard
          title="Peak Concurrent"
          value={session_metrics.peak_concurrent}
          icon="⚡"
          trend={null}
        />
      </div>

      {/* User Activity */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <MetricCard
          title="Active Users"
          value={user_activity.active_users}
          icon="👥"
          trend={null}
        />
        <MetricCard
          title="Commands Executed"
          value={user_activity.total_commands}
          icon="⌨️"
          trend={null}
        />
        <MetricCard
          title="High Risk Users"
          value={user_activity.high_risk_users}
          icon="⚠️"
          trend={null}
          alert={user_activity.high_risk_users > 0}
        />
      </div>

      {/* Sessions by Type */}
      <Card className="p-6">
        <h3 className="text-lg font-semibold mb-4">Sessions by Type</h3>
        {Object.keys(session_metrics.sessions_by_type).length > 0 ? (
          <div className="space-y-3">
            {Object.entries(session_metrics.sessions_by_type).map(([type, count]) => (
              <div key={type} className="flex items-center justify-between">
                <span className="capitalize text-gray-700">{type}</span>
                <div className="flex items-center gap-3">
                  <div className="w-48 bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full"
                      style={{
                        width: `${(count / session_metrics.total_sessions) * 100}%`,
                      }}
                    />
                  </div>
                  <span className="text-sm font-medium w-12 text-right">{count}</span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState message="No session data available" />
        )}
      </Card>
    </div>
  );
};

interface MetricCardProps {
  title: string;
  value: number | string;
  icon: string;
  trend?: number | null;
  alert?: boolean;
}

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  icon,
  trend,
  alert = false,
}) => (
  <Card className={`p-4 ${alert ? 'border-red-500 border-2' : ''}`}>
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-600">{title}</p>
        <p className={`text-2xl font-bold ${alert ? 'text-red-600' : 'text-gray-900'}`}>
          {value}
        </p>
        {trend !== null && trend !== undefined && (
          <p className={`text-xs ${trend >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            {trend >= 0 ? '↑' : '↓'} {Math.abs(trend)}%
          </p>
        )}
      </div>
      <span className="text-3xl">{icon}</span>
    </div>
  </Card>
);

interface EmptyStateProps {
  message: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message }) => (
  <div className="text-center py-8 text-gray-500">
    <p>{message}</p>
  </div>
);
