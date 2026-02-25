import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format, subDays } from 'date-fns';
import {
  Terminal,
  Search,
  Filter,
  AlertTriangle,
  Shield,
  Calendar,
  Download,
  ChevronDown,
  ChevronUp,
  AlertCircle,
} from 'lucide-react';
import { analyticsApi, type CommandAnalysisParams } from '@/api/analytics';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge, StatusBadge } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Button } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import { Pagination } from '@/components/common';
import type { CommandFrequency as CommandFrequencyType } from '@/types';
import clsx from 'clsx';

const periodOptions = [
  { value: '7', label: 'Last 7 days' },
  { value: '14', label: 'Last 14 days' },
  { value: '30', label: 'Last 30 days' },
  { value: '90', label: 'Last 90 days' },
];

const riskLevelOptions = [
  { value: '', label: 'All Risk Levels' },
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
  { value: 'info', label: 'Info' },
];

const riskLevelConfig: Record<string, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType }> = {
  critical: { variant: 'danger', icon: AlertCircle },
  high: { variant: 'danger', icon: AlertTriangle },
  medium: { variant: 'warning', icon: Shield },
  low: { variant: 'success', icon: Shield },
  info: { variant: 'neutral', icon: Terminal },
};

interface RiskSummaryCardProps {
  title: string;
  count: number;
  total: number;
  variant: 'danger' | 'warning' | 'success' | 'neutral';
}

const RiskSummaryCard: React.FC<RiskSummaryCardProps> = ({ title, count, total, variant }) => {
  const percentage = total > 0 ? Math.round((count / total) * 100) : 0;

  const variantClasses: Record<typeof variant, string> = {
    danger: 'bg-danger-500/10 text-danger-400',
    warning: 'bg-warning-500/10 text-warning-400',
    success: 'bg-success-500/10 text-success-400',
    neutral: 'bg-gray-500/10 text-gray-400',
  };

  return (
    <div className={clsx('card', variantClasses[variant])}>
      <div className="card-body">
        <p className="text-sm font-medium uppercase opacity-80">{title}</p>
        <p className="mt-2 text-3xl font-bold">{count.toLocaleString()}</p>
        <p className="mt-1 text-sm opacity-70">{percentage}% of total</p>
      </div>
    </div>
  );
};

interface CommandRowProps {
  command: CommandFrequencyType;
  expanded: boolean;
  onToggle: () => void;
}

