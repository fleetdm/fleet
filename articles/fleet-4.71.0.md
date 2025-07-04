# Fleet 4.71.0 | IdP labels, user certificates, and more...

<div purpose="embedded-content">
   <iframe src="TODO" frameborder="0" allowfullscreen></iframe>
</div>

Fleet 4.71.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.71.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Labels based on identity provider (IdP) groups and departments
- IdP foreign vitals
- Deploy user certificates
- Verify App Store (VPP) app installation

### Labels based on identity provider (IdP) groups and departments

IT admins can now build labels based on users’ IdP groups and departments. This enables different apps, OS settings, queries, and more based on group and department. Learn how to map IdP users to hosts in the [foreign vitals guide](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).

### IdP foreign vitals

Fleet now supports using end users’ IdP department info in [configuration profile variables](https://fleetdm.com/docs/configuration/yaml-files#:~:text=In%20Fleet%20Premium%2C%20you,are%20sent%20to%20hosts). This allows IT admins to deploy a [property list](https://en.wikipedia.org/wiki/Property_list) (via configuration profile) so that third-party tools (i.e. Munki) can automate actions based on department data.

### Deploy user certificates

Fleet can now deploy and renew certificates from Microsoft Network Device Enrollment Service (NDES), DigiCert, and custom Simple Certificate Enrollment Protocol (SCEP) certificate authorities (CAs) directly to the login (user) Keychain. This makes it easier to connect employees to third-party tools that require user-level certificates. Learn more in the ["Connect end users to Wi-Fi or VPN" guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate).

### Verify App Store (VPP) app installation

Fleet now verifies that an App Store app is actually installed on a macOS host. This ensures apps are present before actions like adding them to the dock are triggered. Learn more in the ["Install App Store apps"](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet) guide. 

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.71.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2025-07-11">
<meta name="articleTitle" value="Fleet 4.71.0 | IdP labels, user certificates, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.71.0-1600x900@2x.png">