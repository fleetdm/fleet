# Fleet 4.50.0 | Security agent deployment, AI descriptions, and Mac Admins SOFA support.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/0SSww4lzL_A?si=TzDdP8HmCKwi5EZg" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.50.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Deploy security agents to macOS, Windows, and Linux
* Policy description and resolutions aided by AI
* Mac Admins SOFA support
* `zsh` support


## Deploy security agents to macOS, Windows, and Linux

Fleet enhances the deployment capabilities for IT administrators, particularly concerning security agents. Now available in Fleet Premium, this feature allows administrators to add and deploy security agents directly to macOS, Windows, and Linux hosts through the Software page, the Fleet API, or via GitOps workflows. This deployment functionality requires that the host has a `fleetd` agent with scripts enabled, but notably, it does not necessitate MDM (Mobile Device Management) features to be enabled within Fleet. This new capability supports a more streamlined and efficient approach to enhancing host security across diverse operating environments, allowing IT and security teams to ensure their hosts are protected with the necessary security tools without the complexity of additional infrastructure changes.

For users who self-manage (host) Fleet, this feature requires connecting Fleet with an S3 bucket. See how in the server configuration reference [here](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3).

## Policy description and resolutions aided by AI

Fleet aims to enhance how policy descriptions and resolutions are generated for policies. This new functionality leverages artificial intelligence (AI) to automatically populate policy details directly from SQL queries that define policies. It is important to note that Fleet does not use any data to train large language models (LLMs); only the policy queries (SQL) are sent to the LLM for generating descriptions and resolutions. When administrators create or modify a policy, they can opt to have the description and resolution fields filled instantly by the AI based on the context and content of the SQL query. This process not only simplifies the task of policy creation by providing pre-generated, meaningful explanations and solutions but also ensures consistency and comprehensiveness in policy documentation. 

