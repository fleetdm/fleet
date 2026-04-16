import { test, expect } from '@playwright/test';

// Login flow tests start with a fresh session (no stored auth)
test.use({ storageState: { cookies: [], origins: [] } });

// SimpleSAML test IdP credentials (from tools/saml in docker-compose)
const SSO_ADMIN = {
  username: 'sso_user_3_global_admin',       // SimpleSAML username (not email)
  email: 'sso_user_3_global_admin@example.com',
  password: 'user123#',
  displayName: 'SSO User 3',
};

test.describe('Login flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('blank fields show validation prompts', async ({ page }) => {
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Email field must be completed')).toBeVisible();
    await expect(page.getByText('Password field must be completed')).toBeVisible();
  });

  test('invalid credentials show authentication failed', async ({ page }) => {
    await page.getByPlaceholder('Email').fill('nonexistent@example.com');
    await page.getByPlaceholder('Password').fill('WrongPassword999!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Authentication failed')).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test('valid email with wrong password shows authentication failed', async ({ page }) => {
    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill('WrongPassword999!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Authentication failed')).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test('forgot password link prompts for email', async ({ page }) => {
    await page.getByRole('link', { name: 'Forgot password?' }).click();

    await expect(page).toHaveURL(/\/login\/forgot/);
    await expect(page.getByRole('heading', { name: 'Reset password' })).toBeVisible();
    await expect(page.getByPlaceholder('Email')).toBeVisible();
  });

  test('valid credentials result in successful login', async ({ page }) => {
    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill(process.env.FLEET_ADMIN_PASSWORD!);
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page).toHaveURL(/\/dashboard/);
  });

  // ── SSO login tests ────────────────────────────────────────────────────────
  // These tests use the SimpleSAML IdP from Fleet's docker-compose setup.
  // Set FLEET_SSO_ENABLED=1 when the simplesaml container is running.
  //
  // Before running, SSO must be configured in Fleet:
  //   Identity Provider Name: SimpleSAML
  //   Entity ID: https://localhost:8080
  //   Metadata URL: http://127.0.0.1:9080/simplesaml/saml2/idp/metadata.php
  //
  // And the SSO user must be invited via the Fleet UI or API with SSO enabled.
  test.describe('SSO login', () => {
    test.skip(
      !process.env.FLEET_SSO_ENABLED,
      'SSO tests require FLEET_SSO_ENABLED=1 and the SimpleSAML container running'
    );

    // SSO tests must hit localhost:8080 because the SimpleSAML IdP redirects
    // back to the entity_id (https://localhost:8080) after authentication.
    test.use({ baseURL: 'https://localhost:8080' });

    // Ensure SSO is configured and the test user exists before the suite runs
    test.beforeAll(async ({ request }) => {
      const headers = { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` };

      // Configure SSO with JIT provisioning — Fleet auto-creates the user on
      // first SSO login using SAML attributes, no invite acceptance needed.
      const configRes = await request.patch('/api/latest/fleet/config', {
        headers,
        data: {
          sso_settings: {
            enable_sso: true,
            enable_jit_provisioning: true,
            idp_name: 'SimpleSAML',
            entity_id: 'https://localhost:8080',
            metadata_url: 'http://localhost:9080/simplesaml/saml2/idp/metadata.php',
            sso_server_url: 'https://localhost:8080',
          },
        },
      });
      expect(configRes.ok()).toBeTruthy();
    });

    test('SSO button is visible on the login page', async ({ page }) => {
      await page.goto('/login');

      await expect(
        page.getByRole('button', { name: /sign in with/i })
      ).toBeVisible();
    });

    test('valid SSO credentials result in successful login', async ({ page }) => {
      await page.goto('/login');
      await page.getByRole('button', { name: /sign in with/i }).click();

      // SimpleSAML login page — fill in credentials (uses username, not email)
      await expect(page.locator('input[name="username"]')).toBeVisible({ timeout: 15_000 });
      await page.locator('input[name="username"]').fill(SSO_ADMIN.username);
      await page.locator('input[name="password"]').fill(SSO_ADMIN.password);
      await page.locator('button[type="submit"], input[type="submit"]').click();

      // After IdP redirect back to Fleet, user should land on the dashboard
      await expect(page).toHaveURL(/\/dashboard/, { timeout: 30_000 });
    });

    test('invalid SSO credentials stay on the IdP login page', async ({ page }) => {
      await page.goto('/login');
      await page.getByRole('button', { name: /sign in with/i }).click();

      // SimpleSAML login page — fill in bad credentials
      await expect(page.locator('input[name="username"]')).toBeVisible({ timeout: 15_000 });
      await page.locator('input[name="username"]').fill('bad_user@example.com');
      await page.locator('input[name="password"]').fill('wrongpassword');
      await page.locator('button[type="submit"], input[type="submit"]').click();

      // Should NOT reach the Fleet dashboard
      await expect(page).not.toHaveURL(/\/dashboard/);
    });
  });
});
