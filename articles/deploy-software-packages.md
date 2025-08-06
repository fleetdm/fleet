# Deploy software

![Deploy software](../website/assets/images/articles/deploy-security-agents-1600x900@2x.png)

_Available in Fleet Premium_

In Fleet you can deploy [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps), [App Store (VPP) apps](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet), and custom packages to your hosts.

This guide will walk you through steps to manually install custom packages on your hosts.

Learn more about automatically installing software [the Automatically install software guide](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Prerequisites

* `fleetd` deployed with the `--enable-scripts` flag. If you're using MDM features, scripts are enabled by default.

* If you're self-hosting Fleet, you need an S3 bucket [configured](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3-software-installers-bucket) to store the packages. Increase any load balancer timeouts to at least 5 minutes for the [Add package](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Modify package](https://fleetdm.com/docs/rest-api/rest-api#modify-package) API endpoints.

## Add a custom package

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team "No team" to add a software package.

> Software cannot be added to "All teams."

* Click the “Add Software” button in the top right corner.

* Select the "Custom package" tab.

* Choose a file to upload. `.pkg`, `.msi`, `.exe`, `.rpm`, `.deb`, and `.tar.gz` files are supported.

* If you check the "Automatic install" box, Fleet will create a policy that checks for the existence of the software and will automatically trigger an install on hosts where the software does not exist.

* To allow users to install the software from Fleet Desktop, check the “Self-service” checkbox.

* To customize installer behavior, click on “Advanced options.”

> After the initial package upload, all options, except for automatic install, can be modified. This includes the self-service setting, pre-install query, scripts, and the software package file. However, if the installer package needs to be replaced, the new package must be of the same file type (such as .pkg, .msi, .deb, .rpm, or .exe) and for the same software as the original. Files in .dmg or .zip formats cannot be edited or uploaded for replacement. If you want to enable automatic installs after initial package upload, follow the steps in our [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet) to add an automatic install policy.

### Package metadata extraction

The following metadata is used in uninstall scripts and policies that trigger automatic installation to check whether the software is already installed:
- bundle identifier (`.pkg`)
- product code (`.msi`)
- name (`.deb`, `.rpm`)

Software installer uploads will fail if Fleet can't extract this metadata and version number. For more details check the extractor code for each package type.
- [.pkg extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/xar.go#:~:text=func%20ExtractXARMetadata)
- [.msi extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/msi.go#:~:text=func%20ExtractMSIMetadata)
- [.exe extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/pe.go#:~:text=func%20ExtractPEMetadata)
- [.deb extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/deb.go#:~:text=func%20ExtractDebMetadata)
- [.rpm extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/rpm.go#:~:text=func%20ExtractRPMMetadata)

.tar.gz archives are uploaded as-is without attempting to pull metadata, and will be added successfully as long as they are valid archives, and as long as install and uninstall scripts are supplied.

### Pre-install query

A pre-install query is a valid osquery SQL statement that will be evaluated on the host before installing the software. If provided, the installation will proceed only if the query returns any value.

### Install script

After selecting a file, a default install script will be pre-filled for most installer types. If the software package requires a custom installation process (for example, for .tar.gz archives and [EXE-based Windows installers](https://fleetdm.com/learn-more-about/exe-install-scripts)), this script can be edited. When the script is run, the `$INSTALLER_PATH` environment variable will be set by `fleetd` to where the installer is being run. `$INSTALLER_PATH` will be inside a temporary directory created by the operating system (e.g. `/tmp/[random string]` on Linux hosts).

> For .tar.gz archives, fleetd 1.42.0 or later will extract the archive into `$INSTALLER_PATH` before handing control over to your install script, and will clean this directory up after the install script concludes.

### Post-install script

A post-install script will run after the installation, allowing you to, for example, configure the security agent right after installation. If this script returns a non-zero exit code, the installation will fail, and `fleetd` will attempt to uninstall the software.

### Uninstall script

An uninstall script will run when an admin chooses to uninstall the software from the host on the host details page, or if the post-install script (if supplied) fails. Like the install script, a default uninstall script will be pre-filled after selecting a file for most installer formats, other than EXE-based installers and .tar.gz archives. This script can be edited if the software package requires a custom uninstallation process.

You can use `$PACKAGE_ID` in uninstall scripts as a placeholder for the package IDs (for .pkg files), package name (for Linux installers), product code (for MSIs), or software name (for EXE installers). The Fleet server will substitute `$PACKAGE_ID` on upload.

> Currently, the default MSI uninstaller script only uninstalls that exact installer, rather than earlier/later versions of the same application.

> Uninstall scripts do _not_ download the installer package to a host before running; if a .tar.gz archive includes an uninstall script, the contents of that script and any dependencies should be copied into the uninstall script text field rather than referred to by filename.

## Install the package

After a software package is added to a team, it can be installed on hosts via the UI.

* **Navigate to the Hosts page**: Click on the "Hosts" tab in the main navigation menu.

* **Navigate to the Host details page**: Click the host you want to install the software package.

* **Navigate to the Host software tab**: In the host details, search for the tab named “Software.”

* **Find your software package**: Use the dropdown to select software “Available for install” or use the search bar to search for your software package by name.

* **Install the software package on the host**: In the rightmost column of the table, click on “Actions” > “Install.” Installation will happen automatically or when the host comes online.

* **Track installation status**: by either

    * Checking the status column in the host software table.

    * Navigate to the “Details” tab on the host details page and check the activity log.

Once the package is installed, Fleet will automatically refetch the host's vitals and update the software inventory.

## Edit the package

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team (or "No team") to switch to the team whose software you want to edit.

* **Find your software**: using the filters on the top of the table, you can choose between:

    * “Available for install” filters software can be installed on your hosts.

    * “Self-service” filters software that users can install from Fleet Desktop.

* **Select software package**: Click on a software package to view details.

* **Edit software package**: From the Actions menu, select "Edit." You can edit the package's [self-service](https://fleetdm.com/guides/software-self-service) status, change its target to different sets of hosts, or edit advanced options like pre-install query, install script, post-install script, and uninstall script.

> Editing the advanced options cancels all pending installations and uninstallations for that package. Installs and uninstalls currently running on a host will complete, but results won't appear in Fleet. The software's host counts will be reset.

## Uninstall the package

After a software package is installed on a host, it can be uninstalled on the host via the UI.

* **Navigate to the Hosts page**: Click on the "Hosts" tab in the main navigation menu.

* **Navigate to the Host details page**: Click the host you want to uninstall the software package.

* **Navigate to the Host software tab**: In the host details, search for the tab named “Software.”

* **Find your software package**: Use the dropdown to select software “Available for install” or use the search bar to search for your software package by name.

* **Uninstall the software package from the host**: In the rightmost column of the table, click on “Actions” > “Uninstall.”

* **Track uninstallation status**: by either

    * Checking the status column in the host software table.

    * Navigate to the “Details” tab on the host details page and check the activity log.

## Remove the package

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.

* **Select a team**: Select a team (or "No team") to switch to the team whose software you want to remove.

* **Find your software**: using the filters on the top of the table, you can choose between:

    * “Available for install” filters software can be installed on your hosts.

    * “Self-service” filters software that users can install from Fleet Desktop.

* **Select software package**: Click on a software package to view details.

* **Remove software package**: From the Actions menu, select "Delete." Click the "Delete" button on the dialog.

> Removing a software package from a team will cancel pending installs for hosts that are not in the middle of installing the software but will not uninstall the software from hosts where it is already installed.

## Manage packages with Fleet's REST API

Fleet also provides a REST API for managing software programmatically. The API allows you to add, update, retrieve, list, and delete software. Detailed documentation on Fleet's [REST API is available](https://fleetdm.com/docs/rest-api/rest-api#software), including endpoints for installing and uninstalling packages.

## Manage packages with GitOps

Software packages can be managed via `fleetctl` using [GitOps](https://fleetdm.com/docs/using-fleet/gitops).

Please refer to the documentation for [managing software with GitOps](https://fleetdm.com/docs/using-fleet/gitops#software), for a real-world example, [see how we manage software at Fleet](https://github.com/fleetdm/fleet/tree/main/it-and-security/teams).

> When managing software installers via GitOps, the Fleet server receiving GitOps requests (**not** the machine running fleetctl as part of the GitOps workflow) will download installers from the specified URLs directly.

<meta name="articleTitle" value="Deploy software">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-05-05">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-security-agents-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding and editing software packages in Fleet.">
