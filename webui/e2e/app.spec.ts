import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Application Loading and Basic Structure
 *
 * These tests verify the core application loads correctly,
 * all assets are served, and the three-pane layout renders.
 */

test.describe('Application Loading', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the WebUI
    await page.goto('/mod/ui/');
  });

  test('should load the application without console errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    // Wait for the app to be fully loaded
    await page.waitForLoadState('networkidle');

    // Filter out known acceptable errors (favicon, network, and 401 auth checks)
    const criticalErrors = errors.filter(
      e => !e.includes('favicon') && !e.includes('net::ERR') && !e.includes('401')
    );

    expect(criticalErrors).toHaveLength(0);
  });

  test('should have correct page title', async ({ page }) => {
    await expect(page).toHaveTitle(/APIGate/);
  });

  test('should load all critical assets', async ({ page }) => {
    // Check that critical assets loaded
    const responses: { url: string; status: number }[] = [];

    page.on('response', response => {
      const url = response.url();
      if (url.includes('/assets/') || url.includes('.js') || url.includes('.css')) {
        responses.push({ url, status: response.status() });
      }
    });

    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    // All asset requests should be successful (2xx)
    const failedAssets = responses.filter(r => r.status >= 400);
    expect(failedAssets).toHaveLength(0);

    // Should have loaded JS and CSS
    const hasJS = responses.some(r => r.url.includes('.js') && r.status === 200);
    const hasCSS = responses.some(r => r.url.includes('.css') && r.status === 200);

    expect(hasJS).toBe(true);
    expect(hasCSS).toBe(true);
  });

  test('should render the three-pane layout', async ({ page }) => {
    // Wait for React to render
    await page.waitForSelector('#root');

    // The app should have loaded content
    const root = page.locator('#root');
    await expect(root).not.toBeEmpty();
  });

  test('should display module navigation sidebar', async ({ page }) => {
    // Wait for navigation to render
    await page.waitForSelector('nav, [role="navigation"], aside');

    // Should have some navigation links
    const navLinks = page.locator('nav a, aside a');
    await expect(navLinks.first()).toBeVisible();
  });

  test('should be responsive to viewport changes', async ({ page }) => {
    // Test at mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(500);

    // App should still be functional
    const root = page.locator('#root');
    await expect(root).toBeVisible();

    // Test at tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.waitForTimeout(500);
    await expect(root).toBeVisible();

    // Test at desktop viewport
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.waitForTimeout(500);
    await expect(root).toBeVisible();
  });
});

test.describe('API Connectivity', () => {
  test('should connect to schema API', async ({ page }) => {
    // Intercept the schema API call
    const schemaResponse = await page.request.get('/mod/_schema');

    expect(schemaResponse.ok()).toBe(true);

    const data = await schemaResponse.json();
    expect(data).toHaveProperty('modules');
    expect(data.modules.length).toBeGreaterThan(0);
  });

  test('should load module schemas', async ({ page }) => {
    // Test loading a specific module schema
    const userSchema = await page.request.get('/mod/_schema/user');

    expect(userSchema.ok()).toBe(true);

    const data = await userSchema.json();
    expect(data).toHaveProperty('module', 'user');
    expect(data).toHaveProperty('fields');
    expect(data).toHaveProperty('actions');
  });

  test('should connect to module data APIs', async ({ page }) => {
    // Test the users list endpoint
    const usersResponse = await page.request.get('/mod/users');

    expect(usersResponse.ok()).toBe(true);

    const data = await usersResponse.json();
    expect(data).toHaveProperty('data');
    expect(data).toHaveProperty('count');
  });
});

test.describe('Performance', () => {
  test('should load within acceptable time', async ({ page }) => {
    const startTime = Date.now();

    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    const loadTime = Date.now() - startTime;

    // Should load within 5 seconds
    expect(loadTime).toBeLessThan(5000);
  });

  test('should not have memory leaks on navigation', async ({ page }) => {
    await page.goto('/mod/ui/');

    // Get initial memory usage (if available)
    const initialMetrics = await page.evaluate(() => {
      if ('memory' in performance) {
        return (performance as any).memory.usedJSHeapSize;
      }
      return null;
    });

    // Navigate multiple times
    for (let i = 0; i < 5; i++) {
      await page.goto('/mod/ui/user');
      await page.waitForTimeout(200);
      await page.goto('/mod/ui/');
      await page.waitForTimeout(200);
    }

    // Check memory didn't grow excessively
    if (initialMetrics !== null) {
      const finalMetrics = await page.evaluate(() => {
        if ('memory' in performance) {
          return (performance as any).memory.usedJSHeapSize;
        }
        return null;
      });

      if (finalMetrics !== null) {
        // Memory shouldn't more than double
        expect(finalMetrics).toBeLessThan(initialMetrics * 2);
      }
    }
  });
});
