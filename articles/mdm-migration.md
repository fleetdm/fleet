# MDM migration

This guide provides instructions for migrating devices from your current MDM solution to Fleet.

Migrating from an existing MDM solution to Fleet allows IT admins to consolidate device management and take advantage of Fleet's advanced monitoring and compliance features. This guide will help you smoothly transition your macOS devices to Fleet, whether they are manually enrolled or automatically enrolled (ADE).

> For seamless MDM migration, [view this guide](https://fleetdm.com/guides/seamless-mdm-migration).

## Requirements

- A [deployed Fleet instance](https://fleetdm.com/docs/deploy/deploy-fleet)
- Fleet is connected to Apple Push Notification service (APNs) and Apple Business Manager (ABM). [See macOS MDM setup](https://fleetdm.com/guides/macos-mdm-setup)

## FileVault recovery keys

_Available in Fleet Premium_

When migrating from a previous MDM, end users need to restart or logout of their device to escrow FileVault keys to Fleet. The **My device** page in Fleet Desktop will present users with instructions to reset their key.

To start, enforce FileVault disk encryption and escrow recovery keys in Fleet. Learn how [here](https://fleetdm.com/guides/enforce-disk-encryption).

After turning on disk encryption in Fleet, share [these guided instructions](#how-to-turn-on-disk-encryption) with your end users.

## Activation Lock

In Fleet, the [Activation Lock](https://support.apple.com/en-us/HT208987) feature is disabled by default for automatically enrolled (ADE) hosts.

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

## Migrate manually enrolled hosts

- **Manual Enrollment**: Devices that were manually enrolled in the previous MDM and need to be individually migrated to Fleet.
- **Automatic Enrollment**: Devices enrolled via Apple Business Manager’s automated device enrollment, which can be migrated more seamlessly.

To migrate manually enrolled hosts, we will do the following steps:

1. Prepare to migrate hosts
2. Choose migration workflow and migrate hosts

### Step 1: prepare to migrate hosts

To prepare your hosts for migration, first [enroll](https://fleetdm.com/guides/enroll-hosts) them in Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/guides/enroll-hosts#fleet-desktop).

### Step 2: choose a migration workflow and migrate hosts

There are two manual migration workflows in Fleet: default and end user.

The default migration workflow requires that the IT admin unenrolls hosts from the old MDM solution before the end user can complete migration. This will result in a gap in MDM coverage until the end user completes migration.

The end user migration workflow allows the end user to kick off migration by unenrolling from the old MDM solution on their own. Once the user is unenrolled, they're prompted to turn on MDM features in Fleet. This reduces the gap in MDM coverage.

Configuring the end user migration workflow requires a few additional steps.

#### Default workflow

1. Ensure your end users have access to an admin account on their Mac. End users won't be able to migrate on their own if they have a standard account.
2. In your old MDM solution, unenroll the hosts to be migrated. MacOS does not allow multiple MDMs to be installed at once.
3. Send [these guided instructions](#how-to-turn-on-mdm) to your end users to complete the final few steps via Fleet Desktop.
    * Note that there will be a gap in MDM coverage between when the host is unenrolled from the old MDM and when the host turns on MDM in Fleet.

##### End user experience

1. On their **My device** page, once an end user's device is unenrolled from the old MDM solution, the end user will be given the option to manually download the MDM enrollment profile.
2. Once downloaded, the user will receive a system notification that the Device Enrollment profile needs to be installed in their **System Settings > Profiles** section.
3. After installation, the MDM enrollment profile can be removed by the end user at any time.

##### How to turn on MDM

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions.
  - If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page.
  - If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

#### End user workflow

> Available in Fleet Premium

To watch an animation of the end user experience during the migration workflow, in the Fleet UI, head to **Settings > Integrations > Mobile device management (MDM)**, and scroll down to the **End user migration workflow** section.

In Fleet, you can configure the end user workflow using the Fleet UI or with GitOps using the `fleetctl` tool.

Fleet UI:

1. Select the avatar on the right side of the top navigation and select **Settings > Integrations > Mobile device management (MDM)**.

2. Scroll down to the **End user migration workflow** section and select the toggle to enable the workflow.

3. Under **Mode** choose a mode and enter the webhook URL for your automation tool (ex. Tines) under **Webhook URL** and select **Save**.

4. During the end user migration workflow, an end user's device will have their selected system theme (light or dark) applied. If your logo is not easy to see on both light and dark backgrounds, you can optionally set a logo for each theme:
Head to **Settings** > **Organization settings** >
**Organization info**, add URLs to your logos in the **Organization avatar URL (for dark backgrounds)** and **Organization avatar URL (for light backgrounds)** fields, and select **Save**.

GitOps:

To manage macOS MDM migration configuration using Fleet's best practice GitOps, check out the `macos_migration` key in the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#macos-migration).

To manage your organization's logo for dark and light backgrounds using Fleet's best practice GitOps, check out the `org_info` key in the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#org-info).


Once you've configured the end user workflow, send [these guided instructions](#how-to-turn-on-mdm-end-user) to your end users to complete the final few steps via Fleet Desktop.

## Migrate automatically enrolled (ADE) hosts

> Automatic enrollment is available in Fleet Premium

To migrate automatically enrolled hosts, we will do the following steps:

1. Prepare to migrate hosts
2. Choose migration workflow and migrate hosts

### Step 1: prepare to migrate hosts

1. Connect Fleet to Apple Business Manager (ABM). Learn how [here](https://fleetdm.com/guides/macos-mdm-setup#apple-business-manager-abm).
2. [Enroll](https://fleetdm.com/guides/enroll-hosts) your hosts to Fleet with [Fleetd and Fleet Desktop](https://fleetdm.com/guides/enroll-hosts#fleet-desktop)
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

1. The end user will see a system "Remote Management" modal appear over the screen.

2. After the end user clicks "Enroll" on the system modal, macOS will prompt them for their password and begin the enrollment process.

3. If the user clicks "Not now" on the system modal, the modal will be shown every 3 minutes until the user finishes the enrollment.

4. Once this setting has been approved, the MDM enrollment profile cannot be removed by the end user.

##### How to turn on MDM (default)

1. Select the Fleet icon in your menu bar and select **My device**.

![Fleet icon in menu bar](https://raw.githubusercontent.com/fleetdm/fleet/main/website/assets/images/articles/fleet-desktop-says-hello-world-cover-1600x900@2x.jpg)

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions.
    * If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page.
    * If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

<img width="1400" alt="My device page - turn on MDM" src="https://user-images.githubusercontent.com/5359586/229950406-98343bf7-9653-4117-a8f5-c03359ba0d86.png">

#### End user workflow

> Available in Fleet Premium

The end user migration workflow is supported for automatically enrolled (ADE) and manually enrolled hosts.

To watch a GIF that walks through the end user experience during the migration workflow, in the Fleet UI, head to **Settings > Integrations > Mobile device management (MDM)**, and scroll down to the **End user migration workflow** section.

In Fleet, you can configure the end user workflow using the Fleet UI or with GitOps using the `fleetctl` command-line tool.

Fleet UI:

1. Select the avatar on the right side of the top navigation and select **Settings > Integrations > Mobile device management (MDM)**.

2. Scroll down to the **End user migration workflow** section and select the toggle to enable the workflow.

3. Under **Mode** choose a mode and enter the webhook URL for your automation tool (ex. Tines) under **Webhook URL** and select **Save**.

4. During the end user migration workflow, an end user's device will have their selected system theme (light or dark) applied. If your logo is not easy to see on both light and dark backgrounds, you can optionally set a logo for each theme:
Head to **Settings** > **Organization settings** >
**Organization info**, add URLs to your logos in the **Organization avatar URL (for dark backgrounds)** and **Organization avatar URL (for light backgrounds)** fields, and select **Save**.

GitOps:

To manage macOS MDM migration configuration using Fleet's best practice GitOps, check out the `macos_migration` key in the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#macos-migration).

To manage your organization's logo for dark and light backgrounds using Fleet's best practice GitOps, check out the `org_info` key in the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#org-info).


Once you've configured the end user workflow, send [these guided instructions](#how-to-turn-on-mdm-end-user) to your end users to complete the final few steps via Fleet Desktop.

3. Send [these guided instructions](#how-to-turn-on-mdm-end-user) to your end users to complete the final few steps via Fleet Desktop.

##### How to turn on MDM (end user)

1. Select the Fleet icon in your menu bar and select **Migrate to Fleet**.

2. Select **Start** in the **Migrate to Fleet** popup.

2. On your **My device** page, select the **Turn on MDM** button in the yellow banner and follow the instructions.
    * If you don’t see the yellow banner or the **Turn on MDM** button, select the purple **Refetch** button at the top of the page.
    * If you still don't see the **Turn on MDM** button or the **My device** page presents you with an error, please contact your IT administrator.

## Check migration progress

To see a report of which hosts have successfully migrated to Fleet, have MDM features off, or are still enrolled to your old MDM solution head to the **Dashboard** page by clicking the icon on the left side of the top navigation bar.

Then, scroll down to the **Mobile device management (MDM)** section of the Dashboard, you'll see a breakdown of which hosts have successfully migrated to Fleet, which have MDM features disabled, and which are still enrolled in the previous MDM solution.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-08-14">
<meta name="articleTitle" value="MDM migration">
<meta name="description" value="Instructions for migrating hosts away from an old MDM solution to Fleet.">
