/**
 * Page Object for Compliance Dashboard Page
 */

import { expect, type Page } from '@playwright/test';

export class ComplianceDashboardPage {
  readonly page: Page;

  // Locators
  readonly heading = this.page.getByRole('heading', { name: /compliance dashboard/i });
  readonly frameworkSelector = this.page.getByRole('button', { name: /soc 2|iso 27001|pci dss/i });
  readonly refreshButton = this.page.getByRole('button', { name: /refresh/i });
  readonly exportButton = this.page.getByRole('button', { name: /export report/i });
  readonly overallScoreGauge = this.page.locator('.compliance-gauge');
  readonly compliantStatCard = this.page.locator('text=Compliant').first();
  readonly nonCompliantStatCard = this.page.locator('text=Non-Compliant').first();
  readonly exceptionsStatCard = this.page.locator('text=Exceptions').first();

  constructor(page: Page) {
    this.page = page;
  }

  async goto() {
    await this.page.goto('/compliance/dashboard');
    await this.waitForLoad();
  }

  async waitForLoad() {
    await this.page.waitForLoadState('networkidle');
    await expect(this.heading).toBeVisible();
  }

  async getHeadingText(): Promise<string> {
    const heading = await this.heading.textContent();
    return heading || '';
  }

  async selectFramework(framework: string) {
    await this.frameworkSelector.click();
    await this.page.getByRole('button', { name: framework }).click();
    await this.page.waitForTimeout(500);
  }

  async refresh() {
    await this.refreshButton.click();
    await this.page.waitForTimeout(1000);
  }

  async getOverallScore(): Promise<number> {
    const gaugeText = await this.overallScoreGauge.textContent();
    const match = gaugeText?.match(/(\d+)%/);
    return match ? parseInt(match[1], 10) : 0;
  }

  async getStatCount(statName: string): Promise<number> {
    const stat = this.page.locator(`text=/${statName}/i`).first();
    const text = await stat.textContent();
    const match = text?.match(/\d+/);
    return match ? parseInt(match[0], 10) : 0;
  }

  async viewCategoryDetails(categoryName: string) {
    await this.page.getByText(categoryName).click();
  }

  async exportReport() {
    await this.exportButton.click();
  }
}
