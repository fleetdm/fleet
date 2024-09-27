# Fleet supports Apple’s latest operating systems: macOS 15 Sequoia, iOS 18, and iPadOS 18 

![Fleet supports Apple’s latest operating systems: macOS 15 Sequoia, iOS 18, and iPadOS 18](../website/assets/images/articles/fleet-supports-macos-15-sequoia-ios-18-and-ipados-18-1600x900@2x.jpg)

_Photo by [aditya bhatia](https://www.pexels.com/photo/people-walking-on-a-bridge-between-trees-13809734/)_

Fleet is pleased to announce full support for Apple’s newest operating systems, including macOS 15 Sequoia, iOS 18, and iPadOS 18. With these updates, Fleet ensures seamless management and security capabilities for organizations adopting the latest Apple technology across their device fleet. This release enables IT administrators to confidently manage devices running these new operating systems, leveraging Fleet's robust tools for device monitoring, policy enforcement, and configuration management.

## Noteworthy changes and known issues for macOS 15 Sequoia

macOS 15 Sequoia brings significant changes that may impact device management workflows. Notably, firewall settings are no longer contained in a `.plist` file, resulting in potential reporting issues with osquery. Fleet is working with the osquery community to address this issue and update the `alf` table to correctly report firewall status under Sequoia. For more details on this issue, [follow the progress here](https://github.com/fleetdm/fleet/issues/21802). Additionally, manual installation of unsigned `fleet-osquery.pkg` packages now require extra steps due to changes in Apple's security settings, reflecting a heightened focus on device security.

## Expanding Support for Declarative Device Management (DDM)

As Apple expands the Declarative Device Management (DDM) capabilities with macOS 15 Sequoia, Fleet looks forward to implementing these new functionalities to enhance device management capabilities further. Today, administrators can send DDM JSON payloads directly to hosts, enabling more responsive and granular control over device configurations and settings. Fleet's support for DDM allows organizations to leverage this robust framework to manage devices efficiently without constant communication with the server. For more information on DDM, visit Apple’s [guide to Declarative Device Management](https://support.apple.com/guide/deployment/intro-to-declarative-device-management-depb1bab77f8/1/web/1.0).

## Updates Across macOS, iOS, and iPadOS

Apple has renamed "Profiles" to "Device Management" in all the new operating systems within System Settings. This change affects how administrators and users interact with device management settings on all Apple platforms. For more details on navigating these changes, visit Apple’s [support page](https://it-training.apple.com/tutorials/support/sup530/#Determining-Whether-a-Device-Is-Managed).

Fleet remains committed to supporting Apple's latest advancements, ensuring that your organization can seamlessly integrate and manage devices running macOS 15 Sequoia, iOS 18, and iPadOS 18 with the same reliability and security you expect. Beginning with macOS 16, [Fleet will provide release-day support for major macOS releases](https://fleetdm.com/handbook/engineering#provide-0-day-support-for-major-version-macos-releases), ensuring your fleet is always prepared for the latest updates. Stay tuned for updates as we refine and enhance our support for these new platforms. [Engineering | Fleet handbook](https://fleetdm.com/handbook/engineering#provide-0-day-support-for-major-version-macos-releases)

<meta name="category" value="announcements">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-09-27">
<meta name="articleTitle" value="Fleet supports Apple’s latest operating systems: macOS 15 Sequoia, iOS 18, and iPadOS 18">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-supports-macos-15-sequoia-ios-18-and-ipados-18-1600x900@2x.jpg">
<meta name="description" value="Fleet is pleased to announce full support for macOS 15 Sequoia, iOS 18, and iPadOS 18.">
