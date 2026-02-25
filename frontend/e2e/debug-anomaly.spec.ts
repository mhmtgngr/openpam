import { test } from './fixtures/auth.fixture';

test('debug anomaly page', async ({ authenticatedPage }) => {
  const page = authenticatedPage;

  // Navigate to anomalies page
  await page.goto('/analytics/anomalies');
  await page.waitForLoadState('domcontentloaded').catch(() => {});

  // Wait a bit
  await page.waitForTimeout(2000);

  // Get page content
  const url = page.url();
  console.log('URL:', url);

  // Check localStorage
  const localStorage = await page.evaluate(() => {
    return {
      accessToken: localStorage.getItem('access_token'),
      refreshToken: localStorage.getItem('refresh_token'),
    };
  });
  console.log('LocalStorage:', localStorage);

  // Get HTML body
  const bodyHtml = await page.locator('body').innerHTML();
  console.log('Body HTML length:', bodyHtml.length);
  console.log('Body HTML preview:', bodyHtml.substring(0, 500));

  // Check for React root
  const reactRoot = await page.locator('#root').count();
  console.log('React root count:', reactRoot);

  // Check for any text
  const allText = await page.locator('body').textContent();
  console.log('All text length:', allText?.length || 0);
  console.log('All text preview:', allText?.substring(0, 200) || 'empty');

  // Look for any error messages
  const hasError = await page.getByText(/error|failed|404|unauthorized/i).count();
  console.log('Error messages count:', hasError);

  // Check for h1 elements
  const h1Count = await page.locator('h1').count();
  console.log('H1 count:', h1Count);
  for (let i = 0; i < Math.min(h1Count, 5); i++) {
    const text = await page.locator('h1').nth(i).textContent();
    console.log(`H1[${i}]:`, text);
  }

  // Check for routes issues - maybe we're being redirected
  await page.waitForTimeout(2000);
  const finalUrl = page.url();
  console.log('Final URL:', finalUrl);

  // Take screenshot
  await page.screenshot({ path: 'debug-screenshot.png', fullPage: true });
  console.log('Screenshot saved to debug-screenshot.png');
});
