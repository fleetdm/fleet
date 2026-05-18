'use strict';

const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const run = require('./stale-fleetie-issues.js');
// Pull constants from the script so a future policy change (e.g. STALE_DAYS 730 -> 365) surfaces
// in the boundary tests instead of silently passing because the test hardcoded the old value.
const { STALE_DAYS, CLOSE_DAYS, SELF_ACTIVITY_EPSILON_MS } = run;

const DAY_MS = 24 * 60 * 60 * 1000;
const daysAgoIso = (days) => new Date(Date.now() - days * DAY_MS).toISOString();
const msAgoIso = (ms) => new Date(Date.now() - ms).toISOString();

function makeIssue(overrides = {}) {
  return {
    number: 1,
    html_url: 'https://github.com/o/r/issues/1',
    user: { login: 'getvictor' },
    labels: [],
    updated_at: daysAgoIso(STALE_DAYS + 70),
    state: 'open',
    pull_request: undefined,
    ...overrides,
  };
}

// `at` (ms-since-epoch) takes precedence over `daysAgo` so tests aligning an event to a specific
// updated_at moment can share the same instant rather than two independent Date.now() reads.
function makeStaleLabelEvent({ daysAgo = 100, at } = {}) {
  const created_at = at != null ? new Date(at).toISOString() : daysAgoIso(daysAgo);
  return { event: 'labeled', label: { name: 'stale' }, created_at };
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

// `issuesByPage`: array of pages (each page is an array of issues) returned by paginate.iterator.
// `eventsByIssue`: map from issue.number -> array of event objects returned by paginate(listEvents).
// `failOn`: optional fault-injection { createComment, addLabels, update, removeLabel, listEvents }
//           values can be 'always', an integer (fail until Nth call), or { status: 404 }.
function makeGithub({ issuesByPage = [], eventsByIssue = {}, failOn = {} } = {}) {
  const createCommentCalls = [];
  const addLabelsCalls = [];
  const removeLabelCalls = [];
  const updateCalls = [];
  const listEventsCalls = [];

  const counters = {};
  const shouldFail = (op) => {
    const cfg = failOn[op];
    if (!cfg) return null;
    counters[op] = (counters[op] || 0) + 1;
    if (cfg === 'always') return new Error(`${op} simulated failure`);
    if (typeof cfg === 'number' && counters[op] <= cfg) return new Error(`${op} simulated failure`);
    if (typeof cfg === 'object' && cfg.status && counters[op] === 1) {
      const err = new Error(`${op} simulated failure status ${cfg.status}`);
      err.status = cfg.status;
      return err;
    }
    return null;
  };

  const paginate = async (endpoint, params) => {
    if (endpoint === 'listEvents-sentinel') {
      listEventsCalls.push(params);
      const err = shouldFail('listEvents');
      if (err) throw err;
      return eventsByIssue[params.issue_number] || [];
    }
    if (endpoint === 'listForRepo-sentinel') {
      return issuesByPage.flat();
    }
    return [];
  };
  paginate.iterator = async function* iterator(endpoint) {
    if (endpoint === 'listForRepo-sentinel') {
      for (const page of issuesByPage) yield { data: page };
    }
  };

  return {
    paginate,
    rest: {
      issues: {
        listForRepo: 'listForRepo-sentinel',
        listEvents: 'listEvents-sentinel',
        createComment: async (params) => {
          const err = shouldFail('createComment');
          if (err) throw err;
          createCommentCalls.push(params);
        },
        addLabels: async (params) => {
          const err = shouldFail('addLabels');
          if (err) throw err;
          addLabelsCalls.push(params);
        },
        removeLabel: async (params) => {
          const err = shouldFail('removeLabel');
          if (err) throw err;
          removeLabelCalls.push(params);
        },
        update: async (params) => {
          const err = shouldFail('update');
          if (err) throw err;
          updateCalls.push(params);
        },
      },
    },
    _captured: { createCommentCalls, addLabelsCalls, removeLabelCalls, updateCalls, listEventsCalls },
  };
}

const tmpFilesCreated = [];
function writeHandlesFile(handles) {
  const file = path.join(os.tmpdir(), `fleeties-${process.pid}-${Math.random()}.txt`);
  fs.writeFileSync(file, handles.join('\n') + '\n');
  tmpFilesCreated.push(file);
  return file;
}

test.after(() => {
  for (const f of tmpFilesCreated) {
    try { fs.unlinkSync(f); } catch { /* best-effort cleanup */ }
  }
});

// Pass `issuesByPage` to simulate multi-page pagination. Otherwise `issues` becomes a single page.
async function runWith({
  issues,
  issuesByPage,
  handles = ['getvictor'],
  dryRun = false,
  maxOps = 1000,
  eventsByIssue = {},
  failOn = {},
} = {}) {
  process.env.FLEETIE_HANDLES_FILE = writeHandlesFile(handles);
  process.env.DRY_RUN = dryRun ? 'true' : 'false';
  process.env.MAX_OPERATIONS = String(maxOps);
  const pages = issuesByPage != null ? issuesByPage : [issues || []];
  const github = makeGithub({ issuesByPage: pages, eventsByIssue, failOn });
  const context = makeContext();
  const core = makeCore();
  const result = await run({ github, context, core });
  return { github, core, result };
}

test('marks a Fleetie-authored issue idle >2y as stale', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.deepStrictEqual(github._captured.addLabelsCalls[0].labels, ['stale']);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(result.closed.length, 0);
});

