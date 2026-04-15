import { test, expect } from '@playwright/test';

// Forgot password tests start with a fresh session
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Forgot password', () => {
  test('forgot password link navigates to reset page', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: 'Forgot password?' }).click();

    await expect(page).toHaveURL(/\/login\/forgot/);
    await expect(page.getByRole('heading', { name: 'Reset password' })).toBeVisible();
  });
});
