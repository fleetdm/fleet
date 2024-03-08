# Fleet 4.42.0 | Query performance reporting, host targeting improvements.

![Fleet 4.42.0](../website/assets/images/articles/fleet-4.42.0-1600x900@2x.png)

Fleet 4.42.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.40.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Query results per-host
* Query performance reporting
* Human-endpoint mapping
* Consolidated software view
* Target hosts by serial number

### Query results per-host

Query results are now cached on a per-host basis. Cached query results are designed to significantly aid Fleet users, particularly when investigating the state of a currently offline host. By caching query results, administrators can access and review the latest recorded state of any individual host, even when that host is not actively connected to a network.

This addition benefits IT and security teams, facilitating a more efficient troubleshooting process. Custom data relevant to each host can be added to the Host details page, streamlining diagnostic and issue resolution tasks. This enhancement aligns with Fleet's commitment to providing user-friendly and efficient tools, enhancing the overall device management and security capabilities within the Fleet environment.

### Query performance reporting

Fleet continues to enhance query management: the ability to gauge the performance impact of running queries. Live queries will now gather and display statistics on their impact on system resources. These statistics will be accessible for saved queries located in the Queries tab.

After executing a live query, users can conveniently view the updated performance impact statistics in the Queries pages. This allows for a more informed assessment of how each query affects system resources. Additionally, the resilience of query statistics has been improved; they will no longer reset after a host or agent reboot. This ensures continuity and reliability in monitoring performance over time.

In line with maintaining a streamlined and relevant dataset, statistics for a query will be automatically deleted when the query itself is deleted. The impact of each query will be categorized and displayed in a new "Performance impact" column, with labels such as "Minimal," "Considerable," and "Excessive," offering a clear, at-a-glance understanding of each query's system load. This feature aligns with Fleet's values of openness and objectivity, as it provides transparent and quantifiable data on the impact of queries, empowering users to make more informed decisions.


### Human-endpoint mapping

Fleet continues to focus on enhancing user management capabilities. This new feature will allow administrators to look up devices associated with specific users through the email addresses used with their Identity Provider (IdP), currently supported exclusively on macOS hosts. This addition will benefit organizations utilizing identity management systems, as it streamlines linking users to their respective devices, thereby improving administrative efficiency and device management accuracy. This feature aligns with Fleet's commitment to providing user-friendly, efficient tools for IT and security teams.

