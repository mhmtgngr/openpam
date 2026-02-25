/**
 * AnomalyChart Component
 *
 * Visualizes anomalies detected in the system over time.
 * Supports timeline view, severity breakdown, and type distribution.
 */

import React, { useMemo } from 'react';
import type { AnomalyDetection, AnomalySeverity, AnomalyType } from '@/types';
import { format } from 'date-fns';

interface AnomalyChartProps {
  anomalies: AnomalyDetection[];
  view?: 'timeline' | 'severity' | 'type';
  onAnomalyClick?: (anomaly: AnomalyDetection) => void;
  className?: string;
}

const SEVERITY_COLORS: Record<AnomalySeverity, string> = {
  critical: 'bg-red-500',
  high: 'bg-orange-500',
  medium: 'bg-yellow-500',
  low: 'bg-blue-500',
};

const SEVERITY_BORDER_COLORS: Record<AnomalySeverity, string> = {
  critical: 'border-red-500',
  high: 'border-orange-500',
  medium: 'border-yellow-500',
  low: 'border-blue-500',
};

const SEVERITY_TEXT_COLORS: Record<AnomalySeverity, string> = {
  critical: 'text-red-500',
  high: 'text-orange-500',
  medium: 'text-yellow-500',
  low: 'text-blue-500',
};

interface ChartDataPoint {
  date: string;
  critical: number;
  high: number;
  medium: number;
  low: number;
  total: number;
}

interface SeverityData {
  severity: AnomalySeverity;
  count: number;
  percentage: number;
}

interface TypeData {
  type: AnomalyType;
  count: number;
  percentage: number;
}

