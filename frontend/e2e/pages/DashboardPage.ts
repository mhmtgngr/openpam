import { Page, Locator } from '@playwright/test';

export class DashboardPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly statsCards: Locator;
  readonly activityFeed: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.locator('h1');
    this.statsCards = page.locator('.card').filter({ hasText: /^(Total Users|Active Targets|Active Sessions|Pending Requests)$/ });
    this.activityFeed = page.locator('text=Recent Activity');
  }

  async goto() {
    await this.page.goto('/dashboard');
  }

  async getHeadingText() {
    return this.heading.textContent();
  }

  async getStatsCount() {
    return this.statsCards.count();
  }
}
