## ADDED Requirements

### Requirement: Microsoft Graph credentials are configured per Entra tenant

Fleet SHALL expose `microsoft_graph_credentials` under `AppConfig.MDM`, a list whose entries each contain `tenant_id`, `client_id`, and
`client_secret`. Fleet SHALL use each entry to obtain app-only Microsoft Graph tokens via the OAuth2 client-credentials grant against
`https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token` with scope `https://graph.microsoft.com/.default`.

`tenant_id` and `client_id` SHALL be validated as GUIDs. The feature SHALL be Fleet Premium only.

The existing `windows_entra_tenant_ids` and `windows_entra_client_ids` allowlists SHALL be unchanged, and Fleet SHALL NOT derive Graph
credentials from them. A Graph `client_id` SHALL NOT be required to appear in `windows_entra_client_ids`.

#### Scenario: Premium instance stores a credential

- **GIVEN** a Fleet Premium instance with Windows MDM enabled
- **WHEN** an admin sends `PATCH /config` with one `mdm.microsoft_graph_credentials` entry containing valid GUIDs and a working secret
- **THEN** the credential SHALL be stored
- **AND** an `added_microsoft_graph_credential` activity SHALL be recorded

#### Scenario: Fleet Free rejects the write

- **GIVEN** a Fleet Free instance
- **WHEN** an admin sends `PATCH /config` with any `mdm.microsoft_graph_credentials` entry
- **THEN** the request SHALL be rejected with `ErrMissingLicense`

#### Scenario: Malformed GUID is rejected

- **GIVEN** a Fleet Premium instance
- **WHEN** an admin sends a credential whose `tenant_id` is not a GUID
- **THEN** the request SHALL be rejected with a validation error naming `mdm.microsoft_graph_credentials`

#### Scenario: Credential is verified before it is stored

- **GIVEN** an admin supplies a new or changed credential
- **WHEN** Fleet cannot obtain a token or cannot list a first page with it
- **THEN** the request SHALL be rejected with an invalid-argument error
- **AND** no credential SHALL be stored

### Requirement: At most one credential per tenant

Fleet SHALL enforce uniqueness on `tenant_id` in storage and SHALL reject a configuration containing a duplicate `tenant_id`. This is because
the Windows Autopilot device registry is scoped to an Entra tenant rather than to an application, so two credentials for the same tenant read
identical data while doubling API calls and making per-tenant failure reporting incoherent.

#### Scenario: Duplicate tenant is rejected

- **GIVEN** a Fleet Premium instance
- **WHEN** an admin sends `mdm.microsoft_graph_credentials` containing two entries with the same `tenant_id` and different `client_id` values
- **THEN** the request SHALL be rejected with a validation error
- **AND** neither entry SHALL be stored

### Requirement: At most one credential is accepted in this release

Fleet SHALL reject a configuration containing more than one `microsoft_graph_credentials` entry. The field SHALL remain a list so that
raising this cap later requires only a validation change, with no rename, no scalar-to-list type change, no API break, and no migration.
Storage, the sync loop, per-credential failure handling, and `host_autopilot_devices.tenant_id` SHALL all be implemented for the multi-entry
case regardless of the cap.

#### Scenario: A second credential is rejected

- **GIVEN** a Fleet Premium instance
- **WHEN** an admin sends `mdm.microsoft_graph_credentials` containing two entries with different `tenant_id` values
- **THEN** the request SHALL be rejected with a validation error explaining that only one credential is supported
- **AND** neither entry SHALL be stored

#### Scenario: A single credential is accepted

- **GIVEN** a Fleet Premium instance
- **WHEN** an admin sends exactly one valid entry
- **THEN** it SHALL be stored
- **AND** the sync SHALL pull Autopilot devices from that tenant

### Requirement: Client secrets are stored encrypted and never returned

Fleet SHALL store each `client_secret` encrypted at rest in a dedicated table rather than in `app_config_json` and rather than in
`mdm_config_assets`. Fleet SHALL require a configured server private key before accepting a new secret. Fleet SHALL replace every
`client_secret` with the masked value on read.

#### Scenario: Secret is masked on read

- **GIVEN** a stored credential
- **WHEN** an admin calls `GET /config`
- **THEN** the entry's `client_secret` SHALL be `********`
- **AND** the plaintext secret SHALL NOT appear anywhere in the response

#### Scenario: Resending the mask preserves the stored secret

- **GIVEN** a stored credential
- **WHEN** an admin sends `PATCH /config` with that entry's `client_secret` set to `********` and another field changed
- **THEN** the stored secret SHALL be unchanged
- **AND** no `edited_microsoft_graph_credential` activity SHALL be recorded for the secret alone

#### Scenario: Missing server private key blocks a new secret

- **GIVEN** a Fleet instance with no server private key configured
- **WHEN** an admin supplies a new `client_secret`
- **THEN** the request SHALL be rejected with an error directing the admin to configure the private key

### Requirement: Graph pagination terminates and yields each device once

