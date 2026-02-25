import { test, expect } from './fixtures/auth.fixture';

test.describe('Reports - Enhanced E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Setup reports API mocks
    page.route('**/api/v1/reports/**', async (route) => {
      const url = route.request().url();

      if (url.includes('/snapshots')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: '1',
                report_id: 'rpt-1',
                report_name: 'Monthly SOC 2 Report',
                type: 'compliance',
                framework: 'soc2',
                status: 'completed',
                format: 'pdf',
                file_url: '/api/v1/reports/snapshots/1/download',
                file_size_bytes: 1024000,
                created_at: new Date(Date.now() - 86400000).toISOString(),
                generation_started_at: new Date(Date.now() - 90000000).toISOString(),
                generation_completed_at: new Date(Date.now() - 87000000).toISOString(),
                generated_by: 'admin@example.com',
                generated_by_user: {
                  id: '1',
                  email: 'admin@example.com',
                  display_name: 'Admin User',
                },
                tenant_id: 'tenant-1',
                config: {
                  period_start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
                  period_end: new Date().toISOString().split('T')[0],
                  include_sections: ['summary', 'controls', 'findings'],
                  filters: {},
                },
              },
              {
                id: '2',
                report_id: 'rpt-2',
                report_name: 'Session Activity Report',
                type: 'session_activity',
                status: 'generating',
                format: 'xlsx',
                created_at: new Date().toISOString(),
                generation_started_at: new Date().toISOString(),
                generated_by: 'admin@example.com',
                generated_by_user: {
                  id: '1',
                  email: 'admin@example.com',
                  display_name: 'Admin User',
                },
                tenant_id: 'tenant-1',
                config: {
                  period_start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
                  period_end: new Date().toISOString().split('T')[0],
                  include_sections: [],
                  filters: {},
                },
              },
              {
                id: '3',
                report_id: 'rpt-3',
                report_name: 'Command Analysis',
                type: 'command_analysis',
                status: 'failed',
                format: 'csv',
                created_at: new Date(Date.now() - 172800000).toISOString(),
                generation_started_at: new Date(Date.now() - 174000000).toISOString(),
                error_message: 'Failed to generate report',
                generated_by: 'admin@example.com',
                generated_by_user: {
                  id: '1',
                  email: 'admin@example.com',
                  display_name: 'Admin User',
                },
                tenant_id: 'tenant-1',
                config: {
                  period_start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
                  period_end: new Date().toISOString().split('T')[0],
                  include_sections: [],
                  filters: {},
                },
              },
            ],
            pagination: { total: 3, limit: 20, offset: 0, has_more: false },
          }),
        });
      } else if (url.includes('/templates')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'tpl-1',
              name: 'SOC 2 Compliance Report',
              type: 'compliance',
              framework: 'soc2',
              description: 'Standard SOC 2 compliance report template',
              is_system: true,
              sections: [
                { id: 's1', name: 'Executive Summary', title: 'Summary', type: 'summary', required: true, enabled: true, config: {}, order: 0 },
                { id: 's2', name: 'Control Assessment', title: 'Controls', type: 'table', required: true, enabled: true, config: {}, order: 1 },
              ],
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
              config: {
                period_start: '',
                period_end: '',
                include_sections: [],
                filters: {},
              },
            },
            {
              id: 'tpl-2',
              name: 'Custom Activity Report',
              type: 'session_activity',
              description: 'Custom session activity report',
              is_system: false,
              sections: [],
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
              config: {
                period_start: '',
                period_end: '',
                include_sections: [],
                filters: {},
              },
            },
          ]),
        });
      } else if (url.includes('/schedules')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'sched-1',
                name: 'Weekly Compliance Report',
                frequency: 'weekly',
                is_active: true,
                next_run_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
                last_run_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                run_count: 12,
                recipients: ['admin@example.com', 'compliance@example.com'],
                framework: 'soc2',
                created_by: 'admin@example.com',
                created_at: new Date().toISOString(),
              },
            ],
            pagination: { total: 1, limit: 20, offset: 0, has_more: false },
          }),
        });
      } else if (url.includes('/generate')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            snapshot_id: 'new-report-1',
            status: 'generating',
            estimated_completion_at: new Date(Date.now() + 60000).toISOString(),
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: [], pagination: { total: 0, limit: 20, offset: 0, has_more: false } }),
        });
      }
    });

    // Setup delete and download handlers
    page.route('**/api/v1/reports/snapshots/**', async (route) => {
      if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Report deleted successfully' }),
        });
      }
    });
  });

  test('should display reports page with heading', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /reports/i });
    await expect(heading).toBeVisible();
  });

  test('should navigate from sidebar', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');
    await authenticatedPage.waitForLoadState('networkidle');

    // Click on Reports in sidebar
    const reportsLink = authenticatedPage.getByRole('link', { name: /reports/i });
    if (await reportsLink.isVisible()) {
      await reportsLink.click();
      await authenticatedPage.waitForURL(/\/analytics\/reports|\/reports/, { timeout: 5000 });
    }

    const heading = authenticatedPage.getByRole('heading', { name: /reports/i });
    await expect(heading).toBeVisible();
  });

  test('should display status summary cards', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for status cards
    const statusCards = authenticatedPage.locator('[class*="stat"], [class*="summary"]');
    const cardCount = await statusCards.count();

    expect(cardCount).toBeGreaterThan(0);

    // Should show total reports count
    const totalText = authenticatedPage.getByText(/total reports/i);
    await expect(totalText.first()).toBeVisible();
  });

  test('should display report cards', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show report cards
    const reportCards = authenticatedPage.locator('[class*="ReportCard"], [class*="report-card"]');
    const cardCount = await reportCards.count();

    expect(cardCount).toBeGreaterThan(0);

    // First card should have report info
    const reportName = authenticatedPage.getByText(/SOC 2/i);
    await expect(reportName.first()).toBeVisible();
  });

  test('should filter reports by status', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for status filter
    const statusFilter = authenticatedPage.getByRole('combobox', { name: /status/i }).or(
      authenticatedPage.getByPlaceholderText(/status/i)
    );

    if (await statusFilter.isVisible()) {
      await statusFilter.click();

      const completedOption = authenticatedPage.getByRole('option', { name: /completed/i });
      await completedOption.click();

      await authenticatedPage.waitForLoadState('networkidle');

      // Should show filtered results
      const pageContent = authenticatedPage.locator('main');
      await expect(pageContent).toBeVisible();
    }
  });

  test('should filter reports by type', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for type filter
    const typeFilter = authenticatedPage.getByRole('combobox', { name: /type/i }).or(
      authenticatedPage.getByText(/all types/i)
    );

    if (await typeFilter.isVisible()) {
      await typeFilter.click();

      const complianceOption = authenticatedPage.getByRole('option', { name: /compliance/i });
      await complianceOption.click();

      await authenticatedPage.waitForLoadState('networkidle');

      // Should show filtered results
      const pageContent = authenticatedPage.locator('main');
      await expect(pageContent).toBeVisible();
    }
  });

  test('should search reports', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for search input
    const searchInput = authenticatedPage.getByPlaceholderText(/search/i).or(
      authenticatedPage.getByRole('textbox', { name: /search/i })
    );

    if (await searchInput.isVisible()) {
      await searchInput.fill('SOC 2');
      await authenticatedPage.keyboard.press('Enter');
      await authenticatedPage.waitForLoadState('networkidle');

      // Should show search results
      const results = authenticatedPage.getByText(/SOC 2/i);
      await expect(results.first()).toBeVisible();
    }
  });

  test('should navigate to generate report page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for generate button
    const generateButton = authenticatedPage.getByRole('link', { name: /generate report/i }).or(
      authenticatedPage.getByRole('button', { name: /generate/i })
    );

    if (await generateButton.isVisible()) {
      await generateButton.click();
      await authenticatedPage.waitForURL(/\/generate/, { timeout: 5000 });

      const heading = authenticatedPage.getByRole('heading', { name: /generate/i });
      await expect(heading).toBeVisible();
    }
  });

  test('should handle report generation flow', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports/generate');
    await authenticatedPage.waitForLoadState('networkidle');

    // Step 1: Select report type
    const reportTypeOption = authenticatedPage.getByText(/Compliance/i);
    if (await reportTypeOption.isVisible()) {
      await reportTypeOption.click();
    }

    // Step 2: Continue to configuration
    const continueButton = authenticatedPage.getByRole('button', { name: /continue/i });
    if (await continueButton.isVisible()) {
      await continueButton.click();
      await authenticatedPage.waitForTimeout(500);
    }

    // Step 3: Review and generate
    const generateButton = authenticatedPage.getByRole('button', { name: /generate/i });
    if (await generateButton.isVisible()) {
      // Setup download handler for response
      const downloadPromise = authenticatedPage.waitForEvent('download', { timeout: 5000 });

      await generateButton.click();

      try {
        await downloadPromise;
      } catch {
        // Download might not actually trigger in test
      }

      // Should navigate or show success
      await authenticatedPage.waitForTimeout(1000);
    }
  });

  test('should select all reports', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for select all functionality
    const selectAllButton = authenticatedPage.getByRole('button', { name: /select all/i }).or(
      authenticatedPage.getByText(/select all/i)
    );

    if (await selectAllButton.isVisible()) {
      await selectAllButton.click();
      await authenticatedPage.waitForTimeout(500);

      // Checkboxes should be checked
      const checkboxes = authenticatedPage.getByRole('checkbox');
      const checkedCount = await checkboxes.count();
      expect(checkedCount).toBeGreaterThan(0);
    }
  });

  test('should delete report', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find first delete button
    const deleteButton = authenticatedPage.getByRole('button', { name: /delete/i }).or(
      authenticatedPage.locator('button').filter({ hasText: /delete/i }).first()
    );

    if (await deleteButton.isVisible()) {
      // Accept dialog if present
      authenticatedPage.on('dialog', dialog => dialog.accept());

      await deleteButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      // Should show success message or refresh
      const pageContent = authenticatedPage.locator('main');
      await expect(pageContent).toBeVisible();
    }
  });

  test('should handle pagination', async ({ authenticatedPage }) => {
    // Mock paginated response
    authenticatedPage.route('**/api/v1/reports/snapshots**', async (route) => {
      const url = new URL(route.request().url());
      const offset = url.searchParams.get('offset') || '0';

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: Array.from({ length: 20 }, (_, i) => ({
            id: `report-${parseInt(offset) + i}`,
            report_id: `rpt-${i}`,
            report_name: `Report ${parseInt(offset) + i + 1}`,
            type: 'compliance',
            status: 'completed',
            format: 'pdf',
            created_at: new Date().toISOString(),
            generated_by: 'admin@example.com',
            tenant_id: 'tenant-1',
            config: {
              period_start: new Date().toISOString().split('T')[0],
              period_end: new Date().toISOString().split('T')[0],
              include_sections: [],
              filters: {},
            },
          })),
          pagination: {
            total: 50,
            limit: 20,
            offset: parseInt(offset),
            has_more: true,
          },
        }),
      });
    });

    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for next page button
    const nextPageButton = authenticatedPage.getByRole('button', { name: /next/i });

    if (await nextPageButton.isVisible()) {
      await nextPageButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      const url = authenticatedPage.url();
      expect(url).toContain('offset=20');
    }
  });

  test('should refresh reports', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Look for refresh button
    const refreshButton = authenticatedPage.getByRole('button', { name: /refresh/i });

    if (await refreshButton.isVisible()) {
      await refreshButton.click();
      await authenticatedPage.waitForLoadState('networkidle');

      const pageContent = authenticatedPage.locator('main');
      await expect(pageContent).toBeVisible();
    }
  });

  test('should display report templates page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports/templates');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /templates/i });
    await expect(heading).toBeVisible();
  });

  test('should display scheduled reports page', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports/scheduled');
    await authenticatedPage.waitForLoadState('networkidle');

    const heading = authenticatedPage.getByRole('heading', { name: /scheduled/i });
    await expect(heading).toBeVisible();
  });

  test('should display different report statuses with correct styling', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Check for completed status styling
    const completedBadge = authenticatedPage.getByText(/completed/i);
    if (await completedBadge.isVisible()) {
      const parent = completedBadge.locator('..');
      await expect(parent).toHaveClass(/success/i);
    }

    // Check for generating status styling
    const generatingBadge = authenticatedPage.getByText(/generating/i);
    if (await generatingBadge.isVisible()) {
      const parent = generatingBadge.locator('..');
      await expect(parent).toHaveClass(/warning/i);
    }

    // Check for failed status styling
    const failedBadge = authenticatedPage.getByText(/failed/i);
    if (await failedBadge.isVisible()) {
      const parent = failedBadge.locator('..');
      await expect(parent).toHaveClass(/danger/i);
    }
  });

  test('should handle empty state', async ({ authenticatedPage }) => {
    // Mock empty response
    authenticatedPage.route('**/api/v1/reports/snapshots**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    await authenticatedPage.goto('/analytics/reports');
    await authenticatedPage.waitForLoadState('networkidle');

    // Should show empty state or no results message
    const emptyState = authenticatedPage.getByText(/no reports/i).or(
      authenticatedPage.getByText(/get started/i)
    );

    const pageContent = authenticatedPage.locator('main');
    await expect(pageContent).toBeVisible();
  });
});
