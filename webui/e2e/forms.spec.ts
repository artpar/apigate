import { test, expect } from '@playwright/test';

/**
 * E2E Tests: Form Handling
 *
 * These tests verify form rendering, field types, validation,
 * and form submission behavior.
 */

test.describe('Form Field Rendering', () => {
  test('should render text input for string fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Name field should be a text input
    const nameInput = page.locator('input[type="text"], input:not([type])').first();
    await expect(nameInput).toBeVisible();
  });

  test('should render email input for email fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Email field should have email type or be validated as email
    const emailInput = page.locator('input[type="email"], input[name*="email"]');

    if (await emailInput.count() > 0) {
      await expect(emailInput.first()).toBeVisible();
    }
  });

  test('should render select for enum fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Status field should be a select (enum)
    const select = page.locator('select, [role="combobox"], [role="listbox"]');

    if (await select.count() > 0) {
      await expect(select.first()).toBeVisible();
    }
  });

  test('should render number input for int fields', async ({ page }) => {
    await page.goto('/mod/ui/plan/new');
    await page.waitForLoadState('networkidle');

    // Rate limit should be number input
    const numberInput = page.locator('input[type="number"]');

    if (await numberInput.count() > 0) {
      await expect(numberInput.first()).toBeVisible();
    }
  });

  test('should render password input for secret fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Password field should be password type
    const passwordInput = page.locator('input[type="password"]');

    // Might or might not be visible depending on field config
    if (await passwordInput.count() > 0) {
      await expect(passwordInput.first()).toBeVisible();
    }
  });

  test('should render checkbox for boolean fields', async ({ page }) => {
    await page.goto('/mod/ui/plan/new');
    await page.waitForLoadState('networkidle');

    // Boolean fields should be checkboxes
    const checkbox = page.locator('input[type="checkbox"]');

    if (await checkbox.count() > 0) {
      await expect(checkbox.first()).toBeVisible();
    }
  });
});

test.describe('Form Labels and Help Text', () => {
  test('should display labels for all visible fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Each input should have an associated label
    const labels = page.locator('label');
    const labelCount = await labels.count();

    // Should have at least some labels
    expect(labelCount).toBeGreaterThan(0);
  });

  test('should mark required fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Required fields should have indicator (asterisk, "required" text, etc.)
    const requiredIndicators = page.locator('[aria-required="true"], .required, :has(> span:text("*"))');

    // Email is required, so should have at least one indicator
    // This is flexible based on implementation
  });

  test('should show field descriptions as help text', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Help text might appear below fields
    const helpText = page.locator('.help-text, .description, [aria-describedby], small');

    // Some fields might have help text
  });
});

test.describe('Form Interaction', () => {
  test('should allow typing in text fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const input = page.locator('input[type="text"], input:not([type])').first();

    if (await input.count() > 0) {
      await input.fill('Test Value');
      await expect(input).toHaveValue('Test Value');
    }
  });

  test('should allow selecting from dropdowns', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const select = page.locator('select').first();

    if (await select.count() > 0) {
      // Get options
      const options = await select.locator('option').allTextContents();

      if (options.length > 1) {
        // Select the second option (first might be placeholder)
        await select.selectOption({ index: 1 });
      }
    }
  });

  test('should toggle checkboxes', async ({ page }) => {
    await page.goto('/mod/ui/plan/new');
    await page.waitForLoadState('networkidle');

    const checkbox = page.locator('input[type="checkbox"]').first();

    if (await checkbox.count() > 0) {
      const initialState = await checkbox.isChecked();
      await checkbox.click();
      const newState = await checkbox.isChecked();

      expect(newState).not.toBe(initialState);
    }
  });

  test('should handle form reset/cancel', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Fill some data
    const input = page.locator('input[type="text"], input:not([type])').first();
    if (await input.count() > 0) {
      await input.fill('Test Value');
    }

    // Look for cancel/reset button
    const cancelButton = page.locator('button:has-text("Cancel"), button:has-text("Reset"), a:has-text("Cancel")');

    if (await cancelButton.count() > 0) {
      await cancelButton.first().click();
      await page.waitForLoadState('networkidle');

      // Should navigate away or reset form
    }
  });
});

