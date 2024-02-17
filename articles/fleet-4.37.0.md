# Fleet 4.37.0 | Remote script execution & Puppet support.

![Fleet 4.37.0](../website/assets/images/articles/fleet-4.37.0-1600x900@2x.png)

Fleet 4.37.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.37.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.


## Highlights

* Introducing cross-platform script execution
* Vulnerability dashboard
* Puppet support
* Web user interface improvements


### Introducing cross-platform script execution

_Available in Fleet Premium and Fleet Ultimate_

Fleet adds a significant new feature, allowing IT administrators and security engineers to execute shell scripts across macOS, Windows, and Linux. This addition streamlines processes, offers root-level security control, and enables swift, real-time remediation and investigation. Learn more about Fleet's [cross-platform script execution](https://fleetdm.com/announcements/introducing-cross-platform-script-execution).


### Vulnerability dashboard

_Available in Fleet Premium and Fleet Ultimate_

Fleet is excited to beta the Vulnerability dashboard, which focuses on actionable data for security and IT teams. The dashboard will feature the ability to pin priority Common Vulnerabilities and Exposures (CVEs) and set approved Operating System versions. These features offer a straightforward way to monitor and ensure patch compliance across multiple teams, echoing Fleet's emphasis on 🟠 Ownership and efficient execution of tasks.

The dashboard is designed to facilitate cross-team reporting of vulnerability information, fulfilling a crucial user story: As a member of the security and IT team, the dashboard enables the tracking and reporting of vulnerabilities to ensure that all teams meet compliance standards. This aligns with Fleet's value of 🟣 Openness, encouraging transparent information sharing within the organization.

While the Vulnerability Dashboard is still in development, those interested in this functionality can contact us for more details. We plan to integrate this into the product later, reflecting Fleet's long-term thinking and commitment to 🟠 Ownership. This feature aims to help users act responsibly and proactively in the face of security threats.


### Puppet support

_Available in Fleet Premium and Fleet Ultimate_

The addition of a Puppet module to Fleet serves to strengthen the company's commitment to 🟠 Ownership by streamlining the management of servers and laptops. Puppet, an open-source configuration management tool, automates the alignment of infrastructure to its desired state. In this integration, Fleet leverages Puppet facts to categorize hosts into specific groupings. These groupings then map onto teams within Fleet, ensuring that the correct profiles are assigned to the appropriate teams. 

The system prioritizes regular synchronization of teams and host groupings, reflecting Fleet's focus on 🟢 Results by enabling efficient and reliable execution of tasks. By automating these processes, the Puppet module allows IT and security teams to focus on more complex issues, taking the legwork out of mundane configuration tasks.

The integration ultimately embodies Fleet's value of 🟣 Openness by making it easier for different teams to manage and access relevant configuration profiles. This fosters a more transparent, efficient, and collaborative work environment, helping to keep all team members on the same page regarding system configurations and security protocols.


### Web user interface improvements

In line with Fleet's values of 🟢 Results and 🟣 Openness, the latest 4.37.0 release brings practical improvements to the web user interface, building on the foundations set in [version 4.32.0](https://fleetdm.com/releases/fleet-4.32.0). The update enables users to command-click (or ctrl-click on Windows) on table elements to open them in a new browser tab, enhancing workflow efficiency. This comes after the 4.32.0 update, which made URLs the source of truth for the Manage Queries page table state, adding an extra layer of clarity and transparency. These changes aim to simplify user interactions with the platform while promoting efficient, straightforward management of queries.


## New features, improvements, and bug fixes

* * Added `/scripts/run` and `scripts/run/sync` API endpoints to send a script to be executed on a host and optionally wait for its results.

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


#### Bug fixes


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


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.37.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-09-07">
<meta name="articleTitle" value="Fleet 4.37.0 | Remote script execution & Puppet support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.37.0-1600x900@2x.png">
