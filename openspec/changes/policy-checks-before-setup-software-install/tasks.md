## 0. Spike (de-risk before main implementation)

See `design.md` > Spike for the full plan and success criteria. Record findings in-tree at
`openspec/changes/policy-checks-before-setup-software-install/spike-results.md` (not `ai/`, which is gitignored), then fold the
conclusions back into `design.md` decisions 2 and 3.

- [ ] 0.1 On a feature branch, un-skip a single associated policy during setup and measure init->fresh-`policy_membership`
      latency against a Windows ESP-held host and a Linux fleetd host.
- [ ] 0.2 Confirm whether a normal-priority (non-`ForSetupExperience`) policy-automation install executes while the device is
      held at the Windows ESP; capture the result.
- [ ] 0.3 Confirm NULL->fail triggers `processSoftwareForNewlyFailingPolicies`; capture re-enrollment +
      `continuous_automations_enabled=off` behavior.
- [ ] 0.4 Revise `design.md` decisions 2/3 if needed (most likely: setup experience triggers the policy's automation itself with
      `ForSetupExperience` priority instead of relying on the async newly-failing pipeline).

## 1. Product clarification

- [ ] 1.1 Confirm behavior when the policy scope excludes the host, and the multiple-policies-per-installer case (open
      questions 3, 4). Run-script automations are out of scope (removed from the test plan); only install-software policies gate
      setup software.

## 2. Database migration

- [ ] 2.1 `make migration name=AddPolicyGateToSetupExperienceResults`. Add nullable `policy_id BIGINT UNSIGNED NULL`. No backfill;
      down is a no-op.
- [ ] 2.2 Migration test under `server/datastore/mysql/migrations/tables/`: `applyUpToPrev` -> seed -> `applyNext` -> assert the
      new column exists.

## 3. Backend types

- [ ] 3.1 Add `PolicyID *uint` (`db:"policy_id" json:"-"`) to `SetupExperienceStatusResult` (`server/fleet/setup_experience.go:39-65`).
- [ ] 3.2 Confirm `IsValid()` (`server/fleet/setup_experience.go:67-95`) accepts a software row carrying `policy_id` (orthogonal
      to the mutually-exclusive value columns); adjust only if needed and add a table-driven unit test for the new combination.
      The awaited install reuses the existing `host_software_installs_execution_id` column (no new ref column).

## 4. Datastore: association + un-skip + result lookup

- [ ] 4.1 Extend `EnqueueSetupExperienceItems` (`server/datastore/mysql/setup_experience.go:198-236`, Windows/Linux installer
      branch only) to `LEFT JOIN policies ON software_installer_id` for the host's team and persist the matched `policy_id`
      (deterministic lowest id on ties; log ambiguity). Map `teamID = 0` to `policies.team_id IS NULL` so global policies gate
      No-team hosts (use the existing null-safe team-scoping pattern; do NOT compare `team_id = 0`). Do not touch the VPP / macOS
      branches.
- [ ] 4.2 Add a datastore method to return the associated `policy_id`s for a host's non-terminal setup-experience items (drives
      the selective un-skip).
- [ ] 4.3 Add/extend a datastore method to fetch the freshest `policy_membership` pass/fail for `(host_id, policy_id)` including
      its `updated_at`, so the service can distinguish post-enrollment results from stale ones.
- [ ] 4.4 If decision 0.4 requires prompt re-evaluation on re-enrollment, set `RefetchRequested` / reset policy-reported
      timestamp for the associated policies at enqueue time.
- [ ] 4.5 Mock + interface wiring in `server/fleet/datastore.go`; run `go test ./server/service/` so uninitialized mocks don't
      crash other suites.

## 5. Service: un-skip associated policies during setup

- [ ] 5.1 Change `policyQueriesForHost` (`server/service/osquery.go:915-938`): for a host in setup experience, return only the
      policy queries for the host's associated setup-experience policies instead of `nil`. All other policies stay skipped.
- [ ] 5.2 Unit-test: host in setup with associated policy -> only that policy query returned; host in setup with no associated
      policy -> still empty; host not in setup -> unchanged.

## 6. Service: gate the install in SetupExperienceNextStep

- [ ] 6.1 In `SetupExperienceNextStep` (`ee/server/service/setup_experience.go:233-248`), branch the picked software item on
      `PolicyID`. `NULL` -> existing `ForSetupExperience` install path, unchanged.
- [ ] 6.2 Policy-gated, no fresh result yet -> keep item `running` (awaiting-policy phase), return `(false, nil)`. Ensure the
      associated policy is queued (from task 4/5).
- [ ] 6.3 Policy-gated, `passes=true` -> set item `success` with no install (the skipped outcome); record reason in the detail
      column. Verify no `InsertSoftwareInstallRequest` is issued.
- [ ] 6.4 Policy-gated, `passes=false` -> link the item to the install-software automation's `host_software_installs` execution
      and move to awaiting-automation phase. Per task 0.4, either observe the automation fired by `RecordPolicyQueryExecutions`,
      or trigger the policy's automation explicitly. Guarantee exactly one execution (no double-install).
- [ ] 6.5 Awaiting-automation phase -> mirror the linked install's terminal state into the item `success`/`failure`. Confirm the
      existing install completion callback (`server/service/orbit.go`) advances the item.
- [ ] 6.6 Confirm `requireAllSoftware` cancellation
      (`MaybeCancelPendingSetupExperienceSteps`) and the retry count
      (`setupExperienceSoftwareInstallsRetries`, `server/datastore/mysql/software_installers.go:22`) still hold for the
      automation-driven install.

## 7. Frontend: copy

- [ ] 7.1 Update the Windows/Linux description string in
      `frontend/pages/ManageControlsPage/SetupExperience/cards/InstallSoftware/InstallSoftware.tsx:227-234` to the Figma copy;
      leave macOS/iOS/iPadOS/Android string unchanged. Check the `InstallSoftwareTable` help text against Figma
      (`InstallSoftwareTable.tsx:79-86`) and update only if required.
- [ ] 7.2 Update `InstallSoftware.tests.tsx` copy assertions. Run `yarn test` for the affected suites.

## 8. Tests (backend)

- [ ] 8.1 Datastore integration (`MYSQL_TEST=1`): enqueue records `policy_id` for an associated Windows/Linux installer; null
      for non-associated and for macOS/VPP.
- [ ] 8.2 Service/integration (`MYSQL_TEST=1 REDIS_TEST=1`): full flows from the issue test plan -- policy pass -> skipped;
      policy fail -> install-software automation installs; not installed -> installs; no-policy -> always installs; mixed
      batch -> only failing/no-policy install, passing skipped, all items terminal (no hang).
- [ ] 8.3 Edge cases: fresh host with no prior policy results acts on this run's result; no double-install; policy scope
      excludes host -> install normally (no hang); macOS app with an associated installer-policy still installs.

## 9. Docs / feature guide

- [ ] 9.1 Update `articles/windows-linux-setup-experience.md` "Install software" section: an associated policy is run first; the
      install is skipped when it passes; the policy's automation runs on failure. Sentence case, wrap at 140. Leave
      `articles/setup-experience.md` (macOS) unchanged.
- [ ] 9.2 Add a changes/ entry per repo convention; confirm no audit-log/activity doc changes are needed (none expected).

## 10. Lint + finalize

- [ ] 10.1 `make lint-go-incremental` and `make lint-js`; resolve findings.
- [ ] 10.2 Engineer completes the issue test plan and comments confirmation on #45309.
