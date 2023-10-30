# Fleet 4.36.0 | Saved and scheduled queries merge.

![Fleet 4.36.0](../website/assets/images/articles/fleet-4.36.0-1600x900@2x.png)

Fleet 4.36.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Merging scheduled and saved queries for enhanced usability


### Merging scheduled and saved queries for enhanced usability

In Fleet 4.36.0, we have aligned the functionality of scheduled queries with saved queries, reflecting our commitment to 🟢 Results and 🔵 Objectivity. [Scheduled queries](https://fleetdm.com/docs/get-started/faq#what-happened-to-the-schedule-page) have been merged with saved queries, allowing users to create a query and schedule it at specific intervals or save it for ad-hoc use. As part of the migration, scheduled queries will be copied to each team, and timestamps will be added to query names to prevent naming conflicts. This change aims to enhance usability and configurability by treating queries in the same manner as policies, with team-by-team management.

In line with our values of 🟣 Openness and simplicity, we are also moving towards deprecating the concept of packs, as the new merged query concept has rendered them unnecessary. The main advantages of these changes are the streamlined user interface for query management, increased flexibility in query configurations, and the ability to manage queries on a team-by-team basis. By making Fleet easier to navigate, we are looking to support our users in achieving their goals while fostering a culture of continual improvement.

## New features, improvements, and bug fixes

* Added the `fleetctl upgrade-packs` command to migrate 2017 packs to the new combined schedule and query concept.
* Updated `fleetctl convert` to convert packs to the new combined schedule and query format.
* Updated the `POST /mdm/apple/profiles/match` endpoint to set the bootstrap package and enable end user authentication settings for each new team created via the endpoint to the corresponding values specified in the app config as of the time the applicable team is created.
* Added enroll secret for a new team created with `fleetctl apply` if none is provided.
* Improved SQL autocomplete with dynamic column, table names, and shown metadata.
* Cleaned up styling around table search bars.
* Updated MDM profile verification to fix issue where profiles were marked as failed when a host is transferred to a newly created team that has an identical profile as an older team.
* Added Windows MDM automatic enrollment setup pages to Fleet UI.
* (Beta) Allowed configuring Windows MDM certificates using their contents.
* Updated the icons on the dashboard to new grey designs.
* Ensured DEP profiles are assigned even for devices that already exist and have an op type = "modified".
* Disabled save button for invalid query or policy SQL & missing name.
* Users with no global or team role cannot access the UI.
* Text cells truncate with ellipses if longer than column width.


#### Bug Fixes:



* Fixed styling issue of the active settings tab.
* Fixed response status code to 403 when a user cannot change their password either because they were not requested to by the admin or they have Single-Sign-On (SSO) enabled.
* Fixed issues with end user migration flow.
* Fixed login form cut off when viewport is too short.
* Fixed bug where `os_version` endpoint returned 404 for `no teams` on controls page.
* Fixed delays applying profiles when the Puppet module is used in distributed scenarios.
* Fixed a style issue in the filter host by status dropdown.
* Fixed an issue when a user with `gitops` role was used to validate a configuration with `fleetctl apply --dry-run`.
* Fixed jumping text on the host page label filter dropdown at low viewport widths.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.36.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-08-18">
<meta name="articleTitle" value="Fleet 4.36.0 | Saved and scheduled queries merge.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.36.0-1600x900@2x.png">
