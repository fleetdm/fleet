# Windows Office Vulnerability Detection

This package detects vulnerabilities in Microsoft 365 Apps and Office products on Windows by scraping Microsoft's security updates page.

## Overview

Windows Office uses a version format: `16.0.<build_prefix>.<build_suffix>`

- **Build prefix** identifies the version branch (e.g., `19725` → version `2602`, meaning February 2026)
- **Build suffix** identifies the specific build within that branch

The package:

1. Scrapes [Microsoft's Office security updates page](https://learn.microsoft.com/en-us/officeupdates/microsoft365-apps-security-updates)
2. Builds a bulletin mapping CVEs to fixed versions
3. Compares host software versions against the bulletin to detect vulnerabilities

## Supported Products

- Microsoft 365 Apps for enterprise
- Office LTSC 2024/2021
- Office 2019

## Key Concepts

### Version Branches

Each version branch (e.g., `2602`) has a unique build prefix. The bulletin tracks which CVEs are fixed in which builds.

### Dropped Versions

When Microsoft drops support for a version branch, hosts on that version become vulnerable to all new CVEs. The bulletin includes a resolved version pointing to the oldest fixed version that is newer than the installed version.

### Vulnerability Detection

A host is vulnerable if:

1. **Same version branch**: Host's build suffix < fixed build suffix
2. **Different version branch**: Host must upgrade to a newer version (the fix's build prefix differs from host's)

## Generating Bulletins

```bash
cd cmd/winoffice
go run generate.go
```

This creates a bulletin file in `winoffice_out/` with the naming format `fleet_winoffice_bulletin-YYYY_MM_DD.json`.
