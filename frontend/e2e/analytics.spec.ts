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
});