const CommandRow: React.FC<CommandRowProps> = ({ command, expanded, onToggle }) => {
  const config = riskLevelConfig[command.risk_level] || riskLevelConfig.info;
  const RiskIcon = config.icon;

  return (
    <>
      <tr className="border-b border-gray-800 hover:bg-gray-800/50">
        <td className="py-4">
          <div className="flex items-center gap-3">
            <div className={clsx('rounded-lg p-2', {
              'bg-danger-500/20 text-danger-400': command.risk_level === 'critical' || command.risk_level === 'high',
              'bg-warning-500/20 text-warning-400': command.risk_level === 'medium',
              'bg-success-500/20 text-success-400': command.risk_level === 'low',
              'bg-gray-500/20 text-gray-400': command.risk_level === 'info',
            })}>
              <RiskIcon className="h-5 w-5" />
            </div>
            <div>
              <code className="text-sm font-mono text-white">{command.command}</code>
            </div>
          </div>
        </td>
        <td className="py-4 text-center">
          <Badge variant={config.variant}>{command.risk_level}</Badge>
        </td>
        <td className="py-4 text-center">
          <span className="text-lg font-semibold text-white">{command.count}</span>
        </td>
        <td className="py-4 text-center text-sm text-gray-400">
          {format(new Date(command.first_seen_at), 'MMM d, yyyy')}
        </td>
        <td className="py-4 text-center text-sm text-gray-400">
          {format(new Date(command.last_seen_at), 'MMM d, yyyy')}
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
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              {/* Users who used this command */}
              <div>
                <h4 className="mb-3 text-sm font-medium text-gray-300">Users ({command.users.length})</h4>
                <div className="space-y-2">
                  {command.users.length > 0 ? (
                    command.users.map((user) => (
                      <div
                        key={user.user_id}
                        className="flex items-center justify-between rounded-lg bg-gray-900/50 px-3 py-2"
                      >
                        <span className="text-sm text-white">{user.user_name}</span>
                        <Badge variant="neutral">{user.count}x</Badge>
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-gray-500">No user data available</p>
                  )}
                </div>
              </div>

              {/* Targets where this command was used */}
              <div>
                <h4 className="mb-3 text-sm font-medium text-gray-300">Targets ({command.targets.length})</h4>
                <div className="space-y-2">
                  {command.targets.length > 0 ? (
                    command.targets.map((target) => (
                      <div
                        key={target.target_id}
                        className="flex items-center justify-between rounded-lg bg-gray-900/50 px-3 py-2"
                      >
                        <span className="text-sm text-white">{target.target_name}</span>
                        <Badge variant="neutral">{target.count}x</Badge>
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-gray-500">No target data available</p>
                  )}
                </div>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  );
};

export const CommandAnalysis: React.FC = () => {
  const [days, setDays] = useState(30);
  const [search, setSearch] = useState('');
  const [riskLevel, setRiskLevel] = useState('');
  const [offset, setOffset] = useState(0);
  const [expandedRow, setExpandedRow] = useState<string | null>(null);
  const limit = 20;

  const { data: commandsData, isLoading: isLoadingCommands } = useQuery({
    queryKey: ['commandFrequency', days, search, riskLevel, offset, limit],
    queryFn: () =>
      analyticsApi.getCommandFrequency({
        start_date: subDays(new Date(), days).toISOString(),
        end_date: new Date().toISOString(),
        search,
        risk_level: riskLevel,
        limit,
        offset,
      }),
  });

  const { data: riskSummary, isLoading: isLoadingSummary } = useQuery({
    queryKey: ['commandRiskSummary', days],
    queryFn: () =>
      analyticsApi.getCommandRiskSummary({
        start_date: subDays(new Date(), days).toISOString(),
        end_date: new Date().toISOString(),
      }),
  });

  const handleExport = async () => {
    try {
      const response = await analyticsApi.exportCommandAnalysis({
        start_date: subDays(new Date(), days).toISOString(),
        end_date: new Date().toISOString(),
        search,
        risk_level: riskLevel,
        format: 'csv',
      });

      window.open(response.download_url, '_blank');
    } catch {
      // Error handled by interceptor
    }
  };

  const totalCommands = commandsData?.pagination?.total || 0;

  return (
    <div className="space-y-6">
      <CardHeader
        title="Command Analysis"
        subtitle="Track command usage patterns and identify risky commands"
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
            <Button
              variant="secondary"
              leftIcon={<Download className="h-4 w-4" />}
              onClick={handleExport}
            >
              Export
            </Button>
          </div>
        }
      />

      {/* Risk Summary Cards */}
      {!isLoadingSummary && riskSummary && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <RiskSummaryCard
            title="High Risk"
            count={riskSummary.high_risk_commands}
            total={riskSummary.total_commands}
            variant="danger"
          />
          <RiskSummaryCard
            title="Medium Risk"
            count={riskSummary.medium_risk_commands}
            total={riskSummary.total_commands}
            variant="warning"
          />
          <RiskSummaryCard
            title="Low Risk"
            count={riskSummary.low_risk_commands}
            total={riskSummary.total_commands}
            variant="success"
          />
          <RiskSummaryCard
            title="Total Commands"
            count={riskSummary.total_commands}
            total={riskSummary.total_commands}
            variant="neutral"
          />
        </div>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search commands..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={riskLevelOptions}
              value={riskLevel}
              onChange={(e) => {
                setRiskLevel(e.target.value);
                setOffset(0);
              }}
            />
          </div>
        </div>
      </Card>

      {/* Command List */}
      <Card>
        {isLoadingCommands ? (
          <div className="card-body">
            <LoadingState message="Loading command analysis..." />
          </div>
        ) : !commandsData || commandsData.data.length === 0 ? (
          <div className="card-body">
            <EmptyState title="No commands found" />
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-gray-700">
                    <th className="pb-3 text-left text-sm font-medium text-gray-400">
                      <div className="flex items-center gap-2">
                        <Terminal className="h-4 w-4" />
                        Command
                      </div>
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Risk Level
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Count
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      First Seen
                    </th>
                    <th className="pb-3 text-center text-sm font-medium text-gray-400">
                      Last Seen
                    </th>
                    <th className="pb-3 text-right" />
                  </tr>
                </thead>
                <tbody>
                  {commandsData.data.map((command) => (
                    <CommandRow
                      key={command.command}
                      command={command}
                      expanded={expandedRow === command.command}
                      onToggle={() =>
                        setExpandedRow(expandedRow === command.command ? null : command.command)
                      }
                    />
                  ))}
                </tbody>
              </table>
            </div>

            {totalCommands > limit && (
              <div className="card-footer">
                <Pagination
                  total={totalCommands}
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
