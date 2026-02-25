import { test, expect } from './fixtures/auth.fixture';

test.describe('Debug Policy Test Page', () => {
  test('debug - check what buttons exist', async ({ accessPolicyPage }) => {
    await accessPolicyPage.goto('/policies/access/test');
    await accessPolicyPage.waitForLoadState('networkidle');

    // List all buttons
    const buttons = await accessPolicyPage.locator('button').all();
    console.log('Total buttons:', buttons.length);

    for (let i = 0; i < Math.min(buttons.length, 20); i++) {
      const button = buttons[i];
      const text = await button.textContent();
      const innerHTML = await button.innerHTML();
      console.log(`Button ${i}: text="${text}", html="${innerHTML}"`);
    }

    // Take screenshot
    await accessPolicyPage.screenshot({ path: 'debug-buttons.png' });
  });

  test('debug - try different selectors', async ({ accessPolicyPage }) => {
    await accessPolicyPage.goto('/policies/access/test');
    await accessPolicyPage.waitForLoadState('networkidle');

    // Try to find button with + or expand
    const plusTextButtons = accessPolicyPage.locator('button').filter({ hasText: '+' });
    const count1 = await plusTextButtons.count();
    console.log('Buttons with hasText("+"):', count1);

    const plusRegexButtons = accessPolicyPage.locator('button').filter({ hasText: /\+/ });
    const count2 = await plusRegexButtons.count();
    console.log('Buttons with hasText(/\\+/):', count2);

    // Try with span
    const spanWithPlus = accessPolicyPage.locator('span').filter({ hasText: '+' });
    const count3 = await spanWithPlus.count();
    console.log('Spans with +:', count3);

    // Get all text content of page
    const pageText = await accessPolicyPage.textContent('body');
    console.log('Page contains +:', pageText?.includes('+'));
    console.log('Page contains + as HTML:', pageText?.includes('+'));

    // Take screenshot
    await accessPolicyPage.screenshot({ path: 'debug-selectors.png' });
  });
});
