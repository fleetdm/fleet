# macOS MDM setup

To turn on macOS, iOS, and iPadOS MDM features, follow the instructions on this page to connect Fleet to Apple Push Notification service (APNs).

To use automatic enrollment (aka zero-touch) features on macOS, iOS, and iPadOS, follow instructions to connect Fleet with Apple Business Manager (ABM).

To turn on Windows MDM features, head to this [Windows MDM setup article](https://fleetdm.com/guides/windows-mdm-setup). 

## Apple Push Notification service (APNs)

Apple uses APNs to authenticate and manage interactions between Fleet and hosts.

To connect Fleet to APNs or renew APNs, head to the **Settings > Integrations > Mobile device management (MDM)** page.

> Apple requires that APNs certificates are renewed annually. 
> - If your certificate expires, you will have to turn MDM off and back on for all macOS hosts.
> - Be sure to use the same Apple ID from year-to-year. If you don't, you will have to turn MDM off and back on for all macOS hosts.

## Apple Business Manager (ABM)

> Available in Fleet Premium

To connect Fleet to ABM, you have to add an ABM token to Fleet. To add an ABM token: 

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under "Automatic enrollment", click "Add ABM", and then click "Add ABM" again on the next page. Follow the instructions in the modal and upload an ABM token to Fleet.

When one of your uploaded ABM tokens has expired or is within 30 days of expiring, you will see a warning
banner at the top of page reminding you to renew your token.

To renew an ABM token:

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under "Automatic enrollment", click "Edit", and then fin

After connecting Fleet to ABM, set Fleet to be the MDM for all Macs: 

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment** and click **Edit** next to **Default Server Assignment**.
5. Switch **Mac**, **iPhone**, and **iPad** to Fleet.

macOS, iOS, and iPadOS hosts listed in ABM and associated to a Fleet instance with MDM enabled will sync to Fleet and appear in the Hosts view with the **MDM status** label set to "Pending".

Hosts that automatically enroll will be assigned to a default team. You can configure the default team for macOS, iOS, and iPadOS hosts by:

1. Navigating to the **Settings > Integrations > Mobile device management (MDM)** page and clicking "Edit" under "Automatic enrollment".
2. Click on the "Actions" dropdown for the ABM token you want to update, and then click "Edit teams".
3. Use the dropdowns in the modal to select the default team for each type of host, and click "Save" to save your selections.

If no default team is set for a host platform (macOS, iOS, or iPadOS), then newly enrolled hosts of that platform will be placed in "No team". 

> A host can be transferred to a new (not default) team before it enrolls. In the Fleet UI, you can do this under **Settings** > **Teams**.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-07-02">
<meta name="articleTitle" value="macOS MDM setup">
<meta name="description" value="Learn how to turn on MDM features in Fleet.">
