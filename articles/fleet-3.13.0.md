# Fleet 3.13.0

![Fleet 3.13.0](../website/assets/images/articles/fleet-3.13.0-cover-1600x900@2x.jpg)

Fleet 3.13 is now available on GitHub and Docker Hub! 3.13 introduces improved performance of the additional queries feature, improvements to the fleetctl preview experience, and more.

For the complete summary of changes and release binaries check out the [release notes](https://github.com/fleetdm/fleet/releases/tag/3.13.0) on GitHub.

## Improved performance of the additional queries feature

The additional queries feature in Fleet allows you to add host data to the response payload of the `/hosts` Fleet API endpoint. This feature is helpful when you want to grab specific host information straight from the Fleet API instead of your logging destination. Check out the [Fleet documentation](https://github.com/fleetdm/fleet/blob/7fd439f812611229eb290baee7688638940d2762/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options) for more information on using the additional queries feature.

A Fleet customer reported that their Fleet’s MySQL database experienced a negative performance impact when using the additional queries feature. Fleet 3.13 improves the way the additional host data is stored in order to support Fleet deployments with hundreds of thousands of hosts.

## Improvements to fleetctl preview experience

The `fleetctl preview` command is useful for spinning up Fleet in a local environment to check out new features. This release of Fleet introduces the `fleetctl preview stop` and `fleetctl preview reset` commands.

These commands allow you to either stop or reset the simulated machines running in Docker. Now, you can shut down fleetctl preview without navigating through Docker desktop.

---

## Ready to update?

Visit our [update guide](https://fleetdm.com/docs/using-fleet/updating-fleet) in the Fleet docs for instructions on updating to Fleet 3.13.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2021-06-04">
<meta name="articleTitle" value="Fleet 3.13.0">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-3.13.0-cover-1600x900@2x.jpg">