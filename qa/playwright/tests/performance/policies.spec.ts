import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Policies load times', () => {
  test('Policies', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Policies', async () => {
      await page.goto('/policies/manage');
      await page.waitForLoadState('networkidle');
    });
  });
});
