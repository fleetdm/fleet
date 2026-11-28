# Scale checklist in story/PR flow + load-test gate for hot-path changes

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering`, `#g-product` (template changes) · **Effort:** S · **Wave:** 1

## Problem

Scale costs are evaluated ad hoc. The story template already asks about load testing (`.github/ISSUE_TEMPLATE/story.md` — "Load testing", "Pre-QA load test", "osquery-perf improvements" checkboxes), but nothing makes an engineer answer the design questions that determine scale cost *before* building: is this on the hot path, what's the write frequency, does a table grow, what fans out, what retries. Every preventable item in this epic shipped past review because nobody was prompted to ask.

## Impact

Cheapest issue in the set, and the one that stops the backlog from regrowing. The audits (08, 11, 12, 13, 14) clean up the past; this prevents the future.

## Proposed fix

**Scale checklist** added to the story template's Engineering section (and referenced from `.claude/rules/` so AI-assisted work applies it too). One line each, "N/A" allowed with justification:

- Hot path: does this touch agent check-in, MDM check-in, or distributed read/write? What's the added cost × 400k hosts × frequency?
- Writes: any new write on check-in? Is it conditional on change / batched?
- Growth: any table growing with activity? Retention ships in the same PR (issue 11 policy).
- Fan-out: anything triggered for all hosts at once? What's the splay/rollout mechanism (issue 07)?
- Bounds: new list endpoint paginated (issue 09)? New agent input capped (issue 10)? IN lists chunked (issue 12)?
- Retry: are the new endpoints idempotent under agent retry?
- Background job: meets the four requirements from issue 08?
- Endpoint tier declared (issue 05, once it lands)?

**Load-test gate:** sharpen the existing "Pre-QA load test" checkbox with a concrete bar — any "yes" on the hot-path question requires at least a local rig comparison run (issue 18) before QA, with the budget-delta table pasted into the story; full terraform runs reserved for high-risk changes, per current practice.

## Condition of satisfaction

- Template PR merged; handbook section updated to explain each question with examples.
- Checklist live for one full sprint with the group answering it on every story (spot-check retro).
- `.claude/rules/fleet-go-backend.md` references the checklist so it applies at code-writing time, not just planning time.
