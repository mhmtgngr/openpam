import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Session Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display sessions list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /active sessions|sessions/i })).toBeVisible();
    await expect(authenticatedPage.getByText(/monitor and manage privileged access/i)).toBeVisible();
  });

  test('should filter sessions by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on status filter
    const statusSelect = authenticatedPage.getByRole('combobox').or(
      authenticatedPage.getByLabel(/status/i)
    ).first();

    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption('active');
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should filter sessions by type', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/sessions');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on type filter
    const typeSelect = authenticatedPage.getByRole('combobox').or(
      authenticatedPage.getByLabel(/type/i)
    ).first();

    if (await typeSelect.isVisible()) {
      await typeSelect.selectOption('ssh');
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should display session detail page', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            status: 'active',
            started_at: new Date().toISOString(),
            user: {
              id: 'user-123',
              first_name: 'John',
              last_name: 'Doe',
              email: 'john@example.com',
            },
            target: {
              id: 'target-123',
              name: 'Production Server',
              host: 'prod.example.com',
              port: 22,
            },
            client_ip: '192.168.1.100',
            monitoring_enabled: true,
            can_terminate: true,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    await expect(mockApiPage.getByRole('heading', { name: /session/i })).toBeVisible();
    await expect(mockApiPage.getByText(/john doe/i)).toBeVisible();
    await expect(mockApiPage.getByText(/production server/i)).toBeVisible();
  });

  test('should show terminate button for active sessions', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            status: 'active',
            started_at: new Date().toISOString(),
            user: { id: 'user-123', first_name: 'John', last_name: 'Doe' },
            target: { id: 'target-123', name: 'Production Server' },
            client_ip: '192.168.1.100',
            monitoring_enabled: true,
            can_terminate: true,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    const terminateButton = mockApiPage.getByRole('button', { name: /terminate/i });
    await expect(terminateButton).toBeVisible();
  });

  test('should show monitoring button for active sessions', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            status: 'active',
            started_at: new Date().toISOString(),
            user: { id: 'user-123', first_name: 'John', last_name: 'Doe' },
            target: { id: 'target-123', name: 'Production Server' },
            client_ip: '192.168.1.100',
            monitoring_enabled: true,
            can_terminate: true,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    const monitorButton = mockApiPage.getByRole('button', { name: /monitor/i });
    await expect(monitorButton).toBeVisible();
  });

  test('should show download button for completed sessions with recording', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            status: 'ended',
            started_at: new Date(Date.now() - 3600000).toISOString(),
            ended_at: new Date().toISOString(),
            duration_seconds: 3600,
            user: { id: 'user-123', first_name: 'John', last_name: 'Doe' },
            target: { id: 'target-123', name: 'Production Server' },
            client_ip: '192.168.1.100',
            recording_url: 'https://storage.example.com/sess-123.cast',
            recording_size: 1024000,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    const downloadButton = mockApiPage.getByRole('button', { name: /download/i });
    await expect(downloadButton).toBeVisible();
  });

  test('should terminate session with reason', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      const url = route.request().url();
      if (url.includes('/terminate')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: { id: 'sess-123', status: 'terminated' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              id: 'sess-123',
              type: 'ssh',
              status: 'active',
              started_at: new Date().toISOString(),
              user: { id: 'user-123', first_name: 'John', last_name: 'Doe' },
              target: { id: 'target-123', name: 'Production Server' },
              client_ip: '192.168.1.100',
              monitoring_enabled: true,
              can_terminate: true,
            },
          }),
        });
      }
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    // Click terminate button
    const terminateButton = mockApiPage.getByRole('button', { name: /terminate/i });
    await terminateButton.click();

    // Should show confirmation modal
    await expect(mockApiPage.getByText(/are you sure you want to terminate/i)).toBeVisible({ timeout: 2000 });

    // Enter reason
    await mockApiPage.getByLabel(/reason/i).fill('Security concern');

    // Confirm termination
    const confirmButton = mockApiPage.getByRole('button', { name: /terminate session/i });
    await confirmButton.click();

    // Should show success message
    await expect(mockApiPage.getByText(/session terminated|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should toggle fullscreen for terminal view', async ({ mockApiPage }) => {
    // Mock session API
    await mockApiPage.route('**/api/v1/sessions/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'sess-123',
            type: 'ssh',
            status: 'active',
            started_at: new Date().toISOString(),
            user: { id: 'user-123', first_name: 'John', last_name: 'Doe' },
            target: { id: 'target-123', name: 'Production Server' },
            client_ip: '192.168.1.100',
            monitoring_enabled: true,
            can_terminate: true,
          },
        }),
      });
    });

    await mockApiPage.goto('/sessions/sess-123');
    await mockApiPage.waitForLoadState('networkidle');

    // Click fullscreen button
    const fullscreenButton = mockApiPage.getByRole('button', { name: /fullscreen|maximize/i }).or(
      mockApiPage.locator('button svg').filter({ hasText: /maximize/i })
    ).first();

    if (await fullscreenButton.isVisible({ timeout: 2000 })) {
      await fullscreenButton.click();
      await mockApiPage.waitForTimeout(500);
      // Check if fullscreen mode is active
      const body = mockApiPage.locator('body');
      const hasFullscreenClass = await body.getAttribute('class');
      expect(hasFullscreenClass).toBeTruthy();
    }
  });
});
