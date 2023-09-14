# ChromeOS
For visibility on ChromeOS hosts, Fleet provides the fleetd Chrome extension which provides similar functionality as osquery on other operating systems.

## Adding ChromeOS hosts to Fleet

To learn how to add ChromeOS hosts to Fleet, visit [here](https://fleetdm.com/docs/using-fleet/adding-hosts#add-chromebooks-with-the-fleetd-chrome-extension).

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

## Potential issues and troubleshooting: 
- 1 The extension does not install on our ChromeBooks.
- 2 Chrome Web Browsers on other OSs (Mac/Linux/Windows) get this extension (where it's not needed).

Google Admin is arranged in a hierarchy of Organizational Units (OUs) tree. Each of the OUs can hold a combination of USERs and/or DEVICEs.
Chrome extensions can be set for a specific OU (force-installed, allow install or block). However, Extensions can only be set at USERs level (not DEVICES).
If a chrome extension is deployed to an OU that only has DEVICES, it will not be installed. 
On the other hand if you deploy an extension to an OU that hold USERS with both ChromeBooks and
managed Chrome web browsers (e.g. Chrome browser on a MacBook), it will deploy the extension to that Chrome Web Browser.

### Our recommendation: 
- Create an OU that will hold all USERs with ChromeBooks. Deploy our extension to it (Force-Install).
- Create an OU to holds the managed Chrome Web Browsers of the USERS above (Not the USERS. Just the Chrome Web Brwosers). Make sure our extension is blocked on this OU. 

> Note: When deployed on OSs other than ChromeOS, our Chrome Extension will detect it and not perform any operation.  


<meta name="title" value="Enroll Chromebooks">
<meta name="pageOrderInSection" value="2000">
<meta name="navSection" value="Dig deeper">
