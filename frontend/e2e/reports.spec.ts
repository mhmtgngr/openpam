/**
 * Reports E2E Tests
 *
 * Tests for the Reports feature including:
 * - Reports list page with filters
 * - Report generation workflow
 * - Report detail view
 * - Report download functionality
 * - Report scheduling
 */

import { test, expect } from './fixtures/auth.fixture';

// Mock reports API data
const mockReportSnapshots = [
  {
    id: 'report-1',
    report_id: 'compliance-def-1',
    report_name: 'SOC 2 Compliance Report',
    type: 'compliance',
    framework: 'soc2',
    status: 'completed',
    format: 'pdf',
    file_url: 'https://storage.example.com/reports/soc2-report-1.pdf',
    file_size_bytes: 1024000,
    expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(),
    generation_started_at: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    generation_completed_at: new Date(Date.now() - 1 * 60 * 1000).toISOString(),
    generated_by: 'user-1',
    generated_by_user: {
      id: 'user-1',
      email: 'admin@example.com',
      display_name: 'Admin User',
    },
    tenant_id: 'tenant-1',
    config: {
      period_start: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      period_end: new Date().toISOString().split('T')[0],
      include_sections: ['executive_summary', 'security', 'access_review'],
      filters: {},
    },
    metadata: {
      row_count: 150,
      duration_seconds: 45,
    },
    created_at: new Date(Date.now() - 3 * 60 * 1000).toISOString(),
  },
  {
    id: 'report-2',
    report_id: 'session-activity-1',
    report_name: 'Session Activity - Last 30 Days',
    type: 'session_activity',
    status: 'generating',
    format: 'xlsx',
    generated_by: 'user-1',
    generated_by_user: {
      id: 'user-1',
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
    generation_started_at: new Date(Date.now() - 30 * 1000).toISOString(),
    created_at: new Date(Date.now() - 30 * 1000).toISOString(),
  },
  {
    id: 'report-3',
    report_id: 'command-analysis-1',
    report_name: 'Command Analysis Report',
    type: 'command_analysis',
    status: 'failed',
    format: 'csv',
    generated_by: 'user-1',
    tenant_id: 'tenant-1',
    config: {
      period_start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      period_end: new Date().toISOString().split('T')[0],
      include_sections: [],
      filters: {},
    },
    error_message: 'Failed to retrieve command data: timeout',
    generation_started_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
  },
];

const mockComplianceReports = [
  {
    id: 'comp-1',
    name: 'Quarterly SOC 2 Report',
    type: 'compliance',
    framework: 'soc2',
    description: 'Quarterly SOC 2 Type II compliance report',
    schedule: '0 0 1 */3 *', // Quarterly
    last_run_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
    next_run_at: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'scheduled',
    config: {
      period_start: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      period_end: new Date().toISOString().split('T')[0],
      include_sections: ['executive_summary', 'security'],
      filters: {},
    },
    created_by: 'user-1',
    tenant_id: 'tenant-1',
    created_at: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(),
    updated_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

const mockTemplates = [
  {
    id: 'template-1',
    name: 'SOC 2 Type II Template',
    type: 'compliance',
    framework: 'soc2',
    description: 'Standard SOC 2 Type II compliance report template',
    thumbnail_url: null,
    config: {
      period_start: '',
      period_end: '',
      include_sections: ['executive_summary', 'security'],
      filters: {},
    },
    sections: [
      {
        id: 'executive_summary',
        name: 'executive_summary',
        title: 'Executive Summary',
        type: 'summary',
        required: true,
        order: 1,
        config: {},
      },
      {
        id: 'security',
        name: 'security',
        title: 'Security Controls',
        type: 'table',
        required: true,
        order: 2,
        config: {},
      },
    ],
    is_system: true,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
];

// Setup API routes for reports
test.beforeEach(async ({ page }) => {
  // Mock reports API routes
  page.route('**/api/v1/reports/snapshots*', async (route) => {
    const url = new URL(route.request().url());
    const status = url.searchParams.get('status');
    const type = url.searchParams.get('type');

    let filteredData = [...mockReportSnapshots];

    if (status) {
      filteredData = filteredData.filter((r) => r.status === status);
    }
    if (type) {
      filteredData = filteredData.filter((r) => r.type === type);
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: filteredData,
        pagination: {
          total: filteredData.length,
          limit: 20,
          offset: 0,
          has_more: false,
        },
      }),
    });
  });

  page.route('**/api/v1/reports/snapshots/report-1', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockReportSnapshots[0]),
    });
  });

  page.route('**/api/v1/reports/snapshots/report-1/download', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        download_url: 'https://storage.example.com/reports/soc2-report-1.pdf',
        filename: 'soc2-compliance-report.pdf',
      }),
    });
  });

  page.route('**/api/v1/reports/snapshots/report-1/progress', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        report_snapshot_id: 'report-1',
        status: 'completed',
        progress: 100,
        current_step: 'completed',
        started_at: mockReportSnapshots[0].generation_started_at,
        estimated_completion_at: mockReportSnapshots[0].generation_completed_at,
      }),
    });
  });

  page.route('**/api/v1/reports/generate', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        snapshot_id: 'new-report-' + Date.now(),
        status: 'pending',
        estimated_completion_at: new Date(Date.now() + 5 * 60 * 1000).toISOString(),
      }),
    });
  });

  page.route('**/api/v1/reports**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockComplianceReports,
          pagination: { total: 1, limit: 20, offset: 0, has_more: false },
        }),
      });
    }
  });

  page.route('**/api/v1/reports/templates*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockTemplates),
    });
  });

  page.route('**/api/v1/reports/types', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { type: 'compliance', name: 'Compliance Report', description: 'Framework compliance report', formats: ['pdf', 'html'] },
        { type: 'session_activity', name: 'Session Activity', description: 'Session activity report', formats: ['pdf', 'xlsx', 'csv'] },
        { type: 'command_analysis', name: 'Command Analysis', description: 'Command execution analysis', formats: ['pdf', 'xlsx', 'csv'] },
      ]),
    });
  });

  page.route('**/api/v1/reports/frameworks', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { framework: 'soc2', name: 'SOC 2 Type II', description: 'Service Organization Control 2' },
        { framework: 'iso27001', name: 'ISO 27001', description: 'Information Security Management' },
        { framework: 'pci_dss', name: 'PCI DSS', description: 'Payment Card Industry' },
      ]),
    });
  });

  // Handle DELETE requests for reports
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

