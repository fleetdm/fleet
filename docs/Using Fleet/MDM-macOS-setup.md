# macOS setup

## Overview

MDM features require Apple's Push Notification service (APNs) to control and secure Apple devices. This guide will walk you through how to generate and upload a valid APNs certificate to Fleet in order to use Fleet's MDM features.

[Automated Device Enrollment](https://support.apple.com/en-us/HT204142) allows Macs to automatically enroll to Fleet when they are first set up. This guide will also walk you through how to connect Apple Business Manager (ABM) to Fleet. 

> **Note:** you are only required to connect Apple Business Manager (ABM) to Fleet if you are using Automated Device Enrollment AKA Device Enrollment Program (DEP) AKA "Zero-touch."

## Requirements
To use Fleet's MDM features you need to have:
- A [deployed Fleet instance](../Deploying/Introduction.md).
- A Fleet user with the admin role.

## Apple Push Notification service (APNs)
Apple uses APNs to authenticate and manage interactions between Fleet and the host.

This section will show you how to:
1. Generate the files to connect Fleet to APNs.
2. Generate an APNs certificate from Apple Push Certificates Portal.
3. Configure Fleet with the required files.

### Step 1: generate the required files
For the MDM protocol to function, we need to generate the four following files:
- APNs certificate 
- APNs private key 
- Simple Certificate Enrollment Protocol (SCEP) certificate 
- SCEP private key

The APNs certificates serve as authentication between Fleet and Apple, while the SCEP certificates serve as authentication between Fleet and hosts.

Use either of the following methods to generate the necessary files:

#### Fleet UI

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under **Apple Push Certificates Portal**, select **Request**, then fill out the form. This should generate three files and send an email to you with an attached CSR file.

#### Fleetctl CLI

Run the following command to download three files and send an email to you with an attached CSR file.

2. Under **End user authentication**, enter your IdP credentials and select **Save**.

> If you've already configured [single sign-on (SSO) for logging in to Fleet](https://fleetdm.com/docs/configuration/fleet-server-configuration#okta-idp-configuration), you'll need to create a separate app in your IdP so your end users can't log in to Fleet. In this separate app, use "https://fleetserver.com/api/v1/fleet/mdm/sso/callback" for the SSO URL.

fleetctl CLI:

1. Create `fleet-config.yaml` file or add to your existing `config` YAML file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    end_user_authentication:
      identity_provider_name: "Okta"
      entity_id: "https://fleetserver.com"
      issuer_url: "https://okta-instance.okta.com/84598y345hjdsshsfg/sso/saml/metadata"
      metadata_url: "https://okta-instance.okta.com/84598y345hjdsshsfg/sso/saml/metadata"
  ...
```

### Step 2: generate an APNs certificate
1. Log in to or enroll in [Apple Push Certificates Portal](https://identity.apple.com).
2. Select **Create a Certificate**.
3. Upload your CSR and input a friendly name, such as "Fleet."
4. Download the APNs certificate.

> **Important:** Take note of the Apple ID you use to sign into Apple Push Certificates Portal. You'll need to use the same Apple ID when renewing your APNs certificate.

### Step 3: configure Fleet with the generated files
Restart the Fleet server with the contents of the APNs certificate, APNs private key, SCEP certificate, and SCEP private key in the following environment variables:

> Note: Any environment variable that ends in `_BYTES` expects the file's actual content to be passed in, not a path to the file. If you want to pass in a file path, remove the `_BYTES` suffix from the environment variable.

* [FLEET_MDM_APPLE_APNS_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-cert-bytes)
* [FLEET_MDM_APPLE_APNS_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-key-bytes)
* [FLEET_MDM_APPLE_SCEP_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-scep-cert-bytes)
* [FLEET_MDM_APPLE_SCEP_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-scep-key-bytes)
* [FLEET_MDM_APPLE_SCEP_CHALLENGE](https://fleetdm.com/docs/deploying/configuration#mdm-apple-scep-challenge)

> You do not need to provide the APNs CSR which was emailed to you. 

### Step 4: confirm that Fleet is set up correctly 

Use either of the following methods to confirm that Fleet is set up. You should see information about the APNs certificate such as serial number and renewal date.

#### Fleet UI

Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

#### Fleetctl CLI

```
fleetctl get mdm-apple
```

Learn more about "No team" configuration options [here](./configuration-files/README.md#organization-settings).

> **Important:** Apple requires that APNs certificates are renewed annually. 
> - If your certificate expires, you will have to turn MDM off and back on for all macOS hosts.
> - Be sure to use the same Apple ID from year-to-year. If you don't, you will have to turn MDM off and back on for all macOS hosts.

This section will guide you through how to:
1. Generate the files required to renew your APNs certificate.
2. Renew your APNs certificate in Apple Push Certificates Portal.
3. Configure Fleet with the required files.
4. Confirm that Fleet is set up correctly.

Use either of the following methods to see your APNs certificate's renewal date and other important information:

#### Fleet UI

Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

#### Fleetctl CLI

```sh
fleetctl get mdm-apple
``` 

If your package is a distribution package should see a `Distribution` file.

Run the following command in `fleetctl`. This will download three files and send an email to you with an attached CSR file. You may ignore the SCEP certificate and SCEP key as you do not need these to renew APNs.

```sh
fleetctl generate mdm-apple --email <email> --org <org>
```

### Step 2: renew APNs certificate

1. Log in to or enroll in [Apple Push Certificates Portal](https://identity.apple.com) using the same Apple ID you used to get your original APNs certificate.
2. Click **Renew** next to the expired certificate. 
3. Upload your CSR.
4. Download the new APNs certificate.

### Step 3: configure Fleet with the generated files
Restart the Fleet server with the contents of the APNs certificate and APNs private key in following environment variables:
* [FLEET_MDM_APPLE_APNS_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-cert-bytes)
* [FLEET_MDM_APPLE_APNS_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-key-bytes)

> You do not need to provide the APNs CSR which was emailed to you.

### Step 4: confirm that Fleet is set up correctly

Use either of the following methods to confirm that Fleet is set up:

#### Fleet UI:

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Follow the on-screen instructions in the **Apple Push Certificates Portal** section.

#### Fleetctl CLI:

Run the following command. You should see information about the new APNs certificate such as serial number and renewal date. 

```sh
fleetctl get mdm-apple
```

## Renewing SCEP
The SCEP certificates generated by Fleet and uploaded to the environment variables expire every 10 years. To renew them, regenerate the keys and update the relevant environment variables.

## Apple Business Manager (ABM)

> Available in Fleet Premium

By connecting Fleet to ABM, Macs purchased through Apple or an authorized reseller can automatically enroll to Fleet when they’re first unboxed and set up by your end user.

This section will guide you through how to:

1. Generate certificate and private key for ABM
2. Create a new MDM server record for Fleet in ABM
3. Download the MDM server token from ABM
4. Upload the server token, certificate, and private key to the Fleet server
5. Set the new MDM server as the auto-enrollment server for Macs in ABM

### Step 1: generate the required certificate and private key

User either of the following methods to generate a certificate and private key pair. This pair is how Fleet authenticates itself to ABM:

#### Fleet UI:

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under **Apple Business Manager**, click the "Download" button

#### Fleetctl CLI:

```sh
fleetctl generate mdm-apple-bm
```

### Step 2: create a new MDM server in ABM

Create an MDM server record in ABM which represents Fleet:

1. Log in to or enroll in [ABM](https://business.apple.com) 
2. Click your name at the bottom left of the screen
3. Click **Preferences** 
4. Click **MDM Server Assignment**
5. Click the **Add** button at the top 
6. Enter a name for the server such as "Fleet"
7. Upload the certificate generated in Step 1

### Step 3: download the server token 
In the details page of the newly created server, click **Download Token** at the top. You should receive a `.p7m` file.

### Step 4: upload server token, certificate, and private key to Fleet
With the three generated files, we now give them to the Fleet server so that it can authenticate itself to ABM. 

Restart the Fleet server with the contents of the server token, certificate, and private key in following environment variables:
* [FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-server-token-bytes)
* [FLEET_MDM_APPLE_BM_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-cert-bytes)
* [FLEET_MDM_APPLE_BM_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-key-bytes)

### Step 3: confirm that Fleet is set up correctly

Use either of the following methods to confirm that Fleet is set up correctly. You should see information about the ABM server token such as organization name and renewal date. 

#### Fleet UI:

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Navigate to the **Apple Business Manager** section.

#### Fleetctl CLI:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_setup:
      bootstrap_package: https://github.com/organinzation/repository/bootstrap-package.pkg
  ...
```

### Step 5: set Fleet to be the MDM server for Macs in ABM
Set Fleet to be the MDM for all future Macs purchased via Apple or an authorized reseller: 

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment**
5. Switch Macs to the new Fleet instance.

### Step 6 (optional): set the default team for hosts enrolled via ABM

All automatically-enrolled hosts will be assigned to a default team of your choosing after they are unboxed and set up. The host will receive the configurations and behaviors set for that team. If no default team is set, then the host will be placed in "No Teams". 

> A host can be transferred to a new (not default) team before it enrolls. Learn how [here](./Teams.md#transfer-hosts-to-a-team). Transferring a host will automatically enforce the new team's settings when it enrolls.

Use either of the following methods to change the default team:

#### Fleet UI

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

2. In the Apple Business Manager section, select the **Edit team** button next to **Default team**.

3. Choose a team and select **Save**.

#### Fleetctl CLI

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

2. Set the `mdm.apple_bm_default_team` configuration option to the desired team's name.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

### Pending hosts 
Some time after you purchase a Mac through Apple or an authorized reseller, but before it has been set up, the Mac will appear in ABM as in transit. When the Mac appears in ABM, it will also appear in Fleet with **MDM status** set to "Pending." After the new host is set up, the **MDM Status** will change to "On" and the host will be assigned to the default team.

## Renewing ABM

> Apple expires ABM server tokens certificates once every year or whenever the account that downloaded the token has their password changed. 

Use either of the following methods to see your ABM renewal date and other important information:

#### Fleet UI

In this example, let's assume you have a "Workstations" team as your [default team](./MDM-setup.md#step-6-optional-set-the-default-team-for-hosts-enrolled-via-abm) in Fleet and you want to test your profile before it's used in production. 

To do this, we'll create a new "Workstations (canary)" team and add the automatic enrollment profile to it. Only hosts that automatically enroll to this team will see the custom macOS Setup Assistant.

#### Fleetctl CLI

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      macos_setup:
        macos_setup_assistant: ./path/to/automatic_enrollment_profile.json
    ...
```

Learn more about team configurations options [here](./configuration-files/README.md#teams).

If you want to customize the macOS Setup Assistant for hosts that automatically enroll to "No team," we'll need to create a `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_setup:
      macos_setup_assistant: ./path/to/automatic_enrollment_profile.json
  ...
```

Learn more about configuration options for hosts that aren't assigned to a team [here](./configuration-files/README.md#organization-settings).

3. Add an `mdm.macos_setup.macos_setup_assistant` key to your YAML document. This key accepts a path to your automatic enrollment profile.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to upload the automatic enrollment profile to Fleet.

### Step 3: test the custom macOS Setup Assistant

Testing requires a test Mac that is present in your Apple Business Manager (ABM) account. We will wipe this Mac and use it to test the custom macOS Setup Assistant.

1. Wipe the test Mac by selecting the Apple icon in top left corner of the screen, selecting **System Settings** or **System Preference**, and searching for "Erase all content and settings." Select **Erase All Content and Settings**.

2. In Fleet, navigate to the Hosts page and find your Mac. Make sure that the host's **MDM status** is set to "Pending."

> New Macs purchased through Apple Business Manager appear in Fleet with MDM status set to "Pending." Learn more about these hosts [here](./MDM-setup.md#pending-hosts).

3. Transfer this host to the "Workstations (canary)" team by selecting the checkbox to the left of the host and selecting **Transfer** at the top of the table. In the modal, choose the Workstations (canary) team and select **Transfer**.

4. Boot up your test Mac and complete the custom out-of-the-box setup experience.

<meta name="pageOrderInSection" value="1505">
<meta name="title" value="MDM macOS setup">
<meta name="description" value="Customize your macOS setup experience with Fleet Premium by managing user authentication, Setup Assistant panes, and installing bootstrap packages.">
<meta name="navSection" value="Device management">
