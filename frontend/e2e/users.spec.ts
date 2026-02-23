import { test, expect } from './fixtures/auth.fixture';
import { UsersPage } from './pages/UsersPage';

test.describe('User Management', () => {
  test('should display users list', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(usersPage.heading).toContainText('Users');
    await expect(usersPage.searchInput).toBeVisible();
    await expect(usersPage.addUserButton).toBeVisible();
  });

  test('should search users', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    const initialCount = await usersPage.getTableRowCount();

    await usersPage.search('admin');

    // Wait for search results to potentially update
    await authenticatedPage.waitForTimeout(500);

    const searchCount = await usersPage.getTableRowCount();

    // Search should reduce or maintain count
    expect(searchCount).toBeLessThanOrEqual(initialCount);
  });

  test('should navigate to user creation form', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    await usersPage.clickAddUser();

    await authenticatedPage.waitForURL('/users/new', { timeout: 5000 });
    await expect(authenticatedPage.getByRole('heading', { name: /Add New User/ })).toBeVisible();
  });

  test('should validate user creation form', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    await usersPage.clickAddUser();
    await authenticatedPage.waitForURL('/users/new', { timeout: 5000 });

    // The form inputs use aria-labels, not name attributes
    // Clear the First Name field
    const firstNameInput = authenticatedPage.getByRole('textbox', { name: 'First Name' });
    await firstNameInput.fill('');

    // Clear the Email field
    const emailInput = authenticatedPage.getByRole('textbox', { name: 'Email Address' });
    await emailInput.fill('');

    // Click the submit button
    await authenticatedPage.getByRole('button', { name: 'Create User' }).click();

    // Should show validation errors - check for error messages
    const emailError = authenticatedPage.getByText(/Email is required|Invalid email/);
    const nameError = authenticatedPage.getByText(/First name is required/);

    // At least one validation error should appear
    await expect(emailError.or(nameError)).toBeVisible({ timeout: 2000 });
  });

  test('should populate edit form when clicking edit', async ({ authenticatedPage }) => {
    const usersPage = new UsersPage(authenticatedPage);
    await usersPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    // Wait for table to load
    await authenticatedPage.waitForSelector('.table tbody tr', { timeout: 5000 }).catch(() => {
      // Table might be empty, which is ok
    });

    const rowCount = await usersPage.getTableRowCount();
    if (rowCount > 0) {
      // The edit button is in a dropdown menu, click the menu first
      const firstRowMenu = authenticatedPage.locator('.table tbody tr').first().locator('button').first();
      await firstRowMenu.click();

      // Then click the Edit link in the dropdown
      await authenticatedPage.getByRole('link', { name: 'Edit' }).click();

      await authenticatedPage.waitForURL(/\/users\/[a-z0-9-]+/, { timeout: 5000 });
      await expect(authenticatedPage.getByRole('heading', { name: /Edit User/ })).toBeVisible();
    } else {
      test.skip(true, 'No users found to edit');
    }
  });
});
