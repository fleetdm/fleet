---
name: fleet-gitops
description: Use when working on Fleet GitOps configuration files, including osquery queries, configuration profiles, DDM declarations, software management, and CVE remediation in the it-and-security folder.
---

# Fleet GitOps – kilocode skill

## Queries & Reports

- Only use **Fleet tables and supported columns** when writing osquery queries or Fleet reports.
- Do not reference tables or columns that are not present in the Fleet schema for the target platform.
- Validate tables and column names against the Fleet schema before including them in a query:
  - https://github.com/fleetdm/fleet/tree/main/schema

## Configuration Profiles

When generating or modifying configuration profiles:

- **First-party Apple payloads** (`.mobileconfig`) — validate payload keys, types, and allowed values against the Apple Device Management reference:
  - https://github.com/apple/device-management/tree/release/mdm/profiles
- **Third-party Apple payloads** (`.mobileconfig`) — validate against the ProfileManifests community reference:
  - https://github.com/ProfileManifests/ProfileManifests
- **Windows CSPs** (`.xml`) — validate CSP paths, formats, and allowed values against Microsoft's MDM protocol reference:
  - https://learn.microsoft.com/en-us/windows/client-management/mdm/
- **Android profiles** (`.json`) — validate keys and values against the Android Management API `enterprises.policies` reference:
  - https://developers.google.com/android/management/reference/rest/v1/enterprises.policies

## Software

- When adding software for macOS or Windows hosts, **always check the Fleet-maintained app catalog first** before using a custom package:
  - https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps
- In GitOps YAML, use the `fleet_maintained_apps` key with the app's `slug` to reference a Fleet-maintained app.
- When remediating a CVE, use Fleet's built-in vulnerability detection to identify affected software, then follow the Software section above to deploy a fix — preferring a Fleet-maintained app update where available, otherwise a custom package.

## Declarative Device Management (DDM)

When generating or modifying DDM declarations:

- Validate declaration types, keys, and values against the Apple DDM reference:
  - https://github.com/apple/device-management/tree/release/declarative/declarations
- Ensure the `Type` identifier matches a supported declaration type from the reference.

---

## References

- Fleet GitOps documentation: https://fleetdm.com/docs/configuration/yaml-files
- Fleet API documentation: https://fleetdm.com/docs/rest-api/rest-api

---

*This file will grow as new patterns and constraints are established.*
