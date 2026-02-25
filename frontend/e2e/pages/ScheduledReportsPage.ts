/**
 * Page Object for Scheduled Reports Page
 */

import { expect, type Page } from '@playwright/test';

export class ScheduledReportsPage {
  readonly page: Page;

  // Locators
  readonly heading = this.page.getByRole('heading', { name: /scheduled reports/i });
  readonly newScheduleButton = this.page.getByRole('button', { name: /new schedule/i });
  readonly searchInput = this.page.getByPlaceholder(/search schedules/i);
  readonly frameworkFilter = this.page.locator('select').filter({ hasText: /all frameworks/i });
  readonly statusFilter = this.page.locator('select').filter({ hasText: /all status/i });

  constructor(page: Page) {
    this.page = page;
  }

  async goto() {
    await this.page.goto('/reports/scheduled');
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

  async createNewSchedule() {
    await this.newScheduleButton.click();
  }

  async searchSchedules(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async filterByFramework(framework: string) {
    await this.frameworkFilter.selectOption({ label: framework });
    await this.page.waitForTimeout(500);
  }

  async filterByStatus(status: string) {
    await this.statusFilter.selectOption({ label: status });
    await this.page.waitForTimeout(500);
  }

  async getScheduleCount(): Promise<number> {
    const rows = await this.page.locator('[class*="schedule"]').count();
    return rows;
  }

  async toggleScheduleStatus(scheduleName: string) {
    const row = this.page.getByText(scheduleName).first();
    await row.locator('button').filter({ hasText: /pause|resume|play/i }).first().click();
  }

  async runScheduleNow(scheduleName: string) {
    const row = this.page.getByText(scheduleName).first();
    await row.locator('button[title="Run now"]').click();
  }

  async editSchedule(scheduleName: string) {
    const row = this.page.getByText(scheduleName).first();
    await row.locator('button[title="Edit"]').click();
  }

  async deleteSchedule(scheduleName: string) {
    const row = this.page.getByText(scheduleName).first();
    await row.locator('button[title="Delete"]').click();
    await this.page.getByRole('button', { name: /confirm/i }).click();
  }

  async toggleScheduleHistory(scheduleName: string) {
    const row = this.page.getByText(scheduleName).first();
    await row.locator('button').filter({ hasText: /hide|show/i }).click();
  }

  async getScheduleNames(): Promise<string[]> {
    const names: string[] = [];
    const elements = await this.page.locator('[class*="schedule"] h3, [class*="schedule"] .font-semibold').all();
    for (const el of elements) {
      const text = await el.textContent();
      if (text) names.push(text);
    }
    return names;
  }

  async viewExecutionHistory(scheduleName: string): Promise<number> {
    await this.toggleScheduleHistory(scheduleName);
    await this.page.waitForTimeout(500);
    const executions = await this.page.locator('[class*="execution"]').count();
    return executions;
  }
}
