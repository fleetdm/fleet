# Fleet 4.50.0 | Security agent deployment, AI descriptions, and Mac Admins SOFA support.

![Fleet 4.50.0](../website/assets/images/articles/fleet-4.50.0-1600x900@2x.png)

Fleet 4.50.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Deploy security agents to macOS, Windows, and Linux
* Policy description and resolutions aided by AI
* Mac Admins SOFA support
* `zsh` support


## Deploy security agents to macOS, Windows, and Linux

Fleet enhances the deployment capabilities for IT administrators, particularly concerning security agents. Now available in Fleet Premium, this feature allows administrators to add and deploy security agents directly to macOS, Windows, and Linux hosts through the Software page, the Fleet API, or via GitOps workflows. This deployment functionality requires that the host has a `fleetd` agent with scripts enabled, but notably, it does not necessitate MDM (Mobile Device Management) features to be enabled within Fleet. This new capability supports a more streamlined and efficient approach to enhancing host security across diverse operating environments, allowing IT and security teams to ensure their hosts are protected with the necessary security tools without the complexity of additional infrastructure changes.


## Policy description and resolutions aided by AI

Fleet aims to enhance how policy descriptions and resolutions are generated for policies. This new functionality leverages artificial intelligence (AI) to automatically populate policy details directly from SQL queries that define policies. When administrators create or modify a policy, they can opt to have the description and resolution fields filled instantly by the AI based on the context and content of the SQL query. This process not only simplifies the task of policy creation by providing pre-generated, meaningful explanations and solutions but also ensures consistency and comprehensiveness in policy documentation. 

This improvement enhances the user experience for administrators and end-users by enabling transparent communication of policy purposes and actions to end-users. This can be especially useful in scenarios like scheduled [maintenance windows](https://fleetdm.com/announcements/fleet-in-your-calendar-introducing-maintenance-windows) visible to users through calendar events or device notifications. By automating the generation of detailed, relevant policy descriptions, Fleet helps ensure that all parties understand what each policy entails and why it is important, enhancing the organization's overall security posture and compliance.


## Mac Admins SOFA support

Fleet has integrated support for the Mac Admins [SOFA](https://github.com/macadmins/sofa) (Structured Open Feed Aggregator), enhancing its capabilities to provide comprehensive tracking and surfacing of update information for macOS and iOS environments. SOFA, known for its machine-readable feed and user-friendly web interface, offers continuous updates on XProtect data, OS updates, and detailed release information. This integration within Fleet is facilitated through the recent updates to the [Mac Admins osquery extension](https://github.com/macadmins/osquery-extension), which now includes tables specifically for security release information (`sofa_security_release_info`) and unpatched CVEs (`sofa_unpatched_cves`).

These additions provide Fleet users with valuable tools for monitoring security updates and vulnerability statuses directly within the Fleet environment. Users can access the new SOFA tables at [SOFA Security Release Info](https://fleetdm.com/tables/sofa_security_release_info) and [SOFA Unpatched CVEs](https://fleetdm.com/tables/sofa_unpatched_cves) for detailed insights. For those looking to delve deeper into the application of these tools, Graham Gilbert’s blog post, [Investigating unpatched CVEs with osquery and SOFA](https://grahamgilbert.com/blog/2024/05/03/investigating-unpatched-cves-with-osquery-and-sofa/), offers an in-depth look at leveraging osquery in conjunction with SOFA to enhance digital security and compliance efforts. This integration underscores Fleet's commitment to providing robust, actionable intelligence for IT administrators and security professionals managing Apple devices.


## `zsh` support

Fleet has expanded its scripting capabilities by adding support for `zsh` (Z Shell) scripts, catering to IT administrators' and developers' diverse scripting preferences. This update allows users to execute `zsh` scripts directly within Fleet, providing a flexible and powerful toolset for managing and automating tasks across various systems. By accommodating `zsh`, known for its robust features and interactive use enhancements over `bash`, Fleet enhances its utility for more sophisticated script operations. This support not only broadens the scope of administrative scripts that can be run but also aligns with the ongoing efforts to adapt to the evolving needs of users in dynamic IT environments.





## Changes





## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.50.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-05-20">
<meta name="articleTitle" value="Fleet 4.50.0 | Security agent deployment, AI descriptions, and Mac Admins SOFA support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.50.0-1600x900@2x.png">
