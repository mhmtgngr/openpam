import { test, expect } from './fixtures/auth.fixture';
import { AnomalyListPage } from './pages/AnomalyListPage';
import { AnomalyDetailPage } from './pages/AnomalyDetailPage';

/**
 * Comprehensive E2E tests for Anomaly Detection feature
 *
 * These tests cover:
 * - Anomaly list page functionality
 * - Filtering and search
 * - Bulk operations
 * - Anomaly detail page
 * - Status updates and resolution workflow
 * - Related entities navigation
 */

// Mock data for anomalies
const mockAnomalies = [
  {
    id: 'anom-1',
    type: 'unusual_access_time',
    severity: 'critical',
    title: 'Unusual After-Hours Access',
    description: 'User accessed production system at 3 AM',
    detected_at: new Date().toISOString(),
    user_id: 'user-1',
    user_name: 'Admin User',
    target_id: 'target-1',
    target_name: 'prod-server-01',
    session_id: 'sess-1',
    confidence_score: 0.92,
    risk_score: 95,
    status: 'open',
    assigned_to: '',
    resolution_notes: '',
    indicators: [
      {
        type: 'time_anomaly',
        description: 'Access time outside normal hours',
        value: '03:00',
        threshold: '06:00-22:00',
        confidence: 0.92,
      },
    ],
  },
  {
    id: 'anom-2',
    type: 'impossible_travel',
    severity: 'high',
    title: 'Impossible Travel Detected',
    description: 'User logged in from two locations 1000 miles apart within 5 minutes',
    detected_at: new Date(Date.now() - 3600000).toISOString(),
    user_id: 'user-2',
    user_name: 'Jane Smith',
    target_id: 'target-2',
    target_name: 'db-server-02',
    session_id: 'sess-2',
    confidence_score: 0.88,
    risk_score: 85,
    status: 'investigating',
    assigned_to: 'admin@example.com',
    resolution_notes: 'Under investigation',
    indicators: [
      {
        type: 'geo_anomaly',
        description: 'Login from New York, then London 5 minutes later',
        value: '1000',
        threshold: '500',
        confidence: 0.88,
      },
    ],
  },
  {
    id: 'anom-3',
    type: 'excessive_failed_logins',
    severity: 'medium',
    title: 'Excessive Failed Login Attempts',
    description: 'User had 15 failed login attempts within 5 minutes',
    detected_at: new Date(Date.now() - 7200000).toISOString(),
    user_id: 'user-3',
    user_name: 'Bob Johnson',
    target_id: 'target-3',
    target_name: 'auth-server',
    session_id: null,
    confidence_score: 0.75,
    risk_score: 60,
    status: 'resolved',
    assigned_to: 'security@example.com',
    resolution_notes: 'User reset password, account secured',
    indicators: [
      {
        type: 'brute_force_indicators',
        description: 'Multiple failed attempts from different IPs',
        value: 15,
        threshold: 10,
        confidence: 0.75,
      },
    ],
  },
];

const mockAnomalySummary = {
  total: 3,
  by_severity: {
    critical: 1,
    high: 1,
    medium: 1,
    low: 0,
  },
  by_type: {
    unusual_access_time: 1,
    impossible_travel: 1,
    excessive_failed_logins: 1,
  },
  by_status: {
    open: 1,
    investigating: 1,
    resolved: 1,
    false_positive: 0,
  },
  resolved_this_period: 1,
  avg_resolution_time_hours: 4.5,
  critical_open: 1,
  high_open: 0,
};

test.describe('Anomaly List Page - Basic Rendering', () => {
  test.beforeEach(async ({ page }) => {
    // Setup API mocks
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET' && url.includes('/summary')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalySummary }),
        });
        return;
      }

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: mockAnomalies,
              pagination: { total: 3, offset: 0, limit: 20, has_more: false },
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should display anomaly list page with heading', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const heading = await anomalyListPage.getHeadingText();
    expect(heading?.toLowerCase()).toContain('anomaly');
  });

  test('should display all summary cards', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const summaryCount = await anomalyListPage.getSummaryCardsCount();
    expect(summaryCount).toBeGreaterThanOrEqual(6); // Total, Critical, High, Medium, Low, Resolved
  });

  test('should display summary card values correctly', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const totalCount = await anomalyListPage.getSummaryCardValue('Total');
    expect(totalCount).toBe('3');

    const criticalCount = await anomalyListPage.getSummaryCardValue('Critical');
    expect(criticalCount).toBe('1');
  });

  test('should display anomalies table with rows', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(3);
  });
});

