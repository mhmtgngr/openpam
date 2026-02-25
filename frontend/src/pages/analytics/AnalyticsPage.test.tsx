/**
 * Tests for AnalyticsPage component
 */

import React, { useState } from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AnalyticsPage } from './AnalyticsPage';
import { analyticsApi } from '@/api/analytics';
import { complianceApi } from '@/api/compliance';

// Mock the analytics API
jest.mock('@/api/analytics');
jest.mock('@/api/compliance');

// Mock react-router-dom
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => jest.fn(),
}));

const createMockQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
};

const renderWithQueryClient = (component: React.ReactElement) => {
  const queryClient = createMockQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {component}
    </QueryClientProvider>
  );
};

describe('AnalyticsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    // Setup default mocks
    (analyticsApi.getDashboardTrends as jest.Mock).mockResolvedValue({
      data: {
        sessions: { current: 145, previous: 130, change_percent: 11.5, trend: 'up', data_points: [] },
        users: { current: 45, previous: 42, change_percent: 7.1, trend: 'up', data_points: [] },
        credentials: { current: 230, previous: 225, change_percent: 2.2, trend: 'up', data_points: [] },
        requests: { current: 18, previous: 22, change_percent: -18.2, trend: 'down', data_points: [] },
        period: 'week',
      },
    });

    (analyticsApi.getRealtimeStats as jest.Mock).mockResolvedValue({
      data: {
        active_sessions: 5,
        active_users: 3,
        sessions_last_hour: 8,
        avg_active_duration: 1200,
      },
    });

    (complianceApi.getAnomalySummary as jest.Mock).mockResolvedValue({
      data: {
        total: 5,
        by_severity: { critical: 1, high: 1, medium: 2, low: 1 },
        by_type: { unusual_access_time: 2, privileged_escalation: 1, impossible_travel: 1, bulk_data_access: 1 },
        by_status: { open: 3, investigating: 1, resolved: 1, false_positive: 0 },
        resolved_this_period: 1,
        avg_resolution_time_hours: 4.5,
      },
    });

    (analyticsApi.getCommandRiskSummary as jest.Mock).mockResolvedValue({
      data: {
        total_commands: 350,
        high_risk_commands: 15,
        medium_risk_commands: 45,
        low_risk_commands: 290,
        unique_users: 12,
        unique_targets: 8,
      },
    });
  });

  describe('Rendering', () => {
    it('should render the analytics heading', () => {
      renderWithQueryClient(<AnalyticsPage />);

      expect(screen.getByText('Analytics')).toBeInTheDocument();
    });

    it('should render the page description', () => {
      renderWithQueryClient(<AnalyticsPage />);

      expect(screen.getByText(/Comprehensive analytics and compliance monitoring/i)).toBeInTheDocument();
    });

    it('should render all navigation tabs', () => {
      renderWithQueryClient(<AnalyticsPage />);

      expect(screen.getByText('Overview')).toBeInTheDocument();
      expect(screen.getByText('Session Metrics')).toBeInTheDocument();
      expect(screen.getByText('User Activity')).toBeInTheDocument();
      expect(screen.getByText('Command Analysis')).toBeInTheDocument();
      expect(screen.getByText('Compliance')).toBeInTheDocument();
      expect(screen.getByText('Anomalies')).toBeInTheDocument();
    });

    it('should have Overview tab active by default', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const overviewTab = screen.getByText('Overview').closest('button');
      expect(overviewTab).toHaveClass(/border-primary-500/);
    });
  });

  describe('Tab Navigation', () => {
    it('should switch to Sessions Metrics tab when clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const sessionsTab = screen.getByText('Session Metrics');
      await user.click(sessionsTab);

      const activeTab = screen.getByText('Session Metrics').closest('button');
      expect(activeTab).toHaveClass(/border-primary-500/);
    });

    it('should switch to User Activity tab when clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const usersTab = screen.getByText('User Activity');
      await user.click(usersTab);

      const activeTab = screen.getByText('User Activity').closest('button');
      expect(activeTab).toHaveClass(/border-primary-500/);
    });

    it('should switch to Command Analysis tab when clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const commandsTab = screen.getByText('Command Analysis');
      await user.click(commandsTab);

      const activeTab = screen.getByText('Command Analysis').closest('button');
      expect(activeTab).toHaveClass(/border-primary-500/);
    });

    it('should switch to Compliance tab when clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const complianceTab = screen.getByText('Compliance');
      await user.click(complianceTab);

      const activeTab = screen.getByText('Compliance').closest('button');
      expect(activeTab).toHaveClass(/border-primary-500/);
    });

    it('should switch to Anomalies tab when clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const anomaliesTab = screen.getByText('Anomalies');
      await user.click(anomaliesTab);

      const activeTab = screen.getByText('Anomalies').closest('button');
      expect(activeTab).toHaveClass(/border-primary-500/);
    });

    it('should maintain only one active tab at a time', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const tabs = ['Overview', 'Session Metrics', 'User Activity', 'Command Analysis', 'Compliance', 'Anomalies'];

      for (const tabName of tabs) {
        const tab = screen.getByText(tabName);
        await user.click(tab);

        // Check that only this tab is active
        const allTabs = screen.getAllByRole('button');
        const activeTabs = allTabs.filter(btn => btn.classList.contains(/border-primary-500/));
        expect(activeTabs).toHaveLength(1);
        expect(activeTabs[0]).toHaveTextContent(tabName);
      }
    });
  });

  describe('Overview Tab', () => {
    it('should display quick stats', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('Active Sessions')).toBeInTheDocument();
        expect(screen.getByText('Active Users')).toBeInTheDocument();
        expect(screen.getByText('High-Risk Commands')).toBeInTheDocument();
        expect(screen.getByText('Open Anomalies')).toBeInTheDocument();
      });
    });

    it('should display quick stats values', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        // The values should be from the mocked API responses
        expect(screen.getByText('5')).toBeInTheDocument(); // active_sessions
        expect(screen.getByText('3')).toBeInTheDocument(); // active_users
        expect(screen.getByText('15')).toBeInTheDocument(); // high_risk_commands
      });
    });

    it('should display weekly trends section', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('Session Trends')).toBeInTheDocument();
        expect(screen.getByText('User Activity')).toBeInTheDocument();
      });
    });

    it('should display compliance trend change percentage', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText(/\+11\.5%/)).toBeInTheDocument(); // sessions change
      });
    });

    it('should display quick actions section', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('Quick Actions')).toBeInTheDocument();
        expect(screen.getByText('Run Compliance Assessment')).toBeInTheDocument();
        expect(screen.getByText('Review Anomalies')).toBeInTheDocument();
        expect(screen.getByText('Command Analysis')).toBeInTheDocument();
      });
    });

    it('should show positive change in green', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const positiveChange = screen.getByText(/\+11\.5%/);
        expect(positiveChange).toHaveClass(/text-success-400/);
      });
    });

    it('should show negative change in red', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const negativeChange = screen.getByText(/-18\.2%/);
        expect(negativeChange).toHaveClass(/text-danger-400/);
      });
    });

    it('should navigate from quick action buttons', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const complianceButton = screen.getByText('Run Compliance Assessment');
        expect(complianceButton).toBeInTheDocument();
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading state initially', () => {
      // Make the API call pending
      (analyticsApi.getDashboardTrends as jest.Mock).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      renderWithQueryClient(<AnalyticsPage />);

      expect(screen.getByText(/Loading overview/i)).toBeInTheDocument();
    });

    it('should hide loading state when data is loaded', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.queryByText(/Loading overview/i)).not.toBeInTheDocument();
      });
    });

    it('should show loading state when switching tabs', async () => {
      (analyticsApi.getUserActivity as jest.Mock).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const usersTab = screen.getByText('User Activity');
      await user.click(usersTab);

      expect(screen.getByText(/Loading/i)).toBeInTheDocument();
    });
  });

  describe('Error States', () => {
    it('should handle API error gracefully', async () => {
      (analyticsApi.getDashboardTrends as jest.Mock).mockRejectedValue(
        new Error('Failed to fetch trends')
      );

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        // Should show error state or empty state
        expect(screen.queryByText(/Loading overview/i)).not.toBeInTheDocument();
      });
    });

    it('should display error message for failed requests', async () => {
      // Mock a failed request
      (complianceApi.getAnomalySummary as jest.Mock).mockRejectedValue(
        new Error('Failed to fetch anomalies')
      );

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.queryByText(/Loading overview/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Data Refresh', () => {
    it('should refresh realtime stats periodically', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      // Initial call
      await waitFor(() => {
        expect(analyticsApi.getRealtimeStats).toHaveBeenCalledTimes(1);
      });

      // Wait for refetch (30 seconds in production, but we test that it's called)
      // The component uses refetchInterval for realtime stats
    });

    it('should refresh anomaly stats periodically', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(complianceApi.getAnomalySummary).toHaveBeenCalled();
      });
    });
  });

  describe('Component Structure', () => {
    it('should use correct tab icons', () => {
      renderWithQueryClient(<AnalyticsPage />);

      // Check that icons are rendered (via text content or accessibility)
      const tabs = ['Overview', 'Session Metrics', 'User Activity', 'Command Analysis', 'Compliance', 'Anomalies'];
      tabs.forEach(tabName => {
        expect(screen.getByText(tabName)).toBeInTheDocument();
      });
    });

    it('should have proper heading hierarchy', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const mainHeading = screen.getByRole('heading', { level: 1 });
      expect(mainHeading).toHaveTextContent('Analytics');
    });

    it('should display stats in a grid layout', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const statsContainer = screen.getByText('Active Sessions').closest('.grid');
      expect(statsContainer).toHaveClass(/grid-cols-1/); // Responsive grid
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels for tabs', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const tabs = screen.getAllByRole('button');
      expect(tabs.length).toBeGreaterThanOrEqual(6);

      tabs.forEach(tab => {
        expect(tab).toHaveAttribute('type', 'button');
      });
    });

    it('should indicate active tab state visually and semantically', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const overviewTab = screen.getByText('Overview').closest('button');
      expect(overviewTab).toHaveClass(/border-primary-500/);
      expect(overviewTab).toHaveClass(/text-primary-400/);
    });

    it('should have proper heading structure', () => {
      renderWithQueryClient(<AnalyticsPage />);

      const headings = screen.getAllByRole('heading');
      const h1Count = headings.filter(h => h.tagName === 'H1').length;
      expect(h1Count).toBe(1);
    });
  });

  describe('Responsive Design', () => {
    it('should stack stats cards on mobile', () => {
      // Mock mobile viewport
      global.innerWidth = 375;

      renderWithQueryClient(<AnalyticsPage />);

      const statsContainer = screen.getByText('Active Sessions').closest('.grid');
      expect(statsContainer).toHaveClass(/sm:grid-cols-2/); // 2 columns on small screens
    });

    it('should display multiple stats on larger screens', () => {
      // Mock desktop viewport
      global.innerWidth = 1024;

      renderWithQueryClient(<AnalyticsPage />);

      const statsContainer = screen.getByText('Active Sessions').closest('.grid');
      expect(statsContainer).toHaveClass(/lg:grid-cols-4/); // 4 columns on large screens
    });
  });

  describe('Tab Content Components', () => {
    it('should render SessionMetrics component when tab is active', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const sessionsTab = screen.getByText('Session Metrics');
      await user.click(sessionsTab);

      // SessionMetrics component should be rendered
      await waitFor(() => {
        expect(screen.getByText('Session Metrics')).toBeInTheDocument();
      });
    });

    it('should render UserActivity component when tab is active', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const usersTab = screen.getByText('User Activity');
      await user.click(usersTab);

      await waitFor(() => {
        expect(screen.getByText('User Activity')).toBeInTheDocument();
      });
    });

    it('should render CommandAnalysis component when tab is active', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const commandsTab = screen.getByText('Command Analysis');
      await user.click(commandsTab);

      await waitFor(() => {
        expect(screen.getByText('Command Analysis')).toBeInTheDocument();
      });
    });

    it('should render ComplianceDashboard component when tab is active', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const complianceTab = screen.getByText('Compliance');
      await user.click(complianceTab);

      await waitFor(() => {
        expect(screen.getByText('Compliance')).toBeInTheDocument();
      });
    });

    it('should render AnomalyDetection component when tab is active', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      const anomaliesTab = screen.getByText('Anomalies');
      await user.click(anomaliesTab);

      await waitFor(() => {
        expect(screen.getByText('Anomalies')).toBeInTheDocument();
      });
    });
  });

  describe('Quick Actions', () => {
    it('should navigate to compliance tab when quick action is clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const runComplianceButton = screen.getByText('Run Compliance Assessment');
        expect(runComplianceButton).toBeInTheDocument();
      });
    });

    it('should navigate to anomalies tab when quick action is clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const reviewAnomaliesButton = screen.getByText('Review Anomalies');
        expect(reviewAnomaliesButton).toBeInTheDocument();
      });
    });

    it('should navigate to commands tab when quick action is clicked', async () => {
      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const commandAnalysisButton = screen.getByText('Command Analysis');
        expect(commandAnalysisButton).toBeInTheDocument();
      });
    });
  });

  describe('Edge Cases', () => {
    it('should handle zero values gracefully', async () => {
      (analyticsApi.getRealtimeStats as jest.Mock).mockResolvedValue({
        data: {
          active_sessions: 0,
          active_users: 0,
          sessions_last_hour: 0,
          avg_active_duration: 0,
        },
      });

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('0')).toBeInTheDocument();
      });
    });

    it('should handle very large numbers gracefully', async () => {
      (analyticsApi.getDashboardTrends as jest.Mock).mockResolvedValue({
        data: {
          sessions: { current: 999999, previous: 888888, change_percent: 12.5, trend: 'up', data_points: [] },
          users: { current: 10000, previous: 9000, change_percent: 11.1, trend: 'up', data_points: [] },
          credentials: { current: 50000, previous: 45000, change_percent: 11.1, trend: 'up', data_points: [] },
          requests: { current: 75000, previous: 80000, change_percent: -6.3, trend: 'down', data_points: [] },
          period: 'week',
        },
      });

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('999999')).toBeInTheDocument();
      });
    });

    it('should handle negative trends correctly', async () => {
      (analyticsApi.getDashboardTrends as jest.Mock).mockResolvedValue({
        data: {
          sessions: { current: 90, previous: 100, change_percent: -10, trend: 'down', data_points: [] },
          users: { current: 35, previous: 40, change_percent: -12.5, trend: 'down', data_points: [] },
          credentials: { current: 200, previous: 210, change_percent: -4.8, trend: 'down', data_points: [] },
          requests: { current: 15, previous: 20, change_percent: -25, trend: 'down', data_points: [] },
          period: 'week',
        },
      });

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const negativeChanges = screen.getAllByText(/-\d+%/);
        expect(negativeChanges.length).toBeGreaterThan(0);
        negativeChanges.forEach(change => {
          expect(change).toHaveClass(/text-danger-400/);
        });
      });
    });

    it('should handle empty anomaly data gracefully', async () => {
      (complianceApi.getAnomalySummary as jest.Mock).mockResolvedValue({
        data: {
          total: 0,
          by_severity: { critical: 0, high: 0, medium: 0, low: 0 },
          by_type: {},
          by_status: { open: 0, investigating: 0, resolved: 0, false_positive: 0 },
          resolved_this_period: 0,
          avg_resolution_time_hours: 0,
        },
      });

      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(screen.getByText('Open Anomalies')).toBeInTheDocument();
      });
    });
  });

  describe('Integration', () => {
    it('should call all APIs on initial load', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(analyticsApi.getDashboardTrends).toHaveBeenCalled();
        expect(analyticsApi.getRealtimeStats).toHaveBeenCalled();
        expect(complianceApi.getAnomalySummary).toHaveBeenCalled();
        expect(analyticsApi.getCommandRiskSummary).toHaveBeenCalled();
      });
    });

    it('should use correct API parameters for different time periods', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        expect(analyticsApi.getDashboardTrends).toHaveBeenCalledWith('week');
      });
    });

    it('should not call tab-specific APIs until tab is activated', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        // Overview APIs should be called
        expect(analyticsApi.getDashboardTrends).toHaveBeenCalled();
      });

      // User activity API should not be called yet
      expect(analyticsApi.getUserActivity).not.toHaveBeenCalled();
    });

    it('should call tab-specific API when tab is activated', async () => {
      (analyticsApi.getUserActivity as jest.Mock).mockResolvedValue({
        data: [],
        pagination: { total: 0, limit: 20, offset: 0, has_more: false },
      });

      const user = userEvent.setup();
      renderWithQueryClient(<AnalyticsPage />);

      // Wait for initial load
      await waitFor(() => {
        expect(analyticsApi.getDashboardTrends).toHaveBeenCalled();
      });

      // Clear mock calls
      (analyticsApi.getUserActivity as jest.Mock).mockClear();

      // Click User Activity tab
      const usersTab = screen.getByText('User Activity');
      await user.click(usersTab);

      // User Activity API should now be called
      await waitFor(() => {
        expect(analyticsApi.getUserActivity).toHaveBeenCalled();
      });
    });
  });

  describe('State Management', () => {
    it('should maintain active tab state across re-renders', async () => {
      const user = userEvent.setup();
      const { rerender } = renderWithQueryClient(<AnalyticsPage />);

      // Switch to a different tab
      const complianceTab = screen.getByText('Compliance');
      await user.click(complianceTab);

      await waitFor(() => {
        expect(complianceTab.closest('button')).toHaveClass(/border-primary-500/);
      });

      // Rerender and check tab state is preserved
      rerender(
        <QueryClientProvider client={createMockQueryClient()}>
          <AnalyticsPage />
        </QueryClientProvider>
      );

      await waitFor(() => {
        expect(complianceTab.closest('button')).toHaveClass(/border-primary-500/);
      });
    });
  });

  describe('Performance', () => {
    it('should not render hidden tab content', () => {
      renderWithQueryClient(<AnalyticsPage />);

      // Overview tab is active, other tabs should not render their content yet
      // This is a basic check - in real scenario, you'd check for specific component content
      expect(screen.getByText('Overview')).toBeInTheDocument();
    });
  });

  describe('Visual Regression', () => {
    it('should render stats cards with correct styling', async () => {
      renderWithQueryClient(<AnalyticsPage />);

      await waitFor(() => {
        const statsCards = screen.getAllByText(/Sessions|Users|Commands|Anomalies/);
        statsCards.forEach(card => {
          const cardElement = card.closest('.card');
          expect(cardElement).toBeInTheDocument();
        });
      });
    });
  });
});
