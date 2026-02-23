import { Route } from '@playwright/test';

export const mockDashboardRoutes = async (route: Route) => {
  const url = route.request().url();

  // GET /api/v1/dashboard/stats
  if (url.includes('/api/v1/dashboard/stats') && route.request().method() === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        users: {
          total: 156,
          active: 142,
          new_this_month: 12,
        },
        targets: {
          total: 48,
          online: 42,
          offline: 6,
        },
        sessions: {
          active: 23,
          total_today: 89,
        },
        requests: {
          pending: 7,
          approved_today: 34,
        },
        credentials: {
          total: 234,
          expiring_soon: 12,
        },
      }),
    });
    return;
  }

  // GET /api/v1/dashboard/activity
  if (url.includes('/api/v1/dashboard/activity') && route.request().method() === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          {
            id: '1',
            message: 'User john.doe@example.com accessed target production-db',
            timestamp: new Date().toISOString(),
          },
          {
            id: '2',
            message: 'New user jane.smith@example.com created',
            timestamp: new Date(Date.now() - 3600000).toISOString(),
          },
          {
            id: '3',
            message: 'Credential rotation completed for target api-server',
            timestamp: new Date(Date.now() - 7200000).toISOString(),
          },
        ],
      }),
    });
    return;
  }

  // GET /api/v1/dashboard/pending-requests
  if (url.includes('/api/v1/dashboard/pending') && route.request().method() === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [],
        pagination: {
          total: 0,
          offset: 0,
          limit: 10,
        },
      }),
    });
    return;
  }

  // Default: return empty data for unhandled dashboard routes
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: {} }),
  });
};
