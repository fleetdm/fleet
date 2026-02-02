# Enable Okta Verify on Windows using a SCEP configuration profile

## Introduction

This guide explains how to enable Okta Verify on Windows using a SCEP client certificate delivered by the Windows ClientCertificateInstall CSP. With Fleet's support for Exec commands in configuration profiles, you can now deploy both the SCEP configuration and trigger enrollment in a single configuration profile, eliminating the need for a separate PowerShell script.

## Overview

Okta Verify for Windows uses certificate-based authentication to verify device trust. This requires:

1. A SCEP (Simple Certificate Enrollment Protocol) server endpoint from Okta
2. A Windows configuration profile that configures and triggers SCEP enrollment
3. The certificate to be installed in the User certificate store (not Device)

Fleet now supports Exec commands in configuration profiles, allowing you to deploy both the SCEP configuration and trigger enrollment in a single XML file. The profile uses [Fleet secrets](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) to inject your Okta SCEP details at deployment time, so no file editing is required.

## Prerequisites

Before you begin, you'll need to create Fleet secrets for your Okta SCEP configuration. These secrets will be automatically injected into the configuration profile when it's deployed.

### Step 1: Gather your Okta SCEP details

Collect the following from your Okta tenant:

- **SCEP Server URL**: Your Okta SCEP endpoint (e.g., `https://your-tenant.okta.com/scep/v1/...`)
- **SCEP Challenge**: A static challenge string (avoid special characters like `! @ # $ % ^ & * ( ) _`)
- **CA Thumbprint**: The SHA-256 thumbprint of your Okta CA certificate

### Step 2: Get the CA Thumbprint

Download your Okta CA certificate and extract the SHA-256 thumbprint:

```bash
openssl x509 -in ~/Downloads/ca.cer -noout -fingerprint -sha256
```

The output will look like:
```
SHA256 Fingerprint=E2:18:D7:A7:B0:DF:ED:79:B2:05:73:BA:79:CB:14:B1:FE:EA:D2:7B
```

Remove the colons to get the format needed for Fleet:
```
E218D7A7B0DFED79B20573BA79CB14B1FEEAD27B
```

### Step 3: Create Fleet Secrets

