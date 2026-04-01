import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Software load times', () => {
  test('Software', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Software', async () => {
      await page.goto('/software/titles');
      await page.waitForLoadState('networkidle');
    });
  });

  test('OS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS', async () => {
      await page.goto('/software/os');
      await page.waitForLoadState('networkidle');
    });
  });

  test('Vulnerabilities', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Vulnerabilities', async () => {
      await page.goto('/software/vulnerabilities');
      await page.waitForLoadState('networkidle');
    });
  });
});
