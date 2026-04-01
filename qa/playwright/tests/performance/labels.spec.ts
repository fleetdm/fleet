import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Labels load times', () => {
  test('Labels', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Labels', async () => {
      await page.goto('/labels/manage');
      await expect(page.getByRole('table').locator('tbody').getByRole('row').first()).toBeVisible();
    });
  });
});
