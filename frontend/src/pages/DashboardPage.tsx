import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  Users,
  Server,
  Activity,
  Clock,
  AlertTriangle,
  ArrowRight,
} from 'lucide-react';
import { dashboardApi } from '@/api/dashboard';
import { useAuth } from '@/contexts/AuthContext';
import { Card, CardHeader, ErrorState } from '@/components/common';
import clsx from 'clsx';

export const DashboardPage: React.FC = () => {
  const { user } = useAuth();

  const { data: stats, isLoading, isError, refetch } = useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: () => dashboardApi.getStats(),
  });

  const { data: activity } = useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: () =>
      dashboardApi.getActivity({ limit: 10 }),
  });

  const { data: pendingRequests } = useQuery({
    queryKey: ['dashboard', 'pending'],
    queryFn: () =>
      dashboardApi.getPendingRequests(),
    enabled: ['admin', 'super_admin', 'operator'].includes(user?.role || ''),
  });

  const StatCard: React.FC<{
    title: string;
    value: number | string;
    icon: React.ElementType;
    color: string;
    trend?: { value: number; isPositive: boolean };
    link?: string;
  }> = ({ title, value, icon: Icon, color, trend, link }) => {
    const content = (
      <div className={clsx('card cursor-pointer transition-colors hover:bg-gray-800/80', link && 'group')}>
        <div className="card-body">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-400">{title}</p>
              <p className="mt-2 text-3xl font-semibold text-white">{value}</p>
              {trend && (
                <p
                  className={clsx(
                    'mt-2 flex items-center text-xs',
                    trend.isPositive ? 'text-success-400' : 'text-danger-400'
                  )}
                >
                  {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value)}% from last week
                </p>
              )}
            </div>
            <div
              className={clsx(
                'flex h-12 w-12 items-center justify-center rounded-lg',
                color
              )}
            >
              <Icon className="h-6 w-6" />
            </div>
          </div>
        </div>
      </div>
    );

    if (link) {
      return <Link to={link}>{content}</Link>;
    }

    return content;
  };

  return (
    <div className="space-y-6">
      {/* Welcome */}
      <div>
        <h1 className="text-2xl font-bold text-white">
          Welcome back, {user?.first_name}!
        </h1>
        <p className="mt-1 text-gray-400">
          Here's what's happening with your privileged access management today.
        </p>
      </div>

      {/* Stats Grid */}
      {isError ? (
        <ErrorState
          message="Unable to load dashboard data. The server may be temporarily unavailable."
          onRetry={() => refetch()}
        />
      ) : isLoading ? (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="card animate-pulse">
              <div className="card-body">
                <div className="h-4 w-24 rounded bg-gray-700" />
                <div className="mt-4 h-8 w-16 rounded bg-gray-700" />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            title="Total Users"
            value={stats?.users.total || 0}
            icon={Users}
            color="bg-primary-400/20 text-primary-400"
            link="/users"
          />
          <StatCard
            title="Active Targets"
            value={`${stats?.targets.online || 0}/${stats?.targets.total || 0}`}
            icon={Server}
            color="bg-success-400/20 text-success-400"
            link="/targets"
          />
          <StatCard
            title="Active Sessions"
            value={stats?.sessions.active || 0}
            icon={Activity}
            color="bg-warning-400/20 text-warning-400"
            link="/sessions"
          />
          <StatCard
            title="Pending Requests"
            value={stats?.requests.pending || 0}
            icon={Clock}
            color="bg-danger-400/20 text-danger-400"
            link="/approvals"
          />
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Pending Approvals */}
        {pendingRequests && pendingRequests.data && pendingRequests.data.length > 0 && (
          <Card>
            <CardHeader
              title="Pending Approvals"
              action={
                <Link
                  to="/approvals"
                  className="flex items-center text-sm text-primary-400 hover:text-primary-300"
                >
                  View all
                  <ArrowRight className="ml-1 h-4 w-4" />
                </Link>
              }
            />
            <div className="divide-y divide-gray-700">
              {pendingRequests.data.slice(0, 5).map((request) => {
                const req = request as { id: string; user?: { first_name: string; last_name: string }; target?: { name: string }; created_at: string };
                return (
                  <Link
                    key={req.id}
                    to={`/approvals/${req.id}`}
                    className="block px-6 py-4 transition-colors hover:bg-gray-800/50"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-white">
                          {req.user?.first_name} {req.user?.last_name}
                        </p>
                        <p className="text-xs text-gray-400">
                          Request for {req.target?.name || 'Unknown'}
                        </p>
                      </div>
                      <span className="text-xs text-gray-500">
                        {new Date(req.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </Link>
                );
              })}
            </div>
          </Card>
        )}

        {/* Activity Feed */}
        <Card>
          <CardHeader title="Recent Activity" />
          <div className="divide-y divide-gray-700">
            {activity && activity.data && activity.data.length > 0 ? (
              activity.data.slice(0, 5).map((item) => (
                <div key={item.id} className="px-6 py-4">
                  <p className="text-sm text-white">{item.message}</p>
                  <p className="mt-1 text-xs text-gray-500">
                    {new Date(item.timestamp).toLocaleString()}
                  </p>
                </div>
              ))
            ) : (
              <p className="px-6 py-4 text-sm text-gray-400">No recent activity</p>
            )}
          </div>
        </Card>

        {/* Expiring Credentials */}
        {stats && stats.credentials.expiring_soon > 0 && (
          <Card className="lg:col-span-2">
            <CardHeader
              title="Expiring Credentials"
              subtitle={`${stats.credentials.expiring_soon} credentials expiring soon`}
              action={
                <Link
                  to="/credentials?status=expiring"
                  className="flex items-center text-sm text-primary-400 hover:text-primary-300"
                >
                  View all
                  <ArrowRight className="ml-1 h-4 w-4" />
                </Link>
              }
            />
            <div className="flex items-center gap-3 rounded-md bg-warning-400/10 px-4 py-3">
              <AlertTriangle className="h-5 w-5 text-warning-400" />
              <p className="text-sm text-gray-300">
                Some credentials are expiring soon. Consider rotating them to maintain
                security.
              </p>
            </div>
          </Card>
        )}
      </div>
    </div>
  );
};
