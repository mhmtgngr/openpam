import { test as base, type Page } from '@playwright/test';
import { mockAuthRoutes } from '../handlers/auth';
import { mockDashboardRoutes } from '../handlers/dashboard';
import { mockUserRoutes } from '../handlers/users';
import { mockRequestRoutes } from '../handlers/requests';

// Setup API mocking - must be done before any navigation
const setupMocks = (page: Page) => {
  // Mock all API routes - these MUST be set up before any navigation
  // The patterns match requests to any domain including localhost:8500
  // We need to be specific to avoid catching Vite's module imports

  // Auth routes - MUST handle /api/v1/auth/me for the initial auth check
  page.route('**/api/v1/auth/*', mockAuthRoutes);

  // Dashboard routes
  page.route('**/api/v1/dashboard/*', mockDashboardRoutes);

  // User routes
  page.route('**/api/v1/users', mockUserRoutes);
  page.route('**/api/v1/users/*', mockUserRoutes);

  // Request/Approval routes
  page.route('**/api/v1/requests/*', mockRequestRoutes);
  page.route('**/api/v1/approvals/*', mockRequestRoutes);

  // Session routes - don't set up catch-all, let tests override
  // page.route('**/api/v1/sessions*', async (route) => { ... });

  // Audit routes
  page.route('**/api/v1/audit/*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });

  // Target routes
  page.route('**/api/v1/targets/*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });

  // Credential routes
  page.route('**/api/v1/credentials/*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });

  // MFA routes
  page.route('**/api/v1/auth/mfa/*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { devices: [] } }),
    });
  });
};

// Helper to perform login
const performLogin = async (page: Page) => {
  // Clear all storage using browser context APIs (works across all pages)
  const context = page.context();
  await context.clearCookies();
  await context.clearPermissions();

  // Navigate to app and clear localStorage/sessionStorage
  await page.goto('/');

  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  // Now navigate to login page
  await page.goto('/login');

  // Wait for the form to appear
  // The loading spinner should not appear because:
  // 1. localStorage is empty (no access_token)
  // 2. AuthContext.checkAuth() returns early when no token
  await page.waitForSelector('input[type="email"]', { state: 'visible', timeout: 10000 });

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
  // Authenticated page fixture - always uses mocked API for tests
  authenticatedPage: async ({ page }, use) => {
    // Setup mocks FIRST, before any navigation
    setupMocks(page);

    await performLogin(page);
    await use(page);
  },

  // Page with mocked API (always mocks, even if backend is available)
  // Also sets up auth tokens so the user is authenticated
  mockApiPage: async ({ page }, use) => {
    setupMocks(page);

    // Set up auth tokens directly without going through login flow
    // This allows the page to bypass the login redirect
    await page.goto('/');

    await page.evaluate(() => {
      localStorage.setItem('access_token', 'mock-test-token');
      localStorage.setItem('refresh_token', 'mock-refresh-token');
    });

    await use(page);
  },
});

export const expect = test.expect;
