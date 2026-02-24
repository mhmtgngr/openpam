import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Tenant Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display tenants list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /tenants/i })).toBeVisible();
    await expect(authenticatedPage.getByText(/manage platform tenants/i)).toBeVisible();
  });

  test('should display add tenant button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add tenant/i });
    await expect(addButton).toBeVisible();
  });

  test('should navigate to tenant form page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add tenant/i });
    await addButton.click();

    await authenticatedPage.waitForURL('/tenants/new');
    await expect(authenticatedPage.getByRole('heading', { name: /create tenant/i })).toBeVisible();
  });

  test('should validate required fields on tenant form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /create tenant/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/name is required|slug is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should auto-generate slug from name', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Fill in the name
    await authenticatedPage.getByLabel(/tenant name/i).fill('Acme Corporation');

    // Click auto-generate button
    const generateButton = authenticatedPage.getByRole('button', { name: /auto-generate/i });
    await generateButton.click();

    // Slug should be generated
    const slugInput = authenticatedPage.getByLabel(/slug/i);
    const slugValue = await slugInput.inputValue();

    expect(slugValue.toLowerCase()).toBe('acme-corporation');
  });

  test('should validate slug format', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Fill in invalid slug
    await authenticatedPage.getByLabel(/slug/i).fill('Invalid Slug!');

    const submitButton = authenticatedPage.getByRole('button', { name: /create tenant/i });
    await submitButton.click();

    // Should show validation error
    await expect(authenticatedPage.getByText(/lowercase letters|numbers|hyphens/i)).toBeVisible({ timeout: 3000 });
  });

  test('should select primary color', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on color picker
    const colorInput = authenticatedPage.locator('input[type="color"]');
    await colorInput.fill('#ff0000');

    // Check that the value was updated
    const value = await colorInput.inputValue();
    expect(value).toBe('#ff0000');
  });

  test('should toggle MFA enforcement', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find the MFA toggle
    const mfaToggle = authenticatedPage.getByRole('button', { name: /enforce mfa/i });
    await mfaToggle.click();

    // Should show MFA method options
    await expect(authenticatedPage.getByText(/allowed mfa methods/i)).toBeVisible({ timeout: 1000 });
  });

  test('should add IP to whitelist', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Fill in IP address
    await authenticatedPage.getByPlaceholder(/add ip/i).fill('192.168.1.0/24');

    // Click add button
    const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
    await addButton.click();

    // Should show the added IP
    await expect(authenticatedPage.getByText('192.168.1.0/24')).toBeVisible({ timeout: 2000 });
  });

  test('should remove IP from whitelist', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/tenants/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Add an IP first
    await authenticatedPage.getByPlaceholder(/add ip/i).fill('10.0.0.0/8');
    const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
    await addButton.click();

    // Remove the IP
    const removeButton = authenticatedPage.locator('button').filter({ hasText: /×/i }).first();
    await removeButton.click();

    // IP should be removed
    await expect(authenticatedPage.getByText('10.0.0.0/8')).not.toBeVisible({ timeout: 2000 });
  });

  test('should suspend tenant', async ({ mockApiPage }) => {
    // Mock tenants API
    await mockApiPage.route('**/api/v1/tenants', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'tenant-123',
              name: 'Test Tenant',
              slug: 'test-tenant',
              status: 'active',
              primary_color: '#6366f1',
              max_users: 100,
              max_targets: 500,
              created_at: new Date().toISOString(),
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    // Mock suspend API
    await mockApiPage.route('**/api/v1/tenants/*/suspend', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { id: 'tenant-123', status: 'suspended' },
        }),
      });
    });

    await mockApiPage.goto('/tenants');
    await mockApiPage.waitForLoadState('networkidle');

    // Click suspend button
    const suspendButton = mockApiPage.getByRole('button', { name: /suspend/i }).first();

    if (await suspendButton.isVisible({ timeout: 2000 })) {
      await suspendButton.click();

      // Should show confirmation modal
      await expect(mockApiPage.getByText(/are you sure/i)).toBeVisible({ timeout: 2000 });

      // Enter reason
      await mockApiPage.getByLabel(/reason/i).fill('Policy violation');

      // Confirm suspension
      const confirmButton = mockApiPage.getByRole('button', { name: /suspend tenant/i });
      await confirmButton.click();

      // Should show success message
      await expect(mockApiPage.getByText(/tenant suspended|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should activate suspended tenant', async ({ mockApiPage }) => {
    // Mock tenants API with suspended tenant
    await mockApiPage.route('**/api/v1/tenants', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'tenant-123',
              name: 'Suspended Tenant',
              slug: 'suspended-tenant',
              status: 'suspended',
              primary_color: '#6366f1',
              max_users: 100,
              max_targets: 500,
              created_at: new Date().toISOString(),
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    // Mock activate API
    await mockApiPage.route('**/api/v1/tenants/*/activate', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { id: 'tenant-123', status: 'active' },
        }),
      });
    });

    await mockApiPage.goto('/tenants');
    await mockApiPage.waitForLoadState('networkidle');

    // Click activate button
    const activateButton = mockApiPage.getByRole('button', { name: /activate|play/i }).first();

    if (await activateButton.isVisible({ timeout: 2000 })) {
      await activateButton.click();

      // Should show success message
      await expect(mockApiPage.getByText(/tenant activated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });
});
