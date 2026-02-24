import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Access Requests', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display my requests page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /my requests/i })).toBeVisible();
    await expect(authenticatedPage.getByText(/View and manage your access requests/i)).toBeVisible();
  });

  test('should display create request button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    const createButton = authenticatedPage.getByRole('link', { name: /new request|create request/i }).or(
      authenticatedPage.getByRole('button', { name: /new request|create request/i })
    );
    await expect(createButton.first()).toBeVisible();
  });

  test('should navigate to create request page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    const createButton = authenticatedPage.getByRole('link', { name: /new request|create request/i }).first();
    await createButton.click();

    await authenticatedPage.waitForURL(/\/requests\/new|\/requests\/create/);
    await expect(authenticatedPage.getByRole('heading', { name: /request access/i })).toBeVisible();
  });

  test('should validate required fields on request form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /submit request/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/target is required|reason is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should create a session access request', async ({ mockApiPage }) => {
    // Mock targets API
    await mockApiPage.route('**/api/v1/targets*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'target-123',
              name: 'Production Server',
              type: 'ssh',
              host: 'prod.example.com',
              port: 22,
              environment: 'production',
              sensitivity: 'high',
              status: 'online',
              require_approval: true,
              require_mfa: true,
            },
          ],
          pagination: { total: 1, limit: 100, offset: 0, has_more: false },
        }),
      });
    });

    // Mock create request API
    await mockApiPage.route('**/api/v1/requests', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'req-123',
            status: 'pending',
            type: 'session_access',
            created_at: new Date().toISOString(),
          },
        }),
      });
    });

    await mockApiPage.goto('/requests/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Select request type
    await mockApiPage.getByText(/session access/i).click();

    // Select target
    const targetSelect = mockApiPage.getByLabel(/target/i);
    await targetSelect.selectOption('target-123');

    // Select duration
    await mockApiPage.getByText(/1 hour/i).click();

    // Enter reason
    await mockApiPage.getByLabel(/reason for access/i).fill('Need to troubleshoot production issue');

    // Submit the form
    const submitButton = mockApiPage.getByRole('button', { name: /submit request/i });
    await submitButton.click();

    // Should show success message
    await expect(mockApiPage.getByText(/request created|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should show access requirements for selected target', async ({ mockApiPage }) => {
    // Mock targets API
    await mockApiPage.route('**/api/v1/targets*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'target-123',
              name: 'Production Server',
              type: 'ssh',
              host: 'prod.example.com',
              port: 22,
              environment: 'production',
              sensitivity: 'high',
              status: 'online',
              require_approval: true,
              require_mfa: true,
              max_session_duration: 120,
            },
          ],
          pagination: { total: 1, limit: 100, offset: 0, has_more: false },
        }),
      });
    });

    await mockApiPage.goto('/requests/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Select target
    const targetSelect = mockApiPage.getByLabel(/target/i);
    await targetSelect.selectOption('target-123');

    // Should show access requirements
    await expect(mockApiPage.getByText(/access requirements/i)).toBeVisible({ timeout: 2000 });
    await expect(mockApiPage.getByText(/approval required|yes/i)).toBeVisible();
    await expect(mockApiPage.getByText(/mfa required|yes/i)).toBeVisible();
  });

  test('should validate reason length', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Enter a short reason
    await authenticatedPage.getByLabel(/reason for access/i).fill('Short');

    const submitButton = authenticatedPage.getByRole('button', { name: /submit request/i });
    await submitButton.click();

    // Should show validation error about minimum length
    await expect(authenticatedPage.getByText(/at least 10 characters/i)).toBeVisible({ timeout: 3000 });
  });

  test('should schedule for later', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Enable scheduling
    const scheduleToggle = authenticatedPage.getByRole('button', { name: /schedule for later/i });
    await scheduleToggle.click();

    // Should show date picker
    await expect(authenticatedPage.getByLabel(/scheduled start time/i)).toBeVisible({ timeout: 1000 });
  });
});
