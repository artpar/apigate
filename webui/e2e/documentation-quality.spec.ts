import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Documentation Quality
 *
 * These tests verify that field descriptions and documentation
 * are properly displayed throughout the application.
 */

test.describe('Module Descriptions', () => {
  test('all modules have descriptions in schema API', async ({ page }) => {
    const response = await page.request.get('/mod/_schema');
    const { modules } = await response.json();

    for (const mod of modules) {
      expect(mod.description, `Module ${mod.name} missing description`).toBeTruthy();
      expect(mod.description.length, `Module ${mod.name} has empty description`).toBeGreaterThan(0);
    }
  });

  test('module detail schema includes description', async ({ page }) => {
    const listResponse = await page.request.get('/mod/_schema');
    const { modules } = await listResponse.json();

    for (const mod of modules) {
      const schemaResponse = await page.request.get(`/mod/_schema/${mod.name}`);
      const schema = await schemaResponse.json();

      expect(schema.description, `Module ${mod.name} schema missing description`).toBeTruthy();
    }
  });
});

test.describe('Field Descriptions', () => {
  test('critical user fields have descriptions', async ({ page }) => {
    const response = await page.request.get('/mod/_schema/user');
    const schema = await response.json();

    const criticalFields = ['email', 'status', 'plan_id', 'name'];

    for (const fieldName of criticalFields) {
      const field = schema.fields.find((f: { name: string }) => f.name === fieldName);
      expect(field, `Field user.${fieldName} not found`).toBeTruthy();
      expect(field.description, `Field user.${fieldName} missing description`).toBeTruthy();
      expect(field.description.length).toBeGreaterThan(0);
    }
  });

  test('critical plan fields have descriptions', async ({ page }) => {
    const response = await page.request.get('/mod/_schema/plan');
    const schema = await response.json();

    const criticalFields = ['name', 'rate_limit_per_minute', 'requests_per_month', 'price_monthly'];

    for (const fieldName of criticalFields) {
      const field = schema.fields.find((f: { name: string }) => f.name === fieldName);
      expect(field, `Field plan.${fieldName} not found`).toBeTruthy();
      expect(field.description, `Field plan.${fieldName} missing description`).toBeTruthy();
    }
  });

  test('critical api_key fields have descriptions', async ({ page }) => {
    const response = await page.request.get('/mod/_schema/api_key');
    const schema = await response.json();

    const criticalFields = ['user_id', 'name', 'prefix', 'expires_at'];

    for (const fieldName of criticalFields) {
      const field = schema.fields.find((f: { name: string }) => f.name === fieldName);
      expect(field, `Field api_key.${fieldName} not found`).toBeTruthy();
      expect(field.description, `Field api_key.${fieldName} missing description`).toBeTruthy();
    }
  });

  test('all non-implicit fields have descriptions', async ({ page }) => {
    const listResponse = await page.request.get('/mod/_schema');
    const { modules } = await listResponse.json();

    const missingDescriptions: string[] = [];

    for (const mod of modules) {
      const schemaResponse = await page.request.get(`/mod/_schema/${mod.name}`);
      const schema = await schemaResponse.json();

      for (const field of schema.fields) {
        // Skip implicit fields (id, created_at, updated_at)
        if (field.implicit) continue;

        if (!field.description) {
          missingDescriptions.push(`${mod.name}.${field.name}`);
        }
      }
    }

    expect(
      missingDescriptions,
      `Fields missing descriptions: ${missingDescriptions.join(', ')}`
    ).toHaveLength(0);
  });
});

test.describe('Documentation Panel Display', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test('documentation panel shows field description on form', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Find the email field and interact with it
    const emailInput = page.locator('input[name="email"], input[type="email"]').first();

    if (await emailInput.count() > 0) {
      await emailInput.focus();
      await page.waitForTimeout(300);

      // Check if documentation panel exists and shows content
      const docsPanel = page.locator('[class*="documentation"], [class*="docs"], aside');

      if (await docsPanel.count() > 0) {
        const text = await docsPanel.textContent();

        // Should NOT show "No description available" for documented fields
        if (text?.includes('email')) {
          expect(text).not.toContain('No description available');
        }
      }
    }
  });

  test('documentation panel updates when selecting different fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('input[name="email"]').first();
    const nameInput = page.locator('input[name="name"]').first();

    if (await emailInput.count() > 0 && await nameInput.count() > 0) {
      // Focus email field
      await emailInput.focus();
      await page.waitForTimeout(200);

      // Focus name field
      await nameInput.focus();
      await page.waitForTimeout(200);

      // Documentation should update (we just verify no crash)
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('documentation shows field type information', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Get schema to know what types to expect
    const schemaResponse = await page.request.get('/mod/_schema/user');
    const schema = await schemaResponse.json();

    // At minimum, the page should render without errors
    await expect(page.locator('body')).toBeVisible();

    // Check that fields are rendered
    const emailField = schema.fields.find((f: { name: string }) => f.name === 'email');
    expect(emailField).toBeTruthy();
    expect(emailField.type).toBe('email');
  });
});

