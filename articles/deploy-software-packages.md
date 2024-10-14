# Deploy software packages

![Deploy software](../website/assets/images/articles/deploy-security-agents-1600x900@2x.png)

Fleet [v4.50.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0) introduced the ability to upload and deploy software to your hosts. Fleet [v4.57.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0) added the ability to include an uninstall script and edit software details. Beyond a [bootstrap package](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) at enrollment, deploying software allows you to specify and verify device configuration using a pre-install query and customization of the install, post-install, and uninstall scripts, allowing for key and license deployment and configuration.  Admins can modify these options and settings after the initial upload. This guide will walk you through the steps to upload, configure, install, and uninstall a software package to hosts in your fleet.

## Prerequisites

* Fleet [v4.57.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0).

* `fleetd` 1.25.0 deployed via MDM or built with the `--scripts-enabled` flag.

> `fleetd` prior to 1.33.0 will use a hard-coded uninstall script to clean up from a failed install. As of 1.33.0, the (default or customized) uninstall script will be used to clean up failed installs.

* An S3 bucket [configured](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3-software-installers-bucket) to store the installers.

* Increase any load balancer timeouts to at least 5 minutes for the [Add package](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Modify package](https://fleetdm.com/docs/rest-api/rest-api#modify-package) endpoints.

## Step-by-step instructions

### Access software packages

To access and manage software in Fleet:

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Click on the dropdown at the top left of the page.

> Software packages are tied to a specific team. This allows you to, for example, test a newer release of an application within your IT team before rolling it out to the rest of your organization, or deploy the appropriate architecture-specific installer to both Intel and Apple Silicon Macs.

* **Find your software**: using the filters on the top of the table, you can choose between:

    * “Available for install” filters software that can be installed on your hosts.

    * “Self-service” filters software that end users can install from Fleet Desktop.

* **Select software package**: Click on a software package to view details and access additional actions for the software.

### Add a software package to a team

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team "No team" to add a software package.

> Software cannot be added to "All teams."

* Click the “Add Software” button in the top right corner, and a dialog will appear.

* Choose a file to upload. `.pkg`, `.msi`, `.exe`, and `.deb` files are supported.

> Software installer uploads will fail if Fleet is unable to extract information from the installer package such as bundle ID and version number.

* To allow users to install the software from Fleet Desktop, check the “Self-service” checkbox.

* To customize installer behavior, click on “Advanced options.”

> After the initial package upload, all options can be modified, including the self-service setting, pre-install query, scripts, and even the software package file. When replacing an installer package, the replacement package must be the same type and for the same software as the original package.

#### Pre-install query

A pre-install query is a valid osquery SQL statement that will be evaluated on the host before installing the software. If provided, the installation will proceed only if the query returns any value.

#### Install script

After selecting a file, a default install script will be pre-filled. If the software package requires a custom installation process (for example, if [an EXE-based Windows installer requires custom handling](https://fleetdm.com/learn-more-about/exe-install-scripts)), this script can be edited. When the script is run, the `$INSTALLER_PATH` environment variable will be set by `fleetd` to where the installer is being run.

#### Post-install script

A post-install script will run after the installation, allowing you to, for example, configure the security agent right after installation. If this script returns a non-zero exit code, the installation will fail, and `fleetd` will attempt to uninstall the software.

#### Uninstall script

An uninstall script will run when an admin chooses to uninstall the software from the host on the host details page, or if an install fails for hosts running `fleetd` 1.33.0 or later. Like the install script, a default uninstall script will be pre-filled after selecting a file. This script can be edited if the software package requires a custom uninstallation process.

In addition to the `$INSTALLER_PATH` environment variable supported by install scripts, you can use `$PACKAGE_ID` in uninstall scripts as a placeholder for the package IDs (for .pkg files), package name (for Linux installers), product code (for MSIs), or software name (for EXE installers). The Fleet server will substitute `$PACKAGE_ID` on upload.

### Install a software package on a host

After a software package is added to a team, it can be installed on hosts via the UI.

* **Navigate to the Hosts page**: Click on the "Hosts" tab in the main navigation menu.

* **Navigate to the Host details page**: Click the host you want to install the software package.

* **Navigate to the Host software tab**: In the host details, search for the tab named “Software.”

* **Find your software package**: Use the dropdown to select software “Available for install” or use the search bar to search for your software package by name.

* **Install the software package on the host**: In the rightmost column of the table, click on “Actions” > “Install.” Installation will happen automatically or when the host comes online.

* **Track installation status**: by either

    * Checking the status column in the host software table.

    * Navigate to the “Details” tab on the host details page and check the activity log.

### Edit a software package

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team (or "No team") to switch to the team whose software you want to edit.

* **Find your software**: using the filters on the top of the table, you can choose between:

    * “Available for install” filters software can be installed on your hosts.

    * “Self-service” filters software that users can install from Fleet Desktop.

* **Select software package**: Click on a software package to view details.

* **Edit software package**: From the Actions menu, select "Edit."

> Editing the pre-install query, install script, post-install script, or uninstall script cancels all pending installations and uninstallations for that package, except for installs and uninstalls that are currently running on a host. If a new software package is uploaded, in addition to canceling pending installs and uninstalls, host counts (for installs and pending and failed installs and uninstalls) will be reset to zero, so counts reflect the currently uploaded version of the package.

### Uninstall a software package on a host

After a software package is installed on a host, it can be uninstalled on the host via the UI.

* **Navigate to the Hosts page**: Click on the "Hosts" tab in the main navigation menu.

* **Navigate to the Host details page**: Click the host you want to uninstall the software package.

* **Navigate to the Host software tab**: In the host details, search for the tab named “Software.”

* **Find your software package**: Use the dropdown to select software “Available for install” or use the search bar to search for your software package by name.

* **Uninstall the software package from the host**: In the rightmost column of the table, click on “Actions” > “Uninstall.”  Uninstallation will happen automatically or when the host comes online.

* **Track uninstallation status**: by either

    * Checking the status column in the host software table.

    * Navigate to the “Details” tab on the host details page and check the activity log.

### Remove a software package from a team

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team (or "No team") to switch to the team whose software you want to remove.

* **Find your software**: using the filters on the top of the table, you can choose between:

    * “Available for install” filters software can be installed on your hosts.

    * “Self-service” filters software that users can install from Fleet Desktop.

* **Select software package**: Click on a software package to view details.

* **Remove software package**: From the Actions menu, select "Delete." Click the "Delete" button on the dialog.

> Removing a software package from a team will cancel pending installs for hosts that are not in the middle of installing the software but will not uninstall the software from hosts where it is already installed.

### Manage software with the REST API

Fleet also provides a REST API for managing software programmatically. The API allows you to add, update, retrieve, list, and delete software. Detailed documentation on Fleet's [REST API is available]([https://fleetdm.com/docs/rest-api/rest-api#software](https://fleetdm.com/docs/rest-api/rest-api#software)), including endpoints for installing and uninstalling packages.

### Manage software with GitOps

Software packages can be managed via `fleetctl` using [GitOps](https://fleetdm.com/docs/using-fleet/gitops).

Please refer to the documentation for [managing software with GitOps](https://fleetdm.com/docs/using-fleet/gitops#software), for a real-world example, [see how we manage software at Fleet](https://github.com/fleetdm/fleet/tree/main/it-and-security/teams).

> When managing software installers via GitOps, the Fleet server receiving GitOps requests (**not** the machine running fleetctl as part of the GitOps workflow) will download installers from the specified URLs directly.

## Conclusion

Managing software with Fleet is straightforward and ensures your hosts are equipped with the latest tools. This guide has outlined how to access, add, edit, and remove software packages from a team, install and uninstall from specific hosts, and use the REST API and `fleetctl` to manage software packages. By following these steps, you can effectively maintain software packages across your fleet.

For more information on advanced setups and features, explore Fleet’s [documentation](https://fleetdm.com/docs/using-fleet) and additional [guides](https://fleetdm.com/guides).

<meta name="articleTitle" value="Deploy software packages">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-23">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-security-agents-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding and editing software packages in Fleet.">
