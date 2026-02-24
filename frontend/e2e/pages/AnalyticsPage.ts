import { type Page, Locator } from '@playwright/test';

export class AnalyticsPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly overviewTab: Locator;
  readonly sessionsTab: Locator;
  readonly usersTab: Locator;
  readonly commandsTab: Locator;
  readonly complianceTab: Locator;
  readonly anomaliesTab: Locator;
  readonly activeSessionsStat: Locator;
  readonly quickStatCards: Locator;
  readonly sessionMetricsCard: Locator;
  readonly userActivityTable: Locator;
  readonly commandAnalysisTable: Locator;
  readonly complianceScore: Locator;
  readonly anomalyList: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { name: /analytics/i });
    this.tabs = page.locator('nav button');
    this.overviewTab = page.getByRole('button', { name: /overview/i });
    this.sessionsTab = page.getByRole('button', { name: /session metrics/i });
    this.usersTab = page.getByRole('button', { name: /user activity/i });
    this.commandsTab = page.getByRole('button', { name: /command analysis/i });
    this.complianceTab = page.getByRole('button', { name: /compliance/i });
    this.anomaliesTab = page.getByRole('button', { name: /anomalies/i });
    this.activeSessionsStat = page.getByText(/active sessions/i);
    this.quickStatCards = page.locator('.card').filter({ hasText: /^(Active Sessions|Active Users|High-Risk Commands|Open Anomalies)$/i });
    this.sessionMetricsCard = page.locator('.card').filter({ hasText: /session metrics/i });
    this.userActivityTable = page.getByRole('table').filter({ hasText: /user/i });
    this.commandAnalysisTable = page.getByRole('table').filter({ hasText: /command/i });
    this.complianceScore = page.locator('text=/%').first();
    this.anomalyList = page.locator('.card').filter({ hasText: /anomaly/i });
  }

  async goto() {
    await this.page.goto('/analytics');
    await this.page.waitForLoadState('networkidle');
  }

  async getHeadingText(): Promise<string | null> {
    const heading = this.heading.first();
    if (await heading.isVisible()) {
      return await heading.textContent();
    }
    return null;
  }

  async getTabsCount(): Promise<number> {
    return await this.tabs.count();
  }

  async clickTab(tabName: string) {
    await this.page.getByRole('button', { name: tabName, exact: true }).click();
    await this.page.waitForLoadState('networkidle');
  }

  async isTabActive(tabName: string): Promise<boolean> {
    const tab = this.page.getByRole('button', { name: tabName, exact: true });
    const className = await tab.getAttribute('class');
    return className?.includes('text-primary-400') || className?.includes('border-primary-500') || false;
  }

  async getQuickStatsCount(): Promise<number> {
    return await this.quickStatCards.count();
  }

  async getComplianceScore(): Promise<string | null> {
    const scoreElement = this.complianceScore.first();
    if (await scoreElement.isVisible()) {
      return await scoreElement.textContent();
    }
    return null;
  }
}
