import { test, expect } from '@playwright/test';
import { tableRow } from '../../../helpers/nav';

test.describe('Fleet Free', () => {
  test.skip(
    process.env.FLEET_LICENSE === 'premium',
    'Fleet Free tests should run on a free-tier server only'
  );

  test.describe('Free features work correctly', () => {
    test('dashboard loads on Fleet Free', async ({ page }) => {
      await page.goto('/dashboard');

      await expect(page).toHaveURL(/\/dashboard/);
    });

    test('hosts page is accessible', async ({ page }) => {
      await page.goto('/hosts/manage');

      await expect(page).toHaveURL(/\/hosts\/manage/);
      await expect(
        tableRow(page).or(page.locator('.empty-table__container'))
      ).toBeVisible();
    });

    test('reports page is accessible', async ({ page }) => {
      await page.goto('/queries/manage');

      await expect(page).toHaveURL(/\/queries/);
      await expect(
        tableRow(page).or(page.locator('.empty-table__container'))
      ).toBeVisible();
    });

    test('policies page is accessible', async ({ page }) => {
      await page.goto('/policies/manage');

      await expect(page).toHaveURL(/\/policies/);
      await expect(
        tableRow(page).or(page.locator('.empty-table__container'))
      ).toBeVisible();
    });

    test('packs page is accessible on Fleet Free', async ({ page }) => {
      await page.goto('/packs/manage');

      await expect(page).toHaveURL(/\/packs/);
      await expect(page.getByRole('heading', { name: /packs/i })).toBeVisible();
    });

    test('settings page is accessible on Fleet Free', async ({ page }) => {
      await page.goto('/settings/organization/info');

      await expect(page).toHaveURL(/\/settings/);
      await expect(page.getByRole('heading', { name: /organization/i })).toBeVisible();
    });
  });

  test.describe('Premium features are restricted', () => {
    test('IdP/SCIM settings are not available on Fleet Free', async ({ page }) => {
      await page.goto('/settings/integrations');

      // Premium-only IdP provisioning options should not be visible
      await expect(page.getByText(/SCIM/i)).not.toBeVisible();
    });
  });

  test.describe('GitOps works on Fleet Free', () => {
    test('can read config via API on Fleet Free', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.org_info).toBeDefined();
    });

    test('can list policies via API on Fleet Free', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/policies', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
    });

    test('can list reports via API on Fleet Free', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/queries', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
    });
  });

  test.describe('No errors on free-only workflows', () => {
    test('no console errors on dashboard', async ({ page }) => {
      const errors: string[] = [];
      page.on('console', (msg) => {
        if (msg.type() === 'error') errors.push(msg.text());
      });

      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');

      // Filter out known benign errors (e.g., favicon 404s)
      const realErrors = errors.filter(
        (e) => !e.includes('favicon') && !e.includes('net::ERR')
      );
      expect(realErrors).toEqual([]);
    });

    test('no console errors on hosts page', async ({ page }) => {
      const errors: string[] = [];
      page.on('console', (msg) => {
        if (msg.type() === 'error') errors.push(msg.text());
      });

      await page.goto('/hosts/manage');
      await page.waitForLoadState('networkidle');

      const realErrors = errors.filter(
        (e) => !e.includes('favicon') && !e.includes('net::ERR')
      );
      expect(realErrors).toEqual([]);
    });
  });
});
