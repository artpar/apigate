/**
 * Screenshot Capture for User Journeys
 *
 * This test file captures screenshots at each step of every user journey
 * defined in docs/USER_JOURNEYS.md
 *
 * Usage:
 *   npx playwright test capture-journeys.spec.ts --project=chromium
 *
 * Output:
 *   docs/screenshots/{journey}/
 *   docs/gifs/{journey}.gif
 */

import { test, expect, Page } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

// Screenshot output directory
const SCREENSHOT_DIR = path.join(__dirname, '../../docs/screenshots');
const GIF_FRAMES_DIR = path.join(__dirname, '../../docs/.gif-frames');

// Ensure directories exist
function ensureDir(dir: string) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

// Screenshot helper
async function capture(page: Page, journey: string, step: string, description: string) {
  const journeyDir = path.join(SCREENSHOT_DIR, journey);
  ensureDir(journeyDir);

  const filename = `${step}-${description}.png`;
  await page.screenshot({
    path: path.join(journeyDir, filename),
    fullPage: false,
  });

  console.log(`  ✓ Captured: ${journey}/${filename}`);
}

// GIF frame capture helper (captures sequential frames for later GIF generation)
async function captureGifFrame(page: Page, gifName: string, frameNum: number) {
  const gifDir = path.join(GIF_FRAMES_DIR, gifName);
  ensureDir(gifDir);

  await page.screenshot({
    path: path.join(gifDir, `frame-${String(frameNum).padStart(3, '0')}.png`),
    fullPage: false,
  });
}

// Test credentials
const ADMIN_EMAIL = 'admin@apigate.local';
const ADMIN_PASSWORD = 'AdminPass123!';
const CUSTOMER_EMAIL = 'customer@example.com';
const CUSTOMER_PASSWORD = 'CustomerPass123!';

// ============================================================================
// J1: FIRST-TIME SETUP (Admin)
// ============================================================================
test.describe('J1: First-Time Setup', () => {
  test.skip('capture setup wizard flow', async ({ page }) => {
    // NOTE: This test requires a fresh database with no setup complete
    // Run with: FRESH_DB=1 npx playwright test capture-journeys.spec.ts -g "J1"

    await page.goto('/');

    // J1-01: Welcome/redirect to setup
    await page.waitForURL('**/setup**');
    await capture(page, 'j1-setup', '01', 'welcome');

    // J1-02: Enter upstream URL
    await page.locator('input[name="upstream_url"]').fill('https://api.example.com');
    await capture(page, 'j1-setup', '02', 'upstream');
    await page.locator('button:has-text("Next")').click();

    // J1-03: Create admin account
    await page.locator('input[name="email"]').fill(ADMIN_EMAIL);
    await page.locator('input[name="password"]').fill(ADMIN_PASSWORD);
    await capture(page, 'j1-setup', '03', 'admin');
    await page.locator('button:has-text("Next")').click();

    // J1-04: Create first plan
    await page.locator('input[name="plan_name"]').fill('Free');
    await page.locator('input[name="requests_per_month"]').fill('1000');
    await page.locator('input[name="rate_limit"]').fill('10');
    await capture(page, 'j1-setup', '04', 'plan');
    await page.locator('button:has-text("Complete")').click();

    // J1-05: Setup complete
    await page.waitForURL('**/ui/**');
    await capture(page, 'j1-setup', '05', 'complete');

    // J1-06: Dashboard with checklist
    await page.waitForLoadState('networkidle');
    await capture(page, 'j1-setup', '06', 'dashboard');
  });
});

