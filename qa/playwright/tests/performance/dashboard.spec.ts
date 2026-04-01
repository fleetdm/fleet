import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Dashboard load times', () => {
  test('Dashboard', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Dashboard', async () => {
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
    });
  });
});
