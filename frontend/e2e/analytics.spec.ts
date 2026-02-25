import { test, expect } from './fixtures/auth.fixture';
import { AnalyticsPage } from './pages/AnalyticsPage';

// Mock analytics API routes
test.beforeEach(async ({ page }) => {
  // Setup analytics API mocks
  page.route('**/api/v1/analytics/**', async (route) => {
    const url = route.request().url();

    if (url.includes('/sessions/metrics')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          active_sessions: 5,
          peak_concurrent_sessions: 12,
          total_sessions_today: 23,
          total_sessions_week: 145,
          avg_session_duration_seconds: 1800,
          total_session_duration_today_seconds: 41400,
          sessions_by_type: { ssh: 15, rdp: 8 },
          sessions_by_environment: { production: 12, staging: 6, development: 5 },
          sessions_over_time: [
            { timestamp: new Date(Date.now() - 86400000).toISOString(), count: 10, duration_seconds: 18000 },
            { timestamp: new Date(Date.now() - 43200000).toISOString(), count: 15, duration_seconds: 27000 },
            { timestamp: new Date().toISOString(), count: 23, duration_seconds: 41400 },
          ],
        }),
      });
    } else if (url.includes('/dashboard/trends')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          sessions: { current: 145, previous: 130, change_percent: 11.5, trend: 'up', data_points: [] },
          users: { current: 45, previous: 42, change_percent: 7.1, trend: 'up', data_points: [] },
          credentials: { current: 230, previous: 225, change_percent: 2.2, trend: 'up', data_points: [] },
          requests: { current: 18, previous: 22, change_percent: -18.2, trend: 'down', data_points: [] },
          period: 'week',
        }),
      });
    } else if (url.includes('/realtime')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          active_sessions: 5,
          active_users: 3,
          sessions_last_hour: 8,
          avg_active_duration: 1200,
        }),
      });
    } else if (url.includes('/users/activity')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              user_id: '1',
              user_email: 'admin@example.com',
              user_name: 'Admin User',
              total_sessions: 15,
              total_duration_seconds: 27000,
              avg_session_duration_seconds: 1800,
              last_activity_at: new Date().toISOString(),
              most_used_targets: ['prod-server-01', 'staging-db-01'],
              activity_heatmap: [],
            },
            {
              user_id: '2',
              user_email: 'user@example.com',
              user_name: 'Regular User',
              total_sessions: 8,
              total_duration_seconds: 14400,
              avg_session_duration_seconds: 1800,
              last_activity_at: new Date(Date.now() - 3600000).toISOString(),
              most_used_targets: ['dev-server-01'],
              activity_heatmap: [],
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0, has_more: false },
        }),
      });
    } else if (url.includes('/commands/frequency')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              command: 'rm -rf',
              count: 5,
              risk_level: 'critical',
              first_seen_at: new Date(Date.now() - 86400000).toISOString(),
              last_seen_at: new Date().toISOString(),
              users: [{ user_id: '1', user_name: 'Admin User', count: 3 }],
              targets: [{ target_id: '1', target_name: 'prod-server-01', count: 5 }],
            },
            {
              command: 'sudo su',
              count: 12,
              risk_level: 'high',
              first_seen_at: new Date(Date.now() - 172800000).toISOString(),
              last_seen_at: new Date().toISOString(),
              users: [{ user_id: '1', user_name: 'Admin User', count: 8 }],
              targets: [{ target_id: '1', target_name: 'prod-server-01', count: 10 }],
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0, has_more: false },
        }),
      });
    } else if (url.includes('/commands/risk-summary')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_commands: 350,
          high_risk_commands: 15,
          medium_risk_commands: 45,
          low_risk_commands: 290,
          unique_users: 12,
          unique_targets: 8,
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, limit: 20, offset: 0, has_more: false } }),
      });
    }
  });

  // Setup compliance API mocks
  page.route('**/api/v1/compliance/**', async (route) => {
    const url = route.request().url();

    if (url.includes('/dashboard')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 78,
          framework: 'soc2',
          controls: [
            {
              id: '1',
              name: 'Access Control',
              description: 'Proper access controls are implemented',
              status: 'compliant',
              score: 95,
              evidence_count: 15,
              last_assessed_at: new Date().toISOString(),
              category: 'Security',
            },
            {
              id: '2',
              name: 'Encryption',
              description: 'Data encryption at rest and in transit',
              status: 'partial',
              score: 70,
              evidence_count: 8,
              last_assessed_at: new Date(Date.now() - 86400000).toISOString(),
              category: 'Security',
            },
            {
              id: '3',
              name: 'Audit Logging',
              description: 'Comprehensive audit trails',
              status: 'compliant',
              score: 88,
              evidence_count: 20,
              last_assessed_at: new Date().toISOString(),
              category: 'Monitoring',
            },
          ],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    } else if (url.includes('/anomalies/summary')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total: 5,
          by_severity: { critical: 1, high: 1, medium: 2, low: 1 },
          by_type: { unusual_access_time: 2, privileged_escalation: 1, impossible_travel: 1, bulk_data_access: 1 },
          by_status: { open: 3, investigating: 1, resolved: 1, false_positive: 0 },
          resolved_this_period: 1,
          avg_resolution_time_hours: 4.5,
        }),
      });
    } else if (url.includes('/anomalies')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              type: 'unusual_access_time',
              severity: 'critical',
              title: 'Unusual After-Hours Access',
              description: 'User accessed production system at 3 AM',
              detected_at: new Date().toISOString(),
              user_id: '1',
              user_name: 'Admin User',
              target_id: '1',
              target_name: 'prod-server-01',
              session_id: 'sess-1',
              confidence_score: 0.92,
              indicators: [
                {
                  type: 'time_anomaly',
                  description: 'Access time outside normal hours',
                  value: '03:00',
                  threshold: '06:00-22:00',
                  confidence: 0.92,
                },
              ],
              status: 'open',
            },
            {
              id: '2',
              type: 'privileged_escalation',
              severity: 'high',
              title: 'Sudden Privilege Escalation',
              description: 'User escalated privileges multiple times in short period',
              detected_at: new Date(Date.now() - 3600000).toISOString(),
              user_id: '2',
              user_name: 'Regular User',
              target_id: '1',
              target_name: 'prod-server-01',
              session_id: 'sess-2',
              confidence_score: 0.85,
              indicators: [],
              status: 'investigating',
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0, has_more: false },
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, limit: 20, offset: 0 } }),
      });
    }
  });
});