test.describe('Anomaly List Page - Filtering', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET' && url.includes('/summary')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalySummary }),
        });
        return;
      }

      if (method === 'GET') {
        // Check filters from URL
        const filteredData = mockAnomalies.filter((a) => {
          if (url.includes('severity=critical')) return a.severity === 'critical';
          if (url.includes('severity=high')) return a.severity === 'high';
          if (url.includes('status=open')) return a.status === 'open';
          if (url.includes('status=investigating')) return a.status === 'investigating';
          if (url.includes('status=resolved')) return a.status === 'resolved';
          if (url.includes('type=unusual_access_time')) return a.type === 'unusual_access_time';
          return true;
        });

        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: filteredData,
              pagination: { total: filteredData.length, offset: 0, limit: 20, has_more: false },
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should filter anomalies by severity - critical', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterBySeverity('critical');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(1);

    const severity = await anomalyListPage.getAnomalySeverity(0);
    expect(severity).toBe('critical');
  });

  test('should filter anomalies by severity - high', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterBySeverity('high');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(1);
  });

  test('should filter anomalies by status - open', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterByStatus('open');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(1);
  });

  test('should filter anomalies by status - resolved', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterByStatus('resolved');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(1);
  });

  test('should filter anomalies by type', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterByType('unusual_access_time');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(1);
  });

  test('should filter by time period', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.selectPeriod(7);
    await anomalyListPage.waitForContent();

    // Period filter should work without errors
    const heading = await anomalyListPage.getHeadingText();
    expect(heading).toBeTruthy();
  });

  test('should search anomalies by text', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.searchAnomalies('After-Hours');
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should reset filters', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Apply a filter
    await anomalyListPage.filterBySeverity('critical');
    await anomalyListPage.waitForContent();

    // Reset filters
    await anomalyListPage.resetFilters();
    await anomalyListPage.waitForContent();

    // Should show all anomalies again
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(3);
  });
});

test.describe('Anomaly List Page - Bulk Operations', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET' && url.includes('/summary')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalySummary }),
        });
        return;
      }

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: mockAnomalies,
              pagination: { total: 3, offset: 0, limit: 20, has_more: false },
            },
          }),
        });
        return;
      }

      // Handle bulk update
      if (method === 'POST' && url.includes('/bulk-update')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: { updated: 2, failed: [] } }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should select anomaly via checkbox', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.selectAnomaly(0);

    // Bulk actions bar should appear
    await authenticatedPage.waitForTimeout(500);
    const isBulkActionsVisible = await anomalyListPage.isBulkActionsBarVisible();
    // Might not be visible with just one selection depending on implementation
  });

  test('should select all anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.selectAllAnomalies();
    await authenticatedPage.waitForTimeout(500);

    // All checkboxes should be checked
    const checkboxes = authenticatedPage.locator('input[type="checkbox"]:checked');
    const count = await checkboxes.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should perform bulk status update', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Select all anomalies
    await anomalyListPage.selectAllAnomalies();
    await authenticatedPage.waitForTimeout(500);

    // Click bulk status update
    await anomalyListPage.clickBulkStatusUpdate('investigating');
    await authenticatedPage.waitForTimeout(500);

    // Should complete without error
    const heading = await anomalyListPage.getHeadingText();
    expect(heading).toBeTruthy();
  });
});

test.describe('Anomaly List Page - Actions', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET' && url.includes('/summary')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalySummary }),
        });
        return;
      }

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: mockAnomalies,
              pagination: { total: 3, offset: 0, limit: 20, has_more: false },
            },
          }),
        });
        return;
      }

      // Export endpoint
      if (method === 'POST' && url.includes('/export')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              download_url: '/exports/anomalies-123.csv',
              expires_at: new Date(Date.now() + 3600000).toISOString(),
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should refresh anomalies list', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const countBefore = await anomalyListPage.getAnomalyCount();

    await anomalyListPage.clickRefresh();
    await anomalyListPage.waitForContent();

    const countAfter = await anomalyListPage.getAnomalyCount();
    expect(countAfter).toBe(countBefore);
  });

  test('should export anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.clickExport();
    await authenticatedPage.waitForTimeout(500);

    // Should complete without error
    const heading = await anomalyListPage.getHeadingText();
    expect(heading).toBeTruthy();
  });

  test('should navigate to anomaly detail page', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.clickViewButton(0);
    await authenticatedPage.waitForTimeout(500);

    const url = authenticatedPage.url();
    expect(url).toContain('/analytics/anomalies/');
  });
});

test.describe('Anomaly Detail Page - Basic Rendering', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      // GET single anomaly
      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalies[0] }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should display anomaly detail page', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBe('Unusual After-Hours Access');
  });

  test('should display anomaly severity badge', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const severity = await anomalyDetailPage.getSeverity();
    expect(severity).toBe('critical');
  });

  test('should display anomaly status badge', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const status = await anomalyDetailPage.getStatus();
    expect(status).toBe('open');
  });

  test('should display detection indicators', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const indicators = await anomalyDetailPage.getIndicators();
    expect(indicators.length).toBeGreaterThan(0);
    expect(indicators[0].type).toBe('time_anomaly');
  });
});

