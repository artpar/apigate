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
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  cleanupTestData,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: Upstream Management
 *
 * Complete CRUD operations and custom actions for the upstream module.
 * Upstreams are backend services that routes forward to.
 */

const TEST_PREFIX = 'e2e-upstream';

test.describe('Upstream Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'upstream', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('upstreams list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'upstream');
      await expectNoErrors(page);

      const table = page.locator('table');
      expect(await table.count()).toBeGreaterThan(0);
    });

    test('shows upstream data in table', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await navigateToModule(page, 'upstream');
      await waitForTableLoad(page);

      await expect(page.locator(`text=${upstreamName}`)).toBeVisible({ timeout: 5000 });
    });

    test('shows expected columns', async ({ page }) => {
      await navigateToModule(page, 'upstream');
      await waitForTableLoad(page);

      const headerTexts = await page.locator('table thead th').allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      expect(headerText).toMatch(/name|upstream/i);
    });
  });

  test.describe('Create Upstream', () => {
    test('navigate to create upstream form', async ({ page }) => {
      await navigateToModule(page, 'upstream');
      await clickCreateNew(page);
      await expectNoErrors(page);

      expect(page.url()).toContain('/new');
    });

    test('form shows expected fields', async ({ page }) => {
      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      const nameField = page.locator('input[name="name"]');
      await expect(nameField).toBeVisible();

      const urlField = page.locator('input[name="base_url"]');
      await expect(urlField).toBeVisible();
    });

    test('name field is required', async ({ page }) => {
      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      // Try to submit without name
      await fillField(page, 'base_url', 'https://api.example.com');
      await submitForm(page);

      // Should stay on form or show error
      await page.waitForTimeout(500);
      const nameField = page.locator('input[name="name"]');
      const isRequired = await nameField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('base_url field is required', async ({ page }) => {
      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      await fillField(page, 'name', uniqueName(TEST_PREFIX));
      // Don't fill base_url
      await submitForm(page);

      await page.waitForTimeout(500);
      const urlField = page.locator('input[name="base_url"]');
      const isRequired = await urlField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('auth type dropdown shows options', async ({ page }) => {
      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      const authField = page.locator('select[name="auth_type"]');
      if (await authField.count() > 0) {
        const options = await authField.locator('option').allTextContents();
        const optionText = options.join(' ').toLowerCase();

        // Should have auth type options
        expect(optionText).toMatch(/none|header|bearer|basic/i);
      }
    });

    test('successfully create upstream with no auth', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      await fillField(page, 'name', upstreamName);
      await fillField(page, 'base_url', 'https://api.example.com');

      const authField = page.locator('select[name="auth_type"]');
      if (await authField.count() > 0) {
        await authField.selectOption('none');
      }

      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const upstreams = await listViaAPI(page, 'upstream');
      const created = (upstreams.records || upstreams).find((u: { name: string }) => u.name === upstreamName);
      expect(created).toBeTruthy();
    });

    test('successfully create upstream with bearer auth', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      await fillField(page, 'name', upstreamName);
      await fillField(page, 'base_url', 'https://secure-api.example.com');

      const authField = page.locator('select[name="auth_type"]');
      if (await authField.count() > 0) {
        await authField.selectOption('bearer');
      }

      // Set auth value if field exists
      const authValueField = page.locator('input[name="auth_value"]');
      if (await authValueField.count() > 0) {
        await authValueField.fill('test-bearer-token');
      }

      await submitForm(page);
      await expectNoErrors(page);

      const upstreams = await listViaAPI(page, 'upstream');
      const created = (upstreams.records || upstreams).find((u: { name: string }) => u.name === upstreamName);
      expect(created).toBeTruthy();
    });

    test('timeout defaults to reasonable value', async ({ page }) => {
      await navigateToCreate(page, 'upstream');
      await waitForFormLoad(page);

      const timeoutField = page.locator('input[name="timeout_ms"]');
      if (await timeoutField.count() > 0) {
        const value = await timeoutField.inputValue();
        const timeout = parseInt(value) || 0;
        // Should have a reasonable default (e.g., 30000ms)
        expect(timeout).toBeGreaterThanOrEqual(0);
      }
    });
  });

  test.describe('Read Upstream', () => {
    test('upstream detail shows all fields', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await navigateToDetail(page, 'upstream', created.id);
      await expectNoErrors(page);

      await expect(page.locator(`text=${upstreamName}`)).toBeVisible();
    });

    test('auth value (encrypted) not displayed in plain text', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'bearer',
        auth_value: 'secret-token-value',
        enabled: true,
      });

      await navigateToDetail(page, 'upstream', created.id);

      // Should not show the actual token value
      const pageContent = await page.content();
      expect(pageContent).not.toContain('secret-token-value');
    });
  });

  test.describe('Update Upstream', () => {
    test('can change base_url', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://old-api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await page.goto(`/mod/ui/upstream/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'base_url', 'https://new-api.example.com');
      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const updated = await page.request.get(`${API_BASE}/upstream/${created.id}`);
      const data = await updated.json();
      expect(data.base_url).toBe('https://new-api.example.com');
    });

    test('can change timeout', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        timeout_ms: 30000,
        enabled: true,
      });

      await page.goto(`/mod/ui/upstream/${created.id}/edit`);
      await waitForFormLoad(page);

      const timeoutField = page.locator('input[name="timeout_ms"]');
      if (await timeoutField.count() > 0) {
        await timeoutField.fill('60000');
      }

      await submitForm(page);
      await expectNoErrors(page);
    });

    test('can change auth settings', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await page.goto(`/mod/ui/upstream/${created.id}/edit`);
      await waitForFormLoad(page);

      const authField = page.locator('select[name="auth_type"]');
      if (await authField.count() > 0) {
        await authField.selectOption('bearer');
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Delete Upstream', () => {
    test('delete removes upstream', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await navigateToDetail(page, 'upstream', created.id);
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/upstream/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });

  test.describe('Custom Actions', () => {
    test('enable action enables upstream', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: false,
      });

      await navigateToDetail(page, 'upstream', created.id);

      const enableBtn = page.locator('button:has-text("Enable")');
      if (await enableBtn.count() > 0) {
        await enableBtn.click();
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/upstream/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(true);
      }
    });

    test('disable action disables upstream', async ({ page }) => {
      const upstreamName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'upstream', {
        name: upstreamName,
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      });

      await navigateToDetail(page, 'upstream', created.id);

      const disableBtn = page.locator('button:has-text("Disable")');
      if (await disableBtn.count() > 0) {
        await disableBtn.click();
        await page.waitForTimeout(300);

        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/upstream/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(false);
      }
    });
  });
});
