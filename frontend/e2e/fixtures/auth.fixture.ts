import { test as base } from '@playwright/test';

export const test = base.extend<{
  authenticatedPage: typeof base.prototype['page'];
}>({
  authenticatedPage: async ({ page }, use) => {
    // Login with test credentials
    await page.goto('/login');
    await page.fill('input[type="email"]', 'test@example.com');
    await page.fill('input[type="password"]', 'testpassword123');
    await page.click('button[type="submit"]');

    // Wait for navigation to dashboard
    await page.waitForURL('/dashboard');

    await use(page);
  },
});

export const expect = test.expect;
