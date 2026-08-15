## 1. Scope classification

- [ ] 1.1 Add a scope type and classifier in `server/fleet/` that returns device or user scope for Windows profile bytes, built on `ExtractLocURIsFromProfileBytes` over `WrapSCEPProfileInAtomic`-normalized bytes, treating any canonicalized target under `./User/` as user-scoped.
- [ ] 1.2 Unit test the classifier against device-only, user-only, and mixed profiles, plus the obfuscation cases that motivated PR #49715: LocURI text split by CDATA, comments, and nested elements, and the scope-less `./Vendor/MSFT/` spelling.
- [ ] 1.3 Create the migration with `make migration name=AddWindowsProfileScopeAndUserContext` adding a scope column to `mdm_windows_configuration_profiles` and the observed user-context value plus timestamp to `mdm_windows_enrollments`.
- [ ] 1.4 Backfill the scope column for existing profiles in the migration Up function, using the same classifier, and add a migration test covering device, user, and mixed rows.
- [ ] 1.5 Set the scope column on the profile create and update paths so newly saved profiles carry it, and confirm the batch and GitOps entry points are both covered.

## 2. Observing user context from the OMA-DM session

- [ ] 2.1 Add the `com.microsoft/MDM/LoginStatus` alert type constant and the `user`, `others`, `none` value constants to `server/mdm/microsoft/syncml/syncml.go`.
- [ ] 2.2 Handle `syncml.CmdAlertClientEvent` ("1224") in `processIncomingAlertsCommands` and route LoginStatus items to a new handler; leave other 1224 alert types as no-ops. Read the alert from the authenticated message: the device repeats every alert in MsgID 2 after Fleet's challenge, so the early return on the challenge path stays as it is.
- [ ] 2.3 Add the datastore method that persists the observed value and timestamp on the enrollment, and run `go test ./server/service/` afterwards so uninitialized mocks do not crash other tests.
- [ ] 2.4 Ensure the observation is recorded even when the message is not otherwise persisted, since `ShouldBeTracked` returns false for a SyncHdr-only status and `MDMWindowsSaveResponse` is skipped for that message.
- [ ] 2.5 Unit test the alert handler for each value, for a missing alert (previous observation preserved), and for a malformed alert item.

## 3. Delivery gating in the reconciler

- [ ] 3.1 Add a helper that resolves an enrollment's user-context state to present, can-arrive, or cannot-arrive, using the persisted observation and `microsoft_mdm.IsValidUPN(enroll_user_id)`.
- [ ] 3.2 In the Windows profile reconcile path, skip enqueueing user-scoped profiles for can-arrive enrollments, keeping the profile row Pending with a detail explaining that Fleet is waiting for first sign-in and no command UUID.
- [ ] 3.3 Verify the held row shape does not confuse the install and remove delta queries or `ReconcileWindowsProfilesStatus`, so a held profile is not treated as stuck or as needing a resend loop.
- [ ] 3.4 Fail user-scoped profiles fast for cannot-arrive enrollments, with a detail stating that user-channel profiles require an enrollment carrying a user identity, and ensure the reconciler does not re-enqueue them on later ticks.
- [ ] 3.5 Treat mixed-scope profiles as user-scoped throughout, so they hold or fail as a unit.
- [ ] 3.6 Confirm device-scoped enqueue behavior is untouched by diffing reconciler behavior for a device-only profile set before and after the change.

## 4. Retry accounting

- [ ] 4.1 In `MDMWindowsSaveResponse`, skip the `retries` increment for a user-context rejection when the enrollment is in the can-arrive state, keeping existing accounting for all other states.
- [ ] 4.2 Key the exemption on the enrollment's user-context state, not on the returned status code. A 405 means "user context not initialized" during OOBE and "node unsupported on this Windows edition" with a user signed in, so the code alone cannot carry the decision. Use per-LocURI statuses only to tell a rejection apart from an unrelated `Atomic` 507 rollback.
- [ ] 4.3 Add a regression test proving the nested-418 path still works: a SCEP `Atomic` returning 507 with a nested `Add` 418 and sibling 216s must still go through `handleResendingAlreadyExistsCommands` and normal retry accounting.
- [ ] 4.4 Add a test proving no resend loop exists in the cannot-arrive state.

## 5. Integration tests

