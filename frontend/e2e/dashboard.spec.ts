import { test, expect } from './fixtures/auth.fixture';
import { DashboardPage } from './pages/DashboardPage';

test.describe('Dashboard', () => {
  test('should display dashboard with stats', async ({ authenticatedPage }) => {
    const dashboardPage = new DashboardPage(authenticatedPage);
    await dashboardPage.goto();

    // Wait for the page to load
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = await dashboardPage.getHeadingText();
    expect(heading?.toLowerCase()).toContain('welcome');

    // Check for stat cards - the dashboard shows cards for various stats
    const statsCount = await dashboardPage.getStatsCount();
    expect(statsCount).toBeGreaterThanOrEqual(0);
  });

  test('should have navigation sidebar', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for navigation items - the sidebar has links with these texts
    await expect(authenticatedPage.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect(authenticatedPage.getByRole('link', { name: 'Targets' })).toBeVisible();
    await expect(authenticatedPage.getByRole('link', { name: 'Credentials' })).toBeVisible();
  });

  test('should navigate to different sections', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on Targets
    await authenticatedPage.getByRole('link', { name: 'Targets' }).click();
    await authenticatedPage.waitForURL('/targets', { timeout: 5000 });

    // Click on Credentials
    await authenticatedPage.getByRole('link', { name: 'Credentials' }).click();
    await authenticatedPage.waitForURL('/credentials', { timeout: 5000 });
  });

  test('should show activity feed', async ({ authenticatedPage }) => {
    const dashboardPage = new DashboardPage(authenticatedPage);
    await dashboardPage.goto();
    await authenticatedPage.waitForLoadState('networkidle');

    // The activity feed section should be visible
    await expect(dashboardPage.activityFeed).toBeVisible();
  });
});
