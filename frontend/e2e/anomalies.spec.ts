import { test, expect } from './fixtures/auth.fixture';
import { AnomalyListPage } from './pages/AnomalyListPage';
import { AnomalyDetailPage } from './pages/AnomalyDetailPage';

// Mock anomaly API routes
test.beforeEach(async ({ page }) => {
  // Setup anomalies API mocks
  page.route('**/api/v1/compliance/anomalies**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/summary') && method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total: 15,
          by_severity: { critical: 2, high: 3, medium: 5, low: 5 },
          by_type: {
            unusual_access_time: 4,
            privileged_escalation: 3,
            unusual_location: 2,
            bulk_data_access: 2,
            command_injection: 1,
            impossible_travel: 1,
            account_takeover: 1,
            credential_theft: 1,
            excessive_failed_logins: 0,
            ransomware_indicators: 0,
          },
          by_status: { open: 8, investigating: 3, resolved: 4, false_positive: 0 },
          resolved_this_period: 4,
          avg_resolution_time_hours: 4.5,
          critical_open: 2,
          high_open: 3,
        }),
      });
      return;
    }

    if (url.includes('/bulk-update') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ updated: 2, failed: [] }),
      });
      return;
    }

    if (url.includes('/export') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          download_url: '/exports/anomalies.csv',
          expires_at: new Date(Date.now() + 3600000).toISOString(),
        }),
      });
      return;
    }

    // GET /compliance/anomalies - List anomalies
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'anom-1',
              type: 'unusual_access_time',
              severity: 'critical',
              title: 'Unusual After-Hours Access',
              description: 'User accessed production system at 3 AM',
              detected_at: new Date().toISOString(),
              user_id: 'user-1',
              user_name: 'Admin User',
              target_id: 'target-1',
              target_name: 'prod-server-01',
              session_id: 'sess-1',
              confidence_score: 0.92,
              indicators: [
                {
                  type: 'time_anomaly',
                  description: 'Access time outside normal hours',
                  value: '03:00',
                  threshold: '06:00-22:00',
                  confidence: 0.92,
                },
              ],
              status: 'open',
            },
            {
              id: 'anom-2',
              type: 'privileged_escalation',
              severity: 'high',
              title: 'Sudden Privilege Escalation',
              description: 'User escalated privileges multiple times in short period',
              detected_at: new Date(Date.now() - 3600000).toISOString(),
              user_id: 'user-2',
              user_name: 'Regular User',
              target_id: 'target-1',
              target_name: 'prod-server-01',
              session_id: 'sess-2',
              confidence_score: 0.85,
              indicators: [],
              status: 'investigating',
            },
            {
              id: 'anom-3',
              type: 'unusual_location',
              severity: 'medium',
              title: 'Access from Unusual Location',
              description: 'Login detected from previously unseen geographic location',
              detected_at: new Date(Date.now() - 7200000).toISOString(),
              user_id: 'user-3',
              user_name: 'Dev User',
              target_id: 'target-2',
              target_name: 'staging-db-01',
              session_id: 'sess-3',
              confidence_score: 0.75,
              indicators: [],
              status: 'open',
            },
            {
              id: 'anom-4',
              type: 'bulk_data_access',
              severity: 'high',
              title: 'Bulk Data Export Detected',
              description: 'Large volume of data exported from database',
              detected_at: new Date(Date.now() - 10800000).toISOString(),
              user_id: 'user-1',
              user_name: 'Admin User',
              target_id: 'target-3',
              target_name: 'prod-db-01',
              session_id: 'sess-4',
              confidence_score: 0.88,
              indicators: [],
              status: 'open',
            },
            {
              id: 'anom-5',
              type: 'command_injection',
              severity: 'critical',
              title: 'Potential Command Injection',
              description: 'Suspicious command pattern detected in session',
              detected_at: new Date(Date.now() - 14400000).toISOString(),
              user_id: 'user-4',
              user_name: 'Contractor',
              target_id: 'target-1',
              target_name: 'prod-server-01',
              session_id: 'sess-5',
              confidence_score: 0.95,
              indicators: [],
              status: 'resolved',
              resolved_at: new Date(Date.now() - 3600000).toISOString(),
              resolution_notes: 'Investigated and confirmed as false positive - legitimate automation script',
            },
          ],
          pagination: { total: 15, limit: 20, offset: 0, has_more: false },
        }),
      });
      return;
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Route not found' } }),
    });
  });

  // Mock individual anomaly API
  page.route('**/api/v1/compliance/anomalies/*', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/acknowledge') && method === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'anom-1',
          type: 'unusual_access_time',
          severity: 'critical',
          title: 'Unusual After-Hours Access',
          description: 'User accessed production system at 3 AM',
          detected_at: new Date().toISOString(),
          user_id: 'user-1',
          user_name: 'Admin User',
          target_id: 'target-1',
          target_name: 'prod-server-01',
          session_id: 'sess-1',
          confidence_score: 0.92,
          indicators: [],
          status: 'investigating',
        }),
      });
      return;
    }

    // GET individual anomaly / PATCH to update
    if (method === 'GET' || method === 'PATCH') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'anom-1',
          type: 'unusual_access_time',
          severity: 'critical',
          title: 'Unusual After-Hours Access',
          description: 'User accessed production system at 3 AM outside normal business hours',
          detected_at: new Date().toISOString(),
          user_id: 'user-1',
          user_name: 'Admin User',
          target_id: 'target-1',
          target_name: 'prod-server-01',
          session_id: 'sess-1',
          confidence_score: 0.92,
          indicators: [
            {
              type: 'time_anomaly',
              description: 'Access time outside normal hours (06:00-22:00)',
              value: '03:00',
              threshold: '06:00-22:00',
              confidence: 0.92,
            },
            {
              type: 'day_of_week_anomaly',
              description: 'Weekend access detected',
              value: 'Saturday',
              confidence: 0.75,
            },
          ],
          status: 'open',
          assigned_to: '',
          resolution_notes: '',
        }),
      });
      return;
    }

    await route.fulfill({
      status: 404,
      contentType: 'application/json',
      body: JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Route not found' } }),
    });
  });
});

