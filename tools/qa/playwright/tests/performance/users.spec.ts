import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRow } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Users load times', () => {
  test('Users', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Users', async () => {
      await page.goto('/settings/users');
      await expect(tableRow(page)).toBeVisible();
    });
  });
});
