# Warning
MDM features are not ready for production and are currently in development. These features are disabled by default.

# Supported macOS versions

In Fleet, MDM features are supported for Macs running macOS 12 (Monterey) and higher.

# Overview

To use Fleet's MDM features you first first have to [deploy Fleet](../Deploying/Introduction.md) and [add your hosts to Fleet](./Adding-hosts.md).

MDM features require Apple's Push Notification service (APNs) to control and secure Apple devices. This guide will walk you through how to generate and upload a valid APNs certificate to Fleet in order to use Fleet's MDM features.

[Automated Device Enrollment](https://support.apple.com/en-us/HT204142) allows Macs to automatically enroll to Fleet when they are first set up. This guide will walk you through how to connect Apple Business Manager (ABM) to Fleet. Note that this is only required if you are using Automated Device Enrollment AKA Device Enrollment Program (DEP) AKA "Zero-touch."

> Only users with the admin role in Fleet can complete these setups.

## Apple Push Notification service (APNs)

To connect Fleet to Apple, get these four files using the Fleet UI or the `fleetctl` command-line interface: An APNs certificate, APNs private key, Simple Certificate Enrollment Protocol (SCEP) certificate, and SCEP private key.

To do this, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Follow the instructions under **Apple Push Certificates Portal**.

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple --email <email> --org <org>`.

2. Follow the on-screen instructions.

> Take note of the Apple ID you use to sign into Apple Push Certificates Portal. You'll need to use the same Apple ID when renewing your APNs certificate. Apple requires that APNs certificates are renewed once every year. To renew, see the [APNs Renewal section](#apns-renewal) .

## Renewing APNs

Apple requires that APNs certificates are renewed once every year. You can see the certificate's renewal date and other important APNs information using the Fleet UI or the `fleetctl` command-line interface:

Fleet UI:

1. Head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Look at the **Apple Push Certificates Portal** section.

`fleetctl` CLI:

1. Run `fleetctl get mdm-apple`.

2. Look at the on-screen information.

How to renew the certificate if it's expired or about to expire:

1. Run the `fleetctl generate mdm-apple --email <email> --org <org>` command. Make sure you use the same Apple ID email address that you used when generating the original certificate.

2. Sign in to [Apple Push Certificates Portal](https://identity.apple.com) using the same Apple ID you used to get your original certificate. If you don't use the same Apple ID, you will have to turn MDM off and back on for all macOS hosts.

3. In the **Settings > Integrations > Mobile device management (MDM)** page, under Apple Push Certificates portal, find the serial number of your current certificate. In Apple Push Certificates Portal, click  **Renew** next to the certificate that has the matching serial number. If you don't renew and get a new certificate, you will have to turn MDM off and back on for all macOS hosts.

## Apple Business Manager (ABM)

_Available in Fleet Premium_

Connect Fleet to your ABM account to automatically enroll macOS hosts to Fleet when they’re first unboxed.

If a new macOS host that appears in ABM hasn't been unboxed, it will appear in Fleet with **MDM status** set to "Pending." These hosts will automatically enroll to the default team in Fleet. Learn how to update the default team [here](#default-team).

To connect Fleet to ABM, first create a new MDM server in ABM and then get these two files using the Fleet UI or the `fleetctl` command-line interface: An ABM certificate and private key.

How to create a new MDM server in ABM:

1. Login to [ABM](https://business.apple.com) and click your name at the bottom of the sidebar, click **Preferences**, then click **MDM Server Assignment**.

2. Click the **Add** button, then enter a unique name for the server. A good name to start is "Fleet MDM."

To get the two files, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. Follow the instructions under **Apple Business Manager**.

`fleetctl` CLI:

1. Run `fleetctl generate mdm-apple-bm`.

2. Follow the on-screen instructions.

### Default team

MacOS hosts purchases through Apple or authorized resellers will automatically enroll to the default team in Fleet when they're first unboxed. This means that Fleet will enforce the default team's settings on these hosts.

> After a host enrolls it can be transferred to a different team. Learn how [here](./Teams.md#transfer-hosts-to-a-team). Transferring a host automatically enforces the new team's settings and removes the old team's settings.

To change the default team, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. In the Fleet UI, head to the **Settings > Integrations > Mobile device management (MDM)** page. Users with the admin role can access the settings pages.

2. In the Apple Business Manager section, select the **Edit team** button next to **Default team**.

3. Choose a team and select **Save**.

`fleetctl` CLI:

1. Create a `config` YAML document if you don't have one already. Learn how [here](./configuration-files/README.md#organization-settings). This document is used to change settings in Fleet.

2. Set the `mdm.apple_bm_default_team` configuration option to the desired team's name.

3. Run the `fleetctl apply -f <your-YAML-file-here>` command.

## Renewing ABM

The Apple Business Manager server token expires after a year or whenever the account that downloaded the token has their password changed. To renew the token, follow the [instructions documented in this FAQ](https://fleetdm.com/docs/using-fleet/faq#how-can-i-renew-my-apple-business-manager-server-token).

<meta name="pageOrderInSection" value="1500">
<meta name="title" value="MDM setup">
