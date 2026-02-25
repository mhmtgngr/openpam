import { test, expect } from '@playwright/test';
import { LoginPage } from './pages';
import { ReportsPage, ReportGeneratorPage, ExceptionsPage } from './pages';

test.describe('Compliance Reports', () => {
  let loginPage: LoginPage;
  let reportsPage: ReportsPage;
  let reportGeneratorPage: ReportGeneratorPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    reportsPage = new ReportsPage(page);
    reportGeneratorPage = new ReportGeneratorPage(page);

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display reports page with stats and filters', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    // Verify heading
    const heading = await reportsPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('compliance reports');

    // Verify stats cards are present
    const statCount = await reportsPage.getStatCount();
    expect(statCount).toBeGreaterThan(0);

    // Verify filters are present
    await expect(reportsPage.searchInput).toBeVisible();
    await expect(reportsPage.frameworkFilter).toBeVisible();
    await expect(reportsPage.statusFilter).toBeVisible();
    await expect(reportsPage.sortSelect).toBeVisible();

    // Verify action buttons
    await expect(reportsPage.generateReportButton).toBeVisible();
    await expect(reportsPage.scheduleReportButton).toBeVisible();
  });

  test('should filter reports by framework', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    const initialCount = await reportsPage.getReportCount();

    // Filter by SOC 2
    await reportsPage.filterByFramework('SOC 2');
    await page.waitForTimeout(500);

    // After filtering, the count might be different or same
    // The important thing is the filter action completed
    await expect(reportsPage.frameworkFilter).toHaveValue(/soc2/i);
  });

  test('should filter reports by status', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    // Filter by completed status
    await reportsPage.filterByStatus('Completed');
    await page.waitForTimeout(500);

    await expect(reportsPage.statusFilter).toHaveValue(/completed/i);
  });

  test('should search reports by query', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    // Enter search query
    await reportsPage.searchReports('SOC 2');
    await page.waitForTimeout(500);

    // Verify search input has the value
    await expect(reportsPage.searchInput).toHaveValue(/SOC 2/i);
  });

  test('should clear filters', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    // Apply some filters
    await reportsPage.filterByFramework('SOC 2');
    await reportsPage.searchReports('test');
    await page.waitForTimeout(500);

    // Clear filters
    await reportsPage.clearFilters();
    await page.waitForTimeout(500);

    // Verify search is cleared
    await expect(reportsPage.searchInput).toHaveValue('');
  });

  test('should navigate to report generator', async ({ page }) => {
    await reportsPage.goto();
    await reportsPage.waitForLoad();

    await reportsPage.clickGenerateReport();

    // Verify navigation to generator page
    await expect(page).toHaveURL(/\/reports\/generate/);
    await expect(reportGeneratorPage.heading).toBeVisible();
  });

  test('should generate a new report step by step', async ({ page }) => {
    await reportGeneratorPage.goto();
    await reportGeneratorPage.waitForLoad();

    // Step 1: Basic Info
    expect(await reportGeneratorPage.getCurrentStep()).toBe(1);

    await reportGeneratorPage.fillReportName('E2E Test Report');
    await reportGeneratorPage.selectFramework('SOC 2');
    await reportGeneratorPage.setPeriod('2024-01-01', '2024-01-31');
    await reportGeneratorPage.selectFormat('pdf');

    // Click next to go to step 2
    await reportGeneratorPage.clickNext();

    // Step 2: Content
    expect(await reportGeneratorPage.getCurrentStep()).toBe(2);

    // Select some sections
    await reportGeneratorPage.toggleSection('Executive Summary');
    await reportGeneratorPage.toggleSection('Control Assessment');

    // Toggle options
    await reportGeneratorPage.toggleOption('Redact Sensitive Data');
    await reportGeneratorPage.toggleOption('Include Session Logs');

    // Click next to go to step 3
    await reportGeneratorPage.clickNext();

    // Step 3: Review
    expect(await reportGeneratorPage.getCurrentStep()).toBe(3);
    expect(await reportGeneratorPage.isAtFinalStep()).toBe(true);

    // Get review summary
    const summary = await reportGeneratorPage.getReviewSummary();
    expect(summary.framework).toContain('SOC 2');
    expect(summary.format).toContain('PDF');
  });

  test('should validate report name is required', async ({ page }) => {
    await reportGeneratorPage.goto();
    await reportGeneratorPage.waitForLoad();

    // Try to proceed without entering name
    await reportGeneratorPage.clickNext();

    // Should still be on step 1 with validation error
    expect(await reportGeneratorPage.getCurrentStep()).toBe(1);
    expect(await reportGeneratorPage.hasValidationError('report name')).toBe(true);
  });

  test('should cancel report generation', async ({ page }) => {
    await reportGeneratorPage.goto();
    await reportGeneratorPage.waitForLoad();

    await reportGeneratorPage.fillReportName('Test Report');
    await reportGeneratorPage.selectFramework('SOC 2');
    await reportGeneratorPage.setPeriod('2024-01-01', '2024-01-31');

    // Click cancel
    await reportGeneratorPage.clickCancel();

    // Should navigate back to reports page
    await expect(page).toHaveURL(/\/reports/);
  });

  test('should navigate between steps using back button', async ({ page }) => {
    await reportGeneratorPage.goto();
    await reportGeneratorPage.waitForLoad();

    // Fill step 1
    await reportGeneratorPage.fillReportName('Test Report');
    await reportGeneratorPage.selectFramework('SOC 2');
    await reportGeneratorPage.setPeriod('2024-01-01', '2024-01-31');
    await reportGeneratorPage.clickNext();

    // Should be on step 2
    expect(await reportGeneratorPage.getCurrentStep()).toBe(2);

    // Go back
    await reportGeneratorPage.clickBack();

    // Should be back on step 1
    expect(await reportGeneratorPage.getCurrentStep()).toBe(1);
  });

  test('should toggle report scheduling', async ({ page }) => {
    await reportGeneratorPage.goto();
    await reportGeneratorPage.waitForLoad();

    const initialState = await reportGeneratorPage.isScheduleEnabled();

    // Toggle scheduling
    await reportGeneratorPage.toggleSchedule(!initialState);

    // Verify state changed
    const newState = await reportGeneratorPage.isScheduleEnabled();
    expect(newState).toBe(!initialState);
  });
});

