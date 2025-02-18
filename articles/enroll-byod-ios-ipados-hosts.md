# Enroll BYOD iOS/iPadOS hosts

This guide will walk you through the process of inviting BYOD (Bring Your Own Device) iPhones and iPads to enroll in Fleet.

By enrolling BYOD iPhones and iPads in Fleet, IT admins can manage software installations, enforce settings, and ensure devices comply with company policies. By adding BYOD devices, you can monitor, enforce settings, and manage security on BYOD iPhones and iPads in real-time, providing enhanced control without compromising user autonomy. This helps secure access to organizational resources while maintaining control over device configurations.

## Prerequisites

* Fleet [v4.57.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0).
* [MDM enabled and configured](https://fleetdm.com/guides/macos-mdm-setup)

## Enrolling BYOD iPad/iOS devices in Fleet

* **Step 1: Navigate to the manage hosts page**
    * Click “Hosts” in the top navigation bar
* **Step 2: Choose the team**
    * Select the desired [team](https://fleetdm.com/guides/teams) from the menu at the top of the screen
* **Step 3: Get a link to share with your end users**
    * Click on “Add hosts.”
    * In the modal, select the **iOS & iPadOS** tab.
    * Copy the link to enroll hosts.
* **Step 4: Distribute the link**
    * Share the link with your end users using an introductory email or message.
    * The link provides instructions to guide users through downloading and installing Fleet’s enrollment profile.

> Each team has a unique URL that includes the team's enrollment secret. This enrollment secret ensures that devices are assigned to the correct team during enrollment. When an incorrect enroll secret is provided, users can still download the enrollment profile, but the enrollment itself will fail (403 error).

## Profile-based vs. account-driven enrollment

Currently, BYOD enrollment in Fleet requires end users to install a configuration profile on their device. This is called profile-based _device_ enrollment. Apple recently deprecated profile-based _user_ enrollment (not supported in Fleet) in favor of the new account-driven enrollment: enrollment happens when end users add a Managed Apple Account to their device. Account-driven enrollment in Fleet is coming soon.

## Conclusion

This guide covered how to invite and enroll BYOD iPhones and iPads into Fleet. This allows IT admins to manage software, enforce settings, and ensure compliance with organizational policies. Streamlining the enrollment process will enable you to secure access to company resources while maintaining control over end-user devices.

For more information on device management and other features, explore Fleet’s documentation and guides to optimize your setup and keep your devices fully secure.

See Fleet's [documentation](https://fleetdm.com/docs/using-fleet) and additional [guides](https://fleetdm.com/guides) for more details on advanced setups, software features, and vulnerability detection.


<meta name="articleTitle" value="Enrolling BYOD iPad/iOS devices in Fleet">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-20">
<meta name="description" value="This guide will walk you through the process of inviting BYOD iPhones and iPads to enroll in Fleet.">
