# Conditional access: Google Cloud Identity

With Fleet, you can integrate with Google Cloud Identity to enforce Context-Aware Access (CAA) on macOS hosts.

When a host fails a policy in Fleet, Fleet writes a non-compliant signal into Google Cloud Identity. Workspace admins can then use Context-Aware Access policies to block access to Google Workspace apps and any SAML-federated SaaS until the issue is resolved.

Google Cloud Identity conditional access is supported even if you're not using MDM features in Fleet.

> **Supported editions.** The PATCH that writes the compliance signal requires one of: Cloud Identity Premium, Google Workspace Enterprise Standard or Plus, Google Workspace Education Standard or Plus, Google Workspace Enterprise for Education, or Frontline Standard or Plus. Tenants on Education Fundamentals, Business Starter/Standard/Plus, or Cloud Identity Free will receive `403 PERMISSION_DENIED` on the first sync.

## How resolution works

Fleet matches a host to its Cloud Identity device record using `host.hardware_serial`. The host must have been signed into Endpoint Verification (or another Google-managed surface like Drive for Desktop or Google Mobile Management) at least once for Google to have a record. Fleet then enumerates the deviceUsers under that device and matches by email to the user signed into Endpoint Verification on the host.

This means: **you don't need to install anything Fleet-specific on the device beyond the standard `fleetd` agent.** The integration uses signals Google already collects from devices that have signed into a Workspace identity.

## Step 1: Verify Endpoint Verification is deployed

Cloud Identity Conditional Access requires Endpoint Verification (EV) — Google's Chrome extension plus native helper that registers the device with Cloud Identity. Most Workspace customers already deploy EV by policy.

To confirm EV is installed on a host, log into Chrome with the user's Workspace identity and visit `chrome://policy` — look for `EndpointVerification` entries. The native helper writes to `~/Library/Application Support/Google/Endpoint Verification/accounts.json` on macOS.