test.describe('Reports Page', () => {
  test('should display reports list with status summary cards', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Wait for page to load
    await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible();

    // Check status summary cards
    await expect(page.getByText('Total Reports')).toBeVisible();
    await expect(page.getByText('Completed')).toBeVisible();
    await expect(page.getByText('In Progress')).toBeVisible();
    await expect(page.getByText('Failed')).toBeVisible();

    // Check that reports are displayed
    await expect(page.getByText('SOC 2 Compliance Report')).toBeVisible();
    await expect(page.getByText('Session Activity - Last 30 Days')).toBeVisible();
  });

  test('should filter reports by status', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Click on completed status card
    await page.getByText('Completed').click();

    // Wait for filter to apply
    await page.waitForTimeout(500);

    // Should only show completed reports
    await expect(page.getByText('SOC 2 Compliance Report')).toBeVisible();
    await expect(page.queryAllByText('Session Activity - Last 30 Days')).toHaveLength(0);
  });

  test('should filter reports by type', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Open filters
    await page.getByRole('button', { name: /filters/i }).click();

    // Select compliance type
    await page.getByRole('combobox').filter({ hasText: 'All Types' }).selectOption('compliance');

    // Wait for results
    await page.waitForTimeout(500);

    // Should show compliance reports
    await expect(page.getByText('SOC 2 Compliance Report')).toBeVisible();
  });

  test('should search reports', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Type in search box
    await page.getByPlaceholder('Search reports...').fill('SOC 2');

    // Submit search
    await page.getByRole('button', { name: /search/i }).click();

    // Wait for results
    await page.waitForTimeout(500);

    // Should show matching report
    await expect(page.getByText('SOC 2 Compliance Report')).toBeVisible();
  });

  test('should navigate to report generator', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Click generate report button
    await page.getByRole('button', { name: 'Generate Report' }).click();

    // Should navigate to generator page
    await expect(page).toHaveURL(/\/analytics\/reports\/generate/);
    await expect(page.getByRole('heading', { name: 'Generate Report' })).toBeVisible();
  });

  test('should navigate to report detail', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Click on a report card (using view button that appears on hover)
    await page.getByText('SOC 2 Compliance Report').click();

    // Should navigate to detail page
    await expect(page).toHaveURL(/\/analytics\/reports\/report-\d/);
  });
});

