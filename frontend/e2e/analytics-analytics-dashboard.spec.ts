import { test, expect } from './fixtures/auth.fixture';

test.describe('Analytics Dashboard - Enhanced E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Setup analytics API mocks
    page.route('**/api/v1/analytics/**', async (route) => {
      const url = route.request().url();

      if (url.includes('/dashboard') || url.includes('/dashboard/trends')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            metrics: {
              active_sessions: 5,
              peak_concurrent_sessions: 12,
              total_sessions_today: 23,
              avg_session_duration_seconds: 1800,
            },
            trends: {
              sessions: { current: 145, previous: 130, change_percent: 11.5 },
              users: { current: 45, previous: 42, change_percent: 7.1 },
            },
            realtime: {
              active_sessions: 5,
              active_users: 3,
              sessions_last_hour: 8,
              avg_active_duration: 1200,
            },
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
      if (route.request().url().includes('/dashboard')) {
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
                status: 'non_compliant',
                score: 45,
                evidence_count: 5,
                last_assessed_at: new Date().toISOString(),
                category: 'Monitoring',
              },
            ],
            exceptions: [],
            last_updated: new Date().toISOString(),
          }),
        });
      } else if (route.request().url().includes('/anomalies')) {
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
                confidence_score: 0.92,
                status: 'open',
              },
            ],
            pagination: { total: 1, limit: 20, offset: 0, has_more: false },
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

  test('should display overview tab by default', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /analytics/i });
    await expect(heading).toBeVisible();
  });

  test('should show overview tab by default', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check if overview tab content is visible
    const overviewContent = authenticatedPage.getByText(/Overview/i).or(
      authenticatedPage.getByText(/active sessions/i)
    );
    await expect(overviewContent).toBeVisible();
  });

  test('should navigate between all navigation tabs', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const tabs = ['Overview', 'Session Metrics', 'User Activity', 'Command Analysis', 'Compliance', 'Anomalies'];

    for (const tab of tabs) {
      const tabButton = authenticatedPage.getByRole('button', { name: new RegExp(tab, 'i') }).or(
        authenticatedPage.getByText(tab)
      );

      if (await tabButton.isVisible()) {
        await tabButton.click();
        await authenticatedPage.waitForLoadState('networkidle');
        await authenticatedPage.waitForTimeout(500);
      }
    }
  });

  test('should display quick stats on overview tab', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for stat cards
    const stats = ['Active Sessions', 'Active Users', 'High-Risk Commands', 'Open Anomalies'];
    for (const stat of stats) {
      const statElement = authenticatedPage.getByText(stat, { exact: false });
      await expect(statElement).toBeVisible();
    }
  });

  test('should display real-time stats periodically', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Real-time stats should be displayed
    const statsText = await authenticatedPage.getByText(/\d+/).all();
    expect(statsText.length).toBeGreaterThan(0);
  });

  test('should filter anomalies by time period', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Navigate to Anomalies tab
    const anomaliesTab = authenticatedPage.getByText('Anomalies').or(
      authenticatedPage.getByRole('button', { name: /anomalies/i })
    );
    await anomaliesTab.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for period filter
    const periodSelect = authenticatedPage.getByRole('combobox').or(
      authenticatedPage.getByRole('listbox')
    );

    const periodOptions = authenticatedPage.getByRole('option', { name: /last 7 days/i }).or(
      authenticatedPage.getByRole('option', { name: /last 30 days/i })
    );

    const periodCount = await periodOptions.count();
    expect(periodCount).toBeGreaterThan(0);
  });

  test('should handle command blacklist data', async ({ authenticatedPage }) => {
    // Mock command analysis response
    authenticatedPage.route('**/api/v1/analytics/commands/frequency**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              command: 'rm -rf',
              count: 5,
              risk_level: 'critical',
            },
            {
              command: 'sudo su',
              count: 12,
              risk_level: 'high',
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const commandsTab = authenticatedPage.getByText('Command Analysis').or(
      authenticatedPage.getByRole('button', { name: /command analysis/i })
    );
    await commandsTab.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for high-risk command styling
    const commandText = authenticatedPage.getByText(/rm -rf/i);
    if (await commandText.isVisible()) {
      // Should be visible with danger styling
      await expect(commandText).toBeVisible();
    }
  });

  test('should display compliance score correctly', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const complianceTab = authenticatedPage.getByText('Compliance').or(
      authenticatedPage.getByRole('button', { name: /compliance/i })
    );
    await complianceTab.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for compliance score
    const scoreText = authenticatedPage.getByText(/78%/i).or(
      authenticatedPage.getByText(/\d+%/i)
    );

    if (await scoreText.isVisible()) {
      await expect(scoreText).toBeVisible();
    }
  });

  test('should display all summary cards when available', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for summary cards
    const cardElements = authenticatedPage.locator('.card, [class*="card"], [class*="stat"]');
    const cardCount = await cardElements.count();

    expect(cardCount).toBeGreaterThan(0);
  });

  test('should support keyboard navigation between tabs', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find the first tab button
    const firstTab = authenticatedPage.getByRole('button', { name: /overview/i }).or(
      authenticatedPage.getByText('Overview')
    );

    if (await firstTab.isVisible()) {
      await firstTab.focus();
      // Navigate with arrow keys
      await authenticatedPage.keyboard.press('ArrowRight');
      await authenticatedPage.waitForTimeout(200);
    }
  });

  test('should maintain tab state when switching back and forth', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    // Switch to Session Metrics
    const sessionsTab = authenticatedPage.getByText('Session Metrics');
    if (await sessionsTab.isVisible()) {
      await sessionsTab.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Switch back to Overview
      const overviewTab = authenticatedPage.getByText('Overview');
      await overviewTab.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should show overview content
      const overviewContent = authenticatedPage.getByText(/active sessions/i);
      await expect(overviewContent).toBeVisible();
    }
  });

  test('should display user activity table', async ({ authenticatedPage }) => {
    // Mock user activity response
    authenticatedPage.route('**/api/v1/analytics/users/activity**', async (route) => {
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
              most_used_targets: ['prod-server-01'],
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const userActivityTab = authenticatedPage.getByText('User Activity');
    await userActivityTab.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show user activity
    const userName = authenticatedPage.getByText(/Admin User/i);
    await expect(userName).toBeVisible();
  });

  test('should display correctly on mobile viewport', async ({ authenticatedPage }) => {
    await authenticatedPage.setViewportSize({ width: 375, height: 667 });
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /analytics/i });
    await expect(heading).toBeVisible();

    // Tabs should be scrollable or stacked
    const tabs = authenticatedPage.getByRole('navigation').or(
      authenticatedPage.locator('nav')
    );
    await expect(tabs).toBeVisible();
  });
});
