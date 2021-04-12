# Get installed Windows software

Get all software installed on a Windows computer, including programs, browser plugins, and installed packages.

> This does not included other running processes in the `processes` table.

### Support
Windows

### Query
```sql
SELECT
  name AS name,
  version AS version,
  'Program (Windows)' AS type,
  'programs' AS source
FROM programs
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
  'Broswer plugin (IE)' AS type,
  'ie_extensions' AS source
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Broswer plugin (Chrome)' AS type,
  'chrome_extensions' AS source
FROM chrome_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'Broswer plugin (Firefox)' AS type,
  'firefox_addons' AS source
FROM firefox_addons
UNION
SELECT
  name AS name,
  version AS version,
  'Package (Chocolatey)' AS type,
  'chocolatey_packages' AS source
FROM chocolatey_packages
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
FROM python_packages;
```

### Purpose

Informational

### Remediation

N/A
