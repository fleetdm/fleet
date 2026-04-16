import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRowWithContent, tableOrEmpty } from '../../helpers/nav';
import { applyVulnerableFilter } from '../../helpers/vuln';

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
      // Vitals: wait for actual vital values to render (present on every host)
      await expect(page.getByText('Disk space available')).toBeVisible();
      await expect(page.getByText('Operating system')).toBeVisible();
      // Activity: wait for at least one activity item with a timestamp
      await expect(page.getByText(/ago/).first()).toBeVisible();
    });
  });

  // ── Software tab ────────────────────────────────────────────────────────────
  test('Software inventory', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);

    await measureNav(page, testInfo, 'Software inventory', async () => {
      await page.getByRole('tab', { name: 'Software' }).click();
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  test('Software - Vulnerable filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);
    await page.getByRole('tab', { name: 'Software' }).click();
    await expect(tableRowWithContent(page)).toBeVisible();

    await measureNav(page, testInfo, 'Vulnerable filter', async () => {
      await applyVulnerableFilter(page);
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Software - Library view', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);
    await page.getByRole('tab', { name: 'Software' }).click();
    await expect(tableRowWithContent(page)).toBeVisible();

    await measureNav(page, testInfo, 'Library view', async () => {
      await page.getByRole('tab', { name: 'Library' }).click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // ── Reports tab ─────────────────────────────────────────────────────────────
  test('Reports tab', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto(hostDetailPath);

    await measureNav(page, testInfo, 'Reports tab', async () => {
      await page.getByRole('tab', { name: 'Reports' }).click();
      // Wait for actual report entries to render (not just the count header)
      await expect(page.getByText('Newest results')).toBeVisible();
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
