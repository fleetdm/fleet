# Windows MDM setup

![Windows MDM setup](../website/assets/images/articles/windows-mdm-fleet-1600x900@2x.png)

## Overview

> Windows MDM features are not ready for production and are currently in development. These features are disabled by default.

Turning on Windows MDM features requires configuring Fleet with a certificate and key. This guide will walk you through how to upload these to Fleet and turn on Windows MDM.

Automatic enrollment allows Windows workstations to automatically enroll to Fleet when they are first set up. Automatic enrollment requires Microsoft Azure Active Directory (aka Microsoft Entra). This guide will walk you through how to connect Azure AD to Fleet. 

With Fleet connected to Azure AD, the end user will see Microsoft's default setup experience. You can further customize the initial setup with Windows Autopilot, which is similar to Apple's Automated Device Enrollment (DEP). Autopilot requires a Microsoft Intune license. This guide will also walk you through how to customize the intitial setup with Autopilot.

## Requirements
To use Fleet's Windows MDM features you need to have:
- A [deployed Fleet instance](https://fleetdm.com/docs/deploy/introduction).
- A Fleet user with the admin role.

## Turning on Windows MDM

Fleet uses a certificate and key pair to authenticate and manage interactions between Fleet and Windows host.

This section will show you how to:
1. Generate your certificate and key
2. Configure Fleet with your certificate and key
3. Turn on Windows MDM in Fleet

### Step 1: generate your certificate and key

If you're already using Fleet's macOS MDM features, you already have a certificate and key. These are your SCEP certificate and SCEP private key you used when turning on macOS MDM.

If you're not using macOS MDM features, run the following command to download three files and send an email to you with an attached CSR file.

```
fleetctl generate mdm-apple --email <email> --org <org> 
```

Save the SCEP certificate and SCEP key. These are your certificate and key. You can ignore the downloaded APNs key and the APNs CSR that was sent to your email.

### Step 2: configure Fleet with your certificate and key

1. In your Fleet server configuration, set the contents of the certificate and key in the following environment variables:

> Note: Any environment variable that ends in `_BYTES` expects the file's actual content to be passed in, not a path to the file. If you want to pass in a file path, remove the `_BYTES` suffix from the environment variable.

- [FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-windows-wstep-identity-cert-bytes)
- [FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-windows-wstep-identity-key-bytes)

2. Set the `FLEET_DEV_MDM_ENABLED` environment variable to `1`.

3. Restart the Fleet server.

### Step 3: Turn on Windows MDM in Fleet

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM) enrollment** page.

2. Next to **Turn on Windows MDM** select **Turn on** to navigate to the **Turn on Windows MDM** page.

3. Select **Turn on**.

fleetctl CLI:

1. Create `fleet-config.yaml` file or add to your existing `config` YAML file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    windows_enabled_and_configured: true
  ...
