# Installing Fleet maintained apps on macOS hosts

Fleet has introduced a new premium feature that allows you to install **Fleet maintained apps** directly on
your macOS hosts without having to upload packages manually. This simplifies the process and adds
another source of applications for your fleet.

Fleet starts off with some of the most common and popular apps, enabling you to pull directly from
this curated list and install them on your hosts without any additional configuration.

## Fleet maintained app installation flow

Let’s take a look at the fleet maintained app installation flow.

### Navigating the add software pages

Previously, when you clicked "Add Software," a modal would appear. However, Fleet has restructured
this flow. Now, when you click **Add Software**, you are taken to a page with a more streamlined
structure, offering three different options for adding software:

1. **Fleet-maintained**: A list of apps curated and maintained by Fleet.
2. **App store (VPP)**: If you have a VPP (Volume Purchase Program) configured, you can install apps
   from this connection.
3. **Custom package**: Allows you to upload your custom installers.

This new layout is cleaner and helps better manage the complexity of adding software.

### Adding a Fleet Maintained App

Let's dive into adding an app from the Fleet maintained list.

1. From the **Add software** page, navigate to the **Fleet-maintained** tab.
2. You’ll see a list of popular apps like Chrome, VS Code, and Notion. Select the desired app by
   clicking on the row in the table.
3. After selecting the app, you will be taken to the app details page. Here, you can set the app as a
self-service app, which will allow hosts to install this app on their own. You can also expand the
**Advanced options** which will allow you to edit the following:
   - Pre-install query
   - Installation script
   - Post-install script
   - Uninstall scripts

   These scripts are auto-generated based on the app's Homebrew formula, but you can modify them as
   needed. For example, you can customize the post-install script to perform additional tasks after
   installation.

### Installing the app

Once configured, click **Add Software**. This will download the installer from Homebrew and apply
the necessary installation scripts. The process may take a moment as it pulls the package.

Once completed, the app will be available for install on your hosts.

The app can now be installed on a host in the **Host Details** page under the **Software** tab. You can
select the app you just added and select **Install** from the actions dropdown.

If the app was marked as self-service the host user will be able to install it themselves in the
**Self-service** tab on the **My Device** page.

The install process is visible in the **Activities** tab on this **Host Details** page.

The install process is visible in the  **Activities** section on
the **Details** tab of this **Host Details** page.

### Uninstalling the app

To remove the app, simply select **Uninstall** from the same actions dropdown. Fleet will run the
uninstall script you configred on the host, ensuring a clean removal of the app.

The uninstallation process is also visible in the  **Activities** section on
the **Details** tab of this **Host Details** page.

## How does Fleet maintain these apps?

Fleet checks Homebrew every hour for updates to app definitions. When you add an app, Fleet
downloads the latest version available. However, Fleet does not automatically update apps. To get
the latest version, you’ll need to uninstall the app and reinstall it from the Fleet maintained
list.