- [ ] 5.1 Extend the Windows MDM test client so the LoginStatus alert value is settable per session; it already emits `user` in `pkg/mdm/mdmtest/windows.go`, so add `none` and `others` and the ability to omit the alert.
- [ ] 5.2 Integration test the full defer-then-deliver flow: assign a user-scoped profile with no user context, assert Pending with the hold detail and no queued command, then report `LoginStatus=user` and assert the profile delivers and verifies.
- [ ] 5.3 Integration test the cannot-arrive path: a non-UPN enrollment reports Failed with the explanatory detail and no repeated sends.
- [ ] 5.4 Integration test that a device-scoped profile delivers normally in all three user-context states.
- [ ] 5.5 Add a property test asserting that any profile whose delivered SyncML contains a canonicalized `./User/` target is classified user-scoped, alongside the existing Windows reconcile property tests.

## 6. Hardware verification

The 2026-08-14 Autopilot capture already answered whether the alert is emitted (yes, every session, MsgID 1 and 2, `others` during OOBE and `user` after sign-in), so the remaining runs are about the paths it did not exercise.

- [ ] 6.1 Reproduce the bug itself, which the first Autopilot run did not: linkage must precede user context. Easiest case is a device whose host record already exists in Fleet so the reverse link resolves without waiting for a fresh fleetd enrollment. Capture the status a `./User/` write receives during the `others` window, which is still unmeasured on an Entra enrollment.
- [ ] 6.2 Verify the fix end to end on that same setup: the profile stays Pending through OOBE with the hold detail, then delivers on first sign-in without operator action.
- [ ] 6.3 Run a self-deploying or pre-provisioning Autopilot flow, where no user ever signs in. Confirm LoginStatus reports `none`, and decide from the result whether a hold that never releases should eventually convert into an explanatory failure.
- [ ] 6.4 Re-run the `bl-fix-test` non-Entra case and confirm the new behavior is a fast, explanatory failure rather than the current 500, 507, one wasted retry sequence. Baseline for comparison: responses 565702 and 565703 in the dev database, 26 seconds apart, identical signature.
- [ ] 6.5 For any user-scope test profile, use a node that is supported on the device's Windows edition. `Experience/AllowWindowsSpotlight` is unsupported on Pro and returns 405 regardless of user context; `Experience/AllowTailoredExperiencesWithDiagnosticData` is user-scoped and Pro-supported and was verified working.

## 7. Separate issues found during measurement

Neither belongs in this change, but both were found by the Autopilot capture and should be filed so they are not lost.

- [ ] 7.1 File an issue: `docs/solutions/windows/configuration-profiles/disable Windows Spotlight features – [AllowWindowsSpotlight].xml` targets a node Microsoft documents as unsupported on Windows Pro. On Pro it returns 405, burns its single retry about 90 seconds later, and lands terminally Failed with no explanation. Fleet ships this profile as a recommended solution.
- [ ] 7.2 File an issue: during ESP, one session ran 31 messages in roughly 100 seconds, with messages 3 onward repeating an identical bundle of two `Add=418` and six `Replace=200` every few hundred milliseconds. That is `handleResendingAlreadyExistsCommands` re-issuing commands the device had already acknowledged. It resolved on its own, and task 4.3 must not regress it further.

## 8. Documentation and release notes

- [ ] 8.1 Document the behavior in `articles/custom-os-settings.md` and `articles/creating-windows-csps.md`: which enrollment types support user-channel profiles, Pending until first sign-in, and the mixed-scope hold rule.
- [ ] 8.2 Document the gate and where it sits in the management session in `docs/Contributing/architecture/mdm/windows-mdm-architecture.md`.
- [ ] 8.3 Add a note to the header comment of `docs/solutions/windows/configuration-profiles/install Okta attestation certificate - [Bundle].xml`, since that profile is the one customers hit this with.
- [ ] 8.4 Add a `changes/` entry describing the fix for the release notes.

## 9. Gates

- [ ] 9.1 Run `make lint-go-incremental` and fix findings.
- [ ] 9.2 Run the affected suites: `go test ./server/fleet/...`, `MYSQL_TEST=1 go test ./server/datastore/mysql/...`, and `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...`.
- [ ] 9.3 Run `make lint-go` before opening the PR, and build the PR description from `.github/pull_request_template.md`.
