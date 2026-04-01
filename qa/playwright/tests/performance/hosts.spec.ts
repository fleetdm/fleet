import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

const tableOrEmpty = (page: import('@playwright/test').Page) =>
  tableRow(page).or(page.locator('.empty-table__container'));

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

    // Open the status filter dropdown and select "Online"
    await page.locator('.manage-hosts__status-filter .react-select__control').click();
    const onlineOption = page.locator('[data-testid="dropdown-option"]').filter({ hasText: 'Online' });
    await expect(onlineOption).toBeVisible();

    await measureNav(page, testInfo, 'Online status filter', async () => {
      await onlineOption.click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await page.locator('.label-filter-select__control').click();
    const macOption = page.locator('.label-filter-select__option').filter({ hasText: 'macOS' });
    await expect(macOption).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - macOS', async () => {
      await macOption.click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await page.locator('.label-filter-select__control').click();
    const winOption = page.locator('.label-filter-select__option').filter({ hasText: 'Windows' });
    await expect(winOption).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - Windows', async () => {
      await winOption.click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Platform filter - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await page.locator('.label-filter-select__control').click();
    const linuxOption = page.locator('.label-filter-select__option').filter({ hasText: 'Linux' });
    await expect(linuxOption).toBeVisible();

    await measureNav(page, testInfo, 'Platform filter - Linux', async () => {
      await linuxOption.click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // TODO: Replace with a fixed label ID for test stability
  test('Label filter - first available', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/hosts/manage');
    await expect(tableRow(page)).toBeVisible();

    await page.locator('.label-filter-select__control').click();
    // Pick the first custom label (skip platform labels like macOS, Windows, Linux)
    const platforms = ['macOS', 'Windows', 'Linux', 'ChromeOS', 'iOS', 'iPadOS', 'Android'];
    const allOptions = page.locator('.label-filter-select__option');
    const count = await allOptions.count();
    let labelOption = allOptions.first();
    for (let i = 0; i < count; i++) {
      const text = await allOptions.nth(i).textContent();
      if (text && !platforms.some((p) => text.includes(p))) {
        labelOption = allOptions.nth(i);
        break;
      }
    }
    await expect(labelOption).toBeVisible();

    await measureNav(page, testInfo, 'Label filter', async () => {
      await labelOption.click();
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
