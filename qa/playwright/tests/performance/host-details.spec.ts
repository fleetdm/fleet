import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRow, tableOrEmpty } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Host Details load times', () => {
  let hostDetailPath: string;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext({
      storageState: '.auth/loadtest-admin.json',
    });
    const page = await context.newPage();
    await page.goto('/hosts/manage?order_key=display_name&order_direction=asc');
    await expect(page.getByRole('table').locator('tbody').getByRole('row').first()).toBeVisible();

    hostDetailPath = await page
      .getByRole('table').locator('tbody').getByRole('row').first()
      .getByRole('link').first()
      .getAttribute('href') ?? '';
    await context.close();
  });

  // ── Details page full load ──────────────────────────────────────────────────
  test('Host details page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Host details page', async () => {
      await page.goto(hostDetailPath);
      await expect(page.locator('.vitals-card__info-grid')).toBeVisible();
      await expect(
        page.locator('.past-activity-feed').first()
          .or(page.locator('.past-activity-feed__empty-feed'))
      ).toBeVisible();
      await expect(
        page.locator('.host-queries-card').getByRole('table').locator('tbody').getByRole('row').first()
          .or(page.locator('.host-queries-card .empty-table__container'))
      ).toBeVisible();
    });
  });

  // ── Software tab ────────────────────────────────────────────────────────────
  test('Software inventory', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);

    await measureNav(page, testInfo, 'Software inventory', async () => {
      await page.getByRole('tab', { name: 'Software' }).click();
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Software - Vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);
    await page.getByRole('tab', { name: 'Software' }).click();
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Vulnerable filter', async () => {
      await page.getByRole('button', { name: /filter/i }).click();
      await page.locator('.software-filters-modal .fleet-slider').click();
      await page.locator('.software-filters-modal').getByRole('button', { name: 'Apply' }).click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Software - Library view', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);
    await page.getByRole('tab', { name: 'Software' }).click();
    await expect(tableRow(page)).toBeVisible();

    await measureNav(page, testInfo, 'Library view', async () => {
      await page.getByRole('tab', { name: 'Library' }).click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // ── Policies tab ────────────────────────────────────────────────────────────
  test('Policies tab', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);

    await measureNav(page, testInfo, 'Policies tab', async () => {
      await page.getByRole('tab', { name: 'Policies' }).click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });
});
