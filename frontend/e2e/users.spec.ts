import { test, expect } from './fixtures/auth.fixture';
import { UsersPage } from './pages/UsersPage';

test.describe('User Management', () => {
  test('should display users list', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();

    await expect(usersPage.heading).toHaveText('Users');
    await expect(usersPage.searchInput).toBeVisible();
    await expect(usersPage.addUserButton).toBeVisible();
  });

  test('should search users', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();

    const initialCount = await usersPage.getTableRowCount();

    await usersPage.search('admin');

    // Wait for search results
    await authenticatedPage.waitForTimeout(500);

    const searchCount = await usersPage.getTableRowCount();

    // Search should reduce or maintain count
    expect(searchCount).toBeLessThanOrEqual(initialCount);
  });

  test('should navigate to user creation form', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();

    await usersPage.clickAddUser();

    await authenticatedPage.waitForURL('/users/new');
    await expect(authenticatedPage.locator('text=Add New User')).toBeVisible();
  });

  test('should validate user creation form', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await usersPage.clickAddUser();

    // Try to submit without filling required fields
    await authenticatedPage.click('button:has-text("Create User")');

    // Should show validation errors
    await expect(authenticatedPage.locator('text=Email is required')).toBeVisible();
    await expect(authenticatedPage.locator('text=First name is required')).toBeVisible();
  });

  test('should populate edit form when clicking edit', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();

    // Wait for table to load
    await authenticatedPage.waitForSelector('.table tbody tr');

    const rowCount = await usersPage.getTableRowCount();
    if (rowCount > 0) {
      // Click edit on first user
      await authenticatedPage.locator('.table tbody tr').first().locator('button:has-text("Edit")').click();

      await authenticatedPage.waitForURL(/\/users\/[a-z0-9-]+/);
      await expect(authenticatedPage.locator('text=Edit User')).toBeVisible();
    }
  });
});
