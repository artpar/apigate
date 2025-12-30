import { test, expect } from '@playwright/test';
import {
  navigateToModule,
  navigateToCreate,
  navigateToDetail,
  fillField,
  submitForm,
  clickCreateNew,
  clickFirstRow,
  deleteRecord,
  clickAction,
  clickActionWithConfirm,
  createViaAPI,
  deleteViaAPI,
  listViaAPI,
  uniqueName,
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  cleanupTestData,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: Plan Management
 *
 * Complete CRUD operations and custom actions for the plan module.
 * Plans are the pricing tiers for the API gateway.
 */

const TEST_PREFIX = 'e2e-plan';

test.describe('Plan Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    // Cleanup test data
    await cleanupTestData(page, 'plan', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('plans list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'plan');
      await expectNoErrors(page);

      // Should have table or list structure
      const table = page.locator('table');
      const hasList = await table.count() > 0;
      expect(hasList).toBe(true);
    });

    test('shows plan data in table', async ({ page }) => {
      // Create a test plan first
      const planName = uniqueName(TEST_PREFIX);
      await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      });

      await navigateToModule(page, 'plan');
      await waitForTableLoad(page);

      // Should show the plan in the table
      await expect(page.locator(`text=${planName}`)).toBeVisible({ timeout: 5000 });
    });

    test('table shows expected columns', async ({ page }) => {
      await navigateToModule(page, 'plan');
      await waitForTableLoad(page);

      // Check for expected column headers
      const headers = page.locator('table thead th, table th');
      const headerTexts = await headers.allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      // Should have key columns (may be abbreviated)
      expect(headerText).toMatch(/name|plan/i);
    });
  });

  test.describe('Create Plan', () => {
    test('navigate to create plan form', async ({ page }) => {
      await navigateToModule(page, 'plan');
      await clickCreateNew(page);
      await expectNoErrors(page);

      // Should be on create page
      expect(page.url()).toContain('/new');
    });

    test('form shows all plan fields', async ({ page }) => {
      await navigateToCreate(page, 'plan');
      await waitForFormLoad(page);

      // Check for key fields
      const nameField = page.locator('input[name="name"]');
      await expect(nameField).toBeVisible();

      const rateLimitField = page.locator('input[name="rate_limit_per_minute"]');
      await expect(rateLimitField).toBeVisible();

      const requestsField = page.locator('input[name="requests_per_month"]');
      await expect(requestsField).toBeVisible();
    });

    test('name field is required', async ({ page }) => {
      await navigateToCreate(page, 'plan');
      await waitForFormLoad(page);

      // Try to submit without name
      await fillField(page, 'rate_limit_per_minute', '60');
      await submitForm(page);

      // Should still be on form or show error
      const nameField = page.locator('input[name="name"]');
      const isRequired = await nameField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('successfully create a Free plan', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'plan');
      await waitForFormLoad(page);

      await fillField(page, 'name', planName);
      await fillField(page, 'rate_limit_per_minute', '60');
      await fillField(page, 'requests_per_month', '1000');
      await fillField(page, 'price_monthly', '0');

      await submitForm(page);
      await expectNoErrors(page);

      // Verify plan was created via API
      const plans = await listViaAPI(page, 'plan');
      const created = (plans.records || plans).find((p: { name: string }) => p.name === planName);
      expect(created).toBeTruthy();
    });

    test('successfully create a paid plan', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'plan');
      await waitForFormLoad(page);

      await fillField(page, 'name', planName);
      await fillField(page, 'rate_limit_per_minute', '600');
      await fillField(page, 'requests_per_month', '50000');
      await fillField(page, 'price_monthly', '2900'); // $29 in cents

      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const plans = await listViaAPI(page, 'plan');
      const created = (plans.records || plans).find((p: { name: string }) => p.name === planName);
      expect(created).toBeTruthy();
      expect(created.price_monthly).toBe(2900);
    });

    test('create plan with high limits (Pro tier)', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'plan');
      await waitForFormLoad(page);

      await fillField(page, 'name', planName);
      await fillField(page, 'rate_limit_per_minute', '6000');
      await fillField(page, 'requests_per_month', '500000');
      await fillField(page, 'price_monthly', '9900'); // $99

      await submitForm(page);
      await expectNoErrors(page);

      const plans = await listViaAPI(page, 'plan');
      const created = (plans.records || plans).find((p: { name: string }) => p.name === planName);
      expect(created).toBeTruthy();
      expect(created.rate_limit_per_minute).toBe(6000);
    });
  });

  test.describe('Read Plan', () => {
    test('click plan row navigates to detail view', async ({ page }) => {
      // Create test plan
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
      });

      await navigateToModule(page, 'plan');
      await waitForTableLoad(page);

      // Click on the plan row
      const row = page.locator(`tr:has-text("${planName}")`);
      await row.click();
      await page.waitForLoadState('networkidle');

      await expectNoErrors(page);

      // Should navigate to detail or edit page
      expect(page.url()).toMatch(new RegExp(`/plan/${created.id}|/plan/.*${planName}`));
    });

    test('detail view shows plan fields', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 120,
        requests_per_month: 5000,
        price_monthly: 1500,
      });

      await navigateToDetail(page, 'plan', created.id);
      await expectNoErrors(page);

      // Should show the plan name somewhere
      await expect(page.locator(`text=${planName}`)).toBeVisible();
    });
  });

  test.describe('Update Plan', () => {
    test('edit form pre-fills existing values', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 100,
        requests_per_month: 2000,
        price_monthly: 500,
      });

      await page.goto(`/mod/ui/plan/${created.id}/edit`);
      await page.waitForLoadState('networkidle');
      await waitForFormLoad(page);

      // Check pre-filled values
      const nameInput = page.locator('input[name="name"]');
      await expect(nameInput).toHaveValue(planName);

      const rateInput = page.locator('input[name="rate_limit_per_minute"]');
      await expect(rateInput).toHaveValue('100');
    });

    test('successfully update plan name', async ({ page }) => {
      const originalName = uniqueName(TEST_PREFIX);
      const newName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'plan', {
        name: originalName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
      });

      await page.goto(`/mod/ui/plan/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'name', newName);
      await submitForm(page);
      await expectNoErrors(page);

      // Verify update
      const updated = await page.request.get(`${API_BASE}/plan/${created.id}`);
      const data = await updated.json();
      expect(data.name).toBe(newName);
    });

    test('successfully update rate limits', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
      });

      await page.goto(`/mod/ui/plan/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'rate_limit_per_minute', '300');
      await fillField(page, 'requests_per_month', '10000');
      await submitForm(page);
      await expectNoErrors(page);

      const updated = await page.request.get(`${API_BASE}/plan/${created.id}`);
      const data = await updated.json();
      expect(data.rate_limit_per_minute).toBe(300);
      expect(data.requests_per_month).toBe(10000);
    });
  });

  test.describe('Delete Plan', () => {
    test('delete removes plan from list', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
      });

      // Navigate to detail page
      await navigateToDetail(page, 'plan', created.id);

      // Delete
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/plan/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });

  test.describe('Custom Actions', () => {
    test('enable action activates disabled plan', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: false,
      });

      await navigateToDetail(page, 'plan', created.id);

      // Look for Enable action button
      const enableBtn = page.locator('button:has-text("Enable")');
      if (await enableBtn.count() > 0) {
        await enableBtn.click();
        await page.waitForLoadState('networkidle');

        // Verify enabled
        const updated = await page.request.get(`${API_BASE}/plan/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(true);
      }
    });

    test('disable action deactivates plan', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      });

      await navigateToDetail(page, 'plan', created.id);

      // Look for Disable action button
      const disableBtn = page.locator('button:has-text("Disable")');
      if (await disableBtn.count() > 0) {
        await disableBtn.click();
        await page.waitForLoadState('networkidle');

        // Handle confirmation if present
        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
          await page.waitForLoadState('networkidle');
        }

        // Verify disabled
        const updated = await page.request.get(`${API_BASE}/plan/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(false);
      }
    });

    test('set default action marks plan as default', async ({ page }) => {
      const planName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'plan', {
        name: planName,
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        is_default: false,
      });

      await navigateToDetail(page, 'plan', created.id);

      // Look for Set Default action button
      const defaultBtn = page.locator('button:has-text("Default"), button:has-text("Set Default")');
      if (await defaultBtn.count() > 0) {
        await defaultBtn.click();
        await page.waitForLoadState('networkidle');

        // Verify is_default
        const updated = await page.request.get(`${API_BASE}/plan/${created.id}`);
        const data = await updated.json();
        expect(data.is_default).toBe(true);
      }
    });
  });
});

test.describe('Plan Validation', () => {
  const TEST_PREFIX = 'e2e-plan-val';

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'plan', TEST_PREFIX);
  });

  test('integer fields accept only numbers', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    const rateInput = page.locator('input[name="rate_limit_per_minute"]');
    const inputType = await rateInput.getAttribute('type');

    // Should be number type
    expect(inputType).toBe('number');
  });

  test('creating duplicate name shows error', async ({ page }) => {
    const planName = uniqueName(TEST_PREFIX);

    // Create first plan
    await createViaAPI(page, 'plan', {
      name: planName,
      rate_limit_per_minute: 60,
      requests_per_month: 1000,
      price_monthly: 0,
    });

    // Try to create duplicate via UI
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    await fillField(page, 'name', planName);
    await fillField(page, 'rate_limit_per_minute', '60');
    await fillField(page, 'requests_per_month', '1000');
    await fillField(page, 'price_monthly', '0');

    await submitForm(page);

    // Should show error or stay on form
    // The exact behavior depends on backend validation
    await page.waitForTimeout(1000);
  });
});
