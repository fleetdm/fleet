'use strict';

const test = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const path = require('node:path');

const SCRIPT = path.join(__dirname, 'build-fleetie-handles.js');

function runScript() {
  // Force handbook-only mode so the test is hermetic (no network calls, no token required).
  const env = { ...process.env, READ_ORG_TOKEN: '' };
  delete env.FLEETIE_HANDLES_OUT;
  const out = execFileSync('node', [SCRIPT], {
    env,
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  });
  return out.split('\n').filter(Boolean);
}

let handlesCache = null;
function getHandles() {
  if (!handlesCache) handlesCache = runScript();
  return handlesCache;
}

test('produces a sane number of handles', () => {
  const handles = getHandles();
  assert.ok(handles.length >= 50, `expected at least 50 handles, got ${handles.length}`);
});

test('output is sorted, lowercased, and deduplicated', () => {
  const handles = getHandles();
  const sorted = [...handles].sort();
  assert.deepStrictEqual(handles, sorted, 'output is not sorted');
  for (const handle of handles) {
    assert.strictEqual(handle, handle.toLowerCase(), `handle not lowercased: ${handle}`);
  }
  assert.strictEqual(new Set(handles).size, handles.length, 'output contains duplicates');
});

test('includes known current Fleeties', () => {
  const set = new Set(getHandles());
  for (const handle of ['getvictor', 'lukeheath', 'mikermcneil', 'noahtalerman', 'eashaw']) {
    assert.ok(set.has(handle), `expected current Fleetie ${handle} in handle list`);
  }
});

test('includes known former Fleeties from git history', () => {
  const set = new Set(getHandles());
  // These handles have all appeared in handbook/company/product-groups.md at some point in git history
  // but are not in the file at HEAD.
  for (const handle of ['iansltx', 'mna', 'roperzh', 'ghernandez345']) {
    assert.ok(set.has(handle), `expected former Fleetie ${handle} in handle list`);
  }
});

test('excludes denylisted path segments and the org handle', () => {
  const set = new Set(getHandles());
  for (const denied of ['fleetdm', 'todo', 'user-attachments', 'orgs', 'issues', 'pull', 'apps']) {
    assert.ok(!set.has(denied), `expected ${denied} to be filtered out`);
  }
});

test('all handles match the GitHub username format', () => {
  const handles = getHandles();
  const githubHandleRe = /^[a-z0-9][a-z0-9-]{0,38}$/;
  for (const handle of handles) {
    assert.ok(githubHandleRe.test(handle), `invalid handle in output: ${handle}`);
    assert.ok(!handle.endsWith('-'), `handle ends with hyphen: ${handle}`);
  }
});
