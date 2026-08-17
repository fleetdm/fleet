# Add print support on Android with Fleet

You can add print support to Android hosts by deploying a print service app. Android has no concept of installing an individual printer, through Fleet or any other MDM, since printing on Android depends entirely on a print service app that discovers printers on the network on its own. Deploying that app is the whole task.

> **Note:** The Android Management API does have a `printingPolicy` setting, but it only allows or blocks printing outright. It doesn't configure or install a specific printer, so it doesn't help here.

## Prerequisites

- Fleet, with Android hosts enrolled and [Android MDM turned on](https://fleetdm.com/guides/android-mdm-setup).
- Admin or maintainer access to the fleet you're deploying to.
- [Fleet Premium](https://fleetdm.com/pricing) to scope the app to a subset of hosts with labels.
- The printer manufacturer's print service app, or use [Mopria Print Service](https://play.google.com/store/apps/details?id=org.mopria.printplugin) (`org.mopria.printplugin`) for broad printer support. A manufacturer's own app is often a better choice than Mopria when the printer supports it, since it can expose features (finishing options, ink levels, and similar) that a generic print service can't. Common ones:
  - [HP Print Service Plugin](https://play.google.com/store/apps/details?id=com.hp.android.printservice) (`com.hp.android.printservice`)
  - [Brother Print Service Plugin](https://play.google.com/store/apps/details?id=com.brother.printservice) (`com.brother.printservice`)
  - [Canon Print Service](https://play.google.com/store/apps/details?id=jp.co.canon.android.printservice.plugin) (`jp.co.canon.android.printservice.plugin`)
  - [Epson Print Enabler](https://play.google.com/store/apps/details?id=com.epson.mobilephone.android.epsonprintserviceplugin) (`com.epson.mobilephone.android.epsonprintserviceplugin`)

## Add the print service app

1. In Fleet, go to **Software**, choose a fleet, and select **Add software > App store**.
2. Choose the Android platform, then enter the app ID.
3. Select **Actions > Edit software** after adding it and check **Self-service**, so it appears in the end user's managed Google Play Store for them to install.

> **Note:** Fleet doesn't support automatic installation (pushing an app to every host without end user action) for Android yet. See [this tracking issue](https://github.com/fleetdm/fleet/issues/36424) for status. Until then, self-service is the way to get this app onto hosts without installing it one at a time.

In GitOps:

```yaml
software:
  app_store_apps:
    - app_store_id: "org.mopria.printplugin"
      platform: android
      self_service: true
```

Once the print service app is installed and enabled, printers on the network appear automatically in the Android print dialog. There's no per-printer setup in Fleet.

## Verify

1. On the host's **Host details** page in Fleet, open the **Software** tab and confirm the app shows as installed.
2. On the host, open the **Print** option from any app and confirm the printer appears in the list.
3. Print a test page from the host to confirm the connection works, not just that the app installed.

## Troubleshoot

**The app installs but no printers show up**

Confirm the host and the printer are on the same network. Most print service apps, including Mopria, discover printers over local network broadcast, so the printer won't appear if the host is on a guest network or a separate VLAN.

**The printer appears but jobs fail**

Some manufacturer print service apps need the printer added to Wi-Fi Direct or a specific pairing mode first. Check the manufacturer's own setup instructions for the app, since this varies by printer.

## Further reading

- [Install app store apps](https://fleetdm.com/guides/install-app-store-apps): adding and managing Google Play Store apps in Fleet.
- [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet): the same task on other platforms.

<meta name="articleTitle" value="Add print support on Android with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-17">
<meta name="description" value="Add print support to Android hosts by deploying a print service app like Mopria Print Service.">
