# Fleet 4.93.0 | macOS app patching, Android zero-touch enrollment, Windows admin password rotation, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/0qYhQAycHu0?si=DoIXdNTs-R-1M7p4" allowfullscreen></iframe>
</div>
-->

Fleet 4.93.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.93.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [macOS app patching: prompt end users before deadline](#macos-app-patching-prompt-end-users-before-deadline)
- [Android zero-touch enrollment](#android-zero-touch-enrollment)
- [More Android host vitals](#more-android-host-vitals)
- [Rotate Windows local admin password](#rotate-windows-local-admin-password)
- [Restrict Managed Apple Account sign-in to managed devices](#restrict-managed-apple-account-sign-in-to-managed-devices)
- [Hardware attestation for iOS and iPadOS](#hardware-attestation-for-ios-and-ipados)
- [Reports that cover every host](#reports-that-cover-every-host)

### macOS app patching: prompt end users before deadline

_Available in Fleet Premium_

IT admins can now warn end users before Fleet patches an open macOS app. When a patch policy with `notify_before_patching` fails and the app is open, Fleet Desktop shows a notification listing the apps that will update, waits one hour, shows a reminder five minutes before, and then installs the update. This gives end users time to save their work, instead of waiting until they close the app on their own (`patch_when_closed`).

The deadline is one hour. Setting a custom deadline is [coming soon](https://github.com/fleetdm/fleet/issues/39176).

See what end users experience, including a video walkthrough, in the [patching end user experience guide](https://fleetdm.com/guides/patching-end-user-experience).

GitHub issue: [#39178](https://github.com/fleetdm/fleet/issues/39178)

### Android zero-touch enrollment

_Available in Fleet Premium_

IT admins can now ship company-owned Android devices straight to end users and have them enroll in Fleet the first time they're turned on. In **Settings > Integrations > MDM > Android zero-touch**, copy the DPC extras JSON into Google's zero-touch portal. Zero-touch-enrolled hosts land in "Unassigned." Assigning them to a different fleet automatically is [coming soon](https://github.com/fleetdm/fleet/issues/51479).

Learn how to set it up in the [Android zero-touch enrollment guide](https://fleetdm.com/guides/android-zero-touch-enrollment).

GitHub issue: [#49165](https://github.com/fleetdm/fleet/issues/49165)

### More Android host vitals

IT admins can now see more Android host vitals on the **Host details** page, including whether USB debugging is on, whether a passcode is set, whether Google Play Protect is on, encryption status, security patch level, manufacturer, security posture (from Google's Play Integrity checks), and phone numbers. These vitals are also returned by the [get host API](https://fleetdm.com/docs/rest-api/rest-api#get-host).

GitHub issue: [#49791](https://github.com/fleetdm/fleet/issues/49791)

### Rotate Windows local admin password

_Available in Fleet Premium_

IT admins can now rotate the password for Fleet's managed local admin account on Windows hosts, the same way they already could on macOS. Go to **Host details > Actions > Show managed account** and select **Rotate password**. Fleet also rotates the password automatically an hour after someone views it, so a password shared for troubleshooting doesn't stay valid.

Learn more about the [managed local account on Windows](https://fleetdm.com/guides/windows-linux-setup-experience#managed-local-account-windows).

GitHub issue: [#43489](https://github.com/fleetdm/fleet/issues/43489)

### Restrict Managed Apple Account sign-in to managed devices

_Available in Fleet Premium_

IT admins can now make sure end users sign in to their Managed Apple Account only on devices enrolled in Fleet. This stops company data from syncing to personal, unmanaged Apple devices. In Apple Business, set **Allow Managed Apple Account on** to **Managed devices only** or **Supervised devices only**, and Fleet confirms to Apple that the host is managed during sign-in. 

Hosts that automatically enroll (ADE) are already tied to an Apple Business (AB). Manually enrolled hosts aren't, so Fleet uses your default AB for sign-in. If you've added more than one AB, choose a default for sign-in in **Settings > Integrations > MDM > Apple Business (AB)**. Otherwise, Managed Apple Account sign-in fails on manually enrolled hosts.

Learn more in the [Apple MDM setup guide](https://fleetdm.com/guides/apple-mdm-setup#restrict-apple-account-sign-in-managed-apple-accounts).

> Only turn on this setting if you're just starting to roll out Managed Apple Account sign-in. If your end users already sign in with Managed Apple Accounts, leave it off for now. Hosts enrolled before Fleet 4.93.0 can't sign in to Managed Apple Accounts with this setting on until their next enrollment profile renewal, about every six months.
>
> If you want to turn it on sooner, follow [these instructions](https://docs.google.com/document/d/1f3OZaC9lhN58esD3cXzqEePx2kq_tHX7b1KgwSEOQ3c/edit?tab=t.0).

GitHub issue: [#45829](https://github.com/fleetdm/fleet/issues/45829)

### Hardware attestation for iOS and iPadOS

_Available in Fleet Premium_

Hardware attestation (ACME), which Fleet already supports for Apple silicon Macs, now works for iPhones and iPads assigned to Fleet in Apple Business. With `apple_require_hardware_attestation` on, iPhones and iPads from 2017 or later (with an A11 Bionic chip or later), running iOS or iPadOS 16 or later, prove their hardware matches a known Apple Business record when they enroll. Hosts already enrolled with SCEP move to ACME on their next certificate renewal. Older devices keep enrolling with SCEP.

Learn more in the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files#controls).

GitHub issue: [#51528](https://github.com/fleetdm/fleet/issues/51528)

### Reports that cover every host

Reports now store as many results as you have hosts, instead of stopping at 1,000. A report that returns one result per host covers your whole fleet, so you can bring Jamf extension attributes over to Fleet. Sorting, search, and pagination now run on the server, so large reports stay usable.

Fleet doesn't store a host's result over 512 KB. Learn more in the [reports guide](https://fleetdm.com/guides/reports).

GitHub issue: [#43723](https://github.com/fleetdm/fleet/issues/43723)

## Changes

### IT Admins
- Added a "Notify before patching" option for macOS Fleet-maintained app patch policies. When the app is open, Fleet Desktop tells the end user 1 hour before the app is closed and updated, and again 5 minutes before. Requires Fleet Desktop 1.5.0.
- Added support for Android zero-touch enrollment for company-owned devices.
- Added the new Android host vitals to the Vitals card on the host details page: USB debugging enabled, passcode set, Play Protect enabled, encryption status, manufacturer, security update version, kernel version, bootloader version, software update status, security posture, carrier, phone number, IMEI, and MEID, plus the device's Android API level in a tooltip on the operating system. Phone number, carrier, IMEI, and MEID are shown only for company-owned hosts. On Android hosts the operating system no longer repeats the security patch level, which now has its own vital.
- Added collection of the Android hardware radio identifiers (IMEI and MEID) from AMAPI status reports, returned as `imei` and `meid` by the get host endpoints for company-owned Android hosts.
- Added the ability to rotate the managed local account password on Windows hosts, from Host details > Actions > Show managed account or the `POST /hosts/:id/managed_account_password/rotate` endpoint. As on macOS, viewing the password also schedules an automatic rotation about an hour later.
- Added IdP host vitals (username, full name, groups, department) on Entra-joined Windows hosts enrolled without Fleet MDM, by matching the Entra join user reported by the device to the SCIM-provisioned user. Manually set and end user authentication usernames take precedence.
- Added `host_id` and `host_serial` to the `mdm_enrolled` activity for Windows hosts, so the activity appears on the host's activity timeline and automations can identify the device. Windows automatic (Entra/Autopilot) enrollments report neither the host nor its serial at enrollment time, so their activity is now recorded the first time the enrollment is linked to a host instead of being recorded without them. The activity also reports `installed_from_dep` for Windows, and the activity feed shows the serial and the enrollment type alongside the host name.
- Added support for restricting Managed Apple Account sign-ins via Apple Business, using the GetToken protocol.
- Added the ability for users with the Technician role to clear passcodes on iOS and iPadOS hosts.
- Raised the report cap to the number of hosts when that is higher than `report_cap`, so reports that return one result per host are never clipped.
- Changed reports that reach the cap to keep updating results for hosts already in the report instead of pausing entirely. Only results from hosts not yet in the report are skipped.
- Stopped storing a host's result for a report when it is larger than 512 KB. Only the fetch time is recorded.
- Added `page`, `per_page`, `order_key`, `order_direction` and `query` parameters to `GET /api/v1/fleet/reports/:id/report`, and a `count` field to its response. Without `per_page` the endpoint still returns all results, now ordered by `last_fetched` descending.
- Changed the report results page to paginate, sort, and search server-side instead of loading every result into the browser.
- Added real online and offline status for iOS, iPadOS, and Android hosts based on MDM activity, instead of always reporting them as offline. The status is reflected on the hosts list, host details, dashboard summary, and target counts.
- Added an online history modal on the host details page. Clicking a host's online/offline status opens a 30-day checkerboard of that host's connectivity.
- Added the ability to filter hosts by platform label and disk encryption status at the same time. Disk encryption status rows on Controls > OS settings now link to the host list filtered by both.
- Added a `populate_end_users` query parameter to the "List hosts" API endpoint, which includes each host's end users, with their identity provider (IdP) details and other emails, in the response.
- Added `disk_encryption_enabled` to each host in the response of the list hosts endpoint (`GET /api/v1/fleet/hosts`).
- Added a `--cpu-quota` flag to `fleetctl package` to set the systemd `CPUQuota` enforced on fleetd in Linux packages (deb, rpm, pkg.tar.zst). The default remains 20%.
- Allowed setting the `extensions_autoload` flag in agent options `command_line_flags`. It's still rejected when the `extensions` key is also configured, because fleetd uses that flag to load the extensions it manages.
- Added support for hosts running AMD Ryzen AI Developer Platform, a Debian-based Linux distribution.

### Security Engineers
- Added hardware attestation (ACME) for iPhones and iPads assigned to Fleet in Apple Business. With `apple_require_hardware_attestation` on, devices with an A11 Bionic chip or later running iOS or iPadOS 16 or later prove their hardware matches a known Apple Business record when they enroll. Hosts already enrolled with SCEP move to ACME on their next certificate renewal, and older devices keep enrolling with SCEP.
- Added the ability to limit enrollments to only automated (DEP) Apple Business device enrollments.
- Added the `mdm.apple_one_time_enroll_secrets` server configuration option, which uses one-time enrollment secrets delivered in the fleetd configuration profile for macOS hosts instead of shared enrollment secrets.
- Added vulnerability detection for Go binaries in software inventory, using the Go vulnerability database (https://vuln.go.dev).
- Added the Go module path and Go toolchain version to Go binaries in software inventory. Go binaries now show a Go icon, and their version includes the toolchain they were built with (for example, `v0.21.1 (go1.26.1)`).
- Added `signature_information` with `executable_path` and `executable_sha256` for each Mach-O executable installed by a Homebrew formula under its keg's `bin` and `sbin`, for use with Santa binary rules. Requires an updated fleetd.
- Added multi-signal detection to the `ai_tools` fleetd table, so AI agents that aren't recognized tools (homegrown agents and CrewAI, AutoGen, or LangChain harnesses) are reported instead of being missed. Two new columns, `confidence` and `evidence`, show how certain each detection is and which signals produced it. Hosts may report more `agents` rows than before as a result.
- Added TOML and YAML MCP config parsing to the `ai_tools` table, so MCP servers declared by Grok, Codex, Hermes, and the OpenClaw family are now reported. An MCP server that stores an `Authorization` value in its config is now flagged with the existing `plaintext_secret` risk flag.
- Added support for end users to create their own BitLocker startup PIN from the **My device** page, so a Windows user without local admin rights can satisfy a fleet that requires one. The PIN is stored encrypted, handed to the host's agent exactly once, and cleared. Fleet never shows it back to the end user or to an admin.
- Updated CIS Windows 10 Enterprise benchmark policies from v4.0.0 to v5.0.0.

### Bug fixes and improvements
- Changed the request certificate API to be closed by default: a request signed with a host identity certificate must name the end user recorded for that host, and IdP credentials are only accepted for an allowlisted introspection endpoint. Deployments that relied on unbound device requests or unlisted endpoints must set `server.allow_request_certificate_any_idp` (`FLEET_SERVER_ALLOW_REQUEST_CERTIFICATE_ANY_IDP`) to restore the previous behavior, or configure the new `integrations.certificates_*` settings.
- Added `integrations.certificates_idp_introspection_urls` and `integrations.certificates_idp_client_ids`. Once URLs are configured, IdP credentials are required on every request and the endpoint must be listed. The client ID list optionally constrains the client ID.
- Added `integrations.certificates_disable_host_end_user_binding` to turn off the host end user check per server. The check compares both the CSR email and UPN against the recorded identity and fails closed when the host has none.
- Changed certificate requests to require the CSR UPN to equal the CSR email or its local part, compared case-insensitively, and to reject a CSR with more than one UPN.
- Removed the deprecated `GET /api/v1/fleet/mdm/apple/commandresults` endpoint (deprecated since Fleet 4.40). Use `GET /api/v1/fleet/commands/results` instead.
- Removed the `X-Client-Cert-Serial` certificate authentication path from the device API. iOS and iPadOS hosts authenticate with the host UUID in the URL, which is what Fleet's self-service Web Clip profile uses.
- Limited `GET /software/versions` to 10,000 results per page when no pagination parameters are specified, so the request no longer times out on large software inventories. Use `page` and `per_page` (or `after`) to retrieve the rest.
- Changed software installer downloads by URL to honor Fleet's outbound network settings. Fetching installers from a private address needs `server_allow_private_network_integrations`. Fetching them from a loopback address, including through a loopback proxy, needs `server_bypass_network_blocking`.
- Restricted custom variables (`$FLEET_SECRET_*`) in host name templates to global admins, maintainers, and GitOps users.
- Made authorization errors consistent across the host MDM endpoints. Lock, unlock, wipe, clear passcode, Recovery Lock and managed local account password rotation, MDM command cancellation, and turning off MDM now return a `404` instead of a `403` when the target host is not in one of the caller's fleets.
- Reduced Fleet server CPU spent on HTTP route matching. Requests are now matched by a path trie in front of the existing router, instead of testing a regular expression against each route in registration order.
- Reduced database writer load from scheduled reports that store results: when a host's results haven't changed since they were last stored, Fleet no longer rewrites them and instead updates their "last fetched" time in batches, roughly once an hour.
- Added the `osquery_max_concurrent_query_report_reads` (default 40 per Fleet server) and `osquery_max_concurrent_query_report_writes` (default 20 across all Fleet servers) Fleet server configuration options, which cap how many osquery log requests check against and write to stored report results at once. Requests over a cap skip storing report results instead of queueing on the database, which kept Fleet servers from running out of memory when the database was slow.
- Reduced database lock contention on stored report results when hosts send results, when the cron removes rows over the report cap, and when reports are edited or deleted (including through GitOps). This could stall writes to the database during spikes.
- Sped up decoding of NVD and OSV vulnerability feed files during vulnerability processing by streaming them with `encoding/json/v2`.
- Improved SQL validation in the query editor. Syntax errors now report the line and column where the problem is, instead of a generic message, and a query that isn't a SELECT statement is reported as such, since osquery only runs SELECT statements. Valid queries that were previously reported as syntax errors, including those that use `CROSS JOIN` after an aliased table, now save.
- Improved API error messages for malformed request bodies. Most endpoints previously returned a generic `json decoder error` for any problem in the body, and now describe what was wrong, for example `invalid value type at 'mdm.ios_updates.update_new_hosts': expected bool but got string`. Incomplete or truncated JSON is now reported as `unexpected end of JSON input`.
- Updated the wording of JSON type validation errors returned by the API (for example, when applying GitOps or spec files with a value of the wrong type). Array elements are now identified by index, so a bad value in the first spec is reported at `specs.0.name` rather than `specs.name`.
- Made API error responses consistent in how much of the underlying error they include, and added a correlation ID to error responses so the matching server log line can be found.
- Improved Apple MDM push reliability: APNs notifications are now sent with a 30-day expiration and the documented MDM headers, so Apple stores and retries delivery to offline devices.
- Added an hourly cleanup that deletes Windows MDM enrollments whose host was deleted more than 30 days ago without the device re-enrolling, or that were superseded by a newer enrollment for the same host and not updated for 30 days (configurable with `FLEET_MDM_WINDOWS_ENROLLMENT_RETENTION`), along with their queued commands, results, and responses.
- Changed the fleetd Windows MSI to configure orbit through per-service environment variables instead of command-line flags, so downgrading orbit to a version that predates a newer setting no longer prevents the "Fleet osquery" service from starting.
- Improved Windows MDM host linking so the hardware serial a device reports cannot associate its enrollment with a host that a different device's enrollment already manages. Re-enrolling the same device is unaffected.
- Added a warning log when a Windows host enrolls in MDM presenting an MDM hardware ID (`HWDevID`) already held by a different host, which takes over that host's enrollment and leaves it unmanaged while still reporting MDM as on. Enrollment behavior is unchanged.
- Reworked the Apple Recovery Lock flow to never move to "Verified" before a successful VerifyRecoveryLock command has been received.
- Stopped sending AccountConfiguration commands to Apple devices during AB migration, because they have no effect.
- Restructured the DEP enrollment endpoint's checks to provide better errors on failed enrollments.
- Stopped returning internal error details to osquery and Orbit when enrollment fails. Agents now receive only the stage that failed, and the underlying reason is logged by the Fleet server.
- Changed agent requests whose HTTP message signature can't be verified to receive a uniform, generic error response. Details are logged on the server instead of being returned to the caller.
- Changed the Android enrollment token endpoint to verify the enroll secret before any other checks, so requests with an invalid secret always get the same response.
- Improved input validation for the BYOD (/enroll) end user authentication flow.
- Improved logging and error handling on SCEP endpoints so clients always get a properly formatted response.
- Added a default 60 second timeout to Fleet's outbound HTTP client helper, so a new caller that does not specify a timeout is bounded by default. Existing integrations keep the timeout behavior they had before this change.
- Added timeouts to the `maintained_apps_auto_update` and `cleanup_unused_software_installers` cron jobs.
- Added a time cap to APNs push crons so they no longer hold the cron for long periods.
- Removed the startup health check (`/services/collector/health`) from the native Splunk HEC log destination, so Fleet no longer fails to start when only the HEC event endpoint is reachable.
- Increased the timeout of the fleet-mcp single-host live query request so it waits as long as the Fleet server (`FLEET_LIVE_QUERY_REST_PERIOD`) instead of failing after 30s.
- Renamed the "Role" column to "Permissions" on the Settings > Users and fleet Users pages, and added a badge showing how many API endpoints an API-only user is restricted to.
- Updated the Settings > Users table so the actions dropdown only appears on row hover and clicking anywhere else in a row opens that user's edit page.
- Changed the Hosts page's User email column to show a host's IdP username (or first Chrome profile email if there's no IdP username) instead of a generic "N users" label, with a `+N` count and tooltip listing any additional emails.
- Updated the host device URL API to return the My device (self-service) page URL for iOS/iPadOS hosts, and added the "My device" link on the host details page for them.
- Hid the "My device" link on the host details page for Android and ChromeOS hosts, which have no My device page, and made the host device URL API return a clear error for those platforms.
- Changed the "My device" page heading back to "My device" instead of the end user's full name.
- Added a "Retrying" status to a host's OS settings for Android certificates that Fleet is automatically retrying after a failed install. The status shows the error reported by the host and which attempt Fleet is on. Previously these certificates showed as "Enforcing" with no sign that an attempt had failed.
- Added a disabled "Resend" button with a tooltip for Android configuration profiles on the host's Controls tab, explaining that Android hosts sync configuration profiles automatically (unlike certificates, which can still be resent).
- Added a "Resend" button for Android certificates stuck delivering to a host, instead of only when failed or verified, so a stuck certificate can be retried from the UI.
- Changed adding a Windows Fleet-maintained app to a fleet that already has a different Fleet-maintained app on the same software title (for example, the x64 and ARM64 builds of Firefox Nightly, which both register as "Firefox Nightly") to fail with a clear "Only one of X or Y can be added to the same fleet" error instead of a duplicate-installer error.
- Cleaned up the error toasts shown when a Microsoft Graph credential save fails and when an MDM unenroll returns "already off": the main line is a short, static message and the full backend response is available in the expandable raw-response panel, so the same long reason no longer appears twice in the same toast.
- Added a note to the delete label modal explaining that labels used as custom software targets can't be deleted until they're removed from those targets.
- Added helper text under the enrollment link in the "Add hosts" modal's iOS/iPadOS tab noting that the link must be opened in Safari to work.
- Removed the redundant "Enrollment instructions" sub-header from the Android and iOS/iPadOS panels of the Add hosts modal.
- Added tooltips to the configuration profile action buttons (details, edit, download, delete) so hovering shows the button's action.
- Added a tooltip explaining why the "Last fetched" column on the Hosts page reads "Never" for a host that has checked in but has not yet reported vitals.
- Added a tooltip showing the full label name when labels are truncated in the "Filter by platform or label" dropdown on the Hosts page.
- Updated the report **Interval** help text and the reports guide to explain that the interval counts time the host is awake, so hosts that sleep may report less often.
- Added third-party integration usage statistics (log destinations, webhook, SSO, SCIM, ticket destination, certificate authority, and Apple account provisioning configuration) and renamed the `googleWorkspaceConfigured` key to `idpGoogleWorkspaceConfigured`.
- Added `numPoliciesAutomationEnabledSoftware` to Fleet's anonymous usage statistics, and extended the reported Fleet-maintained apps with whether each is covered by a patch policy and whether that policy has a software automation.
- Upgraded OpenTelemetry dependencies to the latest stable release (otel 1.46.0, otel/log 0.22.0). Telemetry now reports against semantic conventions schema 1.43.0 instead of 1.41.0.
- Updated Go to 1.27.1.
- Fixed `fleetctl gitops` failing with `Error 1213: Deadlock found` on instances with many hosts and policies. Re-applying unchanged policies no longer cleans up their membership, the orphaned membership cleanup no longer locks every membership row of a policy, and the post-apply cleanup now retries on deadlocks.
- Fixed hosts keeping results for policies no longer in scope for them (for example, after leaving a policy's label) when async host processing is enabled for policy membership, which inflated their failing policy counts.
- Fixed re-applying scripts with `fleetctl gitops` or the batch scripts API blocking or deadlocking with queued script runs. Unchanged scripts are now skipped.
- Fixed re-applying software with `fleetctl gitops` or the batch software API blocking or deadlocking with queued software installs in other fleets.
- Fixed re-applying scripts to a fleet via GitOps blocking every host's policy result reporting until the apply finished.
- Fixed `fleetctl` giving up with "timeout awaiting response headers" on requests that take the server more than 45 seconds, such as `fleetctl gitops` applying software to a fleet with many Fleet-maintained apps.
- Fixed `fleetctl gitops` temporarily clearing all Volume Purchasing Program and Apple Business Manager token/fleet assignments when the configuration referenced a fleet created in the same run. A run that failed mid-apply left the assignments permanently removed.
- Fixed the report page, report results API, and host details Reports tab failing when a host stored a result larger than MySQL's sort buffer (256 KB by default), and truncated long result values in host details Reports tab tooltips.
- Fixed reports keeping old stored results after a SQL change (or another change that clears results) when the request ended before the old results were deleted.
- Fixed an issue where characters in a Fleet variable's value could change what a script does. Values are now defined at the top of the script, or escaped for Python, and read as literal text.
- Changed Fleet variables in shell scripts so they're no longer substituted inside single quotes or a quoted heredoc.
- Added support for Fleet variables inside string literals in Python scripts. They aren't supported inside raw (`r"..."`) or bytes (`b"..."`) literals.
- Fixed Windows MDM client certificates being issued with half their advertised lifetime and with the renewal window already open, which made Windows retry certificate renewal every few days from the moment a host enrolled. Certificates are now valid for a full 365 days from issuance, with a one-day allowance for device clock skew, and Windows opens the renewal window 180 days before expiry.
- Fixed Windows automatic enrollment through Microsoft Entra ID during the out-of-box experience (OOBE) stalling at the Terms of Use step. Fleet now accepts the `ms-aadj-redir://auth/mdm` redirect URI that Windows sends in that flow, while still rejecting script-executing schemes.
- Fixed Windows automatic enrollment (Microsoft Entra ID) re-downloading Microsoft's JWT signing keys and leaking a background refresh goroutine. The keys are now fetched once and refreshed hourly.
- Fixed Windows SCEP certificate enrollment failing when a Fleet variable or custom host vital substituted into a profile's `SubjectName` resolved to a value containing an X.500 special character, such as an IdP username like `user+idp@example.com` or an asset tag containing a comma. Fleet now quotes those subject name values so Windows reads each one as a single attribute value, which also prevents a substituted value from adding attributes to the issued certificate's subject.
- Fixed APNs push requests potentially blocking MDM command delivery indefinitely when Apple's push service stalls a response.
- Fixed a bug where a macOS host could be left at the setup experience screen indefinitely when an App Store app install failed.
- Fixed a bug where moving an Apple host between fleets that have an identical configuration profile, while that profile was still pending on the host, left the profile stuck in "Enforcing (pending)" for up to 2 hours.
- Fixed iOS in-house app installs failing on deployments that set `mdm.apple_server_url`, by building both the `InstallApplication` manifest URL and the `.ipa` download URL inside that manifest from the hostname Apple devices reach Fleet on rather than from `server_settings.server_url`.
- Fixed an issue where iOS/iPadOS hosts on Fleet Free never received host vitals because the `apple_mdm_iphone_ipad_refetcher` and `apple_mdm_iphone_ipad_reviver` crons only ran on Fleet Premium.
- Fixed duplicate refetch commands piling up for offline iOS and iPadOS hosts. Fleet now keeps a single outstanding refetch per host and only clears its tracking once the command is answered or no longer queued.
- Fixed an issue where apps the end user installed themselves appeared in the software inventory of a manually enrolled (BYOD) iPhone or iPad after an app was installed through Fleet.
- Fixed VPP apps not scoped to a host still showing up in Self-service when the same app was also added in the fleet for another platform.
- Fixed an issue where the `com.apple.configuration.app.managed` declaration became blocked when it previously was not.
- Fixed Fleet discarding a host's stored disk encryption key when the agent reported a disk encryption error. Fleet now records the error and keeps the key it already has.
- Fixed Fleet reporting a Windows host as "Verified" for disk encryption when its BitLocker startup protectors had been deleted while protection stayed on.
- Fixed Fleet offering a Linux host's previous disk encryption key after detecting that the escrowed LUKS key slot was removed. `GET /api/v1/fleet/hosts/:id/encryption_key` no longer falls back to the archived key for Linux hosts, and the host details page only offers the key while disk encryption is "Verified".
- Fixed Linux hosts prompting the end user for their disk encryption passphrase twice when "Create key" was selected again before the first request finished. Fleet now points the end user at the pop-up already open and says how long to wait.
- Added a `status` field to `POST /api/fleet/orbit/luks_data` and the `linux_escrow_status` capability so fleetd can report progress on a Linux disk encryption key request, letting Fleet tell a prompt still open from one the end user dismissed.
- Fixed Linux wipe so directory-service users (SSSD/LDAP/Kerberos/AD) can't log back in while the wipe is running. The wipe now denies logins through pam_nologin, which reaches accounts that have no `/etc/shadow` entry for `passwd -l` to lock.
- Fixed the Linux wipe being killed and CPU-throttled by fleetd. The wipe now runs in its own systemd unit instead of as a background child of `orbit.service`, so it survives a fleetd restart, isn't limited to 20% of one CPU, and logs to the journal under `fleet-wipe`.
- Fixed the host IdP device mapping (`device_mapping` source `mdm_idp_accounts`, host search, and `$FLEET_VAR_HOST_END_USER_EMAIL_IDP`) not updating when an IdP user's `userName` is changed from one email address to another via SCIM, which caused `GET /hosts` and `GET /hosts/:id` to report different IdP usernames for the same host. Renames to or from a `userName` that is not an email address leave the device mapping unchanged.
- Fixed fleet-level admins and maintainers receiving a permission error when clearing passcodes on hosts in their fleets.
- Fixed the `POST /api/v1/fleet/policies/:policy_id/reset` and `GET /api/v1/fleet/policies/:policy_id/automation_activities` endpoints returning 404 (they were only routed under `/api/2022-04` and `/api/latest`).
- Implemented the documented `host_id` query parameter on `POST /api/v1/fleet/policies/:policy_id/reset` so it resets only that host's result and refreshes the policy's passing/failing counts right away. Previously the parameter was ignored and every host's result was reset.
- Fixed "Reset policy" from a single automation run on the policy details page resetting the policy for all hosts instead of only that run's host.
- Fixed `POST /api/v1/fleet/spec/policies` (GitOps) accepting a `script_id` on a global policy or a script from another fleet, and returning a database error instead of a clear message for a nonexistent script.
- Fixed the `sync_enrolled_host_ids` cron job failing with "expected slice but got int" when the license host limit is enforced (`FLEET_LICENSE_ENFORCE_HOST_LIMIT`), which prevented the Redis set of enrolled host IDs from resyncing with the database.
- Fixed a "Could not transfer hosts" error when transferring more than 65,535 hosts to a fleet at once. The hosts were reassigned, but the follow-up work on the transfer (MDM profile, Android device, and Apple Business Manager bookkeeping) exceeded MySQL's per-statement placeholder limit and failed the request. Those queries are now batched.
- Fixed a spike in database read traffic when transferring large numbers of hosts between fleets.
- Fixed automatic expired host cleanup failing on large backlogs by limiting each cron run to a bounded batch of hosts.
- Fixed a slow query on the host details Software tab (`GET /api/v1/fleet/hosts/{id}/software`) for hosts with many historical software install/uninstall attempts.
- Fixed the software versions list (`GET /api/v1/fleet/software/versions`, and the deprecated `GET /api/v1/fleet/software`) ignoring `min_cvss_score` when `max_cvss_score` was also set, so severity filters like "High" or "Critical" returned lower-severity software.
- Fixed vulnerability detection for Arch Linux and Omarchy hosts by ingesting pacman packages with the epoch removed and the package release split out of the version (e.g. `2:9.0.1-4` is now version `9.0.1`, release `4`), matching how RPM packages are ingested.
- Fixed Go binaries, npm packages, and other non-package software being compared against Linux distro advisories when their name matched a `deb` or `rpm` package.
- Fixed Homebrew executable hashes lingering after a binary is removed from a keg. A host now reports the keg's full set of executables on every run, so an entry disappears within one detail query interval.
- Fixed the `ai_tools` table reporting an agent as running when an unrelated process merely ended with the same characters as its binary name, and naming MCP servers after a fragment of an inline script when one was launched via `node -e`.
- Fixed server-side macOS package version detection resolving the wrong software version for some packages, depending on the structure of their distribution XML.
- Fixed the macOS "i1Profiler up to date" patch policy always failing on hosts running the current version of i1Profiler. The cask version tracks `CFBundleVersion` (`3.8.7.19247`) while i1Profiler.app reports only a marketing version (`3.8.7`) as its `CFBundleShortVersionString`, so the generated policy now compares `CFBundleVersion`.
- Fixed Fleet Desktop showing a phantom "Update" button for the Raspberry Pi Imager Fleet-maintained app by aligning its catalog version with the "v" prefix macOS reports in `bundle_short_version`.
- Fixed patch-when-closed skips showing as "Failed" on Host details > Software > Library. The install status now renders as "Patch skipped" (grey), matching the policy status page, and the host software response carries a `skipped_install` boolean so clients can distinguish a deferred install from a real failure. Also fixed the install status cell tooltip rendering off to the left when the table only has one row.
- Fixed the policy automation Details modal and Automation runs table's Details column showing no explanation when a software install failed because its pre-install query returned no rows.
- Fixed upcoming activities for software installed during setup experience showing as user-initiated instead of Fleet-initiated.
- Fixed GitOps failing to add a macOS setup assistant for a fleet that isn't a default fleet for an Apple Business Manager (ABM) token when more than one ABM token is configured, and added a warning when a saved setup assistant won't take effect until the fleet is tied to an ABM token.
- Fixed GitOps runs failing with "Server private key must be configured" when no certificate authorities are configured.
- Fixed `fleetctl generate-gitops` failing on Fleet Free when Apple MDM is turned on.
- Fixed `fleetctl generate-gitops` adding every `.sh` and `.py` package to the macOS setup experience instead of only the selected ones.
- Fixed `fleetctl generate-gitops` dropping a script package's Linux setup experience selection from the YAML it generates.
- Fixed `fleetctl gitops` reporting configuration profile validation errors without saying which profile failed. Errors from applying custom settings now include the offending profile's name.
- Fixed `fleetctl gitops` reporting script validation errors without saying which script failed. Errors from applying scripts now include the offending script's filename.
- Fixed `PATCH /api/v1/fleet/config`, `fleetctl apply`, and `fleetctl gitops` accepting a `host_expiry_window` of 0 or less while `host_expiry_enabled` is true. The request now fails with a 422, matching the fleet-level setting.
- Fixed an issue where a Windows configuration profile could be added with an empty or whitespace-only name, for example by uploading a file named `.xml`. Adding or editing a profile with such a file is now rejected with "Profile name can't be empty."
- Fixed GitOps mode blocking edits to a manual label's host membership. Hosts can now be added and removed in the Fleet UI while the label's name and description stay managed in YAML.
- Fixed Windows MDM commands enqueued within the same second being delivered to the device in an arbitrary order rather than the order they were created.
- Fixed Android MDM lock, wipe, and clear passcode returning a 500 instead of a 404 when the device no longer exists in the Google Android Enterprise account.
- Fixed Android configuration profiles keeping the previous failure message after the profile transitioned back to verified. The stored detail is now cleared when a profile is verified, so a resolved failure no longer shows a stale error.
- Fixed an Android certificate up for renewal reporting the error from its previous install. The renewal now starts clean, so a host's OS settings no longer show an old error against a renewal that is proceeding normally.
- Fixed the Fleet UI so it renders correctly when the server is configured to send a Content-Security-Policy, by passing the server's nonce to runtime style and script injection.
- Fixed the transfer hosts endpoint returning inconsistent responses for a host outside the requester's fleet versus one that doesn't exist.
- Fixed the free-tier premium license error naming a renamed field by its old name when the request used the new one.
- Fixed the error message shown when a host can't be deleted because its Apple Business assignment couldn't be verified. Internal error details are no longer appended to the message.
- Fixed fleet-mcp rejecting Linux-only osquery tables for hosts on Linux distributions it did not recognize (Flatcar, CoreOS, NixOS, Arch, openSUSE, and others).
- Fixed `fleetctl` installed with `npm install -g fleetctl` exiting successfully without running when the binary download was interrupted or the binary was killed by a signal. The wrapper now reports the error and exits non-zero, and no longer re-downloads the binary on every run on Windows.
- Fixed the host page becoming unresponsive after closing the script run details modal with the X or Escape and opening it again.
- Fixed the Software page redirecting to an invalid fleet ID of `-1` when leaving the Library tab.
- Fixed fleet-scoped users being bounced off the Software > Library tab before their fleets finished loading.
- Fixed policy automations UI so the software, script, and configuration profile dropdowns allow typing to search again.
- Fixed the **My device > Self-service** list sorting by the software's underlying name (package identifier or script filename) instead of its display name.
- Fixed the software list in **Controls > Setup experience > Install software** sorting by the software's underlying name instead of its display name, and made its search match the display name as well as the underlying name.
- Fixed an issue where it was not possible to clear the Apple Account Provisioning in the UI once configured.
- Fixed clicking the info icon on a configuration profile for a platform that no longer has MDM enabled showing a generic error.
- Fixed the "Hosts online" empty state on the fleet dashboard to hide the Turn on button for fleet admins when the setting is disabled at the org level, since only a global admin can lift the org-level gate. Fleet admins now see "Ask an admin" copy in that case, and global admins keep the Turn on button, which now routes to the org-level setting instead of the fleet setting.
- Fixed the unsaved changes prompt appearing when switching tabs on the SSO settings page after a field had been edited and then returned to its original value.
- Updated the forms on the Authentication (SSO) settings page to follow Fleet's form validation patterns, so errors surface on the field being edited instead of on fields that haven't been filled in yet.
- Fixed the "Show MDM commands" toggle on the host details activity feed resetting to off after a page refresh.
- Fixed activity feed copy for role changes made via just-in-time (JIT) provisioning to use passive voice (e.g., "user@example.com was removed from the Engineering fleet via just-in-time (JIT) provisioning") instead of showing the user as both actor and subject.
- Fixed tooltips rendering unstyled inside a table's actions dropdown.
- Fixed long host locations overflowing on the host details page and truncating without a tooltip on the My device page.
- Fixed the disk encryption and bootstrap package status filter dropdowns on the Hosts page rendering larger text than the other filters, which also wrapped long options onto two lines.
- Fixed a horizontal scrollbar on the Labels table for read-only roles, which sat on the table card's bottom border and left its bottom-right corner open.
- Fixed the text cursor appearing after the placeholder or selected value when focusing a searchable dropdown.
- Fixed custom host vitals rendering taller than other host vitals on the host details page.
- Fixed the vertically misaligned pass/fail status icon in the policy results table when running a live policy query.
- Fixed the MDM command details modal showing a text area resize handle.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.93.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-10-02">
<meta name="articleTitle" value="Fleet 4.93.0 | macOS app patching, Android zero-touch enrollment, Windows admin password rotation, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.93.0-1600x900@2x.png">
