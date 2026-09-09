# Fleet 4.90.0 | Windows BitLocker controls, Android vulnerabilities, full DDM profile support, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/1oVLBB3CliQ?si=9Hivg8MCeXXhjwrj" title="0" allowfullscreen></iframe>
</div>

Fleet 4.90.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.90.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Windows: custom BitLocker configuration profiles](#windows-custom-bitlocker-configuration-profiles)
- [Android: OS versions and vulnerabilities on Software > OS](#android-os-versions-and-vulnerabilities-on-software-os)
- [Support for all DDM profiles and assets](#support-for-all-ddm-profiles-and-assets)
- [Custom host vitals for every platform](#custom-host-vitals-for-every-platform)
- [Rename macOS, iOS, and iPadOS hosts](#rename-macos-ios-and-ipados-hosts)
- [macOS local account creation and password sync with any IdP](#macos-local-account-creation-and-password-sync-with-any-idp)
- [Edit a configuration profile's labels or contents without deleting it](#edit-a-configuration-profiles-labels-or-contents-without-deleting-it)
- [Upload multiple custom packages for the same software title](#upload-multiple-custom-packages-for-the-same-software-title)
- [New log destination: Splunk](#new-log-destination-splunk)

### Windows: custom BitLocker configuration profiles

IT Admins can now upload a custom BitLocker configuration profile for Windows hosts, using  [`FLEET_MDM_ENABLE_CUSTOM_DISK_ENCRYPTION` server configuration option](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-enable-custom-disk-encryption). This means Windows disk encryption can be customized beyond Fleet's built-in BitLocker controls, matching the flexibility already available for macOS.

Only self-managed users and customers can modify Fleet server configuration. If you're a managed-cloud customer, please reach out to Fleet about modifying the configuration.

For users that already have `FLEET_MDM_ENABLE_CUSTOM_FILEVAULT` enabled, no changes are necessary. Fleet just added a second, cross-platform name for this key.

GitHub issue: [#43518](https://github.com/fleetdm/fleet/issues/43518)

### Android: OS versions and vulnerabilities

The **Software > OS** page now shows Android OS versions and their known vulnerabilities alongside every other platform Fleet tracks. Security Engineers get one place to see OS-level exposure across a fleet that includes personally-owned (BYOD) Android hosts, with the Android security update version formatted as a date (for example, `2026-07-01`) for easier tracking against Google's monthly security bulletins.

GitHub issue: [#35075](https://github.com/fleetdm/fleet/issues/35075)

### Support for all DDM profiles and assets

Fleet now supports uploading any Apple declarative device management (DDM) configuration or asset, for both the device and user channel. IT Admins can deploy DDM profiles like the Safari extensions settings declaration and reference DDM assets from those profiles. This way, anytime Apple ships new DDM features, IT Admins can use them on day one.

GitHub issue: [#38986](https://github.com/fleetdm/fleet/issues/38986)

### Custom host vitals for every platform

IT Admins can now define custom host vitals, like an asset tag or warranty expiration, for all platforms (macOS, Windows, Linux, iOS/iPadOS, and Android). Custom vitals appear on the **Host details** page and can be used to create labels and as variables in scripts and configuration profiles, so values from another system can drive automation everywhere in your fleet. [Learn more](https://fleetdm.com/guides/custom-host-vitals).

GitHub issue: [#44954](https://github.com/fleetdm/fleet/issues/44954)

### Rename macOS, iOS, and iPadOS hosts

IT Admins can now set a host name template on the **Controls** page for a fleet that applies to macOS, iOS, and iPadOS hosts. This gives every Apple host a standard naming convention, often including its serial number, without having to build a custom automation that sends an MDM rename command to each host. [Learn how](https://fleetdm.com/guides/rename-hosts-with-a-naming-template).

GitHub issue: [#38806](https://github.com/fleetdm/fleet/issues/38806)

### macOS local account creation and password sync with any IdP

During automated (ADE) enrollment, Fleet can now create the end user's macOS local account and keep its password in sync with any identity provider that supports OAuth Resource Owner Password Grant (ROPG) (e.g. Okta). End users get one password, meeting your organization's requirements, to unlock their Mac, their keychain, and third-party tools. [Learn more](https://fleetdm.com/guides/deploying-apple-account-provisioning-with-fleet).

The [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin) is required for local account creation and password sync. Add the app from the Fleet-maintained catalog and configure it to install during new Mac setup. [Learn how](https://fleetdm.com/guides/setup-experience#install-software).

GitHub issue: [#45524](https://github.com/fleetdm/fleet/issues/45524)

### Edit a configuration profile's labels or contents without deleting it

IT Admins can now edit a configuration profile. This includes the profile's labels (switching between include any, include all, and exclude any) or its contents directly from **Controls > OS settings > Configuration profiles**, without deleting and re-uploading it. This works across macOS, Windows, and Android profiles, and it makes staged rollouts easier since editing a profile no longer requires removing the older version from the hosts that already have it.

GitHub issue: [#38869](https://github.com/fleetdm/fleet/issues/38869)

### Upload multiple custom packages for the same software title

IT Admins can now upload up to 10 custom packages for the same software title in the same fleet. This makes it possible to deploy different versions or architectures, like Arm versus Intel builds, or run staged rollouts, using labels to target the right package instead of maintaining separate fleets for each variant.

GitHub issue: [#28108](https://github.com/fleetdm/fleet/issues/28108)

### New log destination: Splunk

Fleet can now send reports and other osquery logs directly to Splunk, without setting up Firehose as a middleman first. Learn how to [send reports directly to Splunk](https://fleetdm.com/guides/log-destinations#splunk).

GitHub issue: [#26333](https://github.com/fleetdm/fleet/issues/26333)

## Changes

### IT Admins
- Added the ability to upload multiple custom packages (up to 10) for the same software title on a team, so IT admins can deploy different versions or architectures (for example, Arm vs. Intel builds or staged rollouts) to label-scoped hosts instead of splitting them across teams. When a host matches more than one package, the first-added package is installed.
- Added support for editing existing configuration profiles (Apple `.mobileconfig`, Apple DDM declarations, Windows, and Android) in place via `PATCH /api/v1/fleet/configuration_profiles/:profile_uuid`.
- Added custom host vitals: admins can define custom host fields, set their values per host manually or via the API, and reference them as `$FLEET_HOST_VITAL_<id>` variables in scripts and configuration profiles.
- Added the ability to enforce a host naming template on macOS, iOS, and iPadOS hosts under Controls > OS settings > Host names, for a fleet or for "No team" (Fleet Premium).
- Added `POST /api/v1/fleet/host_name_template` to set or clear the naming template (`fleet_id` omitted or `0` targets "No team"); an empty template clears it without renaming any host.
- Added a `name_template` key under `controls` in GitOps for fleets and "No team", and included it in `fleetctl generate-gitops` output.
- Added a "Host name" row with enforcement status (Enforcing, Verifying, Verified, Failed) to the host details OS settings modal, including a resend action via `POST /api/v1/fleet/hosts/{id}/name_template/resend`.
- Added host name enforcement statuses to the Controls OS settings aggregate cards and the `os_settings` host filter.
- Added the `edited_host_name_template` activity.
- Added support for Python (`.py`) script-only software packages, which can be uploaded as custom packages (the file contents become the install script) and installed on macOS and Linux hosts, via the UI, REST API, and GitOps.
- Added support for provisioning macOS users during setup and keeping passwords in sync with any OAUTH ROPG supporting IdP via the Fleet Desktop app on macOS 26+ hosts.
- Added UI for configuring Apple account provisioning (FPSSO) in the integrations settings.
- Enabled Microsoft Entra conditional access for self-hosted Fleet Premium instances (previously available only on Fleet Cloud). The `microsoft_compliance_partner.proxy_api_key` server configuration has been removed; the feature is now gated on the Fleet Premium license tier.
- Added native Splunk HEC log destination for osquery status, result, and audit logs.
- Added support for escrowing disk encryption recovery keys from Linux hosts that use TPM-backed full-disk encryption (e.g. Ubuntu 26). On these hosts, orbit escrows a dedicated Fleet-owned snapd recovery key silently, without prompting the end user for a passphrase.
- Added `FLEET_MDM_ENABLE_CUSTOM_DISK_ENCRYPTION` (`mdm.enable_custom_disk_encryption`) as a cross-platform alias for `FLEET_MDM_ENABLE_CUSTOM_FILEVAULT`. When set, it allows both custom Apple MDM profiles for FileVault and custom Windows configuration profiles for BitLocker.
- Enabled "Turn off MDM" button for offline macOS hosts. The unenroll command is now queued and delivered when the device comes back online, consistent with iOS/iPadOS behavior.
- Added enrollment profile URL to the macOS tab in the "Add hosts" modal, with enrollment type selection (company-owned or personal/BYOD) for MDM users.
- Added support for targeting declarations to the user channel on macOS.
- Added the ability to handle DDM assets, and unblocked more declaration types.
- Added the certificates list to the host details page for Windows hosts, showing each certificate's scope (System or User). This requires osquery 5.23.1 or higher on the host.
- Added a "View certificate" modal to Controls > OS settings > Certificates so admins can inspect and copy an existing certificate's details.
- Surfaced hardware-bound ACME certificates on macOS host vitals by retrieving them via the MDM `CertificateList` command when an ACME-bearing configuration profile is installed or re-installed.
- Added "Targeted platforms" column and platform filter dropdown to the Policies page.
- Added optional `platform` query parameter to `GET /api/v1/fleet/policies` and `GET /api/v1/fleet/fleets/{id}/policies` to filter policies by targeted platform.
- Added public IP address to host search, so that searching by IP now matches both the primary (private) IP and the public IP.
- Added Zorin OS as a recognized Linux platform. Hosts running Zorin OS now enroll with `platform=zorin`, appear in the Linux disk-encryption summary, support `.deb` software installs, can be targeted by label platform filters, and have CVEs matched against the underlying Ubuntu LTS OVAL feed (Zorin 16 → Ubuntu 20.04, 17 → 22.04, 18 → 24.04). Unknown future Zorin versions fall through to an unsupported platform string so vulnerability scanning is skipped rather than served stale data from an aging LTS feed.
- Added support for CachyOS (an Arch-based Linux distribution) as a recognized Linux platform.
- Added an "Operating systems" card to the dashboard when Linux or Android is selected.
- Added installed version and available version columns to the self-service software table on the My device page.
- Added the "Applications" / "Full inventory" software filter to the Fleet Desktop **My device > Software** tab for macOS hosts, matching the host details page.
- Added the asynchronous live query endpoint (`POST /api/v1/fleet/reports/run`) to the API endpoints catalog so it can be granted to API-only users that have a restricted API endpoint allowlist.
- Added audit activities when secret variables are created or updated through the `PUT /api/latest/fleet/spec/secret_variables` endpoint.

### Security Engineers
- Added vulnerability (CVE) reporting for Android OS versions on the Software > OS page, where Android previously showed as "Not supported."
- Folded the Android security patch level into the host's OS version so Android versions read as "Android 16 (2026-05-01)", giving vulnerability-relevant granularity per patch level.
- Updated CIS Benchmark policies for Windows 10 Enterprise to align with the CIS Microsoft Windows 10 Enterprise Benchmark v4.0.0 (added, removed, and updated policies per the v4.0.0 change history).
- Added automatic renewal for SCEP and ACME certificates issued by external certificate authorities (Okta Conditional Access, Okta Verify, Hydrant ACME). Add `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` to the certificate's Subject OU to enable.
- Renamed `$FLEET_VAR_SCEP_RENEWAL_ID` to `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`. The legacy name still works.
- Enabled automatic renewal by default in Fleet's generated Conditional Access profile. Existing customers can opt in by redeploying the User scope profile.
- Windows configuration profiles that use a Fleet-proxied SCEP certificate (custom SCEP proxy, NDES, or Smallstep) now report "Verified" only after Fleet observes the issued certificate on the host, instead of reporting "Verified" as soon as the host acknowledged the profile. They report "Failed" when the SCEP proxy request returns an upstream error, or when the certificate is still missing from the host an hour after delivery (once Fleet can confirm the certificate's store was readable).
- Removed the validation, added in Fleet 4.89.0, that rejected custom SCEP proxy certificate authority challenges containing characters outside the ASN.1 PrintableString set (for example, an underscore). Apple devices can enroll certificates using such challenges, so they are accepted again. A fix for Windows certificate enrollment failing with these challenges will ship separately.
- Rejected empty and whitespace-only enroll secrets when creating or updating teams.
- Restricted SCIM endpoint access to global admin users only.
- Removed the unused `/api/mdm/microsoft/auth` Windows MDM STS endpoint. Fleet always advertises the OnPremise auth policy, so no device ever contacted this endpoint. It now returns a 404. Windows MDM enrollment (Autopilot, Settings app, and fleetd-initiated) is unaffected.
- Added a `server_bypass_network_blocking` server config option to allow disabling all outbound network blocking protections for integration HTTP requests in production, for environments where egress is already constrained by external infrastructure.

### Bug fixes and improvements
- Improved software ingestion performance by removing a full table scan of `software_titles` table.
- Optimized memory usage of CVE chart cron job.
- Reduced MySQL reader load when listing hosts with `device_mapping=true` and a search query by evaluating device mapping as a per-row correlated subquery instead of a fully-materialized derived-table join, and by skipping it entirely in the host count query.
- Improved the performance of Windows MDM profile installation across large numbers of hosts by reducing database lock contention when recording command results.
- Improved performance of Orbit config endpoint by batching extension label-membership checks into a single database query.
- Improved performance of host config endpoint by caching scheduled query configuration.
- Improved efficiency of the scheduled query stats aggregation cron job.
- Added better indexing for the Get Next Apple MDM command query.
- Added a long-lived immutable `Cache-Control` header to content-hashed static assets under `/assets/` so browsers and CDNs can cache them across loads instead of refetching the JS/CSS bundle from origin every time.
- Removed the `fleetdm/bomutils` Docker dependency for generating macOS `.pkg` fleetd installers; the Bill of Materials and xar archive are now written by pure-Go code, so `fleetctl package --type pkg` no longer requires Docker, `mkbom`, or `xar`.
- Updated the Render deployment blueprint to use MySQL 8.0.44 (previously 8.0.24), fixing an "Error 1235 ... nesting of unions at the right-hand side" error on Render deployments.
- Improved GitOps consistency by validating batch-applied Windows configuration profiles against the server's current MDM configuration state, while continuing to support previewing (dry run) a config that enables Windows MDM and applies profiles in a single run.
- Added a check for duplicate patch policies when applying GitOps.
- Added an error when `fleet_maintained_app_slug` is set on a non-patch policy in a GitOps yaml file.
- Surfaced a more detailed error message in GitOps if user doesn't have server_private_key configured.
- Improved error message when a mobileconfig profile contains unescaped special characters (e.g. `&`, `<`, `'`, `>`) that cause illegal base64 data errors during plist parsing.
- Updated the invalid NDES admin credentials SCEP error message to point to the correct UI location (Settings > Integrations > Certificate enrollment).
- Improved the Windows MDM enrollment server log for unsupported username and password (OnPremise) enrollment: a device that is not joined to Microsoft Entra ID now receives a clear server log message to join Microsoft Entra ID or enroll with fleetd.
- Added anonymous usage statistics reporting the number of macOS and Windows hosts enrolled in Fleet's MDM.
- Renamed "Create" buttons and links to "Add" across the Fleet UI for consistency.
- Updated link styles in the UI.
- Updated the 404 page with a new illustration and copy consistent with the rest of the app.
- Updated the 500 and 403 error pages to match the design system and reuse the app navigation so the 500 page no longer shows broken image elements.
- Improved the user menu to show individual settings sections for admins.
- Updated Windows MDM end user experience radio button labels from Automatic/Manual to Fleet agent-driven/End user-driven to reduce confusion with MDM status terminology.
- Updated relative "time ago" timestamps to show days instead of months when the timestamp is less than 90 days ago.
- Updated the message shown when refetching a host's vitals takes longer than expected to reflect uncertainty rather than failure, on the host details page, the My device page, and the dashboard's "Welcome to Fleet" card.
- Clarified the delayed host vitals refetch banner to reflect that a refetch was sent and the UI will update when the host responds.
- Removed the default platform filter on the "hosts online" chart, so iOS, iPadOS, and Android hosts are now included by default alongside desktop platforms.
- Removed the elevated white background container from the loading spinner for a flatter, more consistent look.
- Removed the blue active-state background flash when clicking a row in a single-select data table (e.g., **My device > Policies**).
- Updated missed ABM references to AB.
- Hid the Self-service "Install all" button on the unfiltered "All" category so end users can't queue an install of the entire catalog in one click. The button still appears when a specific category is selected.
- Hid self-service categories that have no available software from the category filter on the **My device** page, so users only see categories they can actually install from.
- Added a "no custom SCEP CA configured" empty state to the certificates card.
- Made form validation consistent across more forms (#40410 follow-up): validation errors now appear when leaving a field (on blur) and no longer appear before any input. This covers the policy automations "Other workflows" Destination URL, the add/edit user Email field, and the host status webhook Destination URL (both global settings and fleet settings).
- Fixed recurring Redis `MOVED` errors and silently-dropped report result-count increments on Redis Cluster deployments by grouping `query_results_count` keys by hash slot before pipelining.
- Fixed newly created or updated reports not appearing in the host details "Live report" modal or the reports list until a hard refresh.
- Fixed an issue where an identity provider (IdP) user associated with multiple hosts only had IdP host vitals populated on one of them. All matching hosts are now linked when the SCIM/IdP user is created.
- Fixed a bug where the Add software > App Store picker failed with an error for maintainer and technician roles because listing VPP tokens required admin access.
- Fixed an issue where the tooltip size of "Require BitLocker PIN" was bigger than normal.
- Fixed a bug where the DEP syncer could silently drop device enrollment events when interrupted mid-run (e.g. context cancelled). The sync cursor now only advances after device records are successfully written, ensuring affected devices are replayed on the next sync rather than lost.
- Fixed high memory usage (and occasional osquery watchdog worker restarts) on macOS hosts running the `software_macos` detail query, caused by an unbounded recursive filesystem walk used to de-duplicate Homebrew casks against the `apps` table. The check now uses bounded, non-recursive globs matching the standard cask layout. This also fixes casks that ship no `.app` bundle (e.g. `gcloud-cli`) being incorrectly dropped from software inventory.
- Fixed the "Missing hosts" summary card not showing on the Fleet Free dashboard when a platform other than "All" was selected.
- Fixed an issue where ACME urls would throw a 500 error on malformed URLs.
- Fixed macOS software titles being displayed with an embedded login-helper's name (e.g. "AmphetamineLoginHelper") instead of the parent app's name when the helper bundle shares a bundle identifier with the main app. Embedded `.app` bundles nested under `Contents/` are now excluded at ingestion, and existing mis-named titles are renamed by a one-shot migration that recomputes the name from the title's sibling software rows.
- Fixed long certificate names overflowing the delete certificate modal in Controls > OS settings > Certificates.
- Fixed the policies and users tables intermittently reloading and clearing the current selection or resetting to the first page when the browser window regained focus.
- Fixed a timeout when editing existing Windows configuration profiles for a large team via `POST /api/latest/fleet/mdm/profiles/batch` (GitOps). Now the request stays fast regardless of host count.
- Fixed label membership being incorrectly cleared when a label's query errors out on a host (e.g. the extension socket is unavailable) instead of returning zero rows; existing membership is now left unchanged when a label query fails.
- Fixed observers not seeing the "Show managed account" action on a macOS host's details page, even though the API already allows them to view the managed local account password.
- Fixed an issue where the truncated vulnerabilities list in the Update details modal did not show a tooltip listing the remaining CVEs.
- Fixed an incorrect error message where an `msix` file was parsed as an `ipa` file.
- Fixed sorting of fleets for fleet-level users.
- Fixed stale policy results inflating a host's failing policies count (shown in Fleet Desktop and the host's "Issues" column) after the policy no longer applied to the host (e.g. the host changed teams, or the policy's platform or label scope changed). Stale results are now cleaned up when the host reports its policy results.
- Fixed missing hover state on buttons and dropdowns inside cards in dark mode.
- Fixed the Policies page automations filter disappearing from the UI when switching to the "Unassigned" fleet and selecting a different automation type.
- Fixed the SSO sign-on button text overflowing by using a fixed "Sign in with SSO" label and showing the configured IdP name in a tooltip.
- Fixed an issue where premium MDM calls were being made on a Fleet Free license.
- Fixed cron jobs getting stuck in "expired" when a run is interrupted mid-flight (e.g. during server shutdown); the run now records a terminal "canceled" status, preserving any job errors, instead of being left "pending" until reaped to "expired".
- Fixed several styling issues on the end user enrollment page (BYOD info banner icon, active tab color, banner border, uneven QR code spacing) and added a "Learn more" link to the BYOD info banner. Also fixed enroll secret text incorrectly rendering in blue instead of black in the Add hosts modal.
- Fixed error in re-enrollment to Fleet with EUA on Linux with a different e-mail than the one used in the first enrollment.
- Fixed the vulnerability automations webhook "Destination URL" field to validate on blur (when the user clicks out of the field), consistent with other URL fields in Fleet, instead of only showing an error on save.
- Fixed Google Translate extension causing a 500-page when running live reports.
- Fixed a bug where some symbols changed height based on nearby characters in input fields.
- Fixed the Add certificate modal (Controls > OS settings > Certificates) to only list custom SCEP CAs in the "Certificate authority (CA)" dropdown, matching the modal's help text.
- Fixed an issue where tooltips for full name did not always show.
- Fixed server-side paginated tables (e.g. policies) landing on an empty state after deleting the last row on a page. The table now navigates back to a page with data instead.
- Fixed a server panic ("assignment to entry in nil map") when a host checked in for its osquery config while its agent options had a null `config`.
- Fixed team write endpoints (modify team, modify team agent options, and create team) so that they no longer return plaintext enroll secrets to users who cannot read them (such as GitOps), and applied the same secret masking to the list teams response.
- Fixed a bug where a custom Windows configuration profile/command could bypass Fleet's checks by using a scope-less LocURI.
- Fixed vulnerability detection for Citrix Workspace on Windows by normalizing the software version (e.g. `25.7.1.6` to `2507.1.6`) for Citrix Workspace entries whose name does not include the `YYMM` release, so the generated CPE matches NVD.
- Fixed Citrix Workspace LTSR detection on Windows to include cumulative updates (e.g. 2203 LTSR CU4), so their vulnerabilities report the correct LTSR `resolved_in_version` (e.g. `2402` for CVE-2024-6286) instead of the Current Release version.
- Fixed missing `resolved_in_version` for CVE-2025-63389 on Ollama (resolved in v0.12.4), which was absent because the NVD record only provides a `versionEndIncluding` constraint.
- Fixed vulnerability detection for Python packages on Ubuntu/Debian devices by stripping the "python3-" name prefix during CPE matching.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.90.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-08-05">
<meta name="articleTitle" value="Fleet 4.90.0 | Windows account controls, Android vulnerability visibility, and full DDM support">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.90.0-1600x900@2x.png">
