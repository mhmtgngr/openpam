/**
 * AnalyticsContext - React Context for managing analytics state across components
 * Provides centralized state management for session metrics, user activity, and anomaly detection
 */

import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'react-hot-toast';
import { analyticsApi } from '@/api/analytics';
import { complianceApi } from '@/api/compliance';
import type { SessionMetrics, UserActivity, DashboardTrends, AnomalyDetection } from '@/types';

interface AnalyticsFilters {
  start_date?: string;
  end_date?: string;
  granularity?: 'hour' | 'day' | 'week' | 'month';
}

interface AnomalyFilters {
  start_date?: string;
  end_date?: string;
  severity?: string;
  type?: string;
  status?: string;
}

interface AnalyticsContextType {
  // Session Metrics
  sessionMetrics: SessionMetrics | null;
  isLoadingMetrics: boolean;
  refreshMetrics: (filters?: AnalyticsFilters) => void;

  // Dashboard Trends
  trends: DashboardTrends | null;
  isLoadingTrends: boolean;
  refreshTrends: (period?: 'day' | 'week' | 'month' | 'quarter') => void;

  // User Activity
  userActivity: UserActivity[] | null;
  isLoadingActivity: boolean;
  refreshActivity: (filters?: {
    start_date?: string;
    end_date?: string;
    user_id?: string;
    limit?: number;
    search?: string;
  }) => void;

  // Anomalies
  anomalies: AnomalyDetection[] | null;
  anomalySummary: {
    total: number;
    by_severity: Record<string, number>;
    by_status: Record<string, number>;
    resolved_this_period: number;
  } | null;
  isLoadingAnomalies: boolean;
  refreshAnomalies: (filters?: AnomalyFilters) => void;

  // Real-time Stats
  realtimeStats: {
    active_sessions: number;
    active_users: number;
    sessions_last_hour: number;
    avg_active_duration: number;
  } | null;
  refreshRealtime: () => void;

  // Actions
  updateAnomalyStatus: (id: string, data: { status: string; notes?: string }) => Promise<void>;
  acknowledgeAnomaly: (id: string) => Promise<void>;
}

const AnalyticsContext = createContext<AnalyticsContextType | undefined>(undefined);

interface AnalyticsProviderProps {
  children: ReactNode;
  autoRefresh?: boolean; // Auto-refresh metrics every 30 seconds
  defaultPeriod?: number; // Default days to look back
}

const DEFAULT_FILTERS: AnalyticsFilters = {
  granularity: 'day',
};

