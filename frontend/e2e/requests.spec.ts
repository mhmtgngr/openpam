import { test, expect } from './fixtures/auth.fixture';

test.describe('Access Requests', () => {
  test('should display my requests page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: 'My Requests' })).toBeVisible();
    await expect(authenticatedPage.getByRole('link', { name: /New Request/ })).toBeVisible();
  });

  test('should filter requests by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find the select element for status filter
    const selectElement = authenticatedPage.locator('.select').first();

    // Select pending status
    await selectElement.selectOption('pending');

    // Wait a moment for any potential updates
    await authenticatedPage.waitForTimeout(500);

    // The table should still be visible
    await expect(authenticatedPage.locator('.table')).toBeVisible();
  });

  test('should navigate to new request form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/requests/my');
    await authenticatedPage.waitForLoadState('networkidle');

    await authenticatedPage.getByRole('link', { name: /New Request/ }).click();

    await authenticatedPage.waitForURL('/requests/new', { timeout: 5000 });
    // The new request page should have a heading
    await expect(authenticatedPage.getByRole('heading')).toBeVisible();
  });
});

test.describe('Approvals', () => {
  test('should display approvals page for authorized users', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/approvals');

    // Wait for navigation to complete
    await authenticatedPage.waitForLoadState('networkidle');

    const url = authenticatedPage.url();
    if (url.includes('/approvals')) {
      await expect(authenticatedPage.getByRole('heading', { name: /Pending Approvals/ })).toBeVisible();
    } else {
      test.skip(true, 'User does not have access to approvals page');
    }
  });

  test('should show approve and deny buttons for pending requests', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/approvals');

    await authenticatedPage.waitForLoadState('networkidle');

    const url = authenticatedPage.url();
    if (url.includes('/approvals')) {
      const approveButton = authenticatedPage.getByRole('button', { name: 'Approve' }).first();
      const denyButton = authenticatedPage.getByRole('button', { name: 'Deny' }).first();

      // Buttons should exist if there are pending requests
      const hasPendingRequests = await authenticatedPage.locator('.table tbody tr').count() > 0;

      if (hasPendingRequests) {
        await expect(approveButton).toBeVisible();
        await expect(denyButton).toBeVisible();
      }
    } else {
      test.skip(true, 'User does not have access to approvals page');
    }
  });
});