If EV isn't deployed, see Google's [Endpoint Verification deployment guide](https://support.google.com/a/answer/9007320).

## Step 2: Create a Google Cloud service account

In any Google Cloud project you control:

```sh
gcloud iam service-accounts create fleet-cloud-identity \
  --display-name="Fleet Cloud Identity integration" \
  --project=YOUR_PROJECT
```

Generate a JSON key:

```sh
gcloud iam service-accounts keys create ~/fleet-cloud-identity-key.json \
  --iam-account=fleet-cloud-identity@YOUR_PROJECT.iam.gserviceaccount.com
```

Get the service account's OAuth client ID (numeric, **not** the email):

```sh
gcloud iam service-accounts describe \
  fleet-cloud-identity@YOUR_PROJECT.iam.gserviceaccount.com \
  --format='value(oauth2ClientId)'
```

You'll use this OAuth client ID in the next step.

## Step 3: Enable required APIs

In the same Google Cloud project, enable the Cloud Identity API:

```sh
gcloud services enable cloudidentity.googleapis.com --project=YOUR_PROJECT
```

## Step 4: Authorize Domain-Wide Delegation in Workspace

Domain-wide delegation (DWD) lets the service account act on behalf of a Workspace admin, which is required for the Cloud Identity device APIs.

In the Workspace admin console at [admin.google.com](https://admin.google.com), go to **Security > Access and data control > API controls > Manage Domain Wide Delegation** and click **Add new**.

For **Client ID**, paste the OAuth client ID from Step 2.

For **OAuth scopes**, paste this single line (no spaces):

```text
https://www.googleapis.com/auth/cloud-identity.devices
```

Click **Authorize**.

## Step 5: Find your customer ID

Your Cloud Identity customer ID starts with `C` and is shown at [admin.google.com](https://admin.google.com) under **Account > Account settings > Customer ID**.

You can also retrieve it programmatically:

```sh
curl -s -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://admin.googleapis.com/admin/directory/v1/customers/my_customer" | jq .id
```

## Step 6: Configure Fleet

Add the following block to your Fleet server configuration (YAML):

```yaml
google_cloud_identity:
  # Path to the service-account JSON key from Step 2.
  service_account_json: /path/to/fleet-cloud-identity-key.json

  # A Workspace super-admin email that Fleet will impersonate via DWD.
  # Recommended: a service-only super-admin (e.g. fleet-cloud-identity@example.com)
  # that no human logs into.
  impersonated_admin: fleet-cloud-identity@example.com

  # Your Cloud Identity customer ID, starting with `C` (from Step 5).
  customer_id: C0xxxxxxx

  # Suffix used in the ClientState resource name. Defaults to "fleet".
  partner_suffix: fleet

  # Comma-separated list of email domains Fleet emits signals for.
  # EV accounts on emails outside this list (personal Gmail, third-party
  # Workspace identities) are silently filtered.
  workspace_domains: example.com
```

Then restart the Fleet server.

There is no dedicated admin-console page for this integration in v1; runtime settings are managed via GitOps YAML or the AppConfig API. To enable conditional access for a team, set:

```yaml
# In your team YAML
integrations:
  google_cloud_identity_enabled: true
```

Or for "No team" hosts, set the same field at the org-level AppConfig:

```yaml
# In your global config / AppConfig YAML
integrations:
  google_cloud_identity_enabled: true
```

## Step 7: Flag policies for conditional access

In Fleet, go to **Policies**, edit the policies you want to enforce as conditional access signals, and enable **Conditional access**. Hosts will be marked compliant only when every flagged policy passes for that host.

Fleet shares the per-policy **Conditional access** flag across all three of its conditional access providers — Microsoft Entra, Google Cloud Identity, and Okta. The same flagged policy set drives every provider, so admins flag policies once and don't need to manage parallel lists per integration.

The mechanism each provider uses to enforce compliance differs:

- **Microsoft Entra** and **Google Cloud Identity** are API-push: Fleet writes the per-policy compliance result into the provider's device record so the provider's own conditional-access engine can evaluate it.
- **Okta** is cert-presentation: Fleet issues a per-device certificate that the host presents over mTLS during sign-in, and the Okta integration validates that cert against the same flagged-policy set before letting the device through.

## Step 8: Create a Context-Aware Access policy in Workspace

In [admin.google.com](https://admin.google.com), go to **Security > Access and data control > Context-Aware Access > Access levels**.

Create a new access level with a Custom expression. The CAA expression syntax is:

```cel
device.vendors["fleet-{C-id-without-C}"].is_compliant_device == true
```

Replace `{C-id-without-C}` with your customer ID **without** the leading `C`. For example, if your customer ID is `C0xxxxxxx`, the expression is:

```cel
device.vendors["fleet-0xxxxxxx"].is_compliant_device == true
```

> **Note on identifier ordering.** Google's REST API stores the partner segment in the format `{customer_id_without_C}-{suffix}`, but CAA expressions reference it as `{suffix}-{customer_id_without_C}`. The two orderings refer to the same record — Fleet writes the customer-first form when it PATCHes; CAA reads the suffix-first form. This is documented in Google's Access Context Manager spec.

Then attach that access level to the apps you want to gate.

## Available signals

Fleet writes the following fields, all of which are accessible from CAA expressions and visible to Workspace admins under **Devices > Mobile & endpoints > Endpoints > device > Third-party services > fleet (custom)**:

| Field | Values | CAA accessor |
| --- | --- | --- |
| Compliance state | `COMPLIANT` when every CA-flagged policy passes, else `NON_COMPLIANT` | `device.vendors["fleet-{C-id}"].is_compliant_device` |
| Managed state | `MANAGED` when MDM-enrolled in Fleet, else `UNMANAGED` | `device.vendors["fleet-{C-id}"].is_managed_device` |
| Health score | Graduated — see below | `device.vendors["fleet-{C-id}"].device_health_score` |
| Score reason | Human-readable summary including failing policy names | (admin-console only) |
| Custom ID | Fleet's `host.uuid` | (admin-console only) |
| Asset tags | `source:fleet`, `fleet_team_id:N` when assigned, `fleet_serial:SERIAL`, plus `label:NAME` for every Fleet label the host belongs to | `"label:engineering" in device.vendors["fleet-{C-id}"].asset_tags` |

### Health score mapping

Fleet maps the ratio of failing CA-flagged policies to one of Cloud Identity's five health score values:

| Failing ratio | Health score |
| --- | --- |
| 0% (all passing) | `VERY_GOOD` |
| ≤ 20% | `GOOD` |
| ≤ 50% | `NEUTRAL` |
| < 100% | `POOR` |
| 100% failing, or no CA-flagged policies configured | `VERY_POOR` |

The "no CA-flagged policies configured → `VERY_POOR`" convention applies when the integration is enabled for a team but no policies are flagged for conditional access — Fleet hasn't actually validated anything, so the device shouldn't render as healthy.

### Label asset tags

Every Fleet label the host is in is surfaced as a `label:NAME` entry in `asset_tags`. Admins can branch CAA expressions on team / region / role membership, e.g.:

```cel
device.vendors["fleet-0xxxxxxx"].is_compliant_device == true &&
"label:engineering" in device.vendors["fleet-0xxxxxxx"].asset_tags
```

Label name normalization:

- Names are lowercased and trimmed of leading/trailing whitespace.
- Internal whitespace runs collapse to a single dash. `"All Hosts"` becomes `label:all-hosts`; `"NYC2 - Engineering"` becomes `label:nyc2---engineering`.
- Empty names and names longer than 128 characters are dropped.
- Duplicates after normalization are deduplicated.
- A host with more than 50 labels has the alphabetically first 50 emitted; the rest are dropped. (Google does not document an `assetTags` limit; this cap keeps the PATCH body bounded.)

The built-in Fleet labels — `All Hosts`, `macOS`, `macOS 14`, etc. — are all surfaced. Custom team-scoped labels work the same way.

### Score reason examples

The `scoreReason` field is human-readable and shown to admins in admin.google.com:

- Compliant, single policy: `The 1 CA-flagged Fleet policy is passing.`
- Compliant, multiple policies: `All 5 CA-flagged Fleet policies are passing.`
- One of one failing: `1 of 1 CA-flagged Fleet policies are failing: Disk encryption`
- Multiple failing: `2 of 5 CA-flagged Fleet policies are failing: Disk encryption, Screen lock`

Failing policy names are sorted alphabetically for deterministic output. The field is capped at 1024 characters; long lists are truncated with an ellipsis.

## Latency

Fleet writes the ClientState every time policies finish evaluating on a host (typically once per osquery distributed-query cycle, every 1–10 minutes depending on team configuration). Once written, CAA evaluation is real-time at the next access attempt.

If a user needs to remediate immediately, they can click **Refetch** in the Fleet Desktop tray icon — this triggers an out-of-cycle policy run and a fresh PATCH.

The admin console's per-device device-detail view has a render cache that can lag a few minutes behind the actual PATCH. CAA expressions evaluate against the underlying data immediately; the cached render is only for human eyeballs.

## Troubleshooting

**`PERMISSION_DENIED` on the first sync.** Most common cause: your Workspace edition does not include Cloud Identity Premium security management. See "Supported editions" at the top of this article.

Other causes:

- DWD scope authorization is still propagating (can take up to 30 minutes after configuring Step 4).
- The `impersonated_admin` email is not a Workspace super-admin.

**No deviceUser resolved for a host.** Confirm Endpoint Verification is installed and the user has signed into Chrome at least once with their Workspace identity. The host detail page in Fleet will show "Endpoint Verification not installed" when EV has never run.

**ClientState written but not visible in admin console.** The device-detail page in admin.google.com has a render cache. Refresh the page after a few minutes; the signal will appear under "fleet (custom)".

**Signal not affecting CAA evaluation.** Check that the customer ID in the CAA expression matches your tenant. The order in the CAA expression is suffix-first (`device.vendors["fleet-0xxxxxxx"]`), even though Fleet writes the partner segment customer-first (`010vzyp5-fleet`). Both refer to the same record.

## Become a BeyondCorp Alliance partner (optional)

Fleet operates as a non-Alliance partner by default. This works out of the box — no Google relationship required, and every field Fleet writes is fully visible in the admin console and CAA expressions.

The only differences if Fleet later joins the BeyondCorp Alliance:

- Fleet appears in admin.google.com's third-party integrations picker (otherwise the partner identifier is typed by the customer in CAA expressions).
- The `(custom)` tag disappears from the device-detail view.
- CAA expressions can reference Fleet by its registered global name (e.g. `device.vendors["Fleet"]`) instead of the customer-ID-concatenated form.

Functional capabilities are identical in both modes.

<meta name="articleTitle" value="Conditional access: Google Cloud Identity">
<meta name="authorFullName" value="Robbie Trencheny">
<meta name="authorGitHubUsername" value="robbiet480">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-05-29">
<meta name="description" value="Learn how to enforce conditional access with Fleet and Google Cloud Identity Context-Aware Access.">
