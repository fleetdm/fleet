# ChromeOS

## Adding ChromeOS hosts to Fleet
Fleet provides a Chrome extension which you can install via Google Admin.

To learn how to add ChromeOS hosts to Fleet, visit [here](https://fleetdm.com/docs/using-fleet/adding-hosts#add-chromebooks-with-the-fleetd-chrome-extension).

## Available tables
To see the available tables for ChromeOS, visit [here](https://fleetdm.com/tables/chrome_extensions?platformFilter=chrome).

## Setting the hostname
By default, the hostname for a Chromebook host will be blank. The hostname can be customized in Google Admin by configuring the ["device network hostname template"](https://support.google.com/chrome/a/answer/1375678#zippy=%2Cdevice-network-hostname-template%2Creport-device-os-information).

## Current Limitations in ChromeOS
- Scheduled queries are currently not available in ChromeOS
- The Fleetd Chrome extension must be force-installed by enterprise policy in order to have full access to the host's data.
- More tables will be added in https://github.com/fleetdm/fleet/issues/11037

## Debugging ChromeOS
To learn how to debug the Fleetd Chrome extension, visit [here](https://fleetdm.com/docs/contributing/testing-and-local-development#fleetd-chrome-extension).


<meta name="pageOrderInSection" value="2000">
