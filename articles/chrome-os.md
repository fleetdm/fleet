# ChromeOS
For visibility on ChromeOS hosts, Fleet provides the fleetd Chrome extension which provides similar functionality as osquery on other operating systems.

To learn how to add ChromeOS hosts to Fleet, visit [here](https://fleetdm.com/docs/using-fleet/adding-hosts#enroll-chromebooks).

> The fleetd Chrome browser extension is supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). It is not intended for non-ChromeOS hosts with the Chrome browser installed.

## Available tables
To see the available tables for ChromeOS, visit [here](https://fleetdm.com/tables/chrome_extensions?platformFilter=chrome).

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
To learn how to debug the Fleetd Chrome extension, visit [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Testing-and-local-development.md#fleetd-chrome-extension).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2023-11-21">
<meta name="articleTitle" value="ChromeOS">
<meta name="description" value="Learn about ChromeOS and Fleet.">
