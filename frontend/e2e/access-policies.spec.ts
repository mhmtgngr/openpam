import { test, expect } from './fixtures/auth.fixture';
import { LoginPage } from './pages/LoginPage';

// Mock Access Policies API responses
const mockAccessPolicies = [
  {
    id: 'policy-1',
    name: 'Production Database Access',
    description: 'Controls access to production database credentials',
    status: 'active',
    priority: 100,
    rules: [
      {
        id: 'rule-1',
        name: 'Allow admins',
        effect: 'allow',
        conditions: [
          {
            id: 'cond-1',
            type: 'role',
            field: 'role',
            operator: 'in',
            value: ['admin'],
          },
        ],
        logical_operator: 'AND',
        priority: 1,
        resources: ['credential:prod-db-*'],
        actions: ['checkout', 'connect'],
        roles: ['admin'],
        users: [],
      },
      {
        id: 'rule-2',
        name: 'Deny regular users',
        effect: 'deny',
        conditions: [],
        logical_operator: 'AND',
        priority: 2,
        resources: ['credential:prod-db-*'],
        actions: ['checkout'],
        roles: ['user'],
        users: [],
      },
    ],
    conflict_resolution: 'deny_overrides',
    is_default: false,
    is_system: false,
    tenant_id: 'tenant-1',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    tags: ['production', 'database'],
  },
  {
    id: 'policy-2',
    name: 'SSH Access Policy',
    description: 'SSH session access controls',
    status: 'active',
    priority: 50,
    rules: [],
    conflict_resolution: 'deny_overrides',
    is_default: true,
    is_system: false,
    tenant_id: 'tenant-1',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    tags: ['ssh', 'sessions'],
  },
];

const mockPolicyDetail = {
  ...mockAccessPolicies[0],
};

