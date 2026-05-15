// Marks open issues authored by current/former Fleeties as stale after 2y of inactivity, and closes them
// after 14 more days. Exempts `bug`, `:product`, and `customer-*` labels. Invoked by
// `actions/github-script` from `.github/workflows/close-stale-fleetie-initiated-issues.yml`.
//
// If a stale-labeled issue receives activity (e.g. a comment) after being labeled, the close phase
// removes the stale label instead of closing — mirroring `actions/stale`'s `remove-stale-when-updated`
// behavior. Detected by comparing `updated_at` against the most recent `labeled` event for `stale`.
//
// Inputs (env):
//   FLEETIE_HANDLES_FILE  Path to newline-delimited lowercased GitHub usernames (built by
//                         `build-fleetie-handles.js`).
//   DRY_RUN               'true' to log candidates without writing.
//   MAX_OPERATIONS        Cap on API write operations per run. Default 400. `0` disables writes.
//
// Exports: `async function run({ github, context, core })`. Returns a summary object for tests.

'use strict';

const fs = require('node:fs');

const STALE_DAYS = 730;
const CLOSE_DAYS = 14;
const STALE_LABEL = 'stale';
const STALE_MSG =
  'This issue is stale because it was opened by a current or former Fleetie and has had ' +
  'no activity for 2 years. Please update the issue if it is still relevant; otherwise it ' +
  'will be closed in 14 days.';
const CLOSE_MSG =
  'This issue was closed because it has been inactive for 14 days since being marked as stale.';
// Tolerance for our own label+comment landing milliseconds apart. Distinguishes the bot's own
// activity bump from genuine user activity after labeling.
const SELF_ACTIVITY_EPSILON_MS = 60 * 1000;

const isExempt = (name) => {
  const lower = (name || '').toLowerCase();
  return lower === 'bug' || lower === ':product' || lower.startsWith('customer-');
};

const parseMaxOps = (raw) => {
  const parsed = Number.parseInt(raw ?? '', 10);
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : 400;
};

