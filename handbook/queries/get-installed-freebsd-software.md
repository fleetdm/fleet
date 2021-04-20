# Get installed FreeBSD software

Get all software installed on a FreeBSD computer, including browser plugins and installed packages.

> This does not included other running processes in the `processes` table.

### Support
FreeBSD

### Query
```sql
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Browser plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM firefox_addons
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Atom)' AS type,
  'atom_packages' AS source
FROM atom_packages
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
  'Package (pkg)' AS type,
  'pkg_packages' AS source
FROM pkg_packages;
```

### Purpose

Informational

### Remediation

N/A
