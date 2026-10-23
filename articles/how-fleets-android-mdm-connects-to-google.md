# How Fleet's Android MDM connects to Google

Fleet manages Android devices with Google's [Android Management API](https://developers.google.com/android/management) (AMAPI). Your Fleet server doesn't call Google directly. It calls a proxy at `https://fleetdm.com/api/android/`, which holds Fleet's Google Cloud credentials and forwards each call to AMAPI.

This guide covers the traffic path, how each hop is authenticated, what data passes through fleetdm.com, what happens when a component goes down, and which network paths to allow. Facts come from Fleet's source code at commit [59e246b](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03), and each section links to the code it's based on.

## Summary

- Traffic from Fleet to Google goes through the fleetdm.com proxy.
- Traffic from Google to Fleet skips the proxy. Google Pub/Sub pushes device events straight to your Fleet server.
- Enrolled devices talk to Google, and to your Fleet server if the Fleet Android agent is installed. Devices never contact fleetdm.com.

> **Note:** While fleetdm.com is unreachable, new enrollments fail and admin actions like lock, wipe, and policy changes can't be sent. Devices that are already enrolled keep enforcing their last policy and keep reporting status.

## Components

| Component | Operated by | Role |
| --------- | ----------- | ---- |
| Fleet server | Customer (self-hosted) or Fleet (cloud) | Stores policies and device state. Starts every management call. Receives Pub/Sub events. |
| Android proxy (`fleetdm.com/api/android/*`) | Fleet | Authenticates the Fleet server, maps it to its Android Enterprise, and forwards the call to AMAPI with Fleet's Google service account. |
| Proxy database (`AndroidEnterprise` table) | Fleet | One row per Fleet server: server URL, Android Enterprise ID, per-server secret, license key, and Pub/Sub topic and subscription names. |
| Google AMAPI and Pub/Sub | Google | Owns the Android Enterprise, policies, and device records. Pushes device events. |
| Android device: Android Device Policy app, plus the optional Fleet agent (`com.fleetdm.agent`) | End user or IT | Enrolls with Google, applies the policy, and reports status to Google. The Fleet agent, when deployed, reports to the Fleet server. |

