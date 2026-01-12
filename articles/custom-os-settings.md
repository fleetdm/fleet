# Custom OS settings

In Fleet you can enforce OS settings like security restrictions, screen lock, Wi-Fi, etc., on your macOS, iOS, iPadOS, Windows, and Android hosts using configuration profiles.

## Create configuration profile

For macOS, iOS, and iPadOS hosts, Fleet recommends the [iMazing Profile Creator](https://imazing.com/profile-editor) tool for creating and exporting macOS configuration profiles. Fleet signs these profiles for you. If you have self-signed profiles, run this command to unsign them: `/usr/bin/security cms -D -i  /path/to/profile/profile.mobileconfig | xmllint --format -`

For Windows hosts, copy this [Windows configuration profile template](https://fleetdm.com/example-windows-profile) and update the profile using any [configuration service providers (CSPs)](https://fleetdm.com/guides/creating-windows-csps) from [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

For Android hosts, copy this [Android configuration profile template](https://fleetdm.com/learn-more-about/example-android-profile) and update the profile using the options available in [Android Management API](https://developers.google.com/android/management/reference/rest/v1/enterprises.policies#resource:-policy). To learn how, watch [this video](https://youtu.be/Jk4Zcb2sR1w).

## Enforce

You can enforce OS settings using the Fleet UI, Fleet API, or [Fleet's best practice GitOps](https://github.com/fleetdm/fleet-gitops).

Fleet UI:

1. In the Fleet UI, head to the **Controls > OS settings > Custom settings** page.

2. Choose which team you want to add a configuration profile to by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Select **Add profile** and choose your configuration profile.

4. To edit the OS setting, first remove the old configuration profile and then add the new one. On macOS, iOS, iPadOS, and Android, removing a configuration profile will remove enforcement of the OS setting.

Fleet API: Use the [Add custom OS setting (configuration profile) endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) in the Fleet API.

### Device and user scope

Currently, on macOS and Windows hosts, Fleet supports enforcing OS settings at the device (device scoped) and user (user scoped) levels. The iOS, iPadOS, and Android platforms only support device-scoped configuration profiles. User-scoped declaration (DDM) profiles for macOS are coming soon.

If a macOS host is automatically enrolled (via [ADE](https://support.apple.com/en-us/102300)), user-scoped profiles are delivered to the user that was created during first time setup. For Macs that enrolled and turned on MDM manually, user-scoped profiles are delivered to the user that turned on MDM on the **Fleet Desktop > My device** page.

How to deliver user-scoped configuration profiles:

#### macOS

1. If you use iMazing Profile Creator, open your configuration profile in iMazing, select the **General** tab and update the **Payoad Scope** to **User**.
2. If you edit your configuration profiles in a text editor, open the configuraiton profile in your text editor, find or add the `PayloadScope` key, and set the value to `User`. Here's an example `.mobileconfig` snippet:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	...
	<key>PayloadScope</key>
	<string>User</string>
</dict>
</plist>
```

#### Windows

1. Head to the [Windows configuration profiles (CSPs) documentation](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider) to verify that all the settings in your Windows profile support the user scope. For example, the [SCEP setting](https://learn.microsoft.com/en-us/windows/client-management/mdm/clientcertificateinstall-csp#devicescep) supports both the device and user scope.
2. To make your Windows configuration profiles user scoped, replace `./Device` with `./User` in all `<LocURI>` elements.

#### Upgrading from below 4.71.0

Fleet added support for user-scoped macOS configuration profiles in Fleet 4.71.0. If you're upgrading Fleet from a version below 4.71.0, here's how to prepare your already enrolled hosts for macOS user-scoped configuration profiles:

1. If the host automatically enrolled to Fleet (via ADE), you don't need to take action. Fleet added support for the user-scoped configuration profiles on these hosts.
2. To deliver user-scoped profiles to hosts that manually enrolled and turned on MDM, first turn off MDM and ask end user to [turn on MDM](https://fleetdm.com/guides/mdm-migration#migrate-hosts:~:text=If%20the%20host%20is%20not%20assigned%20to%20Fleet%20in%20ABM%20(manual%20enrollment)%2C%20the%20end%20user%20will%20be%20given%20the%20option%20to%20download%20the%20MDM%20enrollment%20profile%20on%20their%20My%20device%20page.) through the **My device** page.

Edit user-scoped configuration profiles that are already installed on hosts:

1. Check for profiles with `PayloadScope` set to `User`. Already deployed profiles with `PayloadScope` set to `User` won’t be re-installed on hosts automatically.
2. To change them to the user-scope, update the `PayloadIdentifier`, re-add the profile to Fleet, and delete the old profile. This will uninstall the device-scope profile and install the profile in the user scope. If you're using [GitOps](https://fleetdm.com/docs/configuration/yaml-files), just update the `PayloadIdentifier` and run GitOps.

In versions older than 4.71.0, Fleet always delivered configuration profiles to the device scope (even when the profile's `PayloadScope` was set to `User`)

If you want to make sure the profile stays device-scoped, update `PayloadScope` to `System` or remove `PayloadScope` entirely. The default scope in Fleet is `System`. 

## See status

In the Fleet UI, head to the **Controls > OS settings** tab.

In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts.

### Verified

Hosts that applied all OS settings. 

For macOS configuration profiles and device-scoped Windows profiles, Fleet verified by running an osquery query. It can take up to 1 hour ([configurable](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-detail-update-interval)) for these profiles to move from "Verifying" to "Verified".

macOS declarations profiles are verified with a [DDM StatusReport](https://developer.apple.com/documentation/devicemanagement/statusreport)).

User-scoped Windows profiles are "Verified" after Fleet gets a [200 response](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes) from the Windows MDM protocol.

iOS and iPadOS hosts are "Verified" after they acknowledge all MDM commands to apply OS settings. Android hosts are "Verified" after Fleet verifies that the settings is applied in the next [status report](https://developers.google.com/android/management/reference/rest/v1/enterprises.devices).

### Verifying

Hosts that acknowledged all MDM commands to apply OS settings. Fleet is verifying. If the profile wasn't delivered, Fleet will redeliver the profile.

For Windows profiles, when Fleet gets a [200 response](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes) from the Windows MDM protocol, device-scoped profiles are "Verifying" but, currently, user-scoped Windows profiles go straight to "Verified."

### Pending

Hosts that are running MDM commands or will run MDM commands to apply OS settings when they come online.

### Failed

Hosts that failed to apply OS settings. For Windows profiles, status codes are listed in [Microsoft's OMA DM docs](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

In the list of hosts, click on an individual host and click the **OS settings** item to see the status for a specific setting.

Currently, when editing a profile using Fleet's GitOps workflow, it can take 30 seconds for the
profile's status to update to "Pending."

### Special Windows behavior

For Windows configuration profiles with the [Win32 and Desktop Bridge app ADMX policies](https://learn.microsoft.com/en-us/windows/client-management/win32-and-centennial-app-policy-configuration), Fleet only verifies that the host returned a success status code in response to the MDM command to install the configuration profile. You can query the registry keys defined by the ADMX policy. For instance, if an ADMX file defines the following policy:
```
      <policy name="Subteam" class="Machine" displayName="Subteam" key="Software\Policies\employee\Attributes" explainText="Subteam" presentation="String">
         <parentCategory ref="DefaultCategory" />
         <supportedOn ref="SUPPORTED_WIN10" />
         <elements>
            <text id="Subteam" valueName="Subteam" />
         </elements>
      </policy>
```

To verify that the OS setting is applied, run the following osquery query:
```
SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\Software\Policies\employee\Attributes\Subteam';
```

> If your Windows profile fails with the following error: "The MDM protocol returned a success but the result couldn’t be verified by osquery", and the profile includes `[!CDATA []]` sections, [escape the XML](https://www.freeformatter.com/xml-escape.html) instead of using CDATA. For example, `[!CDATA[<enabled/>]]>` should be changed to `&lt;enabled/&gt;`.

### Special Android behvaior

On Android, if some settings from the profile fail (e.g. incompatible device), other settings from the profile will still be applied. Failed settings will be surfaced on **Host > OS settings**.
Also, some settings from the profile might be overridden by another configuration profile, which means if multiple profiles include the same setting, the profile that is delivered most recently will be applied.

The error message will provide the reason from the Android Management API (AMAPI) for why certain settings are not applied. Possible reasons are listed in the [AMAPI docs](https://developers.google.com/android/management/reference/rest/v1/NonComplianceReason).

## Broken profiles

If one or more labels included in the profile's scope are deleted, the profile will not apply to new hosts that enroll.

On macOS, iOS, iPadOS, and Windows, a broken profile will not remove the enforcement of the OS settings applied to existing hosts. To enforce the OS setting on new hosts, delete it and upload it again.

On Android hosts, a broken profile will remove the enforcement of the OS settings for existing hosts. To enforce the OS setting on existing and new hosts, delete it and upload it again.

## Unmanaged profiles

macOS, iOS, and iPadOS profiles installed manually by the end user aren't managed by Fleet. They're not visible and can't be removed from the host via Fleet. Additionally, if a backup is migrated to a new host using [Apple's Migration Assistant](https://support.apple.com/en-us/102613) and it contains configuration profiles, those profiles aren't managed.

To manually remove unmanaged profiles, ask the end user to go to **System Settings > General > Device Management**, select the profile, and select the **- (minus)** button at the bottom of the list.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-27">
<meta name="articleTitle" value="Custom OS settings">
<meta name="description" value="Learn how to enforce custom settings on macOS and Window hosts using Fleet's configuration profiles.">
