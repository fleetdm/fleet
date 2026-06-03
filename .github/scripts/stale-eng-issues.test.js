'use strict';

const test = require('node:test');
const assert = require('node:assert');

const run = require('./stale-eng-issues.js');
// Pull constants from the script so a future policy change (e.g. STALE_DAYS 365 -> 730) surfaces
// in the boundary tests instead of silently passing because the test hardcoded the old value.
const { STALE_DAYS, CLOSE_DAYS, ELIGIBLE_LABEL } = run;

const DAY_MS = 24 * 60 * 60 * 1000;
const daysAgoIso = (days) => new Date(Date.now() - days * DAY_MS).toISOString();

// Eng-initiated issues qualify by carrying the `~engineering-initiated` label, so the default
// issue includes it. Tests that exercise the ineligible path drop it explicitly.
function makeIssue(overrides = {}) {
  return {
    number: 1,
    html_url: 'https://github.com/o/r/issues/1',
    user: { login: 'getvictor' },
    labels: [{ name: ELIGIBLE_LABEL }],
    updated_at: daysAgoIso(STALE_DAYS + 70),
    state: 'open',
    pull_request: undefined,
    ...overrides,
  };
}

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

async function runWith({ issues, issuesByPage, dryRun = false, maxOps = 1000, eventsByIssue = {}, failOn = {} } = {}) {
  process.env.DRY_RUN = dryRun ? 'true' : 'false';
  process.env.MAX_OPERATIONS = String(maxOps);
  const pages = issuesByPage != null ? issuesByPage : [issues || []];
  const github = makeGithub({ issuesByPage: pages, eventsByIssue, failOn });
  const context = makeContext();
  const core = makeCore();
  const result = await run({ github, context, core });
  return { github, core, result };
}

test('marks an eng-initiated issue idle >1y as stale and @-mentions the author', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  assert.match(github._captured.createCommentCalls[0].body, /^@getvictor /, 'stale comment @-mentions the author');
  assert.strictEqual(github._captured.addLabelsCalls.length, 1);
  assert.deepStrictEqual(github._captured.addLabelsCalls[0].labels, ['stale']);
  assert.strictEqual(result.staled.length, 1);
});

test('skips issues without the ~engineering-initiated label', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ labels: [], updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(result.skippedIneligible, 1);
  assert.strictEqual(result.staled.length, 0);
});

test('eligible-label match is case-insensitive', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ labels: [{ name: ELIGIBLE_LABEL.toUpperCase() }], updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(result.staled.length, 1);
});

test('does not exempt bug-labeled eng issues (unlike the Fleetie closer)', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ labels: [{ name: ELIGIBLE_LABEL }, { name: 'bug' }], updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(result.skippedExempt, 0);
  assert.strictEqual(result.staled.length, 1);
});

test('closes a stale-labeled eng issue idle >14d with no activity after labeling', async () => {
  const labeledAt = Date.now() - 20 * DAY_MS;
  const issue = makeIssue({
    number: 2,
    labels: [{ name: ELIGIBLE_LABEL }, { name: 'stale' }],
    updated_at: new Date(labeledAt).toISOString(),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 2: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(github._captured.updateCalls.length, 1);
  assert.strictEqual(github._captured.updateCalls[0].state, 'closed');
  assert.strictEqual(github._captured.updateCalls[0].state_reason, 'not_planned');
  assert.strictEqual(result.closed.length, 1);
});

test('un-stales an eng issue with activity after labeling', async () => {
  const issue = makeIssue({
    number: 3,
    labels: [{ name: ELIGIBLE_LABEL }, { name: 'stale' }],
    updated_at: daysAgoIso(5),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 3: [makeStaleLabelEvent({ daysAgo: 20 })] },
  });
  assert.strictEqual(github._captured.removeLabelCalls.length, 1);
  assert.strictEqual(github._captured.removeLabelCalls[0].name, 'stale');
  assert.strictEqual(result.unstaled.length, 1);
});

test('skips eng issue idle between 14 and 365 days as "not stale yet"', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ updated_at: daysAgoIso(100) })],
  });
  assert.strictEqual(result.skippedNotStaleYet, 1);
  assert.strictEqual(result.staled.length, 0);
});

test('excludes pull requests', async () => {
  const { github } = await runWith({
    issues: [makeIssue({ updated_at: daysAgoIso(STALE_DAYS + 70), pull_request: { url: 'x' } })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
});

test('dry-run never writes', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ updated_at: daysAgoIso(STALE_DAYS + 70) })],
    dryRun: true,
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 0);
  assert.strictEqual(github._captured.addLabelsCalls.length, 0);
  assert.strictEqual(result.dryRun, true);
  assert.strictEqual(result.staled.length, 1);
});

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
