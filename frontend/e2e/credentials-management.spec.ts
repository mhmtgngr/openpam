import { test, expect } from './fixtures/auth.fixture';
import { CredentialsPage } from './pages/CredentialsPage';
import { PoliciesPage } from './pages/PoliciesPage';
import { RolesPage } from './pages/RolesPage';
import { TargetsPage } from './pages/TargetsPage';
import { TenantsPage } from './pages/TenantsPage';

test.describe('Credentials Management', () => {
  test.describe('Credentials List Page', () => {
    test('should display credentials list page', async ({ authenticatedPage }) => {
      // Mock the API to return credentials list
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await expect(credentialsPage.heading).toBeVisible();
    });

    test('should display all credentials', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await expect(authenticatedPage.getByText('Production Database')).toBeVisible();
      await expect(authenticatedPage.getByText('SSH Server Root')).toBeVisible();
    });

    test('should filter credentials by type', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.selectTypeFilter('database');
      await authenticatedPage.waitForTimeout(300);

      await expect(authenticatedPage.getByText('Production Database')).toBeVisible();
    });

    test('should filter credentials by status', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.selectStatusFilter('active');
      await authenticatedPage.waitForTimeout(300);

      await expect(authenticatedPage.getByText('active')).toBeVisible();
    });

    test('should search credentials', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.search('database');
      await authenticatedPage.waitForTimeout(500);

      await expect(authenticatedPage.getByText('Production Database')).toBeVisible();
    });

    test('should navigate to add credential page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              data: [],
              pagination: { total: 0, offset: 0, limit: 20 },
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.clickAddCredential();
      await authenticatedPage.waitForURL('/credentials/new');

      await expect(authenticatedPage.getByRole('heading', { name: /add credential/i })).toBeVisible();
    });

    test('should click rotate button', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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
              ],
              pagination: { total: 1, offset: 0, limit: 20 },
            }),
          });
          return;
        }
        if (route.request().url().includes('/rotate')) {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.clickRotateButton(0);
      await expect(authenticatedPage.getByText(/rotation/i)).toBeVisible({ timeout: 3000 });
    });

    test('should click view button', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/rotate')) {
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

      const credentialsPage = new CredentialsPage(authenticatedPage);
      await credentialsPage.goto();
      await credentialsPage.waitForLoad();

      await credentialsPage.clickViewButton(0);
      await authenticatedPage.waitForURL(/\/credentials\/.+/);

      await expect(authenticatedPage.getByRole('heading')).toBeVisible();
    });
  });

  test.describe('Credentials Detail Page', () => {
    test('should display credential details', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials/cred-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/credentials/cred-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('Production Database')).toBeVisible();
      await expect(authenticatedPage.getByText('admin')).toBeVisible();
      await expect(authenticatedPage.getByText('db.example.com')).toBeVisible();
      await expect(authenticatedPage.getByText('5432')).toBeVisible();
    });

    test('should show credential actions', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials/cred-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/credentials/cred-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('button', { name: /rotate/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /view password/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /edit/i })).toBeVisible();
    });

    test('should display credential activity', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/credentials/cred-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/credentials/cred-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/activity/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/access history/i)).toBeVisible();
    });
  });
});