```

2. Run the fleetctl `apply -f fleet-config.yml` command to turn on Windows MDM.

3. Confirm that Windows MDM is turned on by running `fleetctl get config`.

## Microsoft Azure Active Directory (AD)

> Available in Fleet Premium or Ultimate

By connecting Fleet to Azure AD, Windows workstations can automatically enroll to Fleet when they’re first unboxed and set up by your end user.

This section will guide you through how to:

1. Connect Fleet to Azure AD

2. Test automatic enrollment

### Step 1: connect Fleet to Azure AD

For instructions on how to connect Fleet to Azure AD, in the Fleet UI, select the avatar on the right side of the top navigation and select **Settings > Integrations > Automatic enrollment**. Then, next to **Windows automatic enrollment** select **Details**.

### Step 2: test automatic enrollment

Testing automatic enrollment requires creating a test user in Azure AD and a freshly wiped or new Windows workstation.

1. Sign in to [Azure portal](https://portal.azure.com).

2. At the top of the page search "Users" and select **Users**.

3. Select **+ New user > Create new user**, fill out the details for your test user, and select **Review + Create > Create**.

4. Go back to **Users** and refresh the page to confirm that your test user was created.

5. Open your Windows workstation and follow the setup steps. When you reach the **How would you like to set up?** screen, select **Set up for an organization**. If your workstations has Windows 11, select **Set up for work or school**.

6. Sign in with your test user's credentials and finish the setup steps.

7. When you reach the desktop on your Windows workstation, confirm that your workstation was automatically enrolled to Fleet by selecting the carrot (^) in your taskbar and then selecting the Fleet icon. This will navigate you to this workstation's **My device** page.

8. On the **My device** page, below **My device** confirm that your workstation has a **Status** of "Online."

## Window Autopilot

> Available in Fleet Premium or Ultimate

After you connect Fleet to Azure AD, you can customize the Windows setup experience with [Windows Autopilot](https://learn.microsoft.com/en-us/autopilot/windows-autopilot).

This section will guide you through how to:

1. Buy a Microsoft Intune license. Microsoft requires this for Autopilot.

2. Create an Autopilot profile in Intune

3. Register a test workstation with Autopilot

4. Upload your organization's logo that end users will see during setup

5. Test Autopilot

### Step 1: buy a Microsoft Intune license

Autopilot requires at least one Intune license to edit the Autopilot profile.

1. Sign in to [Microsoft 365 admin center](https://admin.microsoft.com/).

2. In the left-side bar select **Marketplace**.

3. On the **Marketplace** page, select **All products** and in the search bar below **All products** enter "Intune".

4. Find **Microsoft Intune Plan 1 Device** and select **Details**

5. On the **Microsoft Intune Plan 1 Device** page, select **Buy** and follow instructions to purchase the license. 

6. Sign in to [Azure portal](https://portal.azure.com).

7. At the top of the page search "Users" and select **Users**.

8. Select or create your Intune admin user and select **Licenses**.

9. Select **+ Assignments** and assign the **Microsoft Intune Plan 1 Device** to this user.

### Step 2: create an Autopilot profile

1. Sign in to [Microsoft Intune](https://endpoint.microsoft.com/) using the Intune admin user from step 1.

2. In the left-side bar select **Devices > Enroll devices**. Under **Windows Autopilot Deployment Program** select **Deployment Profiles** to navigate to the **Windows Autopilot deployment profiles** page.

3. Select **+ Create profile > Windows PC** and follow steps to create an Autopilot profile. On the **Assignments** step, select **+ Add all devices**.

### Step 3: register a test workstation

1. Open your test workstation and follow these [Microsoft instructions](https://learn.microsoft.com/en-us/autopilot/add-devices#desktop-hash-export) to export your workstations's device hash as a CSV. The CSV should look something like `DeviceHash_DESKTOP-2V08FUI.csv`

2. In Intune, in the left-side bar, select **Devices > Enroll devices**. Under **Windows Autopilot Deployment Program** select **Devices** to navigate to the **Windows Autopilot devices** page.

3. Select **Import** and import your CSV.

4. After Intune finishes the import, refresh the **Windows Autopilot devices** page several times to confirm that your workstation is registered with Autopilot.

### Step 4: upload your organization's logo

1. Navigate to [Azure portal](https://portal.azure.com).

2. At the top of the page, search for "Microsoft Entra ID", select **Microsoft Entra ID**, and then select **Company branding**.

3. On the **Company Branding** page, select **Configure** or **Edit** under **Default sign-in experience**.

4. Select the **Sign-in form** tab and upload your logo to the **Square logo (light theme)** and **Square logo (dark theme)** fields.

5. In the bottom bar, select **Review + Save** and then **Save**.

### Step 5: test Autopilot

1. Wipe your test workstation.

2. After it's been wiped, open your workstation and follow the setup steps. At screen in which you're asked to sign in, you should see the title "Welcome to [your organziation]!" next to the logo you uploaded in step 4.


<meta name="articleTitle" value="Windows MDM setup">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2023-10-23">
<meta name="articleImageUrl" value="../website/assets/images/articles/windows-mdm-fleet-1600x900@2x.png">
<meta name="description" value="Configuring Windows MDM in Fleet.">
