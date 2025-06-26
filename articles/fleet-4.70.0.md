# Fleet 4.70.0 | Entra ID conditional access, Android work profiles, and more...

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/HxBQvlV14Lc?si=VLYS7QxPuP3TLbjG" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.70.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.70.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Entra ID conditional access
- One-time code for custom SCEP certificate authorities (CAs)
- Work profiles for personal (BYOD) Android
- Script reports
- Teams search

### Entra ID conditional access

Fleet now supports [Microsoft Entra ID for conditional access](https://fleetdm.com/guides/entra-conditional-access-integration). This allows IT and Security teams to block third-party app logins when a host is failing one or more policies.

### One-time code for custom SCEP certificate authorities (CAs)

Fleet now supports one-time code verification when requesting certificates from a custom SCEP certificate authority (CA). This adds a layer of security to ensure only hosts enrolled to Fleet can request certificates that [grant access to corporate resouces (Wi-Fi or VPNs)](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Work profiles for personal (BYOD) Android

Fleet has removed the Android MDM feature flag. IT admins can now [enroll BYOD Android hosts](https://fleetdm.com/guides/android-mdm-setup#basic-article) and see host vitals. Support for OS updates, configuration profiles, and more coming soon.

### Script reports

Fleet users can now see which hosts successfully ran a script, which errored, and which are still pending. This helps with troubleshooting and ensures scripts reach all intended hosts. Learn more about running scripts in Fleet in the [scripts guide](https://fleetdm.com/guides/scripts).

### Teams search

Users managing many [teams in Fleet](https://fleetdm.com/guides/teams) can now search in the teams dropdown. This makes it faster to navigate and switch between teams in Fleet's UI.

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.70.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-06-27">
<meta name="articleTitle" value="Fleet 4.70.0 | TODO">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.70.0-1600x900@2x.png">