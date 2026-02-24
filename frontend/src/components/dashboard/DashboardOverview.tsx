import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { analyticsApi } from '../../api/analytics';
import type { SessionMetrics } from '@/types';
import { Activity, Clock, Users, BarChart3 } from 'lucide-react';

interface DashboardOverviewProps {
  dateFrom?: string;
  dateTo?: string;
}

interface DashboardResponse {
  metrics: SessionMetrics;
  trends: {
    sessions: { current: number; previous: number; change_percent: number };
    users: { current: number; previous: number; change_percent: number };
  };
  realtime: {
    active_sessions: number;
    active_users: number;
    sessions_last_hour: number;
    avg_active_duration: number;
  };
}

export const DashboardOverview: React.FC<DashboardOverviewProps> = ({
  dateFrom,
  dateTo,
}) => {
  const [data, setData] = useState<DashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchMetrics = async () => {
      setLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = {};
        if (dateFrom) params.start_date = dateFrom;
        if (dateTo) params.end_date = dateTo;

        const response = await analyticsApi.getDashboard(params);
        setData(response.data);
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

  if (error || !data) {
    return (
      <Card>
        <div className="card-body">
          <div className="text-center text-danger-400">
            <p className="font-semibold">Error Loading Dashboard</p>
            <p className="text-sm text-gray-400">{error || 'Unknown error'}</p>
          </div>
        </div>
      </Card>
    );
  }

  const { metrics, realtime } = data;

  return (
    <div className="space-y-6">
      {/* Session Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Active Sessions"
          value={realtime.active_sessions}
          icon={Activity}
          color="bg-success-400/20 text-success-400"
        />
        <MetricCard
          title="Sessions Today"
          value={metrics.total_sessions_today || 0}
          icon={BarChart3}
          color="bg-primary-400/20 text-primary-400"
        />
        <MetricCard
          title="Active Users"
          value={realtime.active_users}
          icon={Users}
          color="bg-warning-400/20 text-warning-400"
        />
        <MetricCard
          title="Avg Duration"
          value={`${Math.round(metrics.avg_session_duration_seconds / 60)}m`}
          icon={Clock}
          color="bg-info-400/20 text-primary-400"
        />
      </div>

      {/* Sessions by Type */}
      {metrics.sessions_by_type && Object.keys(metrics.sessions_by_type).length > 0 && (
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Sessions by Type</h3>
            <div className="space-y-3">
              {Object.entries(metrics.sessions_by_type).map(([type, count]) => (
                <div key={type} className="flex items-center justify-between">
                  <span className="capitalize text-gray-300">{type}</span>
                  <div className="flex items-center gap-3">
                    <div className="w-48 bg-gray-700 rounded-full h-2">
                      <div
                        className="bg-primary-500 h-2 rounded-full"
                        style={{
                          width: `${(count / (metrics.total_sessions_today || 1)) * 100}%`,
                        }}
                      />
                    </div>
                    <span className="text-sm font-medium w-12 text-right text-white">{count as number}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}
    </div>
  );
};

interface MetricCardProps {
  title: string;
  value: number | string;
  icon: React.ElementType;
  color: string;
}

const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  icon: Icon,
  color,
}) => (
  <Card>
    <div className="card-body">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-400">{title}</p>
          <p className="text-2xl font-bold text-white">
            {value}
          </p>
        </div>
        <div className={`rounded-lg p-3 ${color}`}>
          <Icon className="h-6 w-6" />
        </div>
      </div>
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
