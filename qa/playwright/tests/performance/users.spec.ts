import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

test.describe('Users load times', () => {
  test('Users', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Users', async () => {
      await page.goto('/settings/users');
      await expect(tableRow(page)).toBeVisible();
    });
  });
});
