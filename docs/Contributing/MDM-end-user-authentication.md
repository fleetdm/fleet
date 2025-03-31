# End user authentication

- [Fleet's guide for setting up end user authentication during macOS setup experience](https://fleetdm.com/guides/macos-setup-experience#end-user-authentication-and-end-user-license-agreement-eula)
- On Fleet's [pricing page](https://fleetdm.com/pricing), this feature is called `User account sync` (as of 2025/03/31)

## Notes

`end_user_authentication` setting is global, but `enable_end_user_authentication` is a team setting.

The Fleet SSO endpoint is `<fleet_url>/api/v1/fleet/mdm/sso`. It is set as `configuration_web_url` in Apple's [enrollment profile](https://developer.apple.com/documentation/devicemanagement/profile).

## Issues and limitations

- Fleet does not support OpenID Connect (OIDC) integration. Fleet only supports SAML.
- Fleet expects the SAML username to be an email.

## Diagrams

```mermaid
---
title: End user authentication flow during macOS setup experience
---
sequenceDiagram
    autonumber
    actor Admin
    actor User
    participant fleet as Fleet server
    participant host as macOS host
    participant Apple
    participant IdP
    Admin->>IdP: Enable SAML integration
    Admin->>fleet: Enable end user authentication
    Admin->>Apple: Add macOS to ADE
    fleet->>Apple: Update enrollment profile (DEP sync)
    User->>host: Turn on device
    host->>Apple: Start setup
    host->>fleet: Redirect to sso endpoint
    fleet->>IdP: SAML request (HTTP redirect binding)
    Note left of IdP: Service provider initiated SAML
    User->>IdP: Login to IdP
    IdP->>fleet: SAML response to sso/callback endpoint (Assertion Consumer Service)
    fleet->>fleet: Validate SSO session valid and not expired
    fleet->>fleet: Save username (expect to be email) and display name
    fleet->>fleet: Frontend redirect to enroll endpoint
    host->>fleet: TokenUpdate
    fleet->>Apple: notify
    Note right of fleet: MDM job runs every minute
    host->>fleet: Get AccountConfiguration command
    host->>fleet: osquery results (mdm server_url, etc.)
    Note right of fleet: Refetch runs every hour
    fleet->>fleet: Extract enroll ref from URL and save IdP email
```
