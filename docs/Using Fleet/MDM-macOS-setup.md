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

> To prevent abuse, please use your work email. If your email isn't accepted, please make sure it's not on this [list of blocked emails].(https://github.com/fleetdm/fleet/blob/d5df23964b0b52f1d442b66ffe4451dc2a9ef969/website/api/controllers/deliver-apple-csr.js#L60)

Use either of the following methods to generate the necessary files:

#### Fleet UI

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under **Apple Push Certificates Portal**, select **Request**, then fill out the form. This should generate three files and send an email to you with an attached CSR file.

#### Fleetctl CLI

Run the following command to download three files and send an email to you with an attached CSR file.

```sh
fleetctl generate mdm-apple --email <email> --org <org> 
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

## Renewing APNs 

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

### Step 1: generate the required files
- A new APNs certificate. 
- A new APNs private key.

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

New or wiped macOS hosts that are in ABM, before they've been set up, appear in Fleet with **MDM status** set to "Pending".

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

```sh
fleetctl get mdm-apple
```

### Step 5: set Fleet to be the MDM server for Macs in ABM
Set Fleet to be the MDM for all future Macs purchased via Apple or an authorized reseller: 

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment** and click **Edit** next to **Default Server Assignment**.
5. Switch **Mac** to Fleet.

### Step 6: set the default team for hosts enrolled via ABM

All hosts that automatically enroll will be assigned to the default team. If no default team is set, then the host will be placed in "No team". 

> A host can be transferred to a new (not default) team before it enrolls. In the Fleet UI, you can do this under **Settings** > **Teams**.

Use either of the following methods to change the default team:

#### Fleet UI

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

2. In the Apple Business Manager section, select the **Edit team** button next to **Default team**.

3. Choose a team and select **Save**.

#### Fleetctl CLI

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

2. Set the `mdm.apple_bm_default_team` configuration option to the desired team's name.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

## Renewing ABM

> Apple expires ABM server tokens certificates once every year or whenever the account that downloaded the token has their password changed. 

Use either of the following methods to see your ABM renewal date and other important information:

#### Fleet UI

1. Navigate to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Look at the **Apple Business Manager** section.

#### Fleetctl CLI

```sh
fleetctl get mdm-apple
```

If you have configured Fleet with an Apple Business Manager server token for mobile device management (a Fleet Premium feature), you will eventually need to renew that token. [As documented in the Apple Business Manager User Guide](https://support.apple.com/en-ca/guide/apple-business-manager/axme0f8659ec/web), the token expires after a year or whenever the account that downloaded the token has their password changed.

To renew the token: 
1. Log in to [business.apple.com](https://business.apple.com)
2. Select Fleet's MDM server record
3. Download a new token for that server record
4. In your Fleet server, update the environment variable [FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-server-token-bytes)
5. Restart the Fleet server

<meta name="pageOrderInSection" value="1500">
<meta name="title" value="macOS setup">
<meta name="description" value="Learn how to configure Fleet to use Apple's Push Notification service and connect to Apple Business Manager.">
<meta name="navSection" value="Device management">
