# Fleet 4.81.0 | Lower AWS costs, automatic IdP deprovisioning, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/LAXHRKqxBAk?si=UBaH_TE9L5IEMAtq" title="0" allowfullscreen></iframe>
</div>

Fleet 4.81.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.81.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Lower AWS costs
- Automatic IdP deprovisioning
- Okta as a certificate authority (CA) with a dynamic challenge
- Proxy-ready Fleet Desktop configuration
- Windows profiles behave like macOS

### Lower AWS costs

Fleet now supports [gzip compression](https://developer.mozilla.org/en-US/docs/Glossary/gzip_compression) on agent API responses, reducing outbound bandwidth from your Fleet server. For Fleet users who self-host, that means lower AWS costs with no workflow changes. Currently, gzip compression is currently off by default but will soon default to on. Learn how to [turn on compression](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-gzip-responses). 

GitHub issue: [#37944](https://github.com/fleetdm/fleet/issues/37944)

### Automatic IdP deprovisioning

When a user is removed from your identitify provider (IdP), like Okta, they’re now automatically removed from Fleet by default. No configuration changes needed. Security Engineers no longer need to worry about dangling admin accounts. IT Admins get cleaner offboarding and fewer manual access reviews.

GitHub issue: [#36785](https://github.com/fleetdm/fleet/issues/36785)

### Okta as a certificate authority (CA) with a dynamic challenge

Fleet now supports dynamic challenges when deploying certificates for Okta Verify. Each host gets a unique secret at enrollment, strengthening security. 

To configure Okta as a CA, in Fleet, head to **Settings > Integrations > Certificate authorities**, select **Add CA**, and choose **Okta CA or Microsoft Device Enrollment service (NDES)**. Okta uses NDES under the hood. If you're using static challenges with Okta's CA, choose **Custom Simple Certificate Enrollment Protocol (SCEP)** instead.

GitHub issue: [#34521](https://github.com/fleetdm/fleet/issues/34521)

### Proxy-ready Fleet Desktop configuration

For users that self-host Fleet, you can now configure an [alternative URL](https://fleetdm.com/guides/enroll-hosts#alternative-browser-host) for Fleet Desktop. IT Admins can route traffic through a custom proxy for added control.

GitHub issue: [#33762](https://github.com/fleetdm/fleet/issues/33762)

### Windows profiles behave like macOS

Just like macOS, Windows [configuration profiles](leetdm.com/guides/custom-os-settings) now apply payloads individually. If one payload fails, the rest still succeed, bringing consistency across platforms. IT Admins get faster enforcement of critical controls without waiting on edge-case fixes.

GitHub issue: [#31922](https://github.com/fleetdm/fleet/issues/31922)

## Changes

### IT Admins
- Added support for dynamic SCEP challenges for Okta certs.
- Added a feature to allow IT admins to specify non-atomic Windows MDM profiles.
- Added GitOps support to fleet yaml to apply display_name to software package.
- Added enrollment support for iPod touch.
- Added `hash_sha256` and `package_name` query parameters to the `GET /api/v1/fleet/software/titles` endpoint to allow checking if a custom software package already exists before uploading. Both parameters require `team_id` to be specified.
- Added ability to set default URL for Fleet Desktop.
- Added logic to skip setup experience for hosts that were enrolled > 1 day ago.
- Updated maximum software installer size to be configurable and bumped the default from 3 GB to 10 GiB.
- Added a check to fail any pending in-house app installs and cancel upcoming activities when unenrolling a host.
- Added `gzip_responses` server configuration option that allows the server to gzip API responses when the client indicates support through the `Accept-Encoding: gzip` request header.
- Allowed specifying an Apple Connect JWT for interacting directly with Apple APIs when retrieving VPP app metadata.
- Added logic to .pkg metadata extraction to match the root bundle identifier.
- Moved Windows automatic enrollment configuration instructions out of the UI and into the Windows MDM setup guide.

### Security Engineers
- Added `conditional_access.cert_serial_format` server option to allow specifying the Okta conditional access certificate serial format.
- Improved authentication of `POST /api/v1/osquery/carve/block` requests by parsing and validating `session_id` and `request_id` before processing `data`.
- Redirected users to device policy page when failing conditional access requirements.
- Limited disk encryption key escrowing when global or team setting enabled.
- Differentiated IMP and Integrative Modeling Platform (IMP) while running vulnerability scanning.
- Fixed false negative for Adobe Reader DC CVE-2025-54257 & CVE-2025-54255.

### Other improvements and bug fixes
- Added an environment variable to allow reverting to the old behavior of installing the bootstrap package during macOS MDM migration.
- Added `--with-table-sizes` option to `prepare` command to get approximate row counts of all database tables after a migration completes.
- Updated Fleet UI so that if software is detected as installed on software library page, hide any Fleet install/uninstall failures from page. Admin can view these failures from host details > activities.
- Updated Android certificate app to re-enroll if the host was deleted in Fleet.
- Updated `fleetctl generate-gitops` to output Fleet-maintained apps in a dedicated `fleet_maintained_apps` section of the YAML files.
- When a host is deleted, any associated VPP software installation records are also deleted.
- Global observers and maintainers can now officially read user details, which were already visible to them via the activity feed.
- Iru (Kandji's new name) added to the list of well-known MDM platforms.
- Improved error message when viewing disk encryption key fails because MDM has been turned off and the decryption certificate is no longer valid.
- Updated UI to show VPP version for adding software during setup.
- User sessions and password reset tokens are now cleared whenever a user's password is changed.
- Disallowed use of FLEET_DEV_* environment variables unless `--dev` is passed when serving Fleet.
- Handled the NotNow status from the device during DEP setup experience so it does not delay the release of the device.
- Allowed overriding individual configuration variables for MySQL and object storage when `--dev` is passed when serving Fleet.
- Updated DEP syncing code to use server-protocol-version 9 and handle THROTTLED responses.
- Updated UI styling to the Packs flow.
- Surfaced Google error message for Android profile failures after max retries instead of a generic error.
- Optimized recording of scheduled query results in the database.
- Improved API error message when adding profiles or software with non-existent labels.
- Ignored parenthesized build numbers in UI when comparing versions for update availability (e.g. 5.0 (build 3400)).
- Improved DEP process cooldowns, by limiting how many we process in a single as per Apple's recommendations.
- Improved OpenTelemetry tracing: added proper shutdown to flush pending spans, and added service name/version resource attributes for better trace identification.
- Improved OpenTelemetry error handling: client errors (4xx) no longer set span status to Error or appear in the Exceptions tab, following OTEL semantic conventions. Added separate metrics for client vs server errors (`fleet.http.client_errors`, `fleet.http.server_errors`) with error type attribution. Client errors are also no longer sent to APM/Sentry.
- Internal refactoring: introduced activity bounded context as part of modular monolith architecture. Moved /api/latest/fleet/activities endpoint to new server/activity/ packages.
- Removed a debug-level warning asserting that macOS devices were unauthenticated when enrolling to Fleet.
- Updated gitops related tests to validate that users can get/set the alternative browser hosts fleet desktop setting.
- Updated to Go 1.25.7.
- Fixed a bug with the `PATCH /software/titles/{id}/package` where the categories could not be updated by themselves, another field had to be updated for them to be modified.
- Fixed an issue setting the bootstrap package on teams created by the puppet plugin.
- Fixed an issue where enabling manual agent installation for macOS devices would incorrectly block the addition of setup experience software titles for all platforms.
- Fixed Smallstep CA integration to send Authorization header with first request.
- Fixed an issue where deleted Windows and Linux hosts could re-enroll without re-authenticating when End User Authentication was enabled.
- Fixed a permission issue on software installer custom icons where a team maintainer could not view, edit or delete a custom icon.
- Fixed bug where unfinished Entra Integration setup breaks the UI.
- Fixed SCEP proxy so that it uses standard base64 encoding for PKIOperation GET requests, ensuring compatibility with standard SCEP servers.
- Fixed an issue where queries with common table expressions (CTEs) were marked as having invalid syntax.
- Fixed a bug where installing Xcode via VPP apps on macOS resulted in a failure due to not being able to verify the install.
- Fixed a bug where non utf8 encodings caused an error in pkg metadata extraction.
- Improved error message where there is issue getting the enrollment token during ota enrollment.
- Fixed CVE false positive on ninxsoft/Mist.
- Fixed an issue where `last_install` details were not returned in the Host Software API for failed software installs, preventing users from viewing failure information.
- Fixed saving of policy automation in UI that triggers software installs and script runs.
- Fixed a bug where changes to scripts were causing custom software display names to be deleted.
- Fixed bug where custom icons were ignored for fleet maintained apps in GitOps files.
- Fixed panic in gRPC launcher API handler.
- Fixed a bug where installed software would not show up in the software inventory of an ADE-enrolled macOS host after a wipe and a re-enrollment.
- Fixed issue where MySQL read replicas were not using TLS.
- Fixed bug where `fleetctl gitops` was not sending software categories correctly in all cases.
- Fixed an issue in `fleetctl gitops` that would reset VPP token team assignment when using "All teams".
- Fixed bug in host activity card UI where activities related to MDM commands should be hidden when Apple MDM features are turned off in Fleet.
- Fixed unnecessary error logging when no CPE match is found for software items like VSCode extensions and JetBrains plugins.
- Fixed created_at and updated_at timestamps on API responses for Label and Team creation.
- Fixed issues where different variations of the same software weren't linked to the same software title.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.81.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-02-20">
<meta name="articleTitle" value="Fleet 4.81.0 | Lower AWS costs, automatic IdP deprovisioning, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.81.0-1600x900@2x.png">
