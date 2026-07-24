# Spec: Fleet assignment for Windows automatic enrollment

## ADDED Requirements

### Requirement: Automatically enrolled Windows hosts are assigned to the default fleet
When a Windows MDM enrollment is linked to a host, Fleet SHALL move the host to the configured default fleet iff all of the following hold: the enrollment is automatic (Entra UPN user id, not programmatic), a default fleet is configured at link time, the host has no fleet assigned, and the host is new to Fleet in this enrollment cycle (its host record was created by this enrollment, including re-creation after the host was deleted from Fleet). Pre-existing hosts SHALL never be moved, regardless of their current fleet. Programmatic enrollments SHALL never trigger assignment. The transfer SHALL run the same side effects as a manual fleet transfer (profile reconciliation, disk encryption key handling); no `transferred_hosts` activity is logged.

#### Scenario: New Autopilot host lands in default fleet
- **WHEN** the default fleet is "Team A" and a new Windows host enrolls via Autopilot (automatic, OOBE), then fleetd enrolls it into Fleet
- **THEN** the host is assigned to "Team A"

#### Scenario: No default configured
- **WHEN** no default fleet is configured and a new Windows host enrolls via Autopilot
- **THEN** the host stays in "No team"

#### Scenario: Changing the default affects only future enrollments
- **WHEN** host one enrolled while the default was "Team A", the admin changes the default to "Team B", and host two then enrolls automatically
- **THEN** host two is assigned to "Team B" and host one remains in "Team A"

#### Scenario: Existing host keeps its fleet on re-enrollment
- **WHEN** a host already assigned to "Team C" (manually or via enroll secret) wipes and re-enrolls via Autopilot while the default fleet is "Team A"
- **THEN** the host remains in "Team C"

#### Scenario: Deleted host re-enrolls
- **WHEN** a host is deleted from Fleet and later re-enrolls via Autopilot with a default fleet configured
- **THEN** the re-created host is assigned to the default fleet (parity with ABM restore behavior)

#### Scenario: Programmatic enrollment unaffected
- **WHEN** a Windows host with fleetd (enrolled via a fleet-specific or global enroll secret) is enrolled into MDM programmatically
- **THEN** its fleet is not changed by the default fleet setting

#### Scenario: Host parked in No team stays in No team
- **WHEN** a host that an admin deliberately moved to "No team" re-enrolls automatically while a default fleet is set
- **THEN** the host remains in "No team" (macOS ABM parity: re-enrollment never moves an existing host)

### Requirement: Assignment precedes the Autopilot ESP setup experience
For the OOBE automatic enrollment flow, the default fleet SHALL be applied before Fleet responds to orbit's enrollment, so that orbit's one-shot `setup_experience/init` call enqueues the default fleet's setup experience items and the ESP delivers the default fleet's profiles. To make this ordering deterministic, Fleet SHALL persist the device serial reported via the DevDetail SMBIOS query on the unlinked enrollment row, and the orbit enrollment path SHALL reverse-link a matching unlinked automatic enrollment (by serial) and apply the default fleet before returning.

#### Scenario: ESP uses the default fleet's setup experience
- **WHEN** the default fleet has setup experience software configured and a new host goes through Autopilot OOBE
- **THEN** the ESP installs the default fleet's setup experience items and delivers the default fleet's profiles, not "No team"'s

#### Scenario: No ESP deadlock from late assignment
- **WHEN** "No team" has no setup experience items, the default fleet has items, and a host enrolls via Autopilot
- **THEN** the ESP completes normally within the default fleet's setup experience (it does not wait until the 3 hour timeout)

#### Scenario: Serial unavailable falls back to late linking
- **WHEN** a device reports a placeholder serial (or never answers the DevDetail query) so the reverse link at orbit enrollment finds no match
- **THEN** the host is still assigned to the default fleet when the DevDetail or osquery link path completes, and the ESP falls back to the fleet the host had when orbit initialized setup experience
