# Zero-touch enrollment for company-owned Android devices

Status: research / design proposal. Not yet scheduled.

## Abstract

Fleet can already manage company-owned Android devices, but it cannot provision them without a human
touching each one. Today an admin must generate a short-lived enrollment token in Fleet and then scan a
QR code (or type `afw#setup`) on every device. Fleet has an equivalent zero-touch story on Apple —
Automated Device Enrollment via Apple Business — but nothing on Android. This document specifies what it
would take to close that gap using Google's Android zero-touch enrollment program, so that a device
purchased from a zero-touch reseller ships directly to an employee, and on first boot provisions itself
into the correct Fleet team as a fully managed device with no admin involvement.

The core technical insight is that zero-touch enrollment does not introduce a new enrollment protocol.
Google's zero-touch service simply pre-stages the same Android Management API (AMAPI) enrollment token
Fleet already creates, delivering it to the device through Google's provisioning servers instead of a QR
code. The device then follows the exact AMAPI provisioning path Fleet already supports, and Fleet learns
about it through the Pub/Sub `ENROLLMENT` notification it already handles. That makes this a
configuration-management and lifecycle feature, not a new MDM protocol implementation — the entire
existing profile delivery, software install, command, and host-vitals pipeline is reused unchanged.

The work therefore breaks into four parts. First, a new Google API integration: the **Android Device
Provisioning Partner API** (`androiddeviceprovisioning.googleapis.com`), customer surface, which is
already present in Fleet's vendored `google.golang.org/api` module and needs no new dependency. Second, a
credential model for that API, which is the single largest open question — Google offers a service-account
path (simple, automation-friendly, but requires a manual Google-approved linking request) and an
interactive OAuth path (self-service, but a per-admin refresh token and probable scope verification). Third,
Fleet-side changes: long-lived reusable enrollment tokens, a new table and config assets, a
configuration-per-team model, a reconciliation cron, pending-host records so admins can see devices before
first boot, activities, GitOps keys, REST endpoints, and UI. Fourth, a change to how Fleet identifies the
target team at enrollment time — the current design embeds a mutable enroll secret in the token's
`additionalData`, which silently breaks long-lived zero-touch tokens whenever an admin rotates a team's
enroll secret. That last item is a correctness prerequisite, not a nice-to-have.

The document also proposes a phased delivery in which Phase 1 ships real, usable zero-touch support with
**no** new Google API integration at all — Fleet generates the DPC extras JSON and the admin pastes it into
Google's zero-touch portal once per team. Phase 1 is small, unblocks customers immediately, and de-risks the
credential decision in Phase 2 by proving the token and team-mapping model in production first.

## Table of contents

