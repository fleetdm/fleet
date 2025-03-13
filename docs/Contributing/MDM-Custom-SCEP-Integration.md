# Custom SCEP (Simple Certificate Enrollment Protocol) integration

## Set up dev environment

We will use a SCEP server from https://github.com/micromdm/scep (v2.3.0 as of this writing).

- Download the `scepserver` binary from Releases
- On macOS, remove it from quarantine: `xattr -d com.apple.quarantine ./scepserver-darwin-arm64`
- Initialize and launch the server per instructions on the GitHub page
- The SCEP URL will be like: http://localhost:2016/scep (with `/scep` suffix)

## Architecture diagrams

```mermaid
---
title: Add/edit custom SCEP integration
---
sequenceDiagram
    autonumber
    actor admin as Admin
    participant fleet as Fleet server
    participant scep as Custom SCEP server
    admin->>+fleet: Save configs
    fleet->>fleet: Validate inputs
    fleet->>+scep: GetCACert
    scep-->>-fleet: CA certificate
    fleet->>fleet: Encrypt SCEP challenge
    fleet-->>-admin: Done
```

```mermaid
---
title: Deploy custom SCEP certificate to Apple host
---
sequenceDiagram
    autonumber
    actor admin as Admin
    participant host as Host
    participant fleet as Fleet server
    participant scep as Custom SCEP server
    participant apple as Apple
    admin->>+fleet: Upload SCEP Apple configuration profile
    fleet->>fleet: Validate profile
    fleet-->>-admin: OK

    fleet--)+fleet: Process profiles every 30 seconds
    fleet->>fleet: Validate profile
    fleet->>fleet: Inject Fleet variables
    fleet->>+apple: Push notification (APNS)
    apple-->>-fleet: OK
    deactivate fleet

    host--)+fleet: Idle message
    fleet-->>-host: SCEP profile
    activate host
    host->>host: Generate private key
    host->>+fleet: SCEP: GetCACaps
    fleet->>+scep: SCEP: GetCACaps
    scep-->>-fleet: CA capabilities
    fleet-->>-host: CA capabilities
    host->>+fleet: SCEP: GetCACert
    fleet->>+scep: SCEP: GetCACert
    scep-->>-fleet: CA certificate
    fleet-->>-host: CA certificate
    host->>+fleet: SCEP: PKCSReq
    fleet->>+scep: SCEP: PKCSReq
    scep-->>-fleet: Encrypted certificate
    fleet-->>-host: Encrypted certificate
    host->>host: Add certificate to keychain
    host-->>-fleet: Acknowledged message
    activate fleet
    fleet-->>-host: Empty

    host->>+fleet: Read
    fleet-->>-host: Get profiles command (once an hour)
    
    host->>+fleet: Write (profiles)
    fleet->>fleet: SCEP profile Verified
    fleet-->>-host: OK
```

## Sample SCEP profile

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
             <string>$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_Test_SCEP</string>
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
                            <string>%SerialNumber% WIFI</string>
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
             <string>${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_Test_SCEP}</string>
          </dict>
          <key>PayloadDisplayName</key>
          <string>SCEP #1</string>
          <key>PayloadIdentifier</key>
          <string>com.fleetdm.custom.scep</string>
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
    <string>Fleet.custom.SCEP</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4CD1BD65-1D2C-4E9E-9E18-9BCD400CDEDC</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```
