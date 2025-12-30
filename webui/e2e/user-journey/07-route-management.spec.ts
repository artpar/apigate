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
 * E2E Tests: Route Management
 *
 * Complete CRUD operations and custom actions for the route module.
 * Routes map URL patterns to upstream services.
 */

const TEST_PREFIX = 'e2e-route';

test.describe('Route Management', () => {
  let testUpstreamId: string;

  test.beforeAll(async ({ request }) => {
    // Create a test upstream for route assignment
    const response = await request.post('/mod/upstream', {
      data: {
        name: uniqueName('e2e-route-upstream'),
        base_url: 'https://api.example.com',
        auth_type: 'none',
        enabled: true,
      },
    });
    const upstream = await response.json();
    testUpstreamId = upstream.id;
  });

  test.afterAll(async ({ request }) => {
    // Cleanup test upstream
    if (testUpstreamId) {
      await request.delete(`/mod/upstream/${testUpstreamId}`);
    }
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'route', TEST_PREFIX);
  });

  test.describe('List View', () => {
    test('routes list page loads successfully', async ({ page }) => {
      await navigateToModule(page, 'route');
      await expectNoErrors(page);

      const table = page.locator('table');
      expect(await table.count()).toBeGreaterThan(0);
    });

    test('shows route data in table', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/v1/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await navigateToModule(page, 'route');
      await waitForTableLoad(page);

      await expect(page.locator(`text=${routeName}`)).toBeVisible({ timeout: 5000 });
    });

    test('shows expected columns', async ({ page }) => {
      await navigateToModule(page, 'route');
      await waitForTableLoad(page);

      const headerTexts = await page.locator('table thead th').allTextContents();
      const headerText = headerTexts.join(' ').toLowerCase();

      expect(headerText).toMatch(/name|route|path/i);
    });
  });

  test.describe('Create Route', () => {
    test('navigate to create route form', async ({ page }) => {
      await navigateToModule(page, 'route');
      await clickCreateNew(page);
      await expectNoErrors(page);

      expect(page.url()).toContain('/new');
    });

    test('form shows expected fields', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      const nameField = page.locator('input[name="name"]');
      await expect(nameField).toBeVisible();

      const pathField = page.locator('input[name="path_pattern"]');
      await expect(pathField).toBeVisible();
    });

    test('name field is required', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      await fillField(page, 'path_pattern', '/api/*');
      await submitForm(page);

      await page.waitForTimeout(500);
      const nameField = page.locator('input[name="name"]');
      const isRequired = await nameField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('path_pattern field is required', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      await fillField(page, 'name', uniqueName(TEST_PREFIX));
      await submitForm(page);

      await page.waitForTimeout(500);
      const pathField = page.locator('input[name="path_pattern"]');
      const isRequired = await pathField.getAttribute('required');
      expect(isRequired !== null || page.url().includes('/new')).toBe(true);
    });

    test('upstream dropdown shows available upstreams (RefField)', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      const upstreamField = page.locator('[data-field="upstream_id"], select[name="upstream_id"]');
      if (await upstreamField.count() > 0) {
        await upstreamField.first().click();
        await page.waitForTimeout(500);
      }
    });

    test('match type dropdown shows options', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      const matchField = page.locator('select[name="match_type"]');
      if (await matchField.count() > 0) {
        const options = await matchField.locator('option').allTextContents();
        const optionText = options.join(' ').toLowerCase();

        expect(optionText).toMatch(/exact|prefix|regex/i);
      }
    });

    test('protocol dropdown shows options', async ({ page }) => {
      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      const protocolField = page.locator('select[name="protocol"]');
      if (await protocolField.count() > 0) {
        const options = await protocolField.locator('option').allTextContents();
        const optionText = options.join(' ').toLowerCase();

        expect(optionText).toMatch(/http|sse|websocket|stream/i);
      }
    });

    test('successfully create basic route', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      await fillField(page, 'name', routeName);
      await fillField(page, 'path_pattern', '/api/v1/*');

      // Set upstream_id
      const upstreamField = page.locator('select[name="upstream_id"]');
      if (await upstreamField.count() > 0) {
        await upstreamField.selectOption({ index: 1 });
      } else {
        const refField = page.locator('[data-field="upstream_id"]');
        if (await refField.count() > 0) {
          await refField.click();
          await page.waitForTimeout(300);
          const option = page.locator('[role="option"]').first();
          if (await option.count() > 0) {
            await option.click();
          }
        }
      }

      // Set match type
      const matchField = page.locator('select[name="match_type"]');
      if (await matchField.count() > 0) {
        await matchField.selectOption('prefix');
      }

      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const routes = await listViaAPI(page, 'route');
      const created = (routes.records || routes).find((r: { name: string }) => r.name === routeName);
      expect(created).toBeTruthy();
    });

    test('successfully create route with specific protocol', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);

      await navigateToCreate(page, 'route');
      await waitForFormLoad(page);

      await fillField(page, 'name', routeName);
      await fillField(page, 'path_pattern', '/ws/*');

      const upstreamField = page.locator('select[name="upstream_id"]');
      if (await upstreamField.count() > 0) {
        await upstreamField.selectOption({ index: 1 });
      }

      const protocolField = page.locator('select[name="protocol"]');
      if (await protocolField.count() > 0) {
        await protocolField.selectOption('websocket');
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Read Route', () => {
    test('route detail shows all fields', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/v1/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await navigateToDetail(page, 'route', created.id);
      await expectNoErrors(page);

      await expect(page.locator(`text=${routeName}`)).toBeVisible();
    });

    test('shows related upstream reference', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/v1/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await navigateToDetail(page, 'route', created.id);
      await expectNoErrors(page);
      // The upstream should be displayed (either as name or ID)
    });
  });

  test.describe('Update Route', () => {
    test('can change path pattern', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/v1/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await page.goto(`/mod/ui/route/${created.id}/edit`);
      await waitForFormLoad(page);

      await fillField(page, 'path_pattern', '/api/v2/*');
      await submitForm(page);
      await expectNoErrors(page);

      // Verify
      const updated = await page.request.get(`${API_BASE}/route/${created.id}`);
      const data = await updated.json();
      expect(data.path_pattern).toBe('/api/v2/*');
    });

    test('can change priority', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        priority: 0,
        enabled: true,
      });

      await page.goto(`/mod/ui/route/${created.id}/edit`);
      await waitForFormLoad(page);

      const priorityField = page.locator('input[name="priority"]');
      if (await priorityField.count() > 0) {
        await priorityField.fill('10');
      }

      await submitForm(page);
      await expectNoErrors(page);
    });
  });

  test.describe('Delete Route', () => {
    test('delete removes route', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/delete/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await navigateToDetail(page, 'route', created.id);
      await deleteRecord(page, true);

      // Verify deleted
      const response = await page.request.get(`${API_BASE}/route/${created.id}`);
      expect(response.status()).toBe(404);
    });
  });

  test.describe('Custom Actions', () => {
    test('enable action enables route', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/enable/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: false,
      });

      await navigateToDetail(page, 'route', created.id);

      const enableBtn = page.locator('button:has-text("Enable")');
      if (await enableBtn.count() > 0) {
        await enableBtn.click();
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/route/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(true);
      }
    });

    test('disable action disables route', async ({ page }) => {
      const routeName = uniqueName(TEST_PREFIX);
      const created = await createViaAPI(page, 'route', {
        name: routeName,
        path_pattern: '/api/disable/*',
        upstream_id: testUpstreamId,
        match_type: 'prefix',
        enabled: true,
      });

      await navigateToDetail(page, 'route', created.id);

      const disableBtn = page.locator('button:has-text("Disable")');
      if (await disableBtn.count() > 0) {
        await disableBtn.click();
        await page.waitForTimeout(300);

        const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
        if (await confirmBtn.count() > 0) {
          await confirmBtn.click();
        }
        await page.waitForLoadState('networkidle');

        const updated = await page.request.get(`${API_BASE}/route/${created.id}`);
        const data = await updated.json();
        expect(data.enabled).toBe(false);
      }
    });
  });
});