- [Scope](#scope)
- [How Android zero-touch enrollment actually works](#how-android-zero-touch-enrollment-actually-works)
- [Where Fleet is today](#where-fleet-is-today)
- [Gap analysis](#gap-analysis)
- [Google APIs required](#google-apis-required)
- [Credential and authorization options](#credential-and-authorization-options)
- [Design](#design)
- [Fleet modifications](#fleet-modifications)
- [Phased delivery](#phased-delivery)
- [Edge cases and failure modes](#edge-cases-and-failure-modes)
- [Testing plan](#testing-plan)
- [Security considerations](#security-considerations)
- [Open questions](#open-questions)
- [Effort estimate](#effort-estimate)
- [References](#references)

## Scope

"Zero-touch enrollment for company-owned devices" means different things per platform. This document is
about the Android gap, because that is the only company-owned platform where Fleet has no zero-touch path
at all.

| Platform | Zero-touch mechanism | Fleet status |
| --- | --- | --- |
| macOS / iOS / iPadOS | Apple Automated Device Enrollment (ADE) via Apple Business | Supported. See `docs/Contributing/architecture/mdm/automated-device-enrollment.md` |
| Android (company-owned) | Google Android zero-touch enrollment | **Not supported — this document** |
| Windows | Windows Autopilot (Entra ID + Windows Autopilot service) | Not supported. Separate effort, unrelated APIs |
| ChromeOS | Chrome zero-touch enrollment (same provisioning API, `deviceType` CHROME_OS) | Out of scope; Fleet has no ChromeOS MDM |

Within Android, zero-touch applies to three company-owned management modes. All three are reachable
through the same mechanism, differing only in the `allowPersonalUsage` value on the enrollment token:

- **Fully managed (COBO)** — company-owned, business only. `PERSONAL_USAGE_DISALLOWED`.
- **Company-owned with a work profile (COPE)** — company-owned, personally enabled.
  `PERSONAL_USAGE_ALLOWED`.
- **Dedicated / kiosk (COSU)** — no user identity. `PERSONAL_USAGE_DISALLOWED_USERLESS`. Google made this
  value mandatory for dedicated-device enrollment as of January 2025.

Personally-owned (BYOD) work profile enrollment is explicitly **out of scope**: zero-touch requires the
device to be purchased through a zero-touch reseller and claimed to the organization, which by definition
makes it company-owned.

## How Android zero-touch enrollment actually works

Understanding the actor model matters, because it determines what Fleet can and cannot do.

### Actors

1. **Reseller** — an authorized zero-touch reseller partner. When an organization buys devices, the
   reseller registers each device's hardware identifiers (IMEI/MEID or serial number, plus manufacturer)
   into Google's zero-touch system and *claims* the device to that organization's customer account. **Only
   resellers can create device records.** IT admins cannot, and neither can an EMM. This is a hard
   constraint: Fleet can never add a device to zero-touch.
2. **Customer account** — the organization's zero-touch account, created by the reseller on first
   purchase. Admins manage it at `https://enterprise.google.com/android/zero-touch/customers`. Each
   customer account has an ID and an API resource name of the form `customers/{CUSTOMER_ID}`.
3. **DPC (Device Policy Controller)** — the agent that provisions and manages the device. For AMAPI-based
   EMMs like Fleet, the DPC is Google's own **Android Device Policy**
   (`com.google.android.apps.work.clouddpc`). Fleet does not ship a DPC.
4. **EMM (Fleet)** — provides a console where admins manage zero-touch configurations, using the
   customer API on the admin's behalf.

### Configurations

The unit of zero-touch policy is a **Configuration**, owned by the customer account. A configuration
bundles the DPC to install, the provisioning extras to hand that DPC, support contact info, and a custom
message shown during setup. Fields (all from `customers.configurations`):

| Field | Required | Notes |
| --- | --- | --- |
| `configurationName` | Yes | Shown to admins in the portal. Fleet would use the team name. |
| `dpcResourcePath` | Yes | `customers/{CUST_ID}/dpcs/{DPC_ID}`. Must be discovered via `customers.dpcs.list` and matched to Android Device Policy — do not hardcode. |
| `dpcExtras` | No | JSON string of provisioning extras. **This is where Fleet's enrollment token goes.** |
| `companyName` | Yes | Shown on device during provisioning. Fleet would default to org name from app config. |
| `contactEmail` | Yes | Validated by Google. Shown to the user before provisioning. |
| `contactPhone` | Yes | Shown to the user before provisioning. Digits, spaces, `+`, `-`, `()`. |
| `customMessage` | No | One or two sentences shown before provisioning. |
| `isDefault` | Yes | Applies to all newly purchased devices. **Only one default per customer account.** Setting a new default silently clears the old one. |
| `forcedResetTime` | No | Timeout before forcing a factory reset when setup cannot complete (typically no network). 0–6 hours, default 2 hours. |
| `name`, `configurationId` | Output only | Server-assigned. |

### Device records and configuration assignment

Devices under a customer account expose `deviceIdentifier` (serial number, manufacturer, model,
IMEI/MEID, plus `imei2`/`meid2` for dual-SIM), `deviceMetadata` (reseller-set key/value pairs),
`claims[]` (the zero-touch claim, `SECTION_TYPE_ZERO_TOUCH`), and `configuration` (the applied
configuration path, or empty).

A device gets a configuration one of two ways:

- Implicitly, by the customer account's **default configuration**, which applies to newly claimed devices.
- Explicitly, via `customers.devices:applyConfiguration`, which overrides the default for that device.

`customers.devices:removeConfiguration` unassigns; `customers.devices:unclaim` removes the device from
zero-touch entirely (destructive — the organization loses the claim and must go back to the reseller).

### First-boot flow

1. Device is factory reset or unboxed and connects to a network during setup.
2. It checks in with Google's provisioning servers, identifying itself by hardware ID.
3. If a configuration is assigned, Google returns it and the device enters the fully managed device setup
   wizard, showing `companyName`, `customMessage`, `contactEmail`, and `contactPhone`.
4. The device downloads Android Device Policy from Google Play.
5. Android Device Policy receives `ACTION_PROVISION_MANAGED_DEVICE` with the extras from `dpcExtras`,
   reads the AMAPI enrollment token, and enrolls into the AMAPI enterprise.
6. AMAPI publishes an `ENROLLMENT` notification to the enterprise's Pub/Sub topic. **This is where Fleet
   re-enters the picture, on the code path it already has.**

### When provisioning triggers — and the destructive edge

Zero-touch is evaluated **only during the out-of-box setup wizard**: on first boot, or on the next factory
reset. It is not a push mechanism and cannot provision a device that is already set up and running.
Three consequences matter for Fleet's design:

- **It re-triggers on every factory reset, indefinitely.** A device claimed to a customer account and
  assigned a configuration re-provisions every time it is reset, for as long as the claim exists. The end
  user cannot bypass or defer it. This stickiness is the security property that makes zero-touch valuable
  for company-owned hardware — and the reason a broken configuration is a fleet-wide outage rather than a
  per-device annoyance.
- **Assigning a configuration to a device that is already in use causes a factory reset.** Per Google's
  admin documentation, a device that lacks connectivity during setup skips the zero-touch flow and boots
  unmanaged, but then "resets itself after the first connection to Google servers," giving the user one
  hour of warning. The same applies to a device already in service when a configuration is applied to it.
  **`customers.devices:applyConfiguration` is therefore potentially destructive of end-user data**, which
  is not obvious from the API surface — the method name suggests a metadata write. Fleet must treat it as
  a destructive action in the UI, with explicit warning and confirmation, and the same warning belongs on
  the "set default configuration" path, since the default applies to every newly claimed device.
- **Two distinct reset mechanisms exist**, and they are easy to conflate: `forcedResetTime` (0–6 hours,
  default 2) is the in-setup timeout when the wizard cannot reach Google, while the self-reset above fires
  after setup has completed unmanaged. Both end in a factory reset; only the first is configurable.

### DPC extras for AMAPI

The exact `dpcExtras` payload for an AMAPI-managed device:

```json
{
  "android.app.extra.PROVISIONING_DEVICE_ADMIN_COMPONENT_NAME":
    "com.google.android.apps.work.clouddpc/.receivers.CloudDeviceAdminReceiver",
  "android.app.extra.PROVISIONING_DEVICE_ADMIN_SIGNATURE_CHECKSUM":
    "I5YvS0O5hXY46mb01BlRjq4oJJGs2kuUcHvVkAPEXlg",
  "android.app.extra.PROVISIONING_ADMIN_EXTRAS_BUNDLE": {
    "com.google.android.apps.work.clouddpc.EXTRA_ENROLLMENT_TOKEN": "<AMAPI enrollment token value>"
  }
}
```

Google's EMM guidance on extras, which Fleet should follow:

- Put everything Fleet-specific inside `PROVISIONING_ADMIN_EXTRAS_BUNDLE`, not as top-level extras.
- Do **not** put account credentials or secrets in the configuration. The AMAPI enrollment token is the
  one unavoidable exception, and it is scoped to enrollment only.
- Optionally supported and reasonable for Fleet to expose later:
  `EXTRA_PROVISIONING_LOCALE`, `EXTRA_PROVISIONING_TIME_ZONE`, `EXTRA_PROVISIONING_LOCAL_TIME`,
  `EXTRA_PROVISIONING_LEAVE_ALL_SYSTEM_APPS_ENABLED`, `EXTRA_PROVISIONING_MAIN_COLOR`,
  `EXTRA_PROVISIONING_DISCLAIMERS`.
- Google explicitly advises against `PROVISIONING_DEVICE_ADMIN_PACKAGE_DOWNLOAD_LOCATION`,
  `..._PACKAGE_CHECKSUM`, and cookie-header extras — those belong to other enrollment methods.

### Device eligibility

- Android 8.0+ (or Pixel on 7.1+) per the zero-touch EMM guide. Google's Workspace-facing documentation
  states 9.0+ (Pixel 7.0+) for work-profile devices, so the floor varies by source and management mode —
  verify against the Android Enterprise device list rather than either number.
- Google Mobile Services present.
- Registered by a participating zero-touch reseller and claimed to the organization's customer account.

### Devices the organization already owns

This is the most consequential limitation for adoption, and it should be stated plainly in Fleet's UI and
docs rather than discovered.

An organization's **existing** Android fleet generally cannot be moved to zero-touch. Registration is
performed by the reseller that sold the device, and there is no self-service path — Google's newer
"automatic zero-touch enrollment" framing in the Workspace documentation still requires reseller
registration. A device bought at retail or through a non-participating channel has no route in at all.
Devices purchased from a participating reseller can often be registered retroactively on request, since
the reseller has both the tooling (`partners.devices.claim`) and the sales record, but that is the
reseller's call and outside Fleet's control. Deregistering a device likewise requires going back to the
reseller to reverse.

The practical consequence: zero-touch is a **forward-looking** capability tied to procurement, not a
migration path for an installed base. For devices already owned, Fleet's existing QR-code and
`afw#setup` flows already achieve the same *management* end state — a fully managed device in the correct
team — at the cost of one human touch per device at factory-reset time.

What that does **not** replicate is zero-touch's stickiness. A QR-enrolled device that is factory reset
boots unmanaged; a zero-touch device re-provisions itself. Where the requirement is tamper resistance
rather than provisioning convenience, only reseller registration satisfies it. Fleet's messaging should
distinguish these two value propositions instead of presenting zero-touch purely as a convenience
feature, because they lead to different purchasing decisions.

Platform-specific alternative, noted for completeness and **not** proposed here: Samsung Knox Mobile
Enrollment supports admin self-registration of already-owned devices via the Knox Deployment App or a KME
QR profile, without reseller involvement, and can target AMAPI. It is a separate program from Google
zero-touch and would be a distinct integration effort.

## Where Fleet is today

Fleet's Android MDM lives in `server/mdm/android/` as a deliberately decoupled bounded context (see
`server/mdm/android/README.md`).

### Enrollment token creation

`Service.CreateEnrollmentToken` in `server/mdm/android/service/service.go:549` is the entire current
enrollment surface. It:

1. Verifies Android MDM is configured.
2. Verifies the supplied enroll secret (`svc.ds.VerifyEnrollSecret`), which is what determines the target
   team.
3. Optionally requires and validates an IdP account UUID for end-user authentication.
4. Marshals `{enroll_secret, idp_uuid}` into the token's `additionalData`.
5. Sets `allowPersonalUsage` from a `fully_managed` query parameter — `PERSONAL_USAGE_DISALLOWED` when
   true, `PERSONAL_USAGE_ALLOWED` otherwise.
6. Sets `oneTimeOnly: true` and leaves `duration` unset, so **the token expires in one hour and works
   exactly once**.
7. Returns the token value, a `https://enterprise.google.com/android/enroll?et=<token>` URL, and a QR code.

The frontend surface is `frontend/components/AddHostsModal/PlatformWrapper/AndroidPanel/AndroidPanel.tsx`,
which offers a "work profile" / "fully managed" radio and renders the QR code.

### Enrollment completion

`Service.enrollHost` (`server/mdm/android/service/pubsub.go:599`) handles the Pub/Sub `ENROLLMENT`
notification. It unmarshals `device.EnrollmentTokenData` back into `{enroll_secret, idp_uuid}`, verifies
the enroll secret again, resolves the team, and creates or updates the host. `addNewHost`
(`pubsub.go:893`) is the new-device path; it maps AMAPI `hardwareInfo.serialNumber` into
`host.HardwareSerial` (`pubsub.go:947`), which matters for correlating zero-touch device records later.

### AMAPI client

`androidmgmt.Client` (`server/mdm/android/service/androidmgmt/client.go`) is an interface with two
implementations:

- `ProxyClient` — the default. Routes AMAPI calls through `https://fleetdm.com/api/android/`, using a
  per-enterprise `FleetServerSecret` as a bearer token. Fleet's hosted proxy owns the Google Cloud
  project, the service credentials, and the Pub/Sub topic.
- `GoogleClient` — development only, gated behind `FLEET_DEV_ANDROID_GOOGLE_CLIENT=1` and
  `FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS`. Talks to Google directly with a service account.

This proxy architecture is the most important existing constraint on the design, and is discussed in
detail below.

### Storage

`android_enterprises` holds one row: `enterprise_id`, `signup_name`, `signup_token`, `pubsub_topic_id`,
`user_id`. Secrets live in `mdm_config_assets` under names defined in `server/fleet/mdm.go` —
`android_pubsub_token`, `android_fleet_server_secret`. There is no per-team Android enrollment state
anywhere.

## Gap analysis

| Capability | Needed for zero-touch | Fleet today |
| --- | --- | --- |
| Long-lived, reusable enrollment token | Yes — the token sits in a configuration for months and is used by many devices | 1 hour, `oneTimeOnly: true` |
| Stable team identification in the token | Yes — the token outlives enroll secret rotations | Embeds the mutable enroll secret |
| Zero-touch customer API client | Yes | Does not exist |
| Credential storage for that API | Yes | Does not exist |
| Configuration-per-team model | Yes | Does not exist |
| Reconciliation of Google-side state | Yes — configs drift, admins edit in the portal | Does not exist |
| Pending/awaiting-enrollment host visibility | Desirable, matches ADE | Does not exist for Android |
| Per-device configuration override | Yes — how a device lands in a non-default team | Does not exist |
| Activities / audit trail | Yes | Does not exist |
| GitOps | Yes, for parity | Does not exist |
| `PERSONAL_USAGE_DISALLOWED_USERLESS` (dedicated devices) | For COSU | Not exposed |
| End-user IdP auth during zero-touch provisioning | Depends on requirements | BYOD-cookie-based only; not reachable from zero-touch |

## Google APIs required

### 1. Android Device Provisioning Partner API — customer surface (new)

- **Service**: `androiddeviceprovisioning.googleapis.com`
- **Go package**: `google.golang.org/api/androiddeviceprovisioning/v1` — **already present in Fleet's
  module graph** at `google.golang.org/api v0.269.0` (`go.mod:181`). No new dependency. The generated
  package exposes `CustomersService`, `CustomersConfigurationsService`, `CustomersDevicesService`, and
  `CustomersDpcsService`.
- **OAuth scope**: `https://www.googleapis.com/auth/androidworkzerotouchemm`. Note this is *different*
  from the reseller scope `https://www.googleapis.com/auth/androidworkprovisioning`. The generated Go
  package declares no default scopes, so it must be passed explicitly via `option.WithScopes(...)`.
- **Enablement**: the API must be enabled on the Google Cloud project whose credential is used.
- **Recommended header**: `X-GOOG-API-FORMAT-VERSION: 2`, which Google requires to get structured error
  details — specifically `TosError`, needed to detect the terms-of-service case below.

Methods Fleet needs:

| Method | Use in Fleet |
| --- | --- |
| `customers.list` | Discover the customer account(s) the credential can act on. Returns `customers/{id}`. |
| `customers.dpcs.list` | Find the Android Device Policy resource path for `dpcResourcePath`. Match on the last path component; never hardcode. |
| `customers.configurations.list` | Reconcile Fleet's known configurations against Google's, detect out-of-band edits and deletions. |
| `customers.configurations.create` | Create a configuration per Fleet team. |
| `customers.configurations.patch` | Update `dpcExtras` on token rotation, and metadata on team rename. |
| `customers.configurations.delete` | Remove a configuration when a team is deleted or zero-touch is disabled for it. |
| `customers.devices.list` | Enumerate claimed devices to build pending-host records. Paginated via `nextPageToken`. |
| `customers.devices.get` | Refresh a single device. |
| `customers.devices:applyConfiguration` | Assign a specific device to a specific team's configuration. |
| `customers.devices:removeConfiguration` | Unassign, falling back to the default configuration. |
| `customers.devices:unclaim` | Destructive; expose only behind an explicit confirmation, if at all. |

Not needed: the entire `partners.*` reseller surface. Fleet is not a reseller and should not attempt to
become one — device claiming is a supply-chain function.

**Terms of service handling**: calls return `403 Forbidden` with a `TosError` body until the acting user
has accepted the current zero-touch terms. Fleet must detect this specific case and surface an actionable
message linking to `https://enterprise.google.com/android/zero-touch/customers`, rather than a generic
403. Google can update the terms at any time, so this is not a one-time setup error — it can appear
mid-life on a working integration and must be handled in the cron, not only during setup.

### 2. Android Management API (existing, extended)

No new API, but new usage of it:

- `enterprises.enrollmentTokens.create` with:
  - `duration` set explicitly. The field accepts 1 minute up to roughly 10,000 years; default is 1 hour.
    If the requested duration would overflow the maximum timestamp, Google coerces it.
  - `oneTimeOnly: false`, so many devices can enroll with the same token.
  - `allowPersonalUsage` set per management mode (`PERSONAL_USAGE_DISALLOWED`, `PERSONAL_USAGE_ALLOWED`,
    or `PERSONAL_USAGE_DISALLOWED_USERLESS`).
  - `additionalData` — max 1024 characters, surfaced on the device as `enrollmentTokenData`. This is
    Fleet's only channel for carrying team intent through provisioning, and the 1024-character ceiling is
    a real design constraint.
  - `policyName` — Fleet already points this at `{enterprise}/policies/1`.
- `enterprises.enrollmentTokens.delete` — needed to revoke a leaked or superseded zero-touch token.
- `enterprises.enrollmentTokens.list` — lists active, unexpired tokens; useful for drift detection and
  cleanup.
- `EnrollmentToken.user` is deprecated and ignored; do not use it.

### 3. Google Cloud Pub/Sub (existing)

Unchanged. Zero-touch devices produce the same `ENROLLMENT` notification as QR-code devices. This is
precisely why the feature is tractable.

### 4. Optionally: AMAPI `signinDetail` (only if end-user IdP auth is required on zero-touch)

Zero-touch hands the device a plain enrollment token with no user interaction, so a Fleet-mediated IdP
login cannot happen the way it does for BYOD (which relies on a browser cookie set before the enrollment
token is requested — `mdm.BYODIdpCookieName`, `service.go:508`).

If end-user authentication during zero-touch provisioning is a requirement, the mechanism is AMAPI's
sign-in URL enrollment. An enterprise can have any number of `SigninDetail` entries, uniquely keyed by
(`signinUrl`, `allowPersonalUsage`, `tokenTag`). Each yields a server-generated, read-only
`signinEnrollmentToken`. Placing that token in `dpcExtras` instead of a normal enrollment token causes
the device to open `signinUrl` during provisioning; Fleet's page runs the IdP flow and must finish by
redirecting to `https://enterprise.google.com/android/enroll?et=<token>` on success or
`https://enterprise.google.com/android/enroll/invalid` on failure.

This is a substantial sub-project — it means Fleet serves an unauthenticated, device-facing web flow that
mints enrollment tokens. **Recommendation: exclude it from the initial scope.** Company-owned zero-touch
devices are typically assigned to a person out-of-band, and the AMAPI `provisioningInfo` /
`authenticatedUserEmail` mechanism plus Fleet's existing per-team assignment covers most needs. Revisit
only if customers ask for user-attributed zero-touch specifically.

## Credential and authorization options

This is the decision that gates everything else, and it deserves an explicit decision before
implementation starts.

### Option A — Service account key uploaded by the admin (recommended)

The admin creates a Google Cloud project, enables the Android Device Provisioning Partner API, creates a
service account, downloads its JSON key, and uploads that key to Fleet. Fleet uses two-legged OAuth with
scope `androidworkzerotouchemm`.

- **Pros**: no refresh-token lifecycle, no consent UI, no dependency on an individual admin's Google
  account, and Google explicitly recommends service accounts for "continuously-running automated
  services" — which is exactly what Fleet's reconciliation cron is. It also mirrors Fleet's existing ABM
  token UX almost exactly: do work in the vendor portal, download a credential, upload it to Fleet.
- **Cons**: the service account must be linked to the zero-touch customer account, and per Google's
  current documentation that linking is **a Google Form submission followed by an email confirmation**,
  not a self-service portal action. That is human latency of unknown duration on the customer's critical
  path, and it is opaque to Fleet.
- **Verify before committing**: whether a self-service linking path now exists in the customer portal
  (the reseller portal does have a "Service accounts → Link service account" screen; the customer portal
  may have gained parity), and whether there is a per-organization limit on linked service accounts.

### Option B — Interactive OAuth (three-legged) by the admin

Fleet registers an OAuth client, the admin consents with the Google account their reseller established in
the zero-touch account, and Fleet stores the refresh token. This is what Google's EMM integration guide
describes as the standard EMM console pattern.

- **Pros**: fully self-service, no Google Form, works the moment the admin clicks through consent.
  Architecturally consistent with Fleet's existing AMAPI signup, which already round-trips through
  `fleetdm.com` with a callback URL.
- **Cons**: the grant is bound to one human's Google account, so it breaks when that person's access is
  removed — a recurring support burden. `androidworkzerotouchemm` is likely to be treated as a sensitive
  scope requiring Google app verification, which is a schedule risk. Refresh tokens can be revoked
  server-side without notice, so Fleet needs re-consent detection and prompting. Self-hosted Fleet needs
  either its own OAuth client (admin burden) or a proxy-mediated redirect URI.

### Option C — No API at all: Fleet emits the DPC extras, admin pastes them

Fleet generates a long-lived reusable enrollment token per team and displays the complete `dpcExtras`
JSON with a copy button plus step-by-step portal instructions. The admin creates the configuration by
hand in Google's portal.

- **Pros**: zero Google API integration, zero credential storage, zero authorization risk. Works today
  for every customer including self-hosted and air-gapped-adjacent deployments. Small enough to ship in
  one release.
- **Cons**: not "zero-touch admin" — the admin does per-team manual portal work. No pending-host
  visibility, no per-device team override, no drift detection.

### Recommendation

Ship **Option C as Phase 1**, then **Option A as Phase 2** with **Option B as an alternate path** for
customers blocked on service-account linking. Option C is not a throwaway: it forces Fleet to get the
long-lived token model, the team-mapping fix, and the extras generation right, all of which Phase 2
reuses verbatim. It also means the credential decision does not block customer value.

### The Fleet Cloud proxy question

Fleet's AMAPI traffic defaults through `https://fleetdm.com/api/android/`, with Fleet's hosted project
owning the Google credentials. The zero-touch customer API **cannot** follow that model in the same way,
and this needs to be explicit:

- The zero-touch customer account belongs to the customer, established by *their* reseller. Fleet's
  hosted service account has no relationship to it and cannot be granted one without the customer
  linking it.
- If Fleet linked one shared service account to many customer accounts, `customers.list` would return
  every linked customer to every caller. Fleet would then be responsible for enforcing tenant isolation
  in the proxy, on a credential with no per-tenant scoping. **This is a multi-tenancy hazard and should
  be rejected.**

Therefore: zero-touch credentials are **per-Fleet-instance and customer-supplied**, and zero-touch API
calls go **direct to Google**, not through the proxy — regardless of how AMAPI traffic is routed. This is
a deliberate architectural divergence from the rest of Android MDM and should be documented as such,
because it will surprise people reading the code.

## Design

### Configuration per team

Fleet teams are the natural unit. Each Fleet team that opts into zero-touch gets:

- One long-lived AMAPI enrollment token, carrying that team's identity in `additionalData` and the
  team's chosen management mode in `allowPersonalUsage`.
- One zero-touch Configuration whose `dpcExtras` embeds that token and whose `configurationName` is
  derived from the team name.

Because Google permits exactly one default configuration per customer account, Fleet exposes a single
**"default team for zero-touch"** setting. That team's configuration carries `isDefault: true`; all newly
purchased devices land in it. Any other team's devices are placed via
`customers.devices:applyConfiguration`. Changing the default team means patching two configurations, and
Fleet must handle Google's implicit clearing of the previous default rather than assuming its own view is
authoritative.

A distinct configuration is needed per (team, management mode) pair, since `allowPersonalUsage` is fixed
at token creation. Keeping mode as a per-team setting rather than a per-configuration one keeps the model
comprehensible; teams that need both fully managed and dedicated devices should be separate teams.

### Fixing team identification: the critical prerequisite

Today `additionalData` carries `{"enroll_secret": "...", "idp_uuid": "..."}` and `addNewHost`
(`pubsub.go:908`) hard-fails when `VerifyEnrollSecret` does not match:

```go
enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, enrollmentTokenRequest.EnrollSecret)
if err != nil {
    return ctxerr.Wrap(ctx, err, "verifying enroll secret")
}
```

For a one-hour token this is fine. For a token embedded in a zero-touch configuration for a year, it is a
latent outage: **the moment an admin rotates that team's enroll secret, every subsequent zero-touch
device enrolls successfully into AMAPI and then fails to become a Fleet host.** The device is managed by
Google, invisible in Fleet, and the failure surfaces only as a repeating Pub/Sub error. This is
exactly the kind of silent, delayed breakage that is worst to debug in the field.

Two viable fixes:

1. **Carry a stable team reference.** Extend the `additionalData` payload with a version tag and a
   durable identifier, for example
   `{"v":2,"zt":true,"team_id":3,"enroll_secret":"..."}`, and in `enrollHost`/`addNewHost` prefer
   `team_id` when `zt` is set, treating an enroll-secret mismatch as non-fatal in that case. A team UUID
   would be preferable to a numeric ID for stability across restores, if one is available. Note the 1024
   character `additionalData` limit — it is generous here but must be validated, not assumed.
2. **Re-issue on rotation.** Hook enroll-secret changes and patch every affected zero-touch
   configuration with a freshly minted token.

**Do both.** Fix 1 makes the system correct under drift; fix 2 keeps the stored token consistent with
Fleet's state. Fix 1 alone leaves a stale secret in Google's configuration; fix 2 alone leaves a window
where in-flight provisioning breaks, and fails entirely if the patch call errors.

Whichever payload shape is chosen, it must be versioned, because tokens already in the wild will carry
the v1 shape and must keep working. `getAndroidHostKey` / `GetAndroidDeviceLastTeamID` already override
the token's team for previously-known devices (`pubsub.go:916`), and that precedence must be preserved:
last-known team still wins over the zero-touch token's team, so a factory-reset device returns to where
the admin put it.

### Token lifetime and rotation

Fleet should create zero-touch tokens with an explicit long duration and rotate on demand rather than on
a short clock. Rationale: a mid-provisioning token swap is the highest-risk moment in the whole flow, and
frequent rotation multiplies exposure to that risk for little security benefit — the token's only power
is to enroll a device into the enterprise, and it lives in a store readable only by zero-touch portal
admins.

Concretely:

- Create with `duration` of roughly one year and `oneTimeOnly: false`.
- Store `expires_at` and have the reconciliation cron rotate when within, say, 30 days of expiry.
- On rotation: mint the new token, patch `dpcExtras`, confirm the patch succeeded, and only then delete
  the old token — never the reverse. Keep a grace period before deletion so devices that already
  fetched the old configuration can still complete.
- Expose an explicit admin "rotate zero-touch token" action for the leak case, which deletes the old
  token immediately and accepts the in-flight breakage.

### Pending hosts

`customers.devices.list` returns hardware identifiers before any device has ever booted, and AMAPI
reports `hardwareInfo.serialNumber` at enrollment, which Fleet already stores as `HardwareSerial`
(`pubsub.go:947`). That makes correlation possible and enables an Android analogue of Apple's
`host_dep_assignments` "awaiting enrollment" experience: admins see the devices they bought, in the team
they will land in, before anyone opens a box.

Caveats to design around:

- Serial number is the practical join key, but some devices report IMEI/MEID only, and dual-SIM devices
  have `imei2`/`meid2`. The correlation must try serial first, then IMEI/MEID, and tolerate no match.
- AMAPI's `hardwareInfo.serialNumber` is documented as unavailable for personally-owned devices; for
  company-owned zero-touch devices it should be present, but the code must not assume it.
- Formatting differences (case, leading zeros, whitespace) are a realistic source of missed matches;
  normalize on both sides. `fleet.Preprocess` exists for this class of problem.
- Getting this wrong creates duplicate host records, which is worse than having no pending hosts at all.
  If correlation confidence is low, ship Phase 2 without pending hosts and add them once real device data
  is available.

### Ownership, portability, and offboarding

Two separate registrations exist, owned by different parties, and conflating them leads to wrong
conclusions about lock-in.

**The zero-touch device claim belongs to the customer.** It lives in *their* zero-touch customer account
(`customers/{CUSTOMER_ID}`), created by *their* reseller against *their* organization. Fleet is not a
party to the claim and does not appear in it. Fleet's service account or OAuth grant is merely a
delegated caller that the customer can revoke at any time without affecting the claim.

**The AMAPI enterprise is bound to Fleet.** `EnterprisesCreate` passes `ProjectId(g.androidProjectID)`
(`androidmgmt/google_client.go:92`), so the enterprise is created under a Google Cloud project — and in
the default proxy deployment, that project is Fleet's, not the customer's. This produces an *EMM-managed*
enterprise, which cannot be transferred to another EMM's project. Note the asymmetry: a deployment using
`GoogleClient` with the customer's own credentials would create the enterprise under the customer's
project instead. That path is currently development-only, but it is the lever if enterprise ownership ever
becomes a customer requirement.

**The only coupling between the two systems is the enrollment token string inside `dpcExtras`.** That is a
mutable field in a record the customer owns. This loose coupling is the important architectural property,
and it is worth preserving deliberately rather than by accident.

Consequences for migrating away from Fleet:

- **The zero-touch side migrates cleanly.** The customer's portal admins replace the token in `dpcExtras`
  with the incoming EMM's, or point the device at a new configuration. Devices stay claimed, no reseller
  involvement, no re-registration. Critically, they can do this **without Fleet's cooperation** — even if
  the Fleet instance is decommissioned or the contract has lapsed. Fleet holds no hostage here, and that
  should be stated plainly in customer-facing material rather than left ambiguous.
- **The AMAPI side requires a factory reset per device.** Device Owner cannot be transferred over the air
  between EMM enterprises. AMAPI's DPC migration facility is custom-DPC-to-Android-Device-Policy *within
  one enterprise*, not cross-EMM. This is a universal Android Enterprise constraint, not a Fleet
  limitation, and applies identically in the inbound direction for customers migrating *to* Fleet.
- **Zero-touch makes migration easier, not harder.** The standard playbook is: repoint the configuration
  at the new EMM, then factory reset in batches; each device self-provisions into the new EMM with no
  hands-on step. Zero-touch is the migration mechanism.

**Offboarding hazard — ordering matters, and the current turn-off flow is unsafe.** `enterprises.delete`
performs "a cascaded deletion of all AM API devices associated with the deleted enterprise," and
`Service.DeleteEnterprise` (`service.go:424`, calling `EnterpriseDelete` at `service.go:442`) is what runs
when an admin turns off Android MDM. If zero-touch configurations exist, deleting the enterprise
invalidates the enrollment token still embedded in them. Every subsequent factory reset then hits a
configuration whose token is dead, and the device cannot complete provisioning — ending in the forced
reset behavior described earlier. In other words, **turning off Android MDM in Fleet silently breaks
provisioning for every zero-touch-claimed device, with the damage surfacing only on each device's next
reset.**

Required mitigations:

- Turning off Android MDM must detect existing zero-touch configurations and block or hard-warn, listing
  the affected teams and device count. This should be treated as a release blocker for Phase 2, not a
  polish item.
- Offer a "prepare for migration" action that deletes Fleet's configurations from the customer account
  (leaving claims intact) before enterprise deletion, so devices boot unmanaged rather than looping.
- Document the correct offboarding order: repoint or delete zero-touch configurations → reset devices →
  then delete the enterprise in Fleet.
- Verify the on-device end state after a cascaded enterprise deletion — whether devices become unmanaged,
  remain orphaned but provisioned, or require a reset. This is not documented clearly by Google and needs
  hands-on testing before the offboarding guidance can be considered correct.

### Reconciliation cron

A new cron, following `newAndroidMDMDeviceReconcilerSchedule` (`cmd/fleet/cron.go:2513`) as the pattern
and registered in `cmd/fleet/cron_registration.go`:

1. Verify the credential still works; mark the integration invalid and stop if not.
2. `customers.list` — confirm the expected customer account is still reachable.
3. `customers.dpcs.list` — refresh the Android Device Policy resource path.
4. `customers.configurations.list` — diff against Fleet's rows. Recreate configurations deleted out of
   band; re-patch drifted `dpcExtras`; reconcile which configuration Google considers default.
5. Rotate tokens approaching expiry.
6. `customers.devices.list` (paginated) — upsert pending device records, reconcile per-device
   configuration assignments against team intent.
7. Detect and surface `TosError`.

Reasonable interval: 30 minutes, tunable. It must be bounded and paginated — the existing Android device
reconciler had to have its pagination loop bounded (commit `8a65ecf20bf`), and the same failure mode
applies here. Google's per-project quotas for the provisioning API are undocumented in the pages
reviewed and should be confirmed before choosing an interval for large fleets.

## Fleet modifications

### New package

`server/mdm/android/zerotouch/` (or `server/mdm/android/service/androidprov/`, mirroring `androidmgmt/`)
containing an interface-first client so it can be mocked, exactly as `androidmgmt.Client` is:

```go
// Client is used to interact with the zero-touch enrollment customer API.
// See: https://developers.google.com/zero-touch/reference/customer/rest
type Client interface {
	CustomersList(ctx context.Context) ([]*androiddeviceprovisioning.Company, error)
	DPCsList(ctx context.Context, customerName string) ([]*androiddeviceprovisioning.Dpc, error)

	ConfigurationsList(ctx context.Context, customerName string) ([]*androiddeviceprovisioning.Configuration, error)
	ConfigurationCreate(ctx context.Context, customerName string, cfg *androiddeviceprovisioning.Configuration) (*androiddeviceprovisioning.Configuration, error)
	ConfigurationPatch(ctx context.Context, configName string, cfg *androiddeviceprovisioning.Configuration, updateMask string) (*androiddeviceprovisioning.Configuration, error)
	ConfigurationDelete(ctx context.Context, configName string) error

	DevicesList(ctx context.Context, customerName, pageToken string) (*androiddeviceprovisioning.CustomerListDevicesResponse, error)
	DeviceApplyConfiguration(ctx context.Context, customerName, deviceName, configName string) error
	DeviceRemoveConfiguration(ctx context.Context, customerName, deviceName string) error
}
```

Plus a `mock/` package generated in the same style as `server/mdm/android/mock/client.go`, and an
`IsTermsOfServiceError(err error) bool` helper alongside the existing `IsNotModifiedError` /
`IsBadRequestError` in `androidmgmt/client.go`.

The `arch_test.go` in `server/mdm/android/` enforces the bounded-context import rules; new code must
satisfy it.

### Database

One new table plus a couple of columns. Timestamps at `TIMESTAMP(6)` precision and binary UUIDs per
Fleet's MySQL conventions.

```sql
CREATE TABLE `android_zero_touch_configurations` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `team_id` INT UNSIGNED DEFAULT NULL,
  `global_or_team_id` INT NOT NULL DEFAULT '0',
  -- Google-side identity
  `customer_id` VARCHAR(64) NOT NULL,
  `configuration_id` VARCHAR(64) NOT NULL DEFAULT '',
  `configuration_name` VARCHAR(255) NOT NULL DEFAULT '',
  `is_default` TINYINT(1) NOT NULL DEFAULT '0',
  -- Fleet-side intent
  `management_mode` VARCHAR(40) NOT NULL,  -- allowPersonalUsage value
  `company_name` VARCHAR(255) NOT NULL DEFAULT '',
  `contact_email` VARCHAR(255) NOT NULL DEFAULT '',
  `contact_phone` VARCHAR(64) NOT NULL DEFAULT '',
  `custom_message` TEXT,
  -- Token state (token value itself lives in mdm_config_assets, not here)
  `enrollment_token_name` VARCHAR(255) NOT NULL DEFAULT '',
  `enrollment_token_expires_at` TIMESTAMP(6) NULL DEFAULT NULL,
  `last_synced_at` TIMESTAMP(6) NULL DEFAULT NULL,
  `sync_error` TEXT,
  `created_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_android_zt_global_or_team_id` (`global_or_team_id`),
  KEY `fk_android_zt_team_id` (`team_id`),
  CONSTRAINT `fk_android_zt_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

The `global_or_team_id` / `team_id` pair with a `UNIQUE KEY` on the former follows
`android_app_configurations` and gives correct "No team" handling — worth stating explicitly, since "No
team" is a routine source of bugs in team-scoped features.

For pending hosts, either a second table or reuse of the host pattern:

```sql
CREATE TABLE `android_zero_touch_devices` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `google_device_id` VARCHAR(64) NOT NULL,
  `serial_number` VARCHAR(255) NOT NULL DEFAULT '',
  `imei` VARCHAR(32) NOT NULL DEFAULT '',
  `meid` VARCHAR(32) NOT NULL DEFAULT '',
  `manufacturer` VARCHAR(255) NOT NULL DEFAULT '',
  `model` VARCHAR(255) NOT NULL DEFAULT '',
  `configuration_id` INT UNSIGNED DEFAULT NULL,
  `host_id` INT UNSIGNED DEFAULT NULL,  -- set once correlated post-enrollment
  `deleted_at` TIMESTAMP(6) NULL DEFAULT NULL,
  `created_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_android_zt_devices_google_device_id` (`google_device_id`),
  KEY `idx_android_zt_devices_serial` (`serial_number`),
  KEY `idx_android_zt_devices_host_id` (`host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

Use `make migration name=AddAndroidZeroTouchConfigurations`. If this lands alongside other work, the
migration timestamp will likely need bumping before merge.

### Secrets

New `MDMAssetName` values in `server/fleet/mdm.go`, stored encrypted in `mdm_config_assets`:

- `android_zero_touch_service_account` — the uploaded service-account JSON key (Option A).
- `android_zero_touch_oauth_refresh_token` — refresh token (Option B).
- `android_zero_touch_enrollment_token` — the current long-lived token value per configuration. Storing
  it as an asset rather than a table column keeps it out of ordinary query paths and gets encryption for
  free. Since assets are keyed by name, this needs either a name-per-team scheme or an asset table
  variant that supports scoping — worth checking `mdm_config_assets` for existing per-entity precedent
  (the VPP and ABM token tables solved this differently) before choosing.

### Service layer and API

New endpoints, registered in `server/mdm/android/service/handler.go` following the existing pattern, with
request/response structs and `Err error` fields per Fleet's API conventions:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch` | Integration status: linked customer, credential validity, ToS state, last sync, errors |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/credentials` | Upload service-account key (or begin OAuth) |
| `DELETE` | `/api/_version_/fleet/android_enterprise/zero_touch/credentials` | Disconnect |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations` | List Fleet's configurations |
| `PUT` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id` | Create or update a team's configuration |
| `DELETE` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id` | Remove |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id/rotate_token` | Force rotation |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/dpc_extras/:team_id` | Phase 1: return the copy-paste JSON |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/devices` | Paginated device list with team and enrollment status |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/devices/:id/team` | Move a device to a team |

Conventions to hold to: authorize first in the service method, validate inputs in the service method (not
the endpoint) and accumulate errors into a single `InvalidArgumentError`, wrap all errors with
`ctxerr.Wrap`, and return client errors as 4xx without logging them as server errors. Google-side
validation failures — a rejected `contactEmail`, a malformed phone number — are client errors and must
not surface as 500s. Since `contactEmail` and `contactPhone` are validated by Google, Fleet should
validate them locally too so the admin gets a field-level error instead of a round-trip failure.

Zero-touch is a company-owned-fleet management feature and should be **Premium-gated**, consistent with
ABM/ADE, with the enterprise logic in `ee/server/service/`. Note the precedent that Android COBO wipe was
moved to Free (commit `fa7d9282359`), so this is a product decision to confirm rather than an obvious
one.

### Enrollment path changes

- `CreateEnrollmentToken` gains an internal variant for zero-touch: explicit long `duration`,
  `oneTimeOnly: false`, mode-specific `allowPersonalUsage`, and the v2 `additionalData` payload. Keep the
  existing signature intact for the QR path; a shared private helper taking an options struct is cleaner
  than adding another positional `bool` (the current `fullyManaged bool` parameter is already at the
  limit of readability, and Fleet's style guide favors descriptive naming over stacked booleans).
- `enrollHost` and `addNewHost` in `pubsub.go` learn the v2 payload, prefer `team_id` for zero-touch
  tokens, and stop treating an enroll-secret mismatch as fatal in that case — while preserving the
  existing `GetAndroidDeviceLastTeamID` precedence.
- Enroll-secret mutation paths trigger re-issue of affected zero-touch tokens.

### Activities

New activity types, documented in `docs/Contributing/reference/audit-logs.md`:
`enabled_android_zero_touch`, `disabled_android_zero_touch`,
`created_android_zero_touch_configuration`, `edited_android_zero_touch_configuration`,
`deleted_android_zero_touch_configuration`, `rotated_android_zero_touch_enrollment_token`,
`changed_android_zero_touch_device_team`.

### GitOps

Zero-touch configuration is exactly the kind of state that should be declarative. Add to
`mdm.android_zero_touch` in `default.yml` and per-team files:

```yaml
mdm:
  android_zero_touch:
    management_mode: fully_managed   # fully_managed | work_profile | dedicated
    company_name: "Acme Corp"
    contact_email: "it@acme.example.com"
    contact_phone: "+1 555 0100"
    custom_message: "Contact IT if setup does not complete."
    default: true                    # at most one team may set this
    forced_reset_time: 2h
```

Per `CLAUDE.md`'s GitOps notes, two things need explicit tests: **removal** — deleting the key must delete
the Google-side configuration, which is the classic `apply`-inherited bug — and **all three placements**
(`default.yml`, `teams/*.yml`, `teams/no-team.yml`). Also validate that no more than one team claims
`default: true`, within the incoming payload rather than against the DB, since GitOps is declarative.

### Frontend

- **Settings → Integrations → MDM → Android**: extend `AndroidMdmPage.tsx` with a zero-touch section —
  connect/disconnect credentials, linked customer account, ToS acceptance prompt, last sync, per-team
  configuration editor, and the Phase 1 copy-paste extras view. `AndroidMdmCard.tsx` should reflect
  zero-touch status in the summary.
- **Add hosts modal**: `AndroidPanel.tsx` gains a zero-touch tab explaining that eligible devices need no
  QR code, with a link to the settings page. This is also the right place to state the
  bought-from-a-reserved-reseller prerequisite, since that is where the confusion happens.
- **Hosts list**: surface pending zero-touch devices, matching how ADE "awaiting enrollment" hosts appear.
- Free/Primo gating per the `tier-modes` conventions if Premium-gated.
- Command palette entries for the new settings page per the `command-palette` conventions.

### fleetctl

`fleetctl get android-zero-touch` / equivalent for parity with existing MDM inspection commands, plus
whatever `fleetctl gitops` needs to round-trip the new keys.

### Documentation

- New guide in `articles/` — likely `android-zero-touch-enrollment.md`, following the guide format
  (prerequisites, numbered steps, gotchas inline). The reseller prerequisite and the service-account
  linking wait are the two things to lead with.
- Extend `articles/android-mdm-setup.md`.
- REST API reference for the new endpoints.
- An architecture doc under `docs/Contributing/architecture/mdm/`, parallel to
  `automated-device-enrollment.md`.
- Update `docs/Contributing/product-groups/mdm/android-mdm.md`.

## Phased delivery

**Phase 1 — manual configuration (no new Google API).** Long-lived reusable enrollment tokens; the v2
`additionalData` team-identification fix; re-issue on enroll-secret rotation; DPC extras generation and
display; docs. Delivers working zero-touch for every customer, and proves the token model in production
before any API integration exists.

**Phase 2 — customer API integration.** Credential upload and validation; the client package and mock;
the configuration table; create/patch/delete driven from Fleet; reconciliation cron; activities; ToS
error handling; UI. This is where "manage zero-touch from Fleet" becomes true.

**Phase 3 — device-level management.** Device list sync, pending hosts, per-device configuration
override, host-record correlation, hosts-list integration.

**Phase 4 — polish and parity.** GitOps, fleetctl, dedicated-device (`PERSONAL_USAGE_DISALLOWED_USERLESS`)
support, optional provisioning extras (locale, timezone, system apps), and — only if demanded —
`signinDetail`-based end-user authentication.

Phase 1 is independently shippable and worth doing even if Phases 2–4 are never scheduled.

## Edge cases and failure modes

| Scenario | Consequence | Handling |
| --- | --- | --- |
| Enroll secret rotated while a zero-touch token is live | Devices enroll into AMAPI but never become Fleet hosts — silent, delayed | The v2 `team_id` fix plus re-issue on rotation. **Highest-severity item in this document.** |
| Admin edits or deletes the configuration in Google's portal | Fleet's view is wrong; devices provision with stale extras or not at all | Reconciliation cron detects drift, restores, and records `sync_error` |
| Team deleted in Fleet | Orphaned Google configuration; devices provision into nothing | `ON DELETE CASCADE` on the row plus a cron cleanup pass for the Google-side object |
| Team was the zero-touch default | New purchases have no default | Refuse deletion, or reassign the default and warn |
| Token expires | All zero-touch provisioning fails at once, fleet-wide | Rotate 30 days early; alert on approaching expiry; treat expiry as a first-class monitored condition |
| Google ToS updated | Every API call 403s mid-life | Detect `TosError` with `X-GOOG-API-FORMAT-VERSION: 2`; surface an actionable banner |
| Credential revoked or key deleted | Cron fails silently | Mark integration invalid, surface in UI, stop retrying hot |
| Android MDM turned off in Fleet while zero-touch configurations exist | `enterprises.delete` cascades, invalidating the token in every configuration; each device breaks on its *next* reset, long after the action | Block or hard-warn on turn-off; offer a pre-offboarding configuration cleanup. **Release blocker for Phase 2** |
| Customer migrates to another EMM | Devices need a factory reset regardless of EMM; zero-touch claims are unaffected | Document the repoint-then-reset order; do not delete the Fleet enterprise first |
| Two Fleet instances share a customer account | Configurations fight over `isDefault` | Detect unknown configurations; do not blindly delete configurations Fleet did not create |
| Device factory reset | Re-provisions via zero-touch | Existing `GetAndroidDeviceLastTeamID` restores the last-known team; keep that precedence over the token's team |
| Device unclaimed at the reseller | Zero-touch stops applying; re-registration requires contacting the reseller | Reconcile removals; mark pending device gone rather than deleting history; warn that unclaim is not self-service to undo |
| Device never boots | Pending host lingers forever | Show claim age; allow dismissal |
| Configuration applied to a device already in service | **Device factory resets, destroying end-user data**, after a one-hour warning on next contact with Google | Treat `applyConfiguration` and default-configuration changes as destructive in the UI: explicit warning plus confirmation |
| No network during setup | Zero-touch is skipped and the device boots unmanaged, then self-resets on first connection to Google | Expose `forcedResetTime`; document both reset mechanisms and the failure signature |
| Serial number missing or mismatched | Pending device never correlates, or a duplicate host appears | Multi-key correlation with normalization; prefer no correlation over a wrong one |
| `additionalData` exceeds 1024 chars | Token creation fails | Validate length before the API call; keep the payload minimal |
| Non-GMS or pre-8.0 device | Zero-touch simply does not apply | Document; nothing Fleet can do |

## Testing plan

- **Unit** — table-driven tests with `t.Run` subtests over the client interface via the new mock, per
  Fleet's Go test conventions. Cover configuration diffing, default-configuration transitions, token
  rotation ordering (patch before delete), `additionalData` v1/v2 parsing, and serial normalization.
- **Enroll-secret rotation regression** — the specific case above, asserted end to end: create a
  zero-touch token, rotate the team's enroll secret, deliver a synthetic `ENROLLMENT` notification,
  assert the host lands in the right team. This test is the reason to do the work.
- **Integration** — extend `server/mdm/android/tests/` and the AMAPI mock at `cmd/android-amapi-mock/`
  with a provisioning-API surface, so the cron and endpoints can be exercised without Google.
- **Multi-team** — per `CLAUDE.md`, cover "No team", multiple teams, and the default-team transition.
- **Multi-host** — at least two physically distinct zero-touch devices from different manufacturers in
  manual QA, per Fleet's multi-host testing requirement. Serial/IMEI reporting differences between OEMs
  are the likeliest source of correlation bugs, and one device will not reveal them.
- **GitOps** — add, edit, and *remove* the `android_zero_touch` key in all three file placements.
- **Manual QA prerequisites** — a real zero-touch customer test account, obtainable from Google with a
  corporate email, and at least one reseller-claimed device. Note Google's constraint that the DPC must
  be published on Play, which is satisfied automatically since Fleet uses Android Device Policy.
- **Frontend** — Jest coverage for the new settings section and Add-hosts tab.

## Security considerations

- The enrollment token in `dpcExtras` is a bearer credential for joining the AMAPI enterprise. Its blast
  radius is bounded — it enrolls a device into management, it does not read or write Fleet data — but a
  leaked reusable token lets an attacker enroll arbitrary devices into a team, which pollutes inventory
  and could be used to pull team-scoped configuration profiles and their embedded secrets. That last
  point deserves a threat-model review, since Android profiles support variables and certificate
  templates.
- Never place the Fleet enroll secret, IdP credentials, or any Fleet API token in `dpcExtras`. Google's
  own guidance says not to put secrets in configurations; the enrollment token is the single justified
  exception.
- Service-account keys and OAuth refresh tokens must go in `mdm_config_assets` (encrypted at rest), never
  in `app_config_json`, and must be redacted from all logs and API responses. The existing
  `ProxyClient` logger already demonstrates the redaction pattern for `Authorization` headers.
- Reject the shared-service-account multi-tenant proxy model, as described above.
- Authorization on all new endpoints must be team-scoped where the resource is team-scoped, using the
  double-authorize pattern (generic check, load entity, entity-scoped check).
- `customers.devices:unclaim` is irreversible and requires going back to the reseller to undo. Either do
  not expose it or gate it behind explicit typed confirmation.
- Worth a `fleet-security-auditor` pass before merge, given this touches enrollment and MDM credentials.

## Open questions

1. **Credential model** — Option A or B? This blocks Phase 2 and needs confirmation from Google's Android
   Enterprise partner team on whether customer-account service-account linking is still form-based, and
   on the verification status of the `androidworkzerotouchemm` scope.
2. **Proxy divergence** — is the "zero-touch always goes direct to Google" decision acceptable to the
   Fleet Cloud architecture, given every other Android call is proxied?
3. **Premium gating** — Premium-only, or Free like Android COBO wipe?
4. **End-user authentication** — is user-attributed zero-touch a requirement? If yes, `signinDetail` is a
   separate design effort and should be specified before Phase 1 locks the token model.
5. **Pending hosts** — is serial/IMEI correlation reliable enough across OEMs to ship? Needs real device
   data.
6. **API quotas** — what are Google's rate limits on the provisioning API? Determines cron interval and
   pagination strategy for large fleets.
7. **Multiple customer accounts** — can one organization have several zero-touch customer accounts (for
   example, one per reseller or region)? `customers.list` returns a list, and Google's own quickstarts
   just take the first element. Fleet should not do that silently.
8. **Dedicated devices** — is COSU in scope for the first release, or deferred? It affects whether
   management mode needs to be per-configuration rather than per-team.

## Effort estimate

Rough, assuming a backend engineer plus frontend support, and *excluding* the time to resolve the
credential question and obtain a Google test account — both of which are on the critical path and have
external latency.

| Phase | Backend | Frontend | Notes |
| --- | --- | --- | --- |
| 1 — manual configuration | 1–2 weeks | 2–3 days | The `additionalData` fix is most of the risk |
| 2 — customer API integration | 3–4 weeks | 1–1.5 weeks | Client, mock, migration, cron, endpoints |
| 3 — device-level management | 2–3 weeks | 1 week | Correlation is the uncertain part |
| 4 — polish and parity | 1–2 weeks | 2–3 days | Excludes `signinDetail` |

`signinDetail`-based end-user authentication, if required, is an additional 3–4 weeks and should be
scoped separately.

## References

### Google — zero-touch enrollment

- [Overview](https://developers.google.com/zero-touch/guides/overview)
- [How it works (customers)](https://developers.google.com/zero-touch/guides/customer/how-it-works)
- [EMM integration guide](https://developers.google.com/zero-touch/guides/customer/emm)
- [Customer API reference](https://developers.google.com/zero-touch/reference/customer/rest)
- [`customers.configurations`](https://developers.google.com/zero-touch/reference/customer/rest/v1/customers.configurations)
- [`customers.devices`](https://developers.google.com/zero-touch/reference/customer/rest/v1/customers.devices)
- [Authorization](https://developers.google.com/zero-touch/guides/auth)
- [Service accounts for customers](https://developers.google.com/zero-touch/guides/customer/service-accounts)
- [Python quickstart with a service account](https://developers.google.com/zero-touch/guides/customer/quickstart/python-service-account)
- [Zero-touch enrollment for IT admins](https://support.google.com/work/android/answer/7514005)
- [Zero-touch customer portal](https://enterprise.google.com/android/zero-touch/customers)
- [Sample Colab notebooks](https://github.com/googlesamples/zero-touch-enrollment-colabs)

### Google — Android Management API

- [Enroll and provision a device](https://developers.google.com/android/management/provision-device)
- [`enterprises.enrollmentTokens`](https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens)
- [`AllowPersonalUsage`](https://developers.google.com/android/management/reference/rest/v1/AllowPersonalUsage)
- [Notifications (Pub/Sub)](https://developers.google.com/android/management/notifications)

### Fleet

- `server/mdm/android/README.md` — bounded-context rationale
- `server/mdm/android/service/service.go:549` — `CreateEnrollmentToken`
- `server/mdm/android/service/pubsub.go:599` — `enrollHost`
- `server/mdm/android/service/pubsub.go:893` — `addNewHost`
- `server/mdm/android/service/androidmgmt/client.go` — AMAPI client interface
- `cmd/fleet/cron.go:2513` — Android device reconciler cron pattern
- `docs/Contributing/architecture/mdm/automated-device-enrollment.md` — the Apple analogue
- `docs/Contributing/product-groups/mdm/android-mdm.md`
