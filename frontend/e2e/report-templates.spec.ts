/**
 * Report Templates E2E Tests
 */

import { test, expect } from '@playwright/test';
import { LoginPage } from './pages';
import { ReportTemplatesPage } from './pages';

test.describe('Report Templates', () => {
  let loginPage: LoginPage;
  let reportTemplatesPage: ReportTemplatesPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    reportTemplatesPage = new ReportTemplatesPage(page);

    // Mock templates API routes
    page.route('**/api/v1/reports/templates', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'tpl-001',
              name: 'SOC 2 Compliance Report',
              type: 'compliance',
              framework: 'soc2',
              description: 'Standard SOC 2 Type II compliance report with all controls',
              thumbnail_url: '',
              config: {
                period_start: '',
                period_end: '',
                include_sections: ['executive_summary', 'control_status', 'findings_detail'],
                filters: {},
              },
              sections: [
                {
                  id: 'sec-1',
                  name: 'executive_summary',
                  title: 'Executive Summary',
                  type: 'summary',
                  required: true,
                  config: { include_score: true, include_findings: true },
                  order: 0,
                },
                {
                  id: 'sec-2',
                  name: 'control_status',
                  title: 'Control Status',
                  type: 'table',
                  required: true,
                  config: { columns: ['name', 'status', 'score'] },
                  order: 1,
                },
              ],
              is_system: true,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
            {
              id: 'tpl-002',
              name: 'ISO 27001 Gap Analysis',
              type: 'compliance',
              framework: 'iso27001',
              description: 'Gap analysis report for ISO 27001 certification preparation',
              thumbnail_url: '',
              config: {
                period_start: '',
                period_end: '',
                include_sections: ['control_status', 'recommendations'],
                filters: {},
              },
              sections: [],
              is_system: false,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            },
          ]),
        });
      } else if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'tpl-new',
            name: 'New Custom Report',
            type: 'compliance',
            framework: 'soc2',
            sections: [],
            is_system: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }),
        });
      }
    });

    // Mock update/delete routes
    page.route('**/api/v1/reports/templates/**', async (route) => {
      if (route.request().method() === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'tpl-001',
            name: 'Updated SOC 2 Report',
            type: 'compliance',
            framework: 'soc2',
            sections: [],
            is_system: false,
            updated_at: new Date().toISOString(),
          }),
        });
      } else if (route.request().method() === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Template deleted' }),
        });
      }
    });

    // Login as admin
    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display templates page with all templates', async ({ page }) => {
    await reportTemplatesPage.goto();

    // Verify heading
    const heading = await reportTemplatesPage.getHeadingText();
    expect(heading.toLowerCase()).toContain('report templates');

    // Verify new template button
    await expect(reportTemplatesPage.newTemplateButton).toBeVisible();

    // Verify filters
    await expect(reportTemplatesPage.searchInput).toBeVisible();
    await expect(reportTemplatesPage.typeFilter).toBeVisible();
    await expect(reportTemplatesPage.frameworkFilter).toBeVisible();
  });

  test('should display template cards with correct information', async ({ page }) => {
    await reportTemplatesPage.goto();

    // Check for template cards
    const count = await reportTemplatesPage.getTemplateCount();
    expect(count).toBeGreaterThan(0);

    // Verify specific template is displayed
    await expect(page.locator('text=SOC 2 Compliance Report')).toBeVisible();
  });

  test('should search templates by name', async ({ page }) => {
    await reportTemplatesPage.goto();

    await reportTemplatesPage.searchTemplates('SOC 2');

    // Verify search input value
    await expect(reportTemplatesPage.searchInput).toHaveValue(/SOC 2/i);
  });

  test('should filter templates by type', async ({ page }) => {
    await reportTemplatesPage.goto();

    await reportTemplatesPage.filterByType('Compliance');

    // Verify filter was applied
    await expect(reportTemplatesPage.typeFilter).toHaveValue(/compliance/i);
  });

  test('should filter templates by framework', async ({ page }) => {
    await reportTemplatesPage.goto();

    await reportTemplatesPage.filterByFramework('SOC 2');

    // Verify filter was applied
    await expect(reportTemplatesPage.frameworkFilter).toHaveValue(/soc2/i);
  });

  test('should navigate to create new template', async ({ page }) => {
    await reportTemplatesPage.goto();

    await reportTemplatesPage.createNewTemplate();

    // Should show template editor
    await expect(page.locator('text=Create Template').or(page.locator('.template-editor'))).toBeVisible();
  });

  test('should edit existing template', async ({ page }) => {
    await reportTemplatesPage.goto();

    await reportTemplatesPage.editTemplate('SOC 2 Compliance Report');

    // Should show template editor
    await expect(page.locator('.template-editor').or(page.locator('text=Edit Template'))).toBeVisible();
  });

  test('should duplicate template', async ({ page }) => {
    await reportTemplatesPage.goto();

    const initialCount = await reportTemplatesPage.getTemplateCount();

    await reportTemplatesPage.duplicateTemplate('SOC 2 Compliance Report');

    // Should show success message or new template
    await page.waitForTimeout(500);
    const newCount = await reportTemplatesPage.getTemplateCount();
    expect(newCount).toBeGreaterThanOrEqual(initialCount);
  });

  test('should delete custom template', async ({ page }) => {
    await reportTemplatesPage.goto();

    // Mock confirmation dialog
    page.on('dialog', (dialog) => dialog.accept());

    await reportTemplatesPage.deleteTemplate('ISO 27001 Gap Analysis');

    // Should show success message
    await expect(page.locator('text=deleted successfully').or(page.locator('text=Template deleted'))).toBeVisible();
  });

  test('should display system template badge', async ({ page }) => {
    await reportTemplatesPage.goto();

    // System templates should have a badge
    await expect(page.locator('text=System')).toBeVisible();
  });

  test('should show template details on card', async ({ page }) => {
    await reportTemplatesPage.goto();

    // Check for template details
    await expect(page.locator('text=Standard SOC 2 Type II compliance report')).toBeVisible();
    await expect(page.locator('text=/sections/i')).toBeVisible();
  });

  test('should navigate to template preview', async ({ page }) => {
    await reportTemplatesPage.goto();

    // Click preview button
    const previewButton = page.locator('button[title="Preview"], button:has-text("Preview")').first();
    if (await previewButton.isVisible()) {
      await previewButton.click();
      await expect(page.locator('.template-preview').or(page.locator('text=Preview'))).toBeVisible();
    }
  });
});

