import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Hosts load times', () => {
  test('Hosts list', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Hosts list', async () => {
      await page.goto('/hosts/manage');
      await expect(page.getByRole('table').locator('tbody').getByRole('row').first()).toBeVisible();
    });
  });

  test('Specific host', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    const href = await page.getByRole('table').locator('tbody').getByRole('row').first().getByRole('link').first().getAttribute('href');

    await measureNav(page, testInfo, 'Specific host', async () => {
      await page.goto(href!);
      await page.waitForLoadState('networkidle');
    });
  });
});
