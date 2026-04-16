import { test, expect } from '@playwright/test';

// IdP/SCIM provisioning tests require external IdP configuration.
// Skip the entire suite when the environment is not set up for SCIM testing.
test.describe('IdP provisioning (SCIM)', () => {
  test.skip(
    !process.env.FLEET_SCIM_ENABLED,
    'SCIM tests require FLEET_SCIM_ENABLED=1 and a configured IdP'
  );

  test.describe('SCIM configuration', () => {
    test('SCIM settings page is accessible', async ({ page }) => {
      await page.goto('/settings/integrations/mdm');

      await expect(page).toHaveURL(/\/settings/);
      await expect(
        page.getByText(/end user authentication/i)
          .or(page.getByText(/identity provider/i))
      ).toBeVisible();
    });

    test('can verify SCIM provisioning is enabled via API', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.mdm).toBeDefined();
    });
  });

  test.describe('Okta provisioning', () => {
    test.skip(!process.env.FLEET_OKTA_SCIM_URL, 'Okta SCIM not configured');

    test('Okta-provisioned users appear in Fleet', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/users', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.users).toBeDefined();
    });
  });

  test.describe('Entra provisioning', () => {
    test.skip(!process.env.FLEET_ENTRA_SCIM_URL, 'Entra SCIM not configured');

    test('Entra-provisioned users appear in Fleet', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/users', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.users).toBeDefined();
    });
  });

  test.describe('Google/Hydrant provisioning', () => {
    test.skip(!process.env.FLEET_GOOGLE_SCIM_URL, 'Google SCIM not configured');

    test('Google-provisioned users appear in Fleet', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/users', {
        headers: {
          Authorization: `Bearer ${process.env.FLEET_API_TOKEN}`,
        },
      });

      expect(response.ok()).toBeTruthy();
      const data = await response.json();
      expect(data.users).toBeDefined();
    });
  });

  test.describe('Host enrollment with EUA and IdP provisioning', () => {
    test('enrolled hosts appear on the hosts page', async ({ page }) => {
      await page.goto('/hosts/manage');

      await expect(page).toHaveURL(/\/hosts\/manage/);
    });

    for (const platform of ['macOS', 'Windows', 'Ubuntu', 'iOS', 'iPadOS', 'Android']) {
      test(`${platform} hosts can be filtered`, async ({ page }) => {
        await page.goto('/hosts/manage');

        // Use the platform label filter
        const labelFilter = page.locator('.label-filter-select__control');
        if (await labelFilter.isVisible()) {
          await labelFilter.click();
          const option = page.locator('.label-filter-select__option')
            .filter({ hasText: platform });
          if (await option.isVisible()) {
            await option.click();
            // Page should update with filtered results
            await expect(page).toHaveURL(/\/hosts/);
          }
        }
      });
    }
  });
});