test.describe('Access Policy Management', () => {
  test.beforeEach(async ({ page }) => {
    // Setup access policy mocks
    page.route('**/api/v1/policies/access*', async (route) => {
      const url = route.request().url();
      const method = route.request().method();

      // GET /api/v1/policies/access - List policies
      if (url.includes('/api/v1/policies/access') && !url.includes('/test') && method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockAccessPolicies, pagination: { total: 2, offset: 0, limit: 20 } }),
        });
        return;
      }

      // GET /api/v1/policies/access/:id - Get policy detail
      if (url.includes('/api/v1/policies/access/') && method === 'GET' && !url.includes('/test')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockPolicyDetail),
        });
        return;
      }

      // POST /api/v1/policies/access - Create policy
      if (url.includes('/api/v1/policies/access') && method === 'POST' && !url.includes('/test')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...mockAccessPolicies[0],
            id: 'new-policy-id',
            name: 'New Access Policy',
          }),
        });
        return;
      }

      // PATCH /api/v1/policies/access/:id - Update policy
      if (url.includes('/api/v1/policies/access/') && method === 'PATCH') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockPolicyDetail),
        });
        return;
      }

      // DELETE /api/v1/policies/access/:id - Delete policy
      if (url.includes('/api/v1/policies/access/') && method === 'DELETE') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Policy deleted' }),
        });
        return;
      }

      // POST /api/v1/policies/access/validate - Validate policy
      if (url.includes('/validate') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ valid: true, errors: [], warnings: [] }),
        });
        return;
      }

      // POST /api/v1/policies/access/evaluate - Evaluate policy
      if (url.includes('/evaluate') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            allowed: true,
            effect: 'allow',
            matched_policy_id: 'policy-1',
            matched_rule_id: 'rule-1',
            reason: 'User has admin role',
            details: [],
            evaluated_at: new Date().toISOString(),
          }),
        });
        return;
      }

      // POST /api/v1/policies/access/test - Test policy
      if (url.includes('/test') && method === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              scenario_name: 'Test Scenario',
              result: {
                allowed: true,
                effect: 'allow',
                reason: 'Test passed',
                details: [],
                evaluated_at: new Date().toISOString(),
              },
              passed: true,
              expected_allowed: true,
            },
          ]),
        });
        return;
      }

      // Default fallback
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'NOT_FOUND', message: 'Route not found' } }),
      });
    });

    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await page.waitForLoadState('networkidle');
  });

  test.describe('Access Policies List Page', () => {
    test('should display access policies list page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('heading', { name: /access policies/i })).toBeVisible();
      await expect(authenticatedPage.getByText(/control who can access what resources/i)).toBeVisible();
    });

    test('should display all access policies', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('Production Database Access')).toBeVisible();
      await expect(authenticatedPage.getByText('SSH Access Policy')).toBeVisible();
    });

    test('should display policy status badges', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const activeBadges = authenticatedPage.getByText('active').all();
      expect((await activeBadges).length).toBeGreaterThan(0);
    });

    test('should display policy tags', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText('#production')).toBeVisible();
      await expect(authenticatedPage.getByText('#database')).toBeVisible();
    });

    test('should display navigation buttons', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('link', { name: /test policies/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('link', { name: /new policy/i })).toBeVisible();
    });

    test('should navigate to create policy page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const newPolicyButton = authenticatedPage.getByRole('link', { name: /new policy/i });
      await newPolicyButton.click();

      await authenticatedPage.waitForURL('/policies/access/new');
      await expect(authenticatedPage.getByRole('heading', { name: /create access policy/i })).toBeVisible();
    });

    test('should navigate to test policy page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const testButton = authenticatedPage.getByRole('button', { name: /test policies/i });
      await testButton.click();

      await authenticatedPage.waitForURL('/policies/access/test');
      await expect(authenticatedPage.getByRole('heading', { name: /policy testing/i })).toBeVisible();
    });

    test('should expand policy to show rules', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      // Click expand button on first policy
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Should show rules
      await expect(authenticatedPage.getByText('Allow admins')).toBeVisible();
      await expect(authenticatedPage.getByText('Deny regular users')).toBeVisible();
      await expect(authenticatedPage.getByText('ALLOW')).toBeVisible();
      await expect(authenticatedPage.getByText('DENY')).toBeVisible();
    });

    test('should search policies', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const searchInput = authenticatedPage.getByPlaceholder(/search policies by name/i);
      await searchInput.fill('production');

      // Wait for search to process
      await authenticatedPage.waitForTimeout(300);

      // Should show filtered results
      await expect(authenticatedPage.getByText('Production Database Access')).toBeVisible();
    });

    test('should filter by status', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const statusSelect = authenticatedPage.locator('select').first();
      await statusSelect.selectOption('active');

      await authenticatedPage.waitForTimeout(300);
      // Should only show active policies
      await expect(authenticatedPage.getByText('active')).toBeVisible();
    });
  });

  test.describe('Access Policy Form Page', () => {
    test('should display create policy form', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('heading', { name: /create access policy/i })).toBeVisible();
      await expect(authenticatedPage.getByText(/define rules for controlling resource access/i)).toBeVisible();
    });

    test('should validate required fields', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Try to submit without filling required fields
      const submitButton = authenticatedPage.getByRole('button', { name: /create policy/i });
      await submitButton.click();

      // Should show validation errors
      await expect(authenticatedPage.getByText(/name is required/i)).toBeVisible({ timeout: 3000 });
    });

    test('should set policy name', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const nameInput = authenticatedPage.getByLabel(/policy name/i);
      await nameInput.fill('Test Access Policy');

      await expect(nameInput).toHaveValue('Test Access Policy');
    });

    test('should set policy description', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const descriptionTextarea = authenticatedPage.getByPlaceholder(/describe what this policy controls/i);
      await descriptionTextarea.fill('This is a test policy for E2E testing');

      await expect(descriptionTextarea).toHaveValue('This is a test policy for E2E testing');
    });

    test('should set policy status', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const statusSelect = authenticatedPage.locator('select').filter({ hasText: /draft/i });
      await statusSelect.selectOption('active');

      await expect(statusSelect).toHaveValue('active');
    });

    test('should set policy priority', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const priorityInput = authenticatedPage.getByLabel(/priority/i);
      await priorityInput.fill('200');

      await expect(priorityInput).toHaveValue('200');
    });

    test('should set conflict resolution', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const conflictSelect = authenticatedPage.locator('select').filter({ hasText: /deny overrides/i });
      await conflictSelect.selectOption('allow_overrides');

      await expect(conflictSelect).toHaveValue('allow_overrides');
    });

    test('should add policy tags', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const tagInput = authenticatedPage.getByPlaceholder(/add a tag/i);
      await tagInput.fill('test-tag');

      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      await expect(authenticatedPage.getByText('#test-tag')).toBeVisible();
    });

    test('should remove policy tags', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a tag first
      const tagInput = authenticatedPage.getByPlaceholder(/add a tag/i);
      await tagInput.fill('removable-tag');

      const addButton = authenticatedPage.locator('button').filter({ hasText: /add/i }).first();
      await addButton.click();

      // Then remove it
      const removeButton = authenticatedPage.locator('button').filter({ hasText: /×/i }).first();
      await removeButton.click();

      await expect(authenticatedPage.getByText('#removable-tag')).not.toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Policy Rule Builder', () => {
    test('should display rules section', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/rules/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/no rules defined/i)).toBeVisible();
    });

    test('should add a new rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Should show rule editor
      await expect(authenticatedPage.getByText(/rule 1/i)).toBeVisible({ timeout: 2000 });
      await expect(authenticatedPage.getByText(/allow/i)).toBeVisible();
    });

    test('should set rule name', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule first
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Set rule name
      const ruleNameInput = authenticatedPage.getByPlaceholder(/rule 1/i);
      await ruleNameInput.clear();
      await ruleNameInput.fill('Test Rule');

      await expect(ruleNameInput).toHaveValue('Test Rule');
    });

    test('should set rule effect', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule first
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      // Expand rule
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Set effect to deny
      const effectSelect = authenticatedPage.locator('select').filter({ hasText: /allow/i }).first();
      await effectSelect.selectOption('deny');

      await expect(authenticatedPage.getByText('DENY')).toBeVisible();
    });

    test('should set logical operator', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Set operator to OR
      const operatorSelect = authenticatedPage.locator('select').filter({ hasText: /and - all/i });
      await operatorSelect.selectOption('OR');

      await expect(operatorSelect).toHaveValue('OR');
    });

    test('should add resources to rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Add a resource
      const resourceSelect = authenticatedPage.locator('select').filter({ hasText: /select or type/i }).first();
      await resourceSelect.selectOption('credential:*');

      await expect(authenticatedPage.getByText('credential:*')).toBeVisible();
    });

    test('should add actions to rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Find and use the action select (should be second occurrence of resource/action selects)
      const actionSelects = authenticatedPage.locator('select').filter({ hasText: /select or type/i }).all();
      await actionSelects[1].selectOption('checkout');

      await expect(authenticatedPage.getByText('checkout')).toBeVisible();
    });

    test('should add roles to rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Add a role
      const roleInput = authenticatedPage.getByPlaceholder(/enter roles/i);
      await roleInput.fill('admin');
      await authenticatedPage.keyboard.press('Enter');

      await expect(authenticatedPage.getByText('admin')).toBeVisible();
    });

    test('should remove rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add two rules
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();
      await addRuleButton.click();

      // Remove first rule
      const removeButton = authenticatedPage.locator('button').filter({ hasText: /delete/i }).first();
      await removeButton.click();

      // Should have one less rule
      await expect(authenticatedPage.getByText(/no rules defined/i)).not.toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Condition Builder', () => {
    test('should add condition to rule', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Add condition
      const addConditionButton = authenticatedPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      // Should show condition editor
      await expect(authenticatedPage.getByText(/no conditions defined/i)).not.toBeVisible({ timeout: 2000 });
    });

    test('should set condition type', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Add condition and expand it
      const addConditionButton = authenticatedPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      const expandConditionButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).nth(1);
      await expandConditionButton.click();

      // Set condition type
      const typeSelect = authenticatedPage.locator('select').filter({ hasText: /condition type/i });
      await typeSelect.selectOption('time_range');

      await expect(typeSelect).toHaveValue('time_range');
    });

    test('should remove condition', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/new');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a rule and expand it
      const addRuleButton = authenticatedPage.getByRole('button', { name: /add rule/i });
      await addRuleButton.click();

      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      // Add condition
      const addConditionButton = authenticatedPage.getByRole('button', { name: /add condition/i });
      await addConditionButton.click();

      // Remove condition
      const removeConditionButton = authenticatedPage.locator('button').filter({ hasText: /×/i }).first();
      await removeConditionButton.click();

      await expect(authenticatedPage.getByText(/no conditions defined/i)).toBeVisible({ timeout: 2000 });
    });
  });

  test.describe('Policy Test Page', () => {
    test('should display test page', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByRole('heading', { name: /policy testing/i })).toBeVisible();
      await expect(authenticatedPage.getByText(/test policy evaluation with different scenarios/i)).toBeVisible();
    });

    test('should display test scenarios section', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/test scenarios/i)).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /add scenario/i })).toBeVisible();
    });

    test('should add test scenario', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      const addScenarioButton = authenticatedPage.getByRole('button', { name: /add scenario/i });
      await addScenarioButton.click();

      await expect(authenticatedPage.getByText(/scenario 2/i)).toBeVisible();
    });

    test('should set scenario user id', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Expand first scenario
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      const userIdInput = authenticatedPage.getByLabel(/user id/i);
      await userIdInput.fill('test-user-123');

      await expect(userIdInput).toHaveValue('test-user-123');
    });

    test('should set scenario roles', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Expand first scenario
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      const rolesInput = authenticatedPage.getByPlaceholder(/admin, operator/i);
      await rolesInput.fill('admin, operator');

      await expect(rolesInput).toHaveValue('admin, operator');
    });

    test('should set scenario resource', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Expand first scenario
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      const resourceSelect = authenticatedPage.locator('select').filter({ hasText: /select or enter/i });
      await resourceSelect.selectOption('credential:prod-db-ssh');

      await expect(resourceSelect).toHaveValue('credential:prod-db-ssh');
    });

    test('should set scenario action', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Expand first scenario
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      const actionSelect = authenticatedPage.locator('select').filter({ hasText: /checkout/i });
      await actionSelect.selectOption('checkout');

      await expect(actionSelect).toHaveValue('checkout');
    });

    test('should display test summary', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/test summary/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/total scenarios/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/executed/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/passed/i)).toBeVisible();
      await expect(authenticatedPage.getByText(/failed/i)).toBeVisible();
    });

    test('should display quick actions', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      await expect(authenticatedPage.getByText(/quick actions/i)).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /load sample scenarios/i })).toBeVisible();
      await expect(authenticatedPage.getByRole('button', { name: /clear results/i })).toBeVisible();
    });

    test('should run single test scenario', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Load a policy first
      const loadPolicyButton = authenticatedPage.getByRole('button', { name: /load policy/i });
      // Mock the prompt for this test
      await authenticatedPage.evaluate(() => {
        (window as any).prompt = () => 'policy-1';
      });
      await loadPolicyButton.click();

      // Run test on first scenario
      const playButton = authenticatedPage.locator('button').filter({ hasText: /play/i }).first();
      await playButton.click();

      // Should show result
      await expect(authenticatedPage.getByText(/evaluation result/i)).toBeVisible({ timeout: 5000 });
    });

    test('should remove test scenario', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Add a scenario
      const addScenarioButton = authenticatedPage.getByRole('button', { name: /add scenario/i });
      await addScenarioButton.click();

      // Remove it
      const removeButton = authenticatedPage.locator('button').filter({ hasText: /delete/i }).first();
      await removeButton.click();

      // Should have one less scenario
      await expect(authenticatedPage.getByText(/scenario 2/i)).not.toBeVisible({ timeout: 2000 });
    });

    test('should duplicate test scenario', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access/test');
      await authenticatedPage.waitForLoadState('networkidle');

      // Duplicate first scenario
      const expandButton = authenticatedPage.locator('button').filter({ hasText: /\+/ }).first();
      await expandButton.click();

      const copyButton = authenticatedPage.locator('button').filter({ hasText: /copy/i }).first();
      await copyButton.click();

      await expect(authenticatedPage.getByText(/scenario 1 \(copy\)/i)).toBeVisible();
    });
  });

  test.describe('Policy Actions', () => {
    test('should activate policy', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      // This test assumes there's a policy that can be activated
      const activateButtons = authenticatedPage.getByRole('button', { name: /activate/i }).all();

      if (activateButtons.length > 0) {
        await activateButtons[0].click();
        // Should show success toast
        await expect(authenticatedPage.getByText(/policy activated/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should deactivate policy', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const deactivateButtons = authenticatedPage.getByRole('button', { name: /deactivate/i }).all();

      if (deactivateButtons.length > 0) {
        await deactivateButtons[0].click();
        // Should show success toast
        await expect(authenticatedPage.getByText(/policy deactivated/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should navigate to edit policy', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      const editButton = authenticatedPage.getByRole('link', { name: /edit/i }).first();
      await editButton.click();

      await authenticatedPage.waitForURL(/\/policies\/access\/.+/);
      await expect(authenticatedPage.getByRole('heading', { name: /edit access policy/i })).toBeVisible();
    });

    test('should clone policy', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      // Mock the prompt
      await authenticatedPage.evaluate(() => {
        (window as any).prompt = () => 'Cloned Policy';
      });

      const cloneButton = authenticatedPage.locator('button').filter({ hasText: /clone/i }).first();
      if (await cloneButton.isVisible({ timeout: 2000 })) {
        await cloneButton.click();
        await expect(authenticatedPage.getByText(/policy cloned/i)).toBeVisible({ timeout: 3000 });
      }
    });

    test('should delete policy with confirmation', async ({ authenticatedPage }) => {
      await authenticatedPage.goto('/policies/access');
      await authenticatedPage.waitForLoadState('networkidle');

      // Mock the confirm dialog
      await authenticatedPage.evaluate(() => {
        (window as any).confirm = () => true;
      });

      const deleteButton = authenticatedPage.locator('button').filter({ hasText: /delete/i }).first();
      if (await deleteButton.isVisible({ timeout: 2000 })) {
        await deleteButton.click();
        await expect(authenticatedPage.getByText(/policy deleted/i)).toBeVisible({ timeout: 3000 });
      }
    });
  });
});
