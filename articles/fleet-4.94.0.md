# Fleet 4.94.0 | Configuration profile editor, self-service profiles, multiple App Store app versions, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/N3KoFc4D_uM?si=svfjCTkZC2BezvVF" allowfullscreen></iframe>
</div>
-->

Fleet 4.94.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.94.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Configuration profile editor with names and descriptions](#configuration-profile-editor-with-names-and-descriptions)
- [Self-service configuration profiles for macOS](#self-service-configuration-profiles-for-macos)
- ~~[Multiple versions of the same App Store app](#multiple-versions-of-the-same-app-store-app)~~ ([coming in Fleet 4.95](https://github.com/fleetdm/fleet/issues/47904))
- [Custom end user agreement for Windows enrollment](#custom-end-user-agreement-for-windows-enrollment)
- [Filter software by type](#filter-software-by-type)
- [Android: full app configuration and BYOD work profile wipe](#android-full-app-configuration-and-byod-work-profile-wipe)
- [Android zero-touch enrollment to any fleet](#android-zero-touch-enrollment-to-any-fleet)
- [Rotate a Mac's FileVault recovery key on demand](#rotate-a-macs-filevault-recovery-key-on-demand)

### Configuration profile editor with names and descriptions

IT admins can now add and edit configuration profiles on a dedicated page instead of a modal. Upload a file, paste the profile, or edit it inline. Give each profile a name and a description, so the team knows what it does without opening it. Names and descriptions can also be set through the [API](https://fleetdm.com/docs/rest-api/rest-api#create-configuration-profile) and [GitOps](https://fleetdm.com/docs/configuration/yaml-files#apple-settings-and-windows-settings).

Uploading a replacement file no longer renames the profile, and renaming a profile no longer re-sends it to hosts. On the [batch endpoint](https://fleetdm.com/docs/rest-api/rest-api#batch-update-configuration-profiles), `display_name` is deprecated in favor of `name`.

Learn more in the [configuration profiles guide](https://fleetdm.com/guides/custom-os-settings).

GitHub issue: [#51814](https://github.com/fleetdm/fleet/issues/51814)

### Self-service configuration profiles for macOS

IT admins can now offer configuration profiles that end users install themselves from the **My device** page, instead of pushing every profile to every host. Self-service profiles for iOS and iPadOS are [coming soon](https://github.com/fleetdm/fleet/issues/52841). Profiles can also be marked hidden, so they only appear after the end user turns on **Show hidden profiles**.

Learn more in the [configuration profiles guide](https://fleetdm.com/guides/custom-os-settings#self-service), or set `self_service` and `hidden` in [GitOps](https://fleetdm.com/docs/configuration/yaml-files#apple-settings-and-windows-settings).

GitHub issue: [#46834](https://github.com/fleetdm/fleet/issues/46834)

### Multiple versions of the same App Store app

This feature is [coming in Fleet 4.95](https://github.com/fleetdm/fleet/issues/47904).

### Custom end user agreement for Windows enrollment

IT admins can now replace the default terms page that end users see during [automatic enrollment](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment), with or without [Windows Autopilot](https://fleetdm.com/guides/windows-mdm-setup#windows-autopilot), with their own end user agreement. Upload a Markdown file in **Settings > Integrations > MDM > End user license agreement**, or set [`windows_eula`](https://fleetdm.com/docs/configuration/yaml-files#windows-eula) in GitOps. Adding or deleting an agreement creates an activity. You can also manage it with the [EULA API endpoints](https://fleetdm.com/docs/rest-api/rest-api#create-eula).

Learn more in the [setup experience guide](https://fleetdm.com/guides/setup-experience#end-user-agreement-eula).

GitHub issue: [#50146](https://github.com/fleetdm/fleet/issues/50146)

### Filter software by type

The **Software** page, **Host details**, and **My device** software tables now have a **Types** filter. Use it to narrow inventory to applications, browser extensions, packages, or other kinds of software. On macOS, a **Show helpers** toggle replaces the Applications / Full inventory dropdown. The same filters are available as `source` and `extension_for` parameters on the software API endpoints, like [list software](https://fleetdm.com/docs/rest-api/rest-api#list-software).

See every software type Fleet collects in the [software inventory reference](https://fleetdm.com/guides/software-inventory-reference).

GitHub issue: [#51823](https://github.com/fleetdm/fleet/issues/51823)

### Android: full app configuration and BYOD work profile wipe

[Android app configuration](https://fleetdm.com/guides/install-app-store-apps#managed-configuration) now accepts every Android Management API [`ApplicationPolicy`](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#ApplicationPolicy) field, such as `autoUpdateMode`, `permissionGrants`, and `minimumVersionCode`. It also accepts `credentialProviderPolicy`, so end users on Android 14 and later can pick an approved password manager as their autofill and passkey provider in the work profile.

Personally-owned (BYOD) Android hosts can now be wiped from the [API](https://fleetdm.com/docs/rest-api/rest-api#wipe-host) and fleetctl. Only the work profile is removed, so the end user's personal data stays untouched.

GitHub issues: [#51454](https://github.com/fleetdm/fleet/issues/51454), [#53471](https://github.com/fleetdm/fleet/issues/53471)

### Android zero-touch enrollment to any fleet

IT admins can now choose which fleet company-owned Android hosts land in when they enroll with zero-touch enrollment. Create a zero-touch configuration for each fleet, then apply the right configuration to each device in the zero-touch portal.

Learn how in the [Android zero-touch enrollment guide](https://fleetdm.com/guides/android-zero-touch-enrollment#enroll-hosts-to-multiple-fleets).

GitHub issue: [#51479](https://github.com/fleetdm/fleet/issues/51479)

### Rotate a Mac's FileVault recovery key on demand

IT admins can now rotate a macOS host's FileVault recovery key whenever they need to, for example after the key has been viewed to unlock a device. Rotate it from the **Disk encryption key** modal on **Host details**, or with the [rotate host's disk encryption key API](https://fleetdm.com/docs/rest-api/rest-api#rotate-hosts-disk-encryption-key).

Learn more in the [disk encryption guide](https://fleetdm.com/guides/enforce-disk-encryption#rotate-disk-encryption-key).

GitHub issue: [#50149](https://github.com/fleetdm/fleet/issues/50149)

## Changes


### IT Admins
- Added a page for adding and editing configuration profiles, replacing the add and edit modals. A profile can be uploaded, pasted, or edited inline.
- Added `name` and `description` fields to configuration profiles, set in the UI, the API, or GitOps YAML.
- Changed profile renames so that uploading a replacement file no longer renames the profile, and renaming a profile no longer re-sends it to hosts. The exception is renaming a Windows, Android, or declaration (DDM) profile through GitOps, which re-creates the profile.
- Deprecated `display_name` in favor of `name` on the batch configuration profiles endpoint.
- Added self-service configuration profiles for macOS hosts, which end users install from the My device page.
- Added a `hidden` option for configuration profiles. Hidden profiles appear on My device > Controls only after the end user turns on the "Show hidden profiles" toggle, which is shown only when the host has hidden profiles.
- Added an "Actions" column header on My device > Controls.
- Added a customizable end user agreement (EULA) for Windows. Admins can upload a markdown file of up to 512 KB in Settings > Integrations > MDM > End user license agreement, or set `windows_eula` in GitOps, and Fleet renders it as the terms page shown during Microsoft Entra and Autopilot enrollment. Tables with merged cells, underline, subscript and superscript can be written in HTML, as pandoc exports them from Word documents. When no file is uploaded Fleet keeps showing its default terms. Adding or deleting an agreement on either platform now creates an activity.
- Added a "Types" filter to the Software page, Host details, and My device software tables (`source` and `extension_for` query parameters on the list software titles, list software versions, get host's software, and get device's software endpoints), renamed software type labels, and replaced the macOS Applications / Full inventory dropdown with a "Show helpers" toggle.
- Changed Android app configuration to accept every Android Management API `ApplicationPolicy` field (e.g. `autoUpdateMode`, `permissionGrants`, `minimumVersionCode`) except `packageName` and `installType`, which Fleet sets itself.
- Added support for the `credentialProviderPolicy` option in Android app configuration, allowing end users on Android 14+ to select an allowed app (e.g. a third-party password manager) as their autofill and passkey (credential) provider in the work profile.
- Added the ability to choose a destination fleet for Android zero-touch enrollment. `GET /api/v1/fleet/android_enterprise/zero_touch_configuration` now accepts an optional `fleet_id` query parameter, and the Android zero-touch page has a fleet picker.
- Added wipe for personally-owned (BYOD) Android hosts from the API and fleetctl, on Fleet Free and Premium. On BYOD hosts, only the work profile is removed.
- Allowed users with the technician role to delete hosts, globally or within their fleet (Fleet Premium).
- Added a `hidden` option to fleet-level policies so admins can hide policies from end users in Fleet Desktop, and exposed a failing policies count that excludes hidden policies on the device endpoints.
- Added drag-and-drop support to every file upload surface in Fleet, including custom packages, configuration profiles, scripts, APNs certificate, ABM and VPP tokens, EULA, and bootstrap packages.
- Changed software install, uninstall, and cancellation activities on the dashboard and Host details page to show a title's per-fleet "Software name" override (when one is set) instead of the raw installer or App Store name. Activities recorded before this change, and titles without an override, keep the raw name.
- Changed host name templates to be re-applied automatically (up to 3 times) when a macOS, iOS, or iPadOS host is renamed on the device or rejects the rename, instead of marking the host as failed right away. Hosts currently marked as failed because they were renamed on the device are re-enforced after upgrading Fleet.
- Added the "On (personal)" MDM status for hosts enrolled with a Managed Apple Account or an Android work profile, separate from "On (manual - personal)" for hosts enrolled with a personal enrollment profile, with a matching `mdm_enrollment_status=manual-personal` filter and dashboard count.
- Changed Windows SCEP certificate profiles (custom SCEP, NDES, and Smallstep) to show as verified or failed within a couple of minutes of the host installing them, instead of after the next certificate inventory (up to an hour). This also releases Windows hosts from the setup experience as soon as their certificate is installed.
- Added an hourly cleanup that deletes Windows MDM command history (raw device responses, command results, and the commands themselves) once it has gone 30 days without being recorded or updated, so those tables no longer grow without bound. Commands still waiting to run are kept regardless of age, and so are a wipe command, its result, and the device response that carried it, for as long as a host's wiped status depends on that command. The retention window is configurable with `mdm.windows_command_retention` (`FLEET_MDM_WINDOWS_COMMAND_RETENTION`); setting it to 0 disables the cleanup.
- Added an hourly cleanup of completed Apple MDM commands to keep the associated tables from growing without bound.
- Added an hourly cleanup that permanently deletes Fleet's records of certificates removed from hosts more than 30 days ago, configurable with `server.deleted_host_certificates_retention`.
- Added `path`/`paths` file-reference support to `software.fleet_maintained_apps` and `software.app_store_apps` in GitOps YAML, matching the existing support for `policies`, `labels`, `reports`, and `software.packages`.
- Added direct upload of software packages and bootstrap packages to a GCS installer store: `POST /api/v1/fleet/staged_upload` returns a presigned upload URL, and add package, edit package, and add bootstrap package accept the resulting `upload_id` in place of the file. `fleetctl` uploads bootstrap packages this way when the server reports `staged_upload_available`.
- Renamed the `s3_software_installers_signed_url` server option to `s3_software_installers_gcs_signed_url`. The old name still works and is deprecated.
- Added support for the NDES certificate authority type in the "Request certificate" API.
- Added support for `$FLEET_VAR_NDES_SCEP_CHALLENGE` in scripts, so a script can request a certificate from an Okta or Microsoft NDES certificate authority through the request certificate API.
- Made the Version column sortable on the dashboard "Operating systems" card and the Software > OS page once a specific platform is selected, instead of sorting only by host count. Versions sort numerically by segment (so, for example, "26.10" sorts after "26.6"), Windows feature-update codenames (e.g. "22H1") sort by year and half, and Ubuntu LTS releases (e.g. "22.04.9 LTS") sort numerically too. Sorting by version is disabled while viewing all platforms together (which still sorts by host count), and results are grouped by platform (most hosts first) when a platform filter still spans multiple platforms (e.g. the dashboard's Linux tab). Added `version` as a valid `order_key` for `GET /api/v1/fleet/os_versions`.
- Added the `websocket.orbit_config_enabled` server setting (experimental, requires `websocket.transport_enabled`): Fleet notifies agents connected over the WebSocket transport when their orbit config changes, so they poll it every `websocket.orbit_config_poll_interval` (default 5 minutes) instead of every 30 seconds. Notifications of changes affecting a whole fleet or every host are spread over `websocket.orbit_config_spread` (default 30 seconds).
- Added a `BYPASS_END_USER_AUTH` property to the Windows MSI built by `fleetctl package`, so end user authentication can be skipped at install time (for example `msiexec /i fleetd-base.msi BYPASS_END_USER_AUTH=true`) instead of baking in the `--bypass-end-user-auth` flag. Requires fleetd v1.60.0 or higher.
- Added script package filenames to Software inventory and Setup experience so packages with the same title can be distinguished.
- Added software inventory for NixOS hosts, reported by fleetd as Nix packages.
- Added Outset as a macOS Fleet-maintained app.
- Added `CLAUDE.md` and a `fleet-gitops` Claude Code skill to the `fleetctl new` GitOps repo template, so AI coding agents working in a scaffolded repo have guidance on its layout and how to validate changes.

### Security Engineers
- Added `mdm.windows_one_time_enroll_secrets`. When enabled, a Windows MDM host that Fleet installs fleetd on is sent a per-device, single-use enroll secret instead of the shared global enroll secret. An administrator resending the Fleet-managed "Fleetd enroll secret" configuration profile is how a host whose secret was already used gets another. A shared enroll secret can no longer enroll fleetd as a host that is enrolled in Fleet Windows MDM. A Windows MDM host that is deleted in Fleet is sent a new secret at its next MDM check-in, so fleetd can re-enroll it.
- Added dynamic SCEP challenges for Apple MDM enrollments, with an option to block the static challenge.
- Required hosts in fleets that require IdP authentication to complete it to enroll through automatic enrollment (ADE). Fleet refuses devices that got their automatic enrollment configuration before their fleet required IdP authentication, and records a "Host enrollment failed" activity so admins can reactivate them.
- Added an API endpoint for global admins to rotate the token Fleet puts in automatic enrollment (ADE) profiles for fleets without IdP authentication. The previous token keeps working for a grace period (24 hours by default).
- Added on-demand FileVault recovery key rotation for macOS hosts via `POST /api/v1/fleet/hosts/:id/encryption_key/rotate` and the Disk encryption key modal.
- Added the `bound_host_to_idp_account`, `unbound_host_from_idp_account`, and `refused_host_idp_account_change` activities, recorded whenever enrollment or MDM SSO links, replaces, removes, or refuses a change to a host's IdP account.

### Bug fixes and improvements
- Changed Windows MDM enrollment to no longer replace an enrollment that belongs to a different host (unless that host was wiped or deleted), and an enrollment that fails no longer removes the existing one. fleetd enrollments are checked against the host's orbit node key. Microsoft Entra enrollments are checked against the Entra device ID in the access token (the `deviceid` claim), or for a host enrolled through fleetd, against the Entra device ID of its Autopilot registration. v2.0 access tokens include `deviceid` only when the MDM app registration lists it as an optional access token claim; without it Fleet logs an error and the enrollment is not protected. To free a hardware ID held by a stale host, delete that host.
- Improved Windows MDM host linking so the Autopilot device ID (ZTDID) a device reports cannot associate its enrollment with a host that a different device's enrollment already manages.
- Upgraded the gorilla/mux OpenTelemetry instrumentation to v0.71.0. HTTP server metrics for gorilla-routed requests now use the stable semantic conventions: `http.server.request.duration` in seconds replaces `http.server.duration` in milliseconds, and `http.server.request.body.size` and `http.server.response.body.size` replace `http.server.request.size` and `http.server.response.size`. No Fleet dashboard or alert queries the old names, so no action is expected; any custom dashboard built on them would need updating.
- Changed Windows MDM to record a device's SyncML command responses only after the device has authenticated.
- Changed osquery API endpoints to validate that the authenticating host runs on a platform that supports osquery.
- Changed the "Refetch host" API endpoints to return a `400` for Android hosts, which sync their data automatically and never refetch on demand, instead of a success response. Any pending refetch left on an Android host is cleared.
- Changed `GET /api/v1/fleet/hosts/:id/device_url` to return a 400 error instead of a 500 for iOS/iPadOS hosts that haven't enrolled in MDM yet.
- Hid the "My device" link on Host details for iOS/iPadOS hosts that aren't enrolled in MDM (Pending or Off) or when Fleet Premium isn't active.
- Changed the GitHub Actions workflow generated by `fleetctl new` so pull request dry runs use `pull_request_target`, which makes the workflow, action, and gitops script always run from the default branch. The pull request's configuration is checked out separately and used as data only, so a pull request can no longer modify them to read repository secrets.
- Removed the `!` logic and the pipe to `grep -q` from the `gitops.sh` that `fleetctl new` generates, which failed ShellCheck SC2251 and SC2337.
- Improved the performance of filtering the hosts list by OS settings status on Windows. The filter now reads the maintained per-host profile status rollup instead of re-aggregating every host's configuration profile rows on each request, which made the page take over a minute for fleets with many profiles applied to many hosts.
- Improved the Apple MDM profile reconciler to drain up to 2k hosts per tick, to further reduce database write load.
- Reduced database load from Windows MDM check-ins of Entra and Autopilot enrolled hosts by remembering when fleetd is present instead of looking up the host on every check-in.
- Reduced Windows MDM load during enrollment. Fleet now sends the Autopilot Enrollment Status Page hold commands and the device serial number request once per management session instead of on every message. Sessions end normally while the device waits for fleetd to link it.
- Reduced database load of the `GET /api/v1/fleet/hosts/:id/health` endpoint by only loading the host's vulnerable software instead of its full software inventory with CVE details.
- Improved GitOps so fetching many Fleet-maintained app manifests can no longer time out the request.
- Updated Go to 1.27.3.
- Improved request validation and temporary file cleanup for software and bootstrap package uploads.
- Improved validation of host-reported platform values used when processing OVAL and goval-dictionary vulnerability feeds.
- Moved certificate authority and certificate template secret encryption to Fleet's shared encryption package. Existing encrypted values continue to work without changes.
- Settings that don't currently apply are now shown disabled with a tooltip explaining why, instead of being hidden: the host status webhook fields, the "Automatically migrate hosts" checkbox on the Windows MDM page, and the host expiry window (which is also now indented under its "Host expiry" checkbox).
- Added an empty state to the Controls > OS updates tabs when MDM isn't turned on for a platform.
- Added a Linux tab to Controls > OS updates that links to the guide on enforcing Linux OS updates with a policy and script.
- Renamed the "Play Protect enabled" and "USB debugging enabled" Android host vitals to "Play Protect" and "USB debugging," and changed their values from "True"/"False" to "Enabled"/"Disabled."
- Reworked the vulnerable software severity filter: removed "Custom severity" from the dropdown menu, always show the min/max CVSS score inputs, and display "Custom severity" as the current selection only after a saved score range that doesn't match a built-in severity band is reopened.
- Added an explanation when Live report is disabled because a host is locked, wiped, or its device status is updating.
- Added a message when iOS/iPadOS VPP installs fail with error code 1407 to guide users on how to update apps in Single App Mode.
- Changed the Android enrollment link to show the Android BYOD enrollment instructions (QR code) instead of an irrelevant "Apple MDM is turned off" error when it is opened on a Mac or iOS device with Apple MDM off and Android MDM on.
- Added the missing "Learn more" link to the error shown when uploading an unsigned bootstrap package.
- Updated policy error messages to say "fleet-level policies" instead of "team policies" when `run_script`, `install_software`, or `resend_configuration_profile` is set on a global policy.
- Updated form validation error copy across the migrated forms to the standard wording (e.g. "Enter your email" instead of "Email field must be completed", "Match the password above" instead of "Passwords do not match").
- Moved server-side error messaging on the login, forgot-password, reset-password, and confirm-invite pages from a red banner inside the form to the app's standard toast notification, matching the rest of Fleet.
- Replaced hand-rolled file inputs on the Add asset and Add profile modals with the shared file uploader for consistent drop zone, error, and loading states.
- Wrapped the "Advanced options" sections on the Setup experience > Bootstrap package and Setup assistant pages in a card with the Save button on the right, matching the pattern used elsewhere in the Controls section.
- Made the "View all hosts" and "Check in now" buttons in the Host details MDM status modal always visible (previously shown only on row hover) with Fleet's updated secondary button style, and added a confirmation toast when "Check in now" is clicked.
- Replaced the "Enable disk encryption" checkbox's helper text with a tooltip on the macOS and Windows tabs of Controls > OS settings > Disk encryption.
- Moved the search box to the right-most position in the table controls on the host details Software > Inventory and Library tabs.
- Moved the copy button on the Variables page next to the other row actions, made it always visible, and added a "Copy the variable" tooltip. Renamed the "Variable name" column to "Variable".
- Updated the empty-state text in the host details page Activity card to reference commands instead of scripts and software, which don't apply to every platform.
- Updated the Apple Business table to hide the iPadOS column first on smaller screens.
- Left-aligned the text in dropdown option tooltips (e.g. disabled action explanations) instead of centering it.
- Fixed fleetd and osquery enrollment so a host that holds a host identity certificate can only be re-enrolled with a request signed by that certificate. Rejected enrollments are recorded in the "Fleet rejected an enrollment" activity on the host.
- Fixed Orbit enrollment so that, when `mdm.allow_orbit_end_user_auth_bypass` is set to `false`, a host cannot skip required end user authentication by reporting a macOS platform. Set that option to `false` to enforce end user authentication on every platform.
- Fixed Android devices re-enrolling with a deleted or rotated enroll secret not being rejected.
- Fixed the setup experience MDM SSO flow letting any caller bind any host UUID to their own IdP account. Fleet now only starts the flow for a host it has just asked to authenticate, and a sign-in that completes after the host has enrolled can no longer replace its IdP account.
- Fixed a privilege escalation where a fleet admin could gain global admin access by creating a user with an empty global role.
- Fixed SSO logins accepting a SAML response more than once: verified assertions are now single-use until they expire, so a captured response cannot be replayed to log in again.
- Fixed SAML response validation so that enabling IdP-initiated SSO login no longer skips the `InResponseTo` request ID check for SP-initiated logins.
- Fixed live query result ingestion so a host can no longer submit results for campaigns that do not target it, preventing forged rows from being streamed into other running live queries.
- Fixed query parameter encoding in requests to the Microsoft compliance partner proxy, and required the Microsoft Entra tenant ID to be a valid GUID when setting up the Entra conditional access integration.
- Fixed automatic install and patch policy queries generated from software package metadata so that single quotes in the metadata are escaped.
- Fixed an issue where lower max request size endpoints got overwritten to the global default.
- Fixed Fleet slowing down or becoming unresponsive when many live reports run at the same time.
- Fixed the host list and host count timing out when filtered by a vulnerability (CVE) that affects many software versions.
- Fixed slow responses from `GET /api/v1/fleet/vulnerabilities` when sorting by "created_at" (the "Detected" column) on large deployments.
- Fixed missing operating system vulnerabilities on Windows hosts: Windows Server 2012, 2012 R2, 2016 and 2019 reported none at all, and other Windows hosts could intermittently lose theirs between vulnerability scans.
- Fixed software vulnerability matching treating versions with fewer trailing zeros as older (for example, "157.0" as lower than "157.0.0"), which reported hosts as vulnerable to CVEs already fixed in their version and missed CVEs at the start of an affected range.
- Fixed false positive vulnerabilities where Adobe Lightroom and Adobe Lightroom Classic on macOS were matched against each other's CVEs.
- Fixed false positive vulnerabilities (e.g. CVE-2014-6271) reported for the `bash` Python package, which was incorrectly matched to the GNU bash CPE.
- Fixed the vulnerability details page and `GET /api/v1/fleet/vulnerabilities/:cve` returning "not a known CVE" (404) for CVEs that are detected on hosts but aren't in NVD yet (e.g., published by Microsoft before NVD). These CVEs now load with empty severity, EPSS, and published fields.
- Fixed Windows hosts reporting a stale OS settings status for up to an hour after a configuration profile was resent because a custom host vital or an IdP-sourced variable changed.
- Fixed software installs being stranded in the queue when the fleet holding their installer was deleted. Deleting a fleet now cancels its queued and in-flight installs, the same as deleting a package.
- Fixed software installs left in a pending state by an installer that was deleted in an earlier version. Each one now reports that the installer no longer exists, and the host's queue moves on.
- Fixed the script results and software installs cleanups never deleting pending runs of deleted hosts. They're now deleted once older than the retention period.
- Fixed a host's manual label membership being removed by label results reported by the host itself.
- Fixed the Hosts page showing only "No fleet" hosts under "All fleets" for OS settings, disk encryption, and bootstrap package status filters by clearing those filters when switching to "All fleets" from the fleet dropdown or the command palette. The software status filter is now also cleared from the URL on that switch.
- Fixed the Hosts page ignoring the Apple Business (AB) issues filter when a label was selected, and showing the wrong host count with that filter.
- Fixed `POST /api/v1/fleet/global/policies/delete` returning a 500 "forbidden" error for a nonexistent policy ID (now 404) or an empty `ids` list (now a no-op).
- Fixed `fleetctl gitops --dry-run` returning success before checking Apple's asset list, so an `app_store_id` under `software.app_store_apps` with no license on the fleet's VPP location passed CI and failed the apply part-way through, which could leave fleets without their App Store apps. The dry run now reports `requested app not available on vpp account`, the same error the real apply returns, and makes the same read-only asset request to Apple's VPP service, so GitOps CI needs the same outbound access to Apple as the server already needs for applies. For a fleet that does not exist yet, the dry run also applies the same payload checks as the apply (platform, version names, iOS auto-update window) and checks the app against every unexpired VPP token.
- Fixed GitOps and UI VPP token fleet assignment changes deleting VPP apps and policy install automations on fleets that should stay unchanged.
- Fixed `fleetctl generate-gitops` not adding any fleets under `volume_purchasing_program` when a VPP token is assigned to all fleets.
- Fixed GitOps runs failing with a duplicate name error when .mobileconfig profiles trade names, or one takes a name another profile is giving up, in the same run.
- Fixed GitOps software batch uploads continuing to download remaining packages after one package failed.
- Fixed `fleetctl gitops --dry-run` passing a label whose name or description is longer than 255 characters, which then failed the real apply with a generic error. Fleet now rejects these labels with a clear message in the dry run, on `fleetctl apply`, and in the labels API.
- Fixed `fleetctl apply` silently dropping incorrectly typed paths and labels in fleet custom profile settings. Invalid values now return a validation error.
- Fixed a `fleetctl gitops` crash when `controls.enable_disk_encryption`, `controls.enable_recovery_lock_password`, or `controls.windows_require_bitlocker_pin` is set to a non-boolean value; it now returns a validation error.
- Fixed `fleetctl generate-gitops` so `windows_enabled_and_configured` and `android_enabled_and_configured` are only exported to the global controls file (including on Fleet Free), and no longer leak into fleet YAML files.
- Fixed the duplicate fleet name check in the GitOps `gitops.sh` that `fleetctl new` generates: it never stopped the run, because `set -e` ignores a command starting with `!`.
- Fixed the GitHub Actions workflow generated by `fleetctl new` failing to parse, which caused GitHub to reject it as an invalid workflow file.
- Fixed `fleetctl package` on Windows failing to build a package for another architecture (e.g. `--arch=arm64` on an x86 machine).
- Fixed the "Manage enroll secrets" button on a fleet's detail page staying disabled in GitOps mode when the secrets exception was enabled.
- Fixed Windows 10 and Windows 11 CIS policies that could never pass (firewall logging, printer drivers, NTP server, remote SAM calls), service policies that always passed regardless of service state, and numeric threshold policies that compared registry values as text.
- Fixed Windows and Linux setup experience skipping the install of a Fleet-maintained app whose patch policy has an install automation (Patch when app is closed or Force patch). A patch policy passes on hosts that don't have the app, so it's no longer used to decide whether to skip the install.
- Fixed a deleted Windows host enrolled in MDM through Microsoft Entra ID coming back without its IdP user or SCIM user mapping.
- Fixed end users getting an "MDM features aren't turned on" error when resending a Windows configuration profile from My device on Fleet instances without Apple MDM.
- Fixed macOS setup experience queuing a new release command on every status check while the previous one was unanswered, which could flood a host's MDM command queue.
- Fixed Apple MDM identity certificate renewals getting stuck when push notifications fail for some or all hosts.
- Fixed Apple MDM commands getting stuck for a host when Fleet couldn't fill in a host-specific secret (such as a Recovery Lock password) for a queued command.
- Fixed macOS, iOS, and iPadOS hosts that were erased and re-enrolled showing a failed host name setting instead of being renamed to match the host name template.
- Fixed macOS hosts briefly showing disk encryption as "Action required" with a banner after being transferred to a fleet with disk encryption off. They now show "Removing enforcement" until the FileVault profile is removed.
- Fixed Fleet treating Arch Linux, Arch Linux ARM, Manjaro, Manjaro ARM, CachyOS, and Omarchy hosts as eligible for disk encryption key escrow in the host API and escrow endpoint while the disk encryption summary, host filters, and UI did not. Escrow eligibility is now defined once (Ubuntu, Kubuntu, Zorin OS, and Fedora) and shared by all of them.
- Fixed the re-enrollment instructions and software description shown for iPhones and iPads enrolled with a personal enrollment profile.
- Fixed Platform SSO registration showing "Incorrect username or password" when Fleet's identity provider client settings are wrong. Only a rejected password is now reported as bad credentials; other identity provider errors return a 400 error and are logged.
- Fixed SCIM group `PATCH`, `POST`, and `PUT` requests failing when a member ID doesn't exist in Fleet. `PATCH` rejected the whole request with a `400`, and `POST` and `PUT` returned a `500`. Unknown members are now skipped and logged, and the remaining members are applied. A request where none of the added members exist still returns a `400` that lists only the unknown members, unless it is a `PATCH` that also removes members.
- Fixed a bug that could lead to an improperly formatted AccountConfiguration command during DEP enrollment.
- Fixed the ACME enrollment CA occasionally returning a bad nonce error instead of not found on certain client errors.
- Fixed a typo in the error message returned when Apple Push Notification service delivery fails for all hosts targeted by an MDM command.
- Fixed Android hosts that were factory reset or re-enrolled showing their configuration profiles as verified while none of the profile settings were applied. Fleet now re-sends the host's profiles after assigning the host its own policy, and only verifies profiles against the host's own policy.
- Fixed Android App Store apps installed during the setup experience being marked "Failed" even though they installed successfully. An app that a device status report mentions in neither its application reports nor its non-compliance reports now stays pending for a later report instead of failing immediately, and is marked failed only if it is still unreported after `server.vpp_install_reap_timeout` (24 hours by default). An app that was marked failed is now marked installed if a later status report shows it installed.
- Fixed Android hosts that re-enroll into a different fleet still offering the previous fleet's self-service apps in the managed Google Play Store.
- Fixed Android actions such as lock showing an error even though they succeeded, or being silently dropped, when the Android Management API rate limit was hit more than once in a row. Fleet now retries a rate-limited call only when the retry can start in time for the response, and otherwise reports the failure right away.
- Fixed the Android stuck-command reconciler stopping on every run when a pending command from a previously connected Android Enterprise was at the head of the queue. Such commands are now skipped and marked as failed after 7 days.
- Fixed company-owned Android hosts showing MDM status "Pending" instead of "Off" after Android MDM is turned off. A migration corrects hosts that were already affected.
- Fixed Android host details keeping the original enrollment date after a device re-enrolled, and Android hosts never updating their last-seen time after they were first added.
- Fixed an Android host's disk space changing to "Not supported" after the device re-enrolls. Enrollment payloads carry no storage measurement, so Fleet now keeps the last known value until the next status report reports one.
- Fixed macOS Fleet-maintained app installs replacing an app that wouldn't quit (for example, when the end user canceled a prompt to save changes). The install now fails and leaves the running app in place, instead of deleting the running copy's files and reporting the update as installed.
- Fixed Google Chrome and other Chromium browsers not being detected as open after they auto update themselves.
- Fixed updating an in-house app package failing if a host had failed to install it without verifying.
- Fixed in-house app installs with an MDM error response not being set to failed.
- Fixed repeated verification commands being sent after canceled in-house app and VPP app installs.
- Fixed in-house app (`.ipa`) uploads reading their name, version and bundle identifier from an embedded framework or extension instead of the app itself. Metadata is now always read from the app's own `Payload/<App>.app/Info.plist`, so it no longer depends on the order entries were written to the archive.
- Fixed uninstall scripts for Fleet-maintained apps whose Homebrew cask quits the app by a wildcard bundle ID (balenaEtcher, Default Folder X, Elgato Wave Link, and VyprVPN): the script now quits every running app that matches before removing it, instead of skipping the quit.
- Fixed the Shottr Fleet-maintained app reporting hosts as out of date when the Homebrew version carries a letter suffix (e.g. "1.9.3b") that macOS omits from `bundle_short_version`.
- Fixed the Wispr Flow Fleet-maintained app leaving the installed bundle owned by root, which blocked Wispr Flow's built-in updater and prompted the user for admin credentials on every launch.
- Fixed a failed self-service install with an available update showing as "Update available" with a grey icon on the My device > Self-service page; it now reads "Failed" with the red error icon and opens the Install details modal with the failure details and Retry button, matching every other failed-install surface. The same change applies to the Host details > Software > Library table for admins.
- Fixed the software title page under "All fleets" so it no longer renders per-fleet management surfaces (Actions dropdown, header pills, Library section) with no fleet to act on. Previously, clicking an Actions item stashed modal state that fired open when the user switched to a specific fleet.
- Fixed the "Show details" affordance on install, uninstall, and App Store install activities so it's hidden on Fleet Free, since the install details modal fetches data from premium-only endpoints.
- Fixed the software details modal so that hovering the vulnerabilities list shows the first 10 CVEs and a count of the rest, instead of a tooltip that runs off the screen when software has many CVEs.
- Fixed Observer+ seeing an enabled Update button in the Update details modal on the host details Software > Library tab, which produced a 403 when clicked. Observer+ can still open the modal to view the available version, but the Update button is now hidden for users without software write permission.
- Fixed the missing Run policy action for global Observer+ users on Unassigned policies.
- Fixed the script details modal showing a duration from script output instead of the configured timeout when a script times out.
- Fixed a "This host is offline" error appearing on the host details and My device pages for a newly enrolled host that has not reported vitals yet, such as a Windows host running setup experience.
- Fixed the "Hosts online" and "Vulnerability exposure" charts returning no data when filtered by the Linux platform.
- Fixed "Clear all" in the Vulnerability exposure chart settings on the dashboard clearing filters on both the Hosts and Software tabs. It now clears only the tab you are on, and appears only when that tab has active filters.
- Fixed hosts picked under "Exclude hosts" after clicking "Clear all" in the Vulnerability exposure chart settings being ignored when the filters were applied.
- Fixed the dashboard's "View all software" button so the Software page opens with the selected fleet, and with the vulnerable filter applied when you're on the "Vulnerable" tab.
- Fixed the Dashboard's "Welcome to Fleet" card repeating the "vitals are taking longer than expected" toast on every subsequent host re-fetch (e.g. on window focus) instead of once per refetch attempt.
- Fixed host details and the Welcome to Fleet card scheduling an extra polling loop on every tab switch during a refetch.
- Fixed host details and the Welcome to Fleet card restarting a refetch and repeating the timeout toast when the tab regained focus after a refetch had already timed out.
- Fixed Cancel and Close in live reports and policies to return to the details page when the run starts there, while preserving the return to the edit page for runs started from Edit.
- Fixed report tables with per-column filter inputs (live report, host details report) to show a "No items match the current search criteria" message when a column filter matches zero rows, instead of a blank table body.
- Fixed the login, forgot-password, and reset-password forms to validate on blur, clear errors on focus, and surface every inline error when the submit button is clicked, so users no longer only find out what's wrong after a failed request.
- Fixed the confirm-invite form so pristine required fields stay silent until submit and password mismatch shows on the confirmation field only. Also fixed the "Confirm password" field prefilling itself with the value of "Password" on load, which previously let an unedited invite submit with the confirmation matching the incoming default.
- Fixed the SMTP settings card: server and port now report their own errors instead of sharing a single "Port" placeholder, and pristine required fields no longer render red on the first blur of any other field.
- Fixed the Fleet web address settings card to disable submit only during an in-flight save (it was disabling whenever any error had ever been computed).
- Fixed the Jira and Zendesk integration form to show a specific inline error on each empty required field instead of silently disabling the submit button behind a "Complete all fields" tooltip.
- Fixed the Save button on a fleet's agent options page spinning forever when the YAML has a syntax error.
- Fixed navigating to `/device` without a device token showing a blank page instead of the 404 page.
- Fixed modals rendering wider than the window at narrow viewports (768px), which cut off their edges and padding.
- Fixed the My device > Self service table overlapping column headers and cells at narrow widths by making it horizontally scrollable.
- Fixed toast notifications rendering shifted right of viewport center; they now sit horizontally centered at the bottom of the viewport.
- Fixed the upload list's rounded corners squaring off while the first or last row is hovered, on pages such as Controls > Scripts and Controls > OS settings > Configuration profiles.
- Fixed an icon misalignment in the hosts table for MDM status, and on the host details page in the host vital.
- Fixed the font and spacing of the performance impact value in the report modal opened from the activity feed.
- Fixed missing spacing between the IdP signature certificate label and the download button, and the button stretching full width, on the Okta conditional access modal.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.94.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-10-28">
<meta name="articleTitle" value="Fleet 4.94.0 | Configuration profile editor, self-service profiles, multiple App Store app versions, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.94.0-1600x900@2x.png">
