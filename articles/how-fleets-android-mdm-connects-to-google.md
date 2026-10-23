# How Fleet's Android MDM connects to Google

Fleet manages Android devices with Google's [Android Management API](https://developers.google.com/android/management) (AMAPI). Your Fleet server doesn't call Google directly. It calls a proxy at `https://fleetdm.com/api/android/`, which holds Fleet's Google Cloud credentials and forwards each call to AMAPI. Facts in this guide come from Fleet's source code at commit [59e246b](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03), and each section links to the code it's based on.

## Reliability of fleetdm.com

- **fleetdm.com sits on one path.** Your Fleet server calls it to enroll devices and send admin actions like lock, wipe, and policy changes, and it forwards them to Google. Device check-ins and device events never touch it.
- **No failover. Not multi-region.** If fleetdm.com is unreachable, new enrollments and admin actions fail. Devices that are already enrolled keep enforcing their last policy and keep reporting. Policy changes retry automatically, and lock and wipe must be sent again.
- **99.993% uptime.** Availability is published at [status.fleetdm.com](https://status.fleetdm.com).

## Traffic flow

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
- The proxy generates a random 30-character secret for your Fleet server and returns it once. Fleet stores it and sends it as the bearer secret on every later call.
- Fleet generates its own token (64 random bytes, base64url-encoded) and adds it to the Pub/Sub push URL. Fleet rejects any push whose token doesn't match.

Source: [service.go#L319-L356](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L319-L356) (token generation, enterprise creation, secret storage), [handler.go#L36](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L36) (signup callback route), and [create-android-enterprise.js#L74-L195](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L74-L195) (Pub/Sub setup, secret generation).

### Device enrollment

1. End user to Fleet server: opens the Fleet enrollment page (with optional IdP sign-in).
2. Fleet server to proxy: `POST /v1/enterprises/{id}/enrollmentTokens` with the bearer secret.
3. Proxy to Google: `enrollmentTokens.create`.
4. Google to Fleet server, through the proxy: a one-time token and QR code.
5. End user's device to Google: `enterprise.google.com/android/enroll?et=...`.
6. Google to Fleet server: the `ENROLLMENT` event, pushed directly by Pub/Sub with no proxy.

Enrollment tokens are one-time use and expire after 1 hour (the AMAPI default, since Fleet doesn't set a duration). Fleet creates a new token for each enrollment, so enrollment doesn't work without the proxy.

Source: [service.go#L612-L717](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L612-L717) (token request) and [create-android-enrollment-token.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enrollment-token.js).

### Everything else

- **Admin actions go through the proxy.** Policy and profile changes, app installs, web apps, lock, wipe, clear passcode, device lookups, and unenroll go from the Fleet server to fleetdm.com to Google. Policy changes include the Fleet agent's managed configuration, which carries the Fleet server URL, the host UUID, and an enroll secret.
- **Device events skip the proxy.** Google Pub/Sub pushes four event types straight to `{Fleet URL}/api/v1/fleet/android_enterprise/pubsub?token=...`: `ENROLLMENT`, `STATUS_REPORT`, `COMMAND`, and `USAGE_LOGS`. Fleet processes the first three and drops the rest. The request body limit is 10 MiB.
- **Devices skip the proxy.** Enrolled devices get their policy from Google and report to Google. If the Fleet agent is deployed, it also reports to the Fleet server. Neither path touches fleetdm.com.

Source: [proxy_client.go](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/proxy_client.go) (one method per call), [android.go#L66-L71](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/android.go#L66-L71) (agent managed configuration), [handler.go#L18](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L18), [handler.go#L38](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/handler.go#L38), [service.go#L336-L341](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/service.go#L336-L341) (event types), and [routes.js#L1532-L1546](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/config/routes.js#L1532-L1546) (all 15 proxy endpoints).

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

## How each hop is authenticated

| Hop | Transport | How the caller is authenticated |
| --- | --------- | ------------------------------- |
| Fleet server to fleetdm.com | HTTPS | Per-server bearer secret on every call after setup. Setup calls are keyed on the Fleet server URL and need a one-time token from the admin's Google signup. One Fleet server can't reach another server's enterprise. |
| fleetdm.com to Google | HTTPS | Fleet's Google service account, scoped to one Google Cloud project. Its credentials stay on fleetdm.com. |
| Google Pub/Sub to Fleet server | HTTPS push | A random token in the push URL. This is the only check on this hop, since there's no Google-signed token. |
| Device to Google | HTTPS | Managed by Google (Android Device Policy). |
| Fleet agent to Fleet server | HTTPS | Enroll secret and host UUID from the agent's managed configuration. |

> **Note:** The Pub/Sub token is a query parameter, so keep it out of access logs on any proxy in front of Fleet. If Fleet sits behind an identity-aware proxy, exempt `/api/v1/fleet/android_enterprise/pubsub`.

Source: [get-android-device.js#L36-L58](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/get-android-device.js#L36-L58) (bearer check, same pattern on every authenticated endpoint), [pubsub.go#L81-L109](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/pubsub.go#L81-L109) (push token check), and [create-android-enterprise.js#L131-L133](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L131-L133) (push configuration).

## Data that passes through fleetdm.com

- **Stored.** The Fleet server URL, the Fleet license key, the Android Enterprise ID, the Pub/Sub topic names, and the per-server secret.
- **Forwarded, not stored.** Policy JSON (settings, app lists, and custom profiles), device and operation records from Google, enrollment requests, and commands. Admin-written profiles and commands are forwarded as written, including any credentials they contain.
- **Logged.** Request counts per enterprise go to Datadog every minute. When a Google call fails, the proxy logs the error, which can include the request.

Source: [AndroidEnterprise.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/models/AndroidEnterprise.js) (stored fields) and [modify-android-policies.js](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/modify-android-policies.js) (error logging, same on every endpoint).

## What happens when a component fails

| What fails | Impact | Recovery |
| ---------- | ------ | -------- |
| fleetdm.com is unreachable | New enrollments and admin actions fail. Enrolled devices keep their last policy and keep reporting. | Policy changes retry every 30 seconds. App changes retry for about 3.5 hours, then the admin needs to save the app again. Lock and wipe must be sent again. |
| Your Fleet server is unreachable | Google holds device events for 24 hours and retries. | Events arrive when Fleet comes back. After 24 hours, events are lost. The next status report restores device state, but a lost enrollment event means the end user needs to enroll again. |
| Google's API is unreachable | All Android management stops, with or without fleetdm.com. | Resumes when Google does. |
| Google rate limit (HTTP `429`) | Calls are delayed. All Fleet servers share one Google Cloud project quota. | Admin requests retry once after about 60 seconds. Scheduled jobs retry on their next run. |

Source: [retry_client.go#L15-L24](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service/androidmgmt/retry_client.go#L15-L24) (rate-limit retry), [cron.go#L2288-L2312](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/cmd/fleet/cron.go#L2288-L2312) (30-second profile job), [worker.go#L147-L158](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/worker/worker.go#L147-L158) (app job retries), and [create-android-enterprise.js#L126-L133](https://github.com/fleetdm/fleet/blob/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy/create-android-enterprise.js#L126-L133) (24-hour Pub/Sub retention).

## Google conditional access

> **Note:** This section describes planned work, not a released feature. The design may change before it ships.

Fleet will soon mark company-owned iPhones, iPads, and Android hosts as managed in Google ([#56195](https://github.com/fleetdm/fleet/issues/56195)). Like Android MDM, part of it will go through fleetdm.com ([#56354](https://github.com/fleetdm/fleet/issues/56354)): the end user's Google sign-in, which needs Fleet's Google OAuth client. That client is shared by every Fleet server, so its secret stays on fleetdm.com.

fleetdm.com will:

- Run the end user's Google sign-in. It starts when the end user opens the Fleet app on Android or the **Self-service** web clip on an iPhone or iPad. There's no button to find.
- Use the sign-in to find the device in Google and send its Google device ID to your Fleet server. Fleet refuses a second device for the same host, so a link copied to a personal device can't take over a company-owned host.

fleetdm.com won't mark devices as managed. Your Fleet server does that with a service account you create in your own Google Cloud project, with domain-wide delegation limited to the `cloud-identity.devices` scope. You own the credential, and the Fleet server unmarks a device when its host leaves Fleet or stops being company-owned. No Google credentials pass through fleetdm.com.

Compared to Android MDM:

- fleetdm.com will call your Fleet server, authenticated with the per-server secret. If you restrict inbound traffic, you'll need to allow that endpoint from fleetdm.com. The exact path isn't final yet. See the plan in [#56354](https://github.com/fleetdm/fleet/issues/56354).
- New data will pass through fleetdm.com: the Fleet server URL and, for each device, its host UUID, platform, enrollment status, Google device ID, the end user's Google account, and, on Android, the device's Android ID.

## Further reading

- [Android MDM setup](https://fleetdm.com/guides/android-mdm-setup)
- [Which API endpoints to expose to the public internet?](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet#android)
- [Android proxy source code](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03/website/api/controllers/android-proxy)
- [Fleet server Android MDM source code](https://github.com/fleetdm/fleet/tree/59e246b449a55ed95f1e9e516739c6b2bb723f03/server/mdm/android/service)

<meta name="articleTitle" value="How Fleet's Android MDM connects to Google">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-23">
<meta name="description" value="How Fleet's Android MDM reaches Google through fleetdm.com: reliability, traffic paths, authentication, data handled, and failure modes.">
