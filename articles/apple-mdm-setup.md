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

Connect Fleet to your ABM to allow automatic enrollment for company-owned and [Account-driven User Enrollment](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account) for personal (BYOD) macOS, iOS, and iPadOS hosts.

### Re-enrolling AB hosts

When an AB host re-enrolls in Fleet (e.g., after a wipe or OS reinstall), Fleet automatically:
  - Cancels pending MDM commands, script runs, and software installs
  - Clears completed commands, scripts, and software from the previous enrollment
  - Resets host labels

This means you **do not need to delete** an ABM host from Fleet before 
re-enrolling it. Fleet handles clearing stale state automatically.

> This automatic state clearing does not apply to hosts undergoing ABM MDM migration. During migration, the host's existing state (labels, pending activity) is preserved to ensure a seamless transition from your previous MDM solution.

### To connect Fleet to ABM, you have to add an ABM token to Fleet. To add an ABM token:

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

### Default automatic enrollment profile

When macOS, iOS, or iPadOS hosts automatically enroll through Apple Business, Fleet sends an automatic enrollment (ADE) profile to Apple that controls how the Setup Assistant behaves. If no custom profile is uploaded for a fleet, Fleet uses a built-in default profile.

The default profile sets options such as whether enrollment is mandatory, which Setup Assistant panes are skipped, and whether the MDM profile is removable. See the [Setup Assistant pane options](https://fleetdm.com/learn-more-about/apple-setup-assistant).

#### Where to view the default profile

- **Fleet UI:** Navigate to **Controls > Setup experience > Setup Assistant**. When no custom profile is uploaded, you can select **Download** to download the default profile JSON that your Fleet instance is currently using.
- **API:** `GET /api/v1/fleet/enrollment_profiles/automatic/default`

#### Stored once, never auto-refreshed

The default profile is stored once per Fleet instance — at the time of your first automatic enrollment registration with Apple — and is **not** refreshed by Fleet upgrades, by adding or removing AB tokens, or by any other normal operation. This means that even if a newer version of Fleet ships updated default values, existing Fleet instances will continue using the default profile that was originally stored.

#### Updating to Fleet's latest defaults

There is no in-product "reset to latest default" action today. If you want your Fleet instance to use newer default values introduced in a later Fleet release:

1. Check the latest defaults by reviewing the [REST API documentation](https://fleetdm.com/docs/rest-api/rest-api#get-fleet-default-mdm-setup-enrollment-profile) or by checking a freshly created Fleet instance.
2. Create a custom enrollment profile JSON containing the desired values. See the [Setup Assistant section of the setup experience guide](https://fleetdm.com/guides/setup-experience#setup-assistant) for instructions on creating and uploading a custom profile.
3. Upload it via the Fleet UI (**Controls > Setup experience > Setup Assistant > Add profile**) or the [API](https://fleetdm.com/docs/rest-api/rest-api#update-custom-mdm-setup-enrollment-profile).

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
   - If [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication) is enabled, the end user is prompted to sign in with your organization’s identity provider (IdP).
   - If authentication is successful, or if end user authentication is disabled, the end user is taken to a page with instructions to download the manual enrollment profile and install it on their macOS host.

## Volume Purchasing Program (VPP)

> Available in Fleet Premium

Connect Fleet to VPP to deploy [Apple App Store apps](https://fleetdm.com/guides/install-app-store-apps) to your hosts.

1. In Fleet, select your avatar on the far right of the main navigation menu, and then **Settings > Integrations > MDM**.
2. Under **Apple Business (AB)**, select **Add VPP** next to **Volume Purchasing Program (VPP)**.
3. Sign in to [Apple Business](https://business.apple.com). If your organization doesn't have an account, select **Sign up now**.
4. Head to **Settings > Apps & Books** and download the content token for the organization unit you want to use. Each token is based on an organization unit in Apple Business.
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

- [Managed Service Providers (MSPs)](#msps)
- [Enterprises that acquire](#enterprises-that-acquire) new businesses and as a result inherit new hosts
- [Umbrella organizations](#umbrella-organizations) that preside over entities with separated purchasing authority (i.e. a hospital or university)
- [International organizations](#international-organizations) that manage hosts across multiple countries

### MSPs

For MSPs, the best practice is to have one AB and VPP connection per client.

The default fleets for each client's AB token will look like this:
- macOS: 💻 Client A - Workstations
- iOS: 📱🏢 Client A - Company-owned iPhones
- iPadOS:🔳🏢 Client A - Company-owned iPads

Client A's VPP token will be assigned to the above fleets.

### Enterprises that acquire

For enterprises that acquire, the best practice is to add a new AB and VPP connection for each acquisition.

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

### Umbrella organizations

For umbrella organizations (e.g., a hospital system or university) where each entity has its own purchasing authority, the best practice is to have one AB and VPP connection per entity.

The default fleets for each entity's AB token will look like this:
- macOS: 💻 Entity A - Workstations
- iOS: 📱🏢 Entity A - Company-owned iPhones
- iPadOS: 🔳🏢 Entity A - Company-owned iPads

Entity A's VPP token will be assigned to the above fleets.

### International organizations

> Support for Apple App Store (VPP) apps from non-US stores is [coming soon](https://github.com/fleetdm/fleet/issues/43846).

For international organizations that manage hosts across multiple countries, the best practice is to have one AB and VPP connection per country. Apple Business and VPP tokens are tied to a specific country or region.

The default fleets for each country's AB token will look like this:
- macOS: 💻 Country A - Workstations
- iOS: 📱🏢 Country A - Company-owned iPhones
- iPadOS: 🔳🏢 Country A - Company-owned iPads

Each country's VPP token will be assigned to the above fleets.

## Simple Certificate Enrollment Protocol (SCEP)

Fleet uses SCEP certificates (1 year expiry) to authenticate the requests hosts make to Fleet. Fleet
renews each host's SCEP certificates automatically every 180 days.

For manually enrolled devices, if SCEP certificate renewal fails, MDM will be turned off on the host. The user will need to re-enroll the device to restore MDM management.

## Troubleshooting

### Failed enrollments

If a host is restarted/shut down during macOS Setup Assistant, it will fail to enroll to Fleet. Failed enrollments also happen if Fleet instance is down for an upgrade. When this happens, sometimes hosts automatically restart setup. If that doesn't happen, the best practice is to remotely [wipe the host](https://fleetdm.com/guides/lock-wipe-hosts#wipe-a-host) if the host is connected to Wi-Fi. If it's not, you'll need physical access to [reinstall macOS from Recovery](https://support.apple.com/en-us/102655).

### Apple Business (AB)

Fleet surfaces AB (formerly Apple Business Manager) automatic enrollment profile assignment by retrieving assignment errors and timestamps for each host. While Fleet does not actively monitor push events, admins can view assignment and push timestamps in host details. If a device shows an assignment time but no push time, admins can infer the push did not occur and may need to restart the device or run `sudo profiles renew -type enrollment` for remediation. Error details and timestamps are available for targeted troubleshooting. Customers may need to contact Apple support if an online host never has a push time. 

![Fleet-AB-workflow](https://github.com/fleetdm/fleet/blob/main/website/assets/images/articles/abm-assignment-workflow.jpg)

To view an AB issue:

1. If there is an active issue assigning a profile, a vital called **AB issue** will be on the **Dashboard** page. This will take you to a filtered list of hosts with AB issues.

2. Select a host and click on the MDM status to view details.


> For AB hosts, you do not need to delete the host from Fleet before re-enrolling. Fleet automatically clears pending and completed commands, scripts, software installs, and labels when the host re-enrolls. See [Re-enrolling AB hosts](#re-enrolling-ab-hosts).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="zhumo">
<meta name="authorFullName" value="Mo Zhu">
<meta name="publishedOn" value="2024-07-02">
<meta name="articleTitle" value="Apple MDM setup">
<meta name="description" value="Learn how to turn on MDM features for Apple hosts in Fleet.">
