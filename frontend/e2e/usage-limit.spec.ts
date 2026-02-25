import { test, expect } from '@playwright/test';

test.describe('Usage Limit Error Page', () => {
  test.beforeEach(async ({ page }) => {
    // Clear storage
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
  });

  test('should display usage limit page when navigated directly', async ({ page }) => {
    // Set up error data in localStorage
    const resetTime = new Date(Date.now() + 5 * 60 * 60 * 1000).toISOString();
    await page.evaluate((time) => {
      localStorage.setItem('usage_limit_code', '1308');
      localStorage.setItem('usage_limit_message', 'Usage limit reached for 5 hour. Your limit will reset at 2026-02-25 19:49:14');
      localStorage.setItem('usage_limit_reset_at', time);
      localStorage.setItem('usage_limit_request_id', 'req_test_12345');
    }, resetTime);

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Check main elements are visible
    await expect(page.getByText('Usage Limit Reached')).toBeVisible();
    await expect(page.getByText('Error Code: 1308')).toBeVisible();
    await expect(page.getByText(/Time Until Reset/)).toBeVisible();
    await expect(page.getByText(/Request ID: req_test_12345/)).toBeVisible();
  });

  test('should display countdown timer', async ({ page }) => {
    const resetTime = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
    await page.evaluate((time) => {
      localStorage.setItem('usage_limit_code', '1308');
      localStorage.setItem('usage_limit_message', 'Usage limit reached');
      localStorage.setItem('usage_limit_reset_at', time);
    }, resetTime);

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Get the countdown timer element
    const countdownElement = page.locator('.text-3xl.font-mono.font-bold');
    await expect(countdownElement).toBeVisible();

    // Get initial time
    const initialTime = await countdownElement.textContent();
    expect(initialTime).toBeTruthy();

    // Wait 2 seconds and check that time has decreased
    await page.waitForTimeout(2000);
    const updatedTime = await countdownElement.textContent();
    expect(updatedTime).toBeTruthy();
    expect(updatedTime).not.toBe(initialTime);
  });

  test('should show formatted reset time', async ({ page }) => {
    const specificResetTime = '2026-02-25T19:49:14Z';
    await page.evaluate((time) => {
      localStorage.setItem('usage_limit_code', '1308');
      localStorage.setItem('usage_limit_message', 'Usage limit reached');
      localStorage.setItem('usage_limit_reset_at', time);
    }, specificResetTime);

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Should show the formatted reset time
    await expect(page.getByText(/Resets at:/)).toBeVisible();
  });

  test('should handle missing localStorage data gracefully', async ({ page }) => {
    // Navigate without setting localStorage data
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Should still show the page with default values
    await expect(page.getByText('Usage Limit Reached')).toBeVisible();
    await expect(page.getByText('Error Code: 1308')).toBeVisible();
  });

  test('should have retry button', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('usage_limit_code', '1308');
      localStorage.setItem('usage_limit_message', 'Usage limit reached');
      localStorage.setItem('usage_limit_reset_at', new Date(Date.now() + 3600000).toISOString());
    });

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    const retryButton = page.getByRole('button', { name: /Retry Now/ });
    await expect(retryButton).toBeVisible();

    // Click retry - should navigate to home
    await retryButton.click();
    await page.waitForTimeout(500);

    // Should clear localStorage and navigate
    const storedCode = await page.evaluate(() => localStorage.getItem('usage_limit_code'));
    expect(storedCode).toBeNull();
  });

  test('should have reload page button', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('usage_limit_code', '1308');
      localStorage.setItem('usage_limit_message', 'Usage limit reached');
      localStorage.setItem('usage_limit_reset_at', new Date(Date.now() + 3600000).toISOString());
    });

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    const reloadButton = page.getByRole('button', { name: /Reload Page/ });
    await expect(reloadButton).toBeVisible();
  });

  test('should display help text about upgrading', async ({ page }) => {
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText(/Need higher limits?/)).toBeVisible();
    await expect(page.getByText(/Contact your administrator/)).toBeVisible();
  });

  test('should display about usage limits information card', async ({ page }) => {
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('About Usage Limits')).toBeVisible();
    await expect(page.getByText(/Usage limits help ensure fair resource allocation/)).toBeVisible();
  });

  test('should handle location state for error details', async ({ page }) => {
    // Navigate with state instead of localStorage
    const resetTime = new Date(Date.now() + 3600000).toISOString();

    await page.evaluate((time) => {
      // Navigate with state
      window.history.pushState(
        {
          errorCode: 'USAGE_LIMIT_EXCEEDED',
          errorMessage: 'API rate limit exceeded. Please try again later.',
          resetAt: time,
          requestId: 'req_state_98765',
          from: '/dashboard'
        },
        '',
        '/usage-limit'
      );
    }, resetTime);

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Should display the state-provided error code
    await expect(page.getByText('Error Code: USAGE_LIMIT_EXCEEDED')).toBeVisible();
    await expect(page.getByText(/API rate limit exceeded/)).toBeVisible();
    await expect(page.getByText(/req_state_98765/)).toBeVisible();
  });
});

