import { test as setup } from '@playwright/test';
import * as path from 'path';
import { loginAsAdmin } from '../helpers/auth';

setup('authenticate as admin', async ({ page }) => {
  await loginAsAdmin(page, process.env.FLEET_ADMIN_EMAIL!, process.env.FLEET_ADMIN_PASSWORD!);
  await page.context().storageState({ path: path.resolve(__dirname, '../.auth/e2e-admin.json') });
});
