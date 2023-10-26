# Fleet 4.39.0 | Sonoma support, script library, query reports.

![Fleet 4.39.0](../website/assets/images/articles/fleet-4.39.0-1600x900@2x.png)

Fleet 4.39.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.39.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* macOS Sonoma support
* Script library
* Scheduled query reports

### macOS Sonoma support

Fleet has incorporated support for Apple's latest macOS release, Sonoma. Recognizing the profound shifts in device management introduced with macOS Sonoma, Fleet is keenly working towards assimilating these changes in future updates. Upholding the values of Openness and Empathy, we're not just adapting to these innovations but are also inviting user feedback to ensure that we prioritize the features and enhancements most crucial to our users.

One of the standout features of macOS Sonoma is the introduction of declarative device management (DDM). DDM, an enhancement to the Mobile Device Management (MDM) protocol, is a paradigm shift, offering a more secure and efficient method for macOS device administration. For a deeper dive into DDM and its transformative potential, our dedicated article, "[Embracing the Future: Declarative Device Management](https://fleetdm.com/announcements/embracing-the-future-declarative-device-management)," provides comprehensive insights.

### Script library

_Available in Fleet Premium._

Fleet has enabled the storage and management of saved scripts for macOS. Within the Fleet web user interface, command-line interface(CLI), and API, administrators and maintainers can add, delete, and download a saved script, whether affiliated with a global team or a specific team. 

Beyond just management, authorized users have the capability to execute a saved script on designated hosts. Users can easily view the status of each script and access the most recent script's output, limited to the last 10,000 characters. Importantly, actions related to these scripts are monitored and documented in the activity feed. This enhancement embodies Fleet's values of Empathy and Ownership, empowering admins and users with streamlined script management and clear visibility into script outputs and actions, promoting a more efficient workflow.

### Scheduled query reports

The latest update to Fleet introduces the storage of results from scheduled queries as a report. Each scheduled query can now retain up to 1000 results. For any scheduled query that has results fewer than 1000, Fleet will update these results as new data is received from hosts. 

We’re excited about this powerful feature. Fleet’s query reports allow you to collect and store query data for your hosts, whether online or offline. Storing more data may mean more load on your Fleet database. As you turn on query reports, we recommend watching the DB to see if you need to scale it up. You can always turn off query reports later if you don’t want to scale up.

Users also have the option to disable reports for individual queries using the discard_data field. Saved query results can be disabled in the global configuration. This enhancement fosters Fleet's value of Ownership by providing administrators with greater control over their data and Objectivity by offering clear data storage options and enhanced validation of osquery result logs.

## New features, improvements, and bug fixes

* Added ability to store results of scheduled queries:
  - Will store up to 1000 results for each scheduled query. 
  - If the number of results for a scheduled query is below 1000, then the results will continuously get updated every time the hosts send results to Fleet.
  - Introduced `server_settings.query_reports_disabled` field in global configuration to disable this feature.
  - New API endpoint: `GET /api/_version_/fleet/queries/{id}/report`.
  - New field `discard_data` added to API queries endpoints for toggling report storage for a query. For yaml configurations, use `discard_data: true` to disable result storage.
  - Enhanced osquery result log validation.
  - **NOTE:** This feature enables storing more query data in Fleet. This may impact database performance, depending on the number of queries, their frequency, and the number of hosts in your Fleet instance. For large deployments, we recommend monitoring your database load while gradually adding new query reports to ensure your database is sized appropriately.

* Added scripts tab and table for host details page.

* Added support to return the decrypted disk encryption key of a Windows host.

* Added `GET /hosts/{id}/scripts` endpoint to retrieve status details of saved scripts for a host.

* Added `mdm.os_settings` to `GET /api/v1/hosts/{id}` response.

* Added `POST /api/fleet/orbit/disk_encryption_key` endpoint for Windows hosts to report BitLocker encryption key.

* Added activity logging for script operations (add, delete, edit).

* Added UI for scripts on the controls page.

* Added API endpoints for script management and updated existing ones to accommodate saved script ID.

* Added `GET /mdm/disk_encryption/summary` endpoint for disk encryption summaries for macOS and Windows.

* Added `os_settings` and `os_settings_disk_encryption` filters to various `GET` endpoints for host filtering based on OS settings.

* Enhanced `GET hosts/:id` API response to include more detailed disk encryption data for device client errors.

* Updated controls > disk encryption and host details page to include Windows BitLocker information.

* Improved styling for host details/device user failing policies display.

* Disabled multicursor editing for SQL editors.

* Deprecated `mdm.macos_settings.enable_disk_encryption` in favor of `mdm.enable_disk_encryption`.

* Updated Go version to 1.21.3.

### Bug fixes

* Fixed script content and output formatting issues on the scripts detail modal.

* Fixed a high database load issue in the Puppet match endpoint.

* Fixed setup flows background not covering the entire viewport when resized to some sizes.

* Fixed a bug affecting OS settings information retrieval regarding disk encryption status for Windows hosts.

* Fixed SQL parameters used in the `/api/latest/fleet/labels/{labelID}/hosts` endpoint for certain query parameters, addressing issue 13809.

* Fixed Python's CVE-2021-42919 false positive on macOS which should only affect Linux.

* Fixed a bug causing DEP profiles to sometimes not get assigned correctly to hosts.

* Fixed an issue in the bulk-set of MDM Apple profiles leading to excessive placeholders in SQL.

* Fixed max-height display issue for script content and output in the script details modal.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.39.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-10-26">
<meta name="articleTitle" value="Fleet 4.39.0 | Sonoma support, script library, query reports.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.39.0-1600x900@2x.png">
