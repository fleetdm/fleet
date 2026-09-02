# Zero-touch enrollment for company-owned Android devices

Status: research / design proposal. Not yet scheduled.

## Abstract

Fleet manages company-owned Android devices but cannot provision them without a human touching each one:
an admin generates a one-hour enrollment token and someone scans a QR code on every device. Fleet has a
zero-touch path on Apple (Automated Device Enrollment via Apple Business) and none on Android. This
document specifies what it would take to close that gap using Google's Android zero-touch enrollment, so
that a device purchased from a zero-touch reseller ships straight to an employee and provisions itself
into the correct Fleet team on first boot.

Zero-touch introduces no new enrollment protocol. Google's provisioning service pre-stages the same
Android Management API (AMAPI) enrollment token Fleet already creates and delivers it to the device
instead of a QR code. The device then follows the AMAPI path Fleet already supports, and Fleet is notified
by the Pub/Sub `ENROLLMENT` message it already handles. Profile delivery, software installs, commands, and
host vitals are reused unchanged.

Four workstreams follow. **One:** decide how Fleet reaches the customer's zero-touch account. There are
four options, and the choice drives everything else: Google's **zero-touch iframe**, which is an AMAPI call
and so needs no new credential and routes through Fleet's existing proxy, but generates a configuration the
admin cannot modify; a **service-account key** for the Android Device Provisioning Partner API, which fits an
automated cron but currently requires a Google Form and email confirmation to link; **interactive OAuth**,
self-service but bound to one admin's Google account; or **no integration at all**, with Fleet emitting the
provisioning extras for the admin to paste into Google's portal. **Two:** integrate whichever API that
implies — the provisioning API is already in Fleet's module graph, so no new dependency either way.
**Three:** build the Fleet side — long-lived reusable tokens, a configuration per team, a reconciliation
cron, pending-host records, activities, GitOps keys, REST endpoints, UI. **Four:** change how Fleet
identifies the target team at enrollment. Fleet embeds a mutable enroll secret in the token's
`additionalData`, so rotating a team's enroll secret silently breaks enrollment for every device that Fleet
has not seen before — which is every new zero-touch device. That is a correctness prerequisite, not an
enhancement.

Phase 1 ships usable zero-touch with **no** Google API integration: Fleet emits the DPC extras JSON and the
admin pastes it into Google's portal once per team. It is small, unblocks customers immediately, and proves
the token and team-identity model before the access decision is made. One question should be answered before
anything is built, because it is cheap and it reshapes the plan: whether the zero-touch iframe can carry
per-team provisioning extras. If it can, it removes the credential problem entirely.

## Table of contents

