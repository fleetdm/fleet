# Deploy printers with Fleet

You can deploy printers to your hosts using the same scripts and configuration profiles you'd use for any other setting. Which one to use depends on the platform and whether end users should get a choice of printer.

## Before you start

- Fleet, with hosts enrolled on the platforms you're deploying to.
- Admin or maintainer access to the fleet you're deploying to.
- `fleetd` deployed with scripts enabled for macOS, Windows, and Linux. This happens automatically for hosts with MDM turned on. Otherwise, see [enable scripts](https://fleetdm.com/guides/scripts#enable-scripts).
- [Fleet Premium](https://fleetdm.com/pricing) to offer a printer in self-service or target it with labels. Running a script on demand or via policy automation works on Fleet Free.
- The printer's connection details: its hostname or IP address, and the queue or resource path it prints to.

## Deploy by platform

- [Deploy printers on macOS with Fleet](https://fleetdm.com/guides/deploy-printers-on-macos): a self-service script end users can pick from, or a configuration profile that installs a fixed printer list.
- [Deploy printers on Windows with Fleet](https://fleetdm.com/guides/deploy-printers-on-windows): a self-service script using the built-in IPP driver, or a vendor driver for printers that need one.
- [Deploy printers on Linux with Fleet](https://fleetdm.com/guides/deploy-printers-on-linux): a self-service script using CUPS.
- [Add AirPrint printers on iOS and iPadOS with Fleet](https://fleetdm.com/guides/deploy-printers-on-ios-and-ipados): a configuration profile that lists AirPrint-capable printers. iOS and iPadOS can't install a driver for anything else.
- [Add print support on Android with Fleet](https://fleetdm.com/guides/deploy-printers-on-android): deploy a print service app instead of a specific printer, since Android has no concept of installing an individual printer, MDM or otherwise.

<meta name="articleTitle" value="Deploy printers with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Deploy printers to macOS, Windows, Linux, iOS, iPadOS, and Android hosts using Fleet's existing scripts and configuration profiles.">
