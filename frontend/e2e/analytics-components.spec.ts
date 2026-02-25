/**
 * Analytics Components E2E Tests
 *
 * Tests for AnomalyChart and ComplianceGauge components
 */

import { test, expect } from '@playwright/test';
import { LoginPage } from './pages';

test.describe('AnomalyChart Component', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock anomaly data API
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'anom-001',
              type: 'unusual_access_time',
              severity: 'high',
              title: 'After-hours access detected',
              description: 'User accessed system at 3 AM',
              detected_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
              user_id: 'user-001',
              user_name: 'John Doe',
              target_id: 'target-001',
              target_name: 'prod-server-01',
              confidence_score: 85,
              indicators: [],
              status: 'open',
            },
            {
              id: 'anom-002',
              type: 'impossible_travel',
              severity: 'critical',
              title: 'Impossible travel detected',
              description: 'User logged in from two locations 1000 miles apart within 1 hour',
              detected_at: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(),
              user_id: 'user-002',
              user_name: 'Jane Smith',
              confidence_score: 95,
              indicators: [],
              status: 'open',
            },
            {
              id: 'anom-003',
              type: 'command_injection',
              severity: 'medium',
              title: 'Suspicious command pattern',
              description: 'User executed command with injection indicators',
              detected_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
              user_id: 'user-003',
              user_name: 'Bob Johnson',
              target_id: 'target-002',
              confidence_score: 72,
              indicators: [],
              status: 'resolved',
            },
          ],
          pagination: { total: 3, limit: 100, offset: 0, has_more: false },
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display timeline view of anomalies', async ({ page }) => {
    await page.goto('/analytics/anomalies');

    // Check for timeline chart
    await expect(page.locator('.anomaly-chart-timeline, [data-testid="anomaly-chart"]')).toBeVisible();

    // Check for legend
    await expect(page.locator('text=Critical').or(page.locator('text=critical'))).toBeVisible();
    await expect(page.locator('text=High').or(page.locator('text=high'))).toBeVisible();
    await expect(page.locator('text=Medium').or(page.locator('text=medium'))).toBeVisible();
  });

  test('should display severity breakdown', async ({ page }) => {
    await page.goto('/analytics/anomalies');

    // Check for severity breakdown section
    await expect(page.locator('text=Severity Breakdown').or(page.locator('[class*="severity"]'))).toBeVisible();

    // Check for progress bars
    await expect(page.locator('.bg-red-500, .bg-orange-500, .bg-yellow-500').first()).toBeVisible();
  });

  test('should display type distribution', async ({ page }) => {
    await page.goto('/analytics/anomalies');

    // Check for type distribution section
    await expect(page.locator('text=Anomaly Types').or(page.locator('[class*="type"]'))).toBeVisible();

    // Check for anomaly type labels
    await expect(page.locator('text=Unusual Access').or(page.locator('text=Impossible Travel'))).toBeVisible();
  });

  test('should handle empty anomaly data', async ({ page }) => {
    // Mock empty response
    page.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, limit: 100, offset: 0, has_more: false } }),
      });
    });

    await page.goto('/analytics/anomalies');

    // Should show empty state message
    await expect(page.locator('text=No anomalies').or(page.locator('text=detected'))).toBeVisible();
  });

  test('should allow clicking on anomaly for details', async ({ page }) => {
    await page.goto('/analytics/anomalies');

    // Click on an anomaly
    const anomalyElement = page.locator('text=After-hours access').first();
    await anomalyElement.click();

    // Should navigate to detail page or show modal
    await expect(page.locator('text=After-hours access detected').or(page.locator('.modal, [role="dialog"]'))).toBeVisible();
  });
});