test('closes stale-labeled issue idle >14d with no activity after labeling', async () => {
  // Bot staled 20 days ago; updated_at is ~ same moment (within self-activity epsilon).
  const labeledAt = Date.now() - 20 * DAY_MS;
  const issue = makeIssue({
    number: 2,
    user: { login: 'getvictor' },
    labels: [{ name: 'stale' }],
    updated_at: new Date(labeledAt).toISOString(),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 2: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  assert.strictEqual(github._captured.updateCalls.length, 1);
  assert.strictEqual(github._captured.updateCalls[0].state, 'closed');
  assert.strictEqual(github._captured.updateCalls[0].state_reason, 'not_planned');
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(github._captured.removeLabelCalls.length, 0);
  assert.strictEqual(result.closed.length, 1);
  assert.strictEqual(result.unstaled.length, 0);
});

test('does not close a freshly-staled issue (idle < 14 days, no activity)', async () => {
  // Bot staled 5 days ago, no further activity. Issue is stale-labeled and idle 5 days. Should
  // remain open with the stale label (close phase requires >= 14 days idle).
  const labeledAt = Date.now() - 5 * DAY_MS;
  const issue = makeIssue({
    number: 50,
    user: { login: 'getvictor' },
    labels: [{ name: 'stale' }],
    updated_at: new Date(labeledAt).toISOString(),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 50: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(github._captured.removeLabelCalls.length, 0);
  assert.strictEqual(result.closed.length, 0);
  assert.strictEqual(result.unstaled.length, 0);
  assert.strictEqual(result.skippedNotReadyToClose, 1);
});

test('un-stales (removes label, no close) when activity is detected after stale labeling', async () => {
  // Bot staled 20 days ago; user commented 5 days ago, bumping updated_at.
  const issue = makeIssue({
    number: 3,
    user: { login: 'getvictor' },
    labels: [{ name: 'stale' }],
    updated_at: daysAgoIso(5),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 3: [makeStaleLabelEvent({ daysAgo: 20 })] },
  });
  assert.strictEqual(github._captured.removeLabelCalls.length, 1);
  assert.strictEqual(github._captured.removeLabelCalls[0].name, 'stale');
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(result.unstaled.length, 1);
  assert.strictEqual(result.closed.length, 0);
});

test('un-stale tolerates removeLabel 404 (label already gone) as success', async () => {
  const issue = makeIssue({
    number: 4,
    user: { login: 'getvictor' },
    labels: [{ name: 'stale' }],
    updated_at: daysAgoIso(5),
  });
  const { result } = await runWith({
    issues: [issue],
    eventsByIssue: { 4: [makeStaleLabelEvent({ daysAgo: 20 })] },
    failOn: { removeLabel: { status: 404 } },
  });
  assert.strictEqual(result.unstaled.length, 1);
  assert.strictEqual(result.errored.length, 0);
});

test('skips stale-labeled issue when no labeling event is found in history', async () => {
  const issue = makeIssue({
    number: 5,
    user: { login: 'getvictor' },
    labels: [{ name: 'stale' }],
    updated_at: daysAgoIso(20),
  });
  const { github, core, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 5: [] },
  });
  assert.strictEqual(github._captured.removeLabelCalls.length, 0);
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(result.closed.length, 0);
  assert.strictEqual(result.unstaled.length, 0);
  assert.ok(core._captured.warnings.some((w) => w.includes('no labeling event in history')));
});

test('listEvents failure records an error and continues to next issue', async () => {
  const issues = [
    makeIssue({ number: 6, user: { login: 'getvictor' }, labels: [{ name: 'stale' }], updated_at: daysAgoIso(20) }),
    makeIssue({ number: 7, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
  ];
  const { result } = await runWith({
    issues,
    eventsByIssue: { 6: [makeStaleLabelEvent({ daysAgo: 20 })] },
    failOn: { listEvents: 1 },
  });
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'check-activity');
  assert.strictEqual(result.staled.length, 1, 'second issue should still be staled');
});

test('skips non-Fleetie author', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ user: { login: 'someoneelse' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(result.skippedNonFleetie, 1);
});

test('Fleetie author match is case-insensitive', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ user: { login: 'GetVictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.strictEqual(result.staled.length, 1);
});

test('skips issues exempted by bug, :product, or customer-* labels', async () => {
  for (const labelName of ['bug', ':product', 'customer-acme', 'customer-foo']) {
    const { github, result } = await runWith({
      issues: [
        makeIssue({
          user: { login: 'getvictor' },
          labels: [{ name: labelName }],
          updated_at: daysAgoIso(STALE_DAYS + 70),
        }),
      ],
    });
    assert.strictEqual(github._captured.createCommentCalls.length, 0, `expected no writes for label ${labelName}`);
    assert.strictEqual(result.skippedExempt, 1, `expected ${labelName} to be exempt`);
  }
});

test('dry-run never writes', async () => {
  const labeledAt = Date.now() - 20 * DAY_MS;
  const { github, result } = await runWith({
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
      makeIssue({
        number: 2,
        user: { login: 'getvictor' },
        labels: [{ name: 'stale' }],
        updated_at: new Date(labeledAt).toISOString(),
      }),
    ],
    dryRun: true,
    eventsByIssue: { 2: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(github._captured.updateCalls.length, 0);
  assert.strictEqual(github._captured.removeLabelCalls.length, 0);
  assert.strictEqual(result.dryRun, true);
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(result.closed.length, 1);
});

test('respects even max_operations cap (each modified issue costs 2 writes)', async () => {
  const { github, result } = await runWith({
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 69) }),
      makeIssue({ number: 3, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 68) }),
    ],
    maxOps: 2,
  });
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.strictEqual(result.hitCap, true);
});

