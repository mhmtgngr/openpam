import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { analyticsApi } from '../../api/analytics';

interface DataPoint {
  timestamp: string;
  value: number;
}

interface SessionTrendsProps {
  metric?: string;
  dateFrom?: string;
  dateTo?: string;
}

export const SessionTrends: React.FC<SessionTrendsProps> = ({
  metric = 'sessions',
  dateFrom,
  dateTo,
}) => {
  const [data, setData] = useState<DataPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTrends = async () => {
      setLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = { metric };
        if (dateFrom) params.from = dateFrom;
        if (dateTo) params.to = dateTo;

        const response = await analyticsApi.getTimeSeries(params);
        setData(response || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load trends');
      } finally {
        setLoading(false);
      }
    };

    fetchTrends();
  }, [metric, dateFrom, dateTo]);

  if (loading) {
    return <LoadingState message="Loading trends..." />;
  }

  if (error) {
    return (
      <Card>
        <div className="card-body">
          <div className="text-center text-danger-400">
            <p className="font-semibold">Error Loading Trends</p>
            <p className="text-sm text-gray-400">{error}</p>
          </div>
        </div>
      </Card>
    );
  }

  if (data.length === 0) {
    return (
      <Card>
        <div className="card-body">
          <EmptyState message="No trend data available" />
        </div>
      </Card>
    );
  }

  const maxValue = Math.max(...data.map((d) => d.value));
  const minValue = Math.min(...data.map((d) => d.value));

  return (
    <Card>
      <div className="card-body">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold capitalize text-white">{metric} Trends</h3>
          <select
            value={metric}
            onChange={(e) => {
              // Would trigger parent callback to change metric
              window.location.href = `?metric=${e.target.value}`;
            }}
            className="select rounded px-3 py-1 text-sm bg-gray-800 text-white"
          >
            <option value="sessions">Sessions</option>
            <option value="active_sessions">Active Sessions</option>
            <option value="commands">Commands</option>
            <option value="users">Users</option>
          </select>
        </div>

        {/* Simple bar chart visualization */}
        <div className="relative h-64">
          <div className="flex items-end justify-between h-full gap-1">
            {data.map((point, index) => {
              const height = maxValue > 0 ? (point.value / maxValue) * 100 : 0;
              const isPeak = point.value === maxValue;
              const isLow = point.value === minValue;

              return (
                <div
                  key={index}
                  className="flex-1 flex flex-col items-center group"
                >
                  <div className="relative w-full">
                    <div
                      className={`w-full rounded-t transition-all ${
                        isPeak
                          ? 'bg-success-500'
                          : isLow
                          ? 'bg-primary-300'
                          : 'bg-primary-500'
                      } group-hover:bg-primary-400`}
                      style={{ height: `${Math.max(height, 5)}%` }}
                    />
                    <div className="opacity-0 group-hover:opacity-100 absolute -top-10 left-1/2 transform -translate-x-1/2 bg-gray-900 text-white text-xs px-2 py-1 rounded whitespace-nowrap transition-opacity z-10">
                      {point.value}
                    </div>
                  </div>
                  <span className="text-xs text-gray-500 mt-2 truncate w-full text-center">
                    {new Date(point.timestamp).toLocaleDateString(undefined, {
                      month: 'short',
                      day: 'numeric',
                    })}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Summary stats */}
        <div className="grid grid-cols-3 gap-4 mt-6 pt-4 border-t border-gray-700">
          <div className="text-center">
            <p className="text-sm text-gray-400">Peak</p>
            <p className="text-lg font-semibold text-success-400">{maxValue}</p>
          </div>
          <div className="text-center">
            <p className="text-sm text-gray-400">Average</p>
            <p className="text-lg font-semibold text-white">
              {Math.round(
                data.reduce((sum, d) => sum + d.value, 0) / data.length
              )}
            </p>
          </div>
          <div className="text-center">
            <p className="text-sm text-gray-400">Low</p>
            <p className="text-lg font-semibold text-primary-400">{minValue}</p>
          </div>
        </div>
      </div>
    </Card>
  );
};

interface EmptyStateProps {
  message: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message }) => (
  <div className="text-center py-8 text-gray-500">
    <p>{message}</p>
  </div>
);
