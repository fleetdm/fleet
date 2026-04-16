import { test, expect } from '@playwright/test';
import { loginAsAdmin } from '../../helpers/auth';

test.describe('Logout', () => {
  // Use a blank session so this test never touches the shared auth state.
  // The logout action invalidates the server-side session, which would break
  // the stored cookies used by all other tests.
  test.use({ storageState: { cookies: [], origins: [] } });

  test('sign out returns to login page', async ({ page }) => {
    // Log in with a fresh session
    await loginAsAdmin(
      page,
      process.env.FLEET_ADMIN_EMAIL!,
      process.env.FLEET_ADMIN_PASSWORD!,
    );

    // Sign out
    await page.goto('/dashboard');
    await page.locator('.user-menu-select__value-container').click();
    await page.getByRole('menuitem', { name: 'Sign out' }).click();

    await expect(page).toHaveURL(/\/login/);
  });
});
