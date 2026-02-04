# Deploy custom Android apps

_Available in Fleet Premium_

In Fleet, you can deploy your own custom Android apps (APK files) to your organization's Android hosts. This is useful for distributing internal apps that aren't available on the public Google Play Store.

To deploy custom Android apps, you'll publish them as private apps in the Google Play Console, making them available only to your organization through Android Enterprise.

## Prerequisites

Before deploying custom Android apps, you must first [turn on Android MDM](https://fleetdm.com/guides/android-mdm-setup). Once you've completed that setup, you can follow the directions below.

If you don't already have a Google Play Console account, you'll need to create one. The [Google Play Console](https://play.google.com/console/signup) requires a one-time registration fee of $25.

## Add private app in Google Play Console

1. In the [Google Play Console](https://play.google.com/console), select **Home** from the left navigation.

2. Select **Create app**.

3. Enter your app details:
   - **App name**: Enter the name of your app
   - **Default language**: Select your preferred language
   - **App or game**: Select app
   - **Free or paid**: Select **Free** (private apps must be free)

4. Review and accept the Developer Program Policies and US export laws.

5. Select **Create app**.

### Configure app details

1. After creating the app, you'll be directed to the app dashboard.

2. Complete the required sections in the left navigation:
   - **Store settings**: Configure your app's store listing details
   - **Privacy policy**: Provide your privacy policy URL (required for private apps)
   - **App access**: Specify if your app requires special access or credentials
   - **Ads**: Declare whether your app contains ads
   - **Content rating**: Complete the content rating questionnaire
   - **Target audience**: Select your target age group
   - **App content**: Complete required declarations

### Make the app private

1. First, find your Android Enterprise ID in Fleet. Navigate to **Settings > Integrations > MDM > Android MDM > Edit** and copy Android Enterprise ID (e.g. LC04yu8c9).

2. In the left navigation, go to **Test and release > Advanced settings**.

3. Select **Managed Google Play**, tab on the top, and select **Turn on**.

4. Select **Add organization**, paste your Android Enterprise ID from the first step to **Organization ID** and add **Organization name**, for example "Fleet".

5. Select **Add**, then select **Save** on the bottom, and select **Make app private**.

> The app will now be private and only available to your organization through managed Google Play. It won't appear in the public Google Play Store.

### Upload your custom app package

1. In the left navigation, go to **Test and release > Production**.

2. Select **Create new release**.

3. Upload your package (`.apk` or `.aab`).

4. Release name will be automatically populated after package is uploaded.

5. Select **Save** and then select **Save** on the next screen.

6. Select **Go to overview** and then send **Send 1 change for review**. To confirm select **Send changes for review**.

> The Google Play Console displays messages about app review that can take up to 7 days. However, private apps are typically available for deployment within 10 minutes and they don't go through regular Google Play Store review.

## Add the app to Fleet

After publishing your private app in the Google Play Console, you can add it to Fleet.

1. Find application ID in the Google Play Console on the **Home** page. The app ID will be in the app list under the app name. It looks like "com.yourcompany.appname".

2. In Fleet, head to the **Software** page and select a team in the teams dropdown.

3. Select **Add software > App store**, choose the Android platform, then enter the application ID.

> If your private app doesn't appear in Fleet after adding it, try again in a 10 minutes. Sometimes it takes bit more time for app to became available for Android Enterprise.
 
## Install, edit, and delete custom app

Learn how to install, edit, and delete the app in the [Install app store apps guide](https://fleetdm.com/guides/install-app-store-apps#install-an-app).

## Update new version in Google Play Console

To release a new version of your custom app, please folow the step described in [Upload your custom app package](#upload-your-custom-app-package). The process is the same as for uploading a new app.

<meta name="articleTitle" value="Deploy custom Android apps (APK)">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-02-04">
<meta name="description" value="This guide will walk you through deploying custom Android apps to your organization's Android hosts using Google Play Console and Fleet.">
