# Fleet 4.64.0 | Custom targets for software, Bash scripts, fleetctl for Windows and Linux ARM

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/eCWz8o2nqnQ?si=VdxZbsHu8jDK-Hxn" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.64.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.64.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Custom targets for software
- Bash scripts
- Fleetctl for Windows and Linux ARM

### Custom targets for software

IT admins can now install App Store apps only on macOS hosts that match specific labels. This allows for precise app deployment based host attributes like operating system (OS) version, hardware type, and more, ensuring the right apps reach the right devices.

### Bash scripts

Fleet now supports running Bash scripts (`#!/bin/bash`) on macOS and Linux. IT teams can execute scripts with ["bashisms"](https://mywiki.wooledge.org/Bashism) instead of rewriting these scripts to run in Z shell (Zsh).

Also, IT admins can now edit scripts within the Fleet UI. This eliminates the need to download, modify, and re-upload scripts, making it faster to fix typos or make small adjustments on the fly.

### Fleetctl for Windows and Linux ARM

Fleet users with Window or Linux ARM workstations can now use the [fleetctl](https://fleetdm.com/guides/fleetctl) command-line interface (CLI) to run scripts, queries, and more. This expands Fleet’s CLI capabilities, allowing users to manage hosts on their preferred operating system (OS).

## Changes

### Device management (MDM)
- Included current host status and pending action in lock, unlock, and wipe API calls.
- Disk encryption keys are now archived when they are created or updated. They are never fully deleted from the database.
- Hosts that are restored from ABM no longer have old activities in their feed.

### Orchestration
- Added bash interpreter support for script execution.
- Updated the activities feed with new design.
- Added `fleetctl` on Linux ARM binary to releases.
- Added clearer error states to metadata-related fields in the SSO settings form.
- Enforced consistency of on-click behavior of table rows.
- Added gzip compression for static CSS and JS assets to decrease bundle download times.
- Added API endpoint for updating script contents.
- Implemented various UI improvements to the scripts list.
- Added option to populate users and labels on list hosts endpoint.
- Checked the server for validity of any Fleet invites on load.
- Updateed user form validation to require a password be present when switching a user from SSO to password authentication.
- Updated the way new manual labels are created to better support adding large numbers of hosts at one time.
- Replaced "Include Fleet desktop" with host type radio selection buttons when adding Windows or Linux hosts.
- Disabled webhooks if not present in gitops.

### Software
- Added ability to target app store apps with include/exclude labels.
- Added ability to edit targets or self service option for app store apps.
- Added details modal for add, edit, and delete app store app global activities.
- Added modal to edit script contents.
- Added download url for fleet maintained apps as `url` property on `fleet/software/fleet_maintained_apps/:id`.
- Added "exclude_fleet_maintained_apps" option to `GET /api/v1/fleet/software/titles`.
- Surfaced download URL for Fleet-maintained app when adding the software to Fleet.
- Surfaced cleaner errors when adding Fleet-maintained apps.
- Revised software installer package validation to mark installers with no version as "unknown" for version rather than rejecting them.
- Resolved false negatives on vulnerabilities for IntelliJ IDEA Community Edition on Windows.
- Resolved false-positives for the `pass` Homebrew package and `jira` Python package via a vulnerability feed update available to all Fleet versions on 2025-01-22.
- Fixed a false negative vulnerability reporting for iTerm2 (available to all recent Fleet releases as of January 17th via a vulnerability feed update).

### Bug fixes and improvements
- Removed duplicate Linux lock and wipe scripts from repository.
- Clarified text on the policies and queries pages when no policies/queries exist for the selected team (or All Teams).
- Updated the help text for 3 tabs of the Add hosts modal.
- Improved the look and feel of dropdowns in the UI.
- Improved look and feel of dashboard host count cards including hiding platforms with 0 count.
- Added util wrapper func around semver package to allow for custom preprocessing. Upgraded semver library to 3.3.1 and usage everywhere to version 3. 
- Added link to information about installing fleetd when packages are generated.
- Optimized software ingestion queries to use existing DB indexes in the software titles table.
- Normalized padding spacing for list headers, lists, and help text across various modals.
- Removed the resend button for failed windows disk encryption profiles and add messaging that tells the user that Fleet with automatically retry this profile again.
- Refactored upstream error logic to allow disabling submit button when form errors are present.
- Improved the verified and verifying tooltips on the Profile Status on OS settings page.
- Improved settings context so that user's updates to the team agent options form when they navigate away and back again.
- Improved the teams dropdown so that it gracefully hides overflow from long team names.
- Updated the os settings Target form deadline input tooltip to make it more clear how the deadline works for hosts.
- Updated language in query comppatibility tooltip to clarify that compatibility is based only on tables.
- Optimized logging by ensuring illegal argument errors will no longer be logged at the ERROR level on the server. Since these are client errors, they will be logged at the DEBUG level instead. This will reduce the amount of noise in the server logs and help debugging other issues.
- Raised the frequency of sending anonymous statistics from every 24 hours to every 1 hour.
- Bumped Node.js version to 20.18.1.
- Bumped github cache action to 4.2.0.
- Added server debug logging for unexpected Apple DDM configuration status.
- Removed `fleetctl` binary from the `fleetdm/fleet` docker image.
- Removed erroneous "manage automations" link on dashboard for maintainers.
- Fixed window profiles error message being cut off in the OS settings modal.
- Fixed user page responsiveness to not overflow horizontally.
- Fixed case consistency for "Disk encryption" in host OS settings modal.
- Fixed styling for manage automation buttons and dropdown.
- Fixed a bug where query reports where not being recorded for hosts configured with `--logger_snapshot_event_type=true`.
- Fixed incorrect source value in device mapping REST API documentation.
- Fixed a bug in Fleet's handling of VPP token renewal requests.
- Fixed mail being sent with the incorrect SMTP Domain (thank you @mccormickt).
- Fixed filtering by vulnerable software for ios or ipad host.
- Fixed issue where some Windows MDM profiles were not being sent to hosts when hosts came back online.
- Fixed a bug where adding or removing a host with an identical name to/from a label caused the same action to be performed on other host(s) with the same name as well.
- Fixed Windows MDM issue where SessionID of 0 was not allowed.
- Fixed a bug with paginating team policies.
- Fixed a bug "software not found for checksum" in software ingestion transaction retries.
- Fixed issue with Windows disk encryption where status updates from "Verifying" to "Verified" were sometimes stuck in the "Verifying" state.
- Fixed a bug where server errors returned from the API were not successfully being incorporated into the user form error states.
- Fixed a bug where team admins are unable to enable or disable MFA for a user.
- Fixed a bug where only the first of multiple software titles with the same name and source but different bundle IDs would be successfully inserted into the database.
- Fixed issue verifying Windows CSP profiles that contain ADMX policies.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.64.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-02-18">
<meta name="articleTitle" value="Fleet 4.64.0 | Custom targets for software, Bash scripts, fleetctl for Windows and Linux ARM">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.64.0-1600x900@2x.png">
