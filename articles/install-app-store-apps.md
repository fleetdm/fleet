# Install app store apps

_Available in Fleet Premium_

In Fleet, you can install Apple App Store apps on your macOS, iOS, and iPadOS hosts, including custom apps.

You can also manage which Google Play Store apps are available for self-service in your end user's Android work profiles. Google only allows free Google Play Store apps. [Paid apps aren't supported](https://www.androidenterprise.community/discussions/conversations/distributing-paid-apps/653).

On Windows hosts, you can install Microsoft Store apps.

## Add an app

### Apple (VPP)

> Before using Fleet to manage VPP apps, you must first [turn on Apple MDM](https://fleetdm.com/guides/apple-mdm-setup#turn-on-apple-mdm) and Apple's [Volume Purchasing Program (VPP)](https://fleetdm.com/guides/apple-mdm-setup#volume-purchasing-program-vpp). Once you've completed that setup, you can follow the directions below for each app.

1. Sign in to [Apple Business](https://business.apple.com), select **Apps & Services > App Store**, and find the app. For a custom app you own, select **Custom Apps** instead.

2. Add the app to your library:
   - **Free apps**: Select **Add**. Under **Organizational Units**, choose the units whose content token you uploaded to Fleet, select **Review**, and then select **Add to Library**. Free apps don't have a license limit.
   - **Paid apps**: Select the app's price. Under **Licence Type**, choose **Managed Licences**. Under **Organizational Units**, choose the unit whose content token you uploaded to Fleet, and enter the number of licenses to buy. Then select **Review** and complete the purchase. Learn more in [Apple's documentation](https://support.apple.com/en-gb/guide/business/welcome/web).

3. In Fleet, head to the **Software** page and select a fleet in the fleets dropdown.

4. Select **Add software > App store**, then select the app you just added.

> To see which organizational units can use an app and how many licenses are in use, in Apple Business, select **Apps & Services > Library** and select the app.

> If the developer drops support for a platform (e.g., removes the iPad version), Fleet still shows that platform on fleets where the app was already added, but other fleets can only add the currently supported platforms. To match what Apple currently supports, delete the app from the fleet and add it again.
>
> If you install the app on an iPad from a fleet where iPadOS is still listed, Apple installs the iPhone-only version, which runs in compatibility mode (the iPhone layout scaled up for iPad).

### Google Play (Android)

> Before using Fleet to manage Google Play Store apps, you must first [turn on Android MDM](https://fleetdm.com/guides/android-mdm-setup). Once you've completed that setup, you can follow the directions below for each app.

1. Head to the [Google Play Store](https://play.google.com/store/apps), find the app, and copy the ID at the end of the URL (e.g. "com.android.chrome")

2. In Fleet, head to the **Software** page and select a fleet in the fleets dropdown.

3. Select **Add software > App store**, choose the Android platform, then enter the application ID.

4. Optionally, use the **Target** option to limit which hosts can install the app. Choose **Custom**, then include or exclude hosts with specific labels.

> Dynamic (query) labels don't apply to Android hosts: including one targets no Android hosts, and excluding one has no effect. Use manual or host vitals labels to target Android apps.

#### Install Android web app (web clip)

> Before deploying web apps, make sure to [add Google Chrome](#google-play-android) first. This applies to both work profile (BYO) and fully-managed Android devices. If Chrome isn’t installed, the end user will be prompted to install it and redirected to the managed Play Store.

To add an Android web app, first create the web app using the Fleet API. Send a request to the [`Create Android web app`](https://fleetdm.com/docs/rest-api/rest-api#create-android-web-app).

The response includes an `app_store_id` (e.g. `com.google.enterprise.webapp.x1c41e22ab611cb98`). Use this ID as the application ID in **Add software > App store** (step 3 above).

**Example request**

```sh
curl -X POST https://<your_fleet_server_url>/api/v1/fleet/software/web_apps \
  -H "Authorization: Bearer <your_fleet_api_token>" \
  -F 'title=Acme web app' \
  -F 'url=https://app.acme.com' \
  -F 'icon=@/path/to/app-icon.png'
```

### Microsoft Store (Windows)

1. Head to the [Microsoft Store](https://apps.microsoft.com), find the app, and copy the ID at the end of the URL (e.g. "9PLM9XGG6VKS" from `https://apps.microsoft.com/detail/9PLM9XGG6VKS`).

2. In Fleet, head to the **Software** page and select a fleet in the fleets dropdown.

3. Select **Add software > App store**, choose the Windows platform, then enter the application ID.

4. Optionally, turn on **Self-service**, use the **Target** option to limit which hosts can install the app, and expand **Advanced options** to edit the pre-install query and the install, post-install, and uninstall scripts.

Fleet adds default install and uninstall scripts that install and uninstall the app for all users on the host through the Microsoft Store. In the scripts, Fleet replaces `$MICROSOFT_STORE_ID` with the app's Microsoft Store ID and `$PACKAGE_ID` with its package family name. To go back to the default scripts, delete the app and add it again.

> If an app is also available as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps), use the Fleet-maintained app. Fleet knows its version, and you can [patch it](https://fleetdm.com/guides/patching-end-user-experience).


#### Supported Microsoft Store apps

Fleet supports modern (MSIX) apps from the Microsoft Store. Some apps in the Microsoft Store use their own installer (e.g. MSI or EXE) instead, and Fleet can't add them. To install one of these apps, add it as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) or a [custom package](https://fleetdm.com/guides/deploy-software-packages).

To check an app before you add it, look at the ID at the end of its Microsoft Store URL:

- IDs that start with `9` (e.g. `9PLM9XGG6VKS`) are usually MSIX apps.
- IDs that start with `XP` (e.g. `XP99J3KP4XZ4VV`) are usually apps with their own installer.

> Some Microsoft Store listings can't be installed by Fleet because Microsoft doesn't provide a package for them. This is common for games and for apps that are no longer available in the Microsoft Store.

## Edit or delete the app

1. In Fleet, head to the **Software** page and select a fleet in the fleets dropdown.

2. Search for the app you want to remove and select the app to head to its **Software details** page.

3. To edit the app icon and display name, click **Edit**.

4. To configure the app's self-service categories or change which hosts can install the app, expand the app's version and click the pencil (edit) icon.

5. To remove the app, expand the app's version and click the trash can (delete) icon. If the app has more than one version, delete each version.

## Configure automatic updates for an app (iOS / iPadOS only)

> Currently, Fleet doesn't patch Android apps. Google Play updates them automatically, or the end user can update them from the Play Store in their work profile.

1. In Fleet, head to the **Software** page and select a fleet in the fleets dropdown.

2. Search for the app you want to configure and select the app to head to its **Software details** page, then find the version you want to configure.

3. Expand the version and click the pencil (edit) icon.

4. In the modal dialog that opens, click **Enable auto updates** and configure an update window of at least one hour.

## Update Microsoft Store apps

The Microsoft Store doesn't share version numbers for most apps, so Fleet shows **Latest** as the app's version. Each install gets the latest version available in the Microsoft Store.

The Microsoft Store updates apps automatically. To make sure automatic updates stay on, deploy a [Windows configuration profile](https://fleetdm.com/guides/custom-os-settings) that sets the [ApplicationManagement/AllowAppStoreAutoUpdate](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-applicationmanagement#allowappstoreautoupdate) policy to `1`. Save the profile below as an `.xml` file and add it in Fleet:

```xml
<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/ApplicationManagement/AllowAppStoreAutoUpdate</LocURI>
    </Target>
    <Data>1</Data>
  </Item>
</Replace>
```

After the profile is applied, **App updates** in the Microsoft Store's **Settings** is turned on and grayed out, so end users can't turn it off. To let end users turn automatic updates on or off again, delete the profile.

To update an app right away, select **Reinstall** on the host's **Host details** page. Fleet installs the latest version from the Microsoft Store.

## Managed configuration

Currently, managed configuration isn't supported for Microsoft Store apps.

### Apple (VPP)

Currently, configuration for Apple hosts is supported on iOS and iPadOS. Managed configuration is often referred to as App Config.

Fleet supports any option provided by the app's developer. Each app supports different options. To find the supported options, check the app documentation.

New or updated configuration isn't applied until the end user fully quits and reopens the app.

This example Zoom configuration ensures that the end user has only the SSO login option, and it pre-populates the login URL to `example.zoom.us`. For more information, visit [Zoom docs](https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0064102).

```xml
<dict>
  <key>ForceLoginWithSSO</key>
  <true />
  <key>SetSSOURL</key>
  <string>example</string>
</dict>
```

### Google Play (Android)

All options from [ApplicationPolicy - Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#ApplicationPolicy) are supported except `packageName` and `installType`, which Fleet sets for you.

`managedConfiguration` supports any option provided by the app's developer. Each app supports different options. To find the supported options, check the app documentation.

#### Example (GlobalProtect)

This configuration makes it so the end user won't have to type the portal hostname the first time they open GlobalProtect. It also disables "always on VPN," meaning GlobalProtect won’t automatically connect when the host is online. The end user has to tap **Connect**.

```json
{
  "managedConfiguration": {
    "portal": "example.portal.com",
    "connect_method": "on-demand"
  }  
}

```

Options for GlobalProtect can be found in their [documentation](https://docs.paloaltonetworks.com/globalprotect/administration/globalprotect-apps/deploy-the-globalprotect-app-on-mobiles/manage-the-globalprotect-app-using-other-third-party-mdms/configure-the-globalprotect-app-for-android).

#### Example (Google Calendar)

This configuration allows end users to add widgets from the Google Calendar in their work profile to their home screen.

```json
{
  "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"
}
```

#### Example (1Password)

On Android 14 and above, third-party credential providers are blocked in the work profile unless the app is explicitly allowed. This configuration allows end users to select 1Password as their autofill and passkey (credential) provider in the work profile.

```json
{
  "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"
}
```

## Multiple app instances

Fleet supports adding multiple instances of each app, so that each can have its own configuration, automatic update schedule, label scoping, categories and self-service flag. 

Hosts that install the app or already have it installed will get the configuration of the version they're scoped to by labels. If a host is scoped to multiple versions, Fleet installs the version that was added first.

### Add a version

1. In Fleet, head to the **Software** page, select a fleet in the fleets dropdown, then search for the app and select it to head to its **Software title** page. The first version of the app is named "Default version".

2. Click **Add version** and enter a **Name**. The name must be unique for the app in the fleet.

3. Add a **Configuration**, set the version's other options, select which hosts the version targets, and click **Add**.

### Edit or delete a version

1. Find the version that you want to edit or delete, and click on it to expand it.

2. To edit the version, click on the pencil (edit) icon to the right. Alternatively, click on the self-service or the label icon.

3. To delete the version, click on the trash (delete) icon to the right.

## Install an app

### Apple (VPP)

Apps can be installed manually on each host's **Host details** page. For macOS apps, apps can also be installed via self-service on the end user's **Fleet Desktop > My device** page or [automatically via policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet).

> If the install fails with `ErrorCode` 301 and a `LocalizedDescription` of "Invalid Status Code The response has an invalid status code" it may be because the app has a minimum OS version higher than what the targeted host is running.
> 
> To find the minimum OS version for the app, visit the [App Store](https://apps.apple.com/), find the app, scroll to the bottom, and look for **Compatibility** under **Information**.

> VPP app installs are automatically attempted up to 4 times (1 initial attempt + 3 retries) to handle intermittent issues.

Currently, Apple App Store (VPP) apps can't be uninstalled via Fleet. If the app is uninstalled by the end user, or when the host is unenrolled, the license won't be revoked. You can revoke the license by running [this script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/revoke-vpp-licenses.sh).

> VPP apps on iOS/iPadOS hosts will be uninstalled when the host has MDM features turned off.

#### Convert unmanaged iOS and iPadOS apps to managed

If an end user already installed an app from the App Store, you can take over management by installing the same app through Fleet. Managing the app lets you apply [managed configuration](#managed-configuration) and makes sure the app is removed when the host is unenrolled.

On company-owned hosts (enrolled automatically via Apple Business Manager), the app becomes managed silently. On BYOD hosts and hosts that are manually enrolled, the end user is prompted to allow management. If the end user declines, the app stays unmanaged.

> **Warning:** Fleet can't currently tell that the end user declined. Apple returns the app in the `InstalledApplicationList` MDM command as if it were managed, so Fleet reports the install as successful and lists the app in the host's software inventory. The app may drop out of inventory within minutes, or stay for up to 24 hours. In the meantime, managed configuration isn't applied and the app won't be removed when the host is unenrolled. Fleet plans to use the app's real management status to report these installs as failed. Follow [this issue](https://github.com/fleetdm/fleet/issues/52923) for progress.

To confirm whether an app is actually managed, send [ManagedApplicationList](https://developer.apple.com/documentation/devicemanagement/managed-application-list-command) as a [custom MDM command](https://fleetdm.com/guides/mdm-commands). The response includes a `Status` for each app Fleet tried to manage:

- `Managed`: the app is managed by Fleet.
- `PromptingForManagement`: the end user hasn't answered the prompt yet.
- `ManagementRejected`: the end user declined the prompt, and the app is not managed.

If the end user declined, installing the app again from **Host details** usually prompts them again. Sometimes the prompt doesn't show up, and Apple acknowledges the command without doing anything. If that happens, ask the end user to delete the app, then install it through Fleet again.

> **Note:** If the app wasn't already installed and the end user declines the prompt, the app never appears in the host's software inventory, and the install fails verification after 10 minutes.

#### Updating iOS and iPadOS apps in Single App Mode (kiosk)

[AppLock](https://developer.apple.com/documentation/devicemanagement/applock) forces the selected app to open on the supervised device and prevents the use of other apps.

Apps can't be updated while in Single App Mode. To update iOS and iPadOS apps, temporarily disable Single App Mode. You can use an automation tool like [Tines](https://www.tines.com/) and leverage our [API](https://fleetdm.com/docs/rest-api/rest-api#os-settings) to delete the AppLock configuration profile during the scheduled update window.

#### iOS and iPadOS managed configuration

If the app is in the foreground when the configuration is sent and a newer version of the app is available, the end user sees a prompt to update the app. The configuration is applied whether or not the end user accepts the prompt. The configuration takes effect the next time the app is quit and reopened.

### Google Play (Android)

Android apps can be installed via self-service in the end user's managed Google Play Store (work profile).

### Microsoft Store (Windows)

Apps can be installed and uninstalled manually on each host's **Host details** page, via self-service on the end user's **Fleet Desktop > My device** page, or [automatically via policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet). They can also be installed during [setup experience](https://fleetdm.com/guides/setup-experience).

Fleet runs the install, post-install, and uninstall scripts as the system, like other software. Apps are installed for all users on the host, including users who sign in later. No user needs to be logged in. The first time a user signs in after an install, the app can take a few seconds to appear in their Start menu. Uninstalling an app removes it for all users.

If an end user already installed the app themselves, Fleet installs it for all users, so you don't need to uninstall it first. When you select **Reinstall**, Fleet installs the latest version from the Microsoft Store. App settings and data are kept.

If the app is open during a reinstall, the Microsoft Store closes it to install the update. The app doesn't open again on its own.

Only free apps are supported. Paid apps fail to install with a licensing error.

#### Let end users install only the apps you allow

To stop end users from installing apps from the Microsoft Store themselves, turn off the Microsoft Store app with the configuration profile below. Then add the apps you allow to self-service, so end users install them from **Fleet Desktop > My device**. Fleet can still install Microsoft Store apps when the Microsoft Store app is turned off.

This profile is only supported on Windows Enterprise and Education.

```xml
<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/ADMX_WindowsStore/RemoveWindowsStore_2</LocURI>
    </Target>
    <Data><![CDATA[<enabled/>]]></Data>
  </Item>
</Replace>
```

> Microsoft's [documentation](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-admx-windowsstore#removewindowsstore_2) says access to the Microsoft Store is required to install app updates. With this profile, apps might not update automatically. To update an app, select **Reinstall** in Fleet.

## API and GitOps

Fleet also provides a REST API for managing app store apps programmatically. Learn more in the API [reference docs](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app).

To manage App Store apps using Fleet's best practice GitOps, check out the `app_store_apps` key in [the GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#app-store-apps).

<meta name="articleTitle" value="Install app store apps">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-08">
<meta name="description" value="This guide will walk you through installing Apple App Store, Google Play Store, and Microsoft Store apps on macOS, iOS, iPadOS, Android, and Windows hosts.">
