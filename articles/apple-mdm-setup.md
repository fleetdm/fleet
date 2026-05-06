# Apple MDM setup

To turn on macOS, iOS, and iPadOS MDM features, follow the instructions on this page to connect Fleet to Apple Push Notification service (APNs).

To use automatic enrollment (aka zero-touch) features on macOS, iOS, and iPadOS, follow instructions to connect Fleet with Apple Business (AB).

To turn on Windows MDM features, head to this [Windows MDM setup article](https://fleetdm.com/guides/windows-mdm-setup).

## Turn on Apple MDM

Apple uses Apple Push Notification service (APNs) APNs to authenticate and manage interactions between Fleet and hosts.

> Apple requires that APNs certificates are renewed annually.
> - The recommended approach is to use a shared admin account to generate the CSR ensuring it can be renewed regardless of individual availability.
> - If your certificate expires, you must turn MDM off and back on for all Apple hosts. Until then, configuration profile changes and other MDM commands will remain stuck in “Pending.”
> - Be sure to use the same Apple ID from year-to-year. If you don't, you will have to turn MDM off and back on for all Apple hosts.

How to connect Fleet to APNs:

1. In Fleet, navigate to the **Settings > Integrations > MDM** page.
2. Select **Turn on** for Apple (macOS, iOS, iPadOS) MDM.
3. Select **Download CSR** to download a certificate signing request (CSR) for Apple Push Notification service (APNs).
4. Sign in to [Apple Push Certificates Portal](https://identity.apple.com/pushcert/). If you don't have an Apple Account, create one.
5. In Apple Push Certificates Portal, select **Create a Certificate**, upload your CSR, and download your APNs certificate.
6. Upload APNs certificate (.pem file) in Fleet.

### Renew APNs

1. In Fleet, navigate to the **Settings > Integrations > MDM** page.
2. Select **Edit** next to **Apple MDM turned on**.
3. Select **Renew certificate** and then select **Download CSR** to download a certificate signing request (CSR) for Apple Push Notification service (APNs).
5. Sign in to [Apple Push Certificates Portal](https://identity.apple.com/pushcert/).
6. In Apple Push Certificates Portal, select **Renew** next to your certificate. Make sure that the certificate's **Common Name (CN)** matches the one presented in Fleet. If you choose a different certificate, you must turn MDM off and back on for all Apple hosts.
7. Upload your CSR and download new APNs certificate.
8. Upload APNs certificate (.pem file) in Fleet.

## Apple Business (AB)

> Available in Fleet Premium

Connect Fleet to your AB to allow automatic enrollment for company-owned and [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account) for personal (BYOD) macOS, iOS, and iPadOS hosts.

1. In Fleet, navigate to the **Settings > Integrations > MDM** page.
2. Under **Apple Business (AB)**, select **Add AB**.
3. Select **Download public key** to download a public key for AB.
4. Sign in to [Apple Business](https://business.apple.com). If your organization doesn't have an account, create one.
5. Select **Devices > Management** and select **Add** at the bottom of list.
7. Enter a name for the server such as "Fleet" and upload the public key downloaded in step 3 and select **Next**.
8. Download the service token and select **Done**.
9. In the **Default Device Assignment** section, assign the newly created server as the default for your Macs, iPhones, and iPads. Then select **Save**.
10. In Fleet, upload the service token (.p7m file) downloaded in step 8.

macOS, iOS, and iPadOS hosts listed in AB and assigned to a Fleet will sync to Fleet and appear in the Hosts view with the **MDM status** label set to "Pending".

When one of your uploaded AB tokens has expired or is within 30 days of expiring, you will see a warning banner at the top of page reminding you to renew your token.

### Renew AB:

> Token status is indicated in the **Renew date** column: tokens less than 30 days from expiring will have a yellow indicator, and expired tokens will have a red indicator.

1. Sign in to [Apple Business](https://business.apple.com/).
2. Select **Devices > Management** and select your MDM server.
3. Select the three dots and select **Download Token**.
4. In Fleet, navigate to the **Settings > Integrations > MDM** page.
5. Under **Apple Business (AB)** select **Edit** next to **Company-owned (ADE) and personal (BYOD) enrollment...**, and then find the token that you want to renew.
6. Select the **Actions > Renew** for the token.
7. Upload the token (.p7m file) downloaded in step 3.

### Hosts that automatically enroll will be assigned to a default fleet. You can configure the default fleet for macOS, iOS, and iPadOS hosts:

1. Create a fleet, if you have not already, following [this guide](https://fleetdm.com/guides/fleets).
2. Navigate to the **Settings > Integrations > MDM** page and select **Edit** under **Apple Business (AB)**.
3. Select the **Actions** dropdown for the AB token you want to update, and then select **Edit fleets**.
4. Select the default fleet for each platform, and select **Save** to save your selections.

> If no default fleet is set for a host platform (macOS, iOS, or iPadOS), then newly enrolled hosts of that platform will be placed in "Unassigned".

> A host can be transferred to a new (not default) fleet before it enrolls. In the Fleet UI, you can do this under **Settings** > **Fleets**.

## Turn on MDM on a host

Fleet supports manually turning on MDM for macOS hosts that are already enrolled in Fleet.

End users can turn on MDM from their **Fleet Desktop > My device** page.

### Host is in Apple Business (AB)

#### If a macOS host is listed in AB:

1. The end user will see a **Turn on MDM** banner at the top of their **My device** page.
2. Clicking **Turn on MDM** opens a modal with a step-by-step instruction on how to turn on MDM on their host.
3. After completing the steps, the host has MDM features turned on.

### Host isn't in AB

#### If the host isn’t in AB, users can still turn on MDM:

1. On the **My device** page, the end user sees the same **Turn on MDM** banner.
2. Clicking **Turn on MDM** opens a new tab.
   - If [IdP authentication](https://fleetdm.com/guides/setup-experience#require-idp-authentication) is enabled, the end user is prompted to sign in with your organization’s identity provider (IdP).
   - If authentication is successful, or if IdP authentication is disabled, the end user is taken to a page with instructions to download the manual enrollment profile and install it on their macOS host.

## Volume Purchasing Program (VPP)

> Available in Fleet Premium

Connect Fleet to VPP to deploy [Apple App Store apps](https://fleetdm.com/guides/install-app-store-apps) to your hosts.

1. In Fleet, select your avatar on the far right of the main navigation menu, and then **Settings > Integrations > MDM**.
2. Under **Apple Business (AB)**, select **Add VPP** next to **Volume Purchasing Program (VPP)**.
3. Sign in to [Apple Business](https://business.apple.com). If your organization doesn't have an account, select **Sign up now**.
4. Head to **Settings > Apps & Books** and download the content token for the location you want to use. Each token is based on a location in Apple Business.
5. Upload the content token (.vpptoken file) to Fleet.
6. To assign the VPP token to a specific fleet, find the token in the table of VPP tokens. Select the **Actions** dropdown, and then select **Edit fleets**. Use the picker to select which fleet(s) this VPP token should be assigned to.

### Renew VPP:

> Token status is indicated in the **Renew date** column: tokens less than 30 days from expiring will have a yellow indicator, and expired tokens will have a red indicator.

1. Navigate to the **Settings > Integrations > MDM** page
2. Under **Apple Business (AB)**, select **Edit** next to **Volume Purchasing Program (VPP)** and then find the token that you want to renew.
3. Select the **Actions > Renew** for the token.
4. Sign in to [Apple Business](https://business.apple.com).
5. Head to **Settings > Apps & Books** and download your content token.
6. Upload the content token (.vpptoken file) to Fleet.

## Best practice

Most organizations only need one AB token and one VPP token to manage their macOS, iOS, and iPadOS hosts.

These organizations may need multiple AB and VPP tokens:

- Managed Service Providers (MSPs)
- Enterprises that acquire new businesses and as a result inherit new hosts
- Umbrella organizations that preside over entities with separated purchasing authority (i.e. a hospital or university)

For **MSPs**, the best practice is to have one AB and VPP connection per client.

The default fleets for each client's AB token will look like this:
- macOS: 💻 Client A - Workstations
- iOS: 📱🏢 Client A - Company-owned iPhones
- iPadOS:🔳🏢 Client A - Company-owned iPads

Client A's VPP token will be assigned to the above fleets.

For **enterprises that acquire**, the best practice is to add a new AB and VPP connection for each acquisition.

These will be the default fleets:

Enterprise AB token:
- macOS: 💻 Enterprise - Workstations
- iOS: 📱🏢 Enterprise - Company-owned iPhones
- iPadOS:🔳🏢 Enterprise - Company-owned iPads

The enterprises's VPP token will be assigned to the above fleets.

Acquisition AB token:
- macOS: 💻 Acquisition - Workstations
- iOS: 📱🏢 Acquisition - Company-owned iPhones
- iPadOS:🔳🏢 Acquisition - Company-owned iPads

The acquisitions's VPP token will be assigned to the above fleets.

## Simple Certificate Enrollment Protocol (SCEP)

Fleet uses SCEP certificates (1 year expiry) to authenticate the requests hosts make to Fleet. Fleet
renews each host's SCEP certificates automatically every 180 days.

For manually enrolled devices, if SCEP certificate renewal fails, MDM will be turned off on the host. The user will need to re-enroll the device to restore MDM management.

## Troubleshooting failed enrollments

If a host is turned off due to user action or a low battery during the Setup Assistant, it may fail to enroll. This can also happen if your Fleet instance is down for maintenance when a host tries to enroll automatically during the Setup Assistant. In these cases, hosts usually restart after the user attempts to get past the “Welcome to Mac" screen. The best practice in this situation is to wipe the host with Fleet if it has network connectivity or to [reinstall macOS from Recovery](https://support.apple.com/en-us/102655).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-07-02">
<meta name="articleTitle" value="Apple MDM setup">
<meta name="description" value="Learn how to turn on MDM features for Apple hosts in Fleet.">