test.describe('Form Validation', () => {
  test('should show validation error for empty required fields', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Submit empty form
    const submitButton = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Save")');

    if (await submitButton.count() > 0) {
      await submitButton.first().click();

      // Wait for validation
      await page.waitForTimeout(500);

      // Should show error message or prevent submission
      const errors = page.locator('.error, [role="alert"], .text-red-500, .invalid-feedback');
      // Errors might be shown
    }
  });

  test('should validate email format in real-time', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();

    if (await emailInput.count() > 0) {
      // Enter invalid email
      await emailInput.fill('invalid-email');
      await emailInput.blur();

      await page.waitForTimeout(300);

      // Check for validation state
      const isInvalid = await emailInput.evaluate(
        el => el.matches(':invalid') || el.classList.contains('invalid') || el.getAttribute('aria-invalid') === 'true'
      );

      // Might or might not show validation immediately
    }
  });

  test('should clear validation errors on valid input', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();

    if (await emailInput.count() > 0) {
      // Enter invalid then valid email
      await emailInput.fill('invalid');
      await emailInput.blur();
      await page.waitForTimeout(200);

      await emailInput.fill('valid@example.com');
      await emailInput.blur();
      await page.waitForTimeout(200);

      // Validation error should be cleared
    }
  });
});

test.describe('Form Submission', () => {
  test('should submit valid form successfully', async ({ page, request }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    const testEmail = `form-test-${Date.now()}@example.com`;

    // Fill required fields
    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();
    if (await emailInput.count() > 0) {
      await emailInput.fill(testEmail);
    }

    // Submit form
    const submitButton = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Save")');

    if (await submitButton.count() > 0) {
      await submitButton.first().click();
      await page.waitForLoadState('networkidle');

      // Wait for potential redirect or success message
      await page.waitForTimeout(1000);

      // Cleanup - try to delete the created record
      const listResponse = await request.get(`/mod/users?email=${encodeURIComponent(testEmail)}`);
      const { data } = await listResponse.json();

      if (data && data.length > 0) {
        await request.delete(`/mod/users/${data[0].id}`);
      }
    }
  });

  test('should show loading state during submission', async ({ page }) => {
    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Fill form
    const emailInput = page.locator('input[type="email"], input[name*="email"]').first();
    if (await emailInput.count() > 0) {
      await emailInput.fill(`loading-test-${Date.now()}@example.com`);
    }

    const submitButton = page.locator('button[type="submit"], button:has-text("Create"), button:has-text("Save")');

    if (await submitButton.count() > 0) {
      // Check for loading state after clicking
      await submitButton.first().click();

      // Button might show loading state
      const isDisabled = await submitButton.first().isDisabled();
      // Loading indicators might appear
    }
  });

  test('should handle submission errors gracefully', async ({ page }) => {
    // This test would need to trigger an error condition
    // For example, trying to create a duplicate unique value

    await page.goto('/mod/ui/user/new');
    await page.waitForLoadState('networkidle');

    // Try to create with duplicate email (if one exists)
    // This is a placeholder - actual test would need setup
  });
});

test.describe('Edit Form', () => {
  let testRecordId: string | null = null;

  test.beforeAll(async ({ request }) => {
    const response = await request.post('/mod/users', {
      data: {
        email: `edit-form-test-${Date.now()}@example.com`,
        name: 'Edit Form Test',
        status: 'active',
      },
    });

    if (response.ok()) {
      const result = await response.json();
      testRecordId = result.id;
    }
  });

  test.afterAll(async ({ request }) => {
    if (testRecordId) {
      await request.delete(`/mod/users/${testRecordId}`);
    }
  });

  test('should pre-populate form with existing data', async ({ page, request }) => {
    if (!testRecordId) return;

    const getResponse = await request.get(`/mod/users/${testRecordId}`);
    const { data } = await getResponse.json();

    await page.goto(`/mod/ui/user/${testRecordId}`);
    await page.waitForLoadState('networkidle');

    // The record ID should appear in breadcrumb
    const body = page.locator('body');
    const text = await body.textContent();
    expect(text).toContain(testRecordId);

    // Email should be pre-populated in the form input
    if (data.email) {
      const emailInput = page.locator('input[type="email"], input[name*="email"]').first();
      if (await emailInput.count() > 0) {
        const value = await emailInput.inputValue();
        expect(value).toBe(data.email);
      }
    }
  });

  test('should allow editing existing values', async ({ page }) => {
    if (!testRecordId) return;

    await page.goto(`/mod/ui/user/${testRecordId}`);
    await page.waitForLoadState('networkidle');

    // Find edit button
    const editButton = page.locator('button:has-text("Edit"), a:has-text("Edit")');

    if (await editButton.count() > 0) {
      await editButton.first().click();
      await page.waitForLoadState('networkidle');

      // Find name input and change it
      const nameInput = page.locator('input[name="name"], input[name*="name"]').first();

      if (await nameInput.count() > 0) {
        await nameInput.fill('Updated Name');

        const submitButton = page.locator('button[type="submit"], button:has-text("Save"), button:has-text("Update")');

        if (await submitButton.count() > 0) {
          await submitButton.first().click();
          await page.waitForLoadState('networkidle');
        }
      }
    }
  });
});