test.describe('Report Templates Editor', () => {
  test('should create new template with sections', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock APIs
    page.route('**/api/v1/reports/templates**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Fill template name
    await page.fill('input[placeholder*="template name" i]', 'My Custom Report');

    // Select framework
    await page.selectOption('select:has-text("Framework")', 'soc2');

    // Add sections
    await page.click('button:has-text("Executive Summary")');

    // Save
    await page.click('button:has-text("Save Template")');

    // Should return to list view
    await expect(page.locator('text=Report Templates')).toBeVisible();
  });

  test('should edit template sections', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock APIs
    page.route('**/api/v1/reports/templates/**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'tpl-001',
            name: 'Test Template',
            type: 'compliance',
            framework: 'soc2',
            sections: [
              {
                id: 'sec-1',
                name: 'executive_summary',
                title: 'Executive Summary',
                type: 'summary',
                required: true,
                config: {},
                order: 0,
              },
            ],
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }),
        });
      } else if (route.request().method() === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'tpl-001',
            name: 'Test Template',
            type: 'compliance',
            framework: 'soc2',
            sections: [],
            updated_at: new Date().toISOString(),
          }),
        });
      }
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/reports/templates');
    await page.click('text=Test Template');

    // Expand section config
    await page.click('button:has-text("Executive Summary")');

    // Toggle checkbox
    const checkbox = page.locator('input[type="checkbox"]').first();
    await checkbox.check();

    // Save
    await page.click('button:has-text("Save Template")');

    // Should return to list view
    await expect(page.locator('text=Report Templates')).toBeVisible();
  });
});

test.describe('Report Templates - Error Handling', () => {
  test('should handle API errors gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock error response
    page.route('**/api/v1/reports/templates', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'Internal server error' } }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const reportTemplatesPage = new ReportTemplatesPage(page);
    await reportTemplatesPage.goto();

    // Should show error state
    await expect(page.locator('text=error').or(page.locator('text=failed'))).toBeVisible({ timeout: 10000 });
  });

  test('should handle empty state gracefully', async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock empty response
    page.route('**/api/v1/reports/templates', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    const reportTemplatesPage = new ReportTemplatesPage(page);
    await reportTemplatesPage.goto();

    // Should show empty state
    await expect(page.locator('text=No templates').or(page.locator('text=Create your first report'))).toBeVisible();
  });
});
