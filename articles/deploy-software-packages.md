# Deploy software

![Deploy software](../website/assets/images/articles/deploy-security-agents-1600x900@2x.png)

_Available in Fleet Premium_

In Fleet you can deploy [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps), [App Store (VPP) apps](https://fleetdm.com/guides/install-app-store-apps), and custom packages to your hosts.

This guide will walk you through steps to manually install custom packages on your hosts.

Learn more about automatically installing software [the Automatically install software guide](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Prerequisites

* `fleetd` deployed with the `--enable-scripts` flag. If you're using MDM features, scripts are enabled by default.

* If you're self-hosting Fleet, you need an S3 bucket [configured](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3-software-installers-bucket) to store the packages. Increase any load balancer timeouts to at least 5 minutes for the [Add package](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Modify package](https://fleetdm.com/docs/rest-api/rest-api#modify-package) API endpoints.

## Add a custom package

* Navigate to the **Software** page.
* Select a team (or "No team")
> Software cannot be added to "All teams."
* Click the **Add software** button in the top right corner.
* Select the **Custom package** tab.
* Choose a file to upload. `.pkg`, `.msi`, `.exe`, `.rpm`, `.deb`, `.ipa`, `.tar.gz`, `.sh`, and `.ps1` files are supported.
* If you check the **Automatic install** box, Fleet will create a policy that checks for the existence of the software and will automatically trigger an install on hosts where the software does not exist. 

> **Note:** Automatic install is not supported for payload-free packages (`.sh` and `.ps1` files).
* To allow users to install the software from Fleet Desktop, check the **Self-service** checkbox.
* To customize installer behavior, click on **Advanced options**.

> After the initial package upload, all options, except for automatic install, can be modified. This includes the self-service setting, pre-install query, scripts, and the software package file. However, if the installer package needs to be replaced, the new package must be of the same file type (such as .pkg, .msi, .exe, .deb, .rpm, or .ipa) and for the same software as the original. Files in .dmg or .zip formats cannot be edited or uploaded for replacement. If you want to enable automatic installs after initial package upload, follow the steps in our [automatic software install guide](https://fleetdm.com/guides/automatic-software-install-in-fleet) to add an automatic install policy.

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
- [.ipa extractor code](https://github.com/fleetdm/fleet/blob/main/pkg/file/ipa.go#:~:text=func%20ExtractIPAMetadata)

.tar.gz archives are uploaded as-is without attempting to pull metadata, and will be added successfully as long as they are valid archives, and as long as install and uninstall scripts are supplied.

### Payload-free packages

Payload-free packages (`.sh` and `.ps1` files) are packages that only contain a script that runs directly on hosts without installing traditional software. The script file's contents become the install script.  The `.sh` files are supported for Linux hosts, and`.ps1` files for Windows hosts.

Payload-free packages are useful for:
- Self-service configuration scripts (e.g., connecting to a VPN, configuring printers)
- Running maintenance tasks on demand
- Deploying configuration changes that don't require a traditional installer


Script packages do not support `install_script` (the file contents are the install script), `uninstall_script`, `post_install_script`, `pre_install_query`, and automatic install.

If these parameters are provided when uploading a script package, they will be ignored.


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

Fleet also provides an `$UPGRADE_CODE` placeholder for MSIs. This placeholder is replaced with the `UpgradeCode` extracted from the MSI on upload. If Fleet cannot extract an upgrade code from an MSI when using the default uninstall script, Fleet will use a product code-based uninstall script instead. If Fleet cannot extract an upgrade code from an MSI when a user-provided uninstall script uses the placeholder, the software upload will fail.

> Currently, the default MSI uninstaller script only uninstalls that exact installer, rather than earlier/later versions of the same application.

> Uninstall scripts do _not_ download the installer package to a host before running; if a .tar.gz archive includes an uninstall script, the contents of that script and any dependencies should be copied into the uninstall script text field rather than referred to by filename.

## Install the package

After a software package is added to a team, it can be installed on hosts via the UI.

* Navigate to the **Hosts** page.
* Select the host where you want to install the software package.
* Select **Software > Library** on the host details page.
* Use the dropdown to select software **Available for install** or use the search bar to search for your software by name.
* Select **Install** action in the rightmost column of the table. Install will happen automatically or when the host comes online.
* Check install status in the **Host > Software > Library > Status column** or the **Host > Details > Activity**.

Once the package is installed, Fleet will automatically refetch the host's vitals and update the software inventory.

## Edit the package

* Navigate to the **Software** page, choose a team, and select the software you want to edit.
  * Use a dropdown above the table to filter software **Available for install** or software available in **Self-service**.
* On the **Software details** page select **Actions > Edit software** to edit the software's [self-service](https://fleetdm.com/guides/software-self-service) status, change its target to different sets of hosts, or edit advanced options like pre-install query, install script, post-install script, and uninstall script.
* Select **Actions > Edit appearance** to edit the software's icon and display name. The icon and display name can be edited for software that is available for install. The new icon and display name will appear on the software list and details pages for the team where the package is uploaded, as well as on **My device > Self-service**. If the display name is not set, then the default name (ingested by osquery) will be used.

> Editing the advanced options cancels all pending installations and uninstallations for that package. Installs and uninstalls currently running on a host will complete, but results won't appear in Fleet. The software's host counts will be reset.

## Uninstall the package

After a software package is installed on a host, it can be uninstalled on the host via the UI.

* Navigate to the **Hosts** page.
* Select the host from which you want to uninstall the software package.
* Select **Software > Library** on the host details page.
* Use the dropdown to select software **Available for install** or use the search bar to search for your software by name.
* Select **Install** action in the rightmost column of the table. Uninstall will happen automatically or when the host comes online.
* Check uninstall status in the **Host > Software > Library > Status column** or the **Host > Details > Activity**.

## Delete package

* Navigate to the **Software** page, choose a team, and select the software you want to edit.
  * You can use a dropdown above the table to filter software **Available for install** or software available in **Self-service**.
* On **Software details** page select **Delete** icon in the section where you can see uploaded package file.

> Deleting a software package from a team will cancel pending installs for hosts that are not in the middle of installing the software but will not uninstall the software from hosts where it is already installed.

## Manage packages with Fleet's REST API

Fleet also provides a REST API for managing software programmatically. The API allows you to add, update, retrieve, list, and delete software. Detailed documentation on Fleet's [REST API is available](https://fleetdm.com/docs/rest-api/rest-api#software), including endpoints for installing and uninstalling packages.

## Manage packages with GitOps

Software packages can be managed via `fleetctl` using [GitOps](https://fleetdm.com/docs/using-fleet/gitops).

Please refer to the documentation for [managing software with GitOps](https://fleetdm.com/docs/using-fleet/gitops#software), for a real-world example, [see how we manage software at Fleet](https://github.com/fleetdm/fleet/tree/main/it-and-security/teams).

> When managing software packages via GitOps, the Fleet server receiving GitOps requests (**not** the machine running fleetctl as part of the GitOps workflow) will download installers from the specified URLs directly.

<meta name="articleTitle" value="Deploy software">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-05-05">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-security-agents-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding and editing software packages in Fleet.">
