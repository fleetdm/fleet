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

Four workstreams follow. **One:** integrate the Android Device Provisioning Partner API
(`androiddeviceprovisioning.googleapis.com`), customer surface — already in Fleet's module graph, so no
new dependency. **Two:** choose a credential model for it, the largest open question. Google offers a
service-account path (fits an automated cron, but linking it to a customer account currently requires a
Google Form and email confirmation) and an interactive OAuth path (self-service, but bound to one admin's
Google account with probable scope verification). **Three:** build the Fleet side — long-lived reusable
tokens, a configuration per team, a reconciliation cron, pending-host records, activities, GitOps keys,
REST endpoints, UI. **Four:** change how Fleet identifies the target team at enrollment. Fleet embeds a
mutable enroll secret in the token's `additionalData`; rotating a team's enroll secret then silently
breaks every long-lived zero-touch token. That is a correctness prerequisite, not an enhancement.

Phase 1 ships usable zero-touch with **no** Google API integration: Fleet emits the DPC extras JSON and
the admin pastes it into Google's portal once per team. It is small, unblocks customers immediately, and
proves the token and team-mapping model before the credential decision is made.

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
| Windows | Windows Autopilot | Not supported. Separate effort, unrelated APIs |
| ChromeOS | Chrome zero-touch enrollment (same provisioning API, `deviceType` `CHROME_OS`) | Out of scope; Fleet has no ChromeOS MDM |

Zero-touch supports three management modes, differing only in the `allowPersonalUsage` value on the
enrollment token:

- **Fully managed (COBO)** — `PERSONAL_USAGE_DISALLOWED`.
- **Company-owned with a work profile (COPE)** — `PERSONAL_USAGE_ALLOWED`. Android 8+, though Android 11
  reworked COPE to treat the personal side closer to BYOD.
- **Dedicated / kiosk (COSU)** — `PERSONAL_USAGE_DISALLOWED_USERLESS`, which Google made mandatory for
  dedicated-device enrollment as of January 2025.

Zero-touch does **not** support work profiles on personally-owned devices. Google's provisioning-method
matrix lists zero-touch as supporting COPE, fully managed, and dedicated only. BYOD is therefore out of
scope by mechanism, not just by policy.

## How Android zero-touch enrollment works

### Actors

1. **Reseller** — an authorized zero-touch reseller. On purchase, the reseller registers each device's
   hardware identifiers (IMEI/MEID or serial number, plus manufacturer) and *claims* the device to the
   organization's customer account. **Only resellers can create device records** — not IT admins, not an
   EMM. Fleet can never add a device to zero-touch.
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
5. Android Device Policy receives `ACTION_PROVISION_MANAGED_DEVICE` with the extras from `dpcExtras`,
   reads the enrollment token, and enrolls into the AMAPI enterprise.
6. AMAPI publishes an `ENROLLMENT` notification to the enterprise's Pub/Sub topic. **Fleet re-enters here,
   on the code path it already has.**

### When provisioning triggers

Zero-touch is evaluated **only during the out-of-box setup wizard** — on first boot or after a factory
reset. It is not a push mechanism and cannot provision a device that is already set up and running. Three
consequences shape the design:

- **It re-triggers on every factory reset, for as long as the claim exists.** The end user cannot bypass
  or defer it. This is the security property that makes zero-touch valuable for company-owned hardware,
  and the reason a broken configuration is a fleet-wide outage rather than a per-device annoyance.
- **Applying a configuration to a device already in service triggers a factory reset.** Google's admin
  documentation states that a device lacking connectivity during setup skips zero-touch and boots
  unmanaged, then "resets itself after the first connection to Google servers," warning the user one hour
  ahead. So `customers.devices:applyConfiguration` can destroy end-user data — which the method name does
  not suggest. Fleet must treat it, and changes to the default configuration, as destructive actions
  requiring explicit confirmation.
- **Two reset mechanisms exist and are easily conflated.** `forcedResetTime` (0–6 hours, default 2) is the
  in-setup timeout when the wizard cannot reach Google. The self-reset above fires after setup completed
  unmanaged. Only the first is configurable.

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
2. Verifies the enroll secret (`svc.ds.VerifyEnrollSecret`), which determines the target team.
3. Optionally requires and validates an IdP account UUID for end-user authentication.
4. Marshals `{enroll_secret, idp_uuid}` into the token's `additionalData`.
5. Sets `allowPersonalUsage` from a `fully_managed` query parameter — `PERSONAL_USAGE_DISALLOWED` when
   true, `PERSONAL_USAGE_ALLOWED` otherwise (`service.go:608`).
6. Sets `oneTimeOnly: true` and leaves `duration` unset, so **the token expires in one hour and works
   once**.
7. Returns the token value, an `https://enterprise.google.com/android/enroll?et=<token>` URL, and a QR
   code.

The frontend surface is `frontend/components/AddHostsModal/PlatformWrapper/AndroidPanel/AndroidPanel.tsx`,
which offers a work-profile / fully-managed radio and builds a Fleet enrollment URL.

### Enrollment completion

