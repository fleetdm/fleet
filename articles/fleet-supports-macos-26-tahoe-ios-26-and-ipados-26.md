# Fleet supports Apple’s latest operating systems: macOS Tahoe 26, iOS 26, and iPadOS 26 

![Fleet supports Apple’s latest operating systems: macOS Tahoe 26, iOS 26, and iPadOS 26](../website/assets/images/articles/fleet-supports-macos-26-tahoe-ios-26-and-ipados-26-1060x707@2x.jpg)

_Photo by [MariuszBlach](https://www.istockphoto.com/photo/lake-tahoe-gm480641071-36497954)_

With Apple's releases of macOS Tahoe 26, iOS 26, and iPadOS 26, Fleet provides same-day support for IT teams to upgrade immediately so devices will remain secure and fully managed. This means that all existing Fleet features are tested and bugs are fixed. 

Also, Fleet will support these new features:
- MDM migration with Apple Business (AB)
- Declarative device management (DDM) OS updates and profiles
- Platform Single Sign-on during new Mac setup (coming soon)

All new features go through Fleet's [prioritization process](https://fleetdm.com/handbook/company/product-groups#how-feature-requests-are-prioritized). Excited about a new feature in macOS Tahoe? You can file a [feature request](https://github.com/fleetdm/fleet/issues/new?template=feature-request.md).

## MDM migration with Apple Business (AB)

With macOS Tahoe 26, iOS 26, and iPadOS 26, Apple introduces Device Management Migration: an improved workflow for migrating devices from one management service (MDM) to another. Learn more about configuring and the end user experience in the [Apple docs](https://support.apple.com/guide/deployment/migrate-managed-devices-dep4acb2aa44/web).

If you're planning a macOS migration with the Tahoe workflow, here are the best practices:

- Migrate devices in batches of around 100.
- Alternate batches between Mondays and Wednesdays.
- Set a 1–2 week deadline for each batch.
- Before the first batch, export a device list with each end user's name.
- Group devices into batches in a spreadsheet so stakeholders can plan around travel or other conflicts.

This approach helps limit support tickets, though batch size and frequency can be adjusted to fit your organization's needs.

## Declarative device management (DDM) OS updates and profiles

Apple continues to expand declarative device management (DDM) across macOS, iOS, and iPadOS. Fleet supports [OS updates](https://fleetdm.com/guides/enforce-os-updates) via DDM and [custom declaration (DDM) profiles](https://fleetdm.com/guides/custom-os-settings).

## Platform Single Sign-on during new Mac setup

In macOS Tahoe 26, Apple expanded Platform Single Sign-On (Platform SSO) with a new option for users to enter their credentials as a Setup Assistant step during Automated Device Enrollment. Instead of a local account being created first and enabling Platform SSO afterward, users can now authenticate with their identity provider as the first setup step. This creates a local account on the Mac linked to the user's identity that's kept in sync with their IdP credentials.

This new enrollment workflow is coming soon.

<meta name="category" value="announcements">
<meta name="authorFullName" value="Andrey Kizimenko">
<meta name="authorGitHubUsername" value="AndreyKizimenko">
<meta name="publishedOn" value="2025-09-15">
<meta name="articleTitle" value="Fleet supports Apple’s latest operating systems: macOS Tahoe 26, iOS 26, and iPadOS 26">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-supports-macos-26-tahoe-ios-26-and-ipados-26-1060x707@2x.jpg">
<meta name="description" value="Fleet is pleased to announce full support for macOS Tahoe 26, iOS 26, and iPadOS 26.">