test('does not exceed odd max_operations cap (CodeRabbit regression test)', async () => {
  const { github, result } = await runWith({
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 69) }),
    ],
    maxOps: 3,
  });
  const totalWrites =
    github._captured.createCommentCalls.length +
    github._captured.addLabelsCalls.length +
    github._captured.removeLabelCalls.length +
    github._captured.updateCalls.length;
  assert.ok(totalWrites <= 3, `expected <= 3 writes, got ${totalWrites}`);
  assert.strictEqual(result.hitCap, true);
});

test('MAX_OPERATIONS=0 disables all writes', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
    maxOps: 0,
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(result.hitCap, true);
  assert.strictEqual(result.staled.length, 0);
});

test('write failure in stale phase is recorded and run continues', async () => {
  const { github, result } = await runWith({
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 69) }),
    ],
    failOn: { createComment: 1 },
  });
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'stale');
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
});

test('excludes pull requests', async () => {
  const { github } = await runWith({
    issues: [
      makeIssue({
        user: { login: 'getvictor' },
        updated_at: daysAgoIso(STALE_DAYS + 70),
        pull_request: { url: 'x' },
      }),
    ],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
});

test('skips Fleetie issue idle between 14 and 730 days as "not stale yet"', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ user: { login: 'getvictor' }, updated_at: daysAgoIso(100) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(result.skippedNotStaleYet, 1);
});

test('candidates exclude recently-active issues without the stale label', async () => {
  const { result } = await runWith({
    issues: [
      makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
      makeIssue({ number: 2, user: { login: 'getvictor' }, updated_at: daysAgoIso(5) }),
      makeIssue({ number: 3, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) }),
    ],
  });
  // #1 and #3 qualify (idle >= 14d); #2 is < 14d idle and has no stale label.
  assert.strictEqual(result.candidates, 2);
});

test('candidates include stale-labeled issues regardless of idle time', async () => {
  // A recently-active (5d idle) stale-labeled issue must still be collected so the un-stale
  // path can fire — otherwise it would be missed for 14 days after the user activity.
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 99,
        user: { login: 'getvictor' },
        labels: [{ name: 'stale' }],
        updated_at: daysAgoIso(5),
      }),
    ],
    eventsByIssue: { 99: [makeStaleLabelEvent({ daysAgo: 20 })] },
  });
  assert.strictEqual(result.candidates, 1);
  assert.strictEqual(result.unstaled.length, 1);
});