`Service.enrollHost` (`server/mdm/android/service/pubsub.go:599`) handles the `ENROLLMENT` notification.
It unmarshals `device.EnrollmentTokenData` back into `{enroll_secret, idp_uuid}`, re-verifies the enroll
secret, resolves the team, and creates or updates the host. `addNewHost` (`pubsub.go:893`) is the
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
| `customers.devices:applyConfiguration` | Assign a device to a specific team's configuration. Destructive — see above. |
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
- `enterprises.enrollmentTokens.list` — lists active, unexpired tokens; useful for drift detection.
- `EnrollmentToken.user` is deprecated and ignored. Do not use it.

### 3. Google Cloud Pub/Sub (existing)

Unchanged. Zero-touch devices produce the same `ENROLLMENT` notification as QR-code devices.

### 4. Optional: AMAPI `signinDetail`, only if end-user IdP auth is required

Zero-touch hands the device a plain enrollment token with no user interaction, so Fleet's BYOD IdP flow
cannot apply — it depends on a browser cookie set before the token is requested (`mdm.BYODIdpCookieName`,
`service.go:508`).

AMAPI's sign-in URL enrollment is the mechanism for authenticating a user during provisioning. An
enterprise can hold any number of `SigninDetail` entries, keyed by (`signinUrl`, `allowPersonalUsage`,
`tokenTag`), each yielding a server-generated read-only `signinEnrollmentToken`. The sign-in endpoint must
finish by redirecting to `https://enterprise.google.com/android/enroll?et=<token>` on success or
`https://enterprise.google.com/android/enroll/invalid` on failure.

**Whether a `signinEnrollmentToken` can be placed in `dpcExtras` to drive this from zero-touch is not
documented and must be tested.** Google's provisioning-method matrix treats zero-touch and sign-in URL as
separate methods and does not describe combining them.

Either way this is a substantial sub-project: Fleet would serve an unauthenticated, device-facing web flow
that mints enrollment tokens. **Recommendation: exclude from initial scope.** Company-owned devices are
typically assigned to a person out of band. Revisit only if customers ask for user-attributed zero-touch.

## Credential and authorization options

This decision gates Phase 2 and should be settled before implementation starts.

### Option A — Service account key uploaded by the admin (recommended)

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

### Recommendation

Ship **C as Phase 1**, then **A as Phase 2**, keeping **B** as an alternate for customers blocked on
service-account linking. C is not throwaway — it forces the long-lived token model, the team-mapping fix,
and extras generation, all reused verbatim by Phase 2, and it decouples customer value from the credential
decision.

### The Fleet Cloud proxy question

Fleet's AMAPI traffic defaults through `https://fleetdm.com/api/android/`, with Fleet's hosted project
owning the credentials. Zero-touch cannot follow that model:

- The customer's zero-touch account was established by *their* reseller. Fleet's hosted service account
  has no relationship to it and cannot gain one without the customer linking it.
- A single shared service account linked to many customer accounts would make `customers.list` return
  every linked customer to every caller, putting Fleet's proxy in charge of tenant isolation on a
  credential with no per-tenant scoping. **Reject this.**

Therefore zero-touch credentials are **per-Fleet-instance and customer-supplied**, and zero-touch calls go
**direct to Google**, regardless of how AMAPI traffic is routed. This diverges deliberately from the rest
of Android MDM and should be documented in the code, because it will surprise readers.

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

`additionalData` carries `{"enroll_secret": "...", "idp_uuid": "..."}`, and `addNewHost` (`pubsub.go:908`)
hard-fails when the secret does not verify:

```go
enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, enrollmentTokenRequest.EnrollSecret)
if err != nil {
    return ctxerr.Wrap(ctx, err, "verifying enroll secret")
}
```

Fine for a one-hour token. For a token embedded in a configuration for a year it is a latent outage: **once
an admin rotates that team's enroll secret, every subsequent zero-touch device enrolls into AMAPI and then
fails to become a Fleet host** — managed by Google, invisible in Fleet, visible only as a repeating
Pub/Sub error.

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
for little gain: the token's only power is enrolling a device into the enterprise, and it sits in a store
readable only by zero-touch portal admins.

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
- Normalize case, leading zeros, and whitespace on both sides; `fleet.Preprocess`
  (`server/fleet/utils.go:65`) exists for this.
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
`service.go:442`) runs when an admin turns off Android MDM. If zero-touch configurations exist, deleting
the enterprise invalidates the token still embedded in them, so every subsequent factory reset hits a dead
token and cannot complete provisioning. **Turning off Android MDM therefore breaks provisioning for every
zero-touch-claimed device, with damage surfacing only at each device's next reset.**

Mitigations:

