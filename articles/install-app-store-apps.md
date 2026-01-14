# Install app store apps

_Available in Fleet Premium_

In Fleet, you can install Apple App Store apps on your macOS, iOS, and iPadOS hosts, including custom apps.

You can also manage which Google Play Store apps are available for self-service in your end user's Android work profiles. Google only allows free Google Play Store apps. [Paid apps aren't supported](https://www.androidenterprise.community/discussions/conversations/distributing-paid-apps/653).

## Add an app

### Apple (VPP)

> Before using Fleet to manage VPP apps, you must first [turn on Apple MDM](https://fleetdm.com/guides/apple-mdm-setup#turn-on-apple-mdm) and Apple's [Volume Purchasing Program (VPP)](https://fleetdm.com/guides/apple-mdm-setup#volume-purchasing-program-vpp). Once you've completed that setup, you can follow the directions below for each app.

1. Purchase the relevant app through Apple Business Manager (ABM). You must perform this step even if the app is free, or if it is a custom app you own. Learn how in [Apple's documentation](https://support.apple.com/guide/apple-business-manager/select-and-buy-content-axmc21817890/web).

2. In Fleet, head to the **Software** page and select a team in the teams dropdown.

3. Select **Add software > App store**, then select the app you just purchased.

> Currently, Fleet only supports Apple App Store apps from the United States (US) region. If the app is listed on the [Apple App Store](https://apps.apple.com/) and it has `/us` in the URL (e.g. https://apps.apple.com/us/app/slack/id618783545) then it's supported.

### Google Play (Android)

> Before using Fleet to manage Google Play Store apps, you must first [turn on Android MDM](https://fleetdm.com/guides/android-mdm-setup). Once you've completed that setup, you can follow the directions below for each app.

1. Head to the [Google Play Store](https://play.google.com/store/apps), find the app, and copy the ID at the end of the URL (e.g. "com.android.chrome")

2. In Fleet, head to the **Software** page and select a team in the teams dropdown.

3. Select **Add software > App store**, choose the Android platform, then enter the application ID.

## Edit or remove an app

1. In Fleet, head to the **Software** page and select a team in the teams dropdown.

2. Search for the app you want to remove and select the app to head to it's **Software detail**s** page.

3. To edit the app, on the **Software details** page, select the pencil (edit) icon.

4. To remove the app, on the **Software details** page, select the trash can (delete) icon.

## Install an app

### Apple (VPP)

Apps can be installed manually on each host's **Host details** page. For macOS apps, apps can also be installed via self-service on the end user's **Fleet Desktop > My device** page or [automatically via policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet).

Currently, Apple App Stpre (VPP) apps can't be uninstalled via Fleet.

### Google Play (Android)

Android apps can be installed via self-service in the end user's managed Google Play Store (work profile).

## API and GitOps

Fleet also provides a REST API for managing app store apps programmatically. Learn more in the API [reference docs](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app).

To manage App Store apps using Fleet's best practice GitOps, check out the `app_store_apps` key in [the GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#app-store-apps).

<meta name="articleTitle" value="Install app store apps">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-08">
<meta name="description" value="This guide will walk you through installing Apple App Store and Google Play Store apps on macOS, iOS, iPadOS, and Android hosts.">
