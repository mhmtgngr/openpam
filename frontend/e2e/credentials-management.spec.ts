import { test, expect } from './fixtures/auth.fixture';
import { CredentialsPage } from './pages/CredentialsPage';
import { PoliciesPage } from './pages/PoliciesPage';
import { RolesPage } from './pages/RolesPage';
import { TargetsPage } from './pages/TargetsPage';
import { TenantsPage } from './pages/TenantsPage';

// Setup API mocks for credentials tests
const setupCredentialsMocks = (page: any) => {
  page.route('**/api/v1/credentials**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/credentials - List credentials
    if (url.includes('/api/v1/credentials') && !url.includes('/rotate') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'cred-1',
              name: 'Production Database',
              username: 'admin',
              type: 'database',
              protocol: 'postgresql',
              host: 'db.example.com',
              port: 5432,
              status: 'active',
              last_rotation: '2024-01-15T10:00:00Z',
              next_rotation: '2024-02-15T10:00:00Z',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
            {
              id: 'cred-2',
              name: 'SSH Server Root',
              username: 'root',
              type: 'ssh',
              protocol: 'ssh',
              host: 'server1.example.com',
              port: 22,
              status: 'expiring',
              last_rotation: '2024-01-01T00:00:00Z',
              next_rotation: '2024-02-01T00:00:00Z',
              created_at: '2023-12-15T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          ],
          pagination: { total: 2, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    // POST /api/v1/credentials/:id/rotate - Rotate credential
    if (url.includes('/rotate') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          message: 'Credential rotation initiated',
          rotation_id: 'rot-1',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

test.describe('Credentials Management', () => {
  test.beforeEach(async ({ page }) => {
    setupCredentialsMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Credentials List Page', () => {
    test('should display credentials list page', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(credentialsPage.heading).toBeVisible();
    });

    test('should display all credentials', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Production Database')).toBeVisible();
      await expect(page.getByText('SSH Server Root')).toBeVisible();
    });

    test('should filter credentials by type', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.selectTypeFilter('database');

      await page.waitForTimeout(300);
      // Should show only database credentials
      await expect(page.getByText('Production Database')).toBeVisible();
    });

    test('should filter credentials by status', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.selectStatusFilter('active');

      await page.waitForTimeout(300);
      // Should show only active credentials
      await expect(page.getByText('active')).toBeVisible();
    });

    test('should search credentials', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.search('database');

      await page.waitForTimeout(500);
      await expect(page.getByText('Production Database')).toBeVisible();
    });

    test('should navigate to add credential page', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.clickAddCredential();

      await page.waitForURL('/credentials/new');
      await expect(page.getByRole('heading', { name: /add credential/i })).toBeVisible();
    });

    test('should click rotate button', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.clickRotateButton(0);

      // Should show success message
      await expect(page.getByText(/rotation/i)).toBeVisible({ timeout: 3000 });
    });

    test('should click view button', async ({ page }) => {
      const credentialsPage = new CredentialsPage(page);
      await credentialsPage.goto();
      await page.waitForLoadState('networkidle');

      await credentialsPage.clickViewButton(0);

      // Should navigate to credential detail page
      await page.waitForURL(/\/credentials\/.+/);
      await expect(page.getByRole('heading')).toBeVisible();
    });
  });

  test.describe('Credentials Detail Page', () => {
    test('should display credential details', async ({ page }) => {
      await page.goto('/credentials/cred-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Production Database')).toBeVisible();
      await expect(page.getByText('admin')).toBeVisible();
      await expect(page.getByText('db.example.com')).toBeVisible();
      await expect(page.getByText('5432')).toBeVisible();
    });

    test('should show credential actions', async ({ page }) => {
      await page.goto('/credentials/cred-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByRole('button', { name: /rotate/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /view password/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /edit/i })).toBeVisible();
    });

    test('should display credential activity', async ({ page }) => {
      await page.goto('/credentials/cred-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/activity/i)).toBeVisible();
      await expect(page.getByText(/access history/i)).toBeVisible();
    });
  });
});

