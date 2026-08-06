# Tasks: Windows Autopilot pending hosts

Grouped to match the four sub-issues on [#43481](https://github.com/fleetdm/fleet/issues/43481). Items marked **[spec change]** deviate from
the sub-issue as currently written and need the issue updated first.

## Track 1: foundation (#48849)

### Schema

- [ ] Migration `AddMicrosoftGraphCredentials`: table per `design.md` section 1, `UNIQUE (tenant_id)`, secret as encrypted `blob`.
- [ ] Migration `AddHostAutopilotDevices`: keyed on `host_id`, FK cascade to `hosts`, index on `hardware_serial`.
      **[spec change]** `group_tag` is `varchar(2048)` not `varchar(255)` (Intune max is 2048; test data already exceeds 255). Do not index
      it. Add an `autopilot_device_id` column for the Graph `id` alongside `azure_ad_device_id`.
- [ ] `_test.go` for each (`applyUpToPrev` → seed → `applyNext` → verify). `Down` is a no-op. Run `make dump-test-schema`.
      **[spec change]** No app-config seed migration is needed: the credential lives in its own table, not in `app_config_json`.

### Types and datastore

- [ ] `fleet.MicrosoftGraphCredential` and `fleet.HostAutopilotDevice` types. Add `MicrosoftGraphCredentials` to `AppConfig.MDM` and extend
      `AppConfig.Clone()` (`server/fleet/app.go`).
- [ ] Datastore methods for credential CRUD (list, upsert by tenant, delete, record sync result) and for
      `UpsertHostAutopilotDevice` / `ListHostAutopilotDevices(tenantID)` / `GetHostAutopilotDevice(hostID)`.
- [ ] `make generate-mock`, then `go test ./server/service/` (uninitialized mocks crash sibling tests).

### Graph client

- [ ] `server/microsoft/msgraph`: client-credentials auth, `fleethttp` transport, `ctxerr` wrapping, factory mirroring
      `GoogleWorkspaceDirectoryFactory`.
- [ ] `VerifyCredential` (one token mint plus one page) and `ListWindowsAutopilotDevices`.
- [ ] **[spec change]** Pagination guard, not naive `nextLink` following. Deduplicate by Autopilot `id`; stop when `nextLink` equals the URL
      just requested, when a page yields no unseen `id`, or on a hard page cap; do not set a small `$top`.
- [ ] Tests against `httptest` covering: token acquisition; `@odata.nextLink` paging; **a `nextLink` identical to the request URL terminates
      instead of looping**; **a boundary device repeated across pages is emitted once**; `groupTag`/`serialNumber` parsing; a 2048-character
      tag round-trips; 401 vs 403 vs 429 produce distinguishable wrapped errors.

### Config, premium, activities, GitOps

- [ ] Validate GUID format for `tenant_id`/`client_id`; reject duplicate `tenant_id` in the incoming list with a 422.
- [ ] Premium gate with `ErrMissingLicense`; require `server.PrivateKey` when a new secret is supplied.
- [ ] Extend `AppConfig.Obfuscate()` to mask each `client_secret`; preserve the stored secret when the masked value is sent back.
- [ ] Call `VerifyCredential` on create/change; reject with `NewInvalidArgumentError` on failure.
- [ ] Activities `added_`/`edited_`/`deleted_microsoft_graph_credential` via the activity service.
- [ ] GitOps: `controls.microsoft_graph_credentials` applies through the AppConfig path; `generate-gitops` emits the list with a secret
      placeholder plus a `SecretWarning` per entry, following `apple_account_provisioning.oauth_idp_client_secret`
      (`cmd/fleetctl/fleetctl/generate_gitops.go:1470-1487`). Round-trip test.

## Track 2: sync and reconciliation (#48850)

### Cron

- [ ] `CronMicrosoftAutopilotSync` schedule name; `server/cron/microsoft_autopilot_cron.go`; register in `registerMDMCrons`.
- [ ] No-op when unconfigured or not premium. Iterate credentials with **per-tenant failure isolation**.
- [ ] Empty-response guard so a zero-device response never deletes that tenant's pending hosts. Log at most once per credential, since zero
      devices is a legitimate steady state.
- [ ] Persist `last_synced_at` / `last_sync_error` per credential, distinguishing auth failure, permission failure, and transient errors.
- [ ] **[spec change]** Skip devices whose serial satisfies `fleet.IsPlaceholderHardwareSerial`, with a logged count. Such a host could never
      reconcile at enrollment.

### Pending host lifecycle

- [ ] `IngestWindowsAutopilotDevices` per `design.md` section 4. Critically: resolve the **Windows** MDM solution URL
      (`/api/mdm/microsoft`), not `ResolveAppleMDMURL`, or `mdm_id` points at the Apple Fleet solution.
- [ ] Builtin label membership ("All Hosts", "MS Windows") so pending hosts appear before osquery runs.
- [ ] Removal: hard-delete still-pending hosts; soft-delete only the `host_autopilot_devices` row for already-enrolled devices.
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

- [ ] **[spec change]** Credential **list** UI (add/delete rows) on the Microsoft Entra section, not three standalone inputs. Mirror the
      existing tenant and client ID list widgets on `WindowsAutomaticEnrollmentPage`. **Blocked on Figma from Marko.**
- [ ] Premium gating: keep `<PremiumFeatureMessage />` when `!isPremiumTier`.
- [ ] Wrap controls in `GitOpsModeTooltipWrapper` so they are read-only under `gitops_mode_enabled`.
- [ ] Masked secret placeholder; only send a new secret when the admin changes it.
- [ ] Per-credential sync status and error surfaced on each row.
- [ ] "Group tag" read-only row on host details, shown only when present. Add `group_tag` to `frontend/interfaces/host.ts`.
- [ ] Copy for labels, help text (link to the app-registration guide and the `DeviceManagementServiceConfig.Read.All` permission), and the
      save success/error flashes. New strings, no Figma; get approved before merge.
- [ ] `yarn test` for premium gating, GitOps mode, config round trip, and the group tag row. `make lint-js` passes.

## Track 4: docs and QA (#48852)

- [ ] **[spec change]** Rewrite the merged docs PRs [#50518](https://github.com/fleetdm/fleet/pull/50518) and
      [#50519](https://github.com/fleetdm/fleet/pull/50519): scalar `windows_entra_graph_api_token` becomes the
      `microsoft_graph_credentials` list. Needs product sign-off first.
- [ ] Audit log doc: the three new activities.
- [ ] Feature guide: app registration, `DeviceManagementServiceConfig.Read.All` with admin consent, Intune license requirement, and the
      transfer-on-pending automation. **Document the dedicated Graph-only app as the recommended setup** (verified least-privilege: 403 on
      `/users`, `/devices`, and `/deviceManagement/managedDevices`).
- [ ] **[spec change]** Document that Autopilot pending hosts do not receive the Windows enrollment default fleet, and why.
- [ ] Document client-secret expiry (180 days default, 24 months max) and that Fleet surfaces expiry as a sync failure after the fact.
- [ ] Usage statistics: no changes. State explicitly in the PR.
- [ ] End-to-end QA on a real tenant: pending hosts appear with group tags; enrollment transitions Pending → On (automatic) with no duplicate
      host and the tag retained; removal from Autopilot removes the pending host; premium blocked on both frontend and backend; GitOps
      generate-then-apply round-trips the credentials.
- [ ] Confirmation comment on #43481.

## Blocked on product

- [ ] Figma for the multi-tenant credential list UI.
- [ ] Sign-off on the scalar → list config change and the rename, since both merged docs PRs must be rewritten.
- [ ] Clarify "multiple tags" in the story's edge cases. Graph returns `groupTag` as a single string.
