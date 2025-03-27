# Connect end users to Wi-Fi or VPN with a certificate (from DigiCert, NDES, or custom SCEP)

_Available in Fleet Premium_

Fleet can help your end users connect to Wi-Fi or VPN by adding your certificate authority to issue
certificates. Fleet currently supports
[DigiCert](https://www.digicert.com/digicert-one), [Microsoft
NDES](https://learn.microsoft.com/en-us/windows-server/identity/ad-cs/network-device-enrollment-service-overview),
and custom [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) server.

This guide will walk you through configuring your certificate authority and delivering certificates.

## DigiCert

To install certificates from DigiCert to hosts, do the following steps:

- [Create service user in DigiCert](#step-1-create-service-user-in-digicert)
- [Create certificate profile in DigiCert](#step-2-create-certificate-profile-in-digicert)
- [Connect Fleet to DigiCert](#step-3-connect-fleet-to-digicert)
- [Add a PKCS12 configuration profile to Fleet](#step-4-add-pkcs-12-configuration-profile-to-fleet)

### Step 1: Create service user in DigiCert

1. Head to [DigiCert One](https://one.digicert.com/)
2. Follow instructions [here](https://docs.digicert.com/en/platform-overview/manage-your-accounts/account-manager/users-and-access/service-users/create-a-service-user.html), to create service user, and save service user API token.
> Make sure to assign **User and certificate manager** and **Certificate profile manager** roles
> when creating service user.

### Step 2: Create certificate profile in DigiCert

1. In DigiCert [Trust Lifcycle Manager](https://one.digicert.com/mpki/dashboard), select
   **Policies > Certificate profiles** from the main menu, then select **Create profile from
   template**, and select **Generic Device Certificate** from the list.
2. Add a friendly **Profile name** (e.g. "Fleet - Wi-Fi authentication").
3. Select your **Business unit** and **Issuing CA**.
4. Select **REST API** from **Enrollment method**, then select **3rd party app** from
   **Authentication method** dropdown, and select **Next**.
5. Configure certificate expiration as you wish.
6. In **Subject DN and SAN fields** section, make sure to add **Common name** and **Other name
   (UPN)**. For **Common name**, select **REST request** from **Source for the field's value**
   dropdown, and check **Required**. For **Other name (UPN)**, select **REST Request**, and check
   both **Required** and **Multiple** checkboxes.
7. You can click **Next** and leave all default options until you get to the last step, where you
   need to select service user created in first step from **Select Service User** dropdown, and
   select **Create**

### Step 3: Connect Fleet to DigiCert

1. Go to Fleet, navigate to **Settings**, select **Integrations** tab, and select
   **Certificates**.
2. Select **Add CA** button, and select **DigiCert** from the dropdown on the top.
3. Add **Name** for your certificate authority. It's best to use all caps, beacuse it will be used
   as variable name in configuration profile and name it based on your use case (e.g.
   WIFI_AUTHENTICATION).
4. Keep default **URL**, or adjust if you're using on-prem DigiCert One. URL should match the one
   you use to login to your DigiCert One account.
5. Paste **API token** from the **Step 1** section above.
6. Paste **Profile GUID** of certificate profile created in the **Step 2** section above. To get
   Profile GUID, go to [Certificate profiles](https://one.digicert.com/mpki/policies/profiles) page,
   open your profile, and copy **GUID**.
7. For **CN**, **UPN**, and **Certificate seat ID**, you can use fixed values or one of the [Fleet's
   host
   variables](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings).
8. Select **Add CA**, and your DigiCert certificate authority (CA) should appear in the list.

### Step 4: Add PKCS12 configuration profile to Fleet

[Add a configuration profile](https://fleetdm.com/guides/custom-os-settings) to Fleet, that includes the PKCS12 payload. In the profile, you will need to set `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>` as the `Password` and `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>` as the `Data`.

Replace `<CA_NAME>` part of the variable, with name that you used in the section above, to connect
Fleet to DigiCert (e.g if name of the certificate authority is WIFI_AUTHENTICATION, variable name
will be `$FLEET_VAR_DIGICERT_PASSWORD_WIFI_AUTHENTICATION` and
`FLEET_VAR_DIGICERT_DATA_WIFI_AUTHENTICATION`).

When sending the profile to hosts, Fleet will replace the variables variables with the proper values. Any errors will appear as a **Failed** status on the host details page, in **OS settings**.

When the profile with the DigiCert certificate is resent on the host details page in **OS settings**, Fleet will get a new certificate from DigiCert and create a new seat, which will take 1 license. If you want to revoke a license used by a seat that was created when the initial certificate was issued, go to [Trust Lifcycle Manager > Account > Seats](https://demo.one.digicert.com/mpki/account/seats) and remove the respective seat.

#### Example configuration profile

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>PayloadContent</key>
        <array>
            <dict>
                <key>Password</key>
                <string>$FLEET_VAR_DIGICERT_PASSWORD_CA_NAME</string>
                <key>PayloadContent</key>
                <data>$FLEET_VAR_DIGICERT_DATA_CA_NAME</data>
                <key>PayloadDisplayName</key>
                <string>CertificatePKCS12</string>
                <key>PayloadIdentifier</key>
                <string>com.fleetdm.pkcs12</string>
                <key>PayloadType</key>
                <string>com.apple.security.pkcs12</string>
                <key>PayloadUUID</key>
                <string>ee86cfcb-2409-42c2-9394-1f8113412e04</string>
                <key>PayloadVersion</key>
                <integer>1</integer>
            </dict>
        </array>
        <key>PayloadDisplayName</key>
        <string>DigiCert profile</string>
        <key>PayloadIdentifier</key>
        <string>TopPayloadIdentifier</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>TopPayloadUUID</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
    </dict>
</plist>
```

## Microsoft NDES

To install certificates from Microsoft NDES to hosts, do the following steps:

- [Connect Fleet to NDES](#step-1-connect-fleet-to-ndes)
- [Add SCEP configuration profile to Fleet](#step-2-add-scep-configuration-profile-to-fleet)

### Step 1: Connect Fleet to NDES

1. Go to the Fleet, navigate to **Settings**, select **Integrations** tab, and select **Certificates**.
2. Select **Add CA** button, and select **Microsoft NDES** from the dropdown on the top.
3. Add **SCEP URL** that accepts the SCEP protocol.
4. Add **Admin URL** and associated **Username** and **Password** to get the one-time challenge
   password for SCEP enrollment.
5. Select **Add CA**, and your NDES certificate authority (CA) should appear in the list.


Note:
* The example paths end with `/certsrv/mscep/mscep.dll` and `/certsrv/mscep_admin/` respectively. These path suffixes are the default paths for NDES on Windows Server 2022 and should only be changed if you have customized the paths on your server.
* When saving the configuration, Fleet will attempt to connect to the SCEP server to verify the connection, including retrieving a one-time challenge password. This validation also occurs when adding a new SCEP configuration or updating an existing one via API and GitOps, including dry runs. Please ensure the NDES password cache size is large enough to accommodate this validation.

### Step 2: Add SCEP configuration profile to Fleet

[Add a configuration profile](https://fleetdm.com/guides/custom-os-settings) in Fleet that includes the SCEP payload. In the profile, you will need to set `$FLEET_VAR_NDES_SCEP_CHALLENGE` as the `Challenge` and `$FLEET_VAR_NDES_SCEP_PROXY_URL` as the `URL`.

Adjust the `Subject` values according to your organization's needs. You may set `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` if the hosts were enrolled into Fleet MDM using an IdP (Identity Provider). You can also use any of the [Apple profile variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) to uniquely identify your device.

When sending the profile to hosts, Fleet will replace the variables variables with the proper
values. Any errors will appear as a **Failed** status on the host details page, in **OS settings**.

![NDES SCEP failed profile](../website/assets/images/articles/ndes-scep-failed-profile.png)

#### Example configuration profile

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
             <string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string>
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
                            <string>%SerialNumber% WIFI $FLEET_VAR_HOST_END_USER_EMAIL_IDP</string>
                          </array>
                        </array>
                        <array>
                          <array>
                            <string>OU</string>
                            <string>FLEET DEVICE MANAGEMENT</string>
                          </array>
                        </array>
                    </array>
             <key>URL</key>
             <string>$FLEET_VAR_NDES_SCEP_PROXY_URL</string>
          </dict>
          <key>PayloadDisplayName</key>
          <string>WIFI SCEP</string>
          <key>PayloadIdentifier</key>
          <string>com.apple.security.scep.9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
          <key>PayloadType</key>
          <string>com.apple.security.scep</string>
          <key>PayloadUUID</key>
          <string>9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
          <key>PayloadVersion</key>
          <integer>1</integer>
       </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>SCEP proxy cert</string>
    <key>PayloadIdentifier</key>
    <string>Fleet.WiFi</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4CD1BD65-1D2C-4E9E-9E18-9BCD400CDEDC</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

## Custom SCEP server

To install certificates from Microsoft NDES to hosts, do the following steps:

- [Connect Fleet to custom SCEP server](#step-1-connect-fleet-to-custom-scep-server)
- [Add SCEP configuration profile to Fleet](#step-2-add-scep-configuration-profile-to-fleet2)

### Step 1: Connect Fleet to custom SCEP server

1. Go to the Fleet, navigate to **Settings**, select **Integrations** tab, and select **Certificates**.
2. Select **Add CA** button, and select **Custom** from the dropdown on the top.
3. Add **Name** for your certificate authority. It's best to use all caps, beacuse it will be used
   as variable name in configuration profile and name it based on your use case (e.g.
   WIFI_AUTHENTICATION).
4. Add **SCEP URL** that accepts the SCEP protocol.
5. Add **Challenge** password to authenticate Fleet with your SCEP server.
6. Select **Add CA**, and your custom SCEP certificate authority (CA) should appear in the list.

### Step 2: Add SCEP configuration profile to Fleet

[Add a configuration profile](https://fleetdm.com/guides/custom-os-settings) in Fleet that includes
the SCEP payload. In the profile, you will need to set `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>`
as the `Challenge` and `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>` as the `URL`.

Replace `<CA_NAME>` part of the variable, with name that you used in the section above, to connect
Fleet to DigiCert (e.g if name of the certificate authority is WIFI_AUTHENTICATION, variable name
will be `$FLEET_VAR_DIGICERT_PASSWORD_WIFI_AUTHENTICATION` and
`FLEET_VAR_DIGICERT_DATA_WIFI_AUTHENTICATION`).

Adjust the `Subject` values according to your organization's needs. You may set `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` if the hosts were enrolled into Fleet MDM using an IdP (Identity Provider). You can also use any of the [Apple profile variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0) to uniquely identify your device.

When sending the profile to hosts, Fleet will replace the variables variables with the proper
values. Any errors will appear as a **Failed** status on the host details page, in **OS settings**.

#### Example configuration profile

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
             <string>$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CA_NAME</string>
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
                            <string>%SerialNumber% WIFI $FLEET_VAR_HOST_END_USER_EMAIL_IDP</string>
                          </array>
                        </array>
                        <array>
                          <array>
                            <string>OU</string>
                            <string>FLEET DEVICE MANAGEMENT</string>
                          </array>
                        </array>
                    </array>
             <key>URL</key>
             <string>$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CA_NAME</string>
          </dict>
          <key>PayloadDisplayName</key>
          <string>WIFI SCEP</string>
          <key>PayloadIdentifier</key>
          <string>com.apple.security.scep.9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
          <key>PayloadType</key>
          <string>com.apple.security.scep</string>
          <key>PayloadUUID</key>
          <string>9DCC35A5-72F9-42B7-9A98-7AD9A9CCA3AC</string>
          <key>PayloadVersion</key>
          <integer>1</integer>
       </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>SCEP proxy cert</string>
    <key>PayloadIdentifier</key>
    <string>Fleet.WiFi</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4CD1BD65-1D2C-4E9E-9E18-9BCD400CDEDC</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

## How the SCEP proxy works

Fleet acts as a middleman between the host and the NDES or custom SCEP server. When a host requests a certificate from Fleet, Fleet requests a certificate from the NDES or
custom SCEP server, retrieves the certificate, and sends it back to the host. 

In addition, Fleet does the following:
SCEP proxy:

- Retrieves the one-time challenge password from NDES. The NDES admin password is encrypted in Fleet's database by the [server private key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key). It cannot be retrieved via the API or the web interface. Retrieving passwords for many hosts at once may cause a bottleneck. To avoid long wait times, we recommend a gradual rollout of SCEP profiles.
  - Restarting NDES will clear the password cache and may cause outstanding SCEP profiles to fail.
- Resends the configuration profile to the host if the one-time challenge password has expired.
  - If the host has been offline and the one-time challenge password is more than 60 minutes old, Fleet assumes the password has expired and will resend the profile to the host with a new one-time challenge password.

Certificates will appear in the System Keychain on macOS. During the profile installation,
the OS generates several temporary certificates needed for the SCEP protocol. These certificates may be briefly visible in the Keychain Access app on macOS. The CA certificate must also be installed and marked as trusted on the device for the issued certificate to appear as trusted. The IT admin can send the CA certificate in a separate [CertificateRoot profile](https://developer.apple.com/documentation/devicemanagement/certificateroot?language=objc)

## Assumptions and limitations
* NDES SCEP proxy is currently supported for macOS devices via Apple config profiles. Support for DDM (Declarative Device Management) is coming soon, as is support for iOS, iPadOS, Windows, and Linux.
* Certificate renewal is coming soon.
* Fleet server assumes a one-time challenge password expiration time of 60 minutes.

<meta name="articleTitle" value="Connect end users to Wi-Fi with a certificate (from DigiCert, NDES, or custom SCEP)">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-10-30">
<meta name="description" value="Learn how to automatically connect device to a Wi-Fi by adding your certificate authority and issuing certificate from it.">