# Fleet 4.83.0 | Recovery Lock passwords, patch policies, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/DxCqKE8tNyU?si=CKZm_FL0E4UjJf1T" title="0" allowfullscreen></iframe>
</div>

Fleet 4.83.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.83.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- YAML validation for extraneous keys
- macOS Recovery Lock passwords
- Patch policies for Fleet-maintained apps
- Lock end user info during macOS setup

### YAML validation for extraneous keys

Fleet now returns a clear error when a YAML file contains an unrecognized or misspelled key. Previously, Fleet silently ignored unknown keys, which could cause configurations to take effect without the intended settings applied.

This is especially useful for catching typos and errors in AI-generated GitOps PRs before a misconfiguration takes effect.

> **Note:** After upgrading, you may encounter new errors from previously ignored misspelled or misplaced keys. [Reach out to Fleet](https://fleetdm.com/support) if you need help.

GitHub issue: [#40496](https://github.com/fleetdm/fleet/issues/40496)

### macOS Recovery Lock passwords

Fleet now automatically escrows a unique Recovery Lock password for each macOS host and lets admins rotate it on demand. Learn [how to enable this](https://fleetdm.com/guides/recovery-lock-password).

The Recovery Lock passwords prevents unauthorized access to macOS Recovery Mode. When needed, admins can look up and share the password with an end user and then rotate it afterward so it can't be reused. Automatic rotation after view is [coming soon](https://github.com/fleetdm/fleet/issues/41003).

GitHub issues: [#37497](https://github.com/fleetdm/fleet/issues/37497), [#37498](https://github.com/fleetdm/fleet/issues/37498)

### Patch policies for Fleet-maintained apps

Fleet now supports patch policies for Fleet-maintained apps (FMAs). Unlike traditional policies where admins write and maintain the latest version in the SQL themselves, a patch policy automatically generates and updates the SQL when a new version of the app is released. This removes the maintenance burden of keeping patch policies in sync with new software versions. [Learn more](https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet#patch-policies-for-fleet-maintained-apps).

GitHub issue: [#31914](https://github.com/fleetdm/fleet/issues/31914)

### Lock end user info during macOS setup

Fleet now lets IT admins control whether end users can edit their macOS local account "Full Name" and "Account Name" during the Setup Assistant (out-of-box enrollment flow). When **Lock end user info** is enabled, end users cannot modify these fields during setup.

To configure, head to **Controls > Setup experience** and expand the new **Advanced options** section. The **Lock end user info** option is only available when end user authentication is turned on. This setting is also supported via GitOps using the `controls.setup_experience.lock_end_user_info` key.

GitHub issue: [#38669](https://github.com/fleetdm/fleet/issues/38669)

## Changes

### IT Admins
- Added ability to deploy an Android web app via setup experience or self-service.
- Added ability to set and manually rotate Mac recovery lock passwords.
- Added ability to lock the pre-filled user information for macOS hosts that login via End User Authentication during Setup Experience.
- Added automatic retries for failed software installs, excluding VPP apps.
- Updated host software library to always allow filtering.
- Added retry functionality when adding software installers to Fleet via GitOps.
- Added `fleetctl new` command to initialize a GitOps folder.
- Added support for `paths:` key under `reports:`, `labels:` and `policies:` in GitOps files.
- Added glob support for `configuration_profiles` in GitOps files.
- Added support for referencing `.sh` or `.ps1` script files directly in the GitOps `path` field for software packages.
- Implemented `webhooks_and_tickets_enabled` flag for policies in GitOps.
- Added server config for allowing all Apple MDM declaration types.
- Added ability to use `FLEET_JIT_USER_ROLE_FLEET_` as a prefix on SAML attributes.
- Added `fleet_name` and `fleet_id` columns to hosts CSV export.
- Added resend button in the OS settings modal for iOS and iPadOS hosts.
- Added patch policies for Fleet-maintained apps that automatically update when the app is updated.

### Security Engineers
- Added support for NDES CA for Windows hosts.
- Added vulnerability scanning support for Windows Server 2025 hosts.
- Added OTEL instrumentation to Fleet's internal HTTP client.
- Added Content-Type header to Smallstep authorization requests to prevent Cloudflare from blocking them.
- Added ability to omit `secrets:` in GitOps files to retain existing enroll secrets on server.
- Fixed python package false positives on Ubuntu, such as `python3-setuptools` on Ubuntu 24.04 with version 68.1.2-2ubuntu1.2.
- Fixed false positive vulnerabilities for Mattermost Desktop.

### Other improvements and bug fixes
- Most top-level keys can now be omitted from GitOps files in place of supplying them with an empty value.
- Improved host search to always match against host email addresses, not only when the query looks like an email.
- Prevented a 500 error on the host details page when an MDM command reference in `host_mdm_actions` pointed to a non-existent command (orphan reference).
- Allowed Fleet-maintained apps to be added if they have default categories configured that are not available in older builds from this point forward.
- Migrated to using Policy `critical` option when disallowing Okta conditional access bypass.
- Updated DEP enrollment flow to apply minimum macOS version check when specified.
- Updated GitOps to fail runs when unknown keys are detected in files.
- Updated default last opened time diff to 2m to increase the chances of updating the last opened time for software that is opened frequently.
- Updated the host results endpoint URL to be consistent with the other URLs.
- Added tooltip to batch run result host count to clarify that the count might include deleted hosts.
- Updated table heading and result filter styles.
- Reordered the columns on the Hosts page.
- Updated Fleet desktop to surface custom transparency links to the device user.
- Changed `PostJSONWithTimeout` to log response body in error case.
- Removed unused and confusingly-named `--mdm_apple_scep_signer_allow_renewal_days` config.
- Refactored `NewActivity` functionality by moving it to the new activity bounded context.
- Modified Android certificate renewal logic to make it easier to test.
- Optimized `api/latest/fleet/software/titles` endpoint.
- Trimmed incoming `ABM` suffix for Arch Linux hosts so Arch OSs are grouped together in the database and UI.
- Updated determination process used for selecting which user email address to use when scheduling a maintenance event for a host failing policies.
- Added license checks for `fleet-free` targeting queries by label.
- Added APNs expiry banner in the UI for Fleet free users.
- Added error if GitOps/batch attempts to add setup experience software when manual agent install is enabled.
- Added Fleet-maintained app utilization to anonymous usage statistics collected by Fleet.
- Surfaced data constraints using the proper HTTP status code on the `/api/v1/fleet/scim/users` endpoint.
- Updated macOS device details UI to delay showing FileVault "action required" notifications banner during the first hour after MDM enrollment to allow sufficient time for Fleet to automatically escrow keys from ADE devices.
- Added an early return in the `PUT /hosts/{id}/device_mapping` endpoint so that setting the same IDP email that is already stored no longer triggers unnecessary database updates, activity log entries, or profile resends.
- Improved cleanup functionality so that when deleting a host record, Fleet will now clean up host issues, such as failing policies and critical vulnerabilities associated with the host.
- Improved the way we verify Windows profiles to no longer rely on osquery for faster verification.
- Improved body parsing validation by using `http.MaxBytesReader` and wrapping gzip decode output too.
- Improved rate-limiting on conditional access endpoints.
- Finished migrating code from go-kit/log to slog.
- Updated UI for disabling stored report results for clarity.
- Revised which versions Fleet tests MySQL against to 9.5.0 (unchanged), 8.4.8, 8.0.44, and 8.0.39.
- Deprecated several configuration keys in favor of new names: `custom_settings` -> `configuration_profiles`, `macos_settings` -> `apple_settings`, `macos_setup` -> `setup_experience` and `macos_setup_assistant` -> `apple_setup_assistant`.
- Deprecated `setup_experience.bootstrap_package` in favor of `setup_experience.macos_bootstrap_package`.
- Deprecated `setup_experience.manual_agent_install` in favor of `setup_experience.macos_manual_agent_install`.
- Deprecated `setup_experience.enable_release_device_manually` in favor of `setup_experience.apple_enable_release_device_manually`.
- Deprecated `setup_experience.script` in favor of `setup_experience.macos_script`.
- Fixed an issue where the MDM section on the integration page did not update correctly when Apple MDM is turned off.
- Fixed an issue where iOS/iPadOS hosts couldn't add app store apps from the host library page.
- Fixed inaccurate error message when clearing identity provider settings while end user authentication is enabled.
- Fixed Microsoft NDES CA not being selectable after deleting an existing NDES CA without a page refresh.
- Fixed an issue where Apple setup experience could get stuck, if the device was in the middle of a SCEP renewal, and then re-enrolled.
- Fixed `secure.OpenFile` to self-heal incorrect file permissions via `chmod` instead of returning a fatal error.
- Fixed an issue where personal iOS and iPadOS enrollments could see software in the self-service webclip.
- Fixed table footer rendering unexpectedly in the host targets search dropdown.
- Fixed a security issue where canceling a pending lock or wipe command permanently deleted the original `locked_host`/`wiped_host` activity from the audit log. The original activity is now preserved, and the subsequent cancellation activity serves as the follow-up record.
- Fixed dropdown rendering center of a row and from pushing down save button below open dropdown options.
- Fixed end user authentication form to allow saving cleared IdP settings.
- Fixed inconsistent link styling in UI. 
- Fixed the error resend button overflowing over the edge of the OS settings modal table.
- Fixed CPE matching failing for software names that sanitize to FTS5 reserved keywords (AND, OR, NOT).
- Fixed table shifting left when clicking the copy hash icon in host software inventory.
- Fixed a bug where vulnerability counts increased over time due to orphaned entries remaining in the database after hosts were removed.
- Fixed a bug where software installers could create titles with the wrong platform.
- Fixed a bug where Fleet maintained apps for Windows won't show as available in the list when they actually are.
- Fixed host search in live queries returning no results for observer users when many hosts on inaccessible teams matched the search term before accessible ones.
- Fixed live query host/team targeting to correctly scope `observer_can_run` to the query's own team, preventing observers from targeting hosts on other observed teams.
- Fixed alignment of tooltip text in the certificate details modal.
- Fixed a bug where a policy that links a software to install fails to apply when that software package uses an environment variable in its yaml definition.
- Fixed error message when deleting a certificate authority (that is referenced by a certificate template) to show a helpful message instead of a raw database error.
- Fixed observer query bypass by restricting live query/report team targeting to only teams where the user has sufficient permissions, including global observers who are now limited to the query's own team when `observer_can_run` is true.
- Fixed a bug where manage hosts page header button text would wrap and distort at certain widths.
- Fixed an issue where `$FLEET_SECRET` was being double encoded, if set via GitOps.
- Fixed editing reports on free tier failing due to `labels_include_any` triggering a premium license check.
- Fixed a bug where certain incorrect resolved-in versions were reported for certain vulnerable versions of Citrix Workspace.
- Fixed DigiCert CA UPN variable substitution so each host receives a certificate containing its own unique values instead of another host's substituted values.
- Fixed alignment and spacing of the "rolling" tooltip next to "Arch Linux" in the host vitals card.
- Fixed select-all header checkbox not selecting rows on partial pages where not all rows are selectable.
- Fixed an issue where it was possible to configure manual_agent_install without specifiying a bootstrap package via the API and GitOps.
- Fixed dead rows accumulating in software host counts tables by using an atomic table swap instead of in-place updates during the sync process.
- Fixed a bug where script packages (.sh, .ps1) incorrectly used the unsaved script size limit (10K characters) instead of the saved script limit (500K characters), preventing large scripts from being added as software packages.
- Fixed an issue where Windows MDM profiles could remain in pending if hosts acknowledged them too quickly after upload.
- Fixed an issue where users with the same ID as an invited user would be hidden from the users table, and fixed the users count to include invited users.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.83.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-04-01">
<meta name="articleTitle" value="Fleet 4.83.0 | Recovery Lock passwords, patch policies, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.83.0-1600x900@2x.png">
