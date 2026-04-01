import { test, expect } from '@playwright/test';
import * as path from 'path';

test.use({
  storageState: path.resolve(__dirname, '../../.auth/admin.json'),
});

test.describe('Hosts', () => {
  test('hosts page displays the host list', async ({ page }) => {
    await page.goto('/hosts/manage');

    const firstRow = page.locator('.data-table-block tbody tr').first();
    await expect(firstRow).toBeVisible();
  });
});
