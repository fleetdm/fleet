import { test, expect } from '@playwright/test';
import { measureNav } from '../../helpers/perf';
import { tableRow, tableRowWithContent, tableOrEmpty } from '../../helpers/nav';

test.describe.configure({ mode: 'serial' });

/**
 * First data list item in the content area. These are <li> elements containing
 * a timestamp ("ago"), which distinguishes them from nav/tab listitems.
 */
function contentListItem(page: import('@playwright/test').Page) {
  return page.getByRole('listitem').filter({ hasText: /ago/ }).first();
}

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

    // "View all hosts" button is only visible on row hover
    const firstRow = page.getByRole('table').locator('tbody tr').first();
    await firstRow.hover();

    await measureNav(page, testInfo, 'OS Updates - top OS hosts', async () => {
      await firstRow.getByRole('button', { name: 'View all hosts' }).click();
      await expect(tableRowWithContent(page)).toBeVisible();
    });
  });

  // ── OS Settings ─────────────────────────────────────────────────────────────
  test('OS Settings - Top status hosts', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await page.goto('/controls/os-settings');

    // Find the first status link with hosts, or fall back to "Verified"
    const statusLinks = page.getByRole('link').filter({ hasText: /hosts$/ });
    const count = await statusLinks.count();
    let targetLink = null;
    let maxHosts = 0;

    for (let i = 0; i < count; i++) {
      const text = await statusLinks.nth(i).innerText();
      const match = text.match(/(\d+)\s+hosts?/);
      const hostCount = match ? parseInt(match[1], 10) : 0;
      if (hostCount > maxHosts) {
        maxHosts = hostCount;
        targetLink = statusLinks.nth(i);
      }
    }
    // Fall back to "Verified" if all are 0
    if (!targetLink || maxHosts === 0) {
      targetLink = statusLinks.filter({ hasText: 'Verified' }).first();
    }

    await measureNav(page, testInfo, 'OS Settings - status hosts', async () => {
      await targetLink!.click();
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Configuration profiles', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Configuration profiles', async () => {
      await page.goto('/controls/os-settings/configuration-profiles');
      await expect(contentListItem(page)).toBeVisible();
    });
  });

  test('Certificates', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Certificates', async () => {
      await page.goto('/controls/os-settings/certificates');
      await expect(contentListItem(page)).toBeVisible();
    });
  });

  // ── Setup Experience ────────────────────────────────────────────────────────
  test('Install software - macOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - macOS', async () => {
      await page.goto('/controls/setup-experience/install-software/macos');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Install software - Windows', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Windows', async () => {
      await page.goto('/controls/setup-experience/install-software/windows');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Install software - Linux', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Linux', async () => {
      await page.goto('/controls/setup-experience/install-software/linux');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Install software - iOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - iOS', async () => {
      await page.goto('/controls/setup-experience/install-software/ios');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Install software - iPadOS', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - iPadOS', async () => {
      await page.goto('/controls/setup-experience/install-software/ipados');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  test('Install software - Android', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Install software - Android', async () => {
      await page.goto('/controls/setup-experience/install-software/android');
      await expect(tableOrEmpty(page)).toBeVisible();
    });
  });

  // ── Scripts ─────────────────────────────────────────────────────────────────
  test('Scripts - Library', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Scripts - Library', async () => {
      await page.goto('/controls/scripts/library');
      await expect(contentListItem(page)).toBeVisible();
    });
  });

  test('Scripts - Batch progress', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Scripts - Batch progress', async () => {
      await page.goto('/controls/scripts/progress');
      // Switch to Finished tab which has completed batch runs
      await page.getByRole('tab', { name: 'Finished' }).click();
      await expect(contentListItem(page)).toBeVisible();
    });
  });

  // ── Variables ───────────────────────────────────────────────────────────────
  test('Variables', { tag: ['@loadtest', '@perf'] }, async ({ page }, testInfo) => {
    await measureNav(page, testInfo, 'Variables', async () => {
      await page.goto('/controls/variables');
      await expect(contentListItem(page)).toBeVisible();
    });
  });
});
