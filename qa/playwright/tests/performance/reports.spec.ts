import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

test.describe('Reports load times', () => {
  // ── All fleets ──────────────────────────────────────────────────────────────
  test('All fleets', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'All fleets', async () => {
      await page.goto('/reports/manage');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('All fleets - platform filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'All fleets - platform filter', async () => {
      await page.goto('/reports/manage?platform=darwin');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('All fleets - search', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();

    const reportName = await tableRow(page).locator('td').nth(1).innerText();

    await measureSearch(
      page, testInfo, 'All fleets - search',
      page.getByPlaceholder('Search by name'), reportName!.trim(),
      async () => { await expect(page.getByRole('table').getByText(reportName!.trim()).first()).toBeVisible(); }
    );
  });

  // ── Specific team ───────────────────────────────────────────────────────────
  test('Team page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Team page', async () => {
      await page.locator('.team-dropdown__control').click();
      await page.locator('.team-dropdown__option').nth(1).click();
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Team - platform filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();
    await page.locator('.team-dropdown__control').click();
    await page.locator('.team-dropdown__option').nth(1).click();
    await page.waitForURL(/fleet_id/);
    await expect(tableRow(page)).toBeVisible();

    const currentUrl = page.url();
    const url = new URL(currentUrl);
    url.searchParams.set('platform', 'darwin');

    await measureNav(page, testInfo, 'Team - platform filter', async () => {
      await page.goto(url.pathname + url.search);
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Team - search', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();
    await page.locator('.team-dropdown__control').click();
    await page.locator('.team-dropdown__option').nth(1).click();
    await page.waitForURL(/fleet_id/);
    await expect(tableRow(page)).toBeVisible();

    // Get just the report name text, not the "Inherited" badge suffix
    const reportName = await tableRow(page).locator('td').nth(1).locator('.data-table__tooltip-truncated-text').innerText();

    await measureSearch(
      page, testInfo, 'Team - search',
      page.getByPlaceholder('Search by name'), reportName!.trim(),
      async () => { await expect(page.getByRole('table').getByText(reportName!.trim()).first()).toBeVisible(); }
    );
  });
});
