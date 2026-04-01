import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';

test.describe.configure({ mode: 'serial' });

const tableRow = (page: import('@playwright/test').Page) =>
  page.getByRole('table').locator('tbody').getByRole('row').first();

/** Resolves to the first match of either populated content or empty state */
const contentOrEmpty = (page: import('@playwright/test').Page, populated: string, empty: string) =>
  page.locator(populated).or(page.locator(empty)).first();

test.describe('Controls load times', () => {
  // ── OS Updates ──────────────────────────────────────────────────────────────
  test('OS Updates', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'OS Updates', async () => {
      await page.goto('/controls/os-updates');
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('OS Updates - View hosts for top OS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/controls/os-updates');
    await expect(tableRow(page)).toBeVisible();

    const viewHostsLink = page.locator('.os-hosts-link').first();

    await measureNav(page, testInfo, 'OS Updates - top OS hosts', async () => {
      await viewHostsLink.click();
      await expect(tableRow(page)).toBeVisible();
    });
  });

  // ── OS Settings ─────────────────────────────────────────────────────────────
  test('OS Settings - Top status hosts', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/controls/os-settings');
    await expect(page.locator('.profile-status-aggregate__profile-status-count').first()).toBeVisible();

    const statusCard = page.locator('.profile-status-aggregate__profile-status-count').first();

    await measureNav(page, testInfo, 'OS Settings - status hosts', async () => {
      await statusCard.click();
      await expect(tableRow(page)).toBeVisible();
    });
  });

  test('Custom settings', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Custom settings', async () => {
      await page.goto('/controls/os-settings/custom-settings');
      await expect(
        contentOrEmpty(page, '.upload-list__list-item', '.add-profile-card, .card.empty-profiles')
      ).toBeVisible();
    });
  });

  test('Certificates', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Certificates', async () => {
      await page.goto('/controls/os-settings/certificates');
      await expect(
        contentOrEmpty(page, '.upload-list__list-item', '.add-cert-card')
      ).toBeVisible();
    });
  });

  // ── Setup Experience ────────────────────────────────────────────────────────
  test('Install software - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - macOS', async () => {
      await page.goto('/controls/setup-experience/install-software/macos');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  test('Install software - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Windows', async () => {
      await page.goto('/controls/setup-experience/install-software/windows');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  test('Install software - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Linux', async () => {
      await page.goto('/controls/setup-experience/install-software/linux');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  test('Install software - iOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - iOS', async () => {
      await page.goto('/controls/setup-experience/install-software/ios');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  test('Install software - iPadOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - iPadOS', async () => {
      await page.goto('/controls/setup-experience/install-software/ipados');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  test('Install software - Android', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Android', async () => {
      await page.goto('/controls/setup-experience/install-software/android');
      await expect(
        contentOrEmpty(page, 'table tbody tr', '.empty-table__container')
      ).toBeVisible();
    });
  });

  // ── Scripts ─────────────────────────────────────────────────────────────────
  test('Scripts - Library', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Scripts - Library', async () => {
      await page.goto('/controls/scripts/library');
      await expect(
        contentOrEmpty(page, '.upload-list__list-item', '.card.empty-scripts, .script-uploader')
      ).toBeVisible();
    });
  });

  test('Scripts - Batch progress', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Scripts - Batch progress', async () => {
      await page.goto('/controls/scripts/progress');
      await expect(
        contentOrEmpty(page, '.paginated-list__row', '.script-batch-progress__empty')
      ).toBeVisible();
    });
  });

  // ── Variables ───────────────────────────────────────────────────────────────
  test('Variables', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Variables', async () => {
      await page.goto('/controls/variables');
      await expect(
        contentOrEmpty(page, '.paginated-list__row:not(.paginated-list__header)', '.empty-table__container')
      ).toBeVisible();
    });
  });
});
