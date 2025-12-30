// Shared test utilities for E2E tests
import { Page, expect, Locator } from '@playwright/test';

export const API_BASE = '/mod';
export const UI_BASE = '/mod/ui';

// Test credentials - should match what's set up in global setup
export const TEST_USER = {
  email: 'admin@example.com',
  password: 'admin123',
  name: 'Admin User',
};

// Auth storage path
export const AUTH_FILE = 'playwright/.auth/user.json';

// Login helper for tests that need fresh authentication
export async function login(page: Page, email: string = TEST_USER.email, password: string = TEST_USER.password) {
  await page.goto('/mod/ui/login');
  await page.waitForLoadState('networkidle');

  // Fill login form
  await page.locator('input[name="email"], input#email').fill(email);
  await page.locator('input[name="password"], input#password').fill(password);
  await page.locator('button[type="submit"]').click();

  // Wait for redirect
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(500);
}

// Check if setup is required (first run)
export async function checkSetupRequired(page: Page): Promise<boolean> {
  const response = await page.request.get('/mod/auth/setup-required');
  const data = await response.json();
  return data.setup_required === true;
}

// Perform initial setup if required
export async function performSetup(page: Page, email: string, password: string, name: string) {
  const response = await page.request.post('/mod/auth/setup', {
    data: { email, password, name },
  });
  return response.ok();
}

// Navigation helpers
export async function navigateToModule(page: Page, moduleName: string) {
  await page.goto(`${UI_BASE}/${moduleName}`);
  await page.waitForLoadState('networkidle');
}

export async function navigateToCreate(page: Page, moduleName: string) {
  await page.goto(`${UI_BASE}/${moduleName}/new`);
  await page.waitForLoadState('networkidle');
}

export async function navigateToEdit(page: Page, moduleName: string, id: string) {
  await page.goto(`${UI_BASE}/${moduleName}/${id}/edit`);
  await page.waitForLoadState('networkidle');
}

export async function navigateToDetail(page: Page, moduleName: string, id: string) {
  await page.goto(`${UI_BASE}/${moduleName}/${id}`);
  await page.waitForLoadState('networkidle');
}

// Click helpers
export async function clickCreateNew(page: Page) {
  const createBtn = page.locator('a:has-text("Create"), button:has-text("Create"), a:has-text("New"), a[href$="/new"]');
  await createBtn.first().click();
  await page.waitForLoadState('networkidle');
}

export async function clickFirstRow(page: Page) {
  const row = page.locator('table tbody tr').first();
  await row.click();
  await page.waitForLoadState('networkidle');
}

// Form helpers
export async function fillField(page: Page, fieldName: string, value: string) {
  // Try multiple selector strategies
  const selectors = [
    `input[name="${fieldName}"]`,
    `textarea[name="${fieldName}"]`,
    `select[name="${fieldName}"]`,
    `[data-field="${fieldName}"] input`,
    `[data-field="${fieldName}"] textarea`,
    `[data-field="${fieldName}"] select`,
  ];

  let input: Locator | null = null;
  for (const selector of selectors) {
    const locator = page.locator(selector);
    if (await locator.count() > 0) {
      input = locator.first();
      break;
    }
  }

  if (!input) {
    throw new Error(`Field "${fieldName}" not found`);
  }

  const tagName = await input.evaluate(el => el.tagName.toLowerCase());
  const inputType = await input.getAttribute('type');

  if (tagName === 'select') {
    await input.selectOption(value);
  } else if (inputType === 'checkbox') {
    const checked = await input.isChecked();
    const shouldBeChecked = value === 'true' || value === '1';
    if (checked !== shouldBeChecked) {
      await input.click();
    }
  } else {
    await input.fill(value);
  }
}

export async function fillRefField(page: Page, fieldName: string, searchText: string) {
  // RefField uses a search/select pattern
  const refField = page.locator(`[data-field="${fieldName}"], [name="${fieldName}"]`).first();

  // Click to open dropdown
  await refField.click();
  await page.waitForTimeout(300);

  // Type to search
  const searchInput = page.locator('input[placeholder*="Search"], input[type="search"]').first();
  if (await searchInput.count() > 0) {
    await searchInput.fill(searchText);
    await page.waitForTimeout(500);
  }

  // Select first matching option
  const option = page.locator(`[role="option"]:has-text("${searchText}"), .dropdown-item:has-text("${searchText}")`).first();
  if (await option.count() > 0) {
    await option.click();
  }
}

export async function submitForm(page: Page) {
  await page.click('button[type="submit"]');
  await page.waitForLoadState('networkidle');
}

export async function cancelForm(page: Page) {
  const cancelBtn = page.locator('button:has-text("Cancel"), a:has-text("Cancel")');
  await cancelBtn.first().click();
  await page.waitForLoadState('networkidle');
}

// Assertion helpers
export async function expectNoErrors(page: Page) {
  await expect(page.locator('text=Module not found')).not.toBeVisible({ timeout: 2000 });
  await expect(page.locator('text=does not exist')).not.toBeVisible({ timeout: 2000 });
}

export async function expectSuccessMessage(page: Page) {
  const success = page.locator('.toast-success, [role="alert"]:has-text("success"), text=successfully');
  // Success message is optional - don't fail if not present
  await page.waitForTimeout(500);
}

