import { type Page, Locator } from '@playwright/test';

export class ReportGeneratorPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly reportNameInput: Locator;
  readonly backButton: Locator;
  readonly nextButton: Locator;
  readonly cancelButton: Locator;
  readonly generateButton: Locator;
  readonly frameworkCards: Locator;
  readonly periodStartInput: Locator;
  readonly periodEndInput: Locator;
  readonly formatSelect: Locator;
  readonly stepIndicators: Locator;
  readonly scheduleToggle: Locator;

  constructor(page: Page) {
    this.page = page;
    // Heading
    this.heading = page.getByRole('heading', { level: 1, name: /generate compliance report/i });
    // Inputs
    this.reportNameInput = page.getByLabel(/report name/i);
    this.periodStartInput = page.getByLabel(/start date/i);
    this.periodEndInput = page.getByLabel(/end date/i);
    this.formatSelect = page.locator('select#format').or(page.getByLabel(/output format/i));
    // Buttons
    this.backButton = page.getByRole('button', { name: /back/i });
    this.nextButton = page.getByRole('button', { name: /next/i });
    this.cancelButton = page.getByRole('button', { name: /cancel/i });
    this.generateButton = page.getByRole('button', { name: /generate report/i });
    // Step elements
    this.frameworkCards = page.locator('button').filter({ hasText: /SOC 2|ISO 27001|PCI DSS|HIPAA|GDPR/i });
    this.stepIndicators = page.locator('[class*="step"]').or(page.locator('button').filter({ hasText: /basic info|content|review/i }));
    // Toggle
    this.scheduleToggle = page.locator('button').filter({ hasText: /schedule this report/i }).or(page.getByRole('switch', { name: /schedule/i }));
  }

  async goto() {
    await this.page.goto('/reports/generate');
    await this.page.waitForLoadState('networkidle');
  }

  async waitForLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.heading.waitFor({ state: 'visible' }).catch(() => {});
  }

  async getCurrentStep(): Promise<number> {
    // Step 1 = Basic Info, Step 2 = Content, Step 3 = Review
    const basicInfoActive = await this.page.locator('text=/Basic Info/').isVisible().catch(() => false);
    const contentActive = await this.page.locator('text=/Content/').isVisible().catch(() => false);
    const reviewActive = await this.page.locator('text=/Review/').isVisible().catch(() => false);

    if (basicInfoActive) return 1;
    if (contentActive) return 2;
    if (reviewActive) return 3;
    return 1;
  }

  async fillReportName(name: string) {
    await this.reportNameInput.fill(name);
  }

  async selectFramework(framework: string) {
    const frameworkCard = this.frameworkCards.filter({ hasText: framework });
    await frameworkCard.click();
  }

  async setPeriod(startDate: string, endDate: string) {
    await this.periodStartInput.fill(startDate);
    await this.periodEndInput.fill(endDate);
  }

  async selectPeriodPreset(presetName: string) {
    await this.page.getByRole('button', { name: presetName }).click();
  }

  async selectFormat(format: string) {
    await this.formatSelect.selectOption(format);
  }

  async toggleSchedule(enabled: boolean) {
    const currentState = await this.isScheduleEnabled();
    if (currentState !== enabled) {
      await this.scheduleToggle.click();
    }
  }

  async isScheduleEnabled(): Promise<boolean> {
    // Check if toggle is in checked state
    const toggle = this.scheduleToggle;
    const classList = await toggle.getAttribute('class') || '';
    return classList.includes('bg-primary-500') || classList.includes('checked');
  }

  async toggleSection(sectionName: string) {
    const section = this.page.locator('label').filter({ hasText: sectionName });
    await section.click();
  }

  async isSectionSelected(sectionName: string): Promise<boolean> {
    const section = this.page.locator('label').filter({ hasText: sectionName });
    const checkbox = section.locator('input[type="checkbox"]');
    const isChecked = await checkbox.isChecked();
    return isChecked;
  }

  async toggleFilter(category: string, value: string) {
    const filter = this.page.locator('button').filter({ hasText: value })
      .filter({ has: this.page.locator(`text=/${category}/i`) });
    await filter.click();
  }

  async toggleOption(optionName: string) {
    const optionLabel = this.page.locator('label').filter({ hasText: new RegExp(optionName, 'i') });
    const toggle = optionLabel.locator('button, [role="switch"]');
    await toggle.click();
  }

  async clickBack() {
    await this.backButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickNext() {
    await this.nextButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickCancel() {
    await this.cancelButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickGenerate() {
    await this.generateButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async getReviewSummary(): Promise<{
    name: string;
    framework: string;
    period: string;
    format: string;
    sections: string[];
  }> {
    const reviewCard = this.page.locator('.card').filter({ hasText: /review report configuration/i });

    const getText = async (label: string) => {
      const element = reviewCard.locator(`text=/${label}/i`).first();
      if (await element.isVisible().catch(() => false)) {
        return await element.textContent() || '';
      }
      return '';
    };

    const framework = await getText('Framework');
    const format = await getText('Format');
    const period = await getText('Period');

    // Get selected sections
    const sectionBadges = this.page.locator('[class*="badge"]').or(this.page.locator('.rounded-full'));
    const sections: string[] = [];
    const count = await sectionBadges.count();
    for (let i = 0; i < count; i++) {
      const badge = sectionBadges.nth(i);
      const text = await badge.textContent();
      if (text) sections.push(text.trim());
    }

    return {
      name: '',
      framework: framework.replace('Framework', '').trim(),
      period: period.replace('Period', '').trim(),
      format: format.replace('Format', '').trim().toUpperCase(),
      sections,
    };
  }

  async isAtFinalStep(): Promise<boolean> {
    return await this.page.getByText(/review & generate/i).isVisible().catch(() => false);
  }

  async hasValidationError(message: string): Promise<boolean> {
    return await this.page.getByText(message).isVisible().catch(() => false);
  }
}
