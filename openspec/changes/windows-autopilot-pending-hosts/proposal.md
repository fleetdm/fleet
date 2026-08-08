# Show Windows Autopilot devices as "Pending" hosts

Story: [#43481](https://github.com/fleetdm/fleet/issues/43481). Sub-tasks: #48849 (foundation), #48850 (sync and reconciliation),
#48851 (frontend), #48852 (docs and QA).

## Why

IT admins register Windows devices with Windows Autopilot before those devices are ever powered on, and tag them at that moment with a
**group tag** that encodes intent ("Engineering", "Sales"). Today Fleet knows nothing about a device until it enrolls, so there is no way to
place it in the right fleet ahead of time. `easterwood` and `smither` both want to build automation that transfers hosts to the correct fleet
*before* enrollment, and cannot, because the hosts do not exist in Fleet yet.

Fleet already solves exactly this for Apple: ABM-assigned devices appear as `Pending` hosts before enrollment. This change brings the same
capability to Windows, sourced from Microsoft Graph instead of Apple Business Manager, and carries the Autopilot group tag through so
automation has something to key on.

## What changes

- A new **`/microsoft_graph_credentials`** resource (`GET` to read, declarative `PUT` to reconcile): a list of
  `{tenant_id, client_id, client_secret}` entries, one per Entra tenant, that Fleet uses to authenticate to Microsoft Graph. **Capped at one
  entry for this release**; the list shape exists so lifting the cap later is a validation change rather than an API break. Premium only.
  Secrets stored encrypted in a dedicated table and masked on read. It is deliberately **not** a field on `AppConfig` — see `design.md`
  section 1.
- An **app-wide banner** when a stored credential goes bad, mirroring the existing invalid-ABM-token banner, driven by a single
  server-computed `mdm.microsoft_graph_credential_invalid` boolean on the app config.
- A **Microsoft Graph client** that mints app-only tokens via OAuth2 client credentials and lists Windows Autopilot device identities.
- A **periodic sync cron** that reconciles each tenant's Autopilot registrations into pending Windows hosts, and removes hosts that leave
  the Autopilot list.
- **Enrollment reconciliation** so a pending host is reused, not duplicated, when the device actually enrolls, across all three entry points
  that can arrive first.
- **`group_tag`** exposed on the host list and host detail API responses, and shown read-only on host details.
- **GitOps** support for the credential list under `controls`, applied through the new endpoint rather than the app config (the
  `certificate_authorities` pattern), with `generate-gitops` emitting a secret placeholder.

## What does not change

- The existing `mdm.windows_entra_tenant_ids` and `mdm.windows_entra_client_ids` allowlists keep their current shape and meaning. They
  authorize *inbound* enrollment JWTs; the new credential is *outbound*. See `design.md` for why they are not unified.
- Fleet does not act on the group tag. No auto-transfer, no built-in automation. The tag is data for the customer's automation to consume.
- No fleetd/orbit changes.

## Deviations from the issue as written

Items 1 and 2 need product sign-off before the docs subtask (#48852) lands. Items 3 and 4 are engineering decisions recorded here for
review, not sign-off gates.

1. **The credential is a list, not a single token, even though only one is supported today.** The merged docs PRs
   ([#50518](https://github.com/fleetdm/fleet/pull/50518), [#50519](https://github.com/fleetdm/fleet/pull/50519)) define a scalar
   `controls.windows_entra_graph_api_token`. A scalar cannot ever grow to more than one tenant without an API break, and Fleet deliberately
   supports multi-tenant Windows enrollment today (story #39214, QA'd with two tenants), where an Autopilot registry is per-tenant. This
   release caps the list at one entry, so the shipped capability matches the merged docs; only the wire shape differs, and it differs
   specifically so that raising the cap later costs a validation change instead of a rename plus a type change. Both docs PRs need rewriting,
   not extending.

2. **Renamed.** `windows_entra_graph_api_token` becomes `microsoft_graph_credentials[].client_secret`. There is no Microsoft product called
   "Entra Graph": the API is Microsoft Graph, the data is Intune's, and the credential is an Entra app registration. "Entra" adds no
   information once "graph" is present, and worse, visually groups an outbound credential with the two inbound allowlists it must be
   distinguished from. The value is a client secret, which PR #50519's own prose already calls it while the key said "token".

3. **The credential is a dedicated resource, not a config field.** An earlier draft of this proposal put it on `GET`/`PATCH /config`. It was
   built that way and then moved during implementation. The client secret is encrypted at rest, so it cannot live in `app_config_json`, which
   meant the config field had to be hydrated from the credentials table on every config read, which needed a cache, which needed the sync to
   invalidate that cache — machinery that existed only to hold the field in the wrong place. ABM, VPP, and certificate authorities all keep
   the credential and its per-instance status behind their own endpoints, and `certificate_authorities` shows a key can stay in the GitOps
   YAML while being applied through its own endpoint. The GitOps contract is unchanged; only the HTTP surface moved. This does **not** need
   product sign-off, since the YAML the docs describe is unaffected, but the REST API doc must describe endpoints instead of config fields.

4. **The auth model is client credentials, not a pasted token.** The story says "UI field for user to provide Microsoft Graph API
   authentication token". A Graph access token expires in ~60 minutes (measured: `expires_in: 3599`), so a pasted token breaks within the
   hour. Fleet stores an app-registration credential and mints tokens on demand.

## Open questions for product

- **Banner placement and wording.** Product chose the app-wide banner for a bad credential, matching the invalid-ABM-token treatment; Figma
  to follow. Two details still need deciding: where it sits in `MainContent.tsx`'s single-banner priority order, and how it names the
  offending credential. The ABM banner uses a human-readable `org_name`; a Graph credential's only identity is the tenant GUID. With the cap
  at one, the banner can just say "the Microsoft Graph credential" and defer this, but it must be settled before the cap is lifted.
- **"Multiple tags" in the test plan.** The story's edge cases say "including empty, renamed, and multiple tags". Graph returns `groupTag` as
  a single string and Microsoft's model is one tag per device, so this line is either about a comma-separated convention inside the one
  string, or is a misconception. Needs clarifying before QA writes against it.
- **Proactive secret-expiry warning.** Resolved for reactive detection: a bad credential raises the banner. Still open is whether Fleet
  should warn *before* expiry. Client secrets expire (180 days default, 24 months max), and Fleet cannot read the expiry date without
  `Application.Read.All`, a second permission that would undercut the least-privilege setup. An admin-entered expiry date would allow an
  ahead-of-time warning at the cost of relying on correct manual entry.
- **Filtering hosts by group tag.** Not requested; the automation sweeps the list endpoint. If it is ever wanted, `group_tag` needs a prefix
  index (see `design.md`).

## Risk

Medium, concentrated in the shared Windows enrollment host-matching path (`matchHostDuringEnrollment`, `EnrollOrbit`, the Windows MDM
DevDetail link, and osquery MDM ingest). A defect there merges the wrong hosts or duplicates a host at enrollment, affecting Windows hosts
that have nothing to do with Autopilot. Mitigated by scoping every new match strictly to pending Autopilot hosts and by regression tests
asserting legacy Windows enrollment is untouched.