Create four secrets in Fleet (via **Controls** > **Variables** or GitOps). For detailed guidance on using secrets in configuration profiles, see [Fleet's secrets documentation](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

1. **NODE_NAME**
   - Value: A simple identifier for your SCEP node
   - Example: `OKTA` or `OKTAVERIFY`
   - ⚠️ Use only alphanumeric characters

2. **OKTA_SCEP_URL**
   - Value: Your full Okta SCEP server URL
   - Example: `https://your-tenant.okta.com/scep/v1/your-scep-endpoint`

3. **OKTA_SCEP_CHALLENGE**
   - Value: Your SCEP challenge (plain text, alphanumeric recommended)
   - ⚠️ Rotate if it contains special characters, especially underscores

4. **OKTA_CA_THUMBPRINT**
   - Value: Your CA thumbprint (no colons, no spaces)
   - Example: `E218D7A7B0DFED79B20573BA79CB14B1FEEAD27B`

Fleet will automatically inject these as `$FLEET_SECRET_NODE_NAME`, `$FLEET_SECRET_OKTA_SCEP_URL`, `$FLEET_SECRET_OKTA_SCEP_CHALLENGE`, and `$FLEET_SECRET_OKTA_CA_THUMBPRINT` when the profile is deployed.

## Configuration Profile

Fleet supports deploying Windows configuration profiles with embedded Exec commands. The profile includes:

1. **Add nodes**: Configure all SCEP parameters (URL, Challenge, CA Thumbprint, etc.)
2. **Exec node**: Trigger the enrollment immediately after configuration

### Profile Structure

The configuration profile uses the Windows ClientCertificateInstall CSP with the following structure:

```xml
<Add>
  <!-- SCEP configuration nodes -->
</Add>
<Exec>
  <!-- Trigger enrollment -->
</Exec>
```

### Deployment Steps

1. **Download the ready-to-use profile** from the Fleet repository:
   - [install Okta attestation certificate - [Bundle].xml](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/install%20Okta%20attestation%20certificate%20-%20%5BBundle%5D.xml)

2. **No editing required!** The profile is ready to deploy as-is. It uses:
   - `$FLEET_SECRET_NODE_NAME` for the certificate node name
   - `$FLEET_SECRET_OKTA_SCEP_URL` for the SCEP server URL
   - `$FLEET_SECRET_OKTA_SCEP_CHALLENGE` for the SCEP challenge
   - `$FLEET_SECRET_OKTA_CA_THUMBPRINT` for the CA thumbprint

3. **Deploy the profile** using Fleet:
   - Navigate to **Controls** > **OS settings** > **Custom settings**
   - Upload the XML file (no modifications needed)
   - Assign to the appropriate team or hosts
   - Fleet will automatically replace the secret variables when deploying to each device

### Profile Example

Here's what the key parts of the profile look like:

```xml
<Add>
  <Item>
    <Target>
      <LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_SECRET_NODE_NAME</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">node</Format>
    </Meta>
  </Item>
</Add>
<!-- Additional Add nodes for RetryCount, RetryDelay, KeyUsage, etc. -->
<Add>
  <Item>
    <Target>
      <LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_SECRET_NODE_NAME/Install/ServerURL</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Data>$FLEET_SECRET_OKTA_SCEP_URL</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target>
      <LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_SECRET_NODE_NAME/Install/Challenge</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Data>$FLEET_SECRET_OKTA_SCEP_CHALLENGE</Data>
  </Item>
</Add>
<Add>
  <Item>
    <Target>
      <LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_SECRET_NODE_NAME/Install/CAThumbprint</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Data>$FLEET_SECRET_OKTA_CA_THUMBPRINT</Data>
  </Item>
</Add>
<Exec>
  <Item>
    <Target>
      <LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_SECRET_NODE_NAME/Install/Enroll</LocURI>
    </Target>
  </Item>
</Exec>
```

Fleet automatically replaces the `$FLEET_SECRET_*` variables with your actual values when deploying the profile to devices.

## Important Notes

### Fleet Secrets

The profile uses Fleet secrets for all configuration values:
- `$FLEET_SECRET_NODE_NAME` - Your SCEP certificate node name (e.g., `OKTA`)
- `$FLEET_SECRET_OKTA_SCEP_URL` - Your SCEP server endpoint
- `$FLEET_SECRET_OKTA_SCEP_CHALLENGE` - Your SCEP challenge
- `$FLEET_SECRET_OKTA_CA_THUMBPRINT` - Your CA thumbprint

These are automatically replaced when Fleet deploys the profile to each device. Make sure all four secrets are created in Fleet before deploying the profile.

### User vs. Device Certificate Store

Okta requires the certificate to be installed in the **User** certificate store. The profile uses:
```xml
./User/Vendor/MSFT/ClientCertificateInstall/SCEP/...
```

If you use `./Device` instead, the device will **not** be marked as managed in Okta.

### SCEP Challenge Requirements

Your SCEP challenge should:
- Use only alphanumeric characters (letters and numbers)
- Avoid special characters, especially underscores (`_`), which will break deployment
- Not require base64 encoding (Fleet handles plain text challenges)

Special characters can cause errors like:
```
SCEP: Certificate enroll failed. Result: (The string contains a non-printable character.)
```

If your current challenge contains special characters, consider rotating to a simpler value.

## Verification

After deploying the profile, verify the certificate installation:

### Check Certificate in User Store

Open PowerShell as the logged-in user (not as administrator) and run:

```powershell
Get-ChildItem -Path Cert:\CurrentUser\My | Where-Object {$_.Subject -like "*managementAttestation*"}
```

You should see output similar to:

```
Thumbprint                                Subject
----------                                -------
A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0  CN=<UUID> managementAttestation
```

### Check Device Management Logs

View enrollment activity in the Windows Event Log:

```powershell
Get-WinEvent -LogName Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin -MaxEvents 50
```

Look for successful enrollment events related to your SCEP node.

### Verify in Okta Admin Console

1. Log in to your Okta Admin Console
2. Navigate to **Reports** > **System Log**
3. Filter for device attestation events
4. Confirm your device appears as managed with the correct certificate

## Troubleshooting

### Missing Fleet Secrets

**Symptom**: Profile deployment fails or secrets aren't replaced.

**Cause**: Fleet secrets haven't been created.

**Solution**: 
- Verify you've created all four required secrets in Fleet:
  - `NODE_NAME`
  - `OKTA_SCEP_URL`
  - `OKTA_SCEP_CHALLENGE`
  - `OKTA_CA_THUMBPRINT`
- Check the secret names match exactly (case-sensitive)
- Verify the secrets are available to the team/hosts where you're deploying the profile

### Exec Returns 404: Node Name Issue

**Symptom**: The Exec command fails with a 404 error.

**Cause**: The NODE_NAME secret is missing or contains invalid characters.

**Solution**: 
- Ensure the `NODE_NAME` secret exists in Fleet
- Use only alphanumeric characters in your node name (e.g., `OKTA`, `OKTAVERIFY`)
- Avoid special characters and spaces

### Enrollment Fails Immediately

**Possible causes**:
- Incorrect ServerURL
- Wrong CAThumbprint format (ensure no colons or spaces)
- Device cannot reach the SCEP URL (check network/firewall)

**Solution**: Verify your SCEP configuration details and network connectivity.

### Challenge Rejected

**Symptom**: SCEP server rejects the challenge.

**Cause**: Challenge contains special characters or encoding issues.

**Solution**: 
- Use a simpler plain text challenge (alphanumeric only)
- Avoid special characters, especially underscores
- Consider rotating your SCEP challenge in Okta

### Certificate Not in User Store

**Symptom**: Certificate appears in Device store instead of User store, or not at all.

**Cause**: Profile is using `./Device` instead of `./User` in the LocURI.

**Solution**: Ensure all LocURI paths use `./User/Vendor/MSFT/ClientCertificateInstall/SCEP/...`

### Certificate Exists But Device Not Managed

**Symptom**: Certificate is installed but Okta doesn't mark device as managed.

**Cause**: Certificate is in the wrong store (Device instead of User).

**Solution**: Remove the profile, correct the LocURI paths to use `./User`, and redeploy.

## Certificate Renewal

SCEP certificates have expiration dates. Plan for renewal by:

1. **Monitoring expiration**: Use Fleet queries to identify certificates expiring within 30 days:

```sql
SELECT 
    common_name,
    not_valid_after,
    CAST((julianday(not_valid_after) - julianday('now')) AS INTEGER) as days_until_expiry
FROM certificates
WHERE 
    common_name LIKE '%managementAttestation%'
    AND CAST((julianday(not_valid_after) - julianday('now')) AS INTEGER) < 30;
```
