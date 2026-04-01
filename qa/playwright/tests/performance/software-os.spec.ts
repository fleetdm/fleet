import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

test.describe('Software OS load times', () => {
  test('OS page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS page', async () => {
      await page.goto('/software/os');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Platform filters ────────────────────────────────────────────────────────
  test('Platform filter - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - macOS', async () => {
      await page.goto('/software/os?platform=darwin');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Platform filter - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - Windows', async () => {
      await page.goto('/software/os?platform=windows');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Platform filter - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - Linux', async () => {
      await page.goto('/software/os?platform=linux');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Sorting ─────────────────────────────────────────────────────────────────
  test('Sort hosts ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts ascending', async () => {
      await page.goto('/software/os?order_key=hosts_count&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort hosts descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts descending', async () => {
      await page.goto('/software/os?order_key=hosts_count&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── View hosts for top OS ───────────────────────────────────────────────────
  test('View hosts for top OS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/os?order_key=hosts_count&order_direction=desc');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Top OS hosts page', async () => {
      await tableRow(page).click();
      await expect(page.locator('.software-os-details-page')).toBeVisible();
    });
  });
});
