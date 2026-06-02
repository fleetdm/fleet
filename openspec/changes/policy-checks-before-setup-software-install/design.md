## Context

Setup experience for Windows/Linux is a server-driven, polled state machine. The host calls `/setup_experience/init`
(`ee/server/service/orbit.go:343-377`) which runs `EnqueueSetupExperienceItems`
(`server/datastore/mysql/setup_experience.go:179-362`), inserting one `setup_experience_status_results` row per item, all
`pending`. The host then polls `/setup_experience/status`, each poll calling `SetupExperienceNextStep`
(`ee/server/service/setup_experience.go:178-348`), which installs software items one at a time (FIFO by display name), marking
each `running` then `success`/`failure`. On Windows Autopilot/OOBE-ESP the device is held at the Enrollment Status Page until
all items reach a terminal state. The status enum is `pending|running|success|failure|cancelled`
(`server/fleet/setup_experience.go:8-15`); `success`/`failure` are terminal (and `cancelled` is also final, used when
`requireAllSoftware` cancels remaining items, though `IsTerminalStatus()` itself returns only success/failure).

Policies are osquery SQL queries. Results land via `SubmitDistributedQueryResults` ->
`RecordPolicyQueryExecutions` (`server/datastore/mysql/policies.go:662-796`), which upserts `policy_membership(host_id,
policy_id, passes)` and triggers automations for newly-failing policies (`processSoftwareForNewlyFailingPolicies` /
`server/service/osquery.go:2037+`). A policy's install-software automation is a foreign key on the policy row:
`policies.software_installer_id`. (Policies can also have run-script automations via `policies.script_id`, but those are out of
scope here.)

The load-bearing fact: **`policyQueriesForHost` returns nothing for hosts in setup experience today**
(`server/service/osquery.go:922-929`). Policies never run during setup, so automations never fire during setup. This change
must surgically reverse that for associated policies only.

## Goals / Non-Goals

**Goals**
- Windows/Linux setup-experience software with an associated install-software policy is skipped when the policy passes.
- When the policy fails, the policy's existing install-software automation runs during setup, and the setup-experience item
  reaches a terminal state that reflects that automation.
- Every item still reaches a terminal state (no hang); no double-install; no behavior change for non-associated software or for
  macOS/iOS/iPadOS.

**Non-Goals**
- No on-device UI changes (no fleetd/osquery changes). No REST/YAML/fleetctl/activity/permission changes.
- No explicit policy-assignment surface.
- macOS/Apple setup-experience software is untouched.
- Run-script automations are out of scope (removed from the test plan). Only install-software policies gate setup software.

## Decisions

### Association is implicit via `policies.software_installer_id`, resolved at enqueue time

A Windows/Linux setup-experience item (`software_installers.install_during_setup = 1`) is policy-gated iff a policy scoped to the
host's team has `software_installer_id` equal to that installer. `EnqueueSetupExperienceItems` already joins `software_installers`
for the host's team; extend it to `LEFT JOIN policies p ON p.software_installer_id = si.id` with the team predicate and store the
matched `policy_id` on the new `setup_experience_status_results.policy_id` column. Resolving at enqueue time (rather than per-poll)
fixes the gate for the duration of this setup run and gives us the exact set of policies to un-skip.

**Global / "No team" hosts.** Global-scope policies have `policies.team_id IS NULL`, and "No team" hosts have `team_id` NULL /
`teamID = 0`. The team predicate must therefore map `teamID = 0` to `p.team_id IS NULL` (e.g. `p.team_id <=> NULLIF(?, 0)`), not
`p.team_id = 0`, or the gate would never apply to global policies / No-team hosts. Follow the existing null-safe team-scoping
pattern used elsewhere in `server/datastore/mysql/policies.go`.

If more than one policy targets the same installer, pick deterministically (lowest `policy_id`) and log the ambiguity. (Open
question: confirm product expectation for multi-policy.)

### Un-skip only the associated policies during setup

Change `policyQueriesForHost` (`server/service/osquery.go:915-938`) so that, for a host in setup experience, instead of
returning `nil` it returns only the policy queries whose `policy_id` is in the set of associated policies for that host's
non-terminal setup-experience items. Add a datastore method to fetch that set (or a variant of `PolicyQueriesForHost` filtered
by policy IDs). All other policies remain skipped during setup -> unrelated automations still do not fire mid-setup.

To get a *fresh* result quickly (test plan: "this run's result, not stale"), rely on the fact that a freshly enrolled host has
`policy_membership` NULL / `GetHostPolicyReportedAt` zero, so `shouldUpdate` is already true and the associated policy is sent on
the host's next distributed-query checkin. For re-enrollment where a stale result exists, set `RefetchRequested` (or reset the
host's policy-reported timestamp) at enqueue time so the associated policies are re-sent promptly. No new agent behavior: osquery
runs the query natively and reports through the existing channel.

### `SetupExperienceNextStep` gates the picked item on the policy result; the policy automation performs the action

For the next pending software item (`ee/server/service/setup_experience.go:233-248`):

