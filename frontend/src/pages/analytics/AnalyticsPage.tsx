import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  BarChart3,
  Users,
  Terminal,
  ShieldCheck,
  AlertTriangle,
  LayoutDashboard,
} from 'lucide-react';
import { analyticsApi } from '@/api/analytics';
import { complianceApi } from '@/api/compliance';
import { SessionMetrics, UserActivity, CommandAnalysis, ComplianceDashboard, AnomalyDetection } from '@/components/analytics';
import { Card } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import clsx from 'clsx';

type AnalyticsTab = 'overview' | 'sessions' | 'users' | 'commands' | 'compliance' | 'anomalies';

interface TabConfig {
  id: AnalyticsTab;
  label: string;
  icon: React.ElementType;
}

const tabs: TabConfig[] = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'sessions', label: 'Session Metrics', icon: BarChart3 },
  { id: 'users', label: 'User Activity', icon: Users },
  { id: 'commands', label: 'Command Analysis', icon: Terminal },
  { id: 'compliance', label: 'Compliance', icon: ShieldCheck },
  { id: 'anomalies', label: 'Anomalies', icon: AlertTriangle },
];

interface QuickStatProps {
  title: string;
  value: string | number;
  change?: number;
  icon: React.ElementType;
  variant?: 'default' | 'success' | 'warning' | 'danger';
}

const QuickStat: React.FC<QuickStatProps> = ({ title, value, change, icon: Icon, variant = 'default' }) => {
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
          <div>
            <p className="text-sm text-gray-400">{title}</p>
            <p className="mt-2 text-2xl font-bold text-white">{value}</p>
            {change !== undefined && (
              <p className={clsx(
                'mt-1 text-sm',
                change >= 0 ? 'text-success-400' : 'text-danger-400'
              )}>
                {change >= 0 ? '+' : ''}{change}% from last week
              </p>
            )}
          </div>
          <div className={clsx('rounded-lg bg-gray-800 p-3', variantClasses[variant])}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};

const OverviewTab: React.FC = () => {
  const { data: trends, isLoading } = useQuery({
    queryKey: ['dashboardTrends'],
    queryFn: () => analyticsApi.getDashboardTrends('week').then((res) => res.data),
  });

  const { data: realtime } = useQuery({
    queryKey: ['realtimeStats'],
    queryFn: () => analyticsApi.getRealtimeStats().then((res) => res.data),
    refetchInterval: 30 * 1000, // Refresh every 30 seconds
  });

  const { data: anomalySummary } = useQuery({
    queryKey: ['anomalySummary'],
    queryFn: () => complianceApi.getAnomalySummary().then((res) => res.data),
    refetchInterval: 5 * 60 * 1000,
  });

  const { data: commandSummary } = useQuery({
    queryKey: ['commandRiskSummary'],
    queryFn: () => analyticsApi.getCommandRiskSummary().then((res) => res.data),
    refetchInterval: 5 * 60 * 1000,
  });

  if (isLoading) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <LoadingState message="Loading overview..." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Quick Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <QuickStat
          title="Active Sessions"
          value={realtime?.active_sessions || 0}
          icon={BarChart3}
          variant="success"
        />
        <QuickStat
          title="Active Users"
          value={realtime?.active_users || 0}
          icon={Users}
          variant="default"
        />
        <QuickStat
          title="High-Risk Commands"
          value={commandSummary?.high_risk_commands || 0}
          icon={Terminal}
          variant="danger"
        />
        <QuickStat
          title="Open Anomalies"
          value={(anomalySummary?.by_severity?.critical || 0) + (anomalySummary?.by_severity?.high || 0)}
          icon={AlertTriangle}
          variant="warning"
        />
      </div>

      {/* Weekly Trends */}
      {trends && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white">Session Trends</h3>
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-400">This Week</span>
                  <span className="text-lg font-semibold text-white">{trends.sessions.current}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-400">Change</span>
                  <span className={clsx(
                    'text-sm font-medium',
                    trends.sessions.change_percent >= 0 ? 'text-success-400' : 'text-danger-400'
                  )}>
                    {trends.sessions.change_percent >= 0 ? '+' : ''}{trends.sessions.change_percent}%
                  </span>
                </div>
              </div>
            </div>
          </Card>

          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white">User Activity</h3>
              <div className="mt-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-400">Active Users</span>
                  <span className="text-lg font-semibold text-white">{trends.users.current}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-gray-400">Change</span>
                  <span className={clsx(
                    'text-sm font-medium',
                    trends.users.change_percent >= 0 ? 'text-success-400' : 'text-danger-400'
                  )}>
                    {trends.users.change_percent >= 0 ? '+' : ''}{trends.users.change_percent}%
                  </span>
                </div>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Compliance Overview */}
      <Card>
        <div className="card-body">
          <h3 className="text-lg font-semibold text-white">Quick Actions</h3>
          <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
            <button
              onClick={() => document.getElementById('tab-compliance')?.click()}
              className="rounded-lg border border-gray-700 p-4 text-left transition-colors hover:bg-gray-800"
            >
              <ShieldCheck className="mb-2 h-5 w-5 text-primary-400" />
              <h4 className="font-medium text-white">Run Compliance Assessment</h4>
              <p className="mt-1 text-sm text-gray-400">Check your current compliance status</p>
            </button>
            <button
              onClick={() => document.getElementById('tab-anomalies')?.click()}
              className="rounded-lg border border-gray-700 p-4 text-left transition-colors hover:bg-gray-800"
            >
              <AlertTriangle className="mb-2 h-5 w-5 text-warning-400" />
              <h4 className="font-medium text-white">Review Anomalies</h4>
              <p className="mt-1 text-sm text-gray-400">Investigate detected security anomalies</p>
            </button>
            <button
              onClick={() => document.getElementById('tab-commands')?.click()}
              className="rounded-lg border border-gray-700 p-4 text-left transition-colors hover:bg-gray-800"
            >
              <Terminal className="mb-2 h-5 w-5 text-danger-400" />
              <h4 className="font-medium text-white">Command Analysis</h4>
              <p className="mt-1 text-sm text-gray-400">Review high-risk command usage</p>
            </button>
          </div>
        </div>
      </Card>
    </div>
  );
};

export const AnalyticsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState<AnalyticsTab>('overview');

  const currentTab = tabs.find((t) => t.id === activeTab);
  const CurrentIcon = currentTab?.icon || LayoutDashboard;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Analytics</h1>
        <p className="mt-1 text-sm text-gray-400">
          Comprehensive analytics and compliance monitoring
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-700">
        <nav className="flex gap-1 overflow-x-auto">
          {tabs.map((tab) => {
            const TabIcon = tab.icon;
            return (
              <button
                key={tab.id}
                id={`tab-${tab.id}`}
                onClick={() => setActiveTab(tab.id)}
                className={clsx(
                  'flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors',
                  activeTab === tab.id
                    ? 'border-primary-500 text-primary-400'
                    : 'border-transparent text-gray-400 hover:border-gray-600 hover:text-white'
                )}
              >
                <TabIcon className="h-4 w-4" />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Tab Content */}
      <div>
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'sessions' && <SessionMetrics />}
        {activeTab === 'users' && <UserActivity />}
        {activeTab === 'commands' && <CommandAnalysis />}
        {activeTab === 'compliance' && <ComplianceDashboard />}
        {activeTab === 'anomalies' && <AnomalyDetection />}
      </div>
    </div>
  );
};
