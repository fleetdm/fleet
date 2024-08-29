# Install App Store apps (VPP) on macOS, iOS, and iPadOS using Fleet

![Install VPP apps on macOS using Fleet](../website/assets/images/articles/install-vpp-apps-on-macos-using-fleet-1600x900@2x.png)


Fleet Premium supports the ability to add Apple App Store applications to your software library using the Volume Purchasing Program (VPP) and then install those apps on macOS, iOS, or iPadOS hosts. This guide will walk you through using this feature to add apps from your Apple Business Manager account to Fleet and install those apps on your hosts.

The Volume Purchasing Program is an Apple initiative that allows organizations to purchase and distribute apps and books in bulk. This program is particularly beneficial for organizations that need to deploy multiple apps to many devices. Key benefits of VPP include:
* **Bulk purchasing**: Purchase multiple licenses for an app in one transaction, often with volume discounts.
* **Centralized management**: Manage and distribute purchased apps from a central location.
* **Licensing flexibility**: Reassign app licenses as needed, ensuring efficient use of resources.
* **Streamlined deployment**: Use Fleet to automate the installation and configuration of purchased apps on enrolled devices.
* **Self-Service (macOS only)**: Allow users to assign licenses to their own devices as needed.

By integrating VPP with Fleet, organizations can seamlessly add apps to their software library and deploy them across macOS, iOS, and iPadOS hosts, ensuring that all devices have the necessary applications installed efficiently and effectively.

