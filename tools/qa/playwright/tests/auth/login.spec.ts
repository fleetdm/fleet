import { test, expect } from '@playwright/test';

// Login tests always start with a fresh session
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Login', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('admin can log in', { tag: '@loadtest' }, async ({ page }) => {
    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill(process.env.FLEET_ADMIN_PASSWORD!);
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('shows error for invalid email', async ({ page }) => {
    await page.getByPlaceholder('Email').fill('nonexistent@example.com');
    await page.getByPlaceholder('Password').fill('SomePassword123!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Authentication failed')).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test('shows error for valid email with wrong password', async ({ page }) => {
    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill('WrongPassword999!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Authentication failed')).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test('redirects to dashboard when already authenticated', async ({ browser }) => {
    // Use default authenticated context (no storageState override)
    const context = await browser.newContext({
      storageState: '.auth/e2e-admin.json',
      baseURL: process.env.FLEET_URL,
    });
    const page = await context.newPage();

    await page.goto('/login');
    await expect(page).toHaveURL(/\/dashboard/);

    await context.close();
  });
});
