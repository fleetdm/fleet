import { test, expect } from '@playwright/test';

test.describe('Hosts', () => {
  test('hosts page displays the host list', async ({ page }) => {
    await page.goto('/hosts/manage');

    await expect(page.getByRole('table').locator('tbody').getByRole('row').first()).toBeVisible();
  });
});
