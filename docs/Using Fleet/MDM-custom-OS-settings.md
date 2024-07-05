# Custom OS settings

In Fleet you can enforce OS settings on your macOS and Windows hosts using configuration profiles.

## Enforce OS settings

You can enforce OS settings using the Fleet UI, Fleet API, or [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

For macOS hosts, Fleet recommends the [iMazing Profile Creator](https://imazing.com/profile-editor) tool for creating and exporting macOS configuration profiles.

For Windows hosts, copy out this [Windows configuration profile template](https://fleetdm.com/example-windows-profile) and update the profile using any configuration service providers (CSPs) from [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

Fleet UI:

1. In the Fleet UI, head to the **Controls > OS settings > Custom settings** page.

2. Choose which team you want to add a configuration profile to by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Select **Upload** and choose your configuration profile.

Fleet API: API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile)

### OS settings status

In the Fleet UI, head to the **Controls > OS settings** tab.

In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts:

* Verified: hosts that installed all configuration profiles. Fleet has verified with osquery. Declaration profiles are verified with DDM

* Verifying: hosts that have acknowledged all MDM commands to install configuration profiles. Fleet is verifying the profiles are installed with osquery. If the profile wasn't installed, Fleet will redeliver the profile.

* Pending: hosts that will receive MDM commands to install configuration profiles when the hosts come online.

* Failed: hosts that failed to install configuration profiles. For Windows profiles, the status codes are documented in Microsoft's documentation [here](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

In the list of hosts, click on an individual host and click the **OS settings** item to see the status for a specific setting.

## Variables

Variables can be used in the configuration profile to input host-specific information. When the configuration profile is applied, the `${variable}` is replaced with the value of the [respective host information](https://fleetdm.com/docs/rest-api/rest-api#get-host).  

Available variables:

|    First Header    | Second Header |
| ------------------ | ------------- |
| `${display_name}`    | Host's display name (`host.computer_name`).    |
| `${hardware_serial}` | Host's serial number (`host.hardware_serial`)  |
| `${uuid} `           | Host's UUID (`host.uuid`)                      |


<meta name="pageOrderInSection" value="1505">
<meta name="title" value="Custom OS settings">
<meta name="description" value="Learn how to enforce custom settings on macOS and Window hosts using Fleet's configuration profiles.">
<meta name="navSection" value="Device management">
