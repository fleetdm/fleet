# Fan-out safety: splay all-hosts operations and prove reconcile loops converge

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Wave:** 2

## Problem

Two related failure shapes in operations that touch every host:

1. **Thundering herd:** anything triggered "for all hosts at once" — MDM profile delivery, APNs/WNS push fan-out, config changes — causes every device to respond immediately. A profile push to 200k devices is a self-inflicted DDoS, and it's exactly what a large customer does on onboarding day one.
2. **Amplification loops:** reconcile flows where server action → device check-in → state change → server action again. If any loop fails to detect no-op and stop, it runs forever at fleet scale. This is the realistic "infinite loop" — distributed, with nobody having written `for {}`.

## Impact

High — these are triggered by routine admin actions, not edge cases. Herd spikes blow through the concurrency cap (issue 02) as a wall of legitimate traffic; a non-converging loop is a permanent background DDoS that grows with fleet size.

## Proposed fix

**Splay:**
- Inventory every all-hosts fan-out initiation point (MDM push fan-out, profile/declaration delivery scheduling, config-change notifications, script/software mass deployment).
- Each gets either rate-limited batched rollout (N pushes/sec, configurable) or client-side splay (deliver "act within [0, window)" rather than "act now") — choose per mechanism.
- Default windows sized so a full-fleet push at 400k hosts stays within one container's measured capacity.

**Convergence:**
- For each MDM/profile/declaration reconcile loop: document the terminating condition, and add a test that a steady-state host completes a full cycle with zero resulting writes and zero follow-up pushes.
- Add a detector metric: per-host command/push counts over a window; alert shape for "same host received > N identical commands in 24h" — catches non-convergence in the field before it's an outage.

## Condition of satisfaction

- Inventory table in the issue with the chosen mechanism per fan-out point.
- Local rig scenario: full-fleet profile push against seeded snapshot spreads over the configured window instead of spiking.
- Steady-state no-op tests merged for the MDM reconcile loops.

## Evidence

The osquery-perf `-start_period` flag already simulates splayed vs. simultaneous host activity for test purposes (`cmd/osquery-perf/agent.go`).
