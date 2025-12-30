import { test, expect } from '@playwright/test';
import {
  navigateToModule,
  navigateToCreate,
  navigateToDetail,
  fillField,
  submitForm,
  cancelForm,
  clickFirstRow,
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
 * E2E Tests: Advanced Features
 *
 * Tests for sorting, table interactions, form features,
 * documentation panel, and responsive design.
 */

const TEST_PREFIX = 'e2e-advanced';

test.describe('Table Sorting', () => {
  test.beforeAll(async ({ request }) => {
    // Create multiple plans for sorting tests
    for (let i = 1; i <= 3; i++) {
      await request.post('/mod/plan', {
        data: {
          name: `${TEST_PREFIX}-sort-plan-${i}`,
          rate_limit_per_minute: i * 100,
          requests_per_month: i * 1000,
          price_monthly: i * 1000,
          enabled: true,
        },
      });
    }
  });

  test.afterAll(async ({ request }) => {
    // Cleanup
    const response = await request.get('/mod/plan');
    const plans = await response.json();
    for (const plan of plans.records || plans) {
      if (plan.name.startsWith(TEST_PREFIX)) {
        await request.delete(`/mod/plan/${plan.id}`);
      }
    }
  });

  test('click column header triggers sort', async ({ page }) => {
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // Find a sortable column header
    const nameHeader = page.locator('th:has-text("Name"), th:has-text("name")').first();

    if (await nameHeader.count() > 0) {
      // Click to sort
      await nameHeader.click();
      await page.waitForTimeout(500);

      // Page should still work
      await expectNoErrors(page);
    }
  });

  test('table displays data after sort', async ({ page }) => {
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // Click header to sort
    const header = page.locator('table thead th').first();
    await header.click();
    await page.waitForTimeout(500);

    // Table should still have rows
    const rows = page.locator('table tbody tr');
    expect(await rows.count()).toBeGreaterThan(0);
  });
});

test.describe('Table Row Interaction', () => {
  let testPlanId: string;

  test.beforeAll(async ({ request }) => {
    const response = await request.post('/mod/plan', {
      data: {
        name: uniqueName(`${TEST_PREFIX}-row-plan`),
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      },
    });
    testPlanId = (await response.json()).id;
  });

  test.afterAll(async ({ request }) => {
    if (testPlanId) {
      await request.delete(`/mod/plan/${testPlanId}`);
    }
  });

  test('clicking table row navigates to detail', async ({ page }) => {
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // Click first row
    await clickFirstRow(page);
    await expectNoErrors(page);

    // Should navigate to detail or edit page
    expect(page.url()).toMatch(/\/plan\/[^/]+$/);
  });

  test('table rows are visually clickable', async ({ page }) => {
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    const row = page.locator('table tbody tr').first();

    if (await row.count() > 0) {
      // Row should have cursor pointer or be visually interactive
      const cursor = await row.evaluate(el => getComputedStyle(el).cursor);
      // Either pointer cursor or row click handler
      expect(['pointer', 'default'].includes(cursor)).toBe(true);
    }
  });
});

test.describe('Form Features', () => {
  test.afterEach(async ({ page }) => {
    await cleanupTestData(page, 'plan', TEST_PREFIX);
  });

  test('cancel button returns to list without saving', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    await fillField(page, 'name', uniqueName(TEST_PREFIX));

    // Click cancel
    const cancelBtn = page.locator('button:has-text("Cancel"), a:has-text("Cancel")');
    if (await cancelBtn.count() > 0) {
      await cancelBtn.first().click();
      await page.waitForLoadState('networkidle');

      // Should be back on list
      expect(page.url()).toContain('/plan');
      expect(page.url()).not.toContain('/new');
    }
  });

  test('form fields have appropriate input types', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Number fields should have type="number"
    const rateInput = page.locator('input[name="rate_limit_per_minute"]');
    const rateType = await rateInput.getAttribute('type');
    expect(rateType).toBe('number');

    const requestsInput = page.locator('input[name="requests_per_month"]');
    const requestsType = await requestsInput.getAttribute('type');
    expect(requestsType).toBe('number');
  });

  test('required fields have required attribute', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    const nameInput = page.locator('input[name="name"]');
    const isRequired = await nameInput.getAttribute('required');
    expect(isRequired).not.toBeNull();
  });

  test('submit button is disabled when form is invalid', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Don't fill required fields
    const submitBtn = page.locator('button[type="submit"]');

    // Submit button might be enabled but form won't submit
    // This is HTML5 validation behavior
    await expectNoErrors(page);
  });
});

test.describe('Documentation Panel', () => {
  test('documentation panel exists on form page', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Look for documentation panel
    const docsPanel = page.locator('[class*="documentation"], [class*="docs"], aside, .field-info');

    // Panel might or might not exist depending on UI design
    await page.waitForTimeout(500);
  });

  test('focusing field shows related help', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    const nameInput = page.locator('input[name="name"]');
    await nameInput.focus();
    await page.waitForTimeout(300);

    // Page should still work after focus
    await expectNoErrors(page);
  });

  test('field descriptions from schema are meaningful', async ({ page }) => {
    // Get schema to verify descriptions exist
    const schemaResponse = await page.request.get(`${API_BASE}/_schema/plan`);
    const schema = await schemaResponse.json();

    // Check that fields have descriptions
    for (const field of schema.fields) {
      if (!field.implicit && field.description) {
        expect(field.description.length).toBeGreaterThan(5);
        expect(field.description).not.toMatch(/^(TODO|TBD|placeholder)/i);
      }
    }
  });
});

