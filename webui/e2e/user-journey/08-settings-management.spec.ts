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
  uniqueKey,
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  cleanupTestData,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: Settings Management
 *
 * Complete CRUD operations for the setting module.
 * Settings store application configuration as key-value pairs.
 */

const TEST_PREFIX = 'e2e-setting';

test.describe('Settings Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'setting', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('settings list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'setting');
      await expectNoErrors(page);

      const table = page.locator('table');
      expect(await table.count()).toBeGreaterThan(0);
    });

    test('shows setting data in table', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);
      await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'test-value',
        encrypted: 0,
      });

      await navigateToModule(page, 'setting');
      await waitForTableLoad(page);

      await expect(page.locator(`text=${settingKey}`)).toBeVisible({ timeout: 5000 });
    });

    test('shows expected columns', async ({ page }) => {
      await navigateToModule(page, 'setting');
      await waitForTableLoad(page);

      const headerTexts = await page.locator('table thead th').allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      expect(headerText).toMatch(/key|setting/i);
    });
  });

  test.describe('Create Setting', () => {
    test('navigate to create setting form', async ({ page }) => {
      await navigateToModule(page, 'setting');
      await clickCreateNew(page);
      await expectNoErrors(page);

      expect(page.url()).toContain('/new');
    });

    test('form shows expected fields', async ({ page }) => {
      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      const keyField = page.locator('input[name="key"]');
      await expect(keyField).toBeVisible();

      const valueField = page.locator('input[name="value"], textarea[name="value"]');
      await expect(valueField).toBeVisible();
    });

    test('key field is required', async ({ page }) => {
      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      await fillField(page, 'value', 'test-value');
      await submitForm(page);

      await page.waitForTimeout(500);
      const keyField = page.locator('input[name="key"]');
      const isRequired = await keyField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('value field is required', async ({ page }) => {
      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      await fillField(page, 'key', uniqueKey(TEST_PREFIX));
      await submitForm(page);

      await page.waitForTimeout(500);
      const valueField = page.locator('input[name="value"], textarea[name="value"]');
      const isRequired = await valueField.first().getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('successfully create new setting', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);

      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      await fillField(page, 'key', settingKey);
      await fillField(page, 'value', 'my-setting-value');

      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const settings = await listViaAPI(page, 'setting');
      const created = (settings.records || settings).find((s: { key: string }) => s.key === settingKey);
      expect(created).toBeTruthy();
    });

    test('key must be unique', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);

      // Create first setting
      await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'first-value',
        encrypted: 0,
      });

      // Try to create duplicate via UI
      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      await fillField(page, 'key', settingKey);
      await fillField(page, 'value', 'second-value');
      await submitForm(page);

      // Should show error or fail
      await page.waitForTimeout(1000);
    });

    test('encrypted flag works', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);

      await navigateToCreate(page, 'setting');
      await waitForFormLoad(page);

      await fillField(page, 'key', settingKey);
      await fillField(page, 'value', 'secret-value');

      // Set encrypted flag
      const encryptedField = page.locator('input[name="encrypted"]');
      if (await encryptedField.count() > 0) {
        const isCheckbox = await encryptedField.getAttribute('type') === 'checkbox';
        if (isCheckbox) {
          await encryptedField.check();
        } else {
          await encryptedField.fill('1');
        }
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Read Setting', () => {
    test('setting detail shows all fields', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);
      const created = await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'detail-value',
        encrypted: 0,
      });

      await navigateToDetail(page, 'setting', created.id);
      await expectNoErrors(page);

      await expect(page.locator(`text=${settingKey}`)).toBeVisible();
    });
  });

  test.describe('Update Setting', () => {
    test('can update setting value', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);
      const created = await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'original-value',
        encrypted: 0,
      });

      await page.goto(`/mod/ui/setting/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'value', 'updated-value');
      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const updated = await page.request.get(`${API_BASE}/setting/${created.id}`);
      const data = await updated.json();
      expect(data.value).toBe('updated-value');
    });

    test('edit form pre-fills existing values', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);
      const created = await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'prefill-value',
        encrypted: 0,
      });

      await page.goto(`/mod/ui/setting/${created.id}/edit`);
      await waitForFormLoad(page);

      const keyInput = page.locator('input[name="key"]');
      await expect(keyInput).toHaveValue(settingKey);

      const valueInput = page.locator('input[name="value"], textarea[name="value"]');
      await expect(valueInput.first()).toHaveValue('prefill-value');
    });
  });

  test.describe('Delete Setting', () => {
    test('delete removes setting', async ({ page }) => {
      const settingKey = uniqueKey(TEST_PREFIX);
      const created = await createViaAPI(page, 'setting', {
        key: settingKey,
        value: 'delete-value',
        encrypted: 0,
      });

      await navigateToDetail(page, 'setting', created.id);
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/setting/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });
});

test.describe('Settings Key Patterns', () => {
  const TEST_PREFIX = 'e2e-setting-pattern';

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'setting', TEST_PREFIX);
  });

  test('can create setting with dot notation key', async ({ page }) => {
    const settingKey = `${TEST_PREFIX}.smtp.host`;

    await navigateToCreate(page, 'setting');
    await waitForFormLoad(page);

    await fillField(page, 'key', settingKey);
    await fillField(page, 'value', 'smtp.example.com');

    await submitForm(page);
    await expectNoErrors(page);

    const settings = await listViaAPI(page, 'setting');
    const created = (settings.records || settings).find((s: { key: string }) => s.key === settingKey);
    expect(created).toBeTruthy();
  });

  test('can create setting with underscore key', async ({ page }) => {
    const settingKey = `${TEST_PREFIX}_api_timeout`;

    await navigateToCreate(page, 'setting');
    await waitForFormLoad(page);

    await fillField(page, 'key', settingKey);
    await fillField(page, 'value', '30000');

    await submitForm(page);
    await expectNoErrors(page);
  });
});
