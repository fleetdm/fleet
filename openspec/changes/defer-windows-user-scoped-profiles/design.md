## Context

Fleet has no notion of channel scope for Windows profiles. Unlike Apple profiles, which carry `fleet.PayloadScopeUser`, a Windows profile is just SyncML, and `ReconcileWindowsProfiles` enqueues it for every applicable host as soon as it exists. Whether its LocURIs target `./Device/` or `./User/` never enters the decision.

On Windows there is no separate user session to wait for. Per [MS-MDM], the OMA-DM client normally runs a "mixed" session in which the server may send both device and user settings in one exchange (only AVD splits device and user sessions). So the gate cannot be "wait for a user-context connection to arrive"; it has to be a per-session read of what the device reports.

Windows reports it two ways, both at the start of each session (DM package #1):

1. Generic alert `1224` with `<Type>com.microsoft/MDM/LoginStatus</Type>`, values `user`, `others`, `none`. Microsoft documents this under the heading "Determine when a user is logged in through polling".
2. Alert `1224` with type `com.microsoft/MDM/AADUserToken`, plus an `Authorization: Bearer <Entra user token>` HTTP header. Microsoft's instruction to MDM vendors is explicit: "The management server should check if the token is missing and only send device policies in such case."

Fleet already receives the first signal and throws it away. `processIncomingAlertsCommands` switches on `1201`, `1200`, and `1226`; `syncml.CmdAlertClientEvent = "1224"` is defined but never handled. Fleet's own test client already emits `LoginStatus=user` (`pkg/mdm/mdmtest/windows.go`), and it resends the same body with credentials after the auth challenge, so the alert is present in the trusted message where pending commands are built.

Fleet also never persists the message that carries the alert. `ShouldBeTracked` returns false when `CmdRef` is `"0"`, so a message containing only alerts plus a SyncHdr status has `HasCommands() == false` and `MDMWindowsSaveResponse` is skipped. Any durable record of user context has to be written deliberately.

Two enrollment-level signals separate the paths, and the Autopilot run shows they agree. `enroll_type` is `Device` for Entra join during OOBE and `Full` for programmatic fleetd enrollment. `enroll_user_id` is a real UPN for the former and the orbit node key for the latter. This design keys on UPN-ness through the existing `microsoft_mdm.IsValidUPN` helper, already used at two call sites, because it answers the question that matters directly: whether an MDM user identity can ever be bound to this enrollment. `enroll_type` corroborates but describes the enrollment flavor rather than the user context.

An earlier draft of this design claimed `EnrollmentType` is always `Full` and therefore useless. That was an artifact of the dev database, whose 2412 rows are all programmatic enrollments. The Autopilot measurement disproves it. The conclusion (do not gate on `enroll_type`) stands, but for a different reason than originally written.

### Verified on hardware, Autopilot

Measured 2026-08-14 on a user-driven Autopilot enrollment with Entra join during OOBE, Windows 11 Pro 25H2 build 10.0.26200.8038, capturing all 112 raw SyncML messages across 9 sessions:

- **The LoginStatus alert is sent, in every session.** It appears in MsgID 1 and MsgID 2, and not in MsgID 3. Fleet answers MsgID 1 with a challenge, and the device repeats every alert in the authenticated MsgID 2, so the value is readable inside Fleet's normal authenticated path with no change to the challenge short-circuit.
- **The OOBE value is `others`, not `none`.** During ESP the signed-in account is `defaultuser0`, which is a signed-in user with no MDM account. The value flips to `user` when the Entra user's profile becomes active. A gate keyed on `none`, or on the alert being absent, would never engage.
- The transition was 63 seconds after the enrollment session, on a session Windows opened unprompted carrying alert `1201`.
- The `AADUserToken` alert and the `Authorization: Bearer` header were absent for all 31 messages of the OOBE session and present from the next session onward, both carrying the same 2063 byte token. Either signal would work; LoginStatus is preferred (see D2).
- `enroll_type` was `Device` and `enroll_user_id` was a real UPN. `not_in_oobe` stayed `0` for hours after the device reached the desktop, so it is not usable as a user-context signal.
- A user-scoped `Replace` on a Pro-supported node returned **200** once user context was present, alongside device-scoped controls that also returned 200.

Two results from that run need care:

- A user-scoped write was **never attempted during the `others` window** on this run, so the status code a `./User/` write receives on an Entra enrollment before sign-in is still unmeasured. The customer report and the non-Entra run below are the evidence that it fails.
- A user-scoped `Replace` on `Experience/AllowWindowsSpotlight` returned **405 with user context present and an Entra token on the wire**. That node is documented unsupported on Windows Pro, so it is read-only regardless of who is signed in. A 405 is therefore not proof of missing user context. See D5.

### Verified on hardware, non-Entra

Measured on the `bl-fix-test` Azure VM on 2026-08-14, against the local dev server over the `meeple.top` tunnel:

- A user-scoped SCEP profile produced exactly the reported signature: root node `Add` returns **500**, all 8 sibling `Add`s and the `Exec` return **216** (atomic rollback OK), and the enclosing `Atomic` returns **507**.
- It failed with an interactive user signed in. `quser` showed the local account `fleetadmin` active on the console. So the failure is not "no user is logged in"; it is "no MDM user identity is bound to this enrollment".
- The device is genuinely non-Entra: `dsregcmd /status` reports `AzureAdJoined: NO`, `DomainJoined: NO`, `WorkplaceJoined: NO`, and the enrollment registry key has an empty `UPN` with `EnrollmentType: 0`.
- The device-side CSP `License check` and `CSP Allow check` for `./User/Vendor/MSFT/ClientCertificateInstall/SCEP` both succeed, so the rejection happens at node creation in the user tree, not at CSP permission.
- The retry was consumed **26 seconds** after the first attempt, with an identical status signature. Both attempts are stored in `windows_mdm_responses` (ids 565702 and 565703).
- The `DeviceManagement-Enterprise-Diagnostics-Provider/Debug` channel logs function traces and CSP operations, not SyncML bodies, so it cannot confirm whether this device emits the `1224` LoginStatus alert.

The consequence that shapes this design: on an enrollment with no bound MDM user identity, a user-scoped write fails permanently. Holding such a profile would strand it in Pending forever, and exempting it from the retry counter would resend it forever.

### When the hold actually engages

The Autopilot run did not reproduce the bug, and the reason is structural rather than luck. Fleet cannot queue profiles for a Windows host until the enrollment is linked to a host record, and on an Entra-join enrollment that linkage waits for fleetd. On the measured run the enrollment was created unlinked at 23:35:33, fleetd enrolled and linked the host at 23:36:38, and LoginStatus flipped to `user` in that same second. The window in which profiles were queue-eligible but user context was missing had zero width.

So the failure in #50196 requires linkage to **precede** user context. Three ways that happens:

1. The host record already exists in Fleet, so the reverse link resolves immediately instead of waiting for a fresh fleetd enrollment. Re-enrollment and any device Fleet has seen before fall here.
2. A slower ESP, or any delay between linkage and sign-in.
3. A self-deploying or pre-provisioning Autopilot flow, where no user signs in at all. This is also the only flow that exercises the `none` value.

This narrows where the bug bites without weakening the fix. The hold is cheap, it engages only in the situations that produce the failure, and case 3 is a state the current code has no answer for at all: profiles queued against a device that will never have a user.

## Goals / Non-Goals

**Goals:**

- A user-scoped profile assigned before first sign-in on an Autopilot or Entra-join-during-OOBE host delivers automatically on the first session that reports a user context, instead of failing terminally.
- No retry budget is spent on a condition that has not had a chance to change.
- Where a user-scoped profile can never be delivered, the host reports an actionable reason instead of a raw SyncML status code, and Fleet stops resending.
- Device-scoped profile behavior is byte-for-byte unchanged.

**Non-Goals:**

- Userless Windows enrollment (story #48931). That is the reason a programmatic enrollment has no bound user identity, and fixing it is a separate client-side change.
- Splitting a mixed-scope profile so its device half can deliver early.
- Consuming the Entra user token. It arrives in an HTTP header, and `SyncMLReqMsgContainer.DecodeBody` receives only the body, query parameters, and client certificates.
- Changing the ESP user-scope release path (`handleESPUserReleaseRetry`), which solves the same problem for one specific node and keeps its current behavior.

## Decisions

### D1: Model three user-context states, keyed on whether a user context can ever arrive

| State | How Fleet decides | Behavior for user-scoped profiles |
| --- | --- | --- |
| Present | the current or a prior session reported `LoginStatus=user` | deliver normally |
| Can arrive | `enroll_user_id` is a valid UPN and `user` has not been observed | hold, do not spend retries |
| Cannot arrive | `enroll_user_id` is not a valid UPN, so no MDM user identity is bound | fail fast with an explanatory detail, do not resend |

**Release only on `user`.** The "can arrive" state covers three distinct observations that all mean the same thing operationally: `others` (someone is signed in without an MDM account, which is what OOBE reports), `none` (nobody is signed in), and no observation recorded yet. The Autopilot run makes this concrete: the value during ESP is `others`, so a gate that engaged only on `none` would never fire on the exact flow the issue is about, and a gate that engaged only when the alert is missing would never fire at all.

Rationale: the non-Entra run proves "cannot arrive" is a real and permanent state, so a single hold-everything rule is wrong. Separating the two waiting cases is what keeps Pending honest and keeps the retry exemption from becoming an infinite resend loop.

Alternatives considered:

- Hold on every enrollment until `LoginStatus=user`. Rejected: on a non-Entra host the alert may never report `user`, leaving the profile Pending forever with no explanation. Silent and worse than today's failure.
- Keep sending and retrying everywhere (status quo). Rejected: wastes the retry budget in 26 seconds and reports opaque 500/507 to admins.
- Gate on the MS-MDE2 `EnrollmentType` context item. Rejected: always `Full`, so it cannot separate Entra-join-during-OOBE from fleetd enrollment.

### D2: Read LoginStatus from the SyncML body, not the Authorization header

Handling alert `1224` in `processIncomingAlertsCommands` needs no plumbing, and the existing test client already produces it, so both states are drivable in integration tests today. The header route would require threading `http.Header` into the SyncML decoder for every management request.

The Autopilot run confirms both signals track user context identically, so this is a choice between two working options rather than a bet. LoginStatus wins on three counts: it is a three-state value that distinguishes "no MDM user" from "wrong user" where token presence is only a boolean, it needs no header plumbing, and it keeps a bearer token off a code path that logs.

The alert is read from the authenticated message. The device repeats its alerts in MsgID 2 after Fleet's challenge, so the existing early return on the challenge path stays as it is.

### D3: Persist the observed user-context state on `mdm_windows_enrollments`

The alert appears only in DM package #1, later messages in the same session do not repeat it, Fleet does not persist the message that carries it, and the reconciler cron runs outside any session. A column is the only place all three readers can agree on. Store the last observed value and its timestamp so the value can be interpreted (and so a stale observation is visible in debugging).

### D4: Hold at enqueue time in the reconciler, not at send time

The reconciler consults the persisted state and does not enqueue a held profile, leaving the profile row Pending with a detail explaining what is being waited on and no command UUID.

Rationale: the command queue stays free of commands that are known to be undeliverable, which avoids interacting with the `has_pending_commands` flag accounting and the poll-schedule relaxation logic in `getManagementResponse`. Convergence is fast regardless: Fleet already provisions `AllUsersPollOnFirstLogin = true`, so Windows starts a session at first sign-in, that session persists the new state, and the 30 second profile cron enqueues on its next tick.

The Autopilot run confirms the convergence assumption. Windows opened a session unprompted 63 seconds after the enrollment session, carrying alert `1201` and the flipped LoginStatus, with nothing pushed from the server. Worst-case added latency after sign-in is therefore that session plus one cron tick.

Alternative considered: filter user-scoped commands out of `getPendingMDMCmds` so the same session that reports `LoginStatus=user` ships them immediately. Rejected for now because it leaves undeliverable rows queued and complicates the per-session `has_pending_commands` refresh, for a saving of at most one cron interval.

### D5: Let the state carry the retry exemption, not the status code

A rejection of a user-scoped command skips the `retries` increment in `MDMWindowsSaveResponse` **only while the enrollment is in the "can arrive" state**. In the "present" and "cannot arrive" states the normal accounting applies, and "cannot arrive" is failed fast before a command is ever sent.

The state is what carries this decision, not the status code, because no status code reliably means "user context is missing". The evidence now spans three codes with three different causes:

| Code | Observed | Cause |
| --- | --- | --- |
| 500 | non-Entra device, SCEP root node `Add` | no MDM user identity bound |
| 405 | ESP user-scope release during OOBE | user MDM context not yet initialized |
| 405 | Autopilot device, `AllowWindowsSpotlight`, **user context present** | node unsupported on Windows Pro |

A rule keyed on "405 or 500 on a user-scoped LocURI" would misread that third row as a user-context problem and exempt a genuinely unsupported node from retry accounting. Gating on state instead bounds the damage: an unsupported node in the "can arrive" window gets its retries deferred rather than forgiven, and normal accounting resumes the moment `user` is observed.

This must not disturb the existing nested-418 resend path. The non-Entra VM shows a SCEP `Atomic` legitimately returning 507 because a nested `Add` returned 418 (already exists), which `handleResendingAlreadyExistsCommands` converts into a `Replace` and which then reaches Verified. A 507 on the `Atomic` is therefore not by itself evidence of a user-context problem either; the nested per-LocURI statuses are what distinguish the two.

### D6: Classify scope with the same normalization the delivery path uses

Derive scope from `ExtractLocURIsFromProfileBytes` over the `WrapSCEPProfileInAtomic`-normalized bytes, and treat a profile as user-scoped if any canonicalized target resolves under `./User/`. Persist the result on the profile row and backfill existing rows in the migration, so the reconciler does not re-parse XML and the API can expose scope later.

Rationale: a raw `strings.HasPrefix("./User/")` is bypassable. Scope-less spellings exist, and CDATA, comments, and nested elements can split LocURI text, which is the parser differential fixed in PR #49715 and the reserved-node canonicalization work in #48752. Classification and delivery must not be able to disagree.

## Risks / Trade-offs

- **Classification and delivery disagree, so a profile classified device-scoped ships a `./User/` write** → classify using the same normalization and parser as delivery, and add a property test that asserts every profile whose delivered SyncML contains a canonicalized `./User/` target is classified user-scoped.
- **A held profile sits in Pending forever because the device never reports a user context** → the Autopilot run retires this for user-driven flows: the alert arrives in every session and flips to `user` 63 seconds after enrollment. It remains live for self-deploying and pre-provisioning Autopilot, where no user ever signs in and the value should read `none`. Those hosts are exactly the ones whose user-scoped profiles cannot be delivered, so Pending with an explanatory detail is the honest state, but it needs a decision on whether a hold that never releases should eventually surface as a failure.
- **An unsupported CSP node is mistaken for missing user context** → the state gate, not the status code, controls the exemption (D5). A node that is unsupported on the device SKU returns 405 with user context present, which the "present" state routes to normal accounting.
- **The retry exemption becomes an infinite resend** → the exemption is scoped to the waiting state (D5); the terminal path for "cannot arrive" fails fast and does not resend.
- **UPN-ness of `enroll_user_id` is an imperfect proxy for a bound MDM user identity** → now measured on both sides: the non-Entra enrollment stored an orbit node key with an empty registry `UPN` and failed permanently, and the Autopilot enrollment stored a real UPN and succeeded once user context arrived. `enroll_type` (`Full` versus `Device`) agrees with the proxy on both paths and is available as a cross-check.
- **Mixed-scope profiles hold their device settings too** → accepted and documented. It matches the all-or-nothing semantics `WrapSCEPProfileInAtomic` already imposes, and splitting would break the atomicity SCEP profiles depend on.
- **Pending is overloaded** → a held profile is Pending for a reason an admin cannot guess, so the detail message carries the explanation. Whether this deserves a distinct UI state is an open question.
- **ESP interaction** → none intended. The ESP hold blocks on software install failures only, and its own user-scope release retry is untouched.

## Migration Plan

1. Migration adds the user-context columns to `mdm_windows_enrollments` and the scope column to `mdm_windows_configuration_profiles`, then backfills scope for existing profiles.
2. Deploy is additive. A NULL user-context observation reads as "never observed", which reproduces current behavior, so a partially rolled out fleet behaves as it does today until each enrollment reports in.
3. Rollback is a code revert; the columns can stay. No profile state is destroyed, and held profiles are Pending rather than Failed, so a revert resends them normally.

## Open Questions

Answered by the Autopilot run on 2026-08-14: the alert is emitted in every session, in MsgID 1 and MsgID 2, with `others` during OOBE and `user` once the Entra user's profile is active.

Still open:

- **What status does a `./User/` write get on an Entra enrollment during the `others` window?** The measured run never attempted one, because linkage and user context arrived in the same second. The customer report and the non-Entra 500 are the evidence that it fails, but the code on that exact path is unmeasured. Reproducing it needs one of the three linkage-before-user-context cases above.
- **Does a self-deploying or pre-provisioning Autopilot flow report `none`, and does it ever flip?** This is the only flow that exercises `none`, and it is the one where a hold could legitimately never release.
- **Should a hold that will never release eventually become a failure?** For a device that never gets a user, Pending is honest but silent. A deadline that converts the hold into an explanatory failure would trade one imperfect state for another, and the choice depends on the answer to the previous question.
- Should a held profile surface a distinct state in the UI and API rather than Pending plus a detail string?