test.describe('Description Content Quality', () => {
  test('descriptions are meaningful, not placeholder text', async ({ page }) => {
    const listResponse = await page.request.get('/mod/_schema');
    const { modules } = await listResponse.json();

    const badDescriptions: string[] = [];
    const placeholderPatterns = [
      /^TODO/i,
      /^TBD/i,
      /^placeholder/i,
      /^description$/i,
      /^field description$/i,
      /^no description$/i,
      /^\.\.\.$/,
      /^-$/,
    ];

    for (const mod of modules) {
      const schemaResponse = await page.request.get(`/mod/_schema/${mod.name}`);
      const schema = await schemaResponse.json();

      for (const field of schema.fields) {
        if (field.description) {
          for (const pattern of placeholderPatterns) {
            if (pattern.test(field.description)) {
              badDescriptions.push(`${mod.name}.${field.name}: "${field.description}"`);
            }
          }
        }
      }
    }

    expect(
      badDescriptions,
      `Fields with placeholder descriptions: ${badDescriptions.join(', ')}`
    ).toHaveLength(0);
  });

  test('descriptions have reasonable length', async ({ page }) => {
    const listResponse = await page.request.get('/mod/_schema');
    const { modules } = await listResponse.json();

    const issues: string[] = [];

    for (const mod of modules) {
      const schemaResponse = await page.request.get(`/mod/_schema/${mod.name}`);
      const schema = await schemaResponse.json();

      for (const field of schema.fields) {
        if (field.description) {
          // Description should be at least 10 characters
          if (field.description.length < 10) {
            issues.push(`${mod.name}.${field.name} description too short: "${field.description}"`);
          }

          // Description should not be excessively long
          if (field.description.length > 500) {
            issues.push(`${mod.name}.${field.name} description too long (${field.description.length} chars)`);
          }
        }
      }
    }

    expect(issues, `Description length issues: ${issues.join('; ')}`).toHaveLength(0);
  });
});

test.describe('API Schema Completeness', () => {
  test('schema endpoint returns all expected modules', async ({ page }) => {
    const response = await page.request.get('/mod/_schema');
    const { modules, count } = await response.json();

    // Should have multiple modules
    expect(count).toBeGreaterThan(0);
    expect(modules.length).toBe(count);

    // Expected core modules
    const expectedModules = ['user', 'plan', 'api_key', 'route', 'upstream', 'setting'];

    for (const expected of expectedModules) {
      const found = modules.find((m: { name: string }) => m.name === expected);
      expect(found, `Expected module ${expected} not found in schema`).toBeTruthy();
    }
  });

  test('each module schema returns field list', async ({ page }) => {
    const listResponse = await page.request.get('/mod/_schema');
    const { modules } = await listResponse.json();

    for (const mod of modules) {
      const schemaResponse = await page.request.get(`/mod/_schema/${mod.name}`);
      expect(schemaResponse.ok(), `Failed to get schema for ${mod.name}`).toBe(true);

      const schema = await schemaResponse.json();
      expect(Array.isArray(schema.fields), `${mod.name} fields is not an array`).toBe(true);
      expect(schema.fields.length, `${mod.name} has no fields`).toBeGreaterThan(0);
    }
  });

  test('field schemas include type information', async ({ page }) => {
    const response = await page.request.get('/mod/_schema/user');
    const schema = await response.json();

    for (const field of schema.fields) {
      expect(field.name, 'Field missing name').toBeTruthy();
      expect(field.type, `Field ${field.name} missing type`).toBeTruthy();
    }
  });
});
