# Fleet 4.47.0 | Cross-platform remote wipe, vulnerabilities page, and scripting improvements.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/yDBob6v1MZQ?si=pyNbrHgayW-ANu-a" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.47.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.47.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Remote wipe for macOS, Windows, and Linux
* Vulnerabilities page
* Improved scripting
* Improved Windows configuration profiles
* Per team host status webhook
* Improved DEP profile assignment process
* Policy data in `/hosts` API 


### Remote wipe for macOS, Windows, and Linux


Fleet has added the ability to remotely wipe devices across macOS, Windows, and Linux operating systems. This functionality is essential for IT and security professionals needing to ensure data security, especially when devices may be lost, stolen, or compromised. By facilitating the remote erasure of sensitive information, Fleet provides an added security layer, helping prevent unauthorized access to corporate data. This feature is part of Fleet's ongoing commitment to effectively equip administrators with comprehensive tools for managing and securing their environments. It underscores our focus on providing robust, practical solutions that address the evolving challenges today's IT and security teams face.


### Vulnerabilities page

A dedicated vulnerabilities page within the Software page has been added to provide a centralized overview of all vulnerabilities (CVEs) identified across hosts. This feature enables security engineers to quickly identify, assess, and prioritize CVEs affecting their fleet. More importantly, it offers the functionality to export a list of hosts affected by a specific CVE, streamlining the process of passing crucial information to the engineers responsible for remediation. This development supports proactive security management by offering clear, actionable insights into the fleet's vulnerability status, thus facilitating a more efficient response to potential security threats. This aligns with Fleet's commitment to transparency and actionability, empowering teams with the necessary tools to enhance their security posture effectively.


### Improved scripting


Fleet enhances the scope of remote script execution capabilities by extending support for longer scripts saved within the Fleet platform and enabling the execution of scripts by their name through the `fleetctl` CLI. This improvement directly responds to the needs of IT administrators and security professionals who require the flexibility to run extensive scripts across their device fleets for comprehensive diagnostics, maintenance, or security tasks. Additionally, the ability to execute scripts by name simplifies the process, making script management more efficient and reducing the potential for errors. This update represents Fleet's commitment to providing practical, user-centric solutions that enhance the effectiveness and ease of managing and securing your fleet. It reflects an understanding of modern IT infrastructure's complex, evolving needs and the importance of adaptable, reliable tools in addressing those needs.


### Improved Windows configuration profiles


Fleet now supports the `<Add>` element in Windows configuration profiles, addressing a specific need for IT administrators managing Windows devices. This development allows for more nuanced control over Windows OS settings, including adding new configurations such as Wi-Fi profiles, a functionality particularly useful in scenarios where the `<Replace>` element is ineffective. This enhancement simplifies the management of Windows devices, providing administrators with the flexibility to enforce policies and settings essential for maintaining device security and operational efficiency. Fleet seeks to empower IT professionals, ensuring administrators have the tools to tailor their environments according to specific requirements and best practices.


### Per team host status webhook

Webhooks can be configured at the team level to alert administrators when a specified percentage of their team's hosts go offline. This allows an admin to prioritize webhooks for critical teams while setting a higher threshold for less critical teams. The web UI allows for standard configurations, with additional customizable options available in the configuration file for more tailored setups. Such granularity in notifications ensures that team admins can promptly address potential issues specific to their teams, enhancing their environments' overall responsiveness and management. This addition reflects Fleet's dedication to providing tools that support proactive and informed management, aligning with the platform's commitment to transparency and adaptability in device monitoring and security.


### Improved DEP profile assignment process 

MacOS hosts may occasionally face issues during the Device Enrollment Program (DEP) profile assignment process, now called Automatic Device Enrollment (ADE). Recognizing the challenges posed by the Mobile Device Management (MDM) API's rate limitations, this update implements a smart retry mechanism. When a profile application to a host fails, the process times out and is scheduled to retry within the hour. This approach is designed to mitigate the impact of API rate limits, enhancing the efficiency of profile assignments. Most failed DEP profile assignments are resolved within this timeframe, streamlining the enrollment process and reducing administrative overhead. Fleet is dedicated to simplifying device management tasks, ensuring a smoother, more reliable enrollment experience.


### Policy data in `/hosts` API

Policy data is now included directly within the `GET /hosts` API response in Fleet. This is tailored for users who prefer streamlined data access by querying a single API endpoint to retrieve comprehensive policy data for all hosts. With this enhancement, users can efficiently export this data into an external database, facilitating the custom creation of dashboards and reports that suit their specific monitoring and analysis needs. This development underscores Fleet's dedication to efficiency and adaptability, aiming to provide users with the tools they need for effective and tailored fleet management. By simplifying the process of data aggregation and visualization, Fleet empowers users to understand their device compliance posture better and make informed decisions based on comprehensive policy adherence metrics.




