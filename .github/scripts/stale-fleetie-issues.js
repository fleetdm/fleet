// Marks open issues authored by current/former Fleeties as stale after 2y of inactivity, and closes
// them after 14 more days. Exempts `bug`, `:product`, and `customer-*` labels. Invoked by
// `actions/github-script` from `.github/workflows/close-stale-fleetie-initiated-issues.yml`.
//
// This is a thin wrapper over `stale-issues-core.js`: it provides the author-based eligibility check
// (issue author is a current/former Fleetie) and the Fleetie-specific thresholds and wording. The
// scanning, labeling, closing, and un-staling logic all live in the core. The sibling
// `stale-eng-issues.js` wraps the same core with label-based eligibility.
//
// Inputs (env):
//   FLEETIE_HANDLES_FILE  Path to newline-delimited lowercased GitHub usernames (built by `build-fleetie-handles.js`).
//   DRY_RUN               'true' to log candidates without writing (read by the core).
//   MAX_OPERATIONS        Cap on API write operations per run. Default 400. `0` disables writes / dry-runs (read by the core).
//
// Exports: `async function run({ github, context, core })`. Returns a summary object for tests.

"use strict";

const fs = require("node:fs");
const core_run = require("./stale-issues-core.js");

const STALE_DAYS = 730;
const CLOSE_DAYS = 14;
const STALE_LABEL = "stale";

const staleMessage = (author) =>
  `@${author} this issue is stale because it was opened by a current or former Fleetie and has had ` +
  "no activity for 2 years. Please update the issue if it is still relevant; otherwise it " +
  "will be closed in 14 days.";
const CLOSE_MSG =
  "This issue was closed because it received no further activity for 14 days after being marked stale. " +
  "Any comment would have removed the stale label and prevented closure.";

const isExempt = (name) => {
  const lower = (name || "").toLowerCase();
  return (
    lower === "bug" || lower === ":product" || lower.startsWith("customer-")
  );
};

async function run({ github, context, core }) {
  const handles = new Set(
    fs
      .readFileSync(process.env.FLEETIE_HANDLES_FILE, "utf8")
      .split("\n")
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean)
  );
  core.info(`Loaded ${handles.size} Fleetie handles.`);

  const result = await core_run({
    github,
    context,
    core,
    config: {
      title: "Fleetie stale-issue closer",
      staleDays: STALE_DAYS,
      closeDays: CLOSE_DAYS,
      staleLabel: STALE_LABEL,
      isEligible: (issue) =>
        handles.has(((issue.user && issue.user.login) || "").toLowerCase()),
      isExempt,
      staleMessage,
      closeMessage: () => CLOSE_MSG,
      ineligibleSummaryLabel: "Skipped (non-Fleetie author)",
      summaryLines: [`Fleetie handles loaded: **${handles.size}**`],
    },
  });

  // Preserve the historical field name for callers/tests that predate the shared core.
  return { ...result, skippedNonFleetie: result.skippedIneligible };
}

module.exports = run;
// Exported for test boundary assertions so a future policy change (e.g. STALE_DAYS 730 -> 365)
// surfaces in the boundary tests instead of silently passing on a hardcoded old value.
module.exports.STALE_DAYS = STALE_DAYS;
module.exports.CLOSE_DAYS = CLOSE_DAYS;
module.exports.SELF_ACTIVITY_EPSILON_MS = core_run.SELF_ACTIVITY_EPSILON_MS;
