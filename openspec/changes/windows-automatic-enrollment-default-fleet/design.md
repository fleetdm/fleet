# Design: Default fleet for Windows automatic enrollment

## Context

Windows hosts reach Fleet MDM through two enrollment types (`server/fleet/microsoft_mdm.go`):

- **Programmatic** (`WindowsMDMEnrollTypeProgrammatic`): fleetd triggers enrollment using an orbit node key. The host already exists in Fleet and already has a fleet from its enroll secret.
- **Automatic** (`WindowsMDMEnrollTypeAutomatic`): Entra JWT / WSTEP STS token. Covers Autopilot OOBE, bare Entra-join during OOBE, and Settings app "Access work or school" joins. Persisted marker: `mdm_windows_enrollments.user_id` is a UPN (`microsoft_mdm.IsValidUPN`), and `not_in_oobe` distinguishes OOBE from post-OOBE.

The automatic flow today, for a host new to Fleet:

1. Device enrolls via WSTEP during OOBE. The request carries no identifier that maps to `hosts.uuid`, so the enrollment row is inserted **unlinked** (`host_uuid = ""`). If in OOBE, `awaiting_configuration = Pending` (ESP hold).
2. On every OMA-DM session while unlinked, Fleet injects a Get for `./DevDetail/Ext/Microsoft/SMBIOSSerialNumber` (`server/service/microsoft_mdm.go:1894`). The device answers with its serial on the first session, but the host lookup returns NotFound (no host row yet) and **the serial is discarded**.
3. Fleet pushes the fleetd MSI over MDM using the **global** enroll secret (`enqueueInstallFleetdCommand`). fleetd installs, orbit enrolls, and the host row is created in "No team".
4. Orbit fetches config (`RunSetupExperience = true` while Pending/Active) and calls `POST /orbit/setup_experience/init` **exactly once per install** (guarded by a status file in `orbit/cmd/orbit/orbit.go`). `SetupExperienceInit` (`ee/server/service/orbit.go`) enqueues ESP items for `host.TeamID` **at that moment**.
5. On the next OMA-DM session, `tryLinkUnlinkedEnrollmentFromDevDetail` links the enrollment to the host by serial, and the ESP transitions Pending → Active. `directIngestMDMDeviceIDWindows` (osquery) remains the linking backstop.

