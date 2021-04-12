# Get installed macOS software

Get all software installed on a macOS computer, including apps, browser plugins, and installed packages.

> This does not included other running processes in the `processes` table.

### Support
macOS

### Query
```sql
SELECT
  name AS name,
  bundle_short_version AS version,
  'Application (macOS)' AS type,
  'apps' AS source
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Python)' AS type,
  'python_packages' AS source
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Browswer plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browswer plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM firefox_addons
UNION
SELECT
  name As name,
  version AS version,
  'Browswer plugin (Safari)' AS type,
  'safari_extensions' AS source
FROM safari_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Homebrew)' AS type,
  'homebrew_packages' AS source
FROM homebrew_packages;
```

### Purpose

Informational

### Remediation

N/A
