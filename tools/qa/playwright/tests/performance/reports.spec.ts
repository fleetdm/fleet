import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';
import { tableRow, selectTeam, getNameFromRow } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

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

    const reportName = await getNameFromRow(page);

    await measureSearch(
      page, testInfo, 'All fleets - search',
      page.getByPlaceholder('Search by name'), reportName,
      async () => { await expect(page.getByRole('table').getByText(reportName).first()).toBeVisible(); }
    );
  });

  // ── Specific team ───────────────────────────────────────────────────────────
  test('Team page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Team page', async () => {
      await selectTeam(page);
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Team - platform filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/reports/manage');
    await expect(tableRow(page)).toBeVisible();
    await selectTeam(page);
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
    await selectTeam(page);
    await expect(tableRow(page)).toBeVisible();

    const reportName = await getNameFromRow(page);

    await measureSearch(
      page, testInfo, 'Team - search',
      page.getByPlaceholder('Search by name'), reportName,
      async () => { await expect(page.getByRole('table').getByText(reportName).first()).toBeVisible(); }
    );
  });
});
