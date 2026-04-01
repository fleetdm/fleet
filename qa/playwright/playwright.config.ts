import { defineConfig, devices } from '@playwright/test';
import * as dotenv from 'dotenv';
import * as path from 'path';

dotenv.config({ path: path.resolve(__dirname, '.env') });

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.FLEET_URL,
    trace: 'on-first-retry',
  },

  projects: [
    // Setup project runs global-setup once before all tests
    {
      name: 'setup',
      testMatch: /global-setup\.ts/,
    },
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Tests that need admin session use this stored state
        storageState: '.auth/admin.json',
      },
      dependencies: ['setup'],
    },
  ],

  globalSetup: undefined, // handled via setup project above
});
