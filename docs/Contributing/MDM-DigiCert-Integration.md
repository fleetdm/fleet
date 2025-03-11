# DigiCert integration

## Set up dev environment

- Go to https://demo.one.digicert.com/.
- In Account > Access > Users > `<User name>` > API Tokens, create/retrieve the API token.
- In Trust Lifecycle > Policies > Certificate profiles, create/retrieve the GUID of the profile.

_Notes:_
- Profile enrollment method must be `REST API` and Authentication method must be `3rd Party App`
- To add User Principal Name to the issued certificate, the `Seat type` must be `User`, and `Subject Alternative Name (SAN) > Other name (UPN)` must be set to `REST request` as the source.

## Architecture diagrams

```mermaid
---
title: Enable DigiCert integration
---
sequenceDiagram
    actor admin as Admin
    participant fleet as Fleet server
    participant digicert as DigiCert
    admin->>+fleet: Save configs
    fleet->>fleet: Validate inputs
    fleet->>+digicert: Get profile by GUID
    digicert-->>-fleet: Profile (Active)
    fleet->>fleet: Encrypt API token
    fleet-->>-admin: Done
```

```mermaid
---
title: Deploy DigiCert certificate to Apple host
---
sequenceDiagram
    actor admin as Admin
    participant host as Host
    participant fleet as Fleet server
    participant digicert as DigiCert
    participant apple as Apple
    admin->>+fleet: Upload PKCS12 Apple configuration profile
    fleet->>fleet: Validate profile
    fleet-->>-admin: OK

    fleet--)+fleet: Process profiles every 30 seconds
    fleet->>fleet: Validate profile
    fleet->>fleet: Decrypt API token
    fleet->>+digicert: Get certificate
    digicert-->>-fleet: Certificate
    fleet->>fleet: Save NotValidAfter date
    fleet->>+apple: Push notification (APNS)
    apple-->>-fleet: OK
    deactivate fleet

    host--)+fleet: Idle message
    fleet-->>-host: PKCS12 profile (DigiCert certificate)
    activate host
    host-->>-fleet: Acknowledged message
    activate fleet
    fleet-->>-host: Empty

    host->>+fleet: Read
    fleet-->>-host: Get profiles command (once an hour)
    
    host->>+fleet: Write (profiles)
    fleet->>fleet: PKCS12 profile Verified
    fleet-->>-host: OK
```

## Sample PKCS12 profile

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>PayloadContent</key>
        <array>
            <dict>
                <key>Password</key>
                <string>$FLEET_VAR_DIGICERT_PASSWORD_Test_CA</string>
                <key>PayloadContent</key>
                <data>${FLEET_VAR_DIGICERT_DATA_Test_CA}</data>
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