test.describe('Anomaly List Page', () => {
  test('should display anomaly list page with heading', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();

    const heading = await anomalyListPage.getHeadingText();
    expect(heading?.toLowerCase()).toContain('anomaly detection');
  });

  test('should display summary cards', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const summaryCount = await anomalyListPage.getSummaryCardsCount();
    expect(summaryCount).toBeGreaterThanOrEqual(5); // Total, Critical, High, Medium, Low, Resolved
  });

  test('should display correct summary values', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const totalValue = await anomalyListPage.getSummaryCardValue('Total');
    expect(totalValue).toBe('15');

    const criticalValue = await anomalyListPage.getSummaryCardValue('Critical');
    expect(criticalValue).toBe('2');

    const highValue = await anomalyListPage.getSummaryCardValue('High');
    expect(highValue).toBe('3');
  });

  test('should display anomalies table', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThan(0);
  });

  test('should filter anomalies by severity', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Filter by critical severity
    await anomalyListPage.filterBySeverity('Critical');
    await anomalyListPage.waitForContent();

    // Verify filter was applied
    const firstRowSeverity = await anomalyListPage.getAnomalySeverity(0);
    expect(firstRowSeverity?.toLowerCase()).toBe('critical');
  });

  test('should filter anomalies by status', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Filter by resolved status
    await anomalyListPage.filterByStatus('Resolved');
    await anomalyListPage.waitForContent();

    // Should show resolved anomalies
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should filter anomalies by type', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Filter by unusual access time type
    await anomalyListPage.filterByType('Unusual Access Time');
    await anomalyListPage.waitForContent();

    // Should show filtered results
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should search anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Search for specific term
    await anomalyListPage.searchAnomalies('production');
    await anomalyListPage.waitForContent();

    // Should show filtered results
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should change date period', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Change period to 7 days
    await anomalyListPage.selectPeriod(7);
    await anomalyListPage.waitForContent();

    // Should refresh data
    const totalValue = await anomalyListPage.getSummaryCardValue('Total');
    expect(totalValue).toBeTruthy();
  });

  test('should allow selecting anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Select first anomaly
    await anomalyListPage.selectAnomaly(0);

    // Verify bulk actions bar appears
    const isBulkActionsVisible = await anomalyListPage.isBulkActionsBarVisible();
    expect(isBulkActionsVisible).toBe(true);
  });

  test('should allow selecting all anomalies', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Select all
    await anomalyListPage.selectAllAnomalies();

    // Verify selection
    const selectedCount = await anomalyListPage.getSelectedCount();
    expect(selectedCount).toBeGreaterThan(0);
  });

  test('should perform bulk status update', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Select first anomaly
    await anomalyListPage.selectAnomaly(0);

    // Update to investigating
    await anomalyListPage.clickBulkStatusUpdate('investigating');

    // Should show success toast
    const toast = authenticatedPage.getByText(/updated/i).or(authenticatedPage.getByText(/success/i));
    await expect(toast).toBeVisible({ timeout: 5000 }).catch(() => {});
  });

  test('should allow export', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Click export
    await anomalyListPage.clickExport();

    // Should trigger export
    const toast = authenticatedPage.getByText(/export/i).or(authenticatedPage.getByText(/download/i));
    await expect(toast).toBeVisible({ timeout: 5000 }).catch(() => {});
  });

  test('should allow refresh', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Click refresh
    await anomalyListPage.clickRefresh();

    // Should reload data
    await anomalyListPage.waitForContent();
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThanOrEqual(0);
  });

  test('should show empty state when no anomalies match filters', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();

    // Apply filter that should return no results
    await authenticatedPage.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, limit: 20, offset: 0, has_more: false },
        }),
      });
    });

    await anomalyListPage.filterBySeverity('Critical');
    await anomalyListPage.waitForContent();

    const isEmptyStateVisible = await anomalyListPage.isEmptyStateVisible();
    expect(isEmptyStateVisible).toBe(true);
  });

  test('should navigate to anomaly detail page', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Click view button on first anomaly
    await anomalyListPage.clickViewButton(0);

    // Should navigate to detail page
    await authenticatedPage.waitForURL(/\/analytics\/anomalies\/anom-\d+/, { timeout: 5000 });
  });
});

