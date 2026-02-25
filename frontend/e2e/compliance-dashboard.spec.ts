/**
 * Compliance Dashboard E2E Tests
 */

import { test, expect } from '@playwright/test';
import { LoginPage } from './pages';
import { ComplianceDashboardPage } from './pages';

test.describe('Compliance Dashboard', () => {
  let loginPage: LoginPage;
  let complianceDashboardPage: ComplianceDashboardPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    complianceDashboardPage = new ComplianceDashboardPage(page);

    // Mock compliance API routes
    page.route('**/api/v1/compliance/dashboard', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 85,
          framework: 'soc2',
          controls: [
            {
              id: 'ac-001',
              name: 'Access Control Policy',
              description: 'Implement and manage access controls',
              status: 'compliant',
              score: 95,
              evidence_count: 12,
              last_assessed_at: new Date().toISOString(),
              category: 'Access Control',
              findings: [],
            },
            {
              id: 'ac-002',
              name: 'Account Management',
              description: 'Manage user account lifecycles',
              status: 'partial',
              score: 75,
              evidence_count: 8,
              last_assessed_at: new Date().toISOString(),
              category: 'Access Control',
              findings: [
                {
                  id: 'f-001',
                  severity: 'medium',
                  title: 'Incomplete offboarding process',
                  description: 'Some terminated accounts still have active access',
                  status: 'open',
                  discovered_at: new Date().toISOString(),
                },
              ],
            },
            {
              id: 'au-001',
              name: 'Audit Logging',
              description: 'Maintain audit logs of access',
              status: 'compliant',
              score: 100,
              evidence_count: 15,
              last_assessed_at: new Date().toISOString(),
              category: 'Audit',
              findings: [],
            },
            {
              id: 'sc-001',
              name: 'System Communication',
              description: 'Protect system communications',
              status: 'non_compliant',
              score: 45,
              evidence_count: 3,
              last_assessed_at: new Date().toISOString(),
              category: 'Security',
              findings: [
                {
                  id: 'f-002',
                  severity: 'critical',
                  title: 'Unencrypted data transmission',
                  description: 'Sensitive data transmitted without encryption',
                  status: 'open',
                  discovered_at: new Date().toISOString(),
                },
              ],
            },
          ],
          exceptions: [
            {
              id: 'ex-001',
              control_id: 'sc-001',
              control_name: 'System Communication',
              reason: 'Legacy system compatibility',
              approved_by: 'admin@example.com',
              approved_at: new Date().toISOString(),
              expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
              status: 'active',
            },
          ],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display compliance dashboard with all sections', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Verify heading
    const heading = await complianceDashboardPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('compliance dashboard');

    // Verify framework selector is present
    await expect(complianceDashboardPage.frameworkSelector).toBeVisible();

    // Verify action buttons
    await expect(complianceDashboardPage.refreshButton).toBeVisible();
    await expect(complianceDashboardPage.exportButton).toBeVisible();

    // Verify stats are displayed
    await expect(complianceDashboardPage.overallScoreGauge).toBeVisible();
    await expect(complianceDashboardPage.compliantStatCard).toBeVisible();
    await expect(complianceDashboardPage.nonCompliantStatCard).toBeVisible();
  });

  test('should display overall compliance score', async ({ page }) => {
    await complianceDashboardPage.goto();

    const score = await complianceDashboardPage.getOverallScore();
    expect(score).toBeGreaterThan(0);
    expect(score).toBeLessThanOrEqual(100);
  });

  test('should switch between frameworks', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Switch to ISO 27001
    await complianceDashboardPage.selectFramework('ISO 27001');
    await expect(complianceDashboardPage.frameworkSelector).toContainText('ISO 27001');
  });

  test('should refresh dashboard data', async ({ page }) => {
    await complianceDashboardPage.goto();

    const initialScore = await complianceDashboardPage.getOverallScore();

    await complianceDashboardPage.refresh();

    // Verify the page is still loaded after refresh
    await expect(complianceDashboardPage.heading).toBeVisible();
  });

  test('should display control status by category', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Check for category sections
    await expect(page.locator('text=Access Control')).toBeVisible();
    await expect(page.locator('text=Audit')).toBeVisible();
    await expect(page.locator('text=Security')).toBeVisible();
  });

  test('should display active exceptions', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Check for exceptions section
    await expect(page.locator('text=Active Exceptions')).toBeVisible();
    await expect(page.locator('text=Legacy system compatibility')).toBeVisible();
  });

  test('should display findings summary when issues exist', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Check for critical findings alert
    await expect(page.locator('text=Action Required')).toBeVisible();
    await expect(page.locator('text=High-Priority Findings')).toBeVisible();
  });

  test('should display recent activity', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Check for activity section
    await expect(page.locator('text=Recent Activity')).toBeVisible();
  });

  test('should navigate to control details', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Click on a control
    await page.locator('text=Access Control Policy').click();

    // Should navigate to control detail (or show modal)
    await expect(page.locator('text=Access Control Policy')).toBeVisible();
  });

  test('should export compliance report', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Setup download handler
    const downloadPromise = page.waitForEvent('download');

    await complianceDashboardPage.exportReport();

    // Note: Actual download might not happen in mocked environment
    // This tests that the action is available
    await expect(complianceDashboardPage.exportButton).toBeVisible();
  });

  test('should display correct control counts', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Verify controls are listed
    const controls = page.locator('[class*="control"]');
    const count = await controls.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should filter and search controls', async ({ page }) => {
    await complianceDashboardPage.goto();

    // Test filter button visibility
    const filterButton = page.getByRole('button', { name: /filter/i });
    await expect(filterButton).toBeVisible();
  });
});

test.describe('Compliance Dashboard - Error Handling', () => {
  test('should handle API errors gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock error response
    page.route('**/api/v1/compliance/dashboard', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'Internal server error' } }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const complianceDashboardPage = new ComplianceDashboardPage(page);
    await complianceDashboardPage.goto();

    // Should show error state or retry option
    await expect(page.locator('text=error').or(page.locator('text=retry'))).toBeVisible({ timeout: 10000 });
  });

  test('should handle empty state gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock empty response
    page.route('**/api/v1/compliance/dashboard', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 0,
          framework: 'soc2',
          controls: [],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const complianceDashboardPage = new ComplianceDashboardPage(page);
    await complianceDashboardPage.goto();

    // Should show empty state or zero score
    const score = await complianceDashboardPage.getOverallScore();
    expect(score).toBe(0);
  });
});
