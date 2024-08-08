# Migration guide

This section provides instructions for migrating your hosts away from your old MDM solution to Fleet.

## Requirements

1. A [deployed Fleet instance](../Deploying/Introduction.md)
2. [Fleet connected to Apple](./mdm-setup.md)

## Migrate manually enrolled hosts

1. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
2. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
3. In your old MDM solution, unenroll the hosts to be migrated. MacOS does not allow multiple MDMs to be installed at once.
4. Send [these guided instructions](#how-to-turn-on-mdm) to your end users to complete the final few steps via Fleet Desktop.
    * Note that there will be a gap in MDM coverage between when the host is unenrolled from the old MDM and when the host turns on MDM in Fleet.

### End user experience

1. On their **My device** page, once an end user's device is unenrolled from the old MDM solution, the end user will be given the option to manually download the MDM enrollment profile.
   
2. Once downloaded, the user will receive a system notification that the Device Enrollment profile needs to be installed in their **System Settings > Profiles** section. 
   
3. After installation, the MDM enrollment profile can be removed by the end user at any time.

### How to turn on MDM

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions. 
  - If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. 
  - If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

## Migrate automatically enrolled (DEP) hosts

> Automatic enrollment is available in Fleet Premium or Ultimate

To migrate automatically enrolled hosts, we will do the following steps:

1. Prepare to migrate hosts
2. Choose migration workflow and migrate hosts

### Step 1: prepare to migrate hosts

1. Connect Fleet to Apple Business Manager (ABM). Learn how [here](./mdm-setup.md#apple-business-manager-abm).
2. [Enroll](./Adding-hosts.md) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/docs/using-fleet/adding-hosts#including-fleet-desktop) 
3. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
4. Migrate your hosts to Fleet in ABM:
    1. In ABM, unassign the existing hosts' MDM server from the old MDM solution: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Unassign from the current MDM**, and select **Continue**.
    2. In ABM, assign these hosts' MDM server to Fleet: In ABM, select **Devices** and then select **All Devices**. Then, select **Edit** next to **Edit MDM Server**, select **Assign to the following MDM:**, select your Fleet server in the dropdown, and select **Continue**.

### Step 2: choose migration workflow and migrate hosts

There are two migration workflows in Fleet: default and end user.

The default migration workflow requires that the IT admin unenrolls hosts from the old MDM solution before the end user can complete migration. This will result in a gap in MDM coverage until the end user completes migration.

The end user migration workflow allows the end user to kick-off migration by unenrolling from the old MDM solution on their own. Once the user is unenrolled, they're prompted to turn on MDM features in Fleet. This reduces the gap in MDM coverage.

Configuring the end user migration workflow requires a few additional steps.

#### Default workflow

1. In your old MDM solution, unenroll the hosts to be migrated. MacOS does not allow multiple MDMs to be installed at once.

2. Send [these guided instructions](#how-to-turn-on-mdm-default) to your end users to complete the final few steps via Fleet Desktop.
    * Note that there will be a gap in MDM coverage between when the host is unenrolled from the old MDM and when the host turns on MDM in Fleet.

##### End user experience

1. The end user will receive a "Device Enrollment: &lt;organization&gt; can automatically configure your Mac." system notification within the macOS Notifications Center. 
   
2. After the end user clicks on the system notification, macOS will open the **System Setting > Profiles** and ask the user to "Allow Device Enrollment: &lt;organization&gt; can automatically configure your Mac based on settings provided by your System Administrator."
  
3. If the end user does not install the profile, the system notification will continue to prompt the end user until the setting has been allowed.
   
4. Once this setting has been approved, the MDM enrollment profile cannot be removed by the end user.

##### How to turn on MDM (default)

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions. 
    * If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. 
    * If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

#### End user workflow

> Available in Fleet Premium or Ultimate

The end user migration workflow is supported for automatically enrolled (DEP) hosts.

To watch a GIF that walks through the end user experience during the migration workflow, in the Fleet UI, head to **Settings > Integrations > Mobile device management (MDM)**, and scroll down to the **End user migration workflow** section.

In Fleet, you can configure the end user workflow using the Fleet UI or fleetctl command-line tool.

Fleet UI:

1. Select the avatar on the right side of the top navigation and select **Settings > Integrations > Mobile device management (MDM)**.

2. Scroll down to the **End user migration workflow** section and select the toggle to enable the workflow.

3. Under **Mode** choose a mode and enter the webhook URL for you automation tool (ex. Tines) under **Webhook URL** and select **Save**.

4. During the end user migration workflow, an end user's device will have their selected system theme (light or dark) applied. If your logo is not easy to see on both light and dark backgrounds, you can optionally set a logo for each theme:
Head to **Settings** > **Organization settings** >
**Organization info**, add URLs to your logos in the **Organization avatar URL (for dark backgrounds)** and **Organization avatar URL (for light backgrounds)** fields, and select **Save**.

fleetctl CLI:

1. Create `fleet-config.yaml` file or add to your existing `config` YAML file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_migration:
      enable: true
      mode: "voluntary"
      webhook_url: "https://example.com"
  ...
```

2. Fill in the above keys under the `mdm.macos_migration` key. 

To learn about each option, in the Fleet UI, select the avatar on the right side of the top navigation, select **Settings > Integrations > Mobile device management (MDM)**, and scroll down to the **End user migration workflow** section.

3. During the end user migration workflow, the window will show the Fleet logo on top of a dark and light background (appearance configured by end user).

If want to add a your organization's logo, you can optionally set a logo for each background:

```yaml
apiVersion: v1
kind: config
spec:
  org_info:
    org_logo_url: https://fleetdm.com/images/press-kit/fleet-blue-logo.png
    org_logo_url_light_background: https://fleetdm.com/images/press-kit/fleet-white-logo.png
  ...
```

Add URLs to your logos that are visible on a dark background and light background in the `org_logo_url` and `org_logo_url_light_background` keys respectively. If you only set a logo for one, the Fleet logo will be used for the other.

4. Run the fleetctl `apply -f fleet-config.yml` command to add your configuration.

5. Confirm that your configuration was saved by running `fleetctl get config`.

6. Send [these guided instructions](#how-to-turn-on-mdm-end-user) to your end users to complete the final few steps via Fleet Desktop.

##### How to turn on MDM (end user)

1. Select the Fleet icon in your menu bar and select **Migrate to Fleet**.

2. Select **Start** in the **Migrate to Fleet** popup. 

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions. 
    * If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page. 
    * If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

## Check migration progress

To see a report of which hosts have successfully migrated to Fleet, have MDM features off, or are still enrolled to your old MDM solution head to the **Dashboard** page by clicking the icon on the left side of the top navigation bar. 

Then, scroll down to the **Mobile device management (MDM)** section.

## FileVault recovery keys

_Available in Fleet Premium_

When migrating from a previous MDM, end users need to take action to escrow FileVault keys to Fleet. The **My device** page in Fleet Desktop will present users with instructions to reset their key. 

To start, enforce FileVault (disk encryption) and escrow in Fleet. Learn how [here](./MDM-disk-encryption.md). 

After turning on disk encryption in Fleet, share [these guided instructions](#how-to-turn-on-disk-encryption) with your end users.

If your old MDM solution did not enforce disk encryption, the end user will need to restart or log out of the host.

If your old MDM solution did enforce disk encryption, the end user will need to reset their disk encryption key by following the prompt on the My device page and inputting their password. 

## Activation Lock

In Fleet, the [Activation Lock](https://support.apple.com/en-us/HT208987) feature is disabled by default for automatically enrolled (DEP) hosts.

In 2024, Apple added the ability to manage activation lock in Apple Business Manager (ABM). For devices that are owned by the business and available in ABM, you can [turn off activation lock remotely](https://support.apple.com/en-ca/guide/apple-business-manager/axm812df1dd8/web).

If a device is not available in ABM and has Activation Lock enabled, we recommend asking the end user to follow these instructions to disable Activation Lock before migrating the device to Fleet: https://support.apple.com/en-us/HT208987.

This is because if the Activation Lock is enabled, you will need the Activation Lock bypass code to successfully wipe and reuse the Mac.

However, Activation Lock bypass codes can only be retrieved from the Mac up to 30 days after the device is enrolled. This means that when migrating from your old MDM solution, it’s likely that you’ll be unable to retrieve the Activation Lock bypass code.
   
### How to turn on disk encryption

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, follow the disk encryption instructions in the yellow banner. 
  - If you don’t see the yellow banner, select the purple **Refetch** button at the top of the page. 
  - If you still don't see the yellow banner after a couple minutes or if the **My device** page presents you with an error, please contact your IT administrator.

<img width="1399" alt="My device page - turn on disk encryption" src="https://user-images.githubusercontent.com/5359586/229950451-cfcd2314-a993-48db-aecf-11aac576d297.png">

<meta name="pageOrderInSection" value="1502">
<meta name="title" value="Migration guide">
<meta name="description" value="Instructions for migrating hosts away from an old MDM solution to Fleet.">
<meta name="navSection" value="Device management">
