'use strict';

const test = require('node:test');
const assert = require('node:assert');

const run = require('./stale-eng-issues.js');
// Pull constants from the script so the boundary tests keep exercising the real boundary if a
// future policy change (e.g. STALE_DAYS 365 -> 730) moves it.
const { STALE_DAYS, CLOSE_DAYS, ELIGIBLE_LABEL } = run;
const { DAY_MS, daysAgoIso, makeStaleLabelEvent, makeContext, makeCore, makeGithub } = require('./stale-test-helpers.js');

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

// Core engine behavior (caps, error paths, pagination, un-stale epsilon, etc.) is covered through
// the Fleetie wrapper in stale-fleetie-issues.test.js. These tests pin what the eng wrapper
// configures differently: label-based eligibility, no exempt labels, the 1-year threshold, and the
// eng-specific message wording.

test('marks an eng-initiated issue idle >1y as stale and @-mentions the author', async () => {
  const { github, result } = await runWith({
    issues: [makeIssue({ number: 1, user: { login: 'getvictor' }, updated_at: daysAgoIso(STALE_DAYS + 70) })],
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1);
  const staleComment = github._captured.createCommentCalls[0].body;
  assert.match(staleComment, /^@getvictor /, 'stale comment @-mentions the author');
  assert.ok(staleComment.includes('365 days'), 'stale comment uses the eng wording, not the Fleetie template');
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
  const labeledAt = Date.now() - (CLOSE_DAYS + 6) * DAY_MS;
  const issue = makeIssue({
    number: 2,
    labels: [{ name: ELIGIBLE_LABEL }, { name: 'stale' }],
    updated_at: new Date(labeledAt).toISOString(),
  });
  const { github, result } = await runWith({
    issues: [issue],
    eventsByIssue: { 2: [makeStaleLabelEvent({ at: labeledAt })] },
  });
  assert.strictEqual(github._captured.createCommentCalls.length, 1, 'close comment posted');
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
  assert.strictEqual(github._captured.createCommentCalls.length, 0, 'un-stale writes no comment');
  assert.strictEqual(github._captured.updateCalls.length, 0, 'un-stale does not close');
  assert.strictEqual(result.unstaled.length, 1);
});

test('skips eng issue idle between 14 and 365 days as "not stale yet"', async () => {
  const { result } = await runWith({
    issues: [makeIssue({ updated_at: daysAgoIso(100) })],
  });
  assert.strictEqual(result.skippedNotStaleYet, 1);
  assert.strictEqual(result.staled.length, 0);
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
