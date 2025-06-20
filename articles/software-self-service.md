# Software self-service

![Software self-service](../website/assets/images/articles/software-self-service-1600x900@2x.png)

_Available in Fleet Premium_

Fleet’s self-service software feature empowers end users by allowing them to independently install approved software packages from a curated list through the Fleet Desktop “My device” page. This not only reduces the administrative burden on IT teams but also enhances user productivity and satisfaction. In this guide, we will walk you through the process of uploading, editing, and managing self-service software packages in Fleet, enabling seamless software distribution and management.

> Software packages can be added to a specific team or to "No team." "No team" is the default assignment for hosts that are not part of any specific team.

## Step-by-step instructions

### Add self-service software

1. Select the team to which you want to add the software package from the dropdown in the upper left corner of the page.
2. Select **Software** in the main navigation menu.
3. Press the **Add software** button in the upper right corner of the page.
4. Stay on the **Fleet-maintained** tab to add a Fleet-maintained App, or select one of the other tabs if you want to add an App Store (VPP) app or upload a custom software package.
5. Based on the type of software you would like to add, follow instructions for adding a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps#add-a-fleet-maintained-app), a [VPP app](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet#add-the-app-to-fleet), or a [custom package](https://fleetdm.com/guides/deploy-software-packages#add-a-custom-package). In each case, you can check the **Self-service** box when adding software to make it immediately available for self-service installation once added.

### Enable self-service on existing software

1. Select the team to which you added the software from the dropdown in the upper left corner of the page.
2. Select **Software** in the main navigation menu.
3. To make it easier to find your software, select the **All software** dropdown and choose **Available for install.** This filters the results in the table to show only software that can be installed on hosts. If you don’t see your software, page through the results or search for your software's name in the search bar. Once you find the software, select its title.
4. Press the ✏️ icon, then check **Self-service** in the **Options** section. You can also assign categories to your software, which will organize the display of software to end users on the **My device > Self-service** page.
5. Press the **Save** button.

### Download a self-service software package

1. Select the team to which you added the software from the dropdown in the upper left corner of the page.
2. Select **Software** in the main navigation menu.
3. Select the **All software** dropdown and choose **Self-service.** Page through the results to find your software or search for the software package name in the search bar.
4. Select the row containing the software’s name.
5. Press the **Download** icon (next to the ✏️ icon) to the right of the software package's filename.

### Delete self-service software

1. Select the team to which you added the software from the dropdown in the upper left corner of the page.
2. Select **Software** in the main navigation menu.
3. Select the **All software** dropdown and choose **Self-service.** Page through the results to find your software or search for the software package name in the search bar.
4. Select the row containing the software’s name.
5. Press the 🗑️ icon to the right of the software package's filename, then press the "Delete" button to confirm.

### Install self-service software

To install self-service software on a host:

1. From the Fleet desktop icon in the OS menu bar, select **Self-service.** This will open your default web browser to the list of self-service software packages available to install.
2. Select **Install** to the right of the software title you'd like to install.

### Uninstall self-service software

To uninstall self-service software on a host:

1. From the Fleet desktop icon in the OS menu bar, select **Self-service.** This will open your default web browser to the list of self-service software packages available to uninstall.
2. Select **Uninstall** to the right of the software title you'd like to uninstall.

### Use the REST API for self-service software

Fleet provides a REST API for managing software, including self-service software packages.  Learn more about Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software).

### Manage self-service software with GitOps

To manage self-service software using GitOps, check out the `software` key in the [GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#software).

> Note: When managing Fleet via GitOps, software packages uploaded using the web UI will not persist unless they are also added in GitOps using the `hash_sha256` field.

## Conclusion

Fleet’s self-service software feature not only simplifies software management for IT administrators but also empowers end users by giving them access to necessary software on demand. This feature ensures that your hosts remain secure while improving overall user experience. For further information and advanced management techniques, refer to Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software) and [GitOps](https://fleetdm.com/docs/using-fleet/gitops#software) documentation. 

<meta name="articleTitle" value="Software self-service">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-06-20">
<meta name="articleImageUrl" value="../website/assets/images/articles/software-self-service-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding apps to Fleet for user self-service.">
