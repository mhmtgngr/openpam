import { test, expect } from './fixtures/auth.fixture';
import { DashboardPage } from './pages/DashboardPage';

test.describe('Dashboard', () => {
  test('should display dashboard with stats', async ({ authenticatedPage }) => {
    const dashboardPage = new DashboardPage(authenticatedPage);
    await dashboardPage.goto();

    const heading = await dashboardPage.getHeadingText();
    expect(heading).toContain('Welcome');

    // Check for stat cards
    const statsCount = await dashboardPage.getStatsCount();
    expect(statsCount).toBeGreaterThanOrEqual(4);
  });

  test('should have navigation sidebar', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');

    // Check for navigation items
    await expect(authenticatedPage.locator('text=Dashboard')).toBeVisible();
    await expect(authenticatedPage.locator('text=Targets')).toBeVisible();
    await expect(authenticatedPage.locator('text=Credentials')).toBeVisible();
  });

  test('should navigate to different sections', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');

    // Click on Targets
    await authenticatedPage.click('text=Targets');
    await authenticatedPage.waitForURL('/targets');

    // Click on Credentials
    await authenticatedPage.click('a:has-text("Credentials")');
    await authenticatedPage.waitForURL('/credentials');
  });

  test('should show activity feed', async ({ authenticatedPage }) => {
    const dashboardPage = new DashboardPage(authenticatedPage);
    await dashboardPage.goto();

    await expect(dashboardPage.activityFeed).toBeVisible();
  });
});
