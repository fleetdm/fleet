## Why

Fleet queues Windows user-scoped profiles (`./User/Vendor/MSFT/...`) as soon as they apply to a host, without checking whether the device has an MDM user context yet. During Autopilot and Entra-join-during-OOBE there is no MDM user context, so the write fails, the enclosing `Atomic` rolls back, and the profile lands terminally Failed. It is never re-sent after the user signs in, even though the same write would then succeed. This blocks user-channel SCEP certificates for VPN client auth and IdP device attestation (issue #50196, customer-thumper).

The single automatic retry does not help. It is consumed against the identical condition seconds later, so both attempts fail the same way and the retry budget is spent before a user could possibly appear. Measured on hardware, the two attempts were 26 seconds apart with identical status codes.

The window is narrower than it first appears, which scopes the fix rather than weakening it. Profiles only become eligible once the enrollment is linked to a host record, and on a first-time Autopilot enrollment that linkage waits for fleetd, which in a measured run landed in the same second as first sign-in. The failure needs linkage to precede user context: a device Fleet has already seen, a slower ESP, or a self-deploying flow where nobody signs in at all.

## What Changes

- Fleet reads the OMA-DM generic alert `1224` with type `com.microsoft/MDM/LoginStatus` from each management session and persists the reported user-context state on the enrollment. Fleet already receives this alert and silently discards it.
- Windows profiles gain a notion of channel scope, derived from their canonicalized `Add`/`Replace` LocURIs. A profile is user-scoped if any of its targets resolve to `./User/`.
- User-scoped profiles are **held** (left Pending, not enqueued) on enrollments where an MDM user context can still arrive, and are delivered automatically on the first session that reports one.
- User-scoped profiles **fail fast with an explanatory message** on enrollments that have no bound MDM user identity, where the write can never succeed. Today these retry once and then report raw SyncML status codes with no explanation.
- A user-context rejection no longer consumes the profile's retry budget while Fleet is holding and waiting for a user context.
- Mixed-scope profiles (both `./Device/` and `./User/` targets) are held or failed as a unit, matching the all-or-nothing semantics that `WrapSCEPProfileInAtomic` already imposes.
- Documentation states which enrollment types support user-channel profiles and what Pending means for them.

No API or schema-breaking changes. The observable behavior change is that a user-scoped profile which previously went to Failed within a minute of OOBE enrollment now stays Pending until first sign-in, then reports Verified.

## Capabilities

### New Capabilities

- `windows-user-scoped-profiles`: how Fleet classifies Windows profiles by channel scope, how it learns the device's MDM user context from the OMA-DM session, and the delivery, hold, failure, and retry-accounting rules that follow from it.

### Modified Capabilities

None. `openspec/specs/` has no accepted specs yet, so there are no existing requirements to amend.

## Impact

Affected code:

- `server/service/microsoft_mdm.go`: `processIncomingAlertsCommands` (add the `1224` case), `getPendingMDMCmds` (hold user-scoped commands), `ReconcileWindowsProfiles` and `executeWindowsProfileReconcileBatch` (scope-aware enqueue).
- `server/fleet/windows_mdm.go` and `server/fleet/microsoft_mdm.go`: scope classification built on the existing LocURI canonicalization helpers.
- `server/datastore/mysql/microsoft_mdm.go`: retry accounting in `MDMWindowsSaveResponse` (the `MaxWindowsProfileRetries` branch), plus persistence of the observed user-context state.
- New migration: user-context columns on `mdm_windows_enrollments`.
- `server/mdm/microsoft/syncml/syncml.go`: alert type constant for `com.microsoft/MDM/LoginStatus`.

Affected behavior and surfaces:

- Host details and profile summaries show Pending (not Failed) for held profiles, with a detail message explaining what is being waited on.
- Windows Autopilot and Entra-join-during-OOBE flows. The Enrollment Status Page is unaffected: it already holds only on software install failures, and its own user-scope release path (`handleESPUserReleaseRetry`) keeps its existing behavior.

Docs to update: `articles/custom-os-settings.md`, `articles/creating-windows-csps.md`, `docs/Contributing/architecture/mdm/windows-mdm-architecture.md`, and the header comment in `docs/solutions/windows/configuration-profiles/install Okta attestation certificate - [Bundle].xml`.

Not in scope: userless Windows enrollment (story #48931), which would let a programmatic enrollment bind a user identity and is the separate reason user-channel profiles cannot work on that path today.
