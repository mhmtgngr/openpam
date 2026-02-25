/**
 * AnomalyChart - Visual timeline component for displaying detected anomalies
 * Shows anomalies over time with severity indicators and filtering capabilities
 */

import React, { useState } from 'react';
import { format } from 'date-fns';
import {
  AlertTriangle,
  AlertOctagon,
  Activity,
  ChevronDown,
  ChevronUp,
  Filter,
  Search,
} from 'lucide-react';
import { Card, CardHeader } from '@/components/common';
import { Badge } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import type { AnomalyDetection, AnomalySeverity } from '@/types';
import clsx from 'clsx';

export interface AnomalyChartProps {
  anomalies: AnomalyDetection[];
  onAnomalyClick?: (anomaly: AnomalyDetection) => void;
  startDate?: Date;
  endDate?: Date;
  className?: string;
}

const severityConfig: Record<AnomalySeverity, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType; color: string; bgColor: string }> = {
  critical: { variant: 'danger', icon: AlertOctagon, color: 'text-danger-400', bgColor: 'bg-danger-500/20' },
  high: { variant: 'danger', icon: AlertTriangle, color: 'text-danger-400', bgColor: 'bg-danger-500/20' },
  medium: { variant: 'warning', icon: AlertTriangle, color: 'text-warning-400', bgColor: 'bg-warning-500/20' },
  low: { variant: 'neutral', icon: Activity, color: 'text-gray-400', bgColor: 'bg-gray-500/20' },
};

const severityOrder: AnomalySeverity[] = ['critical', 'high', 'medium', 'low'];

const statusLabels: Record<string, string> = {
  open: 'Open',
  investigating: 'Investigating',
  resolved: 'Resolved',
  false_positive: 'False Positive',
};