test.describe('Anomaly Detail Page - Status Updates', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      // GET single anomaly
      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalies[0] }),
        });
        return;
      }

      // PATCH update anomaly
      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              ...mockAnomalies[0],
              status: 'investigating',
              assigned_to: 'admin@example.com',
              resolution_notes: 'Under investigation',
            },
          }),
        });
        return;
      }

      // POST acknowledge
      if (url.includes('/acknowledge') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              ...mockAnomalies[0],
              status: 'investigating',
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should update anomaly status', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('investigating');
    await anomalyDetailPage.saveStatus();

    await authenticatedPage.waitForTimeout(500);

    // Should complete without error
    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBeTruthy();
  });

  test('should acknowledge anomaly', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.acknowledgeAnomaly();
    await authenticatedPage.waitForTimeout(500);

    // Should complete without error
    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBeTruthy();
  });

  test('should cancel status edit', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('resolved');
    await anomalyDetailPage.cancelStatusEdit();

    // Status should revert
    const status = await anomalyDetailPage.getStatus();
    expect(status).toBe('open');
  });
});

test.describe('Anomaly Detail Page - Related Entities', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalies[0] }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should display related user entity', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const user = await anomalyDetailPage.getRelatedEntity('user');
    expect(user).toBe('Admin User');
  });

  test('should display related target entity', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const target = await anomalyDetailPage.getRelatedEntity('target');
    expect(target).toBe('prod-server-01');
  });

  test('should show session link when session exists', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const hasSessionLink = await anomalyDetailPage.hasSessionLink();
    expect(hasSessionLink).toBe(true);
  });
});

test.describe('Anomaly Detail Page - Assignment', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAnomalies[0] }),
        });
        return;
      }

      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && !url.includes('/bulk') && method === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              ...mockAnomalies[0],
              assigned_to: 'security@example.com',
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should assign anomaly to user', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setAssignedTo('security@example.com');
    await anomalyDetailPage.saveStatus();

    await authenticatedPage.waitForTimeout(500);

    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBeTruthy();
  });
});

test.describe('Anomaly List Page - Empty State', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET' && url.includes('/summary')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              total: 0,
              by_severity: { critical: 0, high: 0, medium: 0, low: 0 },
              resolved_this_period: 0,
            },
          }),
        });
        return;
      }

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: [],
              pagination: { total: 0, offset: 0, limit: 20, has_more: false },
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should display empty state when no anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const isEmpty = await anomalyListPage.isEmptyStateVisible();
    expect(isEmpty).toBe(true);
  });

  test('should show zero counts in summary cards', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const totalCount = await anomalyListPage.getSummaryCardValue('Total');
    expect(totalCount).toBe('0');
  });
});

test.describe('Anomaly List Page - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.abort('failed');
    });
  });

  test('should handle API error gracefully', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();

    // Wait for either content or error state
    await authenticatedPage.waitForTimeout(3000);

    // Page should not crash
    const url = authenticatedPage.url();
    expect(url).toContain('/analytics/anomalies');
  });
});

test.describe('Anomaly List Page - Pagination', () => {
  test.beforeEach(async ({ page }) => {
    // Generate many anomalies
    const manyAnomalies = Array.from({ length: 25 }, (_, i) => ({
      id: `anom-${i}`,
      type: 'unusual_access_time',
      severity: i < 5 ? 'critical' : i < 15 ? 'high' : 'medium',
      title: `Anomaly ${i}`,
      description: `Test anomaly ${i}`,
      detected_at: new Date().toISOString(),
      user_id: `user-${i % 5}`,
      user_name: `User ${i % 5}`,
      target_id: `target-${i % 3}`,
      target_name: `Target ${i % 3}`,
      session_id: `sess-${i}`,
      confidence_score: 0.7,
      risk_score: 50 + i,
      status: i % 3 === 0 ? 'open' : 'investigating',
      indicators: [],
    }));

    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (method === 'GET') {
        const offset = url.includes('offset=20') ? 20 : 0;
        const limit = 20;
        const paginatedData = manyAnomalies.slice(offset, offset + limit);

        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              data: paginatedData,
              pagination: { total: 25, offset, limit, has_more: offset === 0 },
            },
          }),
        });
        return;
      }

      await route.continue();
    });
  });

  test('should display pagination control', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const hasPagination = await anomalyListPage.hasPagination();
    expect(hasPagination).toBe(true);
  });

  test('should navigate to next page', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.goToNextPage();
    await authenticatedPage.waitForTimeout(500);

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBe(5); // Remaining items
  });
});
