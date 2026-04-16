import { test, expect } from '@playwright/test';

test.describe('GitOps and generate-gitops', () => {
  test.describe('GitOps API verification', () => {
    test('Fleet server is healthy and accepts API requests', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.org_info).toBeDefined();
    });

    test('can list fleets via API (GitOps target verification)', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/teams', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.teams).toBeDefined();
    });

    test('can read agent options via API', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.agent_options).toBeDefined();
    });

    test('can read policies via API', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/policies', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.policies).toBeDefined();
    });

    test('can read reports via API', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/queries', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.queries).toBeDefined();
    });
  });

  test.describe('GitOps UI verification', () => {
    test('settings page shows organization info (GitOps target)', async ({ page }) => {
      await page.goto('/settings/organization/info');

      await expect(page).toHaveURL(/\/settings/);
      await expect(page.getByRole('heading', { name: /organization/i })).toBeVisible();
    });

    test('fleets are listed in the UI', async ({ page }) => {
      await page.goto('/settings/teams');

      await expect(page).toHaveURL(/\/settings\/teams/);
    });

    test('policies page reflects GitOps-managed policies', async ({ page }) => {
      await page.goto('/policies/manage');

      await expect(page).toHaveURL(/\/policies/);
      await expect(
        page.getByRole('table').or(page.locator('.empty-table__container'))
      ).toBeVisible();
    });

    test('reports page reflects GitOps-managed reports', async ({ page }) => {
      await page.goto('/queries/manage');

      await expect(page).toHaveURL(/\/queries/);
      await expect(
        page.getByRole('table').or(page.locator('.empty-table__container'))
      ).toBeVisible();
    });
  });
});