// ============================================================================
// J2: PLAN MANAGEMENT (Admin)
// ============================================================================
test.describe('J2: Plan Management', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin
    await page.goto('/ui/login');
    await page.locator('input[name="email"]').fill(ADMIN_EMAIL);
    await page.locator('input[name="password"]').fill(ADMIN_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/ui/**');
  });

  test('capture plan management flow', async ({ page }) => {
    // J2-01: Plans list
    await page.goto('/ui/plan');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j2-plans', '01', 'plans-list');

    // J2-02: Create new form
    await page.locator('a:has-text("Create"), button:has-text("Create")').first().click();
    await page.waitForLoadState('networkidle');
    await capture(page, 'j2-plans', '02', 'create-form');

    // J2-03: Fill free plan
    await page.locator('input[name="name"]').fill('Free Tier');
    await page.locator('input[name="rate_limit_per_minute"]').fill('10');
    await page.locator('input[name="requests_per_month"]').fill('1000');
    await page.locator('input[name="price_monthly"]').fill('0');
    await capture(page, 'j2-plans', '03', 'free-plan');

    // J2-04: Submit and view created
    await page.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    await capture(page, 'j2-plans', '04', 'created');

    // J2-05: Create pro plan
    await page.goto('/ui/plan/new');
    await page.waitForLoadState('networkidle');
    await page.locator('input[name="name"]').fill('Pro');
    await page.locator('input[name="rate_limit_per_minute"]').fill('600');
    await page.locator('input[name="requests_per_month"]').fill('100000');
    await page.locator('input[name="price_monthly"]').fill('2900');
    await capture(page, 'j2-plans', '05', 'pro-plan');
    await page.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');

    // J2-06: Set as default (if action available)
    await page.goto('/ui/plan');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j2-plans', '06', 'default');
  });
});

// ============================================================================
// J3: MONITOR & MANAGE (Admin)
// ============================================================================
test.describe('J3: Monitor & Manage', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/ui/login');
    await page.locator('input[name="email"]').fill(ADMIN_EMAIL);
    await page.locator('input[name="password"]').fill(ADMIN_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/ui/**');
  });

  test('capture monitoring flow', async ({ page }) => {
    // J3-01: Dashboard
    await page.goto('/ui/');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j3-monitor', '01', 'dashboard');

    // J3-02: Users list
    await page.goto('/ui/user');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j3-monitor', '02', 'users-list');

    // J3-03: User details (click first user if exists)
    const userRow = page.locator('table tbody tr').first();
    if (await userRow.count() > 0) {
      await userRow.click();
      await page.waitForLoadState('networkidle');
      await capture(page, 'j3-monitor', '03', 'user-detail');
    }

    // J3-04: API Keys list
    await page.goto('/ui/api_key');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j3-monitor', '04', 'keys-list');

    // J3-05: Key details
    const keyRow = page.locator('table tbody tr').first();
    if (await keyRow.count() > 0) {
      await keyRow.click();
      await page.waitForLoadState('networkidle');
      await capture(page, 'j3-monitor', '05', 'key-detail');
    }
  });
});

// ============================================================================
// J4: CONFIGURE PLATFORM (Admin)
// ============================================================================
test.describe('J4: Configure Platform', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/ui/login');
    await page.locator('input[name="email"]').fill(ADMIN_EMAIL);
    await page.locator('input[name="password"]').fill(ADMIN_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/ui/**');
  });

  test('capture settings flow', async ({ page }) => {
    // J4-01: Settings page
    await page.goto('/ui/setting');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j4-config', '01', 'settings');

    // J4-02: Upstream settings
    await page.goto('/ui/upstream');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j4-config', '02', 'upstream');
  });
});

// ============================================================================
// J5: CUSTOMER ONBOARDING
// ============================================================================
test.describe('J5: Customer Onboarding', () => {
  test('capture signup and login flow', async ({ page }) => {
    // J5-01: Portal home
    await page.goto('/portal/');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j5-onboarding', '01', 'portal-home');

    // J5-02: Signup form
    await page.goto('/portal/signup');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j5-onboarding', '02', 'signup-form');

    // J5-03: Filled form
    const uniqueEmail = `test-${Date.now()}@example.com`;
    await page.locator('input[name="name"]').fill('Test Customer');
    await page.locator('input[name="email"]').fill(uniqueEmail);
    await page.locator('input[name="password"]').fill('TestPass123!');
    await capture(page, 'j5-onboarding', '03', 'filled-form');

    // J5-04: Submit (may succeed or fail based on state)
    await page.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    await capture(page, 'j5-onboarding', '04', 'submitted');

    // J5-05: Login page
    await page.goto('/portal/login');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j5-onboarding', '05', 'login');

    // J5-06: Credentials entered
    await page.locator('input[name="email"]').fill(CUSTOMER_EMAIL);
    await page.locator('input[name="password"]').fill(CUSTOMER_PASSWORD);
    await capture(page, 'j5-onboarding', '06', 'credentials');

    // J5-07: Dashboard (after login)
    await page.locator('button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    await capture(page, 'j5-onboarding', '07', 'dashboard');
  });
});

// ============================================================================
// J6: GET API ACCESS (Customer)
// ============================================================================
test.describe('J6: Get API Access', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/portal/login');
    await page.locator('input[name="email"]').fill(CUSTOMER_EMAIL);
    await page.locator('input[name="password"]').fill(CUSTOMER_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/portal/**');
  });

  test('capture API key creation flow', async ({ page }) => {
    // J6-01: API Keys page
    await page.goto('/portal/keys');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j6-api-access', '01', 'keys-empty');

    // J6-02: Create key dialog
    await page.locator('button:has-text("Create"), a:has-text("Create")').first().click();
    await page.waitForTimeout(500);
    await capture(page, 'j6-api-access', '02', 'create-dialog');

    // J6-03: Name entered
    const nameInput = page.locator('input[name="name"], input[placeholder*="name"]');
    if (await nameInput.count() > 0) {
      await nameInput.fill('Production');
      await capture(page, 'j6-api-access', '03', 'name-entered');
    }

    // J6-04: Key shown (after creation)
    await page.locator('button:has-text("Create"), button[type="submit"]').click();
    await page.waitForLoadState('networkidle');
    await capture(page, 'j6-api-access', '04', 'key-shown');

    // J6-05: Copy confirmation
    const copyBtn = page.locator('button:has-text("Copy")');
    if (await copyBtn.count() > 0) {
      await copyBtn.click();
      await page.waitForTimeout(500);
      await capture(page, 'j6-api-access', '05', 'copied');
    }

    // J6-06: Key in list
    await page.goto('/portal/keys');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j6-api-access', '06', 'in-list');
  });
});

// ============================================================================
// J7: MONITOR USAGE (Customer)
// ============================================================================
test.describe('J7: Monitor Usage', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/portal/login');
    await page.locator('input[name="email"]').fill(CUSTOMER_EMAIL);
    await page.locator('input[name="password"]').fill(CUSTOMER_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/portal/**');
  });

  test('capture usage monitoring flow', async ({ page }) => {
    // J7-01: Usage page
    await page.goto('/portal/usage');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j7-usage', '01', 'usage-page');

    // J7-02: Current period stats
    await capture(page, 'j7-usage', '02', 'current');

    // J7-03: Quota bar (if visible)
    await capture(page, 'j7-usage', '03', 'quota-bar');
  });
});

// ============================================================================
// J8: UPGRADE PLAN (Customer)
// ============================================================================
test.describe('J8: Upgrade Plan', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/portal/login');
    await page.locator('input[name="email"]').fill(CUSTOMER_EMAIL);
    await page.locator('input[name="password"]').fill(CUSTOMER_PASSWORD);
    await page.locator('button[type="submit"]').click();
    await page.waitForURL('**/portal/**');
  });

  test('capture plan upgrade flow', async ({ page }) => {
    // J8-01: Plans page
    await page.goto('/portal/plans');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j8-upgrade', '01', 'plans-page');

    // J8-02: Plan comparison
    await capture(page, 'j8-upgrade', '02', 'comparison');

    // J8-03: Select pro plan
    const proCard = page.locator('text=Pro').first();
    if (await proCard.count() > 0) {
      await proCard.scrollIntoViewIfNeeded();
      await capture(page, 'j8-upgrade', '03', 'select-pro');
    }

    // J8-04: Upgrade button
    const upgradeBtn = page.locator('button:has-text("Upgrade")').first();
    if (await upgradeBtn.count() > 0) {
      await upgradeBtn.scrollIntoViewIfNeeded();
      await capture(page, 'j8-upgrade', '04', 'upgrade-btn');
    }
  });
});

// ============================================================================
// J9: DOCUMENTATION PORTAL
// ============================================================================
test.describe('J9: Documentation Portal', () => {
  test('capture docs flow', async ({ page }) => {
    // J9-01: Docs home
    await page.goto('/docs/');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '01', 'docs-home');

    // J9-02: Quickstart
    await page.goto('/docs/quickstart');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '02', 'quickstart');

    // J9-03: Authentication
    await page.goto('/docs/authentication');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '03', 'auth');

    // J9-04: API Reference
    await page.goto('/docs/reference');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '04', 'reference');

    // J9-05: Examples
    await page.goto('/docs/examples');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '05', 'examples');

    // J9-07: Try It
    await page.goto('/docs/try-it');
    await page.waitForLoadState('networkidle');
    await capture(page, 'j9-docs', '07', 'try-it');
  });
});

// ============================================================================
// ERROR JOURNEYS
// ============================================================================
test.describe('Error States', () => {
  test('capture error states', async ({ page }) => {
    const errorsDir = path.join(SCREENSHOT_DIR, 'errors');
    ensureDir(errorsDir);

    // E1: Invalid API key (via Try It or direct)
    await page.goto('/docs/try-it');
    await page.waitForLoadState('networkidle');

    const keyInput = page.locator('input[placeholder*="API"], input[name*="key"]');
    if (await keyInput.count() > 0) {
      await keyInput.fill('ak_invalid_key_here');

      const sendBtn = page.locator('button:has-text("Send"), button:has-text("Test")');
      if (await sendBtn.count() > 0) {
        await sendBtn.click();
        await page.waitForLoadState('networkidle');
        await page.screenshot({ path: path.join(errorsDir, 'e1-invalid-key.png') });
      }
    }
  });
});