// Setup API mocks for targets tests
const setupTargetsMocks = (page: any) => {
  page.route('**/api/v1/targets**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/targets - List targets
    if (url.includes('/api/v1/targets') && !url.includes('/connect') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'target-1',
              name: 'Web Server 01',
              description: 'Production web server',
              type: 'ssh',
              host: 'web01.example.com',
              port: 22,
              environment: 'production',
              status: 'online',
              platform: 'linux',
              last_seen: '2024-01-15T10:00:00Z',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
            {
              id: 'target-2',
              name: 'Database Server',
              description: 'Primary database',
              type: 'rdp',
              host: 'db.example.com',
              port: 3389,
              environment: 'production',
              status: 'offline',
              platform: 'windows',
              last_seen: '2024-01-14T16:00:00Z',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-14T16:00:00Z',
            },
          ],
          pagination: { total: 2, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    // POST /api/v1/targets/:id/connect - Connect to target
    if (url.includes('/connect') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          session_id: 'session-1',
          message: 'Connection initiated',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
});

test.describe('Targets Management', () => {
  test.beforeEach(async ({ page }) => {
    setupTargetsMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Targets List Page', () => {
    test('should display targets list page', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(targetsPage.heading).toBeVisible();
    });

    test('should display all targets', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Web Server 01')).toBeVisible();
      await expect(page.getByText('Database Server')).toBeVisible();
    });

    test('should filter targets by type', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.selectTypeFilter('ssh');

      await page.waitForTimeout(300);
      await expect(page.getByText('Web Server 01')).toBeVisible();
    });

    test('should filter targets by environment', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.selectEnvironmentFilter('production');

      await page.waitForTimeout(300);
      await expect(page.getByText('production')).toBeVisible();
    });

    test('should search targets', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.search('web');

      await page.waitForTimeout(500);
      await expect(page.getByText('Web Server 01')).toBeVisible();
    });

    test('should navigate to add target page', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.clickAddTarget();

      await page.waitForURL('/targets/new');
      await expect(page.getByRole('heading', { name: /add target/i })).toBeVisible();
    });

    test('should click connect button', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.clickConnectButton(0);

      // Should show connection initiated message
      await expect(page.getByText(/connection/i)).toBeVisible({ timeout: 3000 });
    });

    test('should click view button', async ({ page }) => {
      const targetsPage = new TargetsPage(page);
      await targetsPage.goto();
      await page.waitForLoadState('networkidle');

      await targetsPage.clickViewButton(0);

      await page.waitForURL(/\/targets\/.+/);
      await expect(page.getByRole('heading')).toBeVisible();
    });
  });

  test.describe('Targets Detail Page', () => {
    test('should display target details', async ({ page }) => {
      await page.goto('/targets/target-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Web Server 01')).toBeVisible();
      await expect(page.getByText('Production web server')).toBeVisible();
      await expect(page.getByText('web01.example.com')).toBeVisible();
      await expect(page.getByText('22')).toBeVisible();
    });

    test('should show target status', async ({ page }) => {
      await page.goto('/targets/target-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('online')).toBeVisible();
    });

    test('should show target actions', async ({ page }) => {
      await page.goto('/targets/target-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByRole('button', { name: /connect/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /edit/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /delete/i })).toBeVisible();
    });
  });
});

