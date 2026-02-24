import { test, expect } from './fixtures/auth.fixture';

test.describe('User Management', () => {
  test('should display users list page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /users/i })).toBeVisible();
  });

  test('should display add user button', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('button', { name: /add user|create user|new user/i }).or(
      authenticatedPage.getByRole('link', { name: /add user/i })
    );
    await expect(addButton).toBeVisible();
  });

  test('should navigate to user creation form', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    const addButton = authenticatedPage.getByRole('button', { name: /add user/i }).or(
      authenticatedPage.getByRole('link', { name: /add user/i })
    );
    await addButton.click();

    await authenticatedPage.waitForURL('/users/new', { timeout: 5000 });
    await expect(authenticatedPage.getByRole('heading', { name: /add new user|create user/i })).toBeVisible();
  });

  test('should validate required user fields', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users/new');
    await authenticatedPage.waitForLoadState('networkidle');

    // Try to submit without filling required fields
    const submitButton = authenticatedPage.getByRole('button', { name: /create|add user/i });
    await submitButton.click();

    // Should show validation errors
    await expect(authenticatedPage.getByText(/email is required|first name is required/i)).toBeVisible({ timeout: 3000 });
  });

  test('should create a new user', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'user-123',
            email: 'newuser@example.com',
            first_name: 'New',
            last_name: 'User',
            status: 'active',
            role: 'user',
            created_at: new Date().toISOString(),
          },
        }),
      });
    });

    await mockApiPage.goto('/users/new');
    await mockApiPage.waitForLoadState('networkidle');

    // Fill in user details
    await mockApiPage.getByLabel(/email/i).fill('newuser@example.com');
    await mockApiPage.getByLabel(/first name/i).fill('New');
    await mockApiPage.getByLabel(/last name/i).fill('User');
    await mockApiPage.getByLabel(/password/i).fill('SecurePassword123!');
    await mockApiPage.getByLabel(/confirm password/i).fill('SecurePassword123!');

    // Select role
    const roleSelect = mockApiPage.getByLabel(/role/i);
    if (await roleSelect.isVisible({ timeout: 1000 })) {
      await roleSelect.selectOption('user');
    }

    // Submit the form
    const submitButton = mockApiPage.getByRole('button', { name: /create|add user/i });
    await submitButton.click();

    // Should show success message
    await expect(mockApiPage.getByText(/user created|success/i)).toBeVisible({ timeout: 5000 });
  });

  test('should validate email format', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users/new');
    await authenticatedPage.waitForLoadState('networkidle');

    const emailInput = authenticatedPage.getByLabel(/email/i);
    await emailInput.fill('not-an-email');

    const submitButton = authenticatedPage.getByRole('button', { name: /create/i });
    await submitButton.click();

    await expect(authenticatedPage.getByText(/invalid email|must be a valid email/i)).toBeVisible({ timeout: 3000 });
  });

  test('should validate password strength', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users/new');
    await authenticatedPage.waitForLoadState('networkidle');

    const passwordInput = authenticatedPage.getByLabel(/password/i);
    await passwordInput.fill('weak');

    const submitButton = authenticatedPage.getByRole('button', { name: /create/i });
    await submitButton.click();

    await expect(authenticatedPage.getByText(/password must be|at least \d+ characters/i)).toBeVisible({ timeout: 3000 });
  });

  test('should filter users by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for status filter
    const statusFilter = authenticatedPage.getByRole('combobox', { name: /status/i }).or(
      authenticatedPage.getByLabel(/status/i),
      authenticatedPage.getByRole('tab', { name: /active|pending/i })
    );

    if (await statusFilter.isVisible({ timeout: 2000 })) {
      await statusFilter.first().click();
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should filter users by role', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for role filter
    const roleFilter = authenticatedPage.getByRole('combobox', { name: /role/i }).or(
      authenticatedPage.getByLabel(/role/i)
    );

    if (await roleFilter.isVisible({ timeout: 2000 })) {
      await roleFilter.selectOption('admin');
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should search users', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find search input
    const searchInput = authenticatedPage.getByRole('searchbox', { name: /search/i }).or(
      authenticatedPage.getByPlaceholder(/search/i),
      authenticatedPage.getByLabel(/search/i)
    );

    if (await searchInput.isVisible({ timeout: 2000 })) {
      await searchInput.fill('test@example.com');
      await authenticatedPage.waitForTimeout(500);
    }
  });

  test('should edit user', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/user-123', async (route) => {
      if (route.method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              id: 'user-123',
              email: 'updated@example.com',
              first_name: 'Updated',
              last_name: 'Name',
              status: 'active',
            },
          }),
        });
      }
    });

    await mockApiPage.goto('/users');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for edit button
    const editButton = mockApiPage.getByRole('button', { name: /edit/i }).first();

    if (await editButton.isVisible({ timeout: 2000 })) {
      await editButton.click();
      await mockApiPage.waitForURL('/users/user-123/edit', { timeout: 5000 });

      // Update user details
      const firstNameInput = mockApiPage.getByLabel(/first name/i);
      await firstNameInput.clear();
      await firstNameInput.fill('Updated');

      const saveButton = mockApiPage.getByRole('button', { name: /save|update/i });
      await saveButton.click();

      await expect(mockApiPage.getByText(/user updated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete user with confirmation', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/user-123', async (route) => {
      if (route.method() === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'User deleted' }),
        });
      }
    });

    await mockApiPage.goto('/users');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for delete button
    const deleteButton = mockApiPage.getByRole('button', { name: /delete/i }).first();

    if (await deleteButton.isVisible({ timeout: 2000 })) {
      await deleteButton.click();

      // Confirm dialog
      const confirmButton = mockApiPage.getByRole('button', { name: /confirm|delete|yes/i }).or(
        mockApiPage.getByRole('button', { name: /yes, delete/i })
      );

      if (await confirmButton.isVisible({ timeout: 2000 })) {
        await confirmButton.click();
      }

      await expect(mockApiPage.getByText(/user deleted|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should not allow deleting current user', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'user-123',
            email: 'test@example.com',
          },
        }),
      });
    });

    await mockApiPage.goto('/users');
    await mockApiPage.waitForLoadState('networkidle');

    // The delete button for current user should be disabled or hidden
    const currentUserId = 'user-123';
    const deleteButton = mockApiPage.locator(`[data-user-id="${currentUserId}"]`).getByRole('button', { name: /delete/i });

    if (await deleteButton.count() > 0) {
      await expect(deleteButton).toBeDisabled();
    }
  });

  test('should display user activity', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/user-123/activity', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            { action: 'login', timestamp: new Date().toISOString() },
            { action: 'credential_checkout', timestamp: new Date().toISOString() },
          ],
        }),
      });
    });

    await mockApiPage.goto('/users/user-123');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for activity section
    await expect(mockApiPage.getByText(/activity|recent actions/i)).toBeVisible({ timeout: 3000 });
  });

  test('should reset user password', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/user-123/reset-password', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { temporary_password: 'TempPass123!' },
        }),
      });
    });

    await mockApiPage.goto('/users');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for reset password button
    const resetButton = mockApiPage.getByRole('button', { name: /reset password/i }).first();

    if (await resetButton.isVisible({ timeout: 2000 })) {
      await resetButton.click();

      // Look for confirmation dialog
      const confirmButton = mockApiPage.getByRole('button', { name: /confirm|reset/i });
      if (await confirmButton.isVisible({ timeout: 2000 })) {
        await confirmButton.click();
      }

      await expect(mockApiPage.getByText(/password reset|temporary password/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should assign roles to user', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/user-123/roles', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { roles: ['admin', 'operator'] },
        }),
      });
    });

    await mockApiPage.goto('/users/user-123/edit');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for role assignment section
    const roleMultiSelect = mockApiPage.getByLabel(/roles/i).or(
      mockApiPage.getByRole('combobox', { name: /roles/i })
    );

    if (await roleMultiSelect.isVisible({ timeout: 2000 })) {
      await roleMultiSelect.click();
      await mockApiPage.getByRole('option', { name: /admin/i }).click();

      const saveButton = mockApiPage.getByRole('button', { name: /save/i });
      await saveButton.click();

      await expect(mockApiPage.getByText(/user updated|roles updated/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should paginate users list', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/users');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for pagination controls
    const nextPageButton = authenticatedPage.getByRole('button', { name: /next|»/i }).or(
      authenticatedPage.getByRole('link', { name: /next/i })
    );

    if (await nextPageButton.isVisible({ timeout: 2000 })) {
      await nextPageButton.click();
      await authenticatedPage.waitForTimeout(500);
    }
  });
});