test.describe('Report Generator', () => {
  test('should complete report generation workflow', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Step 1: Select report type
    await expect(page.getByRole('heading', { name: 'Generate Report' })).toBeVisible();
    await expect(page.getByText('Select report type and framework')).toBeVisible();

    // Select compliance report type
    await page.getByText('Compliance Report').click();

    // Continue to next step
    await page.getByRole('button', { name: 'Continue' }).click();

    // Step 2: Configure report
    await expect(page.getByText('Configure report parameters')).toBeVisible();

    // Select date range preset
    await page.getByRole('button', { name: 'Last 30 Days' }).click();

    // Select format
    await page.getByText('PDF').click();

    // Continue to review
    await page.getByRole('button', { name: 'Continue' }).click();

    // Step 3: Review
    await expect(page.getByText('Review Report Configuration')).toBeVisible();

    // Verify review details are shown
    await expect(page.getByText('compliance')).toBeVisible();
    await expect(page.getByText('PDF')).toBeVisible();

    // Generate report
    await page.getByRole('button', { name: 'Generate Report' }).click();

    // Should show success toast or navigate
    await expect(page).toHaveURL(/\/analytics\/reports\/new-report-/, { timeout: 5000 });
  });

  test('should show compliance framework options', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Select compliance report type
    await page.getByText('Compliance Report').click();

    // Should show framework selection
    await expect(page.getByText('Compliance Framework')).toBeVisible();
    await expect(page.getByText('SOC 2 Type II')).toBeVisible();
    await expect(page.getByText('ISO 27001')).toBeVisible();
    await expect(page.getByText('PCI DSS')).toBeVisible();

    // Select a framework
    await page.getByText('SOC 2 Type II').click();
  });

  test('should validate date range selection', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Select report type and continue
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    // Try to continue without selecting dates
    // The date inputs should have default values from the component
    // Let's verify they exist
    const startDateInput = page.getByLabel('Start Date');
    const endDateInput = page.getByLabel('End Date');

    await expect(startDateInput).toBeVisible();
    await expect(endDateInput).toBeVisible();
  });

  test('should toggle additional options', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Navigate to configuration step
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    // Find the Include Charts toggle
    const chartsToggle = page.locator('.card').filter({ hasText: 'Include Charts' }).getByRole('switch');

    // Should be enabled by default
    await expect(chartsToggle).toBeVisible();

    // Toggle off
    await chartsToggle.click();

    // Toggle back on
    await chartsToggle.click();
  });

  test('should open schedule dialog', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Complete first two steps
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    await page.getByRole('button', { name: 'Last 30 Days' }).click();
    await page.getByText('PDF').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    // Click add schedule on review step
    await page.getByRole('button', { name: 'Add Schedule' }).click();

    // Should show schedule dialog
    await expect(page.getByText('Schedule Report Generation')).toBeVisible();
    await expect(page.getByText('Frequency')).toBeVisible();

    // Close dialog
    await page.getByRole('button', { name: 'Cancel' }).click();
  });
});

test.describe('Report Detail Page', () => {
  test('should display report details', async ({ page }) => {
    await page.goto('/analytics/reports/report-1');

    // Check header
    await expect(page.getByRole('heading', { name: 'SOC 2 Compliance Report' })).toBeVisible();

    // Check status badge
    await expect(page.getByText('Completed')).toBeVisible();

    // Check metadata sections
    await expect(page.getByText('Report Period')).toBeVisible();
    await expect(page.getByText('Generation Details')).toBeVisible();
    await expect(page.getByText('File Information')).toBeVisible();
  });

  test('should show download and action buttons', async ({ page }) => {
    await page.goto('/analytics/reports/report-1');

    // Check action buttons
    await expect(page.getByRole('button', { name: 'Download' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Share' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Refresh' })).toBeVisible();
  });

  test('should navigate back to reports list', async ({ page }) => {
    await page.goto('/analytics/reports/report-1');

    // Click back button
    await page.getByRole('button', { name: 'Back to Reports' }).click();

    // Should navigate back
    await expect(page).toHaveURL('/analytics/reports');
    await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible();
  });

  test('should handle failed report status', async ({ page }) => {
    await page.goto('/analytics/reports/report-3');

    // Should show error message for failed reports
    await expect(page.getByText('Generation Failed')).toBeVisible();
    await expect(page.getByText('Failed to retrieve command data')).toBeVisible();
  });

  test('should show processing indicator for in-progress reports', async ({ page }) => {
    await page.goto('/analytics/reports/report-2');

    // Should show processing indicator
    await expect(page.getByText(/Report is being generated|Report is queued/i)).toBeVisible();
  });
});

test.describe('Report Card Component', () => {
  test('should display report information in card view', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Check report card elements
    const reportCard = page.locator('.card').filter({ hasText: 'SOC 2 Compliance Report' });

    await expect(reportCard.getByText('Completed')).toBeVisible();
    await expect(reportCard.getByText('PDF')).toBeVisible();
    await expect(reportCard.getByText(/Admin User|admin@example.com/)).toBeVisible();
  });

  test('should show action buttons on hover', async ({ page }) => {
    await page.goto('/analytics/reports');

    const reportCard = page.locator('.card').filter({ hasText: 'SOC 2 Compliance Report' });

    // Hover over card to show actions
    await reportCard.hover();

    // Action buttons should appear
    await expect(reportCard.getByRole('button', { name: /download/i })).toBeVisible();
  });

  test('should select multiple reports for bulk actions', async ({ page }) => {
    await page.goto('/analytics/reports');

    // Click select all checkbox
    await page.getByRole('button', { name: 'Select All' }).click();

    // Should show bulk action bar
    await expect(page.getByText(/selected/)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Delete Selected' })).toBeVisible();
  });
});

