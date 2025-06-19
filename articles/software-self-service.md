# Software self-service

![Software self-service](../website/assets/images/articles/software-self-service-1600x900@2x.png)

_Available in Fleet Premium_

Fleet’s self-service software feature empowers end users by allowing them to independently install approved software packages from a curated list through the Fleet Desktop “My device” page. This not only reduces the administrative burden on IT teams but also enhances user productivity and satisfaction. In this guide, we will walk you through the process of uploading, editing, and managing self-service software packages in Fleet, enabling seamless software distribution and management.

> Software packages can be added to a specific team or to "No team." "No team" is the default assignment for hosts that are not part of any specific team.

## Step-by-step instructions

### Add self-service software

1. Click “Software” in the main navigation menu.
2. Click the dropdown in the upper left corner of the page and click on the team to which you want to add the software package.
3. Click the “Add software” button in the upper right corner of the page.
4. Stay on the "Fleet-maintained" tab to add a Fleet-maintained App, or select one of the other tabs if you want to add an App Store (VPP) app or upload a custom software package.
5. Based on the type of software you would like to add, follow instructions for adding a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps#add-a-fleet-maintained-app), a [VPP app](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet#add-the-app-to-fleet), or a [custom package](https://fleetdm.com/guides/deploy-software-packages#add-a-custom-package). In each case, you can check the **Self-service** box when adding software to make it immediately available for self-service installation once added.

### Enable self-service on existing software

1. Click “Software” in the main navigation menu.
2. Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. To make it easier to find your software, click on the dropdown to the left of the search bar and select "Available for install". This will filter the results in the table to only show software that can be installed on hosts. If you still don’t see your software, you can page through the results or search for your software's name in the search bar. Once you find the software, click its row in the table.
4. Click the ✏️ icon, then under "Options", check "Self-service." You can also select categories where the software will be visible to end users viewing software on the **My device > Self-service** page.
5. Click "Save."

### Download a self-service software package

1. Click “Software” in the main navigation menu.
2. Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. Click on the dropdown to the left of the search bar and select “Self-service” and page through the results or search for your software package’s name in the search bar.
4. Click on the software package’s name.
5. Click the Download icon (to the left of the ✏️ icon) near the software package's filename.

### Delete self-service software

1. Click “Software” in the main navigation menu.
2. Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. Click on the dropdown to the left of the search bar and select “Self-service” and page through the results or search for your software package’s name in the search bar.
4. Click on the software package’s name.
5. Click the 🗑️ icon near the software package's filename, then click the "Delete" button to confirm.

### Install self-service software

To install the self-service software package on the host:

1. Click on the Fleet Desktop icon in the OS menu bar. Click “Self-service”. This will point your default web browser to the list of self-service software packages in the “My device” page.
2. Click the “Install” link next to the software you want to install.

### Uninstall self-service software

To uninstall a self-service software package on the host:

1. Click on the Fleet Desktop icon in the OS menu bar. Click “Self-service”. This will point your default web browser to the list of self-service software packages in the “My device” page.
2. Click the “Uninstall” link next to the software you want to uninstall.

### Use the REST API for self-service software

Fleet provides a REST API for managing software, including self-service software packages.  Learn more about Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software).

### Manage self-service software with GitOps

To manage self-service software using Fleet's best practice GitOps, check out the `software` key in the [GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#software).

> Note: with GitOps enabled, software packages uploaded using the web UI will not persist unless they are also added in GitOps using the `hash_sha256` field.

## Conclusion

Fleet’s self-service software feature not only simplifies software management for IT administrators but also empowers end users by giving them access to necessary software on demand. This feature ensures that your hosts remain secure while improving overall user experience. For further information and advanced management techniques, refer to Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software) and [GitOps](https://fleetdm.com/docs/using-fleet/gitops#software) documentation. 

<meta name="articleTitle" value="Software self-service">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-06-18">
<meta name="articleImageUrl" value="../website/assets/images/articles/software-self-service-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding apps to Fleet for user self-service.">
