import { test } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Controls load times', () => {
  test('OS Updates', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS Updates', async () => {
      await page.goto('/controls/os-updates');
      await page.waitForLoadState('networkidle');
    });
  });

  test('OS Settings', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS Settings', async () => {
      await page.goto('/controls/os-settings');
      await page.waitForLoadState('networkidle');
    });
  });

  test('Setup Experience', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Setup Experience', async () => {
      await page.goto('/controls/setup-experience');
      await page.waitForLoadState('networkidle');
    });
  });

  test('Scripts', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Scripts', async () => {
      await page.goto('/controls/scripts');
      await page.waitForLoadState('networkidle');
    });
  });

  test('Variables', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Variables', async () => {
      await page.goto('/controls/variables');
      await page.waitForLoadState('networkidle');
    });
  });

});