- `policy_id == NULL` -> unchanged: `InsertSoftwareInstallRequest(..., ForSetupExperience: true)`, status `running`.
- `policy_id != NULL` -> new two-phase handling, item stays `running` (no new enum) with an internal phase marker:
  - **Phase "awaiting policy":** if there is no post-enrollment `policy_membership` row for `(host, policy_id)`, ensure the
    policy is queued (handled by the un-skip + refetch above), keep the item `running`, return `(false, nil)`.
  - When the result is present:
    - `passes = true` -> set status `success`, no install. This is the **skipped** outcome (reuses `success` per the product
      decision; no schema/API enum change). Record the reason in the existing `error`/detail column for observability if useful.
    - `passes = false` -> the policy's install-software automation fires through `RecordPolicyQueryExecutions` ->
      `processSoftwareForNewlyFailingPolicies`. Link the resulting execution to the item and move to phase "awaiting automation."
  - **Phase "awaiting automation":** the item tracks the automation's `host_software_installs` row, stamped with `policy_id` for
    this host. When that install reaches a terminal state, mirror it into the item's `success`/`failure`. Existing
    setup-experience completion handling already reacts to install completion callbacks (`server/service/orbit.go`), so the
    linkage is what makes setup wait correctly.

This honors the chosen model ("policy automation runs it"): setup experience does not itself install policy-gated software; it
observes the policy and the automation it triggers.

### State tracking: internal columns only, no API enum change

`setup_experience_status_results` gains one column:
- `policy_id BIGINT UNSIGNED NULL` (associated policy; `json:"-"`).

No column is needed to record the awaited execution: the row already has `host_software_installs_execution_id`
(`server/fleet/setup_experience.go:44`), which is reused to point at the install-software automation's `host_software_installs`
row. Because `policy_id` is orthogonal to the existing mutually-exclusive value columns (`software_installer_id` /
`vpp_app_team_id` / `setup_experience_script_id`), `IsValid()` (`server/fleet/setup_experience.go:67-95`) needs at most a minor
tweak to allow `policy_id` alongside `software_installer_id`; no run-script script-ref handling is required. The new column is
`json:"-"` -> no API change.

A migration adds this column (`make migration name=AddPolicyGateToSetupExperienceResults`). Down is a no-op per repo
convention; include a migration test.

### Scope guard: Windows/Linux only

Policy gating applies only when the item is a software *installer* (`IsForSoftwarePackage()`, `software_installer_id != nil`) and
the host platform is Windows or Linux. VPP items (`VPPAppTeamID`) and Apple platforms keep today's unconditional path. The
enqueue-time association join is only run for the Windows/Linux installer branch
(`server/datastore/mysql/setup_experience.go:198-236`), so Apple rows never get a `policy_id`.

### Frontend: copy only

`InstallSoftware.tsx:227-234` already branches `selectedPlatform === "windows" || selectedPlatform === "linux"`. Change the
Windows/Linux string to the wireframe copy; leave the macOS/iOS/iPadOS/Android string. Update the regex assertions in
`InstallSoftware.tests.tsx`. The `InstallSoftwareTable` help text (`InstallSoftwareTable.tsx:79-86`) is a separate string; only
change it if the wireframe requires (confirm against Figma).

## Open Questions

1. **Timing during the Windows ESP hold.** The policy result depends on an osquery distributed-query checkin while the device is
   held at the ESP. Confirm the held device still pulls/answers distributed queries promptly (the spike measures init->result
   latency) and that the policy-automation install (normal priority, *not* `ForSetupExperience`) actually executes during the
   hold rather than queuing behind it. If it does not, fall back to having setup experience itself enqueue the automation with
   `ForSetupExperience` priority.
2. **"Newly failing" semantics on first enrollment.** `processSoftwareForNewlyFailingPolicies` fires on a pass->fail transition.
   Confirm NULL(no prior membership)->fail counts as newly-failing so the automation fires on the first result; and decide the
   re-enrollment case where a stale `fail` exists and `continuous_automations_enabled` is off (automation may not re-fire). The
   spike resolves this; a safe fallback is for setup experience to invoke the policy's configured automation explicitly when it
   observes `passes=false`, guaranteeing the action runs exactly once.
3. **Policy scope excludes the host.** If the associated policy's platform/label scope excludes the host, the policy query is
   never delivered and no result arrives. Define the terminal behavior (recommended: treat "policy not applicable to host" as
   "no gate" and install normally, so the item never hangs). Confirm against `PolicyQueriesForHost` label/platform filtering
   (`server/datastore/mysql/policies.go:1151-1180`).
4. **Multiple policies on one installer.** Confirm deterministic selection is acceptable vs. an error.

## Spike (de-risk before main implementation)

The async cross-channel timing (orbit poll vs. osquery distributed query) and the "newly failing" automation semantics are the
two highest risks. A short spike on a feature branch should, against a real Windows ESP host and a Linux fleetd host:
- Measure latency from `/setup_experience/init` to a fresh `policy_membership` result with the associated policy un-skipped.
- Confirm whether a normal-priority policy-automation install runs while the device is held at the ESP.
- Confirm NULL->fail fires the automation, and capture re-enrollment/`continuous_automations_enabled=off` behavior.
Output: `openspec/changes/policy-checks-before-setup-software-install/spike-results.md` (kept in-tree, since `ai/` is gitignored
and would not travel between environments), which may revise open questions 1 and 2 (notably whether setup experience triggers
the automation itself with `ForSetupExperience` priority).

## Migration Plan

Additive nullable `policy_id` column on `setup_experience_status_results`; no backfill (only new setup runs are gated; in-flight rows have
`policy_id = NULL` and behave as today). Rollback is the no-op down migration; gating is inert when `policy_id` is NULL.
