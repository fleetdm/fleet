# Install App store apps

_Available in Fleet Premium_

In Fleet, you can install Apple App Store apps on your macOS, iOS, and iPadOS hosts. 

You can also manage which Google Play Store apps are available for self-serivce in your end user's Android work profiles.

## Apple App Store

* **MDM features**: to use the VPP integration, you must first enable MDM features in Fleet. See the [MDM setup guide](https://fleetdm.com/docs/using-fleet/mdm-setup) for instructions on enabling MDM features.

## Google Play Store

## Add app

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Select **Add software > App store** and choose a platform.

3. To add Apple App Store (VPP) apps to Fleet, you must first purchase them through Apple Business Manager (ABM), even if they are free. Learn how in [Apple's documentation](https://support.apple.com/guide/apple-business-manager/select-and-buy-content-axmc21817890/web).

4. To add Google Play Store (Android) apps, head to the [Google Play Store](https://play.google.com/store/apps), find the app, and copy the ID at the end of the URL (e.g. "com.android.chrome")

## Edit or remove app

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Search for the app you want to remove and select the app to head to it's **Software detail**s** page.

3. To edit the app, on the **Software details** page, select the pencil (edit) icon.

4. To remove the app, on the **Software details** page, select the trash can (delete) icon.

## Install app

Apple App Store (VPP) apps can be installed manually on each host's Host details page. For macOS apps, apps can also be installed via self-service on the end user's **Fleet Desktop > My device** page or [automatically via policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet).

Currently, Android apps can only be installed via self-service in the end user's managed Google Play Store (work profile).

Currently, Apple App Stpre (VPP) apps can't be uninstalled via Fleet.

## Install an app via self-service

1. **Open Fleet from the host**: On the host that will be installing an application through self-service, click on the Fleet Desktop tray icon, then click **My Device**. This will open the browser to the device's page on Fleet.

2. **Navigate to the self-service tab**: Click on the **Self-Service** tab under the device's details.

3. **Locate the app and click install**: Scroll through the list of software to find the app you would like to install, then click the **Install** button underneath it.

## API and GitOps

Fleet also provides a REST API for managing App store apps programmatically. Learn more in the API [reference docs](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app).

To manage App Store apps using Fleet's best practice GitOps, check out the `app_store_apps` key in [the GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#app-store-apps).

<meta name="articleTitle" value="Install App store apps">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-02-28">
<meta name="articleImageUrl" value="../website/assets/images/articles/install-vpp-apps-on-macos-using-fleet-1600x900@2x.png">
<meta name="description" value="This guide will walk you through installing Apple App Store and Google Play Store apps on macOS, iOS, iPadOS, and Android hosts.">
