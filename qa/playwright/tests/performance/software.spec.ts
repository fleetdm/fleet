import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

test.describe('Software load times', () => {
  test('Software page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Software page', async () => {
      await page.goto('/software/titles');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Sorting ─────────────────────────────────────────────────────────────────
  test('Sort by name ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name ascending', async () => {
      await page.goto('/software/titles?order_key=name&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort by name descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name descending', async () => {
      await page.goto('/software/titles?order_key=name&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort by host count ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts ascending', async () => {
      await page.goto('/software/titles?order_key=hosts_count&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort by host count descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts descending', async () => {
      await page.goto('/software/titles?order_key=hosts_count&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Filters ─────────────────────────────────────────────────────────────────
  test('Vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Vulnerable filter', async () => {
      await page.goto('/software/titles?vulnerable=true');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Search ──────────────────────────────────────────────────────────────────
  test('Search software', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/titles');
    await expect(tableRow(page)).toBeVisible();

    const itemName = await tableRow(page).locator('td').nth(1).innerText();

    await measureSearch(
      page, testInfo, 'Search software',
      page.getByPlaceholder('Search by name'), itemName!.trim(),
      async () => { await expect(page.getByRole('table').getByText(itemName!.trim()).first()).toBeVisible(); }
    );
  });

  // ── Show versions ON ────────────────────────────────────────────────────────
  test('Show versions - page load', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - page load', async () => {
      await page.goto('/software/versions');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - sort name ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort name asc', async () => {
      await page.goto('/software/versions?order_key=name&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - sort name descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort name desc', async () => {
      await page.goto('/software/versions?order_key=name&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - sort hosts ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort hosts asc', async () => {
      await page.goto('/software/versions?order_key=hosts_count&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - sort hosts descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort hosts desc', async () => {
      await page.goto('/software/versions?order_key=hosts_count&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - vulnerable', async () => {
      await page.goto('/software/versions?vulnerable=true');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Versions - search', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/versions');
    await expect(tableRow(page)).toBeVisible();

    const itemName = await tableRow(page).locator('td').nth(1).innerText();

    await measureSearch(
      page, testInfo, 'Versions - search',
      page.getByPlaceholder('Search by name'), itemName!.trim(),
      async () => { await expect(page.getByRole('table').getByText(itemName!.trim()).first()).toBeVisible(); }
    );
  });
});
