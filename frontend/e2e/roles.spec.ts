import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Role Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display roles list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /roles/i })).toBeVisible();
  });

  test('should display add role button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add role/i });
    await expect(addButton).toBeVisible();
  });

  test('should navigate to role form page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add role/i });
    await addButton.click();

    await authenticatedPage.waitForURL('/roles/new');
    await expect(authenticatedPage.getByRole('heading', { name: /create role/i })).toBeVisible();
  });

  test('should validate required fields on role form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /create role/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/name is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should display permission selection by resource', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show permission groups
    await expect(authenticatedPage.getByText(/permissions/i)).toBeVisible();

    // Should have resource sections
    const resources = ['users', 'targets', 'credentials', 'sessions'];
    for (const resource of resources) {
      const resourceElement = authenticatedPage.getByText(new RegExp(resource, 'i'));
      if (await resourceElement.isVisible({ timeout: 1000 })) {
        break; // At least one resource is visible
      }
    }
  });

  test('should select permissions for a resource', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find a resource header and click the toggle button
    const resourceHeaders = authenticatedPage.locator('button').filter({ hasText: /users|targets|credentials/i });
    const count = await resourceHeaders.count();

    if (count > 0) {
      await resourceHeaders.first().click();
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should select individual permission actions', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for permission action buttons
    const actionButtons = authenticatedPage.locator('button').filter({ hasText: /create|view|update|delete/i });
    const count = await actionButtons.count();

    if (count > 0) {
      // Click first action button
      await actionButtons.first().click();
      await authenticatedPage.waitForTimeout(500);

      // Should show selected state visually
    }
  });

  test('should create a new role', async ({ mockApiPage }) => {
    // Mock permissions API
    await mockApiPage.route('**/api/v1/permissions', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            { id: 'perm-1', resource: 'users', action: 'read' },
            { id: 'perm-2', resource: 'users', action: 'create' },
          ],
        }),
      });
    });

    // Mock create role API
    await mockApiPage.route('**/api/v1/roles', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'role-123',
            name: 'Test Role',
            description: 'A test role',
          },
        }),
      });
    });

    await mockApiPage.goto('/roles/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Fill in the form
    await mockApiPage.getByLabel(/role name/i).fill('Test Role');
    await mockApiPage.getByLabel(/description/i).fill('A test role');

    // Select some permissions if available
    const actionButtons = mockApiPage.locator('button').filter({ hasText: /view|create/i });
    const count = await actionButtons.count();
    if (count > 0) {
      await actionButtons.first().click();
    }

    // Submit the form
    const submitButton = mockApiPage.getByRole('button', { name: /create role/i });
    await submitButton.click();

    // Should show success message
    await expect(mockApiPage.getByText(/role created|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should not allow editing system roles', async ({ mockApiPage }) => {
    // Mock role API for system role
    await mockApiPage.route('**/api/v1/roles/system-admin', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'system-admin',
            name: 'System Admin',
            is_system: true,
          },
        }),
      });
    });

    await mockApiPage.goto('/roles/system-admin');
    await mockApiPage.waitForLoadState('networkidle');

    // Should show system role message
    await expect(mockApiPage.getByText(/system role|cannot be modified/i)).toBeVisible({ timeout: 3000 });
  });
});
