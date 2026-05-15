// Marks open issues authored by current/former Fleeties as stale after 2y of inactivity, and closes them
// after 14 more days. Exempts `bug`, `:product`, and `customer-*` labels. Invoked by
// `actions/github-script` from `.github/workflows/close-stale-fleetie-initiated-issues.yml`.
//
// Inputs (env):
//   FLEETIE_HANDLES_FILE  Path to newline-delimited lowercased GitHub usernames (built by
//                         `build-fleetie-handles.js`).
//   DRY_RUN               'true' to log candidates without writing.
//   MAX_OPERATIONS        Cap on API write operations per run. Each modified issue costs 2.
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

const isExempt = (name) => {
  const lower = (name || '').toLowerCase();
  return lower === 'bug' || lower === ':product' || lower.startsWith('customer-');
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
  const maxOps = parseInt(process.env.MAX_OPERATIONS, 10) || 200;

  const now = Date.now();
  const daysSince = (iso) => (now - new Date(iso).getTime()) / (1000 * 60 * 60 * 24);

  core.info(`Loaded ${handles.size} Fleetie handles. dry_run=${dryRun}, max_operations=${maxOps}`);

  // Collect candidates first (no writes during iteration) so page boundaries are stable.
  // Sort oldest-updated first; stop as soon as we see an issue younger than CLOSE_DAYS.
  const candidates = [];
  const iterator = github.paginate.iterator(github.rest.issues.listForRepo, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    state: 'open',
    sort: 'updated',
    direction: 'asc',
    per_page: 100,
  });
  let reachedYounger = false;
  for await (const { data } of iterator) {
    for (const issue of data) {
      if (issue.pull_request) continue;
      if (daysSince(issue.updated_at) < CLOSE_DAYS) {
        reachedYounger = true;
        break;
      }
      candidates.push(issue);
    }
    if (reachedYounger) break;
  }
  core.info(`Collected ${candidates.length} open issues idle for at least ${CLOSE_DAYS} days`);

  const staled = [];
  const closed = [];
  let skippedNonFleetie = 0;
  let skippedExempt = 0;
  let skippedNotStaleYet = 0;
  let writes = 0;
  let hitCap = false;

  for (const issue of candidates) {
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
      // Close phase: stale label present + at least CLOSE_DAYS idle since it was applied.
      const entry = { number: issue.number, url: issue.html_url, author, idleDays };
      closed.push(entry);
      core.info(`close: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`);
      if (!dryRun) {
        await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issue.number,
          body: CLOSE_MSG,
        });
        await github.rest.issues.update({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issue.number,
          state: 'closed',
          state_reason: 'not_planned',
        });
        // Each modified issue performs exactly 2 API writes (comment + close).
        writes += 2;
      }
    } else if (idleDays >= STALE_DAYS) {
      const entry = { number: issue.number, url: issue.html_url, author, idleDays };
      staled.push(entry);
      core.info(`stale: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`);
      if (!dryRun) {
        await github.rest.issues.createComment({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issue.number,
          body: STALE_MSG,
        });
        await github.rest.issues.addLabels({
          owner: context.repo.owner,
          repo: context.repo.repo,
          issue_number: issue.number,
          labels: [STALE_LABEL],
        });
        // Each modified issue performs exactly 2 API writes (comment + addLabels).
        writes += 2;
      }
    } else {
      skippedNotStaleYet++;
    }

    if (!dryRun && writes >= maxOps) {
      hitCap = true;
      core.warning(`Reached max_operations=${maxOps}; stopping.`);
      break;
    }
  }

  const fmt = (list) =>
    list.length
      ? list
          .map((e) => `- [#${e.number}](${e.url}) by @${e.author} (idle ${e.idleDays.toFixed(1)}d)`)
          .join('\n')
      : '_none_';

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
    ])
    .addHeading('Marked stale', 3)
    .addRaw(fmt(staled))
    .addBreak()
    .addHeading('Closed', 3)
    .addRaw(fmt(closed))
    .write();

  return { dryRun, candidates: candidates.length, staled, closed, skippedNonFleetie, skippedExempt, skippedNotStaleYet, writes, hitCap };
};
