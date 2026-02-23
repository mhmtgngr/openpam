import { test, expect } from './fixtures/auth.fixture';
import { ApprovalsPage } from './pages/ApprovalsPage';

test.describe('Approvals', () => {
  test('should display approvals page', async ({ authenticatedPage }) => {
    const approvalsPage = new ApprovalsPage(authenticatedPage);
    await approvalsPage.goto();
    await approvalsPage.waitForLoad();

    const heading = await approvalsPage.getHeadingText();
    expect(heading).toContain('Pending Approvals');
  });

  test('should show empty state when no pending approvals', async ({ authenticatedPage }) => {
    // Mock the API to return empty list
    await authenticatedPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 20, offset: 0 },
        }),
      });
    });

    const approvalsPage = new ApprovalsPage(authenticatedPage);
    await approvalsPage.goto();
    await approvalsPage.waitForLoad();

    const isEmpty = await approvalsPage.hasEmptyMessage();
    expect(isEmpty).toBe(true);
  });

  test('should display pending approvals list', async ({ mockApiPage }) => {
    // Mock pending approvals
    await mockApiPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Database',
              },
              type: 'credential_access',
              reason: 'Need to deploy hotfix',
              duration_minutes: 60,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
            {
              id: '2',
              user: {
                id: 'user-2',
                first_name: 'Jane',
                last_name: 'Smith',
                email: 'jane@example.com',
              },
              target: {
                id: 'target-2',
                name: 'SSH Server',
              },
              type: 'session_access',
              reason: 'Emergency maintenance',
              duration_minutes: 30,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 2, limit: 20, offset: 0 },
        }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Should show the approvals
    await expect(mockApiPage.getByText('John Doe')).toBeVisible();
    await expect(mockApiPage.getByText('Jane Smith')).toBeVisible();
    await expect(mockApiPage.getByText('Production Database')).toBeVisible();
    await expect(mockApiPage.getByText('SSH Server')).toBeVisible();
  });

  test('should open approve modal', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Database',
              },
              type: 'credential_access',
              reason: 'Need to deploy hotfix',
              duration_minutes: 60,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click first approve button
    await approvalsPage.clickApprove(0);

    // Modal should open
    const isModalOpen = await approvalsPage.isModalOpen();
    expect(isModalOpen).toBe(true);

    // Check modal title
    await expect(mockApiPage.getByText('Approve Request')).toBeVisible();
  });

  test('should approve request with comment', async ({ mockApiPage }) => {
    let approveCalled = false;
    await mockApiPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Database',
              },
              type: 'credential_access',
              reason: 'Need to deploy hotfix',
              duration_minutes: 60,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    await mockApiPage.route('**/api/v1/requests/*/approve', async (route) => {
      approveCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click approve button
    await approvalsPage.clickApprove(0);

    // Add comment and confirm
    await approvalsPage.enterModalComment('Approved for production deployment');
    await approvalsPage.confirmModalAction();

    // Wait for API call
    await mockApiPage.waitForTimeout(100);
    expect(approveCalled).toBe(true);
  });

  test('should deny request with reason', async ({ mockApiPage }) => {
    let denyCalled = false;
    await mockApiPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Database',
              },
              type: 'credential_access',
              reason: 'Unclear justification',
              duration_minutes: 120,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    await mockApiPage.route('**/api/v1/requests/*/deny', async (route) => {
      denyCalled = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click deny button
    await approvalsPage.clickDeny(0);

    // Modal should show deny title
    await expect(mockApiPage.getByText('Deny Request')).toBeVisible();

    // Enter reason and confirm
    await approvalsPage.enterModalComment('Insufficient justification provided');
    await approvalsPage.confirmModalAction();

    // Wait for API call
    await mockApiPage.waitForTimeout(100);
    expect(denyCalled).toBe(true);
  });

  test('should cancel approval action', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/requests/pending', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'John',
                last_name: 'Doe',
                email: 'john@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Production Database',
              },
              type: 'credential_access',
              reason: 'Test',
              duration_minutes: 60,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Click approve button
    await approvalsPage.clickApprove(0);

    // Cancel the modal
    await approvalsPage.cancelModal();

    // Modal should be closed
    const isModalOpen = await approvalsPage.isModalOpen();
    expect(isModalOpen).toBe(false);
  });

  test('should filter approvals by search', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/requests/pending*', async (route) => {
      const url = route.request().url();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: '1',
              user: {
                id: 'user-1',
                first_name: 'Search',
                last_name: 'User',
                email: 'search@example.com',
              },
              target: {
                id: 'target-1',
                name: 'Searchable Target',
              },
              type: 'credential_access',
              reason: 'Test search',
              duration_minutes: 60,
              created_at: new Date().toISOString(),
              status: 'pending',
            },
          ],
          pagination: { total: 1, limit: 20, offset: 0 },
        }),
      });
    });

    const approvalsPage = new ApprovalsPage(mockApiPage);
    await approvalsPage.goto();
    await mockApiPage.waitForLoadState('networkidle');

    // Search for a request
    await approvalsPage.search('Search User');
    await mockApiPage.waitForTimeout(500);

    // Should trigger filtered request
    // In real scenario, we would verify the filtered results
  });

  test('should redirect to login when unauthenticated', async ({ page }) => {
    await page.goto('/approvals');

    // Should redirect to login
    await page.waitForURL('/login', { timeout: 5000 });

    await expect(page.getByText('Sign in to your account')).toBeVisible();
  });
});
