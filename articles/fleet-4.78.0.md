# Fleet 4.78.0 | iOS and Android self-service, cross-platform certificate deployment, and more...

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.78.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.78.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Self-service software on iOS and Android
- Install work apps on corporate iOS and Android during enrollment
- Cross-platform certificate deployment
- Okta conditional access

### Self-service software on iOS and Android

You can now offer self-service app access on both iOS/iPadOS and Android. Deploy a web-based self-service portal to iPhones ([learn how](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-and-ipados)) and surface approved Play Store apps in managed Google Play. 

This means you can now offer self-service software on macOS, Windows, Linux, iOS/iPadOS, and Android hosts. Learn more about [self-service software](https://fleetdm.com/guides/software-self-service).

### Install work apps on corporate iOS and Android during enrollment

You can now install managed work apps like Slack, Gmail, Zoom, and GlobalProtect during enrollment on personally-owned iOS/iPadOS and Android hosts. Apps are installed as managed, giving you control over corporate data while respecting user privacy. Learn more about installing software during [new host setup](https://fleetdm.com/guides/setup-experience).

### Cross-platform certificate deployment

You can now install certificates from any [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) certificate authority to the user scope on Windows hosts. This helps you connect your end users to Wi-Fi, VPN, and other tools.

Also, you can now deploy SCEP certificates to Android devices using Fleet's [best practice GitOps](). Support for deploying Android certificates using Fleet's UI is coming soon.

This means you can now install certificates on macOS, Windows, Linux, iOS/iPadOS, and Android hosts. See all certificate authorities supported by Fleet in [the guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Okta conditional access

Fleet now supports [Okta for conditional access](https://fleetdm.com/guides/okta-conditional-access-integration). This allows IT and Security teams to block third-party app logins when a host is failing one or more policies.

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.78.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-12-19">
<meta name="articleTitle" value="Fleet 4.78.0 | iOS and Android self-service, cross-platform certificate deployment, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.78.0-1600x900@2x.png">
