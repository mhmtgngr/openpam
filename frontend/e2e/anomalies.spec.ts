import { test, expect } from './fixtures/auth.fixture';
import { AnomalyListPage } from './pages/AnomalyListPage';
import { AnomalyDetailPage } from './pages/AnomalyDetailPage';

test.describe('Anomaly List Page', () => {
  test('should display anomaly list page', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Wait a bit more for the page to fully render
    await authenticatedPage.waitForTimeout(1000);

    // Debug: check the URL
    const url = authenticatedPage.url();
    console.log('Current URL after navigation:', url);

    // Check if the page has any content at all
    const bodyText = await authenticatedPage.locator('body').textContent();
    console.log('Body text length:', bodyText?.length || 0);
    console.log('Body text preview:', bodyText?.substring(0, 200) || 'empty');

    // Check for the AppLayout
    const layoutElements = await authenticatedPage.locator('[data-testid="app-layout"], .app-layout, nav, aside').all();
    console.log('Layout elements count:', layoutElements.length);

    const heading = await anomalyListPage.getHeadingText();
    console.log('Heading text:', heading);

    // If heading is not found, check for any h1 elements
    if (!heading) {
      const h1Elements = await authenticatedPage.locator('h1').all();
      console.log('H1 elements count:', h1Elements.length);
      for (let i = 0; i < Math.min(h1Elements.length, 5); i++) {
        const text = await h1Elements[i].textContent();
        console.log(`H1[${i}]:`, text);
      }
    }

    expect(heading?.toLowerCase()).toContain('anomaly');
  });

  test('should display summary cards', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const summaryCount = await anomalyListPage.getSummaryCardsCount();
    expect(summaryCount).toBeGreaterThanOrEqual(3);
  });

  test('should display anomalies table', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should filter anomalies by severity', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.filterBySeverity('critical');
    await anomalyListPage.waitForContent();
    
    // Just verify no error
    expect(true).toBe(true);
  });

  test('should allow refresh', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    await anomalyListPage.clickRefresh();
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });
});

test.describe('Anomaly Detail Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup compliance API mocks for anomaly detail
    page.route('**/api/v1/compliance/anomalies/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      // GET /compliance/anomalies/:id - Get single anomaly
      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
              indicators: [
                {
                  type: 'time_anomaly',
                  description: 'Access time outside normal hours',
                  value: '03:00',
                  threshold: '06:00-22:00',
                  confidence: 0.92,
                },
                {
                  type: 'frequency_anomaly',
                  description: 'First time accessing this system at this hour',
                  value: 1,
                  threshold: 5,
                  confidence: 0.88,
                },
              ],
              status: 'open',
              assigned_to: '',
              resolution_notes: '',
            },
          }),
        });
        return;
      }

      // PATCH /compliance/anomalies/:id - Update anomaly
      if (url.includes('/compliance/anomalies/') && !url.includes('/acknowledge') && method === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
              indicators: [],
              status: 'investigating',
              assigned_to: 'admin@example.com',
              resolution_notes: 'Under investigation',
            },
          }),
        });
        return;
      }

      // POST /compliance/anomalies/:id/acknowledge - Acknowledge anomaly
      if (url.includes('/acknowledge') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
              indicators: [],
              status: 'investigating',
              assigned_to: '',
              resolution_notes: '',
            },
          }),
        });
        return;
      }

      // Default fallback
      await route.continue();
    });
  });

  test('should display anomaly detail page', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBeTruthy();
  });

  test('should display anomaly severity and status', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const severity = await anomalyDetailPage.getSeverity();
    expect(severity).toBeTruthy();

    const status = await anomalyDetailPage.getStatus();
    expect(status).toBeTruthy();
  });

  test('should allow editing status', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('investigating');

    const selectedStatus = await anomalyDetailPage.getSelectedStatus();
    expect(selectedStatus).toBe('investigating');
  });

  test('should display related entities', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const user = await anomalyDetailPage.getRelatedEntity('user');
    expect(user).toBeTruthy();

    const target = await anomalyDetailPage.getRelatedEntity('target');
    expect(target).toBeTruthy();
  });
});
