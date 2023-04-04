# Warning
MDM features are not ready for production and are currently in development. These features are disabled by default.

# Supported macOS versions

In Fleet, MDM features are supported for Macs running macOS 12 (Monterey) and higher.

# Overview

MDM features require Apple's Push Notification service (APNs) to control and secure Apple devices. This guide will walk you through how to generate and upload a valid APNs certificate to Fleet in order to use Fleet's MDM features.

[Automated Device Enrollment](https://support.apple.com/en-us/HT204142) allows Macs to automatically enroll to Fleet when they are first set up. This guide will walk you through how to connect Apple Business Manager (ABM) to Fleet. Note that this is only required if you are using Automated Device Enrollment AKA Device Enrollment Program (DEP) AKA "Zero-touch."

# Requirements
To use Fleet's MDM features you must have:
1. A [deployed Fleet instance](../Deploying/Introduction.md)
2. A Fleet user with the admin role

## Apple Push Notification service (APNs)
Apple uses APNs to authenticate and manage interactions between Fleet and the host. 

To connect Fleet to APNs, we will do the following steps:
1. Generate four required files
2. Generate an APNs certificate from Apple Push Certificates Portal
3. Configure Fleet with the required files

### Step 1: generate required files
For the MDM protocol to function, we need to generate the four following files:
1. APNs certificate 
2. APNs private key 
3. Simple Certificate Enrollment Protocol (SCEP) certificate 
4. SCEP private key

The APNs certificates serves as authentication between Fleet and Apple, while the SCEP certificates serve as authentication between Fleet and hosts.

To do this, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under **Apple Push Certificates Portal**, select **Request**, then fill out the form. This should generate three files and send an email to you with an attached CSR file.

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple --email <email> --org <org>`. This should download three files and send an email to you with an attached CSR file.

### Step 2: generate an APNs certificate from Apple Push Certificates Portal

1. Log in to or enroll in [Apple Push Certificates Portal](https://identity.apple.com).
2. Select **Create a Certificate**
3. Upload your CSR and input a friendly name, such as "Fleet MDM."
4. Download the APNs certificate

> Take note of the Apple ID you use to sign into Apple Push Certificates Portal. You'll need to use the same Apple ID when renewing your APNs certificate.

### Step 3: configure Fleet with the required files

With the four generated files, we now give them to the Fleet server. 

Restart the Fleet server with the contents of the APNs certificate, APNs private key, SCEP certificate, and SCEP private key in following environment variables:
* [FLEET_MDM_APPLE_APNS_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-cert-bytes)
* [FLEET_MDM_APPLE_APNS_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-apns-key-bytes)
* [FLEET_MDM_APPLE_SCEP_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-scep-cert-bytes)
* [FLEET_MDM_APPLE_SCEP_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-scep-key-bytes)

> You do not need to provide the APNs CSR which was emailed to you. 

Confirm that Fleet is set up by visiting the "Fleet UI" or using "fleetctl."

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Look at the **Apple Push Certificates Portal** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

You should see information about the APNs certificate such as serial number and renewal date. 

## Renewing APNs

> Apple requires that APNs certificates are renewed once every year. 
> * Be sure to do it early. If you renew after a certificate has expired, you will have to turn MDM off and back on for all macOS hosts. 
> * Be sure to use the same Apple ID from year-to-year. If you don't, you will have to unenroll and re-enroll all macOS hosts. 

You can see the certificate's renewal date and other important APNs information using the Fleet UI or the `fleetctl` command-line interface:

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Look at the **Apple Push Certificates Portal** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

2. Look at the on-screen information.

How to renew the certificate if it's expired or about to expire:

1. Run the `fleetctl generate mdm-apple --email <email> --org <org>` command. 

2. Sign in to [Apple Push Certificates Portal](https://identity.apple.com) using the same Apple ID you used to get your original certificate. 

3. In the **Settings > Integrations > Mobile device management (MDM)** page, under Apple Push Certificates portal, find the serial number of your current certificate. In Apple Push Certificates Portal, click  **Renew** next to the certificate that has the matching serial number. 

### Renewing SCEP
TODO

## Apple Business Manager (ABM)

_Available in Fleet Premium_

When purchased through Apple or an authorized reseller, Macs can automatically enroll to Fleet when they’re first unboxed and set up by your end user. To do this, you must connect Fleet to Apple Business Manager (ABM).

To connect Fleet to ABM, we will do the following steps:
1. Generate certificate and private key for ABM
2. Create a new MDM server record for Fleet in ABM
3. Download the MDM server token from ABM
4. Upload the server token, certificate, and private key to the Fleet server
5. Set the new MDM server as the auto-enrollment server for Macs in ABM

### Step 1: generate required certificate and private key

First we will generate a certificate/key pair. This pair is how Fleet authenticates itself to ABM.

To get the two files, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page.
2. Under **Apple Business Manager**, click the "Download" button

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple-bm`.

### Step 2: create a new MDM server in ABM

Next we create an MDM server record in ABM which represents Fleet. How to create a new MDM server in ABM:

1. Log in to or enroll in [ABM](https://business.apple.com) 
2. Click your name at the bottom left of the screen
3. Click **Preferences** 
4. Click **MDM Server Assignment**
5. Click the **Add** button at the top 
6. Enter a name for the server such as "Fleet MDM"
7. Upload the certificate generated in Step 1

### Step 3: download the server token 
1. In the details page of the newly created server, click **Download Token** at the top. You should receive a `.p7m` file.

### Step 4: upload server token, certificate, and private key to Fleet
With the three generated files, we now give them to the Fleet server so that it can authenticate itself to ABM. 

Restart the Fleet server with the contents of the server token, certificate, and private key in following environment variables:
* [FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-server-token-bytes)
* [FLEET_MDM_APPLE_BM_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-cert-bytes)
* [FLEET_MDM_APPLE_BM_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-apple-bm-key-bytes)

Confirm that Fleet is set up by visitng the "Fleet UI" or using "fleetctl."

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Look at the **Apple Business Manager** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

You should see information about the ABM server token such as organization name and renewal date. 

### Step 5: set Fleet to be the MDM server for Macs in ABM
Finally, we set Fleet to be the MDM for all future Macs purchased via Apple or an authorized reseller. 

1. Log in to [Apple Business Manager](business.apple.com)
2. Click your profile icon in the bottom left
3. Click **Preferences**
4. Click **MDM Server Assignment**
5. Switch Macs to the new Fleet instance.

### Step 6 (optional): set the default team for hosts enrolled via ABM

All automatically-enrolled hosts will be assigned to a default team of your choosing after they are unboxed and set up. The host will receive the configurations and behaviors set for that team. If no default team is set, then the host will be placed in "No Teams". 

> A host can be transferred to a new (not default) team before it enrolls. Learn how [here](./Teams.md#transfer-hosts-to-a-team). Transferring a host will automatically enforces the new team's settings when it enrolls.

To change the default team, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page.

2. In the Apple Business Manager section, select the **Edit team** button next to **Default team**.

3. Choose a team and select **Save**.

`fleetctl` CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

2. Set the `mdm.apple_bm_default_team` configuration option to the desired team's name.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

### Pending hosts
Some time after you purchase a Mac through Apple or an authorized reseller, but before it has been set up, the Mac will appear in ABM as in transit. When the Mac appears in ABM, it will also appear in Fleet with **MDM status** set to "Pending." After the new host is set up, the **MDM Status** will change to "On" and the host will be assigned to the default team.

## Renewing ABM

> Apple expires ABM server tokens certificates once every year or whenever the account that downloaded the token has their password changed. 

You can see the renewal date and other important ABM information using the Fleet UI or the `fleetctl` command-line interface:

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page.

2. Look at the **Apple Business Manager** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

To renew the token, follow the [instructions documented in this FAQ](https://fleetdm.com/docs/using-fleet/faq#how-can-i-renew-my-apple-business-manager-server-token).

<meta name="pageOrderInSection" value="1500">
<meta name="title" value="MDM setup">
