# Configure EAP-TLS Wi-Fi on Android

_Available in Fleet Premium_

This guide walks through configuring WPA/WPA2/WPA3 Enterprise Wi-Fi with EAP-TLS authentication on Android devices using Fleet.

Follow steps below to connect your Android hosts to enterprise Wi-Fi networks.

1. [Add SCEP certificate authority](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-scep-simple-certificate-enrollment-protocol-ca) to Fleet
2. [Deployed SCEP certificate](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate) to Android hosts.
3. Add Wi-Fi configuration profile

## Add a Wi-Fi configuration profile

1. Create a JSON file (e.g., `wifi-eap-tls.json`) with the following content, replacing the placeholder values described below.

2. In Fleet, head to **Controls > OS settings > Custom settings**, select **Add profile**, and upload file below.

```json
{
  "openNetworkConfiguration": {
    "Type": "UnencryptedConfiguration",
    "NetworkConfigurations": [
      {
        "GUID": "enterprise-wifi",
        "Name": "Enterprise Wi-Fi",
        "Type": "WiFi",
        "WiFi": {
          "SSID": "<your_SSID>",
          "EAP": {
            "Outer": "EAP-TLS",
            "Identity": "name@example.com",
            "DomainSuffixMatch": ["<CN_of_RADIUS_server_certificate>"],
            "ClientCertType": "KeyPairAlias",
            "ClientCertKeyPairAlias": "<fleet_certificate_name>",
            "ServerCARefs": ["root_ca"]
          },
          "Security": "WPA-EAP"
        }
      }
    ],
    "Certificates": [
      {
        "GUID": "root_ca",
        "Type": "Authority",
        "X509": "<content_of_root_ca_certificate_without_header_and_footer>"
      }
    ]
  }
}
```

### Fields to replace

| Field | Description |
|---|---|
| `SSID` | Must match the router's SSID exactly (case-sensitive). |
| `Name` | Display label, can be anything. For human readability only. |
| `GUID` | Unique identifier for the ONC config entry. Use a different GUID for each network if you have multiple ONC profiles. |
| `Identity` | It's usually user's identifier like email. |
| `DomainSuffixMatch` | Common name (CN) of the RADIUS server's certificate, used to verify the server's identity. |
| `ClientCertKeyPairAlias` | Name of the certificate you added in Fleet under **Controls > OS settings > Certificates**. |
| `X509` | Base64-encoded content of the root CA certificate that signed both server and client certificates. |

## Verify

After the profile is deployed, the device should automatically connect to the configured Wi-Fi network. To check the status, go to the host's **Host details > OS settings** page in Fleet.

<meta name="articleTitle" value="Configure EAP-TLS Wi-Fi on Android">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-24">
<meta name="description" value="Learn how to configure WPA Enterprise Wi-Fi with EAP-TLS authentication on Android devices using Fleet.">
