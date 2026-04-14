# iOS MDM Policy Engine — Architecture Document

## Overview

This document describes the architecture for evaluating compliance policies on iOS/iPadOS devices using MDM channel APIs, creating a parallel to Fleet's osquery-based policy engine.

### Problem Statement

Fleet's policy engine relies on osquery SQL queries running locally on devices. iOS/iPadOS devices cannot run osquery (Apple platform restriction), creating a compliance gap for mobile device management. Customers need the same policy evaluation, automation, and reporting capabilities for iOS/iPadOS that they have for macOS/Windows/Linux.

### Related Issues

- [#36434](https://github.com/fleetdm/fleet/issues/36434) — Server-Side Policy Remediation for Mobile (No Agent)
- [#26337](https://github.com/fleetdm/fleet/issues/26337) — More host vitals for iOS/iPadOS hosts (9 customer labels)
- [#39281](https://github.com/fleetdm/fleet/issues/39281) — iOS/iPadOS: More host vitals
- [#39088](https://github.com/fleetdm/fleet/issues/39088) — iOS/iPadOS: Labels based on host vitals

## Architecture Decision

**UNIFIED-CACHE with thin interface, polling Phase 1.**

After a structured multi-agent architecture debate evaluating three approaches (DDM-first, polling-first, unified-cache), the consensus recommendation is:

1. Build a normalized **device state store** updated from any data channel
2. Ship **Phase 1 with MDM polling** (DeviceInformation, SecurityInfo, InstalledApplicationList commands)
3. Design the store interface so **DDM status reports slot in without migration** (Phase 2)
4. Write MDM policy results to the **existing `policy_membership` table** to get all existing automations (webhooks, scripts, software installs, VPP, conditional access, calendar) for free

### Key Design Principle

The policy evaluator never knows or cares whether data came from MDM polling or DDM status reports. It reads from the device state store and evaluates rules.

## Data Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                     DATA ACQUISITION LAYER                          │
├─────────────────────────┬───────────────────────────────────────────┤
│  Phase 1: MDM Polling   │  Phase 2: DDM Status Reports             │
│                         │                                           │
│  APNs Push → Device     │  StatusSubscription declaration →         │
│  Check-in → MDM Command │  Device sends delta reports on change    │
│  → Plist Response       │  → JSON status items                     │
│                         │                                           │
│  Commands:              │  Status Items:                            │
│  • DeviceInformation    │  • passcode.is-present                   │
│  • SecurityInfo         │  • passcode.is-compliant                 │
│  • InstalledAppList     │  • device.operating-system.version       │
│  • ProfileList          │  • app.managed.list                      │
│  • CertificateList      │  • softwareupdate.*                      │
│  • Restrictions         │  • device.power.battery-health           │
│  • AvailableOSUpdates   │  • 41 production items total             │
└────────────┬────────────┴──────────────────┬────────────────────────┘
             │                               │
             ▼                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    RESPONSE PARSERS                                  │
│                                                                      │
│  ParseDeviceInformationResponse() → map[string]string               │
│  ParseSecurityInfoResponse()      → map[string]string               │
│  ParseInstalledAppListResponse()  → map[string]string               │
│  (Phase 2: ParseDDMStatusReport() → map[string]string)              │
│                                                                      │
│  Output: flat key-value maps with dot-notation keys                 │
│  e.g. "DeviceInformation.OSVersion" → "17.4"                       │
│       "SecurityInfo.PasscodePresent" → "true"                       │
│       "InstalledApplicationList.com.slack.Slack.ShortVersion" → "4" │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    DEVICE STATE STORE                                │
│                                                                      │
│  Interface:                                                          │
│    UpdateDeviceState(hostUUID, map[field]DeviceStateEntry)           │
│    GetDeviceState(hostUUID) → map[field]DeviceStateEntry            │
│                                                                      │
│  DeviceStateEntry:                                                   │
│    Value      string     // the data point value                    │
│    Source     string     // "mdm_poll" or "ddm"                     │
│    ObservedAt time.Time  // when this was observed                  │
│                                                                      │
│  Phase 1: InMemoryDeviceStateStore (sync.RWMutex + map)             │
│  Phase 2: MySQLDeviceStateStore (persistent, queryable)             │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    POLICY EVALUATOR                                  │
│                                                                      │
│  EvaluateMDMPolicy(policyID, hostID, definition, deviceData)        │
│    → MDMPolicyResult{Passes: true/false}                            │
│                                                                      │
│  MDMPolicyDefinition:                                               │
│    Checks []MDMPolicyCheck   (AND logic — all must pass)            │
│                                                                      │
│  MDMPolicyCheck:                                                     │
│    Field    string           // "DeviceInformation.OSVersion"       │
│    Operator string           // eq, neq, gt, version_gte, etc.     │
│    Expected string           // "17.0"                              │
│                                                                      │
│  12 operators: eq, neq, gt, lt, gte, lte, contains, not_contains,  │
│                version_gte, version_lte, exists, not_exists         │
│                                                                      │
│  Pure function — no side effects, no state, fully testable          │
└────────────────────────────┬────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    RESULT STORAGE                                    │
│                                                                      │
│  Write to existing policy_membership table:                         │
│    RecordPolicyQueryExecutions(ctx, host, results, timestamp)       │
│                                                                      │
│  Same table, same pass/fail semantics, same flip detection          │
│  → ALL existing automations work with zero changes:                 │
│    • Webhook notifications                                          │
│    • Script execution                                               │
│    • Software installer triggers                                    │
│    • VPP app installation                                           │
│    • Conditional access (Okta/Entra)                                │
│    • Calendar event creation                                        │
│    • Policy stats aggregation                                       │
│    • Fleet UI policy dashboard                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Policy Check Mapping — MDM Commands to Compliance Questions

### Device Identity & Inventory

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| What OS version is running? | DeviceInformation | OSVersion | device.operating-system.version |
| Is device on minimum OS? | DeviceInformation | OSVersion >= threshold | device.operating-system.version |
| What device model? | DeviceInformation | ModelName, Model | device.model.identifier |
| Is device supervised? | DeviceInformation | IsSupervised | N/A |
| What is serial number? | DeviceInformation | SerialNumber | device.identifier.serial-number |

### Security Posture

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Is passcode set? | SecurityInfo | PasscodePresent | passcode.is-present |
| Is passcode compliant? | SecurityInfo | PasscodeCompliant | passcode.is-compliant |
| Is passcode profile-compliant? | SecurityInfo | PasscodeCompliantWithProfiles | N/A |
| Hardware encryption available? | SecurityInfo | HardwareEncryptionCaps > 0 | N/A |
| Enrolled via DEP? | SecurityInfo | ManagementStatus.EnrolledViaDEP | N/A |
| Is user enrollment (BYOD)? | SecurityInfo | ManagementStatus.IsUserEnrollment | N/A |
| Auto-lock timeout acceptable? | SecurityInfo | AutoLockTime <= threshold | N/A |

### Application Compliance

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Required app installed? | InstalledApplicationList | Identifier in list | app.managed.list |
| Prohibited app installed? | InstalledApplicationList | Identifier NOT in list | app.managed.list |
| App version minimum? | InstalledApplicationList | ShortVersion >= threshold | app.managed.list |
| Beta/TestFlight apps? | InstalledApplicationList | BetaApp == true | N/A |
| Managed app installed? | ManagedApplicationList | Status == "Managed" | app.managed.list |

### Configuration Profiles

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Required profile installed? | ProfileList | PayloadIdentifier in list | management.declarations |
| Profile MDM-managed? | ProfileList | IsManaged == true | N/A |
| Profile removal-protected? | ProfileList | HasRemovalPasscode == true | N/A |

### Certificates

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Required cert installed? | CertificateList | CommonName match | security.certificate.list |
| Identity cert present? | CertificateList | IsIdentity == true | N/A |
| Provisioning profile valid? | ProvisioningProfileList | ExpiryDate > now | N/A |

### Device Health

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Storage critically low? | DeviceInformation | AvailableDeviceCapacity < threshold | N/A |
| Battery level acceptable? | DeviceInformation | BatteryLevel > threshold | N/A |
| Battery health OK? | N/A | N/A | device.power.battery-health |
| Find My enabled? | DeviceInformation | IsDeviceLocatorServiceEnabled | N/A |
| Activation Lock enabled? | DeviceInformation | IsActivationLockEnabled | N/A |
| iCloud Backup enabled? | DeviceInformation | IsCloudBackupEnabled | N/A |
| Last backup recent? | DeviceInformation | LastCloudBackupDate > threshold | N/A |

### OS Updates

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Updates available? | AvailableOSUpdates | count > 0 | softwareupdate.pending-version |
| Critical update pending? | AvailableOSUpdates | IsCritical == true | softwareupdate.pending-version |
| Update failed? | N/A | N/A | softwareupdate.failure-reason |
| Update install state? | N/A | N/A | softwareupdate.install-state |

### Restrictions

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Camera disabled? | Restrictions | allowCamera == false | N/A |
| AirDrop disabled? | Restrictions | allowAirDrop == false | N/A |
| Encrypted backups enforced? | Restrictions | forceEncryptedBackup == true | N/A |
| App Store blocked? | Restrictions | allowAppInstallation == false | N/A |
| Managed Open In enforced? | Restrictions | allowOpenFromManagedToUnmanaged == false | N/A |

### Network

| Compliance Question | MDM Command | Field | DDM Alternative |
|---|---|---|---|
| Device roaming? | DeviceInformation | IsRoaming | N/A |
| Hotspot active? | DeviceInformation | PersonalHotspotEnabled | N/A |
| Data roaming enabled? | DeviceInformation | DataRoamingEnabled | N/A |

## Integration with Existing Policy Automation

The key architectural insight: the existing policy automation pipeline reads from `policy_membership` and doesn't care about the data source. By writing MDM policy results to the same table:

1. **Flip detection** works identically — `FlippingPoliciesForHost()` detects pass→fail and fail→pass transitions
2. **Webhooks** fire on flip via `TriggerFailingPoliciesAutomation()`
3. **Script execution** triggers via `processScriptsForNewlyFailingPolicies()`
4. **Software installers** trigger via `processSoftwareForNewlyFailingPolicies()`
5. **VPP app installation** triggers via `processVPPForNewlyFailingPolicies()`
6. **Conditional access** (Okta/Entra) triggers via `processConditionalAccessForNewlyFailingPolicies()`
7. **Calendar events** trigger via `processCalendarPolicies()`
8. **Policy stats** aggregate via `policy_stats` table
9. **Fleet UI** displays results in the policy dashboard

Zero changes to the automation pipeline required.

## BYOD / User-Enrollment Limitations

User-enrolled (BYOD) devices have restricted MDM command access:

1. **InstalledApplicationList** — Returns only managed apps (not all apps)
2. **CertificateList** — Returns only managed certificates
3. **ProfileList** — Returns only managed profiles
4. **DeviceInformation** — Most fields work but some may be restricted

### Strategy

- Mark each MDMPolicyCheck with whether it requires device enrollment vs user enrollment
- For user-enrolled devices, skip checks that require device enrollment and mark as "not applicable" rather than "fail"
- DDM status items generally work on both enrollment types (Phase 2 advantage)

## iOS Version Fragmentation

| Feature | Minimum iOS | Coverage |
|---|---|---|
| Basic MDM commands | iOS 4+ | Universal |
| DDM support | iOS 15+ | ~95% of active fleet |
| DDM security items (passcode) | iOS 16+ | ~90% |
| DDM software update items | iOS 17+ | ~80% |
| DDM battery health | iOS 17+ | ~80% |

### Strategy

- Phase 1 (MDM polling) works on all iOS versions
- Phase 2 (DDM) gracefully degrades: subscribe to available status items based on device OS version
- Policy evaluator handles missing fields from older devices: configurable behavior (fail-closed vs skip)

## DDM Integration Plan (Phase 2)

### Current State

Fleet's `handleDeclarationStatus()` in `server/service/apple_mdm.go:6568` only processes `StatusItems.Management.Declarations.Configurations`. All other status items (device, passcode, software update, apps) are silently dropped.

### Required Changes

1. **Extend `MDMAppleDDMStatusItems`** (`server/fleet/apple_mdm.go:893`) to capture device state categories:
   ```go
   type MDMAppleDDMStatusItems struct {
       Management MDMAppleDDMStatusManagement `json:"management"`
       Device     map[string]interface{}       `json:"device,omitempty"`
       Passcode   map[string]interface{}       `json:"passcode,omitempty"`
       App        map[string]interface{}       `json:"app,omitempty"`
       SoftwareUpdate map[string]interface{}   `json:"softwareupdate,omitempty"`
   }
   ```

2. **Extend `handleDeclarationStatus()`** to also process device state data:
   - Parse device/passcode/app/softwareupdate status items
   - Convert to DeviceStateEntry with source="ddm"
   - Call `UpdateDeviceState()` on the device state store

3. **Add DDM status subscription**: Send a `com.apple.configuration.management.status-subscriptions` declaration subscribing to policy-relevant status items (passcode.is-present, passcode.is-compliant, device.operating-system.version, etc.)

## Phase Roadmap

### Phase 1 — POC (this branch)
- Types, evaluator, parsers, in-memory store
- Architecture document
- Test suite (35+ test cases)

### Phase 2 — MVP
- MySQL-backed device state store
- MDM polling cron job
- API endpoints for MDM policy CRUD
- Integration with `RecordPolicyQueryExecutions()`
- SecurityInfo command in commander
- `policies.type` column migration

### Phase 3 — DDM Integration
- Extend DDM status report handler
- DDM status subscription management
- Real-time compliance updates
- Staleness detection and trust tiers

### Phase 4 — Full Feature
- Fleet UI for MDM policies
- GitOps support for MDM policy definitions
- Extended MDM command support (ProfileList, CertificateList, Restrictions)
- BYOD-aware policy evaluation
