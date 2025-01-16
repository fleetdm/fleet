# Software self-service

![Software self-service](../website/assets/images/articles/software-self-service-1600x900@2x.png)

Fleet’s self-service software feature empowers end users by allowing them to independently install approved software packages from a curated list through the Fleet Desktop “My device” page. This not only reduces the administrative burden on IT teams but also enhances user productivity and satisfaction. In this guide, we will walk you through the process of uploading, editing, and managing self-service software packages in Fleet, enabling seamless software distribution and management.

## Prerequisites

* Fleet Premium is required for software self-service.

> Software packages can be added to a specific team or to the "No team" category. The "No team" category is the default assignment for hosts that are not part of any specific team.

## Step-by-step instructions

### Adding a self-service software package

1. **Navigate to the Software page**: Click “Software” in the main navigation menu.
2. **Select a team**: Click the dropdown in the upper left corner of the page and click on the team to which you want to add the software package.
3. **Open the “Add software” modal**: Click the “Add software” button in the upper right corner of the page.
4. **Select a software package to upload**: Click “Choose file” in the “Add software” modal and select a software package from your computer.
5. **Select the hosts that you want to target**: Select "All hosts" if you want the software to be available to all your hosts. Select "Custom" to scope the software to specific groups of hosts based on label membership. You can select "Include any", which will scope the software to hosts that have any of the labels you select, or "Exclude any", which will scope the software to hosts that do _not_ have the selected labels.
6. **Advanced options**: If desired, click “Advanced options” to add a pre-install condition or post-install script to your software package.
    * **Pre-install condition**: This is an osquery query that results in true. For example, you might require a specific software title to exist before installing additional extensions.
    * **Post-install script**: This might be used to apply a license key, perform configuration tasks, or execute cleanup tasks after the software installation.
7. **Make the software package self-service**: Check the “Self-service” checkbox to mark the software package as self-service.
8. **Finish the upload**: Click the “Add software” button to finish the upload process.

### Editing a self-service software package

1. **Navigate to the software details page for the software package**: Click “Software” in the main navigation menu.
2. **Select a team**: Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. **Filter by self-service**: To make it easier to find your software package, click on the dropdown to the left of the search bar and select “Self-service”. This will filter the results in the table to only show self-service software packages. If you still don’t see your software package, you can page through the results or search for your software package’s name in the search bar.
4. **Open the details page**: Click on the software package’s name. 
5. **Open the actions dropdown**: Click on the “Actions” dropdown on the far right of the page. From here, you can download the software package, delete the software package, or click “Advanced options” to see the options you configured when adding the software package. 

### Downloading a self-service software package

1. **Navigate to the software details page for the software package**: Click “Software” in the main navigation menu.
2. **Select a team**: Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. **Filter by self-service**: Click on the dropdown to the left of the search bar and select “Self-service” and page through the results or search for your software package’s name in the search bar.
4. **Download the software package**:
* **Option 1**: Click on the down-arrow next to the software package name in the list of self-service software packages to start an immediate download.
* **Option 2**: Click on the software package’s name to open the details page. Click on the “Actions” dropdown on the far right of the page, and then click on “Download” to download the software package to your computer.

### Deleting a self-service software package

1. **Navigate to the software details page for the software package**: Click “Software” in the main navigation menu.
2. **Select a team**: Click the dropdown in the upper left corner of the page and click on the team to which you added the software package.
3. **Filter by self-service**: Click on the dropdown to the left of the search bar and select “Self-service” and page through the results or search for your software package’s name in the search bar.
4. **Open the details page**: Click on the software package’s name.
5. **Open the actions dropdown**: Click on the “Actions” dropdown on the far right of the page.
6. **Delete the software package**: Click on “Delete” to remove the software package from Fleet. Confirm the deletion if prompted.

### Installing self-service software packages

To install the self-service software package on the host:

1. **Navigate to the “Self-service” tab**: Click on the Fleet Desktop icon in the OS menu bar. Click “Self-service”. This will point your default web browser to the list of self-service software packages in the “My device” page.
2. **Install the self-service software package**: Click the “Install” button for the software package you want to install.

### Using the REST API for self-service software packages

Fleet provides a REST API for managing software packages, including self-service software packages.  Learn more about Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software).

### Managing self-service software packages with GitOps

To manage self-service software packages using Fleet's best practice GitOps, check out the `software` key in the [GitOps reference documentation](https://fleetdm.com/docs/using-fleet/gitops#software).

> Note: with GitOps enabled, software packages uploaded using the web UI will not persist.

## Conclusion

Fleet’s self-service software feature not only simplifies software management for IT administrators but also empowers end users by giving them access to necessary software on demand. This feature ensures that your hosts remain secure while improving overall user experience. For further information and advanced management techniques, refer to Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#software) and [GitOps](https://fleetdm.com/docs/using-fleet/gitops#software) documentation. 

<meta name="articleTitle" value="Software self-service">
<meta name="authorFullName" value="Jahziel Villasana-Espinoza">
<meta name="authorGitHubUsername" value="jahzielv">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-08-06">
<meta name="articleImageUrl" value="../website/assets/images/articles/software-self-service-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding apps to Fleet for user self-service.">
