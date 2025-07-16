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

#### User channel for configuration profiles on macOS

Before version 4.71.0, Fleet didn't support sending configuration profiles (`.mobileconfig`) to the macOS user channel (aka "Payload Scope" in iMazing Profile Creator). Profiles with `PayloadScope` set to `User` were delivered to the device channel by default. From Fleet 4.71.0 onward, both device and user channels are supported. 

User-scoped profile is delivered to the user that turned on MDM on the host (installed fleetd or enrolled host via automatic enrollment (ADE)). Hosts enrolled before version 4.71.0, won't have user channel enabled. Currently, you can:
1. Turn off MDM on manually enrolled Mac and ask end user to [turn on MDM](https://fleetdm.com/guides/mdm-migration#migrate-hosts:~:text=If%20the%20host%20is%20not%20assigned%20to%20Fleet%20in%20ABM%20(manual%20enrollment)%2C%20the%20end%20user%20will%20be%20given%20the%20option%20to%20download%20the%20MDM%20enrollment%20profile%20on%20their%20My%20device%20page.) through the **My device** page.
2. Run `sudo profiles renew -type enrollment` on automatically enrolled Mac.

> Fleet will soon improve this and automatically enable the user channel for all macOS hosts. Check out the [issue](https://github.com/fleetdm/fleet/issues/30043).

Support for declaration (DDM) profiles is coming soon.

Existing profiles with `PayloadScope` set to`User` won’t update automatically. These are delivered to the device channel and will remain there until you take action.

To avoid confusion, please follow these steps:
-  Check for profiles with `PayloadScope` set to `User`.
-  To keep delivering them to the device channel, change `PayloadScope` to `System` to reflect the actual scope in your `.mobileconfig`. Also, you can remove `PayloadScope` as the default scope in Fleet is `System`. 
-  To deliver to the user channel, update the identifier(`PayloadIdentifier`) and re-upload the profile.

### See status

In the Fleet UI, head to the **Controls > OS settings** tab.

In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts:

* **Verified**: hosts that applied all OS settings. Fleet verified by running an osquery query on Windows and macOS hosts (declarations profiles are verified with a [DDM StatusReport](https://developer.apple.com/documentation/devicemanagement/statusreport)). Currently, iOS and iPadOS hosts are "Verified" after they acknowledge all MDM commands to apply OS settings.

* **Verifying**: hosts that acknowledged all MDM commands to apply OS settings. Fleet is verifying. If the profile wasn't delivered, Fleet will redeliver the profile.

* **Pending**: hosts that are running MDM commands or will run MDM commands to apply OS settings when they come online.

* **Failed**: hosts that failed to apply OS settings. For Windows profiles, status codes are listed in [Microsoft's OMA DM docs](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

In the list of hosts, click on an individual host and click the **OS settings** item to see the status for a specific setting.

Currently, when editing a profile using Fleet's GitOps workflow, it can take 30 seconds for the
profile's status to update to "Pending."

## How user scoped configuration profiles are assigned

Currently, Fleet supports hosts with one local user. If the host has multiple local users
(eg. User1 and User2), the profile is delivered to the user that turns on MDM on the host. For example, if User1 enrolls
to Fleet during ADE or installs the enrollment profile during BYOD enrollment, User1's local user will get
certificates.

For configuration profiles the default **PayloadScope** is **System**. You must assign **PayloadScope** to be
**User** in your configuration profile to apply it to the user channel.

Finally, only **.mobileconfig** configuration profiles are supported for the user channel. Support for declaration (DDM) profiles is coming soon.

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