test.describe('Targets Management', () => {
  test.describe('Targets List Page', () => {
    test('should display targets list page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await expect(targetsPage.heading).toBeVisible();
    });

    test('should display all targets', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await expect(authenticatedPage.getByText('Web Server 01')).toBeVisible();
      await expect(authenticatedPage.getByText('Database Server')).toBeVisible();
    });

    test('should filter targets by type', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.selectTypeFilter('ssh');
      await authenticatedPage.waitForTimeout(300);

      await expect(authenticatedPage.getByText('Web Server 01')).toBeVisible();
    });

    test('should filter targets by environment', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.selectEnvironmentFilter('production');
      await authenticatedPage.waitForTimeout(300);

      await expect(authenticatedPage.getByText('production')).toBeVisible();
    });

    test('should search targets', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.search('web');
      await authenticatedPage.waitForTimeout(500);

      await expect(authenticatedPage.getByText('Web Server 01')).toBeVisible();
    });

    test('should navigate to add target page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              data: [],
              pagination: { total: 0, offset: 0, limit: 20 },
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.clickAddTarget();
      await authenticatedPage.waitForURL('/targets/new');

      await expect(authenticatedPage.getByRole('heading', { name: /add target/i })).toBeVisible();
    });

    test('should click connect button', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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
              ],
              pagination: { total: 1, offset: 0, limit: 20 },
            }),
          });
          return;
        }
        if (route.request().url().includes('/connect')) {
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.clickConnectButton(0);
      await expect(authenticatedPage.getByText(/connection/i)).toBeVisible({ timeout: 3000 });
    });

    test('should click view button', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/connect')) {
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

      const targetsPage = new TargetsPage(authenticatedPage);
      await targetsPage.goto();
      await targetsPage.waitForLoad();

      await targetsPage.clickViewButton(0);
      await authenticatedPage.waitForURL(/\/targets\/.+/);

      await expect(authenticatedPage.getByRole('heading')).toBeVisible();
    });
  });

  test.describe('Targets Detail Page', () => {
    test('should display target details', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets/target-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/targets/target-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('Web Server 01')).toBeVisible();
      await expect(authenticatedPage.getByText('Production web server')).toBeVisible();
      await expect(authenticatedPage.getByText('web01.example.com')).toBeVisible();
      await expect(authenticatedPage.getByText('22')).toBeVisible();
    });

    test('should show target status', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets/target-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/targets/target-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('online')).toBeVisible();
    });

    test('should show target actions', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/targets/target-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/targets/target-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('button', { name: /connect/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /edit/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /delete/i })).toBeVisible();
    });
  });
});

test.describe('Roles Management', () => {
  test.describe('Roles List Page', () => {
    test('should display roles list page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await expect(rolesPage.heading).toBeVisible();
    });

    test('should display all roles', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await expect(authenticatedPage.getByText('System Administrators')).toBeVisible();
      await expect(authenticatedPage.getByText('Database Access')).toBeVisible();
    });

    test('should search roles', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await rolesPage.search('admin');
      await authenticatedPage.waitForTimeout(500);

      await expect(authenticatedPage.getByText('System Administrators')).toBeVisible();
    });

    test('should navigate to add role page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              data: [],
              pagination: { total: 0, offset: 0, limit: 20 },
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

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await rolesPage.clickAddRole();
      await authenticatedPage.waitForURL('/roles/new');

      await expect(authenticatedPage.getByRole('heading', { name: /add role/i })).toBeVisible();
    });

    test('should click view button', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await rolesPage.clickViewButton(0);
      await authenticatedPage.waitForURL(/\/roles\/.+/);

      await expect(authenticatedPage.getByRole('heading')).toBeVisible();
    });

    test('should click edit button via menu', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const rolesPage = new RolesPage(authenticatedPage);
      await rolesPage.goto();
      await rolesPage.waitForLoad();

      await rolesPage.clickEditButton(0);
      await authenticatedPage.waitForTimeout(500);

      expect(await authenticatedPage.locator('a').filter({ hasText: /edit/i }).count()).toBeGreaterThan(0);
    });
  });

  test.describe('Roles Detail Page', () => {
    test('should display role details', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles/role-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
            },
          }),
        });
      });

      await authenticatedPage.goto('/roles/role-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('System Administrators')).toBeVisible();
      await expect(authenticatedPage.getByText('Full system access')).toBeVisible();
    });

    test('should display assigned users', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles/role-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
            },
          }),
        });
      });

      await authenticatedPage.goto('/roles/role-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/users/i)).toBeVisible();
      await expect(authenticatedPage.getByText('Admin User')).toBeVisible();
    });

    test('should display permissions', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/roles/role-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              id: 'role-1',
              name: 'System Administrators',
              description: 'Full system access',
              permissions: ['*'],
              user_count: 5,
              is_system_role: true,
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-15T10:00:00Z',
            },
          }),
        });
      });

      await authenticatedPage.goto('/roles/role-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/permissions/i)).toBeVisible();
      await expect(authenticatedPage.getByText('*')).toBeVisible();
    });
  });
});