Source: Fleet server [proxy_client.go#L24](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/proxy_client.go#L24) (proxy endpoint) and [service.go#L152-L156](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L152-L156) (client selection). Proxy [routes.js#L1532-L1546](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/config/routes.js#L1532-L1546) (all 15 endpoints) and [AndroidEnterprise.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/models/AndroidEnterprise.js) (database row).

## Traffic flows

### One-time setup: connect Android Enterprise

An admin turns on Android MDM in Fleet. This runs once per Fleet server.

1. Fleet server to proxy: `POST /v1/signupUrls`, with the Fleet server URL in the `Origin` header.
2. Proxy to Google: `signupUrls.create`.
3. Google to Fleet server, through the proxy: the signup URL.
4. Admin's browser to Google: the admin signs up with Google.
5. Google to Fleet server: a redirect to `/api/v1/fleet/android_enterprise/connect/{token}?enterpriseToken=...`.
6. Fleet server to proxy: `POST /v1/enterprises` with the `enterpriseToken`, the license key, and the Pub/Sub push URL.
7. Proxy to Google: creates the Pub/Sub topic and subscription, then calls `enterprises.create`.
8. Proxy to Fleet server: the Enterprise ID and a new per-server secret.

During setup:

- The proxy creates a dedicated Pub/Sub topic and a push subscription that points at your Fleet server. Message retention is 24 hours, and the subscription never expires.
- The proxy generates a random 30-character secret for your Fleet server and returns it once. Fleet stores it in `mdm_config_assets`, encrypted with AES-GCM under the Fleet server private key.
- Fleet generates its own token (64 random bytes, base64url-encoded) and adds it to the Pub/Sub push URL. Fleet rejects any push whose token doesn't match.

To disconnect, an admin turns off Android MDM in **Settings** > **Integrations** > **MDM** > **Android**. This turns off Android MDM for the whole Fleet instance and unenrolls every Android device:

- Fleet calls `DELETE /v1/enterprises/{id}` on the proxy. The proxy deletes the enterprise in Google, then the Pub/Sub topic and subscription, then its own database row.
- If the Google delete fails, the proxy still removes its row and reports success, so the enterprise and its topic can be left behind in Google.
- Fleet clears its own enterprise record, marks all Android hosts as unenrolled, and deletes the stored secrets.

Source: [service.go#L319-L356](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L319-L356) (token generation, enterprise creation, secret storage), [handler.go#L36](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L36) (signup callback route), [apple_mdm.go#L6618](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/datastore/mysql/apple_mdm.go#L6618) (AES-GCM encryption), [create-android-enterprise.js#L74-L195](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L74-L195) (Pub/Sub setup, secret generation), [delete-one-android-enterprise.js#L57-L95](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/delete-one-android-enterprise.js#L57-L95) (teardown, Google error ignored), and [service.go#L460-L519](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L460-L519) (Fleet cleanup).

### Device enrollment

1. End user to Fleet server: opens the Fleet enrollment page (with optional IdP sign-in).
2. Fleet server to proxy: `POST /v1/enterprises/{id}/enrollmentTokens` with the bearer secret.
3. Proxy to Google: `enrollmentTokens.create`.
4. Google to Fleet server, through the proxy: a one-time token and QR code.
5. End user's device to Google: `enterprise.google.com/android/enroll?et=...`.
6. Google to Fleet server: the `ENROLLMENT` event, pushed directly by Pub/Sub with no proxy.

Enrollment tokens are one-time use and expire after 1 hour (the AMAPI default, since Fleet doesn't set a duration). Fleet creates a new token for each enrollment, so enrollment doesn't work without the proxy.

Source: [service.go#L612-L717](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L612-L717) (token request) and [create-android-enrollment-token.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enrollment-token.js).

### Ongoing management: Fleet to Google

Every admin action goes from Fleet to the proxy to AMAPI: policy and profile changes, app installs, web apps, lock, wipe, clear passcode, device lookups, unenroll, and Android Enterprise deletion. Policy changes include the Fleet agent's managed configuration, which carries the Fleet server URL, the host UUID, and an enroll secret.

Source: [proxy_client.go](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/proxy_client.go) (one method per call) and [android.go#L66-L71](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/android.go#L66-L71) (agent managed configuration fields).

### Device events: Google to Fleet (no proxy)

Google Pub/Sub pushes four event types straight to `{Fleet URL}/api/v1/fleet/android_enterprise/pubsub?token=...`: `ENROLLMENT`, `STATUS_REPORT`, `COMMAND`, and `USAGE_LOGS`. Fleet processes the first three. It drops `USAGE_LOGS`, test messages, and unknown types without side effects. The request body limit is 10 MiB.

Source: [handler.go#L18](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L18), [handler.go#L38](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L38), and [service.go#L336-L341](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L336-L341) (event types).

### Device to Google, and device to Fleet server (no proxy)

Enrolled devices get their policy from Google and report to Google. If the Fleet agent is deployed, it reports to the Fleet server URL from its managed configuration. Neither path touches fleetdm.com.

## How each hop is authenticated

Every hop authenticates the caller on each request.

| Hop | Transport | How the caller is authenticated |
| --- | --------- | ------------------------------- |
| Fleet to proxy (signup URL, create enterprise) | HTTPS | No bearer secret yet. The proxy keys the request on the `Origin` header and refuses a second enterprise for a URL that already has one (`409`). Creating an enterprise also needs a valid one-time `enterpriseToken` from the admin's Google signup. |
| Fleet to proxy (all other calls) | HTTPS | `Authorization: Bearer <per-server secret>`. The proxy looks up the enterprise by ID (or by `Origin` for list calls) and rejects the call if the secret doesn't match. One Fleet server can't reach another server's enterprise. |
| Proxy to Google | HTTPS | Fleet's Google service account, scoped to one Google Cloud project. The credentials never leave fleetdm.com. |
| Google Pub/Sub to Fleet | HTTPS push | A random token (64 bytes, base64url) in the push URL query string, compared against an encrypted copy in Fleet's database. This is the only check on this hop. The push subscription carries no Google-signed OIDC token, and Fleet doesn't check that the device belongs to its enterprise. The topic's IAM policy grants publish to Google's Android Device Policy service account (`android-cloud-policy@system.gserviceaccount.com`). |
| Device to Google | HTTPS | Managed by Google (Android Device Policy). |
| Fleet agent to Fleet server | HTTPS | Enroll secret and host UUID from the managed configuration. |

> **Note:** The Pub/Sub token is a query parameter, so keep it out of access logs on any proxy in front of Fleet.

Source: [create-android-signup-url.js#L26-L50](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-signup-url.js#L26-L50) (`Origin` check), [get-android-device.js#L36-L58](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/get-android-device.js#L36-L58) (bearer check, same pattern on every authenticated endpoint), [create-android-enterprise.js#L100-L110](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L100-L110) (topic IAM), [pubsub.go#L81-L109](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/pubsub.go#L81-L109) (push token check), and [create-android-enterprise.js#L131-L133](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L131-L133) (push configuration, no OIDC).

## Network requirements

- The Fleet server needs outbound HTTPS to fleetdm.com.
- Google Pub/Sub must reach `/api/v1/fleet/android_enterprise/pubsub`. If Fleet sits behind an identity-aware proxy, exempt this path. The token in the URL authenticates it.
- Devices that are enrolling must reach the Fleet enrollment page, and devices running the Fleet agent must reach the Fleet server.

Learn more in [Which API endpoints to expose to the public internet?](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet#android)

## Data that passes through fleetdm.com

| Data | When | Stored by the proxy |
| ---- | ---- | ------------------- |
| Fleet server URL, and the Fleet license key (a signed JWT with tier, device count, and expiry) | Setup, once | Yes (`AndroidEnterprise` row) |
| Android Enterprise ID, Pub/Sub topic and subscription names, and per-server secret | Setup | Yes |
| Full AMAPI policy JSON: settings, app lists, managed configurations, and custom profiles as the admin wrote them. Custom profiles are raw AMAPI policy JSON, so anything the admin puts in them passes through, including Wi-Fi credentials in `openNetworkConfiguration`. Includes the Fleet agent managed configuration (Fleet server URL, host UUID, enroll secret, and certificate template IDs). Certificate material doesn't pass through the proxy. | Every policy change | No, forwarded only |
| Device data. The device list returns device names only. Device patch, command, and operation responses return the device or operation record from Google. App lookups return Play Store metadata for a package name. Web app creation sends the URL, title, and icon. Fleet doesn't call `devices.get` through the proxy. | Reconciliation jobs and admin actions | No, forwarded only |
| Enrollment token requests, including the Fleet enroll secret and the IdP account UUID in `additionalData`, and the returned token and QR code | Each enrollment | No, forwarded only |
| Commands: lock, wipe, and clear passcode. Fleet's own buttons send clear passcode with an empty password. The custom command API (`POST /api/v1/fleet/commands/run`) forwards admin-written AMAPI command JSON as is, so an admin can send `RESET_PASSWORD` with a new password through the proxy. | Admin actions | No, forwarded only |

Pub/Sub device events don't pass through fleetdm.com.

The proxy counts AMAPI requests per enterprise and per request type, and sends the counts to Datadog once a minute, tagged with the enterprise ID and the Fleet server origin. When a Google call fails, the proxy logs the full error object, which includes the request it was making. The Google client redacts the `Authorization` header and OAuth fields in that object, but not policy or device bodies.

Source: [AndroidEnterprise.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/models/AndroidEnterprise.js), [proxy_client.go#L101-L120](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/proxy_client.go#L101-L120) (license key sent once), [service.go#L1593-L1598](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L1593-L1598) (agent managed configuration), [profiles.go#L586-L615](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/profiles.go#L586-L615) (custom profiles with `openNetworkConfiguration`), [proxy_client.go#L253-L262](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/proxy_client.go#L253-L262) (device list, names only), [service.go#L1186-L1196](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L1186-L1196) (clear passcode), [service.go#L1325-L1372](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L1325-L1372) (custom command passthrough), [hooks/custom/index.js#L142-L205](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/hooks/custom/index.js#L142-L205) (metrics and tags), and [modify-android-policies.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/modify-android-policies.js) (error logging, same on every endpoint).

## What happens when a component fails

| What fails | Impact | Recovery |
| ---------- | ------ | -------- |
| fleetdm.com proxy is down | New enrollments fail. Admin actions fail: lock, wipe, and policy and app changes. Enrolled devices aren't affected. They keep their last policy and keep reporting. | Profile changes retry every 30 seconds until they go through. App availability changes are queued jobs that retry 5 times over about 3.5 hours, then are marked as failed, and the admin needs to save the app again. Lock, wipe, and clear passcode return an error to the admin and must be sent again. Device events keep arriving and are processed without the proxy. |
| Fleet server is down | Events queue. Pub/Sub holds device events for up to 24 hours and retries the push. No enrollments or admin actions. | Events are delivered when Fleet comes back. An outage longer than 24 hours loses queued events, and the next status report restores device state. A lost `ENROLLMENT` event isn't recovered: the hourly device reconciler only removes hosts Google no longer has, and doesn't add ones Fleet never saw. Those end users need to enroll again. |
| Google AMAPI is down | All Android management stops. This affects every AMAPI customer the same way, with or without a proxy. | Depends on Google. |
| Shared AMAPI quota is exceeded (HTTP `429`) | Calls are delayed. All Fleet servers share one Google Cloud project quota. | Admin requests retry once after 60 seconds plus jitter, within a 75-second budget per request. Scheduled jobs and the enrollment page don't wait: the job stops and retries on its next run, and the enrollment page returns an error so the end user can try again. All 15 proxy endpoints log a `p1:` warning on a `429`. Code comments say this triggers Fleet's P1 alert. |
| Per-server secret is lost or doesn't match | All proxy calls fail (`401`). | There's no self-service fix. Turning off Android MDM also fails, because that call needs the same secret, and Fleet's hourly enterprise check treats a `401` as a temporary error and changes nothing. Deleting the enterprise in the Google Admin console doesn't clear it either, since the check can't reach the proxy. [Contact Fleet](https://fleetdm.com/support) to fix the proxy record. |

Source: [retry_client.go#L15-L24](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/retry_client.go#L15-L24) (retry delay and budget), [profiles.go#L33](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/profiles.go#L33), [software_worker.go#L92](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/worker/software_worker.go#L92) and [service.go#L597-L602](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L597-L602) (jobs and the enrollment page skip the retry), [cron.go#L2288-L2312](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L2288-L2312) (profile manager, 30 seconds), [cron.go#L2989-L3012](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L2989-L3012) (device reconciler, 1 hour), [worker.go#L18](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/worker/worker.go#L18) and [worker.go#L147-L158](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/worker/worker.go#L147-L158) (5 retries and delays), [reconcile_devices.go#L25-L27](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/reconcile_devices.go#L25-L27) (reconciler only removes), [service.go#L826-L880](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L826-L880) (enterprise check, `401` handling), [cron.go#L1688-L1690](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L1688-L1690) (hourly check), [get-android-device.js#L80-L84](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/get-android-device.js#L80-L84) (`p1:` warning), and [create-android-enterprise.js#L126-L133](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L126-L133) (Pub/Sub retention).

## Reliability

- The proxy is part of the fleetdm.com web app, which runs on Heroku. It isn't deployed in multiple regions.
- The proxy is monitored. Per-minute AMAPI request counts per enterprise and per request type go to Datadog, and Google rate-limit errors log a warning that, per code comments, triggers Fleet's P1 alert.
- Proxy traffic is light. It carries setup calls, one call per enrollment, admin actions, and three scheduled jobs:
  - Profile reconciliation every 30 seconds. It only calls Google when something changed.
  - An hourly device list, with one call per 100 devices.
  - An hourly check that the enterprise still exists.
- Device check-ins don't use the proxy.
- Fleet doesn't plan a multi-region deployment or a second failover path for now. Fleet will add one if proxy downtime leads to a measurable rise in support tickets.
- The Fleet server can skip the proxy and call Google directly with a customer-owned service account (`FLEET_DEV_ANDROID_GOOGLE_CLIENT`). This is a development setting and isn't a supported configuration.

Source: [hooks/custom/index.js#L165-L175](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/hooks/custom/index.js#L165-L175) (Datadog), [website/README.md#L20-L24](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/README.md#L20-L24) (Heroku deploys), [cron.go#L1688-L1690](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L1688-L1690), [cron.go#L2288-L2312](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L2288-L2312), and [cron.go#L2989-L3012](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L2989-L3012) (scheduled jobs), and [google_client.go#L25-L37](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/google_client.go#L25-L37) (direct client).

## End user enrollment experience

The admin chooses one of two paths on the **Add hosts** page.

### Personal devices (work profile)

1. IT shares the enrollment link (`{Fleet URL}/enroll?enroll_secret=...`). The end user opens it on the phone and signs in with the company IdP if Fleet requires it.
2. Fleet requests a one-time token from Google through the proxy, and shows an enrollment link and QR code.
3. The end user follows Google's flow. Android creates a work profile, and the personal side isn't touched.
4. Policies and apps install in the work profile. Google sends the `ENROLLMENT` event to Fleet, and the device appears in Fleet.

### Company-owned devices (fully managed)

1. The device must be at the setup wizard, so it's new out of the box or factory reset.
2. IT opens the same enrollment page with `fully_managed=true` on another device. Fleet requests a token through the proxy and shows a QR code.
3. The setup wizard scans the QR code and Google takes over the whole device. Fleet also supports [zero-touch enrollment](https://fleetdm.com/guides/android-zero-touch-enrollment) for devices bought through a reseller.
4. Policies and apps install. Google sends the `ENROLLMENT` event to Fleet, and the device appears in Fleet.

If fleetdm.com is unreachable at step 2 on either path, the page shows an error and the end user can try again later. Nothing on the device changes until step 3.

Source: [AndroidPanel.tsx#L11-L25](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/frontend/components/AddHostsModal/PlatformWrapper/AndroidPanel/AndroidPanel.tsx#L11-L25) (link format, two enrollment types), [service.go#L612-L717](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L612-L717) (token request, IdP session, personal-usage flag), and [enroll-hosts.md#L29-L41](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/articles/enroll-hosts.md#L29-L41).

## Google conditional access

> **Note:** This section describes planned work, not a released feature. The design may change before it ships.

Today, you can block sign-in from iPhones, iPads, and Android devices that aren't managed by Fleet with Google Context-Aware Access (part of Google's BeyondCorp model). Follow the [Conditional access: Google](https://fleetdm.com/guides/google-conditional-access-integration) guide. End users open a Chrome bookmark that loads a page you host, and that page marks company-owned devices as managed in Google ([#54888](https://github.com/fleetdm/fleet/issues/54888)).

Native support in Fleet is [coming soon](https://github.com/fleetdm/fleet/issues/56195). Fleet will mark company-owned iOS, iPadOS, and Android hosts as managed in Google, so you won't need to host your own page. Like Android MDM, it will go through fleetdm.com ([#56354](https://github.com/fleetdm/fleet/issues/56354)), which will hold Fleet's Google credentials.

### Planned setup

1. An admin enters the organization's Google customer ID in Fleet and selects **Connect**.
2. The Fleet server registers with fleetdm.com and gets a per-server secret, the same way Android MDM setup works.
3. The admin signs in with Google to approve Fleet. Setup only finishes after fleetdm.com confirms that the admin belongs to that Google customer, so no one else can connect your Google organization to their Fleet server.

### Planned flow for marking a device as managed

1. The end user selects a button in **Self-service** on a company-owned iPhone, iPad, or Android device and signs in to Google.
2. Google redirects to fleetdm.com.
3. fleetdm.com asks the Fleet server to confirm that the host is company-owned and enrolled to Fleet.
4. fleetdm.com looks up the device in Google and saves its Google device ID on the host in Fleet. Fleet refuses if the host already has a different Google device ID, so a link copied to a personal device can't take over a company-owned host.
5. fleetdm.com marks the device as managed in Google with Fleet's service account. Google Context-Aware Access can then block sign-in from devices that aren't managed.

### What will change compared to Android MDM

- For Android MDM, fleetdm.com never calls your Fleet server. Google conditional access will add calls from fleetdm.com to your Fleet server, authenticated with the per-server secret. If you restrict inbound traffic, you'll need to allow `/api/*/fleet/google_conditional_access/*` and `/api/*/fleet/hosts/*/google_device_id` from fleetdm.com.
- New data will pass through fleetdm.com: the Google customer ID, the Fleet server URL, and, for each device, its host ID and UUID, platform, enrollment status, and Google device ID.

## Further reading

- [Android MDM setup](https://fleetdm.com/guides/android-mdm-setup)
- [Android proxy source code](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy)
- [Fleet server Android MDM source code](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service)

<meta name="articleTitle" value="How Fleet's Android MDM connects to Google">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-23">
<meta name="description" value="How Fleet's Android MDM reaches Google through fleetdm.com: traffic paths, authentication, data handled, failure modes, and network requirements.">
