import { Route } from '@playwright/test';

const mockUsers = [
  {
    id: '1',
    email: 'admin@example.com',
    first_name: 'Admin',
    last_name: 'User',
    role: 'admin',
    status: 'active',
    mfa_enabled: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    last_login_at: new Date().toISOString(),
  },
  {
    id: '2',
    email: 'user@example.com',
    first_name: 'Regular',
    last_name: 'User',
    role: 'user',
    status: 'active',
    mfa_enabled: false,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    last_login_at: new Date(Date.now() - 86400000).toISOString(),
  },
];

export const mockUserRoutes = async (route: Route) => {
  const url = route.request().url();

  // GET /api/v1/users
  if (url.includes('/api/v1/users') && route.request().method() === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: mockUsers,
        pagination: {
          total: mockUsers.length,
          offset: 0,
          limit: 20,
        },
      }),
    });
    return;
  }

  // POST /api/v1/users - Create user
  if (url.includes('/api/v1/users') && route.request().method() === 'POST') {
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          id: '3',
          email: 'new@example.com',
          first_name: 'New',
          last_name: 'User',
          role: 'user',
          status: 'active',
          mfa_enabled: false,
          created_at: new Date().toISOString(),
        },
      }),
    });
    return;
  }

  // Default: return empty array for unhandled user routes
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
  });
};
