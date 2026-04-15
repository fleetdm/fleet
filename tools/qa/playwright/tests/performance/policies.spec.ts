import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRow, tableOrEmpty, selectTeam } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Policies load times', () => {
  // ── All fleets ──────────────────────────────────────────────────────────────
  test('All fleets', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'All fleets', async () => {
      await page.goto('/policies/manage');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('All fleets - Other automation filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'All fleets - Other filter', async () => {
      await page.goto('/policies/manage?automation_type=other');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // ── Specific team ───────────────────────────────────────────────────────────
  test('Team page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/policies/manage');
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Team page', async () => {
      await selectTeam(page);
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Team - Software automation filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/policies/manage');
    await expect(tableRow(page)).toBeVisible();
    await selectTeam(page);
    await expect(tableOrEmpty(page)).toBeVisible();

    const currentUrl = page.url();
    const url = new URL(currentUrl);
    url.searchParams.set('automation_type', 'software');

    await measureNav(page, testInfo, 'Team - Software filter', async () => {
      await page.goto(url.pathname + url.search);
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });
});
