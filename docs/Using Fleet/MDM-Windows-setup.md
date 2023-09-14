# Windows setup

## Supported Windows versions

Windows 10 (Pro) and higher.

## Overview

> Windows MDM features are not ready for production and are currently in development. These features are disabled by default.

Windows MDM features require configuring Fleet with a certificate and key pair. This guide will walk you through how to upload these to Fleet and turn on Windows MDM in order to use Windows MDM features.

Automatic enrollment allows Windows to automatically enroll to Fleet when they are first set up. Automatic enrollment requires Azure Active Directory (Azure AD) This guide will also walk you through how to connect Azure AD to Fleet. 

> **Note** you are only required to connect Azure AD to Fleet if you are using Automatic enrollment AKA "Zero-touch."

## Requirements
To use Fleet's Windows MDM features you need to have:
- A [deployed Fleet instance](../Deploying/Introduction.md).
- A Fleet user with the admin role.

## Configuring Fleet

Fleet uses a certificate and key pair to authenticate and manage interactions between Fleet and the host.

This section will show you how to:
1. Configure Fleet with your certificate and key
2. Turn on Windows MDM in Fleet

### Step 1: configure Fleet with your certificate and key

1. In your Fleet server configuration, set the contents of the certificate and key in the following environment variables:

> Note: Any environment variable that ends in `_BYTES` expects the file's actual content to be passed in, not a path to the file. If you want to pass in a file path, remove the `_BYTES` suffix from the environment variable.

- [FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-windows-wstep-identity-cert-bytes)
- [FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY_BYTES](https://fleetdm.com/docs/deploying/configuration#mdm-windows-wstep-identity-key-bytes)

2. Set the `FLEET_MDM_WINDOWS_ENABLED_AND_CONFIGURED` environment variable to `true`.

3. Restart the Fleet server.

### Step 2: Turn on Windows MDM in Fleet

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

## Azure Active Directory (Azure AD)

> Available in Fleet Premium

By connecting Fleet to Azure AD, Windows hosts can automatically enroll to Fleet when they’re first unboxed and set up by your end user.

<meta name="pageOrderInSection" value="1501">
<meta name="title" value="Windows setup">
<meta name="description" value="Learn how to set up Windows MDM features in Fleet.">
<meta name="navSection" value="Device management">