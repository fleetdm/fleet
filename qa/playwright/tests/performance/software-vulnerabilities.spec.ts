import { test, expect } from '@playwright/test';
import { measureNav, measureSearch } from '../../helpers/perf';
import { tableRow, getNameFromRow } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

test.describe('Vulnerabilities load times', () => {
  test('Vulnerabilities page', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Vulnerabilities page', async () => {
      await page.goto('/software/vulnerabilities');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Filters ─────────────────────────────────────────────────────────────────
  test('Exploited vulnerabilities filter', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Exploited filter', async () => {
      await page.goto('/software/vulnerabilities?exploit=true');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Sorting ─────────────────────────────────────────────────────────────────
  test('Sort severity ascending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Severity ascending', async () => {
      await page.goto('/software/vulnerabilities?order_key=cvss_score&order_direction=asc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Sort severity descending', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Severity descending', async () => {
      await page.goto('/software/vulnerabilities?order_key=cvss_score&order_direction=desc');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── Search ───────────────────────────────────────────────────────────────────
  test('Search CVE', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/software/vulnerabilities');
    await expect(tableRow(page)).toBeVisible();

    const cveName = await getNameFromRow(page, 0);

    await measureSearch(
      page, testInfo, 'Search CVE',
      page.getByPlaceholder('Search by CVE'), cveName,
      async () => { await expect(page.getByRole('table').getByText(cveName).first()).toBeVisible(); }
    );
  });
});
