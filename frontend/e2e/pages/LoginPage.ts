import { Page, Locator } from '@playwright/test';

export class LoginPage {
  readonly page: Page;
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly mfaInput: Locator;
  readonly openPAMTitle: Locator;
  readonly signInTitle: Locator;

  constructor(page: Page) {
    this.page = page;
    // The Input component renders native input with proper attributes
    this.emailInput = page.locator('input[type="email"]').first();
    this.passwordInput = page.locator('input[type="password"]');
    this.submitButton = page.getByRole('button', { name: /Sign In|Verify/ });
    this.mfaInput = page.locator('input[maxlength="6"], input[pattern="[0-9]*"]');
    // Use getByRole to be more specific - heading with exact name
    this.openPAMTitle = page.getByRole('heading', { name: 'OpenPAM' });
    this.signInTitle = page.getByText('Sign in to your account');
  }

  async goto() {
    await this.page.goto('/login');
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  async submitMFA(code: string) {
    await this.mfaInput.fill(code);
    await this.submitButton.click();
  }

  async getErrorMessage() {
    return this.page.getByText(/Invalid credentials|Invalid email/).textContent();
  }
}
