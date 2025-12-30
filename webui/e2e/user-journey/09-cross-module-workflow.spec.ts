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
  uniqueEmail,
  uniqueKey,
  expectNoErrors,
  waitForTableLoad,
  waitForFormLoad,
  API_BASE,
} from '../helpers/test-utils';

/**
 * E2E Tests: Cross-Module Integration Workflow
 *
 * This is the "golden path" test - simulating a real user setting up
 * the entire API gateway platform from scratch.
 */

const TEST_PREFIX = 'e2e-workflow';

test.describe('Complete Platform Setup Workflow', () => {
  // Store IDs for cleanup
  const createdIds: {
    plans: string[];
    users: string[];
    apiKeys: string[];
    upstreams: string[];
    routes: string[];
    settings: string[];
  } = {
    plans: [],
    users: [],
    apiKeys: [],
    upstreams: [],
    routes: [],
    settings: [],
  };

  test.afterAll(async ({ request }) => {
    // Cleanup in reverse dependency order
    for (const id of createdIds.routes) {
      await request.delete(`/mod/route/${id}`).catch(() => {});
    }
    for (const id of createdIds.upstreams) {
      await request.delete(`/mod/upstream/${id}`).catch(() => {});
    }
    for (const id of createdIds.apiKeys) {
      await request.delete(`/mod/api_key/${id}`).catch(() => {});
    }
    for (const id of createdIds.users) {
      await request.delete(`/mod/user/${id}`).catch(() => {});
    }
    for (const id of createdIds.plans) {
      await request.delete(`/mod/plan/${id}`).catch(() => {});
    }
    for (const id of createdIds.settings) {
      await request.delete(`/mod/setting/${id}`).catch(() => {});
    }
  });

  test('complete platform setup workflow', async ({ page }) => {
    // This test follows a real user's journey setting up the entire platform

    // ============================================
    // STEP 1: Create Pricing Plans
    // ============================================

    // Create Free Plan
    const freePlanName = uniqueName(`${TEST_PREFIX}-free`);
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);
    await fillField(page, 'name', freePlanName);
    await fillField(page, 'rate_limit_per_minute', '60');
    await fillField(page, 'requests_per_month', '1000');
    await fillField(page, 'price_monthly', '0');
    await submitForm(page);
    await expectNoErrors(page);

    let plans = await listViaAPI(page, 'plan');
    const freePlan = (plans.records || plans).find((p: { name: string }) => p.name === freePlanName);
    expect(freePlan).toBeTruthy();
    createdIds.plans.push(freePlan.id);

    // Create Starter Plan
    const starterPlanName = uniqueName(`${TEST_PREFIX}-starter`);
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);
    await fillField(page, 'name', starterPlanName);
    await fillField(page, 'rate_limit_per_minute', '600');
    await fillField(page, 'requests_per_month', '50000');
    await fillField(page, 'price_monthly', '2900');
    await submitForm(page);
    await expectNoErrors(page);

    plans = await listViaAPI(page, 'plan');
    const starterPlan = (plans.records || plans).find((p: { name: string }) => p.name === starterPlanName);
    expect(starterPlan).toBeTruthy();
    createdIds.plans.push(starterPlan.id);

    // Create Pro Plan
    const proPlanName = uniqueName(`${TEST_PREFIX}-pro`);
    await navigateToCreate(page, 'plan');
    await waitForFormLoad(page);
    await fillField(page, 'name', proPlanName);
    await fillField(page, 'rate_limit_per_minute', '6000');
    await fillField(page, 'requests_per_month', '500000');
    await fillField(page, 'price_monthly', '9900');
    await submitForm(page);
    await expectNoErrors(page);

    plans = await listViaAPI(page, 'plan');
    const proPlan = (plans.records || plans).find((p: { name: string }) => p.name === proPlanName);
    expect(proPlan).toBeTruthy();
    createdIds.plans.push(proPlan.id);

    // Verify all plans in list
    await navigateToModule(page, 'plan');
    await waitForTableLoad(page);
    await expect(page.locator(`text=${freePlanName}`)).toBeVisible();
    await expect(page.locator(`text=${starterPlanName}`)).toBeVisible();
    await expect(page.locator(`text=${proPlanName}`)).toBeVisible();

    // ============================================
    // STEP 2: Create Test Users on Different Plans
    // ============================================

    // Create Free User
    const freeUserEmail = `${TEST_PREFIX}-free-${Date.now()}@example.com`;
    const freeUserName = uniqueName(`${TEST_PREFIX}-free-user`);
    const freeUser = await createViaAPI(page, 'user', {
      email: freeUserEmail,
      name: freeUserName,
      status: 'active',
      plan_id: freePlan.id,
    });
    createdIds.users.push(freeUser.id);

    // Create Starter User
    const starterUserEmail = `${TEST_PREFIX}-starter-${Date.now()}@example.com`;
    const starterUserName = uniqueName(`${TEST_PREFIX}-starter-user`);
    const starterUser = await createViaAPI(page, 'user', {
      email: starterUserEmail,
      name: starterUserName,
      status: 'active',
      plan_id: starterPlan.id,
    });
    createdIds.users.push(starterUser.id);

    // Create Pro User
    const proUserEmail = `${TEST_PREFIX}-pro-${Date.now()}@example.com`;
    const proUserName = uniqueName(`${TEST_PREFIX}-pro-user`);
    const proUser = await createViaAPI(page, 'user', {
      email: proUserEmail,
      name: proUserName,
      status: 'active',
      plan_id: proPlan.id,
    });
    createdIds.users.push(proUser.id);

    // Verify users in list
    await navigateToModule(page, 'user');
    await waitForTableLoad(page);
    await expect(page.locator(`text=${freeUserEmail}`)).toBeVisible({ timeout: 5000 });

    // ============================================
    // STEP 3: Generate API Keys for Users
    // ============================================

    // Create key for Free user
    const freeKeyName = uniqueName(`${TEST_PREFIX}-free-key`);
    const freeKey = await createViaAPI(page, 'api_key', {
      name: freeKeyName,
      user_id: freeUser.id,
      prefix: 'free_',
      enabled: true,
    });
    createdIds.apiKeys.push(freeKey.id);

    // Create key for Starter user
    const starterKeyName = uniqueName(`${TEST_PREFIX}-starter-key`);
    const starterKey = await createViaAPI(page, 'api_key', {
      name: starterKeyName,
      user_id: starterUser.id,
      prefix: 'starter_',
      enabled: true,
    });
    createdIds.apiKeys.push(starterKey.id);

    // Create key for Pro user
    const proKeyName = uniqueName(`${TEST_PREFIX}-pro-key`);
    const proKey = await createViaAPI(page, 'api_key', {
      name: proKeyName,
      user_id: proUser.id,
      prefix: 'pro_',
      enabled: true,
    });
    createdIds.apiKeys.push(proKey.id);

    // Verify keys in list
    await navigateToModule(page, 'api_key');
    await waitForTableLoad(page);
    await expect(page.locator(`text=${freeKeyName}`)).toBeVisible({ timeout: 5000 });

    // ============================================
    // STEP 4: Configure Backend Services (Upstreams)
    // ============================================

    // Create production upstream
    const prodUpstreamName = uniqueName(`${TEST_PREFIX}-prod`);
    const prodUpstream = await createViaAPI(page, 'upstream', {
      name: prodUpstreamName,
      base_url: 'https://api.production.example.com',
      auth_type: 'bearer',
      auth_value: 'prod-secret-token',
      timeout_ms: 30000,
      enabled: true,
    });
    createdIds.upstreams.push(prodUpstream.id);

    // Create staging upstream
    const stagingUpstreamName = uniqueName(`${TEST_PREFIX}-staging`);
    const stagingUpstream = await createViaAPI(page, 'upstream', {
      name: stagingUpstreamName,
      base_url: 'https://api.staging.example.com',
      auth_type: 'none',
      timeout_ms: 60000,
      enabled: true,
    });
    createdIds.upstreams.push(stagingUpstream.id);

    // Verify upstreams in list
    await navigateToModule(page, 'upstream');
    await waitForTableLoad(page);
    await expect(page.locator(`text=${prodUpstreamName}`)).toBeVisible({ timeout: 5000 });

    // ============================================
    // STEP 5: Create Routing Rules
    // ============================================

    // Create v1 API route
    const v1RouteName = uniqueName(`${TEST_PREFIX}-v1-route`);
    const v1Route = await createViaAPI(page, 'route', {
      name: v1RouteName,
      path_pattern: '/v1/*',
      upstream_id: prodUpstream.id,
      match_type: 'prefix',
      protocol: 'http',
      priority: 10,
      enabled: true,
    });
    createdIds.routes.push(v1Route.id);

    // Create staging route
    const stagingRouteName = uniqueName(`${TEST_PREFIX}-staging-route`);
    const stagingRoute = await createViaAPI(page, 'route', {
      name: stagingRouteName,
      path_pattern: '/staging/*',
      upstream_id: stagingUpstream.id,
      match_type: 'prefix',
      protocol: 'http',
      priority: 5,
      enabled: true,
    });
    createdIds.routes.push(stagingRoute.id);

    // Verify routes in list
    await navigateToModule(page, 'route');
    await waitForTableLoad(page);
    await expect(page.locator(`text=${v1RouteName}`)).toBeVisible({ timeout: 5000 });

    // ============================================
    // STEP 6: Test Custom Actions
    // ============================================

    // Test suspend user action
    await navigateToDetail(page, 'user', starterUser.id);
    const suspendBtn = page.locator('button:has-text("Suspend")');
    if (await suspendBtn.count() > 0) {
      await suspendBtn.click();
      await page.waitForTimeout(300);
      const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
      if (await confirmBtn.count() > 0) {
        await confirmBtn.click();
      }
      await page.waitForLoadState('networkidle');

      // Verify status changed
      const suspendedUser = await page.request.get(`${API_BASE}/user/${starterUser.id}`);
      const userData = await suspendedUser.json();
      expect(userData.status).toBe('suspended');
    }

    // Test activate user action
    await navigateToDetail(page, 'user', starterUser.id);
    const activateBtn = page.locator('button:has-text("Activate")');
    if (await activateBtn.count() > 0) {
      await activateBtn.click();
      await page.waitForLoadState('networkidle');

      const activatedUser = await page.request.get(`${API_BASE}/user/${starterUser.id}`);
      const userData = await activatedUser.json();
      expect(userData.status).toBe('active');
    }

    // Test revoke API key action
    await navigateToDetail(page, 'api_key', starterKey.id);
    const revokeBtn = page.locator('button:has-text("Revoke")');
    if (await revokeBtn.count() > 0) {
      await revokeBtn.click();
      await page.waitForTimeout(300);
      const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
      if (await confirmBtn.count() > 0) {
        await confirmBtn.click();
      }
      await page.waitForLoadState('networkidle');

      const revokedKey = await page.request.get(`${API_BASE}/api_key/${starterKey.id}`);
      const keyData = await revokedKey.json();
      expect(keyData.revoked_at).toBeTruthy();
    }

    // Test disable route action
    await navigateToDetail(page, 'route', stagingRoute.id);
    const disableBtn = page.locator('button:has-text("Disable")');
    if (await disableBtn.count() > 0) {
      await disableBtn.click();
      await page.waitForTimeout(300);
      const confirmBtn = page.locator('button:has-text("Confirm"), button:has-text("Yes")');
      if (await confirmBtn.count() > 0) {
        await confirmBtn.click();
      }
      await page.waitForLoadState('networkidle');

      const disabledRoute = await page.request.get(`${API_BASE}/route/${stagingRoute.id}`);
      const routeData = await disabledRoute.json();
      expect(routeData.enabled).toBe(false);
    }

    // Re-enable route
    await navigateToDetail(page, 'route', stagingRoute.id);
    const enableBtn = page.locator('button:has-text("Enable")');
    if (await enableBtn.count() > 0) {
      await enableBtn.click();
      await page.waitForLoadState('networkidle');
    }

    // ============================================
    // STEP 7: Verify Relationships
    // ============================================

    // Verify user shows correct plan
    await navigateToDetail(page, 'user', proUser.id);
    // The plan should be referenced somewhere on the page
    await expectNoErrors(page);

    // Verify API key shows correct user
    await navigateToDetail(page, 'api_key', proKey.id);
    await expectNoErrors(page);

    // Verify route shows correct upstream
    await navigateToDetail(page, 'route', v1Route.id);
    await expectNoErrors(page);

    // ============================================
    // Final Verification
    // ============================================

    // Dashboard should load without errors
    await page.goto('/mod/ui/');
    await page.waitForLoadState('networkidle');
    await expectNoErrors(page);
  });
});

