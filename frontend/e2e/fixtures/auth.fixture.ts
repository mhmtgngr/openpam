import { test as base, type Page } from '@playwright/test';
import { mockAuthRoutes } from '../handlers/auth';
import { mockDashboardRoutes } from '../handlers/dashboard';
import { mockUserRoutes } from '../handlers/users';
import { mockRequestRoutes } from '../handlers/requests';
import { mockAnalyticsRoutes, mockComplianceRoutes } from '../handlers/analytics';

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
  page.route('**/api/v1/analytics/**', mockAnalyticsRoutes);

  // Compliance routes
  page.route('**/api/v1/compliance/**', mockComplianceRoutes);

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

// Helper to setup access policy specific routes
const setupAccessPolicyRoutes = (page: Page) => {
  const mockAccessPolicies = [
    {
      id: 'policy-1',
      name: 'Production Database Access',
      description: 'Controls access to production database credentials',
      status: 'active',
      priority: 100,
      rules: [
        {
          id: 'rule-1',
          name: 'Allow admins',
          effect: 'allow',
          conditions: [
            {
              id: 'cond-1',
              type: 'role',
              field: 'role',
              operator: 'in',
              value: ['admin'],
            },
          ],
          logical_operator: 'AND',
          priority: 1,
          resources: ['credential:prod-db-*'],
          actions: ['checkout', 'connect'],
          roles: ['admin'],
          users: [],
        },
        {
          id: 'rule-2',
          name: 'Deny regular users',
          effect: 'deny',
          conditions: [],
          logical_operator: 'AND',
          priority: 2,
          resources: ['credential:prod-db-*'],
          actions: ['checkout'],
          roles: ['user'],
          users: [],
        },
      ],
      conflict_resolution: 'deny_overrides',
      is_default: false,
      is_system: false,
      tenant_id: 'tenant-1',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      tags: ['production', 'database'],
    },
    {
      id: 'policy-2',
      name: 'SSH Access Policy',
      description: 'SSH session access controls',
      status: 'active',
      priority: 50,
      rules: [],
      conflict_resolution: 'deny_overrides',
      is_default: true,
      is_system: false,
      tenant_id: 'tenant-1',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
      tags: ['ssh', 'sessions'],
    },
  ];

  const mockPolicyDetail = {
    ...mockAccessPolicies[0],
  };

  // Unregister the generic policies route first, then set up specific ones
  // Note: Playwright doesn't support unregistering routes, so we need to set up
  // our specific routes BEFORE the generic one is registered. This means we need
  // to NOT call setupMocks for policies, or modify setupMocks to skip policies.

  page.route('**/api/v1/policies/access*', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/policies/access - List policies
    if (url.includes('/api/v1/policies/access') && !url.includes('/test') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockAccessPolicies, pagination: { total: 2, offset: 0, limit: 20 } }),
      });
      return;
    }

    // GET /api/v1/policies/access/:id - Get policy detail
    if (url.includes('/api/v1/policies/access/') && method === 'GET' && !url.includes('/test')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPolicyDetail),
      });
      return;
    }

    // POST /api/v1/policies/access - Create policy
    if (url.includes('/api/v1/policies/access') && method === 'POST' && !url.includes('/test')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...mockAccessPolicies[0],
          id: 'new-policy-id',
          name: 'New Access Policy',
        }),
      });
      return;
    }

    // PATCH /api/v1/policies/access/:id - Update policy
    if (url.includes('/api/v1/policies/access/') && method === 'PATCH') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPolicyDetail),
      });
      return;
    }

    // DELETE /api/v1/policies/access/:id - Delete policy
    if (url.includes('/api/v1/policies/access/') && method === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Policy deleted' }),
      });
      return;
    }

    // POST /api/v1/policies/access/validate - Validate policy
    if (url.includes('/validate') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ valid: true, errors: [], warnings: [] }),
      });
      return;
    }

    // POST /api/v1/policies/access/evaluate - Evaluate policy
    if (url.includes('/evaluate') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          allowed: true,
          effect: 'allow',
          matched_policy_id: 'policy-1',
          matched_rule_id: 'rule-1',
          reason: 'User has admin role',
          details: [],
          evaluated_at: new Date().toISOString(),
        }),
      });
      return;
    }

    // POST /api/v1/policies/access/test - Test policy
    if (url.includes('/test') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            scenario_name: 'Test Scenario',
            result: {
              allowed: true,
              effect: 'allow',
              reason: 'Test passed',
              details: [],
              evaluated_at: new Date().toISOString(),
            },
            passed: true,
            expected_allowed: true,
          },
        ]),
      });
      return;
    }

    // Default fallback
    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Route not found' } }),
    });
  });
};

export const test = base.extend<{
  authenticatedPage: typeof base.prototype['page'];
  mockApiPage: typeof base.prototype['page'];
  accessPolicyPage: typeof base.prototype['page'];
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

  // Access policy page fixture - authenticated with access policy specific routes
  accessPolicyPage: async ({ page }, use) => {
    // Setup general mocks first
    setupMocks(page);

    // Setup access policy specific routes (these will be registered AFTER the general ones)
    // In Playwright, routes are matched in LIFO order (last registered, first matched)
    setupAccessPolicyRoutes(page);

    await performLogin(page);
    await use(page);
  },
});

export const expect = test.expect;
