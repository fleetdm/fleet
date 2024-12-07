# macOS MDM setup

To turn on macOS, iOS, and iPadOS MDM features, follow the instructions on this page to connect Fleet to Apple Push Notification service (APNs).

To use automatic enrollment (aka zero-touch) features on macOS, iOS, and iPadOS, follow instructions to connect Fleet with Apple Business Manager (ABM).

To turn on Windows MDM features, head to this [Windows MDM setup article](https://fleetdm.com/guides/windows-mdm-setup).

## Turn on Apple MDM

Apple uses APNs to authenticate and manage interactions between Fleet and hosts.

To connect Fleet to APNs or renew APNs, head to the **Settings > Integrations > Mobile device management (MDM)** page. 

Then click **Turn on** under the Apple (macOS, iOS, iPadOS) MDM section.

> Apple requires that APNs certificates are renewed annually.
> - The recommended approach is to use a shared admin account to generate the CSR ensuring it can be renewed regardless of individual availability.
> - If your certificate expires, you will have to turn MDM off and back on for all macOS hosts.
> - Be sure to use the same Apple ID from year-to-year. If you don't, you will have to turn MDM off and back on for all macOS hosts.

## Automatic enrollment

> Available in Fleet Premium

Add your ABM to automatically enroll newly purchased Apple hosts when they're first unboxed and set up by your end users.

To connect Fleet to ABM, you have to add an ABM token to Fleet. To add an ABM token: 

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under "Automatic enrollment", click "Add ABM", and then follow the instructions in the modal to upload an ABM token to Fleet.

When one of your uploaded ABM tokens has expired or is within 30 days of expiring, you will see a warning banner at the top of page reminding you to renew your token.

To renew an ABM token:

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under "Automatic enrollment", click "Edit", and then find the token that you want to renew. Token status is indicated in the "Renew date" column: tokens less than 30 days from expiring will have a yellow indicator, and expired tokens will have a red indicator. Click the "Actions" dropdown for the token and then click "Renew". Follow the instructions in the modal to download a new token from Apple Business Manager and then upload the new token to Fleet.

After connecting Fleet to ABM, set Fleet to be the MDM for all Macs: 

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment** and click **Edit** next to **Default Server Assignment**.
5. Switch **Mac**, **iPhone**, and **iPad** to Fleet.

macOS, iOS, and iPadOS hosts listed in ABM and associated to a Fleet instance with MDM enabled will sync to Fleet and appear in the Hosts view with the **MDM status** label set to "Pending".

Hosts that automatically enroll will be assigned to a default team. You can configure the default team for macOS, iOS, and iPadOS hosts by:

1. Creating teams, if you have not already, following [this guide](https://fleetdm.com/guides/teams#basic-article). Our [best practice](#best-practice) recommendation is to have a team for each device type.
2. Navigating to the **Settings > Integrations > Mobile device management (MDM)** page and clicking "Edit" under "Automatic enrollment".
3. Clicking on the "Actions" dropdown for the ABM token you want to update, and then clicking "Edit teams".
4. Using the dropdowns in the modal to select the default team for each type of host, and clicking "Save" to save your selections.

> If no default team is set for a host platform (macOS, iOS, or iPadOS), then newly enrolled hosts of that platform will be placed in "No team". 

> A host can be transferred to a new (not default) team before it enrolls. In the Fleet UI, you can do this under **Settings** > **Teams**.

## Volume Purchasing Program (VPP)

> Available in Fleet Premium

To connect Fleet to Apple's VPP, head to the guide [here](https://fleetdm.com/guides/install-vpp-apps-on-macos-using-fleet).

## Best practice

Most organizations only need one ABM token and one VPP token to manage their macOS, iOS, and iPadOS hosts.

These organizations may need multiple ABM and VPP tokens:

- Managed Service Providers (MSPs)
- Enterprises that acquire new businesses and as a result inherit new hosts
- Umbrella organizations that preside over entities with separated purchasing authority (i.e. a hospital or university) 

For **MSPs**, the best practice is to have one ABM and VPP connection per client. 

The default teams in Fleet for each client's ABM token in Fleet will look like this:
- macOS: 💻 Client A - Workstations
- iOS: 📱🏢 Client A - Company-owned iPhones
- iPadOS:🔳🏢 Client A - Company-owned iPads

Client A's VPP token will be assigned to the above teams.

For **enterprises that acquire**, the best practice is to add a new ABM and VPP connection for each acquisition.

These will default teams in Fleet:

Enterprise ABM token:
- macOS: 💻 Enterprise - Workstations
- iOS: 📱🏢 Enterprise - Company-owned iPhones
- iPadOS:🔳🏢 Enterprise - Company-owned iPads

The enterprises's VPP token will be assigned to the above teams.

Acquisition ABM token:
- macOS: 💻 Acquisition - Workstations
- iOS: 📱🏢 Acquisition - Company-owned iPhones
- iPadOS:🔳🏢 Acquisition - Company-owned iPads

The acquisitions's VPP token will be assigned to the above teams.

## Simple Certificate Enrollment Protocol (SCEP)

Fleet uses SCEP certificates (1 year expiry) to authenticate the requests hosts make to Fleet. Fleet renews each host's SCEP certificates automatically every 180 days.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-07-02">
<meta name="articleTitle" value="macOS MDM setup">
<meta name="description" value="Learn how to turn on MDM features in Fleet.">
