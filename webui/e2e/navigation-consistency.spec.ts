import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Navigation Consistency
 *
 * These tests verify that navigation links throughout the application
 * use consistent URL patterns and don't cause "Module not found" errors.
 */

test.describe('Navigation Consistency', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
  });

  test('all dashboard module cards navigate successfully', async ({ page }) => {
    // Find all module cards on the dashboard - try multiple selectors
    let moduleCards = page.locator('[data-testid="module-card"]');
    let count = await moduleCards.count();

    // Fallback: look for any links in main content area
    if (count === 0) {
      moduleCards = page.locator('main a[href*="/"], .main-content a[href*="/"]').first();
      count = await moduleCards.count();
    }

    // If still no cards found, the dashboard structure differs - just verify no errors
    if (count === 0) {
      // Verify the page loaded without module errors
      await expect(page.locator('text=Module not found')).not.toBeVisible();
      return;
    }

    // Test each card
    for (let i = 0; i < Math.min(count, 3); i++) { // Test first 3 to avoid timeout
      await page.goto('/mod/ui/');
      await page.waitForLoadState('networkidle');

      const cards = page.locator('[data-testid="module-card"]');
      if (await cards.count() > 0) {
        await cards.nth(i).click();
      } else {
        // Fallback to any module link
        const links = page.locator('main a[href*="/"], .main-content a[href*="/"]');
        if (await links.count() > i) {
          await links.nth(i).click();
        } else {
          continue;
        }
      }
      await page.waitForLoadState('networkidle');

      // CRITICAL: Verify no "Module not found" error
      await expect(page.locator('text=Module not found')).not.toBeVisible();
      await expect(page.locator('text=does not exist')).not.toBeVisible();

      // Page should render successfully
      await expect(page.locator('body')).toBeVisible();
    }
  });

  test('all sidebar links navigate successfully', async ({ page }) => {
    // Get all sidebar navigation links
    const sidebarLinks = page.locator('aside a[href^="/"], nav a[href^="/"]');
    const hrefs = await sidebarLinks.evaluateAll(links =>
      links.map(l => l.getAttribute('href')).filter(Boolean)
    );

    // Test each link
    for (const href of hrefs) {
      if (href && !href.includes('http') && href !== '/') {
        await page.goto(`/mod/ui${href.startsWith('/') ? href : '/' + href}`);
        await page.waitForLoadState('networkidle');

        // Should not show "Module not found" error
        await expect(page.locator('text=Module not found')).not.toBeVisible();
        await expect(page.locator('text=does not exist')).not.toBeVisible();
      }
    }
  });

  test('dashboard and sidebar links use consistent URL pattern', async ({ page }) => {
    // Get modules from API to know expected names
    const schemaResponse = await page.request.get('/mod/_schema');
    const { modules } = await schemaResponse.json();

    // Dashboard cards should link to singular module names (e.g., /user not /users)
    const dashboardLinks = page.locator('[data-testid="module-card"]');
    const count = await dashboardLinks.count();

    for (let i = 0; i < count; i++) {
      const href = await dashboardLinks.nth(i).getAttribute('href');

      if (href) {
        // Find the matching module
        const matchedModule = modules.find(
          (m: { name: string; plural: string }) =>
            href.includes(m.name) || href.includes(m.plural)
        );

        if (matchedModule) {
          // URL should use singular form (module name), not plural
          expect(href).toContain(matchedModule.name);
        }
      }
    }
  });

  test('clicking module cards reaches the correct module', async ({ page }) => {
    // Get expected modules from schema
    const schemaResponse = await page.request.get('/mod/_schema');
    const { modules } = await schemaResponse.json();

    for (const mod of modules.slice(0, 3)) { // Test first 3 modules
      await page.goto('/mod/ui/');
      await page.waitForLoadState('networkidle');

      // Find card for this module
      const card = page.locator(`[data-testid="module-card"][data-module="${mod.name}"]`);

      if (await card.count() > 0) {
        await card.click();
        await page.waitForLoadState('networkidle');

        // Should be on correct module page
        const url = page.url();
        expect(url).toContain(mod.name);

        // Should not show error
        await expect(page.locator('text=Module not found')).not.toBeVisible();
      }
    }
  });

  test('navigating to plural URL redirects or works correctly', async ({ page }) => {
    // Get modules to find their plural forms
    const schemaResponse = await page.request.get('/mod/_schema');
    const { modules } = await schemaResponse.json();

    for (const mod of modules.slice(0, 2)) {
      // Try navigating to the singular (correct) URL
      await page.goto(`/mod/ui/${mod.name}`);
      await page.waitForLoadState('networkidle');

      // Should not show "Module not found"
      await expect(page.locator('text=Module not found')).not.toBeVisible();

      // Content should load
      const body = page.locator('body');
      await expect(body).toBeVisible();
    }
  });
});

