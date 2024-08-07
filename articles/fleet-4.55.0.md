# Fleet 4.55.0 | MySQL 8, arm64 support, FileVault improvements, VPP support.

![Fleet 4.55.0](../website/assets/images/articles/fleet-4.55.0-1600x900@2x.png)

Fleet 4.55.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.55.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* MySQL 8 support, MySQL 5.7 sunsets
* FileVault key rotation with Escrow Buddy
* FileVault enforcement at enrollment
* Arm64 support
* VPP app support for macOS
* "No team" software support

### MySQL 8 support, MySQL 5.7 sunsets

Fleet has updated its database compatibility by adding support for MySQL 8, while simultaneously dropping support for MySQL 5.7. This change aligns Fleet with the latest advancements in database technology, offering enhanced performance, security, and features available in MySQL 8. Organizations using Fleet are encouraged to upgrade their database systems to MySQL 8 to take full advantage of these improvements. By focusing on the latest supported versions, Fleet ensures that its platform remains robust, secure, and well-equipped to handle the demands of modern IT environments while phasing out older versions that may not provide the same level of performance or security.

### FileVault key rotation with Escrow Buddy

Fleet now includes support for FileVault key rotation using [Escrow Buddy](https://github.com/macadmins/escrow-buddy), a tool developed by the MacAdmins community to securely manage and rotate FileVault recovery keys on macOS devices. This feature allows IT administrators to automate the process of rotating FileVault keys, ensuring that encrypted macOS hosts remain secure while maintaining access control. By integrating with Escrow Buddy, Fleet enables seamless key management, reducing the administrative burden of manually rotating keys and enhancing the overall security posture of macOS environments. This update reflects Fleet's commitment to providing robust security tools that integrate with trusted community resources, ensuring organizations can efficiently manage device encryption and recovery processes.

### FileVault enforcement at enrollment

Fleet now supports enforcing FileVault encryption during the enrollment process for macOS devices, ensuring that all newly enrolled Macs are automatically encrypted. This feature enhances security by mandating that FileVault is enabled as part of the initial device setup, reducing the risk of unencrypted data on managed endpoints. By integrating FileVault enforcement into the enrollment workflow, Fleet helps organizations maintain a consistent security posture across their macOS fleet, ensuring compliance with internal policies and regulatory requirements. This update underscores Fleet's commitment to providing comprehensive security management tools that protect sensitive data and simplify the administration of macOS devices.

### Arm64 support

Fleet now includes support for Linux hosts running on the arm64 architecture. This update enables organizations to integrate a broader range of devices into their Fleet management system, ensuring comprehensive oversight and control across diverse hardware environments. By supporting arm64 Linux hosts, Fleet caters to the growing use of ARM-based systems in various sectors, allowing IT administrators to manage these devices with the same level of detail and efficiency as traditional x86-based hosts. This aligns with Fleet's commitment to providing versatile and inclusive device management solutions, empowering users to maintain a unified and efficient IT infrastructure.

### VPP app support for macOS

Fleet now supports installing Volume Purchase Program (VPP) apps from the Apple App Store on macOS devices. This feature enables IT administrators to deploy and manage apps purchased through Apple's VPP directly to macOS hosts, streamlining the process of distributing essential software across the organization. By integrating VPP app installations into Fleet, organizations can ensure that licensed applications are efficiently deployed to the appropriate devices, improving software management and compliance. This update enhances Fleet's capabilities in managing macOS environments, offering a more seamless and centralized approach to app distribution for enterprise and educational settings.

### "No team" software support

Fleet now supports adding software to the "No team" team, providing greater flexibility in managing software across an organization's devices. This feature allows administrators to deploy and manage software that applies universally without being restricted to specific teams. By adding software to the "No team" team, IT teams can ensure that essential tools and applications are available across all devices, regardless of their team assignment. This update simplifies the management of widely used software and enhances the ability to maintain consistency and compliance across the entire fleet. It reflects Fleet's commitment to offering versatile solutions that cater to diverse organizational needs and streamline device management processes.

## Changes

### Endpoint Operations



## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.55.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-08-07">
<meta name="articleTitle" value="Fleet 4.55.0 | MySQL 8, arm64 support, FileVault improvements, VPP support.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.55.0-1600x900@2x.png">