- [Scope](#scope)
- [How Android zero-touch enrollment works](#how-android-zero-touch-enrollment-works)
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

Android is the only company-owned platform where Fleet has no zero-touch path.

| Platform | Zero-touch mechanism | Fleet status |
| --- | --- | --- |
| macOS / iOS / iPadOS | Apple Automated Device Enrollment (ADE) via Apple Business | Supported. See `docs/Contributing/architecture/mdm/automated-device-enrollment.md` |
| Android (company-owned) | Google Android zero-touch enrollment | **Not supported — this document** |
| Windows | Windows Autopilot (via Entra ID) | Supported. See `docs/Contributing/product-groups/mdm/windows-autopilot.md` and `articles/windows-mdm-setup.md` |
| ChromeOS | Chrome zero-touch enrollment (same provisioning API, `deviceType` `DEVICE_TYPE_CHROME_OS`) | Out of scope; Fleet has no ChromeOS MDM |

Zero-touch supports three management modes, differing only in the `allowPersonalUsage` value on the
enrollment token:

- **Fully managed (COBO)** — `PERSONAL_USAGE_DISALLOWED`.
- **Company-owned with a work profile (COPE)** — `PERSONAL_USAGE_ALLOWED`. **Android 10+ for zero-touch
  specifically**: the zero-touch EMM guide requires Android 8.0+ for fully managed but says "Company-owned
  Android 10+ devices can be provisioned as fully managed or with a work profile." Android 11 further
  reworked COPE to treat the personal side closer to BYOD.
- **Dedicated / kiosk (COSU)** — `PERSONAL_USAGE_DISALLOWED_USERLESS`, which Google made mandatory for
  dedicated-device enrollment as of January 2025.

Zero-touch does **not** support work profiles on personally-owned devices. Google's provisioning guide
lists the methods per ownership type as prose rather than a table; zero-touch appears under both
company-owned sections and is absent from the personally-owned list, which offers only Settings, the
Android Device Policy app, an enrollment-token link, and sign-in URL. BYOD is therefore out of scope by
mechanism, not just by policy.

## How Android zero-touch enrollment works

### Actors

1. **Reseller** — an authorized zero-touch reseller. On purchase, the reseller registers each device's
   hardware identifiers (IMEI/MEID or serial number, plus manufacturer) and *claims* the device to the
   organization's customer account. **Only resellers can create device records.** Google states this
   directly — "Resellers create devices when a customer purchases them for zero-touch enrollment—IT admins
   can't create devices" — and there is no device-create method on the customer API, only
   `partners.devices.claim` on the reseller API. Note that the *same* Google page also says "An
   organization can also create configurations and devices using the zero-touch enrollment portal," which
   contradicts it. Treat the reseller-only rule as correct, since it is the one corroborated by the API
   surface and by the support docs, but the contradiction is unresolved.
2. **Customer account** — the organization's zero-touch account, created by the reseller on first
   purchase, managed at `https://enterprise.google.com/android/zero-touch/customers`, addressed as
   `customers/{CUSTOMER_ID}`.
3. **DPC (Device Policy Controller)** — the agent that provisions and manages the device. For AMAPI-based
   EMMs, that is Google's **Android Device Policy** (`com.google.android.apps.work.clouddpc`). Fleet ships
   no DPC.
4. **EMM (Fleet)** — provides the console where admins manage configurations, calling the customer API on
   the admin's behalf.

### Configurations

A **Configuration** is owned by the customer account and bundles the DPC to install, the provisioning
extras handed to that DPC, support contact details, and a message shown during setup.

| Field | Required | Notes |
| --- | --- | --- |
| `configurationName` | Yes | Shown to admins in the portal. Fleet would use the team name. |
| `dpcResourcePath` | Yes | `customers/{CUST_ID}/dpcs/*`. Discover via `customers.dpcs.list`; do not hardcode. |
| `dpcExtras` | No | JSON string of provisioning extras. **Fleet's enrollment token goes here.** |
| `companyName` | Yes | Shown on device during provisioning. Fleet would default to the org name in app config. |
| `contactEmail` | Yes | Validated by Google. Shown before provisioning. |
| `contactPhone` | Yes | Shown before provisioning. Digits, spaces, `+`, `-`, `()`. |
| `customMessage` | No | One or two sentences shown before provisioning. |
| `isDefault` | Yes | Applies to devices the organization purchases in future. Setting it to `true` sets the previous default's `isDefault` to `false`. |
| `forcedResetTime` | No | Timeout before forcing a factory reset when setup cannot complete, typically for lack of network. 0–6 hours, default 2. |
| `name`, `configurationId` | Output only | Server-assigned. |

### Device records and configuration assignment

Devices expose `deviceIdentifier` (serial number, manufacturer, model, IMEI/MEID, plus `imei2`/`meid2` for
dual-SIM), `deviceMetadata` (reseller-set key/value pairs), `claims[]` (the zero-touch claim,
`SECTION_TYPE_ZERO_TOUCH`), and `configuration` (the applied configuration path, or empty).

A device gets a configuration either implicitly from the account's **default configuration**, which
applies to newly purchased devices, or explicitly via `customers.devices:applyConfiguration`.

`customers.devices:removeConfiguration` removes a configuration from a device.
`customers.devices:unclaim` removes the device from zero-touch entirely; re-registering requires going
back to the reseller.

### First-boot flow

1. Device is unboxed or factory reset and connects to a network during setup.
2. It checks in with Google's provisioning servers, identifying itself by hardware ID.
3. If a configuration is assigned, the device enters the fully managed setup wizard, showing
   `companyName`, `customMessage`, `contactEmail`, and `contactPhone`.
4. The device installs Android Device Policy from Google Play.
5. Android Device Policy is launched with the extras from `dpcExtras`, reads the enrollment token, and
   enrolls into the AMAPI enterprise. (Google's zero-touch guide still names the
   `ACTION_PROVISION_MANAGED_DEVICE` intent here, but that action was deprecated in API level 31 and now
   fails on Android 12+; the current flow uses `ACTION_PROVISION_MANAGED_DEVICE_FROM_TRUSTED_SOURCE` with
   `ACTION_GET_PROVISIONING_MODE`. This is informational for Fleet, which does not implement a DPC.)
6. AMAPI publishes an `ENROLLMENT` notification to the enterprise's Pub/Sub topic. **Fleet re-enters here,
   on the code path it already has.**

### When provisioning triggers

Zero-touch is evaluated **only during the out-of-box setup wizard** — on first boot or after a factory
reset. It is not a push mechanism and cannot provision a device that is already set up and running. Three
consequences shape the design:

- **It re-triggers on every factory reset, for as long as the claim exists.** The end user cannot bypass
  or defer it. This is the security property that makes zero-touch valuable for company-owned hardware,
  and the reason a broken configuration is a fleet-wide outage rather than a per-device annoyance.
- **Assigning a configuration does not reset a device that is already in service.**
  `customers.devices:applyConfiguration` is documented as: "After applying a configuration to a device, the
  device automatically provisions itself on first boot, or next factory reset." Google's IT-admin help says
  the same. Assignment is therefore a staging operation; nothing happens until the next wipe.
  **Verify before building on this.** Google is silent on what happens to an already-provisioned device
  that is later assigned a configuration, so "waits for the next reset" is the documented behaviour rather
  than a guarantee about every state. Note the adjacent sentence quoted in the next bullet is easy to
  misread as applying here — its subject is a device still in the setup wizard, not one in service.
- **A device that skips zero-touch for lack of network resets itself later.** If a device has a
  configuration but no connectivity during setup, zero-touch is skipped and the device boots unmanaged —
  then "resets itself after the first connection to Google servers," warning the user one hour ahead. This
  is distinct from `forcedResetTime` (0–6 hours, default 2), which is the in-setup timeout when the wizard
  cannot reach Google. Only the latter is configurable. Google documents a third trigger in the same
  family: registered dual-SIM devices shipped with Google Play Services older than 24.07.12 factory-reset
  if zero-touch does not provision them during initial setup.

### DPC extras for AMAPI

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

**Google's two documentation sets contradict each other here.** The AMAPI provisioning guide gives exactly
the payload above, including the component name and signature checksum. The zero-touch EMM integration
guide advises against including device-admin component name and checksum extras, calling them
inappropriate for this enrollment method. Follow the AMAPI guide, since it is specific to Android Device
Policy, and verify on real hardware before shipping — a wrong payload here fails at provisioning time on
the device, where it is expensive to debug.

Other guidance from the EMM guide that does apply:

- Put Fleet-specific values inside `PROVISIONING_ADMIN_EXTRAS_BUNDLE`, not as top-level extras.
- Do not put credentials or secrets in a configuration. The enrollment token is the one unavoidable
  exception and is scoped to enrollment.
- Supported and reasonable to expose later: `EXTRA_PROVISIONING_LOCALE`, `EXTRA_PROVISIONING_TIME_ZONE`,
  `EXTRA_PROVISIONING_LOCAL_TIME`, `EXTRA_PROVISIONING_LEAVE_ALL_SYSTEM_APPS_ENABLED`,
  `EXTRA_PROVISIONING_MAIN_COLOR`, `EXTRA_PROVISIONING_DISCLAIMERS`.
- Avoid package download location and cookie-header extras; they belong to other enrollment methods.

### Device eligibility

- Android 8.0+ (Pixel 7.1+) per the zero-touch EMM guide. Google's Workspace-facing documentation says
  9.0+ (Pixel 7.0+) for work-profile devices, so the floor varies by source and mode. Verify against the
  Android Enterprise device list rather than either number.
- Google Mobile Services present.
- Registered by a participating reseller and claimed to the organization's customer account.

### Devices the organization already owns

An organization's existing Android fleet generally cannot be moved onto zero-touch, and this is the most
consequential limitation for adoption. Registration is performed by the reseller that sold the device;
there is no self-service path, and Google's newer "automatic zero-touch enrollment" framing in the
Workspace documentation still requires reseller registration. A device bought at retail or through a
non-participating channel has no route in.

Devices bought from a participating reseller can often be registered retroactively on request — the
reseller has the tooling (`partners.devices.claim`) and the sales record — but that is the reseller's call
and outside Fleet's control. Deregistering also requires the reseller to reverse.

So zero-touch is a forward-looking capability tied to procurement, not a migration path for an installed
base. For devices already owned, Fleet's existing enrollment link and QR code reach the same *management*
end state — a fully managed device in the correct team — at one human touch per device at factory-reset
time.

What that does not replicate is stickiness. A QR-enrolled device that is factory reset boots unmanaged; a
zero-touch device re-provisions itself. Where the requirement is tamper resistance rather than
provisioning convenience, only reseller registration satisfies it. These are two different value
propositions leading to different purchasing decisions and should not be conflated in messaging.

Noted for completeness and **not proposed here**: Samsung Knox Mobile Enrollment supports admin
self-registration of already-owned devices via the Knox Deployment App or a KME QR profile, without
reseller involvement, and can target AMAPI. It is a separate program and a distinct integration effort.

## Where Fleet is today

Fleet's Android MDM lives in `server/mdm/android/` as a decoupled bounded context (see
`server/mdm/android/README.md`).

### Enrollment token creation

`Service.CreateEnrollmentToken` (`server/mdm/android/service/service.go:549`) is the whole current
enrollment surface. It:

1. Verifies Android MDM is configured.
2. Verifies the enroll secret (`svc.ds.VerifyEnrollSecret`, `service.go:558`). Note the result is
   discarded here — the team is not resolved at token-creation time, only at enrollment.
3. Optionally requires and validates an IdP account UUID for end-user authentication.
4. Marshals an `enrollmentTokenRequest` into the token's `additionalData`. **The struct carries only
   `query:` tags, no `json:` tags** (`service.go:480-484`), so the wire format uses Go field names and
   includes a third field:
   `{"EnrollSecret":"...","FullyManaged":false,"IdpUUID":"..."}`. `FullyManaged` is always `false`
   because `CreateEnrollmentToken` never sets it when marshalling. Fleet's own integration tests encode
   this shape (`server/mdm/android/tests/integration_os_version_test.go:82`).
5. Sets `allowPersonalUsage` from a `fully_managed` query parameter — `PERSONAL_USAGE_DISALLOWED` when
   true, `PERSONAL_USAGE_ALLOWED` otherwise (`service.go:608`).
6. Sets `oneTimeOnly: true` and leaves `duration` unset, so **the token expires in one hour and works
   once**.
7. Returns the token value, an `https://enterprise.google.com/android/enroll?et=<token>` URL, and a QR
   code.

The frontend surface is `frontend/components/AddHostsModal/PlatformWrapper/AndroidPanel/AndroidPanel.tsx`,
which offers a work-profile / fully-managed radio and builds a Fleet enrollment URL.

### Enrollment completion

`ENROLLMENT` notifications are dispatched by `ProcessPubSubPush` (`pubsub.go:66`) to
`handlePubSubEnrollment` (`pubsub.go:512`), which calls `Service.enrollHost` (`pubsub.go:599`).
It unmarshals `device.EnrollmentTokenData` back into the same struct, re-verifies the enroll secret,
resolves the team, and creates or updates the host. `addNewHost` (`pubsub.go:893`) is the
new-device path and maps AMAPI `hardwareInfo.serialNumber` into `host.HardwareSerial` (`pubsub.go:947`),
which matters for correlating zero-touch device records later. `ENROLLMENT` is among the notification
types Fleet enables at enterprise creation (`service.go:299`).

### AMAPI client

`androidmgmt.Client` (`server/mdm/android/service/androidmgmt/client.go`) has two implementations:

- `ProxyClient` — the default. Routes AMAPI calls through `https://fleetdm.com/api/android/` using a
  per-enterprise `FleetServerSecret` bearer token. Fleet's hosted proxy owns the Google Cloud project, the
  service credentials, and the Pub/Sub topic.
- `GoogleClient` — development only, selected when `FLEET_DEV_ANDROID_GOOGLE_CLIENT` is `1`/`ON`
  (`service.go:115`) with credentials in `FLEET_DEV_ANDROID_GOOGLE_SERVICE_CREDENTIALS`. Talks to Google
  directly with a service account.

This proxy architecture is the most important existing constraint on the design.

### Storage

`android_enterprises` holds one row: `enterprise_id`, `signup_name`, `signup_token`, `pubsub_topic_id`,
`user_id`. Secrets live in `mdm_config_assets`, encrypted at rest with the server private key
(`server/datastore/mysql/apple_mdm.go:6007`), under names in `server/fleet/mdm.go` —
`android_pubsub_token`, `android_fleet_server_secret`. There is no per-team Android enrollment state.

## Gap analysis

| Capability | Needed for zero-touch | Fleet today |
| --- | --- | --- |
| Long-lived, reusable enrollment token | Yes — one token serves many devices over months | 1 hour, `oneTimeOnly: true` |
| Stable team identification in the token | Yes — the token outlives enroll secret rotations | Embeds the mutable enroll secret |
| Zero-touch customer API client | Yes | Does not exist |
| Credential storage for that API | Yes | Does not exist |
| Configuration-per-team model | Yes | Does not exist |
| Reconciliation of Google-side state | Yes — admins can edit configurations in the portal | Does not exist |
| Pending/awaiting-enrollment host visibility | Desirable, matches ADE | Does not exist for Android |
| Per-device configuration override | Yes — how a device lands in a non-default team | Does not exist |
| Activities / audit trail | Yes | Does not exist |
| GitOps | Yes, for parity | Does not exist |
| `PERSONAL_USAGE_DISALLOWED_USERLESS` (dedicated) | For COSU | Not exposed |
| End-user IdP auth during zero-touch provisioning | Depends on requirements | BYOD-cookie-based only; not reachable from zero-touch |

## Google APIs required

### 1. Android Device Provisioning Partner API — customer surface (new)

- **Service**: `androiddeviceprovisioning.googleapis.com`
- **Go package**: `google.golang.org/api/androiddeviceprovisioning/v1` — already in Fleet's module graph
  via `google.golang.org/api v0.269.0` (`go.mod:181`). No new dependency. It exposes `CustomersService`,
  `CustomersConfigurationsService`, `CustomersDevicesService`, and `CustomersDpcsService`.
- **OAuth scope**: `https://www.googleapis.com/auth/androidworkzerotouchemm`. This differs from the
  reseller scope `https://www.googleapis.com/auth/androidworkprovisioning`. The generated package declares
  no default scopes, so it must be passed explicitly via `option.WithScopes(...)`.
- **Enablement**: the API must be enabled on the Google Cloud project owning the credential.
- **Required header for error handling**: `X-GOOG-API-FORMAT-VERSION: 2`, without which Google does not
  return the structured `TosError` detail described below.

| Method | Use in Fleet |
| --- | --- |
| `customers.list` | Discover the customer accounts the credential can act on. Returns `Company` objects — "the customer accounts the calling user is a member of" — and is paginated. |
| `customers.dpcs.list` | Find the Android Device Policy resource path for `dpcResourcePath`. Match on the last path component; never hardcode. |
| `customers.configurations.list` | Diff Fleet's rows against Google's; detect out-of-band edits and deletions. |
| `customers.configurations.create` | Create a configuration per Fleet team. |
| `customers.configurations.patch` | Update `dpcExtras` on token rotation, metadata on team rename. |
| `customers.configurations.delete` | Remove a configuration when a team is deleted or opts out. |
| `customers.devices.list` | Enumerate claimed devices for pending-host records. Paginated via `nextPageToken`. |
| `customers.devices.get` | Refresh a single device. |
| `customers.devices:applyConfiguration` | Assign a device to a specific team's configuration. Takes effect at the device's next factory reset, not immediately. |
| `customers.devices:removeConfiguration` | Remove a device's configuration. |
| `customers.devices:unclaim` | Destructive and reseller-only to reverse; expose behind explicit confirmation, if at all. |

Not needed: the `partners.*` reseller surface. Device claiming is a supply-chain function and Fleet should
not attempt to become a reseller.

**Terms of service**: calls return `403 Forbidden` with a `TosError` body until the acting user has
accepted the current zero-touch terms. Fleet must detect this case specifically and link to
`https://enterprise.google.com/android/zero-touch/customers`. Google can update terms at any time, so this
is not a setup-only error — it can appear on a working integration and must be handled in the cron.

### 2. Android Management API (existing, extended)

New usage of an API Fleet already calls:

- `enterprises.enrollmentTokens.create` with:
  - `duration` set explicitly. Accepts 1 minute to roughly 10,000 years; default 1 hour. A duration that
    would overflow the maximum timestamp is coerced.
  - `oneTimeOnly: false`, so many devices can use one token.
  - `allowPersonalUsage` per management mode.
  - `additionalData` — max 1024 characters, surfaced on the device as `enrollmentTokenData`. Fleet's only
    channel for carrying team intent through provisioning; the ceiling is a real constraint.
  - `policyName` — Fleet already points this at `{enterprise}/policies/1`
    (`android.DefaultAndroidPolicyID`).
- `enterprises.enrollmentTokens.delete` — revoke a leaked or superseded token.
- `enterprises.enrollmentTokens.list` — lists active, unexpired tokens, but **returns only a partial
  view**: `name`, `expirationTimestamp`, `allowPersonalUsage`, `value`, `qrCode`. `additionalData` and
  `policyName` are absent, so it cannot detect drift in the fields this design cares about. Useful for
  lifecycle cleanup only.
- `EnrollmentToken.user` is deprecated and ignored. Do not use it.

### 3. Google Cloud Pub/Sub (existing)

Unchanged. Zero-touch devices produce the same `ENROLLMENT` notification as QR-code devices.

### 4. Optional: AMAPI `signinDetail`, only if end-user IdP auth is required

Zero-touch hands the device a token with no browser involved, so Fleet's existing IdP flow cannot apply.
That flow takes the IdP account either from an explicit `idp_uuid` query parameter, which is checked first
(`service.go:500`), or from a cookie (`mdm.BYODIdpCookieName`, `service.go:508`) — both of which require a
browser. Note the requirement itself is not BYOD-specific: `RequiresEnrollOTAAuthentication`
(`pkg/mdm/ota_enroll.go`) keys off the team's setup-experience IdP setting and applies regardless of
`fullyManaged`.

AMAPI's sign-in URL enrollment is the mechanism for authenticating a user during provisioning. An
enterprise can hold any number of `SigninDetail` entries, keyed by (`signinUrl`, `allowPersonalUsage`,
`tokenTag`), each yielding a server-generated read-only `signinEnrollmentToken`. The sign-in endpoint must
finish by redirecting to `https://enterprise.google.com/android/enroll?et=<token>` on success or
`https://enterprise.google.com/android/enroll/invalid` on failure.

Google documents this combination explicitly: "Add the resulting `signinEnrollmentToken` as provisioning
extra to a QR code, NFC payload, or **Zero-touch configuration**." Google's canonical `dpcExtras` example is
in fact the *sign-in* variant — its placeholder inside `PROVISIONING_ADMIN_EXTRAS_BUNDLE` is
`"{Sign In URL token}"`, not a plain enrollment token.

**A cheaper mechanism probably covers Fleet's need.** `googleAuthenticationOptions` on the enrollment token
takes `authenticationRequirement` (`OPTIONAL` / `REQUIRED`) and `requiredAccountEmail`, overrides the
enterprise-wide `googleAuthenticationSettings` policy, and hides the Skip button on the Google sign-in
screen. It applies to a plain enrollment token and therefore to zero-touch, giving account-attributed
enrollment with no device-facing web flow. Caveat: it is **not** present in the pinned
`google.golang.org/api v0.269.0` generated code, so it needs a library bump. Google also notes that
`PERSONAL_USAGE_DISALLOWED` already "requires users to sign in with a work email to access the device."

`signinDetail` remains a substantial sub-project — Fleet would serve an unauthenticated, device-facing web
flow that mints enrollment tokens. **Recommendation: exclude `signinDetail` from initial scope and evaluate
`googleAuthenticationOptions` first**, since it is a field on a call Fleet already makes.

## Credential and authorization options

This decision gates Phase 2 and should be settled before implementation starts.

### Option A — Service account key uploaded by the admin

The admin creates a Google Cloud project, enables the API, creates a service account, downloads its JSON
key, and uploads it to Fleet. Fleet uses two-legged OAuth with the `androidworkzerotouchemm` scope.

- **Pros**: no refresh-token lifecycle, no consent UI, no dependency on one person's Google account.
  Google recommends service accounts for continuously-running automated services, which is what the
  reconciliation cron is. Mirrors Fleet's ABM token UX: do work in the vendor portal, download a
  credential, upload it.
- **Cons**: the service account must be linked to the customer account, and per Google's current
  documentation that linking is a **Google Form submission followed by email confirmation**, not a portal
  action. That is human latency of unknown duration on the customer's critical path, opaque to Fleet.
- **Verify first**: whether the customer portal has since gained a self-service linking screen (the
  reseller portal has one), and whether there is a limit on linked service accounts per organization.

### Option B — Interactive OAuth (three-legged)

Fleet registers an OAuth client; the admin consents with the Google account their reseller established in
the zero-touch account; Fleet stores the refresh token. This is the pattern Google's EMM integration guide
describes.

- **Pros**: self-service, no Google Form, works immediately after consent. Consistent with Fleet's AMAPI
  signup, which already round-trips through `fleetdm.com` with a callback URL.
- **Cons**: the grant is bound to one person's Google account and breaks when their access is removed — a
  recurring support burden. `androidworkzerotouchemm` is likely a sensitive scope requiring Google app
  verification, a schedule risk. Refresh tokens can be revoked without notice, so Fleet needs re-consent
  detection. Self-hosted Fleet needs its own OAuth client or a proxy-mediated redirect URI.

### Option C — No API: Fleet emits the DPC extras, admin pastes them

Fleet generates a long-lived reusable token per team and displays the complete `dpcExtras` JSON with a
copy button and portal instructions. The admin creates the configuration by hand.

- **Pros**: no API integration, no credential storage, no authorization risk. Works for every customer
  including self-hosted. Ships in one release.
- **Cons**: per-team manual portal work. No pending-host visibility, no per-device team override, no drift
  detection.

### Option D — The AMAPI zero-touch iframe

Google ships a path purpose-built for AMAPI EMMs that avoids the provisioning API entirely. Fleet calls
`enterprises.webTokens.create` with `iframeFeature: 'ZERO_TOUCH'`, then embeds
`https://enterprise.google.com/android/zero-touch/embedded/companyhome` with `token`, `dpcId`
(`com.google.android.apps.work.clouddpc`), and an optional `dpcExtras` URL parameter carrying URL-encoded
provisioning extras. The admin links their own zero-touch account from inside the iframe.

- **Pros**: `webTokens.create` is an **AMAPI** call, so it routes through Fleet's existing proxy with the
  existing credential. No customer-supplied credential, no Google Form, no OAuth client, no new API
  integration, and no new secret to store. Google's IT-admin help actively points customers at it: "Many
  EMMs also implement the zero-touch iframe to simplify the process of setting up zero-touch devices after
  you purchase them from a reseller."
- **Cons**, and they are structural: "The zero-touch iframe automatically generates a zero-touch
  configuration. This configuration is not modifiable by the IT admin." Fleet sets the extras once via the
  `dpcExtras` URL parameter and otherwise does not control the configuration. That appears to rule out the
  configuration-per-team model this document is built on, and with it per-device team assignment,
  pending-host records, and drift detection — everything that needs `customers.*`. Google also describes
  the generated profile applying to devices in the account that have no profile, which is default-like
  behaviour rather than per-team targeting.
- **Verify before choosing**: whether more than one configuration can be generated per enterprise, and
  whether `dpcExtras` can be varied per team through separate iframe renderings. If it can, Option D may
  dominate A and B outright. If it cannot, Option D is a low-effort single-team path and the
  provisioning-API options remain necessary for team granularity.

### Recommendation

The right sequence depends on the Option D question above, which should be resolved first because it is
cheap to answer and changes the plan materially.

- If Option D supports only one effective configuration per enterprise: ship **C as Phase 1** (it proves
  the token and team-mapping model with no integration), then **A as Phase 2** for team granularity, with
  **B** as an alternate for customers blocked on service-account linking. Consider also offering **D** as a
  one-click path for single-team deployments, since it is nearly free once `webTokens` is wired up.
- If Option D supports per-team extras: make **D** the primary path and drop A and B to a later phase or
  out of scope. It removes the entire credential problem, which is otherwise this project's largest risk.

Either way C remains worth shipping first: it is small, unblocks customers immediately, and everything it
forces Fleet to get right — long-lived tokens, the team-identity fix, extras generation — is reused by every
other option.

### The Fleet Cloud proxy question

This applies to Options A and B only; Option D routes through the existing proxy by construction.

Fleet's AMAPI traffic defaults through `https://fleetdm.com/api/android/`, with Fleet's hosted project
owning the credentials. The **provisioning API** cannot follow that model:

- The customer's zero-touch account was established by *their* reseller. Fleet's hosted service account
  has no relationship to it and cannot gain one without the customer linking it.
- A single shared service account linked to many customer accounts would make `customers.list` return
  every linked customer to every caller, putting Fleet's proxy in charge of tenant isolation on a
  credential with no per-tenant scoping. **Reject this.**

So under A or B, zero-touch credentials are **per-Fleet-instance and customer-supplied** and provisioning
API calls go **direct to Google**, regardless of how AMAPI traffic is routed. That is a deliberate
divergence from the rest of Android MDM and should be commented in the code. Under D the question does not
arise, which is a further argument for resolving D first.

## Design

### Configuration per team

Each Fleet team that opts into zero-touch gets one long-lived enrollment token carrying that team's
identity in `additionalData` and its management mode in `allowPersonalUsage`, plus one Configuration whose
`dpcExtras` embeds that token and whose `configurationName` derives from the team name.

Google permits one default configuration per customer account, so Fleet exposes a single **default team
for zero-touch**. That team's configuration carries `isDefault: true` and receives all newly purchased
devices; other teams' devices are placed with `customers.devices:applyConfiguration`. Changing the default
patches two configurations, and Fleet must read back Google's state rather than assume its own view is
authoritative, because setting a new default silently flips the old one to `false`.

`allowPersonalUsage` is fixed at token creation, so a configuration is per (team, management mode). Keep
mode a per-team setting; teams needing both fully managed and dedicated devices should be separate teams.

### Fixing team identification: the critical prerequisite

`additionalData` carries `{"EnrollSecret":"...","FullyManaged":false,"IdpUUID":"..."}` — Go field names,
because the struct has no `json:` tags — and `addNewHost` hard-fails at `pubsub.go:908` when the secret
does not verify:

```go
enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, enrollmentTokenRequest.EnrollSecret)
if err != nil {
    return ctxerr.Wrap(ctx, err, "verifying enroll secret")
}
```

Fine for a one-hour token. For a token embedded in a configuration for a year it is a latent outage:
**once an admin rotates that team's enroll secret, every zero-touch device Fleet has never seen before
enrolls into AMAPI and then fails to become a Fleet host** — managed by Google, invisible in Fleet, visible
only as a repeating Pub/Sub error.

The scope matters. `enrollHost` branches on `getExistingHost`, and only the new-device path is fatal. For a
device Fleet already knows, a secret that fails to verify is explicitly tolerated (`pubsub.go:628-631`,
`if err != nil && !fleet.IsNotFound(err)`) and the host keeps its existing team. Since `getAndroidHostKey`
is stable across unenroll/re-enroll for the same device and enterprise, a *factory-reset* zero-touch device
takes the tolerant path and survives. The breakage lands on new hardware out of the box — the primary
zero-touch scenario, so the finding stands, but it is not every device.

Two fixes, both needed:

1. **Carry a stable team reference.** Version the `additionalData` payload and add a durable identifier,
   for example `{"v":2,"zt":true,"team_id":3,"enroll_secret":"..."}`. In `enrollHost`/`addNewHost`, prefer
   `team_id` when `zt` is set and treat a secret mismatch as non-fatal in that case. A team UUID would be
   more stable than a numeric ID across restores, if one exists. Validate the payload against the
   1024-character limit rather than assuming headroom.
2. **Re-issue on rotation.** Hook enroll-secret changes and patch every affected configuration with a
   fresh token.

Fix 1 alone leaves a stale secret in Google's configuration. Fix 2 alone leaves a window where in-flight
provisioning breaks, and fails outright if the patch call errors.

The payload must be versioned because v1 tokens already exist in the wild. Preserve the existing
precedence in `pubsub.go:916`, where `GetAndroidDeviceLastTeamID` overrides the token's team for
previously-known devices, so a factory-reset device returns to the team an admin moved it to.

### Token lifetime and rotation

Create tokens with a long explicit duration and rotate on demand rather than on a short clock. A
mid-provisioning token swap is the riskiest moment in the flow, and frequent rotation multiplies that risk
for little gain: the token's only power is enrolling a device into the enterprise.

**This deviates from Google's stated guidance**, which is worth naming rather than glossing: "For security
reasons, it's recommended to delete active enrollment tokens as soon as they're not intended to be used
anymore," and on create, "It's up to the caller's responsibility to manage the lifecycle of newly created
tokens and deleting them when they're not intended to be used anymore." That guidance is written for
short-lived QR enrollment, where a token really is finished after one device. Zero-touch structurally needs
a token that stays valid as long as a configuration references it, so both cannot hold. Mitigation: keep
exactly one live token per configuration, delete superseded ones promptly, and treat the stored value as a
credential — rather than shortening its life until expiry becomes the likelier outage. Record this as a
deliberate deviation.

- Create with `duration` around one year and `oneTimeOnly: false`.
- Store `expires_at`; have the cron rotate within roughly 30 days of expiry.
- On rotation, mint the new token, patch `dpcExtras`, confirm the patch succeeded, and only then delete the
  old token — never the reverse. Keep a grace period so devices that already fetched the old configuration
  can finish.
- Provide an explicit "rotate now" action for the leak case, which deletes the old token immediately and
  accepts in-flight breakage.

### Pending hosts

`customers.devices.list` returns hardware identifiers before a device has ever booted, and AMAPI reports
`hardwareInfo.serialNumber` at enrollment, which Fleet stores as `HardwareSerial` (`pubsub.go:947`). That
enables an Android analogue of Apple's `host_dep_assignments` "awaiting enrollment" view.

Caveats:

- Serial number is the practical join key, but some devices report IMEI/MEID only, and dual-SIM devices
  have `imei2`/`meid2`. Try serial, then IMEI/MEID, and tolerate no match.
- AMAPI documents that on personally-owned devices running Android 12+, `serialNumber` is the same value as
  `enterpriseSpecificId` rather than a real serial. Company-owned zero-touch devices should report a true
  serial, but the code must not assume the field is meaningful.
- Normalize on both sides before comparing. `fleet.Preprocess` (`server/fleet/utils.go:65`) covers only
  whitespace trimming and Unicode NFC — it does **not** case-fold or strip leading zeros, so those need
  handling separately. Google also warns that `serialNumber` "might not be unique across different device
  models," so serial alone is not a safe key. For dual-SIM devices Google advises resellers to register the
  numerically lowest IMEI, which is worth confirming with the customer's reseller.
- A wrong match creates duplicate host records, which is worse than no pending hosts. If confidence is
  low, ship without pending hosts and add them once real device data exists.

### Ownership, portability, and offboarding

Two registrations exist, owned by different parties. Conflating them produces wrong conclusions about
lock-in.

**The zero-touch claim belongs to the customer.** It lives in their customer account, created by their
reseller against their organization. Fleet is not a party to it. Fleet's credential is a delegated caller
the customer can revoke at any time without affecting the claim.

**The AMAPI enterprise is bound to Fleet.** `EnterprisesCreate` passes `ProjectId(g.androidProjectID)`
(`androidmgmt/google_client.go:92`), so the enterprise belongs to a Google Cloud project — Fleet's, in the
default proxy deployment. That makes it an *EMM-managed* enterprise, which cannot be transferred to
another EMM's project. A deployment using `GoogleClient` with the customer's own credentials would create
it under the customer's project instead; that path is development-only today, but it is the lever if
enterprise ownership becomes a customer requirement.

**The only coupling between the two systems is the enrollment token string in `dpcExtras`** — a mutable
field in a record the customer owns.

Migrating away from Fleet:

- **The zero-touch side migrates cleanly.** Portal admins replace the token in `dpcExtras` with the
  incoming EMM's, or point devices at a new configuration. Devices stay claimed; no reseller involvement.
  They can do this **without Fleet's cooperation**, even if the instance is gone. Worth stating plainly in
  customer-facing material.
- **The AMAPI side requires a factory reset per device.** Device Owner cannot be transferred over the air
  between EMM enterprises. AMAPI's DPC migration facility covers custom-DPC-to-Android-Device-Policy
  *within one enterprise*, not cross-EMM. This is a universal Android Enterprise constraint and applies
  identically to customers migrating *to* Fleet.
- **Zero-touch makes migration easier.** Repoint the configuration, then factory reset in batches; each
  device self-provisions into the new EMM with no hands-on step.

**Offboarding hazard: the current turn-off flow is unsafe.** `enterprises.delete` performs "a cascaded
deletion of all AM API devices associated with the deleted enterprise" and is only available for
EMM-managed enterprises. `Service.DeleteEnterprise` (`service.go:424`, calling `EnterpriseDelete` at
`service.go:442`) runs when an admin turns off Android MDM. **It is not the only caller**: a second path
reaches `EnterpriseDelete` at `service.go:820` from `VerifyExistingEnterpriseIfAny` (`service.go:801`), which
the `cleanup_android_enterprise` cron job invokes (`cmd/fleet/cron.go:1600`). The cascade can therefore fire
with no admin action at all. If zero-touch configurations exist, deleting
the enterprise invalidates the token still embedded in them, so every subsequent factory reset hits a dead
token and cannot complete provisioning. **Turning off Android MDM therefore breaks provisioning for every
zero-touch-claimed device, with damage surfacing only at each device's next reset.**

Mitigations:

- Turn-off must detect existing zero-touch configurations and block or hard-warn, listing affected teams
  and device counts. Treat as a Phase 2 release blocker.
- The `cleanup_android_enterprise` cron path needs the same guard. A UI-only check leaves the cascade
  reachable from a background job, which is the worse case because no one is watching when it fires.
- Provide a "prepare for migration" action that deletes Fleet's configurations (leaving claims intact)
  before enterprise deletion, so devices boot unmanaged instead of looping.
- Document the order: repoint or delete configurations → reset devices → delete the enterprise.
- Verify on hardware what state devices land in after a cascaded enterprise deletion. Google does not
  document this clearly, and the offboarding guidance is not trustworthy until someone checks.

### Reconciliation cron

Following `newAndroidMDMDeviceReconcilerSchedule` (`cmd/fleet/cron.go:2515`) and registered in
`cmd/fleet/cron_registration.go`:

1. Verify the credential; mark the integration invalid and stop if it fails.
2. `customers.list` — confirm the expected customer account is still reachable.
3. `customers.dpcs.list` — refresh the Android Device Policy resource path.
4. `customers.configurations.list` — diff against Fleet's rows. Recreate configurations deleted out of
   band, re-patch drifted `dpcExtras`, reconcile which configuration Google considers default.
5. Rotate tokens approaching expiry.
6. `customers.devices.list` (paginated) — upsert pending device records, reconcile per-device assignments
   against team intent.
7. Detect and surface `TosError`. Cheaper pre-check: `Company.termsStatus` is an output-only field on the
   objects `customers.list` already returns (`TERMS_STATUS_ACCEPTED` / `_NOT_ACCEPTED` / `_STALE`), so the
   cron can flag a ToS problem without waiting for a 403.

Suggested interval 30 minutes, tunable. It must be bounded and paginated: the existing Android device
reconciler needed its pagination loop bounded (commit `8a65ecf20bf`) and the same failure mode applies.
Google's per-project quotas for this API are not stated in the documentation reviewed and should be
confirmed before choosing an interval for large fleets.

## Fleet modifications

### New package

`server/mdm/android/zerotouch/`, interface-first so it can be mocked, as `androidmgmt.Client` is:

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

Plus a `mock/` package in the style of `server/mdm/android/mock/client.go`, and an
`IsTermsOfServiceError(err error) bool` helper in this new package, mirroring how `IsNotModifiedError` and
`IsBadRequestError` sit beside the client interface in `androidmgmt/client.go`.

`server/mdm/android/arch_test.go` enforces the bounded-context import rules; new code must satisfy it.

### Database

Two new tables. Timestamps at `TIMESTAMP(6)` precision per Fleet's MySQL conventions.

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
  `management_mode` VARCHAR(40) NOT NULL,  -- store the AMAPI allowPersonalUsage value verbatim,
                                           -- not the GitOps alias; map at the YAML boundary
  `company_name` VARCHAR(255) NOT NULL DEFAULT '',
  `contact_email` VARCHAR(255) NOT NULL DEFAULT '',
  `contact_phone` VARCHAR(64) NOT NULL DEFAULT '',
  `custom_message` TEXT,
  -- Token state (the token value itself lives in mdm_config_assets)
  `enrollment_token_name` VARCHAR(255) NOT NULL DEFAULT '',
  `enrollment_token_expires_at` TIMESTAMP(6) NULL DEFAULT NULL,
  `last_synced_at` TIMESTAMP(6) NULL DEFAULT NULL,
  `sync_error` TEXT,
  `created_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_android_zt_global_or_team_id` (`global_or_team_id`),
  KEY `fk_android_zt_team_id` (`team_id`),
  -- SET NULL, not CASCADE: the row must survive team deletion so the cron knows Fleet created the
  -- Google-side configuration and may therefore delete it. See the failure-mode table.
  CONSTRAINT `fk_android_zt_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

The nullable `team_id` plus non-null `global_or_team_id` pattern is borrowed from
`android_app_configurations`, which uses it to give "No team" a real value (`0`) in a unique key. Note the
difference: that table's unique key is composite, `(global_or_team_id, application_id)`, because a team has
many app configurations. Here a team has exactly one zero-touch configuration, so the unique key is on
`global_or_team_id` alone.

```sql
CREATE TABLE `android_zero_touch_devices` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `google_device_id` VARCHAR(64) NOT NULL,
  `serial_number` VARCHAR(255) NOT NULL DEFAULT '',
  `imei` VARCHAR(32) NOT NULL DEFAULT '',
  `meid` VARCHAR(32) NOT NULL DEFAULT '',
  `manufacturer` VARCHAR(255) NOT NULL DEFAULT '',
  `model` VARCHAR(255) NOT NULL DEFAULT '',
  `zt_configuration_id` INT UNSIGNED DEFAULT NULL,  -- FK to android_zero_touch_configurations.id;
                                                   -- deliberately NOT the Google configuration_id above
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

Create with `make migration name=AddAndroidZeroTouchConfigurations`. If this lands alongside other work
the migration timestamp will likely need bumping before merge.

### Secrets

New `MDMAssetName` values in `server/fleet/mdm.go`, encrypted in `mdm_config_assets`:

- `android_zero_touch_service_account` — uploaded service-account JSON key (Option A).
- `android_zero_touch_oauth_refresh_token` — refresh token (Option B).
- `android_zero_touch_enrollment_token` — the current long-lived token per configuration. Storing it as an
  asset keeps it out of ordinary query paths and gets encryption for free, but the unique key is
  `(name, deletion_uuid)` — effectively name-scoped for live rows — so a per-team naming scheme is needed.
  Check how the ABM and VPP token tables solved per-entity secret storage before choosing.

### Service layer and API

New endpoints in `server/mdm/android/service/handler.go`, with request/response structs carrying
`Err error` per Fleet's API conventions. Paths below use `:name` for readability; Fleet registers with
gorilla-mux brace syntax — `{token}` at `handler.go:35`, `{id:[0-9]+}` elsewhere — so the real
registrations would use `{team_id:[0-9]+}` and `{id:[0-9]+}`.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch` | Status: linked customer, credential validity, ToS state, last sync, errors |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/credentials` | Upload service-account key, or begin OAuth |
| `DELETE` | `/api/_version_/fleet/android_enterprise/zero_touch/credentials` | Disconnect |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations` | List Fleet's configurations |
| `PUT` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id` | Create or update a team's configuration |
| `DELETE` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id` | Remove |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/configurations/:team_id/rotate_token` | Force rotation |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/dpc_extras/:team_id` | Phase 1: the copy-paste JSON |
| `GET` | `/api/_version_/fleet/android_enterprise/zero_touch/devices` | Paginated device list with team and enrollment status |
| `POST` | `/api/_version_/fleet/android_enterprise/zero_touch/devices/:id/team` | Move a device to a team |

Authorize first in the service method; validate inputs there too, accumulating into one
`InvalidArgumentError`; wrap errors with `ctxerr.Wrap`; return client errors as 4xx without logging them
as server errors. Google-side validation failures — a rejected `contactEmail`, a malformed phone number —
are client errors, not 500s. Validate both fields locally as well, so the admin gets a field-level error
instead of a round-trip failure.

**Tier gating needs a product decision.** Zero-touch is a company-owned-fleet feature, which argues for
Premium, consistent with ABM/ADE. The mechanism is less open than it first appears: `server/mdm/android/`
itself contains no license checks, but Android tier gating already exists outside the bounded context —
`fleet.AndroidPremiumOnlyJSONKeys` (`server/fleet/android.go:59`, enforced at `:87`) gates `systemUpdate`
in Android profiles, and `ee/server/service/` does hold Android code (`service.go:47` carries
`androidModule android.Service`; `hosts.go` dispatches Android lock/wipe; `mdm.go` has
`clearPasscodeAndroid`). Gating outside the bounded context, with the module injected into the EE service,
*is* the pattern. Note the counter-precedent that Android COBO wipe was deliberately moved to Free
(commit `fa7d9282359`), so the product call is genuinely open even though the mechanism is not.

### Enrollment path changes

- `CreateEnrollmentToken` gains a zero-touch variant: explicit long `duration`, `oneTimeOnly: false`,
  mode-specific `allowPersonalUsage`, v2 `additionalData`. Keep the existing signature for the QR path and
  factor a private helper taking an options struct rather than adding a second positional bool — the
  current `fullyManaged bool` is already at the limit of readability.
- `enrollHost` and `addNewHost` learn the v2 payload, prefer `team_id` for zero-touch tokens, and stop
  treating a secret mismatch as fatal in that case, while preserving `GetAndroidDeviceLastTeamID`
  precedence.
- Enroll-secret mutation paths re-issue affected zero-touch tokens.

### Activities

New types, documented in `docs/Contributing/reference/audit-logs.md`: `enabled_android_zero_touch`,
`disabled_android_zero_touch`, `created_android_zero_touch_configuration`,
`edited_android_zero_touch_configuration`, `deleted_android_zero_touch_configuration`,
`rotated_android_zero_touch_enrollment_token`, `changed_android_zero_touch_device_team`.

### GitOps

**There is no top-level `mdm:` key in Fleet GitOps.** Valid top-level keys are enumerated at
`pkg/spec/gitops.go:522`: `name`, `settings`, `org_settings`, `agent_options`, `controls`, `policies`,
`reports`, `software`, `labels`, `custom_host_vitals`. Existing Android settings live under
`controls.android_settings`, and MDM connector configuration under `org_settings.mdm`, which is
`default.yml`-only — `pkg/spec/gitops.go:536` rejects `org_settings` in a file that has `name`.

That forces a choice, and it should be made deliberately rather than by picking whichever shape looks
tidiest:

- **Connector-shaped** — put it under `org_settings.mdm`, following `apple_business`, which is documented as
  configurable only for "All fleets." Global credential plus per-team configuration then cannot both live in
  YAML, so per-team settings would need another home.
- **Controls-shaped** — put per-team settings under `controls.android_settings.zero_touch`, which works in
  `default.yml`, `teams/*.yml`, and `teams/no-team.yml`, with the credential handled separately under
  `org_settings.mdm`.

Controls-shaped is the better fit, because the per-team configuration is the part that benefits from being
declarative. Sketch:

```yaml
controls:
  android_settings:
    zero_touch:
      management_mode: fully_managed   # maps to allowPersonalUsage
      company_name: "Acme Corp"
      contact_email: "it@acme.example.com"
      contact_phone: "+1 555 0100"
      custom_message: "Contact IT if setup does not complete."
      default: true                    # at most one team may set this
      forced_reset_time: 2h
```

Per `CLAUDE.md`'s GitOps notes, two cases need explicit tests: **removal** — deleting the key must delete
the Google-side configuration, the classic `apply`-inherited bug — and each placement the chosen shape
actually supports. Validate that at most one team claims `default: true`, within the incoming payload rather
than against the DB.

### Frontend

- **Settings → Integrations → MDM → Android**: extend `AndroidMdmPage.tsx` with a zero-touch section —
  connect/disconnect credentials, linked customer account, ToS prompt, last sync, per-team configuration
  editor, and the Phase 1 copy-paste extras view. Reflect status in `AndroidMdmCard.tsx`.
- **Add hosts modal**: `AndroidPanel.tsx` gains a zero-touch tab noting that eligible devices need no QR
  code, and stating the reseller-registration prerequisite — this is where that confusion surfaces.
- **Hosts list**: show pending zero-touch devices, matching ADE "awaiting enrollment" hosts.
- Free/Primo gating per the `tier-modes` conventions, if gated.
- Command palette entries per the `command-palette` conventions.

### fleetctl

`fleetctl get android-zero-touch` or equivalent, plus whatever `fleetctl gitops` needs to round-trip the
new keys.

### Documentation

- New guide in `articles/`, following the guide format. Lead with the reseller prerequisite and the
  service-account linking wait.
- Extend `articles/android-mdm-setup.md`.
- REST API reference for the new endpoints.
- Architecture doc under `docs/Contributing/architecture/mdm/`, parallel to
  `automated-device-enrollment.md`.
- Update `docs/Contributing/product-groups/mdm/android-mdm.md`.

## Phased delivery

**Phase 1 — manual configuration, no new Google API.** Long-lived reusable tokens; the v2 team-identity
fix; re-issue on enroll-secret rotation; DPC extras generation and display; docs.

**Phase 1a — answer the iframe question.** Days, not weeks. Determine whether the zero-touch iframe can
carry per-team provisioning extras. A yes reshapes Phase 2 into "wire up `enterprises.webTokens.create` and
render an iframe" and deletes the credential workstream; a no confirms the provisioning-API path below.

**Phase 2 — customer API integration** (only if the iframe cannot do per-team). Credential upload and
validation; client package and mock; configuration table; create/patch/delete from Fleet; reconciliation
cron; activities; ToS handling; UI; the turn-off guard **and the `cleanup_android_enterprise` cron guard**.

**Phase 3 — device-level management.** Device list sync, pending hosts, per-device override, host
correlation, hosts-list integration.

**Phase 4 — parity.** GitOps, fleetctl, dedicated-device support, optional provisioning extras, and — only
if `googleAuthenticationOptions` proves insufficient — `signinDetail` authentication.

Phase 1 is independently shippable and worth doing even if Phases 2–4 never are.

## Edge cases and failure modes

| Scenario | Consequence | Handling |
| --- | --- | --- |
| Enroll secret rotated while a zero-touch token is live | Devices enroll into AMAPI but never become Fleet hosts — silent, delayed | v2 `team_id` fix plus re-issue on rotation. **Highest-severity item here** |
| Admin edits or deletes the configuration in Google's portal | Fleet's view is wrong; devices provision with stale extras or not at all | Cron detects drift, restores, records `sync_error` |
| Team deleted in Fleet | Orphaned Google configuration | Needs a tombstone, or `ON DELETE SET NULL` rather than `CASCADE`. **`CASCADE` is wrong**: once the row is gone Fleet cannot tell it created the Google-side configuration, colliding with the "never delete what Fleet did not create" rule below |
| Team was the zero-touch default | New purchases have no default | Refuse deletion, or reassign the default and warn |
| Token expires | All zero-touch provisioning fails at once, fleet-wide | Rotate 30 days early; treat approaching expiry as a monitored condition |
| Google ToS updated | Every API call 403s mid-life | Detect `TosError` with `X-GOOG-API-FORMAT-VERSION: 2`; surface an actionable banner |
| Credential revoked or key deleted | Cron fails silently | Mark integration invalid, surface in UI, stop retrying hot |
| Android MDM turned off while zero-touch configurations exist | `enterprises.delete` cascades and invalidates the embedded token; each device breaks at its *next* reset, long after the action | Block or hard-warn on turn-off; offer pre-offboarding cleanup. **Phase 2 release blocker** |
| Customer migrates to another EMM | Factory reset needed per device regardless of EMM; claims unaffected | Document repoint-then-reset order; do not delete the Fleet enterprise first |
| Two Fleet instances share a customer account | Configurations fight over `isDefault` | Detect unknown configurations; never delete ones Fleet did not create |
| Device factory reset | Re-provisions via zero-touch | `GetAndroidDeviceLastTeamID` restores the last-known team; keep that precedence |
| Device unclaimed at the reseller | Zero-touch stops applying; re-registration needs the reseller | Reconcile removals; mark the pending device gone rather than deleting history |
| Device never boots | Pending host lingers | Show claim age; allow dismissal |
| Configuration applied to a device already in service | Nothing until the device's next factory reset — so an admin may expect an immediate change and see none | State in the UI that assignment takes effect at next reset. Google is silent on edge states here; verify |
| No network during setup | Zero-touch skipped, device boots unmanaged, then self-resets on first connection to Google after a one-hour warning | Expose `forcedResetTime`; document all three reset triggers |
| Serial number missing, substituted, or mismatched | Pending device never correlates, or a duplicate host appears | Multi-key correlation with normalization; prefer no match over a wrong one |
| `additionalData` exceeds 1024 chars | Token creation fails | Validate length before the API call |
| Non-GMS or pre-8.0 device | Zero-touch does not apply | Document; nothing Fleet can do |

## Testing plan

- **Unit** — table-driven `t.Run` subtests over the client interface via the new mock. Cover configuration
  diffing, default-configuration transitions, rotation ordering (patch before delete), `additionalData`
  v1/v2 parsing, and serial normalization.
- **Behaviour to pin down against the real API before relying on it** — whether
  `customers.devices:removeConfiguration` leaves the device with no configuration or falls back to the
  account default. Google documents only "Removes a configuration from device," and the answer determines
  whether removing a team's configuration silently re-homes its devices into the default team.
- **Enroll-secret rotation regression** — create a zero-touch token, rotate the team's enroll secret,
  deliver a synthetic `ENROLLMENT` notification, assert the host lands in the right team. This test is the
  reason to do the work.
- **Integration** — extend `server/mdm/android/tests/` and the mock at `cmd/android-amapi-mock/` with a
  provisioning-API surface, so the cron and endpoints run without Google.
- **Multi-team** — per `CLAUDE.md`: "No team", multiple teams, and the default-team transition.
- **Multi-host** — at least two zero-touch devices from different manufacturers in manual QA, per Fleet's
  multi-host requirement. OEM differences in serial/IMEI reporting are the likeliest correlation bugs and
  one device will not reveal them.
- **GitOps** — add, edit, and *remove* the key in all three placements.
- **Manual QA prerequisites** — a zero-touch customer test account (obtainable from Google with a
  corporate email) and at least one reseller-claimed device. Google's rule that the DPC must be published
  on Play is satisfied automatically, since Fleet uses Android Device Policy.
- **Frontend** — Jest coverage for the new settings section and Add-hosts tab.

## Security considerations

- The enrollment token in `dpcExtras` is a bearer credential for joining the AMAPI enterprise. Its blast
  radius is bounded — it enrolls devices, it cannot read or write Fleet data — but a leaked reusable token
  lets an attacker enroll arbitrary devices into a team, polluting inventory and potentially pulling
  team-scoped configuration profiles. Since Android profiles support variables and certificate templates,
  that warrants a threat-model review.
- Never place the Fleet enroll secret, IdP credentials, or any Fleet API token in `dpcExtras`. Google gives
  the reason, which is stronger than a general secrets-hygiene argument: "A masquerading device may be able
  to use a false IMEI or serial number to read the configuration." Configuration contents are effectively
  readable by anyone who can guess a claimed device's hardware ID.
- Service-account keys and refresh tokens go in `mdm_config_assets`, never `app_config_json`, and must be
  redacted from logs and API responses. `ProxyClient`'s logger already shows the redaction pattern for
  `Authorization` headers (`proxy_client.go:46`).
- Reject the shared-service-account multi-tenant proxy model.
- Team-scope authorization on all new endpoints using the double-authorize pattern.
- `customers.devices:unclaim` is reseller-only to reverse. Either do not expose it or require typed
  confirmation.
- Run a `fleet-security-auditor` pass before merge.

## Open questions

Claims in this document come from Google's published reference documentation or from Fleet's source, except
where the text says otherwise. Where something is inference, contested between Google's own documents, or
needs hardware to settle, it is flagged in place in the section it affects — see in particular the
`dpcExtras` payload, `signinDetail` with zero-touch, service-account linking, and post-offboarding device
state. The questions below are the ones that block decisions.

1. **Can the zero-touch iframe carry per-team provisioning extras?** Answer this first — it is cheap, and a
   yes removes the whole credential problem along with open question 2. Specifically: can more than one
   configuration exist per enterprise via the iframe, and can `dpcExtras` differ per team across renderings?
2. **Credential model, if the iframe cannot do per-team** — Option A or B? Needs Google's Android Enterprise
   partner team on two points: whether customer-account service-account linking is still a Google Form, and
   whether `androidworkzerotouchemm` is a sensitive scope requiring app verification.
3. **Proxy divergence** — under Options A and B, is "provisioning API goes direct to Google" acceptable to
   Fleet Cloud, given every other Android call is proxied? Moot under the iframe.
4. **Does `googleAuthenticationOptions` satisfy the end-user-authentication requirement?** If yes,
   `signinDetail` drops out of scope entirely and a library bump replaces a multi-week sub-project.
5. **Tier gating** — Premium or Free? The mechanism already exists (see the service-layer section); this is
   purely a product call, weighed against the deliberate decision to make Android COBO wipe Free.
6. **End-user authentication** — is user-attributed zero-touch a requirement at all? If it is, question 4
   determines whether it costs a library bump or a multi-week sub-project. Settle this before Phase 1 locks
   the token model.
7. **Pending hosts** — ship in Phase 3 or defer until real device data exists?
8. **Multiple customer accounts** — can one organization hold several (per reseller, per region)?
   `customers.list` is paginated and returns every account the caller belongs to; Google's own quickstarts
   take the first element. Fleet must not do that silently.
9. **Dedicated devices** — COSU in the first release or deferred? Affects whether management mode must be
   per-configuration rather than per-team.

## Effort estimate

Rough, assuming one backend engineer plus frontend support, and excluding the credential decision and
obtaining a Google test account — both on the critical path with external latency.

| Phase | Backend | Frontend | Notes |
| --- | --- | --- | --- |
| 1 — manual configuration | 1–2 weeks | 2–3 days | The `additionalData` fix is most of the risk |
| 1a — iframe question | 2–3 days | — | Gates the shape of Phase 2 |
| 2 — customer API integration | 3–4 weeks | 1–1.5 weeks | Client, mock, migration, cron, endpoints, turn-off and cron guards. Substantially smaller, possibly near-zero, if the iframe suffices |
| 3 — device-level management | 2–3 weeks | 1 week | Correlation is the uncertain part |
| 4 — parity | 1–2 weeks | 2–3 days | Excludes `signinDetail` |

`signinDetail` authentication, if genuinely required, adds 3–4 weeks and should be scoped separately — but
check `googleAuthenticationOptions` first, which is a library bump plus a field.

## References

### Google — zero-touch enrollment

- [Overview](https://developers.google.com/zero-touch/guides/overview)
- [How it works (customers)](https://developers.google.com/zero-touch/guides/customer/how-it-works)
- [EMM integration guide](https://developers.google.com/zero-touch/guides/customer/emm)
- [Customer API reference](https://developers.google.com/zero-touch/reference/customer/rest)
- [`customers.configurations`](https://developers.google.com/zero-touch/reference/customer/rest/v1/customers.configurations)
- [`customers.devices`](https://developers.google.com/zero-touch/reference/customer/rest/v1/customers.devices)
- [Authorization — reseller API](https://developers.google.com/zero-touch/guides/auth) (documents the
  `androidworkprovisioning` scope; the customer-API guides live under `/zero-touch/guides/customer/`)
- [Zero-touch iframe for AMAPI](https://developers.google.com/android/management/zero-touch-iframe)
- [Service accounts for customers](https://developers.google.com/zero-touch/guides/customer/service-accounts)
- [Python quickstart with a service account](https://developers.google.com/zero-touch/guides/customer/quickstart/python-service-account)
- [Zero-touch enrollment for IT admins](https://support.google.com/work/android/answer/7514005)
- [Zero-touch customer portal](https://enterprise.google.com/android/zero-touch/customers)
- [Sample Colab notebooks](https://github.com/googlesamples/zero-touch-enrollment-colabs)

### Google — Android Management API

- [Enroll and provision a device](https://developers.google.com/android/management/provision-device)
- [`enterprises.enrollmentTokens`](https://developers.google.com/android/management/reference/rest/v1/enterprises.enrollmentTokens)
- [`AllowPersonalUsage`](https://developers.google.com/android/management/reference/rest/v1/AllowPersonalUsage)
- [`enterprises.delete`](https://developers.google.com/android/management/reference/rest/v1/enterprises/delete)
- [Migrate existing devices to AMAPI](https://developers.google.com/android/management/dpc-migration)
- [Notifications (Pub/Sub)](https://developers.google.com/android/management/notifications)

### Fleet

- `server/mdm/android/README.md` — bounded-context rationale
- `server/mdm/android/service/service.go:549` — `CreateEnrollmentToken`
- `server/mdm/android/service/service.go:424` — `DeleteEnterprise`
- `server/mdm/android/service/pubsub.go:599` — `enrollHost`
- `server/mdm/android/service/pubsub.go:893` — `addNewHost`
- `server/mdm/android/service/androidmgmt/client.go` — AMAPI client interface
- `cmd/fleet/cron.go:2515` — Android device reconciler cron pattern
- `docs/Contributing/architecture/mdm/automated-device-enrollment.md` — the Apple analogue
- `docs/Contributing/product-groups/mdm/android-mdm.md`