Fleet SHALL NOT implement pagination as an unguarded "follow `@odata.nextLink` until absent" loop. Fleet SHALL deduplicate results by the
Autopilot device `id`, SHALL terminate when `@odata.nextLink` equals the URL just requested or when a page yields no previously unseen `id`,
and SHALL apply a hard page cap. Fleet SHALL NOT assume that serial numbers are unique or that results are ordered by anything it relies on.

This is because the Autopilot device identities collection paginates with an inclusive `$skiptoken=LastSerialNumber='<serial>'` cursor ordered
by serial number. Serial numbers are not unique, and the service can return an `@odata.nextLink` identical to the URL just requested.

#### Scenario: A self-referential nextLink terminates the walk

- **GIVEN** Graph returns a page whose `@odata.nextLink` is identical to the URL just requested
- **WHEN** the client walks pages
- **THEN** the walk SHALL terminate
- **AND** the client SHALL log that a pagination guard fired

#### Scenario: A device repeated across page boundaries is returned once

- **GIVEN** Graph returns a device as the last item of one page and the first item of the next
- **WHEN** the client walks pages
- **THEN** that device SHALL appear exactly once in the result
- **AND** the result count SHALL equal the number of distinct device `id` values

#### Scenario: Devices sharing a serial number do not stall the walk

- **GIVEN** several devices share one serial number
- **WHEN** the client walks pages
- **THEN** the walk SHALL terminate
- **AND** every distinct device `id` SHALL appear in the result

### Requirement: The sync reconciles Autopilot devices into pending Windows hosts

A periodic cron SHALL, for each configured credential, list that tenant's Autopilot devices and reconcile them: create a pending host for a
newly seen device, update a changed group tag in place, and remove a host for a device no longer present. The cron SHALL no-op when no
credential is configured or the license is not premium.

A device whose serial number is a known placeholder SHALL be skipped, with the skipped count logged, since such a host could never be matched
at enrollment.

#### Scenario: A new Autopilot device becomes a pending host

- **GIVEN** a configured credential and a tenant containing an Autopilot device with a real serial
- **WHEN** the sync runs
- **THEN** a host SHALL exist with `platform = 'windows'`, `host_mdm.enrolled = 0`, `host_mdm.installed_from_dep = 1`,
  `osquery_host_id IS NULL`, and `enrollment_status = 'Pending'`
- **AND** its `mdm_id` SHALL reference the Windows Fleet MDM solution, not the Apple one
- **AND** it SHALL be a member of the "All Hosts" and Windows builtin labels

#### Scenario: A changed group tag updates in place

- **GIVEN** a pending host synced from an Autopilot device
- **WHEN** the tag changes in Intune and the sync runs again
- **THEN** the stored group tag SHALL be the new value
- **AND** no additional host SHALL be created

#### Scenario: A device removed from Autopilot removes its pending host

- **GIVEN** a still-pending host synced from an Autopilot device
- **WHEN** the device is removed from Autopilot and the sync runs
- **THEN** the host SHALL be deleted

#### Scenario: An empty response never deletes pending hosts

- **GIVEN** a tenant with existing pending Autopilot hosts
- **WHEN** the sync receives a successful response listing zero devices
- **THEN** no pending host SHALL be deleted

#### Scenario: A placeholder serial is skipped

- **GIVEN** an Autopilot device whose serial is a known placeholder such as `Default string`
- **WHEN** the sync runs
- **THEN** no pending host SHALL be created for it
- **AND** the sync SHALL log that it skipped the device

#### Scenario: One failing tenant does not affect another

- **GIVEN** two configured credentials where one secret has been revoked
- **WHEN** the sync runs
- **THEN** the healthy tenant's devices SHALL reconcile normally
- **AND** the failing tenant's existing pending hosts SHALL be left untouched
- **AND** the failure SHALL be recorded against the failing credential only

### Requirement: Sync failures are surfaced without destroying data

Fleet SHALL record the outcome of each sync per credential. Fleet SHALL distinguish an authentication failure (bad or expired secret) from an
authorization failure (missing permission or admin consent) from a transient failure. Permission detection SHALL key on the HTTP status
rather than a single error-code string, because Graph returns different codes for the same 403 cause. In no failure case SHALL Fleet remove
that tenant's pending hosts.

#### Scenario: An expired secret is reported as a credential problem

- **GIVEN** a credential whose client secret has expired
- **WHEN** the sync runs
- **THEN** the credential's last sync error SHALL identify it as an expired or invalid secret requiring admin action
- **AND** the tenant's pending hosts SHALL remain

#### Scenario: A missing permission is reported distinctly

- **GIVEN** a credential whose app registration lacks `DeviceManagementServiceConfig.Read.All` or its admin consent
- **WHEN** the sync runs and Graph responds 403
- **THEN** the credential's last sync error SHALL identify it as a permission or consent problem
- **AND** detection SHALL NOT depend on which error code string Graph returned

#### Scenario: A transient failure is retried quietly

- **GIVEN** Graph responds 429 with `Retry-After`
- **WHEN** the sync runs
- **THEN** Fleet SHALL honor `Retry-After`
- **AND** SHALL NOT surface the condition to the admin as a credential problem

