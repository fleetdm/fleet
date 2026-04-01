import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

test.describe('Dashboard load times', () => {
  test('Platform cards', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform cards', async () => {
      await page.goto('/dashboard');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
    });
  });

  test('Software block', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/dashboard');
    await measureNav(page, testInfo, 'Software block', async () => {
      await expect(
        page.locator('.home-software').getByRole('table').locator('tbody').getByRole('row').first()
      ).toBeVisible();
    });
  });

  test('Activity block', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/dashboard');
    await measureNav(page, testInfo, 'Activity block', async () => {
      await expect(page.locator('.activity-feed .global-activity-item').first()).toBeVisible();
    });
  });

  test('Filter by macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'macOS', async () => {
      await page.goto('/dashboard/mac');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
      await expect(page.locator('.operating-systems').getByRole('table')).toBeVisible();
    });
  });

  test('Filter by Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Windows', async () => {
      await page.goto('/dashboard/windows');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
      await expect(page.locator('.operating-systems').getByRole('table')).toBeVisible();
    });
  });

  test('Filter by Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Linux', async () => {
      await page.goto('/dashboard/linux');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
    });
  });
});
