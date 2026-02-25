/**
 * Scheduled Reports E2E Tests
 */

import { test, expect } from '@playwright/test';
import { LoginPage } from './pages';
import { ScheduledReportsPage } from './pages';

test.describe('Scheduled Reports', () => {
  let loginPage: LoginPage;
  let scheduledReportsPage: ScheduledReportsPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    scheduledReportsPage = new ScheduledReportsPage(page);

    // Mock schedules API routes
    page.route('**/api/v1/reports/schedules**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'sched-001',
                name: 'Weekly SOC 2 Compliance',
                framework: 'soc2',
                description: 'Weekly compliance report for SOC 2',
                frequency: 'weekly',
                cron_expression: '0 9 * * 1',
                timezone: 'America/New_York',
                next_run_at: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000).toISOString(),
                last_run_at: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
                is_active: true,
                recipients: ['admin@example.com', 'compliance@example.com'],
                distribution_config: {
                  enabled: true,
                  methods: [
                    { type: 'email', enabled: true, config: {} },
                  ],
                },
                report_config: {
                  period_start: '',
                  period_end: '',
                  include_sections: [],
                  filters: {},
                },
                created_by: 'admin@example.com',
                tenant_id: 'tenant-001',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                run_count: 24,
              },
              {
                id: 'sched-002',
                name: 'Monthly Anomaly Summary',
                framework: 'soc2',
                description: 'Monthly summary of detected anomalies',
                frequency: 'monthly',
                cron_expression: '0 8 1 * *',
                timezone: 'America/New_York',
                next_run_at: new Date(Date.now() + 15 * 24 * 60 * 60 * 1000).toISOString(),
                last_run_at: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000).toISOString(),
                is_active: false,
                recipients: ['security@example.com'],
                distribution_config: {
                  enabled: true,
                  methods: [],
                },
                report_config: {
                  period_start: '',
                  period_end: '',
                  include_sections: [],
                  filters: {},
                },
                created_by: 'admin@example.com',
                tenant_id: 'tenant-001',
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                run_count: 6,
              },
            ],
            pagination: {
              total: 2,
              limit: 10,
              offset: 0,
              has_more: false,
            },
          }),
        });
      } else if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'sched-new',
            name: 'New Scheduled Report',
            framework: 'soc2',
            frequency: 'daily',
            is_active: true,
            next_run_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
            created_at: new Date().toISOString(),
          }),
        });
      }
    });

    // Mock schedule operations
    page.route('**/api/v1/reports/schedules/**', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      if (url.includes('/pause') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ is_active: false }),
        });
      } else if (url.includes('/resume') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ is_active: true }),
        });
      } else if (url.includes('/run') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            job_id: 'job-001',
            status: 'running',
          }),
        });
      } else if (url.includes('/history') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'exec-001',
                report_id: 'sched-001',
                report_name: 'Weekly SOC 2 Compliance',
                scheduled_for: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
                executed_at: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
                status: 'completed',
                snapshot_id: 'snap-001',
              },
              {
                id: 'exec-002',
                report_id: 'sched-001',
                report_name: 'Weekly SOC 2 Compliance',
                scheduled_for: new Date(Date.now() - 12 * 24 * 60 * 60 * 1000).toISOString(),
                executed_at: new Date(Date.now() - 12 * 24 * 60 * 60 * 1000).toISOString(),
                status: 'completed',
                snapshot_id: 'snap-002',
              },
            ],
            pagination: { total: 2, limit: 5, offset: 0, has_more: false },
          }),
        });
      } else if (method === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Schedule deleted' }),
        });
      }
    });

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display scheduled reports page', async ({ page }) => {
    await scheduledReportsPage.goto();

    // Verify heading
    const heading = await scheduledReportsPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('scheduled reports');

    // Verify new schedule button
    await expect(scheduledReportsPage.newScheduleButton).toBeVisible();

    // Verify filters
    await expect(scheduledReportsPage.searchInput).toBeVisible();
    await expect(scheduledReportsPage.frameworkFilter).toBeVisible();
    await expect(scheduledReportsPage.statusFilter).toBeVisible();
  });

  test('should display all scheduled reports', async ({ page }) => {
    await scheduledReportsPage.goto();

    const count = await scheduledReportsPage.getScheduleCount();
    expect(count).toBeGreaterThan(0);

    // Check for specific schedules
    await expect(page.locator('text=Weekly SOC 2 Compliance')).toBeVisible();
    await expect(page.locator('text=Monthly Anomaly Summary')).toBeVisible();
  });

  test('should search schedules by name', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.searchSchedules('SOC 2');

    // Verify search input value
    await expect(scheduledReportsPage.searchInput).toHaveValue(/SOC 2/i);
  });

  test('should filter schedules by framework', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.filterByFramework('SOC 2');

    // Verify filter was applied
    await expect(scheduledReportsPage.frameworkFilter).toHaveValue(/soc2/i);
  });

  test('should filter schedules by status', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.filterByStatus('Active');

    // Verify filter was applied
    await expect(scheduledReportsPage.statusFilter).toHaveValue(/active/i);
  });

  test('should create new schedule', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.createNewSchedule();

    // Should show schedule dialog
    await expect(page.locator('.dialog, .modal, [role="dialog"]').or(page.locator('text=New Schedule'))).toBeVisible();
  });

  test('should edit existing schedule', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.editSchedule('Weekly SOC 2 Compliance');

    // Should show edit dialog
    await expect(page.locator('.dialog, .modal, [role="dialog"]').or(page.locator('text=Edit'))).toBeVisible();
  });

  test('should pause active schedule', async ({ page }) => {
    await scheduledReportsPage.goto();

    const initialStatus = page.locator('text=Weekly SOC 2 Compliance').locator('..').locator('svg, .pause, .play');

    await scheduledReportsPage.toggleScheduleStatus('Weekly SOC 2 Compliance');

    // Should update status
    await page.waitForTimeout(500);
    await expect(page.locator('text=paused').or(page.locator('.pause'))).toBeVisible();
  });

  test('should resume paused schedule', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.toggleScheduleStatus('Monthly Anomaly Summary');

    // Should update status
    await page.waitForTimeout(500);
    await expect(page.locator('text=active').or(page.locator('.play'))).toBeVisible();
  });

  test('should run schedule immediately', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.runScheduleNow('Weekly SOC 2 Compliance');

    // Should show success message
    await expect(page.locator('text=generation started').or(page.locator('text=Report generation'))).toBeVisible();
  });

  test('should delete schedule', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.deleteSchedule('Monthly Anomaly Summary');

    // Should show success message
    await expect(page.locator('text=deleted successfully').or(page.locator('text=Schedule deleted'))).toBeVisible();
  });

  test('should toggle schedule history', async ({ page }) => {
    await scheduledReportsPage.goto();

    const executions = await scheduledReportsPage.viewExecutionHistory('Weekly SOC 2 Compliance');

    // Should show executions
    expect(executions).toBeGreaterThan(0);
    await expect(page.locator('text=Recent Runs').or(page.locator('text=executions'))).toBeVisible();
  });

  test('should display schedule details correctly', async ({ page }) => {
    await scheduledReportsPage.goto();

    // Check for schedule details
    await expect(page.locator('text=/Next run/i')).toBeVisible();
    await expect(page.locator('text=/recipients/i')).toBeVisible();
    await expect(page.locator('text=/Run count/i')).toBeVisible();
  });

  test('should display execution history with status', async ({ page }) => {
    await scheduledReportsPage.goto();

    await scheduledReportsPage.toggleScheduleHistory('Weekly SOC 2 Compliance');

    // Check for execution status
    await expect(page.locator('text=completed').or(page.locator('.completed'))).toBeVisible();
  });

  test('should display frequency badges', async ({ page }) => {
    await scheduledReportsPage.goto();

    // Check for frequency indicators
    await expect(page.locator('text=Weekly').or(page.locator('text=weekly'))).toBeVisible();
    await expect(page.locator('text=Monthly').or(page.locator('text=monthly'))).toBeVisible();
  });

  test('should display framework badges', async ({ page }) => {
    await scheduledReportsPage.goto();

    // Check for framework indicators
    await expect(page.locator('text=SOC 2').or(page.locator('text=soc2'))).toBeVisible();
  });
});

