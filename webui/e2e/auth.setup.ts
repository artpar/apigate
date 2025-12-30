import { test as setup, expect } from '@playwright/test';
import { TEST_USER, AUTH_FILE } from './helpers/test-utils';

/**
 * Global authentication setup for E2E tests.
 *
 * This runs once before all tests to:
 * 1. Check if initial setup is required
 * 2. Create test user if needed via registration
 * 3. Login and save auth state
 */
setup('authenticate', async ({ page }) => {
  // Check if setup is required (first run)
  const setupResponse = await page.request.get('/mod/auth/setup-required');
  const setupData = await setupResponse.json();

  if (setupData.setup_required) {
    console.log('Initial setup required - creating admin user...');

    // Perform initial setup
    const setupResult = await page.request.post('/mod/auth/setup', {
      data: {
        email: TEST_USER.email,
        password: TEST_USER.password,
        name: TEST_USER.name,
      },
    });

    if (!setupResult.ok()) {
      const errorData = await setupResult.json();
      console.error('Setup failed:', errorData);
      throw new Error(`Initial setup failed: ${errorData.error || 'Unknown error'}`);
    }

    console.log('Admin user created via setup');
  } else {
    // Try to register the test user (might already exist)
    console.log('Attempting to register test user...');
    const registerResult = await page.request.post('/mod/auth/register', {
      data: {
        email: TEST_USER.email,
        password: TEST_USER.password,
        name: TEST_USER.name,
      },
    });

    if (registerResult.ok()) {
      console.log('Test user registered successfully');
      // Registration auto-logs in - save state and return
      await page.context().storageState({ path: AUTH_FILE });
      return;
    } else {
      const registerData = await registerResult.json();
      console.log('Registration response:', registerData);
      // User might already exist, try login
    }
  }

  // Navigate to login page
  await page.goto('/mod/ui/login');
  await page.waitForLoadState('networkidle');

  // Check if already authenticated (redirected to dashboard)
  if (!page.url().includes('/login')) {
    console.log('Already authenticated');
    await page.context().storageState({ path: AUTH_FILE });
    return;
  }

  // Fill login form
  await page.locator('input#email, input[name="email"]').fill(TEST_USER.email);
  await page.locator('input#password, input[name="password"]').fill(TEST_USER.password);
  await page.locator('button[type="submit"]').click();

  // Wait for redirect after login
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(1000);

  // Check if login succeeded
  if (page.url().includes('/login')) {
    // Login failed - check for error
    const errorText = await page.locator('.text-red-700, .text-red-500, [role="alert"]').textContent();
    console.error('Login failed:', errorText);

    // Maybe user exists with different password - let's check /me endpoint for current session
    const meResponse = await page.request.get('/mod/auth/me');
    if (meResponse.ok()) {
      console.log('Auth check successful - using existing session');
      await page.context().storageState({ path: AUTH_FILE });
      return;
    }

    throw new Error(`Login failed: ${errorText}. Please ensure test user exists with correct password.`);
  }

  // Save auth state
  await page.context().storageState({ path: AUTH_FILE });
  console.log('Auth state saved to', AUTH_FILE);
});
