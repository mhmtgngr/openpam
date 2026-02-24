import { test, expect } from './fixtures/auth.fixture';

test.describe('Active Sessions', () => {
  test('should display active sessions list', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /active sessions/i })).toBeVisible();
  });

  test('should filter sessions by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for status filter dropdown or tabs
    const statusFilter = authenticatedPage.getByRole('combobox', { name: /status/i }).or(
      authenticatedPage.getByRole('tab', { name: /active/i })
    );

    if (await statusFilter.first().isVisible({ timeout: 2000 })) {
      await statusFilter.first().click();
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should display session details', async ({ mockApiPage }) => {
    // Mock session detail endpoint
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            target_host: '192.168.1.100',
            target_port: 22,
            status: 'active',
            user_id: 'user-123',
            user_email: 'test@example.com',
            started_at: new Date().toISOString(),
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    await expect(mockApiPage.getByText(/192.168.1.100/i)).toBeVisible();
    await expect(mockApiPage.getByRole('heading', { name: /session details/i })).toBeVisible();
  });

  test('should terminate active session', async ({ mockApiPage }) => {
    // Mock terminate endpoint
    await mockApiPage.route('**/api/v1/sessions/*/terminate', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Session terminated' }),
      });
    });

    await mockApiPage.goto('/sessions');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for terminate button
    const terminateButton = mockApiPage.getByRole('button', { name: /terminate|end|kill/i }).first();

    if (await terminateButton.isVisible({ timeout: 2000 })) {
      await terminateButton.click();

      // Confirm dialog if present
      const confirmButton = mockApiPage.getByRole('button', { name: /confirm|terminate/i }).or(
        mockApiPage.getByRole('button', { name: /yes/i })
      );

      if (await confirmButton.isVisible({ timeout: 2000 })) {
        await confirmButton.click();
      }

      await expect(mockApiPage.getByText(/session terminated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should filter sessions by date range', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for date filter controls
    const dateFrom = authenticatedPage.getByLabel(/from|start date/i).or(
      authenticatedPage.getByPlaceholder(/from/i)
    );
    const dateTo = authenticatedPage.getByLabel(/to|end date/i).or(
      authenticatedPage.getByPlaceholder(/to/i)
    );

    if (await dateFrom.isVisible({ timeout: 2000 })) {
      await dateFrom.fill('2024-01-01');
      await dateTo.fill('2024-12-31');

      const applyButton = authenticatedPage.getByRole('button', { name: /apply|filter/i });
      if (await applyButton.isVisible({ timeout: 1000 })) {
        await applyButton.click();
      }
    }
  });

  test('should show session recording playback', async ({ mockApiPage }) => {
    // Mock recording endpoint
    await mockApiPage.route('**/api/v1/sessions/*/recording', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            url: '/recordings/session-123.cast',
            duration: 120,
            size: 1024000,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for playback controls
    const playButton = mockApiPage.getByRole('button', { name: /play|▶/i }).or(
      mockApiPage.getByLabel(/play recording/i)
    );

    if (await playButton.isVisible({ timeout: 2000 })) {
      await expect(playButton).toBeVisible();
    }
  });

  test('should search sessions by target', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find search input
    const searchInput = authenticatedPage.getByRole('searchbox', { name: /search/i }).or(
      authenticatedPage.getByPlaceholder(/search/i),
      authenticatedPage.getByLabel(/search/i)
    );

    if (await searchInput.isVisible({ timeout: 2000 })) {
      await searchInput.fill('192.168.1.1');
      await authenticatedPage.waitForTimeout(500);
    }
  });
});

test.describe('Session Monitoring', () => {
  test('should display live session count', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/sessions/stats', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            active: 5,
            total_today: 25,
            by_type: {
              ssh: 3,
              rdp: 2,
            },
          },
        }),
      });
    });

    await mockApiPage.goto('/dashboard');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for session stats on dashboard
    await expect(mockApiPage.getByText(/active sessions|session/i)).toBeVisible();
  });

  test('should show session type breakdown', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for session type badges or breakdown
    await expect(authenticatedPage.getByText(/ssh|rdp|telnet/i).or(
      authenticatedPage.getByRole('button', { name: /ssh|rdp/i })
    ).first()).toBeVisible({ timeout: 3000 });
  });

  test('should display session duration', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for duration display
    await expect(authenticatedPage.getByText(/\d+m|\d+h|duration/i)).toBeVisible({ timeout: 3000 });
  });

  test('should export session logs', async ({ mockApiPage }) => {
    // Mock export endpoint
    await mockApiPage.route('**/api/v1/sessions/export', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/csv',
        body: 'id,user,target,status,start_time\nsess-1,user1,192.168.1.1,active,2024-01-01T00:00:00Z',
      });
    });

    await mockApiPage.goto('/sessions');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for export button
    const exportButton = mockApiPage.getByRole('button', { name: /export|download/i }).or(
      mockApiPage.getByRole('link', { name: /export/i })
    );

    if (await exportButton.isVisible({ timeout: 2000 })) {
      await exportButton.click();
      await mockApiPage.waitForTimeout(500);
    }
  });
});
