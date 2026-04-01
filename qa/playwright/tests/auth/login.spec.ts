import { test, expect } from '@playwright/test';

// Login tests always start with a fresh session
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Login', () => {
  test('admin can log in', { tag: '@loadtest' }, async ({ page }) => {
    await page.goto('/login');

    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill(process.env.FLEET_ADMIN_PASSWORD!);
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page).not.toHaveURL(/\/login/);
  });
});
