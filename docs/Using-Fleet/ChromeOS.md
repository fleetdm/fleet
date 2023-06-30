# ChromeOS

## Adding ChromeOS hosts to Fleet
Fleet provides a Chrome extension which you can install via Google Admin.

> For ChromeOS hosts, the fleetd Chrome extension is installed instead of osquery. This Chrome browser extension is only supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). 

To learn how to add ChromeOS hosts to Fleet, visit [here](https://fleetdm.com/docs/using-fleet/adding-hosts#add-chromebooks-with-the-fleetd-chrome-extension).

> > The fleetd Chrome browser extension is supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). It is not intended for non-ChromeOS hosts with the Chrome browser installed.

## Available tables
To see the available tables for ChromeOS, visit [here](https://fleetdm.com/tables/chrome_extensions?platformFilter=chrome).

## Setting the hostname
By default, the hostname for a Chromebook host will be blank. The hostname can be customized in Google Admin under Devices > Chrome > Settings > Device > Device Settings > Other Settings > [Device network hostname template](https://support.google.com/chrome/a/answer/1375678#zippy=%2Cdevice-network-hostname-template%2Creport-device-os-information).

## Current Limitations in ChromeOS
- Scheduled queries are currently not available in ChromeOS
- The Fleetd Chrome extension must be force-installed by enterprise policy in order to have full access to the host's data.
- More tables will be added in https://github.com/fleetdm/fleet/issues/11037

## Debugging ChromeOS
To learn how to debug the Fleetd Chrome extension, visit [here](https://fleetdm.com/docs/contributing/testing-and-local-development#fleetd-chrome-extension).

<meta name="title" value="ChromeOS">
<meta name="pageOrderInSection" value="2000">