export const AnomalyChart: React.FC<AnomalyChartProps> = ({
  anomalies,
  view = 'timeline',
  onAnomalyClick,
  className = '',
}) => {
  // Process data for timeline view
  const timelineData = useMemo(() => {
    const dataMap = new Map<string, ChartDataPoint>();

    // Sort anomalies by date
    const sortedAnomalies = [...anomalies].sort(
      (a, b) => new Date(a.detected_at).getTime() - new Date(b.detected_at).getTime()
    );

    // Get date range (last 30 days)
    const now = new Date();
    const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);

    // Initialize data points for each day
    for (let i = 0; i < 30; i++) {
      const date = new Date(thirtyDaysAgo.getTime() + i * 24 * 60 * 60 * 1000);
      const dateKey = format(date, 'yyyy-MM-dd');
      dataMap.set(dateKey, {
        date: dateKey,
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
        total: 0,
      });
    }

    // Populate data
    sortedAnomalies.forEach((anomaly) => {
      const dateKey = format(new Date(anomaly.detected_at), 'yyyy-MM-dd');
      const existing = dataMap.get(dateKey);
      if (existing) {
        existing[anomaly.severity]++;
        existing.total++;
      }
    });

    return Array.from(dataMap.values());
  }, [anomalies]);

  // Process data for severity breakdown
  const severityData = useMemo(() => {
    const counts = {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
    } as Record<AnomalySeverity, number>;

    anomalies.forEach((a) => {
      counts[a.severity]++;
    });

    const total = anomalies.length || 1;

    return Object.entries(counts).map(([severity, count]) => ({
      severity: severity as AnomalySeverity,
      count,
      percentage: (count / total) * 100,
    })) as SeverityData[];
  }, [anomalies]);

  // Process data for type distribution
  const typeData = useMemo(() => {
    const typeMap = new Map<AnomalyType, number>();

    anomalies.forEach((a) => {
      typeMap.set(a.type, (typeMap.get(a.type) || 0) + 1);
    });

    const total = anomalies.length || 1;

    return Array.from(typeMap.entries())
      .map(([type, count]) => ({
        type,
        count,
        percentage: (count / total) * 100,
      }))
      .sort((a, b) => b.count - a.count) as TypeData[];
  }, [anomalies]);

  // Timeline View
  if (view === 'timeline') {
    const maxValue = Math.max(...timelineData.map((d) => d.total), 1);

    return (
      <div className={`anomaly-chart-timeline ${className}`}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            Anomaly Timeline (Last 30 Days)
          </h3>
          <div className="flex gap-4 text-sm">
            {Object.entries(SEVERITY_COLORS).map(([severity, color]) => (
              <div key={severity} className="flex items-center gap-2">
                <div className={`w-3 h-3 rounded-full ${color}`} />
                <span className="capitalize text-gray-600 dark:text-gray-400">
                  {severity}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div className="relative h-48 border-b border-l border-gray-200 dark:border-gray-700">
          {/* Y-axis grid lines */}
          {[0, 25, 50, 75, 100].map((percent) => (
            <div
              key={percent}
              className="absolute w-full border-t border-gray-100 dark:border-gray-800"
              style={{ bottom: `${percent}%` }}
            >
              <span className="absolute -left-8 -top-2 text-xs text-gray-500">
                {Math.round((maxValue * percent) / 100)}
              </span>
            </div>
          ))}

          {/* Data bars */}
          <svg className="absolute inset-0 w-full h-full" preserveAspectRatio="none">
            {timelineData.map((day, index) => {
              const x = (index / timelineData.length) * 100;
              const barWidth = (100 / timelineData.length) * 0.8;

              return (
                <g key={day.date}>
                  {/* Stacked bars for each severity */}
                  {(['critical', 'high', 'medium', 'low'] as AnomalySeverity[]).map((severity) => {
                    const count = day[severity];
                    if (count === 0) return null;

                    const height = (count / maxValue) * 100;
                    const y = 100 - height;

                    return (
                      <rect
                        key={severity}
                        x={`${x}%`}
                        y={`${y}%`}
                        width={`${barWidth}%`}
                        height={`${height}%`}
                        fill={SEVERITY_COLORS[severity].replace('bg-', '')}
                        opacity={0.8}
                        className="hover:opacity-100 transition-opacity cursor-pointer"
                        onClick={() => {
                          const dayAnomalies = anomalies.filter(
                            (a) => format(new Date(a.detected_at), 'yyyy-MM-dd') === day.date && a.severity === severity
                          );
                          if (dayAnomalies.length > 0 && onAnomalyClick) {
                            onAnomalyClick(dayAnomalies[0]);
                          }
                        }}
                      >
                        <title>
                          {format(new Date(day.date), 'MMM dd')}: {count} {severity} anomalies
                        </title>
                      </rect>
                    );
                  })}
                </g>
              );
            })}
          </svg>
        </div>

        {/* X-axis labels */}
        <div className="flex justify-between mt-2 text-xs text-gray-500">
          {timelineData
            .filter((_, i) => i % 5 === 0)
            .map((day) => (
              <span key={day.date}>{format(new Date(day.date), 'MMM dd')}</span>
            ))}
        </div>

        {/* Summary stats */}
        <div className="grid grid-cols-4 gap-4 mt-6">
          {(['critical', 'high', 'medium', 'low'] as AnomalySeverity[]).map((severity) => {
            const data = severityData.find((d) => d.severity === severity);
            return (
              <div key={severity} className="text-center">
                <div className={`text-2xl font-bold ${SEVERITY_TEXT_COLORS[severity]}`}>
                  {data?.count || 0}
                </div>
                <div className="text-sm text-gray-500 capitalize">{severity}</div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  // Severity Breakdown View
  if (view === 'severity') {
    return (
      <div className={`anomaly-chart-severity ${className}`}>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Severity Breakdown
        </h3>

        <div className="space-y-4">
          {severityData.map((data) => (
            <div
              key={data.severity}
              className="flex items-center gap-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-750 transition-colors"
              onClick={() => {
                const matching = anomalies.find((a) => a.severity === data.severity);
                if (matching && onAnomalyClick) onAnomalyClick(matching);
              }}
            >
              <div
                className={`w-4 h-4 rounded ${SEVERITY_COLORS[data.severity]} flex-shrink-0`}
              />
              <div className="flex-1">
                <div className="flex justify-between mb-1">
                  <span className="font-medium capitalize text-gray-900 dark:text-white">
                    {data.severity}
                  </span>
                  <span className="text-gray-600 dark:text-gray-400">
                    {data.count} ({data.percentage.toFixed(1)}%)
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full ${SEVERITY_COLORS[data.severity]} transition-all duration-500`}
                    style={{ width: `${data.percentage}%` }}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>

        {anomalies.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No anomalies detected in the selected time range
          </div>
        )}
      </div>
    );
  }

  // Type Distribution View
  if (view === 'type') {
    const typeLabels: Record<AnomalyType, string> = {
      unusual_access_time: 'Unusual Access Time',
      unusual_location: 'Unusual Location',
      privileged_escalation: 'Privileged Escalation',
      bulk_data_access: 'Bulk Data Access',
      command_injection: 'Command Injection',
      ransomware_indicators: 'Ransomware Indicators',
      impossible_travel: 'Impossible Travel',
      account_takeover: 'Account Takeover',
      credential_theft: 'Credential Theft',
      excessive_failed_logins: 'Excessive Failed Logins',
    };

    return (
      <div className={`anomaly-chart-type ${className}`}>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
          Anomaly Types
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {typeData.map((data) => {
            const typeAnomalies = anomalies.filter((a) => a.type === data.type);
            const severities = typeAnomalies.reduce(
              (acc, a) => {
                acc[a.severity]++;
                return acc;
              },
              { critical: 0, high: 0, medium: 0, low: 0 } as Record<AnomalySeverity, number>
            );

            return (
              <div
                key={data.type}
                className={`p-4 rounded-lg border-2 border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 cursor-pointer transition-colors`}
                onClick={() => {
                  if (typeAnomalies.length > 0 && onAnomalyClick) {
                    onAnomalyClick(typeAnomalies[0]);
                  }
                }}
              >
                <div className="flex justify-between items-start mb-2">
                  <h4 className="font-medium text-gray-900 dark:text-white">
                    {typeLabels[data.type]}
                  </h4>
                  <span className="text-lg font-semibold text-gray-700 dark:text-gray-300">
                    {data.count}
                  </span>
                </div>

                <div className="text-sm text-gray-500 mb-3">
                  {data.percentage.toFixed(1)}% of total
                </div>

                {/* Mini severity bar */}
                <div className="flex h-2 rounded-full overflow-hidden">
                  {(['critical', 'high', 'medium', 'low'] as AnomalySeverity[]).map((severity) => {
                    const count = severities[severity];
                    if (count === 0) return null;
                    const width = (count / data.count) * 100;
                    return (
                      <div
                        key={severity}
                        className={SEVERITY_COLORS[severity]}
                        style={{ width: `${width}%` }}
                        title={`${count} ${severity}`}
                      />
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>

        {anomalies.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No anomalies detected in the selected time range
          </div>
        )}
      </div>
    );
  }

  return null;
};

export default AnomalyChart;
