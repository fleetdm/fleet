# Proposal: Default fleet for Windows automatic enrollment

GitHub issue: https://github.com/fleetdm/fleet/issues/41787 (story "Add default fleets for Windows Autopilot-enrolled hosts")
Figma: https://www.figma.com/design/13pmqUXrUen01ufpJu6rAq (wireframes on the "✅ Ready" page)
Reference community PR: https://github.com/fleetdm/fleet/pull/41757 (useful as a starting point, but its team assignment hook is too late; see design.md)

## Why

IT admins using Windows Autopilot (or any Entra-based automatic enrollment) have no way to land Windows hosts in a specific fleet: every automatically enrolled host arrives in "No team" because Fleet pushes fleetd with the global enroll secret, and admins must transfer each host manually. Apple hosts have had this for years via ABM default teams. This story brings Windows to parity.

## What changes

- New global, Premium-only setting: the default fleet for Windows automatic enrollment, surfaced as `mdm.windows_automatic_enrollment.default_fleet` in the config API and in GitOps `org_settings.mdm`.
- Windows hosts that enroll in Fleet MDM via automatic enrollment (Entra JWT: Autopilot OOBE, Entra join during OOBE, Settings app work-or-school join) are assigned to the configured default fleet when they are new to Fleet (no fleet assigned at MDM-link time). Programmatic (fleetd-initiated) enrollments are never affected. Assignment happens before orbit enqueues Autopilot ESP setup experience items, so the right fleet's software, scripts, and profiles apply during OOBE.
- New activity type `edited_windows_automatic_enrollment_default_fleet` with `fleet_id` and `fleet_name` fields (dashboard label "Edited automatic enrollment default fleet: Windows").
- Windows MDM settings page (`/settings/integrations/mdm/windows`) redesign per Figma: new "Automatic enrollment" section with a "Default fleet" dropdown, the "End user experience" radios replaced with a "Turn on MDM programmatically" toggle (same backend field, inverted), and the auto-migration checkbox moved under a new "Migration" heading.
- `fleetctl generate-gitops` exports the setting; the dropdown is read-only in GitOps mode.
- Deleting the fleet that is set as the default clears the setting (mirrors ABM behavior; does not block deletion).

## Capabilities

### New capabilities

- `windows-automatic-enrollment-default-fleet-config`: the setting itself: storage, config API surface, GitOps apply and export, activity logging, permissions and license gating, fleet deletion cleanup.
- `windows-automatic-enrollment-fleet-assignment`: enrollment-time behavior: which enrollments qualify, when the host is assigned, ordering guarantees relative to the Autopilot ESP setup experience, re-enrollment semantics.
- `windows-mdm-settings-page`: the Windows MDM settings page UI changes (Automatic enrollment section, programmatic enrollment toggle, Migration section).

### Modified capabilities

None (no existing openspec specs in this repo yet).

## Impact

- Backend: `server/service/microsoft_mdm.go` (DevDetail linking), `server/service/osquery_utils/queries.go` (link backstop), orbit enrollment path, `server/fleet/` config types, `ee/server/service/` (Premium logic), `server/datastore/mysql/` (new migration and storage), activity types in `server/fleet/activities.go`.
- API: `PATCH /api/v1/fleet/config` and `GET /api/v1/fleet/config` gain `mdm.windows_automatic_enrollment`. No new endpoints (the community PR added dedicated GET/PATCH endpoints; the shipped product docs instead put this in the config API).
- CLI: `fleetctl gitops` (apply) and `fleetctl generate-gitops` (export).
- Frontend: `frontend/pages/admin/IntegrationsPage/cards/MdmSettings/WindowsMdmPage/`.
- Docs already drafted on `docs-v4.91.0` (PRs #49589 YAML, #49591 REST API, #49594 activity, #48658 guide). Known doc inconsistencies to fix during implementation: the API doc describes `default_fleet` as a fleet id string while YAML uses the fleet name; the YAML example is a list while the API is an object; the API doc also added the field to fleet-level (team) mdm objects, which is wrong for a global setting; the guide PR uses stale key names (`windows_autopilot_default_team`, `default_fleets`).
