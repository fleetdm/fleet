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
      // Wait for the software table to have actual data rows (not just the shell)
      await expect(page.getByRole('heading', { name: 'Software' })).toBeVisible();
      const softwareRow = page.getByRole('table').locator('tbody tr').first();
      await expect(softwareRow).toBeVisible();
      await expect(softwareRow.locator('td').first()).not.toBeEmpty();
    });
  });

  test('Activity block', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/dashboard');
    await measureNav(page, testInfo, 'Activity block', async () => {
      // Wait for actual activity items to render, not just the heading
      await expect(page.getByText(/logged in|edited|created|deleted|enabled|disabled|transferred|ran/i).first()).toBeVisible();
    });
  });

  test('Filter by macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'macOS', async () => {
      await page.goto('/dashboard/mac');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
    });
  });

  test('Filter by Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Windows', async () => {
      await page.goto('/dashboard/windows');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
    });
  });

  test('Filter by Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Linux', async () => {
      await page.goto('/dashboard/linux');
      await expect(page.locator('[data-testid="card"]').first()).toBeVisible();
    });
  });
});
