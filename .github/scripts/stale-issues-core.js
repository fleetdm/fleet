// Generic stale-issue engine shared by the eng-initiated and Fleetie-initiated closers. It scans
// open issues, marks eligible idle ones stale (comment + label), closes stale ones after a further
// idle period, and removes the stale label when an issue receives activity after being labeled.
//
// The two callers differ only in how an issue qualifies (label-based vs author-based), the idle
// thresholds, the wording of the comments, and the summary labels. Those differences are passed in
// via `config`; everything below is identical between them. See `stale-eng-issues.js` and
// `stale-fleetie-issues.js` for the two wrappers.
//
// If a stale-labeled issue receives activity (e.g. a comment) after being labeled, the close phase
// removes the stale label instead of closing, mirroring `actions/stale`'s `remove-stale-when-updated`
// behavior. Detected by comparing `updated_at` against the most recent `labeled` event for `stale`.
//
// Inputs (env, read here so both wrappers share the same operator controls):
//   DRY_RUN         'true' to log candidates without writing.
//   MAX_OPERATIONS  Cap on API write operations per run. Default 400. `0` disables writes (treated as a dry run).
//
// config:
//   title                  Heading for the job summary.
//   staleDays              Idle days before an issue is marked stale.
//   closeDays              Further idle days (after labeling) before a stale issue is closed.
//   staleLabel             Label applied to mark an issue stale (default "stale").
//   isEligible(issue)      Returns true if the issue is in scope for this closer.
//   isExempt(labelName)    Returns true if a label exempts the issue from staleness.
//   staleMessage(author)   Comment body posted when marking stale (mentions the author).
//   closeMessage(author)   Comment body posted when closing.
//   ineligibleSummaryLabel Summary line text for the "skipped, out of scope" counter.
//   summaryLines           Extra summary lines (array of strings) injected near the top.
//
// Exports: `async function run({ github, context, core, config })`. Returns a summary object for tests.

"use strict";

const STALE_LABEL_DEFAULT = "stale";
// Tolerance for our own label+comment landing milliseconds apart. Distinguishes the bot's own
// activity bump from genuine user activity after labeling.
const SELF_ACTIVITY_EPSILON_MS = 60 * 1000;
const MS_PER_DAY = 1000 * 60 * 60 * 24;
// Opt in to the 2026-03-10 REST API version on every call.
const GH_API_HEADERS = { "x-github-api-version": "2026-03-10" };

const parseMaxOps = (raw) => {
  const parsed = Number.parseInt(raw ?? "", 10);
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : 400;
};