// Setup API mocks for roles tests
const setupRolesMocks = (page: any) => {
  page.route('**/api/v1/roles**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/roles - List roles
    if (url.includes('/api/v1/roles') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'role-1',
              name: 'System Administrators',
              description: 'Full system access',
              permissions: ['*'],
              user_count: 5,
              is_system_role: true,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
            {
              id: 'role-2',
              name: 'Database Access',
              description: 'Database server access',
              permissions: ['credential:checkout:prod-db-*', 'session:connect:db-*'],
              user_count: 15,
              is_system_role: false,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          ],
          pagination: { total: 2, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    // GET /api/v1/roles/:id - Get role detail
    if (url.includes('/api/v1/roles/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'role-1',
          name: 'System Administrators',
          description: 'Full system access',
          permissions: ['*'],
          user_count: 5,
          is_system_role: true,
          users: [
            { id: 'user-1', name: 'Admin User', email: 'admin@example.com' },
          ],
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-15T10:00:00Z',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

test.describe('Roles Management', () => {
  test.beforeEach(async ({ page }) => {
    setupRolesMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Roles List Page', () => {
    test('should display roles list page', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(rolesPage.heading).toBeVisible();
    });

    test('should display all roles', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('System Administrators')).toBeVisible();
      await expect(page.getByText('Database Access')).toBeVisible();
    });

    test('should search roles', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await rolesPage.search('admin');

      await page.waitForTimeout(500);
      await expect(page.getByText('System Administrators')).toBeVisible();
    });

    test('should navigate to add role page', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await rolesPage.clickAddRole();

      await page.waitForURL('/roles/new');
      await expect(page.getByRole('heading', { name: /add role/i })).toBeVisible();
    });

    test('should click view button', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await rolesPage.clickViewButton(0);

      await page.waitForURL(/\/roles\/.+/);
      await expect(page.getByRole('heading')).toBeVisible();
    });

    test('should click edit button via menu', async ({ page }) => {
      const rolesPage = new RolesPage(page);
      await rolesPage.goto();
      await page.waitForLoadState('networkidle');

      await rolesPage.clickEditButton(0);

      await page.waitForTimeout(500);
      // Should open edit menu
      expect(await page.locator('a').filter({ hasText: /edit/i }).count()).toBeGreaterThan(0);
    });
  });

  test.describe('Roles Detail Page', () => {
    test('should display role details', async ({ page }) => {
      await page.goto('/roles/role-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('System Administrators')).toBeVisible();
      await.expect(page.getByText('Full system access')).toBeVisible();
    });

    test('should display assigned users', async ({ page }) => {
      await page.goto('/roles/role-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/users/i)).toBeVisible();
      await expect(page.getByText('Admin User')).toBeVisible();
    });

    test('should display permissions', async ({ page }) => {
      await page.goto('/roles/role-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/permissions/i)).toBeVisible();
      await expect(page.getByText('*')).toBeVisible();
    });
  });
});

// Setup API mocks for tenants tests
const setupTenantsMocks = (page: any) => {
  page.route('**/api/v1/tenants**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/tenants - List tenants
    if (url.includes('/api/v1/tenants') && !url.includes('/api/v1/tenants/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'tenant-1',
              name: 'Acme Corp',
              slug: 'acme-corp',
              status: 'active',
              settings: {
                session_timeout_minutes: 60,
                mfa_required: true,
              },
              user_count: 150,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
            {
              id: 'tenant-2',
              name: 'Globex Inc',
              slug: 'globex-inc',
              status: 'active',
              settings: {
                session_timeout_minutes: 30,
                mfa_required: false,
              },
              user_count: 75,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          ],
          pagination: { total: 2, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    // GET /api/v1/tenants/:id - Get tenant detail
    if (url.includes('/api/v1/tenants/') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'tenant-1',
          name: 'Acme Corp',
          slug: 'acme-corp',
          status: 'active',
          settings: {
            session_timeout_minutes: 60,
            mfa_required: true,
            password_policy: {
              min_length: 12,
              require_uppercase: true,
              require_lowercase: true,
              require_numbers: true,
              require_special: true,
            },
          },
          user_count: 150,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-15T10:00:00Z',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

test.describe('Tenants Management', () => {
  test.beforeEach(async ({ page }) => {
    setupTenantsMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Tenants List Page', () => {
    test('should display tenants list page', async ({ page }) => {
      const tenantsPage = new TenantsPage(page);
      await tenantsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(tenantsPage.heading).toBeVisible();
    });

    test('should display all tenants', async ({ page }) => {
      const tenantsPage = new TenantsPage(page);
      await tenantsPage.goto();
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Acme Corp')).toBeVisible();
      await expect(page.getByText('Globex Inc')).toBeVisible();
    });

    test('should filter tenants by status', async ({ page }) => {
      const tenantsPage = new TenantsPage(page);
      await tenantsPage.goto();
      await page.waitForLoadState('networkidle');

      await tenantsPage.selectStatusFilter('active');

      await page.waitForTimeout(300);
      await expect(page.getByText('active')).toBeVisible();
    });

    test('should search tenants', async ({ page }) => {
      const tenantsPage = new TenantsPage(page);
      await tenantsPage.goto();
      await page.waitForLoadState('idle');

      await tenantsPage.search('acme');

      await page.waitForTimeout(500);
      await expect(page.getByText('Acme Corp')).toBeVisible();
    });

    test('should navigate to add tenant page', async ({ page }) => {
      const tenantsPage = new TenantsPage(page);
      await tenantsPage.goto();
      await page.waitForLoadState('networkidle');

      await tenantsPage.clickAddTenant();

      await page.waitForURL('/tenants/new');
      await expect(page.getByRole('heading', { name: /add tenant/i })).toBeVisible();
    });
  });

  test.describe('Tenants Detail Page', () => {
    test('should display tenant details', async ({ page }) => {
      await page.goto('/tenants/tenant-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Acme Corp')).toBeVisible();
      await expect(page.getByText('acme-corp')).toBeVisible();
    });

    test('should display tenant settings', async ({ page }) => {
      await page.goto('/tenants/tenant-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/settings/i)).toBeVisible();
      await expect(page.getByText(/session timeout/i)).toBeVisible();
    });

    test('should display tenant users count', async ({ page }) => {
      await page.goto('/tenants/tenant-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText(/150 users/i)).toBeVisible();
    });

    test('should display tenant actions', async ({ page }) => {
      await page.goto('/tenants/tenant-1');
      await page.waitForLoadState('networkidle');

      await expect(page.getByRole('button', { name: /edit/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /settings/i })).toBeVisible();
    });
  });
});

// Setup API mocks for password policies tests
const setupPasswordPoliciesMocks = (page: any) => {
  page.route('**/api/v1/policies/password**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/policies/password - List password policies
    if (url.includes('/api/v1/policies/password') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'policy-1',
              name: 'Default Password Policy',
              description: 'Organization-wide password requirements',
              is_default: true,
              settings: {
                min_length: 12,
                require_uppercase: true,
                require_lowercase: true,
                require_numbers: true,
                require_special: true,
                prevent_common: true,
                max_age_days: 90,
              },
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          ],
          pagination: { total: 1, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

test.describe('Password Policy Management', () => {
  test.beforeEach(async ({ page }) => {
    setupPasswordPoliciesMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Password Policies List Page', () => {
    test('should display password policies page', async ({ page }) => {
      const policiesPage = new PoliciesPage(page);
      await policiesPage.goto('password');
      await page.waitForLoadState('networkidle');

      await expect(policiesPage.heading).toBeVisible();
    });

    test('should display default password policy', async ({ page }) => {
      const policiesPage = new PoliciesPage(page);
      await policiesPage.goto('password');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Default Password Policy')).toBeVisible();
      await expect(page.getByText('Organization-wide password requirements')).toBeVisible();
    });
  });
});

// Setup API mocks for session policies tests
const setupSessionPoliciesMocks = (page: any) => {
  page.route('**/api/v1/policies/session**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    // GET /api/v1/policies/session - List session policies
    if (url.includes('/api/v1/policies/session') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'policy-1',
              name: 'Session Timeout Policy',
              description: 'Controls maximum session duration',
              is_default: true,
              settings: {
                max_duration_minutes: 60,
                terminate_on_timeout: true,
              },
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          ],
          pagination: { total: 1, offset: 0, limit: 20 },
        }),
      });
      return;
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
    });
  });
};

test.describe('Session Policy Management', () => {
  test.beforeEach(async ({ page }) => {
    setupSessionPoliciesMocks(page);
    const loginPage = new (require('./pages/LoginPage'))(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
    });
  });

  test.describe('Session Policies List Page', () => {
    test('should display session policies page', async ({ page }) => {
      const policiesPage = new PoliciesPage(page);
      await policiesPage.goto('session');
      await page.waitForLoadState('networkidle');

      await expect(policiesPage.heading).toBeVisible();
    });

    test('should display session timeout policy', async ({ page }) => {
      const policiesPage = new PoliciesPage(page);
      await policiesPage.goto('session');
      await page.waitForLoadState('networkidle');

      await expect(page.getByText('Session Timeout Policy')).toBeVisible();
      await expect(page.getByText('Controls maximum session duration')).toBeVisible();
    });
  });
});
