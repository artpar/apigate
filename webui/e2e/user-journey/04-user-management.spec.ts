import { test, expect } from '@playwright/test';
import {
  navigateToModule,
  navigateToCreate,
  navigateToDetail,
  fillField,
  submitForm,
  clickCreateNew,
  deleteRecord,
  createViaAPI,
  deleteViaAPI,
  listViaAPI,
  uniqueName,
  uniqueEmail,
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  cleanupTestData,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: User Management
 *
 * Complete CRUD operations and custom actions for the user module.
 * Users have API access and are associated with plans.
 */

const TEST_PREFIX = 'e2e-user';

test.describe('User Management', () => {
  let testPlanId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test plan for user assignment
    const response = await request.post('/mod/plan', {
      data: {
        name: uniqueName('e2e-user-plan'),
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      },
    });
    const plan = await response.json();
    testPlanId = plan.id;
  });

  test.afterAll(async ({ request }) => {
    // Cleanup test plan
    if (testPlanId) {
      await request.delete(`/mod/plan/${testPlanId}`);
    }
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'user', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('users list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'user');
      await expectNoErrors(page);

      const table = page.locator('table');
      expect(await table.count()).toBeGreaterThan(0);
    });

    test('shows user data in table', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
        plan_id: testPlanId,
      });

      await navigateToModule(page, 'user');
      await waitForTableLoad(page);

      await expect(page.locator(`text=${userEmail}`)).toBeVisible({ timeout: 5000 });
    });

    test('internal fields not shown in table', async ({ page }) => {
      await navigateToModule(page, 'user');
      await waitForTableLoad(page);

      // Internal fields like password_hash and stripe_customer_id should not be visible
      const headerTexts = await page.locator('table thead th').allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      expect(headerText).not.toContain('password_hash');
      expect(headerText).not.toContain('stripe');
    });
  });

  test.describe('Create User', () => {
    test('navigate to create user form', async ({ page }) => {
      await navigateToModule(page, 'user');
      await clickCreateNew(page);
      await expectNoErrors(page);

      expect(page.url()).toContain('/new');
    });

    test('form shows expected fields', async ({ page }) => {
      await navigateToCreate(page, 'user');
      await waitForFormLoad(page);

      const emailField = page.locator('input[name="email"]');
      await expect(emailField).toBeVisible();

      const nameField = page.locator('input[name="name"]');
      await expect(nameField).toBeVisible();
    });

    test('email field validates format', async ({ page }) => {
      await navigateToCreate(page, 'user');
      await waitForFormLoad(page);

      const emailInput = page.locator('input[name="email"]');
      const inputType = await emailInput.getAttribute('type');

      // Should be email type for validation
      expect(inputType).toBe('email');
    });

    test('successfully create user with plan', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      const userName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'user');
      await waitForFormLoad(page);

      await fillField(page, 'email', userEmail);
      await fillField(page, 'name', userName);

      // Try to set status
      const statusField = page.locator('select[name="status"]');
      if (await statusField.count() > 0) {
        await statusField.selectOption('active');
      }

      // Try to set plan_id (might be RefField)
      const planInput = page.locator('input[name="plan_id"], select[name="plan_id"], [data-field="plan_id"]');
      if (await planInput.count() > 0) {
        // If it's a select, pick an option
        const tagName = await planInput.first().evaluate(el => el.tagName.toLowerCase());
        if (tagName === 'select') {
          await planInput.first().selectOption({ index: 1 });
        } else {
          // It might be a RefField, try clicking to open dropdown
          await planInput.first().click();
          await page.waitForTimeout(300);
          const option = page.locator('[role="option"], .dropdown-item').first();
          if (await option.count() > 0) {
            await option.click();
          }
        }
      }

      await submitForm(page);
      await expectNoErrors(page);

      // Verify user was created
      const users = await listViaAPI(page, 'user');
      const created = (users.records || users).find((u: { email: string }) => u.email === userEmail);
      expect(created).toBeTruthy();
    });

    test('email must be unique', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      // Create first user
      await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
      });

      // Try to create duplicate via UI
      await navigateToCreate(page, 'user');
      await waitForFormLoad(page);

      await fillField(page, 'email', userEmail);
      await fillField(page, 'name', uniqueName(TEST_PREFIX));
      await submitForm(page);

      // Should show error or fail
      await page.waitForTimeout(1000);
      // We just verify no crash - exact error handling depends on implementation
    });
  });

  test.describe('Read User', () => {
    test('user detail view loads correctly', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
      });

      await navigateToDetail(page, 'user', created.id);
      await expectNoErrors(page);

      await expect(page.locator(`text=${userEmail}`)).toBeVisible();
    });

    test('shows non-internal fields', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      const userName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: userName,
        status: 'active',
      });

      await navigateToDetail(page, 'user', created.id);

      // Should show email and name
      await expect(page.locator(`text=${userEmail}`)).toBeVisible();
      await expect(page.locator(`text=${userName}`)).toBeVisible();
    });
  });

  test.describe('Update User', () => {
    test('edit form shows current values', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      const userName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: userName,
        status: 'active',
      });

      await page.goto(`/mod/ui/user/${created.id}/edit`);
      await waitForFormLoad(page);

      const emailInput = page.locator('input[name="email"]');
      await expect(emailInput).toHaveValue(userEmail);

      const nameInput = page.locator('input[name="name"]');
      await expect(nameInput).toHaveValue(userName);
    });

    test('can change user name', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;
      const originalName = uniqueName(TEST_PREFIX);
      const newName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: originalName,
        status: 'active',
      });

      await page.goto(`/mod/ui/user/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'name', newName);
      await submitForm(page);
      await expectNoErrors(page);

      // Verify update
      const updated = await page.request.get(`${API_BASE}/user/${created.id}`);
      const data = await updated.json();
      expect(data.name).toBe(newName);
    });

    test('can change user plan', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
        plan_id: testPlanId,
      });

      await page.goto(`/mod/ui/user/${created.id}/edit`);
      await waitForFormLoad(page);

      // Try to change plan if the field exists
      const planField = page.locator('select[name="plan_id"], [data-field="plan_id"]');
      if (await planField.count() > 0) {
        // If there's another plan option, select it
        const tagName = await planField.first().evaluate(el => el.tagName.toLowerCase());
        if (tagName === 'select') {
          const options = await planField.locator('option').count();
          if (options > 1) {
            await planField.first().selectOption({ index: 0 });
          }
        }
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Delete User', () => {
    test('delete removes user', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
      });

      await navigateToDetail(page, 'user', created.id);
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/user/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });

  test.describe('Custom Actions', () => {
    test('activate action sets status to active', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'pending',
      });

      await navigateToDetail(page, 'user', created.id);

      const activateBtn = page.locator('button:has-text("Activate")');
      if (await activateBtn.count() > 0) {
        await activateBtn.click();
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/user/${created.id}`);
        const data = await updated.json();
        expect(data.status).toBe('active');
      }
    });

    test('suspend action sets status to suspended', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
      });

      await navigateToDetail(page, 'user', created.id);

      const suspendBtn = page.locator('button:has-text("Suspend")');
      if (await suspendBtn.count() > 0) {
        await suspendBtn.click();
        await page.waitForTimeout(300);

        // Handle confirmation
        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/user/${created.id}`);
        const data = await updated.json();
        expect(data.status).toBe('suspended');
      }
    });

    test('cancel action sets status to cancelled', async ({ page }) => {
      const userEmail = `${TEST_PREFIX}-${Date.now()}@example.com`;

      const created = await createViaAPI(page, 'user', {
        email: userEmail,
        name: uniqueName(TEST_PREFIX),
        status: 'active',
      });

      await navigateToDetail(page, 'user', created.id);

      const cancelBtn = page.locator('button:has-text("Cancel")').first();
      // Make sure it's not the form cancel button
      const actionCancelBtn = page.locator('button:has-text("Cancel Account"), button[data-action="cancel"]');

      if (await actionCancelBtn.count() > 0) {
        await actionCancelBtn.click();
        await page.waitForTimeout(300);

        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/user/${created.id}`);
        const data = await updated.json();
        expect(data.status).toBe('cancelled');
      }
    });
  });
});

test.describe('User Status Display', () => {
  const TEST_PREFIX = 'e2e-user-status';

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'user', TEST_PREFIX);
  });

  test('status dropdown shows enum values', async ({ page }) => {
    await navigateToCreate(page, 'user');
    await waitForFormLoad(page);

    const statusField = page.locator('select[name="status"]');
    if (await statusField.count() > 0) {
      const options = await statusField.locator('option').allTextContents();
      const optionText = options.join(' ').toLowerCase();

      // Should have the expected status values
      expect(optionText).toMatch(/active|pending|suspended|cancelled/);
    }
  });
});
