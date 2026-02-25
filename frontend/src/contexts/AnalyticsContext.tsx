/**
 * Analytics Context
 *
 * Provides state management for analytics data across the application.
 * Handles session metrics, user activity, command analysis, and anomaly detection.
 */

import React, { createContext, useContext, useState, useCallback, useEffect, ReactNode } from 'react';
import { analyticsApi } from '@/api/analytics';
import { complianceApi } from '@/api/compliance';
import type {
  SessionMetrics,
  UserActivity,
  CommandFrequency,
  DashboardTrends,
  AnomalyDetection,
  AnomalyListParams,
  ComplianceDashboard,
  ComplianceFramework,
} from '@/types';

// Time range options
export type TimeRange = '24h' | '7d' | '30d' | '90d' | 'custom';

export interface DateRange {
  start_date: string;
  end_date: string;
}

// Analytics state interface
interface AnalyticsState {
  // Data
  sessionMetrics: SessionMetrics | null;
  dashboardTrends: DashboardTrends | null;
  userActivity: UserActivity[];
  commandFrequency: CommandFrequency[];
  anomalies: AnomalyDetection[];
  complianceDashboard: ComplianceDashboard | null;

  // UI State
  timeRange: TimeRange;
  customDateRange: DateRange | null;
  selectedFramework: ComplianceFramework;
  anomalyFilters: AnomalyListParams;
  loading: boolean;
  error: string | null;

  // Real-time state
  realtimeStats: {
    active_sessions: number;
    active_users: number;
    sessions_last_hour: number;
    avg_active_duration: number;
  } | null;
}

// Context interface
interface AnalyticsContextValue extends AnalyticsState {
  // Actions
  setTimeRange: (range: TimeRange) => void;
  setCustomDateRange: (range: DateRange | null) => void;
  setSelectedFramework: (framework: ComplianceFramework) => void;
  setAnomalyFilters: (filters: Partial<AnomalyListParams>) => void;
  refreshSessionMetrics: () => Promise<void>;
  refreshDashboardTrends: () => Promise<void>;
  refreshUserActivity: () => Promise<void>;
  refreshCommandFrequency: () => Promise<void>;
  refreshAnomalies: () => Promise<void>;
  refreshComplianceDashboard: () => Promise<void>;
  refreshRealtimeStats: () => Promise<void>;
  refreshAll: () => Promise<void>;
  acknowledgeAnomaly: (id: string) => Promise<void>;
  updateAnomalyStatus: (id: string, status: string, notes?: string) => Promise<void>;
  clearError: () => void;
}

const AnalyticsContext = createContext<AnalyticsContextValue | undefined>(undefined);

interface AnalyticsProviderProps {
  children: ReactNode;
  autoRefresh?: boolean;
  refreshInterval?: number; // milliseconds
}

// Helper to get date range from time range
const getDateRange = (range: TimeRange): DateRange | null => {
  if (range === 'custom') return null;

  const end = new Date();
  const start = new Date();

  switch (range) {
    case '24h':
      start.setHours(start.getHours() - 24);
      break;
    case '7d':
      start.setDate(start.getDate() - 7);
      break;
    case '30d':
      start.setDate(start.getDate() - 30);
      break;
    case '90d':
      start.setDate(start.getDate() - 90);
      break;
  }

  return {
    start_date: start.toISOString().split('T')[0],
    end_date: end.toISOString().split('T')[0],
  };
};

