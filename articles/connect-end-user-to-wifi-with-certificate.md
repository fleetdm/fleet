# Connect end users to Wi-Fi or VPN with a certificate (DigiCert, NDES, or custom SCEP)

_Available in Fleet Premium_

Fleet can help your end users connect to Wi-Fi or VPN by deploying certificates from your certificate authority (CA). Fleet currently supports [DigiCert](https://www.digicert.com/digicert-one), [Microsoft NDES](https://learn.microsoft.com/en-us/windows-server/identity/ad-cs/network-device-enrollment-service-overview), and custom [SCEP](https://en.wikipedia.org/wiki/Simple_Certificate_Enrollment_Protocol) server.

Fleet will automatically renew certificates 30 days before expiration. If an end user is on vacation (offline more than 30 days), their certificate might expire and they'll lose access to Wi-Fi or VPN. To get them reconnected, ask your end users to momentarily connect to a different network so that Fleet can deliver a new certificate.

> For information on adding a certificate authority (CA) via GitOps, see the [GitOps documentation](https://fleetdm.com/docs/configuration/yaml-files#integrations).

## DigiCert

To connect end users to W-Fi or VPN with DigiCert certificates, we'll do the following steps:

- [Create service user in DigiCert](#step-1-create-service-user-in-digicert)
- [Create certificate profile in DigiCert](#step-2-create-certificate-profile-in-digicert)
- [Connect Fleet to DigiCert](#step-3-connect-fleet-to-digicert)
- [Add PKCS #12 configuration profile to Fleet](#step-4-add-pkcs-12-configuration-profile-to-fleet)

### Step 1: Create service user in DigiCert

1. Head to [DigiCert One](https://one.digicert.com/)
2. Follow [DigiCert's instructions for creating a service user](https://docs.digicert.com/en/platform-overview/manage-your-accounts/account-manager/users-and-access/service-users/create-a-service-user.html) and save the service user's API token.
> Make sure to assign **User and certificate manager** and **Certificate profile manager** roles
> when creating service user.

### Step 2: Create certificate profile in DigiCert

1. In DigiCert [Trust Lifcycle Manager](https://one.digicert.com/mpki/dashboard), select **Policies > Certificate profiles** from the main menu. Then select **Create profile from template** and select **Generic Device Certificate** from the list.
2. Add a friendly **Profile name** (e.g. "Fleet - Wi-Fi authentication").
3. Select your **Business unit** and **Issuing CA**.
4. Select **REST API** from **Enrollment method**. Then select **3rd party app** from the **Authentication method** dropdown and select **Next**.
5. Configure the certificate expiration. At most organizations, this is set to 90 days.
6. In the **Flow options** section, make sure that **Allow dupliate certificates** is checked.
7. In the **Subject DN and SAN fields** section, make sure to add **Common name**. **Other name (UPN)** is optional. For **Common name**, select **REST request** from **Source for the field's value** dropdown and check **Required**. If you use **Other name (UPN)**, select **REST Request** and check both **Required** and **Multiple**. Organizations usually use device's serial number or user's email, you can use Fleet variables in the next section, and Fleet will replace these variables with the actual values before certificate is delivered to a device.
8. Click **Next** and leave all default options. We'll come back to this later.

### Step 3: Connect Fleet to DigiCert

1. In Fleet, head to **Settings > Integrations > Certificates**.
2. Select **Add CA** and then choose **DigiCert** in the dropdown.
3. Add a **Name** for your certificate authority. The best practice is to create a name based on your use case in all caps snake case (ex. "WIFI_AUTHENTICATION"). We'll use this name later as variable name in a configuration profile.
4. If you're using DigiCert One's cloud offering, keep the default **URL**. If you're using a self-hosted (on-prem) DigiCert One, update the URL to match the one you use to login to your DigiCert One.
5. In **API token**, paste your DigiCert server user's API token (from step 1).
6. In **Profile GUID**, paste your DigiCert One certificate profile GUID (from step 2). To get your profile GUID, in DigiCert, head to the [Certificate profiles](https://one.digicert.com/mpki/policies/profiles) page, open your profile, and copy **GUID**.
7. In **CN**, **UPN**, and **Certificate seat ID**, you can use fixed values or one of the [Fleet's host variables](https://fleetdm.com/docs/configuration/yaml-files#macos-settings-and-windows-settings). Organizations usually use the host's serial number or end user's email to deliver a certificate that's unique to the host.
8. Select **Add CA**. Your DigiCert certificate authority (CA) should appear in your list of CAs in Fleet.

### Step 4: Add PKCS12 configuration profile to Fleet

1. Create a [configuration profile](https://fleetdm.com/guides/custom-os-settings) with a PKCS12 payload. In the profile, for `Password`, use `$FLEET_VAR_DIGICERT_PASSWORD_<CA_NAME>`. For `Data`, use `$FLEET_VAR_DIGICERT_DATA_<CA_NAME>`.

2. Replace the `<CA_NAME>`, with name you created in step 3. For example, if the name of the CA is "WIFI_AUTHENTICATION" the variables will look like this: `$FLEET_VAR_DIGICERT_PASSWORD_WIFI_AUTHENTICATION` and `$FLEET_VAR_DIGICERT_DATA_WIFI_AUTHENTICATION`.

3. In Fleet, head to **Controls > OS settings > Custom settings** and add the configuration profile to deploy certificates to your hosts.

When Fleet delivers the profile to your hosts, Fleet will replace the variables. If something goes wrong, errors will appear on each host's **Host details > OS settings**.

More DigiCert details:
- Each DigiCert device type seat (license) can have multiple certificates only if they have the same CN and seat ID. If a new certificate has a different CN, a new DigiCert license is required.
- If the value for any variable used in step 3 above changes, Fleet will resend the profile. This means, if you use a variable like `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` for CN or seat ID, and the variable's value changes, Fleet will get a new certificate and create a new seat in DigiCert. This will add a new DigiCert license. If you want to revoke a license in DigiCert, head to [**Trust Lifcycle Manager > Account > Seats**](https://demo.one.digicert.com/mpki/account/seats) and remove the seat.
- DigiCert seats aren't automatically revoked when hosts are deleted in Fleet. To revoke a license, ask the team that owns DigiCert to follow the instructions above.


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

To connect end users to W-Fi or VPN with Microsoft NDES certificates, we'll do the following steps:

- [Connect Fleet to NDES](#step-1-connect-fleet-to-ndes)
- [Add SCEP configuration profile to Fleet](#step-2-add-scep-configuration-profile-to-fleet)

### Step 1: Connect Fleet to NDES

1. In Fleet, head to **Settings > **Integrations > Certificates**.
2. Select the **Add CA** button and select **Microsoft NDES** in the dropdown.
3. Add your **SCEP URL**, **Admin URL**, and **Username** and **Password**.
5. Select **Add CA**. Your NDES certificate authority (CA) should appear in the list in Fleet.
The example paths end with `/certsrv/mscep/mscep.dll` and `/certsrv/mscep_admin/` respectively. These path suffixes are the default paths for NDES on Windows Server 2022 and should only be changed if you have customized the paths on your server.

When saving the configuration, Fleet will attempt to connect to the SCEP server to verify the connection, including retrieving a one-time challenge password. This validation also occurs when adding a new SCEP configuration or updating an existing one via API and GitOps, including dry runs. Please ensure the NDES password cache size is large enough to accommodate this validation.

### Step 2: Add SCEP configuration profile to Fleet

1. Create a [configuration profile](https://fleetdm.com/guides/custom-os-settings) with the SCEP payload. In the profile, for `Challenge`, use`$FLEET_VAR_NDES_SCEP_CHALLENGE`. For `URL`, use `$FLEET_VAR_NDES_SCEP_PROXY_URL`, and make sure to add `$FLEET_VAR_SCEP_RENEWAL_ID` to `CN`.


2. If your Wi-Fi or VPN requires certificates that are unique to each host, update the `Subject`. You can use `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` if your hosts automatically enrolled (via ADE) to Fleet with [end user authentication](https://fleetdm.com/docs/rest-api/rest-api#get-human-device-mapping) enabled. You can also use any of the [Apple's built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0).

3. In Fleet, head to **Controls > OS settings > Custom settings** and add the configuration profile to deploy certificates to your hosts.

When Fleet delivers the profile to your hosts, Fleet will replace the variables. If something goes wrong, errors will appear on each host's **Host details > OS settings**.

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
                            <string>%SerialNumber% $WIFI $FLEET_VAR_SCEP_RENEWAL_ID</string>
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

To connect end users to W-Fi or VPN with a custom SCEP server, we'll do the following steps:

- [Connect Fleet to custom SCEP server](#step-1-connect-fleet-to-custom-scep-server)
- [Add SCEP configuration profile to Fleet](#step-2-add-scep-configuration-profile-to-fleet2)

### Step 1: Connect Fleet to custom SCEP server

1. In Fleet, head to **Settings > **Integrations > Certificates**.
2. Select the **Add CA** button and select **Custom** in the dropdown.
3. Add a **Name** for your certificate authority. The best practice is to create a name based on your use case in all caps snake case (ex. "WIFI_AUTHENTICATION"). We'll use this name later as variable name in a configuration profile.
4. Add your **SCEP URL** and **Challenge**.
6. Select **Add CA**.  Your custom SCEP certificate authority (CA) should appear in the list in Fleet.

### Step 2: Add SCEP configuration profile to Fleet

1. Create a [configuration profile](https://fleetdm.com/guides/custom-os-settings) with the SCEP payload. In the profile, for `Challenge`, use`$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>`. For, `URL`, use `$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>`, and make sure to add `$FLEET_VAR_SCEP_RENEWAL_ID` to `CN`.


2. Replace the `<CA_NAME>`, with name you created in step 3. For example, if the name of the CA is "WIFI_AUTHENTICATION" the variables will look like this: `$FLEET_VAR_CUSTOM_SCEP_PASSWORD_WIFI_AUTHENTICATION` and `FLEET_VAR_CUSTOM_SCEP_DIGICERT_DATA_WIFI_AUTHENTICATION`.

3. If your Wi-Fi or VPN requires certificates that are unique to each host, update the `Subject`. You can use `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` if your hosts automatically enrolled (via ADE) to Fleet with [end user authentication]((https://fleetdm.com/docs/rest-api/rest-api#get-human-device-mapping)) enabled. You can also use any of the [Apple's built-in variables](https://support.apple.com/en-my/guide/deployment/dep04666af94/1/web/1.0).

4. In Fleet, head to **Controls > OS settings > Custom settings** and add the configuration profile to deploy certificates to your hosts.

When Fleet delivers the profile to your hosts, Fleet will replace the variables. If something goes wrong, errors will appear on each host's **Host details > OS settings**.

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
                            <string>%SerialNumber% WIFI $FLEET_VAR_SCEP_RENEWAL_ID</string>
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

Fleet acts as a middleman between the host and the NDES or custom SCEP server. When a host requests a certificate from Fleet, Fleet requests a certificate from the NDES or custom SCEP server, retrieves the certificate, and sends it back to the host.

Certificates will appear in the System Keychain on macOS. During the profile installation, the OS generates several temporary certificates needed for the SCEP protocol. These certificates may be briefly visible in the Keychain Access app on macOS. The CA certificate must also be installed and marked as trusted on the device for the issued certificate to appear as trusted. The IT admin can send the CA certificate in a separate [CertificateRoot profile](https://developer.apple.com/documentation/devicemanagement/certificateroot?language=objc)

In addition, Fleet does the following:

NDES SCEP proxy:

- Retrieves the one-time challenge password from NDES. The NDES admin password is encrypted in Fleet's database by the [server private key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key). It cannot be retrieved via the API or the web interface. Retrieving passwords for many hosts at once may cause a bottleneck. To avoid long wait times, we recommend a gradual rollout of SCEP profiles.
  - Restarting NDES will clear the password cache and may cause outstanding SCEP profiles to fail.
- Resends the configuration profile to the host if the one-time challenge password has expired.
  - If the host has been offline and the one-time challenge password is more than 60 minutes old, Fleet assumes the password has expired and will resend the profile to the host with a new one-time challenge password.

Custom SCEP proxy:

- Generates a one-time passcode that is added to the URL in the SCEP profile.
  - When a host makes a certificate request via the URL, the passcode is validated by Fleet prior to retrieving a certificate from the custom SCEP server.
  - This Fleet-managed passcode is valid for 60 minutes. Fleet automatically resends the SCEP profile
    to the host with a new passcode if the host requests a certificate after the passcode has expired.
  - The static challenge configured for the custom SCEP server remains in the SCEP profile.



## Assumptions and limitations

* NDES SCEP proxy is currently supported for macOS devices via Apple config profiles. Support for DDM (Declarative Device Management) is coming soon, as is support for iOS, iPadOS, Windows, and Linux.
* Fleet server assumes a one-time challenge password expiration time of 60 minutes.

## How to deploy certificates to a user's login keychain

You can also upload a certificate to be installed in the login keychain of the currently logged-in user on a macOS host using a user-scoped configuration profile.

1. **Add your CA as before**
  Use the above steps to add integrate your CA with Fleet.
1. **Create a certificate payload**
  Use your preferred tool (e.g., Apple Configurator or a `.mobileconfig` generator) to create a configuration profile that includes your certificate.
2. **Ensure the payload is scoped to the user**
  In the payload, set the `PayloadScope` to `User`. This tells macOS to install the certificate in the user’s login keychain instead of the system keychain.
3. **Upload the configuration profile to Fleet**
  Navigate to **Controls > OS settings > Custom settings** in the Fleet UI. Upload the `.mobileconfig` profile you created.
4. **Assign the profile to the correct hosts**
  Use Fleet’s targeting filters to assign the profile to the appropriate hosts. The certificate will be installed in the login keychain of the user currently logged in on each device.

<meta name="articleTitle" value="Connect end users to Wi-Fi or VPN with a certificate (DigiCert, NDES, or custom SCEP)">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-10-30">
<meta name="description" value="Learn how to automatically connect device to a Wi-Fi by adding your certificate authority and issuing certificate from it.">