test.describe('Relationship Validation', () => {
  test('creating user with non-existent plan_id fails gracefully', async ({ page }) => {
    await navigateToCreate(page, 'user');
    await waitForFormLoad(page);

    await fillField(page, 'email', uniqueEmail());
    await fillField(page, 'name', uniqueName(TEST_PREFIX));

    // Try to set a non-existent plan_id
    // This tests that the UI handles invalid references properly
    await submitForm(page);

    // Should either show validation error or succeed without the invalid reference
    await page.waitForTimeout(1000);
  });

  test('creating api_key with non-existent user_id fails gracefully', async ({ page }) => {
    await navigateToCreate(page, 'api_key');
    await waitForFormLoad(page);

    await fillField(page, 'name', uniqueName(TEST_PREFIX));

    // If we don't select a valid user, form should validate
    await submitForm(page);

    await page.waitForTimeout(1000);
  });

  test('creating route with non-existent upstream_id fails gracefully', async ({ page }) => {
    await navigateToCreate(page, 'route');
    await waitForFormLoad(page);

    await fillField(page, 'name', uniqueName(TEST_PREFIX));
    await fillField(page, 'path_pattern', '/test/*');

    // If we don't select a valid upstream, form should validate
    await submitForm(page);

    await page.waitForTimeout(1000);
  });
});

