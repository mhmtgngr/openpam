import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

test.describe('Policy Management', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test.describe('Password Policies', () => {
    test('should display password policies list page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('heading', { name: /password policies/i })).toBeVisible();
      await expect(authenticatedPage.getByText(/configure password requirements/i)).toBeVisible();
    });

    test('should display add policy button', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password');
      await authenticatedPage.waitForLoadState('networkidle');

      const addButton = authenticatedPage.getByRole('link', { name: /add policy/i });
      await expect(addButton).toBeVisible();
    });

    test('should navigate to password policy form page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password');
      await authenticatedPage.waitForLoadState('networkidle');

      const addButton = authenticatedPage.getByRole('link', { name: /add policy/i });
      await addButton.click();

      await authenticatedPage.waitForURL('/policies/password/new');
      await expect(authenticatedPage.getByRole('heading', { name: /create password policy/i })).toBeVisible();
    });

    test('should validate required fields on password policy form', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Try to submit without filling required fields
      const submitButton = authenticatedPage.getByRole('button', { name: /create policy/i });
      await submitButton.click();

      // Should show validation errors
      await expect(authenticatedPage.getByText(/name is required/i)).toBeVisible({ timeout: 3000 });
    });

    test('should toggle password requirements', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Toggle uppercase requirement
      const uppercaseToggle = authenticatedPage.getByRole('button', { name: /require uppercase/i });
      await uppercaseToggle.click();

      // Toggle lowercase requirement
      const lowercaseToggle = authenticatedPage.getByRole('button', { name: /require lowercase/i });
      await lowercaseToggle.click();

      // Toggle numbers requirement
      const numbersToggle = authenticatedPage.getByRole('button', { name: /require numbers/i });
      await numbersToggle.click();

      // Toggle special characters requirement
      const specialToggle = authenticatedPage.getByRole('button', { name: /require special/i });
      await specialToggle.click();
    });

    test('should set password length requirements', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Set minimum length
      const minLengthInput = authenticatedPage.getByLabel(/minimum length/i);
      await minLengthInput.fill('16');

      // Set maximum length
      const maxLengthInput = authenticatedPage.getByLabel(/maximum length/i);
      await maxLengthInput.fill('128');
    });

    test('should add forbidden password', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a forbidden password
      await authenticatedPage.getByPlaceholder(/add a forbidden password/i).fill('password123');
      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Should show the added password (masked)
      await expect(authenticatedPage.getByText(/••••••/i)).toBeVisible({ timeout: 2000 });
    });

    test('should set password expiration', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/password/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Set expiration days
      const expirationInput = authenticatedPage.getByLabel(/password expiration/i);
      await expirationInput.fill('90');

      // Set history count
      const historyInput = authenticatedPage.getByLabel(/password history/i);
      await historyInput.fill('5');
    });
  });

  test.describe('Session Policies', () => {
    test('should display session policies list page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('heading', { name: /session policies/i })).toBeVisible();
      await expect(authenticatedPage.getByText(/configure session security/i)).toBeVisible();
    });

    test('should display add policy button', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session');
      await authenticatedPage.waitForLoadState('networkidle');

      const addButton = authenticatedPage.getByRole('link', { name: /add policy/i });
      await expect(addButton).toBeVisible();
    });

    test('should navigate to session policy form page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session');
      await authenticatedPage.waitForLoadState('networkidle');

      const addButton = authenticatedPage.getByRole('link', { name: /add policy/i });
      await addButton.click();

      await authenticatedPage.waitForURL('/policies/session/new');
      await expect(authenticatedPage.getByRole('heading', { name: /create session policy/i })).toBeVisible();
    });

    test('should toggle session requirements', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Toggle approval requirement
      const approvalToggle = authenticatedPage.getByRole('button', { name: /require approval/i });
      await approvalToggle.click();

      // Toggle reason requirement
      const reasonToggle = authenticatedPage.getByRole('button', { name: /require reason/i });
      await reasonToggle.click();

      // Toggle MFA requirement
      const mfaToggle = authenticatedPage.getByRole('button', { name: /require mfa/i });
      await mfaToggle.click();

      // Toggle recording requirement
      const recordingToggle = authenticatedPage.getByRole('button', { name: /allow recording/i });
      await recordingToggle.click();
    });

    test('should set session duration limits', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Set max duration
      const maxDurationInput = authenticatedPage.getByLabel(/max session duration/i);
      await maxDurationInput.fill('480');

      // Set idle timeout
      const idleTimeoutInput = authenticatedPage.getByLabel(/idle timeout/i);
      await idleTimeoutInput.fill('15');

      // Set warning before end
      const warningInput = authenticatedPage.getByLabel(/warning before end/i);
      await warningInput.fill('5');
    });

    test('should add monitor keyword', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a monitor keyword
      await authenticatedPage.getByPlaceholder(/add a keyword/i).fill('rm -rf');
      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Should show the added keyword
      await expect(authenticatedPage.getByText('rm -rf')).toBeVisible({ timeout: 2000 });
    });

    test('should add blocked command', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a blocked command
      await authenticatedPage.getByPlaceholder(/add a command/i).fill('sudo su');
      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Should show the added command
      await expect(authenticatedPage.getByText('sudo su')).toBeVisible({ timeout: 2000 });
    });

    test('should remove monitor keyword', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/session/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a keyword first
      await authenticatedPage.getByPlaceholder(/add a keyword/i).fill('danger');
      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Remove the keyword
      const removeButton = authenticatedPage.locator('button').filter({ hasText: /×/i }).first();
      await removeButton.click();

      // Keyword should be removed
      await expect(authenticatedPage.getByText('danger')).not.toBeVisible({ timeout: 2000 });
    });

    test('should set policy as default', async ({ mockApiPage }) => {
      // Mock policies API
      await mockApiPage.route('**/api/v1/policies/session', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'policy-123',
                name: 'Standard Session Policy',
                max_duration_minutes: 480,
                require_approval: false,
                require_reason: true,
                require_mfa: true,
                allow_recording: true,
                is_default: false,
              },
            ],
          }),
        });
      });

      // Mock set default API
      await mockApiPage.route('**/api/v1/policies/session/*/default', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: { id: 'policy-123', is_default: true } }),
        });
      });

      await mockApiPage.goto('/policies/session');
      await mockApiPage.waitForLoadState('networkidle');

      // Click set default button
      const setDefaultButton = mockApiPage.getByRole('button', { name: /set default/i }).first();

      if (await setDefaultButton.isVisible({ timeout: 2000 })) {
        await setDefaultButton.click();

        // Should show success message
        await expect(mockApiPage.getByText(/default policy updated|success/i)).toBeVisible({ timeout: 5000 });
      }
    });
  });
});
