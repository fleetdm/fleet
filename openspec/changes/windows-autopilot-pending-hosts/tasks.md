# Tasks: Windows Autopilot pending hosts

Grouped to match the four sub-issues on [#43481](https://github.com/fleetdm/fleet/issues/43481). Items marked **[spec change]** deviate from
the sub-issue as currently written and need the issue updated first.

## Track 1: foundation (#48849)

### Schema

- [x] One migration `AddWindowsAutopilotTables` creating both tables. **[spec change]** A single migration, and **no `_test.go`** (per
      Victor; note this contradicts `.claude/rules/fleet-database.md`). `Down` is a no-op. Run `make dump-test-schema`.
- [x] `mdm_microsoft_graph_credentials`: `UNIQUE (tenant_id)`, secret as encrypted `blob`, `datetime(6)` timestamps.
      **[spec change]** `mdm_` prefix, matching `mdm_apple_*`.
- [x] `host_autopilot_devices`: keyed on `host_id`, index on `hardware_serial`, `datetime(6)` timestamps.
      **[spec change]** `group_tag` is `varchar(2048)` not `varchar(255)`, unindexed, with a "do not narrow" comment (strict mode makes an
      over-long tag an error that fails the whole batch, not a truncation). `autopilot_device_id` holds the Graph `id`. The AAD column is
      named `entra_device_id`; only the Graph JSON tag keeps the legacy `azureActiveDirectoryDeviceId` spelling.
- [x] **[spec change] No foreign key to `hosts`.** Fleet forbids them and CI enforces it. Add `host_autopilot_devices` to `hostRefs`
      (`server/datastore/mysql/hosts.go`) **and seed it in `testHostsDeleteHosts`**, which asserts every `hostRefs` table has a row before
      the delete and breaks otherwise.

### Types and datastore

- [x] `fleet.MicrosoftGraphCredential` and `fleet.HostAutopilotDevice` types. **[spec change]** The Graph wire type
      `WindowsAutopilotDevice` lives in `server/microsoft/msgraph`, not `server/fleet`.
- [x] **[spec change]** No `MicrosoftGraphCredentials` field on `AppConfig.MDM`. Instead add `MDM.MicrosoftGraphCredentialInvalid`, a stored
      server-computed banner aggregate, with a `PATCH /config` carry-over so a client cannot set it.
- [x] Datastore methods for credential CRUD, plus a metadata read that decrypts nothing for the read path and the banner rollup.
- [x] **[spec change]** Device writes are batched only: `BatchUpsertHostAutopilotDevices` and `BatchSoftDeleteHostAutopilotDevices`, 1000 per
      statement via `common_mysql.BatchProcessSimple`. No single-row variant, since a 100k-device tenant would mean 100k round trips.
      `BatchSoftDeleteHostAutopilotDevices` is new: nothing could previously write `deleted_at` even though both reads filter on it.
- [x] **[spec change]** `RecordMicrosoftGraphSyncResult` takes `*string`, not `error`; the datastore stores a message, it does not classify.
- [x] **[spec change]** `UpsertMicrosoftGraphCredential` resets all sync state, since the service verifies before storing.
- [x] `make generate-mock`, then `go test ./server/service/` (uninitialized mocks crash sibling tests).

### Graph client

- [ ] `server/microsoft/msgraph`: client-credentials auth, `fleethttp` transport, `ctxerr` wrapping, factory mirroring
      `GoogleWorkspaceDirectoryFactory`.
- [ ] `VerifyCredential` (one token mint plus one page) and `ListWindowsAutopilotDevices`.
- [x] **[spec change]** Pagination guard, not naive `nextLink` following. Deduplicate by Autopilot `id`; detect a non-advancing cursor when
      `nextLink` equals the URL just requested or a page yields no unseen `id`; hard page cap as a backstop.
- [x] **[spec change]** A non-advancing cursor **returns an error**, it does not stop gracefully: a truncated list would make the sync delete
      the missing hosts. Pin `$top=1000` rather than using the service default.
- [x] **[spec change]** Reject a `nextLink` off the Graph origin without sending the token to it.
- [x] Tests against `httptest` covering: token acquisition; `@odata.nextLink` paging; **a `nextLink` identical to the request URL errors
      instead of looping**; **a boundary device repeated across pages is emitted once**; a foreign-origin next link is refused;
      `groupTag`/`serialNumber` parsing; 401 vs 403 vs 429 produce distinguishable wrapped errors.

### API, premium, activities, GitOps

- [x] **[spec change]** Dedicated endpoints, not a config field: `GET /microsoft_graph_credentials` and a declarative
      `PUT /microsoft_graph_credentials` accepting `dry_run`. Authorize both against the app config subject to preserve permissions.
- [x] Validate GUID format for `tenant_id`/`client_id`; reject duplicate `tenant_id` in the incoming list with a 422.
- [x] **Cap the list at one entry**, rejecting a second with a 422, enforced in exactly one place. Everything else is built for multi-entry.
- [x] Premium gate with `ErrMissingLicense`; require `server.PrivateKey` when a new secret is supplied.
- [x] Mask `client_secret` on the read endpoint; preserve the stored secret when the mask is sent back.
- [x] Call `VerifyCredential` on create/change only, so re-applying an identical config makes no network call.
- [x] Activities `added_`/`edited_`/`deleted_microsoft_graph_credential` via the activity service.
- [x] **[spec change]** Recompute the banner aggregate after every credential write and delete.
- [x] Authorization test covering the role matrix, including an anonymous caller (every authenticated role can read the config, so that is
      the only case distinguishing "authorized" from "authorization skipped").
- [x] GitOps: the YAML key `controls.microsoft_graph_credentials` is unchanged, but `DoGitOps` now lifts it out of controls and applies it
      through the endpoint, following `certificate_authorities`. The `spec.Group` field is a **pointer** so `fleetctl apply` leaves it nil
      and cannot delete a credential. The decode helper is not named `...Spec`. `generate-gitops` reads from the endpoint and emits a secret
      placeholder plus a `SecretWarning` per entry.

## Track 2: sync and reconciliation (#48850)

### Cron

- [ ] `CronMicrosoftAutopilotSync` schedule name; `server/cron/microsoft_autopilot_cron.go`; register in `registerMDMCrons`.
- [ ] No-op when unconfigured or not premium. Iterate credentials with **per-tenant failure isolation**.
- [ ] Empty-response guard so a zero-device response never deletes that tenant's pending hosts. Log at most once per credential, since zero
      devices is a legitimate steady state.
- [ ] Persist `last_synced_at` / `last_sync_error` per credential, distinguishing auth failure, permission failure, and transient errors.
- [ ] Set `credential_invalid` on auth (`AADSTS*`, 401) or authorization (403) failure; clear it on the next successful sync; never set it on
      429/5xx. Use `msgraph.Error`'s `IsAuthError` / `IsPermissionError` / `IsTransient` rather than re-classifying.
- [ ] **[spec change] Recompute the banner aggregate after every `SetMicrosoftGraphCredentialInvalid` call.** `mdm.microsoft_graph_credential_invalid`
      is stored, so it is only as fresh as its last recomputation. Track 1 covers the credential write paths and cannot cover the sync. Miss
      this and the table is correct, the credentials endpoint is correct, and the banner simply never appears — nothing errors. Test both
      directions end to end.
- [ ] **[spec change]** Do not swallow the Graph client's non-advancing-cursor error or substitute a partial list; that error exists so the
      sync never deletes against a truncated list.
- [ ] **[spec change]** Skip devices whose serial satisfies `fleet.IsPlaceholderHardwareSerial`, with a logged count. Such a host could never
      reconcile at enrollment.

### Pending host lifecycle

- [ ] `IngestWindowsAutopilotDevices` per `design.md` section 4. Critically: resolve the **Windows** MDM solution URL
      (`/api/mdm/microsoft`), not `ResolveAppleMDMURL`, or `mdm_id` points at the Apple Fleet solution.
- [ ] Builtin label membership ("All Hosts", "MS Windows") so pending hosts appear before osquery runs.
- [ ] Removal: hard-delete still-pending hosts; soft-delete only the `host_autopilot_devices` row for already-enrolled devices, via
      `BatchSoftDeleteHostAutopilotDevices`.
- [ ] **[spec change]** Use the batch datastore methods; there is no single-row variant. Diff against stored state and pass only changed
      devices. Note MySQL already suppresses the write for an unchanged row, so the diff saves wire bytes and row locks rather than write
      volume.
- [ ] **[spec change]** No FK cleanup to add: `host_autopilot_devices` is cleaned up through `hostRefs`, not a cascade.
- [ ] `installed_from_dep` audit documented, covering the three Windows `host_mdm` writers and the `mobile_device_management_solutions` row.

### Reconciliation

- [ ] Windows serial-match branch in `matchHostDuringEnrollment`, scoped to pending Autopilot hosts, independent of `isAppleMDMEnabled`.
- [ ] Stop blanking the serial in `EnrollOrbit` and `HostPreviouslyOrbitEnrolled` only when a pending Autopilot host exists for it.
- [ ] `tryLinkUnlinkedEnrollmentFromDevDetail` / `LinkWindowsHostMDMEnrollment`: set `enrolled=1` and keep `installed_from_dep=1` for a
      pending Autopilot host, rather than clearing it via `UpdateMDMInstalledFromDEP`.
- [ ] Verify `directIngestMDMWindows` leaves a reconciled host at `enrolled=1, installed_from_dep=1`.
- [ ] Tests: same host `id` reused via `EnrollOrbit` with group tag retained; same via the DevDetail path; osquery ingest does not reset the
      marker; **regression test that a Windows host with no pending Autopilot row still matches by UUID only, never by serial**.
- [ ] **[spec change]** Test the default-fleet interaction: a pending Autopilot host is *not* moved by
      `maybeAssignWindowsEnrollmentDefaultFleet`, because its `CreatedAt` precedes the enrollment row. Assert this deliberately so it cannot
      regress silently.

### Host responses

- [ ] `group_tag` on `GET /hosts` and `GET /hosts/:id` via `LEFT JOIN host_autopilot_devices`. Omitted or empty when absent.
- [ ] Benchmark the host-list query to confirm no latency regression.

## Track 3: frontend (#48851)

- [ ] Single-credential form (tenant ID, client ID, client secret) on the Microsoft Entra section. The cap at one keeps #48851's original
      three-inputs-and-Save scope intact; no list UI is needed for this release.
- [ ] **[spec change]** Read and write the credential through `GET`/`PUT /microsoft_graph_credentials`, not `configAPI`. It is not in
      `AppContext`, so the section fetches on mount and refetches after save. `PUT` is declarative: clearing the credential means sending an
      empty list, not calling a delete endpoint.
- [ ] **App-wide banner** when the credential is invalid. Add a component alongside `AppleBMTokenInvalidMessage` and insert it into the
      premium branch of the single-banner priority chain in `frontend/components/MainContent/MainContent.tsx:59-83`. **Blocked on Figma from
      Marko** for wording and priority position.
      **[spec change]** No client-side aggregation is needed: read the boolean `config.mdm.microsoft_graph_credential_invalid`, which
      `AppContext` already carries. This is simpler than `hasInvalidABMToken`, which scans a token list.
- [ ] Premium gating: keep `<PremiumFeatureMessage />` when `!isPremiumTier`.
- [ ] Wrap controls in `GitOpsModeTooltipWrapper` so they are read-only under `gitops_mode_enabled`.
- [ ] Masked secret placeholder; only send a new secret when the admin changes it.
- [ ] Per-credential sync status and error surfaced on each row.
- [ ] "Group tag" read-only row on host details, shown only when present. Add `group_tag` to `frontend/interfaces/host.ts`.
- [ ] Copy for labels, help text (link to the app-registration guide and the `DeviceManagementServiceConfig.Read.All` permission), and the
      save success/error flashes. New strings, no Figma; get approved before merge.
- [ ] `yarn test` for premium gating, GitOps mode, the credential round trip through the endpoint, the banner reading the config
      boolean, and the group tag row. `make lint-js` passes.

## Track 4: docs and QA (#48852)

- [ ] **[spec change]** Rewrite the merged docs PRs [#50518](https://github.com/fleetdm/fleet/pull/50518) and
      [#50519](https://github.com/fleetdm/fleet/pull/50519): scalar `windows_entra_graph_api_token` becomes the
      `microsoft_graph_credentials` list. Needs product sign-off first.
- [ ] **[spec change]** The REST API doc covers two **endpoints**, not config fields: `GET` and `PUT /microsoft_graph_credentials`. Call out
      explicitly that `PUT` is declarative (absent tenant is deleted, empty list clears, no `DELETE` endpoint), that the credential is
      verified against Graph before being stored, and that the masked secret round-trips.
- [ ] **[spec change]** Also document `mdm.microsoft_graph_credential_invalid` on `GET /config` as read-only and server-computed: a value
      sent to `PATCH /config` is ignored rather than rejected, so `fleetctl get config` output stays valid input.
- [ ] **[spec change]** The YAML doc notes that `controls.microsoft_graph_credentials` is declarative (removing the key deletes the stored
      credential) and that only `fleetctl gitops` manages it, never `fleetctl apply`.
- [ ] Audit log doc: the three new activities.
- [ ] Feature guide: app registration, `DeviceManagementServiceConfig.Read.All` with admin consent, Intune license requirement, and the
      transfer-on-pending automation. **Document the dedicated Graph-only app as the recommended setup** (verified least-privilege: 403 on
      `/users`, `/devices`, and `/deviceManagement/managedDevices`).
- [ ] **[spec change]** Document that Autopilot pending hosts do not receive the Windows enrollment default fleet, and why.
- [ ] Document client-secret expiry (180 days default, 24 months max) and that Fleet surfaces expiry as a sync failure after the fact.
- [ ] **[spec change]** Document that the banner can lag a real failure by up to one sync interval, and that saving a working credential
      clears it immediately.
- [ ] **[spec change]** Document that rotating a credential resets its sync history: `last_synced_at`, `last_sync_error`, and the invalid
      flag all clear, so the UI reads "never synced" until the next cycle. That is accurate, since the prior sync described a different
      credential.
- [ ] Usage statistics: no changes. State explicitly in the PR.
- [ ] End-to-end QA on a real tenant: pending hosts appear with group tags; enrollment transitions Pending → On (automatic) with no duplicate
      host and the tag retained; removal from Autopilot removes the pending host; premium blocked on both frontend and backend; GitOps
      generate-then-apply round-trips the credentials.
- [ ] **[spec change]** Credential-lifecycle QA, each covering an implementation decision: rotate the secret in Entra without updating Fleet
      (banner raises, then clears on update); remove the API permission (reported as a permission problem, distinct copy); delete the
      credential while pending hosts exist (document whatever happens); `fleetctl gitops` with the key removed deletes the credential while
      `fleetctl apply` with an unrelated file leaves it untouched.
- [ ] Confirmation comment on #43481.

## Blocked on product

- [ ] Figma for the invalid-credential banner: wording, and where it sits in the single-banner priority order.
- [ ] Sign-off on the scalar → list config change (capped at one) and the rename, since both merged docs PRs must be rewritten.
- [ ] Clarify "multiple tags" in the story's edge cases. Graph returns `groupTag` as a single string.
- [ ] Before the one-credential cap is ever lifted: decide how the banner names the offending credential. ABM uses a human-readable
      `org_name`; a Graph credential has only a tenant GUID.
