## Fleet 4.38.0 (Sep 25, 2023)

### Changes

* Updated MDM profile verification so that an install profile command will be retried once if the command resulted in an error or if osquery cannot confirm that the expected profile is installed.

* Ensured post-enrollment commands are sent to devices assigned to Fleet in ABM.

* Ensured hosts assigned to Fleet in ABM come back to pending to the right team after they're deleted.

* Added `labels` to the fleetd extensions feature to allow deploying extensions to hosts that belong to certain labels.

* Changed fleetd Windows extensions file extension from `.ext` to `.ext.exe` to allow their execution on Windows devices (executables on Windows must end with `.exe`).

* Surfaced chrome live query errors to Fleet UI (including errors for specific columns while maintaining successful data in results).

* Fixed delivery of fleetd extensions to devices to only send extensions for the host's platform.

* (Premium only) Added `resolved_in_version` to `/fleet/software` APIs pulled from NVD feed.

* Added database migrations to create the new `scripts` table to store saved scripts.

* Allowed specifying `disable_failing_policies` on the `/api/v1/fleet/hosts/report` API endpoint for increased performance. This is useful if the user is not interested in counting failed policies (`issues` column).

* Added the option to use locally-installed WiX v3 binaries when generating the Fleetd installer for Windows on a Windows machine.

* Added CVE descriptions to the `/fleet/software` API.

* Restored the ability to click on and select/copy text from software bundle tooltips while maintaining the abilities to click the software's name to get more details and to click anywhere else in the row to view all hosts with that software installed.

* Stopped 1password from overly autofilling forms.

* Upgraded Go version to 1.21.1.

### Bug Fixes

* Fixed vulnerability mismatch between the flock browser and the discoteq/flock binary.

* Fixed v4.37.0 performance regressions in the following API endpoints:
  * `/api/v1/fleet/hosts/report`
  * `/api/v1/fleet/hosts` when using `per_page=0` or a large number for `per_page` (in the thousands).

* Fixed script content and output formatting on the scripts detail modal.

* Fixed wrong version numbers for Microsoft Teams in macOS (from invalid format of the form `1.00.XYYYYY` to correct format `1.X.00.YYYYY`).

* Fixed false positive CVE-2020-10146 found on Microsoft Teams.

* Fixed CVE-2013-0340 reporting as a valid vulnerability due to NVD recommendations.

* Fixed save button for a new policy after newly creating another policy.

* Fixed empty query/policy placeholders.

* Fixed used by data when filtering hosts by labels.

* Fixed small copy and alignment issue with status indicators in the Queries page Automations column.

* Fixed strict checks on Windows MDM Automatic Enrollment.

* Fixed software vulnerabilities time ago column for old CVEs.

## Fleet 4.37.0 (Sep 8, 2023)

### Changes

* Added `/scripts/run` and `scripts/run/sync` API endpoints to send a script to be executed on a host and optionally wait for its results.

* Added `POST /api/fleet/orbit/scripts/request` and `POST /api/fleet/orbit/scripts/result` Orbit-specific API endpoints to get a pending script to execute and send the results back, and added an Orbit notification to let the host know it has scripts pending execution.

* Improved performance at scale when applying hundreds of policies to thousands of hosts via `fleetctl apply`.
  - IMPORTANT: In previous versions of Fleet, there was a performance issue (thundering herd) when applying hundreds of policies on a large number of hosts. To avoid this, make sure to deploy this version of Fleet, and make sure Fleet is running for at least 1h (or the configured `FLEET_OSQUERY_POLICY_UPDATE_INTERVAL`) before applying the policies.

* Added pagination to the policies API to increase response time.

* Added policy count endpoints to support pagination on the frontend.

* Added an endpoint to report `fleetd` errors.

* Added logic to report errors during MDM migration.

* Added support in fleetd to execute scripts and send back results (disabled by default).

* Added an activity log when script execution was successfully requested.

* Automatically set the DEP profile to be the same as "no team" (if set) for teams created using the `/match` endpoint (used by Puppet).

* Added JumpCloud to the list of well-known MDM solutions.

* Added `fleetctl run-script` command.

* Made all table links right-clickable.

* Improved the layout of the MDM SSO pages.

* Stored user email when a user turned on MDM features with SSO enabled.

* Updated the copy and image displayed on the MDM migration modal.

* Upgraded Go to v1.19.12.

* Updated the macadmins/osquery-extension to v0.0.15.

* Updated nanomdm dependency.

### Bug Fixes

* Fixed a bug where live query UI and export data tables showed all returned columns.

* Fixed a bug where Jira and/or Zendesk integrations were being removed when an unrelated setting was changed.

* Fixed software ingestion to not re-insert software when incoming fields from hosts were longer than what Fleet supports. This bug caused some CVEs to be reported every time the vulnerability cron ran.
  - IMPORTANT: After deploying this fix, the vulnerability cron will report the CVEs one last time, and subsequent cron runs will not report the CVE (as expected).

* Fixed duplicate policy names in `ee/cis/win-10/cis-policy-queries.yml`.

* Fixed typos in policy queries in the Windows CIS policies YAML (`ee/cis/win-10/cis-policy-queries.yml`).

* Fixed a bug where query stats (aka `Performance impact`) were not being populated in Fleet.

* Added validation to `fleetctl apply` for duplicate policy names in the YAML file and attempting to change the team of an existing policy.

* Optimized host queries when using policy statuses.

* Changed the authentication method during Windows MDM enrollment to use `LoadHostByOrbitNodeKey` instead of `HostByIdentifier`.

* Fixed alignment on long label names on host details label filter dropdown.

* Added UI for script run activity and script details modal.

* Fixed queries navigation bar bug where if in query detail, you could not navigate back to the manage queries table.

* Made policy resolutions that include URLs clickable in the UI.

* Fixed Fleet UI custom query frequency display.

* Fixed live query filter icon and various other live query icons.

* Fixed Fleet UI tabs highlight while tabbing but not on multiple clicks.

* Fixed double scrollbar bug on dashboard page.

## Fleet 4.36.0 (Aug 17, 2023)

* Added the `fleetctl upgrade-packs` command to migrate 2017 packs to the new combined schedule and query concept.

* Updated `fleetctl convert` to convert packs to the new combined schedule and query format.

* Updated the `POST /mdm/apple/profiles/match` endpoint to set the bootstrap package and enable end user authentication settings for each new team created via the endpoint to the corresponding values specified in the app config as of the time the applicable team is created.

* Added enroll secret for a new team created with `fleetctl apply` if none is provided.

* Improved SQL autocomplete with dynamic column, table names, and shown metadata.

* Cleaned up styling around table search bars.

* Updated MDM profile verification to fix issue where profiles were marked as failed when a host 
is transferred to a newly created team that has an identical profile as an older team.

* Added windows MDM automatic enrollment setup pages to Fleet UI.

* (Beta) Allowed configuring Windows MDM certificates using their contents.

* Updated the icons on the dashboard to new grey designs.

* Ensured DEP profiles are assigned even for devices that already exist and have an op type = "modified".

* Disabled save button for invalid query or policy SQL & missing name.

* Users with no global or team role cannot access the UI.

* Text cells truncate with ellipses if longer than column width.
  
**Bug Fixes:**

* Fixed styling issue of the active settings tab.

* Fixed response status code to 403 when a user cannot change their password either because they were not requested to by the admin or they have Single-Sign-On (SSO) enabled.

* Fixed issues with end user migration flow.

* Fixed login form cut off when viewport is too short.

* Fixed bug where `os_version` endpoint returned 404 for `no teams` on controls page.

* Fixed delays applying profiles when the Puppet module is used in distributed scenarios.

* Fixed a style issue in the filter host by status dropdown.

* Fixed an issue when a user with `gitops` role was used to validate a configuration with `fleetctl apply --dry-run`.

* Fixed jumping text on the host page label filter dropdown at low viewport widths.

## Fleet 4.35.2 (Aug 10, 2023)

* Fixed a bug that set a wrong Fleet URL in Windows installers.

## Fleet 4.35.1 (Aug 4, 2023)

* Fixed a migration to account for columns with NULL values as a result of either creating schedules via the API without providing all values or by a race condition with database replicas.

* Fixed a bug that occurred when a user tried to create a custom query from the "query" action on a host's details page.

## Fleet 4.35.0 (Jul 31, 2023)

* Combined the query and schedule features to provide a single interface for creating, scheduling, and tweaking queries at the global and team level.

* Merged all functionality of the schedule page into the queries page.

* Updated the save query modal to include scheduling-related fields.

* Updated queries table schema to allow storing scheduling information and configuration in the queries table.

* Users now able to manage scheduled queries using automations modal.

* The `osquery/config` endpoint now includes scheduled queries for the host's team stored in the `queries` table.

* Query editor now includes frequency and other advanced options.

* Updated macOS MDM setup UI in Fleet UI.

* Changed how team assignment works for the Puppet module, for more details see the [README](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/README.md).

* Allow the Puppet module to read different Fleet URL/token combinations for different environments.

* Updated server logging for webhook requests to mask URL query values if the query param name includes "secret", "token", "key", "password".

* Added support for Azure JWT tokens.

* Set `DeferForceAtUserLoginMaxBypassAttempts` to `1` in the default FileVault profile installed by Fleet.

* Added dark and light mode logo uploads and show the appropriate logo to the macOS MDM migration flow.

* Added MSI installer deployement support through MS-MDM.

* Added support for Windows MDM STS Auth Endpoint.

* Added support for installing Fleetd after enrolling through Azure account.

* Added support for MDM TOS endpoint.

* Updated the "Platforms" column to the more explicit "Compatible with".

* Improved delivery of Apple MDM profiles by not re-sending `InstallProfile` commands if a host switches teams but the profile contents are the same.

* Improved error handling and messaging of SSO login during AEP(DEP) enrollments.

* Improved the reporting of the Puppet module to only report as changed profiles that actually changed during a run.

* Updated ingestion of host detail queries for MDM so hosts that report empty results are counted as "Off".

* Upgraded Go version to v1.19.11.

* If a policy was defined with an invalid query, the desktop endpoint now counts that policy as a failed policy.

* Fixed issue where Orbit repeatedly tries to launch Nudge in the event of a launch error.

* Fixed Observer + should be able to run any query by clicking create new query.

* Fixed the styling of the initial setup flow.

* Fixed URL used to check Gravatar network availability.

## Fleet 4.34.1 (Jul 14, 2023)

* Fixed Observer+ not being able to run some queries.

* If a policy was defined with an invalid query, the desktop endpoint should count that policy as a failed policy.

## Fleet 4.34.0 (Jul 11, 2023)

* Added execution of programmatic Windows MDM enrollment on eligible devices when Windows MDM is enabled.

* Microsoft MDM Enrollment Protocol: Added support for the RequestSecurityToken messages.

* Microsoft MDM Enrollment Protocol: Added support for the DiscoveryRequest messages.

* Microsoft MDM Enrollment Protocol: Added support for the GetPolicies messages.

* Added `enabled_windows_mdm` and `disabled_windows_mdm` activities when a user turns on/off Windows MDM.

* Added support to enable and configure Windows MDM and to notify devices that are able to programmatically enroll.

* Added ability to turn Windows MDM on and off from the Fleet UI.

* Added enable and disable Windows MDM activity UI.

* Updated MDM detail query ingestion to switch MDM profiles from "verifying" or "verified" status to "failed" status when osquery reports that this profile is not installed on the host.

* Added notification and execution of programmatic Windows MDM unenrollment on eligible devices when Windows MDM is disabled.

* Added the `FLEET_DEV_MDM_ENABLED` environment variable to enable the Windows MDM feature during its development and beta period.

* Added the `mdm_enabled` feature flag information to the response payload of the `PATCH /config` endpoint.

* When creating a PolicySpec, return the proper HTTP status code if the team is not found.

* Added CPEMatchingRule type, used for correcting false positives caused by incorrect entries in the NVD dataset.

* Optimized macOS CIS query "Ensure Appropriate Permissions Are Enabled for System Wide Applications" (5.1.5).

* Updated macOS CIS policies 5.1.6 and 5.1.7 to use a new fleetd table `find_cmd` instead of relying on the osquery `file` table to improve performance.

* Implemented the privacy_preferences table for the Fleetd Chrome extension.

* Warnings in fleetctl now go to stderr instead of stdout.

* Updated UI for transferred hosts activity items.

* Added Organization support URL input on the setting page organization info form.

* Added improved ABM 400 error message to the UI.

* Hide any osquery tables or columns from Fleet UI that has hidden set to true to match Fleet website.

* Ignore casing in SAML response for display name. For example the display name attribute can be provided now as `displayname` or `displayName`.

* Provide feedback to users when `fleetctl login` is using EMAIL and PASSWORD environment variables.

* Added a new activity `transferred_hosts` created when hosts are transferred to a new team (or no team).

* Added milliseconds to the timestamp of auto-generated team name when creating a new team in `GET /mdm/apple/profiles/match`.

* Improved dashboard loading states.

* Improved UI for selecting targets.

* Made sure that all configuration profiles and commands are sent to devices if MDM is turned on, even if the device never turned off MDM.

* Fixed bug when reading filevault key in osquery and created new Fleet osquery extension table to read the file directly rather than via filelines table.

* Fixed UI bug on host details and device user pages that caused the software search to not work properly when searching by CVE.

* Fixed not validating the schema used in the Metadata URL.

* Fixed improper HTTP status code if SMTP is invalid.

* Fixed false positives for iCloud on macOS.

* Fixed styling of copy message when copying fields.

* Fixed a bug where an empty file uploaded to `POST /api/latest/fleet/mdm/apple/setup/eula` resulted in a 500; now returns a 400 Bad Request.

* Fixed vulnerability dropdown that was hiding if no vulnerabilities.

* Fixed scroll behavior with disk encryption status.

* Fixed empty software image in sandbox mode.

* Fixed improper HTTP status code when `fleet/forgot_password` endpoint is rate limited. 

* Fixed MaxBurst limit parameter for `fleet/forgot_password` endpoint.

* Fixed a bug where reading from the replica would not read recent writes when matching a set of MDM profiles to a team (the `GET /mdm/apple/profiles/match` endpoint).

* Fixed an issue that displayed Nudge to macOS hosts if MDM was configured but MDM features weren't turned on for the host.

* Fixed tooltip word wrapping on the error cell in the macOS settings table.

* Fixed extraneous loading spinner rendering on the software page.

* Fixed styling bug on setup caused by new font being much wider.

## Fleet 4.33.1 (Jun 20, 2023)

* Fixed ChromeOS add host instructions to use variable Fleet URL.

## Fleet 4.33.0 (Jun 12, 2023)

* Upgraded Go version to 1.19.10.

* Added support for ChromeOS devices.

* Added instructions to inform users how to add ChromeOS hosts.

* Added ChromeOS details to the dashboard, manage hosts, and host details pages.

* Added ability for users to create policies that target ChromeOS.

* Added built-in label for ChromeOS.

* Added query to fill in `device_mapping` from ChromeOS hosts.

* Improved the performance of live query results rendering to address usability issues when querying tens of thousands of hosts.

* Reduced size of live query websocket message by removing unused host data.

* Added the `POST /fleet/mdm/apple/profiles/preassign` endpoint to store profiles to be assigned to a host for subsequent matching with an existing (or new) team.

* Added the `POST /fleet/mdm/apple/profiles/match` endpoint to match pre-assigned profiles to an existing team or create one if needed, and assign the host to that team.

* Updated `GET /mdm/apple/profiles` endpoint to return empty array instead of null if no profiles are found.

* Improved ingestion of MDM devices from ABM:
  - If a device's operation_type is `modified`, but the device doesn't exist in Fleet yet, a DEP profile will be assigned to the device and a new record will be created in Fleet.
  - If a device's operation_type is `deleted`, the device won't be prompted to migrate to Fleet if the feature has been configured.

* Added "Verified" profile status for profiles verified with osquery.

* Added "Action required" status for disk encryption profile in UI for host details and device user pages.

* Added UI for the end user authentication page for MDM macos setup.

* Added new host detail query to verify MDM profiles and updated API to include verified status.

* Added documentation in the guide for `fleetctl get mdm-commands`.

* Moved post-DEP (automatic) MDM enrollment to a worker job for increased resiliency with retries.

* Added better UI error for manual enroll MDM modal.

* Updated `GET /api/_version_/fleet/config` to now omits fields `smtp_settings` and `sso_settings` if not set.

* Added a response payload to the `POST /api/latest/fleet/spec/teams` contributor API endpoint so that it returns an object with a `team_ids_by_name` key which maps team names with their corresponding id.

* Ensure we send post-enrollment commands to MDM devices that are re-enrolling after being wiped.

* Added error message to UI when Redis disconnects during a live query session.

* Optimized query used for listing activities on the dashboard.

* Added ability for users to delete multiple pages of hosts.

* Added ability to deselect label filter on host table.

* Added support for value `null` on `FLEET_JIT_USER_ROLE_GLOBAL` and `FLEET_JIT_USER_ROLE_TEAM_*` SAML attributes. Fleet will accept and ignore such `null` attributes.

* Deprecate `enable_jit_role_sync` setting and only change role for existing users if role attributes are set in the `SAMLResponse`.

* Improved styling in sandbox mode.

* Patched a potential security issue.

* Improved icon clarity.

* Fixed issues with the MDM migration flow.

* Fixed a bug with applying team specs via `fleetctl apply` and updating a team via the `PATCH /api/latest/fleet/mdm/teams/{id}` endpoint so that the MDM updates settings (`minimum_version` and `deadline`) are not cleared if not provided in the payload.

* Fixed table formatting for the output of `fleetctl get mdm-command-results`.

* Fixed the `/api/latest/fleet/mdm/apple_bm` endpoint so that it returns 400 instead of 500 when it fails to authenticate with Apple's Business Manager API, as this indicates a Fleet configuration issue with the Apple BM certificate or token.

* Fixed a bug that would show MDM URLs for the same server as different servers if they contain query parameters.

* Fixed an issue preventing a user with the `gitops` role from applying some MDM settings via `fleetctl apply` (the `macos_setup_assistant` and `bootstrap_package` settings).