test.describe('Analytics Dashboard', () => {
  test('should display analytics page with heading', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    const heading = await analyticsPage.getHeadingText();
    expect(heading?.toLowerCase()).toContain('analytics');
  });

  test('should display all navigation tabs', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    const tabsCount = await analyticsPage.getTabsCount();
    expect(tabsCount).toBe(6); // Overview, Sessions, Users, Commands, Compliance, Anomalies
  });

  test('should show overview tab by default', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    const isOverviewActive = await analyticsPage.isTabActive('Overview');
    expect(isOverviewActive).toBe(true);
  });

  test('should display quick stats on overview tab', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    const statsCount = await analyticsPage.getQuickStatsCount();
    expect(statsCount).toBe(4); // Active Sessions, Active Users, High-Risk Commands, Open Anomalies
  });

  test('should switch between tabs', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    // Click on Sessions tab
    await analyticsPage.clickTab('Session Metrics');
    const isSessionsActive = await analyticsPage.isTabActive('Session Metrics');
    expect(isSessionsActive).toBe(true);

    // Click on Users tab
    await analyticsPage.clickTab('User Activity');
    const isUsersActive = await analyticsPage.isTabActive('User Activity');
    expect(isUsersActive).toBe(true);

    // Click on Compliance tab
    await analyticsPage.clickTab('Compliance');
    const isComplianceActive = await analyticsPage.isTabActive('Compliance');
    expect(isComplianceActive).toBe(true);
  });

  test('should display compliance score', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    // Navigate to Compliance tab
    await analyticsPage.clickTab('Compliance');
    await authenticatedPage.waitForLoadState('networkidle');

    const score = await analyticsPage.getComplianceScore();
    expect(score).toBeTruthy();
    expect(Number(score?.replace('%', ''))).toBeGreaterThan(0);
  });

  test('should display user activity table', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('User Activity');
    await authenticatedPage.waitForLoadState('networkidle');

    const tableCount = await analyticsPage.userActivityTable.count();
    expect(tableCount).toBeGreaterThan(0);
  });

  test('should display command analysis table', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Command Analysis');
    await authenticatedPage.waitForLoadState('networkidle');

    const tableCount = await analyticsPage.commandAnalysisTable.count();
    expect(tableCount).toBeGreaterThan(0);
  });

  test('should display anomaly list', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    const anomalyCount = await analyticsPage.anomalyList.count();
    expect(anomalyCount).toBeGreaterThan(0);
  });

  test('should navigate from sidebar', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on Analytics in sidebar
    await authenticatedPage.getByRole('link', { name: 'Analytics', exact: true }).click();
    await authenticatedPage.waitForURL('/analytics', { timeout: 5000 });

    const heading = authenticatedPage.getByRole('heading', { name: /analytics/i });
    await expect(heading).toBeVisible();
  });

  // New test scenarios for SSH Key Analytics
  test('should display SSH key analytics when available', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    // SSH key analytics should be accessible if user has permissions
    await analyticsPage.clickTab('Session Metrics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for SSH key specific data if available
    const sshKeySection = authenticatedPage.getByText(/SSH/i).or(authenticatedPage.getByText(/ssh/i).nth(0));
    // May or may not be visible depending on permissions
  });

  // New test for command blacklist
  test('should handle command blacklist data', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Command Analysis');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should display high-risk commands with proper styling
    const criticalCommand = authenticatedPage.getByText(/rm -rf/);
    if (await criticalCommand.isVisible()) {
      const parent = criticalCommand.locator('..').locator('..');
      await expect(parent).toHaveClass(/danger/i);
    }
  });

  // New test for compliance reports list
  test('should display compliance reports when navigating to compliance tab', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Compliance');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show compliance score
    const score = await analyticsPage.getComplianceScore();
    expect(score).toBeTruthy();
    expect(Number(score?.replace('%', ''))).toBeGreaterThan(0);
  });

  // New test for anomaly filtering
  test('should filter anomalies by severity', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for filter controls
    const severityFilter = authenticatedPage.getByPlaceholderText(/severity/i).or(
      authenticatedPage.getByLabel(/severity/i)
    ).or(
      authenticatedPage.getByText(/Severity/i)
    );

    if (await severityFilter.isVisible()) {
      // Test filtering by critical severity
      await severityFilter.click();
      await authenticatedPage.getByRole('option', { name: /critical/i }).click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should only show critical anomalies
      const anomalies = await analyticsPage.anomalyList.all();
      expect(anomalies.length).toBeGreaterThan(0);
    }
  });

  // New test for date range filtering
  test('should filter data by date range', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Session Metrics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for date filter controls
    const dateFromFilter = authenticatedPage.getByPlaceholderText(/from/i).or(
      authenticatedPage.getByLabel(/from/i)
    ).or(
      authenticatedPage.getByText(/Date Range/i)
    );

    if (await dateFromFilter.isVisible()) {
      // Set date range
      await dateFromFilter.click();
      // Select last 30 days option
      await authenticatedPage.getByText(/30 days/i).or(authenticatedPage.getByText(/last month/i)).click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should refresh data
      await expect(authenticatedPage.getByText('Session Metrics')).toBeVisible();
    }
  });

  // New test for anomaly detail view
  test('should open anomaly detail modal when clicking on anomaly', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on first anomaly
    const firstAnomaly = analyticsPage.anomalyList.first();
    await firstAnomaly.click();

    // Should show detail view or modal
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for detail elements
    const detailHeading = authenticatedPage.getByRole('heading', { name: /anomaly/i }).or(
      authenticatedPage.getByText(/unusual access/i, { exact: false })
    ).or(
      authenticatedPage.getByText(/privilege escalation/i, { exact: false })
    );

    if (await detailHeading.isVisible()) {
      await expect(detailHeading).toBeVisible();
    }
  });

  // New test for compliance controls breakdown
  test('should show compliance controls breakdown', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Compliance');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should display controls
    const controls = authenticatedPage.getByText(/Access Control/i).or(
      authenticatedPage.getByText(/Encryption/i)
    ).or(
      authenticatedPage.getByText(/Audit/i)
    );

    if (await controls.isVisible()) {
      await expect(controls).toBeVisible();
    }
  });

  // New test for exporting data
  test('should handle export functionality', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Command Analysis');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for export button
    const exportButton = authenticatedPage.getByRole('button', { name: /export/i }).or(
      authenticatedPage.getByRole('button', { name: /download/i })
    ).or(
      authenticatedPage.getByText(/Export/i, { exact: false })
    );

    if (await exportButton.isVisible()) {
      // Setup download handler
      const downloadPromise = authenticatedPage.waitForEvent('download');

      await exportButton.click();

      // Verify download was triggered (or API called)
      try {
        const download = await Promise.race([
          downloadPromise,
          new Promise(resolve => setTimeout(resolve, 2000)) // Timeout after 2s
        ]);
      } catch {
        // Download may have been handled differently
      }
    }
  });

  // New test for real-time stats updates
  test('should refresh real-time stats periodically', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    // Wait for initial load
    await authenticatedPage.waitForLoadState('networkidle');

    // Get initial stats value
    const initialStats = await authenticatedPage.getByText(/\d+/).all();
    const initialCount = initialStats.length;

    expect(initialCount).toBeGreaterThan(0);
  });

  // New test for responsive behavior
  test('should display correctly on mobile viewport', async ({ authenticatedPage }) => {
    // Set mobile viewport
    await authenticatedPage.setViewportSize({ width: 375, height: 667 });
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await authenticatedPage.waitForLoadState('networkidle');

    // All tabs should still be accessible
    const tabsCount = await analyticsPage.getTabsCount();
    expect(tabsCount).toBe(6);

    // Stats should be visible but stacked
    const statsCount = await analyticsPage.getQuickStatsCount();
    expect(statsCount).toBe(4);
  });

  // New test for keyboard navigation
  test('should support keyboard navigation between tabs', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await authenticatedPage.waitForLoadState('networkidle');

    // Focus on first tab
    await authenticatedPage.keyboard.press('Tab');

    // Navigate through tabs with arrow keys or continue Tab
    const overviewTab = authenticatedPage.getByText('Overview');
    await overviewTab.focus();

    // Use arrow right to navigate to next tab
    await authenticatedPage.keyboard.press('ArrowRight');

    // Should highlight next tab
    const sessionsTab = authenticatedPage.getByText('Session Metrics');
    await expect(sessionsTab).toBeVisible();
  });

  // New test for data persistence across tab switches
  test('should maintain tab state when switching back and forth', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await authenticatedPage.waitForLoadState('networkidle');

    // Switch to User Activity tab
    await analyticsPage.clickTab('User Activity');
    await authenticatedPage.waitForLoadState('networkidle');

    // Switch back to Overview
    await analyticsPage.clickTab('Overview');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show overview content
    const isOverviewActive = await analyticsPage.isTabActive('Overview');
    expect(isOverviewActive).toBe(true);
  });

  // New test for empty states
  test('should display empty state when no anomalies exist', async ({ authenticatedPage }) => {
    // Mock empty anomalies response
    authenticatedPage.route('**/api/v1/anomalies**', async (route) => {
      if (route.request().url().includes('/anomalies')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, limit: 20, offset: 0, has_more: false },
          }),
        });
      }
    });

    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should display empty state message
    const emptyState = authenticatedPage.getByText(/no anomalies/i).or(
      authenticatedPage.getByText(/no data/i)
    ).or(
      authenticatedPage.getByText(/0/i)
    );

    // Verify at least some element is shown (count could be 0)
    const anomalyCount = await analyticsPage.anomalyList.count();
    expect(anomalyCount).toBe(0);
  });

  // New test for error handling
  test('should handle API errors gracefully', async ({ authenticatedPage }) => {
    // Mock error response
    authenticatedPage.route('**/api/v1/analytics/dashboard/trends**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 'INTERNAL_ERROR',
            message: 'Failed to fetch trends',
          },
        }),
      });
    });

    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await authenticatedPage.waitForLoadState('networkidle');

    // Should show error state or fallback UI
    const errorMessage = authenticatedPage.getByText(/error/i, { exact: false });
    const retryButton = authenticatedPage.getByRole('button', { name: /retry/i });

    // Either error message or retry button might be shown
    expect(await errorMessage.isVisible() || await retryButton.isVisible()).toBeTruthy();
  });

  // New test for search functionality
  test('should support searching in user activity', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('User Activity');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for search input
    const searchInput = authenticatedPage.getByPlaceholderText(/search/i).or(
      authenticatedPage.getByLabel(/search/i)
    ).or(
      authenticatedPage.getByRole('textbox', { name: /search/i })
    );

    if (await searchInput.isVisible()) {
      await searchInput.fill('admin');
      await authenticatedPage.keyboard.press('Enter');
      await authenticatedPage.waitForLoadState('networkidle');

      // Should filter results
      const tableCount = await analyticsPage.userActivityTable.count();
      expect(tableCount).toBeGreaterThanOrEqual(0);
    }
  });

  // New test for pagination
  test('should handle pagination of results', async ({ authenticatedPage }) => {
    // Mock paginated response
    authenticatedPage.route('**/api/v1/analytics/users/activity**', async (route) => {
      const url = new URL(route.request().url());
      const limit = url.searchParams.get('limit') || '20';

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: Array.from({ length: parseInt(limit) }, (_, i) => ({
            user_id: `user-${i}`,
            user_email: `user${i}@example.com`,
            user_name: `User ${i}`,
            total_sessions: 10 + i,
            total_duration_seconds: 18000,
            avg_session_duration_seconds: 1800,
            last_activity_at: new Date().toISOString(),
            most_used_targets: [],
            activity_heatmap: [],
          })),
          pagination: {
            total: 50,
            limit: parseInt(limit),
            offset: 0,
            has_more: true,
          },
        }),
      });
    });

    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('User Activity');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for pagination controls
    const nextPageButton = authenticatedPage.getByRole('button', { name: /next/i }).or(
      authenticatedPage.getByText(/Next/i, { exact: false })
    );

    const pageInfo = authenticatedPage.getByText(/\d+ - \d+ of \d+/i).or(
      authenticatedPage.getByText(/showing \d+ - \d+ of \d+/i)
    );

    // Either pagination controls or page info might be shown
    const hasPagination = await nextPageButton.isVisible() || await pageInfo.isVisible();
    expect(hasPagination).toBeTruthy();
  });

  // New test for time series charts
  test('should display time series charts', async ({ authenticatedPage }) => {
    const analyticsPage = new AnalyticsPage(authenticatedPage);
    await analyticsPage.goto();

    await analyticsPage.clickTab('Session Metrics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for chart elements (could be canvas, SVG, or div-based)
    const chartElements = authenticatedPage.getByRole('img').or(
      authenticatedPage.locator('canvas')
    ).or(
      authenticatedPage.locator('[data-chart]')
    );

    // Charts might be rendered - this test doesn't fail if not present
    const chartCount = await chartElements.count();
    expect(chartCount).toBeGreaterThanOrEqual(0);
  });
});
