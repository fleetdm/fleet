import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';
import { tableRow, tableOrEmpty, selectStatusFilter, selectPlatformFilter, selectFirstCustomLabel } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Hosts load times', () => {
  test('Hosts list', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Hosts list', async () => {
      await page.goto('/hosts/manage');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Online status filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Online status filter', async () => {
      await selectStatusFilter(page, 'Online');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - macOS', async () => {
      await selectPlatformFilter(page, 'macOS');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - Windows', async () => {
      await selectPlatformFilter(page, 'Windows');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - Linux', async () => {
      await selectPlatformFilter(page, 'Linux');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // TODO: Replace with a fixed label ID for test stability
  test('Label filter - first available', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Label filter', async () => {
      await selectFirstCustomLabel(page);
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Search host by name', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    const firstHostName = await tableRow(page).getByRole('link').first().textContent();

    await measureSearch(
      page, testInfo, 'Search host by name',
      page.getByPlaceholder('Search'), firstHostName!.trim(),
      async () => { await expect(page.getByRole('table').getByText(firstHostName!.trim()).first()).toBeVisible(); }
    );
  });

  test('Sort by Host name ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name ascending', async () => {
      await page.goto('/hosts/manage?order_key=display_name&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort by Host name descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Sort name descending', async () => {
      await page.goto('/hosts/manage?order_key=display_name&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });
});
