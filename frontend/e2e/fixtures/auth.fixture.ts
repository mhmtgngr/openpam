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

  // Policies routes
  page.route('**/api/v1/policies/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });

  // Roles routes
  page.route('**/api/v1/roles**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
  page.route('**/api/v1/roles/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { id: 'role-1', name: 'Test Role', permissions: [] } }),
    });
  });

  // Tenants routes
  page.route('**/api/v1/tenants**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
  page.route('**/api/v1/tenants/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { id: 'tenant-1', name: 'Test Tenant', slug: 'test-tenant' } }),
    });
  });

  // Analytics routes
  page.route('**/api/v1/analytics/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        metrics: { total_sessions: 0, total_users: 0, avg_duration: 0 },
        trends: { sessions: { current: 0, previous: 0 }, users: { current: 0, previous: 0 } },
        realtime: { active_sessions: 0, active_users: 0, sessions_last_hour: 0, avg_active_duration: 0 },
        data: [],
        pagination: { total: 0, offset: 0, limit: 20 }
      }),
    });
  });

  // Compliance routes
  page.route('**/api/v1/compliance/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        overall_score: 100,
        control_count: 0,
        compliant_count: 0,
        non_compliant_count: 0,
        last_assessed: new Date().toISOString(),
        data: [],
        pagination: { total: 0, offset: 0, limit: 20 }
      }),
    });
  });

  // Reports routes
  page.route('**/api/v1/reports/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });

  // Sessions routes
  page.route('**/api/v1/sessions**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
  page.route('**/api/v1/sessions/**', async (route) => {
    const url = route.request().url();
    if (route.request().method() === 'POST' && url.includes('/terminate')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'session-1',
            user: { id: 'user-1', first_name: 'Test', last_name: 'User', email: 'test@example.com' },
            target: { id: 'target-1', name: 'Test Server' },
            type: 'ssh',
            status: 'active',
            started_at: new Date().toISOString(),
            can_terminate: true,
          },
        }),
      });
    }
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

  // Click and wait for navigation - use Promise.race to handle both cases
  // The button might become disabled during loading, so we click once and wait for URL change
  await submitButton.click();

  // Wait for either dashboard URL or any navigation to complete
  await page.waitForURL('**/dashboard', { timeout: 15000 }).catch(() => {
    // If URL wait fails, try waiting for load state instead
    return page.waitForLoadState('load', { timeout: 5000 });
  });

  // Wait for network to be idle to ensure all API calls complete
  await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {
    // If networkidle times out, continue anyway - page might be loaded enough
  });
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
  // This page is NOT authenticated by default - tests must set up their own auth state
  mockApiPage: async ({ page }, use) => {
    setupMocks(page);

    // Start with a clean state
    await page.goto('/');

    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    await use(page);
  },
});

export const expect = test.expect;
