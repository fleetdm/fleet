import { Page, Locator, TestInfo } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const RESULTS_DIR = path.resolve(__dirname, '../.perf-results');

export interface PerfResult {
  section: string;
  label: string;
  elapsed: string;
  ms: number;
}

export function formatElapsed(ms: number): string {
  const s = Math.floor(ms / 1000);
  const remaining = String(ms % 1000).padStart(3, '0');
  return `${s}.${remaining}s`;
}

function saveResult(testInfo: TestInfo, label: string, ms: number): void {
  const elapsed = formatElapsed(ms);
  testInfo.annotations.push({ type: 'load-time', description: `${label} - ${elapsed}` });

  const section = testInfo.titlePath[1].replace(' load times', '');
  const result: PerfResult = { section, label, elapsed, ms };

  fs.mkdirSync(RESULTS_DIR, { recursive: true });
  fs.writeFileSync(
    path.join(RESULTS_DIR, `${Date.now()}-${Math.random().toString(36).slice(2)}.json`),
    JSON.stringify(result),
  );
}

/**
 * Measure the time from initiating a navigation until the target content is visible.
 *
 * Timing: starts immediately before the `navigate` callback runs, ends when the
 * callback's last `expect` resolves. This captures the full user-perceived load:
 * network request + server response + JS parse + React render + API fetch + data render.
 *
 * Uses Date.now() on the Node side because page.goto() destroys the browser's
 * JS context (and any performance.mark() placed before it). The ~50ms of Playwright
 * command overhead is negligible for page loads that take 1-15s under load.
 */
export async function measureNav(
  page: Page,
  testInfo: TestInfo,
  label: string,
  navigate: () => Promise<void>,
): Promise<void> {
  const start = Date.now();
  await navigate();
  const ms = Date.now() - start;
  saveResult(testInfo, label, ms);
}

/**
 * Measure search performance: from typing a query until results render.
 *
 * Timing: starts when the debounced API request fires (not when the user types),
 * ends when the `waitFor` callback's assertion resolves. This isolates the
 * server response + render time from the client-side debounce delay.
 */
export async function measureSearch(
  page: Page,
  testInfo: TestInfo,
  label: string,
  input: Locator,
  query: string,
  waitFor: () => Promise<void>,
  { urlPattern = 'query=' } = {},
): Promise<void> {
  const requestFired = page.waitForRequest((req) => req.url().includes(urlPattern));
  const responseDone = page.waitForResponse(
    (res) => res.url().includes(urlPattern) && res.status() === 200,
  );

  await input.fill(query);

  // Timing starts when the debounced request actually fires
  await requestFired;
  const start = Date.now();

  // Wait for response, then verify rendered content
  await responseDone;
  await waitFor();
  const ms = Date.now() - start;

  saveResult(testInfo, label, ms);
}
