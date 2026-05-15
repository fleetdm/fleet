'use strict';

const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const run = require('./stale-fleetie-issues.js');

const DAY_MS = 24 * 60 * 60 * 1000;
const daysAgoIso = (days) => new Date(Date.now() - days * DAY_MS).toISOString();

function makeIssue(overrides = {}) {
  return {
    number: 1,
    html_url: 'https://github.com/o/r/issues/1',
    user: { login: 'getvictor' },
    labels: [],
    updated_at: daysAgoIso(800),
    state: 'open',
    pull_request: undefined,
    ...overrides,
  };
}

function makeContext() {
  return { repo: { owner: 'o', repo: 'r' } };
}

function makeCore() {
  const infos = [];
  const warnings = [];
  const summaryCalls = [];
  const summary = new Proxy(
    {},
    {
      get(_target, prop) {
        if (prop === 'write') {
          return async () => {
            summaryCalls.push({ method: 'write' });
          };
        }
        return (...args) => {
          summaryCalls.push({ method: String(prop), args });
          return summary;
        };
      },
    },
  );
  return {
    info: (msg) => infos.push(msg),
    warning: (msg) => warnings.push(msg),
    summary,
    _captured: { infos, warnings, summaryCalls },
  };
}

function makeGithub(issuesByPage) {
  const createCommentCalls = [];
  const addLabelsCalls = [];
  const updateCalls = [];
  return {
    paginate: {
      iterator: async function* iterator() {
        for (const page of issuesByPage) {
          yield { data: page };
        }
      },
    },
    rest: {
      issues: {
        listForRepo: 'listForRepo-sentinel',
        createComment: async (params) => {
          createCommentCalls.push(params);
        },
        addLabels: async (params) => {
          addLabelsCalls.push(params);
        },
        update: async (params) => {
          updateCalls.push(params);
        },
      },
    },
    _captured: { createCommentCalls, addLabelsCalls, updateCalls },
  };
}

function writeHandlesFile(handles) {
  const file = path.join(os.tmpdir(), `fleeties-${process.pid}-${Math.random()}.txt`);
  fs.writeFileSync(file, handles.join('\n') + '\n');
  return file;
}

async function runWith({ issues, handles, dryRun = false, maxOps = 1000 }) {
  process.env.FLEETIE_HANDLES_FILE = writeHandlesFile(handles);
  process.env.DRY_RUN = dryRun ? 'true' : 'false';
  process.env.MAX_OPERATIONS = String(maxOps);
  const github = makeGithub([issues]);
  const context = makeContext();
  const core = makeCore();
  const result = await run({ github, context, core });
  return { github, core, result };
}

test('marks a Fleetie-authored issue idle >2y as stale', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(800) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.deepStrictEqual(github._captured.addLabelsCalls[0].labels, ['stale']);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(result.closed.length, 0);
});

test('closes a Fleetie-authored issue that already has stale label and is idle >14d', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({
        number: 2,
        user: { login: 'getvictor' },
        labels: [{ name: 'stale' }],
        updated_at: daysAgoIso(20),
      }),
    ],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  assert.strictEqual(github._captured.updateCalls.length, 1);
  assert.strictEqual(github._captured.updateCalls[0].state, 'closed');
  assert.strictEqual(github._captured.updateCalls[0].state_reason, 'not_planned');
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(result.closed.length, 1);
});

test('skips non-Fleetie author', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [makeIssue({ user: { login: 'someoneelse' }, updated_at: daysAgoIso(800) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(result.skippedNonFleetie, 1);
});

test('Fleetie author match is case-insensitive', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [makeIssue({ user: { login: 'GetVictor' }, updated_at: daysAgoIso(800) })],
  });
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.strictEqual(result.staled.length, 1);
});

test('skips issues exempted by bug, :product, or customer-* labels', async () => {
  for (const labelName of ['bug', ':product', 'customer-acme', 'customer-foo']) {
    const { github, result } = await runWith({
      handles: ['getvictor'],
      issues: [
        makeIssue({
          user: { login: 'getvictor' },
          labels: [{ name: labelName }],
          updated_at: daysAgoIso(800),
        }),
      ],
    });
    assert.strictEqual(
      github._captured.createCommentCalls.length,
      0,
      `expected no writes for label ${labelName}`,
    );
    assert.strictEqual(result.skippedExempt, 1, `expected ${labelName} to be exempt`);
  }
});

test('dry-run never writes', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(800) }),
      makeIssue({
        number: 2,
        user: { login: 'getvictor' },
        labels: [{ name: 'stale' }],
        updated_at: daysAgoIso(20),
      }),
    ],
    dryRun: true,
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  // The candidate lists still reflect what would have happened.
  assert.strictEqual(result.dryRun, true);
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(result.closed.length, 1);
});

test('respects max_operations cap (each modified issue costs 2 writes)', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(800) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(799) }),
      makeIssue({ number: 3, user: { login: 'getvictor' }, updated_at: daysAgoIso(798) }),
    ],
    maxOps: 2,
  });
  assert.strictEqual(github._captured.addLabelsCalls.length, 1, 'should stop after one issue');
  assert.strictEqual(result.hitCap, true);
});

test('excludes pull requests', async () => {
  const { github } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({
        user: { login: 'getvictor' },
        updated_at: daysAgoIso(800),
        pull_request: { url: 'x' },
      }),
    ],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
});

test('skips Fleetie issue idle between 14 and 730 days as "not stale yet"', async () => {
  const { github, result } = await runWith({
    handles: ['getvictor'],
    issues: [makeIssue({ user: { login: 'getvictor' }, updated_at: daysAgoIso(100) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(result.skippedNotStaleYet, 1);
});

test('stops collecting candidates at issues younger than 14 days', async () => {
  // Pages are sorted oldest-first by GitHub. Once we see a young issue, we stop.
  const { result } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(800) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(5) }),
      makeIssue({ number: 3, user: { login: 'getvictor' }, updated_at: daysAgoIso(800) }),
    ],
  });
  // Only #1 should be considered; #2 stops the scan, #3 is never reached.
  assert.strictEqual(result.candidates, 1);
});

test('handles string-form labels in addition to object-form labels', async () => {
  const { result } = await runWith({
    handles: ['getvictor'],
    issues: [
      makeIssue({
        user: { login: 'getvictor' },
        labels: ['bug'], // GitHub sometimes serializes labels as bare strings.
        updated_at: daysAgoIso(800),
      }),
    ],
  });
  assert.strictEqual(result.skippedExempt, 1);
});
