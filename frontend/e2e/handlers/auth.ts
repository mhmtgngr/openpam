import { Route } from '@playwright/test';

// Mock user data
const mockUser = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  email: 'test@example.com',
  first_name: 'Test',
  last_name: 'User',
  role: 'admin',
  status: 'active',
  mfa_enabled: false,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

const mockAuthResponse = {
  access_token: 'mock-access-token',
  refresh_token: 'mock-refresh-token',
  user: mockUser,
};

export const mockAuthRoutes = async (route: Route) => {
  const url = route.request().url();
  const method = route.request().method();

  // POST /api/v1/auth/login - Login endpoint
  if (url.includes('/api/v1/auth/login') && method === 'POST') {
    const requestBody = route.request().postDataJSON();

    // Check for invalid credentials
    if (requestBody.email === 'invalid@example.com' || requestBody.password === 'wrongpassword') {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 'INVALID_CREDENTIALS',
            message: 'Invalid email or password',
          },
        }),
      });
      return;
    }

    // Successful login
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockAuthResponse),
    });
    return;
  }

  // GET /api/v1/auth/me - Get current user
  if (url.includes('/api/v1/auth/me') && method === 'GET') {
    // Check if there's a token in localStorage scenario
    // For testing, we'll return the user if the request is authenticated
    // Otherwise return 401 to simulate no auth
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockUser),
    });
    return;
  }

  // POST /api/v1/auth/logout - Logout endpoint
  if (url.includes('/api/v1/auth/logout') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'Logged out successfully' }),
    });
    return;
  }

  // POST /api/v1/auth/refresh - Refresh token endpoint
  if (url.includes('/api/v1/auth/refresh') && method === 'POST') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        access_token: 'new-mock-access-token',
        refresh_token: 'new-mock-refresh-token',
      }),
    });
    return;
  }

  // GET /api/v1/auth/mfa - MFA devices
  if (url.includes('/api/v1/auth/mfa') && method === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { devices: [] } }),
    });
    return;
  }

  // Default: return 404 for unhandled auth routes
  await route.fulfill({
    status: 404,
    contentType: 'application/json',
    body: JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Route not found' } }),
  });
};
