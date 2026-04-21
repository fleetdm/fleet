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

### Deprecated Versions

A version is considered deprecated if it appeared in older releases but is no longer listed in the most recent release. Deprecated versions get upgrade paths pointing to the oldest newer version that has a fix.

Versions that aren't in the most recent release but also weren't in any older releases (like LTSC versions that appear sporadically) are NOT marked deprecated - they only get direct fixes.

### Vulnerability Detection

A host is vulnerable if:

1. **Supported version**: Host's build suffix < fixed build suffix for that version branch
2. **Deprecated version**: The fix points to a different version branch (host must upgrade)

## Generating Bulletins

```bash
cd cmd/winoffice
go run generate.go
```

This creates a bulletin file in `winoffice_out/` with the naming format `fleet_winoffice_bulletin-YYYY_MM_DD.json`.
