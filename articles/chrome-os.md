# ChromeOS
For visibility on ChromeOS hosts, Fleet provides the fleetd Chrome extension which provides similar functionality as osquery on other operating systems.

Follow the instructions in our [host enrollment guide](https://fleetdm.com/docs/using-fleet/adding-hosts#enroll-chromebooks) to add Chromebooks to Fleet.

> The fleetd Chrome browser extension is supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). It is not intended for non-ChromeOS hosts with the Chrome browser installed.

## Available tables
See our [ChromeOS tables list](https://fleetdm.com/tables/chrome_extensions?platformFilter=chrome) for available tables.

## Setting the hostname
By default, the hostname for a Chromebook host will be blank. The hostname can be customized in Google Admin under Devices > Chrome > Settings > Device > Device Settings > Other Settings > [Device network hostname template](https://support.google.com/chrome/a/answer/1375678#zippy=%2Cdevice-network-hostname-template%2Creport-device-os-information).

## Current limitations in ChromeOS
- Scheduled queries are currently not available in ChromeOS
- The Fleetd Chrome extension must be force-installed by enterprise policy in order to have full access to the host's data.
- More tables that could be added:
  - `disk_events`: https://github.com/fleetdm/fleet/issues/12405
  - `client_certificates`: https://github.com/fleetdm/fleet/issues/12465
  - `usb_devices`: https://github.com/fleetdm/fleet/issues/12780

## Debugging ChromeOS
See our [fleetd Chrome extension testing guide](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#fleetd-chrome-extension) for debugging instructions.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2023-11-21">
<meta name="articleTitle" value="ChromeOS">
<meta name="description" value="Learn about ChromeOS and Fleet.">
