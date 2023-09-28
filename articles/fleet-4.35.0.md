# Fleet 4.35.0 | Improvements and bug fixes.

![Fleet 4.35.0](../website/assets/images/articles/fleet-4.35.0-1600x900@2x.png)

Fleet 4.35.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

Fleet took the opportunity for this release to focus on improvements and bug fixes.

## New features, improvements, and bug fixes

* Combined the query and schedule features to provide a single interface for creating, scheduling, and tweaking queries at the global and team level.

* Merged all functionality of the [schedule page](https://fleetdm.com/docs/get-started/faq#what-happened-to-the-schedule-page "What happened to the schedule page?") into the queries page.

* Updated the save query modal to include scheduling-related fields.

* Updated queries table schema to allow storing scheduling information and configuration in the queries table.

* Users now able to manage scheduled queries using automations modal.

* The `osquery/config` endpoint now includes scheduled queries for the host's team stored in the `queries` table.

* Query editor now includes frequency and other advanced options.

* Updated macOS MDM setup UI in Fleet UI.

* Changed how team assignment works for the Puppet module, for more details see the [README](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/README.md).

* Allow the Puppet module to read different Fleet URL/token combinations for different environments.

* Updated server logging for webhook requests to mask URL query values if the query param name includes "secret", "token", "key", and "password".

* Added support for Azure JWT tokens.

* Set `DeferForceAtUserLoginMaxBypassAttempts` to `1` in the default FileVault profile installed by Fleet.

* Added dark and light mode logo uploads and show the appropriate logo to the macOS MDM migration flow.

* Added MSI installer deployment support through MS-MDM.

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

* Fixed an issue where Orbit repeatedly tries to launch Nudge in the event of a launch error.

* Fixed Observer + should be able to run any query by clicking create new query.

* Fixed the styling of the initial setup flow.

* Fixed URL used to check Gravatar network availability.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.35.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-08-01">
<meta name="articleTitle" value="Fleet 4.35.0 | Improvements and bug fixes.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.35.0-1600x900@2x.png">
