# Fleet 4.26.0 | Easier osquery extensions, external audit log destinations, and cleaner data lakes

![Fleet 4.26.0](../website/assets/images/articles/fleet-4.26.0-1600x900@2x.png)

Fleet 4.26.0 is up and running. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.26.0) or continue reading to get the highlights.

For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Manage osquery extensions with fleetd.
- Log user activity for audits.
- Ingest the latest software data.

## Manage osquery extensions with fleetd
**Available in Fleet Premium**

Fleetd used to only deploy and upgrade osquery and Fleet Desktop on employees’ machines. But many Fleet users require osquery extensions to suit their situations. That meant managing extensions separately with a tool like Munki or a mobile device management (MDM) system.

Fleet 4.26.0 brings the deployment and management of extensions into fleetd — saving you the time and energy it would take to maintain extensions with a separate interface.

Fleetd checks the extension set at a configurable interval (once an hour by default). The osquery versions and extensions specified by your system define the extension set. If the extension differs from the current set (e.g., additions, upgrades, or removals), fleetd will install, upgrade, or delete the appropriate extensions.

Fleetd also checks which team a machine belongs to and applies that team’s extension set. If no team configuration exists, fleetd applies the global extension set. Team extension sets override global sets. Fleetd doesn’t merge global and team options, which was the case before Fleet 4.26.0.

Here’s how to manage extensions with fleetd:

1. Upload new extensions and extension versions to your own TUF server. The TUF server is updated outside of the fleetctl or the Fleet UI.
2. Update the list of extensions by applying a new YAML configuration to your Fleet instance. This can be done by applying a new configuration file from fleetctl or using the agent options pages in the Fleet UI.
3. Make sure each `extensions` object has a `name` and a `channel` attribute in the YAML file.
4. You can specify a specific version number for the extension that matches an identifier in your TUF server. If no version is specified, then fleetd will upgrade to the latest version of that extension available in your TUF server.
5. Fleetd supports all extension types, including Python. But Python extensions must be fully compiled into a binary. Fleetd doesn’t manage Python dependencies.

If an extension fails to apply, fleetd will apply the other extensions and then start osquery with the reduced extension set.

## Log user activity for audits
**Available in Fleet Premium**

Security and IT administrators have long to-do lists and short deadlines. Increasing access to Fleet across the company would help lighten the workload, but more users could mean more chances for things to fall through the cracks. Fleet 4.26.0 gives you extra confidence to extend your user base. 

Now you can stream Fleet user activities to external destinations, aggregating granular data for greater insights in the event issues occur.

To make sure administrative operations run smoothly, Fleet streams activity to log destinations asynchronously. Activity will still appear in the Fleet UI in real time, but streaming this data may take up to 5 minutes.

## Ingest the latest software data
**Available in Fleet Free and Fleet Premium**

You already have a lot of raw data to sift through in your data lake, especially if your organization has hundreds of thousands of devices. What if you could refine your software data before it reaches the lake?

Fleet 4.26.0 reduces the number of calls you have to make to pull software data with the REST API. Each time a host has software added, updated, or deleted, a `host_software_updated_at` timestamp gets updated for that host. The `host_software_updated_at` timestamp is exposed through the API. This lets you send the latest software data to your data lake, so you can avoid drowning in outdated information.

## Fleet MDM
**MDM features are not ready for production and are currently in development. These features are disabled by default.**

Fleet is building a cross-platform MDM to give IT and security teams the visibility and openness they need. Here are the latest developments:

- Added functionality to ingest device information from Apple MDM endpoints so that a device ordered in Apple Business Manager can be surfaced in Fleet.
- Added new activities to the activities API when a device has MDM is turned on or off..
- Added option to filter hosts by MDM status "pending" to surface devices ordered through Apple Business Manager that are still pending enrollment to Fleet.
- Added a flag to indicate if the Apple Business Manager terms and conditions have changed and must be accepted to have automatic enrollment of hosts work again. A banner is added to the output of `fleetctl` commands when this is the case.
- Added side navigation layout to the integration page and conditionally show MDM section.
- Added a modal to allow users to download an enrollment profile required for turning on MDM.
- Added a new configuration option to set the default team for Apple Business Manager.

Are you interested in the Fleet MDM beta? [Schedule a call](https://calendly.com/fleetdm/demo) to save your spot.

## More new features, improvements, and bug fixes
- Added locally-formatted datetime tooltips.
- Added the ability to bookmark a url when it includes the query parameter on the Hosts page.
- Added a way to override a detail query or disable it through app config.
- Added a software_updated_at column denoting when software was updated for a host.
- Updated software empty states.
- Updated all forms to automatically focus on the first entry for better UX.
- Updated the Fleet UI to show pack target details on the right side of the dropdown.
- Updated Fleet UI buttons to follow the new style guide.
- Fixed ingestion of MDM data with empty server URLs (meaning the host is not enrolled to an MDM server).
- Fixed a bug in which Fleet would error when the host doesn’t have MDM data.
- Fixed an issue in which invalid query strings stopped the spinner from timing out.

## Ready to upgrade?
Visit our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.26.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2023-01-16">
<meta name="articleTitle" value="Fleet 4.26.0 | Easier osquery extensions, external audit log destinations, and cleaner data lakes">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.26.0-1600x900@2x.png">
