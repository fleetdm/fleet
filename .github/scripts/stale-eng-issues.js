// Marks open engineering-initiated issues (label `~engineering-initiated`) as stale after 1y of
// inactivity, and closes them after 14 more days. Invoked by `actions/github-script` from
// `.github/workflows/close-stale-eng-initiated-issues.yml`.
//
// This is a thin wrapper over `stale-issues-core.js`: it provides label-based eligibility and the
// eng-initiated thresholds and wording. The scanning, labeling, closing, and un-staling logic all
// live in the core. The sibling `stale-fleetie-issues.js` wraps the same core with author-based
// eligibility.
//
// Why this replaced `actions/stale`: that action posts a single static stale comment with no way to
// template the issue author, so it could not @-mention the author. The core lets us build the
// comment per issue. Behavior otherwise matches the previous `actions/stale` config (365d to stale,
// 14d to close, remove-stale-when-updated).
//
// Inputs (env):
//   DRY_RUN         'true' to log candidates without writing (read by the core).
//   MAX_OPERATIONS  Cap on API write operations per run. Default 400. `0` disables writes / dry-runs (read by the core).
//
// Exports: `async function run({ github, context, core })`. Returns a summary object for tests.

"use strict";

const core_run = require("./stale-issues-core.js");

const STALE_DAYS = 365;
const CLOSE_DAYS = 14;
const STALE_LABEL = "stale";
const ELIGIBLE_LABEL = "~engineering-initiated";

const staleMessage = (author) =>
  `@${author} this issue is stale because it has been open for 365 days with no activity. ` +
  "Please update the issue if it is still relevant; otherwise it will be closed in 14 days.";
const CLOSE_MSG =
  "This issue was closed because it has been inactive for 14 days since being marked as stale.";

async function run({ github, context, core }) {
  const result = await core_run({
    github,
    context,
    core,
    config: {
      title: "Eng-initiated stale-issue closer",
      staleDays: STALE_DAYS,
      closeDays: CLOSE_DAYS,
      staleLabel: STALE_LABEL,
      isEligible: (issue) =>
        (issue.labels || []).some(
          (l) =>
            (typeof l === "string" ? l : (l && l.name) || "").toLowerCase() ===
            ELIGIBLE_LABEL.toLowerCase()
        ),
      staleMessage,
      closeMessage: () => CLOSE_MSG,
      ineligibleSummaryLabel: `Skipped (no ${ELIGIBLE_LABEL} label)`,
    },
  });
  return result;
}

module.exports = run;
// Exported for test boundary assertions so a future policy change surfaces in the boundary tests
// instead of silently passing on a hardcoded old value.
module.exports.STALE_DAYS = STALE_DAYS;
module.exports.CLOSE_DAYS = CLOSE_DAYS;
module.exports.ELIGIBLE_LABEL = ELIGIBLE_LABEL;
module.exports.SELF_ACTIVITY_EPSILON_MS = core_run.SELF_ACTIVITY_EPSILON_MS;