* Fixed `GET /api/v1/fleet/spec/labels/{name}` endpoint so that it now includes the label id.

* Fixed Observer/Observer+ role being able to see team secrets.

* Fixed UI bug where `inherited_page=0` was incorrectly added to some URLs.

* Fixed misaligned icons in UI.

* Fixed tab misalignment caused by new font.

* Fixed dashed line styling on multiline activities.

* Fixed a bug in the users table where users that are observer+ for all of more than one team were listed as "Various roles".

* Fixed 500 error being returned if SSO session is not found.

* Fixed issue with `chrome_extensions` virtual table not returning a path value on `fleetd-chrome`, which was breaking software ingestion.

* Fixed bug with page navigation inside 'My Device' page.

* Fixed a styling bug in the add hosts modal in sandbox mode.

## Fleet 4.32.0 (May 24, 2023)

* Added support to add a EULA as part of the AEP/DEP unboxing flow.

* DEP enrollments configured with SSO now pre-populate the username/fullname fields during account
  creation.

* Integrated the macOS setup assistant feature with Apple DEP so that the setup assistants are assigned to the enrolled devices.

* Re-assign and update the macOS setup assistants (and the default one) whenever required, such as
  when it is modified, when a host is transferred, a team is deleted, etc.

* Added device-authenticated endpoint to signal the Fleet server to send a webhook request with the
  device UUID and serial number to the webhook URL configured for MDM migration.

* Added UI for new automatic enrollment under the integration settings.

* Added UI for end-user migration setup.

* Changed macOS settings UI to always show the profile status aggregate data.

* Revised validation errors returned for `fleetctl mdm run-command`.

* Added `mdm.macos_migration` to app config.

* Added `PATCH /mdm/apple/setup` endpoint.

* Added `enable_end_user_authentication` to `mdm.macos_setup` in global app config and team config
  objects.

* Now tries to infer the bootstrap package name from the URL on upload if a content-disposition header is not provided.

* Added wildcards to host search so when searching for different accented characters you get more results.

* Can now reorder (and bookmark) policy tables by failing count.

* On the login and password reset pages, added email validation and fixed some minor styling bugs.

* Ensure sentence casing on labels on host details page.

* Fix 3 Windows CIS benchmark policies that had false positive results initally merged March 24.

* Fix of Fleet Server returning a duplicate OS version for Windows.

* Improved loading UI for disk encryption controls page.

* The 'GET /api/v1/fleet/hosts/{id}' and 'GET /api/v1/fleet/hosts/identifier/{identifier}' now
  include the software installed path on their payload.

* Third party vulnerability integrations now include the installed path of the vulnerable software
  on each host.

* Greyed out unusable select all queries checkbox.

* Added page header for macOS updates UI.

* Back to queries button returns to previous table state.

* Bookmarkable URLs are now source of truth for Manage Queries page table state.

* Added mechanism to refetch MDM enrollment status of a host pending unenrollment (due to a migration to Fleet) at a high interval.

* Made sure every modal in the UI conforms to a consistent system of widths.

* Team admins and team maintainers cannot save/update a global policy so hide the save button when viewing or running a global policy.

* Policy description has text area instead of one-line area.

* Users can now see the filepath of software on a host.

* Added version info metadata file to Windows installer.

* Fixed a bug where policy automations couldn't be updated without a webhook URL.

* Fixed tooltip misalignment on software page.

## Fleet 4.31.1 (May 10, 2023)

* Fixed a bug that prevented bootstrap packages and the `fleetd` agent from being installed when the server had a database replica configured.

## Fleet 4.31.0 (May 1, 2023)

* Added `gitops` user role to Fleet. GitOps users are users that can manage configuration.

* Added the `fleetctl get mdm-commands` command to get a list of MDM commands that were executed. Added the `GET /api/latest/fleet/mdm/apple/commands` API endpoint.

* Added Fleet UI flows for uploading, downloading, deleting, and viewing information about a Fleet MDM
  bootstrap package.

* Added `apple_bm_enabled_and_configured` to app config responses.

* Added support for the `mdm.macos_setup.macos_setup_assistant` key in the 'config' and 'team' YAML
  payloads supported by `fleetctl apply`.

* Added the endpoints to set, get and delete the macOS setup assistant associated with a team or no team (`GET`, `POST` and `DELETE` methods on the `/api/latest/fleet/mdm/apple/enrollment_profile` path).

* Added functionality to gate Apple MDM login behind SAML authentication.

* Added new "verifying" status for MDM profiles.

* Migrated MDM status values from "applied" to "verifying" and updated associated endpoints.

* Updated macOS settings status filters and aggregate counts to more accurately reflect the status of
FileVault settings.

* Filter out non-`observer_can_run` queries for observers in `fleetctl get queries` to match the UI behavior.

* Fall back to a previous NVD release if the asset we want is not in the latest release.

* Users can now click back to software to return to the filtered host details software tab or filtered manage software page.

* Users can now bookmark software table filters.

* Added a maximum height to the teams dropdown, allowing the user to scroll through a large number of
  teams.

* Present the 403 error page when a user with no access logs in.

* Back to hosts and back to software in host details and software details return to previous table
  state.

* Bookmarkable URLs are now the source of truth for Manage Host and Manage Software table states.

* Removed old Okta configuration that was only documented for internal usage. These configs are being replaced for a general approach to gate profiles behind SSO.

* Removed any host's packs information for observers and observer plus in UI.

* Added `changed_macos_setup_assistant` and `deleted_macos_setup_assistant` activities for the macOS setup assistant setting.

* Hide reset sessions in user dropdown for current user.

* Added a suite of UI logic for premium features in the Sandbox environment.

* In Sandbox, added "Premium Feature" icons for premium-only option to designate a policy as "Critical," as well
  as copy to the tooltip above the icon next to policies designated "Critical" in the Manage policies table.

* Added a star to let a sandbox user know that the "Probability of exploit" column of the Manage
  Software page is a premium feature.

* Added "Premium Feature" icons for premium-only columns of the Vulnerabilities table when in
Sandbox mode.

* Inform prospective customers that Teams is a Premium feature.

* Fixed animation for opening edit user modal.

* Fixed nav bar buttons not responsively resizing when small screen widths cannot fit default size nav bar.

* Fixed a bug with and improved the overall experience of tabbed navigation through the setup flow.

* Fixed `/api/_version/fleet/logout` to return HTTP 401 if unauthorized.

* Fixed endpoint to return proper status code (401) on `/api/fleet/orbit/enroll` if secret is invalid.

* Fixed a bug where a white bar appears at the top of the login page before the app renders.

* Fixed bug in manage hosts table where UI elements related to row selection were displayed to a team
  observer user when that user was also a team and maintainer or admin on another team.

* Fixed bug in add policy UI where a user that is team maintainer or team admin cannot access the UI
  to save a new policy if that user is also an observer on another team.

* Fixed UI bug where dashboard links to hosts filtered by platform did not carry over the selected
  team filter.

* Fixed not showing software card on dashboard when clicking on vulnerabilities.

* Fixed a UI bug where fields on the "My account" page were cut off at smaller viewport widths.

* Fixed software table to match UI spec (responsively hidden vulnerabilities/probability of export column under 990px width).

* Fixed a bug where bundle information displayed in tooltips over a software's name was mistakenly
  hidden.

* Fixed an HTTP 500 on `GET /api/_version_/fleet/hosts` returned when `mdm_enrollment_status` is invalid.

## Fleet 4.30.1 (Apr 12, 2023)

* Fixed a UI bug introduced in version 4.30 where the "Show inherited policies" button was not being
  displayed.

* Fixed inherited schedules not appearing on page reload. 

## Fleet 4.30.0 (Apr 10, 2023)

* Removed both `FLEET_MDM_APPLE_ENABLE` and `FLEET_DEV_MDM_ENABLED` feature flags.

* Automatically send a configuration profile for the `fleetd` agent to teams that use DEP enrollment.

* DEP JSON profiles are now automatically created with default values when the server is run.

* Added the `--mdm` and `--mdm-pending` flags to the `fleetctl get hosts` command to list hosts enrolled in Fleet MDM and pending enrollment in Fleet MDM, respectively.

* Added support for the "enrolled" value for the `mdm_enrollment_status` filter and the new `mdm_name` filter for the "List hosts", "Count hosts" and "List hosts in label" endpoints.

* Added the `fleetctl mdm run-command` command, to run any of the [Apple-supported MDM commands](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) on a host.

* Added the `fleetctl get mdm-command-results` sub-command to get the results for a previously-executed MDM command.

* Added API support to filter the host by the disk encryption status on "GET /hosts", "GET /hosts/count",  and "GET /labels/:id/hosts" endpoints.

* Added API endpoint for disk encryption aggregate status data.

* Automatically install `fleetd` for DEP enrolled hosts.

* Updated hosts' profiles status sync to set to "pending" immediately after an action that affects their list of profiles.

* Updated FileVault configuration profile to disallow device user from disabling full-disk encryption. 

* Updated MDM settings so that they are consistent, and updated documentation for clarity, completeness and correctness.

* Added `observer_plus` user role to Fleet. Observers+ are observers that can run any live query.

* Added a premium-only "Published" column to the vulnerabilities table to display when a vulnerability was first published.

* Improved version detection for macOS apps. This fixes some false positives in macOS vulnerability detection.

* If a new CPE translation rule is pushed, the data in the database should reflect that.

* If a false positive is patched, the data in the database should reflect that.

* Include the published date from NVD in the vulnerability object in the API and the vulnerability webhooks (premium feature only).

* User management table informs which users only have API access.

* Added configuration option `websockets_allow_unsafe_origin` to optionally disable the websocket origin check.

* Added new config `prometheus.basic_auth.disable` to allow running the Prometheus endpoint without HTTP Basic Auth.

* Added missing tables to be cleared on host deletion (those that reference the host by UUID instead of ID).

* Introduced new email backend capable of sending email directly using SES APIs.

* Upgraded Go version to 1.19.8 (includes minor security fixes for HTTP DoS issues).

* Uninstalling applications from hosts will remove the corresponding entry in `software` if no more hosts have the application installed.

* Removed the unused "Issuer URI" field from the single sign-on configuration page of the UI.

* Fixed an issue where some icons would appear clipped at certain zoom levels.

* Fixed a bug where some empty table cells were slightly different colors.

* Fixed e-mail sending on user invites and user e-mail change when SMTP server has credentials.

* Fixed logo misalignment.

* Fixed a bug where for certain org logos, the user could still click on it even outside the navbar.

* Fixed styling bugs on the SelectQueryModal.

* Fixed an issue where custom org logos might be displayed off-center.

* Fixed a UI bug where in certain states, there would be extra space at the right edge of the Manage Hosts table.

## Fleet 4.29.1 (Mar 31, 2023)

* Fixed a migration that was causing `fleet prepare db` to fail due to changes in the collation of the tables. IMPORTANT: please make sure to have a database backup before running migrations.

* Fixed an issue where users would see the incorrect disk encryption banners on the My Device page.

## Fleet 4.29.0 (Mar 22, 2023)

* Added implementation of Fleetd for Chrome.

* Added the `mdm.macos_settings.enable_disk_encryption` option to the `fleetctl apply` configuration
  files of "config" and "team" kind as a Fleet Premium feature.

* Added `mdm.macos_settings.disk_encryption` and `mdm.macos_settings.action_required` status fields in the response for a single host (`GET /hosts/{id}` and `GET /device/{token}` endpoints).

* Added MDM solution name to `host.mdm`in API responses.

* Added support for fleetd to enroll a device using its serial number (in addition to its system
  UUID) to help avoid host-matching issues when a host is first created in Fleet via the MDM
  automatic enrollment (Apple Business Manager).

* Added ability to filter data under the Hosts tab by the aggregate status of hosts' MDM-managed macos
settings.

* Added activity feed items for enabling and disabling disk encryption with MDM.

* Added FileVault banners on the Host Details and My Device pages.

* Added activities for when macOS disk encryption setting is enabled or disabled.

* Added UI for fleet mdm managed disk encryption toggling and the disk encryption aggregate data.

* Added support to update a team's disk encryption via the Modify Team (`PATCH /api/latest/fleet/teams/{id}`) endpoint.

* Added a new API endpoint to gate access to an enrollment profile behind Okta authentication.

* Added new configuration values to integrate Okta in the DEP MDM flow.

* Added `GET /mdm/apple/profiles/summary` endpoint.

* Updated API endpoints that use `team_id` query parameter so that `team_id=0`
  filters results to include only hosts that are not assigned to any team.

* Adjusted the `aggregated_stats` table to compute and store statistics for "no team" in addition to
  per-team and for all teams.

* Added MDM profiles status filter to hosts endpoints.

* Added indicators of aggregate host count for each possible status of MDM-enforced mac settings
  (hidden until 4.30.0).

* As part of JIT provisioning, read user roles from SAML custom attributes.

* Added Win 10 policies for CIS Benchmark 18.x.

* Added Win 10 policies for CIS Benchmark 2.3.17.x.

* Added Win 10 policies for CIS Benchmark 2.3.10.x.

* Documented CIS Windows10 Benchmarks 9.2.x to cis policy queries.

* Document CIS Windows10 Benchmarks 9.3.x to cis policy queries.

* Added button to show query on policy results page.

* Run periodic cleanup of pending `cron_stats` outside the `schedule` package to prevent Fleet outages from breaking cron jobs.

* Added an invitation for users to upgrade to Premium when viewing the Premium-only "macOS updates"
  feature.

* Added an icon on the policy table to indicate if a policy is marked critical.

* Added `"instanceID"` (aka `owner` of `locks`) to `schedule` logging (to help troubleshooting when
  running multiple Fleet instances).

* Introduce UUIDs to Fleet errors and logs.

* Added EndeavourOS, Manjaro, openSUSE Leap and Tumbleweed to HostLinuxOSs.

* Global observer can view settings for all teams.

* Team observers can view the team's settings.

* Updated translation rules so that Docker Desktop can be mapped to the correct CPE.

* Pinned Docker image hashes in Dockerfiles for increased security.

* Remove the `ATTACH` check on SQL osquery queries (osquery bug fixed a while ago in 4.6.0).

* Don't return internal error information on Fleet API requests (internal errors are logged to stderr).

* Fixed an issue when applying the configuration YAML returned by `fleetctl get config` with
  `fleetctl apply` when MDM is not enabled.

* Fixed a bug where `fleetctl trigger` doesn't release the schedule lock when the triggered run
  spans the regularly scheduled interval.

* Fixed a bug that prevented starting the Fleet server with MDM features if Apple Business Manager
  (ABM) was not configured.

* Fixed incorrect MDM-related settings documentation and payload response examples.

* Fixed bug to keep team when clicking on policy tab twice.

* Fixed software table links that were cutting off tooltip.

* Fixed authorization action used on host/search endpoint.

## Fleet 4.28.1 (March 14, 2023) 

* Fixed a bug that prevented starting the Fleet server with MDM features if Apple Business Manager (ABM) was not configured.

## Fleet 4.28.0 (Feb 24, 2023)

* Added logic to ingest and decrypt FileVault recovery keys on macOS if Fleet's MDM is enabled.

* Create activity feed types for the creation, update, and deletion of macOS profiles (settings) via
  MDM.

* Added an API endpoint to retrieve a host disk encryption key for macOS if Fleet's MDM is enabled.

* Added UI implementation for users to upload, download, and deleted macos profiles.

* Added activity feed types for the creation, update, and deletion of macOS profiles (settings) via
  MDM.

* Added API endpoints to create, delete, list, and download MDM configuration profiles.

* Added "edited macos profiles" activity when updating a team's (or no team's) custom macOS settings via `fleetctl apply`.

* Enabled installation and auto-updates of Nudge via Orbit.

* Added support for providing `macos_settings.custom_settings` profiles for team (with Fleet Premium) and no-team levels via `fleetctl apply`.

* Added `--policies-team` flag to `fleetctl apply` to easily import a group of policies into a team.

* Remove requirement for Rosetta in installation of macOS packages on Apple Silicon. The binaries have been "universal" for a while now, but the installer still required Rosetta until now.

* Added max height on org logo image to ensure consistent height of the nav bar.

* UI default policies pre-select targeted platform(s) only.

* Parse the Mac Office release notes and use that for doing vulnerability processing.

* Only set public IPs on the `host.public_ip` field and add documentation on how to properly configure the deployment to ingest correct public IPs from enrolled devices.

* Added tooltip with link to UI when Public IP address cannot be determined.

* Update to better URL validation in UI.

* Set policy platforms using the platform checkboxes as a user would expect the options to successfully save.

* Standardized on a default value for empty cells in the UI.

* Added link to query table in UI source (fleetdm.com/tables/table_name).

* Added live query distributed interval warnings on select targets picker and live query result page.

* Added a macOS settings indicator and modal on the host details and device user pages.

* Added configuration parameters for the filesystem logging destination -- max_size, max_age, and max_backups are now configurable rather than hardcoded values.

* Live query/policy selecting "All hosts" is mutually exclusive from other filters.

* Minor server changes to support Fleetd for ChromeOS (to be released soon).

* Fixed `network_interface_unix` and `network_interface_windows` to ingest "Private IPs" only
  (filter out "Public IPs").

* Fixed how the Fleet MDM server URL is generated when stored for hosts enrolled in Fleet MDM.

* Fixed a panic when loading information for a host enrolled in MDM and its `is_server` field is
  `NULL`.

* Fixed bug with host count on hosts filtered by operating system version.

* Fixed permissions warnings reported by Suspicious Package in macos pkg installers. These warnings
  appeared to be purely cosmetic.

* Fixed UI bug: Long words in activity feed wrap within the div.

## Fleet 4.27.1 (Feb 16, 2023)

* Fixed "Turn off MDM" button appearing on host details without Fleet MDM enabled. 