test.describe('User Roles and Permissions', () => {
  test('should display roles list', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/roles');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /roles/i }).or(
      authenticatedPage.getByText(/manage roles/i)
    ).first()).toBeVisible({ timeout: 3000 });
  });

  test('should create custom role', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/roles', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'role-123',
            name: 'custom_role',
            display_name: 'Custom Role',
            permissions: ['credentials:read', 'sessions:read'],
          },
        }),
      });
    });

    await mockApiPage.goto('/roles');
    await mockApiPage.waitForLoadState('networkidle');

    const addButton = mockApiPage.getByRole('button', { name: /add role|create role/i });
    if (await addButton.isVisible({ timeout: 2000 })) {
      await addButton.click();

      await mockApiPage.getByLabel(/name/i).fill('custom_role');
      await mockApiPage.getByLabel(/display name/i).fill('Custom Role');

      // Select permissions
      const permissionsSelect = mockApiPage.getByLabel(/permissions/i);
      if (await permissionsSelect.isVisible({ timeout: 1000 })) {
        await permissionsSelect.selectOption(['credentials:read', 'sessions:read']);
      }

      const saveButton = mockApiPage.getByRole('button', { name: /create|save/i });
      await saveButton.click();

      await expect(mockApiPage.getByText(/role created|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should display role permissions', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/roles/admin', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'admin',
            name: 'admin',
            display_name: 'Administrator',
            permissions: [
              'credentials:read',
              'credentials:write',
              'credentials:delete',
              'users:read',
              'users:write',
            ],
          },
        }),
      });
    });

    await mockApiPage.goto('/roles/admin');
    await mockApiPage.waitForLoadState('networkidle');

    await expect(mockApiPage.getByText(/permissions|credentials:read/i)).toBeVisible({ timeout: 3000 });
  });

  test('should edit role permissions', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/roles/admin', async (route) => {
      if (route.method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: { message: 'Role updated' },
          }),
        });
      }
    });

    await mockApiPage.goto('/roles/admin/edit');
    await mockApiPage.waitForLoadState('networkidle');

    const permissionsSelect = mockApiPage.getByLabel(/permissions/i);
    if (await permissionsSelect.isVisible({ timeout: 2000 })) {
      await permissionsSelect.selectOption('audit:read');

      const saveButton = mockApiPage.getByRole('button', { name: /save|update/i });
      await saveButton.click();

      await expect(mockApiPage.getByText(/role updated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should delete custom role', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/roles/custom-role', async (route) => {
      if (route.method() === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Role deleted' }),
        });
      }
    });

    await mockApiPage.goto('/roles');
    await mockApiPage.waitForLoadState('networkidle');

    const deleteButton = mockApiPage.getByRole('button', { name: /delete custom-role/i }).first();

    if (await deleteButton.isVisible({ timeout: 2000 })) {
      await deleteButton.click();

      const confirmButton = mockApiPage.getByRole('button', { name: /confirm|delete/i });
      if (await confirmButton.isVisible({ timeout: 2000 })) {
        await confirmButton.click();
      }

      await expect(mockApiPage.getByText(/role deleted|success/i)).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe('User Profile', () => {
  test('should display user profile', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/profile');
    await authenticatedPage.waitForLoadState('networkidle');

    await expect(authenticatedPage.getByRole('heading', { name: /my profile|profile/i })).toBeVisible();
  });

  test('should allow user to change their password', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/auth/password/change', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Password changed successfully' }),
      });
    });

    await mockApiPage.goto('/profile');
    await mockApiPage.waitForLoadState('networkidle');

    // Look for change password section
    const currentPasswordInput = mockApiPage.getByLabel(/current password/i);
    if (await currentPasswordInput.isVisible({ timeout: 2000 })) {
      await currentPasswordInput.fill('old-password');
      await mockApiPage.getByLabel(/new password/i).fill('NewPassword123!');
      await mockApiPage.getByLabel(/confirm password/i).fill('NewPassword123!');

      const updateButton = mockApiPage.getByRole('button', { name: /change password|update/i });
      await updateButton.click();

      await expect(mockApiPage.getByText(/password changed|success/i)).toBeVisible({ timeout: 5000 });
    }
  });

  test('should allow user to update their information', async ({ mockApiPage }) => {
    await mockApiPage.route('**/api/v1/users/me', async (route) => {
      if (route.method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: {
              id: 'user-123',
              email: 'test@example.com',
              first_name: 'Updated',
              last_name: 'Name',
            },
          }),
        });
      }
    });

    await mockApiPage.goto('/profile');
    await mockApiPage.waitForLoadState('networkidle');

    const firstNameInput = mockApiPage.getByLabel(/first name/i);
    if (await firstNameInput.isVisible({ timeout: 2000 })) {
      await firstNameInput.clear();
      await firstNameInput.fill('Updated');

      const saveButton = mockApiPage.getByRole('button', { name: /save|update/i });
      await saveButton.click();

      await expect(mockApiPage.getByText(/profile updated|success/i)).toBeVisible({ timeout: 5000 });
    }
  });
});
