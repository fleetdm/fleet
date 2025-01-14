# Fleet 4.62.0 | Custom targets and automatic policies for software, secrets in configuration profiles and scripts

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/l1IlvGm_jlI" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.62.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.62.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Custom targets for software installs
- Automatic policies for custom packages
- Hide secrets in configuration profiles and scripts

### Custom targets for software installs

IT admins can now install Fleet-maintained apps and custom packages only on macOS, Windows, and Linux hosts within specific labels. This lets you target installations more precisely, tailoring deployments by department, role, or hardware. Learn more about deploying software [here](https://fleetdm.com/guides/deploy-software-packages).

### Automatic policies for custom packages

Fleet now creates policies automatically when you add a custom package. This eliminates the need to manually write policies, making it faster and easier to deploy software across all your hosts. Learn more about automatically installing software [here](https://fleetdm.com/guides/automatic-software-install-in-fleet).

### Hide secrets in configuration profiles and scripts

Fleet ensures that GitHub or GitLab secrets, like API tokens and license keys used in scripts (Shell & PowerShell) and configuration profiles (macOS & Windows), are hidden when viewed or downloaded in Fleet. This protects sensitive information, keeping it secure until it’s deployed to the hosts. Learn more about secrets [here](https://fleetdm.com/guides/secrets-scripts-configuration-profiles).

## Changes

## Endpoint operations
- Updated macos 13, 14 per latest CIS documents. Added macos 15 support.
- Updated queries API to support above targeted platform filtering.
- Updated UI queries page to filter, sort, paginate, etc. via query params in call to server.
- Added searchable query targets and cleaner UI for uses with many teams or labels.

## Device management (MDM)
- Added ability to use secrets (`$FLEET_SECRET_YOURNAME`) in scripts and profiles.
- Added ability to scope Fleet-maintained apps and custom packages via labels in UI, API, and CLI.
- Added capability to automatically generate "trigger policies" for custom software packages.
- Added UI for scoping software via labels.
- Added validation to prevent label deletion if it is used to scope the hosts targeted by a software installer.
- Added ability to filter host software based on label scoping.
- Added support for Fleet secret validation in software installer scripts.
- Updated `fleetctl gitops` to support scope software installers by labels, with the `labels_include_any` or `labels_exclude_any` conditions.
- Updated `fleetctl gitops` to identify secrets in scripts and profiles and saves them on the Fleet server.
- Updated `fleetctl gitops` so that when it updates profiles, if the secret value has changed, the profile is updated on the host.
- Added `/fleet/spec/secret_variables` API endpoint.
- Added functionality for skipping automatic installs if the software is not scoped to the host via labels.
- Added the ability to click a software row on the my device page and see the details of that software's installation on the host.
- Allowed software uninstalls and script-based host lock/unlock/wipe to run while global scripts are disabled.

## Vulnerability management
- Added missing vulncheck data from NVD feeds.
- Fixed MSI parsing for packages including long interned strings (e.g. licenses for the OpenVPN Connect installer).
- Fixed a panic (and resulting failure to load CVE details) on new installs when OS versions have not been populated yet.
- Fixed CVE-2024-10004 false positive on Fleet-supported platforms (vuln is iOS-only and iOS vuln checking is not supported).

## Bug fixes and improvements
- Added license key validation on `fleetctl preview` if a license key is provided; fixes cases where an invalid license key would cause `fleetctl preview` to hang.
- Increased maximum length for installer URLs specified in GitOps to 4000 characters.
- Stopped older scheduled queries from filling logs with errors.
- Changed script upload endpoint (`POST /api/v1/fleet/scripts`) to automatically switch CRLF line endings to LF.
- Fleshed out server response from `queries` endpoint to include `count` and `meta` pagination information.
- Updated platform filtering on queries page to refer to targeted platforms instead of compatible platforms.
- Included osquery pre-releases in daily UI constant update GitHub Actions job.
- Updated to send alert via SNS when a scheduled "cron" job returns errors.
- SNS topic for job error alerts can be configured separately from the existing monitor alert by adding "cron_job_failure_monitoring" to sns_topic_arns_map, otherwise defaults to the using the same topic.
- Improved validation workflow on SMTP settings page.
- Allowed team policy endpoint (`PATCH /api/latest/fleet/teams/{team_id}/policies/{policy_id}`) to receive explicit `null` as a value for `script_id` or `software_title_id` to unset a script or software installer respectively.
- Aliased EAP versions of JetBrains IDEs to "last release version plus all fixes" (e.g. 2024.3 EAP -> 2024.2.99) to avoid vulnerability false positives.
- Removed server error if no private IP was found by detail_query_network_interface.
- Updated `fleetctl` dependencies that cause warnings.
- Added service annotation field to Helm Chart.
- Updated so that on policy deletion any associated pending software installer or scripts are deleted. 
- Added fallback to FileVersion on EXE installers when FileVersion is set but ProductVersion isn't to allow more custom packages to be uploaded.
- Added Mastodon icon and URL to server email templates.
- Improved table text wrapper in UI. 
- Added helpful tooltip for the install software setup experience page.
- Added offset to the tooltips on hover of the profile aggregate status indicators.
- Added the `software_title_id` field to the `added_software` activity details.
- Allow maintainers to manage install software or run scripts on policy automations.
- Removed duplicate software records from homebrew casks already reported in the osquery `apps` table to address false positive vulnerabilities due to lack of bundle_identifier.
- Added the `labels_include_any` and `labels_exclude_any` fields to the software installer activities.
- Updated the get host endpoint to include disk encryption stats for a linux host only if the setting is enabled.
- Updated Helm chart to support customization options such as the Google cloud_sql_proxy in the fleet-migration job.
- Updated example windows policies.
- Added a descriptive error when a GitOps file contains script references that are missing paths.
- Removed `invalid UUID` log message when validating Apple MDM UDID.
- Added validation Fleet secrets embedded into scripts and profiles on ingestion.
- Display the correct percentage of hosts online when there are no hosts online.
- Fixed bug when creating a label to preserve the selected team.
- Fixed export to CSV trimming leading zeros by treating those values as strings.
- Fixed reporting of software uninstall results after a host has been locked/unlocked.
- Fixed issue where minio software was not scanned for vulnerabilities correctly because of unexpected trailing characters in the version string.
- Fixed bug on the "Controls" page where incorrect timestamp information was displayed while the "Current versions" table was loading.
- Fixed policy truncation UI bug.
- Fixed cases where showing results of an inherited query viewed inside a team would include results from hosts not on thta team by adding an optional team_id parameter to queris report endpoint (`GET /api/latest/fleet/queries/{query_id}/report`).
- Fixed issue where deleted Apple config profiles were installing on devices because devices were offline when the profile was added.
- Fixed UI bug involving pagination of subsections within the "Controls" page.
- Fixed "Verifying" disk encryption status count and filter for macOS hosts to not include hosts where end-user action is required.
- Fixed a bug in determining sort type of query result columns by deducing that type from the data present in those columns.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.62.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-01-09">
<meta name="articleTitle" value="Fleet 4.62.0 | Custom targets and automatic policies for software, secrets in configuration profiles/scripts">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.62.0-1600x900@2x.png">
