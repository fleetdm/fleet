# Installing Fleet-maintained apps on macOS hosts

Fleet’s new premium feature lets you quickly install **Fleet-maintained apps** on macOS hosts—no need for manual uploads or extra configuration. This simplifies the process and adds another source of applications for your fleet.

Fleet starts with some of the most common and popular apps, enabling you to pull directly from this curated list and install them on your hosts without any additional configuration.

## Prerequisites

* Fleet Premium is required for Fleet-maintained apps.

> Software packages can be added to a specific team or the "No team" category. The "No team" category is the default assignment for hosts not part of any specific team.

## Fleet-maintained app installation flow

Let’s take a look at the Fleet-maintained app installation flow.

### Navigating the add software pages

Click **Add software** to access three different options for adding software:

1. **Fleet-maintained**: A list of apps curated and maintained by Fleet.
2. **App store (VPP)**: If you have a [VPP (Volume Purchase Program)](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet) configured, you can install apps from this connection.
3. **Custom package**: You can upload your custom installers.

### Adding a Fleet-maintained app

1. From the **Add software** page, navigate to the **Fleet-maintained** tab.
2. You’ll see a list of popular apps, such as Chrome, Visual Studio Code, and Notion. Click on a row in the table to select the desired app.
3. You will be taken to the app details page after selecting the app. Here, you can set the app as a self-service app, allowing hosts to install it on demand. You can also expand the **Advanced options**, which will enable you to edit the following:
   - Pre-install query
   - Installation script
   - Post-install script
   - Uninstall scripts

   These scripts are auto-generated based on the app's Homebrew Cask formula, but you can modify them. Modifying these scripts allows you to tailor the app installation process to your organization's needs, such as automating additional setup tasks or custom configurations post-installation.

### Installing the app

Once configured, click **Add Software**. This will download the installer specified in the Homebrew Cask and apply the installation scripts. The process may take a moment as it pulls the package.

Once completed, the app will be available for install on your hosts.

The app can now be installed on a host in the **Host Details** page under the **Software** tab. Select the app you just added and choose Install from the actions dropdown to do so. Alternatively, host users can install the app via the **Self-service** tab on the **My Device** page if you've marked the app as self-service. You can learn more about [Software self-service](https://fleetdm.com/guides/software-self-service).

You can track the installation process in the **Activities** section on the **Details** tab of this **Host Details** page.

### Uninstalling the app

To remove the app, select **Uninstall** from the same actions dropdown. Fleet will run the uninstall script you configured on the host, ensuring a clean app removal.

The uninstallation process is also visible in the  **Activities** section on the **Details** tab of this **Host Details** page.

## How does Fleet maintain these apps?

Fleet checks [Homebrew Casks](https://github.com/Homebrew/homebrew-cask) every hour for updates to app definitions. When you add an app, Fleet downloads the latest version available. Currently, Fleet does not automatically update apps. To update the app, remove the app and re-add it from the Fleet-maintained list on the Software page, then reinstall it.

<meta name="category" value="guides">
<meta name="authorFullName" value="Gabriel Hernandez">
<meta name="authorGitHubUsername" value="ghernandez345">
<meta name="publishedOn" value="2024-10-16">
<meta name="articleTitle" value="Installing Fleet-maintained apps on macOS hosts.">
