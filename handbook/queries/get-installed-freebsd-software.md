# Get installed FreeBSD software

Get all software installed on a FreeBSD computer, including browser plugins and installed packages.

> This does not included other running processes in the `processes` table.

### Support
FreeBSD

### Query
```sql
SELECT
  name AS name,
  version AS version
  'chrome_extensions' AS source
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version
  'firefox_addons' AS source
FROM firefox_addons
UNION
SELECT
  name AS name,
  version AS version
  'atom_packages' AS source
FROM atom_packages
UNION
SELECT
  name AS name,
  version AS version
  'python_packages' AS source
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  'pkg_packages' AS source
FROM pkg_packages;
```

### Purpose

Informational

### Remediation

N/A
