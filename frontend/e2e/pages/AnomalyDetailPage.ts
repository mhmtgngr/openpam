import { type Page, Locator } from '@playwright/test';

export class AnomalyDetailPage {
  readonly page: Page;
  readonly heading: Locator;
  readonly backButton: Locator;
  readonly severityBadge: Locator;
  readonly statusBadge: Locator;
  readonly acknowledgeButton: Locator;
  readonly updateStatusButton: Locator;
  readonly saveChangesButton: Locator;
  readonly cancelButton: Locator;
  readonly basicInfoCard: Locator;
  readonly indicatorsCard: Locator;
  readonly statusAssignmentCard: Locator;
  readonly relatedEntitiesCard: Locator;
  readonly typeInfoCard: Locator;
  readonly statusSelect: Locator;
  readonly assignedToInput: Locator;
  readonly resolutionNotesTextarea: Locator;
  readonly indicatorCards: Locator;

  constructor(page: Page) {
    this.page = page;
    this.heading = page.getByRole('heading', { level: 1 })
      .or(page.locator('h1'))
      .or(page.getByText(/unusual/i));
    // Back button - look for button with chevron-left icon
    this.backButton = page.locator('button svg.lucide-chevron-left').locator('..');
    this.severityBadge = page.locator('[class*="text-danger-400"], [class*="text-warning-400"], [class*="text-success-400"]');
    this.statusBadge = page.locator('[class*="badge"], .rounded-full');
    this.acknowledgeButton = page.locator('button', { hasText: /acknowledge/i });
    this.updateStatusButton = page.locator('button', { hasText: /update status/i });
    this.saveChangesButton = page.locator('button', { hasText: /save changes/i });
    this.cancelButton = page.locator('button', { hasText: /^cancel$/i });
    this.basicInfoCard = page.locator('.card').filter({ hasText: /basic information/i });
    this.indicatorsCard = page.locator('.card').filter({ hasText: /detection indicators/i });
    this.statusAssignmentCard = page.locator('.card').filter({ hasText: /status.*assignment/i });
    this.relatedEntitiesCard = page.locator('.card').filter({ hasText: /related entities/i });
    this.typeInfoCard = page.locator('.card').filter({ hasText: /anomaly type/i });
    this.statusSelect = page.locator('.card').filter({ hasText: /status.*assignment/i }).locator('select');
    this.assignedToInput = page.locator('input[placeholder*="email" i], input[placeholder*="user id" i]');
    this.resolutionNotesTextarea = page.locator('textarea[placeholder*="investigation" i], textarea[placeholder*="resolution" i]');
    this.indicatorCards = page.locator('[class*="bg-gray-900"]').filter({ hasText: /confidence/i });
  }

  async goto(anomalyId: string) {
    await this.page.goto(`/analytics/anomalies/${anomalyId}`);
    await this.page.waitForLoadState('domcontentloaded').catch(() => {});
    await this.page.waitForTimeout(500);
  }