test('handles string-form labels in addition to object-form labels', async () => {
  const { result } = await runWith({
    issues: [
      makeIssue({
        user: { login: 'getvictor' },
        labels: ['bug'],
        updated_at: daysAgoIso(STALE_DAYS + 70),
      }),
    ],
  });
  assert.strictEqual(result.skippedExempt, 1);
});

// ---------------------------------------------------------------------------
// Boundary tests: catch direction errors and wrong-constant regressions on
// STALE_DAYS, CLOSE_DAYS, and SELF_ACTIVITY_EPSILON_MS.
// ---------------------------------------------------------------------------

test('stale-phase boundary: issue just past STALE_DAYS is staled', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ number: 200, updated_at: daysAgoIso(STALE_DAYS + 0.01) })],
  });
  assert.strictEqual(result.staled.length, 1);
  assert.strictEqual(result.skippedNotStaleYet, 0);
});

test('stale-phase boundary: issue just under STALE_DAYS is skipped as not stale yet', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ number: 201, updated_at: daysAgoIso(STALE_DAYS - 0.1) })],
  });
  assert.strictEqual(result.staled.length, 0);
  assert.strictEqual(result.skippedNotStaleYet, 1);
});

test('close-phase boundary: stale-labeled issue just past CLOSE_DAYS is closed', async () => {
  // Bot staled CLOSE_DAYS + 0.01 days ago (~14.01 days), updated_at at the same instant so the
  // self-activity epsilon is satisfied and the close path's idle gate just barely passes.
  const labeledAt = Date.now() - (CLOSE_DAYS + 0.01) * DAY_MS;
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 202,
        labels: [{ name: 'stale' }],
        updated_at: new Date(labeledAt).toISOString(),
      }),
    ],
    eventsByIssue: { 202: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(result.closed.length, 1);
  assert.strictEqual(result.skippedNotReadyToClose, 0);
});

test('close-phase boundary: stale-labeled issue just under CLOSE_DAYS is not closed', async () => {
  const labeledAt = Date.now() - (CLOSE_DAYS - 0.1) * DAY_MS;
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 203,
        labels: [{ name: 'stale' }],
        updated_at: new Date(labeledAt).toISOString(),
      }),
    ],
    eventsByIssue: { 203: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(result.closed.length, 0);
  assert.strictEqual(result.skippedNotReadyToClose, 1);
});

test('self-activity epsilon: updated_at at the epsilon boundary is treated as self-activity (close)', async () => {
  // labeledAt 30 days ago; updated_at is exactly labeledAt + epsilon, which the script treats as
  // self-activity (the check is strict `>`). Idle is still >= CLOSE_DAYS, so close fires.
  const labeledAt = Date.now() - 30 * DAY_MS;
  const updatedAt = labeledAt + SELF_ACTIVITY_EPSILON_MS;
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 204,
        labels: [{ name: 'stale' }],
        updated_at: new Date(updatedAt).toISOString(),
      }),
    ],
    eventsByIssue: { 204: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(result.closed.length, 1);
  assert.strictEqual(result.unstaled.length, 0);
});

test('self-activity epsilon: updated_at 1ms past the epsilon boundary is treated as user activity (un-stale)', async () => {
  const labeledAt = Date.now() - 30 * DAY_MS;
  const updatedAt = labeledAt + SELF_ACTIVITY_EPSILON_MS + 1;
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 205,
        labels: [{ name: 'stale' }],
        updated_at: new Date(updatedAt).toISOString(),
      }),
    ],
    eventsByIssue: { 205: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(result.unstaled.length, 1);
  assert.strictEqual(result.closed.length, 0);
});

// ---------------------------------------------------------------------------
// Error path tests: each write phase has its own try/catch and phase tag.
// Confirm partial-write states are recorded and don't stop the rest of the run.
// ---------------------------------------------------------------------------

test('stale-phase: addLabels failure after createComment success is recorded', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ number: 300, updated_at: daysAgoIso(STALE_DAYS + 70) })],
    failOn: { addLabels: 1 },
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1, 'createComment ran first');
  assert.strictEqual(result.staled.length, 0);
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'stale');
});

test('close-phase: createComment failure is recorded and run continues', async () => {
  const labeledAt = Date.now() - 20 * DAY_MS;
  const { github, result } = await runWith({
    issues: [
      makeIssue({
        number: 400,
        labels: [{ name: 'stale' }],
        updated_at: new Date(labeledAt).toISOString(),
      }),
      makeIssue({ number: 401, updated_at: daysAgoIso(STALE_DAYS + 70) }),
    ],
    eventsByIssue: { 400: [makeStaleLabelEvent({ at: labeledAt })] },
    failOn: { createComment: 1 },
  });
  assert.strictEqual(github._captured.updateCalls.length, 0, 'update not called when createComment fails');
  assert.strictEqual(result.closed.length, 0);
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'close');
  // Subsequent issue still processed (the close-phase failure only consumes the first
  // createComment; the second issue's stale-phase createComment succeeds).
  assert.strictEqual(result.staled.length, 1);
});

