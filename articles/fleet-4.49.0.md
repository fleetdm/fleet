# Fleet 4.49.0 | VulnCheck's NVD++, device health API, `fleetd` data parsing.

![Fleet 4.49.0](../website/assets/images/articles/fleet-4.49.0-1600x900@2x.png)

Fleet 4.49.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.49.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Enhancing Fleet's vulnerability management with VulnCheck integration
* Device health API includes critical policy and resolution data
* `fleetd` data parsing expansion
* Apply labels using UI or API
* Resend configuration profiles



### Enhancing Fleet's vulnerability management with VulnCheck integration

Fleet is integrating VulnCheck to enhance our vulnerability management capabilities, ensuring our users can manage Common Platform Enumeration (CPE) data more effectively and securely. Utilizing VulnCheck's NVD++ service, Fleet will provide reliable, timely access to vulnerability data, overcoming delays and inconsistencies in the National Vulnerability Database (NVD). This integration improves the accuracy and timeliness of threat detection and streamlines the overall vulnerability management process, empowering IT administrators to identify and mitigate security threats swiftly. Learn more about how this enhancement strengthens Fleet's security framework in our latest blog post: [Enhancing Fleet's Vulnerability Management with VulnCheck Integration](https://fleetdm.com/announcements/enhancing-fleets-vulnerability-management-with-vulncheck-integration).


### Device health API includes critical policy and resolution data

Fleet has updated its device health API to include critical and policy resolution data, enhancing the utility of this API for specific workflow conditions where compliance verification is essential before proceeding. This update allows for real-time authentication checks to ensure a host complies with set policies, thereby supporting secure and compliant operational workflows. By integrating critical compliance data into the device health API, Fleet enables administrators to enforce and verify security policies efficiently, ensuring that only compliant devices proceed in sensitive or critical operations. This enhancement supports thorough compliance management and reinforces secure practices within IT environments, streamlining processes where policy adherence is crucial.


### `fleetd` data parsing expansion

Fleet's agent (`fleetd`) has expanded its data parsing capabilities by adding support for JSON, JSONL, XML, and INI file formats as tables. This functionality allows for more versatile data extraction and management, enabling users to convert these popular data formats directly into queryable tables. This capability is particularly useful for IT and security teams who need to analyze and monitor configuration and data files across various systems within their digital environments efficiently. By facilitating integration and manipulation of data from these diverse formats, Fleet helps ensure that teams can maintain better oversight and faster responsiveness when managing operational and security needs. This feature is a natural extension of Fleet's ongoing efforts to empower IT professionals with comprehensive tools for robust data handling and security management.


### Apply labels using UI or API

Fleet has expanded the flexibility of label management by enabling users to add labels manually through both the UI and API. This capability was previously available only via the CLI. This enhancement allows administrators to more conveniently categorize and manage hosts directly within the user interface or programmatically via the API, aligning with various operational workflows. By streamlining the label application process, Fleet makes it easier for teams to organize and access host data according to specific criteria, thereby improving operational efficiency and responsiveness. This update supports better integration and automation capabilities within IT environments, empowering users to maintain organized and effective device management practices.


### Resend configuration profiles

Fleet has introduced a new feature that allows users to resend a configuration profile to a host, which is crucial for maintaining current settings and certificates. This functionality is particularly beneficial in scenarios where renewing SCEP certificates, signing certificates need updating, or reapplication of existing configurations is required to ensure continuity and compliance. By enabling the reissuance of configuration profiles directly from the platform, Fleet supports continuous device management and security upkeep, facilitating a proactive approach to maintaining and securing digital environments. This feature enhances Fleet's utility for administrators by simplifying the management of device configurations.



## Changes





## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.49.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-04-23">
<meta name="articleTitle" value="Fleet 4.49.0 | VulnCheck's NVD++, device health API, fleetd data parsing.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.49.0-1600x900@2x.png">
