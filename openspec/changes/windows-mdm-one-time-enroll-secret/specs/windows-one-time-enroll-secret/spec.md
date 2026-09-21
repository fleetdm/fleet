## ADDED Requirements

### Requirement: Windows MDM delivers a per-enrollment enroll secret
When one-time enroll secrets are enabled, Fleet SHALL NOT send the global enroll secret to a Windows MDM host. The fleetd install command SHALL carry a secret minted for that specific MDM enrollment.

#### Scenario: Fresh Windows MDM enrollment receives a unique secret
- **WHEN** a Windows host enrolls in Fleet MDM and Fleet enqueues the fleetd install command
- **THEN** the secret carried by the command is not equal to any secret in `enroll_secrets`
- **AND** the secret is recorded server-side bound to that MDM enrollment

#### Scenario: Two hosts enrolling concurrently receive distinct secrets
- **WHEN** two Windows hosts enroll in Fleet MDM at the same time
- **THEN** each receives a different secret
- **AND** neither host's secret can be consumed by the other

#### Scenario: Feature disabled falls back to existing behavior
- **WHEN** `auth.use_one_time_enroll_secrets` is disabled
- **THEN** the fleetd install command carries the global enroll secret exactly as it does today

### Requirement: The secret binds to the MDM enrollment, not to a host row
Fleet SHALL mint the secret against the Windows MDM enrollment, identified by MDM device ID or MDM hardware ID, so that minting succeeds before any `hosts` row exists and while `mdm_windows_enrollments.host_uuid` is still empty.

#### Scenario: Minting succeeds for an unlinked enrollment
- **WHEN** an Entra or Autopilot enrollment is created with `host_uuid = ''` and no matching `hosts` row exists
- **THEN** Fleet mints a secret bound to that enrollment
- **AND** the fleetd install command is enqueued successfully

#### Scenario: Secret survives later host linkage
- **WHEN** an enrollment that already holds an unconsumed secret is later linked to a host row via DevDetail serial, orbit enroll, or osquery ingest
- **THEN** the secret remains valid and consumable by that host

### Requirement: Resolution reuses an unconsumed secret and mints once consumed
Fleet SHALL resolve an existing secret whose `consumed_at` is NULL to that same value, and SHALL mint a new secret only when the enrollment has no unconsumed secret. This matches the macOS behavior in `mintHostOneTimeEnrollSecret`.

#### Scenario: Re-delivery while the secret is live returns the same value
- **WHEN** the fleetd install command is re-enqueued for an enrollment whose secret has not been consumed
- **THEN** the command carries the identical secret value as the previous delivery
- **AND** no additional row is created

#### Scenario: Re-delivery after consumption mints a fresh secret
- **WHEN** the fleetd install command is re-delivered for an enrollment whose previous secret has `consumed_at` set
- **THEN** Fleet mints and delivers a new secret
- **AND** the previously consumed secret remains rejected

#### Scenario: Concurrent delivery does not mint twice
- **WHEN** two MDM sessions for the same enrollment request the secret simultaneously
- **THEN** exactly one secret row is created
- **AND** both sessions receive the same value

### Requirement: The secret carries the enrollment's team
Fleet SHALL scope the minted secret to the team the Windows MDM enrollment resolves to, so a secret recovered from one team cannot enroll a host into another.

#### Scenario: Enrollment with a configured default team
- **WHEN** a Windows MDM enrollment resolves to a configured Windows enrollment default team
- **THEN** the minted secret carries that team
- **AND** a host enrolling with it lands in that team

#### Scenario: Enrollment with no default team
- **WHEN** no Windows enrollment default team is configured
- **THEN** the minted secret carries no team and the host lands in no team

### Requirement: The capability is gated behind the existing Premium flag
Windows one-time enroll secrets SHALL be gated by `auth.use_one_time_enroll_secrets`, which is Premium-only today. The design SHALL NOT depend on teams existing, so the gate can later be opened to Fleet Free without reworking the Windows path.

#### Scenario: Non-Premium license disables the capability
- **WHEN** the server does not have a Premium license and the flag is set
- **THEN** the flag is forced off and Windows MDM continues to receive the global enroll secret

#### Scenario: No team configured still mints successfully
- **WHEN** a Windows MDM enrollment resolves to no team, as it always would on Fleet Free
- **THEN** the minted secret carries no team
- **AND** the host lands in the default (Unassigned) fleet

### Requirement: The secret is single-use across the orbit and osquery planes
Fleet SHALL allow the secret to be consumed exactly once on the orbit plane and once on the osquery plane, with the second plane required to follow within the established second-plane window. Any further presentation SHALL be rejected.

#### Scenario: Normal enrollment consumes both planes
- **WHEN** orbit enrolls with the secret and osqueryd enrolls with the same secret shortly after
- **THEN** both enrollments succeed
- **AND** the secret records use on both planes

#### Scenario: Replay from another machine is rejected
- **WHEN** a secret already consumed by its bound host is presented from a different machine
- **THEN** enrollment is refused
- **AND** the rejection reason is recorded

#### Scenario: Replay claiming the bound host's identifiers is rejected
- **WHEN** a consumed secret is presented from a different machine that claims the bound host's UUID and serial
- **THEN** enrollment is refused

#### Scenario: Second plane after the window is rejected
- **WHEN** the osquery plane presents the secret after the second-plane window has elapsed
- **THEN** enrollment is refused as spent

### Requirement: The plaintext secret is not stored in MDM commands
Fleet SHALL store a placeholder rather than the secret value in the queued Windows MDM command, and SHALL expand it per enrollment at delivery time, so the plaintext secret is not at rest in `windows_mdm_commands` and is not returned by the MDM command-results API.

#### Scenario: Stored command contains no credential
- **WHEN** a fleetd install command is enqueued for a Windows enrollment
- **THEN** `windows_mdm_commands.raw_command` contains the placeholder and not the secret value

#### Scenario: Command-results API does not disclose the secret
- **WHEN** an authorized user reads the command results for that command
- **THEN** the returned payload does not contain the secret value

#### Scenario: Delivered SyncML contains the expanded secret
- **WHEN** the device retrieves the pending command
- **THEN** the delivered document contains the secret minted for that enrollment

### Requirement: One-time secrets stay out of administrator surfaces
Machine-issued one-time secrets SHALL NOT appear in the enroll secrets UI, `fleetctl get enroll_secrets`, GitOps output, or any enroll-secret API response.

#### Scenario: Enroll secret listings exclude one-time secrets
- **WHEN** an administrator lists enroll secrets through the UI, the API, or `fleetctl`
- **THEN** no machine-issued one-time secret is included

#### Scenario: GitOps generation excludes one-time secrets
- **WHEN** `fleetctl generate-gitops` runs against a Fleet with outstanding one-time secrets
- **THEN** the generated output contains none of them

### Requirement: Users cannot place the host-secret placeholder in their own profiles
Fleet SHALL continue to reject user-supplied Windows profiles that reference a `$FLEET_HOST_SECRET_*` variable, so the placeholder remains usable only by Fleet-managed content.

#### Scenario: Windows profile upload with the placeholder is rejected
- **WHEN** a user uploads a Windows configuration profile containing `$FLEET_HOST_SECRET_ENROLL_SECRET`
- **THEN** the upload is refused with an error stating the variable is reserved

#### Scenario: GitOps batch upload with the placeholder is rejected
- **WHEN** the same profile is applied through the batch or GitOps path
- **THEN** the apply is refused with the same error
