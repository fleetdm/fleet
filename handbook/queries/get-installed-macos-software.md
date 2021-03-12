# Get installed macOS software

Get all software installed on a macOS computer, including apps, browser plugins, and installed packages.

> This does not included other running processes in the `processes` table.

### Support
MacOS

### Query
```sql
SELECT
  name AS name,
  bundle_short_version AS version,
  'apps' AS source
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'python_packages' AS source
FROM python_packages
UNION
SELECT
  name as name,
  version AS version
  'chrome_extensions' AS source
FROM chrome_extensions
SELECT
  name as name,
  version AS version
  'firefox_addones' AS source
FROM firefox_addons
SELECT
  name as name,
  version AS version
  'opera_extensions' AS source
FROM opera_extensions
SELECT
  name as name,
  version AS version
  'safari_extensions' AS source
FROM safari_extensions
SELECT
  name as name,
  version AS version
  'homebrew_packages' AS source
FROM homebrew_packages;
```

### Purpose

Informational

### Remediation

N/A
