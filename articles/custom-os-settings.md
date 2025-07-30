# Custom OS settings

In Fleet you can enforce OS settings like security restrictions, screen lock, Wi-Fi etc., on your your macOS, iOS, iPadOS, and Windows hosts using configuration profiles.

Currently, Fleet only supports system (device) level configuration profiles.

## Enforce

You can enforce OS settings using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

For macOS, iOS, and iPadOS hosts, Fleet recommends the [iMazing Profile Creator](https://imazing.com/profile-editor) tool for creating and exporting macOS configuration profiles. Fleet signs these profiles for you. If you have self-signed profiles, run this command to unsign them: `usr/bin/security cms -D -i  /path/to/profile/profile.mobileconfig | xmllint --format -`

For Windows hosts, copy this [Windows configuration profile template](https://fleetdm.com/example-windows-profile) and update the profile using any [configuration service providers (CSPs)](https://fleetdm.com/guides/creating-windows-csps) from [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

Fleet UI:

1. In the Fleet UI, head to the **Controls > OS settings > Custom settings** page.

2. Choose which team you want to add a configuration profile to by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Select **Upload** and choose your configuration profile.

4. To edit the OS setting, first remove the old configuration profile and then add the new one. On macOS, iOS, and iPadOS, removing a configuration profile will remove enforcement of the OS setting.

Fleet API: Use the [Add custom OS setting (configuration profile) endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) in the Fleet API.

### Device and user scope

Currently, on macOS hosts, Fleet supports enforcing OS settings at the device (device-scoped) and user (user-scoped) levels. User-scoped declaration (DDM) profiles and Windows configuration profiles are coming soon.

If a host is automatically enrolled (via [ADE](https://support.apple.com/en-us/102300)), user-scoped profiles are delivered to the user that was created during first time setup. For hosts that enrolled and turned on MDM manually, user-scoped profiles are delivered to the user that installed Fleet's agent (fleetd).

How to deliver user-scoped configuration profiles:
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

#### Upgrading to 4.71.0

When upgrading to 4.71.0, here's how to prepare your already enrolled hosts for user-scoped configuration profiles:
1. If the host automatically enrolled to Fleet (via ADE), you don't need to take action. Fleet added support for the user-scoped configuration profiles on these hosts.
2. To deliver user-scoped profiles to hosts that manually enrolled and turned on MDM, first turn off MDM and ask end user to [turn on MDM](https://fleetdm.com/guides/mdm-migration#migrate-hosts:~:text=If%20the%20host%20is%20not%20assigned%20to%20Fleet%20in%20ABM%20(manual%20enrollment)%2C%20the%20end%20user%20will%20be%20given%20the%20option%20to%20download%20the%20MDM%20enrollment%20profile%20on%20their%20My%20device%20page.) through the **My device** page.

When upgrading to Fleet 4.71.0, here's how to update configuration profiles that are already installed on hosts so that they're delivered to the user scope:

1. Check for profiles with `PayloadScope` set to `User`. Already deployed profiles with `PayloadScope` set to `User` won’t be re-installed on hosts automatically.
2. To change them to the user-scope, update the `PayloadIdentifier`, re-add the profile to Fleet, and delete the old profile. This will uninstall the device-scope profile and install the profile in the user scope. If you're using [GitOps](https://fleetdm.com/docs/configuration/yaml-files), just update the `PayloadIdentifier` and run GitOps.

In versions older than 4.71.0, Fleet always delivered configuration profiles to the device scope (even when the profile's `PayloadScope` was set to `User`)

If you want to make sure the profile stays device-scoped, update `PayloadScope` to `System` or remove `PayloadScope` entirely. The default scope in Fleet is `System`. 

## See status

In the Fleet UI, head to the **Controls > OS settings** tab.

In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts:

* **Verified**: hosts that applied all OS settings. Fleet verified by running an osquery query on Windows and macOS hosts (declarations profiles are verified with a [DDM StatusReport](https://developer.apple.com/documentation/devicemanagement/statusreport)). Currently, iOS and iPadOS hosts are "Verified" after they acknowledge all MDM commands to apply OS settings.

* **Verifying**: hosts that acknowledged all MDM commands to apply OS settings. Fleet is verifying. If the profile wasn't delivered, Fleet will redeliver the profile.

* **Pending**: hosts that are running MDM commands or will run MDM commands to apply OS settings when they come online.

* **Failed**: hosts that failed to apply OS settings. For Windows profiles, status codes are listed in [Microsoft's OMA DM docs](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

In the list of hosts, click on an individual host and click the **OS settings** item to see the status for a specific setting.

Currently, when editing a profile using Fleet's GitOps workflow, it can take 30 seconds for the
profile's status to update to "Pending."

On Windows, due to limitations of the MDM protocol, verification of [Win32 and Desktop Bridge app ADMX
policy](https://learn.microsoft.com/en-us/windows/client-management/win32-and-centennial-app-policy-configuration)
CSPs are limited. Fleet only verifies that the host returned a success status code in response to the MDM
command to install the configuration profile. You can query the registry keys defined by the ADMX
policy. For instance, if an ADMX file defines the following policy:
```
      <policy name="Subteam" class="Machine" displayName="Subteam" key="Software\Policies\employee\Attributes" explainText="Subteam" presentation="String">
         <parentCategory ref="DefaultCategory" />
         <supportedOn ref="SUPPORTED_WIN10" />
         <elements>
            <text id="Subteam" valueName="Subteam" />
         </elements>
      </policy>
```

The following osquery query will return any values set by this policy:
```
SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\Software\Policies\employee\Attributes\Subteam';
```

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-27">
<meta name="articleTitle" value="Custom OS settings">
<meta name="description" value="Learn how to enforce custom settings on macOS and Window hosts using Fleet's configuration profiles.">