### Requirement: A bad credential raises an app-wide banner

Fleet SHALL mark a credential invalid when a sync fails to authenticate or is denied authorization, and SHALL surface that state as an
app-wide banner, matching the existing invalid-ABM-token treatment. The flag SHALL be cleared on the next successful sync. Fleet SHALL NOT
mark a credential invalid for transient failures, so that a Microsoft outage does not raise a credential alarm.

The banner SHALL only render for Fleet Premium, consistent with every other MDM banner in the single-banner priority chain.

#### Scenario: An expired secret raises the banner

- **GIVEN** a stored credential whose client secret has expired
- **WHEN** the sync runs and token acquisition fails
- **THEN** the credential SHALL be marked invalid
- **AND** an admin loading any page SHALL see the banner reporting the Microsoft Graph credential needs attention

#### Scenario: A revoked permission raises the banner

- **GIVEN** a stored credential whose Graph permission or admin consent has been removed
- **WHEN** the sync runs and Graph responds 403
- **THEN** the credential SHALL be marked invalid
- **AND** the banner SHALL be shown

#### Scenario: A transient outage does not raise the banner

- **GIVEN** a stored, valid credential
- **WHEN** the sync runs and Graph responds 429 or 5xx
- **THEN** the credential SHALL NOT be marked invalid
- **AND** no banner SHALL be shown

#### Scenario: Fixing the credential clears the banner

- **GIVEN** a credential marked invalid
- **WHEN** an admin stores a working secret and the next sync succeeds
- **THEN** the credential SHALL no longer be marked invalid
- **AND** the banner SHALL no longer be shown

### Requirement: A pending Autopilot host is reused when the device enrolls

When a Windows device that has a pending Autopilot host enrolls, Fleet SHALL reuse the existing host record rather than creating a second
one, at whichever entry point arrives first: orbit enrollment, the Windows MDM DevDetail serial link, or osquery detail ingest. The host
SHALL become `enrolled = 1` while retaining `installed_from_dep = 1`, and SHALL retain its group tag and its fleet assignment.

Serial matching for Windows SHALL be scoped strictly to hosts that are pending and have an Autopilot record. Windows hosts without one SHALL
continue to match by UUID or osquery identifier only.

#### Scenario: Orbit enrollment reuses the pending host

- **GIVEN** a pending Autopilot host with serial S
- **WHEN** a Windows device with serial S enrolls via orbit
- **THEN** the same host `id` SHALL be reused
- **AND** `enrollment_status` SHALL become `On (automatic)`
- **AND** the group tag SHALL be retained

#### Scenario: The Windows MDM DevDetail link reuses the pending host

- **GIVEN** a pending Autopilot host with serial S
- **WHEN** the Windows MDM session reports serial S via DevDetail
- **THEN** the enrollment SHALL link to that host
- **AND** the host SHALL be `enrolled = 1` with `installed_from_dep = 1`
- **AND** the pending marker SHALL NOT be cleared

#### Scenario: osquery ingest does not reset a reconciled host

- **GIVEN** a reconciled Autopilot host
- **WHEN** osquery detail ingest runs
- **THEN** the host SHALL remain `enrolled = 1` with `installed_from_dep = 1`

#### Scenario: A normal Windows host is never matched by serial

- **GIVEN** a Windows host with no pending Autopilot record but a serial equal to another host's
- **WHEN** it enrolls via orbit
- **THEN** it SHALL match by UUID or osquery identifier only
- **AND** it SHALL NOT be merged with the other host

#### Scenario: A transferred pending host keeps its fleet through enrollment

- **GIVEN** a pending Autopilot host that automation moved to a fleet
- **WHEN** the device enrolls
- **THEN** the host SHALL remain in that fleet
- **AND** the Windows enrollment default fleet SHALL NOT override it

### Requirement: The Autopilot group tag is exposed on host responses

Fleet SHALL return `group_tag` on both the host list and host detail responses for hosts with an Autopilot record, and SHALL omit or empty it
otherwise. The group tag SHALL be stored with capacity for Intune's documented maximum of 2048 characters.

#### Scenario: Group tag is returned on the list endpoint

- **GIVEN** a pending Autopilot host with a group tag
- **WHEN** an admin calls `GET /hosts`
- **THEN** the host entry SHALL include that `group_tag`

#### Scenario: Group tag is returned on the detail endpoint

- **GIVEN** a pending Autopilot host with a group tag
- **WHEN** an admin calls `GET /hosts/:id`
- **THEN** the response SHALL include that `group_tag`

#### Scenario: A host with no group tag omits the field

- **GIVEN** a host with no Autopilot record, or one whose tag is empty
- **WHEN** an admin calls `GET /hosts` or `GET /hosts/:id`
- **THEN** `group_tag` SHALL be absent or empty
- **AND** the host details UI SHALL NOT render a Group tag row

#### Scenario: A maximum-length tag round-trips without truncation

- **GIVEN** an Autopilot device whose group tag is 2048 characters
- **WHEN** the sync stores it and an admin reads the host
- **THEN** the returned `group_tag` SHALL be all 2048 characters
