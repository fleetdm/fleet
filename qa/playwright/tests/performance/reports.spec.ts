import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Reports load times', () => {
  test('Reports', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Reports', async () => {
      await page.goto('/reports/manage');
      await page.waitForLoadState('networkidle');
    });
  });
});
