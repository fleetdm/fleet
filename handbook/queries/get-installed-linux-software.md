# Get installed Linux software

Get all software installed on a Linux computer, including browser plugins and installed packages.

> This does not included other running processes in the `processes` table.

### Support
Linux

### Query
```sql
SELECT
  name AS name,
  version AS version,
  'Package (APT)' AS type,
  'apt_sources' AS source
FROM apt_sources
UNION
SELECT
  name AS name,
  version AS version,
  'Package (deb)' AS type,
  'deb_packages' AS source
FROM deb_packages
UNION
SELECT
  package AS name,
  version AS version,
  'Package (Portage)' AS type,
  'portage_packages' AS source
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  'Package (RPM)' AS type,
  'rpm_packages' AS source
FROM rpm_packages
UNION
SELECT
  name AS name,
  '' AS version,
  'Package (YUM)' AS type,
  'yum_sources' AS source
FROM yum_sources
UNION
SELECT
  name AS name,
  version AS version,
  'Package (NPM)' AS type,
  'npm_packages' AS source
FROM npm_packages
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
