#!/usr/bin/env node
// Golden file test for the OpenAPI generator.
//
// Regenerates the spec and compares it to the committed snapshot
// (tools/openapi/spec.yml). Exits non-zero if they differ.
//
// Usage:
//   node test/golden.js          # diff against snapshot
//   node test/golden.js --update # regenerate the snapshot
'use strict';

const { execFileSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const SNAPSHOT = path.join(ROOT, 'spec.yml');
const TMP = path.join(ROOT, 'build', 'openapi-test.yml');

const update = process.argv.includes('--update');

// Generate a fresh spec.
fs.mkdirSync(path.dirname(TMP), { recursive: true });
execFileSync(process.execPath, [
  path.join(ROOT, 'src', 'index.js'),
  '--out', TMP,
], { stdio: 'inherit' });

if (update) {
  fs.copyFileSync(TMP, SNAPSHOT);
  process.stdout.write(`snapshot updated: ${SNAPSHOT}\n`);
  process.exit(0);
}

if (!fs.existsSync(SNAPSHOT)) {
  process.stderr.write(
    `error: snapshot not found at ${SNAPSHOT}\n` +
    `Run "npm test -- --update" to create it.\n`,
  );
  process.exit(1);
}

const expected = fs.readFileSync(SNAPSHOT, 'utf8');
const actual = fs.readFileSync(TMP, 'utf8');

if (expected === actual) {
  process.stdout.write('golden file test passed — spec matches snapshot.\n');
  process.exit(0);
}

// Show a useful diff.
process.stderr.write('golden file test FAILED — spec differs from snapshot.\n\n');
try {
  execFileSync('diff', ['--unified', SNAPSHOT, TMP], { stdio: 'inherit' });
} catch {
  // diff exits non-zero when files differ; that's expected.
}
process.stderr.write(
  '\nIf the changes are intentional, run "npm test -- --update" to accept them.\n',
);
process.exit(1);
