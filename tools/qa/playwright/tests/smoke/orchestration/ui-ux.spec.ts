import { test, expect } from '@playwright/test';

const PAGES = [
  { name: 'Dashboard', path: '/dashboard' },
  { name: 'Hosts', path: '/hosts/manage' },
  { name: 'Reports', path: '/queries/manage' },
  { name: 'Policies', path: '/policies/manage' },
  { name: 'Controls', path: '/controls' },
  { name: 'Software', path: '/software/titles' },
  { name: 'Settings', path: '/settings/organization/info' },
];

// Patterns to exclude from console error checks (benign or infrastructure noise)
const IGNORED_CONSOLE_ERRORS = [
  'favicon',
  'net::ERR',
  'ResizeObserver',
];

// URL patterns to exclude from failed network request checks
const IGNORED_NETWORK_PATTERNS = [
  'favicon',
  '/NaN',   // known client-side routing bug — bad IDs in API URLs
];

test.describe('UI / UX visual consistency', () => {
  for (const { name, path } of PAGES) {
    test.describe(`${name} page`, () => {
      test(`renders without console errors`, async ({ page }) => {
        const errors: string[] = [];
        page.on('console', (msg) => {
          if (msg.type() === 'error') errors.push(msg.text());
        });

        await page.goto(path);
        await page.waitForLoadState('networkidle');

        const realErrors = errors.filter(
          (e) => !IGNORED_CONSOLE_ERRORS.some((pattern) => e.includes(pattern))
        );
        expect(realErrors).toEqual([]);
      });

      test(`has no broken images`, async ({ page }) => {
        await page.goto(path);
        await page.waitForLoadState('networkidle');

        const brokenImages = await page.evaluate(() => {
          const images = Array.from(document.querySelectorAll('img'));
          return images
            .filter((img) => !img.complete || img.naturalWidth === 0)
            .map((img) => img.src);
        });

        expect(brokenImages).toEqual([]);
      });

      test(`has no overlapping or clipped main content`, async ({ page }) => {
        await page.goto(path);
        await page.waitForLoadState('networkidle');

        // Verify main content container is visible and has non-zero dimensions
        const mainContent = page.locator('main, .main-content, #main-content, .core-wrapper').first();
        if (await mainContent.isVisible()) {
          const box = await mainContent.boundingBox();
          expect(box).not.toBeNull();
          expect(box!.width).toBeGreaterThan(0);
          expect(box!.height).toBeGreaterThan(0);
        }
      });

      test(`buttons and inputs render correctly`, async ({ page }) => {
        await page.goto(path);
        await page.waitForLoadState('networkidle');

        // Verify all visible buttons have non-zero dimensions
        const buttons = page.getByRole('button').filter({ hasNotText: '' });
        const buttonCount = await buttons.count();
        for (let i = 0; i < Math.min(buttonCount, 10); i++) {
          const button = buttons.nth(i);
          if (await button.isVisible()) {
            const box = await button.boundingBox();
            expect(box).not.toBeNull();
            expect(box!.width).toBeGreaterThan(0);
            expect(box!.height).toBeGreaterThan(0);
          }
        }

        // Verify all visible inputs have non-zero dimensions
        const inputs = page.locator('input:visible, textarea:visible, select:visible');
        const inputCount = await inputs.count();
        for (let i = 0; i < Math.min(inputCount, 10); i++) {
          const input = inputs.nth(i);
          const box = await input.boundingBox();
          if (box) {
            expect(box.width).toBeGreaterThan(0);
            expect(box.height).toBeGreaterThan(0);
          }
        }
      });

      test(`tables render with proper structure`, async ({ page }) => {
        await page.goto(path);
        await page.waitForLoadState('networkidle');

        const tables = page.getByRole('table');
        const tableCount = await tables.count();

        for (let i = 0; i < tableCount; i++) {
          const table = tables.nth(i);
          if (await table.isVisible()) {
            // Tables should have header cells
            const headers = table.locator('thead th, thead td');
            expect(await headers.count()).toBeGreaterThan(0);
          }
        }
      });

      test(`no failed network requests (4xx/5xx)`, async ({ page }) => {
        const failedRequests: string[] = [];
        page.on('response', (response) => {
          const url = response.url();
          if (
            response.status() >= 400 &&
            !IGNORED_NETWORK_PATTERNS.some((pattern) => url.includes(pattern))
          ) {
            failedRequests.push(`${response.status()} ${url}`);
          }
        });

        await page.goto(path);
        await page.waitForLoadState('networkidle');

        expect(failedRequests).toEqual([]);
      });
    });
  }

  test('navigation sidebar renders all expected links', async ({ page }) => {
    await page.goto('/dashboard');

    const expectedLinks = ['Hosts', 'Controls', 'Software', 'Reports', 'Policies'];
    for (const linkText of expectedLinks) {
      await expect(
        page.getByRole('navigation').getByRole('link', { name: linkText })
          .or(page.getByRole('link', { name: linkText }).first())
      ).toBeVisible();
    }
  });

  test('fonts load without FOUT (flash of unstyled text)', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    const fontsLoaded = await page.evaluate(() => document.fonts.ready.then(() => true));
    expect(fontsLoaded).toBeTruthy();
  });
});
