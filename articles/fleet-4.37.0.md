# Fleet 4.37.0 | Remote script execution & Puppet support.

![Fleet 4.37.0](../website/assets/images/articles/fleet-4.37.0-1600x900@2x.png)

Fleet 4.37.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.33.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.


## Highlights

* Introducing cross-platform script execution
* Vulnerability dashboard
* Puppet support
* Web user interface improvements

### Introducing cross-platform script execution

_Available in Fleet Premium and Fleet Ultimate_

Fleet adds a significant new feature, allowing IT administrators and security engineers to execute shell scripts across macOS, Windows, and Linux. This addition streamlines processes, offers root-level security control, and enables swift, real-time remediation and investigation. Learn more about Fleet's [cross-platform script execution](introducing-cross-platform-script-execution).


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

* 


#### Bug Fixes:

* 


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.37.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-09-07">
<meta name="articleTitle" value="Fleet 4.37.0 | Remote script execution & Puppet support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.37.0-1600x900@2x.png">
