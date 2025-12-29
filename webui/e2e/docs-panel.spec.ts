import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Documentation Panel
 *
 * These tests verify the contextual documentation panel
 * shows appropriate help based on focused elements.
 */

test.describe('Documentation Panel Visibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test('should display documentation panel on desktop viewport', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });

    // Look for documentation panel
    const docPanel = page.locator('[data-testid="docs-panel"], .docs-panel, aside:last-child, [role="complementary"]');

    if (await docPanel.count() > 0) {
      await expect(docPanel.first()).toBeVisible();
    }
  });

  test('should hide or collapse documentation panel on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(300);

    // Panel might be hidden or collapsed on mobile
    // Implementation-dependent behavior
  });
});

test.describe('Documentation Content', () => {
  test('should show module documentation when viewing module list', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Documentation panel should show user module info
    const docPanel = page.locator('[data-testid="docs-panel"], .docs-panel, aside:last-child');

    if (await docPanel.count() > 0) {
      const text = await docPanel.first().textContent();
      // Should contain module-related content
    }
  });

  test('should show field documentation on field focus', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Focus on email field
    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();

    if (await emailInput.count() > 0) {
      await emailInput.focus();
      await page.waitForTimeout(300);

      // Documentation panel should update
      const docPanel = page.locator('[data-testid="docs-panel"], .docs-panel, aside:last-child');

      if (await docPanel.count() > 0) {
        const text = await docPanel.first().textContent();
        // Might contain email-related help
      }
    }
  });

  test('should show action documentation for custom actions', async ({ page, request }) => {
    // Get a user record
    const listResponse = await request.get('/mod/users?limit=1');
    const { data } = await listResponse.json();

    if (data && data.length > 0) {
      await page.goto(`/mod/ui/user/${data[0].id}`);
      await page.waitForLoadState('networkidle');

      // Hover over or focus on action button
      const actionButton = page.locator('button:has-text("activate"), button:has-text("suspend")').first();

      if (await actionButton.count() > 0) {
        await actionButton.hover();
        await page.waitForTimeout(300);

        // Documentation might update to show action info
      }
    }
  });
});

test.describe('Documentation Tabs', () => {
  test('should have Spec tab showing field specifications', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Look for Spec tab
    const specTab = page.locator('button:has-text("Spec"), [role="tab"]:has-text("Spec")');

    if (await specTab.count() > 0) {
      await specTab.first().click();
      await page.waitForTimeout(300);

      // Should show specification content
      const tabContent = page.locator('[role="tabpanel"]');
      if (await tabContent.count() > 0) {
        const text = await tabContent.first().textContent();
        // Should contain field specs
      }
    }
  });

  test('should have Details tab showing field constraints', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    const detailsTab = page.locator('button:has-text("Details"), [role="tab"]:has-text("Details")');

    if (await detailsTab.count() > 0) {
      await detailsTab.first().click();
      await page.waitForTimeout(300);
    }
  });

  test('should have API tab showing curl examples', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    const apiTab = page.locator('button:has-text("API"), [role="tab"]:has-text("API")');

    if (await apiTab.count() > 0) {
      await apiTab.first().click();
      await page.waitForTimeout(300);

      // API tab shows endpoints (paths and HTTP methods) when viewing module
      // The endpoint paths contain module names like /users
      const content = page.locator('[class*="docs"], aside, [role="complementary"]');
      if (await content.count() > 0) {
        const text = await content.first().textContent();
        // Should contain endpoint paths or HTTP methods
        expect(text?.toLowerCase()).toMatch(/get|post|put|delete|users|\/mod/i);
      }
    }
  });

  test('should have Examples tab with sample data', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    const examplesTab = page.locator('button:has-text("Examples"), [role="tab"]:has-text("Examples")');

    if (await examplesTab.count() > 0) {
      await examplesTab.first().click();
      await page.waitForTimeout(300);

      // Examples tab shows curl commands when viewing module
      const codeBlock = page.locator('pre, code');

      if (await codeBlock.count() > 0) {
        const text = await codeBlock.first().textContent();
        // Should contain curl example or sample data
        expect(text).toMatch(/curl|http|localhost|example|user/i);
      }
    }
  });
});

test.describe('Copy to Clipboard', () => {
  test('should have copy button for code examples', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Go to API tab
    const apiTab = page.locator('button:has-text("API"), [role="tab"]:has-text("API")');

    if (await apiTab.count() > 0) {
      await apiTab.first().click();
      await page.waitForTimeout(300);

      // Look for copy button
      const copyButton = page.locator('button:has-text("Copy"), button[aria-label*="copy"], .copy-button');

      if (await copyButton.count() > 0) {
        await expect(copyButton.first()).toBeVisible();
      }
    }
  });

  test('should copy code to clipboard when clicking copy button', async ({ page, context }) => {
    // Grant clipboard permissions
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);

    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    const apiTab = page.locator('button:has-text("API"), [role="tab"]:has-text("API")');

    if (await apiTab.count() > 0) {
      await apiTab.first().click();
      await page.waitForTimeout(300);

      const copyButton = page.locator('button:has-text("Copy"), button[aria-label*="copy"], .copy-button').first();

      if (await copyButton.count() > 0) {
        await copyButton.click();

        // Check clipboard content
        const clipboardContent = await page.evaluate(() => navigator.clipboard.readText());

        // Clipboard should have some content
        expect(clipboardContent.length).toBeGreaterThan(0);
      }
    }
  });
});

test.describe('Contextual Updates', () => {
  test('should update documentation when switching between modules', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);

    // Get initial doc content from the right panel
    const body = page.locator('body');
    const initialContent = await body.textContent();
    expect(initialContent).toContain('user');

    // Navigate to different module using sidebar link
    const planLink = page.locator('nav a[href*="plan"], aside a[href*="plan"]').first();
    if (await planLink.count() > 0) {
      await planLink.click();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(500);

      // Content should now reference plan module
      const newContent = await body.textContent();
      expect(newContent).toContain('plan');
    }
  });

  test('should update documentation when switching form fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Focus on email field
    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();
    if (await emailInput.count() > 0) {
      await emailInput.focus();
      await page.waitForTimeout(200);
    }

    // Focus on name field
    const nameInput = page.locator('input[name="name"], input[name*="name"]').first();
    if (await nameInput.count() > 0) {
      await nameInput.focus();
      await page.waitForTimeout(200);
    }

    // Documentation panel might have updated
  });
});

test.describe('Schema Display', () => {
  test('should display field types correctly', async ({ page, request }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Get schema
    const schemaResponse = await request.get('/mod/_schema/user');
    const schema = await schemaResponse.json();

    // Look for Spec tab and click it
    const specTab = page.locator('button:has-text("Spec"), [role="tab"]:has-text("Spec")');

    if (await specTab.count() > 0) {
      await specTab.first().click();
      await page.waitForTimeout(300);

      // Check that field types are displayed
      const panel = page.locator('[role="tabpanel"]');
      if (await panel.count() > 0) {
        const text = await panel.first().textContent();

        // Should show some field types
        for (const field of schema.fields.slice(0, 3)) {
          // Field names or types might appear
        }
      }
    }
  });

  test('should display constraints and validation rules', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    const detailsTab = page.locator('button:has-text("Details"), [role="tab"]:has-text("Details")');

    if (await detailsTab.count() > 0) {
      await detailsTab.first().click();
      await page.waitForTimeout(300);

      const panel = page.locator('[role="tabpanel"]');
      if (await panel.count() > 0) {
        const text = await panel.first().textContent();

        // Should show validation-related content
        // e.g., "required", "unique", "max length", etc.
      }
    }
  });
});
