import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';
import { tableRowWithContent, getNameFromRow } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Software load times', () => {
  test('Software page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Software page', async () => {
      await page.goto('/software/titles');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── Sorting ─────────────────────────────────────────────────────────────────
  test('Sort by name ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name ascending', async () => {
      await page.goto('/software/titles?order_key=name&order_direction=asc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Sort by name descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name descending', async () => {
      await page.goto('/software/titles?order_key=name&order_direction=desc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Sort by host count ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts ascending', async () => {
      await page.goto('/software/titles?order_key=hosts_count&order_direction=asc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Sort by host count descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort hosts descending', async () => {
      await page.goto('/software/titles?order_key=hosts_count&order_direction=desc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── Filters ─────────────────────────────────────────────────────────────────
  test('Vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Vulnerable filter', async () => {
      await page.goto('/software/titles?vulnerable=true');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── Search ──────────────────────────────────────────────────────────────────
  test('Search software', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/titles');
    await expect(tableRowWithContent(page)).toBeVisible();

    const itemName = await getNameFromRow(page);

    await measureSearch(
      page, testInfo, 'Search software',
      page.getByRole('textbox', { name: /Search/ }), itemName,
      async () => { await expect(page.getByRole('table').getByText(itemName).first()).toBeVisible(); },
    );
  });

  // ── Show versions ON ────────────────────────────────────────────────────────
  test('Show versions - page load', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - page load', async () => {
      await page.goto('/software/versions');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - sort name ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort name asc', async () => {
      await page.goto('/software/versions?order_key=name&order_direction=asc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - sort name descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort name desc', async () => {
      await page.goto('/software/versions?order_key=name&order_direction=desc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - sort hosts ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort hosts asc', async () => {
      await page.goto('/software/versions?order_key=hosts_count&order_direction=asc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - sort hosts descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - sort hosts desc', async () => {
      await page.goto('/software/versions?order_key=hosts_count&order_direction=desc');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Versions - vulnerable', async () => {
      await page.goto('/software/versions?vulnerable=true');
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Versions - search', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/versions');
    await expect(tableRowWithContent(page)).toBeVisible();

    const itemName = await getNameFromRow(page);

    await measureSearch(
      page, testInfo, 'Versions - search',
      page.getByRole('textbox', { name: /Search/ }), itemName,
      async () => { await expect(page.getByRole('table').getByText(itemName).first()).toBeVisible(); },
    );
  });
});
