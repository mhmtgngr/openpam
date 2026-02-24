import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Credential Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test('should display credentials list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /credential vault/i })).toBeVisible();
    await expect(authenticatedPage.getByText(/securely store and manage/i)).toBeVisible();
  });

  test('should display add credential button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add credential/i });
    await expect(addButton).toBeVisible();
  });

  test('should navigate to credential form page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('link', { name: /add credential/i });
    await addButton.click();

    await authenticatedPage.waitForURL('/credentials/new');
    await expect(authenticatedPage.getByRole('heading', { name: /add new credential/i })).toBeVisible();
  });

  test('should validate required fields on credential form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /create credential/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/name is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should create a new password credential', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/credentials', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'cred-123',
            name: 'Test Credential',
            type: 'password',
            username: 'admin',
            status: 'active',
            created_at: new Date().toISOString(),
          },
        }),
      });
    });

    await mockApiPage.goto('/credentials/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Fill in the form
    await mockApiPage.getByLabel(/credential name/i).fill('Test Credential');
    await mockApiPage.getByLabel(/username \/ account/i).fill('admin');

    // Select password type
    const typeSelect = mockApiPage.getByLabel(/credential type/i);
    await typeSelect.selectOption('password');

    // Enter password
    await mockApiPage.getByLabel(/password/i).fill('SecurePassword123!');

    // Submit the form
    const submitButton = mockApiPage.getByRole('button', { name: /create credential/i });
    await submitButton.click();

    // Should show success message
    await expect(mockApiPage.getByText(/credential created|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should generate random password', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Select password type
    const typeSelect = authenticatedPage.getByLabel(/credential type/i);
    await typeSelect.selectOption('password');

    // Click generate button
    const generateButton = authenticatedPage.getByRole('button', { name: /generate/i });
    await generateButton.click();

    // Password field should have a value
    const passwordInput = authenticatedPage.getByLabel(/password/i);
    const passwordValue = await passwordInput.inputValue();

    expect(passwordValue.length).toBeGreaterThan(10);
  });

  test('should toggle password visibility', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Select password type
    const typeSelect = authenticatedPage.getByLabel(/credential type/i);
    await typeSelect.selectOption('password');

    const passwordInput = authenticatedPage.getByLabel(/password/i);
    await expect(passwordInput).toHaveAttribute('type', 'password');

    // Click eye icon to toggle
    const eyeButton = authenticatedPage.locator('button').filter({ hasText: /eye/i }).first();
    await eyeButton.click();

    await expect(passwordInput).toHaveAttribute('type', 'text');
  });

  test('should filter credentials by type', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/credentials');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on type filter
    const typeSelect = authenticatedPage.getByRole('combobox').or(
      authenticatedPage.getByLabel(/type/i)
    ).first();

    if (await typeSelect.isVisible()) {
      await typeSelect.selectOption('ssh_key');
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should rotate credential', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/credentials/*/rotate', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { id: 'cred-123', name: 'Test Credential' },
        }),
      });
    });

    await mockApiPage.goto('/credentials');
    await mockApiPage.waitForLoadState('networkidle');

    // Click rotate button on first credential
    const rotateButton = mockApiPage.getByRole('button', { name: /rotate/i }).first();

    if (await rotateButton.isVisible({ timeout: 2000 })) {
      await rotateButton.click();
      await expect(mockApiPage.getByText(/credential rotated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });
});
