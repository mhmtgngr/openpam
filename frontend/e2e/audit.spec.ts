import { test, expect } from './fixtures/auth.fixture';
import { AuditPage } from './pages/AuditPage';

test.describe('Audit Logs', () => {
  test('should display audit page', async ({ authenticatedPage }) => {
    const auditPage = new AuditPage(authenticatedPage);
    await auditPage.goto();
    await auditPage.waitForLoad();

    const heading = await auditPage.getHeadingText();
    expect(heading).toContain('Audit Logs');
  });

  test('should show empty state when no audit events', async ({ authenticatedPage }) => {
    // Mock the API to return empty list
    await authenticatedPage.route('**/api/v1/audit*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(authenticatedPage);
    await auditPage.goto();
    await auditPage.waitForLoad();

    const isEmpty = await auditPage.hasEmptyMessage();
    expect(isEmpty).toBe(true);
  });

  test('should display audit events table', async ({ mockApiPage }) => {
    // Mock audit events
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              created_at: new Date().toISOString(),
              actor: {
                email: 'admin@example.com',
              },
              actor_ip: '192.168.1.100',
              action: 'user_login',
              resource_type: 'users',
              resource_id: 'user-1',
              resource_name: 'Admin User',
              outcome: 'success',
              details: {},
            },
            {
              id: '2',
              created_at: new Date(Date.now() - 3600000).toISOString(),
              actor: {
                email: 'user@example.com',
              },
              actor_ip: '192.168.1.50',
              action: 'credential_checkout',
              resource_type: 'credentials',
              resource_id: 'cred-1',
              resource_name: 'Production DB',
              outcome: 'success',
              details: {},
            },
            {
              id: '3',
              created_at: new Date(Date.now() - 7200000).toISOString(),
              actor: {
                email: 'attacker@example.com',
              },
              actor_ip: '10.0.0.1',
              action: 'credential_checkout',
              resource_type: 'credentials',
              outcome: 'denied',
              details: {},
            },
          ],
          pagination: { total: 3, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show audit events
    await expect(mockApiPage.getByText('user_login')).toBeVisible();
    await expect(mockApiPage.getByText('credential_checkout')).toBeVisible();
    await expect(mockApiPage.getByText('admin@example.com')).toBeVisible();
    await expect(mockApiPage.getByText('user@example.com')).toBeVisible();

    // Check for outcome badges
    await expect(mockApiPage.getByText('success')).toBeVisible();
    await expect(mockApiPage.getByText('denied')).toBeVisible();
  });

  test('should filter audit logs by outcome', async ({ mockApiPage }) => {
    let filterRequestCount = 0;
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      const url = route.request().url();
      if (url.includes('outcome=denied')) {
        filterRequestCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              created_at: new Date().toISOString(),
              actor: { email: 'user@example.com' },
              actor_ip: '192.168.1.50',
              action: 'credential_checkout',
              resource_type: 'credentials',
              outcome: 'denied',
              details: {},
            },
          ],
          pagination: { total: 1, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Select outcome filter
    await auditPage.outcomeFilter.selectOption({ label: 'Denied' });
    await mockApiPage.waitForTimeout(500);

    expect(filterRequestCount).toBeGreaterThan(0);
  });

  test('should filter audit logs by resource type', async ({ mockApiPage }) => {
    let filterRequestCount = 0;
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      const url = route.request().url();
      if (url.includes('resource_type=users')) {
        filterRequestCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              created_at: new Date().toISOString(),
              actor: { email: 'admin@example.com' },
              action: 'user_created',
              resource_type: 'users',
              resource_id: 'user-1',
              outcome: 'success',
              details: {},
            },
          ],
          pagination: { total: 1, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Select resource type filter
    await auditPage.resourceFilter.selectOption({ label: 'Users' });
    await mockApiPage.waitForTimeout(500);

    expect(filterRequestCount).toBeGreaterThan(0);
  });

  test('should search audit logs', async ({ mockApiPage }) => {
    let searchRequestCount = 0;
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      const url = route.request().url();
      if (url.includes('search=admin')) {
        searchRequestCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              created_at: new Date().toISOString(),
              actor: { email: 'admin@example.com' },
              action: 'user_login',
              resource_type: 'users',
              outcome: 'success',
              details: {},
            },
          ],
          pagination: { total: 1, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Enter search term
    await auditPage.search('admin');
    await mockApiPage.waitForTimeout(500);

    expect(searchRequestCount).toBeGreaterThan(0);
  });

  test('should export audit logs', async ({ mockApiPage }) => {
    let exportCalled = false;
    await mockApiPage.route('**/api/v1/audit/export', async (route) => {
      exportCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'text/csv',
        body: 'timestamp,actor,action,outcome\n2024-01-01,admin@example.com,login,success\n',
      });
    });

    // Mock the audit list endpoint
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      if (!route.request().url().includes('export')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, limit: 50, offset: 0 },
          }),
        });
      }
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click export button
    await auditPage.clickExport();

    // Wait for download to trigger
    await mockApiPage.waitForTimeout(500);

    expect(exportCalled).toBe(true);
  });

  test('should format timestamps correctly', async ({ mockApiPage }) => {
    const testDate = new Date('2024-01-15T14:30:45Z');
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              created_at: testDate.toISOString(),
              actor: { email: 'admin@example.com' },
              actor_ip: '192.168.1.1',
              action: 'user_login',
              resource_type: 'users',
              outcome: 'success',
              details: {},
            },
          ],
          pagination: { total: 1, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Check for formatted timestamp (format: yyyy-MM-dd HH:mm:ss)
    await expect(mockApiPage.getByText(/2024-01-15.*14:30/)).toBeVisible();
  });

  test('should paginate audit logs', async ({ mockApiPage }) => {
    let page2Requested = false;
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      const url = route.request().url();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: Array.from({ length: 50 }, (_, i) => ({
            id: `event-${i}`,
            created_at: new Date().toISOString(),
            actor: { email: `user${i}@example.com` },
            action: 'test_action',
            resource_type: 'test',
            outcome: 'success',
            details: {},
          })),
          pagination: { total: 150, limit: 50, offset: 0 },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show pagination info when there are more results
    // Check for pagination component or "next" button
    const hasPagination = await mockApiPage.getByRole('button', { name: /next|more|>/i }).count() > 0;
    // Or check for page indicator
    const hasPageInfo = await mockApiPage.getByText(/1\s+of\s+\d+/).isVisible().catch(() => false);

    expect(hasPagination || hasPageInfo).toBe(true);
  });

  test('should redirect to login when unauthenticated', async ({ page }) => {
    await page.goto('/audit');

    // Should redirect to login
    await page.waitForURL('/login', { timeout: 5000 });

    await expect(page.getByText('Sign in to your account')).toBeVisible();
  });

  test('should handle API errors gracefully', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/audit*', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 'INTERNAL_ERROR',
            message: 'Failed to fetch audit logs',
          },
        }),
      });
    });

    const auditPage = new AuditPage(mockApiPage);
    await auditPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show error toast or message
    await expect(mockApiPage.locator('.toast, [data-testid="toast"], .toast-error').or(
      mockApiPage.getByText(/error|failed/i)
    )).toBeVisible({ timeout: 8000 });
  });
});
