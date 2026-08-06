# Design: Windows Autopilot pending hosts

All Microsoft Graph behavior below was verified live against a real Entra tenant with five registered Autopilot devices on 2026-08-06.
Findings marked **[verified]** were observed directly; findings marked **[inferred]** follow from an observed mechanism but were not
reproduced.

## 1. Credential model

### Why not reuse `windows_entra_tenant_ids` / `windows_entra_client_ids`

These are **inbound authorization allowlists**. At enrollment, Fleet validates the Entra JWT the device presents:
`hasAuthorizedAzureTenant` (`server/service/microsoft_mdm.go:950`) matches the `tid` claim against the tenant list, and
`hasAuthorizedAzureAudience` (`:921`) matches `aud` against the client ID list. They answer "who may enroll into me".

The Graph sync runs the opposite direction: Fleet authenticating *as* one app *in* one tenant. The lists are unordered, not index-aligned
with each other, and a single tenant may legitimately hold several allowlisted client IDs (cutover between app registrations, v1/v2 token
transition). There is no unambiguous pair to read. Unifying them would also be a breaking change to two shipped fields that already have UI,
GitOps, and activities.

They also need not overlap. Under least privilege the customer registers a Graph-only app holding just
`DeviceManagementServiceConfig.Read.All`, and that client ID must **not** be added to the enrollment allowlist, since it never issues an
enrollment token.

### Config shape

```yaml
controls:
  windows_entra_tenant_ids: [...]      # unchanged, inbound
  windows_entra_client_ids: [...]      # unchanged, inbound
  microsoft_graph_credentials:         # new, outbound. List shape, max 1 entry for now.
    - tenant_id: 5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4
      client_id: 7f6b1665-51f5-48de-a9b6-ac17539583fb
      client_secret: $WINDOWS_ENTRA_CLIENT_SECRET_ACME
```

API surface is `mdm.microsoft_graph_credentials` on `GET`/`PATCH /config`. The `$VAR` is ordinary GitOps environment interpolation
(`ExpandEnv`, `pkg/spec/spec.go:301`), deliberately not a `$FLEET_SECRET_*` variable, since those are host-delivery variables and this value
never leaves the server.

### One credential per tenant

**[verified]** Two different app registrations in the same tenant returned byte-identical device payloads. The Autopilot registry is scoped
to the tenant, not the app, so a second credential for the same tenant reads the same data. Enforce `UNIQUE (tenant_id)` in the schema and
reject a duplicate `tenant_id` in the incoming list with a 422.

A duplicate would not create duplicate hosts (reconciliation is keyed on tenant plus device), but it doubles Graph calls against Microsoft's
throttling limits and makes failure reporting incoherent: one revoked secret would surface errors for a tenant that is otherwise syncing.

### Capped at one credential for this release

The list carries **at most one entry for now**. A configuration with two or more entries SHALL be rejected with a 422. Multi-tenant sync is
deliberately deferred.

The shape stays a list precisely so lifting the cap later is a validation change and nothing else: no key rename, no type change from scalar
to list, no API break, and no migration, since the table is already keyed on `tenant_id` and the sync already iterates. This is the whole
reason not to ship the scalar the merged docs describe, even though only one credential is supported today.

Everything downstream is written to the multi-entry shape regardless: the table, the per-tenant sync loop, per-credential failure isolation,
and `host_autopilot_devices.tenant_id`. The cap is enforced at exactly one place, config validation, so removing it is a one-line change plus
the UI going from one form to a list.

### Storage

Do **not** use `mdm_config_assets`. It is keyed `UNIQUE (name, deletion_uuid)` and structurally holds one row per credential name. Fleet has
already been through this transition twice: `MDMAssetABMTokenDeprecated` (`server/fleet/mdm.go:1044`) and `MDMAssetVPPTokenDeprecated`
(`:1053`) are the residue of ABM and VPP tokens moving out to dedicated tables when they went multi-instance.

Follow `abm_tokens` / `vpp_tokens` instead: a dedicated table with an encrypted secret column keyed on the external identity.

