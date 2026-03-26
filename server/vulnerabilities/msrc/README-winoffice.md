# Windows Office Vulnerability Matching

This document describes the Windows Office security bulletin format and vulnerability matching logic.

## Overview

Windows Office vulnerability data is scraped from Microsoft's [Office security updates page](https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates) and stored in a JSON bulletin file for runtime vulnerability matching.

## Office Version Model

Microsoft Office uses a versioning scheme with two key components:

```
16.0.<build_prefix>.<build_suffix>
     │              │
     │              └── Increments with each security patch
     └── Identifies the version branch
```

### Microsoft 365 Apps

Microsoft 365 moves to new version branches regularly (monthly or semi-annually depending on channel):
- Version 2602 uses build prefix 19725
- Version 2512 uses build prefix 19530
- Version 2408 uses build prefix 17932

Each version branch has a stable build prefix that doesn't change.

### Office 2019 / Office LTSC

Office 2019 and LTSC editions stay on a fixed version branch but receive monthly patches with incrementing build prefixes:
- Office 2019: Version 1808, build prefixes 10364, 10366, 10367, ... 10417, etc.
- Office LTSC 2021: Version 2108, build prefixes 14326, 14332, 14334, etc.

This means these products have many build prefixes mapping to a single version branch.

## Bulletin JSON Structure

The bulletin uses a version-indexed structure optimized for the primary use case: given a host's Office version, find all CVEs that affect it.

```json
{
  "version": 1,
  "build_prefixes": {
    "19725": "2602",
    "19530": "2512",
    "10417": "1808"
  },
  "versions": {
    "2602": {
      "supported": true,
      "security_updates": [
        {"cve": "CVE-2026-12345", "fixed_build": "16.0.19725.20172"},
        {"cve": "CVE-2026-12346", "fixed_build": "16.0.19725.20172"}
      ]
    },
    "1808": {
      "supported": true,
      "security_updates": [
        {"cve": "CVE-2026-12345", "fixed_build": "16.0.10417.20000"},
        {"cve": "CVE-2024-12345", "fixed_build": "16.0.10400.20100"}
      ]
    }
  }
}
```

### Fields

| Field | Description |
|-------|-------------|
| `version` | Schema version (currently 1) |
| `build_prefixes` | Maps build prefix to version branch (e.g., "19725" → "2602") |
| `versions` | Security data indexed by version branch |
| `versions.<ver>.supported` | Whether this version is currently supported by Microsoft |
| `versions.<ver>.security_updates` | CVEs affecting this version with their fixed builds |

### Why Version-Indexed?

The primary query pattern is: "Given a host version, return all CVEs that affect it."

**Version-indexed (current):**
```
1. Parse version → build_prefix = "19725"
2. Look up version branch → "2602" (O(1))
3. Get versions["2602"].security_updates (O(1))
4. Iterate only CVEs for this version
```


**CVE distribution by version (typical):**

| Version | CVEs | Notes |
|---------|------|-------|
| 2602 | ~7 | Current M365 (few accumulated CVEs) |
| 2512 | ~25 | |
| 2408 | ~200 | Semi-Annual/LTSC 2024 |
| 2108 | ~360 | LTSC 2021 |
| 1808 | ~500 | Office 2019 (most accumulated CVEs) |

## Matching Algorithm

To find all CVEs affecting a host:

```
Input: host_version = "16.0.19725.20204"

1. Parse host version
   → build_prefix = "19725", build_suffix = "20204"

2. Look up version branch from build_prefixes
   → "19725" maps to version "2602"

3. Get security_updates for this version
   → versions["2602"].security_updates (direct access, O(1))

4. For each CVE in security_updates:
   → Parse fixed_build to get fixed_suffix
   → If build_suffix < fixed_suffix: HOST IS VULNERABLE
   → Add to results with fixed_build for remediation

Output: List of {cve, fixed_build} for vulnerable CVEs
```

### Edge Cases

| Scenario | Result |
|----------|--------|
| Unknown build prefix | Error (might be very old or very new) |
| Version not in versions map | No CVEs (version predates tracking) |
| Host build >= fixed build | Not vulnerable to that CVE |
| Host build < fixed build | Vulnerable, include in results |


## Generator

To regenerate the bulletin from live Microsoft data:

```bash
go run cmd/msrc/generate_winoffice.go
cat winoffice_out/fleet_winoffice_bulletin-*.json | jq .
```

## Future Extensibility

When adding Office LTSC 2024/2021 or Office 2019 as separate products with distinct matching rules, the schema can evolve to version 2 with product-level grouping:

```json
{
  "version": 2,
  "products": {
    "microsoft_365": {
      "display_name": "Microsoft 365 Apps",
      "build_prefixes": {...},
      "versions": {...}
    },
    "office_ltsc_2024": {
      "display_name": "Office LTSC 2024",
      "build_prefixes": {...},
      "versions": {...}
    }
  }
}
```
