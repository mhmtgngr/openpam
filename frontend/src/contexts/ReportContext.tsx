/**
 * ReportContext - React Context for managing report state across components
 * Provides centralized state management for reports, filters, and UI state
 */

import React, { createContext, useContext, useState, useCallback, ReactNode, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'react-hot-toast';
import { reportsApi } from '@/api/reports';
import type {
  ReportSnapshot,
  ReportType,
  ReportFormat,
  ReportStatus,
  ReportListParams,
  ReportSchedule,
  ReportTemplate,
  ComplianceFramework,
  Report,
} from '@/types/reports';

interface ReportContextType {
  // State
  snapshots: ReportSnapshot[];
  selectedSnapshot: ReportSnapshot | null;
  isLoading: boolean;
  filters: ReportListParams;
  totalSnapshots: number;

  // Actions
  setSelectedSnapshot: (snapshot: ReportSnapshot | null) => void;
  setFilters: (filters: ReportListParams) => void;
  refreshSnapshots: () => void;
  generateReport: (data: {
    type: ReportType;
    format: ReportFormat;
    period_start: string;
    period_end: string;
    include_sections?: string[];
    filters?: Record<string, unknown>;
    framework?: ComplianceFramework;
  }) => Promise<{ snapshot_id: string; status: string } | undefined>;
  deleteSnapshot: (id: string) => Promise<void>;
  downloadSnapshot: (id: string) => Promise<void>;
  getSnapshotProgress: (id: string) => Promise<void>;

  // Compliance Reports
  complianceReports: Report[];
  isLoadingCompliance: boolean;
  refreshComplianceReports: () => void;

  // Templates
  templates: ReportTemplate[];
  isLoadingTemplates: boolean;
  refreshTemplates: (type?: ReportType) => void;
}

const ReportContext = createContext<ReportContextType | undefined>(undefined);

interface ReportProviderProps {
  children: ReactNode;
  autoRefresh?: boolean; // Auto-refresh snapshots every 30 seconds
}

const DEFAULT_FILTERS: ReportListParams = {
  limit: 20,
  offset: 0,
  sort_by: 'created_at',
  sort_order: 'desc',
};

export const ReportProvider: React.FC<ReportProviderProps> = ({ children, autoRefresh = true }) => {
  const queryClient = useQueryClient();

  // State
  const [filters, setFilters] = useState<ReportListParams>(DEFAULT_FILTERS);
  const [selectedSnapshot, setSelectedSnapshot] = useState<ReportSnapshot | null>(null);

  // Query: Snapshots list
  const {
    data: snapshotsData,
    isLoading: isLoadingSnapshots,
    refetch: refetchSnapshots,
  } = useQuery({
    queryKey: ['reportSnapshots', filters],
    queryFn: () => reportsApi.listSnapshots(filters),
    staleTime: 30 * 1000,
    refetchInterval: autoRefresh ? 30 * 1000 : false,
  });

  // Query: Compliance reports
  const {
    data: complianceData,
    isLoading: isLoadingCompliance,
    refetch: refetchComplianceReports,
  } = useQuery({
    queryKey: ['complianceReports'],
    queryFn: () => reportsApi.list(),
    staleTime: 5 * 60 * 1000,
  });

  // Query: Templates
  const {
    data: templatesData,
    isLoading: isLoadingTemplates,
    refetch: refetchTemplates,
  } = useQuery({
    queryKey: ['reportTemplates'],
    queryFn: () => reportsApi.listTemplates(),
    staleTime: 10 * 60 * 1000,
  });

  // Mutation: Generate report
  const generateMutation = useMutation({
    mutationFn: (data: {
      type: ReportType;
      format: ReportFormat;
      period_start: string;
      period_end: string;
      include_sections?: string[];
      filters?: Record<string, unknown>;
      framework?: ComplianceFramework;
    }) => reportsApi.generate(data),
    onSuccess: (response) => {
      toast.success(`Report generation started (ID: ${response.job_id})`);
      queryClient.invalidateQueries({ queryKey: ['reportSnapshots'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to generate report: ${error.message}`);
    },
  });

  // Mutation: Delete snapshot
  const deleteMutation = useMutation({
    mutationFn: (id: string) => reportsApi.deleteSnapshot(id),
    onSuccess: () => {
      toast.success('Report deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['reportSnapshots'] });
      if (selectedSnapshot) {
        setSelectedSnapshot(null);
      }
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete report: ${error.message}`);
    },
  });

  // Memoized values
  const snapshots = snapshotsData?.data || [];
  const totalSnapshots = snapshotsData?.pagination?.total || 0;
  const complianceReports = complianceData?.data || [];
  const templates = templatesData || [];

  // Actions
  const handleSetFilters = useCallback((newFilters: ReportListParams) => {
    setFilters((prev) => ({ ...prev, ...newFilters }));
  }, []);

  const handleGenerateReport = useCallback(async (data: {
    type: ReportType;
    format: ReportFormat;
    period_start: string;
    period_end: string;
    include_sections?: string[];
    filters?: Record<string, unknown>;
    framework?: ComplianceFramework;
  }) => {
    const result = await generateMutation.mutateAsync(data);
    return result;
  }, [generateMutation]);

  const handleDeleteSnapshot = useCallback(async (id: string) => {
    await deleteMutation.mutateAsync(id);
  }, [deleteMutation]);

  const handleDownloadSnapshot = useCallback(async (id: string) => {
    try {
      const response = await reportsApi.downloadSnapshot(id);
      if (response.download_url) {
        window.open(response.download_url, '_blank');
        toast.success('Report download started');
      }
    } catch (error) {
      toast.error('Failed to download report');
      throw error;
    }
  }, []);

  const handleGetSnapshotProgress = useCallback(async (id: string) => {
    try {
      await reportsApi.getSnapshotProgress(id);
    } catch (error) {
      console.error('Failed to get snapshot progress:', error);
    }
  }, []);

  const handleRefreshTemplates = useCallback((type?: ReportType) => {
    queryClient.invalidateQueries({ queryKey: ['reportTemplates', type] });
  }, [queryClient]);

  const value: ReportContextType = {
    // State
    snapshots,
    selectedSnapshot,
    setSelectedSnapshot,
    isLoading: isLoadingSnapshots,
    filters,
    setFilters: handleSetFilters,
    totalSnapshots,

    // Actions
    refreshSnapshots: () => refetchSnapshots(),
    generateReport: handleGenerateReport,
    deleteSnapshot: handleDeleteSnapshot,
    downloadSnapshot: handleDownloadSnapshot,
    getSnapshotProgress: handleGetSnapshotProgress,

    // Compliance Reports
    complianceReports,
    isLoadingCompliance,
    refreshComplianceReports: () => refetchComplianceReports(),

    // Templates
    templates,
    isLoadingTemplates,
    refreshTemplates: handleRefreshTemplates,
  };

  return <ReportContext.Provider value={value}>{children}</ReportContext.Provider>;
};

export const useReports = (): ReportContextType => {
  const context = useContext(ReportContext);
  if (context === undefined) {
    throw new Error('useReports must be used within a ReportProvider');
  }
  return context;
};
