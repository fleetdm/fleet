# Install app store apps

_Available in Fleet Premium_

In Fleet, you can install Apple App Store apps on your macOS, iOS, and iPadOS hosts. To do this, you must first [turn on Apple MDM](https://fleetdm.com/guides/apple-mdm-setup#turn-on-apple-mdm) and Apple's [Volume Purchasing Program (VPP)](https://fleetdm.com/guides/apple-mdm-setup#volume-purchasing-program-vpp).

You can also manage which Google Play Store apps are available for self-serivce in your end user's Android work profiles. Google only allows free Google Play Store apps. [Paid apps aren't supported](https://www.androidenterprise.community/discussions/conversations/distributing-paid-apps/653).

Currently, Fleet only supports Apple App Store apps from the United States (US) region. If the app is listed on the [Apple App Store](https://apps.apple.com/) and it has `/us` in the URL (e.g. https://apps.apple.com/us/app/slack/id618783545) then it's supported.

## Add app

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Select **Add software > App store** and choose a platform.

3. To add Apple App Store (VPP) apps to Fleet, you must first purchase them through Apple Business Manager (ABM), even if they are free. Learn how in [Apple's documentation](https://support.apple.com/guide/apple-business-manager/select-and-buy-content-axmc21817890/web).

4. To add Google Play Store (Android) apps, head to the [Google Play Store](https://play.google.com/store/apps), find the app, and copy the ID at the end of the URL (e.g. "com.android.chrome")

## Edit or remove app

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Search for the app you want to remove and select the app to head to its **Software details** page.

3. To edit the app's name or icon, click the Actions button and select **Edit appearance**.

4. To configure the app's self-service categories or change which hosts can install the app, click the Actions button and select **Edit software**.

5. To remove the app, click the trash can (delete) icon above the table of installed, pending, and failed hosts.

## Configure automatic updates for an app (iOS / iPadOS only)

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Search for the app you want to configure and select the app to head to its **Software details** page.

3. Click the **Actions** button and select **Schedule auto updates**.

4. In the modal dialog that opens, click **Enable auto updates** and configure an update window of at least one hour. You may also choose to limit which hosts receive auto-updates using the Target configuration (this configuration will also affect which hosts can install the app themselves via the self-service feature).

## Install app

Apple App Store (VPP) apps can be installed manually on each host's Host details page. For macOS apps, apps can also be installed via self-service on the end user's **Fleet Desktop > My device** page or [automatically via policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet).

Currently, Android apps can only be installed via self-service in the end user's managed Google Play Store (work profile).

Currently, Apple App Store (VPP) apps can't be uninstalled via Fleet.

> VPP apps on iOS/iPadOS will be uninstalled when the host is unenrolled from MDM.

## API and GitOps

Fleet also provides a REST API for managing app store apps programmatically. Learn more in the API [reference docs](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app).

To manage App Store apps using Fleet's best practice GitOps, check out the `app_store_apps` key in [the GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#app-store-apps).

<meta name="articleTitle" value="Install app store apps">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-02-28">
<meta name="description" value="This guide will walk you through installing Apple App Store and Google Play Store apps on macOS, iOS, iPadOS, and Android hosts.">
