# Setting up WPA/WPA2/WPA3 Enterprise with EAP TLS authentication for testing

Requirements:
- A router or access point that supports WPA Enterprise with EAP TLS authentication.
- [FreeRADIUS server](https://www.freeradius.org/)
- [micromdm/scep](https://github.com/micromdm/scep) SCEP server

> Instructions are for macOS

These instructions cover end to end setup of WPA Enterprise with EAP TLS and deployment of client certificates with Fleet. FreeRADIUS server comes with built-in certificates so if you don't want to add micromdm/scep CA to Fleet and deploy client certificates, you can skip that step.

## 1. Install FreeRADIUS

```bash
brew install freeradius-server
```

Config directory: `/opt/homebrew/etc/raddb/` (Apple Silicon) or `/usr/local/etc/raddb/` (Intel).

## 2. Set up micromdm/scep as certificate authority (CA)

### Install

```bash
brew install scep
```

Or with Go:

```bash
go install github.com/micromdm/scep/v2/cmd/scepserver@latest
```

### Initialize the certificate authority

Assuming you downloaded the SCEP server source code to `~/scep`.

```bash
cd ~/scep
scepserver ca -init
```

This creates:
- `scep/depot/ca.pem` — CA root certificate
- `scep/depot/ca.key` — CA private key

### Start the SCEP server (for client enrollment)

```bash
scepserver -depot depot -port 8080 -challenge=secret
```

## 3. Generate FreeRADIUS server certificate

```bash
cd ~/scep

# Generate server private key
openssl genrsa -out server.key 2048

# Create CSR
openssl req -new -key server.key \
  -out server.csr \
  -subj "/CN=radius.local/O=MyNetwork"

# Sign the server certificate with the SCEP CA root certificate
openssl x509 -req -in server.csr \
  -CA ~/scep/depot/ca.pem \
  -CAkey ~/scep/depot/ca.key \
  -CAcreateserial \
  -out server.pem \
  -days 365 -sha256
```

## 4. Copy certs to FreeRADIUS

Run these commands to copy the SCEP CA root cert and the server cert/key to FreeRADIUS's cert directory.

```bash
cp ~/scep/depot/ca.pem  /opt/homebrew/etc/raddb/certs/ca.pem
cp ~/scep/server.pem    /opt/homebrew/etc/raddb/certs/server.pem
cp ~/scep/server.key    /opt/homebrew/etc/raddb/certs/server.key
```
This overwrites the default self-signed certs that ship with FreeRADIUS.

## 5. Configure FreeRADIUS EAP module

Go to `/opt/homebrew/etc/raddb/mods-available/eap` and edit the file:

On the top of the document find `default_eap_type` and make sure that it's set to `tls`.

```
eap {
  ...
  default_eap_type = tls 
}
```

In the `tls-config tls-common` section, update the private key settings. Change `private_key_file` from `server.pem` to `server.key` (key and cert are separate files). Comment out `private_key_password` (SCEP-issued key is not password-protected - by default)

```
#   private_key_password =
    private_key_file = ${certdir}/server.key
```

## 6. Configure RADIUS client (router or AP)

Edit `/opt/homebrew/etc/raddb/clients.conf`, add at the bottom:

```
client my-router {
    ipaddr = <router_ip>
    secret = <random_secret>
    require_message_authenticator = no
    nas_type = other
}
```

Replace `<router_ip>` with the IP address of your router/AP, and `<random_secret>` with a random secret string. You will need to provide secret in router settings when connecting RADIUS server.

## 7. Configure the router

Depending on the router model, the configuration steps may vary. Here are general instructions for common models.

1. Open router admin panel (usuallyat `192.168.1.1` or similar)
2. Log in
3. Find WLAN configuration
4. Createa new SSID
5. Set:
   - **Security Mode**: `WPA2-Enterprise` or `WPA3-Enterprise`
   - **Encryption Mode**: `AES`
   - **RADIUS Server IP**: your Mac's IP. Run `ipconfig getifaddr en0` to get IP.
   - **RADIUS Port**: `1812`
   - **RADIUS Shared Secret**: same secret from `clients.conf` above
6. Save

## 8. Start FreeRADIUS server

```bash
radiusd -X
```

Runs in foreground with debug output. Look for `Ready to process requests`.

To stop server use: `Ctrl+C`.

## 9. Certificate deployment on Android via Fleet

Follow [these instructions](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#android-deploy-certificate) to deploy the certificate to Android devices via Fleet.

In addition to certificate being installed, it's necessary to add a configuration profile (`openNetworkConfiguration`) to set up WiFi on the device.

Certificates are visible in **Settings** under **Security**, but when adding new WiFi network manually, those certificates won't be visible. Wi-Fi Settings only shows certificates installed to the Wi-Fi credential store. Your SCEP certificate is installed "for VPN and
apps" (the default store when installed via `DevicePolicyManager.installKeyPair()` - method Fleet's agent uses). That's why it doesn't appear in the Wi-Fi settings in client certificate dropdown.

Open Network Configuration (ONC) via Android Management API (AMAPI) uses a different code path. On Android 12+, when Android Device Policy (built in DPC) processes `ClientCertKeyPairAlias`, it calls `DevicePolicyManager.grantKeyPairToWifiAuth()` programmatically to grant the key pair to the Wi-Fi subsystem. This bypasses the credential store separation — the key pair doesn't need to be in the "Wi-Fi" store, it just needs the system-level grant.

So:
- Manual UI certificate picker → reads from Wi-Fi credential store only → your cert won't show
- ONC policy with ClientCertKeyPairAlias → uses programmatic grant → works regardless of store

### WiFi configuration profile

In the JSON below, replace folowing fields:

- `SSID` - must match the router's SSID exactly (case-sensitive). This is how Android knows which network to connect to.
- `Name` — display label, can be anything you want. It's just for human readability in the policy.
- `GUID` — unique identifier for the ONC config entry, can be any string. It's used internally to reference the network configuration. Just make sure each network config has a different GUID if you have multiple ONC profiles.
- `Identity` — the username to use for EAP authentication. It's usually the user's email, but depends on what RADIUS server expects.
- `ClientCertKeyPairAlias` — replace `<fleet_certificate_name>` with the name of the certificate you added to Fleet.
- `<CN_of_RADIUS_server_certificate>` — the common name (CN) of the RADIUS server's certificate. This is used to verify the server's identity.
- `X509` — content of root CA certificate that signs both server and client certificates. (in our case root CA of micromdm/scep server).

```json
{
  "openNetworkConfiguration": {
    "Type": "UnencryptedConfiguration",
    "NetworkConfigurations": [
      {
        "GUID": "enterprise-wifi",
        "Name": "Enterprise",
        "Type": "WiFi",
        "WiFi": {
          "SSID": "Enterprise",
          "EAP": {
            "Outer": "EAP-TLS",
            "Identity": "$FLEET_VAR_HOST_END_USER_IDP_USERNAME",
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
        "X509": "<content_of_root_ca_certificate>"
      }
    ]
  }
}
```

## Technical details

grantKeyPairToWifiAuth(String alias) — Added in API level 31 (Android 12)

  This is a real method on DevicePolicyManager. The API diff for Android 12 confirms these related methods were added together:

  ┌────────────────────────────────────┬─────────────────────────────────────────────────────────────┐
  │               Method               │                           Purpose                           │
  ├────────────────────────────────────┼─────────────────────────────────────────────────────────────┤
  │ grantKeyPairToWifiAuth(String)     │ Grants a key pair to the Wi-Fi subsystem for authentication │
  ├────────────────────────────────────┼─────────────────────────────────────────────────────────────┤
  │ revokeKeyPairFromWifiAuth(String)  │ Revokes that grant                                          │
  ├────────────────────────────────────┼─────────────────────────────────────────────────────────────┤
  │ isKeyPairGrantedToWifiAuth(String) │ Checks if a key pair has been granted                       │
  └────────────────────────────────────┴─────────────────────────────────────────────────────────────┘

  How the two code paths work

  1. installKeyPair() installs a certificate + private key into the device's keystore, accessible to "VPN and apps." This does not make it
  visible to the Wi-Fi credential picker in Settings.
  2. grantKeyPairToWifiAuth(alias) takes a key pair already installed via installKeyPair() and explicitly grants it to the Wi-Fi subsystem.
  This is what CloudDPC calls when it processes the ClientCertKeyPairAlias field in an ONC policy — it bridges the gap between the app keystore
   and the Wi-Fi credential store.

  That's why certificates show up under Settings > Security but not in the Wi-Fi manual network setup dropdown. The manual UI only reads from
  the Wi-Fi credential store, while ONC policy bypasses that by using the programmatic grant.

  Sources

  - API diff showing grantKeyPairToWifiAuth added in API 31
  - DevicePolicyManager API reference
  - Security - DPC documentation (installKeyPair)
  - Secure Wi-Fi Enterprise configuration
  - AOSP source - DevicePolicyManager.java