export const AnalyticsProvider: React.FC<AnalyticsProviderProps> = ({
  children,
  autoRefresh = true,
  refreshInterval = 60000, // 1 minute default
}) => {
  const [state, setState] = useState<AnalyticsState>({
    sessionMetrics: null,
    dashboardTrends: null,
    userActivity: [],
    commandFrequency: [],
    anomalies: [],
    complianceDashboard: null,
    timeRange: '7d',
    customDateRange: null,
    selectedFramework: 'soc2',
    anomalyFilters: {},
    loading: false,
    error: null,
    realtimeStats: null,
  });

  const clearError = useCallback(() => {
    setState((prev) => ({ ...prev, error: null }));
  }, []);

  const setLoading = useCallback((loading: boolean) => {
    setState((prev) => ({ ...prev, loading }));
  }, []);

  const setError = useCallback((error: string | null) => {
    setState((prev) => ({ ...prev, error, loading: false }));
  }, []);

  // Get current date range parameters
  const getDateRangeParams = useCallback((): DateRange => {
    if (state.timeRange === 'custom' && state.customDateRange) {
      return state.customDateRange;
    }
    const range = getDateRange(state.timeRange);
    return range || { start_date: '', end_date: '' };
  }, [state.timeRange, state.customDateRange]);

  // Refresh session metrics
  const refreshSessionMetrics = useCallback(async () => {
    const params = getDateRangeParams();
    try {
      const data = await analyticsApi.getSessionMetrics(params);
      setState((prev) => ({ ...prev, sessionMetrics: data, error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch session metrics');
    }
  }, [getDateRangeParams, setError]);

  // Refresh dashboard trends
  const refreshDashboardTrends = useCallback(async () => {
    try {
      const period = state.timeRange === '24h' ? 'day' : state.timeRange === '7d' ? 'week' : 'month';
      const data = await analyticsApi.getDashboardTrends(period);
      setState((prev) => ({ ...prev, dashboardTrends: data, error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch dashboard trends');
    }
  }, [state.timeRange, setError]);

  // Refresh user activity
  const refreshUserActivity = useCallback(async () => {
    const params = getDateRangeParams();
    try {
      const data = await analyticsApi.getUserActivity({ ...params, limit: 50 });
      setState((prev) => ({ ...prev, userActivity: data.data || [], error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch user activity');
    }
  }, [getDateRangeParams, setError]);

  // Refresh command frequency
  const refreshCommandFrequency = useCallback(async () => {
    const params = getDateRangeParams();
    try {
      const data = await analyticsApi.getCommandFrequency({ ...params, limit: 100 });
      setState((prev) => ({ ...prev, commandFrequency: data.data || [], error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch command frequency');
    }
  }, [getDateRangeParams, setError]);

  // Refresh anomalies
  const refreshAnomalies = useCallback(async () => {
    try {
      const data = await complianceApi.getAnomalies({
        ...state.anomalyFilters,
        limit: 100,
      });
      setState((prev) => ({ ...prev, anomalies: data.data || [], error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch anomalies');
    }
  }, [state.anomalyFilters, setError]);

  // Refresh compliance dashboard
  const refreshComplianceDashboard = useCallback(async () => {
    try {
      const data = await complianceApi.getDashboard(state.selectedFramework);
      setState((prev) => ({ ...prev, complianceDashboard: data, error: null }));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch compliance dashboard');
    }
  }, [state.selectedFramework, setError]);

  // Refresh real-time stats
  const refreshRealtimeStats = useCallback(async () => {
    try {
      const data = await analyticsApi.getRealtimeStats();
      setState((prev) => ({ ...prev, realtimeStats: data, error: null }));
    } catch (err) {
      // Don't set error for real-time stats failures
      console.warn('Failed to fetch real-time stats:', err);
    }
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([
        refreshSessionMetrics(),
        refreshDashboardTrends(),
        refreshUserActivity(),
        refreshCommandFrequency(),
        refreshAnomalies(),
        refreshComplianceDashboard(),
        refreshRealtimeStats(),
      ]);
    } finally {
      setLoading(false);
    }
  }, [
    refreshSessionMetrics,
    refreshDashboardTrends,
    refreshUserActivity,
    refreshCommandFrequency,
    refreshAnomalies,
    refreshComplianceDashboard,
    refreshRealtimeStats,
    setLoading,
  ]);

  // Set time range
  const setTimeRange = useCallback((range: TimeRange) => {
    setState((prev) => ({ ...prev, timeRange: range }));
  }, []);

  // Set custom date range
  const setCustomDateRange = useCallback((range: DateRange | null) => {
    setState((prev) => ({ ...prev, customDateRange: range, timeRange: range ? 'custom' : '7d' }));
  }, []);

  // Set selected framework
  const setSelectedFramework = useCallback((framework: ComplianceFramework) => {
    setState((prev) => ({ ...prev, selectedFramework: framework }));
  }, []);

  // Set anomaly filters
  const setAnomalyFilters = useCallback((filters: Partial<AnomalyListParams>) => {
    setState((prev) => ({
      ...prev,
      anomalyFilters: { ...prev.anomalyFilters, ...filters },
    }));
  }, []);

  // Acknowledge anomaly
  const acknowledgeAnomaly = useCallback(async (id: string) => {
    try {
      await complianceApi.acknowledgeAnomaly(id);
      await refreshAnomalies();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to acknowledge anomaly');
    }
  }, [refreshAnomalies, setError]);

  // Update anomaly status
  const updateAnomalyStatus = useCallback(async (id: string, status: string, notes?: string) => {
    try {
      await complianceApi.updateAnomaly(id, { status: status as 'open' | 'investigating' | 'resolved' | 'false_positive', resolution_notes: notes });
      await refreshAnomalies();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update anomaly status');
    }
  }, [refreshAnomalies, setError]);

  // Initial load
  useEffect(() => {
    refreshAll();
  }, []);

  // Refresh when time range changes
  useEffect(() => {
    if (state.timeRange !== 'custom' || state.customDateRange) {
      refreshSessionMetrics();
      refreshUserActivity();
      refreshCommandFrequency();
    }
  }, [state.timeRange, state.customDateRange]);

  // Refresh when framework changes
  useEffect(() => {
    refreshComplianceDashboard();
  }, [state.selectedFramework]);

  // Refresh when anomaly filters change
  useEffect(() => {
    refreshAnomalies();
  }, [state.anomalyFilters]);

  // Auto-refresh real-time stats
  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      refreshRealtimeStats();
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [autoRefresh, refreshInterval, refreshRealtimeStats]);

  const value: AnalyticsContextValue = {
    ...state,
    setTimeRange,
    setCustomDateRange,
    setSelectedFramework,
    setAnomalyFilters,
    refreshSessionMetrics,
    refreshDashboardTrends,
    refreshUserActivity,
    refreshCommandFrequency,
    refreshAnomalies,
    refreshComplianceDashboard,
    refreshRealtimeStats,
    refreshAll,
    acknowledgeAnomaly,
    updateAnomalyStatus,
    clearError,
  };

  return (
    <AnalyticsContext.Provider value={value}>
      {children}
    </AnalyticsContext.Provider>
  );
};

export const useAnalytics = (): AnalyticsContextValue => {
  const context = useContext(AnalyticsContext);
  if (context === undefined) {
    throw new Error('useAnalytics must be used within an AnalyticsProvider');
  }
  return context;
};

export default AnalyticsContext;
