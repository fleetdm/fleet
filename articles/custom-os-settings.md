# Custom OS settings

In Fleet you can enforce OS settings like security restrictions, screen lock, Wi-Fi etc., on your your macOS, iOS, iPadOS, and Windows hosts using configuration profiles.

## Enforce OS settings

You can enforce OS settings using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

For macOS, iOS, and iPadOS hosts, Fleet recommends the [iMazing Profile Creator](https://imazing.com/profile-editor) tool for creating and exporting macOS configuration profiles.

For Windows hosts, copy out this [Windows configuration profile template](https://fleetdm.com/example-windows-profile) and update the profile using any configuration service providers (CSPs) from [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

Fleet UI:

1. In the Fleet UI, head to the **Controls > OS settings > Custom settings** page.

2. Choose which team you want to add a configuration profile to by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Select **Upload** and choose your configuration profile.

4. To modify the OS setting, first remove the old configuration profile and then add the new one.

> On macOS, iOS, and iPadOS, removing a configuration profile will remove enforcement of the OS setting.

Fleet API: API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile)

### OS settings status

In the Fleet UI, head to the **Controls > OS settings** tab.

In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts:

* **Verified**: hosts that applied all OS settings. Fleet verified by running an osquery query on macOS and Windows hosts (declarations profiles are verified with [DDM](https://support.apple.com/en-gb/guide/deployment/depb1bab77f8/web)). Currently,iOS and iPadOS hosts will have "Verified" status after they acknowledge all MDM commands to apply OS settings.

* Verifying: hosts that have acknowledged all MDM commands to apply OS settings. Fleet is verifying the OS settings are applied with osquery on macOS (declarations are verified with [DDM](https://support.apple.com/en-gb/guide/deployment/depb1bab77f8/web)) and Windows hosts. If the profile wasn't installed, Fleet will redeliver the profile.

* Pending: hosts that will receive MDM commands to apply OS settings when the hosts come online.

* Failed: hosts that failed to apply OS settings. For Windows profiles, the status codes are documented in Microsoft's documentation [here](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

In the list of hosts, click on an individual host and click the **OS settings** item to see the status for a specific setting.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-27">
<meta name="articleTitle" value="Custom OS settings">
<meta name="description" value="Learn how to enforce custom settings on macOS and Window hosts using Fleet's configuration profiles.">
