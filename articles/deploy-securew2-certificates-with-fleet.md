# Deploy SecureW2 certificates with Fleet

_Available in Fleet Premium_

SecureW2 JoinNow PKI issues the certificates your hosts use for passwordless Wi-Fi and VPN access through SecureW2 CloudRADIUS. This guide connects SecureW2 to Fleet so every Apple host (macOS, iOS, and iPadOS) gets a SecureW2 certificate and joins your network without the end user doing anything. Fleet renews the certificate automatically before it expires.

## Choose ACME or SCEP

SecureW2 can issue certificates to Fleet-managed hosts over two protocols. Most of the setup in SecureW2 is the same for both.

- **ACME** (recommended for Apple hosts): The host proves its identity with Apple's [Managed Device Attestation](https://support.apple.com/guide/deployment/managed-device-attestation-dep28afbde6a/web) and can keep its private key in the Secure Enclave, where it can't be exported. SecureW2 generates the configuration profile for you.
- **SCEP**: Uses Fleet's custom SCEP certificate authority (CA). Choose SCEP when a third-party app, like a VPN client, needs to read the certificate, or when you want the same CA in Fleet for other platforms.

## Prerequisites

- A SecureW2 JoinNow PKI subscription and admin access to the JoinNow Management Portal.
- Fleet Premium with [Apple MDM turned on](https://fleetdm.com/guides/apple-mdm-setup).
- Hosts enrolled in Fleet's MDM. Installing Fleet's agent (fleetd) alone isn't enough, because Fleet delivers certificates with configuration profiles.
- For ACME, hosts on macOS 13.1, iOS 16, or iPadOS 16 or later. The profile SecureW2 generates uses a hardware-bound key, which Macs only support on macOS 14 or later with Apple silicon or a T2 chip. For other Macs, set `HardwareBound` to `false` and remove `Attest`.
- For SCEP on a self-hosted Fleet server, a [server private key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key).

## Step 1: Create an intermediate CA in SecureW2

1. In the JoinNow Management Portal, go to **Dynamic PKI > Certificate Authorities** and select **Add Certificate Authority**.
2. For **Generate CA For**, select **Device and User Authentication**.
3. For **Type**, select **Intermediate CA**, and for **Certificate Authority**, select your organization's root CA.
4. Enter a **Common Name**, for example "Fleet ACME CA" or "Fleet SCEP CA".
5. Set **Key Size** to **2048** and **Signature Algorithm** to **SHA-256**.
6. Enter the **Validity Period (in years)** and select **Save**.

## Step 2: Create a certificate template

1. On the same **Dynamic PKI > Certificate Authorities** page, scroll to **Certificate Templates** and select **Add Certificate Template**.
2. Enter a **Name** and **Display Description**, for example "Fleet Wi-Fi certificate".
3. Keep the default **Subject** and **SAN** values.
4. Set the **Validity Period** (for example, `1y`) and keep **Signature Algorithm** at **SHA-256**.
5. Under **Extended Key Usage**, in **Use Certificate For**, select **Client Authentication**.
6. Optional: Select **Notify admin on certificate expiry** to email your SecureW2 admins before certificates expire.
7. Select **Save**.

> **Note:** Fleet's [automatic renewal](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal) finds each certificate by a renewal ID that Fleet puts in the certificate request's OU (organizational unit). If you change the template's **Subject**, keep the OU from the request so it appears in the issued certificate. Set the validity period to 2 days or longer, since Fleet can't renew shorter-lived certificates.

## Step 3: Create a device management platform for Fleet

The device management platform is what SecureW2 trusts to request certificates. Creating it gives you the credentials Fleet needs.

1. Go to **Integrations Hub > Device Management Platforms** and select **Add**.
2. Enter a **Name**, for example "Fleet", and a **Description**.
3. For **Type**, select the option that matches your protocol:
   - ACME: **ACME Client Certificate Enrollment Token**
   - SCEP: **SCEP (Multi-Vendor) Enrollment Token**
4. For **Vendor**, select **Other**, and for **Vendor Name**, enter "Fleet".
5. SCEP only: For **Certificate Authority**, select the intermediate CA from step 1.
6. Select **Save**. SecureW2 downloads a file to your computer:
   - ACME: a `.mobileconfig` profile with your enrollment token. You'll upload it to Fleet in step 5.
   - SCEP: a `.csv` file with the **SCEP URL** and **Secret**. You'll enter these in Fleet in step 5.

Both files contain a credential that lets a device request certificates. Store them in a secrets manager and delete the downloaded copy when you're done.

> **Note:** Use SecureW2's static **SCEP (Multi-Vendor) Enrollment Token**. SecureW2's dynamic SCEP challenge isn't compatible with Fleet's NDES integration, so Fleet connects to SecureW2 as a custom SCEP CA instead.

## Step 4: Create SecureW2 policies

SecureW2 uses three policies to decide who gets a certificate and what network access it grants. Create them in this order, because each one refers to the previous one.

### Policy workflow

The policy workflow identifies hosts that come from Fleet.

1. Go to **Policy Management > Policy Workflows** and select **Add Policy Workflow**.
2. Enter a **Name**, for example "Fleet policy workflow", and select **Save**.
3. On the **Conditions** tab, for **Core Provider**, select the device management platform from step 3.
4. Select **Update**.

### Enrollment policy

The enrollment policy decides which CA and template issue certificates to that workflow.

1. Go to **Policy Management > Enrollment** and select **Add Enrollment Policy**.
2. Enter a **Name**, for example "Fleet enrollment policy", and select **Save**.
3. On the **Conditions** tab, for **Policy Workflow**, select your Fleet policy workflow. For **Device Role**, select **DEFAULT DEVICE ROLE**.
4. On the **Settings** tab, for **Use Certificate Authority**, select the intermediate CA from step 1. For **Use Certificate Template**, select the template from step 2.
5. Keep the other defaults and select **Update**.

### Network policy

The network policy tells SecureW2 CloudRADIUS how to respond when a host with a Fleet-issued certificate connects.

1. Go to **Policy Management > Network** and select **Add Network Policy**.
2. Enter a **Name**, for example "Fleet network policy", and select **Save**.
3. On the **Conditions** tab, select **Match All**, then select **Add rule**.
4. Expand **Identity**, select **Policy Workflow**, and select **Save**.
5. For **Role Equals**, select your Fleet policy workflow.
6. On the **Settings** tab, keep **Access** set to **Allow**.
7. Optional: Turn on MFA or select **Add Attribute** to return RADIUS attributes, like a VLAN assignment, to your Wi-Fi controller. Without attributes, SecureW2 sends a plain Access-Accept for any valid certificate.
8. Select **Update**.

## Step 5: Deploy the certificate with Fleet

Follow the option that matches the device management platform type you created in step 3.

### Option A: ACME

The profile from step 3 already contains an ACME payload with your SecureW2 enrollment token and a Wi-Fi payload that uses the ACME certificate. You only need to make two edits.

1. Open the `.mobileconfig` file from step 3 in a text editor.
2. In the Wi-Fi payload, replace `<Wifi Name>` in `SSID_STR` with your network's SSID.
3. In the ACME payload's `Subject`, add an `OU` entry with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` so Fleet can renew the certificate. Keep the `CN` value that SecureW2 generated.
4. In Fleet, select the fleet your hosts belong to, go to **Controls > OS settings > Configuration profiles**, and select **Add profile**.
5. Upload the edited file, choose **All hosts** or target specific labels, and select **Add profile**.

Keep the ACME and Wi-Fi payloads in the same profile. The Wi-Fi payload's top-level `PayloadCertificateUUID` must match the ACME payload's `PayloadUUID`, and Apple only resolves that reference within a single profile. If your SecureW2 profile has `PayloadCertificateUUID` inside `EAPClientConfiguration`, move it up a level, as shown in the example.

> **Warning:** With `HardwareBound` set to `true`, macOS keeps the identity in the Secure Enclave, and only Apple's built-in clients can use it: Wi-Fi (802.1X), the built-in VPN client, and VPN apps built as Network Extension VPN providers. Third-party apps that read certificates from the keychain, like Cisco Secure Client on macOS, can't. For those apps, set `HardwareBound` to `false`, add `AllowAllAppsAccess` set to `true`, and remove `Attest` (it requires a hardware-bound key), or use [option B](#option-b-scep) instead.

<details>
<summary>Example ACME and Wi-Fi profile</summary>

Your profile from SecureW2 includes your own `ClientIdentifier`, `CN`, and UUIDs. Don't copy those values from this example.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>Attest</key>
            <true/>
            <key>ClientIdentifier</key>
            <string><SECUREW2_CLIENT_IDENTIFIER></string>
            <key>DirectoryURL</key>
            <string>https://connector.securew2.com/acme/</string>
            <key>ExtendedKeyUsage</key>
            <array>
                <string>1.3.6.1.5.5.7.3.2</string>
            </array>
            <key>HardwareBound</key>
            <true/>
            <key>KeySize</key>
            <integer>384</integer>
            <key>KeyType</key>
            <string>ECSECPrimeRandom</string>
            <key>Subject</key>
            <array>
                <array>
                    <array>
                        <string>CN</string>
                        <string><SECUREW2_GENERATED_CN></string>
                    </array>
                </array>
                <array>
                    <array>
                        <string>OU</string>
                        <string>$FLEET_VAR_CERTIFICATE_RENEWAL_ID</string>
                    </array>
                </array>
            </array>
            <key>SubjectAltName</key>
            <dict>
                <key>ntPrincipalName</key>
                <string>%SerialNumber%@example.com</string>
            </dict>
            <key>PayloadDisplayName</key>
            <string>SecureW2 ACME certificate</string>
            <key>PayloadIdentifier</key>
            <string>com.example.securew2.acme</string>
            <key>PayloadType</key>
            <string>com.apple.security.acme</string>
            <key>PayloadUUID</key>
            <string>04BEE422-FB34-48C5-A993-7705DB35F37B</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
        <dict>
            <key>AutoJoin</key>
            <true/>
            <key>EncryptionType</key>
            <string>WPA2</string>
            <key>HIDDEN_NETWORK</key>
            <false/>
            <key>SSID_STR</key>
            <string><YOUR_SSID></string>
            <key>EAPClientConfiguration</key>
            <dict>
                <key>AcceptEAPTypes</key>
                <array>
                    <integer>13</integer>
                </array>
                <key>TLSTrustedServerNames</key>
                <array>
                    <string>radius01.securew2.com</string>
                </array>
            </dict>
            <key>PayloadCertificateUUID</key>
            <string>04BEE422-FB34-48C5-A993-7705DB35F37B</string>
            <key>PayloadDisplayName</key>
            <string>SecureW2 Wi-Fi</string>
            <key>PayloadIdentifier</key>
            <string>com.example.securew2.wifi</string>
            <key>PayloadType</key>
            <string>com.apple.wifi.managed</string>
            <key>PayloadUUID</key>
            <string>B86D4C50-84E0-42DF-8D63-71822AA4CEE4</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>SecureW2 Wi-Fi and certificate</string>
    <key>PayloadIdentifier</key>
    <string>com.example.securew2</string>
    <key>PayloadScope</key>
    <string>System</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>6535F148-3142-4C2C-83CB-A7F1F7FACE90</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

</details>

### Option B: SCEP

1. In Fleet, go to **Settings > Integrations > Certificate authorities** and select **Add CA**.
2. Select **Custom Simple Certificate Enrollment Protocol (SCEP)**.
3. Enter a **Name** in all caps snake case, for example "SECUREW2_WIFI". Fleet uses this name in the profile variables below.
4. Enter the **SCEP URL** from the `.csv` file as **SCEP URL** and the **Secret** as **Challenge**, then select **Add CA**.
5. Create a configuration profile with a SCEP payload and a Wi-Fi payload (see the example below). In the SCEP payload:
   - For `Challenge`, use `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_{CA_NAME}`.
   - For `URL`, use `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_{CA_NAME}`.
   - In `Subject`, add an `OU` entry with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`.
   - Replace `{CA_NAME}` with the name from step 3. For example, `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_SECUREW2_WIFI`.
6. In the Wi-Fi payload, set `PayloadCertificateUUID` to the SCEP payload's `PayloadUUID`, and set `SSID_STR` to your network's SSID.
7. In Fleet, select the fleet your hosts belong to, go to **Controls > OS settings > Configuration profiles**, and select **Add profile**.
8. Upload the profile, choose **All hosts** or target specific labels, and select **Add profile**.

Fleet replaces the variables when it delivers the profile, and proxies each host's request to SecureW2. For Windows and Android hosts, use the profiles in the [custom SCEP section](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-scep-simple-certificate-enrollment-protocol-ca) of Fleet's certificate guide with the same CA.

<details>
<summary>Example SCEP and Wi-Fi profile</summary>

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadContent</key>
            <dict>
                <key>Challenge</key>
                <string>$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_SECUREW2_WIFI</string>
                <key>Key Type</key>
                <string>RSA</string>
                <key>Key Usage</key>
                <integer>5</integer>
                <key>Keysize</key>
                <integer>2048</integer>
                <key>Subject</key>
                <array>
                    <array>
                        <array>
                            <string>CN</string>
                            <string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string>
                        </array>
                    </array>
                    <array>
                        <array>
                            <string>OU</string>
                            <string>$FLEET_VAR_CERTIFICATE_RENEWAL_ID</string>
                        </array>
                    </array>
                </array>
                <key>URL</key>
                <string>$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_SECUREW2_WIFI</string>
            </dict>
            <key>PayloadDisplayName</key>
            <string>SecureW2 SCEP certificate</string>
            <key>PayloadIdentifier</key>
            <string>com.example.securew2.scep</string>
            <key>PayloadType</key>
            <string>com.apple.security.scep</string>
            <key>PayloadUUID</key>
            <string>9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
        <dict>
            <key>AutoJoin</key>
            <true/>
            <key>EncryptionType</key>
            <string>WPA2</string>
            <key>HIDDEN_NETWORK</key>
            <false/>
            <key>SSID_STR</key>
            <string><YOUR_SSID></string>
            <key>EAPClientConfiguration</key>
            <dict>
                <key>AcceptEAPTypes</key>
                <array>
                    <integer>13</integer>
                </array>
                <key>TLSTrustedServerNames</key>
                <array>
                    <string>radius01.securew2.com</string>
                </array>
            </dict>
            <key>PayloadCertificateUUID</key>
            <string>9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
            <key>PayloadDisplayName</key>
            <string>SecureW2 Wi-Fi</string>
            <key>PayloadIdentifier</key>
            <string>com.example.securew2.wifi</string>
            <key>PayloadType</key>
            <string>com.apple.wifi.managed</string>
            <key>PayloadUUID</key>
            <string>4F0A1C2B-5D6E-4F70-8A9B-0C1D2E3F4A5B</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>SecureW2 Wi-Fi and certificate</string>
    <key>PayloadIdentifier</key>
    <string>com.example.securew2</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4CD1BD65-1D2C-4E9E-9E18-9BCD400CDEDC</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

</details>

> **Note:** `TLSTrustedServerNames` must match the name on your CloudRADIUS server's certificate. The examples use SecureW2's default, `radius01.securew2.com`. If yours differs, use your own value, or hosts will reject the RADIUS server.

## Verify

1. In Fleet, open a host's **Host details** page and select **OS settings**. The profile should show **Verified**. Errors appear here if something failed.
2. On the same page, check the host's [certificates](https://fleetdm.com/guides/view-certificates-in-host-vitals). Fleet lists hardware-bound ACME certificates on macOS hosts too, even though they aren't in the keychain.
3. Confirm the certificate's subject includes an OU that starts with `fleet-`. That's the renewal ID, and Fleet needs it to renew the certificate.
4. On the host, confirm it joined your Wi-Fi network without a prompt.
5. In the JoinNow Management Portal, go to **Dynamic PKI > Certificate Authorities** and select **Certificates** next to your intermediate CA to see the issued certificate.

## Troubleshoot

**A VPN client or other app can't find the certificate**

The ACME identity is hardware-bound, so only Apple's built-in clients can use it. Set `HardwareBound` to `false` and `AllowAllAppsAccess` to `true` in the ACME payload, or switch to SCEP. See the warning in [option A](#option-a-acme).

**The certificate expires instead of renewing**

The issued certificate is missing the renewal ID. Confirm the profile has `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the `OU`, and that your SecureW2 certificate template keeps the OU from the request. Then resend the profile from the host's **Host details** page. Learn more in [renewal](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal).

**End users are asked to trust the RADIUS server**

The host can't validate the CloudRADIUS server's certificate. Check that `TLSTrustedServerNames` matches the server's certificate name. To skip the prompt entirely, add the CA that issued the RADIUS server certificate to the profile as a certificate payload. Then, in `EAPClientConfiguration`, set `PayloadCertificateAnchorUUID` to an array with that payload's UUID.

**The SCEP profile fails on every host**

Fleet can't get a certificate from SecureW2. Edit the CA in **Settings > Integrations > Certificate authorities** and re-enter the **SCEP URL** and **Challenge** from the `.csv` file. Also check that the enrollment policy points to the right CA and template.

## Further reading

- [Deploy certificates to connect end users to third-party tools](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate)
- [Enterprise Wi-Fi authentication methods](https://fleetdm.com/guides/enterprise-wifi-authentication-methods)
- [Fleet variables](https://fleetdm.com/guides/fleet-variables)

<meta name="articleTitle" value="Deploy SecureW2 certificates with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-11-10">
<meta name="category" value="guides">
<meta name="description" value="Deploy SecureW2 JoinNow PKI certificates to Apple hosts with ACME or SCEP so they join Wi-Fi without end user action.">
