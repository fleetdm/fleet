import * as fs from 'fs';
import * as path from 'path';
import { PerfResult } from './perf';

const RESULTS_DIR = path.resolve(__dirname, '../.perf-results');

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

function colorize(elapsed: string, ms: number): string {
  if (!supportsColor) return elapsed;
  if (ms > 15000) return `\x1b[31m${elapsed}\x1b[0m`; // red
  if (ms > 5000)  return `\x1b[33m${elapsed}\x1b[0m`; // orange
  return elapsed;
}

export default async function globalTeardown() {
  if (!fs.existsSync(RESULTS_DIR)) return;

  const files = fs.readdirSync(RESULTS_DIR).filter((f) => f.endsWith('.json'));
  if (files.length === 0) return;

  const results: PerfResult[] = files
    .sort()
    .map((f) => JSON.parse(fs.readFileSync(path.join(RESULTS_DIR, f), 'utf-8')));

  fs.rmSync(RESULTS_DIR, { recursive: true });

  const grouped = new Map<string, PerfResult[]>();
  for (const result of results) {
    if (!grouped.has(result.section)) grouped.set(result.section, []);
    grouped.get(result.section)!.push(result);
  }

  const sections = [...grouped.keys()].sort((a, b) => {
    const ai = SECTION_ORDER.indexOf(a);
    const bi = SECTION_ORDER.indexOf(b);
    if (ai !== -1 && bi !== -1) return ai - bi;
    if (ai !== -1) return -1;
    if (bi !== -1) return 1;
    return a.localeCompare(b);
  });

  const labelW = Math.max(...results.map((r) => r.label.length)) + 2;
  const sectionW = Math.max(...sections.map((s) => s.length)) + 2;
  const totalW = sectionW + labelW + 12;

  console.log('\n' + '─'.repeat(totalW));
  console.log(' Performance Summary');
  console.log('─'.repeat(totalW));
  console.log(` ${'Section'.padEnd(sectionW)}${'Page'.padEnd(labelW)}Load Time`);
  console.log('─'.repeat(totalW));

  for (const section of sections) {
    const entries = grouped.get(section)!;
    for (let i = 0; i < entries.length; i++) {
      const sectionCol = i === 0 ? section : '';
      console.log(` ${sectionCol.padEnd(sectionW)}${entries[i].label.padEnd(labelW)}${colorize(entries[i].elapsed, entries[i].ms)}`);
    }
  }

  console.log('─'.repeat(totalW) + '\n');
}
