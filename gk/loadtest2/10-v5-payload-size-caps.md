# v5.0: Payload size caps on agent-submitted data (breaking)

> **Template:** 🎟 Story · **Labels:** `story`, `reliability` · **Effort:** M · **Milestone:** v5.0.0 · **Wave:** 3

## Goal

| User story |
|:--|
| As a Fleet operator, |
| I want one misbehaving or hostile host to be unable to distort the whole pipeline, |
| so that a device reporting 400k packages or gigabyte log batches degrades only its own data, not the server. |

## Why v5.0

Today the implicit contract is "the server stores whatever agents send." Introducing enforced caps changes observable behavior (truncated inventory, rejected oversized batches), so the major release is the moment to define the limits product-wide, document them, and have fleetd cooperate — rather than adding inconsistent per-field caps reactively after incidents.

## Changes

- **Define the limit table** (product + engineering sign-off), e.g.: max request body per agent endpoint; max rows per result batch; max software inventory entries per host; max string length for fields that land in indexed columns; max scheduled-query result size.
- **Server enforcement with two behaviors, chosen per data type:**
  - *Truncate + flag*: inventory-style data — store up to the cap, mark the host record as truncated so the UI/API can surface "inventory incomplete."
  - *Reject with 413*: submission-style data (log batches) — agent splits or drops per its own buffer policy; osquery already handles rejected log batches by retrying from its buffer.
- **fleetd cooperation:** orbit respects the documented caps proactively (split batches, cap collected rows) so well-behaved agents never hit server rejection.
- **Metrics:** `truncated_payloads_total{endpoint}` so support can find pathological hosts instead of discovering them via DB size.
- **Never `node_invalid`** on a 413 path (invariant from issue 03).

**Product checklist:**
- UI changes: "inventory truncated" indicator on host details (small; needs design).
- REST API changes: documented limits per endpoint, PR to reference docs release branch.
- Compatibility: breaking in v5.0.0 — release notes; limits chosen so <0.1% of real-world hosts are affected (validate against cloud telemetry before fixing numbers).

## Condition of satisfaction

- A simulated pathological host (osquery-perf flags already support inflated software counts) produces truncated-but-functional host data and zero measurable impact on other hosts' ingestion latency on the local rig.
- All caps documented; fleetd released with cooperating behavior.
