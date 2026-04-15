import { test, expect } from '@playwright/test';

// Validation tests always start with a fresh session
test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Login validation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('shows validation errors when both fields are empty', async ({ page }) => {
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Email field must be completed')).toBeVisible();
    await expect(page.getByText('Password field must be completed')).toBeVisible();
  });

  test('shows validation error when email is empty', async ({ page }) => {
    await page.getByPlaceholder('Password').fill('SomePassword123!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Email field must be completed')).toBeVisible();
  });

  test('shows validation error when password is empty', async ({ page }) => {
    await page.getByPlaceholder('Email').fill(process.env.FLEET_ADMIN_EMAIL!);
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Password field must be completed')).toBeVisible();
  });

  test('shows validation error for invalid email format', async ({ page }) => {
    await page.getByPlaceholder('Email').fill('notanemail');
    await page.getByPlaceholder('Password').fill('SomePassword123!');
    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page.getByText('Email must be a valid email address')).toBeVisible();
  });
});