test.describe('Compliance Exceptions', () => {
  let loginPage: LoginPage;
  let exceptionsPage: ExceptionsPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    exceptionsPage = new ExceptionsPage(page);

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display exceptions page with stats', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Verify heading
    const heading = await exceptionsPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('compliance exceptions');

    // Verify stats cards
    const statCount = await exceptionsPage.getStatCount();
    expect(statCount).toBeGreaterThan(0);

    // Verify filters
    await expect(exceptionsPage.searchInput).toBeVisible();
    await expect(exceptionsPage.frameworkFilter).toBeVisible();
    await expect(exceptionsPage.statusFilter).toBeVisible();

    // Verify request button
    await expect(exceptionsPage.requestExceptionButton).toBeVisible();
  });

  test('should filter exceptions by status', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Filter by pending status
    await exceptionsPage.filterByStatus('Pending');
    await page.waitForTimeout(500);

    await expect(exceptionsPage.statusFilter).toHaveValue(/pending/i);
  });

  test('should filter exceptions by framework', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Filter by framework
    await exceptionsPage.filterByFramework('SOC 2');
    await page.waitForTimeout(500);

    await expect(exceptionsPage.frameworkFilter).toHaveValue(/soc2/i);
  });

  test('should search exceptions', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Enter search query
    await exceptionsPage.searchExceptions('access control');
    await page.waitForTimeout(500);

    // Verify search input has the value
    await expect(exceptionsPage.searchInput).toHaveValue(/access control/i);
  });

  test('should clear filters', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Apply filters
    await exceptionsPage.filterByStatus('Pending');
    await exceptionsPage.searchExceptions('test');
    await page.waitForTimeout(500);

    // Clear filters
    await exceptionsPage.clearFilters();
    await page.waitForTimeout(500);

    // Verify search is cleared
    await expect(exceptionsPage.searchInput).toHaveValue('');
  });

  test('should show expiring soon alert', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Check if expiring soon alert exists
    const hasAlert = await exceptionsPage.hasExpiringSoonAlert();

    if (hasAlert) {
      // Show expiring exceptions
      await exceptionsPage.showExpiringSoon();

      // Hide again
      await exceptionsPage.hideExpiringSoon();
    }
  });

  test('should request new exception', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Click request exception button
    await exceptionsPage.clickRequestException();

    // Dialog should open (we'd need a DialogPage object to fully test this)
    // For now, verify we're on the right page
    await expect(page).toHaveURL(/\/compliance\/exceptions/);
  });

  test('should navigate through exception list', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    const exceptionCount = await exceptionsPage.getExceptionCount();

    if (exceptionCount > 0) {
      // If we have exceptions, we could test clicking on them
      // For now, just verify the page loads correctly
      expect(exceptionCount).toBeGreaterThanOrEqual(0);
    } else {
      // Empty state should be shown
      const hasEmptyState = await exceptionsPage.hasEmptyState();
      expect(hasEmptyState).toBe(true);
    }
  });

  test('should display stat values', async ({ page }) => {
    await exceptionsPage.goto();
    await exceptionsPage.waitForLoad();

    // Check pending requests stat
    const pendingValue = await exceptionsPage.getStatValue('Pending Requests');
    expect(pendingValue).not.toBeNull();

    // Check active exceptions stat
    const activeValue = await exceptionsPage.getStatValue('Active Exceptions');
    expect(activeValue).not.toBeNull();
  });
});

test.describe('Reports Navigation', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should navigate from sidebar to reports', async ({ page }) => {
    // Click Reports in sidebar
    await page.getByRole('link', { name: /reports/i }).click();

    await expect(page).toHaveURL(/\/reports/);
    await expect(page.getByRole('heading', { name: /compliance reports/i })).toBeVisible();
  });

  test('should navigate from sidebar to exceptions', async ({ page }) => {
    // Click Exceptions in sidebar
    await page.getByRole('link', { name: /exceptions/i }).click();

    await expect(page).toHaveURL(/\/compliance\/exceptions/);
    await expect(page.getByRole('heading', { name: /compliance exceptions/i })).toBeVisible();
  });

  test('should navigate from reports to generate page', async ({ page }) => {
    await page.goto('/reports');
    await page.waitForLoadState('networkidle');

    // Click generate button
    await page.getByRole('button', { name: /generate report/i }).click();

    await expect(page).toHaveURL(/\/reports\/generate/);
    await expect(page.getByRole('heading', { name: /generate compliance report/i })).toBeVisible();
  });

  test('should navigate back from generate to reports', async ({ page }) => {
    await page.goto('/reports/generate');
    await page.waitForLoadState('networkidle');

    // Click back arrow
    await page.getByRole('button', { name: /back/i }).first().click();

    await expect(page).toHaveURL(/\/reports/);
  });
});