## Changes

### Endpoint operations
- Implemented UI for team-specific host status webhooks.
- Added Unicode and emoji support for policy and team names.
- Allowed gitops user to access specific endpoints.
- Enabled setting host status webhook at the team level via REST API and fleetctl.
- GET /hosts API endpoint now populates policies with `populate_policies=true` query parameter.
- Supported custom options set via CLI in the UI for host status webhook settings.
- Surfaced VS code extensions in the software inventory.
- Added a "No team" team option when running live queries from the UI.
- Fixed tranferring hosts between teams across multiple pages.
- Fixed policy deletion not updating policy count.
- Fixed RuntimeError in fleetd-chrome and buggy filters for exporting hosts.

### Device management (MDM)
- Added wipe command to fleetctl and the `POST /api/v1/fleet/hosts/:id/wipe` Fleet Premium API endpoint.
- Updated `fleetctl run-script` to include new flags and `POST /scripts/run/sync` API to receive new parameters.
- Enabled usage of `<Add>` nodes in Windows MDM profiles.
- Added backend functionality for the new way of storing script contents and updated the script character limit.
- Updated the database schema to support the increase in script size.
- Prevented running cleanup tasks and re-enqueuing commands for hosts on SCEP renewals.
- Improved osquery queries for MDM detection.
- Prevented redundant ADE profile assignment.
- Updated fleetctl gitops, default MDM configs were set to default values when not defined.
- Displayed disk encryption status in macOS as "verifying."
- Allowed GitOps user to access MDM hosts and profiles endpoints.
- Added UI for wiping a host with Fleet MDM.
- Rolled up MDM solutions by name on the dashboard MDM card.
- Added functionality to surface MDM devices where DEP assignment failed.
- Fixed MDM profile installation error visibility.
- Fixed Windows MDM profile command "Type" column display.
- Fixed an issue with macOS ADE enrollments getting a "method not allowed" error.
- Fixed Munki issues truncated tooltip bug.
- Fixed a bug causing Windows hosts to appear when filtering by bootstrap package status.

### Vulnerability management
- Reduced vulnerability processing time by optimizing the vulnerability dictionary grouping.
- Fixed an issue with `mdm.enable_disk_encryption` JSON null values causing issues.
- Fixed vulnerability processing for non-ASCII software names.

### Bug fixes and improvements
- Upgraded Golang version to 1.21.7.
- Updated page descriptions and fixed alignment of critical policy checkboxes.
- Adjusted font size for tooltips in the settings page to follow design guidelines.
- Fixed a bug where the "Done" button on the add hosts modal could be covered.
- Fixed UI styling and alignment issues across various pages and modals.
- Fixed the position of live query/policy host search icon and UI loading states.
- Fixed issues with how errors were captured in Sentry for improved precision and coverage.

## Fleet 4.46.2 (Mar 4, 2024)

### Bug fixes

* Fixed a bug where the pencil icons next to the edit query name and description fields were inconsistently spaced.
* Fixed an issue with `mdm.enable_disk_encryption` where a `null` JSON value caused issues with MDM profiles in the `PATCH /api/v1/fleet/config` endpoint.
* Displayed disk encryption status in macOS as "verifying" while Fleet verified if the escrowed key could be decrypted.
* Fixed UI styling of loading state for automatic enrollment settings page.

## Fleet 4.46.1 (Feb 27, 2024)

### Bug fixes

* Fixed a bug in running queries via API.
	- Query campaign not clearing from Redis after timeout
* Added logging when a Redis connection is blocked for a long time waiting for live query results.
* Added support for the `redis.conn_wait_timeout` configuration setting for Redis standalone (it was previously only supported on Redis cluster).
* Added Redis cleanup of inactive queries in a cron job, so temporary Redis failures to stop a live query doesn't leave such queries around for a long time.
* Fixed orphaned live queries in Redis when client terminates connection 
	- `POST /api/latest/fleet/queries/{id}/run`
	- `GET /api/latest/fleet/queries/run`
	- `POST /api/latest/fleet/hosts/identifier/{identifier}/query` 
	- `POST /api/latest/fleet/hosts/{id}/query`
* Added --server_frequent_cleanups_enabled (FLEET_SERVER_FREQUENT_CLEANUPS_ENABLED) flag to enable cron job to clean up stale data running every 15 minutes. Currently disabled by default.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.47.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-03-12">
<meta name="articleTitle" value="Fleet 4.47.0 | Cross-platform remote wipe, vulnerabilities page, and scripting improvements.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.47.0-1600x900@2x.png">