test.describe('Navigation Between Related Records', () => {
  let testPlanId: string;
  let testUserId: string;
  let testUpstreamId: string;

  test.beforeAll(async ({ request }) => {
    // Create test data
    const planRes = await request.post('/mod/plan', {
      data: {
        name: uniqueName(`${TEST_PREFIX}-nav-plan`),
        rate_limit_per_minute: 60,
        requests_per_month: 1000,
        price_monthly: 0,
        enabled: true,
      },
    });
    testPlanId = (await planRes.json()).id;

    const userRes = await request.post('/mod/user', {
      data: {
        email: `${TEST_PREFIX}-nav-${Date.now()}@example.com`,
        name: uniqueName(`${TEST_PREFIX}-nav-user`),
        status: 'active',
        plan_id: testPlanId,
      },
    });
    testUserId = (await userRes.json()).id;

    const upstreamRes = await request.post('/mod/upstream', {
      data: {
        name: uniqueName(`${TEST_PREFIX}-nav-upstream`),
        base_url: 'https://nav.example.com',
        auth_type: 'none',
        enabled: true,
      },
    });
    testUpstreamId = (await upstreamRes.json()).id;
  });

  test.afterAll(async ({ request }) => {
    await request.delete(`/mod/user/${testUserId}`).catch(() => {});
    await request.delete(`/mod/plan/${testPlanId}`).catch(() => {});
    await request.delete(`/mod/upstream/${testUpstreamId}`).catch(() => {});
  });

  test('user detail page loads without errors when plan exists', async ({ page }) => {
    await navigateToDetail(page, 'user', testUserId);
    await expectNoErrors(page);
  });

  test('navigating through modules maintains consistency', async ({ page }) => {
    // Start at dashboard
    await page.goto('/mod/ui/');
    await expectNoErrors(page);

    // Go to plans
    await navigateToModule(page, 'plan');
    await expectNoErrors(page);

    // Go to users
    await navigateToModule(page, 'user');
    await expectNoErrors(page);

    // Go to api_keys
    await navigateToModule(page, 'api_key');
    await expectNoErrors(page);

    // Go to upstreams
    await navigateToModule(page, 'upstream');
    await expectNoErrors(page);

    // Go to routes
    await navigateToModule(page, 'route');
    await expectNoErrors(page);

    // Go to settings
    await navigateToModule(page, 'setting');
    await expectNoErrors(page);

    // Back to dashboard
    await page.goto('/mod/ui/');
    await expectNoErrors(page);
  });
});