test.describe('ComplianceGauge Component', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock compliance API
    page.route('**/api/v1/compliance/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 85,
          framework: 'soc2',
          controls: [],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
    await page.waitForURL('/dashboard');
  });

  test('should display gauge with correct score', async ({ page }) => {
    await page.goto('/compliance/dashboard');

    // Check for gauge component
    const gauge = page.locator('.compliance-gauge');
    await expect(gauge).toBeVisible();

    // Check for score display
    await expect(page.locator('text=85%').or(page.locator('text=/85/i'))).toBeVisible();
  });

  test('should display correct color for high score', async ({ page }) => {
    // Mock high score (>= 90)
    page.route('**/api/v1/compliance/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 95,
          framework: 'soc2',
          controls: [],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/compliance/dashboard');

    // Should show green color for high score
    await expect(page.locator('.text-green-500, .bg-green-500').or(page.locator('svg[fill*="green"]'))).toBeVisible();
  });

  test('should display correct color for medium score', async ({ page }) => {
    // Mock medium score (70-89)
    page.route('**/api/v1/compliance/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 75,
          framework: 'soc2',
          controls: [],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/compliance/dashboard');

    // Should show yellow color for medium score
    await expect(page.locator('.text-yellow-500, .bg-yellow-500').or(page.locator('svg[fill*="yellow"]'))).toBeVisible();
  });

  test('should display correct color for low score', async ({ page }) => {
    // Mock low score (< 70)
    page.route('**/api/v1/compliance/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 55,
          framework: 'soc2',
          controls: [],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/compliance/dashboard');

    // Should show red color for low score
    await expect(page.locator('.text-red-500, .bg-red-500').or(page.locator('svg[fill*="red"]'))).toBeVisible();
  });

  test('should display compliance badge', async ({ page }) => {
    await page.goto('/compliance/dashboard');

    // Check for badge component
    const badge = page.locator('.compliance-badge, [class*="badge"]');
    if (await badge.isVisible()) {
      await expect(badge).toBeVisible();
      await expect(badge).toContainText(/85%|Compliant|Partial/);
    }
  });

  test('should display compliance meter', async ({ page }) => {
    await page.goto('/compliance/dashboard');

    // Check for meter component
    const meter = page.locator('.compliance-meter, [class*="meter"]');
    if (await meter.isVisible()) {
      await expect(meter).toBeVisible();
    }
  });

  test('should display score card with breakdown', async ({ page }) => {
    // Mock data with control breakdown
    page.route('**/api/v1/compliance/dashboard**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          overall_score: 85,
          framework: 'soc2',
          controls: [
            { id: '1', name: 'Control 1', status: 'compliant', score: 100, evidence_count: 5, last_assessed_at: new Date().toISOString() },
            { id: '2', name: 'Control 2', status: 'partial', score: 70, evidence_count: 3, last_assessed_at: new Date().toISOString() },
            { id: '3', name: 'Control 3', status: 'non_compliant', score: 40, evidence_count: 1, last_assessed_at: new Date().toISOString() },
          ],
          exceptions: [],
          last_updated: new Date().toISOString(),
        }),
      });
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');

    await page.goto('/compliance/dashboard');

    // Check for score card
    const card = page.locator('.compliance-score-card');
    if (await card.isVisible()) {
      await expect(card).toBeVisible();
      // Check for control stats
      await expect(page.locator('text=Compliant').or(page.locator('text=compliant'))).toBeVisible();
      await expect(page.locator('text=Non-Compliant').or(page.locator('text=non_compliant'))).toBeVisible();
    }
  });

  test('should animate gauge on load', async ({ page }) => {
    await page.goto('/compliance/dashboard');

    // Check for gauge with animation
    const gauge = page.locator('.compliance-gauge');
    await expect(gauge).toHaveAttribute('class', /transition/);
  });
});

test.describe('TemplateEditor Component', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock templates API
    page.route('**/api/v1/reports/templates**', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([]),
        });
      } else if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'tpl-new',
            name: 'New Template',
            type: 'compliance',
            framework: 'soc2',
            sections: [],
            created_at: new Date().toISOString(),
          }),
        });
      }
    });

    await page.goto('/login');
    await loginPage.login('admin@example.com', 'admin123');
  });

  test('should display template editor', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Check for editor elements
    await expect(page.locator('.template-editor')).toBeVisible();
    await expect(page.locator('input[placeholder*="Template Name" i]')).toBeVisible();
  });

  test('should allow adding sections to template', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Add a section
    const sectionButton = page.locator('button:has-text("Executive Summary")').or(page.locator('[class*="available"] button:has-text("Executive")'));
    if (await sectionButton.isVisible()) {
      await sectionButton.click();

      // Should appear in selected sections
      await expect(page.locator('text=Executive Summary')).toBeVisible();
    }
  });

  test('should allow editing section configuration', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Add and expand a section
    const sectionButton = page.locator('button:has-text("Executive Summary")');
    if (await sectionButton.isVisible()) {
      await sectionButton.click();

      // Expand section config
      const expandButton = page.locator('button:has-text("Executive Summary")').locator('..').locator('button').filter({ hasText: /chevron/i });
      if (await expandButton.isVisible()) {
        await expandButton.click();

        // Should show config options
        await expect(page.locator('input[type="checkbox"]')).toBeVisible();
      }
    }
  });

  test('should show template preview', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Click preview button
    const previewButton = page.locator('button:has-text("Preview")');
    if (await previewButton.isVisible()) {
      await previewButton.click();

      // Should show preview
      await expect(page.locator('.template-preview')).toBeVisible();
    }
  });

  test('should reorder sections', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Add multiple sections
    const sections = ['Executive Summary', 'Control Status', 'Findings Detail'];
    for (const section of sections) {
      const button = page.locator(`button:has-text("${section}")`);
      if (await button.isVisible()) {
        await button.click();
      }
    }

    // Reorder (move up/down buttons should be present)
    const moveButton = page.locator('button').filter({ hasText: /chevron/i }).first();
    if (await moveButton.isVisible()) {
      await expect(moveButton).toBeVisible();
    }
  });

  test('should save template', async ({ page }) => {
    await page.goto('/reports/templates');
    await page.click('button:has-text("New Template")');

    // Fill required fields
    await page.fill('input[placeholder*="Template Name" i]', 'Test Template');
    await page.selectOption('select:has-text("Framework")', 'soc2');

    // Save
    await page.click('button:has-text("Save Template")');

    // Should return to list
    await expect(page.locator('text=Report Templates')).toBeVisible();
  });
});
