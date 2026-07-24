## ADDED Requirements

### Requirement: Per-platform managed local account enablement
The system SHALL expose a `managed_local_account_settings` object with an `enabled` boolean (default `false`) under `mdm.apple_settings` and `mdm.windows_settings`, for both global config and teams, settable through the REST API, the Fleet UI, and GitOps.

#### Scenario: Enable for Windows via config PATCH
- **WHEN** an admin on a premium license sends `PATCH /api/v1/fleet/config` with `mdm.windows_settings.managed_local_account_settings.enabled: true`
- **THEN** the value persists, is returned by `GET /api/v1/fleet/config`, and an `enabled_managed_local_account` activity is logged

#### Scenario: Disable for Windows
- **WHEN** an admin sets `mdm.windows_settings.managed_local_account_settings.enabled: false` for a team
- **THEN** the value persists, a `disabled_managed_local_account` activity is logged, and newly enrolling Windows hosts in that team no longer receive the account

### Requirement: Deprecated setup_experience fields alias the Apple values
The system SHALL treat `setup_experience.enable_managed_local_account` and `setup_experience.end_user_local_account_type` as deprecated aliases of `apple_settings.managed_local_account_settings.enabled` and `apple_settings.end_user_local_account_type`. Writes on either surface MUST converge to one stored value, and reads MUST return consistent values on both surfaces across every write path (config PATCH, team PATCH, setup-experience PATCH, GitOps team apply).

#### Scenario: Legacy write reflects on the new surface
- **WHEN** a client sets `setup_experience.enable_managed_local_account: true` via `PATCH /setup_experience`
- **THEN** `GET /api/v1/fleet/config` returns `true` for both `setup_experience.enable_managed_local_account` and `apple_settings.managed_local_account_settings.enabled`

#### Scenario: Conflicting writes rejected
- **WHEN** one payload sets the deprecated field and the new Apple field to different values
- **THEN** the API returns a 422 validation error and persists nothing

#### Scenario: Deprecated field does not control Windows
- **WHEN** a client sets `setup_experience.enable_managed_local_account: true` and `windows_settings.managed_local_account_settings` is absent
- **THEN** the Windows setting remains `false`

### Requirement: Apple end user account type validation
The system SHALL accept only `admin`, `standard`, or `none` for `apple_settings.end_user_local_account_type` (default `admin`), SHALL require the Apple managed local account to be enabled when the type is `standard` or `none`, and SHALL reject an account type on `windows_settings`.

#### Scenario: Standard without managed account rejected
- **WHEN** a payload sets `end_user_local_account_type: "standard"` while the Apple `managed_local_account_settings.enabled` resolves to `false`
- **THEN** the API returns a validation error naming the requirement

#### Scenario: Account type on Windows rejected
- **WHEN** a payload includes an end user account type under `windows_settings`
- **THEN** the API returns a validation error

### Requirement: Premium gating
The system SHALL reject changes to `managed_local_account_settings` (either platform) and `apple_settings.end_user_local_account_type` on non-premium licenses.

#### Scenario: Free tier rejected
- **WHEN** a Fleet Free deployment sends `PATCH /api/v1/fleet/config` changing `windows_settings.managed_local_account_settings.enabled`
- **THEN** the API responds with the missing-license error and persists nothing

### Requirement: GitOps round trip
The GitOps pipeline SHALL accept the new nested keys and the deprecated `setup_experience` spellings on input, SHALL persist the Windows setting for team-scoped applies, and `fleetctl generate-gitops` SHALL emit `managed_local_account_settings` under each enabled platform section (plus `end_user_local_account_type` under `apple_settings` when not `admin`) instead of a TODO placeholder.

#### Scenario: Team-scoped gitops persists the Windows setting
- **WHEN** `fleetctl gitops` applies a team YAML with `controls.windows_settings.managed_local_account_settings.enabled: true`
- **THEN** the team config stores the value and `GET /api/v1/fleet/teams/{id}` returns it

#### Scenario: Lossless generate and re-apply
- **WHEN** the feature is enabled for both platforms and `fleetctl generate-gitops` runs, and the generated YAML is applied back with `fleetctl gitops`
- **THEN** the generated YAML contains the nested fields with no TODO placeholder for them, and the re-apply produces no configuration change