test.describe('Usage Limit Inline Component', () => {
  test('should render inline usage limit error', async ({ page }) => {
    // This test would be used if the inline component is used elsewhere
    // For now, we're testing the page directly
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Check that warning colors are present (using the inline styling)
    const warningBadge = page.locator('.bg-warning-500\\/10, .text-warning-500');
    await expect(warningBadge.first()).toBeVisible();
  });
});

test.describe('Usage Limit API Error Handling', () => {
  test('should redirect to usage limit page on API error code 1308', async ({ page }) => {
    // Mock an API response that returns usage limit error
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: '1308',
            message: 'Usage limit reached for 5 hour. Your limit will reset at 2026-02-25 19:49:14',
          },
          request_id: 'req_api_12345',
          market_context: {
            competitors_analyzed: [],
            key_gaps: [],
            trends: [],
            unique_selling_points: [],
          },
        }),
      });
    });

    // Set auth token first
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
      localStorage.setItem('refresh_token', 'test-refresh');
    });

    // Navigate to a page that would trigger an API call
    await page.goto('/dashboard');

    // Should be redirected to usage limit page
    await page.waitForTimeout(1000); // Wait for redirect
    const url = page.url();
    expect(url).toContain('/usage-limit');
  });

  test('should redirect to usage limit page on 429 status', async ({ page }) => {
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: 'RATE_LIMIT_EXCEEDED',
            message: 'Too many requests',
            details: {
              reset_at: new Date(Date.now() + 3600000).toISOString(),
            },
          },
          request_id: 'req_429_test',
        }),
      });
    });

    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
      localStorage.setItem('refresh_token', 'test-refresh');
    });

    await page.goto('/dashboard');

    await page.waitForTimeout(1000);
    const url = page.url();
    expect(url).toContain('/usage-limit');
  });

  test('should extract reset time from error message when not in details', async ({ page }) => {
    const resetTimeStr = '2026-02-25 19:49:14';
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({
          error: {
            code: '1308',
            message: `Usage limit reached. Your limit will reset at ${resetTimeStr}`,
          },
          request_id: 'req_extract_test',
        }),
      });
    });

    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('access_token', 'test-token');
      localStorage.setItem('refresh_token', 'test-refresh');
    });

    await page.goto('/dashboard');

    await page.waitForTimeout(1000);
    expect(page.url()).toContain('/usage-limit');

    // The reset time should be extracted and stored
    const storedResetTime = await page.evaluate(() => localStorage.getItem('usage_limit_reset_at'));
    expect(storedResetTime).toBeTruthy();
  });
});

test.describe('Usage Limit Visual Design', () => {
  test('should display clock icon', async ({ page }) => {
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Check for SVG icon (clock icon)
    const clockIcon = page.locator('svg').filter({ hasText: '' }).first();
    await expect(clockIcon).toBeVisible();
  });

  test('should have proper color scheme for warning state', async ({ page }) => {
    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Check for warning color classes
    const warningElements = page.locator('.text-warning-500');
    await expect(warningElements.first()).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    await page.goto('/usage-limit');
    await page.waitForLoadState('networkidle');

    // Main content should still be visible
    await expect(page.getByText('Usage Limit Reached')).toBeVisible();

    // Buttons should stack vertically on mobile
    const buttons = page.locator('.btn');
    await expect(buttons.first()).toBeVisible();
  });
});
