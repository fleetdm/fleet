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

To connect Fleet to ABM or renew ABM, head to the **Settings > Integrations > Automatic enrollment > Apple Business Manager** page.

After connecting Fleet to ABM, set Fleet to be the MDM for all Macs: 

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment** and click **Edit** next to **Default Server Assignment**.
5. Switch **Mac**, **iPhone**, and **iPad** to Fleet.

New or wiped macOS, iOS, and iPadOS hosts that are in ABM, before they've been set up, appear in Fleet with **MDM status** set to "Pending".

All macOS hosts that automatically enroll will be assigned to the default team. If no default team is set, then the host will be placed in "No team". 

> A host can be transferred to a new (not default) team before it enrolls. In the Fleet UI, you can do this under **Settings** > **Teams**.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-07-02">
<meta name="articleTitle" value="macOS MDM setup">
<meta name="description" value="Learn how to turn on MDM features in Fleet.">