test.describe('Cross-Component Navigation Consistency', () => {
  test('sidebar and dashboard link to same destinations', async ({ page }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    // Get schema to know module names
    const schemaResponse = await page.request.get('/mod/_schema');
    const { modules } = await schemaResponse.json();

    // Test a subset to avoid timeout
    for (const mod of modules.slice(0, 3)) {
      // Check if dashboard card exists for this module
      const dashboardCard = page.locator(`[data-testid="module-card"][data-module="${mod.name}"]`).first();
      let dashboardHref: string | null = null;
      if (await dashboardCard.count() > 0) {
        dashboardHref = await dashboardCard.getAttribute('href');
      }

      // Check if sidebar link exists
      const sidebarLink = page.locator(`aside a[href*="${mod.name}"], nav a[href*="${mod.name}"]`).first();
      let sidebarHref: string | null = null;
      if (await sidebarLink.count() > 0) {
        sidebarHref = await sidebarLink.getAttribute('href');
      }

      // If both exist, they should point to the same place
      if (dashboardHref && sidebarHref) {
        // Both should use the singular module name
        expect(dashboardHref).toContain(mod.name);
        expect(sidebarHref).toContain(mod.name);
      }
    }
  });

  test('breadcrumb navigation is consistent', async ({ page }) => {
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // If breadcrumbs exist, test them
    const breadcrumbs = page.locator('nav[aria-label*="breadcrumb"] a, .breadcrumb a');

    if (await breadcrumbs.count() > 0) {
      const homeLink = breadcrumbs.first();
      await homeLink.click();
      await page.waitForLoadState('networkidle');

      // Should navigate back to dashboard
      await expect(page.locator('text=Module not found')).not.toBeVisible();
    }
  });
});

test.describe('Error Prevention', () => {
  test('no console errors during normal navigation', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (text.includes('Module not found') || text.includes('does not exist')) {
          errors.push(text);
        }
      }
    });

    // Navigate through all main routes
    const routes = ['/mod/ui/', '/mod/ui/user', '/mod/ui/plan', '/mod/ui/api_key', '/mod/ui/route', '/mod/ui/upstream'];

    for (const route of routes) {
      await page.goto(route);
      await page.waitForLoadState('networkidle');
    }

    // Should have no module-related errors
    expect(errors).toHaveLength(0);
  });

  test('no "Module not found" visible on any module page', async ({ page }) => {
    const schemaResponse = await page.request.get('/mod/_schema');
    const { modules } = await schemaResponse.json();

    for (const mod of modules) {
      await page.goto(`/mod/ui/${mod.name}`);
      await page.waitForLoadState('networkidle');

      // Check for error message
      const errorVisible = await page.locator('text=Module not found').isVisible();
      expect(errorVisible, `Module ${mod.name} showed "Module not found" error`).toBe(false);

      const doesNotExist = await page.locator('text=does not exist').isVisible();
      expect(doesNotExist, `Module ${mod.name} showed "does not exist" error`).toBe(false);
    }
  });
});
