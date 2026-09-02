# Fleet 4.91.0 | Windows local admin accounts, Autopilot default fleets, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/db7vcJEueuY?si=zhxBZVJwtYURAmXw" title="0" allowfullscreen></iframe>
</div>

Fleet 4.91.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.91.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

Three of this release's highlights close gaps in Windows device management: a local admin account for troubleshooting, automatic fleet assignment for Autopilot-enrolled hosts, and clearer software inventory for Windows Store apps. Read on for those, plus updates to webhooks, patch policies, scripts, DDM, Apple Business Manager, and OS updates.

## Highlights

- [Windows: create a local admin account](#windows-create-a-local-admin-account)
- [Default fleet for Windows Autopilot-enrolled hosts](#default-fleet-for-windows-autopilot-enrolled-hosts)
- [Webhooks for host activities](#webhooks-for-host-activities)
- [Patch policies: install when the app is closed](#patch-policies-install-when-the-app-is-closed)
- [Built-in variables in scripts](#built-in-variables-in-scripts)
- [Release a host from Apple Business](#release-a-host-from-apple-business)
- [Apple hardware marketing names](#apple-hardware-marketing-names)
- [OS updates: update to latest after a deadline](#os-updates-update-to-latest-after-a-deadline)

### Windows: create a local admin account

_Available in Fleet Premium_

IT admins can now create managed local admin accounts on Windows hosts. Turn on **Create hidden admin** for Windows under **Controls > Setup experience > Users**, or set `windows_settings.enable_managed_local_account` in GitOps. Fleet creates the account, hides it from the sign-in screen, and generates a unique, escrowed password for each host. All roles in Fleet can retrieve the password from **Host details > Actions > Show managed account** when IT needs a break-glass account for troubleshooting.

On macOS, managed accounts are created only at enrollment time, and only for hosts that automatically enroll via Apple Business Manager. On Windows, they're created on every host with MDM turned on, including hosts that enrolled earlier.

GitHub issue: [#43488](https://github.com/fleetdm/fleet/issues/43488)

### Default fleet for Windows Autopilot-enrolled hosts

IT admins can now choose a default fleet for hosts that enroll through [Windows Autopilot](https://fleetdm.com/guides/windows-mdm-setup#windows-autopilot). New Autopilot-enrolled Windows hosts land in that fleet automatically, instead of sitting in "Unassigned" until an admin transfers them by hand. Only global admins can set the default, and it's available in GitOps as `mdm.windows_automatic_enrollment.default_fleet`.

GitHub issue: [#41787](https://github.com/fleetdm/fleet/issues/41787)

### Webhooks for host activities

_Available in Fleet Premium_

IT Admins and Security Engineers can now send a webhook for every activity tied to a specific host, like a script run, software install, or configuration profile install. Configure a webhook URL per fleet from that fleet's **Hosts** page, or with [`webhook_settings.host_activities_webhook` in GitOps](https://fleetdm.com/docs/configuration/yaml-files#host-activities-webhook), to feed a [SIEM](https://www.gartner.com/reviews/market/security-information-event-management) or trigger automations in a third-party automation tool (e.g. [Tines](https://www.tines.com/))

GitHub issue: [#40493](https://github.com/fleetdm/fleet/issues/40493)

### Patch policies: install when the app is closed

_Available in Fleet Premium_

Patch policies for [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) can now wait until the app is closed before installing an update, instead of forcing the update while an end user is mid-task. Choose **Patch only when app is closed** when adding or editing a Fleet-maintained app, and Fleet skips the install (logged as "Install skipped") until the app quits, then installs it on the next automation run.

GitHub issue: [#39962](https://github.com/fleetdm/fleet/issues/39962)

### Built-in variables in scripts

_Available in Fleet Premium_

Fleet's [built-in variables](https://fleetdm.com/guides/fleet-variables), like `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`, now work in shell, PowerShell, and Python script content on macOS, Windows, Linux hosts, not just configuration profiles. IT Admins can write one script that inserts a host's serial number, IdP username, or other host-specific values, instead of hardcoding a value per host. Fleet validates variable references when a script is uploaded, so a script referencing a variable that doesn't exist is rejected before it ever runs.

GitHub issue: [#46837](https://github.com/fleetdm/fleet/issues/46837)

### Release a host from Apple Business

_Available in Fleet Premium_

IT admins can now release a host from Apple Business (ABM) directly from the Fleet UI or API, without signing into the ABM portal. Use the **Release from Apple Business** action on an eligible host's details page, or release multiple hosts at once via the API. Only Fleet users with [admin role](https://fleetdm.com/guides/role-based-access) can release a host.

GitHub issue: [#47633](https://github.com/fleetdm/fleet/issues/47633)

### Apple hardware marketing names

Fleet now shows the Apple hardware marketing name, like "MacBook Pro 16-inch, M3, 2023," on the **Hosts**, **Host details**, and **Fleet Desktop > My device** pages, instead of the raw hardware model identifier (for example, "Mac15,9"). This makes it easier to tell which hardware a host is running without looking up the model identifier yourself.

GitHub issue: [#46818](https://github.com/fleetdm/fleet/issues/46818)

### OS updates: update to latest after a deadline

_Available in Fleet Premium_

IT Admins managing macOS, iOS, and iPadOS OS updates can now set enforcement to always target the latest OS version, with a deadline measured in days after each release, instead of pinning to a specific version and a fixed calendar date. As Apple ships new OS versions, Fleet updates the enforced target automatically, so admins don't have to bump the minimum version by hand every time Apple releases an update.

GitHub issue: [#39085](https://github.com/fleetdm/fleet/issues/39085)

## Changes

### IT Admins
- Added the ability to create a managed local admin account on Windows hosts. Requires `fleetd` 1.60.0 or higher.
- Added a default fleet for new Windows MDM enrollments (Fleet Premium). IT admins can pick the fleet that hosts enrolling through user-driven Windows MDM enrollment (Windows Autopilot, Entra join) are automatically assigned to. The fleet is assigned before the Autopilot Enrollment Status Page runs, so the default fleet's software, scripts, and configuration profiles apply during out-of-box setup.
- Added the option to keep macOS, iOS, and iPadOS hosts on the latest OS version (Fleet Premium). Setting `minimum_version` to `latest` along with `deadline_days` tells Fleet to automatically track the newest version Apple publishes for each host's hardware and to set the update deadline that many days after the version has been released.
- Added activity automations for fleets (Fleet Premium): a per-fleet webhook, configured from the Hosts page, that sends a request to a destination URL whenever an activity linked to one of the fleet's hosts is created.
- Added a "Patch when closed" option for patch policies that only patches an app on a host if the app is not running.
- Added support for Fleet's built-in variables (e.g. `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`) in scripts, including software install, post-install, and uninstall scripts. Variables are resolved per host at execution time and require a Fleet Premium license.
- Added Adobe plugins to software inventory: Fleet now detects Adobe Creative Cloud plugins (CEP and UXP extensions) on macOS and Windows hosts and lists them on the Software page and host details with the software type "Plugin (Adobe)", including version and host count.
- Added 29 new iOS/iPadOS device vitals, such as battery level, accessibility settings, cellular technology, cloud backup status, organization info, MDM options, device attestation, and cellular service subscriptions, collected via the existing `DeviceInformation` MDM refetch and shown in the host API response and a new "View all" vitals modal on the host details page. Personal (BYOD) enrollments don't receive these new fields, to avoid exposing information about a device the organization doesn't own.
- Added marketing name display for Apple devices (macOS, iOS, iPadOS) on the Hosts and Host details pages. The "Hardware model" field now shows human-readable names (e.g. "MacBook Pro (16-inch, 2021)") instead of raw identifiers (e.g. "MacBookPro18,1").
- Added support for releasing devices from Apple Business inside Fleet.
- Added the `s3_software_installers_signed_url` configuration option to serve software installer, in-house app, and bootstrap package downloads via GCS presigned URLs (the GCS counterpart to CloudFront URL signing), so clients download directly from object storage instead of streaming through the Fleet server.
- Added support for custom host vitals (`$FLEET_HOST_VITAL_<id>`) in Android configuration profiles and managed app configuration, including per-host value expansion at delivery and automatic resend when a host's value changes.
- Added support for `$FLEET_HOST_VITAL_<id>` custom host vital variables in host name templates, including per-host resolution, validation of referenced vital IDs, and automatic re-delivery when a host's vital value changes.
- Added support for nested groups in Entra in IdP vitals.
- Added `linux` as a label platform option, which targets hosts on any Linux distribution.
- Added a sortable "Added to Fleet" column to the hosts table, showing when each host last enrolled with Fleet.
- Added support for enabling/disabling software inventory per-fleet via `PATCH /api/v1/fleet/fleets/{id}` with `{"features": {"enable_software_inventory": <bool>}}`. The key follows PATCH-merge semantics: when omitted, the stored value is unchanged.
- Added a `--bypass-end-user-auth` flag to `fleetctl package` that configures the generated fleetd installer to skip the end-user authentication prompt during enrollment on Linux and Windows hosts (e.g. when the end user already authenticated via another MDM). Requires fleetd v1.60.0 or higher.
- Added `host_id` and `host_serial` to the `mdm_enrolled` activity for Apple (macOS, iOS, iPadOS) enrollments, and the activity now appears on the host's activity timeline.
- Added `token_invalid` to the ABM token API responses and `dep_device_error` (a human-readable message) to `GET /hosts/:id/dep_assignment`, to help identify why a host's Apple Business Manager device lookup or ABM token isn't returning expected data (e.g. a rejected or invalid-signature token, unsigned terms, a server-side error, or the device no longer being assigned to Fleet).

### Security Engineers
- Changed Orbit enrollment to determine end user authentication requirements from server policy rather than client-advertised capabilities. The `mdm.allow_orbit_end_user_auth_bypass` server setting (enabled by default) controls whether hosts that do not complete end user authentication may enroll into a fleet that requires it; set it to `false` to strictly enforce end user authentication for all Orbit enrollments.
- Added a `user_mfa_requested` activity, recorded when valid credentials are submitted for an MFA-enabled account and a verification email is sent.
- Added `created_setup_experience_script` and `deleted_setup_experience_script` activities so that adding, replacing, or removing a setup experience script (via the API or GitOps) is recorded in the audit log.
- Updated the macOS CIS benchmark policies to the latest CIS releases: macOS 14 Sonoma v3.1.0, macOS 15 Sequoia v2.1.0, and macOS 26 Tahoe v1.1.0.
- Excluded Adobe plugins from vulnerability scanning, so no vulnerabilities are reported for them. No vulnerability data source maps an Adobe CEP or UXP extension to a CVE; Adobe files CVEs against the host application (Photoshop, Acrobat, and so on), which Fleet already scans.

### Bug fixes and improvements
- Updated configuration profiles scoped by dynamic (query-based) labels to preserve a host's current profile state while the host's membership in a label is still unknown. Profile changes only happen after the label has been evaluated at the host's next refetch, so adding a new exclude-any (or include-all) label to a profile does not immediately remove it from hosts that have not run the label's query yet.
- Improved the performance of the configuration profiles status summary (`GET /api/latest/fleet/configuration_profiles/summary`) for Windows hosts so it no longer times out on large fleets.
- Reduced database load when software installed-path updates hit lock contention: the transaction now fails fast instead of retrying, and the host's next software report reconciles the paths.
- Improved software ingestion performance at scale by batching `host_software_installed_paths` deletes (previously unbounded single statements).
- Blocked host enrollment with empty or whitespace-only enroll secrets across all enrollment paths (osquery, Orbit, Apple MDM, Android), and removed any pre-existing empty enroll secrets.
- Updated Windows SCEP profiles to fail with a clear message when the certificate authority challenge contains characters Windows doesn't support (ASN.1 PrintableString), instead of showing "Verified" while no certificate is installed.
- Improved input validation for the Windows MDM enrollment flow.
- Normalized LocURI values before validation in Windows profile handling.
- Improved LocURI validation in Windows profile handling to canonicalize element content before checking.
- Updated the macOS disk encryption banner on the Host details and My device pages to tell IT admins and end users that ADE-enrolled hosts escrow their FileVault key automatically on the next refetch, instead of asking the end user to log out.
- Added a clear error message when adding both Mozilla Firefox and Firefox ESR (which share a bundle identifier) to the same fleet, so admins understand only one of them can be added instead of seeing a generic conflict error.
- Added deduplication and out-of-order protection to the Android MDM Pub/Sub notification handler. Duplicate deliveries from Google Pub/Sub no longer re-run the setup experience or emit duplicate activities, and a stale device-deleted notification arriving after a re-enrollment no longer leaves the host stuck showing unenrolled.
- Bounded the Android device reconciliation cron's Google API pagination so a malformed or cycling response can no longer cause an unbounded loop, and added periodic progress logging during pagination.
- Bounded how much of a Google Workspace directory one IdP sync pass pulls, so a very large directory or a misbehaving response can no longer paginate indefinitely or grow server memory without limit. A sync that exceeds a limit fails with an error naming the limit (visible as the last IdP sync status) instead of ingesting a partial directory.
- Clarified the failed-install modal copy on Android hosts to explain that the end user can retry via the Google Play Store in their work profile.
- Added a disabled Refetch button with a tooltip on the Host details page for Android hosts, explaining that Android hosts sync data automatically and linking to how to sync manually.
- Removed the misleading Self-service preview tab from the "Edit appearance" modal for Android apps. Android apps are installed from the Play Store rather than the Fleet self-service web view, and updating the appearance will not change anything in the Play Store.
- Disabled the Android MDM "Connect" and "Turn off Android MDM" buttons with a GitOps tooltip when GitOps mode is enabled, matching the Windows MDM behavior.
- Added software upload progress logging to fleetctl GitOps.
- Added a diagnostic message when an install script can't be run (exit code -1), such as when the interpreter in its shebang is missing on the host, instead of reporting no output.
- Allowed `.py` script-only packages to be assigned a `setup_experience_platform` (`darwin` or `linux`), matching `.sh`.
- Added a software package maximum size error message in the UI to fix inconsistent errors across different browsers.
- Normalized login responses for accounts with MFA enabled to follow authentication best practices.
- Made MFA login token redemption atomic so a single one-time token can no longer be used to create more than one session under concurrent requests.
- Ensured a password reset token can only be used once.
- Added 255-character caps to additional user-supplied name and description inputs (API user, custom variable, certificate, label, pack, certificate authority forms) to prevent the same class of overflow bug.
- Reworked the fleets dropdown to make the search input discoverable at 10+ fleets and added an "Add fleet" affordance for global admins.
- Restricted deleting a fleet to global write permissions (global admin or GitOps), matching the existing restriction on creating one.
- Restricted the manual MDM enrollment profile endpoint (`GET /api/v1/fleet/enrollment_profiles/manual`) to global and fleet-scoped admins and maintainers.
- Clarified the Controls > OS updates empty state to say "Apple or Windows MDM must be turned on" and link to MDM settings, so it no longer implies MDM is off when only Android MDM is enabled.
- Improved the Apple Business success toast shown after editing fleet assignments to name the organization.
- Renamed the "Program (Windows)" software type to "Application (Windows)", which includes apps installed through the Windows app store.
- Added tooltips to the "Agent", "Last restarted", and "Status" column headers on the Hosts page explaining which platforms are supported and why.
- Updated the "Last opened" tooltip on the Host details Software table to explain why it's only supported for native macOS, Windows, and Linux apps and packages.
- Removed the `cellProps.rows.length === 1` workaround (which suppressed the tooltip whenever the table had exactly one row) by adding the correct CSS, fixing the tooltip overflowing when the host table has only one row.
- Ranked "Select API endpoints" search results by relevance (exact match, then prefix, then whole-word, then substring, matched against both name and path) instead of leaving them in an unranked catalog order, and fixed a bug where the table's default sort silently discarded that ranking.
- Removed pagination from the "Select API endpoints" search results table (Settings > Users > Add API-only user > Specific API endpoints), relying on the results dropdown's existing scrollbar instead.
- Improved empty state copy on the software title detail page when the software is not found in the selected fleet.
- Made the per-platform entries in the dashboard "Hosts enrolled" chart keyboard accessible: each platform with hosts is now focusable via Tab, activatable with Enter/Space, has a visible focus indicator, and exposes an accessible name (e.g. "macOS hosts").
- Enforced API-only endpoint restrictions on chart endpoints.
- Updated button styles across the Fleet UI to use the new bordered secondary and subdued button variants.
- Aligned the configuration profiles empty state with the assets tab styling so admin and maintainer users see a consistent empty state.
- Aligned the configuration profiles and assets empty states for technicians with the shared EmptyState styling.
- Added a `FLEET_DEV_SKIP_S3_CONFIG` environment variable to skip applying local S3 dev defaults and creating test S3 buckets when running `fleet serve --dev`.
- Added `--since-commit` and `--commit` flags to the `migration-cleanup` tool, allowing migration rename scans scoped to a commit range on `main` or a single commit, in addition to the existing `--branch` mode.
- Split `ListHostSoftware` and `ModifyAppConfig` into smaller helpers so that those packages can be checked by the nilaway linter.
- Removed an unneeded dependency that does not support Apple M chips.
- Updated Go to 1.26.7.
- Fixed an App Store or in-house app install that a device acknowledged but never verified leaving the host unable to run anything else. Fleet now fails such an install after 24 hours and releases the host's activity queue, so scripts, software installs, and uninstalls waiting behind it run. The wait is configurable with `FLEET_SERVER_VPP_INSTALL_REAP_TIMEOUT`. An install whose command has not reached the device yet is left alone until it can no longer be delivered.
- Fixed Fleet holding on to an App Store app verification command after it had nothing left to verify on that host, which delayed verifying the next install on the same host by up to a day.
- Fixed software (packages, App Store apps, and in-house apps) targeted with "exclude any" on a host vitals label being hidden from, and blocked for, every host instead of only the label's members.
- Fixed duplicate software inventory entries (same name, version, and source with the same vulnerabilities but different host counts) that could appear on instances upgraded to Fleet v4.76.0 or later. Existing duplicate giflib (Homebrew) entries are merged during the database migration on upgrade.
- Fixed a false-negative vulnerability report where Firefox Developer Edition on macOS was not matched to any CVEs because Fleet generated no CPE for it.
- Fixed the SCEP proxy so that Windows profiles using a custom SCEP proxy certificate authority have their one-time Fleet challenge validated before a PKIOperation request is forwarded to the certificate authority. Requests with a missing, incorrect, or expired challenge are now rejected.
- Fixed the SCEP proxy so that a Windows profile pending removal can no longer be used to relay SCEP requests to the configured certificate authority, and so that proxy error responses no longer include the certificate authority's URL.
- Fixed a few edge cases for Apple profile reconciliation when devices respond with NotNow in certain scenarios.
- Fixed an issue where re-enrolling an Apple device with a different type, e.g. Manual -> ADE, would not update the enrollment type correctly.
- Fixed a bug where an Apple configuration profile (DDM declaration) could become undeletable if the set of allowed declaration types changed after the profile was added (for example, when a server configuration flag was toggled). Deleting a profile no longer re-runs upload-time validation.
- Fixed Fleet storing an unusable disk encryption key and logging an escrow activity for macOS hosts that aren't enrolled in Fleet's MDM.
- Fixed team-level BitLocker PIN enforcement never reaching Windows hosts when Apple MDM was not configured on the server: the "Create PIN" banner appeared on the My device page, but the queries and MDM command that enable PIN setup were never sent.
- Fixed Windows software whose inventory name includes the version (e.g. "Granola 7.373.2") not matching the Fleet-maintained app's software title, which prevented the uninstall action from appearing and listed each version as its own software title. Applies to Fleet-maintained apps that have been added as an installer. Existing mismatched titles are merged when the app is added, shortly after server startup, and hourly thereafter.
- Fixed a bug where a failed macOS Fleet-maintained app install could be reported as successfully installed because the install script did not check the exit code of the command that installed the app. This covers generated install scripts, the custom install scripts used by some apps (including Google Chrome, Zoom, Microsoft Edge, and Webex), and the already-published scripts of frozen apps.
- Fixed the Docker Desktop macOS Fleet-maintained app not reporting an installed version or "Installed" status on hosts that already have Docker Desktop. The app matched on the embedded Electron bundle identifier (`com.electron.dockerdesktop`) instead of the identifier the installed app reports (`com.docker.docker`), and embedded bundles are excluded from software inventory. Existing Docker Desktop installers are re-pointed to the correct software title on upgrade.
- Fixed a bug where the patch policy for the Windows Git Fleet-maintained app always returned "Pass", even on hosts running an outdated version, so update automations never triggered. The generated query matched `programs.name LIKE 'Git %'`, but Git for Windows registers itself in the registry as exactly `Git`.
- Fixed the macOS "Steam up to date" patch policy always failing on hosts that have the current version of Steam installed. Steam.app ships without a `CFBundleShortVersionString`, so the generated policy now compares `CFBundleVersion`, which agrees with the version Fleet reports in software inventory.
- Fixed Fleet no longer recognizing hosts running Omarchy as Linux. Omarchy 4 ships its own `/etc/os-release` and reports `platform=omarchy`, where earlier versions inherited `arch` from Arch Linux and were covered by Fleet's Arch support. Host vitals and software inventory populate again, disk encryption and key escrow are available, policies and labels scoped to `linux` apply, and scripts can be run from the host's Actions menu. Omarchy hosts continue to roll up onto the "Arch Linux" / "rolling" row in Software > OS.
- Fixed Android hosts staying stuck on a pending Lock, Wipe, or Clear passcode when Google never delivered the command's result to Fleet. Fleet now checks the command's outcome directly with Google once a day and updates the host, so the command can be re-issued.
- Fixed the Hosts page so that managed Android hosts display their serial number instead of "Not supported" when one is reported.
- Fixed an issue where GitOps could apply MDM SSO configuration values that were invalid.
- Fixed an issue where `generate-gitops` would export an empty `apple_business` section if the AB default fleets had only been set via the UI.
- Fixed the Google Calendar integration scheduling maintenance events over users' Focus Time and Out of office blocks.
- Fixed the SCIM last request telemetry so that authorization failures (403 Forbidden) from unauthorized users are no longer persisted, preventing them from overwriting the admin-visible SCIM status.
- Fixed an edge case where deactivating a SCIM user did not deprovision the matching Fleet user if the user's identifiers were changed in the same request.
- Fixed the Helm chart's Deployment and `fleet-vulnprocessing` CronJob templates so empty/unset entries in `environments` (e.g. the default `FLEET_SERVER_PRIVATE_KEY: ""`) are omitted from the rendered container env list instead of being emitted as an empty-string env var, which previously collided with the same key supplied via `envsFrom` and caused server-side apply to reject the object with a "duplicate entries for key" error.
- Fixed password reset so that case-mutated reset tokens are no longer accepted; tokens are now matched case-sensitively.
- Fixed an issue where an SSO-only invitation could be accepted with a password, creating a local password-authenticated account and bypassing SSO enforcement. The authentication mode is now derived solely from the invite.
- Fixed the hosts list endpoint sometimes returning software title details that didn't match the applied filter.
- Fixed the software title details response so that installer script contents and managed app configuration are only returned to users authorized to read the installer, and so that a request without a fleet is authorized against "No team" instead of skipping the scope check. This applies to `GET /api/v1/fleet/software/titles/{id}` and to the `software_title` included in `GET /api/v1/fleet/hosts` when filtering by `software_title_id`.
- Fixed a software title request without a fleet so that it only resolves titles in fleets the requester can see. Previously a title reachable only through a software package, App Store app, or in-house app in another fleet was returned, which told the requester that software they have no access to exists.
- Fixed device-authenticated ("My device") software uninstall so that it applies the same self-service and label-scope rules as the self-service install path, instead of accepting any package on the host's fleet.
- Fixed the OS versions API (`GET /api/latest/fleet/os_versions`) to return a validation error for an unsupported `platform` filter and a "not found" error for an unknown OS version ID, instead of a successful but empty or null-filled response. Also corrected the `max_vulnerabilities` validation message so the `>=` character is no longer returned HTML-escaped.
- Fixed the delete host endpoint returning inconsistent responses for a host outside the requester's fleet versus one that doesn't exist.
- Fixed the host transfer activity (`transferred_hosts`) to only record host IDs that actually exist, so non-existent host IDs passed to `POST /api/latest/fleet/hosts/transfer` can no longer be injected into the audit trail. No activity is created when none of the requested hosts exist.
- Fixed the policy automations table showing only one host for automation runs that cover multiple hosts.
- Fixed query (report) responses so that pack metadata (ID, name, description) is only included when the requesting user is authorized to read packs, preventing cross-fleet pack metadata disclosure via query name collisions.
- Fixed the live query results websocket stream returning a different error for a nonexistent campaign versus one owned by another user.
- Fixed the self-service "Install all" button so its count and install target scope to the current search query. Previously, typing a search would filter the visible list while "Install all" still counted (and queued) every item in the selected category, including software the search had filtered out.
- Fixed device-authenticated ("My device") endpoints so that host policies are returned in a device-safe representation that no longer exposes the policy author's name and email or the policy's raw SQL query.
- Fixed a bug where a `.sh` or `.py` software package with Windows-style (CRLF) line endings was accepted at upload but failed to install on Linux hosts with a "No such file or directory" error. Line endings are now normalized to Unix-style before the script is stored.
- Fixed a potential resource exhaustion issue in the MSI metadata parser.
- Fixed a race condition that allowed a one-time software installer download token to be redeemed more than once when many requests raced concurrently, by making the token consumption atomic.
- Fixed the "Add software" error for a file whose contents don't match a supported installer format: it no longer says "Couldn't edit software" on an add and no longer implies the file extension is the problem.
- Fixed software installer validation errors reporting the wrong action verb (add vs. edit).
- Fixed the install rejection message for `.sh`/`.py` script packages to say they can be installed on macOS and Linux hosts, rather than Linux only.
- Fixed long fleet names overflowing the fleets table, fleet detail page header, teams dropdown, and manage enroll secrets modal. Fleet name inputs now cap at 255 characters (matching the database column), and the API and GitOps now return a clear validation error instead of a raw "Data too long" MySQL error when a longer name is submitted.
- Fixed long label names overflowing the Labels card on the host details page.
- Fixed a bug where modifying a label could silently persist a membership change without recording an audit activity when the label's metadata update failed (e.g. renaming to a name that already exists). Label metadata and membership are now saved in a single transaction, so a failed update rolls back both.
- Fixed the Setup experience Users settings to default the account type to "Admin" for fleets whose config predates the setting, instead of showing no selection.
- Fixed sort direction on the "Last seen" and "Last fetched" columns of the hosts table so that sort descending puts the oldest date (biggest "days ago" number) first, matching the visible duration rather than the underlying timestamp.
- Fixed the "Last restarted" vital showing on ChromeOS hosts, where it's not actually collected.
- Fixed the status-filter dropdown's selected-value icon (e.g. the disk encryption, bootstrap package, and policy status filters on the Hosts page) rendering near-black and barely visible in dark mode.
- Fixed UI to show a direct button instead of a single-item dropdown on the Labels page (for users without edit/delete permissions) and the Integrations page (Jira/Zendesk).
- Fixed styling issues with the gap between icons and text across the product.
- Fixed the page content shifting left when opening a role dropdown near the bottom of the New user form (`/settings/users/new/human`). Dropdowns now flip upward when there's no room below the trigger.
- Fixed the page content shifting left when opening the Actions dropdown on the last rows of a table (labels, users, fleets), and the resulting jump when a delete modal opened. The dropdown menu now flips upward when there's no room below the trigger.
- Fixed an accessibility issue where the resend button would not show up when focused inside the OS settings modal.
- Fixed tooltips wrapping icons or numeric values (like the Issues count on the host details page) showing a text (I-beam) cursor on hover, which made them look editable; those tooltips now use the default arrow cursor. Underlined-text tooltips are unchanged.
- Fixed the My device > Self-service search bar not sitting flush right when the "Install all" button isn't rendered.
- Fixed an empty summary card rendering above the Vitals section on the host details page for Fleet Free hosts with no summary content (Android, and iOS/iPadOS with no OS settings).
- Fixed data table column headers stretching vertically when the table has no rows (e.g. while refetching from a zero-result search on the My device software page).
- Fixed the misaligned icon in `notify.success`/`notify.error` toast notifications so it sits on the first line of the message on both single- and multi-line toasts.
- Fixed low color contrast on the "Inherited" tag and unified the styling of tags (e.g. "Inherited," "API," "Patch," host filter chips, and host label pills) across the UI to match the design system.
- Fixed helper text under checkboxes and radio buttons so it aligns with the label instead of the control.
- Fixed misaligned app icons on the macOS setup experience "Setting up your device" screen.
- Fixed an issue where the user-scoped icon wasn't showing for iOS and iPadOS hosts.
- Fixed an issue where Fleet would show a turn on MDM banner before knowing the device state.
- Fixed the script and query editors so that scrolling after a single click no longer selects text instead of scrolling.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.91.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-09-02">
<meta name="articleTitle" value="Fleet 4.91.0 | Windows local admin accounts, Autopilot default fleets, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.91.0-1600x900@2x.png">
