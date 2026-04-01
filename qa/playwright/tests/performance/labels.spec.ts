import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Labels load times', () => {
  test('Labels', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Labels', async () => {
      await page.goto('/labels/manage');
      await page.waitForLoadState('networkidle');
    });
  });
});