```sql
CREATE TABLE `microsoft_graph_credentials` (
  `id`             int unsigned NOT NULL AUTO_INCREMENT,
  `tenant_id`      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `client_id`      varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `client_secret`  blob NOT NULL,             -- encrypted with the server private key, as abm_tokens.token
  `credential_invalid` tinyint(1) NOT NULL DEFAULT '0',   -- drives the app-wide banner, as abm_tokens.token_invalid
  `last_synced_at` timestamp NULL DEFAULT NULL,
  `last_sync_error` text COLLATE utf8mb4_unicode_ci,
  `created_at`     timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_microsoft_graph_credentials_tenant_id` (`tenant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

### Validation, masking, premium

Mirror `apple_account_provisioning` (`server/service/appconfig.go:499-548`, `:801-854`), which is the closest and most recent precedent for a
config-adjacent secret:

- `tenant_id` and `client_id` must be GUIDs. Reuse `windowsEntraGUIDRegex` from `server/service/microsoft_mdm.go`.
- Premium gate: reject writes with `ErrMissingLicense` when not premium, alongside the existing `windows_entra_*` gates
  (`server/service/appconfig.go:2033-2038`).
- Require `server.PrivateKey` to be configured when a new secret is supplied, matching `appconfig.go:831`.
- Mask on read: extend `AppConfig.Obfuscate()` (`server/fleet/app.go:867`) to replace each entry's `client_secret` with
  `fleet.MaskedPassword`. On write, an incoming value equal to the mask preserves the stored secret.
- On a new or changed credential, call `VerifyCredential` (one token mint plus one page) and return `NewInvalidArgumentError` on failure,
  matching how Jira/Zendesk credentials are validated on config write.
- Activities `added_`/`edited_`/`deleted_microsoft_graph_credential`, written through the activity service, never `ds.NewActivity` directly.

## 2. Microsoft Graph client

New package `server/microsoft/msgraph`. Uses `golang.org/x/oauth2/clientcredentials` (`golang.org/x/oauth2 v0.36.0` is already in `go.mod`;
this is its first use in the tree). Token URL `https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token`, scope
`https://graph.microsoft.com/.default`. Transport via `fleethttp`, never a bare `http.Client`. Errors wrapped with `ctxerr`.

Expose a factory taking a credential and returning a `Client`, mirroring `GoogleWorkspaceDirectoryFactory`
(`server/cron/google_workspace_cron.go:28`), so the cron does not import the concrete client and tests can inject a fake.

### Pagination is hazardous and must not be implemented naively

`GET /v1.0/deviceManagement/windowsAutopilotDeviceIdentities` paginates with `$skiptoken=LastSerialNumber='<serial>'`, an **inclusive**
cursor ordered by serial number.

- **[verified]** At `$top=2`, following `@odata.nextLink` returned **7 rows for 5 devices**. The boundary device is repeated on the next
  page.
- **[verified]** At `$top=1`, the returned `@odata.nextLink` is **byte-identical to the URL just requested**. A loop that follows `nextLink`
  until it is absent never terminates.
- **[inferred]** Because the cursor is a serial and serials are not unique, a run of devices sharing one serial that meets or exceeds the
  page size prevents the cursor from advancing, producing the same loop at any page size. This is not exotic: the test tenant contains a
  device with serial `Default string`, which is a real GMKtec M5 PLUS mini-PC, not a VM, and Fleet already classifies that string as junk in
  `placeholderHardwareSerials` (`server/fleet/hosts.go:67`).

Requirements:

1. Deduplicate by the Autopilot `id` while paging. `id` is the registration's own identifier and is distinct from
   `azureActiveDirectoryDeviceId`.
2. Carry an explicit loop guard: stop when `nextLink` equals the URL just requested, when a page yields no previously unseen `id`, or on a
   hard page cap. Log when a guard fires.
3. Do not set a small `$top`. Use the service default.
4. Never assume serial ordering or uniqueness anywhere in reconciliation.

### Response shape

Fields present on every record **[verified]**: `id`, `groupTag`, `serialNumber`, `azureActiveDirectoryDeviceId`, `managedDeviceId`,
`enrollmentState`, `manufacturer`, `model`, `systemFamily`, `skuNumber`, `productKey`, `purchaseOrderIdentifier`, `displayName`,
`addressableUserName`, `userPrincipalName`, `resourceName`, `lastContactedDateTime`.

Fleet consumes `id`, `serialNumber`, `groupTag`, `azureActiveDirectoryDeviceId`. Notes:

- `managedDeviceId` uses the all-zeros GUID `00000000-0000-0000-0000-000000000000` as its "none" sentinel, not null.
- `enrollmentState` reflects *Intune* enrollment, not Fleet enrollment. Do **not** filter on it: a device that once enrolled in Intune is
  still a legitimate pending host for Fleet. The test tenant has one `enrolled` device and four `notContacted`.
- Real serials observed span `0000-0000-4226-9691-9034-3814-15`, `VICTOR1776257483`, `Default string`, and
  `VMware-56 4d 51 82 9a 05 fe b6-01 de 74 fe 6a 4b 4e c6` (53 characters, embedded spaces). Matching must be exact and space-preserving.

### Failure taxonomy

All statuses **[verified]** against the live tenant.

| Layer | Signal | Meaning | Response |
|---|---|---|---|
| Entra token endpoint | `AADSTS7000215` | Wrong secret | Record on the credential, surface to admin |
| Entra token endpoint | `AADSTS7000222` | Expired secret | Record on the credential, surface to admin |
| Graph | 401 | Token rejected | Record, retry next cycle |
| Graph | 403 | Valid token, permission or consent missing | Record with a distinct message, admin action |
| Graph | 429 / 5xx | Transient | Retry quietly, honor `Retry-After` |

Graph returns **two different error codes for the same 403 cause**: `Authorization_RequestDenied` on directory endpoints and `Forbidden` on
Intune endpoints. Do not key permission detection on a single code string; key on the HTTP status.

In every failure case the tenant's existing pending hosts are left untouched. Only a successful 200 may remove a host.

### Surfacing a bad credential to the admin

Product decision: use the **app-wide banner**, the same treatment as an invalid ABM token. Figma to follow.

The precedent is exact and should be mirrored rather than reinvented:

| ABM | Windows Graph |
|---|---|
| `abm_tokens.token_invalid` (migration `20260721090128`) | `microsoft_graph_credentials.credential_invalid` |
| `SetABMTokenInvalidForOrgName` / `IsABMTokenInvalidForOrgName` (`apple_mdm.go:6608`, `:6627`) | same pair keyed on `tenant_id` |
| `ABMToken.TokenInvalid` json `token_invalid` (`server/fleet/mdm.go:200`) | `credential_invalid` on the credential response |
| `hasInvalidABMToken` computed in `App.tsx:150` | same shape for the Graph credential |
| `<AppleBMTokenInvalidMessage orgNames={...} />` in the `MainContent.tsx` priority chain | new banner component in the same chain |

`MainContent.tsx:59-83` shows exactly one banner at a time in a fixed priority order, and every banner except the APNs one sits inside the
`isPremiumTier` branch. The Graph credential banner belongs in that premium branch. Its position in the priority order is a product call;
placing it after the VPP banner and before the license-expiry banner is the natural default, since it is narrower in blast radius than the
Apple MDM banners above it.

The flag is set by the sync, not at config-write time. A credential is verified on write (see section 1), so it is valid when stored; it goes
bad later through expiry, revocation, or a permission change. Set `credential_invalid = 1` on an authentication or authorization failure
(token-endpoint `AADSTS*` errors, Graph 401, Graph 403) and clear it on the next successful sync. Do **not** set it for transient 429 or 5xx,
or a Microsoft outage would raise a credential alarm across every deployment.

**Open question for product.** The ABM banner names the offending token by `org_name`, which is human-readable. A Graph credential's only
identity is the tenant GUID, which is not. Options: show the GUID, add an admin-supplied display label to the credential, or read the tenant
name from Graph (`/organization`), which would require another permission and undercut the least-privilege setup. With the cap at one
credential this barely matters, since the banner can simply say "the Microsoft Graph credential", but it needs deciding before the cap is
lifted.

## 3. Sync cron

`CronMicrosoftAutopilotSync CronScheduleName = "microsoft_autopilot_sync"` in `server/fleet/cron_schedules.go`. New
`server/cron/microsoft_autopilot_cron.go` modeled on `google_workspace_cron.go`, registered in `cmd/fleet/cron_registration.go` inside
`registerMDMCrons`. Default interval 5 minutes.

The job no-ops when no credentials are configured or the license is not premium. It then iterates credentials, one tenant at a time.

- **Failures are isolated per tenant.** One bad credential must not stop the others, and must not affect that tenant's existing hosts.
- **Empty-response guard.** A successful response listing zero devices must not delete every pending host for that tenant. Mirror the
  `len(gwUsers) == 0` guard at `google_workspace_cron.go:209`. Note this is the legitimate steady state for a tenant with no registrations,
  so it must log at most once, not every cycle.
- Reconciliation per tenant: create new, update changed group tags in place (the tag is mutable in Intune), remove absent.

### Placeholder serials

A device whose `serialNumber` satisfies `fleet.IsPlaceholderHardwareSerial` must not be used to create or match a pending host by serial. Two
such devices would otherwise collapse onto one host row. `tryLinkUnlinkedEnrollmentFromDevDetail` already skips these serials for the same
reason (`server/service/microsoft_mdm.go`, the `IsPlaceholderHardwareSerial` check).

Open decision for implementation: either skip such devices entirely (simplest, and they cannot be reconciled at enrollment anyway since the
serial is the only join key available pre-osquery), or create the pending host but exclude it from serial matching. This design recommends
**skipping with a logged count**, since a pending host that can never reconcile is worse than an absent one.

## 4. Pending host lifecycle

### Creation

New datastore method `IngestWindowsAutopilotDevices`, modeled on `IngestMDMAppleDevicesFromDEPSync`
(`server/datastore/mysql/apple_mdm.go:1901`). In a `withRetryTxx` transaction, per new device:

- `hosts` row with `platform='windows'`, `hardware_serial` set, `osquery_host_id = NULL`, and `last_enrolled_at`/`detail_updated_at` at the
  never-timestamp sentinel (`common_mysql.GetDefaultNonZeroTime()`) so `CleanupExpiredHosts` does not immediately delete it. Follow
  `createHostFromMDMDB` (`apple_mdm.go:1688`).
- `host_mdm` row with `enrolled=0, installed_from_dep=1`, which renders as `Pending` through the generated `enrollment_status` column
  (`schema.sql:924`). That column and the `filterHostsByMDM` pending filter (`hosts.go:1649`) are already platform-agnostic, and the platform
  guard at `hosts.go:1653` already includes `windows`.
- **Do not reuse `upsertMDMAppleHostMDMInfoDB` verbatim.** It resolves `server_url` via `ResolveAppleMDMURL` (`apple_mdm.go:2079`) to the
  Apple MDM path. `mobile_device_management_solutions` is keyed on `(name, server_url)`, so that would point the pending Windows host's
  `mdm_id` at the *Apple* Fleet solution. Resolve the Windows path (`/api/mdm/microsoft`, `server/mdm/microsoft/microsoft_mdm.go:14`)
  instead.
- "All Hosts" and "MS Windows" builtin label membership, so the host appears in host lists before osquery ever runs. Mirror
  `upsertMDMAppleHostLabelMembershipDB` (`apple_mdm.go:2120`), which looks labels up by name and tolerates their absence.
- `host_autopilot_devices` row carrying the group tag, Autopilot `id`, AAD device ID, serial, and tenant.

### Removal

Mirror `DeleteHostDEPAssignments` (`apple_mdm.go:2463`): hard-delete hosts that are still pending; for devices that already enrolled,
soft-delete only the `host_autopilot_devices` row and keep the host.

### `installed_from_dep` audit

Reusing the Apple marker drives the existing generated column with no schema change and correctly flips to `On (automatic)` on enrollment.
Every reference must be audited (`grep -rn installed_from_dep server/`) to confirm no Windows pending host leaks into an Apple-only query.
Spot checks already done: `apple_mdm.go:3857` is scoped `AND h.platform = 'darwin'`, and `filterHostsByMDMBootstrapPackageStatus`
(`hosts.go:1977`) ends with `AND hh.platform = 'darwin'`. The load-bearing risk is the three Windows `host_mdm` writers below.

## 5. Enrollment reconciliation

A pending Autopilot host must merge with the enrolling device at whichever entry point arrives first. Three paths currently mishandle it.

**1. Serial match (`matchHostDuringEnrollment`, `server/datastore/mysql/hosts.go:2297`).** The serial branch is restricted to
`platform IN ('darwin','ios','ipados')` and gated on `isAppleMDMEnabled` (`:2338`). `EnrollOrbit` additionally blanks the serial outright for
Windows (`:2401-2405`), as does `HostPreviouslyOrbitEnrolled` (`:2582-2585`).

Add a Windows branch scoped strictly to pending Autopilot hosts, and stop blanking the serial only when such a host exists for it:

```sql
SELECT id, ..., 2 priority, platform FROM hosts h
  JOIN host_mdm hm ON hm.host_id = h.id
  JOIN host_autopilot_devices had ON had.host_id = h.id AND had.deleted_at IS NULL
  WHERE h.hardware_serial = ? AND h.platform = 'windows'
    AND hm.enrolled = 0 AND hm.installed_from_dep = 1
  ORDER BY h.id LIMIT 1
```

The new branch must not depend on `isAppleMDMEnabled`. Note `orbitEnrollingWithOsqueryIdentifier` (`:2334`) suppresses serial matching
entirely when orbit sends an osquery identifier, which happens only under `--host-identifier=instance`
(`orbit/cmd/orbit/orbit.go:905-908`); the default path sends none, so the branch is reachable.

**2. Windows MDM DevDetail link (`tryLinkUnlinkedEnrollmentFromDevDetail`, `server/service/microsoft_mdm.go:1672`).** Already resolves the
host by serial and calls `LinkWindowsHostMDMEnrollment` (`server/service/osquery_utils/queries.go:3208`), which never sets `enrolled=1` and
calls `UpdateMDMInstalledFromDEP(hostID, false)` when `MDMNotInOOBE` (`:3233`), clearing the pending marker. Linking to a pending Autopilot
host must set `enrolled=1` and keep `installed_from_dep=1`.

**3. osquery detail ingest (`directIngestMDMWindows`, `server/service/osquery_utils/queries.go:2800`).** Derives `automatic` from
`aad_resource_id` presence and `MDMNotInOOBE`, and its comment at `:2826` asserts Pending is macOS-only. After a real Autopilot enrollment
this resolves to `automatic=true` and is harmless, but a reconciled host must be verified to stay `enrolled=1, installed_from_dep=1` rather
than being reset to `On (manual)`.

The `host_autopilot_devices` row is keyed by `host_id`, so the group tag survives all three transitions for free.

### Interaction with the Windows enrollment default fleet

`maybeAssignWindowsEnrollmentDefaultFleet` (`server/service/osquery_utils/queries.go:3269`) assigns the configured default fleet only when
`host.CreatedAt` is at or after the enrollment row's `CreatedAt`, on the assumption that MDM-first ordering means a freshly created host.

Autopilot pending hosts invert that: the sync creates the host row potentially days before enrollment, so `host.CreatedAt` precedes the
enrollment and the default fleet is **not** applied. That is the correct outcome for this story, since the whole point is that automation has
already placed the host in the right fleet and must not be overridden. But it is a behavior change for anyone relying on the Windows
enrollment default fleet who then turns on Autopilot sync: those hosts stop receiving the default fleet and stay in Unassigned unless the
automation moves them. This must be called out in the feature guide.

## 6. `group_tag` exposure

**The column must be `varchar(2048)`, not `varchar(255)`.** Intune's documented maximum is 2048 characters, confirmed by the customer, and the
test tenant now holds tags of 63, 127, 255, and 257 characters plus one empty. A `varchar(255)` column truncates silently.

Consequences:

- Do **not** index `group_tag`. At utf8mb4 a 2048-character column is 8192 bytes, over InnoDB's 3072-byte index key limit. If filtering by
  group tag is ever wanted, it needs a prefix index such as `KEY (group_tag(191))`.
- This retroactively strengthens the "separate table, not a column on `hosts`" decision. An 8 KB-capable column on Fleet's hottest table
  would be a genuine cost; in `host_autopilot_devices` it is not.

`host_autopilot_devices` gains the Autopilot `id` alongside the AAD device ID, and reconciliation keys on `id`, which is stable for the life
of the registration and unique, unlike the serial.

Expose `group_tag` on both `GET /hosts` (list) and `GET /hosts/:id` (detail) via
`LEFT JOIN host_autopilot_devices had ON had.host_id = h.id AND had.deleted_at IS NULL`. The list endpoint is the one that matters: the
customer's automation wants one paginated sweep followed by a batch transfer. Model the response field on `dep_assigned_to_fleet`
(`server/fleet/hosts.go:463`). Empty is the normal case, so the field is omitted or empty when absent, and the host details row is hidden.
Benchmark the list query to confirm the added join does not regress latency.
