# Add AirPrint printers on iOS and iPadOS with Fleet

You can add printers to iOS and iPadOS hosts with a configuration profile. iOS and iPadOS don't support installing a printer driver, so this only works for [AirPrint-capable](https://support.apple.com/en-us/102895) printers. The AirPrint payload populates a list of AirPrint printers for the Print dialog to show, it doesn't add support for printers that aren't AirPrint-capable.

## Prerequisites

- Fleet, with iOS or iPadOS hosts enrolled.
- Admin or maintainer access to the fleet you're deploying to.
- [Fleet Premium](https://fleetdm.com/pricing) to scope the profile to a subset of hosts with labels.
- The printer's IP address or hostname and resource path. Confirm the printer supports AirPrint before you start.

## Add the printer

1. Build an AirPrint (`com.apple.airprint`) payload in [iMazing Profile Creator](https://imazing.com/profile-editor), starting from this [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/ios-ipados/configuration-profiles/airprint.mobileconfig). For each printer, add its IP address or hostname and resource path. Or skip the GUI and have an AI coding agent generate and validate the payload for you. See [Build and validate configuration profiles with AI instead of a GUI](https://fleetdm.com/guides/build-configuration-profiles-with-ai).
2. In Fleet, go to **Controls > OS settings > Configuration profiles** and choose a fleet.
3. Select **Add profile** and upload the `.mobileconfig` file.

Unlike the macOS Printing payload, [you can deliver more than one AirPrint payload](https://support.apple.com/guide/deployment/airprint-payload-settings-dep3b4cf515/web) to the same host.

> **Note:** This profile pushes to every targeted host until [self-service configuration profiles](https://github.com/fleetdm/fleet/issues/46834), planned for a future release of Fleet, ship.

In GitOps:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/ios/profiles/floor2-airprint.mobileconfig
```

## Verify

1. On the host's **Host details** page in Fleet, open the **OS settings** tab and confirm the profile shows as **Verified**.
2. On the host, open the **Print** dialog from any app and confirm the printer appears under **Printer**.
3. Print a test page from the host to confirm the connection works, not just that the profile installed.

## Troubleshoot

**The printer doesn't appear in the Print dialog**

Confirm the printer actually supports AirPrint. Many network printers only support it as an add-on firmware feature that has to be turned on in the printer's own settings. If it doesn't support AirPrint at all, this payload won't help, since it only lists AirPrint printers rather than installing a driver.

**The profile is verified but the printer doesn't respond**

Confirm the IP address or hostname and resource path against the printer's own network settings page. A profile can install successfully with a resource path that doesn't match any real print queue on the printer.

See [Configuration profiles](https://fleetdm.com/guides/custom-os-settings) for how Fleet delivers and verifies profiles, and [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet) for the same task on other platforms.

<meta name="articleTitle" value="Add AirPrint printers on iOS and iPadOS with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Add AirPrint-capable printers to iOS and iPadOS hosts with a configuration profile.">
