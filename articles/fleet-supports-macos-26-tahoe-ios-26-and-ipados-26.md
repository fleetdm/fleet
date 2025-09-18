# Fleet supports Apple’s latest operating systems: macOS Tahoe 26, iOS 26, and iPadOS 26 

![Fleet supports Apple’s latest operating systems: macOS Tahoe 26, iOS 26, and iPadOS 26](../website/assets/images/articles/fleet-supports-macos-26-tahoe-ios-26-and-ipados-26.jpg)

_Photo by [MariuszBlach](https://www.istockphoto.com/photo/lake-tahoe-gm480641071-36497954)_

With Apple's releases of macOS Tahoe 26, iOS 26, and iPadOS 26, Fleet provides same-day support for IT teams to upgrade immediately so devices will remain secure and fully managed. This means that all existing Fleet features are tested and bugs are fixed. 

Also, Fleet will support these new features:
- MDM migration with Apple Business Manager (ABM)
- Declarative device management (DDM) OS updates and profiles
- Platform Single Sign-on during new Mac setup (coming soon)

All new features go through Fleet's [prioritization process](https://fleetdm.com/handbook/company/product-groups#how-feature-requests-are-prioritized). Excited about a new feature in macOS Tahoe? You can file a [feature request](https://github.com/fleetdm/fleet/issues/new?template=feature-request.md).

## MDM migration with Apple Business Manager (ABM)

With MacOS Tahoe 26, iOS 26, and iPadOS 26, Apple introduces Device Management Migration: an improved workflow for migrating devices from one management service (MDM) to another. 

In Apple Business Manager (ABM), admins can assign devices to a new MDM server and set a migration deadline. Users receive clear notifications that enrollment into a new management service is required. If users do not act before the migration deadline, enrollment into the new MDM is enforced automatically, eliminating the need for device wipes, scripts, or manual workarounds that previously made MDM migrations more difficult, complex and time-consuming.

Fleet is ready to support this migration workflow, making it easier for organizations to migrate devices with minimal disruption. Learn more about configuring migration in the [Apple docs](https://support.apple.com/guide/deployment/migrate-managed-devices-dep4acb2aa44/web).

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
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-supports-macos-26-tahoe-ios-26-and-ipados-26.jpg">
<meta name="description" value="Fleet is pleased to announce full support for macOS Tahoe 26, iOS 26, and iPadOS 26.">