export const AnalyticsProvider: React.FC<AnalyticsProviderProps> = ({
  children,
  autoRefresh = true,
  defaultPeriod = 30,
}) => {
  const queryClient = useQueryClient();
  const [metricsFilters, setMetricsFilters] = useState<AnalyticsFilters>(DEFAULT_FILTERS);
  const [trendsPeriod, setTrendsPeriod] = useState<'day' | 'week' | 'month' | 'quarter'>('week');

  // Query: Session Metrics
  const {
    data: sessionMetrics,
    isLoading: isLoadingMetrics,
    refetch: refetchMetrics,
  } = useQuery({
    queryKey: ['sessionMetrics', metricsFilters],
    queryFn: () => analyticsApi.getSessionMetrics(metricsFilters),
    staleTime: 5 * 60 * 1000,
    refetchInterval: autoRefresh ? 30 * 1000 : false,
  });

  // Query: Dashboard Trends
  const {
    data: trends,
    isLoading: isLoadingTrends,
    refetch: refetchTrends,
  } = useQuery({
    queryKey: ['dashboardTrends', trendsPeriod],
    queryFn: () => analyticsApi.getDashboardTrends(trendsPeriod),
    staleTime: 5 * 60 * 1000,
    refetchInterval: autoRefresh ? 60 * 1000 : false,
  });

  // Query: Real-time Stats
  const {
    data: realtimeStats,
    refetch: refetchRealtime,
  } = useQuery({
    queryKey: ['realtimeStats'],
    queryFn: () => analyticsApi.getRealtimeStats(),
    staleTime: 30 * 1000,
    refetchInterval: autoRefresh ? 30 * 1000 : false,
  });

  // Query: Anomaly Summary
  const {
    data: anomalySummary,
    refetch: refetchAnomalySummary,
  } = useQuery({
    queryKey: ['anomalySummary', metricsFilters],
    queryFn: () => complianceApi.getAnomalySummary({
      start_date: metricsFilters.start_date,
      end_date: metricsFilters.end_date,
    }),
    staleTime: 2 * 60 * 1000,
    refetchInterval: autoRefresh ? 2 * 60 * 1000 : false,
  });

  // Mutation: Update Anomaly Status
  const updateAnomalyMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { status: string; notes?: string } }) =>
      complianceApi.updateAnomaly(id, data),
    onSuccess: () => {
      toast.success('Anomaly updated successfully');
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
      queryClient.invalidateQueries({ queryKey: ['anomalySummary'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to update anomaly: ${error.message}`);
    },
  });

  // Mutation: Acknowledge Anomaly
  const acknowledgeAnomalyMutation = useMutation({
    mutationFn: (id: string) => complianceApi.acknowledgeAnomaly(id),
    onSuccess: () => {
      toast.success('Anomaly acknowledged');
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to acknowledge anomaly: ${error.message}`);
    },
  });

  // Actions
  const handleRefreshMetrics = useCallback((filters?: AnalyticsFilters) => {
    if (filters) {
      setMetricsFilters((prev) => ({ ...prev, ...filters }));
    }
    refetchMetrics();
  }, [refetchMetrics]);

  const handleRefreshTrends = useCallback((period?: 'day' | 'week' | 'month' | 'quarter') => {
    if (period) {
      setTrendsPeriod(period);
    }
    refetchTrends();
  }, [refetchTrends]);

  const handleRefreshActivity = useCallback((filters?: {
    start_date?: string;
    end_date?: string;
    user_id?: string;
    limit?: number;
    search?: string;
  }) => {
    queryClient.invalidateQueries({ queryKey: ['userActivity', filters] });
  }, [queryClient]);

  const handleRefreshAnomalies = useCallback((filters?: AnomalyFilters) => {
    queryClient.invalidateQueries({ queryKey: ['anomalies', filters] });
    refetchAnomalySummary();
  }, [queryClient, refetchAnomalySummary]);

  const handleUpdateAnomalyStatus = useCallback(async (id: string, data: { status: string; notes?: string }) => {
    await updateAnomalyMutation.mutateAsync({ id, data });
  }, [updateAnomalyMutation]);

  const handleAcknowledgeAnomaly = useCallback(async (id: string) => {
    await acknowledgeAnomalyMutation.mutateAsync(id);
  }, [acknowledgeAnomalyMutation]);

  const value: AnalyticsContextType = {
    // Session Metrics
    sessionMetrics: sessionMetrics?.data || null,
    isLoadingMetrics,
    refreshMetrics: handleRefreshMetrics,

    // Dashboard Trends
    trends: trends?.data || null,
    isLoadingTrends,
    refreshTrends: handleRefreshTrends,

    // User Activity
    userActivity: null, // Populated by individual components as needed
    isLoadingActivity: false,
    refreshActivity: handleRefreshActivity,

    // Anomalies
    anomalies: null, // Populated by individual components as needed
    anomalySummary: anomalySummary?.data || null,
    isLoadingAnomalies: false,
    refreshAnomalies: handleRefreshAnomalies,

    // Real-time Stats
    realtimeStats: realtimeStats?.data || null,
    refreshRealtime: refetchRealtime,

    // Actions
    updateAnomalyStatus: handleUpdateAnomalyStatus,
    acknowledgeAnomaly: handleAcknowledgeAnomaly,
  };

  return <AnalyticsContext.Provider value={value}>{children}</AnalyticsContext.Provider>;
};

export const useAnalytics = (): AnalyticsContextType => {
  const context = useContext(AnalyticsContext);
  if (context === undefined) {
    throw new Error('useAnalytics must be used within an AnalyticsProvider');
  }
  return context;
};