Steps 4 and 5 race. Any design that assigns the fleet at link time (or later, like community PR #41757 which hooks the osquery backstop) loses the race in practice: fleetd install takes minutes while orbit calls init seconds after enrolling. Consequences of assigning after init:

- ESP installs the wrong fleet's setup experience software.
- Deadlock-to-timeout: if "No team" has no setup experience items, init enqueues nothing; after the transfer, `handleESPRelease` sees empty results but `HasWindowsSetupExperienceItemsForTeam(newTeam) = true` and waits for an enqueue that never comes (orbit never re-calls init), until the 3 hour ESP timeout fails the device.

Apple parity reference: ABM default teams are applied when the host is first created (ABM sync ingest or restore after deletion); re-enrollment never moves an existing host. Deleting a team that is an ABM default clears the reference (`ee/server/service/teams.go`, `CleanRemovedTeam`); deletion is not blocked.

## Goals / Non-goals

**Goals:**

- A global admin can set one default fleet for Windows automatic enrollment (UI, config API, GitOps).
- Automatically enrolled Windows hosts that are new to Fleet land in that fleet, and the fleet is set **before** the ESP setup experience is enqueued, so the correct software, scripts, and profiles apply during OOBE.
- Behavior mirrors ABM defaults where it makes sense: existing hosts keep their fleet on re-enrollment; changing the default only affects future enrollments; deleting the default fleet clears the setting.
- Premium only, global admin (and GitOps role via gitops) only.

**Non-goals:**

- Per-Entra-tenant or per-platform defaults (single global setting; can evolve later like ABM went from `apple_bm_default_team` to per-token teams).
- Distinguishing "true Autopilot" (deployment profile present) from other Entra automatic enrollments. Fleet has no reliable signal for this; the product naming ("automatic enrollment") intentionally covers all of them.
- Retroactively moving already-enrolled hosts when the setting changes.
- Windows server exclusion logic changes (existing `is_server` handling is untouched).

## Decisions

### 1. Storage: `team_id` in a dedicated single-row table, fleet name surfaced in the config API

New table `windows_automatic_enrollment_config` (single row, `team_id` nullable FK to `teams(id)` with `ON DELETE SET NULL`). The config API and GitOps surface the fleet **name** under `mdm.windows_automatic_enrollment.default_fleet` (empty string = unassigned), hydrated at read time by joining `teams`.

- Why not the fleet name in `app_config_json` (legacy `apple_bm_default_team` pattern): renames silently orphan the reference; that pattern was deprecated for ABM for exactly this reason.
- Why not id-as-string in the API (what the REST API doc PR #49591 currently shows): GitOps is name-based and the YAML doc (#49589) uses the name; ids in YAML would be unusable across environments. Follow-up doc fix required either way, so converge on name. The UI maps name to id using the fleets list it already loads.
- Why not the community PR's `windows_mdm_default_team` table plus dedicated endpoints: the shipped product docs put the setting in the config API, not new endpoints. Keep the table idea (it gives the FK cleanup for free) but drop the endpoints.
- `AppConfig.MDM` gains `WindowsAutomaticEnrollment optjson.Any`-style struct (`optjson.Slice`/`optjson.Object` conventions as for `AppleBusinessManager`): present for API/gitops round-tripping, but the DB row is the source of truth. `ModifyAppConfig` validates the name against existing fleets, resolves to id, writes the table, and emits the activity when the value changed.

### 2. Assignment point: link-time hook plus a reverse link at orbit enrollment so assignment precedes ESP init

Two pieces:

1. **Persist the serial on the unlinked enrollment.** Add `hardware_serial` (nullable) to `mdm_windows_enrollments`. When `tryLinkUnlinkedEnrollmentFromDevDetail` gets a serial but the host lookup is NotFound, store the serial on the enrollment row instead of discarding it. Placeholder serials (`IsPlaceholderHardwareSerial`) are not stored, matching existing skip logic.
2. **Reverse link at orbit enroll.** In the Windows orbit enrollment path, after the host row is created or matched, look up an unlinked automatic enrollment by `hardware_serial` (primary DB). If found: link it (reuse `LinkWindowsHostMDMEnrollment`) and apply the default fleet, all before the enroll response returns. Orbit's subsequent config fetch and `SetupExperienceInit` then see the correct fleet, eliminating the race by construction (the serial reaches Fleet on the first OMA-DM session, minutes before fleetd finishes installing).

The assignment itself lives in one helper called from `LinkWindowsHostMDMEnrollment` so all three link paths (reverse link, DevDetail, osquery backstop) get it. The late paths are degraded-but-correct fallbacks for edge cases (placeholder serials, DevDetail never answered); for those, accept that ESP may have used No team's items, which is still no worse than today.

**Assignment rule:** apply the default fleet iff all of: the enrollment is automatic (`IsValidUPN(user_id)`), a default fleet is configured at link time, the host has no fleet assigned, and the host is new to Fleet in this enrollment cycle. "New in this cycle" means the host row was created at or after the MDM enrollment row (deleted-then-re-enrolled hosts qualify because their row is re-created; a pre-existing host that an admin parked in "No team" does not, so it stays in "No team", exact macOS ABM parity). In the reverse-link path this is known directly (orbit enrollment just created the row); in the late link paths, compare `hosts.created_at` against the enrollment row's `created_at`. Transfer via the same side-effect path as a manual transfer (`BulkSetPendingMDMHostProfiles` etc.), not a bare datastore `AddHostsToTeam` (the community PR's bare call skips profile reconciliation). No `transferred_hosts` activity is logged, mirroring ABM ingest.

- Why not plain `TeamID == nil`: it would pull a host deliberately parked in "No team" into the default fleet on automatic re-enrollment; product decision (Victor, 2026-07-24) is that such hosts stay in "No team", matching macOS.
- Why not gate `SetupExperienceInit` until the link completes: orbit treats an init error as terminal for the boot (no retry), so holding init would break setup experience entirely.

### 3. ESP-time consistency check

`handleESPRelease`'s empty-results disambiguation already reads the host's current fleet, so once assignment precedes init there is nothing to change there. The design adds no new ESP states.

### 4. GitOps

`org_settings.mdm.windows_automatic_enrollment.default_fleet` (object, not the list shown in the YAML doc example; doc to be fixed). The referenced fleet may be created in the same GitOps run, so reuse the existing ABM/VPP mechanism in `cmd/fleetctl/fleetctl/gitops.go`: validate the name against declared teams, apply after teams exist, honor dry-run assumptions. Omitting the key is a no-op; explicit empty string clears the setting. `generate-gitops` exports the current value.

### 5. UI (Figma "✅ Ready" page, section "Settings > Integrations > MDM > Windows")

All on `WindowsMdmPage` (`/settings/integrations/mdm/windows`):

- New "Automatic enrollment" section: "Default fleet" dropdown listing all fleets plus "Unassigned" (default). Helper text: "Hosts that automatically enroll (Windows Autopilot) are added to this fleet." with a Learn more link. Disabled with tooltip "Fleet must be connected to Entra to set a default fleet." when no Entra connection is configured (`config.mdm.windows_entra_tenant_ids` empty); the tooltip's Learn more goes to `/settings/integrations/automatic-enrollment/windows`. Read-only in GitOps mode. Premium only (section hidden on Free, and the backend rejects the setting without a Premium license).
- Replace the "End user experience" radios with a "Turn on MDM programmatically" toggle bound to the existing `mdm.enable_turn_on_windows_mdm_manually` (toggle on = `false`). Tooltip: "When enabled, MDM is turned on when Fleet's agent is installed. When disabled, end users turn on MDM manually in Settings > Access work or school (requires Microsoft Entra). Only applies to manual enrollment." Learn more: `https://fleetdm.com/learn-more-about/mdm-enrollment` (route merged in #49603). UI-only change; no backend field changes.
- Existing "Automatically migrate hosts connected to another MDM solution" checkbox moves under a new "Migration" heading.
- Saving uses the existing `PATCH /config` call from this page, now including `windows_automatic_enrollment` when the dropdown changed.

### 6. Activity

`edited_windows_automatic_enrollment_default_fleet`, fields `fleet_id` and `fleet_name` (both null when cleared), emitted from `ModifyAppConfig` (covers UI, API, and GitOps) only when the value actually changed. Dashboard label: "Edited automatic enrollment default fleet: Windows". Feed copy per Figma: "<user> edited the default fleet for Windows automatic enrollment hosts to <fleet>." Already documented in audit-logs on `docs-v4.91.0` (#49594).

### 7. Fleet deletion

`ON DELETE SET NULL` on the FK plus explicit cleanup in the EE delete-team service path is enough; no activity on implicit clearing. Do not block deletion (community PR blocks; ABM does not).

## Risks / Trade-offs

- [Reverse link matches the wrong host on duplicate serials] → Same exposure as existing serial linking; placeholder serials are excluded exactly like `tryLinkUnlinkedEnrollmentFromDevDetail` does, and the osquery backstop self-corrects links.
- [Serial never arrives before orbit enrolls (device slow to answer DevDetail, or placeholder serial)] → Late assignment via the DevDetail/osquery link paths still lands the host in the right fleet; only the ESP content may reflect No team. Document in the guide that No team setup experience should be kept empty when using a default fleet, or accept the fallback.
- [Settings app (BYOD) automatic enrollments are included] → Intentional per product naming, but means a BYOD Entra join with MDM-pushed fleetd gets the default fleet. If QA/product wants Autopilot-OOBE-only, the rule gains `not_in_oobe = false`; one-line change, decide during review.
- [Clock-based "new in this cycle" check in late link paths] → `hosts.created_at` vs enrollment `created_at` comparison needs a small grace window for sub-second ordering; the reverse-link path (the common case) does not rely on timestamps at all.
- [GitOps ordering: fleet created in the same run] → Reuse the proven ABM/VPP deferred-apply machinery rather than inventing a new pass.
- [Migration adds a column to `mdm_windows_enrollments`, a hot table] → Nullable column addition, no backfill, no index needed beyond an index on (`hardware_serial`) scoped to unlinked lookups; verify with the standard migration test.

## Migration plan

1. DB migration: new `windows_automatic_enrollment_config` table; add `hardware_serial` to `mdm_windows_enrollments`.
2. Backend, CLI, frontend land together behind the Premium license check; no feature flag needed (setting defaults to unassigned, which is today's behavior).
3. Rollback: the setting is additive; downgrading Fleet leaves the table unused and hosts enroll into No team as before.

## Open questions

- Autopilot-only (`not_in_oobe = false`) vs all automatic enrollments: this design says all automatic, matching product naming. Confirm with product (melpike) during review.
- Exact `optjson` shape in `AppConfig` (object with `default_fleet` string) vs a flatter field: follow whatever `AppleBusinessManager` conventions make round-tripping cleanest; decide at implementation.

## Resolved decisions

- Host parked in "No team" that re-enrolls automatically stays in "No team" (Victor, 2026-07-24). Encoded in the assignment rule above.
- Deleting the fleet referenced as the default clears the reference; deletion is not blocked (Victor, 2026-07-24).
- API doc uses the fleet name (not id string); YAML doc uses object form. Fixed on a branch off `docs-v4.91.0` ahead of implementation.
