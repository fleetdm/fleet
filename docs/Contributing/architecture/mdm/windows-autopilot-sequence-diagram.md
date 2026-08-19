# Windows Autopilot enrollment with Fleet - sequence diagram

This diagram shows the end-to-end flow of a new Windows device going through Autopilot enrollment and becoming fully managed by Fleet.

```mermaid
sequenceDiagram
    autonumber
    participant OEM as OEM / Reseller
    participant MSAutopilot as Microsoft<br/>Autopilot Service
    participant Intune as Microsoft Intune
    participant Entra as Microsoft Entra ID
    participant MSjwks as Microsoft JWKS<br/>Endpoint
    participant Device as Windows Device
    participant Fleet as Fleet Server
    participant Fleetd as Fleetd Agent<br/>(on device)

    rect rgb(240, 248, 255)
        Note over OEM, MSAutopilot: Phase 1 — Device registration (before device ships)
        OEM->>MSAutopilot: Register hardware hash<br/>(SMBIOS: serial, UUID, TPM, SKU)
        Note right of OEM: OEM extracts hash during<br/>manufacturing, or IT admin<br/>uploads via Get-WindowsAutopilotInfo
    end

    rect rgb(255, 248, 240)
        Note over Intune, Fleet: Phase 2 — IT admin configuration (one-time setup)
        Note over Intune: Admin creates Autopilot<br/>deployment profile with<br/>group tag
        Intune->>MSAutopilot: Assign profile to device<br/>group (by group tag)
        Note over Entra: Admin registers Fleet as<br/>MDM app in Entra ID
        Entra-->>Fleet: Entra app registration points<br/>MDM enrollment URL to Fleet<br/>(Discovery endpoint)
        Note over Fleet: Admin configures Fleet with<br/>Entra tenant ID + client ID<br/>in MDM settings
    end

    rect rgb(240, 255, 240)
        Note over Device, Fleet: Phase 3 — Device OOBE and Autopilot
        Device->>Device: Power on, connect to internet
        Device->>MSAutopilot: "Who am I?" (sends hardware hash)
        MSAutopilot-->>Device: Autopilot profile<br/>(branding, Entra join config,<br/>MDM enrollment URL = Fleet)
        Device->>Device: Display branded OOBE
        Device->>Entra: User signs in / device joins Entra
        Entra-->>Device: Azure AD JWT token<br/>(contains: UPN, tenant ID,<br/>audience, scopes)
    end

    rect rgb(248, 240, 255)
        Note over Device, Fleet: Phase 4 — MDM enrollment (MS-MDE2 protocol)

        Note over Device, Fleet: Step 1: Discovery
        Device->>Fleet: POST /api/mdm/microsoft/discovery<br/>SOAP Discover request (with email)
        Fleet-->>Device: Discovery response<br/>(policy URL, enrollment URL,<br/>auth policy = OnPremise)

        Note over Device, Fleet: Step 2: Policy (MS-XCEP)
        Device->>Fleet: POST /api/mdm/microsoft/policy<br/>GetPolicies + JWT as BinarySecurityToken
        Fleet->>MSjwks: Fetch signing keys<br/>(login.microsoftonline.com/.../keys)
        MSjwks-->>Fleet: JWKS public keys
        Fleet->>Fleet: Validate JWT signature,<br/>tenant ID, audience, issuer
        Fleet-->>Device: Certificate policy<br/>(SHA256+RSA, 2-year validity)

        Note over Device, Fleet: Step 3: Enrollment (MS-WSTEP)
        Device->>Fleet: POST /api/mdm/microsoft/enroll<br/>RequestSecurityToken +<br/>self-signed CSR + DeviceID + JWT
        Fleet->>Fleet: Validate JWT (again)
        Fleet->>Fleet: Sign device CSR with<br/>Fleet identity certificate
        Fleet->>Fleet: Create enrollment record<br/>(mdm_windows_enrollments table)<br/>awaiting_configuration = Pending
        Fleet-->>Device: RequestSecurityTokenResponse<br/>(signed cert, provisioning doc,<br/>management endpoint URL)
    end

    rect rgb(255, 255, 240)
        Note over Device, Fleet: Phase 5 — Enrollment Status Page (ESP)
        Device->>Device: Enter ESP<br/>("Setting up your device...")
        loop SyncML polling during ESP
            Device->>Fleet: POST /api/mdm/microsoft/management<br/>OMA-DM SyncML (mTLS with signed cert)
            Fleet->>Fleet: Enqueue MDM commands:<br/>profiles, policies, BitLocker,<br/>setup experience software
            Fleet-->>Device: SyncML response with<br/>pending commands
            Device->>Fleet: Command results + status
        end
    end

    rect rgb(255, 240, 245)
        Note over Device, Fleetd: Phase 6 — Fleetd installation and host linkage
        Fleet->>Device: Deliver fleetd installer<br/>via setup experience package
        Device->>Device: Install fleetd
        Fleetd->>Fleet: Osquery enrollment<br/>(reports mdm_device_id)
        Fleet->>Fleet: Link host_uuid to<br/>mdm_windows_enrollments record
        Note over Fleet: Device now fully managed:<br/>MDM enrollment + osquery host linked
    end

    rect rgb(240, 255, 248)
        Note over Device, Fleet: Phase 7 — OOBE complete
        Device->>Fleet: not_in_oobe = true
        Fleet->>Fleet: awaiting_configuration = None<br/>Relax poll schedule (30s to 8h)<br/>(fleetd handles on-demand wake)
        Device->>Device: User reaches desktop
        Note over Device, Fleet: Ongoing: fleetd reports inventory,<br/>Fleet pushes config changes,<br/>no WNS needed (fleetd wakes device)
    end
```

## Phase summary

| Phase | What happens | Key actors |
|-------|-------------|------------|
| 1. Device registration | OEM registers hardware hash with Microsoft during manufacturing | OEM, Microsoft Autopilot |
| 2. IT admin config | One-time setup: Autopilot profile in Intune, Entra app registration pointing to Fleet, Fleet MDM settings | Intune, Entra, Fleet |
| 3. OOBE and Autopilot | Device powers on, identifies itself via hardware hash, gets Autopilot profile, user joins Entra | Device, Autopilot, Entra |
| 4. MDM enrollment | MS-MDE2 protocol: Discovery, Policy (MS-XCEP), Enrollment (MS-WSTEP) with JWT auth | Device, Fleet, Microsoft JWKS |
| 5. ESP | Device polls Fleet via SyncML for profiles, policies, and software during setup screen | Device, Fleet |
| 6. Fleetd install | Fleetd installed via setup experience, osquery enrolls, host linked to MDM enrollment | Fleetd, Fleet |
| 7. Steady state | OOBE complete, poll interval relaxed, fleetd handles on-demand sync | Device, Fleet |