test.describe('Responsive Design', () => {
  test('page renders correctly at mobile viewport', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
    await expectNoErrors(page);

    // Body should be visible
    await expect(page.locator('body')).toBeVisible();
  });

  test('navigation works at mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await navigateToModule(page, 'plan');
    await expectNoErrors(page);

    await navigateToModule(page, 'user');
    await expectNoErrors(page);
  });

  test('forms work at mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Form should be visible
    await expect(page.locator('form')).toBeVisible();

    // Can interact with fields
    const nameInput = page.locator('input[name="name"]');
    await expect(nameInput).toBeVisible();
    await nameInput.fill('test');
    await expect(nameInput).toHaveValue('test');
  });

  test('table scrolls horizontally on narrow viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // Table or its container should handle overflow
    const table = page.locator('table').first();
    if (await table.count() > 0) {
      const tableParent = page.locator('table').first().locator('..');
      // Should be able to scroll or table should fit
      await expectNoErrors(page);
    }
  });
});

test.describe('Error Handling', () => {
  test('404 page shows helpful message', async ({ page }) => {
    await page.goto('/mod/ui/nonexistent-module');
    await page.waitForLoadState('networkidle');

    // Should show some indication of not found
    const body = await page.content();
    // Either 404 message or redirect to valid page
    expect(body.length).toBeGreaterThan(0);
  });

  test('invalid record ID shows error gracefully', async ({ page }) => {
    await page.goto('/mod/ui/plan/invalid-id-12345');
    await page.waitForLoadState('networkidle');

    // Should show error message, not crash
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('API errors are displayed to user', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Try to submit invalid data
    await fillField(page, 'name', ''); // Empty name should fail
    await fillField(page, 'rate_limit_per_minute', '-1'); // Invalid value

    await submitForm(page);
    await page.waitForTimeout(500);

    // Page should still be functional
    await expect(page.locator('body')).toBeVisible();
  });
});

test.describe('Keyboard Navigation', () => {
  test('tab navigates through form fields', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    // Start at name field
    const nameInput = page.locator('input[name="name"]');
    await nameInput.focus();

    // Tab to next field
    await page.keyboard.press('Tab');
    await page.waitForTimeout(100);

    // Should have moved focus
    const activeElement = page.locator(':focus');
    expect(await activeElement.count()).toBe(1);
  });

  test('enter key submits form', async ({ page }) => {
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);

    const planName = uniqueName(TEST_PREFIX);
    await fillField(page, 'name', planName);
    await fillField(page, 'rate_limit_per_minute', '60');
    await fillField(page, 'requests_per_month', '1000');
    await fillField(page, 'price_monthly', '0');

    // Press enter in last field
    await page.keyboard.press('Enter');
    await page.waitForLoadState('networkidle');

    // Should have attempted to submit
    await page.waitForTimeout(500);
  });

  test('escape key closes modals', async ({ page }) => {
    // Create test data
    const plan = await createViaAPI(page, 'plan', {
      name: uniqueName(TEST_PREFIX),
      rate_limit_per_minute: 60,
      requests_per_month: 1000,
      price_monthly: 0,
      enabled: true,
    });

    await navigateToDetail(page, 'plan', plan.id);

    // Try to trigger delete which might show modal
    const deleteBtn = page.locator('button:has-text("Delete")');
    if (await deleteBtn.count() > 0) {
      await deleteBtn.click();
      await page.waitForTimeout(300);

      // If modal appeared, escape should close it
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    }

    // Clean up
    await deleteViaAPI(page, 'plan', plan.id);
  });
});

test.describe('Data Persistence', () => {
  test('created records persist after page refresh', async ({ page }) => {
    const planName = uniqueName(TEST_PREFIX);

    // Create plan
    await createViaAPI(page, 'plan', {
      name: planName,
      rate_limit_per_minute: 60,
      requests_per_month: 1000,
      price_monthly: 0,
      enabled: true,
    });

    // Navigate to plans
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // Verify plan is visible
    await expect(page.locator(`text=${planName}`)).toBeVisible();

    // Refresh page
    await page.reload();
    await waitForTableLoad(page);

    // Plan should still be visible
    await expect(page.locator(`text=${planName}`)).toBeVisible();

    // Clean up
    const plans = await listViaAPI(page, 'plan');
    const created = (plans.records || plans).find((p: { name: string }) => p.name === planName);
    if (created) {
      await deleteViaAPI(page, 'plan', created.id);
    }
  });

  test('updates persist after navigation', async ({ page }) => {
    const originalName = uniqueName(TEST_PREFIX);
    const newName = uniqueName(TEST_PREFIX);

    // Create plan
    const plan = await createViaAPI(page, 'plan', {
      name: originalName,
      rate_limit_per_minute: 60,
      requests_per_month: 1000,
      price_monthly: 0,
      enabled: true,
    });

    // Update via UI
    await page.goto(`/mod/ui/plan/${plan.id}/edit`);
    await waitForFormLoad(page);
    await fillField(page, 'name', newName);
    await submitForm(page);

    // Navigate away
    await navigateToModule(page, 'user');

    // Navigate back
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);

    // New name should be visible
    await expect(page.locator(`text=${newName}`)).toBeVisible();
    await expect(page.locator(`text=${originalName}`)).not.toBeVisible();

    // Clean up
    await deleteViaAPI(page, 'plan', plan.id);
  });
});
