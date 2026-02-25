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
    this.heading = page.getByRole('heading', { level: 1 });
    this.backButton = page.getByRole('button', {}).filter({ hasText: '' }).locator('svg').first();
    this.severityBadge = page.locator('[class*="text-danger-400"], [class*="text-warning-400"], [class*="text-success-400"], [class*="text-gray-400"]');
    this.statusBadge = page.locator('[class*="badge"]').or(page.locator('.rounded-full'));
    this.acknowledgeButton = page.getByRole('button', { name: /acknowledge/i });
    this.updateStatusButton = page.getByRole('button', { name: /update status/i });
    this.saveChangesButton = page.getByRole('button', { name: /save changes/i });
    this.cancelButton = page.getByRole('button', { name: /cancel/i });
    this.basicInfoCard = page.locator('.card').filter({ hasText: /basic information/i });
    this.indicatorsCard = page.locator('.card').filter({ hasText: /detection indicators/i });
    this.statusAssignmentCard = page.locator('.card').filter({ hasText: /status & assignment/i });
    this.relatedEntitiesCard = page.locator('.card').filter({ hasText: /related entities/i });
    this.typeInfoCard = page.locator('.card').filter({ hasText: /anomaly type/i });
    this.statusSelect = page.locator('select').filter({ hasText: /status/i });
    this.assignedToInput = page.getByPlaceholder(/email or user id/i);
    this.resolutionNotesTextarea = page.getByPlaceholder(/investigation notes/i).or(page.getByPlaceholder(/resolution/i));
    this.indicatorCards = page.locator('[class*="bg-gray-900/50"]').filter({ hasText: /confidence/i });
  }

  async goto(anomalyId: string) {
    await this.page.goto(`/analytics/anomalies/${anomalyId}`);
    await this.page.waitForLoadState('networkidle');
  }

  async getHeadingText(): Promise<string | null> {
    const heading = this.heading.first();
    if (await heading.isVisible().catch(() => false)) {
      return await heading.textContent();
    }
    return null;
  }

  async getAnomalyTitle(): Promise<string | null> {
    const title = this.page.locator('h1.text-xl').or(this.page.locator('.text-xl.font-bold'));
    if (await title.isVisible().catch(() => false)) {
      return await title.textContent();
    }
    return null;
  }

  async getAnomalyDescription(): Promise<string | null> {
    const description = this.page.locator('p.text-sm.text-gray-400');
    const descriptions = await description.all();
    for (const desc of descriptions) {
      const text = await desc.textContent();
      if (text && text.length > 20) {
        return text;
      }
    }
    return null;
  }

  async getSeverity(): Promise<string | null> {
    const severityText = await this.page.getByText(/(critical|high|medium|low)/i).first().textContent();
    if (severityText) {
      const match = severityText.match(/(critical|high|medium|low)/i);
      return match ? match[1].toLowerCase() : null;
    }
    return null;
  }

  async getStatus(): Promise<string | null> {
    const statusText = await this.page.getByText(/(open|investigating|resolved|false positive)/i).first().textContent();
    if (statusText) {
      const match = statusText.match(/(open|investigating|resolved|false positive)/i);
      return match ? match[1].toLowerCase() : null;
    }
    return null;
  }

  async getConfidenceScore(): Promise<number | null> {
    const confidenceText = await this.page.getByText(/\d+%/).first().textContent();
    if (confidenceText) {
      const match = confidenceText.match(/(\d+)%/);
      return match ? parseInt(match[1], 10) : null;
    }
    return null;
  }

  async clickAcknowledge() {
    await this.acknowledgeButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickUpdateStatus() {
    await this.updateStatusButton.click();
  }

  async clickSaveChanges() {
    await this.saveChangesButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async clickCancel() {
    await this.cancelButton.click();
  }

  async clickBack() {
    const backButton = this.page.locator('button').filter({ hasText: '' }).locator('svg.lucide-chevron-left').locator('..');
    await backButton.click();
    await this.page.waitForLoadState('networkidle');
  }

  async setStatus(status: 'open' | 'investigating' | 'resolved' | 'false_positive') {
    await this.statusSelect.selectOption({ label: new RegExp(status, 'i') });
  }

  async setAssignedTo(value: string) {
    await this.assignedToInput.fill(value);
  }

  async setResolutionNotes(notes: string) {
    await this.resolutionNotesTextarea.fill(notes);
  }

  async getSelectedStatus(): Promise<string | null> {
    const selectedOption = await this.statusSelect.inputValue();
    return selectedOption || null;
  }

  async getAssignedToValue(): Promise<string> {
    return await this.assignedToInput.inputValue();
  }

  async getResolutionNotesValue(): Promise<string> {
    return await this.resolutionNotesTextarea.inputValue();
  }

  async isEditingMode(): Promise<boolean> {
    // Check if the save/cancel buttons are visible
    return await this.saveChangesButton.isVisible().catch(() => false) ||
           await this.cancelButton.isVisible().catch(() => false);
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
    const typeElement = card.locator('.font-medium').or(card.locator('p.font-medium'));
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
    const isStatusDisabled = await this.statusSelect.isDisabled();
    const isAssignedToDisabled = await this.assignedToInput.isDisabled();
    const isNotesDisabled = await this.resolutionNotesTextarea.isDisabled();
    return isStatusDisabled && isAssignedToDisabled && isNotesDisabled;
  }

  async getBasicInfoValue(label: string): Promise<string | null> {
    const card = this.basicInfoCard;
    const labelElement = card.getByText(label);
    if (await labelElement.isVisible().catch(() => false)) {
      const valueElement = labelElement.locator('..').locator('p.text-sm.text-white').or(labelElement.locator('..').locator('.text-white'));
      if (await valueElement.isVisible().catch(() => false)) {
        return await valueElement.textContent();
      }
    }
    return null;
  }

  async getRelatedEntity(type: 'user' | 'target'): Promise<string | null> {
    const card = this.relatedEntitiesCard;
    const entityRow = card.getByText(new RegExp(type, 'i')).locator('..');
    const nameElement = entityRow.locator('.font-medium.text-white').or(entityRow.locator('p:nth-child(2)'));
    if (await nameElement.isVisible().catch(() => false)) {
      return await nameElement.textContent();
    }
    return null;
  }

  async waitForContent(): Promise<void> {
    await Promise.race([
      this.heading.waitFor({ state: 'visible' }).catch(() => {}),
      this.page.waitForTimeout(2000),
    ]);
  }
}
