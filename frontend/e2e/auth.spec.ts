import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';

test.describe('Authentication', () => {
  test('should display login page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await expect(page.locator('text=OpenPAM')).toBeVisible();
    await expect(page.locator('text=Sign in to your account')).toBeVisible();
    await expect(loginPage.emailInput).toBeVisible();
    await expect(loginPage.passwordInput).toBeVisible();
  });

  test('should show validation errors for empty form', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.submitButton.click();

    // Should show validation errors
    await expect(page.locator('text=Email is required')).toBeVisible();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();

    await loginPage.login('invalid@example.com', 'wrongpassword');

    // Should show error message (via toast or inline)
    await expect(page.locator('text=Invalid credentials')).toBeVisible({ timeout: 5000 });
  });

  test('should redirect to dashboard after successful login', async ({ page }) => {
    const loginPage = new LoginPage(page);
    const dashboardPage = new DashboardPage(page);

    await loginPage.goto();
    await loginPage.login('test@example.com', 'testpassword123');

    // Wait for navigation
    await page.waitForURL('/dashboard', { timeout: 10000 });

    const heading = await dashboardPage.getHeadingText();
    expect(heading).toContain('Welcome');
  });

  test('should redirect to login when accessing protected route unauthenticated', async ({ page }) => {
    await page.goto('/dashboard');

    // Should redirect to login
    await page.waitForURL('/login');
    await expect(page.locator('text=Sign in to your account')).toBeVisible();
  });
});

test.describe('MFA Setup', () => {
  test('should display MFA setup flow', async ({ page }) => {
    // This test assumes a user who needs to set up MFA
    await page.goto('/mfa/setup');

    await expect(page.locator('text=Set Up Two-Factor Authentication')).toBeVisible();
    await expect(page.locator('text=Authenticator App')).toBeVisible();
  });
});