* Upgrade Go to 1.19.6 to remediate some low severity [denial of service vulnerabilities](https://groups.google.com/g/golang-announce/c/V0aBFqaFs_E/m/CnYKgKwBBQAJ) in the standard library.

## Fleet 4.27.0 (Feb 3, 2023)

* Added API endpoint to unenroll a host from Fleet's MDM.

* Added Request CSR and Change default MDM BM team modals to Integrations > MDM.

* Added a `notifications` object to the response payload of `GET /api/fleet/orbit/config` that includes a `renew_enrollment_profile` field to indicate to fleetd that it needs to run a command on the device to renew the DEP enrollment profile.

* Added modal for automatic enrollment of a macOS host to MDM.

* Integrated with CSR request endpoint in fleet UI.

* Updated `Select targets` UI so that `Platforms`, `Teams`, and `Labels` become `AND` filters. Selecting 2 or more `Platforms`, `Teams`, and `Labels` continue to behave as `OR` filters.

* Added new activities to the activities API when a host is enrolled/unenrolled from Fleet's MDM.

* Implemented macOS update version content panel.

* Added an activity `edited_macos_min_version` when the required minimum macOS version is updated.

* Added the `GET /device/{token}/mdm/apple/manual_enrollment_profile` endpoint to allow downloading the manual MDM enrollment profile from the "My Device" page in Fleet Desktop.

* Run authorization checks before processing policy specs.

* Implemented the new Controls page and updated styling of the site-level navigation.

* Made `fleetctl get teams --yaml` output compatible with `fleetctl apply -f`.

* Added the `POST /api/v1/fleet/mdm/apple/request_csr` endpoint to trigger a Certificate Signing Request to fleetdm.com and return the associated APNs private key and SCEP certificate and key.

* Added mdm enrollment status and mdm server url to `GET /hosts` and `GET /hosts/:id` endpoint
  responses.

* Added keys to the `GET /config` and `GET /device/:token` endpoints to inform if Fleet's MDM is properly configured.

* Add edited min macos version activity.

* User can hover over host UUID to see and copy full ID string.

* Made the 'Back to all hosts' link on the host details page fall back to the default path to the
  manage hosts page. This addresses a bug in this functionality when the user navigates directly
  with the URL.

* Implemented the ability for an authorized user to unenroll a host from MDM on its host details page. The host must be enrolled in MDM and online.

* Added nixos to the list of platforms that are detected at linux distributions.

* Allow to configure a minimum macOS version and a deadline for hosts enrolled into Fleet's MDM.

* Added license expiry to account information page for premium users.

* Removed stale time from loading team policies/policy automation so users are provided accurate team data when toggling between teams.

* Updated to software empty states and host details empty states.

* Changed default hosts per page from 100 to 50.

* Support `CrOS` as a valid platform string for customers with ChromeOS hosts.

* Clean tables at smaller screen widths.

* Log failed login attempts for user+pw and SSO logins (in the activity feed).

* Added `meta` attribute to `GET /activities` endpoint that includes pagination metadata. Fixed edge case
  on UI for pagination buttons on activities card.

* Fleet Premium shows pending hosts on the dashboard and manage host page.

* Use stricter file permissions in `fleetctl updates add` command.

* When table only has 1 host, remove bulky tooltip overflow.

* Documented the Apple Push Notification service (APNs) and Apple Business Manager (ABM) setup and renewal steps.

* Fixed pagination on manage host page.


## Fleet 4.26.0 (Jan 13, 2023)

* Added functionality to ingest device information from Apple MDM endpoints so that an MDM device can
  be surfaced in Fleet results while the initial enrollment of the device is pending.

* Added new activities to the activities API when a host is enrolled/unenrolled from Fleet's MDM.

* Added option to filter hosts by MDM enrollment status "pending" to surface devices ordered through
  Apple Business Manager that are still pending enrollment in Fleet's MDM.

* Added a flag to indicate if the Apple Business Manager terms and conditions have changed and must
  be accepted to have automatic enrollment of hosts work again. A banner is added to the output of
  `fleetctl` commands when this is the case.

* Added side navigation layout to the integration page and conditionally show MDM section.

* Added application configuration: mdm.apple_bm_default_team.

* Added modal to allow user to download an enrollment profile required for Fleet MDM enrollment.

* Added new configuration option to set default team for Apple Business Manager.

* Added a software_updated_at column denoting when software was updated for a host.

* Generate audit log for activities (supported log plugins are: `filesystem`, `firehose`, `kinesis`, `lambda`, `pubsub`, `kafkarest`, and `stdout`).

* Added locally-formated datetime tooltips.

* Updated software empty states.

* Autofocus first entry of all forms for better UX.

* Added pendo to sandbox instances.

* Added bookmarkability of url when it includes the `query` query param on the manage hosts page.

* Pack target details show on right side of dropdown.

* Updated buttons to the the new style guide.

* Added a way to override a detail query or disable it through app config.

* Invalid query string will not result in neverending spinner.

* Fixed an issue causing enrollment profiles to fail if the server URL had a trailing slash.

* Fixed an issue that made the first SCEP enrollment during the MDM check-in flow fail in a new setup.

* Fixed panic in `/api/{version}/fleet/hosts/{d}/mdm` when the host does not have MDM data.

* Fixed ingestion of MDM data with empty server URLs (meaning the host is not enrolled to an MDM server).

* Removed stale time from loading team policies/policy automation so users are provided accurate team data when toggling between teams.

## Fleet 4.25.0 (Dec 22, 2022)

* Added new activity that records create/edit/delete user roles.

* Log all successful logins as activity and all attempts with ip in stderr.

* Added API endpoint to generate DEP public and private keys.

* Added ability to mark policy as critical with Fleet Premium.

* Added ability to mark policies run automation for all already failing hosts.

* Added `fleet serve` configuration flags for Apple Push Notification service (APNs) and Simple
  Certificate Enrollment Protocol (SCEP) certificates and keys.

* Added `fleet serve` configuration flags for Apple Business Manager (BM).

* Added `fleetctl trigger` command to trigger an ad hoc run of all jobs in a specified cron
  schedule.

* Added the `fleetctl get mdm_apple` command to retrieve the Apple MDM configuration information. MDM features are not ready for production and are currently in development. These features are disabled by default.

* Added the `fleetctl get mdm_apple_bm` command to retrieve the Apple Business Manager configuration information.

* Added `fleetctl` command to generate APNs CSR and SCEP CA certificate and key pair.

* Add `fleetctl` command to generate DEP public and private keys.

* Windows installer now ensures that the installed osquery version gets removed before installing Orbit.

* Build on Ubuntu 20 to resolve glibc changes that were causing issues for older Docker runtimes.

* During deleting host flow, inform users how to prevent re-enrolling hosts.

* Added functionality to report if a carve failed along with its error message.

* Added the `redis.username` configuration option for setups that use Redis ACLs.

* Windows installer now ensures that no files are left on the filesystem when orbit uninstallation
  process is kicked off.

* Improve how we are logging failed detail queries and windows os version queries.

* Spiffier UI: Add scroll shadows to indicate horizontal scrolling to user.

* Add counts_update_at attribute to GET /hosts/summary/mdm response. update GET /labels/:id/hosts to
  filter by mdm_id and mdm_enrollment_status query params. add mobile_device_management_solution to
  response from GET /labels/:id/hosts when including mdm_id query param. add mdm information to UI for
  windows/all dashboard and host details.

* Fixed `fleetctl query` to use custom HTTP headers if configured.

* Fixed how we are querying and ingesting disk encryption in linux to workaround an osquery bug.

* Fixed buggy input field alignments.

* Fixed to multiselect styling.

* Fixed bug where manually triggering a cron run that preempts a regularly scheduled run causes
  an unexpected shift in the start time of the next interval.

* Fixed an issue where the height of the label for some input fields changed when an error message is displayed.

* Fixed the alignment of the "copy" and "show" button icons in the manage enroll secrets and get API
  token modals.

## Fleet 4.24.1 (Dec 7, 2022)

**This is a security release.**

* Update Go to 1.19.4

## Fleet 4.24.0 (Dec 1, 2022)

* Improve live query activity item in the activity feed on the Dashboard page. Each item will include the user’s name, as well as an option to show the query. If the query has been saved, the item will include the query’s name.

* Improve navigation on Host details page and Dashboard page by adding the ability to navigate back to a tab (ex. Policies) and filter (ex. macOS) respectively.

* Improved performance of the Fleet server by decreasing CPU usage by 20% and memory usage by 3% on average.

* Added tooltips and updated dropdown choices on Hosts and Host details pages to clarify the meanings of "Status: Online" and "Status: Offline."

* Added “Void Linux” to the list of recognized distributions.

* Added clickable rows to software tables to view all hosts filtered by software.

* Added support for more OS-specific osquery command-line flags in the agent options.

* Added links to evented tables and columns that require user context in the query side panel.

* Improved CPU and memory usage of Fleet.

* Removed the Preview payload button from the usage statistics page, as well as its associated logic and unique styles. [See the example usage statistics payload](https://fleetdm.com/docs/using-fleet/usage-statistics#what-is-included-in-usage-statistics-in-fleet) in the Using Fleet documentation.

* Removed tooltips and conditional coloring in the disk space graph for Linux hosts.

* Reduced false negatives for the query used to determine encryption status on Linux systems.

* Fixed long software name from aligning centered.

* Fixed a discrepancy in the height of input labels when there’s a validation error.

## Fleet 4.23.0 (Nov 14, 2022)

* Added preview screenshots for Jira and Zendesk vulnerability tickets for Premium users.

* Improve host detail query to populate primary ip and mac address on host.

* Add option to show public IP address in Hosts table.

* Improve ingress resource by replacing the template with a most recent version, that enables:
  - Not having any annotation hardcoded, all annotations are optional.
  - Custom path, as of now it was hardcoded to `/*`, but depending on the ingress controller, it can require an extra annotation to work with regular expressions.
  - Specify ingressClassName, as it was hardcoded to `gce`, and this is a setting that might be different on each cluster.

* Added ingestion of host orbit version from `orbit_info` osquery extension table.

* Added number of hosts enrolled by orbit version to usage statistics payload.

* Added number of hosts enrolled by osquery version to usage statistics payload.

* Added arch and linuxmint to list of linux distros so that their data is displayed and host count includes them.

* When submitting invalid agent options, inform user how to override agent options using fleetctl force flag.

* Exclude Windows Servers from mdm lists and aggregated data.

* Activity feed includes editing team config file using fleetctl.

* Update Go to 1.19.3.

* Host details page includes information about the host's disk encryption.

* Information surfaced to device user includes all summary/about information surfaced in host details page.

* Support low_disk_space filter for endpoint /labels/{id}/hosts.

* Select targets pages implements cleaner icons.

* Added validation of unknown keys for the Apply Teams Spec request payload (`POST /spec/teams` endpoint).

* Orbit MSI installer now includes the necessary manifest file to use windows_event_log as a logger_plugin.

* UI allows for filtering low disk space hosts by platform.

* Add passed policies column on the inherited policies table for teams.

* Use the MSRC security bulletins to scan for Windows vulnerabilities. Detected vulnerabilities are inserted in a new table, 'operating_system_vulnerabilities'.

* Added vulnerability scores to Jira and Zendesk integrations for Fleet Premium users.

* Improve database usage to prevent some deadlocks.

* Added ingestion of disk encryption status for hosts, and added that flag in the response of the `GET /hosts/{id}` API endpoint.

* Trying to add a host with 0 enroll secrets directs user to manage enroll secrets.

* Detect Windows MDM solutions and add mdm endpoints.

* Styling updates on login and forgot password pages.

* Add UI polish and style fixes for query pages.

* Update styling of tooltips and modals.

* Update colors, issues icon.

* Cleanup dashboard styling.

* Add tooling for writing integration tests on the frontend.

* Fixed host details page so munki card only shows for mac hosts.

* Fixed a bug where duplicate vulnerability webhook requests, jira, and zendesk tickets were being
  made when scanning for vulnerabilities. This affected ubuntu and redhat hosts that support OVAL
  vulnerability detection.

* Fixed bug where password reset token expiration was not enforced.

* Fixed a bug in `fleetctl apply` for teams, where a missing `agent_options` key in the YAML spec
  file would clear the existing agent options for the team (now it leaves it unchanged). If the key
  is present but empty, then it clears the agent options.

* Fixed bug with our CPE matching process. UTM.app was matching to the wrong CPE.

* Fixed an issue where fleet would send invalid usage stats if no hosts were enrolled.

* Fixed an Orbit MSI installer bug that caused Orbit files not to be removed during uninstallation.

## Fleet 4.22.1 (Oct 27, 2022)

* Fixed the error response of the `/device/:token/desktop` endpoint causing problems on free Fleet Desktop instances on versions `1.3.x`.

## Fleet 4.22.0 (Oct 20, 2022)

* Added usage statistics for the weekly count of aggregate policy violation days. One policy violation day is counted for each policy that a host is failing, measured as of the time the count increments. The count increments once per 24-hour interval and resets each week.

* Fleet Premium: Add ability to see how many and which hosts have low disk space (less than 32GB available) on the **Home** page.

* Fleet Premium: Add ability to see how many and which hosts are missing (offline for at least 30 days) on the **Home** page.

* Improved the query console by indicating which columns are required in the WHERE clause, indicated which columns are platform-specific, and adding example queries for almost all osquery tables in the right sidebar. These improvements are also live on [fleetdm.com/tables](https://fleetdm.com/tables)

* Added a new display name for hosts in the Fleet UI. To determine the display name, Fleet uses the `computer_name` column in the [`system_info` table](https://fleetdm.com/tables/system_info). If `computer_name` isn't present, the `hostname` is used instead.

* Added functionality to consider device tokens as expired after one hour. This change is not compatible with older versions of Fleet Desktop. We recommend to manually update Orbit and Fleet Desktop to > v1.0.0 in addition to upgrading the server if:
  * You're managing your own TUF server.
  * You have auto-updates disabled (`fleetctl package [...] --disable-updates`)
  * You have channels pinned to an older version (`fleetctl package [...] --orbit-channel 1.0.0 --desktop-channel 1.1.0`).

* Added security headers to HTML, CSV, and installer responses.

* Added validation of the `command_line_flags` object in the Agent Options section of Organization Settings and Team Settings.

* Added logic to clean up irrelevant policies for a host on re-enrollment (e.g., if a host changes its OS from linux to macOS or it changes teams).

* Added the `inherited_policies` array to the `GET /teams/{team_id}/policies` endpoint that lists the global policies inherited by the team, along with the pass/fail counts for the hosts on that team.

* Added a new UI state for when results are coming in from a live query or policy query.

* Added better team name suggestions to the Create teams modal.

* Clarified last seen time and last fetched time in the Fleet UI.

* Translated technical error messages returned by Agent options validation to be more user-friendly.

* Renamed machine serial to serial number and IPv4 properly to private IP address.

* Fleet Premium: Updated Fleet Desktop to use the `/device/{token}/desktop` API route to display the number of failing policies.

* Made host details software tables more responsive by adding links to software details.

* Fixed a bug in which a user would not be rerouted to the Home page if already logged in.

* Fixed a bug in which clicking the select all checkbox did not select all in some cases.

* Fixed a bug introduced in 4.21.0 where a Windows-specific query was being sent to non-Windows hosts, causing an error in query ingestion for `directIngestOSWindows`.

* Fixed a bug in which uninstalled software (DEB packages) appeared in Fleet.

* Fixed a bug in which a team that didn't have `config.features` settings was edited via the UI, then both `features.enable_host_users` and `features.enable_software_inventory` would be false instead of the global default.

* Fixed a bug that resulted in false negatives for vulnerable versions of Zoom, Google Chrome, Adobe Photoshop, Node.js, Visual Studio Code, Adobe Media Encoder, VirtualBox, Adobe Premiere Pro, Pip, and Firefox software.

* Fixed bug that caused duplicated vulnerabilities to be sent to third-party integrations.

* Fixed panic in `ingestKubequeryInfo` query ingestion.

* Fixed a bug in which `host_count` and `user_count` returned as `0` in the `teams/{id}` endpoint.

* Fixed a bug in which tooltips for Munki issue would be cut off at the edge of the browser window.

* Fixed a bug in which tooltips for Munki issue would be cut off at the edge of the browser window.

* Fixed a bug in which running `fleetctl apply` with the `--dry-run` flag would fail in some cases.

* Fixed a bug in which **Hosts** table displayed 20 hosts per page.

* Fixed a server panic that occured when a team was edited via YAML without an `agent_options` key.

* Fixed an bug where Pop!\_OS hosts were not being included in the linux hosts count on the hosts dashboard page.


## Fleet 4.21.0 (Sep 28, 2022)

* Fleet Premium: Added the ability to know how many hosts and which hosts, on a team, are failing a global policy.

* Added validation to the `config` and `teams` configuration files. Fleet can be managed with [configuration files (YAML syntax)](https://fleetdm.com/docs/using-fleet/configuration-files) and the fleetctl command line tool. 

* Added the ability to manage osquery flags remotely. This requires [Orbit, Fleet's agent manager](https://fleetdm.com/announcements/introducing-orbit-your-fleet-agent-manager). If at some point you revoked an old enroll secret, this feature won't work for hosts that were added to Fleet using this old enroll secret. To manage osquery flags on these hosts, we recommend deploying a new package. Check out the instructions [here on GitHub](https://github.com/fleetdm/fleet/issues/7377).

* Added a `/api/v1/fleet/device/{token}/desktop` API route that returns only the number of failing policies for a specific host.

* Added support for kubequery.

* Added support for an `AC_TEAM_ID` environment variable when creating [signed installers for macOS hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#signing-installers).

* Made cards on the **Home** page clickable.

* Added es_process_file_events, password_policy, and windows_update_history tables to osquery.

* Added activity items to capture when, and by who, agent options are edited.

* Added logging to capture the user’s email upon successful login.

* Increased the size of placeholder text from extra small to small.

* Fixed an error that cleared the form when adding a new integration.

* Fixed an error generating Windows packages with the fleetctl package on non-English localizations of Windows.

* Fixed a bug that showed the small screen overlay when trying to print.

* Fixed the UI bug that caused the label filter dropdown to go under the table header.

* Fixed side panel tooltips to not be wider than side panel causing scroll bug.


## Fleet 4.20.1 (Sep 15, 2022)

**This is a security release.**

* **Security**: Upgrade Go to 1.19.1 to resolve a possible HTTP denial of service vulnerability ([CVE-2022-27664](https://nvd.nist.gov/vuln/detail/CVE-2022-27664)).

* Fixed a bug in which [vulnerability automations](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations) sent duplicate webhooks.

* Fixed a bug in which logging in with single sign-on (SSO) did not work after a failed authorization attempt.

* Fixed a migration error. This only affects Fleet instances that use MariaDB. MariaDB is not [officially supported](https://fleetdm.com/docs/deploying/faq#what-mysql-versions-are-supported). Future issues specific to MariaDB may not be fixed quickly (or at all). We strongly advise migrating to MySQL 8.0.19+.

* Fixed a bug on the **Edit pack** page in which no targets are shown in the target picker.

* Fixed a styling bug on the **Host details > Query > Select a query** modal.

## Fleet 4.20.0 (Sep 9, 2022)

* Add ability to know how many hosts, and which hosts, have Munki issues. This information is presented on the **Home > macOS** page and **Host details** page. This information is also available in the [`GET /api/v1/fleet/macadmins`](https://fleetdm.com/docs/using-fleet/rest-api#get-aggregated-hosts-mobile-device-management-mdm-and-munki-information) and [`GET /api/v1/fleet/hosts/{id}/macadmins`](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-mobile-device-management-mdm-and-munki-information) and API routes.

* Fleet Premium: Added ability to test features, like software inventory, on canary teams by adding a [`features` section](https://fleetdm.com/docs/using-fleet/configuration-files#features) to the `teams` YAML document.

* Improved vulnerability detection for macOS hosts by improving detection of Zoom, Ruby, and Node.js vulnerabilities. Warning: For users that download and sync Fleet's vulnerability feeds manually, there are [required adjustments](https://github.com/fleetdm/fleet/issues/6628) or else vulnerability processing will stop working. Users with the default vulnerability processing settings can safely upgrade without adjustments.

* Fleet Premium: Improved the vulnerability automations by adding vulnerability scores (EPSS probability, CVSS scores, and CISA-known exploits) to the webhook payload. Read more about vulnerability automations on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations).

* Renamed the `host_settings` section to `features` in the the [`config` YAML file](https://fleetdm.com/docs/using-fleet/configuration-files#features). But `host_settings` is still supported for backwards compatibility.

* Improved the activity feed by adding the ability to see who modified agent options and when modifications occurred. This information is available on the Home page in the Fleet UI and the [`GET /activites` API route](https://fleetdm.com/docs/using-fleet/rest-api#activities).

* Improved the [`config` YAML documentation](https://fleetdm.com/docs/using-fleet/configuration-files#organization-settings).

* Improved the **Hosts** page for smaller screen widths.

* Improved the building of osquery installers for Windows (`.msi` packages).

* Added a **Show query** button on the **Schedule** page, which adds the ability to quickly see a query's SQL.

* Improved the Fleet UI by adding loading spinners to all buttons that create or update entities in Fleet (e.g., users).

* Fixed a bug in which a user could not reach some teams in the UI via pagination if there were more than 20 teams.

* Fixed a bug in which a user could not reach some users in the UI via pagination if there were more than 20 users.

* Fixed a bug in which duplicate vulnerabilities (CVEs) sometimes appeared on **Software details** page.

* Fixed a bug in which the count in the **Issues** column (exclamation tooltip) in the **Hosts** table would sometimes not appear.

* Fixed a bug in which no error message would appear if there was an issue while setting up Fleet.

* Fixed a bug in which no error message would appear if users were creating or editing a label with a name or description that was too long.

* Fixed a big in which the example payload for usage statistics included incorrect key names.

* Fixed a bug in which the count above the **Software** table would sometimes not appear.

* Fixed a bug in which the **Add hosts** button would not be displayed when search returned 0 hosts.

* Fixed a bug in which modifying filters on the **Hosts** page would not return the user to the first page of the **Hosts** table. 

## Fleet 4.19.1 (Sep 1, 2022)

* Fix a migration error that may occur when upgrading to Fleet 4.19.0.

* Fix a bug in which the incorrect operating system was displayed for Windows hosts on the **Hosts** page and **Host details** page.

## Fleet 4.19.0 (Aug 22, 2022)

* Warning: Please upgrade to 4.19.1 instead of 4.19.0 due to a migration error included in 4.19.0. Like all releases, Fleet 4.19.1 includes all changes included in 4.19.0.

* Fleet Premium: De-anonymize usage statistics by adding an `organization` property to the usage statistics payload. For Fleet Free instances, organization is reported as "unknown". Documentation on how to disable usage statistics, can be found [here on fleetdm.com](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics).

* Fleet Premium: Added support for Just-in-time (JIT) user provisioning via SSO. This adds the ability to
automatically create Fleet user accounts when a new users attempts to log in to Fleet via SSO. New
Fleet accounts are given the [Observer role](https://fleetdm.com/docs/using-fleet/permissions#user-permissions).

* Improved performance for aggregating software inventory. Aggregate software inventory is displayed on the **Software page** in the Fleet UI.

* Added the ability to see the vendor for Windows programs in software inventory. Vendor data is available in the [`GET /software` API route](https://fleetdm.com/docs/using-fleet/rest-api#software).

* Added a **Mobile device management (MDM) solutions** table to the **Home > macOS** page. This table allows users to see a list of all MDM solutions their hosts are enrolled to and drill down to see which hosts are enrolled to each solution. Note that MDM solutions data is updated as hosts send fresh osquery results to Fleet. This typically occurs in an hour or so of upgrading.

* Added a **Operating systems** table to the **Home > Windows** page. This table allows users to see a list of all Windows operating systems (ex. Windows 10 Pro 21H2) their hosts are running and drill down to see which hosts are running which version. Note that Windows operating system data is updated as hosts send fresh osquery results to Fleet. This typically occurs in an hour or so of upgrading.

* Added a message in `fleetctl` to that notifies users to run `fleet prepare` instead of `fleetctl prepare` when running database migrations for Fleet.

* Improved the Fleet UI by maintaining applied, host filters when a user navigates back to the Hosts page from an
individual host's **Host details** page.

* Improved the Fleet UI by adding consistent styling for **Cancel** buttons.

* Improved the **Queries**, **Schedule**, and **Policies** pages in the Fleet UI by page size to 20
  items. 

* Improve the Fleet UI by informing the user that Fleet only supports screen widths above 768px.

* Added support for asynchronous saving of the hosts' scheduled query statistics. This is an
experimental feature and should only be used if you're seeing performance issues. Documentation
for this feature can be found [here on fleetdm.com](https://fleetdm.com/docs/deploying/configuration#osquery-enable-async-host-processing).

* Fixed a bug in which the **Operating system** and **Munki versions** cards on the **Home > macOS**
page would not stack vertically at smaller screen widths.

* Fixed a bug in which multiple Fleet Desktop icons would appear on macOS computers.

* Fixed a bug that prevented Windows (`.msi`) installers from being generated on Windows machines.

## Fleet 4.18.0 (Aug 1, 2022)

* Added a Call to Action to the failing policy banner in Fleet Desktop. This empowers end-users to manage their device's compliance. 

* Introduced rate limiting for device authorized endpoints to improve the security of Fleet Desktop. 

* Improved styling for tooltips, dropdowns, copied text, checkboxes and buttons. 

* Fixed a bug in the Fleet UI causing text to be truncated in tables. 

* Fixed a bug affecting software vulnerabilities count in Host Details.

* Fixed "Select Targets" search box and updated to reflect currently supported search values: hostname, UUID, serial number, or IPv4.

* Improved disk space reporting in Host Details. 

* Updated frequency formatting for Packs to match Schedules. 

* Replaced "hosts" count with "results" count for live queries.

* Replaced "Uptime" with "Last restarted" column in Host Details.

* Removed vulnerabilities that do not correspond to a CVE in Fleet UI and API.

* Added standard password requirements when users are created by an admin.

* Updated the regexp we use for detecting the major/minor version on OS platforms.

* Improved calculation of battery health based on cycle count. “Normal” corresponds to cycle count < 1000 and “Replacement recommended” corresponds to cycle count >= 1000.

* Fixed an issue with double quotes usage in SQL query, caused by enabling `ANSI_QUOTES` in MySQL.

* Added automated tests for Fleet upgrades.

## Fleet 4.17.1 (Jul 22, 2022)

* Fixed a bug causing an error when converting users to SSO login. 

* Fixed a bug causing the **Edit User** modal to hang when editing multiple users.

* Fixed a bug that caused Ubuntu hosts to display an inaccurate OS version. 

* Fixed a bug affecting exporting live query results.

* Fixed a bug in the Fleet UI affecting live query result counts.

* Improved **Battery Health** processing to better reflect the health of batteries for M1 Macs.

## Fleet 4.17.0 (Jul 8, 2022)

* Added the number of hosts enrolled by operating system (OS) and its version to usage statistics. Also added the weekly active users count to usage statistics.
Documentation on how to disable usage statistics, can be found [here on fleetdm.com](https://fleetdm.com/docs/using-fleet/usage-statistics#disable-usage-statistics).

* Fleet Premium and Fleet Free: Fleet desktop is officially out of beta. This application shows users exactly what's going on with their device and gives them the tools they need to make sure it is secure and aligned with policies. They just need to click an icon in their menu bar. 

* Fleet Premium and Fleet Free: Fleet's osquery installer is officially out of beta. Orbit is a lightweight wrapper for osquery that allows you to easily deploy, configure and keep osquery up-to-date across your organization. 

* Added native support for M1 Macs.

* Added battery health tracking to **Host details** page.

* Improved reporting of error states on the health dashboard and added separate health checks for MySQL and Redis with `/healthz?check=mysql` and `/healthz?check=redis`.

* Improved SSO login failure messaging.

* Fixed osquery tables that report incorrect platforms.

* Added `docker_container_envs` table to the osquery table schema on the **Query* page.

* Updated Fleet host detail query so that the `os_version` for Ubuntu hosts reflects the accurate patch number.

* Improved accuracy of `software_host_counts` by removing hosts from the count if any software has been uninstalled.

* Improved accuracy of the `last_restarted` date. 

* Fixed `/api/_version_/fleet/hosts/identifier/{identifier}` to return the correct value for `host.status`.

* Improved logging when fleetctl encounters permissions errors.

* Added support for scanning RHEL-based and Fedora hosts for vulnerable software using OVAL definitions.

* Fixed SQL generated for operating system version policies to reduce false negatives.

## Fleet 4.16.0 (Jun 20, 2022)

* Fleet Premium: Added the ability to set a Custom URL for the "Transparency" link included in Fleet Desktop. This allows you to use custom branding, as well as gives you control over what information you want to share with your end-users. 

* Fleet Premium: Added scoring to vulnerability detection, including EPSS probability score, CVSS base score, and known exploits. This helps you to quickly categorize which threats need attention today, next week, next month, or "someday."

* Added a ticket-workflow for policy automations. Configured Fleet to automatically create a Jira issue or Zendesk ticket when one or more hosts fail a specific policy.

* Added [Open Vulnerability and Assement Language](https://access.redhat.com/solutions/4161) (`OVAL`) processing for Ubuntu hosts. This increases the accuracy of detected vulnerabilities. 

* Added software details page to the Fleet UI.

* Improved live query experience by saving the state of selected targets and adding count of visible results when filtering columns.

* Fixed an issue where the **Device user** page redirected to login if an expired session token was present. 

* Fixed an issue that caused a delay in availability of **My device** in Fleet Desktop.

* Added support for custom headers for requests made to `fleet` instances by the `fleetctl` command.

* Updated to an improved `users` query in every query we send to osquery.

* Fixed `no such table` errors for `mdm` and `munki_info` for vanilla osquery MacOS hosts.

* Fixed data inconsistencies in policy counts caused when a host was re-enrolled without a team or in a different one.

* Fixed a bug affecting `fleetctl debug` `archive` and `errors` commands on Windows.

* Added `/api/_version_/fleet/device/{token}/policies` to retrieve policies for a specific device. This endpoint can only be accessed with a premium license.

* Added `POST /targets/search` and `POST /targets/count` API endpoints.

* Updated `GET /software`, `GET /software/{:id}`, and `GET /software/count` endpoints to no include software that has been removed from hosts, but not cleaned up yet (orphaned).

## Fleet 4.15.0 (May 26, 2022)

* Expanded beta support for vulnerability reporting to include both Zendesk and Jira integration. This allows users to configure Fleet to automatically create a Zendesk ticket or Jira issue when a new vulnerability (CVE) is detected on your hosts.

* Expanded beta support for Fleet Desktop to Mac and Windows hosts. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a host, generate a Fleet-osquery installer with `fleetctl package` and include the `--fleet-desktop` flag. Then, open this installer on the device.

* Added the ability to see when software was last used on Mac hosts in the **Host Details** view in the Fleet UI. Allows you to know how recently an application was accessed and is especially useful when making decisions about whether to continue subscriptions for paid software and distributing licensces. 

* Improved security by increasing the minimum password length requirement for Fleet users to 12 characters.

* Added Policies tab to **Host Details** page for Fleet Premium users.

* Added `device_mapping` to host information in UI and API responses.

* Deprecated "MIA" host status in UI and API responses.

* Added CVE scores to `/software` API endpoint responses when available.

* Added `all_linux_count` and `builtin_labels` to `GET /host_summary` response.

* Added "Bundle identifier" information as tooltip for macOS applications on Software page.

* Fixed an issue with detecting root directory when using `orbit shell`.

* Fixed an issue with duplicated hosts being sent in the vulnerability webhook payload.

* Added the ability to select columns when exporting hosts to CSV.

* Improved the output of `fleetclt debug errors` and added the ability to print the errors to stdout via the `-stdout` flag.

* Added support for Docker Compose V2 to `fleetctl preview`.

* Added experimental option to save responses to `host_last_seen` queries to the database in batches as well as the ability to configure `enable_async_host_processing` settings for `host_last_seen`, `label_membership` and `policy_membership` independently. 
* Expanded `wifi_networks` table to include more data on macOS and fixed compatibility issues with newer MacOS releases.

* Improved precision in unseen hosts reports sent by the host status webhook.

* Increased MySQL `group_concat_max_len` setting from default 1024 to 4194304.

* Added validation for pack scheduled query interval.

* Fixed instructions for enrolling hosts using osqueryd.

## Fleet 4.14.0 (May 9, 2022)

* Added beta support for Jira integration. This allows users to configure Fleet to
  automatically create a Jira issue when a new vulnerability (CVE) is detected on
  your hosts.

* Added a "Show query" button on the live query results page. This allows users to double-check the
  syntax used and compare this to their results without leaving the current view.

* Added a [Postman
  Collection](https://www.postman.com/fleetdm/workspace/fleet/collection/18010889-c5604fe6-7f6c-44bf-a60c-46650d358dde?ctx=documentation)
  for the Fleet API. This allows users to easily interact with Fleet's API routes so that they can
  build and test integrations.

* Added beta support for Fleet Desktop on Linux. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a Linux device, first add the
`--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
includes Fleet Desktop. Then, open this installer on the device.

* Added `last_opened_at` property, for macOS software, to the **Host details** API route (`GET /hosts/{id}`).

* Improved the **Settings** pages in the the Fleet UI.

* Improved error message retuned when running `fleetctl query` command with missing or misspelled hosts.

* Improved the empty states and forms on the **Policies** page, **Queries** page, and **Host details** page in the Fleet UI.

* All duration settings returned by `fleetctl get config --include-server-config` were changed from
nanoseconds to an easy to read format.

* Fixed a bug in which the "Bundle identifier" tooltips displayed on **Host details > Software** did not
  render correctly.

* Fixed a bug in which the Fleet UI would render an empty Google Chrome profiles on the **Host details** page.

* Fixed a bug in which the Fleet UI would error when entering the "@" characters in the **Search targets** field.

* Fixed a bug in which a scheduled query would display the incorrect name when editing the query on
  the **Schedule** page.

* Fixed a bug in which a deprecation warning would be displayed when generating a `deb` or `rpm`
  Fleet-osquery package when running the `fleetctl package` command.

* Fixed a bug that caused panic errors when running the `fleet serve --debug` command.

## Fleet 4.13.2 (Apr 25, 2022)

* Fixed a bug with os versions not being updated. Affected deployments using MySQL < 5.7.22 or equivalent AWS RDS Aurora < 2.10.1.

## Fleet 4.13.1 (Apr 20, 2022)

* Fixed an SSO login issue introduced in 4.13.0.

* Fixed authorization errors encountered on the frontend login and live query pages.

## Fleet 4.13.0 (Apr 18, 2022)

### This is a security release.

* **Security**: Fixed several post-authentication authorization issues. Only Fleet Premium users that
  have team users are affected. Fleet Free users do not have access to the teams feature and are
  unaffected. See the following security advisory for details: https://github.com/fleetdm/fleet/security/advisories/GHSA-pr2g-j78h-84cr

* Improved performance of software inventory on Windows hosts.

* Added `basic​_auth.username` and `basic_auth.password` [Prometheus configuration options](https://fleetdm.com/docs/deploying/configuration#prometheus). The `GET
/metrics` API route is now disabled if these configuration options are left unspecified. 

* Fleet Premium: Add ability to specify a team specific "Destination URL" for policy automations.
This allows the user to configure Fleet to send a webhook request to a unique location for
policies that belong to a specific team. Documentation on what data is included the webhook
request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations)

* Added the ability to see the total number of hosts with a specific macOS version (ex. 12.3.1) on the
**Home > macOS** page. This information is also available via the [`GET /os_versions` API route](https://fleetdm.com/docs/using-fleet/rest-api#get-host-os-versions).

* Added the ability to sort live query results in the Fleet UI.

* Added a "Vulnerabilities" column to **Host details > Software** page. This allows the user see and search for specific vulnerabilities (CVEs) detected on a specific host.

* Updated vulnerability automations to fire anytime a vulnerability (CVE), that is detected on a
  host, was published to the
  National Vulnerability Database (NVD) in the last 30 days, is detected on a host. In previous
  versions of Fleet, vulnerability automations would fire anytime a CVE was published to NVD in the
  last 2 days.

* Updated the **Policies** page to ask the user to wait to see accurate passing and failing counts for new and recently edited policies.

* Improved API-only (integration) users by removing the requirement to reset these users' passwords
  before use. Documentation on how to use API-only users can be found here on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#using-fleetctl-with-an-api-only-user).

* Improved the responsiveness of the Fleet UI by adding tablet screen width support for the **Software**,
  **Queries**, **Schedule**, **Policies**, **Host details**, **Settings > Teams**, and **Settings > Users** pages.

* Added Beta support for integrating with Jira to automatically create a Jira issue when a
  new vulnerability (CVE) is detected on a host in Fleet. 

* Added Beta support for Fleet Desktop on Windows. Fleet Desktop allows the device user to see
information about their device. To add Fleet Desktop to a Windows device, first add the
`--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
includes Fleet Desktop. Then, open this installer on the device.

* Fixed a bug in which downloading [Fleet's vulnerability database](https://github.com/fleetdm/nvd) failed if the destination directory specified
was not in the `tmp/` directory.

* Fixed a bug in which the "Updated at" time was not being updated for the "Mobile device management
(MDM) enrollment" and "Munki versions" information on the **Home > macOS** page.

* Fixed a bug in which Fleet would consider Docker network interfaces to be a host's primary IP address.

* Fixed a bug in which tables in the Fleet UI would present misaligned buttons.

* Fixed a bug in which Fleet failed to connect to Redis in standalone mode.
## Fleet 4.12.1 (Apr 4, 2022)

* Fixed a bug in which a user could not log in with basic authentication. This only affects Fleet deployments that use a [MySQL read replica](https://fleetdm.com/docs/deploying/configuration#mysql).

## Fleet 4.12.0 (Mar 24, 2022)

* Added ability to update which platform (macOS, Windows, Linux) a policy is checked on.

* Added ability to detect compatibility for custom policies.

* Increased the default session duration to 5 days. Session duration can be updated using the
  [`session_duration` configuration option](https://fleetdm.com/docs/deploying/configuration#session-duration).

* Added ability to see the percentage of hosts that responded to a live query.

* Added ability for user's with [admin permissions](https://fleetdm.com/docs/using-fleet/permissions#user-permissions) to update any user's password.

* Added [`content_type_value` Kafka REST Proxy configuration
  option](https://fleetdm.com/docs/deploying/configuration#kafkarest-content-type-value) to allow
  the use of different versions of the Kafka REST Proxy.

* Added [`database_path` GeoIP configuration option](https://fleetdm.com/docs/deploying/configuration#database-path) to specify a GeoIP database. When configured,
  geolocation information is presented on the **Host details** page and in the `GET /hosts/{id}` API route.

* Added ability to retrieve a host's public IP address. This information is available on the **Host
  details** page and `GET /hosts/{id}` API route.

* Added instructions and materials needed to add hosts to Fleet using [plain osquery](https://fleetdm.com/docs/using-fleet/adding-hosts#plain-osquery). These instructions
can be found in **Hosts > Add hosts > Advanced** in the Fleet UI.

* Added Beta support for Fleet Desktop on macOS. Fleet Desktop allows the device user to see
  information about their device. To add Fleet Desktop to a macOS device, first add the
  `--fleet-desktop` flag to the `fleectl package` command to generate a Fleet-osquery installer that
  includes Fleet Desktop. Then, open this installer on the device.

* Reduced the noise of osquery status logs by only running a host vital query, which populate the
**Host details** page, when the query includes tables that are compatible with a specific host.

* Fixed a bug on the **Edit pack** page in which the "Select targets" element would display the hover effect for the wrong target.

* Fixed a bug on the **Software** page in which software items from deleted hosts were not removed.

* Fixed a bug in which the platform for Amazon Linux 2 hosts would be displayed incorrectly.

## Fleet 4.11.0 (Mar 7, 2022)

* Improved vulnerability processing to reduce the number of false positives for RPM packages on Linux hosts.

* Fleet Premium: Added a `teams` key to the `packs` yaml document to allow adding teams as targets when using CI/CD to manage query packs.

* Fleet premium: Added the ability to retrieve configuration for a specific team with the `fleetctl get team --name
<team-name-here>` command.

* Removed the expiration for API tokens for API-only users. API-only users can be created using the
  `fleetctl user create --api-only` command.

* Improved performance of the osquery query used to collect software inventory for Linux hosts.

* Updated the activity feed on the **Home page** to include add, edit, and delete policy activities.
  Activity information is also available in the `GET /activities` API route.

* Updated Kinesis logging plugin to append newline character to raw message bytes to properly format NDJSON for downstream consumers.

* Clarified why the "Performance impact" for some queries is displayed as "Undetermined" in the Fleet
  UI.

* Added instructions for using plain osquery to add hosts to Fleet in the Fleet View these instructions by heading to **Hosts > Add hosts > Advanced**.

* Fixed a bug in which uninstalling Munki from one or more hosts would result in inaccurate Munki
  versions displayed on the **Home > macOS** page.

* Fixed a bug in which a user, with access limited to one or more teams, was able to run a live query
against hosts in any team. This bug is not exposed in the Fleet UI and is limited to users of the
`POST run` API route. 

* Fixed a bug in the Fleet UI in which the "Select targets" search bar would not return the expected hosts.

* Fixed a bug in which global agent options were not updated correctly when editing these options in
the Fleet UI.

* Fixed a bug in which the Fleet UI would incorrectly tag some URLs as invalid.

* Fixed a bug in which the Fleet UI would attempt to connect to an SMTP server when SMTP was disabled.

* Fixed a bug on the Software page in which the "Hosts" column was not filtered by team.

* Fixed a bug in which global maintainers were unable to add and edit policies that belonged to a
  specific team.

* Fixed a bug in which the operating system version for some Linux distributions would not be
displayed properly.

* Fixed a bug in which configuring an identity provider name to a value shorter than 4 characters was
not allowed.

* Fixed a bug in which the avatar would not appear in the top navigation.


## Fleet 4.10.0 (Feb 13, 2022)

* Upgraded Go to 1.17.7 with security fixes for crypto/elliptic (CVE-2022-23806), math/big (CVE-2022-23772), and cmd/go (CVE-2022-23773). These are not likely to be high impact in Fleet deployments, but we are upgrading in an abundance of caution.

* Added aggregate software and vulnerability information on the new **Software** page.

* Added ability to see how many hosts have a specific vulnerable software installed on the
  **Software** page. This information is also available in the `GET /api/v1/fleet/software` API route.

* Added ability to send a webhook request if a new vulnerability (CVE) is
found on at least one host. Documentation on what data is included the webhook
request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations#vulnerability-automations).

* Added aggregate Mobile Device Management and Munki data on the **Home** page.

* Added email and URL validation across the entire Fleet UI.

* Added ability to filter software by "Vulnerable" on the **Host details** page.

* Updated standard policy templates to use new naming convention. For example, "Is FileVault enabled on macOS
devices?" is now "Full disk encryption enabled (macOS)."

* Added db-innodb-status and db-process-list to `fleetctl debug` command.

* Fleet Premium: Added the ability to generate a Fleet installer and manage enroll secrets on the **Team details**
  page. 

* Added the ability for users with the observer role to view which platforms (macOS, Windows, Linux) a query
  is compatible with.

* Improved the experience for editing queries and policies in the Fleet UI.

* Improved vulnerability processing for NPM packages.

* Added supports triggering a webhook for newly detected vulnerabilities with a list of affected hosts.

* Added filter software by CVE.

* Added the ability to disable scheduled query performance statistics.

* Added the ability to filter the host summary information by platform (macOS, Windows, Linux) on the **Home** page.

* Fixed a bug in Fleet installers for Linux in which a computer restart would stop the host from
  reporting to Fleet.

* Made sure ApplyTeamSpec only works with premium deployments.

* Disabled MDM, Munki, and Chrome profile queries on unsupported platforms to reduce log noise.

* Properly handled paths in CVE URL prefix.

## Fleet 4.9.1 (Feb 2, 2022)

### This is a security release.

* **Security**: Fixed a vulnerability in Fleet's SSO implementation that could allow a malicious or compromised SAML Service Provider (SP) to log into Fleet as an existing Fleet user. See https://github.com/fleetdm/fleet/security/advisories/GHSA-ch68-7cf4-35vr for details.

* Allowed MSI packages generated by `fleetctl package` to reinstall on Windows without uninstall.

* Fixed a bug in which a team's scheduled queries didn't render correctly on the **Schedule** page.

* Fixed a bug in which a new policy would always get added to "All teams" rather than the selected team.

## Fleet 4.9.0 (Jan 21, 2022)

* Added ability to apply a `policy` yaml document so that GitOps workflows can be used to create and
  modify policies.

* Added ability to run a live query that returns 1,000+ results in the Fleet UI by adding
  client-side pagination to the results table.

* Improved the accuracy of query platform compatibility detection by adding recognition for queries
  with the `WITH` expression.

* Added ability to open a page in the Fleet UI in a new tab by "right-clicking" an item in the navigation.

* Improved the [live query API route (`GET /api/v1/queries/run`)](https://fleetdm.com/docs/using-fleet/rest-api#run-live-query) so that it successfully return results for Fleet
  instances using a load balancer by reducing the wait period to 25 seconds.

* Improved performance of the Fleet UI by updating loading states and reducing the number of requests
  made to the Fleet API.

* Improved performance of the MySQL database by updating the queries used to populate host vitals and
  caching the results.

* Added [`read_timeout` Redis configuration
  option](https://fleetdm.com/docs/deploying/configuration#redis-read-timeout) to customize the
  maximum amount of time Fleet should wait to receive a response from a Redis server.

* Added [`write_timeout` Redis configuration
  option](https://fleetdm.com/docs/deploying/configuration#redis-write-timeout) to customize the
  maximum amount of time Fleet should wait to send a command to a Redis server.

* Fixed a bug in which browser extensions (Google Chrome, Firefox, and Safari) were not included in
  software inventory.

* Improved the security of the **Organization settings** page by preventing the browser from requesting
  to save SMTP credentials.

* Fixed a bug in which an existing pack's targets were not cleaned up after deleting hosts, labels, and teams.

* Fixed a bug in which non-existent queries and policies would not return a 404 not found response.

### Performance

* Our testing demonstrated an increase in max devices served in our load test infrastructure to 70,000 from 60,000 in v4.8.0.

#### Load Test Infrastructure

* Fleet server
  * AWS Fargate
  * 2 tasks with 1024 CPU units and 2048 MiB of RAM.

* MySQL
  * Amazon RDS
  * db.r5.2xlarge

* Redis
  * Amazon ElastiCache 
  * cache.m5.large with 2 replicas (no cluster mode)

#### What was changed to accomplish these improvements?

* Optimized the updating and fetching of host data to only send and receive the bare minimum data
  needed. 

* Reduced the number of times host information is updated by caching more data.

* Updated cleanup jobs and deletion logic.

#### Future improvements

* At maximum DB utilization, we found that some hosts fail to respond to live queries. Future releases of Fleet will improve upon this.

## Fleet 4.8.0 (Dec 31, 2021)

* Added ability to configure Fleet to send a webhook request with all hosts that failed a
  policy. Documentation on what data is included the webhook
  request and when the webhook request is sent can be found here on [fleedm.com/docs](https://fleetdm.com/docs/using-fleet/automations).

* Added ability to find a user's device in Fleet by filtering hosts by email associated with a Google Chrome
  profile. Requires the [macadmins osquery
  extension](https://github.com/macadmins/osquery-extension) which comes bundled in [Fleet's osquery
  installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). 
  
* Added ability to see a host's Google Chrome profile information using the [`GET
  api/v1/fleet/hosts/{id}/device_mapping` API
  route](https://fleetdm.com/docs/using-fleet/rest-api#get-host-device-mapping).

* Added ability to see a host's mobile device management (MDM) enrollment status, MDM server URL,
  and Munki version on a host's **Host details** page. Requires the [macadmins osquery
  extension](https://github.com/macadmins/osquery-extension) which comes bundled in [Fleet's osquery
  installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). 

* Added ability to see a host's MDM and Munki information with the [`GET
  api/v1/fleet/hosts/{id}/macadmins` API
  route](https://fleetdm.com/docs/using-fleet/rest-api#list-mdm-and-munki-information-if-available).

* Improved the handling of certificates in the `fleetctl package` command by adding a check for a
  valid PEM file.

* Updated [Prometheus Go client library](https://github.com/prometheus/client_golang) which
  results in the following breaking changes to the [`GET /metrics` API
  route](https://fleetdm.com/docs/using-fleet/monitoring-fleet#metrics):
  `http_request_duration_microseconds` is now `http_request_duration_seconds_bucket`,
  `http_request_duration_microseconds_sum` is now `http_request_duration_seconds_sum`,
  `http_request_duration_microseconds_count` is now `http_request_duration_seconds_count`,
  `http_request_size_bytes` is now `http_request_size_bytes_bucket`, and `http_response_size_bytes`
  is now `http_response_size_bytes_bucket`

* Improved performance when searching and sorting hosts in the Fleet UI.

* Improved performance when running a live query feature by reducing the load on Redis.

* Improved performance when viewing software installed across all hosts in the Fleet
  UI.

* Fixed a bug in which the Fleet UI presented the option to download an undefined certificate in the "Generate installer" instructions.

* Fixed a bug in which database migrations failed when using MariaDB due to a migration introduced in Fleet 4.7.0.

* Fixed a bug that prevented hosts from checking in to Fleet when Redis was down.

## Fleet 4.7.0 (Dec 14, 2021)

* Added ability to create, modify, or delete policies in Fleet without modifying saved queries. Fleet
  4.7.0 introduces breaking changes to the `/policies` API routes to separate policies from saved
  queries in Fleet. These changes will not affect any policies previously created or modified in the
  Fleet UI.

* Turned on vulnerability processing for all Fleet instances with software inventory enabled.
  [Vulnerability processing in Fleet](https://fleetdm.com/docs/using-fleet/vulnerability-processing)
  provides the ability to see all hosts with specific vulnerable software installed. 

* Improved the performance of the "Software" table on the **Home** page.

* Improved performance of the MySQL database by changing the way a host's users information   is saved.

* Added ability to select from a library of standard policy templates on the **Policies** page. These
  pre-made policies ask specific "yes" or "no" questions about your hosts. For example, one of
  these policy templates asks "Is Gatekeeper enabled on macOS devices?"

* Added ability to ask whether or not your hosts have a specific operating system installed by
  selecting an operating system policy on the **Host details** page. For example, a host that is
  running macOS 12.0.1 will present a policy that asks "Is macOS 12.0.1 installed on macOS devices?"

* Added ability to specify which platform(s) (macOS, Windows, and/or Linux) a policy is checked on.

* Added ability to generate a report that includes which hosts are answering "Yes" or "No" to a 
  specific policy by running a policy's query as a live query.

* Added ability to see the total number of installed software software items across all your hosts.

* Added ability to see an example scheduled query result that is sent to your configured log
  destination. Select "Schedule a query" > "Preview data" on the **Schedule** page to see the 
  example scheduled query result.

* Improved the host's users information by removing users without login shells and adding users 
  that are not associated with a system group.

* Added ability to see a Fleet instance's missing migrations with the `fleetctl debug migrations`
  command. The `fleet serve` and `fleet prepare db` commands will now fail if any unknown migrations
  are detected.

* Added ability to see syntax errors as your write a query in the Fleet UI.

* Added ability to record a policy's resolution steps that can be referenced when a host answers "No" 
  to this policy.

* Added server request errors to the Fleet server logs to allow for troubleshooting issues with the 
Fleet server in non-debug mode.

* Increased default login session length to 24 hours.

* Fixed a bug in which software inventory and disk space information was not retrieved for Debian hosts.

* Fixed a bug in which searching for targets on the **Edit pack** page negatively impacted performance of 
  the MySQL database.

* Fixed a bug in which some Fleet migrations were incompatible with MySQL 8.

* Fixed a bug that prevented the creation of osquery installers for Windows (.msi) when a non-default 
  update channel is specified.

* Fixed a bug in which the "Software" table on the home page did not correctly filtering when a
  specific team was selected on the **Home** page.

* Fixed a bug in which users with "No access" in Fleet were presented with a perpetual 
  loading state in the Fleet UI.

## Fleet 4.6.2 (Nov 30, 2021)

* Improved performance of the **Home** page by removing total hosts count from the "Software" table.

* Improved performance of the **Queries** page by adding pagination to the list of queries.

* Fixed a bug in which the "Shell" column of the "Users" table on the **Host details** page would sometimes fail to update.

* Fixed a bug in which a host's status could quickly alternate between "Online" and "Offline" by increasing the grace period for host status.

* Fixed a bug in which some hosts would have a missing `host_seen_times` entry.

* Added an `after` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow for cursor pagination.

* Added a `disable_failing_policies` parameter to the [`GET /hosts` API route](https://fleetdm.com/docs/using-fleet/rest-api#list-hosts) to allow the API request to respond faster if failing policies count information is not needed.

## Fleet 4.6.1 (Nov 21, 2021)

* Fixed a bug (introduced in 4.6.0) in which Fleet used progressively more CPU on Redis, resulting in API and UI slowdowns and inconsistency.

* Made `fleetctl apply` fail when the configuration contains invalid fields.

## Fleet 4.6.0 (Nov 18, 2021)

* Fleet Premium: Added ability to filter aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page. The aggregate host data is also available in the [`GET /host_summary API route`](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-summary).

* Fleet Premium: Added ability to move pending invited users between teams.

* Fleet Premium: Added `fleetctl updates rotate` command for rotation of keys in the updates system. The `fleetctl updates` command provides the ability to [self-manage an agent update server](https://fleetdm.com/docs/deploying/fleetctl-agent-updates).

* Enabled the software inventory by default for new Fleet instances. The software inventory feature can be turned on or off using the [`enable_software_inventory` configuration option](https://fleetdm.com/docs/using-fleet/vulnerability-processing#setup).

* Updated the JSON payload for the host status webhook by renaming the `"message"` property to `"text"` so that the payload can be received and displayed in Slack.

* Removed the deprecated `app_configs` table from Fleet's MySQL database. The `app_config_json` table has replaced it.

* Improved performance of the policies feature for Fleet instances with over 100,000 hosts.

* Added instructions in the Fleet UI for generating an osquery installer for macOS, Linux, or Windows. Documentation for generating an osquery installer and distributing the installer to your hosts to add them to Fleet can be found here on [fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/adding-hosts)

* Added ability to see all the software, and filter by vulnerable software, installed across all your hosts on the **Home** page. Each software's `name`, `version`, `hosts_count`, `vulnerabilities`, and more is also available in the [`GET /software` API route](https://fleetdm.com/docs/using-fleet/rest-api#software) and `fleetctl get software` command.

* Added ability to add, edit, and delete enroll secrets on the **Hosts** page.

* Added ability to see aggregate host data such as platforms (macOS, Windows, and Linux) and status (online, offline, and new) the **Home** page.

* Added ability to see all of the queries scheduled to run on a specific host on the **Host details** page immediately after a query is added to a schedule or pack.

* Added a "Shell" column to the "Users" table on the **Host details** page so users can now be filtered to see only those who have logged in.

* Packaged osquery's `certs.pem` in `fleetctl package` to improve TLS compatibility.

* Added support for packaging an osquery flagfile with `fleetctl package --osquery-flagfile`.

* Used "Fleet osquery" rather than "Orbit osquery" in packages generated by `fleetctl package`.

* Clarified that a policy in Fleet is a yes or no question you can ask about your hosts by replacing "Passing" and "Failing" text with "Yes" and "No" respectively on the **Policies** page and **Host details** page.

* Added ability to see the original author of a query on the **Query** page.

* Improved the UI for the "Software" table and "Policies" table on the **Host details** page so that it's easier to pivot to see all hosts with a specific software installed or answering "No" to a specific policy.

* Fixed a bug in which modifying a specific target for a live query, in target selector UI, would deselect a different target.

* Fixed a bug in which the user was navigated to a non existent page, in the Fleet UI, after saving a pack.

* Fixed a bug in which long software names in the "Software" table caused the bundle identifier tooltip to be inaccessible.

## Fleet 4.5.1 (Nov 10, 2021)

* Fixed performance issues with search filtering on manage queries page.

* Improved correctness and UX for query platform compatibility.

* Fleet Premium: Shows correct hosts when a team is selected.

* Fixed a bug preventing login for new SSO users.

* Added always return the `disabled` value in the `GET /api/v1/fleet/packs/{id}` API (previously it was
  sometimes left out).

## Fleet 4.5.0 (Nov 1, 2021)

* Fleet Premium: Added a Team admin user role. This allows users to delegate the responsibility of managing team members in Fleet. Documentation for the permissions associated with the Team admin and other user roles can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions).

* Added Apache Kafka logging plugin. Documentation for configuring Kafka as a logging plugin can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#kafka-rest-proxy-logging). Thank you to Joseph Macaulay for adding this capability.

* Added support for [MinIO](https://min.io/) as a file carving backend. Documentation for configuring MinIO as a file carving backend can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#minio). Thank you to Chandra Majumdar and Ben Edwards for adding this capability.

* Added support for generating a `.pkg` osquery installer on Linux without dependencies (beyond Docker) with the `fleetctl package` command.

* Improved the performance of vulnerability processing by making the process consume less RAM. Documentation for the vulnerability processing feature can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Added the ability to run a live query and receive results using only the Fleet REST API with a `GET /api/v1/fleet/queries/run` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#run-live-query).

* Added ability to see whether a specific host is "Passing" or "Failing" a policy on the **Host details** page. This information is also exposed in the `GET api/v1/fleet/hosts/{id}` API route. In Fleet, a policy is a "yes" or "no" question you can ask of all your hosts.

* Added the ability to quickly see the total number of "Failing" policies for a particular host on the **Hosts** page with a new "Issues" column. Total "Issues" are also revealed on a specific host's **Host details** page.

* Added the ability to see which platforms (macOS, Windows, Linux) a specific query is compatible with. The compatibility detected by Fleet is estimated based on the osquery tables used in the query.

* Added the ability to see whether your queries have a "Minimal," "Considerable," or "Excessive" performance impact on your hosts. Query performance information is only collected when a query runs as a scheduled query.

  * Running a "Minimal" query, very frequently, has little to no impact on your host's performance.

  * Running a "Considerable" query, frequently, can have a noticeable impact on your host's performance.

  * Running an "Excessive" query, even infrequently, can have a significant impact on your host’s performance.

* Added the ability to see a list of hosts that have a specific software version installed by selecting a software version on a specific host's **Host details** page. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](https://fleetdm.com/docs/deploying/configuration#feature-flags).

* Added the ability to see all vulnerable software detected across all your hosts with the `GET /api/v1/fleet/software` API route. Documentation for this new API route can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/rest-api#software).

* Added the ability to see the exact number of hosts that selected filters on the **Hosts** page. This ability is also available when using the `GET api/v1/fleet/hosts/count` API route.

* Added ability to automatically "Refetch" host vitals for a particular host without manually reloading the page.

* Added ability to connect to Redis with TLS. Documentation for configuring Fleet to use a TLS connection to the Redis server can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-use-tls).

* Added `cluster_read_from_replica` Redis to specify whether or not to prefer readying from a replica when possible. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-read-from-replica).

* Improved experience of the Fleet UI by preventing autocomplete in forms.

* Fixed a bug in which generating an `.msi` osquery installer on Windows would fail with a permission error.

* Fixed a bug in which turning on the host expiry setting did not remove expired hosts from Fleet.

* Fixed a bug in which the Software inventory for some host's was missing `bundle_identifier` information.

## Fleet 4.4.3 (Oct 21, 2021)

* Cached AppConfig in redis to speed up requests and reduce MySQL load.

* Fixed migration compatibility with MySQL GTID replication.

* Improved performance of software listing query.

* Improved MSI generation compatibility (for macOS M1 and some Virtualization configurations) in `fleetctl package`.

## Fleet 4.4.2 (Oct 14, 2021)

* Fixed migration errors under some MySQL configurations due to use of temporary tables.

* Fixed pagination of hosts on host dashboard.

* Optimized HTTP requests on host search.

## Fleet 4.4.1 (Oct 8, 2021)

* Fixed database migrations error when updating from 4.3.2 to 4.4.0. This did not effect upgrades
  between other versions and 4.4.0.

* Improved logging of errors in fleet serve.

## Fleet 4.4.0 (Oct 6, 2021)

* Fleet Premium: Teams Schedules show inherited queries from All teams (global) Schedule.

* Fleet Premium: Team Maintainers can modify and delete queries, and modify the Team Schedule.

* Fleet Premium: Team Maintainers can delete hosts from their teams.

* `fleetctl get hosts` now shows host additional queries if there are any.

* Update default homepage to new dashboard.

* Added ability to bulk delete hosts based on manual selection and applied filters.

* Added display macOS bundle identifiers on software table if available.

* Fixed scroll position when navigating to different pages.

* Fleet Premium: When transferring a host from team to team, clear the Policy results for that host.

* Improved stability of host vitals (fix cases of dropping users table, disk space).

* Improved performance and reliability of Policy database migrations.

* Provided a more clear error when a user tries to delete a query that is set in a Policy.

* Fixed query editor Delete key and horizontal scroll issues.

* Added cleaner buttons and icons on Manage Hosts Page.

## Fleet 4.3.2 (Sept 29, 2021)

* Improved database performance by reducing the amount of MySQL database queries when a host checks in.

* Fixed a bug in which users with the global maintainer role could not edit or save queries. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Fixed a bug in which policies were checked about every second and add a `policy_update_interval` osquery configuration option. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-policy-update-interval).

* Fixed a bug in which edits to a query’s name, description, SQL did not appear until the user refreshed the Edit query page.

* Fixed a bug in which the hosts count for a label returned 0 after modifying a label’s name or description.

* Improved error message when attempting to create or edit a user with an email that already exists.

## Fleet 4.3.1 (Sept 21, 2021)

* Added `fleetctl get software` command to list all software and the detected vulnerabilities. The Vulnerable software feature is currently in Beta. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://fleetdm.com/docs/using-fleet/vulnerability-processing).

* Added `fleetctl vulnerability-data-stream` command to sync the vulnerabilities processing data streams by hand.

* Added `disable_data_sync` vulnerabilities configuration option to avoid downloading the data streams. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#disable-data-sync).

* Only shows observers the queries they have permissions to run on the **Queries** page. In, Fleet 4.0.0, the Admin, Maintainer, and Observer user roles were introduced. Documentation for the permissions associated with each role can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/using-fleet/permissions). 

* Added `connect_retry_attempts` Redis configuration option to retry failed connections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-connect-retry-attempts).

* Added `cluster_follow_redirections` Redis configuration option to follow cluster redirections. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#redis-cluster-follow-redirections).

* Added `max_jitter_percent` osquery configuration option to prevent all hosts from returning data at roughly the same time. Note that this improves the Fleet server performance, but it will now take longer for new labels to populate. Documentation for this configuration option can be found [here on fleetdm.com/docs](https://fleetdm.com/docs/deploying/configuration#osquery-max-jitter-percent).

* Improved the performance of database migrations.

* Reduced database load for label membership recording.

* Added fail early if the process does not have permissions to write to the logging file.

* Added completely skip trying to save a host's users and software inventory if it's disabled to reduce database load. 

* Fixed a bug in which team maintainers were unable to run live queries against the hosts assigned to their team(s).

* Fixed a bug in which a blank screen would intermittently appear on the **Hosts** page.

* Fixed a bug detecting disk space for hosts.

## Fleet 4.3.0 (Sept 13, 2021)

* Added Policies feature for detecting device compliance with organizational policies.

* Run/edit query experience has been completely redesigned.

* Added support for MySQL read replicas. This allows the Fleet server to scale to more hosts.

* Added configurable webhook to notify when a specified percentage of hosts have been offline for over the specified amount of days.

* Added `fleetctl package` command for building Orbit packages.

* Added enroll secret dialog on host dashboard.

* Exposed free disk space in gigs and percentage for hosts.

* Added 15-minute interval option on Schedule page.

* Cleaned up advanced options UI.

* 404 and 500 page now include buttons for Osquery community Slack and to file an issue

* Updated all empty and error states for cleaner UI.

* Added warning banners in Fleet UI and `fleetctl` for license expiration.

* Rendered query performance information on host vitals page pack section.

* Improved performance for app loading.

* Made team schedule names more user friendly and hide the stats for global and team schedules when showing host pack stats.

* Displayed `query_name` in when referencing scheduled queries for more consistent UI/UX.

* Query action added for observers on host vitals page.

* Added `server_settings.debug_host_ids` to gather more detailed information about what the specified hosts are sending to fleet.

* Allowed deeper linking into the Fleet application by saving filters in URL parameters.

* Renamed Basic Tier to Premium Tier, and Core Tier to Free Tier.

* Improved vulnerability detection compatibility with database configurations.

* MariaDB compatibility fixes: add explicit foreign key constraint and on cascade delete for host_software to allow for hosts with software to be deleted.

* Fixed migration that was incompatible with MySQL primary key requirements (default on DigitalOcean MySQL 5.8).

* Added 30 second SMTP timeout for mail configuration.

* Fixed display of platform Labels on manage hosts page

* Fixed a bug recording scheduled query statistics.

* When a label is removed, ignore query executions for that label.

* Added fleet serve config to change the redis connection timeout and keep alive interval.

* Removed hardcoded limits in label searches when targeting queries.

* Allow host users to be readded.

* Moved email template images from github to fleetdm.com.

* Fixed bug rendering CPU in host vitals.

* Updated the schema for host_users to allow for bulk inserts without locking, and allow for users without unique uid.

* When using dynamic vulnerability processing node, try to create the vulnerability.databases-path.

* Fixed `fleetctl get host <hostname>` to properly output JSON when the command line flag is supplied i.e `fleetctl get host --json foobar`

## Fleet 4.2.4 (Sept 2, 2021)

* Fixed a bug in which live queries would fail for deployments that use Redis Cluster.

* Fixed a bug in which some new Fleet deployments don't include the default global agent options. Documentation for global and team agent options can be found [here](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options).

* Improved how a host's `users` are stored in MySQL to prevent deadlocks. This information is available in the "Users" table on each host's **Host details** page and in the `GET /api/v1/fleet/hosts/{id}` API route.

## Fleet 4.2.3 (Aug 23, 2021)

* Added ability to troubleshoot connection issues with the `fleetctl debug connection` command.

* Improved compatibility with MySQL variants (MariaDB, Aurora, etc.) by removing usage of JSON_ARRAYAGG.

* Fixed bug in which live queries would stop returning results if more than 5 seconds goes by without a result. This bug was introduced in 4.2.1.

* Eliminated double-logging of IP addresses in osquery endpoints.

* Update host details after transferring a host on the details page.

* Logged errors in osquery endpoints to improve debugging.

## Fleet 4.2.2 (Aug 18, 2021)

* Added a new built in label "All Linux" to target all hosts that run any linux flavor.

* Allowed finer grained configuration of the vulnerability processing capabilities.

* Fixed performance issues when updating pack contents.

* Fixed a build issue that caused external network access to panic in certain Linux distros (Ubuntu).

* Fixed rendering of checkboxes in UI when modals appear.

* Orbit: synced critical file writes to disk.

* Added "-o" flag to fleetctl convert command to ensure consistent output rather than relying on shell redirection (this was causing issues with file encodings).

* Fixed table column wrapping for manage queries page.

* Fixed wrapping in Label pills.

* Side panels in UI have a fresher look, Teams/Roles UI greyed out conditionally.

* Improved sorting in UI tables.

* Improved detection of CentOS in label membership.

## Fleet 4.2.1 (Aug 14, 2021)

* Fixed a database issue with MariaDB 10.5.4.

* Displayed updated team name after edit.

* When a connection from a live query websocket is closed, Fleet now timeouts the receive and handles the different cases correctly to not hold the connection to Redis.

* Added read live query results from Redis in a thread safe manner.

* Allows observers and maintainers to refetch a host in a team they belong to.

## Fleet 4.2.0 (Aug 11, 2021)

* Added the ability to simultaneously filter hosts by status (`online`, `offline`, `new`, `mia`) and by label on the **Hosts** page.

* Added the ability to filter hosts by team in the Fleet UI, fleetctl CLI tool, and Fleet API. *Available for Fleet Basic customers*.

* Added the ability to create a Team schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1). *Available for Fleet Basic customers*.

* Added Beta Vulnerable software feature which surfaces vulnerable software on the **Host details** page and the `GET /api/v1/fleet/hosts/{id}` API route. For information on how to configure the Vulnerable software feature and how exactly Fleet processes vulnerabilities, check out the [Vulnerability processing documentation](https://fleetdm.com/docs/using-fleet/vulnerability-processing#vulnerability-processing).

* Added the ability to see which logging destination is configured for Fleet in the Fleet UI. To see this information, head to the **Schedule** page and then select "Schedule a query." Configured logging destination information is also available in the `GET api/v1/fleet/config` API route.

* Improved the `fleetctl preview` experience by downloading Fleet's standard query library and loading the queries into the Fleet UI.

* Improved the user interface for the **Packs** page and **Queries** page in the Fleet UI.

* Added the ability to modify scheduled queries in your Schedule in Fleet. The Schedule feature was released in Fleet 4.1.0. For more information on the new Schedule feature, check out the [Fleet 4.1.0 release blog post](https://blog.fleetdm.com/fleet-4-1-0-57dfa25e89c1).

* Added the ability to disable the Users feature in Fleet by setting the new `enable_host_users` key to `true` in the `config` yaml, configuration file. For documentation on using configuration files in yaml syntax, check out the [Using yaml files in Fleet](https://fleetdm.com/docs/using-fleet/configuration-files#using-yaml-files-in-fleet) documentation.

* Improved performance of the Software inventory feature. Software inventory is currently under a feature flag. To enable this feature flag, check out the [feature flag documentation](https://fleetdm.com/docs/deploying/configuration#feature-flags).

* Improved performance of inserting `pack_stats` in the database. The `pack_stats` information is used to display "Frequency" and "Last run" information for a specific host's scheduled queries. You can find this information on the **Host details** page.

* Improved Fleet server logging so that it is more uniform.

* Fixed a bug in which a user with the Observer role was unable to run a live query.

* Fixed a bug that prevented the new **Home** page from being displayed in some Fleet instances.

* Fixed a bug that prevented accurate sorting issues across multiple pages on the **Hosts** page.

## Fleet 4.1.0 (Jul 26, 2021)

The primary additions in Fleet 4.1.0 are the new Schedule and Activity feed features.

Scheduled lets you add queries which are executed on your devices at regular intervals without having to understand or configure osquery query packs. For experienced Fleet and osquery users, the ability to create new, and modify existing, query packs is still available in the Fleet UI and fleetctl command-line tool. To reach the **Packs** page in the Fleet UI, head to **Schedule > Advanced**.

Activity feed adds the ability to observe when, and by whom, queries are changes, packs are created, live queries are run, and more. The Activity feed feature is located on the new Home page in the Fleet UI. Select the logo in the top right corner of the Fleet UI to navigate to the new **Home** page.

### New features breakdown

* Added ability to create teams and update their respective agent options and enroll secrets using the new `teams` yaml document and fleetctl. Available in Fleet Basic.

* Added a new **Home** page to the Fleet UI. The **Home** page presents a breakdown of the enrolled hosts by operating system.

* Added a "Users" table on the **Host details** page. The `username` information displayed in the "Users" table, as well as the `uid`, `type`, and `groupname` are available in the Fleet REST API via the `/api/v1/fleet/hosts/{id}` API route.

* Added ability to create a user without an invitation. You can now create a new user by heading to **Settings > Users**, selecting "Create user," and then choosing the "Create user" option.

* Added ability to search and sort installed software items in the "Software" table on the **Host details** page. 

* Added ability to delete a user from Fleet using a new `fleetctl user delete` command.

* Added ability to retrieve hosts' `status`, `display_text`, and `labels` using the `fleetctl get hosts` command.

* Added a new `user_roles` yaml document that allows users to manage user roles via fleetctl. Available in Fleet Basic.

* Changed default ordering of the "Hosts" table in the Fleet UI to ascending order (A-Z).

* Improved performance of the Software inventory feature by reducing the amount of inserts and deletes are done in the database when updating each host's
software inventory.

* Removed YUM and APT sources from Software inventory.

* Fixed an issue in which disabling SSO at the organization level would not disable SSO for all users.

* Fixed an issue with data migrations in which enroll secrets are duplicated after the `name` column was removed from the `enroll_secrets` table.

* Fixed an issue in which it was not possible to clear host settings by applying the `config` yaml document. This allows users to successfully remove the `additional_queries` property after adding it.

* Fixed printing of failed record count in AWS Kinesis/Firehose logging plugins.

* Fixed compatibility with GCP Memorystore Redis due to missing CLUSTER command.


## Fleet 4.0.1 (Jul 01, 2021)

* Fixed an issue in which migrations failed on MariaDB MySQL.

* Allowed `http` to be used when configuring `fleetctl` for `localhost`.

* Fixed a bug in which Team information was missing for hosts looked up by Label. 

## Fleet 4.0.0 (Jun 29, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/9-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/10-Teams.md).

### New features breakdown

* Added the ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Added the ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Added the ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Added Redis cluster support. Available in Fleet Core.

* Fixed a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. The following sections call out changes to consider when upgrading to Fleet 4.0.0:

* The structure of Fleet's `.tar.gz` and `.zip` release archives have changed slightly. Deployments that use the binary artifacts may need to update scripts or tooling. The `fleetdm/fleet` Docker container maintains the same API.

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Changeed your SAML SSO URI to use fleet instead of kolide . This is due to the changes to Fleet's API routes outlined in the section above.

* Changeed configuration option `server_tlsprofile` to `server_tls_compatibility`. This options previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity. `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration.

* The `username` artifact has been removed in favor of the more recognizable `name` (Full name). As a result the `email` artifact is now used for uniqueness in Fleet. Upon upgrading to Fleet 4.0.0, existing users will have the `name` field populated with `username`. SAML users may need to update their username mapping to match user emails.

* As of Fleet 4.0.0, Fleet Device Management Inc. periodically collects anonymous information about your instance. Sending usage statistics is turned off by default for users upgrading from a previous version of Fleet. Read more about the exact information collected [here](https://github.com/fleetdm/fleet/blob/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/11-Usage-statistics.md).

## Fleet 4.0.0 RC3 (Jun 25, 2021)

Primarily teste the new release workflows. Relevant changelog will be updated for Fleet 4.0. 

## Fleet 4.0.0 RC2 (Jun 18, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access.

* Check out the [permissions documentation](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/8-Permissions.md) for a breakdown of the new user roles.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily act on consistent groups of hosts. 

* Read more about the Teams feature in [the documentation here](https://github.com/fleetdm/fleet/blob/5e40afa8ba28fc5cdee813dfca53b84ee0ee65cd/docs/1-Using-Fleet/9-Teams.md).

### New features breakdown

* Added the ability to define a user's access to features in Fleet by introducing the Admin, Maintainer, and Observer roles. Available in Fleet Core.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Teams: Added the ability to enroll hosts to one team using team specific enroll secrets.

* Teams: Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Teams: Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Teams: Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

* Added the ability to create an API-only user. API-only users cannot access the Fleet UI. These users can access all Fleet API endpoints and `fleetctl` features. Available in Fleet Core.

* Added Redis cluster support. Available in Fleet Core.

* Fixed a bug that prevented the columns chosen for the "Hosts" table from persisting after logging out of Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Use strictly `fleet` in Fleet's configuration, API routes, and environment variables. Users must update all usage of `kolide` in these items (deprecated since Fleet 3.8.0).

* Changed configuration option `server_tlsprofile` to `server_tls_compatability`. This option previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues


There are currently no known issues in this release. However, we recommend only upgrading to Fleet 4.0.0-rc2 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

## Fleet 4.0.0 RC1 (Jun 10, 2021)

The primary additions in Fleet 4.0.0 are the new Role-based access control (RBAC) and Teams features. 

RBAC adds the ability to define a user's access to information and features in Fleet. This way, more individuals in an organization can utilize Fleet with appropriate levels of access. Check out the [permissions documentation](https://fleetdm.com/docs/using-fleet/permissions) for a breakdown of the new user roles and their respective capabilities.

Teams adds the ability to separate hosts into exclusive groups. This way, users can easily observe and apply operations to consistent groups of hosts. Read more about the Teams feature in [the documentation here](https://fleetdm.com/docs/using-fleet/teams).

There are several known issues that will be fixed for the stable release of Fleet 4.0.0. Therefore, we recommend only upgrading to Fleet 4.0.0 RC1 for testing purposes. Please file a GitHub issue for any issues discovered when testing Fleet 4.0.0!

### New features breakdown

* Added the ability to define a user's access to information and features in Fleet by introducing the Admin, Maintainer, and Observer roles.

* Added the ability to separate hosts into exclusive groups with the Teams feature. The Teams feature is available for Fleet Basic customers. Check out the list below for the new functionality included with Teams:

* Added the ability to enroll hosts to one team using team specific enroll secrets.

* Added the ability to manually transfer hosts to a different team in the Fleet UI.

* Added the ability to apply unique agent options to each team. Note that "osquery options" have been renamed to "agent options."

* Added the ability to grant users access to one or more teams. This allows you to define a user's access to specific groups of hosts in Fleet.

### Upgrade plan

Fleet 4.0.0 is a major release and introduces several breaking changes and database migrations. 

* Used strictly `fleet` in Fleet's configuration, API routes, and environment variables. This means that you must update all usage of `kolide` in these items. The backwards compatibility introduced in Fleet 3.8.0 is no longer valid in Fleet 4.0.0.

* Changed configuration option `server_tlsprofile` to `server_tls_compatability`. This options previously had an inconsistent key name.

* Replaced the use of the `api/v1/fleet/spec/osquery/options` with `api/v1/fleet/config`. In Fleet 4.0.0, "osquery options" are now called "agent options." The new agent options are moved to the Fleet application config spec file and the `api/v1/fleet/config` API endpoint.

* Enrolled secrets no longer have "names" and are now either global or for a specific team. Hosts no longer store the “name” of the enroll secret that was used. Users that want to be able to segment hosts (for configuration, queries, etc.) based on the enrollment secret should use the Teams feature in Fleet Basic.

* `auth_jwt_key` and `auth_jwt_key_file` are no longer accepted as configuration. 

* JWT encoding is no longer used for session keys. Sessions now default to expiring in 4 hours of inactivity.

### Known issues

* Query packs cannot be targeted to teams.

## Fleet 3.13.0 (Jun 3, 2021)

* Improved performance of the `additional_queries` feature by moving `additional` query results into a separate table in the MySQL database. Please note that the `/api/v1/fleet/hosts` API endpoint now return only the requested `additional` columns. See documentation on the changes to the hosts API endpoint [here](https://github.com/fleetdm/fleet/blob/06b2e564e657492bfbc647e07eb49fd4efca5a03/docs/1-Using-Fleet/3-REST-API.md#list-hosts).

* Fixed a bug in which running a live query in the Fleet UI would return no results and the query would seem "hung" on a small number of devices.

* Improved viewing live query errors in the Fleet UI by including the “Errors” table in the full screen view.

* Improved `fleetctl preview` experience by adding the `fleetctl preview reset` and `fleetctl preview stop` commands to reset and stop simulated hosts running in Docker.

* Added several improvements to the Fleet UI including additional contrast on checkboxes and dropdown pills.

## Fleet 3.12.0 (May 19, 2021)

* Added scheduled queries to the _Host details_ page. Surface the "Name", "Description", "Frequency", and "Last run" information for each query in a pack that apply to a specific host.

* Improved the freshness of host vitals by adding the ability to "refetch" the data on the _Host details_ page.

* Added ability to copy log fields into Google Cloud Pub/Sub attributes. This allows users to use these values for subscription filters.

* Added ability to duplicate live query results in Redis. When the `redis_duplicate_results` configuration option is set to `true`, all live query results will be copied to an additional Redis Pub/Sub channel named LQDuplicate.

* Added ability to controls the server-side HTTP keepalive property. Turning off keepalives has helped reduce outstanding TCP connections in some deployments.

* Fixed an issue on the _Packs_ page in which Fleet would incorrectly handle the configured `server_url_prefix`.

## Fleet 3.11.0 (Apr 28, 2021)

* Improved Fleet performance by batch updating host seen time instead of updating synchronously. This improvement reduces MySQL CPU usage by ~33% with 4,000 simulated hosts and MySQL running in Docker.

* Added support for software inventory, introducing a list of installed software items on each host's respective _Host details_ page. This feature is flagged off by default (for now). Check out [the feature flag documentation for instructions on how to turn this feature on](https://fleetdm.com/docs/deploying/configuration#software-inventory).

* Added Windows support for `fleetctl` agent autoupdates. The `fleetctl updates` command provides the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Made running common queries more convenient by adding the ability to select a saved query directly from a host's respective _Host details_ page.

* Fixed an issue on the _Query_ page in which Fleet would override the CMD + L browser hotkey.

* Fixed an issue in which a host would display an unreasonable time in the "Last fetched" column.

## Fleet 3.10.1 (Apr 6, 2021)

* Fixed a frontend bug that prevented the "Pack" page and "Edit pack" page from rendering in the Fleet UI. This issue occurred when the `platform` key, in the requested pack's configuration, was set to any value other than `darwin`, `linux`, `windows`, or `all`.

## Fleet 3.10.0 (Mar 31, 2021)

* Added `fleetctl` agent auto-updates beta which introduces the ability to self-manage an agent update server. Available for Fleet Basic customers.

* Added option for Identity Provider-Initiated (IdP-initiated) Single Sign-On (SSO).

* Improved logging. All errors are logged regardless of log level, some non-errors are logged regardless of log level (agent enrollments, runs of live queries etc.), and all other non-errors are logged on debug level.

* Improved login resilience by adding rate-limiting to login and password reset attempts and preventing user enumeration.

* Added Fleet version and Go version in the My Account page of the Fleet UI.

* Improved `fleetctl preview` to ensure the latest version of Fleet is fired up on every run. In addition, the Fleet UI is now accessible without having to click through browser security warning messages.

* Added prefer storing IPv4 addresses for host details.

## Fleet 3.9.0 (Mar 9, 2021)

* Added configurable host identifier to help with duplicate host enrollment scenarios. By default, Fleet's behavior does not change (it uses the identifier configured in osquery's `--host_identifier` flag), but for users with overlapping host UUIDs changing `--osquery_host_identifier` to `instance` may be helpful. 

* Made cool-down period for host enrollment configurable to control load on the database in scenarios in which hosts are using the same identifier. By default, the cooldown is off, reverting to the behavior of Fleet <=3.4.0. The cooldown can be enabled with `--osquery_enroll_cooldown`.

* Refreshed the Fleet UI with a new layout and horizontal navigation bar.

* Trimmed down the size of Fleet binaries.

* Improved handling of config_refresh values from osquery clients.

* Fixed an issue with IP addresses and host additional info dropping.

## Fleet 3.8.0 (Feb 25, 2021)

* Added search, sort, and column selection in the hosts dashboard.

* Added AWS Lambda logging plugin.

* Improved messaging about number of hosts responding to live query.

* Updated host listing API endpoints to support search.

* Added fixes to the `fleetctl preview` experience.

* Fixed `denylist` parameter in scheduled queries.

* Fixed an issue with errors table rendering on live query page.

* Deprecated `KOLIDE_` environment variable prefixes in favor of `FLEET_` prefixes. Deprecated prefixes continue to work and the Fleet server will log warnings if the deprecated variable names are used. 

* Deprecated `/api/v1/kolide` routes in favor of `/api/v1/fleet`. Deprecated routes continue to work and the Fleet server will log warnings if the deprecated routes are used. 

* Added Javascript source maps for development.

## Fleet 3.7.1 (Feb 3, 2021)

* Changed the default `--server_tls_compatibility` to `intermediate`. The new settings caused TLS connectivity issues for users in some environments. This new default is a more appropriate balance of security and compatibility, as recommended by Mozilla.

## Fleet 3.7.0 (Feb 3, 2021)

### This is a security release.

* **Security**: Fixed a vulnerability in which a malicious actor with a valid node key can send a badly formatted request that causes the Fleet server to exit, resulting in denial of service. See https://github.com/fleetdm/fleet/security/advisories/GHSA-xwh8-9p3f-3x45 and the linked content within that advisory.

* Added new Host details page which includes a rich view of a specific host’s attributes.

* Revealed live query errors in the Fleet UI and `fleetctl` to help target and diagnose hosts that fail.

* Added Helm chart to make it easier for users to deploy to Kubernetes.

* Added support for `denylist` parameter in scheduled queries.

* Added debug flag to `fleetctl` that enables logging of HTTP requests and responses to stderr.

* Improved the `fleetctl preview` experience to include adding containerized osquery agents, displaying login information, creating a default directory, and checking for Docker daemon status.

* Added improved error handling in host enrollment to make debugging issues with the enrollment process easier.

* Upgraded TLS compatibility settings to match Mozilla.

* Added comments in generated flagfile to add clarity to different features being configured.

* Fixed a bug in Fleet UI that allowed user to edit a scheduled query after it had been deleted from a pack.


## Fleet 3.6.0 (Jan 7, 2021)

* Added the option to set up an S3 bucket as the storage backend for file carving.

* Built Docker container with Fleet running as non-root user.

* Added support to read in the MySQL password and JWT key from a file.

* Improved the `fleetctl preview` experience by automatically completing the setup process and configuring fleetctl for users.

* Restructured the documentation into three top-level sections titled "Using Fleet," "Deployment," and "Contribution."

* Fixed a bug that allowed hosts to enroll with an empty enroll secret in new installations before setup was completed.

* Fixed a bug that made the query editor render strangely in Safari.

## Fleet 3.5.1 (Dec 14, 2020)

### This is a security release.

* **Security**: Introduced XML validation library to mitigate Go stdlib XML parsing vulnerability effecting SSO login. See https://github.com/fleetdm/fleet/security/advisories/GHSA-w3wf-cfx3-6gcx and the linked content within that advisory.

Follow up: Rotated `--auth_jwt_key` to invalidate existing sessions. Audit for suspicious activity in the Fleet server.

* **Security**: Prevents new queries from using the SQLite `ATTACH` command. This is a mitigation for the osquery vulnerability https://github.com/osquery/osquery/security/advisories/GHSA-4g56-2482-x7q8.

Follow up: Audit existing saved queries and logs of live query executions for possible malicious use of `ATTACH`. Upgrade osquery to 4.6.0 to prevent `ATTACH` queries from executing.

* Update icons and fix hosts dashboard for wide screen sizes.

## Fleet 3.5.0 (Dec 10, 2020)

* Refresh the Fleet UI with new colors, fonts, and Fleet logos.

* All releases going forward will have the fleectl.exe.zip on the release page.

* Added documentation for the authentication Fleet REST API endpoints.

* Added FAQ answers about the stress test results for Fleet, configuring labels, and resetting auth tokens.

* Fixed a performance issue users encountered when multiple hosts shared the same UUID by adding a one minute cooldown.

* Improved the `fleetctl preview` startup experience.

* Fixed a bug preventing the same query from being added to a scheduled pack more than once in the Fleet UI.


## Fleet 3.4.0 (Nov 18, 2020)

* Added NPM installer for `fleetctl`. Install via `npm install -g osquery-fleetctl`.

* Added `fleetctl preview` command to start a local test instance of the Fleet server with Docker.

* Added `fleetctl debug` commands and API endpoints for debugging server performance.

* Added additional_info_filters parameter to get hosts API endpoint for filtering returned additional_info.

* Updated package import paths from github.com/kolide/fleet to github.com/fleetdm/fleet.

* Added first of the Fleet REST API documentation.

* Added documentation on monitoring with Prometheus.

* Added documentation to FAQ for debugging database connection errors.

* Fixed fleetctl Windows compatibility issues.

* Fixed a bug preventing usernames from containing the @ symbol.

* Fixed a bug in 3.3.0 in which there was an unexpected database migration warning.

## Fleet 3.3.0 (Nov 05, 2020)

With this release, Fleet has moved to the new github.com/fleetdm/fleet
repository. Please follow changes and releases there.

* Added file carving functionality.

* Added `fleetctl user create` command.

* Added osquery options editor to admin pages in UI.

* Added `fleetctl query --pretty` option for pretty-printing query results. 

* Added ability to disable packs with `fleetctl apply`.

* Improved "Add New Host" dialog to walk the user step-by-step through host enrollment.

* Improved 500 error page by allowing display of the error.

* Added partial transition of branding away from "Kolide Fleet".

* Fixed an issue with case insensitive enroll secret and node key authentication.

* Fixed an issue with `fleetctl query --quiet` flag not actually suppressing output.


## Fleet 3.2.0 (Aug 08, 2020)

* Added `stdout` logging plugin.

* Added AWS `kinesis` logging plugin.

* Added compression option for `filesystem` logging plugin.

* Added support for Redis TLS connections.

* Added osquery host identifier to EnrollAgent logs.

* Added osquery version information to output of `fleetctl get hosts`.

* Added hostname to UI delete host confirmation modal.

* Updated osquery schema to 4.5.0.

* Updated osquery versions available in schedule query UI.

* Updated MySQL driver.

* Removed support for (previously deprecated) `old` TLS profile.

* Fixed cleanup of queries in bad state. This should resolve issues in which users experienced old live queries repeatedly returned to hosts. 

* Fixed output kind of `fleetctl get options`.

## Fleet 3.1.0 (Aug 06, 2020)

* Added configuration option to set Redis database (`--redis_database`).

* Added configuration option to set MySQL connection max lifetime (`--mysql_conn_max_lifetime`).

* Added support for printing a single enroll secret by name.

* Fixed bug with label_type in older fleetctl yaml syntax.

* Fixed bug with URL prefix and Edit Pack button. 

## Kolide Fleet 3.0.0 (Jul 23, 2020)

* Backend performance overhaul. The Fleet server can now handle hundreds of thousands of connected hosts.

* Pagination implemented in the web UI. This makes the UI usable for any host count supported by the backend.

* Added capability to collect "additional" information from hosts. Additional queries can be set to be updated along with the host detail queries. This additional information is returned by the API.

* Removed extraneous network interface information to optimize server performance. Users that require this information can use the additional queries functionality to retrieve it.

* Added "manual" labels implementation. Static labels can be set by providing a list of hostnames with `fleetctl`.

* Added JSON output for `fleetctl get` commands.

* Added `fleetctl get host` to retrieve details for a single host.

* Updated table schema for osquery 4.4.0.

* Added support for multiple enroll secrets.

* Logging verbosity reduced by default. Logs are now much less noisy.

* Fixed import of github.com/kolide/fleet Go packages for consumers outside of this repository.

## Kolide Fleet 2.6.0 (Mar 24, 2020)

* Added server logging for X-Forwarded-For header.

* Added `--osquery_detail_update_interval` to set interval of host detail updates.
  Set this (along with `--osquery_label_update_interval`) to a longer interval
  to reduce server load in large deployments.

* Fixed MySQL deadlock errors by adding retries and backoff to transactions.

## Kolide Fleet 2.5.0 (Jan 26, 2020)

* Added `fleetctl goquery` command to bring up the github.com/AbGuthrie/goquery CLI.

* Added ability to disable live queries in web UI and `fleetctl`.

* Added `--query-name` option to `fleetctl query`. This allows using the SQL from a saved query.

* Added `--mysql-protocol` flag to allow connection to MySQL by domain socket.

* Improved server logging. Add logging for creation of live queries. Add username information to logging for other endpoints.

* Allows CREATE queries in the web UI.

* Fixed a bug in which `fleetctl query` would exit before any results were returned when latency to the Fleet server was high.

* Fixed an error initializing the Fleet database when MySQL does not have event permissions.

* Deprecated "old" TLS profile.

## Kolide Fleet 2.4.0 (Nov 12, 2019)

* Added `--server_url_prefix` flag to configure a URL prefix to prepend on all Fleet URLs. This can be useful to run fleet behind a reverse-proxy on a hostname shared with other services.

* Added option to automatically expire hosts that have not checked in within a certain number of days. Configure this in the "Advanced Options" of "App Settings" in the browser UI.

* Added ability to search for hosts by UUID when targeting queries.

* Allows SAML IdP name to be as short as 4 characters.

## Kolide Fleet 2.3.0 (Aug 14, 2019)

### This is a security release.

* Security: Upgraded Go to 1.12.8 to fix CVE-2019-9512, CVE-2019-9514, and CVE-2019-14809.

* Added capability to export packs, labels, and queries as yaml in `fleetctl get` with the `--yaml` flag. Include queries with a pack using `--with-queries`.

* Modified email templates to load image assets from Github CDN rather than Fleet server (fixes broken images in emails when Fleet server is not accessible from email clients).

* Added warning in query UI when Redis is not functioning.

* Fixed minor bugs in frontend handling of scheduled queries.

* Minor styling changes to frontend.


## Kolide Fleet 2.2.0 (Jul 16, 2019)

* Added GCP PubSub logging plugin. Thanks to Michael Samuel for adding this capability.

* Improved escaping for target search in live query interface. It is now easier to target hosts with + and * characters in the name.

* Server and browser performance improved to reduced loading of hosts in frontend. Host status will only update on page load when over 100 hosts are present.

* Utilized details sent by osquery in enrollment request to more quickly display details of new hosts. Also fixes a bug in which hosts could not complete enrollment if certain platform-dependent options were used.

* Fixed a bug in which the default query runs after targets are edited.

## Kolide Fleet 2.1.2 (May 30, 2019)

* Prevented sending of SMTP credentials over insecure connection

* Added prefix generated SAML IDs with 'id' (improves compatibility with some IdPs)

## Kolide Fleet 2.1.1 (Apr 25, 2019)

* Automatically pulls AWS STS credentials for Firehose logging if they are not specified in config.

* Fixed bug in which log output did not include newlines separating characters.

* Fixed bug in which the default live query was run when navigating to a query by URL.

* Updated logic for setting primary NIC to ignore link-local or loopback interfaces.

* Disabled editing of logged in user email in admin panel (instead, use the "Account Settings" menu in top left).

* Fixed a panic resulting from an invalid config file path.

## Kolide Fleet 2.1.0 (Apr 9, 2019)

* Added capability to log osquery status and results to AWS Firehose. Note that this deprecated some existing logging configuration (`--osquery_status_log_file` and `--osquery_result_log_file`). Existing configurations will continue to work, but will be removed at some point.

* Automatically cleans up "incoming hosts" that do not complete enrollment.

* Fixed bug with SSO requests that caused issues with some IdPs.

* Hid built-in platform labels that have no hosts.

* Fixed references to Fleet documentation in emails.

* Minor improvements to UI in places where editing objects is disabled.

## Kolide Fleet 2.0.2 (Jan 17, 2019)

* Improved performance of `fleetctl query` with high host counts.

* Added `fleetctl get hosts` command to retrieve a list of enrolled hosts.

* Added support for Login SMTP authentication method (Used by Office365).

* Added `--timeout` flag to `fleetctl query`.

* Added query editor support for control-return shortcut to run query.

* Allowed preselection of hosts by UUID in query page URL parameters.

* Allowed username to be specified in `fleetctl setup`. Default behavior remains to use email as username.

* Fixed conversion of integers in `fleetctl convert`.

* Upgraded major Javascript dependencies.

* Fixed a bug in which query name had to be specified in pack yaml.

## Kolide Fleet 2.0.1 (Nov 26, 2018)

* Fixed a bug in which deleted queries appeared in pack specs returned by fleetctl.

* Fixed a bug getting entities with spaces in the name.

## Kolide Fleet 2.0.0 (Oct 16, 2018)

* Stable release of Fleet 2.0.

* Supports custom certificate authorities in fleetctl client.

* Added support for MySQL 8 authentication methods.

* Allows INSERT queries in editor.

* Updated UI styles.

* Fixed a bug causing migration errors in certain environments.

See changelogs for release candidates below to get full differences from 1.0.9
to 2.0.0.

## Kolide Fleet 2.0.0 RC5 (Sep 18, 2018)

* Fixed a security vulnerability that would allow a non-admin user to elevate privileges to admin level.

* Fixed a security vulnerability that would allow a non-admin user to modify other user's details.

* Reduced the information that could be gained by an admin user trying to port scan the network through the SMTP configuration.

* Refactored and add testing to authorization code.

## Kolide Fleet 2.0.0 RC4 (August 14, 2018)

* Exposed the API token (to be used with fleetctl) in the UI.

* Updated autocompletion values in the query editor.

* Fixed a longstanding bug that caused pack targets to sometimes update incorrectly in the UI.

* Fixed a bug that prevented deletion of labels in the UI.

* Fixed error some users encountered when migrating packs (due to deleted scheduled queries).

* Updated favicon and UI styles.

* Handled newlines in pack JSON with `fleetctl convert`.

* Improved UX of fleetctl tool.

* Fixed a bug in which the UI displayed the incorrect logging type for scheduled queries.

* Added support for SAML providers with whitespace in the X509 certificate.

* Fixed targeting of packs to individual hosts in the UI.

## Kolide Fleet 2.0.0 RC3 (June 21, 2018)

* Fixed a bug where duplicate queries were being created in the same pack but only one was ever delivered to osquery. A migration was added to delete duplicate queries in packs created by the UI.
  * It is possible to schedule the same query with different options in one pack, but only via the CLI.
  * If you thought you were relying on this functionality via the UI, note that duplicate queries will be deleted when you run migrations as apart of a cleanup fix. Please check your configurations and make sure to create any double-scheduled queries via the CLI moving forward.

* Fixed a bug in which packs created in UI could not be loaded by fleetctl.

* Fixed a bug where deleting a query would not delete it from the packs that the query was scheduled in.

## Kolide Fleet 2.0.0 RC2 (June 18, 2018)

* Fixed errors when creating and modifying packs, queries and labels in UI.

* Fixed an issue with the schema of returned config JSON.

* Handled newlines when converting query packs with fleetctl convert.

* Added last seen time hover tooltip in Fleet UI.

* Fixed a null pointer error when live querying via fleetctl.

* Explicitly set timezone in MySQL connection (improves timestamp consistency).

* Allowed native password auth for MySQL (improves compatibility with Amazon RDS).

## Kolide Fleet 2.0.0 (currently preparing for release)

The primary new addition in Fleet 2 is the new `fleetctl` CLI and file-format, which dramatically increases the flexibility and control that administrators have over their osquery deployment. The CLI and the file format are documented [in the Fleet documentation](https://fleetdm.com/docs/using-fleet/fleetctl-cli).

### New Features

* New `fleetctl` CLI for managing your entire osquery workflow via CLI, API, and source controlled files!
  * You can use `fleetctl` to manage osquery packs, queries, labels, and configuration.

* In addition to the CLI, Fleet 2.0.0 introduces a new file format for articulating labels, queries, packs, options, etc. This format is designed for composability, enabling more effective sharing and re-use of intelligence.

```yaml
apiVersion: v1
kind: query
spec:
  name: pending_updates
  query: >
    select value
    from plist
    where
      path = "/Library/Preferences/ManagedInstalls.plist" and
      key = "PendingUpdateCount" and
      value > "0";
```

* Run live osquery queries against arbitrary subsets of your infrastructure via the `fleetctl query` command.

* Use `fleetctl setup`, `fleetctl login`, and `fleetctl logout` to manage the authentication life-cycle via the CLI.

* Use `fleetctl get`, `fleetctl apply`, and `fleetctl delete` to manage the state of your Fleet data.

* Manage any osquery option you want and set platform-specific overrides with the `fleetctl` CLI and file format.

### Upgrade Plan

* Managing osquery options via the UI has been removed in favor of the more flexible solution provided by the CLI. If you have customized your osquery options with Fleet, there is [a database migration](./server/datastore/mysql/migrations/data/20171212182458_MigrateOsqueryOptions.go) which will port your existing data into the new format when you run `fleet prepare db`. To download your osquery options after migrating your database, run `fleetctl get options > options.yaml`. Further modifications to your options should occur in this file and it should be applied with `fleetctl apply -f ./options.yaml`.

## Kolide Fleet 1.0.8 (May 3, 2018)

* Osquery 3.0+ compatibility!

* Included RFC822 From header in emails (for email authentication)

## Kolide Fleet 1.0.7 (Mar 30, 2018)

* Now supports FileAccesses in FIM configuration.

* Now populates network interfaces on windows hosts in host view.

* Added flags for configuring MySQL connection pooling limits.

* Fixed bug in which shard and removed keys are dropped in query packs returned to osquery clients.

* Fixed handling of status logs with unexpected fields.

## Kolide Fleet 1.0.6 (Dec 4, 2017)

* Added remote IP in the logs for all osqueryd/launcher requests. (#1653)

* Fixed bugs that caused logs to sometimes be omitted from the logwriter. (#1636, #1617)

* Fixed a bug where request bodies were not being explicitly closed. (#1613)

* Fixed a bug where SAML client would create too many HTTP connections. (#1587)

* Fixed bug in which default query was run instead of entered query. (#1611)

* Added pagination to the Host browser pages for increased performance. (#1594)

* Fixed bug rendering hosts when clock speed cannot be parsed. (#1604)

## Kolide Fleet 1.0.5 (Oct 17, 2017)

* Renamed the binary from kolide to fleet.

* Added support for Kolide Launcher managed osquery nodes.

* Removed license requirements.

* Updated documentation link in the sidebar to point to public GitHub documentation.

* Added FIM support.

* Title on query page correctly reflects new or edit mode.

* Fixed issue on new query page where last query would be submitted instead of current.

* Fixed issue where user menu did not work on Firefox browser.

* Fixed issue cause SSO to fail for ADFS.

* Fixed issue validating signatures in nested SAML assertions..

## Kolide 1.0.4 (Jun 1, 2017)

* Added feature that allows users to import existing Osquery configuration files using the [configimporter](https://github.com/kolide/configimporter) utility.

* Added support for Osquery decorators.

* Added SAML single sign on support.

* Improved online status detection.

  The Kolide server now tracks the `distributed_interval` and `config_tls_refresh` values for each individual host (these can be different if they are set via flagfile and not through Kolide), to ensure that online status is represented as accurately as possible.

* Kolide server now requires `--auth_jwt_key` to be specified at startup.

  If no JWT key is provided by the user, the server will print a new suggested random JWT key for use.

* Fixed bug in which deleted packs were still displayed on the query sidebar.

* Fixed rounding error when showing % of online hosts.

* Removed --app_token_key flag.

* Fixed issue where heavily loaded database caused host authentication failures.

* Fixed issue where osquery sends empty strings for integer values in log results.

## Kolide 1.0.3 (April 3, 2017)

* Log rotation is no longer the default setting for Osquery status and results logs. To enable log rotation use the `--osquery_enable_log_rotation` flag.

* Added a debug endpoint for collecting performance statistics and profiles.

  When `kolide serve --debug` is used, additional handlers will be started to provide access to profiling tools. These endpoints are authenticated with a randomly generated token that is printed to the Kolide logs at startup. These profiling tools are not intended for general use, but they may be useful when providing performance-related bug reports to the Kolide developers.

* Added a workaround for CentOS6 detection.

  Osquery 2.3.2 incorrectly reports an empty value for `platform` on CentOS6 hosts. We added a workaround to properly detect platform in Kolide, and also [submitted a fix](https://github.com/facebook/osquery/pull/3071) to upstream osquery.

* Ensured hosts enroll in labels immediately even when `distributed_interval` is set to a long interval.

* Optimizations reduce the CPU and DB usage of the manage hosts page.

* Managed packs page now loads much quicker when a large number of hosts are enrolled.

* Fixed bug with the "Reset Options" button.

* Fixed 500 error resulting from saving unchanged options.

* Improved validation for SMTP settings.

* Added command line support for `modern`, `intermediate`, and `old` TLS configuration
profiles. The profile is set using the following command line argument.
```
--server_tls_compatibility=modern
```
See https://wiki.mozilla.org/Security/Server_Side_TLS for more information on the different profile options.

* The Options Configuration item in the sidebar is now only available to admin users.

  Previously this item was visible to non-admin users and if selected, a blank options page would be displayed since server side authorization constraints prevent regular users from viewing or changing options.

* Improved validation for the Kolide server URL supplied in setup and configuration.

* Fixed an issue importing osquery configurations with numeric values represented as strings in JSON.

## Kolide 1.0.2 (March 14, 2017)

* Fixed an issue adding additional targets when querying a host

* Shows loading spinner while newly added Host Details are saved

* Shows a generic computer icon when when referring to hosts with an unknown platform instead of the text "All"

* Kolide will now warn on startup if there are database migrations not yet completed.

* Kolide will prompt for confirmation before running database migrations.

  To disable this, use `kolide prepare db --no-prompt`.

* Kolide now supports emoji, so you can 🔥 to your heart's content.

* When setting the platform for a scheduled query, selecting "All" now clears individually selected platforms.

* Updated Host details cards UI

* Lowered HTTP timeout settings.

  In an effort to provide a more resilient web server, timeouts are more strictly enforced by the Kolide HTTP server (regardless of whether or not you're using the built-in TLS termination).

* Hardened TLS server settings.

  For customers using Kolide's built-in TLS server (if the `server.tls` configuration is `true`), the server was hardened to only accept modern cipher suites as recommended by [Mozilla](https://wiki.mozilla.org/Security/Server_Side_TLS#Modern_compatibility).

* Improve the mechanism used to calculate whether or not hosts are online.

  Previously, hosts were categorized as "online" if they had been seen within the past 30 minutes. To make the "online" status more representative of reality, hosts are marked "online" if the Kolide server has heard from them within two times the lowest polling interval as described by the Kolide-managed osquery configuration. For example, if you've configured osqueryd to check-in with Kolide every 10 seconds, only hosts that Kolide has heard from within the last 20 seconds will be marked "online".

* Updated Host details cards UI

* Added support for rotating the osquery status and result log files by sending a SIGHUP signal to the kolide process.

* Fixed Distributed Query compatibility with load balancers and Safari.

  Customers running Kolide behind a web balancer lacking support for websockets were unable to use the distributed query feature. Also, in certain circumstances, Safari users with a self-signed cert for Kolide would receive an error. This release add a fallback mechanism from websockets using SockJS for improved compatibility.

* Fixed issue with Distributed Query Pack results full screen feature that broke the browser scrolling abilities.

* Fixed bug in which host counts in the sidebar did not match up with displayed hosts.

## Kolide 1.0.1 (February 27, 2017)

* Fixed an issue that prevented users from replacing deleted labels with a new label of the same name.

* Improved the reliability of IP and MAC address data in the host cards and table.

* Added full screen support for distributed query results.

* Enabled users to double click on queries and packs in a table to see their details.

* Reprompted for a password when a user attempts to change their email address.

* Automatically decorates the status and result logs with the host's UUID and hostname.

* Fixed an issue where Kolide users on Safari were unable to delete queries or packs.

* Improved platform detection accuracy.

  Previously Kolide was determining platform based on the OS of the system osquery was built on instead of the OS it was running on. Please note: Offline hosts may continue to report an erroneous platform until they check-in with Kolide.

* Fixed bugs where query links in the pack sidebar pointed to the wrong queries.

* Improved MySQL compatibility with stricter configurations.

* Allows users to edit the name and description of host labels.

* Added basic table autocompletion when typing in the query composer.

* Now support MySQL client certificate authentication. More details can be found in the [Configuring the Fleet binary docs](./docs/infrastructure/configuring-the-fleet-binary.md).

* Improved security for user-initiated email address changes.

  This improvement ensures that only users who own an email address and are logged in as the user who initiated the change can confirm the new email.

  Previously it was possible for Administrators to also confirm these changes by clicking the confirmation link.

* Fixed an issue where the setup form rejects passwords with certain characters.

  This change resolves an issue where certain special characters like "." where rejected by the client-side JS that controls the setup form.

* Now automatically logs in the user once initial setup is completed.
