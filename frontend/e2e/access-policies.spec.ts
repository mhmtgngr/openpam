import { test, expect } from './fixtures/auth.fixture';

// Note: Access policy API mocking is now handled by the accessPolicyPage fixture

test.describe('Access Policy Management', () => {
  test.describe('Access Policies List Page', () => {
    test('should display access policies list page', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByRole('heading', { name: /access policies/i })).toBeVisible();
      await expect(accessPolicyPage.getByText(/control who can access what resources/i)).toBeVisible();
    });

    test('should display all access policies', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByText('Production Database Access')).toBeVisible();
      await expect(accessPolicyPage.getByText('SSH Access Policy')).toBeVisible();
    });

    test('should display policy status badges', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const activeBadges = accessPolicyPage.getByText('active').all();
      expect((await activeBadges).length).toBeGreaterThan(0);
    });

    test('should display policy tags', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByText('#production')).toBeVisible();
      await expect(accessPolicyPage.getByText('#database')).toBeVisible();
    });

    test('should display navigation buttons', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByRole('link', { name: /test policies/i })).toBeVisible();
      await expect(accessPolicyPage.getByRole('link', { name: /new policy/i })).toBeVisible();
    });

    test('should navigate to create policy page', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const newPolicyButton = accessPolicyPage.getByRole('link', { name: /new policy/i });
      await newPolicyButton.click();

      await accessPolicyPage.waitForURL('/policies/access/new');
      await expect(accessPolicyPage.getByRole('heading', { name: /create access policy/i })).toBeVisible();
    });

    test('should navigate to test policy page', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const testButton = accessPolicyPage.getByRole('button', { name: /test policies/i });
      await testButton.click();

      await accessPolicyPage.waitForURL('/policies/access/test');
      await expect(accessPolicyPage.getByRole('heading', { name: /policy testing/i })).toBeVisible();
    });

    test('should expand policy to show rules', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Find the first policy card by its name, then locate its expand button
      // The expand button is in the actions section of each card
      const firstPolicyCard = accessPolicyPage.getByText('Production Database Access');
      await expect(firstPolicyCard).toBeVisible();

      // The expand button is the chevron button in the policy card
      // We need to find the Card that contains "Production Database Access"
      // Then find the button with SVG chevron icon within that card
      const policyCard = accessPolicyPage.locator('.card').filter({ hasText: 'Production Database Access' });
      // The expand button contains an SVG - use locator to find it
      // The chevron icon is lucide-react's ChevronDown or ChevronUp
      const expandButton = policyCard.locator('button svg').first();
      await expandButton.click();

      // Wait for expansion animation
      await accessPolicyPage.waitForTimeout(500);

      // Should show rules
      await expect(accessPolicyPage.getByText('Allow admins')).toBeVisible({ timeout: 5000 });
      await expect(accessPolicyPage.getByText('Deny regular users')).toBeVisible();
      // Use exact matching for effect types to avoid matching rule names like "Allow admins"
      await expect(accessPolicyPage.getByText('ALLOW', { exact: true })).toBeVisible();
      await expect(accessPolicyPage.getByText('DENY', { exact: true })).toBeVisible();
    });

    test('should search policies', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const searchInput = accessPolicyPage.getByPlaceholder(/search policies by name/i);
      await searchInput.fill('production');

      // Wait for search to process
      await accessPolicyPage.waitForTimeout(300);

      // Should show filtered results
      await expect(accessPolicyPage.getByText('Production Database Access')).toBeVisible();
    });

    test('should filter by status', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const statusSelect = accessPolicyPage.locator('select').first();
      await statusSelect.selectOption('active');

      await accessPolicyPage.waitForTimeout(300);
      // Should only show active policies - use badge selector to avoid dropdown options
      const activeBadges = accessPolicyPage.locator('.badge').filter({ hasText: 'active' });
      await expect(activeBadges.first()).toBeVisible();
    });
  });

  test.describe('Access Policy Form Page', () => {
    test('should display create policy form', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByRole('heading', { name: /create access policy/i })).toBeVisible();
      await expect(accessPolicyPage.getByText(/define rules for controlling resource access/i)).toBeVisible();
    });

    test('should validate required fields', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // The form has HTML5 native validation (required attribute on inputs)
      // which prevents form submission and shows browser validation UI.
      // To test React's custom validation, we need to bypass native validation.
      // We'll fill the name field (which has required) and then submit,
      // which will trigger the custom validation for rules.

      // Fill the name field to pass HTML5 validation
      const nameInput = accessPolicyPage.getByLabel(/policy name/i);
      await nameInput.fill('Test Policy');

      // Now submit - rules validation should trigger
      const submitButton = accessPolicyPage.getByRole('button', { name: /create policy/i });
      await submitButton.click();

      // Wait for React state to update and re-render
      await accessPolicyPage.waitForTimeout(500);

      // Should show validation error for rules (at least one rule is required)
      await expect(accessPolicyPage.getByText(/at least one rule is required/i)).toBeVisible({ timeout: 3000 });
    });

    test('should set policy name', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const nameInput = accessPolicyPage.getByLabel(/policy name/i);
      await nameInput.fill('Test Access Policy');

      await expect(nameInput).toHaveValue('Test Access Policy');
    });

    test('should set policy description', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const descriptionTextarea = accessPolicyPage.getByPlaceholder(/describe what this policy controls/i);
      await descriptionTextarea.fill('This is a test policy for E2E testing');

      await expect(descriptionTextarea).toHaveValue('This is a test policy for E2E testing');
    });

    test('should set policy status', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const statusSelect = accessPolicyPage.locator('select').filter({ hasText: /draft/i });
      await statusSelect.selectOption('active');

      await expect(statusSelect).toHaveValue('active');
    });

    test('should set policy priority', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const priorityInput = accessPolicyPage.getByLabel(/priority/i);
      await priorityInput.fill('200');

      await expect(priorityInput).toHaveValue('200');
    });

    test('should set conflict resolution', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const conflictSelect = accessPolicyPage.locator('select').filter({ hasText: /deny overrides/i });
      await conflictSelect.selectOption('allow_overrides');

      await expect(conflictSelect).toHaveValue('allow_overrides');
    });

    test('should add policy tags', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      const tagInput = accessPolicyPage.getByPlaceholder(/add a tag/i);
      await tagInput.fill('test-tag');

      const addButton = accessPolicyPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      await expect(accessPolicyPage.getByText('#test-tag')).toBeVisible();
    });

    test('should remove policy tags', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a tag first
      const tagInput = accessPolicyPage.getByPlaceholder(/add a tag/i);
      await tagInput.fill('removable-tag');

      const addButton = accessPolicyPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Then remove it
      const removeButton = accessPolicyPage.locator('button').filter({ hasText: /×/i }).first();
      await removeButton.click();

      await expect(accessPolicyPage.getByText('#removable-tag')).not.toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Policy Rule Builder', () => {
    test('should display rules section', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByRole('heading', { name: 'Rules' })).toBeVisible();
      await expect(accessPolicyPage.getByText(/no rules defined/i)).toBeVisible();
    });

    test('should add a new rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Verify initial state - no rules defined
      await expect(accessPolicyPage.getByText(/no rules defined/i)).toBeVisible();

      // The "Add Rule" button is inside the PolicyRuleBuilder component
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for DOM to update - use a more explicit wait
      await accessPolicyPage.waitForLoadState('domcontentloaded').catch(() => {});
      await accessPolicyPage.waitForTimeout(500);

      // After adding a rule, "No rules defined" should disappear
      await expect(accessPolicyPage.getByText(/no rules defined/i)).not.toBeVisible({ timeout: 2000 });
    });

    test('should set rule name', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule first
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Find the rule name input - it's the first input in the rules section
      // The rule name input doesn't have a default value attribute, so we select it by position
      const ruleNameInput = accessPolicyPage.locator('.card-body').filter({ hasText: /Rules/i }).locator('input').first();
      await expect(ruleNameInput).toBeVisible({ timeout: 3000 });

      // Get initial value and set new value
      const initialValue = await ruleNameInput.inputValue();
      await ruleNameInput.clear();
      await ruleNameInput.fill('Test Rule');

      await expect(ruleNameInput).toHaveValue('Test Rule');
    });

    test('should set rule effect', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule first
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added - the first rule is expanded by default
      await accessPolicyPage.waitForTimeout(500);

      // Set effect to deny - find the select by its adjacent label "Effect"
      const effectLabel = accessPolicyPage.getByText('Effect').all();
      // Find the Effect label within the expanded rule section
      const effectSelect = accessPolicyPage.locator('.card-body').filter({ hasText: /Rules/i })
        .locator('*').filter({ hasText: 'Effect' })
        .locator('..')
        .locator('select').filter(async (select, _) => {
          const options = await select.locator('option').allTextContents();
          return options.includes('Allow') && options.includes('Deny');
        }).first();

      await effectSelect.selectOption('deny');

      await expect(accessPolicyPage.getByText('DENY', { exact: true })).toBeVisible();
    });

    test('should set logical operator', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Set operator to OR - use a more direct selector
      // Find the select within the expanded rule that has AND/OR options
      const operatorSelect = accessPolicyPage.locator('.card-body').filter({ hasText: /Rules/i })
        .locator('select').filter(async (select, _) => {
          const options = await select.locator('option').allTextContents();
          return options.some(o => o.includes('AND') && o.includes('All conditions')) &&
                 options.some(o => o.includes('OR') && o.includes('At least one'));
        }).first();

      await operatorSelect.selectOption('OR');

      await expect(operatorSelect).toHaveValue('OR');
    });

    test('should add resources to rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add a resource - find the Resources select by its label
      const resourceSelect = accessPolicyPage.locator('.card-body').filter({ hasText: /Rules/i })
        .locator('*').filter({ hasText: 'Resources' })
        .locator('..')
        .locator('select').first();

      await resourceSelect.selectOption('credential:*');

      await expect(accessPolicyPage.getByText('credential:*')).toBeVisible();
    });

    test('should add actions to rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add an action - find select with "checkout" option
      const actionSelect = accessPolicyPage.locator('select').filter(async (select, _) => {
        const options = await select.locator('option').allTextContents();
        return options.includes('checkout');
      }).first();

      await actionSelect.selectOption('checkout');

      await expect(accessPolicyPage.getByText('checkout')).toBeVisible();
    });

    test('should add roles to rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add a role
      const roleInput = accessPolicyPage.getByPlaceholder(/enter roles/i);
      await roleInput.fill('admin');
      await accessPolicyPage.keyboard.press('Enter');

      await expect(accessPolicyPage.getByText('admin')).toBeVisible();
    });

    test('should remove rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add two rules
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();
      await addRuleButton.click();

      // Remove first rule
      const removeButton = accessPolicyPage.locator('button').filter({ hasText: /delete/i }).first();
      await removeButton.click();

      // Should have one less rule
      await expect(accessPolicyPage.getByText(/no rules defined/i)).not.toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Condition Builder', () => {
    test('should add condition to rule', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add condition
      const addConditionButton = accessPolicyPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      // Should show condition editor
      await expect(accessPolicyPage.getByText(/no conditions defined/i)).not.toBeVisible({ timeout: 2000 });
    });

    test('should set condition type', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add condition - conditions are auto-expanded
      const addConditionButton = accessPolicyPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      // Wait for condition to be added
      await accessPolicyPage.waitForTimeout(500);

      // Set condition type
      const typeSelect = accessPolicyPage.locator('select').filter({ hasText: /condition type/i });
      await typeSelect.selectOption('time_range');

      await expect(typeSelect).toHaveValue('time_range');
    });

    test('should remove condition', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/new');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a rule - it's expanded by default
      const addRuleButton = accessPolicyPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Wait for rule to be added
      await accessPolicyPage.waitForTimeout(500);

      // Add condition
      const addConditionButton = accessPolicyPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      // Wait for condition to be added
      await accessPolicyPage.waitForTimeout(500);

      // Remove condition - use the × button in the condition card
      const removeConditionButton = accessPolicyPage.locator('button').filter({ hasText: /×/i }).first();
      await removeConditionButton.click();

      await expect(accessPolicyPage.getByText(/no conditions defined/i)).toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Policy Test Page', () => {
    test('should display test page', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByRole('heading', { name: /policy testing/i })).toBeVisible();
      await expect(accessPolicyPage.getByText(/test policy evaluation with different scenarios/i)).toBeVisible();
    });

    test('should display test scenarios section', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByText(/test scenarios/i)).toBeVisible();
      await expect(accessPolicyPage.getByRole('button', { name: /add scenario/i })).toBeVisible();
    });

    test('should add test scenario', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      const addScenarioButton = accessPolicyPage.getByRole('button', { name: /add scenario/i });
      await addScenarioButton.click();

      await expect(accessPolicyPage.getByText(/scenario 2/i)).toBeVisible();
    });

    test('should set scenario user id', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Expand first scenario - the button contains actual "+" text
      const expandButton = accessPolicyPage.locator('button').filter({ hasText: '+' }).first();
      await expandButton.click();

      // Wait for expansion
      await accessPolicyPage.waitForTimeout(500);

      const userIdInput = accessPolicyPage.getByLabel(/user id/i);
      await userIdInput.fill('test-user-123');

      await expect(userIdInput).toHaveValue('test-user-123');
    });

    test('should set scenario roles', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Expand first scenario - the button contains actual "+" text
      const expandButton = accessPolicyPage.locator('button').filter({ hasText: '+' }).first();
      await expandButton.click();

      // Wait for expansion
      await accessPolicyPage.waitForTimeout(500);

      const rolesInput = accessPolicyPage.getByPlaceholder(/admin, operator/i);
      await rolesInput.fill('admin, operator');

      await expect(rolesInput).toHaveValue('admin, operator');
    });

    test('should set scenario resource', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Expand first scenario - the button contains actual "+" text
      const expandButton = accessPolicyPage.locator('button').filter({ hasText: '+' }).first();
      await expandButton.click();

      // Wait for expansion
      await accessPolicyPage.waitForTimeout(500);

      const resourceSelect = accessPolicyPage.locator('select').filter({ hasText: /select or enter/i });
      await resourceSelect.selectOption('credential:prod-db-ssh');

      await expect(resourceSelect).toHaveValue('credential:prod-db-ssh');
    });

    test('should set scenario action', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Expand first scenario - the button contains actual "+" text
      const expandButton = accessPolicyPage.locator('button').filter({ hasText: '+' }).first();
      await expandButton.click();

      // Wait for expansion
      await accessPolicyPage.waitForTimeout(500);

      const actionSelect = accessPolicyPage.locator('select').filter({ hasText: /checkout/i });
      await actionSelect.selectOption('checkout');

      await expect(actionSelect).toHaveValue('checkout');
    });

    test('should display test summary', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByText(/test summary/i)).toBeVisible();
      await expect(accessPolicyPage.getByText(/total scenarios/i)).toBeVisible();
      await expect(accessPolicyPage.getByText(/executed/i)).toBeVisible();
      await expect(accessPolicyPage.getByText(/passed/i)).toBeVisible();
      await expect(accessPolicyPage.getByText(/failed/i)).toBeVisible();
    });

    test('should display quick actions', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      await expect(accessPolicyPage.getByText(/quick actions/i)).toBeVisible();
      await expect(accessPolicyPage.getByRole('button', { name: /load sample scenarios/i })).toBeVisible();
      await expect(accessPolicyPage.getByRole('button', { name: /clear results/i })).toBeVisible();
    });

    test('should run single test scenario', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Load a policy first
      const loadPolicyButton = accessPolicyPage.getByRole('button', { name: /load policy/i });
      // Mock the prompt for this test
      await accessPolicyPage.evaluate(() => {
        (window as any).prompt = () => 'policy-1';
      });
      await loadPolicyButton.click();

      // Run test on first scenario
      const playButton = accessPolicyPage.locator('button').filter({ hasText: /play/i }).first();
      await playButton.click();

      // Should show result
      await expect(accessPolicyPage.getByText(/evaluation result/i)).toBeVisible({ timeout: 5000 });
    });

    test('should remove test scenario', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Add a scenario
      const addScenarioButton = accessPolicyPage.getByRole('button', { name: /add scenario/i });
      await addScenarioButton.click();

      // Remove it
      const removeButton = accessPolicyPage.locator('button').filter({ hasText: /delete/i }).first();
      await removeButton.click();

      // Should have one less scenario
      await expect(accessPolicyPage.getByText(/scenario 2/i)).not.toBeVisible({ timeout: 2000 });
    });

    test('should duplicate test scenario', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access/test');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Duplicate first scenario - the button contains actual "+" text for expansion
      const expandButton = accessPolicyPage.locator('button').filter({ hasText: '+' }).first();
      await expandButton.click();

      // Wait for expansion
      await accessPolicyPage.waitForTimeout(500);

      const copyButton = accessPolicyPage.locator('button').filter({ hasText: /copy/i }).first();
      await copyButton.click();

      await expect(accessPolicyPage.getByText(/scenario 1 \(copy\)/i)).toBeVisible();
    });
  });

  test.describe('Policy Actions', () => {
    test('should activate policy', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      // This test assumes there's a policy that can be activated
      const activateButtons = accessPolicyPage.getByRole('button', { name: /activate/i }).all();

      if (activateButtons.length > 0) {
        await activateButtons[0].click();
        // Should show success toast
        await expect(accessPolicyPage.getByText(/policy activated/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should deactivate policy', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const deactivateButtons = accessPolicyPage.getByRole('button', { name: /deactivate/i }).all();

      if (deactivateButtons.length > 0) {
        await deactivateButtons[0].click();
        // Should show success toast
        await expect(accessPolicyPage.getByText(/policy deactivated/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should navigate to edit policy', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      const editButton = accessPolicyPage.getByRole('link', { name: /edit/i }).first();
      await editButton.click();

      await accessPolicyPage.waitForURL(/\/policies\/access\/.+/);
      await expect(accessPolicyPage.getByRole('heading', { name: /edit access policy/i })).toBeVisible();
    });

    test('should clone policy', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Mock the prompt
      await accessPolicyPage.evaluate(() => {
        (window as any).prompt = () => 'Cloned Policy';
      });

      const cloneButton = accessPolicyPage.locator('button').filter({ hasText: /clone/i }).first();
      if (await cloneButton.isVisible({ timeout: 2000 })) {
        await cloneButton.click();
        await expect(accessPolicyPage.getByText(/policy cloned/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should delete policy with confirmation', async ({ accessPolicyPage }) => {
      await accessPolicyPage.goto('/policies/access');
      await accessPolicyPage.waitForLoadState('networkidle');

      // Mock the confirm dialog
      await accessPolicyPage.evaluate(() => {
        (window as any).confirm = () => true;
      });

      const deleteButton = accessPolicyPage.locator('button').filter({ hasText: /delete/i }).first();
      if (await deleteButton.isVisible({ timeout: 2000 })) {
        await deleteButton.click();
        await expect(accessPolicyPage.getByText(/policy deleted/i)).toBeVisible({ timeout: 3000 });
      }
    });
  });
});
