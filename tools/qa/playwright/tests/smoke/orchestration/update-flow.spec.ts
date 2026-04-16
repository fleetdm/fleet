import { test, expect } from '@playwright/test';
import { tableRow } from '../../../helpers/nav';

test.describe('Update flow', () => {
  test('dashboard loads after upgrade', async ({ page }) => {
    await page.goto('/dashboard');

    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.getByRole('heading').first()).toBeVisible();
  });

  test('hosts page is accessible and lists hosts', async ({ page }) => {
    await page.goto('/hosts/manage');

    await expect(page).toHaveURL(/\/hosts\/manage/);
    await expect(
      tableRow(page).or(page.locator('.empty-table__container'))
    ).toBeVisible();
  });

  test('controls page is accessible', async ({ page }) => {
    await page.goto('/controls');

    await expect(page).toHaveURL(/\/controls/);
    await expect(
      page.locator('.controls-page, .os-updates, .setup-experience, .scripts')
        .or(page.getByRole('heading').first())
    ).toBeVisible();
  });

  test('reports page is accessible', async ({ page }) => {
    await page.goto('/queries/manage');

    await expect(page).toHaveURL(/\/reports|\/queries/);
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

  test('settings page is accessible', async ({ page }) => {
    await page.goto('/settings/organization/info');

    await expect(page).toHaveURL(/\/settings/);
    await expect(page.getByRole('heading', { name: /organization/i })).toBeVisible();
  });

  test('previously created hosts still exist after upgrade', async ({ page }) => {
    await page.goto('/hosts/manage');

    await expect(tableRow(page)).toBeVisible({ timeout: 10_000 });
  });

  test('previously created reports still exist after upgrade', async ({ page }) => {
    await page.goto('/queries/manage');

    await expect(tableRow(page)).toBeVisible({ timeout: 10_000 });
  });
});