This improvement enhances the user experience for administrators and end-users by enabling transparent communication of policy purposes and actions to end-users. This can be especially useful in scenarios like scheduled [maintenance windows](https://fleetdm.com/announcements/fleet-in-your-calendar-introducing-maintenance-windows) visible to users through calendar events or device notifications. By automating the generation of detailed, relevant policy descriptions, Fleet helps ensure that all parties understand what each policy entails and why it is important, enhancing the organization's overall security posture and compliance.


## Mac Admins SOFA support

Fleet has integrated support for the Mac Admins [SOFA](https://github.com/macadmins/sofa) (Structured Open Feed Aggregator), enhancing its capabilities to provide comprehensive tracking and surfacing of update information for macOS hosts. SOFA, known for its machine-readable feed and user-friendly web interface, offers continuous updates on XProtect data, OS updates, and detailed release information. This integration within Fleet is facilitated through Graham Gilbert's recent updates to the [Mac Admins osquery extension](https://github.com/macadmins/osquery-extension), which includes tables specifically for security release information ([`sofa_security_release_info`](https://fleetdm.com/tables/sofa_security_release_info)) and unpatched CVEs ([`sofa_unpatched_cves`](https://fleetdm.com/tables/sofa_unpatched_cves)).

These additions provide Fleet users with valuable tools for monitoring security updates and vulnerability statuses directly within the Fleet environment. Users can access the new SOFA tables at [SOFA Security Release Info](https://fleetdm.com/tables/sofa_security_release_info) and [SOFA Unpatched CVEs](https://fleetdm.com/tables/sofa_unpatched_cves) for detailed insights. For those looking to delve deeper into the application of these tools, Graham Gilbert’s blog post, [Investigating unpatched CVEs with osquery and SOFA](https://grahamgilbert.com/blog/2024/05/03/investigating-unpatched-cves-with-osquery-and-sofa/), offers an in-depth look at leveraging osquery in conjunction with SOFA to enhance digital security and compliance efforts. This integration underscores Fleet's commitment to providing robust, actionable intelligence for IT administrators and security professionals managing Apple devices.


## `zsh` support

Fleet has expanded its scripting capabilities by adding support for `zsh` (Z Shell) scripts, catering to IT administrators' and developers' diverse scripting preferences. This update allows users to execute `zsh` scripts directly within Fleet, providing a flexible and powerful toolset for managing and automating tasks across various systems. By accommodating `zsh`, known for its robust features and interactive use enhancements over `bash`, Fleet enhances its utility for more sophisticated script operations. This support not only broadens the scope of administrative scripts that can be run but also aligns with the ongoing efforts to adapt to the evolving needs of users in dynamic IT environments.





## Changes

### Endpoint operations

- Added optional AI-generated policy descriptions and remediations. 
- Added flag to enable deletion of old activities and associated data in cleanup cron job.
- Added support for escaping `$` (with `\`) in gitops yaml files.
- Optimized policy_stats updates to not lock the policy_membership table.
- Optimized the hourly host_software count query to reduce individual query runtime.
- Updated built-in labels to support being applied via `fleetctl apply`.

### Device management (MDM)

- Added endpoints to upload, delete, and download software installers.
- Added ability to upload software from the UI.
- Added functionality to filter hosts by software installer status.
- Added support to the global activity feed for "Added software" and "Deleted software" actions.
- Added the `POST /api/fleet/orbit/software_install/result` endpoint for fleetd to send results for a software installation attempt.
- Added the `GET /api/v1/fleet/hosts/{id}/software` endpoint to list the installed software for the host.
- Added support for uploading and running zsh scripts on macOS and Linux hosts.
- Added the `cron` job to periodically remove unused software installers from the store.
- Added a new command `fleetctl api` to easily use fleetctl to hit any REST endpoint via the CLI.
- Added support to extract package name and version from software installers.
- Added the uninstalled but available software installers to the response payload of the "List software titles" endpoint.
- Updated MySQL host_operating_system insert statement to reduce table lock time.
- Updated software page to support new add software feature.
- Updated fleetctl to print team id as part of the `fleetctl get teams` command.
- Implemented an S3-based and local filesystem-based storage abstraction for software installers.

### Vulnerability management

- Added OVAL vulnerability scanning support on Ubuntu 22.10, 23.04, 23.10, and 24.04.

### Bug fixes and improvements

- Fixed ingestion of private IPv6 address from agent.
- Fixed a bug where a singular software version in the Software table generated a tooltip unnecessarily.
- Fixed bug where updating user via `/api/v1/fleet/users/:id` endpoint sometimes did not update activity feed.
- Fixed bug where hosts query results were not cleared after transferring the host to other teams.
- Fixed a bug where the returned `count` field included hosts that the user did not have permission to see.
- Fixed issue where resolved_in_version was not returning if the version number differed by a 4th part.
- Fixed MySQL sort buffer overflow when fetching activities.
- Fixed a bug with users not being collected on Linux devices.
- Fixed typo in Powershell scripts for installing Windows software.
- Fixed an issue with software severity column display in Fleet UI.
- Fixed the icon on Software OS table to show a Linux icon for Linux operating systems.
- Fixed missing tooltips in disabled "Calendar events" manage automations dropdown option.
- Updated switched accordion text.
- Updated sort the host details page queries table case-insensitively.
- Added support for ExternalId in STS Assume Role APIs.

## Fleet 4.49.4 (May 20, 2024)

### Bug fixes

* Fixed an issue with SCEP renewals that could prevent commands to renew from being enqueued.

## Fleet 4.49.3 (May 06, 2024)

### Bug fixes

* Improved Windows OS version reporting.
* Fixed a bug where when updating a policy's 'platform' field, the aggregated policy stats were not cleared.
* Improved URL and email validation in the UI.

## Fleet 4.49.2 (Apr 30, 2024)

### Bug fixes

* Restored missing tooltips when hovering over the disabled "Calendar events" manage automations dropdown option.
* Fixed an issue on Windows hosts enrolled in MDM via Azure AD where the command to install Fleetd on the device was sent repeatedly, even though `fleetd` had been properly installed.
* Improved handling of different scenarios and edge cases when hosts turned on/off MDM.
* Fixed issue with uploading of some signed Apple mobileconfig profiles.
* Added an informative flash message when the user tries to save a query with invalid platform(s).
* Fixed bug where Linux host wipe would repeat if the host got re-enrolled.

## Fleet 4.49.1 (Apr 26, 2024)

### Bug fixes

* Fixed a bug that prevented the Fleet server from starting if Windows MDM was configured but Apple MDM wasn't.




## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.50.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-05-22">
<meta name="articleTitle" value="Fleet 4.50.0 | Security agent deployment, AI descriptions, and Mac Admins SOFA support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.50.0-1600x900@2x.png">
