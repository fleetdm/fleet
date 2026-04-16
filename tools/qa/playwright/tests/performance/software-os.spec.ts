import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRowWithContent } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Software OS load times', () => {
  test('OS page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS page', async () => {
      await page.goto('/software/os');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── Platform filters ────────────────────────────────────────────────────────
  test('Platform filter - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - macOS', async () => {
      await page.goto('/software/os?platform=darwin');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Platform filter - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - Windows', async () => {
      await page.goto('/software/os?platform=windows');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Platform filter - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Platform - Linux', async () => {
      await page.goto('/software/os?platform=linux');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── Sorting ─────────────────────────────────────────────────────────────────
  test('Sort hosts ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts ascending', async () => {
      await page.goto('/software/os?order_key=hosts_count&order_direction=asc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Sort hosts descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts descending', async () => {
      await page.goto('/software/os?order_key=hosts_count&order_direction=desc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── View hosts for top OS ───────────────────────────────────────────────────
  test('View hosts for top OS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/os?order_key=hosts_count&order_direction=desc');
    await expect(tableRowWithContent(page)).toBeVisible();

    await measureNav(page, testInfo, 'Top OS hosts page', async () => {
      await page.getByRole('table').locator('tbody tr').first().click();
      // Wait for the vulnerabilities table to populate on the OS detail page
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });
});
