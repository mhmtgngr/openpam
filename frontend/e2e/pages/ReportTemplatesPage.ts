/**
 * Page Object for Report Templates Page
 */

import { expect, type Page } from '@playwright/test';

export class ReportTemplatesPage {
  readonly page: Page;

  // Locators
  readonly heading = this.page.getByRole('heading', { name: /report templates/i });
  readonly newTemplateButton = this.page.getByRole('button', { name: /new template/i });
  readonly searchInput = this.page.getByPlaceholder(/search templates/i);
  readonly typeFilter = this.page.locator('select').filter({ hasText: /all types/i });
  readonly frameworkFilter = this.page.locator('select').filter({ hasText: /all frameworks/i });

  constructor(page: Page) {
    this.page = page;
  }

  async goto() {
    await this.page.goto('/reports/templates');
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

  async createNewTemplate() {
    await this.newTemplateButton.click();
  }

  async searchTemplates(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  async filterByType(type: string) {
    await this.typeFilter.selectOption({ label: type });
    await this.page.waitForTimeout(500);
  }

  async filterByFramework(framework: string) {
    await this.frameworkFilter.selectOption({ label: framework });
    await this.page.waitForTimeout(500);
  }

  async getTemplateCount(): Promise<number> {
    const cards = await this.page.locator('.template-card, [class*="template"]').count();
    return cards;
  }

  async editTemplate(templateName: string) {
    const card = this.page.getByText(templateName).first();
    await card.locator('button[title="Edit"]').click();
  }

  async duplicateTemplate(templateName: string) {
    const card = this.page.getByText(templateName).first();
    await card.locator('button[title="Duplicate"]').click();
  }

  async deleteTemplate(templateName: string) {
    const card = this.page.getByText(templateName).first();
    await card.locator('button[title="Delete"]').click();
    await this.page.getByRole('button', { name: /confirm/i }).click();
  }

  async selectTemplate(templateName: string) {
    await this.page.getByText(templateName).first().click();
  }

  async getTemplateNames(): Promise<string[]> {
    const names: string[] = [];
    const elements = await this.page.locator('.template-card h3, [class*="template"] h3').all();
    for (const el of elements) {
      const text = await el.textContent();
      if (text) names.push(text);
    }
    return names;
  }
}
