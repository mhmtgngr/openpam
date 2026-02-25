import { test, expect } from './fixtures/auth.fixture';
import { AnomalyListPage } from './pages/AnomalyListPage';

test('debug anomaly page - with console capture', async ({ authenticatedPage }) => {
  const page = authenticatedPage;

  // Capture console messages
  const logs: string[] = [];
  page.on('console', msg => {
    logs.push(`${msg.type()}: ${msg.text()}`);
  });

  // Capture errors
  const errors: string[] = [];
  page.on('pageerror', error => {
    errors.push(error.toString());
  });

  // Log page state
  console.log('URL after auth:', page.url());

  // Check localStorage
  const token = await page.evaluate(() => localStorage.getItem('access_token'));
  console.log('Token exists:', !!token);

  // Navigate to anomalies
  await page.goto('/analytics/anomalies');

  // Wait for potential rendering
  await page.waitForTimeout(3000);

  console.log('URL after navigation:', page.url());

  // Print console logs
  console.log('Console logs:', logs);
  console.log('Page errors:', errors);

  // Check if React app is mounted
  const reactRoot = await page.evaluate(() => {
    const root = document.getElementById('root');
    return {
      exists: !!root,
      innerHTML: root ? root.innerHTML.substring(0, 500) : null,
      childCount: root ? root.childElementCount : 0,
    };
  });
  console.log('React root:', reactRoot);

  // Try navigating to dashboard to see if it works
  await page.goto('/dashboard');
  await page.waitForTimeout(2000);

  const dashboardRoot = await page.evaluate(() => {
    const root = document.getElementById('root');
    return {
      exists: !!root,
      innerHTML: root ? root.innerHTML.substring(0, 500) : null,
      childCount: root ? root.childElementCount : 0,
    };
  });
  console.log('Dashboard root:', dashboardRoot);
});
