import { type Page, Locator } from '@playwright/test';

export class AnalyticsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly quickStatCards: Locator;
  readonly anomalyList: Locator;
  readonly userActivityTable: Locator;
  readonly commandAnalysisTable: Locator;

  constructor(page: Page) {
    this.page = page;
    // The page uses h1 with text "Analytics"
    this.heading = page.getByRole('heading', { level: 1, name: /analytics/i });
    // Tabs are buttons in a nav element with IDs like tab-overview
    this.tabs = page.locator('nav button');
    // Quick stat cards - each has class .card and contains the title text
    this.quickStatCards = page.locator('.card').filter({ hasText: /^(Active Sessions|Active Users|High-Risk Commands|Open Anomalies)/i });
    // Anomaly list - any card containing "anomaly"
    this.anomalyList = page.locator('.card').filter({ hasText: /anomaly/i });
    // Tables
    this.userActivityTable = page.locator('table').filter({ hasText: /user/i });
    this.commandAnalysisTable = page.locator('table').filter({ hasText: /command/i });
  }

  async goto() {
    await this.page.goto('/analytics');
    await this.page.waitForLoadState('networkidle');
  }

  async getHeadingText(): Promise<string | null> {
    const heading = this.heading.first();
    if (await heading.isVisible().catch(() => false)) {
      return await heading.textContent();
    }
    return null;
  }

  async getTabsCount(): Promise<number> {
    return await this.tabs.count();
  }

  async getTabById(tabId: string): Promise<Locator> {
    return this.page.locator(`button#tab-${tabId}`);
  }

  async clickTab(tabName: string) {
    // Map tab display names to their IDs
    const tabIdMap: Record<string, string> = {
      'Overview': 'overview',
      'Session Metrics': 'sessions',
      'User Activity': 'users',
      'Command Analysis': 'commands',
      'Compliance': 'compliance',
      'Anomalies': 'anomalies',
    };

    const tabId = tabIdMap[tabName] || tabName.toLowerCase();
    const tab = this.page.locator(`button#tab-${tabId}`);

    // Wait for tab to be visible and click it
    await tab.waitFor({ state: 'visible', timeout: 5000 });
    await tab.click();
    await this.page.waitForLoadState('networkidle');
  }

  async isTabActive(tabName: string): Promise<boolean> {
    const tabIdMap: Record<string, string> = {
      'Overview': 'overview',
      'Session Metrics': 'sessions',
      'User Activity': 'users',
      'Command Analysis': 'commands',
      'Compliance': 'compliance',
      'Anomalies': 'anomalies',
    };

    const tabId = tabIdMap[tabName] || tabName.toLowerCase();
    const tab = this.page.locator(`button#tab-${tabId}`);

    if (await tab.count() === 0) {
      return false;
    }

    const className = await tab.getAttribute('class');
    return className?.includes('text-primary-400') || className?.includes('border-primary-500') || false;
  }

  async getQuickStatsCount(): Promise<number> {
    return await this.quickStatCards.count();
  }

  async getStatValue(statName: string): Promise<string | null> {
    const card = this.page.locator('.card').filter({ hasText: new RegExp(statName, 'i') }).first();
    const valueElement = card.locator('.text-2xl').first();
    if (await valueElement.isVisible().catch(() => false)) {
      return await valueElement.textContent();
    }
    return null;
  }

  async getComplianceScore(): Promise<string | null> {
    const scoreElement = this.page.locator('text=/\\d{1,3}%/').first();
    if (await scoreElement.isVisible().catch(() => false)) {
      return await scoreElement.textContent();
    }
    return null;
  }

  async getTableRowCount(tableSelector: string): Promise<number> {
    const table = this.page.locator(tableSelector).first();
    const rows = table.locator('tbody tr');
    return await rows.count();
  }

  async waitForContent(): Promise<void> {
    await Promise.race([
      this.quickStatCards.first().waitFor({ state: 'visible' }).catch(() => {}),
      this.page.waitForSelector('[data-testid="loading-state"]', { state: 'hidden' }).catch(() => {}),
      this.page.waitForTimeout(2000),
    ]);
  }
}