test.describe('Scheduled Reports Dialog', () => {
  test('should create schedule with all options', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock APIs
    page.route('**/api/v1/reports/schedules**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, limit: 10, offset: 0, has_more: false } }),
      });
    });

    page.route('**/api/v1/reports/schedules', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'sched-new',
            name: 'Daily Security Report',
            framework: 'soc2',
            frequency: 'daily',
            is_active: true,
          }),
        });
      }
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/reports/scheduled');
    await page.click('button:has-text("New Schedule")');

    // Fill name
    await page.fill('input[name="name"], input[placeholder*="name" i]', 'Daily Security Report');

    // Select framework
    await page.selectOption('select:has-text("Framework")', 'soc2');

    // Select frequency
    await page.selectOption('select:has-text("Frequency")', 'daily');

    // Add recipients
    await page.fill('input[name="recipients"], input[placeholder*="email" i]', 'admin@example.com');

    // Save
    await page.click('button:has-text("Create"), button:has-text("Save")');

    // Should return to list
    await expect(page.locator('text=Scheduled Reports')).toBeVisible();
  });
});

test.describe('Scheduled Reports - Error Handling', () => {
  test('should handle API errors gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock error response
    page.route('**/api/v1/reports/schedules', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'Internal server error' } }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const scheduledReportsPage = new ScheduledReportsPage(page);
    await scheduledReportsPage.goto();

    // Should show error state
    await expect(page.locator('text=error').or(page.locator('text=failed'))).toBeVisible({ timeout: 10000 });
  });

  test('should handle empty state gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock empty response
    page.route('**/api/v1/reports/schedules', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, limit: 10, offset: 0, has_more: false } }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const scheduledReportsPage = new ScheduledReportsPage(page);
    await scheduledReportsPage.goto();

    // Should show empty state
    await expect(page.locator('text=No scheduled').or(page.locator('text=Create a schedule'))).toBeVisible();
  });

  test('should handle run failure gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock schedule list
    page.route('**/api/v1/reports/schedules**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'sched-001',
              name: 'Test Schedule',
              framework: 'soc2',
              frequency: 'daily',
              is_active: true,
              recipients: [],
              distribution_config: { enabled: true, methods: [] },
              report_config: { period_start: '', period_end: '', include_sections: [], filters: {} },
              created_by: 'admin@example.com',
              tenant_id: 'tenant-001',
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
              run_count: 0,
            },
          ],
          pagination: { total: 1, limit: 10, offset: 0, has_more: false },
        }),
      });
    });

    // Mock run failure
    page.route('**/api/v1/reports/schedules/*/run**', async (route) => {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'Cannot run: report configuration incomplete' } }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const scheduledReportsPage = new ScheduledReportsPage(page);
    await scheduledReportsPage.goto();

    await scheduledReportsPage.runScheduleNow('Test Schedule');

    // Should show error message
    await expect(page.locator('text=error').or(page.locator('text=failed').or(page.locator('text=incomplete'))).toBeVisible();
  });
});
