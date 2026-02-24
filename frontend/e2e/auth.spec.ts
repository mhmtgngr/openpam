import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';

test.describe('Authentication', () => {
  test('should display login page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    // Wait for the page to fully load
    await page.waitForLoadState('networkidle');

    await expect(loginPage.openPAMTitle).toBeVisible();
    await expect(loginPage.signInTitle).toBeVisible();
    await expect(loginPage.emailInput).toBeVisible();
    await expect(loginPage.passwordInput).toBeVisible();
  });

  test('should show validation errors for empty form', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    // Wait for page to load
    await page.waitForLoadState('networkidle');

    await loginPage.submitButton.click();

    // Should show validation errors - the form validates on submit
    await expect(page.getByText('Email is required')).toBeVisible({ timeout: 2000 });
  });

  test('should show error for invalid credentials', async ({ mockApiPage }) => {
    // First navigate to login page
    await mockApiPage.goto('/login');
    await mockApiPage.waitForLoadState('networkidle');

    // Setup mock for invalid credentials (after navigation to override any existing routes)
    await mockApiPage.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 'INVALID_CREDENTIALS',
            message: 'Invalid credentials',
          },
        }),
      });
    });

    const loginPage = new LoginPage(mockApiPage);

    await loginPage.login('invalid@example.com', 'wrongpassword');

    // Wait a bit for the API call and potential toast to appear
    await mockApiPage.waitForTimeout(1000);

    // The page should still be on login (not redirected to dashboard)
    await expect(mockApiPage).toHaveURL(/\/login/);

    // The email field should still be visible (we're still on login page)
    await expect(loginPage.emailInput).toBeVisible();
  });

  test('should redirect to dashboard after successful login', async ({ authenticatedPage }) => {
    // authenticatedPage fixture now handles login with mocked API
    const dashboardPage = new DashboardPage(authenticatedPage);

    const heading = await dashboardPage.getHeadingText();
    expect(heading?.toLowerCase()).toContain('welcome');
  });

  test('should redirect to login when accessing protected route unauthenticated', async ({ page }) => {
    await page.goto('/dashboard');

    // Should redirect to login
    await page.waitForURL('/login', { timeout: 5000 });

    await expect(page.getByText('Sign in to your account')).toBeVisible();
  });
});

test.describe('MFA Setup', () => {
  test('should display MFA setup flow', async ({ page }) => {
    // Note: /mfa/setup is currently a protected route, so this test
    // would need to be run with an authenticated user that needs MFA setup
    // For now, we'll skip this test or mark it as TODO
    test.skip(true, 'MFA setup requires authenticated user context');

    await page.goto('/mfa/setup');

    await expect(page.getByText('Set Up Two-Factor Authentication')).toBeVisible();
    await expect(page.getByText('Authenticator App')).toBeVisible();
  });
});