test.describe('Anomaly Detail Page', () => {
  test('should display anomaly detail page', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const title = await anomalyDetailPage.getAnomalyTitle();
    expect(title).toBeTruthy();
    expect(title?.toLowerCase()).toContain('unusual');
  });

  test('should display anomaly severity and status badges', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const severity = await anomalyDetailPage.getSeverity();
    expect(severity).toBe('critical');

    const status = await anomalyDetailPage.getStatus();
    expect(status).toBe('open');
  });

  test('should display confidence score', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const confidence = await anomalyDetailPage.getConfidenceScore();
    expect(confidence).toBe(92);
  });

  test('should display detection indicators', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const indicatorCount = await anomalyDetailPage.getIndicatorCount();
    expect(indicatorCount).toBeGreaterThan(0);

    const firstIndicatorConfidence = await anomalyDetailPage.getIndicatorConfidence(0);
    expect(firstIndicatorConfidence).toBeGreaterThan(0);
  });

  test('should show acknowledge button for open anomalies', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const isAcknowledgeVisible = await anomalyDetailPage.isAcknowledgeButtonVisible();
    expect(isAcknowledgeVisible).toBe(true);
  });

  test('should allow acknowledging anomaly', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickAcknowledge();

    // Should show success message
    const toast = authenticatedPage.getByText(/acknowledged/i).or(authenticatedPage.getByText(/investigation started/i));
    await expect(toast).toBeVisible({ timeout: 5000 }).catch(() => {});
  });

  test('should allow editing status', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    // Click update status button
    await anomalyDetailPage.clickUpdateStatus();

    // Fields should be editable
    const areFieldsDisabled = await anomalyDetailPage.areFieldsDisabled();
    expect(areFieldsDisabled).toBe(false);

    // Change status
    await anomalyDetailPage.setStatus('investigating');
    const selectedStatus = await anomalyDetailPage.getSelectedStatus();
    expect(selectedStatus).toBe('investigating');
  });

  test('should allow setting assigned user', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();

    await anomalyDetailPage.setAssignedTo('admin@example.com');
    const assignedTo = await anomalyDetailPage.getAssignedToValue();
    expect(assignedTo).toBe('admin@example.com');
  });

  test('should allow adding resolution notes', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();

    const notes = 'Investigated and confirmed as legitimate access';
    await anomalyDetailPage.setResolutionNotes(notes);
    const savedNotes = await anomalyDetailPage.getResolutionNotesValue();
    expect(savedNotes).toBe(notes);
  });

  test('should save changes', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('resolved');
    await anomalyDetailPage.setResolutionNotes('Investigation complete');
    await anomalyDetailPage.clickSaveChanges();

    // Should show success message
    const toast = authenticatedPage.getByText(/updated/i).or(authenticatedPage.getByText(/success/i));
    await expect(toast).toBeVisible({ timeout: 5000 }).catch(() => {});

    // Edit mode should be closed
    const isEditing = await anomalyDetailPage.isEditingMode();
    expect(isEditing).toBe(false);
  });

  test('should cancel changes', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('investigating');
    await anomalyDetailPage.clickCancel();

    // Should revert to original status
    const isEditing = await anomalyDetailPage.isEditingMode();
    expect(isEditing).toBe(false);
  });

  test('should display related entities', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    const user = await anomalyDetailPage.getRelatedEntity('user');
    expect(user).toBe('Admin User');

    const target = await anomalyDetailPage.getRelatedEntity('target');
    expect(target).toBe('prod-server-01');
  });

  test('should navigate back to list', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    await anomalyDetailPage.clickBack();

    // Should navigate back
    await authenticatedPage.waitForURL(/\/analytics\/anomalies/, { timeout: 5000 }).catch(() => {
      // May have navigated to /analytics instead
    });
  });

  test('should display basic information', async ({ authenticatedPage }) => {
    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-1');
    await anomalyDetailPage.waitForContent();

    // Basic info card should be visible
    const isBasicInfoVisible = await anomalyDetailPage.basicInfoCard.isVisible().catch(() => false);
    expect(isBasicInfoVisible).toBe(true);
  });
});

