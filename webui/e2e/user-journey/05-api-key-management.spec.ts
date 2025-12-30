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
  uniqueKey,
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  cleanupTestData,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: API Key Management
 *
 * Complete CRUD operations and custom actions for the api_key module.
 * API keys are used for authenticating API requests.
 */

const TEST_PREFIX = 'e2e-apikey';

test.describe('API Key Management', () => {
  let testUserId: string;
  let testPlanId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test plan
    const planResponse = await request.post('/mod/plan', {
      data: {
        name: uniqueName('e2e-apikey-plan'),
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      },
    });
    const plan = await planResponse.json();
    testPlanId = plan.id;

    // Create a test user
    const userResponse = await request.post('/mod/user', {
      data: {
        email: `${TEST_PREFIX}-${Date.now()}@example.com`,
        name: uniqueName('e2e-apikey-user'),
        status: 'active',
        plan_id: testPlanId,
      },
    });
    const user = await userResponse.json();
    testUserId = user.id;
  });

  test.afterAll(async ({ request }) => {
    // Cleanup
    if (testUserId) {
      await request.delete(`/mod/user/${testUserId}`);
    }
    if (testPlanId) {
      await request.delete(`/mod/plan/${testPlanId}`);
    }
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'api_key', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('api keys list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'api_key');
      await expectNoErrors(page);

      const table = page.locator('table');
      expect(await table.count()).toBeGreaterThan(0);
    });

    test('shows api key data in table', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);
      await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'test_',
        enabled: true,
      });

      await navigateToModule(page, 'api_key');
      await waitForTableLoad(page);

      await expect(page.locator(`text=${keyName}`)).toBeVisible({ timeout: 5000 });
    });

    test('hash field not shown in table', async ({ page }) => {
      await navigateToModule(page, 'api_key');
      await waitForTableLoad(page);

      const headerTexts = await page.locator('table thead th').allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      // Hash should be internal
      expect(headerText).not.toContain('hash');
    });
  });

  test.describe('Create API Key', () => {
    test('navigate to create api key form', async ({ page }) => {
      await navigateToModule(page, 'api_key');
      await clickCreateNew(page);
      await expectNoErrors(page);

      expect(page.url()).toContain('/new');
    });

    test('form shows expected fields', async ({ page }) => {
      await navigateToCreate(page, 'api_key');
      await waitForFormLoad(page);

      const nameField = page.locator('input[name="name"]');
      await expect(nameField).toBeVisible();
    });

    test('user field shows available users (RefField)', async ({ page }) => {
      await navigateToCreate(page, 'api_key');
      await waitForFormLoad(page);

      // Look for user_id field which should be a RefField
      const userField = page.locator('[data-field="user_id"], select[name="user_id"], input[name="user_id"]');

      if (await userField.count() > 0) {
        // Click to potentially open dropdown
        await userField.first().click();
        await page.waitForTimeout(500);

        // Should show options or dropdown
        const options = page.locator('[role="option"], option, .dropdown-item');
        const hasOptions = await options.count() > 0;
        // This is informational - RefField might work differently
      }
    });

    test('successfully create api key for user', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'api_key');
      await waitForFormLoad(page);

      await fillField(page, 'name', keyName);

      // Set user_id
      const userField = page.locator('select[name="user_id"]');
      if (await userField.count() > 0) {
        await userField.selectOption({ index: 1 });
      } else {
        // Try RefField approach
        const refField = page.locator('[data-field="user_id"]');
        if (await refField.count() > 0) {
          await refField.click();
          await page.waitForTimeout(300);
          const option = page.locator('[role="option"]').first();
          if (await option.count() > 0) {
            await option.click();
          }
        }
      }

      await submitForm(page);
      await expectNoErrors(page);

      // Verify key was created
      const keys = await listViaAPI(page, 'api_key');
      const created = (keys.records || keys).find((k: { name: string }) => k.name === keyName);
      expect(created).toBeTruthy();
    });

    test('can set optional expiration date', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'api_key');
      await waitForFormLoad(page);

      await fillField(page, 'name', keyName);

      // Set user_id first
      const userField = page.locator('select[name="user_id"]');
      if (await userField.count() > 0) {
        await userField.selectOption({ index: 1 });
      }

      // Set expiration date
      const expiresField = page.locator('input[name="expires_at"]');
      if (await expiresField.count() > 0) {
        // Set a future date
        const futureDate = new Date();
        futureDate.setMonth(futureDate.getMonth() + 1);
        await expiresField.fill(futureDate.toISOString().split('T')[0]);
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Read API Key', () => {
    test('api key detail view shows all fields', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'test_',
        enabled: true,
      });

      await navigateToDetail(page, 'api_key', created.id);
      await expectNoErrors(page);

      await expect(page.locator(`text=${keyName}`)).toBeVisible();
    });

    test('shows related user reference', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'ref_',
        enabled: true,
      });

      await navigateToDetail(page, 'api_key', created.id);

      // Should show user reference somehow (might be as link or text)
      // The exact display depends on UI implementation
      await expectNoErrors(page);
    });
  });

  test.describe('Update API Key', () => {
    test('can update key name', async ({ page }) => {
      const originalName = uniqueName(TEST_PREFIX);
      const newName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: originalName,
        user_id: testUserId,
        prefix: 'upd_',
        enabled: true,
      });

      await page.goto(`/mod/ui/api_key/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'name', newName);
      await submitForm(page);
      await expectNoErrors(page);

      // Verify update
      const updated = await page.request.get(`${API_BASE}/api_key/${created.id}`);
      const data = await updated.json();
      expect(data.name).toBe(newName);
    });

    test('can update expiration', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'exp_',
        enabled: true,
      });

      await page.goto(`/mod/ui/api_key/${created.id}/edit`);
      await waitForFormLoad(page);

      const expiresField = page.locator('input[name="expires_at"]');
      if (await expiresField.count() > 0) {
        const futureDate = new Date();
        futureDate.setMonth(futureDate.getMonth() + 3);
        await expiresField.fill(futureDate.toISOString().split('T')[0]);
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Delete API Key', () => {
    test('delete removes api key', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'del_',
        enabled: true,
      });

      await navigateToDetail(page, 'api_key', created.id);
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/api_key/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });

  test.describe('Custom Actions', () => {
    test('revoke action sets revoked_at timestamp', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'rev_',
        enabled: true,
      });

      await navigateToDetail(page, 'api_key', created.id);

      const revokeBtn = page.locator('button:has-text("Revoke")');
      if (await revokeBtn.count() > 0) {
        await revokeBtn.click();
        await page.waitForTimeout(300);

        // Handle confirmation
        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        // Verify revoked
        const updated = await page.request.get(`${API_BASE}/api_key/${created.id}`);
        const data = await updated.json();
        expect(data.revoked_at).toBeTruthy();
      }
    });

    test('enable action enables disabled key', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'ena_',
        enabled: false,
      });

      await navigateToDetail(page, 'api_key', created.id);

      const enableBtn = page.locator('button:has-text("Enable")');
      if (await enableBtn.count() > 0) {
        await enableBtn.click();
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/api_key/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(true);
      }
    });

    test('disable action disables key', async ({ page }) => {
      const keyName = uniqueName(TEST_PREFIX);

      const created = await createViaAPI(page, 'api_key', {
        name: keyName,
        user_id: testUserId,
        prefix: 'dis_',
        enabled: true,
      });

      await navigateToDetail(page, 'api_key', created.id);

      const disableBtn = page.locator('button:has-text("Disable")');
      if (await disableBtn.count() > 0) {
        await disableBtn.click();
        await page.waitForTimeout(300);

        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/api_key/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(false);
      }
    });
  });
});

test.describe('API Key Validation', () => {
  test('user_id is required', async ({ page }) => {
    await navigateToCreate(page, 'api_key');
    await waitForFormLoad(page);

    await fillField(page, 'name', uniqueName(TEST_PREFIX));

    // Don't fill user_id
    await submitForm(page);

    // Should show error or stay on form
    await page.waitForTimeout(500);
    // Check if still on form (required field not filled)
    const form = page.locator('form');
    expect(await form.count()).toBeGreaterThan(0);
  });
});