test.describe('Tenants Management', () => {
  test.describe('Tenants List Page', () => {
    test('should display tenants list page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/api/v1/tenants/')) {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const tenantsPage = new TenantsPage(authenticatedPage);
      await tenantsPage.goto();
      await tenantsPage.waitForLoad();

      await expect(tenantsPage.heading).toBeVisible();
    });

    test('should display all tenants', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/api/v1/tenants/')) {
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
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: {}, pagination: { total: 0, offset: 0, limit: 20 } }),
        });
      });

      const tenantsPage = new TenantsPage(authenticatedPage);
      await tenantsPage.goto();
      await tenantsPage.waitForLoad();

      await expect(authenticatedPage.getByText('Acme Corp')).toBeVisible();
      await expect(authenticatedPage.getByText('Globex Inc')).toBeVisible();
    });

    test('should filter tenants by status', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/api/v1/tenants/')) {
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

      const tenantsPage = new TenantsPage(authenticatedPage);
      await tenantsPage.goto();
      await tenantsPage.waitForLoad();

      await tenantsPage.selectStatusFilter('active');
      await authenticatedPage.waitForTimeout(300);

      await expect(authenticatedPage.getByText('active')).toBeVisible();
    });

    test('should search tenants', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants**', async (route) => {
        if (route.request().method() === 'GET' && !route.request().url().includes('/api/v1/tenants/')) {
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

      const tenantsPage = new TenantsPage(authenticatedPage);
      await tenantsPage.goto();
      await tenantsPage.waitForLoad();

      await tenantsPage.search('acme');
      await authenticatedPage.waitForTimeout(500);

      await expect(authenticatedPage.getByText('Acme Corp')).toBeVisible();
    });

    test('should navigate to add tenant page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants**', async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              data: [],
              pagination: { total: 0, offset: 0, limit: 20 },
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

      const tenantsPage = new TenantsPage(authenticatedPage);
      await tenantsPage.goto();
      await tenantsPage.waitForLoad();

      await tenantsPage.clickAddTenant();
      await authenticatedPage.waitForURL('/tenants/new');

      await expect(authenticatedPage.getByRole('heading', { name: /add tenant/i })).toBeVisible();
    });
  });

  test.describe('Tenants Detail Page', () => {
    test('should display tenant details', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants/tenant-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
            },
          }),
        });
      });

      await authenticatedPage.goto('/tenants/tenant-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('Acme Corp')).toBeVisible();
      await expect(authenticatedPage.getByText('acme-corp')).toBeVisible();
    });

    test('should display tenant settings', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants/tenant-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/tenants/tenant-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/settings/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/session timeout/i)).toBeVisible();
    });

    test('should display tenant users count', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants/tenant-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/tenants/tenant-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/150 users/i)).toBeVisible();
    });

    test('should display tenant actions', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/tenants/tenant-1', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
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
          }),
        });
      });

      await authenticatedPage.goto('/tenants/tenant-1');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('button', { name: /edit/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /settings/i })).toBeVisible();
    });
  });
});

test.describe('Password Policy Management', () => {
  test.describe('Password Policies List Page', () => {
    test('should display password policies page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/policies/password**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const policiesPage = new PoliciesPage(authenticatedPage);
      await policiesPage.goto('password');
      await policiesPage.waitForLoad();

      await expect(policiesPage.heading).toBeVisible();
    });

    test('should display default password policy', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/policies/password**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const policiesPage = new PoliciesPage(authenticatedPage);
      await policiesPage.goto('password');
      await policiesPage.waitForLoad();

      await expect(authenticatedPage.getByText('Default Password Policy')).toBeVisible();
      await expect(authenticatedPage.getByText('Organization-wide password requirements')).toBeVisible();
    });
  });
});

test.describe('Session Policy Management', () => {
  test.describe('Session Policies List Page', () => {
    test('should display session policies page', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/policies/session**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const policiesPage = new PoliciesPage(authenticatedPage);
      await policiesPage.goto('session');
      await policiesPage.waitForLoad();

      await expect(policiesPage.heading).toBeVisible();
    });

    test('should display session timeout policy', async ({ authenticatedPage }) => {
      await authenticatedPage.route('**/api/v1/policies/session**', async (route) => {
        if (route.request().method() === 'GET') {
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

      const policiesPage = new PoliciesPage(authenticatedPage);
      await policiesPage.goto('session');
      await policiesPage.waitForLoad();

      await expect(authenticatedPage.getByText('Session Timeout Policy')).toBeVisible();
      await expect(authenticatedPage.getByText('Controls maximum session duration')).toBeVisible();
    });
  });
});