  async getHeadingText(): Promise<string | null> {
    await this.page.waitForTimeout(500);
    const elements = await this.heading.all();
    for (const el of elements) {
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text) return text;
      }
    }
    return null;
  }

  async getAnomalyTitle(): Promise<string | null> {
    const title = this.page.locator('h1.text-xl, .text-xl.font-bold, h1');
    const count = await title.count();
    for (let i = 0; i < count; i++) {
      const el = title.nth(i);
      if (await el.isVisible().catch(() => false)) {
        return await el.textContent();
      }
    }
    return null;
  }

  async getAnomalyDescription(): Promise<string | null> {
    const description = this.page.locator('p.text-sm.text-gray-400, p.text-gray-400');
    const count = await description.count();
    for (let i = 0; i < count; i++) {
      const el = description.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text && text.length > 20) {
          return text;
        }
      }
    }
    return null;
  }

  async getSeverity(): Promise<string | null> {
    const severityText = this.page.getByText(/(critical|high|medium|low)/i);
    const count = await severityText.count();
    for (let i = 0; i < Math.min(count, 5); i++) {
      const el = severityText.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text) {
          const match = text.match(/(critical|high|medium|low)/i);
          if (match) return match[1].toLowerCase();
        }
      }
    }
    return null;
  }

  async getStatus(): Promise<string | null> {
    const statusText = this.page.getByText(/(open|investigating|resolved|false positive)/i);
    const count = await statusText.count();
    for (let i = 0; i < Math.min(count, 5); i++) {
      const el = statusText.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text) {
          const match = text.match(/(open|investigating|resolved|false positive)/i);
          if (match) return match[1].toLowerCase();
        }
      }
    }
    return null;
  }

  async getConfidenceScore(): Promise<number | null> {
    const confidenceText = this.page.getByText(/\d+%/);
    const count = await confidenceText.count();
    for (let i = 0; i < Math.min(count, 3); i++) {
      const el = confidenceText.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const text = await el.textContent();
        if (text) {
          const match = text.match(/(\d+)%/);
          if (match) return parseInt(match[1], 10);
        }
      }
    }
    return null;
  }

  async clickAcknowledge() {
    const count = await this.acknowledgeButton.count();
    if (count > 0) {
      await this.acknowledgeButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickUpdateStatus() {
    const count = await this.updateStatusButton.count();
    if (count > 0) {
      await this.updateStatusButton.first().click();
    }
  }

  async clickSaveChanges() {
    const count = await this.saveChangesButton.count();
    if (count > 0) {
      await this.saveChangesButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async clickCancel() {
    const count = await this.cancelButton.count();
    if (count > 0) {
      await this.cancelButton.first().click();
    }
  }

  async clickBack() {
    const count = await this.backButton.count();
    if (count > 0) {
      await this.backButton.first().click();
      await this.page.waitForTimeout(500);
    }
  }

  async setStatus(status: 'open' | 'investigating' | 'resolved' | 'false_positive') {
    await this.statusSelect.selectOption(status);
  }

  async setAssignedTo(value: string) {
    const count = await this.assignedToInput.count();
    if (count > 0) {
      await this.assignedToInput.first().fill(value);
    }
  }

  async setResolutionNotes(notes: string) {
    const count = await this.resolutionNotesTextarea.count();
    if (count > 0) {
      await this.resolutionNotesTextarea.first().fill(notes);
    }
  }

  async getSelectedStatus(): Promise<string | null> {
    const count = await this.statusSelect.count();
    if (count > 0) {
      return await this.statusSelect.first().inputValue();
    }
    return null;
  }

  async getAssignedToValue(): Promise<string> {
    const count = await this.assignedToInput.count();
    if (count > 0) {
      return await this.assignedToInput.first().inputValue();
    }
    return '';
  }

  async getResolutionNotesValue(): Promise<string> {
    const count = await this.resolutionNotesTextarea.count();
    if (count > 0) {
      return await this.resolutionNotesTextarea.first().inputValue();
    }
    return '';
  }

  async isEditingMode(): Promise<boolean> {
    const saveVisible = await this.saveChangesButton.isVisible().catch(() => false);
    const cancelVisible = await this.cancelButton.isVisible().catch(() => false);
    return saveVisible || cancelVisible;
  }

  async getIndicatorCount(): Promise<number> {
    return await this.indicatorCards.count();
  }

  async getIndicatorConfidence(index: number): Promise<number | null> {
    const card = this.indicatorCards.nth(index);
    const confidenceText = await card.getByText(/\d+%/).textContent();
    if (confidenceText) {
      const match = confidenceText.match(/(\d+)%/);
      return match ? parseInt(match[1], 10) : null;
    }
    return null;
  }

  async getIndicatorType(index: number): Promise<string | null> {
    const card = this.indicatorCards.nth(index);
    const typeElement = card.locator('.font-medium, p.font-medium');
    if (await typeElement.isVisible().catch(() => false)) {
      return await typeElement.textContent();
    }
    return null;
  }

  async isAcknowledgeButtonVisible(): Promise<boolean> {
    return await this.acknowledgeButton.isVisible().catch(() => false);
  }

  async isUpdateStatusButtonVisible(): Promise<boolean> {
    return await this.updateStatusButton.isVisible().catch(() => false);
  }

  async areFieldsDisabled(): Promise<boolean> {
    const statusDisabled = await this.statusSelect.isDisabled().catch(() => true);
    const assignedDisabled = await this.assignedToInput.isDisabled().catch(() => true);
    const notesDisabled = await this.resolutionNotesTextarea.isDisabled().catch(() => true);
    return statusDisabled && assignedDisabled && notesDisabled;
  }

  async getBasicInfoValue(label: string): Promise<string | null> {
    const card = this.basicInfoCard;
    const labelElement = card.getByText(label);
    const count = await labelElement.count();
    for (let i = 0; i < count; i++) {
      const el = labelElement.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const valueElement = el.locator('..').locator('p.text-white, .text-white');
        const valueCount = await valueElement.count();
        for (let j = 0; j < valueCount; j++) {
          if (await valueElement.nth(j).isVisible().catch(() => false)) {
            return await valueElement.nth(j).textContent();
          }
        }
      }
    }
    return null;
  }

  async getRelatedEntity(type: 'user' | 'target'): Promise<string | null> {
    const card = this.relatedEntitiesCard;
    const entityRow = card.getByText(new RegExp(type, 'i'));
    const count = await entityRow.count();
    for (let i = 0; i < count; i++) {
      const el = entityRow.nth(i);
      if (await el.isVisible().catch(() => false)) {
        const nameElement = el.locator('..').locator('.font-medium, p.text-white');
        const nameCount = await nameElement.count();
        for (let j = 0; j < nameCount; j++) {
          if (await nameElement.nth(j).isVisible().catch(() => false)) {
            return await nameElement.nth(j).textContent();
          }
        }
      }
    }
    return null;
  }

  async waitForContent(): Promise<void> {
    await Promise.race([
      this.heading.first().waitFor({ state: 'visible', timeout: 5000 }).catch(() => {}),
      this.page.waitForTimeout(3000),
    ]);
  }
}
