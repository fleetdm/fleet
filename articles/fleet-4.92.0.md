# Fleet 4.92.0 | Android commands, Windows Autopilot hosts, custom FileVault, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.92.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.92.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Android: run any command](#android-run-any-command)
- [Android: scope self-service software](#android-scope-self-service-software)
- [Fleet assignment for Windows Autopilot hosts before enrollment](#fleet-assignment-for-windows-autopilot-hosts-before-enrollment)
- [Custom macOS FileVault](#custom-macos-filevault)
- [Cancel upcoming lock/wipe commands](#cancel-upcoming-lock-wipe-commands)
- [Policy automations: resend a configuration profile](#policy-automations-resend-a-configuration-profile)
- [Filter vulnerability exposure by severity](#filter-vulnerability-exposure-by-severity)
- [Require SSO for Fleet Desktop](#require-sso-for-fleet-desktop)

### Android: run any command

IT admins can now send any command from the [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand) to an Android host via `fleetctl mdm run-command` or [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command), in addition to any Apple (macOS, iOS, iPadOS) or Windows command. This unlocks automations for commands Fleet doesn't have built-in UI/API action for yet, like [requesting device info](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.REQUEST_DEVICE_INFO) or [relinquishing ownership](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices/issueCommand#CommandType.ENUM_VALUES.RELINQUISH_OWNERSHIP).

IT admins can see Android commands results on a host's **Host details** page or via `fleetctl get mdm-commands` and `fleetctl get mdm-command-results`, the same way we already could for Apple and Windows hosts. This makes it easier to troubleshoot a command sent to an Android host, whether it came from the Fleet UI, GitOps, or a custom command sent through the API.

See an example Android command, like rebooting a host, in the [MDM commands guide](https://fleetdm.com/guides/mdm-commands#examples).

GitHub issues: [#23232](https://github.com/fleetdm/fleet/issues/23232), [#33158](https://github.com/fleetdm/fleet/issues/33158)

### Android: scope self-service software

_Available in Fleet Premium_

IT admins adding software to Android hosts' managed Google Play Store can now target hosts using labels, the same targeting options already available for macOS, Windows, and Linux software. This makes it possible to make an app available in self-service on a more specific set of Android hosts instead of every host in a fleet. Learn how to [add an Android app](https://fleetdm.com/guides/install-app-store-apps#google-play-android).

GitHub issue: [#33062](https://github.com/fleetdm/fleet/issues/33062)

Scoping an different Android app configuration to specifc hosts is [coming soon](https://github.com/fleetdm/fleet/issues/47904).

### Fleet assignment for Windows Autopilot hosts before enrollment

_Available in Fleet Premium_

Windows Autopilot now show up in Fleet as "Pending" hosts, along with their Autopilot group tag, before they ever enroll. IT admins can manually transfer a pending host to the right fleet from the **Hosts** page, or build an automation on top of Fleet's API to do it before the host ever enrolls, instead of waiting for it to land in "Unassigned" first.

GitHub issue: [#43481](https://github.com/fleetdm/fleet/issues/43481)

### Custom macOS FileVault

_Available in Fleet Premium_

IT admins can now add custom disk encryption (FileVault) settings for macOS. This makes it possible to upload a [custom `FDEFileVaultOptions` configuration profile](https://fleetdm.com/guides/custom-disk-encryption-profiles), for example, to defer FileVault until the next login, or allow a third-party tool such as Xcreds to enforce FileVault, while Fleet still escrows the recovery key.

GitHub issue: [#48654](https://github.com/fleetdm/fleet/issues/48654)

### Cancel upcoming lock/wipe commands

IT admins can now cancel a pending lock, wipe, clear passcode, or enable lost mode command on Apple (macOS, iOS, iPadOS) hosts before a host receives it, from the host's **Host details > Activity > Upcoming > MDM commands** or via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#cancel-hosts-pending-mdm-command). Even if the host runs a lock command before the cancellation reaches it, Fleet still shows the unlock PIN, once the result comes back.

Lock on Windows/Linux and wipe on Linux run as scripts, but can be canceled via **Host details > Activity > Upcoming**, or via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#cancel-hosts-upcoming-activity). Wipe on Windows and lock, wipe, and clear passcode on Android aren't cancelable yet.

GitHub issue: [#43181](https://github.com/fleetdm/fleet/issues/43181)

### Policy automations: resend a configuration profile

_Available in Fleet Premium_

IT admins can now automatically resends a configuration profile when a host fails the policy, the same way Fleet already supports automatically [running scripts](https://fleetdm.com/guides/policy-automation-run-script) and [installing software](https://fleetdm.com/guides/automatic-software-install-in-fleet). This makes it possible to build automated fixes for configuration drift, like renewing a certificate, fixing a Wi-Fi profile, or re-enforcing a CIS benchmark setting.

GitHub issue: [#40637](https://github.com/fleetdm/fleet/issues/40637)

### Filter vulnerability exposure by severity

_Available in Fleet Premium_

The vulnerability exposure chart on the dashboard can now be filtered by severity (CVSS score), giving Security Engineers more control over what the chart shows. We can also set the default severity filter for the whole organization by configuring [`cvss_min`/`cvss_max` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#features).

GitHub issue: [#47326](https://github.com/fleetdm/fleet/issues/47326)

### Require SSO for Fleet Desktop

_Available in Fleet Premium_

IT admins can now require end users to sign in with SSO before they can access self-service in Fleet Desktop for macOS, Windows, Linux, or iOS/iPadOS hosts. In **Organization settings > Fleet Desktop**, turn on the new "End user authentication" setting, or set [`fleet_desktop.sso_enabled` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-desktop), to add an extra layer of authentication in front of Fleet Desktop.

GitHub issue: [#47116](https://github.com/fleetdm/fleet/issues/47116)

## Changes

### IT Admins
- Added support for running custom Android MDM commands via the Fleet API.
- Added support for listing and viewing results of Android custom MDM commands via the API and fleetctl CLI.
- Added support for automatically installing in-house apps (`.ipa`) on iOS and iPadOS hosts when they enroll into Fleet.
- Added Windows devices registered in a connected Microsoft Entra tenant's Autopilot registry to Fleet as pending hosts before they enroll; they become regular hosts on enrollment without creating a duplicate.
- Added support for resending a configuration profile on policy failure as part of a policy automation.
- Added collection of additional Android host vitals from AMAPI status reports (USB debugging, passcode set, Google Play Protect, encryption status, manufacturer, security update version, kernel and bootloader version, software update status, API level, security posture, and per-SIM phone numbers), returned by the get host endpoints for Android hosts.
- Added a QR code for the enrollment link on the Android and iOS/iPadOS tabs of the **Add hosts** modal, so the enrollment flow can be started by scanning instead of copying the link to a device.
- Added the `osquery.config_in_memory_cache` server configuration option and disabled the in-memory cache of the osquery config's scheduled-report section by default; set it to `true` to re-enable the cache.
- Added `server.endpoint_request_size_overrides` to configure a max request body size per API endpoint, with the largest of the endpoint's default and the override winning.
- Added `mdm.is_personal_enrollment` to host API responses, reporting whether the last MDM enrollment Fleet recorded for the host was personal (BYOD). Unlike `mdm.enrollment_status`, it is not cleared when the host unenrolls.
- Added `mdm.bootstrap_token_escrowed` to the get host and get host by Fleet Desktop token API responses, so admins can see whether Fleet has escrowed a macOS host's bootstrap token without querying the database directly.
- Added support for `display_name` on Fleet-maintained apps in GitOps.
- Added the ability to sort the versions table by version on the Software details page.

### Security Engineers
- Added per-platform disk encryption settings: enforcement and key escrow for macOS, enforcement for Windows, and key escrow for Linux.
- Added a new "End user authentication" setting (`fleet_desktop.sso_enabled`, Fleet Premium) that requires end users to sign in via single sign-on (SSO) before accessing Fleet Desktop's "My device" page.
- Added a severity (CVSS score) filter to the Vulnerability exposure dashboard chart, which now requires Fleet Premium, and updated the severity filter on the **Software**, **Host details**, and **My device** pages so the min and max score inputs appear only when custom severity is selected.
- Added an audit activity when a deprovisioned SCIM user has no email to match a Fleet account, so operators can spot accounts that may remain active.
- Added the ability to cancel a pending Apple MDM lock, wipe, clear passcode, or enable lost mode command before the host receives it, via `DELETE /api/v1/fleet/hosts/:id/commands/:command_uuid`. If the host executes a canceled lock or wipe anyway, Fleet now restores the host's lock/wipe state (including the unlock PIN) when the result arrives.

### Bug fixes and improvements
- Reduced the size of the Fleet UI JavaScript bundle by ~89% (14.9 MB to 1.6 MB gzipped) by serving Fleet-maintained app icons as individual static files instead of embedding all ~1,100 of them in the bundle downloaded on every page load.
- Reduced the size of the Fleet UI JavaScript bundle by a further ~62% (1.6 MB to 632 KB gzipped) by loading each page's code when its route is opened rather than compiling every page into the bundle downloaded on first load.
- Reduced memory usage of the vulnerabilities cron by streaming NVD matches to the database in bounded chunks instead of holding every matched vulnerability in memory (8.5 GB → 2.1 GB peak on a 22.7M-match fleet), and inserts now start during matching so large fleets no longer exceed the vulnerability processing time limit.
- Reduced database load when processing osquery result logs by resolving scheduled query names in a single batch lookup instead of one query per result.
- Added an "Inactive" status with an explanatory tooltip to the Users table for accounts that haven't been used for 30+ days. Regular users are inactive when they haven't logged in (or had session activity) for 30+ days; API-only users are inactive when their token has made no API requests for 30+ days. Fleet now records each user's last login time in a new `last_login_at` field, reports last session activity in a new `last_activity_at` field, and returns a server-computed `status` field (`active`, `inactive`, or `no_access`) from the users API.
- Added an experimental WebSocket notification transport for fleetd agents (ADR-0011), disabled by default (enable with the `websocket.transport_enabled` server configuration): connected agents are pushed a "check now" notification when a live query targets them or when interval work (labels, policies, host vitals, refetch) is due, instead of polling `distributed/read` every 10 seconds.
- Added conditional request (etag) support to the osquery config endpoint (`/api/osquery/config`), behind the `osquery.config_etags` server option (`FLEET_OSQUERY_CONFIG_ETAGS`, default off). When it's enabled, agents that send an `etag` field in the request body receive the minimal `{"etag":"ok"}` response when their configuration is unchanged, reducing agent config bandwidth; agents that don't send the field see no change. While it's off, every config request is served exactly as before.
- Added an `osquery.redis_config_etags` server option (`FLEET_OSQUERY_REDIS_CONFIG_ETAGS`, default off, and requires `osquery.config_etags`): when both are enabled, config check-ins with a matching etag are answered directly from a Redis-backed ETag store, skipping the config build and its database reads entirely. Fleets with uniform configs share one ETag per fleet and platform; fleets with label-scoped reports use isolated per-host ETags invalidated whenever a host's label results are recorded. The short circuit fails open (any Redis error falls back to a full build) and is bypassed automatically for deployments with 2017 packs.
- Windows user-scoped configuration profiles (`./User/...`) are now held in "Pending" until a user signs in, instead of failing during setup. On hosts enrolled by a user (Windows Autopilot, Entra ID) that user releases the hold; on hosts enrolled by installing fleetd, any signed-in user does, and Windows applies the settings to whoever is signed in. User-scoped profiles that already failed on an earlier Fleet version are not resent automatically: after upgrading, resend them once (or edit the profile) and Fleet will deliver them when a user is signed in.
- Removing a Windows user-scoped configuration profile now waits for a signed-in user as well, instead of being reported as removed while the setting was still applied.
- Windows configuration profiles now retry up to 3 times before being marked "Failed", matching Apple. Retries cover profiles the host rejects and profiles whose Fleet-proxied SCEP certificate never arrives. A profile stays "Pending" while Fleet retries.
- Fleet no longer marks a configuration profile "Failed" when it briefly can't reach the NDES admin URL to fetch a SCEP challenge. The profile stays pending and Fleet tries again. Challenge failures that need an admin to act (invalid credentials, a full password cache, or an account without SCEP enroll permission) still fail the profile immediately with the same message as before.
- Escaped values interpolated into the conditional access Apple configuration profile so a value containing markup is carried as literal text.
- Fleet now clears dynamic labels and pending commands and software installs when an Android host re-enrolls. Manually assigned labels are preserved, and pending software installs are reported as failed. Past host activities for the host are also cleared unless `preserve_host_activities_on_reenrollment` is enabled.
- Increased Android certificate delivery retries from 3 to 7 and added exponential backoff between attempts.
- Improved SCIM user deactivation to more reliably deprovision the matching Fleet user.
- Moved the host's OS settings table out of the "OS settings" modal into a dedicated **Controls** tab on Host details and My device.
- Moved toast notifications to the bottom center of the screen so they no longer cover buttons in the bottom-right of pages and modals.
- Updated the macOS enroll modal's "Company-owned" label and helper text to match iOS/iPadOS and Android ("Company-owned (fully-managed)").
- Updated wipe modal for Apple and Windows hosts to call out deleting may remove the Wipe Pending status.
- Updated the message end users see when they take too long to sign in during MDM enrollment to say their session may have timed out, instead of a generic error.
- Added a descriptive message when a script produces empty standard output.
- Improved `fleetctl generate-gitops` to emit software titles in name order instead of the default `hosts_count` order.
- Improved the hosts report CSV export (`GET /api/v1/fleet/hosts/report`) so that exported cell values are treated as text by spreadsheet applications.
- Changed duration fields in Fleet server logs (e.g. `took`) to render as human-readable strings with time units (e.g. `"1.116187ms"`) instead of raw nanosecond numbers. Log pipelines that parse these fields numerically will need to be updated.
- Changed requests denied by an API-only user's endpoint restrictions to return a distinct 403 message ("endpoint not permitted for this API-only user"), logged at info level with the route and denial reason, so they can be distinguished from role-based permission denials.
- Enforced API-only endpoint restrictions on the debug routes so a restricted API-only token can no longer reach `/debug/*`.
- Removed the unused `jq` binary from the `fleetdm/fleet` Docker image to reduce the image's attack surface and prevent SBOM scanners from flagging `jq` CVEs.
- Deprecated `osquery_max_log_write_body_size` and `osquery_max_distributed_write_body_size` in favor of new configs.
- Deprecated `fleetdm/bomutils` docker image. Starting in 4.90.0, `fleetctl` does not use `fleetdm/bomutils` to generate `.pkg` fleetd installers.
- Updated the EPSS scores feed to download from its new canonical URL (`epss.empiricalsecurity.com`) instead of relying on the redirect from the old host (`epss.cyentia.com`).
- Improved outbound address filtering to cover the unspecified addresses (`0.0.0.0` and `::`), the deprecated IPv4-compatible IPv6 form, and IPv4 addresses reached through a NAT64 prefix.
- Made vulnerability host count updates recover automatically after an interrupted table swap.
- Added support for sending blank APNS pings to Apple devices.
- Improved parsing of Apple MachineInfo blobs during enrollment.
- Improved validation of the `order_key` parameter when listing software, rejecting sort keys that aren't supported instead of passing them through to the query.
- Improved validation of the `order_key` parameter on `GET /api/v1/fleet/activities` and `GET /api/v1/fleet/hosts/{id}/activities`, rejecting unsupported sort keys.
- Added MDM profile counts (Apple, Windows, Android configuration profiles and Apple DDM declarations) to usage statistics for troubleshooting.
- Added conditional config request support to `osquery-perf` simulated hosts, including stats for conditional requests and estimated bandwidth saved, for load-testing the above.
- Added a `useFormValidation` hook as the single source of truth for the documented form validation behavior, and applied it to the new user and new API-only user forms.
- Updated the DDM asset error message shown when the `Authentication` key is provided to explain that Fleet defaults to `MDM` authentication.
- Removed mention of osquery when viewing a DDM profile that is verifying.
- Removed the QR code from the "Add hosts" enrollment instructions for company-owned Android hosts, since fully-managed enrollment isn't done by scanning a code.
- Disabled the **Save** button in the end user migration workflow while a request is in flight.
- Documented why `safari_extensions` returns empty without Full Disk Access or a `uid` constraint (`users` JOIN/`CROSS JOIN`, or `WHERE uid = ...`), noted the `/Applications`-only scan limitation, fixed standard Safari inventory SQL to include a `users` CROSS JOIN, and replaced legacy `.safariextz` bash equivalents with modern Safari App/Web Extension paths.
- Fleet now restores BitLocker protection on Windows hosts that are encrypted but whose protection is off, where disk encryption is enforced. This includes hosts that require a startup PIN but do not have one yet, and hosts where an admin, third-party software, or a previous MDM forbade the TPM-only protector Fleet needs, which Fleet now clears. Such a host shows "Enforcing" while Fleet repairs it, and "Action required" with the reason in its disk encryption details only once fleetd reports it cannot.
- Fixed a Windows host's disk encryption status naming an action the end user cannot take. A host whose protection is off no longer asks them to create a BitLocker PIN, since Windows only offers PIN setup on a protected volume, and a repair waiting on a pending restart now asks for that restart on both the host details page and the end user's My device page.
- Fixed an issue where a second user signing in to a Windows device that was already enrolled via Autopilot would get stuck on the "Account setup" Enrollment Status Page for up to 3 hours.
- Fixed built-in and starter library Linux labels being too strict to match derived distributions, so hosts running Pop!_OS, Linux Mint, Zorin OS, Debian, older Fedora releases, Amazon Linux, and SUSE now appear in the Linux labels that apply to them.
- Fixed an upgrade failure where a MySQL "Prepared statement needs to be re-prepared" error (1615) aborted a database migration instead of being retried.
- Fixed osquery and orbit config endpoints returning HTTP 500 when a host references a recently deleted team by invalidating the Redis host cache on team deletion.
- Fixed the dashboard "Hosts enrolled" chart drill-down including pending hosts in the filtered host list, by adding an "Enrolled hosts" status filter (`status=enrolled` on the list hosts and count hosts endpoints) that excludes hosts pending MDM enrollment.
- Fixed pending hosts (Apple Business Manager, Windows Autopilot) showing a "Fetching fresh vitals" spinner and a "This host is offline. Please try refetching host vitals later." error on the host details page, even though nobody asked for a refetch.
- Fixed "Turn off MDM" being available again after it succeeded, which let an offline host be sent duplicate unenroll commands.
- Fixed an Apple configuration profile that was removed and then added back before the host came online being stripped from the host and reinstalled, instead of staying in place.
- Fixed ACME certificates deployed via user-scoped profiles not showing on the macOS host details page, and recorded them under the enrolled user's scope instead of the system keychain.
- Fixed an issue where a misleading detail was shown for pending/verifying DDM profiles.
- Fixed iOS/iPadOS refetch getting blocked when an online device acknowledged a refetch command before Fleet finished recording the command as sent.
- Fixed duplicate IdP device mapping ("2 users") shown for a host after it re-enrolled through ADE with end user authentication when its IdP username had previously been set manually. The enrollment now replaces the manually set mapping instead of adding a second one.
- Fixed the host details Vitals card replacing **Enrollment ID** with an empty **Serial number** after a personal (BYOD) Android host is unenrolled.
- Fixed Android MDM commands returning 500 instead of the actual error code from the Google Android Enterprise API.
- Fixed an issue where Homebrew cask metadata for a Fleet-maintained app (macOS) could reach the generated install/uninstall scripts without being escaped for shell use. All cask-controlled values interpolated into generated scripts are now escaped, so shell metacharacters in the metadata are treated as literal text.
- Fixed macOS Fleet-maintained apps with in-bundle login items or background helpers (e.g. 1Password's browser helper) always being detected as open. The Fleet-managed "app is open" check now matches only the app's own executable instead of any process running inside the app bundle.
- Fixed Fleet-maintained apps selecting the wrong version to be active.
- Fixed the Fleet-maintained app auto-update cron not updating install and uninstall scripts when it advances an app to a new version.
- Fixed the Fleet-maintained apps auto-update cron job not updating scripts when they change in the manifest without a version change.
- Fixed software search to match on `bundle_identifier` and custom `display_name` so admins and end users can find macOS custom packages by their visible name.
- Fixed the transient empty state shown when changing Self-service search and category filters.
- Fixed self-service reinstall and uninstall buttons remaining disabled after cancelling the uninstall confirmation modal.
- Fixed the VPP install details modal retrying the command results request four times when the result isn't available yet and the API returns a 404.
- Fixed activity feed rendering a blank actor for failed iPad VPP installs: auto-update terminal failures now attribute to Fleet, and any install whose initiating admin has since been deleted also renders "Fleet" instead of an empty name.
- Fixed uninstalling `.sh` and `.py` script packages from macOS hosts, which was rejected even though installing them there is allowed. The rejection message for other platforms now names them the same way the install message does ("macOS and Linux" rather than "linux").
- Fixed the "Advanced options" reveal button on the Edit software modal not expanding for `.msix` packages (e.g., Claude, Slack on Windows), which prevented users from viewing or editing install/uninstall scripts.
- Fixed the error toast shown when selecting a custom package with an unsupported extension so the friendly message stays on the main line and the extension reason appears in the expandable raw-response panel.
- Fixed the **Save** button never activating when adding a package from the software title page while GitOps mode is enabled.
- Fixed the "Software" automation filter on the Policies page to no longer include patch policies. Added a dedicated "Patch" filter option.
- Fixed the total number of retries of a failing software install automation across policy runs not being limited.
- Fixed exclude-label-scoped policies running, and their automations firing, on newly enrolled hosts before the host had evaluated the label's membership.
- Fixed an issue where label-scoped reports could run on hosts outside the target label (or be skipped for hosts inside it).
- Fixed a race where a newly created or edited label-scoped query could briefly be delivered to every host (and out-of-scope results stored in its report) because the query row was committed before its labels.
- Fixed built-in labels being overwritten, renamed, or deleted by supplying a label name that differs from the built-in name only in letter casing.
- Fixed the modify label endpoint so that a dynamic label's membership can no longer be cleared by sending an empty `hosts` or `host_ids` list, which is now rejected like a non-empty one.
- Fixed the label spec endpoints so that the host membership list only includes hosts the requesting user is authorized to see, preventing cross-team host ID disclosure through global manual labels.
- Fixed an issue where disabled packs could still be applied to hosts in certain targeting configurations.
- Fixed query (report) results submitted to `/api/osquery/log` so that they are only accepted when the query is actually scheduled for the submitting host, preventing an enrolled host from adding rows to reports it was never assigned or from streaming results for those queries to a log destination.
- Fixed report descriptions so that newlines are preserved when the description is displayed, instead of being collapsed onto a single line.
- Fixed the fleet and global schedule endpoints accepting reports that belong to a different fleet, and made them return the same "not found" response for a report outside the caller's access as for one that doesn't exist.
- Fixed live query authorization so that an empty team selection is authorized like an omitted one.
- Fixed policy result ingestion so a host can no longer report results for policies it is not assigned, preventing forged policy membership across fleet, platform, and label scopes.
- Fixed an issue where an activity might not be created when an MDM command was enqueued via the API.
- Fixed `GET /api/v1/fleet/hosts/identifier/:identifier` disclosing host details to GitOps users, who are denied on all other host read endpoints. Unlike `GET /api/v1/fleet/hosts/:id`, which returns an error for GitOps users, this endpoint still succeeds and returns the host's `id` (and nothing else), for backwards compatibility with the deprecated Puppet module.
- Fixed authentication and enrollment tokens (including session, invite, email-change, MFA, host device, and MDM enrollment/installer tokens) so that case-mutated tokens are no longer accepted; these tokens are now matched case-sensitively.
- Fixed deleting a certificate template that doesn't exist returning a 500 internal server error. Users authorized to manage certificate templates now get a 404, and users who aren't get a 403 whether or not the template exists.
- Fixed getting a certificate template by ID disclosing whether templates a user can't access exist. Reading a template on another fleet now returns a 404, the same as a template that doesn't exist.
- Fixed the add/edit certificate authority modals showing a generic "Please try again." error instead of the invalid URL error returned by the server. The error now names the certificate authority, for example "Invalid Hydrant URL. Please correct and try again."
- Fixed editing only the username or only the password of an NDES SCEP certificate authority skipping validation against the NDES server. Fleet now verifies the credentials on save and returns an error if they're wrong, instead of saving a broken certificate authority whose misconfiguration only surfaced later as a profile failure on hosts.
- Fixed editing only the SCEP URL of an NDES SCEP certificate authority failing with a `"password" must be set when modifying an existing certificate authority` error. The password field is now cleared when the SCEP URL changes, so it's re-entered and sent with the update.
- Fixed an empty username or password being saved on an NDES SCEP certificate authority.
- Fixed adding or editing a certificate authority with a bad NDES admin URL or credentials showing a generic "Please try again." message instead of "Invalid admin URL or credentials."
- Fixed updating an NDES SCEP certificate authority with the masked password (`********`) returned by the GET endpoint sending the mask to the NDES server as the literal password and failing with a misleading "invalid credentials" error. The mask is now rejected with an invalid-password error, matching GitOps behavior.
- Fleet now skips validating NDES credentials against the NDES server when an update leaves the admin URL, username, and password unchanged, so a no-op edit doesn't consume a slot in NDES's password cache.
- Fixed an unreachable NDES admin URL (timeout, DNS failure, connection refused) being reported as "Invalid admin URL or credentials" when editing an NDES SCEP certificate authority. It's now reported as "Couldn't connect to admin URL."
- Fixed the Save button staying enabled in the edit certificate authority modal after Fleet clears the unchanged NDES password, which let the form submit an empty password.
- Fixed editing a Windows configuration profile so that uploading a replacement file with a different name updates the profile's name.
- Fixed uploading a Windows configuration profile with a file name longer than 255 characters returning a database error.
- Fixed a bug where Windows disk encryption showed a "Resend" action that always failed, since BitLocker enforcement isn't a configuration profile.
- Fixed an issue where Observer, Observer+ and Technician could not see the managed account rotation banner.
- Fixed the error returned when a free-tier request sets a premium-only field so that it names the field as it appears in the request payload (for example `critical`) instead of Fleet's internal Go field name (for example `Critical`).
- Fixed a query returning a MySQL error if hit with unsupported platforms, by now returning an empty result.
- Fixed an issue where stale fleet names could appear in `mdm.apple_business` after renaming a fleet.
- Fixed false positive vulnerabilities reported for JetBrains `teamcity-cli` installed via Homebrew, which was incorrectly matched to the TeamCity CI server's CPE.
- Fixed `fleetctl gitops` silently dropping `ios_updates` and `ipados_updates` when it creates a new fleet.
- Fixed `fleetctl generate-gitops` failing with an unsupported Content-Type error when the org logo is an SVG.
- Fixed `fleetctl generate-gitops` not using `.sh` and `.ps1` extensions for install scripts.
- Fixed an issue where deleting an ADE device after turning off Apple Business could leave orphaned rows.
- Fixed form validation on the new/edit user and API-only user forms: field errors no longer appear before a field has been edited, clear as soon as the field is focused, and the "select at least one fleet" error now renders on the selector instead of as a toast.
- Updated validation error copy on the new/edit user and API-only user forms to the standard wording (e.g. "Enter an email" instead of "Email field must be completed").
- Fixed the fleet, role, and API access controls staying editable while a user was being saved on the new/edit user and API-only user forms.
- Fixed error toasts showing an expandable "Raw response" panel containing an empty `{}` when the underlying error carried no details.
- Fixed inconsistent spacing around the enrollment URL on the iOS & iPadOS tab of the **Add hosts** modal, so it now matches the macOS and Android tabs.
- Fixed the confirmation checkbox on the "Clear passcode" host modal rendering in green instead of red, so it now matches the destructive "Clear passcode" button.
- Fixed incorrect background color on authentication pages (login, SSO, registration) in dark mode.
- Fixed autofilled inputs showing a white background in dark mode. The autofill style now uses theme colors instead of a hardcoded white, and covers Firefox via the standard `:autofill` selector.
- Fixed label color for install/post-install/uninstall script fields in software package advanced options to match Fleet's standard form label color.
- Fixed long unbreakable words (e.g. file paths in inline code) overflowing the table info side panel on the report and policy editor pages.
- Matched the height of the host status and platform/label filter dropdowns on the Hosts page.
- Fixed the "Collecting results..." empty state on the report details page reading "about about X hours".

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.92.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-09-21">
<meta name="articleTitle" value="Fleet 4.92.0 | Android commands, Windows Autopilot hosts, custom FileVault, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.92.0-1600x900@2x.png">
