# Configure EAP-TLS Wi-Fi on Android

_Available in Fleet Premium_

This guide walks through configuring enterprise Wi-Fi network (802.1X) with EAP-TLS method on Android hosts. Supported on fully-managed and work profile (BYOD) hosts.

Follow steps below to connect your Android hosts to enterprise Wi-Fi:

1. [Add SCEP certificate authority](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-scep-simple-certificate-enrollment-protocol-ca) to Fleet
2. [Deployed SCEP certificate](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate) to Android hosts.
3. [Add Wi-Fi configuration profile](#add-a-wi-fi-configuration-profile) to Fleet.

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
          "AutoConnect": false,
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
| `GUID` | Unique identifier for the each network. Use a different GUID for each network if you have multiple networks under `NetworkConfigurations`, or multiple configuration profiles with `openNetworkConfiguration` setting. |
| `AutoConnect` | Determines if the network is automatically connected. This setting is independent of the auto-connect option per network available to end users in the host's Wi-Fi settings. |
| `Identity` | It's usually user's identifier like email. |
| `DomainSuffixMatch` | Domain suffix used to verify the RADIUS server's identity. The host checks that the server certificate's SAN DNS name (or CN if no SAN is present) ends with this suffix. |
| `ClientCertKeyPairAlias` | Name of the certificate you added in Fleet under **Controls > OS settings > Certificates**. |
| `X509` | Base64-encoded content of the root CA certificate that signed both server and client certificates. Exclude header and footer (`-----BEGIN CERTIFICATE-----` and `-----END CERTIFICATE-----`). |

## See status

To check the status, go to the host and select **OS settings** in Fleet.

If the profile shows `"openNetworkConfiguration" setting couldn't apply to a host. Reason: INVALID_VALUE.` error, the certificate specified in `ClientCertKeyPairAlias` isn't available on the host. Verify the name matches the certificate in **Controls > OS settings > Certificates** and that the certificate deployed successfully.

If a Wi‑Fi configuration profile is deployed before the certificate is installed on the host, it will fail with the same error. This will be fixed in [#42405](https://github.com/fleetdm/fleet/issues/42405). In the mean time, delete and re-add the Wi‑Fi profile after the certificate is installed.

## End user experience

The network is saved but the end user must select it once in Wi-Fi settings. After that, the device reconnects automatically.

To skip the manual step, set `AutoConnect` to `true` in the profile — the device will connect automatically without end user action, unless end user disables auto-connect for this network.

<meta name="articleTitle" value="Configure EAP-TLS Wi-Fi on Android">
<meta name="authorFullName" value="Marko Lisica">
<meta name="authorGitHubUsername" value="marko-lisica">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-24">
<meta name="description" value="Learn how to configure enterprise Wi-Fi network (802.1X) with EAP-TLS method on Android hosts in Fleet.">
