import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Users load times', () => {
  test('Users', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Users', async () => {
      await page.goto('/settings/users');
      await page.waitForLoadState('networkidle');
    });
  });
});
