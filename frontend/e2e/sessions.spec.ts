import { test, expect } from './fixtures/auth.fixture';
import { SessionsPage } from './pages/SessionsPage';

test.describe('Sessions', () => {
  test('should display sessions page', async ({ authenticatedPage }) => {
    const sessionsPage = new SessionsPage(authenticatedPage);
    await sessionsPage.goto();
    await sessionsPage.waitForLoad();

    const heading = await sessionsPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('session');
  });

  test('should show empty state when no sessions', async ({ authenticatedPage }) => {
    // Mock the API to return empty list
    await authenticatedPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(authenticatedPage);
    await sessionsPage.goto();
    await sessionsPage.waitForLoad();

    const isEmpty = await sessionsPage.hasEmptyMessage();
    expect(isEmpty).toBe(true);
  });

  test('should display active sessions', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Server',
              },
              type: 'ssh',
              status: 'active',
              started_at: new Date(Date.now() - 3600000).toISOString(),
              duration_seconds: 3600,
              can_terminate: true,
            },
            {
              id: 'session-2',
              user: {
                id: 'user-2',
                first_name: 'Jane',
                last_name: 'Smith',
                email: 'jane@example.com',
              },
              target: {
                id: 'target-2',
                name: 'Database Server',
              },
              type: 'rdp',
              status: 'active',
              started_at: new Date(Date.now() - 1800000).toISOString(),
              duration_seconds: 1800,
              can_terminate: true,
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show sessions
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'John Doe' })).toBeVisible();
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'Jane Smith' })).toBeVisible();
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'Production Server' })).toBeVisible();
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'Database Server' })).toBeVisible();
  });

  test('should show active status indicator', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: {
                id: 'user-1',
                first_name: 'Active',
                last_name: 'User',
                email: 'active@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Test Server',
              },
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              duration_seconds: 0,
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show active status with ping animation
    const activeIndicator = mockApiPage.locator('.animate-ping').or(
      mockApiPage.locator('[class*="ping"]')
    );
    await expect(activeIndicator).toBeVisible();
  });

  test('should filter sessions by status', async ({ mockApiPage }) => {
    let statusFilterCount = 0;
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      const url = route.request().url();
      if (url.includes('status=active')) {
        statusFilterCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Server' },
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Select status filter
    await sessionsPage.statusFilter.selectOption({ label: 'Active' });
    await mockApiPage.waitForTimeout(500);

    expect(statusFilterCount).toBeGreaterThan(0);
  });

  test('should filter sessions by type', async ({ mockApiPage }) => {
    let typeFilterCount = 0;
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      const url = route.request().url();
      if (url.includes('type=ssh')) {
        typeFilterCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'SSH Server' },
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Select type filter
    await sessionsPage.typeFilter.selectOption({ label: 'SSH' });
    await mockApiPage.waitForTimeout(500);

    expect(typeFilterCount).toBeGreaterThan(0);
  });

  test('should search sessions', async ({ mockApiPage }) => {
    let searchRequestCount = 0;
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      const url = route.request().url();
      if (url.includes('search=john')) {
        searchRequestCount++;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Server' },
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Enter search term
    await sessionsPage.search('john');
    await mockApiPage.waitForTimeout(500);

    expect(searchRequestCount).toBeGreaterThan(0);
  });

  test('should open terminate modal', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Production Server' },
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click terminate button
    await sessionsPage.clickTerminate(0);

    // Modal should open
    const isModalOpen = await sessionsPage.isModalOpen();
    expect(isModalOpen).toBe(true);

    // Check modal content
    await expect(mockApiPage.getByText('Terminate Session')).toBeVisible();
    await expect(mockApiPage.getByText(/Are you sure/)).toBeVisible();
  });

  test('should terminate session with reason', async ({ mockApiPage }) => {
    let terminateCalled = false;
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      // Initial list request
      if (!route.request().url().includes('terminate')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'session-1',
                user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
                target: { id: 'target-1', name: 'Server' },
                type: 'ssh',
                status: 'active',
                started_at: new Date().toISOString(),
                can_terminate: true,
              },
            ],
            pagination: { total: 1, limit: 20, offset: 0 },
          }),
        });
      } else {
        // Terminate request
        terminateCalled = true;
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click terminate button
    await sessionsPage.clickTerminate(0);

    // Enter reason and confirm
    await sessionsPage.enterTerminateReason('Security violation detected');
    await sessionsPage.confirmTerminate();

    // Wait for API call
    await mockApiPage.waitForTimeout(100);
    expect(terminateCalled).toBe(true);
  });

  test('should cancel terminate action', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      if (!route.request().url().includes('terminate')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'session-1',
                user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
                target: { id: 'target-1', name: 'Server' },
                type: 'ssh',
                status: 'active',
                started_at: new Date().toISOString(),
                can_terminate: true,
              },
            ],
            pagination: { total: 1, limit: 20, offset: 0 },
          }),
        });
      }
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click terminate button
    await sessionsPage.clickTerminate(0);

    // Cancel the modal
    await sessionsPage.cancelTerminate();

    // Modal should be closed
    const isModalOpen = await sessionsPage.isModalOpen();
    expect(isModalOpen).toBe(false);
  });

  test('should show duration for active sessions', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Server' },
              type: 'ssh',
              status: 'active',
              started_at: new Date(Date.now() - 300000).toISOString(), // 5 minutes ago
              can_terminate: true,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show duration in Xm Xs format for active sessions
    const durationText = await mockApiPage.getByText(/\d+m\s+\d+s/).textContent().catch(() => null);
    expect(durationText).toBeTruthy();
  });

  test('should redirect to login when unauthenticated', async ({ page }) => {
    await page.goto('/sessions');

    // Should redirect to login
    await page.waitForURL('/login', { timeout: 5000 });

    await expect(page.getByText('Sign in to your account')).toBeVisible();
  });

  test('should show ended sessions with different status', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Server' },
              type: 'ssh',
              status: 'ended',
              started_at: new Date(Date.now() - 7200000).toISOString(),
              duration_seconds: 7200,
              can_terminate: false,
            },
            {
              id: 'session-2',
              user: { id: 'user-2', first_name: 'Jane', last_name: 'Smith', email: 'jane@example.com' },
              target: { id: 'target-2', name: 'Database' },
              type: 'rdp',
              status: 'terminated',
              started_at: new Date(Date.now() - 3600000).toISOString(),
              duration_seconds: 3600,
              can_terminate: false,
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show ended/terminated sessions (in the table data, not just the dropdown)
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'ended' })).toBeVisible();
    await expect(mockApiPage.getByRole('cell').filter({ hasText: 'terminated' })).toBeVisible();
  });

  test('should not show terminate button for ended sessions', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'session-1',
              user: { id: 'user-1', first_name: 'John', last_name: 'Doe', email: 'john@example.com' },
              target: { id: 'target-1', name: 'Server' },
              type: 'ssh',
              status: 'ended',
              started_at: new Date(Date.now() - 3600000).toISOString(),
              duration_seconds: 3600,
              can_terminate: false,
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should not show terminate button for ended sessions
    const terminateButtons = mockApiPage.getByRole('button', { name: /Terminate/i });
    const count = await terminateButtons.count();
    expect(count).toBe(0);
  });

  test('should handle session list pagination', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: Array.from({ length: 20 }, (_, i) => ({
            id: `session-${i}`,
            user: { id: `user-${i}`, first_name: `User${i}`, last_name: 'Test', email: `user${i}@example.com` },
            target: { id: `target-${i}`, name: `Server ${i}` },
            type: 'ssh',
            status: 'ended',
            started_at: new Date().toISOString(),
            duration_seconds: 3600,
            can_terminate: false,
          })),
          pagination: { total: 100, limit: 20, offset: 0 },
        }),
      });
    });

    const sessionsPage = new SessionsPage(mockApiPage);
    await sessionsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show pagination when there are more results
    const hasPagination = await mockApiPage.getByRole('button', { name: /next|more|>/i }).count() > 0 ||
      await mockApiPage.getByText(/1\s+of\s+\d+/).isVisible().catch(() => false);
    expect(hasPagination).toBe(true);
  });
});
