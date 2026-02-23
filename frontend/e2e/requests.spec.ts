import { test, expect } from './fixtures/auth.fixture';

test.describe('Access Requests', () => {
  test('should display my requests page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');

    await expect(authenticatedPage.locator('h1:has-text("My Requests")')).toBeVisible();
    await expect(authenticatedPage.locator('text=New Request')).toBeVisible();
  });

  test('should filter requests by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');

    // Click on status filter
    await authenticatedPage.selectOption('select', 'pending');

    // Should reload with filter
    await authenticatedPage.waitForTimeout(500);
    await expect(authenticatedPage.locator('.table')).toBeVisible();
  });

  test('should navigate to new request form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.click('button:has-text("New Request")');

    await authenticatedPage.waitForURL('/requests/new');
    await expect(authenticatedPage.locator('text=New Access Request')).toBeVisible();
  });
});

test.describe('Approvals', () => {
  test('should display approvals page for authorized users', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/approvals');

    // May be redirected if not authorized
    const url = authenticatedPage.url();
    if (url.includes('/approvals')) {
      await expect(authenticatedPage.locator('h1:has-text("Pending Approvals")')).toBeVisible();
    }
  });

  test('should show approve and deny buttons for pending requests', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/approvals');

    const url = authenticatedPage.url();
    if (url.includes('/approvals')) {
      const approveButton = authenticatedPage.locator('button:has-text("Approve")').first();
      const denyButton = authenticatedPage.locator('button:has-text("Deny")').first();

      // Buttons should exist if there are pending requests
      const hasPendingRequests = await authenticatedPage.locator('.table tbody tr').count() > 0;

      if (hasPendingRequests) {
        await expect(approveButton).toBeVisible();
        await expect(denyButton).toBeVisible();
      }
    }
  });
});
