# Protocol

This sequence diagram outlines the manual MDM enrollment process.

```mermaid
sequenceDiagram
    participant windows as Windows
    participant orbit as Orbit
    participant server as fleet server

    orbit->>+server: POST /api/fleet/orbit/enroll<br/>enroll_secret, hardware_uuid, etc.
    server-->>-orbit: orbit_node_key

    loop every 30 seconds
        orbit->>+server: POST /api/fleet/orbit/config<br/>orbit_node_key
        server-->>-orbit: pending notifications
    end

    note over orbit: Receive enrollment notification<br/>needs_programmatic_windows_mdm_enrollment<br/>windows_mdm_discovery_endpoint
    orbit->>windows: mdmregistration.dll<br/>RegisterDeviceWithManagement<br/>discovery endpoint, node key

    windows->>+server: POST /api/mdm/microsoft/discovery
    server-->>-windows: EnrollmentServiceURL, EnrollmentPolicyServiceUrl

    windows->>+server: POST /api/mdm/microsoft/policy<br/>DeviceEnrollmentUserToken (node key)
    server-->>-windows: Policy Schema, Certificate requirements
    activate windows
    note left of windows: Generate keypair
    deactivate windows
    windows->>+server: POST /api/mdm/microsoft/enroll<br/>Self-signed CSR & cert values, DeviceID<br/>DeviceEnrollmentUserToken (node key)
    note right of server: Creates certificate signed by WSTEP ident key
    server-->>-windows: Signed certificate, management endpoint, enrollment parameters

    loop SYNCML MDM Protocol (mTLS)
        windows->>+server: POST /api/mdm/microsoft/management<br/>DeviceID
        server-->>-windows: Response
    end
```

# Glossary

## WSTEP

[WSTEP](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-wstep/ac55b8cc-9ade-4982-b135-991d574ade74) is the protocol Microsoft uses to automate certificate requesting and singing. It is similar to the SCEP process used by macOS.

The certificate created through the WSTEP process is used to authenticate mTLS between the host and management endpoint.

## SyncML

[SyncML](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-wstep/ac55b8cc-9ade-4982-b135-991d574ade74) is an XML dialect used by Microsoft for Device Management.

## mTLS

[Mutual Transport Layer Security](https://www.cloudflare.com/learning/access-management/what-is-mutual-tls/) is a method for securing communications between two parties, in which both parties present signed certificates. This is different from standard TLS, where only the most provides a certificate. This allows both parties to authenticate the other's identity.

## MDM Protocol Summary

https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f



<meta name="pageOrderInSection" value="2900">