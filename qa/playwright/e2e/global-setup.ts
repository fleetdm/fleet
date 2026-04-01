import { chromium, expect } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';

const AUTH_DIR = path.resolve(__dirname, '../.auth');

async function saveSession(email: string, password: string, outFile: string) {
  const browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();

  await page.goto('/login');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill(password);
  await page.getByRole('button', { name: 'Log in' }).click();
  await expect(page).not.toHaveURL(/\/login/);

  await context.storageState({ path: outFile });
  await browser.close();
}

export default async function globalSetup() {
  fs.mkdirSync(AUTH_DIR, { recursive: true });

  const adminEmail = process.env.FLEET_ADMIN_EMAIL!;
  const adminPassword = process.env.FLEET_ADMIN_PASSWORD!;

  if (!adminEmail || !adminPassword) {
    throw new Error('FLEET_ADMIN_EMAIL and FLEET_ADMIN_PASSWORD must be set in .env');
  }

  await saveSession(adminEmail, adminPassword, path.join(AUTH_DIR, 'admin.json'));

  // Add more roles here as needed, e.g.:
  // await saveSession(observerEmail, observerPassword, path.join(AUTH_DIR, 'observer.json'));
}