- Turn-off must detect existing zero-touch configurations and block or hard-warn, listing affected teams
  and device counts. Treat as a Phase 2 release blocker.
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
7. Detect and surface `TosError`.

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
  `management_mode` VARCHAR(40) NOT NULL,  -- allowPersonalUsage value
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
  CONSTRAINT `fk_android_zt_team_id` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`) ON DELETE CASCADE
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

Create with `make migration name=AddAndroidZeroTouchConfigurations`. If this lands alongside other work
the migration timestamp will likely need bumping before merge.

### Secrets

New `MDMAssetName` values in `server/fleet/mdm.go`, encrypted in `mdm_config_assets`:

- `android_zero_touch_service_account` — uploaded service-account JSON key (Option A).
- `android_zero_touch_oauth_refresh_token` — refresh token (Option B).
- `android_zero_touch_enrollment_token` — the current long-lived token per configuration. Storing it as an
  asset keeps it out of ordinary query paths and gets encryption for free, but assets are keyed by name
  alone, so a per-team scheme is needed. Check how the ABM and VPP token tables solved per-entity secret
  storage before choosing.

### Service layer and API

New endpoints in `server/mdm/android/service/handler.go`, with request/response structs carrying
`Err error` per Fleet's API conventions:

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

**Tier gating needs a decision and a mechanism.** Zero-touch is a company-owned-fleet feature, which
argues for Premium, consistent with ABM/ADE. But `server/mdm/android/` contains no license or tier checks
today, and the bounded context has no Android code in `ee/server/service/`, so there is no established
pattern to follow — unlike most Fleet features. Note also that Android COBO wipe was moved to Free
(commit `fa7d9282359`). Both the product decision and the gating mechanism are open.

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

Per `CLAUDE.md`'s GitOps notes, two cases need explicit tests: **removal** — deleting the key must delete
the Google-side configuration, the classic `apply`-inherited bug — and **all three placements**
(`default.yml`, `teams/*.yml`, `teams/no-team.yml`). Validate that at most one team claims
`default: true`, within the incoming payload rather than against the DB.

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

**Phase 2 — customer API integration.** Credential upload and validation; client package and mock;
configuration table; create/patch/delete from Fleet; reconciliation cron; activities; ToS handling; UI;
the turn-off guard.

**Phase 3 — device-level management.** Device list sync, pending hosts, per-device override, host
correlation, hosts-list integration.

**Phase 4 — parity.** GitOps, fleetctl, dedicated-device support, optional provisioning extras, and — only
if demanded — `signinDetail` authentication.

Phase 1 is independently shippable and worth doing even if Phases 2–4 never are.

## Edge cases and failure modes

| Scenario | Consequence | Handling |
| --- | --- | --- |
| Enroll secret rotated while a zero-touch token is live | Devices enroll into AMAPI but never become Fleet hosts — silent, delayed | v2 `team_id` fix plus re-issue on rotation. **Highest-severity item here** |
| Admin edits or deletes the configuration in Google's portal | Fleet's view is wrong; devices provision with stale extras or not at all | Cron detects drift, restores, records `sync_error` |
| Team deleted in Fleet | Orphaned Google configuration | `ON DELETE CASCADE` plus a cron pass to delete the Google-side object |
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
| Configuration applied to a device already in service | **Factory reset, destroying end-user data**, after a one-hour warning on next contact with Google | Treat `applyConfiguration` and default changes as destructive: warning plus confirmation |
| No network during setup | Zero-touch skipped, device boots unmanaged, then self-resets on first connection to Google | Expose `forcedResetTime`; document both reset mechanisms |
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
- Never place the Fleet enroll secret, IdP credentials, or any Fleet API token in `dpcExtras`.
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

1. **Credential model** — Option A or B? Blocks Phase 2. Needs Google's Android Enterprise partner team on
   two points: whether customer-account service-account linking is still a Google Form, and whether
   `androidworkzerotouchemm` is a sensitive scope requiring app verification.
2. **Proxy divergence** — is "zero-touch goes direct to Google" acceptable to Fleet Cloud, given every
   other Android call is proxied?
3. **Tier gating** — Premium or Free, and by what mechanism, given the Android bounded context has none?
4. **End-user authentication** — is user-attributed zero-touch a requirement? If so, specify
   `signinDetail` before Phase 1 locks the token model.
5. **Pending hosts** — ship in Phase 3 or defer until real device data exists?
6. **Multiple customer accounts** — can one organization hold several (per reseller, per region)?
   `customers.list` is paginated and returns every account the caller belongs to; Google's own quickstarts
   take the first element. Fleet must not do that silently.
7. **Dedicated devices** — COSU in the first release or deferred? Affects whether management mode must be
   per-configuration rather than per-team.

## Effort estimate

Rough, assuming one backend engineer plus frontend support, and excluding the credential decision and
obtaining a Google test account — both on the critical path with external latency.

| Phase | Backend | Frontend | Notes |
| --- | --- | --- | --- |
| 1 — manual configuration | 1–2 weeks | 2–3 days | The `additionalData` fix is most of the risk |
| 2 — customer API integration | 3–4 weeks | 1–1.5 weeks | Client, mock, migration, cron, endpoints, turn-off guard |
| 3 — device-level management | 2–3 weeks | 1 week | Correlation is the uncertain part |
| 4 — parity | 1–2 weeks | 2–3 days | Excludes `signinDetail` |

`signinDetail` authentication, if required, adds 3–4 weeks and should be scoped separately.

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
