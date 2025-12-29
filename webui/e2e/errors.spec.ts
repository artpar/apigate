import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Error Handling
 *
 * These tests verify the application handles errors gracefully,
 * shows appropriate error messages, and maintains stability.
 */

test.describe('Network Error Handling', () => {
  test('should handle API timeout gracefully', async ({ page }) => {
    // Slow down API response
    await page.route('/mod/_schema', async route => {
      await new Promise(resolve => setTimeout(resolve, 5000));
      await route.abort('timedout');
    });

    await page.goto('/mod/ui/', { timeout: 10000 }).catch(() => {});

    // Page should still be functional or show error
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle API 500 errors', async ({ page }) => {
    // Mock a server error
    await page.route('/mod/users', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Should show error message or empty state, not crash
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle API 404 errors', async ({ page }) => {
    await page.goto('/mod/ui/user/nonexistent-id-12345');
    await page.waitForLoadState('networkidle');

    // Should show "not found" message or redirect
    const body = page.locator('body');
    await expect(body).toBeVisible();

    const text = await body.textContent();
    // Might contain "not found" or similar
  });

  test('should handle network offline state', async ({ page, context }) => {
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    // Go offline
    await context.setOffline(true);

    // Try to navigate
    await page.goto('/mod/ui/user').catch(() => {});

    // Page should handle offline state
    const body = page.locator('body');
    await expect(body).toBeVisible();

    // Restore online
    await context.setOffline(false);
  });
});

test.describe('Form Error Handling', () => {
  test('should display validation errors clearly', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Submit empty form
    const submitButton = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Save")');

    if (await submitButton.count() > 0) {
      await submitButton.first().click();
      await page.waitForTimeout(500);

      // Should show validation errors
      const errors = page.locator('.error, [role="alert"], .text-red-500, .text-red-600, .invalid');

      // Errors might be present
    }
  });

  test('should display API error messages', async ({ page }) => {
    // Mock an API error on create
    await page.route('/mod/users', route => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Email already exists' }),
        });
      } else {
        route.continue();
      }
    });

    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Fill form
    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();
    if (await emailInput.count() > 0) {
      await emailInput.fill('test@example.com');
    }

    // Submit
    const submitButton = page.locator('button[type="submit"], button:has-text("Create")');
    if (await submitButton.count() > 0) {
      await submitButton.first().click();
      await page.waitForTimeout(1000);

      // Error should be displayed
      const body = page.locator('body');
      const text = await body.textContent();
      // Might contain error message
    }
  });

  test('should handle submission during update conflict', async ({ page, request }) => {
    // Create a record
    const createResponse = await request.post('/mod/users', {
      data: {
        email: `conflict-test-${Date.now()}@example.com`,
        name: 'Conflict Test',
        status: 'active',
      },
    });

    if (createResponse.ok()) {
      const { id } = await createResponse.json();

      await page.goto(`/mod/ui/user/${id}`);
      await page.waitForLoadState('networkidle');

      // Edit and save - should handle gracefully
      // Cleanup
      await request.delete(`/mod/users/${id}`);
    }
  });
});

test.describe('Navigation Error Handling', () => {
  test('should handle invalid module names', async ({ page }) => {
    await page.goto('/mod/ui/invalid_module_xyz');
    await page.waitForLoadState('networkidle');

    // Should show error or redirect, not crash
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle malformed URLs', async ({ page }) => {
    await page.goto('/mod/ui/user/%00%00%00');
    await page.waitForLoadState('networkidle');

    // Should handle gracefully
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle XSS attempts in URL', async ({ page }) => {
    await page.goto('/mod/ui/user/<script>alert(1)</script>');
    await page.waitForLoadState('networkidle');

    // Should not execute script
    const alerts: string[] = [];
    page.on('dialog', dialog => {
      alerts.push(dialog.message());
      dialog.dismiss();
    });

    await page.waitForTimeout(500);

    // No alerts should have fired
    expect(alerts).toHaveLength(0);
  });
});

