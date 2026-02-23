import { Page, Locator } from '@playwright/test';

export class DashboardPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly statsCards: Locator;
  readonly activityFeed: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { level: 1 }).first();
    // Look for stat cards by their title text
    this.statsCards = page.locator('.card').filter({ hasText: /Total Users|Active Targets|Active Sessions|Pending Requests/ });
    this.activityFeed = page.getByText('Recent Activity');
  }

  async goto() {
    await this.page.goto('/dashboard');
  }

  async getHeadingText() {
    await this.heading.waitFor({ state: 'visible' });
    return this.heading.textContent();
  }

  async getStatsCount() {
    return this.statsCards.count();
  }
}
