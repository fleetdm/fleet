# Fleet 4.57.0 | Software improvements, policy automation, GitLab support.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/xcCmYm7eLZM?si=ZzrSy2lSo0DMdzq-" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.57.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
* Software improvements
* Policy automation: install software
* iPhone/iPad BYOD
* GitLab pipelines for GitOps

### Software improvements

Fleet allows admins to edit software items directly, offering greater control over software management across hosts. This feature allows IT teams to modify details such as software names or versions, ensuring the software inventory remains accurate and aligned with organizational needs. Additionally, Fleet has introduced the option to uninstall software from hosts, simplifying the removal of unwanted or outdated applications. 

For most cases, Fleet handles the uninstall process automatically, with the uninstall script conveniently located under “Advanced options.” However, Fleet stands out by allowing administrators to view and tweak the script if needed. This flexibility is beneficial when a host is in a unique state or the automatic uninstall process encounters issues. Fleet strives to provide full transparency into what’s under the hood, enabling IT teams to make necessary adjustments for specific scenarios. These updates enhance the efficiency of software management while maintaining flexibility, reflecting Fleet’s commitment to providing user-centric and adaptable solutions.

### Policy automation: install software

Admins can automatically trigger software installations when a policy fails, adding a proactive approach to maintaining compliance and security. This feature is handy when a device is found to have a vulnerable version of software installed. If a policy detects this vulnerability, Fleet can automatically install a secure, updated version of the software to remediate the issue and bring the host back into compliance. This automation helps IT teams address vulnerabilities quickly and efficiently, without manual intervention, ensuring that devices across the fleet remain secure and up-to-date. It highlights Fleet’s commitment to streamlining device management and enhancing security through automation.

### iPhone/iPad BYOD

Fleet now supports Bring Your Own Device (BYOD) enrollment for iPhone (iOS) and iPad (iPadOS) devices, providing organizations with a more flexible approach to managing employee-owned devices. This feature allows employees to enroll personal iPhones and iPads into Fleet’s Mobile Device Management (MDM) system, enabling IT teams to enforce security policies, manage configurations, and ensure compliance without needing complete control over the entire device. With BYOD enrollment, companies can balance security and privacy, seamlessly managing work-related configurations on personal devices while respecting the end user’s control over their personal data. This update enhances Fleet’s capabilities for managing various devices and supports organizations with modern, flexible workforce environments.

### GitLab pipelines for GitOps

Fleet now supports GitLab pipelines for its [GitOps integration](https://github.com/fleetdm/fleet-gitops), expanding the flexibility of how organizations manage their device configurations and policies through version control. With GitLab pipelines, IT teams can automate the deployment and management of Fleet configurations directly from their GitLab repositories, streamlining workflows and ensuring that changes are tracked, tested, and deployed consistently across their fleet. This integration enhances the automation and reliability of device management, enabling teams to adopt a more scalable and auditable approach to managing their Fleet environment. By supporting both GitLab and existing CI/CD tools, Fleet continues to empower organizations to implement modern, efficient workflows for managing configurations and policies.

## Changes

**NOTE:** Beginning with Fleet v4.55.0, Fleet no longer supports MySQL 5.7 because it has reached [end of life](https://mattermost.com/blog/mysql-5-7-reached-eol-upgrade-to-mysql-8-x-today/#:~:text=In%20October%202023%2C%20MySQL%205.7,to%20upgrade%20to%20MySQL%208.). The minimum version supported is MySQL 8.0.36.

**Endpoint Operations**

- Added support for configuring policy installers via GitOps.
- Added support for policies in "No team" that run on hosts that belong to "No team".
- Added reserved team names: "All teams" and "No team".
- Added support the software status filter for 'No teams' on the hosts page.
- Enable 'No teams' funcitonality for the policies page and associated workflows.
- Added reset install counts and cancel pending installs/uninstalls when GitOps installer updates change package contents.
- Added support for software installer packages, self-service flag, scripts, pre-install query, and self-service availability to be edited in-place rather than deleted and re-added.

**Device Management (MDM)**

- Added feature allowing automatic installation of software on hosts that fail policies.
- Added feature for end users to enroll BYOD devices into Fleet MDM.
- Added the ability to use Fleet to uninstall packages from hosts.
- Added an endpoint for getting an OTA MDM profile for enrolling iOS and iPadOS hosts.
- Added protocol support for OTA enrollment and automatic team assignment for hosts.
- Added validation of Setup Assistant profiles on profile upload.
- Added validation to prevent installing software on a host with a pending installation.
- Allowed custom SCEP CA certificates with any kind of extendedKeyUsage attributes.
- Modified `POST /api/latest/fleet/software/batch` endpoint to be asynchronous and added a new endpoint `GET /api/latest/fleet/software/batch/{request_uuid}` to retrieve the result of the batch upload.

**Vulnerability Management**

- Fixed a false negative vulnerability for git.
- Fixed false positive vulnerabilities for minio.
- Fixed an issue where virtual box for macOS wasn't matching against the NVD product name.
- Fixed Ubuntu python package false positive vulnerabilities by removing duplicate entries for ubuntu python packages installed by dpkg and renaming remaining pip installed packages to match OVAL definitions.

**Bug fixes and improvements**

- Updated Go to go1.23.1.
- Removed validation of APNS certificate from server startup.
- Removed invalid node keys from server logs.
- Improved the UX of turning off MDM on an offline host.
- Improved clarity of GitOps VPP app ID type errors.
- Improved gitops error message about enabling windows MDM.
- Improved messaging for VPP token constraint errors.
- Improved loading state for UI tables when no data is present yet.
- Improved permissions so that hosts can no longer access installers that aren't directly assigned to them.
- Improved verification of premium license before uploading VPP tokens.
- Added "0 items" description on empty software tables for UI consistency.
- Updated the macos target minimum version tooltip.
- Fixed logic to properly catch and log APNs errors.
- Fixed UI overflow issues with OS settings table data.
- Fixed regression for checking email used to get a signed CSR.
- Fixed bugs on enrollment profiles when the organization name contains invalid XML characters.
- Fixed an issue with cron profiles delivery failing if a Windows VM is enrolled twice.
- Fixed issue where Fleet server could start when an expired ABM certificate was provided as server config.
- Fixed self-service checkbox appearing when iOS or iPadOS app is selected.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.57.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-09-23">
<meta name="articleTitle" value="Fleet 4.57.0 | Software improvements, policy automation, GitLab support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.57.0-1600x900@2x.png">
