import { test, expect } from '@playwright/test';
import * as path from 'path';

// Override the global storageState — login tests must start with a fresh session
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Login', () => {
  test('admin can log in and session is saved', async ({ page, context }) => {
    await page.goto('/login');

    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByPlaceholder('Password').fill(process.env.FLEET_ADMIN_PASSWORD!);
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page).not.toHaveURL(/\/login/);

    await context.storageState({
      path: path.resolve(__dirname, '../../.auth/admin.json'),
    });
  });
});