test('close-phase: update failure after createComment success is recorded', async () => {
  const labeledAt = Date.now() - 20 * DAY_MS;
  const { github, result } = await runWith({
    issues: [
      makeIssue({
        number: 402,
        labels: [{ name: 'stale' }],
        updated_at: new Date(labeledAt).toISOString(),
      }),
    ],
    eventsByIssue: { 402: [makeStaleLabelEvent({ at: labeledAt })] },
    failOn: { update: 1 },
  });
  // The mock records calls only on success, so updateCalls stays empty when update throws. The
  // behavior we care about: createComment succeeded (the script got past it), close didn't
  // complete, and the failure is attributed to the close phase.
  assert.strictEqual(github._captured.createCommentCalls.length, 1, 'createComment succeeded');
  assert.strictEqual(result.closed.length, 0);
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'close');
});

test('un-stale phase: non-404 removeLabel failure is recorded', async () => {
  // Distinct from "tolerates 404" — non-404 errors go to `errored`, not silently ignored.
  const { result } = await runWith({
    issues: [
      makeIssue({
        number: 500,
        labels: [{ name: 'stale' }],
        updated_at: daysAgoIso(5),
      }),
    ],
    eventsByIssue: { 500: [makeStaleLabelEvent({ daysAgo: 20 })] },
    failOn: { removeLabel: 1 },
  });
  assert.strictEqual(result.unstaled.length, 0);
  assert.strictEqual(result.errored.length, 1);
  assert.strictEqual(result.errored[0].phase, 'unstale');
});

// ---------------------------------------------------------------------------
// Pagination: confirms issues across multiple iterator pages are all processed.
// ---------------------------------------------------------------------------

test('processes issues across multiple paginated pages', async () => {
  const { result } = await runWith({
    issuesByPage: [
      [makeIssue({ number: 600, updated_at: daysAgoIso(STALE_DAYS + 50) })],
      [makeIssue({ number: 601, updated_at: daysAgoIso(STALE_DAYS + 30) })],
      [makeIssue({ number: 602, updated_at: daysAgoIso(STALE_DAYS + 10) })],
    ],
  });
  assert.strictEqual(result.candidates, 3);
  assert.strictEqual(result.staled.length, 3);
});

// ---------------------------------------------------------------------------
// Mixed-population integration: all four outcomes in one run, exercising the
// interaction between the candidate iteration and the phase-specific branches.
// ---------------------------------------------------------------------------

test('mixed population: stale + close + un-stale + error all in one run', async () => {
  const labeledAtClose = Date.now() - 20 * DAY_MS;
  // Order matters: #700 is staled-by-bot AND first in the list so its (failing) listEvents call
  // is the first one — failOn.listEvents=1 fires there, not on #701 or #702.
  const issues = [
    makeIssue({
      number: 700,
      labels: [{ name: 'stale' }],
      updated_at: daysAgoIso(20),
    }),
    makeIssue({ number: 701, updated_at: daysAgoIso(STALE_DAYS + 70) }),
    makeIssue({
      number: 702,
      labels: [{ name: 'stale' }],
      updated_at: new Date(labeledAtClose).toISOString(),
    }),
    makeIssue({
      number: 703,
      labels: [{ name: 'stale' }],
      updated_at: daysAgoIso(5),
    }),
  ];
  const { result } = await runWith({
    issues,
    eventsByIssue: {
      // 700 has no listEvents entry because we make it fail anyway
      702: [makeStaleLabelEvent({ at: labeledAtClose })],
      703: [makeStaleLabelEvent({ daysAgo: 20 })],
    },
    failOn: { listEvents: 1 },
  });
  assert.strictEqual(result.candidates, 4);
  assert.strictEqual(result.staled.length, 1, '#701 staled');
  assert.strictEqual(result.closed.length, 1, '#702 closed');
  assert.strictEqual(result.unstaled.length, 1, '#703 un-staled');
  assert.strictEqual(result.errored.length, 1, '#700 errored on listEvents');
  assert.strictEqual(result.errored[0].phase, 'check-activity');
  assert.strictEqual(result.errored[0].number, 700);
});
