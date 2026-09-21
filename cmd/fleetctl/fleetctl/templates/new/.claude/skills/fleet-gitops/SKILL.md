---
name: fleet-gitops
description: Help with this repository's Fleet GitOps configuration, including queries, profiles, software, and DDM declarations, validated against upstream references.
allowed-tools: Read, Grep, Glob, Edit, Write, WebFetch, WebSearch
effort: high
---

You are helping with this repository's Fleet GitOps configuration: $ARGUMENTS

This repo configures a live Fleet instance — CI applies `default.yml` and `fleets/*.yml` (and everything they reference under `platforms/`) on every push to the default branch. Apply the following constraints for all work in this session.

## Queries & reports

- Only use **Fleet tables and supported columns** when writing osquery queries for policies or reports.
- Do not reference tables or columns that aren't present in the Fleet schema for the target platform.
- Validate table and column names against the Fleet schema before including them in a query:
  - https://fleetdm.com/tables

## Configuration profiles

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

- When adding software for macOS, Windows, or Linux hosts, **always check the Fleet-maintained app catalog first** before using a custom package:
  - https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps
- In a fleet's YAML, use the `fleet_maintained_apps` key with the app's `slug` to reference a Fleet-maintained app.
- When remediating a CVE, identify the affected software from Fleet's vulnerability detection, then follow the guidance above — prefer a Fleet-maintained app update where available, otherwise a custom package.

## Declarative Device Management (DDM)

When generating or modifying DDM declarations:

- Validate declaration types, keys, and values against the Apple DDM reference:
  - https://github.com/apple/device-management/tree/release/declarative/declarations
- Ensure the `Type` identifier matches a supported declaration type from the reference.
- Give every declaration a unique `Identifier` — contour and similar tools derive it from the last component of the declaration type, so two declarations of different types can collide if named carelessly.

## Apple profile authoring with contour (optional)

If [contour](https://github.com/macadmins/contour) is installed in this repo (look for `.contour/config.toml`), prefer driving it over hand-writing Apple profiles and declarations — it validates against Apple's embedded schema and produces deterministic output. Run `contour help-ai` for its command index rather than guessing flags. Contour is a separate, independently-versioned project — if `.contour/config.toml` isn't present, it isn't set up for this repo; don't assume its commands are available.

## References

- Fleet GitOps documentation: https://fleetdm.com/docs/configuration/yaml-files
- Fleet REST API documentation: https://fleetdm.com/docs/rest-api/rest-api