async function run({ github, context, core, config }) {
  const {
    title,
    staleDays,
    closeDays,
    staleLabel = STALE_LABEL_DEFAULT,
    isEligible,
    isExempt = () => false,
    staleMessage,
    closeMessage,
    ineligibleSummaryLabel = "Skipped (out of scope)",
    summaryLines = [],
  } = config;

  const maxOps = parseMaxOps(process.env.MAX_OPERATIONS);
  // MAX_OPERATIONS=0 is the kill switch: scan and report what would happen, but make no writes.
  // Treating it as dry-run (rather than breaking on the first cap check) keeps the summary complete.
  const dryRun =
    String(process.env.DRY_RUN).toLowerCase() === "true" || maxOps === 0;

  // All time math is in UTC-equivalent epoch milliseconds: Date.now() is UTC, and the GitHub
  // REST API returns ISO 8601 strings with a `Z` suffix, so timezone and DST cannot affect the
  // result. `daysSince` is a 24-hour-day count, not a calendar-day count.
  const now = Date.now();
  const daysSince = (iso) => (now - new Date(iso).getTime()) / MS_PER_DAY;

  core.info(
    `${title}: dry_run=${dryRun}, max_operations=${maxOps}, stale_days=${staleDays}, close_days=${closeDays}`
  );

  // Collect candidates by scanning all open issues. Two groups qualify:
  //   1. Idle >= closeDays — feeds the stale and close phases.
  //   2. Currently stale-labeled regardless of idle time — feeds the un-stale phase so a user
  //      comment on a stale issue removes the label on the next run, not closeDays later.
  const candidates = [];
  const iterator = github.paginate.iterator(github.rest.issues.listForRepo, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    state: "open",
    sort: "updated",
    direction: "asc",
    per_page: 100,
    headers: GH_API_HEADERS,
  });
  for await (const { data } of iterator) {
    for (const issue of data) {
      if (issue.pull_request) continue;
      const idleDays = daysSince(issue.updated_at);
      const hasStaleLabel = (issue.labels || []).some(
        (l) =>
          (typeof l === "string" ? l : (l && l.name) || "").toLowerCase() ===
          staleLabel.toLowerCase()
      );
      if (idleDays >= closeDays || hasStaleLabel) {
        candidates.push(issue);
      }
    }
  }
  core.info(
    `Collected ${candidates.length} open candidate issues (idle >= ${closeDays}d or ${staleLabel}-labeled)`
  );

  const staled = [];
  const closed = [];
  const unstaled = [];
  const errored = [];
  let skippedIneligible = 0;
  let skippedExempt = 0;
  let skippedNotStaleYet = 0;
  let skippedNotReadyToClose = 0;
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

    if (!isEligible(issue)) {
      skippedIneligible++;
      continue;
    }

    const author = (issue.user && issue.user.login) || "";

    const labelNames = (issue.labels || []).map(
      (l) => (typeof l === "string" ? l : l.name) || ""
    );
    if (labelNames.some(isExempt)) {
      skippedExempt++;
      continue;
    }

    const idleDays = daysSince(issue.updated_at);
    const alreadyStale = labelNames.some(
      (n) => n.toLowerCase() === staleLabel.toLowerCase()
    );

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
          headers: GH_API_HEADERS,
        });
      } catch (err) {
        core.warning(
          `listEvents failed for #${issue.number}: ${err.message}; skipping`
        );
        errored.push({
          number: issue.number,
          phase: "check-activity",
          message: err.message,
        });
        continue;
      }

      let lastStaleLabelEvent = null;
      for (let i = events.length - 1; i >= 0; i--) {
        const e = events[i];
        if (
          e.event === "labeled" &&
          ((e.label && e.label.name) || "").toLowerCase() ===
            staleLabel.toLowerCase()
        ) {
          lastStaleLabelEvent = e;
          break;
        }
      }

      if (!lastStaleLabelEvent) {
        core.warning(
          `#${issue.number}: has '${staleLabel}' label but no labeling event in history; skipping`
        );
        continue;
      }

      const labeledAt = new Date(lastStaleLabelEvent.created_at).getTime();
      const updatedAt = new Date(issue.updated_at).getTime();
      const activityAfterLabel =
        updatedAt > labeledAt + SELF_ACTIVITY_EPSILON_MS;

      if (activityAfterLabel) {
        const entry = {
          number: issue.number,
          url: issue.html_url,
          author,
          idleDays,
        };
        core.info(
          `unstale: #${issue.number} by @${author} (activity after label)`
        );
        if (dryRun) {
          unstaled.push(entry);
        } else {
          try {
            await github.rest.issues.removeLabel({
              owner,
              repo,
              issue_number: issue.number,
              name: staleLabel,
              headers: GH_API_HEADERS,
            });
            writes += 1;
            unstaled.push(entry);
          } catch (err) {
            if (err && err.status === 404) {
              // Label already gone (idempotent success).
              unstaled.push(entry);
            } else {
              core.warning(
                `unstale failed for #${issue.number}: ${err.message}`
              );
              errored.push({
                number: issue.number,
                phase: "unstale",
                message: err.message,
              });
            }
          }
        }
        continue;
      }

      // Close phase: stale label present, no activity since labeling, AND >= closeDays idle.
      // The idle gate matters because candidate collection accepts stale-labeled issues regardless
      // of idle time (so the un-stale path runs promptly on activity); without this check, a
      // freshly-staled issue with no activity would be closed on the very next run.
      if (idleDays < closeDays) {
        skippedNotReadyToClose++;
        continue;
      }
      const entry = {
        number: issue.number,
        url: issue.html_url,
        author,
        idleDays,
      };
      core.info(
        `close: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`
      );
      if (dryRun) {
        closed.push(entry);
      } else {
        try {
          // Close before commenting. A close-comment that lands without the close would bump
          // updated_at, and the next run would misread the bot's own comment as user activity and
          // un-stale a still-open issue, dropping it from the cycle for another staleDays.
          await github.rest.issues.update({
            owner,
            repo,
            issue_number: issue.number,
            state: "closed",
            state_reason: "not_planned",
            headers: GH_API_HEADERS,
          });
          writes += 1;
          await github.rest.issues.createComment({
            owner,
            repo,
            issue_number: issue.number,
            body: closeMessage(author),
            headers: GH_API_HEADERS,
          });
          writes += 1;
          closed.push(entry);
        } catch (err) {
          core.warning(`close failed for #${issue.number}: ${err.message}`);
          errored.push({
            number: issue.number,
            phase: "close",
            message: err.message,
          });
        }
      }
    } else if (idleDays >= staleDays) {
      const entry = {
        number: issue.number,
        url: issue.html_url,
        author,
        idleDays,
      };
      core.info(
        `stale: #${issue.number} by @${author}, idle ${idleDays.toFixed(1)}d`
      );
      if (dryRun) {
        staled.push(entry);
      } else {
        try {
          // Label before commenting. A stale-comment that lands without the label would bump
          // updated_at and reset the staleness clock for another staleDays with nothing to show for
          // it; a label without the comment still closes on schedule, just without the warning.
          await github.rest.issues.addLabels({
            owner,
            repo,
            issue_number: issue.number,
            labels: [staleLabel],
            headers: GH_API_HEADERS,
          });
          writes += 1;
          await github.rest.issues.createComment({
            owner,
            repo,
            issue_number: issue.number,
            body: staleMessage(author),
            headers: GH_API_HEADERS,
          });
          writes += 1;
          staled.push(entry);
        } catch (err) {
          core.warning(`stale failed for #${issue.number}: ${err.message}`);
          errored.push({
            number: issue.number,
            phase: "stale",
            message: err.message,
          });
        }
      }
    } else {
      skippedNotStaleYet++;
    }
  }

  // Each entry is an HTML <li> so it renders correctly inside <ul>. We use raw <a> tags rather
  // than markdown links because GitHub Actions summary surrounds list content with HTML blocks
  // (from addHeading / addList), which suspends markdown parsing for child content — markdown
  // bullets via addRaw collapse onto one line in that context.
  const fmtItem = (e) =>
    `<a href="${e.url}">#${e.number}</a> by @${
      e.author
    } (idle ${e.idleDays.toFixed(1)}d)`;
  const fmtErrorItem = (e) => `#${e.number} (${e.phase}): ${e.message}`;
  const appendList = (s, items, fn) =>
    items.length ? s.addList(items.map(fn)) : s.addRaw("_none_").addEOL();

  let summary = core.summary
    .addHeading(title)
    .addRaw(`Mode: **${dryRun ? "dry-run" : "live"}**`)
    .addBreak();
  for (const line of summaryLines) {
    summary = summary.addRaw(line).addBreak();
  }
  summary = summary
    .addRaw(`Open issues considered: **${candidates.length}**`)
    .addList([
      `${ineligibleSummaryLabel}: ${skippedIneligible}`,
      `Skipped (exempt label): ${skippedExempt}`,
      `Skipped (in scope but younger than ${staleDays} days): ${skippedNotStaleYet}`,
      `Skipped (stale-labeled but not yet ${closeDays}d idle): ${skippedNotReadyToClose}`,
      `Marked stale this run: ${staled.length}`,
      `Closed this run: ${closed.length}`,
      `Un-staled this run (activity after label): ${unstaled.length}`,
      `Errors: ${errored.length}`,
    ])
    .addHeading("Marked stale", 3);
  summary = appendList(summary, staled, fmtItem);
  summary = summary.addHeading("Closed", 3);
  summary = appendList(summary, closed, fmtItem);
  summary = summary.addHeading("Un-staled (activity after label)", 3);
  summary = appendList(summary, unstaled, fmtItem);
  summary = summary.addHeading("Errors", 3);
  summary = appendList(summary, errored, fmtErrorItem);
  await summary.write();

  // Returned for test assertions only. The production caller (the workflow) discards this and
  // reads `core.summary` instead.
  return {
    dryRun,
    candidates: candidates.length,
    staled,
    closed,
    unstaled,
    errored,
    skippedIneligible,
    skippedExempt,
    skippedNotStaleYet,
    skippedNotReadyToClose,
    hitCap,
  };
}

module.exports = run;
// Exported for test boundary assertions so a future change surfaces in the tests that exercise the
// boundary, instead of silently passing because the test hardcoded the old value.
module.exports.SELF_ACTIVITY_EPSILON_MS = SELF_ACTIVITY_EPSILON_MS;
