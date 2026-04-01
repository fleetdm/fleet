import { Page, TestInfo } from '@playwright/test';
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

export async function measureNav(
  page: Page,
  testInfo: TestInfo,
  label: string,
  navigate: () => Promise<void>
): Promise<void> {
  const start = Date.now();
  await navigate();
  const ms = Date.now() - start;
  const elapsed = formatElapsed(ms);

  testInfo.annotations.push({ type: 'load-time', description: `${label} - ${elapsed}` });

  const section = testInfo.titlePath[1].replace(' load times', '');
  const result: PerfResult = { section, label, elapsed, ms };

  fs.mkdirSync(RESULTS_DIR, { recursive: true });
  fs.writeFileSync(
    path.join(RESULTS_DIR, `${start}-${Math.random().toString(36).slice(2)}.json`),
    JSON.stringify(result)
  );
}