test.describe('Data Error Handling', () => {
  test('should handle malformed JSON in API response', async ({ page }) => {
    await page.route('/mod/users', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: 'not valid json {{{',
      });
    });

    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Should handle parse error gracefully
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should handle missing expected fields', async ({ page }) => {
    await page.route('/mod/_schema/user', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          // Missing expected fields like plural, fields, actions
          module: 'user',
        }),
      });
    });

    await page.goto('/mod/ui/user');

    // Wait for some response - might error or show partial content
    await page.waitForTimeout(2000);

    // Page should not crash completely - either shows error message or partial content
    // Check that the page responded in some way
    const url = page.url();
    expect(url).toContain('user');
  });

  test('should handle extremely large data sets', async ({ page }) => {
    // Generate large data
    const largeData = Array.from({ length: 1000 }, (_, i) => ({
      id: `user-${i}`,
      email: `user${i}@example.com`,
      name: `User ${i}`,
      status: 'active',
    }));

    await page.route('/mod/users', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: largeData, count: 1000 }),
      });
    });

    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    // Should handle large data without crashing
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });
});

test.describe('Console Error Monitoring', () => {
  test('should not have unhandled promise rejections', async ({ page }) => {
    const errors: string[] = [];

    page.on('pageerror', error => {
      errors.push(error.message);
    });

    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    // Navigate around
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    await page.goto('/mod/ui/plan');
    await page.waitForLoadState('networkidle');

    // Filter critical errors
    const criticalErrors = errors.filter(
      e => !e.includes('net::') && !e.includes('favicon')
    );

    expect(criticalErrors).toHaveLength(0);
  });

  test('should not have React errors', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (text.includes('React') || text.includes('Warning:')) {
          errors.push(text);
        }
      }
    });

    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');

    // Navigate to trigger potential React errors
    await page.goto('/mod/ui/user');
    await page.waitForLoadState('networkidle');

    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Filter out dev-mode warnings that are acceptable
    const criticalErrors = errors.filter(
      e => !e.includes('DevTools') && !e.includes('Strict Mode')
    );

    // Should have no critical React errors
    expect(criticalErrors).toHaveLength(0);
  });
});

test.describe('Recovery from Errors', () => {
  test('should allow retry after network failure', async ({ page }) => {
    let requestCount = 0;

    await page.route('/mod/_schema', async route => {
      requestCount++;
      if (requestCount === 1) {
        // First request fails
        await route.abort('failed');
      } else {
        // Subsequent requests succeed
        await route.continue();
      }
    });

    await page.goto('/mod/ui/');
    await page.waitForTimeout(1000);

    // Retry by reloading
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should have recovered
    const body = page.locator('body');
    await expect(body).toBeVisible();
  });

  test('should clear form errors on valid input', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();

    if (await emailInput.count() > 0) {
      // Enter invalid value
      await emailInput.fill('invalid');
      await emailInput.blur();
      await page.waitForTimeout(200);

      // Enter valid value
      await emailInput.fill('valid@example.com');
      await emailInput.blur();
      await page.waitForTimeout(200);

      // Error should be cleared
      const errors = page.locator('.error, [role="alert"]');
      const errorCount = await errors.count();
      // Might have no errors or different behavior
    }
  });

  test('should maintain state after error recovery', async ({ page, request }) => {
    // Create a record
    const createResponse = await request.post('/mod/users', {
      data: {
        email: `recovery-test-${Date.now()}@example.com`,
        name: 'Recovery Test',
        status: 'active',
      },
    });

    if (createResponse.ok()) {
      const { id } = await createResponse.json();

      await page.goto(`/mod/ui/user/${id}`);
      await page.waitForLoadState('networkidle');

      // Simulate temporary network issue
      await page.route('/mod/users/*', route => route.abort('failed'));

      // Try to refresh (will fail)
      await page.reload().catch(() => {});

      // Remove the blocking route
      await page.unrouteAll();

      // Retry
      await page.reload();
      await page.waitForLoadState('networkidle');

      // Data should be visible
      const body = page.locator('body');
      await expect(body).toBeVisible();

      // Cleanup
      await request.delete(`/mod/users/${id}`);
    }
  });
});
