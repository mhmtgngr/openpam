import { Route } from '@playwright/test';

export const mockRequestRoutes = async (route: Route) => {
  const url = route.request().url();

  // GET /api/v1/requests/my
  if (url.includes('/api/v1/requests/my') && route.request().method() === 'GET') {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [],
        pagination: {
          total: 0,
          offset: 0,
          limit: 20,
        },
      }),
    });
    return;
  }

  // GET /api/v1/approvals/pending
  if (url.includes('/api/v1/approvals') || url.includes('/approvals/pending')) {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [],
        pagination: {
          total: 0,
          offset: 0,
          limit: 20,
        },
      }),
    });
    return;
  }

  // POST /api/v1/requests - Create request
  if (url.includes('/api/v1/requests') && route.request().method() === 'POST') {
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          id: '1',
          status: 'pending',
          created_at: new Date().toISOString(),
        },
      }),
    });
    return;
  }

  // Default: return empty data for unhandled request routes
  await route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: [], pagination: { total: 0, offset: 0, limit: 20 } }),
  });
};
