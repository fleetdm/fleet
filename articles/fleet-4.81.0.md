# Fleet 4.81.0 | Lower AWS costs, automatic IdP deprovisioning, and more...


Fleet 4.81.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.81.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Lower AWS costs
- Automatic IdP deprovisioning
- Okta as a certificate authority (CA) with a dynamic challenge
- Proxy-ready Fleet Desktop configuration
- Windows profiles behave like macOS

### Lower AWS costs

Fleet now supports [gzip compression](https://developer.mozilla.org/en-US/docs/Glossary/gzip_compression) on agent API responses, reducing outbound bandwidth from your Fleet server. For Fleet users who self-host, that means lower AWS costs with no workflow changes. Currently, gzip compression is currently off by default but will soon default to on. Learn how to [turn on compression](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-gzip-responses).

### Automatic IdP deprovisioning

When a user is removed from your identitify provider (IdP), like Okta, they’re now automatically removed from Fleet by default. No configuration changes needed. Security Engineers no longer need to worry about dangling admin accounts. IT Admins get cleaner offboarding and fewer manual access reviews.

### Okta as a certificate authority (CA) with a dynamic challenge

Fleet now supports dynamic challenges when deploying certificates for Okta Verify. Each host gets a unique secret at enrollment, strengthening security. 

To configure Okta as a CA, in Fleet, head to **Settings > Integrations > Certificate authorities**, select **Add CA**, and choose **Okta CA or Microsoft Device Enrollment service (NDES)**. Okta uses NDES under the hood. If you're using static challenges with Okta's CA, choose **Custom Simple Certificate Enrollment Protocol (SCEP)** instead.

### Proxy-ready Fleet Desktop configuration

For users that self-host Fleet, you can now configure an [alternative URL](https://fleetdm.com/guides/enroll-hosts#alternative-browser-host) for Fleet Desktop. IT Admins can route traffic through a custom proxy for added control.

### Windows profiles behave like macOS

Just like macOS, Windows [configuration profiles](leetdm.com/guides/custom-os-settings) now apply payloads individually. If one payload fails, the rest still succeed, bringing consistency across platforms. IT Admins get faster enforcement of critical controls without waiting on edge-case fixes.

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.81.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-02-18">
<meta name="articleTitle" value="Fleet 4.81.0 | Lower AWS costs, automatic IdP deprovisioning, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.81.0-1600x900@2x.png">