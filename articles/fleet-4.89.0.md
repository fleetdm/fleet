# Fleet 4.89.0 | Windows setup experience improvements, Android variables everywhere, and more...

Fleet 4.89.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.89.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Windows setup experience: continue past a failed install](#windows-setup-experience-continue-past-a-failed-install)
- [Android: host vital variables everywhere](#android-host-vital-variables-everywhere)
- [Default fleet for BYOD iOS/iPadOS enrollment](#default-fleet-for-byod-ios-ipados-enrollment)
- [Auto-update, pin, and roll back Fleet-maintained apps](#auto-update-pin-and-roll-back-fleet-maintained-apps)
- [Filter and save the vulnerability exposure chart](#filter-and-save-the-vulnerability-exposure-chart)
- [Policy status page](#policy-status-page)
- [Script-only packages: pre-install query, post-install, and uninstall scripts](#script-only-packages-pre-install-query-post-install-and-uninstall-scripts)
- [IdP host vitals from Google Workspace](#idp-host-vitals-from-google-workspace)

### Windows setup experience: continue past a failed install

_Available in Fleet Premium_

When required setup software fails to install during [Windows automatic enrollment](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment) (Autopilot or non-Autopilot), end users now see exactly which software failed. If the IT admin hasn't checked **Cancel setup if software fails**, the end user can continue past the failure and install the missing software later from self-service. If that option is checked, setup stops and the end user is told to reset the device and try again. Either way, end users get a clear next step instead of a stuck setup screen, which means fewer support tickets for IT admins.

GitHub issue: [#45948](https://github.com/fleetdm/fleet/issues/45948)

### Android: host vital variables everywhere

IT admins can now use any host vital variable (`$FLEET_VAR_HOST_`), like a host's UUID or the end user's IdP email, in Android configuration profiles, certificate templates, and managed app configuration. This makes it possible to deploy a host-specific value as part of an app's configuration, for example, passing a host's UUID to Duo as a trusted endpoint identifier, or a user's email as the identity for [EAP-TLS Wi-Fi authentication](https://fleetdm.com/guides/configure-eap-tls-wifi-android). For certificates, Fleet also detects when a host vital variable's value changes and automatically resends the certificate so it stays accurate. See all host vital variables in the [built-in variables guide](https://fleetdm.com/guides/fleet-variables).

GitHub issues: [#45353](https://github.com/fleetdm/fleet/issues/45353), [#41968](https://github.com/fleetdm/fleet/issues/41968), [#37406](https://github.com/fleetdm/fleet/issues/37406)

### Default fleet for BYOD iOS/iPadOS enrollment

IT admins can now choose a default fleet for iOS and iPadOS hosts that [enroll via Account-driven User Enrollment (BYOD)](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account). This means personal iPhones and iPads automatically land in the right fleet on enrollment, so they get the correct configuration profiles and software without an admin having to move them manually.

GitHub issue: [#30871](https://github.com/fleetdm/fleet/issues/30871)

### Auto-update, pin, and roll back Fleet-maintained apps

_Available in Fleet Premium_

IT admins can now control exactly which version of a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) their hosts run. Pin a Fleet-maintained app to a specific version to stop it from auto-updating, or roll back to the previous version if a new release causes problems, all from the software title's page. If you're relying on auto-update, Fleet checks for new versions hourly, so hosts stay current without an IT admin re-adding the app.

GitHub issue: [#38504](https://github.com/fleetdm/fleet/issues/38504)

### Filter and save the vulnerability exposure chart

_Available in Fleet Premium_

Security Engineers can now filter the [vulnerability exposure chart](https://fleetdm.com/guides/dashboard-vulnerability-exposure) by software category (operating system, browsers, Microsoft Office, or Adobe apps), EPSS exploit probability, known active exploits (CISA KEV), and specific CVEs to exclude, so the chart reflects the risk registry they actually track instead of every vulnerability Fleet detects. These default filters can now be set and persisted via [GitOps (YAML)](https://fleetdm.com/docs/configuration/yaml-files#features), so they load automatically the next time the chart opens. Filters changed directly in the Fleet UI aren't saved, whether GitOps mode is on or off.

GitHub issues: [#44746](https://github.com/fleetdm/fleet/issues/44746), [#47327](https://github.com/fleetdm/fleet/issues/47327)

### Policy status page

IT admins get a historical view of [policy automation](https://fleetdm.com/guides/automations#policy-automations) runs: pass/fail status for every host, alongside the output of the software install or script run that the automation triggered. This makes it much faster to troubleshoot a host that keeps failing a policy, since admins no longer have to dig through separate activity logs to piece together what happened.

GitHub issue: [#38670](https://github.com/fleetdm/fleet/issues/38670)

### Script-only packages: pre-install query, post-install, and uninstall scripts

_Available in Fleet Premium_

IT admins can now add a pre-install query, a post-install script, and an uninstall script to [script-only software packages](https://fleetdm.com/guides/deploy-software-packages#script-only-packages), matching the behavior already available for custom packages. This means script-only packages can now offer an uninstall option and the same install verification other packages already have.

GitHub issue: [#42797](https://github.com/fleetdm/fleet/issues/42797)

### IdP host vitals from Google Workspace

_Available in Fleet Premium_

Fleet users who use Google Workspace (GW) as their identity provider (IdP) can now populate [IdP host vitals](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts) (group, department, username, email, and full name) directly from GW, without building a custom integration. Since Google Workspace doesn't support the [SCIM protocol](https://scim.cloud/), Fleet pulls directory data from Google's API on a schedule. Once connected, IT admins can scope configuration profiles, software, and policies using IdP host vital labels, the same way they would with an Okta or Entra SCIM integration.

GitHub issue: [#42915](https://github.com/fleetdm/fleet/issues/42915)

## Changes

### IT Admins
- Added the ability to target a policy to hosts using a combination of "include" and "exclude" labels.
- Added the ability to run a policy check before installing Windows and Linux setup experience software. When a team policy's install-software automation points at a setup experience installer, Fleet runs that policy during setup and skips the install when it passes (the software is already installed and up to date), speeding up the end user setup experience. When the policy fails, the software is installed as part of setup experience.
- Changed calendar remediation events to be scheduled on the next business day (skipping weekends) after a policy failure, instead of always being scheduled on the next Tuesday.
- Updated policy details page to show automations and labels as a single property. Also changed the layout of policy properties.
- Added automation runs table to the policy details page, showing per-host automation outcomes with filtering, search, and a reset policy action.
- Added per-host activity log entries when policy automations (webhook, tickets, Google Calendar, and Microsoft conditional access) fail or succeed.
- Added `POST /api/v1/fleet/policies/:policy_id/reset` endpoint to reset a policy's pass/fail results, clearing counts and membership immediately.
- Added `GET /api/v1/fleet/policies/:id/automation_activities` endpoint to list automation activities for a policy.
- Added the ability to keep Fleet-maintained apps automatically updated to the latest version, pin them to a specific version or major version, or roll back to a previously cached version, from the UI and via GitOps (Fleet Premium).
- Surfaced `.sh` script-only software packages on the macOS tab of Controls > Setup experience > Install software, with selections tracked independently from the Linux tab.
- Added `setup_experience_platform` on software packages in GitOps YAML so `.sh` script-only installers can be selected for the macOS setup experience declaratively, matching the per-platform UI selection. The value is authoritative on every batch apply and reconciles the cross-platform selection table.
- Added support for pre-install query, post-install script, and uninstall script on script-only packages (`.sh` and `.ps1`) via the UI, REST API, and GitOps.
- Added an error on the Windows enrollment status page (ESP) when setup experience software fails to install during automatic enrollment (Autopilot and other OOBE flows) and "Cancel setup if software fails" is turned off.
- Added "🛟 Support" as a new default self-service software category.
- Added support for `$FLEET_VAR_HOST_*` variables in Android configuration profiles.
- Added support for `$FLEET_VAR_HOST_*` variables in Android managed app configuration.
- Android certificate templates and managed app configurations are now automatically resent when IdP variable values change.
- Added support for defining the default fleet BYO Apple devices enroll into.
- Added a Google Workspace integration that maps identity provider (IdP) users to hosts, populating IdP host vitals directly from your Google Workspace directory.
- Added an activity feed entry when a user runs a custom Apple or Windows MDM command, visible in both the global activity feed and the host's activity feed.
- Added an activity when editing the managed local account setting using the update fleet endpoint or GitOps.
- Enabled tracking of mobile devices for the "hosts online" chart, and added default filtering to that chart that excludes mobile platforms.
- Added tooltips on the Settings > Users and My account pages to show assigned fleets and roles when a user has multiple.

### Security Engineers
- Started collecting non-critical CVEs, filtering them out of charts by default.
- Added the ability to filter vulnerable software by severity (CVSS score) and known exploit status on the Fleet Desktop **My device > Software** tab (Fleet Premium). The corresponding `min_cvss_score`, `max_cvss_score`, and `exploit` query parameters were added to the `GET /device/{token}/software` API endpoint.
- Added more filtering options for the Vulnerability Exposure chart.
- Added ability to set default Vulnerability Exposure chart filters via GitOps.
- Improved certificate renewal validation in the host identity SCEP service.
- Added support for all IdP variables and host platform in certificate template subject names and SANs.
- Improved input validation for conditional access SCEP enrollment.
- Validated that a custom SCEP proxy certificate authority challenge contains only printable characters, so Windows certificate enrollment no longer fails with "The string contains a non-printable character" (for example, when the challenge contains an underscore). Existing challenges are only re-validated when changed.
- Restricted authorization for team membership management operations.
- Made authorization more robust when creating labels from manual hosts.
- Improved fleet scope validation for software title lookups.
- Restricted authorization for conditional access Okta IdP asset endpoints so that observer and observer+ roles can no longer read them.
- Improved session handling during password reset flows.
- Cleared the SSO authentication cookie after successful authentication for fully-managed Android enrollment.
- Added private network IP blocking to Fleet's HTTP client. Loopback and cloud metadata addresses (127.0.0.0/8, 169.254.0.0/16) are always blocked. RFC 1918 and other private ranges are blocked by default; use `--allow_private_network_integrations` to allow them for environments with on-prem integrations (e.g. EJBCA, Jira, SCEP servers on private networks).
- Added the `s3.carves_cleanup_disabled` server setting to skip S3 file carve reconciliation for deployments that rely solely on the bucket's lifecycle policy to remove carve objects.
- Added the `s3.carves_cleanup_max_per_run` and `s3.carves_cleanup_concurrency` server settings to tune how many carves the S3 cleanup reconciles per run and how many concurrent S3 requests it makes.
- Updated the SigNoz OTEL dashboards under `tools/signoz/` to template and filter on the `deployment.environment` resource attribute, with the environment variable defaulting to `default`, so multiple Fleet environments reporting to the same SigNoz backend can be scoped per environment.

### Bug fixes and improvements
- Updated Go to 1.26.5.
- Updated checkbox labels in the Fleet UI to use positive language, making it clearer what each setting enables rather than what it disables.
- Improved Windows MDM configuration profile performance. Changes to Windows profiles now reach hosts more quickly. Large changes that affect many hosts at once, such as adding or removing profiles across a team or transferring many hosts between teams, now finish faster and put significantly less load on Fleet's database, keeping the server responsive at scale.
- Improved validation on batch script executions.
- Updated golang.org/x/image to v0.42.0 to resolve CVE-2026-33813 (WebP decoder denial of service on 32-bit platforms).
- Redesigned in-app success and error notifications as toasts. Error notifications now persist until dismissed and can be expanded to show the server's raw response.
- Added configurable batch size `FLEET_MDM_ANDROID_BATCH_SIZE` (default: 1000 hosts) for Android MDM operations to prevent overwhelming the Google Android Management API.
- Added batching and staggered scheduling for Android software installation jobs to spread AMAPI load across multiple worker ticks.
- Improved the error message shown when saving a custom variable without the required server private key configured.
- Improved software tooltips on the host details page to display the human-friendly software name and correct action labels for scripts.
- Improved orbit check-in performance by deriving the Fleet MDM connection state from existing host MDM data instead of running a separate 3-table JOIN query on every check-in for every host.
- Improved `fleetctl` to detect when SSO is enabled on the Fleet server and display a helpful message directing users to authenticate using an API token instead of email and password.
- Refactored `makeAndroidAppAvailable` to use staggered job queuing instead of sleeping between batches inside a single worker job.
- Updated the checkerboard graph to make it clearer which square represents the current time and which squares are in the future.
- Windows configuration profiles are now queued immediately when a host enrolls in Windows MDM, instead of waiting for the next profile reconciliation cron pass.
- Improved query validation logic around policy creation.
- Updated the "installed during setup" tooltip on Controls > Setup experience > Install software to clarify that installation order depends on software name (0-9, then A-Z), and that software without a policy is installed before software with a policy.
- Navigate back to the report details page after saving changes to a report.
- Enabled automatic refreshing of report results when the window is refocused and every 5 seconds while waiting for results to arrive (skipped when report caching is disabled).
- Reduced database write pressure on the Windows MDM check-in path by gzip-compressing stored device response envelopes.
- Updated the Fleet-maintained apps item count to reflect the total number of apps, counting an app's macOS and Windows versions separately (for example, a search for "Zoom" that returns Zoom and Zoom Rooms on both platforms shows 4 items).
- Moved and updated tooltip from the Vulnerabilities column on the **Software > OS** page to "Not supported", explaining which platforms support vulnerability detection.
- Improved some GitOps error messages around bootstrap packages, setup assistant and scripts.
- Fixed fleet-scoped context when retrieving a list of users in a fleet.
- Fixed an issue where cleanup of expired file carves stored in S3 could stall on buckets containing a large number of objects, which prevented other scheduled cleanup and aggregation tasks from running.
- Fixed the MDM command details modal showing a generic error, instead of a clear message, for a command sent to a host that was later wiped and re-enrolled.
- Fixed SAML SSO callback URLs (both login and MDM end user authentication) duplicating the subpath when Fleet is deployed under a URL prefix, which broke authentication. The callback URL is now built so the subpath appears exactly once whether or not the server URL was configured with the prefix.
- Fixed the **My device > Self-service** page briefly showing the "Update" button again on apps that had just finished updating, instead of holding the "Updated" state while the software inventory refreshes.
- Fixed a bug where selecting a policy on the host details or self-service policies page reset the list back to the first page.
- Fixed a 500 error when a host reported a software install result for a deleted software installer. When an installer is deleted, records of its pending installations will be set to canceled instead of completely deleted.
- Fixed Copied! confirmation badges showing the wrong border color and clipping in dark mode.
- Fixed installers, VPP apps, and in-house apps sometimes missing from a host's software details page when more than one install or uninstall was queued for the same item.
- Fixed a server panic when validating a Windows configuration profile that mixes SCEP and non-SCEP `<LocURI>` elements with a non-SCEP element first. The profile is now rejected with a clear validation error.
- Fixed a bug where selected hosts could not be removed (the "X" did nothing) on the live report target selection screen.
- Fixed a bug where if a script-only package was provided with spaces in the path name in a GitOps run, it would fail validation.
- Fixed the GitOps mode tooltip on disabled settings fields so it points at the field's label instead of the center of the label, input, and help text.
- Fixed the dashboard "Hosts enrolled" chart showing an incorrect platform percentage breakdown.
- Fixed Windows MDM not re-installing fleetd on a wiped or re-imaged device that re-enrolls through Autopilot/Entra (OOBE). The server previously treated stale host orbit info as proof fleetd was present and skipped the install, leaving the device MDM-enrolled but without fleetd and hanging the Enrollment Status Page; it now re-delivers fleetd when the host has not checked in since the current enrollment.
- Fixed "My device" page to sort software by display name instead of installer filename when a custom display name is set.
- Fixed a bug where running many concurrent live queries that each target a small number of hosts could overload Redis and slow down host check-ins.
- Fixed browser Back button being trapped on the script batch progress and details pages.
- Fixed a bug where all MDM commands in the command list were incorrectly displayed as "custom MDM command". Only commands run via the custom MDM command API now display this label.
- Fixed fleet-mcp `run_live_query` returning a 403 error for users with the observer+ role. Multi-host live queries now run as an ad-hoc live query campaign (raw SQL, streamed over the results websocket) instead of creating a temporary saved query, so they require only the live-query permission that observer+ already has.
- Fixed GitOps `volume_purchasing_program` failing when using `All fleets` for the `fleets` field.
- Fixed Fleet-maintained apps that share a macOS bundle identifier (for example Firefox and Firefox ESR) so that adding one no longer renames its software title to the other, and no longer shows the other as already added.
- Fixed a generic error in the software install activity modal when using Fleet Free to show a Fleet Premium message instead.
- Fixed an unclear error message that happened when running `fleetctl generate-gitops` with an existing patch policy for an installer that no longer references a Fleet-maintained app because it was deleted from the catalog.
- Fixed the configuration profiles batch endpoint timing out when removing many Windows profiles from a team with a large number of hosts. Deleting Windows profiles (including clearing a team's profiles via GitOps, deleting individual profiles, and deleting a team) now returns quickly and the profiles are removed from hosts in the background by Fleet, the same way profile changes are already delivered.
- Fixed the policy and report details pages briefly showing the previously-viewed policy/report's content when navigating between them.
- Fixed horizontal scrollbar showing up when there is nothing to scroll in report and policy results tables.
- Fixed an issue where Windows and Linux hosts that had already enrolled were prompted for end user authentication (an SSO browser tab) when fleetd re-enrolled after a service restart. Re-enrollment of an already-enrolled host no longer requires end user authentication; only genuinely new devices are prompted.
- Fixed a bug where adding a script-only package via path in GitOps made fleetctl generate-gitops produce an invalid file.
- Fixed an issue where Missing hosts filter and dashboard card incorrectly reported iOS, iPadOS, and Android hosts.
- Fixed password reset, user invite, MFA login, change-email confirmation, and SMTP test emails to no longer duplicate the URL prefix in their links when Fleet is deployed under a subpath.
- Fixed software title details pages timing out for installers, VPP apps, and in-house apps with a large backlog of pending host activities.
- Fixed macOS configuration profiles getting stuck in "Verifying" when a host reported a profile install date in a 12-hour time format.
- Fixed the Fleet-maintained apps list being cut off so that apps near the end of the alphabet were unreachable. The list is now paginated (100 apps per page), and the platform and "Hide added apps" filters are applied across the full library instead of only the loaded apps.
- Fixed GitOps relative path lookup for controls.setup_experience.(apple_setup_assistant, macos_script, software.package_path) in unassigned.yml, and org_logo_paths under org_settings.
- Fixed a bug where a script executed in a scheduled batch would still execute on hosts that had been transferred to a different fleet between the time the batch was scheduled and the time it later executed
- Fixed a bug where the MDM command results endpoint might not return hostnames for all returned hosts
- Fixed the activity feed showing a focus outline when an activity was clicked. The outline now appears only when tabbing to an activity with the keyboard, matching the focus behavior used elsewhere in the UI.
- Fixed the agent settings YAML editor (global and fleet-level) hiding `command_line_flags` behind a comment when set to `{}` or `null`. Those values now render as-is, since they have special semantics (they clear all local osquery flags on hosts).
- Fixed "Select all matching hosts" to display the actual total host count instead of "50+" in both the hosts table header and the delete hosts modal.
- Fixed an issue where the macOS "Update new hosts to latest" OS update setting could stay enabled in GitOps after `minimum_version` and `deadline` were cleared; when `update_new_hosts` isn't explicitly set, it now defaults to enabled only while a minimum version and deadline are configured.
- Fixed an issue where more than 8 entries for OS versions would not be paginated.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.89.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-07-15">
<meta name="articleTitle" value="Fleet 4.89.0 | Windows setup experience improvements, Android variables everywhere, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.89.0-1600x900@2x.png">