export const AnomalyChart: React.FC<AnomalyChartProps> = ({
  anomalies,
  onAnomalyClick,
  startDate,
  endDate,
  className,
}) => {
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  // Group anomalies by date
  const groupedAnomalies = React.useMemo(() => {
    const filtered = anomalies.filter((anomaly) => {
      if (severityFilter !== 'all' && anomaly.severity !== severityFilter) return false;
      if (statusFilter !== 'all' && anomaly.status !== statusFilter) return false;
      if (searchQuery && !anomaly.title.toLowerCase().includes(searchQuery.toLowerCase()) &&
          !anomaly.description.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    });

    const groups = filtered.reduce((acc, anomaly) => {
      const dateKey = format(new Date(anomaly.detected_at), 'yyyy-MM-dd');
      if (!acc[dateKey]) {
        acc[dateKey] = [];
      }
      acc[dateKey].push(anomaly);
      return acc;
    }, {} as Record<string, AnomalyDetection[]>);

    // Sort by date (newest first)
    return Object.fromEntries(
      Object.entries(groups).sort(([a], [b]) => b.localeCompare(a))
    );
  }, [anomalies, severityFilter, statusFilter, searchQuery]);

  const toggleGroup = (dateKey: string) => {
    setExpandedGroups((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(dateKey)) {
        newSet.delete(dateKey);
      } else {
        newSet.add(dateKey);
      }
      return newSet;
    });
  };

  const toggleAll = () => {
    const allKeys = Object.keys(groupedAnomalies);
    if (expandedGroups.size === allKeys.length) {
      setExpandedGroups(new Set());
    } else {
      setExpandedGroups(new Set(allKeys));
    }
  };

  const getSeverityCount = () => {
    const counts = { critical: 0, high: 0, medium: 0, low: 0 };
    anomalies.forEach((a) => {
      if (counts[a.severity as AnomalySeverity] !== undefined) {
        counts[a.severity as AnomalySeverity]++;
      }
    });
    return counts;
  };

  const severityCounts = getSeverityCount();

  return (
    <Card className={className}>
      <CardHeader
        title="Anomaly Timeline"
        subtitle="Visual timeline of detected security anomalies"
        action={
          <div className="flex items-center gap-2">
            <Badge variant="danger" className={severityCounts.critical > 0 ? 'animate-pulse' : ''}>
              {severityCounts.critical} Critical
            </Badge>
            <Badge variant="danger">{severityCounts.high} High</Badge>
            <Badge variant="warning">{severityCounts.medium} Medium</Badge>
          </div>
        }
      />

      <div className="card-body space-y-4">
        {/* Filters */}
        <div className="flex flex-wrap gap-3">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
            <Input
              placeholder="Search anomalies..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
          <Select
            options={[
              { value: 'all', label: 'All Severities' },
              { value: 'critical', label: 'Critical' },
              { value: 'high', label: 'High' },
              { value: 'medium', label: 'Medium' },
              { value: 'low', label: 'Low' },
            ]}
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value)}
            className="w-40"
          />
          <Select
            options={[
              { value: 'all', label: 'All Statuses' },
              { value: 'open', label: 'Open' },
              { value: 'investigating', label: 'Investigating' },
              { value: 'resolved', label: 'Resolved' },
              { value: 'false_positive', label: 'False Positive' },
            ]}
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="w-40"
          />
        </div>

        {/* Expand/Collapse All */}
        <div className="flex items-center justify-between border-b border-gray-800 pb-2">
          <p className="text-sm text-gray-400">
            {Object.values(groupedAnomalies).flat().length} anomalies in {Object.keys(groupedAnomalies).length} days
          </p>
          <Button
            variant="ghost"
            size="sm"
            onClick={toggleAll}
            leftIcon={expandedGroups.size === Object.keys(groupedAnomalies).length ?
              <ChevronUp className="h-4 w-4" /> :
              <ChevronDown className="h-4 w-4" />
            }
          >
            {expandedGroups.size === Object.keys(groupedAnomalies).length ? 'Collapse All' : 'Expand All'}
          </Button>
        </div>

        {/* Timeline */}
        <div className="space-y-4">
          {Object.keys(groupedAnomalies).length === 0 ? (
            <div className="py-8 text-center text-gray-500">
              <Filter className="mx-auto h-12 w-12 mb-2 opacity-50" />
              <p>No anomalies match the current filters</p>
            </div>
          ) : (
            Object.entries(groupedAnomalies).map(([dateKey, dayAnomalies]) => {
              const isExpanded = expandedGroups.has(dateKey);
              const sortedAnomalies = dayAnomalies.sort((a, b) =>
                severityOrder.indexOf(a.severity as AnomalySeverity) -
                severityOrder.indexOf(b.severity as AnomalySeverity)
              );

              return (
                <div key={dateKey} className="border-l-2 border-gray-800 pl-4">
                  {/* Date Header */}
                  <button
                    onClick={() => toggleGroup(dateKey)}
                    className="flex items-center gap-2 text-left hover:text-white transition-colors"
                  >
                    {isExpanded ? (
                      <ChevronUp className="h-4 w-4 text-gray-500" />
                    ) : (
                      <ChevronDown className="h-4 w-4 text-gray-500" />
                    )}
                    <span className="text-sm font-medium text-gray-300">
                      {format(new Date(dateKey), 'EEEE, MMMM d, yyyy')}
                    </span>
                    <Badge variant="neutral" className="text-xs">
                      {dayAnomalies.length}
                    </Badge>
                  </button>

                  {/* Anomalies for this day */}
                  {isExpanded && (
                    <div className="mt-3 space-y-2">
                      {sortedAnomalies.map((anomaly) => {
                        const config = severityConfig[anomaly.severity as AnomalySeverity];
                        const SeverityIcon = config.icon;
                        const statusLabel = statusLabels[anomaly.status] || anomaly.status;

                        return (
                          <div
                            key={anomaly.id}
                            onClick={() => onAnomalyClick?.(anomaly)}
                            className={clsx(
                              'flex items-start gap-3 rounded-lg border p-3 transition-all cursor-pointer',
                              'hover:shadow-md',
                              {
                                'border-danger-500/30 bg-danger-500/5': anomaly.severity === 'critical' || anomaly.severity === 'high',
                                'border-warning-500/30 bg-warning-500/5': anomaly.severity === 'medium',
                                'border-gray-700 bg-gray-900/50': anomaly.severity === 'low',
                              }
                            )}
                          >
                            {/* Severity Icon */}
                            <div className={clsx('rounded-lg p-2', config.bgColor, config.color)}>
                              <SeverityIcon className="h-4 w-4" />
                            </div>

                            {/* Content */}
                            <div className="flex-1 min-w-0">
                              <div className="flex items-start justify-between gap-2">
                                <h4 className="text-sm font-medium text-white truncate">
                                  {anomaly.title}
                                </h4>
                                <Badge variant={config.variant} className="shrink-0 text-xs capitalize">
                                  {anomaly.severity}
                                </Badge>
                              </div>
                              <p className="text-xs text-gray-400 line-clamp-2 mt-1">
                                {anomaly.description}
                              </p>

                              {/* Metadata */}
                              <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500">
                                <span>{format(new Date(anomaly.detected_at), 'HH:mm')}</span>
                                <span>•</span>
                                <span className="capitalize">{statusLabel}</span>
                                {anomaly.user_name && (
                                  <>
                                    <span>•</span>
                                    <span>{anomaly.user_name}</span>
                                  </>
                                )}
                                <span>•</span>
                                <span>Confidence: {Math.round(anomaly.confidence_score * 100)}%</span>
                              </div>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </Card>
  );
};

export default AnomalyChart;
