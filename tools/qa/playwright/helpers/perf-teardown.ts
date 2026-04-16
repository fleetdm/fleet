import * as fs from 'fs';
import * as path from 'path';
import { PerfResult, formatElapsed } from './perf';

const RESULTS_DIR = path.resolve(__dirname, '../.perf-results');
const HISTORY_DIR = path.resolve(__dirname, '../.perf-history');
const MAX_HISTORY_RUNS = 10;
const COMPARE_RUNS = 3;

const SECTION_ORDER = [
  'Dashboard',
  'Hosts',
  'Host Details',
  'Controls',
  'Software',
  'Software OS',
  'Vulnerabilities',
  'Reports',
  'Policies',
  'Labels',
  'Users',
];

const supportsColor = process.stdout.isTTY && !process.env.NO_COLOR;

function c(code: string, text: string): string {
  return supportsColor ? `\x1b[${code}m${text}\x1b[0m` : text;
}

/** Color the current run time by severity. */
function colorCurrent(elapsed: string, ms: number): string {
  if (ms > 15000) return c('31', elapsed); // red
  if (ms > 5000) return c('33', elapsed);  // yellow
  return elapsed;                           // no color — normal
}

/**
 * Color a previous run's time based on whether current got better or worse.
 * Green = current improved (previous was slower).
 * Yellow = current regressed (previous was faster).
 */
function colorPrevious(prevMs: number, currentMs: number): string {
  const prevElapsed = formatElapsed(prevMs);
  const delta = currentMs - prevMs;
  if (Math.abs(delta) < 200) return c('90', prevElapsed);  // gray — negligible
  if (delta < 0) return c('32', prevElapsed);               // green — current is faster
  return c('33', prevElapsed);                               // yellow — current is slower
}

function sortSections(sections: string[]): string[] {
  return [...sections].sort((a, b) => {
    const ai = SECTION_ORDER.indexOf(a);
    const bi = SECTION_ORDER.indexOf(b);
    if (ai !== -1 && bi !== -1) return ai - bi;
    if (ai !== -1) return -1;
    if (bi !== -1) return 1;
    return a.localeCompare(b);
  });
}

export default async function globalTeardown() {
  if (!fs.existsSync(RESULTS_DIR)) return;

  const files = fs.readdirSync(RESULTS_DIR).filter((f) => f.endsWith('.json'));
  if (files.length === 0) return;

  // ── Collect current run results ────────────────────────────────────────────
  const currentResults: PerfResult[] = files
    .sort()
    .map((f) => JSON.parse(fs.readFileSync(path.join(RESULTS_DIR, f), 'utf-8')));

  fs.rmSync(RESULTS_DIR, { recursive: true });

  // ── Save to history ────────────────────────────────────────────────────────
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  const timestamp = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;

  const runDir = path.join(HISTORY_DIR, timestamp);
  fs.mkdirSync(runDir, { recursive: true });
  fs.writeFileSync(path.join(runDir, 'results.json'), JSON.stringify(currentResults, null, 2));

  // ── Load previous runs for comparison ──────────────────────────────────────
  const allRuns = fs
    .readdirSync(HISTORY_DIR)
    .filter((d) => fs.statSync(path.join(HISTORY_DIR, d)).isDirectory())
    .sort();

  const previousRunDirs = allRuns.filter((d) => d !== timestamp).slice(-COMPARE_RUNS);

  const previousRuns: { name: string; results: Map<string, number> }[] = [];
  for (const dir of previousRunDirs) {
    const resultsPath = path.join(HISTORY_DIR, dir, 'results.json');
    if (!fs.existsSync(resultsPath)) continue;
    const results: PerfResult[] = JSON.parse(fs.readFileSync(resultsPath, 'utf-8'));
    const map = new Map<string, number>();
    for (const r of results) map.set(`${r.section}|${r.label}`, r.ms);
    previousRuns.push({ name: dir, results: map });
  }

  // ── Prune old runs beyond MAX_HISTORY_RUNS ─────────────────────────────────
  if (allRuns.length > MAX_HISTORY_RUNS) {
    const toDelete = allRuns.slice(0, allRuns.length - MAX_HISTORY_RUNS);
    for (const dir of toDelete) {
      fs.rmSync(path.join(HISTORY_DIR, dir), { recursive: true });
    }
  }

  // ── Build grouped results ──────────────────────────────────────────────────
  const grouped = new Map<string, PerfResult[]>();
  for (const result of currentResults) {
    if (!grouped.has(result.section)) grouped.set(result.section, []);
    grouped.get(result.section)!.push(result);
  }

  const sections = sortSections([...grouped.keys()]);

  // ── Print summary table ────────────────────────────────────────────────────
  const labelW = Math.max(...currentResults.map((r) => r.label.length), 4) + 2;
  const sectionW = Math.max(...sections.map((s) => s.length), 7) + 2;
  const colW = 12;
  const hasPrev = previousRuns.length > 0;
  const prevHeaders = previousRuns.map((_, i) => `prev-${i + 1}`);

  const totalW = sectionW + labelW + colW + (hasPrev ? colW * previousRuns.length : 0) + 2;

  console.log('\n' + '─'.repeat(totalW));
  console.log(' Performance Summary');
  console.log('─'.repeat(totalW));

  let header = ` ${'Section'.padEnd(sectionW)}${'Page'.padEnd(labelW)}${'Current'.padEnd(colW)}`;
  for (const h of prevHeaders) header += h.padStart(colW);
  console.log(header);
  console.log('─'.repeat(totalW));

  for (const section of sections) {
    const entries = grouped.get(section)!;
    for (let i = 0; i < entries.length; i++) {
      const e = entries[i];
      const sectionCol = i === 0 ? section : '';

      // Current column — color by absolute severity
      const currentCol = colorCurrent(e.elapsed, e.ms);
      // Pad accounting for ANSI escape codes (9 chars per color sequence)
      const currentHasColor = currentCol !== e.elapsed;
      let line = ` ${sectionCol.padEnd(sectionW)}${e.label.padEnd(labelW)}${currentCol.padEnd(colW + (currentHasColor ? 9 : 0))}`;

      // Previous columns — color by comparison to current
      for (const prev of previousRuns) {
        const prevMs = prev.results.get(`${e.section}|${e.label}`);
        if (prevMs !== undefined) {
          const colored = colorPrevious(prevMs, e.ms);
          const prevElapsed = formatElapsed(prevMs);
          const prevHasColor = colored !== prevElapsed;
          line += colored.padStart(colW + (prevHasColor ? 9 : 0));
        } else {
          line += c('90', '—').padStart(colW + (supportsColor ? 9 : 0));
        }
      }

      console.log(line);
    }
  }

  console.log('─'.repeat(totalW));
  if (hasPrev) {
    console.log(c('90', ` ${previousRuns.length} previous run(s) | ${c('32', 'green')}${c('90', ' = current faster | ')}${c('33', 'yellow')}${c('90', ' = current slower | History: ')}${allRuns.length <= MAX_HISTORY_RUNS ? allRuns.length : MAX_HISTORY_RUNS}${c('90', `/${MAX_HISTORY_RUNS} stored`)}`));
  } else {
    console.log(c('90', ' First run — no history to compare against.'));
  }
  console.log('');
}