## Prerequisites
* **MDM features**: to use the VPP integration, you must first enable MDM features in Fleet. See the [MDM setup guide](https://fleetdm.com/docs/using-fleet/mdm-setup) for instructions on enabling MDM features.
* **Teams**: Apps can only be added to a specific Team. You can manage teams by selecting your avatar in the top navigation and then **Settings > Teams**. (Note: Apps can also be added to the 'No Team' team, which contains hosts not assigned to any other team.) You can control which team uses which VPP token by assigning teams to the VPP token. Each token may have multiple teams assigned to it, but each team may be assigned to only 1 token.

> As of Fleet 4.55.0, there is a [known issue](https://github.com/fleetdm/fleet/issues/20686) that uninstalled or deleted VPP apps will continue to show a status of `installed`.

## Accessing the VPP configuration

1. **Navigate to the MDM integration settings page**: Click your avatar on the far right of the main navigation menu, and then **Settings > Integrations > "Mobile device management (MDM)"**

2. **Add your VPP token**: Scroll to the "Volume Purchasing Program (VPP)" section. Click "Add VPP", and then click "Add VPP" again on the following page. Follow the directions on the modal to get your VPP token from Apple Business Manager, and then click the "Upload" button at the bottom to upload it to Fleet.

3. **Edit the team assignment for the new token**: Find the token in the table of VPP tokens. Click the "Actions" dropdown, and then click "Edit teams". Use the picker to select which team(s) this VPP token should be assigned to.

## Purchasing apps

To add apps to Fleet, you must first purchase them through Apple Business Manager, even if they are free. This ensures that all apps are appropriately licensed and available for distribution via the Volume Purchasing Program (VPP). For detailed instructions on selecting and buying content, please refer to Apple’s documentation on [purchasing apps through Apple Business Manager](https://support.apple.com/guide/apple-business-manager/select-and-buy-content-axmc21817890/web).

## Add an app to Fleet

1. **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

2. **Select your team**: Click on the "All teams" dropdown in the top left of the page and select your desired team.

3. **Open the "Add software" modal**: Click on the "Add software" button in the top right of the page.

4. **View your available apps**: Click on the "App Store (VPP)" tab in the "Add software" modal. The modal will list the apps that you have purchased through VPP but still need to add to Fleet.

5. **Add an app**: Select an app from the list. You may optionally check the "Self-Service" box at the bottom left of the modal if you wish for the software to be available for user-initiated installs. Finally, click the "Add software" button in the bottom right of the modal. The app should appear in the software list for the selected team.

## Remove an app from Fleet

1. **Navigate to the Software page**: Click "Software" in the main navigation menu.

2. **Find the app you want to remove**: Search for the app using the search bar in the top right corner of the table.

3. **Access the app's details page**: Click on the app's name in the table.

4. **Remove the app**: Click on the "Actions" dropdown on the right side of the page. Click "Delete," then click "Delete" on the confirmation modal. Deleting an app will not uninstall the app from the hosts on which it was previously installed.

## Installing apps on macOS, iOS, and iPadOS hosts

1. **Add the host to the relevant team.**

2. **Go to the host's detail page**: Click the "Hosts" tab in the main navigation menu. Filter the hosts by the team, and click the host's name to see its details page.

3. **Find the app**: Click the "Software" tab on the host details page. Search for the software you added in the software table's search bar. Instead of searching, you can also filter software by clicking the **All software** dropdown and selecting **Available for install.**

4. **Install the app**: Click the "Actions" dropdown on the far right of the app's entry in the
   table. Click "Install" to trigger an install. This action will send an MDM command to the host
   instructing it to install the app. If the host is offline, the upcoming install will show up in
   the **Details** -> **Activity** -> **Upcoming** tab of this page. After the app is installed and
   the host details are refetched, the app will show up as **Installed** in the **Software** tab.

## Installing apps on macOS using self-service

1. **Open Fleet from the host**: On the host that will be installing an application through self-service, click on the Fleet Desktop tray icon, then click **My Device**. This will open the browser to the device's page on Fleet.

2. **Navigate to the self-service tab**: Click on the **Self-Service** tab under the device's details.

3. **Locate the app and click install**: Scroll through the list of software to find the app you would like to install, then click the **Install** button underneath it.

## Renewing an expired or expiring VPP token

When one of your uploaded VPP tokens has expired or is within 30 days of expiring, you will see a warning
banner at the top of page reminding you to renew your token. You can do this with the following steps:

1. **Navigate to the MDM integration settings page**: Click your avatar on the far right of the main navigation menu, and then **Settings > Integrations > "Mobile device management (MDM)"** Scroll to the "Volume Purchasing Program (VPP)" section, and click "Edit".

2. **Renew the token**: Find the VPP token that you want to renew in the table. Token status is indicated in the "Renew date" column: tokens less than 30 days from expiring will have a yellow indicator, and expired tokens will have a red indicator. Click the "Actions" dropdown for the token and then click "Renew". Follow the instructions in the modal to download a new token from Apple Business Manager and then upload the new token to Fleet.

## Deleting a VPP token

To remove VPP tokens from Fleet:

1. **Navigate to the MDM integration settings page**: Click your avatar on the far right of the main navigation menu, and then **Settings > Integrations > "Mobile device management (MDM)"** Scroll to the "Volume Purchasing Program (VPP)" section, and click "Edit". 

2. **Delete the token**: Find the VPP token that you want to delete in the table. Click the "Actions" dropdown for that token, and then click "Delete". Click "Delete" in the confirmation modal to finish deleting the token.

## Managing apps with GitOps

To manage App Store apps using Fleet's best practice GitOps, check out the `software` key in the GitOps reference documentation [here](https://fleetdm.com/docs/using-fleet/gitops#software).

## REST API

Fleet also provides a REST API for managing apps programmatically. You can add, install, and delete apps via this API and manage your organization’s VPP tokens. Learn more about Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api).

## Conclusion

This feature extends Fleet's capabilities for managing macOS, iOS, and iPadOS hosts. Whether you manage your hosts' software via uploaded installers or via the App Store VPP integration, Fleet provides you with the tools you need to manage your hosts effectively.

<meta name="articleTitle" value="Install VPP apps on macOS using Fleet">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-08-12">
<meta name="articleImageUrl" value="../website/assets/images/articles/install-vpp-apps-on-macos-using-fleet-1600x900@2x.png">
<meta name="description" value="This guide will walk you through installing VPP apps on macOS, iOS, and iPadOS using Fleet.">
