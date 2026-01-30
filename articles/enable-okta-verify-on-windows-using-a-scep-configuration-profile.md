# Enable Okta Verify on Windows using a SCEP configuration profile

## Introduction

This guide explains how to enable [Okta Verify](https://help.okta.com/en-us/content/topics/mobile/okta-verify-overview.htm) on Windows using a SCEP client certificate delivered by the Windows **ClientCertificateInstall** CSP and then applied using an **Exec** command. This pattern is useful when your MDM payload cannot send **Add or Replace** nodes together with an **Exec** in one transaction.

You will deploy the SCEP profile first, then call **Enroll** via Exec to request the client certificate.

**Files**
* [Profile XML](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/install%20Okta%20attestation%20certificate%20-%20%5BBundle%5D.xml)
* [Powershell script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/trigger%20scep%20enrollment.ps1)

---

## Order at a glance

1. Get your CA **thumbprint**, choose **{yourCertName}**, and locate your SCEP **URL** and **Challenge**.  
2. Create Fleet **secrets** for URL, Challenge, CA thumbprint, and API token.  
3. Use the Fleet repo XML CSP profile and replace only the required placeholders.  
4. Deploy the profile to devices.  
5. Update the **Exec** script to use the same `{yourCertName}` and your secrets, then run it.  
6. Verify the certificate is installed.  
7. Plan and automate **renewal**.

---

## Prerequisites

* Windows devices enrolled to Fleet MDM  
* Okta SCEP endpoint with a static challenge  
* Root CA certificate thumbprint for the SCEP issuing CA  
* Fleet API token stored as a secret  
* Optional GitOps workflow if you manage Fleet configuration as code

---

## Step 1. Collect your values

### 1.1 Get the CA thumbprint

**Windows PowerShell**
```powershell
Get-FileHash -Path "C:\Path\To\ca.cer" -Algorithm SHA256 | Select-Object -ExpandProperty Hash
```

**macOS or Linux**
```bash
openssl x509 -in ~/Downloads/ca.cer -noout -fingerprint -sha256
# Output looks like:
# SHA256 Fingerprint=E2:18:D7:A7:B0:DF:ED:79:B2:05:73:BA:79:CB:14:B1:FE:EA:D2:7B
# Remove the colons:
# E218D7A7B0DFED79B20573BA79CB14B1FEEAD27B
```

Use the hex string without colons or spaces in the secret you will create below.

### 1.2 Choose your SCEP node name

Pick a simple value for `{yourCertName}`, for example `OKTA` or `OKTAVERIFY`. You will use this exact value:
* in the XML profile path `.../SCEP/{yourCertName}/Install/...`  
* in the Exec path `.../SCEP/{yourCertName}/Install/Enroll`

### 1.3 Get your SCEP URL and Challenge

* `{yourScepUrl}` is your Okta SCEP endpoint.  
* `{yourScepChallenge}` is your static SCEP challenge. This profile expects **plain text**. Avoid special characters that can break XML or transport. Recommended: letters, numbers, underscore. If your challenge contains characters such as `! @ # $ % ^ & * ( )`, rotate to a simpler value.

**Quick checklist**
* {yourCertName} chosen  
* {yourScepUrl} confirmed  
* {yourScepChallenge} validated (plain text, simple characters)  
* {yourScepCAThumbprint} ready (no colons, no spaces)

---

## Step 2. Create Fleet secrets

Follow Fleet’s guide: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles

Create these secrets in Fleet (Controls → Variables) or via GitOps:

| Secret name | Value you collected |
|---|---|
| `FLEET_SECRET_OKTA_SCEP_URL` | `{yourScepUrl}` |
| `FLEET_SECRET_OKTA_SCEP_CHALLENGE` | `{yourScepChallenge}` (plain text, simple characters) |
| `FLEET_SECRET_OKTA_CA_THUMBPRINT` | SHA256 thumbprint with no colons, no spaces
| `FLEET_SECRET_API` | Fleet API token used by the Exec script |

Optional convenience secret:
* `FLEET_SECRET_OKTA_CERT_NAME` set to `{yourCertName}`

**Security notes**
* Fleet does not hide the secret in script results. Don't print/echo your secrets to the console output.

---

## Step 3. Use Fleet’s XML CSP profile

Source file in the Fleet repo:
```
docs/solutions/Windows/configuration-profiles/install Okta attestation certificate - [Bundle].xml
```

Only change the following placeholders:

* `{yourCertName}` set to the SCEP node name you chose in Step 1.2  
* `{yourScepUrl}` replaced with `$FLEET_SECRET_OKTA_SCEP_URL`  
* `{yourScepChallenge}` replaced with `$FLEET_SECRET_OKTA_SCEP_CHALLENGE` (plain text, simple characters)  
* `{yourScepCAThumbprint}` replaced with `$FLEET_SECRET_OKTA_CA_THUMBPRINT` (no colons, no spaces)

**Important**  
Use the same `{yourCertName}` in both the profile path and the Exec path. If they differ, the Exec will 404.

### Replace just these lines in the profile

```xml
<!-- SCEP Server URL -->
<Item>
  <Target>
    <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/{yourCertName}/Install/ServerURL</LocURI>
  </Target>
  <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
  <Data>$FLEET_SECRET_OKTA_SCEP_URL</Data>
</Item>

<!-- SCEP Challenge (plain text) -->
<Item>
  <Target>
    <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/{yourCertName}/Install/Challenge</LocURI>
  </Target>
  <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
  <Data>$FLEET_SECRET_OKTA_SCEP_CHALLENGE</Data>
</Item>

<!-- SCEP CA Thumbprint (no colons) -->
<Item>
  <Target>
    <LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/{yourCertName}/Install/CAThumbprint</LocURI>
  </Target>
  <Meta><Format xmlns="syncml:metinf">chr</Format></Meta>
  <Data>$FLEET_SECRET_OKTA_CA_THUMBPRINT</Data>
</Item>
```

Keep the other defaults from the file (KeyLength 2048, KeyUsage 160, HashAlgorithm `SHA-1`, SubjectName `CN=$FLEET_VAR_HOST_UUID managementAttestation`, EKUMapping, RetryCount, RetryDelay).

Deploy the profile to your Windows hosts using Fleet.

---

## Step 4. Update the Exec script and run Enroll

Script location in repo:  
`docs/solutions/Windows/scripts/trigger-scep-enrollment.ps1`

Your Exec must target the same `{yourCertName}` as in the profile. Example path:
```
./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/{yourCertName}/Install/Enroll
```

Update your PowerShell script to read the API token from the secret, set your node name, and build the correct LocURI.

```powershell
# ----- USER SETTINGS -----
# Add your secrets in Fleet (Controls > Variables) or via GitOps.
# The variable named "API" becomes FLEET_SECRET_API
# Full guidance: https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles

$NODE_NAME = "OKTA"               # must match {yourCertName} in the XML
$FLEET_API = "$FLEET_SECRET_API"  # injected by Fleet

$locUri = "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$NODE_NAME/Install/Enroll"
# ...construct and send the Exec command body referencing $locUri...
```

Run the script from Fleet so secrets inject automatically.

---

## Step 5. Verify enrollment

**PowerShell**
```powershell
Get-ChildItem Cert:\LocalMachine\My |
  Where-Object { $_.Subject -like "*managementAttestation*" } |
  Format-List Subject, Thumbprint, NotAfter
```

**GUI**
* Open `certlm.msc`  
* Personal > Certificates  
* Confirm a certificate whose Subject contains `managementAttestation`

---

## Step 6. Renewal

* Automated workflow. Use a Fleet query to find certificates expiring within 30 days and trigger the Exec command for those hosts.

Find certs expiring within 30 days:
```TODO!
```

---

## Troubleshooting

* Exec returns 404: node name mismatch. Ensure `{yourCertName}` in XML equals `$NODE_NAME` in the script.  
* Enrollment fails immediately: check `ServerURL`, `CAThumbprint` format, and that the device can reach the SCEP URL.  
* Challenge rejected: try a simpler plain text challenge, or base64 encode and update the XML `<Data>`.  
* Nothing in `Cert:\LocalMachine\My`: review Device Management logs  
  ```powershell
  Get-WinEvent -LogName Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin -MaxEvents 50
  ```

---

<meta name="articleTitle" value="Enable Okta Verify on Windows using a SCEP configuration profile">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-01-23">
<meta name="description" value="Enable Okta Verify on Windows using a SCEP client certificate with ClientCertificateInstall CSP, Exec enrollment, Fleet secrets, verification, troubleshooting, and renewal.">
