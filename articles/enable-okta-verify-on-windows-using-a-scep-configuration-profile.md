# Enable Okta Verify on Windows using a SCEP configuration profile

## Introduction

This guide explains how to enable Okta Verify on Windows using a SCEP client certificate delivered by the Windows ClientCertificateInstall CSP. Fleet supports Exec commands in configuration profiles, allowing you to deploy the SCEP configuration and trigger enrollment in a single profile.

## Files

**Profile XML**: [install Okta attestation certificate - [Bundle].xml](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/install%20Okta%20attestation%20certificate%20-%20%5BBundle%5D.xml)

The profile is ready to use as-is. Fleet will replace the `$FLEET_SECRET_*` variables with your actual values when deploying to each device.

## Prerequisites

### 1. Gather your Okta details

Collect from your Okta tenant:

* **SCEP URL**: Your Okta SCEP endpoint
* **SCEP Challenge**: Your static SCEP challenge (plain text, avoid special characters)
* **CA Thumbprint**: The SHA-256 thumbprint of your Okta CA certificate

### 2. Get your CA thumbprint

Download your Okta CA certificate and extract the SHA-256 thumbprint.

**macOS/Linux**:
```bash
openssl x509 -in ~/Downloads/ca.cer -noout -fingerprint -sha256
```

**Windows**:
```powershell
certutil -hashfile ca.cer SHA256
```

Output will look like:
```
SHA256 Fingerprint=E2:18:D7:A7:B0:DF:ED:79:B2:05:73:BA:79:CB:14:B1:FE:EA:D2:7B
```

Remove the colons:
```
E218D7A7B0DFED79B20573BA79CB14B1FEEAD27B
```

### 3. SCEP challenge requirements

* Your SCEP challenge should be plain text
* Avoid special characters that can break XML or transport
* **Recommended**: letters, numbers only
* If your challenge contains `! @ # $ % ^ & * ( ) _`, rotate to a simpler value

## Quick checklist

* SCEP URL confirmed
* SCEP challenge validated (plain text, simple characters)
* CA thumbprint ready (no colons, no spaces)

## Deployment

### 1. Create Fleet secrets

Follow Fleet's guide: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles

Create these secrets in Fleet (**Controls** > **Variables**) or via GitOps:

| Secret name | Value |
|-------------|-------|
| `OKTA_SCEP_URL` | Your SCEP endpoint URL |
| `OKTA_SCEP_CHALLENGE` | Your challenge (plain text, simple characters) |
| `OKTA_CA_THUMBPRINT` | Your thumbprint (no colons, no spaces) |

### 2. Deploy the profile

1. Download the profile XML (link above)
2. Navigate to **Controls** > **OS settings** > **Custom settings** in Fleet
3. Upload the XML file (no editing required)
4. Assign to your team or hosts

Fleet automatically replaces `$FLEET_SECRET_OKTA_SCEP_URL`, `$FLEET_SECRET_OKTA_SCEP_CHALLENGE`, and `$FLEET_SECRET_OKTA_CA_THUMBPRINT` when deploying. The certificate ID is automatically managed by Fleet using `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID`.

## Verify the certificate

### Check the User cert store

Open PowerShell as the logged-in user (not administrator):

```powershell
Get-ChildItem -Path Cert:\CurrentUser\My | Where-Object {$_.Subject -like "*managementAttestation*"}
```

Expected output:
```
Thumbprint                                Subject
----------                                -------
A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0  CN=<SERIAL> managementAttestation
```

### Check the device management logs

```powershell
Get-WinEvent -LogName Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin -MaxEvents 50
```

### Verify in Okta

1. Log in to Okta Admin Console
2. Navigate to **Reports** > **System Log**
3. Filter for device attestation events
4. Confirm the device appears as managed

## Troubleshooting

### Exec returns 404

* Check that all three Fleet secrets exist (`OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, `OKTA_CA_THUMBPRINT`)
* Verify the profile was uploaded correctly
* Review Device Management logs for details

### Enrollment fails immediately

Check:
* ServerURL is correct
* CAThumbprint format (no colons or spaces)
* Device can reach the SCEP URL (network/firewall)

### Challenge rejected

* Try a simpler plain text challenge (alphanumeric only)
* Avoid special characters, especially underscores
* If your challenge contains `! @ # $ % ^ & * ( ) _`, rotate to a simpler value in Okta

### Nothing in Cert:\LocalMachine\My

**Note**: Okta requires certificates in the **User** store (`Cert:\CurrentUser\My`), not the Device store.

Review Device Management logs:
```powershell
Get-WinEvent -LogName Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin -MaxEvents 50
```

## Plan and automate renewal

### Monitor expiration

Use a Fleet policy to identify devices with certificates expiring within 30 days:

```sql
SELECT 1 
FROM certificates
WHERE 
    common_name LIKE '%managementAttestation%'
    AND julianday(not_valid_after) - julianday('now') < 30;
```

This policy will:
- **Fail**: When a certificate exists and expires within 30 days (needs renewal)
- **Pass**: When no certificate exists yet, or certificate is valid for more than 30 days

### Automated workflow

To renew certificates, you can:

**Manual redeployment**: Redeploy the same configuration profile to trigger renewal

## Important notes

* **Fleet secrets**: Fleet does not hide secrets in profile results. Make sure all three secrets are created before deploying (`OKTA_SCEP_URL`, `OKTA_SCEP_CHALLENGE`, `OKTA_CA_THUMBPRINT`).
* **User vs Device store**: Okta requires certificates in the User store. The profile uses `./User/` paths. If you use `./Device`, the device will **not** be marked as managed in Okta.
* **Certificate ID**: Fleet automatically manages the certificate node name using `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` - no manual configuration needed.

---

<meta name="articleTitle" value="Enable Okta Verify on Windows">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-23">
<meta name="description" value="Enable Okta Verify on Windows using a SCEP configuration profile">
