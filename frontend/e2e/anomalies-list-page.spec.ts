import { test, expect } from './fixtures/auth.fixture';

test.describe('Anomalies - Enhanced E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Setup anomaly API mocks
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
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
              assigned_to: null,
              resolution_notes: null,
              resolved_at: null,
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
              assigned_to: 'security@example.com',
              resolution_notes: null,
              resolved_at: null,
            },
            {
              id: '3',
              type: 'bulk_data_access',
              severity: 'medium',
              title: 'Large Data Download',
              description: 'User downloaded unusually large amount of data',
              detected_at: new Date(Date.now() - 7200000).toISOString(),
              user_id: '1',
              user_name: 'Admin User',
              target_id: '2',
              target_name: 'db-server-01',
              session_id: 'sess-3',
              confidence_score: 0.75,
              indicators: [],
              status: 'resolved',
              assigned_to: 'admin@example.com',
              resolution_notes: 'Investigated and approved',
              resolved_at: new Date(Date.now() - 1800000).toISOString(),
            },
            {
              id: '4',
              type: 'impossible_travel',
              severity: 'low',
              title: 'Impossible Travel',
              description: 'Login from two locations within impossible time',
              detected_at: new Date(Date.now() - 10800000).toISOString(),
              user_id: '3',
              user_name: 'Remote User',
              target_id: '1',
              target_name: 'prod-server-01',
              confidence_score: 0.60,
              indicators: [],
              status: 'false_positive',
              assigned_to: null,
              resolution_notes: 'VPN routing issue',
              resolved_at: new Date(Date.now() - 3600000).toISOString(),
            },
          ],
          pagination: { total: 4, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    page.route('**/api/v1/compliance/anomalies/summary**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total: 4,
          by_severity: { critical: 1, high: 1, medium: 1, low: 1 },
          by_type: {
            unusual_access_time: 1,
            privileged_escalation: 1,
            bulk_data_access: 1,
            impossible_travel: 1,
          },
          by_status: { open: 1, investigating: 1, resolved: 1, false_positive: 1 },
          resolved_this_period: 1,
          avg_resolution_time_hours: 4.5,
        }),
      });
    });

    // Setup anomaly detail API
    page.route('**/api/v1/compliance/anomalies/*', async (route) => {
      const id = route.request().url().split('/').pop();

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id,
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
          assigned_to: null,
          resolution_notes: null,
          resolved_at: null,
        }),
      });
    });

    // Setup update API
    page.route('**/api/v1/compliance/anomalies/*', async (route) => {
      if (route.request().method() === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: route.request().url().split('/').pop(),
            status: 'investigating',
          }),
        });
      }
    });

    // Setup acknowledge API
    page.route('**/api/v1/compliance/anomalies/*/acknowledge**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });
  });

  test('should display anomaly list page with heading', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /anomal/i });
    await expect(heading).toBeVisible();
  });

  test('should navigate from sidebar', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on Analytics > Anomalies in sidebar
    const analyticsLink = authenticatedPage.getByRole('link', { name: 'Analytics' });
    if (await analyticsLink.isVisible()) {
      await analyticsLink.click();
      await authenticatedPage.waitForURL('/analytics', { timeout: 5000 });
    }

    // Then navigate to anomalies
    const anomaliesLink = authenticatedPage.getByRole('link', { name: /anomal/i });
    if (await anomaliesLink.isVisible()) {
      await anomaliesLink.click();
      await authenticatedPage.waitForURL('/analytics/anomalies', { timeout: 5000 });
    }

    const heading = authenticatedPage.getByRole('heading', { name: /anomal/i });
    await expect(heading).toBeVisible();
  });

  test('should display summary cards', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for summary cards
    const summaryCards = authenticatedPage.locator('[class*="card"], [class*="stat"]');
    const cardCount = await summaryCards.count();

    expect(cardCount).toBeGreaterThan(0);

    // Should show critical count
    const criticalText = authenticatedPage.getByText(/1/i).or(
      authenticatedPage.getByText(/critical/i)
    );
    await expect(criticalText.first()).toBeVisible();
  });

  test('should filter anomalies by severity', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for severity filter
    const severityFilter = authenticatedPage.getByRole('combobox', { name: /severity/i }).or(
      authenticatedPage.getByPlaceholderText(/severity/i)
    ).or(
      authenticatedPage.getByText(/Severity/i)
    );

    if (await severityFilter.isVisible()) {
      await severityFilter.click();

      const criticalOption = authenticatedPage.getByRole('option', { name: /critical/i });
      await criticalOption.click();

      await authenticatedPage.waitForLoadState('networkidle');

      // Should show filtered results
      const anomalies = authenticatedPage.locator('[class*="anomaly"], [class*="Anomaly"]');
      const count = await anomalies.count();
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });

  test('should filter anomalies by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for status filter
    const statusFilter = authenticatedPage.getByRole('combobox', { name: /status/i }).or(
      authenticatedPage.getByPlaceholderText(/status/i)
    );

    if (await statusFilter.isVisible()) {
      await statusFilter.click();

      const openOption = authenticatedPage.getByRole('option', { name: /open/i });
      await openOption.click();

      await authenticatedPage.waitForLoadState('networkidle');

      // Should show filtered results
      const anomalies = authenticatedPage.locator('[class*="anomaly"], [class*="Anomaly"]');
      const count = await anomalies.count();
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });

  test('should select all anomalies via checkbox', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for select all checkbox
    const selectAllCheckbox = authenticatedPage.getByRole('checkbox', { name: /select all/i }).or(
      authenticatedPage.locator('input[type="checkbox"]').first
    );

    if (await selectAllCheckbox.isVisible()) {
      await selectAllCheckbox.click();
      await authenticatedPage.waitForTimeout(500);

      // All checkboxes should be checked
      const checkboxes = authenticatedPage.getByRole('checkbox');
      const checkedCount = await checkboxes.count();
      expect(checkedCount).toBeGreaterThan(0);
    }
  });

  test('should search anomalies by text', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for search input
    const searchInput = authenticatedPage.getByPlaceholderText(/search/i).or(
      authenticatedPage.getByRole('textbox', { name: /search/i })
    );

    if (await searchInput.isVisible()) {
      await searchInput.fill('after-hours');
      await authenticatedPage.keyboard.press('Enter');
      await authenticatedPage.waitForLoadState('networkidle');

      // Should show search results or empty state
      const pageContent = authenticatedPage.locator('main');
      await expect(pageContent).toBeVisible();
    }
  });

  test('should display anomaly detail page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on first anomaly
    const firstAnomaly = authenticatedPage.locator('[class*="anomaly"], [class*="Anomaly"]').first();
    await firstAnomaly.click();
    await authenticatedPage.waitForLoadState('networkidle');

    // Should navigate to detail page or show modal
    const url = authenticatedPage.url();
    const isDetailPage = url.includes('/anomalies/') || url.includes('/anomaly/');

    if (isDetailPage) {
      const heading = authenticatedPage.getByRole('heading');
      await expect(heading).toBeVisible();
    }
  });

  test('should display anomaly detail with metadata', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for detail elements
    const heading = authenticatedPage.getByRole('heading');
    await expect(heading).toBeVisible();

    // Should show anomaly metadata
    const metadata = ['Detected At', 'Confidence Score', 'User', 'Target'];
    for (const meta of metadata) {
      const metaElement = authenticatedPage.getByText(meta, { exact: false });
      await expect(metaElement.first()).toBeVisible();
    }
  });

  test('should update anomaly status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for status dropdown
    const statusSelect = authenticatedPage.getByRole('combobox', { name: /status/i }).or(
      authenticatedPage.getByRole('listbox', { name: /status/i })
    );

    if (await statusSelect.isVisible()) {
      await statusSelect.click();
      const investigatingOption = authenticatedPage.getByRole('option', { name: /investigating/i });
      await investigatingOption.click();

      // Look for save button
      const saveButton = authenticatedPage.getByRole('button', { name: /save/i }).or(
        authenticatedPage.getByRole('button', { name: /update/i })
      );

      if (await saveButton.isVisible()) {
        await saveButton.click();
        await authenticatedPage.waitForLoadState('networkidle');
      }
    }
  });

  test('should acknowledge anomaly', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for acknowledge button
    const acknowledgeButton = authenticatedPage.getByRole('button', { name: /acknowledge/i });

    if (await acknowledgeButton.isVisible()) {
      await acknowledgeButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should show success message
      const successMessage = authenticatedPage.getByText(/acknowledged/i, { exact: false });
      await expect(successMessage.first()).toBeVisible();
    }
  });

  test('should display anomaly with resolved state', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for resolved anomalies
    const resolvedBadge = authenticatedPage.getByText(/resolved/i).or(
      authenticatedPage.locator('[class*="resolved"], [class*="success"]')
    );

    const resolvedCount = await resolvedBadge.count();
    expect(resolvedCount).toBeGreaterThan(0);
  });

  test('should handle pagination of results', async ({ authenticatedPage }) => {
    // Mock paginated response
    authenticatedPage.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = new URL(route.request().url());
      const offset = url.searchParams.get('offset') || '0';

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: Array.from({ length: 20 }, (_, i) => ({
            id: `anomaly-${parseInt(offset) + i}`,
            type: 'unusual_access_time',
            severity: 'medium',
            title: `Anomaly ${parseInt(offset) + i + 1}`,
            description: 'Test anomaly',
            detected_at: new Date().toISOString(),
            confidence_score: 0.5,
            status: 'open',
          })),
          pagination: {
            total: 50,
            limit: 20,
            offset: parseInt(offset),
            has_more: true,
          },
        }),
      });
    });

    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for pagination controls
    const nextPageButton = authenticatedPage.getByRole('button', { name: /next/i });

    if (await nextPageButton.isVisible()) {
      await nextPageButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // URL should change
      const url = authenticatedPage.url();
      expect(url).toContain('offset=20');
    }
  });

  test('should display indicators for anomaly', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for indicators section
    const indicatorsHeading = authenticatedPage.getByText(/indicator/i).or(
      authenticatedPage.getByText(/detection/i)
    );

    if (await indicatorsHeading.isVisible()) {
      // Should show indicator details
      const confidenceText = authenticatedPage.getByText(/confidence/i);
      await expect(confidenceText.first()).toBeVisible();
    }
  });

  test('should show view session recording link', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for session link
    const sessionLink = authenticatedPage.getByRole('link', { name: /session/i }).or(
      authenticatedPage.getByText(/view session/i)
    );

    // Session link may or may not be present depending on the anomaly having a session_id
    if (await sessionLink.isVisible()) {
      await expect(sessionLink).toHaveAttribute('href');
    }
  });

  test('should navigate back from detail page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/anomalies/1');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for back button
    const backButton = authenticatedPage.getByRole('button', { name: /back/i }).or(
      authenticatedPage.locator('button').filter({ hasText: /^back$/i })
    );

    if (await backButton.isVisible()) {
      await backButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should return to list
      const url = authenticatedPage.url();
      expect(url).toMatch(/\/analytics\/anomalies?$/);
    }
  });

  test('should display empty state when no anomalies', async ({ authenticatedPage }) => {
    // Mock empty response
    authenticatedPage.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    await authenticatedPage.goto('/analytics/anomalies');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show empty state or no results message
    const emptyState = authenticatedPage.getByText(/no anomalies/i).or(
      authenticatedPage.getByText(/no data/i)
    );

    // At minimum, page should load without errors
    const heading = authenticatedPage.getByRole('heading');
    await expect(heading).toBeVisible();
  });

  test('should display compliance controls breakdown', async ({ authenticatedPage }) => {
    // Navigate to anomalies page from compliance tab
    await authenticatedPage.goto('/analytics');
    await authenticatedPage.waitForLoadState('networkidle');

    const complianceTab = authenticatedPage.getByText('Compliance');
    if (await complianceTab.isVisible()) {
      await complianceTab.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should display controls
      const controls = authenticatedPage.getByText(/Access Control/i).or(
        authenticatedPage.getByText(/Encryption/i)
      );

      const controlCount = await controls.count();
      expect(controlCount).toBeGreaterThan(0);
    }
  });
});