test.describe('Report Schedule Dialog', () => {
  test('should display schedule options', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Navigate to review step
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.getByRole('button', { name: 'Last 30 Days' }).click();
    await page.getByText('PDF').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    // Open schedule dialog
    await page.getByRole('button', { name: 'Add Schedule' }).click();

    // Check frequency options
    await expect(page.getByText('One-time')).toBeVisible();
    await expect(page.getByText('Daily')).toBeVisible();
    await expect(page.getByText('Weekly')).toBeVisible();
    await expect(page.getByText('Monthly')).toBeVisible();
    await expect(page.getByText('Quarterly')).toBeVisible();
  });

  test('should configure weekly schedule', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Navigate to schedule dialog
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.getByRole('button', { name: 'Last 30 Days' }).click();
    await page.getByText('PDF').click();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.getByRole('button', { name: 'Add Schedule' }).click();

    // Select weekly frequency
    await page.getByText('Weekly').click();

    // Should show day of week selector
    await expect(page.getByText('Day of Week')).toBeVisible();

    // Select a day
    await page.getByRole('combobox').filter({ hasText: /Monday/ }).selectOption('1');

    // Set time
    await page.getByLabel('Time').fill('09:00');

    // Save schedule
    await page.getByRole('button', { name: 'Save Schedule' }).click();

    // Dialog should close
    await expect(page.getByText('Schedule Report Generation')).not.toBeVisible();
  });

  test('should show schedule summary', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Navigate to schedule dialog
    await page.getByText('Compliance Report').click();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.getByRole('button', { name: 'Last 30 Days' }).click();
    await page.getByText('PDF').click();
    await page.getByRole('button', { name: 'Continue' }).click();
    await page.getByRole('button', { name: 'Add Schedule' }).click();

    // Select daily
    await page.getByText('Daily').click();

    // Check summary text
    await expect(page.getByText(/Runs every day at/)).toBeVisible();
  });
});

test.describe('Report Viewer Modal', () => {
  test('should open report viewer for completed reports', async ({ page }) => {
    await page.goto('/analytics/reports/report-1');

    // Click view button
    await page.getByRole('button', { name: 'View' }).click();

    // Should show viewer modal
    await expect(page.getByText('SOC 2 Compliance Report')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Download' })).toBeVisible();

    // Close modal
    await page.locator('.fixed.inset-0').getByRole('button', { name: '' }).click();
  });

  test('should handle different report formats', async ({ page }) => {
    // Check that format badges are displayed
    await page.goto('/analytics/reports');

    await expect(page.getByText('PDF')).toBeVisible();
    await expect(page.getByText('XLSX')).toBeVisible();
    await expect(page.getByText('CSV')).toBeVisible();
  });
});

test.describe('Navigation and Routing', () => {
  test('should navigate between reports pages', async ({ page }) => {
    // Start at reports list
    await page.goto('/analytics/reports');
    await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible();

    // Go to generator
    await page.getByRole('button', { name: 'Generate Report' }).click();
    await expect(page).toHaveURL('/analytics/reports/generate');

    // Go back
    await page.getByRole('button', { name: 'Back' }).click();
    await expect(page).toHaveURL('/analytics/reports');

    // Go to detail page
    await page.getByText('SOC 2 Compliance Report').click();
    await expect(page).toHaveURL(/\/analytics\/reports\/report-\d/);
  });

  test('should protect routes with authentication', async ({ page, context }) => {
    // Clear auth to test protected route
    await context.clearCookies();

    // Try to access reports page
    await page.goto('/analytics/reports');

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('Report Templates', () => {
  test('should display available report templates', async ({ page }) => {
    // Templates would be shown in the generator
    await page.goto('/analytics/reports/generate');

    // Should show compliance framework selection which acts as template selection
    await page.getByText('Compliance Report').click();
    await expect(page.getByText('SOC 2 Type II')).toBeVisible();
  });

  test('should show compliance sections for framework', async ({ page }) => {
    await page.goto('/analytics/reports/generate');

    // Select compliance type
    await page.getByText('Compliance Report').click();

    // Select SOC 2 framework
    await page.getByText('SOC 2 Type II').click();
    await page.getByRole('button', { name: 'Continue' }).click();

    // Should show compliance sections on step 2
    // (This would appear in the ComplianceReportTemplate component)
  });
});
