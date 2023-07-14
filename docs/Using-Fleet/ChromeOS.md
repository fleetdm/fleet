# ChromeOS
For visibility on ChromeOS hosts, Fleet provides the fleetd Chrome extension which provides similar functionality as osquery on other operating systems.

## Adding ChromeOS hosts to Fleet

To learn how to add ChromeOS hosts to Fleet, visit [here](https://fleetdm.com/docs/using-fleet/adding-hosts#add-chromebooks-with-the-fleetd-chrome-extension).

> The fleetd Chrome browser extension is supported on ChromeOS operating systems that are managed using [Google Admin](https://admin.google.com). It is not intended for non-ChromeOS hosts with the Chrome browser installed.

## Available tables
To see the available tables for ChromeOS, visit [here](https://fleetdm.com/tables/chrome_extensions?platformFilter=chrome).

## Setting the hostname
By default, the hostname for a Chromebook host will be blank. The hostname can be customized in Google Admin under Devices > Chrome > Settings > Device > Device Settings > Other Settings > [Device network hostname template](https://support.google.com/chrome/a/answer/1375678#zippy=%2Cdevice-network-hostname-template%2Creport-device-os-information).

## Current Limitations in ChromeOS
- Scheduled queries are currently not available in ChromeOS
- The Fleetd Chrome extension must be force-installed by enterprise policy in order to have full access to the host's data.
- More tables will be added in https://github.com/fleetdm/fleet/issues/11037

## Required access
In order to function properly, the ChromeOS extension requests permission to use the following Chrome APIs:

- activeTab
- alarms
- cookies
- enterprise.deviceAttributes
- enterprise.hardwarePlatform
- enterprise.networkingAttributes
- enterprise.platformKeys
- gcm
- history
- identity
- identity.email
- idle
- loginState
- management
- privacy
- proxy
- platformKeys
- sessions
- storage
- system.cpu
- system.display
- system.memory
- system.storage
- unlimitedStorage
- tabs

If any of these APIs are disabled, tables relying on that data will not return results as expected. 

> The [`enterprise.hardware_platform` API](https://chromeenterprise.google/policies/#EnterpriseHardwarePlatformAPIEnabled) is disabled by default and must be explicitly enabled. Without this API, the extension cannot gather hardware information from the host. 

## Debugging ChromeOS
To learn how to debug the Fleetd Chrome extension, visit [here](https://fleetdm.com/docs/contributing/testing-and-local-development#fleetd-chrome-extension).

<meta name="title" value="ChromeOS">
<meta name="pageOrderInSection" value="2000">
