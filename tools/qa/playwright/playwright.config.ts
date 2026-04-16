import { defineConfig, devices } from '@playwright/test';
import * as dotenv from 'dotenv';
import * as path from 'path';

const suite = process.env.SUITE || 'e2e';
dotenv.config({ path: path.resolve(__dirname, suite === 'loadtest' ? '.env.loadtest' : '.env'), quiet: true });

export default defineConfig({
  globalTeardown: './helpers/perf-teardown.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.FLEET_URL,
    trace: 'on-first-retry',
    ignoreHTTPSErrors: true,
  },

  projects: [
    // ── E2E ────────────────────────────────────────────────────────────────────
    {
      name: 'e2e-setup',
      testDir: './setup',
      testMatch: /e2e\.setup\.ts/,
    },
    {
      name: 'e2e',
      testDir: './tests',
      grepInvert: /@perf/,
      use: {
        ...devices['Desktop Chrome'],
        storageState: '.auth/e2e-admin.json',
      },
      dependencies: ['e2e-setup'],
    },

    // ── Loadtest ───────────────────────────────────────────────────────────────
    {
      name: 'loadtest-setup',
      testDir: './setup',
      testMatch: /loadtest\.setup\.ts/,
    },
    {
      name: 'loadtest',
      testDir: './tests',
      grep: /@loadtest/,
      timeout: 60000,
      use: {
        ...devices['Desktop Chrome'],
        storageState: '.auth/loadtest-admin.json',
      },
      expect: { timeout: 30000 },
      dependencies: ['loadtest-setup'],
      retries: 0,
    },
  ],
});