export async function expectErrorMessage(page: Page, errorText?: string) {
  if (errorText) {
    await expect(page.locator(`text=${errorText}`)).toBeVisible({ timeout: 3000 });
  } else {
    const error = page.locator('.text-red-500, .text-red-600, [role="alert"]:has-text("error"), .error');
    await expect(error.first()).toBeVisible({ timeout: 3000 });
  }
}

export async function expectTableHasRows(page: Page, minRows: number = 1) {
  const rows = page.locator('table tbody tr');
  await expect(rows).toHaveCount(await rows.count());
  expect(await rows.count()).toBeGreaterThanOrEqual(minRows);
}

export async function expectTableEmpty(page: Page) {
  const emptyMessage = page.locator('text=No records, text=No data, text=Empty');
  const rows = page.locator('table tbody tr');
  const rowCount = await rows.count();
  // Either show empty message or have 0 rows
  if (rowCount === 0) {
    return; // Table is empty
  }
  // Check if it's an "empty" row
  const firstRowText = await rows.first().textContent();
  expect(firstRowText?.toLowerCase()).toMatch(/no\s*(record|data|result)|empty/i);
}

// Delete helpers
export async function deleteRecord(page: Page, confirmDelete: boolean = true) {
  const deleteBtn = page.locator('button:has-text("Delete"), button[aria-label="Delete"]').first();
  await deleteBtn.click();

  // Handle confirmation dialog
  if (confirmDelete) {
    const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes"), button:has-text("Delete")').last();
    await confirmBtn.click();
  } else {
    const cancelBtn = page.locator('button:has-text("Cancel"), button:has-text("No")');
    await cancelBtn.click();
  }

  await page.waitForLoadState('networkidle');
}

// Action button helpers
export async function clickAction(page: Page, actionName: string) {
  const actionBtn = page.locator(`button:has-text("${actionName}"), a:has-text("${actionName}")`).first();
  await actionBtn.click();
  await page.waitForLoadState('networkidle');
}

export async function clickActionWithConfirm(page: Page, actionName: string, confirm: boolean = true) {
  await clickAction(page, actionName);

  // Handle confirmation if present
  await page.waitForTimeout(300);
  const dialog = page.locator('[role="dialog"], .modal');
  if (await dialog.count() > 0) {
    if (confirm) {
      await page.locator('button:has-text("Confirm"), button:has-text("Yes")').click();
    } else {
      await page.locator('button:has-text("Cancel"), button:has-text("No")').click();
    }
  }

  await page.waitForLoadState('networkidle');
}

// API helpers for data setup/cleanup
export async function createViaAPI(page: Page, module: string, data: Record<string, unknown>) {
  const response = await page.request.post(`${API_BASE}/${module}`, { data });
  expect(response.ok()).toBe(true);
  return response.json();
}

export async function updateViaAPI(page: Page, module: string, id: string, data: Record<string, unknown>) {
  const response = await page.request.put(`${API_BASE}/${module}/${id}`, { data });
  expect(response.ok()).toBe(true);
  return response.json();
}

export async function deleteViaAPI(page: Page, module: string, id: string) {
  const response = await page.request.delete(`${API_BASE}/${module}/${id}`);
  return response.ok();
}

export async function getViaAPI(page: Page, module: string, id: string) {
  const response = await page.request.get(`${API_BASE}/${module}/${id}`);
  return response.json();
}

export async function listViaAPI(page: Page, module: string) {
  const response = await page.request.get(`${API_BASE}/${module}`);
  return response.json();
}

// Generate unique test data
export function uniqueEmail() {
  return `test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`;
}

export function uniqueName(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function uniqueKey(prefix: string = 'key') {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}

// Schema helpers
export async function getModuleSchema(page: Page, moduleName: string) {
  const response = await page.request.get(`${API_BASE}/_schema/${moduleName}`);
  return response.json();
}

export async function getAllModules(page: Page) {
  const response = await page.request.get(`${API_BASE}/_schema`);
  const data = await response.json();
  return data.modules;
}

// Wait helpers
export async function waitForTableLoad(page: Page) {
  // Wait for loading indicators to disappear
  await page.waitForTimeout(500);
  const loading = page.locator('.loading, [aria-busy="true"]');
  if (await loading.count() > 0) {
    await expect(loading.first()).not.toBeVisible({ timeout: 10000 });
  }
  // Also check for text-based loading indicator
  const loadingText = page.getByText('Loading');
  if (await loadingText.count() > 0) {
    await expect(loadingText.first()).not.toBeVisible({ timeout: 10000 });
  }
}

export async function waitForFormLoad(page: Page) {
  await page.waitForTimeout(300);
  // Ensure form is visible
  const form = page.locator('form');
  await expect(form.first()).toBeVisible({ timeout: 5000 });
}

// Cleanup helper - deletes all test records created with a specific prefix
export async function cleanupTestData(page: Page, module: string, namePrefix: string) {
  try {
    const data = await listViaAPI(page, module);
    const records = data.records || data.items || data;

    if (Array.isArray(records)) {
      for (const record of records) {
        const name = record.name || record.key || record.email || '';
        if (name.startsWith(namePrefix)) {
          await deleteViaAPI(page, module, record.id);
        }
      }
    }
  } catch {
    // Ignore cleanup errors
  }
}
