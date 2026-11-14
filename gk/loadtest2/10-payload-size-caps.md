# Payload size caps on agent-submitted data

> **Template:** 🎟 Story · **Labels:** `story`, `reliability` · **Effort:** M · **Wave:** 3 · **Not targeting v5.0** (per PR review: 09 pagination is the prioritized breaking change; this ships additively — truncate-and-flag is new defensive behavior, not a contract break)

## Goal

| User story |
|:--|
| As a Fleet operator, |
| I want one misbehaving or hostile host to be unable to distort the whole pipeline, |
| so that a device reporting 400k packages or gigabyte log batches degrades only its own data, not the server. |

## What exists today

A global request body size limit already exists (`server.default_max_request_body_size`) — this issue is NOT starting from zero. The gaps are: (a) the global limit is coarse — within an accepted body, per-field/per-row data is unbounded (400k software rows fit under a body cap); (b) nothing is documented for agents; (c) rejection behavior may itself be a bug — see first task.

## FIRST TASK — verify osquery oversized-batch retry loop (possible live bug, no v5.0 dependency)

Per review (JordanMontgomery): if osquery's log batch exceeds the body limit and gets rejected, osquery may retry the same oversized batch from its buffer indefinitely — a permanent per-host retry loop that no cap tuning fixes. Verify with osquery-perf against a low `default_max_request_body_size`. If confirmed: the log endpoints likely need accept-and-truncate (or accept-and-drop-oversized-rows) semantics instead of rejection, and this escalates to its own expedited fix independent of the rest of this story.

## Changes

- **Define the limit table** (product + engineering sign-off), building on the existing `default_max_request_body_size`: per-agent-endpoint body limits; max rows per result batch; max software inventory entries per host; max string length for fields that land in indexed columns; max scheduled-query result size.
- **Server enforcement with two behaviors, chosen per data type:**
  - *Truncate + flag*: inventory-style data — store up to the cap, mark the host record as truncated so the UI/API can surface "inventory incomplete."
  - *Reject with 413*: only where the agent provably handles rejection without looping (pending the first-task verification above — osquery retrying rejected batches from its buffer forever would rule this out for log endpoints).
- **fleetd cooperation:** orbit respects the documented caps proactively where it owns the payload (split batches, cap collected rows). Known limitation: orbit cannot bound osquery-owned submissions (distributed results, logger traffic), so server-side truncate/flag remains the real guard for those paths.
- **Metrics:** `truncated_payloads_total{endpoint}` so support can find pathological hosts instead of discovering them via DB size.
- **Never `node_invalid`** on a 413 path (invariant from issue 03).

**Product checklist:**
- UI changes: "inventory truncated" indicator on host details (small; needs design).
- REST API changes: documented limits per endpoint, PR to reference docs release branch.
- Compatibility: additive defensive behavior (truncate-and-flag), release-notes worthy but not major-release-gated; limits chosen so <0.1% of real-world hosts are affected (validate against cloud telemetry before fixing numbers).

## Condition of satisfaction

- A simulated pathological host (osquery-perf flags already support inflated software counts) produces truncated-but-functional host data and zero measurable impact on other hosts' ingestion latency on the local rig.
- All caps documented; fleetd released with cooperating behavior.
