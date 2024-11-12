# Fleet-maintained apps

_Available in Fleet Premium_

In Fleet, you can install Fleet-maintained apps on macOS hosts without the need for manual uploads or extra configuration. This simplifies the process and adds another source of applications for your fleet.

Fleet starts with some of the most common and popular apps, enabling you to pull directly from this curated list and install them on your hosts without any additional configuration.

## Add a Fleet-maintained app

1. Head to the **Software** page and click **Add software**.
2. From the **Add software** page, navigate to the **Fleet-maintained** tab.
3. You’ll see a list of popular apps, such as Chrome, Visual Studio Code, and Notion. Click on a row in the table to select the desired app.
4. You will be taken to the app details page after selecting the app. Here, you can set the app as a self-service app, allowing hosts to install it on demand. You can also expand the **Advanced options**, which will enable you to edit the following:

   - Pre-install query
   - Installation script
   - Post-install script
   - Uninstall scripts

These scripts are auto-generated based on the app's Homebrew Cask formula, but you can modify them. Modifying these scripts allows you to tailor the app installation process to your organization's needs, such as automating additional setup tasks or custom configurations post-installation.

## Install the app

Once configured, click **Add Software**. This will download the installer specified in the Homebrew Cask and apply the installation scripts. The process may take a moment as it pulls the package.

Once completed, the app will be available for install on your hosts.

When you add a Fleet-maintained app, Fleet downloads the latest version available to a configured s3 bucket. The Host downloads the package through Fleet from s3 at install.

The app can now be installed on a host in the **Host Details** page under the **Software** tab. Select the app you just added and choose Install from the actions dropdown to do so. Alternatively, host users can install the app via the **Self-service** tab on the **My Device** page if you've marked the app as self-service. You can learn more about [Software self-service](https://fleetdm.com/guides/software-self-service).

You can track the installation process in the **Activities** section on the **Details** tab of this **Host Details** page.

## Uninstall the app

To remove the app, select **Uninstall** from the same actions dropdown. Fleet will run the uninstall script you configured on the host, ensuring a clean app removal.

The uninstallation process is also visible in the  **Activities** section on the **Details** tab of this **Host Details** page.

## Update the app

Currently, Fleet does not automatically update apps. To update the app, remove the app and re-add it from the Fleet-maintained list on the **Software** page, then reinstall it.

## How does Fleet maintain these apps?

Fleet:

- verifies, installs, uninstalls & tests all Fleet-maintained apps
- verifies the translation of all Homebrew scripts we use
- checks Homebrew cask metadata at [Homebrew Casks](https://github.com/Homebrew/homebrew-cask) every hour for updates to Fleet-maintained app definitions
- DOES NOT directly pull casks from Homebrew sources to computers

<meta name="category" value="guides">
<meta name="authorFullName" value="Gabriel Hernandez">
<meta name="authorGitHubUsername" value="ghernandez345">
<meta name="publishedOn" value="2024-10-16">
<meta name="articleTitle" value="Fleet-maintained apps">
