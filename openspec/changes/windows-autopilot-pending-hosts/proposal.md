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

- A new **`mdm.microsoft_graph_credentials`** config: a list of `{tenant_id, client_id, client_secret}` entries, one per Entra tenant, that
  Fleet uses to authenticate to Microsoft Graph. Premium only. Secrets stored encrypted and masked on read.
- A **Microsoft Graph client** that mints app-only tokens via OAuth2 client credentials and lists Windows Autopilot device identities.
- A **periodic sync cron** that reconciles each tenant's Autopilot registrations into pending Windows hosts, and removes hosts that leave
  the Autopilot list.
- **Enrollment reconciliation** so a pending host is reused, not duplicated, when the device actually enrolls, across all three entry points
  that can arrive first.
- **`group_tag`** exposed on the host list and host detail API responses, and shown read-only on host details.
- **GitOps** support for the credential list under `controls`, with `generate-gitops` emitting a secret placeholder.

## What does not change

- The existing `mdm.windows_entra_tenant_ids` and `mdm.windows_entra_client_ids` allowlists keep their current shape and meaning. They
  authorize *inbound* enrollment JWTs; the new credential is *outbound*. See `design.md` for why they are not unified.
- Fleet does not act on the group tag. No auto-transfer, no built-in automation. The tag is data for the customer's automation to consume.
- No fleetd/orbit changes.

## Deviations from the issue as written

Both need product sign-off before the docs subtask (#48852) lands.

1. **The credential is a list, not a single token.** The merged docs PRs ([#50518](https://github.com/fleetdm/fleet/pull/50518),
   [#50519](https://github.com/fleetdm/fleet/pull/50519)) define a scalar `controls.windows_entra_graph_api_token`. That cannot work for more
   than one tenant, and Fleet deliberately supports multi-tenant Windows enrollment today (story #39214, QA'd with two tenants). An Autopilot
   registry is per-tenant, so a scalar credential silently surfaces exactly one tenant's devices and no error anywhere explains the absence of
   the rest. Both docs PRs need rewriting, not extending.

2. **Renamed.** `windows_entra_graph_api_token` becomes `microsoft_graph_credentials[].client_secret`. There is no Microsoft product called
   "Entra Graph": the API is Microsoft Graph, the data is Intune's, and the credential is an Entra app registration. "Entra" adds no
   information once "graph" is present, and worse, visually groups an outbound credential with the two inbound allowlists it must be
   distinguished from. The value is a client secret, which PR #50519's own prose already calls it while the key said "token".

3. **The auth model is client credentials, not a pasted token.** The story says "UI field for user to provide Microsoft Graph API
   authentication token". A Graph access token expires in ~60 minutes (measured: `expires_in: 3599`), so a pasted token breaks within the
   hour. Fleet stores an app-registration credential and mints tokens on demand.

## Open questions for product

- **Multi-tenant UI.** #48851 scoped three inputs and a Save button. A credential list needs add/delete rows like the existing tenant and
  client ID lists on `WindowsAutomaticEnrollmentPage`. No Figma exists for this.
- **"Multiple tags" in the test plan.** The story's edge cases say "including empty, renamed, and multiple tags". Graph returns `groupTag` as
  a single string and Microsoft's model is one tag per device, so this line is either about a comma-separated convention inside the one
  string, or is a misconception. Needs clarifying before QA writes against it.
- **Proactive secret-expiry warning.** Client secrets expire (180 days default, 24 months max). Fleet cannot read the expiry without
  `Application.Read.All`, a second permission. This proposal surfaces expiry as a sync failure after the fact. An admin-entered expiry date
  would allow warning ahead of time, at the cost of relying on correct manual entry.
- **Filtering hosts by group tag.** Not requested; the automation sweeps the list endpoint. If it is ever wanted, `group_tag` needs a prefix
  index (see `design.md`).

## Risk

Medium, concentrated in the shared Windows enrollment host-matching path (`matchHostDuringEnrollment`, `EnrollOrbit`, the Windows MDM
DevDetail link, and osquery MDM ingest). A defect there merges the wrong hosts or duplicates a host at enrollment, affecting Windows hosts
that have nothing to do with Autopilot. Mitigated by scoping every new match strictly to pending Autopilot hosts and by regression tests
asserting legacy Windows enrollment is untouched.
