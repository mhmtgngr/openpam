import { test as base, type Page } from '@playwright/test';
import { mockAuthRoutes } from '../handlers/auth';
import { mockDashboardRoutes } from '../handlers/dashboard';
import { mockUserRoutes } from '../handlers/users';
import { mockRequestRoutes } from '../handlers/requests';

// Helper to check if backend is available
const isBackendAvailable = () => {
  return process.env.CI !== undefined || process.env.BACKEND_AVAILABLE === '1';
};

// Setup API mocking
const setupMocks = (page: Page) => {
  // Mock all API routes
  page.route('**/api/v1/auth/**', mockAuthRoutes);
  page.route('**/api/v1/dashboard/**', mockDashboardRoutes);
  page.route('**/api/v1/users**', mockUserRoutes);
  page.route('**/api/v1/requests/**', mockRequestRoutes);
  page.route('**/api/v1/approvals/**', mockRequestRoutes);
  // Catch-all for any API calls
  page.route('**/api/**', async (route) => {
    // Return empty successful response for unhandled routes
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

// Helper to perform login
const performLogin = async (page: Page) => {
  // Navigate to login page first
  await page.goto('/login');

  // Clear localStorage after navigation to ensure no stale auth state
  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  // Wait for the loading spinner to disappear and the form to be visible
  // The page may show a loading spinner initially while checking auth
  await page.waitForSelector('input[type="email"]', { state: 'visible', timeout: 15000 });

  const emailInput = page.locator('input[type="email"]').first();
  const passwordInput = page.locator('input[type="password"]');
  const submitButton = page.getByRole('button', { name: /Sign In/ });

  await emailInput.fill('test@example.com');
  await passwordInput.fill('testpassword123');
  await submitButton.click();

  await page.waitForURL('/dashboard', { timeout: 15000 });
  await page.waitForLoadState('networkidle');
};

export const test = base.extend<{
  authenticatedPage: typeof base.prototype['page'];
  mockApiPage: typeof base.prototype['page'];
}>({
  // Authenticated page fixture - uses mocked API if no backend available
  authenticatedPage: async ({ page }, use) => {
    // Setup mocks if backend is not available
    if (!isBackendAvailable()) {
      setupMocks(page);
    }

    await performLogin(page);
    await use(page);
  },

  // Page with mocked API (always mocks, even if backend is available)
  mockApiPage: async ({ page }, use) => {
    setupMocks(page);
    await use(page);
  },
});

export const expect = test.expect;
