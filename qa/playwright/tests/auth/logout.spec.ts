import { test, expect } from '@playwright/test';

test.describe('Logout', () => {
  test('sign out returns to login page', async ({ page }) => {
    await page.goto('/dashboard');
    await page.locator('.user-menu-select__value-container').click();
    await page.getByRole('menuitem', { name: 'Sign out' }).click();

    await expect(page).toHaveURL(/\/login/);
  });
});