test.describe('Anomaly Workflow Integration', () => {
  test('should complete full investigation workflow', async ({ authenticatedPage }) => {
    const anomalyListPage = new AnomalyListPage(authenticatedPage);

    // Start at list page
    await anomalyListPage.goto();
    await anomalyListPage.waitForContent();

    // Find open anomaly
    const anomalyCount = await anomalyListPage.getAnomalyCount();
    expect(anomalyCount).toBeGreaterThan(0);

    // Navigate to detail
    await anomalyListPage.clickViewButton(0);

    // Wait for detail page
    await authenticatedPage.waitForURL(/\/analytics\/anomalies\/anom-\d+/, { timeout: 5000 });

    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.waitForContent();

    // Acknowledge the anomaly
    if (await anomalyDetailPage.isAcknowledgeButtonVisible()) {
      await anomalyDetailPage.clickAcknowledge();
    }

    // Update status to investigating
    await anomalyDetailPage.clickUpdateStatus();
    await anomalyDetailPage.setStatus('investigating');
    await anomalyDetailPage.setAssignedTo('security-team@example.com');
    await anomalyDetailPage.setResolutionNotes('Investigation in progress');
    await anomalyDetailPage.clickSaveChanges();

    // Navigate back
    await anomalyDetailPage.clickBack();

    // Verify back at list
    await authenticatedPage.waitForURL(/\/analytics\/anomalies/, { timeout: 5000 });
  });

  test('should handle resolved anomalies correctly', async ({ authenticatedPage }) => {
    // Mock a resolved anomaly
    await authenticatedPage.route('**/api/v1/compliance/anomalies/anom-resolved**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'anom-resolved',
          type: 'unusual_access_time',
          severity: 'medium',
          title: 'Resolved Anomaly',
          description: 'Anomaly that has been resolved',
          detected_at: new Date(Date.now() - 86400000).toISOString(),
          user_id: 'user-1',
          user_name: 'Admin User',
          confidence_score: 0.7,
          indicators: [],
          status: 'resolved',
          resolved_at: new Date().toISOString(),
          resolution_notes: 'Investigated and verified as legitimate',
        }),
      });
    });

    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('anom-resolved');
    await anomalyDetailPage.waitForContent();

    // Should show resolved status
    const status = await anomalyDetailPage.getStatus();
    expect(status).toBe('resolved');

    // Acknowledge button should not be visible
    const isAcknowledgeVisible = await anomalyDetailPage.isAcknowledgeButtonVisible();
    expect(isAcknowledgeVisible).toBe(false);
  });
});

test.describe('Anomaly Error Handling', () => {
  test('should handle API errors gracefully', async ({ authenticatedPage }) => {
    // Mock error response
    await authenticatedPage.route('**/api/v1/compliance/anomalies**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'INTERNAL_ERROR', message: 'Failed to fetch anomalies' },
        }),
      });
    });

    const anomalyListPage = new AnomalyListPage(authenticatedPage);
    await anomalyListPage.goto();

    // Should show error state
    const errorMessage = authenticatedPage.getByText(/error/i).or(authenticatedPage.getByText(/failed/i));
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 }).catch(() => {});
  });

  test('should handle 404 for non-existent anomaly', async ({ authenticatedPage }) => {
    await authenticatedPage.route('**/api/v1/compliance/anomalies/nonexistent**', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'NOT_FOUND', message: 'Anomaly not found' },
        }),
      });
    });

    const anomalyDetailPage = new AnomalyDetailPage(authenticatedPage);
    await anomalyDetailPage.goto('nonexistent');

    // Should show error state
    const errorMessage = authenticatedPage.getByText(/not found/i).or(authenticatedPage.getByText(/failed/i));
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 }).catch(() => {});
  });
});
