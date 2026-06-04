# Fleet 4.86.0 | Rotate local admin password, Windows setup experience, Platform SSO, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/Auc_KHR7HLk?si=bRTL8pJEat3id0zP" title="0" allowfullscreen></iframe>
</div>

Fleet 4.86.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.86.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Rotate local admin password](#rotate-local-admin-password)
- [Windows setup experience: cancel if software fails](#windows-setup-experience-cancel-if-software-fails)
- [Platform SSO during macOS Setup Assistant](#platform-sso-during-macos-setup-assistant)
- [Upload your org logo](#upload-your-org-logo)
- [Automatic SCEP and ACME certificate renewal](#automatic-scep-and-acme-certificate-renewal)
- [iOS/iPadOS software for user-enrolled hosts](#iosipados-software-for-user-enrolled-hosts)

### Rotate local admin password

Building on the [local admin account](https://fleetdm.com/releases/fleet-4-85-0#create-a-local-admin-account-during-macos-setup) introduced in 4.85, Fleet now lets admins rotate the hidden account's password directly from the **Host details** page. After an admin views the password, Fleet automatically rotates an hour later which limits how long a credential is valid after it's been seen. IT admins can also rotate immediately using the **Rotate password** button at any time. Every rotation is logged as an activity, whether triggered manually or automatically by Fleet.

GitHub issue: [#37142](https://github.com/fleetdm/fleet/issues/37142)

### Windows setup experience: cancel if software fails

Fleet now gives IT admins control over what happens when setup experience software fails during a Windows Autopilot enrollment ([OOBE](https://learn.microsoft.com/en-us/windows-hardware/customize/desktop/customize-oobe-in-windows-11#oobe-flow)). Turning on **Cancel setup if software fails** in **Controls > Setup experience** causes the device to display a failure screen and prompt the end user to restart if any setup software doesn't install successfully. Without this toggle, failed installs are surfaced in host details but the device proceeds to the desktop anyway. A `canceled_setup_experience` activity is logged when the feature triggers, making it easy to review what went wrong.

GitHub issue: [#38785](https://github.com/fleetdm/fleet/issues/38785)

### Platform SSO during macOS Setup Assistant

Fleet now supports configuring [Platform SSO with Okta](https://fleetdm.com/guides/deploying-okta-platform-sso-with-fleet) during macOS [Automated Device Enrollment (ADE)](https://fleetdm.com/articles/apple-device-enrollment-program). With this enabled, end users log in to their Mac using the same credentials they use for Okta. Microsoft Entra support is [coming soon](https://github.com/fleetdm/fleet/issues/45587).

GitHub issue: [#30674](https://github.com/fleetdm/fleet/issues/30674)

### Upload your org logo

IT admins can now upload their organization's logo directly to their Fleet instance with no external hosting required. Separate images can be set for light and dark mode. Upload during Fleet's initial setup or update later from **Settings > Organization settings**. Logos set here appear in Fleet's masthead and can also be managed via the API or [GitOps](https://fleetdm.com/docs/configuration/yaml-files).

GitHub issue: [#39016](https://github.com/fleetdm/fleet/issues/39016)

### Automatic SCEP and ACME certificate renewal

Fleet automatically re-pushes configuration profiles containing SCEP or ACME certificates before they expire. This now includes certificates that aren't proxied through Fleet. This covers, for example, certificates deployed for [Okta conditional access](https://fleetdm.com/guides/okta-conditional-access-integration) (SCEP) and Okta Verify (SCEP with a static challenge). The renewal logic follows the [same pattern](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal) already used for Fleet-proxied SCEP certificates. No Fleet configuration changes are required; just include the `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` variable in the certificate profile's organizational unit (OU) and Fleet handles the rest.

GitHub issue: [#40639](https://github.com/fleetdm/fleet/issues/40639)

### iOS/iPadOS software for user-enrolled hosts

Fleet now supports installing Apple App Store (VPP) and in-house (`.ipa`) apps on iOS and iPadOS hosts enrolled via [Account-driven User Enrollment](https://support.apple.com/guide/deployment/account-driven-enrollment-methods-dep4d9e9cd26/web) with a Managed Apple Account. IT admins can install apps from the **Host details** page, and end users can install from self-service. Setup experience software also installs automatically on enrollment.

GitHub issue: [#31138](https://github.com/fleetdm/fleet/issues/31138)

## Changes

### IT Admins

- Added automatic rotation of managed local admin account passwords after they have been viewed.
- Added a `require_all_software_windows` setting to cancel the Windows setup experience if any software install fails during Autopilot enrollment, matching the existing macOS behavior.
- Added GitOps support for uploading custom org logos. `fleetctl gitops` accepts `org_logo_path_dark_mode` and `org_logo_path_light_mode` keys to upload local files, and `fleetctl generate-gitops` exports Fleet-hosted logos as local files alongside path keys while keeping external URLs as `org_logo_url_*_mode` keys.
- Added support for installing VPP and in-house (`.ipa`) apps on iOS and iPadOS hosts enrolled via Account-Driven User Enrollment with a Managed Apple Account.
- Enabled self-service software installs from the My device page for user-enrolled iOS and iPadOS hosts.
- Enabled setup experience software in Controls > Setup experience to install automatically on user-enrolled iOS and iPadOS hosts at enrollment.
- Provisioned a VPP client user per Managed Apple Account on first install, and associated VPP licenses to the user rather than the device, supporting Apple's up-to-5-devices-per-user licensing semantics.
- Added managed app configuration for iOS and iPadOS apps (VPP and in-house), configurable via UI, REST API, and GitOps, with `$FLEET_VAR_*` substitution.
- Added support for VPP apps purchased from non-US-based Apple Business accounts.
- Added the ability to upload a custom organization logo for light and dark modes, hosted by Fleet, replacing the previous URL-only flow on the setup screen and organization settings page.
- Added `include_all` label scope to policies, and `include_all` and `include_any` label scopes to reports, including support via GitOps and `fleetctl`.
- Added a "Custom" target dropdown when creating or updating reports under the premium tier.
- Added an "Include all" option to the "Custom" target dropdown on Policies for premium users only.
- Added permissions for the GitOps user to list software titles.
- Added support for setting `gitops_mode_enabled` and `repository_url` via GitOps.
- Added output to GitOps for scripts, indicating how many scripts would be applied (dry run) or were applied.
- Added activity entries for retried software installs and script runs from policy automations.
- Added an activity when hosts fail enrollment profile renewal.
- Added activities when users create, edit, or delete labels (`created_label`, `edited_label`, and `deleted_label`).
- Added "Hosts online", "Hosts enrolled", and "Vulnerability exposure" charts to the dashboard.
- Added an option to convert and return a PEM-encoded X.509 certificate instead of a PEM-encoded PKCS7 envelope from the Request a Certificate endpoint.
- Added a deprecation warning when using `setup_experience.software` or `macos_setup.software` keys in config.
- Released `fleetctl` as a `pkg` for macOS.
- Released `fleetctl` as an `msi` for Windows.
- Enabled wiping a host to cancel all of its upcoming activities.
- Updated the default automatic enrollment profile, and added the ability to download and view the applied default profile.
- Updated OS version reporting for iOS and iPadOS to include the Rapid Security Response suffix (e.g. `(a)`) when the device reports a `SupplementalOSVersionExtra` field via MDM.
- Updated fleetd and MDM enroll activities to display the serial number and preserve the osquery-provided display name.
- Required the `--host` flag for `fleetctl get mdm-commands`, and deprecated `GET /api/v1/fleet/commands` without a `host_identifier`.
- Cleared host vitals on ABM host re-enrollment, with a config option to preserve past host activities.

### Security Engineers

- Added macOS 26 CIS Benchmark v1.0.0.
- Updated CIS Windows 11 Enterprise benchmark policies from v4.0.0 to v5.0.1, adding 17 new L1 policies and updating 42 existing policy titles.
- Surfaced hardware-bound ACME certificates on macOS host vitals by retrieving them via the MDM `CertificateList` command when an ACME-bearing configuration profile is installed or re-installed.
- Added SVG support for custom organization logos, with strict server-side sanitization to reject scripts and other unsafe SVG content.
- Added support for the `subject_alternative_name` field on Android certificate templates.
- Optimized OSV vulnerability scanning to query distinct software per OS version rather than per host, reducing redundant database queries for many hosts sharing the same packages.
- Improved vulnerability scanning performance by using a per-vendor product cache during CVE matching to optimize `translate_cpe_to_cve`.

### Bug fixes and improvements

- Updated Go to 1.26.3.
- Removed debug symbols from fleet and fleetctl executables to reduce binary size.
- Reduced database load from `GET /api/latest/fleet/device/{token}/desktop` and other Fleet Desktop endpoints when invalid or expired device auth tokens are presented, by resolving the token to a host ID with a single-table indexed lookup before running the multi-join host-details query.
- Improved Windows MDM performance when transferring large numbers of hosts between teams or applying bulk profile changes. These operations now return quickly and roll out profile updates to Windows hosts in the background, so host check-ins and other MDM activity are no longer slowed down while a large change is in progress.
- Added a Redis-backed cache for host lookups on the osquery and orbit authentication paths. Successful lookups are cached for 60s (±10% jitter) and invalidated on writes that mutate cached host fields. Reduces reader-side DB load at scale without changing the HTTP contract. Requires Redis 6.2 or later.
- Added a missing uninstall option on the host software library even when an installer has no matching software in the host's inventory.
- Improved Windows MDM profile removal performance by scoping the desired-state subquery.
- Improved Windows MDM profile removal performance by skipping redundant database writes for verified-remove ACKs.
- Consolidated non-variable templated Windows MDM profile command inserts from one per-profile to a single bulk insert.
- Added a periodic cron job to clean up the Windows MDM command queue, reducing write pressure during ACK transactions.
- Made host team assignment sticky across orbit and osquery re-enrollments.
- Improved errors returned from the API when running fleetctl commands by dropping path and status code.
- Improved validation of order parameters on list endpoints.
- Added the `orbit.debug_logging_on_enroll_duration` agent option to enable orbit debug logging for a specified time period after enrollment.
- Improved validation for invalid `order_key` values in `/api/v1/fleet/commands`, `/api/v1/fleet/mdm/commands`, and `/api/v1/fleet/mdm/apple/commands` endpoints.
- Improved the error message when the `name` key is omitted from a GitOps YAML file.
- Improved the error message when deleting a label used for targeting a software installation.
- Updated `fleetctl gitops` to warn when `labels:` is specified in no-team or unassigned files, where it is not supported.
- Updated the expired Fleet Premium license CLI banner to link to https://fleetdm.com/learn-more-about/downgrading instead of a stale FAQ anchor.
- Updated the Edit label page to reference "fleets" instead of "teams" when a label is associated with a fleet.
- Updated the setup experience Users card with a link to PSSO local account documentation.
- Updated empty state copy to be action-oriented. Headers describe the current state ("No hosts", "No policies for this fleet") instead of prompting action. Body text explains what to expect. CTA buttons are explicit ("Add policy", "Schedule a report") and permission-gated.
- Updated empty states on Hosts, Reports, Policies, and Software pages so search bars, filters, and dropdowns remain visible but disabled when empty, avoiding layout shift when the first item is added. Item count remains visible.
- Updated Settings, Fleets, Ticket destinations, Certificates, and Identity provider pages with consistent page descriptions and learn-more links.
- Updated empty state visuals to a fresher, consistent design.
- Updated timestamps with tooltips on the host Vitals component to always use `cursor: pointer`.
- Updated the version of the checkout action in the `fleetctl new` template to avoid Node warnings.
- Updated the MSI builder to skip packaging the unusable "dummy" secret value when building `fleetd-base.msi` for Autopilot installs.
- Scoped install commands for user-enrolled hosts to the host's Managed Apple Account (`clientUserIds`) instead of `serialNumbers`, so apps install on the correct user account on the device.
- Surfaced a clear host-level error when license association fails during install (for example, no licenses available or the user has reached the 5-device limit) instead of failing silently.
- Made `created_at` upper-bound filtering consistent on the list activities API. The endpoint now caps results at `now` by default whether or not `start_created_at` is provided, matching the documented behavior of `end_created_at`.
- Unified access to global and team policies in the UI by using the now-generic `GET /api/latest/fleet/policies/:id` endpoint.
- Wrapped `Get-ItemProperty` calls in try/catch blocks during registry enumeration to gracefully handle terminating exceptions (e.g. `System.InvalidCastException`) from malformed registry entries, logging the offending path instead of aborting.
- Replaced the cryptic "startTLS error: ..." flash with a prescriptive message when saving SMTP settings fails because SSL/TLS is disabled but STARTTLS is still enabled. Added a tooltip on the SSL/TLS checkbox pointing to the STARTTLS toggle in Advanced options.
- Removed a dead SQL condition in `hostVPPInstalls` that was misleading but harmless. Android VPP apps never produce `nano_command_results` entries (they use Google's Android Management API, not nanoMDM), so the previous `(hvsi.platform != 'android' OR ncr.id IS NULL)` guard was a tautology. Replaced with a clarifying comment.
- Fixed filtering on the `/api/v1/fleet/labels/:id/hosts` endpoint.
- Fixed the `usage_statistics` cron failing against fleetdm.com when a large number of near-identical network errors accumulated in the error store.
- Fixed `fleetctl gitops` failing with HTTP 500 on subsequent runs when a custom software icon's bytes were missing or had failed integrity in the icon store. The server now returns a 409 Conflict from the metadata-only icon update path, and the gitops client falls back to a full upload to recover the bytes automatically.
- Fixed SAML JIT provisioning so `FLEET_JIT_USER_ROLE_*` attributes with empty, whitespace-only, or missing values are treated as `null` and ignored instead of failing SSO login.
- Fixed an issue where GitOps controls with only certain keys would not be seen as set.
- Fixed recovery lock password not being retrievable for hosts transferred to a team with recovery lock disabled.
- Fixed Fleet's Docker image failing to start in Kubernetes with an `unknown userid` error, triggered by a fleetctl dependency side effect.
- Fixed a GitOps failure ("converting NULL to uint is unsupported") when moving labels from global to fleet scope, caused by deleted label associations with NULL `label_id` values in `mdm_configuration_profile_labels` and `mdm_declaration_labels`.
- Fixed `fleetctl gitops --dry-run` intermittently failing with "Resource Not Found" when a team's `software` config was empty.
- Fixed the MDM SSO callback returning a "missing profile" error for Android enrollment when Apple MDM is not configured.
- Fixed the team PATCH endpoint rejecting `mdm.enable_disk_encryption` on Fleet deployments where only Windows MDM is configured. Team-level BitLocker enforcement can now be toggled when either Apple MDM or Windows MDM is configured.
- Fixed an issue where the disk encryption table on the Controls > Disk encryption page did not support horizontal scrolling at narrow viewport widths.
- Fixed Linux total disk space being double-counted when a filesystem was bind-mounted at multiple paths (e.g. snap-confine's `/tmp/snap.rootfs_*`).
- Fixed `fleetctl gitops` rejecting `path:` values whose actual filenames contained glob metacharacters even when the file existed at that literal path.
- Fixed GitOps failing when it attempted to create a label and a consumer of that label (e.g. a profile) in the same run.
- Fixed `gitops --dry-run` to reject label specs with invalid `platform` values.
- Fixed the SSO invite acceptance flow by resolving the email from the invite token.
- Fixed batch script endpoints to return 404 Not Found instead of 200 when the batch execution ID does not exist: `/api/v1/fleet/scripts/batch/:id`, `/api/v1/fleet/scripts/batch/summary/:id`, and `/api/v1/fleet/scripts/batch/:id/cancel`.
- Fixed a class of silent SCEP managed-certificate renewal failures by recovering `host_mdm_managed_certificates` rows that previously got stuck after the cert ingest matcher missed linking a renewed certificate.
- Fixed the upcoming activity count on the host details page not updating after installing or uninstalling software.
- Fixed an issue where GitOps incorrectly rejected keys in Google Calendar API key JSON.
- Fixed 500 errors on `POST /api/v1/fleet/scim/Users` when the matched host was already mapped to a SCIM user. The host now gets reassigned to the newly-created SCIM user.
- Fixed an incorrect CPE match on the "slate" Homebrew program.
- Fixed a bug where applying GitOps to a script-only package by `hash_sha256` reference would wipe the install script, causing self-service installs to silently no-op.
- Fixed `fleetctl vulnerability-data-stream` to also download OSV (Ubuntu and RHEL) artifacts.
- Fixed a missing `deleted_policy` activity when a patch policy is removed by GitOps as a result of its underlying Fleet-maintained app installer being removed from the YAML.
- Fixed a nil-pointer panic in the Android Enterprise Pub/Sub endpoint that occurred when Google's Android Management API sent a device payload missing `hardwareInfo`, `softwareInfo`, or `memoryInfo`.
- Fixed an issue where, if a custom Apple MDM URL was set, SSO for end user auth would fail.
- Fixed slow load times and timeouts on the list MDM commands API (`GET /api/v1/fleet/commands`) on Fleet deployments with many Windows hosts. The endpoint now caps `per_page` at 1,000 (default 10) and `page` at 100. Requests above either limit return HTTP 400. To traverse beyond 100 pages, use cursor pagination via the `after` query parameter.
- Fixed GitOps dry-run to correctly detect the conflict when both `macos_manual_agent_install` and `macos_script` are configured under `setup_experience`. Previously, the dry-run would succeed while the actual GitOps run would fail.
- Fixed `fleetctl gitops apply` not clearing stale broken `mdm_configuration_profile_labels` rows after a referenced label was deleted, which caused profiles to remain enforced on hosts regardless of updated label targeting.
- Fixed `GET /api/v1/fleet/commands` returning a SQL error when called with `host_identifier` and the `after` cursor parameter, particularly with `order_key=command_uuid` or `order_key=hostname`.
- Fixed a UI bug where editing an existing global user to enable two-factor authentication failed with a 422 error.
- Fixed an issue where an old APNs cert would stay in memory until a restart, instead of correctly updating in place.
- Fixed a UI inconsistency with non-center-aligned Fleet premium messages on Fleet Free.
- Fixed a bug where duplicate software installers for Linux could be added.
- Fixed the "Back to host details" button on a report's details page navigating to the reports list instead of the host's details page after creating a report from a host.
- Fixed IdP host vitals (full name, department, groups) not populating on the host details page for macOS devices migrated from another MDM via the Tahoe (macOS 26+) end-user-authentication flow.
- Fixed `POST /api/v1/fleet/queries` returning HTTP 500 when `name` or `query` is JSON `null`. The endpoint now returns HTTP 400.
- Fixed `GET /api/latest/fleet/policies/:id` (and alias `GET /api/v1/fleet/global/policies/:id`) to return and properly populate team policies, and to perform an authorization check on team policies before returning.
- Fixed an issue where GitOps dry-run would not validate Apple config profile payload scope conflicts or the use of unknown Fleet variables in all types of profiles.
- Fixed Android hosts being auto-deleted by host expiry on every cleanup tick after re-enrolling, which previously caused an hourly enroll/delete loop while host expiry was enabled.
- Fixed an issue where replica lag could lead to devices not being assigned a setup experience profile on device sync from DEP.
- Fixed the Location and MDM status vitals on the My device page rendering as clickable links even though they had no associated modal, by rendering them as plain text in read-only contexts.
- Fixed the Export hosts button to always reflect the current sort, search, and filter state instead of potentially using stale values.
- Fixed a false-positive `update_conditional_access_bypass` activity that was created whenever any app config setting was changed while Okta conditional access was already configured with `bypass_disabled: true`. Also stopped the related side effect of clearing existing conditional access bypass records on those unrelated saves.
- Fixed UI elements in the script library not respecting GitOps mode when enabled.
- Fixed `POST /packs` with a JSON null name silently creating a pack with an empty name. The endpoint now returns a 400 Bad Request, matching the behavior for an empty-string name.
- Fixed stale "Selected hosts" on the Edit label page after a previous edit by invalidating the related query caches on success, and when navigating between manual labels by scoping the hosts cache per label and keying the form on the actual host set.
- Fixed subtle text alignment issues in the UI.
- Fixed a file descriptor leak in vulnerability processing where deleted `goval_dictionary` sqlite files were kept open until Fleet server restart.
- Fixed setup experience remaining stuck for up to 90 minutes after a software installer was edited or deleted while a host was installing it.
- Fixed the Policy details modal not closing when navigating back to the Host details page with the browser's back button.
- Fixed software titles list sorting to use display name instead of installer filename when a custom display name is set.
- Fixed the missing "Conditional access" section header on the Settings > Integrations > Conditional access page on Fleet Free.
- Fixed `fleetctl gitops` silently accepting labels with invalid parameter combinations (e.g. manual labels with query/criteria/platform).
- Fixed validation that rejected enabling end user authentication on Fleet deployments without Apple MDM configured. End user authentication covers macOS Setup Assistant, Windows MDM, and Linux Orbit enrollment, so the toggle now works on Windows-only and Linux-only fleets as long as the IdP is configured.
- Fixed an issue where the MDM solution name reported for a host could flip between values across osquery ingestions when the MDM server URL contained substrings matching multiple known MDM vendors.
- Fixed a bug where `enable_host_users` defaulted to `false` on a fresh Fleet install instead of the documented default `true`, causing the host details page to show "User collection has been disabled."
- Fixed the IdP "Department" host vital not populating for users whose IdP-to-SCIM mapping included enterprise extension attributes that Fleet does not store.
- Fixed the Actions dropdown in the Run script modal within the Host details page automatically closing after 2-3s.
- Fixed GitOps dry runs failing when a VPP app references a label that was added in the same run.
- Fixed a bug where enrolling an Android device on a Fleet instance with Apple MDM disabled produced a duplicate host record.
- Fixed Fleet-scoped users getting a 403 when viewing past activities on a host that has user-initiated activities (e.g. lock/wipe/run script/install software), and fixed missing permissions on host activity items for fleet-scoped users.

- See the [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.86.0) for the full list of bug fixes and improvements.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.86.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-05-29">
<meta name="articleTitle" value="Fleet 4.86.0 | Rotate local admin password, Windows setup experience, Platform SSO, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.86.0-1600x900@2x.png">