Stay tuned for upcoming updates and more inclusive features as we continue to enhance Fleet's versatility across different operating systems. To glimpse what's in store, please follow our progress on this story in the current sprint: [Human-endpoint mapping expansion](https://github.com/fleetdm/fleet/issues/15057).


### Consolidated software view

As part of Fleet's commitment to enhancing user experience and administrative efficiency, the organization of software listings in the UI is being updated to streamline the Software page for IT administrators. Admins can now view a consolidated list of software installed across their fleet, organized by software title. This organization method will enable admins to easily identify the most popular software used in their environment, irrespective of the versions installed. As part of this initial release, popular applications will be represented with icons. What applications would you like icons for? Comment on this [issue](https://www.google.com/url?q=https://github.com/fleetdm/fleet/issues/14674&sa=D&source=docs&ust=1703174293969231&usg=AOvVaw1rsTNYKMBqvgUiUF7y3pP_), or contribute by making pull requests for [icons](https://github.com/fleetdm/fleet/blob/main/frontend/pages/SoftwarePage/components/icons/index.ts#L19-L31) of your favorite apps, fostering a collaborative development environment.


### Target hosts by serial number

Administrators now have the capability to target hosts directly by their serial numbers through the `fleetctl` command line. When using `fleetctl run-script` and `fleetctl mdm run-command` commands, identifying and managing specific hosts within the Fleet environment is streamlined. By enabling direct targeting of hosts by serial number, Fleet enhances the precision and efficiency of administrative tasks, aligning with its commitment to providing powerful and user-friendly device management tools. This addition is particularly useful for scenarios where quick identification and action on individual hosts are required, improving overall workflow efficiency.


## Changes

* **Endpoint operations**:
  - Added `fleet/device/{token}/ping` endpoint for agent token checks.
  - Added `GET /hosts/{id}/health` endpoint for host health data.
  - Added `--host-identifier` option to fleetd for enrolling with a random identifier.
  - Added capability to look up hosts based on IdP email.
  - Updated manage hosts UI to filter hosts by `software_version_id` and `software_title_id`.
  - Added ability to filter hosts by `software_version_id` and `software_title_id` in various endpoints.
  - **NOTE:**: Database migrations may take up to five minutes to complete based on number of software items. 
  - Live queries now collect and display updated stats.
  - Live query stats are cleared when query SQL is modified.
  - Added UI features to incorporate new live query stats.
  - Improved host query reports and host detail query tab UI.
  - Added firehose delivery addon update for improved data handling.

* **Vulnerability management**:
  - Added `GET software/versions` and `GET software/versions/{id}` endpoints for software version management.
  - Deprecated `GET software` and `GET software/{id}` endpoints.
  - Added new software pages in Fleet UI, including software titles and versions.
  - Resolved scan error during OVAL vulnerability processing.

* **Device management (MDM)**:
  - Removed the `FLEET_DEV_MDM_ENABLED` feature flag for Windows MDM.
  - Enabled `fleetctl` to configure Windows MDM profiles for teams and "no team".
  - Added database tables to support the Windows profiles feature.
  - Added support to configure Windows OS updates requirements.
  - Introduced new MDM profile endpoints: `POST /mdm/profiles`, `DELETE /mdm/profiles/{id}`, `GET /mdm/profiles/{id}`, `GET /mdm/profiles`, `GET /mdm/profiles/summary`.
  - Added validation to disallow custom MDM profiles with certain names.
  - Added deployment of Windows OS updates settings to targeted hosts.
  - Changed the Apple profiles ID to a prefixed UUID format.
  - Enabled targeting hosts by serial number in `fleetctl run-script` and `fleetctl mdm run-command`.
  - Added UI for uploading, deleting, downloading, and viewing Windows custom MDM profiles.

## Bug fixes and improvements

  - Updated Go version to 1.21.5.
  - Query reports now only show results for hosts with user permissions.
  - Global observers can now see all queries regardless of the observerCanRun value.
  - Added whitespace rendering in policy descriptions and resolutions.
  - Added truncation to dropdown options in query tables documentation.
  - `POST /api/v1/fleet/scripts/run/sync` timeout now returns error code 408 instead of 504.
  - Fixed possible deadlocks in `software` data ingestion and `host_batteries` upsert.
  - Fixed button text wrapping in UI for Settings > Integrations > MDM.
  - Fixed a bug where opening a modal on the Users page reset the table to the first page.
  - Fixed a bug preventing label selection while the label search field was active.
  - Fixed issues with UI loading indicators and placeholder texts.
  - Fixed a fleetctl issue where running a query by name created a new query instead of using the existing one.
  - Fixed `installed_from_dep` in `mdm_enrolled` activity for DEP device re-enrollment.
  - Fixed a bug in line breaks affecting UI functionality.
  - Fixed Syncml cmd data support for raw data.
  - Added "copied!" message to the copy button on inputs.
  - Fixed an edge case where caching could lead to lost organization settings in multiple instance scenarios.
  - Fixed `GET /hosts/{id}/health` endpoint reporting.
  - Fixed validation bugs allowing `overrides.platform` field to be set to `null`.
  - Fixed an issue with policy counts showing 0 post-upgrade.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.42.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-12-21">
<meta name="articleTitle" value="Fleet 4.42.0 | Query performance reporting, host targeting improvements.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.42.0-1600x900@2x.png">
