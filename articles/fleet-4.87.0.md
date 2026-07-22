# Fleet 4.87.0 | 800+ new apps, custom OS updates, non-admin local accounts, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/Mv2kbxItSbI?si=1bPEQvi8BO6VLUIv" title="0" allowfullscreen></iframe>
</div>

Fleet 4.87.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.87.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [800+ new Fleet-maintained apps](#800-new-fleet-maintained-apps)
- [Custom OS update profiles](#custom-os-update-profiles)
- [Configuration profiles: Include + exclude](#configuration-profiles-include-exclude)
- [macOS local account: non-admin (standard) or skip](#macos-local-account-non-admin-standard-or-skip)
- [Self-service software categories](#self-service-software-categories)
- [Android commands: Lock, wipe, & clear passcode](#android-commands-lock-wipe-clear-passcode)
- [Policy automation continuous retry](#policy-automation-continuous-retry)
- [Command palette](#command-palette)

### 800+ new Fleet-maintained apps

_Available in Fleet Premium_

Fleet 4.87 adds 800+ new Fleet-maintained apps which brings [the catalog](https://fleetdm.com/software-catalog) to over 1,250 apps. IT admins can add any of these under **Software > Add software > Fleet-maintained** and deploy with a single click.

Windows gets its biggest catalog expansion yet. Highlights include:

- **Microsoft Office**, **PowerShell**, **PowerToys**, **Power BI**, **Power Automate**, and **SQL Server Management Studio** for Windows-centric environments
- **Git**, **Node.js**, **Python 3.13 and 3.14**, and **PostgreSQL 15–18** for development teams
- **Windsurf** and **Kiro** for developers using AI-powered coding IDEs
- **Dell Command Update** and **Lenovo Dock Manager** for hardware fleet management
- **Nessus Agent** for vulnerability scanning and **Bitwarden** for password management

New macOS apps include **Kiro**, **Codex**, and **OpenCode** for AI-assisted development, plus hundreds more tools across productivity, design, security, and media.

### Custom OS update profiles

_Available in Fleet Premium_

Fleet now supports deploying custom [Declarative Device Management (DDM) Software Update enforcement](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/softwareupdate.enforcement.specific.yaml) declarations on macOS, iOS, and iPadOS, as well as custom Windows profiles using the [Windows Update CSPs](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update). This gives IT admins full control over OS update enforcement, including the exact enforcement deadline time.

Fleet enforces mutual exclusion with its built-in OS update controls: configuring both returns a clear error, so nothing conflicts silently.

GitHub issue: [#38802](https://github.com/fleetdm/fleet/issues/38802)

### Configuration profiles: Include + exclude

_Available in Fleet Premium_

Configuration profiles now support combining the **Include any** label targeting, a host receives a profile if it matches any label in the include list, with the new **Exclude any** option. This way, IT admins can define broad inclusions and exclude specific hosts without writing complex label queries.

For example: deliver a Wi-Fi profile to all macOS devices (`include_any: macOS`) while excluding hosts tagged "Guest" or "Loaner." Both options work across all platforms: macOS, iOS, iPadOS, Windows, and Android.

GitHub issue: [#32073](https://github.com/fleetdm/fleet/issues/32073)

### macOS local account: non-admin (standard) or skip

_Available in Fleet Premium_

Building on the [local admin account](https://fleetdm.com/releases/fleet-4-85-0#create-a-local-admin-account-during-macos-setup) introduced in 4.85 and [password rotation](https://fleetdm.com/releases/fleet-4-86-0#rotate-local-admin-password) added in 4.86, Fleet now lets IT admins control the end-user account type during macOS Setup Assistant. On the **Controls > Setup experience > Users** page, choose **Standard** to create a non-admin end-user account, or **Skip** to skip end-user account creation entirely. This is useful when the hidden admin is the only local account the device needs. Selecting **Standard** or **Skip** automatically requires the hidden local admin to be created.

GitHub issue: [#41781](https://github.com/fleetdm/fleet/issues/41781)

### Self-service software categories

_Available in Fleet Premium_

IT admins can now create custom software categories to bucket applications by team, role, or project (e.g., "Product development") so end users can get fully set up for their projects. End users see an **Install all in category** button that installs all apps in a category, in alphanumeric order, with a single click.

GitHub issue: [#39018](https://github.com/fleetdm/fleet/issues/39018)

### Android commands: Lock, wipe, & clear passcode

_Available in Fleet Premium_

Fleet can now send lock, wipe, and clear passcode commands to Android hosts directly from the **Host details** page. For company-owned (fully managed) devices, all three commands are available. For personally-owned (BYOD) Android hosts, lock and clear passcode are available and scoped to the work profile. Each action is logged in Fleet's [audit logs](https://github.com/fleetdm/fleet/blob/main/docs/api/global-audit-logs.md). The [`fleetctl` CLI tool](https://fleetdm.com/guides/fleetctl) also supports these via `fleetctl mdm lock`, `fleetctl mdm wipe`, and `fleetctl mdm clear-passcode` commands.

GitHub issue: [#41683](https://github.com/fleetdm/fleet/issues/41683)

### Policy automation continuous retry

_Available in Fleet Premium_

A new **Run automation on every failure** option lets IT admins trigger software installation or script-run automations every time a host fails a policy check, not just the first time. If a host falls back out of compliance after an initial remediation or the initial remediation fails, Fleet automatically runs the fix again without manual intervention. 

GitHub issue: [#42651](https://github.com/fleetdm/fleet/issues/42651)

### Command palette

Fleet now includes a command palette. Press ⌘+K (or Ctrl+K on Windows and Linux) from anywhere in the app to instantly navigate to any page, trigger any action, or jump to any setting. The palette respects your role by showing or hiding items based on your permissions. Fleet Premium users with multiple fleets can jump directly to the fleet switcher with ⌘+Shift+F (Ctrl+Shift+F on Windows and Linux). Sub-pages let you search hosts, software titles, reports, and policies by name without leaving the keyboard.

GitHub issue: [#43757](https://github.com/fleetdm/fleet/issues/43757)

## Changes

### IT Admins
- Added 236 new Fleet-maintained apps for Windows, including Microsoft Office, PowerShell, PowerToys, Power BI, Power Automate, SQL Server Management Studio, Microsoft .NET Runtime 8 and 10, Git, Node.js, Python 3.13 and 3.14, PostgreSQL 15–18, Windsurf, Kiro, Dell Command Update, Lenovo Dock Manager, Nessus Agent, Bitwarden, Canva, Miro, Snagit, Tableau Desktop, VirtualBox, TortoiseGit, GitHub Desktop, and more.
- Added 727 new Fleet-maintained apps for macOS, including Kiro, Codex, OpenCode, Claude DevTools, Granola, Logitune, and hundreds more tools across development, security, productivity, and design.
- Added the ability to deploy custom OS update configuration profiles for Apple and Windows.
- Added support for issuing Lock, Wipe, and Clear passcode commands to Android hosts. Lock and Clear passcode work for both BYO (personal) and COBO (company-owned) Android hosts; Wipe is COBO-only. For BYO hosts, Unenroll now issues an AMAPI WIPE under the hood, which removes only the work profile and leaves personal data intact. All Android commands are issued with `duration=315360000s` (10 years), matching the pending-forever queue semantics Fleet uses for Apple and Windows MDM.
- Made the Wipe command available to Fleet Free users for Android (company-owned) hosts, in both the UI and the API. Wipe for macOS, iOS, iPadOS, Linux, and Windows hosts remains a Fleet Premium feature.
- Android host display name now uses "{IdP first name}'s {hardware model}" when an IdP account is associated.
- Reduced Windows MDM server and database load by relaxing the device management poll schedule from 1 minute to 8 hours for hosts running a version of fleetd that supports on-demand Windows MDM sync (1.57.0 and later). When commands are queued, the server wakes these devices through fleetd to start a management session, so command delivery stays near real-time. Hosts on older fleetd versions keep the previous poll behavior.
- Renamed Apple Business Manager (ABM) terminology to Apple Business (AB) in the API, GitOps YAML, and `fleetctl` CLI. The new `/api/v1/fleet/ab_tokens` and `/api/v1/fleet/mdm/apple/ab_public_key` endpoints, `mdm.apple_business` YAML key, and `fleetctl get mdm-ab`/`fleetctl generate mdm-ab` commands are canonical. The now-deprecated `/abm_tokens`, `/mdm/apple/abm_public_key`, `apple_business_manager`, `mdm-apple-bm` aliases continue to work for backwards compatibility and log a deprecation warning when used.
- `labels_exclude_any` can now be combined with `labels_include_all` or `labels_include_any` when uploading MDM configuration profiles, allowing hosts to be included by label membership and excluded by another set of labels simultaneously.
- Added support for setting the end user account type to `standard` for a standard (non-admin) user or `none` to skip end-user account creation, both requiring a local admin account.
- Added a "Continuous" option to policy automations that re-runs script and software automations on every subsequent policy failure, with editable automations now available directly on the policy create, edit, and details pages.
- Added the ability for users with the Technician role to transfer hosts between fleets (Fleet Premium only). Global technicians can transfer hosts via the Fleet UI (manage hosts and host details pages) and the REST API. Fleet-scoped technicians can transfer hosts between fleets they manage via the REST API.
- Added Self-service categories page (Premium) under Software > Library for managing custom categories per fleet, including add, edit, and delete flows.
- Added Categories button to the Software > Library page that navigates to the new categories page.
- Replaced the static category sidebar on the My device > Self-service page with a custom-category dropdown driven by the org's self-service categories, and added an "Install all (n)" button per category (with a confirmation modal) that posts to `/device/{token}/software/install_all?category_id=:id`.
- Added `macos_applications` filter for host software list.
- Added Fleet "Spotlight" - A command palette that opens when pressing Command + K or Control + K.
- Added a "My device" button on the host details User card so global admins can open the host's end-user My device page in a new tab; Fleet refreshes or generates the device auth token as needed so the link is always valid.
- Showed the end user's IdP full name (e.g. "Jane Doe's device") on the My device page header and browser tab when available; falls back to "My device" otherwise.
- Added support for configuring an optional SES sender domain.

### Security Engineers
- Added support for validating Microsoft Entra v2 access tokens during Windows MDM enrollment. Effective July 1, 2026, new on-premises MDM applications created via the Entra portal flow issue v2 access tokens whose audience (`aud`) is the application's client ID; adding the client ID lets these applications enroll Windows hosts. Existing v1 tokens (audience = Fleet server URL) continue to work unchanged.
- Hardened in-house iOS app distribution by requiring a per-install token in the manifest and package download URLs. The token is minted when an install is enqueued, bound to the target host, and expires after 6 hours, aligning the in-house download flow with the URL-token authentication already used by Fleet's MDM installer and software installer download endpoints.
- Added GCS IAM authentication support for software installers S3 storage using Google Application Default Credentials (ADC) bearer tokens instead of S3 HMAC keys. Configurable via `s3_software_installers_gcs_iam_auth`.
- Added GCS IAM authentication support for file carving S3 storage. Configurable via `s3_carves_gcs_iam_auth`.
- Added route-aware head sampling for OpenTelemetry trace export. When `tracing_enabled` is on, agent firehose endpoints (osquery distributed read/write, orbit ping/config, device desktop/ping) are sampled at 0.1% by default, admin reads at 2%, and everything else (enroll, SCEP, MDM checkin, cron jobs, GitOps batch) at 100%. Liveness probes (`/healthz`, `/version`, `/metrics`) are dropped unconditionally.
- Added `GET`/`PATCH /debug/trace_sampler` (admin only, behind the existing `/debug` auth) for adjusting ratios or flipping a 100% `force_full` debug window at runtime. Each Fleet replica polls the new `trace_sampler_settings` row every 60 seconds and applies changes without a restart.
- Updated the vulnerability processing guide to clarify Linux vulnerability scanning coverage, including a per-distribution table covering OS/kernel, system packages, and cross-platform packages and which scanner is used for each.

### Bug fixes and improvements
- Updated Go to 1.26.4
- Significantly improved performance of the Apple profile and DDM reconciler.
- Improved the performance of listing labels with host counts by aggregating membership counts in a single pass instead of a per-label subquery, and skipping the unnecessary join to the hosts table when the requesting user can see all hosts.
- Android profiles now use content checksums to determine when to re-sync, avoiding unnecessary re-delivery on unrelated policy changes.
- Long policy resolution text now wraps on the policy details page instead of being truncated.
- Updated initialization semantics around `api_endpoints`. The catalog is now loaded from the embedded YAML once at package initialization time.
- Added Python 3.14 and Python 3.13 as Windows Fleet-maintained apps.
- Normalized Python's reported version on Windows (e.g. `3.14.5150.0` -> `3.14.5`) so software inventory and vulnerability matching use the real version.
- Replaced the "Osquery" column with a richer "Agent" column on the Hosts page that shows Orbit version with a tooltip displaying osquery, Orbit, and Fleet Desktop versions.
- Hid "Issues" and "Private IP address" columns by default for new Fleet instances.
- Added hosts page tooltip to MDM status on hover.
- Added certificate rollover process to MDM assets tool.
- Added a migration cleanup tool for recovering failed starts after renumbered migrations.
- Added each platform's percentage of total enrolled hosts to the "Hosts enrolled" card tooltip on the dashboard.
- Updated conditional access policy query to use parameter binding for platform filter.
- Rejected Windows MDM configuration profiles that don't contain at least one supported SyncML top-level element (`<Replace>`, `<Add>`, `<Exec>`, or `<Atomic>`), so non-XML or empty payloads are caught at upload instead of failing on devices.
- Updated to now prevent deleting a label that is in use by an MDM configuration profile or declaration, returning an error instead of silently breaking the profile's label targeting.
- Raised the default `FLEET_REDIS_HOST_CACHE_TTL` from 60s to 180s and removed the reverse-index GETs that the host-update invalidation path performed. Together these reduce DB reader load and lower Redis CPU usage.
- Surfaced `continuous_automations_enabled` in GitOps YAML (read and generated by `fleetctl generate-gitops`).
- Stopped the 1Password autofill icon from appearing on Fleet UI inputs that are not credential fields.
- Hid the "Rotate password" button in the Recovery Lock password modal for users with the Observer role, instead of showing it as disabled.
- Updated Android Enterprise connect to surface real error messages to the user.
- Updated self-service activity copy to passive voice without an "end user" actor (e.g. "GitHub Desktop was installed on this host (self-service).") on both the host activity feed and the dashboard global activity feed.
- Updated GitOps error message about exceptions to include the URL to visit to disable exceptions.
- Updated the error displayed when GitOps encounters an unknown env var to account for cases where the string is a literal that needs escaping.
- Removed orphaned duplicate SCEP certificates from the per-user keychain automatically after an Okta conditional access profile is reinstalled or renewed on macOS hosts.
- Reduced the Apple MDM lock state cleanup timeout from 5 minutes to 1 minute, decreasing the time a recently unlocked host may still appear as locked in Fleet.
- Rejected Windows MDM configuration profiles whose `<LocURI>` is empty, starts with `/`, or contains `..` path traversal segments, so invalid OMA-DM URIs are caught at upload instead of failing on devices.
- Refactored `ListHostSoftware` and `ModifyAppConfig` into smaller helpers so nilaway can analyze them for nil-pointer dereferences.
- Refactored MDM profile label-targeting logic (include all/any, exclude any) into a shared platform-neutral package so Apple and Windows reconcilers use the same rules.
- Slimmed down the `POST /api/v1/fleet/targets` response to omit unused fields.
- GitOps now prints a message for each software package it will delete.
- Fixed the Add host modal so its read-only installer command fields can no longer be resized.
- Fixed an issue where the checkerboard would be colored based on relative percentages rather than relative absolute value.
- Fixed a race condition where deleting a policy while a host had an outstanding distributed query for that policy caused a foreign key constraint error during `/api/v1/osquery/distributed/write`.
- Fixed SCEP PKIOperation handler incorrectly decoding base64 `+` characters as spaces.
- Fixed software installer edits cancelling pending setup experience installs and causing setup experience to fail if all software is required.
- Fixed a bug where navigating to the Fleet root URL returned a 404 in subpath deployments.
- Fixed bug in `apply` to prevent `setup_experience` in software items from being renamed to `macos_setup`.
- Fixed a bug where the "Add custom variable" modal would clear entered values when switching focus to another browser tab or application window.
- Fixed `fleetctl preview` disabling dashboard chart data collection (Hosts online, Vulnerability exposure) on startup.
- Fixed a race condition after Windows BYOD MDM enrollment (Settings > Access work or school > Connect) where `mdm_windows_enrollments.host_uuid` stayed empty for several seconds, causing server-side enrollment lookups to miss. The enrollment is now linked to the Fleet host record at the first management session via OMA-DM DevDetail/SMBIOSSerialNumber instead of waiting for osquery's distributed-read backfill.
- Fixed MDM status column in the host table showing "On (automatic)" instead of "On (company-owned)".
- Fixed logout/login redirects to respect the URL prefix in subpath deployments.
- Fixed the `mdm_unenrolled` activity not appearing in a host's activity timeline on the host details page.
- Fixed software titles displaying the raw package name instead of the admin-set display name in the policy automations list and edit modal, the patch automation CTA, the hosts software filter pill, and the setup experience software row.
- Fixed an issue where ADE-enrolled macOS hosts didn't report FileVault until restarted.
- Fixed Android profiles temporarily failing when transferred to a team with certificates by ensuring certificates are provisioned before dependent profiles are applied.
- Fixed an issue where the "Get host's OS settings" API endpoint returned an error when only Android MDM was enabled.
- Fixed `fleetctl get fleets` (and `fleetctl get teams`) so the software section, including each app's `setup_experience` value, reflects the real configuration instead of being read from the (potentially stale) team config. Software is now fetched from the software titles and setup experience endpoints, which are the source of truth.
- Fixed an issue where GitOps would fail on the first run after deleting the bootstrap package in the UI.
- Fixed login failing with an "Authentication Required" error when Fleet is served over HTTP, by storing the auth token in a non-secure cookie outside of HTTPS contexts.
- Fixed Android devices losing their team assignment and certificate configuration when the host record is deleted and the device re-enrolls.
- Fixed a bug where host vitals labels (e.g. IdP group/department labels) scoped to a fleet/team never got any hosts. The membership cron only looked at global labels, and team-scoped IdP labels also failed to populate due to an incorrect SQL join.
- Fixed inline error for duplicate certificate name not showing when the conflicting certificate is on a different page.
- Fixed a server out-of-memory crash that could occur when Apple's VPP (App and Book Management) API repeatedly returned transient errors (HTTP 500 with Retry-After, or error 9646) during VPP API operations (e.g., app installs, user registration, license seat releases).
- Fixed Fedora wipe to delete btrfs snapshots (including read-only ones) before wiping the filesystem, preventing snapshots from surviving the wipe.
- Fixed Scripts library action buttons (edit, download, delete) being unreachable via keyboard navigation, and added accessible labels so screen readers can distinguish them.
- Fixed corrupted vulnerabilities download removing existing detections.
- Fixed iOS and iPadOS logos on the OS list in dark theme.
- Fixed a bug where deleting one of multiple duplicate DEP hosts did not resolve the duplicate. Fleet no longer recreates a pending host record when another host with the same serial and platform still exists.
- Fixed an issue where updating the device mapping for a host with no user, or a non-existent IdP user, would not resend config profiles using IdP variables.
- Fixed a bug where the carve cleanup cron job called the MySQL implementation instead of the S3-aware implementation on S3-configured deployments, meaning expired carves were never marked as expired in S3. Also fixed a panic in S3 carve cleanup that occurred when there were no non-expired carves.
- Fixed Android Enterprise page not refreshing after connecting or disconnecting Android MDM, so the Enterprise ID and card state are visible without a manual page reload.
- Fixed `List certificate templates` API docs: query parameter was incorrectly documented as `fleet` instead of `fleet_id`, causing the parameter to be silently ignored and returning no results.
- Fixed a bug where Android device check-ins could silently revert admin team transfers.
- Fixed `GET /api/v1/fleet/vulnerabilities` returning raw SQL errors when using cursor pagination (`after`) with `order_key` set to `cve`, `hosts_count`, or `cve_published`.
- Fixed a bug where patch policies with software install automations used an inactive, older installer and not the latest.
- Fixed "Show example payload" button being incorrectly disabled in GitOps mode on the "Other workflows" and "Calendar events" policy automation modals.
- Fixed stale pending MDM profiles reappearing after globally toggling Apple or Windows MDM off and back on.
- Fixed the live policy page not using the full page width like the live query page does.
- Fixed a bug where in GitOps, if a patch policy was specified with a different FMA slug for the install software automation, it would be used for the query instead of the slug for the patch policy itself.
- Fixed false positive vulnerability CVE-2017-17522 reported for Python (this CVE is disputed and not exploitable).
- Fixed false positive vulnerability CVE-2023-36632 reported for Python (this CVE is disputed; the reported behavior is intentional).
- Fixed false positive vulnerability CVE-2024-3219 reported for Python on macOS and Linux hosts (this CVE only affects Windows).
- Fixed the `GET /api/v1/fleet/hosts` endpoint so that filtering Android hosts by `os_name=Android` and `os_version=<version>` returns the matching hosts. Android hosts now populate the `operating_systems` table on enrollment and on every status report, and also appear in the `GET /api/v1/fleet/os_versions` aggregation and OS list in the UI with the Android logo.
- Fixed "User email" in device_mapping being unset in GET /api/v1/fleet/hosts for Windows and Linux hosts enrolling with end-user authentication.
- Fixed `GET /api/v1/fleet/software/versions` returning HTTP 422 "too many placeholders" when called without a `per_page` parameter on instances with large software inventories.
- Fixed host software list surfacing stale installer metadata after a Fleet-maintained app was replaced, which caused label scope to be evaluated against the previous installer and disagree with the install endpoint.
- Fixed the "host is offline" banner on the My device page incorrectly appearing during the first few minutes after an enrollment.
- Fixed software title icon not-found errors (and other 4xx errors) being reported as server-side exceptions in OTEL traces, APM, Sentry, and the Redis-backed debug errors endpoint.
- Fixed the host's Software UI showing a date decades in the past (e.g. "over 46 years ago") instead of "Never" for apps reporting a sentinel `last_opened_time` such as `315532800` (1980-01-01 UTC) that were never opened. Added a migration to clear these sentinel values from previously ingested software.
- Fixed latency issues with /vulnerabilities and filtered /software/versions queries.
- Fixed `fleetctl gitops` to refuse to apply SSO / EUA config that is missing required fields, if SSO is enabled globally or EUA is enabled on any team.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.87.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-06-19">
<meta name="articleTitle" value="Fleet 4.87.0 | 800+ new apps, custom OS updates, Android commands, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.87.0-1600x900@2x.png">
