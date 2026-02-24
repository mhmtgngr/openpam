import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Target Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display targets list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/targets');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /targets/i })).toBeVisible();
    await expect(authenticatedPage.getByText(/Manage target systems/i)).toBeVisible();
  });

  test('should display add target button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/targets');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add target/i });
    await expect(addButton).toBeVisible();
  });

  test('should navigate to target form page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/targets');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add target/i });
    await addButton.click();

    await authenticatedPage.waitForURL('/targets/new');
    await expect(authenticatedPage.getByRole('heading', { name: /add new target/i })).toBeVisible();
  });

  test('should validate required fields on target form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/targets/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /create target/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/name is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should create a new target', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/targets', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'target-123',
            name: 'Test Target',
            type: 'ssh',
            host: 'test.example.com',
            port: 22,
            environment: 'development',
            sensitivity: 'medium',
            status: 'offline',
            created_at: new Date().toISOString(),
          },
        }),
      });
    });

    await mockApiPage.goto('/targets/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Fill in the form
    await mockApiPage.getByLabel(/target name/i).fill('Test Target');
    await mockApiPage.getByLabel(/hostname \/ ip/i).fill('test.example.com');
    await mockApiPage.getByLabel(/port/i).fill('22');

    // Submit the form
    const submitButton = mockApiPage.getByRole('button', { name: /create target/i });
    await submitButton.click();

    // Should redirect to targets list or show success
    await expect(mockApiPage.getByText(/target created|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should filter targets by type', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/targets');
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

  test('should test connection to target', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/targets/*/test', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            status: 'online',
            latency_ms: 45,
          },
        }),
      });
    });

    await mockApiPage.goto('/targets');
    await mockApiPage.waitForLoadState('networkidle');

    // If there are targets in the list, click test button
    const testButton = mockApiPage.getByRole('button', { name: /test/i }).first();

    if (await testButton.isVisible({ timeout: 2000 })) {
      await testButton.click();
      await expect(mockApiPage.getByText(/connection successful/i)).toBeVisible({ timeout: 5000 });
    }
  });
});
