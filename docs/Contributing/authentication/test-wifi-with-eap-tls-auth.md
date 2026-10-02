# Setting up WPA/WPA2/WPA3 Enterprise with EAP TLS authentication for testing

Requirements:
- A router or access point that supports WPA Enterprise with EAP TLS authentication.
- [FreeRADIUS server](https://www.freeradius.org/)
- [micromdm/scep](https://github.com/micromdm/scep) SCEP server

> Instructions cover running FreeRADIUS and micromdm/scep on macOS

These instructions cover end to end setup of WPA Enterprise with EAP TLS and deployment of client certificates with Fleet on Android hosts. FreeRADIUS server comes with built-in certificates so if you don't want to add micromdm/scep CA to Fleet and deploy client certificates, you can skip that step.

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

## 9. Certificate deployment and Wi-Fi configuration on Android

Follow [these instructions](https://fleetdm.com/guides/configure-eap-tls-wifi-android)