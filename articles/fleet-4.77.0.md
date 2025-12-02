# Fleet 4.77.0 | iOS/iPadOS self-service and enterprise packages, edit IdP username, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/2oLyaV7rIXM?si=FgWAi9K8KhEXWd_f" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.77.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.77.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- iOS/iPadOS self-service
- Deploy enterprise iOS/iPadOS packages
- Edit IdP username
- Enforce authentication during enrollment
- Connect end users on Windows and Linux to Wi-Fi/VPN
- Activity for deleted hosts
- More Fleet-maintained apps

### iOS/iPadOS self-service

You can deploy a self-service web app to your iOS/iPadOS hosts. This gives end users a simple way to install managed apps on their own, reducing IT load and speeding up access. Learn more in the [self-service guide](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-ipados).

### Deploy enterprise iOS/iPadOS packages

You can now deploy enterprise (`.ipa`) packages to iPhones and iPads using Fleet’s [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#software) and [API](https://fleetdm.com/docs/rest-api/rest-api#add-package). Perfect for distributing pre-release internal apps to testers or employees.

### Edit IdP username

You can now update a host’s identity provider (IdP) username directly from the Fleet UI (**Host details** page) or [API](https://fleetdm.com/docs/rest-api/rest-api#update-human-device-mapping). This makes it easier to maintain [human-to-host mapping](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts), especially if you don't require end user authentication during new host setup.

### Enforce authentication during enrollment

You can now require end users to authenticate with your IdP before Fleet installs software or runs policies on company-owned Windows and Linux setup. This ensures only authenticated users get access to company resources. Learn more in the [Windows and Linux setup guide](https://fleetdm.com/guides/windows-linux-setup-experience#end-user-authentication).

Also, you can require end users to authenticate when turning on and/or enrolling a Mac via profile-based device enrollment. [Learn more](https://fleetdm.com/guides/apple-mdm-setup#manual-enrollment).

### Connect end users on Linux and Windows to Wi-Fi/VPN

You can deploy certificates from any certificate authority (CA) that supports [Simple Certificate Enrollment Protocol (SCEP)](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificates to Windows hosts. This enables access to corporate Wi-Fi and VPN resources. [Learn how](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#custom-scep-simple-certificate-enrollment-protocol). Currently, certificates can only be deliverd to the host's device scope. User scope is coming soon. 

Also, you can now deliver certificates from any CA that supports [Enrollment over Secure Protocol (EST)](https://en.wikipedia.org/wiki/Enrollment_over_Secure_Transport) certificates to Linux hosts. This way, you can connect end users to Wi-Fi, VPN, or internal tools. [Learn how](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#custom-est-enrollment-over-secure-transport).

### Activity for deleted hosts

Deleted host events are now included in the activity feed. If a host is removed, you’ll still have a record for audits and historical tracking.

### More Fleet-maintained apps

Fleet added [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) for macOS (Claude, ChatGPT, Outlook, Webex, Spotify) and Windows (Slack, Zoom, Firefox), plus many more apps. See all Fleet-maintained apps in the [software catalog](https://fleetdm.com/software-catalog).

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.77.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-11-24">
<meta name="articleTitle" value="Fleet 4.77.0 | iOS/iPadOS self-service and enterprise packages, edit IdP username, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.77.0-1600x900@2x.png">