module.exports = async function run({ github, context, core }) {
  const handles = new Set(
    fs
      .readFileSync(process.env.FLEETIE_HANDLES_FILE, 'utf8')
      .split('\n')
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean),
  );

  const dryRun = String(process.env.DRY_RUN).toLowerCase() === 'true';
  const maxOps = parseMaxOps(process.env.MAX_OPERATIONS);

  const now = Date.now();
  const daysSince = (iso) => (now - new Date(iso).getTime()) / (1000 * 60 * 60 * 24);

  core.info(`Loaded ${handles.size} Fleetie handles. dry_run=${dryRun}, max_operations=${maxOps}`);

  // Collect candidates by scanning all open issues. Two groups qualify:
  //   1. Idle >= CLOSE_DAYS — feeds the stale and close phases.
  //   2. Currently `stale`-labeled regardless of idle time — feeds the un-stale phase so a user
  //      comment on a stale issue removes the label on the next run, not 14 days later.
  const candidates = [];
  const iterator = github.paginate.iterator(github.rest.issues.listForRepo, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    state: 'open',
    sort: 'updated',
    direction: 'asc',
    per_page: 100,
  });
  for await (const { data } of iterator) {
    for (const issue of data) {
      if (issue.pull_request) continue;
      const idleDays = daysSince(issue.updated_at);
      const hasStaleLabel = (issue.labels || []).some(
        (l) => (typeof l === 'string' ? l : (l && l.name) || '').toLowerCase() === STALE_LABEL,
      );
      if (idleDays >= CLOSE_DAYS || hasStaleLabel) {
        candidates.push(issue);
      }
    }
  }
  core.info(`Collected ${candidates.length} open candidate issues (idle >= ${CLOSE_DAYS}d or stale-labeled)`);

  const staled = [];
  const closed = [];
  const unstaled = [];
  const errored = [];
  let skippedNonFleetie = 0;
  let skippedExempt = 0;
  let skippedNotStaleYet = 0;
  let writes = 0;
  let hitCap = false;

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  for (const issue of candidates) {
    // Conservative pre-check: each iteration may do up to 2 writes (stale or close phases). The
    // unstale path is 1 write, so this estimate is safe but slightly over-conservative on that path.
    if (!dryRun && writes + 2 > maxOps) {
      hitCap = true;
      core.warning(`Reached max_operations=${maxOps}; stopping.`);
      break;
    }

    const author = (issue.user && issue.user.login ? issue.user.login : '').toLowerCase();
    if (!handles.has(author)) {
      skippedNonFleetie++;
      continue;
    }

    const labelNames = (issue.labels || []).map((l) => (typeof l === 'string' ? l : l.name) || '');
    if (labelNames.some(isExempt)) {
      skippedExempt++;
      continue;
    }

    const idleDays = daysSince(issue.updated_at);
    const alreadyStale = labelNames.some((n) => n.toLowerCase() === STALE_LABEL);

    if (alreadyStale) {
      // Determine whether there's been activity after the stale label was applied. If so, remove the
      // stale label (mirroring actions/stale's remove-stale-when-updated) and skip the close.
      let events;
      try {
        events = await github.paginate(github.rest.issues.listEvents, {
          owner,
          repo,
          issue_number: issue.number,
          per_page: 100,
        });
      } catch (err) {
        core.warning(`listEvents failed for #${issue.number}: ${err.message}; skipping`);
        errored.push({ number: issue.number, phase: 'check-activity', message: err.message });
        continue;
      }

      let lastStaleLabelEvent = null;
      for (let i = events.length - 1; i >= 0; i--) {
        const e = events[i];
        if (e.event === 'labeled' && e.label && e.label.name === STALE_LABEL) {
          lastStaleLabelEvent = e;
          break;
        }
      }

      if (!lastStaleLabelEvent) {
        core.warning(`#${issue.number}: has '${STALE_LABEL}' label but no labeling event in history; skipping`);
        continue;
      }

      const labeledAt = new Date(lastStaleLabelEvent.created_at).getTime();
      const updatedAt = new Date(issue.updated_at).getTime();
      const activityAfterLabel = updatedAt > labeledAt + SELF_ACTIVITY_EPSILON_MS;

      if (activityAfterLabel) {
        const entry = { number: issue.number, url: issue.html_url, author, idleDays };
        core.info(`unstale: #${issue.number} by @${author} (activity after label)`);
        if (dryRun) {
          unstaled.push(entry);
        } else {
          try {
            await github.rest.issues.removeLabel({
              owner,
              repo,
              issue_number: issue.number,
              name: STALE_LABEL,
            });
            writes += 1;
            unstaled.push(entry);
          } catch (err) {
            if (err && err.status === 404) {
              // Label already gone — idempotent success.
              unstaled.push(entry);
            } else {
              core.warning(`unstale failed for #${issue.number}: ${err.message}`);
              errored.push({ number: issue.number, phase: 'unstale', message: err.message });
            }
          }
        }
        continue;
      }

      // Close phase: stale label present, no activity since labeling, >= CLOSE_DAYS idle.
      const entry = { number: issue.number, url: issue.html_url, author, idleDays };
      core.info(`close: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`);
      if (dryRun) {
        closed.push(entry);
      } else {
        try {
          await github.rest.issues.createComment({
            owner,
            repo,
            issue_number: issue.number,
            body: CLOSE_MSG,
          });
          await github.rest.issues.update({
            owner,
            repo,
            issue_number: issue.number,
            state: 'closed',
            state_reason: 'not_planned',
          });
          writes += 2;
          closed.push(entry);
        } catch (err) {
          core.warning(`close failed for #${issue.number}: ${err.message}`);
          errored.push({ number: issue.number, phase: 'close', message: err.message });
        }
      }
    } else if (idleDays >= STALE_DAYS) {
      const entry = { number: issue.number, url: issue.html_url, author, idleDays };
      core.info(`stale: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`);
      if (dryRun) {
        staled.push(entry);
      } else {
        try {
          await github.rest.issues.createComment({
            owner,
            repo,
            issue_number: issue.number,
            body: STALE_MSG,
          });
          await github.rest.issues.addLabels({
            owner,
            repo,
            issue_number: issue.number,
            labels: [STALE_LABEL],
          });
          writes += 2;
          staled.push(entry);
        } catch (err) {
          core.warning(`stale failed for #${issue.number}: ${err.message}`);
          errored.push({ number: issue.number, phase: 'stale', message: err.message });
        }
      }
    } else {
      skippedNotStaleYet++;
    }
  }

  const fmt = (list) =>
    list.length
      ? list
          .map((e) => `- [#${e.number}](${e.url}) by @${e.author} (idle ${e.idleDays.toFixed(1)}d)`)
          .join('\n')
      : '_none_';
  const fmtErrors = (list) =>
    list.length ? list.map((e) => `- #${e.number} (${e.phase}): ${e.message}`).join('\n') : '_none_';

  await core.summary
    .addHeading('Fleetie stale-issue closer')
    .addRaw(`Mode: **${dryRun ? 'dry-run' : 'live'}**`)
    .addBreak()
    .addRaw(`Fleetie handles loaded: **${handles.size}**`)
    .addBreak()
    .addRaw(`Open issues considered: **${candidates.length}**`)
    .addList([
      `Skipped (non-Fleetie author): ${skippedNonFleetie}`,
      `Skipped (exempt label: bug, :product, customer-*): ${skippedExempt}`,
      `Skipped (Fleetie-authored but younger than ${STALE_DAYS} days): ${skippedNotStaleYet}`,
      `Marked stale this run: ${staled.length}`,
      `Closed this run: ${closed.length}`,
      `Un-staled this run (activity after label): ${unstaled.length}`,
      `Errors: ${errored.length}`,
    ])
    .addHeading('Marked stale', 3)
    .addRaw(fmt(staled))
    .addBreak()
    .addHeading('Closed', 3)
    .addRaw(fmt(closed))
    .addBreak()
    .addHeading('Un-staled (activity after label)', 3)
    .addRaw(fmt(unstaled))
    .addBreak()
    .addHeading('Errors', 3)
    .addRaw(fmtErrors(errored))
    .write();

  return {
    dryRun,
    candidates: candidates.length,
    staled,
    closed,
    unstaled,
    errored,
    skippedNonFleetie,
    skippedExempt,
    skippedNotStaleYet,
    writes,
    hitCap,
  };
};
