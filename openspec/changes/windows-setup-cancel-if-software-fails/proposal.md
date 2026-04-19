# Proposal: Windows setup experience, cancel if software fails

## Intent
Windows hosts enrolling via Autopilot today land on the desktop as soon as enrollment completes, with no
visible progress for profiles, setup software, or certificates, and no way for an admin to hold the user in
OOBE until "work blocking" software is present. Introduce a Windows Enrollment Status Page (ESP) that tracks
profile, SCEP, and setup-software progress during OOBE, and add a per-team `require_all_software_windows`
switch that blocks the user with a "Try again" message when any critical item fails. This brings Windows to
parity with the macOS `require_all_software` contract and gives IT admins the guarantees their organizations
require.

## Scope
- New per-team config field `require_all_software_windows` (default false) via REST API, YAML, and UI
- Rename `require_all_software` to `require_all_software_macos` with a backward compatible alias
- Detect Autopilot OOBE enrollments and mark hosts awaiting_configuration
- Drive the Windows Enrollment Status Page via DMClient and EnrollmentStatusTracking CSPs
- Finalize the ESP: release on success, block with "Try again" when require_all_software_windows=true and
  any item failed, or time out after 3 hours
- Emit canceled_setup_experience activity on Windows cancellation

## Approach
Three phases, each independently deployable. Phase 1 adds the config field and detects Autopilot OOBE at
enrollment. Phase 2 initializes the ESP on the first qualifying Microsoft Management checkin and returns
ongoing status inline on subsequent checkins. Phase 3 finalizes the flow when items reach terminal state or
the 3 hour timeout fires. The existing setup_experience_status_results table and the
CancelPendingSetupExperienceSteps helper are reused. Windows plugs into getManagementResponse as the single
SyncML entry point.
